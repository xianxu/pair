package couchcmd

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/notifyosc"
)

func TestArchiveWarningPrimaryHub(t *testing.T) {
	r, slots := hubMessageRig(t)
	for _, b := range slots {
		r.waitWorkspace(b)
	}
	owner, ok := r.s.archiveWarningOwner()
	if !ok || owner != slots[0] {
		t.Fatalf("owner=%+v ok=%v", owner, ok)
	}
	var facts []couchcore.AttentionFact
	var events []string
	r.s.SetAttention(func(f couchcore.AttentionFact) { facts = append(facts, f); events = append(events, "attention") })
	var heading, desktop, status string
	r.s.SetDesktopNotify(func(_ context.Context, title, body string) error {
		heading, desktop = title, body
		events = append(events, "desktop")
		return nil
	})
	body := "database stale; see /archive/continuous.log"
	got, err := r.s.notifyArchiveWarning(context.Background(), body, func(v string) { status = v; events = append(events, "status") })
	if err != nil || got != body || status != body || desktop != body {
		t.Fatalf("body=%q status=%q desktop=%q err=%v", got, status, desktop, err)
	}
	if len(facts) != 1 {
		t.Fatalf("facts=%+v", facts)
	}
	n := facts[0]
	if n.Kind != couchcore.AttentionNotify || n.Notify.Thread != bindingThread(slots[0]) || n.Notify.Label != "pair:0" || n.Notify.Title != "History backup" || n.Notify.Body != body || !strings.HasPrefix(n.Notify.ID, "notify-") {
		t.Fatalf("fact=%+v", n)
	}
	if heading != "[pair:0] History backup" || strings.Join(events, ",") != "attention,status,desktop" {
		t.Fatalf("heading=%q events=%v", heading, events)
	}
}

func TestArchiveWarningUnavailableOwner(t *testing.T) {
	for _, fault := range []string{"missing", "disconnected"} {
		t.Run(fault, func(t *testing.T) {
			r, slots := hubMessageRig(t)
			for _, b := range slots {
				r.waitWorkspace(b)
			}
			if fault == "missing" {
				r.s.authority.hub = ""
			} else {
				r.disconnect(slots[0])
			}
			var facts []couchcore.AttentionFact
			r.s.SetAttention(func(f couchcore.AttentionFact) { facts = append(facts, f) })
			var desktop, status string
			r.s.SetDesktopNotify(func(_ context.Context, _, body string) error { desktop = body; return nil })
			got, err := r.s.notifyArchiveWarning(context.Background(), "stale; /archive/continuous.log", func(v string) { status = v })
			if err != nil || len(facts) != 1 || facts[0].Notify.Thread != (couchcore.ThreadAddress{}) || facts[0].Notify.Body != got || got != desktop || got != status || !strings.Contains(got, "primary hub unavailable; full report in continuous.log") {
				t.Fatalf("got=%q desktop=%q status=%q facts=%+v err=%v", got, desktop, status, facts, err)
			}
		})
	}
}

func TestArchiveWarningDesktopFailures(t *testing.T) {
	for _, fault := range []string{"uncertain", "no-console"} {
		t.Run(fault, func(t *testing.T) {
			r, slots := hubMessageRig(t)
			for _, b := range slots {
				r.waitWorkspace(b)
			}
			facts, writes, notices := 0, 0, 0
			r.s.SetAttention(func(couchcore.AttentionFact) { facts++ })
			uncertain := errors.New("uncertain write")
			if fault == "uncertain" {
				r.s.SetDesktopNotify(func(context.Context, string, string) error { writes++; return uncertain })
			}
			got, err := r.s.notifyArchiveWarning(context.Background(), "stale; continuous.log", func(string) { notices++ })
			if err == nil || got != "stale; continuous.log" || facts != 1 || notices != 1 {
				t.Fatalf("body=%q facts=%d notices=%d err=%v", got, facts, notices, err)
			}
			if fault == "uncertain" && (writes != 1 || !errors.Is(err, uncertain)) {
				t.Fatalf("writes=%d err=%v", writes, err)
			}
		})
	}
}

