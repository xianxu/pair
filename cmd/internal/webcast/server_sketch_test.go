package webcast

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

type sketchHTTPFixture struct {
	s    *Server
	bd   *board.Board
	w    *board.Writer
	hub  *recording.Hub
	mem  *recording.MemSink
	ctrl *drive.MemController
	a, b drive.Seat
	node recording.NodeID
}

func newSketchHTTPFixture(t *testing.T) *sketchHTTPFixture {
	t.Helper()
	f := &sketchHTTPFixture{hub: recording.NewHub(256), mem: &recording.MemSink{}}
	f.bd = board.New(notes.Structural{}, func(r recording.Record, doc int) error { return f.w.Emit(r, doc) })
	f.w = board.NewWriter(recording.Tee(f.mem, f.hub, f.bd), nil)
	f.bd.WithAnnotationCleanup(f.w.ClearAnnotations)
	if err := f.bd.SetAnnotationAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	b := recording.NewBuilder("sketch-http")
	p := b.El("p", nil, b.Txt("Source"))
	f.node = p.ID
	b.Page(recording.SegLive, "http://x/", p)
	b.Recs[1].(*recording.Seg).PageLoad = &recording.PageLoad{Target: "tab1", Loader: "sketch-http-load", Root: 1}
	for _, r := range b.Recs {
		if err := f.w.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	f.ctrl = drive.NewMemController(f.w, nil, func() int64 { return 0 }, time.Hour)
	if err := f.ctrl.SetDriveAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	f.a, _ = f.ctrl.Join("Ana")
	f.b, _ = f.ctrl.Join("Bo")
	f.s = NewServer("view", NewFakeBrowser(), time.Second, f.hub, nil).WithBoard(f.bd).WithPointer("pointer").WithDrive(f.ctrl, "driver")
	f.s.sketches = f.bd.NewSketchService(f.w.WriteSketch, f.w.ObserveSketch, f.hub.CandidateSketchSource)
	t.Cleanup(func() { f.ctrl.Close(); f.bd.Close(); f.w.Close() })
	return f
}
func (f *sketchHTTPFixture) grant(t *testing.T, seat drive.Seat) uint64 {
	t.Helper()
	if err := f.ctrl.Request(seat.Key); err != nil {
		t.Fatal(err)
	}
	if err := f.ctrl.Grant(seat.Viewer.ID); err != nil {
		t.Fatal(err)
	}
	return f.ctrl.DrawingAuthority().Epoch
}
func (f *sketchHTTPFixture) request(link, kind, key, epoch string, body io.Reader) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/"+link+"/"+kind, body)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set(drive.SeatHeader, key)
	r.Header.Set("X-Couch-Annotation-Generation", strconv.FormatUint(f.bd.AnnotationCapability().Generation, 10))
	r.Header.Set("X-Couch-Drawing-Epoch", epoch)
	w := httptest.NewRecorder()
	f.s.ServeHTTP(w, r)
	return w
}
func (f *sketchHTTPFixture) post(t *testing.T, kind string, seat drive.Seat, epoch uint64, payload map[string]any) board.SketchReply {
	t.Helper()
	payload["epoch"] = epoch
	body, _ := json.Marshal(payload)
	w := f.request("driver", kind, seat.Key, strconv.FormatUint(epoch, 10), bytes.NewReader(body))
	if w.Code != http.StatusOK {
		t.Fatalf("%s: %d %s", kind, w.Code, w.Body.String())
	}
	reply, err := board.ParseSketchReply(w.Body.Bytes())
	if err != nil {
		t.Fatalf("reply %s: %v", w.Body.String(), err)
	}
	return reply
}
func (f *sketchHTTPFixture) begin(t *testing.T, seat drive.Seat, epoch uint64, op string) board.SketchReply {
	t.Helper()
	v := f.post(t, "sketch-begin", seat, epoch, map[string]any{"operation": op, "sequence": 0, "seg": 1, "color": 0})
	if v.Outcome != board.SketchAccepted || v.Source == nil {
		t.Fatalf("begin %+v", v)
	}
	return v
}

type sketchReadProbe struct {
	reads  int
	reader io.Reader
	before func()
}

