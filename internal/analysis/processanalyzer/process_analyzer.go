// Package processanalyzer adapts an explicitly supplied external analyzer
// process to the language-neutral analysis.Analyzer contract.
//
// The adapter owns only process invocation and protocol translation. The
// common analysis.Host remains responsible for selection, option precedence,
// result normalization, and final result validation.
package processanalyzer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/processprotocol"
)

const (
	// DefaultMaxFrameBytes bounds one complete stdout protocol line.
	DefaultMaxFrameBytes = processprotocol.DefaultMaxFrameBytes
	// DefaultMaxStderrBytes bounds retained stderr context from one process.
	DefaultMaxStderrBytes = 1 * 1024 * 1024
	// DefaultHelloTimeout limits the initial plugin handshake.
	DefaultHelloTimeout = 5 * time.Second
	// DefaultOperationTimeout limits an invocation when its caller has no
	// earlier deadline.
	DefaultOperationTimeout = 60 * time.Second

	cancelWriteTimeout = 100 * time.Millisecond
	cleanupWaitTimeout = 5 * time.Second
)

// Config contains host-owned process limits. A zero or negative value uses
// the documented default so callers cannot accidentally remove a bound.
type Config struct {
	MaxFrameBytes    int
	MaxStderrBytes   int
	HelloTimeout     time.Duration
	OperationTimeout time.Duration
}

// DefaultConfig returns the v1 process limits.
func DefaultConfig() Config {
	return Config{
		MaxFrameBytes:    DefaultMaxFrameBytes,
		MaxStderrBytes:   DefaultMaxStderrBytes,
		HelloTimeout:     DefaultHelloTimeout,
		OperationTimeout: DefaultOperationTimeout,
	}
}

func (config Config) normalized() Config {
	defaults := DefaultConfig()
	if config.MaxFrameBytes <= 0 {
		config.MaxFrameBytes = defaults.MaxFrameBytes
	}
	if config.MaxStderrBytes <= 0 {
		config.MaxStderrBytes = defaults.MaxStderrBytes
	}
	if config.HelloTimeout <= 0 {
		config.HelloTimeout = defaults.HelloTimeout
	}
	if config.OperationTimeout <= 0 {
		config.OperationTimeout = defaults.OperationTimeout
	}
	return config
}

// Analyzer is a process-backed implementation of analysis.Analyzer.
type Analyzer struct {
	descriptor       processprotocol.Descriptor
	baseDirectory    string
	command          string
	args             []string
	workingDirectory string
	config           Config
}

var _ analysis.Analyzer = (*Analyzer)(nil)

var requestSequence uint64

// New validates a descriptor and resolves relative executable values from
// the current working directory. Use LoadDescriptor for a descriptor file so
// relative values resolve from that file's directory.
func New(descriptor processprotocol.Descriptor) (*Analyzer, error) {
	baseDirectory, err := os.Getwd()
	if err != nil {
		return nil, analysis.WrapHostError(analysis.ErrInvalidManifest, "external analyzer base directory could not be resolved", err, nil)
	}
	return NewWithBaseDirectory(descriptor, baseDirectory, DefaultConfig())
}

// NewWithConfig validates a descriptor with explicit process limits.
func NewWithConfig(descriptor processprotocol.Descriptor, config Config) (*Analyzer, error) {
	baseDirectory, err := os.Getwd()
	if err != nil {
		return nil, analysis.WrapHostError(analysis.ErrInvalidManifest, "external analyzer base directory could not be resolved", err, nil)
	}
	return NewWithBaseDirectory(descriptor, baseDirectory, config)
}

