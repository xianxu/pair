package launcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

const maxRestartMarkerBytes = 6*checkpoint.MaxBytes + 6*maxContinuationArgsBytes + 64*1024

func (OSRuntime) ReadCheckpoint(path string) (checkpoint.Checkpoint, error) {
	return checkpoint.ReadFile(path)
}

// RequestCouchContinuation crosses the existing internal CLI boundary without
// importing Couch into the launcher. Snapshot bytes never enter argv or output.
func (OSRuntime) RequestCouchContinuation(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// RunLaunch prepends the installed executable directory and asset bin to PATH.
	cmd := exec.CommandContext(ctx, "couch", "--internal", "request-continuation", path)
	cmd.WaitDelay = time.Second
	diagnostic := &continuationDiagnostic{}
	cmd.Stderr = diagnostic
	cmd.Stdout = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Couch request: %w: %s; inspect retained continuation status before retrying", err, strings.TrimSpace(diagnostic.text))
	}
	return nil
}

type continuationDiagnostic struct{ text string }

func (b *continuationDiagnostic) Write(p []byte) (int, error) {
	n := len(p)
	left := 4096 - len(b.text)
	if left > 0 {
		if len(p) > left {
			p = p[:left]
		}
		b.text += string(p)
	}
	return n, nil
}

func (r OSRuntime) ReadRestartMarker(session string) (RestartMarker, bool, error) {
	path, ok := restartMarkerPath(session)
	if !ok {
		return RestartMarker{}, false, errors.New("invalid restart marker path")
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return RestartMarker{}, false, nil
	}
	if err != nil {
		return RestartMarker{}, false, err
	}
	if !info.Mode().IsRegular() {
		return RestartMarker{}, false, errors.New("restart marker is not a regular file")
	}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return RestartMarker{}, false, nil
	}
	if err != nil {
		return RestartMarker{}, false, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return RestartMarker{}, false, err
	}
	if !st.Mode().IsRegular() {
		return RestartMarker{}, false, errors.New("restart marker is not a regular file")
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxRestartMarkerBytes+1))
	if err != nil {
		return RestartMarker{}, false, err
	}
	if len(raw) > maxRestartMarkerBytes {
		return RestartMarker{}, false, errors.New("restart marker exceeds size bound")
	}
	marker, err := decodeRestartMarker(string(raw))
	return marker, true, err
}

// One cache-wide lock serializes marker publication and checked acknowledgment.
// It is a single fixed-size file, retained for the lifetime of Pair's cache.
func withRestartMarkerLock(path string, fn func() error) (err error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	lock, err := (sessionledger.OSRuntime{}).Lock(filepath.Join(dir, ".restart.lock"))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, lock.Close()) }()
	return fn()
}

func (r OSRuntime) writeRestartMarker(session string, m RestartMarker) error {
	raw := serializeRestartMarker(m)
	if _, err := decodeRestartMarker(raw); err != nil {
		return err
	}
	if len(raw) > maxRestartMarkerBytes {
		return errors.New("restart marker exceeds size bound")
	}
	path, ok := restartMarkerPath(session)
	if !ok {
		return errors.New("invalid restart marker path")
	}
	return withRestartMarkerLock(path, func() error {
		if err := r.WriteAtomic(path, raw); err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		err = errors.Join(f.Sync(), f.Close())
		if err != nil {
			return err
		}
		return (sessionledger.OSRuntime{}).SyncDirectory(filepath.Dir(path))
	})
}

func (r OSRuntime) AcknowledgeRestartMarker(session string, expected RestartMarker) error {
	path, ok := restartMarkerPath(session)
	if !ok {
		return errors.New("invalid restart marker path")
	}
	return withRestartMarkerLock(path, func() error {
		current, present, err := r.ReadRestartMarker(session)
		if err != nil {
			return err
		}
		if !present || !sameRestartMarker(current, expected) {
			return nil
		} // replacement published a newer request
		if err := os.Remove(path); err != nil {
			return err
		}
		return (sessionledger.OSRuntime{}).SyncDirectory(filepath.Dir(path))
	})
}
