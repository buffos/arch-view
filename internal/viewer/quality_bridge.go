package viewer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/live"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/quality"
	"github.com/buffo/arch-view/internal/quality/adapter"
)

const (
	qualityProfileDirectoryName           = "quality-profiles"
	qualityBaselineDirectoryName          = "quality-baselines"
	qualityProfilesSchemaVersion          = "arch-view.quality-profiles/v1"
	qualityRulesSchemaVersion             = "arch-view.quality-rules/v1"
	qualityEvaluationRequestSchemaVersion = "arch-view.quality-evaluation-request/v1"
	qualityEvaluationSchemaVersion        = "arch-view.quality-evaluation/v1"
	qualityProfileSaveSchemaVersion       = "arch-view.quality-profile-save/v1"
	qualityBaselineCreateSchemaVersion    = "arch-view.quality-baseline-create/v1"
	maxQualityProfileBytes                = 1 << 20
	maxQualityEvaluationRequestBytes      = 1 << 20
	maxQualityBaselineBytes               = 8 << 20
)

// qualityProfileDescriptor is the safe, human-facing part of a discovered
// profile. It deliberately contains only a project-relative filename; the
// server never sends the absolute profile path to the browser.
type qualityProfileDescriptor struct {
	Name             string `json:"name"`
	FileName         string `json:"file_name"`
	ProfileID        string `json:"profile_id,omitempty"`
	ProfileVersion   string `json:"profile_version,omitempty"`
	BaselineID       string `json:"baseline_id,omitempty"`
	BaselineRevision string `json:"baseline_revision,omitempty"`
	Status           string `json:"status"`
	Message          string `json:"message,omitempty"`
}

type qualityProfilesHTTPResponse struct {
	SchemaVersion string                     `json:"schema_version"`
	Status        string                     `json:"status"`
	Directory     string                     `json:"directory,omitempty"`
	Profiles      []qualityProfileDescriptor `json:"profiles"`
	Message       string                     `json:"message,omitempty"`
}

// qualityRuleCatalogEntry combines the stable catalog descriptor with the
// selected profile's state. Parameters are returned only as typed rule
// configuration blocks so the browser can submit a temporary toggle set
// without learning rule-specific IDs or schemas.
type qualityRuleCatalogEntry struct {
	quality.RuleDescriptor
	Enabled         bool                     `json:"enabled"`
	Configured      bool                     `json:"configured"`
	Parameters      quality.TypedConfigBlock `json:"parameters"`
	ParameterSource string                   `json:"parameter_source"`
	Severity        string                   `json:"severity"`
}

type qualityRulesHTTPResponse struct {
	SchemaVersion string                    `json:"schema_version"`
	Status        string                    `json:"status"`
	Profile       *qualityProfileDescriptor `json:"profile,omitempty"`
	Rules         []qualityRuleCatalogEntry `json:"rules"`
	Message       string                    `json:"message,omitempty"`
}

type qualityEvaluationRequest struct {
	SchemaVersion  string                `json:"schema_version"`
	ProfileID      string                `json:"profile_id"`
	ProfileVersion string                `json:"profile_version"`
	Scope          string                `json:"scope,omitempty"`
	RuleBindings   []quality.RuleBinding `json:"rule_bindings,omitempty"`
}

type qualityEvaluationHTTPResponse struct {
	SchemaVersion string                     `json:"schema_version"`
	Status        string                     `json:"status"`
	ScopeID       string                     `json:"scope_id"`
	Profile       qualityProfileDescriptor   `json:"profile"`
	Report        *quality.QualityEvaluation `json:"report,omitempty"`
	Message       string                     `json:"message,omitempty"`
}

type qualityProfileSaveRequest struct {
	SchemaVersion        string                `json:"schema_version"`
	ProfileID            string                `json:"profile_id"`
	ProfileVersion       string                `json:"profile_version"`
	SourceProfileID      string                `json:"source_profile_id,omitempty"`
	SourceProfileVersion string                `json:"source_profile_version,omitempty"`
	FileName             string                `json:"file_name,omitempty"`
	RuleBindings         []quality.RuleBinding `json:"rule_bindings"`
}

type qualityProfileSaveHTTPResponse struct {
	SchemaVersion string                   `json:"schema_version"`
	Status        string                   `json:"status"`
	Profile       qualityProfileDescriptor `json:"profile"`
	Message       string                   `json:"message,omitempty"`
}

type qualityBaselineCreateRequest struct {
	SchemaVersion   string   `json:"schema_version"`
	Scope           string   `json:"scope,omitempty"`
	ProfileID       string   `json:"profile_id"`
	ProfileVersion  string   `json:"profile_version"`
	BaselineID      string   `json:"baseline_id"`
	Revision        string   `json:"revision,omitempty"`
	FileName        string   `json:"file_name"`
	Reason          string   `json:"reason"`
	Owner           string   `json:"owner,omitempty"`
	FindingIDs      []string `json:"finding_ids,omitempty"`
	AllActive       bool     `json:"all_active,omitempty"`
	AttachToProfile bool     `json:"attach_to_profile"`
}

type qualityBaselineCreateHTTPResponse struct {
	SchemaVersion string                    `json:"schema_version"`
	Status        string                    `json:"status"`
	BaselineFile  string                    `json:"baseline_file"`
	BaselineID    string                    `json:"baseline_id"`
	Revision      string                    `json:"revision,omitempty"`
	EntryCount    int                       `json:"entry_count"`
	Attached      bool                      `json:"attached"`
	Profile       *qualityProfileDescriptor `json:"profile,omitempty"`
	Message       string                    `json:"message,omitempty"`
}

type loadedQualityProfile struct {
	descriptor qualityProfileDescriptor
	profile    quality.QualityProfile
}

type qualityProfileCatalog struct {
	response qualityProfilesHTTPResponse
	profiles map[string]loadedQualityProfile
}

