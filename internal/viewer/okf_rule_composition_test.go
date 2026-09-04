package viewer

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestOKFHTTPValidatesAppliesAndRepairsRuleComposition(t *testing.T) {
	root := t.TempDir()
	writeOKFHTTPFixture(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: topic\ntitle: Source title\ncustom: retained\n---\n")
	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	host := httptest.NewServer(server.Handler())
	defer host.Close()
	rule := func(priority int, outputs map[string]any) domain.RuleInvocation {
		outputs["field"], outputs["value"] = "custom", "retained"
		return domain.RuleInvocation{RuleID: "okf.rule.metadata_equals", Version: "1", Enabled: true, Priority: priority, Parameters: outputs}
	}
	value := domain.Profile{ProfileID: "project:composition", Name: "Composition", Bases: []string{"builtin:neutral"}, Rules: []domain.RuleInvocation{
		rule(10, map[string]any{"label": "High priority", "role": "left", "annotations": map[string]any{"a": 1, "shared": "same"}}),
		rule(10, map[string]any{"role": "right", "annotations": map[string]any{"b": 2, "shared": "same"}}),
		rule(1, map[string]any{"label": "Low priority"}),
	}}
	validation := postOKFHTTP(t, host.URL+"/v1/okf/profiles/validate", http.MethodPost, value)
	var validated struct {
		Data struct {
			Valid       bool                `json:"valid"`
			Diagnostics []domain.Diagnostic `json:"diagnostics"`
		} `json:"data"`
	}
	decodeOKFHTTP(t, validation, &validated)
	if validation.status != http.StatusOK || !validated.Data.Valid || len(validated.Data.Diagnostics) != 0 {
		t.Fatalf("valid composition rejected: %d %+v", validation.status, validated)
	}
	created := postOKFHTTP(t, host.URL+"/v1/okf/profiles/save-as", http.MethodPost, map[string]any{"profile": value, "new_profile_id": "composition"})
	if created.status != http.StatusCreated {
		t.Fatalf("save as: %d %s", created.status, created.body)
	}
	for _, selection := range []struct {
		path string
		body map[string]any
	}{
		{"bundle", map[string]any{"bundle_id": ".okf"}},
		{"profile", map[string]any{"profile_id": value.ProfileID}},
	} {
		response := postOKFHTTP(t, host.URL+"/v1/okf/sessions/default/"+selection.path, http.MethodPut, selection.body)
		if response.status != http.StatusOK {
			t.Fatalf("select %s: %d", selection.path, response.status)
		}
	}
	for _, repaired := range []bool{false, true} {
		if repaired {
			value.Rules[1].Parameters["role"] = "left"
			response := postOKFHTTP(t, host.URL+"/v1/okf/profiles/project:composition", http.MethodPut, value)
			if response.status != http.StatusOK {
				t.Fatalf("repair: %d %s", response.status, response.body)
			}
		}
		response := getOKFHTTP(t, host.URL+"/v1/okf/sessions/default/projection")
		var envelope struct {
			Data domain.ProjectionSnapshot `json:"data"`
		}
		decodeOKFHTTP(t, response, &envelope)
		if response.status != http.StatusOK || len(envelope.Data.Nodes) != 1 {
			t.Fatalf("projection: %d %+v", response.status, envelope.Data)
		}
		node := envelope.Data.Nodes[0]
		wantRole := ""
		if repaired {
			wantRole = "left"
		}
		if node.Label != "High priority" || node.Role != wantRole || node.Annotations["a"] != float64(1) || node.Annotations["b"] != float64(2) || node.Annotations["shared"] != "same" {
			t.Fatalf("composition changed compatible outputs: %+v", node)
		}
		conflicts := 0
		for _, diagnostic := range envelope.Data.Diagnostics {
			if diagnostic.Code == "okf_rule_conflict" {
				conflicts++
				if diagnostic.ConceptID != "root" || diagnostic.Details["field"] != "role" {
					t.Fatalf("unscoped conflict: %+v", diagnostic)
				}
			}
		}
		if (!repaired && conflicts != 1) || (repaired && conflicts != 0) {
			t.Fatalf("repaired=%v conflicts=%d", repaired, conflicts)
		}
	}
}
