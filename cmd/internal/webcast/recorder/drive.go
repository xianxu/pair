package recorder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/xianxu/pair/cmd/internal/cdp"
	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Remote driving (#467): a driver's batch runs as one input of the queue,
// so its input is ordered with everything else the stream holds. The run
// holds the observer in every context of the cast tab (what it had pending
// is the operator's or the page's: recorded without by), dispatches, then
// takes. During the hold the observer tags what it sends with by, through
// the binding, in order (a click at once, before a navigation can take its
// document); the take sends the rest, tagged. Every exit takes; a hold the
// take can't reach (a dialog, a context gone) expires in the page.

var _ drive.TabRunner = (*Recorder)(nil)

// Opened reports whether the recorder has opened its recording (the first
// go-live). It never goes back: a record emitted after it reads true is
// ordered after the open, so it isn't dropped as before-live.
func (r *Recorder) Opened() bool { return r.openedFlag.Load() }

// WithTab runs fn with the cast tab on the recorder's goroutine (#467).
func (r *Recorder) WithTab(ctx context.Context, by string, fn func(drive.Tab) error) error {
	return r.acknowledged(ctx, func() error { return r.withTab(ctx, by, fn) })
}

// acknowledged owns cancellation while queued; a started effect owns its
// completion and cannot lose an uncertain result when its caller cancels.
func (r *Recorder) acknowledged(ctx context.Context, fn func() error) error {
	// Claim the queued callback or cancel it before it starts. Once running,
	// await its bounded outcome even if the HTTP caller leaves: an uncertain
	// external effect must still reach the controller's recovery state.
	var phase atomic.Uint32 // 0 queued, 1 running, 2 canceled before execution
	result := make(chan error, 1)
	q := query{done: make(chan struct{}), fn: func() {
		if phase.CompareAndSwap(0, 1) {
			if err := ctx.Err(); err != nil {
				result <- &capability.Failure{Outcome: capability.Retryable, Cause: err}
			} else {
				result <- fn()
			}
		}
	}}
	if !r.enqueue(q, false) {
		return drive.ErrClosed
	}
	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		if phase.CompareAndSwap(0, 2) {
			return &capability.Failure{Outcome: capability.Retryable, Cause: ctx.Err()}
		}
	}
	return <-result
}

func (r *Recorder) withTab(ctx context.Context, by string, fn func(drive.Tab) error) error {
	if err := ctx.Err(); err != nil {
		return &capability.Failure{Outcome: capability.Retryable, Cause: err} // nothing submitted
	}
	if !r.live || r.top == "" {
		return fn(&cdpTab{r: r}) // the rules refuse it (not live)
	}

	if r.opts.UIGuard != nil {
		if err := r.opts.UIGuard(r.top, "acquire"); err != nil {
			return fmt.Errorf("%w: %v", drive.ErrInput, err)
		}
		session := r.top
		id, _ := r.topContext(session)
		token := fmt.Sprintf("input-%d", uiToken.Add(1))
		r.inputUncertain = false
		guardCtx, cancel := context.WithTimeout(r.ctx, callTimeout)
		ok := uiCall(guardCtx, r.opts.Call, session, id, "acquire", token, 0)
		cancel()
		if !ok {
			r.opts.UIGuard(session, "unknown")
			return fmt.Errorf("%w: %v", drive.ErrInput, errUISuppression)
		}
		defer func() {
			if r.inputUncertain {
				r.opts.UIGuard(session, "unknown")
				if r.opts.Logf != nil {
					r.opts.Logf("Remote input completion is uncertain; close and reopen the cast Chrome to restore Couch UI.")
				}
				return
			}
			// Every issued input returned a definitive reply. A release only
			// touches this document's token; replacement documents use guard state.
			restore, cancel := context.WithTimeout(r.ctx, 5*time.Second)
			defer cancel()
			uiCall(restore, r.opts.Call, session, id, "release", token, 0)
			r.opts.UIGuard(session, "release")
		}()
	}
	contexts := r.castContexts()
	for _, c := range contexts {
		if items, ok := r.observerCall(c, "__couchcastHold", by); ok {
			r.observeItems(r.t(r.opts.Now()), c.session, c.id, items, "")
		}
	}
	defer func() {
		// The take sends the rest through the binding, tagged, and returns
		// nothing: the driver's reports all arrive as ordered binding
		// events. A failed take is the hold's to expire.
		for _, c := range contexts {
			r.observerCall(c, "__couchcastTake", "")
		}
	}()
	return fn(&cdpTab{r: r})
}

type worldContext struct {
	session string
	id      int64
}

