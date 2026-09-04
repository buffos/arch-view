package contract

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestErrorResponsePreservesCanonicalContextOutcomes(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		err    error
		code   string
		status int
	}{
		{name: "cancelled", err: context.Canceled, code: "okf_operation_cancelled", status: http.StatusConflict},
		{name: "wrapped timeout", err: domain.WrapError("okf_configuration_invalid", 500, "configuration failed", context.DeadlineExceeded), code: "okf_operation_timeout", status: http.StatusGatewayTimeout},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			failure, status := ErrorResponse(testCase.err, "request-1")
			if status != testCase.status || failure.Error.Code != testCase.code || failure.Meta.RequestID != "request-1" {
				t.Fatalf("failure=%#v status=%d", failure, status)
			}
		})
	}
	if _, status := ErrorResponse(errors.New("failed"), "request-2"); status != http.StatusInternalServerError {
		t.Fatalf("unknown error status = %d", status)
	}
}

func TestErrorResponsePreservesPersistenceCodeAndDiagnosticContext(t *testing.T) {
	details := map[string]any{"profile_id": "project:working", "expected_revision": "old", "actual_revision": "new"}
	diagnostics := []domain.Diagnostic{{
		Code: "okf_configuration_write_failed", Severity: "error", Category: "persistence",
		ProfileID: "project:working", OperationID: "save-1", Message: "Cannot replace configuration",
		Recovery: "Close the application holding the file and retry.",
	}}
	for _, scenario := range []struct {
		code   string
		status int
	}{
		{"okf_configuration_write_failed", http.StatusInternalServerError},
		{"okf_revision_conflict", http.StatusConflict},
		{"okf_idempotency_conflict", http.StatusConflict},
	} {
		t.Run(scenario.code, func(t *testing.T) {
			err := domain.NewError(scenario.code, scenario.status, "Configuration command failed", details).WithDiagnostics(diagnostics...)
			failure, status := ErrorResponse(err, "request-save-1")
			if status != scenario.status || failure.Error.Code != scenario.code || failure.Error.Message != err.Message {
				t.Fatalf("persistence error was reclassified: %+v status=%d", failure, status)
			}
			if failure.Meta.RequestID != "request-save-1" || !reflect.DeepEqual(failure.Error.Details, details) || !reflect.DeepEqual(failure.Error.Diagnostics, diagnostics) {
				t.Fatalf("persistence context lost: %+v", failure)
			}
		})
	}
}
