package xerr

// Kind classifies where an error originated, which in turn decides
// whether it is safe to expose to a client by default.
type Kind string

const (
	// KindDomain marks a violation of a core business/domain rule tied to a
	// specific entity or invariant (e.g. "order already shipped", "user not
	// found"). Safe to expose by default.
	KindDomain Kind = "domain"

	// KindApplication marks a cross-cutting application-layer concern that
	// is not about a specific domain entity (auth, authorization, rate
	// limiting, request-contract validation). Safe to expose by default.
	KindApplication Kind = "application"

	// KindInfrastructure marks a failure from an external system the
	// service depends on (database, cache, queue, network, third-party
	// API). Never safe to expose by default: the client should only see
	// that something failed, while full detail stays in server-side logs.
	KindInfrastructure Kind = "infrastructure"

	// KindUnknown marks an unclassified or unexpected error. Treated the
	// same as KindInfrastructure: unsafe to expose by default.
	KindUnknown Kind = "unknown"
)

func (k Kind) String() string { return string(k) }

// Safe reports whether errors of this kind are exposed to clients by
// default. It can always be overridden per-error with WithExpose.
func (k Kind) Safe() bool {
	switch k {
	case KindDomain, KindApplication:
		return true
	case KindInfrastructure, KindUnknown:
		return false
	default:
		return false
	}
}

// CodesKind maps an error Code to its default Kind. Codes not present
// default to KindUnknown (unsafe to expose).
var CodesKind = map[Code]Kind{
	// System / Internal
	CodeInternalError:      KindUnknown,
	CodeUnknownError:       KindUnknown,
	CodeServiceUnavailable: KindInfrastructure,
	CodePanic:              KindUnknown,

	// Request
	CodeBadRequest:       KindDomain,
	CodeValidationFailed: KindDomain,
	CodeMalformedJSON:    KindDomain,
	CodeMissingField:     KindDomain,
	CodeInvalidParam:     KindDomain,

	// Authentication
	CodeUnauthorized:        KindApplication,
	CodeInvalidCredentials:  KindApplication,
	CodeInvalidToken:        KindApplication,
	CodeExpiredToken:        KindApplication,
	CodeRefreshTokenInvalid: KindApplication,

	// Authorization
	CodeForbidden:         KindApplication,
	CodePermissionDenied:  KindApplication,
	CodeInsufficientScope: KindApplication,

	// Resource
	CodeNotFound:        KindDomain,
	CodeAlreadyExists:   KindDomain,
	CodeResourceLocked:  KindDomain,
	CodeResourceDeleted: KindDomain,

	// Business Logic
	CodeConflict:        KindDomain,
	CodeOperationFailed: KindDomain,
	CodeInvalidState:    KindDomain,

	// Rate Limit / Security
	CodeTooManyRequests: KindApplication,

	// Storage / Database
	CodeDatabaseError:   KindInfrastructure,
	CodeDuplicateKey:    KindInfrastructure,
	CodeForeignKeyError: KindInfrastructure,
	CodeRecordNotFound:  KindInfrastructure,

	// External / Network
	CodeNetworkError:    KindInfrastructure,
	CodeTimeout:         KindInfrastructure,
	CodeExternalService: KindInfrastructure,
}
