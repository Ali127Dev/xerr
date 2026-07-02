package xerr

import (
	"encoding/json"
)

// Error represents a structured application error.
type Error struct {
	code    Code
	message string
	err     error
	meta    map[string]ErrorReason
}

func (e *Error) Error() string {
	if e.message != "" {
		return e.message
	}
	return string(e.code)
}

func (e *Error) Unwrap() error { return e.err }

func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	if e.code != t.code {
		return false
	}
	if len(e.meta) != len(t.meta) {
		return false
	}
	for k, v := range t.meta {
		if e.meta[k] != v {
			return false
		}
	}
	return true
}

func (e *Error) Code() Code      { return e.code }
func (e *Error) Message() string { return e.message }
func (e *Error) Err() error      { return e.err }
func (e *Error) Meta() map[string]ErrorReason {
	if e.meta == nil {
		return nil
	}
	cp := make(map[string]ErrorReason, len(e.meta))
	for k, v := range e.meta {
		cp[k] = v
	}
	return cp
}

func (e *Error) HTTPStatus() int {
	return e.code.HTTPStatus()
}

func (e *Error) MarshalJSON() ([]byte, error) {
	type response struct {
		Code    Code                   `json:"code"`
		Message string                 `json:"message,omitempty"`
		Meta    map[string]ErrorReason `json:"meta,omitempty"`
	}

	return json.Marshal(response{
		Code:    e.code,
		Message: e.Error(),
		Meta:    e.meta,
	})
}
