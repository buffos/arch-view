package processprotocol

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/buffo/arch-view/internal/analysis"
)

const fixtureProcessEnv = "ARCH_VIEW_EXTERNAL_FIXTURE"

// TestExternalPluginFixtureProcess is a test-only process entrypoint. It is
// selected explicitly by the parent test binary and is never registered in an
// analyzer registry or shipped with the application.
func TestExternalPluginFixtureProcess(t *testing.T) {
	if os.Getenv(fixtureProcessEnv) != "1" {
		return
	}
	// A compiled Go test binary normally writes PASS to stdout after the test
	// returns. Exit from the fixture entrypoint explicitly so stdout remains
	// protocol-only, just as it would for a real external analyzer process.
	defer os.Exit(0)

	mode := os.Getenv("ARCH_VIEW_EXTERNAL_FIXTURE_MODE")
	manifest := fixtureManifest()
	if mode == "stderr-log" {
		_, _ = fmt.Fprintln(os.Stderr, "fixture diagnostic log")
	}
	if mode == "stdout-log" {
		fixtureWriteRaw("fixture log on stdout\n")
		return
	}
	if mode == "hello-request-id" {
		fixtureWriteRaw(`{"type":"hello","request_id":"unexpected","protocol":"arch-view.analyzer/v1","manifest":{"id":"org.example.external-fixture","version":"1.0.0","language":"fixture","api_version":"arch-view.analyzer/v1","detection_markers":[{"kind":"file","value":"fixture.marker","weight":1}],"capabilities":["detect"],"options":[]}}` + "\n")
		return
	}
	if mode == "manifest-mismatch" {
		manifest.ID = "org.example.external-fixture.other"
	}
	if mode == "unsupported-hello" {
		fixtureWriteRaw(`{"type":"hello","protocol":"arch-view.analyzer/v9","manifest":{"id":"org.example.external-fixture","version":"1.0.0","language":"fixture","api_version":"arch-view.analyzer/v9","detection_markers":[{"kind":"file","value":"fixture.marker","weight":1}],"capabilities":["detect"],"options":[]}}` + "\n")
		return
	}
	fixtureWriteFrame(Frame{Type: FrameHello, Protocol: ProtocolVersion, Manifest: &manifest})

	decoder := NewDecoder(os.Stdin)
	request, err := decoder.ReadFrame()
	if err != nil {
		fixtureFail("read request", err)
	}
	if request.Type != FrameDetect && request.Type != FrameAnalyze {
		fixtureFail("unexpected request", fmt.Errorf("got %q", request.Type))
	}

	if mode == "delayed" {
		delayMilliseconds := 50
		if value, parseErr := strconv.Atoi(os.Getenv("ARCH_VIEW_EXTERNAL_FIXTURE_DELAY_MS")); parseErr == nil && value >= 0 {
			delayMilliseconds = value
		}
		time.Sleep(time.Duration(delayMilliseconds) * time.Millisecond)
	}
	if mode == "wait-for-cancel" {
		cancel, cancelErr := decoder.ReadFrame()
		if cancelErr != nil {
			fixtureFail("read cancel", cancelErr)
		}
		if cancel.Type != FrameCancel || cancel.RequestID != request.RequestID {
			fixtureFail("unexpected cancel", fmt.Errorf("got %s/%s", cancel.Type, cancel.RequestID))
		}
		fixtureWriteFrame(Frame{Type: FrameFatal, RequestID: request.RequestID, Code: "cancelled", Message: "fixture cancelled"})
		return
	}

	switch mode {
	case "malformed-json":
		fixtureWriteRaw("{not valid JSON\n")
	case "unknown-frame":
		fixtureWriteRaw(`{"type":"unknown","request_id":"` + request.RequestID + `"}` + "\n")
	case "wrong-request-id":
		fixtureWriteWrongRequestID(request)
	case "duplicate-terminal":
		fixtureWriteValidExchange(request, manifest)
		fixtureWriteFrame(Frame{Type: FrameDone, RequestID: request.RequestID, Status: analysis.StatusComplete})
	case "late-output":
		fixtureWriteValidExchange(request, manifest)
		fixtureWriteFrame(Frame{
			Type:       FrameDiagnostic,
			RequestID:  request.RequestID,
			Diagnostic: &analysis.Diagnostic{Code: "late", Severity: "warning", Message: "late output", Recoverable: true},
		})
	case "oversized-line":
		size := DefaultMaxFrameBytes + 1
		if value, parseErr := strconv.Atoi(os.Getenv("ARCH_VIEW_EXTERNAL_FIXTURE_OVERSIZE_BYTES")); parseErr == nil && value > 0 {
			size = value
		}
		fixtureWriteRaw(strings.Repeat("x", size) + "\n")
	case "fatal":
		fixtureWriteFrame(Frame{Type: FrameFatal, RequestID: request.RequestID, Code: "fixture_failure", Message: "intentional fixture failure"})
	case "diagnostic-analyze":
		fixtureWriteDiagnosticExchange(request, manifest)
	case "partial-analyze":
		fixtureWriteAnalyzeExchange(request, manifest, analysis.StatusPartial)
	default:
		fixtureWriteValidExchange(request, manifest)
	}
}

