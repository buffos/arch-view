package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestLiveCommandUsageAndValidation(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"live", "--help"}, &stdout, &stderr); code != 2 {
		t.Fatalf("live help exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), "live start") || !strings.Contains(stdout.String(), "ensure-current") {
		t.Fatalf("live usage = %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"live", "status", "--endpoint", "http://127.0.0.1:1"}, &stdout, &stderr); code == 0 {
		t.Fatal("status unexpectedly succeeded against an unavailable endpoint")
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"live", "wait", "--endpoint", "http://127.0.0.1:1", "--consistency", "invalid"}, &stdout, &stderr); code != 2 {
		t.Fatalf("invalid wait consistency exit code = %d", code)
	}

	stdout.Reset()
	stderr.Reset()
	missing := filepath.Join(t.TempDir(), "missing")
	if code := run([]string{"live", "start", "--project", missing}, &stdout, &stderr); code == 0 {
		t.Fatal("live start unexpectedly accepted a missing project root")
	}
	if strings.Contains(stdout.String(), "session_id") {
		t.Fatalf("invalid live start exposed a session: %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"live", "start", "--project", ".", "--allow-policy-writes"}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "policy-token") {
		t.Fatalf("missing policy authorization: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"open", "--live", "--project", missing, "--no-watch", "--no-source-index", "--session-id", "session:flags"}, &stdout, &stderr); code == 2 && strings.Contains(stderr.String(), "flag provided but not defined") {
		t.Fatalf("open --live rejected its live-only flags: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestLiveLatestReadyWaitPollsUntilARevisionExists(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		calls++
		writer.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			_, _ = writer.Write([]byte(`{"schema_version":"arch-view.query/v1","returned_consistency":"unavailable","freshness":{"status":"initializing"},"result":{"state":"initializing"}}`))
			return
		}
		_, _ = writer.Write([]byte(`{"schema_version":"arch-view.query/v1","revision":3,"returned_consistency":"latest_ready","freshness":{"status":"current"},"result":{"state":"ready"}}`))
	}))
	t.Cleanup(server.Close)

	var stdout, stderr bytes.Buffer
	if code := runLiveWait([]string{"--endpoint", server.URL, "--consistency", "latest_ready", "--timeout", "1s"}, false, &stdout, &stderr); code != 0 {
		t.Fatalf("latest-ready wait code=%d stderr=%q", code, stderr.String())
	}
	if calls != 2 || !strings.Contains(stdout.String(), `"revision":3`) {
		t.Fatalf("latest-ready wait calls=%d output=%q", calls, stdout.String())
	}

	neverReady := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"schema_version":"arch-view.query/v1","returned_consistency":"unavailable","freshness":{"status":"initializing"},"result":{"state":"initializing"}}`))
	}))
	t.Cleanup(neverReady.Close)
	stdout.Reset()
	stderr.Reset()
	if code := runLiveWait([]string{"--endpoint", neverReady.URL, "--consistency", "latest_ready", "--timeout", "20ms"}, false, &stdout, &stderr); code == 0 || !strings.Contains(stderr.String(), context.DeadlineExceeded.Error()) {
		t.Fatalf("timed wait code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestLiveStatusAndWaitCommandsUseEndpointResponses(t *testing.T) {
	var statusCalls, ensureCalls int
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/status":
			statusCalls++
			_, _ = writer.Write([]byte(`{"schema_version":"arch-view.query/v1","revision":4,"freshness":{"status":"current"}}`))
		case "/ensure-current":
			ensureCalls++
			if request.Method != http.MethodPost {
				writer.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			_, _ = writer.Write([]byte(`{"schema_version":"arch-view.query/v1","revision":5,"returned_consistency":"current"}`))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	var stdout, stderr bytes.Buffer
	if code := runLiveStatus([]string{"--endpoint", server.URL}, &stdout, &stderr); code != 0 {
		t.Fatalf("live status exit code = %d stderr=%s", code, stderr.String())
	}
	var status struct {
		HTTPStatus int `json:"http_status"`
		Response   struct {
			Revision int `json:"revision"`
		} `json:"response"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
		t.Fatalf("decode live status: %v; output=%q", err, stdout.String())
	}
	if status.HTTPStatus != http.StatusOK || status.Response.Revision != 4 || statusCalls != 1 {
		t.Fatalf("live status response = %#v calls=%d", status, statusCalls)
	}

	stdout.Reset()
	stderr.Reset()
	if code := runLiveWait([]string{"--endpoint", server.URL, "--consistency", "require_current", "--timeout", "1s"}, true, &stdout, &stderr); code != 0 {
		t.Fatalf("live ensure-current exit code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"revision":5`) || ensureCalls != 1 {
		t.Fatalf("live ensure-current response = %q calls=%d", stdout.String(), ensureCalls)
	}

	stdout.Reset()
	stderr.Reset()
	if code := runLiveStatus([]string{"--endpoint", server.URL + "/status"}, &stdout, &stderr); code != 0 {
		t.Fatalf("live status should accept the copied status endpoint: code=%d stderr=%s", code, stderr.String())
	}
	if statusCalls != 2 {
		t.Fatalf("normalized status endpoint calls = %d, want 2", statusCalls)
	}
}

func TestMCPCommandRejectsUnsafeHTTPConfiguration(t *testing.T) {
	host, err := newBuiltInHost()
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := runMCPCommand(host, []string{"--project", project, "--transport", "authenticated_http"}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "auth-token") {
		t.Fatalf("missing HTTP authentication: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runMCPCommand(host, []string{"--project", project, "--transport", "unknown"}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "unsupported MCP transport") {
		t.Fatalf("invalid MCP transport: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
