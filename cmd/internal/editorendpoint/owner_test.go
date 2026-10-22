package editorendpoint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

type ownerProcess struct {
	mu       sync.Mutex
	identity string
	err      error
	reads    int
	pids     map[int]int
}

func (p *ownerProcess) ReadIdentity(pid int) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.reads++
	if p.pids == nil {
		p.pids = map[int]int{}
	}
	p.pids[pid]++
	return p.identity, p.err
}

type ownerEditorAttachment struct {
	binding EndpointBinding
	status  string
}

type ownerRPC struct {
	mu             sync.Mutex
	calls          int
	nilReply       bool
	replyFile      string
	held           chan struct{}
	started        chan struct{}
	notices        []EndpointBinding
	file           string
	attachments    map[string]ownerEditorAttachment
	noticeHeld     chan struct{}
	noticeStarted  chan struct{}
	noticeErr      error
	noticeCanceled chan struct{}
	noticeTeardown chan struct{}
}

func ownerAttachmentKey(b EndpointBinding) string {
	return b.EditorIncarnation + ":" + b.DocumentID + ":" + b.AttachID
}
func (r *ownerRPC) stage(q RegisterRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.attachments == nil {
		r.attachments = map[string]ownerEditorAttachment{}
	}
	b := EndpointBinding{Namespace: q.Namespace, DocumentID: q.DocumentID, EditorIncarnation: q.EditorIncarnation, AttachID: q.AttachID, PID: q.PID, Socket: q.Socket, Buffer: q.Buffer, Mode: q.Mode, File: r.file}
	r.attachments[ownerAttachmentKey(b)] = ownerEditorAttachment{binding: b, status: "pending"}
}
func (r *ownerRPC) accepted(b EndpointBinding) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.attachments[ownerAttachmentKey(b)] = ownerEditorAttachment{binding: b, status: "current"}
}
func (r *ownerRPC) attachment(b EndpointBinding) ownerEditorAttachment {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.attachments[ownerAttachmentKey(b)]
}
func (r *ownerRPC) Describe(ctx context.Context, q RegisterRequest) (*EditorDescription, error) {
	r.mu.Lock()
	r.calls++
	held, started, nilReply := r.held, r.started, r.nilReply
	d := EditorDescription{Namespace: q.Namespace, DocumentID: q.DocumentID, EditorIncarnation: q.EditorIncarnation, AttachID: q.AttachID, PID: q.PID, Socket: q.Socket, Buffer: q.Buffer, Mode: q.Mode, File: r.file}
	if r.attachments != nil {
		a, ok := r.attachments[ownerAttachmentKey(d)]
		if !ok || a.status == "retired" {
			nilReply = true
		} else {
			d = a.binding
		}
	}
	if r.replyFile != "" {
		d.File = r.replyFile
	}
	r.mu.Unlock()
	if started != nil {
		select {
		case started <- struct{}{}:
		default:
		}
	}
	if held != nil {
		select {
		case <-held:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if nilReply {
		return nil, nil
	}
	return &d, nil
}
func (r *ownerRPC) Superseded(ctx context.Context, b EndpointBinding) error {
	r.mu.Lock()
	r.notices = append(r.notices, b)
	held, started, failure, canceled, teardown := r.noticeHeld, r.noticeStarted, r.noticeErr, r.noticeCanceled, r.noticeTeardown
	r.mu.Unlock()
	if started != nil {
		select {
		case started <- struct{}{}:
		default:
		}
	}
	if held != nil {
		select {
		case <-held:
		case <-ctx.Done():
			if canceled != nil {
				close(canceled)
			}
			if teardown != nil {
				<-teardown
			}
			return ctx.Err()
		}
	}
	if failure != nil {
		return failure
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := ownerAttachmentKey(b)
	if a, ok := r.attachments[key]; ok && exact(a.binding, b) {
		a.status = "retired"
		r.attachments[key] = a
	}
	return nil
}

// ownerPublicationLedger records the actual encoded payload handed to durable
// IO, independently of model encoding and of the fixture's legacy write count.
type ownerPublicationLedger struct {
	mu    sync.Mutex
	bytes int64
	sizes []int
}
type ownerPublicationBytes struct {
	bytes int64
	sizes []int
}

func (l *ownerPublicationLedger) record(raw []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.bytes += int64(len(raw))
	l.sizes = append(l.sizes, len(raw))
}
func (l *ownerPublicationLedger) snapshot() ownerPublicationBytes {
	l.mu.Lock()
	defer l.mu.Unlock()
	return ownerPublicationBytes{bytes: l.bytes, sizes: append([]int(nil), l.sizes...)}
}

type ownerFixture struct {
	publications ownerPublicationLedger
	o            *EndpointOwner
	cfg          EndpointConfig
	rpc          *ownerRPC
	proc         *ownerProcess
	request      RegisterRequest
	writes       int
	fail         bool
	failAfter    bool
}

func newOwnerFixture(t *testing.T) *ownerFixture {
	t.Helper()
	dir, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	socket := filepath.Join(endpointTestRuntime(t), "sock")
	listener, e := net.Listen("unix", socket)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { listener.Close() })
	f := &ownerFixture{rpc: &ownerRPC{file: filepath.Join(dir, "doc.md")}, proc: &ownerProcess{identity: "birth"}}
	f.request = req(incA, docA, "1", "desktop")
	f.request.Namespace = dir
	f.request.Socket = socket
	f.cfg = EndpointConfig{Store: EndpointStore{Namespace: dir, Path: filepath.Join(dir, "state.json"), Stage: filepath.Join(dir, "stage.json"), WriteFile: func(path string, raw []byte, _ string) error {
		f.publications.record(raw)
		f.writes++
		if f.fail {
			return errors.New("uncertain write")
		}
		if e := os.WriteFile(path, raw, 0600); e != nil {
			return e
		}
		if f.failAfter {
			return errors.New("ack lost after publication")
		}
		return nil
	}}, RPC: f.rpc, Processes: f.proc, Resolve: func(context.Context, string) (string, error) {
		f.rpc.mu.Lock()
		defer f.rpc.mu.Unlock()
		return f.rpc.file, nil
	}, Random: func() (string, error) { return docA, nil }}
	f.o, e = NewEndpointOwner(context.Background(), f.cfg)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { f.o.Close() })
	return f
}
func TestOwnerPublishRetryRetirementAndNoProbeUnregister(t *testing.T) {
	f := newOwnerFixture(t)
	ctx := context.Background()
	p := f.o.Register(ctx, f.request)
	if p.Code != "ok" || !p.Changed || f.writes != 1 {
		t.Fatalf("first register %+v writes%d", p, f.writes)
	}
	b := *p.EditorEndpoint
	p = f.o.Register(ctx, f.request)
	if p.Code != "ok" || p.Changed || f.writes != 1 {
		t.Fatal("retry republished")
	}
	calls := f.rpc.calls
	u := UnregisterRequest{Version: 1, Namespace: b.Namespace, EditorIncarnation: b.EditorIncarnation, PID: b.PID, Entries: []UnregisterEntry{{DocumentID: b.DocumentID, AttachID: b.AttachID, BindingGeneration: b.BindingGeneration}}}
	p = f.o.Unregister(ctx, u)
	if !p.Changed || f.writes != 2 || f.rpc.calls != calls {
		t.Fatal("unregister was not pure one publication")
	}
	p = f.o.Register(ctx, f.request)
	if p.Code != "superseded" || f.rpc.calls != calls {
		t.Fatal("retired attempt probed or reacquired")
	}
}
func TestOwnerPublicationFailurePoisonsUntilReopen(t *testing.T) {
	f := newOwnerFixture(t)
	f.fail = true
	p := f.o.Register(context.Background(), f.request)
	if p.Code == "ok" {
		t.Fatal("acknowledged failed publication")
	}
	f.fail = false
	p = f.o.Register(context.Background(), f.request)
	if p.Code == "ok" {
		t.Fatal("uncertain owner reused")
	}
	f.o.Close()
	var e error
	f.o, e = NewEndpointOwner(context.Background(), f.cfg)
	if e != nil {
		t.Fatal(e)
	}
	if p = f.o.Register(context.Background(), f.request); p.Code != "ok" {
		t.Fatalf("reopen failed %+v", p)
	}
}

func TestOwnerRetryUsesNoNewEntropy(t *testing.T) {
	f := newOwnerFixture(t)
	if p := f.o.Register(context.Background(), f.request); p.Code != "ok" {
		t.Fatal(p)
	}
	f.o.cfg.Random = func() (string, error) { return "", errors.New("entropy unavailable") }
	if p := f.o.Register(context.Background(), f.request); p.Code != "ok" || p.Changed {
		t.Fatalf("exact retry required new entropy: %+v", p)
	}
}
func TestOwnerLookupRetiresMovedDocumentGeneration(t *testing.T) {
	f := newOwnerFixture(t)
	if p := f.o.Register(context.Background(), f.request); p.Code != "ok" {
		t.Fatal(p)
	}
	f.rpc.file = filepath.Join(f.request.Namespace, "moved.md")
	p := f.o.Lookup(context.Background(), f.request.Namespace, docA)
	if p.EditorEndpoint != nil || p.Code != "ok" {
		t.Fatalf("returned stale path: %+v", p)
	}
	p = f.o.Register(context.Background(), f.request)
	if p.Code != "superseded" {
		t.Fatalf("moved generation reacquired: %+v", p)
	}
}

func TestOwnerRegisterLaneAndObserverCancellation(t *testing.T) {
	f := newOwnerFixture(t)
	hold := make(chan struct{})
	started := make(chan struct{}, 4)
	f.rpc.held = hold
	f.rpc.started = started
	observer, cancel := context.WithCancel(context.Background())
	a := make(chan EndpointReply, 1)
	go func() { a <- f.o.Register(observer, f.request) }()
	<-started
	cancel()
	b := make(chan EndpointReply, 1)
	second := f.request
	second.AttachID = "2"
	go func() { b <- f.o.Register(context.Background(), second) }()
	select {
	case <-started:
		t.Fatal("second validation overtook held first admission")
	case <-time.After(20 * time.Millisecond):
	}
	close(hold)
	if p := <-a; p.Code != "ok" || !p.Changed {
		t.Fatalf("canceled observer dropped owned publication: %+v", p)
	}
	if p := <-b; p.Code != "ok" || !p.Changed {
		t.Fatal(p)
	}
	if f.writes != 2 {
		t.Fatalf("publications %d", f.writes)
	}
	f.rpc.mu.Lock()
	defer f.rpc.mu.Unlock()
	if len(f.rpc.notices) != 1 || f.rpc.notices[0].AttachID != "1" {
		t.Fatal("notice was not exact displaced tuple")
	}
}

// ownerHeldEvidence captures one negative observation before allowing the
// successor to publish; later reads observe the healthy restored authority.
type ownerHeldEvidence struct {
	mu               sync.Mutex
	kind             string
	captured         bool
	started, release chan struct{}
	file             string
	process          EndpointProcessSource
}

func (h *ownerHeldEvidence) hold(ctx context.Context, kind string) bool {
	h.mu.Lock()
	capture := h.kind == kind && !h.captured
	if capture {
		h.captured = true
	}
	h.mu.Unlock()
	if !capture {
		return false
	}
	close(h.started)
	select {
	case <-h.release:
	case <-ctx.Done():
	}
	return true
}
func (h *ownerHeldEvidence) ReadIdentity(pid int) (string, error) {
	if h.hold(context.Background(), "dead") {
		return "", syscall.ESRCH
	}
	return h.process.ReadIdentity(pid)
}
func (h *ownerHeldEvidence) resolve(ctx context.Context, _ string) (string, error) {
	if h.hold(ctx, "absent") {
		return "", ErrDocumentNotRegistered
	}
	return h.file, nil
}
func TestOwnerLateNegativeLookupCannotRetireSuccessor(t *testing.T) {
	for _, kind := range []string{"nil", "mismatch", "dead", "absent"} {
		t.Run(kind, func(t *testing.T) {
			f := newOwnerFixture(t)
			if p := f.o.Register(t.Context(), f.request); p.Code != "ok" {
				t.Fatal(p)
			}
			held := make(chan struct{})
			var release sync.Once
			t.Cleanup(func() { release.Do(func() { close(held) }) })
			started := make(chan struct{}, 1)
			if kind == "nil" || kind == "mismatch" {
				f.rpc.mu.Lock()
				f.rpc.held, f.rpc.started = held, started
				f.rpc.nilReply = kind == "nil"
				if kind == "mismatch" {
					f.rpc.replyFile = filepath.Join(f.request.Namespace, "other.md")
				}
				f.rpc.mu.Unlock()
			} else {
				evidence := &ownerHeldEvidence{kind: kind, started: started, release: held, file: f.rpc.file, process: f.proc}
				f.o.cfg.Processes = evidence
				f.o.cfg.Resolve = evidence.resolve
			}
			lookup := make(chan EndpointReply, 1)
			go func() { lookup <- f.o.Lookup(t.Context(), f.request.Namespace, docA) }()
			<-started
			f.rpc.mu.Lock()
			f.rpc.held, f.rpc.started = nil, nil
			f.rpc.nilReply, f.rpc.replyFile = false, ""
			f.rpc.mu.Unlock()
			second := f.request
			second.AttachID = "2"
			if kind == "dead" {
				second.EditorIncarnation = incB
				second.PID++
			}
			current := f.o.Register(t.Context(), second)
			if current.Code != "ok" {
				t.Fatal(current)
			}
			release.Do(func() { close(held) })
			if p := <-lookup; p.Code != "stale" || p.Changed || p.EditorEndpoint != nil {
				t.Fatalf("late %s falsely confirmed absence: %+v", kind, p)
			}
			if p := f.o.Lookup(t.Context(), f.request.Namespace, docA); p.Code != "ok" || p.EditorEndpoint == nil || !exact(*p.EditorEndpoint, *current.EditorEndpoint) {
				t.Fatalf("late %s erased successor: %+v", kind, p)
			}
			if f.writes != 2 {
				t.Fatalf("stale retirement republished: %d", f.writes)
			}
		})
	}
}

func TestOwnerCloseJoinsHeldNativeProbe(t *testing.T) {
	f := newOwnerFixture(t)
	f.rpc.held = make(chan struct{})
	f.rpc.started = make(chan struct{}, 1)
	done := make(chan EndpointReply, 1)
	go func() { done <- f.o.Register(context.Background(), f.request) }()
	<-f.rpc.started
	if e := f.o.Close(); e != nil {
		t.Fatal(e)
	}
	select {
	case p := <-done:
		if p.Code == "ok" {
			t.Fatal("canceled probe acknowledged")
		}
	default:
		t.Fatal("Close returned before admitted operation joined")
	}
}

// Each snapshot is O(S) in retained process/document pairs. M full-snapshot
// publications may write O(M*S) bytes; in particular initial registrations are
// allowed quadratic cumulative bytes. These bounds do not claim incremental IO.
func ownerPublicationBudget(got ownerPublicationBytes, writes, maxSnapshot int) error {
	if len(got.sizes) != writes {
		return fmt.Errorf("publications=%d want=%d", len(got.sizes), writes)
	}
	var sum int64
	for _, size := range got.sizes {
		if size <= 0 || size > maxSnapshot || size > MaxStateBytes {
			return fmt.Errorf("snapshot bytes=%d exceeds envelope=%d", size, maxSnapshot)
		}
		sum += int64(size)
	}
	if got.bytes != sum || got.bytes > int64(writes)*int64(maxSnapshot) {
		return fmt.Errorf("cumulative bytes=%d sum=%d budget=%d", got.bytes, sum, int64(writes)*int64(maxSnapshot))
	}
	return nil
}
func ownerReopenPlateau(initial, current, pairs, round, publications, firstPublications int) error {
	// Retained pairs stay fixed. Only canonical decimal AttachID and Revision
	// widths may grow across repeated attempts; no attempt history is retained.
	allowed := pairs*(len(strconv.Itoa(round))-1) + len(strconv.Itoa(publications)) - len(strconv.Itoa(firstPublications))
	if current < initial || current > initial+allowed {
		return fmt.Errorf("reopen snapshot=%d initial=%d allowed decimal growth=%d", current, initial, allowed)
	}
	return nil
}
func TestOwnerBatchOnePublicationAndPlateau(t *testing.T) {
	const rounds = 16
	// This deliberately generous absolute envelope covers the bounded fixture's
	// paths and scalar fields while the N->2N ratio and plateau checks below
	// reject superlinear snapshots and per-attempt retained growth.
	const headerEnvelope, recordEnvelope = 1024, 2048
	type measurement struct {
		initialSnapshot, finalSnapshot, writes int
		initialBytes, totalBytes               int64
	}
	measurements := map[int]measurement{}
	for _, count := range []int{8, 16} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			f := newOwnerFixture(t)
			var firstPlateau int
			var observed measurement
			for round := 1; round <= rounds; round++ {
				entries := make([]UnregisterEntry, 0, count)
				for i := 0; i < count; i++ {
					r := f.request
					r.DocumentID = fmt.Sprintf("%032x", i+100)
					r.AttachID = strconv.Itoa(round)
					before := f.publications.snapshot()
					p := f.o.Register(t.Context(), r)
					if p.Code != "ok" || !p.Changed || p.EditorEndpoint == nil {
						t.Fatal(p)
					}
					after := f.publications.snapshot()
					if len(after.sizes) != len(before.sizes)+1 || after.bytes <= before.bytes {
						t.Fatalf("registration publication bytes: before=%d/%d after=%d/%d", len(before.sizes), before.bytes, len(after.sizes), after.bytes)
					}
					if err := ownerPublicationBudget(after, (round-1)*(count+1)+i+1, headerEnvelope+count*recordEnvelope); err != nil {
						t.Fatal(err)
					}
					if round == 1 {
						if i > 0 {
							growth := after.sizes[i] - after.sizes[i-1]
							if growth < 0 || growth > recordEnvelope {
								t.Fatalf("per-pair snapshot growth=%d", growth)
							}
						}
						if i == count-1 {
							observed.initialSnapshot = after.sizes[len(after.sizes)-1]
							observed.initialBytes = after.bytes
						}
					}
					entries = append(entries, UnregisterEntry{DocumentID: r.DocumentID, AttachID: r.AttachID, BindingGeneration: p.EditorEndpoint.BindingGeneration})
					if again := f.o.Register(t.Context(), r); again.Code != "ok" || again.Changed {
						t.Fatalf("exact retry: %+v", again)
					}
					retry := f.publications.snapshot()
					if len(retry.sizes) != len(after.sizes) || retry.bytes != after.bytes {
						t.Fatal("retry added publication bytes")
					}
				}
				before, calls := f.publications.snapshot(), f.rpc.calls
				u := UnregisterRequest{Version: 1, Namespace: f.request.Namespace, EditorIncarnation: incA, PID: 42, Entries: entries}
				p := f.o.Unregister(t.Context(), u)
				after := f.publications.snapshot()
				if p.Code != "ok" || !p.Changed || len(after.sizes) != len(before.sizes)+1 || after.bytes-before.bytes != int64(after.sizes[len(after.sizes)-1]) || f.rpc.calls != calls {
					t.Fatal("batch did not make exactly one pure snapshot publication")
				}
				if err := ownerPublicationBudget(after, round*(count+1), headerEnvelope+count*recordEnvelope); err != nil {
					t.Fatal(err)
				}
				currentSize := after.sizes[len(after.sizes)-1]
				if round == 1 {
					firstPlateau = currentSize
				}
				if err := ownerReopenPlateau(firstPlateau, currentSize, count, round, len(after.sizes), count+1); err != nil {
					t.Fatal(err)
				}
				if p = f.o.Unregister(t.Context(), u); p.Code != "ok" || p.Changed {
					t.Fatalf("noop batch: %+v", p)
				}
				noop := f.publications.snapshot()
				if len(noop.sizes) != len(after.sizes) || noop.bytes != after.bytes {
					t.Fatal("noop batch added publication bytes")
				}
				f.o.Close()
				var err error
				f.o, err = NewEndpointOwner(t.Context(), f.cfg)
				if err != nil {
					t.Fatal(err)
				}
				saved, err := f.cfg.Store.Load()
				if err != nil || len(saved.Records) != count {
					t.Fatalf("reopened retention records=%d err=%v", len(saved.Records), err)
				}
				info, err := os.Stat(f.cfg.Store.Path)
				if err != nil || info.Size() != int64(currentSize) {
					t.Fatalf("writer/disk snapshot size diverged: err=%v", err)
				}
				if p = f.o.Lookup(t.Context(), f.request.Namespace, entries[0].DocumentID); p.Code != "ok" || p.EditorEndpoint != nil {
					t.Fatalf("reopened retirement lookup: %+v", p)
				}
				reopened := f.publications.snapshot()
				if len(reopened.sizes) != len(after.sizes) || reopened.bytes != after.bytes {
					t.Fatal("reopen/reload/empty lookup added publication bytes")
				}
			}
			all := f.publications.snapshot()
			observed.finalSnapshot = all.sizes[len(all.sizes)-1]
			observed.writes = len(all.sizes)
			observed.totalBytes = all.bytes
			if observed.writes != rounds*(count+1) || f.writes != observed.writes {
				t.Fatalf("publication count=%d legacy=%d", observed.writes, f.writes)
			}
			measurements[count] = observed
			t.Logf("N=%d rounds=%d initialSnapshot=%d initialPublicationBytes=%d finalSnapshot=%d totalPublicationBytes=%d publications=%d", count, rounds, observed.initialSnapshot, observed.initialBytes, observed.finalSnapshot, observed.totalBytes, observed.writes)
			// Negative controls mutate only the independent byte ledger/result:
			// duplicated durable IO and retained per-attempt data must trip bounds.
			duplicate := ownerPublicationBytes{bytes: all.bytes + int64(observed.finalSnapshot), sizes: append(append([]int(nil), all.sizes...), observed.finalSnapshot)}
			if ownerPublicationBudget(duplicate, observed.writes, headerEnvelope+count*recordEnvelope) == nil {
				t.Fatal("byte budget accepted duplicated writer accounting")
			}
			if ownerReopenPlateau(firstPlateau, observed.finalSnapshot+128, count, rounds, observed.writes, count+1) == nil {
				t.Fatal("plateau accepted growing retained attempt metadata")
			}
		})
	}
	small, large := measurements[8], measurements[16]
	if len(measurements) != 2 {
		return
	}
	if large.initialSnapshot*10 > small.initialSnapshot*22 {
		t.Fatalf("N->2N snapshot growth exceeds linear envelope: N8=%d N16=%d", small.initialSnapshot, large.initialSnapshot)
	}
	if large.initialBytes*10 > small.initialBytes*45 {
		t.Fatalf("initial cumulative publication growth exceeds quadratic envelope: N8=%d N16=%d", small.initialBytes, large.initialBytes)
	}
	if large.totalBytes*10 > small.totalBytes*45 {
		t.Fatalf("M*S cumulative publication envelope: N8=%d N16=%d", small.totalBytes, large.totalBytes)
	}
	if large.writes-rounds != 2*(small.writes-rounds) {
		t.Fatalf("N->2N publication counts inflated: N8=%d N16=%d", small.writes, large.writes)
	}
}

