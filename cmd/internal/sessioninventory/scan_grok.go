package sessioninventory

import (
	"encoding/json"
	"errors"
	"net/url"
	"path"
	"strings"
	"time"
)

// grok 1.0.46 keeps one directory per session under
// ~/.grok/sessions/<url-encoded cwd>/<uuidv7>/. Its updates.jsonl is the
// authoritative conversation log: one JSON-RPC-shaped ACP record per line,
// every one naming the session it belongs to. The directory's other files
// (summary.json, events.jsonl, chat_history.jsonl, …) and the per-cwd
// prompt_history.jsonl are not artifacts, and a session directory with no
// updates.jsonl (a stub `grok login` leaves behind) holds no conversation.
// Subagent transcripts under <session>/subagents/ are out of scope (#410) and
// skipped silently.
const (
	grokSessionsRoot = "grok-sessions"
	grokSchema       = "grok-v1"
	grokTranscript   = "updates.jsonl"
)

// grokUpdateMethods are the JSON-RPC methods a grok updates.jsonl record may
// carry (measured at 1.0.46): standard ACP updates and grok's own extension
// (turn_completed). The ONE set the scanner, the event normalizer and the
// context-meter parser accept.
var grokUpdateMethods = map[string]bool{"session/update": true, "_x.ai/session/update": true}

// grokTimestampBounds keeps a record's epoch-seconds chronology inside the
// window the rest of the pipeline can marshal (the qoder BR-19 rule).
var grokTimestampBounds = [2]int64{
	time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
	time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC).Unix(),
}

func ScanGrok(runtime Runtime) ScanResult {
	var result ScanResult
	for _, root := range runtime.NativeRoots(AgentGrok) {
		files, diagnostics, ok := scannerFiles(runtime, AgentGrok, root)
		result.Diagnostics = append(result.Diagnostics, diagnostics...)
		if !ok {
			continue
		}
		for _, entry := range files {
			fact, diagnostics, ok := scanGrokFile(runtime, entry)
			result.Diagnostics = append(result.Diagnostics, diagnostics...)
			if ok {
				result.Facts = append(result.Facts, fact)
			}
		}
	}
	return result
}

func scanGrokFile(runtime Runtime, entry FileEntry) (Fact, []Diagnostic, bool) {
	artifact := entry.Artifact
	artifact.Kind = ArtifactTranscript
	entry.Artifact = artifact
	nativeID, recognized := grokPathFact(artifact.RelativePath)
	if !recognized {
		if path.Base(artifact.RelativePath) == grokTranscript && !strings.Contains(artifact.RelativePath, "/subagents/") {
			return Fact{}, []Diagnostic{artifactDiagnostic(DiagnosticSchemaNearMiss, AgentGrok, nil, artifact, "unrecognized Grok v1 path")}, false
		}
		return Fact{}, nil, false
	}
	state := newGrokScannerState(entry, nativeID)
	var diagnostics []Diagnostic
	err := visitJSONLines(runtime, artifact, unlimitedRecordSize, func(line []byte) bool {
		applyGrokRecord(&state, entry, line, &diagnostics)
		return false
	})
	if err != nil {
		diagnostics = append(diagnostics, artifactDiagnostic(DiagnosticSchemaNearMiss, AgentGrok, &nativeID, artifact, err.Error()))
		state.Disputed = true
	}
	return scannerStateFact(state, []Artifact{artifact}), diagnostics, true
}

// grokRecord is the envelope every updates.jsonl line carries. Only the fields
// identity and chronology rest on are decoded here; the event normalizer reads
// the update itself.
type grokRecord struct {
	Timestamp json.RawMessage `json:"timestamp"`
	Method    string          `json:"method"`
	Params    struct {
		SessionID string `json:"sessionId"`
		Update    struct {
			SessionUpdate string `json:"sessionUpdate"`
		} `json:"update"`
	} `json:"params"`
}

