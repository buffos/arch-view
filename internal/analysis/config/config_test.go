package config

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/viewer/layout"
)

type configFixtureAnalyzer struct{ manifest analysis.Manifest }

func (value configFixtureAnalyzer) Manifest() analysis.Manifest { return value.manifest }
func (value configFixtureAnalyzer) Detect(context.Context, analysis.DetectRequest) (analysis.DetectionCandidate, error) {
	return analysis.DetectionCandidate{AnalyzerID: value.manifest.ID, Confidence: 1}, nil
}
func (value configFixtureAnalyzer) Analyze(context.Context, analysis.AnalyzeRequest) (analysis.AnalysisResult, error) {
	return analysis.AnalysisResult{}, nil
}

func configRegistry() *analysis.Registry {
	registry := analysis.NewRegistry()
	_ = registry.Register(configFixtureAnalyzer{manifest: analysis.Manifest{
		ID: "org.example.go", Version: "1.0.0", Language: "go", APIVersion: analysis.AnalyzerAPIVersion,
		DetectionMarkers: []analysis.DetectionMarker{{Kind: "file", Value: "go.mod", Weight: 1}},
		Options: []analysis.OptionDescriptor{
			{Name: "include_tests", Type: "boolean", Default: false},
			{Name: "secret", Type: "string", Default: nil, Sensitive: true},
		},
	}})
	return registry
}

func TestDecodeV1AndV2Configuration(t *testing.T) {
	registry := configRegistry()
	v1, err := layout.EncodeConfig(layout.DefaultProfile())
	if err != nil {
		t.Fatalf("encode v1: %v", err)
	}
	value, err := Decode(v1, registry)
	if err != nil {
		t.Fatalf("decode v1: %v", err)
	}
	if value.SchemaVersion != SchemaVersionV1 || value.Analysis != nil || value.Layout.Algorithm != "layered" {
		t.Fatalf("v1 value = %#v", value)
	}

	v2 := []byte(`{
        "schema_version": "arch-view.config/v2",
        "layout": {"algorithm": "layered", "options": {}},
        "analysis": {
            "exclude": ["generated/**"],
            "include": [{"analyzer_id": "org.example.go", "globs": ["src/**"]}],
            "assignments": [
                {"path": "src", "analyzer_id": "org.example.go", "options": {"include_tests": true}},
                {"path": ".", "analyzer_id": "org.missing", "options": {}}
            ]
        }
    }`)
	value, err = DecodeAt(v2, t.TempDir(), ".", registry)
	if err != nil {
		t.Fatalf("decode v2: %v", err)
	}
	if value.Analysis == nil || len(value.Analysis.Assignments) != 2 || value.Analysis.Assignments[0].ProjectRoot != "." || value.Analysis.Assignments[1].ProjectRoot != "src" {
		t.Fatalf("normalized assignments = %#v", value.Analysis)
	}
	if len(value.Analysis.Diagnostics) != 1 || value.Analysis.Diagnostics[0].Code != string(analysis.ErrAssignmentAnalyzerUnavailable) {
		t.Fatalf("unavailable diagnostics = %#v", value.Analysis.Diagnostics)
	}
	if value.Fingerprint == "" || value.Analysis.Fingerprint == "" {
		t.Fatalf("missing fingerprints = %#v", value)
	}
}

func TestNearestInvalidConfigurationDoesNotFallThrough(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	valid := []byte(`{"schema_version":"arch-view.config/v1","layout":{"algorithm":"layered","options":{}}}`)
	if err := os.WriteFile(filepath.Join(root, FileName), valid, 0o644); err != nil {
		t.Fatalf("write ancestor: %v", err)
	}
	if err := os.WriteFile(filepath.Join(child, FileName), []byte(`{"schema_version":"arch-view.config/v2","layout":{"algorithm":"layered","options":{}},"analysis":{"assignments":[{"path":"../bad","analyzer_id":"org.example.go"}]}}`), 0o644); err != nil {
		t.Fatalf("write nearest: %v", err)
	}
	_, err := LoadNearestAt(root, child, configRegistry())
	if analysis.ErrorCodeOf(err) != analysis.ErrAnalysisConfigInvalid {
		t.Fatalf("error code = %q, want %q (%v)", analysis.ErrorCodeOf(err), analysis.ErrAnalysisConfigInvalid, err)
	}
	if !strings.Contains(err.Error(), "nearest .archview.json is invalid") {
		t.Fatalf("error = %v", err)
	}
}

