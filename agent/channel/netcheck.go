package channel

import (
	"github.com/aliyun/aliyun_assist_client/agent/checknet"
)

// NetcheckReply represents
type NetcheckReply struct {
	Result int `json:"result"`
	Timestamp int64 `json:"timestamp"`
}

func LastNetcheckReply() *NetcheckReply {
	report := checknet.RecentReport()
	if report == nil {
		return nil
	}

	reply := NetcheckReply{
		Result: report.Result,
		Timestamp: report.FinishedTime.Local().Unix(),
	}
	return &reply
}