// ValidateGrokDelta applies complete records to a cloned scanner state. A
// record that is not the envelope, or names another session, disputes the
// transcript: the path identity is the only one a Grok transcript has.
func ValidateGrokDelta(entry FileEntry, prior *ScannerState, records []FramedJSONLRecord) (ScannerState, []Diagnostic, error) {
	artifact := entry.Artifact
	artifact.Kind = ArtifactTranscript
	entry.Artifact = artifact
	nativeID, recognized := grokPathFact(artifact.RelativePath)
	if !recognized {
		return ScannerState{}, nil, errors.New("unrecognized Grok v1 path")
	}
	state := newGrokScannerState(entry, nativeID)
	if prior != nil {
		if err := ValidateScannerState(*prior); err != nil {
			return ScannerState{}, nil, err
		}
		state = cloneScannerState(*prior)
		if state.Agent != AgentGrok || state.NativeID != nativeID || state.IdentityAnchor != nativeID || state.Role != RoleRoot || state.ScannerSchema != grokSchema {
			return ScannerState{}, nil, errors.New("Grok scanner state does not match artifact")
		}
	}
	var diagnostics []Diagnostic
	for _, record := range records {
		applyGrokRecord(&state, entry, record.Bytes, &diagnostics)
	}
	if err := ValidateScannerState(state); err != nil {
		return ScannerState{}, diagnostics, err
	}
	return state, diagnostics, nil
}

func newGrokScannerState(entry FileEntry, nativeID string) ScannerState {
	return ScannerState{Version: ScannerStateVersion, Agent: AgentGrok, NativeID: nativeID, IdentityAnchor: nativeID, Role: RoleRoot, ScannerSchema: grokSchema, Chronology: fallbackTime(entry)}
}

func applyGrokRecord(state *ScannerState, entry FileEntry, line []byte, diagnostics *[]Diagnostic) {
	if len(line) == 0 {
		return
	}
	artifact := entry.Artifact
	artifact.Kind = ArtifactTranscript
	dispute := func(code DiagnosticCode, detail string) {
		state.Disputed = true
		*diagnostics = append(*diagnostics, artifactDiagnostic(code, AgentGrok, &state.NativeID, artifact, detail))
	}
	var record grokRecord
	if err := decodeStrictJSON(line, &record); err != nil || !grokUpdateMethods[record.Method] || record.Params.Update.SessionUpdate == "" {
		dispute(DiagnosticSchemaNearMiss, "malformed Grok JSONL record")
		return
	}
	if record.Params.SessionID != state.NativeID {
		dispute(DiagnosticNodeMalformed, "Grok record sessionId contradicts path identity")
		return
	}
	seconds, ok := grokEpochSeconds(record.Timestamp)
	if !ok {
		dispute(DiagnosticSchemaNearMiss, "Grok record timestamp is not bounded epoch seconds")
		return
	}
	if !state.FirstRecordValidated {
		state.Chronology = &NativeTime{Value: time.Unix(seconds, 0).UTC(), Source: TimeSourceMetadata}
	}
	state.FirstRecordValidated = true
}

func grokEpochSeconds(raw json.RawMessage) (int64, bool) {
	var seconds int64
	if json.Unmarshal(raw, &seconds) != nil {
		return 0, false
	}
	return seconds, seconds >= grokTimestampBounds[0] && seconds <= grokTimestampBounds[1]
}

// grokPathFact recognizes <url-encoded absolute cwd>/<uuid>/updates.jsonl.
func grokPathFact(relativePath string) (string, bool) {
	parts := strings.Split(path.Clean(relativePath), "/")
	if len(parts) != 3 || parts[2] != grokTranscript || !uuidPattern.MatchString(parts[1]) {
		return "", false
	}
	cwd, err := url.PathUnescape(parts[0])
	if err != nil || !strings.HasPrefix(cwd, "/") {
		return "", false
	}
	return parts[1], true
}
