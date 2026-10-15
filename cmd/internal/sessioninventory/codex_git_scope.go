package sessioninventory

import (
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type codexGitIdentity struct{ common, primary, relative string }

func codexGitRoot(cwd string) string {
	for d, n := cwd, 0; n < 128; n++ {
		p := filepath.Join(d, ".git")
		st, err := os.Stat(p)
		if err == nil {
			if !st.IsDir() {
				return d
			}
			if _, err := os.Stat(filepath.Join(p, "HEAD")); err == nil {
				return d
			}
		}
		parent := filepath.Dir(d)
		if parent == d {
			break
		}
		d = parent
	}
	return ""
}
func codexGitPath(path, base, prefix string) (string, error) {
	st, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !st.Mode().IsRegular() || st.Size() > 65536 {
		return "", errCodexConfigObservation
	}
	b, err := codexReadBoundedRegular(path, 65536)
	if err != nil {
		return "", err
	}
	v := strings.TrimSpace(string(b))
	if !strings.HasPrefix(v, prefix) {
		return "", errCodexConfigObservation
	}
	v = strings.TrimSpace(strings.TrimPrefix(v, prefix))
	if v == "" || strings.ContainsAny(v, "\r\n") {
		return "", errCodexConfigObservation
	}
	if !filepath.IsAbs(v) {
		v = filepath.Join(base, v)
	}
	// Trust resolution rejects an original git-dir symlink before canonicalizing.
	// Retain that provenance; unmodeled nested aliases stay native compatibility.
	target, err := os.Lstat(v)
	if err != nil {
		return "", err
	}
	if target.Mode()&os.ModeSymlink != 0 || codexNestedSymlink(v) {
		return "", errCodexConfigObservation
	}
	return filepath.EvalSymlinks(v)
}
func codexRepositoryIdentity(cwd string) (codexGitIdentity, bool) {
	c, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		return codexGitIdentity{}, false
	}
	root := codexGitRoot(c)
	if root == "" {
		return codexGitIdentity{}, false
	}
	git := filepath.Join(root, ".git")
	st, err := os.Lstat(git)
	if err != nil || st.Mode()&os.ModeSymlink != 0 {
		return codexGitIdentity{}, false
	}
	common := ""
	if st.IsDir() {
		common, err = filepath.EvalSymlinks(git)
	} else if st.Mode().IsRegular() {
		var admin string
		admin, err = codexGitPath(git, root, "gitdir:")
		if err == nil {
			common, err = codexGitPath(filepath.Join(admin, "commondir"), admin, "")
		}
		if err == nil {
			var registered string
			registered, err = filepath.EvalSymlinks(filepath.Join(common, "worktrees"))
			if err == nil && filepath.Dir(admin) != registered {
				err = errCodexConfigObservation
			}
			back, e := codexGitPath(filepath.Join(admin, "gitdir"), admin, "")
			if e != nil || back != git {
				err = errCodexConfigObservation
			}
		}
	} else {
		return codexGitIdentity{}, false
	}
	if err != nil {
		return codexGitIdentity{}, false
	}
	primary := filepath.Dir(common)
	primaryGit := filepath.Join(primary, ".git")
	ps, err := os.Lstat(primaryGit)
	if err != nil || !ps.IsDir() || ps.Mode()&os.ModeSymlink != 0 {
		return codexGitIdentity{}, false
	}
	pc, err := filepath.EvalSymlinks(primaryGit)
	if err != nil || pc != common {
		return codexGitIdentity{}, false
	}
	relative, err := filepath.Rel(root, c)
	if err != nil {
		return codexGitIdentity{}, false
	}
	return codexGitIdentity{common: common, primary: primary, relative: relative}, true
}
func codexWorktreeCWDs(cwd string) ([]string, error) {
	id, ok := codexRepositoryIdentity(cwd)
	if !ok {
		return []string{cwd}, nil
	}
	canonical, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		return nil, err
	}
	out := []string{cwd}
	seen := map[string]bool{cwd: true}
	if !seen[canonical] {
		out = append(out, canonical)
		seen[canonical] = true
	}
	appendRoot := func(root string) {
		c, e := filepath.EvalSymlinks(filepath.Join(root, id.relative))
		if e != nil || seen[c] {
			return
		}
		st, e := os.Stat(c)
		if e != nil || !st.IsDir() {
			return
		}
		other, ok := codexRepositoryIdentity(c)
		if ok && other.common == id.common && other.relative == id.relative {
			seen[c] = true
			out = append(out, c)
		}
	}
	appendRoot(id.primary)
	dir := filepath.Join(id.common, "worktrees")
	entries, err := codexWorktreeEntries(dir)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	if len(entries) > 256 {
		return nil, errCodexConfigObservation
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		admin, err := filepath.EvalSymlinks(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		parent, err := filepath.EvalSymlinks(dir)
		if err != nil {
			return nil, err
		}
		if filepath.Dir(admin) != parent {
			continue
		}
		git, e := codexGitPath(filepath.Join(admin, "gitdir"), admin, "")
		if e != nil || filepath.Base(git) != ".git" {
			continue
		}
		appendRoot(filepath.Dir(git))
	}
	return out, nil
}
func codexProjectConfigLayers(cwd, home string, layers []codexConfigLayer, load func(string) (codexConfigLayer, error)) ([]codexConfigLayer, error) {
	merged := map[string]any{}
	for _, l := range layers {
		codexMergeConfig(merged, l.values)
	}
	markers := []string{".git"}
	if v, ok := merged["project_root_markers"]; ok {
		a, ok := v.([]any)
		if !ok {
			return nil, errCodexConfigObservation
		}
		markers = nil
		for _, v := range a {
			s, ok := v.(string)
			if !ok {
				return nil, errCodexConfigObservation
			}
			markers = append(markers, s)
		}
	}
	root := cwd
	found := false
	for d, n := cwd, 0; n < 128; n++ {
		for _, m := range markers {
			st, err := os.Stat(filepath.Join(d, m))
			if err != nil {
				continue
			}
			if m == ".git" && st.IsDir() {
				if _, err := os.Stat(filepath.Join(d, m, "HEAD")); err != nil {
					continue
				}
			}
			root = d
			found = true
			break
		}
		if found || filepath.Dir(d) == d {
			break
		}
		d = filepath.Dir(d)
	}
	primary := ""
	if id, ok := codexRepositoryIdentity(cwd); ok {
		primary = id.primary
	} else if gitRoot := codexGitRoot(cwd); gitRoot != "" {
		if st, err := os.Lstat(filepath.Join(gitRoot, ".git")); err == nil && !st.IsDir() {
			return nil, errCodexConfigObservation
		}
	}
	trust := map[string]any{}
	if v, ok := merged["projects"]; ok {
		var yes bool
		trust, yes = v.(map[string]any)
		if !yes {
			return nil, errCodexConfigObservation
		}
	}
	isTrusted := func(d string) bool {
		for _, p := range []string{d, root, primary} {
			if p == "" {
				continue
			}
			canonical, _ := filepath.EvalSymlinks(p)
			keys := []string{canonical, p}
			for _, k := range keys {
				if m, ok := trust[k].(map[string]any); ok {
					if v, ok := m["trust_level"]; ok {
						return v == "trusted"
					}
				}
			}
		}
		return false
	}
	dirs := []string{}
	for d, n := cwd, 0; n < 128; n++ {
		dirs = append(dirs, d)
		if d == root {
			break
		}
		p := filepath.Dir(d)
		if p == d {
			return nil, errCodexConfigObservation
		}
		d = p
	}
	if len(dirs) == 128 {
		return nil, errCodexConfigObservation
	}
	out := []codexConfigLayer{}
	homeCanonical, _ := filepath.EvalSymlinks(home)
	for i := len(dirs) - 1; i >= 0; i-- {
		d := dirs[i]
		folder := filepath.Join(d, ".codex")
		c, _ := filepath.EvalSymlinks(folder)
		if folder == home || (c != "" && c == homeCanonical) || !isTrusted(d) {
			continue
		}
		l, err := load(filepath.Join(folder, "config.toml"))
		if err != nil {
			return nil, err
		}
		for _, k := range []string{"model_provider", "model_providers", "profile", "profiles", "openai_base_url", "chatgpt_base_url", "notify"} {
			delete(l.values, k)
		}
		out = append(out, l)
	}
	return out, nil
}

// Bound enumeration before allocation, and never open a substituted special file.
func codexWorktreeEntries(path string) ([]os.DirEntry, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	entries, err := f.ReadDir(257)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(entries) > 256 {
		return nil, errCodexConfigObservation
	}
	return entries, nil
}
