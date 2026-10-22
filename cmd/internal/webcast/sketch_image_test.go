package webcast

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"github.com/xianxu/pair/cmd/internal/webcast/sketchimage"
)

func imageBatch(t *testing.T, n int) sketchimage.Batch {
	t.Helper()
	b := recording.NewBuilder("image")
	node := b.El("div", []string{"id", "original", "style", "background:#ffffff;width:600px;height:400px"})
	b.Page(recording.SegLive, "https://source.test/", node)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("s%d", i)
		op := func(seq uint64) recording.SketchOperation {
			return recording.SketchOperation{Operation: fmt.Sprintf("%s-%d", id, seq), Sequence: seq, Digest: strings.Repeat("a", 64)}
		}
		at := int64(10 + i*10)
		b.At(at).Add(&recording.SketchBegin{SketchOperation: op(0), ID: id, Author: recording.Author{ID: "operator", Color: "#ff0000"}, Owner: "operator", Doc: 1, Tab: "tab1", VP: recording.Viewport{W: 1280, H: 800, DPR: 1}})
		// Same timestamp must not widen the begin prefix.
		b.Add(&recording.Attr{ID: node.ID, Name: "id", V: recording.Str("too-late")})
		b.At(at + 1).Add(&recording.SketchStrokeBegin{SketchOperation: op(1), Sketch: id, ID: "p", At: recording.NoteAt{Node: node.ID, Box: &recording.Box{0, 0, 600, 400}}, Width: 12})
		b.At(at + 2).Add(&recording.SketchPoints{SketchOperation: op(2), Sketch: id, ID: "p", Pts: []recording.SketchPoint{{40, 40, 0}, {90, 40, 0}, {140, 40, 1}}})
		b.At(at + 3).Add(&recording.SketchStrokeEnd{SketchOperation: op(3), Sketch: id, ID: "p"})
		b.At(at + 4).Add(&recording.SketchLabel{SketchOperation: op(4), Sketch: id, Label: "Move this"})
	}
	views, err := query.New(b.Recs, nil).SketchesAt(b.T)
	if err != nil {
		t.Fatal(err)
	}
	return sketchimage.Batch{RecordingDir: t.TempDir(), Records: b.Recs, Sketches: views}
}
func TestSketchImageServerFreezesExactBeginAndConfinesAssets(t *testing.T) {
	b := imageBatch(t, 1)
	plans, err := sketchimage.ValidateBatch(b)
	if err != nil {
		t.Fatal(err)
	}
	s, err := serveSketchImages(b, plans)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	res, err := http.Get(s.base + "/0/records")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if bytes.Contains(raw, []byte("too-late")) || !bytes.Contains(raw, []byte("original")) {
		t.Fatalf("source widened past begin: %s", raw)
	}
	// The server must never read a replacement events file.
	os.WriteFile(filepath.Join(b.RecordingDir, "events.jsonl"), []byte("replacement"), 0600)
	res, err = http.Get(s.base + "/0/records")
	if err != nil {
		t.Fatal(err)
	}
	second, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !bytes.Equal(raw, second) {
		t.Fatal("reread recording events")
	}
	res, err = http.Get(s.base + "/0/sketch-image.html")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(res.Header.Get("Content-Security-Policy"), "default-src 'none'") {
		t.Fatalf("unsafe page headers: %v", res.Header)
	}
}
func TestSketchImageAssetsRejectEscapingSymlinkAndWrongHash(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "assets"), 0700)
	body := []byte("private")
	sum := sha256.Sum256(body)
	hash := hex.EncodeToString(sum[:])
	outside := filepath.Join(t.TempDir(), "private")
	os.WriteFile(outside, body, 0600)
	if err := os.Symlink(outside, filepath.Join(dir, "assets", hash)); err != nil {
		t.Fatal(err)
	}
	cache, err := newImageAssetCache(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.close()
	if _, err := cache.get(hash); err == nil {
		t.Fatal("read escaping recording asset")
	}
	os.Remove(filepath.Join(dir, "assets", hash))
	os.WriteFile(filepath.Join(dir, "assets", hash), []byte("changed"), 0600)
	if _, err := cache.get(hash); err == nil {
		t.Fatal("served body under another content hash")
	}
}
func TestSketchImageRendererCanceledBeforeChrome(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if images, err := NewSketchImageRenderer("missing-chrome").Render(ctx, imageBatch(t, 1)); err == nil || images != nil {
		t.Fatalf("canceled renderer: %v", err)
	}
}

func TestSketchImageOpenReadyCancellation(t *testing.T) {
	// Launch failure must join all owned state rather than leave a profile.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := launchHeadless(ctx, "/missing/chrome"); err == nil {
		t.Fatal("missing browser succeeded")
	}
}

