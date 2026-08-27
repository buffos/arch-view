package processprotocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
)

func TestFrameCodecRoundTripsOneJSONObjectPerLine(t *testing.T) {
	manifest := fixtureManifest()
	result := analysis.AnalysisResult{Status: analysis.StatusComplete}
	frames := []Frame{
		{Type: FrameHello, Protocol: ProtocolVersion, Manifest: &manifest},
		{Type: FrameDetect, RequestID: "req-1", ProjectRoot: "."},
		{Type: FrameAnalyze, RequestID: "req-1", ProjectRoot: ".", Selection: &analysis.AnalyzerSelection{AnalyzerID: manifest.ID, Mode: "explicit-id"}, Options: &analysis.EffectiveOptions{Values: map[string]any{}, Fingerprint: "fixture-options"}},
		{Type: FrameCancel, RequestID: "req-1", Reason: "caller cancelled"},
		{Type: FrameCandidate, RequestID: "req-1", Candidate: &analysis.DetectionCandidate{AnalyzerID: manifest.ID, Confidence: 1, Reason: "marker"}},
		{Type: FrameResult, RequestID: "req-1", Result: &result},
		{Type: FrameDiagnostic, RequestID: "req-1", Diagnostic: &analysis.Diagnostic{Code: "fixture.info", Severity: "info", Message: "hello", Recoverable: true}},
		{Type: FrameDone, RequestID: "req-1", Status: analysis.StatusComplete},
		{Type: FrameFatal, RequestID: "req-1", Code: "fixture_failure", Message: "failed", Details: map[string]any{"retryable": false}},
	}

	var encoded bytes.Buffer
	encoder := NewEncoder(&encoded)
	for _, frame := range frames {
		if err := encoder.WriteFrame(frame); err != nil {
			t.Fatalf("write %s: %v", frame.Type, err)
		}
	}
	if got := bytes.Count(encoded.Bytes(), []byte{'\n'}); got != len(frames) {
		t.Fatalf("newline count = %d, want %d", got, len(frames))
	}

	decoder := NewDecoder(&encoded)
	for index, want := range frames {
		got, err := decoder.ReadFrame()
		if err != nil {
			t.Fatalf("read frame %d: %v", index, err)
		}
		if got.Type != want.Type || got.RequestID != want.RequestID {
			t.Fatalf("frame %d = %#v, want type/request %s/%s", index, got, want.Type, want.RequestID)
		}
	}
	if _, err := decoder.ReadFrame(); !errors.Is(err, io.EOF) {
		t.Fatalf("end of stream error = %v, want EOF", err)
	}
}

func TestDecoderRejectsMalformedUnknownAndIncompleteFrames(t *testing.T) {
	manifest := fixtureManifest()
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	validHello := `{"type":"hello","protocol":"arch-view.analyzer/v1","manifest":` + string(manifestJSON) + `}`
	tests := []struct {
		name string
		line string
		want error
	}{
		{name: "malformed json", line: "{not-json\n", want: ErrMalformedJSON},
		{name: "trailing json", line: `{"type":"cancel","request_id":"req"}{"type":"cancel","request_id":"req"}` + "\n", want: ErrMalformedJSON},
		{name: "array instead of object", line: "[]\n", want: ErrMalformedJSON},
		{name: "unknown type", line: `{"type":"mystery","request_id":"req"}` + "\n", want: ErrUnknownFrameType},
		{name: "missing type", line: `{"request_id":"req"}` + "\n", want: ErrMissingField},
		{name: "empty request id", line: `{"type":"cancel","request_id":" "}` + "\n", want: ErrInvalidRequestID},
		{name: "missing project root", line: `{"type":"detect","request_id":"req"}` + "\n", want: ErrMissingField},
		{name: "missing analyze selection", line: `{"type":"analyze","request_id":"req","project_root":"."}` + "\n", want: ErrMissingField},
		{name: "missing candidate payload", line: `{"type":"candidate","request_id":"req"}` + "\n", want: ErrMissingField},
		{name: "incomplete candidate payload", line: `{"type":"candidate","request_id":"req","candidate":{}}` + "\n", want: ErrInvalidPayload},
		{name: "invalid done status", line: `{"type":"done","request_id":"req","status":"unknown"}` + "\n", want: ErrInvalidPayload},
		{name: "missing fatal message", line: `{"type":"fatal","request_id":"req","code":"failed"}` + "\n", want: ErrMissingField},
		{name: "hello request id", line: validHello[:len(validHello)-1] + `,"request_id":"req"}` + "\n", want: ErrInvalidRequestID},
		{name: "unsupported hello", line: strings.Replace(validHello, "arch-view.analyzer/v1", "arch-view.analyzer/v9", 1) + "\n", want: ErrUnsupportedProtocol},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ReadFrame(strings.NewReader(test.line))
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestFrameCodecEnforcesMaximumLineSize(t *testing.T) {
	line := `{"type":"cancel","request_id":"req"}` + "\n"
	if _, err := ReadFrameWithLimit(strings.NewReader(line), len(line)-1); !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("oversized read error = %v, want frame-too-large", err)
	}

	frame := Frame{Type: FrameDiagnostic, RequestID: "req", Diagnostic: &analysis.Diagnostic{Code: "large", Severity: "warning", Message: strings.Repeat("x", 64), Recoverable: true}}
	if err := WriteFrameWithLimit(io.Discard, frame, 32); !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("oversized write error = %v, want frame-too-large", err)
	}
}

