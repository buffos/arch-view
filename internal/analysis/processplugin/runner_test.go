package processplugin

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/processprotocol"
)

func TestRunDetectEmitsHelloCandidateAndDone(t *testing.T) {
	analyzer := &runnerTestAnalyzer{}
	request := processprotocol.Frame{
		Type:        processprotocol.FrameDetect,
		RequestID:   "detect-1",
		ProjectRoot: "/fixture",
	}
	input := encodedFrame(t, request)
	var output bytes.Buffer
	var stderr bytes.Buffer

	if err := Run(context.Background(), analyzer, strings.NewReader(input), &output, &stderr); err != nil {
		t.Fatalf("run: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	frames := decodeFrames(t, output.Bytes())
	if got := frameTypes(frames); !reflect.DeepEqual(got, []processprotocol.FrameType{
		processprotocol.FrameHello,
		processprotocol.FrameCandidate,
		processprotocol.FrameDone,
	}) {
		t.Fatalf("frame types = %#v", got)
	}
	if frames[0].Manifest == nil || frames[0].Manifest.ID != analyzer.manifest().ID {
		t.Fatalf("hello manifest = %#v", frames[0].Manifest)
	}
	if frames[1].Candidate == nil || frames[1].Candidate.AnalyzerID != analyzer.manifest().ID {
		t.Fatalf("candidate = %#v", frames[1].Candidate)
	}
	if frames[2].Status != analysis.StatusComplete {
		t.Fatalf("done status = %q", frames[2].Status)
	}

	validateSession(t, analyzer.manifest(), request, frames)
}

func TestRunAnalyzeForwardsOptionsAndStreamsDiagnostics(t *testing.T) {
	analyzer := &runnerTestAnalyzer{}
	options := analysis.EffectiveOptions{
		Values: map[string]any{
			"include_tests": true,
			"labels":        []any{"one", "two"},
		},
		Sources:     map[string]string{"include_tests": "cli"},
		Fingerprint: "options-fingerprint",
	}
	selection := analysis.AnalyzerSelection{
		AnalyzerID:     analyzer.manifest().ID,
		Mode:           "explicit-id",
		Confidence:     1,
		MatchedMarkers: []string{"fixture.marker"},
		Reason:         "fixture",
	}
	request := processprotocol.Frame{
		Type:        processprotocol.FrameAnalyze,
		RequestID:   "analyze-1",
		ProjectRoot: "/fixture",
		Selection:   &selection,
		Options:     &options,
	}
	input := encodedFrame(t, request)
	var output bytes.Buffer
	var stderr bytes.Buffer

	if err := Run(context.Background(), analyzer, strings.NewReader(input), &output, &stderr); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !reflect.DeepEqual(analyzer.seenSelection, selection) {
		t.Fatalf("selection = %#v, want %#v", analyzer.seenSelection, selection)
	}
	wantOptions := options
	wantOptions.Values = map[string]any{
		"include_tests": true,
		"labels":        []string{"one", "two"},
	}
	if !reflect.DeepEqual(analyzer.seenOptions, wantOptions) {
		t.Fatalf("options = %#v, want %#v", analyzer.seenOptions, wantOptions)
	}
	frames := decodeFrames(t, output.Bytes())
	if got := frameTypes(frames); !reflect.DeepEqual(got, []processprotocol.FrameType{
		processprotocol.FrameHello,
		processprotocol.FrameDiagnostic,
		processprotocol.FrameResult,
		processprotocol.FrameDone,
	}) {
		t.Fatalf("frame types = %#v", got)
	}
	if frames[1].Diagnostic == nil || frames[1].Diagnostic.Code != "fixture.warning" {
		t.Fatalf("diagnostic frame = %#v", frames[1].Diagnostic)
	}
	if frames[2].Result == nil || frames[2].Result.OptionsFingerprint != options.Fingerprint {
		t.Fatalf("result frame = %#v", frames[2].Result)
	}
	if frames[3].Status != analysis.StatusComplete {
		t.Fatalf("done status = %q", frames[3].Status)
	}

	validateSession(t, analyzer.manifest(), request, frames)
}

func TestRunCancellationEmitsOnlyCancelledFatal(t *testing.T) {
	analyzer := &runnerTestAnalyzer{waitForCancellation: true}
	request := analyzeRequest(analyzer.manifest().ID, "cancel-1")
	reader, writer := io.Pipe()
	var output bytes.Buffer
	var stderr bytes.Buffer
	done := make(chan error, 1)
	go func() {
		done <- Run(context.Background(), analyzer, reader, &output, &stderr)
	}()

	if err := processprotocol.WriteFrame(writer, request); err != nil {
		t.Fatalf("write request: %v", err)
	}
	cancel := processprotocol.Frame{Type: processprotocol.FrameCancel, RequestID: request.RequestID, Reason: "test cancellation"}
	if err := processprotocol.WriteFrame(writer, cancel); err != nil {
		t.Fatalf("write cancel: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("run: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	frames := decodeFrames(t, output.Bytes())
	if got := frameTypes(frames); !reflect.DeepEqual(got, []processprotocol.FrameType{
		processprotocol.FrameHello,
		processprotocol.FrameFatal,
	}) {
		t.Fatalf("frame types = %#v", got)
	}
	if frames[1].Code != "cancelled" || !strings.Contains(frames[1].Message, "test cancellation") {
		t.Fatalf("fatal frame = %#v", frames[1])
	}
	if analyzer.cancellationObserved == nil {
		t.Fatal("analyzer did not observe cancellation")
	}

	validateSession(t, analyzer.manifest(), request, append(frames[:1], frames[1:]...))
	if err := writer.Close(); err != nil {
		t.Fatalf("close input: %v", err)
	}
}

func TestRunAnalyzerErrorEmitsFatal(t *testing.T) {
	analyzer := &runnerTestAnalyzer{analyzerError: analysis.NewHostError(analysis.ErrUnsupportedProject, "fixture is unsupported", nil)}
	request := analyzeRequest(analyzer.manifest().ID, "error-1")
	var output bytes.Buffer
	var stderr bytes.Buffer

	if err := Run(context.Background(), analyzer, strings.NewReader(encodedFrame(t, request)), &output, &stderr); err != nil {
		t.Fatalf("run: %v", err)
	}
	frames := decodeFrames(t, output.Bytes())
	if got := frameTypes(frames); !reflect.DeepEqual(got, []processprotocol.FrameType{
		processprotocol.FrameHello,
		processprotocol.FrameFatal,
	}) {
		t.Fatalf("frame types = %#v", got)
	}
	if frames[1].Code != string(analysis.ErrUnsupportedProject) {
		t.Fatalf("fatal code = %q", frames[1].Code)
	}
	if !strings.Contains(stderr.String(), "fixture is unsupported") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	validateSession(t, analyzer.manifest(), request, frames)
}

type runnerTestAnalyzer struct {
	seenSelection        analysis.AnalyzerSelection
	seenOptions          analysis.EffectiveOptions
	waitForCancellation  bool
	cancellationObserved chan struct{}
	analyzerError        error
}

func (a *runnerTestAnalyzer) manifest() analysis.Manifest {
	return analysis.Manifest{
		ID:         "org.example.runner-fixture",
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
		}, {
			Name:    "labels",
			Type:    "string[]",
			Default: []string{},
		}},
	}
}

func (a *runnerTestAnalyzer) Manifest() analysis.Manifest { return a.manifest() }

func (a *runnerTestAnalyzer) Detect(context.Context, analysis.DetectRequest) (analysis.DetectionCandidate, error) {
	return analysis.DetectionCandidate{
		AnalyzerID:     a.manifest().ID,
		Confidence:     1,
		MatchedMarkers: []string{"fixture.marker"},
		Reason:         "fixture marker",
	}, nil
}

func (a *runnerTestAnalyzer) Analyze(ctx context.Context, request analysis.AnalyzeRequest) (analysis.AnalysisResult, error) {
	a.seenSelection = request.Selection
	a.seenOptions = request.Options
	if a.waitForCancellation {
		a.cancellationObserved = make(chan struct{})
		<-ctx.Done()
		close(a.cancellationObserved)
		return analysis.AnalysisResult{}, ctx.Err()
	}
	if a.analyzerError != nil {
		return analysis.AnalysisResult{}, a.analyzerError
	}
	return analysis.AnalysisResult{
		RunID:  "runner-fixture",
		Status: analysis.StatusComplete,
		Analyzer: analysis.AnalyzerInfo{
			ID:         a.manifest().ID,
			Version:    a.manifest().Version,
			Language:   a.manifest().Language,
			APIVersion: a.manifest().APIVersion,
		},
		Project: analysis.ProjectInfo{
			RootLabel: "fixture",
			Boundary:  "fixture.marker",
		},
		OptionsFingerprint: request.Options.Fingerprint,
		Diagnostics: []analysis.Diagnostic{{
			Code:        "fixture.warning",
			Severity:    "warning",
			Message:     "streamed fixture warning",
			Recoverable: true,
		}},
	}, nil
}

func analyzeRequest(analyzerID, requestID string) processprotocol.Frame {
	selection := analysis.AnalyzerSelection{AnalyzerID: analyzerID, Mode: "explicit-id"}
	options := analysis.EffectiveOptions{Values: map[string]any{"include_tests": false}, Fingerprint: "fixture-options"}
	return processprotocol.Frame{
		Type:        processprotocol.FrameAnalyze,
		RequestID:   requestID,
		ProjectRoot: "/fixture",
		Selection:   &selection,
		Options:     &options,
	}
}

func encodedFrame(t *testing.T, frame processprotocol.Frame) string {
	t.Helper()
	var buffer bytes.Buffer
	if err := processprotocol.WriteFrame(&buffer, frame); err != nil {
		t.Fatalf("encode frame: %v", err)
	}
	return buffer.String()
}

func decodeFrames(t *testing.T, data []byte) []processprotocol.Frame {
	t.Helper()
	decoder := processprotocol.NewDecoder(bytes.NewReader(data))
	frames := make([]processprotocol.Frame, 0)
	for {
		frame, err := decoder.ReadFrame()
		if errors.Is(err, io.EOF) {
			return frames
		}
		if err != nil {
			t.Fatalf("decode output: %v", err)
		}
		frames = append(frames, frame)
	}
}

func frameTypes(frames []processprotocol.Frame) []processprotocol.FrameType {
	types := make([]processprotocol.FrameType, 0, len(frames))
	for _, frame := range frames {
		types = append(types, frame.Type)
	}
	return types
}

func validateSession(t *testing.T, manifest analysis.Manifest, request processprotocol.Frame, frames []processprotocol.Frame) {
	t.Helper()
	session, err := processprotocol.NewProtocolSession(request.Type, request.RequestID, manifest)
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	if err := session.AcceptPluginFrame(frames[0]); err != nil {
		t.Fatalf("accept hello: %v", err)
	}
	if err := session.AcceptHostFrame(request); err != nil {
		t.Fatalf("accept request: %v", err)
	}
	for _, frame := range frames[1:] {
		if err := session.AcceptPluginFrame(frame); err != nil {
			t.Fatalf("accept plugin frame %q: %v", frame.Type, err)
		}
	}
	if !session.Complete() {
		t.Fatal("session did not complete")
	}
}
