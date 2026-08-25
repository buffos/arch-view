package analysis

import (
	"reflect"
	"sort"
	"strings"
	"sync"
)

type Registry struct {
	mu        sync.RWMutex
	analyzers map[string]Analyzer
}

func NewRegistry() *Registry {
	return &Registry{analyzers: make(map[string]Analyzer)}
}

func (r *Registry) Register(analyzer Analyzer) error {
	if isNilAnalyzer(analyzer) {
		return NewHostError(ErrInvalidManifest, "cannot register a nil analyzer", nil)
	}
	manifest := analyzer.Manifest()
	if err := ValidateManifest(manifest); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.analyzers[manifest.ID]; exists {
		return NewHostError(ErrDuplicateAnalyzer, "analyzer id is already registered", map[string]any{"id": manifest.ID})
	}
	r.analyzers[manifest.ID] = analyzer
	return nil
}

func isNilAnalyzer(analyzer Analyzer) bool {
	if analyzer == nil {
		return true
	}
	value := reflect.ValueOf(analyzer)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func (r *Registry) Get(id string) (Analyzer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	analyzer, ok := r.analyzers[id]
	return analyzer, ok
}

func (r *Registry) ByLanguage(language string) []Analyzer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	language = strings.ToLower(language)
	result := make([]Analyzer, 0)
	for _, analyzer := range r.analyzers {
		if analyzer.Manifest().Language == language {
			result = append(result, analyzer)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Manifest().ID < result[j].Manifest().ID
	})
	return result
}

func (r *Registry) List() []Analyzer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Analyzer, 0, len(r.analyzers))
	for _, analyzer := range r.analyzers {
		result = append(result, analyzer)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Manifest().ID < result[j].Manifest().ID
	})
	return result
}

func (r *Registry) ListManifests() []Manifest {
	analyzers := r.List()
	result := make([]Manifest, 0, len(analyzers))
	for _, analyzer := range analyzers {
		result = append(result, analyzer.Manifest())
	}
	return result
}