func fixtureManifest() analysis.Manifest {
	return analysis.Manifest{
		ID:         "org.example.external-fixture",
		Version:    "1.0.0",
		Language:   "fixture",
		APIVersion: analysis.AnalyzerAPIVersion,
		DetectionMarkers: []analysis.DetectionMarker{{
			Kind:   "file",
			Value:  "fixture.marker",
			Weight: 1,
		}},
		Capabilities: []string{"detect", "static_dependencies"},
		Options: []analysis.OptionDescriptor{{
			Name:    "include_tests",
			Type:    "boolean",
			Default: false,
		}},
	}
}

func fixtureWriteValidExchange(request Frame, manifest analysis.Manifest) {
	switch request.Type {
	case FrameDetect:
		fixtureWriteFrame(Frame{
			Type:      FrameCandidate,
			RequestID: request.RequestID,
			Candidate: &analysis.DetectionCandidate{
				AnalyzerID:     manifest.ID,
				Confidence:     1,
				MatchedMarkers: []string{"fixture.marker"},
				Reason:         "detected by fixture.marker",
			},
		})
		fixtureWriteFrame(Frame{Type: FrameDone, RequestID: request.RequestID, Status: analysis.StatusComplete})
	case FrameAnalyze:
		fixtureWriteAnalyzeExchange(request, manifest, analysis.StatusComplete)
	}
}

func fixtureWriteDiagnosticExchange(request Frame, manifest analysis.Manifest) {
	fixtureWriteFrame(Frame{
		Type:       FrameDiagnostic,
		RequestID:  request.RequestID,
		Diagnostic: &analysis.Diagnostic{Code: "fixture.warning", Severity: "warning", Message: "streamed fixture warning", Recoverable: true},
	})
	fixtureWriteAnalyzeExchange(request, manifest, analysis.StatusComplete)
}

func fixtureWriteAnalyzeExchange(request Frame, manifest analysis.Manifest, status analysis.AnalysisStatus) {
	optionsFingerprint := ""
	if request.Options != nil {
		optionsFingerprint = request.Options.Fingerprint
	}
	result := analysis.AnalysisResult{
		RunID:  "fixture-run",
		Status: status,
		Analyzer: analysis.AnalyzerInfo{
			ID:         manifest.ID,
			Version:    manifest.Version,
			Language:   manifest.Language,
			APIVersion: manifest.APIVersion,
		},
		Project: analysis.ProjectInfo{
			RootLabel: "fixture",
			Boundary:  "fixture.marker",
		},
		OptionsFingerprint: optionsFingerprint,
	}
	if status == analysis.StatusPartial {
		result.Diagnostics = []analysis.Diagnostic{{
			Code:        "fixture.partial",
			Severity:    "warning",
			Message:     "fixture returned a partial result",
			Recoverable: true,
		}}
	}
	fixtureWriteFrame(Frame{Type: FrameResult, RequestID: request.RequestID, Result: &result})
	fixtureWriteFrame(Frame{Type: FrameDone, RequestID: request.RequestID, Status: status})
}

func fixtureWriteWrongRequestID(request Frame) {
	wrongID := "wrong-request-id"
	if request.Type == FrameDetect {
		candidate := fixtureManifest().ID
		fixtureWriteFrame(Frame{
			Type:      FrameCandidate,
			RequestID: wrongID,
			Candidate: &analysis.DetectionCandidate{AnalyzerID: candidate, Confidence: 1, Reason: "wrong request id"},
		})
		return
	}
	result := analysis.AnalysisResult{
		RunID:  "fixture-run",
		Status: analysis.StatusComplete,
		Analyzer: analysis.AnalyzerInfo{
			ID:         fixtureManifest().ID,
			Version:    "1.0.0",
			Language:   "fixture",
			APIVersion: analysis.AnalyzerAPIVersion,
		},
		Project: analysis.ProjectInfo{RootLabel: "fixture", Boundary: "fixture.marker"},
	}
	fixtureWriteFrame(Frame{Type: FrameResult, RequestID: wrongID, Result: &result})
}

func fixtureWriteFrame(frame Frame) {
	if err := WriteFrame(os.Stdout, frame); err != nil {
		fixtureFail("write frame", err)
	}
}

func fixtureWriteRaw(value string) {
	if _, err := io.WriteString(os.Stdout, value); err != nil {
		fixtureFail("write raw frame", err)
	}
}

