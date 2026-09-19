package xerr_test

import (
	"testing"

	"github.com/Ali127Dev/xerr/v3"
)

func TestViolation_DefaultMessage(t *testing.T) {
	tests := []struct {
		name string
		v    xerr.Violation
		want string
	}{
		{
			"required",
			xerr.Violation{Field: fieldEmail, Reason: xerr.ErrorReasonRequired},
			"email is required",
		},
		{
			"too_short with param",
			xerr.Violation{Field: fieldPassword, Reason: xerr.ErrorReasonTooShort, Params: map[string]any{"min": 8}},
			"password must be at least 8 characters",
		},
		{
			"too_short without param",
			xerr.Violation{Field: fieldPassword, Reason: xerr.ErrorReasonTooShort},
			"password is too short",
		},
		{
			"invalid_value with allowed param",
			xerr.Violation{Field: "status", Reason: xerr.ErrorReasonInvalidValue, Params: map[string]any{"allowed": []string{"draft", "published"}}},
			"status must be one of [draft published]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.DefaultMessage(); got != tt.want {
				t.Fatalf("DefaultMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestError_DefaultMessage_PrefersExplicitMessage(t *testing.T) {
	e := xerr.New(xerr.CodeConflict,
		xerr.WithMessage("coupon already redeemed"),
		xerr.WithViolation("code", xerr.ErrorReasonAlreadyExists),
	)

	if got := e.DefaultMessage(); got != "coupon already redeemed" {
		t.Fatalf("DefaultMessage() = %q, want the explicit message", got)
	}
}

func TestError_DefaultMessage_FallsBackToViolations(t *testing.T) {
	e := xerr.New(xerr.CodeValidationFailed,
		xerr.WithViolation(fieldEmail, xerr.ErrorReasonRequired),
		xerr.WithViolation(fieldPassword, xerr.ErrorReasonTooShort, xerr.P("min", 8)),
	)

	want := "email is required; password must be at least 8 characters"
	if got := e.DefaultMessage(); got != want {
		t.Fatalf("DefaultMessage() = %q, want %q", got, want)
	}
}

func TestError_DefaultMessage_GenericFallback(t *testing.T) {
	safe := xerr.New(xerr.CodeNotFound)
	if got := safe.DefaultMessage(); got != string(xerr.CodeNotFound) {
		t.Fatalf("DefaultMessage() = %q, want the code string", got)
	}

	unsafe := xerr.New(xerr.CodeDatabaseError)
	if got := unsafe.DefaultMessage(); got != "something went wrong" {
		t.Fatalf("DefaultMessage() = %q, want the generic unsafe fallback", got)
	}
}
