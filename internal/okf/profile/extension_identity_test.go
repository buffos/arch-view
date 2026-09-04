package profile

import (
	"github.com/buffo/arch-view/internal/okf/domain"
	"testing"
)

func TestExtensionIdentityRejectsAmbiguousNames(t *testing.T) {
	for _, value := range [][2]string{{"", "1"}, {"plain", "1"}, {".provider", "1"}, {"provider.", "1"}, {"a..b", "1"}, {"a.b", ""}, {"a.b@1", "2"}, {"a.b", "1@2"}, {"a. b", "1"}, {"a.b", "1\n"}} {
		if domain.ValidExtensionIdentity(value[0], value[1]) {
			t.Fatalf("invalid identity accepted: %q", value)
		}
	}
	if !domain.ValidExtensionIdentity("vendor.diagnostic.check", "1.2") {
		t.Fatal("valid versioned identity rejected")
	}
}
