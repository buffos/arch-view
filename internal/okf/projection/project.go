package projection

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/hierarchy"
	"github.com/buffo/arch-view/internal/okf/ports"
	"github.com/buffo/arch-view/internal/okf/profile"
)

func Build(ctx context.Context, index domain.BundleIndex, effective domain.Profile, navigation domain.NavigationState, registry *profile.Registry) (domain.ProjectionSnapshot, error) {
	if registry == nil {
		registry = profile.NewRegistry()
	}
	if err := contextErr(ctx); err != nil {
		return domain.ProjectionSnapshot{}, domain.ContextOperationError(err, "projection")
	}
	if navigation.Depth == 0 {
		navigation.Depth = effective.Navigation.DefaultDepth
	}
	if navigation.Depth < 1 && !navigation.Full {
		return domain.ProjectionSnapshot{}, domain.NewError("okf_depth_invalid", 400, "navigation depth must be at least one", map[string]any{"depth": navigation.Depth})
	}
	containment, diagnostics := hierarchy.Normalize(index, effective.Hierarchy)
	maxNodes, maxRelationships, limitDiagnostics := safeLimits(index.BundleID, effective)
	diagnostics = append(diagnostics, limitDiagnostics...)
	for indexValue := range diagnostics {
		diagnostics[indexValue].BundleID = index.BundleID
		diagnostics[indexValue].ProfileID = effective.ProfileID
	}
	for _, diagnostic := range index.Diagnostics {
		diagnostics = append(diagnostics, diagnostic)
	}

	if navigation.FocusRoot != "" {
		if _, exists := index.Documents[navigation.FocusRoot]; !exists {
			return domain.ProjectionSnapshot{}, domain.NewError("okf_concept_not_found", 404, "focus root is not in the selected bundle", map[string]any{"concept_id": navigation.FocusRoot})
		}
	}
	children, parents := containmentTree(containment)
	candidates, candidateDepths := traversal(index, children, navigation)
	visible := make([]string, 0, len(candidates))
	visibleSet := make(map[string]bool)
	ruleResults := make(map[string]ports.RuleResult, len(candidates))
	nodeBudgetExceeded := false
	for _, conceptID := range candidates {
		if err := contextErr(ctx); err != nil {
			return domain.ProjectionSnapshot{}, domain.ContextOperationError(err, "projection")
		}
		if len(visible) >= maxNodes {
			nodeBudgetExceeded = true
			break
		}
		document := index.Documents[conceptID]
		result, ruleDiagnostics := registry.Evaluate(ctx, document, effective)
		if err := contextErr(ctx); err != nil {
			return domain.ProjectionSnapshot{}, domain.ContextOperationError(err, "projection")
		}
		for indexValue := range ruleDiagnostics {
			ruleDiagnostics[indexValue].BundleID = index.BundleID
			ruleDiagnostics[indexValue].ProfileID = effective.ProfileID
		}
		diagnostics = append(diagnostics, ruleDiagnostics...)
		ruleResults[conceptID] = result
		if result.Visible != nil && !*result.Visible {
			continue
		}
		visible = append(visible, conceptID)
		visibleSet[conceptID] = true
	}
	truncated := nodeBudgetExceeded
	if nodeBudgetExceeded {
		diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_projection_truncated", Severity: "warning", Category: "scale", BundleID: index.BundleID, ProfileID: effective.ProfileID, Message: "The projection reached the configured node limit; hidden nodes remain available through focus or a smaller scope.", Details: map[string]any{"max_nodes": maxNodes}, Recovery: "Focus a hidden subtree or increase the profile limit within the application cap."})
	}
	nodes := make([]domain.SceneNode, 0, len(visible))
	legendTokens := make(map[string]domain.LegendEntry)
	for _, conceptID := range visible {
		document := index.Documents[conceptID]
		ruleResult := ruleResults[conceptID]
		state, stateDiagnostics, err := profile.EvaluateConceptState(ctx, conceptID, index, effective, children[conceptID], registry, ruleResults)
		if err != nil {
			return domain.ProjectionSnapshot{}, domain.ContextOperationError(err, "projection")
		}
		diagnostics = append(diagnostics, stateDiagnostics...)
		declared, effectiveState, role := state.Declared, state.Effective, state.Role
		annotations := domain.CloneMap(ruleResult.Annotations)
		if annotations == nil {
			annotations = map[string]any{}
		}
		isRollup := state.IsRollup
		if state.RolledUp {
			annotations["state_rollup"] = "structural_children"
		}
		token := ruleResult.Token
		if token == "" {
			token = effective.Style.StateTokens[effectiveState]
		}
		if token == "" {
			token = effective.Style.DefaultToken
		}
		style := effective.Style.Tokens[token]
		isRoot := navigation.FocusRoot == conceptID || parents[conceptID] == ""
		if isRoot {
			style = applyDecoration(style, effective.Style.Decorations["root"])
		}
		if isRollup {
			style = applyDecoration(style, effective.Style.Decorations["rollup"])
		}
		shape := ruleResult.Shape
		if shape == "" {
			shape = style.Shape
		}
		if shape == "" {
			shape = "rounded_rectangle"
		}
		shapeDefinition, knownShape := registry.ResolveShape(shape)
		if !knownShape {
			diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_shape_unsupported", Severity: "warning", Category: "presentation", BundleID: index.BundleID, ProfileID: effective.ProfileID, ConceptID: conceptID, Message: "The selected shape is not registered; using the default shape.", Details: map[string]any{"shape": shape}, Recovery: "Select a registered shape or install its trusted host provider."})
			shape = "rounded_rectangle"
			shapeDefinition, _ = registry.ResolveShape(shape)
		}
		label := ruleResult.Label
		if label == "" {
			label = document.Title
		}
		if strings.TrimSpace(label) == "" {
			label = strings.TrimSuffix(pathBase(document.SourcePath), ".md")
		}
		annotations["source_path"] = document.SourcePath
		nodes = append(nodes, domain.SceneNode{
			ShapeDefinition: &shapeDefinition,
			ID:              conceptID, ConceptID: conceptID, SourcePath: document.SourcePath, Title: label, Label: label, Type: document.Type, Role: role,
			DeclaredState: declared, EffectiveState: effectiveState, PresentationFields: nodeFields(document, effective, declared, effectiveState, role, label), PresentationStyle: style, Shape: shape, PresentationToken: token, Annotations: annotations,
			IsRoot: isRoot, IsRollup: isRollup, HierarchyPath: hierarchyPath(conceptID, parents), ChildrenVisible: hasVisibleChild(conceptID, children, visibleSet), Depth: candidateDepths[conceptID],
		})
		if _, exists := legendTokens[token]; !exists {
			baseStyle := effective.Style.Tokens[token]
			legendShape := baseStyle.Shape
			if legendShape == "" {
				legendShape = "rounded_rectangle"
			}
			legendTokens[token] = domain.LegendEntry{Token: token, Label: tokenLabel(token, effectiveState), Shape: legendShape, Fill: baseStyle.Fill}
		}
	}
	relationships, hiddenRelationships, relationshipBudgetExceeded := projectedRelationships(containment, index.Relationships, visibleSet, effective, maxRelationships)
	if relationshipBudgetExceeded {
		truncated = true
		diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_relationships_truncated", Severity: "warning", Category: "scale", BundleID: index.BundleID, ProfileID: effective.ProfileID, Message: "Some relationships are hidden by the active relationship limit or visibility policy.", Details: map[string]any{"max_relationships": maxRelationships}})
	}
	legend := make([]domain.LegendEntry, 0, len(legendTokens))
	for _, value := range legendTokens {
		legend = append(legend, value)
	}
	sort.SliceStable(legend, func(left, right int) bool { return legend[left].Token < legend[right].Token })
	status := domain.ProjectionReady
	if truncated {
		status = domain.ProjectionTruncated
	}
	projection := domain.ProjectionSnapshot{
		Status: status, ProjectionRevision: projectionRevision(index, effective, navigation),
		Source:     domain.ProjectionSource{BundleID: index.BundleID, SourceRevision: index.SourceRevision},
		Profile:    domain.ProjectionProfile{ProfileID: effective.ProfileID, ProfileRevision: effective.Revision, Layout: domain.LayoutSettings{Algorithm: effective.Layout.Algorithm, Options: domain.CloneMap(effective.Layout.Options)}},
		Navigation: navigation, Nodes: nodes, Relationships: relationships,
		Counts: domain.ProjectionCounts{VisibleNodes: len(nodes), HiddenNodes: len(index.Documents) - len(nodes), VisibleRelationships: len(relationships), HiddenRelationships: hiddenRelationships},
		Legend: legend, Diagnostics: boundDiagnostics(diagnostics), GeneratedAt: time.Now().UTC(),
	}
	if len(projection.Nodes) == 0 && len(index.Documents) > 0 {
		projection.Diagnostics = append(projection.Diagnostics, domain.Diagnostic{Code: "okf_projection_empty", Severity: "info", Category: "navigation", BundleID: index.BundleID, ProfileID: effective.ProfileID, Message: "No concepts are visible under the current profile and navigation scope."})
	}
	if err := contextErr(ctx); err != nil {
		return domain.ProjectionSnapshot{}, domain.ContextOperationError(err, "projection")
	}
	return projection, nil
}

