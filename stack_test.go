package xerr_test

import (
	"strings"
	"testing"

	"github.com/Ali127Dev/xerr/v2"
)

func TestError_Stack_EmptyByDefault(t *testing.T) {
	e := xerr.New(xerr.CodeDatabaseError)
	if e.Stack() != "" {
		t.Fatalf("Stack() = %q, want empty (WithStack not used)", e.Stack())
	}
}

func causingFuncForTest() *xerr.Error {
	return xerr.New(xerr.CodeDatabaseError, xerr.WithStack())
}

func TestError_Stack_CapturesCallSite(t *testing.T) {
	e := causingFuncForTest()

	s := e.Stack()
	if s == "" {
		t.Fatal("Stack() should be non-empty when WithStack is used")
	}
	if !strings.Contains(s, "causingFuncForTest") {
		t.Fatalf("Stack() = %q, want it to include the actual call site", s)
	}
	if strings.Contains(s, "captureStack") || strings.Contains(s, "WithStack") {
		t.Fatalf("Stack() = %q, leaked internal library frames", s)
	}
}

func panickingFuncForTest() (err *xerr.Error) {
	defer func() {
		if v := recover(); v != nil {
			err = xerr.Recover(v)
		}
	}()
	panic("boom")
}

func TestRecover_CapturesPanicAsError(t *testing.T) {
	e := panickingFuncForTest()

	if e == nil {
		t.Fatal("Recover should have produced an *Error")
	}
	if e.Code() != xerr.CodePanic {
		t.Fatalf("Code() = %q, want %q", e.Code(), xerr.CodePanic)
	}
	if e.Exposed() {
		t.Fatal("a recovered panic should not be exposed to clients")
	}
	if !strings.Contains(e.Message(), "boom") {
		t.Fatalf("Message() = %q, want it to mention the panic value", e.Message())
	}

	s := e.Stack()
	if !strings.Contains(s, "panickingFuncForTest") {
		t.Fatalf("Stack() = %q, want it to include the panicking call site", s)
	}
}

func TestRecover_WithErrorValue(t *testing.T) {
	cause := errPanic{"db closed"}

	var got *xerr.Error
	func() {
		defer func() {
			if v := recover(); v != nil {
				got = xerr.Recover(v)
			}
		}()
		panic(cause)
	}()

	if got.Err() != cause {
		t.Fatalf("Err() = %v, want the original error value %v", got.Err(), cause)
	}
}

func TestRecover_NilReturnsNil(t *testing.T) {
	if xerr.Recover(nil) != nil {
		t.Fatal("Recover(nil) should return nil")
	}
}

type errPanic struct{ msg string }

func (e errPanic) Error() string { return e.msg }
