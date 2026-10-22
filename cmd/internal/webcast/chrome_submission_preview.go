package webcast

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"slices"
	"strconv"
	"sync"
)

type previewAwaitingDelivery struct {
	host      core.PreviewHost
	operation uint64
}

type chromeSubmissionPreview struct {
	awaiting *previewAwaitingDelivery
	mu       sync.Mutex
	host     core.PreviewHost
	scope    *operatorSketchScope
	serial   uint64
	closed   bool
	subs     map[chan struct{}]struct{}
}

// refreshSubmissionPreviewHost runs only on Chrome's event loop, before its
// state is published. A cached host fence makes chord-time freeze IO-free.
func (c *Chrome) refreshSubmissionPreviewHost(s *loopState) {
	var scope *operatorSketchScope
	if c.opts.Record != nil && c.opts.Record.Operator != nil && s.ending == notEnding {
		active := castTab(s)
		session := ""
		for candidate, target := range s.sessions {
			if target == active && s.contexts[candidate] > 0 && (session == "" || candidate < session) {
				session = candidate
			}
		}
		if session != "" {
			scope = c.operatorSketchScope(s, session, s.contexts[session])
		}
	}
	p := &c.submissionPreview
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.scope == scope {
		return
	}
	p.scope = scope
	p.awaiting = nil
	p.host = core.PreviewHost{}
	if scope != nil {
		if p.serial == ^uint64(0) {
			p.closed = true
			p.scope = nil
			for ch := range p.subs {
				close(ch)
				delete(p.subs, ch)
			}
			return
		}
		p.serial++
		identity, _ := json.Marshal(scope.scope)
		p.host = core.PreviewHost{ID: string(identity), Generation: p.serial}
	}
	p.notifyLocked()
}
func (p *chromeSubmissionPreview) notifyLocked() {
	for ch := range p.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
func (c *Chrome) closeSubmissionPreview() {
	p := &c.submissionPreview
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	p.scope = nil
	p.awaiting = nil
	p.host = core.PreviewHost{}
	for ch := range p.subs {
		close(ch)
		delete(p.subs, ch)
	}
}
func (c *Chrome) SubmissionPreviewHost(ctx context.Context) (core.PreviewHost, <-chan struct{}, error) {
	if err := ctx.Err(); err != nil {
		return core.PreviewHost{}, nil, err
	}
	p := &c.submissionPreview
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.scope == nil || p.host.ID == "" {
		return core.PreviewHost{}, nil, ErrNotRecording
	}
	select {
	case <-p.scope.life:
		return core.PreviewHost{}, nil, board.ErrStale
	default:
	}
	return p.host, p.scope.life, nil
}
func (c *Chrome) SubmissionPreviewUpdates() (<-chan struct{}, func()) {
	p := &c.submissionPreview
	p.mu.Lock()
	defer p.mu.Unlock()
	ch := make(chan struct{}, 1)
	if p.closed {
		close(ch)
		return ch, func() {}
	}
	if p.subs == nil {
		p.subs = map[chan struct{}]struct{}{}
	}
	p.subs[ch] = struct{}{}
	ch <- struct{}{}
	return ch, func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		if _, ok := p.subs[ch]; ok {
			delete(p.subs, ch)
			close(ch)
		}
	}
}

// withSubmissionPreviewHost is the shared linearization guard between
// lifecycle retirement and the short board freeze; its callback performs no IO.
func (c *Chrome) withSubmissionPreviewHost(host core.PreviewHost, freeze func() (board.Selection, error)) (board.Selection, error) {
	p := &c.submissionPreview
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.scope == nil || host.ID == "" || p.host != host {
		return board.Selection{}, board.ErrStale
	}
	select {
	case <-p.scope.life:
		return board.Selection{}, board.ErrStale
	default:
	}
	return freeze()
}
func (c *Chrome) FreezeSubmissionPreview(host core.PreviewHost, g uint64, chosen []board.SelectedSketch) (board.Selection, error) {
	if c.opts.Record == nil || c.opts.Record.Operator == nil {
		return board.Selection{}, ErrNotRecording
	}
	return c.withSubmissionPreviewHost(host, func() (board.Selection, error) { return c.opts.Record.Operator.FreezeSubmissionSelection(g, chosen) })
}
func (c *Chrome) retireSubmissionPreviewScope(scope *operatorSketchScope) {
	p := &c.submissionPreview
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.scope != scope {
		return
	}
	p.scope = nil
	p.host = core.PreviewHost{}
	p.awaiting = nil
	p.notifyLocked()
}

type submissionPreviewItem struct {
	RecordingID string `json:"recording_id"`
	ID          string `json:"id"`
	Sequence    uint64 `json:"next_sequence"`
	Digest      string `json:"digest"`
}
type submissionPreviewStamp struct {
	Board     string `json:"board"`
	Delivered string `json:"delivered"`
}
type submissionPreviewValue struct {
	Host       string                     `json:"host"`
	Generation string                     `json:"generation"`
	Revision   string                     `json:"revision"`
	Operation  string                     `json:"operation"`
	Stamp      submissionPreviewStamp     `json:"stamp"`
	Items      []submissionPreviewItem    `json:"items"`
	Remaining  int                        `json:"remaining"`
	Cursor     board.SketchDeliveryCursor `json:"cursor"`
	Accepted   bool                       `json:"accepted,omitempty"`
}

