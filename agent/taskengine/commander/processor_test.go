package commander

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/commandermanager"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskerrors"
	commander_client "github.com/aliyun/aliyun_assist_client/interprocess/commander/client"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCommanderProcessorCancel(t *testing.T) {
	var (
		mockClient    *commander_client.CommanderClient
		mockClientErr *taskerrors.CommanderError

		mockEffective bool
		mockCancelErr *taskerrors.CommanderError

		SetFinalStatusCalled bool
	)
	var c *commandermanager.Commander
	defer gomonkey.ApplyMethod(reflect.TypeOf(c), "Client", func(*commandermanager.Commander) (*commander_client.CommanderClient, *taskerrors.CommanderError) {
		return mockClient, mockClientErr
	}).Reset()
	var ct *commander_client.CommanderClient
	defer gomonkey.ApplyMethod(reflect.TypeOf(ct), "CancelSubmitted", func(*commander_client.CommanderClient, logrus.FieldLogger, context.Context, string) (bool, *taskerrors.CommanderError) {
		return mockEffective, mockCancelErr
	}).Reset()
	var p *CommanderProcessor
	defer gomonkey.ApplyMethod(reflect.TypeOf(p), "SetFinalStatus", func(*CommanderProcessor, int, int, *taskerrors.CommanderError) {
		SetFinalStatusCalled = true
	}).Reset()

	testCases := []struct {
		Name          string
		MockClient    *commander_client.CommanderClient
		MockClientErr *taskerrors.CommanderError
		MockEffective bool
		MockCancelErr *taskerrors.CommanderError

		ExpectedErr               *taskerrors.CommanderError
		ExpectedFinalStatusCalled bool
	}{
		{
			Name:          "client error",
			MockClientErr: taskerrors.NewCommanderError("client", "clent error", "clent error ..."),

			ExpectedErr:               taskerrors.NewCommanderError("client", "clent error", "clent error ..."),
			ExpectedFinalStatusCalled: false,
		},
		{
			Name:          "CancelSubmitted error",
			MockClient:    &commander_client.CommanderClient{},
			MockCancelErr: taskerrors.NewCommanderError("cancel", "cancel submitted", "cancel submitted ..."),

			ExpectedErr:               taskerrors.NewCommanderError("cancel", "cancel submitted", "cancel submitted ..."),
			ExpectedFinalStatusCalled: false,
		},
		{
			Name:       "CancelSubmitted, effective false",
			MockClient: &commander_client.CommanderClient{},

			ExpectedFinalStatusCalled: true,
		},
		{
			Name:          "CancelSubmitted, effective true",
			MockClient:    &commander_client.CommanderClient{},
			MockEffective: true,

			ExpectedFinalStatusCalled: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			SetFinalStatusCalled = false
			mockClient = tc.MockClient
			mockClientErr = tc.MockClientErr
			mockCancelErr = tc.MockCancelErr
			mockEffective = tc.MockEffective

			p := &CommanderProcessor{
				logger: logrus.New(),
			}
			err := p.Cancel()

			if tc.ExpectedErr == nil {
				assert.Nil(t, err)
			} else {
				assert.Equal(t, tc.ExpectedErr, err)
			}
			assert.Equal(t, tc.ExpectedFinalStatusCalled, SetFinalStatusCalled)
		})
	}

}

func TestCommanderProcessorSetFinalStatus(t *testing.T) {
	var (
		doneFuncCalledTimes int
	)
	p := CommanderProcessor{
		stdoutWriter: os.Stdout,
		stderrWriter: os.Stderr,
	}
	p.doneFunc = func() {
		doneFuncCalledTimes += 1
	}

	var (
		exitCode   = 100
		taskStatus = 1000
		taskErr    = taskerrors.NewCommanderError("cancel", "cancel submitted", "cancel submitted ...")
	)
	p.SetFinalStatus(exitCode, taskStatus, taskErr)
	assert.Equal(t, exitCode, p.exitCode)
	assert.Equal(t, taskStatus, p.resStatus)
	assert.Equal(t, taskErr, p.resError)
	assert.Equal(t, 1, doneFuncCalledTimes)
	assert.Nil(t, p.stdoutWriter)
	assert.Nil(t, p.stderrWriter)

	p.SetFinalStatus(exitCode+1, taskStatus+1, taskerrors.NewCommanderError("1", "1d", "1"))
	assert.Equal(t, exitCode, p.exitCode)
	assert.Equal(t, taskStatus, p.resStatus)
	assert.Equal(t, taskErr, p.resError)
	assert.Equal(t, 1, doneFuncCalledTimes)
	assert.Nil(t, p.stdoutWriter)
	assert.Nil(t, p.stderrWriter)
}

func TestCommanderProcessorWriteOutput(t *testing.T) {
	p := CommanderProcessor{
		stdoutWriter: os.Stdout,
		stderrWriter: os.Stderr,
	}
	p.WriteOutput(1, "hello")
	p.stdoutWriter = nil
	p.stderrWriter = nil
	p.WriteOutput(1, "hello")
}
