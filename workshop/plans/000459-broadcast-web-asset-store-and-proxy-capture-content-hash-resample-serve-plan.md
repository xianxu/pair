# broadcast-web asset store and proxy — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The real `assets.Store` (dedup by hash, images resampled to a budget, CSS absolutized, persisted through the
recording's `AssetWriter`) and `assets.Proxy`, which serves `<cast>/assets/<hash>` to viewers with CSS `url()`s rewritten
to the proxy, so a viewer never requests the product's origin.

**Architecture:** Capture stays #458's: its recorder fetches bodies over CDP (`Network.getResponseBody`,
`Page.getResourceContent`) and calls `Store.Put` (TL, 10-12). #459 is what sits behind that call and in front of the viewer.
One pure CSS URL scanner serves both directions: absolutize at `Put` (against the capture URL) and point at the proxy at
serve time (absolute URL → `<hash>`, relative to `assets/`). The `Server` owns a `Proxy`; the cast binds the store to it
when the recording opens (lazily, at first go-live). The JS viewer gets the same `image-set()` fix, and both scanners
are held to one shared corpus.

**Tech Stack:** Go (stdlib `image/png|jpeg|gif`, `golang.org/x/image` v0.46.0 for `webp` decode and `draw.CatmullRom`
scaling), node:test for `css.js`, headless Chrome for the existing live viewer tests.

TL decisions (ops:0, 10-12): (1) capture stays #458's; #459 = real Store + `assets.Proxy` with a later `Bind`; whichever of
#458/#459 lands second adds the bind in `recordingOutput` and points couchcmd's `NewAssets` at the real store.
(2) absolutize at `Put`, and the atlas says so. (3) budget only (`MaxPixels`); a declared size is a later refinement.
(4) build and test against fakes and fixtures; the live blog cast is checked once #458 merges, or in the joint first-live-cast
session, whichever is first.

**Dependency (vetted):** `golang.org/x/image`: the Go project's own module, latest v0.47.0 released 2026-10-09; pinned at
v0.46.0. It parses untrusted bytes, so the store checks `image.DecodeConfig` against a pixel ceiling before any decode
(a decompression bomb is refused, not decoded). Alternative considered: decode in Chrome over CDP (no Go decoder on
untrusted input), rejected: it needs a live tab per image and puts the store on the recorder's CDP connection.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `RewriteCSS` | `cmd/internal/webcast/assets/css.go` | new |
| `Resample` | `cmd/internal/webcast/assets/image.go` | new |
| `rewriteCSS` (`image-set()` strings) | `cmd/internal/webcast/web/css.js` | modified |
| CSS URL corpus | `tests/webcast/css_urls.json` | new |

