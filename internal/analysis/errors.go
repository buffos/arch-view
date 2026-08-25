package analysis

import (
	"context"
	"errors"
	"fmt"
)

type ErrorCode string

const (
	ErrInvalidRequest     ErrorCode = "invalid_request"
	ErrInvalidManifest    ErrorCode = "invalid_manifest"
	ErrAPIIncompatible    ErrorCode = "api_incompatible"
	ErrDuplicateAnalyzer  ErrorCode = "duplicate_analyzer"
	ErrNoAnalyzer         ErrorCode = "no_analyzer"
	ErrAmbiguousAnalyzer  ErrorCode = "ambiguous_analyzer"
	ErrUnsupportedProject ErrorCode = "unsupported_project"
	ErrUnreadableProject  ErrorCode = "unreadable_project"
	ErrModuleSelection    ErrorCode = "module_selection"
	ErrInvalidOptions     ErrorCode = "invalid_options"
	ErrCancelled          ErrorCode = "cancelled"
	ErrAnalyzerFailed     ErrorCode = "analyzer_failed"
	ErrResultInvalid      ErrorCode = "result_invalid"
	ErrHostFailure        ErrorCode = "host_failure"
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

func ExitCodeForError(err error) int {
	switch ErrorCodeOf(err) {
	case ErrInvalidRequest, ErrInvalidManifest, ErrAPIIncompatible,
		ErrDuplicateAnalyzer, ErrAmbiguousAnalyzer, ErrModuleSelection, ErrInvalidOptions:
		return 2
	case ErrNoAnalyzer, ErrUnsupportedProject, ErrUnreadableProject:
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
