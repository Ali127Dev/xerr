package xerr

import "fmt"

type Error struct {
	Code    Code           `json:"code"`
	Message string         `json:"message"`
	Status  int            `json:"-"`
	Err     error          `json:"-"`
	Meta    map[string]any `json:"meta,omitempty"`
}

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %s", e.Code, e.Err.Error())
	}
	return fmt.Sprintf("unknown error with code: %s", e.Code)
}

func (e *Error) Unwrap() error {
	return e.Err
}

func New(code Code, opts ...ErrorOption) *Error {
	err := &Error{
		Code: code,
		Meta: make(map[string]any),
	}

	for _, opt := range opts {
		opt(err)
	}
	return err
}

type ErrorOption func(*Error)

func WithMessage(msg string) ErrorOption {
	return func(e *Error) {
		e.Message = msg
	}
}

func WithStatus(status int) ErrorOption {
	return func(e *Error) {
		e.Status = status
	}
}

func WithMeta(key string, value any) ErrorOption {
	return func(e *Error) {
		if e.Meta == nil {
			e.Meta = make(map[string]any)
		}
		e.Meta[key] = value
	}
}
