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
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

const (
	unreported = ".log"
	reported   = ".crash"
	// exitMarker opens a file this process wrote itself with RecordExit: a
	// deliberate exit's reason, not a runtime panic (pair#409).
	exitMarker = "couch-exit: "
	// headBytes is how much of a file Classify sees: enough for the marker and
	// a one-line reason.
	headBytes = 256
	stampForm = "20060102T150405Z"
	// MaxEntries bounds the startup scan and the gc walk: one file per console
	// run, so a directory this large is debris, not history.
	MaxEntries = 4096
)

// Dir is the crash directory inside one couch store namespace. One store per
// singleton selection, so isolated roots keep their own crashes.
func Dir(store string) string { return filepath.Join(store, "crash") }

// ProcessAlive is the liveness the startup scan and gc both ask about a
// crash file's pid.
func ProcessAlive(pid int) bool { return procutil.Alive(strconv.Itoa(pid)) }

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
	// Exited: the process recorded why it chose to exit (RecordExit), e.g. the
	// terminal stopped accepting output.
	Exited
)

type File struct {
	Name string
	Size int64
	// Head is the start of a non-empty unreported file (at most headBytes).
	Head string
}

type Finding struct {
	Name string
	Kind Kind
	// Reason is an Exited file's recorded reason.
	Reason string
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
			out = append(out, Finding{Name: f.Name, Kind: Reported})
		case f.Size > 0 && strings.HasPrefix(f.Head, exitMarker):
			reason, _, _ := strings.Cut(strings.TrimPrefix(f.Head, exitMarker), "\n")
			out = append(out, Finding{Name: f.Name, Kind: Exited, Reason: reason})
		case f.Size > 0:
			out = append(out, Finding{Name: f.Name, Kind: Crashed})
		default:
			out = append(out, Finding{Name: f.Name, Kind: Abrupt})
		}
	}
	return out
}

// Report is one previous ending to tell the operator about.
type Report struct {
	Kind Kind
	Path string
	// Reason is an Exited report's recorded reason.
	Reason string
}

// Summary folds every previous ending into ONE status-row sentence: each
// standing control notice holds the row until displaced, so N reports would
// stack N notices.
func Summary(reports []Report) string {
	var latest, latestExit, latestExitPath string
	crashes, exits, abrupt := 0, 0, 0
	for _, r := range reports {
		switch r.Kind {
		case Crashed:
			crashes++
			latest = r.Path // names sort by timestamp: the last is the newest
		case Exited:
			exits++
			latestExit, latestExitPath = r.Reason, r.Path
		case Abrupt:
			abrupt++
		}
	}
	var parts []string
	if exits > 0 {
		// The path is named too: anything written after the reason (a panic
		// during the exit) is in the same file.
		s := "previous couch exited: " + latestExit
		if latestExitPath != "" {
			s += " — see " + latestExitPath
		}
		if exits > 1 {
			s += fmt.Sprintf(" (+%d earlier)", exits-1)
		}
		parts = append(parts, s)
	}
	if crashes > 0 {
		s := "previous couch crashed — see " + latest
		if crashes > 1 {
			s += fmt.Sprintf(" (+%d earlier)", crashes-1)
		}
		parts = append(parts, s)
	}
	switch {
	case abrupt == 1:
		parts = append(parts, "previous couch ended abruptly (no panic recorded)")
	case abrupt > 1:
		parts = append(parts, fmt.Sprintf("%d previous couch runs ended abruptly (no panic recorded)", abrupt))
	}
	return strings.Join(parts, "; ")
}

type Capture struct {
	file *os.File
}

var (
	activeMu sync.Mutex
	active   *Capture
)

