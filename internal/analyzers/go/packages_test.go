package goanalyzer

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/sourceindex"
	"github.com/buffo/arch-view/internal/quality"
	"github.com/buffo/arch-view/internal/quality/adapter"
)

func TestAnalyzeDiscoversPackagesImportsAndEvidence(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "go.mod"), "module example.com/app\ngo 1.22\n")
	writeFixture(t, filepath.Join(root, "main.go"), `package app

import (
	"fmt"
	"appengine"
	"example.com/app/internal/service"
	"github.com/acme/library"
)

func main() { fmt.Println(service.Name, library.Name) }
`)
	writeFixture(t, filepath.Join(root, "internal", "service", "service.go"), `package service

const Name = "service"
`)

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{
		ProjectRoot: root,
		Options:     goOptions(t, nil),
	})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if result.Status != analysis.StatusComplete {
		t.Fatalf("status = %q, want complete; diagnostics=%#v", result.Status, result.Diagnostics)
	}
	if result.SourceIndex == nil || len(result.SourceIndex.Snapshots) != 1 || len(result.SourceIndex.Snapshots[0].Files) != 2 || len(result.SourceIndex.Snapshots[0].Symbols) == 0 {
		t.Fatalf("source-index attachment = %#v, want one snapshot with files and declarations", result.SourceIndex)
	}
	repeat, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{
		ProjectRoot: root,
		Options:     goOptions(t, nil),
	})
	if err != nil {
		t.Fatalf("repeat analyze: %v", err)
	}
	firstJSON, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal first analysis: %v", err)
	}
	repeatJSON, err := json.Marshal(repeat)
	if err != nil {
		t.Fatalf("marshal repeated analysis: %v", err)
	}
	if string(firstJSON) != string(repeatJSON) {
		t.Fatalf("analysis is not deterministic:\n%s\n%s", firstJSON, repeatJSON)
	}
	if len(result.Modules) != 2 {
		t.Fatalf("module count = %d, want 2; modules=%#v", len(result.Modules), result.Modules)
	}
	if result.Modules[0].ID != "go:example.com/app" || result.Modules[1].ID != "go:example.com/app/internal/service" {
		t.Fatalf("module ids = %#v", moduleIDs(result.Modules))
	}
	localFound := false
	for _, relationship := range result.Relationships {
		if relationship.ToModuleID == "go:example.com/app/internal/service" {
			localFound = true
			if len(relationship.SourceReferenceIDs) != 1 {
				t.Fatalf("local relationship evidence = %#v", relationship.SourceReferenceIDs)
			}
		}
	}
	if !localFound {
		t.Fatalf("local relationship not found: %#v", result.Relationships)
	}
	if !hasReferenceScope(result.References, "standard_library") || !hasReferenceScope(result.References, "external") || referenceScope(result.References, "appengine") != "external" {
		t.Fatalf("non-local references = %#v", result.References)
	}
	for _, source := range result.SourceReferences {
		if source.Kind == "import" && (source.Start == nil || source.Start.Line < 1 || source.Start.Column < 1) {
			t.Fatalf("import source lacks location: %#v", source)
		}
	}
}

