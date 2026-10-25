package couchcmd

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// The stateful endpoint owns delivery completion, while registry admission and
// cached source/owner identity continue through the real service and sockets.
type noticeStartupEndpoint struct {
	hubQuietEndpoint
	mu       sync.Mutex
	messages []couchmessage.Message
}

func (e *noticeStartupEndpoint) Deliver(ctx context.Context, m couchmessage.Message, _ func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	e.mu.Lock()
	e.messages = append(e.messages, m)
	e.mu.Unlock()
	return couchmessage.Receipt{Status: couchmessage.Submitted}, ctx.Err()
}
func (e *noticeStartupEndpoint) snapshot() []couchmessage.Message {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]couchmessage.Message(nil), e.messages...)
}

func noticeStartupSurvivor() *couchcore.StartupObservation {
	return &couchcore.StartupObservation{Kind: couchcore.StartupNormal, Hubs: map[string]couchcore.StartupHubObservation{"/repo0": {Survived: true}}}
}

func noticeStartupOccurrence(t *testing.T, d *noticeDelivery) couchcore.NoticeDeliveryOccurrence {
	t.Helper()
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.state.Entries) != 1 || len(d.admissions) != 0 || len(d.projections) != 0 {
		t.Fatalf("startup obligation was lost or routed before readiness: entries=%+v admissions=%+v projections=%+v", d.state.Entries, d.admissions, d.projections)
	}
	return d.state.Entries[0].Occurrence
}

func noticeStartupNoBell(t *testing.T, console *couchtty.Console, screen func() string, marker string) {
	t.Helper()
	console.Notify(couchtty.Notice{Kind: "startup-test-barrier", Body: marker})
	// Notify repaints synchronously once Run has started; ordinary actor chrome
	// retains feed text without drawing it, so wait on the actual slot chrome.
	testwait.Until(t, "startup Console initial chrome", func() bool { return strings.Contains(screen(), "unrelated focused slot") })
	if got := screen(); strings.Contains(got, couchtty.AttentionGlyph) {
		t.Fatalf("startup raised a human attention badge:\n%s", got)
	}
}

func TestNoticeStartupGlobalTraceBuffersUntilPrimaryConnects(t *testing.T) {
	r, slots := hubMessageRig(t)
	for _, b := range slots {
		r.waitWorkspace(b)
	}
	primary := slots[0]
	r.disconnect(primary)
	endpoint := &noticeStartupEndpoint{hubQuietEndpoint: hubQuietEndpoint{at: time.Now().Add(-time.Hour)}}
	r.world.mu.Lock()
	r.world.endpoints[primary] = endpoint
	r.world.mu.Unlock()
	console, host, _ := warningConsole(t)
	d := newNoticeDelivery(console)
	console.SetNoticeDelivery(d)
	t.Cleanup(console.CloseNoticeDelivery)
	screen := startupWarmScreen(t, 120, 30, host.Written)
	// This is the actual early trace producer; no service has bound the adapter.
	if err := console.SetEventTrace(filepath.Join(t.TempDir(), "missing", "startup.trace"), time.Now()); err == nil {
		t.Fatal("trace unexpectedly opened")
	}
	occurrence := noticeStartupOccurrence(t, d)
	if occurrence.Class != couchcore.NoticeEventTraceFailed || occurrence.Thread != (couchcore.ThreadAddress{}) || occurrence.OwnerRoot != "" {
		t.Fatalf("trace inferred focused-slot ownership: %+v", occurrence)
	}
	noticeStartupNoBell(t, console, screen, "before-bind")
	r.s.installNoticeDelivery(console, noticeStartupSurvivor())
	before := r.world.counts()
	for i := 0; i < 3; i++ {
		d.reconcile()
		if owner := r.s.NoticeOwner(occurrence); owner.State != couchcore.NoticeOwnerBooting {
			t.Fatalf("surviving primary is not booting: %+v", owner)
		}
	}
	noticeStartupOccurrence(t, d)
	if got := r.world.counts(); got != before {
		t.Fatalf("pending adapter performed authority probes: %v -> %v", before, got)
	}
	if _, known, _ := r.s.broker.ReceiptSnapshot(occurrence.ID); known {
		t.Fatal("secondary hub took a booting primary's global warning")
	}
	noticeStartupNoBell(t, console, screen, "awaiting-primary")
	r.wrapper(primary)
	r.waitConnected(primary, true)
	r.waitWorkspace(primary)
	testwait.Until(t, "one canonical startup receipt", func() bool { _, known, _ := r.s.broker.ReceiptSnapshot(occurrence.ID); return known })
	testwait.Until(t, "one completed real startup delivery", func() bool {
		receipt, known, delivering := r.s.broker.ReceiptSnapshot(occurrence.ID)
		return known && !delivering && receipt.Status == couchmessage.Submitted && len(endpoint.snapshot()) == 1
	})
	testwait.Until(t, "startup routing obligation retired", func() bool { return noticeDeliveryRetained(d) == 0 })
	receipt, known, _ := r.s.broker.ReceiptSnapshot(occurrence.ID)
	if !known || receipt.Message.To != primary || receipt.Message.From != primary || receipt.Message.Body != occurrence.Body {
		t.Fatalf("wrong global recipient or canonical occurrence: %+v", receipt)
	}
	afterConnect := r.world.counts()
	for i := 0; i < 3; i++ {
		d.reconcile()
		if owner := r.s.NoticeOwner(occurrence); owner.State != couchcore.NoticeOwnerEnabled || owner.Binding != lifecycleBindingKey(primary) {
			t.Fatalf("connected owner=%+v", owner)
		}
	}
	d.Close()
	if got := endpoint.snapshot(); len(got) != 1 || got[0].ID != occurrence.ID {
		t.Fatalf("startup warning resent: %+v", got)
	}
	if got := r.world.counts(); got != afterConnect {
		t.Fatalf("settled adapter performed discovery: %v -> %v", afterConnect, got)
	}
	noticeStartupNoBell(t, console, screen, "primary-delivered")
}

