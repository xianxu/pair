package sessioninventory

import (
	"slices"
	"testing"
)

func TestDiagnosticRegistryIsExhaustiveAndCanonical(t *testing.T) {
	t.Parallel()

	want := map[DiagnosticCode]Severity{
		DiagnosticStorageAbsent:               SeverityInfo,
		DiagnosticConformanceNoSample:         SeverityInfo,
		DiagnosticSchemaNearMiss:              SeverityWarning,
		DiagnosticParentMissing:               SeverityWarning,
		DiagnosticBindingStale:                SeverityWarning,
		DiagnosticProcessChanged:              SeverityWarning,
		DiagnosticTurnUnusable:                SeverityWarning,
		DiagnosticSendIncomplete:              SeverityWarning,
		DiagnosticSendAborted:                 SeverityWarning,
		DiagnosticStorageUnreadable:           SeverityError,
		DiagnosticNodeMalformed:               SeverityError,
		DiagnosticParentConflict:              SeverityError,
		DiagnosticDuplicateConflict:           SeverityError,
		DiagnosticBindingConflict:             SeverityError,
		DiagnosticArtifactPathInvalid:         SeverityError,
		DiagnosticPairRecordMalformed:         SeverityError,
		DiagnosticScopeRejected:               SeverityError,
		DiagnosticConformancePrivacyViolation: SeverityError,
	}
	if !slices.Equal(DiagnosticCodes(), sortedDiagnosticCodes(want)) {
		t.Fatalf("registry codes = %#v, want %#v", DiagnosticCodes(), sortedDiagnosticCodes(want))
	}
	for code, severity := range want {
		got := diagnostic(code, AgentClaude, nil, "first detail")
		if got.Severity != severity {
			t.Errorf("%s severity = %s, want %s", code, got.Severity, severity)
		}
		other := diagnostic(code, AgentClaude, nil, "different detail")
		if got.StableID != other.StableID {
			t.Errorf("%s ID changed with free-form detail: %s != %s", code, got.StableID, other.StableID)
		}
	}
}

func TestSortInventoryCoalescesDiagnosticsAndUsesSeverityOrder(t *testing.T) {
	t.Parallel()

	errorDiagnostic := diagnostic(DiagnosticNodeMalformed, AgentClaude, nil, "first")
	duplicate := diagnostic(DiagnosticNodeMalformed, AgentClaude, nil, "second")
	warningDiagnostic := diagnostic(DiagnosticSchemaNearMiss, AgentClaude, nil, "warning")
	infoDiagnostic := diagnostic(DiagnosticStorageAbsent, AgentClaude, nil, "info")
	got := SortInventory(Inventory{Diagnostics: []Diagnostic{infoDiagnostic, duplicate, warningDiagnostic, errorDiagnostic}})
	if len(got.Diagnostics) != 3 {
		t.Fatalf("diagnostics = %#v, want identical diagnostics coalesced", got.Diagnostics)
	}
	if got.Diagnostics[0].Severity != SeverityError || got.Diagnostics[1].Severity != SeverityWarning || got.Diagnostics[2].Severity != SeverityInfo {
		t.Fatalf("severity order = %#v, want error/warning/info", got.Diagnostics)
	}
}

func sortedDiagnosticCodes(values map[DiagnosticCode]Severity) []DiagnosticCode {
	result := make([]DiagnosticCode, 0, len(values))
	for code := range values {
		result = append(result, code)
	}
	slices.Sort(result)
	return result
}
