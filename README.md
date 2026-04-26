# xerr — Structured, Elegant, Production‑Grade Error Handling for Go 🚀🔥

xerr is a clean, robust, and enterprise‑grade error handling framework for Go applications. It provides a predictable, expressive, and type‑safe way to represent errors across your entire stack — from domain logic all the way to HTTP delivery layers.

Built for real‑world backend architectures (DDD, Clean Architecture, microservices), xerr helps you:

- 🧩 Standardize error shapes across all layers
- 🎯 Attach metadata for debugging and observability
- 📡 Cleanly expose errors in JSON form over HTTP
- 🧠 Work seamlessly with `errors.Is` / `errors.As`
- 🧵 Preserve underlying error chains
- ⚡ Enrich errors dynamically with functional options
- 🔒 Safely hide internal error details from clients

---

## ✨ Features

### 🔐 Private Fields, Public Getters

Ensures strict encapsulation and prevents accidental mutation.

### 🧰 Functional Options (WithMessage, WithStatus, WithMeta, WithErr)

Build expressive, context‑rich errors without clutter.

### 🪢 Full Support for Native Error Wrapping

`errors.Is`, `errors.As`, and `Unwrap` work flawlessly.

### 🎨 Clean JSON Serialization

Uses a custom `MarshalJSON` method to expose **only safe fields**.

### 🧭 Automatic HTTP Status Mapping

Every error maps to an appropriate HTTP status code.

### 🧼 Minimal, Focused, Zero Dependencies

Pure Go. Zero bells and whistles. Maximum clarity.

---

## 📦 Installation

```sh
go get github.com/Ali127Dev/xerr
```

---

## 🚀 Quick Start

```go
err := xerr.New(
    xerr.CodeValidationFailed,
    xerr.WithMessage("email format is invalid"),
    xerr.WithMeta("field", "email"),
)
```

Wrapped error:

```go
if err := db.First(&u).Error; err != nil {
    return xerr.Wrap(err, xerr.CodeNotFound,
        xerr.WithMessage("user not found"),
        xerr.WithMeta("user_id", userID),
    )
}
```

---

## 🔄 JSON Output

Errors automatically serialize into a clean client‑safe format:

```json
{
  "code": "VALIDATION_FAILED",
  "message": "email format is invalid",
  "meta": { "field": "email" }
}
```

Internal fields like `err` and `status` are hidden.

---

## 🧩 Full API Reference

### `New(code Code, opts ...ErrorOption)`

Creates a fresh structured error.

### `Wrap(err error, code Code, opts ...ErrorOption)`

Wraps an underlying Go error with your business/domain error.

### `WithMessage(string)`

Human‑readable error description.

### `WithStatus(int)`

Override HTTP status mapping.

### `WithMeta(key string, value any)`

Attach arbitrary structured metadata.

### `WithErr(error)`

Attach an underlying error without exposing it.

---

## 🧵 How Error Messages Are Formatted

The error string prioritizes fields in this order:

1. Custom message
2. Underlying `err.Error()`
3. The code value

Example:

```go
fmt.Println(err.Error())
// VALIDATION_FAILED: email format is invalid
```

---

## 🌐 Using with Gin (Recommended)

```go
func ErrorHandler(c *gin.Context) {
    c.Next()

    if len(c.Errors) == 0 {
        return
    }

    err := c.Errors.Last().Err

    xe, ok := err.(*xerr.Error)
    if !ok { // convert native errors automatically
        xe = xerr.Wrap(err, xerr.CodeInternal)
    }

    c.JSON(xe.HTTPStatus(), xe)
}
```

---

## 📘 Swagger Integration

xerr provides a dedicated DTO for API documentation to ensure consistent and properly documented error responses in Swagger / OpenAPI.

```go
type SwaggerErrOutput struct {
    Code    Code           `json:"code" example:"INVALID_ARGUMENT"`
    Message string         `json:"message,omitempty" example:"invalid request body"`
    Meta    map[string]any `json:"meta,omitempty" example:"field=username"`
} // @name Error
```

### 🎯 Why a Separate Swagger Model?

Although `xerr.Error` already controls JSON serialization via `MarshalJSON`,

Swagger generators (like `swaggo/swag`) rely on struct definitions for schema generation.

SwaggerErrOutput exists purely for documentation purposes and ensures:

- ✅ Stable and predictable OpenAPI schema
- ✅ Accurate example values in Swagger UI
- ✅ No leakage of internal fields (err, status)
- ✅ Clear API contract for frontend teams

### 🧩 Usage Example in Handler

```go
// @Failure 400 {object} xerr.SwaggerErrOutput
// @Failure 401 {object} xerr.SwaggerErrOutput
// @Failure 500 {object} xerr.SwaggerErrOutput
```

```json
{
  "code": "INVALID_ARGUMENT",
  "message": "invalid request body",
  "meta": {
    "field": "username"
  }
}

```

---

## 🧪 Why xerr?

- 📘 Perfect for Clean Architecture
- 🧱 Works beautifully with domain‑driven design
- 🚦 Rich metadata for observability
- 🎯 Predictable structure for frontend consumption
- 📡 Ideal for microservices and HTTP APIs

---

## ❤️ Contributing

Pull requests and feature ideas are welcome! Help make error handling in Go beautiful.

---

## 📝 License

MIT — Do whatever you want. Build cool stuff.

---

## 🌟 Final Note

xerr is built for developers who want **consistency, clarity, and confidence** across their entire Go backend.

If that’s you — welcome aboard 🚀🔥
