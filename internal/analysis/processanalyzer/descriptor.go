package processanalyzer

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/processprotocol"
)

func wrapDescriptorError(err error) error {
	code := analysis.ErrorCodeOf(err)
	if code == analysis.ErrHostFailure {
		code = analysis.ErrInvalidManifest
	}
	return analysis.WrapHostError(code, "external analyzer descriptor is invalid", err, nil)
}

func validateDescriptorStrings(descriptor processprotocol.Descriptor) error {
	if strings.TrimSpace(descriptor.Command) != descriptor.Command {
		return analysis.NewHostError(analysis.ErrInvalidManifest, "external analyzer command cannot contain surrounding whitespace", nil)
	}
	if hasUnsafeProcessText(descriptor.Command) {
		return analysis.NewHostError(analysis.ErrInvalidManifest, "external analyzer command contains unsafe control characters", nil)
	}
	for _, arg := range descriptor.Args {
		if hasUnsafeProcessText(arg) {
			return analysis.NewHostError(analysis.ErrInvalidManifest, "external analyzer argument contains unsafe control characters", nil)
		}
	}
	if descriptor.WorkingDirectory != "" {
		if strings.TrimSpace(descriptor.WorkingDirectory) != descriptor.WorkingDirectory || hasUnsafeProcessText(descriptor.WorkingDirectory) {
			return analysis.NewHostError(analysis.ErrInvalidManifest, "external analyzer working directory is unsafe", nil)
		}
	}
	return nil
}

func hasUnsafeProcessText(value string) bool {
	return strings.ContainsAny(value, "\x00\r\n")
}

func normalizeBaseDirectory(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", analysis.NewHostError(analysis.ErrInvalidManifest, "external analyzer base directory is required", nil)
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrInvalidManifest, "external analyzer base directory could not be normalized", err, map[string]any{"base_directory": value})
	}
	absolute = filepath.Clean(absolute)
	info, err := os.Stat(absolute)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrInvalidManifest, "external analyzer base directory could not be read", err, map[string]any{"base_directory": absolute})
	}
	if !info.IsDir() {
		return "", analysis.NewHostError(analysis.ErrInvalidManifest, "external analyzer base directory is not a directory", map[string]any{"base_directory": absolute})
	}
	return absolute, nil
}

func resolveCommand(value, baseDirectory string) (string, error) {
	if !isPathLike(value) {
		return value, nil
	}
	resolved := resolveRelative(value, baseDirectory)
	info, err := os.Stat(resolved)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrInvalidManifest, "external analyzer command path could not be resolved", err, map[string]any{"command": value, "resolved": resolved})
	}
	if info.IsDir() {
		return "", analysis.NewHostError(analysis.ErrInvalidManifest, "external analyzer command path is a directory", map[string]any{"command": value, "resolved": resolved})
	}
	return resolved, nil
}

func resolveArguments(values []string, baseDirectory string) ([]string, error) {
	args := append([]string(nil), values...)
	for index, value := range args {
		if !isPathLike(value) {
			continue
		}
		resolved := resolveRelative(value, baseDirectory)
		info, err := os.Stat(resolved)
		if err == nil && !info.IsDir() {
			args[index] = resolved
		}
	}
	return args, nil
}

func resolveWorkingDirectory(value, baseDirectory string) (string, error) {
	resolved := baseDirectory
	if value != "" {
		resolved = resolveRelative(value, baseDirectory)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrInvalidManifest, "external analyzer working directory could not be resolved", err, map[string]any{"working_directory": value, "resolved": resolved})
	}
	if !info.IsDir() {
		return "", analysis.NewHostError(analysis.ErrInvalidManifest, "external analyzer working directory is not a directory", map[string]any{"working_directory": value, "resolved": resolved})
	}
	return resolved, nil
}

func resolveRelative(value, baseDirectory string) string {
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Clean(filepath.Join(baseDirectory, value))
}

func isPathLike(value string) bool {
	if value == "" || strings.HasPrefix(value, "-") || filepath.IsAbs(value) {
		return filepath.IsAbs(value)
	}
	return strings.ContainsAny(value, `/\\`) || strings.HasPrefix(value, ".") || filepath.Ext(value) != ""
}

func cloneDescriptor(value processprotocol.Descriptor) processprotocol.Descriptor {
	value.Manifest = cloneManifest(value.Manifest)
	value.Args = append([]string(nil), value.Args...)
	return value
}

func cloneManifest(value analysis.Manifest) analysis.Manifest {
	value.DetectionMarkers = append([]analysis.DetectionMarker(nil), value.DetectionMarkers...)
	value.Capabilities = append([]string(nil), value.Capabilities...)
	options := append([]analysis.OptionDescriptor(nil), value.Options...)
	value.Options = make([]analysis.OptionDescriptor, len(options))
	for index, option := range options {
		value.Options[index] = option
		value.Options[index].AllowedValues = append([]string(nil), option.AllowedValues...)
	}
	return value
}
