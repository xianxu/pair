package couchcmd

import (
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"testing"
	"time"
)

func TestWrapperActivityBackgroundWork(t *testing.T) {
	now := time.Now()
	old := now.Add(-72 * time.Hour)
	yes, no := true, false
	row := couchcore.ActionableThreadSummary{Address: couchcore.ThreadAddress{RepoScope: "scope", Tag: "background"}, Agent: "claude"}
	b := couchmessage.Binding{Scope: "scope", Tag: "background", Agent: "claude"}
	for _, tc := range []struct {
		name                string
		background, settled *bool
		binding             couchmessage.Binding
		want                time.Time
	}{
		{"settled silent background", &yes, &yes, b, now},
		{"nil settled background", &yes, nil, b, now},
		{"completed background", &no, &yes, b, old},
		{"unknown with legacy settled", nil, nil, b, time.Time{}},
		{"background false legacy settled", &no, nil, b, time.Time{}},
		{"wrong scope", &yes, &yes, couchmessage.Binding{Scope: "other", Tag: b.Tag, Agent: b.Agent}, time.Time{}},
		{"wrong tag", &yes, &yes, couchmessage.Binding{Scope: b.Scope, Tag: b.Tag + "extra", Agent: b.Agent}, time.Time{}},
		{"wrong agent", &yes, &yes, couchmessage.Binding{Scope: b.Scope, Tag: b.Tag, Agent: "codex"}, time.Time{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &messageService{authority: messageAuthority{now: func() time.Time { return now }}}
			live := map[string]couchmessage.SlotLiveness{"pair:1": {Binding: tc.binding, Settled: tc.settled, BackgroundWork: tc.background, LastWorkAt: old}}
			s.liveness.Store(&live)
			if got := s.WrapperActivity(row); !got.Equal(tc.want) {
				t.Fatalf("activity = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestBackgroundWorkCrossesSessionTransport(t *testing.T) {
	w := newWaitRig(t, true, true, nil)
	yes, no := true, false
	wait := func(want *bool) couchmessage.SlotLiveness {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			live, ok := w.s.LivenessForThread(w.worker.Scope, w.worker.Tag)
			if ok && (live.BackgroundWork == nil) == (want == nil) && (want == nil || *live.BackgroundWork == *want) {
				if live.Settled == nil || !*live.Settled {
					t.Fatal("background hint changed settled")
				}
				return live
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("background hint never crossed transport")
		return couchmessage.SlotLiveness{}
	}
	w.client.BackgroundWork(&yes)
	active := wait(&yes)
	beforeCompletion := time.Now()
	w.client.BackgroundWork(&no)
	completed := wait(&no)
	if completed.LastWorkAt.Before(beforeCompletion) || completed.LastWorkAt.Before(active.LastWorkAt) {
		t.Fatalf("completion did not stamp monotonic server work: %+v", completed)
	}
	w.client.BackgroundWork(nil)
	unknown := wait(nil)
	if !unknown.LastWorkAt.Equal(completed.LastWorkAt) {
		t.Fatal("unknown invented completion")
	}
}
