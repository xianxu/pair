package webcast

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"

	"github.com/xianxu/pair/cmd/internal/cdp"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func canonicalFeedbackRevision(s string) (uint64, error) {
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil || strconv.FormatUint(n, 10) != s {
		return 0, board.ErrStale
	}
	return n, nil
}

// This private boundary has no optional fields. Roundtrip comparison additionally
// rejects case aliases, null invented zero values, and noncanonical uint strings.
func decodeFeedbackVisibility(raw []byte) (core.FeedbackVisibility, error) {
	var out core.FeedbackVisibility
	if len(raw) > core.MaxFeedbackVisibilityBytes {
		return out, fmt.Errorf("feedback visibility exceeds byte budget")
	}
	if err := strictjson.Decode(raw, &out); err != nil {
		return out, err
	}
	if out.Sketches == nil || out.Notes == nil || uint64(out.Scene.RecordIndex) > recording.MaxSketchInteger || uint64(out.Scene.Doc) > recording.MaxSketchInteger {
		return out, board.ErrStale
	}
	if err := core.ValidateFeedbackVisibility(out); err != nil {
		return out, err
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		return out, err
	}
	var actual, canonical any
	if err = json.Unmarshal(raw, &actual); err != nil {
		return out, err
	}
	if err = json.Unmarshal(encoded, &canonical); err != nil {
		return out, err
	}
	if !reflect.DeepEqual(actual, canonical) {
		return core.FeedbackVisibility{}, fmt.Errorf("feedback visibility fields are missing or noncanonical")
	}
	return out, nil
}

