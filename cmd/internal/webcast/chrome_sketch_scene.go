package webcast

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

var ErrSketchSceneRetired = errors.New("webcast: private sketch scene retired")

type SketchSceneCaller interface {
	Call(context.Context, string, string, any, any) error
}
type SketchSceneScope struct {
	ParentSession string
	ParentContext int64
	Loader        string
}
type SketchSceneAsset struct {
	Hash string `json:"hash"`
	MIME string `json:"mime"`
	Size int64  `json:"size"`
}
type SketchSceneReady struct {
	Ready  bool               `json:"ready"`
	Doc    int                `json:"doc"`
	Tab    string             `json:"tab"`
	Time   int64              `json:"time"`
	VP     recording.Viewport `json:"vp"`
	Scroll [2]float64         `json:"scroll"`
}
type SketchSceneHit struct {
	Node recording.NodeID `json:"node"`
	Box  recording.Box    `json:"box"`
}

// SketchSceneSession belongs to the existing single CDP delivery worker. The
// atomic retirement flag also fences cancellation/context retirement from its
// owner. Neither a late response nor a page navigation creates a new world.
type SketchSceneSession struct {
	caller         SketchSceneCaller
	scope          SketchSceneScope
	current        func(SketchSceneScope) bool
	exclude        func(string, string) error
	retired        atomic.Bool
	attempted      bool
	session, frame string
	world          int64
	attached       bool
	assetURL       func(string) error
}

func NewSketchSceneSession(caller SketchSceneCaller, scope SketchSceneScope, current func(SketchSceneScope) bool, exclude func(frameID, session string) error) *SketchSceneSession {
	return &SketchSceneSession{caller: caller, scope: scope, current: current, exclude: exclude}
}
func SketchSceneShellURL() string {
	return "data:text/html;couch-sketch=1," + url.PathEscape(`<!doctype html><title>Couch sketch scene</title><style>html,body{margin:0;overflow:hidden}iframe{border:0;display:block}</style><iframe id="page" sandbox="allow-same-origin" title="Recorded moment"></iframe>`)
}
func (s *SketchSceneSession) check() error {
	if s.retired.Load() || s.current == nil || !s.current(s.scope) {
		s.retired.Store(true)
		return ErrSketchSceneRetired
	}
	return nil
}
func (s *SketchSceneSession) call(ctx context.Context, session, method string, params, result any) error {
	if err := s.check(); err != nil {
		return err
	}
	callCtx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	err := s.caller.Call(callCtx, session, method, params, result)
	if stale := s.check(); stale != nil {
		return stale
	}
	return err
}
func (s *SketchSceneSession) evaluate(ctx context.Context, session string, world int64, expression string, out any) error {
	var result struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		Exception json.RawMessage `json:"exceptionDetails"`
	}
	if err := s.call(ctx, session, "Runtime.evaluate", map[string]any{"expression": expression, "contextId": world, "returnByValue": true, "awaitPromise": true, "allowUnsafeEvalBlockedByCSP": false}, &result); err != nil {
		s.Retire()
		return err
	}
	if len(result.Exception) > 0 {
		return errors.New("private sketch scene execution refused")
	}
	if out != nil {
		if len(result.Result.Value) == 0 {
			return errors.New("private sketch scene response missing")
		}
		return json.Unmarshal(result.Result.Value, out)
	}
	return nil
}

