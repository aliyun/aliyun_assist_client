// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package tsize

import (
	"os"
	"os/signal"
	"unsafe"

	"golang.org/x/sys/unix"
)

type winsize struct {
	rows uint16
	cols uint16
	x    uint16
	y    uint16
}

var unixSyscall = unix.Syscall

func getTerminalSize(fp *os.File) (s Size, err error) {
	ws := winsize{}

	_, _, errno := unixSyscall(
		unix.SYS_IOCTL,
		fp.Fd(),
		uintptr(unix.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)))

	if errno != 0 {
		err = errno
		return
	}

	s = Size{
		Width:  int(ws.cols),
		Height: int(ws.rows),
	}

	return
}

func getTerminalSizeChanges(sc chan Size, done chan struct{}) error {
	ch := make(chan os.Signal, 1)

	sig := unix.SIGWINCH

	signal.Notify(ch, sig)
	go func() {
		for {
			select {
			case <-ch:
				var err error
				s, err := getTerminalSize(os.Stdout)
				if err == nil {
					sc <- s
				}
			case <-done:
				signal.Reset(sig)
				close(ch)
				return
			}
		}
	}()

	return nil
}
