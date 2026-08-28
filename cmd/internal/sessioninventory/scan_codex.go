package sessioninventory

import (
	"bytes"
	"encoding/json"
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
	nativeID, recognized := codexPathID(artifact.RelativePath)
	if !recognized {
		if strings.HasSuffix(artifact.RelativePath, ".jsonl") {
			return Fact{}, []Diagnostic{artifactDiagnostic(DiagnosticSchemaNearMiss, AgentCodex, nil, artifact, "unrecognized Codex v1 path")}, false
		}
		return Fact{}, nil, false
	}

	var record struct {
		Timestamp string `json:"timestamp"`
		Type      string `json:"type"`
		Payload   struct {
			ID             string          `json:"id"`
			ParentThreadID *string         `json:"parent_thread_id"`
			Source         json.RawMessage `json:"source"`
		} `json:"payload"`
	}
	seen := false
	err := visitJSONLines(runtime, artifact, metadataRecordLimit, func(line []byte) bool {
		seen = true
		if len(line) != 0 {
			_ = decodeStrictJSON(line, &record)
		}
		return true
	})
	if err != nil || !seen || record.Type != "session_meta" || !uuidPattern.MatchString(record.Payload.ID) {
		detail := "missing or invalid first Codex session_meta"
		if err != nil {
			detail = err.Error()
		}
		return Fact{}, []Diagnostic{artifactDiagnostic(DiagnosticSchemaNearMiss, AgentCodex, &nativeID, artifact, detail)}, false
	}
	chronology := metadataTime(record.Timestamp)
	if chronology == nil {
		chronology = fallbackTime(entry)
	}
	if record.Payload.ID != nativeID {
		return Fact{
			Agent:     AgentCodex,
			NativeID:  nativeID,
			Role:      RoleUnknown,
			Time:      chronology,
			Disputed:  true,
			Artifacts: []Artifact{artifact},
		}, []Diagnostic{artifactDiagnostic(DiagnosticParentConflict, AgentCodex, &nativeID, artifact, "Codex metadata ID disagrees with path ID")}, true
	}

	role, parentID, ok := codexRole(record.Payload.ParentThreadID, record.Payload.Source)
	if !ok {
		return Fact{}, []Diagnostic{artifactDiagnostic(DiagnosticSchemaNearMiss, AgentCodex, &nativeID, artifact, "Codex source/parent shape is not allowlisted")}, false
	}
	return Fact{
		Agent:     AgentCodex,
		NativeID:  nativeID,
		Role:      role,
		ParentID:  parentID,
		Time:      chronology,
		Resumable: role == RoleRoot,
		Artifacts: []Artifact{artifact},
	}, nil, true
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
