// Package sourceindex owns the normalized source-facts attachment, extractor
// registry, and structural read model shared by analyzers and viewers.
package sourceindex

import (
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/syntax"
)

// SourceFactExtractor is the language-owned strategy boundary. Core assembly
// never switches on a language name; it resolves strategies by their declared
// capabilities and supported languages.
type SourceFactExtractor interface {
	ID() string
	Version() string
	Capabilities() []analysis.CapabilityDescriptor
	Extract(SourceFactInput) (FactBatch, error)
}

// Registry stores immutable extractor strategies by explicit ID and version.
// Registration is safe for concurrent readers after construction.
type Registry struct {
	mu         sync.RWMutex
	extractors map[string]SourceFactExtractor
}

func NewRegistry() *Registry {
	return &Registry{extractors: make(map[string]SourceFactExtractor)}
}

// Register adds an extractor. Re-registering the same implementation for the
// same ID/version is idempotent; a different implementation is rejected.
func (r *Registry) Register(extractor SourceFactExtractor) error {
	if r == nil {
		return analysis.NewHostError(analysis.ErrSourceExtractorFailed, "source-fact extractor registry is nil", nil)
	}
	if isNilExtractor(extractor) {
		return analysis.NewHostError(analysis.ErrSourceExtractorFailed, "source-fact extractor is nil", nil)
	}
	id := strings.TrimSpace(extractor.ID())
	version := strings.TrimSpace(extractor.Version())
	if id == "" || version == "" || strings.ContainsAny(id+version, "\x00\r\n") {
		return analysis.NewHostError(analysis.ErrSourceExtractorFailed, "source-fact extractor requires a safe ID and version", map[string]any{"id": id, "version": version})
	}
	capabilities := extractor.Capabilities()
	if err := validateCapabilities(capabilities); err != nil {
		return err
	}
	key := extractorKey(id, version)
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.extractors[key]; ok {
		if sameExtractor(existing, extractor) {
			return nil
		}
		return analysis.NewHostError(analysis.ErrSourceExtractorFailed, "a different source-fact extractor already uses this ID and version", map[string]any{"id": id, "version": version})
	}
	r.extractors[key] = extractor
	return nil
}

func (r *Registry) List() []SourceFactExtractor {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]SourceFactExtractor, 0, len(r.extractors))
	for _, extractor := range r.extractors {
		result = append(result, extractor)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ID() == result[j].ID() {
			return result[i].Version() < result[j].Version()
		}
		return result[i].ID() < result[j].ID()
	})
	return result
}

// Resolve returns the unique registered extractor that advertises capability
// for language. An absent match is an explicit unsupported capability.
func (r *Registry) Resolve(language, capability string) (SourceFactExtractor, error) {
	language = strings.ToLower(strings.TrimSpace(language))
	capability = strings.TrimSpace(capability)
	if r == nil || language == "" || capability == "" {
		return nil, analysis.NewHostError(analysis.ErrSourceExtractorUnavailable, "source-fact extractor capability is unavailable", map[string]any{"language": language, "capability": capability})
	}
	matches := make([]SourceFactExtractor, 0)
	for _, extractor := range r.List() {
		for _, descriptor := range extractor.Capabilities() {
			if descriptor.ID != capability || !languageSupported(descriptor, language) {
				continue
			}
			matches = append(matches, extractor)
			break
		}
	}
	if len(matches) == 0 {
		return nil, analysis.NewHostError(analysis.ErrSourceExtractorUnavailable, "no source-fact extractor advertises the requested capability", map[string]any{"language": language, "capability": capability})
	}
	if len(matches) > 1 {
		ids := make([]string, 0, len(matches))
		for _, extractor := range matches {
			ids = append(ids, extractor.ID()+"@"+extractor.Version())
		}
		return nil, analysis.NewHostError(analysis.ErrSourceExtractorFailed, "multiple source-fact extractors advertise the same language capability", map[string]any{"language": language, "capability": capability, "extractors": ids})
	}
	return matches[0], nil
}

