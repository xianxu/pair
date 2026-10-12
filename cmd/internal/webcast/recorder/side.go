package recorder

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/assets"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// The side workers make the slow calls (asset bodies, screenshots and
// layout) off the recorder's goroutine, one at a time; their results come
// back as inputs, so the goroutine stays the stream's single writer.

type assetJob struct {
	store              assets.Store
	session, requestID string // a response seen by the Network domain
	frameID            string // or a resource already loaded (Page.getResourceContent)
	url, mime          string
}

type shotJob struct {
	at       time.Time
	top      string
	sessions []string
	vp       rec.Viewport
	scroll   [2]float64
}

// queueAsset hands a body to fetch to the asset worker; a full queue drops
// it (counted in the log), never blocking the recorder.
func (r *Recorder) queueAsset(j assetJob) {
	if r.assets == nil {
		return
	}
	j.store = r.assets
	if _, ok := r.m.st.Assets[j.url]; ok && j.requestID == "" {
		return // already in this recording, and not a fresh response
	}
	select {
	case r.assetQ <- j:
	default:
		r.assetsLost++ // reported in Stats, not one line per asset
	}
}

// goAssets queues the images, stylesheets and fonts a page already loaded
// before it went live (the Network domain only sees later loads).
func (r *Recorder) goAssets() {
	if r.assets == nil {
		return
	}
	for _, s := range r.castSessions() {
		var res struct {
			Tree frameResources `json:"frameTree"`
		}
		if _, err := r.call(s, "Page.getResourceTree", nil, &res); err != nil {
			continue
		}
		res.Tree.walk(func(frameID string, rs []resource) {
			for _, x := range rs {
				if assetType(x.Type) {
					r.queueAsset(assetJob{session: s, frameID: frameID, url: x.URL, mime: x.MIMEType})
				}
			}
		})
	}
}

type resource struct {
	URL      string `json:"url"`
	Type     string `json:"type"`
	MIMEType string `json:"mimeType"`
}

type frameResources struct {
	Frame struct {
		ID string `json:"id"`
	} `json:"frame"`
	Resources   []resource       `json:"resources"`
	ChildFrames []frameResources `json:"childFrames"`
}

func (f frameResources) walk(fn func(string, []resource)) {
	fn(f.Frame.ID, f.Resources)
	for _, c := range f.ChildFrames {
		c.walk(fn)
	}
}

func (r *Recorder) assetWorker() {
	defer r.side.Done()
	for j := range r.assetQ {
		var res struct {
			Body    string `json:"body"`    // Network.getResponseBody
			Content string `json:"content"` // Page.getResourceContent
			B64     bool   `json:"base64Encoded"`
		}
		var err error
		if j.requestID != "" {
			_, err = r.call(j.session, "Network.getResponseBody", map[string]any{"requestId": j.requestID}, &res)
		} else {
			_, err = r.call(j.session, "Page.getResourceContent", map[string]any{"frameId": j.frameID, "url": j.url}, &res)
		}
		text := res.Body
		if j.requestID == "" {
			text = res.Content
		}
		body := []byte(text)
		if err == nil && res.B64 {
			body, err = base64.StdEncoding.DecodeString(text)
		}
		var a *rec.Asset
		if err == nil {
			a, err = j.store.Put(r.ctx, assets.Capture{URL: j.url, MIME: j.mime, Body: body})
		}
		if err != nil {
			r.post(assetFailed{})
			continue
		}
		r.post(assetDone{a})
	}
}

// post queues a side worker's result.
func (r *Recorder) post(in any) { r.enqueue(in, false) }

// assetFailed: a body couldn't be fetched or stored; counted.
type assetFailed struct{}

// maybeShot asks for a screenshot and layout when one is due.
func (r *Recorder) maybeShot(now time.Time) {
	if !r.live || r.frames == nil || r.opts.ShotEvery < 0 || now.Before(r.nextShot) {
		return
	}
	j := shotJob{at: now, top: r.top, sessions: r.castSessions(), vp: r.m.st.VP, scroll: r.m.TopScroll()}
	select {
	case r.shotQ <- j:
		r.nextShot = now.Add(r.opts.ShotEvery)
	default: // one is in flight
	}
}

// layoutDoc is one document of a DOMSnapshot capture.
type layoutDoc struct {
	session, frameID string
	scroll           [2]float64
	backend          []int
	bounds           []rec.Box
}

