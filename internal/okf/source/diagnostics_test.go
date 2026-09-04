package source

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestDiagnosticLimitPreservesErrorsAndReportsOmission(t *testing.T) {
	for _, errorPosition := range []int{0, maxDiagnostics - 1, maxDiagnostics, maxDiagnostics + 1} {
		var values []domain.Diagnostic
		var all []domain.Diagnostic
		for i := 0; i < maxDiagnostics+2; i++ {
			value := domain.Diagnostic{Code: "fixture", Severity: "warning"}
			if i == errorPosition {
				value.Severity = "error"
			}
			all = append(all, value)
			values = appendDiagnostic(values, value)
		}
		for _, bounded := range [][]domain.Diagnostic{values, boundDiagnostics(all)} {
			if len(bounded) != maxDiagnostics || bounded[len(bounded)-1].Code != "okf_diagnostics_truncated" {
				t.Fatalf("missing bound/notice: %d", len(bounded))
			}
			hasError := false
			for _, value := range bounded {
				hasError = hasError || value.Severity == "error"
			}
			if !hasError {
				t.Fatalf("error at %d was lost", errorPosition)
			}
		}
	}
}

func TestSourceLinkDiagnosticOverflowIsVisible(t *testing.T) {
	root := t.TempDir()
	var content strings.Builder
	content.WriteString("---\ntype: concept\n---\n")
	for i := 0; i < maxDiagnostics+1; i++ {
		fmt.Fprintf(&content, "[missing](missing-%d.md)\n", i)
	}
	writeFile(t, filepath.Join(root, ".okf", "root.md"), content.String())
	candidates, _ := NewFilesystemScanner().Scan(context.Background(), root)
	if len(candidates) != 1 || !candidates[0].Selectable {
		t.Fatalf("warning-only bundle rejected: %#v", candidates)
	}
	diagnostics := candidates[0].Diagnostics
	if len(diagnostics) != maxDiagnostics || diagnostics[len(diagnostics)-1].Code != "okf_diagnostics_truncated" {
		t.Fatalf("omitted warnings not reported: %#v", diagnostics)
	}
}
