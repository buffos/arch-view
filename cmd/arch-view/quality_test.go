package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/quality"
	qualitypolicy "github.com/buffo/arch-view/internal/quality/policy"
)

func TestAnalyzeQualityProfileEmitsReportAndExportProjections(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/quality\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {\n\tprintln(\"quality\")\n}\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	profilePath := filepath.Join(t.TempDir(), "quality-profile.json")
	profile := quality.QualityProfile{
		SchemaVersion:  quality.SchemaVersion,
		ProfileID:      "profile:cli-test",
		ProfileVersion: "1.0.0",
		EnabledRules: []quality.RuleBinding{{
			RuleID:      "source:file.max-lines",
			RuleVersion: "1.0.0",
			Enabled:     true,
			Parameters: quality.TypedConfigBlock{
				Namespace:     "rule-config:source-file-size",
				SchemaVersion: "1.0.0",
				Payload:       map[string]any{"operator": "greater_than", "limit": 1, "unit": "unit:line"},
			},
			Severity: quality.SeverityWarning,
		}},
		SeverityPolicy: quality.TypedConfigBlock{Namespace: "severity:default", SchemaVersion: "1.0.0", Payload: map[string]any{}},
		Constraints:    []quality.ArchitectureConstraint{},
		Extensions:     []quality.ExtensionBlock{},
	}
	profileData, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("marshal quality profile: %v", err)
	}
	if err := os.WriteFile(profilePath, profileData, 0o644); err != nil {
		t.Fatalf("write quality profile: %v", err)
	}

	analysisPath := filepath.Join(t.TempDir(), "quality-analysis.json")
	var stdout, stderr bytes.Buffer
	code := run([]string{"analyze", "--project", root, "--language", "go", "--quality-profile", profilePath, "--quality-exit-on", "warning", "--format", "analysis-json", "--output", analysisPath}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("quality analysis exit code = %d, stderr=%s", code, stderr.String())
	}
	data, err := os.ReadFile(analysisPath)
	if err != nil {
		t.Fatalf("read quality analysis: %v", err)
	}
	var result analysis.AnalysisResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("decode quality analysis: %v", err)
	}
	if result.QualityReport == nil || len(result.QualityReport.Findings) == 0 {
		t.Fatalf("quality report = %#v, want an active file finding", result.QualityReport)
	}
	if result.QualityReport.Findings[0].MessageCode != "quality:file-lines-exceeded" || !strings.Contains(result.QualityReport.Findings[0].Message, "File has") {
		t.Fatalf("file threshold finding = %#v, want the human-readable file message", result.QualityReport.Findings[0])
	}
	if err := quality.ValidateQualityEvaluation(*result.QualityReport); err != nil {
		t.Fatalf("decoded quality report validation: %v", err)
	}

	modelPath := filepath.Join(t.TempDir(), "quality-model.json")
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"model", "normalize", "--input", analysisPath, "--output", modelPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("normalize quality model exit code = %d, stderr=%s", code, stderr.String())
	}

	htmlPath := filepath.Join(t.TempDir(), "quality.html")
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"export", "--input", modelPath, "--format", "html", "--output", htmlPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("HTML quality export exit code = %d, stdout=%s, stderr=%s", code, stdout.String(), stderr.String())
	}
	htmlData, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("read quality HTML export: %v", err)
	}
	if !strings.Contains(string(htmlData), `"quality_report"`) {
		t.Fatal("HTML export did not preserve quality_report")
	}

	svgPath := filepath.Join(t.TempDir(), "quality.svg")
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"export", "--input", modelPath, "--format", "svg", "--output", svgPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("SVG quality export exit code = %d, stderr=%s", code, stderr.String())
	}
	svgData, err := os.ReadFile(svgPath)
	if err != nil {
		t.Fatalf("read quality SVG export: %v", err)
	}
	if !strings.Contains(string(svgData), `<quality-report status="available"`) || !strings.Contains(string(svgData), `affected-files="`) {
		t.Fatal("SVG export did not annotate the quality report")
	}
}

func TestParseQualityExitPolicyDefaultsAndRejectsInvalidValues(t *testing.T) {
	policy, err := parseQualityExitPolicy("warning", []string{"active,baseline", "active"})
	if err != nil {
		t.Fatalf("parse warning policy: %v", err)
	}
	if policy.MinimumSeverity != quality.SeverityWarning || len(policy.Statuses) != 2 || policy.Statuses[0] != quality.StatusActive || policy.Statuses[1] != quality.StatusBaseline {
		t.Fatalf("warning policy = %#v", policy)
	}
	anyPolicy, err := parseQualityExitPolicy("any", nil)
	if err != nil || anyPolicy.MinimumSeverity != quality.SeverityInfo || len(anyPolicy.Statuses) != 1 || anyPolicy.Statuses[0] != quality.StatusActive {
		t.Fatalf("any policy = %#v, err=%v", anyPolicy, err)
	}
	if _, err := parseQualityExitPolicy("critical", nil); err == nil {
		t.Fatal("invalid quality exit severity unexpectedly succeeded")
	}
}

