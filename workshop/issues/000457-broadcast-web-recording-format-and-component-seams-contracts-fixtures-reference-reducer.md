---
id: 000457
status: open
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '36be90e60e9811626f246f052af8eda008e219b1' # card fields mirrored from issue-cards; edit via sdlc
---

# broadcast-web: recording format and component seams (contracts, fixtures, reference reducer)

## Problem

couch-broadcast-web (ops `teams/devinfra/projects/couch-broadcast-web/project.md`) has decided its recorder base: our own on CDP plus a small isolated-world observer (pair#453, PR #251; the operator, 10-11). Five components are to be built in parallel next: recorder, asset store and proxy, live viewer, notes model, agent queries. They can only run in parallel if the contracts between them exist first. The recording format is also the product's core asset: what agents read.

## Spec

Define the contracts; build none of the components.

1. **Recording and stream format**, versioned, one shape for the live stream and the on-disk recording: a recording directory (see the project's format sketch: snapshot, events, notes, pointer, assets, frames) and JSON Lines records. Cover: segments (a new full snapshot per go-live, per new document, per tab switch); per-frame snapshots and DOM changes with our own stable node ids (not CDP's session-scoped ids), including cross-origin iframes and shadow roots; observer events (form values, scroll offsets, selection, CSS rule text); navigation, viewport, tab switch; asset references by content hash. Leave room for a compact encoding later (#453 measured 1.15–2× rrweb's volume); the logical shape comes first.
2. **A reference reducer in Go:** apply a recording to get the DOM (and observer state) at time t. It is the executable spec of the format, and the notes model and agent queries build on it.
3. **A validator** for recordings.
4. **Component interfaces in Go, each with a stateful fake:** the recorder's output sink (live fan-out and file writer), the asset store (put/get by content hash, resample policy), the notes model's entry points (anchor resolve, focused context, crop), the query surface; and the viewer's contract (the JS side: apply snapshot, apply changes, resolve asset URLs). Name where the recorder plugs into pair#454's `webcast.Cast` (today's stub stream).
5. **Fixtures:** convert #453's prototype recordings (branch `453-recorder-prototypes`) into the format for its test pages, plus a page of the operator's blog; check them in with their reference screenshots.

Write the format as an atlas page (`atlas/webcast-format.md`).

## Done when

- The format doc, the Go types, the reference reducer and the validator exist, with tests; every fixture validates and reduces.
- Each component interface has a fake and a test; the seam into `webcast.Cast` is named.
- Each wave-1 issue (pair#458–#462) can start from this without asking: its interface, fake and fixtures are named in the atlas page.
- The operator reviewed the format doc before close (the format is the product): ask the TL (ops:0) to arrange it.

## Plan

- [ ] Format and node-id scheme (atlas page)
- [ ] Go types, validator, reference reducer
- [ ] Interfaces and fakes for each component; seam into webcast.Cast
- [ ] Fixtures from #453's prototypes and the blog
- [ ] Operator review of the format; close

## Log

### 2026-10-11
