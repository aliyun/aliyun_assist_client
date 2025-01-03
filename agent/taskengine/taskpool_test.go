package taskengine

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/flagging"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

func TestDispatcher(t *testing.T) {
	var concurrency atomic.Int32
	taskFunc := func() {
		concurrency.Add(1)
		time.Sleep(100 * time.Millisecond)
		concurrency.Add(-1)
	}
	maxConcurrencyList := []int{10, 50, 100, 200, 500}
	taskCountList := []int{1, 20, 30, 100, 500, 1000, 2000}
	dispatcher := GetDispatcher()
	lastTimestamp := time.Now()
	for i := range taskCountList {
		taskCount := taskCountList[i]
		for k := range maxConcurrencyList {
			maxCon := maxConcurrencyList[k]
			thisTimestamp := time.Now()
			fmt.Printf("%d taskCount:%d maxConcurrency: %d\n", thisTimestamp.Sub(lastTimestamp).Milliseconds(), taskCount, maxCon)
			lastTimestamp = thisTimestamp
			flagging.UpdateConfigRuntime(logrus.New(), []string{flagging.TASK_CONCURRENCY_HARDLIMIT, strconv.Itoa(maxCon)})

			for j := 0; j < taskCount; j += 1 {
				dispatcher.PutTask(taskFunc)
			}
			// 等待任务运行起来
			time.Sleep(100 * time.Millisecond)

			for {
				cur1 := int(concurrency.Load())
				cur2 := dispatcher.Concurrency()
				assert.LessOrEqual(t, cur1, maxCon)
				assert.LessOrEqual(t, cur2, maxCon)
				if cur1 == 0 {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
	}
}
