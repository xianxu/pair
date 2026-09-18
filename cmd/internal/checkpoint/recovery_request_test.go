package checkpoint

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func recoveryRequest(t *testing.T) Request {
	t.Helper()
	cp, err := New("/other-worktree/checkpoint.md", "---\ntype: continuation\nagent: codex\n---\n## NEXT ACTION\nrecover this exact task\n")
	if err != nil {
		t.Fatal(err)
	}
	return Request{Version: Version, ID: RequestID("scope", "tag", 3, cp.Digest), Checkpoint: cp, Source: Source{Agent: "codex", Session: "pair-source", LaunchOrdinal: 3}, CreatedAt: time.Unix(10, 0).UTC(), Phase: Pending, SourceAbsence: &SourceAbsence{Session: "pair-source", LaunchOrdinal: 3, ObservedAt: time.Unix(11, 0).UTC(), RecordRevision: 4}}
}

func TestRecoveryRequestAuthorityValidation(t *testing.T) {
	base := recoveryRequest(t)
	if err := base.Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*Request){
		"missing authority and helper": func(r *Request) { r.SourceAbsence = nil },
		"partial helper":               func(r *Request) { r.Source.Helper.PID = 7 },
		"both authorities":             func(r *Request) { r.SourcePark = "park" },
		"foreign session":              func(r *Request) { r.SourceAbsence.Session = "other" },
		"foreign generation":           func(r *Request) { r.SourceAbsence.LaunchOrdinal++ },
		"missing observation":          func(r *Request) { r.SourceAbsence.ObservedAt = time.Time{} },
		"missing revision":             func(r *Request) { r.SourceAbsence.RecordRevision = 0 },
		"pending attempt":              func(r *Request) { r.Attempt = "a" },
	} {
		t.Run(name, func(t *testing.T) {
			r := base.Clone()
			mutate(&r)
			if err := r.Validate(); err == nil {
				t.Fatal("invalid recovery request accepted")
			}
		})
	}
	raw, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Request
	if err = json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if err = decoded.Validate(); err != nil || !reflect.DeepEqual(decoded, base) {
		t.Fatalf("round trip = %+v %v", decoded, err)
	}
	clone := base.Clone()
	clone.SourceAbsence.Session = "changed"
	if base.SourceAbsence.Session != "pair-source" {
		t.Fatal("source absence clone aliases original")
	}
}

