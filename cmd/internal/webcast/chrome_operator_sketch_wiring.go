package webcast

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
)

// The cast's existing drive-capability subscription wakes this worker before
// a held ctl write settles, including witness changes with the same enum.
func (c *Chrome) RefreshDrawingAuthority() { c.refreshOperator() }

func (c *Chrome) initOperatorSketch(b *board.Board) *operatorSketchRuntime {
	ro := c.opts.Record
	if ro == nil || ro.Sketches == nil || ro.SketchAuthority == nil {
		return nil
	}
	runtime := newOperatorSketchRuntime(operatorSketchRuntimeOptions{Caller: c.conn, Board: b, Service: ro.Sketches, Authority: ro.SketchAuthority, Go: c.work.Go,
		Exclude:  func(frame, session string) error { return c.ExcludeSketchFrame(c.ctx, frame, session) },
		AssetURL: c.ExcludeSketchAssetURL,
		AssetRead: func(ctx context.Context, a board.SketchAuthority, id, source, hash string, offset, length int) ([]byte, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if err := ro.Sketches.AdmitSourceAsset(a, id, source, hash); err != nil {
				return nil, err
			}
			if ro.SketchAsset == nil {
				return nil, ErrNotRecording
			}
			data, total, err := ro.SketchAsset(hash, offset, length)
			if err != nil {
				return nil, err
			}
			if offset > total || len(data) != min(length, total-offset) {
				return nil, errors.New("private asset chunk changed")
			}
			if err = ro.Sketches.AdmitSourceAsset(a, id, source, hash); err != nil {
				return nil, err
			}
			return data, nil
		},
	})
	c.operatorSketch = runtime
	return runtime
}

type operatorDrawingPermission struct {
	permission operatorPermission
	projection string
}

type operatorDrawingValue struct {
	Revision   string `json:"revision"`
	Generation string `json:"generation"`
	Epoch      uint64 `json:"epoch"`
	Holder     string `json:"holder"`
	Ready      bool   `json:"ready"`
	Active     bool   `json:"active"`
}

// Drawing permission follows the same uncertainty fence as ordinary annotation
// permission. A timed-out enable must acknowledge a newer false before retry.
func (c *Chrome) deliverOperatorDrawing(target operatorTarget, b *board.Board, p *operatorDrawingPermission, next func() uint64) {
	ro := c.opts.Record
	if ro == nil || ro.DrawingAuthority == nil {
		return
	}
	authority := ro.DrawingAuthority()
	annotation := b.AnnotationCapability()
	desired := authority.Ready && authority.Holder == "operator" && target.active
	value := operatorDrawingValue{Generation: strconv.FormatUint(annotation.Generation, 10), Epoch: authority.Epoch, Holder: authority.Holder, Ready: desired, Active: target.active}
	projection, _ := json.Marshal(value)
	if p.projection != string(projection) {
		p.projection = string(projection)
		p.permission.desired = desired
		p.permission.revision = next()
		if p.permission.phase != operatorPermissionFencing {
			p.permission.phase = operatorPermissionPending
		}
	}
	if p.permission.phase == operatorPermissionReady {
		return
	}
	value.Revision = strconv.FormatUint(p.permission.revision, 10)
	value.Ready = p.permission.allowed()
	data, _ := json.Marshal(value)
	var result struct {
		Result struct {
			Value *operatorDrawingValue `json:"value"`
		} `json:"result"`
		Exception json.RawMessage `json:"exceptionDetails"`
	}
	ctx, cancel := context.WithTimeout(c.ctx, callTimeout)
	err := c.conn.Call(ctx, target.session, "Runtime.evaluate", map[string]any{"contextId": target.context, "returnByValue": true, "expression": "globalThis.__couchcastSketchAuthority&&globalThis.__couchcastSketchAuthority(" + string(data) + ")"}, &result)
	cancel()
	receipt := result.Result.Value
	if err == nil && len(result.Exception) == 0 && receipt != nil && receipt.Revision == value.Revision && receipt.Generation == value.Generation && receipt.Epoch == value.Epoch && receipt.Ready == value.Ready {
		if p.permission.phase == operatorPermissionFencing && p.permission.desired {
			p.permission.revision = next()
			p.permission.phase = operatorPermissionPending
		} else {
			p.permission.phase = operatorPermissionReady
		}
	} else if value.Ready && (err != nil || len(result.Exception) != 0) {
		p.permission.revision = next()
		p.permission.phase = operatorPermissionFencing
	}
}

// Identity binding waits until the parent has installed the journal headers,
// so a large bootstrap cannot exhaust retries before its anchor is eligible.
func (r *operatorSketchRuntime) originalsReady(target operatorTarget) bool {
	if r == nil {
		return true
	}
	state := r.states[operatorContextID{target.session, target.context}]
	if state == nil || state.scope != target.sketchScope || state.retired || state.delivery != nil {
		return false
	}
	cursor := r.opts.Board.OperatorSketchCursor()
	return state.cursor == cursor
}
