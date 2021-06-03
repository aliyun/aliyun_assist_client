package taskengine

import "sync"

const maxPendingTasks  = 50
const maxRunningTasks  = 10

type TaskFunction func()

var poolTask *taskPool
var lockPool    sync.Mutex

type taskPool struct {
	taskQueue   chan TaskFunction
}

func GetPool() *taskPool {
	lockPool.Lock()
	defer lockPool.Unlock()

	if poolTask == nil {
		poolTask = newTaskPool()
	}

	return poolTask
}

func newTaskPool() *taskPool {
	pool := &taskPool {
		taskQueue: make(chan TaskFunction, maxPendingTasks),
	}
	pool.start()
	return pool
}

func (p *taskPool) start() {
	for i := 0; i < maxRunningTasks; i++ {
		go func() {
			p.slave()
		}()
	}
}

func (p *taskPool) slave() {
	for task := range p.taskQueue {
		task()
	}
}

func (pool *taskPool) RunTask(task TaskFunction) {
	pool.taskQueue <- task
}

// Another global task pool for pre-checking tasks with limited concurrency
var (
	_precheckPool *taskPool
	_precheckPoolLock sync.Mutex
)

func GetPrecheckPool() *taskPool {
	if _precheckPool == nil {
		_precheckPoolLock.Lock()
		defer _precheckPoolLock.Unlock()

		if _precheckPool == nil {
			_precheckPool = newTaskPool()
		}
	}

	return _precheckPool
}
