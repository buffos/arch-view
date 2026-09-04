package viewer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/buffo/arch-view/internal/okf/application"
	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestProfileValidationHTTPIncludesEffectiveDraft(t *testing.T) {
	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.okf.SaveProfileAs(context.Background(), domain.Profile{Navigation: domain.NavigationSettings{DefaultDepth: 7}}, "", "base", "", "base", nil)
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	response := postOKFHTTP(t, httpServer.URL+"/v1/okf/profiles/validate", http.MethodPost, map[string]any{"profile_id": "project:draft", "bases": []string{"project:base"}})
	var envelope struct {
		Data application.ProfilePreview `json:"data"`
	}
	decodeOKFHTTP(t, response, &envelope)
	if response.status != http.StatusOK || !envelope.Data.Valid || envelope.Data.Profile.Navigation.DefaultDepth != 0 || envelope.Data.EffectiveProfile.Navigation.DefaultDepth != 7 {
		t.Fatalf("declaration and effective draft response: %#v", envelope)
	}
}
