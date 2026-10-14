package notedraft

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Couch owns one process per capture root; live casts register their recordings
// so recovery never mistakes a peer writer for a crashed owner.
var captureOwnersMu sync.Mutex
var captureOwners = map[string]map[string]*Service{}

const DraftTTL = 10 * time.Minute
const OperationBudget = 36 * time.Second

type Owner struct {
	Identity   string `json:"identity,omitempty"`
	Scope      string `json:"scope"`
	Generation uint64 `json:"generation"`
}
type Evidence struct {
	Page    core.PageIdentity `json:"page"`
	Crop    core.Crop         `json:"crop"`
	BoxText core.BoxText      `json:"box_text"`
}
type CaptureSource interface {
	Capture(context.Context, board.PreparedNote) (Evidence, []byte, error)
}
type Options struct {
	Root, RecordingPath string
	Capture             CaptureSource
	Now                 func() time.Time
	Write               func(context.Context, board.AnnotationAdmission, board.NoteReservation, func() error, func(board.OperatorOutcome, error)) error
	Observe             func(context.Context, string) (*recording.Note, bool, error)
	Release             func(string)
	ObservePublication  func(context.Context, string) (core.Submission, core.PublicationObservation, error)
}
type Status struct {
	State       string    `json:"state"`
	Request     string    `json:"request,omitempty"`
	Draft       string    `json:"draft,omitempty"`
	NoteID      string    `json:"id,omitempty"`
	CapturedAt  time.Time `json:"captured_at,omitempty"`
	Unavailable string    `json:"unavailable,omitempty"`
	Error       string    `json:"error,omitempty"`
}
type captureRecord struct {
	ID                string             `json:"id"`
	Owner             Owner              `json:"owner"`
	Request           string             `json:"request"`
	RecordingPath     string             `json:"recording_path"`
	Prepared          board.PreparedNote `json:"prepared"`
	Operator          bool               `json:"operator,omitempty"`
	State             State              `json:"state"`
	Created           time.Time          `json:"created"`
	Expires           time.Time          `json:"expires"`
	Evidence          Evidence           `json:"evidence"`
	NoteID            string             `json:"note_id,omitempty"`
	SaveRequest       string             `json:"save_request,omitempty"`
	SavePayload       string             `json:"save_payload,omitempty"`
	Problem           string             `json:"problem,omitempty"`
	BundlePath        string             `json:"bundle_path,omitempty"`
	PendingSubmission string             `json:"pending_submission,omitempty"`
	Note              *recording.Note    `json:"note,omitempty"`
	Ephemeral         bool               `json:"-"`
	// Volatile authority and worker cancellation never enter the local metadata.
	admission board.AnnotationAdmission
	cancel    context.CancelFunc
}
type Service struct {
	mu       sync.Mutex
	opts     Options
	disk     *diskStore
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	jobs     chan string
	records  map[string]*captureRecord
	revoked  map[string]time.Time
	key      [32]byte
	closed   bool
	ownerKey string
}

func New(opts Options) (*Service, error) {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if !filepath.IsAbs(opts.RecordingPath) || filepath.Clean(opts.RecordingPath) != opts.RecordingPath {
		return nil, errors.New("recording path must be absolute and clean")
	}
	if opts.Capture == nil {
		return nil, errors.New("capture source required")
	}
	disk, err := newDiskStore(opts.Root)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{opts: opts, disk: disk, ctx: ctx, cancel: cancel, done: make(chan struct{}), jobs: make(chan string, 8), records: map[string]*captureRecord{}, revoked: map[string]time.Time{}}
	if _, err = rand.Read(s.key[:]); err != nil {
		cancel()
		return nil, err
	}
	captureOwnersMu.Lock()
	defer captureOwnersMu.Unlock()
	ownerKey := canonicalRecordingPath(opts.RecordingPath)
	s.ownerKey = ownerKey
	if captureOwners[disk.root] == nil {
		captureOwners[disk.root] = map[string]*Service{}
	}
	if captureOwners[disk.root][ownerKey] != nil {
		cancel()
		return nil, errors.New("recording already has an active capture owner")
	}
	captureOwners[disk.root][ownerKey] = s
	registered := false
	defer func() {
		if !registered {
			delete(captureOwners[disk.root], ownerKey)
		}
	}()
	// Startup recovery never treats abandoned preparations as saved notes.
	ids, err := disk.list()
	if err != nil {
		cancel()
		return nil, err
	}
	for _, id := range ids {
		data, e := disk.metadata(id)
		if e != nil {
			continue
		} // Preserve unreadable evidence; unrelated notes remain usable.
		var r captureRecord
		if e = json.Unmarshal(data, &r); e != nil || validCaptureRecord(&r, id) != nil {
			continue
		}
		if r.RecordingPath != opts.RecordingPath {
			continue
		}
		if r.State.Phase == Preparing || r.State.Phase == Ready || r.State.Phase == Retired {
			if e = disk.remove(id); e != nil {
				cancel()
				return nil, e
			}
			continue
		}
		s.records[id] = &r
	}
	if err = s.recoverReferencesLocked(); err != nil {
		cancel()
		return nil, err
	}
	if err = s.recoverRecording(); err != nil {
		cancel()
		return nil, err
	}
	if err = s.disk.sweepSources(opts.Now()); err != nil {
		cancel()
		return nil, err
	}
	// Unknown publication is retained for the next bounded sweep or request.
	_ = s.ReconcilePending(ctx, opts.ObservePublication)
	_ = s.recoverInactiveLocked(ctx)
	registered = true
	go s.run()
	return s, nil
}

