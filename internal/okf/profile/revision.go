package profile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

// Revision identifies registered rule identities and validated shape definitions.
// It does not call provider metadata methods: a broken description must not make
// otherwise usable profiles unavailable. Full metadata has its own catalog hash.
func (registry *Registry) Revision() (string, error) {
	registry.mu.RLock()
	rules := make([]string, 0, len(registry.strategies))
	for key := range registry.strategies {
		rules = append(rules, key)
	}
	registry.mu.RUnlock()
	sort.Strings(rules)
	data, err := json.Marshal(struct {
		Rules       []string                 `json:"rules"`
		Shapes      []domain.ShapeDefinition `json:"shapes"`
		Details     []ports.Extension        `json:"details"`
		Diagnostics []ports.Extension        `json:"diagnostics"`
	}{rules, registry.ShapeCatalog(), registry.DetailRendererCatalog(), registry.DiagnosticProviderCatalog()})
	if err != nil {
		return "", domain.WrapError("okf_extension_catalog_failed", 500, "registered definitions cannot be encoded", err)
	}
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:]), nil
}
