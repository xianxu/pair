package sessioninventory

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

var (
	ErrPathEscape  = errors.New("session inventory path escapes its storage root")
	ErrRootUnknown = errors.New("session inventory storage root is unknown")
)

type OSRuntime struct {
	roots    map[Agent][]StorageRoot
	pairRoot StorageRoot
}

var _ Runtime = OSRuntime{}

func NewOSRuntime(homeDir, pairDataDir string) OSRuntime {
	return OSRuntime{
		roots: map[Agent][]StorageRoot{
			AgentClaude: {{Agent: AgentClaude, Name: "claude-projects", Path: filepath.Join(homeDir, ".claude", "projects")}},
			AgentCodex:  {{Agent: AgentCodex, Name: "codex-sessions", Path: filepath.Join(homeDir, ".codex", "sessions")}},
			AgentAgy: {
				{Agent: AgentAgy, Name: "agy-conversations", Path: filepath.Join(homeDir, ".gemini", "antigravity-cli", "conversations")},
				{Agent: AgentAgy, Name: "agy-brain", Path: filepath.Join(homeDir, ".gemini", "antigravity-cli", "brain")},
			},
			AgentMuse: {{Agent: AgentMuse, Name: "muse-sessions", Path: filepath.Join(homeDir, ".local", "share", "muse", "sessions")}},
		},
		pairRoot: StorageRoot{Name: "pair-data", Path: pairDataDir},
	}
}

func DefaultOSRuntime(pairDataDir string) (OSRuntime, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return OSRuntime{}, err
	}
	return NewOSRuntime(homeDir, pairDataDir), nil
}

func (r OSRuntime) NativeRoots(agent Agent) []StorageRoot {
	return append([]StorageRoot(nil), r.roots[agent]...)
}

func (r OSRuntime) PairDataRoot() StorageRoot { return r.pairRoot }

func (r OSRuntime) ListFiles(requested StorageRoot) ([]FileEntry, error) {
	root, ok := r.authorizedRoot(requested.Name)
	if !ok {
		return nil, ErrRootUnknown
	}
	var result []FileEntry
	err := filepath.WalkDir(root.Path, func(filePath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if filePath == root.Path || entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: %s", ErrPathEscape, entry.Name())
		}
		relativePath, err := filepath.Rel(root.Path, filePath)
		if err != nil {
			return err
		}
		artifact := Artifact{StorageRoot: root.Name, RelativePath: filepath.ToSlash(relativePath)}
		if !validArtifact(artifact) {
			return fmt.Errorf("%w: %s", ErrPathEscape, relativePath)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		modTime := info.ModTime().UTC()
		result = append(result, FileEntry{
			Artifact:  artifact,
			Size:      info.Size(),
			BirthTime: fileBirthTime(filePath),
			ModTime:   &modTime,
		})
		return nil
	})
	return result, err
}

func (r OSRuntime) ReadFile(artifact Artifact, limit int64) ([]byte, error) {
	filePath, err := r.resolveArtifact(artifact)
	if err != nil {
		return nil, err
	}
	if limit < 0 {
		return nil, ErrReadLimit
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > limit {
		return nil, ErrReadLimit
	}
	return content, nil
}

func (r OSRuntime) QuerySQLite(artifact Artifact, query string, limit int64) (SQLiteResult, error) {
	filePath, err := r.resolveArtifact(artifact)
	if err != nil {
		return SQLiteResult{}, err
	}
	if limit < 0 {
		return SQLiteResult{}, ErrReadLimit
	}
	stdout := newBoundedBuffer(limit)
	stderr := newBoundedBuffer(8192)
	command := exec.Command("sqlite3", "-readonly", "-header", "-csv", filePath, query)
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		if stdout.exceeded || stderr.exceeded {
			return SQLiteResult{}, ErrReadLimit
		}
		return SQLiteResult{}, fmt.Errorf("sqlite3: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	if stdout.exceeded {
		return SQLiteResult{}, ErrReadLimit
	}
	records, err := csv.NewReader(bytes.NewReader(stdout.Bytes())).ReadAll()
	if err != nil {
		return SQLiteResult{}, fmt.Errorf("parse sqlite3 csv: %w", err)
	}
	if len(records) == 0 {
		return SQLiteResult{}, nil
	}
	result := SQLiteResult{Columns: append([]string(nil), records[0]...), Rows: make([][]string, len(records)-1)}
	for i, row := range records[1:] {
		result.Rows[i] = append([]string(nil), row...)
	}
	return result, nil
}

func (OSRuntime) ProcessChildren() map[string][]string {
	children := procutil.ProcessChildren()
	for parent := range children {
		sort.Strings(children[parent])
	}
	return children
}

func (OSRuntime) ProcessIdentity(pid string) string { return procutil.Identity(pid) }

func (OSRuntime) OpenFiles(pid string) []string {
	files := procutil.LsofNames(pid)
	sort.Strings(files)
	return files
}

func (r OSRuntime) authorizedRoot(name string) (StorageRoot, bool) {
	if r.pairRoot.Name == name {
		return r.pairRoot, true
	}
	for _, roots := range r.roots {
		for _, root := range roots {
			if root.Name == name {
				return root, true
			}
		}
	}
	return StorageRoot{}, false
}

func (r OSRuntime) resolveArtifact(artifact Artifact) (string, error) {
	if !validArtifact(artifact) {
		return "", ErrPathEscape
	}
	root, ok := r.authorizedRoot(artifact.StorageRoot)
	if !ok {
		return "", ErrRootUnknown
	}
	if err := rejectRelativeSymlinks(root.Path, filepath.FromSlash(artifact.RelativePath)); err != nil {
		return "", err
	}
	rootPath, err := filepath.EvalSymlinks(root.Path)
	if err != nil {
		return "", err
	}
	filePath, err := filepath.EvalSymlinks(filepath.Join(root.Path, filepath.FromSlash(artifact.RelativePath)))
	if err != nil {
		return "", err
	}
	relativePath, err := filepath.Rel(rootPath, filePath)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", ErrPathEscape
	}
	return filePath, nil
}

func rejectRelativeSymlinks(rootPath, relativePath string) error {
	current := rootPath
	for _, component := range strings.Split(relativePath, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: %s", ErrPathEscape, relativePath)
		}
	}
	return nil
}

func fileBirthTime(filePath string) *time.Time {
	var arguments []string
	switch runtime.GOOS {
	case "darwin":
		arguments = []string{"-f", "%B", filePath}
	case "linux":
		arguments = []string{"-c", "%W", filePath}
	default:
		return nil
	}
	output, err := exec.Command("stat", arguments...).Output()
	if err != nil {
		return nil
	}
	seconds, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil || seconds <= 0 {
		return nil
	}
	value := time.Unix(seconds, 0).UTC()
	return &value
}

type boundedBuffer struct {
	buffer   bytes.Buffer
	limit    int64
	exceeded bool
}

func newBoundedBuffer(limit int64) *boundedBuffer { return &boundedBuffer{limit: limit} }

func (b *boundedBuffer) Write(content []byte) (int, error) {
	originalLength := len(content)
	remaining := b.limit - int64(b.buffer.Len())
	if remaining < int64(len(content)) {
		b.exceeded = true
		if remaining <= 0 {
			return originalLength, nil
		}
		content = content[:remaining]
	}
	_, _ = b.buffer.Write(content)
	return originalLength, nil
}

func (b *boundedBuffer) Bytes() []byte { return b.buffer.Bytes() }

func (b *boundedBuffer) String() string { return b.buffer.String() }
