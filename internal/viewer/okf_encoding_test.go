package viewer

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestOKFProfilePathDecodesEachSegmentExactlyOnce(t *testing.T) {
	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	for _, id := range []string{"literal%20value", "folder/profile", "plus+space name"} {
		t.Run(id, func(t *testing.T) {
			created := postOKFHTTP(t, httpServer.URL+"/v1/okf/profiles/save-as", http.MethodPost, map[string]any{"profile": map[string]any{"name": "Encoded"}, "new_profile_id": id})
			if created.status != http.StatusCreated {
				t.Fatalf("create: %d %s", created.status, created.body)
			}
			endpoint := httpServer.URL + "/v1/okf/profiles/" + url.PathEscape("project:"+id)
			saved := postOKFHTTP(t, endpoint, http.MethodPut, map[string]any{"name": "Updated"})
			if saved.status != http.StatusOK {
				t.Fatalf("save: %d %s", saved.status, saved.body)
			}
			deleted := postOKFHTTP(t, endpoint, http.MethodDelete, map[string]any{})
			if deleted.status != http.StatusOK {
				t.Fatalf("delete: %d %s", deleted.status, deleted.body)
			}
		})
	}
}
