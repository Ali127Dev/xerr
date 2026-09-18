# xerr — Structured, Layer-Aware Error Handling for Go

xerr is a small, zero-dependency error handling library built for DDD / clean-architecture Go services. It gives you one `Error` type that carries full detail end-to-end — domain rule violations, business/application errors, infrastructure failures — and decides, by default, what's safe to hand back to a client versus what stays in your logs.

- 🧩 One `Error` type across every layer: domain, application, infrastructure
- 🛡️ Infrastructure/unknown errors are hidden from clients **by default** — log the real thing, return a generic response
- 🎯 Field-level `Violations` with a closed, translatable `Reason` enum plus free-form `Params` for dynamic detail (`min`, `max`, ...)
- 💬 An optional, independent `Message` for when the backend should own the exact wording — use violations, a message, both, or neither
- 🪢 Full `errors.Is` / `errors.As` / `Unwrap` support, chain-safe across multiple wrap layers
- 🧵 Internal-only `Diagnostics` for log context that never serializes to JSON
- 📋 `slog.LogValuer` built in — pass an `*Error` straight to `log/slog` and get every field structured, no boilerplate
- 🩻 Opt-in call-stack capture (`WithStack`) and a `Recover` helper that turns a panic into an `*Error` with a stack attached
- 🗣️ `DefaultMessage()` for a best-effort plain-English fallback when there's no frontend to build one
- 🧼 Zero dependencies, pure Go

---

## Install

```sh
go get github.com/Ali127Dev/xerr/v2
```

---

## The core idea: `Kind` decides exposure

Every `Code` has a default `Kind`:

| Kind | Meaning | Exposed to client by default |
|---|---|---|
| `KindDomain` | A core business rule tied to an entity (`user not found`, `order already shipped`) | ✅ yes |
| `KindApplication` | Cross-cutting app-layer concern (`unauthorized`, `too many requests`) | ✅ yes |
| `KindInfrastructure` | An external dependency failed (DB, cache, queue, network, third-party API) | ❌ no |
| `KindUnknown` | Unclassified / unexpected | ❌ no |

```go
err := xerr.New(xerr.CodeDatabaseError, xerr.WithErr(pqErr))

err.Kind()     // KindInfrastructure
err.Exposed()  // false — nothing about this reaches the client
err.Error()    // full detail, for your logger:
               // "DATABASE_ERROR (infrastructure): dial tcp 10.0.0.5:5432: connection refused"

json.Marshal(err)
// {"code":"INTERNAL_SERVER_ERROR"}
```

You never have to remember to redact a database error by hand — the library does it because of what kind of error it is. `WithKind` and `WithExpose` exist for the exceptions.

---

## Translating an infrastructure error into a domain error

The pattern this is built around: a repository wraps the raw failure as an infrastructure error (full detail, hidden from clients); the service layer catches it and re-wraps it as a domain error the client is meant to see — while the original stays attached for logs and `errors.As`.

```go
// repository layer
func (r *UserRepo) FindByID(ctx context.Context, id string) (*User, error) {
    var u User
    if err := r.db.First(&u, "id = ?", id).Error; err != nil {
        return nil, xerr.Wrap(err, xerr.CodeRecordNotFound,
            xerr.WithDiagnostic(xerr.DiagnosticOperation, "UserRepo.FindByID"),
        )
    }
    return &u, nil
}

// service layer
func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    u, err := s.repo.FindByID(ctx, id)
    if err != nil {
        // translate: infra detail -> a safe, well-known domain error
        return nil, xerr.New(xerr.CodeNotFound,
            xerr.WithMessage("user not found"),
            xerr.WithErr(err), // chain preserved for logs / errors.As
        )
    }
    return u, nil
}
```

At the HTTP boundary:

```go
xe, _ := xerr.FromError(err)
c.JSON(xe.HTTPStatus(), xe) // {"code":"RESOURCE_NOT_FOUND","message":"user not found"}
```

Your logger, meanwhile, can call `xe.Error()` (or walk `errors.Unwrap`) and see the entire chain, including the original `sql: no rows in result set`.

---

## Two ways to describe an error to the client — use either, both, or neither

