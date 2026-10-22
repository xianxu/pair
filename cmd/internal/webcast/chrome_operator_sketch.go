package webcast

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// These envelopes carry no authority. The Chrome event loop supplies the exact
// session, isolated context, loader and closeable document lifetime separately.
type operatorSketchEnvelope struct {
	Request    string          `json:"request"`
	Kind       string          `json:"kind"`
	Generation string          `json:"generation"`
	Epoch      uint64          `json:"epoch"`
	Body       json.RawMessage `json:"body"`
}

func parseOperatorSketchEnvelope(raw []byte) (operatorSketchEnvelope, error) {
	var e operatorSketchEnvelope
	if len(raw) > board.MaxSketchBody+1024 {
		return e, board.ErrShape
	}
	known := map[string]bool{"request": true, "kind": true, "generation": true, "epoch": true, "body": true}
	owned, unknown, shapeErr := strictjson.SplitObject(raw, known, nil)
	if shapeErr != nil || len(unknown) != 0 || len(owned) != 5 {
		return e, board.ErrShape
	}
	if len(raw) > board.MaxSketchBody+1024 || strictjson.ValidateUnicode(raw) != nil || strictjson.Decode(raw, &e) != nil {
		return e, board.ErrShape
	}
	n, err := strconv.ParseUint(e.Generation, 10, 64)
	if err != nil || strconv.FormatUint(n, 10) != e.Generation || e.Epoch > recording.MaxSketchInteger || len(e.Request) == 0 || len(e.Request) > 128 || strings.TrimSpace(e.Request) != e.Request || len(e.Body) == 0 || bytes.Equal(bytes.TrimSpace(e.Body), []byte("null")) {
		return e, board.ErrShape
	}
	if strings.HasPrefix(e.Kind, "sketch-") {
		if _, err := board.ParseSketchCommand(e.Kind, e.Body); err != nil {
			return e, err
		}
	} else {
		switch e.Kind {
		case "scene-open", "scene-prepare", "scene-hit", "scene-placement", "scene-dispose":
			if err := validateOperatorSketchSceneBody(e.Kind, e.Body); err != nil {
				return e, err
			}
		default:
			return e, board.ErrShape
		}
	}
	return e, nil
}

func validateOperatorSketchSceneBody(kind string, raw []byte) error {
	keys := map[string]bool{}
	switch kind {
	case "scene-open", "scene-dispose":
	case "scene-prepare":
		keys["sketch"], keys["source"] = true, true
	case "scene-hit":
		keys["point"] = true
	case "scene-placement":
		keys["path"] = true
	}
	owned, unknown, err := strictjson.SplitObject(raw, keys, nil)
	if err != nil || len(unknown) != 0 || len(owned) != len(keys) {
		return board.ErrShape
	}
	switch kind {
	case "scene-prepare":
		var body struct {
			Sketch string                       `json:"sketch"`
			Source board.SketchSourceDescriptor `json:"source"`
		}
		if strictjson.Decode(raw, &body) != nil {
			return board.ErrShape
		}
		// Reuse the source owner's exact descriptor grammar and numeric bounds.
		reply, _ := json.Marshal(map[string]any{"operation": "scene", "sketch": body.Sketch, "sequence": 0, "next_sequence": 0, "outcome": "accepted", "code": "", "source": owned["source"]})
		if _, err := board.ParseSketchReply(reply); err != nil {
			return err
		}
	case "scene-hit":
		var coordinates []json.RawMessage
		if json.Unmarshal(owned["point"], &coordinates) != nil || len(coordinates) != 2 {
			return board.ErrShape
		}
		for _, coordinate := range coordinates {
			if bytes.Equal(bytes.TrimSpace(coordinate), []byte("null")) {
				return board.ErrShape
			}
		}
		var body struct {
			Point []float64 `json:"point"`
		}
		if strictjson.Decode(raw, &body) != nil || len(body.Point) != 2 {
			return board.ErrShape
		}
		for _, n := range body.Point {
			if n < -recording.SketchMaxCoordinate || n > recording.SketchMaxCoordinate {
				return board.ErrShape
			}
		}
	case "scene-placement":
		value := bytes.TrimSpace(owned["path"])
		if len(value) == 0 || value[0] != '{' {
			return board.ErrShape
		}
	}
	return nil
}

