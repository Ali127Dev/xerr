# Changelog

All notable changes to this module are documented here. Format loosely
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

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
