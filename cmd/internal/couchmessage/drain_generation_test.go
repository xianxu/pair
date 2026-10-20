package couchmessage

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func drainFenceFixture(t *testing.T, epoch uint64, kind DrainControlKind) DrainFence {
	t.Helper()
	ns, err := DrainNamespaceKey("/test/couch")
	if err != nil {
		t.Fatal(err)
	}
	return DrainFence{Namespace: ns, Generation: fmt.Sprintf("00000000-0000-4000-8000-%012d", epoch), Epoch: epoch, Instance: "wrapper-instance", Kind: kind}
}
func drainGenerationStep(t *testing.T, s DrainGeneration, k DrainGenerationEventKind, f DrainFence, id string, d DrainControlDisposition) DrainGeneration {
	t.Helper()
	next, err := ReduceDrainGeneration(s, DrainGenerationEvent{Kind: k, Fence: f, ID: id, Disposition: d})
	if err != nil {
		t.Fatal(err)
	}
	return next
}
func TestDrainFenceValidatedComparableNamespaceAuthority(t *testing.T) {
	f := drainFenceFixture(t, 1, DrainNoticeControl)
	if err := f.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*DrainFence){func(f *DrainFence) { f.Namespace = "" }, func(f *DrainFence) { f.Namespace = strings.Repeat("A", 64) }, func(f *DrainFence) { f.Generation = "old" }, func(f *DrainFence) { f.Epoch = 0 }, func(f *DrainFence) { f.Instance = "" }, func(f *DrainFence) { f.Instance = "bad\n" }, func(f *DrainFence) { f.Kind = "other" }} {
		bad := f
		mutate(&bad)
		if err := bad.Validate(); err == nil {
			t.Fatalf("accepted invalid fence %+v", bad)
		}
	}
	if !((DrainFence{}).IsZero()) || f.IsZero() {
		t.Fatal("optional fence zero detection")
	}
	for _, ns := range []string{"relative", "/test/../couch"} {
		if _, err := DrainNamespaceKey(ns); err == nil {
			t.Fatalf("noncanonical namespace %s", ns)
		}
	}
	other, err := DrainNamespaceKey("/test/other")
	if err != nil || other == f.Namespace {
		t.Fatal("namespace conflation")
	}
}
func TestDrainGenerationRetiresOnlyKnownRevocation(t *testing.T) {
	f := drainFenceFixture(t, 1, DrainNoticeControl)
	s, err := NewDrainGeneration(f.Namespace, f.Instance)
	if err != nil {
		t.Fatal(err)
	}
	s = drainGenerationStep(t, s, DrainAdmitControl, f, "notice", "")
	before := s.View()
	s = drainGenerationStep(t, s, DrainAdmitControl, f, "notice", "")
	if s.View() != before {
		t.Fatal("duplicate admission mutated generation")
	}
	s = drainGenerationStep(t, s, DrainBeginInput, f, "notice", "")
	if _, err := ReduceDrainGeneration(s, DrainGenerationEvent{Kind: DrainBeginInput, Fence: f, ID: "notice"}); err == nil {
		t.Fatal("repeated input granted")
	}
	s = drainGenerationStep(t, s, DrainSettleControl, f, "notice", DrainControlCommitted)
	next := drainFenceFixture(t, 2, DrainNoticeControl)
	if _, err := ReduceDrainGeneration(s, DrainGenerationEvent{Kind: DrainAdmitControl, Fence: next, ID: "new"}); err == nil {
		t.Fatal("settled input retired without revocation")
	}
	s = drainGenerationStep(t, s, DrainRevokeControl, f, "notice", DrainControlUnknown)
	if s.Allows(f, "notice") {
		t.Fatal("unknown revoke retained input authority")
	}
	if _, err := ReduceDrainGeneration(s, DrainGenerationEvent{Kind: DrainAdmitControl, Fence: next, ID: "new"}); err == nil {
		t.Fatal("unknown revocation retired")
	}
	s = drainGenerationStep(t, s, DrainRevokeControl, f, "notice", DrainControlCommitted)
	s = drainGenerationStep(t, s, DrainAdmitControl, next, "new", "")
	if s.Allows(f, "notice") || !s.Allows(next, "new") {
		t.Fatal("retired callback regained authority")
	}
	if _, err := ReduceDrainGeneration(s, DrainGenerationEvent{Kind: DrainAdmitControl, Fence: f, ID: "notice"}); err == nil {
		t.Fatal("old epoch revived")
	}
}
func TestDrainGenerationReadOnlyStatusAndConflicts(t *testing.T) {
	f := drainFenceFixture(t, 1, DrainNoticeControl)
	s, _ := NewDrainGeneration(f.Namespace, f.Instance)
	s = drainGenerationStep(t, s, DrainAdmitControl, f, "notice", "")
	before := s.View()
	future := drainFenceFixture(t, 2, DrainNoticeControl)
	if _, err := s.Lookup(future, "new"); err == nil {
		t.Fatal("future lookup succeeded")
	}
	if s.View() != before {
		t.Fatal("lookup advanced watermark")
	}
	for _, change := range []func(*DrainFence){func(f *DrainFence) { f.Namespace = strings.Repeat("a", 64) }, func(f *DrainFence) { f.Instance = "other" }, func(f *DrainFence) { f.Generation = "00000000-0000-4000-8000-999999999999" }} {
		bad := f
		change(&bad)
		if _, err := ReduceDrainGeneration(s, DrainGenerationEvent{Kind: DrainAdmitControl, Fence: bad, ID: "notice"}); err == nil {
			t.Fatalf("conflicting authority admitted %+v", bad)
		}
	}
	if _, err := ReduceDrainGeneration(s, DrainGenerationEvent{Kind: DrainAdmitControl, Fence: f, ID: "other"}); err == nil {
		t.Fatal("second notice slot allocated")
	}
	interrupt := f
	interrupt.Kind = DrainInterruptControl
	if _, err := ReduceDrainGeneration(s, DrainGenerationEvent{Kind: DrainAdmitControl, Fence: interrupt, ID: "notice"}); err == nil {
		t.Fatal("cross-kind ID reuse")
	}
	s = drainGenerationStep(t, s, DrainRevokeControl, interrupt, "interrupt", DrainControlNotCommitted)
	if s.Allows(interrupt, "interrupt") {
		t.Fatal("revoke-before-admit granted input")
	}
	s = drainGenerationStep(t, s, DrainAdmitControl, interrupt, "interrupt", "")
	if s.Allows(interrupt, "interrupt") {
		t.Fatal("late admission bypassed tombstone")
	}
}
func TestDrainGenerationThousandCyclesHaveConstantResidue(t *testing.T) {
	run := func(cycles int) (int, int) {
		f := drainFenceFixture(t, 1, DrainNoticeControl)
		s, _ := NewDrainGeneration(f.Namespace, f.Instance)
		peak, effects := 0, 0
		for n := 1; n <= cycles; n++ {
			for _, kind := range []DrainControlKind{DrainNoticeControl, DrainInterruptControl} {
				f = drainFenceFixture(t, uint64(n), kind)
				id := fmt.Sprintf("%s-%d", kind, n)
				s = drainGenerationStep(t, s, DrainAdmitControl, f, id, "")
				if !s.Allows(f, id) {
					t.Fatal("new input denied")
				}
				s = drainGenerationStep(t, s, DrainBeginInput, f, id, "")
				effects++
				s = drainGenerationStep(t, s, DrainSettleControl, f, id, DrainControlCommitted)
				s = drainGenerationStep(t, s, DrainRevokeControl, f, id, DrainControlCommitted)
			}
			current := 0
			for _, c := range s.View().Controls {
				if c.ID != "" {
					current++
				}
			}
			peak = max(peak, current)
			old := drainFenceFixture(t, uint64(max(n-1, 1)), DrainNoticeControl)
			if n > 1 && s.Allows(old, fmt.Sprintf("notice-%d", n-1)) {
				t.Fatal("old callback allowed")
			}
		}
		return peak, effects
	}
	p1, e1 := run(1000)
	p2, e2 := run(2000)
	if p1 != 2 || p2 != 2 || e1 != 2000 || e2 != 4000 {
		t.Fatalf("unbounded residue/effects %d %d %d %d", p1, p2, e1, e2)
	}
}
func FuzzDrainFenceValidation(f *testing.F) {
	f.Add("namespace", "generation", uint64(1), "instance", "notice")
	f.Fuzz(func(t *testing.T, ns, g string, e uint64, i, k string) {
		f := DrainFence{Namespace: ns, Generation: g, Epoch: e, Instance: i, Kind: DrainControlKind(k)}
		if err := f.Validate(); err == nil && (f.IsZero() || f.Epoch == 0 || len(f.Namespace) != 64) {
			t.Fatal("invalid fence accepted")
		}
	})
}

func TestDrainFenceWireRejectsCaseAliasesAndIncompleteAuthority(t *testing.T) {
	fence := drainFenceFixture(t, 1, DrainNoticeControl)
	raw, err := json.Marshal(fence)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{strings.Replace(string(raw), `"Epoch":1`, `"Epoch":1,"epoch":2`, 1), strings.Replace(string(raw), `"Namespace":`, `"namespace":`, 1), strings.Replace(string(raw), `"Kind":"notice"`, `"Kind":"notice","future":true`, 1), `null`, `{}`} {
		var f DrainFence
		if err := json.Unmarshal([]byte(bad), &f); err == nil {
			t.Fatalf("unsafe wire fence accepted %s", bad)
		}
	}
	var decoded DrainFence
	if err := json.Unmarshal(raw, &decoded); err != nil || decoded != fence {
		t.Fatal("canonical wire refused", err)
	}
}
