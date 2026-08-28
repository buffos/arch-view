package analysis

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
)

func TestHostHonorsExplicitSelectionAndOptionPrecedence(t *testing.T) {
	root := t.TempDir()
	analyzer := &fakeAnalyzer{manifest: validManifest("org.example.go", "go")}
	var received AnalyzeRequest
	analyzer.analyze = func(ctx context.Context, request AnalyzeRequest) (AnalysisResult, error) {
		received = request
		return validResult(analyzer.manifest), nil
	}
	registry := NewRegistry()
	if err := registry.Register(analyzer); err != nil {
		t.Fatalf("register: %v", err)
	}
	result, err := NewHost(registry).Run(context.Background(), RunRequest{
		ProjectRoot:    root,
		Language:       "go",
		ProjectOptions: map[string]any{"include_tests": false},
		CLIOptions:     map[string]any{"include_tests": true},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if received.Selection.Mode != "explicit-language" {
		t.Fatalf("selection mode = %q, want explicit-language", received.Selection.Mode)
	}
	if got := received.Options.Values["include_tests"]; got != true {
		t.Fatalf("include_tests = %#v, want true", got)
	}
	if result.OptionsFingerprint == "" || result.RunID == "" {
		t.Fatalf("host did not record options/run identity: %#v", result)
	}
}

func TestHostAutoDetectionChoosesUniqueHighestConfidence(t *testing.T) {
	root := t.TempDir()
	low := &fakeAnalyzer{
		manifest: validManifest("org.example.low", "low"),
		detect: func(context.Context, DetectRequest) (DetectionCandidate, error) {
			return DetectionCandidate{AnalyzerID: "org.example.low", Confidence: 0.4, Reason: "low"}, nil
		},
	}
	high := &fakeAnalyzer{
		manifest: validManifest("org.example.high", "high"),
		detect: func(context.Context, DetectRequest) (DetectionCandidate, error) {
			return DetectionCandidate{AnalyzerID: "org.example.high", Confidence: 0.9, Reason: "high"}, nil
		},
	}
	registry := NewRegistry()
	for _, analyzer := range []*fakeAnalyzer{low, high} {
		if err := registry.Register(analyzer); err != nil {
			t.Fatalf("register: %v", err)
		}
	}
	result, err := NewHost(registry).Run(context.Background(), RunRequest{ProjectRoot: root})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Analyzer.ID != "org.example.high" {
		t.Fatalf("selected analyzer = %q, want high", result.Analyzer.ID)
	}
}

func TestHostRejectsTiedAutoDetection(t *testing.T) {
	root := t.TempDir()
	registry := NewRegistry()
	for _, id := range []string{"org.example.alpha", "org.example.beta"} {
		analyzer := &fakeAnalyzer{
			manifest: validManifest(id, id[stringsLastDot(id):]),
			detect: func(ctx context.Context, request DetectRequest) (DetectionCandidate, error) {
				return DetectionCandidate{AnalyzerID: id, Confidence: 1, Reason: "tie"}, nil
			},
		}
		if err := registry.Register(analyzer); err != nil {
			t.Fatalf("register: %v", err)
		}
	}
	_, err := NewHost(registry).Run(context.Background(), RunRequest{ProjectRoot: root})
	if ErrorCodeOf(err) != ErrAmbiguousAnalyzer {
		t.Fatalf("error code = %q, want %q", ErrorCodeOf(err), ErrAmbiguousAnalyzer)
	}
}

func TestHostContainsAnalyzerPanicsAndFailures(t *testing.T) {
	root := t.TempDir()
	tests := []struct {
		name string
		fn   func(context.Context, AnalyzeRequest) (AnalysisResult, error)
		want ErrorCode
	}{
		{
			name: "panic",
			fn: func(context.Context, AnalyzeRequest) (AnalysisResult, error) {
				panic("boom")
			},
			want: ErrAnalyzerFailed,
		},
		{
			name: "failure",
			fn: func(context.Context, AnalyzeRequest) (AnalysisResult, error) {
				return AnalysisResult{}, errors.New("failed")
			},
			want: ErrAnalyzerFailed,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			analyzer := &fakeAnalyzer{manifest: validManifest("org.example."+test.name, test.name), analyze: test.fn}
			registry := NewRegistry()
			if err := registry.Register(analyzer); err != nil {
				t.Fatalf("register: %v", err)
			}
			_, err := NewHost(registry).Run(context.Background(), RunRequest{ProjectRoot: root, AnalyzerID: analyzer.manifest.ID})
			if ErrorCodeOf(err) != test.want {
				t.Fatalf("error code = %q, want %q", ErrorCodeOf(err), test.want)
			}
		})
	}
}

func TestHostConvertsArbitraryPanicValuesToSerializableDetails(t *testing.T) {
	root := t.TempDir()
	analyzer := &fakeAnalyzer{manifest: validManifest("org.example.unserializable-panic", "panic")}
	analyzer.analyze = func(context.Context, AnalyzeRequest) (AnalysisResult, error) {
		panic(make(chan int))
	}
	registry := NewRegistry()
	if err := registry.Register(analyzer); err != nil {
		t.Fatalf("register: %v", err)
	}
	_, err := NewHost(registry).Run(context.Background(), RunRequest{ProjectRoot: root, AnalyzerID: analyzer.manifest.ID})
	if ErrorCodeOf(err) != ErrAnalyzerFailed {
		t.Fatalf("error code = %q, want %q", ErrorCodeOf(err), ErrAnalyzerFailed)
	}
	if _, marshalErr := MarshalError(err); marshalErr != nil {
		t.Fatalf("marshal panic error: %v", marshalErr)
	}
}

func TestHostCancellationCannotBecomeCompletion(t *testing.T) {
	root := t.TempDir()
	started := make(chan struct{})
	analyzer := &fakeAnalyzer{manifest: validManifest("org.example.cancel", "cancel")}
	analyzer.analyze = func(ctx context.Context, request AnalyzeRequest) (AnalysisResult, error) {
		close(started)
		<-ctx.Done()
		return validResult(analyzer.manifest), nil
	}
	registry := NewRegistry()
	if err := registry.Register(analyzer); err != nil {
		t.Fatalf("register: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	resultCh := make(chan error, 1)
	go func() {
		_, err := NewHost(registry).Run(ctx, RunRequest{ProjectRoot: root, AnalyzerID: analyzer.manifest.ID})
		resultCh <- err
	}()
	<-started
	cancel()
	if err := <-resultCh; ErrorCodeOf(err) != ErrCancelled {
		t.Fatalf("error code = %q, want %q", ErrorCodeOf(err), ErrCancelled)
	}
}

func TestHostRejectsInvalidResultReferences(t *testing.T) {
	root := t.TempDir()
	analyzer := &fakeAnalyzer{manifest: validManifest("org.example.invalid", "invalid")}
	analyzer.analyze = func(ctx context.Context, request AnalyzeRequest) (AnalysisResult, error) {
		result := validResult(analyzer.manifest)
		result.Modules = []ModuleObservation{{ID: "a"}}
		result.Relationships = []RelationshipObservation{{
			ID:           "a-to-missing",
			Type:         "depends_on",
			FromModuleID: "a",
			ToModuleID:   "missing",
		}}
		return result, nil
	}
	registry := NewRegistry()
	if err := registry.Register(analyzer); err != nil {
		t.Fatalf("register: %v", err)
	}
	_, err := NewHost(registry).Run(context.Background(), RunRequest{ProjectRoot: root, AnalyzerID: analyzer.manifest.ID})
	if ErrorCodeOf(err) != ErrResultInvalid {
		t.Fatalf("error code = %q, want %q", ErrorCodeOf(err), ErrResultInvalid)
	}
}

func TestHostRejectsWindowsAbsoluteEvidencePathOnEveryPlatform(t *testing.T) {
	root := t.TempDir()
	analyzer := &fakeAnalyzer{manifest: validManifest("org.example.invalid-path", "invalid-path")}
	analyzer.analyze = func(context.Context, AnalyzeRequest) (AnalysisResult, error) {
		result := validResult(analyzer.manifest)
		result.SourceReferences = []SourceReference{{ID: "source", Path: `C:\outside.go`, Kind: "file"}}
		return result, nil
	}
	registry := NewRegistry()
	if err := registry.Register(analyzer); err != nil {
		t.Fatalf("register: %v", err)
	}
	_, err := NewHost(registry).Run(context.Background(), RunRequest{ProjectRoot: root, AnalyzerID: analyzer.manifest.ID})
	if ErrorCodeOf(err) != ErrResultInvalid {
		t.Fatalf("error code = %q, want %q", ErrorCodeOf(err), ErrResultInvalid)
	}
}

func TestHostRejectsUnserializableResultMetadata(t *testing.T) {
	root := t.TempDir()
	analyzer := &fakeAnalyzer{manifest: validManifest("org.example.unserializable-result", "unserializable")}
	analyzer.analyze = func(context.Context, AnalyzeRequest) (AnalysisResult, error) {
		result := validResult(analyzer.manifest)
		result.Modules = []ModuleObservation{{ID: "module", Metadata: map[string]any{"unsupported": func() {}}}}
		return result, nil
	}
	registry := NewRegistry()
	if err := registry.Register(analyzer); err != nil {
		t.Fatalf("register: %v", err)
	}
	_, err := NewHost(registry).Run(context.Background(), RunRequest{ProjectRoot: root, AnalyzerID: analyzer.manifest.ID})
	if ErrorCodeOf(err) != ErrResultInvalid {
		t.Fatalf("error code = %q, want %q", ErrorCodeOf(err), ErrResultInvalid)
	}
}

func TestMarshalErrorFallsBackWhenDetailsAreNotSerializable(t *testing.T) {
	err := NewHostError(ErrAnalyzerFailed, "failed", map[string]any{"unsupported": make(chan int)})
	data, marshalErr := MarshalError(err)
	if marshalErr != nil {
		t.Fatalf("marshal error: %v", marshalErr)
	}
	var decoded struct {
		Error HostError `json:"error"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode fallback error: %v; data=%s", err, string(data))
	}
	if decoded.Error.Code != ErrAnalyzerFailed {
		t.Fatalf("fallback code = %q, want %q", decoded.Error.Code, ErrAnalyzerFailed)
	}
}

func TestHostRejectsMismatchedResultOptionsFingerprint(t *testing.T) {
	root := t.TempDir()
	analyzer := &fakeAnalyzer{manifest: validManifest("org.example.fingerprint", "fingerprint")}
	analyzer.analyze = func(context.Context, AnalyzeRequest) (AnalysisResult, error) {
		result := validResult(analyzer.manifest)
		result.OptionsFingerprint = "not-the-host-fingerprint"
		return result, nil
	}
	registry := NewRegistry()
	if err := registry.Register(analyzer); err != nil {
		t.Fatalf("register: %v", err)
	}
	_, err := NewHost(registry).Run(context.Background(), RunRequest{ProjectRoot: root, AnalyzerID: analyzer.manifest.ID})
	if ErrorCodeOf(err) != ErrResultInvalid {
		t.Fatalf("error code = %q, want %q", ErrorCodeOf(err), ErrResultInvalid)
	}
}

func TestExitCodesMatchContract(t *testing.T) {
	tests := []struct {
		code ErrorCode
		want int
	}{
		{ErrAmbiguousAnalyzer, 2},
		{ErrUnsupportedProject, 3},
		{ErrAnalyzerFailed, 4},
		{ErrCancelled, 130},
	}
	for _, test := range tests {
		if got := ExitCodeForError(NewHostError(test.code, "test", nil)); got != test.want {
			t.Errorf("%s exit code = %d, want %d", test.code, got, test.want)
		}
	}
}

func TestHostWithRuntimeRecordsValidatedProvenance(t *testing.T) {
	root := t.TempDir()
	analyzer := &fakeAnalyzer{manifest: validManifest("org.example.runtime", "runtime")}
	var selection AnalyzerSelection
	analyzer.analyze = func(_ context.Context, request AnalyzeRequest) (AnalysisResult, error) {
		selection = request.Selection
		return validResult(analyzer.manifest), nil
	}
	registry := NewRegistry()
	if err := registry.Register(analyzer); err != nil {
		t.Fatalf("register: %v", err)
	}
	base := NewHost(registry)
	runtimeHost, err := base.WithRuntime(RuntimeSelection{Mode: RuntimeModePackaged, Source: "application-index", Platform: "windows-amd64"})
	if err != nil {
		t.Fatalf("clone host with runtime: %v", err)
	}
	result, err := runtimeHost.Run(context.Background(), RunRequest{ProjectRoot: root, AnalyzerID: analyzer.manifest.ID})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if selection.RuntimeMode != RuntimeModePackaged || selection.RuntimeSource != "application-index" || selection.RuntimePlatform != "windows-amd64" {
		t.Fatalf("selection runtime = %#v", selection)
	}
	if result.Analyzer.RuntimeMode != RuntimeModePackaged || result.Analyzer.RuntimeSource != "application-index" || result.Analyzer.RuntimePlatform != "windows-amd64" {
		t.Fatalf("result runtime = %#v", result.Analyzer)
	}
	if base.Runtime() != (RuntimeSelection{}) {
		t.Fatalf("base runtime changed = %#v", base.Runtime())
	}
}

func TestHostRejectsIncompleteRuntimeProvenance(t *testing.T) {
	base := NewHost(NewRegistry())
	if _, err := base.WithRuntime(RuntimeSelection{Mode: RuntimeModePackaged}); ErrorCodeOf(err) != ErrInvalidRequest {
		t.Fatalf("incomplete runtime error code = %q, want %q", ErrorCodeOf(err), ErrInvalidRequest)
	}
	if _, err := base.WithRuntime(RuntimeSelection{Source: "application-index"}); ErrorCodeOf(err) != ErrInvalidRequest {
		t.Fatalf("mode-less runtime error code = %q, want %q", ErrorCodeOf(err), ErrInvalidRequest)
	}
}

func TestNormalizeProjectRootRejectsFiles(t *testing.T) {
	file := t.TempDir() + string(os.PathSeparator) + "file"
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	_, err := normalizeProjectRoot(file)
	if ErrorCodeOf(err) != ErrInvalidRequest {
		t.Fatalf("error code = %q, want %q", ErrorCodeOf(err), ErrInvalidRequest)
	}
}

func stringsLastDot(value string) int {
	for index := len(value) - 1; index >= 0; index-- {
		if value[index] == '.' {
			return index + 1
		}
	}
	return 0
}
