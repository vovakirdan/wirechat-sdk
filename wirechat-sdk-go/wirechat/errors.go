package wirechat

import (
	"errors"
	"fmt"
)

// ErrorCode represents a categorized error type.
type ErrorCode int

const (
	// Protocol Errors (from server error responses)
	ErrorUnknown ErrorCode = iota
	ErrorUnsupportedVersion
	ErrorUnauthorized
	ErrorInvalidMessage
	ErrorBadRequest
	ErrorRoomNotFound
	ErrorAlreadyJoined
	ErrorNotInRoom
	ErrorAccessDenied
	ErrorRateLimited
	ErrorInternalServer

	// Client-side Errors
	ErrorConnection
	ErrorDisconnected
	ErrorTimeout
	ErrorInvalidConfig
	ErrorNotConnected
	ErrorSerialization
)

// String returns the string representation of an ErrorCode.
func (e ErrorCode) String() string {
	switch e {
	case ErrorUnknown:
		return "unknown"
	case ErrorUnsupportedVersion:
		return "unsupported_version"
	case ErrorUnauthorized:
		return "unauthorized"
	case ErrorInvalidMessage:
		return "invalid_message"
	case ErrorBadRequest:
		return "bad_request"
	case ErrorRoomNotFound:
		return "room_not_found"
	case ErrorAlreadyJoined:
		return "already_joined"
	case ErrorNotInRoom:
		return "not_in_room"
	case ErrorAccessDenied:
		return "access_denied"
	case ErrorRateLimited:
		return "rate_limited"
	case ErrorInternalServer:
		return "internal_error"
	case ErrorConnection:
		return "connection_error"
	case ErrorDisconnected:
		return "disconnected"
	case ErrorTimeout:
		return "timeout"
	case ErrorInvalidConfig:
		return "invalid_config"
	case ErrorNotConnected:
		return "not_connected"
	case ErrorSerialization:
		return "serialization_error"
	default:
		return fmt.Sprintf("unknown_code_%d", e)
	}
}

// ParseErrorCode converts a protocol error code string to ErrorCode.
func ParseErrorCode(code string) ErrorCode {
	switch code {
	case "unsupported_version":
		return ErrorUnsupportedVersion
	case "unauthorized":
		return ErrorUnauthorized
	case "invalid_message":
		return ErrorInvalidMessage
	case "bad_request":
		return ErrorBadRequest
	case "room_not_found":
		return ErrorRoomNotFound
	case "already_joined":
		return ErrorAlreadyJoined
	case "not_in_room":
		return ErrorNotInRoom
	case "access_denied":
		return ErrorAccessDenied
	case "rate_limited":
		return ErrorRateLimited
	case "internal_error":
		return ErrorInternalServer
	default:
		return ErrorUnknown
	}
}

// WirechatError is a structured error with code and context.
type WirechatError struct {
	Code    ErrorCode
	Message string
	Wrapped error
}

// Error implements the error interface.
func (e *WirechatError) Error() string {
	if e.Wrapped != nil {
		return fmt.Sprintf("%s: %s (wrapped: %v)", e.Code, e.Message, e.Wrapped)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error for errors.Unwrap support.
func (e *WirechatError) Unwrap() error {
	return e.Wrapped
}

// Is implements errors.Is interface for error comparison.
func (e *WirechatError) Is(target error) bool {
	t, ok := target.(*WirechatError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// NewError creates a new WirechatError with the given code and message.
func NewError(code ErrorCode, message string) *WirechatError {
	return &WirechatError{
		Code:    code,
		Message: message,
	}
}

// WrapError wraps an existing error with a WirechatError.
func WrapError(code ErrorCode, message string, err error) *WirechatError {
	return &WirechatError{
		Code:    code,
		Message: message,
		Wrapped: err,
	}
}

// FromProtocolError converts a protocol Error to WirechatError.
func FromProtocolError(e *Error) *WirechatError {
	if e == nil {
		return nil
	}
	return &WirechatError{
		Code:    ParseErrorCode(e.Code),
		Message: e.Msg,
	}
}

// IsProtocolError checks if an error is a protocol error (from server).
func IsProtocolError(err error) bool {
	if err == nil {
		return false
	}
	var we *WirechatError
	if !errors.As(err, &we) {
		return false
	}
	// Protocol errors are those that come from the server
	return we.Code >= ErrorUnsupportedVersion && we.Code <= ErrorInternalServer
}

// IsConnectionError checks if an error is a connection-related error.
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}
	var we *WirechatError
	if !errors.As(err, &we) {
		return false
	}
	return we.Code == ErrorConnection || we.Code == ErrorDisconnected || we.Code == ErrorTimeout
}
