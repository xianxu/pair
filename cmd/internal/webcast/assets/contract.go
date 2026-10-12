package assets

import (
	"bytes"
	"context"
	"errors"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// RunStoreContract checks a Store. The real store (#459) and MemStore both
// run it.
func RunStoreContract(t recording.TB, newStore func() Store) {
	t.Helper()
	ctx := context.Background()
	css := Capture{URL: "http://x/a.css", MIME: "text/css", Body: []byte("body{background:url(http://x/bg.png)}")}
	font := Capture{URL: "http://x/f.woff2", MIME: "font/woff2", Body: []byte("wOF2 not really")}

	s := newStore()
	a, err := s.Put(ctx, css)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if a.URL != css.URL || a.MIME != css.MIME || !recording.ValidHash(a.Hash) {
		t.Fatalf("binding %+v", a)
	}
	body, err := s.Get(a.Hash)
	if err != nil || recording.HashOf(body) != a.Hash || int64(len(body)) != a.Size {
		t.Fatalf("the stored body must hash to its name and size: %v", err)
	}
	// CSS and fonts are exact (Principle 4), and so carry no orig.
	for _, c := range []Capture{css, font} {
		got, _ := s.Put(ctx, c)
		b, _ := s.Get(got.Hash)
		if !bytes.Equal(b, c.Body) || got.Orig != nil {
			t.Errorf("%s must be stored exactly", c.MIME)
		}
	}
	// Same bytes, same hash: the store deduplicates.
	again, _ := s.Put(ctx, Capture{URL: "http://x/copy.css", MIME: "text/css", Body: css.Body})
	if again.Hash != a.Hash {
		t.Errorf("the same bytes under two URLs must share a hash")
	}
	if got, ok := s.Lookup(css.URL); !ok || got.Hash != a.Hash {
		t.Errorf("lookup after put: %+v %v", got, ok)
	}
	// A URL put again with new bytes is rebound.
	re, _ := s.Put(ctx, Capture{URL: css.URL, MIME: "text/css", Body: []byte("body{}")})
	if got, _ := s.Lookup(css.URL); got.Hash != re.Hash || re.Hash == a.Hash {
		t.Errorf("a URL put again must rebind")
	}
	if _, ok := s.Lookup("http://x/never"); ok {
		t.Errorf("lookup of an unknown URL")
	}
	if _, err := s.Get(recording.HashOf([]byte("never"))); !errors.Is(err, ErrNotFound) {
		t.Errorf("get of an unknown hash: %v, want ErrNotFound", err)
	}
	// An image: whatever it stores, orig is set exactly when the body
	// changed, and the binding names the stored body.
	img := Capture{URL: "http://x/p.png", MIME: "image/png", Body: tinyPNG}
	ia, err := s.Put(ctx, img)
	if err != nil {
		t.Fatalf("put image: %v", err)
	}
	ib, err := s.Get(ia.Hash)
	if err != nil || recording.HashOf(ib) != ia.Hash {
		t.Fatalf("image body: %v", err)
	}
	if changed := !bytes.Equal(ib, img.Body); changed != (ia.Orig != nil) {
		t.Errorf("orig must be set exactly when the body was resampled (changed=%v orig=%+v)", changed, ia.Orig)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.Put(cancelled, font); err == nil {
		t.Errorf("put with a cancelled context must fail")
	}
}

// tinyPNG is a valid 1×1 PNG.
var tinyPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0xf8, 0xcf, 0xc0, 0xf0,
	0x1f, 0x00, 0x05, 0x00, 0x01, 0xff, 0x89, 0x99, 0x3d, 0x1d, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45,
	0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}
