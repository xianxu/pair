package diagnosticlog

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Segment struct {
	Path      string    `json:"path"`
	Bytes     int64     `json:"bytes"`
	LastWrite time.Time `json:"last_write"`
	Eligible  bool      `json:"eligible"`
	Reason    string    `json:"reason"`
}

// DecideSegment is independent of session activity and Couch visibility.
func DecideSegment(now, lastWrite time.Time) bool {
	return !lastWrite.IsZero() && !lastWrite.After(now) && !now.Before(lastWrite.Add(RetentionPeriod))
}

// Preview is read-only: it does not create a lock, initialize or recover state.
// At most limit segment metadata entries are visited. Callers schedule more work
// by repeating after collection; a current file is always an additional row.
func Preview(path string, options Options, limit int) ([]Segment, error) {
	rows, _, done, e := previewManagedPage(path, options, "", limit)
	if e == nil && !done {
		e = ErrMore
	}
	return rows, e
}

// Collect revalidates and unlinks only exact expired generations. It never
// removes the stable lock inode or scans arbitrary external trace filenames.
func Collect(path string, options Options, limit int) ([]Segment, error) {
	rows, _, done, e := collectManagedPage(path, options, "", limit)
	if e == nil && !done {
		e = ErrMore
	}
	return rows, e
}
func collectManagedPage(path string, options Options, cursor string, limit int) (removed []Segment, next string, complete bool, err error) {
	if err := options.checkContext(); err != nil {
		return nil, cursor, false, err
	}
	p, e := canonical(path)
	if e != nil {
		return nil, cursor, false, e
	}
	options = normalized(options)
	complete = true
	e = lockedOptions(p, false, options, func() error {
		if err := recoverPublication(p, options); err != nil {
			return err
		}
		s, e := load(p)
		if os.IsNotExist(e) {
			if _, e := os.Lstat(p); !os.IsNotExist(e) {
				return errors.New("diagnostic current lacks metadata")
			}
			if options.Proof == nil {
				return ErrUnknownWriters
			}
			if e := options.prove(p, nil); e != nil {
				return e
			}
			if _, e := os.Lstat(directory(p)); e == nil {
				// Only an empty directory is a recoverable final retirement;
				// unknown payload or metadata still blocks instead of being lost.
				if err := options.checkContext(); err != nil {
					return err
				}
				if e = os.Remove(directory(p)); e != nil {
					return e
				}
				if e = syncDir(filepath.Dir(p)); e != nil {
					return e
				}
			} else if !os.IsNotExist(e) {
				return e
			}
			return retire(p, options)
		}
		if e != nil {
			return e
		}
		if e = validate(s, p); e != nil {
			return e
		}
		if options.Proof == nil {
			return ErrUnknownWriters
		}
		if e = options.prove(p, s.Writers); e != nil {
			return e
		}
		if s.Deleting != nil {
			if e = recoverDeletion(p, &s, options); e != nil {
				return e
			}
		}
		if s.Pending != nil {
			if e = recoverRotation(p, &s, options); e != nil {
				return e
			}
		}
		var rows []Segment
		rows, next, complete, e = previewPageLocked(p, options, cursor, limit, true)
		if e != nil {
			return e
		}
		for _, r := range rows {
			if err := options.checkContext(); err != nil {
				return err
			}
			if !r.Eligible {
				continue
			}
			g := s.Current
			if r.Path != p {
				if e = readJSON(r.Path+".json", &g); e != nil {
					return e
				}
			}
			s.Deleting = &g
			if err := options.checkContext(); err != nil {
				return err
			}
			if e = save(p, s, true, options); e != nil {
				return e
			}
			if e = fault(options, "delete-intent"); e != nil {
				return e
			}
			if e = recoverDeletion(p, &s, options); e != nil {
				return e
			}
			removed = append(removed, r)
		}
		if len(removed) > 0 {
			if e = syncDir(directory(p)); e != nil {
				return e
			}
			if e = syncDir(filepath.Dir(p)); e != nil {
				return e
			}
		}
		if e = cleanupTemps(p, options, limit); e != nil {
			return e
		}
		return retireEmpty(p, s, options)
	})
	return removed, next, complete, e
}