func (s *Server) handleQualityProfiles(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	catalog, err := s.discoverQualityProfiles()
	if err != nil {
		writeHTTPError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(writer, http.StatusOK, catalog.response)
}

func (s *Server) handleQualityProfileSave(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPut {
		writeMethodNotAllowed(writer, http.MethodPut)
		return
	}
	if s.rejectLivePolicyWrite(writer, "save_quality_profile") {
		return
	}
	input, err := decodeQualityProfileSaveRequest(request, false)
	if err != nil {
		writeHTTPError(writer, http.StatusBadRequest, err)
		return
	}
	catalog, err := s.discoverQualityProfiles()
	if err != nil {
		writeHTTPError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	selected, ok := catalog.profiles[qualityProfileKey(input.ProfileID, input.ProfileVersion)]
	if !ok {
		writeHTTPError(writer, http.StatusUnprocessableEntity, analysis.NewHostError(analysis.ErrInvalidOptions, "the selected quality profile is unavailable or invalid", map[string]any{
			"profile_id":      input.ProfileID,
			"profile_version": input.ProfileVersion,
		}))
		return
	}
	profile := selected.profile
	profile.EnabledRules = cloneQualityRuleBindings(input.RuleBindings)
	validated, err := quality.ValidateQualityProfile(profile, quality.NewDefaultCatalog())
	if err != nil {
		writeQualityBridgeError(writer, err)
		return
	}
	directory, err := qualityProfileDirectory(s.getSourceRoot())
	if err != nil || directory == "" {
		if err == nil {
			err = analysis.NewHostError(analysis.ErrInvalidRequest, "quality profile directory is unavailable", nil)
		}
		writeHTTPError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	fileName := selected.descriptor.FileName
	if err := validateQualityProfileFileName(fileName); err != nil {
		writeHTTPError(writer, http.StatusForbidden, err)
		return
	}
	if err := writeQualityProfile(filepath.Join(directory, fileName), validated); err != nil {
		writeHTTPError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	selected.descriptor.Status = "available"
	writeJSON(writer, http.StatusOK, qualityProfileSaveHTTPResponse{SchemaVersion: qualityProfileSaveSchemaVersion, Status: "saved", Profile: selected.descriptor, Message: "Quality profile saved."})
}

func (s *Server) handleQualityProfileSaveAs(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPut {
		writeMethodNotAllowed(writer, http.MethodPut)
		return
	}
	if s.rejectLivePolicyWrite(writer, "save_quality_profile_as") {
		return
	}
	input, err := decodeQualityProfileSaveRequest(request, true)
	if err != nil {
		writeHTTPError(writer, http.StatusBadRequest, err)
		return
	}
	catalog, err := s.discoverQualityProfiles()
	if err != nil {
		writeHTTPError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	sourceID, sourceVersion := input.SourceProfileID, input.SourceProfileVersion
	if sourceID == "" {
		sourceID, sourceVersion = input.ProfileID, input.ProfileVersion
	}
	selected, ok := catalog.profiles[qualityProfileKey(sourceID, sourceVersion)]
	if !ok {
		writeHTTPError(writer, http.StatusUnprocessableEntity, analysis.NewHostError(analysis.ErrInvalidOptions, "the source quality profile is unavailable or invalid", map[string]any{
			"profile_id":      sourceID,
			"profile_version": sourceVersion,
		}))
		return
	}
	profile := selected.profile
	profile.ProfileID = input.ProfileID
	profile.ProfileVersion = input.ProfileVersion
	profile.EnabledRules = cloneQualityRuleBindings(input.RuleBindings)
	// Baseline entries are exact to the source profile identity. A copied
	// profile must not inherit a reference that can never suppress its findings.
	profile.Baseline = nil
	validated, err := quality.ValidateQualityProfile(profile, quality.NewDefaultCatalog())
	if err != nil {
		writeQualityBridgeError(writer, err)
		return
	}
	directory, err := qualityProfileDirectoryForWrite(s.getSourceRoot())
	if err != nil {
		writeHTTPError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	if err := validateQualityProfileFileName(input.FileName); err != nil {
		writeHTTPError(writer, http.StatusForbidden, err)
		return
	}
	path := filepath.Join(directory, input.FileName)
	if err := writeNewQualityProfile(path, validated); err != nil {
		status := http.StatusUnprocessableEntity
		if errors.Is(err, os.ErrExist) {
			status = http.StatusConflict
		}
		writeHTTPError(writer, status, err)
		return
	}
	name := strings.TrimSuffix(input.FileName, filepath.Ext(input.FileName))
	descriptor := qualityProfileDescriptor{Name: name, FileName: input.FileName, ProfileID: validated.ProfileID, ProfileVersion: validated.ProfileVersion, Status: "available"}
	if validated.Baseline != nil {
		descriptor.BaselineID = validated.Baseline.BaselineID
		descriptor.BaselineRevision = validated.Baseline.Revision
	}
	writeJSON(writer, http.StatusCreated, qualityProfileSaveHTTPResponse{SchemaVersion: qualityProfileSaveSchemaVersion, Status: "created", Profile: descriptor, Message: "New quality profile saved."})
}

func (s *Server) handleQualityBaselineCreate(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	if s.rejectLivePolicyWrite(writer, "create_baseline") {
		return
	}
	if s.getSourceRoot() == "" {
		writeHTTPError(writer, http.StatusForbidden, analysis.NewHostError(analysis.ErrInvalidRequest, "quality baselines can only be saved in a project-backed viewer session", nil))
		return
	}
	input, err := decodeQualityBaselineCreateRequest(request)
	if err != nil {
		writeHTTPError(writer, http.StatusBadRequest, err)
		return
	}
	catalog, err := s.discoverQualityProfiles()
	if err != nil {
		writeHTTPError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	selected, ok := catalog.profiles[qualityProfileKey(input.ProfileID, input.ProfileVersion)]
	if !ok {
		writeHTTPError(writer, http.StatusUnprocessableEntity, analysis.NewHostError(analysis.ErrInvalidOptions, "the selected quality profile is unavailable or invalid", map[string]any{
			"profile_id":      input.ProfileID,
			"profile_version": input.ProfileVersion,
		}))
		return
	}
	scopeID, value, err := s.qualityEvaluationModel(input.Scope)
	if err != nil {
		writeHTTPError(writer, qualityEvaluationHTTPStatus(err), err)
		return
	}
	fallback := value.QualityReport
	report := s.qualityReportForScope(scopeID, fallback)
	if report == nil {
		writeHTTPError(writer, http.StatusConflict, analysis.NewHostError(analysis.ErrInvalidRequest, "run the selected quality profile before creating a baseline", map[string]any{"scope_id": scopeID}))
		return
	}
	if report.ProfileID != input.ProfileID || report.ProfileVersion != input.ProfileVersion {
		writeHTTPError(writer, http.StatusConflict, analysis.NewHostError(analysis.ErrInvalidRequest, "the loaded quality report does not belong to the selected profile; run the profile again", map[string]any{
			"report_profile_id":      report.ProfileID,
			"report_profile_version": report.ProfileVersion,
		}))
		return
	}
	directory, err := qualityBaselineDirectoryForWrite(s.getSourceRoot())
	if err != nil {
		writeHTTPError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	if err := validateQualityBaselineFileName(input.FileName); err != nil {
		writeHTTPError(writer, http.StatusForbidden, err)
		return
	}
	path := filepath.Join(directory, input.FileName)
	if err := ensureQualityBaselineDestinationAvailable(path); err != nil {
		writeHTTPError(writer, http.StatusConflict, err)
		return
	}
	selectedFindingIDs, err := baselineFindingIDs(*report, input)
	if err != nil {
		writeHTTPError(writer, http.StatusBadRequest, err)
		return
	}
	baseline := quality.Baseline{
		SchemaVersion: quality.BaselineSchemaVersion,
		BaselineID:    input.BaselineID,
		Revision:      input.Revision,
		Entries:       []quality.BaselineEntry{},
		Extensions:    []quality.ExtensionBlock{},
	}
	for _, findingID := range selectedFindingIDs {
		entry, entryErr := quality.CreateBaselineEntry(*report, findingID, input.Reason, input.Owner)
		if entryErr != nil {
			writeQualityBridgeError(writer, entryErr)
			return
		}
		baseline, err = quality.AddBaselineEntry(baseline, entry)
		if err != nil {
			writeQualityBridgeError(writer, err)
			return
		}
	}
	if err := quality.ValidateBaseline(baseline); err != nil {
		writeQualityBridgeError(writer, err)
		return
	}
	var descriptor *qualityProfileDescriptor
	var attachedProfile *quality.QualityProfile
	profilePath := ""
	if input.AttachToProfile {
		profile := selected.profile
		profile.Baseline = &quality.BaselineRef{BaselineID: baseline.BaselineID, Revision: baseline.Revision}
		validated, validationErr := quality.ValidateQualityProfile(profile, quality.NewDefaultCatalog())
		if validationErr != nil {
			writeQualityBridgeError(writer, validationErr)
			return
		}
		profilePath = filepath.Join(s.getSourceRoot(), qualityProfileDirectoryName, selected.descriptor.FileName)
		if err := validateQualityProfileFileName(selected.descriptor.FileName); err != nil {
			writeHTTPError(writer, http.StatusForbidden, err)
			return
		}
		attachedProfile = &validated
	}
	if err := writeQualityBaseline(path, baseline); err != nil {
		writeHTTPError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	if attachedProfile != nil {
		if err := writeQualityProfile(profilePath, *attachedProfile); err != nil {
			if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
				err = analysis.WrapHostError(analysis.ErrHostFailure, "quality profile update failed and the new baseline could not be rolled back", errors.Join(err, removeErr), map[string]any{"baseline_file": input.FileName})
			}
			writeHTTPError(writer, http.StatusUnprocessableEntity, err)
			return
		}
		selected.descriptor.BaselineID = baseline.BaselineID
		selected.descriptor.BaselineRevision = baseline.Revision
		descriptor = &selected.descriptor
	}
	message := "Quality baseline created."
	if input.AttachToProfile {
		message = "Quality baseline created and attached to the selected profile. Run the profile again to apply suppressions."
	}
	writeJSON(writer, http.StatusCreated, qualityBaselineCreateHTTPResponse{
		SchemaVersion: qualityBaselineCreateSchemaVersion,
		Status:        "created",
		BaselineFile:  filepath.ToSlash(filepath.Join(qualityBaselineDirectoryName, input.FileName)),
		BaselineID:    baseline.BaselineID,
		Revision:      baseline.Revision,
		EntryCount:    len(baseline.Entries),
		Attached:      input.AttachToProfile,
		Profile:       descriptor,
		Message:       message,
	})
}

func decodeQualityBaselineCreateRequest(request *http.Request) (qualityBaselineCreateRequest, error) {
	data, err := io.ReadAll(io.LimitReader(request.Body, maxQualityBaselineBytes+1))
	if err != nil {
		return qualityBaselineCreateRequest{}, analysis.WrapHostError(analysis.ErrInvalidRequest, "quality baseline request could not be read", err, nil)
	}
	if len(data) > maxQualityBaselineBytes {
		return qualityBaselineCreateRequest{}, analysis.NewHostError(analysis.ErrInvalidRequest, "quality baseline request is too large", map[string]any{"max_bytes": maxQualityBaselineBytes})
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var input qualityBaselineCreateRequest
	if err := decoder.Decode(&input); err != nil {
		return qualityBaselineCreateRequest{}, analysis.WrapHostError(analysis.ErrInvalidRequest, "quality baseline request is invalid JSON", err, nil)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return qualityBaselineCreateRequest{}, analysis.NewHostError(analysis.ErrInvalidRequest, "quality baseline request contains trailing data", nil)
		}
		return qualityBaselineCreateRequest{}, analysis.WrapHostError(analysis.ErrInvalidRequest, "quality baseline request contains trailing data", err, nil)
	}
	if input.SchemaVersion != qualityBaselineCreateSchemaVersion {
		return qualityBaselineCreateRequest{}, analysis.NewHostError(analysis.ErrInvalidRequest, "quality baseline request schema version is unsupported", map[string]any{"schema_version": input.SchemaVersion, "expected": qualityBaselineCreateSchemaVersion})
	}
	input.Scope = strings.TrimSpace(input.Scope)
	input.ProfileID = strings.TrimSpace(input.ProfileID)
	input.ProfileVersion = strings.TrimSpace(input.ProfileVersion)
	input.BaselineID = strings.TrimSpace(input.BaselineID)
	input.Revision = strings.TrimSpace(input.Revision)
	input.FileName = strings.TrimSpace(input.FileName)
	input.Reason = strings.TrimSpace(input.Reason)
	input.Owner = strings.TrimSpace(input.Owner)
	for index := range input.FindingIDs {
		input.FindingIDs[index] = strings.TrimSpace(input.FindingIDs[index])
	}
	if input.ProfileID == "" || input.ProfileVersion == "" || input.BaselineID == "" || input.FileName == "" || input.Reason == "" {
		return qualityBaselineCreateRequest{}, analysis.NewHostError(analysis.ErrInvalidRequest, "quality baseline requires profile identity, baseline identity, file_name, and reason", nil)
	}
	if input.AllActive == (len(input.FindingIDs) > 0) {
		return qualityBaselineCreateRequest{}, analysis.NewHostError(analysis.ErrInvalidRequest, "quality baseline must select all_active or one or more finding_ids", nil)
	}
	return input, nil
}

func baselineFindingIDs(report quality.QualityEvaluation, input qualityBaselineCreateRequest) ([]string, error) {
	selected := append([]string(nil), input.FindingIDs...)
	if input.AllActive {
		selected = make([]string, 0, len(report.Findings))
		for _, finding := range report.Findings {
			if finding.Status == quality.StatusActive {
				selected = append(selected, finding.ID)
			}
		}
	}
	if len(selected) == 0 {
		return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "quality baseline requires at least one active finding", nil)
	}
	seen := make(map[string]struct{}, len(selected))
	for _, findingID := range selected {
		if findingID == "" {
			return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "quality baseline finding_ids may not be empty", nil)
		}
		if _, ok := seen[findingID]; ok {
			return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "quality baseline finding_ids must be unique", map[string]any{"finding_id": findingID})
		}
		seen[findingID] = struct{}{}
		found := false
		for _, finding := range report.Findings {
			if (finding.ID == findingID || finding.FindingKey == findingID) && finding.Status == quality.StatusActive {
				found = true
				break
			}
		}
		if !found {
			return nil, analysis.NewHostError(analysis.ErrInvalidOptions, "quality baseline can only include active findings in the loaded report", map[string]any{"finding_id": findingID})
		}
	}
	sort.Strings(selected)
	return selected, nil
}

func decodeQualityProfileSaveRequest(request *http.Request, saveAs bool) (qualityProfileSaveRequest, error) {
	data, err := io.ReadAll(io.LimitReader(request.Body, maxQualityProfileBytes+1))
	if err != nil {
		return qualityProfileSaveRequest{}, analysis.WrapHostError(analysis.ErrInvalidRequest, "quality profile save request could not be read", err, nil)
	}
	if len(data) > maxQualityProfileBytes {
		return qualityProfileSaveRequest{}, analysis.NewHostError(analysis.ErrInvalidRequest, "quality profile save request is too large", map[string]any{"max_bytes": maxQualityProfileBytes})
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var input qualityProfileSaveRequest
	if err := decoder.Decode(&input); err != nil {
		return qualityProfileSaveRequest{}, analysis.WrapHostError(analysis.ErrInvalidRequest, "quality profile save request is invalid JSON", err, nil)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return qualityProfileSaveRequest{}, analysis.NewHostError(analysis.ErrInvalidRequest, "quality profile save request contains trailing data", nil)
		}
		return qualityProfileSaveRequest{}, analysis.WrapHostError(analysis.ErrInvalidRequest, "quality profile save request contains trailing data", err, nil)
	}
	if input.SchemaVersion != qualityProfileSaveSchemaVersion {
		return qualityProfileSaveRequest{}, analysis.NewHostError(analysis.ErrInvalidRequest, "quality profile save request schema version is unsupported", map[string]any{"schema_version": input.SchemaVersion, "expected": qualityProfileSaveSchemaVersion})
	}
	input.ProfileID = strings.TrimSpace(input.ProfileID)
	input.ProfileVersion = strings.TrimSpace(input.ProfileVersion)
	input.SourceProfileID = strings.TrimSpace(input.SourceProfileID)
	input.SourceProfileVersion = strings.TrimSpace(input.SourceProfileVersion)
	input.FileName = strings.TrimSpace(input.FileName)
	if input.ProfileID == "" || input.ProfileVersion == "" || input.RuleBindings == nil {
		return qualityProfileSaveRequest{}, analysis.NewHostError(analysis.ErrInvalidRequest, "quality profile save requires profile identity and rule_bindings", nil)
	}
	if saveAs && input.FileName == "" {
		return qualityProfileSaveRequest{}, analysis.NewHostError(analysis.ErrInvalidRequest, "saving a new quality profile requires file_name", nil)
	}
	return input, nil
}

func qualityProfileDirectoryForWrite(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "quality profiles can only be saved in a project-backed viewer session", nil)
	}
	candidate := filepath.Join(root, qualityProfileDirectoryName)
	if err := os.MkdirAll(candidate, 0o755); err != nil {
		return "", analysis.WrapHostError(analysis.ErrHostFailure, "quality profile directory could not be created", err, map[string]any{"directory": qualityProfileDirectoryName})
	}
	return qualityProfileDirectory(root)
}

func qualityBaselineDirectoryForWrite(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "quality baselines can only be saved in a project-backed viewer session", nil)
	}
	candidate := filepath.Join(root, qualityBaselineDirectoryName)
	if err := os.MkdirAll(candidate, 0o755); err != nil {
		return "", analysis.WrapHostError(analysis.ErrHostFailure, "quality baseline directory could not be created", err, map[string]any{"directory": qualityBaselineDirectoryName})
	}
	return qualityBaselineDirectory(root)
}

func qualityBaselineDirectory(root string) (string, error) {
	candidate := filepath.Join(root, qualityBaselineDirectoryName)
	info, err := os.Stat(candidate)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", analysis.WrapHostError(analysis.ErrUnreadableProject, "quality baseline directory could not be inspected", err, map[string]any{"directory": qualityBaselineDirectoryName})
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrUnreadableProject, "quality baseline directory symlinks could not be resolved", err, map[string]any{"directory": qualityBaselineDirectoryName})
	}
	if !samePathRoot(root, resolved) {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "quality baseline directory must remain inside the opened project", map[string]any{"directory": qualityBaselineDirectoryName})
	}
	if !info.IsDir() {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "quality baseline path is not a directory", map[string]any{"directory": qualityBaselineDirectoryName})
	}
	return resolved, nil
}

