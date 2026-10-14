package couchcmd

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"testing"
	"time"
)

func TestSupervisionFactsDeliveryAndSnapshots(t *testing.T) {
	w := newWaitRig(t, true, true, nil)
	c, at := installTestSupervision(w, func(string) {})
	sink := &factSink{}
	log := testFactLog(t, sink)
	w.s.facts.Store(log)
	supervisionSendCLI(t, w.serviceRig, w.worker, "hub", "facts-report", "Landed #543.")
	supervisionWaitReceipt(t, w.serviceRig, w.worker, "facts-report", couchmessage.Submitted)
	c.tick(context.Background(), at, nil)
	c.tick(context.Background(), at.Add(15*time.Second), nil)
	c.tick(context.Background(), at.Add(time.Minute), nil)
	supervisionEventually(t, "delivery and snapshots missing", func() bool {
		reports, snapshots := 0, 0
		for _, f := range sink.rows(t) {
			if f.Event == "hub_delivery" && f.Generated != nil && !*f.Generated {
				reports++
				if f.Subject != log.subject(w.hub) || f.Sender != log.subject(w.worker) || f.Hint != "landed" || f.Outcome != "confirmed" {
					t.Errorf("bad report %+v", f)
				}
			}
			if f.Event == "snapshot" {
				snapshots++
				if len(f.Slots) != 2 {
					t.Errorf("snapshot excludes hub: %+v", f)
				}
			}
		}
		return reports == 1 && snapshots == 2
	})
}
func TestSupervisionFactsRegistryAndSubmissions(t *testing.T) {
	r := newServiceRig(t)
	sink := &factSink{}
	log := testFactLog(t, sink)
	r.s.facts.Store(log)
	b := r.world.add(1)
	r.attach(b, "pane")
	_, stop := r.wrapper(b)
	r.waitConnected(b, true)
	if err := r.s.broker.ReconcileObservation(b, couchmessage.Observation{LastActivity: time.Now(), Sequence: 4, Submission: 3}); err != nil {
		t.Fatal(err)
	}
	stop()
	r.waitConnected(b, false)
	supervisionEventually(t, "state/submit facts missing", func() bool {
		connected, removed, submitted := false, false, false
		for _, f := range sink.rows(t) {
			if f.Subject != log.subject(b) {
				continue
			}
			switch f.Event {
			case "slot_state":
				if f.Connected != nil {
					if *f.Connected {
						connected = true
					} else {
						removed = true
					}
				}
			case "operator_submit":
				submitted = f.Submission == 3
			}
		}
		return connected && removed && submitted
	})
}
func TestSupervisionFactsFailedSinkDoesNotFailDelivery(t *testing.T) {
	w := newWaitRig(t, true, true, nil)
	installTestSupervision(w, func(string) {})
	sink := &factSink{fail: true}
	log := testFactLog(t, sink)
	w.s.facts.Store(log)
	supervisionSendCLI(t, w.serviceRig, w.worker, "hub", "facts-failed-sink", "Progress: done.")
	supervisionWaitReceipt(t, w.serviceRig, w.worker, "facts-failed-sink", couchmessage.Submitted)
	supervisionEventually(t, "failed logging not counted", func() bool { log.mu.Lock(); defer log.mu.Unlock(); return log.dropped > 0 })
}

func TestSupervisionFactsGeneratedWaitIsMarked(t *testing.T) {
	w := newWaitRig(t, false, false, claudePermissionTail)
	installTestSupervision(w, func(string) {})
	sink := &factSink{}
	log := testFactLog(t, sink)
	w.s.facts.Store(log)
	w.tickAt(11 * time.Minute)
	supervisionEventually(t, "generated wait fact missing", func() bool {
		for _, f := range sink.rows(t) {
			if f.Event == "hub_delivery" && f.Generated != nil && *f.Generated {
				return true
			}
		}
		return false
	})
}

// The delivery boundary may outlive the recipient's connected workspace. Its
// diagnostic classification must describe the recipient accepted before IO.
type factDisconnectEndpoint struct {
	serviceEndpointFake
	disconnect func()
}

func (e factDisconnectEndpoint) Deliver(ctx context.Context, m couchmessage.Message, _ func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	if e.disconnect != nil {
		e.disconnect()
	}
	return couchmessage.Receipt{Status: couchmessage.Submitted}, ctx.Err()
}

func TestSupervisionFactsAllHubRecipientsPreserveDeliveryClassification(t *testing.T) {
	for _, tc := range []struct {
		name     string
		from, to int
		want     bool
	}{
		{"main-cross-hub", 2, 0, true},
		{"secondary-cross-hub", 3, 1, true},
		{"worker", 0, 2, false},
	} {
		for _, disconnect := range []bool{false, true} {
			name := tc.name
			if disconnect {
				name += "-disconnect"
			}
			t.Run(name, func(t *testing.T) {
				r, b := hubMessageRig(t)
				sink := &factSink{}
				log := testFactLog(t, sink)
				r.s.facts.Store(log)
				endpoint := factDisconnectEndpoint{}
				if disconnect {
					endpoint.disconnect = func() {
						r.s.broker.Disconnect(b[tc.to])
						r.s.mu.Lock()
						delete(r.s.workspaces, b[tc.to])
						r.s.mu.Unlock()
					}
				}
				m := couchmessage.Message{ID: "fact-classification", From: b[tc.from], To: b[tc.to], Body: "Landed."}
				receipt, err := (messageEndpoint{service: r.s, binding: m.To, endpoint: endpoint}).Deliver(context.Background(), m, nil)
				if err != nil || receipt.Status != couchmessage.Submitted {
					t.Fatalf("delivery: %+v %v", receipt, err)
				}
				// A positive writer barrier also makes the non-hub absence assertion
				// deterministic; Close intentionally does not drain diagnostic rows.
				log.submitted(m.From, 1)
				supervisionEventually(t, "fact writer barrier missing", func() bool {
					for _, f := range sink.rows(t) {
						if f.Event == "operator_submit" {
							return true
						}
					}
					return false
				})
				count := 0
				for _, f := range sink.rows(t) {
					if f.Event != "hub_delivery" {
						continue
					}
					count++
					if f.Subject != log.subject(m.To) || f.Sender != log.subject(m.From) || f.Outcome != "confirmed" || f.Generated == nil || *f.Generated {
						t.Errorf("incorrect delivery evidence: %+v", f)
					}
				}
				want := 0
				if tc.want {
					want = 1
				}
				if count != want {
					t.Fatalf("hub_delivery facts=%d, want %d", count, want)
				}
			})
		}
	}
}