// NewWithBaseDirectory validates a descriptor and resolves its relative
// command, script arguments, and working directory from baseDirectory.
func NewWithBaseDirectory(descriptor processprotocol.Descriptor, baseDirectory string, config Config) (*Analyzer, error) {
	if err := processprotocol.ValidateDescriptor(descriptor); err != nil {
		return nil, wrapDescriptorError(err)
	}
	baseDirectory, err := normalizeBaseDirectory(baseDirectory)
	if err != nil {
		return nil, err
	}
	if err := validateDescriptorStrings(descriptor); err != nil {
		return nil, err
	}

	command, err := resolveCommand(descriptor.Command, baseDirectory)
	if err != nil {
		return nil, err
	}
	args, err := resolveArguments(descriptor.Args, baseDirectory)
	if err != nil {
		return nil, err
	}
	workingDirectory, err := resolveWorkingDirectory(descriptor.WorkingDirectory, baseDirectory)
	if err != nil {
		return nil, err
	}

	return &Analyzer{
		descriptor:       cloneDescriptor(descriptor),
		baseDirectory:    baseDirectory,
		command:          command,
		args:             args,
		workingDirectory: workingDirectory,
		config:           config.normalized(),
	}, nil
}

// LoadDescriptor reads and validates one explicitly supplied descriptor file.
// It never starts the descriptor command.
func LoadDescriptor(path string) (*Analyzer, error) {
	return LoadDescriptorWithConfig(path, DefaultConfig())
}

// LoadDescriptorWithConfig reads one descriptor file with explicit process
// limits. The file path is normalized before relative descriptor values are
// resolved.
func LoadDescriptorWithConfig(path string, config Config) (*Analyzer, error) {
	if strings.TrimSpace(path) == "" {
		return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "external analyzer descriptor path is required", nil)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, analysis.WrapHostError(analysis.ErrInvalidManifest, "external analyzer descriptor path could not be normalized", err, map[string]any{"path": path})
	}
	absolute = filepath.Clean(absolute)
	data, err := os.ReadFile(absolute)
	if err != nil {
		return nil, analysis.WrapHostError(analysis.ErrInvalidManifest, "external analyzer descriptor could not be read", err, map[string]any{"path": absolute})
	}
	descriptor, err := processprotocol.DecodeDescriptor(data)
	if err != nil {
		return nil, wrapDescriptorError(err)
	}
	return NewWithBaseDirectory(descriptor, filepath.Dir(absolute), config)
}

// NewFromDescriptor is an alias for LoadDescriptor for callers that prefer a
// constructor-shaped name.
func NewFromDescriptor(path string) (*Analyzer, error) {
	return LoadDescriptor(path)
}

// Descriptor returns a defensive copy of the validated descriptor.
func (a *Analyzer) Descriptor() processprotocol.Descriptor {
	if a == nil {
		return processprotocol.Descriptor{}
	}
	return cloneDescriptor(a.descriptor)
}

// Manifest returns the descriptor manifest without launching a process.
func (a *Analyzer) Manifest() analysis.Manifest {
	if a == nil {
		return analysis.Manifest{}
	}
	return cloneManifest(a.descriptor.Manifest)
}

// Detect runs one isolated process detection request.
func (a *Analyzer) Detect(ctx context.Context, request analysis.DetectRequest) (analysis.DetectionCandidate, error) {
	requestID := nextRequestID()
	selection := processprotocol.Frame{
		Type:        processprotocol.FrameDetect,
		RequestID:   requestID,
		ProjectRoot: request.ProjectRoot,
	}
	session, err := a.execute(ctx, processprotocol.FrameDetect, requestID, selection)
	if err != nil {
		return analysis.DetectionCandidate{}, err
	}
	candidate, ok := session.Candidate()
	if !ok {
		return analysis.DetectionCandidate{}, a.failure("detect did not return a candidate", nil, session, nil)
	}
	return candidate, nil
}

// Analyze runs one isolated process analysis request. The common host applies
// final result validation after this method returns.
func (a *Analyzer) Analyze(ctx context.Context, request analysis.AnalyzeRequest) (analysis.AnalysisResult, error) {
	requestID := nextRequestID()
	selection := request.Selection
	options := request.Options
	requestFrame := processprotocol.Frame{
		Type:        processprotocol.FrameAnalyze,
		RequestID:   requestID,
		ProjectRoot: request.ProjectRoot,
		Selection:   &selection,
		Options:     &options,
	}
	session, err := a.execute(ctx, processprotocol.FrameAnalyze, requestID, requestFrame)
	if err != nil {
		return analysis.AnalysisResult{}, err
	}
	result, ok := session.ResultWithDiagnostics()
	if !ok {
		return analysis.AnalysisResult{}, a.failure("analyze did not return a result", nil, session, nil)
	}
	return result, nil
}

