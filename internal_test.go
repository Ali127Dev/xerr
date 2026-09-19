package xerr

import "testing"

// TestAllCodesHaveHTTPStatusAndKind is a white-box invariant check: every
// code registered in codeKinds must also have an explicit entry in
// codeStatuses, and vice versa, so the two hand-maintained built-in
// literals (kind.go, http_status.go) never drift apart. RegisterCode
// itself can't drift — it always writes both maps together under one
// lock — so this specifically guards the built-in tables, which is why
// it needs direct (unexported) access instead of going through the
// public RegisteredCodes().
func TestAllCodesHaveHTTPStatusAndKind(t *testing.T) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	for code := range codeKinds {
		if _, ok := codeStatuses[code]; !ok {
			t.Errorf("code %q has a Kind but no HTTP status mapping", code)
		}
	}
	for code := range codeStatuses {
		if _, ok := codeKinds[code]; !ok {
			t.Errorf("code %q has an HTTP status but no Kind mapping", code)
		}
	}
}
