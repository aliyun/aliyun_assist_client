package port

import (
	"context"
	"errors"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/session/channel"
	"github.com/aliyun/aliyun_assist_client/agent/session/sessionresult"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type _mockAddr struct{}

func (m *_mockAddr) Network() string { return "tcp" }
func (m *_mockAddr) String() string  { return "1.1.1.1" }

func TestPortPluginWritePump(t *testing.T) {
	testCases := []struct {
		Name          string
		ReadErr       error
		SendErr       error
		ExpectErrCode string
	}{
		{
			Name:          "ok",
			ReadErr:       nil,
			SendErr:       nil,
			ExpectErrCode: "Ok",
		},
		{
			Name:          "conn read err",
			ReadErr:       errors.New("some err"),
			SendErr:       nil,
			ExpectErrCode: "ReadFromTargetPortFailed",
		},
		{
			Name:          "send err",
			ReadErr:       nil,
			SendErr:       errors.New("some err"),
			ExpectErrCode: "SendingDataFailed",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			var conn *net.TCPConn
			defer gomonkey.ApplyMethod(reflect.TypeOf(conn), "Read", func(conn *net.TCPConn, buf []byte) (int, error) {
				n := copy(buf, []byte("hello"))
				return n, tc.ReadErr
			}).Reset()
			defer gomonkey.ApplyMethod(reflect.TypeOf(conn), "RemoteAddr", func(conn *net.TCPConn) net.Addr {
				return &_mockAddr{}
			}).Reset()
			var c *channel.SessionChannel
			defer gomonkey.ApplyMethod(reflect.TypeOf(c), "SendStreamDataMessage", func(*channel.SessionChannel, []byte) error {
				return tc.SendErr
			}).Reset()

			p := PortPlugin{
				logger:      logrus.New(),
				conn:        &net.TCPConn{},
				dataChannel: &channel.SessionChannel{},
			}
			p.exitCtx, p.exitFunc = context.WithCancelCause(context.Background())
			go p.writePump()
			if tc.Name == "ok" {
				p.exitFunc(sessionresult.NewOk("ok"))
			}
			time.Sleep(time.Second)
			contextErr := context.Cause(p.exitCtx)
			res, ok := contextErr.(*sessionresult.SessionResult)
			assert.True(t, ok)
			assert.Equal(t, tc.ExpectErrCode, res.Code)
		})
	}
}
