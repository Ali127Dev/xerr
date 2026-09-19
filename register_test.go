package xerr_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Ali127Dev/xerr/v3"
)

func TestRegisterCode_RegistersKindAndHTTPStatus(t *testing.T) {
	code := xerr.Code("TEST_REGISTER_CODE_SUCCESS")

	xerr.RegisterCode(code, xerr.KindDomain, http.StatusTeapot)

	if got := code.Kind(); got != xerr.KindDomain {
		t.Fatalf("Kind() = %q, want %q", got, xerr.KindDomain)
	}
	if got := code.HTTPStatus(); got != http.StatusTeapot {
		t.Fatalf("HTTPStatus() = %d, want %d", got, http.StatusTeapot)
	}

	e := xerr.New(code, xerr.WithMessage("teapot"))
	if !e.Exposed() {
		t.Fatal("a KindDomain code registered via RegisterCode should be exposed by default")
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got["code"] != string(code) {
		t.Fatalf("code = %v, want %v", got["code"], code)
	}
}

func TestRegisterCode_EmptyCodePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("RegisterCode should panic on an empty code")
		}
	}()
	xerr.RegisterCode(xerr.Code(""), xerr.KindDomain, http.StatusForbidden)
}

func TestRegisterCode_LowercaseCodePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("RegisterCode should panic on a lower-case code")
		}
	}()
	xerr.RegisterCode(xerr.Code("plan_not_included"), xerr.KindDomain, http.StatusForbidden)
}

func TestRegisterCode_InvalidKindPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("RegisterCode should panic on a Kind outside the four defined constants")
		}
	}()
	xerr.RegisterCode(xerr.Code("TEST_REGISTER_CODE_INVALID_KIND"), xerr.Kind("bogus"), http.StatusForbidden)
}

func TestRegisterCode_HTTPStatusOutOfRangePanics(t *testing.T) {
	tests := []struct {
		name       string
		codeSuffix string
		status     int
	}{
		{"below 400", "BELOW_400", http.StatusOK},
		{"above 599", "ABOVE_599", 600},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("RegisterCode should panic on httpStatus=%d", tt.status)
				}
			}()
			xerr.RegisterCode(xerr.Code("TEST_REGISTER_CODE_STATUS_"+tt.codeSuffix), xerr.KindDomain, tt.status)
		})
	}
}

func TestRegisterCode_DuplicateRegistrationPanics(t *testing.T) {
	code := xerr.Code("TEST_REGISTER_CODE_DUPLICATE")
	xerr.RegisterCode(code, xerr.KindDomain, http.StatusForbidden)

	defer func() {
		if recover() == nil {
			t.Fatal("RegisterCode should panic when the code is already registered")
		}
	}()
	xerr.RegisterCode(code, xerr.KindApplication, http.StatusConflict)
}

func TestRegisterCode_CollisionWithBuiltinCodePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("RegisterCode should panic when colliding with a built-in code")
		}
	}()
	xerr.RegisterCode(xerr.CodeNotFound, xerr.KindDomain, http.StatusNotFound)
}

func TestExposedCodes_OnlyContainsCodesSafeByDefault(t *testing.T) {
	code := xerr.Code("TEST_EXPOSED_CODES_DOMAIN")
	xerr.RegisterCode(code, xerr.KindDomain, http.StatusForbidden)

	codes := xerr.ExposedCodes()

	if _, ok := codes[xerr.CodeNotFound]; !ok {
		t.Fatal("ExposedCodes() should contain CodeNotFound (KindDomain)")
	}
	if _, ok := codes[code]; !ok {
		t.Fatal("ExposedCodes() should contain a code just registered with KindDomain")
	}
	if _, ok := codes[xerr.CodeDatabaseError]; ok {
		t.Fatal("ExposedCodes() should not contain CodeDatabaseError (KindInfrastructure)")
	}

	for c, k := range codes {
		if !k.Safe() {
			t.Fatalf("ExposedCodes() returned %q with unsafe kind %q", c, k)
		}
	}
}

func TestAllCodesHaveNonZeroHTTPStatus(t *testing.T) {
	for code := range xerr.RegisteredCodes() {
		if status := code.HTTPStatus(); status == 0 {
			t.Errorf("code %q has HTTPStatus() == 0", code)
		}
	}
}

func TestRegisteredCodes_ReturnsCopy(t *testing.T) {
	codes := xerr.RegisteredCodes()

	before := len(codes)
	codes[xerr.Code("TEST_REGISTERED_CODES_MUTATION")] = xerr.KindDomain

	if got := len(xerr.RegisteredCodes()); got != before {
		t.Fatalf("RegisteredCodes() = %d entries after mutating a prior result, want %d (should be a defensive copy)", got, before)
	}
}

func TestExposedCodes_ReturnsCopy(t *testing.T) {
	codes := xerr.ExposedCodes()

	before := len(codes)
	codes[xerr.Code("TEST_EXPOSED_CODES_MUTATION")] = xerr.KindDomain

	if got := len(xerr.ExposedCodes()); got != before {
		t.Fatalf("ExposedCodes() = %d entries after mutating a prior result, want %d (should be a defensive copy)", got, before)
	}
}
