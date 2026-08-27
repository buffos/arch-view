package main

import (
	"fmt"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/processanalyzer"
)

// loadExternalPlugins validates every descriptor before changing the host
// registry. This keeps an invalid or duplicate descriptor from leaving a
// partially registered plugin set behind.
func loadExternalPlugins(host *analysis.Host, paths []string) error {
	if len(paths) == 0 {
		return nil
	}

	registered := make(map[string]struct{})
	for _, manifest := range host.ListManifests() {
		registered[manifest.ID] = struct{}{}
	}
	loaded := make([]analysis.Analyzer, 0, len(paths))
	for _, path := range paths {
		analyzer, err := processanalyzer.LoadDescriptor(path)
		if err != nil {
			return err
		}
		manifest := analyzer.Manifest()
		if _, exists := registered[manifest.ID]; exists {
			return analysis.NewHostError(analysis.ErrDuplicateAnalyzer, "analyzer id is already registered", map[string]any{
				"id":   manifest.ID,
				"path": path,
			})
		}
		registered[manifest.ID] = struct{}{}
		loaded = append(loaded, analyzer)
	}

	for _, analyzer := range loaded {
		if err := host.Register(analyzer); err != nil {
			return fmt.Errorf("register external analyzer %q: %w", analyzer.Manifest().ID, err)
		}
	}
	return nil
}
