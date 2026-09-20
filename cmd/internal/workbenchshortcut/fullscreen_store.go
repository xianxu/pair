package workbenchshortcut

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
)

// FullscreenStore persists the return pane and serializes a thread's toggles.
// A busy lock returns (nil, false, nil); diagnostics are best effort and silent.
type FullscreenStore interface {
	Read() (string, error)
	Write(string) error
	Clear() error
	TryLock() (unlock func(), acquired bool, err error)
	LogFailure(error)
}

// FullscreenReturnStore uses the already-selected repository scope directory.
// The caller holds TryLock across the complete observation/action sequence.
type FullscreenReturnStore struct {
	DataDir string
	Tag     string
}

const fullscreenPaneIDBytes = 256
const fullscreenFailureBytes = 4096

func (s FullscreenReturnStore) paths() (artifactpath.Paths, error) {
	p, ok := panePaths(s.DataDir, s.Tag)
	if !ok {
		return artifactpath.ResolveScoped(s.DataDir, s.Tag)
	}
	return p, nil
}

func (s FullscreenReturnStore) Read() (string, error) {
	p, err := s.paths()
	if err != nil {
		return "", err
	}
	// readPaneID uses an unbounded os.ReadFile. Keep this reader bounded even
	// when a foreign writer replaces or grows the record during the read.
	f, err := os.OpenFile(p.FullscreenReturn(), os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return "", err
	}
	if !st.Mode().IsRegular() {
		return "", errors.New("fullscreen return record is not regular")
	}
	data, err := io.ReadAll(io.LimitReader(f, fullscreenPaneIDBytes+2))
	if err != nil {
		return "", err
	}
	if len(data) > fullscreenPaneIDBytes+1 {
		return "", errors.New("fullscreen return record too large")
	}
	id := strings.TrimSuffix(string(data), "\n")
	if err := validateFullscreenRecord(id); err != nil {
		return "", err
	}
	return id, nil
}

func validateFullscreenRecord(id string) error {
	if len(id) == 0 || len(id) > fullscreenPaneIDBytes || strings.ContainsAny(id, "\r\n\x00") || strings.TrimSpace(id) != id {
		return errors.New("invalid fullscreen return record")
	}
	return nil
}

func (s FullscreenReturnStore) Write(id string) error {
	p, err := s.paths()
	if err != nil {
		return err
	}
	if err := validateFullscreenRecord(id); err != nil {
		return err
	}
	return writePaneID(p.FullscreenReturn(), id)
}

func (s FullscreenReturnStore) Clear() error {
	p, err := s.paths()
	if err != nil {
		return err
	}
	err = os.Remove(p.FullscreenReturn())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (s FullscreenReturnStore) TryLock() (func(), bool, error) {
	p, err := s.paths()
	if err != nil {
		return nil, false, err
	}
	if err := os.MkdirAll(p.ScopeDir(), 0700); err != nil {
		return nil, false, err
	}
	f, err := os.OpenFile(p.FullscreenLock(), os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return nil, false, err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, false, err
	}
	if !st.Mode().IsRegular() {
		f.Close()
		return nil, false, errors.New("fullscreen lock is not regular")
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var once sync.Once
	// Never unlink: waiters and future invocations must use the same inode.
	return func() { once.Do(func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); _ = f.Close() }) }, true, nil
}

func (s FullscreenReturnStore) LogFailure(failure error) {
	if failure == nil {
		return
	}
	p, err := s.paths()
	if err != nil {
		return
	}
	if err := os.MkdirAll(p.ScopeDir(), 0700); err != nil {
		return
	}
	// Resolve registration from this store, never the invoking agent's ambient
	// HOME or PAIR_DATA_DIR. The managed writer owns rotation and retention.
	opts := diagnosticlog.EnvironmentOptions(func(key string) string {
		if key == "PAIR_DATA_DIR" {
			return p.ScopeDir()
		}
		return ""
	})
	w, err := diagnosticlog.Open(p.FullscreenDiagnostics(), opts)
	if err != nil {
		return
	}
	defer w.Close()
	detail := failure.Error()
	if len(detail) > fullscreenFailureBytes {
		detail = detail[:fullscreenFailureBytes]
	}
	data, err := json.Marshal(struct {
		Time  time.Time `json:"time"`
		Error string    `json:"error"`
	}{time.Now().UTC(), detail})
	if err == nil {
		_, _ = w.Write(append(data, '\n'))
	}
}