// Deletion has an exact durable intent just like rotation. A crash after unlink
// cannot leave a missing current file permanently blocking future writers.
func recoverDeletion(path string, s *diskState, o Options) error {
	if err := o.checkContext(); err != nil {
		return err
	}
	g := *s.Deleting
	p := path
	if g.Name != "" {
		if !validSegment(g.Name) {
			return errors.New("invalid diagnostic deletion")
		}
		p = segmentPath(path, g.Name)
	}
	if g.Name != "" {
		if e := syncExistingParent(filepath.Dir(p), directory(path), o); e != nil {
			return e
		}
		var metadata generation
		if e := readJSON(p+".json", &metadata); e == nil {
			if metadata != g {
				return errors.New("diagnostic deletion metadata changed")
			}
		} else if !os.IsNotExist(e) {
			return e
		}
	}
	if st, e := regular(p); e == nil {
		if e = matches(st, g); e != nil {
			return e
		}
		if err := o.checkContext(); err != nil {
			return err
		}
		if e = os.Remove(p); e != nil {
			return e
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	if e := syncExistingParent(filepath.Dir(p), deletionBase(path, g), o); e != nil {
		return e
	}
	if e := fault(o, "delete-payload"); e != nil {
		return e
	}
	if g.Name != "" {
		if err := o.checkContext(); err != nil {
			return err
		}
		if e := os.Remove(p + ".json"); e != nil && !os.IsNotExist(e) {
			return e
		}
		if e := syncExistingParent(filepath.Dir(p), deletionBase(path, g), o); e != nil {
			return e
		}
		if e := fault(o, "delete-metadata"); e != nil {
			return e
		}
		if e := removeEmptyParents(filepath.Dir(p), directory(path), o); e != nil {
			return e
		}
	} else {
		s.Current = generation{}
	}
	if e := fault(o, "delete-retire-intent"); e != nil {
		return e
	}
	s.Deleting = nil
	if err := o.checkContext(); err != nil {
		return err
	}
	return save(path, *s, true, o)
}
func retire(path string, o Options) error {
	if err := o.checkContext(); err != nil {
		return err
	}
	if o.Retire != nil {
		return o.Retire(o.context(), RegistryEntry{Version: 1, Path: path, Directory: directory(path), Lock: lockPath(path)})
	}
	return nil
}
func retireEmpty(path string, s diskState, o Options) error {
	if s.Current.Identity != (identity{}) || s.Pending != nil || s.Deleting != nil {
		return nil
	}
	for _, r := range s.Writers {
		if !definitelyDead(r) {
			return nil
		}
	}
	f, e := os.Open(directory(path))
	if e != nil {
		return e
	}
	names, e := f.Readdirnames(2)
	f.Close()
	if e != nil && e != io.EOF {
		return e
	}
	if len(names) != 1 || names[0] != "state.json" {
		return nil
	}
	// Open republishes registrations while holding this same log lock. It cannot
	// write through a registry row retired by a concurrent collector.
	if err := o.checkContext(); err != nil {
		return err
	}
	if e = os.Remove(filepath.Join(directory(path), "state.json")); e != nil {
		return e
	}
	if e = syncDir(directory(path)); e != nil {
		return e
	}
	if e = fault(o, "retire-state"); e != nil {
		return e
	}
	if err := o.checkContext(); err != nil {
		return err
	}
	if e = os.Remove(directory(path)); e != nil {
		return e
	}
	if e = syncDir(filepath.Dir(path)); e != nil {
		return e
	}
	if e = fault(o, "retire-directory"); e != nil {
		return e
	}
	return retire(path, o)
}

// Atomic replacement temps left by process death contain only derived
// diagnostic metadata. Their fixed generated-name grammar and age are checked
// under the writer lock; unknown entries remain untouched.
func cleanupTemps(path string, o Options, limit int) error {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	f, e := os.Open(directory(path))
	if e != nil {
		return e
	}
	names, e := f.Readdirnames(limit)
	f.Close()
	if e != nil && e != io.EOF {
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
		p := filepath.Join(directory(path), name)
		st, e := regular(p)
		if e != nil {
			return e
		}
		if !DecideSegment(o.Now(), st.ModTime()) {
			continue
		}
		if err := o.checkContext(); err != nil {
			return err
		}
		if e = os.Remove(p); e != nil {
			return e
		}
		changed = true
	}
	if changed {
		return syncDir(directory(path))
	}
	return nil
}

func deletionBase(path string, g generation) string {
	if g.Name == "" {
		return filepath.Dir(path)
	}
	return directory(path)
}