// Open expects a trusted isolated-parent expression yielding the actual iframe
// object, not a page selector or name. Root must exclude its frame/session from
// recording/normal input before this method uploads any private module/data.
func (s *SketchSceneSession) Open(ctx context.Context, parentFrameExpression string) (err error) {
	if err = s.check(); err != nil {
		return err
	}
	if s.attempted || s.caller == nil || s.exclude == nil || s.scope.ParentSession == "" || s.scope.ParentContext <= 0 || s.scope.Loader == "" || len(parentFrameExpression) > 16<<10 {
		return errors.New("invalid private sketch scene scope")
	}
	s.attempted = true
	defer func() {
		if err != nil {
			s.retired.Store(true)
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), callTimeout)
			defer cancel()
			_ = s.Dispose(cleanupCtx)
		}
	}()
	var object struct {
		Result struct {
			ObjectID string `json:"objectId"`
		} `json:"result"`
		Exception json.RawMessage `json:"exceptionDetails"`
	}
	if err = s.call(ctx, s.scope.ParentSession, "Runtime.evaluate", map[string]any{"contextId": s.scope.ParentContext, "expression": parentFrameExpression, "awaitPromise": true, "allowUnsafeEvalBlockedByCSP": false}, &object); err != nil {
		return err
	}
	if object.Result.ObjectID == "" || len(object.Exception) > 0 {
		return errors.New("private scene frame is unavailable")
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), callTimeout)
		defer cancel()
		_ = s.caller.Call(releaseCtx, s.scope.ParentSession, "Runtime.releaseObject", map[string]any{"objectId": object.Result.ObjectID}, nil)
	}()
	var described struct {
		Node struct {
			FrameID    string   `json:"frameId"`
			Name       string   `json:"localName"`
			Attributes []string `json:"attributes"`
		} `json:"node"`
	}
	if err = s.call(ctx, s.scope.ParentSession, "DOM.describeNode", map[string]any{"objectId": object.Result.ObjectID}, &described); err != nil {
		return err
	}
	attrs := map[string]string{}
	for i := 0; i+1 < len(described.Node.Attributes); i += 2 {
		attrs[described.Node.Attributes[i]] = described.Node.Attributes[i+1]
	}
	if described.Node.FrameID == "" || described.Node.Name != "iframe" || strings.TrimSpace(attrs["sandbox"]) != "allow-same-origin" {
		return errors.New("private scene requires an opaque unscripted sandbox frame")
	}
	s.frame = described.Node.FrameID
	s.session = s.scope.ParentSession
	if err = s.exclude(s.frame, s.session); err != nil {
		return err
	}
	var world struct {
		Context int64 `json:"executionContextId"`
	}
	params := map[string]any{"frameId": s.frame, "worldName": "couch-private-sketch"}
	err = s.call(ctx, s.session, "Page.createIsolatedWorld", params, &world)
	if err != nil {
		if errors.Is(err, ErrSketchSceneRetired) {
			return err
		}
		var targets struct {
			Infos []struct{ TargetID, Type string } `json:"targetInfos"`
		}
		if err = s.call(ctx, "", "Target.getTargets", map[string]any{}, &targets); err != nil {
			return err
		}
		found := false
		for _, target := range targets.Infos {
			if target.TargetID == s.frame && target.Type == "iframe" {
				found = true
				break
			}
		}
		if !found {
			return errors.New("private scene frame session unavailable")
		}
		var attached struct {
			Session string `json:"sessionId"`
		}
		if err = s.call(ctx, "", "Target.attachToTarget", map[string]any{"targetId": s.frame, "flatten": true}, &attached); err != nil {
			return err
		}
		if attached.Session == "" {
			return errors.New("private frame attachment missing")
		}
		s.session = attached.Session
		s.attached = true
		if err = s.exclude(s.frame, s.session); err != nil {
			return err
		}
		if err = s.call(ctx, s.session, "Page.createIsolatedWorld", params, &world); err != nil {
			return err
		}
	}
	if world.Context <= 0 {
		return errors.New("private child context missing")
	}
	s.world = world.Context
	var opaque bool
	if err = s.evaluate(ctx, s.session, s.world, `location.origin==='null'&&document.title==='Couch sketch scene'&&!!document.getElementById('page')`, &opaque); err != nil {
		return err
	}
	if !opaque {
		return errors.New("private scene did not retain its opaque shell")
	}
	script, e := buildSketchSceneScript()
	if e != nil {
		return e
	}
	return s.installModule(ctx, script)
}
func (s *SketchSceneSession) installModule(ctx context.Context, script string) error {
	token := fmt.Sprintf("scene-%d", s.world)
	quoted, _ := json.Marshal(token)
	if err := s.evaluate(ctx, s.session, s.world, `globalThis.__couchcastSketchModule={token:`+string(quoted)+`,offset:0,parts:[]};true`, nil); err != nil {
		return err
	}
	for offset := 0; offset < len(script); offset += recording.SketchSourceChunkSize {
		end := min(len(script), offset+recording.SketchSourceChunkSize)
		part := base64.StdEncoding.EncodeToString([]byte(script[offset:end]))
		partJSON, _ := json.Marshal(part)
		expression := fmt.Sprintf(`(()=>{const m=globalThis.__couchcastSketchModule;if(!m||m.token!==%s||m.offset!==%d)throw Error('retired module transfer');const part=atob(%s);m.parts.push(part);m.offset+=part.length;return true;})()`, quoted, offset, partJSON)
		if err := s.evaluate(ctx, s.session, s.world, expression, nil); err != nil {
			return err
		}
	}
	expression := `(()=>{const m=globalThis.__couchcastSketchModule;delete globalThis.__couchcastSketchModule;const binary=m.parts.join('');const bytes=Uint8Array.from(binary,c=>c.charCodeAt(0));(0,eval)(new TextDecoder('utf-8',{fatal:true}).decode(bytes));return !!globalThis.__couchcastSketchScene;})()`
	var installed bool
	if err := s.evaluate(ctx, s.session, s.world, expression, &installed); err != nil {
		return err
	}
	if !installed {
		return errors.New("private scene module installation missing")
	}
	return nil
}
func (s *SketchSceneSession) invoke(ctx context.Context, method string, args any, out any) error {
	if s.world <= 0 {
		return ErrSketchSceneRetired
	}
	encoded, err := json.Marshal(args)
	if err != nil {
		return err
	}
	return s.evaluate(ctx, s.session, s.world, `globalThis.__couchcastSketchScene.`+method+`(...`+string(encoded)+`)`, out)
}