// Install reports the previous incarnations' files, then routes this
// process's crash output to a fresh file.
//
// The caller must hold couch's singleton lease, which proves no other owner is
// live. A dying owner still holds its pid while it writes its panic -- its lease
// is released by a defer before the runtime prints -- so a .log whose pid is
// alive is left alone rather than misread as an abrupt end.
//
// A file that cannot be reported is an error alongside a working capture, not
// instead of one: one stuck stale file must not disable capture for good.
func Install(dir string, now time.Time, pid int, alive func(pid int) bool) (*Capture, []Report, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, nil, err
	}
	files, err := list(dir)
	var reports []Report
	var problems []error
	if err != nil {
		problems = append(problems, err)
	}
	for _, f := range Classify(files) {
		path := filepath.Join(dir, f.Name)
		if _, owner, _, _ := parseName(f.Name); f.Kind != Reported && alive(owner) {
			continue
		}
		switch f.Kind {
		case Crashed, Exited:
			// Renamed first, reported second: a report whose rename failed
			// would come back on every start.
			next := strings.TrimSuffix(path, unreported) + reported
			if err := os.Rename(path, next); err != nil {
				problems = append(problems, err)
				continue
			}
			reports = append(reports, Report{Kind: f.Kind, Path: next, Reason: f.Reason})
		case Abrupt:
			if err := os.Remove(path); err != nil {
				problems = append(problems, err)
				continue
			}
			reports = append(reports, Report{Kind: Abrupt, Path: path})
		}
	}
	name := fmt.Sprintf("%s-%d%s", now.UTC().Format(stampForm), pid, unreported)
	file, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, reports, errors.Join(append(problems, err)...)
	}
	if err := debug.SetCrashOutput(file, debug.CrashOptions{}); err != nil {
		file.Close()
		os.Remove(file.Name())
		return nil, reports, errors.Join(append(problems, err)...)
	}
	capture := &Capture{file: file}
	activeMu.Lock()
	previous := active
	active = capture
	activeMu.Unlock()
	// One capture per process: SetCrashOutput already replaced the old one's
	// fd, so the old file only needs its normal end.
	if previous != nil && previous.file != nil {
		previous.file.Close()
		if info, err := os.Stat(previous.file.Name()); err == nil && info.Size() == 0 {
			os.Remove(previous.file.Name())
		}
		previous.file = nil
	}
	return capture, reports, errors.Join(problems...)
}

// RecordExit writes why this process is about to exit into its crash file,
// so the next start reports it once, as it would a panic (pair#409). A run
// that ends silently on purpose -- the terminal stopped accepting output --
// otherwise leaves nothing. With no capture installed it does nothing. Finish
// keeps the file: only an empty one is removed.
func RecordExit(reason string) error {
	activeMu.Lock()
	defer activeMu.Unlock()
	if active == nil || active.file == nil {
		return nil
	}
	line := exitMarker + strings.ReplaceAll(reason, "\n", " ") + "\n"
	if _, err := active.file.WriteString(line); err != nil {
		return err
	}
	return active.file.Sync()
}

// Finish ends the installed capture after a NORMAL return from the program's
// run, and only there. It must never run from a defer: Go runs defers while a
// panic unwinds, BEFORE the runtime writes the panic, so a deferred Finish
// would switch crash output off and delete the file the panic was about to
// fill (#397 BR-1). The program's main calls it after Run returns; a panic
// never reaches that line.
func Finish() error {
	activeMu.Lock()
	capture := active
	active = nil
	activeMu.Unlock()
	return capture.Close()
}

// Close ends capture on a clean exit. An empty file is removed: only a file
// that survives its process says something. Same rule as Finish: never from a
// defer.
func (c *Capture) Close() error {
	if c == nil || c.file == nil {
		return nil
	}
	activeMu.Lock()
	if active == c {
		active = nil
	}
	activeMu.Unlock()
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
	names, err := f.Readdirnames(MaxEntries + 1)
	f.Close()
	if err != nil && len(names) == 0 && !errors.Is(err, io.EOF) {
		return nil, err
	}
	// Over the bound, work on the first MaxEntries and say so, rather than
	// refuse: a refusal would leave gc unable ever to drain the directory.
	var overflow error
	if len(names) > MaxEntries {
		names = names[:MaxEntries]
		overflow = fmt.Errorf("crash directory %s holds more than %d entries", dir, MaxEntries)
	}
	sort.Strings(names)
	var out []File
	for _, name := range names {
		info, err := os.Lstat(filepath.Join(dir, name))
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		file := File{Name: name, Size: info.Size()}
		if filepath.Ext(name) == unreported && info.Size() > 0 {
			file.Head = readHead(filepath.Join(dir, name))
		}
		out = append(out, file)
	}
	return out, overflow
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
	if err != nil && files == nil {
		return nil, err
	}
	var rows []diagnosticlog.Segment
	if err != nil {
		rows = append(rows, diagnosticlog.Segment{Path: dir, Reason: err.Error()})
	}
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

// readHead is the start of a file, at most headBytes; unreadable reads as
// empty, which classifies as a crash rather than hiding the file.
func readHead(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	buf := make([]byte, headBytes)
	n, _ := io.ReadFull(f, buf)
	return string(buf[:n])
}
