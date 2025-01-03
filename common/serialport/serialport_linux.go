//go:build linux
// +build linux

package serialport

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	comport   = "/dev/ttyS0"
	comportPV = "/dev/hvc0"
)

type SerialPort struct {
	fileHandle *os.File
}

// newSerialPort creates a serial port object with predefined parameters.
func newSerialPort() (sp *SerialPort) {
	return &SerialPort{
		fileHandle: nil,
	}
}

// OpenPort opens the serial port which MUST be done before WritePort is called.
func (sp *SerialPort) openPort(name string) (err error) {
	fileHandle, err := os.OpenFile(name, syscall.O_RDWR, 0)

	if err != nil {
		return fmt.Errorf("Unable to open serial port %v: %v\n", name, err.Error())
	}

	baudRate := uint32(syscall.B115200)
	state := syscall.Termios{
		Cflag:  syscall.CS8 | syscall.CREAD | syscall.B115200,
		Oflag:  0,
		Ospeed: baudRate,
	}

	fd := fileHandle.Fd()
	if _, _, err := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(syscall.TCSETS),
		uintptr(unsafe.Pointer(&state)),
		0,
		0,
		0,
	); err != 0 {
		return fmt.Errorf("Unable to configure serial port %v: %v\n", name, err.Error())
	}

	sp.fileHandle = fileHandle

	return nil
}

func (sp *SerialPort) OpenPort() (err error) {
	if err = sp.openPort(comport); err != nil {
		if err = sp.openPort(comportPV); err != nil {
			return fmt.Errorf("Error opening serial port: %v", err.Error())
		}
	}

	return nil
}

func (sp *SerialPort) ClosePort() {
	if sp.fileHandle == nil {
		return
	}
	sp.fileHandle.Close()
	return
}

func (sp *SerialPort) WritePort(data []byte) error {
	if len(data) > MaxLenOfSingleWrite {
		data = data[:MaxLenOfSingleWrite]
	}
	if _, err := sp.fileHandle.Write(data); err != nil {
		return fmt.Errorf("Error occurred while writing to serial port: %v\n", err.Error())
	}
	sp.fileHandle.Write([]byte{'\n'})
	return nil
}

func GetSerialPort() (*SerialPort, error) {
	sp := newSerialPort()
	if err := sp.OpenPort(); err != nil {
		return nil, err
	}
	return sp, nil
}
