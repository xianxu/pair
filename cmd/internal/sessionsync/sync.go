package sessionsync

import (
	"context"
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

// WriteRequest replaces frontmatter, one section (Section != ""), or the
// whole file, if that scope still has IfHash, the hash `read` gave.
type WriteRequest struct {
	File, Section, Body, IfHash string
	Frontmatter                 bool
	Accepts
	Fold []string
}

// Result reports what a write did beyond the write itself.
type Result struct {
	// Owner is the owner observed under the successful write guard.
	Owner  string
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
	Registry  *registryStore
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "diff", "-u", "--label", "shadow", "--label", "file", shadow, file).Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		err = nil
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	return string(out), err
}

// file is one read of a session-sync file and its companions.
type file struct {
	registration *DocumentRegistration
	lockText     string
	lockPaths    []string
	paths        Paths
	content      string
	// shadow is the file as the agent last saw it: what it wrote, plus
	// what it read whole. hasShadow is false before the first write.
	shadow    string
	hasShadow bool
	holder    Holder
	owner     string
	state     State
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
	fl, err := openFlock(path, func() error { _, err := load(path); return err })
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

// openFlock opens path's flock file, creating it only when check (the
// unlocked validation) passes, so a wrong path leaves nothing behind. Only
// the read under the lock judges the file: a couch writer rewrites it in
// place (truncate, then write), so an unlocked read can catch it torn, and
// that writer holds the flock file, which is never removed (pair#444).
func openFlock(path string, check func() error) (*os.File, error) {
	p := PathsOf(path).Flock
	fl, err := os.OpenFile(p, os.O_RDWR, 0)
	if !errors.Is(err, os.ErrNotExist) {
		return fl, err
	}
	if err := check(); err != nil {
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
	if f.shadow, f.hasShadow, err = readOptional(f.paths.Shadow); err != nil {
		return nil, err
	}
	lock, exists, err := readOptional(f.paths.Lock)
	if err != nil {
		return nil, err
	}
	f.holder = ParseHolder(lock, exists)
	f.lockText = lock
	f.owner = ParseOwner(content)
	if f.state, err = LoadState(f.paths); err != nil {
		return nil, fmt.Errorf("reading %s: %w", f.paths.State, err)
	}
	return f, nil
}

// guard is the human's pessimistic lock: the operator's lock blocks every
// write.
func (f *file) guard(now time.Time) error {
	if f.holder == HolderOperator {
		reason := "the operator holds the lock; queue the update until their submit"
		if age, ok := OperatorLockAge(f.lockText, now); ok && age > 30*time.Minute {
			reason = fmt.Sprintf("the operator holds the lock (age %s, more than 30 minutes); ping the operator to submit or release it, then queue the update", age.Round(time.Second))
		}
		return &Refusal{Exit: ExitOperatorLock, Code: "operator-lock",
			Reason: reason}
	}
	return nil
}

// target is the section's body (or the whole file) that a hash names.
func (f *file) target(section string, frontmatter bool) (string, error) {
	if frontmatter {
		body, err := FrontmatterBody(f.content)
		if err != nil {
			return "", refuse("%v", err)
		}
		return body, nil
	}
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
// Reading the whole file is the agent seeing every edit in it, so it
// refreshes the shadow and `diff` goes empty (pair#448).
func (s *Sync) Read(path, section string) (ReadResult, error) {
	return s.read(path, section, false)
}

// ReadFrontmatter acknowledges only the leading block, retaining unread body edits.
func (s *Sync) ReadFrontmatter(path string) (ReadResult, error) {
	return s.read(path, "", true)
}

func (s *Sync) read(path, section string, frontmatter bool) (ReadResult, error) {
	return withDocument(s, path, func(f *file) (ReadResult, error) {
		body, err := f.target(section, frontmatter)
		if err != nil {
			return ReadResult{}, err
		}
		if frontmatter {
			shadow, err := f.frontmatterShadow(body)
			if err != nil {
				return ReadResult{}, err
			}
			if !f.hasShadow || shadow != f.shadow {
				if err := f.saveShadow(shadow); err != nil {
					return ReadResult{}, err
				}
			}
		} else if section == "" && (!f.hasShadow || f.shadow != f.content) {
			if err := f.saveShadow(f.content); err != nil {
				return ReadResult{}, err
			}
		}
		return ReadResult{File: f.paths.File, Section: section, Hash: Hash(body), Holder: f.holder, Body: body}, nil
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
	paths := f.state.SubmitPaths
	if f.registration != nil {
		paths = f.registration.Paths
	}
	// Plain deliveries are owed by the owner's session file only (pair#599).
	return rows, OwedForDocument(rows, f.state, f.paths.File, paths, IsSessionSync(f.content), s.Canonical), nil
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
	return withDocument(s, path, func(f *file) (Status, error) {
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

// DiffText is the operator's edits the agent hasn't seen. It is
// information, not a gate: a write checks only its own section's hash.
func (s *Sync) DiffText(path string) (Holder, string, error) {
	var holder Holder
	out, err := withDocument(s, path, func(f *file) (string, error) {
		holder = f.holder
		if f.content == f.shadow {
			return "", nil
		}
		shadow := f.paths.Shadow
		if _, err := os.Stat(shadow); err != nil {
			shadow = os.DevNull
		}
		return s.Diff(shadow, f.paths.File)
	})
	return holder, out, err
}

// Write replaces its target if it still has the hash the agent read,
// restamps section/whole-file writes, records its scoped shadow, and folds
// the given owed events. Edits elsewhere in the file don't block a section
// write, and stay in `diff` until the agent reads or writes them.
func (s *Sync) Write(req WriteRequest) (Result, error) {
	if req.Frontmatter && req.Section != "" {
		return Result{}, refuse("frontmatter and section scopes are mutually exclusive")
	}
	if strings.TrimSpace(req.Body) == "" {
		return Result{}, refuse("the body is empty")
	}
	if req.IfHash == "" {
		return Result{}, refuse("write needs --if-hash: `couch --session-sync read` the section first and pass its hash")
	}
	return withDocument(s, req.File, func(f *file) (Result, error) { req.File = f.paths.File; return s.write(f, req) })
}

// shadowAfter is the shadow once the agent has written next: a whole-file
// write is all of it, a section write changes only that section (and the
// stamp it restamped), so the operator's edits elsewhere stay unread in
// `diff` (pair#448). Without a shadow there is no baseline to keep. A shadow
// without that section (the operator added it) stays as it is: `diff` then
// shows the agent's own write too, which is noise, never a lost edit.
func (f *file) shadowAfter(section, body, next string, now time.Time) string {
	if section == "" || !f.hasShadow {
		return next
	}
	shadow, err := ReplaceSection(f.shadow, section, body)
	if err != nil {
		return f.shadow
	}
	return Restamp(shadow, now)
}

// frontmatterShadow advances only the block, even before the first baseline.
func (f *file) frontmatterShadow(block string) (string, error) {
	if !f.hasShadow {
		return block, nil
	}
	first, _, _ := strings.Cut(f.shadow, "\n")
	if strings.TrimSpace(first) != "---" {
		return block + "\n" + f.shadow, nil
	}
	shadow, err := ReplaceFrontmatter(f.shadow, block)
	if err != nil {
		return "", refuse("shadow frontmatter: %v", err)
	}
	return shadow, nil
}

// saveShadow prepares a read baseline before publishing it. Writes use
// their prepared bundle, including already-encoded horizon and fold state.
func (f *file) saveShadow(content string) error {
	if f.registration != nil {
		if err := validateRegisteredMetadata(content, f.state, f.registration.Paths); err != nil {
			return err
		}
	}
	pinState, err := encodeState(f.state)
	if err != nil {
		return err
	}
	return f.publishShadow(content, pinState)
}

// publishShadow is the one shadow writer. Preserve a sidecar-less legacy
// horizon before changing shadow mtime; every byte was prepared before effects.
func (f *file) publishShadow(content string, pinState []byte) error {
	if _, err := os.Stat(f.paths.State); errors.Is(err, os.ErrNotExist) {
		if err := writeAtomic(f.paths.State, pinState); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err := writeAtomic(f.paths.Shadow, []byte(content)); err != nil {
		return err
	}
	f.shadow, f.hasShadow = content, true
	return nil
}

// preparedWrite is the complete immutable publication input. Preparation owns
// all deterministic refusals; effects never re-validate an older mutable state.
type preparedWrite struct {
	document, shadow, hash    string
	state                     State
	stateBytes, pinStateBytes []byte
}

func (f *file) prepareWrite(req WriteRequest, rows []Entry, now time.Time) (preparedWrite, error) {
	next := req.Body
	var err error
	if req.Frontmatter {
		if next, err = ReplaceFrontmatter(f.content, req.Body); err != nil {
			return preparedWrite{}, refuse("%v", err)
		}
	} else if req.Section != "" {
		if next, err = ReplaceSection(f.content, req.Section, req.Body); err != nil {
			return preparedWrite{}, refuse("%v", err)
		}
	}
	validContent := IsSessionSync(next) && ParseOwner(next) == f.owner
	if f.registration != nil {
		validContent = EffectiveOwner(next, f.registration.RegisteredOwner) == f.owner && (!IsSessionSync(f.content) || IsSessionSync(next))
	}
	if !validContent {
		if f.registration != nil && !IsSessionSync(f.content) {
			return preparedWrite{}, refuse("the new content must preserve the effective Couch owner %q", f.owner)
		}
		return preparedWrite{}, refuse("the new content must keep the frontmatter (`type: session-sync` and owner %q, as `couch-owner:` or legacy `owner:`)", f.owner)
	}
	if !req.Frontmatter {
		next = Restamp(next, now)
	}
	shadow := f.shadowAfter(req.Section, req.Body, next, now)
	if req.Frontmatter {
		block, _ := FrontmatterBody(next) // replacement already validated the block
		if shadow, err = f.frontmatterShadow(block); err != nil {
			return preparedWrite{}, err
		}
	}
	state := f.state
	state.LastWrite = &now
	state.Fold(req.Fold, rows)
	if f.registration != nil {
		if len(next) > documentMaxBytes {
			return preparedWrite{}, registryLimit()
		}
		if err := validateRegisteredMetadata(shadow, state, f.registration.Paths); err != nil {
			return preparedWrite{}, err
		}
	}
	target := next
	if req.Frontmatter {
		target, err = FrontmatterBody(next)
		if err != nil {
			return preparedWrite{}, refuse("%v", err)
		}
	} else if req.Section != "" {
		if target, err = SectionBody(next, req.Section); err != nil {
			return preparedWrite{}, refuse("%v", err)
		}
	}
	stateBytes, err := encodeState(state)
	if err != nil {
		return preparedWrite{}, err
	}
	pinStateBytes, err := encodeState(f.state)
	if err != nil {
		return preparedWrite{}, err
	}
	return preparedWrite{document: next, shadow: shadow, hash: Hash(target), state: state, stateBytes: stateBytes, pinStateBytes: pinStateBytes}, nil
}

func (s *Sync) write(f *file, req WriteRequest) (Result, error) {
	if err := f.guard(s.now()); err != nil {
		return Result{}, err
	}
	current, err := f.target(req.Section, req.Frontmatter)
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
	prepared, err := f.prepareWrite(req, rows, s.now())
	if err != nil {
		return Result{}, err
	}
	if err := replaceShared(req.File, []byte(prepared.document)); err != nil {
		return Result{}, err
	}
	if err := f.publishShadow(prepared.shadow, prepared.pinStateBytes); err != nil {
		return Result{}, err
	}
	if err := writeAtomic(f.paths.State, prepared.stateBytes); err != nil {
		return Result{}, err
	}
	f.content, f.state = prepared.document, prepared.state
	res := Result{Owner: f.owner, Folded: req.Fold, Hash: prepared.hash}
	// The guard and the write can't be one step against parley, which takes
	// no flock; say so when the operator took their turn in between.
	lockPaths := f.lockPaths
	if len(lockPaths) == 0 {
		lockPaths = []string{f.paths.Lock}
	}
	for _, path := range lockPaths {
		var lock string
		var exists bool
		var lockErr error
		if f.registration != nil {
			raw, err := readRegular(path, 4096)
			lock = string(raw)
			exists = err == nil
			lockErr = err
		} else {
			lock, exists, lockErr = readOptional(path)
		}
		if lockErr != nil && !errors.Is(lockErr, os.ErrNotExist) {
			res.Warning = "the operator lock could not be inspected after this write; check the editor before continuing"
			break
		}
		if ParseHolder(lock, exists) == HolderOperator {
			res.Warning = "the operator took the lock during this write; their buffer reloads with it"
			break
		}
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
