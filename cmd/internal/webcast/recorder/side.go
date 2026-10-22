package recorder

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/assets"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// The side workers make the slow calls (asset bodies, screenshots and
// layout) off the recorder's goroutine, one at a time; their results come
// back as inputs, so the goroutine stays the stream's single writer.

type assetJob struct {
	store              assets.Store
	load               bool   // not held by Chrome: load it in the page's context
	session, requestID string // a response seen by the Network domain
	frameID            string // or a resource already loaded (Page.getResourceContent)
	url, mime          string
}

type shotJob struct {
	contextID int64
	doc       int
	at        time.Time
	top       string
	sessions  []string
	vp        rec.Viewport
	scroll    [2]float64
}

// Asset ownership: a URL's body is fetched once per recording, by the first
// source to claim it, whatever order the sources arrive in.
//   - A Network response (the body the page got) claims its URL, and takes
//     it from a reference still waiting.
//   - A resource Chrome already held at a segment's start claims its URL.
//   - A reference (an element's src, a FontFace font's src) waits
//     refSettle, so responses claim first; if nothing has claimed its URL by
//     then, it is loaded in the page's context. Only what the page loaded
//     eagerly is referenced: never an image with a srcset or in a <picture>
//     (the browser's choice isn't known) or a lazy one (not loaded yet; a
//     response comes when it is).
//
// A body that changes under one URL within a recording keeps its first
// capture (a dev server busts caches with new URLs).
var refSettle = 2 * time.Second // a var so tests can shorten it

// owned: a URL's body has been, or is being, fetched in this recording.
func (r *Recorder) own(u string) bool {
	if r.owned[u] {
		return false
	}
	r.owned[u] = true
	delete(r.pending, u)
	return true
}

// queueAsset hands a body to fetch to the asset worker; a full queue drops
// it (counted in Stats) and gives the URL up, never blocking the recorder.
func (r *Recorder) queueAsset(j assetJob) {
	if r.assets == nil {
		return
	}
	j.store = r.assets
	if !r.own(j.url) {
		return
	}
	select {
	case r.assetQ <- j:
	default:
		delete(r.owned, j.url) // a later sighting may try again
		r.assetsLost++         // reported in Stats, not one line per asset
	}
}

// queueRef notes the asset an element references, to load after refSettle
// if nothing else brings it.
func (r *Recorder) queueRef(ref Ref, now time.Time) {
	if ref.Srcset || ref.Lazy || ref.Chosen {
		return
	}
	r.queueURL(ref.Session, resolveURL(ref.DocURL, ref.Raw), now)
}

// queueURL waits refSettle, then loads an asset nothing else has brought.
func (r *Recorder) queueURL(session, u string, now time.Time) {
	if r.assets == nil || u == "" || !strings.HasPrefix(u, "http") || r.owned[u] {
		return
	}
	if _, ok := r.m.st.Assets[u]; ok {
		return
	}
	if _, ok := r.pending[u]; !ok {
		r.pending[u] = pendingRef{session: session, due: now.Add(refSettle)}
	}
}

type pendingRef struct {
	session string
	due     time.Time
}

// flushRefs loads the references whose wait is over.
func (r *Recorder) flushRefs(now time.Time) {
	var due []string
	for u, p := range r.pending {
		if !now.Before(p.due) {
			due = append(due, u)
		}
	}
	sort.Strings(due)
	for _, u := range due {
		p := r.pending[u]
		r.queueAsset(assetJob{session: p.session, frameID: r.frameIDs[p.session], url: u, load: true})
	}
}

// resolveURL makes a reference absolute; "" when it can't.
func resolveURL(base, ref string) string {
	b, err := url.Parse(base)
	if err != nil {
		return ""
	}
	u, err := b.Parse(strings.TrimSpace(ref))
	if err != nil {
		return ""
	}
	u.Fragment = ""
	return u.String()
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
		body, mime, err := r.fetchAsset(j)
		var a *rec.Asset
		if err == nil {
			a, err = j.store.Put(r.ctx, assets.Capture{URL: j.url, MIME: mime, Body: body})
		}
		if err != nil {
			r.post(assetFailed{})
			continue
		}
		r.post(assetDone{a})
	}
}

