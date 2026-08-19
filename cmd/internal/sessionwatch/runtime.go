package sessionwatch

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/adapt"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/transcript"
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

func (OSRuntime) ReadFirstLine(path string) ([]byte, error) {
	return transcript.ReadFirstEvent(path)
}

func (OSRuntime) ModTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

func (OSRuntime) BirthTime(path string) (time.Time, error) {
	out, err := exec.Command("stat", "-f", "%B", path).Output()
	if err != nil {
		return time.Time{}, err
	}
	sec, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(sec, 0), nil
}

func (OSRuntime) ListFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}

func (OSRuntime) Descendants(root string) ([]string, error) {
	return procutil.DescendantPIDs(root, procutil.ProcessChildren()), nil
}

func (OSRuntime) LsofPaths(pid string) ([]string, error) {
	return procutil.LsofNames(pid), nil
}

func (OSRuntime) ProcessAlive(pid string) bool {
	return procutil.Alive(pid)
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
