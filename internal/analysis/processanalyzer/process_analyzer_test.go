package processanalyzer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/processprotocol"
)

const (
	testFixtureEnv      = "ARCH_VIEW_PROCESS_ANALYZER_FIXTURE"
	testFixtureModeEnv  = "ARCH_VIEW_PROCESS_ANALYZER_MODE"
	testFixtureManifest = "org.example.process-fixture"
)

// TestProcessAnalyzerFixtureProcess is an explicitly launched test-only
// process. It exits before the test harness can write PASS to stdout so the
// child stream remains protocol-only.
func TestProcessAnalyzerFixtureProcess(t *testing.T) {
	if os.Getenv(testFixtureEnv) != "1" {
		return
	}
	defer os.Exit(0)

	manifest := processFixtureManifest()
	mode := os.Getenv(testFixtureModeEnv)
	if mode == "stderr" {
		_, _ = fmt.Fprint(os.Stderr, strings.Repeat("stderr-context ", 128))
	}
	if mode == "stdout" {
		_, _ = fmt.Fprintln(os.Stdout, "plugin log on stdout")
		return
	}
	if mode == "hello-mismatch" {
		manifest.ID = "org.example.process-fixture.other"
	}
	if mode == "hello-unsupported" {
		writeFixtureRaw(`{"type":"hello","protocol":"arch-view.analyzer/v9","manifest":{"id":"org.example.process-fixture","version":"1.0.0","language":"fixture","api_version":"arch-view.analyzer/v9","detection_markers":[{"kind":"file","value":"fixture.marker","weight":1}],"capabilities":["detect","static_dependencies"],"options":[{"name":"include_tests","type":"boolean","default":false}]}}` + "\n")
		return
	}
	writeFixtureFrame(processprotocol.Frame{Type: processprotocol.FrameHello, Protocol: processprotocol.ProtocolVersion, Manifest: &manifest})
	decoder := processprotocol.NewDecoder(os.Stdin)
	request, err := decoder.ReadFrame()
	if err != nil {
		fixtureExit("read request", err)
	}
	if mode == "assert-options" && (request.Options == nil || request.Options.Values["include_tests"] != true) {
		writeFixtureFrame(processprotocol.Frame{Type: processprotocol.FrameFatal, RequestID: request.RequestID, Code: "options_not_forwarded", Message: "include_tests was not forwarded"})
		return
	}
	if mode == "delay" {
		time.Sleep(500 * time.Millisecond)
	}
	if mode == "cancel" {
		cancel, cancelErr := decoder.ReadFrame()
		if cancelErr != nil || cancel.Type != processprotocol.FrameCancel || cancel.RequestID != request.RequestID {
			fixtureExit("read cancel", cancelErr)
		}
		writeFixtureFrame(processprotocol.Frame{Type: processprotocol.FrameFatal, RequestID: request.RequestID, Code: "cancelled", Message: "fixture cancelled"})
		return
	}

	switch mode {
	case "malformed", "stderr":
		writeFixtureRaw("{not json\n")
	case "unknown":
		writeFixtureRaw(`{"type":"unknown","request_id":"` + request.RequestID + `"}` + "\n")
	case "wrong-id":
		writeFixtureFrame(processprotocol.Frame{Type: processprotocol.FrameResult, RequestID: "wrong-id", Result: &analysis.AnalysisResult{Status: analysis.StatusComplete}})
	case "duplicate-terminal":
		writeFixtureResult(request, manifest, analysis.StatusComplete)
		writeFixtureFrame(processprotocol.Frame{Type: processprotocol.FrameDone, RequestID: request.RequestID, Status: analysis.StatusComplete})
	case "late":
		writeFixtureResult(request, manifest, analysis.StatusComplete)
		writeFixtureFrame(processprotocol.Frame{Type: processprotocol.FrameDiagnostic, RequestID: request.RequestID, Diagnostic: &analysis.Diagnostic{Code: "late", Severity: "warning", Message: "late output", Recoverable: true}})
	case "oversized":
		writeFixtureRaw(strings.Repeat("x", DefaultMaxFrameBytes+1) + "\n")
	case "fatal":
		writeFixtureFrame(processprotocol.Frame{Type: processprotocol.FrameFatal, RequestID: request.RequestID, Code: "fixture_failure", Message: "intentional fixture failure"})
	case "diagnostic":
		writeFixtureFrame(processprotocol.Frame{Type: processprotocol.FrameDiagnostic, RequestID: request.RequestID, Diagnostic: &analysis.Diagnostic{Code: "fixture.warning", Severity: "warning", Message: "streamed warning", Recoverable: true}})
		writeFixtureResult(request, manifest, analysis.StatusComplete)
	case "partial":
		writeFixtureResult(request, manifest, analysis.StatusPartial)
	default:
		if request.Type == processprotocol.FrameDetect {
			writeFixtureFrame(processprotocol.Frame{Type: processprotocol.FrameCandidate, RequestID: request.RequestID, Candidate: &analysis.DetectionCandidate{AnalyzerID: manifest.ID, Confidence: 1, MatchedMarkers: []string{"fixture.marker"}, Reason: "detected by fixture.marker"}})
			writeFixtureFrame(processprotocol.Frame{Type: processprotocol.FrameDone, RequestID: request.RequestID, Status: analysis.StatusComplete})
			return
		}
		writeFixtureResult(request, manifest, analysis.StatusComplete)
	}
}

