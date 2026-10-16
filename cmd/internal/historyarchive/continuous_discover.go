package historyarchive

import (
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// ContinuousKinds are the sources a continuous pass archives (pair#592). The
// inventory's Couch stores and archive catalog are continuous too, but need
// per-slot or self-referential handling and stay out of this pass.
var ContinuousKinds = []string{"claude-transcripts", "codex-transcripts", "qoder-transcripts", "codex-database", "ops-documents", "sdlc-runtime"}

// SourceRoot is one resolved place a continuous kind reads from, with the
// identity its archived files are keyed by.
type SourceRoot struct {
	Kind     string
	Identity SourceIdentity
	Patterns []string
	Snapshot bool
}

// ContinuousRoots resolves every root of the continuous kinds. A kind whose
// root does not exist is skipped (an agent never installed has nothing to
// archive). ops-documents has one root per team under OpsRoot; sdlc-runtime has
// one per git common directory among the repositories under Workspace, so
// worktree slots of one repository share it.
func ContinuousRoots(ctx context.Context, inv SourceInventory, cfg ContinuousConfig, host, home string, env map[string]string) ([]SourceRoot, error) {
	var out []SourceRoot
	for _, kind := range ContinuousKinds {
		rule, err := inv.Resolve(kind)
		if err != nil {
			return nil, err
		}
		add := func(id SourceIdentity) {
			if info, err := os.Stat(id.Root); err == nil && info.IsDir() {
				out = append(out, SourceRoot{Kind: kind, Identity: id, Patterns: rule.Patterns, Snapshot: rule.SnapshotRequired})
			}
		}
		switch rule.RootToken {
		case "ops-worktree":
			if cfg.OpsRoot == "" {
				continue
			}
			teams, _ := filepath.Glob(filepath.Join(cfg.OpsRoot, "teams", "*"))
			sort.Strings(teams)
			for _, team := range teams {
				add(SourceIdentity{Host: host, Root: team, Repository: cfg.OpsRoot})
			}
		case "git-common":
			if cfg.Workspace == "" {
				continue
			}
			children, _ := os.ReadDir(cfg.Workspace)
			seen := map[string]bool{}
			for _, child := range children {
				dir := filepath.Join(cfg.Workspace, child.Name())
				if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
					continue
				}
				common, err := gitCommonDir(ctx, dir)
				if err != nil || seen[common] {
					continue
				}
				seen[common] = true
				add(SourceIdentity{Host: host, Root: common, Repository: common, Worktree: dir})
			}
		default:
			root, err := inv.ResolveRoot(kind, "", home, env)
			if err != nil {
				continue
			}
			add(SourceIdentity{Host: host, Root: root})
		}
	}
	return out, nil
}

func gitCommonDir(ctx context.Context, dir string) (string, error) {
	out, err := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		return "", err
	}
	return filepath.Clean(strings.TrimSpace(string(out))), nil
}

// Discover lists the regular files under root that the patterns name. A
// pattern matching a directory takes every regular file beneath it (Codex
// nests sessions by date). Symlinks are never followed or archived.
func Discover(ctx context.Context, root string, patterns []string) ([]Candidate, error) {
	fsys := os.DirFS(root)
	seen := map[string]bool{}
	var out []Candidate
	take := func(rel string, d fs.DirEntry) error {
		if seen[rel] || d.Type()&fs.ModeSymlink != 0 || !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		seen[rel] = true
		out = append(out, Candidate{Path: rel, Size: info.Size(), Mtime: info.ModTime().UTC(), Regular: true})
		return nil
	}
	for _, pattern := range patterns {
		matches, err := fs.Glob(fsys, pattern)
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			info, err := os.Lstat(filepath.Join(root, m))
			if err != nil || info.Mode()&fs.ModeSymlink != 0 {
				continue
			}
			if !info.IsDir() {
				if err := take(m, fs.FileInfoToDirEntry(info)); err != nil {
					return nil, err
				}
				continue
			}
			err = fs.WalkDir(fsys, m, func(rel string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if d.Type()&fs.ModeSymlink != 0 {
					return nil
				}
				if d.IsDir() {
					return ctx.Err()
				}
				return take(rel, d)
			})
			if err != nil {
				return nil, err
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}