func TestDescriptorDecodeValidatesExplicitArgvShape(t *testing.T) {
	valid := `{"schema_version":"arch-view.plugin/v1","manifest":{"id":"org.example.external-fixture","version":"1.0.0","language":"fixture","api_version":"arch-view.analyzer/v1","detection_markers":[{"kind":"file","value":"fixture.marker","weight":1}],"capabilities":["detect"],"options":[]},"command":"python","args":["analyzer.py"],"working_directory":"."}`
	descriptor, err := DecodeDescriptor([]byte(valid))
	if err != nil {
		t.Fatalf("decode descriptor: %v", err)
	}
	if descriptor.Command != "python" || len(descriptor.Args) != 1 || descriptor.WorkingDirectory != "." {
		t.Fatalf("descriptor = %#v", descriptor)
	}

	tests := []struct {
		name string
		data string
	}{
		{name: "wrong schema", data: strings.Replace(valid, "arch-view.plugin/v1", "arch-view.plugin/v2", 1)},
		{name: "empty command", data: strings.Replace(valid, `"command":"python"`, `"command":"  "`, 1)},
		{name: "unknown field", data: strings.TrimSuffix(valid, "}") + `,"callback":"run"}`},
		{name: "trailing json", data: valid + "{}"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := DecodeDescriptor([]byte(test.data)); !errors.Is(err, ErrDescriptorInvalid) {
				t.Fatalf("error = %v, want descriptor-invalid", err)
			}
		})
	}
}

func TestSessionValidatorAcceptsDetectAndAnalyzeLifecycles(t *testing.T) {
	manifest := fixtureManifest()
	tests := []struct {
		name      string
		operation FrameType
	}{
		{name: "detect", operation: FrameDetect},
		{name: "analyze", operation: FrameAnalyze},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session, err := NewSessionValidator(test.operation, "req-1", manifest)
			if err != nil {
				t.Fatalf("new session: %v", err)
			}
			hello := manifest
			if err := session.AcceptPluginFrame(Frame{Type: FrameHello, Protocol: ProtocolVersion, Manifest: &hello}); err != nil {
				t.Fatalf("hello: %v", err)
			}
			if err := session.AcceptHostFrame(fixtureRequest(test.operation)); err != nil {
				t.Fatalf("request: %v", err)
			}
			if test.operation == FrameDetect {
				if err := session.AcceptPluginFrame(Frame{Type: FrameCandidate, RequestID: "req-1", Candidate: &analysis.DetectionCandidate{AnalyzerID: manifest.ID, Confidence: 1, Reason: "marker"}}); err != nil {
					t.Fatalf("candidate: %v", err)
				}
			} else {
				if err := session.AcceptPluginFrame(Frame{Type: FrameDiagnostic, RequestID: "req-1", Diagnostic: &analysis.Diagnostic{Code: "warning", Severity: "warning", Message: "streamed", Recoverable: true}}); err != nil {
					t.Fatalf("diagnostic: %v", err)
				}
				result := analysis.AnalysisResult{RunID: "run-1", Status: analysis.StatusPartial, Analyzer: analysis.AnalyzerInfo{ID: manifest.ID, Version: manifest.Version, Language: manifest.Language, APIVersion: manifest.APIVersion}, Project: analysis.ProjectInfo{RootLabel: "fixture", Boundary: "fixture.marker"}}
				if err := session.AcceptPluginFrame(Frame{Type: FrameResult, RequestID: "req-1", Result: &result}); err != nil {
					t.Fatalf("result: %v", err)
				}
				if err := session.AcceptPluginFrame(Frame{Type: FrameDone, RequestID: "req-1", Status: analysis.StatusPartial}); err != nil {
					t.Fatalf("done: %v", err)
				}
				if len(session.Diagnostics()) != 1 {
					t.Fatalf("diagnostics = %#v, want one", session.Diagnostics())
				}
				if _, ok := session.Result(); !ok {
					t.Fatal("session did not retain result")
				}
				if !session.Complete() {
					t.Fatal("analyze session is not complete")
				}
				return
			}
			if err := session.AcceptPluginFrame(Frame{Type: FrameDone, RequestID: "req-1", Status: analysis.StatusComplete}); err != nil {
				t.Fatalf("done: %v", err)
			}
			if candidate, ok := session.Candidate(); !ok || candidate.AnalyzerID != manifest.ID {
				t.Fatalf("candidate = %#v, ok=%t", candidate, ok)
			}
			if !session.Complete() {
				t.Fatal("detect session is not complete")
			}
		})
	}
}

