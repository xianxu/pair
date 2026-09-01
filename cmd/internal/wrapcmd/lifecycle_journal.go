package wrapcmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessionwatch"
)

const lifecycleJournalMaxRecord = 64 << 10

// One sentinel byte beyond the record limit lets an advance reject an
// over-limit unterminated record without a second filesystem pass.
const lifecycleJournalReadChunk = lifecycleJournalMaxRecord + 1

// LifecycleJournalTailer incrementally reads committed records appended after
// it opens. It permanently stops when the journal is replaced, truncated, or
// malformed so stale transport state can never be mistaken for authority.
type LifecycleJournalTailer struct {
	path          string
	launchOrdinal uint64
	position      int64
	pending       []byte
	identity      os.FileInfo
	stopped       error
}

// OpenLifecycleJournalTailer snapshots the prior EOF. A not-yet-created file
// is valid: its first creation establishes the identity.
func OpenLifecycleJournalTailer(path string, launchOrdinal uint64) (*LifecycleJournalTailer, error) {
	if path == "" || launchOrdinal == 0 {
		return nil, errors.New("lifecycle journal path or launch ordinal is empty")
	}
	t := &LifecycleJournalTailer{path: path, launchOrdinal: launchOrdinal}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return t, nil
	}
	if err != nil {
		return nil, fmt.Errorf("stat lifecycle journal: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("lifecycle journal is not a regular file")
	}
	t.identity = info
	t.position = info.Size()
	return t, nil
}

func (t *LifecycleJournalTailer) Advance() ([]sessionwatch.LifecycleRecord, error) {
	if t.stopped != nil {
		return nil, t.stopped
	}
	info, err := os.Stat(t.path)
	if errors.Is(err, os.ErrNotExist) && t.identity == nil {
		return nil, nil
	}
	if err != nil {
		return nil, t.stop(fmt.Errorf("stat lifecycle journal: %w", err))
	}
	if t.identity == nil {
		t.identity = info
	} else if !os.SameFile(t.identity, info) {
		return nil, t.stop(errors.New("lifecycle journal was replaced"))
	}
	if info.Size() < t.position {
		return nil, t.stop(errors.New("lifecycle journal was truncated"))
	}
	if info.Size() == t.position {
		return nil, nil
	}
	f, err := os.Open(t.path)
	if err != nil {
		return nil, t.stop(fmt.Errorf("open lifecycle journal: %w", err))
	}
	defer f.Close()
	openedInfo, err := f.Stat()
	if err != nil {
		return nil, t.stop(fmt.Errorf("stat open lifecycle journal: %w", err))
	}
	if !os.SameFile(t.identity, openedInfo) {
		return nil, t.stop(errors.New("lifecycle journal was replaced while opening"))
	}
	if _, err := f.Seek(t.position, io.SeekStart); err != nil {
		return nil, t.stop(fmt.Errorf("seek lifecycle journal: %w", err))
	}
	remaining := info.Size() - t.position
	if remaining > lifecycleJournalReadChunk {
		remaining = lifecycleJournalReadChunk
	}
	newRaw, err := io.ReadAll(io.LimitReader(f, remaining))
	if err != nil {
		return nil, t.stop(fmt.Errorf("read lifecycle journal: %w", err))
	}
	t.position += int64(len(newRaw))
	t.pending = append(t.pending, newRaw...)
	if len(t.pending) > lifecycleJournalMaxRecord && !bytes.ContainsRune(t.pending, '\n') {
		return nil, t.stop(errors.New("lifecycle journal record exceeds 64 KiB"))
	}

	var records []sessionwatch.LifecycleRecord
	for {
		newline := bytes.IndexByte(t.pending, '\n')
		if newline < 0 {
			break
		}
		line := append([]byte(nil), t.pending[:newline]...)
		t.pending = t.pending[newline+1:]
		if len(line)+1 > lifecycleJournalMaxRecord {
			return nil, t.stop(errors.New("lifecycle journal record exceeds 64 KiB"))
		}
		var record sessionwatch.LifecycleRecord
		decoder := json.NewDecoder(bufio.NewReader(bytes.NewReader(line)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&record); err != nil {
			return nil, t.stop(fmt.Errorf("decode lifecycle journal: %w", err))
		}
		if record.Version != 1 || record.Agent == "" || record.LaunchOrdinal == 0 ||
			record.ArtifactGeneration == "" || record.Source == "" || record.Outcome == "" ||
			record.TranscriptPath == "" || record.TranscriptOffset < 0 || record.EventTimestamp.IsZero() {
			return nil, t.stop(errors.New("invalid lifecycle journal record"))
		}
		if record.LaunchOrdinal == t.launchOrdinal {
			records = append(records, record)
		}
	}
	if len(t.pending) > lifecycleJournalMaxRecord {
		return nil, t.stop(errors.New("lifecycle journal record exceeds 64 KiB"))
	}
	return records, nil
}

type lifecycleJournalAdvancer interface {
	Advance() ([]sessionwatch.LifecycleRecord, error)
}

func followLifecycleJournal(tailer lifecycleJournalAdvancer, records chan<- sessionwatch.LifecycleRecord, failures chan<- error, stop <-chan struct{}) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			observed, err := tailer.Advance()
			if err != nil {
				select {
				case failures <- err:
				case <-stop:
				}
				return
			}
			for _, record := range observed {
				select {
				case records <- record:
				case <-stop:
					return
				}
			}
		}
	}
}

func (t *LifecycleJournalTailer) stop(err error) error {
	t.stopped = err
	return err
}