func TestOwnerSweepFairBoundedSharedPIDAndNoIdleRPC(t *testing.T) {
	f := newOwnerFixture(t)
	state := EndpointState{Version: 1, Namespace: f.request.Namespace, Revision: 1, Records: []EndpointRecord{}}
	for i := 0; i < 130; i++ {
		b := EndpointBinding{Namespace: state.Namespace, DocumentID: fmt.Sprintf("%032x", i+1), EditorIncarnation: fmt.Sprintf("%032x", i+1), AttachID: "1", PID: i + 100, ProcessIdentity: "birth", Socket: f.request.Socket, Buffer: 1, Mode: "desktop", BindingGeneration: docA, File: f.rpc.file}
		state.Records = append(state.Records, EndpointRecord{Binding: b, Status: "current"})
	}
	// A second retained process incarnation shares a PID: the native observation
	// is shared, while comparison still applies to each exact saved birth.
	extra := state.Records[0]
	extra.Binding.DocumentID = fmt.Sprintf("%032x", 999)
	extra.Binding.EditorIncarnation = fmt.Sprintf("%032x", 999)
	extra.Status = "retired"
	state.Records = append(state.Records, extra)
	if e := f.cfg.Store.Publish(state); e != nil {
		t.Fatal(e)
	}
	f.o.Close()
	var e error
	f.o, e = NewEndpointOwner(context.Background(), f.cfg)
	if e != nil {
		t.Fatal(e)
	}
	before := f.writes
	for pass := 0; pass < 3; pass++ {
		previous := f.proc.reads
		if e = f.o.Sweep(context.Background()); e != nil {
			t.Fatal(e)
		}
		if f.proc.reads-previous != 64 {
			t.Fatalf("pass read %d PIDs", f.proc.reads-previous)
		}
	}
	if len(f.proc.pids) != 130 || f.rpc.calls != 0 || f.writes != before {
		t.Fatalf("sweep unfair or polled RPC/rewrote: %d %d %d", len(f.proc.pids), f.rpc.calls, f.writes)
	}
	f.proc.err = syscall.ESRCH
	if e = f.o.Sweep(context.Background()); e != nil {
		t.Fatal(e)
	}
	if f.writes != before+1 {
		t.Fatal("dead process pass did not make one publication")
	}
	saved, e := f.cfg.Store.Load()
	if e != nil || len(saved.Records) >= len(state.Records) {
		t.Fatal("dead processes retained")
	}
}
func TestOwnerShutdownFenceBeforePublication(t *testing.T) {
	f := newOwnerFixture(t)
	f.o.cfg.Random = func() (string, error) { f.o.cancel(); return docA, nil }
	p := f.o.Register(context.Background(), f.request)
	if p.Code == "ok" || f.writes != 0 {
		t.Fatalf("published beyond canceled owner fence: %+v %d", p, f.writes)
	}
}
func TestOwnerUnknownResolverRetainsButPositiveAbsenceRetires(t *testing.T) {
	f := newOwnerFixture(t)
	if p := f.o.Register(context.Background(), f.request); p.Code != "ok" {
		t.Fatal(p)
	}
	f.o.cfg.Resolve = func(context.Context, string) (string, error) { return "", os.ErrPermission }
	p := f.o.Lookup(context.Background(), f.request.Namespace, docA)
	if p.Code != "unavailable" || p.Changed || f.writes != 1 {
		t.Fatal("unknown resolver result changed authority")
	}
	f.o.cfg.Resolve = func(context.Context, string) (string, error) { return "", ErrDocumentNotRegistered }
	p = f.o.Lookup(context.Background(), f.request.Namespace, docA)
	if p.Code != "ok" || !p.Changed || f.writes != 2 {
		t.Fatalf("positive absence did not report durable retirement: %+v", p)
	}
	p = f.o.Register(context.Background(), f.request)
	if p.Code != "superseded" {
		t.Fatal("positive absence forgotten")
	}
}
func TestOwnerUncertainCommittedWriteReopenPreservesLostAckRetry(t *testing.T) {
	f := newOwnerFixture(t)
	f.failAfter = true
	p := f.o.Register(context.Background(), f.request)
	if p.Code == "ok" {
		t.Fatal("uncertain committed write acknowledged")
	}
	f.failAfter = false
	f.o.Close()
	var e error
	f.o, e = NewEndpointOwner(context.Background(), f.cfg)
	if e != nil {
		t.Fatal(e)
	}
	before := f.writes
	p = f.o.Register(context.Background(), f.request)
	if p.Code != "ok" || p.Changed || f.writes != before {
		t.Fatalf("lost-ack retry changed authority: %+v", p)
	}
}
func TestOwnerValidationConcurrencyBound(t *testing.T) {
	f := newOwnerFixture(t)
	if p := f.o.Register(context.Background(), f.request); p.Code != "ok" {
		t.Fatal(p)
	}
	hold := make(chan struct{})
	started := make(chan struct{}, 8)
	f.rpc.held = hold
	f.rpc.started = started
	done := make(chan EndpointReply, 5)
	for i := 0; i < 4; i++ {
		go func() { done <- f.o.Lookup(context.Background(), f.request.Namespace, docA) }()
	}
	for i := 0; i < 4; i++ {
		<-started
	}
	r := f.request
	r.AttachID = "2"
	go func() { done <- f.o.Register(context.Background(), r) }()
	select {
	case <-started:
		t.Fatal("more than four native validations entered")
	case <-time.After(20 * time.Millisecond):
	}
	close(hold)
	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestOwnerConfirmedAbsenceReturnsSuccessfulNull(t *testing.T) {
	for _, published := range []bool{false, true} {
		t.Run(fmt.Sprint(published), func(t *testing.T) {
			f := newOwnerFixture(t)
			if published {
				if p := f.o.Register(t.Context(), f.request); p.Code != "ok" {
					t.Fatal(p)
				}
				f.rpc.nilReply = true
			}
			before := f.writes
			p := f.o.Lookup(t.Context(), f.request.Namespace, docA)
			if p.Version != 1 || p.Code != "ok" || p.Error != "" || p.EditorEndpoint != nil || p.Changed != published {
				t.Fatalf("positive absence: %+v", p)
			}
			if f.writes-before != map[bool]int{false: 0, true: 1}[published] {
				t.Fatalf("absence publications=%d", f.writes-before)
			}
		})
	}
}

func TestOwnerMissingRegisteredDocumentIsNotRegistered(t *testing.T) {
	f := newOwnerFixture(t)
	f.o.cfg.Resolve = func(context.Context, string) (string, error) {
		return "", fmt.Errorf("resolver: %w", ErrDocumentNotRegistered)
	}
	p := f.o.Register(t.Context(), f.request)
	if p.Version != 1 || p.Code != "not-registered" || p.Changed || p.EditorEndpoint != nil || f.writes != 0 || f.rpc.calls != 0 || f.proc.reads != 0 {
		t.Fatalf("missing document: %+v writes=%d rpc=%d process=%d", p, f.writes, f.rpc.calls, f.proc.reads)
	}
}

func TestOwnerPublicationErrorWireOutcomes(t *testing.T) {
	for _, operation := range []string{"register", "lookup", "unregister", "register-cleanup"} {
		for _, after := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/after=%v", operation, after), func(t *testing.T) {
				f := newOwnerFixture(t)
				var binding *EndpointBinding
				if operation != "register" {
					p := f.o.Register(t.Context(), f.request)
					if p.Code != "ok" {
						t.Fatal(p)
					}
					binding = p.EditorEndpoint
				}
				if after {
					f.failAfter = true
				} else {
					f.fail = true
				}
				var p EndpointReply
				switch operation {
				case "register":
					p = f.o.Register(t.Context(), f.request)
				case "lookup":
					f.rpc.nilReply = true
					p = f.o.Lookup(t.Context(), f.request.Namespace, docA)
				case "register-cleanup":
					f.o.cfg.Resolve = func(context.Context, string) (string, error) { return "", ErrDocumentNotRegistered }
					p = f.o.Register(t.Context(), f.request)
				case "unregister":
					p = f.o.Unregister(t.Context(), UnregisterRequest{Version: 1, Namespace: binding.Namespace, EditorIncarnation: binding.EditorIncarnation, PID: binding.PID, Entries: []UnregisterEntry{{DocumentID: binding.DocumentID, AttachID: binding.AttachID, BindingGeneration: binding.BindingGeneration}}})
				}
				if p.Version != 1 || p.Code != "uncertain" || p.Changed || p.EditorEndpoint != nil {
					t.Fatalf("failed publication: %+v", p)
				}
				writes := f.writes
				f.fail = false
				f.failAfter = false
				if p = f.o.Lookup(t.Context(), f.request.Namespace, docA); p.Code != "unavailable" || f.writes != writes {
					t.Fatalf("poisoned lookup: %+v writes=%d/%d", p, f.writes, writes)
				}
				if p = f.o.Register(t.Context(), f.request); p.Code != "unavailable" || f.writes != writes {
					t.Fatalf("poisoned registration: %+v writes=%d/%d", p, f.writes, writes)
				}
			})
		}
	}
}

