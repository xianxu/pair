package couchmessage

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func registryBinding(slot, tag string, pid int) Binding {
	return Binding{Slot: slot, Repository: "/repo/.git", Scope: "scope", Tag: tag, Session: "s-" + tag, Nonce: "n", Agent: "codex", Version: "1", PID: pid, Start: "start"}
}

func step(t *testing.T, r *Registry, e RegistryEvent) []RegistryEffect {
	t.Helper()
	fx, err := r.Advance(e)
	if err != nil {
		t.Fatalf("advance %+v: %v", e, err)
	}
	return fx
}

func kinds(fx []RegistryEffect) []RegistryEffectKind {
	out := make([]RegistryEffectKind, len(fx))
	for i, f := range fx {
		out[i] = f.Kind
	}
	return out
}

func TestRegistryHelloAdmitsWithoutPane(t *testing.T) {
	r := NewRegistry()
	b := registryBinding("pair:1", "a", 10)
	fx := step(t, r, RegistryEvent{Kind: SessionOpened, Token: 1, Binding: b})
	if len(fx) != 1 || fx[0].Kind != EffectAdmit || fx[0].Pane != "" {
		t.Fatalf("hello without any client pane must admit the wrapper: %+v", fx)
	}
	fx = step(t, r, RegistryEvent{Kind: AdmissionDone, Token: 1})
	if len(fx) != 1 || fx[0].Kind != EffectConnect || !r.Connected()[b] {
		t.Fatalf("wrapper admission effects %+v", fx)
	}
}

func TestRegistryClientDetachAndReplacementKeepWrapperConnected(t *testing.T) {
	r := NewRegistry()
	b := registryBinding("pair:1", "a", 10)
	step(t, r, RegistryEvent{Kind: PaneChanged, Thread: b.Thread(), Pane: "h1"})
	step(t, r, RegistryEvent{Kind: SessionOpened, Token: 1, Binding: b})
	step(t, r, RegistryEvent{Kind: AdmissionDone, Token: 1, Pane: "h1"})
	for _, pane := range []PaneHandle{"", "h2"} {
		if fx := step(t, r, RegistryEvent{Kind: PaneChanged, Thread: b.Thread(), Pane: pane}); len(fx) != 0 || !r.Connected()[b] {
			t.Fatalf("client pane %q altered live wrapper membership: %+v", pane, fx)
		}
	}
	if fx := step(t, r, RegistryEvent{Kind: SessionClosed, Token: 1}); len(fx) != 1 || fx[0].Kind != EffectDisconnect || r.Connected()[b] {
		t.Fatalf("wrapper close did not disconnect: %+v", fx)
	}
}

func TestRegistryClientReplacementDoesNotInvalidateWrapperAdmission(t *testing.T) {
	r := NewRegistry()
	b := registryBinding("pair:1", "a", 10)
	step(t, r, RegistryEvent{Kind: PaneChanged, Thread: b.Thread(), Pane: "h1"})
	step(t, r, RegistryEvent{Kind: SessionOpened, Token: 1, Binding: b})
	step(t, r, RegistryEvent{Kind: PaneChanged, Thread: b.Thread(), Pane: "h2"})
	if fx := step(t, r, RegistryEvent{Kind: AdmissionDone, Token: 1, Pane: "h1"}); len(fx) != 1 || fx[0].Kind != EffectConnect || !r.Connected()[b] {
		t.Fatalf("local pane replacement invalidated wrapper authority: %+v", fx)
	}
}

