package checknet

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/hybrid/instance"
	"github.com/aliyun/aliyun_assist_client/common/serialport"
	"github.com/stretchr/testify/assert"
)

func TestReportNetworkBlockToSerialPort(t *testing.T) {
	heartbeatErr := fmt.Errorf("heartbeat error")
	defer func() {
		_lastTimeReportNetworkBlockToSerialPort = nil
	}()

	var (
		callTimesOfGetSerialPort int
	)
	defer gomonkey.ApplyFunc(instance.IsHybrid, func() bool {
		return false
	}).Reset()
	defer gomonkey.ApplyFunc(serialport.GetSerialPort, func() (*serialport.SerialPort, error) {
		callTimesOfGetSerialPort += 1
		time.Sleep(time.Second * time.Duration(2))
		return &serialport.SerialPort{}, nil
	}).Reset()
	var s *serialport.SerialPort
	defer gomonkey.ApplyMethod(reflect.TypeOf(s), "WritePort", func(s *serialport.SerialPort, p []byte) error {
		assert.True(t, strings.HasPrefix(strings.TrimSpace(string(p)), _reportNetworkBlockPrefix))
		return nil
	}).Reset()

	wg := sync.WaitGroup{}
	groutineN := 3
	for i := 0; i < groutineN; i += 1 {
		wg.Add(1)
		go func() {
			ReportNetworkBlockToSerialPort(heartbeatErr)
			wg.Done()
		}()
	}

	wg.Wait()

	assert.Equal(t, 1, callTimesOfGetSerialPort)
}

func TestReportNetworkBlockToSerialPortInHybrid(t *testing.T) {
	heartbeatErr := fmt.Errorf("heartbeat error")
	defer func() {
		_lastTimeReportNetworkBlockToSerialPort = nil
	}()

	var (
		isHybrid            bool
		getSerialPortCalled bool
	)
	defer gomonkey.ApplyFunc(instance.IsHybrid, func() bool {
		return isHybrid
	}).Reset()
	defer gomonkey.ApplyFunc(serialport.GetSerialPort, func() (*serialport.SerialPort, error) {
		getSerialPortCalled = true
		return &serialport.SerialPort{}, nil
	}).Reset()
	var s *serialport.SerialPort
	defer gomonkey.ApplyMethod(reflect.TypeOf(s), "WritePort", func(s *serialport.SerialPort, p []byte) error {
		assert.True(t, strings.HasPrefix(strings.TrimSpace(string(p)), _reportNetworkBlockPrefix))
		return nil
	}).Reset()

	isHybrid = true
	ReportNetworkBlockToSerialPort(heartbeatErr)
	assert.Equal(t, !isHybrid, getSerialPortCalled)

	isHybrid = !isHybrid
	ReportNetworkBlockToSerialPort(heartbeatErr)
	assert.Equal(t, !isHybrid, getSerialPortCalled)
}

// ///////////


func TestReportNoNetworkCollectResToSerialPortt(t *testing.T) {
	contentReport := "content to report"

	var (
		callTimesOfGetSerialPort int
	)
	defer gomonkey.ApplyFunc(instance.IsHybrid, func() bool {
		return false
	}).Reset()
	defer gomonkey.ApplyFunc(serialport.GetSerialPort, func() (*serialport.SerialPort, error) {
		callTimesOfGetSerialPort += 1
		time.Sleep(time.Second * time.Duration(2))
		return &serialport.SerialPort{}, nil
	}).Reset()
	var s *serialport.SerialPort
	defer gomonkey.ApplyMethod(reflect.TypeOf(s), "WritePort", func(s *serialport.SerialPort, p []byte) error {
		assert.True(t, strings.HasPrefix(strings.TrimSpace(string(p)), _reportNoNetworkCollectResPrefix))
		return nil
	}).Reset()

	wg := sync.WaitGroup{}
	groutineN := 3
	for i := 0; i < groutineN; i += 1 {
		wg.Add(1)
		go func() {
			ReportNoNetworkCollectResToSerialPort(contentReport)
			wg.Done()
		}()
	}

	wg.Wait()

	assert.Equal(t, 1, callTimesOfGetSerialPort)
}

func TestReportNoNetworkCollectResToSerialPortInHybrid(t *testing.T) {
	contentReport := "content to report"

	var (
		isHybrid            bool
		getSerialPortCalled bool
	)
	defer gomonkey.ApplyFunc(instance.IsHybrid, func() bool {
		return isHybrid
	}).Reset()
	defer gomonkey.ApplyFunc(serialport.GetSerialPort, func() (*serialport.SerialPort, error) {
		getSerialPortCalled = true
		return &serialport.SerialPort{}, nil
	}).Reset()
	var s *serialport.SerialPort
	defer gomonkey.ApplyMethod(reflect.TypeOf(s), "WritePort", func(s *serialport.SerialPort, p []byte) error {
		fmt.Println(string(p))
		assert.True(t, strings.HasPrefix(strings.TrimSpace(string(p)), _reportNoNetworkCollectResPrefix))
		return nil
	}).Reset()

	isHybrid = true
	ReportNoNetworkCollectResToSerialPort(contentReport)
	assert.Equal(t, !isHybrid, getSerialPortCalled)

	isHybrid = !isHybrid
	ReportNoNetworkCollectResToSerialPort(contentReport)
	assert.Equal(t, !isHybrid, getSerialPortCalled)
}
