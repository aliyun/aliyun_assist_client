//go:build windows

package processutil

var (
	sleepCommand = []string{"powershell.exe", "-command", "Start-Sleep -Second 5"}
)