**1. Structured violations** — the frontend builds the copy itself from a stable `Reason`, translated locally, with `Params` for the dynamic bits:

```go
err := xerr.New(xerr.CodeValidationFailed,
    xerr.WithViolation("password", xerr.ErrorReasonTooShort, xerr.P("min", 8)),
    xerr.WithViolation("email", xerr.ErrorReasonInvalidFormat),
)
```

```json
{
  "code": "VALIDATION_FAILED",
  "violations": [
    { "field": "password", "reason": "too_short", "params": { "min": 8 } },
    { "field": "email", "reason": "invalid_format" }
  ]
}
```

**2. A direct message** — the backend owns the exact string, the frontend just shows it:

```go
err := xerr.New(xerr.CodeConflict, xerr.WithMessage("this coupon has already been redeemed"))
```

```json
{ "code": "CONFLICT", "message": "this coupon has already been redeemed" }
```

Nothing stops you from setting both on the same error — a message for a toast plus violations for inline field errors.

`Reason` is a closed enum on purpose: hand the list to your frontend team once as a translation table (`required`, `invalid_format`, `too_short`, `too_long`, `too_small`, `too_large`, `mismatch`, `already_exists`, `not_found`, `corrupted`, `expired`, `invalid_value`) and it won't drift. `Params` is where you break that discipline deliberately, for values a template needs but that aren't part of the enum itself.

---

## Internal-only diagnostics

`Diagnostics` never serialize to JSON — they're for your logger only, e.g. which operation was running:

```go
xerr.New(xerr.CodeDatabaseError,
    xerr.WithErr(err),
    xerr.WithDiagnostic(xerr.DiagnosticOperation, "CreateUser"),
    xerr.WithDiagnostic(xerr.DiagnosticResource, "users"),
)
```

---

## Logging: `slog.LogValuer`

`*Error` implements `slog.LogValuer`, so `log/slog` renders every field your logger needs — including the ones a client never sees (`Kind`, `Diagnostics`, the wrapped cause) — without hand-writing each attribute:

```go
slog.Error("request failed", "err", xerr.Wrap(dbErr, xerr.CodeDatabaseError,
    xerr.WithDiagnostic(xerr.DiagnosticOperation, "CreateUser"),
))
```

```json
{
  "msg": "request failed",
  "err": {
    "code": "DATABASE_ERROR",
    "kind": "infrastructure",
    "cause": "dial tcp 10.0.0.5:5432: connection refused",
    "diagnostics": { "operation": "CreateUser" }
  }
}
```

Nothing here is filtered by `Exposed` — this path is for your logger, never for a client response.

---

## Stack traces: opt-in, plus automatic on `Recover`

`WithStack()` captures the current call stack, retrievable via `Stack()`. It's opt-in because `runtime.Callers` isn't free, and most errors (a failed validation, a 404) don't need one — reach for it on the ones you do, typically `KindInfrastructure` / `KindUnknown`:

```go
if err != nil {
    return xerr.Wrap(err, xerr.CodeExternalService, xerr.WithStack())
}
```

`Recover` — for turning a panic into an `*Error` — always captures one, since a stack trace is the entire reason to catch a panic:

```go
func (s *Service) Handle(ctx context.Context, req Request) (resp Response, err error) {
    defer func() {
        if v := recover(); v != nil {
            err = xerr.Recover(v)
        }
    }()
    // ...
}
```

The result uses `CodePanic` (`KindUnknown`, unsafe to expose — the client only ever sees a generic internal error). `Stack()` is log-only: never part of `MarshalJSON`, and deliberately left out of `Error()`'s single-line output — call it explicitly, or let `slog.LogValuer` include it automatically when present.

---

## `DefaultMessage()` — a fallback, not a localization system

For contexts with no frontend to build a message from `Reason` + `Params` — a CLI tool, a server log meant for a human, a quick prototype — `DefaultMessage()` renders a best-effort English sentence:

```go
err := xerr.New(xerr.CodeValidationFailed,
    xerr.WithViolation("email", xerr.ErrorReasonRequired),
    xerr.WithViolation("password", xerr.ErrorReasonTooShort, xerr.P("min", 8)),
)

err.DefaultMessage()
// "email is required; password must be at least 8 characters"
```

