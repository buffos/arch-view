package application

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestInvalidRuleSavePreservesConfiguration(t *testing.T) {
	root := t.TempDir()
	service := New(root)
	ctx := context.Background()
	saved, err := service.SaveProfileAs(ctx, domain.Profile{ProfileID: "draft"}, "", "valid", "", "valid", nil)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".archview.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	invalid := domain.Profile{ProfileID: "draft", Rules: []domain.RuleInvocation{{RuleID: "okf.rule.visibility", Enabled: true, Parameters: map[string]any{"field": "hidden", "value": true, "visible": "false"}}}}
	_, err = service.SaveProfileAs(ctx, invalid, "", "invalid", saved.Revision, "invalid", nil)
	if !hasApplicationCode(err, "okf_profile_invalid") {
		t.Fatalf("invalid rule saved: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatalf("invalid rule changed configuration: %v", err)
	}
}
