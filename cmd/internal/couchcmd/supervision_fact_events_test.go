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
