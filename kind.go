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

// known reports whether k is one of the four defined Kind constants.
// Used by RegisterCode to reject a typo'd or made-up Kind at
// registration time instead of letting it silently default through
// Safe()'s "default: false" branch.
func (k Kind) known() bool {
	switch k {
	case KindDomain, KindApplication, KindInfrastructure, KindUnknown:
		return true
	default:
		return false
	}
}

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

// codeKinds maps an error Code to its default Kind. Codes not present
// default to KindUnknown (unsafe to expose).
//
// Unexported since v3: the only way to add a code is RegisterCode,
// which takes the lock Code.Kind()/Code.HTTPStatus() use for reads and
// validates its input, instead of letting a caller write an
// unvalidated, unlocked entry straight into this map (as was possible
// through the exported CodesKind var in v2).
var codeKinds = map[Code]Kind{
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