func TestRecoverySourceAbsentPhaseTransitions(t *testing.T) {
	for _, phase := range AllPhases() {
		t.Run(string(phase), func(t *testing.T) {
			r := recoveryRequest(t)
			absence := *r.SourceAbsence
			r.SourceAbsence = nil
			r.Source.Helper = Process{PID: 42, Identity: "source"}
			r.Phase = phase
			if phase != Pending {
				r.Attempt = "a"
			}
			if phase == Failed {
				r.Failure = "source vanished"
			}
			if phase == Complete {
				r.SourcePark = "park"
				r.Target = &Target{Process: Process{PID: 43, Identity: "target"}, ObservedAt: time.Unix(12, 0)}
			}
			before := r.Clone()
			next, err := Advance(r, Event{Kind: SourceAbsent, RequestID: r.ID, Attempt: r.Attempt, SourceAbsence: &absence})
			if phase == Complete {
				if err == nil {
					t.Fatal("complete authority changed")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if next.Phase != phase || next.Attempt != r.Attempt || next.Failure != r.Failure || next.SourceAbsence == nil || !reflect.DeepEqual(r, before) {
				t.Fatalf("absence changed lifecycle or input: %+v", next)
			}
			if _, err := Advance(r, Event{Kind: SourceAbsent, RequestID: r.ID, Attempt: "obsolete", SourceAbsence: &absence}); err == nil {
				t.Fatal("obsolete attempt accepted")
			}
		})
	}
}

func TestRecoveryTargetGenerationSurvivesSuccessiveRetries(t *testing.T) {
	r := recoveryRequest(t)
	next, err := Advance(r, Event{Kind: Begin, RequestID: r.ID, Attempt: "a"})
	if err != nil {
		t.Fatal(err)
	}
	r = next
	for i, attempt := range []string{"a", "b"} {
		generation := TargetGeneration{Agent: "codex", Session: "pair-source", Attempt: attempt, LaunchOrdinal: uint64(4 + i)}
		target := Process{PID: 43 + i, Identity: attempt}
		r, err = Advance(r, Event{Kind: Registered, RequestID: r.ID, Attempt: attempt, At: time.Unix(20+int64(i), 0), Target: &target, TargetGeneration: &generation})
		if err != nil {
			t.Fatal(err)
		}
		if r.Target.Generation == nil || *r.Target.Generation != generation {
			t.Fatalf("lost generation: %+v", r)
		}
		clone := r.Clone()
		clone.Target.Generation.Session = "other"
		if r.Target.Generation.Session != "pair-source" {
			t.Fatal("target generation alias")
		}
		r, err = Advance(r, Event{Kind: Fail, RequestID: r.ID, Attempt: attempt, Failure: "target dead"})
		if err != nil {
			t.Fatal(err)
		}
		r, err = Advance(r, Event{Kind: RetryAbsent, RequestID: r.ID, Attempt: string(rune('b' + i))})
		if err != nil {
			t.Fatal(err)
		}
		if r.Target != nil || r.PreviousTargetGeneration == nil || *r.PreviousTargetGeneration != generation || r.Source.LaunchOrdinal != 3 {
			t.Fatalf("bad retry witness: %+v", r)
		}
		clone = r.Clone()
		clone.PreviousTargetGeneration.Session = "other"
		if r.PreviousTargetGeneration.Session != "pair-source" {
			t.Fatal("previous generation alias")
		}
	}
}

func TestRecoveryTargetGenerationRejectsForeignWitness(t *testing.T) {
	r := recoveryRequest(t)
	r.Phase = Running
	r.Attempt = "a"
	target := Process{PID: 43, Identity: "target"}
	for name, mutate := range map[string]func(*TargetGeneration){
		"agent": func(g *TargetGeneration) { g.Agent = "claude" }, "session": func(g *TargetGeneration) { g.Session = "other" }, "attempt": func(g *TargetGeneration) { g.Attempt = "other" }, "ordinal": func(g *TargetGeneration) { g.LaunchOrdinal = 3 },
	} {
		t.Run(name, func(t *testing.T) {
			g := TargetGeneration{Agent: "codex", Session: "pair-source", Attempt: "a", LaunchOrdinal: 4}
			mutate(&g)
			if _, err := Advance(r, Event{Kind: Registered, RequestID: r.ID, Attempt: "a", At: time.Unix(20, 0), Target: &target, TargetGeneration: &g}); err == nil {
				t.Fatal("foreign generation accepted")
			}
		})
	}
}

func TestRecoveryPreviousGenerationRequiresSourceAuthority(t *testing.T) {
	r := recoveryRequest(t)
	r.SourceAbsence = nil
	r.Source.Helper = Process{PID: 42, Identity: "source"}
	r.Phase = Running
	r.Attempt = "b"
	r.PreviousTargetGeneration = &TargetGeneration{Agent: "codex", Session: "pair-source", Attempt: "a", LaunchOrdinal: 4}
	if err := r.Validate(); err == nil {
		t.Fatal("previous target accepted without any source retirement authority")
	}
}

func TestRecoveryRegisteredGenerationCannotBeReplacedOrErased(t *testing.T) {
	r := recoveryRequest(t)
	r.Phase = Running
	r.Attempt = "a"
	r.Target = &Target{Process: Process{PID: 43, Identity: "old-helper"}, ObservedAt: time.Unix(12, 0), Generation: &TargetGeneration{Agent: "codex", Session: "pair-source", Attempt: "a", LaunchOrdinal: 4}}
	before := r.Clone()
	replacement := Process{PID: 44, Identity: "reattached-helper"}
	next, err := Advance(r, Event{Kind: Registered, RequestID: r.ID, Attempt: r.Attempt, Target: &replacement, At: time.Unix(20, 0)})
	if err != nil || next.Target.Generation == nil || *next.Target.Generation != *r.Target.Generation || next.Target.ObservedAt != r.Target.ObservedAt {
		t.Fatalf("helper reattachment lost original target proof: %+v %v", next, err)
	}
	changed := *r.Target.Generation
	changed.LaunchOrdinal++
	if _, err := Advance(r, Event{Kind: Registered, RequestID: r.ID, Attempt: r.Attempt, Target: &replacement, TargetGeneration: &changed, At: time.Unix(20, 0)}); err == nil {
		t.Fatal("same attempt changed target generation")
	}
	if !reflect.DeepEqual(r, before) {
		t.Fatal("registration mutated request input")
	}
}

func TestRecoveryRetryAbsentAdoptsExactGenerationReceipt(t *testing.T) {
	r := recoveryRequest(t)
	r.Phase = Failed
	r.Attempt = "a"
	r.Failure = "owner exited before saving target"
	generation := TargetGeneration{Agent: "codex", Session: "pair-source", Attempt: "a", LaunchOrdinal: 4}
	next, err := Advance(r, Event{Kind: RetryAbsent, RequestID: r.ID, Attempt: "b", TargetGeneration: &generation})
	if err != nil || next.PreviousTargetGeneration == nil || *next.PreviousTargetGeneration != generation {
		t.Fatalf("retry lost recovered receipt: %+v %v", next, err)
	}
	generation.Attempt = "foreign"
	if _, err := Advance(r, Event{Kind: RetryAbsent, RequestID: r.ID, Attempt: "b", TargetGeneration: &generation}); err == nil {
		t.Fatal("retry accepted foreign attempt receipt")
	}
}
