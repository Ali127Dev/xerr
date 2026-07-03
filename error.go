package xerr

import (
	"encoding/json"
	"strings"
)

// Error represents a structured application error.
type Error struct {
	code    Code
	message string
	err     error
	meta    map[string]ErrorReason
}

func (e *Error) Error() string {
	var b strings.Builder

	b.WriteString(e.code.String())

	if e.message != "" {
		b.WriteString(": ")
		b.WriteString(e.message)
	}

	if len(e.meta) > 0 {
		b.WriteString(" [")

		i := 0
		for field, reason := range e.meta {
			if i > 0 {
				b.WriteString(", ")
			}

			b.WriteString(field)
			b.WriteByte('=')
			b.WriteString(reason.String())

			i++
		}

		b.WriteByte(']')
	}

	if e.err != nil {
		b.WriteString(": ")
		b.WriteString(e.err.Error())
	}

	return b.String()
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
		Code Code                   `json:"code"`
		Meta map[string]ErrorReason `json:"meta,omitempty"`
	}

	return json.Marshal(response{
		Code: e.code,
		Meta: e.meta,
	})
}