func (c *Chrome) PublishSubmissionPreview(ctx context.Context, pub core.PreviewPublication) error {
	selection := pub.Selection()
	if c.opts.Record == nil || c.opts.Record.Operator == nil || len(selection.Items) > core.MaxSubmissionSketches || selection.Remaining < 0 || selection.Remaining > 256 || pub.Operation() == 0 {
		return errors.New("submission preview is unavailable")
	}
	chosen := make([]board.SelectedSketch, 0, len(selection.Items))
	value := submissionPreviewValue{Host: pub.Host().ID, Generation: strconv.FormatUint(pub.Host().Generation, 10), Revision: strconv.FormatUint(pub.Revision(), 10), Operation: strconv.FormatUint(pub.Operation(), 10), Stamp: submissionPreviewStamp{Board: strconv.FormatUint(selection.Stamp.Board, 10), Delivered: strconv.FormatUint(selection.Stamp.Delivered, 10)}, Items: []submissionPreviewItem{}, Remaining: selection.Remaining}
	for _, item := range selection.Items {
		if item.RecordingID != c.opts.Record.ID {
			return errors.New("submission preview recording changed")
		}
		chosen = append(chosen, board.SelectedSketch{ID: item.SketchID, NextSequence: item.Sequence, Digest: item.Digest})
		value.Items = append(value.Items, submissionPreviewItem{RecordingID: item.RecordingID, ID: item.SketchID, Sequence: item.Sequence, Digest: item.Digest})
	}
	current := func() error {
		_, err := c.FreezeSubmissionPreview(pub.Host(), selection.Stamp.Board, chosen)
		return err
	}
	if err := current(); err != nil {
		return err
	}
	fence := &c.submissionPreview
	fence.mu.Lock()
	scope := fence.scope
	fence.mu.Unlock()
	if scope == nil {
		return board.ErrStale
	}
	value.Cursor = c.opts.Record.Operator.OperatorSketchCursor()
	if !c.awaitSubmissionPreviewDelivery(pub.Host(), pub.Operation()) {
		return board.ErrStale
	}
	err := deliverSubmissionPreview(ctx, c.conn, scope.scope, value, current)
	if err == nil || ctx.Err() != nil {
		c.finishSubmissionPreviewDelivery(pub.Host(), pub.Operation())
	}
	return err
}

// Awaiting is one owned publication, never per-point state. Delivery progress
// only wakes the manager when it can unblock a previously unavailable preview.
func (c *Chrome) awaitSubmissionPreviewDelivery(host core.PreviewHost, operation uint64) bool {
	p := &c.submissionPreview
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.host != host || p.scope == nil {
		return false
	}
	p.awaiting = &previewAwaitingDelivery{host: host, operation: operation}
	return true
}
func (c *Chrome) finishSubmissionPreviewDelivery(host core.PreviewHost, operation uint64) {
	p := &c.submissionPreview
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.awaiting != nil && p.awaiting.host == host && p.awaiting.operation == operation {
		p.awaiting = nil
	}
}
func (c *Chrome) submissionPreviewDeliveryProgress() {
	p := &c.submissionPreview
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.closed && p.awaiting != nil {
		p.notifyLocked()
	}
}

// deliverSubmissionPreview is the injected CDP seam. Validation brackets IO;
// cancellation never makes a late JavaScript result authoritative.
func deliverSubmissionPreview(ctx context.Context, caller SketchSceneCaller, scope SketchSceneScope, value submissionPreviewValue, current func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := current(); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var result struct {
		Result struct {
			Value *submissionPreviewValue `json:"value"`
		} `json:"result"`
		Exception json.RawMessage `json:"exceptionDetails"`
	}
	bounded, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	err = caller.Call(bounded, scope.ParentSession, "Runtime.evaluate", map[string]any{"contextId": scope.ParentContext, "returnByValue": true, "expression": "globalThis.__couchcastSketchSubmissionPreview&&globalThis.__couchcastSketchSubmissionPreview(" + string(data) + ")"}, &result)
	if err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = current(); err != nil {
		return err
	}
	ack := result.Result.Value
	if len(result.Exception) > 0 || ack == nil || !ack.Accepted || ack.Host != value.Host || ack.Generation != value.Generation || ack.Revision != value.Revision || ack.Operation != value.Operation || ack.Stamp != value.Stamp || ack.Remaining != value.Remaining || ack.Cursor != value.Cursor || !slices.Equal(ack.Items, value.Items) {
		return fmt.Errorf("submission preview was not acknowledged; review the updated preview")
	}
	return nil
}
func (c *Cast) SubmissionPreviewHost(ctx context.Context) (core.PreviewHost, <-chan struct{}, error) {
	if browser, ok := c.browser.(interface {
		SubmissionPreviewHost(context.Context) (core.PreviewHost, <-chan struct{}, error)
	}); ok {
		return browser.SubmissionPreviewHost(ctx)
	}
	return core.PreviewHost{}, nil, ErrNotRecording
}
func (c *Cast) SubmissionPreviewUpdates() (<-chan struct{}, func()) {
	if browser, ok := c.browser.(interface {
		SubmissionPreviewUpdates() (<-chan struct{}, func())
	}); ok {
		return browser.SubmissionPreviewUpdates()
	}
	ch := make(chan struct{})
	close(ch)
	return ch, func() {}
}
func (c *Cast) PublishSubmissionPreview(ctx context.Context, p core.PreviewPublication) error {
	if browser, ok := c.browser.(interface {
		PublishSubmissionPreview(context.Context, core.PreviewPublication) error
	}); ok {
		return browser.PublishSubmissionPreview(ctx, p)
	}
	return ErrNotRecording
}
func (c *Cast) FreezeSubmissionPreview(host core.PreviewHost, g uint64, chosen []board.SelectedSketch) (board.Selection, error) {
	if browser, ok := c.browser.(interface {
		FreezeSubmissionPreview(core.PreviewHost, uint64, []board.SelectedSketch) (board.Selection, error)
	}); ok {
		return browser.FreezeSubmissionPreview(host, g, chosen)
	}
	return board.Selection{}, ErrNotRecording
}