func fixtureFail(operation string, err error) {
	_, _ = fmt.Fprintf(os.Stderr, "fixture %s failed: %v\n", operation, err)
	os.Exit(1)
}

type fixtureRun struct {
	Frames   []Frame
	ReadErr  error
	Stderr   string
	WaitErr  error
	Duration time.Duration
}

func runFixture(t *testing.T, mode string, operation FrameType, maxFrameBytes int, extraEnv ...string) fixtureRun {
	t.Helper()
	command := exec.CommandContext(context.Background(), os.Args[0], "-test.run", "^TestExternalPluginFixtureProcess$", "--")
	command.Env = append(os.Environ(), fixtureProcessEnv+"=1", "ARCH_VIEW_EXTERNAL_FIXTURE_MODE="+mode)
	command.Env = append(command.Env, extraEnv...)
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatalf("fixture stdout pipe: %v", err)
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		t.Fatalf("fixture stderr pipe: %v", err)
	}
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatalf("fixture stdin pipe: %v", err)
	}
	startedAt := time.Now()
	if err := command.Start(); err != nil {
		t.Fatalf("start fixture: %v", err)
	}

	decoder := NewDecoderWithLimit(bufio.NewReader(stdout), maxFrameBytes)
	result := fixtureRun{}
	first, readErr := decoder.ReadFrame()
	if readErr != nil {
		result.ReadErr = readErr
	} else {
		result.Frames = append(result.Frames, first)
		request := fixtureRequest(operation)
		if err := WriteFrame(stdin, request); err != nil {
			result.ReadErr = err
		} else if mode == "wait-for-cancel" {
			if err := WriteFrame(stdin, Frame{Type: FrameCancel, RequestID: request.RequestID, Reason: "test cancellation"}); err != nil {
				result.ReadErr = err
			} else if err := stdin.Close(); err != nil {
				result.ReadErr = err
			}
		} else if err := stdin.Close(); err != nil {
			result.ReadErr = err
		}
		if result.ReadErr == nil {
			for {
				frame, frameErr := decoder.ReadFrame()
				if errors.Is(frameErr, io.EOF) {
					break
				}
				if frameErr != nil {
					result.ReadErr = frameErr
					break
				}
				result.Frames = append(result.Frames, frame)
			}
		}
	}
	stderrBytes, stderrErr := io.ReadAll(stderr)
	if result.ReadErr == nil && stderrErr != nil {
		result.ReadErr = stderrErr
	}
	result.Stderr = string(stderrBytes)
	result.WaitErr = command.Wait()
	result.Duration = time.Since(startedAt)
	return result
}

func fixtureRequest(operation FrameType) Frame {
	requestID := "req-1"
	if operation == FrameDetect {
		return Frame{Type: FrameDetect, RequestID: requestID, ProjectRoot: "."}
	}
	return Frame{
		Type:        FrameAnalyze,
		RequestID:   requestID,
		ProjectRoot: ".",
		Selection:   &analysis.AnalyzerSelection{AnalyzerID: fixtureManifest().ID, Mode: "explicit-id"},
		Options:     &analysis.EffectiveOptions{Values: map[string]any{}, Sources: map[string]string{}, Fingerprint: "fixture-options"},
	}
}

func TestExternalFixtureSupportsDetectAndAnalyze(t *testing.T) {
	tests := []struct {
		name      string
		operation FrameType
		wantType  FrameType
	}{
		{name: "detect", operation: FrameDetect, wantType: FrameCandidate},
		{name: "analyze", operation: FrameAnalyze, wantType: FrameResult},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			run := runFixture(t, "valid", test.operation, 16*1024)
			if run.ReadErr != nil {
				t.Fatalf("fixture read: %v", run.ReadErr)
			}
			if run.WaitErr != nil {
				t.Fatalf("fixture wait: %v; stderr=%s", run.WaitErr, run.Stderr)
			}
			if len(run.Frames) != 3 {
				t.Fatalf("frame count = %d, want 3: %#v", len(run.Frames), run.Frames)
			}
			if run.Frames[0].Type != FrameHello || run.Frames[1].Type != test.wantType || run.Frames[2].Type != FrameDone {
				t.Fatalf("frame types = %s, %s, %s", run.Frames[0].Type, run.Frames[1].Type, run.Frames[2].Type)
			}
			session, err := NewSessionValidator(test.operation, "req-1", fixtureManifest())
			if err != nil {
				t.Fatalf("new session: %v", err)
			}
			if err := session.AcceptPluginFrame(run.Frames[0]); err != nil {
				t.Fatalf("session hello: %v", err)
			}
			if err := session.AcceptHostFrame(fixtureRequest(test.operation)); err != nil {
				t.Fatalf("session request: %v", err)
			}
			for _, frame := range run.Frames[1:] {
				if err := session.AcceptPluginFrame(frame); err != nil {
					t.Fatalf("session frame %s: %v", frame.Type, err)
				}
			}
			if !session.Complete() {
				t.Fatal("fixture session is not complete")
			}
		})
	}
}

