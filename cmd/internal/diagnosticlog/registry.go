package diagnosticlog

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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
func register(ctx context.Context, root string, entry RegistryEntry) error {
	return registerWithHook(ctx, root, entry, nil)
}
func registerWithHook(ctx context.Context, root string, entry RegistryEntry, hook func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
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
	return registryLocked(ctx, root, func() error {
		return publishJSON(dir, filepath.Join(dir, fmt.Sprintf("%x.json", sum)), entry, true, Options{Context: ctx}, hook)
	})
}

type RegistryPage struct {
	Entries    []RegistryEntry
	NextOffset int
	Complete   bool
}

// EnumerateRoot reads up to limit exact entries without initializing storage.
// The caller supplies a cursor to avoid repeatedly visiting the first page.
func EnumerateRoot(ctx context.Context, root string, offset, limit int) (RegistryPage, error) {
	if err := ctx.Err(); err != nil {
		return RegistryPage{}, err
	}
	if offset < 0 {
		return RegistryPage{}, errors.New("negative registry cursor")
	}
	nextOffset := offset
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	dir, e := checkedRegistry(root)
	if os.IsNotExist(e) {
		return RegistryPage{NextOffset: nextOffset, Complete: true}, nil
	}
	if e != nil {
		return RegistryPage{}, e
	}
	f, e := os.Open(dir)
	if os.IsNotExist(e) {
		return RegistryPage{NextOffset: nextOffset, Complete: true}, nil
	}
	if e != nil {
		return RegistryPage{}, e
	}
	defer f.Close()
	// Registry cardinality is bounded by configured distinct trace paths, but
	// cursor traversal still checks a caller-specified per-page bound.
	for offset > 0 {
		if err := ctx.Err(); err != nil {
			return RegistryPage{}, err
		}
		n := min(offset, 100)
		names, e := f.Readdirnames(n)
		offset -= len(names)
		if e == io.EOF {
			return RegistryPage{NextOffset: nextOffset, Complete: true}, nil
		}
		if e != nil {
			return RegistryPage{}, e
		}
	}
	if err := ctx.Err(); err != nil {
		return RegistryPage{}, err
	}
	names, e := f.Readdirnames(limit)
	nextOffset += len(names)
	complete := e == io.EOF
	if e != nil && e != io.EOF {
		return RegistryPage{}, e
	}
	var entries []RegistryEntry
	for _, n := range names {
		if err := ctx.Err(); err != nil {
			return RegistryPage{}, err
		}
		if !strings.HasSuffix(n, ".json") {
			continue
		}
		var entry RegistryEntry
		if e = readJSON(filepath.Join(RegistryDirectory(root), n), &entry); e != nil {
			return RegistryPage{}, e
		}
		if err := ctx.Err(); err != nil {
			return RegistryPage{}, err
		}
		p, e := canonical(entry.Path)
		if e != nil {
			return RegistryPage{}, e
		}
		sum := sha256.Sum256([]byte(p))
		if entry.Version != 1 || p != entry.Path || entry.Directory != directory(p) || entry.Lock != lockPath(p) || n != fmt.Sprintf("%x.json", sum) {
			return RegistryPage{}, errors.New("invalid diagnostic registry entry")
		}
		entries = append(entries, entry)
	}
	return RegistryPage{Entries: entries, NextOffset: nextOffset, Complete: complete}, nil
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
	o.Registry = func(ctx context.Context, entry RegistryEntry) error { return register(ctx, p, entry) }
	o.Retire = func(ctx context.Context, entry RegistryEntry) error { return unregister(ctx, p, entry) }
	return o
}

func unregister(ctx context.Context, root string, entry RegistryEntry) error {
	return registryLocked(ctx, root, func() error { return unregisterLocked(ctx, root, entry) })
}
func unregisterLocked(ctx context.Context, root string, entry RegistryEntry) error {
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

// Registry publishers share this permanent inode; recovery never takes a log
// lock, so log→registry lock ordering is preserved. Busy maintenance yields.
func registryLocked(ctx context.Context, root string, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	dir, err := checkedRegistry(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	f, err := openRegular(filepath.Join(dir, "registry.lock"), syscall.O_CREAT|syscall.O_RDWR)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return ErrBusy
		}
		return err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := clearPublication(dir, Options{Context: ctx}); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return fn()
}
func RecoverRegistry(ctx context.Context, root string) error {
	return registryLocked(ctx, root, func() error { return nil })
}
