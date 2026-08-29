package sessioninventory

import (
	"bytes"
	"encoding/json"
	"errors"
	"path"
	"regexp"
	"strconv"
	"strings"
)

var (
	codexDatePartPattern = regexp.MustCompile(`^[0-9]{2}$`)
	codexYearPattern     = regexp.MustCompile(`^[0-9]{4}$`)
	codexRolloutPattern  = regexp.MustCompile(`^rollout-.+-([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\.jsonl$`)
)

func ScanCodex(runtime Runtime) ScanResult {
	var result ScanResult
	for _, root := range runtime.NativeRoots(AgentCodex) {
		files, diagnostics, ok := scannerFiles(runtime, AgentCodex, root)
		result.Diagnostics = append(result.Diagnostics, diagnostics...)
		if !ok {
			continue
		}
		for _, entry := range files {
			fact, diagnostics, ok := scanCodexFile(runtime, entry)
			result.Diagnostics = append(result.Diagnostics, diagnostics...)
			if ok {
				result.Facts = append(result.Facts, fact)
			}
		}
	}
	return result
}

func scanCodexFile(runtime Runtime, entry FileEntry) (Fact, []Diagnostic, bool) {
	artifact := entry.Artifact
	artifact.Kind = ArtifactTranscript
	entry.Artifact = artifact
	nativeID, recognized := codexPathID(artifact.RelativePath)
	if !recognized {
		if strings.HasSuffix(artifact.RelativePath, ".jsonl") {
			return Fact{}, []Diagnostic{artifactDiagnostic(DiagnosticSchemaNearMiss, AgentCodex, nil, artifact, "unrecognized Codex v1 path")}, false
		}
		return Fact{}, nil, false
	}

	state := newCodexScannerState(entry, nativeID)
	var diagnostics []Diagnostic
	err := visitJSONLines(runtime, artifact, metadataRecordLimit, func(line []byte) bool {
		applyCodexRecord(&state, entry, line, &diagnostics)
		return false
	})
	if err != nil || !state.FirstRecordValidated {
		detail := "missing or invalid first Codex session_meta"
		if err != nil {
			detail = err.Error()
		}
		diagnostics = append(diagnostics, artifactDiagnostic(DiagnosticSchemaNearMiss, AgentCodex, &nativeID, artifact, detail))
		return Fact{}, diagnostics, false
	}
	return scannerStateFact(state, []Artifact{artifact}), diagnostics, true
}

type codexRecord struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type codexSessionMeta struct {
	ID             string          `json:"id"`
	ParentThreadID *string         `json:"parent_thread_id"`
	Source         json.RawMessage `json:"source"`
}

// ValidateCodexDelta preserves the first session_meta invariant while checking
// every later complete JSONL record through the observed EOF.
func ValidateCodexDelta(entry FileEntry, prior *ScannerState, records []FramedJSONLRecord) (ScannerState, []Diagnostic, error) {
	artifact := entry.Artifact
	artifact.Kind = ArtifactTranscript
	entry.Artifact = artifact
	nativeID, recognized := codexPathID(artifact.RelativePath)
	if !recognized {
		return ScannerState{}, nil, errors.New("unrecognized Codex v1 path")
	}
	state := newCodexScannerState(entry, nativeID)
	if prior != nil {
		if err := ValidateScannerState(*prior); err != nil {
			return ScannerState{}, nil, err
		}
		state = cloneScannerState(*prior)
		if state.Agent != AgentCodex || state.NativeID != nativeID || state.IdentityAnchor != nativeID || state.ScannerSchema != "codex-v1" {
			return ScannerState{}, nil, errors.New("Codex scanner state does not match artifact")
		}
	}
	var diagnostics []Diagnostic
	for _, record := range records {
		applyCodexRecord(&state, entry, record.Bytes, &diagnostics)
	}
	if !state.FirstRecordValidated {
		state.Disputed = true
	}
	if err := ValidateScannerState(state); err != nil {
		return ScannerState{}, diagnostics, err
	}
	return state, diagnostics, nil
}

func newCodexScannerState(entry FileEntry, nativeID string) ScannerState {
	return ScannerState{Version: ScannerStateVersion, Agent: AgentCodex, NativeID: nativeID, IdentityAnchor: nativeID, Role: RoleUnknown, ScannerSchema: "codex-v1", Chronology: fallbackTime(entry)}
}

