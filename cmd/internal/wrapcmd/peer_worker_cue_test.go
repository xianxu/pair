package wrapcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestPeerWorkerCueWitnessSurvivesOrdinaryReceiptEviction(t *testing.T) {
	f, d := peerIntegrationFixture(t)
	d.binding = peerBatchTestBinding("pair:1")
	d.initWorkerCue(7)
	now := time.Now()
	d.now = func() time.Time { return now }
	m := peerTestMessage(now)
	m.From = d.binding
	m.To = d.binding
	m.Origin = couchmessage.OriginSystem
	m.Body = couchmessage.WorkerStartNoticeBody
	m.ID = couchmessage.WorkerStartNoticeID(d.binding, 7)
	if err := d.reserve(m.ID, d.sequence); err != nil {
		t.Fatal(err)
	}
	if err := d.enqueue(m); err != nil {
		t.Fatal(err)
	}
	if got := d.workerCueReport(); got.State != couchmessage.WorkerCueCommitted {
		t.Fatalf("commit: %+v", got)
	}
	var out bytes.Buffer
	f.output(peerEmptyComposer())
	f.proxy.dispatchPeer(&out)
	if got := d.workerCueReport(); got.State != couchmessage.WorkerCueCommitted {
		t.Fatalf("paste is not confirmation: %+v", got)
	}
	now = now.Add(PeerSubmitDelay + time.Second)
	f.proxy.dispatchPeer(&out)
	f.proxy.turnActive.Store(true)
	f.proxy.dispatchPeer(&out)
	if got := d.workerCueReport(); got.State != couchmessage.WorkerCueSubmitted {
		t.Fatalf("confirmation: %+v", got)
	}
	for i := 0; i < couchmessage.MaxRecentDeliveries+3; i++ {
		d.mu.Lock()
		d.reservation = ""
		d.mu.Unlock()
		next := peerTestMessage(now)
		next.ID = fmt.Sprintf("ordinary-%d", i)
		next.To = d.binding
		if err := d.reserve(next.ID, d.sequence); err != nil {
			t.Fatal(err)
		}
		if err := d.enqueue(next); err != nil {
			t.Fatal(err)
		}
		d.mu.Lock()
		d.current.Status = couchmessage.Cancelled
		d.mu.Unlock()
	}
	if _, ok := d.recent.Get(m.ID); ok {
		t.Fatal("test failed to evict ordinary cache")
	}
	if r, ok := d.retained(m.ID); !ok || r.Status != couchmessage.Submitted || r.Message != m {
		t.Fatalf("sticky receipt lost: %+v %v", r, ok)
	}
	if err := d.reserve(m.ID, d.sequence); err == nil {
		t.Fatal("sticky cue was reservable again")
	}
	replacement := newPeerDelivery(d.binding, time.Now)
	replacement.initWorkerCue(8)
	if got := replacement.workerCueReport(); got.State != couchmessage.WorkerCueNeeded || got.LaunchOrdinal != 8 {
		t.Fatalf("exec not new opportunity: %+v", got)
	}
}

func TestPeerWorkerCueBatchFollowerAndPeerTextNegative(t *testing.T) {
	f, d, messages, _ := peerBatchFixture(t)
	d.initWorkerCue(3)
	cue := messages[1]
	cue.Origin = couchmessage.OriginSystem
	cue.From = d.binding
	cue.ID = couchmessage.WorkerStartNoticeID(d.binding, 3)
	cue.Body = couchmessage.WorkerStartNoticeBody
	messages[1] = cue
	f.output(peerEmptyComposer())
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	if got := d.workerCueReport(); got.State != couchmessage.WorkerCueNeeded {
		t.Fatal("prepared head consumed follower")
	}
	if err := d.finalizeBatch(messages[0].ID, messages); err != nil {
		t.Fatal(err)
	}
	if got := d.workerCueReport(); got.State != couchmessage.WorkerCueCommitted || got.ID != cue.ID {
		t.Fatalf("follower ignored: %+v", got)
	}
	d2 := newPeerDelivery(d.binding, time.Now)
	d2.initWorkerCue(3)
	cue.Origin = couchmessage.OriginPeer
	if err := d2.reserve(cue.ID, d2.sequence); err != nil {
		t.Fatal(err)
	}
	if err := d2.enqueue(cue); err != nil {
		t.Fatal(err)
	}
	if got := d2.workerCueReport(); got.State != couchmessage.WorkerCueNeeded {
		t.Fatal("peer copying canonical text consumed cue")
	}
}

