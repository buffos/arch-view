package projection

import (
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

func TestRelationshipCountsIncludePolicyHiddenLinks(t *testing.T) {
	containment := []domain.Relationship{{RelationshipID: "parent", Kind: domain.RelationshipContainment, From: "a", To: "b"}}
	semantic := []domain.Relationship{{RelationshipID: "link", Kind: domain.RelationshipSemantic, From: "b", To: "a"}}
	effective := domain.Profile{Relationships: domain.RelationshipSettings{ShowContainment: true}}
	visible, hidden, truncated := projectedRelationships(containment, semantic, map[string]bool{"a": true, "b": true}, effective, 10)
	if len(visible) != 1 || hidden != 1 || truncated {
		t.Fatalf("policy counts: visible=%d hidden=%d truncated=%v", len(visible), hidden, truncated)
	}
	effective.Relationships.ShowSemantic = true
	visible, hidden, truncated = projectedRelationships(containment, semantic, map[string]bool{"a": true, "b": true}, effective, 1)
	if len(visible) != 1 || hidden != 1 || !truncated {
		t.Fatalf("budget counts: visible=%d hidden=%d truncated=%v", len(visible), hidden, truncated)
	}
}

func TestNodeFieldTruncationDefendsAgainstInvalidProfileLimits(t *testing.T) {
	source := strings.Repeat("α", profile.MaxNodeFieldText+50)
	for _, maximum := range []int{-1, 0, 1, profile.MaxNodeFieldText + 1} {
		output := truncateNodeField(source, maximum)
		if len([]rune(output)) > profile.MaxNodeFieldText {
			t.Fatalf("unbounded field for limit %d", maximum)
		}
	}
}
