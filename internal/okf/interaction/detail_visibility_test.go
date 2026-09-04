package interaction

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

func TestEffectiveDetailDefaultsAndExplicitStateHiding(t *testing.T) {
	registry := profile.NewRegistry()
	effective, _ := registry.ResolveProfile(profile.DefaultProfileID)
	if !effective.State.ShowDeclared || !effective.Details.ShowRawMarkdown || !effective.Details.ShowUnknown {
		t.Fatal("effective profile must expose the detail defaults used by the backend")
	}
	index := domain.BundleIndex{ConceptOrder: []string{"one"}, Documents: map[string]domain.ConceptDocument{
		"one": {ConceptID: "one", SourcePath: "one.md", Markdown: "Body", Frontmatter: map[string]any{"state": "specified"}, UnknownFrontmatter: map[string]any{"custom": "value"}},
	}}
	effective.State.Field = "state"
	effective.State.Mapping = map[string]string{"specified": "specified"}
	for _, visible := range []bool{true, false} {
		encoded, _ := json.Marshal(map[string]any{"show_declared": visible, "field": "state", "mapping": map[string]string{"specified": "specified"}})
		if err := json.Unmarshal(encoded, &effective.State); err != nil {
			t.Fatal(err)
		}
		detail, err := Detail(context.Background(), index, effective, "one", registry)
		if err != nil {
			t.Fatal(err)
		}
		if (detail.DeclaredState != "") != visible || detail.EffectiveState != "specified" {
			t.Fatalf("visible=%v: declared=%q effective=%q", visible, detail.DeclaredState, detail.EffectiveState)
		}
		if detail.RawMarkdown != "Body" || detail.MappedMetadata["custom"] != "value" {
			t.Fatal("declared-state setting changed unrelated source detail")
		}
	}
}
