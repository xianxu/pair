package couchcmd

import (
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

func TestConditionPickerReconcilesOwnerWithoutPickerChange(t *testing.T) {
	w := newWaitRig(t, false, true, recordedPermissionTail)
	var mu sync.Mutex
	latest := map[couchcore.ThreadAddress]couchcore.AttentionCondition{}
	w.s.SetAttention(func(f couchcore.AttentionFact) {
		if f.Kind == couchcore.AttentionPrompt {
			mu.Lock()
			latest = f.Live
			mu.Unlock()
		}
	})
	w.reportPicker(true)
	live := *w.s.liveness.Load()
	w.s.publishBlocked(live)
	mu.Lock()
	n := len(latest)
	mu.Unlock()
	if n != 0 {
		t.Fatalf("connected owner picker entered bell: %d", n)
	}
	w.disconnect(w.hub)
	w.s.publishBlocked(live)
	mu.Lock()
	_, present := latest[bindingThread(w.worker)]
	mu.Unlock()
	if !present {
		t.Fatal("unchanged picker failed to gain unavailable-owner fallback")
	}
	w.v2wrapper(w.hub, "h1", false)
	w.s.publishBlocked(live)
	mu.Lock()
	n = len(latest)
	mu.Unlock()
	if n != 0 {
		t.Fatal("owner return failed to clear unchanged picker fallback")
	}
}

func TestConditionQuestionWithoutHubIsJudgedAndFallsBack(t *testing.T) {
	w := newWaitRig(t, true, true, claudeQuestionTail)
	w.disconnect(w.hub)
	var mu sync.Mutex
	var latest couchcore.AttentionFact
	w.s.SetAttention(func(f couchcore.AttentionFact) {
		if f.Kind == couchcore.AttentionQuestion {
			mu.Lock()
			latest = f
			mu.Unlock()
		}
	})
	w.tickAt(11 * time.Minute)
	mu.Lock()
	_, present := latest.Live[bindingThread(w.worker)]
	mu.Unlock()
	if !present {
		t.Fatal("no-hub question was never classified for fallback")
	}
	if _, question := questionEpisode(w.watcher.seen[w.worker]); !question {
		t.Fatal("no-hub question episode missing")
	}
}

func TestConditionQuestionConnectedOwnerStaysOutOfBell(t *testing.T) {
	w := newWaitRig(t, true, true, claudeQuestionTail)
	var mu sync.Mutex
	var latest couchcore.AttentionFact
	w.s.SetAttention(func(f couchcore.AttentionFact) {
		if f.Kind == couchcore.AttentionQuestion {
			mu.Lock()
			latest = f
			mu.Unlock()
		}
	})
	w.tickAt(11 * time.Minute)
	mu.Lock()
	n := len(latest.Live)
	mu.Unlock()
	if n != 0 {
		t.Fatalf("connected-owner question entered bell: %d", n)
	}
	w.disconnect(w.hub)
	w.tickAt(12 * time.Minute)
	mu.Lock()
	_, present := latest.Live[bindingThread(w.worker)]
	mu.Unlock()
	if !present {
		t.Fatal("unchanged question failed to gain unavailable-owner fallback")
	}
}

func TestConditionQuestionUnavailableOwnerLaterUsesCanonicalSendOnce(t *testing.T) {
	w := newWaitRig(t, true, true, claudeQuestionTail)
	w.disconnect(w.hub)
	if got := w.tickAt(11 * time.Minute); len(got) != 0 {
		t.Fatalf("sent without owner: %+v", got)
	}
	w.v2wrapper(w.hub, "h1", false)
	w.waitSettled(w.hub, false)
	if got := w.tickAt(12 * time.Minute); len(got) != 1 {
		t.Fatalf("connected owner did not receive preserved question: %+v", got)
	}
	if got := w.tickAt(13 * time.Minute); len(got) != 1 {
		t.Fatalf("same question sent twice: %+v", got)
	}
}

func TestConditionPickerPublishesTypedEpisodeAndClearOnce(t *testing.T) {
	w := newWaitRig(t, false, true, recordedPermissionTail)
	var mu sync.Mutex
	var notices []couchtty.Notice
	publish := func(n couchtty.Notice) {
		if n.Class == couchcore.NoticeLivePicker {
			mu.Lock()
			notices = append(notices, n)
			mu.Unlock()
		}
	}
	w.s.noticeOutput.Store(&publish)
	w.reportPicker(true)
	live := w.s.liveness.Load()
	w.s.publishBlocked(*live)
	w.reportPicker(false)
	mu.Lock()
	defer mu.Unlock()
	if len(notices) != 2 {
		t.Fatalf("picker publication repeated: %+v", notices)
	}
	first, clear := notices[0], notices[1]
	if first.Thread != bindingThread(w.worker) || first.Label != w.worker.Slot || !first.HubHandled || first.Cleared || first.Condition == "" {
		t.Fatalf("invalid picker provenance: %+v", first)
	}
	if !clear.Cleared || clear.Condition != first.Condition || clear.Thread != first.Thread {
		t.Fatalf("clear changed exact condition identity: %+v", clear)
	}
}

func TestConditionJudgedQuestionKeepsCanonicalDeliveryOwnership(t *testing.T) {
	w := newWaitRig(t, true, true, claudeQuestionTail)
	var notices []couchtty.Notice
	publish := func(n couchtty.Notice) {
		if n.Class == couchcore.NoticeJudgedQuestion {
			notices = append(notices, n)
		}
	}
	w.s.noticeOutput.Store(&publish)
	w.tickAt(11 * time.Minute)
	w.tickAt(12 * time.Minute)
	if len(notices) != 1 || !notices[0].HubHandled || notices[0].Thread != bindingThread(w.worker) || notices[0].Condition == "" {
		t.Fatalf("question publication lost canonical owner: %+v", notices)
	}
}

func conditionDelivery(t *testing.T, w *waitRig) *noticeDelivery {
	t.Helper()
	d := newNoticeDelivery(testConsole())
	d.Bind(w.s)
	publish := d.Publish
	w.s.noticeOutput.Store(&publish)
	t.Cleanup(d.Close)
	return d
}

func TestConditionConnectedHubPickerHasChipWithoutSelfSendOrBell(t *testing.T) {
	w := newWaitRig(t, false, true, recordedPermissionTail)
	d := conditionDelivery(t, w)
	picker := true
	var mu sync.Mutex
	var latest couchcore.AttentionFact
	w.s.SetAttention(func(f couchcore.AttentionFact) {
		if f.Kind == couchcore.AttentionPrompt {
			mu.Lock()
			latest = f
			mu.Unlock()
		}
	})
	w.s.publishBlocked(map[string]couchmessage.SlotLiveness{w.hub.Slot: {Binding: w.hub, Picker: &picker}})
	w.s.blocked.Lock()
	blocked := w.s.blocked.last[bindingThread(w.hub)]
	w.s.blocked.Unlock()
	if !blocked {
		t.Fatal("hub picker lost blocked chip")
	}
	noticeDeliveryEventually(t, func() bool { return noticeDeliveryRetained(d) == 0 })
	mu.Lock()
	n := len(latest.Live)
	mu.Unlock()
	if n != 0 {
		t.Fatal("connected hub picker raised bell")
	}
	w.mu.Lock()
	admitted, delivered := len(w.admitted), len(w.delivered)
	w.mu.Unlock()
	if admitted != 0 || delivered != 0 {
		t.Fatalf("hub picker sent to itself: admitted=%d delivered=%d", admitted, delivered)
	}
}

func TestConditionPickerKeepsTimedCanonicalOwnerWithoutAdapterDuplicate(t *testing.T) {
	w := newWaitRig(t, false, true, recordedPermissionTail)
	conditionDelivery(t, w)
	w.reportPicker(true)
	if got := w.tickAt(50 * time.Second); len(got) != 0 {
		t.Fatalf("adapter bypassed picker threshold: %+v", got)
	}
	if got := w.tickAt(2 * time.Minute); len(got) != 1 {
		t.Fatalf("canonical picker notice missing: %+v", got)
	}
	w.repaint(3*time.Minute, 2)
	if got := w.tickAt(4 * time.Minute); len(got) != 1 {
		t.Fatalf("picker repaint duplicated adapter admission: %+v", got)
	}
}

func TestConditionDisabledOwnerDoesNotGrantFallback(t *testing.T) {
	w := newWaitRig(t, true, true, claudeQuestionTail)
	setTestAutomation(t, w.s, "/repo1", "off")
	var mu sync.Mutex
	eligible := map[couchcore.AttentionKind]int{}
	w.s.SetAttention(func(f couchcore.AttentionFact) { mu.Lock(); eligible[f.Kind] = len(f.Live); mu.Unlock() })
	w.reportPicker(true)
	w.s.publishBlocked(*w.s.liveness.Load())
	w.reportPicker(false)
	w.tickAt(11 * time.Minute)
	mu.Lock()
	defer mu.Unlock()
	if eligible[couchcore.AttentionPrompt] != 0 || eligible[couchcore.AttentionQuestion] != 0 {
		t.Fatalf("disabled connected owner granted fallback: %+v", eligible)
	}
}
