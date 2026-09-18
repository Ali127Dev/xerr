package xerr

import (
	"fmt"
	"strings"
)

// staticViolationMessages holds the reasons whose fallback sentence
// needs nothing beyond the field name. Reasons whose sentence depends on
// Params (a bound, an allowed set, ...) are handled separately in
// DefaultMessage, since they each need their own param lookup.
var staticViolationMessages = map[ErrorReason]string{
	ErrorReasonRequired:      "%s is required",
	ErrorReasonInvalidFormat: "%s has an invalid format",
	ErrorReasonMismatch:      "%s does not match",
	ErrorReasonAlreadyExists: "%s already exists",
	ErrorReasonNotFound:      "%s was not found",
	ErrorReasonCorrupted:     "%s is corrupted",
	ErrorReasonExpired:       "%s has expired",
}

// DefaultMessage renders a plain-English fallback sentence for a
// Violation, substituting Params where its template needs them.
//
// This is not a localization system: there is no catalog, no locale
// negotiation, nothing pluggable. It exists for the contexts that have
// no frontend to own translation — a CLI tool, a server log meant for a
// human, a quick prototype. Wherever there is a real client, prefer
// letting it build its own copy from Reason + Params (that's what the
// enum is for); reach for DefaultMessage only as a fallback.
func (v Violation) DefaultMessage() string {
	switch v.Reason {
	case ErrorReasonInvalidValue:
		if allowed, ok := v.Params["allowed"]; ok {
			return fmt.Sprintf("%s must be one of %v", v.Field, allowed)
		}
		return fmt.Sprintf("%s has an invalid value", v.Field)
	case ErrorReasonTooShort:
		if minLen, ok := v.Params["min"]; ok {
			return fmt.Sprintf("%s must be at least %v characters", v.Field, minLen)
		}
		return fmt.Sprintf("%s is too short", v.Field)
	case ErrorReasonTooLong:
		if maxLen, ok := v.Params["max"]; ok {
			return fmt.Sprintf("%s must be at most %v characters", v.Field, maxLen)
		}
		return fmt.Sprintf("%s is too long", v.Field)
	case ErrorReasonTooSmall:
		if minVal, ok := v.Params["min"]; ok {
			return fmt.Sprintf("%s must be at least %v", v.Field, minVal)
		}
		return fmt.Sprintf("%s is too small", v.Field)
	case ErrorReasonTooLarge:
		if maxVal, ok := v.Params["max"]; ok {
			return fmt.Sprintf("%s must be at most %v", v.Field, maxVal)
		}
		return fmt.Sprintf("%s is too large", v.Field)
	}

	if tmpl, ok := staticViolationMessages[v.Reason]; ok {
		return fmt.Sprintf(tmpl, v.Field)
	}
	return fmt.Sprintf("%s is invalid (%s)", v.Field, v.Reason)
}

// DefaultMessage returns a best-effort human-readable message: the
// explicit Message if one was set, otherwise each Violation's
// DefaultMessage joined together, otherwise a generic fallback based on
// Kind. Same scope note as Violation.DefaultMessage: a fallback for
// contexts with no frontend to build their own copy, not a replacement
// for one that has.
func (e *Error) DefaultMessage() string {
	if e.message != "" {
		return e.message
	}

	if len(e.violations) > 0 {
		parts := make([]string, len(e.violations))
		for i, v := range e.violations {
			parts[i] = v.DefaultMessage()
		}
		return strings.Join(parts, "; ")
	}

	if e.kind.Safe() {
		return e.code.String()
	}
	return "something went wrong"
}
