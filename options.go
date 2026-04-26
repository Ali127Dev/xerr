package xerr

// ErrorOption is a functional constructor modifier.
type ErrorOption func(*Error)

// WithMessage sets a human-readable message.
func WithMessage(msg string) ErrorOption {
	return func(e *Error) {
		e.message = msg
	}
}

// WithStatus overrides the HTTP status.
func WithStatus(status int) ErrorOption {
	return func(e *Error) {
		e.status = status
	}
}

// WithErr sets the wrapped underlying error.
func WithErr(err error) ErrorOption {
	return func(e *Error) {
		e.err = err
	}
}

// WithMeta adds a metadata entry.
func WithMeta(key string, value any) ErrorOption {
	return func(e *Error) {
		if e.meta == nil {
			e.meta = make(map[string]any)
		}
		e.meta[key] = value
	}
}