func processFixtureManifest() analysis.Manifest {
	return analysis.Manifest{
		ID:         testFixtureManifest,
		Version:    "1.0.0",
		Language:   "fixture",
		APIVersion: analysis.AnalyzerAPIVersion,
		DetectionMarkers: []analysis.DetectionMarker{{
			Kind: "file", Value: "fixture.marker", Weight: 1,
		}},
		Capabilities: []string{"detect", "static_dependencies"},
		Options:      []analysis.OptionDescriptor{{Name: "include_tests", Type: "boolean", Default: false}},
	}
}

func writeFixtureResult(request processprotocol.Frame, manifest analysis.Manifest, status analysis.AnalysisStatus) {
	result := analysis.AnalysisResult{
		RunID:  "fixture-run",
		Status: status,
		Analyzer: analysis.AnalyzerInfo{
			ID: manifest.ID, Version: manifest.Version, Language: manifest.Language, APIVersion: manifest.APIVersion,
		},
		Project: analysis.ProjectInfo{RootLabel: "fixture", Boundary: "fixture.marker"},
	}
	if request.Options != nil {
		result.OptionsFingerprint = request.Options.Fingerprint
	}
	if status == analysis.StatusPartial {
		result.Diagnostics = []analysis.Diagnostic{{Code: "fixture.partial", Severity: "warning", Message: "partial fixture result", Recoverable: true}}
	}
	writeFixtureFrame(processprotocol.Frame{Type: processprotocol.FrameResult, RequestID: request.RequestID, Result: &result})
	writeFixtureFrame(processprotocol.Frame{Type: processprotocol.FrameDone, RequestID: request.RequestID, Status: status})
}

func writeFixtureFrame(frame processprotocol.Frame) {
	if err := processprotocol.WriteFrame(os.Stdout, frame); err != nil {
		fixtureExit("write frame", err)
	}
}

func writeFixtureRaw(value string) {
	if _, err := io.WriteString(os.Stdout, value); err != nil {
		fixtureExit("write raw frame", err)
	}
}

func fixtureExit(operation string, err error) {
	if err == nil {
		err = errors.New("unexpected fixture state")
	}
	_, _ = fmt.Fprintf(os.Stderr, "fixture %s failed: %v\n", operation, err)
	os.Exit(1)
}

func newProcessFixtureAnalyzer(t *testing.T, mode string, config Config) *Analyzer {
	t.Helper()
	t.Setenv(testFixtureEnv, "1")
	t.Setenv(testFixtureModeEnv, mode)
	return newProcessFixtureAnalyzerWithEnv(t, config)
}

