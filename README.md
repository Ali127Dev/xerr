# xerr — Structured, Layer-Aware Error Handling for Go

`xerr` is a small, zero-dependency Go library for handling errors in a DDD / clean-architecture service. It gives you **one error type** to use in every layer of your app — domain, application, infrastructure — and it automatically decides what's safe to send back to an HTTP client versus what should only ever appear in your logs.

This guide walks through every feature, step by step, with runnable code. No prior knowledge of the library is assumed.

- [Why this library exists](#why-this-library-exists)
- [Install](#install)
- [Step 1: Create your first error](#step-1-create-your-first-error)
- [Step 2: Understand `Code`](#step-2-understand-code)
- [Step 3: Understand `Kind` and who gets to see what](#step-3-understand-kind-and-who-gets-to-see-what)
- [Step 4: Override the default with `WithKind` / `WithExpose`](#step-4-override-the-default-with-withkind--withexpose)
- [Step 5: Three ways to describe an error to a client](#step-5-three-ways-to-describe-an-error-to-a-client)
- [Step 6: `Diagnostics` — notes that never leave the server](#step-6-diagnostics--notes-that-never-leave-the-server)
- [Step 7: Wrapping an existing error](#step-7-wrapping-an-existing-error)
- [Step 8: The full DDD pattern — repository → service → HTTP](#step-8-the-full-ddd-pattern--repository--service--http)
- [Step 9: `errors.Is`, `errors.As`, and `FromError`](#step-9-errorsis-errorsas-and-fromerror)
- [Step 10: Logging with `log/slog`](#step-10-logging-with-logslog)
- [Step 11: Call-stack capture](#step-11-call-stack-capture)
- [Step 12: Recovering from panics](#step-12-recovering-from-panics)
- [Step 13: `DefaultMessage()` — a plain-English fallback](#step-13-defaultmessage--a-plain-english-fallback)
- [Step 14: HTTP status codes](#step-14-http-status-codes)
- [Step 15: Wiring it into an HTTP framework](#step-15-wiring-it-into-an-http-framework)
- [Step 16: Swagger / OpenAPI docs](#step-16-swagger--openapi-docs)
- [Reference: built-in codes](#reference-built-in-codes)
- [Reference: built-in violation reasons](#reference-built-in-violation-reasons)
- [Reference: full API](#reference-full-api)
- [Gotcha: typed nil](#gotcha-typed-nil)
- [License](#license)

---

## Why this library exists

In a real backend you get three very different kinds of errors, and they need three very different responses:

1. **Domain errors** — "this order was already shipped", "this user was not found". These are expected, well-understood, and the client is *supposed* to see them.
2. **Application errors** — "unauthorized", "too many requests". Also expected, also safe to show.
3. **Infrastructure errors** — "the database connection timed out", "the payment gateway is unreachable". These are *not* safe to show. Leaking a raw database error message to a client is both bad UX and a security smell — but you still very much want the full message in your logs.

Without a library, you end up hand-writing an `if` somewhere in every handler to decide "is this error safe to show?" — and it's easy to forget, once, and leak something you shouldn't have.

`xerr` bakes that decision into the error itself, so you can't forget it.

---

## Install

```sh
go get github.com/Ali127Dev/xerr/v2
```

Import it like this:

```go
import "github.com/Ali127Dev/xerr/v2"
```

The Go package name is still `xerr` (the `/v2` is just part of the module path, required by Go once a library makes a breaking change — see [CHANGELOG.md](CHANGELOG.md)). So in code you still write `xerr.New(...)`, `xerr.Code`, etc., exactly as shown below.

---

## Step 1: Create your first error

The most basic thing you can do is create an error with a `Code`:

```go
err := xerr.New(xerr.CodeNotFound)
```

That's it — `err` is now a `*xerr.Error`, which implements Go's standard `error` interface, so you can return it, wrap it, log it, and compare it just like any other error.

---

## Step 2: Understand `Code`

A `Code` is a short, machine-readable, all-caps string identifying *what kind of problem* happened — e.g. `"RESOURCE_NOT_FOUND"`, `"VALIDATION_FAILED"`, `"DATABASE_ERROR"`. The client's frontend can safely branch on these strings (`if (error.code === "RESOURCE_NOT_FOUND")`) without ever having to parse a human sentence.

`xerr` ships a set of common codes ready to use — see the [full table below](#reference-built-in-codes). You can also define your own:

```go
const CodeCouponExpired xerr.Code = "COUPON_EXPIRED"
```

A bare constant like that isn't enough on its own — `xerr` has no idea what `Kind` or HTTP status it should carry, so until you register it, `CodeCouponExpired.Kind()` reports `KindUnknown` and `.HTTPStatus()` reports `500`. Teach `xerr` about it with `RegisterCode`, typically from an `init()` so it happens once, before your server starts accepting traffic:

```go
func init() {
    xerr.RegisterCode(CodeCouponExpired, xerr.KindDomain, http.StatusConflict)
}
```

`RegisterCode` panics if the code is already registered — whether that's one of `xerr`'s own built-ins or a code your application registered earlier — so a typo that collides with an existing code fails loudly at startup instead of silently changing that code's behavior. It also panics on malformed input: an empty code, a `Kind` outside the four defined constants, or an `httpStatus` outside 400-599. `xerr` itself stays generic and knows nothing about any single application's domain (entity names, plan tiers, feature flags, ...) — every app-specific code, like `CodeCouponExpired` above, is defined and registered by the application that owns it.

Once registered, a custom code behaves exactly like a built-in one everywhere — `Kind()`, `HTTPStatus()`, `Exposed()`, `MarshalJSON`, `ExposedCodes()` (below).

Every `Code` has two things attached to it automatically:

```go
xerr.CodeNotFound.HTTPStatus() // 404
xerr.CodeNotFound.Kind()       // xerr.KindDomain
```

`HTTPStatus()` is what you'd expect — the right HTTP status for that kind of problem. `Kind()` is explained next, and it's the most important concept in this library.

---

## Step 3: Understand `Kind` and who gets to see what

Every `Code` belongs to a `Kind`:

| `Kind` | What it means | Example codes | Safe for a client to see? |
|---|---|---|---|
| `KindDomain` | A rule about your business entities | `CodeNotFound`, `CodeValidationFailed`, `CodeConflict` | ✅ Yes |
| `KindApplication` | A cross-cutting app concern, not tied to one entity | `CodeUnauthorized`, `CodeTooManyRequests` | ✅ Yes |
| `KindInfrastructure` | An external system failed (DB, network, cache, ...) | `CodeDatabaseError`, `CodeTimeout` | ❌ No |
| `KindUnknown` | Anything unclassified / unexpected | `CodeInternalError`, `CodePanic` | ❌ No |

`Kind` decides `Exposed()` — whether the error's message and violations are allowed to reach a client:

```go
domainErr := xerr.New(xerr.CodeNotFound)
domainErr.Kind()    // KindDomain
domainErr.Exposed() // true

infraErr := xerr.New(xerr.CodeDatabaseError)
infraErr.Kind()    // KindInfrastructure
infraErr.Exposed() // false
```

This matters most when you serialize the error to JSON for an HTTP response — `MarshalJSON` looks at `Exposed()` and decides what to include:

```go
safe := xerr.New(xerr.CodeValidationFailed, xerr.WithMessage("email is invalid"))
json.Marshal(safe)
// {"code":"VALIDATION_FAILED","message":"email is invalid"}

unsafe := xerr.New(xerr.CodeDatabaseError, xerr.WithMessage("connection refused"))
json.Marshal(unsafe)
// {"code":"INTERNAL_SERVER_ERROR"}
```

Look closely at that second example: the `message` is gone, **and the `code` itself changed** to a generic `INTERNAL_SERVER_ERROR`. This is on purpose — the specific code `DATABASE_ERROR` is itself information you probably don't want a stranger fingerprinting your stack with. Nothing about what really happened crosses the boundary.

But the full information is *never lost* — it's just not in the JSON. You still have it in the Go value itself:

```go
unsafe.Code()             // CodeDatabaseError  (the real one)
unsafe.Kind()              // KindInfrastructure
unsafe.Message()           // "connection refused"
unsafe.Error()             // "DATABASE_ERROR (infrastructure): connection refused"
```

So: log `unsafe.Error()` (or better, pass `unsafe` straight to `log/slog` — see [Step 10](#step-10-logging-with-logslog)), and send `unsafe` (the same Go value!) to `json.Marshal` for the HTTP response. One value, two different views, decided automatically.

---

## Step 4: Override the default with `WithKind` / `WithExpose`

Sometimes the default is wrong for one specific error. Two options let you override it:

```go
// Force a database error to behave like a domain error (rare — be careful):
err := xerr.New(xerr.CodeDatabaseError, xerr.WithKind(xerr.KindDomain))

// Or leave the Kind alone but just force exposure on/off directly:
err := xerr.New(xerr.CodeDatabaseError, xerr.WithExpose(true))  // now safe to expose
err := xerr.New(xerr.CodeValidationFailed, xerr.WithExpose(false)) // now hidden, even though it's a domain error
```

`WithExpose` is the more direct and usually the right tool — it overrides exactly the "should this leak" decision, without also changing what `Kind()` reports (which is useful for filtering/metrics separately from exposure).

---

## Step 5: Three ways to describe an error to a client

You get to choose (per error) how the client should learn *what* went wrong. None is required, and you can combine them freely.

### Option A — Structured `Violations` (the client builds its own text)

Good when you have a frontend that wants to translate error messages itself, or render them next to specific form fields.

```go
err := xerr.New(xerr.CodeValidationFailed,
    xerr.WithViolation("email", xerr.ErrorReasonInvalidFormat),
    xerr.WithViolation("password", xerr.ErrorReasonTooShort, xerr.P("min", 8)),
)
```

```json
{
  "code": "VALIDATION_FAILED",
  "violations": [
    { "field": "email", "reason": "invalid_format" },
    { "field": "password", "reason": "too_short", "params": { "min": 8 } }
  ]
}
```

- `field` — which field the problem is about.
- `reason` — a **fixed, stable string** from the [`ErrorReason` enum](#reference-built-in-violation-reasons). Give this list to your frontend team once, they build one translation table (`too_short` → "must be at least {min} characters" in every supported language), and it never needs to change again, no matter how the English wording evolves.
- `params` — the dynamic values a translated message needs, e.g. `{"min": 8}`. This is the one place that's intentionally *not* a fixed enum, because it has to carry arbitrary numbers/strings.

If you already have a list of violations built by something else (e.g. a third-party validation library), attach them all at once instead of one by one:

```go
var violations []xerr.Violation
// ... fill violations from your validator ...
err := xerr.New(xerr.CodeValidationFailed, xerr.WithViolations(violations...))
```

`WithViolation` and `WithViolations` can be combined and both can be called more than once — every call appends.

### Option B — A direct `Message` (the backend decides the exact text)

Good for one-off cases where there's no "field", just a sentence — or when you don't have (or don't trust) a frontend translation layer for this specific case.

```go
err := xerr.New(xerr.CodeConflict, xerr.WithMessage("this coupon has already been redeemed"))
```

```json
{ "code": "CONFLICT", "message": "this coupon has already been redeemed" }
```

### Using both together

Nothing stops you from setting both — e.g. a short message for a toast notification, plus violations for inline field errors:

```go
err := xerr.New(xerr.CodeValidationFailed,
    xerr.WithMessage("please fix the highlighted fields"),
    xerr.WithViolation("email", xerr.ErrorReasonInvalidFormat),
)
```

Remember: both `Message` and `Violations` only reach the client if `Exposed()` is true (see [Step 3](#step-3-understand-kind-and-who-gets-to-see-what)). Set them on any error, regardless of `Kind` — they just won't be serialized if that error turns out to be unsafe to expose.

### Option C — Error-level `Params` (dynamic detail with no natural field)

`Violation.Params` carries dynamic values for one field's rule (`{"min": 8}` next to an email violation). Sometimes the dynamic detail isn't about a single field at all — it's about the error as a whole: *which* resource, *what* limit was hit. That's what `WithParam` is for:

```go
err := xerr.New(xerr.CodeInvalidParam,
    xerr.WithMessage("seat limit exceeded"),
    xerr.WithParam("resource", "seats"),
    xerr.WithParam("max", 5),
)
```

```json
{ "code": "INVALID_PARAMETER", "message": "seat limit exceeded", "params": { "resource": "seats", "max": 5 } }
```

Same rule as `Message` and `Violations`: `params` is only serialized when `Exposed()` is true, and `.Params()` always has the full value server-side regardless.

---

## Step 6: `Diagnostics` — notes that never leave the server

`Diagnostics` are for internal debugging notes that must **never** reach a client, no matter what — not even if the error is otherwise `Exposed()`. Use them for things like which operation was running, or an internal resource id:

```go
err := xerr.New(xerr.CodeDatabaseError,
    xerr.WithDiagnostic(xerr.DiagnosticOperation, "CreateUser"),
    xerr.WithDiagnostic(xerr.DiagnosticResource, "users"),
)
```

They show up in `err.Error()` (for plain-text logs) and in `LogValue()` (for `log/slog`), but `json.Marshal(err)` never includes them, under any circumstance. Built-in keys are `DiagnosticOperation`, `DiagnosticReason`, `DiagnosticResource` — `DiagnosticKey` is just a `string` type, so you can define your own too.

**Rule of thumb:** if it must never leak, it's a `Diagnostic`. If it's fine to leak *when the error is Exposed*, it's a `Message`, a `Violation`, or a `Param`.

---

## Step 7: Wrapping an existing error

`New` starts a fresh error. `Wrap` does the same thing but also attaches an existing Go error as the *cause*, which is preserved for logs and for `errors.Is` / `errors.As`:

```go
row := db.QueryRow("SELECT ...")
if err := row.Scan(&user); err != nil {
    return xerr.Wrap(err, xerr.CodeRecordNotFound)
}
```

`Wrap(nil, ...)` returns `nil` — handy when you write `return xerr.Wrap(err, ...)` at the end of a function and `err` might already be `nil`.

You can see the wrapped cause with `.Err()`, and it also shows up automatically at the end of `.Error()`:

```go
err.Err()   // the original *sql.ErrNoRows (or whatever it was)
err.Error() // "RECORD_NOT_FOUND (infrastructure): sql: no rows in result set"
```

`.Err()` is **never** included in the JSON response — same rule as `Diagnostics`.

---

## Step 8: The full DDD pattern — repository → service → HTTP

This is the pattern the whole library is built around: an infrastructure failure gets wrapped where it happens, then **translated** into a safe domain error one layer up, while the original stays attached for your logs.

```go
// --- repository layer ---
// A raw database error. Kind defaults to KindInfrastructure (unsafe),
// so even if this accidentally bubbled all the way to an HTTP response
// by itself, nothing about the database would leak.
func (r *UserRepo) FindByID(ctx context.Context, id string) (*User, error) {
    var u User
    if err := r.db.First(&u, "id = ?", id).Error; err != nil {
        return nil, xerr.Wrap(err, xerr.CodeRecordNotFound,
            xerr.WithDiagnostic(xerr.DiagnosticOperation, "UserRepo.FindByID"),
        )
    }
    return &u, nil
}

// --- service layer ---
// Translate the infra failure into a well-known, client-safe domain
// error. The original error stays reachable through Unwrap/errors.As.
func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    u, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, xerr.New(xerr.CodeNotFound,
            xerr.WithMessage("user not found"),
            xerr.WithErr(err), // chain preserved
        )
    }
    return u, nil
}

// --- HTTP layer ---
func GetUserHandler(w http.ResponseWriter, r *http.Request) {
    user, err := userService.GetUser(r.Context(), id)
    if err != nil {
        xe, _ := xerr.FromError(err)
        slog.Error("get user failed", "err", xe) // full detail, safely
        w.WriteHeader(xe.HTTPStatus())
        json.NewEncoder(w).Encode(xe) // {"code":"RESOURCE_NOT_FOUND","message":"user not found"}
        return
    }
    // ...
}
```

What the client sees: `{"code":"RESOURCE_NOT_FOUND","message":"user not found"}` — clean and safe.

What your logs see (via `slog.Error("...", "err", xe)`): the full chain, including the original `sql: no rows in result set` and the `UserRepo.FindByID` diagnostic — because `slog`'s view of an `*xerr.Error` is a completely different, unfiltered view from the JSON one. That's covered next.

---

## Step 9: `errors.Is`, `errors.As`, and `FromError`

`*xerr.Error` works with Go's standard `errors` package.

**`errors.Is`** — compares by `Code` only (not message, not violations — those are runtime detail that shouldn't matter for identity checks):

```go
if errors.Is(err, xerr.New(xerr.CodeNotFound)) {
    // handle "not found" generically, wherever it came from
}
```

**`errors.As` / `FromError`** — pull the concrete `*xerr.Error` back out of an error chain (e.g. after it's been wrapped by `fmt.Errorf("...: %w", err)` somewhere):

```go
var xe *xerr.Error
if errors.As(err, &xe) {
    fmt.Println(xe.Code(), xe.Kind())
}

// FromError is the same thing, just shorter to write:
xe, ok := xerr.FromError(err)
```

**`Unwrap`** — also works, since `xerr.Error` implements the standard `Unwrap() error` method, so `errors.Unwrap(err)` walks the chain one step at a time just like it would for any wrapped error.

---

## Step 10: Logging with `log/slog`

`*xerr.Error` implements `slog.LogValuer`. Pass it straight to your logger and every field gets logged as structured data — **including the fields the client never sees** (`Kind`, `Diagnostics`, the wrapped cause, and the stack trace if you captured one):

```go
slog.Error("request failed", "err", xerr.Wrap(dbErr, xerr.CodeDatabaseError,
    xerr.WithDiagnostic(xerr.DiagnosticOperation, "CreateUser"),
))
```

Produces (with the JSON handler):

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

No manual `"code", xe.Code(), "kind", xe.Kind(), ...` boilerplate needed at every call site — just log the error value itself.

---

## Step 11: Call-stack capture

`WithStack()` records the current call stack, so later you can see exactly *where* an error was created — very useful for infrastructure errors you're trying to debug.

```go
if err != nil {
    return xerr.Wrap(err, xerr.CodeExternalService, xerr.WithStack())
}
```

```go
fmt.Println(xe.Stack())
// github.com/you/app/internal/payment.(*Client).Charge
//     /home/you/app/internal/payment/client.go:42
// github.com/you/app/internal/service.(*OrderService).Pay
//     /home/you/app/internal/service/order.go:88
// ...
```

It's **opt-in on purpose**. Capturing a stack costs a little bit of time, and most errors (a failed validation that happens on every request) don't need one. Reach for it on the errors you'll actually want to debug — typically the `KindInfrastructure` / `KindUnknown` ones.

`Stack()` is log-only: it's never part of the JSON response, and it's deliberately left out of `Error()`'s one-line output too (so your logs don't get a giant multi-line blob by accident) — call `.Stack()` explicitly, or just log the error through `slog` (see [Step 10](#step-10-logging-with-logslog)), which includes it automatically whenever one was captured.

---

## Step 12: Recovering from panics

`Recover` turns a recovered panic into a normal `*xerr.Error`, with a stack trace captured automatically — because a stack trace is the entire reason you'd want to catch a panic in the first place.

```go
func (s *Service) Handle(ctx context.Context, req Request) (resp Response, err error) {
    defer func() {
        if v := recover(); v != nil {
            err = xerr.Recover(v)
        }
    }()

    // ... code that might panic ...
    return doSomething(req)
}
```

The resulting error uses `CodePanic` (`KindUnknown` — unsafe by default, so a client only ever sees a generic internal-error response, never the panic message). `Recover(nil)` returns `nil`, so it's safe to call unconditionally right after `recover()`.

---

## Step 13: `DefaultMessage()` — a plain-English fallback

Sometimes there's no frontend to build a message from `Reason` + `Params` — a CLI tool, a log meant for a human to read directly, a quick prototype. `DefaultMessage()` gives you a best-effort English sentence:

```go
err := xerr.New(xerr.CodeValidationFailed,
    xerr.WithViolation("email", xerr.ErrorReasonRequired),
    xerr.WithViolation("password", xerr.ErrorReasonTooShort, xerr.P("min", 8)),
)

fmt.Println(err.DefaultMessage())
// "email is required; password must be at least 8 characters"
```

The rule it follows: use the explicit `Message` if one was set; otherwise join every `Violation`'s own `DefaultMessage()`; otherwise fall back to a generic sentence based on `Kind` (the code itself if the error is safe, or just `"something went wrong"` if it isn't).

**Important:** this is *not* a translation/localization system. There's no language catalog, no locale switching, nothing pluggable — it only ever produces English. Wherever you actually have a frontend, prefer letting it build the message itself from `Reason` + `Params` (that's exactly what that enum exists for). Use `DefaultMessage()` only where that's not an option.

---

## Step 14: HTTP status codes

Every error already knows its HTTP status, derived from its `Code`:

```go
xerr.New(xerr.CodeNotFound).HTTPStatus()        // 404
xerr.New(xerr.CodeDatabaseError).HTTPStatus()   // 500
xerr.New(xerr.CodeTooManyRequests).HTTPStatus() // 429
```

See the [full table](#reference-built-in-codes) below for every built-in code.

---

## Step 15: Wiring it into an HTTP framework

A minimal example with [Gin](https://github.com/gin-gonic/gin), as a central error-handling middleware:

```go
func ErrorHandler(c *gin.Context) {
    c.Next()

    if len(c.Errors) == 0 {
        return
    }

    err := c.Errors.Last().Err

    xe, ok := xerr.FromError(err)
    if !ok {
        // some code returned a plain error, not an *xerr.Error — treat
        // it as an unknown, unsafe-to-expose failure
        xe = xerr.New(xerr.CodeInternalError, xerr.WithErr(err))
    }

    slog.Error("request failed", "err", xe)
    c.JSON(xe.HTTPStatus(), xe)
}
```

The same pattern works with `net/http`, Echo, Fiber, chi, etc. — the important part is always the same two lines: log the full `*xerr.Error`, then `json.Marshal`/serialize that same value for the response body.

---

## Step 16: Swagger / OpenAPI docs

For `swaggo/swag`-style generators, use the dedicated DTOs — they mirror the exact JSON shape without exposing any internal fields:

```go
// @Failure 400 {object} xerr.SwaggerErrOutput
// @Failure 404 {object} xerr.SwaggerErrOutput
// @Failure 500 {object} xerr.SwaggerErrOutput
```

```go
type SwaggerErrOutput struct {
    Code       string                   `json:"code" example:"VALIDATION_FAILED"`
    Message    string                   `json:"message,omitempty" example:"invalid request body"`
    Params     map[string]any           `json:"params,omitempty" example:"resource:product"`
    Violations []SwaggerViolationOutput `json:"violations,omitempty"`
}

type SwaggerViolationOutput struct {
    Field  string         `json:"field" example:"email"`
    Reason string         `json:"reason" example:"invalid_format"`
    Params map[string]any `json:"params,omitempty" example:"min:8"`
}
```

### Making `code` render as an enum

`SwaggerErrOutput.Code` is a plain `string`, not a generated enum — `xerr` only knows its own built-in codes plus whatever your application registered with `RegisterCode`, so it can't bake a closed set into this struct without also knowing every domain code your application defines. If you want `code` to render as an OpenAPI enum, build the value list yourself and apply it as a `swaggo` `enums:"..."` tag (or an equivalent doc-generation step) on your own copy of the struct:

```go
func exposedCodeNames() []string {
    names := make([]string, 0)
    for code := range xerr.ExposedCodes() { // built-ins + everything you registered
        names = append(names, string(code))
    }
    sort.Strings(names)
    return names
}
```

`ExposedCodes()` returns every code whose *default* `Kind` is safe to expose — see [Step 2](#step-2-understand-code) for `RegisterCode` and the note there about `WithExpose` overrides not being reflected here.

---

## Reference: built-in codes

Every code below has a default `Kind` (safe to expose or not) and a default HTTP status, both overridable with `WithKind` / `WithExpose`.

### System / Internal

| `Code` | `Kind` | HTTP Status |
|---|---|---|
| `CodeInternalError` (`INTERNAL_SERVER_ERROR`) | `KindUnknown` | 500 |
| `CodeUnknownError` (`UNKNOWN_ERROR`) | `KindUnknown` | 500 |
| `CodeServiceUnavailable` (`SERVICE_UNAVAILABLE`) | `KindInfrastructure` | 503 |
| `CodePanic` (`PANIC`) | `KindUnknown` | 500 |

### Request

| `Code` | `Kind` | HTTP Status |
|---|---|---|
| `CodeBadRequest` (`BAD_REQUEST`) | `KindDomain` | 400 |
| `CodeValidationFailed` (`VALIDATION_FAILED`) | `KindDomain` | 400 |
| `CodeMalformedJSON` (`MALFORMED_JSON`) | `KindDomain` | 400 |
| `CodeMissingField` (`MISSING_REQUIRED_FIELD`) | `KindDomain` | 400 |
| `CodeInvalidParam` (`INVALID_PARAMETER`) | `KindDomain` | 400 |

### Authentication

| `Code` | `Kind` | HTTP Status |
|---|---|---|
| `CodeUnauthorized` (`UNAUTHORIZED`) | `KindApplication` | 401 |
| `CodeInvalidCredentials` (`INVALID_CREDENTIALS`) | `KindApplication` | 401 |
| `CodeInvalidToken` (`INVALID_TOKEN`) | `KindApplication` | 401 |
| `CodeExpiredToken` (`TOKEN_EXPIRED`) | `KindApplication` | 401 |
| `CodeRefreshTokenInvalid` (`INVALID_REFRESH_TOKEN`) | `KindApplication` | 401 |

### Authorization

| `Code` | `Kind` | HTTP Status |
|---|---|---|
| `CodeForbidden` (`FORBIDDEN`) | `KindApplication` | 403 |
| `CodePermissionDenied` (`PERMISSION_DENIED`) | `KindApplication` | 403 |
| `CodeInsufficientScope` (`INSUFFICIENT_SCOPE`) | `KindApplication` | 403 |

### Resource

| `Code` | `Kind` | HTTP Status |
|---|---|---|
| `CodeNotFound` (`RESOURCE_NOT_FOUND`) | `KindDomain` | 404 |
| `CodeAlreadyExists` (`RESOURCE_ALREADY_EXISTS`) | `KindDomain` | 409 |
| `CodeResourceLocked` (`RESOURCE_LOCKED`) | `KindDomain` | 423 |
| `CodeResourceDeleted` (`RESOURCE_DELETED`) | `KindDomain` | 410 |

### Business logic

| `Code` | `Kind` | HTTP Status |
|---|---|---|
| `CodeConflict` (`CONFLICT`) | `KindDomain` | 409 |
| `CodeOperationFailed` (`OPERATION_FAILED`) | `KindDomain` | 422 |
| `CodeInvalidState` (`INVALID_STATE`) | `KindDomain` | 409 |

### Rate limit / security

| `Code` | `Kind` | HTTP Status |
|---|---|---|
| `CodeTooManyRequests` (`TOO_MANY_REQUESTS`) | `KindApplication` | 429 |

### Storage / database

| `Code` | `Kind` | HTTP Status |
|---|---|---|
| `CodeDatabaseError` (`DATABASE_ERROR`) | `KindInfrastructure` | 500 |
| `CodeDuplicateKey` (`DUPLICATE_KEY`) | `KindInfrastructure` | 409 |
| `CodeForeignKeyError` (`FOREIGN_KEY_CONSTRAINT`) | `KindInfrastructure` | 409 |
| `CodeRecordNotFound` (`RECORD_NOT_FOUND`) | `KindInfrastructure` | 404 |

> Notice `CodeRecordNotFound` (infra, hidden) vs `CodeNotFound` (domain, shown) — this pair is exactly the repository-vs-domain distinction from [Step 8](#step-8-the-full-ddd-pattern--repository--service--http).

### External / network

| `Code` | `Kind` | HTTP Status |
|---|---|---|
| `CodeNetworkError` (`NETWORK_ERROR`) | `KindInfrastructure` | 502 |
| `CodeTimeout` (`TIMEOUT`) | `KindInfrastructure` | 504 |
| `CodeExternalService` (`EXTERNAL_SERVICE_ERROR`) | `KindInfrastructure` | 502 |

You are not limited to these — define your own `Code` constants freely (`const CodeCouponExpired xerr.Code = "COUPON_EXPIRED"`) and call `RegisterCode` to give it a real `Kind` and HTTP status (see [Step 2](#step-2-understand-code)). An unregistered code defaults to `KindUnknown` (hidden) and HTTP 500, or you can skip registration and just override per error with `WithKind`/`WithExpose` instead.

---

## Reference: built-in violation reasons

Use these with `WithViolation(field, reason, params...)`. `params` in the table below are the `Param` keys that [`DefaultMessage()`](#step-13-defaultmessage--a-plain-english-fallback) understands for that reason — your own frontend translation table can use the same keys, or different ones entirely, since this is just a suggestion, not enforced by the type system.

| `ErrorReason` | String value | Relevant params |
|---|---|---|
| `ErrorReasonRequired` | `required` | — |
| `ErrorReasonInvalidFormat` | `invalid_format` | — |
| `ErrorReasonInvalidValue` | `invalid_value` | `allowed` |
| `ErrorReasonTooShort` | `too_short` | `min` |
| `ErrorReasonTooLong` | `too_long` | `max` |
| `ErrorReasonTooSmall` | `too_small` | `min` |
| `ErrorReasonTooLarge` | `too_large` | `max` |
| `ErrorReasonMismatch` | `mismatch` | — |
| `ErrorReasonAlreadyExists` | `already_exists` | — |
| `ErrorReasonNotFound` | `not_found` | — |
| `ErrorReasonCorrupted` | `corrupted` | — |
| `ErrorReasonExpired` | `expired` | — |

---

## Reference: full API

### Constructors

| Function | What it does |
|---|---|
| `New(code Code, opts ...ErrorOption) *Error` | Create a fresh error. `Kind` defaults from `code.Kind()`. |
| `Wrap(err error, code Code, opts ...ErrorOption) *Error` | Same as `New`, plus attaches `err` as the cause. Returns `nil` if `err` is `nil`. |
| `Recover(v any, opts ...ErrorOption) *Error` | Converts a recovered panic value (from `recover()`) into an error with `CodePanic` and a captured stack. Returns `nil` if `v` is `nil`. |
| `FromError(err error) (*Error, bool)` | Finds an `*Error` anywhere in `err`'s chain. Thin wrapper over `errors.As`. |

### Registration

| Function | What it does |
|---|---|
| `RegisterCode(code Code, kind Kind, httpStatus int)` | Registers a new `Code` with its `Kind` and HTTP status. Panics if `code` is already registered (built-in or previously registered), or if `code`/`kind`/`httpStatus` is malformed (empty code, unknown `Kind`, status outside 400-599). See [Step 2](#step-2-understand-code). |
| `ExposedCodes() map[Code]Kind` | Every registered code whose default `Kind` is safe to expose (`Kind.Safe()`) — built-ins and anything from `RegisterCode`. Handy for a Swagger enum or a sync test. See [Step 16](#step-16-swagger--openapi-docs). |

### Options (pass any combination to `New`/`Wrap`/`Recover`)

| Option | Effect |
|---|---|
| `WithMessage(string)` | Sets a ready-to-display message. Sent to client only if `Exposed()`. |
| `WithParam(key string, value any)` | Sets one error-level param (e.g. `resource`, `max`). Sent to client only if `Exposed()`. Repeatable; a later call with the same key overwrites it. |
| `WithViolation(field string, reason ErrorReason, params ...Param)` | Appends one field violation. Repeatable. |
| `WithViolations(vs ...Violation)` | Appends a pre-built batch of violations. |
| `WithErr(error)` | Attaches the wrapped cause. Log-only, never serialized. |
| `WithDiagnostic(key DiagnosticKey, value string)` | Attaches an internal-only debug note. Log-only, never serialized. |
| `WithKind(Kind)` | Overrides the code's default `Kind`. |
| `WithExpose(bool)` | Overrides whether `Message`/`Violations`/`Params` are sent to a client. |
| `WithStack()` | Captures the current call stack for `.Stack()`. |

### Types

| Type | Shape |
|---|---|
| `Error` | The error type itself. Implements `error`, `Unwrap() error`, `Is(error) bool`, `MarshalJSON`, `slog.LogValuer`. |
| `Code` | `string`. Has `.String()`, `.HTTPStatus() int`, `.Kind() Kind`. |
| `Kind` | `string`: `KindDomain`, `KindApplication`, `KindInfrastructure`, `KindUnknown`. Has `.String()`, `.Safe() bool`. |
| `Violation` | `struct { Field string; Reason ErrorReason; Params map[string]any }`. Has `.DefaultMessage() string`. |
| `Param` | `struct { Key string; Value any }`. Build with `P(key, value)`. |
| `ErrorReason` | `string` enum — see [table above](#reference-built-in-violation-reasons). |
| `DiagnosticKey` | `string`. Built-ins: `DiagnosticOperation`, `DiagnosticReason`, `DiagnosticResource`. |
| `SwaggerErrOutput`, `SwaggerViolationOutput` | Plain DTOs for Swagger/OpenAPI doc generators. |

### `*Error` methods

| Method | Returns | Notes |
|---|---|---|
| `Code() Code` | The real code | Always safe to log; `MarshalJSON` substitutes `CodeInternalError` when not `Exposed()`. |
| `Kind() Kind` | The classification | Log-only, never in JSON. |
| `Message() string` | The explicit message, if set | Sent to client only if `Exposed()`. |
| `Params() map[string]any` | A defensive copy of error-level params | Sent to client only if `Exposed()`. |
| `Violations() []Violation` | A defensive copy | Sent to client only if `Exposed()`. |
| `Diagnostics() map[DiagnosticKey]string` | A defensive copy | Log-only, never in JSON, even if `Exposed()`. |
| `Err() error` | The wrapped cause, if any | Log-only, never in JSON. |
| `Stack() string` | Formatted call stack, or `""` | Log-only, never in JSON. Only set if `WithStack()`/`Recover` was used. |
| `Exposed() bool` | Whether `Message`/`Params`/`Violations` reach the client | Defaults to `Kind().Safe()`; overridden by `WithExpose`. |
| `HTTPStatus() int` | HTTP status for `Code()` | |
| `DefaultMessage() string` | Best-effort English fallback text | See [Step 13](#step-13-defaultmessage--a-plain-english-fallback). |
| `Error() string` | One-line description for plain-text logs | Always full detail; excludes `Stack()`. |
| `Unwrap() error` | The wrapped cause | For `errors.Unwrap`/`errors.Is`/`errors.As`. |
| `Is(target error) bool` | Whether `target` has the same `Code()` | For `errors.Is`. |
| `MarshalJSON() ([]byte, error)` | Client-safe JSON | See [Step 3](#step-3-understand-kind-and-who-gets-to-see-what). |
| `LogValue() slog.Value` | Structured log view | See [Step 10](#step-10-logging-with-logslog). Unfiltered — includes everything, unlike `MarshalJSON`. |

---

## Gotcha: typed nil

`Wrap(nil, ...)` and `Recover(nil)` return a literal `nil` of the concrete type `*xerr.Error`. That's completely fine as long as you keep using that concrete type — but Go has a well-known trap if you assign it to a plain `error` interface variable:

```go
var err error = xerr.Wrap(nil, xerr.CodeInternalError)
err != nil // true! even though there's no real error
```

This happens because an `error` interface value is only truly `nil` when *both* its type and its value are nil — and here the type (`*xerr.Error`) is not nil, only the value inside it is. This is a general Go language behavior, not something specific to `xerr` — just keep it in mind at the exact point where a `*xerr.Error` gets assigned to a plain `error`. In practice, this is rarely an issue: `return xerr.Wrap(err, ...)` from a function that already declares an `error` return type has this exact shape, so always check the *original* `err` for `nil` first, the same way you would before calling `Wrap` at all.

---

## License

MIT
