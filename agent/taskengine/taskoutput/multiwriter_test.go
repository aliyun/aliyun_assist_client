package taskoutput

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/aliyun/aliyun_assist_client/common/langutil"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"golang.org/x/text/encoding/simplifiedchinese"
)

const (
	multiLanguageContent = `Brazil: Este é um teste em português do Brasil. China: 这是一段中文测试。Germany: Dies ist ein Test auf Deutsch. Japan: これは日本語のテストです。Russia: Это тест на русском языке. United States: This is an English test.
Brazil: Este é um teste em português do Brasil. China: 这是一段中文测试。Germany: Dies ist ein Test auf Deutsch. Japan: これは日本語のテストです。Russia: Это тест на русском языке. United States: This is an English test.
Brazil: Este é um teste em português do Brasil. China: 这是一段中文测试。Germany: Dies ist ein Test auf Deutsch. Japan: これは日本語のテストです。Russia: Это тест на русском языке. United States: This is an English test.
Brazil: Este é um teste em português do Brasil. China: 这是一段中文测试。Germany: Dies ist ein Test auf Deutsch. Japan: これは日本語のテストです。Russia: Это тест на русском языке. United States: This is an English test.
Brazil: Este é um teste em português do Brasil. China: 这是一段中文测试。Germany: Dies ist ein Test auf Deutsch. Japan: これは日本語のテストです。Russia: Это тест на русском языке. United States: This is an English test.`
)

func TestNewMultiWriter(t *testing.T) {
	defer gomonkey.ApplyFunc(pathutil.GetScriptPath, func() (string, error) {
		return os.TempDir(), nil
	}).Reset()

	testCases := []struct {
		Name        string
		SetPreSize  int
		SetPostSize int
		Size        int
	}{
		{
			Name:        "less_pre_size",
			SetPreSize:  100,
			SetPostSize: 100,
			Size:        50,
		},
		{
			Name:        "equal_pre_size",
			SetPreSize:  100,
			SetPostSize: 100,
			Size:        100,
		},
		{
			Name:        "less_post_size",
			SetPreSize:  100,
			SetPostSize: 100,
			Size:        150,
		},
		{
			Name:        "equal_post_size",
			SetPreSize:  100,
			SetPostSize: 100,
			Size:        200,
		},
		{
			Name:        "more_post_size",
			SetPreSize:  100,
			SetPostSize: 100,
			Size:        300,
		},
	}

	for _, tc := range testCases {
		// only legacy buffer
		t.Run(tc.Name, func(t *testing.T) {

			testF := func(t *testing.T, disableRingbuffer bool, writeToFile bool, encodingTrans bool) {
				var mw *MultiWriter
				var err error
				content := generateRandom(tc.Size)

				if encodingTrans {
					var encodedContent []byte
					encodedContent, err = langutil.Utf8ToGbk(content)
					assert.Nil(t, err)
					content = encodedContent
					mw = NewMultiWriter(logrus.New(), simplifiedchinese.GBK.NewDecoder())
				} else {
					mw = NewMultiWriter(logrus.New(), nil)
				}
				defer mw.Release()
				err = mw.InitOutputBuffer(tc.SetPreSize, tc.SetPostSize, disableRingbuffer)
				assert.Nil(t, err)

				filename := fmt.Sprintf("%s_%d", tc.Name, time.Now().UnixMilli())
				if writeToFile {
					err = mw.InitOutputFile(filename)
					assert.Nil(t, err)
				}

				half := len(content) / 2
				mw.Write(content[:half])
				mw.Write(content[half:])
				// wait buffer write
				time.Sleep(500 * time.Microsecond)

				pre := mw.BufferReadPre()
				all := mw.BufferReadAll()
				allFromStart := mw.BufferReadAllFromStart()
				switch {
				case tc.Size <= tc.SetPreSize:
					assert.Equal(t, content, pre)
					assert.Equal(t, 0, len(all))

					if !disableRingbuffer {
						assert.True(t, mw.IsBufComplete())
						assert.Equal(t, content, allFromStart)
					} else {
						assert.False(t, mw.IsBufComplete())
					}

					assert.Zero(t, mw.BufferDropped())
				case tc.Size <= tc.SetPostSize+tc.SetPreSize:
					assert.Equal(t, content[:len(pre)], pre)
					assert.Equal(t, content, append(pre, all...))

					if !disableRingbuffer {
						assert.True(t, mw.IsBufComplete())
						assert.Equal(t, content, allFromStart)
					} else {
						assert.False(t, mw.IsBufComplete())
					}

					assert.Zero(t, mw.BufferDropped())
				case tc.Size > tc.SetPostSize+tc.SetPreSize:
					assert.Equal(t, content[:len(pre)], pre)
					contentExpected := make([]byte, tc.SetPreSize+tc.SetPostSize)
					copy(contentExpected, content[:tc.SetPreSize])
					copy(contentExpected[tc.SetPreSize:], content[len(content)-tc.SetPostSize:])
					assert.Equal(t, contentExpected, append(pre, all...))

					if !disableRingbuffer {
						assert.False(t, mw.IsBufComplete())
						assert.NotEqual(t, content, allFromStart)
					} else {
						assert.False(t, mw.IsBufComplete())
					}

					assert.Equal(t, tc.Size-(tc.SetPreSize+tc.SetPostSize), mw.BufferDropped())
				}

				if writeToFile {
					filePath, createErr, writeErr := mw.FileState()
					assert.Nil(t, createErr)
					assert.Nil(t, writeErr)

					fileContent, err := os.ReadFile(filePath)
					assert.Nil(t, err)
					fmt.Println(string(filePath))
					assert.Equal(t, content, fileContent)

					mw.Release()
					assert.False(t, fileutil.CheckFileIsExist(filePath))
				}

			}
			// Just buffer
			fmt.Println("ringbuffer\tno_file\tno_transform")
			testF(t, false, false, false)
			fmt.Println("legacybuffer\tno_file\tno_transform")
			testF(t, true, false, false)

			// buffer + file
			fmt.Println("ringbuffer\twrite_file\tno_transform")
			testF(t, false, true, false)
			fmt.Println("legacybuffer\twrite_file\tno_transform")
			testF(t, true, true, false)

			// buffer + file, and transform encoding
			fmt.Println("ringbuffer\twrite_file\ttransform")
			testF(t, false, true, true)
			fmt.Println("legacybuffer\twrite_file\ttransform")
			testF(t, true, true, false)

		})
	}
}

