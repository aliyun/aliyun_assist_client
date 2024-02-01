package taskengine

import (
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	heavylock "github.com/viney-shih/go-lock"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/models"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
	"github.com/aliyun/aliyun_assist_client/agent/util/atomicutil"
	"github.com/aliyun/aliyun_assist_client/agent/flagging"
)

const (
	ErrUpdatingProcedureRunning = -7
)

const (
	NormalTaskType  = 0
	SessionTaskType = 1
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
	// FetchingTaskCounter indicates how many goroutines are fetching tasks
	FetchingTaskCounter atomicutil.AtomicInt32

	// Indicating whether is enabled to fetch tasks, ONLY operated by atomic operation
	_neverDirectWrite_Atomic_FetchingTaskEnabled int32 = 0

	_periodicTaskSchedules     map[string]*PeriodicTaskSchedule
	_periodicTaskSchedulesLock sync.Mutex

	// Indicating whether startup fetch(reason=startup) has been done
	_startupFetched atomic.Bool
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

func IsStartupFetched() bool {
	return _startupFetched.Load()
}

func Fetch(from_kick bool, taskId string, taskType int) int {
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
	// Immediately release fetchingTaskLock to let other goroutine fetching
	// tasks go, but keep updating safe
	FetchingTaskLock.Unlock()

	// Increase fetchingTaskCounter to indicate there is a goroutine fetching
	// tasks, which the updating goroutine MUST notice and decrease it to let
	// updating goroutine go.
	FetchingTaskCounter.Add(1)
	defer FetchingTaskCounter.Add(-1)

	var task_size int
	var isColdstart bool
	fetchReason := FetchOnKickoff
	if taskType == NormalTaskType && taskId == "" && !_startupFetched.Swap(true) {
		fetchReason = FetchOnStartup
		// `isColdstart` only make sense for FetchOnStartup
		isColdstart, _ = flagging.IsColdstart()
		if from_kick {
			log.GetLogger().WithFields(logrus.Fields{
				"from_kick": from_kick,
			}).Infoln("Merge the fetch operations for the kick_off task and the startup task.")
		}
	}
	task_size = fetchTasks(fetchReason, taskId, taskType, isColdstart)

	for i := 0; i < 1 && from_kick && task_size == 0; i++ {
		time.Sleep(time.Duration(3) * time.Second)
		task_size = fetchTasks(FetchOnKickoff, taskId, taskType, false)
	}

	return task_size
}

func fetchTasks(reason FetchReason, taskId string, taskType int, isColdstart bool) int {
	taskInfos := FetchTaskList(reason, taskId, taskType, isColdstart)
	SendFiles(taskInfos.sendFiles)
	DoSessionTask(taskInfos.sessionInfos)
	for _, v := range taskInfos.runInfos {
		dispatchRunTask(v)
	}

	for _, v := range taskInfos.stopInfos {
		dispatchStopTask(v)
	}

	for _, v := range taskInfos.testInfos {
		dispatchTestTask(v)
	}

	return len(taskInfos.runInfos) + len(taskInfos.stopInfos) + len(taskInfos.sessionInfos) + len(taskInfos.sendFiles)
}

func dispatchRunTask(taskInfo models.RunTaskInfo) {
	fetchLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId":        taskInfo.TaskId,
		"InvokeVersion": taskInfo.InvokeVersion,
		"Phase":         "Fetched",
	})
	fetchLogger.Info("Fetched to be run")

	taskFactory := GetTaskFactory()
	var existedTask *Task
	if existedTask, _ = taskFactory.GetTask(taskInfo.TaskId); existedTask == nil {
		existedTask = getPeriodicTask(taskInfo.TaskId)
	}
	if existedTask != nil {
		if existedTask.taskInfo.InvokeVersion != taskInfo.InvokeVersion {
			// Task existed but with different InvokeVersion needs rehandle
			fetchLogger.Infof("Existed task with InvokeVersion[%d] needs rehandle",
				existedTask.taskInfo.InvokeVersion)
			switch taskInfo.Repeat {
			case models.RunTaskCron, models.RunTaskRate, models.RunTaskAt:
				fetchLogger.Infof("Cancel periodic task with invocaVersion[%d] quietly", existedTask.taskInfo.InvokeVersion)
				cancelPeriodicTask(existedTask.taskInfo, true)
			default:
				fetchLogger.Warning("Existed task is not Period. New task is duplicately fetched, ignore it")
				return
			}
		} else {
			// Tasks should not be duplicately handled
			fetchLogger.Warning("Ignored duplicately fetched task")
			return
		}
	}

	// Reuse specified logger across task scheduling phase
	scheduleLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId":        taskInfo.TaskId,
		"InvokeVersion": taskInfo.InvokeVersion,
		"Phase":         "Scheduling",
	})
	switch taskInfo.Repeat {
	case models.RunTaskOnce, models.RunTaskNextRebootOnly, models.RunTaskEveryReboot:
		t := NewTask(taskInfo, nil, nil)

		scheduleLogger.Info("Schedule non-periodic task")
		// Non-periodic tasks are managed by TaskFactory
		if err := taskFactory.AddTask(t); err != nil {
			scheduleLogger.Error("Add task failed: ", err.Error())
			return
		}
		pool := GetPool()
		pool.RunTask(func() {
			code, err := t.Run()
			if code != 0 || err != nil {
				metrics.GetTaskFailedEvent(
					"taskid", t.taskInfo.TaskId,
					"InvokeVersion", strconv.Itoa(t.taskInfo.InvokeVersion),
					"errormsg", err.Error(),
					"reason", strconv.Itoa(int(code)),
				).ReportEvent()
			}
			taskFactory := GetTaskFactory()
			taskFactory.RemoveTaskByName(t.taskInfo.TaskId)
		})
		scheduleLogger.Info("Scheduled for pending or running")
	case models.RunTaskCron, models.RunTaskRate, models.RunTaskAt:
		// Periodic tasks are managed by _periodicTaskSchedules
		err := schedulePeriodicTask(taskInfo)
		if err != nil {
			scheduleLogger.WithFields(logrus.Fields{
				"taskInfo": taskInfo,
			}).WithError(err).Errorln("Failed to schedule periodic task")
		} else {
			scheduleLogger.Infoln("Succeed to schedule periodic task")
		}
	default:
		scheduleLogger.WithFields(logrus.Fields{
			"taskInfo": taskInfo,
		}).Errorln("Unknown repeat type")
	}
}

