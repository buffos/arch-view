package export

import (
	"fmt"
	"html"
	"sort"
	"strconv"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/routing"
	"github.com/buffo/arch-view/internal/viewer/scene"
)

func renderSVG(value model.Model, request Request) ([]byte, map[string]any, error) {
	if err := checkContext(request.Context); err != nil {
		return nil, nil, err
	}
	sceneSnapshot, err := scene.BuildSceneWithOptions(value, request.ViewPath, "overview", scene.SceneOptions{
		ReferenceVisibility: request.ReferenceVisibility,
		ReferenceScopes:     request.ReferenceScopes,
	})
	if err != nil {
		return nil, nil, err
	}
	layout := buildDeterministicLayout(sceneSnapshot)
	return renderSVGDocument(value, sceneSnapshot, layout), layoutProvenance(), nil
}

func renderSVGDocument(value model.Model, scene scene.SceneSnapshot, layout deterministicLayout) []byte {
	var builder strings.Builder
	width := formatNumber(layout.Width)
	height := formatNumber(layout.Height)
	builder.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" version="1.1" role="img" aria-labelledby="arch-view-title arch-view-description" viewBox="0 0 ` + width + ` ` + height + `" width="` + width + `" height="` + height + `" preserveAspectRatio="xMidYMid meet">`)
	builder.WriteString(`<title id="arch-view-title">` + xmlEscape("Arch View architecture · "+value.Project.RootLabel) + `</title>`)
	builder.WriteString(`<desc id="arch-view-description">` + xmlEscape(svgDescription(scene)) + `</desc>`)
	writeSVGMetadata(&builder, value, scene, layout)
	builder.WriteString(`<style>`)
	builder.WriteString(svgStyles)
	builder.WriteString(`</style><defs><marker id="arch-view-arrow" markerWidth="9" markerHeight="9" refX="8" refY="4.5" orient="auto"><path d="M 0 0 L 9 4.5 L 0 9 z" fill="#7483a9"/></marker></defs>`)
	builder.WriteString(`<g class="edges">`)
	for _, relationship := range scene.VisibleRelationships {
		from, fromOK := layout.Positions[relationship.FromVisibleID]
		to, toOK := layout.Positions[relationship.ToVisibleID]
		if !fromOK || !toOK {
			continue
		}
		pathValue, labelX, labelY := svgEdgeGeometry(relationship, from, to, layout.Edges[relationship.ID])
		className := "edge-line " + svgStateClass(relationship.CycleState)
		builder.WriteString(`<g class="edge-group" data-relationship-id="` + xmlEscape(relationship.ID) + `" data-count="` + strconv.Itoa(relationship.Count) + `" data-cycle-state="` + xmlEscape(relationship.CycleState) + `"`)
		if relationship.TargetScope != "" {
			builder.WriteString(` data-reference-scope="` + xmlEscape(relationship.TargetScope) + `"`)
		}
		if relationship.ConfidenceState != "" {
			builder.WriteString(` data-confidence-state="` + xmlEscape(relationship.ConfidenceState) + `"`)
		}
		if relationship.ConfidenceBasis != "" {
			builder.WriteString(` data-confidence-basis="` + xmlEscape(relationship.ConfidenceBasis) + `"`)
		}
		if relationship.ConfidenceScore != nil {
			builder.WriteString(` data-confidence-score="` + formatNumber(*relationship.ConfidenceScore) + `"`)
		}
		builder.WriteString(` data-evidence-ids="` + xmlEscape(strings.Join(relationship.SourceReferenceIDs, ",")) + `" role="img" aria-label="` + xmlEscape(relationship.AccessibleLabel) + `"><title>` + xmlEscape(relationship.AccessibleLabel) + `</title><path class="` + className + `" d="` + xmlEscape(pathValue) + `" marker-end="url(#arch-view-arrow)"/><text class="edge-label" x="` + formatNumber(labelX) + `" y="` + formatNumber(labelY) + `" text-anchor="middle">` + strconv.Itoa(relationship.Count) + `</text></g>`)
	}
	builder.WriteString(`</g><g class="nodes">`)
	for _, node := range scene.VisibleNodes {
		position, ok := layout.Positions[node.ID]
		if !ok {
			continue
		}
		builder.WriteString(svgNodeMarkup(node, position))
	}
	builder.WriteString(`</g></svg>`)
	return []byte(builder.String())
}

