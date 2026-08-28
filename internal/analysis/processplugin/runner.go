// Package processplugin provides the child-side runner used by compiled
// analyzer entrypoints.
//
// The runner owns only process protocol orchestration. Analyzer selection,
// option resolution, result normalization, and final semantic validation stay
// in the shared analysis host and the existing language analyzers.
package processplugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/processprotocol"
)

// Config contains child-side protocol limits. A zero or negative value uses
// the published process-protocol default.
type Config struct {
	MaxFrameBytes int
}

// DefaultConfig returns the v1 compiled-plugin runner limits.
func DefaultConfig() Config {
	return Config{MaxFrameBytes: processprotocol.DefaultMaxFrameBytes}
}

func (config Config) normalized() Config {
	if config.MaxFrameBytes <= 0 {
		return DefaultConfig()
	}
	return config
}

// Run serves one analyzer process session using the published NDJSON
// protocol. The process emits one hello frame, accepts one detect or analyze
// request, and emits exactly one terminal done or fatal frame.
//
// The stderr writer is used only for runtime error logging. It is never used
// for protocol frames; stdout remains protocol-only.
func Run(ctx context.Context, analyzer analysis.Analyzer, stdin io.Reader, stdout, stderr io.Writer) error {
	return RunWithConfig(ctx, analyzer, stdin, stdout, stderr, DefaultConfig())
}

// RunWithConfig is Run with explicit protocol limits, primarily for focused
// tests and hosts that intentionally use a smaller frame budget.
func RunWithConfig(ctx context.Context, analyzer analysis.Analyzer, stdin io.Reader, stdout, stderr io.Writer, config Config) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if analyzer == nil {
		return fail(stderr, "analyzer is nil", errors.New("compiled analyzer runner requires an analyzer"))
	}
	manifest := analyzer.Manifest()
	if err := analysis.ValidateManifest(manifest); err != nil {
		return fail(stderr, "analyzer manifest is invalid", err)
	}
	if ctx.Err() != nil {
		return fail(stderr, "compiled analyzer runner was cancelled before hello", ctx.Err())
	}

	config = config.normalized()
	encoder := processprotocol.NewEncoderWithLimit(stdout, config.MaxFrameBytes)
	helloManifest := manifest
	hello := processprotocol.Frame{
		Type:     processprotocol.FrameHello,
		Protocol: processprotocol.ProtocolVersion,
		Manifest: &helloManifest,
	}
	if err := encoder.WriteFrame(hello); err != nil {
		return fail(stderr, "compiled analyzer hello could not be written", err)
	}

	decoder := processprotocol.NewDecoderWithLimit(stdin, config.MaxFrameBytes)
	request, err := decoder.ReadFrame()
	if err != nil {
		return fail(stderr, "compiled analyzer request could not be read", err)
	}
	if request.Type != processprotocol.FrameDetect && request.Type != processprotocol.FrameAnalyze {
		return fail(stderr, "compiled analyzer received an unsupported operation", fmt.Errorf("frame type %q", request.Type))
	}

	session, err := processprotocol.NewProtocolSession(request.Type, request.RequestID, manifest)
	if err != nil {
		return fail(stderr, "compiled analyzer protocol session could not be created", err)
	}
	if err := session.AcceptPluginFrame(hello); err != nil {
		return fail(stderr, "compiled analyzer hello was rejected by the session validator", err)
	}
	if err := session.AcceptHostFrame(request); err != nil {
		return fail(stderr, "compiled analyzer request was rejected by the session validator", err)
	}

	return serve(ctx, analyzer, manifest, decoder, encoder, session, request, stderr)
}

type operationOutput struct {
	candidate analysis.DetectionCandidate
	result    analysis.AnalysisResult
	err       error
}

type frameInput struct {
	frame processprotocol.Frame
	err   error
}

