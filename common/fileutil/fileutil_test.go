package fileutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

func TestCheckDirectoryIsExist(t *testing.T) {
	testCases := []struct {
		Name     string
		FilePath string
		Expect   bool
	}{
		{
			Name:     "not-exist",
			FilePath: "/not/exist/path",
			Expect:   false,
		},
		{
			Name:     "regular-file",
			FilePath: filepath.Join(os.TempDir(), "test-regular-file"),
			Expect:   false,
		},
		{
			Name:     "directory",
			FilePath: filepath.Join(os.TempDir(), "test-directory"),
			Expect:   true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			switch tt.Name {
			case "not-exist":
				assert.Equal(t, tt.Expect, CheckDirectoryIsExist(tt.FilePath))
			case "regular-file":
				f, err := os.Create(tt.FilePath)
				assert.Nil(t, err)
				defer os.Remove(tt.FilePath)
				f.Close()
				assert.True(t, CheckFileIsExist(tt.FilePath))
				assert.Equal(t, tt.Expect, CheckDirectoryIsExist(tt.FilePath))
			case "directory":
				err := os.MkdirAll(tt.FilePath, os.ModePerm)
				assert.Nil(t, err)
				defer os.RemoveAll(tt.FilePath)
				assert.Equal(t, tt.Expect, CheckDirectoryIsExist(tt.FilePath))
			}
		})
	}
}

// TODO-FIXME: For unknown reason (*os.File).Close cannot be monkey-patched via
// gomonkey.ApplyMethod or gomonkey.ApplyMethodFunc. Maybe some unexpected
// reflection issue?
func TestWriteAndSyncFile(t *testing.T) {
	errOpenFile := errors.New("os.OpenFile error")
	errFileWrite := errors.New("(*os.File).Write error")
	errFileSync := errors.New("(*os.File).Sync error")

	const tempFilePath = "test_write_sync_file.tmp"
	defer os.Remove(tempFilePath)

	t.Run("OpenFileFailed", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(os.OpenFile, func(name string, flag int, perm os.FileMode) (*os.File, error) {
			return nil, errOpenFile
		})
		defer patches.Reset()

		err := WriteAndSyncFile(tempFilePath, []byte("OpenFileFailed"), 0o644)
		assert.Error(t, err)
		assert.Equal(t, errOpenFile, err)
	})

	t.Run("FileWriteFailed", func(t *testing.T) {
		patches := gomonkey.ApplyMethod((*os.File)(nil), "Write", func(f *os.File, p []byte) (n int, err error) {
			return 0, errFileWrite
		})
		defer patches.Reset()

		err := WriteAndSyncFile(tempFilePath, []byte("FileWriteFailed"), 0o644)
		assert.Error(t, err)
		assert.Equal(t, errFileWrite, err)
	})

	t.Run("FileSyncFailed", func(t *testing.T) {
		patches := gomonkey.ApplyMethod((*os.File)(nil), "Sync", func(f *os.File) error {
			return errFileSync
		})
		defer patches.Reset()

		err := WriteAndSyncFile(tempFilePath, []byte("FileSyncFailed"), 0o644)
		assert.Error(t, err)
		assert.Equal(t, errFileSync, err)
	})

	t.Run("FileWriteAndSyncFailed", func(t *testing.T) {
		patches := gomonkey.ApplyMethod((*os.File)(nil), "Write", func(f *os.File, p []byte) (n int, err error) {
			return 0, errFileWrite
		})
		patches.ApplyMethod((*os.File)(nil), "Sync", func(f *os.File) error {
			return errFileSync
		})
		defer patches.Reset()

		err := WriteAndSyncFile(tempFilePath, []byte("FileWriteAndSyncFailed"), 0o644)
		assert.Error(t, err)
		assert.Equal(t, errFileWrite, err)
	})

	t.Run("FileWriteAndSyncFailed", func(t *testing.T) {
		patches := gomonkey.ApplyMethod((*os.File)(nil), "Write", func(f *os.File, p []byte) (n int, err error) {
			return 0, errFileWrite
		})
		patches.ApplyMethod((*os.File)(nil), "Sync", func(f *os.File) error {
			return errFileSync
		})
		defer patches.Reset()

		err := WriteAndSyncFile(tempFilePath, []byte("FileWriteAndSyncFailed"), 0o644)
		assert.Error(t, err)
		assert.Equal(t, errFileWrite, err)
	})

	t.Run("NormalCase", func(t *testing.T) {
		expectedData := []byte("NormalCase")
		var actualData []byte

		// Mock Write
		patches := gomonkey.ApplyMethod((*os.File)(nil), "Write", func(f *os.File, p []byte) (n int, err error) {
			actualData = append(actualData, p...)
			return len(p), nil
		})

		// Mock Sync
		patches.ApplyMethod((*os.File)(nil), "Sync", func(f *os.File) error {
			return nil
		})

		defer patches.Reset()

		err := WriteAndSyncFile(tempFilePath, expectedData, 0644)
		assert.NoError(t, err)
		assert.Equal(t, expectedData, actualData)
	})
}
