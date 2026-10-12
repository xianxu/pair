package assets

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// A reopened recording serves what its asset records bind, from disk, and
// takes nothing new.
func TestReopenServesTheRecordingsAssets(t *testing.T) {
	rec, err := recording.ReadDir(recording.Fixture("rec-blog"))
	if err != nil {
		t.Fatal(err)
	}
	s := Reopen(rec)
	p := NewProxy()
	p.Bind(s)
	var css int
	for _, r := range rec.Records {
		a, ok := r.(*recording.Asset)
		if !ok {
			continue
		}
		if typ, ok := s.TypeOf(a.Hash); !ok || typ != a.MIME {
			t.Errorf("%s: type %q %v, want %q", a.URL, typ, ok, a.MIME)
		}
		if got, ok := s.Lookup(a.URL); !ok || got.Hash != a.Hash {
			t.Errorf("%s: bound to %v", a.URL, got)
		}
		b, err := s.Get(a.Hash)
		if err != nil || recording.HashOf(b) != a.Hash {
			t.Errorf("%s: body %v", a.URL, err)
		}
		if baseType(a.MIME) == "text/css" {
			css++
			res := get(t, p, a.Hash)
			if res.StatusCode != http.StatusOK || !strings.HasPrefix(res.Header.Get("Content-Type"), "text/css") {
				t.Errorf("%s: served %d %s", a.URL, res.StatusCode, res.Header.Get("Content-Type"))
			}
		}
	}
	if css == 0 {
		t.Fatal("the blog fixture has no stylesheet asset: the check saw nothing")
	}
	if _, err := s.Put(context.Background(), Capture{URL: "http://x/a.png", MIME: "image/png", Body: []byte("x")}); !errors.Is(err, ErrReadOnly) {
		t.Errorf("Put: %v, want ErrReadOnly", err)
	}
}

// A body missing from the directory is not found, not an error page.
func TestReopenMissingBody(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "rec")
	w, err := recording.NewDirWriter(dir)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := w.PutAsset([]byte("picture"))
	if err != nil {
		t.Fatal(err)
	}
	b := recording.NewBuilder("gone")
	b.Add(&recording.Asset{URL: "http://x/p.png", Hash: hash, MIME: "image/png", Size: 7})
	for _, r := range b.Recs {
		if err := w.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	w.Close()
	rec, err := recording.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	os.Remove(filepath.Join(dir, recording.AssetName(hash)))
	if _, err := Reopen(rec).Get(hash); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get: %v, want ErrNotFound", err)
	}
}
