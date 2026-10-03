// Package host contains the typed desktop service boundary. Business modules
// remain usable without a Wails application or event loop.
package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

type Error struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Operation string `json:"operation,omitempty"`
	cause     error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s", e.Operation, e.Message)
	}
	return e.Message
}
func (e *Error) Unwrap() error     { return e.cause }
func invalid(message string) error { return &Error{Code: "INVALID_ARGUMENT", Message: message} }
func failure(operation string, err error) error {
	if err == nil {
		return nil
	}
	var hostError *Error
	if errors.As(err, &hostError) {
		return err
	}
	code, message := "OPERATION_FAILED", "操作失败，请重试或查看应用日志"
	if errors.Is(err, context.Canceled) {
		code, message = "CANCELLED", "操作已取消"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		code, message = "TIMEOUT", "操作超时，请重试"
	}
	return &Error{Code: code, Message: message, Operation: operation, cause: err}
}

// MarshalError is also applied to Wails argument decoding failures. Those occur
// before a service method executes and otherwise expose an untyped transport error.
func MarshalError(err error) []byte {
	var e *Error
	if !errors.As(err, &e) {
		e = &Error{Code: "INVALID_ARGUMENT", Message: "调用参数格式不正确"}
	}
	data, _ := json.Marshal(e)
	return data
}