- **RewriteCSS(text, base, to)** — one pass over a stylesheet finding every URL a browser would load: `url(…)` (bare,
  `"…"`, `'…'`), `@import "…"`, and plain strings directly inside `image-set()`/`-webkit-image-set()`. Each is resolved
  against `base` (http/https only; `data:`, `#frag`, empty and unparseable kept), passed to `to(abs)`, and replaced by
  `url("<to>")` only when the result differs from the raw text. So an already-absolute sheet stays byte-identical
  (`RunStoreContract`'s exact CSS). Comments are copied untouched.
  - **Relationships:** used by `Recorded.Put` (`to` = identity, absolutize) and `Proxy` (`to` = `Lookup` → `<hash>`, else
    the absolute URL, which the viewer then resolves itself; `render.js` `css(…, own=true)`).
  - **DRY rationale:** the two directions are one scan with different `to`; the JS viewer's scan is held to the same
    corpus, so "which spans are URLs" has one written rule in two languages.
- **Resample(body, mime, maxPixels)** — returns the body exact when it isn't a decodable raster (SVG, AVIF, unknown),
  is within budget, or is an animated GIF; refuses one whose header claims more than `bombPixels` (64 MP). Otherwise
  scales to fit `maxPixels` keeping the aspect ratio (CatmullRom), encodes JPEG q85 when opaque and PNG when not, and
  returns the new MIME and `recording.Orig{Size,W,H}`.
  - **Future extensions:** a declared/rendered size (TL (3)) widens the budget per capture; `Capture.DeclaredW/H` is
    already there.
- **CSS URL corpus** — `[{css, base, urls}]`: the absolute URLs, in order, that a scan must hand to `to`. Read by
  `css_test.go` and `css.test.mjs`.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Recorded` (the real `Store`) | `cmd/internal/webcast/assets/recorded.go` | new | `recording.AssetWriter` (disk) |
| `Proxy` | `cmd/internal/webcast/assets/proxy.go` | new | HTTP to viewers |
| `Server` `assets/<hash>` route, `Server.Assets()` | `cmd/internal/webcast/server.go` | modified | the shared broadcast Host |

- **Recorded** — `NewRecorded(w recording.AssetWriter, p Policy)`. `Put`: context check; a cache keyed by the
  *source* body's hash skips repeat work (same bytes → same stored asset, decoded once); classify by MIME without
  params: `text/css` → `RewriteCSS(body, c.URL, identity)`; `image/*` → `Resample`; else exact. Persist through
  `w.PutAsset`, remember `hash → mime` and `url → asset`. Slow work (decode/encode) runs outside the lock. `Get` reads
  the body back through the writer when it can (`DirWriter.Asset`, a small additive read method), else from memory
  (a writer without a reader, i.e. tests' fakes). `TypeOf(hash)` gives the proxy the stored MIME. Runs
  `RunStoreContract`.
  - **Injected into:** #458's recorder through `CastConfig.NewAssets` (wired by whichever lands second).
  - **Lifecycle (ARCH, at-plan):** `assets/<hash>` files are created by `Recorded` through the recording's
    `DirWriter`, last needed by whoever replays the recording, removed when the operator deletes the recording
    directory (the existing recording lifecycle, atlas/webcast-format.md). Growth per distinct body: an image is at
    most `MaxPixels` (≈1920×1080, under ~1 MB as JPEG q85); CSS and fonts as served. The in-memory maps die with the
    cast.
- **Proxy** — `NewProxy()`, `Bind(Store)`, `Serve(w, r, hash)`. 404 when unbound, for an invalid or unknown hash, or a
  MIME outside the allow-list (images, `text/css`, fonts), so a recorded page can't get script or HTML served from
  the viewer's origin. CSS bodies are rewritten at serve time (bindings change as assets arrive, so `Cache-Control:
  no-store` stays); other bodies are immutable by hash (`private, max-age=31536000, immutable`). Every asset response
  carries `Content-Security-Policy: default-src 'none'; img-src 'self' data:; font-src 'self' data:; style-src
  'unsafe-inline'; sandbox` and `nosniff`. MIME comes from `TypeOf` when the store has it, else 404.
- **Server** — creates its own `Proxy` in `NewServer` (no signature change, so #458's branch keeps merging) and routes
  `<token>/assets/<hash>` to it; `Server.Assets()` is what the cast binds.

## Done when (restated for the plan)

- `Recorded` passes `RunStoreContract`; tests cover dedup across URLs, CSS absolutizing (relative `url()`, `@import`,
  `image-set` strings), resample of a large image to ≤ `MaxPixels` with `orig` (the size bound test), exact SVG and
  animated GIF, and the bomb refusal.
- The proxy serves bodies by hash with CSS rewritten to `<hash>`; refusals and headers are tested.
- The live viewer tests (`TestLiveViewerPage`, `TestLiveReplayFixtures`) serve the fixtures' assets (the blog's
  Tailwind CSS, Inter woff2, images) through `Recorded` + `Proxy` instead of the test stand-in, and
  `TestLiveViewerPage` asserts the viewer made **no request outside its own origin** (CDP `Network.requestWillBeSent`).
- The live blog cast is checked once #458 merges (TL (4)); not a gate on this issue.

## Tasks

### Task 1: CSS URL scanner (Go) and the shared corpus

**Files:** create `cmd/internal/webcast/assets/css.go`, `css_test.go`, `tests/webcast/css_urls.json`.

- [ ] Corpus cases: bare/quoted `url()`, `url( "../a.png" )`, `@import "i.css"`, `@import url(i.css)`,
  `image-set("a.png" 1x, url(b.png) 2x)`, `-webkit-image-set('c.png' 1x)`, `data:` and `#f` kept, a URL inside a
  comment kept, nested parens in `image-set(… type("image/webp"))` (the `type()` string is not a URL).
- [ ] `TestRewriteCSSCorpus` (urls handed to `to` equal the corpus), `TestRewriteCSSExactWhenUnchanged`
  (identity `to` on an absolute sheet returns the input bytes), `TestAbsolutize` (relative → absolute output text).
- [ ] Run `go test ./cmd/internal/webcast/assets -run CSS` → FAIL; implement a single-pass scanner (states: text,
  comment, string, `url(`, `image-set(` depth); → PASS. Commit `#459: CSS URL scanner and corpus`.

### Task 2: `image-set()` in the viewer

**Files:** modify `cmd/internal/webcast/web/css.js`, `tests/webcast/css.test.mjs`.

- [ ] Test: the corpus drives `rewriteCSS` (recording each `resolve(abs)` call) and `image-set("a.png" 1x)` rewrites to
  `image-set(url("assets/aa") 1x)`. `node --test tests/webcast/css.test.mjs` → FAIL.
- [ ] In `rewriteCSS`, a first pass wraps plain strings at the top level of `image-set(`/`-webkit-image-set(` into
  `url("…")`; the existing single pass then rewrites every `url()` once. → PASS. Commit.

### Task 3: Resample

**Files:** `go get golang.org/x/image@v0.46.0`; create `assets/image.go`, `image_test.go`.

- [ ] Tests: 4000×3000 noisy PNG with `MaxPixels` 1920×1080 → stored w×h ≤ budget, aspect within 1px, `orig` =
  {len, 4000, 3000}, output decodes (the size bound test); an opaque one → `image/jpeg`; alpha → `image/png`; within
  budget → exact; SVG → exact; 2-frame GIF → exact; a PNG header claiming 20000×20000 → error; WebP decodes.
- [ ] FAIL → implement → PASS. Commit.

### Task 4: `Recorded`

**Files:** create `assets/recorded.go`, `recorded_test.go`; add `DirWriter.Asset(hash)` in
`recording/dir.go` (+ test).

- [ ] Tests: `RunStoreContract(t, …)` over a `DirWriter` in `t.TempDir()` and over a writer with no reader; a large
  image Put twice under two URLs decodes once (counting fake writer) and shares the hash; relative CSS stored
  absolute; `Get` reads back from disk; `TypeOf` gives the stored (possibly changed) MIME.
- [ ] FAIL → implement → PASS. Commit.

### Task 5: `Proxy` and the Server route

**Files:** create `assets/proxy.go`, `proxy_test.go`; modify `cmd/internal/webcast/server.go`, `server_test.go`.

- [ ] Proxy tests (httptest): unbound → 404; bound, unknown/invalid hash → 404; a font → exact bytes, its MIME,
  immutable cache, the asset CSP, nosniff; CSS → `url()`s of bound assets become `url("<hash>")`, unbound stay
  absolute; rebinding a URL changes the next CSS response; a stored `text/html` or `text/javascript` body → 404.
- [ ] Server test: `GET /tok/assets/<hash>` served after `s.Assets().Bind(store)`; another token's path 404.
- [ ] FAIL → implement → PASS. Commit.

### Task 6: Fixtures through the real store; no-origin check

**Files:** modify `cmd/internal/webcast/replay_test.go`, `viewer_live_test.go`; atlas.

- [ ] A test helper `fixtureProxy(t, rec)` puts every fixture asset body through a `Recorded` (URL and MIME from the
  asset records) and binds a `Proxy`; `fixtureServer` and `viewerServer` serve through it. `serveFixtureAsset` and its
  regex go.
- [ ] `TestLiveViewerPage` enables CDP `Network`, collects `requestWillBeSent` URLs, and fails on any outside the test
  server's origin (other than `data:`/`blob:`).
- [ ] `WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run 'TestLive'` → PASS with the same fidelity numbers.
- [ ] Atlas: `webcast-format.md` Assets paragraph says the store absolutizes CSS at `Put` and the proxy rewrites at serve
  time, including `image-set()` strings (Revisions note); `webcast.md` maps `assets.Recorded`/`Proxy` and the binding
  rule; `artifactpath/manifest.go` classifies the new `.go` files. Commit.

### Task 7: Verify

- [ ] `go test ./cmd/internal/artifactpath ./cmd/internal/webcast/...`, then the full suite per the repo routine
  (`make -k test`, scratchpad-TMPDIR `test-changelog`, `go test ./...`).
