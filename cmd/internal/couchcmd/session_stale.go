package couchcmd

import (
	"context"
	"fmt"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
)

// The existing watcher offers coalesced work. Only run owns tick and its state;
// external status commands never run on the wait/supervision ticker's goroutine.
type sessionStaleWatcher struct {
	svc    *messageService
	sync   *sessionsync.Sync
	now    func() time.Time
	check  func(context.Context, string) (string, error)
	notice func(string)
	state  staleReminderState
	wake   chan struct{}
	cursor int
}

func newConsoleSessionStaleWatcher(svc *messageService, store string, console *couchtty.Console) *sessionStaleWatcher {
	// Skipped freshness checks are routine diagnostics; generated stale
	// reminders keep their independent supervision/message owner.
	return newSessionStaleWatcher(svc, store, func(text string) { console.Notify(couchtty.Notice{Kind: "status", Body: text}) })
}

func newSessionStaleWatcher(svc *messageService, store string, notice func(string)) *sessionStaleWatcher {
	return &sessionStaleWatcher{svc: svc, sync: sessionsync.New(store), now: time.Now, check: runStaleStatusCheck, notice: notice, wake: make(chan struct{}, 1)}
}
func (w *sessionStaleWatcher) offer() {
	select {
	case w.wake <- struct{}{}:
	default:
	}
}
func (w *sessionStaleWatcher) run(ctx context.Context) {
	defer advanceStaleReminder(&w.state, staleReminderEvent{kind: staleCancelled})
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.wake:
			w.tick(ctx)
		}
	}
}
func (w *sessionStaleWatcher) fault(err error) {
	if err != nil && advanceStaleReminder(&w.state, staleReminderEvent{kind: staleFault, err: err}).log {
		w.diagnostic(err)
	}
}
func (w *sessionStaleWatcher) diagnostic(err error) {
	if w.notice != nil {
		text := err.Error()
		if len(text) > 512 {
			text = text[:512]
		}
		w.notice(fmt.Sprintf("session freshness reminder skipped: %q", text))
	}
}
func (w *sessionStaleWatcher) observe(file string) (staleReminderObservation, error) {
	info, err := w.sync.InspectRegistered(file)
	o := staleReminderObservation{document: info}
	if err != nil {
		return o, err
	}
	if live := w.svc.liveness.Load(); live != nil {
		o.live = (*live)[info.Owner]
	}
	o.pending, o.known = w.svc.broker.ResourceIdle(o.live.Binding)
	o.hub = w.svc.isHubBinding(o.live.Binding)
	return o, nil
}
func (w *sessionStaleWatcher) tick(parent context.Context) {
	if parent.Err() != nil {
		advanceStaleReminder(&w.state, staleReminderEvent{kind: staleCancelled})
		return
	}
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	rows, err := w.sync.List()
	if err != nil {
		w.fault(err)
		return
	}
	ids := make(map[string]bool, len(rows))
	for _, r := range rows {
		ids[r.ID] = true
	}
	advanceStaleReminder(&w.state, staleReminderEvent{kind: staleRegistry, ids: ids})
	for visited := 0; visited < len(rows) && ctx.Err() == nil; visited++ {
		row := rows[w.cursor%len(rows)]
		w.cursor = (w.cursor + 1) % len(rows)
		if !w.state.due(row.ID, w.now()) {
			continue
		}
		before, err := w.observe(row.File)
		if err != nil {
			w.fault(err)
			continue
		}
		if before.document.ID != row.ID {
			continue
		}
		if !advanceStaleReminder(&w.state, staleReminderEvent{kind: staleTick, at: w.now(), observation: before}).check {
			continue
		}
		commandCtx, stop := context.WithTimeout(ctx, 5*time.Second)
		output, checkErr := w.check(commandCtx, before.document.File)
		stop()
		if parent.Err() != nil {
			advanceStaleReminder(&w.state, staleReminderEvent{kind: staleCancelled})
			return
		}
		body := ""
		after := staleReminderObservation{}
		if checkErr == nil {
			body, checkErr = staleReminderBody(before.document.File, output)
		}
		if checkErr == nil {
			// Failure to re-inspect is invalidated authority, never proof of freshness.
			after, err = w.observe(before.document.File)
			if err != nil {
				w.fault(err)
			}
		}
		fx := advanceStaleReminder(&w.state, staleReminderEvent{kind: staleChecked, at: w.now(), observation: after, body: body, err: checkErr})
		if fx.log {
			w.diagnostic(checkErr)
		}
		if !fx.send {
			continue
		}
		id, sendErr := newRequestID()
		if sendErr == nil {
			coordinator := w.svc.supervision.Load()
			if coordinator == nil {
				sendErr = fmt.Errorf("supervision unavailable")
			} else {
				sendCtx, stop := context.WithTimeout(ctx, 5*time.Second)
				_, sendErr = coordinator.sendGenerated(sendCtx, before.live.Binding, before.live.Binding, id, body, false)
				stop()
			}
		}
		advanceStaleReminder(&w.state, staleReminderEvent{kind: staleSent, err: sendErr})
		if sendErr != nil {
			w.fault(sendErr)
		}
	}
}