func TestQualityBaselineCommandCreatesBaselineFromAnalysisReport(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/baseline\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {\n\tprintln(\"baseline\")\n}\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	profilePath := filepath.Join(t.TempDir(), "quality-profile.json")
	profile := quality.QualityProfile{
		SchemaVersion:  quality.SchemaVersion,
		ProfileID:      "profile:baseline-cli",
		ProfileVersion: "1.0.0",
		EnabledRules: []quality.RuleBinding{{
			RuleID: "source:file.max-lines", RuleVersion: "1.0.0", Enabled: true,
			Parameters: quality.TypedConfigBlock{Namespace: "rule-config:source-file-size", SchemaVersion: "1.0.0", Payload: map[string]any{"operator": "greater_than", "limit": 1, "unit": "unit:line"}},
			Severity:   quality.SeverityWarning,
		}},
		SeverityPolicy: quality.TypedConfigBlock{Namespace: "severity:default", SchemaVersion: "1.0.0", Payload: map[string]any{}},
		Constraints:    []quality.ArchitectureConstraint{},
		Extensions:     []quality.ExtensionBlock{},
	}
	profileData, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("marshal profile: %v", err)
	}
	if err := os.WriteFile(profilePath, profileData, 0o644); err != nil {
		t.Fatalf("write profile: %v", err)
	}

	analysisPath := filepath.Join(t.TempDir(), "analysis.json")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"analyze", "--project", root, "--language", "go", "--quality-profile", profilePath, "--format", "analysis-json", "--output", analysisPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("analyze exit code = %d, stderr=%s", code, stderr.String())
	}
	data, err := os.ReadFile(analysisPath)
	if err != nil {
		t.Fatalf("read analysis: %v", err)
	}
	var result analysis.AnalysisResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("decode analysis: %v", err)
	}
	if result.QualityReport == nil || len(result.QualityReport.Findings) != 1 {
		t.Fatalf("quality report = %#v, want one finding", result.QualityReport)
	}

	baselinePath := filepath.Join(t.TempDir(), "baseline.json")
	stdout.Reset()
	stderr.Reset()
	code := run([]string{"quality", "baseline", "--input", analysisPath, "--output", baselinePath, "--baseline-id", "baseline:main", "--finding", result.QualityReport.Findings[0].ID, "--reason", "accepted legacy size", "--owner", "team"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("baseline exit code = %d, stdout=%s, stderr=%s", code, stdout.String(), stderr.String())
	}
	baselineData, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatalf("read baseline: %v", err)
	}
	var baseline quality.Baseline
	if err := json.Unmarshal(baselineData, &baseline); err != nil {
		t.Fatalf("decode baseline: %v", err)
	}
	if err := quality.ValidateBaseline(baseline); err != nil {
		t.Fatalf("validate generated baseline: %v", err)
	}
	if len(baseline.Entries) != 1 || baseline.Entries[0].Reason != "accepted legacy size" || baseline.Entries[0].Owner != "team" {
		t.Fatalf("generated baseline = %#v", baseline)
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"quality", "baseline", "--input", analysisPath, "--output", baselinePath, "--baseline-id", "baseline:main", "--finding", result.QualityReport.Findings[0].ID, "--reason", "accepted legacy size"}, &stdout, &stderr); code == 0 {
		t.Fatal("baseline command overwrote an existing file without --overwrite")
	}
}

func TestQualityBaselineCommandHelpSucceeds(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"quality", "baseline", "--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("quality baseline help exit code = %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "arch-view quality baseline") {
		t.Fatalf("quality baseline help = %q", stdout.String())
	}
}