func dispatchStopTask(taskInfo models.RunTaskInfo) {
	log.GetLogger().WithFields(logrus.Fields{
		"TaskId":        taskInfo.TaskId,
		"InvokeVersion": taskInfo.InvokeVersion,
		"Phase":         "Fetched",
	}).Info("Fetched to be canceled")

	cancelLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId":        taskInfo.TaskId,
		"InvokeVersion": taskInfo.InvokeVersion,
		"Phase":         "Cancelling",
	})
	taskFactory := GetTaskFactory()
	switch taskInfo.Repeat {
	case models.RunTaskOnce, models.RunTaskNextRebootOnly, models.RunTaskEveryReboot:
		scheduledTask, ok := taskFactory.GetTask(taskInfo.TaskId)
		if ok {
			cancelLogger.Info("Cancel task and invocation")
			scheduledTask.Cancel(false)
			cancelLogger.Info("Canceled task and invocation")
		} else {
			response, err := sendStoppedOutput(taskInfo.TaskId, taskInfo.InvokeVersion, 0, 0, 0, 0, "", stopReasonKilled)
			cancelLogger.WithFields(logrus.Fields{
				"response": response,
			}).WithError(err).Warning("Force cancelling task not found due to finished or error")
		}
	case models.RunTaskCron, models.RunTaskRate, models.RunTaskAt:
		// Periodic tasks are managed by _periodicTaskSchedules
		err := cancelPeriodicTask(taskInfo, false)
		if err != nil {
			cancelLogger.WithFields(logrus.Fields{
				"taskInfo": taskInfo,
			}).WithError(err).Errorln("Failed to cancel periodic task")
		} else {
			cancelLogger.Infoln("Succeed to cancel periodic task")
		}
	default:
		cancelLogger.WithFields(logrus.Fields{
			"taskInfo": taskInfo,
		}).Errorln("Unknown repeat type")
	}
}

