package sessioninventory

import "sort"

var diagnosticRegistry = map[DiagnosticCode]Severity{
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

func DiagnosticCodes() []DiagnosticCode {
	codes := make([]DiagnosticCode, 0, len(diagnosticRegistry))
	for code := range diagnosticRegistry {
		codes = append(codes, code)
	}
	sort.Slice(codes, func(i, j int) bool { return codes[i] < codes[j] })
	return codes
}

func diagnosticSeverity(code DiagnosticCode) (Severity, bool) {
	severity, ok := diagnosticRegistry[code]
	return severity, ok
}
