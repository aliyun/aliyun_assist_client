package taskengine

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	heavylock "github.com/viney-shih/go-lock"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
)

const (
	ErrUpdatingProcedureRunning = -7
)

// PeriodicTaskSchedule consists of timer and reusable invocation data structure
// for periodic task
type PeriodicTaskSchedule struct {
	timer              *timermanager.Timer
	reusableInvocation *Task
}

var (
	// FetchingTaskLock indicates whether one goroutine is fetching tasks
	FetchingTaskLock heavylock.CASMutex

	// Indicating whether is enabled to fetch tasks, ONLY operated by atomic operation
	_neverDirectWrite_Atomic_FetchingTaskEnabled int32 = 0

	_periodicTaskSchedules     map[string]*PeriodicTaskSchedule
	_periodicTaskSchedulesLock sync.Mutex
)

func init() {
	FetchingTaskLock = heavylock.NewCASMutex()

	_periodicTaskSchedules = make(map[string]*PeriodicTaskSchedule)
}

// EnableFetchingTask sets prviate indicator to allow fetching tasks
func EnableFetchingTask() {
	atomic.StoreInt32(&_neverDirectWrite_Atomic_FetchingTaskEnabled, 1)
}

func isEnabledFetchingTask() bool {
	state := atomic.LoadInt32(&_neverDirectWrite_Atomic_FetchingTaskEnabled)
	return state != 0
}

func Fetch(from_kick bool) int {
	// Fetching task should be allowed before all core components of agent have
	// been correctly initialized. This critical indicator would be set at the
	// end of program.run method
	if !isEnabledFetchingTask() {
		log.GetLogger().WithFields(logrus.Fields{
			"from_kick": from_kick,
		}).Infoln("Fetching tasks is disabled due to network is not ready")
		return 0
	}

	// NOTE: sync.Mutex from Go standard library does not support try-lock
	// operation like std::mutex in C++ STL, which makes it slightly hard for
	// goroutines of fetching tasks and checking updates to coopearate gracefully.
	// Futhermore, it does not support try-lock operation with specified timeout,
	// which makes it hard for goroutines of fetching tasks to wait in queue but
	// just throw many message about lock accquisition failure confusing others.
	// THUS heavy weight lock from github.com/viney-shih/go-lock library is used
	// to provide graceful locking mechanism for goroutine coopeartion. The cost
	// would be, some performance lost.
	if !FetchingTaskLock.TryLockWithTimeout(time.Duration(2) * time.Second) {
		log.GetLogger().WithFields(logrus.Fields{
			"from_kick": from_kick,
		}).Infoln("Fetching tasks is canceled due to another running fetching or updating process.")
		return ErrUpdatingProcedureRunning
	}
	defer FetchingTaskLock.Unlock()

	var task_size int

	if from_kick {
		task_size = fetchTasks("kickoff")
	} else {
		task_size = fetchTasks("startup")
	}

	for i := 0; i < 1 && from_kick && task_size == 0; i++ {
		time.Sleep(time.Duration(3) * time.Second)
		task_size = fetchTasks("kickoff")
	}

	return task_size
}

func fetchTasks(reason string) int {
	runInfo, stopInfo, sendFileInfo := FetchTaskList(reason)
	SendFiles(sendFileInfo)

	// NOTE:
	// * Non-periodic tasks are managed by TaskFactory
	// * Periodic tasks are managed by _periodicTaskSchedules
	taskFactory := GetTaskFactory()
	for _, v := range runInfo {
		fetchLogger := log.GetLogger().WithFields(logrus.Fields{
			"TaskId": v.TaskId,
			"Phase":  "Fetched",
		})
		fetchLogger.Info("Fetched to be run")

		// Tasks should not be duplicately handled
		if taskFactory.ContainsTaskByName(v.TaskId) {
			fetchLogger.Warning("Ignored duplicately fetched task")
			continue
		}

		if v.Cronat == "" {
			// Reuse specified logger across task scheduling phase
			scheduleLogger := log.GetLogger().WithFields(logrus.Fields{
				"TaskId": v.TaskId,
				"Phase":  "Scheduling",
			})
			t := NewTask(v)

			scheduleLogger.Info("Schedule non-periodic task")
			taskFactory.AddTask(t)
			pool := GetPool()
			pool.RunTask(t)
			scheduleLogger.Info("Scheduled for pending or running")
		} else {
			err := schedulePeriodicTask(v)
			if err != nil {
				log.GetLogger().WithFields(logrus.Fields{
					"taskInfo": v,
				}).WithError(err).Errorln("Failed to schedule periodic task")
			} else {
				log.GetLogger().WithFields(logrus.Fields{
					"taskInfo": v,
				}).Infoln("Succeed to schedule periodic task")
			}
		}
	}

	for _, v := range stopInfo {
		log.GetLogger().WithFields(logrus.Fields{
			"TaskId": v.TaskId,
			"Phase":  "Fetched",
		}).Info("Fetched to be canceled")

		if v.Cronat == "" {
			cancelLogger := log.GetLogger().WithFields(logrus.Fields{
				"TaskId": v.TaskId,
				"Phase":  "Cancelling",
			})
			// NOTE: Non-periodic tasks are managed by TaskFactory. Those tasks
			// does not exist in TaskFactory need not to be canceled.
			if !taskFactory.ContainsTaskByName(v.TaskId) {
				cancelLogger.Warning("Ignore task not found due to finished or error")
				continue
			}

			cancelLogger.Info("Cancel task and invocation")
			v, _ := taskFactory.GetTask(v.TaskId)
			v.Cancel()
			cancelLogger.Info("Canceled task and invocation")
		} else {
			err := cancelPeriodicTask(v)
			if err != nil {
				log.GetLogger().WithFields(logrus.Fields{
					"taskInfo": v,
				}).WithError(err).Errorf("Failed to cancel periodic task %s", v.TaskId)
			} else {
				log.GetLogger().WithFields(logrus.Fields{
					"taskInfo": v,
				}).Infof("Succeed to cancel periodic task %s", v.TaskId)
			}
		}
	}

	return len(runInfo) + len(stopInfo)
}