// An exec keeps the whole binding; both arrival orders of the old connection's
// close and the new hello must disconnect the old incarnation and connect the
// new one exactly once.
func TestRegistryIdenticalBindingOnNewTokenDisconnectsThenConnects(t *testing.T) {
	for _, closeFirst := range []bool{true, false} {
		t.Run(fmt.Sprint("closeFirst=", closeFirst), func(t *testing.T) {
			r := NewRegistry()
			b := registryBinding("pair:1", "a", 10)
			step(t, r, RegistryEvent{Kind: PaneChanged, Thread: b.Thread(), Pane: "h1"})
			step(t, r, RegistryEvent{Kind: SessionOpened, Token: 1, Binding: b})
			step(t, r, RegistryEvent{Kind: AdmissionDone, Token: 1, Pane: "h1"})
			var all []RegistryEffect
			if closeFirst {
				all = append(all, step(t, r, RegistryEvent{Kind: SessionClosed, Token: 1})...)
				all = append(all, step(t, r, RegistryEvent{Kind: SessionOpened, Token: 2, Binding: b})...)
			} else {
				all = append(all, step(t, r, RegistryEvent{Kind: SessionOpened, Token: 2, Binding: b})...)
				all = append(all, step(t, r, RegistryEvent{Kind: SessionClosed, Token: 1})...)
			}
			all = append(all, step(t, r, RegistryEvent{Kind: AdmissionDone, Token: 2, Pane: "h1"})...)
			want := []RegistryEffectKind{EffectDisconnect, EffectAdmit, EffectConnect}
			if fmt.Sprint(kinds(all)) != fmt.Sprint(want) || !r.Connected()[b] {
				t.Fatalf("effects %v, want %v", kinds(all), want)
			}
		})
	}
}

func TestRegistryAdmissionFailureBacksOffThenDormantUntilTargeted(t *testing.T) {
	r := NewRegistry()
	b := registryBinding("pair:1", "a", 10)
	step(t, r, RegistryEvent{Kind: PaneChanged, Thread: b.Thread(), Pane: "h1"})
	step(t, r, RegistryEvent{Kind: SessionOpened, Token: 1, Binding: b})
	for i, delay := range RetryDelays {
		fx := step(t, r, RegistryEvent{Kind: AdmissionDone, Token: 1, Pane: "h1", Err: errors.New("no")})
		if len(fx) != 1 || fx[0].Kind != EffectScheduleRetry || fx[0].Delay != delay || fx[0].Attempt != i+1 {
			t.Fatalf("attempt %d effects %+v", i+1, fx)
		}
		if fx := step(t, r, RegistryEvent{Kind: RetryDue, Token: 1, Attempt: i}); len(fx) != 0 {
			t.Fatalf("stale retry fired %+v", fx)
		}
		step(t, r, RegistryEvent{Kind: RetryDue, Token: 1, Attempt: i + 1})
	}
	if fx := step(t, r, RegistryEvent{Kind: AdmissionDone, Token: 1, Pane: "h1", Err: errors.New("no")}); len(fx) != 0 {
		t.Fatalf("retried past the bound: %+v", fx)
	}
	if p, _ := r.Phase(1); p != Dormant {
		t.Fatalf("phase %v", p)
	}
	if fx := step(t, r, RegistryEvent{Kind: SessionActivity, Token: 1}); len(fx) != 0 {
		t.Fatalf("activity woke a dormant session: %+v", fx)
	}
	if fx := step(t, r, RegistryEvent{Kind: SendTargeted, Slot: b.Slot}); len(fx) != 1 || fx[0].Kind != EffectAdmit {
		t.Fatalf("targeted send did not re-admit: %+v", fx)
	}
}

func TestRegistryConnectFailureRetriesAndDisconnectsNothing(t *testing.T) {
	r := NewRegistry()
	b := registryBinding("pair:1", "a", 10)
	step(t, r, RegistryEvent{Kind: PaneChanged, Thread: b.Thread(), Pane: "h1"})
	step(t, r, RegistryEvent{Kind: SessionOpened, Token: 1, Binding: b})
	step(t, r, RegistryEvent{Kind: AdmissionDone, Token: 1, Pane: "h1"})
	if fx := step(t, r, RegistryEvent{Kind: ConnectFailed, Token: 1, Pane: "h0", Err: errors.New("stale")}); len(fx) != 0 || !r.Connected()[b] {
		t.Fatalf("a failure for another pane applied: %+v", fx)
	}
	fx := step(t, r, RegistryEvent{Kind: ConnectFailed, Token: 1, Pane: "h1", Err: errors.New("actor capacity reached")})
	if len(fx) != 1 || fx[0].Kind != EffectScheduleRetry || r.Connected()[b] {
		t.Fatalf("connect failure effects %+v connected=%v", fx, r.Connected()[b])
	}
	if fx := step(t, r, RegistryEvent{Kind: RetryDue, Token: 1, Attempt: 1}); len(fx) != 1 || fx[0].Kind != EffectAdmit {
		t.Fatalf("retry %+v", fx)
	}
}

