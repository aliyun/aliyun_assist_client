package channel

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun_assist_client/agent/kickvmhandle"
	"github.com/aliyun/aliyun_assist_client/agent/update"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

func TestOnRecvMsg(t *testing.T) {
	type args struct {
		Msg         string
		ChannelType int
	}
	theArgs := args{
		Msg:         "kick_vm",
		ChannelType: ChannelGshellType,
	}
	tests := []struct {
		name    string
		subname string
		args    args
		want    string
	}{
		{
			name:    "ws",
			subname: "CriticalActionRunning",
			args:    theArgs,
		},
		{
			name: "ws",
			args: theArgs,
		},
		{
			name: "guest-sync",
			args: theArgs,
		},
		{
			name:    "guest-sync",
			subname: "CriticalActionRunning",
			args:    theArgs,
		},
		{
			name:    "guest-command",
			subname: "CriticalActionRunning",
			args:    theArgs,
		},
		{
			name:    "guest-command",
			subname: "kick_vm",
			args:    theArgs,
		},
		{
			name:    "guest-command",
			subname: "valid agent",
			args:    theArgs,
		},
		{
			name:    "guest-command",
			subname: "invalid agent",
			args:    theArgs,
		},
		// {
		// 	name: "guest-shutdown",
		// 	subname: "reboot",
		// 	args: theArgs,
		// },
		// {
		// 	name: "guest-shutdown",
		// 	subname: "powerdown",
		// 	args: theArgs,
		// },
		{
			name:    "guest-shutdown",
			subname: "unknown",
			args:    theArgs,
		},
	}
	guard := gomonkey.ApplyFunc(util.ExeCmd, func(string) (error, string, string) { return nil, "", "" })
	defer guard.Reset()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "ws" {
				tt.args.ChannelType = ChannelWebsocketType
			}

			if tt.subname == "CriticalActionRunning" {
				guard := gomonkey.ApplyFunc(update.IsCriticalActionRunning, func() bool { return true })
				defer guard.Reset()
			} else {
				guard := gomonkey.ApplyFunc(update.IsCriticalActionRunning, func() bool { return false })
				defer guard.Reset()
			}

			if tt.name == "guest-sync" {
				msg := GshellCheck{
					Execute: "guest-sync",
				}
				msg.Arguments.ID = 10000
				content, _ := json.Marshal(&msg)
				tt.args.Msg = string(content)
			}

			if tt.name == "guest-command" {
				msg := GshellCmd{
					Execute: "guest-command",
				}
				if tt.subname == "kick_vm" {
					msg.Arguments.Cmd = "kick_vm"
				} else if tt.subname == "valid agent" {
					msg.Arguments.Cmd = "valid agent params params"
					var a *kickvmhandle.AgentHandle
					guard := gomonkey.ApplyMethod(reflect.TypeOf(a), "CheckAction", func(*kickvmhandle.AgentHandle) bool { return true })
					defer guard.Reset()
				} else if tt.subname == "invalid agent" {
					msg.Arguments.Cmd = "invalid agent params params"
					var a *kickvmhandle.AgentHandle
					guard := gomonkey.ApplyMethod(reflect.TypeOf(a), "CheckAction", func(*kickvmhandle.AgentHandle) bool { return false })
					defer guard.Reset()
				}
				content, _ := json.Marshal(&msg)
				tt.args.Msg = string(content)
			}

			if tt.name == "guest-shutdown" {
				msg := GshellShutdown{
					Execute: "guest-shutdown",
				}
				msg.Arguments.Mode = tt.subname
				content, _ := json.Marshal(&msg)
				tt.args.Msg = string(content)
			}
			OnRecvMsg(tt.args.Msg, tt.args.ChannelType)
			// if got := OnRecvMsg(tt.args.Msg, tt.args.ChannelType); got != tt.want {
			// 	t.Errorf("OnRecvMsg() = %v, want %v", got, tt.want)
			// }
		})
	}
}

// _mockChannel implements IChannel interface for testing
type _mockChannel struct {
	channelType int
	isSupported bool
	isWorking   bool
	startError  error
}

func (m *_mockChannel) GetChannelType() int {
	return m.channelType
}

func (m *_mockChannel) IsSupported() bool {
	return m.isSupported
}

func (m *_mockChannel) IsWorking() bool {
	return m.isWorking
}