func TestSessionValidatorRejectsProtocolStateViolations(t *testing.T) {
	manifest := fixtureManifest()
	preHello, err := NewSessionValidator(FrameDetect, "req-1", manifest)
	if err != nil {
		t.Fatalf("new pre-hello session: %v", err)
	}
	if err := preHello.AcceptHostFrame(fixtureRequest(FrameDetect)); !errors.Is(err, ErrProtocolState) {
		t.Fatalf("pre-hello request error = %v, want protocol state", err)
	}
	newSession := func(t *testing.T, operation FrameType) *SessionValidator {
		t.Helper()
		session, err := NewSessionValidator(operation, "req-1", manifest)
		if err != nil {
			t.Fatalf("new session: %v", err)
		}
		if err := session.AcceptPluginFrame(Frame{Type: FrameHello, Protocol: ProtocolVersion, Manifest: &manifest}); err != nil {
			t.Fatalf("hello: %v", err)
		}
		if err := session.AcceptHostFrame(fixtureRequest(operation)); err != nil {
			t.Fatalf("request: %v", err)
		}
		return session
	}

	tests := []struct {
		name string
		want error
		call func(*SessionValidator) error
	}{
		{name: "wrong request id", want: ErrRequestIDMismatch, call: func(session *SessionValidator) error {
			return session.AcceptPluginFrame(Frame{Type: FrameCandidate, RequestID: "wrong", Candidate: &analysis.DetectionCandidate{AnalyzerID: fixtureManifest().ID, Confidence: 1, Reason: "marker"}})
		}},
		{name: "duplicate candidate", want: ErrDuplicatePayload, call: func(session *SessionValidator) error {
			candidate := Frame{Type: FrameCandidate, RequestID: "req-1", Candidate: &analysis.DetectionCandidate{AnalyzerID: fixtureManifest().ID, Confidence: 1, Reason: "marker"}}
			if err := session.AcceptPluginFrame(candidate); err != nil {
				return err
			}
			return session.AcceptPluginFrame(candidate)
		}},
		{name: "duplicate terminal", want: ErrDuplicateTerminal, call: func(session *SessionValidator) error {
			candidate := Frame{Type: FrameCandidate, RequestID: "req-1", Candidate: &analysis.DetectionCandidate{AnalyzerID: fixtureManifest().ID, Confidence: 1, Reason: "marker"}}
			if err := session.AcceptPluginFrame(candidate); err != nil {
				return err
			}
			terminal := Frame{Type: FrameDone, RequestID: "req-1", Status: analysis.StatusComplete}
			if err := session.AcceptPluginFrame(terminal); err != nil {
				return err
			}
			return session.AcceptPluginFrame(terminal)
		}},
		{name: "late output", want: ErrLateFrame, call: func(session *SessionValidator) error {
			candidate := Frame{Type: FrameCandidate, RequestID: "req-1", Candidate: &analysis.DetectionCandidate{AnalyzerID: fixtureManifest().ID, Confidence: 1, Reason: "marker"}}
			if err := session.AcceptPluginFrame(candidate); err != nil {
				return err
			}
			if err := session.AcceptPluginFrame(Frame{Type: FrameDone, RequestID: "req-1", Status: analysis.StatusComplete}); err != nil {
				return err
			}
			return session.AcceptPluginFrame(Frame{Type: FrameDiagnostic, RequestID: "req-1", Diagnostic: &analysis.Diagnostic{Code: "late", Severity: "warning", Message: "late", Recoverable: true}})
		}},
		{name: "done before payload", want: ErrProtocolState, call: func(session *SessionValidator) error {
			return session.AcceptPluginFrame(Frame{Type: FrameDone, RequestID: "req-1", Status: analysis.StatusComplete})
		}},
		{name: "fatal after payload", want: ErrProtocolState, call: func(session *SessionValidator) error {
			candidate := Frame{Type: FrameCandidate, RequestID: "req-1", Candidate: &analysis.DetectionCandidate{AnalyzerID: fixtureManifest().ID, Confidence: 1, Reason: "marker"}}
			if err := session.AcceptPluginFrame(candidate); err != nil {
				return err
			}
			return session.AcceptPluginFrame(Frame{Type: FrameFatal, RequestID: "req-1", Code: "failed", Message: "too late"})
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session := newSession(t, FrameDetect)
			if err := test.call(session); !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}

	cancelled := newSession(t, FrameAnalyze)
	if err := cancelled.AcceptHostFrame(Frame{Type: FrameCancel, RequestID: "req-1", Reason: "cancelled"}); err != nil {
		t.Fatalf("cancel request: %v", err)
	}
	result := analysis.AnalysisResult{RunID: "late", Status: analysis.StatusComplete, Analyzer: analysis.AnalyzerInfo{ID: manifest.ID, Version: manifest.Version, Language: manifest.Language, APIVersion: manifest.APIVersion}, Project: analysis.ProjectInfo{RootLabel: "fixture", Boundary: "fixture.marker"}}
	if err := cancelled.AcceptPluginFrame(Frame{Type: FrameResult, RequestID: "req-1", Result: &result}); !errors.Is(err, ErrProtocolState) {
		t.Fatalf("result after cancel error = %v, want protocol state", err)
	}

	mismatch := newSession(t, FrameDetect)
	other := fixtureManifest()
	other.Version = "2.0.0"
	if err := mismatch.AcceptPluginFrame(Frame{Type: FrameHello, Protocol: ProtocolVersion, Manifest: &other}); !errors.Is(err, ErrLateFrame) && !errors.Is(err, ErrProtocolState) {
		// The second hello is intentionally rejected as state, while the
		// dedicated mismatch check below covers the first-frame path.
		t.Fatalf("second hello error = %v", err)
	}

	first, err := NewSessionValidator(FrameDetect, "req-1", manifest)
	if err != nil {
		t.Fatalf("new mismatch session: %v", err)
	}
	if err := first.AcceptPluginFrame(Frame{Type: FrameHello, Protocol: ProtocolVersion, Manifest: &other}); !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("manifest mismatch error = %v, want invalid payload", err)
	}
}

func TestSessionValidatorLeavesCanonicalResultValidationToAnalysisPackage(t *testing.T) {
	manifest := fixtureManifest()
	session, err := NewSessionValidator(FrameAnalyze, "req-1", manifest)
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	if err := session.AcceptPluginFrame(Frame{Type: FrameHello, Protocol: ProtocolVersion, Manifest: &manifest}); err != nil {
		t.Fatalf("hello: %v", err)
	}
	if err := session.AcceptHostFrame(fixtureRequest(FrameAnalyze)); err != nil {
		t.Fatalf("request: %v", err)
	}
	invalid := analysis.AnalysisResult{}
	if err := session.AcceptPluginFrame(Frame{Type: FrameResult, RequestID: "req-1", Result: &invalid}); err != nil {
		t.Fatalf("protocol rejected result envelope: %v", err)
	}
	if err := analysis.ValidateAnalysisResult(invalid, manifest, t.TempDir()); analysis.ErrorCodeOf(err) != analysis.ErrResultInvalid {
		t.Fatalf("canonical validation error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrResultInvalid)
	}
}

func TestMergeDiagnosticsDeduplicatesStreamedAndResultDiagnostics(t *testing.T) {
	manifest := fixtureManifest()
	warning := analysis.Diagnostic{Code: "fixture.warning", Severity: "warning", Message: "same", Recoverable: true}
	result := analysis.AnalysisResult{
		Diagnostics: []analysis.Diagnostic{warning},
		Analyzer:    analysis.AnalyzerInfo{ID: manifest.ID, Version: manifest.Version, Language: manifest.Language, APIVersion: manifest.APIVersion},
		Project:     analysis.ProjectInfo{RootLabel: "fixture", Boundary: "fixture.marker"},
		RunID:       "run-1",
		Status:      analysis.StatusComplete,
	}
	merged := MergeDiagnostics(result, []analysis.Diagnostic{
		warning,
		{Code: "fixture.info", Severity: "info", Message: "different", Recoverable: true},
	})
	if len(merged.Diagnostics) != 2 {
		t.Fatalf("merged diagnostics = %#v, want two unique diagnostics", merged.Diagnostics)
	}
	if merged.Diagnostics[0].Code != "fixture.info" || merged.Diagnostics[1].Code != "fixture.warning" {
		t.Fatalf("merged diagnostics order = %#v, want deterministic code order", merged.Diagnostics)
	}
}

func TestExternalFixtureProducesIntentionalProtocolViolations(t *testing.T) {
	tests := []struct {
		name string
		want error
	}{
		{name: "malformed-json", want: ErrMalformedJSON},
		{name: "unknown-frame", want: ErrUnknownFrameType},
		{name: "oversized-line", want: ErrFrameTooLarge},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			extra := []string(nil)
			maxFrameBytes := 16 * 1024
			if test.name == "oversized-line" {
				extra = []string{"ARCH_VIEW_EXTERNAL_FIXTURE_OVERSIZE_BYTES=20000"}
			}
			run := runFixture(t, test.name, FrameDetect, maxFrameBytes, extra...)
			if !errors.Is(run.ReadErr, test.want) {
				t.Fatalf("read error = %v, want %v; frames=%#v; stderr=%s", run.ReadErr, test.want, run.Frames, run.Stderr)
			}
			if run.WaitErr != nil {
				t.Fatalf("fixture wait: %v; stderr=%s", run.WaitErr, run.Stderr)
			}
		})
	}

	wrongID := runFixture(t, "wrong-request-id", FrameDetect, 16*1024)
	assertFixtureSessionStart(t, wrongID, FrameDetect)
	wrongSession, err := NewSessionValidator(FrameDetect, "req-1", fixtureManifest())
	if err != nil {
		t.Fatalf("new wrong-id session: %v", err)
	}
	if err := wrongSession.AcceptPluginFrame(wrongID.Frames[0]); err != nil {
		t.Fatalf("wrong-id hello: %v", err)
	}
	if err := wrongSession.AcceptHostFrame(fixtureRequest(FrameDetect)); err != nil {
		t.Fatalf("wrong-id request: %v", err)
	}
	if err := wrongSession.AcceptPluginFrame(wrongID.Frames[1]); !errors.Is(err, ErrRequestIDMismatch) {
		t.Fatalf("wrong-id output error = %v, want request-id mismatch", err)
	}

	duplicate := runFixture(t, "duplicate-terminal", FrameDetect, 16*1024)
	if duplicate.ReadErr != nil || duplicate.WaitErr != nil {
		t.Fatalf("duplicate fixture errors: read=%v wait=%v stderr=%s", duplicate.ReadErr, duplicate.WaitErr, duplicate.Stderr)
	}
	duplicateSession := newFixtureSessionFromFrames(t, duplicate, FrameDetect)
	if err := duplicateSession.AcceptPluginFrame(duplicate.Frames[1]); err != nil {
		t.Fatalf("duplicate candidate: %v", err)
	}
	if err := duplicateSession.AcceptPluginFrame(duplicate.Frames[2]); err != nil {
		t.Fatalf("duplicate first terminal: %v", err)
	}
	if err := duplicateSession.AcceptPluginFrame(duplicate.Frames[3]); !errors.Is(err, ErrDuplicateTerminal) {
		t.Fatalf("duplicate terminal error = %v, want duplicate terminal", err)
	}

	late := runFixture(t, "late-output", FrameDetect, 16*1024)
	if late.ReadErr != nil || late.WaitErr != nil {
		t.Fatalf("late fixture errors: read=%v wait=%v stderr=%s", late.ReadErr, late.WaitErr, late.Stderr)
	}
	lateSession := newFixtureSessionFromFrames(t, late, FrameDetect)
	if err := lateSession.AcceptPluginFrame(late.Frames[1]); err != nil {
		t.Fatalf("late candidate: %v", err)
	}
	if err := lateSession.AcceptPluginFrame(late.Frames[2]); err != nil {
		t.Fatalf("late terminal: %v", err)
	}
	if err := lateSession.AcceptPluginFrame(late.Frames[3]); !errors.Is(err, ErrLateFrame) {
		t.Fatalf("late output error = %v, want late frame", err)
	}

	mismatch := runFixture(t, "manifest-mismatch", FrameDetect, 16*1024)
	if mismatch.ReadErr != nil || mismatch.WaitErr != nil {
		t.Fatalf("mismatch fixture errors: read=%v wait=%v stderr=%s", mismatch.ReadErr, mismatch.WaitErr, mismatch.Stderr)
	}
	mismatchSession, err := NewSessionValidator(FrameDetect, "req-1", fixtureManifest())
	if err != nil {
		t.Fatalf("new mismatch fixture session: %v", err)
	}
	if err := mismatchSession.AcceptPluginFrame(mismatch.Frames[0]); !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("manifest mismatch error = %v, want invalid payload", err)
	}

	unsupported := runFixture(t, "unsupported-hello", FrameDetect, 16*1024)
	if !errors.Is(unsupported.ReadErr, ErrUnsupportedProtocol) {
		t.Fatalf("unsupported hello error = %v, want unsupported protocol", unsupported.ReadErr)
	}
	helloID := runFixture(t, "hello-request-id", FrameDetect, 16*1024)
	if !errors.Is(helloID.ReadErr, ErrInvalidRequestID) {
		t.Fatalf("hello request-id error = %v, want invalid request ID", helloID.ReadErr)
	}
}