func writeSVGMetadata(builder *strings.Builder, value model.Model, scene scene.SceneSnapshot, layout deterministicLayout) {
	provenance := layoutProvenance()
	builder.WriteString(`<metadata><arch-view schema-version="` + xmlEscape(value.SchemaVersion) + `" model-id="` + xmlEscape(value.ModelID) + `" model-revision="` + xmlEscape(value.ModelID) + `" status="` + xmlEscape(string(value.Status)) + `" reference-visibility="` + xmlEscape(scene.ReferenceVisibility) + `" layout-engine="` + xmlEscape(fmt.Sprint(provenance["engine"])) + `" layout-algorithm="` + xmlEscape(fmt.Sprint(provenance["algorithm"])) + `" layout-algorithm-version="` + xmlEscape(fmt.Sprint(provenance["algorithm_version"])) + `" layout-width="` + formatNumber(layout.Width) + `" layout-height="` + formatNumber(layout.Height) + `">`)
	builder.WriteString(`<summary visible-nodes="` + strconv.Itoa(scene.Summary.VisibleNodeCount) + `" visible-relationships="` + strconv.Itoa(scene.Summary.VisibleRelationshipCount) + `" modules="` + strconv.Itoa(scene.Summary.ModuleCount) + `" references="` + strconv.Itoa(scene.Summary.ReferenceCount) + `" cycles="` + strconv.Itoa(scene.Summary.CycleCount) + `" diagnostics="` + strconv.Itoa(scene.Summary.DiagnosticCount) + `" evidence="` + strconv.Itoa(scene.Summary.EvidenceCount) + `"/>`)
	builder.WriteString(`<references>`)
	referenceContributors := make(map[string][]analysis.RelationshipObservation, len(value.References))
	for _, relationship := range value.Relationships {
		if relationship.ToReferenceID != "" {
			referenceContributors[relationship.ToReferenceID] = append(referenceContributors[relationship.ToReferenceID], relationship)
		}
	}
	for _, reference := range value.References {
		contributors := referenceContributors[reference.ID]
		relationshipIDs := make([]string, 0, len(contributors))
		sourceReferenceIDs := []string{}
		for _, relationship := range contributors {
			relationshipIDs = append(relationshipIDs, relationship.ID)
			sourceReferenceIDs = appendUniqueStrings(sourceReferenceIDs, relationship.SourceReferenceIDs...)
		}
		sort.Strings(relationshipIDs)
		sort.Strings(sourceReferenceIDs)
		confidenceState, confidenceBasis, confidenceScore := referenceConfidence(contributors)
		builder.WriteString(`<reference id="` + xmlEscape(reference.ID) + `" name="` + xmlEscape(reference.Name) + `" scope="` + xmlEscape(reference.Scope) + `" relationship-count="` + strconv.Itoa(len(relationshipIDs)) + `" relationship-ids="` + xmlEscape(strings.Join(relationshipIDs, ",")) + `" source-reference-ids="` + xmlEscape(strings.Join(sourceReferenceIDs, ",")) + `" confidence-state="` + xmlEscape(confidenceState) + `"`)
		if confidenceBasis != "" {
			builder.WriteString(` confidence-basis="` + xmlEscape(confidenceBasis) + `"`)
		}
		if confidenceScore != nil {
			builder.WriteString(` confidence-score="` + formatNumber(*confidenceScore) + `"`)
		}
		builder.WriteString(`/>`)
	}
	builder.WriteString(`</references><evidence>`)
	for _, source := range value.SourceReferences {
		builder.WriteString(`<source-reference id="` + xmlEscape(source.ID) + `" path="` + xmlEscape(source.Path) + `" kind="` + xmlEscape(source.Kind) + `"`)
		if source.Start != nil {
			builder.WriteString(` start="` + positionValue(source.Start) + `"`)
		}
		if source.End != nil {
			builder.WriteString(` end="` + positionValue(source.End) + `"`)
		}
		if source.Symbol != "" {
			builder.WriteString(` symbol="` + xmlEscape(source.Symbol) + `"`)
		}
		builder.WriteString(`/>`)
	}
	builder.WriteString(`</evidence><cycles>`)
	for _, cycle := range value.Derived.Cycles {
		builder.WriteString(`<cycle id="` + xmlEscape(cycle.ID) + `" module-ids="` + xmlEscape(strings.Join(cycle.ModuleIDs, ",")) + `" relationship-ids="` + xmlEscape(strings.Join(cycle.RelationshipIDs, ",")) + `"/>`)
	}
	builder.WriteString(`</cycles><diagnostics>`)
	for _, diagnostic := range value.Diagnostics {
		builder.WriteString(`<diagnostic id="` + xmlEscape(diagnostic.ID) + `" code="` + xmlEscape(diagnostic.Code) + `" severity="` + xmlEscape(diagnostic.Severity) + `" recoverable="` + strconv.FormatBool(diagnostic.Recoverable) + `" message="` + xmlEscape(diagnostic.Message) + `"`)
		if diagnostic.Subject != "" {
			builder.WriteString(` subject="` + xmlEscape(diagnostic.Subject) + `"`)
		}
		if diagnostic.Path != "" {
			builder.WriteString(` path="` + xmlEscape(diagnostic.Path) + `"`)
		}
		builder.WriteString(` source-reference-ids="` + xmlEscape(strings.Join(diagnostic.SourceReferenceIDs, ",")) + `"/>`)
	}
	builder.WriteString(`</diagnostics><layers>`)
	for _, layer := range value.Derived.Layers {
		builder.WriteString(`<layer number="` + strconv.Itoa(layer.Layer) + `" module-ids="` + xmlEscape(strings.Join(layer.ModuleIDs, ",")) + `"/>`)
	}
	builder.WriteString(`</layers></arch-view></metadata>`)
}

