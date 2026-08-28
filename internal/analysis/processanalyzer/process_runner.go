package processanalyzer

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/processprotocol"
)

type frameEvent struct {
	frame processprotocol.Frame
	err   error
}

type processRunner struct {
	command       *exec.Cmd
	stdin         io.WriteCloser
	stdout        io.ReadCloser
	stderr        io.ReadCloser
	stderrBuffer  *boundedBuffer
	stderrDone    chan struct{}
	events        chan frameEvent
	stopReader    chan struct{}
	maxFrameBytes int
	stopOnce      sync.Once
	stdinOnce     sync.Once
	killOnce      sync.Once
	waitOnce      sync.Once
	wait          chan error
}

func startRunner(analyzer *Analyzer, operation processprotocol.FrameType) (*processRunner, error) {
	command := exec.Command(analyzer.command, analyzer.args...)
	command.Dir = analyzer.workingDirectory
	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, analysis.WrapHostError(analysis.ErrAnalyzerFailed, "external analyzer stdin could not be opened", err, map[string]any{"operation": operation})
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, analysis.WrapHostError(analysis.ErrAnalyzerFailed, "external analyzer stdout could not be opened", err, map[string]any{"operation": operation})
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, analysis.WrapHostError(analysis.ErrAnalyzerFailed, "external analyzer stderr could not be opened", err, map[string]any{"operation": operation})
	}
	if err := command.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		_ = stderr.Close()
		return nil, analysis.WrapHostError(analysis.ErrAnalyzerFailed, "external analyzer process could not be started", err, map[string]any{"operation": operation, "command": append([]string{analyzer.command}, analyzer.args...)})
	}

	runner := &processRunner{
		command:       command,
		stdin:         stdin,
		stdout:        stdout,
		stderr:        stderr,
		stderrBuffer:  &boundedBuffer{limit: analyzer.config.MaxStderrBytes},
		stderrDone:    make(chan struct{}),
		events:        make(chan frameEvent, 8),
		stopReader:    make(chan struct{}),
		maxFrameBytes: analyzer.config.MaxFrameBytes,
		wait:          make(chan error, 1),
	}
	go runner.readFrames(analyzer.config.MaxFrameBytes)
	go func() {
		_, _ = io.Copy(runner.stderrBuffer, runner.stderr)
		close(runner.stderrDone)
	}()
	return runner, nil
}

func (r *processRunner) readFrames(maxFrameBytes int) {
	defer close(r.events)
	decoder := processprotocol.NewDecoderWithLimit(r.stdout, maxFrameBytes)
	for {
		frame, err := decoder.ReadFrame()
		event := frameEvent{frame: frame, err: err}
		select {
		case r.events <- event:
		case <-r.stopReader:
			return
		}
		if err != nil {
			return
		}
	}
}

func (r *processRunner) next(ctx context.Context) (frameEvent, bool) {
	select {
	case event, ok := <-r.events:
		return event, ok
	case <-ctx.Done():
		return frameEvent{}, false
	}
}

func (r *processRunner) write(frame processprotocol.Frame) error {
	return processprotocol.WriteFrameWithLimit(r.stdin, frame, r.maxFrameBytes)
}

func (r *processRunner) writeContext(ctx context.Context, frame processprotocol.Frame) error {
	if r == nil {
		return errors.New("external analyzer process runner is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	result := make(chan error, 1)
	go func() { result <- r.write(frame) }()
	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *processRunner) closeStdin() {
	r.stdinOnce.Do(func() { _ = r.stdin.Close() })
}

func (r *processRunner) cancelAndTerminate(requestID string, cause error) error {
	if r == nil {
		return nil
	}
	reason := "cancelled"
	if cause != nil {
		reason = cause.Error()
	}
	cancelFrame := processprotocol.Frame{Type: processprotocol.FrameCancel, RequestID: requestID, Reason: reason}
	cancelContext, cancel := context.WithTimeout(context.Background(), cancelWriteTimeout)
	writeErr := r.writeContext(cancelContext, cancelFrame)
	cancel()
	if writeErr != nil {
		r.closeStdin()
	}
	return r.terminate()
}

func (r *processRunner) finish() error {
	if r == nil {
		return nil
	}
	r.closeStdin()
	waitErr := r.awaitWait()
	if errors.Is(waitErr, errProcessCleanupTimeout) {
		// A process that ignores EOF must not survive a successful protocol
		// exchange. Close its pipes and kill it before waiting again.
		r.stopReaderOnce()
		r.closePipes()
		r.kill()
		waitErr = r.awaitWait()
	}
	r.stopReaderOnce()
	r.closePipes()
	r.awaitStderr()
	return waitErr
}

func (r *processRunner) terminate() error {
	if r == nil {
		return nil
	}
	r.stopReaderOnce()
	r.closeStdin()
	r.closePipes()
	r.kill()
	waitErr := r.awaitWait()
	r.awaitStderr()
	return waitErr
}

func (r *processRunner) closePipes() {
	if r == nil {
		return
	}
	_ = r.stdout.Close()
	_ = r.stderr.Close()
}

func (r *processRunner) kill() {
	if r == nil {
		return
	}
	r.killOnce.Do(func() {
		if r.command.Process != nil {
			_ = r.command.Process.Kill()
		}
	})
}

func (r *processRunner) stopReaderOnce() {
	r.stopOnce.Do(func() { close(r.stopReader) })
}

func (r *processRunner) awaitWait() error {
	timer := time.NewTimer(cleanupWaitTimeout)
	defer timer.Stop()
	r.waitOnce.Do(func() {
		go func() { r.wait <- r.command.Wait() }()
	})
	select {
	case err := <-r.wait:
		return err
	case <-timer.C:
		return errProcessCleanupTimeout
	}
}

func (r *processRunner) awaitStderr() {
	timer := time.NewTimer(cleanupWaitTimeout)
	defer timer.Stop()
	select {
	case <-r.stderrDone:
	case <-timer.C:
	}
}

func (r *processRunner) stderrText() string {
	if r == nil || r.stderrBuffer == nil {
		return ""
	}
	return r.stderrBuffer.String()
}

type boundedBuffer struct {
	mu        sync.Mutex
	limit     int
	data      []byte
	truncated bool
}

func (b *boundedBuffer) Write(value []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := b.limit - len(b.data)
	if remaining > 0 {
		keep := value
		if len(keep) > remaining {
			keep = keep[:remaining]
			b.truncated = true
		}
		b.data = append(b.data, keep...)
	} else if len(value) > 0 {
		b.truncated = true
	}
	return len(value), nil
}

func (b *boundedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	value := string(b.data)
	if b.truncated {
		value += "\n[stderr truncated]"
	}
	return value
}

var errProcessCleanupTimeout = errors.New("external analyzer process did not terminate after cleanup")
