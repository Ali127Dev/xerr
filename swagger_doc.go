package xerr

type SwaggerErrOutput struct {
	Code    Code           `json:"code" example:"BAD_REQUEST"`
	Message string         `json:"message,omitempty" example:"invalid request body"`
	Meta    map[string]any `json:"meta,omitempty" swaggertype:"object,string" example:"field=username"`
} //	@name	Error