// The wrapper and its Unix delivery endpoint outlive two private supervisors.
// Cover both shutdown during confirmation and after confirmation. Neither
// reconnect may authorize another paste; a same-PID replacement gets a new cue.
func TestPeerWorkerCuePrivateSupervisorReplacementCountsNativeInput(t *testing.T) {
	for _, beforeConfirmation := range []bool{true, false} {
		t.Run(fmt.Sprintf("before_confirmation_%t", beforeConfirmation), func(t *testing.T) {
			lifetime, cancel := context.WithCancel(context.Background())
			defer cancel()
			namespace := t.TempDir()
			binding := peerBatchTestBinding("pair:1")
			binding.PID = os.Getpid()
			f, _ := peerIntegrationFixture(t)
			d := newPeerDelivery(binding, time.Now)
			d.initWorkerCue(7)
			f.proxy.peer = d
			var clock atomic.Int64
			clock.Store(time.Now().UnixNano())
			d.now = func() time.Time { return time.Unix(0, clock.Load()) }
			endpointSocket, err := couchmessage.EndpointSocket(namespace, binding)
			if err != nil {
				t.Fatal(err)
			}
			endpoint, err := couchmessage.StartServer(lifetime, endpointSocket, d.handleEndpoint)
			if err != nil {
				t.Fatal(err)
			}
			defer endpoint.Close()
			registrySocket, err := couchmessage.SocketPath(namespace, "registry")
			if err != nil {
				t.Fatal(err)
			}
			reports := make(chan *couchmessage.WorkerCueReport, 30)
			startSupervisor := func() (*couchmessage.Broker, *couchmessage.SessionServer) {
				t.Helper()
				broker := couchmessage.NewBroker(lifetime, time.Now, func(context.Context, couchmessage.Binding) (bool, error) { return true, nil })
				if err := broker.Register(binding, couchmessage.RemoteEndpoint{Namespace: namespace, Binding: binding}); err != nil {
					t.Fatal(err)
				}
				session, err := couchmessage.StartSessionServer(lifetime, registrySocket, couchmessage.SessionHandler{
					Open: func(_ couchmessage.SessionToken, h couchmessage.SessionHello) error {
						reports <- h.WorkerCue
						return nil
					},
					Frame:  func(_ couchmessage.SessionToken, f couchmessage.SessionFrame) { reports <- f.WorkerCue },
					Closed: func(couchmessage.SessionToken) {},
				})
				if err != nil {
					t.Fatal(err)
				}
				return broker, session
			}
			await := func(want couchmessage.WorkerCueState) {
				t.Helper()
				deadline := time.After(3 * time.Second)
				for {
					select {
					case report := <-reports:
						if report != nil && report.State == want {
							return
						}
					case <-deadline:
						t.Fatalf("no report %s", want)
					}
				}
			}
			startClient := func(receiver *peerDelivery) context.CancelFunc {
				c := couchmessage.NewSessionClient()
				c.SetBuild(couchmessage.BuildIdentity{SHA256: strings.Repeat("ab", 32)})
				receiver.mu.Lock()
				receiver.session = c
				initial := receiver.observationLocked()
				receiver.mu.Unlock()
				c.WorkerCue(receiver.workerCueReport())
				c.Update(initial)
				ctx, stop := context.WithCancel(lifetime)
				go c.Run(ctx, registrySocket, binding, couchmessage.ReconnectBackoff{Base: time.Millisecond, Cap: 10 * time.Millisecond, StableAfter: time.Hour}, time.Millisecond)
				return stop
			}
			broker, session := startSupervisor()
			stopClient := startClient(d)
			defer stopClient()
			await(couchmessage.WorkerCueNeeded)
			id := couchmessage.WorkerStartNoticeID(binding, 7)
			if _, err := broker.SendSystem(lifetime, binding, id, couchmessage.Route{Target: binding.Slot}, couchmessage.WorkerStartNoticeBody); err != nil {
				t.Fatal(err)
			}
			if !waitFor(t, func() bool { return d.workerCueReport().State == couchmessage.WorkerCueCommitted }) {
				t.Fatal("real socket commit missing")
			}
			f.output(peerEmptyComposer())
			var input bytes.Buffer
			f.proxy.dispatchPeer(&input)
			if strings.Count(input.String(), couchmessage.WorkerStartNoticeBody) != 1 {
				t.Fatalf("native paste: %q", input.String())
			}
			if !beforeConfirmation {
				clock.Add(int64(PeerSubmitDelay + time.Second))
				f.proxy.dispatchPeer(&input)
				f.proxy.turnActive.Store(true)
				f.proxy.dispatchPeer(&input)
				await(couchmessage.WorkerCueSubmitted)
			}
			if err := session.Close(); err != nil {
				t.Fatal(err)
			}
			if err := broker.Close(); err != nil {
				t.Fatal(err)
			}
			// Discard old queued reports: the following report must come from new hello.
			for len(reports) > 0 {
				<-reports
			}
			broker, session = startSupervisor()
			expected := couchmessage.WorkerCueSubmitted
			expectedReceipt := couchmessage.Submitted
			if beforeConfirmation {
				await(couchmessage.WorkerCueCommitted)
				f.proxy.dispatchPeer(&input)
				await(couchmessage.WorkerCueUncertain)
				expected = couchmessage.WorkerCueUncertain
				expectedReceipt = couchmessage.Indeterminate
			} else {
				await(couchmessage.WorkerCueSubmitted)
			}
			if err := session.Close(); err != nil {
				t.Fatal(err)
			}
			if err := broker.Close(); err != nil {
				t.Fatal(err)
			}
			for len(reports) > 0 {
				<-reports
			}
			broker, session = startSupervisor()
			defer broker.Close()
			defer session.Close()
			await(expected)
			r, err := (couchmessage.RemoteEndpoint{Namespace: namespace, Binding: binding}).Retained(lifetime, id)
			if err != nil || r.Status != expectedReceipt || r.Message.ID != id {
				t.Fatalf("exact recovery: %+v %v", r, err)
			}
			f.proxy.dispatchPeer(&input)
			if strings.Count(input.String(), couchmessage.WorkerStartNoticeBody) != 1 {
				t.Fatal("supervisor replacement duplicated native cue")
			}
			stopClient()
			// Same process birth/nonce/binding, new launch ordinal after wrapper exec.
			if err := endpoint.Close(); err != nil {
				t.Fatal(err)
			}
			replacement := newPeerDelivery(binding, time.Now)
			replacement.initWorkerCue(8)
			f2, _ := peerIntegrationFixture(t)
			f2.proxy.peer = replacement
			replacementEndpoint, err := couchmessage.StartServer(lifetime, endpointSocket, replacement.handleEndpoint)
			if err != nil {
				t.Fatal(err)
			}
			defer replacementEndpoint.Close()
			stopReplacement := startClient(replacement)
			defer stopReplacement()
			await(couchmessage.WorkerCueNeeded)
			if couchmessage.WorkerStartNoticeID(binding, 8) == id {
				t.Fatal("same-PID reexec reused cue ID")
			}
			replacementID := couchmessage.WorkerStartNoticeID(binding, 8)
			if _, err := broker.SendSystem(lifetime, binding, replacementID, couchmessage.Route{Target: binding.Slot}, couchmessage.WorkerStartNoticeBody); err != nil {
				t.Fatal(err)
			}
			if !waitFor(t, func() bool { return replacement.workerCueReport().State == couchmessage.WorkerCueCommitted }) {
				t.Fatal("replacement cue never committed")
			}
			f2.output(peerEmptyComposer())
			var replacementInput bytes.Buffer
			f2.proxy.dispatchPeer(&replacementInput)
			if strings.Count(input.String()+replacementInput.String(), couchmessage.WorkerStartNoticeBody) != 2 {
				t.Fatal("replacement did not get exactly one new native cue")
			}

		})
	}
}

