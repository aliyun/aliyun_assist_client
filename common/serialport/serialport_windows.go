//go:build windows
// +build windows

package serialport

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	kernel32       = "kernel32.dll"
	defaultComPort = "\\\\.\\COM1"
)

// Dcb structure
// http://pinvoke.net/default.aspx/Structures/DCB.html
type Dcb struct {
	DCBlength  uint32
	BaudRate   uint32
	flags      [4]byte
	wReserved  uint16
	XonLim     uint16
	XoffLim    uint16
	ByteSize   byte
	Parity     byte
	StopBits   byte
	XonChar    byte
	XoffChar   byte
	ErrorChar  byte
	EOFChar    byte
	EvtChar    byte
	wReserved1 uint16
}

type SerialPort struct {
	kernel32   *syscall.DLL
	dcb        Dcb
	handle     syscall.Handle
	fileHandle *os.File
	port       string
}

// NewSerialPort creates a serial port object with predefined parameters.
func newSerialPort(port string) (sp *SerialPort) {
	var dcb Dcb
	dcb.DCBlength = uint32(unsafe.Sizeof(dcb))
	dcb.BaudRate = uint32(115200)
	dcb.ByteSize = 8
	dcb.Parity = 0
	dcb.StopBits = 0

	kernel32Loaded := syscall.MustLoadDLL(kernel32)

	return &SerialPort{
		kernel32:   kernel32Loaded,
		dcb:        dcb,
		handle:     0,
		fileHandle: nil,
		port:       port,
	}
}

// OpenPort opens the serial port which MUST be done before WritePort is called.
func (sp *SerialPort) OpenPort() (err error) {
	var comPortName *uint16
	comPortName, err = syscall.UTF16PtrFromString(sp.port)
	if err != nil {
		return fmt.Errorf("Error occurred while opening serial port: %v", err.Error())
	}

	// open COM1 port and create a handle.
	sp.handle, err = syscall.CreateFile(
		comPortName,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		0,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0)
	if err != nil {
		return fmt.Errorf("Error occurred while opening serial port: %v", err.Error())
	}

	sp.fileHandle = os.NewFile(uintptr(sp.handle), sp.port)

	// set communication state with default values.
	var r uintptr
	r, _, err = sp.kernel32.MustFindProc("SetCommState").Call(
		uintptr(sp.handle),
		uintptr(unsafe.Pointer(&sp.dcb)),
	)
	if r == 0 {
		return fmt.Errorf("Error occurred while opening serial port: %v", err.Error())
	}

	return nil
}

// ClosePort closes the serial port, which MUST be done at the end.
func (sp *SerialPort) ClosePort() error {
	if sp.fileHandle == nil {
		return fmt.Errorf("Error occurred while closing serial port: Port must be opened")
	}
	if err := sp.fileHandle.Close(); err != nil {
		return fmt.Errorf("Error occurred while closing serial port: %v", err.Error())
	}
	return nil
}

func (sp *SerialPort) WritePort(data []byte) error {
	if len(data) > MaxLenOfSingleWrite {
		data = data[:MaxLenOfSingleWrite]
	}
	var done uint32
	if err := syscall.WriteFile(sp.handle, data, &done, nil); err != nil {
		return fmt.Errorf("Error occurred while writing to serial port: %v", err.Error())
	}
	syscall.WriteFile(sp.handle, []byte{'\n'}, &done, nil)

	return nil
}

func GetSerialPort() (*SerialPort, error) {
	sp := newSerialPort(defaultComPort)
	if err := sp.OpenPort(); err != nil {
		return nil, err
	}
	return sp, nil
}