func validateQualityProfileFileName(value string) error {
	if value == "" || value == "." || value == ".." || strings.ContainsAny(value, `/\\`) || strings.Contains(value, "..") || !strings.EqualFold(filepath.Ext(value), ".json") {
		return analysis.NewHostError(analysis.ErrInvalidRequest, "quality profile file name must be one direct .json file", map[string]any{"file_name": value})
	}
	return nil
}

func validateQualityBaselineFileName(value string) error {
	if value == "" || value == "." || value == ".." || strings.ContainsAny(value, `/\\`) || strings.Contains(value, "..") || !strings.EqualFold(filepath.Ext(value), ".json") {
		return analysis.NewHostError(analysis.ErrInvalidRequest, "quality baseline file name must be one direct .json file", map[string]any{"file_name": value})
	}
	return nil
}

func writeQualityProfile(path string, profile quality.QualityProfile) error {
	data, err := encodeQualityProfile(path, profile)
	if err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return analysis.NewHostError(analysis.ErrInvalidRequest, "quality profile destination may not be a symlink", map[string]any{"path": filepath.Base(path)})
	} else if err != nil && !os.IsNotExist(err) {
		return analysis.WrapHostError(analysis.ErrUnreadableProject, "quality profile destination could not be inspected", err, map[string]any{"path": filepath.Base(path)})
	}
	return writeQualityProfileData(path, data, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
}

