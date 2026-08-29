package sessioninventory

import (
	"encoding/json"
	"path"
	"strings"
)

func ScanClaude(runtime Runtime) ScanResult {
	var result ScanResult
	for _, root := range runtime.NativeRoots(AgentClaude) {
		files, diagnostics, ok := scannerFiles(runtime, AgentClaude, root)
		result.Diagnostics = append(result.Diagnostics, diagnostics...)
		if !ok {
			continue
		}
		for _, entry := range files {
			fact, diagnostics, ok := scanClaudeFile(runtime, entry)
			result.Diagnostics = append(result.Diagnostics, diagnostics...)
			if ok {
				result.Facts = append(result.Facts, fact)
			}
		}
	}
	return result
}

func scanClaudeFile(runtime Runtime, entry FileEntry) (Fact, []Diagnostic, bool) {
	artifact := entry.Artifact
	artifact.Kind = ArtifactTranscript
	nativeID, parentID, role, recognized := claudePathFact(artifact.RelativePath)
	if !recognized {
		if strings.HasSuffix(artifact.RelativePath, ".jsonl") {
			return Fact{}, []Diagnostic{artifactDiagnostic(DiagnosticSchemaNearMiss, AgentClaude, nil, artifact, "unrecognized Claude v1 path")}, false
		}
		return Fact{}, nil, false
	}

	expectedSessionID := nativeID
	if parentID != nil {
		expectedSessionID = *parentID
	}
	chronology := fallbackTime(entry)
	contradiction := false
	var diagnostics []Diagnostic
	err := visitJSONLines(runtime, artifact, jsonRecordLimit, false, func(line []byte) bool {
		if len(line) == 0 {
			return false
		}
		var record struct {
			Timestamp   string `json:"timestamp"`
			SessionID   string `json:"sessionId"`
			IsSidechain *bool  `json:"isSidechain"`
			Type        string `json:"type"`
			Message     struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		}
		if err := decodeStrictJSON(line, &record); err != nil {
			diagnostics = append(diagnostics, artifactDiagnostic(DiagnosticSchemaNearMiss, AgentClaude, &nativeID, artifact, "malformed Claude JSONL record"))
			return false
		}
		if record.SessionID != "" && record.SessionID != expectedSessionID {
			contradiction = true
			return true
		}
		if record.IsSidechain != nil && ((*record.IsSidechain && role == RoleRoot) || (!*record.IsSidechain && role == RoleSubagent)) {
			contradiction = true
			return true
		}
		if chronology == nil || chronology.Source != TimeSourceMetadata {
			if parsed := metadataTime(record.Timestamp); parsed != nil {
				chronology = parsed
			}
		}
		return false
	})
	if err != nil {
		diagnostics = append(diagnostics, artifactDiagnostic(DiagnosticSchemaNearMiss, AgentClaude, &nativeID, artifact, err.Error()))
	}
	if contradiction {
		diagnostics = append(diagnostics, artifactDiagnostic(DiagnosticNodeMalformed, AgentClaude, &nativeID, artifact, "Claude metadata contradicts path identity"))
	}
	return Fact{
		Agent:          AgentClaude,
		NativeID:       nativeID,
		Role:           role,
		ParentID:       parentID,
		Time:           chronology,
		Resumable:      role == RoleRoot && !contradiction,
		Disputed:       contradiction,
		Artifacts:      []Artifact{artifact},
		EdgeProvenance: edgeProvenance(role, "claude-v1", artifact),
	}, diagnostics, true
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
