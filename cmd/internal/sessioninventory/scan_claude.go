package sessioninventory

import (
	"encoding/json"
	"errors"
	"path"
	"strings"
)

type claudeRecord struct {
	Timestamp   string `json:"timestamp"`
	SessionID   string `json:"sessionId"`
	IsSidechain *bool  `json:"isSidechain"`
	Type        string `json:"type"`
	Message     struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

func ScanClaude(runtime Runtime) ScanResult {
	return scanClaudeFamily(runtime, AgentClaude, "claude-v1")
}

// scanClaudeFamily runs the claude-family record transition for one agent and
// scanner schema. Claude and qoder transcripts share the record shape
// (type/sessionId/isSidechain/timestamp/message.role) and the path layout
// (<project>/<uuid>.jsonl, <uuid>/subagents/agent-*.jsonl).
func scanClaudeFamily(runtime Runtime, agent Agent, schema string) ScanResult {
	var result ScanResult
	for _, root := range runtime.NativeRoots(agent) {
		files, diagnostics, ok := scannerFiles(runtime, agent, root)
		result.Diagnostics = append(result.Diagnostics, diagnostics...)
		if !ok {
			continue
		}
		for _, entry := range files {
			fact, diagnostics, ok := scanClaudeFamilyFile(runtime, entry, agent, schema)
			result.Diagnostics = append(result.Diagnostics, diagnostics...)
			if ok {
				result.Facts = append(result.Facts, fact)
			}
		}
	}
	return result
}

func scanClaudeFamilyFile(runtime Runtime, entry FileEntry, agent Agent, schema string) (Fact, []Diagnostic, bool) {
	artifact := entry.Artifact
	artifact.Kind = ArtifactTranscript
	entry.Artifact = artifact
	nativeID, _, _, recognized := claudePathFact(artifact.RelativePath)
	if !recognized {
		if strings.HasSuffix(artifact.RelativePath, ".jsonl") {
			return Fact{}, []Diagnostic{artifactDiagnostic(DiagnosticSchemaNearMiss, agent, nil, artifact, "unrecognized "+schema+" path")}, false
		}
		return Fact{}, nil, false
	}

	state, diagnostics, err := validateClaudeFamilyDelta(entry, nil, nil, agent, schema)
	if err != nil {
		return Fact{}, []Diagnostic{artifactDiagnostic(DiagnosticSchemaNearMiss, agent, &nativeID, artifact, err.Error())}, false
	}
	err = visitJSONLines(runtime, artifact, unlimitedRecordSize, func(line []byte) bool {
		applyClaudeFamilyRecord(&state, entry, line, &diagnostics, agent, schema)
		return false
	})
	if err != nil {
		diagnostics = append(diagnostics, artifactDiagnostic(DiagnosticSchemaNearMiss, agent, &nativeID, artifact, err.Error()))
		state.Disputed = true
	}
	return scannerStateFact(state, []Artifact{artifact}), diagnostics, true
}

// ValidateClaudeDelta applies complete records to a cloned scanner state. It
// uses the same record transition as the full scanner.
func ValidateClaudeDelta(entry FileEntry, prior *ScannerState, records []FramedJSONLRecord) (ScannerState, []Diagnostic, error) {
	return validateClaudeFamilyDelta(entry, prior, records, AgentClaude, "claude-v1")
}

func validateClaudeFamilyDelta(entry FileEntry, prior *ScannerState, records []FramedJSONLRecord, agent Agent, schema string) (ScannerState, []Diagnostic, error) {
	artifact := entry.Artifact
	artifact.Kind = ArtifactTranscript
	entry.Artifact = artifact
	nativeID, parentID, role, recognized := claudePathFact(artifact.RelativePath)
	if !recognized {
		return ScannerState{}, nil, errors.New("unrecognized " + schema + " path")
	}
	anchor := nativeID
	if parentID != nil {
		anchor = *parentID
	}
	state := ScannerState{Version: ScannerStateVersion, Agent: agent, NativeID: nativeID, IdentityAnchor: anchor, Role: role, ParentID: cloneString(parentID), ScannerSchema: schema, Chronology: fallbackTime(entry)}
	if prior != nil {
		if err := ValidateScannerState(*prior); err != nil {
			return ScannerState{}, nil, err
		}
		state = cloneScannerState(*prior)
		if state.Agent != agent || state.NativeID != nativeID || state.IdentityAnchor != anchor || state.Role != role || !equalString(state.ParentID, parentID) || state.ScannerSchema != schema {
			return ScannerState{}, nil, errors.New(schema + " scanner state does not match artifact")
		}
	}
	var diagnostics []Diagnostic
	for _, record := range records {
		applyClaudeFamilyRecord(&state, entry, record.Bytes, &diagnostics, agent, schema)
	}
	if err := ValidateScannerState(state); err != nil {
		return ScannerState{}, diagnostics, err
	}
	return state, diagnostics, nil
}

func applyClaudeFamilyRecord(state *ScannerState, entry FileEntry, line []byte, diagnostics *[]Diagnostic, agent Agent, schema string) {
	if len(line) == 0 {
		return
	}
	artifact := entry.Artifact
	artifact.Kind = ArtifactTranscript
	var record claudeRecord
	if err := decodeStrictJSON(line, &record); err != nil {
		state.Disputed = true
		*diagnostics = append(*diagnostics, artifactDiagnostic(DiagnosticSchemaNearMiss, agent, &state.NativeID, artifact, "malformed "+schema+" JSONL record"))
		return
	}
	contradiction := record.SessionID != "" && record.SessionID != state.IdentityAnchor
	contradiction = contradiction || record.IsSidechain != nil && ((*record.IsSidechain && state.Role == RoleRoot) || (!*record.IsSidechain && state.Role == RoleSubagent))
	if contradiction {
		state.Disputed = true
		*diagnostics = append(*diagnostics, artifactDiagnostic(DiagnosticNodeMalformed, agent, &state.NativeID, artifact, schema+" metadata contradicts path identity"))
		return
	}
	state.FirstRecordValidated = true
	if state.Chronology == nil || state.Chronology.Source != TimeSourceMetadata {
		if parsed := metadataTime(record.Timestamp); parsed != nil {
			state.Chronology = parsed
		}
	}
}

func equalString(left, right *string) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func claudePathFact(relativePath string) (string, *string, Role, bool) {
	parts := strings.Split(path.Clean(relativePath), "/")
	if len(parts) == 2 && strings.HasSuffix(parts[1], ".jsonl") {
		nativeID := strings.TrimSuffix(parts[1], ".jsonl")
		if parts[0] != "" && uuidPattern.MatchString(nativeID) {
			return nativeID, nil, RoleRoot, true
		}
		return "", nil, RoleUnknown, false
	}
	if len(parts) == 4 && parts[2] == "subagents" && len(parts[3]) > 6 && parts[3][:6] == "agent-" && strings.HasSuffix(parts[3], ".jsonl") {
		parentID := parts[1]
		nativeID := strings.TrimSuffix(parts[3][6:], ".jsonl")
		if parts[0] != "" && uuidPattern.MatchString(parentID) && asciiIDPattern.MatchString(nativeID) {
			return nativeID, &parentID, RoleSubagent, true
		}
	}
	return "", nil, RoleUnknown, false
}
