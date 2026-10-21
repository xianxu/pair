package operationalrun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/xianxu/pair/cmd/internal/durablefile"
)

var stageName = regexp.MustCompile(`^\.run-stage-[0-9]+$`)
var stageFileName = regexp.MustCompile(`^\.(record|output)-[0-9]+$`)

type Store struct {
	Namespace string
	fault     func(string) error
}

func (s Store) fail(step string) error {
	if s.fault != nil {
		return s.fault(step)
	}
	return nil
}
func (s Store) task(k RunKey) (string, error) {
	if !k.valid() {
		return "", errors.New("invalid operational run key")
	}
	if !filepath.IsAbs(s.Namespace) || filepath.Clean(s.Namespace) != s.Namespace {
		return "", errors.New("invalid operational namespace")
	}
	return filepath.Join(s.Namespace, "schedule", k.Hub, k.Task), nil
}
func (s Store) Publish(ctx context.Context, key RunKey, r Record, output []byte) error {
	task, e := s.task(key)
	if e != nil {
		return e
	}
	if !sameKey(key, r.Key) {
		return errors.New("operational record key mismatch")
	}
	raw, e := encodeRecord(r)
	if e != nil {
		return e
	}
	if len(output) > OutputLimit {
		return errors.New("operational output exceeds byte cap")
	}
	unlock, e := taskLock(ctx, task, true)
	if e != nil {
		return e
	}
	defer unlock()
	if e = cleanupStages(ctx, task); e != nil {
		return e
	}
	runs := filepath.Join(task, "runs")
	if e = safeDir(runs, true); e != nil {
		return e
	}
	dest := filepath.Join(runs, key.ID)
	if _, e = os.Lstat(dest); e == nil {
		old, out, e := s.read(key)
		if e != nil {
			return e
		}
		encoded, e := encodeRecord(old)
		if e != nil {
			return e
		}
		if bytes.Equal(encoded, raw) && bytes.Equal(out, output) {
			return errors.Join(durablefile.SyncDirectory(runs), durablefile.SyncDirectory(task))
		}
		return errors.New("immutable operational run already exists")
	} else if !errors.Is(e, os.ErrNotExist) {
		return e
	}
	entries, e := names(runs)
	if e != nil {
		return e
	}
	if len(entries) >= 4096 {
		return errors.New("operational run inventory full; retain scheduler result and collect history before retrying publication")
	}
	stage, e := os.MkdirTemp(task, ".run-stage-")
	if e != nil {
		return e
	}
	defer func() {
		_ = os.Remove(filepath.Join(stage, "record.json"))
		_ = os.Remove(filepath.Join(stage, "output"))
		_ = os.Remove(stage)
	}()
	if e = durablefile.WriteAtomic(filepath.Join(stage, "record.json"), raw, ".record-"); e != nil {
		return e
	}
	if e = durablefile.WriteAtomic(filepath.Join(stage, "output"), output, ".output-"); e != nil {
		return e
	}
	if e = s.fail("publish-ready"); e != nil {
		return e
	}
	if e = ctx.Err(); e != nil {
		return e
	}
	if e = os.Rename(stage, dest); e != nil {
		return e
	}
	return errors.Join(durablefile.SyncDirectory(runs), durablefile.SyncDirectory(task))
}
func (s Store) readMetadata(key RunKey) (Record, error) {
	var r Record
	task, e := s.task(key)
	if e != nil {
		return r, e
	}
	dir := filepath.Join(task, "runs", key.ID)
	if _, e = bundleIdentity(dir); e != nil {
		return r, e
	}
	raw, e := readFile(filepath.Join(dir, "record.json"), RecordLimit)
	if e != nil {
		return r, e
	}
	if e = json.Unmarshal(raw, &r); e != nil {
		return r, e
	}
	if !sameKey(key, r.Key) {
		return r, errors.New("operational run key mismatch")
	}
	return r, nil
}
func (s Store) read(key RunKey) (Record, []byte, error) {
	r, e := s.readMetadata(key)
	if e != nil {
		return r, nil, e
	}
	task, e := s.task(key)
	if e != nil {
		return r, nil, e
	}
	out, e := readFile(filepath.Join(task, "runs", key.ID, "output"), OutputLimit)
	return r, out, e
}
func (s Store) Read(ctx context.Context, key RunKey) (Record, []byte, error) {
	var r Record
	task, e := s.task(key)
	if e != nil {
		return r, nil, e
	}
	unlock, e := taskLock(ctx, task, false)
	if e != nil {
		return r, nil, e
	}
	defer unlock()
	return s.read(key)
}
func (s Store) List(ctx context.Context, hub, task string, limit int) ([]Record, error) {
	k := RunKey{Hub: hub, Task: task, ID: "unused"}
	dir, e := s.task(k)
	if e != nil {
		return nil, e
	}
	unlock, e := taskLock(ctx, dir, false)
	if errors.Is(e, os.ErrNotExist) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	defer unlock()
	names, e := names(filepath.Join(dir, "runs"))
	if e != nil {
		return nil, e
	}
	var rows []Record
	for _, n := range names {
		if e = ctx.Err(); e != nil {
			return nil, e
		}
		k.ID = n
		r, e := s.readMetadata(k)
		if e != nil {
			return nil, e
		}
		rows = append(rows, r)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].EndedAt.Equal(rows[j].EndedAt) {
			return rows[i].Key.ID > rows[j].Key.ID
		}
		return rows[i].EndedAt.After(rows[j].EndedAt)
	})
	if limit <= 0 || limit > MaxRunsPerTask {
		limit = MaxRunsPerTask
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

// names bounds discovery including malformed entries; callers never follow them.
func names(dir string) ([]string, error) {
	if e := safeDir(dir, false); errors.Is(e, os.ErrNotExist) {
		return nil, nil
	} else if e != nil {
		return nil, e
	}
	f, e := os.Open(dir)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	ns, e := f.Readdirnames(4097)
	if e != nil && !errors.Is(e, io.EOF) {
		return nil, e
	}
	if len(ns) > 4096 {
		return nil, errors.New("operational directory inventory exceeds 4096 entries")
	}
	sort.Strings(ns)
	return ns, nil
}

// A task lock fences abandoned private staging directories from live publishers.
// Their reserved names never grant authority over an unknown child.
func cleanupStages(ctx context.Context, task string) error {
	ns, e := names(task)
	if e != nil {
		return e
	}
	for _, name := range ns {
		if e = ctx.Err(); e != nil {
			return e
		}
		if !stageName.MatchString(name) {
			continue
		}
		path := filepath.Join(task, name)
		children, e := names(path)
		if e != nil {
			return e
		}
		for _, child := range children {
			if child != "record.json" && child != "output" && !stageFileName.MatchString(child) {
				return errors.New("unknown operational publication staging member")
			}
			id, e := identity(filepath.Join(path, child))
			if e != nil {
				return e
			}
			if !os.FileMode(id.Mode).IsRegular() {
				return errors.New("unsafe operational staging member")
			}
		}
		for _, child := range children {
			if e = ctx.Err(); e != nil {
				return e
			}
			if e = os.Remove(filepath.Join(path, child)); e != nil {
				return e
			}
		}
		if e = os.Remove(path); e != nil {
			return e
		}
	}
	return durablefile.SyncDirectory(task)
}