type operatorSketchRequest struct {
	Scope    SketchSceneScope
	Lifetime <-chan struct{}
	Envelope operatorSketchEnvelope
}
type operatorSketchTarget struct {
	Scope    SketchSceneScope
	Lifetime <-chan struct{}
	Active   bool
}
type operatorSketchRuntimeOptions struct {
	Caller    SketchSceneCaller
	Board     *board.Board
	Service   *board.SketchService
	Authority func(epoch, generation uint64) (board.SketchAuthority, error)
	Exclude   func(frameID, session string) error
	AssetRead func(context.Context, board.SketchAuthority, string, string, string, int, int) ([]byte, error)
	AssetURL  func(context.Context, string, string) error
	// Go tracks asynchronous writer work in Chrome's existing shutdown group.
	Go func(func())
}
type operatorSketchReply struct {
	Request string `json:"request"`
	OK      bool   `json:"ok"`
	Value   any    `json:"value,omitempty"`
	Error   string `json:"error,omitempty"`
}
type operatorSketchCompletion struct {
	state     *operatorSketchContext
	reply     operatorSketchReply
	authority board.SketchAuthority
	beginKey  string
}
type operatorSketchContext struct {
	scope    SketchSceneScope
	lifetime <-chan struct{}
	active   bool
	scene    *SketchSceneSession
	prepared *operatorSketchPreparedSource
	original map[string]board.SketchAuthority
	begin    map[string]board.SketchAuthority
	job      *operatorSketchSceneJob
	cursor   board.SketchDeliveryCursor
	delivery *operatorSketchJournalTransfer
	retryAt  time.Time
	retired  bool
	replies  []operatorSketchReply
}
type operatorSketchPreparedSource struct {
	sketch    string
	source    board.SketchSourceDescriptor
	authority board.SketchAuthority
}
type operatorSketchSceneJob struct {
	request   string
	sketch    string
	source    board.SketchSourceDescriptor
	authority board.SketchAuthority
	stepper   *SketchSceneTransfer
	stage     int
	assets    []SketchSceneAsset
	asset     int
	handle    string
}
type operatorSketchJournalTransfer struct {
	raw           []byte
	handle, epoch string
	seq           uint64
	offset        int
	begun         bool
}
type operatorSketchRuntime struct {
	opts        operatorSketchRuntimeOptions
	requests    chan operatorSketchRequest
	completions chan operatorSketchCompletion
	wake        chan struct{}
	states      map[operatorContextID]*operatorSketchContext
	order       []operatorContextID
	next        int
	jobs        int
	serial      uint64
	ctx         context.Context
	cancel      context.CancelFunc
	closed      atomic.Bool
	work        sync.WaitGroup
	frames      sync.Map
}

