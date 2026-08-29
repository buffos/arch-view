package processprotocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

const (
	// ProtocolVersion is the version negotiated by the external analyzer
	// handshake. Minor versions may add fields only when they remain
	// semantically compatible with the host.
	ProtocolVersion = analysis.AnalyzerAPIVersion

	// DescriptorSchemaVersion identifies the explicit local plugin descriptor.
	DescriptorSchemaVersion = "arch-view.plugin/v1"

	// DefaultMaxFrameBytes bounds one complete NDJSON line, including its
	// terminating newline when present.
	DefaultMaxFrameBytes = 8 * 1024 * 1024
)

type FrameType string

const (
	FrameHello      FrameType = "hello"
	FrameDetect     FrameType = "detect"
	FrameAnalyze    FrameType = "analyze"
	FrameCancel     FrameType = "cancel"
	FrameCandidate  FrameType = "candidate"
	FrameResult     FrameType = "result"
	FrameDiagnostic FrameType = "diagnostic"
	FrameDone       FrameType = "done"
	FrameFatal      FrameType = "fatal"
)

// Frame is the versioned protocol envelope. Result payloads intentionally use
// the existing analysis.AnalysisResult type rather than introducing a second
// model schema at the process boundary.
type Frame struct {
	Type        FrameType                    `json:"type"`
	RequestID   string                       `json:"request_id,omitempty"`
	Protocol    string                       `json:"protocol,omitempty"`
	Manifest    *analysis.Manifest           `json:"manifest,omitempty"`
	ProjectRoot string                       `json:"project_root,omitempty"`
	Selection   *analysis.AnalyzerSelection  `json:"selection,omitempty"`
	Options     *analysis.EffectiveOptions   `json:"options,omitempty"`
	SourceScope *analysis.SourceScope        `json:"source_scope,omitempty"`
	Candidate   *analysis.DetectionCandidate `json:"candidate,omitempty"`
	Result      *analysis.AnalysisResult     `json:"result,omitempty"`
	Diagnostic  *analysis.Diagnostic         `json:"diagnostic,omitempty"`
	Status      analysis.AnalysisStatus      `json:"status,omitempty"`
	Code        string                       `json:"code,omitempty"`
	Message     string                       `json:"message,omitempty"`
	Details     map[string]any               `json:"details,omitempty"`
	Reason      string                       `json:"reason,omitempty"`
}

// Descriptor is the typed representation of an explicitly supplied local
// process-plugin descriptor. Command and Args are argv values; they are never
// interpreted as a shell command string.
type Descriptor struct {
	SchemaVersion    string            `json:"schema_version"`
	Manifest         analysis.Manifest `json:"manifest"`
	Command          string            `json:"command"`
	Args             []string          `json:"args,omitempty"`
	WorkingDirectory string            `json:"working_directory,omitempty"`
}

// PluginDescriptor is kept as a descriptive alias for callers that use the
// domain term from the architecture contract.
type PluginDescriptor = Descriptor

type ErrorKind string

const (
	ErrorMalformedJSON       ErrorKind = "malformed_json"
	ErrorFrameTooLarge       ErrorKind = "frame_too_large"
	ErrorUnknownFrameType    ErrorKind = "unknown_frame_type"
	ErrorMissingField        ErrorKind = "missing_field"
	ErrorInvalidRequestID    ErrorKind = "invalid_request_id"
	ErrorRequestIDMismatch   ErrorKind = "request_id_mismatch"
	ErrorUnsupportedProtocol ErrorKind = "unsupported_protocol"
	ErrorInvalidPayload      ErrorKind = "invalid_payload"
	ErrorProtocolState       ErrorKind = "protocol_state"
	ErrorLateFrame           ErrorKind = "late_frame"
	ErrorDuplicateTerminal   ErrorKind = "duplicate_terminal"
	ErrorDuplicatePayload    ErrorKind = "duplicate_payload"
	ErrorUnexpectedFrame     ErrorKind = "unexpected_frame"
	ErrorDescriptorInvalid   ErrorKind = "descriptor_invalid"
)

