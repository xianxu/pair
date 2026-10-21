package mockserve

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
)

type Snapshot struct {
	Version string
	Digest  string
	Entries int
	Bytes   int64
}
type scanEntry struct {
	name      string
	directory bool
	size      int64
	digest    [32]byte
}

// Scan visits each directory once and reads every regular file once. It retains
// only bounded entry metadata; content hashes detect changes independent of mtimes.
func (p *Project) Scan(ctx context.Context) (Snapshot, error) {
	var result Snapshot
	entries := make([]scanEntry, 0)
	versions := make([]string, 0)
	selection := ""
	hasSelection := false
	hasFixtures := false
	var walk func(*os.Root, string) error
	walk = func(root *os.Root, dir string) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		f, e := root.Open(".")
		if e != nil {
			return e
		}
		defer f.Close()
		st, e := f.Stat()
		if e != nil {
			return e
		}
		if !st.IsDir() {
			return fmt.Errorf("%s is not a directory", dir)
		}
		for {
			if e := ctx.Err(); e != nil {
				return e
			}
			batch, e := f.ReadDir(64)
			if e != nil && e != io.EOF {
				return e
			}
			for _, de := range batch {
				if e := ctx.Err(); e != nil {
					return e
				}
				if p.scanWork != nil {
					p.scanWork(1, 0)
				}
				result.Entries++
				if result.Entries > MaxEntries {
					return fmt.Errorf("project exceeds %d entries", MaxEntries)
				}
				name := de.Name()
				if dir != "." {
					name = dir + "/" + name
				}
				st, e := root.Lstat(de.Name())
				if e != nil {
					return e
				}
				entry := scanEntry{name: name, directory: st.IsDir()}
				if st.IsDir() {
					entries = append(entries, entry)
					if dir == "." {
						if _, e := ParseVersion(name); e == nil {
							versions = append(versions, name)
						}
					}
					child, e := p.openChildDirectory(root, de.Name(), name)
					if e != nil {
						return e
					}
					e = walk(child, name)
					child.Close()
					if e != nil {
						return e
					}
					continue
				}
				if !st.Mode().IsRegular() {
					return fmt.Errorf("%s is not a regular file or directory", name)
				}
				limit := MaxBytes - result.Bytes
				if strings.HasSuffix(strings.ToLower(name), ".html") && limit > MaxHTMLBytes {
					limit = MaxHTMLBytes
				}
				if name == ".mock-current" && limit > selectionLimit {
					limit = selectionLimit
				}
				if st.Size() > limit {
					return fmt.Errorf("%s exceeds remaining scan limit of %d bytes", name, limit)
				}
				data, e := p.readForScan(ctx, root, name, limit)
				if e != nil {
					return e
				}
				entry.size = int64(len(data))
				result.Bytes += entry.size
				entry.digest = sha256.Sum256(data)
				entries = append(entries, entry)
				if name == "fixtures.json" {
					if !json.Valid(data) {
						return fmt.Errorf("fixtures.json is not valid JSON")
					}
					hasFixtures = true
				}
				if name == ".mock-current" {
					hasSelection = true
					selection = strings.TrimSuffix(string(data), "\n")
					if _, e := ParseVersion(selection); e != nil {
						return fmt.Errorf("invalid .mock-current: %w", e)
					}
				}
			}
			if e == io.EOF {
				return nil
			}
		}
	}
	if e := walk(p.root, "."); e != nil {
		return Snapshot{}, e
	}
	if !hasFixtures {
		return Snapshot{}, fmt.Errorf("fixtures.json is required")
	}
	explicit := ""
	if hasSelection {
		explicit = selection
	}
	version, e := SelectVersion(versions, explicit)
	if e != nil {
		return Snapshot{}, e
	}
	result.Version = version
	hasIndex := false
	for _, entry := range entries {
		if entry.name == version+"/index.html" && !entry.directory {
			hasIndex = true
		}
	}
	if !hasIndex {
		return Snapshot{}, fmt.Errorf("%s/index.html is required", version)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	h := sha256.New()
	fmt.Fprintf(h, "version:%d:%s\n", len(version), version)
	for _, entry := range entries {
		fmt.Fprintf(h, "%d:%s:%t:%d:", len(entry.name), entry.name, entry.directory, entry.size)
		h.Write(entry.digest[:])
	}
	result.Digest = hex.EncodeToString(h.Sum(nil))
	return result, nil
}

func (p *Project) readForScan(ctx context.Context, root *os.Root, name string, limit int64) ([]byte, error) {
	f, e := p.openRegularAt(root, path.Base(name), name)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		return nil, e
	}
	if !st.Mode().IsRegular() || st.Size() > limit {
		return nil, fmt.Errorf("invalid or oversized file %s", name)
	}
	data := make([]byte, 0, int(st.Size()))
	buf := make([]byte, 32<<10)
	for {
		if e := ctx.Err(); e != nil {
			return nil, e
		}
		n, e := f.Read(buf)
		if p.scanWork != nil {
			p.scanWork(0, int64(n))
		}
		if int64(len(data))+int64(n) > limit {
			return nil, fmt.Errorf("%s exceeds remaining scan limit of %d bytes", name, limit)
		}
		data = append(data, buf[:n]...)
		if e == io.EOF {
			return data, nil
		}
		if e != nil {
			return nil, e
		}
	}
}
