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
	readChunkSize       = int64(64 << 10)
	unlimitedRecordSize = int64(-1)
)

func recordExceedsLimit(size int, limit int64) bool {
	return limit != unlimitedRecordSize && int64(size) > limit
}

var (
	errTruncatedRecord = errors.New("session inventory JSONL record is not newline terminated")
	uuidPattern        = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	asciiIDPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)
)

func visitJSONLines(runtime Runtime, artifact Artifact, lineLimit int64, visit func([]byte) bool) error {
	return visitJSONLinesAt(runtime, artifact, lineLimit, func(line []byte, _ uint64) bool {
		return visit(line)
	})
}

func visitJSONLinesAt(runtime Runtime, artifact Artifact, lineLimit int64, visit func([]byte, uint64) bool) error {
	tail, stopped, err := frameJSONLArtifact(runtime, artifact, lineLimit, func(line []byte, start uint64) bool {
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		return visit(line, start)
	})
	if err != nil {
		return err
	}
	if !stopped && len(tail) != 0 {
		return errTruncatedRecord
	}
	return nil
}

// frameJSONLArtifact is the one chunked reader under every consumer of an
// append-only JSONL artifact. It reads in readChunkSize ranges, calls line for
// each newline-terminated record (CR kept — a consumer that wants it gone
// strips it) with the record's byte offset. A nonnegative recordLimit caps
// each record; unlimitedRecordSize disables that cutoff. The file itself is
// unbounded — its length is defined to grow. What an unterminated tail MEANS
// is the consumer's call, so it comes back rather than being judged here:
// the transcript framer treats it as a truncated record, the ledger reader
// hands it to ParseLedger, which records a malformed ordinal (#237).
// stopped reports that line returned true before the end.
func frameJSONLArtifact(runtime Runtime, artifact Artifact, recordLimit int64, line func([]byte, uint64) bool) (tail []byte, stopped bool, err error) {
	var pending []byte
	var readOffset int64
	var pendingOffset uint64
	for {
		chunk, eof, err := runtime.ReadAt(artifact, readOffset, readChunkSize)
		if err != nil {
			return nil, false, err
		}
		if len(chunk) == 0 && !eof {
			return nil, false, errors.New("session inventory runtime returned an empty non-final range")
		}
		readOffset += int64(len(chunk))
		pending = append(pending, chunk...)
		for {
			newline := bytes.IndexByte(pending, '\n')
			if newline < 0 {
				break
			}
			if recordExceedsLimit(newline, recordLimit) {
				return nil, false, ErrReadLimit
			}
			record := pending[:newline]
			pending = pending[newline+1:]
			if line(record, pendingOffset) {
				return nil, true, nil
			}
			pendingOffset += uint64(newline + 1)
		}
		if recordExceedsLimit(len(pending), recordLimit) {
			return nil, false, ErrReadLimit
		}
		if eof {
			return pending, false, nil
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

// readJSONLArtifact returns the whole body of an append-only JSONL artifact,
// with an optional per-record bound enforced by frameJSONLArtifact. Inventory
// consumers disable that bound because writers have no matching ceiling. The
// owner ledger was read with a whole-file cap instead, so a thread that had
// been relaunched often enough became unresumable the moment its ledger
// crossed 8 MiB (#237). The body comes back byte for byte, partial last line
// included: ParseLedger owns the unterminated-tail rule (a malformed ordinal,
// not an error), and a ledger mid-append must stay readable.
func readJSONLArtifact(runtime Runtime, artifact Artifact, recordLimit int64) ([]byte, error) {
	var body []byte
	tail, _, err := frameJSONLArtifact(runtime, artifact, recordLimit, func(record []byte, _ uint64) bool {
		body = append(append(body, record...), '\n')
		return false
	})
	if err != nil {
		return nil, err
	}
	return append(body, tail...), nil
}
