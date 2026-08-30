package viewer

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis/orchestration"
	goanalyzer "github.com/buffo/arch-view/internal/analyzers/go"
	"github.com/buffo/arch-view/internal/quality"
)

func TestQualityBridgeDiscoversProfilesAndEvaluatesLoadedModel(t *testing.T) {
	root := t.TempDir()
	writeViewerSourceFixture(t, root)
	value := sourceIndexedViewerModel(t, root)
	profile := bridgeFileSizeProfile("profile:human", 1)
	writeQualityBridgeProfile(t, root, "human.json", profile)

	server, err := NewServer(value, ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	profilesResponse, err := http.Get(httpServer.URL + "/v1/quality/profiles")
	if err != nil {
		t.Fatalf("GET quality profiles: %v", err)
	}
	var profiles qualityProfilesHTTPResponse
	if err := json.NewDecoder(profilesResponse.Body).Decode(&profiles); err != nil {
		_ = profilesResponse.Body.Close()
		t.Fatalf("decode quality profiles: %v", err)
	}
	_ = profilesResponse.Body.Close()
	if profilesResponse.StatusCode != http.StatusOK || profiles.Status != "available" || len(profiles.Profiles) != 1 {
		t.Fatalf("quality profiles response = %d %#v", profilesResponse.StatusCode, profiles)
	}
	if profiles.Profiles[0].ProfileID != profile.ProfileID || profiles.Profiles[0].ProfileVersion != profile.ProfileVersion || profiles.Profiles[0].Status != "available" {
		t.Fatalf("quality profile descriptor = %#v", profiles.Profiles[0])
	}

	rulesResponse, err := http.Get(httpServer.URL + "/v1/quality/rules?profile_id=" + url.QueryEscape(profile.ProfileID) + "&profile_version=" + url.QueryEscape(profile.ProfileVersion))
	if err != nil {
		t.Fatalf("GET quality rules: %v", err)
	}
	var rules qualityRulesHTTPResponse
	if err := json.NewDecoder(rulesResponse.Body).Decode(&rules); err != nil {
		_ = rulesResponse.Body.Close()
		t.Fatalf("decode quality rules: %v", err)
	}
	_ = rulesResponse.Body.Close()
	if rulesResponse.StatusCode != http.StatusOK || rules.Status != "available" || len(rules.Rules) < 15 {
		t.Fatalf("quality rules response = %d %#v", rulesResponse.StatusCode, rules)
	}
	var fileRule, signalRule *qualityRuleCatalogEntry
	for index := range rules.Rules {
		rule := &rules.Rules[index]
		switch rule.ID {
		case "source:file.max-lines":
			fileRule = rule
		case "signal:solid.srp":
			signalRule = rule
		}
	}
	if fileRule == nil || !fileRule.Enabled || !fileRule.Configured || signalRule == nil || signalRule.Enabled || signalRule.Configured {
		t.Fatalf("quality rule profile state = file %#v signal %#v", fileRule, signalRule)
	}

	body, err := json.Marshal(qualityEvaluationRequest{
		SchemaVersion:  qualityEvaluationRequestSchemaVersion,
		ProfileID:      profile.ProfileID,
		ProfileVersion: profile.ProfileVersion,
		Scope:          "all",
	})
	if err != nil {
		t.Fatalf("marshal quality evaluation request: %v", err)
	}
	evaluateResponse, err := http.Post(httpServer.URL+"/v1/quality/evaluate", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST quality evaluation: %v", err)
	}
	var evaluated qualityEvaluationHTTPResponse
	if err := json.NewDecoder(evaluateResponse.Body).Decode(&evaluated); err != nil {
		_ = evaluateResponse.Body.Close()
		t.Fatalf("decode quality evaluation: %v", err)
	}
	_ = evaluateResponse.Body.Close()
	if evaluateResponse.StatusCode != http.StatusOK || evaluated.Status != "available" || evaluated.ScopeID != "all" || evaluated.Report == nil {
		t.Fatalf("quality evaluation response = %d %#v", evaluateResponse.StatusCode, evaluated)
	}
	if evaluated.Report.ProfileID != profile.ProfileID || evaluated.Report.ProfileVersion != profile.ProfileVersion {
		t.Fatalf("evaluated report profile = %#v", evaluated.Report)
	}

	modelQualityURL := httpServer.URL + "/v1/models/" + url.PathEscape(value.ModelID) + "/quality"
	qualityResponse, err := http.Get(modelQualityURL)
	if err != nil {
		t.Fatalf("GET evaluated quality report: %v", err)
	}
	var stored qualityReportHTTPResponse
	if err := json.NewDecoder(qualityResponse.Body).Decode(&stored); err != nil {
		_ = qualityResponse.Body.Close()
		t.Fatalf("decode evaluated quality report: %v", err)
	}
	_ = qualityResponse.Body.Close()
	if qualityResponse.StatusCode != http.StatusOK || stored.Status != "available" || stored.Report == nil || stored.Report.EvaluationID != evaluated.Report.EvaluationID {
		t.Fatalf("stored quality report = %d %#v", qualityResponse.StatusCode, stored)
	}

	findingsResponse, err := http.Get(modelQualityURL + "/findings?limit=1")
	if err != nil {
		t.Fatalf("GET evaluated quality findings: %v", err)
	}
	var findings qualityFindingsHTTPResponse
	if err := json.NewDecoder(findingsResponse.Body).Decode(&findings); err != nil {
		_ = findingsResponse.Body.Close()
		t.Fatalf("decode evaluated quality findings: %v", err)
	}
	_ = findingsResponse.Body.Close()
	if findingsResponse.StatusCode != http.StatusOK || findings.Status != "available" || findings.ReportID != evaluated.Report.EvaluationID {
		t.Fatalf("evaluated quality findings = %d %#v", findingsResponse.StatusCode, findings)
	}

	temporary := quality.RuleBinding{
		RuleID:      "source:file.max-lines",
		RuleVersion: "1.0.0",
		Enabled:     false,
		Parameters: quality.TypedConfigBlock{
			Namespace:     "rule-config:source-file-size",
			SchemaVersion: "1.0.0",
			Payload:       map[string]any{"operator": quality.OperatorGreaterThan, "limit": 1, "unit": "unit:line"},
		},
		Severity: quality.SeverityWarning,
	}
	temporaryBody, err := json.Marshal(qualityEvaluationRequest{
		SchemaVersion:  qualityEvaluationRequestSchemaVersion,
		ProfileID:      profile.ProfileID,
		ProfileVersion: profile.ProfileVersion,
		Scope:          "all",
		RuleBindings:   []quality.RuleBinding{temporary},
	})
	if err != nil {
		t.Fatalf("marshal temporary quality evaluation request: %v", err)
	}
	temporaryResponse, err := http.Post(httpServer.URL+"/v1/quality/evaluate", "application/json", bytes.NewReader(temporaryBody))
	if err != nil {
		t.Fatalf("POST temporary quality evaluation: %v", err)
	}
	var temporaryEvaluated qualityEvaluationHTTPResponse
	if err := json.NewDecoder(temporaryResponse.Body).Decode(&temporaryEvaluated); err != nil {
		_ = temporaryResponse.Body.Close()
		t.Fatalf("decode temporary quality evaluation: %v", err)
	}
	_ = temporaryResponse.Body.Close()
	if temporaryResponse.StatusCode != http.StatusOK || temporaryEvaluated.Report == nil || !strings.Contains(temporaryEvaluated.Message, "not modified") {
		t.Fatalf("temporary quality evaluation = %d %#v", temporaryResponse.StatusCode, temporaryEvaluated)
	}
}

func TestQualityBridgeCreatesAndAttachesBaselineFromLoadedReport(t *testing.T) {
	root := t.TempDir()
	writeViewerSourceFixture(t, root)
	profile := bridgeFileSizeProfile("profile:human", 1)
	writeQualityBridgeProfile(t, root, "human.json", profile)
	server, err := NewServer(sourceIndexedViewerModel(t, root), ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	evaluationBody, err := json.Marshal(qualityEvaluationRequest{
		SchemaVersion:  qualityEvaluationRequestSchemaVersion,
		ProfileID:      profile.ProfileID,
		ProfileVersion: profile.ProfileVersion,
		Scope:          "all",
	})
	if err != nil {
		t.Fatalf("marshal evaluation request: %v", err)
	}
	evaluationResponse, err := http.Post(httpServer.URL+"/v1/quality/evaluate", "application/json", bytes.NewReader(evaluationBody))
	if err != nil {
		t.Fatalf("evaluate quality report: %v", err)
	}
	var evaluated qualityEvaluationHTTPResponse
	if err := json.NewDecoder(evaluationResponse.Body).Decode(&evaluated); err != nil {
		_ = evaluationResponse.Body.Close()
		t.Fatalf("decode quality report: %v", err)
	}
	_ = evaluationResponse.Body.Close()
	if evaluationResponse.StatusCode != http.StatusOK || evaluated.Report == nil || len(evaluated.Report.Findings) == 0 {
		t.Fatalf("initial quality report = %d %#v", evaluationResponse.StatusCode, evaluated)
	}

	baselineBody, err := json.Marshal(map[string]any{
		"schema_version":    "arch-view.quality-baseline-create/v1",
		"scope":             "all",
		"profile_id":        profile.ProfileID,
		"profile_version":   profile.ProfileVersion,
		"baseline_id":       "baseline:human",
		"revision":          "1.0.0",
		"file_name":         "human.json",
		"reason":            "accepted findings are tracked for remediation",
		"owner":             "architecture",
		"all_active":        true,
		"attach_to_profile": true,
	})
	if err != nil {
		t.Fatalf("marshal baseline request: %v", err)
	}
	createResponse := postQualityBaseline(t, server.Handler(), httpServer.URL+"/v1/quality/baselines/create", baselineBody)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create baseline status = %d body = %s", createResponse.Code, createResponse.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode baseline response: %v", err)
	}
	if created["status"] != "created" || created["baseline_file"] != "quality-baselines/human.json" {
		t.Fatalf("baseline response = %#v", created)
	}

	baselineData, err := os.ReadFile(filepath.Join(root, "quality-baselines", "human.json"))
	if err != nil {
		t.Fatalf("read generated baseline: %v", err)
	}
	var baseline quality.Baseline
	if err := json.Unmarshal(baselineData, &baseline); err != nil {
		t.Fatalf("decode generated baseline: %v", err)
	}
	if err := quality.ValidateBaseline(baseline); err != nil {
		t.Fatalf("validate generated baseline: %v", err)
	}
	if len(baseline.Entries) != 1 {
		t.Fatalf("generated baseline entries = %d, want 1", len(baseline.Entries))
	}

	profileData, err := os.ReadFile(filepath.Join(root, qualityProfileDirectoryName, "human.json"))
	if err != nil {
		t.Fatalf("read updated profile: %v", err)
	}
	var updatedProfile quality.QualityProfile
	if err := json.Unmarshal(profileData, &updatedProfile); err != nil {
		t.Fatalf("decode updated profile: %v", err)
	}
	if updatedProfile.Baseline == nil || updatedProfile.Baseline.BaselineID != baseline.BaselineID || updatedProfile.Baseline.Revision != baseline.Revision {
		t.Fatalf("updated profile baseline = %#v", updatedProfile.Baseline)
	}

	evaluationResponse, err = http.Post(httpServer.URL+"/v1/quality/evaluate", "application/json", bytes.NewReader(evaluationBody))
	if err != nil {
		t.Fatalf("re-evaluate quality report: %v", err)
	}
	if err := json.NewDecoder(evaluationResponse.Body).Decode(&evaluated); err != nil {
		_ = evaluationResponse.Body.Close()
		t.Fatalf("decode re-evaluated quality report: %v", err)
	}
	_ = evaluationResponse.Body.Close()
	if evaluationResponse.StatusCode != http.StatusOK || evaluated.Report == nil || evaluated.Report.Findings[0].Status != quality.StatusSuppressed {
		t.Fatalf("re-evaluated quality report = %d %#v", evaluationResponse.StatusCode, evaluated.Report)
	}

	conflict := postQualityBaseline(t, server.Handler(), httpServer.URL+"/v1/quality/baselines/create", baselineBody)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("duplicate baseline status = %d body = %s", conflict.Code, conflict.Body.String())
	}
}