func writeNewQualityProfile(path string, profile quality.QualityProfile) error {
	data, err := encodeQualityProfile(path, profile)
	if err != nil {
		return err
	}
	err = writeQualityProfileData(path, data, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
	if errors.Is(err, os.ErrExist) {
		return analysis.WrapHostError(analysis.ErrInvalidRequest, "the destination quality profile already exists", err, map[string]any{"file_name": filepath.Base(path)})
	}
	return err
}

func encodeQualityProfile(path string, profile quality.QualityProfile) ([]byte, error) {
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return nil, analysis.WrapHostError(analysis.ErrHostFailure, "quality profile could not be encoded", err, map[string]any{"path": path})
	}
	return append(data, '\n'), nil
}

func writeQualityProfileData(path string, data []byte, flags int) error {
	file, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return analysis.WrapHostError(analysis.ErrHostFailure, "quality profile could not be opened for writing", err, map[string]any{"path": filepath.Base(path)})
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return analysis.WrapHostError(analysis.ErrHostFailure, "quality profile could not be written", err, map[string]any{"path": filepath.Base(path)})
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return analysis.WrapHostError(analysis.ErrHostFailure, "quality profile could not be flushed", err, map[string]any{"path": filepath.Base(path)})
	}
	if err := file.Close(); err != nil {
		return analysis.WrapHostError(analysis.ErrHostFailure, "quality profile could not be closed", err, map[string]any{"path": filepath.Base(path)})
	}
	return nil
}

