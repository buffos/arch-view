// Command docs-demo creates the small public fixture used by the
// documentation site's self-contained demo.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/buffo/arch-view/internal/analysis"
	exporter "github.com/buffo/arch-view/internal/export"
	"github.com/buffo/arch-view/internal/model/canonical"
)

func main() {
	output := flag.String("output", "", "HTML output file")
	flag.Parse()
	if *output == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: docs-demo -output <file>")
		os.Exit(2)
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	value, err := canonical.Normalize(analysis.AnalysisResult{
		Status: analysis.StatusComplete,
		Analyzer: analysis.AnalyzerInfo{
			ID:         "org.archview.demo",
			Version:    "1.0.0",
			Language:   "go",
			APIVersion: analysis.AnalyzerAPIVersion,
		},
		Project: analysis.ProjectInfo{
			RootLabel:  "payments-demo",
			Boundary:   "go.mod",
			ModulePath: "example.com/payments",
		},
		Modules: []analysis.ModuleObservation{
			{ID: "module:web", Language: "go", Kind: "package", Name: "web", DisplayName: "web", Hierarchy: []string{"web"}, SourceReferenceIDs: []string{}, Tags: []string{}},
			{ID: "module:orders", Language: "go", Kind: "package", Name: "orders", DisplayName: "orders", Hierarchy: []string{"orders"}, SourceReferenceIDs: []string{}, Tags: []string{}},
			{ID: "module:payments", Language: "go", Kind: "package", Name: "payments", DisplayName: "payments", Hierarchy: []string{"payments"}, SourceReferenceIDs: []string{}, Tags: []string{}},
			{ID: "module:shared", Language: "go", Kind: "package", Name: "shared", DisplayName: "shared", Hierarchy: []string{"shared"}, SourceReferenceIDs: []string{}, Tags: []string{}},
		},
		References:       []analysis.Reference{},
		SourceReferences: []analysis.SourceReference{},
		Relationships: []analysis.RelationshipObservation{
			{ID: "relationship:web-orders", Type: "depends_on", FromModuleID: "module:web", ToModuleID: "module:orders", SourceReferenceIDs: []string{}},
			{ID: "relationship:orders-payments", Type: "depends_on", FromModuleID: "module:orders", ToModuleID: "module:payments", SourceReferenceIDs: []string{}},
			{ID: "relationship:web-shared", Type: "depends_on", FromModuleID: "module:web", ToModuleID: "module:shared", SourceReferenceIDs: []string{}},
			{ID: "relationship:orders-shared", Type: "depends_on", FromModuleID: "module:orders", ToModuleID: "module:shared", SourceReferenceIDs: []string{}},
		},
		Diagnostics: []analysis.Diagnostic{},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := exporter.Write(value, exporter.Request{
		Format:              exporter.FormatHTML,
		OutputPath:          *output,
		ReferenceVisibility: "hidden",
		Overwrite:           true,
		Context:             context.Background(),
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
