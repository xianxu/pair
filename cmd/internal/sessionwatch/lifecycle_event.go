package sessionwatch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/xianxu/pair/cmd/internal/commitoutcome"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

const lifecycleRecordMaxBytes = 64 << 10

// LifecycleRecord is one launch-authorized transcript observation. Journal
// byte offsets are deliberately absent: transport position is not identity.
type LifecycleRecord struct {
	Version            int       `json:"v"`
	Agent              string    `json:"agent"`
	LaunchOrdinal      uint64    `json:"launch_ordinal"`
	ArtifactGeneration string    `json:"artifact_generation"`
	Source             string    `json:"source"`
	Outcome            string    `json:"outcome"`
	TurnID             string    `json:"turn_id,omitempty"`
	Message            string    `json:"message,omitempty"`
	TranscriptPath     string    `json:"transcript_path"`
	TranscriptOffset   int64     `json:"transcript_record_offset"`
	EventTimestamp     time.Time `json:"event_timestamp"`
}

// ValidateLifecycleRecord defines the complete journal grammar shared by its
// sole producer and consumer. Only authorized Codex transcript observations
// may become notification authority.
func ValidateLifecycleRecord(r LifecycleRecord) error {
	if r.Version != 1 || r.Agent != "codex" || r.LaunchOrdinal == 0 || r.ArtifactGeneration == "" ||
		r.Source != "transcript" || r.TurnID == "" || r.TranscriptPath == "" ||
		r.TranscriptOffset < 0 || r.EventTimestamp.IsZero() {
		return errors.New("invalid lifecycle record")
	}
	switch r.Outcome {
	case "started", "completed", "aborted", "error":
	default:
		return errors.New("invalid lifecycle record outcome")
	}
	return nil
}

// DecodeLifecycleRecord accepts exactly one complete, validated JSON record.
func DecodeLifecycleRecord(line []byte) (LifecycleRecord, error) {
	var record LifecycleRecord
	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return LifecycleRecord{}, fmt.Errorf("decode lifecycle record: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return LifecycleRecord{}, errors.New("lifecycle record has trailing data")
	}
	if err := ValidateLifecycleRecord(record); err != nil {
		return LifecycleRecord{}, err
	}
	return record, nil
}

func (r LifecycleRecord) sameObservation(other LifecycleRecord) bool {
	return r.Version == other.Version && r.Agent == other.Agent &&
		r.LaunchOrdinal == other.LaunchOrdinal && r.ArtifactGeneration == other.ArtifactGeneration &&
		r.Source == other.Source && r.Outcome == other.Outcome && r.TurnID == other.TurnID &&
		r.Message == other.Message && r.TranscriptPath == other.TranscriptPath &&
		r.TranscriptOffset == other.TranscriptOffset && r.EventTimestamp.Equal(other.EventTimestamp)
}

