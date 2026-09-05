package profile

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
	"github.com/buffo/arch-view/internal/okf/presentation"
)

const (
	DefaultProfileID = "builtin:neutral"
	FogProfileID     = "builtin:fog-of-war"
	HardMaxNodes     = 3000
	HardMaxRelations = 30000
	MaxNodeFields    = 3
	MaxNodeFieldText = 80
)

type Registry struct {
	mu                  sync.RWMutex
	strategies          map[string]ports.RuleStrategy
	profiles            map[string]domain.Profile
	projectProfileID    map[string]struct{}
	shapes              *presentation.ShapeRegistry
	detailRenderers     map[string]registeredDetailRenderer
	diagnosticProviders map[string]registeredDiagnosticProvider
	extensionMetadata   map[string]ports.Extension
}

func NewRegistry() *Registry {
	registry := &Registry{strategies: make(map[string]ports.RuleStrategy), profiles: make(map[string]domain.Profile), projectProfileID: make(map[string]struct{})}
	registry.shapes = presentation.NewDefaultShapeRegistry()
	registry.detailRenderers = make(map[string]registeredDetailRenderer)
	_ = registry.RegisterDetailRenderer(commonMarkRenderer{})
	for _, strategy := range []ports.RuleStrategy{
		metadataEqualsStrategy{}, metadataContainsStrategy{}, stateMappingStrategy{}, labelTemplateStrategy{}, visibilityStrategy{},
	} {
		_ = registry.Register(strategy)
	}
	for _, value := range builtins() {
		registry.profiles[value.ProfileID] = domain.CloneProfile(value)
	}
	return registry
}

func (registry *Registry) Register(strategy ports.RuleStrategy) (err error) {
	return registry.registerStrategy(strategy, nil)
}

func (registry *Registry) registerStrategy(strategy ports.RuleStrategy, metadata *ports.Extension) (err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("rule registration failed: %v", failure)
		}
	}()
	if strategy == nil {
		return fmt.Errorf("rule strategy is required")
	}
	id, version := strategy.ID(), strategy.Version()
	if !domain.ValidExtensionIdentity(id, version) {
		return fmt.Errorf("rule strategy requires a namespaced id and unambiguous version")
	}
	key := id + "@" + version
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.strategies[key]; exists {
		return fmt.Errorf("rule strategy %q is already registered", key)
	}
	registry.strategies[key] = strategy
	if metadata != nil {
		if registry.extensionMetadata == nil {
			registry.extensionMetadata = make(map[string]ports.Extension)
		}
		registry.extensionMetadata[key] = *metadata
	}
	return nil
}

func (registry *Registry) Resolve(id, version string) (ports.RuleStrategy, bool) {
	if version == "" {
		version = "1"
	}
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	strategy, ok := registry.strategies[id+"@"+version]
	return strategy, ok
}

func (registry *Registry) Catalog() []ports.Extension {
	registry.mu.RLock()
	strategies := make([]ports.RuleStrategy, 0, len(registry.strategies))
	for _, strategy := range registry.strategies {
		strategies = append(strategies, strategy)
	}
	registry.mu.RUnlock()
	result := make([]ports.Extension, 0, len(strategies))
	for _, strategy := range strategies {
		extension := ports.Extension{ID: strategy.ID(), Version: strategy.Version(), Kind: "rule", Description: strategy.Description(), Capabilities: []string{"renderer-neutral", "deterministic"}}
		if provider, ok := strategy.(ports.ParameterSchemaProvider); ok {
			extension.ParameterSchema = domain.CloneMap(provider.ParameterSchema())
		}
		registry.mu.RLock()
		metadata, custom := registry.extensionMetadata[extension.ID+"@"+extension.Version]
		registry.mu.RUnlock()
		if custom {
			extension = cloneExtensionMetadata(metadata)
		}
		result = append(result, extension)
	}
	sort.SliceStable(result, func(left, right int) bool {
		if result[left].ID != result[right].ID {
			return result[left].ID < result[right].ID
		}
		return result[left].Version < result[right].Version
	})
	return result
}

