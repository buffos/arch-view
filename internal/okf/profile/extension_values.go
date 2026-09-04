package profile

import (
	"bytes"
	"encoding/json"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

// Capture JSON-compatible host values into owned containers. UseNumber retains
// exact integers; ordinary float64 decoding can round extension metadata.
func captureExtensionValue[T any](value T) (T, error) {
	var captured T
	data, err := json.Marshal(value)
	if err != nil {
		return captured, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	err = decoder.Decode(&captured)
	return captured, err
}

func cloneExtensionMetadata(value ports.Extension) ports.Extension {
	value.Capabilities = append([]string(nil), value.Capabilities...)
	value.ParameterSchema = domain.CloneMap(value.ParameterSchema)
	value.DefinitionSchema = domain.CloneMap(value.DefinitionSchema)
	return value
}
