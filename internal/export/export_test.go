package export

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/model/canonical"
	"github.com/buffo/arch-view/internal/viewer/scene"
)

func TestRenderJSONIsCanonicalAndPreservesPartialFacts(t *testing.T) {
	value := exportFixtureModel(t)
	request := Request{Format: FormatJSON}

	firstMetadata, first, err := Render(value, request)
	if err != nil {
		t.Fatalf("first JSON render: %v", err)
	}
	secondMetadata, second, err := Render(value, request)
	if err != nil {
		t.Fatalf("second JSON render: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("identical JSON exports are not byte-stable")
	}
	if firstMetadata.ContentHash != secondMetadata.ContentHash || firstMetadata.Bytes != len(first) {
		t.Fatalf("metadata does not describe deterministic bytes: first=%#v second=%#v", firstMetadata, secondMetadata)
	}

	var decoded model.Model
	if err := json.Unmarshal(first, &decoded); err != nil {
		t.Fatalf("decode exported model: %v", err)
	}
	if err := canonical.Validate(decoded); err != nil {
		t.Fatalf("validate exported model: %v", err)
	}
	if decoded.Status != model.StatusPartial || len(decoded.Diagnostics) == 0 || len(decoded.References) != 4 {
		t.Fatalf("export lost partial/reference facts: status=%q diagnostics=%d references=%d", decoded.Status, len(decoded.Diagnostics), len(decoded.References))
	}
	if len(decoded.Derived.Cycles) == 0 || len(decoded.Derived.Layers) == 0 {
		t.Fatalf("export lost derived cycles/layers: %#v", decoded.Derived)
	}
}

func TestRenderHTMLIsSelfContainedAndUsesEmbeddedSceneCatalog(t *testing.T) {
	value := exportFixtureModel(t)
	metadata, data, err := Render(value, Request{
		Format:              FormatHTML,
		ReferenceVisibility: scene.ReferenceVisibilityHidden,
	})
	if err != nil {
		t.Fatalf("HTML render: %v", err)
	}
	if metadata.LayoutProvenance["engine"] != "deterministic-export" {
		t.Fatalf("HTML layout provenance = %#v", metadata.LayoutProvenance)
	}
	html := string(data)
	for _, forbidden := range []string{
		`<link rel="stylesheet"`,
		`<script src=`,
		`<script type="module"`,
		`href="/assets/`,
		`href="http://`,
		`href="https://`,
		`import {`,
		`from "./`,
		`/assets/`,
	} {
		if strings.Contains(html, forbidden) {
			t.Fatalf("self-contained HTML contains external resource %q", forbidden)
		}
	}
	for _, required := range []string{
		"window.__ARCH_VIEW_EXPORT__",
		"self-contained architecture export",
		"source not embedded",
		"internal/a/a.go",
		"standard_library",
		`"scenes"`,
		`"layouts"`,
	} {
		if !strings.Contains(strings.ToLower(html), strings.ToLower(required)) {
			t.Fatalf("self-contained HTML is missing %q", required)
		}
	}
}

func TestRenderSVGIsStaticAccessibleDeterministicAndTraceable(t *testing.T) {
	value := exportFixtureModel(t)
	request := Request{
		Format:              FormatSVG,
		ReferenceVisibility: scene.ReferenceVisibilityExpanded,
	}
	firstMetadata, first, err := Render(value, request)
	if err != nil {
		t.Fatalf("first SVG render: %v", err)
	}
	secondMetadata, second, err := Render(value, request)
	if err != nil {
		t.Fatalf("second SVG render: %v", err)
	}
	if !bytes.Equal(first, second) || firstMetadata.ContentHash != secondMetadata.ContentHash {
		t.Fatal("identical SVG exports are not byte-stable")
	}
	svg := string(first)
	for _, required := range []string{
		`<title id="arch-view-title">`,
		`<desc id="arch-view-description">`,
		`data-module-id=`,
		`data-relationship-id=`,
		`data-reference-scope="standard_library"`,
		`data-confidence-state="high"`,
		`layout-engine="deterministic-export"`,
		`layout-algorithm="layered-orthogonal-v1"`,
		`.edge-line.feedback`,
		`relationship-ids="rel-cmd-fmt"`,
		`confidence-state="high"`,
		`data-cycle-state="cycle"`,
		`data-diagnostic-state=`,
		`internal/a/a.go`,
	} {
		if !strings.Contains(svg, required) {
			t.Fatalf("SVG is missing %q", required)
		}
	}
	if strings.Contains(svg, "<script") || strings.Contains(svg, "<image") || strings.Contains(svg, "<use") {
		t.Fatal("SVG contains an executable or externally-loaded element")
	}
	if !strings.Contains(svg, `data-module-id="ref-fmt"`) {
		t.Fatal("expanded SVG does not expose the individual reference node")
	}
}

func TestVisualExportsRespectLocalFirstReferenceVisibility(t *testing.T) {
	value := exportFixtureModel(t)
	hidden, _, err := Render(value, Request{Format: FormatSVG, ReferenceVisibility: scene.ReferenceVisibilityHidden})
	if err != nil {
		t.Fatalf("hidden SVG render: %v", err)
	}
	aggregated, _, err := Render(value, Request{Format: FormatSVG, ReferenceVisibility: scene.ReferenceVisibilityAggregated})
	if err != nil {
		t.Fatalf("aggregated SVG render: %v", err)
	}
	hiddenData := string(renderedBytes(t, value, Request{Format: FormatSVG, ReferenceVisibility: scene.ReferenceVisibilityHidden}))
	aggregatedData := string(renderedBytes(t, value, Request{Format: FormatSVG, ReferenceVisibility: scene.ReferenceVisibilityAggregated}))
	if strings.Contains(hiddenData, `data-module-id="ref-fmt"`) || strings.Contains(hiddenData, `data-module-id="reference-boundary:standard_library"`) {
		t.Fatal("hidden SVG exposed a reference node")
	}
	if !strings.Contains(aggregatedData, `data-module-id="reference-boundary:standard_library"`) {
		t.Fatal("aggregated SVG did not expose a standard-library boundary node")
	}
	if hidden.ContentHash == aggregated.ContentHash {
		t.Fatal("reference visibility did not change the visual artifact")
	}
}

func TestWriteProtectsExistingOutputAndReplacesAtomically(t *testing.T) {
	value := exportFixtureModel(t)
	output := filepath.Join(t.TempDir(), "architecture.json")

	metadata, err := Write(value, Request{Format: FormatJSON, OutputPath: output})
	if err != nil {
		t.Fatalf("initial write: %v", err)
	}
	initial, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read initial output: %v", err)
	}
	if metadata.Bytes != len(initial) {
		t.Fatalf("metadata bytes=%d, file bytes=%d", metadata.Bytes, len(initial))
	}

	if err := os.WriteFile(output, []byte("sentinel"), 0o644); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}
	if _, err := Write(value, Request{Format: FormatJSON, OutputPath: output}); analysis.ErrorCodeOf(err) != analysis.ErrInvalidRequest {
		t.Fatalf("existing output error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrInvalidRequest)
	}
	sentinel, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read protected output: %v", err)
	}
	if string(sentinel) != "sentinel" {
		t.Fatalf("protected output changed to %q", string(sentinel))
	}

	if _, err := Write(value, Request{Format: FormatJSON, OutputPath: output, Overwrite: true}); err != nil {
		t.Fatalf("overwrite output: %v", err)
	}
	replaced, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read replaced output: %v", err)
	}
	if !bytes.Equal(replaced, initial) {
		t.Fatal("overwrite did not restore the canonical artifact")
	}
}

