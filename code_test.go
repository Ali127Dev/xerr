package xerr_test

import (
	"net/http"
	"testing"

	"github.com/Ali127Dev/xerr/v3"
)

func TestCode_HTTPStatus_KnownCode(t *testing.T) {
	if got := xerr.CodeNotFound.HTTPStatus(); got != http.StatusNotFound {
		t.Fatalf("HTTPStatus() = %d, want %d", got, http.StatusNotFound)
	}
}

func TestCode_HTTPStatus_UnknownCodeFallsBackToInternal(t *testing.T) {
	unknown := xerr.Code("SOME_UNREGISTERED_CODE")
	if got := unknown.HTTPStatus(); got != http.StatusInternalServerError {
		t.Fatalf("HTTPStatus() = %d, want %d", got, http.StatusInternalServerError)
	}
}

func TestCode_Kind_UnknownCodeFallsBackToUnknown(t *testing.T) {
	unknown := xerr.Code("SOME_UNREGISTERED_CODE")
	if got := unknown.Kind(); got != xerr.KindUnknown {
		t.Fatalf("Kind() = %q, want %q", got, xerr.KindUnknown)
	}
}