// SketchSceneTransfer is advanced by the existing delivery worker one Step
// per turn. Each step performs at most one bounded read and one CDP call.
type SketchSceneTransfer struct {
	scene         *SketchSceneSession
	handle        string
	total, offset int
	read          func(int, int) ([]byte, error)
}

// SketchSceneAssetLimit matches the recorder fallback resource capture envelope.
const SketchSceneAssetLimit = 32 << 20

func (s *SketchSceneSession) BeginTransfer(ctx context.Context, handle string, total int, read func(int, int) ([]byte, error)) (*SketchSceneTransfer, error) {
	return s.beginTransfer(ctx, handle, total, "source", recording.SketchSourcePrefixLimit, read)
}

func (s *SketchSceneSession) BeginAssetTransfer(ctx context.Context, handle string, total int, read func(int, int) ([]byte, error)) (*SketchSceneTransfer, error) {
	return s.beginTransfer(ctx, handle, total, "asset", SketchSceneAssetLimit, read)
}

func (s *SketchSceneSession) beginTransfer(ctx context.Context, handle string, total int, mode string, limit int, read func(int, int) ([]byte, error)) (*SketchSceneTransfer, error) {
	if total < 0 || total > limit || read == nil {
		return nil, board.ErrSketchDeliveryLimit
	}
	if err := s.invoke(ctx, "beginTransfer", []any{handle, total, mode}, nil); err != nil {
		return nil, err
	}
	return &SketchSceneTransfer{scene: s, handle: handle, total: total, read: read}, nil
}
func (t *SketchSceneTransfer) Step(ctx context.Context) (bool, error) {
	if err := t.scene.check(); err != nil {
		return false, err
	}
	if t.offset == t.total {
		return true, nil
	}
	length := min(recording.SketchSourceChunkSize, t.total-t.offset)
	part, err := t.read(t.offset, length)
	if err != nil {
		return false, err
	}
	if len(part) != length {
		return false, errors.New("private scene transfer length mismatch")
	}
	var ack struct {
		Offset int `json:"offset"`
		Total  int `json:"total"`
	}
	if err = t.scene.invoke(ctx, "appendTransfer", []any{t.handle, t.offset, base64.StdEncoding.EncodeToString(part)}, &ack); err != nil {
		return false, err
	}
	if ack.Offset != t.offset+length || ack.Total != t.total {
		return false, errors.New("private scene transfer acknowledgment mismatch")
	}
	t.offset += length
	return t.offset == t.total, nil
}
func (s *SketchSceneSession) transfer(ctx context.Context, handle string, total int, read func(int, int) ([]byte, error)) error {
	stepper, err := s.BeginTransfer(ctx, handle, total, read)
	if err != nil {
		return err
	}
	for {
		done, err := stepper.Step(ctx)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
}
func (s *SketchSceneSession) FinishSource(ctx context.Context, sketch string, source board.SketchSourceDescriptor) ([]SketchSceneAsset, error) {
	descriptor := map[string]any{"sketch": sketch, "time": source.Time, "doc": source.Doc, "tab": source.Tab, "length": source.Length}
	var result struct {
		Assets []SketchSceneAsset `json:"assets"`
	}
	if err := s.invoke(ctx, "prepareSource", []any{source.Handle, descriptor}, &result); err != nil {
		return nil, err
	}
	return result.Assets, nil
}

// FrameID and the observer are private Go-side recorder integration seams.
func (s *SketchSceneSession) FrameID() string                                 { return s.frame }
func (s *SketchSceneSession) SetAssetURLObserver(observer func(string) error) { s.assetURL = observer }
func (s *SketchSceneSession) FinishAsset(ctx context.Context, handle string, asset SketchSceneAsset) error {
	var result struct {
		URL string `json:"url"`
	}
	if err := s.invoke(ctx, "installAsset", []any{handle, asset.Hash, asset.MIME}, &result); err != nil {
		return err
	}
	if !strings.HasPrefix(result.URL, "blob:null/") {
		s.Retire()
		return errors.New("private scene asset origin refused")
	}
	if s.assetURL != nil {
		if err := s.assetURL(result.URL); err != nil {
			s.Retire()
			return err
		}
	}
	return nil
}

func (s *SketchSceneSession) Prepare(ctx context.Context, sketch string, source board.SketchSourceDescriptor, read func(int, int) ([]byte, error)) ([]SketchSceneAsset, error) {
	if err := s.transfer(ctx, source.Handle, source.Length, read); err != nil {
		return nil, err
	}
	return s.FinishSource(ctx, sketch, source)
}
func (s *SketchSceneSession) InstallAsset(ctx context.Context, handle string, asset SketchSceneAsset, read func(int, int) ([]byte, error)) error {
	if !recording.ValidHash(asset.Hash) || asset.Size < 0 || asset.Size > SketchSceneAssetLimit {
		return board.ErrSketchDeliveryLimit
	}
	stepper, err := s.BeginAssetTransfer(ctx, handle, int(asset.Size), read)
	if err != nil {
		return err
	}
	for {
		done, err := stepper.Step(ctx)
		if err != nil {
			return err
		}
		if done {
			break
		}
	}
	return s.FinishAsset(ctx, handle, asset)
}
func (s *SketchSceneSession) Render(ctx context.Context) (SketchSceneReady, error) {
	var out SketchSceneReady
	err := s.invoke(ctx, "render", []any{}, &out)
	return out, err
}
func (s *SketchSceneSession) HitTest(ctx context.Context, point [2]float64) (SketchSceneHit, error) {
	var out SketchSceneHit
	err := s.invoke(ctx, "hitTest", []any{point}, &out)
	return out, err
}
func (s *SketchSceneSession) Placement(ctx context.Context, path any) (*recording.Box, error) {
	var out *recording.Box
	err := s.invoke(ctx, "placement", []any{path}, &out)
	return out, err
}
func (s *SketchSceneSession) Retire() { s.retired.Store(true) }

func (s *SketchSceneSession) Dispose(ctx context.Context) error {
	s.retired.Store(true)
	cleanupCtx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	ctx = cleanupCtx
	var failure error
	if s.world > 0 {
		failure = s.caller.Call(ctx, s.session, "Runtime.evaluate", map[string]any{"contextId": s.world, "expression": "globalThis.__couchcastSketchScene?.dispose()", "allowUnsafeEvalBlockedByCSP": false}, nil)
	}
	if s.attached {
		failure = errors.Join(failure, s.caller.Call(ctx, "", "Target.detachFromTarget", map[string]any{"sessionId": s.session}, nil))
	}
	s.world = 0
	s.attached = false
	return failure
}