func serve(
	ctx context.Context,
	analyzer analysis.Analyzer,
	manifest analysis.Manifest,
	decoder *processprotocol.Decoder,
	encoder *processprotocol.Encoder,
	session *processprotocol.ProtocolSession,
	request processprotocol.Frame,
	stderr io.Writer,
) error {
	operationContext, cancel := context.WithCancel(ctx)
	defer cancel()

	output := make(chan operationOutput, 1)
	go func() {
		output <- invoke(operationContext, analyzer, manifest, request)
	}()

	// The host normally sends no more input until the operation is complete,
	// but keeping one bounded reader active allows the existing adapter to
	// cancel a running operation without inventing a second control channel.
	input := make(chan frameInput, 1)
	go func() {
		frame, err := decoder.ReadFrame()
		input <- frameInput{frame: frame, err: err}
	}()

	for {
		select {
		case result := <-output:
			if ctx.Err() != nil {
				return emitFatal(stderr, session, encoder, request.RequestID, "cancelled", "compiled analyzer operation was cancelled", nil)
			}
			if result.err != nil {
				report(stderr, "compiled analyzer operation failed", result.err)
				return emitErrorFatal(stderr, session, encoder, request.RequestID, manifest, result.err)
			}
			return emitOutput(stderr, session, encoder, request, result)

		case event := <-input:
			if errors.Is(event.err, io.EOF) {
				// EOF after the request simply means the host has no cancel
				// frame to send. Continue waiting for the analyzer result.
				input = nil
				continue
			}
			if event.err != nil {
				cancel()
				report(stderr, "compiled analyzer control stream failed", event.err)
				return emitProtocolFatal(stderr, session, encoder, request.RequestID, event.err)
			}
			if event.frame.Type != processprotocol.FrameCancel {
				cancel()
				err := fmt.Errorf("unexpected frame %q while operation %q was running", event.frame.Type, request.Type)
				report(stderr, "compiled analyzer received an invalid control frame", err)
				return emitProtocolFatal(stderr, session, encoder, request.RequestID, err)
			}
			if err := session.AcceptHostFrame(event.frame); err != nil {
				cancel()
				report(stderr, "compiled analyzer cancellation frame was rejected", err)
				return emitProtocolFatal(stderr, session, encoder, request.RequestID, err)
			}
			cancel()
			return emitFatal(stderr, session, encoder, request.RequestID, "cancelled", cancelMessage(event.frame.Reason), nil)

		case <-ctx.Done():
			cancel()
			return emitFatal(stderr, session, encoder, request.RequestID, "cancelled", "compiled analyzer operation was cancelled", nil)
		}
	}
}

func invoke(ctx context.Context, analyzer analysis.Analyzer, manifest analysis.Manifest, request processprotocol.Frame) (output operationOutput) {
	defer func() {
		if recovered := recover(); recovered != nil {
			output = operationOutput{err: fmt.Errorf("analyzer panicked: %v", recovered)}
		}
	}()

	switch request.Type {
	case processprotocol.FrameDetect:
		candidate, err := analyzer.Detect(ctx, analysis.DetectRequest{ProjectRoot: request.ProjectRoot})
		return operationOutput{candidate: candidate, err: err}
	case processprotocol.FrameAnalyze:
		selection := *request.Selection
		options := canonicalizeOptions(*request.Options, manifest)
		result, err := analyzer.Analyze(ctx, analysis.AnalyzeRequest{
			ProjectRoot: request.ProjectRoot,
			Selection:   selection,
			Options:     options,
		})
		return operationOutput{result: result, err: err}
	default:
		return operationOutput{err: fmt.Errorf("unsupported operation %q", request.Type)}
	}
}

// canonicalizeOptions restores the typed representation produced by
// analysis.ResolveOptions after the protocol decoder has unmarshaled option
// values through any. In particular, JSON arrays arrive as []any, while the
// existing analyzers intentionally consume string[] options as []string.
// Values that do not match a declared string[] option are left untouched so
// the host remains the authority for option validation.
func canonicalizeOptions(options analysis.EffectiveOptions, manifest analysis.Manifest) analysis.EffectiveOptions {
	values := make(map[string]any, len(options.Values))
	for name, value := range options.Values {
		values[name] = value
	}
	for _, descriptor := range manifest.Options {
		if descriptor.Type != "string[]" {
			continue
		}
		items, ok := values[descriptor.Name].([]any)
		if !ok {
			continue
		}
		stringsValue := make([]string, len(items))
		valid := true
		for index, item := range items {
			stringValue, itemOK := item.(string)
			if !itemOK {
				valid = false
				break
			}
			stringsValue[index] = stringValue
		}
		if valid {
			values[descriptor.Name] = stringsValue
		}
	}
	sources := make(map[string]string, len(options.Sources))
	for name, source := range options.Sources {
		sources[name] = source
	}
	options.Values = values
	options.Sources = sources
	return options
}

