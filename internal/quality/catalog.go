package quality

import (
	"reflect"
	"sort"
	"strings"
	"sync"
)

// MetricProvider computes versioned facts from an evaluation input. A
// provider may support more than one metric capability.
type MetricProvider interface {
	ID() string
	Version() string
	Capabilities() []CapabilityDescriptor
	Compute(EvaluationContext) (MetricBatch, error)
}

// QualityRule evaluates one registered rule against the facts available in an
// evaluation context. Rules are additive strategies; the evaluator does not
// switch on rule or language IDs.
type QualityRule interface {
	ID() string
	Version() string
	AssessmentKind() string
	RequiredCapabilities() []string
	Evaluate(EvaluationContext, MetricBatch) (RuleResult, error)
}

// ParameterSchemaProvider lets a rule publish typed configuration metadata
// without forcing that method onto third-party QualityRule implementations.
type ParameterSchemaProvider interface {
	ParameterSchema() ParameterSchema
}

// RuleDescriptorProvider lets a rule publish human-facing catalog metadata.
type RuleDescriptorProvider interface {
	Descriptor() RuleDescriptor
}

// DefaultParameterProvider lets a rule expose a safe, human-facing starting
// configuration for temporary catalog evaluations. Persisted profiles remain
// authoritative; defaults are only used when a viewer user enables a rule for
// the current session.
type DefaultParameterProvider interface {
	DefaultParameters() TypedConfigBlock
}

type ParameterSchema struct {
	Namespace       string                    `json:"namespace"`
	SchemaVersion   string                    `json:"schema_version"`
	Fields          map[string]ParameterField `json:"fields"`
	AllowAdditional bool                      `json:"allow_additional"`
}

type ParameterField struct {
	Kind          string   `json:"kind"`
	Required      bool     `json:"required"`
	AllowedValues []string `json:"allowed_values,omitempty"`
	Minimum       *float64 `json:"minimum,omitempty"`
	Maximum       *float64 `json:"maximum,omitempty"`
}

type RuleDescriptor struct {
	ID                   string          `json:"id"`
	Version              string          `json:"version"`
	AssessmentKind       string          `json:"assessment_kind"`
	RequiredCapabilities []string        `json:"required_capabilities"`
	ParameterSchema      ParameterSchema `json:"parameter_schema"`
	DefaultSeverity      string          `json:"default_severity"`
	Description          string          `json:"description,omitempty"`
	Limitations          []string        `json:"limitations,omitempty"`
}

// Catalog is the open/closed registry for metric and rule strategies. It is
// safe to populate during setup and read concurrently during evaluation.
type Catalog struct {
	mu              sync.RWMutex
	metricProviders map[string]MetricProvider
	qualityRules    map[string]QualityRule
}

func NewCatalog() *Catalog {
	return &Catalog{
		metricProviders: make(map[string]MetricProvider),
		qualityRules:    make(map[string]QualityRule),
	}
}

// NewDefaultCatalog returns a catalog containing the initial source and
// architecture strategies. New strategies can be registered without editing
// the evaluator or a language switch.
func NewDefaultCatalog() *Catalog {
	catalog := NewCatalog()
	for _, provider := range []MetricProvider{newSourceMetricProvider(), newArchitectureMetricProvider(), newSolidMetricProvider()} {
		if err := catalog.RegisterMetricProvider(provider); err != nil {
			panic(err)
		}
	}
	for _, rule := range []QualityRule{
		newFileSizeRule(),
		newCallableSizeRule(),
		newComplexityRule(),
		newNestingRule(),
		newDocumentationRule(),
		newEfferentCouplingRule(),
		newAfferentCouplingRule(),
		newCycleRule(),
		newForbiddenDependencyRule(),
		newLayerDirectionRule(),
	} {
		if err := catalog.RegisterQualityRule(rule); err != nil {
			panic(err)
		}
	}
	for _, rule := range newSolidSignalRules() {
		if err := catalog.RegisterQualityRule(rule); err != nil {
			panic(err)
		}
	}
	return catalog
}

