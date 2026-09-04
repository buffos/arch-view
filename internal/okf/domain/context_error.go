package domain

import (
	"context"
	"errors"
)

// ContextOperationError preserves the cause and the canonical timeout status.
func ContextOperationError(err error, operation string) *Error {
	if errors.Is(err, context.DeadlineExceeded) {
		return WrapError("okf_operation_timeout", 504, operation+" exceeded its processing deadline", err)
	}
	return WrapError("okf_operation_cancelled", 409, operation+" was cancelled", err)
}
