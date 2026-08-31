package live

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestHTTPTransportPreservesQueryEnvelopeAndRejectsUnsafeRequests(t *testing.T) {
	session := newTransportTestSession(t, "session:http")
	server := httptest.NewServer(NewHTTPServer(NewLocalQueryAdapter(session), HTTPServerOptions{Transport: TransportLocalHTTP, SessionID: "session:http", MaxRequestBytes: 128, MaxResponseBytes: 64 * 1024}).Handler())
	t.Cleanup(server.Close)
	base := server.URL + "/v1/live/session:http"

	response, err := http.Get(base + "/status")
	if err != nil {
		t.Fatal(err)
	}
	body := readHTTPBody(t, response)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d, body=%s", response.StatusCode, body)
	}
	var envelope QueryEnvelope
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.SchemaVersion != QuerySchemaVersion || envelope.SessionID != "session:http" || envelope.Revision != 1 || envelope.ReturnedConsistency != string(ConsistencyLatestReady) {
		t.Fatalf("HTTP status envelope = %#v", envelope)
	}

	unsafe := postHTTP(t, base+"/search/files", `{"query":{"path_prefix":"../outside"}}`, nil)
	if unsafe.StatusCode != http.StatusBadRequest || !strings.Contains(unsafe.Body, "QueryInvalid") {
		t.Fatalf("unsafe path response = %#v", unsafe)
	}
	wrongSession := postHTTP(t, server.URL+"/v1/live/other/search/files", `{}`, nil)
	if wrongSession.StatusCode != http.StatusNotFound {
		t.Fatalf("wrong session response = %#v", wrongSession)
	}
	tooLarge := postHTTP(t, base+"/search/files", `{"query":{"name":"`+strings.Repeat("x", 200)+`"}}`, nil)
	if tooLarge.StatusCode < http.StatusBadRequest || tooLarge.StatusCode >= http.StatusInternalServerError {
		t.Fatalf("oversized request response = %#v", tooLarge)
	}
}

func TestAuthenticatedHTTPRequiresTokenAndOrigin(t *testing.T) {
	session := newTransportTestSession(t, "session:auth-http")
	handler := NewHTTPServer(NewLocalQueryAdapter(session), HTTPServerOptions{Transport: TransportAuthenticatedHTTP, AuthToken: "secret", AllowedOrigins: []string{"http://allowed"}, SessionID: "session:auth-http"}).Handler()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	endpoint := server.URL + "/v1/live/session:auth-http/status"

	for name, headers := range map[string]map[string]string{
		"missing token": {"Origin": "http://allowed"},
		"wrong token":   {"Origin": "http://allowed", "Authorization": "Bearer wrong"},
		"wrong origin":  {"Origin": "http://other", "Authorization": "Bearer secret"},
	} {
		t.Run(name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodGet, endpoint, nil)
			if err != nil {
				t.Fatal(err)
			}
			for key, value := range headers {
				request.Header.Set(key, value)
			}
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = response.Body.Close() }()
			if response.StatusCode != http.StatusForbidden {
				t.Fatalf("status = %d", response.StatusCode)
			}
			if headers["Origin"] == "http://allowed" && response.Header.Get("Access-Control-Allow-Origin") != "http://allowed" {
				t.Fatalf("authorized origin could not read the authentication error: %q", response.Header.Get("Access-Control-Allow-Origin"))
			}
		})
	}

	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer secret")
	request.Header.Set("Origin", "http://allowed")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK || response.Header.Get("Access-Control-Allow-Origin") != "http://allowed" {
		t.Fatalf("authorized response status=%d origin=%q", response.StatusCode, response.Header.Get("Access-Control-Allow-Origin"))
	}
}

func TestHTTPOriginPolicyDefaultsToSameOriginAndAllowsAuthenticatedPreflight(t *testing.T) {
	session := newTransportTestSession(t, "session:origin")
	local := httptest.NewServer(NewHTTPServer(NewLocalQueryAdapter(session), HTTPServerOptions{Transport: TransportLocalHTTP, SessionID: "session:origin"}).Handler())
	t.Cleanup(local.Close)
	endpoint := local.URL + "/v1/live/session:origin/status"

	crossOrigin, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	crossOrigin.Header.Set("Origin", "https://attacker.example")
	response, err := http.DefaultClient.Do(crossOrigin)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusForbidden || response.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("cross-origin local response status=%d allow-origin=%q", response.StatusCode, response.Header.Get("Access-Control-Allow-Origin"))
	}

	sameOrigin, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	sameOrigin.Header.Set("Origin", local.URL)
	response, err = http.DefaultClient.Do(sameOrigin)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Access-Control-Allow-Origin") != local.URL {
		t.Fatalf("same-origin response status=%d allow-origin=%q", response.StatusCode, response.Header.Get("Access-Control-Allow-Origin"))
	}

	authenticated := httptest.NewServer(NewHTTPServer(NewLocalQueryAdapter(session), HTTPServerOptions{Transport: TransportAuthenticatedHTTP, AuthToken: "secret", AllowedOrigins: []string{"https://client.example"}, SessionID: "session:origin"}).Handler())
	t.Cleanup(authenticated.Close)
	preflight, err := http.NewRequest(http.MethodOptions, authenticated.URL+"/v1/live/session:origin/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	preflight.Header.Set("Origin", "https://client.example")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodGet)
	response, err = http.DefaultClient.Do(preflight)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNoContent || response.Header.Get("Access-Control-Allow-Origin") != "https://client.example" {
		t.Fatalf("preflight response status=%d allow-origin=%q", response.StatusCode, response.Header.Get("Access-Control-Allow-Origin"))
	}
}