func ensureQualityBaselineDestinationAvailable(path string) error {
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return analysis.NewHostError(analysis.ErrInvalidRequest, "quality baseline destination may not be a symlink", map[string]any{"path": filepath.Base(path)})
		}
		return analysis.NewHostError(analysis.ErrInvalidRequest, "the destination quality baseline already exists", map[string]any{"file_name": filepath.Base(path)})
	} else if !os.IsNotExist(err) {
		return analysis.WrapHostError(analysis.ErrUnreadableProject, "quality baseline destination could not be inspected", err, map[string]any{"path": filepath.Base(path)})
	}
	return nil
}

func writeQualityBaseline(path string, baseline quality.Baseline) error {
	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return analysis.WrapHostError(analysis.ErrHostFailure, "quality baseline could not be encoded", err, map[string]any{"path": path})
	}
	data = append(data, '\n')
	if err := ensureQualityBaselineDestinationAvailable(path); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return analysis.WrapHostError(analysis.ErrHostFailure, "quality baseline could not be opened for writing", err, map[string]any{"path": filepath.Base(path)})
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return analysis.WrapHostError(analysis.ErrHostFailure, "quality baseline could not be written", err, map[string]any{"path": filepath.Base(path)})
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return analysis.WrapHostError(analysis.ErrHostFailure, "quality baseline could not be flushed", err, map[string]any{"path": filepath.Base(path)})
	}
	if err := file.Close(); err != nil {
		return analysis.WrapHostError(analysis.ErrHostFailure, "quality baseline could not be closed", err, map[string]any{"path": filepath.Base(path)})
	}
	return nil
}

func readQualityBaseline(path string) (quality.Baseline, error) {
	file, err := os.Open(path)
	if err != nil {
		return quality.Baseline{}, fmt.Errorf("baseline could not be read")
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, maxQualityBaselineBytes+1))
	if err != nil {
		return quality.Baseline{}, fmt.Errorf("baseline could not be read")
	}
	if len(data) > maxQualityBaselineBytes {
		return quality.Baseline{}, fmt.Errorf("baseline exceeds the %d byte limit", maxQualityBaselineBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var baseline quality.Baseline
	if err := decoder.Decode(&baseline); err != nil {
		return quality.Baseline{}, fmt.Errorf("baseline JSON is invalid: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return quality.Baseline{}, fmt.Errorf("baseline JSON contains trailing data")
		}
		return quality.Baseline{}, fmt.Errorf("baseline JSON contains trailing data: %w", err)
	}
	if err := quality.ValidateBaseline(baseline); err != nil {
		return quality.Baseline{}, err
	}
	return baseline, nil
}

func loadQualityBaselineForProfile(root string, profile quality.QualityProfile) (*quality.Baseline, error) {
	if profile.Baseline == nil {
		return nil, nil
	}
	directory, err := qualityBaselineDirectory(root)
	if err != nil || directory == "" {
		return nil, err
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, analysis.WrapHostError(analysis.ErrUnreadableProject, "quality baseline directory could not be read", err, map[string]any{"directory": qualityBaselineDirectoryName})
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}
		baseline, readErr := readQualityBaseline(filepath.Join(directory, entry.Name()))
		if readErr != nil || baseline.BaselineID != profile.Baseline.BaselineID || profile.Baseline.Revision != "" && baseline.Revision != profile.Baseline.Revision {
			continue
		}
		return &baseline, nil
	}
	return nil, nil
}

func (s *Server) handleQualityRules(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}

	catalog := quality.NewDefaultCatalog()
	response := qualityRulesHTTPResponse{
		SchemaVersion: qualityRulesSchemaVersion,
		Status:        "available",
		Rules:         make([]qualityRuleCatalogEntry, 0, len(catalog.ListQualityRules())),
	}
	profileID := strings.TrimSpace(request.URL.Query().Get("profile_id"))
	profileVersion := strings.TrimSpace(request.URL.Query().Get("profile_version"))
	if (profileID == "") != (profileVersion == "") {
		writeHTTPError(writer, http.StatusBadRequest, analysis.NewHostError(analysis.ErrInvalidRequest, "quality rule catalog requires both profile_id and profile_version", nil))
		return
	}

	configured := make(map[string]quality.RuleBinding)
	if profileID != "" {
		profiles, err := s.discoverQualityProfiles()
		if err != nil {
			writeHTTPError(writer, http.StatusUnprocessableEntity, err)
			return
		}
		selected, ok := profiles.profiles[qualityProfileKey(profileID, profileVersion)]
		if !ok {
			writeHTTPError(writer, http.StatusUnprocessableEntity, analysis.NewHostError(analysis.ErrInvalidOptions, "the selected quality profile is unavailable or invalid", map[string]any{"profile_id": profileID, "profile_version": profileVersion}))
			return
		}
		response.Profile = &selected.descriptor
		for _, binding := range selected.profile.EnabledRules {
			configured[qualityProfileKey(binding.RuleID, binding.RuleVersion)] = binding
		}
	}

	for _, descriptor := range catalog.ListQualityRules() {
		key := qualityProfileKey(descriptor.ID, descriptor.Version)
		binding, configuredInProfile := configured[key]
		parameterSource := "catalog default"
		if !configuredInProfile {
			var hasDefault bool
			binding, hasDefault = catalog.DefaultRuleBinding(descriptor.ID, descriptor.Version)
			if !hasDefault {
				continue
			}
		} else {
			parameterSource = "selected profile"
		}
		response.Rules = append(response.Rules, qualityRuleCatalogEntry{
			RuleDescriptor:  descriptor,
			Enabled:         binding.Enabled,
			Configured:      configuredInProfile,
			Parameters:      binding.Parameters,
			ParameterSource: parameterSource,
			Severity:        binding.Severity,
		})
	}
	if profileID == "" {
		response.Message = "All catalog rules are shown. Select a quality profile to compare or run them."
	}
	writeJSON(writer, http.StatusOK, response)
}