func (a *Analyzer) execute(ctx context.Context, operation processprotocol.FrameType, requestID string, requestFrame processprotocol.Frame) (*processprotocol.SessionValidator, error) {
	if a == nil {
		return nil, analysis.NewHostError(analysis.ErrAnalyzerFailed, "external analyzer is nil", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		return nil, a.cancelled(operation, ctx.Err(), "external analyzer was cancelled before launch", "")
	}

	// Always apply the adapter's operation cap. context.WithTimeout naturally
	// preserves an earlier caller deadline while still bounding callers that
	// provide a later deadline.
	operationContext, operationCancel := context.WithTimeout(ctx, a.config.OperationTimeout)
	defer operationCancel()

	runner, err := startRunner(a, operation)
	if err != nil {
		return nil, err
	}
	cleaned := false
	defer func() {
		if !cleaned {
			_ = runner.terminate()
		}
	}()

	session, err := processprotocol.NewProtocolSession(operation, requestID, a.Manifest())
	if err != nil {
		return nil, a.failureWithStderr("external analyzer session could not be created", err, nil, nil, runner.stderrText())
	}

	helloContext, helloCancel := context.WithTimeout(operationContext, a.config.HelloTimeout)
	helloEvent, helloOK := runner.next(helloContext)
	helloCancel()
	if !helloOK {
		if ctx.Err() != nil {
			_ = runner.terminate()
			cleaned = true
			return nil, a.cancelled(operation, ctx.Err(), "external analyzer was cancelled during handshake", runner.stderrText())
		}
		if helloContext.Err() == context.DeadlineExceeded {
			_ = runner.terminate()
			cleaned = true
			return nil, a.cancelled(operation, context.DeadlineExceeded, "external analyzer hello timed out", runner.stderrText())
		}
		_ = runner.terminate()
		cleaned = true
		return nil, a.failureWithStderr("external analyzer exited before hello", nil, nil, nil, runner.stderrText())
	}
	if helloEvent.err != nil {
		if ctx.Err() != nil {
			_ = runner.terminate()
			cleaned = true
			return nil, a.cancelled(operation, ctx.Err(), "external analyzer was cancelled during handshake", runner.stderrText())
		}
		_ = runner.terminate()
		cleaned = true
		return nil, a.failureWithStderr("external analyzer hello could not be read", helloEvent.err, nil, nil, runner.stderrText())
	}
	if err := session.AcceptPluginFrame(helloEvent.frame); err != nil {
		_ = runner.terminate()
		cleaned = true
		return nil, a.failureWithStderr("external analyzer hello was rejected", err, nil, nil, runner.stderrText())
	}

	if err := session.AcceptHostFrame(requestFrame); err != nil {
		_ = runner.terminate()
		cleaned = true
		return nil, a.failureWithStderr("external analyzer request was rejected", err, nil, nil, runner.stderrText())
	}
	if err := runner.writeContext(operationContext, requestFrame); err != nil {
		_ = runner.terminate()
		cleaned = true
		if ctx.Err() != nil {
			return nil, a.cancelled(operation, ctx.Err(), "external analyzer was cancelled while sending the request", runner.stderrText())
		}
		if operationContext.Err() == context.DeadlineExceeded {
			return nil, a.cancelled(operation, context.DeadlineExceeded, "external analyzer operation timed out while sending the request", runner.stderrText())
		}
		return nil, a.failureWithStderr("external analyzer request could not be written", err, nil, nil, runner.stderrText())
	}

	terminalSeen := false
	for {
		event, ok := runner.next(operationContext)
		if !ok {
			if ctx.Err() != nil {
				_ = runner.cancelAndTerminate(requestID, ctx.Err())
				cleaned = true
				return nil, a.cancelled(operation, ctx.Err(), "external analyzer was cancelled", runner.stderrText())
			}
			if operationContext.Err() == context.DeadlineExceeded {
				_ = runner.cancelAndTerminate(requestID, context.DeadlineExceeded)
				cleaned = true
				return nil, a.cancelled(operation, context.DeadlineExceeded, "external analyzer operation timed out", runner.stderrText())
			}
			_ = runner.terminate()
			cleaned = true
			return nil, a.failureWithStderr("external analyzer stopped producing protocol frames", nil, session, nil, runner.stderrText())
		}
		if event.err != nil {
			if errors.Is(event.err, io.EOF) || (terminalSeen && errors.Is(event.err, os.ErrClosed)) {
				break
			}
			if ctx.Err() != nil {
				_ = runner.cancelAndTerminate(requestID, ctx.Err())
				cleaned = true
				return nil, a.cancelled(operation, ctx.Err(), "external analyzer was cancelled", runner.stderrText())
			}
			_ = runner.terminate()
			cleaned = true
			return nil, a.failureWithStderr("external analyzer protocol stream could not be read", event.err, session, nil, runner.stderrText())
		}
		if err := session.AcceptPluginFrame(event.frame); err != nil {
			_ = runner.terminate()
			cleaned = true
			return nil, a.failureWithStderr("external analyzer protocol frame was rejected", err, session, nil, runner.stderrText())
		}
		if _, ok := session.Terminal(); ok {
			terminalSeen = true
			// Closing stdin lets a well-behaved one-request plugin finish even if
			// it is waiting for EOF after its terminal frame. Further stdout is
			// still read and validated so late or duplicate frames are rejected.
			runner.closeStdin()
		}
	}

	waitErr := runner.finish()
	cleaned = true
	if ctx.Err() != nil {
		return nil, a.cancelled(operation, ctx.Err(), "external analyzer was cancelled", runner.stderrText())
	}
	if operationContext.Err() != nil {
		return nil, a.cancelled(operation, operationContext.Err(), "external analyzer operation timed out", runner.stderrText())
	}
	if !terminalSeen || !session.Complete() {
		return nil, a.failureWithStderr("external analyzer ended without a complete terminal exchange", waitErr, session, nil, runner.stderrText())
	}
	terminal, _ := session.Terminal()
	if terminal.Type == processprotocol.FrameFatal {
		cause := errors.New(terminal.Message)
		details := map[string]any{"fatal_code": terminal.Code}
		if terminal.Details != nil {
			details["fatal_details"] = terminal.Details
		}
		return nil, a.failureWithStderr("external analyzer reported a fatal error", cause, session, details, runner.stderrText())
	}
	if waitErr != nil {
		return nil, a.failureWithStderr("external analyzer exited unsuccessfully", waitErr, session, nil, runner.stderrText())
	}
	return session, nil
}

func (a *Analyzer) failure(message string, cause error, session *processprotocol.SessionValidator, extra map[string]any) error {
	return a.failureWithStderr(message, cause, session, extra, "")
}

func (a *Analyzer) failureWithStderr(message string, cause error, session *processprotocol.SessionValidator, extra map[string]any, stderr string) error {
	details := map[string]any{
		"operation": "unknown",
	}
	if session != nil {
		details["operation"] = string(session.Operation())
		details["request_id"] = session.RequestID()
	}
	if a != nil {
		details["analyzer_id"] = a.descriptor.Manifest.ID
		details["command"] = append([]string{a.command}, a.args...)
	}
	for key, value := range extra {
		details[key] = value
	}
	if stderr != "" {
		details["stderr"] = stderr
	}
	var protocolErr *processprotocol.ProtocolError
	if errors.As(cause, &protocolErr) {
		details["protocol_kind"] = string(protocolErr.Kind)
	}
	return analysis.WrapHostError(analysis.ErrAnalyzerFailed, message, cause, details)
}

func (a *Analyzer) cancelled(operation processprotocol.FrameType, cause error, message, stderr string) error {
	details := map[string]any{
		"operation":   string(operation),
		"analyzer_id": a.descriptor.Manifest.ID,
	}
	if stderr != "" {
		details["stderr"] = stderr
	}
	return analysis.WrapHostError(analysis.ErrCancelled, message, cause, details)
}

func nextRequestID() string {
	return fmt.Sprintf("req-%d", atomic.AddUint64(&requestSequence, 1))
}
