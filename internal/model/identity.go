package model

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// StableID returns the deterministic identity format used by model-owned
// graph and projection records.
func StableID(kind string, parts ...string) string {
	payload := kind + "\x00" + strings.Join(parts, "\x00")
	sum := sha256.Sum256([]byte(payload))
	return "model:" + kind + ":" + hex.EncodeToString(sum[:8])
}
