// Package xerr provides a structured, layer-aware error handling system
// for DDD / clean-architecture Go services.
//
// Key features:
//   - Strongly typed, machine-readable error codes (Code), extensible
//     with your own application-specific codes via RegisterCode
//   - A Kind (domain / application / infrastructure / unknown) that
//     decides, by default, whether an error is safe to expose to a
//     client — so infrastructure failures can be logged in full while
//     only a generic response crosses the client boundary
//   - Field-level Violations with a closed, translatable Reason enum
//     plus free-form Params for dynamic detail (e.g. a length bound)
//   - An optional, independent human-readable Message for when the
//     backend should own the exact client-facing wording
//   - Internal-only Diagnostics for log context that never leaves the
//     server
//   - JSON-safe responses (no internal leakage)
//   - Error wrapping compatible with errors.Is / errors.As
//   - A slog.LogValuer implementation, so passing an *Error straight to
//     log/slog logs every field structured, no boilerplate
//   - Opt-in call-stack capture (WithStack) and a Recover helper that
//     turns a recovered panic into an *Error with a stack attached
//   - DefaultMessage for a best-effort plain-English fallback where
//     there is no frontend to build one
//   - Swagger-friendly error output model
//
// # What goes where
//
// Every Error field is always available server-side (for logging,
// errors.Is/errors.As, metrics); only a subset ever reaches the client,
// and that subset is chosen automatically by Kind:
//
//	field        your logs (Error(), LogValue, accessors)   the client (MarshalJSON)
//	----------   ------------------------------------------  --------------------------------
//	Code         always                                      always (real value if Exposed,
//	                                                          otherwise CodeInternalError)
//	Kind         always                                      never
//	Message      always                                      only if Exposed
//	Params       always                                      only if Exposed
//	Violations   always                                      only if Exposed
//	Diagnostics  always                                      never, even if Exposed
//	Err (cause)  always                                       never
//	Stack        always, if captured                          never
//
// Exposed defaults to Kind.Safe(): true for KindDomain / KindApplication,
// false for KindInfrastructure / KindUnknown. Override with WithExpose
// for the rare exception. See Error's doc comment for the full rationale.
package xerr