// BR-8: a newer wrapper for the slot that never passes admission must not
// strand the older, working one.
func TestRegistryUnadmittedNewerSessionDisplacesNothing(t *testing.T) {
	r := NewRegistry()
	old, ghost := registryBinding("pair:1", "a", 10), registryBinding("pair:1", "a", 11)
	step(t, r, RegistryEvent{Kind: PaneChanged, Thread: old.Thread(), Pane: "h1"})
	step(t, r, RegistryEvent{Kind: SessionOpened, Token: 1, Binding: old})
	step(t, r, RegistryEvent{Kind: SessionOpened, Token: 2, Binding: ghost})
	// The older admission completes while the newer is still being checked.
	step(t, r, RegistryEvent{Kind: AdmissionDone, Token: 1, Pane: "h1"})
	if !r.Connected()[old] {
		t.Fatal("older session stranded by an unadmitted newer one")
	}
	for i := 0; i <= len(RetryDelays); i++ {
		step(t, r, RegistryEvent{Kind: AdmissionDone, Token: 2, Pane: "h1", Err: errors.New("ghost")})
		step(t, r, RegistryEvent{Kind: RetryDue, Token: 2, Attempt: i + 1})
	}
	if p, _ := r.Phase(2); p != Dormant || !r.Connected()[old] {
		t.Fatalf("ghost %v, old connected %v", p, r.Connected()[old])
	}
	// A newer session that is admitted does take over.
	step(t, r, RegistryEvent{Kind: SendTargeted, Slot: "pair:1"})
	fx := step(t, r, RegistryEvent{Kind: AdmissionDone, Token: 2, Pane: "h1"})
	if want := []RegistryEffectKind{EffectDisconnect, EffectConnect}; fmt.Sprint(kinds(fx)) != fmt.Sprint(want) || r.Connected()[old] || !r.Connected()[ghost] {
		t.Fatalf("takeover %v", kinds(fx))
	}
}

func TestRegistryCapacity(t *testing.T) {
	r := NewRegistry()
	for i := 0; i < MaxActors; i++ {
		step(t, r, RegistryEvent{Kind: SessionOpened, Token: SessionToken(i + 1), Binding: registryBinding(fmt.Sprintf("pair:%d", i), "a", 10+i)})
	}
	if _, err := r.Advance(RegistryEvent{Kind: SessionOpened, Token: 999, Binding: registryBinding("pair:999", "a", 9)}); !errors.Is(err, ErrRegistryFull) {
		t.Fatalf("capacity err %v", err)
	}
}