// fetchAsset reads a body from Chrome: a response it just saw, a resource
// it already holds, or, when it holds none (a font, an image from its
// memory cache), a load in the page's own context (its cookies, its cache);
// never from outside the browser.
func (r *Recorder) fetchAsset(j assetJob) ([]byte, string, error) {
	var res struct {
		Body    string `json:"body"`    // Network.getResponseBody
		Content string `json:"content"` // Page.getResourceContent
		B64     bool   `json:"base64Encoded"`
	}
	var err error
	text := ""
	switch {
	case j.requestID != "":
		_, err = r.call(j.session, "Network.getResponseBody", map[string]any{"requestId": j.requestID}, &res)
		text = res.Body
	case !j.load:
		_, err = r.call(j.session, "Page.getResourceContent", map[string]any{"frameId": j.frameID, "url": j.url}, &res)
		text = res.Content
	}
	if j.requestID == "" && (j.load || err != nil) {
		return r.loadResource(j)
	}
	if err != nil {
		return nil, "", err
	}
	if !res.B64 {
		return []byte(text), j.mime, nil
	}
	body, err := base64.StdEncoding.DecodeString(text)
	return body, j.mime, err
}

// loadResource loads a URL in the page's context (Network.loadNetworkResource)
// and reads its stream.
func (r *Recorder) loadResource(j assetJob) ([]byte, string, error) {
	var res struct {
		Resource struct {
			Success bool              `json:"success"`
			Status  int               `json:"httpStatusCode"`
			Stream  string            `json:"stream"`
			Headers map[string]string `json:"headers"`
		} `json:"resource"`
	}
	if _, err := r.call(j.session, "Network.loadNetworkResource", map[string]any{"frameId": j.frameID, "url": j.url,
		"options": map[string]any{"disableCache": false, "includeCredentials": true}}, &res); err != nil {
		return nil, "", err
	}
	if !res.Resource.Success || res.Resource.Stream == "" || res.Resource.Status >= 400 {
		return nil, "", fmt.Errorf("load %s: status %d", j.url, res.Resource.Status)
	}
	defer r.call(j.session, "IO.close", map[string]any{"handle": res.Resource.Stream}, nil)
	var body []byte
	for {
		var chunk struct {
			Data string `json:"data"`
			B64  bool   `json:"base64Encoded"`
			EOF  bool   `json:"eof"`
		}
		if _, err := r.call(j.session, "IO.read", map[string]any{"handle": res.Resource.Stream, "size": 1 << 20}, &chunk); err != nil {
			return nil, "", err
		}
		part := []byte(chunk.Data)
		if chunk.B64 {
			var err error
			if part, err = base64.StdEncoding.DecodeString(chunk.Data); err != nil {
				return nil, "", err
			}
		}
		body = append(body, part...)
		if chunk.EOF {
			break
		}
		if len(body) > maxAssetBody {
			return nil, "", fmt.Errorf("load %s: over %d bytes", j.url, maxAssetBody)
		}
	}
	mime := j.mime
	for k, v := range res.Resource.Headers {
		if strings.EqualFold(k, "content-type") {
			mime = strings.TrimSpace(strings.SplitN(v, ";", 2)[0])
		}
	}
	if mime == "" {
		mime = "application/octet-stream"
	}
	return body, mime, nil
}

// maxAssetBody bounds one loaded body (a page's own responses are bounded
// by Chrome's network buffer instead).
const maxAssetBody = 32 << 20

// post queues a side worker's result.
func (r *Recorder) post(in any) { r.enqueue(in, false) }

// assetFailed: a body couldn't be fetched or stored; counted.
type assetFailed struct{}

