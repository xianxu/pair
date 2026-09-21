package sessioninventory

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"
)

// claudeFamilySpec carries every grammar admission a claude-family member
// needs, so a qoder-only widening cannot silently change claude's scanner.
// The shared record type stays conservative (ISO timestamps only) and each
// spec opts into what its agent measurably writes.
type claudeFamilySpec struct {
	agent Agent
	// schema names this producer's reviewed record contract; diagnostic
	// details and ProviderContractFor key off it.
	schema string
	// acceptsMillis admits integer epoch-millisecond timestamps. Claude
	// writes ISO strings only; qoder's bookkeeping records (runtime-config,
	// active-leaf) carry epoch millis.
	acceptsMillis bool
}

var claudeFamilySpecs = map[Agent]claudeFamilySpec{
	AgentClaude: {agent: AgentClaude, schema: "claude-v1"},
	AgentQoder:  {agent: AgentQoder, schema: "qoder-v1", acceptsMillis: true},
}

// millisFloor/millisCeil bound the epoch-millisecond path to 2000-01-01 ..
// 9999-12-31, matching what the ISO path can represent: a native transcript is
// this program's untrusted input, and an unbounded time.UnixMilli either
// fabricates a year the rest of the pipeline cannot marshal (year 292278994)
// or yields a zero time that fails state validation.
const (
	millisFloor = int64(946684800000)
	millisCeil  = int64(253402300799999)
)

func claudeFamilySpecFor(agent Agent) (claudeFamilySpec, bool) {
	spec, ok := claudeFamilySpecs[agent]
	return spec, ok
}

type claudeRecord struct {
	Timestamp   claudeFamilyTime `json:"timestamp"`
	SessionID   string           `json:"sessionId"`
	IsSidechain *bool            `json:"isSidechain"`
	Type        string           `json:"type"`
	Message     struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

// claudeFamilyTime accepts the timestamp encodings the claude family can
// carry: ISO-8601 strings (every message record) and epoch-millisecond
// numbers (qoder's bookkeeping records). Whether a number is admissible is
// the per-agent spec's decision, made in applyClaudeFamilyRecord.
type claudeFamilyTime struct {
	ISO    string
	Millis *int64
}

func (value *claudeFamilyTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	if len(data) != 0 && data[0] == '"' {
		return json.Unmarshal(data, &value.ISO)
	}
	var millis int64
	if err := json.Unmarshal(data, &millis); err != nil {
		return err
	}
	value.Millis = &millis
	return nil
}

func (value claudeFamilyTime) nativeTime(spec claudeFamilySpec) (*NativeTime, error) {
	if value.Millis == nil {
		return metadataTime(value.ISO), nil
	}
	if !spec.acceptsMillis {
		return nil, fmt.Errorf("%s does not write numeric timestamps", spec.schema)
	}
	if *value.Millis < millisFloor || *value.Millis > millisCeil {
		return nil, fmt.Errorf("%s timestamp %d outside the representable window", spec.schema, *value.Millis)
	}
	return &NativeTime{Value: time.UnixMilli(*value.Millis).UTC(), Source: TimeSourceMetadata}, nil
}

func ScanClaude(runtime Runtime) ScanResult {
	return scanClaudeFamily(runtime, AgentClaude)
}

// scanClaudeFamily runs the claude-family record transition for one agent.
// Claude and qoder transcripts share the record shape
// (type/sessionId/isSidechain/timestamp/message.role) and the path layout
// (<project>/<uuid>.jsonl, <uuid>/subagents/agent-*.jsonl); what differs is
// carried by the agent's claudeFamilySpec.
func scanClaudeFamily(runtime Runtime, agent Agent) ScanResult {
	spec, ok := claudeFamilySpecFor(agent)
	if !ok {
		return ScanResult{Diagnostics: []Diagnostic{diagnostic(DiagnosticSchemaNearMiss, agent, nil, "unsupported agent")}}
	}
	var result ScanResult
	for _, root := range runtime.NativeRoots(agent) {
		files, diagnostics, ok := scannerFiles(runtime, agent, root)
		result.Diagnostics = append(result.Diagnostics, diagnostics...)
		if !ok {
			continue
		}
		for _, entry := range files {
			fact, diagnostics, ok := scanClaudeFamilyFile(runtime, entry, spec)
			result.Diagnostics = append(result.Diagnostics, diagnostics...)
			if ok {
				result.Facts = append(result.Facts, fact)
			}
		}
	}
	return result
}

func scanClaudeFamilyFile(runtime Runtime, entry FileEntry, spec claudeFamilySpec) (Fact, []Diagnostic, bool) {
	agent, schema := spec.agent, spec.schema
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

	state, diagnostics, err := validateClaudeFamilyDelta(entry, nil, nil, spec)
	if err != nil {
		return Fact{}, []Diagnostic{artifactDiagnostic(DiagnosticSchemaNearMiss, agent, &nativeID, artifact, err.Error())}, false
	}
	err = visitJSONLines(runtime, artifact, unlimitedRecordSize, func(line []byte) bool {
		applyClaudeFamilyRecord(&state, entry, line, &diagnostics, spec)
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
	spec, _ := claudeFamilySpecFor(AgentClaude)
	return validateClaudeFamilyDelta(entry, prior, records, spec)
}

func validateClaudeFamilyDelta(entry FileEntry, prior *ScannerState, records []FramedJSONLRecord, spec claudeFamilySpec) (ScannerState, []Diagnostic, error) {
	agent, schema := spec.agent, spec.schema
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
		applyClaudeFamilyRecord(&state, entry, record.Bytes, &diagnostics, spec)
	}
	if err := ValidateScannerState(state); err != nil {
		return ScannerState{}, diagnostics, err
	}
	return state, diagnostics, nil
}

func applyClaudeFamilyRecord(state *ScannerState, entry FileEntry, line []byte, diagnostics *[]Diagnostic, spec claudeFamilySpec) {
	if len(line) == 0 {
		return
	}
	agent, schema := spec.agent, spec.schema
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
		parsed, err := record.Timestamp.nativeTime(spec)
		if err != nil {
			state.Disputed = true
			*diagnostics = append(*diagnostics, artifactDiagnostic(DiagnosticNodeMalformed, agent, &state.NativeID, artifact, err.Error()))
			return
		}
		if parsed != nil {
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