func (r *Recorder) shotWorker() {
	defer r.side.Done()
	for j := range r.shotQ {
		d := shotDone{at: j.at, scale: shotScale}
		var res struct {
			Data string `json:"data"`
		}
		clip := map[string]any{"x": j.scroll[0], "y": j.scroll[1], "width": j.vp.W, "height": j.vp.H, "scale": shotScale}
		if _, err := r.call(j.top, "Page.captureScreenshot", map[string]any{"format": "png", "clip": clip}, &res); err != nil {
			d.failed = err
			r.post(d)
			continue
		}
		var err error
		if d.png, err = base64.StdEncoding.DecodeString(res.Data); err == nil {
			if cfg, err2 := png.DecodeConfig(bytes.NewReader(d.png)); err2 == nil {
				d.w, d.h = cfg.Width, cfg.Height
			} else {
				err = err2
			}
		}
		if err != nil {
			d.failed = fmt.Errorf("screenshot: %w", err)
			r.post(d)
			continue
		}
		for _, s := range j.sessions {
			d.docs = append(d.docs, r.captureLayout(s)...)
		}
		r.post(d)
	}
}

// captureLayout is a session's DOMSnapshot: each frame session captures
// its own (#457: the top session's capture leaves out-of-process frames
// out).
func (r *Recorder) captureLayout(session string) []layoutDoc {
	var snap struct {
		Docs []struct {
			FrameID int     `json:"frameId"`
			ScrollX float64 `json:"scrollOffsetX"`
			ScrollY float64 `json:"scrollOffsetY"`
			Nodes   struct {
				Backend []int `json:"backendNodeId"`
			} `json:"nodes"`
			Layout struct {
				NodeIndex []int       `json:"nodeIndex"`
				Bounds    [][]float64 `json:"bounds"`
			} `json:"layout"`
		} `json:"documents"`
		Strings []string `json:"strings"`
	}
	if _, err := r.call(session, "DOMSnapshot.captureSnapshot", map[string]any{"computedStyles": []string{}}, &snap); err != nil {
		return nil
	}
	var out []layoutDoc
	for _, d := range snap.Docs {
		ld := layoutDoc{session: session, scroll: [2]float64{d.ScrollX, d.ScrollY}}
		if d.FrameID >= 0 && d.FrameID < len(snap.Strings) {
			ld.frameID = snap.Strings[d.FrameID]
		}
		for i, ni := range d.Layout.NodeIndex {
			if ni >= len(d.Nodes.Backend) || i >= len(d.Layout.Bounds) || len(d.Layout.Bounds[i]) < 4 {
				continue
			}
			b := d.Layout.Bounds[i]
			ld.backend = append(ld.backend, d.Nodes.Backend[ni])
			ld.bounds = append(ld.bounds, rec.Box{b[0], b[1], b[2], b[3]})
		}
		out = append(out, ld)
	}
	return out
}

// shotDone writes a screenshot and the layout boxes taken with it, mapped
// into top-page coordinates.
func (r *Recorder) shotDone(d shotDone) {
	if !r.live || r.frames == nil {
		return
	}
	if d.failed != nil {
		r.framesLost++
		return
	}
	t := r.t(d.at)
	if t < r.m.st.T {
		t = r.m.st.T
	}
	f := &rec.Frame{File: fmt.Sprintf("frames/%d.png", r.shotN), W: d.w, H: d.h, Scale: d.scale}
	f.SetTime(t)
	if err := r.frames.PutFrame(f, d.png); err != nil {
		r.framesLost++
		return
	}
	r.shotN++
	boxes := map[rec.NodeID]rec.Box{}
	for _, doc := range d.docs {
		// The top document's bounds are page coordinates already; a frame's
		// are its own document's, mapped through its viewport's place in
		// the top page.
		off := doc.scroll
		if !(doc.session == r.top && doc.frameID == r.frameIDs[r.top]) {
			var ok bool
			if off, ok = r.frameOffset(doc.session, doc.frameID, 0); !ok {
				continue
			}
		}
		for i, b := range doc.backend {
			id, ok := r.m.IDOf(doc.session, b)
			if !ok {
				continue
			}
			if n := r.m.st.Node(id); n == nil || n.Y != rec.YEl {
				continue
			}
			x := doc.bounds[i]
			boxes[id] = rec.Box{round2(off[0] + x[0] - doc.scroll[0]), round2(off[1] + x[1] - doc.scroll[1]), round2(x[2]), round2(x[3])}
		}
	}
	r.m.Emit(t, &rec.Layout{Boxes: boxes})
}

func round2(f float64) float64 {
	if f < 0 {
		return -round2(-f)
	}
	return float64(int64(f*100+0.5)) / 100
}
