package sessionwatch

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/xianxu/pair/cmd/internal/adapt"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

// OSRuntime implements Runtime with real process and filesystem calls.
type OSRuntime struct {
	logger *adapt.Logger
}

func NewOSRuntime(logger *adapt.Logger) OSRuntime {
	return OSRuntime{logger: logger}
}

func (OSRuntime) Now() time.Time { return time.Now() }
func (OSRuntime) Sleep(d time.Duration) {
	time.Sleep(d)
}

func (OSRuntime) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }

func (OSRuntime) ModTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

func (OSRuntime) ProcessIdentity(pid string) string {
	return procutil.Identity(pid)
}

func (OSRuntime) NativeRuntime(home, dataDir string) sessioninventory.Runtime {
	return sessioninventory.NewOSRuntime(home, dataDir)
}

func (OSRuntime) LedgerAppender() LedgerAppender {
	return sessionledger.LedgerStore{Runtime: sessionledger.OSRuntime{}}
}

func (OSRuntime) CatalogStore() sessioninventory.CatalogStore {
	return sessioninventory.CatalogStore{Runtime: sessioninventory.CatalogOSRuntime{}}
}

func (OSRuntime) AtomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

func (r OSRuntime) Log(outcome adapt.Outcome, detail string) {
	r.logger.Log(3, "session-id", outcome, detail)
}

func ParseDurationSeconds(s string, fallback time.Duration) time.Duration {
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return fallback
	}
	return time.Duration(n) * time.Second
}