func (p *chromeSubmissionPreview) invalidateVisibilityLocked() {
	p.visibility = nil
	p.resolvedSelection = nil
	if p.visibilitySerial == ^uint64(0) {
		p.visibilityRevision = 0
		return
	}
	p.visibilitySerial++
}
func (c *Chrome) handleFeedbackVisibilityBinding(s *loopState, event cdp.Event) bool {
	if event.Method != "Runtime.bindingCalled" || len(event.Params) > 4096 {
		return false
	}
	var binding struct {
		Name    string `json:"name"`
		Context int64  `json:"executionContextId"`
		Payload string `json:"payload"`
	}
	if json.Unmarshal(event.Params, &binding) != nil || binding.Name != bindingName {
		return false
	}
	var tag struct {
		T string `json:"t"`
	}
	if json.Unmarshal([]byte(binding.Payload), &tag) != nil || tag.T != "feedback-view" {
		return false
	}
	var message struct {
		T        string `json:"t"`
		Revision string `json:"revision"`
	}
	if strictjson.Decode([]byte(binding.Payload), &message) != nil || message.T != "feedback-view" {
		return true
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal([]byte(binding.Payload), &fields) != nil || len(fields) != 2 || fields["t"] == nil || fields["revision"] == nil {
		return true
	}
	revision, err := canonicalFeedbackRevision(message.Revision)
	if err != nil || revision == 0 {
		return true
	}
	if binding.Context <= 0 || s.contexts[event.SessionID] != binding.Context || s.sessions[event.SessionID] == "" || s.sessions[event.SessionID] != castTab(s) {
		return true
	}
	c.refreshSubmissionPreviewHost(s)
	p := &c.submissionPreview
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.scope == nil || p.scope.scope.ParentSession != event.SessionID || p.scope.scope.ParentContext != binding.Context || p.visibilitySerial == ^uint64(0) {
		return true
	}
	select {
	case <-p.scope.life:
		return true
	default:
	}
	if revision < p.visibilityRevision {
		return true
	}
	p.visibilityRevision = revision
	// Invalidation and completed-paint notification may intentionally share the
	// revision. Both retire prior cached authority before waking the consumer.
	p.invalidateVisibilityLocked()
	p.notifyLocked()
	return true
}

type feedbackFence struct {
	scope            SketchSceneScope
	revision, serial uint64
}

func (p *chromeSubmissionPreview) feedbackFenceLocked(host core.PreviewHost) (feedbackFence, error) {
	if p.closed || p.scope == nil || p.host != host || host.ID == "" || p.visibilityRevision == 0 || p.visibilitySerial == ^uint64(0) {
		return feedbackFence{}, board.ErrStale
	}
	select {
	case <-p.scope.life:
		return feedbackFence{}, board.ErrStale
	default:
	}
	return feedbackFence{p.scope.scope, p.visibilityRevision, p.visibilitySerial}, nil
}
func (p *chromeSubmissionPreview) feedbackCurrentLocked(host core.PreviewHost, fence feedbackFence) bool {
	current, err := p.feedbackFenceLocked(host)
	return err == nil && current == fence
}
func (c *Chrome) ReadSubmissionVisibility(ctx context.Context, host core.PreviewHost) (core.FeedbackVisibility, error) {
	if c.conn == nil {
		return core.FeedbackVisibility{}, ErrNotRecording
	}
	return c.readSubmissionVisibility(ctx, host, c.conn)
}
func (c *Chrome) readSubmissionVisibility(ctx context.Context, host core.PreviewHost, caller SketchSceneCaller) (core.FeedbackVisibility, error) {
	if err := ctx.Err(); err != nil {
		return core.FeedbackVisibility{}, err
	}
	p := &c.submissionPreview
	p.mu.Lock()
	fence, err := p.feedbackFenceLocked(host)
	p.mu.Unlock()
	if err != nil {
		return core.FeedbackVisibility{}, err
	}
	var result struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		Exception json.RawMessage `json:"exceptionDetails"`
	}
	bounded, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	err = caller.Call(bounded, fence.scope.ParentSession, "Runtime.evaluate", map[string]any{"contextId": fence.scope.ParentContext, "returnByValue": true, "expression": "globalThis.__couchcastFeedbackVisibility()"}, &result)
	if err == nil {
		err = bounded.Err()
	}
	var view core.FeedbackVisibility
	if err == nil && len(result.Exception) > 0 {
		err = errors.New("feedback paint is not ready")
	}
	if err == nil {
		view, err = decodeFeedbackVisibility(result.Result.Value)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.feedbackCurrentLocked(host, fence) {
		return core.FeedbackVisibility{}, board.ErrStale
	}
	if err == nil && (view.Revision != fence.revision || c.opts.Record == nil || view.Scene.RecordingID != c.opts.Record.ID) {
		err = board.ErrStale
	}
	if err == nil && p.visibility != nil && !reflect.DeepEqual(*p.visibility, view) {
		err = board.ErrStale
	}
	if err != nil {
		p.invalidateVisibilityLocked()
		return core.FeedbackVisibility{}, err
	}
	owned := core.CloneFeedbackVisibility(view)
	p.visibility = &owned
	return core.CloneFeedbackVisibility(view), nil
}

type feedbackPathResolver func(context.Context, string, int, string, [][]int) ([]recording.NodeID, error)

func (c *Chrome) ResolveSubmissionPreview(ctx context.Context, host core.PreviewHost, selection core.PreviewSelection) (core.PreviewSelection, error) {
	if c.rec == nil {
		return core.PreviewSelection{}, ErrNotRecording
	}
	return c.resolveSubmissionPreview(ctx, host, selection, c.rec.ResolveOperatorPaths)
}
func (c *Chrome) resolveSubmissionPreview(ctx context.Context, host core.PreviewHost, selection core.PreviewSelection, resolve feedbackPathResolver) (core.PreviewSelection, error) {
	if err := ctx.Err(); err != nil {
		return core.PreviewSelection{}, err
	}
	selection = cloneVisibilitySelection(selection)
	if _, _, err := previewBoardWitness(selection); err != nil {
		return core.PreviewSelection{}, err
	}
	p := &c.submissionPreview
	p.mu.Lock()
	fence, err := p.feedbackFenceLocked(host)
	if err == nil && (p.visibility == nil || !reflect.DeepEqual(p.visibility, selection.Visibility)) {
		err = board.ErrStale
	}
	p.mu.Unlock()
	if err != nil {
		return core.PreviewSelection{}, err
	}
	byID := map[string]core.VisibleSketch{}
	for _, v := range selection.Visibility.Sketches {
		byID[v.ID] = v
	}
	paths := [][]int{}
	for _, v := range selection.Items {
		for _, path := range byID[v.SketchID].Paths {
			paths = append(paths, slices.Clone(path.Locator))
		}
	}
	scene := selection.Visibility.Scene
	ids, err := resolve(ctx, fence.scope.ParentSession, scene.Doc, scene.Tab, paths)
	if err == nil {
		err = ctx.Err()
	}
	if err == nil && len(ids) != len(paths) {
		err = board.ErrStale
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.feedbackCurrentLocked(host, fence) || p.visibility == nil || !reflect.DeepEqual(p.visibility, selection.Visibility) {
		return core.PreviewSelection{}, board.ErrStale
	}
	if err != nil {
		p.resolvedSelection = nil
		return core.PreviewSelection{}, err
	}
	selection.ResolvedNodes = slices.Clone(ids)
	owned := cloneVisibilitySelection(selection)
	p.resolvedSelection = &owned
	return cloneVisibilitySelection(selection), nil
}
func cloneVisibilitySelection(s core.PreviewSelection) core.PreviewSelection {
	s.Items = slices.Clone(s.Items)
	s.Candidates = slices.Clone(s.Candidates)
	s.Notes = slices.Clone(s.Notes)
	s.ResolvedNodes = slices.Clone(s.ResolvedNodes)
	if s.Page != nil {
		page := *s.Page
		s.Page = &page
	}
	if s.Visibility != nil {
		v := core.CloneFeedbackVisibility(*s.Visibility)
		s.Visibility = &v
	}
	return s
}
