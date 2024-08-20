package zipfile

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"path/filepath"
)

func PeekFile(zipFile string, target string) ([]byte, error) {
	return PeekFileContext(context.Background(), zipFile, target)
}

func PeekFileContext(ctx context.Context, zipFile string, target string) ([]byte, error) {
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return nil, err
	}
	defer zipReader.Close()

	cleanedTarget := filepath.Clean(target)
	for _, f := range zipReader.File {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if f.FileInfo().IsDir() {
				continue
			}
			if filepath.Clean(f.Name) != cleanedTarget {
				continue
			}

			return func() ([]byte, error) {
				compressed, err := f.Open()
				if err != nil {
					return nil, err
				}
				defer compressed.Close()

				return io.ReadAll(compressed)
			}()
		}
	}

	return nil, os.ErrNotExist
}

// Unzip zipFile to destDir, all decompressed files will be synchronized to make sure 
// being writen into disk if syncFile is true.
func Unzip(zipFile string, destDir string, syncFile bool) error {
	return UnzipContext(context.Background(), zipFile, destDir, syncFile)
}

// Unzip zipFile to destDir, all decompressed files will be synchronized to make sure 
// being writen into disk if syncFile is true.
func UnzipContext(ctx context.Context, zipFile string, destDir string, syncFile bool) error {
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	for _, f := range zipReader.File {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			fpath := filepath.Join(destDir, f.Name)
			if f.FileInfo().IsDir() {
				os.MkdirAll(fpath, os.ModePerm)
			} else {
				if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
					return err
				}

				err := func() error {
					inFile, err := f.Open()
					if err != nil {
						return err
					}
					defer inFile.Close()

					outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
					if err != nil {
						return err
					}
					defer outFile.Close()
					if syncFile {
						defer outFile.Sync()
					}

					_, err = io.Copy(outFile, inFile)
					return err
				}()
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
