package sessioninventory

import (
	"path"
	"strings"
)

func ScanMuse(runtime Runtime) ScanResult {
	var result ScanResult
	for _, root := range runtime.NativeRoots(AgentMuse) {
		files, diagnostics, ok := scannerFiles(runtime, AgentMuse, root)
		result.Diagnostics = append(result.Diagnostics, diagnostics...)
		if !ok {
			continue
		}
		for _, entry := range files {
			fact, diagnostics, ok := scanMuseFile(runtime, entry)
			result.Diagnostics = append(result.Diagnostics, diagnostics...)
			if ok {
				result.Facts = append(result.Facts, fact)
			}
		}
	}
	return result
}

func scanMuseFile(runtime Runtime, entry FileEntry) (Fact, []Diagnostic, bool) {
	artifact := entry.Artifact
	artifact.Kind = ArtifactTranscript
	nativeID, parentID, role, recognized := musePathFact(artifact.RelativePath)
	if !recognized {
		if strings.HasSuffix(artifact.RelativePath, "session.jsonl") {
			return Fact{}, []Diagnostic{artifactDiagnostic(DiagnosticSchemaNearMiss, AgentMuse, nil, artifact, "unrecognized Muse v1 path")}, false
		}
		return Fact{}, nil, false
	}

	var diagnostics []Diagnostic
	contradiction := false
	err := visitJSONLines(runtime, artifact, jsonRecordLimit, func(line []byte) bool {
		if len(line) == 0 {
			return false
		}
		var record struct {
			PayloadType string `json:"payload_type"`
			Payload     struct {
				Kind  string `json:"kind"`
				RunID string `json:"run_id"`
				Event struct {
					Kind   string `json:"kind"`
					Prompt string `json:"prompt"`
				} `json:"event"`
			} `json:"payload"`
		}
		if err := decodeStrictJSON(line, &record); err != nil {
			diagnostics = append(diagnostics, artifactDiagnostic(DiagnosticSchemaNearMiss, AgentMuse, &nativeID, artifact, "malformed Muse JSONL record"))
			return false
		}
		if record.PayloadType != "runtime.session" || record.Payload.Kind != "run" || record.Payload.Event.Kind != "started" {
			return false
		}
		if role == RoleSubagent && record.Payload.RunID != "" && record.Payload.RunID != nativeID {
			contradiction = true
			return true
		}
		return false
	})
	if err != nil {
		diagnostics = append(diagnostics, artifactDiagnostic(DiagnosticSchemaNearMiss, AgentMuse, &nativeID, artifact, err.Error()))
	}
	if contradiction {
		diagnostics = append(diagnostics, artifactDiagnostic(DiagnosticNodeMalformed, AgentMuse, &nativeID, artifact, "Muse child run_id contradicts path identity"))
	}
	return Fact{
		Agent:          AgentMuse,
		NativeID:       nativeID,
		Role:           role,
		ParentID:       parentID,
		Time:           fallbackTime(entry),
		Resumable:      role == RoleRoot && !contradiction,
		Disputed:       contradiction,
		Artifacts:      []Artifact{artifact},
		EdgeProvenance: edgeProvenance(role, "muse-v1", artifact),
	}, diagnostics, true
}

func musePathFact(relativePath string) (string, *string, Role, bool) {
	parts := strings.Split(path.Clean(relativePath), "/")
	if len(parts) == 5 && validDateParts(parts[:3]) && uuidPattern.MatchString(parts[3]) && parts[4] == "session.jsonl" {
		return parts[3], nil, RoleRoot, true
	}
	if len(parts) == 7 && validDateParts(parts[:3]) && uuidPattern.MatchString(parts[3]) && parts[4] == "subagent" && uuidPattern.MatchString(parts[5]) && parts[6] == "session.jsonl" {
		parentID := parts[3]
		return parts[5], &parentID, RoleSubagent, true
	}
	return "", nil, RoleUnknown, false
}

func validDateParts(parts []string) bool {
	return len(parts) == 3 && codexYearPattern.MatchString(parts[0]) && codexDatePartPattern.MatchString(parts[1]) && codexDatePartPattern.MatchString(parts[2])
}
