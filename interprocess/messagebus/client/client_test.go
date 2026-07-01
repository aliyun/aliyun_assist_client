package client

import (
	"errors"
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
	"context"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"google.golang.org/grpc"
	"github.com/aliyun/aliyun_assist_client/interprocess/messagebus/buses"
)

func TestConnectWithTimeout(t *testing.T) {
	var (
		mockDialNeedTime time.Duration = time.Millisecond * time.Duration(500)
		mockDialSuccess []bool
		mockDialErr error = errors.New("timeout")
		mockIdx int
	)
	defer gomonkey.ApplyFunc(grpc.DialContext, func (context.Context, string, ...grpc.DialOption) (*grpc.ClientConn, error){
		defer func() {
			mockIdx += 1
		}()
		
		if mockDialSuccess[mockIdx] {
			return nil, nil
		}
		time.Sleep(mockDialNeedTime)
		return nil, mockDialErr
	}).Reset()
	
	testCases := []struct{
		Name string
		DialNeedTime time.Duration
		DialSuccess []bool
		ExpectedTimeout time.Duration
		ExpectedErr error
	}{
		{
			Name: "success",
			DialSuccess: []bool{true, true},
			ExpectedTimeout: time.Millisecond * time.Duration(100),
			ExpectedErr: nil,
		},
		{
			Name: "fail-success",
			DialNeedTime: time.Millisecond * time.Duration(500),
			DialSuccess: []bool{false, true},
			ExpectedTimeout: mockDialNeedTime + time.Millisecond * time.Duration(100),
			ExpectedErr: nil,
		},
		{
			Name: "fail-fail",
			DialNeedTime: time.Millisecond * time.Duration(500),
			DialSuccess: []bool{false, false},
			ExpectedTimeout: mockDialNeedTime * 2 + time.Millisecond * time.Duration(100),
			ExpectedErr: mockDialErr,
		},
	}
	
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T){
			mockDialSuccess = tt.DialSuccess
			mockIdx = 0
			now := time.Now()
			_, err := ConnectWithTimeout(logrus.New(), buses.NewEndpoint(buses.UnixDomainSocketProtocol, "/mock/path"), time.Second)
			elapsed := time.Since(now)
			
			assert.ErrorIs(t, tt.ExpectedErr, err)
			assert.True(t, tt.ExpectedTimeout > elapsed)
		})
	}
}