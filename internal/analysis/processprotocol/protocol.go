package processprotocol

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"sort"
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

// Decoder reads one JSON object per line and enforces a bounded line size.
// Use one Decoder for a stream; the package-level ReadFrame helper is intended
// for one-shot reads.
type Decoder struct {
	reader *bufio.Reader
	max    int
}

func NewDecoder(reader io.Reader) *Decoder {
	return NewDecoderWithLimit(reader, DefaultMaxFrameBytes)
}

func NewDecoderWithLimit(reader io.Reader, maxBytes int) *Decoder {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxFrameBytes
	}
	if reader == nil {
		return &Decoder{max: maxBytes}
	}
	if buffered, ok := reader.(*bufio.Reader); ok {
		return &Decoder{reader: buffered, max: maxBytes}
	}
	return &Decoder{reader: bufio.NewReader(reader), max: maxBytes}
}

func (d *Decoder) ReadFrame() (Frame, error) {
	if d == nil || d.reader == nil {
		return Frame{}, io.EOF
	}
	line, err := readBoundedLine(d.reader, d.max)
	if err != nil {
		return Frame{}, err
	}
	if len(bytes.TrimSpace(line)) == 0 {
		return Frame{}, newProtocolError(ErrorMalformedJSON, "", "", "blank lines are not protocol frames", ErrMalformedJSON)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(line, &fields); err != nil || fields == nil {
		if err == nil {
			err = errors.New("frame must be a JSON object")
		}
		return Frame{}, newProtocolError(ErrorMalformedJSON, "", "", "frame must be one JSON object", err)
	}
	var frame Frame
	if err := json.Unmarshal(line, &frame); err != nil {
		return Frame{}, newProtocolError(ErrorMalformedJSON, "", "", "frame fields have invalid JSON types", err)
	}
	if err := validateFrameShape(frame, fields); err != nil {
		return Frame{}, err
	}
	return frame, nil
}

func readBoundedLine(reader *bufio.Reader, maxBytes int) ([]byte, error) {
	var line []byte
	for {
		part, err := reader.ReadSlice('\n')
		if len(part) > 0 {
			if len(line)+len(part) > maxBytes {
				return nil, newProtocolError(ErrorFrameTooLarge, "", "", fmt.Sprintf("line exceeds %d bytes", maxBytes), ErrFrameTooLarge)
			}
			line = append(line, part...)
		}
		switch {
		case err == nil:
			return line, nil
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case errors.Is(err, io.EOF):
			if len(line) == 0 {
				return nil, io.EOF
			}
			return line, nil
		default:
			return nil, err
		}
	}
}

func ReadFrame(reader io.Reader) (Frame, error) {
	return NewDecoder(reader).ReadFrame()
}

func ReadFrameWithLimit(reader io.Reader, maxBytes int) (Frame, error) {
	return NewDecoderWithLimit(reader, maxBytes).ReadFrame()
}

// Encoder writes protocol frames with exactly one trailing newline per frame.
type Encoder struct {
	writer io.Writer
	max    int
}

func NewEncoder(writer io.Writer) *Encoder {
	return NewEncoderWithLimit(writer, DefaultMaxFrameBytes)
}

func NewEncoderWithLimit(writer io.Writer, maxBytes int) *Encoder {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxFrameBytes
	}
	return &Encoder{writer: writer, max: maxBytes}
}

func (e *Encoder) WriteFrame(frame Frame) error {
	if e == nil {
		return errors.New("protocol encoder is nil")
	}
	return writeFrame(e.writer, frame, e.max)
}

func WriteFrame(writer io.Writer, frame Frame) error {
	return writeFrame(writer, frame, DefaultMaxFrameBytes)
}

func WriteFrameWithLimit(writer io.Writer, frame Frame, maxBytes int) error {
	return writeFrame(writer, frame, maxBytes)
}

