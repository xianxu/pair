package diagnosticlog

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

var ErrMore = errors.New("diagnostic page incomplete")

const currentCursor = "current"
const leafGenerations = 16

func segmentPath(path, name string) string {
	token := strings.TrimSuffix(strings.TrimPrefix(name, "segment-"), ".log")
	return filepath.Join(directory(path), "segments", token[0:1], token[1:2], token[2:3], token[3:4], name)
}
func ensureSegmentDirectory(path, name string) error {
	dir := filepath.Join(directory(path), "segments")
	parts := []string{dir}
	token := name[8:12]
	for _, ch := range token {
		dir = filepath.Join(dir, string(ch))
		parts = append(parts, dir)
	}
	for _, p := range parts {
		e := os.Mkdir(p, 0700)
		if e != nil && !os.IsExist(e) {
			return e
		}
		if e == nil {
			if e = syncDir(filepath.Dir(p)); e != nil {
				return e
			}
		}
		st, e := os.Lstat(p)
		if e != nil {
			return e
		}
		if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return errors.New("invalid diagnostic segment directory")
		}
	}
	return nil
}
func directoryNames(path string, limit int) ([]string, error) {
	fd, e := syscall.Open(path, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if e != nil {
		return nil, e
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	names, e := f.Readdirnames(limit)
	if e != nil && e != io.EOF {
		return nil, e
	}
	return names, nil
}
func removeEmptyParents(path, base string) error {
	for path != base {
		if e := os.Remove(path); e != nil {
			if errors.Is(e, syscall.ENOTEMPTY) || errors.Is(e, syscall.EEXIST) {
				return nil
			}
			if !os.IsNotExist(e) {
				return e
			}
		}
		if e := syncDir(filepath.Dir(path)); e != nil {
			return e
		}
		path = filepath.Dir(path)
	}
	return nil
}
func normalizedLimit(limit int) int {
	if limit <= 0 || limit > 100 {
		return 100
	}
	return limit
}
func validCursor(cursor string) bool {
	return cursor == "" || cursor == currentCursor || validSegment(cursor)
}

// generationPage uses a fixed-depth 16-way radix directory. Each leaf contains
// at most16 generations (32 payload/metadata names). No full-directory sort or
// unstable directory offset is used. A page visits at most128*(limit+1) names;
// malformed overfull nodes fail closed after one bounded read.
func generationPage(path, after string, limit int, cleanup bool) (names []string, next string, complete bool, err error) {
	limit = normalizedLimit(limit)
	budget := 128 * (limit + 1)
	next = after
	complete = true
	var walk func(string, string) bool
	walk = func(dir, prefix string) bool {
		upper := "segment-" + prefix + strings.Repeat("f", 32-len(prefix)) + ".log"
		if after != "" && upper <= after {
			return true
		}
		capNames := 17
		if len(prefix) == 4 {
			capNames = 2*leafGenerations + 1
		}
		needed := capNames
		if cleanup && len(prefix) == 4 {
			needed += 2*leafGenerations + 1
		}
		if budget < needed {
			complete = false
			return false
		}
		if cleanup && len(prefix) == 4 {
			if e := cleanupGenerationTemps(dir); e != nil && !os.IsNotExist(e) {
				err = e
				return false
			}
			budget -= 2*leafGenerations + 1
		}
		children, e := directoryNames(dir, capNames)
		if os.IsNotExist(e) {
			next = upper
			return true
		}
		if e != nil {
			err = e
			return false
		}
		budget -= len(children)
		if len(children) >= capNames {
			err = errors.New("diagnostic radix node exceeds bounded capacity")
			return false
		}
		sort.Strings(children)
		if len(prefix) < 4 {
			for _, child := range children {
				if len(child) != 1 || !strings.Contains("0123456789abcdef", child) {
					err = errors.New("invalid diagnostic radix branch")
					return false
				}
				if !walk(filepath.Join(dir, child), prefix+child) {
					return false
				}
			}
		} else {
			raw := map[string]bool{}
			meta := map[string]bool{}
			for _, child := range children {
				name := strings.TrimSuffix(child, ".json")
				if !validSegment(name) || name[8:12] != prefix {
					err = errors.New("invalid diagnostic radix leaf")
					return false
				}
				if child == name {
					raw[name] = true
				} else {
					meta[name] = true
				}
			}
			if len(raw) != len(meta) {
				err = errors.New("unpaired diagnostic generation")
				return false
			}
			for name := range raw {
				if !meta[name] {
					err = errors.New("unpaired diagnostic generation")
					return false
				}
			}
			for _, child := range children {
				if !strings.HasSuffix(child, ".json") {
					continue
				}
				name := strings.TrimSuffix(child, ".json")
				if name <= after {
					continue
				}
				names = append(names, name)
				next = name
				if len(names) >= limit {
					complete = false
					return false
				}
			}
		}
		if cleanup {
			if e := os.Remove(dir); e != nil && !errors.Is(e, syscall.ENOTEMPTY) && !errors.Is(e, syscall.EEXIST) && !os.IsNotExist(e) {
				err = e
				return false
			}
		}
		next = upper
		return true
	}
	walk(filepath.Join(directory(path), "segments"), "")
	if err != nil {
		return nil, next, false, err
	}
	if complete {
		next = ""
	}
	return
}

func previewPageLocked(path string, o Options, cursor string, limit int, cleanup bool) (out []Segment, next string, complete bool, err error) {
	if !validCursor(cursor) {
		return nil, cursor, false, errors.New("invalid diagnostic page cursor")
	}
	limit = normalizedLimit(limit)
	s, e := load(path)
	if e != nil {
		return nil, cursor, false, e
	}
	if e = validate(s, path); e != nil {
		return nil, cursor, false, e
	}
	if s.Pending != nil || s.Deleting != nil {
		return nil, cursor, false, errors.New("diagnostic recovery pending")
	}
	reason := ""
	if o.Proof == nil {
		reason = ErrUnknownWriters.Error()
	} else if e = o.Proof(path, s.Writers); e != nil {
		reason = e.Error()
	}
	add := func(p string, g generation) error {
		st, e := regular(p)
		if e != nil {
			return e
		}
		if e = matches(st, g); e != nil {
			return e
		}
		eligible := reason == "" && DecideSegment(o.Now(), g.LastWrite)
		why := reason
		if why == "" && !eligible {
			why = "within seven-day retention"
		}
		out = append(out, Segment{p, st.Size(), g.LastWrite, eligible, why})
		return nil
	}
	if cursor == "" && s.Current.Identity != (identity{}) {
		if e = add(path, s.Current); e != nil {
			return nil, cursor, false, e
		}
		if len(out) == limit {
			return out, currentCursor, false, nil
		}
	}
	after := cursor
	if after == currentCursor {
		after = ""
	}
	names, next, complete, e := generationPage(path, after, limit-len(out), cleanup)
	if e != nil {
		return nil, cursor, false, e
	}
	for _, name := range names {
		p := segmentPath(path, name)
		var g generation
		if e = readJSON(p+".json", &g); e != nil {
			return nil, cursor, false, e
		}
		if g.Name != name || g.Start.IsZero() || g.LastWrite.IsZero() {
			return nil, cursor, false, errors.New("invalid diagnostic generation")
		}
		if e = add(p, g); e != nil {
			return nil, cursor, false, e
		}
	}
	if !complete && next == "" {
		next = currentCursor
	}
	return out, next, complete, nil
}

func previewManagedPage(path string, options Options, cursor string, limit int) (rows []Segment, next string, complete bool, err error) {
	p, e := canonical(path)
	if e != nil {
		return nil, cursor, false, e
	}
	options = normalized(options)
	err = locked(p, false, func() error {
		var e error
		rows, next, complete, e = previewPageLocked(p, options, cursor, limit, false)
		return e
	})
	return
}

// PreviewLegacyPage is read-only. Empty cursor starts with the current file;
// subsequent opaque cursors traverse immutable generations despite removals.
func PreviewLegacyPage(path string, options Options, cursor string, limit int) ([]Segment, string, bool, error) {
	if !validCursor(cursor) {
		return nil, cursor, false, errors.New("invalid diagnostic page cursor")
	}
	p, e := canonical(path)
	if e != nil {
		return nil, cursor, false, e
	}
	if _, e = os.Lstat(directory(p)); e == nil {
		return previewManagedPage(p, options, cursor, limit)
	} else if !os.IsNotExist(e) {
		return nil, cursor, false, e
	}
	rows, e := previewLegacyCurrent(p, options)
	if os.IsNotExist(e) {
		return nil, "", true, nil
	}
	return rows, "", true, e
}
func CollectLegacyPage(path string, options Options, cursor string, limit int) ([]Segment, string, bool, error) {
	if !validCursor(cursor) {
		return nil, cursor, false, errors.New("invalid diagnostic page cursor")
	}
	p, e := canonical(path)
	if e != nil {
		return nil, cursor, false, e
	}
	if _, e = os.Lstat(directory(p)); e == nil {
		return collectManagedPage(p, options, cursor, limit)
	} else if !os.IsNotExist(e) {
		return nil, cursor, false, e
	}
	if _, e = os.Lstat(p); os.IsNotExist(e) {
		return collectManagedPage(p, options, cursor, limit)
	} else if e != nil {
		return nil, cursor, false, e
	}
	adopted, e := adoptLegacy(p, options)
	if e != nil || !adopted {
		return nil, "", true, e
	}
	return collectManagedPage(p, options, cursor, limit)
}

// Leaf temporary files are solely atomic generation-metadata replacements.
// Under the protocol lock none can be an active write; interrupted generation
// publication is recovered before scanning or deleting immutable generations.
func cleanupGenerationTemps(dir string) error {
	names, e := directoryNames(dir, 2*leafGenerations+1)
	if e != nil {
		return e
	}
	changed := false
	for _, name := range names {
		suffix, ok := strings.CutPrefix(name, ".pending-")
		if !ok {
			continue
		}
		n, e := strconv.ParseUint(suffix, 10, 64)
		if e != nil || strconv.FormatUint(n, 10) != suffix {
			continue
		}
		p := filepath.Join(dir, name)
		if _, e = regular(p); e != nil {
			return e
		}
		if e = os.Remove(p); e != nil {
			return e
		}
		changed = true
	}
	if changed {
		return syncDir(dir)
	}
	return nil
}
