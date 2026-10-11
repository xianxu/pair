package sessionsync

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Refusal is a gate that stopped an operation before it changed anything.
// Exit and Code are the CLI's exit status and JSON code; they match the
// prototype (tl-status.py) so its callers' handling carries over.
type Refusal struct {
	Exit   int
	Code   string
	Reason string
}

func (r *Refusal) Error() string { return r.Reason }

const (
	ExitRefused      = 1
	ExitOperatorLock = 3
	ExitUnreadEdits  = 4
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

// Accepts are the agent's acknowledgements, shared by write and release so
// every refusal's remedy is reachable from either verb.
type Accepts struct {
	// Edits: the agent has read the operator's edits (`diff`).
	Edits bool
	// Gap: the agent has checked its messages past a ledger gap.
	Gap bool
}

// WriteRequest replaces one section (Section != "") or the whole file.
type WriteRequest struct {
	File, Section, Body string
	Accepts
	Fold []string
}

// ReleaseRequest hands the turn back.
type ReleaseRequest struct {
	File string
	Accepts
	Fold []string
}

// Result reports what a write or release did beyond the write itself.
type Result struct {
	Folded   []string
	Released bool
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

// guard is the turn rule: the operator's lock blocks every write, and the
// operator's unread edits block until the agent says it has read them.
func (f *file) guard(acceptEdits bool) error {
	if f.holder == HolderOperator {
		return &Refusal{Exit: ExitOperatorLock, Code: "operator-lock",
			Reason: "the operator holds the lock; queue the update until their submit"}
	}
	if f.content != f.shadow && !acceptEdits {
		return &Refusal{Exit: ExitUnreadEdits, Code: "unread-edits",
			Reason: "the file has edits you haven't read; run `couch --session-sync diff`, resolve them, then pass --accept-edits"}
	}
	return nil
}

// owed reads the owner's ledger rows and what this file owes from them. A
// ledger gap reaching past the file's horizon refuses until the agent
// accepts it: then the owed list is incomplete, never silently short.
func (s *Sync) owed(f *file, acceptGap bool) ([]Entry, []OwedEvent, error) {
	if !ValidSlot(f.owner) {
		return nil, nil, refuse("frontmatter `owner: %s` is not a Couch slot address (repo:N, as `couch --actors` lists it)", f.owner)
	}
	gap, err := s.Ledger.Gap(f.owner)
	if err != nil {
		return nil, nil, err
	}
	if gap != nil && !gap.Last.Before(f.state.Since) && (f.state.GapAccepted == nil || gap.Last.After(*f.state.GapAccepted)) {
		if !acceptGap {
			return nil, nil, &Refusal{Exit: ExitRefused, Code: "ledger-gap", Reason: fmt.Sprintf(
				"Couch failed to record a delivery to %s at %s (%s), so owed events may be incomplete; check your recent messages, then pass --accept-gap to write or release",
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
	f, err := load(path)
	if err != nil {
		return Status{}, err
	}
	if err := s.Ledger.Recording(); err != nil {
		return Status{}, err
	}
	_, owed, err := s.owed(f, false)
	if err != nil {
		return Status{}, err
	}
	return Status{File: path, Owner: f.owner, Holder: f.holder, LastWrite: f.state.LastWrite, Owed: owed}, nil
}

// DiffText is the operator's edits since the agent's last write.
func (s *Sync) DiffText(path string) (Holder, string, error) {
	f, err := load(path)
	if err != nil {
		return "", "", err
	}
	if f.content == f.shadow {
		return f.holder, "", nil
	}
	shadow := f.paths.Shadow
	if _, err := os.Stat(shadow); err != nil {
		shadow = os.DevNull
	}
	out, err := s.Diff(shadow, path)
	return f.holder, out, err
}

// Write replaces a section or the whole file, restamps it, refreshes the
// shadow, and folds the given owed events, all behind the guards.
func (s *Sync) Write(req WriteRequest) (Result, error) {
	if strings.TrimSpace(req.Body) == "" {
		return Result{}, refuse("the body is empty")
	}
	f, err := load(req.File)
	if err != nil {
		return Result{}, err
	}
	if err := f.guard(req.Edits); err != nil {
		return Result{}, err
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
		return Result{}, refuse("the new content must keep the frontmatter (`type: session-sync`, `owner: %s`)", f.owner)
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
	res := Result{Folded: req.Fold}
	// The guard and the write can't be one step; say so when the operator
	// took their turn in between.
	if lock, exists, _ := readOptional(f.paths.Lock); ParseHolder(lock, exists) == HolderOperator {
		res.Warning = "the operator took the lock during this write; their buffer reloads with it"
	}
	return res, nil
}

// Release hands the turn back: it deletes an agent-held lock and folds the
// owed submits (handing the turn back answers them) plus the given IDs. It
// keeps the write guards, so an unread edit can't be released past; with
// Accepts.Edits it records the operator's edits as read (the shadow takes
// the file as it is) without a write.
func (s *Sync) Release(req ReleaseRequest) (Result, error) {
	f, err := load(req.File)
	if err != nil {
		return Result{}, err
	}
	if err := f.guard(req.Edits); err != nil {
		return Result{}, err
	}
	rows, owed, err := s.owed(f, req.Gap)
	if err != nil {
		return Result{}, err
	}
	if err := folds(req.Fold, owed); err != nil {
		return Result{}, err
	}
	if f.content != f.shadow {
		if err := writeAtomic(f.paths.Shadow, []byte(f.content)); err != nil {
			return Result{}, err
		}
	}
	all := append([]string{}, req.Fold...)
	for _, e := range owed {
		if e.Kind == KindSubmit {
			all = append(all, e.ID)
		}
	}
	f.state.Fold(all, rows)
	if err := saveState(f.paths, f.state); err != nil {
		return Result{}, err
	}
	res := Result{Folded: all}
	if f.holder == HolderAgent {
		if err := os.Remove(f.paths.Lock); err != nil && !errors.Is(err, os.ErrNotExist) {
			return res, err
		}
		res.Released = true
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