func (registry *Registry) SetProjectProfiles(values []domain.Profile) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	for profileID := range registry.projectProfileID {
		delete(registry.profiles, profileID)
	}
	registry.projectProfileID = make(map[string]struct{}, len(values))
	for _, value := range values {
		if value.ProfileID == "" || strings.HasPrefix(value.ProfileID, "builtin:") {
			continue
		}
		value.Origin = "project_local"
		value.Immutable = false
		value = declaration(value)
		value.Revision = domain.ProfileRevision(value)
		registry.profiles[value.ProfileID] = domain.CloneProfile(value)
		registry.projectProfileID[value.ProfileID] = struct{}{}
	}
}

func (registry *Registry) RemoveProjectProfile(profileID string) {
	if strings.HasPrefix(profileID, "project:") {
		registry.mu.Lock()
		defer registry.mu.Unlock()
		delete(registry.profiles, profileID)
		delete(registry.projectProfileID, profileID)
	}
}

func (registry *Registry) Profile(profileID string) (domain.Profile, bool) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	value, ok := registry.profiles[profileID]
	return domain.CloneProfile(value), ok
}

func (registry *Registry) Profiles() []domain.Profile {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	result := make([]domain.Profile, 0, len(registry.profiles))
	for _, value := range registry.profiles {
		result = append(result, domain.CloneProfile(value))
	}
	sort.SliceStable(result, func(left, right int) bool { return result[left].ProfileID < result[right].ProfileID })
	return result
}

func (registry *Registry) ResolveProfile(profileID string) (domain.Profile, []domain.Diagnostic) {
	if profileID == "" {
		profileID = DefaultProfileID
	}
	registry.mu.RLock()
	value, diagnostics := registry.resolve(profileID, nil)
	registry.mu.RUnlock()
	parameterDiagnostics := registry.validateResolvedParameters(value)
	parameterDiagnostics = append(parameterDiagnostics, registry.validateShapeReferences(value)...)
	parameterDiagnostics = append(parameterDiagnostics, registry.validateDetailRenderer(value)...)
	diagnostics = append(diagnostics, parameterDiagnostics...)
	if len(parameterDiagnostics) > 0 {
		value.Status = domain.ProfileInvalid
	}
	if value.ProfileID != "" {
		value.Revision = domain.ProfileRevision(value)
	}
	return value, diagnostics
}

func (registry *Registry) resolve(profileID string, trail []string) (domain.Profile, []domain.Diagnostic) {
	value, exists := registry.profiles[profileID]
	if !exists {
		return domain.Profile{}, []domain.Diagnostic{{Code: "okf_profile_not_found", Severity: "error", Category: "profile", ProfileID: profileID, Message: "The requested profile is not available.", Recovery: "Choose another profile or repair the project configuration."}}
	}
	for _, item := range trail {
		if item == profileID {
			return domain.CloneProfile(value), []domain.Diagnostic{{Code: "okf_profile_cycle", Severity: "error", Category: "profile", ProfileID: profileID, Message: "Profile inheritance contains a cycle.", Recovery: "Remove the cyclic base reference."}}
		}
	}
	result := domain.Profile{}
	diagnostics := make([]domain.Diagnostic, 0)
	for _, baseID := range value.Bases {
		base, baseDiagnostics := registry.resolve(baseID, append(trail, profileID))
		diagnostics = append(diagnostics, baseDiagnostics...)
		if base.ProfileID != "" {
			result = merge(result, base)
		}
	}
	result = merge(result, value)
	result.ProfileID = value.ProfileID
	result.Name = value.Name
	result.Origin = value.Origin
	result.Immutable = value.Immutable
	result.Revision = value.Revision
	result.Status = domain.ProfileValid
	if len(trail) > 0 {
		return result, diagnostics
	}
	result = normalize(result)
	for _, rule := range result.Rules {
		if !rule.Enabled {
			continue
		}
		if _, ok := registry.strategies[strategyKey(rule.RuleID, rule.Version)]; !ok {
			diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_rule_not_found", Severity: "warning", Category: "rule", ProfileID: profileID, Message: "A configured rule is not registered and was skipped.", Details: map[string]any{"rule_id": rule.RuleID, "version": rule.Version}})
			result.Status = domain.ProfilePartiallyApplied
		}
	}
	if len(diagnostics) > 0 && result.Status == domain.ProfileValid {
		result.Status = domain.ProfilePartiallyApplied
	}
	if result.Navigation.DefaultDepth < 1 || result.Navigation.MaxNodes < 1 || result.Navigation.MaxNodes > HardMaxNodes || result.Navigation.MaxRelationships < 1 || result.Navigation.MaxRelationships > HardMaxRelations {
		diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_limit_invalid", Severity: "error", Category: "scale", ProfileID: profileID, Message: "Profile limits must be positive and within the application hard caps."})
	}
	diagnostics = append(diagnostics, validateNodeFields(result)...)
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "okf_profile_not_found" || diagnostic.Code == "okf_profile_cycle" || diagnostic.Code == "okf_limit_invalid" || diagnostic.Code == "okf_node_fields_invalid" {
			result.Status = domain.ProfileInvalid
			break
		}
	}
	return result, diagnostics
}