var (
	ErrMalformedJSON       = errors.New("malformed JSON frame")
	ErrFrameTooLarge       = errors.New("protocol frame exceeds the maximum size")
	ErrUnknownFrameType    = errors.New("unknown protocol frame type")
	ErrMissingField        = errors.New("protocol frame is missing a required field")
	ErrInvalidRequestID    = errors.New("protocol request ID is invalid")
	ErrRequestIDMismatch   = errors.New("protocol request ID does not match the active request")
	ErrUnsupportedProtocol = errors.New("protocol version is not supported")
	ErrInvalidPayload      = errors.New("protocol payload is invalid")
	ErrProtocolState       = errors.New("protocol session state is invalid")
	ErrLateFrame           = errors.New("protocol frame arrived after the terminal frame")
	ErrDuplicateTerminal   = errors.New("protocol session has more than one terminal frame")
	ErrDuplicatePayload    = errors.New("protocol session has more than one result payload")
	ErrUnexpectedFrame     = errors.New("protocol frame is not valid for this operation")
	ErrDescriptorInvalid   = errors.New("plugin descriptor is invalid")
)

type ProtocolError struct {
	Kind      ErrorKind
	Message   string
	FrameType FrameType
	RequestID string
	Cause     error
}

func (e *ProtocolError) Error() string {
	if e == nil {
		return ""
	}
	if e.FrameType != "" {
		return fmt.Sprintf("%s (%s): %s", e.Kind, e.FrameType, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}

func (e *ProtocolError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *ProtocolError) Is(target error) bool {
	if e == nil {
		return false
	}
	switch target {
	case ErrMalformedJSON:
		return e.Kind == ErrorMalformedJSON
	case ErrFrameTooLarge:
		return e.Kind == ErrorFrameTooLarge
	case ErrUnknownFrameType:
		return e.Kind == ErrorUnknownFrameType
	case ErrMissingField:
		return e.Kind == ErrorMissingField
	case ErrInvalidRequestID:
		return e.Kind == ErrorInvalidRequestID
	case ErrRequestIDMismatch:
		return e.Kind == ErrorRequestIDMismatch
	case ErrUnsupportedProtocol:
		return e.Kind == ErrorUnsupportedProtocol
	case ErrInvalidPayload:
		return e.Kind == ErrorInvalidPayload
	case ErrProtocolState:
		return e.Kind == ErrorProtocolState
	case ErrLateFrame:
		return e.Kind == ErrorLateFrame
	case ErrDuplicateTerminal:
		return e.Kind == ErrorDuplicateTerminal
	case ErrDuplicatePayload:
		return e.Kind == ErrorDuplicatePayload
	case ErrUnexpectedFrame:
		return e.Kind == ErrorUnexpectedFrame
	case ErrDescriptorInvalid:
		return e.Kind == ErrorDescriptorInvalid
	default:
		return false
	}
}

func newProtocolError(kind ErrorKind, frameType FrameType, requestID, message string, cause error) error {
	return &ProtocolError{
		Kind:      kind,
		FrameType: frameType,
		RequestID: requestID,
		Message:   message,
		Cause:     cause,
	}
}

var protocolVersionPattern = regexp.MustCompile(`^arch-view\.analyzer/v1(?:\.[0-9]+)?$`)

func knownFrameType(frameType FrameType) bool {
	switch frameType {
	case FrameHello, FrameDetect, FrameAnalyze, FrameCancel, FrameCandidate, FrameResult, FrameDiagnostic, FrameDone, FrameFatal:
		return true
	default:
		return false
	}
}

func validStatus(status analysis.AnalysisStatus) bool {
	switch status {
	case analysis.StatusComplete, analysis.StatusPartial, analysis.StatusFailed, analysis.StatusCancelled:
		return true
	default:
		return false
	}
}

func validRequestID(requestID string) bool {
	return strings.TrimSpace(requestID) != "" && !strings.ContainsAny(requestID, "\r\n\x00")
}

func supportedProtocol(protocol string) bool {
	return protocolVersionPattern.MatchString(protocol)
}

// ValidateFrame validates the envelope and frame-specific required fields.
// Canonical result contents remain the responsibility of
// analysis.ValidateAnalysisResult after the host has merged streamed
// diagnostics.
func ValidateFrame(frame Frame) error {
	return validateFrameShape(frame, nil)
}

func validateFrameShape(frame Frame, fields map[string]json.RawMessage) error {
	if frame.Type == "" {
		return newProtocolError(ErrorMissingField, frame.Type, frame.RequestID, "frame requires type", nil)
	}
	if !knownFrameType(frame.Type) {
		return newProtocolError(ErrorUnknownFrameType, frame.Type, frame.RequestID, "frame type is not recognized", nil)
	}

	if frame.Type == FrameHello {
		if frame.RequestID != "" || hasField(fields, "request_id") {
			return newProtocolError(ErrorInvalidRequestID, frame.Type, frame.RequestID, "hello must not contain a request ID", nil)
		}
		if !supportedProtocol(frame.Protocol) {
			return newProtocolError(ErrorUnsupportedProtocol, frame.Type, frame.RequestID, "hello protocol is not supported", nil)
		}
		if frame.Manifest == nil {
			return newProtocolError(ErrorMissingField, frame.Type, frame.RequestID, "hello requires manifest", nil)
		}
		if err := analysis.ValidateManifest(*frame.Manifest); err != nil {
			return newProtocolError(ErrorInvalidPayload, frame.Type, frame.RequestID, "hello manifest is invalid", err)
		}
		return nil
	}

	if !validRequestID(frame.RequestID) {
		return newProtocolError(ErrorInvalidRequestID, frame.Type, frame.RequestID, "non-hello frames require a non-empty request ID", nil)
	}

	switch frame.Type {
	case FrameDetect:
		if strings.TrimSpace(frame.ProjectRoot) == "" {
			return newProtocolError(ErrorMissingField, frame.Type, frame.RequestID, "detect requires project_root", nil)
		}
	case FrameAnalyze:
		if strings.TrimSpace(frame.ProjectRoot) == "" {
			return newProtocolError(ErrorMissingField, frame.Type, frame.RequestID, "analyze requires project_root", nil)
		}
		if frame.Selection == nil {
			return newProtocolError(ErrorMissingField, frame.Type, frame.RequestID, "analyze requires selection", nil)
		}
		if strings.TrimSpace(frame.Selection.AnalyzerID) == "" || strings.TrimSpace(frame.Selection.Mode) == "" {
			return newProtocolError(ErrorInvalidPayload, frame.Type, frame.RequestID, "analyze selection requires analyzer_id and mode", nil)
		}
		if frame.Options == nil {
			return newProtocolError(ErrorMissingField, frame.Type, frame.RequestID, "analyze requires options", nil)
		}
		if frame.Options.Values == nil || strings.TrimSpace(frame.Options.Fingerprint) == "" {
			return newProtocolError(ErrorInvalidPayload, frame.Type, frame.RequestID, "analyze options require values and fingerprint", nil)
		}
	case FrameCancel:
		// Cancel has no additional required payload. Reason is optional.
	case FrameCandidate:
		if frame.Candidate == nil {
			return newProtocolError(ErrorMissingField, frame.Type, frame.RequestID, "candidate requires candidate", nil)
		}
		if err := validateCandidate(*frame.Candidate); err != nil {
			return newProtocolError(ErrorInvalidPayload, frame.Type, frame.RequestID, "candidate payload is invalid", err)
		}
	case FrameResult:
		if frame.Result == nil {
			return newProtocolError(ErrorMissingField, frame.Type, frame.RequestID, "result requires result", nil)
		}
	case FrameDiagnostic:
		if frame.Diagnostic == nil {
			return newProtocolError(ErrorMissingField, frame.Type, frame.RequestID, "diagnostic requires diagnostic", nil)
		}
		if err := validateDiagnostic(*frame.Diagnostic); err != nil {
			return newProtocolError(ErrorInvalidPayload, frame.Type, frame.RequestID, "diagnostic payload is invalid", err)
		}
	case FrameDone:
		if !validStatus(frame.Status) {
			return newProtocolError(ErrorInvalidPayload, frame.Type, frame.RequestID, "done status is invalid", nil)
		}
	case FrameFatal:
		if strings.TrimSpace(frame.Code) == "" || strings.TrimSpace(frame.Message) == "" {
			return newProtocolError(ErrorMissingField, frame.Type, frame.RequestID, "fatal requires code and message", nil)
		}
	}
	return nil
}

func hasField(fields map[string]json.RawMessage, name string) bool {
	if fields == nil {
		return false
	}
	_, ok := fields[name]
	return ok
}

func validateCandidate(candidate analysis.DetectionCandidate) error {
	if strings.TrimSpace(candidate.AnalyzerID) == "" {
		return errors.New("candidate requires analyzer_id")
	}
	if candidate.Confidence < 0 || candidate.Confidence > 1 || math.IsNaN(candidate.Confidence) || math.IsInf(candidate.Confidence, 0) {
		return errors.New("candidate confidence must be between zero and one")
	}
	if strings.TrimSpace(candidate.Reason) == "" {
		return errors.New("candidate requires reason")
	}
	return nil
}

func validateDiagnostic(diagnostic analysis.Diagnostic) error {
	if strings.TrimSpace(diagnostic.Code) == "" || strings.TrimSpace(diagnostic.Message) == "" {
		return errors.New("diagnostic requires code and message")
	}
	switch diagnostic.Severity {
	case "info", "warning", "error":
		return nil
	default:
		return fmt.Errorf("diagnostic severity %q is invalid", diagnostic.Severity)
	}
}
