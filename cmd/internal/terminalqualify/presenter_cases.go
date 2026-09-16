package terminalqualify

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/ttyio"
)

func PresenterCases() []Case {
	const source = "workshop/plans/000255-terminal-abstraction-plan.md#chunk-2-m2-repaired-core-endpoint-and-presenter"
	cases := []Case{
		{ID: "hidden-origin", Capability: "selected presenter with hidden endpoint query/effect origin", Expected: Observation{"selected": "a", "a-input": "x", "b-reply": "\x1b[3;4R", "hidden-draw": "false", "title-count": "1", "repeat-effects": "false"}, Integration: observePresenterHidden},
		{ID: "partial-parent-write", Capability: "partial parent failure closes input admission", Expected: Observation{"failed": "true", "admitted": "", "accepted": "5", "child-input": "", "input-refused": "true", "refresh-refused": "true", "extra-writes": "0"}, Integration: observePresenterFailure},
		{ID: "drag-destination", Capability: "press-owned destination and old gesture cancellation", Expected: Observation{"a-input": "\x1b[<0;3;2M\x1b[<32;3;2M\x1b[<0;3;2m", "b-input": "\x1b[<0;2;2M\x1b[<0;2;2m", "chrome-input": "false"}, Integration: observePresenterDrag},
		{ID: "presenter-admission", Capability: "input waits for successful parent presentation", Expected: Observation{"pending-admitted": "", "pending-input": "", "paused": "true", "admitted": "a", "delivered": "x"}, Integration: observePresenterAdmission},
		{ID: "presenter-release", Capability: "release cancels blocked paint and rejects late writes", Expected: Observation{"released": "true", "select-cancelled": "true", "late-refused": "true", "extra-writes": "0"}, Integration: observePresenterRelease},
	}
	for i := range cases {
		cases[i].Target = "presenter"
		cases[i].Source = source
	}
	return cases
}

func selectQualification(ctx context.Context, p *terminal.Presenter, e *terminal.Endpoint) error {
	return p.Select(ctx, e, terminal.Geometry{Cols: 8, Rows: 5}, make([]terminal.Cell, 8))
}

func observePresenterHidden(ctx context.Context) (Observation, error) {
	parent := ttyio.NewFake()
	p := terminal.NewPresenter(parent, terminal.CouchAnyMotion)
	defer p.Release(ctx)
	a, aw, err := scenarioEndpoint("a")
	if err != nil {
		return nil, err
	}
	defer a.Close()
	b, bw, err := scenarioEndpoint("b")
	if err != nil {
		return nil, err
	}
	defer b.Close()
	if err = p.Register(ctx, b); err != nil {
		return nil, err
	}
	if err = selectQualification(ctx, p, a); err != nil {
		return nil, err
	}
	before := string(parent.Bytes())
	effects, err := b.Feed([]byte("B\x1b[3;4H\x1b[6n\x1b]2;hidden-title\a"), time.Now())
	if err != nil {
		return nil, err
	}
	if err = b.Flush(ctx); err != nil {
		return nil, err
	}
	// Either an explicit stale-source error or a coalesced no-op is permissible;
	// neither may change the selected view or emit hidden drawing commands.
	_ = p.Present(ctx, b)
	// Present is coalesced. The ordered barrier drains any requested refresh
	// before inspecting the wire, so a scheduled hidden write cannot evade us.
	if err = p.Flush(ctx); err != nil {
		return nil, err
	}
	observed := Observation{"hidden-draw": strconv.FormatBool(string(parent.Bytes()) != before)}
	if err = p.Input(ctx, uv.KeyPressEvent{Code: 'x', Text: "x"}); err != nil {
		return nil, err
	}
	if err = a.Flush(ctx); err != nil {
		return nil, err
	}
	observed["selected"] = p.View().Selected
	observed["a-input"] = string(aw.Bytes())
	observed["b-reply"] = string(bw.Bytes())
	policy := terminal.EffectPolicy{Title: true}
	if _, err = p.EmitEffects(ctx, effects.Effects, policy); err != nil {
		return nil, err
	}
	once := string(parent.Bytes())
	observed["title-count"] = strconv.Itoa(strings.Count(once, "\x1b]2;hidden-title\x1b\\"))
	if _, err = p.EmitEffects(ctx, effects.Effects, policy); err != nil {
		return nil, err
	}
	observed["repeat-effects"] = strconv.FormatBool(string(parent.Bytes()) != once)
	return observed, nil
}

func observePresenterFailure(ctx context.Context) (Observation, error) {
	parent := ttyio.NewFake()
	p := terminal.NewPresenter(parent, terminal.ChildRequested)
	defer p.Release(ctx)
	e, input, err := scenarioEndpoint("a")
	if err != nil {
		return nil, err
	}
	defer e.Close()
	boom := errors.New("synthetic broken parent")
	parent.Enqueue(ttyio.WriteStep{Limit: 5, Err: boom})
	failure := selectQualification(ctx, p, e)
	observed := Observation{"failed": strconv.FormatBool(errors.Is(failure, boom) && p.View().State == terminal.Failed), "admitted": p.View().Admitted, "accepted": strconv.Itoa(len(parent.Bytes()))}
	observed["input-refused"] = strconv.FormatBool(p.Input(ctx, uv.KeyPressEvent{Code: 'x', Text: "x"}) != nil)
	calls := parent.Calls()
	observed["refresh-refused"] = strconv.FormatBool(p.Present(ctx, e) != nil)
	observed["extra-writes"] = strconv.Itoa(parent.Calls() - calls)
	if err = e.Flush(ctx); err != nil {
		return nil, err
	}
	observed["child-input"] = string(input.Bytes())
	return observed, nil
}