func newProcessFixtureAnalyzerWithEnv(t *testing.T, config Config) *Analyzer {
	t.Helper()
	descriptor := processprotocol.Descriptor{
		SchemaVersion:    processprotocol.DescriptorSchemaVersion,
		Manifest:         processFixtureManifest(),
		Command:          os.Args[0],
		Args:             []string{"-test.run", "^TestProcessAnalyzerFixtureProcess$", "--"},
		WorkingDirectory: t.TempDir(),
	}
	analyzer, err := NewWithBaseDirectory(descriptor, mustWorkingDirectory(t), config)
	if err != nil {
		t.Fatalf("new process analyzer: %v", err)
	}
	return analyzer
}

func mustWorkingDirectory(t *testing.T) string {
	t.Helper()
	value, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	return value
}

func TestManifestDoesNotLaunchAndDescriptorValidationIsStrict(t *testing.T) {
	t.Setenv(testFixtureEnv, "")
	descriptor := processprotocol.Descriptor{
		SchemaVersion: processprotocol.DescriptorSchemaVersion,
		Manifest:      processFixtureManifest(),
		Command:       os.Args[0],
	}
	analyzer, err := New(descriptor)
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	if got := analyzer.Manifest(); got.ID != testFixtureManifest {
		t.Fatalf("manifest id = %q", got.ID)
	}
	if os.Getenv(testFixtureEnv) == "1" {
		t.Fatal("manifest inspection unexpectedly ran the fixture")
	}

	tests := []struct {
		name     string
		mutate   func(*processprotocol.Descriptor)
		wantCode analysis.ErrorCode
	}{
		{name: "unsupported api", mutate: func(value *processprotocol.Descriptor) { value.Manifest.APIVersion = "arch-view.analyzer/v9" }, wantCode: analysis.ErrAPIIncompatible},
		{name: "duplicate option", mutate: func(value *processprotocol.Descriptor) {
			value.Manifest.Options = append(value.Manifest.Options, value.Manifest.Options[0])
		}, wantCode: analysis.ErrInvalidManifest},
		{name: "empty command", mutate: func(value *processprotocol.Descriptor) { value.Command = " " }, wantCode: analysis.ErrInvalidManifest},
		{name: "unsafe command", mutate: func(value *processprotocol.Descriptor) { value.Command = "python\x00evil" }, wantCode: analysis.ErrInvalidManifest},
		{name: "unsafe argument", mutate: func(value *processprotocol.Descriptor) { value.Args = []string{"script.py\n"} }, wantCode: analysis.ErrInvalidManifest},
		{name: "missing working directory", mutate: func(value *processprotocol.Descriptor) { value.WorkingDirectory = "missing-directory" }, wantCode: analysis.ErrInvalidManifest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := descriptor
			value.Manifest = cloneManifest(value.Manifest)
			test.mutate(&value)
			_, err := New(value)
			if got := analysis.ErrorCodeOf(err); got != test.wantCode {
				t.Fatalf("error code = %q, want %q; err=%v", got, test.wantCode, err)
			}
		})
	}
}

func TestProcessAnalyzerDetectAndAnalyze(t *testing.T) {
	analyzer := newProcessFixtureAnalyzer(t, "valid", Config{})
	candidate, err := analyzer.Detect(context.Background(), analysis.DetectRequest{ProjectRoot: t.TempDir()})
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if candidate.AnalyzerID != testFixtureManifest || candidate.Confidence != 1 {
		t.Fatalf("candidate = %#v", candidate)
	}

	result, err := analyzer.Analyze(context.Background(), analysis.AnalyzeRequest{
		ProjectRoot: t.TempDir(),
		Selection:   analysis.AnalyzerSelection{AnalyzerID: testFixtureManifest, Mode: "explicit-id", Confidence: 1},
		Options:     analysis.EffectiveOptions{Values: map[string]any{"include_tests": true}, Sources: map[string]string{"include_tests": "cli"}, Fingerprint: "fixture-options"},
	})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if result.Status != analysis.StatusComplete || result.OptionsFingerprint != "fixture-options" {
		t.Fatalf("result = %#v", result)
	}
}

