// +build windows
package channel

func getGshellPath() (gshellPath string, err error) {
	return 	"\\\\.\\Global\\org.qemu.guest_agent.0", nil
}