# Changelog

All notable changes to this module are documented here. Format loosely
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [3.0.0] - 2026-09-19

A deliberate breaking release with one purpose: close the direct-registry-mutation
backdoor v2.1.0 left open. `RegisterCode` was already the documented way to add a
code; this release makes it the *only* way, by unexporting the maps it wrote into.

### ⚠️ Breaking changes

- **Module path is now `github.com/Ali127Dev/xerr/v3`.** Required by Go's module
  versioning rules for a breaking v3+ API. Import as
  `github.com/Ali127Dev/xerr/v3` — the package name is still `xerr`, so call
  sites don't otherwise change. See [README.md](README.md#migrating-from-v2-to-v3)
  for the full migration guide.
- **`CodesKind` and `CodesHttpStatus` are no longer exported.** They were
  `map[Code]Kind` / `map[Code]int` package vars that any caller could write into
  directly — no validation, no locking, no duplicate check — bypassing
  `RegisterCode` entirely. They're now unexported (`codeKinds`, `codeStatuses`);
  `RegisterCode` is the only way to add a code. If you read these maps to
  enumerate codes, use the new `RegisteredCodes()` (all codes) or the existing
  `ExposedCodes()` (safe-to-expose codes only) instead — both return a
  defensive copy.
- **`RegisterCode` validates the code's format.** It already panicked (since
  v2.1.0) on a duplicate/collision, an empty code, an unknown `Kind`, or an
  `httpStatus` outside 400-599. It now also panics if `code` doesn't match
  `^[A-Z][A-Z0-9_]*$` — the same upper-case-with-underscores shape every
  built-in code already follows. A code registered under the old, looser rules
  that happened to already match this shape is unaffected.

### Added

- `RegisteredCodes() map[Code]Kind` — every registered code (built-in and
  anything from `RegisterCode`), regardless of whether it's safe to expose.
  The replacement for iterating the old exported `CodesKind` directly.
- A structural test (`TestNoExportedMutableGlobals`, parses the package's own
  source with `go/parser`) that fails the build if the package ever gains
  another exported package-level `var` of map or slice type — the guardrail
  that keeps this release's backdoor closed for good.

### Notes

- Deliberately unchanged: registering a code concurrently with traffic that's
  already reading the registry is still not recommended. The `sync.RWMutex`
  added in v2.1.0 makes concurrent registration data-race-free, but a code
  registered mid-traffic can still be seen as unregistered by requests that
  raced ahead of the write. The documented pattern remains: register from
  `init()`, before your server starts accepting traffic.
- No other exported identifier was renamed or removed.

## [2.1.0] - 2026-09-19

A backward-compatible feature release: applications can now register
their own domain-specific `Code`s instead of writing directly to the
exported `CodesKind`/`CodesHttpStatus` maps, and errors can carry
dynamic detail that isn't tied to one field.

### Added

- `RegisterCode(code Code, kind Kind, httpStatus int)` — registers a new
  `Code` with its `Kind` and HTTP status, so `Code.Kind()` /
  `Code.HTTPStatus()` (and everything built on them) recognize it
  exactly like a built-in code. Panics on a duplicate registration —
  whether the collision is with a built-in code or one your application
  registered earlier — instead of silently overwriting it. Also panics
  on malformed input: an empty `code`, a `kind` outside the four defined
  `Kind` constants, or an `httpStatus` outside 400-599. Guards
  `CodesKind`/`CodesHttpStatus` with a `sync.RWMutex`, also now used by
  `Code.Kind()`/`Code.HTTPStatus()` for their reads, so registration is
  safe under `-race` as long as it happens before those reads race with
  it (typically: register from `init()`, before your server starts
  accepting traffic).
- `ExposedCodes() map[Code]Kind` — every registered code (built-in or
  via `RegisterCode`) whose default `Kind` is safe to expose. Meant for
  building a Swagger/OpenAPI enum for the `code` field, or a test that
  keeps an application's public codes in sync with what's registered.
- `WithParam(key string, value any) ErrorOption` and `Error.Params()` —
  error-level dynamic detail (e.g. `{"resource": "product", "max": 5}`),
  analogous to a `Violation`'s `Params` but scoped to the whole error.
  Follows the exact same exposure rule as `Message` and `Violations`:
  included in `MarshalJSON` only when `Exposed()`, always included in
  `Error()` and `LogValue()`.
- `SwaggerErrOutput.Params map[string]any` — brings the Swagger DTO back
  in sync with the real `MarshalJSON` shape now that `Error` can carry
  params. Its doc comment explains why `Code` still can't be a generated
  enum (xerr doesn't know an application's domain codes) and how to
  build one yourself from `ExposedCodes()`.