func TestLiveSketchImageHistoricalPressureAndBatch(t *testing.T) {
	bin := liveFidelityChrome(t)
	b := imageBatch(t, 8)
	renderer := NewSketchImageRenderer(bin)
	started := time.Now()
	images, err := renderer.Render(context.Background(), b)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("eight historical sketches rendered and cleaned up in %s", time.Since(started))
	if time.Since(started) > 30*time.Second {
		t.Fatal("batch exceeds existing submission preparation deadline")
	}
	plans, err := sketchimage.ValidateBatch(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := sketchimage.ValidateImages(plans, images); err != nil {
		t.Fatal(err)
	}
	im, err := png.Decode(bytes.NewReader(images[0].PNG))
	if err != nil {
		t.Fatal(err)
	}
	p := plans[0]
	redAt := func(x, y int) bool { r, g, bl, _ := im.At(x, y).RGBA(); return r > 50000 && g < 18000 && bl < 18000 }
	at := func(x, y float64) (int, int) {
		return int(math.Round((x - p.Region[0]) * p.Scale)), int(math.Round((y - p.Region[1]) * p.Scale))
	}
	x, y := at(140, 44)
	if !redAt(x, y) {
		t.Fatal("full-pressure endpoint missing")
	}
	x, y = at(40, 44)
	if redAt(x, y) {
		t.Fatal("pressure flattened: light endpoint as thick as full pressure")
	}
	// Exact boundary is independently inspected through the actual browser API.
	server, err := serveSketchImages(b, plans)
	if err != nil {
		t.Fatal(err)
	}
	defer server.close()
	browser, err := launchHeadless(context.Background(), bin)
	if err != nil {
		t.Fatal(err)
	}
	profile := browser.profile
	if err := browser.openReady(context.Background(), server.base+"/0/sketch-image.html", "sketchImage"); err != nil {
		t.Fatal(err)
	}
	var sourceID string
	if err := browser.eval(context.Background(), `document.querySelector('iframe:not([hidden])').contentDocument.querySelector('#original')?.id || 'missing'`, &sourceID); err != nil {
		t.Fatal(err)
	}
	if sourceID != "original" {
		t.Fatalf("rendered later same-time DOM: %q", sourceID)
	}
	if err := browser.closeJoined(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(profile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("browser profile remains: %v", err)
	}
	sketchimage.RunRendererContract(t, renderer, b)
}

func TestSketchImageAssetsOnlyFromBeginPrefix(t *testing.T) {
	b := imageBatch(t, 1)
	if err := os.Mkdir(filepath.Join(b.RecordingDir, "assets"), 0700); err != nil {
		t.Fatal(err)
	}
	makeAsset := func(body string, at int64) *recording.Asset {
		sum := sha256.Sum256([]byte(body))
		hash := hex.EncodeToString(sum[:])
		if err := os.WriteFile(filepath.Join(b.RecordingDir, "assets", hash), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		return &recording.Asset{Header: recording.Header{T: at}, URL: "https://source.test/style.css", Hash: hash, MIME: "text/css", Size: int64(len(body))}
	}
	early := makeAsset("body{color:red}", 9)
	late := makeAsset("body{color:blue}", 10)
	index := b.Sketches[0].Source.RecordIndex
	records := append([]recording.Record(nil), b.Records[:index]...)
	records = append(records, early, b.Records[index], late)
	records = append(records, b.Records[index+1:]...)
	b.Records = records
	var err error
	b.Sketches, err = query.New(records, nil).SketchesAt(14)
	if err != nil {
		t.Fatal(err)
	}
	plans, err := sketchimage.ValidateBatch(b)
	if err != nil {
		t.Fatal(err)
	}
	s, err := serveSketchImages(b, plans)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	for _, check := range []struct {
		hash   string
		status int
	}{{early.Hash, 200}, {late.Hash, 404}} {
		res, err := http.Get(s.base + "/0/assets/" + check.hash)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != check.status {
			t.Fatalf("asset %s: %d want%d", check.hash, res.StatusCode, check.status)
		}
	}
}

func TestLiveSketchImageCancellationJoinsChrome(t *testing.T) {
	bin := liveFidelityChrome(t)
	b := imageBatch(t, 8)
	scratch := t.TempDir()
	t.Setenv("TMPDIR", scratch)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		images, err := NewSketchImageRenderer(bin).Render(ctx, b)
		if images != nil {
			done <- errors.New("cancellation published partial images")
			return
		}
		done <- err
	}()
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	sawBrowser := false
	for !sawBrowser {
		select {
		case err := <-done:
			t.Fatalf("render ended before cancellation: %v", err)
		case <-deadline.C:
			t.Fatal("Chrome did not acquire its profile")
		case <-tick.C:
			entries, _ := os.ReadDir(scratch)
			for _, entry := range entries {
				if _, err := os.Lstat(filepath.Join(scratch, entry.Name(), "SingletonLock")); err == nil {
					sawBrowser = true
					break
				}
			}
		}
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation: %v", err)
		}
	case <-deadline.C:
		t.Fatal("renderer did not join canceled Chrome")
	}
	entries, err := os.ReadDir(scratch)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("canceled browser retained resources: %v", entries)
	}
}