func containmentTree(relationships []domain.Relationship) (map[string][]string, map[string]string) {
	children := make(map[string][]string)
	parents := make(map[string]string)
	for _, relationship := range relationships {
		children[relationship.From] = append(children[relationship.From], relationship.To)
		parents[relationship.To] = relationship.From
	}
	for key := range children {
		sort.Strings(children[key])
	}
	return children, parents
}

func traversal(index domain.BundleIndex, children map[string][]string, navigation domain.NavigationState) ([]string, map[string]int) {
	roots := make([]string, 0)
	if navigation.FocusRoot != "" {
		roots = append(roots, navigation.FocusRoot)
	} else {
		seen := make(map[string]bool)
		for _, childList := range children {
			for _, child := range childList {
				seen[child] = true
			}
		}
		for _, conceptID := range index.ConceptOrder {
			if !seen[conceptID] {
				roots = append(roots, conceptID)
			}
		}
		if len(roots) == 0 {
			roots = append(roots, index.ConceptOrder...)
		}
	}
	depths := make(map[string]int)
	result := make([]string, 0, len(index.ConceptOrder))
	var visit func(string, int)
	visit = func(conceptID string, depth int) {
		if _, exists := depths[conceptID]; exists {
			return
		}
		depths[conceptID] = depth
		result = append(result, conceptID)
		if !navigation.Full && depth >= navigation.Depth {
			return
		}
		for _, child := range children[conceptID] {
			visit(child, depth+1)
		}
	}
	for _, root := range roots {
		visit(root, 0)
	}
	return result, depths
}

