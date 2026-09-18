package xerr

import (
	"errors"
	"fmt"
)

// Recover converts a recovered panic value into an *Error. Call it
// directly from a deferred recover:
//
//	defer func() {
//	    if v := recover(); v != nil {
//	        err = xerr.Recover(v)
//	    }
//	}()
//
// The result uses CodePanic (KindUnknown, unsafe to expose — the client
// only ever sees a generic internal error) and always carries a captured
// stack (see Stack), since a stack trace is the entire reason to catch
// a panic in the first place. Returns nil if v is nil (recover() is
// commonly called unconditionally; a nil v means there was no panic).
func Recover(v any, opts ...ErrorOption) *Error {
	if v == nil {
		return nil
	}

	e := &Error{
		code:  CodePanic,
		kind:  CodePanic.Kind(),
		stack: captureStack(3),
	}

	if cause, ok := v.(error); ok {
		e.err = cause
		e.message = fmt.Sprintf("panic: %s", cause.Error())
	} else {
		e.message = fmt.Sprintf("panic: %v", v)
		e.err = errors.New(e.message)
	}

	for _, opt := range opts {
		opt(e)
	}
	return e
}