// TestRegistryInterleavingsKeepInvariants drives random event orders —
// including late admission results, stale retries, coalesced pane changes,
// exec re-registration and replacement — and checks after every step that the
// broker, driven only by the emitted effects, agrees with the registry.
func TestRegistryInterleavingsKeepInvariants(t *testing.T) {
	threads := []string{"a", "b"}
	slotOf := map[string]string{"a": "pair:1", "b": "pair:2"}
	panes := []PaneHandle{"", "h1", "h2", "h3"}
	seen := map[RegistryEffectKind]int{}
	displacedConnected := 0
	for seed := int64(0); seed < 400; seed++ {
		rng := rand.New(rand.NewSource(seed))
		r := NewRegistry()
		broker := map[Binding]bool{}
		open := map[SessionToken]Binding{}
		var pendingAdmits []RegistryEffect
		var pendingRetries []RegistryEffect
		next := SessionToken(1)
		var trace []string
		for step := 0; step < 60; step++ {
			var e RegistryEvent
			switch rng.Intn(7) {
			case 0: // a wrapper starts; sometimes an exec with an identical binding
				tag := threads[rng.Intn(len(threads))]
				b := registryBinding(slotOf[tag], tag, 10+rng.Intn(2))
				e = RegistryEvent{Kind: SessionOpened, Token: next, Binding: b}
				open[next] = b
				next++
			case 1:
				for tok := range open {
					e = RegistryEvent{Kind: SessionClosed, Token: tok}
					delete(open, tok)
					break
				}
			case 2:
				tag := threads[rng.Intn(len(threads))]
				e = RegistryEvent{Kind: PaneChanged, Thread: ThreadKey{"scope", tag}, Pane: panes[rng.Intn(len(panes))]}
			case 3, 4: // an outstanding admission completes, in any order
				if len(pendingAdmits) > 0 {
					i := rng.Intn(len(pendingAdmits))
					a := pendingAdmits[i]
					pendingAdmits = append(pendingAdmits[:i], pendingAdmits[i+1:]...)
					var err error
					if rng.Intn(3) == 0 {
						err = errors.New("check failed")
					}
					e = RegistryEvent{Kind: AdmissionDone, Token: a.Token, Pane: a.Pane, Err: err}
				}
			case 5:
				if len(pendingRetries) > 0 {
					i := rng.Intn(len(pendingRetries))
					a := pendingRetries[i]
					pendingRetries = append(pendingRetries[:i], pendingRetries[i+1:]...)
					e = RegistryEvent{Kind: RetryDue, Token: a.Token, Attempt: a.Attempt}
				}
			case 6:
				tag := threads[rng.Intn(len(threads))]
				e = RegistryEvent{Kind: SendTargeted, Slot: slotOf[tag]}
			}
			if e == (RegistryEvent{}) {
				continue
			}
			trace = append(trace, fmt.Sprintf("k%d t%d %s/%d %s/%s err=%v a%d", e.Kind, e.Token, e.Binding.Slot, e.Binding.PID, e.Thread.Tag, e.Pane, e.Err != nil, e.Attempt))
			fx, err := r.Advance(e)
			if err != nil {
				t.Fatalf("seed %d: %v\n%v", seed, err, trace)
			}
			for _, f := range fx {
				seen[f.Kind]++
				if f.Kind == EffectDisconnect && r.sessions[f.Token] != nil && r.sessions[f.Token].phase == Displaced {
					displacedConnected++
				}
				switch f.Kind {
				case EffectAdmit:
					if r.panes[f.Binding.Thread()] != f.Pane {
						t.Fatalf("seed %d: Admit against a non-current pane\n%v", seed, trace)
					}
					pendingAdmits = append(pendingAdmits, f)
				case EffectScheduleRetry:
					if f.Attempt > len(RetryDelays) {
						t.Fatalf("seed %d: retry past the bound", seed)
					}
					pendingRetries = append(pendingRetries, f)
				case EffectConnect:
					if rng.Intn(5) == 0 {
						// The broker refuses: the failure comes back as an
						// event and the broker never held the actor.
						fx2, err := r.Advance(RegistryEvent{Kind: ConnectFailed, Token: f.Token, Pane: f.Pane, Err: errors.New("refused")})
						if err != nil {
							t.Fatal(err)
						}
						for _, g := range fx2 {
							if g.Kind == EffectScheduleRetry {
								pendingRetries = append(pendingRetries, g)
							}
						}
						continue
					}
					// The broker displaces same-slot actors on Register; the
					// registry must already have disconnected them.
					for other := range broker {
						if other != f.Binding && sameSlot(other, f.Binding) {
							t.Fatalf("seed %d: Connect relied on broker displacement of %v\n%v", seed, other, trace)
						}
					}
					broker[f.Binding] = true
				case EffectDisconnect:
					delete(broker, f.Binding)
				}
			}
			want := r.Connected()
			if fmt.Sprint(want) != fmt.Sprint(broker) {
				t.Fatalf("seed %d: broker %v != registry %v\n%v", seed, broker, want, trace)
			}
			slots := map[string]bool{}
			for b := range want {
				if slots[b.Slot] {
					t.Fatalf("seed %d: two connected bindings for %s", seed, b.Slot)
				}
				slots[b.Slot] = true
			}
			// Newest wins: an older session may stay connected while a newer
			// one is still being checked (delivery can complete at the old
			// incarnation), but never once a newer one is admitted.
			for tok, s := range r.sessions {
				if !r.connected(s) {
					continue
				}
				for other, o := range r.sessions {
					if other > tok && sameSlot(o.binding, s.binding) && o.phase == Admitted {
						t.Fatalf("seed %d: session %d connected under admitted newer %d\n%v", seed, tok, other, trace)
					}
				}
			}
		}
	}
	// The generator must reach every effect and the displacement path, or the
	// invariants above held over nothing.
	for _, k := range []RegistryEffectKind{EffectAdmit, EffectConnect, EffectDisconnect, EffectScheduleRetry} {
		if seen[k] < 50 {
			t.Fatalf("effect %d seen only %d times: %v", k, seen[k], seen)
		}
	}
	if displacedConnected < 10 {
		t.Fatalf("displacement of a connected session exercised %d times", displacedConnected)
	}
}

