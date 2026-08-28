package processprotocol

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

// DecodeDescriptor decodes one strict JSON descriptor object and validates its
// schema version, manifest, and argv command fields.
func DecodeDescriptor(data []byte) (Descriptor, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var descriptor Descriptor
	if err := decoder.Decode(&descriptor); err != nil {
		return Descriptor{}, newProtocolError(ErrorDescriptorInvalid, "", "", "descriptor is not valid JSON", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Descriptor{}, newProtocolError(ErrorDescriptorInvalid, "", "", "descriptor must contain one JSON object", ErrDescriptorInvalid)
		}
		return Descriptor{}, newProtocolError(ErrorDescriptorInvalid, "", "", "descriptor has trailing JSON", err)
	}
	if err := ValidateDescriptor(descriptor); err != nil {
		return Descriptor{}, err
	}
	return descriptor, nil
}

func ReadDescriptor(reader io.Reader) (Descriptor, error) {
	if reader == nil {
		return Descriptor{}, newProtocolError(ErrorDescriptorInvalid, "", "", "descriptor reader is nil", ErrDescriptorInvalid)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return Descriptor{}, newProtocolError(ErrorDescriptorInvalid, "", "", "descriptor could not be read", err)
	}
	return DecodeDescriptor(data)
}

func ValidateDescriptor(descriptor Descriptor) error {
	if descriptor.SchemaVersion != DescriptorSchemaVersion {
		return newProtocolError(ErrorDescriptorInvalid, "", "", "descriptor schema version is not supported", ErrDescriptorInvalid)
	}
	if err := analysis.ValidateManifest(descriptor.Manifest); err != nil {
		return newProtocolError(ErrorDescriptorInvalid, "", "", "descriptor manifest is invalid", err)
	}
	if strings.TrimSpace(descriptor.Command) == "" {
		return newProtocolError(ErrorDescriptorInvalid, "", "", "descriptor command is required", ErrDescriptorInvalid)
	}
	if descriptor.WorkingDirectory != "" && strings.TrimSpace(descriptor.WorkingDirectory) == "" {
		return newProtocolError(ErrorDescriptorInvalid, "", "", "descriptor working_directory cannot be blank", ErrDescriptorInvalid)
	}
	return nil
}
