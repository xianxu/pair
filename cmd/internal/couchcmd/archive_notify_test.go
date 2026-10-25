package couchcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/notifyosc"
)

func TestArchiveWarningTypedPrimaryOwnership(t *testing.T) {
	for _, state := range []string{"enabled", "disabled", "missing", "disconnected"} {
		t.Run(state, func(t *testing.T) {
			r, slots := hubMessageRig(t, func(a messageAuthority) messageAuthority {
				if state == "missing" {
					a.hub = ""
				}
				return a
			})
			for _, binding := range slots {
				r.waitWorkspace(binding)
			}
			if state == "disabled" {
				r.s.automation = newHubAutomationOwner(map[string]bool{"/repo0": false, "/repo1": true}, couchcore.StartupUnknown, nil)
			}
			if state == "disconnected" {
				r.disconnect(slots[0])
			}
			var notices []couchtty.Notice
			output := func(n couchtty.Notice) { notices = append(notices, n) }
			r.s.noticeOutput.Store(&output)
			effects := 0
			r.s.SetAttention(func(couchcore.AttentionFact) { effects++ })
			r.s.SetDesktopNotify(func(context.Context, string, string) error { effects++; return errors.New("desktop unavailable") })
			got, err := r.s.notifyArchiveWarning(context.Background(), "database stale; continuous.log", func(string) { effects++ })
			if err != nil || effects != 0 || len(notices) != 1 {
				t.Fatalf("notices=%+v direct effects=%d err=%v", notices, effects, err)
			}
			n := notices[0]
			if n.Class != couchcore.NoticeArchiveInventoryWarning || n.Body != got || n.Thread != (couchcore.ThreadAddress{}) || n.OwnerRoot != "" || n.HubHandled {
				t.Fatalf("global archive notice=%+v", n)
			}
			owner := r.s.NoticeOwner(couchcore.NoticeDeliveryOccurrence{})
			want := couchcore.NoticeOwnerEnabled
			if state == "disabled" {
				want = couchcore.NoticeOwnerDisabled
			}
			if state == "missing" || state == "disconnected" {
				want = couchcore.NoticeOwnerUnavailable
			}
			if owner.State != want {
				t.Fatalf("owner=%+v want state=%d", owner, want)
			}
			if want != couchcore.NoticeOwnerUnavailable && owner.Binding != lifecycleBindingKey(slots[0]) {
				t.Fatalf("global warning selected alternate hub: %+v", owner)
			}
		})
	}
}

func TestArchiveWarningLegacyCallbackWithoutTypedOutput(t *testing.T) {
	r, slots := hubMessageRig(t)
	for _, b := range slots {
		r.waitWorkspace(b)
	}
	calls := 0
	got, err := r.s.notifyArchiveWarning(context.Background(), "stale", func(body string) {
		calls++
		if body != "stale" {
			t.Errorf("body=%q", body)
		}
	})
	if err != nil || got != "stale" || calls != 1 {
		t.Fatalf("body=%q calls=%d err=%v", got, calls, err)
	}
}

func TestArchiveWarningCancelledHasNoEffects(t *testing.T) {
	r, _ := hubMessageRig(t)
	effects := 0
	output := func(couchtty.Notice) { effects++ }
	r.s.noticeOutput.Store(&output)
	r.s.SetAttention(func(couchcore.AttentionFact) { effects++ })
	r.s.SetDesktopNotify(func(context.Context, string, string) error { effects++; return nil })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := r.s.notifyArchiveWarning(ctx, "stale", func(string) { effects++ })
	if !errors.Is(err, context.Canceled) || effects != 0 {
		t.Fatalf("effects=%d err=%v", effects, err)
	}
}

func TestArchiveWarningSanitizedBoundedFallback(t *testing.T) {
	r, slots := hubMessageRig(t)
	for _, b := range slots {
		r.waitWorkspace(b)
	}
	var notice couchtty.Notice
	output := func(n couchtty.Notice) { notice = n }
	r.s.noticeOutput.Store(&output)
	got, err := r.s.notifyArchiveWarning(context.Background(), "stale\x1b\a; continuous.log", nil)
	if err != nil || got != "stale; continuous.log" || notice.Body != got {
		t.Fatalf("body=%q notice=%+v err=%v", got, notice, err)
	}
	for _, disconnected := range []bool{false, true} {
		if disconnected {
			r.disconnect(slots[0])
		}
		got, err = r.s.notifyArchiveWarning(context.Background(), strings.Repeat("界", 5000), nil)
		if err != nil || len(got) > notifyosc.MaxMessageBytes || !utf8.ValidString(got) || notice.Body != got {
			t.Fatalf("bytes=%d valid=%v err=%v", len(got), utf8.ValidString(got), err)
		}
		if disconnected && !strings.Contains(got, "primary hub unavailable; full report in continuous.log") {
			t.Fatalf("fallback lost retained report: %q", got)
		}
	}
}

func TestArchiveWarningOwnerUsesLatestBinding(t *testing.T) {
	r, slots := hubMessageRig(t)
	for _, b := range slots {
		r.waitWorkspace(b)
	}
	old := slots[0]
	latest := old
	latest.Session += "-replacement"
	latest.Tag += "-replacement"
	latest.Nonce += "-replacement"
	live := *r.s.liveness.Load()
	next := make(map[string]couchmessage.SlotLiveness, len(live))
	for k, v := range live {
		next[k] = v
	}
	v := next[old.Slot]
	v.Binding = latest
	next[old.Slot] = v
	r.s.liveness.Store(&next)
	connected := map[couchmessage.Binding]bool{latest: true}
	r.s.connected.Store(&connected)
	owner, ok := r.s.archiveWarningOwner()
	if !ok || owner != latest {
		t.Fatalf("owner=%+v ok=%v", owner, ok)
	}
}

func TestArchiveWarningDailyGatePublishesTypedFeedOnce(t *testing.T) {
	r, slots := hubMessageRig(t)
	for _, b := range slots {
		r.waitWorkspace(b)
	}
	var notices []couchtty.Notice
	output := func(n couchtty.Notice) { notices = append(notices, n) }
	r.s.noticeOutput.Store(&output)
	legacy := 0
	scheduler := newArchiveScheduler(t.TempDir(), func(string) { legacy++ })
	if err := os.MkdirAll(scheduler.store, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scheduler.store, "continuous.log"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	scheduler.warn = func(ctx context.Context, body string) (string, error) {
		return r.s.notifyArchiveWarning(ctx, body, scheduler.notify)
	}
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	key := archiveWarningKey{Destination: "remote-backup"}
	scheduler.warnOnce(context.Background(), now, key, "remote backup stale")
	scheduler.warnOnce(context.Background(), now, key, "remote backup stale")
	if len(notices) != 1 || legacy != 0 {
		t.Fatalf("one daily occurrence published typed=%+v legacy=%d", notices, legacy)
	}
	scheduler.warnOnce(context.Background(), now.Add(24*time.Hour), key, "remote backup stale")
	if len(notices) != 2 || legacy != 0 {
		t.Fatalf("next day typed=%+v legacy=%d", notices, legacy)
	}
}
