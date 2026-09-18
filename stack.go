package xerr

import "runtime"

const maxStackFrames = 32

// WithStack captures the current call stack, for later inspection via
// Stack(). Opt-in and skipped by default: runtime.Callers is cheap but
// not free, and errors constructed in a hot path (e.g. per-field
// validation, run once per request) shouldn't pay for it unasked.
// Reach for it on the errors you'll actually want a trace for —
// typically KindInfrastructure / KindUnknown ones. Recover always
// captures a stack, since that is the entire point of catching a panic.
func WithStack() ErrorOption {
	return func(e *Error) {
		e.stack = captureStack(4)
	}
}

// captureStack records the call stack starting skip frames up from its
// own caller. Shared by WithStack (called via New/Wrap's option loop)
// and Recover (called directly), which is why the skip count is passed
// in rather than hardcoded.
func captureStack(skip int) []uintptr {
	pcs := make([]uintptr, maxStackFrames)
	n := runtime.Callers(skip, pcs)
	return pcs[:n]
}