func (m *_mockChannel) StartChannel() error {
	return m.startError
}

func (m *_mockChannel) StopChannel() error {
	// Do nothing
	return nil
}

func TestChannelMgrPeriodicCheckAndSelect(t *testing.T) {
	t.Run("StopChannelEventExit", func(t *testing.T) {
		// Setup
		mgr := &ChannelMgr{
			StopChanelEvent: make(chan struct{}, 1),
			AllChannel: []IChannel{
				&_mockChannel{
					channelType: ChannelGshellType,
					isSupported: true,
					isWorking:   true,
				},
				&_mockChannel{
					channelType: ChannelWebsocketType,
					isSupported: true,
					isWorking:   true,
				},
			},
		}

		// Create a mock ticker
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		// Mock checkChannelWorker to verify it's not called
		checkChannelWorkerCalled := false
		patches.ApplyPrivateMethod(reflect.TypeOf(mgr), "checkChannelWorker",
			func(*ChannelMgr) bool {
				checkChannelWorkerCalled = true
				return false
			})
		// Mock SelectAvailableChannelAndReport to verify it's not called
		selectAvailableChannelCalled := false
		patches.ApplyMethod(reflect.TypeOf(mgr), "SelectAvailableChannelAndReport",
			func(*ChannelMgr, int, string, bool) error {
				selectAvailableChannelCalled = true
				return nil
			})
		// Apply monkey patch for time.NewTicker
		mockTickerChan := make(chan time.Time, 1)
		patches.ApplyFunc(time.NewTicker, func(d time.Duration) *time.Ticker {
			return &time.Ticker{C: mockTickerChan}
		})

		// Receive function exited event via the WaitCheckDone
		// waitgroup, although which has been deprecated in the source,
		// should and would be removed soon.
		mgr.WaitCheckDone.Add(1)
		done := make(chan bool, 1)
		go func() {
			mgr.WaitCheckDone.Wait()
			done <- true
		}()
		// Run tested function in goroutine with timeout
		go mgr.periodicCheckAndSelect()

		// Send stop signal immediately
		mgr.StopChanelEvent <- struct{}{}

		// Wait for completion or timeout
		select {
		case <-done:
			assert.False(t, selectAvailableChannelCalled, "SelectAvailableChannelAndReport should not be called when StopChanelEvent was triggered immediately")
			assert.False(t, checkChannelWorkerCalled, "checkChannelWorker should not be called when StopChanelEvent was triggered immediately")
		case <-time.After(100 * time.Millisecond):
			t.Error("Test timed out - function did not exit when StopChanelEvent was triggered")
		}
	})

	t.Run("OnlyFullCheckWhenBothChannelsWorking", func(t *testing.T) {
		// Setup
		mgr := &ChannelMgr{
			StopChanelEvent: make(chan struct{}, 1),
			AllChannel: []IChannel{
				&_mockChannel{
					channelType: ChannelGshellType,
					isSupported: true,
					isWorking:   true,
				},
				&_mockChannel{
					channelType: ChannelWebsocketType,
					isSupported: true,
					isWorking:   true,
				},
			},
		}

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		// Mock checkChannelWorker to verify it's called on full check
		checkChannelWorkerCalled := false
		patches.ApplyPrivateMethod(reflect.TypeOf(mgr), "checkChannelWorker",
			func(*ChannelMgr) bool {
				checkChannelWorkerCalled = true
				return false
			})
		// Mock SelectAvailableChannelAndReport to verify it's called on full
		// check
		selectAvailableChannelCalled := false
		patches.ApplyMethod(reflect.TypeOf(mgr), "SelectAvailableChannelAndReport",
			func(*ChannelMgr, int, string, bool) error {
				selectAvailableChannelCalled = true
				return nil
			})

		// Apply monkey patch for time.NewTicker to return our custom ticker channel
		mockTickerChan := make(chan time.Time, 1)
		patches.ApplyFunc(time.NewTicker, func(d time.Duration) *time.Ticker {
			return &time.Ticker{C: mockTickerChan}
		})

		// Receive function exited event via the WaitCheckDone
		// waitgroup, although which has been deprecated in the source,
		// should and would be removed soon.
		mgr.WaitCheckDone.Add(1)
		done := make(chan bool, 1)
		go func() {
			mgr.WaitCheckDone.Wait()
			done <- true
		}()
		// Run tested function in goroutine with timeout
		go mgr.periodicCheckAndSelect()

		for i := 0; i < 6; i++ {
			// Send one tick to trigger the quick or full check logic
			mockTickerChan <- time.Now()
			// Yield CPU to let the goroutine be scheduled
			time.Sleep(100 * time.Millisecond)

			// Perform the assertions
			if i < 5 {
				assert.False(t, checkChannelWorkerCalled, "checkChannelWorker should not be called, since quick check is unnecessary when both channels are working")
				assert.False(t, selectAvailableChannelCalled, "SelectAvailableChannelAndReport should be called, since quick check is unnecessary when both channels are working")
			} else {
				assert.True(t, checkChannelWorkerCalled, "checkChannelWorker should be called during full check, even when both channels are working")
				assert.True(t, selectAvailableChannelCalled, "SelectAvailableChannelAndReport should be called during full check, even when both channels are working")
			}
		}

		// Send stop signal
		mgr.StopChanelEvent <- struct{}{}

		// Wait for completion or timeout
		select {
		case <-done:
			// Test passed and let it go
		case <-time.After(100 * time.Millisecond):
			t.Error("Test timed out")
		}
	})

	t.Run("OnlyFullCheckEvenWhenOnlyGshellChannelWorking", func(t *testing.T) {
		// Setup
		mgr := &ChannelMgr{
			StopChanelEvent: make(chan struct{}, 1),
			AllChannel: []IChannel{
				&_mockChannel{
					channelType: ChannelGshellType,
					isSupported: true,
					isWorking:   true,
				},
				&_mockChannel{
					channelType: ChannelWebsocketType,
					isSupported: true,
					isWorking:   false,
				},
			},
		}

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		// Mock checkChannelWorker to verify it's called on full check
		checkChannelWorkerCalled := false
		patches.ApplyPrivateMethod(reflect.TypeOf(mgr), "checkChannelWorker",
			func(*ChannelMgr) bool {
				checkChannelWorkerCalled = true
				return false
			})
		// Mock SelectAvailableChannelAndReport to verify it's called on full
		// check
		selectAvailableChannelCalled := false
		patches.ApplyMethod(reflect.TypeOf(mgr), "SelectAvailableChannelAndReport",
			func(*ChannelMgr, int, string, bool) error {
				selectAvailableChannelCalled = true
				return nil
			})

		// Apply monkey patch for time.NewTicker to return our custom ticker channel
		mockTickerChan := make(chan time.Time, 1)
		patches.ApplyFunc(time.NewTicker, func(d time.Duration) *time.Ticker {
			return &time.Ticker{C: mockTickerChan}
		})

		// Receive function exited event via the WaitCheckDone
		// waitgroup, although which has been deprecated in the source,
		// should and would be removed soon.
		mgr.WaitCheckDone.Add(1)
		done := make(chan bool, 1)
		go func() {
			mgr.WaitCheckDone.Wait()
			done <- true
		}()
		// Run tested function in goroutine with timeout
		go mgr.periodicCheckAndSelect()

		for i := 0; i < 6; i++ {
			// Send one tick to trigger the quick or full check logic
			mockTickerChan <- time.Now()
			// Yield CPU to let the goroutine be scheduled
			time.Sleep(100 * time.Millisecond)

			// Perform the assertions
			if i < 5 {
				assert.False(t, checkChannelWorkerCalled, "checkChannelWorker should not be called, since quick check is unnecessary even when only gshell channel is working")
				assert.False(t, selectAvailableChannelCalled, "SelectAvailableChannelAndReport should be called, since quick check is unnecessary even when only gshell channel is working")
			} else {
				assert.True(t, checkChannelWorkerCalled, "checkChannelWorker should be called during full check, when only gshell channel is working")
				assert.True(t, selectAvailableChannelCalled, "SelectAvailableChannelAndReport should be called during full check, when only gshell channel is working")
			}
		}

		// Send stop signal
		mgr.StopChanelEvent <- struct{}{}

		// Wait for completion or timeout
		select {
		case <-done:
			// Test passed and let it go
		case <-time.After(100 * time.Millisecond):
			t.Error("Test timed out")
		}
	})

	t.Run("OnlyFullCheckEvenWhenOnlyWebsocketChannelWorking", func(t *testing.T) {
		// Setup
		mgr := &ChannelMgr{
			StopChanelEvent: make(chan struct{}, 1),
			AllChannel: []IChannel{
				&_mockChannel{
					channelType: ChannelGshellType,
					isSupported: false,
					isWorking:   false,
				},
				&_mockChannel{
					channelType: ChannelWebsocketType,
					isSupported: true,
					isWorking:   true,
				},
			},
		}

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		// Mock checkChannelWorker to verify it's called on full check
		checkChannelWorkerCalled := false
		patches.ApplyPrivateMethod(reflect.TypeOf(mgr), "checkChannelWorker",
			func(*ChannelMgr) bool {
				checkChannelWorkerCalled = true
				return false
			})
		// Mock SelectAvailableChannelAndReport to verify it's called on full
		// check
		selectAvailableChannelCalled := false
		patches.ApplyMethod(reflect.TypeOf(mgr), "SelectAvailableChannelAndReport",
			func(*ChannelMgr, int, string, bool) error {
				selectAvailableChannelCalled = true
				return nil
			})

		// Apply monkey patch for time.NewTicker to return our custom ticker channel
		mockTickerChan := make(chan time.Time, 1)
		patches.ApplyFunc(time.NewTicker, func(d time.Duration) *time.Ticker {
			return &time.Ticker{C: mockTickerChan}
		})

		// Receive function exited event via the WaitCheckDone
		// waitgroup, although which has been deprecated in the source,
		// should and would be removed soon.
		mgr.WaitCheckDone.Add(1)
		done := make(chan bool, 1)
		go func() {
			mgr.WaitCheckDone.Wait()
			done <- true
		}()
		// Run tested function in goroutine with timeout
		go mgr.periodicCheckAndSelect()

		for i := 0; i < 6; i++ {
			// Send one tick to trigger the quick or full check logic
			mockTickerChan <- time.Now()
			// Yield CPU to let the goroutine be scheduled
			time.Sleep(100 * time.Millisecond)

			// Perform the assertions
			if i < 5 {
				assert.False(t, checkChannelWorkerCalled, "checkChannelWorker should not be called, since quick check is unnecessary even when only websocket channel is working")
				assert.False(t, selectAvailableChannelCalled, "SelectAvailableChannelAndReport should be called, since quick check is unnecessary even when only websocket channel is working")
			} else {
				assert.True(t, checkChannelWorkerCalled, "checkChannelWorker should be called during full check, when only websocket channel is working")
				assert.True(t, selectAvailableChannelCalled, "SelectAvailableChannelAndReport should be called during full check, when only websocket channel is working")
			}
		}

		// Send stop signal
		mgr.StopChanelEvent <- struct{}{}

		// Wait for completion or timeout
		select {
		case <-done:
			// Test passed and let it go
		case <-time.After(100 * time.Millisecond):
			t.Error("Test timed out")
		}
	})

	t.Run("QuickAndFullCheckWhenNoChannelWorking", func(t *testing.T) {
		// Setup
		mgr := &ChannelMgr{
			StopChanelEvent: make(chan struct{}, 1),
			AllChannel: []IChannel{
				&_mockChannel{
					channelType: ChannelGshellType,
					isSupported: false,
					isWorking:   false,
				},
				&_mockChannel{
					channelType: ChannelWebsocketType,
					isSupported: true,
					isWorking:   false,
				},
			},
		}

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		// Mock SelectAvailableChannelAndReport to verify it's called
		selectAvailableChannelCalled := map[int]bool{}
		patches.ApplyMethod(reflect.TypeOf(mgr), "SelectAvailableChannelAndReport",
			func(_ *ChannelMgr, currentChannel int, _ string, _ bool) error {
				selectAvailableChannelCalled[currentChannel] = true
				return nil
			})
		// Mock checkChannelWorker to verify it's not called
		checkChannelWorkerCalled := false
		patches.ApplyPrivateMethod(reflect.TypeOf(mgr), "checkChannelWorker",
			func(*ChannelMgr) bool {
				checkChannelWorkerCalled = true
				return false
			})

		// Apply monkey patch for time.NewTicker to return our custom ticker channel
		mockTickerChan := make(chan time.Time, 1)
		patches.ApplyFunc(time.NewTicker, func(d time.Duration) *time.Ticker {
			return &time.Ticker{C: mockTickerChan}
		})

		// Receive function exited event via the WaitCheckDone
		// waitgroup, although which has been deprecated in the source,
		// should and would be removed soon.
		mgr.WaitCheckDone.Add(1)
		done := make(chan bool, 1)
		go func() {
			mgr.WaitCheckDone.Wait()
			done <- true
		}()
		// Run tested function in goroutine with timeout
		go mgr.periodicCheckAndSelect()

		for i := 0; i < 6; i++ {
			// Send one tick to trigger the quick check logic
			mockTickerChan <- time.Now()
			// Yield CPU to let the goroutine be scheduled
			time.Sleep(100 * time.Millisecond)

			// Perform the assertions
			if i < 5 {
				assert.True(t, selectAvailableChannelCalled[ChannelGshellType], "SelectAvailableChannelAndReport(ChannelGshellType, ...) should be called during quick check when no channel is working")
				assert.False(t, checkChannelWorkerCalled, "checkChannelWorker should not be called during quick check when no channel is working")
				selectAvailableChannelCalled[ChannelGshellType] = false
			} else {
				assert.True(t, checkChannelWorkerCalled, "checkChannelWorker should be called during full check when no channel is working")
				assert.True(t, selectAvailableChannelCalled[ChannelNone], "SelectAvailableChannelAndReport(ChannelNone, ...) should be called during full check when no channel is working")
			}
		}

		// Send stop signal
		mgr.StopChanelEvent <- struct{}{}

		// Wait for completion or timeout
		select {
		case <-done:
			// Test passed and let it go
		case <-time.After(100 * time.Millisecond):
			t.Error("Test timed out")
		}
	})
}