func (p *sketchReadProbe) Read(b []byte) (int, error) {
	p.reads++
	if p.before != nil {
		fn := p.before
		p.before = nil
		fn()
	}
	return p.reader.Read(b)
}
func TestSketchHTTPAuthorityRefusesBeforeBody(t *testing.T) {
	f := newSketchHTTPFixture(t)
	epoch := f.grant(t, f.a)
	for _, tc := range []struct {
		name, link, key, epoch string
		status                 int
	}{
		{"view", "view", f.a.Key, fmt.Sprint(epoch), 403},
		{"pointer", "pointer", f.a.Key, fmt.Sprint(epoch), 403},
		{"ungranted seat", "driver", f.b.Key, fmt.Sprint(epoch), 403},
		{"bad key", "driver", "invalid", fmt.Sprint(epoch), 401},
		{"stale epoch", "driver", f.a.Key, fmt.Sprint(epoch + 1), 403},
		{"noncanonical epoch", "driver", f.a.Key, "0" + fmt.Sprint(epoch), 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			probe := &sketchReadProbe{reader: strings.NewReader(`{}`)}
			w := f.request(tc.link, "sketch-begin", tc.key, tc.epoch, probe)
			if w.Code != tc.status || probe.reads != 0 {
				t.Fatalf("status=%d reads=%d body=%s", w.Code, probe.reads, w.Body.String())
			}
		})
	}
}
func TestSketchHTTPRevocationDuringBodyAndHeaderMismatch(t *testing.T) {
	f := newSketchHTTPFixture(t)
	epoch := f.grant(t, f.a)
	body := fmt.Sprintf(`{"operation":"begin","sequence":0,"epoch":%d,"seg":1,"color":0}`, epoch)
	probe := &sketchReadProbe{reader: strings.NewReader(body), before: func() { f.grant(t, f.b) }}
	w := f.request("driver", "sketch-begin", f.a.Key, fmt.Sprint(epoch), probe)
	if w.Code != 403 {
		t.Fatalf("body revoked admission: %d %s", w.Code, w.Body.String())
	}
	for _, r := range f.mem.Records() {
		if r.Kind() == "sketch-begin" {
			t.Fatal("revoked begin emitted")
		}
	}
	epoch = f.ctrl.DrawingAuthority().Epoch
	w = f.request("driver", "sketch-begin", f.b.Key, fmt.Sprint(epoch), strings.NewReader(fmt.Sprintf(`{"operation":"mismatch","sequence":0,"epoch":%d,"seg":1,"color":0}`, epoch+1)))
	if w.Code != 400 {
		t.Fatalf("header/body mismatch %d", w.Code)
	}
}
func TestSketchHTTPStreamPressureRetryAndReplay(t *testing.T) {
	f := newSketchHTTPFixture(t)
	epoch := f.grant(t, f.a)
	observer, cancel := f.hub.Subscribe()
	defer cancel()
	for {
		select {
		case <-observer:
			continue
		default:
		}
		break
	}
	v := f.begin(t, f.a, epoch, "begin")
	if got := f.begin(t, f.a, epoch, "begin"); got.Sketch != v.Sketch {
		t.Fatal("retry chose another identity")
	}
	for _, command := range []struct {
		kind string
		body map[string]any
	}{
		{"sketch-stroke-begin", map[string]any{"operation": "stroke", "sequence": 1, "sketch": v.Sketch, "id": "s", "node": f.node, "box": []float64{0, 0, 50, 20}}},
		{"sketch-points", map[string]any{"operation": "points", "sequence": 2, "sketch": v.Sketch, "id": "s", "chunk": 0, "pts": [][]float64{{1, 2, .2}, {3, 4, .9}}}},
	} {
		if got := f.post(t, command.kind, f.a, epoch, command.body); got.Outcome != board.SketchAccepted {
			t.Fatal(got)
		}
	}
	seen := false
	for !seen {
		select {
		case r := <-observer:
			if p, ok := r.(*recording.SketchPoints); ok {
				if p.Pts[1][2] != .9 {
					t.Fatal("pressure lost")
				}
				seen = true
			}
		case <-time.After(time.Second):
			t.Fatal("observer did not receive incomplete stroke")
		}
	}
	for _, r := range f.mem.Records() {
		if r.Kind() == "sketch-stroke-end" {
			t.Fatal("partial visibility waited for end")
		}
	}
	if got := f.post(t, "sketch-stroke-end", f.a, epoch, map[string]any{"operation": "end", "sequence": 3, "sketch": v.Sketch, "id": "s"}); got.Outcome != board.SketchAccepted {
		t.Fatal(got)
	}
	files := map[string][]byte{}
	for name, records := range recording.Split(f.mem.Records()) {
		for _, r := range records {
			line, _ := recording.Encode(r)
			files[name] = append(append(files[name], line...), '\n')
			if bytes.Contains(line, []byte(f.a.Key)) {
				t.Fatal("private seat key persisted")
			}
		}
	}
	if bad, _ := recording.Validate(files); len(bad) != 0 {
		t.Fatal(bad)
	}
}
func TestSketchHTTPHandoffOldAuthorLabelAndOwnerIsolation(t *testing.T) {
	f := newSketchHTTPFixture(t)
	firstEpoch := f.grant(t, f.a)
	first := f.begin(t, f.a, firstEpoch, "A-begin")
	secondEpoch := f.grant(t, f.b)
	second := f.begin(t, f.b, secondEpoch, "B-begin")
	wrong := f.post(t, "sketch-status", f.b, firstEpoch, map[string]any{"operation": "A-begin", "sequence": 0, "sketch": first.Sketch})
	if wrong.Outcome != board.SketchRefused {
		t.Fatal("another seat queried original receipt")
	}
	label := f.post(t, "sketch-label", f.a, firstEpoch, map[string]any{"operation": "label", "sequence": 1, "sketch": first.Sketch, "label": "Ana intent"})
	if label.Outcome != board.SketchAccepted {
		t.Fatalf("old author label %+v", label)
	}
	status := f.post(t, "sketch-status", f.b, secondEpoch, map[string]any{"operation": "B-begin", "sequence": 0, "sketch": second.Sketch})
	if status.Outcome != board.SketchAccepted || status.Source == nil || status.Source.Handle != second.Source.Handle {
		t.Fatal("old label disturbed new drawer")
	}
	probe := &sketchReadProbe{reader: strings.NewReader(`{}`)}
	if w := f.request("driver", "sketch-source", f.a.Key, fmt.Sprint(firstEpoch), probe); w.Code != 403 || probe.reads != 0 {
		t.Fatal("old author regained drawing/source authority")
	}
}
func TestSketchHTTPSourceAndAssetScope(t *testing.T) {
	f := newSketchHTTPFixture(t)
	epoch := f.grant(t, f.a)
	hash := strings.Repeat("a", 64)
	if err := f.w.Put(&recording.Asset{URL: "http://x/image.png", Hash: hash, MIME: "image/png", Size: 4}); err != nil {
		t.Fatal(err)
	}
	v := f.begin(t, f.a, epoch, "begin")
	f.s.sketchAsset = func(h string, offset, length int) ([]byte, int, error) {
		if h != hash {
			t.Fatal("unauthorized asset reached reader")
		}
		data := []byte("DATA")
		return data[offset:min(offset+length, len(data))], len(data), nil
	}
	for _, tc := range []struct{ kind, asset string }{{"sketch-source", ""}, {"sketch-asset", hash}} {
		body := map[string]any{"operation": "read", "sequence": 0, "epoch": epoch, "sketch": v.Sketch, "source": v.Source.Handle, "offset": 0, "length": 4}
		if tc.asset != "" {
			body["asset"] = tc.asset
		}
		raw, _ := json.Marshal(body)
		w := f.request("driver", tc.kind, f.a.Key, fmt.Sprint(epoch), bytes.NewReader(raw))
		if w.Code != 200 {
			t.Fatalf("read %d %s", w.Code, w.Body.String())
		}
		var reply struct {
			Source        string
			Offset, Total int
			Body          string
		}
		if json.Unmarshal(w.Body.Bytes(), &reply) != nil || reply.Source != v.Source.Handle || reply.Offset != 0 {
			t.Fatal("bad transfer reply")
		}
		data, err := base64.StdEncoding.DecodeString(reply.Body)
		if err != nil || len(data) != 4 {
			t.Fatal("bad transfer body")
		}
	}
	body := fmt.Sprintf(`{"operation":"bad-asset","sequence":0,"epoch":%d,"sketch":%q,"source":%q,"asset":%q,"offset":0,"length":4}`, epoch, v.Sketch, v.Source.Handle, strings.Repeat("b", 64))
	if w := f.request("driver", "sketch-asset", f.a.Key, fmt.Sprint(epoch), strings.NewReader(body)); w.Code == 200 {
		t.Fatal("asset outside frozen source was served")
	}
}

