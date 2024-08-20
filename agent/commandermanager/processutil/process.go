package processutil

import (
	"time"
)

// pidAlive checks whether a pid is alive.
func pidAlive(pid int) bool {
	return _pidAlive(pid)
}

// pidWait blocks for a process to exit.
func PidWait(pid int) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !pidAlive(pid) {
			break
		}
	}

	return nil
}

func RestartServer(pid int) error {
	return _restartServer(pid)
}
