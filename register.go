package xerr

import (
	"fmt"
	"maps"
	"regexp"
	"sync"
)

// registryMu guards concurrent access to codeKinds and codeStatuses.
// Code.Kind() and Code.HTTPStatus() take a read lock; RegisterCode takes
// a write lock. Both maps are unexported (since v3): RegisterCode is the
// only way to add a code, so every entry is guaranteed to have gone
// through its validation and through this same lock — there is no
// longer a way to write an unvalidated, unlocked entry directly.
var registryMu sync.RWMutex

// codeFormat is the shape RegisterCode requires of a new Code: an
// upper-case identifier starting with a letter, matching the style of
// every built-in code (e.g. "RESOURCE_NOT_FOUND", "PLAN_NOT_INCLUDED").
var codeFormat = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

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
// match correctly:
//   - code must be non-empty and match ^[A-Z][A-Z0-9_]*$ (upper-case
//     letters, digits, and underscores, starting with a letter) — the
//     same shape as every built-in code;
//   - kind must be one of the four defined Kind constants;
//   - httpStatus must fall in the 400-599 range (a registered code
//     always represents a client- or server-side failure — there is no
//     legitimate 2xx/3xx error code).
func RegisterCode(code Code, kind Kind, httpStatus int) {
	if code == "" {
		panic("xerr: RegisterCode: code must not be empty")
	}
	if !codeFormat.MatchString(string(code)) {
		panic(fmt.Sprintf("xerr: RegisterCode: code %q must match %s (upper-case letters, digits, and underscores, starting with a letter)", code, codeFormat.String()))
	}
	if !kind.known() {
		panic(fmt.Sprintf("xerr: RegisterCode: %q is not a valid Kind", kind))
	}
	if httpStatus < 400 || httpStatus > 599 {
		panic(fmt.Sprintf("xerr: RegisterCode: httpStatus %d is outside the valid 400-599 range", httpStatus))
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := codeKinds[code]; exists {
		panic(fmt.Sprintf("xerr: RegisterCode: code %q is already registered", code))
	}
	if _, exists := codeStatuses[code]; exists {
		panic(fmt.Sprintf("xerr: RegisterCode: code %q is already registered", code))
	}

	codeKinds[code] = kind
	codeStatuses[code] = httpStatus
}

// RegisteredCodes returns every registered Code (built-in and anything
// added via RegisterCode) together with its default Kind — regardless
// of whether that Kind is safe to expose. Use ExposedCodes instead if
// you only want the subset that's safe to expose by default.
func RegisteredCodes() map[Code]Kind {
	registryMu.RLock()
	defer registryMu.RUnlock()

	out := make(map[Code]Kind, len(codeKinds))
	maps.Copy(out, codeKinds)
	return out
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

	out := make(map[Code]Kind, len(codeKinds))
	for code, kind := range codeKinds {
		if kind.Safe() {
			out[code] = kind
		}
	}
	return out
}