func TestPeerWorkerCueLostCommitAckRetainsWitness(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	namespace := t.TempDir()
	binding := peerBatchTestBinding("pair:1")
	d := newPeerDelivery(binding, time.Now)
	d.initWorkerCue(7)
	socket, err := couchmessage.EndpointSocket(namespace, binding)
	if err != nil {
		t.Fatal(err)
	}
	server, err := couchmessage.StartServer(ctx, socket, func(ctx context.Context, raw []byte) ([]byte, error) {
		reply, err := d.handleEndpoint(ctx, raw)
		var request couchmessage.EndpointRequest
		if json.Unmarshal(raw, &request) == nil && request.Op == "commit" && err == nil {
			return nil, errors.New("controlled lost commit acknowledgement")
		}
		return reply, err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	m := couchmessage.Message{Origin: couchmessage.OriginSystem, From: binding, To: binding, ID: couchmessage.WorkerStartNoticeID(binding, 7), Body: couchmessage.WorkerStartNoticeBody, Deadline: time.Now().Add(couchmessage.AdmissionTimeout)}
	remote := couchmessage.RemoteEndpoint{Namespace: namespace, Binding: binding}
	if err := remote.Reserve(ctx, m.ID, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := remote.Deliver(ctx, m, nil); err == nil {
		t.Fatal("test failed to lose acknowledgement")
	}
	if got := d.workerCueReport(); got.State != couchmessage.WorkerCueCommitted || got.ID != m.ID {
		t.Fatalf("ack loss reset witness: %+v", got)
	}
	if r, err := remote.Retained(ctx, m.ID); err != nil || r.Message.ID != m.ID || r.Message.Origin != m.Origin || r.Message.From != m.From || r.Message.To != m.To || r.Message.Body != m.Body || !r.Message.Deadline.Equal(m.Deadline) || r.Status != couchmessage.Queued {
		t.Fatalf("retained commit: %+v %v", r, err)
	}
	var committed *couchmessage.AlreadyCommittedError
	if err := remote.Reserve(ctx, m.ID, 0); !errors.As(err, &committed) {
		t.Fatalf("duplicate reservation not fenced: %v", err)
	}
	unknown := newPeerDelivery(binding, time.Now)
	unknown.initWorkerCue(0)
	if unknown.workerCueReport() != nil {
		t.Fatal("unknown launch ordinal advertised needed")
	}
}
