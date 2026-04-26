package xerr

// New creates a new structured xerr.Error.
func New(code Code, opts ...ErrorOption) *Error {
	err := &Error{
		code: code,
		meta: make(map[string]any),
	}

	for _, opt := range opts {
		opt(err)
	}
	return err
}

// Wrap converts a raw error into an xerr.Error with a given code.
func Wrap(err error, code Code, opts ...ErrorOption) *Error {
	if err == nil {
		return nil
	}

	e := &Error{
		code: code,
		err:  err,
		meta: make(map[string]any),
	}

	for _, opt := range opts {
		opt(e)
	}
	return e
}
