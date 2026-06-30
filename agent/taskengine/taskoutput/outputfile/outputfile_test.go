package outputfile

import (
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/stretchr/testify/assert"
)

func TestOutputFileWriter(t *testing.T) {
	outputFilePath := filepath.Join(os.TempDir(), "output_test_file")
	defer os.Remove(outputFilePath)
	wr, err := NewOutputFileWriter(outputFilePath)
	assert.Nil(t, err)

	count := 0
	var expectedContent []byte
	for i := 1; i < 50; i += 1 {
		content := generateRandom(i)
		expectedContent = append(expectedContent, content...)
		n, err := wr.Write(content)
		assert.Nil(t, err)
		count += n
	}

	filePath, n, err := wr.State()
	assert.Nil(t, err)
	assert.Equal(t, int64(count), n)

	realContent, err := os.ReadFile(filePath)
	assert.Nil(t, err)
	assert.Equal(t, expectedContent, realContent)

	assert.Nil(t, nil, wr.Remove())
	assert.False(t, fileutil.CheckFileIsExist(filePath))
}

func generateRandom(length int) []byte {
	if length == 0 {
		return []byte{}
	}
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return b
}
