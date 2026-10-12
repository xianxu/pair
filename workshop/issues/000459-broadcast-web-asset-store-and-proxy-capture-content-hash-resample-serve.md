---
id: 000459
status: open
deps: [000457]
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '5c3980aa6558fb76b495bce8085f45962d276313' # card fields mirrored from issue-cards; edit via sdlc
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

- The blog's assets (Tailwind CSS, the Inter variable font, images) are captured, deduplicated by hash, resampled, and served; a viewer page renders with no request to the origin.
- Tests against the fake recorder sink and fixtures; a size bound test for resampling.

## Plan

- [ ] Capture via CDP, content-hash store
- [ ] Resample policy; CSS URL rewriting
- [ ] Serve over the broadcast endpoint
- [ ] Tests; live check on the blog

## Log

### 2026-10-11
