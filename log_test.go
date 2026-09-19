package xerr_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/Ali127Dev/xerr/v3"
)

func TestError_LogValue_IncludesUnexposedFields(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	cause := errors.New("dial tcp: connection refused")
	e := xerr.New(xerr.CodeDatabaseError,
		xerr.WithMessage("could not reach primary"),
		xerr.WithErr(cause),
		xerr.WithParam("host", "db-primary"),
		xerr.WithDiagnostic(xerr.DiagnosticOperation, "CreateUser"),
	)

	logger.Error("request failed", "err", e)

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	errAttr, ok := got["err"].(map[string]any)
	if !ok {
		t.Fatalf("err attr = %v", got["err"])
	}

	// Fields that MarshalJSON hides from a client (kind, diagnostics,
	// cause) must still show up in the log, since this is a different
	// boundary entirely.
	if errAttr["code"] != string(xerr.CodeDatabaseError) {
		t.Fatalf("code = %v", errAttr["code"])
	}
	if errAttr["kind"] != string(xerr.KindInfrastructure) {
		t.Fatalf("kind = %v", errAttr["kind"])
	}
	if errAttr["message"] != "could not reach primary" {
		t.Fatalf("message = %v", errAttr["message"])
	}
	if errAttr["cause"] != cause.Error() {
		t.Fatalf("cause = %v", errAttr["cause"])
	}
	diagnostics, ok := errAttr["diagnostics"].(map[string]any)
	if !ok || diagnostics["operation"] != "CreateUser" {
		t.Fatalf("diagnostics = %v", errAttr["diagnostics"])
	}
	params, ok := errAttr["params"].(map[string]any)
	if !ok || params["host"] != "db-primary" {
		t.Fatalf("params = %v", errAttr["params"])
	}
}

func TestError_LogValue_OmitsEmptyFields(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	logger.Error("failed", "err", xerr.New(xerr.CodeNotFound))

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	errAttr := got["err"].(map[string]any)

	for _, field := range []string{"message", "params", "violations", "diagnostics", "cause", "stack"} {
		if _, present := errAttr[field]; present {
			t.Errorf("unexpected field %q in log output: %v", field, errAttr[field])
		}
	}
}
