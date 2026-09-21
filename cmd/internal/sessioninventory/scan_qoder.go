package sessioninventory

// ScanQoder scans qoder's claude-family transcripts under ~/.qoder/projects.
// The record transition and path shape are shared with claude (scan_claude.go);
// only the agent constant, schema id, and storage root differ.
func ScanQoder(runtime Runtime) ScanResult {
	return scanClaudeFamily(runtime, AgentQoder, "qoder-v1")
}

// ValidateQoderDelta applies complete records to a cloned scanner state.
func ValidateQoderDelta(entry FileEntry, prior *ScannerState, records []FramedJSONLRecord) (ScannerState, []Diagnostic, error) {
	return validateClaudeFamilyDelta(entry, prior, records, AgentQoder, "qoder-v1")
}
