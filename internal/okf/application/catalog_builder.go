package application

import (
	"context"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

// catalogBuilder prepares independent source indexes without accessing project
// configuration or session state. The caller owns atomic publication.
type catalogBuilder struct {
	scanner  ports.BundleScanner
	indexer  ports.BundleIndexer
	adapters []ports.RelationshipAdapter
}

type catalogBuild struct {
	candidates  []domain.BundleCandidate
	indexes     map[string]domain.BundleIndex
	diagnostics []domain.Diagnostic
}

func (builder catalogBuilder) build(ctx context.Context, root string) (catalogBuild, error) {
	candidates, diagnostics := builder.scanner.Scan(ctx, root)
	if err := contextErr(ctx); err != nil {
		return catalogBuild{}, contextOperationError(err, "catalog refresh")
	}
	candidates = cloneCandidates(candidates)
	diagnostics = domain.CloneDiagnostics(diagnostics)
	indexes := make(map[string]domain.BundleIndex)
	for indexValue := range candidates {
		if err := contextErr(ctx); err != nil {
			return catalogBuild{}, contextOperationError(err, "catalog refresh")
		}
		candidate := &candidates[indexValue]
		if !candidate.Selectable {
			continue
		}
		indexed, err := builder.indexer.Index(ctx, *candidate)
		if err != nil {
			candidate.Status = statusFromError(err)
			candidate.Selectable = false
			candidate.Diagnostics = append(candidate.Diagnostics, diagnosticsFromError(err)...)
			continue
		}
		indexed, adapterDiagnostics, adapterErr := applyRelationshipAdapters(ctx, indexed, builder.adapters)
		if adapterErr != nil {
			return catalogBuild{}, adapterErr
		}
		indexed.Diagnostics = boundDiagnostics(append(indexed.Diagnostics, adapterDiagnostics...))
		candidate.Diagnostics = boundDiagnostics(append(candidate.Diagnostics, adapterDiagnostics...))
		indexes[candidate.BundleID] = indexed
		candidate.ConceptCount = len(indexed.ConceptOrder)
		candidate.SourceRevision = indexed.SourceRevision
	}
	return catalogBuild{candidates: candidates, indexes: indexes, diagnostics: diagnostics}, nil
}
