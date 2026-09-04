package domain

import "encoding/json"

func cloneStrings(value []string) []string {
	if value == nil {
		return nil
	}
	return append([]string{}, value...)
}

// Features distinguishes inheritance (nil/omitted) from an explicit clear ([]).
// A pointer avoids encoding/json's omitempty collapsing these two intents.
func (value LayoutSettings) MarshalJSON() ([]byte, error) {
	type wire struct {
		Algorithm string         `json:"algorithm,omitempty"`
		Options   map[string]any `json:"options,omitempty"`
		Features  *[]string      `json:"features,omitempty"`
	}
	out := wire{Algorithm: value.Algorithm, Options: value.Options}
	if value.Features != nil {
		out.Features = &value.Features
	}
	return json.Marshal(out)
}
