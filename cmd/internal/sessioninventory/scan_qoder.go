package sessioninventory

// ScanQoder scans qoder's claude-family transcripts under ~/.qoder/projects.
// The record transition and path shape are shared with claude (scan_claude.go);
// only the agent's claudeFamilySpec differs (qoder admits epoch-millisecond
// bookkeeping timestamps).
func ScanQoder(runtime Runtime) ScanResult {
	return scanClaudeFamily(runtime, AgentQoder)
}

// ValidateQoderDelta applies complete records to a cloned scanner state.
func ValidateQoderDelta(entry FileEntry, prior *ScannerState, records []FramedJSONLRecord) (ScannerState, []Diagnostic, error) {
	spec, _ := claudeFamilySpecFor(AgentQoder)
	return validateClaudeFamilyDelta(entry, prior, records, spec)
}
