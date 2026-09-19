package xerr

import "errors"

// New creates a new structured xerr.Error. Its Kind defaults to the
// Code's registered Kind (see RegisterCode) and can be overridden with
// WithKind.
func New(code Code, opts ...ErrorOption) *Error {
	e := &Error{
		code: code,
		kind: code.Kind(),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Wrap converts a raw error into an xerr.Error with a given code,
// preserving err for Unwrap/errors.Is/errors.As and log output. Its Kind
// defaults to the Code's registered Kind (see RegisterCode) and can be
// overridden with WithKind. Returns nil if err is nil.
func Wrap(err error, code Code, opts ...ErrorOption) *Error {
	if err == nil {
		return nil
	}

	e := &Error{
		code: code,
		kind: code.Kind(),
		err:  err,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// FromError extracts an *Error from err's chain, if present. It is a
// thin convenience wrapper around errors.As, handy in HTTP middleware
// that needs to branch on whether an error is already an xerr.Error.
func FromError(err error) (*Error, bool) {
	var e *Error
	ok := errors.As(err, &e)
	return e, ok
}