func (s *Server) handleQualityEvaluation(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	if s.getSourceRoot() == "" {
		writeHTTPError(writer, http.StatusForbidden, analysis.NewHostError(analysis.ErrInvalidRequest, "quality evaluation is unavailable for a model-only viewer session", nil))
		return
	}
	input, err := decodeQualityEvaluationRequest(request)
	if err != nil {
		writeHTTPError(writer, http.StatusBadRequest, err)
		return
	}
	catalog, err := s.discoverQualityProfiles()
	if err != nil {
		writeHTTPError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	selectedProfile, ok := catalog.profiles[qualityProfileKey(input.ProfileID, input.ProfileVersion)]
	if !ok {
		writeHTTPError(writer, http.StatusUnprocessableEntity, analysis.NewHostError(analysis.ErrInvalidOptions, "the selected quality profile is unavailable or invalid", map[string]any{
			"profile_id":      input.ProfileID,
			"profile_version": input.ProfileVersion,
		}))
		return
	}
	if s.liveSession != nil {
		s.handleLiveQualityEvaluation(writer, request, input, selectedProfile)
		return
	}

	scopeID, value, err := s.qualityEvaluationModel(input.Scope)
	if err != nil {
		writeHTTPError(writer, qualityEvaluationHTTPStatus(err), err)
		return
	}
	profile := selectedProfile.profile
	temporaryRuleSelection := input.RuleBindings != nil
	if temporaryRuleSelection {
		profile.EnabledRules = cloneQualityRuleBindings(input.RuleBindings)
	}
	evaluationInput, err := adapter.EvaluationInputFromModel(value)
	if err != nil {
		writeQualityBridgeError(writer, err)
		return
	}
	baseline, err := loadQualityBaselineForProfile(s.getSourceRoot(), profile)
	if err != nil {
		writeHTTPError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	evaluationInput.Baseline = baseline
	report, err := quality.EvaluateQualityProfile(profile, evaluationInput, quality.NewDefaultCatalog())
	if err != nil {
		writeQualityBridgeError(writer, err)
		return
	}
	s.storeQualityReport(scopeID, report)
	message := ""
	if temporaryRuleSelection {
		message = "Applied the selected rules for this session; the project profile was not modified."
	}
	writeJSON(writer, http.StatusOK, qualityEvaluationHTTPResponse{
		SchemaVersion: qualityEvaluationSchemaVersion,
		Status:        "available",
		ScopeID:       scopeID,
		Profile:       selectedProfile.descriptor,
		Report:        &report,
		Message:       message,
	})
}

// handleLiveQualityEvaluation adapts the historical viewer response to the
// shared live quality gateway. The browser keeps its existing response shape,
// while revision selection, currentness, temporary evaluation, and report
// semantics remain owned by live and deterministic-quality services.
func (s *Server) handleLiveQualityEvaluation(writer http.ResponseWriter, request *http.Request, input qualityEvaluationRequest, selected loadedQualityProfile) {
	if s.liveSession == nil {
		writeHTTPError(writer, http.StatusUnprocessableEntity, analysis.NewHostError(analysis.ErrInvalidRequest, "live quality evaluation is unavailable", nil))
		return
	}
	requestValue := live.QualityEvaluationRequest{
		SessionID:      s.liveSession.Config().SessionID,
		Consistency:    live.ConsistencyRequireCurrent,
		ProfileID:      input.ProfileID,
		ProfileVersion: input.ProfileVersion,
		RuleBindings:   cloneQualityRuleBindings(input.RuleBindings),
		Persist:        false,
	}
	envelope, err := s.liveSession.QualityGateway().EvaluateQuality(request.Context(), requestValue)
	if err != nil {
		converted := liveViewerHostError(err)
		writeHTTPError(writer, liveViewerHTTPStatus(converted), converted)
		return
	}
	result, ok := envelope.Result.(live.QualityEvaluationResult)
	if !ok || result.Report == nil {
		writeHTTPError(writer, http.StatusUnprocessableEntity, analysis.NewHostError(analysis.ErrInvalidModel, "live quality evaluation returned no report", nil))
		return
	}
	s.storeQualityReportAt("all", *result.Report, envelope.Revision)
	message := ""
	if input.RuleBindings != nil {
		message = "Applied the selected rules for this session; the project profile was not modified."
	}
	writeJSON(writer, http.StatusOK, qualityEvaluationHTTPResponse{
		SchemaVersion: qualityEvaluationSchemaVersion,
		Status:        "available",
		ScopeID:       "all",
		Profile:       selected.descriptor,
		Report:        result.Report,
		Message:       message,
	})
}

// rejectLivePolicyWrite closes the legacy viewer write routes when the viewer
// is backed by a live session. Live policy writes belong to the explicit MCP
// or CLI policy operation, where the session permission and authorization are
// checked and an audit record is returned.
func (s *Server) rejectLivePolicyWrite(writer http.ResponseWriter, operation string) bool {
	if s == nil || s.liveSession == nil {
		return false
	}
	writeHTTPError(writer, http.StatusForbidden, analysis.NewHostError(analysis.ErrorCode(live.ErrorQualityPolicyPermissionDenied), "quality policy writes are disabled in the live viewer; use an explicitly authorized live policy operation", map[string]any{
		"operation":  operation,
		"session_id": s.liveSession.Config().SessionID,
	}))
	return true
}

func cloneQualityRuleBindings(bindings []quality.RuleBinding) []quality.RuleBinding {
	if bindings == nil {
		return nil
	}
	result := make([]quality.RuleBinding, len(bindings))
	copy(result, bindings)
	return result
}

func decodeQualityEvaluationRequest(request *http.Request) (qualityEvaluationRequest, error) {
	data, err := io.ReadAll(io.LimitReader(request.Body, maxQualityEvaluationRequestBytes+1))
	if err != nil {
		return qualityEvaluationRequest{}, analysis.WrapHostError(analysis.ErrInvalidRequest, "quality evaluation request could not be read", err, nil)
	}
	if len(data) > maxQualityEvaluationRequestBytes {
		return qualityEvaluationRequest{}, analysis.NewHostError(analysis.ErrInvalidRequest, "quality evaluation request is too large", map[string]any{"max_bytes": maxQualityEvaluationRequestBytes})
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var input qualityEvaluationRequest
	if err := decoder.Decode(&input); err != nil {
		return qualityEvaluationRequest{}, analysis.WrapHostError(analysis.ErrInvalidRequest, "quality evaluation request is invalid JSON", err, nil)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return qualityEvaluationRequest{}, analysis.NewHostError(analysis.ErrInvalidRequest, "quality evaluation request contains trailing data", nil)
		}
		return qualityEvaluationRequest{}, analysis.WrapHostError(analysis.ErrInvalidRequest, "quality evaluation request contains trailing data", err, nil)
	}
	if input.SchemaVersion != qualityEvaluationRequestSchemaVersion {
		return qualityEvaluationRequest{}, analysis.NewHostError(analysis.ErrInvalidRequest, "quality evaluation request schema version is unsupported", map[string]any{"schema_version": input.SchemaVersion, "expected": qualityEvaluationRequestSchemaVersion})
	}
	input.ProfileID = strings.TrimSpace(input.ProfileID)
	input.ProfileVersion = strings.TrimSpace(input.ProfileVersion)
	input.Scope = strings.TrimSpace(input.Scope)
	if input.ProfileID == "" || input.ProfileVersion == "" {
		return qualityEvaluationRequest{}, analysis.NewHostError(analysis.ErrInvalidRequest, "quality evaluation requires a profile id and version", nil)
	}
	return input, nil
}

func (s *Server) qualityEvaluationModel(scope string) (string, model.Model, error) {
	requested := strings.TrimSpace(scope)
	if !s.isAggregate() {
		if requested != "" && !strings.EqualFold(requested, "all") {
			return "", model.Model{}, analysis.NewHostError(analysis.ErrInvalidRequest, "a concrete analysis scope is unavailable for this viewer session", map[string]any{"scope_id": requested})
		}
		value := s.snapshot()
		if value.ModelID == "" {
			return "", model.Model{}, analysis.NewHostError(analysis.ErrInvalidModel, "quality evaluation has no loaded canonical model", nil)
		}
		return "all", value, nil
	}
	aggregate := s.aggregateSnapshot()
	if aggregate == nil {
		return "", model.Model{}, analysis.NewHostError(analysis.ErrInvalidModel, "quality evaluation has no loaded analysis run", nil)
	}
	if requested == "" || strings.EqualFold(requested, "all") {
		selected, err := aggregate.SelectAnalysisScope("all")
		if err != nil {
			return "", model.Model{}, err
		}
		return "all", selected.Model, usableQualityModel(selected.Model, "all")
	}
	selected, err := aggregate.SelectAnalysisScope(requested)
	if err != nil {
		return "", model.Model{}, err
	}
	return requested, selected.Model, usableQualityModel(selected.Model, requested)
}

func usableQualityModel(value model.Model, scope string) error {
	if value.ModelID == "" {
		return analysis.NewHostError(analysis.ErrAnalysisScopeNotFound, "quality evaluation is unavailable because the selected scope has no usable model", map[string]any{"scope_id": scope})
	}
	return nil
}

func qualityEvaluationHTTPStatus(err error) int {
	switch analysis.ErrorCodeOf(err) {
	case analysis.ErrAnalysisScopeNotFound:
		return http.StatusNotFound
	case analysis.ErrInvalidRequest, analysis.ErrInvalidOptions:
		return http.StatusBadRequest
	default:
		return http.StatusUnprocessableEntity
	}
}

func writeQualityBridgeError(writer http.ResponseWriter, err error) {
	var validationErr *quality.ProfileValidationError
	if errors.As(err, &validationErr) {
		writeHTTPError(writer, http.StatusUnprocessableEntity, analysis.NewHostError(analysis.ErrorCode(quality.ErrorProfileInvalid), validationErr.Error(), map[string]any{"diagnostics": validationErr.Diagnostics}))
		return
	}
	var qualityErr *quality.QualityError
	if errors.As(err, &qualityErr) {
		writeHTTPError(writer, qualityHTTPStatus(qualityErr.Code), analysis.NewHostError(analysis.ErrorCode(qualityErr.Code), qualityErr.Message, qualityErr.Details))
		return
	}
	writeHTTPError(writer, http.StatusUnprocessableEntity, analysis.WrapHostError(analysis.ErrInvalidModel, "quality evaluation failed", err, nil))
}

func (s *Server) discoverQualityProfiles() (qualityProfileCatalog, error) {
	response := qualityProfilesHTTPResponse{
		SchemaVersion: qualityProfilesSchemaVersion,
		Directory:     qualityProfileDirectoryName,
		Profiles:      []qualityProfileDescriptor{},
	}
	root := s.getSourceRoot()
	if root == "" {
		response.Status = "unavailable"
		response.Message = "Quality profiles are unavailable in a model-only viewer session."
		return qualityProfileCatalog{response: response, profiles: map[string]loadedQualityProfile{}}, nil
	}
	directory, err := qualityProfileDirectory(root)
	if err != nil {
		return qualityProfileCatalog{}, err
	}
	if directory == "" {
		response.Status = "empty"
		response.Message = "No quality profiles were found. Add JSON profiles to quality-profiles/ to enable in-viewer evaluation."
		return qualityProfileCatalog{response: response, profiles: map[string]loadedQualityProfile{}}, nil
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		if os.IsNotExist(err) {
			response.Status = "empty"
			response.Message = "No quality profiles were found. Add JSON profiles to quality-profiles/ to enable in-viewer evaluation."
			return qualityProfileCatalog{response: response, profiles: map[string]loadedQualityProfile{}}, nil
		}
		return qualityProfileCatalog{}, analysis.WrapHostError(analysis.ErrUnreadableProject, "quality profile directory could not be read", err, map[string]any{"directory": qualityProfileDirectoryName})
	}

	documents := make([]loadedQualityProfile, 0, len(entries))
	profileCatalog := quality.NewDefaultCatalog()
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if name == "" {
			name = entry.Name()
		}
		document := loadedQualityProfile{descriptor: qualityProfileDescriptor{Name: name, FileName: entry.Name(), Status: "invalid"}}
		profile, readErr := readQualityProfile(filepath.Join(directory, entry.Name()))
		if readErr != nil {
			document.descriptor.Message = readErr.Error()
			documents = append(documents, document)
			continue
		}
		validated, validationErr := quality.ValidateQualityProfile(profile, profileCatalog)
		if validationErr != nil {
			document.descriptor.Message = validationErr.Error()
			documents = append(documents, document)
			continue
		}
		document.profile = validated
		document.descriptor.ProfileID = validated.ProfileID
		document.descriptor.ProfileVersion = validated.ProfileVersion
		if validated.Baseline != nil {
			document.descriptor.BaselineID = validated.Baseline.BaselineID
			document.descriptor.BaselineRevision = validated.Baseline.Revision
		}
		document.descriptor.Status = "available"
		documents = append(documents, document)
	}

	sort.Slice(documents, func(i, j int) bool {
		left, right := documents[i].descriptor, documents[j].descriptor
		return left.Name < right.Name || left.Name == right.Name && (left.ProfileID < right.ProfileID || left.ProfileID == right.ProfileID && left.ProfileVersion < right.ProfileVersion)
	})
	profiles := make(map[string]loadedQualityProfile)
	duplicateIndexes := make(map[string][]int)
	for index, document := range documents {
		if document.descriptor.Status == "available" {
			key := qualityProfileKey(document.descriptor.ProfileID, document.descriptor.ProfileVersion)
			duplicateIndexes[key] = append(duplicateIndexes[key], index)
		}
	}
	for key, indexes := range duplicateIndexes {
		if len(indexes) == 1 {
			profiles[key] = documents[indexes[0]]
			continue
		}
		for _, index := range indexes {
			documents[index].descriptor.Status = "invalid"
			documents[index].descriptor.Message = "another profile uses the same profile id and version"
		}
	}
	for _, document := range documents {
		response.Profiles = append(response.Profiles, document.descriptor)
	}
	available, invalid := 0, 0
	for _, profile := range response.Profiles {
		if profile.Status == "available" {
			available++
		} else {
			invalid++
		}
	}
	switch {
	case available == 0 && invalid == 0:
		response.Status = "empty"
		response.Message = "No quality profiles were found. Add JSON profiles to quality-profiles/ to enable in-viewer evaluation."
	case available == 0:
		response.Status = "invalid"
		response.Message = "No usable quality profiles were found. Review the invalid profile entries."
	case invalid > 0:
		response.Status = "partial"
		response.Message = "Some quality profile files are invalid and cannot be selected."
	default:
		response.Status = "available"
	}
	return qualityProfileCatalog{response: response, profiles: profiles}, nil
}

func qualityProfileDirectory(root string) (string, error) {
	candidate := filepath.Join(root, qualityProfileDirectoryName)
	info, err := os.Stat(candidate)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", analysis.WrapHostError(analysis.ErrUnreadableProject, "quality profile directory could not be inspected", err, map[string]any{"directory": qualityProfileDirectoryName})
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrUnreadableProject, "quality profile directory symlinks could not be resolved", err, map[string]any{"directory": qualityProfileDirectoryName})
	}
	if !samePathRoot(root, resolved) {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "quality profile directory must remain inside the opened project", map[string]any{"directory": qualityProfileDirectoryName})
	}
	if !info.IsDir() {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "quality profile path is not a directory", map[string]any{"directory": qualityProfileDirectoryName})
	}
	return resolved, nil
}