func svgNodeMarkup(node scene.VisibleNode, position deterministicLayoutNode) string {
	className := "node-shape " + svgStateClass(node.Kind) + " " + svgStateClass(node.CycleState)
	if node.DiagnosticState != "none" {
		className += " " + svgStateClass(node.DiagnosticState)
	}
	subtitle := node.Kind
	if node.ReferenceScope != "" {
		subtitle += " · " + strings.ReplaceAll(node.ReferenceScope, "_", " ")
	}
	if node.Layer != nil {
		subtitle += " · L" + strconv.Itoa(*node.Layer)
	} else if len(node.Layers) > 0 {
		layers := make([]string, len(node.Layers))
		for index, layer := range node.Layers {
			layers[index] = "L" + strconv.Itoa(layer)
		}
		subtitle += " · " + strings.Join(layers, ", ")
	} else {
		subtitle += " · —"
	}
	if node.Counts.ModuleCount > 1 {
		subtitle += " · " + strconv.Itoa(node.Counts.ModuleCount) + " modules"
	}
	status := nodeStatus(node)
	return `<g class="node-group" data-module-id="` + xmlEscape(node.ID) + `" data-module-ids="` + xmlEscape(strings.Join(node.ModuleIDs, ",")) + `" data-node-kind="` + xmlEscape(node.Kind) + `" data-cycle-state="` + xmlEscape(node.CycleState) + `" data-diagnostic-state="` + xmlEscape(node.DiagnosticState) + `" data-confidence-state="` + xmlEscape(node.ConfidenceState) + `" data-evidence-ids="` + xmlEscape(strings.Join(node.EvidenceIDs, ",")) + `"` + referenceScopeAttribute(node.ReferenceScope) + ` role="group" aria-label="` + xmlEscape(node.AccessibleLabel) + `"><title>` + xmlEscape(node.AccessibleLabel) + `</title><desc>` + xmlEscape(nodeDetails(node)) + `</desc><rect class="` + className + `" x="` + formatNumber(position.X) + `" y="` + formatNumber(position.Y) + `" width="` + formatNumber(position.Width) + `" height="` + formatNumber(position.Height) + `" rx="12"/><text class="node-label" x="` + formatNumber(position.X+14) + `" y="` + formatNumber(position.Y+30) + `">` + xmlEscape(truncate(node.Label, 25)) + `</text><text class="node-subtitle" x="` + formatNumber(position.X+14) + `" y="` + formatNumber(position.Y+51) + `">` + xmlEscape(truncate(subtitle, 29)) + `</text><text class="node-subtitle" x="` + formatNumber(position.X+14) + `" y="` + formatNumber(position.Y+68) + `">` + xmlEscape(truncate(status, 29)) + `</text></g>`
}

