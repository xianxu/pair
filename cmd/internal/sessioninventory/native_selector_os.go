package sessioninventory

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"os/exec"
	"path/filepath"
)

func nativeSelectorCodexRuntime(rt Runtime, home string) (Runtime, error) {
	if !filepath.IsAbs(home) || filepath.Clean(home) != home {
		return nil, errors.New("Codex SQLite home is not canonical")
	}
	if r, ok := rt.(OSRuntime); ok {
		physical, err := filepath.EvalSymlinks(home)
		if err != nil || physical != home {
			return nil, errors.New("Codex SQLite home is not a verified physical directory")
		}
		roots := make(map[Agent][]StorageRoot, len(r.roots))
		for agent, items := range r.roots {
			roots[agent] = append([]StorageRoot(nil), items...)
		}
		roots[AgentCodex] = append(roots[AgentCodex], StorageRoot{Agent: AgentCodex, Name: "codex-state", Path: home})
		r.roots = roots
		return r, nil
	}
	return rt, nil
}
func validateCodexSelectorPath(rt Runtime, id, path string) error {
	if id == "" || !filepath.IsAbs(path) {
		return ErrPathEscape
	}
	for _, root := range rt.NativeRoots(AgentCodex) {
		if root.Name != "codex-sessions" {
			continue
		}
		rel, err := filepath.Rel(root.Path, path)
		if err != nil {
			continue
		}
		a := Artifact{StorageRoot: root.Name, RelativePath: filepath.ToSlash(rel), Kind: ArtifactTranscript}
		if !validArtifact(a) {
			continue
		}
		got, ok := codexPathID(a.RelativePath)
		if !ok || got != id {
			continue
		}
		if _, err := rt.StatFile(a); err == nil {
			return nil
		}
	}
	return ErrPathEscape
}

// QueryNativeSelectorSQLite is bounded and cancellable; it neither initializes
// Codex state nor runs its scan-and-repair fallback.
func (r OSRuntime) QueryNativeSelectorSQLite(ctx context.Context, a Artifact, query string, limit int64) (SQLiteResult, error) {
	if err := ctx.Err(); err != nil {
		return SQLiteResult{}, err
	}
	path, err := r.resolveArtifact(a)
	if err != nil {
		return SQLiteResult{}, err
	}
	if limit <= 0 || limit > 1<<20 {
		return SQLiteResult{}, ErrReadLimit
	}
	stdout, stderr := newBoundedBuffer(limit), newBoundedBuffer(8192)
	command := exec.CommandContext(ctx, "sqlite3", "-readonly", "-csv", "-header", path, query)
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return SQLiteResult{}, err
	}
	if stdout.exceeded || stderr.exceeded {
		return SQLiteResult{}, ErrReadLimit
	}
	records, err := csv.NewReader(bytes.NewReader(stdout.Bytes())).ReadAll()
	if err != nil {
		return SQLiteResult{}, err
	}
	if len(records) == 0 {
		return SQLiteResult{}, nil
	}
	return SQLiteResult{Columns: records[0], Rows: records[1:]}, nil
}
