package projectdocument

import (
	"bytes"
	"encoding/json"
)

// RequireV2 upgrades a document without changing section ownership. The
// established v2 contract requires an analysis object, even for empty policy.
func RequireV2(raw map[string]json.RawMessage) {
	raw["schema_version"] = json.RawMessage(`"arch-view.config/v2"`)
	if len(raw["analysis"]) == 0 || bytes.Equal(bytes.TrimSpace(raw["analysis"]), []byte("null")) {
		raw["analysis"] = json.RawMessage(`{}`)
	}
}