func svgEdgeGeometry(relationship scene.VisibleRelationship, from, to deterministicLayoutNode, routed deterministicLayoutEdge) (string, float64, float64) {
	if len(routed.Points) > 1 {
		route := routing.Polyline(routed.Points, routing.Point{X: routed.LabelX, Y: routed.LabelY})
		if path := pathFromRoute(route); path != "" {
			return path, route.Label.X, route.Label.Y
		}
	}
	if relationship.FromVisibleID == relationship.ToVisibleID {
		route := routing.SelfLoop(nodeBox(from))
		return pathFromRoute(route), route.Label.X, route.Label.Y
	}
	fallback := orthogonalEdge(from, to)
	route := routing.Polyline(fallback.Points, routing.Point{X: fallback.LabelX, Y: fallback.LabelY})
	return pathFromRoute(route), route.Label.X, route.Label.Y
}

func nodeBox(node deterministicLayoutNode) routing.NodeBox {
	return routing.NodeBox{X: node.X, Y: node.Y, Width: node.Width, Height: node.Height}

}

func pathFromRoute(route routing.Route) string {
	parts := make([]string, 0)
	for _, section := range route.Sections {
		parts = append(parts, "M", formatNumber(section.Start.X), formatNumber(section.Start.Y))
		for _, segment := range section.Segments {
			switch segment.Kind {
			case routing.SegmentKindLine:
				parts = append(parts, "L", formatNumber(segment.To.X), formatNumber(segment.To.Y))
			case routing.SegmentKindCubic:
				if segment.Control1 == nil || segment.Control2 == nil {
					return ""
				}
				parts = append(parts,
					"C",
					formatNumber(segment.Control1.X), formatNumber(segment.Control1.Y)+",",
					formatNumber(segment.Control2.X), formatNumber(segment.Control2.Y)+",",
					formatNumber(segment.To.X), formatNumber(segment.To.Y),
				)
			default:
				return ""
			}
		}
	}
	return strings.Join(parts, " ")
}

func referenceScopeAttribute(scope string) string {
	if scope == "" {
		return ""
	}
	return ` data-reference-scope="` + xmlEscape(scope) + `"`
}

