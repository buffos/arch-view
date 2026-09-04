package profile

import (
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

// RegisterShape is a host composition operation, before serving sessions.
func (registry *Registry) RegisterShape(provider ports.ShapeProvider) error {
	return registry.shapes.Register(provider)
}

func (registry *Registry) ResolveShape(reference string) (domain.ShapeDefinition, bool) {
	id, version, present := strings.Cut(reference, "@")
	if !present {
		version = "1"
	}
	if !strings.Contains(id, ".") {
		id = "okf.shape." + id
	}
	return registry.shapes.Resolve(id, version)
}

func (registry *Registry) ShapeCatalog() []domain.ShapeDefinition {
	return registry.shapes.Catalog()
}

func (registry *Registry) validateShapeReferences(value domain.Profile) []domain.Diagnostic {
	keys := make([]string, 0, len(value.Style.Tokens))
	for key := range value.Style.Tokens {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var diagnostics []domain.Diagnostic
	for _, key := range keys {
		reference := value.Style.Tokens[key].Shape
		if reference == "" {
			continue
		}
		if _, exists := registry.ResolveShape(reference); !exists {
			diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_shape_unsupported", Severity: "error", Category: "presentation", ProfileID: value.ProfileID,
				Message: "A style token references an unregistered shape.", Details: map[string]any{"token": key, "shape": reference}, Recovery: "Select a registered shape and version before saving the profile."})
		}
	}
	return diagnostics
}
