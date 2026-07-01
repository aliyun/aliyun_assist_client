package log

import (
	"fmt"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestGetLogger(t *testing.T) {
	InitLog("test", "", false)
	assert.NotNil(t, GetLogger())
}

func TestUpdateLogFileCountLimit(t *testing.T) {
	t.Run("ShouldUpdateRotationCountWhenInt", func(t *testing.T) {
		InitLogWithRotationParams("test", "", false)
		assert.Equal(t, _rotateLog.GetRotationCount(), uint(365))
		UpdateLogFileCountLimit(GetLogger(), int64(5))
		assert.Equal(t, _rotateLog.GetRotationCount(), uint(5))

		UpdateLogFileCountLimit(GetLogger(), int64(-5))
		assert.Equal(t, _rotateLog.GetRotationCount(), uint(5))
	})
}

func TestUpdateLogSizeLimit(t *testing.T) {
	t.Run("ShouldUpdateLogSizeLimitWhenInt", func(t *testing.T) {
		InitLogWithRotationParams("test", "", false)
		assert.Equal(t, _rotateLog.GetRotationSize(), int64(1024*1024*1024))
		UpdateLogSizeLimit(GetLogger(), int64(1024*1024))
		assert.Equal(t, _rotateLog.GetRotationSize(), int64(1024*1024))

		UpdateLogSizeLimit(GetLogger(), int64(-5))
		assert.Equal(t, _rotateLog.GetRotationSize(), int64(1024*1024))
	})
}

// MemoryHook captures log entries for testing
type MemoryHook struct {
	Entries []*logrus.Entry
}

func NewMemoryHook() *MemoryHook {
	return &MemoryHook{Entries: make([]*logrus.Entry, 0)}
}

func (h *MemoryHook) Fire(entry *logrus.Entry) error {
	h.Entries = append(h.Entries, entry)
	return nil
}

func (h *MemoryHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *MemoryHook) Reset() {
	h.Entries = h.Entries[:0]
}

func (h *MemoryHook) HasErrorLevel() bool {
	for _, e := range h.Entries {
		if e.Level == logrus.ErrorLevel {
			return true
		}
	}
	return false
}

func (h *MemoryHook) GetErrorMessages() []string {
	var msgs []string
	for _, e := range h.Entries {
		if e.Level == logrus.ErrorLevel {
			msgs = append(msgs, e.Message)
		}
	}
	return msgs
}

func TestUpdateRotationParams_NoError(t *testing.T) {
	tmpDir := "test_log"
	InitLogWithRotationParams("test.log", "test_log", false)
	defer os.RemoveAll(tmpDir)

	// 创建带 hook 的 logger
	logger := logrus.New()
	logHook := NewMemoryHook()
	logger.AddHook(logHook)
	logger.SetOutput(io.Discard) // 不输出到控制台

	var wg sync.WaitGroup
	var mu sync.Mutex
	var testErr error

	const rounds = 100

	for i := 0; i < rounds; i++ {
		countVal := int64(i%50 + 1)
		sizeVal := int64((i%3 + 1) * 10 * 1024 * 1024)

		wg.Add(2)
		mu.Lock()
		logHook.Reset() // 清空上一轮日志
		mu.Unlock()

		go func(cv int64) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					testErr = fmt.Errorf("panic: %v", r)
					mu.Unlock()
				}
			}()
			UpdateLogFileCountLimit(logger, cv)
		}(countVal)

		go func(sv int64) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					testErr = fmt.Errorf("panic: %v", r)
					mu.Unlock()
				}
			}()
			UpdateLogSizeLimit(logger, sv)
		}(sizeVal)

		wg.Wait()

		// 检查是否有 error 日志
		mu.Lock()
		if testErr != nil {
			t.Fatalf("Unexpected panic: %v", testErr)
		}
		if logHook.HasErrorLevel() {
			t.Errorf("Expected no error logs, but got errors:")
			for _, msg := range logHook.GetErrorMessages() {
				t.Errorf("  - %s", msg)
			}
		}
		mu.Unlock()

		if _rotationCount != uint(countVal) {
			t.Errorf("Expected _rotationCount=%d, got %d", countVal, _rotationCount)
		}
		if _rotationSize != sizeVal {
			t.Errorf("Expected _rotationSize=%d, got %d", sizeVal, _rotationSize)
		}

		if _rotateLog == nil {
			t.Fatal("Expected _rotateLog to be reinitialized, but it's nil")
		}

		_, err := _rotateLog.Write([]byte("test log\n"))
		if err != nil {
			t.Errorf("Failed to write to rotated log: %v", err)
		}

		time.Sleep(10 * time.Millisecond)
	}

	// 最终清理
	if _rotateLog != nil {
		if err := _rotateLog.Close(); err != nil {
			t.Errorf("Final close failed: %v", err)
		}
	}
}
