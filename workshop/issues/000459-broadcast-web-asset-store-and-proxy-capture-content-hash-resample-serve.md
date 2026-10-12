---
id: 000459
status: working
deps: [000457]
github_issue:
created: 2026-10-11
updated: 2026-10-12
estimate_hours:
card_mirror: '6b9a7e0b82548d233a1da928d76b3f3628446d71' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T02:21:10-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "e6177353", done: "ff10eccd"}
---

# broadcast-web: asset store and proxy — capture, content hash, resample, serve

## Problem

Wave 1 of couch-broadcast-web (ops `teams/devinfra/projects/couch-broadcast-web/project.md`; read its Direction, Development approach and Principles). Depends on pair#457 (recording format and component seams): build against its format, Go interfaces, fakes and fixtures, not against the other wave-1 components, which are being built at the same time. Your component sits behind its #457 interface; test it against fakes and the fixtures, and add contract tests the other side can run. If the format or a seam is wrong for you, don't change it locally: report it to the TL (ops:0), who revises #457's contract. Browser runs are headless and never take focus from Couch. The real test app is the operator's blog (`~/workspace/xianxu.dev`, `npm run dev`).

## Spec

Implement #457's asset store interface and the proxy:
- Capture: response bodies for images, CSS and fonts the cast page loads (CDP `Network`/`Fetch`), keyed by content hash, stored per recording.
- Resample: images keep their declared size and are resampled to a resolution budget, so a recording's size is bounded; CSS and fonts exact.
- Rewrite: URLs inside CSS (`url()`, `@import`, `@font-face`) and in the stream (`src`, `srcset`, `href` of stylesheets) point at the proxy.
- Serve: over pair#454's shared broadcast endpoint to viewers; viewers never reach the product's servers.
- Mocks and apps come from a local server; `file://` is out of scope.

## Done when

- The blog's assets (Tailwind CSS, the Inter variable font, images) are deduplicated by hash, resampled, and served through the real store and proxy; a viewer page renders the blog fixture with no request outside its own origin.
- Tests against the fixtures and the store contract; a size bound test for resampling.
- (TL 10-12) Capture over CDP is #458's (it calls `Store.Put`); the live blog cast is checked once #458 merges or in the joint first-live-cast session, not as this issue's gate.

## Plan

Durable plan: `workshop/plans/000459-broadcast-web-asset-store-and-proxy-capture-content-hash-resample-serve-plan.md` (single pass, one close).

- [x] CSS URL scanner (Go) + shared corpus; `image-set()` strings in the viewer's `css.js`
- [x] Resample (budget, bomb ceiling, WebP via golang.org/x/image)
- [x] `assets.Recorded` (the real Store) over `AssetWriter`; `DirWriter.Asset`
- [x] `assets.Proxy` + the Server's `assets/<hash>` route
- [x] Fixtures through the real store/proxy in the live viewer tests; no-origin check; atlas
- [x] Full verification

## Log

### 2026-10-11

### 2026-10-12

- Claimed from pair:4 (TL ops:0). #458 (unlanded, M2 closed) already captures bodies over CDP and calls `Store.Put`, and creates the store lazily at first go-live (`CastConfig.NewAssets`). Asked the TL; decisions: capture stays #458's; #459 = real Store + `assets.Proxy` bound later; whichever of #458/#459 lands second wires `proxy.Bind` and couchcmd's `NewAssets`; the store absolutizes CSS URLs at `Put` (dedup by hash makes a serve-time base ambiguous), atlas to say so; budget only (no declared size yet); live blog check after #458 merges.
- The blog (Astro) serves WebP from its image pipeline, so resampling needs a WebP decoder: `golang.org/x/image` (Go project; v0.47.0 on 10-09), pinned v0.46.0, with a DecodeConfig pixel ceiling before any decode.
- `golang.org/x/image` v0.46.0 needs `golang.org/x/sys` v0.48.0, so MVS bumped x/sys from v0.44.0.
- Built: `assets.RewriteCSS` (one scanner, two directions: absolutize at Put, point at `<hash>` at serve), `Resample`, `assets.Recorded` (real store; runs `RunStoreContract` over a DirWriter, a reader-less writer and the fake), `assets.Proxy` (Server owns it, `Server.Assets().Bind(store)`), `DirWriter.Asset` (read back; shares `Recording.Asset`'s reader). The viewer's `css.js` missed plain strings in `image-set()`; fixed, and both scanners are held to `tests/webcast/css_urls.json` (each mutation-checked: dropping the image-set branch fails 4 Go cases / 2 node tests).
- Discovery: the live viewer tests stood in for the proxy with a regex over the fixture bodies, which hid that the blog fixture's 1738×2130 screenshot is over the budget. The real store resamples it to a new hash. In production the recorder emits the store's asset record, so the tests now do the same (`throughStore` replaces each fixture asset record with the store's; canonical state holds no asset table, so pins hold).
- Live (headless Chrome 154, `WEBCAST_LIVE_CHROME=1`): replay fidelity unchanged (blog 0.22–2.00%, 2.27% at +150 ms on frame 2; app ≤0.79%; stress ≤1.66%); boxOf 215/217; the blog viewer made 82 requests, 7 through the proxy (stylesheet, Inter woff2 and images all among them), none outside the cast (`about:invalid` is css.js's BLOCKED). Unbinding the store fails the test.

