package taskengine

import (
	"errors"
	"sync"

	"github.com/aliyun/aliyun_assist_client/agent/log"
)

var taskFactory *TaskFactory
var lock    sync.Mutex

type TaskFactory struct {
	tasks map[string]*Task
	m    sync.Mutex
}

func GetTaskFactory() *TaskFactory {
	lock.Lock()
	defer lock.Unlock()

	if taskFactory == nil {
		taskFactory = &TaskFactory{
			tasks: make(map[string]*Task),
		}
	}

	return taskFactory
}

func (t *TaskFactory) AddTask(task *Task) error {
	return t.AddNamedTask(task.taskInfo.TaskId, task)
}

func (t *TaskFactory) AddNamedTask(name string, task *Task) error {
	t.m.Lock()
	defer t.m.Unlock()
	if _, ok := t.tasks[name]; ok {
		log.GetLogger().Error("Add named task failed: duplicated taskId ", name)
		return errors.New("Task name duplicated")
	}
	t.tasks[name] = task
	return nil
}

func (t *TaskFactory) GetTask(name string) (*Task, bool) {
	t.m.Lock()
	defer t.m.Unlock()

	task, ok := t.tasks[name]
	return task, ok
}

func (t *TaskFactory) RemoveTaskByName(name string) {
	t.m.Lock()
	defer t.m.Unlock()

	delete(t.tasks, name)
}

func (t *TaskFactory) ContainsTaskByName(name string) bool {
	t.m.Lock()
	defer t.m.Unlock()

	_, ok := t.tasks[name]
	return ok
}

// IsAnyTaskRunning returns true when any task exists in TaskFactory, otherwise
// false.
func (t *TaskFactory) IsAnyTaskRunning() bool {
	t.m.Lock()
	defer t.m.Unlock()

	return len(t.tasks) > 0
}

// IsAnyNonPeriodicTaskRunning scans each task registered in TaskFactory which
// means "running" and checks whether it is non-periodic task.
func (t *TaskFactory) IsAnyNonPeriodicTaskRunning() bool {
	t.m.Lock()
	defer t.m.Unlock()

	for _, task := range t.tasks {
		// NOTE: Currently we only consider non-periodic tasks
		if task.taskInfo.Cronat == "" {
			return true
		}
	}
	return false
}
