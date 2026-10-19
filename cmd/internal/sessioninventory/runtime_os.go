package sessioninventory

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

var (
	ErrPathEscape    = errors.New("session inventory path escapes its storage root")
	ErrRootUnknown   = errors.New("session inventory storage root is unknown")
	ErrStorageAbsent = errors.New("session inventory storage root is absent")
	// ErrNotRegularArtifact is a named artifact that is not a regular file; a
	// listing rejects the same entry as a ListingIssuesError.
	ErrNotRegularArtifact = errors.New("session inventory artifact is not a regular file")
)

// pair:155-concept integration new M1
type OSRuntime struct {
	roots    map[Agent][]StorageRoot
	pairRoot StorageRoot
	launches *OwnerLaunchCache
	work     QueryWorkObserver
}

// WithOwnerLaunchCache is this runtime keeping owners' current launches in
// cache (#507), for a long-lived caller that queries the same owners again.
func (r OSRuntime) WithOwnerLaunchCache(cache *OwnerLaunchCache) OSRuntime {
	r.launches = cache
	return r
}

func (r OSRuntime) OwnerLaunchCache() *OwnerLaunchCache { return r.launches }

func (r OSRuntime) WithWorkObserver(observer QueryWorkObserver) OSRuntime {
	r.work = observer
	return r
}
func (r OSRuntime) ReadWork(artifact Artifact, bytes int64) {
	if r.work != nil {
		r.work.ReadWork(artifact, bytes)
	}
}
func (r OSRuntime) DecodeWork(artifact Artifact, bytes int64) {
	if r.work != nil {
		r.work.DecodeWork(artifact, bytes)
	}
}

var _ Runtime = OSRuntime{}

// nativeRootLayout is each agent's native transcript roots: a stable name and
// the location under the home directory. The names are what persisted proofs
// and stored transcript bindings carry; locations come only from here.
var nativeRootLayout = map[Agent][]struct {
	name string
	home []string
}{
	AgentClaude: {{"claude-projects", []string{".claude", "projects"}}},
	AgentCodex:  {{"codex-sessions", []string{".codex", "sessions"}}},
	AgentAgy: {
		{"agy-conversations", []string{".gemini", "antigravity-cli", "conversations"}},
		{"agy-brain", []string{".gemini", "antigravity-cli", "brain"}},
	},
	AgentMuse:  {{"muse-sessions", []string{".local", "share", "muse", "sessions"}}},
	AgentQoder: {{"qoder-projects", []string{".qoder", "projects"}}},
	AgentGrok:  {{grokSessionsRoot, []string{".grok", "sessions"}}},
}

// NativeRootNames is the agent's native root names, without touching the
// filesystem.
func NativeRootNames(agent Agent) []string {
	var names []string
	for _, root := range nativeRootLayout[agent] {
		names = append(names, root.name)
	}
	return names
}

func NewOSRuntime(homeDir, pairDataDir string) OSRuntime {
	roots := map[Agent][]StorageRoot{}
	for agent, layout := range nativeRootLayout {
		for _, root := range layout {
			roots[agent] = append(roots[agent], StorageRoot{Agent: agent, Name: root.name, Path: filepath.Join(append([]string{homeDir}, root.home...)...)})
		}
	}
	return OSRuntime{roots: roots, pairRoot: StorageRoot{Name: "pair-data", Path: pairDataDir}}
}

func DefaultOSRuntime(pairDataDir string) (OSRuntime, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return OSRuntime{}, err
	}
	return NewOSRuntime(homeDir, pairDataDir), nil
}

// WithCodexHome scopes only the canonical transcript namespace for an explicit ID.
func (r OSRuntime) WithCodexHome(home string) (OSRuntime, error) {
	if !filepath.IsAbs(home) || strings.ContainsRune(home, 0) {
		return OSRuntime{}, errors.New("Codex home requires an absolute path")
	}
	roots := make(map[Agent][]StorageRoot, len(r.roots))
	for a, v := range r.roots {
		roots[a] = append([]StorageRoot(nil), v...)
	}
	roots[AgentCodex] = []StorageRoot{{Agent: AgentCodex, Name: "codex-sessions", Path: filepath.Join(home, "sessions")}}
	r.roots = roots
	return r, nil
}

