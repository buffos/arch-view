package main

import (
	"encoding/json"
	"testing"

	"github.com/buffo/arch-view/internal/analysis/orchestration"
)

func decodeAggregateRun(t *testing.T, data []byte) orchestration.AnalysisRun {
	t.Helper()
	var run orchestration.AnalysisRun
	if err := json.Unmarshal(data, &run); err != nil {
		t.Fatalf("decode aggregate run: %v", err)
	}
	if run.SchemaVersion != orchestration.AggregateSchemaVersion {
		t.Fatalf("schema = %q, want %q", run.SchemaVersion, orchestration.AggregateSchemaVersion)
	}
	return run
}

func hasAggregateAnalyzer(run orchestration.AnalysisRun, analyzerID string) bool {
	for _, scope := range run.Scopes {
		if scope.Analyzer.ID == analyzerID {
			return true
		}
	}
	return false
}

func hasAggregateLanguage(run orchestration.AnalysisRun, language string) bool {
	for _, scope := range run.Scopes {
		if scope.Analyzer.Language == language {
			return true
		}
	}
	return false
}