func readQualityProfile(path string) (quality.QualityProfile, error) {
	file, err := os.Open(path)
	if err != nil {
		return quality.QualityProfile{}, fmt.Errorf("profile could not be read")
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, maxQualityProfileBytes+1))
	if err != nil {
		return quality.QualityProfile{}, fmt.Errorf("profile could not be read")
	}
	if len(data) > maxQualityProfileBytes {
		return quality.QualityProfile{}, fmt.Errorf("profile exceeds the %d byte limit", maxQualityProfileBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var profile quality.QualityProfile
	if err := decoder.Decode(&profile); err != nil {
		return quality.QualityProfile{}, fmt.Errorf("profile JSON is invalid: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return quality.QualityProfile{}, fmt.Errorf("profile JSON contains trailing data")
		}
		return quality.QualityProfile{}, fmt.Errorf("profile JSON contains trailing data: %w", err)
	}
	return profile, nil
}

func qualityProfileKey(id, version string) string {
	return strings.TrimSpace(id) + "\x00" + strings.TrimSpace(version)
}

func qualityReportKey(scope string) string {
	if strings.TrimSpace(scope) == "" || strings.EqualFold(strings.TrimSpace(scope), "all") {
		return "all"
	}
	return strings.TrimSpace(scope)
}

func (s *Server) storeQualityReport(scope string, report quality.QualityEvaluation) {
	s.storeQualityReportAt(scope, report, 0)
}

func (s *Server) storeQualityReportAt(scope string, report quality.QualityEvaluation, revision int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.qualityReports == nil {
		s.qualityReports = make(map[string]quality.QualityEvaluation)
	}
	if s.qualityReportRevisions == nil {
		s.qualityReportRevisions = make(map[string]int)
	}
	s.qualityReports[qualityReportKey(scope)] = report
	s.qualityReportRevisions[qualityReportKey(scope)] = revision
}

func (s *Server) qualityReportForScope(scope string, fallback *quality.QualityEvaluation) *quality.QualityEvaluation {
	s.mu.RLock()
	report, ok := s.qualityReports[qualityReportKey(scope)]
	s.mu.RUnlock()
	if ok {
		copyReport := report
		return &copyReport
	}
	if fallback == nil {
		return nil
	}
	copyReport := *fallback
	return &copyReport
}

// qualityReportForRequest keeps temporary live evaluations tied to the same
// revision as the model request. A temporary report must never leak into a
// later revision after a watcher rebuild.
func (s *Server) qualityReportForRequest(scope string, fallback *quality.QualityEvaluation, request *http.Request) *quality.QualityEvaluation {
	if s == nil || s.liveSession == nil {
		return s.qualityReportForScope(scope, fallback)
	}
	revision, supplied, err := requestedLiveRevision(request)
	if err != nil {
		return fallback
	}
	if !supplied {
		if record, queryErr := s.liveSession.QueryLatestReady(request.Context()); queryErr == nil && record != nil {
			revision = record.Snapshot.Revision
		}
	}
	key := qualityReportKey(scope)
	s.mu.RLock()
	report, ok := s.qualityReports[key]
	reportRevision := s.qualityReportRevisions[key]
	s.mu.RUnlock()
	if ok && reportRevision == revision {
		copyReport := report
		return &copyReport
	}
	if fallback == nil {
		return nil
	}
	copyReport := *fallback
	return &copyReport
}