// #421: Liveness reports an admitted session's own claims, unknown for a
// legacy session, and nothing for a slot it cannot pin to one incarnation.
func TestRegistryLiveness(t *testing.T) {
	r := NewRegistry()
	admit := func(token SessionToken, b Binding, build *BuildIdentity, pane PaneHandle) {
		step(t, r, RegistryEvent{Kind: PaneChanged, Thread: b.Thread(), Pane: pane})
		step(t, r, RegistryEvent{Kind: SessionOpened, Token: token, Binding: b, Build: build})
		step(t, r, RegistryEvent{Kind: AdmissionDone, Token: token, Pane: pane})
	}
	modern := registryBinding("pair:1", "a", 101)
	legacy := registryBinding("pair:2", "b", 102)
	admit(1, modern, &BuildIdentity{SHA256: strings.Repeat("0", 64)}, "h1")
	admit(2, legacy, nil, "h2")
	if l := r.Liveness()["pair:1"]; l.Build == nil || l.Settled != nil {
		t.Fatalf("modern before any report: %+v", l)
	}
	yes := true
	step(t, r, RegistryEvent{Kind: SessionActivity, Token: 1, Settled: &yes})
	step(t, r, RegistryEvent{Kind: SessionActivity, Token: 2, Settled: &yes}) // ignored: legacy
	if l := r.Liveness()["pair:1"]; l.Settled == nil || !*l.Settled || l.Binding != modern {
		t.Fatalf("modern after report: %+v", l)
	}
	if l, ok := r.Liveness()["pair:2"]; !ok || l.Build != nil || l.Settled != nil {
		t.Fatalf("legacy must stay unknown: %+v %v", l, ok)
	}
	// A new incarnation for pair:1 displaces the old one (one admitted session
	// per slot), and its own claims replace the old ones: no inherited Settled.
	next := registryBinding("pair:1", "c", 103)
	admit(3, next, nil, "h3")
	if l, ok := r.Liveness()["pair:1"]; !ok || l.Binding != next || l.Build != nil || l.Settled != nil {
		t.Fatalf("new incarnation: %+v %v", l, ok)
	}
}

func TestRegistryLastWorkBelongsToTheAdmittedSession(t *testing.T) {
	r := NewRegistry()
	b := registryBinding("pair:1", "peer-only", 101)
	build := &BuildIdentity{SHA256: strings.Repeat("0", 64)}
	admit := func(token SessionToken, build *BuildIdentity) {
		step(t, r, RegistryEvent{Kind: SessionOpened, Token: token, Binding: b, Build: build})
		step(t, r, RegistryEvent{Kind: AdmissionDone, Token: token})
	}
	admit(1, build)
	busy, settled := false, true
	at := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	report := func(token SessionToken, state *bool, observed time.Time) {
		// No genuine operator submission or Pair-log activity is involved.
		step(t, r, RegistryEvent{Kind: SessionActivity, Token: token, Settled: state, At: observed})
	}
	report(1, &busy, at)
	if got := r.Liveness()[b.Slot].LastWorkAt; !got.Equal(at) {
		t.Fatalf("peer-only work = %v, want %v", got, at)
	}
	done := at.Add(30 * time.Hour)
	report(1, &settled, done) // silent command wait ends without transcript/output change
	for _, refresh := range []time.Time{done.Add(-time.Hour), done.Add(48 * time.Hour)} {
		report(1, &settled, refresh)
		if got := r.Liveness()[b.Slot].LastWorkAt; !got.Equal(done) {
			t.Fatalf("settled refresh moved last work to %v, want completion %v", got, done)
		}
	}
	admit(2, build) // even a byte-identical binding is a new connection
	report(2, &settled, done.Add(50*time.Hour))
	report(1, &busy, done.Add(51*time.Hour)) // displaced source cannot lend work
	if got := r.Liveness()[b.Slot].LastWorkAt; !got.IsZero() {
		t.Fatalf("replacement inherited old session activity: %v", got)
	}
	step(t, r, RegistryEvent{Kind: SessionClosed, Token: 2})
	if _, ok := r.Liveness()[b.Slot]; ok {
		t.Fatal("disconnected session kept activity authority")
	}
	admit(3, nil)
	report(3, &busy, done)
	if got := r.Liveness()[b.Slot]; got.Settled != nil || !got.LastWorkAt.IsZero() {
		t.Fatalf("legacy session gained work evidence: %+v", got)
	}
}

