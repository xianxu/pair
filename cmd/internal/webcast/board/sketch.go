package board

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// SketchAuthority binds work to its original authenticated owner and driving
// turn. MetadataCheck proves seat lifetime without granting drawing authority.
type SketchAuthority struct {
	Author               recording.Author
	Owner                string
	Epoch                uint64
	Annotation           AnnotationAdmission
	Check, MetadataCheck func() error
}

type sketchOrigin struct {
	owner, author string
	epoch         uint64
}
type sketchFlight struct {
	digest   string
	sketch   string
	sequence uint64
	done     chan struct{}
	reply    SketchReply
}
type sketchRefusal struct {
	key, digest string
	reply       SketchReply
}

// SketchService holds ephemeral ingress and one scene lease. The Board's leaf
// mutex protects it; no dispatch lock is held across an acknowledged write.
type SketchService struct {
	board                *Board
	write                func(context.Context, SketchWrite) (SketchWriteResult, error)
	observe              func(context.Context, string, string) (SketchObservation, error)
	candidate            func(*recording.SketchBegin, int, int) (*recording.SketchSource, error)
	now                  func() time.Time
	origins              map[string]sketchOrigin
	begins               map[string]string
	flights              map[string]*sketchFlight
	busy                 map[string]string
	refusals             map[string]sketchRefusal
	source               *recording.SketchSource
	handle, sourceSketch string
	lastMutation         time.Time
	leaseVersion         uint64
	wake                 chan struct{}
}

func (b *Board) NewSketchService(write func(context.Context, SketchWrite) (SketchWriteResult, error), observe func(context.Context, string, string) (SketchObservation, error), candidate func(*recording.SketchBegin, int, int) (*recording.SketchSource, error)) *SketchService {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.sketchService == nil {
		b.sketchService = &SketchService{board: b, write: write, observe: observe, candidate: candidate, now: time.Now, origins: map[string]sketchOrigin{}, begins: map[string]string{}, flights: map[string]*sketchFlight{}, busy: map[string]string{}, refusals: map[string]sketchRefusal{}, wake: make(chan struct{}, 1)}
	}
	return b.sketchService
}

func (b *Board) retireSketchSourceLocked() {
	s := b.sketchService
	if s == nil {
		return
	}
	if b.closed {
		s.wakeLocked()
	}
	if s.source == nil {
		return
	}
	m, ok := b.st.Sketches.Metadata(s.sourceSketch)
	if b.closed || !ok || m.Phase != "drawing" {
		s.source = nil
		s.sourceSketch = ""
		s.handle = ""
		s.wakeLocked()
	}
}

func (s *SketchService) wakeLocked() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// Run owns one reusable timer for this cast. Its caller cancels and joins it;
// accepted mutations coalesce wakeups and never allocate a timer per point.
func (s *SketchService) Run(ctx context.Context) {
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	if !timer.Stop() {
		<-timer.C
	}
	for {
		s.board.mu.Lock()
		closed := s.board.closed
		var tick <-chan time.Time
		if s.source != nil {
			delay := max(time.Duration(0), s.lastMutation.Add(2*time.Minute).Sub(s.now()))
			timer.Reset(delay)
			tick = timer.C
		}
		s.board.mu.Unlock()
		if closed {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-s.wake:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case <-tick:
			_ = s.Expire(ctx, s.now())
		}
	}
}

