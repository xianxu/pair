package couchcmd

import (
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestWrapperActivityUsesAdmittedSessionWithoutOperatorSends(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	old := now.Add(-4 * 24 * time.Hour)
	busy, settled := false, true
	row := couchcore.ActionableThreadSummary{Address: couchcore.ThreadAddress{RepoScope: "0123456789abcdef", Tag: "peer-only"}, Agent: "codex", State: couchcore.ThreadLive}
	binding := couchmessage.Binding{Scope: row.Address.RepoScope, Tag: string(row.Address.Tag), Agent: row.Agent}
	for _, tc := range []struct {
		name string
		live *couchmessage.SlotLiveness
		want time.Time
	}{
		{"silent peer turn and background wait", &couchmessage.SlotLiveness{Binding: binding, Settled: &busy, LastWorkAt: old}, now},
		{"peer turn completed", &couchmessage.SlotLiveness{Binding: binding, Settled: &settled, LastWorkAt: now.Add(-time.Minute)}, now.Add(-time.Minute)},
		{"idle over a day", &couchmessage.SlotLiveness{Binding: binding, Settled: &settled, LastWorkAt: old}, old},
		{"unknown legacy", &couchmessage.SlotLiveness{Binding: binding}, time.Time{}},
		{"disconnected", nil, time.Time{}},
		{"other scope", &couchmessage.SlotLiveness{Binding: couchmessage.Binding{Scope: "other", Tag: binding.Tag, Agent: binding.Agent}, Settled: &busy}, time.Time{}},
		{"tag prefix", &couchmessage.SlotLiveness{Binding: couchmessage.Binding{Scope: binding.Scope, Tag: binding.Tag + "-extra", Agent: binding.Agent}, Settled: &busy}, time.Time{}},
		{"other agent", &couchmessage.SlotLiveness{Binding: couchmessage.Binding{Scope: binding.Scope, Tag: binding.Tag, Agent: "claude"}, Settled: &busy}, time.Time{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &messageService{authority: messageAuthority{now: func() time.Time { return now }}}
			live := map[string]couchmessage.SlotLiveness{}
			if tc.live != nil {
				live["pair:1"] = *tc.live
			}
			s.liveness.Store(&live)
			if got := s.WrapperActivity(row); !got.Equal(tc.want) {
				t.Fatalf("wrapper activity = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestWrapperActivityCompletionCrossesTheExistingSessionTransport(t *testing.T) {
	w := newWaitRig(t, false, false, nil)
	live, ok := w.s.LivenessForThread(w.worker.Scope, w.worker.Tag)
	if !ok || live.LastWorkAt.IsZero() {
		t.Fatalf("busy session did not record server-time work: %+v", live)
	}
	beforeCompletion := time.Now()
	w.client.Settle(true) // no output, operator submission or transcript write
	w.waitSettled(w.worker, true)
	live, _ = w.s.LivenessForThread(w.worker.Scope, w.worker.Tag)
	if live.LastWorkAt.Before(beforeCompletion) {
		t.Fatalf("silent completion kept old activity %v (before %v)", live.LastWorkAt, beforeCompletion)
	}
	row := couchcore.ActionableThreadSummary{Address: couchcore.ThreadAddress{RepoScope: w.worker.Scope, Tag: couchcore.ThreadTag(w.worker.Tag)}, Agent: w.worker.Agent}
	if got := w.s.WrapperActivity(row); !got.Equal(live.LastWorkAt) {
		t.Fatalf("settled read moved completion time: %v, want %v", got, live.LastWorkAt)
	}
}
