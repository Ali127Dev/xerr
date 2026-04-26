// Package xerr provides a structured, extensible, and safe error handling system.
//
// Key features:
//   - Strongly typed machine-readable error codes
//   - Human-readable messages
//   - Optional metadata for debugging or business context
//   - JSON-safe responses (no internal leakage)
//   - Error wrapping compatible with errors.Is / errors.As
//   - Swagger-friendly error output model
//
// This package is designed for production-grade service and microservices.
package xerr
