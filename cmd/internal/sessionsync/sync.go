package sessionsync

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Refusal is a gate that stopped an operation before it changed anything.
// Exit and Code are the CLI's exit status and JSON code; they match the
// prototype (tl-status.py) so its callers' handling carries over.
type Refusal struct {
	Exit   int
	Code   string
	Reason string
	// Hash is the current hash on a `stale` refusal.
	Hash string
}

func (r *Refusal) Error() string { return r.Reason }

const (
	ExitRefused      = 1
	ExitOperatorLock = 3
	// ExitStale: the section changed since the agent read it. (4 was
	// unread-edits, retired by pair#439; not reused.)
	ExitStale = 6
)

func refuse(format string, args ...any) error {
	return &Refusal{Exit: ExitRefused, Code: "refused", Reason: fmt.Sprintf(format, args...)}
}

// Status is what `status` reports; the JSON shape is agreed with
// parley.nvim#314.
type Status struct {
	File      string
	Owner     string
	Holder    Holder
	LastWrite *time.Time
	Owed      []OwedEvent
}

// Accepts are the agent's acknowledgements.
type Accepts struct {
	// Gap: the agent has checked its messages past a ledger gap.
	Gap bool
}

// WriteRequest replaces one section (Section != "") or the whole file, if
// that section (or file) still has IfHash, the hash `read` gave.
type WriteRequest struct {
	File, Section, Body, IfHash string
	Accepts
	Fold []string
}

// Result reports what a write did beyond the write itself.
type Result struct {
	Folded []string
	// Hash is the written section's (or file's) new hash, for a next write.
	Hash string
	// Warning is set when the operator took the lock while the write was in
	// flight: the write landed, and their buffer will reload under them.
	Warning string
}

// Sync runs the guarded operations. File paths are absolute.
type Sync struct {
	Ledger    *Ledger
	Now       func() time.Time
	Diff      func(shadow, file string) (string, error)
	Canonical func(string) string
}

// New wires the real clock, `diff -u` and symlink resolution.
func New(storeDir string) *Sync {
	return &Sync{Ledger: &Ledger{Dir: LedgerDir(storeDir)}, Now: time.Now, Diff: UnifiedDiff, Canonical: Canonical}
}

// Canonical resolves symlinks so two spellings of one file compare equal;
// a path that does not resolve compares as given.
func Canonical(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return filepath.Clean(p)
}

// UnifiedDiff is `diff -u` with stable labels; exit 1 means "differs".
func UnifiedDiff(shadow, file string) (string, error) {
	out, err := exec.Command("diff", "-u", "--label", "shadow", "--label", "file", shadow, file).Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		err = nil
	}
	return string(out), err
}

// file is one read of a session-sync file and its companions.
type file struct {
	paths   Paths
	content string
	shadow  string
	holder  Holder
	owner   string
	state   State
}

func readOptional(p string) (string, bool, error) {
	raw, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	return string(raw), err == nil, err
}

// locked runs fn on a fresh read of path under couch's exclusive flock, so
// no other couch command lands between fn's checks and its writes. parley
// takes no flock; its edits are fenced by the operator's .lock instead.
func locked[T any](path string, fn func(*file) (T, error)) (T, error) {
	var zero T
	fl, err := openFlock(path)
	if err != nil {
		return zero, err
	}
	defer fl.Close()
	if err := syscall.Flock(int(fl.Fd()), syscall.LOCK_EX); err != nil {
		return zero, err
	}
	f, err := load(path)
	if err != nil {
		return zero, err
	}
	return fn(f)
}

// openFlock opens path's flock file, creating it only for a valid
// session-sync file so a wrong path leaves nothing behind. Only the read
// under the lock judges the file: a writer rewrites it in place (truncate,
// then write), so an unlocked read can catch it torn, and that writer holds
// the flock file, which is never removed (pair#444).
func openFlock(path string) (*os.File, error) {
	p := PathsOf(path).Flock
	fl, err := os.OpenFile(p, os.O_RDWR, 0)
	if !errors.Is(err, os.ErrNotExist) {
		return fl, err
	}
	if _, err := load(path); err != nil {
		// A writer that created the flock file since may be mid-write.
		if fl, ferr := os.OpenFile(p, os.O_RDWR, 0); ferr == nil {
			return fl, nil
		}
		return nil, err
	}
	return os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0o600)
}

