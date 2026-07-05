package xerr

type DiagnosticKey string

const (
	DiagnosticOperation DiagnosticKey = "operation"
	DiagnosticReason    DiagnosticKey = "reason"
	DiagnosticResource  DiagnosticKey = "resource"
)

func (d DiagnosticKey) String() string { return string(d) }