func projectedRelationships(containment []domain.Relationship, semantic []domain.Relationship, visible map[string]bool, effective domain.Profile, limit int) ([]domain.SceneRelationship, int, bool) {
	all := make([]domain.Relationship, 0, len(containment)+len(semantic))
	if effective.Relationships.ShowContainment {
		all = append(all, containment...)
	}
	if effective.Relationships.ShowSemantic {
		all = append(all, semantic...)
	}
	sort.SliceStable(all, func(left, right int) bool {
		if all[left].Kind != all[right].Kind {
			return all[left].Kind < all[right].Kind
		}
		return all[left].RelationshipID < all[right].RelationshipID
	})
	result := make([]domain.SceneRelationship, 0, len(all))
	hidden := len(containment) + len(semantic) - len(all)
	budgetExceeded := false
	for _, relationship := range all {
		if !visible[relationship.From] || !visible[relationship.To] {
			hidden++
			continue
		}
		if len(result) >= limit {
			hidden++
			budgetExceeded = true
			continue
		}
		result = append(result, domain.SceneRelationship{ID: relationship.RelationshipID, RelationshipID: relationship.RelationshipID, Kind: relationship.Kind, From: relationship.From, To: relationship.To, FromVisibleID: relationship.From, ToVisibleID: relationship.To, Label: relationship.Kind, AccessibleLabel: relationship.Kind + " from " + relationship.From + " to " + relationship.To, Provenance: relationship.Provenance})
	}
	return result, hidden, budgetExceeded
}