func load(path string) (*file, error) {
	f := &file{paths: PathsOf(path)}
	content, ok, err := readOptional(path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, refuse("%s does not exist", path)
	}
	if !IsSessionSync(content) {
		return nil, refuse("%s is not a session-sync file (frontmatter `type: session-sync`)", path)
	}
	f.content = content
	if f.shadow, _, err = readOptional(f.paths.Shadow); err != nil {
		return nil, err
	}
	lock, exists, err := readOptional(f.paths.Lock)
	if err != nil {
		return nil, err
	}
	f.holder = ParseHolder(lock, exists)
	f.owner = ParseOwner(content)
	if f.state, err = LoadState(f.paths); err != nil {
		return nil, fmt.Errorf("reading %s: %w", f.paths.State, err)
	}
	return f, nil
}

// guard is the human's pessimistic lock: the operator's lock blocks every
// write.
func (f *file) guard() error {
	if f.holder == HolderOperator {
		return &Refusal{Exit: ExitOperatorLock, Code: "operator-lock",
			Reason: "the operator holds the lock; queue the update until their submit"}
	}
	return nil
}

// target is the section's body (or the whole file) that a hash names.
func (f *file) target(section string) (string, error) {
	if section == "" {
		return f.content, nil
	}
	body, err := SectionBody(f.content, section)
	if err != nil {
		return "", refuse("%v", err)
	}
	return body, nil
}

// ReadResult is one section (or the whole file) and the hash a write
// must name.
type ReadResult struct {
	File, Section, Hash string
	Holder              Holder
	Body                string
}

// Read returns a section (Section != "") or the whole file with its hash.
func (s *Sync) Read(path, section string) (ReadResult, error) {
	return locked(path, func(f *file) (ReadResult, error) {
		body, err := f.target(section)
		if err != nil {
			return ReadResult{}, err
		}
		return ReadResult{File: path, Section: section, Hash: Hash(body), Holder: f.holder, Body: body}, nil
	})
}

// owed reads the owner's ledger rows and what this file owes from them. A
// ledger gap reaching past the file's horizon refuses until the agent
// accepts it: then the owed list is incomplete, never silently short.
func (s *Sync) owed(f *file, acceptGap bool) ([]Entry, []OwedEvent, error) {
	if !ValidSlot(f.owner) {
		return nil, nil, refuse("frontmatter owner %q (`couch-owner:`, or legacy `owner:`) is not a Couch slot address (repo:N, as `couch --actors` lists it)", f.owner)
	}
	gap, err := s.Ledger.Gap(f.owner)
	if err != nil {
		return nil, nil, err
	}
	if gap != nil && !gap.Last.Before(f.state.Since) && (f.state.GapAccepted == nil || gap.Last.After(*f.state.GapAccepted)) {
		if !acceptGap {
			return nil, nil, &Refusal{Exit: ExitRefused, Code: "delivery-gap", Reason: fmt.Sprintf(
				"Couch failed to record a delivery to %s at %s (%s), so owed events may be incomplete; check your recent messages, then pass --accept-gap to write",
				f.owner, gap.Last.Local().Format(time.DateTime), gap.Cause)}
		}
		f.state.GapAccepted = &gap.Last
	}
	rows, err := s.Ledger.Read(f.owner)
	if err != nil {
		return nil, nil, err
	}
	return rows, Owed(rows, f.state, f.paths.File, s.Canonical), nil
}

// folds validates ids against what is owed: folding something not owed is a
// typo or a stale ID, never a no-op to wave through.
func folds(ids []string, owed []OwedEvent) error {
	open := map[string]bool{}
	for _, e := range owed {
		open[e.ID] = true
	}
	for _, id := range ids {
		if !open[id] {
			return refuse("--fold %s: no such owed event (see `couch --session-sync status`)", id)
		}
	}
	return nil
}

// Status reports the lock, the last write and the owed events. It refuses
// when Couch is not recording deliveries, so "nothing owed" is never a
// blind spot.
func (s *Sync) Status(path string) (Status, error) {
	return locked(path, func(f *file) (Status, error) {
		if err := s.Ledger.Recording(); err != nil {
			return Status{}, err
		}
		_, owed, err := s.owed(f, false)
		if err != nil {
			return Status{}, err
		}
		return Status{File: path, Owner: f.owner, Holder: f.holder, LastWrite: f.state.LastWrite, Owed: owed}, nil
	})
}