func (catalog *Catalog) RegisterMetricProvider(provider MetricProvider) error {
	if catalog == nil || isNilStrategy(provider) {
		return newQualityError(ErrorCatalogInvalid, "metric provider is nil", nil)
	}
	id := strings.TrimSpace(provider.ID())
	version := strings.TrimSpace(provider.Version())
	if !validNamespacedID(id) || !validVersion(version) {
		return newQualityError(ErrorCatalogInvalid, "metric provider requires a safe ID and version", map[string]any{"id": id, "version": version})
	}
	capabilities := provider.Capabilities()
	seen := make(map[string]struct{}, len(capabilities))
	for _, capability := range capabilities {
		if !validNamespacedID(capability.ID) || !validVersion(capability.Version) {
			return newQualityError(ErrorCatalogInvalid, "metric provider capability requires a safe ID and version", map[string]any{"provider_id": id, "capability": capability.ID})
		}
		if _, exists := seen[capability.ID]; exists {
			return newQualityError(ErrorCatalogInvalid, "metric provider capabilities must be unique", map[string]any{"provider_id": id, "capability": capability.ID})
		}
		seen[capability.ID] = struct{}{}
	}
	key := catalogKey(id, version)
	catalog.mu.Lock()
	defer catalog.mu.Unlock()
	if existing, ok := catalog.metricProviders[key]; ok {
		if sameStrategy(existing, provider) {
			return nil
		}
		return newQualityError(ErrorCatalogConflict, "a different metric provider already uses this ID and version", map[string]any{"id": id, "version": version})
	}
	catalog.metricProviders[key] = provider
	return nil
}

func (catalog *Catalog) RegisterQualityRule(rule QualityRule) error {
	if catalog == nil || isNilStrategy(rule) {
		return newQualityError(ErrorCatalogInvalid, "quality rule is nil", nil)
	}
	id := strings.TrimSpace(rule.ID())
	version := strings.TrimSpace(rule.Version())
	assessment := strings.TrimSpace(rule.AssessmentKind())
	if !validNamespacedID(id) || !validVersion(version) {
		return newQualityError(ErrorCatalogInvalid, "quality rule requires a safe ID and version", map[string]any{"id": id, "version": version})
	}
	if assessment != AssessmentExact && assessment != AssessmentSignal {
		return newQualityError(ErrorCatalogInvalid, "quality rule assessment kind is invalid", map[string]any{"rule_id": id, "assessment_kind": assessment})
	}
	capabilities := normalizeStrings(rule.RequiredCapabilities())
	for _, capability := range capabilities {
		if !validNamespacedID(capability) {
			return newQualityError(ErrorCatalogInvalid, "quality rule required capability is invalid", map[string]any{"rule_id": id, "capability": capability})
		}
	}
	descriptor := descriptorForRule(rule)
	if descriptor.ID != id || descriptor.Version != version || descriptor.AssessmentKind != assessment {
		return newQualityError(ErrorCatalogInvalid, "quality rule descriptor does not match strategy identity", map[string]any{"rule_id": id, "version": version})
	}
	if descriptor.DefaultSeverity == "" {
		descriptor.DefaultSeverity = defaultSeverity(assessment)
	}
	if !validSeverity(descriptor.DefaultSeverity) {
		return newQualityError(ErrorCatalogInvalid, "quality rule default severity is invalid", map[string]any{"rule_id": id, "severity": descriptor.DefaultSeverity})
	}
	key := catalogKey(id, version)
	catalog.mu.Lock()
	defer catalog.mu.Unlock()
	if existing, ok := catalog.qualityRules[key]; ok {
		if sameStrategy(existing, rule) {
			return nil
		}
		return newQualityError(ErrorCatalogConflict, "a different quality rule already uses this ID and version", map[string]any{"id": id, "version": version})
	}
	catalog.qualityRules[key] = rule
	return nil
}

func (catalog *Catalog) ResolveMetricProvider(id, version string) (MetricProvider, bool) {
	if catalog == nil {
		return nil, false
	}
	catalog.mu.RLock()
	defer catalog.mu.RUnlock()
	provider, ok := catalog.metricProviders[catalogKey(strings.TrimSpace(id), strings.TrimSpace(version))]
	return provider, ok
}

func (catalog *Catalog) ResolveQualityRule(id, version string) (QualityRule, bool) {
	if catalog == nil {
		return nil, false
	}
	catalog.mu.RLock()
	defer catalog.mu.RUnlock()
	rule, ok := catalog.qualityRules[catalogKey(strings.TrimSpace(id), strings.TrimSpace(version))]
	return rule, ok
}

func (catalog *Catalog) hasRuleID(id string) bool {
	if catalog == nil {
		return false
	}
	catalog.mu.RLock()
	defer catalog.mu.RUnlock()
	for _, rule := range catalog.qualityRules {
		if rule.ID() == id {
			return true
		}
	}
	return false
}