func assertFixtureSessionStart(t *testing.T, run fixtureRun, operation FrameType) {
	t.Helper()
	if run.ReadErr != nil || run.WaitErr != nil {
		t.Fatalf("fixture start errors: read=%v wait=%v stderr=%s", run.ReadErr, run.WaitErr, run.Stderr)
	}
	if len(run.Frames) < 2 || run.Frames[0].Type != FrameHello {
		t.Fatalf("fixture frames = %#v", run.Frames)
	}
	if operation == FrameDetect && run.Frames[1].Type != FrameCandidate {
		t.Fatalf("fixture first payload = %s, want candidate", run.Frames[1].Type)
	}
}

func newFixtureSessionFromFrames(t *testing.T, run fixtureRun, operation FrameType) *SessionValidator {
	t.Helper()
	assertFixtureSessionStart(t, run, operation)
	session, err := NewSessionValidator(operation, "req-1", fixtureManifest())
	if err != nil {
		t.Fatalf("new fixture session: %v", err)
	}
	if err := session.AcceptPluginFrame(run.Frames[0]); err != nil {
		t.Fatalf("fixture hello: %v", err)
	}
	if err := session.AcceptHostFrame(fixtureRequest(operation)); err != nil {
		t.Fatalf("fixture request: %v", err)
	}
	return session
}

