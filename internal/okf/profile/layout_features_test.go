package profile

import (
	"encoding/json"
	"github.com/buffo/arch-view/internal/okf/domain"
	"reflect"
	"testing"
)

func TestLayoutFeaturesInheritAndExplicitlyClear(t *testing.T) {
	var profiles []domain.Profile
	if err := json.Unmarshal([]byte(`[
	  {"profile_id":"project:base","layout":{"features":["ports"]}},
	  {"profile_id":"project:inherits","bases":["project:base"]},
	  {"profile_id":"project:clear","bases":["project:base"],"layout":{"features":[]}}
	]`), &profiles); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(profiles)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &profiles); err != nil {
		t.Fatal(err)
	}
	registry := NewRegistry()
	registry.SetProjectProfiles(profiles)
	inherited, _ := registry.ResolveProfile("project:inherits")
	cleared, _ := registry.ResolveProfile("project:clear")
	if !reflect.DeepEqual(inherited.Layout.Features, []string{"ports"}) || cleared.Layout.Features == nil || len(cleared.Layout.Features) != 0 {
		t.Fatalf("inherit %+v clear %+v", inherited.Layout, cleared.Layout)
	}
	cloned := domain.CloneProfile(profiles[2])
	if cloned.Layout.Features == nil {
		t.Fatal("clone lost clear")
	}
	if profiles[1].Layout.Features != nil {
		t.Fatal("omission became clear")
	}
}