func (s *PeriodicTaskSchedule) startExclusiveInvocation() {
	// Reuse specified logger across task scheduling phase
	invocateLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId": s.reusableInvocation.taskInfo.TaskId,
		"Phase":  "PeriodicInvocating",
	})

	// NOTE: TaskPool has been closely wired with TaskFactory, thus:
	taskFactory := GetTaskFactory()
	// (3) Existed invocation in TaskFactory means task is running.
	if taskFactory.ContainsTaskByName(s.reusableInvocation.taskInfo.TaskId) {
		invocateLogger.Warn("Skip invocation since overlapped with existing invocation")
		return
	}

	invocateLogger.Info("Schedule new invocation of periodic task")
	// (2) Every time of invocation need to add itself into TaskFactory at first.
	taskFactory.AddTask(s.reusableInvocation)
	pool := GetPool()
	// (1) TaskPool.RunTask would remove task from TaskFactory.tasks map when finished.
	pool.RunTask(s.reusableInvocation)
	invocateLogger.Info("Scheduled new pending or running invocation")
}

func schedulePeriodicTask(taskInfo RunTaskInfo) error {
	timerManager := timermanager.GetTimerManager()
	if timerManager == nil {
		return errors.New("Global TimerManager instance is not initialized")
	}

	_periodicTaskSchedulesLock.Lock()
	defer _periodicTaskSchedulesLock.Unlock()

	// Reuse specified logger across task scheduling phase
	scheduleLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId": taskInfo.TaskId,
		"Phase":  "Scheduling",
	})

	// 1. Check whether periodic task has been registered in local task storage,
	// and had corresponding timer in timer manager
	_, ok := _periodicTaskSchedules[taskInfo.TaskId]
	if ok {
		scheduleLogger.Warn("Ignore periodic task registered in local")
		return nil
	}

	// 2. Create PeriodicTaskSchedule object
	scheduleLogger.Info("Create timer of periodic task")
	periodicTaskSchedule := &PeriodicTaskSchedule{
		timer: nil,
		// Invocations of periodic task is not allowed to overlap, so Task struct
		// for invocation data can be reused.
		reusableInvocation: NewTask(taskInfo),
	}
	// 3. Create cron expression timer and register into TimerManager
	// NOTE: reusableInvocation is binded to callback via closure feature of golang,
	// maybe explicit passing into callback like "data" for traditional thread
	// would be better
	timer, err := timerManager.CreateCronTimer(func() {
		periodicTaskSchedule.startExclusiveInvocation()
	}, taskInfo.Cronat)
	if err != nil {
		return err
	}
	// then bind it to periodicTaskSchedule object
	periodicTaskSchedule.timer = timer
	scheduleLogger.Info("Created timer of periodic task")

	// 4. Register schedule object into _periodicTaskSchedules
	_periodicTaskSchedules[taskInfo.TaskId] = periodicTaskSchedule
	scheduleLogger.Info("Registered periodic task")

	// 5. Current API of TimerManager requires manual startup of timer
	scheduleLogger.Info("Run timer of periodic task")
	_, err = timer.Run()
	if err != nil {
		timerManager.DeleteTimer(periodicTaskSchedule.timer)
		delete(_periodicTaskSchedules, taskInfo.TaskId)
		return err
	}
	scheduleLogger.Info("Running timer of periodic task")

	return nil
}

func cancelPeriodicTask(taskInfo RunTaskInfo) error {
	timerManager := timermanager.GetTimerManager()
	if timerManager == nil {
		return errors.New("Global TimerManager instance is not initialized")
	}

	_periodicTaskSchedulesLock.Lock()
	defer _periodicTaskSchedulesLock.Unlock()

	cancelLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId": taskInfo.TaskId,
		"Phase":  "Cancelling",
	})

	// 1. Check whether task is registered in local storage
	periodicTaskSchedule, ok := _periodicTaskSchedules[taskInfo.TaskId]
	if !ok {
		return fmt.Errorf("Unregistered periodic task %s", taskInfo.TaskId)
	}

	// 2. Delete timer of periodic task from TimerManager, which contains stopping
	// timer operation
	timerManager.DeleteTimer(periodicTaskSchedule.timer)
	cancelLogger.Infof("Stop and remove timer of periodic task")

	// 3. Delete registered task record from local storage
	delete(_periodicTaskSchedules, taskInfo.TaskId)
	cancelLogger.Infof("Deregistered periodic task")

	// 4. Cancel existing invocation of periodic task and send ACK
	runningInvocation, ok := GetTaskFactory().GetTask(taskInfo.TaskId)
	if ok {
		cancelLogger.Infof("Cancel running invocation of periodic task")
		runningInvocation.Cancel()
		cancelLogger.Infof("Canceled running invocation of periodic task")
	} else {
		cancelLogger.Infof("Not need to cancel running invocation of periodic task")
		// Since no running
		lastInvocation := periodicTaskSchedule.reusableInvocation
		lastInvocation.sendOutput("canceled", lastInvocation.getReportString(lastInvocation.output))
		cancelLogger.Infof("Sent canceled ACK with output of last invocation")
	}
	return nil
}