Order of precedence: the explicit `Message` if set, otherwise each `Violation.DefaultMessage()` joined together, otherwise a generic fallback based on `Kind`. There's no catalog and no locale negotiation — wherever a real client exists, prefer letting it build its own copy from `Reason` + `Params`, which is what that enum is for. Use this only as the fallback it is.

---

## `errors.Is` / `errors.As`

`Is` matches by `Code` only — message, violations, and diagnostics are runtime detail, so a bare sentinel works:

```go
if errors.Is(err, xerr.New(xerr.CodeNotFound)) {
    // ...
}

var xe *xerr.Error
if errors.As(err, &xe) {
    log.Error(xe.Error(), "code", xe.Code(), "kind", xe.Kind())
}

// or, equivalently:
xe, ok := xerr.FromError(err)
```

---

## API reference

### Constructors

- `New(code Code, opts ...ErrorOption) *Error` — a fresh error. `Kind` defaults from `code.Kind()`.
- `Wrap(err error, code Code, opts ...ErrorOption) *Error` — wraps an underlying error; returns `nil` if `err` is `nil`.
- `FromError(err error) (*Error, bool)` — convenience wrapper over `errors.As`.
- `Recover(v any, opts ...ErrorOption) *Error` — converts a recovered panic value into an `*Error` with `CodePanic` and a captured stack; returns `nil` if `v` is `nil`.

### Options

- `WithMessage(string)` — direct, ready-to-display client message.
- `WithViolation(field string, reason ErrorReason, params ...Param)` — structured field violation; repeatable.
- `WithViolations(vs ...Violation)` — append an already-built batch of violations (e.g. from a third-party validator).
- `WithErr(error)` — attach the wrapped cause (log-only, never serialized).
- `WithDiagnostic(DiagnosticKey, string)` — internal-only debug context (log-only).
- `WithKind(Kind)` — override the code's default `Kind`.
- `WithExpose(bool)` — override the kind-based exposure default.
- `WithStack()` — capture the current call stack for `Stack()`. Opt-in; `Recover` always captures one.

### `*Error` methods

`Code()`, `Kind()`, `Message()`, `Err()`, `Violations()`, `Diagnostics()`, `Exposed()`, `HTTPStatus()`, `Stack()`, `DefaultMessage()`, `Error()`, `Unwrap()`, `Is()`, `MarshalJSON()`, `LogValue()`.

### A caveat: typed nil

`Wrap(nil, ...)` and `Recover(nil)` return a literal `nil` of type `*xerr.Error`. That's fine as long as you keep using the concrete `*xerr.Error` type — but if you assign the result directly to a variable of interface type `error`, the classic Go footgun applies: `err != nil` will be `true` even though the underlying pointer is nil (a "typed nil"), and calling a method on it will panic. Don't do this:

```go
var err error = xerr.Wrap(nil, xerr.CodeInternalError) // err != nil is now true!
```

This is inherent to how Go interfaces work, not specific to xerr — just keep it in mind at the boundary where a `*xerr.Error` gets assigned to a plain `error`.

---

## Using with Gin

```go
func ErrorHandler(c *gin.Context) {
    c.Next()

    if len(c.Errors) == 0 {
        return
    }

    err := c.Errors.Last().Err

    xe, ok := xerr.FromError(err)
    if !ok {
        xe = xerr.New(xerr.CodeInternalError, xerr.WithErr(err))
    }

    logger.Error(xe.Error(), "code", xe.Code(), "kind", xe.Kind())
    c.JSON(xe.HTTPStatus(), xe)
}
```

---

## Swagger integration

`SwaggerErrOutput` / `SwaggerViolationOutput` mirror the real JSON shape for `swaggo/swag`-style doc generation, without leaking internal fields:

```go
// @Failure 400 {object} xerr.SwaggerErrOutput
// @Failure 404 {object} xerr.SwaggerErrOutput
// @Failure 500 {object} xerr.SwaggerErrOutput
```

---

## License

MIT
