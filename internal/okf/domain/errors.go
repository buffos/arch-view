package domain

import "fmt"

type Error struct {
	Code        string
	Message     string
	Status      int
	Details     map[string]any
	Diagnostics []Diagnostic
	cause       error
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	return err.Message
}

func (err *Error) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.cause
}

func NewError(code string, status int, message string, details map[string]any) *Error {
	return &Error{Code: code, Status: status, Message: message, Details: details}
}

func DiagnosticFor(code, category, message string) Diagnostic {
	return Diagnostic{Code: code, Severity: "error", Category: category, Message: message}
}

func (err *Error) WithDiagnostics(diagnostics ...Diagnostic) *Error {
	err.Diagnostics = append([]Diagnostic(nil), diagnostics...)
	return err
}

func WrapError(code string, status int, message string, cause error) *Error {
	if cause == nil {
		return NewError(code, status, message, nil)
	}
	value := NewError(code, status, fmt.Sprintf("%s: %v", message, cause), nil)
	value.cause = cause
	return value
}