func TestNoticeStartupSurvivingHubOffRetiresWithoutReplay(t *testing.T) {
	r, slots := hubMessageRig(t)
	for _, b := range slots {
		r.waitWorkspace(b)
	}
	primary := slots[0]
	r.disconnect(primary)
	if _, err := r.s.controlAutomation("/repo0", "off"); err != nil {
		t.Fatal(err)
	}
	console, host, _ := warningConsole(t)
	d := newNoticeDelivery(console)
	console.SetNoticeDelivery(d)
	t.Cleanup(console.CloseNoticeDelivery)
	console.Warn(couchtty.Notice{Class: couchcore.NoticeStartupConfigurationWarning, Kind: "status", Body: "startup configuration could not be retained"})
	occurrence := noticeStartupOccurrence(t, d)
	r.s.installNoticeDelivery(console, noticeStartupSurvivor())
	if owner := r.s.NoticeOwner(occurrence); owner.State != couchcore.NoticeOwnerBooting {
		t.Fatalf("off policy erased actual startup witness: %+v", owner)
	}
	r.wrapper(primary)
	r.waitConnected(primary, true)
	r.waitWorkspace(primary)
	if owner := r.s.NoticeOwner(occurrence); owner.State != couchcore.NoticeOwnerDisabled {
		t.Fatalf("connected off hub did not retire witness: %+v", owner)
	}
	testwait.Until(t, "disabled startup occurrence retired", func() bool { d.reconcile(); return noticeDeliveryRetained(d) == 0 })
	if _, err := r.s.controlAutomation("/repo0", "on"); err != nil {
		t.Fatal(err)
	}
	d.reconcile()
	d.Close()
	if _, known, _ := r.s.broker.ReceiptSnapshot(occurrence.ID); known {
		t.Fatal("enabling hub replayed disabled startup warning")
	}
	screen := startupWarmScreen(t, 120, 30, host.Written)
	noticeStartupNoBell(t, console, screen, "disabled-startup-retired")
	// Once actual connection has ended the witness, a later disconnect is truly
	// unavailable; a stale Survived bit cannot keep future warnings booting.
	r.disconnect(primary)
	if owner := r.s.NoticeOwner(occurrence); owner.State != couchcore.NoticeOwnerUnavailable {
		t.Fatalf("startup witness survived actual connection: %+v", owner)
	}
}

func TestNoticeStartupOrdinaryMultiSlotFeedbackHasNoBellOrProbes(t *testing.T) {
	r, slots := hubMessageRig(t)
	for _, b := range slots {
		r.waitWorkspace(b)
	}
	console, host, _ := warningConsole(t)
	d := newNoticeDelivery(console)
	console.SetNoticeDelivery(d)
	t.Cleanup(console.CloseNoticeDelivery)
	r.s.installNoticeDelivery(console, &couchcore.StartupObservation{Kind: couchcore.StartupNormal})
	before := r.world.counts()
	for _, slot := range slots {
		console.Notify(couchtty.Notice{Kind: "startup-feedback-" + slot.Slot, Thread: bindingThread(slot), Body: "Attached " + slot.Slot})
		console.Warn(couchtty.Notice{Class: couchcore.NoticeRoutine, Kind: "startup-routine-" + slot.Slot, Thread: bindingThread(slot), Body: "Already attached " + slot.Slot})
	}
	d.reconcile()
	d.Close()
	d.mu.Lock()
	sequence, retained := d.sequence, len(d.state.Entries)
	d.mu.Unlock()
	if sequence != 0 || retained != 0 {
		t.Fatalf("ordinary startup created obligations: sequence=%d retained=%d", sequence, retained)
	}
	if got := r.world.counts(); got != before {
		t.Fatalf("ordinary startup probed authority: %v -> %v", before, got)
	}
	screen := startupWarmScreen(t, 120, 30, host.Written)
	noticeStartupNoBell(t, console, screen, "ordinary-startup-ready")
}