func newOperatorSketchRuntime(opts operatorSketchRuntimeOptions) *operatorSketchRuntime {
	ctx, cancel := context.WithCancel(context.Background())
	return &operatorSketchRuntime{opts: opts, requests: make(chan operatorSketchRequest, 64), completions: make(chan operatorSketchCompletion, 4), wake: make(chan struct{}, 1), states: map[operatorContextID]*operatorSketchContext{}, ctx: ctx, cancel: cancel}
}
func (r *operatorSketchRuntime) signal() {
	select {
	case r.wake <- struct{}{}:
	default:
	}
}
func (r *operatorSketchRuntime) Wake() <-chan struct{} { return r.wake }
func (r *operatorSketchRuntime) Enqueue(q operatorSketchRequest) bool {
	if r.closed.Load() {
		return false
	}
	q.Envelope.Body = bytes.Clone(q.Envelope.Body)
	select {
	case r.requests <- q:
		r.signal()
		return true
	default:
		return false
	}
}
func operatorSketchAlive(life <-chan struct{}) bool {
	if life == nil {
		return false
	}
	select {
	case <-life:
		return false
	default:
		return true
	}
}
func (r *operatorSketchRuntime) context(q operatorSketchRequest) *operatorSketchContext {
	if !operatorSketchAlive(q.Lifetime) || q.Scope.ParentContext <= 0 || q.Scope.ParentSession == "" || q.Scope.Loader == "" {
		return nil
	}
	key := operatorContextID{q.Scope.ParentSession, q.Scope.ParentContext}
	state := r.states[key]
	if state != nil && (state.scope != q.Scope || state.retired) {
		return nil
	}
	if state == nil {
		if len(r.states) >= 4 {
			return nil
		}
		state = &operatorSketchContext{scope: q.Scope, lifetime: q.Lifetime, original: map[string]board.SketchAuthority{}, begin: map[string]board.SketchAuthority{}}
		r.states[key] = state
		r.order = append(r.order, key)
	}
	if state.lifetime != q.Lifetime {
		return nil
	}
	return state
}

// Reconcile runs on the existing worker. Closing a lifetime immediately fences
// every in-flight CDP/read/write; cleanup follows here without reinstallation.
func (r *operatorSketchRuntime) Reconcile(targets []operatorSketchTarget) {
	live := map[operatorContextID]operatorSketchTarget{}
	for _, target := range targets {
		live[operatorContextID{target.Scope.ParentSession, target.Scope.ParentContext}] = target
	}
	changed := false
	// Release disposable inactive caches before active-first admission. Original
	// operation authority stays attached to a retained live document context.
	for key, state := range r.states {
		target, present := live[key]
		if !present || !operatorSketchAlive(state.lifetime) || target.Scope != state.scope || target.Lifetime != state.lifetime {
			state.retired = true
			changed = true
			if state.scene != nil {
				state.scene.Retire()
			}
			continue
		}
		if !target.Active {
			state.delivery = nil
			state.cursor = board.SketchDeliveryCursor{}
			if state.scene != nil {
				state.scene.Retire()
			}
		}
		if !target.Active && state.scene == nil && state.job == nil && len(state.original) == 0 && len(state.begin) == 0 && len(state.replies) == 0 {
			r.removeContext(key)
			changed = true
		}
	}
	for _, active := range []bool{true, false} {
		for _, target := range targets {
			if target.Active != active {
				continue
			}
			state := r.context(operatorSketchRequest{Scope: target.Scope, Lifetime: target.Lifetime})
			if state != nil {
				if state.active != active {
					changed = true
				}
				state.active = active
			}
		}
	}
	if changed {
		r.signal()
	}
}
func (r *operatorSketchRuntime) removeContext(key operatorContextID) {
	delete(r.states, key)
	for i, k := range r.order {
		if k == key {
			r.order = append(r.order[:i], r.order[i+1:]...)
			if r.next > i {
				r.next--
			}
			break
		}
	}
}

