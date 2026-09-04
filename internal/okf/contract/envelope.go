package contract

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"

	"github.com/buffo/arch-view/internal/okf/domain"
)

type Meta struct {
	RequestID   string `json:"request_id"`
	OperationID string `json:"operation_id,omitempty"`
	Revision    string `json:"revision,omitempty"`
}

type Success struct {
	Data        any                 `json:"data"`
	Diagnostics []domain.Diagnostic `json:"diagnostics"`
	Meta        Meta                `json:"meta"`
}

type FailureError struct {
	Code        string              `json:"code"`
	Message     string              `json:"message"`
	Details     map[string]any      `json:"details,omitempty"`
	Diagnostics []domain.Diagnostic `json:"diagnostics,omitempty"`
}

type Failure struct {
	Error FailureError `json:"error"`
	Meta  Meta         `json:"meta"`
}

func RequestID(request *http.Request) string {
	if value := request.Header.Get("X-Request-ID"); value != "" {
		return value
	}
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return "req_" + hex.EncodeToString(bytes[:])
	}
	return "req_local"
}

func ErrorResponse(err error, requestID string) (Failure, int) {
	if errors.Is(err, context.Canceled) {
		return Failure{Error: FailureError{Code: "okf_operation_cancelled", Message: "the OKF operation was cancelled", Diagnostics: []domain.Diagnostic{{Code: "okf_operation_cancelled", Severity: "error", Category: "cancellation", Message: "the OKF operation was cancelled"}}}, Meta: Meta{RequestID: requestID}}, http.StatusConflict
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return Failure{Error: FailureError{Code: "okf_operation_timeout", Message: "the OKF operation exceeded its processing deadline", Diagnostics: []domain.Diagnostic{{Code: "okf_operation_timeout", Severity: "error", Category: "cancellation", Message: "the OKF operation exceeded its processing deadline"}}}, Meta: Meta{RequestID: requestID}}, http.StatusGatewayTimeout
	}
	var value *domain.Error
	if errors.As(err, &value) {
		status := value.Status
		if status == 0 {
			status = http.StatusInternalServerError
		}
		return Failure{Error: FailureError{Code: value.Code, Message: value.Message, Details: value.Details, Diagnostics: value.Diagnostics}, Meta: Meta{RequestID: requestID}}, status
	}
	if err == nil {
		return Failure{Error: FailureError{Code: "okf_projection_failed", Message: "an unknown OKF error occurred"}, Meta: Meta{RequestID: requestID}}, http.StatusInternalServerError
	}
	return Failure{Error: FailureError{Code: "okf_projection_failed", Message: err.Error()}, Meta: Meta{RequestID: requestID}}, http.StatusInternalServerError
}
