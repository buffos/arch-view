package hierarchy

import (
	"path"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/okf/domain"
)

type claim struct {
	parent     string
	child      string
	proof      domain.Provenance
	additional []domain.Provenance
}

// Normalize separates containment from semantic links. The returned list is
// deterministic and contains only unambiguous navigable containment edges.
func Normalize(index domain.BundleIndex, settings domain.HierarchySettings) ([]domain.Relationship, []domain.Diagnostic) {
	claims := make(map[string][]claim)
	diagnostics := make([]domain.Diagnostic, 0)
	explicitClaims := make(map[string]bool)
	for _, conceptID := range index.ConceptOrder {
		document := index.Documents[conceptID]
		if settings.UseExplicit {
			for _, reference := range document.ExplicitParents {
				explicitClaims[conceptID] = true
				addClaim(claims, &diagnostics, conceptID, reference, document.SourcePath, "explicit_parent", index)
			}
			for _, reference := range document.ExplicitChildren {
				childID := normalizeConceptReference(reference)
				if childID == "" {
					diagnostics = append(diagnostics, hierarchyDiagnostic(index.BundleID, conceptID, "okf_relationship_unresolved", "An explicit child reference is not a safe concept path."))
					continue
				}
				if _, exists := index.Documents[childID]; !exists {
					diagnostics = append(diagnostics, hierarchyDiagnostic(index.BundleID, conceptID, "okf_relationship_unresolved", "An explicit child reference does not identify a concept in this bundle."))
					continue
				}
				explicitClaims[childID] = true
				claims[childID] = append(claims[childID], claim{parent: conceptID, child: childID, proof: domain.Provenance{Source: "explicit_children", Path: document.SourcePath, Explanation: reference}})
			}
		}
	}

	parents := make(map[string]claim)
	for _, conceptID := range index.ConceptOrder {
		candidates := deduplicateClaims(claims[conceptID])
		if len(candidates) == 0 && settings.UseFilesystem && !explicitClaims[conceptID] {
			if parentID := filesystemParent(index, conceptID); parentID != "" {
				candidates = []claim{{parent: parentID, child: conceptID, proof: domain.Provenance{Source: "filesystem_fallback", Path: index.Documents[conceptID].SourcePath, Explanation: "No selected explicit parent claim was present."}}}
			}
		}
		if len(candidates) > 1 {
			diagnostics = append(diagnostics, hierarchyDiagnostic(index.BundleID, conceptID, "okf_hierarchy_conflict", "Multiple containment parents were declared; the ambiguous claims were excluded."))
			continue
		}
		if len(candidates) == 1 {
			candidate := candidates[0]
			if candidate.parent == candidate.child {
				diagnostics = append(diagnostics, hierarchyDiagnostic(index.BundleID, conceptID, "okf_hierarchy_conflict", "A concept cannot contain itself; the claim was excluded."))
				continue
			}
			parents[conceptID] = candidate
		}
	}

	removeCycles(index.BundleID, parents, &diagnostics)
	relationships := make([]domain.Relationship, 0, len(parents))
	children := make([]string, 0, len(parents))
	for childID := range parents {
		children = append(children, childID)
	}
	sort.Strings(children)
	for _, childID := range children {
		candidate := parents[childID]
		relationships = append(relationships, domain.Relationship{
			RelationshipID: "containment:" + candidate.parent + ":" + candidate.child,
			Kind:           domain.RelationshipContainment,
			From:           candidate.parent,
			To:             candidate.child,
			Provenance:     append([]domain.Provenance{candidate.proof}, candidate.additional...),
		})
	}
	return relationships, diagnostics
}

