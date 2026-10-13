package notedraft

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
)

const DraftTTL = 10 * time.Minute
const OperationBudget = 36 * time.Second

type Owner struct {
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
	BundlePath        string             `json:"bundle_path,omitempty"`
	PendingSubmission string             `json:"pending_submission,omitempty"`
	// Volatile authority and worker cancellation never enter the local metadata.
	admission board.AnnotationAdmission
	cancel    context.CancelFunc
}
type Service struct {
	mu      sync.Mutex
	opts    Options
	disk    *diskStore
	ctx     context.Context
	cancel  context.CancelFunc
	done    chan struct{}
	jobs    chan string
	records map[string]*captureRecord
	key     [32]byte
	closed  bool
}

func New(opts Options) (*Service, error) {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Capture == nil {
		return nil, errors.New("capture source required")
	}
	disk, err := newDiskStore(opts.Root)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{opts: opts, disk: disk, ctx: ctx, cancel: cancel, done: make(chan struct{}), jobs: make(chan string, 8), records: map[string]*captureRecord{}}
	if _, err = rand.Read(s.key[:]); err != nil {
		cancel()
		return nil, err
	}
	// Startup recovery never treats abandoned preparations as saved notes.
	ids, err := disk.list()
	if err != nil {
		cancel()
		return nil, err
	}
	for _, id := range ids {
		data, _, e := disk.read(id)
		if e != nil {
			cancel()
			return nil, e
		}
		var r captureRecord
		if e = json.Unmarshal(data, &r); e != nil || r.ID != id {
			cancel()
			return nil, fmt.Errorf("invalid capture record %s", id)
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
	count, used, err := s.disk.usage()
	if err != nil {
		return Status{}, err
	}
	reserved := int64(0)
	for _, other := range s.records {
		if other.State.Phase == Preparing {
			reserved += MaxPNGBytes
		}
	}
	if reason == "" {
		reason = Capacity(count, used+reserved, 1, MaxPNGBytes)
	}
	if reason != "" {
		r.State, _, _ = Step(r.State, Event{Kind: CaptureFailed, Reason: reason})
		token, err := s.token(r)
		if err != nil {
			return Status{}, err
		}
		return Status{State: "ready", Request: request, Draft: token, Unavailable: reason}, nil
	}
	data, err := json.Marshal(r)
	if err != nil {
		return Status{}, err
	}
	if err = s.disk.put(r.ID, data, nil); err != nil {
		return Status{}, err
	}
	s.records[r.ID] = r
	s.jobs <- r.ID
	return s.statusLocked(r), nil
}

func (s *Service) statusLocked(r *captureRecord) Status {
	out := Status{Request: r.Request, Draft: r.ID, NoteID: r.NoteID, Unavailable: r.State.Unavailable, CapturedAt: r.Evidence.Crop.CapturedAt}
	switch r.State.Phase {
	case Preparing:
		out.State = "pending"
	case Ready:
		out.State = "ready"
	case Attaching:
		out.State = "pending"
		out.Error = "save not confirmed; retry Save note to check"
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
	data, err := json.Marshal(copy)
	if err != nil {
		return err
	}
	if next.Phase == Retired {
		err = s.disk.remove(r.ID)
	} else {
		err = s.disk.update(r.ID, data)
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
		_, err := s.untoken(owner, id)
		return err
	}
	r := s.findLocked(owner, id)
	if r == nil {
		return nil
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
	if err = core.ValidatePNG(png); err == nil {
		err = evidence.Page.Validate()
	}
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
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.opts.Now()
	for id, r := range s.records {
		if err := ctx.Err(); err != nil {
			return err
		}
		if r.State.Phase == Retired && !now.Before(r.Expires) {
			delete(s.records, id)
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
	}
	return nil
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
	return &r, nil
}