func opaqueID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

func (s *Service) Prepare(ctx context.Context, owner Owner, request string, a board.AnnotationAdmission, p board.PreparedNote, operator bool) (Status, error) {
	if owner.Scope == "" || request == "" || len(request) > 128 {
		return Status{}, errors.New("invalid capture request")
	}
	if err := ctx.Err(); err != nil {
		return Status{}, err
	}
	if err := a.Check(); err != nil {
		return Status{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return Status{}, errors.New("cast has ended")
	}
	if s.isRevoked(owner, request) {
		return Status{State: "cancelled", Request: request}, nil
	}
	for _, r := range s.records {
		if r.Owner == owner && r.Request == request {
			return s.statusLocked(r), nil
		}
	}
	now := s.opts.Now()
	r := &captureRecord{ID: opaqueID(), Owner: owner, Request: request, RecordingPath: s.opts.RecordingPath, Prepared: p, Operator: operator, Created: now, Expires: now.Add(DraftTTL), admission: a}
	r.State, _, _ = Step(State{}, Event{Kind: Begin})
	reason := ""
	if len(s.jobs) == cap(s.jobs) {
		reason = CaptureBusy
	}
	if reason == "" {
		if err := s.disk.sweepSources(now); err != nil {
			reason = StoreFull
		}
	}
	if reason == "" {
		data, err := json.Marshal(r)
		if err != nil {
			return Status{}, err
		}
		if err = s.disk.reserve(r.ID, data); err != nil {
			reason = StoreFull

		}
	}
	if reason != "" {
		r.State, _, _ = Step(r.State, Event{Kind: CaptureFailed, Reason: reason})
		token, err := s.token(r)
		if err != nil {
			return Status{}, err
		}
		out := s.statusLocked(r)
		out.Draft = token
		return out, nil
	}
	s.records[r.ID] = r
	s.jobs <- r.ID
	return s.statusLocked(r), nil
}

func (s *Service) statusLocked(r *captureRecord) Status {
	out := Status{Request: r.Request, Draft: r.ID, NoteID: r.NoteID, Unavailable: r.State.Unavailable, CapturedAt: r.Evidence.Crop.CapturedAt}
	out.Error = r.Problem
	switch r.State.Phase {
	case Preparing:
		out.State = "pending"
	case Ready:
		out.State = "ready"
		if r.State.Unavailable != "" && r.State.Unavailable != CaptureBusy && r.State.Unavailable != StoreFull {
			out.State = "failed"
		}
	case Attaching:
		out.State = "pending"
		if out.Error == "" {
			out.Error = "save not confirmed; retry Save note to check"
		}
	case Attached, Bundled, Transferring:
		out.State = "saved"
	case Retired:
		out.State = "cancelled"
	}
	return out
}
func (s *Service) Status(owner Owner, id string) (Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.HasPrefix(id, "u.") {
		r, err := s.untoken(owner, id)
		if err != nil {
			return Status{}, err
		}
		if current := s.records[r.ID]; current != nil && current.Owner == owner {
			r = current
		}
		out := s.statusLocked(r)
		out.Draft = id
		return out, nil
	}
	r := s.findLocked(owner, id)
	if r == nil {
		return Status{}, errors.New("capture draft not found")
	}
	return s.statusLocked(r), nil
}
func (s *Service) findLocked(owner Owner, id string) *captureRecord {
	for _, r := range s.records {
		if r.Owner == owner && (r.ID == id || r.Request == id) {
			return r
		}
	}
	return nil
}
func (s *Service) applyLocked(r *captureRecord, event Event) error {
	next, _, err := Step(r.State, event)
	if err != nil {
		return err
	}
	copy := *r
	copy.State = next
	if r.Ephemeral {
		r.State = next
		return nil
	}
	if next.Phase == Retired {
		// Persist terminal ownership before retryable unlink operations.
		if r.State.Phase != Retired {
			if err = s.persistLocked(&copy); err != nil {
				return err
			}
		}
		r.State = next
		if err = s.disk.remove(r.ID); err == nil {
			err = s.disk.removeRef(r.ID)
		}
	} else {
		err = s.persistLocked(&copy)
	}

	if err == nil {
		r.State = next
	}
	return err
}
func (s *Service) Cancel(owner Owner, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.HasPrefix(id, "u.") {
		r, err := s.untoken(owner, id)
		if err != nil {
			return err
		}
		if current := s.records[r.ID]; current != nil && (current.State.Phase == Attached || current.State.Phase == Bundled || current.State.Phase == Transferring) {
			return nil
		}
		return s.revokeLocked(owner, r.Request)
	}
	r := s.findLocked(owner, id)
	if r == nil {
		return s.revokeLocked(owner, id)
	}
	if r.cancel != nil {
		r.cancel()
	}
	return s.applyLocked(r, Event{Kind: Cancel})
}
func (s *Service) Revoke(owner Owner) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.records {
		if r.Owner == owner {
			if r.cancel != nil {
				r.cancel()
			}
			_ = s.applyLocked(r, Event{Kind: Expire})
		}
	}
}

