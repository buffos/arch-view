package processprotocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/buffo/arch-view/internal/analysis"
)

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