func TestMultiWriterTransform(t *testing.T) {
	defer gomonkey.ApplyFunc(pathutil.GetScriptPath, func() (string, error) {
		return os.TempDir(), nil
	}).Reset()

	mw := NewMultiWriter(logrus.New(), simplifiedchinese.GBK.NewDecoder())
	defer mw.Release()
	
	err := mw.InitOutputBuffer(6000, 12000, false)
	assert.Nil(t, err)
	filename := fmt.Sprintf("%s_%d", "TestMultiWriterTransform", time.Now().UnixMilli())
	err = mw.InitOutputFile(filename)
	assert.Nil(t, err)

	encodedContent, err := langutil.Utf8ToGbk([]byte(multiLanguageContent))
	assert.Nil(t, err)
	n, err := mw.Write(encodedContent)
	assert.Nil(t, err)
	// n must be exactly equal to the length of the content to be written
	assert.Equal(t, len(encodedContent), n)
	// mw.writtenN must be exactly equal to the length of the content actually stored in the buffer
	assert.Equal(t, len(multiLanguageContent), mw.writtenN)

	filePath, createErr, writeErr := mw.FileState()
	assert.Nil(t, createErr)
	assert.Nil(t, writeErr)
	fileContent, err := os.ReadFile(filePath)
	assert.Nil(t, err)
	assert.Equal(t, []byte(multiLanguageContent), fileContent)

	time.Sleep(time.Second)
	precontent := mw.BufferReadPre()
	assert.Equal(t, []byte(multiLanguageContent), precontent)
	allcontent := mw.BufferReadAllFromStart()
	assert.Equal(t, []byte(multiLanguageContent), allcontent)
}

func BenchmarkTransformer(b *testing.B) {
	defer gomonkey.ApplyFunc(pathutil.GetScriptPath, func() (string, error) {
		return os.TempDir(), nil
	}).Reset()
	testCases := []struct {
		Name              string
		WriteFile         bool
		EncodingTransform bool
	}{
		{
			Name:              "no transformer(only buffer)",
			WriteFile:         false,
			EncodingTransform: false,
		},
		{
			Name:              "no transformer(buffer + file)",
			WriteFile:         true,
			EncodingTransform: false,
		},
		{
			Name:              "transformer(only buffer)",
			WriteFile:         false,
			EncodingTransform: true,
		},
		{
			Name:              "transformer(buffer + file)",
			WriteFile:         true,
			EncodingTransform: true,
		},
	}

	utf8Bytes := generateRandom(1024 * 1000)
	gbkBytes, err := langutil.Utf8ToGbk(utf8Bytes)
	assert.Nil(b, err)

	for _, tc := range testCases {
		var mw *MultiWriter
		var content []byte

		if tc.EncodingTransform {
			content = gbkBytes
			mw = NewMultiWriter(logrus.New(), simplifiedchinese.GBK.NewDecoder())
		} else {
			content = utf8Bytes
			mw = NewMultiWriter(logrus.New(), nil)
		}

		err := mw.InitOutputBuffer(6000, 6000, false)
		assert.Nil(b, err)

		if tc.WriteFile {
			filename := fmt.Sprintf("%s_%d", tc.Name, time.Now().UnixMilli())
			err = mw.InitOutputFile(filename)
			assert.Nil(b, err)
		}

		b.Run(tc.Name, func(b *testing.B) {
			// fmt.Println("b.Run: ", tc.Name)
			for start := 0; start < len(content); {
				end := start + 1024
				if end > len(content) {
					end = len(content)
				}
				mw.Write(content[start:end])
				start = end
			}
		})
		// fmt.Println(len(utf8Bytes))
		// fmt.Println(len(gbkBytes))
		// fmt.Println(len(content))
		// fmt.Println(mw.writtenN)
		mw.Release()
	}
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
