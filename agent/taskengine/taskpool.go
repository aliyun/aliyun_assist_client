package taskengine

import (
	"sync"

	"github.com/aliyun/aliyun_assist_client/agent/flagging"
	"go.uber.org/atomic"
)

type TaskFunction func()

type taskDispatcher struct {
	taskQueue   []TaskFunction
	concurrency atomic.Int32

	cond sync.Cond
	l    sync.Mutex
}

var (
	_taskDispatcher     *taskDispatcher
	_taskDispatcherOnce sync.Once
)

func GetDispatcher() *taskDispatcher {
	_taskDispatcherOnce.Do(func() {
		tp := &taskDispatcher{}
		tp.cond = *sync.NewCond(&tp.l)
		go tp.run()
		_taskDispatcher = tp
	})
	return _taskDispatcher
}

// PutTask puts task into task queue
func (d *taskDispatcher) PutTask(task TaskFunction) {
	d.l.Lock()
	defer d.l.Unlock()
	d.taskQueue = append(d.taskQueue, task)
	d.cond.Signal()
}

// Concurrency returns the number of concurrent tasks
func (d *taskDispatcher) Concurrency() int {
	return int(d.concurrency.Load())
}

// MaxConcurrency returns the maximum number of concurrent tasks
func (d *taskDispatcher) MaxConcurrency() int64 {
	return flagging.GetTaskConcurrencyHardlimit()
}

func (d *taskDispatcher) popTask() TaskFunction {
	d.l.Lock()
	defer d.l.Unlock()
	return d.popTaskUnsafe()
}

func (d *taskDispatcher) popTaskUnsafe() TaskFunction {
	if len(d.taskQueue) > 0 {
		tf := d.taskQueue[0]
		d.taskQueue = d.taskQueue[1:]
		return tf
	}
	return nil
}

func (d *taskDispatcher) run() {
	d.l.Lock()
	defer d.l.Unlock()
	for {
		// If there are no tasks in the queue or the number of concurrent tasks
		// has reached the upper limit, it will be blocked here.
		for len(d.taskQueue) == 0 || d.concurrency.Load() >= int32(d.MaxConcurrency()) {
			d.cond.Wait()
		}
		d.concurrency.Add(1)
		go func(tf TaskFunction) {
			for {
				tf()
				// If there are tasks waiting to be executed in the queue,
				// continue to take them out for execution to avoid repeated
				// creation of goroutine.
				if tf = d.popTask(); tf == nil {
					break
				}
			}
			d.concurrency.Add(-1)
			d.cond.Signal()
		}(d.popTaskUnsafe())
	}
}
