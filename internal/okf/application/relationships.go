package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

// applyRelationshipAdapters adds optional, renderer-neutral semantic edges to
// an indexed bundle. Adapters receive a defensive copy and cannot participate
// in containment traversal or mutate the source index owned by the service.
func applyRelationshipAdapters(ctx context.Context, value domain.BundleIndex, adapters []ports.RelationshipAdapter) (domain.BundleIndex, []domain.Diagnostic, error) {
	if len(adapters) == 0 {
		return value, nil, nil
	}
	result := domain.CloneIndex(value)
	diagnostics := make([]domain.Diagnostic, 0)
	seen := make(map[string]struct{}, len(result.Relationships))
	for _, relationship := range result.Relationships {
		if relationship.RelationshipID != "" {
			seen[relationship.RelationshipID] = struct{}{}
		}
	}
	for _, adapter := range adapters {
		if err := contextErr(ctx); err != nil {
			return domain.BundleIndex{}, domain.CloneDiagnostics(diagnostics), relationshipAdapterCancellation(err)
		}
		if adapter == nil {
			continue
		}
		adapterID, adapterVersion, relationships, adapterDiagnostics := evaluateRelationshipAdapter(ctx, adapter, value)
		if err := contextErr(ctx); err != nil {
			return domain.BundleIndex{}, boundDiagnostics(diagnostics), relationshipAdapterCancellation(err)
		}
		adapterDiagnostics = domain.CloneDiagnostics(adapterDiagnostics)
		for index := range adapterDiagnostics {
			diagnostic := &adapterDiagnostics[index]
			if diagnostic.BundleID == "" {
				diagnostic.BundleID = value.BundleID
			}
			if diagnostic.Category == "" {
				diagnostic.Category = "relationship_adapter"
			}
			if diagnostic.Code == "" {
				diagnostic.Code = "okf_relationship_adapter_diagnostic"
			}
		}
		diagnostics = append(diagnostics, adapterDiagnostics...)
		for relationshipIndex, relationship := range relationships {
			if err := contextErr(ctx); err != nil {
				return domain.BundleIndex{}, boundDiagnostics(diagnostics), relationshipAdapterCancellation(err)
			}
			if relationship.Kind != domain.RelationshipSemantic {
				diagnostics = append(diagnostics, adapterDiagnostic(value.BundleID, adapterID, "okf_relationship_adapter_kind_invalid", "relationship adapters may contribute semantic links only", map[string]any{"kind": relationship.Kind}))
				continue
			}
			if _, exists := result.Documents[relationship.From]; !exists {
				diagnostics = append(diagnostics, adapterDiagnostic(value.BundleID, adapterID, "okf_relationship_adapter_endpoint_missing", "a relationship adapter referenced a missing source concept", map[string]any{"from": relationship.From, "to": relationship.To}))
				continue
			}
			if _, exists := result.Documents[relationship.To]; !exists {
				diagnostics = append(diagnostics, adapterDiagnostic(value.BundleID, adapterID, "okf_relationship_adapter_endpoint_missing", "a relationship adapter referenced a missing target concept", map[string]any{"from": relationship.From, "to": relationship.To}))
				continue
			}
			if strings.TrimSpace(relationship.RelationshipID) == "" {
				relationship.RelationshipID = fmt.Sprintf("adapter:%s:%s:%s:%d", adapterID, relationship.From, relationship.To, relationshipIndex)
			}
			if _, exists := seen[relationship.RelationshipID]; exists {
				diagnostics = append(diagnostics, adapterDiagnostic(value.BundleID, adapterID, "okf_relationship_adapter_duplicate", "a relationship adapter produced a duplicate relationship ID", map[string]any{"relationship_id": relationship.RelationshipID}))
				continue
			}
			seen[relationship.RelationshipID] = struct{}{}
			relationship.Provenance = append(append([]domain.Provenance(nil), relationship.Provenance...), domain.Provenance{Source: "relationship_adapter", Path: adapterID, Explanation: adapterVersion})
			result.Relationships = append(result.Relationships, relationship)
		}
	}
	return result, boundDiagnostics(diagnostics), nil
}

func adapterDiagnostic(bundleID, adapterID, code, message string, details map[string]any) domain.Diagnostic {
	details = domain.CloneMap(details)
	if details == nil {
		details = map[string]any{}
	}
	details["adapter_id"] = adapterID
	return domain.Diagnostic{Code: code, Severity: "warning", Category: "relationship_adapter", BundleID: bundleID, Message: message, Details: details, Recovery: "Repair or disable the relationship adapter and refresh the catalog."}
}

func relationshipAdapterCancellation(err error) error {
	return domain.ContextOperationError(err, "relationship adaptation")
}

func evaluateRelationshipAdapter(ctx context.Context, adapter ports.RelationshipAdapter, index domain.BundleIndex) (id, version string, relationships []domain.Relationship, diagnostics []domain.Diagnostic) {
	id = "anonymous"
	defer func() {
		if recovered := recover(); recovered != nil {
			relationships = nil
			diagnostics = []domain.Diagnostic{adapterDiagnostic(index.BundleID, id, "okf_relationship_adapter_failed", fmt.Sprintf("relationship adapter panicked: %v", recovered), nil)}
		}
	}()
	if declaredID := strings.TrimSpace(adapter.ID()); declaredID != "" {
		id = declaredID
	}
	version = strings.TrimSpace(adapter.Version())
	relationships, diagnostics = adapter.Relationships(ctx, domain.CloneIndex(index))
	return
}