func applyCodexRecord(state *ScannerState, entry FileEntry, line []byte, diagnostics *[]Diagnostic) {
	artifact := entry.Artifact
	artifact.Kind = ArtifactTranscript
	invalidFirst := func(detail string) {
		state.Disputed = true
		*diagnostics = append(*diagnostics, artifactDiagnostic(DiagnosticSchemaNearMiss, AgentCodex, &state.NativeID, artifact, detail))
	}
	if len(line) == 0 {
		if !state.FirstRecordValidated {
			invalidFirst("missing or invalid first Codex session_meta")
		}
		return
	}
	var record codexRecord
	if err := decodeStrictJSON(line, &record); err != nil {
		invalidFirst("malformed Codex JSONL record")
		return
	}
	if !state.FirstRecordValidated && record.Type != "session_meta" {
		invalidFirst("missing or invalid first Codex session_meta")
		return
	}
	if record.Type != "session_meta" {
		return
	}
	var metadata codexSessionMeta
	if decodeStrictJSON(record.Payload, &metadata) != nil || !uuidPattern.MatchString(metadata.ID) {
		invalidFirst("missing or invalid first Codex session_meta")
		return
	}
	role, parentID, ok := codexRole(metadata.ParentThreadID, metadata.Source)
	if !ok {
		invalidFirst("Codex source/parent shape is not allowlisted")
		return
	}
	if metadata.ID != state.NativeID {
		state.Role = RoleUnknown
		state.ParentID = nil
		state.FirstRecordValidated = true
		state.Disputed = true
		if parsed := metadataTime(record.Timestamp); parsed != nil {
			state.Chronology = parsed
		}
		*diagnostics = append(*diagnostics, artifactDiagnostic(DiagnosticParentConflict, AgentCodex, &state.NativeID, artifact, "Codex metadata ID disagrees with path ID"))
		return
	}
	if state.FirstRecordValidated && (state.Role != role || !equalString(state.ParentID, parentID)) {
		state.Disputed = true
		*diagnostics = append(*diagnostics, artifactDiagnostic(DiagnosticParentConflict, AgentCodex, &state.NativeID, artifact, "Codex metadata role disagrees with authorized state"))
		return
	}
	wasValidated, wasDisputed := state.FirstRecordValidated, state.Disputed
	state.Role = role
	state.ParentID = cloneString(parentID)
	state.IdentityAnchor = state.NativeID
	state.FirstRecordValidated = true
	if !wasValidated && !wasDisputed {
		state.Disputed = false
	}
	if parsed := metadataTime(record.Timestamp); parsed != nil {
		state.Chronology = parsed
	}
}

func codexPathID(relativePath string) (string, bool) {
	parts := strings.Split(path.Clean(relativePath), "/")
	if len(parts) != 4 || !codexYearPattern.MatchString(parts[0]) || !codexDatePartPattern.MatchString(parts[1]) || !codexDatePartPattern.MatchString(parts[2]) {
		return "", false
	}
	match := codexRolloutPattern.FindStringSubmatch(parts[3])
	if len(match) != 2 || !uuidPattern.MatchString(match[1]) {
		return "", false
	}
	return match[1], true
}

func codexRole(parentID *string, source json.RawMessage) (Role, *string, bool) {
	var sourceString string
	if json.Unmarshal(source, &sourceString) == nil {
		if parentID == nil && (sourceString == "cli" || sourceString == "exec") {
			return RoleRoot, nil, true
		}
		return RoleUnknown, nil, false
	}
	if parentID == nil || *parentID == "" || !uuidPattern.MatchString(*parentID) || !validCodexSubagentObject(source, *parentID) {
		return RoleUnknown, nil, false
	}
	return RoleSubagent, cloneString(parentID), true
}

func validCodexSubagentObject(source json.RawMessage, parentID string) bool {
	var object map[string]json.RawMessage
	if json.Unmarshal(source, &object) != nil || len(object) != 1 {
		return false
	}
	subagent, ok := object["subagent"]
	if !ok {
		return false
	}
	var shape map[string]json.RawMessage
	if json.Unmarshal(subagent, &shape) != nil {
		return false
	}
	if len(shape) == 0 {
		return true
	}
	if len(shape) != 1 {
		return false
	}
	spawn, ok := shape["thread_spawn"]
	if !ok {
		return false
	}
	var spawnShape map[string]json.RawMessage
	if json.Unmarshal(spawn, &spawnShape) != nil {
		return false
	}
	if len(spawnShape) == 1 {
		depthRaw, ok := spawnShape["depth"]
		return ok && validCodexDepth(depthRaw)
	}
	if len(spawnShape) != 5 {
		return false
	}
	depthRaw, depthOK := spawnShape["depth"]
	nicknameRaw, nicknameOK := spawnShape["agent_nickname"]
	pathRaw, pathOK := spawnShape["agent_path"]
	roleRaw, roleOK := spawnShape["agent_role"]
	parentRaw, parentOK := spawnShape["parent_thread_id"]
	if !depthOK || !nicknameOK || !pathOK || !roleOK || !parentOK || !validCodexDepth(depthRaw) || !jsonString(nicknameRaw) || !jsonStringOrNull(pathRaw) || !jsonStringOrNull(roleRaw) {
		return false
	}
	var nestedParent string
	return json.Unmarshal(parentRaw, &nestedParent) == nil && nestedParent == parentID
}

func validCodexDepth(depthRaw json.RawMessage) bool {
	decoder := json.NewDecoder(bytes.NewReader(depthRaw))
	decoder.UseNumber()
	var depth json.Number
	if decoder.Decode(&depth) != nil {
		return false
	}
	integer, err := strconv.ParseInt(string(depth), 10, 64)
	return err == nil && integer >= 1
}

func jsonString(raw json.RawMessage) bool {
	var value string
	return json.Unmarshal(raw, &value) == nil
}

func jsonStringOrNull(raw json.RawMessage) bool {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return true
	}
	return jsonString(raw)
}
