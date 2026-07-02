package xerr

type ErrorReason string

const (
	ErrorReasonRequired      ErrorReason = "required"
	ErrorReasonInvalidFormat ErrorReason = "invalid_format"
	ErrorReasonTooShort      ErrorReason = "too_short"
	ErrorReasonTooLong       ErrorReason = "too_long"
	ErrorReasonMismatch      ErrorReason = "mismatch"
	ErrorReasonAlreadyExists ErrorReason = "already_exists"
	ErrorReasonNotFound      ErrorReason = "not_found"
	ErrorReasonCorrupted     ErrorReason = "corrupted"
	ErrorReasonExpired       ErrorReason = "expired"
)

func (e ErrorReason) String() string { return string(e) }