func TestRegistryReconnectSeedsWorkWithoutRenewingSettledRefreshes(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name           string
		activity, want time.Time
	}{
		{"recent completed peer turn", now.Add(-time.Hour), now.Add(-time.Hour)},
		{"old idle stays old", now.Add(-4 * 24 * time.Hour), now.Add(-4 * 24 * time.Hour)},
		{"unknown stays unknown", time.Time{}, time.Time{}},
		{"future wrapper clock bounded", now.Add(time.Hour), now},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRegistry()
			b := registryBinding("pair:1", "peer-only", 101)
			step(t, r, RegistryEvent{Kind: SessionOpened, Token: 1, Binding: b, Build: &BuildIdentity{SHA256: strings.Repeat("a", 64)}})
			step(t, r, RegistryEvent{Kind: AdmissionDone, Token: 1})
			settled := true
			step(t, r, RegistryEvent{Kind: SessionActivity, Token: 1, Settled: &settled, At: now, Observation: Observation{LastActivity: tc.activity}})
			for _, event := range []RegistryEvent{
				{Kind: PaneChanged, Thread: b.Thread(), Pane: "new-client"},
				{Kind: SessionActivity, Token: 1, Settled: &settled, At: now.Add(48 * time.Hour), Observation: Observation{LastActivity: now.Add(48 * time.Hour)}},
			} {
				if got := r.Liveness()[b.Slot].LastWorkAt; !got.Equal(tc.want) {
					t.Fatalf("seed = %v, want %v", got, tc.want)
				}
				step(t, r, event)
			}
			if got := r.Liveness()[b.Slot].LastWorkAt; !got.Equal(tc.want) {
				t.Fatalf("settled refresh renewed seed to %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRegistryWorkerCueSnapshotOwnedAndReconnectNotNewLaunch(t *testing.T) {
	r := NewRegistry()
	b := registryBinding("pair:1", "a", 7)
	cue := &WorkerCueReport{LaunchOrdinal: 7, State: WorkerCueNeeded}
	step(t, r, RegistryEvent{Kind: SessionOpened, Token: 1, Binding: b, Build: &testBuild, WorkerCue: cue})
	step(t, r, RegistryEvent{Kind: AdmissionDone, Token: 1})
	cue.State = WorkerCueUncertain
	if r.Liveness()[b.Slot].WorkerCue.State != WorkerCueNeeded {
		t.Fatal("open aliased caller")
	}
	committed := &WorkerCueReport{LaunchOrdinal: 7, State: WorkerCueCommitted, ID: WorkerStartNoticeID(b, 7)}
	step(t, r, RegistryEvent{Kind: SessionActivity, Token: 1, WorkerCue: committed})
	snapshot := r.Liveness()[b.Slot]
	snapshot.WorkerCue.State = WorkerCueNeeded
	if r.Liveness()[b.Slot].WorkerCue.State != WorkerCueCommitted {
		t.Fatal("snapshot aliases registry")
	}
	step(t, r, RegistryEvent{Kind: SessionOpened, Token: 2, Binding: b, Build: &testBuild, WorkerCue: committed})
	step(t, r, RegistryEvent{Kind: AdmissionDone, Token: 2})
	step(t, r, RegistryEvent{Kind: SessionActivity, Token: 1, WorkerCue: &WorkerCueReport{LaunchOrdinal: 99, State: WorkerCueNeeded}})
	if got := r.Liveness()[b.Slot].WorkerCue; *got != *committed {
		t.Fatalf("stale session/reconnect changed launch: %+v", got)
	}
	step(t, r, RegistryEvent{Kind: SessionOpened, Token: 3, Binding: b})
	step(t, r, RegistryEvent{Kind: AdmissionDone, Token: 3})
	if r.Liveness()[b.Slot].WorkerCue != nil {
		t.Fatal("legacy session inherited cue capability")
	}
}
