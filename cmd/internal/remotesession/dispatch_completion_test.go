package remotesession

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

// completionRelease exposes an event rendezvous, never a timing assumption.
// Accepted work may need to report a result even after cancellation; Join must
// not claim that such a producer disappeared while its result is held.
type completionRelease struct {
	sequence uint64
	release  chan struct{}
	once     sync.Once
}

func newCompletionRelease(sequence uint64) *completionRelease {
	return &completionRelease{sequence: sequence, release: make(chan struct{})}
}
func (r *completionRelease) open() { r.once.Do(func() { close(r.release) }) }

type completionBackend struct {
	*fixtureDocuments
	jobs        chan fixtureDocumentOperation
	holdResult  bool
	accepted    chan *completionRelease
	joinEntered chan struct{}
	stop        context.CancelFunc
	done        chan struct{}
	effects     atomic.Int64
	handoffs    atomic.Int64
}

func (f *completionBackend) Dispatch(ctx context.Context, request DocumentRequest, permit Permit) (<-chan DocumentResult, bool) {
	result := make(chan DocumentResult, 1)
	select {
	case <-f.done:
		return nil, false
	default:
	}
	select {
	case f.jobs <- fixtureDocumentOperation{ctx, request, permit, result}:
		f.handoffs.Add(1)
		return result, true
	default:
		return nil, false
	}
}
func (f *completionBackend) work(ctx context.Context) {
	defer close(f.done)
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-f.jobs:
			snapshot, err := f.Read(job.ctx, job.request.Document)
			if err != nil || snapshot.Revision != job.request.BaseRevision || !job.permit.EffectDoor(job.request.Document, func() bool { f.effects.Add(1); return true }) {
				job.result <- DocumentResult{Outcome: "refused"}
				continue
			}
			if f.holdResult {
				held := newCompletionRelease(job.request.Sequence)
				f.accepted <- held
				<-held.release
			}
			job.result <- DocumentResult{Outcome: "applied", Revision: job.request.BaseRevision, ReceiptID: fmt.Sprintf("receipt-%d", job.request.Sequence)}
		}
	}
}
func (f *completionBackend) Join(ctx context.Context) error {
	select {
	case f.joinEntered <- struct{}{}:
	default:
	}
	f.stop()
	select {
	case <-f.done:
		return f.fixtureDocuments.Join(ctx)
	case <-ctx.Done():
		return ctx.Err()
	}
}

type completionFixture struct {
	s              *Supervisor
	e              *remoteEpoch
	session        sessionRef
	identity       AccessIdentity
	cookie         string
	backend        *completionBackend
	reset          func() error
	auditEntered   chan *completionRelease
	intents        atomic.Int64
	results        atomic.Int64
	resultAttempts atomic.Int64
	failResult     bool
}

