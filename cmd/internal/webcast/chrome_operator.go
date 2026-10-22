package webcast

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"github.com/xianxu/pair/cmd/internal/webcast/recorder"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

type operatorDelivery struct {
	session string
	context int64
	reply   recorder.OperatorReply
}
type operatorTarget struct {
	session        string
	context        int64
	active         bool
	sketchScope    SketchSceneScope
	sketchLifetime <-chan struct{}
}

// Delivery revision is independent of authority generation: switching the
// active target can change permission without changing Board authority.
type operatorContextID struct {
	session string
	context int64
}
type operatorPermission struct {
	generation      uint64
	active, desired bool
	revision        uint64
	phase           operatorPermissionPhase
}
type operatorPermissionPhase uint8

const (
	operatorPermissionPending operatorPermissionPhase = iota
	operatorPermissionReady
	operatorPermissionFencing // an unconfirmed enable needs a newer false receipt
)

func (p *operatorPermission) allowed() bool { return p.desired && p.phase != operatorPermissionFencing }
func (p *operatorPermission) want(generation uint64, active, allowed bool, next func() uint64) {
	if p.revision != 0 && p.generation == generation && p.active == active && p.desired == allowed {
		return
	}
	p.generation, p.active, p.desired = generation, active, allowed
	p.revision = next()
	if p.phase != operatorPermissionFencing {
		p.phase = operatorPermissionPending
	}
}
func operatorPermissionExpression(revision, generation uint64, allowed bool) string {
	return fmt.Sprintf(`(()=>{const hook=globalThis.__couchcastAnnotationAllowed;if(typeof hook!=='function')return null;const revision=%q;const previous=globalThis.__couchcastAnnotationDelivery;if(!previous||BigInt(revision)>BigInt(previous.revision)){hook(%t,%q);globalThis.__couchcastAnnotationDelivery={revision,generation:%q,allowed:%t};}return globalThis.__couchcastAnnotationDelivery;})()`, strconv.FormatUint(revision, 10), allowed, strconv.FormatUint(generation, 10), strconv.FormatUint(generation, 10), allowed)
}
func (c *Chrome) deliverOperatorPermission(target operatorTarget, p *operatorPermission, next func() uint64) {
	if p.phase == operatorPermissionReady {
		return
	}
	allowed := p.allowed()
	var result struct {
		Result struct {
			Value *struct {
				Revision   string `json:"revision"`
				Generation string `json:"generation"`
				Allowed    bool   `json:"allowed"`
			} `json:"value"`
		} `json:"result"`
		Exception json.RawMessage `json:"exceptionDetails"`
	}
	ctx, cancel := context.WithTimeout(c.ctx, callTimeout)
	err := c.conn.Call(ctx, target.session, "Runtime.evaluate", map[string]any{"contextId": target.context, "returnByValue": true, "expression": operatorPermissionExpression(p.revision, p.generation, allowed)}, &result)
	cancel()
	receipt := result.Result.Value
	if err == nil && len(result.Exception) == 0 && receipt != nil && receipt.Revision == strconv.FormatUint(p.revision, 10) && receipt.Allowed == allowed && receipt.Generation == strconv.FormatUint(p.generation, 10) {
		if p.phase == operatorPermissionFencing && p.desired {
			p.revision, p.phase = next(), operatorPermissionPending
		} else {
			p.phase = operatorPermissionReady
		}
		return
	}
	// A missing hook has no effect; mount sends another hello when it is ready.
	// An error is uncertain: Call cancellation does not cancel Chrome execution.
	if allowed && (err != nil || len(result.Exception) != 0) {
		p.revision, p.phase = next(), operatorPermissionFencing
	}
}

// inactiveOperatorView retires page visibility without losing confirmed display
// knowledge. Reuse the same snapshot's mask so its notes cannot cross a revision.
func inactiveOperatorView(view board.OperatorView) board.OperatorView {
	return board.OperatorView{Revision: view.Revision, FeedbackMask: view.FeedbackMask, Notes: []*recording.Note{}, Strokes: []board.OperatorStroke{}}
}

