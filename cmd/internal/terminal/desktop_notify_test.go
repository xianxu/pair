package terminal

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/ttyio"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPresenterDesktopNotifySerializedAndNoReplay(t *testing.T) {
	p, host, e, _ := presenterFixture(t, AnyMotion)
	block := make(chan struct{})
	var release sync.Once
	defer release.Do(func() { close(block) })
	host.Enqueue(ttyio.WriteStep{Block: block})
	first := make(chan error, 1)
	go func() { first <- p.Copy(context.Background(), []byte("clip")) }()
	<-host.Started()
	second := make(chan error, 1)
	go func() { second <- p.Notify(context.Background(), "Couch; café", "ready; 世界") }()
	select {
	case err := <-second:
		t.Fatalf("notification overtook copy: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	// Opening the barrier lets the presenter's single worker commit both controls.
	hostBytes := "\x1b]52;c;Y2xpcA==\x1b\\\x1b]777;notify;Couch, café;ready; 世界\x1b\\"
	release.Do(func() { close(block) })
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	if err := <-second; err != nil {
		t.Fatal(err)
	}
	if string(host.Bytes()) != hostBytes {
		t.Fatalf("wire %q", host.Bytes())
	}
	selectPresenter(t, p, e)
	if err := p.Present(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(host.Bytes()), "777;notify;") != 1 {
		t.Fatal("notification replayed")
	}
}

func TestPresenterDesktopNotifyCancellationAndFailures(t *testing.T) {
	t.Run("cancelled", func(t *testing.T) {
		p, host, _, _ := presenterFixture(t, AnyMotion)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := p.Notify(ctx, "Couch", "ready"); !errors.Is(err, context.Canceled) {
			t.Fatalf("error %v", err)
		}
		if len(host.Bytes()) != 0 {
			t.Fatal("cancelled notification wrote")
		}
	})
	for _, limit := range []int{0, 5} {
		t.Run(string(rune('0'+limit)), func(t *testing.T) {
			p, host, _, _ := presenterFixture(t, AnyMotion)
			boom := errors.New("host failed")
			host.Enqueue(ttyio.WriteStep{Limit: limit, ZeroProgress: limit == 0, Err: boom})
			if err := p.Notify(context.Background(), "Couch", "ready"); !errors.Is(err, boom) {
				t.Fatalf("error %v", err)
			}
			if len(host.Bytes()) != limit {
				t.Fatalf("accepted %d", len(host.Bytes()))
			}
			before := host.Calls()
			if err := p.Notify(context.Background(), "Couch", "again"); err == nil {
				t.Fatal("failed presenter accepted notify")
			}
			if host.Calls() != before {
				t.Fatal("retried failed notification")
			}
		})
	}
}

func TestPresenterSuppressedNotificationsAlwaysAcknowledged(t *testing.T) {
	for _, allow := range []bool{false, true} {
		p, host, e, _ := presenterFixture(t, AnyMotion)
		if err := p.Register(context.Background(), e); err != nil {
			t.Fatal(err)
		}
		out, err := e.Feed([]byte("\x1b]99;;secret\a"), time.Now())
		if err != nil {
			t.Fatal(err)
		}
		policy := EffectPolicy{Bell: true, Title: true, Clipboard: true, Notifications: allow}
		got, err := p.EmitEffects(context.Background(), out.Effects, policy)
		if err != nil || len(got) != 1 {
			t.Fatalf("ack %+v %v", got, err)
		}
		got, err = p.EmitEffects(context.Background(), out.Effects, policy)
		if err != nil || len(got) != 0 || len(host.Bytes()) != 0 {
			t.Fatalf("replay or forwarded %+v %v %q", got, err, host.Bytes())
		}
	}
}

func TestPresenterDesktopPartialWritesCompleteOneEnvelope(t *testing.T) {
	p, host, _, _ := presenterFixture(t, AnyMotion)
	host.Enqueue(ttyio.WriteStep{Limit: 3})
	host.Enqueue(ttyio.WriteStep{Limit: 2})
	if err := p.Notify(context.Background(), "Couch", "ready"); err != nil {
		t.Fatal(err)
	}
	if got := string(host.Bytes()); got != "\x1b]777;notify;Couch;ready\x1b\\" {
		t.Fatalf("wire %q", got)
	}
}

func TestPresenterDesktopRejectsInvalidWithoutHostWrite(t *testing.T) {
	p, host, _, _ := presenterFixture(t, AnyMotion)
	for _, tc := range [][2]string{{"", "ready"}, {"Couch", "\a"}, {strings.Repeat("x", 257), "ready"}, {"Couch", strings.Repeat("x", 4097)}} {
		if err := p.Notify(context.Background(), tc[0], tc[1]); err == nil {
			t.Fatal("accepted invalid notification")
		}
	}
	if host.Calls() != 0 {
		t.Fatal("invalid notification touched host")
	}
}

func TestPresenterDesktopCancelBlockedHostWrite(t *testing.T) {
	p, host, _, _ := presenterFixture(t, AnyMotion)
	host.Enqueue(ttyio.WriteStep{Block: make(chan struct{})})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- p.Notify(ctx, "Couch", "ready") }()
	<-host.Started()
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("error %v", err)
	}
	if len(host.Bytes()) != 0 {
		t.Fatal("cancelled blocked host accepted bytes")
	}
}

// Both ways into a stall must report non-success without queuing a replay.
func TestPresenterDesktopNotifyStallsReportFailureWithoutReplay(t *testing.T) {
	for _, alreadyStalled := range []bool{true, false} {
		name := "notification stalls"
		if alreadyStalled {
			name = "already stalled"
		}
		t.Run(name, func(t *testing.T) {
			p, host, e, _ := presenterFixture(t, AnyMotion)
			selectPresenter(t, p, e)
			gate := make(chan struct{})
			var release sync.Once
			defer release.Do(func() { close(gate) })
			for i := 0; i < 200; i++ {
				host.Enqueue(ttyio.WriteStep{Block: gate})
			}
			if alreadyStalled {
				// Existing best-effort controls still initiate recovery without failing Couch.
				if err := p.Copy(context.Background(), []byte("dropped")); err != nil {
					t.Fatal(err)
				}
			}
			before := len(host.Bytes())
			err := p.Notify(context.Background(), "Couch", "must-not-replay")
			if err == nil {
				t.Error("dropped notification reported success")
			}
			if !alreadyStalled && !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("notification write deadline lost: %v", err)
			}
			if p.Stalls() != 1 || presenterFailed(p) {
				t.Fatal("stall did not preserve recoverable presenter")
			}
			if len(host.Bytes()) != before {
				t.Fatal("blocked host accepted notification bytes")
			}
			e.Feed([]byte("\x1b[Hback"), time.Now())
			if err := p.Present(context.Background(), e); err != nil {
				t.Fatal(err)
			}
			release.Do(func() { close(gate) })
			waitFor(t, "repaint after notify stall", func() bool { return strings.Contains(string(host.Bytes()[before:]), "back") })
			if err := p.Notify(context.Background(), "Couch", "fresh"); err != nil {
				t.Fatal(err)
			}
			wire := string(host.Bytes()[before:])
			if strings.Contains(wire, "must-not-replay") || strings.Count(wire, "777;notify;") != 1 {
				t.Fatalf("notification replayed: %q", wire)
			}
		})
	}
}