func sketchLookup(a SketchAuthority, c SketchCommand) string {
	id := c.Sketch
	if c.Sequence == 0 {
		id = ""
	}
	return fmt.Sprintf("%s\x00%s\x00%d\x00%s\x00%s", a.Owner, a.Author.ID, c.Epoch, id, c.Operation)
}
func (s *SketchService) check(a SketchAuthority, drawing bool) error {
	if a.Author.ID == "" || a.Owner == "" {
		return ErrOwner
	}
	if err := a.Annotation.Check(); err != nil {
		return err
	}
	if a.MetadataCheck != nil {
		if err := a.MetadataCheck(); err != nil {
			return err
		}
	}
	if drawing {
		if a.Check == nil {
			return ErrDisabled
		}
		return a.Check()
	}
	return nil
}
func sketchFailure(c SketchCommand, code string) SketchReply {
	return SketchReply{Operation: c.Operation, Sketch: c.Sketch, Sequence: c.Sequence, Outcome: SketchRefused, Code: code}
}
func (s *SketchService) ownedLocked(a SketchAuthority, id string, epoch uint64) bool {
	o, ok := s.origins[id]
	return ok && o.owner == a.Owner && o.author == a.Author.ID && o.epoch == epoch
}
func (s *SketchService) statusLocked(c SketchCommand, outcome SketchOutcome, code string) SketchReply {
	v := SketchReply{Operation: c.Operation, Sketch: c.Sketch, Sequence: c.Sequence, Outcome: outcome, Code: code}
	if m, ok := s.board.st.Sketches.Metadata(c.Sketch); ok {
		v.NextSequence = m.NextSequence
		tops, _ := s.board.st.Sketches.StackTops(c.Sketch)
		v.Status = &SketchStatus{Phase: m.Phase, Reason: m.Reason, Label: m.Label, OpenStroke: m.OpenStroke, VisibleTop: tops.VisibleTop, RedoTop: tops.RedoTop}
		if s.source != nil && s.handle != "" && s.sourceSketch == c.Sketch && m.Phase == "drawing" {
			v.Source = &SketchSourceDescriptor{Handle: s.handle, Time: m.T, Length: s.source.Length(), Doc: m.Doc, Tab: m.Tab, VP: m.VP}
		}
	}
	return v
}