func TestSketchHTTPDelayedABABeginAndDisabledAnnotationSpendNoBody(t *testing.T) {
	f := newSketchHTTPFixture(t)
	old := f.grant(t, f.a)
	f.grant(t, f.b)
	f.grant(t, f.a)
	probe := &sketchReadProbe{reader: strings.NewReader(`{}`)}
	if w := f.request("driver", "sketch-begin", f.a.Key, fmt.Sprint(old), probe); w.Code != 403 || probe.reads != 0 {
		t.Fatal("old A body was admitted after A->B->A")
	}
	if err := f.bd.SetAnnotationAllowed(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	probe = &sketchReadProbe{reader: strings.NewReader(`{}`)}
	if w := f.request("driver", "sketch-begin", f.a.Key, fmt.Sprint(f.ctrl.DrawingAuthority().Epoch), probe); w.Code != 403 || probe.reads != 0 {
		t.Fatal("disabled annotation spent body")
	}
}

func TestSketchHTTPStrictCommandAndBodyBudget(t *testing.T) {
	f := newSketchHTTPFixture(t)
	epoch := f.grant(t, f.a)
	base := fmt.Sprintf(`{"operation":"begin","sequence":0,"epoch":%d,"seg":1,"color":0}`, epoch)
	for _, bad := range []string{strings.TrimSuffix(base, "}") + `,"owner":"operator"}`, strings.TrimSuffix(base, "}") + `,"epoch":1}`, strings.Replace(base, `"seg":1`, `"seg":null`, 1)} {
		w := f.request("driver", "sketch-begin", f.a.Key, fmt.Sprint(epoch), strings.NewReader(bad))
		if w.Code != 400 {
			t.Fatalf("malformed command %d %s", w.Code, w.Body.String())
		}
	}
	w := f.request("driver", "sketch-begin", f.a.Key, fmt.Sprint(epoch), strings.NewReader(strings.Repeat(" ", board.MaxSketchBody+1)))
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("body cap status %d", w.Code)
	}
	for _, r := range f.mem.Records() {
		if r.Kind() == "sketch-begin" {
			t.Fatal("malformed command emitted a begin")
		}
	}
}

