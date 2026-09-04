package viewer

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestOKFHTTPRejectsStaleEditorAndReplaysCreation(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	host := httptest.NewServer(server.Handler())
	defer host.Close()
	creation := map[string]any{
		"profile":        domain.Profile{Name: "Original"},
		"new_profile_id": "shared", "operation_id": "create-shared",
	}
	var created struct {
		Data domain.ProjectConfiguration `json:"data"`
	}
	response := postOKFHTTP(t, host.URL+"/v1/okf/profiles/save-as", http.MethodPost, creation)
	decodeOKFHTTP(t, response, &created)
	if response.status != http.StatusCreated || len(created.Data.Profiles) != 1 {
		t.Fatalf("create: %d %s", response.status, response.body)
	}
	readCatalog := func() domain.ProfileCatalog {
		t.Helper()
		var result struct {
			Data domain.ProfileCatalog `json:"data"`
		}
		response := getOKFHTTP(t, host.URL+"/v1/okf/profiles")
		decodeOKFHTTP(t, response, &result)
		if response.status != http.StatusOK {
			t.Fatalf("catalog: %d %s", response.status, response.body)
		}
		return result.Data
	}
	first, second := readCatalog(), readCatalog()
	if first.ConfigurationRevision == "" || !reflect.DeepEqual(first, second) {
		t.Fatal("editors did not start from the same catalog")
	}
	newer := created.Data.Profiles[0]
	newer.Name = "First editor saved"
	response = postOKFHTTP(t, host.URL+"/v1/okf/profiles/project:shared", http.MethodPut, map[string]any{
		"profile": newer, "expected_revision": first.ConfigurationRevision, "operation_id": "first-save",
	})
	if response.status != http.StatusOK {
		t.Fatalf("first save: %d %s", response.status, response.body)
	}
	readBytes := func() []byte {
		t.Helper()
		value, err := os.ReadFile(filepath.Join(root, ".archview.json"))
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	before, savedCatalog := readBytes(), readCatalog()
	if savedCatalog.ConfigurationRevision == first.ConfigurationRevision {
		t.Fatal("successful save did not advance the configuration revision")
	}
	foundSaved := false
	for _, profile := range savedCatalog.Profiles {
		if profile.ProfileID == newer.ProfileID {
			foundSaved = profile.Name == newer.Name
		}
	}
	if !foundSaved {
		t.Fatal("successful save did not publish the first editor's profile")
	}
	stale := created.Data.Profiles[0]
	stale.Name = "Stale editor overwrite"
	response = postOKFHTTP(t, host.URL+"/v1/okf/profiles/project:shared", http.MethodPut, map[string]any{
		"profile": stale, "expected_revision": second.ConfigurationRevision, "operation_id": "second-save",
	})
	assertConflict := func(response okfHTTPResponse, code string) {
		t.Helper()
		var failure struct {
			Error struct{ Code string } `json:"error"`
		}
		decodeOKFHTTP(t, response, &failure)
		if response.status != http.StatusConflict || failure.Error.Code != code {
			t.Fatalf("want %s: %d %s", code, response.status, response.body)
		}
	}
	assertConflict(response, "okf_revision_conflict")
	var replay struct {
		Data domain.ProjectConfiguration `json:"data"`
	}
	response = postOKFHTTP(t, host.URL+"/v1/okf/profiles/save-as", http.MethodPost, creation)
	decodeOKFHTTP(t, response, &replay)
	if response.status != http.StatusCreated || !reflect.DeepEqual(replay.Data, created.Data) {
		t.Fatalf("creation replay must return original result: %d %s", response.status, response.body)
	}
	creation["new_profile_id"] = "different"
	response = postOKFHTTP(t, host.URL+"/v1/okf/profiles/save-as", http.MethodPost, creation)
	assertConflict(response, "okf_idempotency_conflict")
	if !bytes.Equal(before, readBytes()) || !reflect.DeepEqual(savedCatalog, readCatalog()) {
		t.Fatal("stale save or creation retry changed the saved configuration")
	}
}
