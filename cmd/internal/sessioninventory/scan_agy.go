package sessioninventory

import (
	"bytes"
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
	header, _, err := runtime.ReadAt(database, 0, int64(len(sqliteHeader)))
	if err != nil || !bytes.Equal(header, sqliteHeader) {
		return Fact{}, []Diagnostic{artifactDiagnostic(DiagnosticSchemaNearMiss, AgentAgy, &nativeID, database, "missing SQLite v3 header")}, false
	}
	schema, err := runtime.QuerySQLite(database, agyTrajectorySchemaQuery, metadataRecordLimit)
	if err != nil || !validAgySchema(schema) {
		return Fact{}, []Diagnostic{artifactDiagnostic(DiagnosticSchemaNearMiss, AgentAgy, &nativeID, database, "trajectory_meta schema is not Agy v1")}, false
	}
	facts, err := runtime.QuerySQLite(database, agyTrajectoryFactsQuery, metadataRecordLimit)
	if err != nil || len(facts.Rows) != 1 || len(facts.Rows[0]) != 4 || facts.Rows[0][0] != nativeID || facts.Rows[0][1] != "text" || facts.Rows[0][2] != "integer" || facts.Rows[0][3] != "integer" {
		return Fact{}, []Diagnostic{artifactDiagnostic(DiagnosticSchemaNearMiss, AgentAgy, &nativeID, database, "trajectory_meta identity row is not Agy v1")}, false
	}

	artifacts := []Artifact{database}
	var diagnostics []Diagnostic
	if transcriptEntry.Artifact.RelativePath == "" {
		diagnostics = append(diagnostics, artifactDiagnostic(DiagnosticParentMissing, AgentAgy, &nativeID, database, "Agy transcript join is missing"))
	} else {
		transcript := transcriptEntry.Artifact
		transcript.Kind = ArtifactTranscript
		artifacts = append(artifacts, transcript)
	}
	return Fact{
		Agent:     AgentAgy,
		NativeID:  nativeID,
		Role:      RoleRoot,
		Time:      fallbackTime(databaseEntry),
		Resumable: true,
		Artifacts: artifacts,
	}, diagnostics, true
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