func emitOutput(stderr io.Writer, session *processprotocol.ProtocolSession, encoder *processprotocol.Encoder, request processprotocol.Frame, output operationOutput) error {
	switch request.Type {
	case processprotocol.FrameDetect:
		candidate := output.candidate
		candidateFrame := processprotocol.Frame{
			Type:      processprotocol.FrameCandidate,
			RequestID: request.RequestID,
			Candidate: &candidate,
		}
		if err := session.AcceptPluginFrame(candidateFrame); err != nil {
			report(stderr, "compiled analyzer candidate was rejected", err)
			return emitProtocolFatal(stderr, session, encoder, request.RequestID, err)
		}
		if err := encoder.WriteFrame(candidateFrame); err != nil {
			return fail(stderr, "compiled analyzer candidate could not be written", err)
		}
		done := processprotocol.Frame{Type: processprotocol.FrameDone, RequestID: request.RequestID, Status: analysis.StatusComplete}
		if err := session.AcceptPluginFrame(done); err != nil {
			return fail(stderr, "compiled analyzer detect terminal frame was rejected", err)
		}
		if err := encoder.WriteFrame(done); err != nil {
			return fail(stderr, "compiled analyzer detect terminal frame could not be written", err)
		}
		return nil

	case processprotocol.FrameAnalyze:
		if !validStatus(output.result.Status) {
			err := fmt.Errorf("analysis result has invalid status %q", output.result.Status)
			report(stderr, "compiled analyzer result was rejected", err)
			return emitProtocolFatal(stderr, session, encoder, request.RequestID, err)
		}
		for _, diagnostic := range output.result.Diagnostics {
			diagnostic := diagnostic
			diagnosticFrame := processprotocol.Frame{
				Type:       processprotocol.FrameDiagnostic,
				RequestID:  request.RequestID,
				Diagnostic: &diagnostic,
			}
			if err := session.AcceptPluginFrame(diagnosticFrame); err != nil {
				report(stderr, "compiled analyzer diagnostic was rejected", err)
				return emitProtocolFatal(stderr, session, encoder, request.RequestID, err)
			}
			if err := encoder.WriteFrame(diagnosticFrame); err != nil {
				return fail(stderr, "compiled analyzer diagnostic could not be written", err)
			}
		}
		result := output.result
		resultFrame := processprotocol.Frame{
			Type:      processprotocol.FrameResult,
			RequestID: request.RequestID,
			Result:    &result,
		}
		if err := session.AcceptPluginFrame(resultFrame); err != nil {
			report(stderr, "compiled analyzer result was rejected", err)
			return emitProtocolFatal(stderr, session, encoder, request.RequestID, err)
		}
		if err := encoder.WriteFrame(resultFrame); err != nil {
			return fail(stderr, "compiled analyzer result could not be written", err)
		}
		done := processprotocol.Frame{Type: processprotocol.FrameDone, RequestID: request.RequestID, Status: result.Status}
		if err := session.AcceptPluginFrame(done); err != nil {
			return fail(stderr, "compiled analyzer analyze terminal frame was rejected", err)
		}
		if err := encoder.WriteFrame(done); err != nil {
			return fail(stderr, "compiled analyzer analyze terminal frame could not be written", err)
		}
		return nil
	default:
		return fail(stderr, "compiled analyzer output used an unsupported operation", fmt.Errorf("frame type %q", request.Type))
	}
}

func emitErrorFatal(stderr io.Writer, session *processprotocol.ProtocolSession, encoder *processprotocol.Encoder, requestID string, manifest analysis.Manifest, cause error) error {
	code := string(analysis.ErrorCodeOf(cause))
	if errors.Is(cause, context.Canceled) || errors.Is(cause, context.DeadlineExceeded) {
		code = "cancelled"
	}
	if code == "" || code == string(analysis.ErrHostFailure) {
		code = string(analysis.ErrAnalyzerFailed)
	}
	details := map[string]any{"analyzer_id": manifest.ID}
	var hostErr *analysis.HostError
	if errors.As(cause, &hostErr) && hostErr.Details != nil {
		details["error_details"] = hostErr.Details
	}
	return emitFatal(stderr, session, encoder, requestID, code, errorMessage(cause), details)
}

func emitProtocolFatal(stderr io.Writer, session *processprotocol.ProtocolSession, encoder *processprotocol.Encoder, requestID string, cause error) error {
	details := map[string]any{"protocol_error": cause.Error()}
	var protocolErr *processprotocol.ProtocolError
	if errors.As(cause, &protocolErr) {
		details["protocol_kind"] = string(protocolErr.Kind)
	}
	return emitFatal(stderr, session, encoder, requestID, "protocol_error", "compiled analyzer protocol exchange failed", details)
}

func emitFatal(stderr io.Writer, session *processprotocol.ProtocolSession, encoder *processprotocol.Encoder, requestID, code, message string, details map[string]any) error {
	if strings.TrimSpace(code) == "" {
		code = "analyzer_failed"
	}
	if strings.TrimSpace(message) == "" {
		message = "compiled analyzer failed"
	}
	frame := processprotocol.Frame{
		Type:      processprotocol.FrameFatal,
		RequestID: requestID,
		Code:      code,
		Message:   message,
		Details:   details,
	}
	if err := session.AcceptPluginFrame(frame); err != nil {
		return fail(stderr, "compiled analyzer fatal frame was rejected", err)
	}
	if err := encoder.WriteFrame(frame); err != nil {
		return fail(stderr, "compiled analyzer fatal frame could not be written", err)
	}
	return nil
}

func validStatus(status analysis.AnalysisStatus) bool {
	switch status {
	case analysis.StatusComplete, analysis.StatusPartial, analysis.StatusFailed, analysis.StatusCancelled:
		return true
	default:
		return false
	}
}

func cancelMessage(reason string) string {
	if strings.TrimSpace(reason) == "" {
		return "compiled analyzer operation was cancelled"
	}
	return "compiled analyzer operation was cancelled: " + reason
}

func errorMessage(err error) string {
	if err == nil || strings.TrimSpace(err.Error()) == "" {
		return "compiled analyzer failed"
	}
	return err.Error()
}

func report(stderr io.Writer, message string, err error) {
	if stderr == nil {
		return
	}
	if err == nil {
		_, _ = fmt.Fprintln(stderr, "arch-view analyzer plugin:", message)
		return
	}
	_, _ = fmt.Fprintf(stderr, "arch-view analyzer plugin: %s: %v\n", message, err)
}

func fail(stderr io.Writer, message string, err error) error {
	report(stderr, message, err)
	return err
}
