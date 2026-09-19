package xerr_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Ali127Dev/xerr/v2"
)

// Shared field names reused across this package's test tables.
const (
	fieldEmail    = "email"
	fieldPassword = "password"
	mutatedValue  = "mutated"
)

func TestNew_DefaultKindFromCode(t *testing.T) {
	tests := []struct {
		code xerr.Code
		want xerr.Kind
	}{
		{xerr.CodeValidationFailed, xerr.KindDomain},
		{xerr.CodeUnauthorized, xerr.KindApplication},
		{xerr.CodeDatabaseError, xerr.KindInfrastructure},
		{xerr.CodeInternalError, xerr.KindUnknown},
		{xerr.Code("SOME_UNREGISTERED_CODE"), xerr.KindUnknown},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			e := xerr.New(tt.code)
			if e.Kind() != tt.want {
				t.Fatalf("Kind() = %q, want %q", e.Kind(), tt.want)
			}
		})
	}
}

func TestError_ExposedDefaultsByKind(t *testing.T) {
	safe := xerr.New(xerr.CodeValidationFailed, xerr.WithMessage("bad input"))
	if !safe.Exposed() {
		t.Fatal("domain-kind error should be exposed by default")
	}

	unsafe := xerr.New(xerr.CodeDatabaseError, xerr.WithMessage("connection refused"))
	if unsafe.Exposed() {
		t.Fatal("infrastructure-kind error should not be exposed by default")
	}
}

func TestError_WithExposeOverride(t *testing.T) {
	forcedSafe := xerr.New(xerr.CodeDatabaseError, xerr.WithExpose(true))
	if !forcedSafe.Exposed() {
		t.Fatal("WithExpose(true) should override the infrastructure default")
	}

	forcedUnsafe := xerr.New(xerr.CodeValidationFailed, xerr.WithExpose(false))
	if forcedUnsafe.Exposed() {
		t.Fatal("WithExpose(false) should override the domain default")
	}
}

func TestError_WithKindOverride(t *testing.T) {
	e := xerr.New(xerr.CodeDatabaseError, xerr.WithKind(xerr.KindDomain))
	if e.Kind() != xerr.KindDomain {
		t.Fatalf("Kind() = %q, want %q", e.Kind(), xerr.KindDomain)
	}
	if !e.Exposed() {
		t.Fatal("overridden domain kind should be exposed by default")
	}
}

func TestError_MarshalJSON_Exposed(t *testing.T) {
	e := xerr.New(xerr.CodeValidationFailed,
		xerr.WithMessage("email format is invalid"),
		xerr.WithViolation(fieldEmail, xerr.ErrorReasonInvalidFormat),
		xerr.WithViolation(fieldPassword, xerr.ErrorReasonTooShort, xerr.P("min", 8)),
	)

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got["code"] != string(xerr.CodeValidationFailed) {
		t.Fatalf("code = %v, want %v", got["code"], xerr.CodeValidationFailed)
	}
	if got["message"] != "email format is invalid" {
		t.Fatalf("message = %v", got["message"])
	}

	violations, ok := got["violations"].([]any)
	if !ok || len(violations) != 2 {
		t.Fatalf("violations = %v", got["violations"])
	}

	second, ok := violations[1].(map[string]any)
	if !ok {
		t.Fatalf("violations[1] = %v", violations[1])
	}
	params, ok := second["params"].(map[string]any)
	if !ok || params["min"] != float64(8) {
		t.Fatalf("violations[1].params = %v", second["params"])
	}
}

func TestError_WithParam_ExposedInJSON(t *testing.T) {
	e := xerr.New(xerr.CodeInvalidParam,
		xerr.WithMessage("usage limit exceeded"),
		xerr.WithParam("resource", "seats"),
		xerr.WithParam("max", 5),
	)

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	params, ok := got["params"].(map[string]any)
	if !ok {
		t.Fatalf("params = %v", got["params"])
	}
	if params["resource"] != "seats" || params["max"] != float64(5) {
		t.Fatalf("params = %v", params)
	}
}

func TestError_WithParam_HiddenWhenNotExposed(t *testing.T) {
	e := xerr.New(xerr.CodeDatabaseError, xerr.WithParam("table", "users"))

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	want := `{"code":"INTERNAL_SERVER_ERROR"}`
	if string(data) != want {
		t.Fatalf("Marshal() = %s, want %s", data, want)
	}

	// Params stays available server-side regardless of Exposed.
	if e.Params()["table"] != "users" {
		t.Fatalf("Params() = %v", e.Params())
	}
}

func TestParams_ReturnsCopy(t *testing.T) {
	e := xerr.New(xerr.CodeInvalidParam, xerr.WithParam("resource", "seats"))

	p := e.Params()
	p["resource"] = mutatedValue

	if e.Params()["resource"] != "seats" {
		t.Fatal("Params() should return a defensive copy")
	}
}

func TestError_MarshalJSON_NotExposed(t *testing.T) {
	e := xerr.New(xerr.CodeDatabaseError,
		xerr.WithMessage("dial tcp 10.0.0.5:5432: connection refused"),
		xerr.WithErr(errors.New("pq: connection refused")),
	)

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	want := `{"code":"INTERNAL_SERVER_ERROR"}`
	if string(data) != want {
		t.Fatalf("Marshal() = %s, want %s", data, want)
	}
}