func TestExternalFixtureSeparatesStderrFromProtocol(t *testing.T) {
	run := runFixture(t, "stderr-log", FrameDetect, 16*1024)
	if run.ReadErr != nil {
		t.Fatalf("fixture read: %v", run.ReadErr)
	}
	if run.WaitErr != nil {
		t.Fatalf("fixture wait: %v; stderr=%s", run.WaitErr, run.Stderr)
	}
	if !strings.Contains(run.Stderr, "fixture diagnostic log") {
		t.Fatalf("stderr = %q, want fixture diagnostic log", run.Stderr)
	}
	if len(run.Frames) != 3 || run.Frames[0].Type != FrameHello {
		t.Fatalf("stderr fixture frames = %#v", run.Frames)
	}

	contaminated := runFixture(t, "stdout-log", FrameDetect, 16*1024)
	if !errors.Is(contaminated.ReadErr, ErrMalformedJSON) {
		t.Fatalf("stdout contamination error = %v, want malformed JSON", contaminated.ReadErr)
	}
}

func TestExternalFixtureEmitsDiagnosticsFatalDelayedAndPartialStreams(t *testing.T) {
	diagnostic := runFixture(t, "diagnostic-analyze", FrameAnalyze, 16*1024)
	if diagnostic.ReadErr != nil || diagnostic.WaitErr != nil {
		t.Fatalf("diagnostic fixture errors: read=%v wait=%v stderr=%s", diagnostic.ReadErr, diagnostic.WaitErr, diagnostic.Stderr)
	}
	if len(diagnostic.Frames) != 4 || diagnostic.Frames[1].Type != FrameDiagnostic || diagnostic.Frames[2].Type != FrameResult {
		t.Fatalf("diagnostic fixture frames = %#v", diagnostic.Frames)
	}

	fatal := runFixture(t, "fatal", FrameAnalyze, 16*1024)
	if fatal.ReadErr != nil || fatal.WaitErr != nil {
		t.Fatalf("fatal fixture errors: read=%v wait=%v stderr=%s", fatal.ReadErr, fatal.WaitErr, fatal.Stderr)
	}
	if len(fatal.Frames) != 2 || fatal.Frames[1].Type != FrameFatal {
		t.Fatalf("fatal fixture frames = %#v", fatal.Frames)
	}

	partial := runFixture(t, "partial-analyze", FrameAnalyze, 16*1024)
	if partial.ReadErr != nil || partial.WaitErr != nil {
		t.Fatalf("partial fixture errors: read=%v wait=%v stderr=%s", partial.ReadErr, partial.WaitErr, partial.Stderr)
	}
	if partial.Frames[1].Result == nil || partial.Frames[1].Result.Status != analysis.StatusPartial || partial.Frames[2].Status != analysis.StatusPartial {
		t.Fatalf("partial fixture frames = %#v", partial.Frames)
	}

	delayed := runFixture(t, "delayed", FrameDetect, 16*1024, "ARCH_VIEW_EXTERNAL_FIXTURE_DELAY_MS=40")
	if delayed.ReadErr != nil || delayed.WaitErr != nil {
		t.Fatalf("delayed fixture errors: read=%v wait=%v stderr=%s", delayed.ReadErr, delayed.WaitErr, delayed.Stderr)
	}
	if delayed.Duration < 30*time.Millisecond {
		t.Fatalf("delayed fixture duration = %s, want at least 30ms", delayed.Duration)
	}

	cancelled := runFixture(t, "wait-for-cancel", FrameAnalyze, 16*1024)
	if cancelled.ReadErr != nil || cancelled.WaitErr != nil {
		t.Fatalf("cancel fixture errors: read=%v wait=%v stderr=%s", cancelled.ReadErr, cancelled.WaitErr, cancelled.Stderr)
	}
	if len(cancelled.Frames) != 2 || cancelled.Frames[1].Type != FrameFatal {
		t.Fatalf("cancel fixture frames = %#v", cancelled.Frames)
	}
	cancelSession := newFixtureSessionFromFrames(t, cancelled, FrameAnalyze)
	if err := cancelSession.AcceptHostFrame(Frame{Type: FrameCancel, RequestID: "req-1", Reason: "test cancellation"}); err != nil {
		t.Fatalf("cancel request: %v", err)
	}
	if err := cancelSession.AcceptPluginFrame(cancelled.Frames[1]); err != nil {
		t.Fatalf("cancel terminal: %v", err)
	}
}
