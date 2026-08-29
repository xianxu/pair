package sessioninventory

import (
	"bytes"
	"encoding/json"
	"errors"
	"path"
	"sort"
	"strings"
)

const (
	agyTrajectorySchemaQuery = "pragma table_info(trajectory_meta)"
	agyTrajectoryFactsQuery  = "select cascade_id, typeof(cascade_id), typeof(trajectory_type), typeof(source) from trajectory_meta limit 2"
)

var sqliteHeader = []byte("SQLite format 3\x00")

func ScanAgy(runtime Runtime) ScanResult {
	var result ScanResult
	databases := make(map[string]FileEntry)
	transcripts := make(map[string]FileEntry)
	for _, root := range runtime.NativeRoots(AgentAgy) {
		files, diagnostics, ok := scannerFiles(runtime, AgentAgy, root)
		result.Diagnostics = append(result.Diagnostics, diagnostics...)
		if !ok {
			continue
		}
		for _, entry := range files {
			switch root.Name {
			case "agy-conversations":
				if nativeID, ok := agyDatabasePathID(entry.Artifact.RelativePath); ok {
					databases[nativeID] = entry
				} else if strings.HasSuffix(entry.Artifact.RelativePath, ".db") {
					artifact := entry.Artifact
					artifact.Kind = ArtifactDatabase
					result.Diagnostics = append(result.Diagnostics, artifactDiagnostic(DiagnosticSchemaNearMiss, AgentAgy, nil, artifact, "unrecognized Agy v1 database path"))
				}
			case "agy-brain":
				if nativeID, ok := agyTranscriptPathID(entry.Artifact.RelativePath); ok {
					transcripts[nativeID] = entry
				}
			}
		}
	}

	ids := make([]string, 0, len(databases))
	for nativeID := range databases {
		ids = append(ids, nativeID)
	}
	sort.Strings(ids)
	for _, nativeID := range ids {
		entry := databases[nativeID]
		fact, diagnostics, ok := scanAgyDatabase(runtime, nativeID, entry, transcripts[nativeID])
		result.Diagnostics = append(result.Diagnostics, diagnostics...)
		if ok {
			result.Facts = append(result.Facts, fact)
		}
		delete(transcripts, nativeID)
	}
	for nativeID, entry := range transcripts {
		artifact := entry.Artifact
		artifact.Kind = ArtifactTranscript
		result.Diagnostics = append(result.Diagnostics, artifactDiagnostic(DiagnosticParentMissing, AgentAgy, &nativeID, artifact, "Agy transcript has no authorized conversation database"))
	}
	return result
}

func scanAgyDatabase(runtime Runtime, nativeID string, databaseEntry, transcriptEntry FileEntry) (Fact, []Diagnostic, bool) {
	database := databaseEntry.Artifact
	database.Kind = ArtifactDatabase
	artifacts := []Artifact{database}
	var diagnostics []Diagnostic
	if transcriptEntry.Artifact.RelativePath == "" {
		if detail := validateAgyDatabaseEvidence(runtime, database, nativeID); detail != "" {
			return Fact{}, []Diagnostic{artifactDiagnostic(DiagnosticSchemaNearMiss, AgentAgy, &nativeID, database, detail)}, false
		}
		diagnostics = append(diagnostics, artifactDiagnostic(DiagnosticParentMissing, AgentAgy, &nativeID, database, "Agy transcript join is missing"))
		return Fact{Agent: AgentAgy, NativeID: nativeID, Role: RoleRoot, Time: fallbackTime(databaseEntry), Resumable: true, Artifacts: artifacts}, diagnostics, true
	}

	transcript := transcriptEntry.Artifact
	transcript.Kind = ArtifactTranscript
	transcriptEntry.Artifact = transcript
	var records []FramedJSONLRecord
	err := visitJSONLinesAt(runtime, transcript, jsonRecordLimit, func(line []byte, offset uint64) bool {
		records = append(records, FramedJSONLRecord{Offset: int64(offset), Bytes: append([]byte(nil), line...)})
		return false
	})
	state, found, validateErr := ValidateAgyDelta(runtime, databaseEntry, transcriptEntry, nil, records)
	diagnostics = append(diagnostics, found...)
	if validateErr != nil {
		return Fact{}, append(diagnostics, artifactDiagnostic(DiagnosticSchemaNearMiss, AgentAgy, &nativeID, transcript, validateErr.Error())), false
	}
	if err != nil {
		state.Disputed = true
		diagnostics = append(diagnostics, artifactDiagnostic(DiagnosticSchemaNearMiss, AgentAgy, &nativeID, transcript, err.Error()))
	}
	if !state.FirstRecordValidated {
		return Fact{}, diagnostics, false
	}
	artifacts = append(artifacts, transcript)
	return scannerStateFact(state, artifacts), diagnostics, true
}

