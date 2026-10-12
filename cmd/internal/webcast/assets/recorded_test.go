package assets

import (
	"bytes"
	"context"
	"image"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// memWriter is an AssetWriter that can't read back, counting writes.
type memWriter struct{ puts int }

func (m *memWriter) PutAsset(body []byte) (string, error) {
	m.puts++
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
	if a.Hash != b.Hash || w.puts != 1 || b.URL != "http://x/copy.png" {
		t.Errorf("the same body under two URLs is stored once: %+v %+v puts=%d", a, b, w.puts)
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
