package sessioninventory

import (
	"errors"
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
	entry.Artifact = artifact
	nativeID, _, _, recognized := musePathFact(artifact.RelativePath)
	if !recognized {
		if strings.HasSuffix(artifact.RelativePath, "session.jsonl") {
			return Fact{}, []Diagnostic{artifactDiagnostic(DiagnosticSchemaNearMiss, AgentMuse, nil, artifact, "unrecognized Muse v1 path")}, false
		}
		return Fact{}, nil, false
	}

	state, diagnostics, err := ValidateMuseDelta(entry, nil, nil)
	if err != nil {
		return Fact{}, []Diagnostic{artifactDiagnostic(DiagnosticSchemaNearMiss, AgentMuse, &nativeID, artifact, err.Error())}, false
	}
	err = visitJSONLines(runtime, artifact, jsonRecordLimit, func(line []byte) bool {
		applyMuseRecord(&state, entry, line, &diagnostics)
		return false
	})
	if err != nil {
		diagnostics = append(diagnostics, artifactDiagnostic(DiagnosticSchemaNearMiss, AgentMuse, &nativeID, artifact, err.Error()))
		state.Disputed = true
	}
	return scannerStateFact(state, []Artifact{artifact}), diagnostics, true
}

type museRecord struct {
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

func ValidateMuseDelta(entry FileEntry, prior *ScannerState, records []FramedJSONLRecord) (ScannerState, []Diagnostic, error) {
	artifact := entry.Artifact
	artifact.Kind = ArtifactTranscript
	entry.Artifact = artifact
	nativeID, parentID, role, recognized := musePathFact(artifact.RelativePath)
	if !recognized {
		return ScannerState{}, nil, errors.New("unrecognized Muse v1 path")
	}
	anchor := nativeID
	if parentID != nil {
		anchor = *parentID
	}
	state := ScannerState{Version: ScannerStateVersion, Agent: AgentMuse, NativeID: nativeID, IdentityAnchor: anchor, Role: role, ParentID: cloneString(parentID), ScannerSchema: "muse-v1", Chronology: fallbackTime(entry)}
	if prior != nil {
		if err := ValidateScannerState(*prior); err != nil {
			return ScannerState{}, nil, err
		}
		state = cloneScannerState(*prior)
		if state.Agent != AgentMuse || state.NativeID != nativeID || state.IdentityAnchor != anchor || state.Role != role || !equalString(state.ParentID, parentID) || state.ScannerSchema != "muse-v1" {
			return ScannerState{}, nil, errors.New("Muse scanner state does not match artifact")
		}
	}
	var diagnostics []Diagnostic
	for _, record := range records {
		applyMuseRecord(&state, entry, record.Bytes, &diagnostics)
	}
	if err := ValidateScannerState(state); err != nil {
		return ScannerState{}, diagnostics, err
	}
	return state, diagnostics, nil
}

func applyMuseRecord(state *ScannerState, entry FileEntry, line []byte, diagnostics *[]Diagnostic) {
	if len(line) == 0 {
		return
	}
	artifact := entry.Artifact
	artifact.Kind = ArtifactTranscript
	var record museRecord
	if err := decodeStrictJSON(line, &record); err != nil {
		state.Disputed = true
		*diagnostics = append(*diagnostics, artifactDiagnostic(DiagnosticSchemaNearMiss, AgentMuse, &state.NativeID, artifact, "malformed Muse JSONL record"))
		return
	}
	state.FirstRecordValidated = true
	if record.PayloadType == "runtime.session" && record.Payload.Kind == "run" && record.Payload.Event.Kind == "started" && state.Role == RoleSubagent && record.Payload.RunID != "" && record.Payload.RunID != state.NativeID {
		state.Disputed = true
		*diagnostics = append(*diagnostics, artifactDiagnostic(DiagnosticNodeMalformed, AgentMuse, &state.NativeID, artifact, "Muse child run_id contradicts path identity"))
	}
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