func TestReplaceFileDoesNotClobberExistingTargetWithoutOverwrite(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "temporary.json")
	target := filepath.Join(directory, "architecture.json")
	if err := os.WriteFile(source, []byte("replacement"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(target, []byte("sentinel"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := replaceFile(source, target, false); err == nil {
		t.Fatal("no-overwrite replacement succeeded over an existing target")
	}
	targetData, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(targetData) != "sentinel" {
		t.Fatalf("existing target changed to %q", string(targetData))
	}
	sourceData, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source after refused replacement: %v", err)
	}
	if string(sourceData) != "replacement" {
		t.Fatalf("source changed after refused replacement to %q", string(sourceData))
	}
}

func TestRenderRejectsUnsupportedEmbeddingInvalidViewAndCancellation(t *testing.T) {
	value := exportFixtureModel(t)
	if _, _, err := Render(value, Request{Format: FormatJSON, EmbedSource: true}); analysis.ErrorCodeOf(err) != analysis.ErrUnsupportedOption {
		t.Fatalf("embed source error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrUnsupportedOption)
	}
	if _, _, err := Render(value, Request{Format: FormatHTML, ViewPath: []string{"does-not-exist"}}); analysis.ErrorCodeOf(err) != analysis.ErrInvalidRequest {
		t.Fatalf("invalid view path error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrInvalidRequest)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := Render(value, Request{Format: FormatSVG, Context: ctx}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled render error = %v, want context.Canceled", err)
	}
}

func renderedBytes(t *testing.T, value model.Model, request Request) []byte {
	t.Helper()
	_, data, err := Render(value, request)
	if err != nil {
		t.Fatalf("render %s: %v", request.ReferenceVisibility, err)
	}
	return data
}

func exportFixtureModel(t *testing.T) model.Model {
	t.Helper()
	result := analysis.AnalysisResult{
		RunID:  "run-export-fixture",
		Status: analysis.StatusPartial,
		Analyzer: analysis.AnalyzerInfo{
			ID:         "org.archview.go",
			Version:    "1.0.0",
			Language:   "go",
			APIVersion: analysis.AnalyzerAPIVersion,
		},
		Project: analysis.ProjectInfo{RootLabel: "export-fixture", Boundary: "go.mod", ModulePath: "example.com/export"},
		Modules: []analysis.ModuleObservation{
			{ID: "go:example.com/export/cmd", Language: "go", Kind: "package", Name: "cmd", DisplayName: "example.com/export/cmd", Hierarchy: []string{"cmd"}, SourceReferenceIDs: []string{"src-cmd-file"}, Tags: []string{"entrypoint"}},
			{ID: "go:example.com/export/internal/a", Language: "go", Kind: "package", Name: "a", DisplayName: "example.com/export/internal/a", Hierarchy: []string{"internal", "a"}, SourceReferenceIDs: []string{"src-a-file"}, Tags: []string{"core"}},
			{ID: "go:example.com/export/internal/b", Language: "go", Kind: "package", Name: "b", DisplayName: "example.com/export/internal/b", Hierarchy: []string{"internal", "b"}, SourceReferenceIDs: []string{"src-b-file"}, Tags: []string{"core"}},
		},
		References: []analysis.Reference{
			{ID: "ref-dynamic", Name: "plugin", Scope: "dynamic", Language: "go"},
			{ID: "ref-ext", Name: "github.com/acme/lib", Scope: "external", Language: "go"},
			{ID: "ref-fmt", Name: "fmt", Scope: "standard_library", Language: "go"},
			{ID: "ref-missing", Name: "example.com/missing", Scope: "unresolved", Language: "go"},
		},
		SourceReferences: []analysis.SourceReference{
			{ID: "src-a-file", Path: "internal/a/a.go", Kind: "file"},
			{ID: "src-a-b", Path: "internal/a/a.go", Start: &analysis.Position{Line: 5, Column: 2}, End: &analysis.Position{Line: 5, Column: 28}, Kind: "import"},
			{ID: "src-a-ext", Path: "internal/a/a.go", Start: &analysis.Position{Line: 6, Column: 2}, End: &analysis.Position{Line: 6, Column: 24}, Kind: "import"},
			{ID: "src-b-file", Path: "internal/b/b.go", Kind: "file"},
			{ID: "src-b-a", Path: "internal/b/b.go", Start: &analysis.Position{Line: 5, Column: 2}, End: &analysis.Position{Line: 5, Column: 28}, Kind: "import"},
			{ID: "src-b-missing", Path: "internal/b/b.go", Start: &analysis.Position{Line: 7, Column: 2}, End: &analysis.Position{Line: 7, Column: 30}, Kind: "import"},
			{ID: "src-cmd-a", Path: "cmd/main.go", Start: &analysis.Position{Line: 5, Column: 2}, End: &analysis.Position{Line: 5, Column: 28}, Kind: "import"},
			{ID: "src-cmd-dynamic", Path: "cmd/main.go", Start: &analysis.Position{Line: 7, Column: 2}, End: &analysis.Position{Line: 7, Column: 20}, Kind: "import"},
			{ID: "src-cmd-file", Path: "cmd/main.go", Kind: "file"},
			{ID: "src-cmd-fmt", Path: "cmd/main.go", Start: &analysis.Position{Line: 6, Column: 2}, End: &analysis.Position{Line: 6, Column: 12}, Kind: "import"},
		},
		Relationships: []analysis.RelationshipObservation{
			{ID: "rel-a-b", Type: "depends_on", FromModuleID: "go:example.com/export/internal/a", ToModuleID: "go:example.com/export/internal/b", SourceReferenceIDs: []string{"src-a-b"}, Confidence: &analysis.Confidence{Basis: "resolved", Score: 1}},
			{ID: "rel-a-ext", Type: "depends_on", FromModuleID: "go:example.com/export/internal/a", ToReferenceID: "ref-ext", SourceReferenceIDs: []string{"src-a-ext"}, Confidence: &analysis.Confidence{Basis: "resolved", Score: 0.9}},
			{ID: "rel-b-a", Type: "depends_on", FromModuleID: "go:example.com/export/internal/b", ToModuleID: "go:example.com/export/internal/a", SourceReferenceIDs: []string{"src-b-a"}, Confidence: &analysis.Confidence{Basis: "resolved", Score: 1}},
			{ID: "rel-b-missing", Type: "depends_on", FromModuleID: "go:example.com/export/internal/b", ToReferenceID: "ref-missing", SourceReferenceIDs: []string{"src-b-missing"}, Confidence: &analysis.Confidence{Basis: "unresolved", Score: 0.2}},
			{ID: "rel-cmd-a", Type: "depends_on", FromModuleID: "go:example.com/export/cmd", ToModuleID: "go:example.com/export/internal/a", SourceReferenceIDs: []string{"src-cmd-a"}, Confidence: &analysis.Confidence{Basis: "resolved", Score: 1}},
			{ID: "rel-cmd-dynamic", Type: "depends_on", FromModuleID: "go:example.com/export/cmd", ToReferenceID: "ref-dynamic", SourceReferenceIDs: []string{"src-cmd-dynamic"}, Confidence: &analysis.Confidence{Basis: "dynamic", Score: 0.4}},
			{ID: "rel-cmd-fmt", Type: "depends_on", FromModuleID: "go:example.com/export/cmd", ToReferenceID: "ref-fmt", SourceReferenceIDs: []string{"src-cmd-fmt"}, Confidence: &analysis.Confidence{Basis: "resolved", Score: 1}},
		},
		Diagnostics: []analysis.Diagnostic{{
			Code:        "go_unresolved_import",
			Severity:    "warning",
			Message:     "could not resolve imported package",
			Subject:     "go:example.com/export/internal/b",
			Path:        "internal/b/b.go",
			Location:    &analysis.Position{Line: 7, Column: 2},
			Recoverable: true,
		}},
	}
	value, err := canonical.Normalize(result)
	if err != nil {
		t.Fatalf("normalize export fixture: %v", err)
	}
	return value
}
