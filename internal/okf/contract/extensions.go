package contract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/buffo/arch-view/internal/okf/domain"
)

// ExtensionCatalogRevision identifies the ordered catalog returned by the
// application, including schemas. JSON object key order does not affect it.
func ExtensionCatalogRevision(extensions []any) (string, error) {
	data, err := json.Marshal(extensions)
	if err != nil {
		return "", domain.WrapError("okf_extension_catalog_failed", 500, "extension catalog metadata cannot be encoded", err)
	}
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:]), nil
}
