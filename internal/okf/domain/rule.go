package domain

import (
	"bytes"
	"encoding/json"
)

// UnmarshalJSON treats an omitted enabled flag as enabled. This keeps the
// declarative profile format concise while still preserving an explicit false
// value for disabled rules.
func (rule *RuleInvocation) UnmarshalJSON(data []byte) error {
	type rawRule struct {
		RuleID     string         `json:"rule_id"`
		Version    string         `json:"version,omitempty"`
		Priority   int            `json:"priority"`
		Enabled    *bool          `json:"enabled"`
		Parameters map[string]any `json:"parameters,omitempty"`
	}
	var value rawRule
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	enabled := true
	if value.Enabled != nil {
		enabled = *value.Enabled
	}
	*rule = RuleInvocation{RuleID: value.RuleID, Version: value.Version, Priority: value.Priority, Enabled: enabled, Parameters: value.Parameters}
	return nil
}