func TestOwnerEncodedCapacityRefusalKeepsAuthorityUsable(t *testing.T) {
	f := newOwnerFixture(t)
	first := f.o.Register(t.Context(), f.request)
	if first.Code != "ok" {
		t.Fatal(first)
	}
	state, err := f.cfg.Store.Load()
	if err != nil {
		t.Fatal(err)
	}
	state.Unknown = map[string]json.RawMessage{"retained": json.RawMessage(`""`)}
	raw, err := EncodeEndpointState(state)
	if err != nil {
		t.Fatal(err)
	}
	state.Unknown["retained"] = json.RawMessage(`"` + strings.Repeat("x", MaxStateBytes-len(raw)-64) + `"`)
	raw, err = EncodeEndpointState(state)
	if err != nil || len(raw) != MaxStateBytes-64 {
		t.Fatalf("near-capacity state bytes=%d err=%v", len(raw), err)
	}
	if err = f.cfg.Store.Publish(state); err != nil {
		t.Fatal(err)
	}
	f.o.Close()
	f.o, err = NewEndpointOwner(t.Context(), f.cfg)
	if err != nil {
		t.Fatal(err)
	}
	before := f.writes
	incoming := f.request
	incoming.DocumentID = strings.Repeat("b", 32)
	incoming.Buffer++
	p := f.o.Register(t.Context(), incoming)
	if p.Code != "capacity" || p.Changed || p.EditorEndpoint != nil || f.writes != before {
		t.Fatalf("encoded-capacity refusal Code=%s Changed=%v writes=%d/%d", p.Code, p.Changed, f.writes, before)
	}
	saved, err := f.cfg.Store.Load()
	if err != nil || saved.Revision != state.Revision || len(saved.Records) != 1 {
		t.Fatalf("refusal changed authority revision=%d records=%d err=%v", saved.Revision, len(saved.Records), err)
	}
	if p = f.o.Lookup(t.Context(), f.request.Namespace, docA); p.Code != "ok" || p.EditorEndpoint == nil || p.EditorEndpoint.BindingGeneration != first.EditorEndpoint.BindingGeneration {
		t.Fatalf("refusal poisoned lookup: Code=%s", p.Code)
	}
	if err = f.o.Sweep(t.Context()); err != nil {
		t.Fatalf("refusal poisoned live sweep: %v", err)
	}
	b := first.EditorEndpoint
	p = f.o.Unregister(t.Context(), UnregisterRequest{Version: 1, Namespace: b.Namespace, EditorIncarnation: b.EditorIncarnation, PID: b.PID, Entries: []UnregisterEntry{{DocumentID: b.DocumentID, AttachID: b.AttachID, BindingGeneration: b.BindingGeneration}}})
	if p.Code != "ok" || !p.Changed || f.writes != before+1 {
		t.Fatalf("refusal poisoned unregister: Code=%s Changed=%v writes=%d", p.Code, p.Changed, f.writes)
	}
	f.proc.err = syscall.ESRCH
	if err = f.o.Sweep(t.Context()); err != nil {
		t.Fatalf("refusal poisoned collection: %v", err)
	}
	saved, err = f.cfg.Store.Load()
	if err != nil || len(saved.Records) != 0 || f.writes != before+2 {
		t.Fatalf("post-refusal collection records=%d writes=%d err=%v", len(saved.Records), f.writes, err)
	}
	if string(saved.Unknown["retained"]) != string(state.Unknown["retained"]) {
		t.Fatal("capacity handling changed retained unknown bytes")
	}
}