func applyDecoration(token domain.StyleToken, decoration domain.StyleDecoration) domain.StyleToken {
	if decoration.Fill != "" {
		token.Fill = decoration.Fill
	}
	if decoration.Stroke != "" {
		token.Stroke = decoration.Stroke
	}
	if decoration.Text != "" {
		token.Text = decoration.Text
	}
	if decoration.StrokeWidth > 0 {
		token.StrokeWidth = decoration.StrokeWidth
	}
	if decoration.StrokeDasharray != "" {
		token.StrokeDasharray = decoration.StrokeDasharray
	}
	return token
}

const defaultNodeFieldMaxLength = 48

func nodeFields(document domain.ConceptDocument, effective domain.Profile, declared, effectiveState, role, label string) []domain.NodeFieldValue {
	fields := make([]domain.NodeFieldValue, 0, len(effective.NodeFields))
	for _, configured := range effective.NodeFields {
		source := strings.TrimSpace(configured.Source)
		value, ok := nodeFieldValue(document, source, declared, effectiveState, role, label)
		if !ok || strings.TrimSpace(value) == "" {
			continue
		}
		fields = append(fields, domain.NodeFieldValue{Source: source, Label: nodeFieldLabel(configured.Label, source), Value: truncateNodeField(value, configured.MaxLength)})
	}
	return fields
}

func nodeFieldValue(document domain.ConceptDocument, source, declared, effectiveState, role, label string) (string, bool) {
	normalized := strings.ToLower(source)
	switch normalized {
	case "concept_id":
		return document.ConceptID, document.ConceptID != ""
	case "title", "label":
		return label, label != ""
	case "description":
		return document.Description, document.Description != ""
	case "type":
		return document.Type, document.Type != ""
	case "role":
		return role, role != ""
	case "source_path":
		return document.SourcePath, document.SourcePath != ""
	case "declared_state":
		return declared, declared != ""
	case "effective_state":
		return effectiveState, effectiveState != ""
	case "tags":
		return strings.Join(document.Tags, ", "), len(document.Tags) > 0
	}
	key := ""
	if strings.HasPrefix(normalized, "frontmatter.") {
		key = source[len("frontmatter."):]
	} else if strings.HasPrefix(normalized, "frontmatter:") {
		key = source[len("frontmatter:"):]
	}
	if key == "" {
		return "", false
	}
	value, exists := document.Frontmatter[key]
	if !exists {
		return "", false
	}
	return stringifyNodeField(value)
}

func stringifyNodeField(value any) (string, bool) {
	if value == nil {
		return "", false
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text), strings.TrimSpace(text) != ""
	}
	encoded, err := json.Marshal(value)
	if err == nil {
		text := strings.TrimSpace(string(encoded))
		return text, text != "" && text != "null"
	}
	text := strings.TrimSpace(fmt.Sprint(value))
	return text, text != ""
}

func nodeFieldLabel(configured, source string) string {
	if label := strings.TrimSpace(configured); label != "" {
		return label
	}
	label := source
	if index := strings.LastIndexAny(label, ".:"); index >= 0 {
		label = label[index+1:]
	}
	label = strings.NewReplacer("_", " ", "-", " ").Replace(label)
	if label == "" {
		return "Value"
	}
	return strings.ToUpper(label[:1]) + label[1:]
}

