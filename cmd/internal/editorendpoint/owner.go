package editorendpoint

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
)

// ErrDocumentNotRegistered is positive resolver absence. Other resolver errors
// leave cached authority intact because they do not prove a document is gone.
var ErrDocumentNotRegistered = errors.New("document not registered")

// Only the operation that attempted this publication has an uncertain outcome.
// The owner retains the underlying poison so later operations are unavailable.
var errEndpointPublication = errors.New("endpoint publication uncertain")

// Capacity applies only to a candidate refused before durable IO, never to
// an identically named error returned by an invoked writer or a poisoned owner.
var errEndpointCandidateCapacity = errors.New("endpoint candidate capacity exceeded")

type EndpointConfig struct {
	Store     EndpointStore
	RPC       EditorRPC
	Processes EndpointProcessSource
	Resolve   func(context.Context, string) (string, error)
	Random    func() (string, error)
}

// EndpointOwner is the sole durable writer. Admission and native validation are
// bounded independently from its short authority/publication critical section.
// It starts no goroutines; Close cancels and joins every admitted operation.
type EndpointOwner struct {
	cfg          EndpointConfig
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.Mutex
	state        EndpointState
	poison       error
	closed       bool
	wg           sync.WaitGroup
	admitted     chan struct{}
	registerLane chan struct{}
	validation   chan struct{}
	sweepCursor  int
}

