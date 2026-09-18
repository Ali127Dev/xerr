package xerr

// Violation describes a single field-level rule violation.
//
// Reason is the stable, translatable identifier the client uses to look
// up a localized message template (see ErrorReason). Params carries the
// dynamic values that template needs — e.g. {"min": 8} alongside
// ErrorReasonTooShort — and is the only place free-form data belongs:
// Reason itself must stay within the fixed ErrorReason enum so a
// frontend translation table never goes stale.
type Violation struct {
	Field  string         `json:"field"`
	Reason ErrorReason    `json:"reason"`
	Params map[string]any `json:"params,omitempty"`
}

// Param is a single key/value entry attached to a Violation. Build one
// with P and pass it to WithViolation.
type Param struct {
	Key   string
	Value any
}

// P builds a Violation Param.
func P(key string, value any) Param {
	return Param{Key: key, Value: value}
}