// RetireFrame is safe on Chrome's event goroutine. Atomic retirement happens
// before queued worker cleanup and never uploads into a replacement child.
func (r *operatorSketchRuntime) RetireFrame(frameID string) {
	if scene, ok := r.frames.Load(frameID); ok {
		scene.(*SketchSceneSession).Retire()
		r.signal()
	}
}
func (r *operatorSketchRuntime) disposeScene(ctx context.Context, state *operatorSketchContext) error {
	if state.scene == nil {
		return nil
	}
	scene := state.scene
	r.frames.Delete(scene.FrameID())
	state.scene = nil
	state.prepared = nil
	state.job = nil
	return scene.Dispose(ctx)
}
func (r *operatorSketchRuntime) queue(state *operatorSketchContext, reply operatorSketchReply) {
	if operatorSketchAlive(state.lifetime) && len(state.replies) < 64 {
		state.replies = append(state.replies, reply)
		r.signal()
	}
}
func (r *operatorSketchRuntime) fail(state *operatorSketchContext, request string, err error) {
	r.queue(state, operatorSketchReply{Request: request, Error: err.Error()})
}
func (r *operatorSketchRuntime) authority(e operatorSketchEnvelope) (board.SketchAuthority, error) {
	if r.opts.Authority == nil {
		return board.SketchAuthority{}, errors.New("sketch authority unavailable")
	}
	n, err := strconv.ParseUint(e.Generation, 10, 64)
	if err != nil {
		return board.SketchAuthority{}, board.ErrShape
	}
	return r.opts.Authority(e.Epoch, n)
}
func (r *operatorSketchRuntime) handle(ctx context.Context, q operatorSketchRequest) {
	state := r.context(q)
	if state == nil {
		key := operatorContextID{q.Scope.ParentSession, q.Scope.ParentContext}
		if operatorSketchAlive(q.Lifetime) && r.states[key] == nil && len(r.states) >= 4 {
			temporary := &operatorSketchContext{scope: q.Scope, lifetime: q.Lifetime}
			_ = r.evaluateParent(ctx, temporary, "__couchcastSketchReply", operatorSketchReply{Request: q.Envelope.Request, Error: "Four sketch contexts are already held."}, nil)
		}
		return
	}
	e := q.Envelope
	// Revalidate the typed envelope so alternate internal callers cannot bypass
	// the strict binding boundary or disagree about epoch.
	raw, _ := json.Marshal(e)
	if _, err := parseOperatorSketchEnvelope(raw); err != nil {
		r.fail(state, e.Request, err)
		return
	}
	if strings.HasPrefix(e.Kind, "sketch-") {
		r.command(state, e)
		return
	}
	var authority board.SketchAuthority
	var err error
	if e.Kind != "scene-dispose" {
		authority, err = r.authority(e)
		if err != nil {
			r.fail(state, e.Request, err)
			return
		}
	}
	switch e.Kind {
	case "scene-open":
		if err = authority.Annotation.Check(); err != nil {
			break
		}
		if authority.Check == nil {
			err = board.ErrDisabled
			break
		}
		if err = authority.Check(); err != nil {
			break
		}
		if err = strictjson.Decode(e.Body, &struct{}{}); err != nil {
			break
		}
		if state.scene != nil {
			err = errors.New("private scene already opened")
			break
		}
		state.scene = NewSketchSceneSession(r.opts.Caller, state.scope, func(scope SketchSceneScope) bool {
			return scope == state.scope && state.active && !state.retired && operatorSketchAlive(state.lifetime) && !r.closed.Load()
		}, func(frameID, session string) error {
			if r.opts.Exclude == nil {
				return errors.New("private frame exclusion unavailable")
			}
			if err := r.opts.Exclude(frameID, session); err != nil {
				return err
			}
			r.frames.Store(frameID, state.scene)
			return nil
		})
		if r.opts.AssetURL != nil {
			state.scene.SetAssetURLObserver(func(url string) error { return r.opts.AssetURL(r.ctx, state.scene.FrameID(), url) })
		}
		err = state.scene.Open(ctx, "globalThis.__couchcastPrivateSceneFrame")
		if err == nil {
			r.queue(state, operatorSketchReply{Request: e.Request, OK: true, Value: map[string]bool{"open": true}})
		}
	case "scene-prepare":
		if state.scene == nil || state.job != nil {
			err = errors.New("private scene unavailable")
			break
		}
		var body struct {
			Sketch string                       `json:"sketch"`
			Source board.SketchSourceDescriptor `json:"source"`
		}
		if err = strictjson.Decode(e.Body, &body); err != nil {
			break
		}
		if original, ok := state.original[body.Sketch]; ok {
			authority = original
		}
		if r.opts.Service == nil {
			err = errors.New("sketch service unavailable")
			break
		}
		var length int
		length, err = r.opts.Service.SourceLength(authority, body.Sketch, body.Source.Handle)
		if err != nil {
			break
		}
		if length != body.Source.Length || length <= 0 || length > recording.SketchSourcePrefixLimit {
			err = board.ErrSketchDeliveryLimit
			break
		}
		state.prepared = nil
		state.job = &operatorSketchSceneJob{request: e.Request, sketch: body.Sketch, source: body.Source, authority: authority}
		r.signal()
	case "scene-hit":
		var body struct {
			Point [2]float64 `json:"point"`
		}
		if err = strictjson.Decode(e.Body, &body); err != nil {
			break
		}
		if state.scene == nil || state.job != nil {
			err = errors.New("private scene unavailable")
			break
		}
		if err = r.checkPreparedSource(state); err != nil {
			state.scene.Retire()
			break
		}
		var value SketchSceneHit
		value, err = state.scene.HitTest(ctx, body.Point)
		if err == nil {
			r.queue(state, operatorSketchReply{Request: e.Request, OK: true, Value: value})
		}
	case "scene-placement":
		var body struct {
			Path json.RawMessage `json:"path"`
		}
		if err = strictjson.Decode(e.Body, &body); err != nil {
			break
		}
		if len(body.Path) == 0 || state.scene == nil || state.job != nil {
			err = errors.New("private scene unavailable")
			break
		}
		if err = r.checkPreparedSource(state); err != nil {
			state.scene.Retire()
			break
		}
		var value *recording.Box
		value, err = state.scene.Placement(ctx, body.Path)
		if err == nil {
			r.queue(state, operatorSketchReply{Request: e.Request, OK: true, Value: value})
		}
	case "scene-dispose":
		if err = strictjson.Decode(e.Body, &struct{}{}); err != nil {
			break
		}
		state.job = nil
		if state.scene != nil {
			err = r.disposeScene(ctx, state)
		}
		if err == nil {
			r.queue(state, operatorSketchReply{Request: e.Request, OK: true, Value: map[string]bool{"disposed": true}})
		}
	}
	if err != nil {
		r.fail(state, e.Request, err)
	}
}
func (r *operatorSketchRuntime) command(state *operatorSketchContext, e operatorSketchEnvelope) {
	command, err := board.ParseSketchCommand(e.Kind, e.Body)
	if err != nil || command.Epoch != e.Epoch {
		r.fail(state, e.Request, board.ErrShape)
		return
	}
	if e.Kind == "sketch-source" || e.Kind == "sketch-asset" {
		r.fail(state, e.Request, errors.New("scene transfer is private"))
		return
	}
	if r.opts.Service == nil || r.jobs >= 4 {
		r.fail(state, e.Request, errors.New("sketch command queue busy"))
		return
	}
	authority, err := r.authority(e)
	if old, ok := state.original[command.Sketch]; ok {
		authority, err = old, nil
	}
	beginKey := fmt.Sprintf("%d:%s", command.Epoch, command.Operation)
	if command.Sketch == "" {
		if old, ok := state.begin[beginKey]; ok {
			authority, err = old, nil
		}
	}
	if err != nil {
		r.fail(state, e.Request, err)
		return
	}
	if command.Kind == "sketch-begin" && state.scene == nil {
		r.fail(state, e.Request, errors.New("private scene preflight missing"))
		return
	}
	if command.Kind == "sketch-begin" {
		if len(state.begin) >= recording.SketchMaxCount {
			r.fail(state, e.Request, board.ErrSketchDeliveryLimit)
			return
		}
	}
	capturedScene := state.scene
	drawing := command.Kind != "sketch-label" && command.Kind != "sketch-status"
	baseCheck, baseMetadata := authority.Check, authority.MetadataCheck
	if baseCheck != nil {
		authority.Check = func() error {
			if drawing && (capturedScene == nil || capturedScene.retired.Load()) {
				return ErrSketchSceneRetired
			}
			if !operatorSketchAlive(state.lifetime) || r.closed.Load() {
				return ErrSketchSceneRetired
			}
			if baseCheck != nil {
				return baseCheck()
			}
			return nil
		}
	}
	authority.MetadataCheck = func() error {
		if !operatorSketchAlive(state.lifetime) || r.closed.Load() {
			return ErrSketchSceneRetired
		}
		if baseMetadata != nil {
			return baseMetadata()
		}
		return nil
	}
	if command.Kind == "sketch-begin" {
		state.begin[beginKey] = authority
	}
	r.jobs++
	run := func() {
		ctx, cancel := context.WithTimeout(r.ctx, callTimeout)
		defer cancel()
		var reply board.SketchReply
		if command.Kind == "sketch-status" {
			reply = r.opts.Service.Status(ctx, authority, command)
		} else {
			reply = r.opts.Service.Execute(ctx, authority, command)
		}
		completion := operatorSketchCompletion{state: state, authority: authority, reply: operatorSketchReply{Request: e.Request, OK: true, Value: reply}}
		if command.Kind == "sketch-begin" {
			completion.beginKey = beginKey
		}
		select {
		case r.completions <- completion:
			r.signal()
		case <-r.ctx.Done():
		}
	}
	r.work.Add(1)
	tracked := func() { defer r.work.Done(); run() }
	if r.opts.Go != nil {
		r.opts.Go(tracked)
	} else {
		go tracked()
	}
}
func (r *operatorSketchRuntime) finishCompletion(c operatorSketchCompletion) {
	r.jobs--
	if reply, ok := c.reply.Value.(board.SketchReply); ok && c.beginKey != "" && reply.Outcome == board.SketchRefused {
		delete(c.state.begin, c.beginKey)
	}
	if reply, ok := c.reply.Value.(board.SketchReply); ok && reply.Sketch != "" && reply.Outcome == board.SketchAccepted && len(c.state.original) < recording.SketchMaxCount {
		if _, exists := c.state.original[reply.Sketch]; !exists {
			c.state.original[reply.Sketch] = c.authority
		}
	}
	r.queue(c.state, c.reply)
}
func (r *operatorSketchRuntime) sceneTurn(ctx context.Context, state *operatorSketchContext) {
	job := state.job
	scene := state.scene
	var err error
	switch job.stage {
	case 0:
		job.stepper, err = scene.BeginTransfer(ctx, job.source.Handle, job.source.Length, func(offset, length int) ([]byte, error) {
			return r.opts.Service.ReadSource(job.authority, job.sketch, job.source.Handle, offset, length)
		})
		if err == nil {
			job.stage = 1
		}
	case 1, 4:
		var done bool
		done, err = job.stepper.Step(ctx)
		if err == nil && done {
			job.stage++
		}
	case 2:
		job.assets, err = scene.FinishSource(ctx, job.sketch, job.source)
		if err == nil {
			job.stage = 3
		}
	case 3:
		if job.asset == len(job.assets) {
			job.stage = 6
			break
		}
		asset := job.assets[job.asset]
		if r.opts.AssetRead == nil {
			err = errors.New("private asset reader unavailable")
			break
		}
		if err = r.opts.Service.AdmitSourceAsset(job.authority, job.sketch, job.source.Handle, asset.Hash); err != nil {
			break
		}
		r.serial++
		job.handle = fmt.Sprintf("asset-%d", r.serial)
		job.stepper, err = scene.BeginAssetTransfer(ctx, job.handle, int(asset.Size), func(offset, length int) ([]byte, error) {
			if err := r.opts.Service.AdmitSourceAsset(job.authority, job.sketch, job.source.Handle, asset.Hash); err != nil {
				return nil, err
			}
			return r.opts.AssetRead(r.ctx, job.authority, job.sketch, job.source.Handle, asset.Hash, offset, length)
		})
		if err == nil {
			job.stage = 4
		}
	case 5:
		err = scene.FinishAsset(ctx, job.handle, job.assets[job.asset])
		if err == nil {
			job.asset++
			job.stage = 3
		}
	case 6:
		if _, err = r.opts.Service.SourceLength(job.authority, job.sketch, job.source.Handle); err != nil {
			break
		}
		var ready SketchSceneReady
		ready, err = scene.Render(ctx)
		if err == nil {
			state.prepared = &operatorSketchPreparedSource{sketch: job.sketch, source: job.source, authority: job.authority}
			r.queue(state, operatorSketchReply{Request: job.request, OK: true, Value: ready})
			state.job = nil
		}
	}
	if err != nil {
		scene.Retire()
		state.job = nil
		r.fail(state, job.request, err)
	}
}
func (r *operatorSketchRuntime) checkPreparedSource(state *operatorSketchContext) error {
	if state.prepared == nil || r.opts.Service == nil {
		return errors.New("private source not prepared")
	}
	length, err := r.opts.Service.SourceLength(state.prepared.authority, state.prepared.sketch, state.prepared.source.Handle)
	if err != nil {
		return err
	}
	if length != state.prepared.source.Length {
		return board.ErrStale
	}
	return nil
}
func (r *operatorSketchRuntime) evaluateParent(ctx context.Context, state *operatorSketchContext, hook string, value any, out any) error {
	if !operatorSketchAlive(state.lifetime) {
		return ErrSketchSceneRetired
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var result struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		Exception json.RawMessage `json:"exceptionDetails"`
	}
	callCtx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	err = r.opts.Caller.Call(callCtx, state.scope.ParentSession, "Runtime.evaluate", map[string]any{"contextId": state.scope.ParentContext, "expression": "globalThis." + hook + "?.(" + string(raw) + ")", "returnByValue": true, "awaitPromise": true, "allowUnsafeEvalBlockedByCSP": false}, &result)
	if !operatorSketchAlive(state.lifetime) {
		return ErrSketchSceneRetired
	}
	if err != nil {
		return err
	}
	if len(result.Exception) > 0 {
		return errors.New("sketch parent delivery refused")
	}
	if out != nil {
		return json.Unmarshal(result.Result.Value, out)
	}
	return nil
}
func (r *operatorSketchRuntime) journalTurn(ctx context.Context, state *operatorSketchContext) bool {
	if r.opts.Board == nil || !state.active || time.Now().Before(state.retryAt) {
		return false
	}
	if state.delivery == nil {
		if state.cursor == r.opts.Board.OperatorSketchCursor() {
			return false
		}
		delivery, err := r.opts.Board.OperatorSketchSince(state.cursor)
		if err != nil {
			return false
		}
		raw, err := json.Marshal(delivery)
		if err != nil {
			return false
		}
		var epoch string
		var seq uint64
		if delivery.Bootstrap != nil {
			epoch, seq = delivery.Bootstrap.Epoch, delivery.Bootstrap.Seq
		} else if delivery.Batch != nil {
			epoch, seq = delivery.Batch.Epoch, delivery.Batch.To
		} else {
			return false
		}
		r.serial++
		state.delivery = &operatorSketchJournalTransfer{raw: raw, handle: fmt.Sprintf("journal-%d", r.serial), epoch: epoch, seq: seq}
	}
	d := state.delivery
	if !d.begun {
		var ack struct {
			Offset int `json:"offset"`
		}
		err := r.evaluateParent(ctx, state, "__couchcastSketchDelivery", map[string]any{"phase": "begin", "handle": d.handle, "epoch": d.epoch, "total": len(d.raw)}, &ack)
		if err == nil && ack.Offset == 0 {
			d.begun = true
			return true
		}
		state.retryAt = time.Now().Add(250 * time.Millisecond)
		return false
	}
	length := min(board.MaxSketchTransfer, len(d.raw)-d.offset)
	payload := map[string]any{"phase": "chunk", "handle": d.handle, "epoch": d.epoch, "total": len(d.raw), "offset": d.offset, "base64": base64.StdEncoding.EncodeToString(d.raw[d.offset : d.offset+length])}
	var ack struct {
		Offset        int    `json:"offset"`
		Epoch         string `json:"epoch"`
		Seq           uint64 `json:"seq"`
		NeedBootstrap bool   `json:"needBootstrap"`
	}
	if err := r.evaluateParent(ctx, state, "__couchcastSketchDelivery", payload, &ack); err != nil {
		state.retryAt = time.Now().Add(250 * time.Millisecond)
		return false
	}
	if ack.Offset != d.offset+length {
		state.retryAt = time.Now().Add(250 * time.Millisecond)
		return false
	}
	if ack.Offset == len(d.raw) {
		if ack.NeedBootstrap {
			state.cursor = board.SketchDeliveryCursor{}
			state.delivery = nil
		} else if ack.Epoch == d.epoch && ack.Seq == d.seq {
			state.cursor = board.SketchDeliveryCursor{Epoch: d.epoch, Seq: d.seq}
			state.delivery = nil
		} else {
			state.retryAt = time.Now().Add(250 * time.Millisecond)
			return false
		}
	} else {
		d.offset = ack.Offset
	}
	return true
}

