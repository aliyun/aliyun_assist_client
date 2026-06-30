package outputbuffer

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOutputBuffer(t *testing.T) {
	type testOutputBufferCase struct {
		preQuota    int
		postQuota   int
		contentLen  int
		dropped     int
		testReadAll bool
	}
	tests := []testOutputBufferCase{
		{
			preQuota:    0,
			postQuota:   0,
			contentLen:  300,
			dropped:     300,
		},
		{
			preQuota:    600,
			postQuota:   0,
			contentLen:  300,
			dropped:     0,
		},
		{
			preQuota:    600,
			postQuota:   0,
			contentLen:  600,
			dropped:     0,
		},
		{
			preQuota:    600,
			postQuota:   0,
			contentLen:  1200,
			dropped:     600,
		},
		{
			preQuota:    0,
			postQuota:   600,
			contentLen:  300,
			dropped:     0,
		},
		{
			preQuota:    0,
			postQuota:   600,
			contentLen:  600,
			dropped:     0,
		},
		{
			preQuota:    0,
			postQuota:   600,
			contentLen:  1200,
			dropped:     600,
		},
		{
			preQuota:    600,
			postQuota:   1200,
			contentLen:  300,
			dropped:     0,
		},
		{
			preQuota:    600,
			postQuota:   1200,
			contentLen:  600,
			dropped:     0,
		},
		{
			preQuota:    600,
			postQuota:   1200,
			contentLen:  800,
			dropped:     0,
		},
		{
			preQuota:    600,
			postQuota:   1200,
			contentLen:  1800,
			dropped:     0,
		},
		{
			preQuota:    600,
			postQuota:   1200,
			contentLen:  3000,
			dropped:     1200,
		},
	}
	dotest := func(outputBuffer OutputBuf, test testOutputBufferCase) {
		var res []byte
		// ReadXxx return nil before Init
		res = outputBuffer.ReadPre()
		assert.Equal(t, 0, len(res))
		res = outputBuffer.ReadAll()
		assert.Equal(t, 0, len(res))
		assert.Equal(t, 0, outputBuffer.Dropped())
		w, err := outputBuffer.Init(test.preQuota, test.postQuota)
		assert.Equal(t, nil, err)
		content := generateRandom(test.contentLen)
		n, err := w.Write(content)
		assert.Equal(t, test.contentLen, n)
		assert.Equal(t, nil, err)
		time.Sleep(time.Millisecond * 100) // wait scan
		if test.testReadAll {
			allOutput := outputBuffer.ReadAll()
			assert.Equal(t, 0, len(outputBuffer.ReadAll()))
			if test.contentLen <= test.preQuota {
				assert.Equal(t, string(content), string(allOutput))
			} else {
				assert.Equal(t, string(content[:test.preQuota]), string(allOutput[:test.preQuota]))
				if test.contentLen <= test.preQuota+test.postQuota {
					assert.Equal(t, string(content[test.preQuota:]), string(allOutput[test.preQuota:]))
				} else {
					assert.Equal(t, string(content[len(content)-test.postQuota:]), string(allOutput[test.preQuota:]))
				}
			}
		} else {
			preOutput := outputBuffer.ReadPre()
			assert.Equal(t, 0, len(outputBuffer.ReadPre()))
			if test.contentLen <= test.preQuota {
				assert.Equal(t, string(content), string(preOutput))
			} else {
				if string(content[:test.preQuota]) != string(preOutput) {
					fmt.Printf("test: %+v\n", test)
					fmt.Println("content: ", string(content))
				}
				assert.Equal(t, string(content[:test.preQuota]), string(preOutput))
				outputBuffer.ReadPre()
				postOutput := outputBuffer.ReadAll()
				assert.Equal(t, 0, len(outputBuffer.ReadAll()))
				if test.contentLen <= test.preQuota+test.postQuota {
					if string(content[test.preQuota:]) != string(postOutput) {
						fmt.Println("test: ", test)
						fmt.Println("content: ", string(content))
					}
					assert.Equal(t, string(content[test.preQuota:]), string(postOutput))
				} else {
					assert.Equal(t, string(content[len(content)-test.postQuota:]), string(postOutput))
				}
			}
		}
		assert.Equal(t, test.dropped, outputBuffer.Dropped())
		err = outputBuffer.Uninit()
		assert.Nil(t, err)
		// ReadXxx return nil before Init
		res = outputBuffer.ReadPre()
		assert.Equal(t, 0, len(res))
		res = outputBuffer.ReadAll()
		assert.Equal(t, 0, len(res))
	}
	outputBuffer := &OutputBuffer{}
	legacyOutputBuffer := &LegacyOutputBuffer{}
	for _, test := range tests {
		dotest(legacyOutputBuffer, test)
		dotest(outputBuffer, test)
		test.testReadAll = true
		dotest(legacyOutputBuffer, test)
		dotest(outputBuffer, test)
	}
}