func TestSchemaArtifactsDeclareThePublishedFrameAndDescriptorSurface(t *testing.T) {
	protocol := readSchema(t, "docs", "architecture", "analyze-source", "plugin-runtime", "external-protocol-v1.schema.json")
	definitions, ok := protocol["$defs"].(map[string]any)
	if !ok {
		t.Fatal("protocol schema has no $defs")
	}
	for _, name := range []string{"hello", "detect", "analyze", "cancel", "candidate", "result", "diagnostic", "done", "fatal"} {
		if _, ok := definitions[name]; !ok {
			t.Fatalf("protocol schema missing %s definition", name)
		}
	}
	if protocol["oneOf"] == nil {
		t.Fatal("protocol schema has no frame oneOf")
	}

	descriptor := readSchema(t, "docs", "architecture", "analyze-source", "plugin-runtime", "external-plugin-descriptor-v1.schema.json")
	properties, ok := descriptor["properties"].(map[string]any)
	if !ok {
		t.Fatal("descriptor schema has no properties")
	}
	for _, name := range []string{"schema_version", "manifest", "command", "args", "working_directory"} {
		if _, ok := properties[name]; !ok {
			t.Fatalf("descriptor schema missing %s", name)
		}
	}
	if descriptor["additionalProperties"] != false {
		t.Fatal("descriptor schema must reject unrecognized callback/shell fields")
	}
}

func readSchema(t *testing.T, parts ...string) map[string]any {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test source")
	}
	path := filepath.Join(append([]string{filepath.Dir(sourceFile), "..", "..", ".."}, parts...)...)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema %s: %v", path, err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("parse schema %s: %v", path, err)
	}
	return schema
}