// AppendLifecycleRecord appends one newline-committed record under the shared
// ledger-style lock. If durability becomes indeterminate, it reconciles by the
// record's stable launch/artifact/offset identity before considering a retry.
func AppendLifecycleRecord(runtime sessionledger.Runtime, path string, record LifecycleRecord) (err error) {
	if runtime == nil || path == "" {
		return errors.New("lifecycle journal path or runtime is empty")
	}
	if err := ValidateLifecycleRecord(record); err != nil {
		return err
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode lifecycle record: %w", err)
	}
	if len(encoded)+1 > lifecycleRecordMaxBytes {
		return errors.New("lifecycle record exceeds 64 KiB")
	}
	dir := filepath.Dir(path)
	if err := runtime.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create lifecycle journal directory: %w", err)
	}
	lock, err := runtime.Lock(path + ".lock")
	if err != nil {
		return fmt.Errorf("lock lifecycle journal: %w", err)
	}
	outcome := commitoutcome.NotAuthoritative
	defer func() {
		if unlockErr := lock.Close(); unlockErr != nil {
			err = commitoutcome.Join(outcome, err, fmt.Errorf("unlock lifecycle journal: %w", unlockErr))
		}
	}()

	raw, readErr := runtime.ReadFile(path)
	if errors.Is(readErr, os.ErrNotExist) {
		raw, readErr = nil, nil
	}
	if readErr != nil {
		return fmt.Errorf("read lifecycle journal: %w", readErr)
	}
	if lifecycleIdentityPresent(raw, record) {
		return reconcileLifecycleRecord(runtime, path, dir, record)
	}
	var payload []byte
	if len(raw) > 0 && raw[len(raw)-1] != '\n' {
		payload = append(payload, '\n')
	}
	payload = append(payload, encoded...)
	payload = append(payload, '\n')
	file, err := runtime.OpenAppend(path, 0o600)
	if err != nil {
		return fmt.Errorf("open lifecycle journal: %w", err)
	}
	written, writeErr := lifecycleWriteAll(file, payload)
	if writeErr != nil {
		closeErr := file.Close()
		if written != len(payload) {
			return errors.Join(fmt.Errorf("append lifecycle journal: %w", writeErr), closeErr)
		}
		outcome = commitoutcome.Indeterminate
		if reconcileErr := reconcileLifecycleRecord(runtime, path, dir, record); reconcileErr == nil {
			outcome = commitoutcome.Committed
			return nil
		} else {
			return commitoutcome.Wrap(outcome, errors.Join(writeErr, closeErr, reconcileErr))
		}
	}
	outcome = commitoutcome.Indeterminate
	if syncErr := file.Sync(); syncErr != nil {
		closeErr := file.Close()
		if reconcileErr := reconcileLifecycleRecord(runtime, path, dir, record); reconcileErr == nil {
			outcome = commitoutcome.Committed
			return nil
		} else {
			return commitoutcome.Wrap(outcome, errors.Join(syncErr, closeErr, reconcileErr))
		}
	}
	if closeErr := file.Close(); closeErr != nil {
		outcome = commitoutcome.Committed
		return commitoutcome.Wrap(outcome, fmt.Errorf("close lifecycle journal: %w", closeErr))
	}
	if syncErr := runtime.SyncDirectory(dir); syncErr != nil {
		if reconcileErr := reconcileLifecycleRecord(runtime, path, dir, record); reconcileErr == nil {
			outcome = commitoutcome.Committed
			return nil
		} else {
			return commitoutcome.Wrap(outcome, errors.Join(syncErr, reconcileErr))
		}
	}
	outcome = commitoutcome.Committed
	return nil
}

func reconcileLifecycleRecord(runtime sessionledger.Runtime, path, dir string, record LifecycleRecord) error {
	raw, err := runtime.ReadFile(path)
	if err != nil || !lifecycleIdentityPresent(raw, record) {
		return errors.Join(errors.New("attempted lifecycle identity is not committed"), err)
	}
	file, err := runtime.OpenAppend(path, 0o600)
	if err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return errors.Join(err, file.Close())
	}
	if err := file.Close(); err != nil {
		return err
	}
	return runtime.SyncDirectory(dir)
}

func lifecycleIdentityPresent(raw []byte, want LifecycleRecord) bool {
	if len(raw) == 0 || raw[len(raw)-1] != '\n' {
		return false
	}
	for _, line := range bytes.Split(raw[:len(raw)-1], []byte{'\n'}) {
		got, err := DecodeLifecycleRecord(line)
		if err == nil && got.sameObservation(want) {
			return true
		}
	}
	return false
}

func lifecycleWriteAll(w io.Writer, raw []byte) (int, error) {
	written := 0
	for len(raw) > 0 {
		n, err := w.Write(raw)
		written += n
		if err != nil {
			return written, err
		}
		if n <= 0 {
			return written, io.ErrShortWrite
		}
		raw = raw[n:]
	}
	return written, nil
}
