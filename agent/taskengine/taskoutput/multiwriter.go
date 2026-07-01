package taskoutput

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskoutput/outputbuffer"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskoutput/outputfile"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"golang.org/x/text/transform"
)

type MultiWriter struct {
	outputBuffer      outputbuffer.OutputBuf
	outputBufferW     io.Writer
	disableRingBuffer bool
	preQuota          int
	postQuota         int

	outputFile    *outputfile.OutputFileWriter
	createFileErr error

	transformer transform.Transformer
	// writtenN is the length of the content actually written into the buffer
	// mw.writtenN is used to determine whether the content of the buffer is complete
	writtenN int
	logger   logrus.FieldLogger
}

func NewMultiWriter(logger logrus.FieldLogger, transformer transform.Transformer) *MultiWriter {
	logger = logger.WithField("transformer", true)
	return &MultiWriter{
		transformer: transformer,
		logger:      logger,
	}
}

func (m *MultiWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	if t := m.transformer; t != nil {
		p, _, err = transform.Bytes(t, p)
		if err != nil {
			m.logger.WithError(err).Error("Transform encoding failed")
		}
	}
	m.writtenN += len(p)

	if m.outputBuffer == nil && m.outputFile == nil {
		return 0, fmt.Errorf("No writers present.")
	}
	if m.outputBufferW != nil {
		if _, err := m.outputBufferW.Write(p); err != nil {
			m.logger.WithError(err).Error("write buffer failed")
		}
	}
	if m.outputFile != nil {
		if _, err := m.outputFile.Write(p); err != nil {
			m.logger.WithError(err).Error("write file failed")
		}
	}
	return n, nil
}

func (m *MultiWriter) InitOutputBuffer(preQuota, postQuota int, disableRingBuffer bool) error {
	m.logger.WithFields(logrus.Fields{
		"preQuota":          preQuota,
		"postQuota":         postQuota,
		"disableRingBuffer": disableRingBuffer,
	}).Info("init output buffer")
	var buf outputbuffer.OutputBuf
	if disableRingBuffer {
		buf = &outputbuffer.LegacyOutputBuffer{}
	} else {
		buf = &outputbuffer.OutputBuffer{}
	}

	if w, err := buf.Init(preQuota, postQuota); err != nil {
		return err
	} else {
		m.outputBuffer = buf
		m.outputBufferW = w
		m.disableRingBuffer = disableRingBuffer
		m.preQuota = preQuota
		m.postQuota = postQuota
		return nil
	}
}

func (m *MultiWriter) InitOutputFile(fileName string) error {
	scriptDir, err := pathutil.GetScriptPath()
	if err != nil {
		m.createFileErr = fmt.Errorf("get script directory faield, %w", err)
		return err
	}
	filePath := filepath.Join(scriptDir, fileName)
	m.logger.WithFields(logrus.Fields{
		"filePath": filePath,
	}).Info("init output file")
	f, err := outputfile.NewOutputFileWriter(filePath)
	if err != nil {
		m.createFileErr = err
		return err
	}
	m.outputFile = f
	return nil
}

func (m *MultiWriter) BufferReadPre() []byte {
	if b := m.outputBuffer; b != nil {
		return b.ReadPre()
	}
	return nil
}

func (m *MultiWriter) BufferReadAll() []byte {
	if b := m.outputBuffer; b != nil {
		return b.ReadAll()
	}
	return nil
}

func (m *MultiWriter) BufferDropped() int {
	if b := m.outputBuffer; b != nil {
		return b.Dropped()
	}
	return 0
}

func (m *MultiWriter) BufferReadAllFromStart() []byte {
	if b := m.outputBuffer; b != nil && !m.disableRingBuffer {
		return b.ReadAllFromStart()
	}
	return nil
}

func (m *MultiWriter) FileState() (filePath string, createErr error, writeErr error) {
	if m.outputFile != nil {
		filePath, _, writeErr = m.outputFile.State()
	}
	createErr = m.createFileErr
	return
}

func (m *MultiWriter) IsBufComplete() bool {
	if m.outputBuffer == nil {
		return false
	}
	if !m.disableRingBuffer {
		return (m.preQuota + m.postQuota) >= m.writtenN
	}
	return false
}

func (m *MultiWriter) Release() {
	if b := m.outputBuffer; b != nil {
		m.outputBuffer = nil
		m.outputBufferW = nil
		m.disableRingBuffer = false
		m.preQuota = 0
		m.postQuota = 0
		b.Uninit()
	}
	if f := m.outputFile; f != nil {
		m.outputFile = nil
		f.Remove()
	}
	m.createFileErr = nil
	m.writtenN = 0
}