func strategyKey(id, version string) string {
	if version == "" {
		version = "1"
	}
	return id + "@" + version
}

func Validate(value domain.Profile, registry *Registry) (domain.Profile, []domain.Diagnostic) {
	if registry == nil {
		registry = NewRegistry()
	}
	value = declaration(value)
	effective, _ := registry.ResolveCandidate(value)
	diagnostics := make([]domain.Diagnostic, 0)
	if value.ProfileID == "" {
		diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_profile_invalid", Severity: "error", Category: "profile", Message: "Profile ID is required."})
	}
	if value.Origin == "builtin" || strings.HasPrefix(value.ProfileID, "builtin:") {
		value.Origin = "builtin"
		value.Immutable = true
	}
	if effective.Navigation.MaxNodes < 1 || effective.Navigation.MaxNodes > HardMaxNodes || effective.Navigation.MaxRelationships < 1 || effective.Navigation.MaxRelationships > HardMaxRelations {
		diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_limit_invalid", Severity: "error", Category: "scale", ProfileID: value.ProfileID, Message: "Profile limits must be positive and within the application hard caps."})
	}
	diagnostics = append(diagnostics, validateNodeFields(effective)...)
	diagnostics = append(diagnostics, registry.validateShapeReferences(effective)...)
	diagnostics = append(diagnostics, registry.validateDetailRenderer(effective)...)
	if effective.Navigation.DefaultDepth < 1 {
		diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_depth_invalid", Severity: "error", Category: "navigation", ProfileID: value.ProfileID, Message: "Default depth must be at least one."})
	}
	diagnostics = append(diagnostics, validateBases(value, registry)...)
	for _, rule := range effective.Rules {
		if !rule.Enabled {
			continue
		}
		if strategy, ok := registry.Resolve(rule.RuleID, rule.Version); !ok {
			diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_rule_not_found", Severity: "error", Category: "rule", ProfileID: value.ProfileID, Message: "Profile references an unavailable rule.", Details: map[string]any{"rule_id": rule.RuleID}})
		} else if err := validateRuleParameters(strategy, rule.Parameters); err != nil {
			diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_rule_invalid", Severity: "error", Category: "rule", ProfileID: value.ProfileID, Message: err.Error(), Details: map[string]any{"rule_id": rule.RuleID}})
		}
	}
	if len(diagnostics) > 0 {
		value.Status = domain.ProfileInvalid
	} else {
		value.Status = domain.ProfileValid
	}
	return value, diagnostics
}

func validateBases(value domain.Profile, registry *Registry) []domain.Diagnostic {
	diagnostics := make([]domain.Diagnostic, 0)
	var visit func(string, []string)
	visit = func(profileID string, trail []string) {
		for _, item := range trail {
			if item == profileID {
				diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_profile_cycle", Severity: "error", Category: "profile", ProfileID: value.ProfileID, Message: "Profile inheritance contains a cycle.", Details: map[string]any{"profile_id": profileID}})
				return
			}
		}
		base, exists := registry.Profile(profileID)
		if !exists {
			diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_profile_not_found", Severity: "error", Category: "profile", ProfileID: value.ProfileID, Message: "Profile references an unavailable base profile.", Details: map[string]any{"base_profile_id": profileID}})
			return
		}
		for _, baseID := range base.Bases {
			visit(baseID, append(trail, profileID))
		}
	}
	for _, baseID := range value.Bases {
		visit(baseID, []string{value.ProfileID})
	}
	return diagnostics
}