func TestHTTPAndMCPStatusHaveEquivalentSemanticEnvelope(t *testing.T) {
	session := newTransportTestSession(t, "session:parity")
	httpServer := httptest.NewServer(NewHTTPServer(NewLocalQueryAdapter(session), HTTPServerOptions{Transport: TransportLocalHTTP, SessionID: "session:parity"}).Handler())
	t.Cleanup(httpServer.Close)
	response, err := http.Get(httpServer.URL + "/v1/live/session:parity/status")
	if err != nil {
		t.Fatal(err)
	}
	httpBody := readHTTPBody(t, response)
	var httpEnvelope QueryEnvelope
	if err := json.Unmarshal([]byte(httpBody), &httpEnvelope); err != nil {
		t.Fatal(err)
	}

	input := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_snapshot_status","arguments":{}}}` + "\n")
	var output bytes.Buffer
	if err := NewMCPServer(NewLocalQueryAdapter(session)).Serve(context.Background(), input, &output); err != nil {
		t.Fatal(err)
	}
	var mcp map[string]any
	if err := json.Unmarshal(output.Bytes(), &mcp); err != nil {
		t.Fatal(err)
	}
	structured := mcp["result"].(map[string]any)["structuredContent"].(map[string]any)
	if structured["schema_version"] != httpEnvelope.SchemaVersion || structured["session_id"] != httpEnvelope.SessionID || int(structured["revision"].(float64)) != httpEnvelope.Revision || structured["returned_consistency"] != httpEnvelope.ReturnedConsistency {
		t.Fatalf("HTTP=%#v MCP=%#v", httpEnvelope, structured)
	}
}

func TestHTTPAndMCPFileQueriesHaveEquivalentSemanticResults(t *testing.T) {
	session := newTransportTestSession(t, "session:query-parity")
	httpServer := httptest.NewServer(NewHTTPServer(NewLocalQueryAdapter(session), HTTPServerOptions{Transport: TransportLocalHTTP, SessionID: "session:query-parity"}).Handler())
	t.Cleanup(httpServer.Close)
	httpResponse := postHTTP(t, httpServer.URL+"/v1/live/session:query-parity/search/files", `{"max_items":10,"max_bytes":65536}`, nil)
	if httpResponse.StatusCode != http.StatusOK {
		t.Fatalf("HTTP file query status=%d body=%s", httpResponse.StatusCode, httpResponse.Body)
	}
	var httpEnvelope map[string]any
	if err := json.Unmarshal([]byte(httpResponse.Body), &httpEnvelope); err != nil {
		t.Fatal(err)
	}

	input := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"find_files","arguments":{"max_items":10,"max_bytes":65536}}}` + "\n")
	var output bytes.Buffer
	if err := NewMCPServer(NewLocalQueryAdapter(session)).Serve(context.Background(), input, &output); err != nil {
		t.Fatal(err)
	}
	var response map[string]any
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	mcpEnvelope := response["result"].(map[string]any)["structuredContent"].(map[string]any)
	for _, field := range []string{"schema_version", "session_id", "snapshot_id", "revision", "requested_consistency", "returned_consistency", "scope_ids", "result", "result_count", "omitted_fields", "capabilities", "budget"} {
		if !reflect.DeepEqual(httpEnvelope[field], mcpEnvelope[field]) {
			t.Fatalf("field %s differs: HTTP=%#v MCP=%#v", field, httpEnvelope[field], mcpEnvelope[field])
		}
	}
}

func TestHTTPPathUnescapesOpaqueIDsWithoutSplittingEncodedSlashes(t *testing.T) {
	parts, ok := splitLivePath("/v1/live/session%3Aencoded/modules/github.com%2Fexample%2Fservice")
	if !ok {
		t.Fatal("encoded live path was rejected")
	}
	want := []string{"session:encoded", "modules", "github.com/example/service"}
	if len(parts) != len(want) {
		t.Fatalf("path parts = %#v, want %#v", parts, want)
	}
	for index := range want {
		if parts[index] != want[index] {
			t.Fatalf("path part %d = %q, want %q", index, parts[index], want[index])
		}
	}
}

type httpResult struct {
	StatusCode int
	Body       string
}

func postHTTP(t *testing.T, endpoint, body string, headers map[string]string) httpResult {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	value, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return httpResult{StatusCode: response.StatusCode, Body: string(value)}
}

func readHTTPBody(t *testing.T, response *http.Response) string {
	t.Helper()
	defer func() { _ = response.Body.Close() }()
	value, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(value)
}
