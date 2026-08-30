package observations

import (
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analyzers/go/scanner"
)

func TestSourceIndexScopePreservesOrchestrationFingerprints(t *testing.T) {
	sources := strings.Repeat("a", 64)
	policy := strings.Repeat("b", 64)
	scope := sourceIndexScope(&analysis.SourceScope{
		ProjectRoot:                 "service",
		MatchedSourceSetFingerprint: "sha256:" + sources,
		PolicyFingerprint:           "sha256:" + policy,
	}, scanner.Project{RelativeModuleRoot: "service"})
	if scope.SourceScopeFingerprint.Value != sources || scope.SourceScopeFingerprint.Algorithm != analysis.SourceHashAlgorithm {
		t.Fatalf("source scope fingerprint = %#v, want the matched source digest", scope.SourceScopeFingerprint)
	}
	if scope.SourcePolicyFingerprint == nil || scope.SourcePolicyFingerprint.Value != policy || scope.SourcePolicyFingerprint.Algorithm != analysis.SourceHashAlgorithm {
		t.Fatalf("source policy fingerprint = %#v, want the policy digest", scope.SourcePolicyFingerprint)
	}
}