// maybeShot asks for a screenshot and layout when one is due.
func (r *Recorder) maybeShot(now time.Time) {
	if !r.live || r.frames == nil || r.opts.ShotEvery < 0 || now.Before(r.nextShot) {
		return
	}
	contextID, _ := r.topContext(r.top)
	j := shotJob{contextID: contextID, doc: r.m.st.DocSeg, at: now, top: r.top, sessions: r.castSessions(), vp: r.m.st.VP, scroll: r.m.TopScroll()}
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
		d := shotDone{at: j.at, doc: j.doc, top: j.top}
		var res struct {
			Data string `json:"data"`
		}
		// A screenshot shows the page as of its capture: its time is the
		// call's midpoint, as #453's harness stamped the fixtures' frames.
		t0 := r.opts.Now()
		err := CaptureBackgroundScreenshot(r.ctx, r.opts.Call, j.top, j.contextID, map[string]any{"format": "png"}, &res)
		if errors.Is(err, ErrBackgroundCaptureDeferred) {
			continue
		}
		d.at = t0.Add(r.opts.Now().Sub(t0) / 2)
		if err != nil {
			d.failed = err
			r.post(d)
			continue
		}
		var full []byte
		full, err = base64.StdEncoding.DecodeString(res.Data)
		if err == nil {
			d.png, d.w, d.h, err = LowRes(full, j.vp.DPR)
		}
		if err == nil && j.vp.W > 0 {
			d.scale = float64(d.w) / float64(j.vp.W) // image px per CSS px
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
	if !r.live || r.frames == nil || d.doc != r.m.st.DocSeg || d.top != r.top {
		return
	}
	if d.failed != nil {
		r.framesLost++
		return
	}
	// The frame is at the capture's midpoint (the format's frame t), never
	// before its segment. Records kept arriving while the layout was
	// captured after it, so the layout record alone takes the stream's
	// time: clamping the frame there too stamped a busy page's pixels up to
	// ~90ms after what they show (#472).
	f := &rec.Frame{File: fmt.Sprintf("frames/%d.png", r.shotN), W: d.w, H: d.h, Scale: d.scale}
	f.SetTime(max(r.t(d.at), r.segT))
	t := max(f.T, r.m.st.T)
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

// LowRes is a viewport screenshot at half its CSS size, box-filtered: the
// fidelity screenshots' form, as #453's harness made the fixtures' frames.
// dpr is the screenshot's device pixels per CSS pixel, so a Retina cast's
// frame is the same size as anyone's. The viewport is captured whole and
// scaled here, because CDP's clip is in document coordinates and offsets a
// scrolled page. The fidelity check scales its replays the same way.
func LowRes(full []byte, dpr float64) (out []byte, w, h int, err error) {
	src, err := png.Decode(bytes.NewReader(full))
	if err != nil {
		return nil, 0, 0, err
	}
	b := src.Bounds()
	rgba := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), src, b.Min, draw.Src)
	// Device pixels per output pixel: half the CSS size, but never less
	// than one source pixel each (a DPR under 0.5 is not upsampled).
	f := max(2*dpr, 1)
	w, h = int(float64(b.Dx())/f), int(float64(b.Dy())/f)
	if w == 0 || h == 0 {
		return nil, 0, 0, fmt.Errorf("screenshot %dx%d is too small", b.Dx(), b.Dy())
	}
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		y0, y1 := int(float64(y)*f), min(max(int(float64(y+1)*f), int(float64(y)*f)+1), b.Dy())
		for x := 0; x < w; x++ {
			x0, x1 := int(float64(x)*f), min(max(int(float64(x+1)*f), int(float64(x)*f)+1), b.Dx())
			var sum [4]int
			n := 0
			for sy := y0; sy < y1; sy++ {
				row := rgba.Pix[sy*rgba.Stride:]
				for sx := x0; sx < x1; sx++ {
					p := row[sx*4 : sx*4+4]
					sum[0] += int(p[0])
					sum[1] += int(p[1])
					sum[2] += int(p[2])
					sum[3] += int(p[3])
					n++
				}
			}
			o := dst.Pix[y*dst.Stride+x*4:]
			for c := 0; c < 4; c++ {
				o[c] = uint8(sum[c] / max(n, 1))
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), w, h, nil
}