func addClaim(claims map[string][]claim, diagnostics *[]domain.Diagnostic, childID, reference, sourcePath, source string, index domain.BundleIndex) {
	parentID := normalizeConceptReference(reference)
	if parentID == "" {
		*diagnostics = append(*diagnostics, hierarchyDiagnostic(index.BundleID, childID, "okf_relationship_unresolved", "An explicit parent reference is not a safe concept path."))
		return
	}
	if parentID == childID {
		claims[childID] = append(claims[childID], claim{parent: parentID, child: childID, proof: domain.Provenance{Source: source, Path: sourcePath, Explanation: reference}})
		return
	}
	if _, exists := index.Documents[parentID]; !exists {
		*diagnostics = append(*diagnostics, hierarchyDiagnostic(index.BundleID, childID, "okf_relationship_unresolved", "An explicit parent reference does not identify a concept in this bundle."))
		return
	}
	claims[childID] = append(claims[childID], claim{parent: parentID, child: childID, proof: domain.Provenance{Source: source, Path: sourcePath, Explanation: reference}})
}

func normalizeConceptReference(value string) string {
	value = strings.TrimSpace(strings.TrimPrefix(value, "/"))
	if value == "" || strings.Contains(value, "\\") {
		return ""
	}
	value = path.Clean(value)
	if value == "." || value == ".." || strings.HasPrefix(value, "../") {
		return ""
	}
	if !strings.HasSuffix(strings.ToLower(value), ".md") {
		value += ".md"
	}
	if len(value) >= len(".md") && strings.EqualFold(value[len(value)-len(".md"):], ".md") {
		return value[:len(value)-len(".md")]
	}
	return value
}

func deduplicateClaims(values []claim) []claim {
	seen := make(map[string]int)
	proofs := make(map[string]map[domain.Provenance]bool)
	result := make([]claim, 0, len(values))
	for _, value := range values {
		key := value.parent + "\x00" + value.child
		if value.parent == "" {
			continue
		}
		if position, exists := seen[key]; exists {
			if !proofs[key][value.proof] {
				result[position].additional = append(result[position].additional, value.proof)
				proofs[key][value.proof] = true
			}
			continue
		}
		seen[key] = len(result)
		proofs[key] = map[domain.Provenance]bool{value.proof: true}
		result = append(result, value)
	}
	sort.SliceStable(result, func(left, right int) bool { return result[left].parent < result[right].parent })
	return result
}

func filesystemParent(index domain.BundleIndex, conceptID string) string {
	document := index.Documents[conceptID]
	directory := path.Dir(document.SourcePath)
	for directory != "." && directory != "" && directory != "/" {
		parentPath := path.Join(path.Dir(directory), path.Base(directory)+".md")
		parentID := strings.TrimSuffix(parentPath, ".md")
		if _, exists := index.Documents[parentID]; exists {
			return parentID
		}
		next := path.Dir(directory)
		if next == directory {
			break
		}
		directory = next
	}
	return ""
}

func removeCycles(bundleID string, parents map[string]claim, diagnostics *[]domain.Diagnostic) {
	done := make(map[string]bool)
	keys := make([]string, 0, len(parents))
	for child := range parents {
		keys = append(keys, child)
	}
	sort.Strings(keys)
	for _, child := range keys {
		trail := []string{}
		positions := make(map[string]int)
		node := child
		for !done[node] {
			if start, exists := positions[node]; exists {
				for _, member := range trail[start:] {
					delete(parents, member)
				}
				*diagnostics = append(*diagnostics, hierarchyDiagnostic(bundleID, node, "okf_hierarchy_conflict", "A containment cycle was excluded from navigable hierarchy."))
				break
			}
			positions[node] = len(trail)
			trail = append(trail, node)
			parent, exists := parents[node]
			if !exists {
				break
			}
			node = parent.parent
		}
		for _, member := range trail {
			done[member] = true
		}
	}
}

func hierarchyDiagnostic(bundleID, conceptID, code, message string) domain.Diagnostic {
	return domain.Diagnostic{Code: code, Severity: "warning", Category: "hierarchy", Message: message, BundleID: bundleID, ConceptID: conceptID, Recovery: "Review the source claim or choose a different hierarchy mapping."}
}