func writeFrame(writer io.Writer, frame Frame, maxBytes int) error {
	if writer == nil {
		return errors.New("protocol writer is nil")
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxFrameBytes
	}
	if err := ValidateFrame(frame); err != nil {
		return err
	}
	data, err := json.Marshal(frame)
	if err != nil {
		return newProtocolError(ErrorInvalidPayload, frame.Type, frame.RequestID, "frame could not be encoded as JSON", err)
	}
	if len(data)+1 > maxBytes {
		return newProtocolError(ErrorFrameTooLarge, frame.Type, frame.RequestID, fmt.Sprintf("line exceeds %d bytes", maxBytes), ErrFrameTooLarge)
	}
	data = append(data, '\n')
	for len(data) > 0 {
		written, writeErr := writer.Write(data)
		if written > 0 {
			data = data[written:]
		}
		if writeErr != nil {
			return writeErr
		}
		if written == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}

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

// SessionValidator enforces the stateful one-process/one-request protocol
// rules. It does not replace analysis.ValidateAnalysisResult; callers should
// run that validator on Result() after streamed diagnostics are merged.
type SessionValidator struct {
	operation        FrameType
	requestID        string
	expectedManifest analysis.Manifest

	helloReceived       bool
	hostRequestReceived bool
	cancelSent          bool
	payloadReceived     bool
	candidate           *analysis.DetectionCandidate
	result              *analysis.AnalysisResult
	diagnostics         []analysis.Diagnostic
	terminal            *Frame
}

// ProtocolSession is the domain-facing name for SessionValidator.
type ProtocolSession = SessionValidator

func NewSessionValidator(operation FrameType, requestID string, expectedManifest analysis.Manifest) (*SessionValidator, error) {
	if operation != FrameDetect && operation != FrameAnalyze {
		return nil, newProtocolError(ErrorUnexpectedFrame, operation, requestID, "session operation must be detect or analyze", ErrUnexpectedFrame)
	}
	if !validRequestID(requestID) {
		return nil, newProtocolError(ErrorInvalidRequestID, operation, requestID, "session requires a valid request ID", ErrInvalidRequestID)
	}
	if err := analysis.ValidateManifest(expectedManifest); err != nil {
		return nil, newProtocolError(ErrorInvalidPayload, operation, requestID, "expected session manifest is invalid", err)
	}
	return &SessionValidator{
		operation:        operation,
		requestID:        requestID,
		expectedManifest: expectedManifest,
		diagnostics:      []analysis.Diagnostic{},
	}, nil
}

func NewProtocolSession(operation FrameType, requestID string, expectedManifest analysis.Manifest) (*ProtocolSession, error) {
	return NewSessionValidator(operation, requestID, expectedManifest)
}

func (s *SessionValidator) AcceptHostFrame(frame Frame) error {
	if s == nil {
		return newProtocolError(ErrorProtocolState, frame.Type, frame.RequestID, "session validator is nil", ErrProtocolState)
	}
	if s.terminal != nil {
		return newProtocolError(ErrorLateFrame, frame.Type, frame.RequestID, "host frame arrived after terminal output", ErrLateFrame)
	}
	if err := ValidateFrame(frame); err != nil {
		return err
	}
	if !s.helloReceived {
		return newProtocolError(ErrorProtocolState, frame.Type, frame.RequestID, "host request cannot precede the plugin hello", ErrProtocolState)
	}
	if frame.RequestID != s.requestID {
		return newProtocolError(ErrorRequestIDMismatch, frame.Type, frame.RequestID, "host frame request ID does not match session", ErrRequestIDMismatch)
	}
	switch frame.Type {
	case s.operation:
		if s.hostRequestReceived {
			return newProtocolError(ErrorProtocolState, frame.Type, frame.RequestID, "session accepts one operation request", ErrProtocolState)
		}
		s.hostRequestReceived = true
		return nil
	case FrameCancel:
		if !s.hostRequestReceived {
			return newProtocolError(ErrorProtocolState, frame.Type, frame.RequestID, "cancel requires an active operation request", ErrProtocolState)
		}
		if s.cancelSent {
			return newProtocolError(ErrorProtocolState, frame.Type, frame.RequestID, "session accepts one cancel request", ErrProtocolState)
		}
		s.cancelSent = true
		return nil
	default:
		return newProtocolError(ErrorUnexpectedFrame, frame.Type, frame.RequestID, "frame is not a host-to-plugin request", ErrUnexpectedFrame)
	}
}

func (s *SessionValidator) AcceptPluginFrame(frame Frame) error {
	if s == nil {
		return newProtocolError(ErrorProtocolState, frame.Type, frame.RequestID, "session validator is nil", ErrProtocolState)
	}
	if s.terminal != nil {
		if frame.Type == FrameDone || frame.Type == FrameFatal {
			return newProtocolError(ErrorDuplicateTerminal, frame.Type, frame.RequestID, "session already has a terminal frame", ErrDuplicateTerminal)
		}
		return newProtocolError(ErrorLateFrame, frame.Type, frame.RequestID, "frame arrived after terminal output", ErrLateFrame)
	}
	if err := ValidateFrame(frame); err != nil {
		return err
	}
	if !s.helloReceived {
		if frame.Type != FrameHello {
			return newProtocolError(ErrorProtocolState, frame.Type, frame.RequestID, "hello must be the first plugin frame", ErrProtocolState)
		}
		if !sameManifest(*frame.Manifest, s.expectedManifest) {
			return newProtocolError(ErrorInvalidPayload, frame.Type, frame.RequestID, "hello manifest does not match the descriptor manifest", ErrInvalidPayload)
		}
		s.helloReceived = true
		return nil
	}
	if frame.Type == FrameHello {
		return newProtocolError(ErrorProtocolState, frame.Type, frame.RequestID, "hello may occur only once as the first frame", ErrProtocolState)
	}
	if !s.hostRequestReceived {
		return newProtocolError(ErrorProtocolState, frame.Type, frame.RequestID, "plugin output arrived before the host request", ErrProtocolState)
	}
	if frame.RequestID != s.requestID {
		return newProtocolError(ErrorRequestIDMismatch, frame.Type, frame.RequestID, "plugin frame request ID does not match session", ErrRequestIDMismatch)
	}
	if s.cancelSent && (frame.Type == FrameCandidate || frame.Type == FrameResult) {
		return newProtocolError(ErrorProtocolState, frame.Type, frame.RequestID, "session cannot accept a result payload after cancellation", ErrProtocolState)
	}

	switch frame.Type {
	case FrameDiagnostic:
		if s.payloadReceived {
			return newProtocolError(ErrorProtocolState, frame.Type, frame.RequestID, "diagnostics must precede the result payload", ErrProtocolState)
		}
		s.diagnostics = append(s.diagnostics, *frame.Diagnostic)
		return nil
	case FrameCandidate:
		if s.operation != FrameDetect {
			return newProtocolError(ErrorUnexpectedFrame, frame.Type, frame.RequestID, "candidate is valid only for detect", ErrUnexpectedFrame)
		}
		if s.payloadReceived {
			return newProtocolError(ErrorDuplicatePayload, frame.Type, frame.RequestID, "detect accepts one candidate", ErrDuplicatePayload)
		}
		if frame.Candidate.AnalyzerID != s.expectedManifest.ID {
			return newProtocolError(ErrorInvalidPayload, frame.Type, frame.RequestID, "candidate analyzer_id does not match the hello manifest", ErrInvalidPayload)
		}
		candidate := *frame.Candidate
		candidate.MatchedMarkers = append([]string(nil), candidate.MatchedMarkers...)
		s.candidate = &candidate
		s.payloadReceived = true
		return nil
	case FrameResult:
		if s.operation != FrameAnalyze {
			return newProtocolError(ErrorUnexpectedFrame, frame.Type, frame.RequestID, "result is valid only for analyze", ErrUnexpectedFrame)
		}
		if s.payloadReceived {
			return newProtocolError(ErrorDuplicatePayload, frame.Type, frame.RequestID, "analyze accepts one result", ErrDuplicatePayload)
		}
		result := *frame.Result
		s.result = &result
		s.payloadReceived = true
		return nil
	case FrameDone:
		if !s.payloadReceived {
			return newProtocolError(ErrorProtocolState, frame.Type, frame.RequestID, "done requires one candidate or result payload", ErrProtocolState)
		}
		if s.operation == FrameDetect && frame.Status != analysis.StatusComplete {
			return newProtocolError(ErrorProtocolState, frame.Type, frame.RequestID, "detect done status must be complete", ErrProtocolState)
		}
		if s.result != nil && frame.Status != s.result.Status {
			return newProtocolError(ErrorProtocolState, frame.Type, frame.RequestID, "done status must agree with result status", ErrProtocolState)
		}
		terminal := frame
		s.terminal = &terminal
		return nil
	case FrameFatal:
		if s.payloadReceived {
			return newProtocolError(ErrorProtocolState, frame.Type, frame.RequestID, "fatal cannot follow a candidate or result payload", ErrProtocolState)
		}
		terminal := frame
		s.terminal = &terminal
		return nil
	default:
		return newProtocolError(ErrorUnexpectedFrame, frame.Type, frame.RequestID, "frame is not valid plugin output", ErrUnexpectedFrame)
	}
}

func sameManifest(left, right analysis.Manifest) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}

