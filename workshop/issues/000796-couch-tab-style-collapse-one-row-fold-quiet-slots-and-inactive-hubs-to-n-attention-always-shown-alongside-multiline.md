---
id: 000796
status: working
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours: 2.593
collaboration-mode: collaboration
card_mirror: '72cf60263a100e63966fbba301b3d96607273ab4' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T13:53:50-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Couch: --tab-style collapse (one row; fold quiet slots and inactive hubs to +N, attention always shown) alongside multiline

## Problem

The tab bar must remain useful across a large fleet without changing child height as chips or notices arrive. The operator prioritizes collapse as the default; multiline remains an explicit choice. Wrap-induced chrome-height transitions are implicated in #810's resize-without-repaint idle-admission stalls.

## Spec

Operator (10-22, after accepting pair#779's two-line wrap): "another way to deal with the width limit is to keep it at
one line but have some intelligent collapsing. for example, the inactive hub's worker slots can be collapsed. slots
without either notification (amber) or blocking (red) can be collapsed to just a number, e.g.
`pair  :1 (red)  :4 (amber)  +4` (indicating 4 more slots hidden). can we make such a mode and selectable from command
line? maybe --tab-style collapse | multiline."

Design:
- `couch --tab-style multiline|collapse` at launch (default `collapse`; `multiline` retains pair#779 behaviour); a runtime toggle
  key is a possible follow-up, not in scope.
- **collapse:** one tab row. Within each repo group, show the active slot, every slot needing attention (amber
  notification, red blocked), with useful :0 anchors when space allows, and fold the rest into `+N`. Quiet
  inactive hubs fold with their quiet workers. Proposed count interaction opens the switcher exactly scoped by
  owner hub and group; no in-row expansion. At impossible widths retain the active/attention priority and mark
  hidden attention in `+N !`; the active chip is never folded. These detailed choices await approval of the plan.
- Reuse pair#779's measured packing, hit-map and chrome-row publication: collapse is a different input to the same
  packer, not a second renderer. The status-text cap (pair#789) applies in both modes.

## Done when

- Bare launch defaults to collapse; explicit multiline remains selectable, with CLI validation and actual Console wiring tested.
- Collapse maintains one chrome row across actor/attention updates and width changes whenever terminal height permits one chrome row; child-height and no-wrap-induced-resize regressions cover #810's mechanism. Physical terminal size changes still resize PTYs.

- With `--tab-style collapse`, a fleet of N slots per group renders in one row: active plus attention slots shown,
  the rest as `+N` (render test), and clicking `+N` opens that exact group in the switcher (test).
- `multiline` is unchanged (existing tests).
- Staged for an operator look.


## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only; calibration is currently stale/provisional per sdlc estimate-source. Current-stack reuse is already available: measured renderer, PresentThreads, menu reducer, FakeHost/FakeRunner and native PTY harness; no greenfield external API or library discovery. Familiarity1.0 (same #779/#789 chrome area). Thorough approved plan earns design x0.2 for implementation primitives and15% design buffer; v3.1 implementation values are40% of v2 table units.

Derivation, before buffer: issue/spec shaping design0.30+impl0.08; style parser/plumbing smaller module raw design0.30 x0.2=0.06, impl0.50 x0.4=0.20; collapse packing TUI raw design1.50 x0.2=0.30, impl1.00 x0.4=0.40; scoped menu TUI raw design1.00 x0.2=0.20, impl0.80 x0.4=0.32; geometry/caller integration cross-cutting raw design0.50 x0.2=0.10, impl0.50 x0.4=0.20; docs design0.10 x0.2=0.02, impl0.15 x0.4=0.06; close/stage review design0.20 x0.2=0.04, impl0.40 x0.4=0.16. Design subtotal1.02; implementation1.42; design buffer0.153; total2.593 focused hours. Machine queue time is excluded; native/default compatibility and operator iteration are the main uncertainty.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.30 impl=0.08
item: smaller-go-module design=0.06 impl=0.20
item: tui-screen design=0.30 impl=0.40
item: tui-screen design=0.20 impl=0.32
item: cross-cutting-refactor design=0.10 impl=0.20
item: atlas-docs design=0.02 impl=0.06
item: milestone-review design=0.04 impl=0.16
design-buffer: 0.15
total: 2.593
```

## Plan

- [ ] Review and approve [durable plan](../plans/000796-collapse-tab-style-plan.md).
- [ ] Implement and verify CLI/style policy, measured collapse packing and exact scoped count navigation.
- [ ] Verify native geometry, safety guards, multiline compatibility and full recorded suite; stage for operator acceptance, close and land.

## Log

### 2026-10-22

Claimed and started from main at TL6a768f10 while #789's isolated integration suite waits. No #789 commits are stacked here. #789 remains branch3306f2d16 with check session61444/lease-3428295058/attempta71b7262e10d47a252d630372095d2ff; return there for full-suite result, fixed review and merge. start-plan hit ariadne#557's generated workflow dirt: saved /tmp/pair796-seeded-merge-check.diff, restored only .github/workflows/merge-check.yml per TLc4e804fc, retried successfully; no seeded gate changes committed.

Read-only survey found existing chipKept keeps every hub and existing aggregate overflow has no click spans; collapse needs mode-specific candidate policy and count spans from the same clipping pass. A fuzzy switcher Filter cannot express exact owner+group identity. Proposed typed scope and count navigation are in the durable plan, awaiting TL approval. No implementation started.

Plan-quality round2 CLEAN: PQ-1 addressed; no open blocking findings. Estimate derived only after this pass.

Plan-quality round1 REWORK/PQ-1: scenario inventories needed named production-function test strategies. Refined parser/extractor/CLI, collapse preparation/packing/drawing, scope filtering/reducer and revision-fenced opener coverage with adversarial inputs and mechanical guards. No policy change; implementation remains gated.

## Revisions

### 2026-10-22 — operator priority and default (TL0afd9a82)

Collapse becomes the default launch style; multiline remains available. Add #810's wrap-induced resize-without-repaint mechanism to the Spec and acceptance tests. This supersedes the initial multiline-default sentence. The single-row policy removes content-driven chrome-height transitions, while real terminal-width changes still resize PTYs; it does not claim to repair every possible idle-admission stall.

### 2026-10-22 — detailed policy proposal, awaiting TL approval

The durable plan selects the original Spec's scoped-switcher alternative rather than in-row expansion, and makes owner+group scoping exact rather than fuzzy label matching. It proposes folding quiet inactive hubs themselves and preserving attention-first scarcity with +N !; these refine the earlier all-:0-visible wording and acknowledge physical width limits. Policy state/recovery, work bounds, multiline compatibility and native geometry tests are specified in the plan. No code until approval.

### 2026-10-22 — TL approval (3fa9a5f6)

TL approved default one-row collapse, explicit unchanged multiline packer, quiet inactive hubs folding, active/attention priority, +N ! fallback at impossible widths, and +N opening the switcher exactly scoped to owner hub and group with Escape clearing scope. Proceed to change-code; the staged operator look remains required before landing. This accepts the detailed policy proposal above.