// A single worker owns operator delivery. Views coalesce and replies have a
// bounded queue; lost replies are reconciled by request ID, never by resubmitting.
func (c *Chrome) startOperator(b *board.Board) {
	if b == nil {
		return
	}
	views, cancel := b.SubscribeOperator()
	permissions, cancelPermissions := b.SubscribeAnnotationCapability()
	sketchRuntime := c.initOperatorSketch(b)
	var sketchWake, sketchRecords <-chan struct{}
	cancelSketch := func() {}
	if sketchRuntime != nil {
		sketchWake = sketchRuntime.Wake()
		sketchRecords, cancelSketch = b.SubscribeSketchDelivery()
	}
	c.work.Add(1)
	go func() {
		defer c.work.Done()
		defer cancel()
		defer cancelPermissions()
		defer cancelSketch()
		if sketchRuntime != nil {
			defer sketchRuntime.Close(context.WithoutCancel(c.ctx))
		}
		// One worker and one 250ms timer reconcile unacknowledged contexts; normal
		// notifications may reconcile sooner. Each call has callTimeout, with no
		// overlapping delivery goroutines. ACK disables timer selection; context
		// disappearance prunes state, and cast shutdown stops worker and ticker.
		retry := time.NewTicker(250 * time.Millisecond)
		defer retry.Stop()
		states := map[operatorContextID]*operatorPermission{}
		originals := map[operatorContextID]*operatorOriginals{}
		drawingPermissions := map[operatorContextID]*operatorDrawingPermission{}
		var revision uint64
		next := func() uint64 { revision++; return revision }
		var retryC <-chan time.Time
		view := b.OperatorView()
		for {
			var reply *operatorDelivery
			normalDelivery := true
			select {
			case <-c.ctx.Done():
				return
			case v, ok := <-views:
				if !ok {
					return
				}
				view = v
			case <-c.operatorRefresh:
			case <-sketchWake:
				normalDelivery = false
			case _, ok := <-sketchRecords:
				if !ok {
					return
				}
				normalDelivery = false
			case _, ok := <-permissions:
				if !ok {
					return
				}
			case <-retryC:
			case d := <-c.operatorReplies:
				reply = &d
			}
			var targets []operatorTarget
			done := make(chan struct{})
			select {
			case c.requests <- func(s *loopState) {
				for session, id := range s.contexts {
					scope := c.operatorSketchScope(s, session, id)
					targets = append(targets, operatorTarget{session: session, context: id, active: s.sessions[session] == castTab(s), sketchScope: scope.scope, sketchLifetime: scope.life})
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
			live := map[operatorContextID]bool{}
			originalsPending := false
			var sketchTargets []operatorSketchTarget
			for _, target := range targets {
				key := operatorContextID{target.session, target.context}
				live[key] = true
				permission := states[key]
				if permission == nil {
					permission = &operatorPermission{}
					states[key] = permission
				}
				snapshot := b.AnnotationCapability()
				permission.want(snapshot.Generation, target.active, snapshot.Phase == capability.On && target.active, next)
				c.deliverOperatorPermission(target, permission, next)
				if sketchRuntime != nil {
					p := drawingPermissions[key]
					if p == nil {
						p = &operatorDrawingPermission{}
						drawingPermissions[key] = p
					}
					c.deliverOperatorDrawing(target, b, p, next)
					sketchTargets = append(sketchTargets, operatorSketchTarget{Scope: target.sketchScope, Lifetime: target.sketchLifetime, Active: target.active})
					if target.active {
						cache := originals[key]
						if cache == nil {
							cache = &operatorOriginals{}
							originals[key] = cache
						}
						revision := b.SketchAnchorRevision()
						if cache.sketchRevision != revision || cache.sketchDocument != view.Doc {
							originalsPending = true
							cache.sketchPending = true
							if sketchRuntime.originalsReady(target) {
								cache.acceptSketchAnchors(view.Doc, revision, b.SketchAnchorNodes(view.Doc))
							}
						}
					}
				}
				if !normalDelivery {
					continue
				}
				name := "__couchcastOperatorView"
				var value any = view
				if reply != nil {
					if reply.session != target.session || reply.context != target.context {
						continue
					}
					name, value = "__couchcastOperatorReply", reply.reply
				} else if !target.active {
					delete(originals, key)
					value = inactiveOperatorView(view)
				}
				data, _ := json.Marshal(value)
				ctx, cancel := context.WithTimeout(c.ctx, callTimeout)
				_ = c.conn.Call(ctx, target.session, "Runtime.evaluate", map[string]any{"expression": "globalThis." + name + " && globalThis." + name + "(" + string(data) + ")", "contextId": target.context}, nil)
				cancel()
				if reply == nil && target.active {
					cache := originals[key]
					if cache == nil {
						cache = &operatorOriginals{}
						originals[key] = cache
					}
					bindingView := view
					bindingView.Revision += cache.sketchRevision
					originalsPending = c.deliverOperatorOriginals(target, bindingView, cache) || originalsPending
				}
			}
			if sketchRuntime != nil {
				sketchRuntime.Reconcile(sketchTargets)
				if sketchRuntime.Turn(c.ctx) {
					c.submissionPreviewDeliveryProgress()
				}
			}
			retryC = nil
			if originalsPending {
				retryC = retry.C
			}
			if sketchRuntime != nil && sketchRuntime.Pending() {
				retryC = retry.C
			}
			for key, p := range states {
				if !live[key] {
					delete(states, key)
					delete(originals, key)
					delete(drawingPermissions, key)
					continue
				}
				if p.phase != operatorPermissionReady {
					retryC = retry.C
				}
				if drawing := drawingPermissions[key]; drawing != nil && drawing.permission.phase != operatorPermissionReady {
					retryC = retry.C
				}
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
	c.submissionPreviewDeliveryProgress()
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
	var admission board.AnnotationAdmission
	if mode != "watch" {
		var err error
		admission, err = c.opts.Record.Operator.AdmitAnnotation()
		if err != nil {
			return err
		}
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
	if mode != "watch" {
		if err := admission.Check(); err != nil {
			return err
		}
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