func (r OSRuntime) NativeRoots(agent Agent) []StorageRoot {
	return append([]StorageRoot(nil), r.roots[agent]...)
}

func (r OSRuntime) PairDataRoot() StorageRoot { return r.pairRoot }

func (r OSRuntime) LoadSessionInventoryCatalog() (Catalog, error) {
	paths, err := artifactpath.ResolveSelectedScope(r.pairRoot.Path)
	if err != nil {
		return Catalog{}, err
	}
	return (CatalogStore{Runtime: observedCatalogRuntime{owner: r, artifact: Artifact{StorageRoot: r.pairRoot.Name, RelativePath: filepath.Base(paths.SessionInventoryCatalog())}}}).Read(paths.SessionInventoryCatalog())
}

func (r OSRuntime) PublishSessionInventoryValidations(validations []TargetValidation) error {
	paths, err := artifactpath.ResolveSelectedScope(r.pairRoot.Path)
	if err != nil {
		return err
	}
	return PublishTargetValidations(CatalogStore{Runtime: observedCatalogRuntime{owner: r, artifact: Artifact{StorageRoot: r.pairRoot.Name, RelativePath: filepath.Base(paths.SessionInventoryCatalog())}}}, paths.SessionInventoryCatalog(), validations)
}

func (r OSRuntime) ListFiles(requested StorageRoot) ([]FileEntry, error) {
	root, ok := r.authorizedRoot(requested.Name)
	if !ok {
		return nil, ErrRootUnknown
	}
	var result []FileEntry
	var rejected []Artifact
	err := filepath.WalkDir(root.Path, func(filePath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if filePath == root.Path || entry.IsDir() {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		if entry.Type()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			relativePath, relErr := filepath.Rel(root.Path, filePath)
			if relErr != nil {
				return relErr
			}
			rejected = append(rejected, Artifact{StorageRoot: root.Name, RelativePath: filepath.ToSlash(relativePath)})
			return nil
		}
		relativePath, err := filepath.Rel(root.Path, filePath)
		if err != nil {
			return err
		}
		artifact := Artifact{StorageRoot: root.Name, RelativePath: filepath.ToSlash(relativePath)}
		if !validArtifact(artifact) {
			return fmt.Errorf("%w: %s", ErrPathEscape, relativePath)
		}
		fileEntry, entryErr := fileEntryFor(artifact, filePath, info)
		if entryErr != nil {
			return entryErr
		}
		result = append(result, fileEntry)
		return nil
	})
	if os.IsNotExist(err) {
		return nil, ErrStorageAbsent
	}
	if err != nil {
		return result, err
	}
	if len(rejected) != 0 {
		return result, &ListingIssuesError{Artifacts: rejected}
	}
	return result, nil
}

// StatFile describes one named artifact exactly as ListFiles would, without
// walking its storage root (pair#420): reads of a bound session need only the
// artifacts its proof names.
func (r OSRuntime) StatFile(artifact Artifact) (FileEntry, error) {
	filePath, err := r.resolveArtifact(artifact)
	if err != nil {
		return FileEntry{}, err
	}
	info, err := os.Lstat(filePath)
	if err != nil {
		return FileEntry{}, err
	}
	if !info.Mode().IsRegular() {
		return FileEntry{}, fmt.Errorf("%w: %s", ErrNotRegularArtifact, artifact.RelativePath)
	}
	return fileEntryFor(artifact, filePath, info)
}

