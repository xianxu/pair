package couchcmd

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// A Claude Code 2.1.296 permission prompt as the wrapper renders it (pair#464's
// recorded pair:3 screen, trimmed).
var recordedPermissionTail = []string{
	"──────────────────────────────────────────────────────────────",
	" Bash command",
	" Copy fixtures into testdata and save the prototype patch",
	"╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌",
	" ‹dim›│‹/dim› rm -rf $D/$n",
	"╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌",
	" ⚠ Claude Code will automatically deny this request in 1:53",
	"",
	" Do you want to proceed?",
	" ❯ 1. Yes",
	"   2. No",
	"",
	" Esc to cancel · Tab to amend",
}

// reportPicker has the worker's wrapper report its picker and waits until
// Couch's liveness shows it.
func (w *waitRig) reportPicker(open bool) {
	w.t.Helper()
	w.client.Picker(open)
	deadline := time.Now().Add(3 * time.Second)
	for {
		if live, ok := w.s.LivenessForThread(w.worker.Scope, w.worker.Tag); ok && live.Picker != nil && *live.Picker == open {
			return
		}
		if time.Now().After(deadline) {
			w.t.Fatalf("worker never reported Picker=%v", open)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// pair#465 Done-when: one specific notice per picker episode, after about a
// minute, not the general ten.
func TestPickerNoticeAfterAMinuteNamesThePicker(t *testing.T) {
	w := newWaitRig(t, false, true, recordedPermissionTail)
	w.reportPicker(true)
	if got := w.tickAt(50 * time.Second); len(got) != 0 {
		t.Fatalf("notice before a minute: %+v", got)
	}
	got := w.tickAt(70 * time.Second)
	if len(got) != 1 {
		t.Fatalf("want one notice at 70s, got %d: %+v", len(got), got)
	}
	want := "slot pair:2 waiting 1m behind a picker: (permission prompt) Bash command: rm -rf $D/$n — Do you want to proceed? [1. Yes | 2. No] (couch --peek pair:2"
	if m := got[0]; m.From != w.worker || m.To != w.hub || !strings.HasPrefix(m.Body, want) {
		t.Fatalf("notice %+v\nwant prefix %q", got[0], want)
	}
	// The general notice does not repeat the episode at ten minutes.
	if got := w.tickAt(11 * time.Minute); len(got) != 1 {
		t.Fatalf("episode notified %d times", len(got))
	}
}

// A picker reported partway through an episode the general path already
// notified is the same episode: one notice, not two (plan review, ARCH-ORDER).
func TestPickerArrivingMidEpisodeDoesNotNotifyTwice(t *testing.T) {
	w := newWaitRig(t, false, true, recordedPermissionTail)
	if got := w.tickAt(11 * time.Minute); len(got) != 1 || !strings.Contains(got[0].Body, "on a prompt") {
		t.Fatalf("general notice %+v", got)
	}
	w.reportPicker(true)
	if got := w.tickAt(12 * time.Minute); len(got) != 1 {
		t.Fatalf("the picker re-notified its episode: %+v", got)
	}
}

func TestPickerNoticeMutedSlotIsSilent(t *testing.T) {
	w := newWaitRig(t, false, true, recordedPermissionTail)
	w.reportPicker(true)
	if err := os.WriteFile(w.watcher.optOut, []byte("pair:2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := w.tickAt(2 * time.Minute); len(got) != 0 {
		t.Fatalf("muted slot notified: %+v", got)
	}
}

func TestPickerSummaryBoundsEachPart(t *testing.T) {
	long := strings.Repeat("x", 500)
	got := pickerSummary(couchmessage.Picker{Kind: "permission prompt", Title: "Do you want to proceed?", Detail: "Bash command: " + long, Options: []string{"1. " + long, "2. No"}})
	if !strings.Contains(got, "… — Do you want to proceed? [1. ") || len(got) > maxWaitSummary+maxPickerOptions+64 {
		t.Fatalf("summary %q (%d bytes)", got, len(got))
	}
	if got := pickerSummary(couchmessage.Picker{}); got != "(picker text not recognized)" {
		t.Fatalf("unrecognized: %q", got)
	}
}

// pair#465: the Console hears which threads are behind a picker, and hears
// again when one closes.
func TestBlockedThreadsReachTheConsoleView(t *testing.T) {
	w := newWaitRig(t, false, true, recordedPermissionTail)
	var mu sync.Mutex
	var sets []map[couchcore.ThreadAddress]bool
	w.s.SetBlockedView(func(set map[couchcore.ThreadAddress]bool) {
		mu.Lock()
		sets = append(sets, set)
		mu.Unlock()
	})
	thread := couchcore.ThreadAddress{RepoScope: w.worker.Scope, Tag: couchcore.ThreadTag(w.worker.Tag)}
	last := func() map[couchcore.ThreadAddress]bool {
		mu.Lock()
		defer mu.Unlock()
		if len(sets) == 0 {
			return nil
		}
		return sets[len(sets)-1]
	}
	w.reportPicker(true)
	if !last()[thread] || len(last()) != 1 {
		t.Fatalf("view after the picker opened: %v", last())
	}
	w.reportPicker(false)
	if len(last()) != 0 {
		t.Fatalf("view after the picker closed: %v", last())
	}
}

// A picker that repaints (Claude Code's auto-deny countdown) keeps the slot's
// LastActivity fresh; its wait still runs from when it opened.
func TestPickerNoticeCountsFromOpeningThroughRepaints(t *testing.T) {
	w := newWaitRig(t, false, true, recordedPermissionTail)
	w.reportPicker(true)
	w.repaint(65*time.Second, 2)
	if got := w.tickAt(70 * time.Second); len(got) != 1 || !strings.Contains(got[0].Body, "behind a picker") {
		t.Fatalf("a repainting picker went unreported: %+v", got)
	}
	// The countdown repaints again: the same wait, no second notice (#468).
	w.repaint(80*time.Second, 3)
	if got := w.tickAt(85 * time.Second); len(got) != 1 {
		t.Fatalf("a repaint re-notified the picker: %+v", got)
	}
}

// repaint has the worker's wrapper report output at offset past now and waits
// until the broker lists it.
func (w *waitRig) repaint(offset time.Duration, seq uint64) {
	w.t.Helper()
	repaint := time.Now().Add(offset)
	w.client.Update(couchmessage.Observation{LastActivity: repaint, Sequence: seq})
	deadline := time.Now().Add(3 * time.Second)
	for {
		rows, err := w.s.broker.Actors(context.Background(), w.hub)
		if err != nil {
			w.t.Fatal(err)
		}
		for _, r := range rows {
			if r.Binding == w.worker && !r.LastActivity.Before(repaint) {
				return
			}
		}
		if time.Now().After(deadline) {
			w.t.Fatal("the repaint never reached the broker")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// #673 BR-3: the blocked-prompt set reaches the attention list as prompt
// conditions: the slot appears when its picker opens and leaves when it
// closes, whoever answered it.
func TestBlockedPromptsReachTheAttentionList(t *testing.T) {
	w := newWaitRig(t, false, true, recordedPermissionTail)
	var mu sync.Mutex
	var prompts []map[couchcore.ThreadAddress]couchcore.AttentionCondition
	w.s.SetAttention(func(f couchcore.AttentionFact) {
		if f.Kind != couchcore.AttentionPrompt {
			return
		}
		mu.Lock()
		prompts = append(prompts, f.Live)
		mu.Unlock()
	})
	last := func() map[couchcore.ThreadAddress]couchcore.AttentionCondition {
		mu.Lock()
		defer mu.Unlock()
		if len(prompts) == 0 {
			return nil
		}
		return prompts[len(prompts)-1]
	}
	thread := bindingThread(w.worker)
	w.reportPicker(true)
	if got := last(); len(got) != 1 || got[thread] != (couchcore.AttentionCondition{Label: w.worker.Slot, Title: attentionPromptTitle}) {
		t.Fatalf("prompt set after the picker opened: %+v", got)
	}
	w.reportPicker(false)
	if got := last(); got == nil || len(got) != 0 {
		t.Fatalf("prompt set after the picker closed: %+v", got)
	}
}
