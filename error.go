package xerr

import (
	"encoding/json"
)

// Error represents a structured application error.
type Error struct {
	code    Code
	message string
	err     error
	meta    map[string]any
}

func (e *Error) Error() string {
	switch {
	case e.message != "":
		return e.message
	default:
		return string(e.code)
	}
}

func (e *Error) Unwrap() error { return e.err }

func (e *Error) Code() Code           { return e.code }
func (e *Error) Message() string      { return e.message }
func (e *Error) Meta() map[string]any { return e.meta }
func (e *Error) Err() error           { return e.err }

func (e *Error) HTTPStatus() int {
	return e.code.HTTPStatus()
}

func (e *Error) MarshalJSON() ([]byte, error) {
	type response struct {
		Code    Code           `json:"code"`
		Message string         `json:"message,omitempty"`
		Meta    map[string]any `json:"meta,omitempty"`
	}

	return json.Marshal(response{
		Code:    e.code,
		Message: e.message,
		Meta:    e.meta,
	})
}