// ValidateAgyDelta revalidates the keyed SQLite identity seam and applies
// complete joined-transcript records to one cloned root state. SQLite is never
// treated as append-only.
func ValidateAgyDelta(runtime Runtime, databaseEntry, transcriptEntry FileEntry, prior *ScannerState, records []FramedJSONLRecord) (ScannerState, []Diagnostic, error) {
	database := databaseEntry.Artifact
	database.Kind = ArtifactDatabase
	databaseEntry.Artifact = database
	transcript := transcriptEntry.Artifact
	transcript.Kind = ArtifactTranscript
	transcriptEntry.Artifact = transcript
	nativeID, databaseRecognized := agyDatabasePathID(database.RelativePath)
	transcriptID, transcriptRecognized := agyTranscriptPathID(transcript.RelativePath)
	if !databaseRecognized || !transcriptRecognized || transcriptID != nativeID {
		return ScannerState{}, nil, errors.New("Agy database/transcript join is invalid")
	}
	state := ScannerState{Version: ScannerStateVersion, Agent: AgentAgy, NativeID: nativeID, IdentityAnchor: nativeID, Role: RoleRoot, ScannerSchema: "agy-v1", Chronology: fallbackTime(databaseEntry)}
	if prior != nil {
		if err := ValidateScannerState(*prior); err != nil {
			return ScannerState{}, nil, err
		}
		state = cloneScannerState(*prior)
		if state.Agent != AgentAgy || state.NativeID != nativeID || state.IdentityAnchor != nativeID || state.Role != RoleRoot || state.ScannerSchema != "agy-v1" {
			return ScannerState{}, nil, errors.New("Agy scanner state does not match joined artifacts")
		}
	}
	var diagnostics []Diagnostic
	if detail := validateAgyDatabaseEvidence(runtime, database, nativeID); detail != "" {
		state.Disputed = true
		diagnostics = append(diagnostics, artifactDiagnostic(DiagnosticSchemaNearMiss, AgentAgy, &nativeID, database, detail))
		return state, diagnostics, nil
	}
	state.FirstRecordValidated = true
	applyAgyTranscriptRecords(&state, transcript, records, &diagnostics)
	if err := ValidateScannerState(state); err != nil {
		return ScannerState{}, diagnostics, err
	}
	return state, diagnostics, nil
}

func applyAgyTranscriptRecords(state *ScannerState, transcript Artifact, records []FramedJSONLRecord, diagnostics *[]Diagnostic) {
	for _, framed := range records {
		if len(framed.Bytes) == 0 {
			continue
		}
		var record map[string]json.RawMessage
		if decodeStrictJSON(framed.Bytes, &record) != nil || record == nil {
			state.Disputed = true
			*diagnostics = append(*diagnostics, artifactDiagnostic(DiagnosticSchemaNearMiss, AgentAgy, &state.NativeID, transcript, "malformed Agy transcript JSONL record"))
		}
	}
}

func validateAgyDatabaseEvidence(runtime Runtime, database Artifact, nativeID string) string {
	header, _, err := runtime.ReadAt(database, 0, int64(len(sqliteHeader)))
	if err != nil || !bytes.Equal(header, sqliteHeader) {
		return "missing SQLite v3 header"
	}
	schema, err := runtime.QuerySQLite(database, agyTrajectorySchemaQuery, metadataRecordLimit)
	if err != nil || !validAgySchema(schema) {
		return "trajectory_meta schema is not Agy v1"
	}
	facts, err := runtime.QuerySQLite(database, agyTrajectoryFactsQuery, metadataRecordLimit)
	if err != nil || !validAgyIdentityFacts(facts, nativeID) {
		return "trajectory_meta identity row is not Agy v1"
	}
	return ""
}

func validAgyIdentityFacts(facts SQLiteResult, nativeID string) bool {
	return len(facts.Rows) == 1 && len(facts.Rows[0]) == 4 && facts.Rows[0][0] == nativeID && facts.Rows[0][1] == "text" && facts.Rows[0][2] == "integer" && facts.Rows[0][3] == "integer"
}

func validAgySchema(result SQLiteResult) bool {
	nameIndex, typeIndex := columnIndex(result.Columns, "name"), columnIndex(result.Columns, "type")
	if nameIndex < 0 || typeIndex < 0 {
		return false
	}
	required := map[string]string{"cascade_id": "TEXT", "trajectory_type": "INTEGER", "source": "INTEGER"}
	for _, row := range result.Rows {
		if nameIndex >= len(row) || typeIndex >= len(row) {
			return false
		}
		if _, ok := required[row[nameIndex]]; ok {
			if strings.ToUpper(row[typeIndex]) != required[row[nameIndex]] {
				return false
			}
			delete(required, row[nameIndex])
		}
	}
	return len(required) == 0
}

func columnIndex(columns []string, wanted string) int {
	for index, column := range columns {
		if column == wanted {
			return index
		}
	}
	return -1
}

func agyDatabasePathID(relativePath string) (string, bool) {
	parts := strings.Split(path.Clean(relativePath), "/")
	if len(parts) != 1 || !strings.HasSuffix(parts[0], ".db") {
		return "", false
	}
	nativeID := strings.TrimSuffix(parts[0], ".db")
	return nativeID, uuidPattern.MatchString(nativeID)
}

func agyTranscriptPathID(relativePath string) (string, bool) {
	parts := strings.Split(path.Clean(relativePath), "/")
	if len(parts) != 4 || parts[1] != ".system_generated" || parts[2] != "logs" || parts[3] != "transcript.jsonl" || !uuidPattern.MatchString(parts[0]) {
		return "", false
	}
	return parts[0], true
}
