package xerr

import "net/http"

// Code represents a machine-readable application error code.
type Code string

func (c Code) String() string { return string(c) }

func (c Code) HTTPStatus() int {
	registryMu.RLock()
	defer registryMu.RUnlock()

	if status, ok := CodesHttpStatus[c]; ok {
		return status
	}
	return http.StatusInternalServerError
}

// Kind reports the default Kind for this code, which in turn decides
// whether an error carrying it is exposed to clients by default. See
// CodesKind.
func (c Code) Kind() Kind {
	registryMu.RLock()
	defer registryMu.RUnlock()

	if kind, ok := CodesKind[c]; ok {
		return kind
	}
	return KindUnknown
}

// ========================
// System / Internal Errors
// ========================

const (
	CodeInternalError      Code = "INTERNAL_SERVER_ERROR"
	CodeUnknownError       Code = "UNKNOWN_ERROR"
	CodeServiceUnavailable Code = "SERVICE_UNAVAILABLE"
	CodePanic              Code = "PANIC"
)

// ========================
// Request Errors
// ========================

const (
	CodeBadRequest       Code = "BAD_REQUEST"
	CodeValidationFailed Code = "VALIDATION_FAILED"
	CodeMalformedJSON    Code = "MALFORMED_JSON"
	CodeMissingField     Code = "MISSING_REQUIRED_FIELD"
	CodeInvalidParam     Code = "INVALID_PARAMETER"
)

// ========================
// Authentication Errors
// ========================

const (
	CodeUnauthorized        Code = "UNAUTHORIZED"
	CodeInvalidCredentials  Code = "INVALID_CREDENTIALS" //nolint:gosec
	CodeInvalidToken        Code = "INVALID_TOKEN"
	CodeExpiredToken        Code = "TOKEN_EXPIRED"
	CodeRefreshTokenInvalid Code = "INVALID_REFRESH_TOKEN"
)

// ========================
// Authorization Errors
// ========================

const (
	CodeForbidden         Code = "FORBIDDEN"
	CodePermissionDenied  Code = "PERMISSION_DENIED"
	CodeInsufficientScope Code = "INSUFFICIENT_SCOPE"
)

// ========================
// Resource Errors
// ========================

const (
	CodeNotFound        Code = "RESOURCE_NOT_FOUND"
	CodeAlreadyExists   Code = "RESOURCE_ALREADY_EXISTS"
	CodeResourceLocked  Code = "RESOURCE_LOCKED"
	CodeResourceDeleted Code = "RESOURCE_DELETED"
)

// ========================
// Business Logic Errors
// ========================

const (
	CodeConflict        Code = "CONFLICT"
	CodeOperationFailed Code = "OPERATION_FAILED"
	CodeInvalidState    Code = "INVALID_STATE"
)

// ========================
// Rate Limit / Security
// ========================

const (
	CodeTooManyRequests Code = "TOO_MANY_REQUESTS"
)

// ========================
// Storage / Database
// ========================

const (
	CodeDatabaseError   Code = "DATABASE_ERROR"
	CodeDuplicateKey    Code = "DUPLICATE_KEY"
	CodeForeignKeyError Code = "FOREIGN_KEY_CONSTRAINT"
	CodeRecordNotFound  Code = "RECORD_NOT_FOUND"
)

// ========================
// External / Network
// ========================

const (
	CodeNetworkError    Code = "NETWORK_ERROR"
	CodeTimeout         Code = "TIMEOUT"
	CodeExternalService Code = "EXTERNAL_SERVICE_ERROR"
)
