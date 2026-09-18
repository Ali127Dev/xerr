package xerr

// SwaggerErrOutput represents the public JSON structure of xerr.Error.
//
// This is used only for Swagger and documentation.
// Actual API responses are produced by MarshalJSON of xerr.Error.
type SwaggerErrOutput struct {
	Code       string                   `json:"code" example:"VALIDATION_FAILED"`
	Message    string                   `json:"message,omitempty" example:"invalid request body"`
	Violations []SwaggerViolationOutput `json:"violations,omitempty"`
} // @name Error

// SwaggerViolationOutput represents the public JSON structure of a
// xerr.Violation, for Swagger and documentation purposes only.
type SwaggerViolationOutput struct {
	Field  string         `json:"field" example:"email"`
	Reason string         `json:"reason" example:"invalid_format"`
	Params map[string]any `json:"params,omitempty" swaggertype:"object" example:"min:8"`
} // @name ErrorViolation