func TestOwnerSupersededNoticeFollowsPublicationAndFailurePreservesRetry(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprintf("failed=%v", fail), func(t *testing.T) {
			f := newOwnerFixture(t)
			f.rpc.stage(f.request)
			first := f.o.Register(t.Context(), f.request)
			if first.Code != "ok" || first.EditorEndpoint == nil {
				t.Fatal(first)
			}
			f.rpc.accepted(*first.EditorEndpoint)
			second := f.request
			second.AttachID = "2"
			f.rpc.stage(second)
			held, started := make(chan struct{}), make(chan struct{}, 1)
			var release sync.Once
			t.Cleanup(func() { release.Do(func() { close(held) }) })
			f.rpc.mu.Lock()
			f.rpc.noticeHeld, f.rpc.noticeStarted = held, started
			if fail {
				f.rpc.noticeErr = errors.New("editor notice failed")
			}
			f.rpc.mu.Unlock()
			completed := make(chan EndpointReply, 1)
			go func() { completed <- f.o.Register(t.Context(), second) }()
			<-started
			// Delivery has not completed, yet both the durable record and a fresh
			// authoritative lookup must already select the pending successor.
			saved, err := f.cfg.Store.Load()
			if err != nil {
				t.Fatal(err)
			}
			durable := selected(saved, docA)
			if durable == nil || durable.AttachID != "2" || saved.Revision != 2 {
				t.Fatalf("notice preceded publication: revision=%d binding=%+v", saved.Revision, durable)
			}
			if p := f.o.Lookup(t.Context(), second.Namespace, second.DocumentID); p.Code != "ok" || p.EditorEndpoint == nil || !exact(*p.EditorEndpoint, *durable) {
				t.Fatalf("held notice hid successor: %+v", p)
			}
			if old := f.rpc.attachment(*first.EditorEndpoint); old.status != "current" {
				t.Fatalf("held notice changed previous local attachment: %s", old.status)
			}
			if pending := f.rpc.attachment(*durable); pending.status != "pending" {
				t.Fatalf("unacknowledged attachment status=%s", pending.status)
			}
			select {
			case p := <-completed:
				t.Fatalf("registration abandoned held notice: %+v", p)
			default:
			}
			release.Do(func() { close(held) })
			reply := <-completed
			if reply.Code != "ok" || !reply.Changed || reply.EditorEndpoint == nil || !exact(*reply.EditorEndpoint, *durable) {
				t.Fatalf("notification changed committed result: %+v", reply)
			}
			wantOld := "retired"
			if fail {
				wantOld = "current"
			}
			if old := f.rpc.attachment(*first.EditorEndpoint); old.status != wantOld {
				t.Fatalf("notification result local status=%s want=%s", old.status, wantOld)
			}
			if pending := f.rpc.attachment(*durable); pending.status != "pending" {
				t.Fatalf("old notification affected successor: %s", pending.status)
			}
			f.rpc.accepted(*reply.EditorEndpoint)
			retry := f.o.Register(t.Context(), second)
			if retry.Code != "ok" || retry.Changed || retry.EditorEndpoint == nil || !exact(*retry.EditorEndpoint, *durable) {
				t.Fatalf("notice failure disrupted exact retry: %+v", retry)
			}
			f.rpc.mu.Lock()
			notices := len(f.rpc.notices)
			f.rpc.mu.Unlock()
			if f.writes != 2 || notices != 1 {
				t.Fatalf("retry repeated publication/notification: writes=%d notices=%d", f.writes, notices)
			}
		})
	}
}

