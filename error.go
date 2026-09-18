package xerr

import (
	"encoding/json"
	"maps"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// Error represents a structured application error.
//
// An Error always carries its full detail in memory — every field below
// — so a single value works for both destinations: pass it to your
// logger directly (via Error(), or by logging Code/Kind/Diagnostics as
// structured fields) and pass the same value to json.Marshal for the
// HTTP response. The two destinations see different things on purpose:
//
//	field        goes to your logs (Error() / accessors)   goes to the client (MarshalJSON)
//	----------   -----------------------------------------  -------------------------------
//	Code         always                                     always — real code if Exposed,
//	                                                         otherwise CodeInternalError
//	Kind         always                                     never (not serialized)
//	Message      always                                     only if Exposed
//	Violations   always                                     only if Exposed
//	Diagnostics  always                                     never — not even when Exposed
//	Err (cause)  always (via Error() / Unwrap())             never
//	Stack        always, if captured (via Stack())           never
//
// Exposed defaults from Kind: KindDomain and KindApplication are safe by
// default, KindInfrastructure and KindUnknown are not — see Kind.Safe.
// WithExpose overrides the default per error when a case genuinely needs
// it. Diagnostics are the one field with no exposure switch at all: they
// exist specifically for data that must never reach a client (internal
// identifiers, operation names, raw connection strings, ...), so treat
// WithDiagnostic as the "this can never leak" bucket and WithMessage /
// WithViolation as the "this is fine to leak, subject to Exposed" bucket.
type Error struct {
	// code is always logged and always present in the client response —
	// as the real Code when Exposed, or CodeInternalError otherwise.
	code Code

	// kind is always logged. Never sent to the client; it only decides
	// the Exposed default.
	kind Kind

	// message is always logged. Sent to the client only when Exposed.
	message string

	// err is the wrapped cause. Always logged (Error()/Unwrap()/errors.As
	// walk it in full). Never sent to the client, regardless of Exposed.
	err error

	// expose is the WithExpose override, nil meaning "use kind.Safe()".
	expose *bool

	// violations is always logged. Sent to the client only when Exposed.
	violations []Violation

	// diagnostics is always logged, and never sent to the client — not
	// even when Exposed. Put anything here that must never leak.
	diagnostics map[DiagnosticKey]string

	// stack is the call stack captured by WithStack or Recover, if any.
	// Always logged (via Stack()), never sent to the client.
	stack []uintptr
}

func (e *Error) Error() string {
	var b strings.Builder

	b.WriteString(e.code.String())
	b.WriteString(" (")
	b.WriteString(e.kind.String())
	b.WriteByte(')')

	if e.message != "" {
		b.WriteString(": ")
		b.WriteString(e.message)
	}

	if len(e.violations) > 0 {
		b.WriteString(" [")
		for i, v := range e.violations {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(v.Field)
			b.WriteByte('=')
			b.WriteString(v.Reason.String())
		}
		b.WriteByte(']')
	}

	if len(e.diagnostics) > 0 {
		keys := make([]string, 0, len(e.diagnostics))
		for k := range e.diagnostics {
			keys = append(keys, string(k))
		}
		sort.Strings(keys)

		b.WriteString(" {")
		for i, k := range keys {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(k)
			b.WriteByte('=')
			b.WriteString(e.diagnostics[DiagnosticKey(k)])
		}
		b.WriteByte('}')
	}

	if e.err != nil {
		b.WriteString(": ")
		b.WriteString(e.err.Error())
	}

	return b.String()
}

func (e *Error) Unwrap() error { return e.err }

// Is reports whether target is an *Error with the same Code. Message,
// Violations, and diagnostics are runtime detail and intentionally
// excluded, so xerr.New(xerr.CodeNotFound) works as a sentinel for
// errors.Is regardless of what detail a concrete instance carries.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.code == t.code
}

