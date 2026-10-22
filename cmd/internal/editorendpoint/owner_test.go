package editorendpoint

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
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

type ownerRPC struct {
	mu       sync.Mutex
	calls    int
	nilReply bool
	held     chan struct{}
	started  chan struct{}
	notices  []EndpointBinding
	file     string
}

func (r *ownerRPC) Describe(ctx context.Context, q RegisterRequest) (*EditorDescription, error) {
	r.mu.Lock()
	r.calls++
	held, started, nilReply := r.held, r.started, r.nilReply
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
	return &EditorDescription{Namespace: q.Namespace, DocumentID: q.DocumentID, EditorIncarnation: q.EditorIncarnation, AttachID: q.AttachID, PID: q.PID, Socket: q.Socket, Buffer: q.Buffer, Mode: q.Mode, File: r.file}, nil
}
func (r *ownerRPC) Superseded(_ context.Context, b EndpointBinding) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notices = append(r.notices, b)
	return nil
}

type ownerFixture struct {
	o         *EndpointOwner
	cfg       EndpointConfig
	rpc       *ownerRPC
	proc      *ownerProcess
	request   RegisterRequest
	writes    int
	fail      bool
	failAfter bool
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
	}}, RPC: f.rpc, Processes: f.proc, Resolve: func(context.Context, string) (string, error) { return f.rpc.file, nil }, Random: func() (string, error) { return docA, nil }}
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
	if p.EditorEndpoint != nil || p.Code != "absent" {
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
func TestOwnerLateNegativeLookupCannotRetireSuccessor(t *testing.T) {
	f := newOwnerFixture(t)
	if p := f.o.Register(context.Background(), f.request); p.Code != "ok" {
		t.Fatal(p)
	}
	held := make(chan struct{})
	started := make(chan struct{}, 1)
	f.rpc.mu.Lock()
	f.rpc.held = held
	f.rpc.started = started
	f.rpc.nilReply = true
	f.rpc.mu.Unlock()
	lookup := make(chan EndpointReply, 1)
	go func() { lookup <- f.o.Lookup(context.Background(), f.request.Namespace, docA) }()
	<-started
	f.rpc.mu.Lock()
	f.rpc.held = nil
	f.rpc.started = nil
	f.rpc.nilReply = false
	f.rpc.mu.Unlock()
	second := f.request
	second.AttachID = "2"
	if p := f.o.Register(context.Background(), second); p.Code != "ok" {
		t.Fatal(p)
	}
	close(held)
	<-lookup
	if p := f.o.Lookup(context.Background(), f.request.Namespace, docA); p.Code != "ok" || p.EditorEndpoint.AttachID != "2" {
		t.Fatalf("late lookup erased successor: %+v", p)
	}
	if f.writes != 2 {
		t.Fatalf("stale retirement republished: %d", f.writes)
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
func TestOwnerBatchOnePublicationAndPlateau(t *testing.T) {
	for _, count := range []int{8, 16} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			f := newOwnerFixture(t)
			for round := 1; round <= 4; round++ {
				entries := make([]UnregisterEntry, 0, count)
				for i := 0; i < count; i++ {
					r := f.request
					r.DocumentID = fmt.Sprintf("%032x", i+100)
					r.AttachID = strconv.Itoa(round)
					p := f.o.Register(context.Background(), r)
					if p.Code != "ok" {
						t.Fatal(p)
					}
					entries = append(entries, UnregisterEntry{DocumentID: r.DocumentID, AttachID: r.AttachID, BindingGeneration: p.EditorEndpoint.BindingGeneration})
					before := f.writes
					if again := f.o.Register(context.Background(), r); again.Changed || f.writes != before {
						t.Fatal("retry rewrote state")
					}
				}
				before, calls := f.writes, f.rpc.calls
				u := UnregisterRequest{Version: 1, Namespace: f.request.Namespace, EditorIncarnation: incA, PID: 42, Entries: entries}
				p := f.o.Unregister(context.Background(), u)
				if !p.Changed || f.writes != before+1 || f.rpc.calls != calls {
					t.Fatal("batch did not make one pure publication")
				}
				if p = f.o.Unregister(context.Background(), u); p.Changed || f.writes != before+1 {
					t.Fatal("noop batch published")
				}
				s, e := f.cfg.Store.Load()
				if e != nil || len(s.Records) != count {
					t.Fatalf("retention grew %d %v", len(s.Records), e)
				}
			}
		})
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
	if p.Code != "absent" || !p.Changed || f.writes != 2 {
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