func (registry *Registry) Evaluate(ctx context.Context, document domain.ConceptDocument, value domain.Profile) (ports.RuleResult, []domain.Diagnostic) {
	rules := append([]domain.RuleInvocation(nil), value.Rules...)
	sort.SliceStable(rules, func(left, right int) bool {
		if rules[left].Priority != rules[right].Priority {
			return rules[left].Priority > rules[right].Priority
		}
		if rules[left].RuleID != rules[right].RuleID {
			return rules[left].RuleID < rules[right].RuleID
		}
		return rules[left].Version < rules[right].Version
	})
	result := ports.RuleResult{Annotations: make(map[string]any)}
	diagnostics := make([]domain.Diagnostic, 0)
	priority := make(map[string]int)
	for _, invocation := range rules {
		if !invocation.Enabled {
			continue
		}
		strategy, ok := registry.Resolve(invocation.RuleID, invocation.Version)
		if !ok {
			continue
		}
		output, err := evaluateRule(ctx, strategy, document, invocation)
		if err != nil {
			diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_rule_invalid", Severity: "warning", Category: "rule", Message: err.Error(), Details: map[string]any{"rule_id": invocation.RuleID}})
			continue
		}
		diagnostics = append(diagnostics, output.Diagnostics...)
		applyResult(&result, priority, invocation.Priority, output, &diagnostics, document.ConceptID)
	}
	return result, diagnostics
}

func applyResult(result *ports.RuleResult, priorities map[string]int, rulePriority int, output ports.RuleResult, diagnostics *[]domain.Diagnostic, conceptID string) {
	applyScalar := func(field string, current *string, value string) {
		if value == "" {
			return
		}
		if old, exists := priorities[field]; !exists || rulePriority > old {
			*current = value
			priorities[field] = rulePriority
			return
		} else if rulePriority == old && *current != value {
			*diagnostics = append(*diagnostics, domain.Diagnostic{Code: "okf_rule_conflict", Severity: "warning", Category: "rule", ConceptID: conceptID, Message: "Equal-priority rules produced conflicting scalar presentation values.", Details: map[string]any{"field": field}})
			*current = ""
		}
	}
	if output.Visible != nil {
		if old, exists := priorities["visible"]; !exists || rulePriority > old {
			result.Visible = output.Visible
			priorities["visible"] = rulePriority
		} else if rulePriority == old && result.Visible != nil && *result.Visible != *output.Visible {
			*diagnostics = append(*diagnostics, domain.Diagnostic{Code: "okf_rule_conflict", Severity: "warning", Category: "rule", ConceptID: conceptID, Message: "Equal-priority visibility rules conflicted; the node remains neutral."})
			result.Visible = nil
		}
	}
	applyScalar("role", &result.Role, output.Role)
	applyScalar("token", &result.Token, output.Token)
	applyScalar("shape", &result.Shape, output.Shape)
	applyScalar("label", &result.Label, output.Label)
	applyScalar("effective_state", &result.EffectiveState, output.EffectiveState)
	keys := make([]string, 0, len(output.Annotations))
	for key := range output.Annotations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		applyAnnotation(result, priorities, rulePriority, key, output.Annotations[key], diagnostics, conceptID)
	}
}