func appendUniqueStrings(values []string, additions ...string) []string {
	seen := make(map[string]struct{}, len(values)+len(additions))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, value := range additions {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}

func referenceConfidence(values []analysis.RelationshipObservation) (string, string, *float64) {
	if len(values) == 0 {
		return "unknown", "", nil
	}
	minimum := float64(1)
	hasConfidence := false
	bases := make(map[string]struct{})
	for _, value := range values {
		if value.Confidence == nil {
			continue
		}
		hasConfidence = true
		if value.Confidence.Score < minimum {
			minimum = value.Confidence.Score
		}
		bases[value.Confidence.Basis] = struct{}{}
	}
	if !hasConfidence {
		return "unknown", "", nil
	}
	basis := ""
	if len(bases) == 1 {
		for value := range bases {
			basis = value
		}
	} else {
		basis = "mixed"
	}
	return exportConfidenceState(minimum), basis, &minimum
}

func exportConfidenceState(score float64) string {
	switch {
	case score >= 0.9:
		return "high"
	case score >= 0.6:
		return "medium"
	default:
		return "low"
	}
}

func nodeStatus(node scene.VisibleNode) string {
	if node.CycleState != "none" {
		return "cycle: " + node.CycleState
	}
	if node.DiagnosticState != "none" {
		return "diagnostic: " + node.DiagnosticState
	}
	if node.Counts.InternalRelationshipCount > 0 {
		return strconv.Itoa(node.Counts.InternalRelationshipCount) + " internal relationships"
	}
	if node.ReferenceScope != "" {
		return "reference: " + strings.ReplaceAll(node.ReferenceScope, "_", " ")
	}
	return "identity " + node.IdentityState
}

func nodeDetails(node scene.VisibleNode) string {
	return fmt.Sprintf("%s · %s · %d module(s) · %d relationship(s) · %d evidence item(s)", node.Label, node.Kind, node.Counts.ModuleCount, node.Counts.RelationshipCount, node.Counts.EvidenceCount)
}

func svgDescription(scene scene.SceneSnapshot) string {
	return fmt.Sprintf("%d visible nodes, %d directed relationships, %d cycles, %d diagnostics, and %d evidence links in the %s hierarchy.", scene.Summary.VisibleNodeCount, scene.Summary.VisibleRelationshipCount, scene.Summary.CycleCount, scene.Summary.DiagnosticCount, scene.Summary.EvidenceCount, hierarchyLabel(scene.HierarchyPath))
}

func hierarchyLabel(path []string) string {
	if len(path) == 0 {
		return "top-level"
	}
	return strings.Join(path, " / ")
}

func truncate(value string, length int) string {
	runes := []rune(value)
	if len(runes) <= length {
		return value
	}
	return string(runes[:length-1]) + "…"
}

func formatNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func positionValue(position *analysis.Position) string {
	return strconv.Itoa(position.Line) + ":" + strconv.Itoa(position.Column)
}

func xmlEscape(value string) string {
	return html.EscapeString(value)
}

func svgStateClass(value string) string {
	if value == "" {
		return "none"
	}
	var builder strings.Builder
	for _, character := range strings.ToLower(value) {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') || character == '_' || character == '-' {
			builder.WriteRune(character)
		} else {
			builder.WriteRune('-')
		}
	}
	return builder.String()
}

const svgStyles = `
:root { color-scheme: dark; font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
svg { background: #090e1d; }
.edge-line { fill: none; stroke: #7483a9; stroke-width: 3; vector-effect: non-scaling-stroke; }
.edge-line.cycle { stroke: #f87171; }
.edge-line.feedback { stroke: #fbbf74; stroke-dasharray: 7 5; }
.edge-label { fill: #d8e2ff; font-size: 16px; font-weight: 800; paint-order: stroke; stroke: #090e1d; stroke-width: 6px; stroke-linejoin: round; }
.node-shape { fill: #272343; stroke: #8b5cf6; stroke-width: 3; }
.node-shape.reference { fill: #103c3b; stroke: #2dd4bf; stroke-dasharray: 7 5; }
.node-shape.warning { stroke: #fbbf24; }
.node-shape.error { stroke: #f87171; }
.node-shape.cycle { stroke: #f87171; }
.node-label { fill: #f4f7ff; font-size: 18px; font-weight: 800; }
.node-subtitle { fill: #a7b4d6; font-size: 12px; font-weight: 600; }
`
