package xerr

// ErrorOption is a functional constructor modifier.
type ErrorOption func(*Error)

// WithMessage sets a ready-to-display, human-readable message. Use this
// when the backend should own the exact wording the client shows.
// Independent of WithViolation — set either, both, or neither.
func WithMessage(msg string) ErrorOption {
	return func(e *Error) {
		e.message = msg
	}
}

// WithErr attaches the underlying error being wrapped, preserving it for
// Unwrap/errors.Is/errors.As and for Error()'s log output. It is never
// exposed via MarshalJSON.
func WithErr(err error) ErrorOption {
	return func(e *Error) {
		e.err = err
	}
}

// WithKind overrides the Kind the error would otherwise inherit from its
// Code (see CodesKind), and with it, the default answer to Exposed.
func WithKind(k Kind) ErrorOption {
	return func(e *Error) {
		e.kind = k
	}
}

// WithExpose forces whether Message and Violations are safe to return to
// a client, overriding the Kind-based default. Use this for the rare
// exception: an infrastructure error that is actually safe to describe,
// or a domain error that happens to carry sensitive detail.
func WithExpose(expose bool) ErrorOption {
	return func(e *Error) {
		e.expose = &expose
	}
}

// WithViolation adds a field-level violation: a stable, translatable
// Reason plus optional Params for the dynamic values a message template
// needs (e.g. P("min", 8)). Use this when the client should build its
// own localized message. Independent of WithMessage — set either, both,
// or neither. Can be called multiple times to report several violations.
func WithViolation(field string, reason ErrorReason, params ...Param) ErrorOption {
	return func(e *Error) {
		v := Violation{Field: field, Reason: reason}
		if len(params) > 0 {
			v.Params = make(map[string]any, len(params))
			for _, p := range params {
				v.Params[p.Key] = p.Value
			}
		}
		e.violations = append(e.violations, v)
	}
}

// WithViolations appends a batch of already-built violations in one
// call — handy when a third-party validator (e.g. go-playground/validator)
// already produced its own list and you're translating it into
// xerr.Violation instead of building each one by hand with WithViolation.
// Combines with WithViolation; both are additive.
func WithViolations(vs ...Violation) ErrorOption {
	return func(e *Error) {
		e.violations = append(e.violations, vs...)
	}
}

// WithDiagnostic attaches internal-only debug context (e.g. the
// operation being performed, or a resource identifier). Diagnostics
// appear in Error()'s log output but are never exposed via MarshalJSON.
func WithDiagnostic(key DiagnosticKey, value string) ErrorOption {
	return func(e *Error) {
		if e.diagnostics == nil {
			e.diagnostics = make(map[DiagnosticKey]string)
		}
		e.diagnostics[key] = value
	}
}