func NewEndpointOwner(ctx context.Context, cfg EndpointConfig) (*EndpointOwner, error) {
	if ctx == nil || cfg.RPC == nil || cfg.Processes == nil || cfg.Resolve == nil || cfg.Random == nil {
		return nil, errors.New("incomplete endpoint owner configuration")
	}
	state, e := cfg.Store.Load()
	if e != nil {
		return nil, e
	}
	owned, cancel := context.WithCancel(ctx)
	return &EndpointOwner{cfg: cfg, ctx: owned, cancel: cancel, state: state, admitted: make(chan struct{}, 65), registerLane: make(chan struct{}, 1), validation: make(chan struct{}, 4)}, nil
}
func endpointFailure(code string, e error) EndpointReply {
	if code == "unavailable" {
		switch {
		case errors.Is(e, errEndpointPublication):
			code = "uncertain"
		case errors.Is(e, errEndpointCandidateCapacity):
			code = "capacity"
		}
	}
	p := EndpointReply{Version: 1, Code: code}
	if e != nil {
		p.Error = e.Error()
	}
	return p
}
func (o *EndpointOwner) begin(observer context.Context) error {
	if observer == nil {
		return errors.New("nil observer context")
	}
	if e := observer.Err(); e != nil {
		return e
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed || o.ctx.Err() != nil {
		return errors.New("endpoint owner closed")
	}
	if o.poison != nil {
		return fmt.Errorf("endpoint authority uncertain: %w", o.poison)
	}
	select {
	case o.admitted <- struct{}{}:
		o.wg.Add(1)
		return nil
	default:
		return errors.New("endpoint admission capacity exceeded")
	}
}
func (o *EndpointOwner) end() { <-o.admitted; o.wg.Done() }
func (o *EndpointOwner) acquire(observer context.Context, lane chan struct{}) error {
	select {
	case lane <- struct{}{}:
		if e := observer.Err(); e != nil {
			<-lane
			return e
		}
		if e := o.ctx.Err(); e != nil {
			<-lane
			return e
		}
		return nil
	case <-observer.Done():
		return observer.Err()
	case <-o.ctx.Done():
		return o.ctx.Err()
	}
}
func (o *EndpointOwner) snapshot() (EndpointState, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.poison != nil {
		return EndpointState{}, o.poison
	}
	if o.ctx.Err() != nil {
		return EndpointState{}, o.ctx.Err()
	}
	return cloneState(o.state), nil
}

// publishLocked is the only publication site. Only an invoked writer error
// poisons this instance, including a write error before rename. Deterministic
// pre-writer refusals leave authority usable; attempted IO requires reopening.
func (o *EndpointOwner) publishLocked(next EndpointState, changed bool) error {
	if o.closed || o.ctx.Err() != nil {
		return errors.New("endpoint owner closed")
	}
	if !changed {
		return nil
	}
	if attempted, e := o.cfg.Store.publish(next); e != nil {
		if !attempted {
			if errors.Is(e, ErrEndpointCapacity) {
				return fmt.Errorf("%w: %w", errEndpointCandidateCapacity, e)
			}
			return e
		}
		o.poison = e
		return fmt.Errorf("%w: %w", errEndpointPublication, e)
	}
	o.state = next
	return nil
}
func selected(s EndpointState, document string) *EndpointBinding {
	for _, r := range s.Records {
		if r.Binding.DocumentID == document {
			if b := CurrentBinding(r); b != nil {
				copy := *b
				return &copy
			}
		}
	}
	return nil
}
func bindingRequest(b EndpointBinding) RegisterRequest {
	return RegisterRequest{Version: 1, Namespace: b.Namespace, DocumentID: b.DocumentID, EditorIncarnation: b.EditorIncarnation, AttachID: b.AttachID, PID: b.PID, Socket: b.Socket, Buffer: b.Buffer, Mode: b.Mode}
}
func sameTuple(b EndpointBinding, r RegisterRequest) bool {
	return b.Namespace == r.Namespace && b.DocumentID == r.DocumentID && b.EditorIncarnation == r.EditorIncarnation && b.AttachID == r.AttachID && b.PID == r.PID && b.Socket == r.Socket && b.Buffer == r.Buffer && b.Mode == r.Mode
}

// knownRefusal needs no invented native identity or entropy to reject an attempt
// which durable high-water has already made permanently ineligible.
func knownRefusal(s EndpointState, r RegisterRequest) *EndpointReply {
	for _, x := range s.Records {
		b := x.Binding
		if b.DocumentID != r.DocumentID || b.EditorIncarnation != r.EditorIncarnation {
			continue
		}
		old, _ := attach(b.AttachID)
		incoming, _ := attach(r.AttachID)
		code := ""
		if incoming < old {
			code = "stale"
		} else if incoming == old {
			if x.Status != "current" {
				code = "superseded"
			} else if !sameTuple(b, r) {
				code = "conflict"
			}
		}
		if code != "" {
			p := endpointFailure(code, nil)
			return &p
		}
	}
	return nil
}

type endpointProbe struct {
	identity, file string
	kind           string
	err            error
}

func (o *EndpointOwner) probe(r RegisterRequest, expected string) endpointProbe {
	file, e := o.cfg.Resolve(o.ctx, r.DocumentID)
	if e != nil {
		if errors.Is(e, ErrDocumentNotRegistered) {
			return endpointProbe{kind: "absent", err: e}
		}
		return endpointProbe{kind: "unknown", err: e}
	}
	if !canonicalPath(file) {
		return endpointProbe{kind: "unknown", err: errors.New("resolver returned noncanonical file")}
	}
	before, e := o.cfg.Processes.ReadIdentity(r.PID)
	if e != nil || before == "" {
		kind := "unknown"
		if expected != "" {
			c := ClassifyEndpointProcess(expected, before, e)
			if c == EndpointProcessDead || c == EndpointProcessReplaced {
				kind = "dead"
			}
		}
		return endpointProbe{kind: kind, err: e}
	}
	if expected != "" && before != expected {
		return endpointProbe{kind: "dead"}
	}
	if e = ValidateEndpointSocket(r.Socket); e != nil {
		return endpointProbe{kind: "unknown", err: e}
	}
	d, e := o.cfg.RPC.Describe(o.ctx, r)
	if e != nil {
		return endpointProbe{kind: "unknown", err: e}
	}
	afterFile, e := o.cfg.Resolve(o.ctx, r.DocumentID)
	if e != nil {
		if errors.Is(e, ErrDocumentNotRegistered) {
			return endpointProbe{kind: "absent", err: e}
		}
		return endpointProbe{kind: "unknown", err: e}
	}
	after, e := o.cfg.Processes.ReadIdentity(r.PID)
	switch ClassifyEndpointProcess(before, after, e) {
	case EndpointProcessDead, EndpointProcessReplaced:
		return endpointProbe{kind: "dead", identity: before}
	case EndpointProcessUnknown:
		return endpointProbe{kind: "unknown", err: e}
	}
	if e = ValidateEndpointSocket(r.Socket); e != nil {
		return endpointProbe{kind: "unknown", err: e}
	}
	if d == nil {
		return endpointProbe{kind: "nil", identity: before, file: file}
	}
	if afterFile != file || d.File != file || !sameTuple(*d, r) {
		return endpointProbe{kind: "mismatch", identity: before, file: file, err: errors.New("editor description does not match registered document and endpoint")}
	}
	return endpointProbe{kind: "alive", identity: before, file: file}
}
func (o *EndpointOwner) negativeEvent(s EndpointState, r RegisterRequest, probe endpointProbe) *EndpointEvent {
	b := selected(s, r.DocumentID)
	if b == nil {
		return nil
	}
	if probe.kind == "dead" && b.EditorIncarnation == r.EditorIncarnation && b.PID == r.PID {
		c := ProcessEvidence{EditorIncarnation: b.EditorIncarnation, PID: b.PID, ProcessIdentity: b.ProcessIdentity}
		return &EndpointEvent{Collect: &c}
	}
	if probe.kind == "absent" || ((probe.kind == "nil" || probe.kind == "mismatch") && sameTuple(*b, r)) {
		return &EndpointEvent{Retire: b}
	}
	return nil
}
func (o *EndpointOwner) applyNegative(event *EndpointEvent) (bool, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.applyNegativeLocked(event)
}
func (o *EndpointOwner) applyNegativeLocked(event *EndpointEvent) (bool, error) {
	if event == nil {
		return false, nil
	}
	if o.poison != nil {
		return false, o.poison
	}
	n, p, e := ApplyEndpointEvent(o.state, *event)
	if e != nil {
		return false, e
	}
	if e = o.publishLocked(n, p.Changed); e != nil {
		return false, e
	}
	return p.Changed, nil
}
func (o *EndpointOwner) Register(observer context.Context, r RegisterRequest) EndpointReply {
	if e := ValidateEndpointRequest(r); e != nil {
		return endpointFailure("invalid", e)
	}
	if r.Namespace != o.cfg.Store.Namespace {
		return endpointFailure("invalid", errors.New("namespace mismatch"))
	}
	if e := o.begin(observer); e != nil {
		return endpointFailure("unavailable", e)
	}
	defer o.end()
	if e := o.acquire(observer, o.registerLane); e != nil {
		return endpointFailure("unavailable", e)
	}
	defer func() { <-o.registerLane }()
	s, e := o.snapshot()
	if e != nil {
		return endpointFailure("unavailable", e)
	}
	if p := knownRefusal(s, r); p != nil {
		return *p
	}
	if e = o.acquire(o.ctx, o.validation); e != nil {
		return endpointFailure("unavailable", e)
	}
	defer func() { <-o.validation }()
	expected := ""
	for _, record := range s.Records {
		if record.Binding.EditorIncarnation == r.EditorIncarnation && record.Binding.PID == r.PID {
			expected = record.Binding.ProcessIdentity
			break
		}
	}
	probed := o.probe(r, expected)
	if old := selected(s, r.DocumentID); probed.kind == "alive" && old != nil && sameTuple(*old, r) && old.File != probed.file {
		probed.kind = "mismatch"
	}
	if probed.kind != "alive" {
		changed, negativeErr := o.applyNegative(o.negativeEvent(s, r, probed))
		if negativeErr != nil {
			return endpointFailure("unavailable", negativeErr)
		}
		code := "unavailable"
		if probed.kind == "absent" {
			code = "not-registered"
		}
		p := endpointFailure(code, probed.err)
		p.Changed = changed
		return p
	}
	// Inspect an incumbent desktop's native birth once. A positively dead exact
	// process can be collected together with this registration in one publication.
	var collect *ProcessEvidence
	if old := selected(s, r.DocumentID); old != nil && old.Mode == "desktop" && r.Mode == "headless" {
		observed := probed.identity
		var readErr error
		if old.PID != r.PID {
			observed, readErr = o.cfg.Processes.ReadIdentity(old.PID)
		}
		classification := ClassifyEndpointProcess(old.ProcessIdentity, observed, readErr)
		if classification == EndpointProcessDead || classification == EndpointProcessReplaced {
			collect = &ProcessEvidence{EditorIncarnation: old.EditorIncarnation, PID: old.PID, ProcessIdentity: old.ProcessIdentity}
		}
	}
	generation := ""
	for _, record := range s.Records {
		b := record.Binding
		if b.EditorIncarnation == r.EditorIncarnation && b.DocumentID == r.DocumentID && b.AttachID == r.AttachID {
			generation = b.BindingGeneration
			break
		}
	}
	if generation == "" {
		generation, e = o.cfg.Random()
		if e != nil {
			return endpointFailure("unavailable", e)
		}
	}
	o.mu.Lock()
	if o.poison != nil {
		e = o.poison
		o.mu.Unlock()
		return endpointFailure("unavailable", e)
	}
	next := o.state
	changed := false
	previous := selected(next, r.DocumentID)
	if collect != nil {
		var p EndpointReply
		next, p, e = ApplyEndpointEvent(next, EndpointEvent{Collect: collect})
		if e != nil {
			o.mu.Unlock()
			return endpointFailure("unavailable", e)
		}
		changed = p.Changed
	}
	next, p, e := ApplyEndpointEvent(next, EndpointEvent{Register: &r, ProcessIdentity: probed.identity, BindingGeneration: generation, File: probed.file})
	if e != nil {
		o.mu.Unlock()
		return endpointFailure("unavailable", e)
	}
	changed = changed || p.Changed
	p.Changed = changed
	e = o.publishLocked(next, changed)
	o.mu.Unlock()
	if e != nil {
		return endpointFailure("unavailable", e)
	}
	if previous != nil && p.EditorEndpoint != nil && !exact(*previous, *p.EditorEndpoint) {
		_ = o.cfg.RPC.Superseded(o.ctx, *previous)
	}
	return p
}
func (o *EndpointOwner) Lookup(observer context.Context, namespace, document string) EndpointReply {
	if namespace != o.cfg.Store.Namespace || !validHex(document) {
		return endpointFailure("invalid", errors.New("invalid endpoint lookup"))
	}
	if e := o.begin(observer); e != nil {
		return endpointFailure("unavailable", e)
	}
	defer o.end()
	if e := o.acquire(observer, o.validation); e != nil {
		return endpointFailure("unavailable", e)
	}
	defer func() { <-o.validation }()
	s, e := o.snapshot()
	if e != nil {
		return endpointFailure("unavailable", e)
	}
	b := selected(s, document)
	if b == nil {
		return endpointFailure("ok", nil)
	}
	probed := o.probe(bindingRequest(*b), b.ProcessIdentity)
	if probed.kind == "alive" && probed.file != b.File {
		probed.kind = "mismatch"
	}
	if probed.kind != "alive" {
		// Negative process/editor evidence belongs to the captured generation.
		// Fence both its mutation and the resulting absence claim under one lock.
		o.mu.Lock()
		defer o.mu.Unlock()
		if o.poison != nil {
			return endpointFailure("unavailable", o.poison)
		}
		if o.closed || o.ctx.Err() != nil {
			return endpointFailure("unavailable", errors.New("endpoint owner closed"))
		}
		if current := selected(o.state, document); current != nil && !exact(*current, *b) {
			return endpointFailure("stale", nil)
		}
		changed, negativeErr := o.applyNegativeLocked(o.negativeEvent(s, bindingRequest(*b), probed))
		if negativeErr != nil {
			return endpointFailure("unavailable", negativeErr)
		}
		if probed.kind == "unknown" {
			return endpointFailure("unavailable", probed.err)
		}
		p := endpointFailure("ok", nil)
		p.Changed = changed
		return p
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.poison != nil {
		return endpointFailure("unavailable", o.poison)
	}
	if o.closed || o.ctx.Err() != nil {
		return endpointFailure("unavailable", errors.New("endpoint owner closed"))
	}
	current := selected(o.state, document)
	if current == nil || !exact(*current, *b) {
		return endpointFailure("stale", nil)
	}
	return EndpointReply{Version: 1, Code: "ok", EditorEndpoint: current}
}
func (o *EndpointOwner) Unregister(observer context.Context, r UnregisterRequest) EndpointReply {
	if e := ValidateEndpointRequest(r); e != nil {
		return endpointFailure("invalid", e)
	}
	if r.Namespace != o.cfg.Store.Namespace {
		return endpointFailure("invalid", errors.New("namespace mismatch"))
	}
	if e := o.begin(observer); e != nil {
		return endpointFailure("unavailable", e)
	}
	defer o.end()
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.poison != nil {
		return endpointFailure("unavailable", o.poison)
	}
	next, p, e := ApplyEndpointEvent(o.state, EndpointEvent{Unregister: &r})
	if e != nil {
		return endpointFailure("unavailable", e)
	}
	if e = o.publishLocked(next, p.Changed); e != nil {
		return endpointFailure("unavailable", e)
	}
	return p
}
func (o *EndpointOwner) Sweep(observer context.Context) error {
	if e := o.begin(observer); e != nil {
		return e
	}
	defer o.end()
	if e := o.acquire(observer, o.validation); e != nil {
		return e
	}
	defer func() { <-o.validation }()
	s, e := o.snapshot()
	if e != nil {
		return e
	}
	groups := map[int][]ProcessEvidence{}
	seen := map[ProcessEvidence]bool{}
	for _, r := range s.Records {
		b := r.Binding
		c := ProcessEvidence{EditorIncarnation: b.EditorIncarnation, PID: b.PID, ProcessIdentity: b.ProcessIdentity}
		if !seen[c] {
			seen[c] = true
			groups[b.PID] = append(groups[b.PID], c)
		}
	}
	pids := make([]int, 0, len(groups))
	for pid := range groups {
		pids = append(pids, pid)
	}
	sort.Ints(pids)
	if len(pids) == 0 {
		return nil
	}
	o.mu.Lock()
	start := o.sweepCursor % len(pids)
	count := len(pids)
	if count > 64 {
		count = 64
	}
	o.sweepCursor = (start + count) % len(pids)
	o.mu.Unlock()
	var dead []ProcessEvidence
	for i := 0; i < count; i++ {
		if e = o.ctx.Err(); e != nil {
			return e
		}
		pid := pids[(start+i)%len(pids)]
		observed, readErr := o.cfg.Processes.ReadIdentity(pid)
		for _, c := range groups[pid] {
			kind := ClassifyEndpointProcess(c.ProcessIdentity, observed, readErr)
			if kind == EndpointProcessDead || kind == EndpointProcessReplaced {
				dead = append(dead, c)
			}
		}
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.poison != nil {
		return o.poison
	}
	next := o.state
	changed := false
	for _, c := range dead {
		var p EndpointReply
		next, p, e = ApplyEndpointEvent(next, EndpointEvent{Collect: &c})
		if e != nil {
			return e
		}
		changed = changed || p.Changed
	}
	return o.publishLocked(next, changed)
}
func (o *EndpointOwner) Close() error {
	o.mu.Lock()
	o.closed = true
	o.cancel()
	o.mu.Unlock()
	o.wg.Wait()
	return nil
}
