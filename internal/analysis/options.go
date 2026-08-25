package analysis

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

func ResolveOptions(manifest Manifest, projectOptions, cliOptions map[string]any) (EffectiveOptions, error) {
	values := make(map[string]any, len(manifest.Options))
	sources := make(map[string]string, len(manifest.Options))
	descriptors := make(map[string]OptionDescriptor, len(manifest.Options))
	for _, descriptor := range manifest.Options {
		descriptors[descriptor.Name] = descriptor
		value, err := normalizeOptionValue(descriptor, descriptor.Default)
		if err != nil {
			return EffectiveOptions{}, WrapHostError(ErrInvalidManifest, "analyzer default option is invalid", err, map[string]any{"option": descriptor.Name})
		}
		values[descriptor.Name] = value
		sources[descriptor.Name] = "default"
	}
	if err := applyOptionLayer(values, sources, descriptors, projectOptions, "project"); err != nil {
		return EffectiveOptions{}, err
	}
	if err := applyOptionLayer(values, sources, descriptors, cliOptions, "cli"); err != nil {
		return EffectiveOptions{}, err
	}
	fingerprintBytes, err := json.Marshal(values)
	if err != nil {
		return EffectiveOptions{}, WrapHostError(ErrInvalidOptions, "effective options could not be fingerprinted", err, nil)
	}
	fingerprint := sha256.Sum256(fingerprintBytes)
	return EffectiveOptions{
		Values:      values,
		Sources:     sources,
		Fingerprint: hex.EncodeToString(fingerprint[:]),
	}, nil
}

func applyOptionLayer(values map[string]any, sources map[string]string, descriptors map[string]OptionDescriptor, layer map[string]any, source string) error {
	for name, value := range layer {
		descriptor, ok := descriptors[name]
		if !ok {
			return NewHostError(ErrInvalidOptions, "option is not supported by the selected analyzer", map[string]any{"option": name})
		}
		normalized, err := normalizeOptionValue(descriptor, value)
		if err != nil {
			return WrapHostError(ErrInvalidOptions, "option value is invalid", err, map[string]any{"option": name, "source": source})
		}
		values[name] = normalized
		sources[name] = source
	}
	return nil
}

func normalizeOptionValue(descriptor OptionDescriptor, value any) (any, error) {
	switch descriptor.Type {
	case "boolean":
		if value == nil {
			return nil, fmt.Errorf("option %s cannot be null", descriptor.Name)
		}
		typed, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("option %s expects boolean, got %T", descriptor.Name, value)
		}
		return typed, nil
	case "string":
		if value == nil {
			return nil, nil
		}
		typed, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("option %s expects string, got %T", descriptor.Name, value)
		}
		if len(descriptor.AllowedValues) > 0 && !contains(descriptor.AllowedValues, typed) {
			return nil, fmt.Errorf("option %s does not allow %q", descriptor.Name, typed)
		}
		return typed, nil
	case "string[]":
		typed, err := stringSlice(value)
		if err != nil {
			return nil, fmt.Errorf("option %s: %w", descriptor.Name, err)
		}
		sort.Strings(typed)
		return typed, nil
	default:
		return nil, fmt.Errorf("option %s declares unsupported type %q", descriptor.Name, descriptor.Type)
	}
}

func stringSlice(value any) ([]string, error) {
	if value == nil {
		return []string{}, nil
	}
	switch typed := value.(type) {
	case []string:
		result := make([]string, len(typed))
		copy(result, typed)
		return result, nil
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			stringValue, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("expects an array of strings, got item %T", item)
			}
			result = append(result, stringValue)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("expects an array of strings, got %T", value)
	}
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
