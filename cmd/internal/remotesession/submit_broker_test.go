package remotesession

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// submitEndpoint is a stateful private wrapper seam: reservations, exact
// delivered messages and terminal receipts survive subsequent broker reads.
// couchmessage's FakeDeliveryEndpoint lives in its _test package, so it cannot
// be imported here. This fixture exercises its actual public endpoint seam.
type submitEndpoint struct {
	mu        sync.Mutex
	reserved  string
	receipts  map[string]couchmessage.Receipt
	delivered chan couchmessage.Receipt
	count     atomic.Int64
}

func newSubmitEndpoint() *submitEndpoint {
	return &submitEndpoint{receipts: make(map[string]couchmessage.Receipt), delivered: make(chan couchmessage.Receipt, 32)}
}
func (f *submitEndpoint) Observe(context.Context) (couchmessage.Observation, error) {
	return couchmessage.Observation{Sequence: 1, LastActivity: time.Now().Add(-time.Hour)}, nil
}
func (f *submitEndpoint) Reserve(_ context.Context, id string, sequence uint64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if receipt, ok := f.receipts[id]; ok {
		return &couchmessage.AlreadyCommittedError{Receipt: receipt}
	}
	if f.reserved != "" || sequence != 1 {
		return couchmessage.ErrRecipientBusy
	}
	f.reserved = id
	return nil
}
func (f *submitEndpoint) Release(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.reserved == id {
		f.reserved = ""
	}
	return nil
}
func (f *submitEndpoint) Deliver(ctx context.Context, message couchmessage.Message, _ func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	f.mu.Lock()
	if f.reserved != message.ID {
		f.mu.Unlock()
		return couchmessage.Receipt{}, errors.New("missing exact reservation")
	}
	receipt := couchmessage.Receipt{Message: message, Status: couchmessage.Submitted}
	f.receipts[message.ID] = receipt
	f.count.Add(1)
	f.mu.Unlock()
	select {
	case f.delivered <- receipt:
		return receipt, nil
	case <-ctx.Done():
		return couchmessage.Receipt{}, ctx.Err()
	}
}
func (f *submitEndpoint) Retained(_ context.Context, id string) (couchmessage.Receipt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.receipts[id]; ok {
		return r, nil
	}
	return couchmessage.Receipt{}, couchmessage.ErrUnknownDelivery
}

type capableSubmitEndpoint struct{ *submitEndpoint }

func (*capableSubmitEndpoint) SupportsRemoteProvenance(context.Context) error { return nil }

type brokerDocumentJob struct {
	ctx     context.Context
	request DocumentRequest
	permit  Permit
	result  chan DocumentResult
}

// brokerSubmitDocuments is #758's private stateful seam, not a public nvim
// adapter or native-agent acceptance. The supervisor's admission gate protects
// only a nonblocking queue handoff. Broker and origin IO run on this owned worker.
type brokerSubmitDocuments struct {
	*fixtureDocuments
	broker   *couchmessage.Broker
	origins  *OriginStore
	from, to couchmessage.Binding
	path     string
	jobs     chan brokerDocumentJob
	submits  chan brokerDocumentJob
	entered  chan struct{}
	held     <-chan struct{}
	stop     context.CancelFunc
	done     chan struct{}
}