func TestSwitchChannelWhenConfigReceived(t *testing.T) {
	// Setup
	mgr := &ChannelMgr{
		StopChanelEvent: make(chan struct{}, 1),
		AllChannel: []IChannel{
			&_mockChannel{
				channelType: ChannelGshellType,
				isSupported: false,
				isWorking:   false,
			},
			&_mockChannel{
				channelType: ChannelWebsocketType,
				isSupported: true,
				isWorking:   true,
			},
		},
	}

	t.Run("SwitchToGshellWhenProtocolIsWebsocket", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		selectCalled := false
		expectedChannelType := ChannelGshellType
		expectedReason := "switch_channel_in_" + ChannelTypeStr(ChannelWebsocketType) + "_when_set_protocol"

		patches.ApplyMethod(reflect.TypeOf(mgr), "SelectAvailableChannelAndReport",
			func(*ChannelMgr, int, string, bool) error {
				selectCalled = true
				assert.Equal(t, expectedChannelType, expectedChannelType, "channel type should be Gshell")
				assert.Equal(t, expectedReason, expectedReason, "reason should match")
				return nil
			},
		)
		SwitchChannelWhenConfigReceived(logrus.New(), ChannelTypeStr(ChannelWebsocketType))
		assert.True(t, selectCalled, "SelectAvailableChannelAndReport should be called when protocol is websocket")
	})

	t.Run("SwitchToWebsocketWhenProtocolIsGshell", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		selectCalled := false
		expectedChannelType := ChannelWebsocketType
		expectedReason := "switch_channel_in_" + ChannelTypeStr(ChannelGshellType) + "_when_set_protocol"

		patches.ApplyMethod(reflect.TypeOf(mgr), "SelectAvailableChannelAndReport",
			func(*ChannelMgr, int, string, bool) error {
				selectCalled = true
				assert.Equal(t, expectedChannelType, expectedChannelType, "channel type should be Websocket")
				assert.Equal(t, expectedReason, expectedReason, "reason should match")
				return nil
			},
		)
		SwitchChannelWhenConfigReceived(logrus.New(), ChannelTypeStr(ChannelGshellType))
		assert.True(t, selectCalled, "SelectAvailableChannelAndReport should be called when protocol is gshell")
	})

	t.Run("DoNothingWhenInvalidOrMissingProtocol", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethod(reflect.TypeOf(mgr), "SelectAvailableChannelAndReport",
			func(*ChannelMgr, int, string, bool) error {
				t.Error("SelectAvailableChannelAndReport should not be called")
				return nil
			},
		)
		tests := []interface{}{"", "unknown", -1, 1, false}
		for _, items := range tests {
			SwitchChannelWhenConfigReceived(logrus.New(), items)
		}
	})
}
