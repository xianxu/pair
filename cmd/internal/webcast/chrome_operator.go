package webcast

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/recorder"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

type operatorDelivery struct {
	session string
	context int64
	reply   recorder.OperatorReply
}
type operatorTarget struct {
	session string
	context int64
	active  bool
}

// A single worker owns operator delivery. Views coalesce and replies have a
// bounded queue; lost replies are reconciled by request ID, never by resubmitting.
func (c *Chrome) startOperator(b *board.Board) {
	if b == nil {
		return
	}
	views, cancel := b.SubscribeOperator()
	c.work.Add(1)
	go func() {
		defer c.work.Done()
		defer cancel()
		view := b.OperatorView()
		for {
			var reply *operatorDelivery
			select {
			case <-c.ctx.Done():
				return
			case v, ok := <-views:
				if !ok {
					return
				}
				view = v
			case <-c.operatorRefresh:
			case d := <-c.operatorReplies:
				reply = &d
			}
			var targets []operatorTarget
			done := make(chan struct{})
			select {
			case c.requests <- func(s *loopState) {
				for session, id := range s.contexts {
					targets = append(targets, operatorTarget{session, id, s.sessions[session] == castTab(s)})
				}
				close(done)
			}:
			case <-c.ctx.Done():
				return
			}
			select {
			case <-done:
			case <-c.ctx.Done():
				return
			}
			for _, target := range targets {
				name := "__couchcastOperatorView"
				var value any = view
				if reply != nil {
					if reply.session != target.session || reply.context != target.context {
						continue
					}
					name, value = "__couchcastOperatorReply", reply.reply
				} else if !target.active {
					value = board.OperatorView{Revision: view.Revision, Notes: []*recording.Note{}, Strokes: []board.OperatorStroke{}}
				}
				data, _ := json.Marshal(value)
				ctx, cancel := context.WithTimeout(c.ctx, callTimeout)
				_ = c.conn.Call(ctx, target.session, "Runtime.evaluate", map[string]any{"expression": "globalThis." + name + " && globalThis." + name + "(" + string(data) + ")", "contextId": target.context}, nil)
				cancel()
			}
		}
	}()
}

func (c *Chrome) operatorReply(session string, id int64, reply recorder.OperatorReply) {
	select {
	case c.operatorReplies <- operatorDelivery{session, id, reply}:
	default:
	}
}
func (c *Chrome) refreshOperator() {
	select {
	case c.operatorRefresh <- struct{}{}:
	default:
	}
}

// UIGuard updates target lifetime state before a remote dispatch. Replacement
// documents start hidden and cannot reveal their host while the guard is held.
func (c *Chrome) uiGuard(session, action string) error {
	done := make(chan struct{})
	var err error
	select {
	case c.requests <- func(s *loopState) {
		if s.uiGuards == nil {
			s.uiGuards = map[string]string{}
		}
		if action == "acquire" && s.uiGuards[session] != "" {
			err = errors.New("remote input is uncertain; close and reopen the cast Chrome")
		} else {
			if action == "release" {
				delete(s.uiGuards, session)
			} else {
				s.uiGuards[session] = action
			}
			if id := s.contexts[session]; id != 0 {
				c.goSetControl(session, id, s.phase == PhaseLive, s.driver, s.uiGuards[session] != "")
			}
		}
		close(done)
	}:
	case <-c.ctx.Done():
		return ErrEnded
	}
	select {
	case <-done:
		return err
	case <-c.ctx.Done():
		return ErrEnded
	}
}

// SetAnnotationMode selects the same mode as the operator rail, without raising
// Chrome or moving page focus. The isolated-world policy rechecks availability.
func (c *Chrome) SetAnnotationMode(mode string) error {
	if mode != "watch" && mode != "point" && mode != "note" {
		return errors.New("unknown annotation mode")
	}
	if c.opts.Record == nil || c.opts.Record.Operator == nil {
		return ErrNotRecording
	}
	var target operatorTarget
	done := make(chan struct{})
	select {
	case c.requests <- func(s *loopState) {
		session := sessionOf(s, castTab(s))
		target = operatorTarget{session: session, context: s.contexts[session]}
		close(done)
	}:
	case <-c.ctx.Done():
		return ErrEnded
	}
	select {
	case <-done:
	case <-c.ctx.Done():
		return ErrEnded
	}
	if target.context == 0 {
		return errors.New("the cast page is not ready for annotation")
	}
	data, _ := json.Marshal(mode)
	var result struct {
		Result struct {
			Value bool `json:"value"`
		} `json:"result"`
	}
	if err := c.call(target.session, "Runtime.evaluate", map[string]any{"contextId": target.context, "returnByValue": true, "expression": "!!(globalThis.__couchcastOperatorMode && globalThis.__couchcastOperatorMode(" + string(data) + "))"}, &result); err != nil {
		return err
	}
	if !result.Result.Value {
		return errors.New("annotation is unavailable while paused or a viewer is driving")
	}
	return nil
}

func (c *Cast) SetAnnotationMode(mode string) error {
	if b, ok := c.browser.(interface{ SetAnnotationMode(string) error }); ok {
		return b.SetAnnotationMode(mode)
	}
	return ErrNotRecording
}
