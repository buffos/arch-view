package live

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/sourceindex"
	gosyntax "github.com/buffo/arch-view/internal/analysis/syntax/go"
	"github.com/buffo/arch-view/internal/analyzers/go/sourcefacts"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/model/canonical"
	"github.com/buffo/arch-view/internal/quality"
	qualityadapter "github.com/buffo/arch-view/internal/quality/adapter"
	qualitypolicy "github.com/buffo/arch-view/internal/quality/policy"
)

func TestQualityPolicyCommandsEnforceAuthorizationFreshnessAndAudit(t *testing.T) {
	root := t.TempDir()
	profile := policyThresholdProfile()
	store, err := qualitypolicy.NewFileStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveProfile(context.Background(), profile, "source.json", false); err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	modelValue, report := policyModelAndReport(t, profile)
	validatedProfile, err := quality.ValidateQualityProfile(profile, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("validate seeded profile: %v", err)
	}
	expectedProfileDigest := quality.ProfileDigest(validatedProfile)
	if report.ProfileDigest == nil || *report.ProfileDigest != expectedProfileDigest {
		t.Fatalf("quality report profile digest mismatch: report=%+v expected=%+v", report.ProfileDigest, expectedProfileDigest)
	}
	resolver := NewMemoryQualityProfileResolver(profile)
	catalog := quality.NewDefaultCatalog()
	policyService := NewFileQualityPolicyService(qualitypolicy.NewService(store, catalog))
	config := testLiveConfig("session:policy-read-only")
	config.SourceIndexRequest = SourceIndexRequest{Enabled: true}
	config.QualityRequest = &QualityRequest{ProfileID: profile.ProfileID, ProfileVersion: profile.ProfileVersion}
	session, err := StartLiveSession(context.Background(), config, root, SessionOptions{
		Scanner: StaticScanner{Result: ScanResult{Model: modelValue, QualityReport: &report}}, Fingerprinter: StaticFingerprinter{Value: testInput("policy")},
		QualityCatalog: catalog, Profiles: resolver, PolicyService: policyService,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	if err := session.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}

	findingKey := report.Findings[0].FindingKey
	preview, err := session.QualityGateway().ExecuteQualityPolicyCommand(context.Background(), QualityPolicyCommand{
		Operation: PolicyPreviewBaseline, ProfileID: profile.ProfileID, ProfileVersion: profile.ProfileVersion,
		BaselineID: "baseline:policy", FindingKeys: []string{findingKey}, Reason: "reviewed",
	})
	if err != nil {
		t.Fatalf("baseline preview: %v", err)
	}
	previewResult := preview.Result.(QualityPolicyResult)
	if previewResult.Status != "preview" || len(previewResult.FindingKeys) != 1 || previewResult.FindingKeys[0] != findingKey || previewResult.PolicyIdentity.ReportRevision != 1 || previewResult.PolicyIdentity.ProfileID != profile.ProfileID || len(previewResult.PolicyIdentity.RuleVersions) != 1 {
		t.Fatalf("preview result = %#v", previewResult)
	}
	if _, err := os.Stat(filepath.Join(root, qualitypolicy.BaselineDirectoryName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preview changed baseline directory: %v", err)
	}

	_, err = session.QualityGateway().ExecuteQualityPolicyCommand(context.Background(), QualityPolicyCommand{
		Operation: PolicyCreateBaseline, ProfileID: profile.ProfileID, ProfileVersion: profile.ProfileVersion,
		ReportRevision: 1, BaselineID: "baseline:policy", FindingKeys: []string{findingKey}, FileName: "policy.json", Reason: "reviewed", Authorization: "nope",
	})
	assertLiveErrorCode(t, err, "mcp_permission_denied")
	if _, err := os.Stat(filepath.Join(root, qualitypolicy.BaselineDirectoryName, "policy.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("denied baseline changed policy files: %v", err)
	}

	writeConfig := config
	writeConfig.SessionID = "session:policy-write"
	writeConfig.PermissionPolicy.AllowedOperations = append(append([]Operation(nil), config.PermissionPolicy.AllowedOperations...), OperationQualityProfileWrite, OperationBaselineWrite)
	writeSession, err := StartLiveSession(context.Background(), writeConfig, root, SessionOptions{
		Scanner: StaticScanner{Result: ScanResult{Model: modelValue, QualityReport: &report}}, Fingerprinter: StaticFingerprinter{Value: testInput("policy")},
		QualityCatalog: catalog, Profiles: resolver, PolicyService: policyService,
		PolicyAuthorizer: PolicyAuthorizerFunc(func(_ context.Context, value PolicyAuthorization) error {
			if value.Authorization != "allow" {
				return errors.New("unexpected authorization")
			}
			return nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = writeSession.Close() })
	if err := writeSession.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}

	saved, err := writeSession.QualityGateway().ExecuteQualityPolicyCommand(context.Background(), QualityPolicyCommand{
		Operation: PolicySaveProfile, SourceProfileID: profile.ProfileID, SourceProfileVersion: profile.ProfileVersion,
		FileName: "saved.json", Authorization: "allow",
	})
	if err != nil {
		t.Fatalf("authorized profile save: %v", err)
	}
	savedResult := saved.Result.(QualityPolicyResult)
	if savedResult.Status != "saved" || savedResult.Write == nil || savedResult.Audit == nil || !savedResult.Audit.Authorized {
		t.Fatalf("saved result = %#v", savedResult)
	}
	if _, err := os.Stat(filepath.Join(root, qualitypolicy.ProfileDirectoryName, "saved.json")); err != nil {
		t.Fatalf("saved profile missing: %v", err)
	}
	_, err = writeSession.QualityGateway().ExecuteQualityPolicyCommand(context.Background(), QualityPolicyCommand{
		Operation: PolicySaveProfileAs, Profile: &profile, ProfileID: "profile:copy", ProfileVersion: "1.0.0", FileName: "saved.json", Authorization: "allow",
	})
	assertLiveErrorCode(t, err, ErrorQualityProfileConflict)

	created, err := writeSession.QualityGateway().ExecuteQualityPolicyCommand(context.Background(), QualityPolicyCommand{
		Operation: PolicyCreateBaseline, ReportRevision: 1, ReportID: report.EvaluationID,
		ProfileID: profile.ProfileID, ProfileVersion: profile.ProfileVersion, BaselineID: "baseline:policy",
		FindingKeys: []string{findingKey}, FileName: "policy.json", Reason: "reviewed", Authorization: "allow",
	})
	if err != nil {
		t.Fatalf("authorized baseline create: %v", err)
	}
	createdResult := created.Result.(QualityPolicyResult)
	if createdResult.Status != "created" || createdResult.Write == nil || createdResult.Audit == nil || createdResult.Write.RelativePath != "quality-baselines/policy.json" {
		t.Fatalf("created result = %#v", createdResult)
	}
	before, err := os.ReadFile(filepath.Join(root, qualitypolicy.BaselineDirectoryName, "policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = writeSession.QualityGateway().ExecuteQualityPolicyCommand(context.Background(), QualityPolicyCommand{
		Operation: PolicyCreateBaseline, ReportRevision: 1, ReportID: report.EvaluationID,
		ProfileID: profile.ProfileID, ProfileVersion: profile.ProfileVersion, BaselineID: "baseline:policy",
		FindingKeys: []string{findingKey}, FileName: "policy.json", Reason: "changed", Authorization: "allow",
	})
	assertLiveErrorCode(t, err, ErrorQualityProfileConflict)
	after, err := os.ReadFile(filepath.Join(root, qualitypolicy.BaselineDirectoryName, "policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("conflicting baseline request changed the existing document")
	}
}

func TestQualityPolicyRejectsStaleReportBeforeWriting(t *testing.T) {
	root := t.TempDir()
	profile := policyThresholdProfile()
	modelValue, report := policyModelAndReport(t, profile)
	catalog := quality.NewDefaultCatalog()
	resolver := NewMemoryQualityProfileResolver(profile)
	store, err := qualitypolicy.NewFileStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveProfile(context.Background(), profile, "source.json", false); err != nil {
		t.Fatal(err)
	}
	fingerprinter := &mutableFingerprinter{value: testInput("first")}
	config := testLiveConfig("session:policy-stale")
	config.SourceIndexRequest = SourceIndexRequest{Enabled: true}
	config.QualityRequest = &QualityRequest{ProfileID: profile.ProfileID, ProfileVersion: profile.ProfileVersion}
	config.PermissionPolicy.AllowedOperations = append(append([]Operation(nil), config.PermissionPolicy.AllowedOperations...), OperationBaselineWrite)
	session, err := StartLiveSession(context.Background(), config, root, SessionOptions{
		Scanner: StaticScanner{Result: ScanResult{Model: modelValue, QualityReport: &report}}, Fingerprinter: fingerprinter,
		QualityCatalog: catalog, Profiles: resolver, PolicyService: NewFileQualityPolicyService(qualitypolicy.NewService(store, catalog)),
		PolicyAuthorizer: PolicyAuthorizerFunc(func(context.Context, PolicyAuthorization) error { return nil }),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	if err := session.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	fingerprinter.Set(testInput("second"))
	_, err = session.QualityGateway().ExecuteQualityPolicyCommand(context.Background(), QualityPolicyCommand{
		Operation: PolicyCreateBaseline, ReportRevision: 1, ReportID: report.EvaluationID,
		BaselineID: "baseline:stale", FindingKeys: []string{report.Findings[0].FindingKey}, FileName: "stale.json", Reason: "reviewed", Authorization: "allow",
	})
	assertLiveErrorCode(t, err, ErrorBaselineRevisionStale)
	if _, err := os.Stat(filepath.Join(root, qualitypolicy.BaselineDirectoryName, "stale.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale baseline changed policy files: %v", err)
	}
}

func TestQualityPolicyRejectsPartialCoverageWithoutMutatingSource(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "main.go")
	sourceContent := []byte("package main\n\nfunc Main() {}\n")
	if err := os.WriteFile(sourcePath, sourceContent, 0o644); err != nil {
		t.Fatal(err)
	}
	profile := policyThresholdProfile()
	modelValue, report := policyModelAndReport(t, profile)
	report.Coverage = append([]quality.QualityCoverage(nil), report.Coverage...)
	report.Coverage[0].Status = quality.CoveragePartial
	report.Coverage[0].Reason = "fixture coverage is incomplete"
	report.Coverage[0].Provenance.Status = quality.CoveragePartial
	report, err := quality.NormalizeQualityEvaluation(report)
	if err != nil {
		t.Fatalf("normalize partial report: %v", err)
	}
	withoutReport, err := canonical.WithQualityReport(modelValue, nil)
	if err != nil {
		t.Fatal(err)
	}
	modelValue, err = canonical.WithQualityReport(withoutReport, &report)
	if err != nil {
		t.Fatal(err)
	}
	catalog := quality.NewDefaultCatalog()
	store, err := qualitypolicy.NewFileStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveProfile(context.Background(), profile, "source.json", false); err != nil {
		t.Fatal(err)
	}
	config := testLiveConfig("session:policy-partial")
	config.SourceIndexRequest = SourceIndexRequest{Enabled: true}
	config.QualityRequest = &QualityRequest{ProfileID: profile.ProfileID, ProfileVersion: profile.ProfileVersion}
	config.PermissionPolicy.AllowedOperations = append(config.PermissionPolicy.AllowedOperations, OperationBaselineWrite)
	session, err := StartLiveSession(context.Background(), config, root, SessionOptions{
		Scanner: StaticScanner{Result: ScanResult{Model: modelValue, QualityReport: &report}}, Fingerprinter: StaticFingerprinter{Value: testInput("partial")},
		QualityCatalog: catalog, Profiles: NewMemoryQualityProfileResolver(profile), PolicyService: NewFileQualityPolicyService(qualitypolicy.NewService(store, catalog)),
		PolicyAuthorizer: PolicyAuthorizerFunc(func(context.Context, PolicyAuthorization) error { return nil }),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	if err := session.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, err = session.QualityGateway().ExecuteQualityPolicyCommand(context.Background(), QualityPolicyCommand{
		Operation: PolicyCreateBaseline, ReportRevision: 1, ReportID: report.EvaluationID,
		BaselineID: "baseline:partial", FindingKeys: []string{report.Findings[0].FindingKey}, FileName: "partial.json", Reason: "reviewed", Authorization: "allow",
	})
	assertLiveErrorCode(t, err, ErrorQualityPolicyIncompatible)
	if _, err := os.Stat(filepath.Join(root, qualitypolicy.BaselineDirectoryName, "partial.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial baseline changed policy files: %v", err)
	}
	after, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(sourceContent) {
		t.Fatalf("policy command changed source: %q", after)
	}
}

func policyThresholdProfile() quality.QualityProfile {
	return quality.QualityProfile{
		SchemaVersion: quality.SchemaVersion, ProfileID: "profile:policy", ProfileVersion: "1.0.0",
		EnabledRules:   []quality.RuleBinding{{RuleID: "source:file.max-lines", RuleVersion: "1.0.0", Enabled: true, Parameters: quality.TypedConfigBlock{Namespace: "rule-config:source-file-size", SchemaVersion: "1.0.0", Payload: map[string]any{"operator": "greater_than", "limit": 1, "unit": "unit:line"}}}},
		SeverityPolicy: quality.TypedConfigBlock{}, Constraints: []quality.ArchitectureConstraint{}, Extensions: []quality.ExtensionBlock{},
	}
}

func policyModelAndReport(t *testing.T, profile quality.QualityProfile) (model.Model, quality.QualityEvaluation) {
	t.Helper()
	content := "package main\n\nfunc Main() {}\n"
	index, _, err := sourceindex.BuildSourceIndex(context.Background(), sourceindex.BuildInput{
		Scope:                 analysis.ScopeContext{ScopeID: "scope:policy", ProjectRoot: ".", Mode: analysis.SourceIndexScopeMode},
		Producer:              analysis.ProducerContext{AnalyzerID: "analyzer:test", AnalyzerVersion: "1.0.0"},
		Files:                 []sourceindex.SourceFileInput{{Path: "main.go", Content: []byte(content), Language: analysis.LanguageRef{ID: "language:go"}, Roles: []string{"role:source"}}},
		RequestedCapabilities: []string{sourceindex.CapabilityFiles, sourceindex.CapabilitySize, sourceindex.CapabilityDeclarations, sourceindex.CapabilityDocumentation, sourceindex.CapabilityVisibility},
		SyntaxProvider:        gosyntax.NewProvider(), Extractors: sourcefacts.NewRegistry(),
	})
	if err != nil {
		t.Fatalf("build policy source index: %v", err)
	}
	value, err := canonical.Normalize(analysis.AnalysisResult{Status: analysis.StatusComplete, Analyzer: analysis.AnalyzerInfo{ID: "analyzer:test", Version: "1.0.0", Language: "go", APIVersion: analysis.AnalyzerAPIVersion}, Project: analysis.ProjectInfo{RootLabel: "policy", Boundary: "repository"}, SourceIndex: &index, Modules: []analysis.ModuleObservation{}, References: []analysis.Reference{}, SourceReferences: []analysis.SourceReference{}, Relationships: []analysis.RelationshipObservation{}, Diagnostics: []analysis.Diagnostic{}})
	if err != nil {
		t.Fatalf("normalize policy model: %v", err)
	}
	report, err := qualityadapter.EvaluateModel(profile, value, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate policy model: %v", err)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("policy report findings = %#v", report.Findings)
	}
	withReport, err := canonical.WithQualityReport(value, &report)
	if err != nil {
		t.Fatalf("attach policy report: %v", err)
	}
	return withReport, report
}