func TestManagedBaselineAddIsLoadedAutomaticallyByAnalyze(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/managed-baseline\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {\n\tprintln(\"managed baseline\")\n}\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "quality-profiles"), 0o755); err != nil {
		t.Fatalf("create quality profile directory: %v", err)
	}
	profile := quality.QualityProfile{
		SchemaVersion: quality.SchemaVersion, ProfileID: "profile:managed-baseline", ProfileVersion: "1.0.0",
		EnabledRules: []quality.RuleBinding{{
			RuleID: "source:file.max-lines", RuleVersion: "1.0.0", Enabled: true,
			Parameters: quality.TypedConfigBlock{Namespace: "rule-config:source-file-size", SchemaVersion: "1.0.0", Payload: map[string]any{"operator": "greater_than", "limit": 1, "unit": "unit:line"}}, Severity: quality.SeverityWarning,
		}},
		SeverityPolicy: quality.TypedConfigBlock{Namespace: "severity:default", SchemaVersion: "1.0.0", Payload: map[string]any{}}, Constraints: []quality.ArchitectureConstraint{}, Extensions: []quality.ExtensionBlock{},
	}
	profileData, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("marshal managed profile: %v", err)
	}
	profilePath := filepath.Join(root, "quality-profiles", "main.json")
	if err := os.WriteFile(profilePath, profileData, 0o644); err != nil {
		t.Fatalf("write managed profile: %v", err)
	}
	firstPath := filepath.Join(t.TempDir(), "first.json")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"analyze", "--project", root, "--language", "go", "--quality-profile", profilePath, "--format", "analysis-json", "--output", firstPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("first analyze exit code = %d, stderr=%s", code, stderr.String())
	}
	firstData, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatalf("read first report: %v", err)
	}
	var first analysis.AnalysisResult
	if err := json.Unmarshal(firstData, &first); err != nil {
		t.Fatalf("decode first report: %v", err)
	}
	if first.QualityReport == nil || len(first.QualityReport.Findings) != 1 || first.QualityReport.Findings[0].Status != quality.StatusActive {
		t.Fatalf("first quality report = %#v", first.QualityReport)
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"quality", "baseline", "add", "--project", root, "--profile", "quality-profiles/main.json", "--baseline-file", "main.json", "--input", firstPath, "--finding", first.QualityReport.Findings[0].FindingKey, "--reason", "accepted after review"}, &stdout, &stderr); code != 0 {
		t.Fatalf("managed baseline add exit code = %d, stdout=%s, stderr=%s", code, stdout.String(), stderr.String())
	}
	var appendResult managedBaselineCLIResult
	if err := json.Unmarshal(stdout.Bytes(), &appendResult); err != nil {
		t.Fatalf("decode managed baseline result: %v, output=%q", err, stdout.String())
	}
	if appendResult.Status != "updated" || appendResult.Revision != "1.0.0" || len(appendResult.AddedFindingKeys) != 1 {
		t.Fatalf("managed baseline result = %#v", appendResult)
	}
	secondPath := filepath.Join(t.TempDir(), "second.json")
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"analyze", "--project", root, "--language", "go", "--quality-profile", profilePath, "--format", "analysis-json", "--output", secondPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("second analyze exit code = %d, stderr=%s", code, stderr.String())
	}
	secondData, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatalf("read second report: %v", err)
	}
	var second analysis.AnalysisResult
	if err := json.Unmarshal(secondData, &second); err != nil {
		t.Fatalf("decode second report: %v", err)
	}
	if second.QualityReport == nil || len(second.QualityReport.Findings) != 1 || second.QualityReport.Findings[0].Status != quality.StatusSuppressed {
		t.Fatalf("automatic baseline report = %#v, want one suppressed finding", second.QualityReport)
	}
	baselineData, err := os.ReadFile(filepath.Join(root, qualitypolicy.BaselineDirectoryName, "main.json"))
	if err != nil {
		t.Fatalf("read managed baseline: %v", err)
	}
	var explicit quality.Baseline
	if err := json.Unmarshal(baselineData, &explicit); err != nil {
		t.Fatalf("decode managed baseline: %v", err)
	}
	explicit.BaselineID = "baseline:explicit"
	explicitPath := filepath.Join(t.TempDir(), "explicit.json")
	if err := writeQualityBaselineFile(explicitPath, explicit, false); err != nil {
		t.Fatalf("write explicit baseline: %v", err)
	}
	explicitReportPath := filepath.Join(t.TempDir(), "explicit-report.json")
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"analyze", "--project", root, "--language", "go", "--quality-profile", profilePath, "--quality-baseline", explicitPath, "--format", "analysis-json", "--output", explicitReportPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("explicit baseline analyze exit code = %d, stderr=%s", code, stderr.String())
	}
	explicitData, err := os.ReadFile(explicitReportPath)
	if err != nil {
		t.Fatalf("read explicit baseline report: %v", err)
	}
	var explicitResult analysis.AnalysisResult
	if err := json.Unmarshal(explicitData, &explicitResult); err != nil {
		t.Fatalf("decode explicit baseline report: %v", err)
	}
	if explicitResult.QualityReport == nil || len(explicitResult.QualityReport.Findings) != 1 || explicitResult.QualityReport.Findings[0].Status != quality.StatusSuppressed || explicitResult.QualityReport.Baseline == nil || explicitResult.QualityReport.Baseline.BaselineID != "baseline:explicit" {
		t.Fatalf("explicit baseline report = %#v", explicitResult.QualityReport)
	}
	thirdPath := filepath.Join(t.TempDir(), "third.json")
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"analyze", "--project", root, "--language", "go", "--quality-profile", profilePath, "--no-quality-baseline", "--format", "analysis-json", "--output", thirdPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("no-baseline analyze exit code = %d, stderr=%s", code, stderr.String())
	}
	thirdData, err := os.ReadFile(thirdPath)
	if err != nil {
		t.Fatalf("read third report: %v", err)
	}
	var third analysis.AnalysisResult
	if err := json.Unmarshal(thirdData, &third); err != nil {
		t.Fatalf("decode third report: %v", err)
	}
	if third.QualityReport == nil || len(third.QualityReport.Findings) != 1 || third.QualityReport.Findings[0].Status != quality.StatusActive {
		t.Fatalf("no-baseline report = %#v, want one active finding", third.QualityReport)
	}
}