func TestSourceScopePolicyNormalizesLoadedAbsoluteInvocationRoot(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "frontend")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	data := []byte(`{"schema_version":"arch-view.config/v2","layout":{"algorithm":"layered","options":{}},"analysis":{"exclude":["generated/**"]}}`)
	if err := os.WriteFile(filepath.Join(root, FileName), data, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	value, err := LoadNearestAt(root, child, configRegistry())
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	policy, err := value.SourceScopePolicy(child)
	if err != nil {
		t.Fatalf("source policy: %v", err)
	}
	if policy.InvocationRoot != "frontend" || len(policy.Exclude) != 1 || policy.Exclude[0] != "generated/**" {
		t.Fatalf("policy = %#v", policy)
	}
}

func TestUnsafeAndDuplicateAssignmentPathsAreRejected(t *testing.T) {
	paths := []string{"C:/outside", "/outside", "../outside", "src//nested", "src/*.go"}
	for _, assignmentPath := range paths {
		t.Run(assignmentPath, func(t *testing.T) {
			data := []byte(`{"schema_version":"arch-view.config/v2","layout":{"algorithm":"layered","options":{}},"analysis":{"assignments":[{"path":` + quoteJSON(assignmentPath) + `,"analyzer_id":"org.example.go"}]}}`)
			_, err := Decode(data, configRegistry())
			if analysis.ErrorCodeOf(err) != analysis.ErrAnalysisConfigInvalid && analysis.ErrorCodeOf(err) != analysis.ErrAnalysisAssignmentInvalid {
				t.Fatalf("error code = %q (%v)", analysis.ErrorCodeOf(err), err)
			}
		})
	}
	duplicate := []byte(`{"schema_version":"arch-view.config/v2","layout":{"algorithm":"layered","options":{}},"analysis":{"assignments":[{"path":".","analyzer_id":"org.example.go"},{"path":"./","analyzer_id":"org.example.go"}]}}`)
	_, err := Decode(duplicate, configRegistry())
	if analysis.ErrorCodeOf(err) != analysis.ErrAssignmentDuplicatePath {
		t.Fatalf("duplicate error code = %q (%v)", analysis.ErrorCodeOf(err), err)
	}
}

func TestConfigurationRejectsAmbiguousFiltersAndRedactsSensitiveValues(t *testing.T) {
	duplicateInclude := []byte(`{"schema_version":"arch-view.config/v2","layout":{"algorithm":"layered","options":{}},"analysis":{"include":[{"analyzer_id":"org.example.go","globs":["src/**"]},{"analyzer_id":"org.example.go","globs":["test/**"]}]}}`)
	_, err := Decode(duplicateInclude, configRegistry())
	if analysis.ErrorCodeOf(err) != analysis.ErrAnalysisScopeFilterInvalid {
		t.Fatalf("duplicate include code = %q (%v)", analysis.ErrorCodeOf(err), err)
	}

	secret := "do-not-leak-this-value"
	data := []byte(`{"schema_version":"arch-view.config/v2","layout":{"algorithm":"layered","options":{}},"analysis":{"assignments":[{"path":".","analyzer_id":"org.example.go","options":{"secret":` + quoteJSON(secret) + `}}]}}`)
	_, err = Decode(data, configRegistry())
	if analysis.ErrorCodeOf(err) != analysis.ErrAnalysisAssignmentInvalid || strings.Contains(err.Error(), secret) {
		t.Fatalf("sensitive option error = %q (%v)", err, err)
	}
	encoded, _ := json.Marshal(err)
	if strings.Contains(string(encoded), secret) {
		t.Fatalf("sensitive option leaked in JSON error: %s", encoded)
	}
}

func TestV2LayoutSavePreservesAnalysisSection(t *testing.T) {
	root := t.TempDir()
	data := []byte(`{"schema_version":"arch-view.config/v2","layout":{"algorithm":"layered","options":{}},"analysis":{"exclude":["generated/**"],"assignments":[]}}`)
	configPath := filepath.Join(root, FileName)
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatalf("write v2: %v", err)
	}
	session := layout.NewSession(root)
	if err := session.SaveActive(layout.DefaultProfile()); err != nil {
		t.Fatalf("save v2 layout: %v", err)
	}
	updated, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read v2: %v", err)
	}
	if !strings.Contains(string(updated), `"schema_version": "arch-view.config/v2"`) || !strings.Contains(string(updated), `"analysis"`) || !strings.Contains(string(updated), `generated/**`) {
		t.Fatalf("v2 analysis was not preserved: %s", updated)
	}
}

func quoteJSON(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}