func TestOwnerCloseCancelsAndJoinsHeldSupersededNotice(t *testing.T) {
	f := newOwnerFixture(t)
	f.rpc.stage(f.request)
	first := f.o.Register(t.Context(), f.request)
	if first.Code != "ok" || first.EditorEndpoint == nil {
		t.Fatal(first)
	}
	f.rpc.accepted(*first.EditorEndpoint)
	second := f.request
	second.AttachID = "2"
	f.rpc.stage(second)
	held, started := make(chan struct{}), make(chan struct{}, 1)
	canceled, teardown := make(chan struct{}), make(chan struct{})
	var release sync.Once
	t.Cleanup(func() { release.Do(func() { close(teardown) }) })
	f.rpc.mu.Lock()
	f.rpc.noticeHeld, f.rpc.noticeStarted = held, started
	f.rpc.noticeCanceled, f.rpc.noticeTeardown = canceled, teardown
	f.rpc.mu.Unlock()
	completed := make(chan EndpointReply, 1)
	go func() { completed <- f.o.Register(t.Context(), second) }()
	<-started
	saved, err := f.cfg.Store.Load()
	if err != nil {
		t.Fatal(err)
	}
	successor := selected(saved, docA)
	if successor == nil || successor.AttachID != "2" {
		t.Fatal("held notification began before publication")
	}
	closed := make(chan error, 1)
	go func() { closed <- f.o.Close() }()
	<-canceled
	select {
	case err := <-closed:
		t.Fatalf("Close returned before notification teardown: %v", err)
	default:
	}
	select {
	case p := <-completed:
		t.Fatalf("operation returned before notification teardown: %+v", p)
	default:
	}
	release.Do(func() { close(teardown) })
	if err := <-closed; err != nil {
		t.Fatal(err)
	}
	reply := <-completed
	if reply.Code != "ok" || reply.EditorEndpoint == nil || !exact(*reply.EditorEndpoint, *successor) {
		t.Fatalf("canceled notification lost committed successor: %+v", reply)
	}
	saved, err = f.cfg.Store.Load()
	if err != nil || saved.Revision != 2 || selected(saved, docA) == nil || !exact(*selected(saved, docA), *successor) {
		t.Fatalf("shutdown rewrote committed publication: revision=%d err=%v", saved.Revision, err)
	}
	if f.rpc.attachment(*first.EditorEndpoint).status != "current" || f.rpc.attachment(*successor).status != "pending" {
		t.Fatal("canceled old notification changed local attachment ownership")
	}
}

