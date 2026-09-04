package profile

import (
	"testing"

	"github.com/buffo/arch-view/internal/okf/ports"
)

type registrationRule struct {
	ports.RuleStrategy
	id      func() string
	version func() string
}

func (rule registrationRule) ID() string      { return rule.id() }
func (rule registrationRule) Version() string { return rule.version() }

func TestRuleRegistrationRejectsInvalidMetadataWithoutPublishing(t *testing.T) {
	for _, identity := range [][2]string{{"test.rule@2", "1"}, {"test.rule", "2@1"}, {"test..rule", "1"}, {"test.rule", " 1"}} {
		registry := NewRegistry()
		rule := registrationRule{id: func() string { return identity[0] }, version: func() string { return identity[1] }}
		if err := registry.Register(rule); err == nil {
			t.Fatalf("invalid metadata accepted: %v", identity)
		}
		if _, ok := registry.Resolve(identity[0], identity[1]); ok {
			t.Fatal("rejected rule was published")
		}
	}
}

func TestRuleRegistrationContainsMetadataPanicsAndReadsIdentityOnce(t *testing.T) {
	registry := NewRegistry()
	for _, callback := range []string{"id", "version"} {
		rule := registrationRule{id: func() string {
			if callback == "id" {
				panic("id failure")
			}
			return "test.panic_registration"
		}, version: func() string { panic("version failure") }}
		if err := registry.Register(rule); err == nil {
			t.Fatal("metadata failure should be a registration error")
		}
	}
	ids, versions := 0, 0
	rule := registrationRule{id: func() string { ids++; return "test.register_once" }, version: func() string { versions++; return "1" }}
	if err := registry.Register(rule); err != nil {
		t.Fatal(err)
	}
	if ids != 1 || versions != 1 {
		t.Fatalf("callbacks executed %d/%d times", ids, versions)
	}
	if _, ok := registry.Resolve("test.register_once", "1"); !ok {
		t.Fatal("valid rule missing")
	}
}