func TestAnalyzePublishesStructuralFactsForQualitySignals(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "go.mod"), "module example.com/solid\ngo 1.22\n")
	writeFixture(t, filepath.Join(root, "service.go"), `package solid

type Dependency interface {
	Read() error
	Write() error
}

type Base struct{}
type Client struct{}

type Service struct {
	Base
	dep    Dependency
	client *Client
}

func (s *Service) Decode(value any) error {
	switch value.(type) {
	case string:
		return nil
	default:
		return nil
	}
}
`)

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: goOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if result.SourceIndex == nil || len(result.SourceIndex.Snapshots) != 1 {
		t.Fatalf("source index = %#v, want one Go snapshot", result.SourceIndex)
	}
	snapshot := result.SourceIndex.Snapshots[0]
	if !sourceCapabilityObserved(snapshot, sourceindex.CapabilitySolidStructure) {
		t.Fatalf("SOLID source capability = %#v, want observed", snapshot)
	}

	input, err := adapter.EvaluationInputFromSourceIndex(*result.SourceIndex)
	if err != nil {
		t.Fatalf("adapt source index: %v", err)
	}
	profile := quality.QualityProfile{
		SchemaVersion:  quality.SchemaVersion,
		ProfileID:      "profile:solid-end-to-end",
		ProfileVersion: "1.0.0",
		EnabledRules: []quality.RuleBinding{
			{RuleID: "signal:solid.srp", RuleVersion: "1.0.0", Enabled: true, Parameters: solidRuleParameters()},
			{RuleID: "signal:solid.ocp", RuleVersion: "1.0.0", Enabled: true, Parameters: solidRuleParameters()},
			{RuleID: "signal:solid.lsp", RuleVersion: "1.0.0", Enabled: true, Parameters: solidRuleParameters()},
			{RuleID: "signal:solid.isp", RuleVersion: "1.0.0", Enabled: true, Parameters: solidRuleParameters()},
			{RuleID: "signal:solid.dip", RuleVersion: "1.0.0", Enabled: true, Parameters: solidRuleParameters()},
		},
		SeverityPolicy: quality.TypedConfigBlock{Namespace: "severity:default", SchemaVersion: "1.0.0", Payload: map[string]any{}},
		Constraints:    []quality.ArchitectureConstraint{},
		Extensions:     []quality.ExtensionBlock{},
	}
	report, err := quality.EvaluateQualityProfile(profile, input, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate SOLID profile: %v", err)
	}
	for _, ruleID := range []string{"signal:solid.srp", "signal:solid.ocp", "signal:solid.lsp", "signal:solid.isp", "signal:solid.dip"} {
		coverage := qualityCoverageForRule(report.Coverage, ruleID)
		if coverage.Status == quality.CoverageUnsupported || coverage.Status == quality.CoverageNotEvaluable {
			t.Fatalf("%s coverage = %#v, want evaluated structural facts", ruleID, coverage)
		}
	}
}

func sourceCapabilityObserved(snapshot analysis.SourceIndexSnapshot, capability string) bool {
	for _, value := range snapshot.Coverage {
		if value.Capability == capability {
			return value.Status == analysis.FactStatusObserved
		}
	}
	return false
}

func solidRuleParameters() quality.TypedConfigBlock {
	return quality.TypedConfigBlock{Namespace: "rule-config:solid-signal", SchemaVersion: "1.0.0", Payload: map[string]any{}}
}

func qualityCoverageForRule(values []quality.QualityCoverage, ruleID string) quality.QualityCoverage {
	for _, value := range values {
		if value.RuleID == ruleID {
			return value
		}
	}
	return quality.QualityCoverage{RuleID: ruleID, Status: "missing"}
}

func TestAnalyzeReportsUnresolvedAndCgoImportsAsPartial(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "go.mod"), "module example.com/app\ngo 1.22\n")
	writeFixture(t, filepath.Join(root, "main.go"), `package app

import (
	"C"
	"example.com/app/missing"
)

var _ = missing.Value
var _ = C.int(1)
`)

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: goOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if result.Status != analysis.StatusPartial {
		t.Fatalf("status = %q, want partial", result.Status)
	}
	if !hasDiagnostic(result.Diagnostics, "go_unresolved_import") || !hasDiagnostic(result.Diagnostics, "go_cgo_import") {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
	if !hasReferenceTargetScope(result.References, "unresolved") || !hasReferenceTargetScope(result.References, "cgo") {
		t.Fatalf("references = %#v", result.References)
	}
	for _, relationship := range result.Relationships {
		if strings.Contains(relationship.ID, "missing") {
			t.Fatalf("relationship id should be stable hash, got %q", relationship.ID)
		}
	}
}

func TestAnalyzePreservesParserErrorLocation(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "go.mod"), "module example.com/app\ngo 1.22\n")
	writeFixture(t, filepath.Join(root, "broken.go"), "package broken\n\nfunc broken( {\n")

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: goOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "go_parse_error" {
			if diagnostic.Location == nil || diagnostic.Location.Line < 3 {
				t.Fatalf("parse diagnostic location = %#v, want source line >= 3", diagnostic.Location)
			}
			return
		}
	}
	t.Fatalf("parse diagnostic missing: %#v", result.Diagnostics)
}

