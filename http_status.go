package xerr

import "net/http"

// CodesHttpStatus maps an error Code to default HTTP status.
//
// Prefer RegisterCode to add a new code: it takes the same lock
// Code.Kind() / Code.HTTPStatus() use for reads, and it panics on a
// collision instead of silently overwriting an existing entry. Writing
// to this map directly still works, for backward compatibility, but
// bypasses both of those protections.
var CodesHttpStatus = map[Code]int{
	// ========================
	// System / Internal Errors
	// ========================

	CodeInternalError:      http.StatusInternalServerError, // 500
	CodeUnknownError:       http.StatusInternalServerError, // 500
	CodeServiceUnavailable: http.StatusServiceUnavailable,  // 503
	CodePanic:              http.StatusInternalServerError, // 500

	// ========================
	// Request Errors
	// ========================

	CodeBadRequest:       http.StatusBadRequest, // 400
	CodeValidationFailed: http.StatusBadRequest, // 400
	CodeMalformedJSON:    http.StatusBadRequest, // 400
	CodeMissingField:     http.StatusBadRequest, // 400
	CodeInvalidParam:     http.StatusBadRequest, // 400

	// ========================
	// Authentication Errors
	// ========================

	CodeUnauthorized:        http.StatusUnauthorized, // 401
	CodeInvalidCredentials:  http.StatusUnauthorized, // 401
	CodeInvalidToken:        http.StatusUnauthorized, // 401
	CodeExpiredToken:        http.StatusUnauthorized, // 401
	CodeRefreshTokenInvalid: http.StatusUnauthorized, // 401

	// ========================
	// Authorization Errors
	// ========================

	CodeForbidden:         http.StatusForbidden, // 403
	CodePermissionDenied:  http.StatusForbidden, // 403
	CodeInsufficientScope: http.StatusForbidden, // 403

	// ========================
	// Resource Errors
	// ========================

	CodeNotFound:        http.StatusNotFound, // 404
	CodeAlreadyExists:   http.StatusConflict, // 409
	CodeResourceLocked:  http.StatusLocked,   // 423
	CodeResourceDeleted: http.StatusGone,     // 410

	// ========================
	// Business Logic Errors
	// ========================

	CodeConflict:        http.StatusConflict,            // 409
	CodeOperationFailed: http.StatusUnprocessableEntity, // 422
	CodeInvalidState:    http.StatusConflict,            // 409

	// ========================
	// Rate Limit / Security
	// ========================

	CodeTooManyRequests: http.StatusTooManyRequests, // 429

	// ========================
	// Storage / Database
	// ========================

	CodeDatabaseError:   http.StatusInternalServerError, // 500
	CodeDuplicateKey:    http.StatusConflict,            // 409
	CodeForeignKeyError: http.StatusConflict,            // 409
	CodeRecordNotFound:  http.StatusNotFound,            // 404

	// ========================
	// External / Network
	// ========================

	CodeNetworkError:    http.StatusBadGateway,     // 502
	CodeTimeout:         http.StatusGatewayTimeout, // 504
	CodeExternalService: http.StatusBadGateway,     // 502
}
