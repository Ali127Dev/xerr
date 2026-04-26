package xerr

import "net/http"

type Code string

func (c Code) String() string { return string(c) }

func (c Code) HTTPStatus() int {
	if status, ok := CodesHttpStatus[c]; ok {
		return status
	}
	return http.StatusInternalServerError
}

// ========================
// System / Internal Errors
// ========================

const (
	CodeInternalError      Code = "INTERNAL_SERVER_ERROR"
	CodeUnknownError       Code = "UNKNOWN_ERROR"
	CodeServiceUnavailable Code = "SERVICE_UNAVAILABLE"
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
	CodeInvalidCredentials  Code = "INVALID_CREDENTIALS"
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