func newCompletionFixture(t *testing.T, stage string) *completionFixture {
	t.Helper()
	s, reset := supervisorFixture(t, &broadcast.FakeTunnel{})
	documents := newFixtureDocuments(t, DocumentSnapshot{Ref: DocumentRef{ID: originDocA, Generation: "editor-1"}, Revision: "r1", Text: "stateful document"})
	ctx, cancel := context.WithCancel(t.Context())
	backend := &completionBackend{fixtureDocuments: documents, jobs: make(chan fixtureDocumentOperation, 1), holdResult: stage == "backend-result", accepted: make(chan *completionRelease, 1), joinEntered: make(chan struct{}, 1), stop: cancel, done: make(chan struct{})}
	s.backend = backend
	go backend.work(ctx)
	t.Cleanup(func() { cancel(); <-backend.done })
	if err := s.On(); err != nil {
		t.Fatal(err)
	}
	awaitRemote(t, s, "on")
	s.gate.mu.Lock()
	e := s.epoch
	s.gate.mu.Unlock()
	identity := AccessIdentity{issuer: "issuer", subject: "human", audience: "app", email: "human@example.com", expires: time.Now().Add(time.Hour)}
	credentials, err := e.sessions.Handshake(identity, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	session, err := e.sessions.Admit(credentials.Cookie, credentials.CSRF, identity, time.Now(), true)
	if err != nil {
		t.Fatal(err)
	}
	fixture := &completionFixture{s: s, e: e, session: session, identity: identity, cookie: credentials.Cookie, backend: backend, reset: reset, auditEntered: make(chan *completionRelease, 1)}
	original := s.audit.appendLine
	s.audit.appendLine = func(path string, raw []byte) error {
		record, err := decodeAuditRecord(raw)
		if err != nil {
			return err
		}
		if record.kind == "result" {
			fixture.resultAttempts.Add(1)
		}
		if record.kind == "result" && stage == "result-audit" {
			held := newCompletionRelease(record.Sequence)
			fixture.auditEntered <- held
			<-held.release
		}
		if record.kind == "result" && fixture.failResult {
			return errors.New("controlled result publication failure")
		}
		if err := original(path, raw); err != nil {
			return err
		}
		if record.kind == "intent" {
			fixture.intents.Add(1)
		} else {
			fixture.results.Add(1)
		}
		return nil
	}
	return fixture
}
func (f *completionFixture) dispatch(ctx context.Context, request DocumentRequest) (DocumentResult, error) {
	return f.s.Dispatch(ctx, f.e, f.session, f.identity, "192.0.2.9", request)
}
func (f *completionFixture) paused(t *testing.T, stage string) *completionRelease {
	t.Helper()
	source := f.backend.accepted
	if stage == "result-audit" {
		source = f.auditEntered
	}
	select {
	case held := <-source:
		t.Cleanup(held.open)
		return held
	case <-t.Context().Done():
		t.Fatal("completion rendezvous not reached")
		return nil
	}
}
func (f *completionFixture) assertReplay(t *testing.T, request DocumentRequest, expected DocumentResult, uncertain bool) {
	t.Helper()
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	// A retired session cannot use Dispatch again. Inspect the actual owner's
	// completed replay value under its gate instead of granting it a new session.
	f.s.gate.mu.Lock()
	admission, err := f.session.operations.replay.Admit(request.Sequence, raw)
	f.s.gate.mu.Unlock()
	var result DocumentResult
	if err != nil || admission.kind != replayCached || admission.uncertain != uncertain {
		t.Fatalf("replay kind=%d uncertain=%v err=%v", admission.kind, admission.uncertain, err)
	}
	if err := json.Unmarshal(admission.result, &result); err != nil || result != expected {
		t.Fatalf("replay=%+v expected=%+v err=%v", result, expected, err)
	}
}
func (f *completionFixture) assertCounts(t *testing.T, n int) {
	t.Helper()
	if effects, handoffs := f.backend.effects.Load(), f.backend.handoffs.Load(); effects != int64(n) || handoffs != int64(n) {
		t.Fatalf("effects=%d handoffs=%d expected%d", effects, handoffs, n)
	}
	if intents, results := f.intents.Load(), f.results.Load(); intents != int64(n) || results != int64(n) {
		t.Fatalf("durable intents=%d results=%d expected%d", intents, results, n)
	}
	_, rows, err := readAuditSegment(f.s.paths.Audit())
	if err != nil || len(rows) != 2*n {
		t.Fatalf("durable rows=%d expected%d err=%v", len(rows), 2*n, err)
	}
}

// The applicable completion cuts are genuinely different: before the backend
// publishes a result; after the result is known but its audit is unpublished;
// and after both are settled. Revoke at each cut, without sleeps or shortened
// deadlines, and assert what is known independently of whether the observer lives.
func TestRemoteDispatchCompletionRevocationOrderings(t *testing.T) {
	for _, stage := range []string{"backend-result", "result-audit", "settled"} {
		for _, event := range []string{"observer-cancel", "logout", "kill"} {
			t.Run(stage+"/"+event, func(t *testing.T) {
				f := newCompletionFixture(t, stage)
				request := submitRequest(1)
				observer, cancelObserver := context.WithCancel(t.Context())
				defer cancelObserver()
				stopEpoch := context.AfterFunc(f.e.ctx, cancelObserver)
				defer stopEpoch()
				observed, closeObserved := documentSessionContext(observer, f.session)
				defer closeObserved()
				completed := make(chan dispatchCompletion, 1)
				go func() { result, err := f.dispatch(observed, request); completed <- dispatchCompletion{result, err} }()
				var held *completionRelease
				var first dispatchCompletion
				if stage == "settled" {
					select {
					case first = <-completed:
					case <-t.Context().Done():
						t.Fatal("settled completion absent")
					}
					f.e.requests.Wait()
					if first.err != nil || first.result.Outcome != "applied" || first.result.ReceiptID != "receipt-1" {
						t.Fatalf("settled=%+v", first)
					}
				} else {
					held = f.paused(t, stage)
				}
				if effects := f.backend.effects.Load(); effects != 1 {
					t.Fatalf("not exactly one accepted effect: %d", effects)
				}
				switch event {
				case "observer-cancel":
					cancelObserver()
				case "logout":
					f.e.sessions.Logout(f.cookie)
				case "kill":
					f.s.Kill()
				}
				if stage != "settled" {
					select {
					case first = <-completed:
					case <-t.Context().Done():
						t.Fatal("revoked observer did not return")
					}
					expectedReceipt := ""
					if stage == "result-audit" {
						expectedReceipt = "receipt-1"
					}
					if first.err == nil || first.result.Outcome != "uncertain" || first.result.ReceiptID != expectedReceipt {
						t.Fatalf("observer at %s/%s=%+v", stage, event, first)
					}
					duplicate, err := f.dispatch(t.Context(), request)
					if event == "observer-cancel" {
						if err != nil || duplicate.Outcome != "pending" {
							t.Fatalf("pending duplicate=%+v %v", duplicate, err)
						}
					} else if err == nil {
						t.Fatalf("revoked session replay admitted: %+v", duplicate)
					}
					if event == "kill" {
						select {
						case <-f.backend.joinEntered:
						case <-t.Context().Done():
							t.Fatal("cleanup did not join owned producer")
						}
						if f.s.Snapshot().CanReset || f.reset() == nil {
							t.Fatal("reset allowed before accepted completion joined")
						}
						canceled, cancel := context.WithCancel(t.Context())
						cancel()
						if err := f.s.Join(canceled); !errors.Is(err, context.Canceled) {
							t.Fatalf("Join claimed cleanup while completion held: %v", err)
						}
						if stage == "backend-result" {
							f.e.requests.Wait()
						}
					}
					held.open()
				}
				f.e.requests.Wait()
				expected := DocumentResult{Outcome: "applied", Revision: "r1", ReceiptID: "receipt-1"}
				uncertain := stage == "backend-result" && event == "kill"
				if uncertain {
					expected = DocumentResult{Outcome: "uncertain"}
				}
				f.assertReplay(t, request, expected, uncertain)
				if event == "observer-cancel" {
					duplicate, err := f.dispatch(t.Context(), request)
					if err != nil || duplicate != expected {
						t.Fatalf("confirmed duplicate=%+v %v", duplicate, err)
					}
				}
				f.assertCounts(t, 1)
				if event == "kill" {
					if err := f.s.Join(t.Context()); err != nil {
						t.Fatal(err)
					}
					if !f.s.Snapshot().CanReset {
						t.Fatal("confirmed cleanup did not permit reset")
					}
					if err := f.reset(); err != nil {
						t.Fatal(err)
					}
					awaitRemote(t, f.s, "off")
				}
			})
		}
	}
}

func TestRemoteDispatchCanceledCompletionCyclesAtNAnd2N(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			f := newCompletionFixture(t, "result-audit")
			for sequence := 1; sequence <= n; sequence++ {
				request := submitRequest(uint64(sequence))
				observer, cancel := context.WithCancel(t.Context())
				completed := make(chan dispatchCompletion, 1)
				go func() { result, err := f.dispatch(observer, request); completed <- dispatchCompletion{result, err} }()
				held := f.paused(t, "result-audit")
				if held.sequence != uint64(sequence) {
					t.Fatalf("wrong held result %d", held.sequence)
				}
				cancel()
				first := <-completed
				expected := DocumentResult{Outcome: "applied", Revision: "r1", ReceiptID: fmt.Sprintf("receipt-%d", sequence)}
				if first.err == nil || first.result.Outcome != "uncertain" || first.result.ReceiptID != expected.ReceiptID {
					t.Fatalf("canceled observer=%+v", first)
				}
				if pending, err := f.dispatch(t.Context(), request); err != nil || pending.Outcome != "pending" {
					t.Fatalf("pending=%+v %v", pending, err)
				}
				held.open()
				f.e.requests.Wait()
				for range 2 {
					duplicate, err := f.dispatch(t.Context(), request)
					if err != nil || duplicate != expected {
						t.Fatalf("completed duplicate=%+v %v", duplicate, err)
					}
				}
				f.assertReplay(t, request, expected, false)
			}
			f.assertCounts(t, n)
		})
	}
}

