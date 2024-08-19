package docker

import (
	"context"
	"io"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/docker/docker/api/types"
	dockerstdcopy "github.com/docker/docker/pkg/stdcopy"
)

// TODO: FIXME: Support context, i.e., support canceling, for reading and writing

func streamStdoutFromHijacked(ctx context.Context, hijackedResponse *types.HijackedResponse,
	stdoutWriter io.Writer, stderrWriter io.Writer) error {
	if stdoutWriter == nil {
		stdoutWriter = io.Discard
	}
	if stderrWriter == nil {
		stderrWriter = io.Discard
	}

	_, err := dockerstdcopy.StdCopy(stdoutWriter, stderrWriter, hijackedResponse.Reader)
	return err
}

func streamHijacked(ctx context.Context, hijackedResponse *types.HijackedResponse,
	stdoutWriter io.Writer, stderrWriter io.Writer, stdinReader io.Reader) error {
	stdoutReceived := make(chan error)
	if stdoutWriter != nil || stderrWriter != nil {
		go func() {
			stdoutReceived <- streamStdoutFromHijacked(ctx, hijackedResponse, stdoutWriter, stderrWriter)
		}()
	}

	stdinSent := make(chan error)
	go func() {
		if stdinReader != nil {
			_, err := io.Copy(hijackedResponse.Conn, stdinReader)
			stdinSent <- err
		}
		hijackedResponse.CloseWrite()
		close(stdinSent)
	}()

	select {
	case err := <- stdoutReceived:
		return err
	case err := <- stdinSent:
		if err != nil {
			log.GetLogger().WithError(err).Warnln("Error encountered during sending stdin data, which is ignored")
		}
		if stdoutWriter != nil || stderrWriter != nil {
			return <- stdoutReceived
		}
	}
	return nil
}
