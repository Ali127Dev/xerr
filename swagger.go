package xerr

// SwaggerErrOutput represents the public JSON structure of xerr.Error.
//
// This is used only for Swagger and documentation.
// Actual API responses are produced by MarshalJSON of xerr.Error.
type SwaggerErrOutput struct {
	Code    string         `json:"code" example:"INVALID_ARGUMENT"`
	Message string         `json:"message,omitempty" example:"invalid request body"`
	Meta    map[string]any `json:"meta,omitempty" swaggertype:"object,string" example:"field:username,error:invalid_length"`
} // @name Error