### Notes

- `CodesKind` and `CodesHttpStatus` remain exported for backward
  compatibility — writing to them directly still works — but their doc
  comments now point to `RegisterCode` as the safe, validated way to add
  a code.
- No existing exported identifier was renamed or removed.

## [2.0.0] - 2026-09-18

This release reworks the public API around one idea: an error's `Kind`
(domain / application / infrastructure / unknown) decides, by default,
whether it's safe to hand back to a client. See [README.md](README.md)
and the package doc comment (`doc.go`) for the full design.

The API breaks compared to `v1.2.4`, so the module path now carries the
major version as Go requires: `go get github.com/Ali127Dev/xerr/v2`.
Import it as `github.com/Ali127Dev/xerr/v2` — the Go package name is
still `xerr`, so call sites (`xerr.New(...)`, etc.) don't change.

### ⚠️ Breaking changes

- **Module path is now `github.com/Ali127Dev/xerr/v2`.** Required by Go's
  module versioning rules once a module ships a breaking v2+ API; a bare
  `v2.0.0` tag on the old path would not be resolvable by `go get`.
- **`meta map[string]ErrorReason` replaced by `Violations []Violation`.**
  `WithMeta(key, value)` is gone; use `WithViolation(field, reason,
  params...)` (or the new `WithViolations` for a pre-built batch)
  instead. A `Violation` is `{Field, Reason, Params}` — `Reason` stays
  the closed, translatable enum `ErrorReason` always was; `Params` is
  new, a `map[string]any` for dynamic values a message template needs
  (e.g. `{"min": 8}` alongside `ErrorReasonTooShort`).
- **`MarshalJSON` output changed for unsafe-by-default errors.** Any
  error whose `Kind` is `KindInfrastructure` or `KindUnknown` (e.g.
  `CodeDatabaseError`, `CodeInternalError`) now serializes to a bare
  `{"code":"INTERNAL_SERVER_ERROR"}` — no message, no violations — so
  infrastructure detail can no longer leak into a client response by
  accident. Override with `WithExpose(true)` for the rare case where
  that's actually wanted. Domain/application errors are unaffected and
  now additionally carry a `violations` field when set.
- **`Error.Is` now matches by `Code` only.** It previously also required
  exact `meta` equality, which made `xerr.New(xerr.CodeNotFound)` an
  unreliable sentinel for `errors.Is` the moment a concrete error
  carried any detail. It's now a proper sentinel.
- Removed `WithMeta`; `Meta()` accessor removed (see `Violations()`).

### Added

- `Kind` type (`KindDomain`, `KindApplication`, `KindInfrastructure`,
  `KindUnknown`) and `Code.Kind()` / `CodesKind`, mirroring the existing
  `Code.HTTPStatus()` / `CodesHttpStatus` pattern.
- `Error.Exposed()`, plus `WithKind` and `WithExpose` to override the
  Kind-based default per error.
- `Violation` type and `WithViolation` / `WithViolations` options; `P()`
  helper for building `Violation.Params` entries.
- New `ErrorReason` values: `ErrorReasonTooSmall`, `ErrorReasonTooLarge`,
  `ErrorReasonInvalidValue`.
- New codes: `CodeNetworkError`, `CodeTimeout`, `CodeExternalService`,
  `CodePanic`.
- `FromError(err) (*Error, bool)` — thin `errors.As` convenience wrapper.
- `WithStack()` option and `Error.Stack()` — opt-in call-stack capture.
- `Recover(v any, opts ...ErrorOption) *Error` — converts a recovered
  panic into an `*Error` (`CodePanic`, unsafe to expose) with a stack
  always captured.
- `Error.LogValue()` — implements `slog.LogValuer`, so `slog.Error("...",
  "err", xerr.Wrap(...))` logs every field structured, including the
  ones `MarshalJSON` hides from clients (`Kind`, `Diagnostics`, `Err`).
- `Violation.DefaultMessage()` / `Error.DefaultMessage()` — a best-effort
  plain-English fallback message for contexts with no frontend to build
  one from `Reason` + `Params`. Not a localization system: no catalog,
  no locale negotiation.
- `SwaggerViolationOutput`, and `SwaggerErrOutput.Violations` replacing
  its old `Meta` field, to match the real JSON shape.

### Fixed

- `Error()`'s log-line formatting was non-deterministic (map iteration
  order for violations/diagnostics); output is now stable.

### Other

- `.golangci.yml` migrated from the golangci-lint v1 config schema to
  v2 (`golangci-lint migrate`), matching the v2 toolchain. CI's install
  step now points at the `/v2` module path so it actually installs v2
  instead of silently falling back to the last v1 release.
