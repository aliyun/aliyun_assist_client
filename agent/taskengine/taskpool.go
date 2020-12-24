package taskengine

import "sync"

const maxPendingTasks  = 50
const maxRunningTasks  = 10

var poolTask *taskPool
var lockPool    sync.Mutex

type taskPool struct {
	taskQueue   chan *Task
}

func GetPool() *taskPool {
	lockPool.Lock()
	defer lockPool.Unlock()

	if poolTask == nil {
		poolTask = &taskPool {
			taskQueue: make(chan *Task, maxPendingTasks),
		}
		poolTask.start()
	}

	return poolTask
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
		task.Run()
		taskFactory := GetTaskFactory()
		taskFactory.RemoveTaskByName(task.taskInfo.TaskId)
	}
}

func (pool *taskPool) RunTask(task *Task) {
	pool.taskQueue <- task
}




