package presentation

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

// ShapeRegistry snapshots provider definitions at registration. Callers cannot
// mutate active geometry and provider callbacks never run under its lock.
type ShapeRegistry struct {
	mu          sync.RWMutex
	definitions map[string]domain.ShapeDefinition
}

// Unit-box dimensions below float64 precision at 1 are not usable extents.
// This also prevents dividing text dimensions by subnormal fractions.
const minimumContentExtent = 0x1p-52

func NewShapeRegistry() *ShapeRegistry {
	return &ShapeRegistry{definitions: make(map[string]domain.ShapeDefinition)}
}

func (registry *ShapeRegistry) Register(provider ports.ShapeProvider) (err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("shape provider failed: %v", failure)
		}
	}()
	if provider == nil {
		return fmt.Errorf("shape provider is required")
	}
	value := provider.Definition()
	if err := validateShape(value); err != nil {
		return err
	}
	value.Points = append([]domain.ShapePoint(nil), value.Points...)
	registry.mu.Lock()
	defer registry.mu.Unlock()
	key := value.ID + "@" + value.Version
	if _, exists := registry.definitions[key]; exists {
		return fmt.Errorf("shape %q is already registered", key)
	}
	registry.definitions[key] = value
	return nil
}

func (registry *ShapeRegistry) Resolve(id, version string) (domain.ShapeDefinition, bool) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	value, ok := registry.definitions[id+"@"+version]
	value.Points = append([]domain.ShapePoint(nil), value.Points...)
	return value, ok
}

func (registry *ShapeRegistry) Catalog() []domain.ShapeDefinition {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	values := make([]domain.ShapeDefinition, 0, len(registry.definitions))
	for _, value := range registry.definitions {
		value.Points = append([]domain.ShapePoint(nil), value.Points...)
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].ID != values[j].ID {
			return values[i].ID < values[j].ID
		}
		return values[i].Version < values[j].Version
	})
	return values
}

func validateShape(value domain.ShapeDefinition) error {
	if !domain.ValidExtensionIdentity(value.ID, value.Version) || strings.TrimSpace(value.Description) == "" {
		return fmt.Errorf("shape requires namespaced ID, version, and description")
	}
	unit := func(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && v <= 1 }
	box := value.Content
	if !unit(box.X) || !unit(box.Y) || !unit(box.Width) || !unit(box.Height) || box.Width < minimumContentExtent || box.Height < minimumContentExtent || box.X+box.Width > 1 || box.Y+box.Height > 1 {
		return fmt.Errorf("shape content must be inside unit coordinates with extents at least 2^-52")
	}
	if !unit(value.CornerRadius) || value.CornerRadius > 0.5 {
		return fmt.Errorf("invalid shape corner radius")
	}
	validateGeometry, exists := geometryValidators[value.Geometry]
	if !exists {
		return fmt.Errorf("unsupported shape geometry %q", value.Geometry)
	}
	if value.Geometry == "polygon" && (len(value.Points) < 3 || len(value.Points) > 64) {
		return fmt.Errorf("polygon requires 3 to 64 points")
	}
	for _, point := range value.Points {
		if !unit(point.X) || !unit(point.Y) {
			return fmt.Errorf("shape points must use unit coordinates")
		}
	}
	return validateGeometry(value)
}
