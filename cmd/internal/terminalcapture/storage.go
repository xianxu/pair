package terminalcapture

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xianxu/pair/cmd/internal/strictjson"
	"golang.org/x/sys/unix"
)

const (
	captureRootBytes     int64 = 32 << 30
	captureRootSessions        = 64
	captureMetadataBytes       = 4096
)

type captureBudget struct {
	SchemaVersion int   `json:"schema_version"`
	MaxBytes      int64 `json:"max_bytes"`
}

// Reservations survive recorder shutdown and process crashes. Only an explicit
// removal or move of a saved session releases its space; admission never deletes
// evidence. The lock inode persists so concurrent openers always lock one file.
func openCaptureFile(parent string, limit int64) (string, *os.File, error) {
	return openCaptureFileWithBudget(parent, limit, captureRootBytes, captureRootSessions)
}

func openCaptureFileWithBudget(parent string, limit, capacity int64, sessions int) (string, *os.File, error) {
	if limit <= 0 || limit > capacity || sessions <= 0 {
		return "", nil, fmt.Errorf("capture reservation exceeds root budget")
	}
	if err := os.MkdirAll(parent, 0700); err != nil {
		return "", nil, fmt.Errorf("capture directory: %w", err)
	}
	lock, err := openCaptureRegular(filepath.Join(parent, ".capture.lock"), os.O_CREATE|os.O_RDWR)
	if err != nil {
		return "", nil, fmt.Errorf("capture admission lock: %w", err)
	}
	defer lock.Close()
	if err := unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			return "", nil, fmt.Errorf("capture admission busy; retry launch: %w", err)
		}
		return "", nil, fmt.Errorf("capture admission lock: %w", err)
	}
	defer unix.Flock(int(lock.Fd()), unix.LOCK_UN)
	entries, err := os.ReadDir(parent)
	if err != nil {
		return "", nil, err
	}
	var reserved int64
	count := 0
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "session-") {
			continue
		}
		count++
		if count >= sessions {
			return "", nil, fmt.Errorf("capture root session limit reached (%d); retain saved sessions elsewhere or choose a fresh capture directory", sessions)
		}
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return "", nil, fmt.Errorf("capture session %q is not a regular directory", entry.Name())
		}
		charge, err := captureSessionCharge(filepath.Join(parent, entry.Name()), capacity)
		if err != nil {
			return "", nil, fmt.Errorf("capture session %q: %w; retain it and choose a fresh capture directory", entry.Name(), err)
		}
		if charge > capacity-reserved {
			return "", nil, fmt.Errorf("capture root byte budget reached; retain saved sessions elsewhere or choose a fresh capture directory")
		}
		reserved += charge
	}
	if limit > capacity-reserved {
		return "", nil, fmt.Errorf("capture root byte budget reached; retain saved sessions elsewhere or choose a fresh capture directory")
	}
	dir, err := os.MkdirTemp(parent, "session-")
	if err != nil {
		return "", nil, err
	}
	// Roll back only this attempt's own files. Never remove a preexisting session
	// or recursively remove contents another participant might have added.
	createdBudget, createdEvents, success := false, false, false
	defer func() {
		if success {
			return
		}
		if createdEvents {
			_ = os.Remove(filepath.Join(dir, "events.jsonl"))
		}
		if createdBudget {
			_ = os.Remove(filepath.Join(dir, "budget.json"))
		}
		_ = os.Remove(dir)
	}()
	budgetFile, err := os.OpenFile(filepath.Join(dir, "budget.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", nil, err
	}
	createdBudget = true
	err = json.NewEncoder(budgetFile).Encode(captureBudget{SchemaVersion: 1, MaxBytes: limit})
	err = errors.Join(err, budgetFile.Close())
	if err != nil {
		return "", nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, "events.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", nil, err
	}
	createdEvents, success = true, true
	return dir, f, nil
}

// O_NOFOLLOW rejects link substitutions; O_NONBLOCK prevents a corrupt FIFO
// masquerading as metadata from hanging optional diagnostic startup.
func openCaptureRegular(path string, flags int) (*os.File, error) {
	fd, err := unix.Open(path, flags|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0600)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	info, err := f.Stat()
	if err == nil && !info.Mode().IsRegular() {
		err = fmt.Errorf("not a regular file")
	}
	if err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func captureSessionCharge(dir string, capacity int64) (int64, error) {
	events, err := openCaptureRegular(filepath.Join(dir, "events.jsonl"), os.O_RDONLY)
	if err != nil {
		return 0, fmt.Errorf("unknown capture events: %w", err)
	}
	defer events.Close()
	info, err := events.Stat()
	if err != nil {
		return 0, err
	}
	meta, err := openCaptureRegular(filepath.Join(dir, "budget.json"), os.O_RDONLY)
	if errors.Is(err, os.ErrNotExist) {
		return legacyCaptureCharge(events, info.Size())
	}
	if err != nil {
		return 0, fmt.Errorf("unknown capture reservation: %w", err)
	}
	defer meta.Close()
	data, err := io.ReadAll(io.LimitReader(meta, captureMetadataBytes+1))
	if err != nil {
		return 0, err
	}
	if len(data) > captureMetadataBytes {
		return 0, fmt.Errorf("capture reservation metadata too large")
	}
	var budget captureBudget
	if err := strictjson.Decode(data, &budget); err != nil {
		return 0, fmt.Errorf("invalid capture reservation: %w", err)
	}
	if budget.SchemaVersion != 1 || budget.MaxBytes <= 0 || budget.MaxBytes > capacity || info.Size() > budget.MaxBytes {
		return 0, fmt.Errorf("invalid capture reservation or stream exceeds reservation")
	}
	return budget.MaxBytes, nil
}

// Legacy captures have no reservation. Only a newline-terminated final end
// record proves the old recorder has stopped appending; an unknown/live legacy
// capture cannot safely share an aggregate budget with newly admitted writers.
func legacyCaptureCharge(f *os.File, size int64) (int64, error) {
	length := min(size, int64(captureMetadataBytes))
	if length == 0 {
		return 0, fmt.Errorf("unfinished legacy capture")
	}
	tail := make([]byte, int(length))
	if _, err := f.ReadAt(tail, size-length); err != nil {
		return 0, err
	}
	if tail[len(tail)-1] != '\n' {
		return 0, fmt.Errorf("unfinished legacy capture")
	}
	tail = tail[:len(tail)-1]
	i := bytes.LastIndexByte(tail, '\n')
	if i < 0 && size > length {
		return 0, fmt.Errorf("legacy final record exceeds inspection limit")
	}
	var end Record
	if err := strictjson.Decode(tail[i+1:], &end); err != nil || end.Version != 1 || end.Kind != "capture-end" || (end.Status != "complete" && end.Status != "incomplete") {
		return 0, fmt.Errorf("unfinished or unknown legacy capture")
	}
	return size, nil
}