func observePresenterDrag(ctx context.Context) (Observation, error) {
	parent := ttyio.NewFake()
	p := terminal.NewPresenter(parent, terminal.CouchAnyMotion)
	defer p.Release(ctx)
	a, aw, err := scenarioEndpoint("a")
	if err != nil {
		return nil, err
	}
	defer a.Close()
	b, bw, err := scenarioEndpoint("b")
	if err != nil {
		return nil, err
	}
	defer b.Close()
	for _, e := range []*terminal.Endpoint{a, b} {
		if _, err = e.Feed([]byte("\x1b[?1002;1006h"), time.Now()); err != nil {
			return nil, err
		}
	}
	if err = selectQualification(ctx, p, a); err != nil {
		return nil, err
	}
	for _, event := range []uv.Event{uv.MouseClickEvent{X: 2, Y: 1, Button: uv.MouseLeft}, uv.MouseMotionEvent{X: 2, Y: 1, Button: uv.MouseLeft}} {
		if err = p.Input(ctx, event); err != nil {
			return nil, err
		}
	}
	if err = selectQualification(ctx, p, b); err != nil {
		return nil, err
	}
	for _, event := range []uv.Event{uv.MouseMotionEvent{X: 3, Y: 2, Button: uv.MouseLeft}, uv.MouseReleaseEvent{X: 3, Y: 2, Button: uv.MouseLeft}, uv.MouseClickEvent{X: 1, Y: 1, Button: uv.MouseLeft}, uv.MouseReleaseEvent{X: 1, Y: 1, Button: uv.MouseLeft}} {
		if err = p.Input(ctx, event); err != nil {
			return nil, err
		}
	}
	if err = a.Flush(ctx); err != nil {
		return nil, err
	}
	if err = b.Flush(ctx); err != nil {
		return nil, err
	}
	observed := Observation{"a-input": string(aw.Bytes()), "b-input": string(bw.Bytes())}
	before := string(bw.Bytes())
	if err = p.Input(ctx, uv.MouseClickEvent{X: 2, Y: 4, Button: uv.MouseLeft}); err != nil {
		return nil, err
	}
	if err = b.Flush(ctx); err != nil {
		return nil, err
	}
	observed["chrome-input"] = strconv.FormatBool(string(bw.Bytes()) != before)
	return observed, nil
}

func observePresenterAdmission(ctx context.Context) (Observation, error) {
	parent := ttyio.NewFake()
	p := terminal.NewPresenter(parent, terminal.CouchAnyMotion)
	defer p.Release(ctx)
	e, input, err := scenarioEndpoint("a")
	if err != nil {
		return nil, err
	}
	defer e.Close()
	block := make(chan struct{})
	parent.Enqueue(ttyio.WriteStep{Block: block})
	selected := make(chan error, 1)
	go func() { selected <- selectQualification(ctx, p, e) }()
	select {
	case <-parent.Started():
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	observed := Observation{"pending-admitted": p.View().Admitted, "pending-input": string(input.Bytes())}
	sent := make(chan error, 1)
	go func() { sent <- p.Input(ctx, uv.KeyPressEvent{Code: 'x', Text: "x"}) }()
	paused := true
	var inputErr error
	returned := false
	select {
	case inputErr = <-sent:
		paused = false
		returned = true
	case <-time.After(20 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	observed["paused"] = strconv.FormatBool(paused)
	close(block)
	select {
	case err = <-selected:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}
	if !returned {
		select {
		case inputErr = <-sent:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if inputErr != nil {
		return nil, inputErr
	}
	if err = e.Flush(ctx); err != nil {
		return nil, err
	}
	observed["admitted"] = p.View().Admitted
	observed["delivered"] = string(input.Bytes())
	return observed, nil
}

func observePresenterRelease(ctx context.Context) (Observation, error) {
	parent := ttyio.NewFake()
	p := terminal.NewPresenter(parent, terminal.CouchAnyMotion)
	defer p.Release(ctx)
	e, _, err := scenarioEndpoint("a")
	if err != nil {
		return nil, err
	}
	defer e.Close()
	parent.Enqueue(ttyio.WriteStep{Block: make(chan struct{})})
	selected := make(chan error, 1)
	go func() { selected <- selectQualification(ctx, p, e) }()
	select {
	case <-parent.Started():
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if err = p.Release(ctx); err != nil {
		return nil, err
	}
	select {
	case err = <-selected:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	observed := Observation{"released": strconv.FormatBool(p.View().State == terminal.Released), "select-cancelled": strconv.FormatBool(err != nil)}
	calls := parent.Calls()
	observed["late-refused"] = strconv.FormatBool(p.Present(ctx, e) != nil)
	observed["extra-writes"] = strconv.Itoa(parent.Calls() - calls)
	return observed, nil
}