// RevokeScope retires unsaved work across permission generations when the
// authenticated owner's final event stream disconnects. Uncertain writes retain
// their evidence until the writer outcome can be observed.
func (s *Service) RevokeScope(scope string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if scope == "" {
		return
	}
	s.revoked["scope\x00"+scope] = s.opts.Now().Add(DraftTTL)
	for _, r := range s.records {
		if r.Owner.Scope == scope && (r.State.Phase == Preparing || r.State.Phase == Ready || r.State.Phase == Attaching) {
			if r.cancel != nil {
				r.cancel()
			}
			_ = s.applyLocked(r, Event{Kind: Expire})
		}
	}
}

// RevokePreparations retires unsaved work on a view/capability change. An
// uncertain attachment retains its evidence until the writer is reconciled.
func (s *Service) RevokePreparations() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.records {
		if r.State.Phase == Preparing || r.State.Phase == Ready || r.State.Phase == Attaching {
			if r.cancel != nil {
				r.cancel()
			}
			_ = s.applyLocked(r, Event{Kind: Expire})
		}
	}
}
func (s *Service) run() {
	defer close(s.done)
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			_ = s.Sweep(s.ctx)
		case id := <-s.jobs:
			s.capture(id)
		}
	}
}
func (s *Service) capture(id string) {
	s.mu.Lock()
	r := s.records[id]
	if r == nil || r.State.Phase != Preparing {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithDeadline(s.ctx, r.Created.Add(OperationBudget))
	r.cancel = cancel
	prepared, admission := r.Prepared, r.admission
	s.mu.Unlock()
	evidence, png, err := s.opts.Capture.Capture(ctx, prepared)
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		err = admission.Check()
	}
	cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	r.cancel = nil
	if r.State.Phase != Preparing {
		return
	}
	if err != nil {
		_ = s.applyLocked(r, Event{Kind: CaptureFailed, Reason: "crop unavailable: " + err.Error()})
		return
	}
	if evidence.Crop.Unavailable != "" {
		_ = s.applyLocked(r, Event{Kind: CaptureFailed, Reason: evidence.Crop.Unavailable})
		return
	}
	err = validateCaptureEvidence(evidence, prepared, png)
	if err != nil {
		_ = s.applyLocked(r, Event{Kind: CaptureFailed, Reason: "crop unavailable: " + err.Error()})
		return
	}
	next, _, _ := Step(r.State, Event{Kind: CaptureComplete})
	copy := *r
	copy.State = next
	copy.Evidence = evidence
	copy.Expires = s.opts.Now().Add(DraftTTL)
	data, err := json.Marshal(copy)
	if err == nil {
		err = s.disk.put(id, data, png)
	}
	if err != nil {
		_ = s.applyLocked(r, Event{Kind: CaptureFailed, Reason: StoreFull})
		return
	}
	r.State = next
	r.Evidence = evidence
	r.Expires = copy.Expires
}
func (s *Service) Sweep(ctx context.Context) error {
	bounded, cancel := context.WithTimeout(ctx, OperationBudget)
	defer cancel()
	// External observations run outside the service mutex. Their reducers recheck
	// ownership after the writer/publication barrier returns.
	var first error
	if err := s.ReconcilePending(bounded, s.opts.ObservePublication); err != nil {
		first = err
	}
	s.mu.Lock()
	var attaching []string
	for id, r := range s.records {
		if r.State.Phase == Attaching {
			attaching = append(attaching, id)
		}
	}
	s.mu.Unlock()
	if s.opts.Observe != nil {
		for _, id := range attaching {
			if err := bounded.Err(); err != nil {
				if first == nil {
					first = err
				}
				break
			}
			if err := s.reconcileWrite(bounded, id); err != nil && first == nil {
				first = err
			}
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.opts.Now()
	for key, until := range s.revoked {
		if !now.Before(until) {
			delete(s.revoked, key)
		}
	}
	for id, r := range s.records {
		if err := ctx.Err(); err != nil {
			return err
		}
		if r.State.Phase == Retired {
			if err := s.applyLocked(r, Event{Kind: Expire}); err != nil {
				return err
			}
			if !now.Before(r.Expires) {
				delete(s.records, id)
			}
			continue
		}
		if (r.State.Phase == Preparing || r.State.Phase == Ready) && !now.Before(r.Expires) {
			if r.cancel != nil {
				r.cancel()
			}
			if err := s.applyLocked(r, Event{Kind: Expire}); err != nil {
				return err
			}
		}
		if (r.State.Phase == Attached || r.State.Phase == Bundled || r.State.Phase == Transferring) && recordingAbsent(r.RecordingPath) {
			if err := s.applyLocked(r, Event{Kind: Resolve}); err != nil {
				return err
			}
		}
	}
	if err := s.disk.sweepSources(now); err != nil {
		return err
	}
	if captureOwnersMu.TryLock() {
		if err := s.recoverInactiveLocked(bounded); err != nil && first == nil {
			first = err
		}
		captureOwnersMu.Unlock()
	}
	return first
}

func (s *Service) Close() error {
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		s.cancel()
		for _, r := range s.records {
			if r.State.Phase == Preparing || r.State.Phase == Ready {
				_ = s.applyLocked(r, Event{Kind: Expire})
			}
		}
	}
	s.mu.Unlock()
	select {
	case <-s.done:
		captureOwnersMu.Lock()
		key := s.ownerKey
		if captureOwners[s.disk.root][key] == s {
			delete(captureOwners[s.disk.root], key)
		}
		captureOwnersMu.Unlock()
		return nil
	case <-time.After(OperationBudget):
		return errors.New("capture shutdown timed out")
	}
}

// Capacity fallback tokens contain no crop bytes and consume no store slot.
// The signature binds immutable preparation, owner, generation and expiry.
func (s *Service) token(r *captureRecord) (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	m := hmac.New(sha256.New, s.key[:])
	m.Write(data)
	return "u." + base64.RawURLEncoding.EncodeToString(data) + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil)), nil
}
func (s *Service) untoken(owner Owner, token string) (*captureRecord, error) {
	if len(token) > 32768 {
		return nil, errors.New("invalid capture token")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid capture token")
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}
	m := hmac.New(sha256.New, s.key[:])
	m.Write(data)
	if !hmac.Equal(sig, m.Sum(nil)) {
		return nil, errors.New("invalid capture token")
	}
	var r captureRecord
	if err = json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	if r.Owner != owner || r.RecordingPath != s.opts.RecordingPath || !s.opts.Now().Before(r.Expires) {
		return nil, errors.New("capture token expired or belongs to another viewer")
	}
	if until, ok := s.revoked["scope\x00"+owner.Scope]; ok && !r.Created.After(until.Add(-DraftTTL)) && s.records[r.ID] == nil {
		return nil, errors.New("capture draft cancelled")
	}
	if s.isRevoked(owner, r.Request) {
		return nil, errors.New("capture draft cancelled")
	}
	return &r, nil
}

