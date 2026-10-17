package couchmessage

import (
	"strings"
	"testing"
	"time"
)

func TestRegistryBackgroundWorkTransitions(t *testing.T) {
	at := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	yes, no := true, false
	for _, tc := range []struct {
		name    string
		states  []*bool
		offsets []time.Duration
		want    time.Duration
		final   *bool
	}{
		{"silent long wait", []*bool{&yes, &yes}, []time.Duration{0, 48 * time.Hour}, 48 * time.Hour, &yes},
		{"completion", []*bool{&yes, &no, &no}, []time.Duration{0, time.Hour, 48 * time.Hour}, time.Hour, &no},
		{"unknown ends hint without completion", []*bool{&yes, nil, nil}, []time.Duration{0, time.Hour, 48 * time.Hour}, 0, nil},
		{"idle refresh inert", []*bool{&no, &no, nil}, []time.Duration{0, time.Hour, 48 * time.Hour}, -1, nil},
		{"older active monotonic", []*bool{&yes, &yes}, []time.Duration{time.Hour, 0}, time.Hour, &yes},
		{"older completion monotonic", []*bool{&yes, &no}, []time.Duration{time.Hour, 0}, time.Hour, &no},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRegistry()
			b := registryBinding("pair:1", "background", 101)
			step(t, r, RegistryEvent{Kind: SessionOpened, Token: 1, Binding: b, Build: &BuildIdentity{SHA256: strings.Repeat("a", 64)}})
			step(t, r, RegistryEvent{Kind: AdmissionDone, Token: 1})
			for i, state := range tc.states {
				step(t, r, RegistryEvent{Kind: SessionActivity, Token: 1, Settled: &yes, BackgroundWork: state, At: at.Add(tc.offsets[i])})
			}
			live := r.Liveness()[b.Slot]
			if live.Settled == nil || !*live.Settled {
				t.Fatal("background work changed settled admission evidence")
			}
			if (live.BackgroundWork == nil) != (tc.final == nil) || (tc.final != nil && *live.BackgroundWork != *tc.final) {
				t.Fatalf("background hint = %v; want %v", live.BackgroundWork, tc.final)
			}
			want := time.Time{}
			if tc.want >= 0 {
				want = at.Add(tc.want)
			}
			if !live.LastWorkAt.Equal(want) {
				t.Fatalf("last work = %v; want %v", live.LastWorkAt, want)
			}
		})
	}
}

func TestRegistryBackgroundWorkOwnedAndSessionScoped(t *testing.T) {
	r := NewRegistry()
	b := registryBinding("pair:1", "background", 101)
	build := &BuildIdentity{SHA256: strings.Repeat("a", 64)}
	admit := func(token SessionToken, build *BuildIdentity) {
		step(t, r, RegistryEvent{Kind: SessionOpened, Token: token, Binding: b, Build: build})
		step(t, r, RegistryEvent{Kind: AdmissionDone, Token: token})
	}
	at := time.Now()
	active := true
	report := func(token SessionToken) {
		step(t, r, RegistryEvent{Kind: SessionActivity, Token: token, BackgroundWork: &active, At: at})
	}
	admit(1, build)
	report(1)
	active = false
	live := r.Liveness()[b.Slot]
	if live.BackgroundWork == nil || !*live.BackgroundWork {
		t.Fatal("registry aliased caller pointer or lost active hint")
	}
	*live.BackgroundWork = false
	if got := r.Liveness()[b.Slot]; got.BackgroundWork == nil || !*got.BackgroundWork {
		t.Fatal("snapshot mutated registry evidence")
	}
	step(t, r, RegistryEvent{Kind: PaneChanged, Thread: b.Thread()})
	if got := r.Liveness()[b.Slot]; got.BackgroundWork == nil || !*got.BackgroundWork {
		t.Fatal("pane detach erased wrapper evidence")
	}
	admit(2, build)
	active = true
	report(1)
	if got := r.Liveness()[b.Slot]; got.BackgroundWork != nil || !got.LastWorkAt.IsZero() {
		t.Fatalf("replacement inherited evidence: %+v", got)
	}
	step(t, r, RegistryEvent{Kind: SessionClosed, Token: 2})
	if _, ok := r.Liveness()[b.Slot]; ok {
		t.Fatal("disconnect kept evidence")
	}
	admit(3, nil)
	report(3)
	if got := r.Liveness()[b.Slot]; got.BackgroundWork != nil || !got.LastWorkAt.IsZero() {
		t.Fatalf("legacy forged evidence: %+v", got)
	}
}