func merge(base, overlay domain.Profile) domain.Profile {
	if base.ProfileID == "" {
		base = domain.CloneProfile(overlay)
		return base
	}
	result := domain.CloneProfile(base)
	result = domain.MergeProfileExtensions(result, overlay)
	if overlay.Name != "" {
		result.Name = overlay.Name
	}
	if overlay.Origin != "" {
		result.Origin = overlay.Origin
	}
	if overlay.Bases != nil {
		result.Bases = append([]string(nil), overlay.Bases...)
	}
	if overlay.Rules != nil {
		result.Rules = append([]domain.RuleInvocation(nil), overlay.Rules...)
	}
	if overlay.Hierarchy.Configured || overlay.Hierarchy != (domain.HierarchySettings{}) {
		full := !overlay.Hierarchy.ExplicitConfigured && !overlay.Hierarchy.FilesystemConfigured
		if full || overlay.Hierarchy.ExplicitConfigured {
			result.Hierarchy.UseExplicit = overlay.Hierarchy.UseExplicit
		}
		if full || overlay.Hierarchy.FilesystemConfigured {
			result.Hierarchy.UseFilesystem = overlay.Hierarchy.UseFilesystem
		}
		result.Hierarchy.Configured = true
		result.Hierarchy.ExplicitConfigured, result.Hierarchy.FilesystemConfigured = true, true
	}
	if overlay.Relationships.Configured || overlay.Relationships != (domain.RelationshipSettings{}) {
		full := !overlay.Relationships.ContainmentConfigured && !overlay.Relationships.SemanticConfigured
		if full || overlay.Relationships.ContainmentConfigured {
			result.Relationships.ShowContainment = overlay.Relationships.ShowContainment
		}
		if full || overlay.Relationships.SemanticConfigured {
			result.Relationships.ShowSemantic = overlay.Relationships.ShowSemantic
		}
		result.Relationships.Configured = true
		result.Relationships.ContainmentConfigured, result.Relationships.SemanticConfigured = true, true
	}
	if overlay.State.FieldConfigured || overlay.State.Field != "" {
		result.State.Field = overlay.State.Field
		result.State.FieldConfigured = true
	}
	if overlay.State.MappingConfigured || overlay.State.Mapping != nil {
		result.State.Mapping = cloneStringMap(overlay.State.Mapping)
		result.State.MappingConfigured = true
	}
	if overlay.State.RollUpConfigured || overlay.State.RollUp {
		result.State.RollUp = overlay.State.RollUp
		result.State.RollUpConfigured = true
	}
	if overlay.State.ShowDeclaredConfigured || overlay.State.ShowDeclared {
		result.State.ShowDeclared = overlay.State.ShowDeclared
		result.State.ShowDeclaredConfigured = true
	}
	if overlay.NodeFields != nil {
		result.NodeFields = append([]domain.NodeField(nil), overlay.NodeFields...)
	}
	if overlay.Navigation.DepthConfigured || overlay.Navigation.DefaultDepth != 0 {
		result.Navigation.DefaultDepth = overlay.Navigation.DefaultDepth
		result.Navigation.DepthConfigured = true
	}
	if overlay.Navigation.NodesConfigured || overlay.Navigation.MaxNodes != 0 {
		result.Navigation.MaxNodes = overlay.Navigation.MaxNodes
		result.Navigation.NodesConfigured = true
	}
	if overlay.Navigation.RelationshipsConfigured || overlay.Navigation.MaxRelationships != 0 {
		result.Navigation.MaxRelationships = overlay.Navigation.MaxRelationships
		result.Navigation.RelationshipsConfigured = true
	}
	if overlay.Style.DefaultToken != "" {
		result.Style.DefaultToken = overlay.Style.DefaultToken
	}
	if overlay.Style.Tokens != nil {
		result.Style.Tokens = cloneTokens(overlay.Style.Tokens)
	}
	if overlay.Style.StateTokens != nil {
		result.Style.StateTokens = cloneStringMap(overlay.Style.StateTokens)
	}
	if overlay.Style.Decorations != nil {
		if result.Style.Decorations == nil {
			result.Style.Decorations = make(map[string]domain.StyleDecoration)
		}
		for key, decoration := range overlay.Style.Decorations {
			result.Style.Decorations[key] = decoration
		}
	}
	if overlay.Details.RawConfigured || overlay.Details.ShowRawMarkdown {
		result.Details.ShowRawMarkdown = overlay.Details.ShowRawMarkdown
		result.Details.RawConfigured = true
	}
	if overlay.Details.Renderer != nil {
		result.Details.Renderer = domain.CloneProfile(overlay).Details.Renderer
	}
	if overlay.Details.UnknownConfigured || overlay.Details.ShowUnknown {
		result.Details.ShowUnknown = overlay.Details.ShowUnknown
		result.Details.UnknownConfigured = true
	}
	if overlay.Layout.Algorithm != "" {
		result.Layout.Algorithm = overlay.Layout.Algorithm
	}
	if overlay.Layout.Options != nil {
		result.Layout.Options = domain.CloneMap(overlay.Layout.Options)
	}
	if overlay.Layout.Features != nil {
		result.Layout.Features = append([]string{}, overlay.Layout.Features...)
	}
	return result
}

