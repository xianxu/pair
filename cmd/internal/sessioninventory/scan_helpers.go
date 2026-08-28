package sessioninventory

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"time"
)

const (
	metadataRecordLimit = int64(1 << 20)
	jsonRecordLimit     = int64(8 << 20)
	readChunkSize       = int64(64 << 10)
)

var (
	errTruncatedRecord = errors.New("session inventory JSONL record is not newline terminated")
	uuidPattern        = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	asciiIDPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)
)

func visitJSONLines(runtime Runtime, artifact Artifact, lineLimit int64, visit func([]byte) bool) error {
	var pending []byte
	var offset int64
	for {
		chunk, eof, err := runtime.ReadAt(artifact, offset, readChunkSize)
		if err != nil {
			return err
		}
		if len(chunk) == 0 && !eof {
			return errors.New("session inventory runtime returned an empty non-final range")
		}
		offset += int64(len(chunk))
		pending = append(pending, chunk...)
		for {
			newline := bytes.IndexByte(pending, '\n')
			if newline < 0 {
				break
			}
			if int64(newline) > lineLimit {
				return ErrReadLimit
			}
			line := pending[:newline]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			pending = pending[newline+1:]
			if visit(line) {
				return nil
			}
		}
		if int64(len(pending)) > lineLimit {
			return ErrReadLimit
		}
		if eof {
			if len(pending) != 0 {
				return errTruncatedRecord
			}
			return nil
		}
	}
}

func metadataTime(value string) *NativeTime {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil
	}
	return &NativeTime{Value: parsed.UTC(), Source: TimeSourceMetadata}
}

func fallbackTime(entry FileEntry) *NativeTime {
	if entry.BirthTime != nil && !entry.BirthTime.IsZero() {
		return &NativeTime{Value: entry.BirthTime.UTC(), Source: TimeSourceBirth}
	}
	if entry.ModTime != nil && !entry.ModTime.IsZero() {
		return &NativeTime{Value: entry.ModTime.UTC(), Source: TimeSourceMTime}
	}
	return nil
}

func artifactDiagnostic(code DiagnosticCode, agent Agent, nativeID *string, artifact Artifact, detail string) Diagnostic {
	diagnostic := diagnostic(code, agent, nativeID, detail)
	diagnostic.Path = &Artifact{StorageRoot: artifact.StorageRoot, RelativePath: artifact.RelativePath, Kind: artifact.Kind}
	diagnostic.StableID = diagnosticID(diagnostic)
	return diagnostic
}

func storageDiagnostic(agent Agent, root StorageRoot, err error) Diagnostic {
	diagnostic := diagnostic(DiagnosticStorageUnreadable, agent, nil, fmt.Sprintf("%s: %v", root.Name, err))
	diagnostic.SourceRef = &root.Name
	diagnostic.StableID = diagnosticID(diagnostic)
	return diagnostic
}

func scannerFiles(runtime Runtime, agent Agent, root StorageRoot) ([]FileEntry, []Diagnostic, bool) {
	files, err := runtime.ListFiles(root)
	if err == nil {
		return files, nil, true
	}
	if errors.Is(err, ErrStorageAbsent) {
		diagnostic := diagnostic(DiagnosticStorageAbsent, agent, nil, "native storage root is absent")
		diagnostic.SourceRef = &root.Name
		diagnostic.StableID = diagnosticID(diagnostic)
		return nil, []Diagnostic{diagnostic}, false
	}
	var issues *ListingIssuesError
	if errors.As(err, &issues) {
		diagnostics := make([]Diagnostic, 0, len(issues.Artifacts))
		for _, artifact := range issues.Artifacts {
			diagnostics = append(diagnostics, artifactDiagnostic(DiagnosticNodeMalformed, agent, nil, artifact, "non-regular native storage entry rejected"))
		}
		return files, diagnostics, true
	}
	return files, []Diagnostic{storageDiagnostic(agent, root, err)}, len(files) != 0
}

func decodeStrictJSON(line []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(line))
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values in one record")
		}
		return err
	}
	return nil
}

func edgeProvenance(role Role, schema string, artifact Artifact) []EdgeProvenance {
	if role != RoleSubagent {
		return nil
	}
	return []EdgeProvenance{{Schema: schema, Artifact: artifact}}
}
