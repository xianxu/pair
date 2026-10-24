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

	"github.com/xianxu/pair/cmd/internal/editorial"
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
	// Invalid lists every bad record on an `edits-invalid` refusal.
	Invalid []BadEdit
}

func (r *Refusal) Error() string { return r.Reason }

const (
	ExitRefused      = 1
	ExitOperatorLock = 3
	// ExitStale: the section changed since the agent read it. (4 was
	// unread-edits, retired by pair#439; not reused.)
	ExitStale = 6
	// ExitUnavailable is namespace/store access unavailability for operator
	// commands, distinct from ownership, authority and document-path refusals.
	ExitUnavailable = 7
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
	// edits, set by Apply, are resolved against the file under the flock and
	// become the whole-file body.
	edits    []EditRecord
	resolved []ResolvedEdit
}

// ApplyRequest is `apply`: records resolved against the whole file, all or
// nothing (pair#661). IfHash is the whole-file hash from `read --all`.
type ApplyRequest struct {
	File, IfHash string
	Edits        []EditRecord
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
	// RoundWriter publishes a round sidecar; nil is writeAtomic.
	RoundWriter func(path string, raw []byte) error
}

func (s *Sync) writeRound(path string, raw []byte) error {
	if s.RoundWriter != nil {
		return s.RoundWriter(path, raw)
	}
	return writeAtomic(path, raw)
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
	policy    *editorial.DocumentPolicy
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
	File, Section, Hash  string
	Holder               Holder
	Body                 string
	Editorial            *editorial.DocumentPolicy
	EditorialCoordinates string
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
		policy, err := f.editorialPolicy()
		if err != nil {
			return ReadResult{}, err
		}
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
		hash, err := targetPolicyHash(f.content, section, frontmatter, policy)
		if err != nil {
			return ReadResult{}, err
		}
		return ReadResult{File: f.paths.File, Section: section, Hash: hash, Holder: f.holder, Body: body, Editorial: policy, EditorialCoordinates: "document-byte-offsets"}, nil
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

// RoundResult is `round --json`'s answer: the round sidecar, nil when there
// is none, and whether a submit of the file was recorded after it.
type RoundResult struct {
	Round   *Round
	Cleared bool
}

// Round reads the round sidecar and whether the next submit cleared it.
// Cleared reads the raw ledger rows, so a folded submit still clears.
func (s *Sync) Round(path string) (RoundResult, error) {
	return withDocument(s, path, func(f *file) (RoundResult, error) {
		if err := s.Ledger.Recording(); err != nil {
			return RoundResult{}, err
		}
		raw, err := readRound(f.paths.Round)
		if err != nil || raw == nil {
			return RoundResult{}, err
		}
		round, err := decodeRound(raw)
		if err != nil {
			return RoundResult{}, err
		}
		rows, err := s.Ledger.Read(f.owner)
		if err != nil {
			return RoundResult{}, err
		}
		paths := f.state.SubmitPaths
		if f.registration != nil {
			paths = f.registration.Paths
		}
		return RoundResult{Round: round, Cleared: SubmittedSince(rows, f.paths.File, paths, round.At, s.Canonical)}, nil
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

// Apply makes a precise edit: the write path with a record-built body,
// resolved after the hash and owed checks, so a stale file refuses before any
// record is looked at.
func (s *Sync) Apply(req ApplyRequest) (Result, error) {
	if req.IfHash == "" {
		return Result{}, refuse("apply needs --if-hash: `couch --session-sync read --all` the file first and pass its hash")
	}
	if len(req.Edits) == 0 {
		return Result{}, refuse("apply needs at least one record")
	}
	edits := req.Edits
	return withDocument(s, req.File, func(f *file) (Result, error) {
		return s.write(f, WriteRequest{File: f.paths.File, IfHash: req.IfHash, Accepts: req.Accepts, Fold: req.Fold, edits: edits})
	})
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
	// roundChanged: the write changed the file, so its round replaces the
	// previous one; round is nil when it would pass roundMaxBytes.
	roundChanged bool
	round        []byte
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
	policy, err := f.editorialPolicy()
	if err != nil {
		return preparedWrite{}, err
	}
	content, err := contentStep(f.content, next, req.resolved, now, !req.Frontmatter, policy)
	if err != nil {
		return preparedWrite{}, err
	}
	next = content.Next
	body := req.Body
	if !req.Frontmatter && req.Section != "" {
		// The written bytes (markers) are what the shadow and hash see.
		if body, err = SectionBody(next, req.Section); err != nil {
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
	shadow := f.shadowAfter(req.Section, body, next, now)
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
	nextPolicy, err := compileEditorial(next)
	if err != nil {
		return preparedWrite{}, err
	}
	nextHash, err := targetPolicyHash(next, req.Section, req.Frontmatter, nextPolicy)
	if err != nil {
		return preparedWrite{}, refuse("%v", err)
	}
	stateBytes, err := encodeState(state)
	if err != nil {
		return preparedWrite{}, err
	}
	pinStateBytes, err := encodeState(f.state)
	if err != nil {
		return preparedWrite{}, err
	}
	prepared := preparedWrite{document: next, shadow: shadow, hash: nextHash, state: state, stateBytes: stateBytes, pinStateBytes: pinStateBytes}
	if content.Changed {
		prepared.roundChanged = true
		prepared.round, _ = encodeRound(content.Round)
	}
	return prepared, nil
}

func (s *Sync) write(f *file, req WriteRequest) (Result, error) {
	if err := f.guard(s.now()); err != nil {
		return Result{}, err
	}
	policy, err := f.editorialPolicy()
	if err != nil {
		return Result{}, err
	}
	h, err := targetPolicyHash(f.content, req.Section, req.Frontmatter, policy)
	if err != nil {
		return Result{}, err
	}
	if h != req.IfHash {
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
	if req.edits != nil {
		if req.Body, req.resolved, err = resolveBatch(f.content, req.edits); err != nil {
			return Result{}, err
		}
	}
	prepared, err := f.prepareWrite(req, rows, s.now())
	if err != nil {
		return Result{}, err
	}
	// A changing write drops the previous round before the document moves, so
	// a failure later leaves no round, never an old round over new text.
	if prepared.roundChanged {
		if err := os.Remove(f.paths.Round); err != nil && !errors.Is(err, os.ErrNotExist) {
			return Result{}, refuse("the previous round could not be cleared: %v", err)
		}
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
	if prepared.roundChanged {
		switch {
		case prepared.round == nil:
			res.Warning = "this round is too large to keep; parley shows no highlight for it"
		case s.writeRound(f.paths.Round, prepared.round) != nil:
			res.Warning = "the round could not be saved; parley shows no highlight for this write"
		}
	}
	// Registered operator-acquire shares this writer's registry transaction.
	// Legacy/direct sidecar clients do not; retain their best-effort warning
	// when protection appears after the guard but before this write completes.
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
