package viewer

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestUnknownProfileShapeCannotWriteConfiguration(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	host := httptest.NewServer(server.Handler())
	defer host.Close()
	invalid := map[string]any{"name": "Invalid shape", "style": map[string]any{"tokens": map[string]any{"custom": map[string]any{"shape": "missing.shape@1"}}}}
	response := postOKFHTTP(t, host.URL+"/v1/okf/profiles/save-as", http.MethodPost, map[string]any{"new_profile_id": "shape", "profile": invalid})
	if response.status != http.StatusBadRequest || !bytes.Contains(response.body, []byte("okf_shape_unsupported")) {
		t.Fatalf("invalid Save As=%d %s", response.status, response.body)
	}
	configPath := filepath.Join(root, ".archview.json")
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("invalid Save As touched configuration: %v", err)
	}
	response = postOKFHTTP(t, host.URL+"/v1/okf/profiles/save-as", http.MethodPost, map[string]any{"new_profile_id": "shape", "profile": map[string]any{"name": "Valid"}})
	if response.status != http.StatusCreated {
		t.Fatalf("valid Save As=%d %s", response.status, response.body)
	}
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	response = postOKFHTTP(t, host.URL+"/v1/okf/profiles/project:shape", http.MethodPut, invalid)
	if response.status != http.StatusBadRequest {
		t.Fatalf("invalid Save=%d %s", response.status, response.body)
	}
	after, err := os.ReadFile(configPath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("invalid Save changed configuration: %v", err)
	}
}