func validateNodeFields(value domain.Profile) []domain.Diagnostic {
	if len(value.NodeFields) == 0 {
		return nil
	}
	diagnostics := make([]domain.Diagnostic, 0)
	seen := make(map[string]struct{}, len(value.NodeFields))
	for index, field := range value.NodeFields {
		source := strings.TrimSpace(field.Source)
		valid := source != ""
		lowerSource := strings.ToLower(source)
		if strings.HasPrefix(lowerSource, "frontmatter.") && strings.TrimSpace(source[len("frontmatter."):]) == "" {
			valid = false
		}
		if strings.HasPrefix(lowerSource, "frontmatter:") && strings.TrimSpace(source[len("frontmatter:"):]) == "" {
			valid = false
		}
		if field.MaxLength < 0 || field.MaxLength > MaxNodeFieldText {
			valid = false
		}
		if _, exists := seen[source]; exists {
			valid = false
		}
		seen[source] = struct{}{}
		if !valid {
			diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_node_fields_invalid", Severity: "error", Category: "profile", ProfileID: value.ProfileID, Message: "Node field configuration is invalid.", Details: map[string]any{"index": index, "source": field.Source, "max_length": field.MaxLength}, Recovery: "Use up to three unique sources and keep max_length between 0 and 80."})
		}
	}
	if len(value.NodeFields) > MaxNodeFields {
		diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_node_fields_invalid", Severity: "error", Category: "profile", ProfileID: value.ProfileID, Message: "A profile may configure at most three node fields.", Details: map[string]any{"count": len(value.NodeFields), "max": MaxNodeFields}, Recovery: "Remove optional fields or move them to concept detail."})
	}
	return diagnostics
}

func builtins() []domain.Profile {
	neutral := normalize(domain.Profile{ProfileID: DefaultProfileID, Name: "Neutral", Origin: "builtin", Immutable: true, Revision: "builtin:neutral:v1", Layout: defaultOKFLayout()})
	fog := normalize(domain.Profile{ProfileID: FogProfileID, Name: "Fog of war", Origin: "builtin", Bases: []string{DefaultProfileID}, Immutable: true, Revision: "builtin:fog-of-war:v1", Layout: defaultOKFLayout(), State: domain.StateSettings{Field: "state", Mapping: map[string]string{"foggy": "foggy", "bounded": "bounded", "specified": "specified", "implemented": "implemented"}, RollUp: true, ShowDeclared: true}, NodeFields: []domain.NodeField{{Source: "frontmatter.state", Label: "state"}}, Style: domain.StyleSettings{DefaultToken: "state.unknown", StateTokens: map[string]string{"foggy": "state.foggy", "bounded": "state.bounded", "specified": "state.specified", "implemented": "state.implemented"}, Decorations: map[string]domain.StyleDecoration{
		"root":   {Fill: "#4338ca", Stroke: "#c4b5fd", Text: "#ffffff", StrokeWidth: 2},
		"rollup": {Stroke: "#f472b6", StrokeWidth: 3},
	}}})
	fog.Rules = fogRollupRules()
	return []domain.Profile{neutral, fog}
}

func defaultTokens() map[string]domain.StyleToken {
	return map[string]domain.StyleToken{
		"state.unknown":     {ID: "state.unknown", Fill: "#eef2ff", Stroke: "#7483a9", Text: "#1f2a44", Shape: "rounded_rectangle"},
		"state.foggy":       {ID: "state.foggy", Fill: "#d9dee9", Stroke: "#718096", Text: "#263241", Shape: "rounded_rectangle"},
		"state.bounded":     {ID: "state.bounded", Fill: "#d8f0f2", Stroke: "#287c85", Text: "#164b51", Shape: "rounded_rectangle"},
		"state.specified":   {ID: "state.specified", Fill: "#fff0c2", Stroke: "#9f6b00", Text: "#573b00", Shape: "rounded_rectangle"},
		"state.implemented": {ID: "state.implemented", Fill: "#d9f4df", Stroke: "#24743b", Text: "#153b21", Shape: "rounded_rectangle"},
	}
}

func cloneStringMap(value map[string]string) map[string]string {
	if value == nil {
		return nil
	}
	result := make(map[string]string, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}

func cloneTokens(value map[string]domain.StyleToken) map[string]domain.StyleToken {
	result := make(map[string]domain.StyleToken, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}

func cloneDecorations(value map[string]domain.StyleDecoration) map[string]domain.StyleDecoration {
	result := make(map[string]domain.StyleDecoration, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}
