//go:build !linux && !windows
// +build !linux,!windows

package serialport

import (
	"fmt"
)

type SerialPort struct{}

func (sp *SerialPort) ClosePort() {}

func (sp *SerialPort) WritePort(data []byte) error {
	return nil
}

func GetSerialPort() (*SerialPort, error) {
	return nil, fmt.Errorf("unsupported")
}
