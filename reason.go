package xerr

// ErrorReason is a stable, translatable identifier for why a single field
// violated a rule. It intentionally stays a closed enum: the frontend can
// build a static localization table keyed by these values. Anything that
// varies per-occurrence (a length bound, an allowed set, ...) belongs in
// a Violation's Params instead of growing this enum.
type ErrorReason string

const (
	ErrorReasonRequired      ErrorReason = "required"
	ErrorReasonInvalidFormat ErrorReason = "invalid_format"
	ErrorReasonInvalidValue  ErrorReason = "invalid_value" // params: allowed
	ErrorReasonTooShort      ErrorReason = "too_short"     // params: min
	ErrorReasonTooLong       ErrorReason = "too_long"      // params: max
	ErrorReasonTooSmall      ErrorReason = "too_small"     // params: min
	ErrorReasonTooLarge      ErrorReason = "too_large"     // params: max
	ErrorReasonMismatch      ErrorReason = "mismatch"
	ErrorReasonAlreadyExists ErrorReason = "already_exists"
	ErrorReasonNotFound      ErrorReason = "not_found"
	ErrorReasonCorrupted     ErrorReason = "corrupted"
	ErrorReasonExpired       ErrorReason = "expired"
)

func (e ErrorReason) String() string { return string(e) }
