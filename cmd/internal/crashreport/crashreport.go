// Package crashreport keeps a console-owning couch's fatal panics on disk.
//
// A Go panic writes its stack only to stderr, and couch's stderr is the
// terminal it then redraws over, so a crash used to leave nothing behind
// (#397). Install points the runtime's crash output at a per-run file; the
// next incarnation reports what the previous one left, once.
package crashreport

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
)

const (
	unreported = ".log"
	reported   = ".crash"
	stampForm  = "20060102T150405Z"
	// maxEntries bounds the startup scan and the gc walk: one file per console
	// run, so a directory this large is debris, not history.
	maxEntries = 4096
)

// Dir is the crash directory inside one couch store namespace. One store per
// singleton selection, so isolated roots keep their own crashes.
func Dir(store string) string { return filepath.Join(store, "crash") }

func parseName(name string) (time.Time, int, string, bool) {
	ext := filepath.Ext(name)
	if ext != unreported && ext != reported {
		return time.Time{}, 0, "", false
	}
	stamp, pidText, ok := strings.Cut(strings.TrimSuffix(name, ext), "-")
	if !ok {
		return time.Time{}, 0, "", false
	}
	at, err := time.Parse(stampForm, stamp)
	pid, perr := strconv.Atoi(pidText)
	if err != nil || perr != nil || pid <= 0 || strconv.Itoa(pid) != pidText {
		return time.Time{}, 0, "", false
	}
	return at, pid, ext, true
}

// Kind is what a previous incarnation's file says about how it ended.
type Kind int

const (
	// Crashed: the runtime wrote a fatal panic or error into the file.
	Crashed Kind = iota + 1
	// Abrupt: the file is empty but was never removed, so the process died
	// without a panic -- SIGKILL, power loss, or the memory killer.
	Abrupt
	// Reported: a crash already shown once, kept for retention.
	Reported
)

type File struct {
	Name string
	Size int64
}

type Finding struct {
	Name string
	Kind Kind
}

// Classify is the pure decision over a directory listing. Names outside the
// grammar are not ours and are ignored.
func Classify(files []File) []Finding {
	var out []Finding
	for _, f := range files {
		_, _, ext, ok := parseName(f.Name)
		switch {
		case !ok:
		case ext == reported:
			out = append(out, Finding{f.Name, Reported})
		case f.Size > 0:
			out = append(out, Finding{f.Name, Crashed})
		default:
			out = append(out, Finding{f.Name, Abrupt})
		}
	}
	return out
}

// Report is one previous ending to tell the operator about.
type Report struct {
	Kind Kind
	Path string
}

func (r Report) Notice() string {
	switch r.Kind {
	case Crashed:
		return "previous couch crashed — see " + r.Path
	case Abrupt:
		return "previous couch ended abruptly (no panic recorded)"
	}
	return ""
}

type Capture struct {
	file *os.File
}

// Install reports the previous incarnations' files, then routes this
// process's crash output to a fresh file.
//
// The caller must hold couch's singleton lease: that is what proves every file
// already in dir belongs to a process that is gone.
func Install(dir string, now time.Time, pid int) (*Capture, []Report, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, nil, err
	}
	files, err := list(dir)
	if err != nil {
		return nil, nil, err
	}
	var reports []Report
	for _, f := range Classify(files) {
		path := filepath.Join(dir, f.Name)
		switch f.Kind {
		case Crashed:
			// Renamed first, reported second: a report whose rename failed
			// would come back on every start.
			next := strings.TrimSuffix(path, unreported) + reported
			if err := os.Rename(path, next); err != nil {
				return nil, reports, err
			}
			reports = append(reports, Report{Kind: Crashed, Path: next})
		case Abrupt:
			if err := os.Remove(path); err != nil {
				return nil, reports, err
			}
			reports = append(reports, Report{Kind: Abrupt, Path: path})
		}
	}
	name := fmt.Sprintf("%s-%d%s", now.UTC().Format(stampForm), pid, unreported)
	file, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, reports, err
	}
	if err := debug.SetCrashOutput(file, debug.CrashOptions{}); err != nil {
		file.Close()
		os.Remove(file.Name())
		return nil, reports, err
	}
	return &Capture{file: file}, reports, nil
}

// Close ends capture on a clean exit. An empty file is removed: only a file
// that survives its process says something.
func (c *Capture) Close() error {
	if c == nil || c.file == nil {
		return nil
	}
	file := c.file
	c.file = nil
	err := debug.SetCrashOutput(nil, debug.CrashOptions{})
	info, statErr := file.Stat()
	err = errors.Join(err, file.Close())
	if statErr == nil && info.Size() == 0 {
		err = errors.Join(err, os.Remove(file.Name()))
	}
	return err
}

// list reads regular files only (Lstat), bounded.
func list(dir string) ([]File, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	names, err := f.Readdirnames(maxEntries + 1)
	f.Close()
	if err != nil && len(names) == 0 && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(names) > maxEntries {
		return nil, fmt.Errorf("crash directory %s holds more than %d entries", dir, maxEntries)
	}
	sort.Strings(names)
	var out []File
	for _, name := range names {
		info, err := os.Lstat(filepath.Join(dir, name))
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		out = append(out, File{Name: name, Size: info.Size()})
	}
	return out, nil
}

// Sweep is pair gc's retention for crash files: the same age rule as other
// diagnostics. The diagnosticlog writer protocol does not apply -- the runtime
// writes these once, at death, outside any writer registration. A file whose
// owner is still running is never removed.
func Sweep(dir string, now time.Time, alive func(pid int) bool, apply bool, limit int) ([]diagnosticlog.Segment, error) {
	files, err := list(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rows []diagnosticlog.Segment
	for _, f := range files {
		if len(rows) >= limit {
			break
		}
		_, pid, ext, ok := parseName(f.Name)
		if !ok {
			continue
		}
		path := filepath.Join(dir, f.Name)
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		row := diagnosticlog.Segment{Path: path, Bytes: info.Size(), LastWrite: info.ModTime()}
		switch {
		case ext == unreported && alive(pid):
			row.Reason = "couch that owns it is running"
		case !diagnosticlog.DecideSegment(now, info.ModTime()):
			row.Reason = diagnosticlog.RetainedReason
		default:
			row.Eligible = true
		}
		if row.Eligible && apply {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				row.Eligible, row.Reason = false, err.Error()
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}
