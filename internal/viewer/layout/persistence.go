package layout

import (
	"encoding/json"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/projectdocument"
)

func saveLayoutDocument(path string, profile LayoutProfile, initial []byte) ([]byte, error) {
	data, err := projectdocument.Update(path, initial, func(current []byte) ([]byte, error) {
		raw, err := projectdocument.Object(current)
		if err != nil {
			return nil, err
		}
		encoded, err := json.Marshal(profile)
		if err != nil {
			return nil, err
		}
		raw["layout"] = encoded
		updated, err := json.MarshalIndent(raw, "", "  ")
		return append(updated, '\n'), err
	})
	if err != nil {
		return nil, analysis.WrapHostError(analysis.ErrHostFailure, "layout configuration could not be updated atomically", err, map[string]any{"path": path})
	}
	return data, nil
}