func (f *brokerSubmitDocuments) Dispatch(ctx context.Context, request DocumentRequest, permit Permit) (<-chan DocumentResult, bool) {
	result := make(chan DocumentResult, 1)
	select {
	case <-f.done:
		return nil, false
	default:
	}
	select {
	case f.jobs <- brokerDocumentJob{ctx, request, permit, result}:
		return result, true
	default:
		return nil, false
	}
}
func (f *brokerSubmitDocuments) Join(ctx context.Context) error {
	f.stop()
	select {
	case <-f.done:
		return f.fixtureDocuments.Join(ctx)
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (f *brokerSubmitDocuments) work(ctx context.Context) {
	defer close(f.done)
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-f.jobs:
			select {
			case f.entered <- struct{}{}:
			default:
			}
			if f.held != nil {
				select {
				case <-ctx.Done():
					job.result <- DocumentResult{Outcome: "refused"}
					return
				case <-job.ctx.Done():
					job.result <- DocumentResult{Outcome: "refused"}
					continue
				case <-f.held:
				}
			}
			// The real adapter rechecks registration/editor/revision at this handoff.
			snapshot, err := f.Read(job.ctx, job.request.Document)
			if err != nil || snapshot.Revision != job.request.BaseRevision || job.request.Action != ActionSubmit {
				job.result <- DocumentResult{Outcome: "refused"}
				continue
			}
			if !job.permit.EffectDoor(job.request.Document, func() bool {
				select {
				case f.submits <- job:
					return true
				default:
					return false
				}
			}) {
				job.result <- DocumentResult{Outcome: "refused"}
			}
		case job := <-f.submits:
			marked, err := f.origins.Marked(job.request.Document.ID)
			if err != nil {
				job.result <- DocumentResult{Outcome: "refused"}
				continue
			}
			source := job.permit.Source()
			callContext, cancelCall := context.WithCancel(job.ctx)
			stopCall := context.AfterFunc(ctx, cancelCall)
			receipt, err := f.broker.SendRemote(callContext, f.from, fmt.Sprintf("remote-submit-%d", job.request.Sequence), couchmessage.Route{Target: f.to.Slot}, "submitted: "+f.path, couchmessage.RemoteSource{Email: source.Email, IP: source.IP}, marked, job.request.Document.ID)
			stopCall()
			cancelCall()
			result := DocumentResult{Outcome: "refused"}
			if err == nil {
				result = DocumentResult{Outcome: "applied", Revision: job.request.BaseRevision, ReceiptID: receipt.Message.ID}
			}
			job.result <- result
		}
	}
}

func newSubmitBrokerFixture(t *testing.T, capable bool) (documentHTTPFixture, *brokerSubmitDocuments, *submitEndpoint) {
	t.Helper()
	httpFixture := newDocumentHTTPFixture(t)
	endpoint := newSubmitEndpoint()
	broker := couchmessage.NewBroker(t.Context(), time.Now, nil)
	from := submitBrokerBinding("pair:1")
	to := submitBrokerBinding("ops:0")
	if err := broker.Register(from, newSubmitEndpoint()); err != nil {
		t.Fatal(err)
	}
	var receiver couchmessage.DeliveryEndpoint = endpoint
	if capable {
		receiver = &capableSubmitEndpoint{endpoint}
	}
	// This registered document belongs to the hub, whose normal allowance is
	// large enough for the approved N=8/16 accounting exercise.
	if err := broker.RegisterWithAllowance(to, receiver, couchmessage.HubAllowance); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	backend := &brokerSubmitDocuments{fixtureDocuments: httpFixture.documents, broker: broker, origins: httpFixture.supervisor.origins, from: from, to: to, path: filepath.Join(t.TempDir(), "session.md"), jobs: make(chan brokerDocumentJob, 1), submits: make(chan brokerDocumentJob, 1), entered: make(chan struct{}, 1), stop: cancel, done: make(chan struct{})}
	httpFixture.supervisor.backend = backend
	go backend.work(ctx)
	t.Cleanup(func() { cancel(); <-backend.done; _ = broker.Close() })
	return httpFixture, backend, endpoint
}
func submitBrokerBinding(slot string) couchmessage.Binding {
	return couchmessage.Binding{Slot: slot, Repository: "/private/fixture/.git", Scope: "private", Tag: slot, Session: "fixture-" + slot, Nonce: "nonce", PID: 1, Start: "private-start", Agent: "codex", Version: "fixture"}
}

