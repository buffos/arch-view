package layout

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

// NewSession discovers the nearest profile and keeps its source root for
// subsequent Save and Save As operations.
func NewSession(sourceRoot string) Session {
	if sourceRoot != "" {
		if absolute, err := filepath.Abs(sourceRoot); err == nil {
			sourceRoot = filepath.Clean(absolute)
		}
	}
	session := discoverLayoutSession(sourceRoot)
	session.sourceRoot = sourceRoot
	return session
}

// DefaultProfile returns a fresh copy of the built-in layout profile.
func DefaultProfile() LayoutProfile {
	return defaultLayoutProfile()
}

// ValidateProfile checks and canonicalizes a profile without mutating the
// caller's map.
func ValidateProfile(profile LayoutProfile) (LayoutProfile, error) {
	return validateLayoutProfile(profile)
}

// EncodeConfig serializes a validated profile using the stable config schema.
func EncodeConfig(profile LayoutProfile) ([]byte, error) {
	return encodeLayoutConfig(profile)
}

// DecodeConfig parses and validates one .archview.json document.
func DecodeConfig(data []byte) (LayoutProfile, error) {
	return decodeLayoutConfig(data)
}

// Response returns the session state in the HTTP-neutral response schema.
func (session Session) Response() LayoutConfigResponse {
	return session.response()
}

// Apply replaces the current in-memory profile and keeps persistence metadata
// unchanged.
func (session *Session) Apply(profile LayoutProfile) error {
	profile, err := validateLayoutProfile(profile)
	if err != nil {
		return err
	}
	session.profile = profile
	session.status = "valid"
	session.origin = "session"
	session.diagnostics = sessionLayoutDiagnostics(session.sourceRoot)
	session.canSave = session.activePath != "" && session.activeOrigin != "" && session.status == "valid"
	return nil
}

// Reset restores built-in defaults for the current session.
func (session *Session) Reset() {
	session.profile = defaultLayoutProfile()
	session.status = "valid"
	session.origin = "session"
	session.diagnostics = sessionLayoutDiagnostics(session.sourceRoot)
	session.canSave = session.activePath != "" && session.activeOrigin != "" && session.status == "valid"
}

// SaveActive writes the profile to the discovered configuration path. It
// never creates a new file; callers must use SaveAs for that.
func (session *Session) SaveActive(profile LayoutProfile) error {
	profile, err := validateLayoutProfile(profile)
	if err != nil {
		return err
	}
	activePath := session.activePath
	if activePath == "" || !session.canSave {
		if activePath == "" {
			return analysis.NewHostError(analysis.ErrSaveAsRequired, "there is no active .archview.json file; use Save As to choose a destination", nil)
		}
		return analysis.NewHostError(analysis.ErrInvalidOptions, "the nearest .archview.json is invalid; use Save As after correcting the profile", map[string]any{"path": activePath})
	}
	data, err := saveLayoutDocument(activePath, profile, nil)
	if err != nil {
		return err
	}
	session.profile = profile
	session.origin = session.activeOrigin
	session.status = "valid"
	session.canSave = true
	session.diagnostics = sessionLayoutDiagnostics(session.sourceRoot)
	session.rawDocument = append(session.rawDocument[:0], data...)
	return nil
}

// SaveAs writes a profile to an explicitly selected existing directory and
// makes that file the active persistence target.
func (session *Session) SaveAs(profile LayoutProfile, destinationDir string, confirm bool) error {
	if !confirm {
		return analysis.NewHostError(analysis.ErrInvalidOptions, "Save As requires explicit confirmation", nil)
	}
	directory, err := normalizeLayoutDestination(destinationDir)
	if err != nil {
		return err
	}
	profile, err = validateLayoutProfile(profile)
	if err != nil {
		return err
	}
	configPath := filepath.Join(directory, layoutConfigFileName)
	data, err := encodeLayoutConfigPreserving(profile, session.analysisRaw, session.rawDocument)
	if err != nil {
		return err
	}
	if !session.canSaveAs || session.sourceRoot == "" {
		return analysis.NewHostError(analysis.ErrPersistenceUnavailable, "Save As is unavailable for a model-only session", nil)
	}
	origin := activeLayoutOrigin(session.sourceRoot, configPath)
	data, err = saveLayoutDocument(configPath, profile, data)
	if err != nil {
		return err
	}
	session.profile = profile
	session.activePath = configPath
	session.activeOrigin = origin
	session.origin = origin
	session.status = "valid"
	session.canSave = true
	session.canSaveAs = true
	session.diagnostics = nil
	session.rawDocument = append(session.rawDocument[:0], data...)
	return nil
}

func normalizeLayoutDestination(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "Save As destination directory is required", nil)
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrInvalidRequest, "Save As destination directory could not be normalized", err, map[string]any{"directory": value})
	}
	absolute = filepath.Clean(absolute)
	info, err := os.Stat(absolute)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrUnreadableProject, "Save As destination directory could not be read", err, map[string]any{"directory": absolute})
	}
	if !info.IsDir() {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "Save As destination must be a directory", map[string]any{"directory": absolute})
	}
	return absolute, nil
}

func sessionLayoutDiagnostics(sourceRoot string) []LayoutDiagnostic {
	if sourceRoot == "" {
		return []LayoutDiagnostic{{Code: "persistence_unavailable", Severity: "info", Message: "This model-only session can apply layout settings for the current session, but has no project directory for persistence."}}
	}
	return nil
}