func TestRemoteDispatchFailedCompletionAfterObserverRetirement(t *testing.T) {
	for _, event := range []string{"observer-cancel", "logout"} {
		t.Run(event, func(t *testing.T) {
			f := newCompletionFixture(t, "result-audit")
			f.failResult = true
			observer, cancel := context.WithCancel(t.Context())
			defer cancel()
			observed, closeObserved := documentSessionContext(observer, f.session)
			defer closeObserved()
			request := submitRequest(1)
			completed := make(chan dispatchCompletion, 1)
			go func() { result, err := f.dispatch(observed, request); completed <- dispatchCompletion{result, err} }()
			held := f.paused(t, "result-audit")
			if event == "observer-cancel" {
				cancel()
			} else {
				f.e.sessions.Logout(f.cookie)
			}
			first := <-completed
			if first.err == nil || first.result.Outcome != "uncertain" || first.result.ReceiptID != "receipt-1" {
				t.Fatalf("known receipt lost before failed publication: %+v", first)
			}
			held.open()
			f.e.requests.Wait()
			// The failed result write kills the epoch. Completion remains pinned as
			// uncertainty with its already-known receipt, never a fresh retriable effect.
			if err := f.s.Join(t.Context()); err != nil {
				t.Fatal(err)
			}
			f.assertReplay(t, request, DocumentResult{Outcome: "uncertain", Revision: "r1", ReceiptID: "receipt-1"}, true)
			if duplicate, err := f.dispatch(t.Context(), request); err == nil {
				t.Fatalf("failed publication replay admitted: %+v", duplicate)
			}
			if f.backend.effects.Load() != 1 || f.backend.handoffs.Load() != 1 || f.intents.Load() != 1 || f.resultAttempts.Load() != 1 || f.results.Load() != 0 {
				t.Fatalf("nonlinear failed completion: effects%d handoffs%d intents%d result-attempts%d durable-results%d", f.backend.effects.Load(), f.backend.handoffs.Load(), f.intents.Load(), f.resultAttempts.Load(), f.results.Load())
			}
			_, rows, err := readAuditSegment(f.s.paths.Audit())
			if err != nil || len(rows) != 1 || rows[0].kind != "intent" {
				t.Fatalf("failed publication fabricated a durable result: %+v %v", rows, err)
			}
			if submitted, err := f.s.origins.Submitted(originDocA); err != nil || !submitted {
				t.Fatalf("uncertain submit origin lost: %v %v", submitted, err)
			}
		})
	}
}
