package domain

import "encoding/json"

// Preserve explicit empty maps separately from omitted maps during persistence.
func (value StyleSettings) MarshalJSON() ([]byte, error) {
	var tokens *map[string]StyleToken
	var states *map[string]string
	var decorations *map[string]StyleDecoration
	if value.Tokens != nil {
		tokens = &value.Tokens
	}
	if value.StateTokens != nil {
		states = &value.StateTokens
	}
	if value.Decorations != nil {
		decorations = &value.Decorations
	}
	return json.Marshal(struct {
		DefaultToken string                      `json:"default_token,omitempty"`
		Tokens       *map[string]StyleToken      `json:"tokens,omitempty"`
		StateTokens  *map[string]string          `json:"state_tokens,omitempty"`
		Decorations  *map[string]StyleDecoration `json:"decorations,omitempty"`
	}{value.DefaultToken, tokens, states, decorations})
}