// Code returns the machine-readable error code. Safe to log always; safe
// to return to a client always (MarshalJSON substitutes CodeInternalError
// for the real value when the error is not Exposed).
func (e *Error) Code() Code { return e.code }

// Kind returns the error's classification. Log-only: never sent to a
// client, and not part of MarshalJSON's output.
func (e *Error) Kind() Kind { return e.kind }

// Message returns the human-readable message, if any. Safe to log
// always. Only sent to a client when Exposed is true.
func (e *Error) Message() string { return e.message }

// Err returns the wrapped underlying cause, if any. Log-only: include it
// in your logger output (or call Error(), which already does), but never
// serialize it to a client — it commonly holds raw driver/library errors
// (SQL, HTTP client, ...) that can leak internal topology.
func (e *Error) Err() error { return e.err }

// HTTPStatus returns the HTTP status associated with this error's Code.
func (e *Error) HTTPStatus() int {
	return e.code.HTTPStatus()
}

// Exposed reports whether this error's Message and Violations are safe
// to return to a client. It defaults to Kind.Safe() and can be
// overridden per-error with WithExpose.
func (e *Error) Exposed() bool {
	if e.expose != nil {
		return *e.expose
	}
	return e.kind.Safe()
}

// Violations returns a copy of the field-level violations attached to
// this error. Safe to log always. Only sent to a client when Exposed is
// true.
func (e *Error) Violations() []Violation {
	if e.violations == nil {
		return nil
	}
	cp := make([]Violation, len(e.violations))
	copy(cp, e.violations)
	return cp
}

// Diagnostics returns a copy of the internal-only debug context attached
// to this error (e.g. which operation was running, an internal resource
// id). Log-only, unconditionally: unlike Message and Violations,
// diagnostics have no Exposed switch and are never serialized by
// MarshalJSON no matter what — put here anything that must never reach
// a client.
func (e *Error) Diagnostics() map[DiagnosticKey]string {
	if e.diagnostics == nil {
		return nil
	}
	cp := make(map[DiagnosticKey]string, len(e.diagnostics))
	maps.Copy(cp, e.diagnostics)
	return cp
}

// Stack returns the call stack captured by WithStack or Recover, if any,
// formatted one frame per line as "function\n\tfile:line". Returns "" if
// no stack was captured. Log-only: never part of MarshalJSON's output,
// and deliberately excluded from Error()'s compact single-line output —
// call Stack() explicitly (or log the *Error via slog, whose LogValue
// includes it) when you want it.
func (e *Error) Stack() string {
	if len(e.stack) == 0 {
		return ""
	}

	frames := runtime.CallersFrames(e.stack)

	var b strings.Builder
	for {
		frame, more := frames.Next()
		b.WriteString(frame.Function)
		b.WriteString("\n\t")
		b.WriteString(frame.File)
		b.WriteByte(':')
		b.WriteString(strconv.Itoa(frame.Line))
		if more {
			b.WriteByte('\n')
			continue
		}
		break
	}

	return b.String()
}

// MarshalJSON produces the client-safe JSON representation of the error.
// Only three fields are ever eligible to appear: code, message, and
// violations — Kind, Diagnostics, and the wrapped Err are never
// serialized, under any circumstance.
//
// When Exposed is false, the response collapses further, to a bare
// {"code":"INTERNAL_SERVER_ERROR"}: the specific code, message, and any
// wrapped error stay available server-side via Error(), Code(), and
// Diagnostics(), but never reach the client.
func (e *Error) MarshalJSON() ([]byte, error) {
	type response struct {
		Code       Code        `json:"code"`
		Message    string      `json:"message,omitempty"`
		Violations []Violation `json:"violations,omitempty"`
	}

	if !e.Exposed() {
		return json.Marshal(response{Code: CodeInternalError})
	}

	return json.Marshal(response{
		Code:       e.code,
		Message:    e.message,
		Violations: e.violations,
	})
}