func postQualityBaseline(t *testing.T, handler http.Handler, endpoint string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestQualityBridgeListsInvalidProfilesWithoutMakingThemSelectable(t *testing.T) {
	root := t.TempDir()
	profileDir := filepath.Join(root, qualityProfileDirectoryName)
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatalf("create profile directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, "broken.json"), []byte(`{"schema_version":"arch-view.quality/v1","profile_id":"bad"}`), 0o644); err != nil {
		t.Fatalf("write invalid profile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, "ignored.txt"), []byte("not a profile"), 0o644); err != nil {
		t.Fatalf("write ignored file: %v", err)
	}

	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	response, err := http.Get(httpServer.URL + "/v1/quality/profiles")
	if err != nil {
		t.Fatalf("GET invalid quality profiles: %v", err)
	}
	var profiles qualityProfilesHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&profiles); err != nil {
		_ = response.Body.Close()
		t.Fatalf("decode invalid quality profiles: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || profiles.Status != "invalid" || len(profiles.Profiles) != 1 || profiles.Profiles[0].Status != "invalid" || profiles.Profiles[0].ProfileID != "" {
		t.Fatalf("invalid quality profiles response = %d %#v", response.StatusCode, profiles)
	}
}

func TestQualityBridgeSavesExistingAndNewQualityProfiles(t *testing.T) {
	root := t.TempDir()
	writeViewerSourceFixture(t, root)
	profile := bridgeFileSizeProfile("profile:human", 500)
	writeQualityBridgeProfile(t, root, "human.json", profile)
	server, err := NewServer(sourceIndexedViewerModel(t, root), ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	bindings := cloneQualityRuleBindings(profile.EnabledRules)
	bindings[0].Enabled = false
	saveBody, err := json.Marshal(map[string]any{
		"schema_version":  "arch-view.quality-profile-save/v1",
		"profile_id":      profile.ProfileID,
		"profile_version": profile.ProfileVersion,
		"rule_bindings":   bindings,
	})
	if err != nil {
		t.Fatalf("marshal existing profile save: %v", err)
	}
	saveResponse := putQualityProfile(t, server.Handler(), httpServer.URL+"/v1/quality/profiles/save", saveBody)
	if saveResponse.Code != http.StatusOK {
		t.Fatalf("save existing profile status = %d body = %s", saveResponse.Code, saveResponse.Body.String())
	}
	var saved quality.QualityProfile
	data, err := os.ReadFile(filepath.Join(root, qualityProfileDirectoryName, "human.json"))
	if err != nil {
		t.Fatalf("read saved existing profile: %v", err)
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("decode saved existing profile: %v", err)
	}
	if len(saved.EnabledRules) != 1 || saved.EnabledRules[0].Enabled {
		t.Fatalf("saved existing profile rules = %#v", saved.EnabledRules)
	}

	newProfile := profile
	newProfile.ProfileID = "profile:review"
	newProfile.EnabledRules = profile.EnabledRules
	newBody, err := json.Marshal(map[string]any{
		"schema_version":         "arch-view.quality-profile-save/v1",
		"profile_id":             newProfile.ProfileID,
		"profile_version":        newProfile.ProfileVersion,
		"source_profile_id":      profile.ProfileID,
		"source_profile_version": profile.ProfileVersion,
		"file_name":              "review.json",
		"rule_bindings":          profile.EnabledRules,
	})
	if err != nil {
		t.Fatalf("marshal new profile save: %v", err)
	}
	newResponse := putQualityProfile(t, server.Handler(), httpServer.URL+"/v1/quality/profiles/save-as", newBody)
	if newResponse.Code != http.StatusCreated {
		t.Fatalf("save new profile status = %d body = %s", newResponse.Code, newResponse.Body.String())
	}
	newData, err := os.ReadFile(filepath.Join(root, qualityProfileDirectoryName, "review.json"))
	if err != nil {
		t.Fatalf("read saved new profile: %v", err)
	}
	var created quality.QualityProfile
	if err := json.Unmarshal(newData, &created); err != nil {
		t.Fatalf("decode saved new profile: %v", err)
	}
	if created.ProfileID != newProfile.ProfileID || len(created.EnabledRules) != 1 || !created.EnabledRules[0].Enabled {
		t.Fatalf("saved new profile = %#v", created)
	}
	duplicateBody := append([]byte(nil), newBody...)
	duplicateResponse := putQualityProfile(t, server.Handler(), httpServer.URL+"/v1/quality/profiles/save-as", duplicateBody)
	if duplicateResponse.Code != http.StatusConflict {
		t.Fatalf("duplicate save-as status = %d body = %s", duplicateResponse.Code, duplicateResponse.Body.String())
	}
	afterDuplicate, err := os.ReadFile(filepath.Join(root, qualityProfileDirectoryName, "review.json"))
	if err != nil {
		t.Fatalf("read profile after duplicate save-as: %v", err)
	}
	if !bytes.Equal(newData, afterDuplicate) {
		t.Fatal("duplicate save-as overwrote the existing quality profile")
	}
}

func putQualityProfile(t *testing.T, handler http.Handler, endpoint string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPut, endpoint, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	// The helper is intentionally kept at the HTTP seam so the save contract is
	// exercised through the same mux used by the real viewer.
	handler.ServeHTTP(response, request)
	return response
}

func TestQualityBridgeEvaluatesConcreteAggregateScopeWithoutMutatingAggregateIdentity(t *testing.T) {
	root := t.TempDir()
	writeViewerSourceFixture(t, root)
	result := sourceIndexedAnalysisResult(t, root)
	manifest := goanalyzer.New().Manifest()
	job := orchestration.AnalyzerJob{
		ScopeID:             "scope-bridge",
		ProjectRoot:         root,
		RelativeProjectRoot: ".",
		LogicalAnalyzerID:   manifest.ID,
		AnalyzerVersion:     manifest.Version,
		Language:            manifest.Language,
		Status:              orchestration.JobComplete,
		Result:              &result,
		Manifest:            manifest,
	}
	plan := orchestration.JobPlan{PlanVersion: orchestration.JobPlanSchemaVersion, RepositoryRoot: root, InvocationRoot: ".", DiscoveryPolicyVersion: orchestration.DiscoveryPolicyVersion, SourceScopePolicy: orchestration.SourceScopePolicy{PolicyVersion: orchestration.SourceScopePolicyVersion, InvocationRoot: "."}, Jobs: []orchestration.AnalyzerJob{job}}
	run, err := orchestration.AggregateScopeResults(orchestration.ExecutionSnapshot{RunID: "run-quality-bridge", Plan: plan, Jobs: []orchestration.AnalyzerJob{job}})
	if err != nil {
		t.Fatalf("aggregate quality bridge fixture: %v", err)
	}
	if run.Model == nil {
		t.Fatalf("aggregate fixture has no combined model")
	}
	originalModelID := run.Model.ModelID
	profile := bridgeFileSizeProfile("profile:aggregate", 1)
	writeQualityBridgeProfile(t, root, "aggregate.json", profile)
	server, err := NewAggregateServer(run, ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatalf("NewAggregateServer() error = %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	body, err := json.Marshal(qualityEvaluationRequest{SchemaVersion: qualityEvaluationRequestSchemaVersion, ProfileID: profile.ProfileID, ProfileVersion: profile.ProfileVersion, Scope: "scope-bridge"})
	if err != nil {
		t.Fatalf("marshal aggregate evaluation request: %v", err)
	}
	evaluateResponse, err := http.Post(httpServer.URL+"/v1/quality/evaluate", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST aggregate quality evaluation: %v", err)
	}
	var evaluated qualityEvaluationHTTPResponse
	if err := json.NewDecoder(evaluateResponse.Body).Decode(&evaluated); err != nil {
		_ = evaluateResponse.Body.Close()
		t.Fatalf("decode aggregate quality evaluation: %v", err)
	}
	_ = evaluateResponse.Body.Close()
	if evaluateResponse.StatusCode != http.StatusOK || evaluated.ScopeID != "scope-bridge" || evaluated.Report == nil {
		t.Fatalf("aggregate quality evaluation = %d %#v", evaluateResponse.StatusCode, evaluated)
	}
	if run.Model.ModelID != originalModelID {
		t.Fatalf("viewer quality evaluation mutated aggregate identity: got %q want %q", run.Model.ModelID, originalModelID)
	}
	qualityURL := httpServer.URL + "/v1/models/" + url.PathEscape(originalModelID) + "/quality?scope=scope-bridge"
	qualityResponse, err := http.Get(qualityURL)
	if err != nil {
		t.Fatalf("GET concrete aggregate quality report: %v", err)
	}
	var stored qualityReportHTTPResponse
	if err := json.NewDecoder(qualityResponse.Body).Decode(&stored); err != nil {
		_ = qualityResponse.Body.Close()
		t.Fatalf("decode concrete aggregate quality report: %v", err)
	}
	_ = qualityResponse.Body.Close()
	if qualityResponse.StatusCode != http.StatusOK || stored.Status != "available" || stored.Report == nil || stored.Report.ProfileID != profile.ProfileID {
		t.Fatalf("concrete aggregate quality report = %d %#v", qualityResponse.StatusCode, stored)
	}
}

func bridgeFileSizeProfile(id string, limit int) quality.QualityProfile {
	return quality.QualityProfile{
		SchemaVersion:  quality.SchemaVersion,
		ProfileID:      id,
		ProfileVersion: "1.0.0",
		EnabledRules: []quality.RuleBinding{{
			RuleID:      "source:file.max-lines",
			RuleVersion: "1.0.0",
			Enabled:     true,
			Parameters: quality.TypedConfigBlock{
				Namespace:     "rule-config:source-file-size",
				SchemaVersion: "1.0.0",
				Payload:       map[string]any{"operator": quality.OperatorGreaterThan, "limit": limit, "unit": "unit:line"},
			},
		}},
		SeverityPolicy: quality.TypedConfigBlock{Namespace: "severity:default", SchemaVersion: "1.0.0", Payload: map[string]any{}},
		Constraints:    []quality.ArchitectureConstraint{},
		Extensions:     []quality.ExtensionBlock{},
	}
}

func writeQualityBridgeProfile(t *testing.T, root, name string, profile quality.QualityProfile) {
	t.Helper()
	directory := filepath.Join(root, qualityProfileDirectoryName)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatalf("create quality profile directory: %v", err)
	}
	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("marshal quality profile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(directory, name), data, 0o644); err != nil {
		t.Fatalf("write quality profile: %v", err)
	}
}

// Keep the test's response bodies bounded if a future failure writes an
// unexpected HTML page instead of JSON.
