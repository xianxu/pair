package assets

import (
	"mime"
	"net/http"
	"strings"
	"sync"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Proxy serves a cast's asset bodies to its viewers at <cast>/assets/<hash>
// (the viewer's resolveAsset), so a viewer never reaches the product's
// servers (Principle 4). Its store is bound when the recording opens, at
// the first go-live; until then every asset is not found.
//
// A stylesheet is served with its URLs pointing at the proxy: a URL bound
// to an asset becomes that asset's hash, relative to assets/, and one not
// bound yet stays absolute, for the viewer to resolve when it arrives
// (render.js keeps the proxy's own URLs and resolves the rest). Bindings
// change as assets arrive, so stylesheets are not cached; every other body
// is immutable under its hash.
type Proxy struct {
	mu    sync.RWMutex
	store Store
}

// Typed is a store that knows each body's MIME (Recorded). The proxy serves
// only what it can type.
type Typed interface {
	TypeOf(hash string) (string, bool)
}

// assetPolicy is every asset response's CSP: a body opened on its own (an
// SVG) runs nothing and loads nothing but the cast's own assets.
const assetPolicy = "default-src 'none'; img-src 'self' data:; font-src 'self' data:; style-src 'self' 'unsafe-inline'; sandbox"

func NewProxy() *Proxy { return &Proxy{} }

// Bind sets the store bodies are served from.
func (p *Proxy) Bind(s Store) {
	p.mu.Lock()
	p.store = s
	p.mu.Unlock()
}

// Serve answers a GET for assets/<hash>.
func (p *Proxy) Serve(w http.ResponseWriter, r *http.Request, hash string) {
	p.mu.RLock()
	s := p.store
	p.mu.RUnlock()
	typed, _ := s.(Typed)
	if s == nil || typed == nil || !recording.ValidHash(hash) {
		http.NotFound(w, r)
		return
	}
	typ, ok := typed.TypeOf(hash)
	if !ok || !servable(typ) {
		http.NotFound(w, r)
		return
	}
	body, err := s.Get(hash)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h := w.Header()
	h.Set("Content-Security-Policy", assetPolicy)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Type", typ)
	if baseType(typ) == "text/css" {
		h.Set("Cache-Control", "no-store")
		body = []byte(RewriteCSS(string(body), "", func(abs string) string {
			if a, ok := s.Lookup(abs); ok {
				return a.Hash
			}
			return abs
		}))
	} else {
		h.Set("Cache-Control", "private, max-age=31536000, immutable")
	}
	w.Write(body)
}

// servable: images, stylesheets and fonts, the bodies a drawn page loads.
// Anything else a recorded page could label its responses (HTML, script)
// is never served from the viewer's origin.
func servable(typ string) bool {
	t := baseType(typ)
	switch {
	case t == "text/css", strings.HasPrefix(t, "font/"):
		return true
	case strings.HasPrefix(t, "image/"):
		return true
	}
	switch t {
	case "application/font-woff", "application/font-woff2", "application/x-font-woff", "application/x-font-ttf",
		"application/x-font-otf", "application/x-font-opentype", "application/font-sfnt", "application/vnd.ms-fontobject":
		return true
	}
	return false
}

func baseType(typ string) string {
	t, _, err := mime.ParseMediaType(typ)
	if err != nil {
		return ""
	}
	return t
}
