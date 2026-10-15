package couchcore

import (
	"context"
	"testing"
)

func TestExplicitParkAfterWarmAttachRefusesIndependentLifecycleOwner(t *testing.T) {
	for _, shape := range []string{"live and hosted", "start this couch is driving"} {
		for _, owner := range []Liveness{Live, Unknown} {
			t.Run(shape+"/"+owner.String(), func(t *testing.T) {
				var record ThreadRecord
				for _, tc := range everyThreadShape(t) {
					if tc.name == shape {
						record = tc.record
						break
					}
				}
				env, before, session := readonlyWarmFixture(t, record, owner)
				actor, h := assertReadonlyWarmAttach(t, env, before, session)
				lifecycle := withParkLifecycle(t, env, before, actor.PID, actor.Identity)
				env.Artifacts.SetPairSession(before.Address, session, true)
				if _, err := env.Couch.Park(context.Background(), before.Address, ""); err == nil {
					t.Fatal("explicit park took over independently owned lifecycle")
				}
				if len(lifecycle.Lifecycle.trace) != 0 {
					t.Fatalf("refusal reached quit transaction: %v", lifecycle.Lifecycle.trace)
				}
				if !h.Alive() || len(env.Proc.Signals) != 0 {
					t.Fatal("refusal stopped a process")
				}
				assertWarmRecordUnchanged(t, env, before)
			})
		}
	}
}

func TestExplicitParkAfterWarmAttachRequiresExactLocalHelper(t *testing.T) {
	for _, state := range []string{"unknown", "reused"} {
		t.Run(state, func(t *testing.T) {
			env, before, session := readonlyWarmFixture(t, warmDetachedRecord(t), Live)
			actor, h := assertReadonlyWarmAttach(t, env, before, session)
			lifecycle := withParkLifecycle(t, env, before, actor.PID, actor.Identity)
			env.Artifacts.SetPairSession(before.Address, session, true)
			if state == "unknown" {
				env.Proc.SetUnknown(actor.PID)
			} else {
				env.Proc.Set(actor.PID, "reused-start-token")
			}
			if _, err := env.Couch.Park(context.Background(), before.Address, ""); err == nil {
				t.Fatal("unproved helper adopted")
			}
			if len(lifecycle.Lifecycle.trace) != 0 || !h.Alive() || len(env.Proc.Signals) != 0 {
				t.Fatal("refusal caused lifecycle effects")
			}
			assertWarmRecordUnchanged(t, env, before)
		})
	}
}
