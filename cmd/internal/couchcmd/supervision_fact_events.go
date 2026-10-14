package couchcmd

import (
	"context"
	"io"
	"path/filepath"
	"sort"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
)

func openSupervisionFacts(root, namespace string) *supervisionFactLog {
	return newSupervisionFactLog(context.Background(), time.Now, func(ctx context.Context) (io.WriteCloser, error) {
		opts := diagnosticlog.EnvironmentOptions(func(k string) string {
			if k == "PAIR_DATA_DIR" {
				return root
			}
			return ""
		})
		opts.Context = ctx
		opts.HardMaxBytes = true
		return diagnosticlog.Open(filepath.Join(namespace, "supervision.jsonl"), opts)
	})
}
func (l *supervisionFactLog) delivery(m couchmessage.Message, r couchmessage.Receipt, err error, generated bool) {
	outcome := "uncertain"
	if err == nil {
		switch r.Status {
		case couchmessage.Submitted:
			outcome = "confirmed"
		case couchmessage.Cancelled, couchmessage.Expired:
			outcome = "cancelled"
		}
	}
	l.offer(supervisionFact{Event: "hub_delivery", Subject: l.subject(m.To), Incarnation: l.incarnation(m.To), Sender: l.subject(m.From), Message: l.hash("message", m.ID), Hint: factKindHint(m.Body), Outcome: outcome, Generated: &generated})
}
func (l *supervisionFactLog) submitted(b couchmessage.Binding, n uint64) {
	l.offer(supervisionFact{Event: "operator_submit", Subject: l.subject(b), Incarnation: l.incarnation(b), Submission: n})
}
func sameFactBool(a, b *bool) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

type factSlotState struct {
	connected       bool
	settled, picker *bool
}

func factStates(connected map[couchmessage.Binding]bool, live map[string]couchmessage.SlotLiveness) map[couchmessage.Binding]factSlotState {
	states := make(map[couchmessage.Binding]factSlotState, len(connected))
	for b, on := range connected {
		states[b] = factSlotState{connected: on}
	}
	for _, v := range live {
		s := states[v.Binding]
		s.settled, s.picker = v.Settled, v.Picker
		states[v.Binding] = s
	}
	return states
}
func (l *supervisionFactLog) registry(before, after map[couchmessage.Binding]factSlotState) {
	for b, old := range before {
		if _, ok := after[b]; !ok {
			on := false
			l.offer(supervisionFact{Event: "slot_state", Subject: l.subject(b), Incarnation: l.incarnation(b), Connected: &on, Settled: old.settled, Picker: old.picker})
		}
	}
	for b, current := range after {
		old, ok := before[b]
		if ok && old.connected == current.connected && sameFactBool(old.settled, current.settled) && sameFactBool(old.picker, current.picker) {
			continue
		}
		on := current.connected
		l.offer(supervisionFact{Event: "slot_state", Subject: l.subject(b), Incarnation: l.incarnation(b), Connected: &on, Settled: current.settled, Picker: current.picker})
	}
}
func (l *supervisionFactLog) snapshot(now time.Time, rows []couchmessage.Candidate, live map[string]couchmessage.SlotLiveness, b *couchmessage.Broker) {
	l.mu.Lock()
	if !l.lastSnapshot.IsZero() && now.Before(l.lastSnapshot.Add(time.Minute)) {
		l.mu.Unlock()
		return
	}
	l.lastSnapshot = now
	l.mu.Unlock()
	slots := make([]supervisionFactSlot, 0, len(rows))
	for _, r := range rows {
		f := supervisionFactSlot{Subject: l.subject(r.Binding), Incarnation: l.incarnation(r.Binding), Known: r.Known}
		if r.Known {
			resting, pending, quiet := r.Resting, r.Pending, max(int64(0), now.Sub(r.LastActivity).Milliseconds())
			f.Resting, f.Pending, f.QuietMS = &resting, &pending, &quiet
		}
		if v, ok := live[r.Binding.Slot]; ok && v.Binding == r.Binding {
			f.Settled, f.Picker = v.Settled, v.Picker
		}
		if n, known := b.Submission(r.Binding); known {
			f.Submission = &n
		}
		slots = append(slots, f)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i].Subject < slots[j].Subject })
	l.offer(supervisionFact{Event: "snapshot", Slots: slots})
}