func TestOwnerWriterErrorCannotMasqueradeAsCandidateCapacity(t *testing.T) {
	f := newOwnerFixture(t)
	f.o.cfg.Store.WriteFile = func(string, []byte, string) error { return ErrEndpointCapacity }
	if p := f.o.Register(t.Context(), f.request); p.Code != "uncertain" {
		t.Fatalf("attempted writer failure Code=%s", p.Code)
	}
	if p := f.o.Lookup(t.Context(), f.request.Namespace, docA); p.Code != "unavailable" {
		t.Fatalf("poison classified as candidate refusal: Code=%s", p.Code)
	}
}

func TestOwnerPrewriterStoreValidationIsUnavailableWithoutPoison(t *testing.T) {
	f := newOwnerFixture(t)
	if err := os.Chmod(f.request.Namespace, 0755); err != nil {
		t.Fatal(err)
	}
	p := f.o.Register(t.Context(), f.request)
	if err := os.Chmod(f.request.Namespace, 0700); err != nil {
		t.Fatal(err)
	}
	if p.Code != "unavailable" || p.Changed || f.writes != 0 {
		t.Fatalf("prewriter validation: Code=%s Changed=%v writes=%d", p.Code, p.Changed, f.writes)
	}
	if p = f.o.Lookup(t.Context(), f.request.Namespace, docA); p.Code != "ok" || p.EditorEndpoint != nil {
		t.Fatalf("prewriter validation poisoned empty authority: Code=%s", p.Code)
	}
	if p = f.o.Register(t.Context(), f.request); p.Code != "ok" || !p.Changed || f.writes != 1 {
		t.Fatalf("prewriter validation poisoned admission: Code=%s Changed=%v writes=%d", p.Code, p.Changed, f.writes)
	}
}
