package assets

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// memWriter is an AssetWriter that can't read back, counting writes.
type memWriter struct{ puts atomic.Int32 }

func (m *memWriter) PutAsset(body []byte) (string, error) {
	m.puts.Add(1)
	return recording.HashOf(body), nil
}

func dirWriter(t *testing.T) *recording.DirWriter {
	t.Helper()
	w, err := recording.NewDirWriter(filepath.Join(t.TempDir(), "rec"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { w.Close() })
	return w
}

func TestRecordedContract(t *testing.T) {
	t.Run("dir", func(t *testing.T) {
		RunStoreContract(t, func() Store { return NewRecorded(dirWriter(t), DefaultPolicy) })
	})
	t.Run("no reader", func(t *testing.T) {
		RunStoreContract(t, func() Store { return NewRecorded(&memWriter{}, DefaultPolicy) })
	})
	t.Run("fake", func(t *testing.T) { RunStoreContract(t, func() Store { return NewMemStore() }) })
}

func TestRecordedResamplesOncePerBody(t *testing.T) {
	w := &memWriter{}
	s := NewRecorded(w, Policy{MaxPixels: 100 * 100})
	big := encodePNG(t, noisy(400, 300, true))
	ctx := context.Background()
	a, err := s.Put(ctx, Capture{URL: "http://x/a.png", MIME: "image/png", Body: big})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := s.Put(ctx, Capture{URL: "http://x/copy.png", MIME: "image/png", Body: big})
	if a.Hash != b.Hash || w.puts.Load() != 1 || b.URL != "http://x/copy.png" {
		t.Errorf("the same body under two URLs is stored once: %+v %+v puts=%d", a, b, w.puts.Load())
	}
	if a.MIME != "image/jpeg" || a.Orig == nil || a.Orig.W != 400 {
		t.Errorf("binding %+v", a)
	}
	body, _ := s.Get(a.Hash)
	cfg, _, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil || cfg.Width*cfg.Height > 100*100 {
		t.Errorf("stored %dx%d: %v", cfg.Width, cfg.Height, err)
	}
	if typ, _ := s.TypeOf(a.Hash); typ != "image/jpeg" {
		t.Errorf("TypeOf %q", typ)
	}
}

func TestRecordedAbsolutizesCSS(t *testing.T) {
	w := dirWriter(t)
	s := NewRecorded(w, DefaultPolicy)
	ctx := context.Background()
	css := []byte(`@import "base.css"; body{background:url(../img/bg.png)} a{b:image-set("x.png" 1x)}`)
	a, err := s.Put(ctx, Capture{URL: "http://x/css/site.css", MIME: "text/css; charset=utf-8", Body: css})
	if err != nil {
		t.Fatal(err)
	}
	want := `@import "http://x/css/base.css"; body{background:url("http://x/img/bg.png")} a{b:image-set("http://x/css/x.png" 1x)}`
	got, err := w.Asset(a.Hash) // read back from disk: DirWriter.Asset
	if err != nil || string(got) != want {
		t.Errorf("stored %q (%v)", got, err)
	}
	// The same bytes from another directory mean other URLs: another body.
	b, _ := s.Put(ctx, Capture{URL: "http://x/other/site.css", MIME: "text/css", Body: css})
	if b.Hash == a.Hash {
		t.Errorf("a relative sheet loaded from two places must store two bodies")
	}
	if _, err := s.Get(recording.HashOf([]byte("never"))); err != ErrNotFound {
		t.Errorf("unknown hash: %v", err)
	}
}

func TestRecordedRefusesBomb(t *testing.T) {
	s := NewRecorded(&memWriter{}, DefaultPolicy)
	if _, err := s.Put(context.Background(), Capture{URL: "http://x/b.png", MIME: "image/png", Body: pngHeader(20000, 20000)}); err == nil {
		t.Errorf("a bomb must not be stored")
	}
	if _, ok := s.Lookup("http://x/b.png"); ok {
		t.Errorf("a refused body must not be bound")
	}
}

// TestRecordedDecodesOneAtATime: concurrent captures of one image are
// decoded once (the decode slot is process-wide, and the store is checked
// again after waiting for it).
func TestRecordedDecodesOneAtATime(t *testing.T) {
	w := &memWriter{}
	s := NewRecorded(w, Policy{MaxPixels: 100 * 100})
	big := encodePNG(t, noisy(400, 300, true))
	var wg sync.WaitGroup
	hashes := make([]string, 8)
	for i := range hashes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a, err := s.Put(context.Background(), Capture{URL: fmt.Sprintf("http://x/%d.png", i), MIME: "image/png", Body: big})
			if err != nil {
				t.Error(err)
				return
			}
			hashes[i] = a.Hash
		}()
	}
	wg.Wait()
	if n := w.puts.Load(); n != 1 {
		t.Errorf("one body decoded and stored %d times", n)
	}
	for _, h := range hashes {
		if h != hashes[0] {
			t.Errorf("hashes differ: %q", hashes)
		}
	}
}

// TestRecordedWaitHonoursContext: a capture waiting for the decode slot
// gives up with its context.
func TestRecordedWaitHonoursContext(t *testing.T) {
	decoding <- struct{}{}
	defer func() { <-decoding }()
	s := NewRecorded(&memWriter{}, Policy{MaxPixels: 10})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := s.Put(ctx, Capture{URL: "http://x/a.png", MIME: "image/png", Body: encodePNG(t, noisy(40, 30, true))}); err == nil {
		t.Errorf("a Put waiting for the decode slot must end with its context")
	}
	// A font needs no decode and doesn't wait.
	if _, err := s.Put(context.Background(), Capture{URL: "http://x/f.woff2", MIME: "font/woff2", Body: []byte("f")}); err != nil {
		t.Error(err)
	}
}

func TestRecordedFirstTypeWins(t *testing.T) {
	s := NewRecorded(&memWriter{}, DefaultPolicy)
	a := put(t, s, "http://x/a.woff2", "font/woff2", "same bytes")
	b := put(t, s, "http://x/a.html", "text/html", "same bytes")
	if typ, _ := s.TypeOf(a.Hash); typ != "font/woff2" || b.MIME != typ {
		t.Errorf("a body's type moved to a later label: served as %s, recorded as %s", typ, b.MIME)
	}
}