func TestAnalyzeHonorsSourceExclusionsAndBuildOptions(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "go.mod"), "module example.com/app\ngo 1.22\n")
	writeFixture(t, filepath.Join(root, "root.go"), "package root\n")
	writeFixture(t, filepath.Join(root, "tests", "tests_test.go"), "package tests\n")
	writeFixture(t, filepath.Join(root, "root_test.go"), "package root_test\n")
	writeFixture(t, filepath.Join(root, "generated", "generated.go"), "// Code generated by fixture. DO NOT EDIT.\npackage generated\n")
	writeFixture(t, filepath.Join(root, "tagged", "tagged.go"), "//go:build integration\n\npackage tagged\n")
	writeFixture(t, filepath.Join(root, "vendor", "vendor.go"), "package vendor\n")
	writeFixture(t, filepath.Join(root, "external", "external.go"), "package external\n")
	writeFixture(t, filepath.Join(root, "excluded", "excluded.go"), "package excluded\n")
	writeFixture(t, filepath.Join(root, "build", "output.go"), "package output\n")
	writeFixture(t, filepath.Join(root, "internal", "build", "build.go"), "package build\n")
	writeFixture(t, filepath.Join(root, "nested", "go.mod"), "module example.com/nested\n")
	writeFixture(t, filepath.Join(root, "nested", "nested.go"), "package nested\n")

	defaultResult, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: goOptions(t, nil)})
	if err != nil {
		t.Fatalf("default analyze: %v", err)
	}
	if len(defaultResult.Modules) != 3 || defaultResult.Modules[0].ID != "go:example.com/app" {
		t.Fatalf("default modules = %#v", moduleIDs(defaultResult.Modules))
	}

	fullResult, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{
		ProjectRoot: root,
		Options: goOptions(t, map[string]any{
			"include_tests":     true,
			"include_generated": true,
			"build_tags":        []string{"integration"},
		}),
	})
	if err != nil {
		t.Fatalf("expanded analyze: %v", err)
	}
	if fullResult.Status != analysis.StatusComplete {
		t.Fatalf("expanded analyze status = %q, diagnostics=%#v", fullResult.Status, fullResult.Diagnostics)
	}
	ids := moduleIDs(fullResult.Modules)
	for _, expected := range []string{"go:example.com/app", "go:example.com/app/tests", "go:example.com/app/generated", "go:example.com/app/tagged"} {
		if !containsString(ids, expected) {
			t.Fatalf("expanded modules missing %q: %#v", expected, ids)
		}
	}
	for _, id := range ids {
		if strings.Contains(id, "/vendor") || strings.Contains(id, "/external") || strings.Contains(id, "/nested") || id == "go:example.com/app/build" {
			t.Fatalf("excluded directory became module: %q", id)
		}
	}
	if !containsString(ids, "go:example.com/app/internal/build") {
		t.Fatalf("legitimate nested package was excluded: %#v", ids)
	}

	excludedResult, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{
		ProjectRoot: root,
		Options: goOptions(t, map[string]any{
			"exclude": []string{"excluded/**"},
		}),
	})
	if err != nil {
		t.Fatalf("explicit exclusion analyze: %v", err)
	}
	if containsString(moduleIDs(excludedResult.Modules), "go:example.com/app/excluded") {
		t.Fatalf("explicitly excluded module was discovered: %#v", moduleIDs(excludedResult.Modules))
	}
}

func moduleIDs(values []analysis.ModuleObservation) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.ID)
	}
	return result
}

func hasReferenceScope(values []analysis.Reference, scope string) bool {
	for _, value := range values {
		if value.Scope == scope {
			return true
		}
	}
	return false
}

func referenceScope(values []analysis.Reference, name string) string {
	for _, value := range values {
		if value.Name == name {
			return value.Scope
		}
	}
	return ""
}

func hasReferenceTargetScope(values []analysis.Reference, targetScope string) bool {
	for _, value := range values {
		if value.Metadata["target_scope"] == targetScope {
			return true
		}
	}
	return false
}

func hasDiagnostic(values []analysis.Diagnostic, code string) bool {
	for _, value := range values {
		if value.Code == code {
			return true
		}
	}
	return false
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
