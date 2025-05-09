package taskengine

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun_assist_client/agent/flagging"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/models"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskerrors"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/common/requester"
	"github.com/aliyun/aliyun_assist_client/internal/testutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

func TestEnableFetchingTask(t *testing.T) {
	res := isEnabledFetchingTask()
	assert.Equal(t, false, res)
	EnableFetchingTask()
	res = isEnabledFetchingTask()
	assert.Equal(t, true, res)
}

func mockMetrics() {
	httpmock.Activate()
	requester.NilTransport.Set()
	const mockRegion = "cn-test100"
	testutil.MockMetaServer(mockRegion)

	httpmock.RegisterResponder("POST",
		fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/metrics", mockRegion),
		func(h *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(200, "success"), nil
		})
}

func TestFetch(t *testing.T) {
	mockMetrics()
	defer requester.NilTransport.Clear()
	defer httpmock.DeactivateAndReset()
	type args struct {
		from_kick   bool
		taskId      string
		taskType    int
		isColdstart bool
	}
	tests := []struct {
		name                  string
		args                  args
		want                  int
		isEnabledFetchingTask bool
		lockFetchingTaskLock  bool
	}{
		{
			name: "disableFetchingTask",
			args: args{},
			want: 0,
		},
		{
			name: "FetchingTaskLock.TryLockWithTimeout",
			args: args{},
			want: ErrUpdatingProcedureRunning,
		},
		{
			name: "from_kick",
			args: args{
				from_kick: true,
			},
			want: 10,
		},
		{
			name: "from_kick",
			args: args{
				from_kick: false,
			},
			want: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name != "disableFetchingTask" {
				EnableFetchingTask()
			}
			if tt.name == "FetchingTaskLock.TryLockWithTimeout" {
				FetchingTaskLock.Lock()
				defer FetchingTaskLock.Unlock()
			}
			if tt.name == "from_kick" {
				guard := gomonkey.ApplyFunc(fetchTasks, func(reason FetchReason, taskId string, taskType int, isColdstart bool) (int, error) {
					return 10, nil
				})
				defer guard.Reset()
			}
			if got := Fetch(tt.args.from_kick, tt.args.taskId, tt.args.taskType); got != tt.want {
				t.Errorf("Fetch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fetchTasks(t *testing.T) {
	mockMetrics()
	defer requester.NilTransport.Clear()
	defer httpmock.DeactivateAndReset()

	const mockRegion = "cn-test100"
	guard_GetServerHost := gomonkey.ApplyFunc(util.GetServerHost, func() string {
		return mockRegion + ".axt.aliyun.com"
	})
	defer guard_GetServerHost.Reset()

	type args struct {
		reason      FetchReason
		taskId      string
		taskType    int
		isColdstart bool
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "normal",
			args: args{},
			want: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "normal" {
				gomonkey.ApplyFunc(FetchTaskList, func(reason FetchReason, taskId string, taskType int, isColdstart bool) (*taskCollection, error) {
					return &taskCollection{
						runInfos:     []models.RunTaskInfo{models.RunTaskInfo{}},
						stopInfos:    []models.RunTaskInfo{models.RunTaskInfo{}},
						testInfos:    []models.RunTaskInfo{models.RunTaskInfo{}},
						sendFiles:    []models.SendFileTaskInfo{models.SendFileTaskInfo{}},
						sessionInfos: []models.SessionTaskInfo{models.SessionTaskInfo{}},
					}, nil
				})
			}
			if got, _ := fetchTasks(tt.args.reason, tt.args.taskId, tt.args.taskType, tt.args.isColdstart); got != tt.want {
				t.Errorf("fetchTasks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dispatchRunTask(t *testing.T) {
	flagging.InitConfig(logrus.New())
	mockMetrics()
	defer requester.NilTransport.Clear()
	defer httpmock.DeactivateAndReset()
	type args struct {
		taskInfo models.RunTaskInfo
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "taskHasExist",
			args: args{
				taskInfo: models.RunTaskInfo{
					TaskId: "abc",
				},
			},
		},
		{
			name: "taskRepeatOnce",
			args: args{
				taskInfo: models.RunTaskInfo{
					TaskId: "abc",
					Repeat: models.RunTaskOnce,
				},
			},
		},
		{
			name: "taskPeriod",
			args: args{
				taskInfo: models.RunTaskInfo{
					TaskId: "abc",
					Repeat: models.RunTaskCron,
				},
			},
		},
		{
			name: "taskUnknown",
			args: args{
				taskInfo: models.RunTaskInfo{
					TaskId: "abc",
					Repeat: models.RunTaskRepeatType("unknown"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "taskHasExist" {
				taskFactory := GetTaskFactory()
				task, _ := NewTask(tt.args.taskInfo, nil, nil, onTaskReportError)
				taskFactory.AddTask(task)
				defer taskFactory.RemoveTaskByName(tt.args.taskInfo.TaskId)
			} else if tt.name == "taskRepeatOnce" {
				var t *Task
				guard := gomonkey.ApplyMethod(reflect.TypeOf(t), "Run", func(*Task) (taskerrors.ErrorCode, error) {
					return 1, errors.New("some error")
				})
				defer guard.Reset()
			}
			dispatchRunTask(tt.args.taskInfo)
		})
	}
}

func Test_dispatchStopTask(t *testing.T) {
	mockMetrics()
	defer requester.NilTransport.Clear()
	defer httpmock.DeactivateAndReset()

	const mockRegion = "cn-test100"
	guard_GetServerHost := gomonkey.ApplyFunc(util.GetServerHost, func() string {
		return mockRegion + ".axt.aliyun.com"
	})
	defer guard_GetServerHost.Reset()
	httpmock.RegisterResponder("POST",
		util.GetStoppedOutputService(),
		func(h *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(200, "success"), nil
		})

	type args struct {
		taskInfo models.RunTaskInfo
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "taskHasExist",
			args: args{
				taskInfo: models.RunTaskInfo{
					TaskId: "abc",
					Repeat: models.RunTaskOnce,
				},
			},
		},
		{
			name: "taskRepeatOnce",
			args: args{
				taskInfo: models.RunTaskInfo{
					TaskId: "abc",
					Repeat: models.RunTaskOnce,
				},
			},
		},
		{
			name: "taskPeriod",
			args: args{
				taskInfo: models.RunTaskInfo{
					TaskId: "abc",
					Repeat: models.RunTaskCron,
				},
			},
		},
		{
			name: "taskUnknown",
			args: args{
				taskInfo: models.RunTaskInfo{
					TaskId: "abc",
					Repeat: models.RunTaskRepeatType("unknown"),
				},
			},
		},
	}
	var tsk *Task
	guard := gomonkey.ApplyMethod(reflect.TypeOf(tsk), "Cancel", func(*Task) error {
		return nil
	})
	defer guard.Reset()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "taskHasExist" {
				taskFactory := GetTaskFactory()
				task, _ := NewTask(tt.args.taskInfo, nil, nil, onTaskReportError)
				// task := &Task{
				// 	taskInfo: tt.args.taskInfo,
				// 	processer: &host.HostProcessor{},
				// }
				taskFactory.AddTask(task)
				defer taskFactory.RemoveTaskByName(tt.args.taskInfo.TaskId)
			} else if tt.name == "taskRepeatOnce" {
				// var t *Task
				// guard := gomonkey.ApplyMethod(reflect.TypeOf(t), "Cancel", func(*Task) {})
				// defer guard.Reset()
			}
			dispatchStopTask(tt.args.taskInfo)
		})
	}
}

func Test_dispatchTestTask(t *testing.T) {
	mockMetrics()
	defer requester.NilTransport.Clear()
	defer httpmock.DeactivateAndReset()
	type args struct {
		taskInfo models.RunTaskInfo
	}
	tests := []struct {
		name string
		args args
	}{{
		name: "taskHasExist",
		args: args{
			taskInfo: models.RunTaskInfo{
				TaskId: "abc",
				Repeat: models.RunTaskOnce,
			},
		},
	},
		{
			name: "taskRepeatOnce",
			args: args{
				taskInfo: models.RunTaskInfo{
					TaskId: "abc",
					Repeat: models.RunTaskOnce,
				},
			},
		},
		{
			name: "taskUnknown",
			args: args{
				taskInfo: models.RunTaskInfo{
					TaskId: "abc",
					Repeat: models.RunTaskRepeatType("unknown"),
				},
			},
		}, // TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "taskHasExist" {
				taskFactory := GetTaskFactory()
				task, _ := NewTask(tt.args.taskInfo, nil, nil, onTaskReportError)
				// task := &Task{
				// 	taskInfo: tt.args.taskInfo,
				// }
				taskFactory.AddTask(task)
				defer taskFactory.RemoveTaskByName(tt.args.taskInfo.TaskId)
			} else if tt.name == "taskRepeatOnce" {
				var t *Task
				guard := gomonkey.ApplyMethod(reflect.TypeOf(t), "PreCheck", func(*Task, bool) error { return nil })
				defer guard.Reset()
			}
			dispatchTestTask(tt.args.taskInfo)
		})
	}
}

func TestPeriodicTaskSchedule_startExclusiveInvocation(t *testing.T) {
	mockMetrics()
	defer requester.NilTransport.Clear()
	defer httpmock.DeactivateAndReset()
	type fields struct {
		timer              *timermanager.Timer
		reusableInvocation *Task
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "taskExist",
			fields: fields{
				timer: nil,
				reusableInvocation: &Task{
					taskInfo: models.RunTaskInfo{
						TaskId:        "abc",
						InvokeVersion: 1,
					},
				},
			},
		},
		{
			name: "normal",
			fields: fields{
				timer: nil,
				reusableInvocation: &Task{
					taskInfo: models.RunTaskInfo{
						TaskId:        "abc",
						InvokeVersion: 1,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "taskExist" {
				taskFactory := GetTaskFactory()
				task, _ := NewTask(tt.fields.reusableInvocation.taskInfo, nil, nil, onTaskReportError)
				// task := &Task{
				// 	taskInfo: tt.fields.reusableInvocation.taskInfo,
				// }
				taskFactory.AddTask(task)
				defer taskFactory.RemoveTaskByName(tt.fields.reusableInvocation.taskInfo.TaskId)
			} else if tt.name == "normal" {
				var t *Task
				guard := gomonkey.ApplyMethod(reflect.TypeOf(t), "Run", func(*Task) (taskerrors.ErrorCode, error) {
					return 1, errors.New("some error")
				})
				defer guard.Reset()
			}
			s := &PeriodicTaskSchedule{
				timer:              tt.fields.timer,
				reusableInvocation: tt.fields.reusableInvocation,
			}
			s.startExclusiveInvocation()
		})
	}
}

func Test_schedulePeriodicTask(t *testing.T) {
	mockMetrics()
	defer requester.NilTransport.Clear()
	defer httpmock.DeactivateAndReset()
	type args struct {
		taskInfo models.RunTaskInfo
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "TimerManagerNotInitialized",
			wantErr: true,
		},
		{
			name: "taskExist",
			args: args{
				taskInfo: models.RunTaskInfo{
					InstanceId:    "fake-instance-id",
					CommandType:   "RunShellScript",
					TaskId:        "abc",
					CommandId:     "fake-command-id",
					TimeOut:       "60",
					InvokeVersion: 1,
				},
			},
			wantErr: false,
		},
		{
			name: "normal",
			args: args{
				taskInfo: models.RunTaskInfo{
					TaskId:        "abc",
					Cronat:        "0 0 0 1 1 1",
					InvokeVersion: 1,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "TimerManagerNotInitialized" {
				guard := gomonkey.ApplyFunc(timermanager.GetTimerManager, func() *timermanager.TimerManager { return nil })
				defer guard.Reset()
			} else if tt.name == "taskExist" {
				timermanager.InitTimerManager()
				_periodicTaskSchedulesLock.Lock()
				task, _ := NewTask(tt.args.taskInfo, nil, nil, onTaskReportError)
				_periodicTaskSchedules[tt.args.taskInfo.TaskId] = &PeriodicTaskSchedule{
					timer:              nil,
					reusableInvocation: task,
				}
				_periodicTaskSchedulesLock.Unlock()
				defer func() {
					_periodicTaskSchedulesLock.Lock()
					delete(_periodicTaskSchedules, tt.args.taskInfo.TaskId)
					_periodicTaskSchedulesLock.Unlock()
				}()
			} else if tt.name == "normal" {
				timermanager.InitTimerManager()
				var t *timermanager.Timer
				guard := gomonkey.ApplyMethod(reflect.TypeOf(t), "Run", func(*timermanager.Timer) (*timermanager.Timer, error) { return nil, errors.New("some error") })
				defer guard.Reset()
			}
			if err := schedulePeriodicTask(tt.args.taskInfo); (err != nil) != tt.wantErr {
				t.Errorf("schedulePeriodicTask() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_cancelPeriodicTask(t *testing.T) {
	mockMetrics()
	defer requester.NilTransport.Clear()
	defer httpmock.DeactivateAndReset()

	const mockRegion = "cn-test100"
	guard_GetServerHost := gomonkey.ApplyFunc(util.GetServerHost, func() string {
		return mockRegion + ".axt.aliyun.com"
	})
	defer guard_GetServerHost.Reset()
	httpmock.RegisterResponder("POST",
		util.GetStoppedOutputService(),
		func(h *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(200, "success"), nil
		})

	type args struct {
		taskInfo models.RunTaskInfo
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// {
		// 	name:    "TimerManagerNotInitialized",
		// 	wantErr: true,
		// },
		// {
		// 	name: "taskNotExist",
		// 	args: args{
		// 		taskInfo: models.RunTaskInfo{
		// 			TaskId: "abc",
		// 		},
		// 	},
		// 	wantErr: true,
		// },
		{
			name: "cancleTask",
			args: args{
				taskInfo: models.RunTaskInfo{
					TaskId: "abc",
				},
			},
			wantErr: false,
		},
		{
			name: "noNeedCancelTask",
			args: args{
				taskInfo: models.RunTaskInfo{
					TaskId: "abc",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "TimerManagerNotInitialized" {
				guard := gomonkey.ApplyFunc(timermanager.GetTimerManager, func() *timermanager.TimerManager { return nil })
				defer guard.Reset()
			} else if tt.name == "taskNotExist" {
				timermanager.InitTimerManager()
			} else if tt.name == "cancleTask" {
				timermanager.InitTimerManager()
				_periodicTaskSchedulesLock.Lock()
				timerManager := timermanager.GetTimerManager()
				timer, _ := timerManager.CreateCronTimer(func() {}, "0 0 0 1 1 1")
				task, _ := NewTask(tt.args.taskInfo, nil, nil, onTaskReportError)
				_periodicTaskSchedules[tt.args.taskInfo.TaskId] = &PeriodicTaskSchedule{
					timer:              timer,
					reusableInvocation: task,
				}
				_periodicTaskSchedulesLock.Unlock()
				defer func() {
					_periodicTaskSchedulesLock.Lock()
					delete(_periodicTaskSchedules, tt.args.taskInfo.TaskId)
					_periodicTaskSchedulesLock.Unlock()
				}()
				task, _ = NewTask(tt.args.taskInfo, nil, nil, onTaskReportError)
				GetTaskFactory().AddTask(task)
				defer GetTaskFactory().RemoveTaskByName(tt.args.taskInfo.TaskId)
				var t *Task
				guard := gomonkey.ApplyMethod(reflect.TypeOf(t), "Cancel", func(*Task) error {
					return nil
				})
				defer guard.Reset()
			} else if tt.name == "noNeedCancelTask" {
				timermanager.InitTimerManager()
				_periodicTaskSchedulesLock.Lock()
				timerManager := timermanager.GetTimerManager()
				timer, _ := timerManager.CreateCronTimer(func() {}, "0 0 0 1 1 1")
				task, _ := NewTask(tt.args.taskInfo, nil, nil, onTaskReportError)
				_periodicTaskSchedules[tt.args.taskInfo.TaskId] = &PeriodicTaskSchedule{
					timer:              timer,
					reusableInvocation: task,
				}
				_periodicTaskSchedulesLock.Unlock()
				defer func() {
					_periodicTaskSchedulesLock.Lock()
					delete(_periodicTaskSchedules, tt.args.taskInfo.TaskId)
					_periodicTaskSchedulesLock.Unlock()
				}()
				guard := gomonkey.ApplyFunc(util.HttpPost, func(string, string, string) (string, error) { return "", nil })
				defer guard.Reset()
			}
			if err := cancelPeriodicTask(tt.args.taskInfo, false); (err != nil) != tt.wantErr {
				t.Errorf("cancelPeriodicTask() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFetchStartUp(t *testing.T) {
	var (
		fetchTasksErr     error
		fetchReasonList   []FetchReason
		fetchTasksErrList []error
	)

	defer gomonkey.ApplyFunc(fetchTasks, func(reason FetchReason, taskId string, taskType int, isColdstart bool) (int, error) {
		fetchReasonList = append(fetchReasonList, reason)
		err := fetchTasksErr
		fetchTasksErrList = append(fetchTasksErrList, err)
		return 10, err
	}).Reset()
	defer gomonkey.ApplyFunc(flagging.IsColdstart, func() (bool, error) {
		return false, nil
	}).Reset()

	testNormalFetch := func(t *testing.T) {
		_startupFetchedFinished = false
		_startupFetchedDone = make(chan struct{})

		fetchTasksErr = nil
		fetchReasonList = []FetchReason{}
		fetchTasksErrList = []error{}

		for i := 0; i < 5; i += 1 {
			Fetch(i != 0, "", 0)
		}

		assert.Equal(t, 5, len(fetchReasonList))
		assert.Equal(t, []FetchReason{FetchOnStartup, FetchOnKickoff, FetchOnKickoff, FetchOnKickoff, FetchOnKickoff}, fetchReasonList)

		// wait retrying goroutine return
		time.Sleep(time.Second)
	}

	testSuccessInRetrying := func(t *testing.T) {
		_startupFetchedFinished = false
		_startupFetchedDone = make(chan struct{})

		fetchTasksErr = errors.New("fetch error")
		fetchReasonList = []FetchReason{}
		fetchTasksErrList = []error{}

		fetchRetryInterval = time.Duration(10) * time.Millisecond

		Fetch(false, "", 0)
		time.Sleep(time.Duration(10) * time.Millisecond)
		fetchTasksErr = nil
		// wait retrying goroutine return
		time.Sleep(time.Second)

		for i := range fetchReasonList {
			assert.Equal(t, FetchOnStartup, fetchReasonList[i])
		}
		for i := 0; i < len(fetchTasksErrList); i += 1 {
			if i != len(fetchTasksErrList)-1 {
				assert.NotNil(t, fetchTasksErrList[i])
			} else {
				assert.Nil(t, fetchTasksErrList[i])
			}
		}
	}

	// Initially fetchTasks() will fail, and the first Fetch() will continue to retry with
	// reason=FetchOnStartup until fetchTasks() succeeds. During this period, other
	// Fetch() will try reason=FetchOnStartup too.
	testFetchFailAtBeginning := func(t *testing.T) {
		_startupFetchedFinished = false
		_startupFetchedDone = make(chan struct{})

		fetchTasksErr = errors.New("fetch error")
		fetchReasonList = []FetchReason{}
		fetchTasksErrList = []error{}

		fetchRetryInterval = time.Duration(10) * time.Millisecond
		taskSizeList := []int{}

		fetchDone := sync.WaitGroup{}
		fetchStart := make(chan struct{})
		for i := 0; i < 50; i += 1 {
			fetchDone.Add(1)
			go func(from_kick bool) {
				<-fetchStart
				if from_kick {
					time.Sleep(time.Duration(10) * time.Millisecond)
				}
				taskSizeList = append(taskSizeList, Fetch(from_kick, "", 0))
				fetchDone.Done()
			}(i != 0)
		}
		close(fetchStart)
		// only the first Fetch() goroutine will keep retrying, other Fetch() goroutines will exit directly
		time.Sleep(time.Duration(50) * time.Millisecond)
		// wait the first Fetch() goroutine return
		fetchTasksErr = nil
		fetchDone.Wait()

		// the first Fetch() will try more than 5 times (50/10=5), keep reason=FetchOnStartup
		assert.Greater(t, len(fetchReasonList), 5)
		success := 0
		fmt.Println(len(fetchReasonList))
		for i := 0; i < len(fetchReasonList); i++ {
			// 从失败到第一次成功都应该是 startup 的
			// 第二次成功及以后得都应该是 kickoff 的
			if fetchTasksErrList[i] != nil {
				assert.Equal(t, FetchOnStartup, fetchReasonList[i])
			}
			if fetchTasksErrList[i] == nil {
				success++
			}
			if success == 1 {
				fmt.Println(i)
				assert.Equal(t, FetchOnStartup, fetchReasonList[i])
			}
			if success > 1 {
				assert.Equal(t, FetchOnKickoff, fetchReasonList[i])
			}
		}

		// wait retrying goroutine return
		time.Sleep(time.Second)
	}

	// Initially fetchTasks() will fail, when first Fetch() continues to retry with
	// reason=FetchOnStartup, if other Fetch() success the first Fetch() will return.
	testTrigerNextRetry := func(t *testing.T) {
		_startupFetchedFinished = false
		_startupFetchedDone = make(chan struct{})

		fetchTasksErr = errors.New("fetch error")
		fetchReasonList = []FetchReason{}
		fetchTasksErrList = []error{}

		fetchRetryInterval = time.Duration(100) * time.Hour

		// the Fetch() goroutine will wait for next retry, because fetchRetryInterval is very large
		fetchDone := sync.WaitGroup{}
		fetchDone.Add(1)
		go func() {
			Fetch(false, "", 0)
			fetchDone.Done()
		}()
		time.Sleep(time.Duration(100) * time.Millisecond)

		assert.Equal(t, 1, len(fetchReasonList))
		assert.Equal(t, FetchOnStartup, fetchReasonList[0])

		// call Fetch() to fetch tasks for startup
		for i := 0; i < 20; i += 1 {
			Fetch(true, "", 0)
		}

		fetchTasksErr = nil
		// This successful Fetch() will tell the first Fetch() to return
		Fetch(true, "", 0)
		fetchDone.Wait()

		// call Fetch() to fetch tasks for kickoff
		for i := 0; i < 20; i += 1 {
			Fetch(true, "", 0)
		}

		assert.Equal(t, 42, len(fetchReasonList))
		for r := range fetchReasonList[:21] {
			assert.Equal(t, FetchOnStartup, fetchReasonList[r])
			assert.NotNil(t, fetchTasksErrList[r])
		}
		assert.Equal(t, FetchOnStartup, fetchReasonList[21])
		assert.Nil(t, fetchTasksErrList[21])
		fetchReasonList = fetchReasonList[22:]
		fetchTasksErrList = fetchTasksErrList[22:]
		for r := range fetchReasonList {
			assert.Equal(t, FetchOnKickoff, fetchReasonList[r])
			assert.Nil(t, fetchTasksErrList[r])
		}

		// wait retrying goroutine return
		time.Sleep(time.Second)
	}

	EnableFetchingTask()
	fmt.Println("------- testNormalFetch ----------------")
	t.Run("testNormalFetch", testNormalFetch)

	fmt.Println("------- testSuccessInRetrying -----------------")
	t.Run("testSuccessInRetrying", testSuccessInRetrying)

	fmt.Println("------- testFetchFailAtBeginning ----------------")
	t.Run("testFetchFailAtBeginning", testFetchFailAtBeginning)

	fmt.Println("------- testTrigerNextRetry ----------------")
	t.Run("testTrigerNextRetry", testTrigerNextRetry)
}
