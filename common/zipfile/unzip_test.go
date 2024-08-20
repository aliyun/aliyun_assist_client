package zipfile

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

func TestUnzip(t *testing.T) {
	// Make a zip file
	file_1 := filepath.Join(os.TempDir(), "file_1")
	content_1 := "content_1"
	file_2 := filepath.Join(os.TempDir(), "file_2")
	content_2 := "content_2"
	zipFile := filepath.Join(os.TempDir(), "testzip.zip")
	unzipDir := filepath.Join(os.TempDir(), "unzip_dir_test")
	os.Mkdir(unzipDir, os.ModePerm)
	defer os.RemoveAll(zipFile)
	defer os.RemoveAll(unzipDir)

	assert.Nil(t, os.WriteFile(file_1, []byte(content_1), os.ModePerm))
	assert.Nil(t, os.WriteFile(file_2, []byte(content_2), os.ModePerm))
	defer os.Remove(file_1)
	defer os.Remove(file_2)
	func() {
		newZipFile, err := os.Create(zipFile)
		assert.Nil(t, err)
		defer newZipFile.Close()

		zipWriter := zip.NewWriter(newZipFile)
		defer zipWriter.Close()
		assert.Nil(t, addToZip(zipWriter, file_1, "file_1"))
		assert.Nil(t, addToZip(zipWriter, file_2, "subdir/file_2"))
	}()

	// Mock sync
	var f *os.File
	var syncCount int
	guard_sync := gomonkey.ApplyMethod(reflect.TypeOf(f), "Sync", func() error {
		syncCount += 1
		return nil
	})
	defer guard_sync.Reset()

	tests := []struct {
		Name     string
		WantErr  error
		needSync bool
	}{
		{
			Name:     "OK",
			WantErr:  nil,
			needSync: false,
		},
		{
			Name:     "Timeout",
			WantErr:  context.DeadlineExceeded,
			needSync: false,
		},
		{
			Name:     "Sync",
			WantErr:  nil,
			needSync: true,
		},
	}
	for _, tt := range tests {
		os.RemoveAll(unzipDir)
		os.Mkdir(unzipDir, os.ModePerm)
		syncCount = 0
		ctx := context.Background()
		var cancel context.CancelFunc
		if tt.Name == "Timeout" {
			ctx, cancel = context.WithTimeout(ctx, time.Millisecond*time.Duration(10))
			time.Sleep(time.Millisecond * time.Duration(50))
		}
		err := UnzipContext(ctx, zipFile, unzipDir, tt.needSync)
		assert.Equal(t, tt.WantErr, err)
		if err == nil {
			// check file
			checkFileContent(t, filepath.Join(unzipDir, "file_1"), content_1)
			checkFileContent(t, filepath.Join(unzipDir, "subdir", "file_2"), content_2)
		}
		// check sync
		if tt.needSync {
			assert.Equal(t, 2, syncCount)
		} else {
			assert.Equal(t, 0, syncCount)
		}

		if cancel != nil {
			cancel()
		}
	}

}

// addToZip add file into zipfile
func addToZip(zipWriter *zip.Writer, filename string, filenameInZip string) error {
	fileToZip, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	zipFileWriter, err := zipWriter.Create(filenameInZip)
	if err != nil {
		return err
	}

	_, err = io.Copy(zipFileWriter, fileToZip)
	return err
}

func checkFileContent(t *testing.T, filename, expectContent string) {
	content, err := os.ReadFile(filename)
	assert.Nil(t, err)
	assert.Equal(t, expectContent, string(content))
}
