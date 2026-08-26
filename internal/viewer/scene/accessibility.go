package scene

import (
	"fmt"
	"strings"
)

func nodeAccessibleLabel(node VisibleNode) string {
	parts := []string{fmt.Sprintf("%s %s", node.Kind, node.Label)}
	if node.ReferenceScope != "" {
		parts = append(parts, node.ReferenceScope+" reference scope")
	}
	if node.Counts.ModuleCount > 0 {
		parts = append(parts, fmt.Sprintf("%d module(s)", node.Counts.ModuleCount))
	}
	if node.Layer != nil {
		parts = append(parts, fmt.Sprintf("layer %d", *node.Layer))
	} else if len(node.Layers) > 0 {
		parts = append(parts, fmt.Sprintf("layers %s", joinInts(node.Layers)))
	}
	if node.CycleState != "none" {
		parts = append(parts, node.CycleState)
	}
	if node.DiagnosticState != "none" {
		parts = append(parts, node.DiagnosticState+" diagnostics")
	}
	parts = append(parts, node.IdentityState+" identity")
	if node.Counts.InternalRelationshipCount > 0 {
		parts = append(parts, fmt.Sprintf("%d internal relationship(s)", node.Counts.InternalRelationshipCount))
	}
	if node.ReferenceScope != "" {
		parts = append(parts, node.ConfidenceState+" confidence")
	}
	return strings.Join(parts, "; ")
}

func relationshipAccessibleLabel(relationship VisibleRelationship, fromLabel, toLabel string) string {
	parts := []string{fmt.Sprintf("%s from %s to %s", relationship.Type, fromLabel, toLabel), fmt.Sprintf("%d contributor(s)", relationship.Count), relationship.ConfidenceState + " confidence"}
	if relationship.TargetScope != "" {
		parts = append(parts, relationship.TargetScope+" reference scope")
	}
	if relationship.CycleState != "none" {
		parts = append(parts, relationship.CycleState)
	}
	return strings.Join(parts, "; ")
}

func joinInts(values []int) string {
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = fmt.Sprint(value)
	}
	return strings.Join(parts, ", ")
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func buildAccessibility(nodes []VisibleNode, relationships []VisibleRelationship, cycles []CycleIndicator, diagnostics []DiagnosticIndicator, references []ReferenceDetail) Accessibility {
	accessibility := Accessibility{
		ReadingOrder: []string{},
		Descriptions: map[string]string{},
	}
	for _, node := range nodes {
		accessibility.ReadingOrder = append(accessibility.ReadingOrder, node.ID)
		accessibility.Descriptions[node.ID] = node.AccessibleLabel
	}
	for _, relationship := range relationships {
		accessibility.ReadingOrder = append(accessibility.ReadingOrder, relationship.ID)
		accessibility.Descriptions[relationship.ID] = relationship.AccessibleLabel
	}
	for _, cycle := range cycles {
		accessibility.ReadingOrder = append(accessibility.ReadingOrder, cycle.ID)
		accessibility.Descriptions[cycle.ID] = cycle.Label
	}
	for _, diagnostic := range diagnostics {
		accessibility.ReadingOrder = append(accessibility.ReadingOrder, diagnostic.ID)
		accessibility.Descriptions[diagnostic.ID] = diagnostic.Label
	}
	for _, detail := range references {
		accessibilityID := "reference-detail:" + detail.ID
		accessibility.ReadingOrder = append(accessibility.ReadingOrder, accessibilityID)
		accessibility.Descriptions[accessibilityID] = detail.AccessibleLabel
	}
	return accessibility
}
