package xerr

import (
	"encoding/json"
	"fmt"
)

// Error represents a structured application error.
type Error struct {
	code    Code
	message string
	status  int
	err     error
	meta    map[string]any
}

func (e *Error) Error() string {
	switch {
	case e.message != "":
		return fmt.Sprintf("%s: %s", e.code, e.message)
	case e.err != nil:
		return fmt.Sprintf("%s: %s", e.code, e.err.Error())
	default:
		return string(e.code)
	}
}

func (e *Error) Unwrap() error { return e.err }

func (e *Error) Code() Code           { return e.code }
func (e *Error) Message() string      { return e.message }
func (e *Error) Meta() map[string]any { return e.meta }
func (e *Error) Status() int          { return e.status }
func (e *Error) Err() error           { return e.err }

func (e *Error) HTTPStatus() int {
	if e.status != 0 {
		return e.status
	}
	return e.code.HTTPStatus()
}

func (e *Error) MarshalJSON() ([]byte, error) {
	type response struct {
		Code    Code           `json:"code"`
		Message string         `json:"message,omitempty"`
		Meta    map[string]any `json:"meta,omitempty"`
	}

	msg := e.message
	if msg == "" && e.err != nil {
		msg = e.err.Error()
	}

	return json.Marshal(response{
		Code:    e.code,
		Message: msg,
		Meta:    e.meta,
	})
}
