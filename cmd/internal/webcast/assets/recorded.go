package assets

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"sync"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Recorded is the real Store (#459): a recording's asset bodies, persisted
// through its AssetWriter (the DirWriter, one file per hash under assets/).
// Images over the policy's budget are resampled; a stylesheet's URLs are
// made absolute against its own URL, so a body shared by URLs in different
// directories means one thing (atlas/webcast-format.md, Assets); fonts and
// everything else are exact.
//
// Bodies are read back through the writer when it can read (DirWriter.Asset),
// so a long cast doesn't hold its images in memory; a writer that can't (a
// test's) has them kept here.
type Recorded struct {
	w      recording.AssetWriter
	read   func(hash string) ([]byte, error)
	policy Policy

	mu     sync.Mutex
	types  map[string]string           // stored hash → stored MIME
	urls   map[string]*recording.Asset // URL → its binding
	done   map[string]*recording.Asset // source body's hash → what it was stored as
	bodies map[string][]byte           // only when the writer can't read
}

// assetReader is a writer that can read back what it stored.
type assetReader interface {
	Asset(hash string) ([]byte, error)
}

// NewRecorded stores through w under policy p.
func NewRecorded(w recording.AssetWriter, p Policy) *Recorded {
	s := &Recorded{w: w, policy: p, types: map[string]string{}, urls: map[string]*recording.Asset{}, done: map[string]*recording.Asset{}}
	if r, ok := w.(assetReader); ok {
		s.read = r.Asset
	} else {
		s.bodies = map[string][]byte{}
	}
	return s
}

// Put stores c and binds its URL. The recorder calls it from one worker,
// so a URL's bindings follow its captures; concurrent Puts of one URL bind
// in the order they finish.
func (s *Recorded) Put(ctx context.Context, c Capture) (*recording.Asset, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key := recording.HashOf(c.Body) + " " + c.MIME
	if c.isCSS() {
		key += " " + c.URL // absolutizing depends on where it was loaded from
	}
	stored, ok := s.stored(key)
	if !ok && isImage(c) {
		// One decode at a time (decoding); a body that arrived twice is
		// decoded once, by whichever got here first.
		select {
		case decoding <- struct{}{}:
			defer func() { <-decoding }()
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		stored, ok = s.stored(key)
	}
	if !ok {
		var err error
		if stored, err = s.store(c); err != nil {
			return nil, err
		}
	}
	a := *stored
	a.URL = c.URL
	s.mu.Lock()
	defer s.mu.Unlock()
	s.done[key] = stored
	s.urls[c.URL] = &a
	return clone(&a), nil
}

// decoding admits one image decode at a time, process-wide (every cast's
// store): at the bomb ceiling (64 MP) a decode holds about 256 MB of
// source pixels, plus a result within the budget (about 8 MB at 1920×1080).
// Concurrent decodes would multiply that.
var decoding = make(chan struct{}, 1)

func (s *Recorded) stored(key string) (*recording.Asset, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.done[key]
	return a, ok
}

// store makes and persists the stored form of one capture; the slow part
// (decode, scale, encode) runs outside the lock.
func (s *Recorded) store(c Capture) (*recording.Asset, error) {
	body, typ := c.Body, c.MIME
	var orig *recording.Orig
	switch {
	case c.isCSS():
		body = []byte(RewriteCSS(string(c.Body), c.URL, func(abs string) string { return abs }))
	case isImage(c):
		var err error
		if body, typ, orig, err = Resample(c.Body, c.MIME, s.policy.MaxPixels); err != nil {
			return nil, err
		}
	}
	hash, err := s.w.PutAsset(body)
	if err != nil {
		return nil, fmt.Errorf("assets: store %s: %w", c.URL, err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if first, ok := s.types[hash]; ok {
		typ = first // the same bytes under another label keep the first, in the record too
	} else {
		s.types[hash] = typ
	}
	if s.bodies != nil {
		s.bodies[hash] = bytes.Clone(body)
	}
	return &recording.Asset{Hash: hash, MIME: typ, Size: int64(len(body)), Orig: orig}, nil
}

func (s *Recorded) Get(hash string) ([]byte, error) {
	s.mu.Lock()
	_, known := s.types[hash]
	body, kept := s.bodies[hash]
	s.mu.Unlock()
	switch {
	case !known:
		return nil, ErrNotFound
	case kept:
		return bytes.Clone(body), nil
	}
	b, err := s.read(hash)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	return b, err
}

func (s *Recorded) Lookup(url string) (*recording.Asset, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.urls[url]
	if !ok {
		return nil, false
	}
	return clone(a), true
}

// TypeOf is the MIME a stored body is served as: the proxy's Content-Type.
func (s *Recorded) TypeOf(hash string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.types[hash]
	return t, ok
}

func clone(a *recording.Asset) *recording.Asset {
	c := *a
	if a.Orig != nil {
		o := *a.Orig
		c.Orig = &o
	}
	return &c
}

func (c Capture) isCSS() bool { return baseType(c.MIME) == "text/css" }

func isImage(c Capture) bool { return strings.HasPrefix(baseType(c.MIME), "image/") }
