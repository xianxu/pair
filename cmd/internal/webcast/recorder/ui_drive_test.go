package recorder

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/cdp"
	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

type fakeTargetGuard struct {
	mu            sync.Mutex
	actions       []string
	held, unknown bool
}

func (g *fakeTargetGuard) apply(session, action string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if session != "A" {
		return errors.New("wrong target")
	}
	g.actions = append(g.actions, action)
	switch action {
	case "acquire":
		if g.unknown {
			return errors.New("target quarantined")
		}
		g.held = true
	case "release":
		g.held = false
	case "unknown":
		g.held = true
		g.unknown = true
	}
	return nil
}
func (g *fakeTargetGuard) state() (bool, bool, string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.held, g.unknown, strings.Join(g.actions, ",")
}
func installDriveUI(t *testing.T, h *harness, p *fakePage) *fakeTargetGuard {
	t.Helper()
	g := &fakeTargetGuard{}
	if err := h.r.ask(context.Background(), func() { h.confirmUI("A", p.frameID, 7); h.r.opts.UIGuard = g.apply }); err != nil {
		t.Fatal(err)
	}
	h.settle()
	return g
}
func TestUIGuardDefinitiveInputRestores(t *testing.T) {
	for _, tc := range []struct {
		name    string
		failure error
	}{
		{"success", nil}, {"cdp refusal", &cdp.Error{Code: -32000, Message: "refused"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, d, p, btn := drivenHarness(t, nil)
			g := installDriveUI(t, h, p)
			d.mu.Lock()
			d.inputErr = tc.failure
			d.mu.Unlock()
			err := drive.NewTabInjector(h.r).Inject(context.Background(), "v1", clickOn(h, h.id("A", btn.backend)))
			if (err != nil) != (tc.failure != nil) {
				t.Fatalf("input error %v", err)
			}
			held, unknown, actions := g.state()
			if held || unknown || actions != "acquire,release" || h.ui.active("A") != 0 {
				t.Fatalf("guard %v %v %s tokens %d", held, unknown, actions, h.ui.active("A"))
			}
		})
	}
}
func TestUIGuardUncertainInputKeepsTokensAfterLateCompletion(t *testing.T) {
	dialog := make(chan struct{})
	h, _, p, btn := drivenHarness(t, dialog)
	g := installDriveUI(t, h, p)
	issued := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseInput := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(releaseInput)
	completed := make(chan struct{})
	previous := h.fc.more
	h.r.ask(context.Background(), func() {
		h.fc.more = func(session, method string, params map[string]any, p *fakePage) (any, error, bool) {
			if strings.HasPrefix(method, "Input.") {
				close(issued)
				<-release
				defer close(completed)
			}
			return previous(session, method, params, p)
		}
	})
	finished := make(chan error, 1)
	batch := clickOn(h, h.id("A", btn.backend))
	go func() { finished <- drive.NewTabInjector(h.r).Inject(context.Background(), "v1", batch) }()
	select {
	case <-issued:
	case <-time.After(5 * time.Second):
		t.Fatal("input never issued")
	}
	close(dialog)
	select {
	case err := <-finished:
		if !errors.Is(err, drive.ErrDialog) {
			t.Fatalf("error %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("dialog did not release caller")
	}
	held, unknown, actions := g.state()
	if !held || !unknown || actions != "acquire,unknown" || h.ui.active("A") != 1 {
		t.Fatalf("uncertain guard %v %v %s tokens %d", held, unknown, actions, h.ui.active("A"))
	}
	releaseInput()
	select {
	case <-completed:
	case <-time.After(5 * time.Second):
		t.Fatal("late input did not finish")
	}
	h.settle()
	held, unknown, actions = g.state()
	if !held || !unknown || actions != "acquire,unknown" || h.ui.active("A") != 1 {
		t.Fatal("late completion restored uncertain controls")
	}
	// Quarantine prevents another dispatch without consuming the held token.
	if err := drive.NewTabInjector(h.r).Inject(context.Background(), "v1", batch); !errors.Is(err, drive.ErrInput) {
		t.Fatalf("quarantined input: %v", err)
	}
}
func TestUIGuardPreparedEditorDoesNotBlockRemoteInput(t *testing.T) {
	h, _, replies := operatorHarness(t)
	h.fc.mu.Lock()
	p := h.fc.pages["A"]
	h.fc.mu.Unlock()
	node := p.body().kids[0]
	d := &driveTab{hit: node.backend}
	h.r.ask(context.Background(), func() { h.fc.more = d.more })
	g := installDriveUI(t, h, p)
	prepareOperator(t, h, replies)
	ctl(h, rec.CtlRequest, rec.CtlGrant)
	if err := drive.NewTabInjector(h.r).Inject(context.Background(), "v1", clickOn(h, h.id("A", node.backend))); err != nil {
		t.Fatalf("draft blocked input: %v", err)
	}
	h.r.ask(context.Background(), func() {
		if h.r.operator == nil || h.r.operator.draft == nil {
			t.Error("held draft was discarded")
		}
	})
	if held, unknown, actions := g.state(); held || unknown || actions != "acquire,release" {
		t.Fatalf("guard %v %v %s", held, unknown, actions)
	}
	if inputs, _ := d.log(); len(inputs) == 0 {
		t.Fatal("remote click never dispatched")
	}
}