func dispatchTestTask(taskInfo models.RunTaskInfo) {
	fetchLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId":        taskInfo.TaskId,
		"InvokeVersion": taskInfo.InvokeVersion,
		"Phase":         "Fetched",
	})
	fetchLogger.Info("Fetched to be run")

	taskFactory := GetTaskFactory()
	// Tasks should not be duplicately handled
	if taskFactory.ContainsTaskByName(taskInfo.TaskId) {
		fetchLogger.Warning("Ignored duplicately fetched task")
		return
	}

	// Reuse specified logger across task scheduling phase
	scheduleLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId":        taskInfo.TaskId,
		"InvokeVersion": taskInfo.InvokeVersion,
		"Phase":         "Scheduling",
	})
	switch taskInfo.Repeat {
	case models.RunTaskOnce, models.RunTaskCron, models.RunTaskNextRebootOnly, models.RunTaskEveryReboot, models.RunTaskRate, models.RunTaskAt:
		t := NewTask(taskInfo, nil, nil)

		scheduleLogger.Info("Schedule testing task to be pre-checked")
		pool := GetPrecheckPool()
		pool.RunTask(func() {
			t.PreCheck(true)
		})
		scheduleLogger.Info("Scheduled testing task to be pre-checked")
	default:
		scheduleLogger.WithFields(logrus.Fields{
			"taskInfo": taskInfo,
		}).Errorln("Unknown repeat type")
	}
}

func (s *PeriodicTaskSchedule) startExclusiveInvocation() {
	// Reuse specified logger across task scheduling phase
	invocateLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId":        s.reusableInvocation.taskInfo.TaskId,
		"InvokeVersion": s.reusableInvocation.taskInfo.InvokeVersion,
		"Phase":         "PeriodicInvocating",
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
	pool.RunTask(func() {
		code, err := s.reusableInvocation.Run()
		if code != 0 || err != nil {
			metrics.GetTaskFailedEvent(
				"taskid", s.reusableInvocation.taskInfo.TaskId,
				"InvokeVersion", strconv.Itoa(s.reusableInvocation.taskInfo.InvokeVersion),
				"errormsg", err.Error(),
				"reason", strconv.Itoa(int(code)),
			).ReportEvent()
		}
		taskFactory := GetTaskFactory()
		taskFactory.RemoveTaskByName(s.reusableInvocation.taskInfo.TaskId)
	})
	invocateLogger.Info("Scheduled new pending or running invocation")
}