func truncateNodeField(value string, maximum int) string {
	limit := maximum
	if limit <= 0 {
		limit = defaultNodeFieldMaxLength
	}
	if limit > profile.MaxNodeFieldText {
		limit = profile.MaxNodeFieldText
	}
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	if limit <= 1 {
		return string(runes[:limit])
	}
	return string(runes[:limit-1]) + "…"
}

func hierarchyPath(conceptID string, parents map[string]string) []string {
	pathValues := []string{conceptID}
	seen := map[string]bool{conceptID: true}
	for current := conceptID; ; {
		parent, exists := parents[current]
		if !exists || seen[parent] {
			break
		}
		seen[parent] = true
		pathValues = append([]string{parent}, pathValues...)
		current = parent
	}
	return pathValues
}

func hasVisibleChild(conceptID string, children map[string][]string, visible map[string]bool) bool {
	for _, child := range children[conceptID] {
		if visible[child] {
			return true
		}
	}
	return false
}

func tokenLabel(token, state string) string {
	if state != "" && state != "unknown" {
		return state
	}
	return token
}

func pathBase(value string) string {
	parts := strings.Split(value, "/")
	return parts[len(parts)-1]
}

func safeLimits(bundleID string, effective domain.Profile) (int, int, []domain.Diagnostic) {
	maxNodes := effective.Navigation.MaxNodes
	maxRelationships := effective.Navigation.MaxRelationships
	diagnostics := make([]domain.Diagnostic, 0, 2)
	if maxNodes < 1 {
		maxNodes = 1000
		diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_limit_invalid", Severity: "warning", Category: "scale", BundleID: bundleID, ProfileID: effective.ProfileID, Message: "The profile node limit was invalid; the default safety limit was used.", Details: map[string]any{"max_nodes": effective.Navigation.MaxNodes, "fallback": maxNodes}})
	} else if maxNodes > profile.HardMaxNodes {
		diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_limit_invalid", Severity: "warning", Category: "scale", BundleID: bundleID, ProfileID: effective.ProfileID, Message: "The profile node limit exceeded the application cap; the hard cap was used.", Details: map[string]any{"max_nodes": effective.Navigation.MaxNodes, "hard_cap": profile.HardMaxNodes}})
		maxNodes = profile.HardMaxNodes
	}
	if maxRelationships < 1 {
		maxRelationships = 10000
		diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_limit_invalid", Severity: "warning", Category: "scale", BundleID: bundleID, ProfileID: effective.ProfileID, Message: "The profile relationship limit was invalid; the default safety limit was used.", Details: map[string]any{"max_relationships": effective.Navigation.MaxRelationships, "fallback": maxRelationships}})
	} else if maxRelationships > profile.HardMaxRelations {
		diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_limit_invalid", Severity: "warning", Category: "scale", BundleID: bundleID, ProfileID: effective.ProfileID, Message: "The profile relationship limit exceeded the application cap; the hard cap was used.", Details: map[string]any{"max_relationships": effective.Navigation.MaxRelationships, "hard_cap": profile.HardMaxRelations}})
		maxRelationships = profile.HardMaxRelations
	}
	return maxNodes, maxRelationships, diagnostics
}

func projectionRevision(index domain.BundleIndex, effective domain.Profile, navigation domain.NavigationState) string {
	return "sha256:" + hashText(index.SourceRevision, effective.ProfileID, effective.Revision, navigation.FocusRoot, fmt.Sprint(navigation.Depth), fmt.Sprint(navigation.Full))
}

func hashText(values ...string) string {
	hash := sha256.New()
	for _, value := range values {
		_, _ = hash.Write([]byte(value))
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func contextErr(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func boundDiagnostics(values []domain.Diagnostic) []domain.Diagnostic {
	if len(values) <= 200 {
		return append([]domain.Diagnostic(nil), values...)
	}
	result := append([]domain.Diagnostic(nil), values[:199]...)
	return append(result, domain.Diagnostic{Code: "okf_diagnostics_truncated", Severity: "warning", Category: "scale", Message: "Additional diagnostics were omitted at the projection limit."})
}
