package taskengine

import (
	"sync"
)

var sessionFactory *SessionFactory
var session_lock    sync.Mutex

type SessionFactory struct {
	tasks map[string]*SessionTask
	m    sync.Mutex
}

func GetSessionFactory() *SessionFactory {
	session_lock.Lock()
	defer session_lock.Unlock()

	if sessionFactory == nil {
		sessionFactory = &SessionFactory{
			tasks: make(map[string]*SessionTask),
		}
	}

	return sessionFactory
}

func (t *SessionFactory) AddSessionTask(task *SessionTask) {
	t.m.Lock()
	defer t.m.Unlock()
	t.tasks[task.sessionId] = task
}


func (t *SessionFactory) GetTask(name string) (*SessionTask, bool) {
	t.m.Lock()
	defer t.m.Unlock()

	task, ok := t.tasks[name]
	return task, ok
}

func (t *SessionFactory) RemoveTask(name string) {
	t.m.Lock()
	defer t.m.Unlock()

	delete(t.tasks, name)
}

func (t *SessionFactory) ContainsTask(name string) bool {
	t.m.Lock()
	defer t.m.Unlock()

	_, ok := t.tasks[name]
	return ok
}

func (t *SessionFactory) IsAnyTaskRunning() bool {
	t.m.Lock()
	defer t.m.Unlock()

	return len(t.tasks) > 0
}