func TestSketchHTTPAnnotationRevokedWhileReadingBody(t *testing.T) {
	f := newSketchHTTPFixture(t)
	epoch := f.grant(t, f.a)
	body := fmt.Sprintf(`{"operation":"begin","sequence":0,"epoch":%d,"seg":1,"color":0}`, epoch)
	probe := &sketchReadProbe{reader: strings.NewReader(body), before: func() {
		if err := f.bd.SetAnnotationAllowed(context.Background(), false); err != nil {
			t.Fatal(err)
		}
	}}
	if w := f.request("driver", "sketch-begin", f.a.Key, fmt.Sprint(epoch), probe); w.Code != 403 {
		t.Fatalf("annotation revocation status %d %s", w.Code, w.Body.String())
	}
}

func TestSketchHTTPExpiredSeatAndInvalidSourceRange(t *testing.T) {
	f := newSketchHTTPFixture(t)
	epoch := f.grant(t, f.a)
	v := f.begin(t, f.a, epoch, "begin")
	body := fmt.Sprintf(`{"operation":"read","sequence":0,"epoch":%d,"sketch":%q,"source":%q,"offset":%d,"length":4}`, epoch, v.Sketch, v.Source.Handle, v.Source.Length+1)
	if w := f.request("driver", "sketch-source", f.a.Key, fmt.Sprint(epoch), strings.NewReader(body)); w.Code != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("source range %d %s", w.Code, w.Body.String())
	}
	if err := f.ctrl.Close(); err != nil {
		t.Fatal(err)
	}
	probe := &sketchReadProbe{reader: strings.NewReader(`{}`)}
	if w := f.request("driver", "sketch-status", f.a.Key, fmt.Sprint(epoch), probe); w.Code != http.StatusGone || probe.reads != 0 {
		t.Fatal("retired credential spent metadata body")
	}
}

func TestSketchHTTPHeldAssetReadDoesNotOutliveSource(t *testing.T) {
	for _, retirement := range []string{"label", "expiry"} {
		t.Run(retirement, func(t *testing.T) {
			f := newSketchHTTPFixture(t)
			epoch := f.grant(t, f.a)
			hash := strings.Repeat("a", 64)
			if err := f.w.Put(&recording.Asset{URL: "http://x/private.png", Hash: hash, MIME: "image/png", Size: 7}); err != nil {
				t.Fatal(err)
			}
			v := f.begin(t, f.a, epoch, "begin")
			entered, allow := make(chan struct{}), make(chan struct{})
			f.s.sketchAsset = func(string, int, int) ([]byte, int, error) { close(entered); <-allow; return []byte("PRIVATE"), 7, nil }
			body := fmt.Sprintf(`{"operation":"asset","sequence":0,"epoch":%d,"sketch":%q,"source":%q,"asset":%q,"offset":0,"length":7}`, epoch, v.Sketch, v.Source.Handle, hash)
			done := make(chan *httptest.ResponseRecorder, 1)
			go func() {
				done <- f.request("driver", "sketch-asset", f.a.Key, fmt.Sprint(epoch), strings.NewReader(body))
			}()
			select {
			case <-entered:
			case <-time.After(time.Second):
				t.Fatal("asset read not reached")
			}
			if retirement == "label" {
				if got := f.post(t, "sketch-label", f.a, epoch, map[string]any{"operation": "label", "sequence": 1, "sketch": v.Sketch, "label": "Saved"}); got.Outcome != board.SketchAccepted {
					t.Fatal(got)
				}
			} else {
				if err := f.s.sketches.Expire(context.Background(), time.Now().Add(3*time.Minute)); err != nil {
					t.Fatal(err)
				}
			}
			close(allow)
			w := <-done
			if w.Code == 200 || strings.Contains(w.Body.String(), base64.StdEncoding.EncodeToString([]byte("PRIVATE"))) {
				t.Fatalf("retired source returned private bytes: %d %s", w.Code, w.Body.String())
			}
		})
	}
}
