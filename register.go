package xerr

import (
	"fmt"
	"sync"
)

// registryMu guards concurrent access to CodesKind and CodesHttpStatus.
// Code.Kind() and Code.HTTPStatus() take a read lock; RegisterCode takes
// a write lock. The two maps stay exported for backward compatibility
// (see their own doc comments), but RegisterCode is the safe, documented
// way to add a code: unlike writing to the maps directly, it takes the
// same lock those reads use, and it panics on a collision instead of
// silently overwriting a code you didn't mean to touch.
var registryMu sync.RWMutex

// RegisterCode registers a new application Code with its Kind and HTTP
// status, so Code.Kind() / Code.HTTPStatus() — and everything built on
// them: Exposed(), MarshalJSON, ExposedCodes() — recognize it exactly
// like a built-in code.
//
// xerr intentionally does not know about any single application's
// domain codes (e.g. a "PLAN_NOT_INCLUDED" or "USAGE_LIMIT_EXCEEDED"
// that only makes sense to one service). RegisterCode is how the owning
// application teaches xerr about them, typically from an init() so
// registration happens once, before the server accepts traffic:
//
//	const CodePlanNotIncluded xerr.Code = "PLAN_NOT_INCLUDED"
//
//	func init() {
//	    xerr.RegisterCode(CodePlanNotIncluded, xerr.KindDomain, http.StatusForbidden)
//	}
//
// RegisterCode panics if code is already registered — whether it's one
// of xerr's own built-in codes or one your application registered
// earlier — since a silent overwrite would change the Kind/status of an
// existing code out from under whatever already depends on it. Register
// each code exactly once; RegisterCode is not meant to be called from a
// hot path.
//
// It also panics on malformed input, so a mistake fails loudly at
// startup instead of quietly registering a code no error will ever
// match correctly: code must not be empty, kind must be one of the
// defined Kind constants, and httpStatus must fall in the 400-599
// range (a registered code always represents a client- or server-side
// failure — there is no legitimate 2xx/3xx error code).
func RegisterCode(code Code, kind Kind, httpStatus int) {
	if code == "" {
		panic("xerr: RegisterCode: code must not be empty")
	}
	if !kind.known() {
		panic(fmt.Sprintf("xerr: RegisterCode: %q is not a valid Kind", kind))
	}
	if httpStatus < 400 || httpStatus > 599 {
		panic(fmt.Sprintf("xerr: RegisterCode: httpStatus %d is outside the valid 400-599 range", httpStatus))
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := CodesKind[code]; exists {
		panic(fmt.Sprintf("xerr: RegisterCode: code %q is already registered", code))
	}
	if _, exists := CodesHttpStatus[code]; exists {
		panic(fmt.Sprintf("xerr: RegisterCode: code %q is already registered", code))
	}

	CodesKind[code] = kind
	CodesHttpStatus[code] = httpStatus
}

// ExposedCodes returns every registered Code whose default Kind is safe
// to expose to a client (Kind.Safe()) — built-in codes and anything
// added via RegisterCode alike. Use it to build an OpenAPI/Swagger enum
// for the response "code" field, or in a test asserting your
// application's public codes stay in sync with what's actually
// registered.
//
// This reflects each Code's registered default only. An individual
// *Error can still override that default with WithExpose, so a code
// appearing here is not a guarantee every instance of it is exposed —
// only that it is by default.
func ExposedCodes() map[Code]Kind {
	registryMu.RLock()
	defer registryMu.RUnlock()

	out := make(map[Code]Kind, len(CodesKind))
	for code, kind := range CodesKind {
		if kind.Safe() {
			out[code] = kind
		}
	}
	return out
}