// Turn performs one request, completion, reply, transfer step or journal chunk.
// The root select loop calls it once, then returns to ordinary operator work.
func (r *operatorSketchRuntime) Turn(ctx context.Context) bool {
	if r.closed.Load() {
		return false
	}
	// A child navigation is fenced atomically on the event goroutine; retire
	// its resources before any queued command can try to use that scene.
	for _, key := range r.order {
		state := r.states[key]
		if state != nil && state.scene != nil && state.scene.retired.Load() {
			if state.job != nil {
				r.fail(state, state.job.request, ErrSketchSceneRetired)
			}
			_ = r.disposeScene(ctx, state)
			r.signal()
			return true
		}
	}
	select {
	case completion := <-r.completions:
		r.finishCompletion(completion)
		r.signal()
		return true
	default:
	}
	select {
	case request := <-r.requests:
		r.handle(ctx, request)
		r.signal()
		return true
	default:
	}
	for i := 0; i < len(r.order); i++ {
		r.next %= len(r.order)
		key := r.order[r.next]
		r.next++
		state := r.states[key]
		if state == nil {
			continue
		}
		if state.retired || !operatorSketchAlive(state.lifetime) {
			_ = r.disposeScene(ctx, state)
			r.removeContext(key)
			return true
		}
		if len(state.replies) > 0 {
			reply := state.replies[0]
			_ = r.evaluateParent(ctx, state, "__couchcastSketchReply", reply, nil)
			state.replies = state.replies[1:]
			r.signal()
			return true
		}
		if state.job != nil {
			r.sceneTurn(ctx, state)
			r.signal()
			return true
		}
		if r.journalTurn(ctx, state) {
			r.signal()
			return true
		}
	}
	return false
}
func (r *operatorSketchRuntime) Close(ctx context.Context) {
	if r.closed.Swap(true) {
		return
	}
	r.cancel()
	for _, state := range r.states {
		_ = r.disposeScene(ctx, state)
	}
	r.states = map[operatorContextID]*operatorSketchContext{}
}

// Pending asks the existing retry timer to revisit uncertain deliveries without
// spinning the worker on a missing hook or a refused ACK.
func (r *operatorSketchRuntime) Pending() bool {
	if r.jobs > 0 || len(r.requests) > 0 || len(r.completions) > 0 {
		return true
	}
	for _, state := range r.states {
		if state.job != nil || state.delivery != nil || len(state.replies) > 0 {
			return true
		}
	}
	return false
}