func (s *SessionValidator) Complete() bool {
	return s != nil && s.helloReceived && s.hostRequestReceived && s.terminal != nil
}

func (s *SessionValidator) HelloReceived() bool {
	return s != nil && s.helloReceived
}

func (s *SessionValidator) RequestID() string {
	if s == nil {
		return ""
	}
	return s.requestID
}

func (s *SessionValidator) Operation() FrameType {
	if s == nil {
		return ""
	}
	return s.operation
}

func (s *SessionValidator) Candidate() (analysis.DetectionCandidate, bool) {
	if s == nil || s.candidate == nil {
		return analysis.DetectionCandidate{}, false
	}
	candidate := *s.candidate
	candidate.MatchedMarkers = append([]string(nil), candidate.MatchedMarkers...)
	return candidate, true
}

func (s *SessionValidator) Result() (analysis.AnalysisResult, bool) {
	if s == nil || s.result == nil {
		return analysis.AnalysisResult{}, false
	}
	return *s.result, true
}

func (s *SessionValidator) Diagnostics() []analysis.Diagnostic {
	if s == nil {
		return nil
	}
	return append([]analysis.Diagnostic(nil), s.diagnostics...)
}

// MergeDiagnostics appends streamed diagnostics to a canonical result and
// removes exact duplicates in deterministic JSON order. It deliberately does
// not validate the result; analysis.ValidateAnalysisResult remains the
// authoritative semantic validator after this merge.
func MergeDiagnostics(result analysis.AnalysisResult, streamed []analysis.Diagnostic) analysis.AnalysisResult {
	all := make([]analysis.Diagnostic, 0, len(result.Diagnostics)+len(streamed))
	all = append(all, result.Diagnostics...)
	all = append(all, streamed...)
	unique := make(map[string]analysis.Diagnostic, len(all))
	for index, diagnostic := range all {
		encoded, err := json.Marshal(diagnostic)
		if err != nil {
			unique[fmt.Sprintf("unserializable:%d:%T", index, diagnostic)] = diagnostic
			continue
		}
		unique[string(encoded)] = diagnostic
	}
	keys := make([]string, 0, len(unique))
	for key := range unique {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result.Diagnostics = make([]analysis.Diagnostic, 0, len(keys))
	for _, key := range keys {
		result.Diagnostics = append(result.Diagnostics, unique[key])
	}
	return result
}

func (s *SessionValidator) ResultWithDiagnostics() (analysis.AnalysisResult, bool) {
	result, ok := s.Result()
	if !ok {
		return analysis.AnalysisResult{}, false
	}
	return MergeDiagnostics(result, s.Diagnostics()), true
}

func (s *SessionValidator) Terminal() (Frame, bool) {
	if s == nil || s.terminal == nil {
		return Frame{}, false
	}
	return *s.terminal, true
}