// Execute reserves one immutable operation before dispatch. Started writer work
// owns settlement even when the HTTP caller disappears; another exact caller
// observes that same flight rather than emitting a second physical event.
func (s *SketchService) Execute(ctx context.Context, a SketchAuthority, c SketchCommand) SketchReply {
	digest, err := SketchCommandDigest(c)
	if err != nil || c.Kind == "sketch-status" || c.Kind == "sketch-source" || c.Kind == "sketch-asset" {
		return sketchFailure(c, "shape")
	}
	if c.Epoch != a.Epoch {
		return sketchFailure(c, "epoch")
	}
	drawing := c.Kind != "sketch-label"
	// Exact receipts may be inspected after the original driving turn ended.
	if err := s.check(a, false); err != nil {
		return sketchFailure(c, "unavailable")
	}
	b := s.board
	key := sketchLookup(a, c)
	b.mu.Lock()
	if b.closed || b.broken != nil {
		b.mu.Unlock()
		return sketchFailure(c, "closed")
	}
	if c.Kind == "sketch-begin" {
		if id := s.begins[key]; id != "" {
			c.Sketch = id
		}
	}
	refusalKey := c.Sketch
	if c.Kind == "sketch-begin" {
		refusalKey = "begin"
	}
	if old, ok := s.refusals[refusalKey]; ok && old.key == key {
		b.mu.Unlock()
		if old.digest != digest {
			return sketchFailure(c, "conflict")
		}
		return old.reply
	}
	if c.Sketch != "" && !s.ownedLocked(a, c.Sketch, c.Epoch) {
		b.mu.Unlock()
		return sketchFailure(c, "owner")
	}
	if old := s.flights[key]; old != nil {
		if old.digest != digest {
			b.mu.Unlock()
			return sketchFailure(c, "conflict")
		}
		done := old.done
		b.mu.Unlock()
		select {
		case <-done:
			b.mu.Lock()
			v := old.reply
			b.mu.Unlock()
			return v
		case <-ctx.Done():
			return SketchReply{Operation: c.Operation, Sketch: old.sketch, Sequence: c.Sequence, Outcome: SketchUnknown, Code: "pending"}
		}
	}
	if receipt, ok := b.st.Sketches.Receipt(c.Sketch, c.Operation); ok {
		if receipt.Digest != digest || receipt.Sequence != c.Sequence {
			b.mu.Unlock()
			return sketchFailure(c, "conflict")
		}
		v := s.statusLocked(c, SketchAccepted, "")
		b.mu.Unlock()
		return v
	}
	pageSketch := c.Sketch
	if c.Kind == "sketch-begin" {
		pageSketch = ""
	}
	if !b.sketchOnCurrentPageLocked(pageSketch) {
		b.mu.Unlock()
		return sketchFailure(c, "stale")
	}
	if len(s.flights) >= 64 {
		b.mu.Unlock()
		return sketchFailure(c, "busy")
	}
	if c.Kind == "sketch-begin" && c.Sketch == "" {
		if len(s.origins) >= recording.SketchMaxCount || s.busy["begin"] != "" {
			b.mu.Unlock()
			return sketchFailure(c, "busy")
		}
		c.Sketch = b.newID()
		s.origins[c.Sketch] = sketchOrigin{a.Owner, a.Author.ID, c.Epoch}
		s.begins[key] = c.Sketch
	}
	busyKey := c.Sketch
	if c.Kind == "sketch-begin" {
		busyKey = "begin"
	}
	if s.busy[busyKey] != "" {
		b.mu.Unlock()
		return sketchFailure(c, "busy")
	}
	flight := &sketchFlight{digest: digest, sketch: c.Sketch, sequence: c.Sequence, done: make(chan struct{})}
	s.flights[key] = flight
	s.busy[busyKey] = key
	var record recording.Record
	op := recording.SketchOperation{Operation: c.Operation, Sequence: c.Sequence, Digest: digest}
	doc := 0
	if c.Kind == "sketch-begin" {
		doc, err = b.pageLocked(c.Seg)
		if err == nil {
			author := a.Author
			author.Color = b.palette[c.Color]
			record = &recording.SketchBegin{SketchOperation: op, ID: c.Sketch, Author: author, Owner: a.Owner, Doc: doc, Tab: b.st.Tab.ID, VP: b.st.VP}
		}
	} else {
		m, ok := b.st.Sketches.Metadata(c.Sketch)
		if !ok {
			err = ErrUnknown
		} else {
			doc = m.Doc
		}
		if c.Kind == "sketch-label" {
			doc = 0
		}
		source := s.source
		switch c.Kind {
		case "sketch-stroke-begin":
			if source == nil || s.sourceSketch != c.Sketch {
				err = ErrStale
				break
			}
			at, e := b.model.Anchor(source.State(), c.Node)
			if e != nil {
				err = ErrStale
				break
			}
			box := c.Box
			at.Box = &box
			record = &recording.SketchStrokeBegin{SketchOperation: op, Sketch: c.Sketch, ID: c.ID, At: at, Width: 3}
		case "sketch-points":
			record = &recording.SketchPoints{SketchOperation: op, Sketch: c.Sketch, ID: c.ID, Chunk: c.Chunk, Pts: append([]recording.SketchPoint(nil), c.Pts...)}
		case "sketch-stroke-end":
			record = &recording.SketchStrokeEnd{SketchOperation: op, Sketch: c.Sketch, ID: c.ID, Truncated: c.Truncated}
		case "sketch-undo":
			record = &recording.SketchUndo{SketchOperation: op, Sketch: c.Sketch, ID: c.ID}
		case "sketch-redo":
			record = &recording.SketchRedo{SketchOperation: op, Sketch: c.Sketch, ID: c.ID}
		case "sketch-label":
			record = &recording.SketchLabel{SketchOperation: op, Sketch: c.Sketch, Label: c.Label}
		}
	}
	b.mu.Unlock()
	result := SketchWriteResult{Outcome: SketchRefused}
	if err == nil {
		err = s.check(a, drawing)
	}
	if err == nil && s.write == nil {
		err = ErrDisabled
	}
	if err == nil {
		result, err = s.write(ctx, SketchWrite{Record: record, Document: doc, Check: func() error {
			if err := s.check(a, drawing); err != nil {
				return err
			}
			b.mu.Lock()
			defer b.mu.Unlock()
			if !b.pageFeedbackReadyLocked(record) {
				return ErrStale
			}
			if drawing && c.Kind != "sketch-begin" {
				if s.source == nil || s.sourceSketch != c.Sketch || s.now().Sub(s.lastMutation) >= 2*time.Minute {
					return ErrStale
				}
			}
			return nil
		}, Accepted: func() {
			if drawing {
				b.mu.Lock()
				s.lastMutation = s.now()
				s.leaseVersion++
				if c.Kind == "sketch-begin" && s.busy[busyKey] == key {
					delete(s.busy, busyKey)
				}
				s.wakeLocked()
				b.mu.Unlock()
			}
		}, Source: func(begin *recording.SketchBegin) (*recording.SketchSource, error) {
			if s.candidate == nil {
				return nil, ErrNotLive
			}
			return s.candidate(begin, recording.SketchSourcePrefixLimit, recording.SketchSourceStateLimit)
		}})
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if s.busy[busyKey] == key {
		delete(s.busy, busyKey)
	}
	if err != nil && result.Outcome == "" {
		result.Outcome = SketchRefused
	}
	if result.Outcome == SketchAccepted {
		if result.Source != nil {
			if m, ok := b.st.Sketches.Metadata(c.Sketch); ok && m.Phase == "drawing" && !b.closed {
				s.source = result.Source
				s.sourceSketch = c.Sketch
				s.handle = b.newID()
			}
		}
	}
	if result.Outcome == SketchUnknown && result.Source != nil && !b.closed {
		if m, ok := b.st.Sketches.Metadata(c.Sketch); ok && m.Phase == "drawing" {
			s.source = result.Source
			s.sourceSketch = c.Sketch
			s.handle = ""
			s.lastMutation = s.now()
		}
	}
	s.wakeLocked()
	code := ""
	if err != nil || result.Err != nil {
		code = "write-failed"
	}
	flight.reply = s.statusLocked(c, result.Outcome, code)
	close(flight.done)
	// Uncertain effects remain retained and cannot turn into accepted merely
	// because the reducer saw them. Definite results use core receipts on retry.
	if result.Outcome != SketchUnknown {
		delete(s.flights, key)
	}
	if result.Outcome == SketchRefused && c.Kind == "sketch-begin" {
		delete(s.origins, c.Sketch)
		delete(s.begins, key)
	}
	if result.Outcome == SketchRefused {
		s.refusals[refusalKey] = sketchRefusal{key, digest, flight.reply}
	}
	return flight.reply
}

// Status fences the writer before proving absence. Local reducer receipts alone
// cannot establish persistence after a sink failure.
func (s *SketchService) Status(ctx context.Context, a SketchAuthority, c SketchCommand) SketchReply {
	if err := s.check(a, false); err != nil {
		return sketchFailure(c, "unavailable")
	}
	b := s.board
	key := sketchLookup(a, c)
	b.mu.Lock()
	if c.Sketch == "" {
		c.Sketch = s.begins[key]
	}
	refusalKey := c.Sketch
	if c.Sequence == 0 {
		refusalKey = "begin"
	}
	if old, ok := s.refusals[refusalKey]; ok && old.key == key {
		b.mu.Unlock()
		return old.reply
	}
	if c.Sketch != "" && !s.ownedLocked(a, c.Sketch, c.Epoch) {
		b.mu.Unlock()
		return sketchFailure(c, "owner")
	}
	if f := s.flights[key]; f != nil {
		if f.sequence != c.Sequence {
			b.mu.Unlock()
			return sketchFailure(c, "conflict")
		}
		v := s.statusLocked(c, SketchUnknown, "pending")
		select {
		case <-f.done:
			v = f.reply
		default:
		}
		b.mu.Unlock()
		return v
	}
	b.mu.Unlock()
	if s.observe == nil {
		return SketchReply{Operation: c.Operation, Sketch: c.Sketch, Sequence: c.Sequence, Outcome: SketchUnknown, Code: "unavailable"}
	}
	ob, err := s.observe(ctx, c.Sketch, c.Operation)
	b.mu.Lock()
	defer b.mu.Unlock()
	if err != nil {
		return s.statusLocked(c, SketchUnknown, "unavailable")
	}
	if ob.Found {
		if ob.Receipt.Sequence != c.Sequence {
			return sketchFailure(c, "conflict")
		}
		return s.statusLocked(c, SketchAccepted, "")
	}
	return s.statusLocked(c, SketchNotObserved, "")
}

func (s *SketchService) ReadSource(a SketchAuthority, id, handle string, offset, length int) ([]byte, error) {
	if err := s.check(a, true); err != nil {
		return nil, err
	}
	b := s.board
	b.mu.Lock()
	defer b.mu.Unlock()
	m, ok := b.st.Sketches.Metadata(id)
	if b.closed || !ok || m.Phase != "drawing" || !s.ownedLocked(a, id, a.Epoch) || s.source == nil || s.sourceSketch != id || (handle == "" || s.handle != handle) || s.now().Sub(s.lastMutation) >= 2*time.Minute {
		return nil, ErrStale
	}
	return s.source.Read(offset, length)
}

// AdmitSourceAsset authorizes only hashes in this exact frozen source. The
// caller reads a bounded disk chunk after the lease check, never a live URL.
func (s *SketchService) AdmitSourceAsset(a SketchAuthority, id, handle, hash string) error {
	if err := s.check(a, true); err != nil {
		return err
	}
	b := s.board
	b.mu.Lock()
	defer b.mu.Unlock()
	m, ok := b.st.Sketches.Metadata(id)
	if b.closed || !ok || m.Phase != "drawing" || !s.ownedLocked(a, id, a.Epoch) || s.source == nil || s.sourceSketch != id || (handle == "" || s.handle != handle) || s.now().Sub(s.lastMutation) >= 2*time.Minute || !s.source.AllowsAsset(hash) {
		return ErrStale
	}
	return nil
}

func (s *SketchService) SourceLength(a SketchAuthority, id, handle string) (int, error) {
	if _, err := s.ReadSource(a, id, handle, 0, 1); err != nil {
		return 0, err
	}
	s.board.mu.Lock()
	defer s.board.mu.Unlock()
	if s.source == nil || s.sourceSketch != id || s.handle != handle {
		return 0, ErrStale
	}
	return s.source.Length(), nil
}

// Expire submits to the same writer as mutations. The final check compares the
// last accepted mutation, so an expiry queued before fresh work cannot win late.
func (s *SketchService) Expire(ctx context.Context, now time.Time) error {
	b := s.board
	b.mu.Lock()
	id := s.sourceSketch
	last := s.lastMutation
	version := s.leaseVersion
	m, ok := b.st.Sketches.Metadata(id)
	if s.source == nil || !ok || m.Phase != "drawing" || now.Sub(last) < 2*time.Minute {
		b.mu.Unlock()
		return nil
	}
	b.mu.Unlock()
	r := &recording.SketchInterrupt{SketchOperation: recording.SketchOperation{Operation: fmt.Sprintf("expiry-%s-%d", id, m.NextSequence), Sequence: m.NextSequence, Digest: fmt.Sprintf("%064x", m.NextSequence)}, Sketch: id, Reason: recording.SketchInterruptExpiry}
	result, err := s.write(ctx, SketchWrite{Record: r, Document: 0, Check: func() error {
		b.mu.Lock()
		defer b.mu.Unlock()
		if b.closed || s.sourceSketch != id || s.leaseVersion != version || !s.lastMutation.Equal(last) {
			return errors.New("sketch lease renewed")
		}
		return nil
	}})
	if err == nil && result.Outcome != SketchAccepted {
		err = result.Err
		if err == nil {
			err = errors.New("sketch expiry was not acknowledged")
		}
	}
	b.mu.Lock()
	if s.sourceSketch == id && s.leaseVersion == version && s.lastMutation.Equal(last) {
		s.source = nil
		s.sourceSketch = ""
		s.handle = ""
	}
	b.mu.Unlock()
	return err
}