func (catalog *Catalog) ListMetricProviders() []MetricProvider {
	if catalog == nil {
		return nil
	}
	catalog.mu.RLock()
	defer catalog.mu.RUnlock()
	result := make([]MetricProvider, 0, len(catalog.metricProviders))
	for _, provider := range catalog.metricProviders {
		result = append(result, provider)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ID() == result[j].ID() {
			return result[i].Version() < result[j].Version()
		}
		return result[i].ID() < result[j].ID()
	})
	return result
}

func (catalog *Catalog) ListQualityRules() []RuleDescriptor {
	if catalog == nil {
		return nil
	}
	catalog.mu.RLock()
	defer catalog.mu.RUnlock()
	result := make([]RuleDescriptor, 0, len(catalog.qualityRules))
	for _, rule := range catalog.qualityRules {
		result = append(result, descriptorForRule(rule))
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ID == result[j].ID {
			return result[i].Version < result[j].Version
		}
		return result[i].ID < result[j].ID
	})
	return result
}

// DefaultRuleBinding returns a disabled binding with the rule's default
// parameters and severity. It is deliberately a catalog operation so callers
// do not need to switch on rule IDs when presenting or evaluating all rules.
func (catalog *Catalog) DefaultRuleBinding(id, version string) (RuleBinding, bool) {
	rule, ok := catalog.ResolveQualityRule(id, version)
	if !ok {
		return RuleBinding{}, false
	}
	descriptor := descriptorForRule(rule)
	parameters := TypedConfigBlock{
		Namespace:     descriptor.ParameterSchema.Namespace,
		SchemaVersion: descriptor.ParameterSchema.SchemaVersion,
		Payload:       map[string]any{},
	}
	if provider, hasDefaults := rule.(DefaultParameterProvider); hasDefaults {
		parameters = provider.DefaultParameters()
	}
	return RuleBinding{
		RuleID:      descriptor.ID,
		RuleVersion: descriptor.Version,
		Enabled:     false,
		Parameters:  parameters,
		Severity:    descriptor.DefaultSeverity,
	}, true
}

func (catalog *Catalog) ListQualityCapabilities() []CapabilityDescriptor {
	if catalog == nil {
		return nil
	}
	seen := map[string]CapabilityDescriptor{}
	for _, provider := range catalog.ListMetricProviders() {
		for _, capability := range provider.Capabilities() {
			key := capability.ID + "\x00" + capability.Version
			seen[key] = cloneCapabilityDescriptor(capability)
		}
	}
	result := make([]CapabilityDescriptor, 0, len(seen))
	for _, capability := range seen {
		result = append(result, capability)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ID == result[j].ID {
			return result[i].Version < result[j].Version
		}
		return result[i].ID < result[j].ID
	})
	return result
}

func descriptorForRule(rule QualityRule) RuleDescriptor {
	if provider, ok := rule.(RuleDescriptorProvider); ok {
		descriptor := provider.Descriptor()
		descriptor.RequiredCapabilities = normalizeStrings(descriptor.RequiredCapabilities)
		descriptor.Limitations = normalizeStrings(descriptor.Limitations)
		if descriptor.ParameterSchema.Fields == nil {
			descriptor.ParameterSchema.Fields = map[string]ParameterField{}
		}
		return descriptor
	}
	schema := ParameterSchema{Fields: map[string]ParameterField{}, AllowAdditional: true}
	if provider, ok := rule.(ParameterSchemaProvider); ok {
		schema = provider.ParameterSchema()
	}
	return RuleDescriptor{
		ID:                   rule.ID(),
		Version:              rule.Version(),
		AssessmentKind:       rule.AssessmentKind(),
		RequiredCapabilities: normalizeStrings(rule.RequiredCapabilities()),
		ParameterSchema:      schema,
		DefaultSeverity:      defaultSeverity(rule.AssessmentKind()),
	}
}

func catalogKey(id, version string) string { return id + "\x00" + version }

func isNilStrategy(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}

func sameStrategy(left, right any) bool {
	leftValue := reflect.ValueOf(left)
	rightValue := reflect.ValueOf(right)
	if !leftValue.IsValid() || !rightValue.IsValid() || leftValue.Type() != rightValue.Type() {
		return false
	}
	if leftValue.Type().Comparable() {
		return leftValue.Interface() == rightValue.Interface()
	}
	if reflect.DeepEqual(left, right) {
		return true
	}
	switch leftValue.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Func, reflect.Slice:
		return leftValue.Pointer() == rightValue.Pointer()
	default:
		return false
	}
}

func cloneCapabilityDescriptor(value CapabilityDescriptor) CapabilityDescriptor {
	value.SupportedLanguages = append([]string(nil), value.SupportedLanguages...)
	return value
}

func normalizeStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
