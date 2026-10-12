---
id: 000457
status: codecomplete
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-12
estimate_hours:
card_mirror: 'c072d01247067b6de6f516a1b9d915cc05af4eb5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-11T23:18:56-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 1.03
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

Durable plan: `workshop/plans/000457-broadcast-web-recording-format-and-component-seams-contracts-fixtures-reference-reducer-plan.md`. Format spec: `atlas/webcast-format.md`.

- [x] Format and node-id scheme (atlas page)
- [x] Records and codec
- [x] Reference reducer and canonical state
- [x] Validator, directory layout, catch-up
- [x] Recorder-side seams and fakes (Sink, Live, FakeHub, FakeRecorder, live contract)
- [x] assets / notes / query interfaces with fakes and contract suites
- [x] Fixtures from #453's prototype and the blog (fixturegen), canonical pins
- [x] Operator review of the format (approved 2026-10-12, no changes); close

## Log

### 2026-10-11
- Claimed at pair:3 from the TL's dispatch (ops:0). #454 landed meanwhile (PR #252); merged main, so the seam names `webcast.LaunchOptions`/`Server.events` directly.
- #453's prototype runs were gitignored, so fixtures are re-recorded: prototype B in the scratchpad, extended with layout (`DOMSnapshot`), click/focus capture, `frameNavigated` and asset bodies. Chrome 154 headless. Blog page: `/2026/10/ariadne-1-building-blocks-of-ai-coding` on the dev server.
- Found for #458: the prototype's 100ms observer poll loses a click that navigates (the page unloads first); `DOMSnapshot.captureSnapshot` on the top session leaves out OOPIF documents (layout for cross-origin frames needs a per-session capture).
- `change-code` inferred the quick flow (design under 500 lines). The code is far past the shell's 100 lines, so close runs the full review. Pinning the flow is the operator's call, so I left it.
- Built: `recording` (records and codec, a strict atomic reducer, Canonical, Validate/ValidateDir, CatchUp, DirWriter/ReadDir, Builder, Sink/Live, MemSink/FakeHub/FakeRecorder, RunLiveContract), and `assets`/`notes`/`query` (interface, fake, contract suite each). The live contract was mutation-checked: registering a subscriber outside the catch-up lock fails it.
- Fixtures: the converter emits only what a live reducer accepts. 0 records dropped across 9.6k raw lines; regeneration is byte-identical. Sizes: stress 1.5 MB, blog 2.0 MB, app 0.5 MB.
- Plan revisions (goldens → pins; resume = same document): see the plan's Revisions.
- Not done here, by design: the fidelity render of fixtures against `frames/` belongs to the viewer (#460) and the recorder (#458). Both now have reference screenshots and pins to test against.

### 2026-10-12
- 2026-10-12: closed — Advisory fixes after the SHIP close: CheckCatchUp (page, assets, DocSeg, live notes) is used by TestCatchUpEquivalence and by every fixture pin; writeFileAtomic for every DirWriter body (PutAsset and PutFrame); plan tables corrected in place. go test ./cmd/internal/webcast/... and ./cmd/internal/artifactpath pass after merging main. (fixed-then-ship); review verdict: SHIP
- 2026-10-12: closed — Operator approved the format 2026-10-12. Review round 1 (BR-1..BR-4) fixed, each with a test that fails when the fix is reverted: catch-up keeps the document segment (DocSeg and live notes equivalent at every cut and every fixture pin); sheets claim their ids and kids reuse only ids under the replaced children; Sink carries records only, bodies have one writer through AssetWriter (atomic), Live refuses frames (in RunLiveContract); deterministic join test at every point. Fixtures validate with zero findings and their pins match. make -k test: all suites pass except a Lua timing test (comment_codec linear-time) that failed under load and passes 3/3 alone (the branch touches no nvim code); then go test ./... -count=1 exit 0 with no failures.; review verdict: SHIP
- 2026-10-12: flow upgraded quick → full — 4456 added lines in code files (limit 100); an earlier round of this close already ran the full review
- The operator reviewed `atlas/webcast-format.md` and approved it as is (in the pair:3 pane: "go ahead, approved").
- Full `make -k test` green after merging main (#456): exit 0, 247 Go packages ok. The first run had found two misses of mine, now fixed: the artifact inventory and a `.gitignore` entry for the `fixturegen` main package. The couchcore deadline test that also failed in that run passes 3/3 alone, and the branch doesn't touch couchcore.

