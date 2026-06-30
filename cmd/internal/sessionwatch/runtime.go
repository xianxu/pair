package sessionwatch

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/adapt"
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
	out, err := exec.Command("ps", "-axo", "pid=,ppid=").Output()
	if err != nil {
		return []string{root}, nil
	}
	children := map[string][]string{}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		children[fields[1]] = append(children[fields[1]], fields[0])
	}
	queue := []string{root}
	seen := map[string]bool{root: true}
	for i := 0; i < len(queue); i++ {
		for _, child := range children[queue[i]] {
			if child == "" || seen[child] {
				continue
			}
			seen[child] = true
			queue = append(queue, child)
		}
	}
	return queue, nil
}

func (OSRuntime) LsofPaths(pid string) ([]string, error) {
	out, err := exec.Command("lsof", "-p", pid, "-Fn").Output()
	if err != nil {
		return nil, nil
	}
	var paths []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "n") {
			paths = append(paths, strings.TrimPrefix(line, "n"))
		}
	}
	return paths, nil
}

func (OSRuntime) ProcessAlive(pid string) bool {
	return exec.Command("kill", "-0", pid).Run() == nil
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