// castContexts are the observer's contexts in the cast tab: its page
// session and every frame session under it.
func (r *Recorder) castContexts() []worldContext {
	var out []worldContext
	for session, ctxs := range r.contexts {
		if !r.underTop(session) {
			continue
		}
		for id, frame := range ctxs {
			if r.excludedFrame(frame) || r.excludedSession(session) {
				continue
			}
			out = append(out, worldContext{session, id})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].session != out[j].session {
			return out[i].session < out[j].session
		}
		return out[i].id < out[j].id
	})
	return out
}

func (r *Recorder) underTop(session string) bool {
	for seen := 0; session != "" && seen < 64; seen++ {
		if session == r.top {
			return true
		}
		session = r.parents[session]
	}
	return false
}

// observerCall evaluates the observer's hold or take in one context and
// returns its items; false when it can't (a dialog, a context gone).
func (r *Recorder) observerCall(c worldContext, fn, arg string) ([]Item, bool) {
	var res struct {
		Result struct {
			Value string `json:"value"`
		} `json:"result"`
	}
	quoted, _ := json.Marshal(arg)
	params := map[string]any{"expression": "globalThis." + fn + " ? globalThis." + fn + "(" + string(quoted) + ") : '[]'", "contextId": c.id, "returnByValue": true}
	if err := r.inputCall(c.session, "Runtime.evaluate", params, &res); err != nil {
		return nil, false
	}
	var items []Item
	if json.Unmarshal([]byte(res.Result.Value), &items) != nil {
		return nil, false
	}
	return items, true
}

// inputCall is a CDP call a JavaScript dialog would stop: refused while
// one is open, and given up the moment one opens. The call itself is left
// to finish or time out on its own goroutine (bounded by callTimeout).
func (r *Recorder) inputCall(session, method string, params, result any) error {
	dialog := r.dialog()
	select {
	case <-dialog:
		return &capability.Failure{Outcome: capability.Retryable, Cause: drive.ErrDialog}
	default:
	}
	done := make(chan error, 1)
	r.side.Add(1)
	go func() {
		defer r.side.Done()
		_, err := r.call(session, method, params, result)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			var definitive *cdp.Error
			if !errors.As(err, &definitive) {
				r.inputUncertain = true
			}
		}
		return err
	case <-dialog:
		r.inputUncertain = true
		return &capability.Failure{Outcome: capability.Broken, Cause: drive.ErrDialog}
	}
}

// dialog is closed while the cast tab shows a JavaScript dialog.
func (r *Recorder) dialog() <-chan struct{} {
	if r.opts.Dialog == nil {
		return nil
	}
	return r.opts.Dialog()
}

// cdpTab is drive.Tab on the recorder's goroutine: the mirror's state, CDP
// hit tests and input into the cast tab's page session.
type cdpTab struct{ r *Recorder }

func (t *cdpTab) Page() *rec.State { return t.r.m.State() }

func (t *cdpTab) At(x, y float64) (rec.NodeID, error) {
	var res struct {
		BackendNodeID int `json:"backendNodeId"`
	}
	// DOM.getNodeForLocation hit-tests at top-page CSS px, the act's own
	// coordinates, and finds only what is in the viewport (pair#505,
	// measured: at scroll 236, page (94,405) is the link the viewer aimed
	// at and viewport (94,169) is "No node found"; subtracting the scroll
	// made every click after a scroll stale). Not into user-agent shadow
	// DOM: an <input>'s inner editor isn't a node the recording has
	// (measured: every click on an input read as stale).
	err := t.r.inputCall(t.r.top, "DOM.getNodeForLocation", map[string]any{
		"x": int(x), "y": int(y), "includeUserAgentShadowDOM": false, "ignorePointerEventsNone": true,
	}, &res)
	var ce *cdp.Error
	switch {
	case err == nil:
		id, _ := t.r.m.IDOf(t.r.top, res.BackendNodeID)
		return id, nil
	case errors.Is(err, drive.ErrDialog):
		return 0, err
	case errors.As(err, &ce) && strings.HasPrefix(ce.Message, "No node found"):
		return 0, nil // nothing there (off the viewport): stale
	}
	return 0, fmt.Errorf("%w: hit test: %v", drive.ErrInput, err)
}

func (t *cdpTab) Dispatch(a drive.Act) error {
	for i, c := range cdpAct(a, t.r.m.TopScroll()) {
		if err := t.r.inputCall(t.r.top, c.Method, c.Params, nil); err != nil {
			// Composite acts may have performed earlier commands even if
			// this command failed before submission. Never replay that act.
			if i > 0 {
				err = &capability.Failure{Outcome: capability.Broken, Cause: err}
			}
			if errors.Is(err, drive.ErrDialog) {
				return err
			}
			return fmt.Errorf("%w: %s: %w", drive.ErrInput, c.Method, err)
		}
	}
	return nil
}