// DiffText is the operator's edits since the agent's last write. It is
// information, not a gate: a write checks only its own section's hash.
func (s *Sync) DiffText(path string) (Holder, string, error) {
	var holder Holder
	out, err := locked(path, func(f *file) (string, error) {
		holder = f.holder
		if f.content == f.shadow {
			return "", nil
		}
		shadow := f.paths.Shadow
		if _, err := os.Stat(shadow); err != nil {
			shadow = os.DevNull
		}
		return s.Diff(shadow, path)
	})
	return holder, out, err
}

// Write replaces a section or the whole file if it still has the hash the
// agent read, restamps it, refreshes the shadow, and folds the given owed
// events. Edits elsewhere in the file don't block a section write.
func (s *Sync) Write(req WriteRequest) (Result, error) {
	if strings.TrimSpace(req.Body) == "" {
		return Result{}, refuse("the body is empty")
	}
	if req.IfHash == "" {
		return Result{}, refuse("write needs --if-hash: `couch --session-sync read` the section first and pass its hash")
	}
	return locked(req.File, func(f *file) (Result, error) { return s.write(f, req) })
}

func (s *Sync) write(f *file, req WriteRequest) (Result, error) {
	if err := f.guard(); err != nil {
		return Result{}, err
	}
	current, err := f.target(req.Section)
	if err != nil {
		return Result{}, err
	}
	if h := Hash(current); h != req.IfHash {
		return Result{}, &Refusal{Exit: ExitStale, Code: "stale", Hash: h, Reason: fmt.Sprintf(
			"it changed since you read it (hash now %s, you passed %s); read it again, recompute, and retry", h, req.IfHash)}
	}
	rows, owed, err := s.owed(f, req.Gap)
	if err != nil {
		return Result{}, err
	}
	if err := folds(req.Fold, owed); err != nil {
		return Result{}, err
	}
	next := req.Body
	if req.Section != "" {
		if next, err = ReplaceSection(f.content, req.Section, req.Body); err != nil {
			return Result{}, refuse("%v", err)
		}
	}
	if !IsSessionSync(next) || ParseOwner(next) != f.owner {
		return Result{}, refuse("the new content must keep the frontmatter (`type: session-sync` and owner %q, as `couch-owner:` or legacy `owner:`)", f.owner)
	}
	now := s.Now()
	next = Restamp(next, now)
	// In place, not rename: parley watches this file's inode.
	if err := os.WriteFile(req.File, []byte(next), 0o644); err != nil {
		return Result{}, err
	}
	if err := writeAtomic(f.paths.Shadow, []byte(next)); err != nil {
		return Result{}, err
	}
	f.state.LastWrite = &now
	f.state.Fold(req.Fold, rows)
	if err := saveState(f.paths, f.state); err != nil {
		return Result{}, err
	}
	f.content = next
	written, err := f.target(req.Section)
	if err != nil {
		return Result{}, err
	}
	res := Result{Folded: req.Fold, Hash: Hash(written)}
	// The guard and the write can't be one step against parley, which takes
	// no flock; say so when the operator took their turn in between.
	if lock, exists, _ := readOptional(f.paths.Lock); ParseHolder(lock, exists) == HolderOperator {
		res.Warning = "the operator took the lock during this write; their buffer reloads with it"
	}
	return res, nil
}

// Recording is the daemon's marker that it records deliveries: a Couch
// that predates the ledger, or none at all, writes no rows, which must not
// read as "nothing owed".
type Recording struct {
	PID     int
	Started time.Time
}

func (l *Ledger) recordingPath() string { return filepath.Join(l.Dir, "recording.json") }

// MarkRecording is written by Couch's message service when it starts.
func (l *Ledger) MarkRecording(r Recording) error {
	if err := os.MkdirAll(l.Dir, 0o700); err != nil {
		return err
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return writeAtomic(l.recordingPath(), raw)
}

// Recording refuses unless a Couch has marked this store as recording. A
// failed append is a per-slot Gap, not a store-wide state.
func (l *Ledger) Recording() error {
	if _, err := os.Stat(l.recordingPath()); err != nil {
		return &Refusal{Exit: ExitRefused, Code: "not-recording",
			Reason: "this Couch store has no delivery ledger yet; restart Couch on a build with pair#437 so owed events are recorded"}
	}
	return nil
}
