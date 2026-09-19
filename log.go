package xerr

import "log/slog"

// LogValue implements slog.LogValuer. Passing an *Error to log/slog
// renders every field your logger should see — Code, Kind, Message,
// Params, Violations, Diagnostics, the wrapped Err, and Stack if
// captured — as structured attributes, without hand-writing each one at
// the call site:
//
//	slog.Error("request failed", "err", xerr.Wrap(dbErr, xerr.CodeDatabaseError))
//
// This is deliberately the log-only view: unlike MarshalJSON, it never
// consults Exposed and always includes Kind, Diagnostics, and Err — the
// exact fields the client boundary hides — because this path is for
// your logger, never for a client response.
func (e *Error) LogValue() slog.Value {
	attrs := make([]slog.Attr, 0, 8)

	attrs = append(attrs,
		slog.String("code", e.code.String()),
		slog.String("kind", e.kind.String()),
	)

	if e.message != "" {
		attrs = append(attrs, slog.String("message", e.message))
	}

	if len(e.params) > 0 {
		attrs = append(attrs, slog.Any("params", e.params))
	}

	if len(e.violations) > 0 {
		attrs = append(attrs, slog.Any("violations", e.violations))
	}

	if len(e.diagnostics) > 0 {
		attrs = append(attrs, slog.Any("diagnostics", e.diagnostics))
	}

	if e.err != nil {
		attrs = append(attrs, slog.String("cause", e.err.Error()))
	}

	if len(e.stack) > 0 {
		attrs = append(attrs, slog.String("stack", e.Stack()))
	}

	return slog.GroupValue(attrs...)
}
