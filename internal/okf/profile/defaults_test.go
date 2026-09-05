package profile

import (
	"encoding/json"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestDetailDefaultsPreserveExplicitFalse(t *testing.T) {
	var declaration domain.Profile
	if err := json.Unmarshal([]byte(`{"profile_id":"project:hidden","state":{"show_declared":false},"details":{"show_raw_markdown":false,"show_unknown_frontmatter":false}}`), &declaration); err != nil {
		t.Fatal(err)
	}
	effective := normalize(declaration)
	if effective.State.ShowDeclared || effective.Details.ShowRawMarkdown || effective.Details.ShowUnknown {
		t.Fatal("normalization overwrote explicit hidden detail settings")
	}
	defaults := normalize(domain.Profile{ProfileID: "project:defaults"})
	if !defaults.State.ShowDeclared || !defaults.Details.ShowRawMarkdown || !defaults.Details.ShowUnknown {
		t.Fatal("omitted settings must reflect the enabled backend defaults")
	}
}

func TestOKFLayoutDefaultsUseLayeredWithJunctionsAndPorts(t *testing.T) {
	for _, profileID := range []string{DefaultProfileID, FogProfileID} {
		effective, diagnostics := NewRegistry().ResolveProfile(profileID)
		if len(diagnostics) != 0 {
			t.Fatalf("%s diagnostics = %#v", profileID, diagnostics)
		}
		if effective.Layout.Algorithm != "layered" {
			t.Fatalf("%s layout algorithm = %q, want layered", profileID, effective.Layout.Algorithm)
		}
		if !sameStrings(effective.Layout.Features, []string{"junctions", "ports"}) {
			t.Fatalf("%s layout features = %#v, want junctions and ports", profileID, effective.Layout.Features)
		}
	}
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
