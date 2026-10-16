package couchcore

import (
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"testing"
	"time"
)

type resourceFakeWorld struct {
	at    time.Time
	rows  []procutil.Process
	obs   []ResourceUsageObservation
	slots []ResourceSlot
	known bool
}

func (w *resourceFakeWorld) sample() ResourceSample {
	w.at = w.at.Add(time.Minute)
	return ResourceSample{At: w.at, Table: w.rows, TableKnown: w.known, Slots: w.slots, Observations: w.obs}
}
func TestResourceWorldResetPartialAndRestart(t *testing.T) {
	p, _ := ParseResourcePolicy(nil)
	k := ResourceProcessKey{12, "a"}
	w := resourceFakeWorld{at: time.Now(), known: true, rows: []procutil.Process{{PID: 12, Identity: "a"}}, slots: []ResourceSlot{{Slot: "r:1", OwnershipKnown: true, Roots: []ResourceRoot{{Key: k, Slot: "r:1"}}}}, obs: []ResourceUsageObservation{{Key: k, CPUKnown: true, DiskKnown: true, CPUTimeNS: 120e9, DiskWriteBytes: 200}}}
	st, _, _ := AdvanceResources(ResourceState{}, w.sample(), p)
	w.obs[0].CPUTimeNS = 10
	w.obs[0].DiskWriteBytes = 10
	st, v, _ := AdvanceResources(st, w.sample(), p)
	if v.Processes[0].CPUPercent != nil || v.Processes[0].WriteBytesPerSecond != nil {
		t.Fatal("counter reset generated rate")
	}
	w.known = false
	st, _, _ = AdvanceResources(st, w.sample(), p)
	w.known = true
	w.obs[0].CPUTimeNS += 60e9
	st, v, _ = AdvanceResources(st, w.sample(), p)
	if v.Processes[0].CPUPercent != nil {
		t.Fatal("rate bridged partial table")
	}
	b, _ := json.Marshal(st)
	var loaded ResourceState
	if e := json.Unmarshal(b, &loaded); e != nil {
		t.Fatal(e)
	}
	_, v, _ = AdvanceResources(ResetResourceBaselines(loaded), w.sample(), p)
	if v.Processes[0].CPUPercent != nil {
		t.Fatal("restart retained baseline")
	}
	w.rows = nil
	st, _, _ = AdvanceResources(st, w.sample(), p)
	if len(st.Provenance) != 0 {
		t.Fatal("dead generation retained")
	}
}
func TestResourceValidationRejectsBadState(t *testing.T) {
	if ValidateResourceState(ResourceState{Episodes: []ResourceEpisode{{Key: "x", Phase: "invalid"}}}) == nil {
		t.Fatal("bad phase accepted")
	}
	if ValidateResourceState(ResourceState{Provenance: []ResourceProvenance{{ResourceRoot: ResourceRoot{Key: ResourceProcessKey{1, "a"}, Slot: "r:1"}, Root: ResourceProcessKey{1, "a"}}}}) == nil {
		t.Fatal("unverified workspace accepted")
	}
}
func TestResourceValidationRejectsMalformedEffectAndTotals(t *testing.T) {
	for _, s := range []ResourceState{{Episodes: []ResourceEpisode{{Key: "x", Metric: "invalid", Phase: "clear"}}}, {Episodes: []ResourceEpisode{{Key: "x", Metric: "process-rss", Phase: "pending", Effect: &ResourceEffect{ID: "x", Metric: "process-rss"}}}}, {SlotTotals: []ResourceSlotTotal{{Slot: "r:1"}, {Slot: "r:1"}}}} {
		if ValidateResourceState(s) == nil {
			t.Fatalf("invalid state accepted: %+v", s)
		}
	}
}
