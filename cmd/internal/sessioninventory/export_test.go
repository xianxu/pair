package sessioninventory

// Unexported seams handed to the external test package, which is the only
// one that can use the shared FakeRuntime without an import cycle.
var (
	ReadJSONLArtifact  = readJSONLArtifact
	VisitJSONLinesAt   = visitJSONLinesAt
	ErrTruncatedRecord = errTruncatedRecord
)