func TestProcessAnalyzerForwardsOptionsAndMergesDiagnostics(t *testing.T) {
	analyzer := newProcessFixtureAnalyzer(t, "assert-options", Config{})
	result, err := analyzer.Analyze(context.Background(), analysis.AnalyzeRequest{
		ProjectRoot: t.TempDir(),
		Selection:   analysis.AnalyzerSelection{AnalyzerID: testFixtureManifest, Mode: "explicit-id"},
		Options:     analysis.EffectiveOptions{Values: map[string]any{"include_tests": true}, Fingerprint: "options"},
	})
	if err != nil {
		t.Fatalf("option forwarding: %v", err)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("assert-options diagnostics = %#v", result.Diagnostics)
	}

	analyzer = newProcessFixtureAnalyzer(t, "diagnostic", Config{})
	result, err = analyzer.Analyze(context.Background(), analysis.AnalyzeRequest{
		ProjectRoot: t.TempDir(),
		Selection:   analysis.AnalyzerSelection{AnalyzerID: testFixtureManifest, Mode: "explicit-id"},
		Options:     analysis.EffectiveOptions{Values: map[string]any{}, Fingerprint: "options"},
	})
	if err != nil {
		t.Fatalf("diagnostic analyze: %v", err)
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "fixture.warning" {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
}

func TestProcessAnalyzerPartialResultRemainsAvailable(t *testing.T) {
	analyzer := newProcessFixtureAnalyzer(t, "partial", Config{})
	result, err := analyzer.Analyze(context.Background(), analysis.AnalyzeRequest{
		ProjectRoot: t.TempDir(),
		Selection:   analysis.AnalyzerSelection{AnalyzerID: testFixtureManifest, Mode: "explicit-id"},
		Options:     analysis.EffectiveOptions{Values: map[string]any{}, Fingerprint: "options"},
	})
	if err != nil {
		t.Fatalf("partial analyze: %v", err)
	}
	if result.Status != analysis.StatusPartial || len(result.Diagnostics) != 1 || !result.Diagnostics[0].Recoverable {
		t.Fatalf("partial result = %#v", result)
	}
}

func TestProcessAnalyzerRejectsProtocolFailuresAndRetainsBoundedStderr(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{name: "manifest mismatch", mode: "hello-mismatch"},
		{name: "unsupported hello", mode: "hello-unsupported"},
		{name: "stdout contamination", mode: "stdout"},
		{name: "malformed json", mode: "malformed"},
		{name: "unknown frame", mode: "unknown"},
		{name: "wrong request id", mode: "wrong-id"},
		{name: "duplicate terminal", mode: "duplicate-terminal"},
		{name: "late output", mode: "late"},
		{name: "oversized line", mode: "oversized"},
		{name: "fatal frame", mode: "fatal"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			analyzer := newProcessFixtureAnalyzer(t, test.mode, Config{MaxFrameBytes: 64 * 1024})
			_, err := analyzer.Analyze(context.Background(), analysis.AnalyzeRequest{
				ProjectRoot: t.TempDir(),
				Selection:   analysis.AnalyzerSelection{AnalyzerID: testFixtureManifest, Mode: "explicit-id"},
				Options:     analysis.EffectiveOptions{Values: map[string]any{}, Fingerprint: "options"},
			})
			if got := analysis.ErrorCodeOf(err); got != analysis.ErrAnalyzerFailed {
				t.Fatalf("error code = %q, want analyzer_failed; err=%v", got, err)
			}
		})
	}

	analyzer := newProcessFixtureAnalyzer(t, "stderr", Config{MaxStderrBytes: 64})
	_, err := analyzer.Analyze(context.Background(), analysis.AnalyzeRequest{
		ProjectRoot: t.TempDir(),
		Selection:   analysis.AnalyzerSelection{AnalyzerID: testFixtureManifest, Mode: "explicit-id"},
		Options:     analysis.EffectiveOptions{Values: map[string]any{}, Fingerprint: "options"},
	})
	var hostErr *analysis.HostError
	if !errors.As(err, &hostErr) {
		t.Fatalf("stderr error = %v", err)
	}
	encoded, marshalErr := json.Marshal(hostErr.Details)
	if marshalErr != nil {
		t.Fatalf("marshal details: %v", marshalErr)
	}
	if len(encoded) > 2000 || !strings.Contains(string(encoded), "stderr-context") || !strings.Contains(string(encoded), "stderr truncated") {
		t.Fatalf("bounded stderr details = %s", encoded)
	}
}

