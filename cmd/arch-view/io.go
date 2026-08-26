package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/buffo/arch-view/internal/analysis"
)

func writeJSON(writer io.Writer, value any) int {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return 4
	}
	return 0
}

func writeFileJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return analysis.WrapHostError(analysis.ErrHostFailure, "analysis result could not be serialized", err, nil)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return analysis.WrapHostError(analysis.ErrHostFailure, "analysis result could not be written", err, map[string]any{"output": path})
	}
	return nil
}

func writeError(writer io.Writer, err error) {
	data, marshalErr := analysis.MarshalError(err)
	if marshalErr != nil {
		_, _ = fmt.Fprintln(writer, err)
		return
	}
	_, _ = fmt.Fprintln(writer, string(data))
}