func fileEntryFor(artifact Artifact, filePath string, info os.FileInfo) (FileEntry, error) {
	modTime := info.ModTime().UTC()
	metadata, err := readFileMetadata(filePath, info)
	if err != nil {
		return FileEntry{}, err
	}
	return FileEntry{
		Artifact:        artifact,
		StableFileID:    metadata.StableFileID,
		GenerationToken: metadata.GenerationToken,
		MutationToken:   metadata.MutationToken,
		Size:            info.Size(),
		BirthTime:       metadata.BirthTime,
		ModTime:         &modTime,
	}, nil
}

// ReadFile accepts -1 for unlimited artifact bytes; nonnegative limits remain
// enforced. Namespace and regular-file validation apply in both modes.
func (r OSRuntime) ReadFile(artifact Artifact, limit int64) ([]byte, error) {
	filePath, err := r.resolveArtifact(artifact)
	if err != nil {
		return nil, err
	}
	if limit < unlimitedRecordSize {
		return nil, ErrReadLimit
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if limit == unlimitedRecordSize {
		content, err := io.ReadAll(file)
		r.ReadWork(artifact, int64(len(content)))
		return content, err
	}
	content, err := io.ReadAll(io.LimitReader(file, limit+1))
	r.ReadWork(artifact, int64(len(content)))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > limit {
		return nil, ErrReadLimit
	}
	return content, nil
}

func (r OSRuntime) ReadAt(artifact Artifact, offset, limit int64) ([]byte, bool, error) {
	filePath, err := r.resolveArtifact(artifact)
	if err != nil {
		return nil, false, err
	}
	if offset < 0 || limit < 0 || uint64(limit) > uint64(^uint(0)>>1) {
		return nil, false, ErrReadLimit
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()
	content := make([]byte, int(limit))
	read, err := file.ReadAt(content, offset)
	r.ReadWork(artifact, int64(read))
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, false, err
	}
	return content[:read], errors.Is(err, io.EOF), nil
}

// QuerySQLite accepts -1 for unlimited CSV output. Queries remain read-only,
// and stderr diagnostics retain their independent bound in either mode.
func (r OSRuntime) QuerySQLite(artifact Artifact, query string, limit int64) (SQLiteResult, error) {
	return r.QuerySQLiteContext(context.Background(), artifact, query, limit)
}

// QuerySQLiteContext cancels and joins the read-only SQLite subprocess.
func (r OSRuntime) QuerySQLiteContext(ctx context.Context, artifact Artifact, query string, limit int64) (SQLiteResult, error) {
	if err := ctx.Err(); err != nil {
		return SQLiteResult{}, err
	}
	filePath, err := r.resolveArtifact(artifact)
	if err != nil {
		return SQLiteResult{}, err
	}
	if limit < unlimitedRecordSize {
		return SQLiteResult{}, ErrReadLimit
	}
	stdout := newBoundedBuffer(limit)
	stderr := newBoundedBuffer(8192)
	command := exec.CommandContext(ctx, "sqlite3", "-readonly", "-csv", "-header", filePath, query)
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return SQLiteResult{}, ctx.Err()
		}
		if stdout.exceeded || stderr.exceeded {
			return SQLiteResult{}, ErrReadLimit
		}
		return SQLiteResult{}, fmt.Errorf("sqlite3: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	if stdout.exceeded {
		return SQLiteResult{}, ErrReadLimit
	}
	r.ReadWork(artifact, int64(len(stdout.Bytes())))
	r.DecodeWork(artifact, int64(len(stdout.Bytes())))
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

type boundedBuffer struct {
	buffer   bytes.Buffer
	limit    int64
	exceeded bool
}

func newBoundedBuffer(limit int64) *boundedBuffer { return &boundedBuffer{limit: limit} }

func (b *boundedBuffer) Write(content []byte) (int, error) {
	if b.limit == unlimitedRecordSize {
		return b.buffer.Write(content)
	}
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

// ResolveArtifactPath exposes the same root and symlink authorization used by
// artifact reads for consumers that pass a transcript reference to another tool.
func (r OSRuntime) ResolveArtifactPath(artifact Artifact) (string, error) {
	return r.resolveArtifact(artifact)
}