func TestProcessAnalyzerCancellationAndTimeoutTerminateChild(t *testing.T) {
	analyzer := newProcessFixtureAnalyzer(t, "cancel", Config{})
	ctx, cancel := context.WithCancel(context.Background())
	resultCh := make(chan error, 1)
	started := time.Now()
	go func() {
		_, err := analyzer.Analyze(ctx, analysis.AnalyzeRequest{
			ProjectRoot: t.TempDir(),
			Selection:   analysis.AnalyzerSelection{AnalyzerID: testFixtureManifest, Mode: "explicit-id"},
			Options:     analysis.EffectiveOptions{Values: map[string]any{}, Fingerprint: "options"},
		})
		resultCh <- err
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()
	err := <-resultCh
	if got := analysis.ErrorCodeOf(err); got != analysis.ErrCancelled {
		t.Fatalf("cancel error code = %q, want cancelled; err=%v", got, err)
	}
	if elapsed := time.Since(started); elapsed > cleanupWaitTimeout {
		t.Fatalf("cancellation cleanup took %s", elapsed)
	}

	analyzer = newProcessFixtureAnalyzer(t, "delay", Config{OperationTimeout: 50 * time.Millisecond})
	started = time.Now()
	_, err = analyzer.Analyze(context.Background(), analysis.AnalyzeRequest{
		ProjectRoot: t.TempDir(),
		Selection:   analysis.AnalyzerSelection{AnalyzerID: testFixtureManifest, Mode: "explicit-id"},
		Options:     analysis.EffectiveOptions{Values: map[string]any{}, Fingerprint: "options"},
	})
	if got := analysis.ErrorCodeOf(err); got != analysis.ErrAnalyzerFailed {
		t.Fatalf("timeout error code = %q, want analyzer_failed; err=%v", got, err)
	}
	if elapsed := time.Since(started); elapsed > cleanupWaitTimeout {
		t.Fatalf("timeout cleanup took %s", elapsed)
	}
}

func TestProcessAnalyzerRunsThroughCommonHostValidation(t *testing.T) {
	analyzer := newProcessFixtureAnalyzer(t, "valid", Config{})
	registry := analysis.NewRegistry()
	if err := registry.Register(analyzer); err != nil {
		t.Fatalf("register: %v", err)
	}
	root := t.TempDir()
	host := analysis.NewHost(registry)
	result, err := host.Run(context.Background(), analysis.RunRequest{
		ProjectRoot: root,
		AnalyzerID:  testFixtureManifest,
		CLIOptions:  map[string]any{"include_tests": true},
	})
	if err != nil {
		t.Fatalf("host run: %v", err)
	}
	if result.Analyzer.ID != testFixtureManifest || result.Status != analysis.StatusComplete || result.OptionsFingerprint == "" {
		t.Fatalf("host result = %#v", result)
	}
}

func TestLoadDescriptorResolvesRelativeWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	working := filepath.Join(root, "runtime")
	if err := os.Mkdir(working, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	descriptor := processprotocol.Descriptor{
		SchemaVersion:    processprotocol.DescriptorSchemaVersion,
		Manifest:         processFixtureManifest(),
		Command:          os.Args[0],
		Args:             []string{"-test.run", "^TestProcessAnalyzerFixtureProcess$", "--"},
		WorkingDirectory: "runtime",
	}
	data, err := json.Marshal(descriptor)
	if err != nil {
		t.Fatalf("marshal descriptor: %v", err)
	}
	path := filepath.Join(root, "plugin.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write descriptor: %v", err)
	}
	analyzer, err := LoadDescriptor(path)
	if err != nil {
		t.Fatalf("load descriptor: %v", err)
	}
	if analyzer.Descriptor().WorkingDirectory != "runtime" {
		t.Fatalf("descriptor = %#v", analyzer.Descriptor())
	}
}