func revocationKey(owner Owner, request string) string {
	return fmt.Sprintf("%s\x00%d\x00%s", owner.Scope, owner.Generation, request)
}
func (s *Service) isRevoked(owner Owner, request string) bool {
	until, ok := s.revoked[revocationKey(owner, request)]
	return ok && s.opts.Now().Before(until)
}
func (s *Service) revokeLocked(owner Owner, request string) error {
	if request == "" || len(request) > 128 {
		return errors.New("invalid capture request")
	}
	now := s.opts.Now()
	for key, until := range s.revoked {
		if !now.Before(until) {
			delete(s.revoked, key)
		}
	}
	if len(s.revoked) >= 4096 {
		return errors.New("too many pending capture cancellations")
	}
	s.revoked[revocationKey(owner, request)] = now.Add(DraftTTL)
	return nil
}

func validCaptureRecord(r *captureRecord, id string) error {
	if validRecord(id) != nil || r.ID != id || r.Owner.Scope == "" || len(r.Owner.Scope) > 256 || len(r.Owner.Identity) > 256 || r.Request == "" || len(r.Request) > 128 || !filepath.IsAbs(r.RecordingPath) || filepath.Clean(r.RecordingPath) != r.RecordingPath || r.Created.IsZero() || r.Expires.Before(r.Created) || r.Prepared.Document < 1 {
		return errors.New("invalid capture identity or preparation")
	}
	switch r.State.Phase {
	case Preparing, Ready, Attaching, Attached, Transferring, Bundled, Retired:
	default:
		return errors.New("invalid capture phase")
	}
	if (r.State.Phase == Attaching || r.State.Phase == Attached || r.State.Phase == Transferring || r.State.Phase == Bundled) && r.NoteID == "" {
		return errors.New("capture owner has no note")
	}
	if r.Note != nil && (r.Note.ID != r.NoteID || r.Note.Capture != r.ID || r.Note.Owner != r.Owner.Identity) {
		return errors.New("capture note identity mismatch")
	}
	if r.State.Phase == Transferring && !core.ValidID(r.PendingSubmission) {
		return errors.New("transfer has no publication identity")
	}
	if r.State.Phase != Transferring && r.PendingSubmission != "" {
		return errors.New("unexpected pending publication")
	}
	if r.BundlePath != "" && (!filepath.IsAbs(r.BundlePath) || filepath.Clean(r.BundlePath) != r.BundlePath || filepath.Base(r.BundlePath) != "submission.json") {
		return errors.New("invalid bundle reference")
	}
	if r.State.Phase == Bundled && r.BundlePath == "" {
		return errors.New("bundled capture has no reference")
	}
	return nil
}
func recordingAbsent(path string) bool {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return false
	}
	_, err := os.Lstat(path)
	return os.IsNotExist(err)
}