func TestSignedRemoteSubmitThroughRealBrokerAtNAnd2N(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			f, backend, endpoint := newSubmitBrokerFixture(t, true)
			if err := f.supervisor.origins.MarkRemote(originDocA); err != nil {
				t.Fatal(err)
			}
			for sequence := 1; sequence <= n; sequence++ {
				request := submitRequest(uint64(sequence))
				for range 2 {
					response := f.request(t, "submit", request)
					if response.Code != 200 || !strings.Contains(response.Body.String(), "applied") {
						t.Fatalf("submit=%d %s", response.Code, response.Body.String())
					}
				}
				var receipt couchmessage.Receipt
				select {
				case receipt = <-endpoint.delivered:
				case <-t.Context().Done():
					t.Fatal("broker did not reach receiver")
				}
				message := receipt.Message
				if receipt.Status != couchmessage.Submitted || message.Origin != couchmessage.OriginRemote || message.RemoteSource != (couchmessage.RemoteSource{Email: "human@example.com", IP: "192.0.2.9"}) || message.DocumentID != originDocA || !message.RemoteEdited || message.Body != "submitted: "+backend.path {
					t.Fatalf("trusted facts lost: %+v", receipt)
				}
				if !strings.Contains(couchmessage.Envelope(message), "\n[remote human@example.com 192.0.2.9]\nsubmitted: "+backend.path) {
					t.Fatalf("missing native-input header: %q", couchmessage.Envelope(message))
				}
				if brokerReceipt, err := backend.broker.Status(backend.from, message.ID); err != nil || brokerReceipt.Message != message {
					t.Fatalf("broker receipt changed facts: %+v %v", brokerReceipt, err)
				}
			}
			if count := endpoint.count.Load(); count != int64(n) {
				t.Fatalf("actual deliveries=%d want%d", count, n)
			}
			if seen, err := f.supervisor.origins.Submitted(originDocA); err != nil || !seen {
				t.Fatalf("submit_seen missing: %v %v", seen, err)
			}
			_, rows, err := readAuditSegment(f.supervisor.paths.Audit())
			if err != nil || len(rows) != 2*n {
				t.Fatalf("audit rows=%d want%d err=%v", len(rows), 2*n, err)
			}
		})
	}
}

func TestSignedRemoteSubmitRejectsWebProvenanceBeforeBroker(t *testing.T) {
	f, _, endpoint := newSubmitBrokerFixture(t, true)
	for _, forged := range []string{`"RemoteSource":{"Email":"attacker@example.com","IP":"192.0.2.1"}`, `"Origin":"remote"`, `"RemoteEdited":true`, `"DocumentID":"` + originDocB + `"`, `"body":"submitted: /etc/passwd"`} {
		body := `{"sequence":1,"action":"submit","document":{"id":"` + originDocA + `","generation":"editor-1"},"base_revision":"r1",` + forged + `}`
		response := remoteRequest(t, f.server, "POST", f.path+"v1/submit", f.token, body, f.headers)
		if response.Code != 400 {
			t.Fatalf("forged source accepted=%d %s", response.Code, response.Body.String())
		}
	}
	if endpoint.count.Load() != 0 {
		t.Fatal("web-controlled provenance reached broker")
	}
	if seen, err := f.supervisor.origins.Submitted(originDocA); err != nil || seen {
		t.Fatalf("rejected request marked submitted: %v %v", seen, err)
	}
}

func TestSignedRemoteSubmitLegacyReceiverRefusesWithoutDelivery(t *testing.T) {
	f, backend, endpoint := newSubmitBrokerFixture(t, false)
	response := f.request(t, "submit", submitRequest(1))
	if response.Code != 200 || !strings.Contains(response.Body.String(), "refused") {
		t.Fatalf("legacy receiver=%d %s", response.Code, response.Body.String())
	}
	if endpoint.count.Load() != 0 {
		t.Fatal("legacy receiver received privileged mail")
	}
	receipt, err := backend.broker.Status(backend.from, "remote-submit-1")
	if err != nil || receipt.Status != couchmessage.Refused || receipt.Message.Origin != couchmessage.OriginRemote {
		t.Fatalf("capability refusal lost trusted facts: %+v %v", receipt, err)
	}
}

func TestSignedRemoteSubmitKillAtFinalHandoffHasZeroDelivery(t *testing.T) {
	f, backend, endpoint := newSubmitBrokerFixture(t, true)
	release := make(chan struct{})
	backend.held = release
	finished := make(chan struct{})
	go func() { defer close(finished); f.request(t, "submit", submitRequest(1)) }()
	select {
	case <-backend.entered:
	case <-t.Context().Done():
		t.Fatal("final handoff not reached")
	}
	response := remoteRequest(t, f.server, "POST", "/remote/kill", f.token, "{}", map[string]string{"X-Remote-Kill": "1"})
	if response.Code != 204 {
		t.Fatalf("kill=%d", response.Code)
	}
	close(release)
	select {
	case <-finished:
	case <-t.Context().Done():
		t.Fatal("submit observer not joined")
	}
	select {
	case <-backend.done:
	case <-t.Context().Done():
		t.Fatal("owned backend worker not joined")
	}
	if endpoint.count.Load() != 0 {
		t.Fatal("delivery crossed kill's final-handoff fence")
	}
	if seen, err := f.supervisor.origins.Submitted(originDocA); err != nil || !seen {
		t.Fatalf("conservative submit evidence lost: %v %v", seen, err)
	}
}
