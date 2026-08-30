package analysis

import (
	"context"
	"errors"
	"fmt"
)

type ErrorCode string

const (
	ErrInvalidRequest            ErrorCode = "invalid_request"
	ErrInvalidManifest           ErrorCode = "invalid_manifest"
	ErrAPIIncompatible           ErrorCode = "api_incompatible"
	ErrDuplicateAnalyzer         ErrorCode = "duplicate_analyzer"
	ErrNoAnalyzer                ErrorCode = "no_analyzer"
	ErrAmbiguousAnalyzer         ErrorCode = "ambiguous_analyzer"
	ErrUnsupportedProject        ErrorCode = "unsupported_project"
	ErrUnreadableProject         ErrorCode = "unreadable_project"
	ErrModuleSelection           ErrorCode = "module_selection"
	ErrInvalidOptions            ErrorCode = "invalid_options"
	ErrAnalysisConfigInvalid     ErrorCode = "analysis_config_invalid"
	ErrAnalysisAssignmentInvalid ErrorCode = "analysis_assignment_invalid"
	// ErrAssignmentPathInvalid is kept as a semantic alias for callers that
	// classify the invalid assignment by its failing field.
	ErrAssignmentPathInvalid            ErrorCode = ErrAnalysisAssignmentInvalid
	ErrAssignmentDuplicatePath          ErrorCode = "analysis_assignment_duplicate_path"
	ErrAssignmentOptionInvalid          ErrorCode = "analysis_assignment_option_invalid"
	ErrAssignmentAnalyzerUnavailable    ErrorCode = "analysis_analyzer_unavailable"
	ErrAnalysisScopeFilterInvalid       ErrorCode = "analysis_scope_filter_invalid"
	ErrAnalysisScopeNotFound            ErrorCode = "analysis_scope_not_found"
	ErrAnalysisScopeStale               ErrorCode = "analysis_scope_stale"
	ErrAnalysisSelectionConflict        ErrorCode = "analysis_selection_conflict"
	ErrCLISelectionConflict             ErrorCode = ErrAnalysisSelectionConflict
	ErrAnalysisCacheUnavailable         ErrorCode = "analysis_cache_unavailable"
	ErrUnsupportedOption                ErrorCode = "unsupported_option"
	ErrSaveAsRequired                   ErrorCode = "save_as_required"
	ErrPersistenceUnavailable           ErrorCode = "persistence_unavailable"
	ErrInvalidModel                     ErrorCode = "invalid_model"
	ErrCancelled                        ErrorCode = "cancelled"
	ErrAnalyzerFailed                   ErrorCode = "analyzer_failed"
	ErrResultInvalid                    ErrorCode = "result_invalid"
	ErrHostFailure                      ErrorCode = "host_failure"
	ErrAnalyzerPackageIndexInvalid      ErrorCode = "analyzer_package_index_invalid"
	ErrAnalyzerPackageNotFound          ErrorCode = "analyzer_package_not_found"
	ErrAnalyzerPlatformUnsupported      ErrorCode = "analyzer_platform_unsupported"
	ErrAnalyzerPackageIntegrityMismatch ErrorCode = "analyzer_package_integrity_mismatch"
	ErrAnalyzerPackageManifestMismatch  ErrorCode = "analyzer_package_manifest_mismatch"
	ErrAnalyzerPackageAPIIncompatible   ErrorCode = "analyzer_package_api_incompatible"
	ErrAnalyzerRuntimeOverrideRequired  ErrorCode = "analyzer_runtime_override_required"
	ErrAnalyzerPackageLaunchFailed      ErrorCode = "analyzer_package_launch_failed"
	ErrSourceScopeUnavailable           ErrorCode = "source_scope_unavailable"
	ErrSourceFileEnumerationFailed      ErrorCode = "source_file_enumeration_failed"
	ErrSourceFileHashFailed             ErrorCode = "source_file_hash_failed"
	ErrSourceExtractorUnavailable       ErrorCode = "source_extractor_unavailable"
	ErrSourceExtractorFailed            ErrorCode = "source_extractor_failed"
	ErrSourceFactInvalid                ErrorCode = "source_fact_invalid"
	ErrSourceSpanInvalid                ErrorCode = "source_span_invalid"
	ErrSourceReferenceInvalid           ErrorCode = "source_reference_invalid"
	ErrSourceRelationInvalid            ErrorCode = "source_relation_invalid"
	ErrDocumentationSelectionInvalid    ErrorCode = "documentation_selection_invalid"
	ErrMetricFactInvalid                ErrorCode = "metric_fact_invalid"
	ErrSourceIndexDigestMismatch        ErrorCode = "source_index_digest_mismatch"
	ErrSourceIndexCapabilityUnsupported ErrorCode = "source_index_capability_unsupported"
)

type HostError struct {
	Code    ErrorCode      `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	Cause   error          `json:"-"`
}

func (e *HostError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *HostError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func NewHostError(code ErrorCode, message string, details map[string]any) *HostError {
	return &HostError{Code: code, Message: message, Details: details}
}

func WrapHostError(code ErrorCode, message string, cause error, details map[string]any) *HostError {
	return &HostError{Code: code, Message: message, Cause: cause, Details: details}
}

func ErrorCodeOf(err error) ErrorCode {
	var hostErr *HostError
	if errors.As(err, &hostErr) {
		return hostErr.Code
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return ErrCancelled
	}
	return ErrHostFailure
}

// IsAnalyzerPackageError reports failures that must remain terminal while a
// packaged runtime is being selected. In particular, callers must not turn a
// missing or rejected package into an implicit in-process fallback.
func IsAnalyzerPackageError(err error) bool {
	switch ErrorCodeOf(err) {
	case ErrAnalyzerPackageIndexInvalid,
		ErrAnalyzerPackageNotFound,
		ErrAnalyzerPlatformUnsupported,
		ErrAnalyzerPackageIntegrityMismatch,
		ErrAnalyzerPackageManifestMismatch,
		ErrAnalyzerPackageAPIIncompatible,
		ErrAnalyzerPackageLaunchFailed:
		return true
	default:
		return false
	}
}

func ExitCodeForError(err error) int {
	switch ErrorCodeOf(err) {
	case ErrInvalidRequest, ErrInvalidManifest, ErrAPIIncompatible,
		ErrDuplicateAnalyzer, ErrAmbiguousAnalyzer, ErrModuleSelection, ErrInvalidOptions,
		ErrAnalysisConfigInvalid, ErrAnalysisAssignmentInvalid, ErrAssignmentDuplicatePath,
		ErrAssignmentOptionInvalid, ErrAnalysisSelectionConflict, ErrAnalysisScopeStale,
		ErrAnalysisScopeFilterInvalid,
		ErrAnalysisScopeNotFound,
		ErrUnsupportedOption, ErrSaveAsRequired, ErrPersistenceUnavailable,
		ErrAnalyzerPackageIndexInvalid, ErrAnalyzerRuntimeOverrideRequired:
		return 2
	case ErrNoAnalyzer, ErrUnsupportedProject, ErrUnreadableProject, ErrInvalidModel,
		ErrAnalyzerPackageNotFound, ErrAnalyzerPlatformUnsupported:
		return 3
	case ErrCancelled:
		return 130
	default:
		return 4
	}
}

func ExitCodeForStatus(status AnalysisStatus) int {
	switch status {
	case StatusComplete, StatusPartial:
		return 0
	case StatusCancelled:
		return 130
	default:
		return 4
	}
}
