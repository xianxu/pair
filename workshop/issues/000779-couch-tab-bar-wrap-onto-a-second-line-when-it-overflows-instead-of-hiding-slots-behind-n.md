---
id: 000779
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours: 5.814
collaboration-mode: collaboration
card_mirror: '51b64acb2f6e7d4ce42d6666a04472c2e965f8c3' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T16:42:14-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Couch tab bar: wrap onto a second line when it overflows, instead of hiding slots behind +N

## Problem

## Spec

Operator report (10-21 13:08, ops session file item 12, screenshot `ops/assets/session/2026-10-21.13-08-21.150.jpg`):
Couch's tab bar ran out of space. With ops, ariadne, blog, pair, parley, you-decide, ops-echo and echo groups and their
slots, the row overflows and the tail collapses into "+3", so slots are hidden.

Wrap the tab bar onto a **second line** when it doesn't fit (like text wrapping), instead of truncating to "+N":
- Wrap at group boundaries where possible (keep a repo's slots together); break inside a group only if one group alone
  is wider than the line.
- At most two lines; if two lines still don't fit, keep today's "+N" overflow on the second line's end.
- The Chrome/cast status group stays where it is (left of line one).
- The rest of the layout (panes, footer) shrinks by the extra line only when the second line is in use.
- Resizing the terminal re-wraps without flicker; the active tab stays visible.

## Done when

- Render tests: fits on one line (unchanged), needs two lines (wraps at a group boundary), one oversized group, and
  overflow beyond two lines (+N), at several widths; active tab always visible.
- Pane geometry accounts for the second line (no overlap), with a test.
- Attention chips take priority over ordinary chips; any hidden attention is explicit in +N !. At physically impossible widths, the active chip remains clipped but clickable and a non-fitting whole overflow marker is omitted.
- Both-row input/anchors and broadcast LIVE/privacy/annotation guards derive from the same accepted chrome height, including resize and private switcher frames.

## Plan

- [x] Deliver the approved two-row layout through shared compositor, Console and broadcast geometry, with render/interaction/native pins; see workshop/plans/000779-tab-bar-wrap-plan.md.

## Log

### 2026-10-21

Claimed and start-plan completed in pair:4 after pair#757 landed (PR532). Includes the manual #757 archive repair, ariadne#532 class: commit a2d91a471ac3bb0b10984194b0990fbde8fdb71f moves two unchanged browser evidence logs from noncanonical plan.md.side into workshop/history/plans/000757-mock-to-real-plan.side. Both renames are100%; no content changes. TL receipt c1d5a9d6-7e31-44f6-a670-5823fcadadf7 explicitly approves riding with this issue and requires this note in the PR description.

Initial read-only audit: one-row assumptions exist in couchtty rendering/geometry/input/anchors, terminal composition/presenter resizing, and broadcast LIVE/pointer checks, private-frame filtering and annotation guards. This leaves the quick-flow code shell; prepare one atomic full-flow plan with pure packing and a shared trusted chrome-height contract before implementation. No product code written. Geometry and packing audits delegated read-only under AGENTS section3; plan approval will go through the hub. Private #757 GOCACHE removed; new Go calls use the repaired machine default on main SSD.

Implementation checkpoint: full-flow change-code CLEAN; estimate review INFO is nonblocking. Qualification is included in Console/API fixtures and the two reserved operator UX rounds.

Implemented measured two-row packing with row-aware spans and priority overflow; rectangular trusted Frame.ChromeRows/Presenter.ChromeState; Console atomic layout and pending revision maps; both-row input/anchors; trusted broadcast/viewer guards; and native PTY resizing. Existing publication mutex now also protects confirmation comparison through the Console hit-map read (ARCH-ORDER): without it a fully drawn Codes menu could lose a click between Presenter admission and Console commit. The existing Codes test reproduced repeated close failures; fixed Codes/pending-writer/stale-revision race stress passed20 repetitions in17.147s. No production polling, deadline changes or new workers.

Observed RED pins include old one-row child size23 instead of22, two-row Compose refusal, unconfirmed wheel delivery, spoofed child LIVE, stale row anchoring, wrong JS guards and wide control overlap. Controlled-writer coverage admits pending extents, rejects clicks while partial writes persist, keeps keys/paste, and confirms the same revision after automatic repaint. Retained menus reflow against the composed frame. Old synthetic test frames now declare their trusted chrome rows, and control helpers locate line1 rather than the physical final line.

Verification: full couchtty package PASS52.455s; terminal package PASS43.848s and focused race1.727s; broadcast package PASS48.606s and race145.241s; browser Node100/100PASS; seeded packing fuzz74,478 executions plus counted64/128 growth pins. Native Claude/Codex-labelled real PTYs PASS0.502s and race1.799s, checking selected/background kernel sizes80x7 ->18x6 ->80x7 and physical bottom output. Inventory and exhaustive artifact-source classification PASS. Direct exploratory package runs were stopped diagnostically after identifying obsolete test oracles; these are not recorded suite evidence. Final recorded check still required.

TL b07bf583 authorizes suite and close meanwhile; pair:0 stage remains held until #752 lands, then TL will choose #779's short look versus #737's day. No staging or landing before that decision and live acceptance.

Recorded check: affected on pushed3e86e8f202a00e7329c21859f6234ef5f9e578eb fell back to suite because the fresh package graph lacks generated embedded runtime files before its build. Retained suite lease1753927196, attemptde0d49c262ff59ee38784bd76a5e30a2, terminal22260; queued normal FIFO position10. This is a queued check, not a pass. Boundary review may overlap under tl-worker; successful recorded evidence and operator look remain landing preconditions. PR535 includes the explicit manual #757 archive-repair note.

Close round1 REWORK has one documentation-only BR-1 (core-concepts-code-traceability): provisional planner names in Core concepts/verification tables did not match statusPacking/renderStatusBar/packStatusChips. Verified delivered symbols, corrected both tables, appended the plan revision and added the prevention lesson. No runtime finding or source change; the retained recorded suite on3e86e8f2 remains valid. Re-run close; operator stage is still pending TL sequence.

## Revisions

- 2026-10-21: TL c5c04f63 approves full-flow shared trusted chrome height, row1 attention badge, final-row notices and clipped-but-clickable active at absurd widths. Attention must be prioritized over ordinary chips; overflow explicitly marks hidden attention (+N !). Durable plan records the full consumer sweep and awaits TL approval.

- 2026-10-21: TL a01bc13f approves the durable plan and authorizes change-code. Stage is explicitly sequenced after #752 lands and the hub clears pair:0; TL will coordinate with #737's live day. No code/estimate yet; full-flow plan-quality gate precedes the derived estimate.

- 2026-10-21: PQ round1 found admitted-versus-painted geometry and tiny active/control priority underspecified. TL ba876ba6 approves revisioned pending/confirmed chrome input admission and active-first tiny fallback with fail-closed LIVE. Durable plan revised with controlled-writer strategy and named function guards; re-run plan-quality before estimating or coding.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only. Calibration is stale, so provisional. Plan-quality round2 CLEAN preceded derivation. Covers shaping already done, all code/tests/docs, native and operator qualification, review and landing activity; queue idle excluded.

Primitive mapping: issue/spec uses0.8 design without spec discount and0.2 raw impl. Packing and Console each use a TUI screen with1.5 raw design ×0.2 thorough-spec; raw impl0.8/1.0. Terminal trusted metadata/confirmation is a typed-data prototype (0.8 design ×0.2,0.5 raw impl); rectangular geometry propagation is a cross-cutting refactor (0.8 ×0.2,0.5 impl). Broadcast/frame admission is existing API integration (1.0 ×0.2,1.5 impl), and viewer guard interaction is one bounded UI primitive (0.8 ×0.2,0.8 impl). Docs/review use0.2 design ×0.2 and raw impl0.2/0.5. Two operator UX rounds reserve0.3 design and0.3 raw impl each; no spec discount for undecided feedback. Every impl value below is raw ×0.40 exactly once; familiarity1.0 (existing Go/JS/compositor, no new stack). Existing row drawer, Presenter FIFO, stateful fixtures and server contracts are reused; none assumes a missing library or new framework, so no greenfield/library discount claimed. Thorough-plan design buffer15%.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.8 impl=0.08
item: tui-screen design=0.3 impl=0.32
item: typed-data-prototype design=0.16 impl=0.20
item: cross-cutting-refactor design=0.16 impl=0.20
item: tui-screen design=0.3 impl=0.40
item: api-integration design=0.2 impl=0.60
item: tui-screen design=0.16 impl=0.32
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.20
item: ux-rename-iteration design=0.6 impl=0.24
design-buffer: 0.15
total: 5.814
```