func schedulePeriodicTask(taskInfo models.RunTaskInfo) error {
	timerManager := timermanager.GetTimerManager()
	if timerManager == nil {
		return errors.New("Global TimerManager instance is not initialized")
	}

	_periodicTaskSchedulesLock.Lock()
	defer _periodicTaskSchedulesLock.Unlock()

	// Reuse specified logger across task scheduling phase
	scheduleLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId":        taskInfo.TaskId,
		"InvokeVersion": taskInfo.InvokeVersion,
		"Phase":         "Scheduling",
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
		reusableInvocation: nil,
	}
	// 3. Create timer based on expression and register into TimerManager
	// NOTE: reusableInvocation is binded to callback via closure feature of golang,
	// maybe explicit passing into callback like "data" for traditional thread
	// would be better
	var timer *timermanager.Timer
	var err error
	var scheduleLocation *time.Location = nil
	var onFinish FinishCallback = nil
	if taskInfo.Repeat == models.RunTaskRate {
		creationTimeSeconds := taskInfo.CreationTime / 1000
		creationTimeMs := taskInfo.CreationTime % 1000
		creationTime := time.Unix(creationTimeSeconds, creationTimeMs*int64(time.Millisecond))
		timer, err = timerManager.CreateRateTimer(func() {
			periodicTaskSchedule.startExclusiveInvocation()
		}, taskInfo.Cronat, creationTime)
	} else if taskInfo.Repeat == models.RunTaskAt {
		timer, err = timerManager.CreateAtTimer(func() {
			periodicTaskSchedule.startExclusiveInvocation()
		}, taskInfo.Cronat)
	} else {
		timer, err = timerManager.CreateCronTimer(func() {
			periodicTaskSchedule.startExclusiveInvocation()
		}, taskInfo.Cronat)
	}
	if err != nil {
		// Report errors for invalid cron/rate/at expression
		var response string
		var reportErr error
		if cronParameterErr, ok := err.(timermanager.CronParameterError); ok {
			// Only report string constant code to luban
			response, reportErr = reportInvalidTask(taskInfo.TaskId, taskInfo.InvokeVersion, invalidParamCron, cronParameterErr.Code())
		} else {
			response, reportErr = reportInvalidTask(taskInfo.TaskId, taskInfo.InvokeVersion, invalidParamCron, err.Error())
		}
		scheduleLogger.WithFields(logrus.Fields{
			"expression": taskInfo.Cronat,
			"reportErr":  reportErr,
			"response":   response,
		}).WithError(err).Info("Report errors for invalid cron/rate/at expression")
		return err
	}
	// Special attributes for additional reporting of cron tasks
	if taskInfo.Repeat == models.RunTaskCron {
		cronScheduled, ok := timer.Schedule.(*timermanager.CronScheduled)
		if !ok {
			// Should never run into logic here
			errorMessage := "Unexpected schedule object when invoking onFinish callback for cron schedule!"
			scheduleLogger.Errorln(errorMessage)
			return errors.New(errorMessage)
		}
		scheduleLocation = cronScheduled.Location()
		onFinish = func() {
			onFinishLogger := log.GetLogger().WithFields(logrus.Fields{
				"TaskId":        taskInfo.TaskId,
				"InvokeVersion": taskInfo.InvokeVersion,
				"Phase":         "onFinishCallback",
			})

			cronScheduled, ok := timer.Schedule.(*timermanager.CronScheduled)
			if !ok {
				// Should never run into logic here
				onFinishLogger.Errorln("Unexpected schedule object when invoking onFinish callback for cron schedule!")
				return
			}

			if cronScheduled.NoNextRun() {
				response, err := sendStoppedOutput(taskInfo.TaskId, taskInfo.InvokeVersion, 0, 0, 0, 0, "", stopReasonCompleted)
				onFinishLogger.WithFields(logrus.Fields{
					"response": response,
				}).WithError(err).Infoln("Sent completion event for cron task on last invocation finished")
			}
		}
	}
	// then bind them to periodicTaskSchedule object
	periodicTaskSchedule.timer = timer
	periodicTaskSchedule.reusableInvocation = NewTask(taskInfo, scheduleLocation, onFinish)
	scheduleLogger.Info("Created timer and schedule object of periodic task")

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

// Cancel periodic task. If quietly is false, notify server the task is canceled.
func cancelPeriodicTask(taskInfo models.RunTaskInfo, quietly bool) error {
	timerManager := timermanager.GetTimerManager()
	if timerManager == nil {
		return errors.New("Global TimerManager instance is not initialized")
	}

	_periodicTaskSchedulesLock.Lock()
	defer _periodicTaskSchedulesLock.Unlock()

	cancelLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId":        taskInfo.TaskId,
		"InvokeVersion": taskInfo.InvokeVersion,
		"Quietly":       quietly,
		"Phase":         "Cancelling",
	})

	// 1. Check whether task is registered in local storage
	periodicTaskSchedule, ok := _periodicTaskSchedules[taskInfo.TaskId]
	if !ok && !quietly {
		response, err := sendStoppedOutput(taskInfo.TaskId, taskInfo.InvokeVersion, 0, 0, 0, 0, "", stopReasonKilled)
		cancelLogger.WithFields(logrus.Fields{
			"response": response,
		}).WithError(err).Warning("Force cancelling periodic task unregistered due to finished or previous errors")
		return nil
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
		runningInvocation.Cancel(quietly)
		cancelLogger.Infof("Canceled running invocation of periodic task")
	} else {
		cancelLogger.Infof("Not need to cancel running invocation of periodic task")
		// Since no running
		if !quietly {
			lastInvocation := periodicTaskSchedule.reusableInvocation
			lastInvocation.sendOutput("canceled", lastInvocation.getReportString(lastInvocation.output))
			cancelLogger.Infof("Sent canceled ACK with output of last invocation")
		}
	}
	return nil
}

func getPeriodicTask(taskName string) *Task {
	_periodicTaskSchedulesLock.Lock()
	defer _periodicTaskSchedulesLock.Unlock()

	periodicTaskSchedule, ok := _periodicTaskSchedules[taskName]
	if ok {
		return periodicTaskSchedule.reusableInvocation
	}
	return nil
}