func TestError_Error_IncludesFullDetailRegardlessOfExposure(t *testing.T) {
	cause := errors.New("dial tcp: connection refused")
	e := xerr.New(xerr.CodeDatabaseError,
		xerr.WithMessage("could not reach primary"),
		xerr.WithErr(cause),
		xerr.WithDiagnostic(xerr.DiagnosticOperation, "CreateUser"),
	)

	s := e.Error()

	for _, want := range []string{
		string(xerr.CodeDatabaseError),
		string(xerr.KindInfrastructure),
		"could not reach primary",
		"operation=CreateUser",
		cause.Error(),
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("Error() = %q, missing %q", s, want)
		}
	}
}

func TestError_Error_Deterministic(t *testing.T) {
	e := xerr.New(xerr.CodeInternalError,
		xerr.WithDiagnostic(xerr.DiagnosticResource, "user:42"),
		xerr.WithDiagnostic(xerr.DiagnosticOperation, "CreateUser"),
		xerr.WithDiagnostic(xerr.DiagnosticReason, "timeout"),
	)

	first := e.Error()
	for i := 0; i < 20; i++ {
		if got := e.Error(); got != first {
			t.Fatalf("Error() is non-deterministic: %q vs %q", first, got)
		}
	}
}

func TestError_Unwrap_And_ErrorsAs(t *testing.T) {
	cause := errors.New("boom")
	e := xerr.New(xerr.CodeInternalError, xerr.WithErr(cause))

	if !errors.Is(e, cause) {
		t.Fatal("errors.Is(e, cause) should be true")
	}

	var target *xerr.Error
	if !errors.As(e, &target) {
		t.Fatal("errors.As should find the *Error")
	}
	if target != e {
		t.Fatal("errors.As should return the same *Error")
	}
}

func TestError_ChainedWrap_PreservesFullDetail(t *testing.T) {
	// Repository layer: a raw DB error becomes an infrastructure xerr.
	infra := xerr.New(xerr.CodeRecordNotFound, xerr.WithErr(errors.New("sql: no rows in result set")))

	// Service layer: translated into a safe, client-facing domain error,
	// keeping the infra error attached for logs.
	domain := xerr.New(xerr.CodeNotFound,
		xerr.WithMessage("user not found"),
		xerr.WithErr(infra),
	)

	if !domain.Exposed() {
		t.Fatal("domain error should be exposed")
	}

	data, err := json.Marshal(domain)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if strings.Contains(string(data), "sql: no rows") {
		t.Fatalf("client JSON leaked infra detail: %s", data)
	}

	var inner *xerr.Error
	if !errors.As(domain.Err(), &inner) {
		t.Fatal("expected the wrapped infra *Error to be reachable")
	}
	if inner.Code() != xerr.CodeRecordNotFound {
		t.Fatalf("inner.Code() = %q, want %q", inner.Code(), xerr.CodeRecordNotFound)
	}
	if inner.Exposed() {
		t.Fatal("inner infra error should not be exposed")
	}
}

func TestError_Is_MatchesByCodeOnly(t *testing.T) {
	sentinel := xerr.New(xerr.CodeNotFound)

	concrete := xerr.New(xerr.CodeNotFound,
		xerr.WithMessage("user not found"),
		xerr.WithViolation("id", xerr.ErrorReasonNotFound),
	)

	if !errors.Is(concrete, sentinel) {
		t.Fatal("errors.Is should match by Code, ignoring message/violations")
	}

	other := xerr.New(xerr.CodeConflict)
	if errors.Is(concrete, other) {
		t.Fatal("errors.Is should not match a different Code")
	}
}

func TestWrap_NilReturnsNil(t *testing.T) {
	if xerr.Wrap(nil, xerr.CodeInternalError) != nil {
		t.Fatal("Wrap(nil, ...) should return nil")
	}
}

func TestFromError(t *testing.T) {
	e := xerr.New(xerr.CodeConflict)
	wrapped := fmt.Errorf("handler: %w", e)

	got, ok := xerr.FromError(wrapped)
	if !ok || got != e {
		t.Fatalf("FromError() = (%v, %v), want (%v, true)", got, ok, e)
	}

	_, ok = xerr.FromError(errors.New("plain"))
	if ok {
		t.Fatal("FromError() should be false for a non-xerr error")
	}
}

func TestWithViolations_Batch(t *testing.T) {
	batch := []xerr.Violation{
		{Field: fieldEmail, Reason: xerr.ErrorReasonRequired},
		{Field: fieldPassword, Reason: xerr.ErrorReasonTooShort, Params: map[string]any{"min": 8}},
	}

	e := xerr.New(xerr.CodeValidationFailed,
		xerr.WithViolations(batch...),
		xerr.WithViolation("username", xerr.ErrorReasonAlreadyExists),
	)

	got := e.Violations()
	if len(got) != 3 {
		t.Fatalf("Violations() has %d entries, want 3", len(got))
	}
	if got[0].Field != fieldEmail || got[1].Field != fieldPassword || got[2].Field != "username" {
		t.Fatalf("Violations() = %+v, order/content mismatch", got)
	}
}

func TestViolations_ReturnsCopy(t *testing.T) {
	e := xerr.New(xerr.CodeValidationFailed, xerr.WithViolation(fieldEmail, xerr.ErrorReasonRequired))

	v := e.Violations()
	v[0].Field = mutatedValue

	if e.Violations()[0].Field != fieldEmail {
		t.Fatal("Violations() should return a defensive copy")
	}
}

func TestDiagnostics_ReturnsCopy(t *testing.T) {
	e := xerr.New(xerr.CodeInternalError, xerr.WithDiagnostic(xerr.DiagnosticOperation, "op"))

	d := e.Diagnostics()
	d[xerr.DiagnosticOperation] = mutatedValue

	if e.Diagnostics()[xerr.DiagnosticOperation] != "op" {
		t.Fatal("Diagnostics() should return a defensive copy")
	}
}
