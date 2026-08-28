package tsanalyzer

import (
	"context"

	"github.com/buffo/arch-view/internal/analysis"
)

type tsImportExtractor interface {
	Extract(context.Context, string, string, string) tsImportExtraction
}

type tsImportExtraction struct {
	Observations []tsImportObservation
	Diagnostics  []analysis.Diagnostic
	BackendError error
}

func tsImportBackendDiagnostic(repositoryPath string, err error) analysis.Diagnostic {
	message := "TypeScript syntax backend failed."
	if err != nil {
		message += " " + err.Error()
	}
	return analysis.Diagnostic{
		Code:        "typescript_syntax_backend",
		Severity:    "warning",
		Message:     message,
		Path:        repositoryPath,
		Location:    &analysis.Position{Line: 1, Column: 1},
		Recoverable: true,
	}
}
