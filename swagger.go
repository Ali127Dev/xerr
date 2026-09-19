package xerr

// SwaggerErrOutput represents the public JSON structure of xerr.Error.
//
// This is used only for Swagger and documentation.
// Actual API responses are produced by MarshalJSON of xerr.Error.
//
// Code is typed as a plain string here, not an enum: xerr only knows its
// own built-in codes plus whatever the owning application registers via
// RegisterCode, so it cannot bake a closed set into this struct without
// also knowing every domain code the application defines. If you want
// "code" to render as an OpenAPI enum, build the value list yourself —
// combine xerr.ExposedCodes() with your own domain codes — and apply it
// as a swaggo `enums:"..."` tag (or an equivalent doc-generation step)
// on your own copy of this struct.
type SwaggerErrOutput struct {
	Code       string                   `json:"code" example:"VALIDATION_FAILED"`
	Message    string                   `json:"message,omitempty" example:"invalid request body"`
	Params     map[string]any           `json:"params,omitempty" swaggertype:"object" example:"resource:product"`
	Violations []SwaggerViolationOutput `json:"violations,omitempty"`
} // @name Error

// SwaggerViolationOutput represents the public JSON structure of a
// xerr.Violation, for Swagger and documentation purposes only.
type SwaggerViolationOutput struct {
	Field  string         `json:"field" example:"email"`
	Reason string         `json:"reason" example:"invalid_format"`
	Params map[string]any `json:"params,omitempty" swaggertype:"object" example:"min:8"`
} // @name ErrorViolation