// recoverRecording only infers absence from complete, bounded, valid persisted
// streams. A torn tail or failed read is unknown and keeps all saved evidence.
func (s *Service) recoverRecording() error {
	root, err := os.OpenRoot(s.opts.RecordingPath)
	if err != nil {
		if recordingAbsent(s.opts.RecordingPath) {
			for _, r := range s.records {
				if r.State.Phase == Attaching {
					if e := s.applyLocked(r, Event{Kind: Expire}); e != nil {
						return e
					}
					if e := s.applyLocked(r, Event{Kind: WriteAbsent}); e != nil {
						return e
					}
				} else if r.State.Phase == Attached || r.State.Phase == Bundled || r.State.Phase == Transferring {
					if e := s.applyLocked(r, Event{Kind: Resolve}); e != nil {
						return e
					}
				}
			}
		}
		return nil
	}
	defer root.Close()
	files := map[string][]recording.Located{}
	for _, name := range []string{recording.FileEvents, recording.FileNotes} {
		data, e := readBounded(root, name, 64<<20)
		if os.IsNotExist(e) && name == recording.FileNotes {
			continue
		}
		if e != nil || len(data) > 0 && !bytes.HasSuffix(data, []byte{'\n'}) {
			return nil
		}
		located, bad := recording.DecodeFile(name, data)
		if len(bad) > 0 {
			return nil
		}
		files[name] = located
	}
	state, err := recording.Reduce(recording.Records(recording.Merge(files)), math.MaxInt64)
	if err != nil || state.Head == nil {
		return nil
	}
	observed := map[string]*recording.NoteView{}
	for _, n := range state.Notes {
		observed[n.ID] = n
	}
	for _, r := range s.records {
		n, present := observed[r.NoteID]
		if present && (n.Capture != r.ID || n.Owner != r.Owner.Identity) {
			continue
		}
		if r.State.Phase == Attaching {
			if present {
				if err = s.applyLocked(r, Event{Kind: WriteConfirmed}); err != nil {
					return err
				}
			} else {
				if err = s.applyLocked(r, Event{Kind: Expire}); err != nil {
					return err
				}
				if err = s.applyLocked(r, Event{Kind: WriteAbsent}); err != nil {
					return err
				}
				continue
			}
		}
		if (r.State.Phase == Attached || r.State.Phase == Bundled || r.State.Phase == Transferring) && (!present || n.State == recording.NoteResolved) {
			if err = s.applyLocked(r, Event{Kind: Resolve}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) PendingSubmissions() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := map[string]bool{}
	for _, r := range s.records {
		if r.State.Phase == Transferring && r.PendingSubmission != "" {
			seen[r.PendingSubmission] = true
		}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
func (s *Service) ReconcilePending(ctx context.Context, observe func(context.Context, string) (core.Submission, core.PublicationObservation, error)) error {
	if observe == nil {
		return nil
	}
	bounded, cancel := context.WithTimeout(ctx, OperationBudget)
	defer cancel()
	for _, id := range s.PendingSubmissions() {
		if err := bounded.Err(); err != nil {
			return err
		}
		sub, obs, err := observe(bounded, id)
		if err != nil {
			return err
		}
		if obs.State == core.PublicationUnknown || obs.State == "" {
			return errors.New("publication not confirmed; retry submission")
		}
		if sub.ID == "" && obs.State == core.PublicationAbsent {
			sub.ID = id
		}
		if sub.ID != id {
			return errors.New("publication observation identity mismatch")
		}
		if err = s.ReconcilePublication(bounded, sub, obs); err != nil {
			return err
		}
	}
	return nil
}

func canonicalRecordingPath(path string) string {
	// Resolve existing ancestors as well: deleting a recording must not change
	// the identity of its still-active service (notably macOS /var aliases).
	var suffix []string
	ancestor := filepath.Clean(path)
	for {
		if real, err := filepath.EvalSymlinks(ancestor); err == nil {
			for i := len(suffix) - 1; i >= 0; i-- {
				real = filepath.Join(real, suffix[i])
			}
			return real
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			return filepath.Clean(path)
		}
		suffix = append(suffix, filepath.Base(ancestor))
		ancestor = parent
	}
}

// The active-owner registry is held throughout this bounded recovery window.
// Historical notes are loaded into temporary owners, never the current cast.
func (s *Service) recoverInactiveLocked(ctx context.Context) error {
	bounded, cancel := context.WithTimeout(ctx, OperationBudget)
	defer cancel()
	groups := map[string]map[string]*captureRecord{}
	collect := func(id string, data []byte) error {
		var r captureRecord
		if json.Unmarshal(data, &r) != nil || validCaptureRecord(&r, id) != nil {
			return nil // Unknown metadata never authorizes deleting the retained bytes.
		}
		if captureOwners[s.disk.root][canonicalRecordingPath(r.RecordingPath)] != nil {
			return nil
		}
		if groups[r.RecordingPath] == nil {
			groups[r.RecordingPath] = map[string]*captureRecord{}
		}
		groups[r.RecordingPath][id] = &r
		return nil
	}
	ids, err := s.disk.list()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err = bounded.Err(); err != nil {
			return err
		}
		data, e := s.disk.metadata(id)
		if e != nil {
			continue
		}
		if e = collect(id, data); e != nil {
			return e
		}
	}
	refs, err := s.disk.listRefs()
	if err != nil {
		return err
	}
	for _, id := range refs {
		if err = bounded.Err(); err != nil {
			return err
		}
		data, e := s.disk.readRef(id)
		if e != nil {
			continue
		}
		if e = collect(id, data); e != nil {
			return e
		}
	}
	for path, records := range groups {
		if err = bounded.Err(); err != nil {
			return err
		}
		opts := s.opts
		opts.RecordingPath = path
		owner := &Service{opts: opts, disk: s.disk, records: records}
		for _, r := range records {
			if r.State.Phase == Preparing || r.State.Phase == Ready || r.State.Phase == Retired {
				if e := owner.applyLocked(r, Event{Kind: Expire}); e != nil {
					return e
				}
			}
		}
		if e := owner.recoverReferencesLocked(); e != nil {
			return e
		}
		if e := owner.recoverRecording(); e != nil {
			return e
		}
		// Unknown publication remains durable for a later sweep. One bad historical
		// recording must not prevent unrelated safe recovery in this window.
		_ = owner.ReconcilePending(bounded, opts.ObservePublication)
	}
	return nil
}
