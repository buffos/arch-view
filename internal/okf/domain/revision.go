package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// ProfileRevision includes effective inherited settings when called after
// composition. Client-provided revisions and runtime status are not inputs.
func ProfileRevision(value Profile) string {
	value.Revision, value.Status = "", ""
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:])
}
