package outputfile

import (
	"os"
)

type OutputFileWriter struct {
	fw   *os.File
	path string
	e    error
	n    int64
}

func NewOutputFileWriter(filePath string) (*OutputFileWriter, error) {
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &OutputFileWriter{
		fw:   f,
		path: filePath,
	}, nil
}

func (pw *OutputFileWriter) Write(p []byte) (n int, err error) {
	n, err = pw.fw.Write(p)
	pw.n += int64(n)
	if err != nil && pw.e == nil {
		pw.e = err
	}
	return
}

func (pw *OutputFileWriter) Remove() error {
	if pw.fw != nil {
		pw.fw.Close()
	}
	return os.Remove(pw.path)
}

func (pw *OutputFileWriter) State() (string, int64, error) {
	return pw.path, pw.n, pw.e
}

