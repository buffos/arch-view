package live

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/buffo/arch-view/internal/quality"
	qualitypolicy "github.com/buffo/arch-view/internal/quality/policy"
)

func TestMCPServeReturnsProtocolMessagesWithoutStdoutNoise(t *testing.T) {
	session := newTransportTestSession(t, "session:mcp")
	server := NewMCPServer(NewLocalQueryAdapter(session))
	input := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_snapshot_status","arguments":{"max_items":10,"max_bytes":4096}}}
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"unknown_tool","arguments":{}}}
`)
	var output bytes.Buffer
	if err := server.Serve(context.Background(), input, &output); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("MCP output lines = %d, output=%q", len(lines), output.String())
	}
	var initialize map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &initialize); err != nil {
		t.Fatal(err)
	}
	if initialize["jsonrpc"] != "2.0" || initialize["id"].(float64) != 1 || initialize["result"].(map[string]any)["serverInfo"].(map[string]any)["name"] != MCPServerName {
		t.Fatalf("initialize response = %#v", initialize)
	}
	var status map[string]any
	if err := json.Unmarshal([]byte(lines[2]), &status); err != nil {
		t.Fatal(err)
	}
	structured := status["result"].(map[string]any)["structuredContent"].(map[string]any)
	if structured["schema_version"] != QuerySchemaVersion || structured["session_id"] != "session:mcp" || structured["revision"].(float64) != 1 {
		t.Fatalf("status tool envelope = %#v", structured)
	}
	var unknown map[string]any
	if err := json.Unmarshal([]byte(lines[3]), &unknown); err != nil {
		t.Fatal(err)
	}
	if unknown["result"].(map[string]any)["isError"] != true {
		t.Fatalf("unknown tool response = %#v", unknown)
	}
}

func TestMCPResourcesAndMalformedRequestsAreStructured(t *testing.T) {
	session := newTransportTestSession(t, "session:mcp-resources")
	server := NewMCPServer(NewLocalQueryAdapter(session))
	input := strings.NewReader("not-json\n{" + `"jsonrpc":"2.0","id":1,"method":"resources/list"` + "}\n" + `{"jsonrpc":"2.0","id":2,"method":"resources/read","params":{"uri":"archview://session/session:mcp-resources/status"}}` + "\n")
	var output bytes.Buffer
	if err := server.Serve(context.Background(), input, &output); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("resource output lines = %d, output=%q", len(lines), output.String())
	}
	var parseError map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &parseError); err != nil {
		t.Fatal(err)
	}
	if parseError["error"].(map[string]any)["code"].(float64) != -32700 {
		t.Fatalf("parse error = %#v", parseError)
	}
	var resources map[string]any
	if err := json.Unmarshal([]byte(lines[2]), &resources); err != nil {
		t.Fatal(err)
	}
	if len(resources["result"].(map[string]any)["contents"].([]any)) != 1 {
		t.Fatalf("resource response = %#v", resources)
	}
}

func TestMCPPositiveIntegerParsingIsStrict(t *testing.T) {
	if value, err := parsePositiveInt("12"); err != nil || value != 12 {
		t.Fatalf("valid integer = %d, %v", value, err)
	}
	if _, err := parsePositiveInt("12junk"); err == nil {
		t.Fatal("integer parser accepted trailing text")
	}
}

func TestMCPOversizedMessageReturnsStructuredError(t *testing.T) {
	server := NewMCPServer(NewLocalQueryAdapter(newTransportTestSession(t, "session:mcp-large")))
	server.MaxMessageBytes = 16
	var output bytes.Buffer
	if err := server.Serve(context.Background(), strings.NewReader(strings.Repeat("x", 64)+"\n"), &output); err != nil {
		t.Fatal(err)
	}
	var response map[string]any
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("oversized response = %q: %v", output.String(), err)
	}
	if response["error"].(map[string]any)["code"].(float64) != -32600 {
		t.Fatalf("oversized response = %#v", response)
	}
}

func TestMCPRejectsUnknownToolArgumentsAndAdvertisesClosedSchemas(t *testing.T) {
	session := newTransportTestSession(t, "session:mcp-schema")
	server := NewMCPServer(NewLocalQueryAdapter(session))
	input := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n" + `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"find_files","arguments":{"max_itmes":10}}}` + "\n")
	var output bytes.Buffer
	if err := server.Serve(context.Background(), input, &output); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	var listed map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &listed); err != nil {
		t.Fatal(err)
	}
	tools := listed["result"].(map[string]any)["tools"].([]any)
	if len(tools) == 0 {
		t.Fatal("MCP tool catalog is empty")
	}
	schema := tools[0].(map[string]any)["inputSchema"].(map[string]any)
	if schema["additionalProperties"] != false || schema["properties"] == nil {
		t.Fatalf("MCP tool schema = %#v", schema)
	}
	var rejected map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &rejected); err != nil {
		t.Fatal(err)
	}
	result := rejected["result"].(map[string]any)
	if result["isError"] != true || !strings.Contains(result["structuredContent"].(map[string]any)["message"].(string), "invalid") {
		t.Fatalf("unknown tool argument response = %#v", rejected)
	}
}

func TestMCPServeClosesBlockingInputOnCancellation(t *testing.T) {
	session := newTransportTestSession(t, "session:mcp-cancel")
	reader, writer := io.Pipe()
	t.Cleanup(func() { _ = writer.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- NewMCPServer(NewLocalQueryAdapter(session)).Serve(ctx, reader, io.Discard)
	}()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Serve cancellation error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("MCP Serve remained blocked after cancellation")
	}
}

func TestMCPQualityBaselinesCanBeReadSelectedAndAppended(t *testing.T) {
	root := t.TempDir()
	profile := policyThresholdProfile()
	store, err := qualitypolicy.NewFileStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveProfile(context.Background(), profile, "main.json", false); err != nil {
		t.Fatalf("save profile: %v", err)
	}
	modelValue, report := policyModelAndReport(t, profile)
	config := testLiveConfig("session:mcp-baselines")
	config.SourceIndexRequest = SourceIndexRequest{Enabled: true}
	config.QualityRequest = &QualityRequest{ProfileID: profile.ProfileID, ProfileVersion: profile.ProfileVersion}
	config.PermissionPolicy.AllowedOperations = []Operation{OperationBaselineRead, OperationBaselineWrite, OperationQualityEvaluate, OperationQualityProfileRead}
	session, err := StartLiveSession(context.Background(), config, root, SessionOptions{
		Scanner: StaticScanner{Result: ScanResult{Model: modelValue, QualityReport: &report}}, Fingerprinter: StaticFingerprinter{Value: testInput("mcp-baselines")},
		QualityCatalog: quality.NewDefaultCatalog(), Profiles: NewMemoryQualityProfileResolver(profile),
		PolicyService:    NewFileQualityPolicyService(qualitypolicy.NewService(store, quality.NewDefaultCatalog())),
		PolicyAuthorizer: PolicyAuthorizerFunc(func(context.Context, PolicyAuthorization) error { return nil }),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	if err := session.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	server := NewMCPServer(NewLocalQueryAdapter(session))
	before, err := server.executeTool(context.Background(), "get_quality_baselines", json.RawMessage(`{"max_items":10}`))
	if err != nil {
		t.Fatalf("list baselines before append: %v", err)
	}
	if result, ok := before.Result.(QualityBaselinesResult); !ok || result.Total != 0 {
		t.Fatalf("baseline list before append = %#v", before.Result)
	}
	key := report.Findings[0].FindingKey
	appended, err := server.executeTool(context.Background(), "append_baseline", mustJSON(t, map[string]any{
		"profile_id": profile.ProfileID, "profile_version": profile.ProfileVersion, "file_name": "main.json", "finding_keys": []string{key}, "reason": "accepted after agent review", "authorization": "allow",
	}))
	if err != nil {
		t.Fatalf("append baseline: %v", err)
	}
	appendResult, ok := appended.Result.(QualityPolicyResult)
	if !ok || appendResult.Status != "updated" || appendResult.Baseline == nil || appendResult.Baseline.Revision != "1.0.0" || len(appendResult.AddedFindingKeys) != 1 || !appendResult.ReevaluationRequired {
		t.Fatalf("append result = %#v", appended.Result)
	}
	read, err := server.executeTool(context.Background(), "get_quality_baselines", mustJSON(t, map[string]any{"file_names": []string{"main.json"}, "include_entries": true, "max_items": 1}))
	if err != nil {
		t.Fatalf("read baseline entries: %v", err)
	}
	readResult, ok := read.Result.(QualityBaselinesResult)
	if !ok || readResult.Total != 1 || len(readResult.Items) != 1 || len(readResult.Items[0].Entries) != 1 || readResult.Items[0].Entries[0].FindingKey != key {
		t.Fatalf("baseline entry result = %#v", read.Result)
	}
	if _, err := server.executeTool(context.Background(), "get_quality_baselines", mustJSON(t, map[string]any{"file_names": []string{"../main.json"}})); err == nil || !strings.Contains(err.Error(), "direct project-local JSON") {
		t.Fatalf("unsafe baseline filename error = %v", err)
	}
	profileEvaluation, err := server.executeTool(context.Background(), "evaluate_quality", mustJSON(t, map[string]any{
		"profile_id": profile.ProfileID, "profile_version": profile.ProfileVersion,
	}))
	if err != nil {
		t.Fatalf("profile baseline evaluation: %v", err)
	}
	profileEvaluationResult, ok := profileEvaluation.Result.(QualityEvaluationResult)
	if !ok || profileEvaluationResult.Report == nil || len(profileEvaluationResult.Report.Findings) != 1 || profileEvaluationResult.Report.Findings[0].Status != quality.StatusSuppressed {
		t.Fatalf("profile baseline evaluation = %#v", profileEvaluation.Result)
	}
	evaluated, err := server.executeTool(context.Background(), "evaluate_quality", mustJSON(t, map[string]any{
		"profile_id": profile.ProfileID, "profile_version": profile.ProfileVersion, "baseline_mode": string(BaselineModeSelected), "baseline_files": []string{"main.json"},
	}))
	if err != nil {
		var liveErr *QueryError
		if errors.As(err, &liveErr) {
			t.Fatalf("selected baseline evaluation: %v details=%#v", err, liveErr.Details)
		}
		t.Fatalf("selected baseline evaluation: %v", err)
	}
	evaluationResult, ok := evaluated.Result.(QualityEvaluationResult)
	if !ok || evaluationResult.Report == nil || len(evaluationResult.Report.Findings) != 1 || evaluationResult.Report.Findings[0].Status != quality.StatusSuppressed {
		t.Fatalf("selected baseline evaluation = %#v", evaluated.Result)
	}
	cleanEvaluation, err := server.executeTool(context.Background(), "evaluate_quality", mustJSON(t, map[string]any{
		"profile_id": profile.ProfileID, "profile_version": profile.ProfileVersion, "baseline_mode": string(BaselineModeNone),
	}))
	if err != nil {
		t.Fatalf("clean evaluation: %v", err)
	}
	cleanResult, ok := cleanEvaluation.Result.(QualityEvaluationResult)
	if !ok || cleanResult.Report == nil || len(cleanResult.Report.Findings) != 1 || cleanResult.Report.Findings[0].Status == quality.StatusSuppressed {
		t.Fatalf("clean evaluation = %#v", cleanEvaluation.Result)
	}
	appendedAgain, err := server.executeTool(context.Background(), "append_baseline", mustJSON(t, map[string]any{
		"profile_id": profile.ProfileID, "profile_version": profile.ProfileVersion, "report_id": cleanResult.Report.EvaluationID,
		"file_name": "main.json", "finding_keys": []string{key}, "reason": "accepted after agent review", "authorization": "allow",
	}))
	if err != nil {
		t.Fatalf("append baseline from clean evaluation: %v", err)
	}
	appendedAgainResult, ok := appendedAgain.Result.(QualityPolicyResult)
	if !ok || appendedAgainResult.Status != "unchanged" || appendedAgainResult.ReevaluationRequired {
		t.Fatalf("append from clean evaluation = %#v", appendedAgain.Result)
	}
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