func TestArchiveWarningCancelledHasNoEffects(t *testing.T) {
	r, _ := hubMessageRig(t)
	effects := 0
	r.s.SetAttention(func(couchcore.AttentionFact) { effects++ })
	r.s.SetDesktopNotify(func(context.Context, string, string) error { effects++; return nil })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := r.s.notifyArchiveWarning(ctx, "stale", func(string) { effects++ })
	if !errors.Is(err, context.Canceled) || effects != 0 {
		t.Fatalf("effects=%d err=%v", effects, err)
	}
}

func TestArchiveWarningSingleHubAndBoundedFallback(t *testing.T) {
	world := newMessageWorld()
	world.hub = "/repo0"
	r := &serviceRig{t: t, world: world, slotGit: map[string]couchcore.SlotGitStatus{}}
	r.init()
	hub := r.connect(0)
	r.waitWorkspace(hub)
	var title, body string
	r.s.SetDesktopNotify(func(_ context.Context, h, b string) error { title, body = h, b; return nil })
	got, err := r.s.notifyArchiveWarning(context.Background(), "stale\x1b\a; continuous.log", nil)
	if err != nil || title != "History backup" || got != "stale; continuous.log" || got != body {
		t.Fatalf("title=%q body=%q got=%q err=%v", title, body, got, err)
	}
	got, err = r.s.notifyArchiveWarning(context.Background(), strings.Repeat("界", 5000), nil)
	if err != nil || len(got) > notifyosc.MaxMessageBytes || !utf8.ValidString(got) || got != body {
		t.Fatalf("connected bytes=%d valid=%v err=%v", len(got), utf8.ValidString(got), err)
	}
	r.disconnect(hub)
	got, err = r.s.notifyArchiveWarning(context.Background(), strings.Repeat("界", 5000), nil)
	if err != nil || len(got) > notifyosc.MaxMessageBytes || !utf8.ValidString(got) || !strings.Contains(got, "primary hub unavailable; full report in continuous.log") {
		t.Fatalf("bytes=%d valid=%v err=%v", len(got), utf8.ValidString(got), err)
	}
}

func TestArchiveWarningKeepsExactThreadAfterAttention(t *testing.T) {
	r, slots := hubMessageRig(t)
	for _, b := range slots {
		r.waitWorkspace(b)
	}
	var item couchcore.AttentionItem
	r.s.SetAttention(func(f couchcore.AttentionFact) { item = f.Notify; r.disconnect(slots[0]) })
	writes := 0
	r.s.SetDesktopNotify(func(context.Context, string, string) error { writes++; return nil })
	_, err := r.s.notifyArchiveWarning(context.Background(), "stale", nil)
	if err != nil || item.Thread != bindingThread(slots[0]) || writes != 1 {
		t.Fatalf("item=%+v writes=%d err=%v", item, writes, err)
	}
	if _, ok := r.s.archiveWarningOwner(); ok {
		t.Fatal("disconnected primary was replaced by a secondary")
	}
}

func TestArchiveWarningOwnerUsesLatestBinding(t *testing.T) {
	r, slots := hubMessageRig(t)
	for _, b := range slots {
		r.waitWorkspace(b)
	}
	old := slots[0]
	// Model an admitted replacement's published liveness and connected snapshot.
	latest := old
	latest.Session += "-replacement"
	latest.Tag += "-replacement"
	latest.Nonce += "-replacement"
	live := *r.s.liveness.Load()
	nextLive := make(map[string]couchmessage.SlotLiveness, len(live))
	for k, v := range live {
		nextLive[k] = v
	}
	v := nextLive[old.Slot]
	v.Binding = latest
	nextLive[old.Slot] = v
	r.s.liveness.Store(&nextLive)
	connected := map[couchmessage.Binding]bool{latest: true}
	r.s.connected.Store(&connected)
	owner, ok := r.s.archiveWarningOwner()
	if !ok || owner != latest {
		t.Fatalf("owner=%+v ok=%v", owner, ok)
	}
	var item couchcore.AttentionItem
	r.s.SetAttention(func(f couchcore.AttentionFact) { item = f.Notify })
	r.s.SetDesktopNotify(func(context.Context, string, string) error { return nil })
	if _, err := r.s.notifyArchiveWarning(context.Background(), "stale", nil); err != nil || item.Thread != bindingThread(latest) {
		t.Fatalf("item=%+v err=%v", item, err)
	}
}