func (r *Registry) ListCapabilities() []analysis.CapabilityDescriptor {
	if r == nil {
		return nil
	}
	seen := make(map[string]analysis.CapabilityDescriptor)
	for _, extractor := range r.List() {
		for _, descriptor := range extractor.Capabilities() {
			key := descriptor.ID + "\x00" + descriptor.Version + "\x00" + strings.Join(descriptor.SupportedLanguages, ",")
			seen[key] = descriptor
		}
	}
	result := make([]analysis.CapabilityDescriptor, 0, len(seen))
	for _, descriptor := range seen {
		result = append(result, descriptor)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ID == result[j].ID {
			return result[i].Version < result[j].Version
		}
		return result[i].ID < result[j].ID
	})
	return result
}

func extractorKey(id, version string) string { return id + "\x00" + version }

func isNilExtractor(extractor SourceFactExtractor) bool {
	if extractor == nil {
		return true
	}
	value := reflect.ValueOf(extractor)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func sameExtractor(left, right SourceFactExtractor) bool {
	leftValue := reflect.ValueOf(left)
	rightValue := reflect.ValueOf(right)
	if leftValue.Type() != rightValue.Type() {
		return false
	}
	if leftValue.Type().Comparable() {
		return leftValue.Interface() == rightValue.Interface()
	}
	if leftValue.Kind() == reflect.Pointer || leftValue.Kind() == reflect.Map || leftValue.Kind() == reflect.Func || leftValue.Kind() == reflect.Slice {
		return leftValue.Pointer() == rightValue.Pointer()
	}
	return false
}

func validateCapabilities(values []analysis.CapabilityDescriptor) error {
	seen := make(map[string]struct{}, len(values))
	for index := range values {
		value := &values[index]
		value.ID = strings.TrimSpace(value.ID)
		value.Version = strings.TrimSpace(value.Version)
		if value.ID == "" || strings.ContainsAny(value.ID+value.Version, "\x00\r\n") {
			return analysis.NewHostError(analysis.ErrSourceExtractorFailed, "source-fact capability requires a safe ID", map[string]any{"id": value.ID})
		}
		if _, exists := seen[value.ID]; exists {
			return analysis.NewHostError(analysis.ErrSourceExtractorFailed, "source-fact extractor capabilities must be unique", map[string]any{"id": value.ID})
		}
		seen[value.ID] = struct{}{}
		for languageIndex := range value.SupportedLanguages {
			value.SupportedLanguages[languageIndex] = strings.ToLower(strings.TrimSpace(value.SupportedLanguages[languageIndex]))
			if value.SupportedLanguages[languageIndex] == "" {
				return analysis.NewHostError(analysis.ErrSourceExtractorFailed, "source-fact capability language cannot be empty", map[string]any{"id": value.ID})
			}
		}
		sort.Strings(value.SupportedLanguages)
	}
	return nil
}

func languageSupported(descriptor analysis.CapabilityDescriptor, language string) bool {
	if len(descriptor.SupportedLanguages) == 0 {
		return true
	}
	for _, supported := range descriptor.SupportedLanguages {
		if strings.EqualFold(supported, language) {
			return true
		}
	}
	return false
}

// SourceFileInput is the immutable input to the source-index core. ModuleID
// is an architecture-model reference and is never used to infer containment.
type SourceFileInput struct {
	Path               string
	Content            []byte
	Language           analysis.LanguageRef
	Roles              []string
	AnalysisStatus     string
	Provenance         analysis.FactProvenance
	ModuleID           string
	SourceReferenceIDs []string
}

// SourceFactInput is passed to an extractor while its syntax tree is owned by
// the caller. Extractors must not retain the tree or content after Extract.
type SourceFactInput struct {
	File                  analysis.FileRecord
	Content               []byte
	Tree                  syntax.Node
	Scope                 analysis.ScopeContext
	RequestedCapabilities []string
}

// FactBatch is provisional extractor output. The core assigns opaque IDs and
// resolves symbol/documentation references before publishing a snapshot.
type FactBatch struct {
	Symbols       []analysis.SymbolRecord
	Documentation []analysis.DocumentationRecord
	Occurrences   []analysis.SymbolOccurrence
	Relations     []analysis.CodeRelation
	Metrics       []analysis.MetricFact
	Coverage      []analysis.CoverageRecord
}

// BuildInput describes one source-index snapshot build.
type BuildInput struct {
	Scope                 analysis.ScopeContext
	Producer              analysis.ProducerContext
	Files                 []SourceFileInput
	RequestedCapabilities []string
	SyntaxProvider        syntax.Provider
	Extractors            *Registry
}
