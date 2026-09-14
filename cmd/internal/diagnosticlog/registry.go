package diagnosticlog

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type RegistryEntry struct {
	Version   int    `json:"version"`
	Path      string `json:"path"`
	Directory string `json:"directory"`
	Lock      string `json:"lock"`
}

// RegistryDirectory contains exact writer-published paths, never wildcard
// authority. Registering a path alone does not authorize its collection.
func RegistryDirectory(root string) string { return filepath.Join(root, ".retention", "diagnostics") }
func register(root string, entry RegistryEntry) error {
	dir := RegistryDirectory(root)
	for _, p := range []string{filepath.Join(root, ".retention"), dir} {
		if e := os.Mkdir(p, 0700); e != nil && !os.IsExist(e) {
			return e
		}
		st, e := os.Lstat(p)
		if e != nil {
			return e
		}
		if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return errors.New("invalid diagnostic registry directory")
		}
	}
	sum := sha256.Sum256([]byte(entry.Path))
	return writeJSON(filepath.Join(dir, fmt.Sprintf("%x.json", sum)), entry, true)
}

// EnumerateRoot reads up to limit exact entries without initializing storage.
// The caller supplies a cursor to avoid repeatedly visiting the first page.
func EnumerateRoot(root string, offset, limit int) ([]RegistryEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	dir, e := checkedRegistry(root)
	if os.IsNotExist(e) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	f, e := os.Open(dir)
	if os.IsNotExist(e) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	defer f.Close()
	// Registry cardinality is bounded by configured distinct trace paths, but
	// cursor traversal still checks a caller-specified per-page bound.
	for offset > 0 {
		n := min(offset, 100)
		names, e := f.Readdirnames(n)
		offset -= len(names)
		if e == io.EOF {
			return nil, nil
		}
		if e != nil {
			return nil, e
		}
	}
	names, e := f.Readdirnames(limit)
	if e != nil && e != io.EOF {
		return nil, e
	}
	var entries []RegistryEntry
	for _, n := range names {
		if !strings.HasSuffix(n, ".json") {
			continue
		}
		var entry RegistryEntry
		if e = readJSON(filepath.Join(RegistryDirectory(root), n), &entry); e != nil {
			return nil, e
		}
		p, e := canonical(entry.Path)
		if e != nil {
			return nil, e
		}
		sum := sha256.Sum256([]byte(p))
		if entry.Version != 1 || p != entry.Path || entry.Directory != directory(p) || entry.Lock != lockPath(p) || n != fmt.Sprintf("%x.json", sum) {
			return nil, errors.New("invalid diagnostic registry entry")
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// EnvironmentOptions uses only an explicitly selected Pair directory; tests or
// standalone traces never fall back to real HOME. Root registration runs under the stable
// log lock and takes no root lock; collection may hold root then log.
func EnvironmentOptions(getenv func(string) string) Options {
	o := Options{Proof: DefaultProof}
	selected := getenv("PAIR_DATA_DIR")
	if selected == "" {
		return o
	}
	p, e := filepath.Abs(selected)
	if e != nil {
		return o
	}
	p, e = filepath.EvalSymlinks(p)
	if e != nil {
		return o
	}
	if filepath.Base(filepath.Dir(p)) == "repos" {
		p = filepath.Dir(filepath.Dir(p))
	}
	o.Root = p
	o.Registry = func(entry RegistryEntry) error { return register(p, entry) }
	o.Retire = func(entry RegistryEntry) error { return unregister(p, entry) }
	return o
}

func unregister(root string, entry RegistryEntry) error {
	if _, e := checkedRegistry(root); os.IsNotExist(e) {
		return nil
	} else if e != nil {
		return e
	}
	sum := sha256.Sum256([]byte(entry.Path))
	path := filepath.Join(RegistryDirectory(root), fmt.Sprintf("%x.json", sum))
	var found RegistryEntry
	if e := readJSON(path, &found); os.IsNotExist(e) {
		return nil
	} else if e != nil {
		return e
	}
	if found != entry {
		return errors.New("diagnostic registry changed")
	}
	if e := os.Remove(path); e != nil && !os.IsNotExist(e) {
		return e
	}
	return syncDir(RegistryDirectory(root))
}

func checkedRegistry(root string) (string, error) {
	canonicalRoot, e := filepath.EvalSymlinks(root)
	if e != nil {
		return "", e
	}
	for _, p := range []string{filepath.Join(canonicalRoot, ".retention"), RegistryDirectory(canonicalRoot)} {
		st, e := os.Lstat(p)
		if e != nil {
			return "", e
		}
		if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("invalid diagnostic registry directory")
		}
	}
	return RegistryDirectory(canonicalRoot), nil
}
