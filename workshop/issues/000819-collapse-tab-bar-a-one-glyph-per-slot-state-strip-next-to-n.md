---
id: 000819
status: codecomplete
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours: 1.25375
collaboration-mode: collaboration
card_mirror: '3973c22a8a7bfa08ffd3f34a050aac27262e2f78' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T16:54:59-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 0.36
---

# Collapse tab bar: a one-glyph-per-slot state strip next to +N

## Problem

Collapsed counts hide the checkout-state evidence operators can see on full chips. Expose one compact marker per hidden live slot while retaining the default one-row layout and exact group navigation.

## Spec

Operator request (10-22, after accepting pair#796's collapse tab bar): next to a collapsed group's `+N`, show **one
glyph per hidden slot** for its state, so the worktrees' activity reads at a glance without opening the menu.
The operator's sketch (using ⌥ for the branch glyph): `pair⌥  +9 ⌥⌥+-⌥*--⌥`, nine glyphs for nine hidden slots.

- Reuse the state glyphs the full chips already show (e.g. the branch glyph for a slot on an issue branch, the
  attention/bell and other existing markers); one cell each, in slot order; no new state vocabulary.
- Width: the strip counts toward the collapse planner's width budget (grapheme-correct, per pair#796's width fix); if
  it doesn't fit, fall back to `+N` alone (attention still forces `!`).
- Clicking anywhere on `+N` or its strip opens that group's menu (whole-chip hitbox).
- Live slots only, consistent with pair#796's counts.

## Done when

- A collapsed group renders one glyph per hidden live slot in order, using the existing chip glyphs (test).
- When the strip doesn't fit, the chip falls back to `+N` with no clipping mid-glyph (test).
- The strip is part of the clickable chip (physical-cell click test).

## Plan

- [x] Derive one compact existing checkout-state marker per exact hidden live member in slot order.
- [x] Admit complete strips only into spare width after the shared base packing; draw/hit-test count and strip together with existing owner styling.
- [x] Verify projection, width, physical clicks, privacy/geometry, pending/live and counted-work bounds; document the remaining operator-stage and recorded-suite landing gates in the Log.

Detailed design is on this issue branch at workshop/plans/000819-folded-slot-state-strip-plan.md; TL approved the plan (c4f6db71); change-code gates precede implementation.

## Log

### 2026-10-22 — planning under authorized parallel queue
- 2026-10-22: closed — Regression-first compact mapping and exact membership; focused collapse, physical count/separator/strip clicks and owner styles PASS0.517s; full couchtty including native geometry PASS51.541s; artifactpath13.452s, repolint14.741s, reader inventory and whitespace checks PASS. README/atlas updated. Exact-source recorded suite lease-2835455686 is queued; TL5f6b7266 authorizes close overlap while required staged operator look waits behind753. Both remain mandatory landing gates.; review verdict: SHIP

Claim/start-plan from clean main09c8964de39cdb5e6bd9db20601c232446af6949, independent of789. TL62d7df8e authorizes819 planning while789 isolated suite waits, returning for789 fixed review/merge on its result. Current789 lease-2426150993/session41982 remains untouched; no merge runner active. start-plan refresh hit the known seeded merge-check workflow change (ariadne557); saved /Volumes/TestScratch/tmp/pair-slot4/pair819-seeded-wpn5rb4w/merge-check.diff, restored only that generated file per existing TL instruction and retried. No workflow change is included here.

Initial issue-show ran against the older resting checkout and appeared empty. After start-plan caught up, origin/main and local details both contain TL's original sketch/spec; no missing-origin filing repair is needed. TLebda3dbd clarified checkout state and exact single-marker priority, neutral dot, retained dimming, whole-chip navigation and +N fallback. The durable plan recommends optional decoration after base packing, retaining the single measured renderer and confirmed click route (ARCH-DRY/PURE). Alternatives are recorded. No code changed; full-flow design awaits TL review before change-code. Estimate intentionally not derived before plan-quality.

## Revisions

### 2026-10-22 — state meaning and single-cell priority (TLebda3dbd)

Checkout state only, not agent idle/working. Reuse existing SlotGlyph evidence; choose dirty, diverged, ahead/behind, then issue-branch marker, in that priority. Clean/resting/unknown becomes a neutral dot. One cell for each hidden live slot in slot order; exact count membership, owner dimming and full count/strip group navigation remain. If a complete strip cannot fit, use +N alone; existing hidden-attention ! remains. This clarifies the original attention/bell example: attention actors stay explicit under796 and do not enter hidden strips.

### 2026-10-22 — TL plan approval (c4f6db71)

Approved optional strip allocation after base packing, left-to-right using spare columns: each strip fits whole or the group remains plain +N; detail never displaces active/attention/counts/controls. One hit span, unchanged multiline. Proceed to change-code; the real staged operator look remains required before landing. #789 retained snapshot suite is independent and will be resumed for its fixed review/merge on result.

### 2026-10-22 — Plan-quality refinement

PQ-1 requested function-specific adversarial test strategies instead of prose case inventories. Compressed the durable plan accordingly; no product or architecture change. First gate refused before implementation; rerunning after the refinement. #789 suite remains independently queued.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; the source is marked stale, so calibration is provisional. No novel library or external API: existing glyphs, packer, styles and confirmed group navigation supply the seams. Familiarity 1.0 follows the just-landed #796 implementation.

Primitive decomposition: compact glyph/membership extension is a smaller Go module (raw design0.15 × thorough-spec0.2 =0.03; raw impl0.50 ×0.40 =0.20). Packing/rendering/navigation regressions form one TUI screen extension (raw design1.25 ×0.2 =0.25; raw impl1.00 ×0.40 =0.40). Docs use raw design0.125 ×0.2 =0.025 and impl0.20 ×0.40 =0.08. One atomic review/staged verification chunk uses raw design0.10 ×0.2 =0.02 and impl0.50 ×0.40 =0.20. Design subtotal0.325, buffered15%=0.37375; implementation0.88; total1.25375h. Queue wait is not implementation effort.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.03 impl=0.20
item: tui-screen design=0.25 impl=0.40
item: atlas-docs design=0.025 impl=0.08
item: milestone-review design=0.02 impl=0.20
design-buffer: 0.15
total: 1.25375
```

### 2026-10-22 — implementation and targeted proof

change-code passed plan-quality after PQ-1 compression and accepted the reconciled estimate (INFO: verification execution may exceed the allowance). Bounded core mapper work observed RED for missing CompactSlotGlyph, then PASS0.337s across generated states, malformed inputs and terminal-cell oracle. TTY regressions observed semantic RED: missing drawn strip and empty hidden membership. Implemented the pure closed mapper, exact hidden-member derivation and one shared counted width measurement; post-packing optional decoration never changes base actor/count choice. Existing count span extends through all physical digits, separator and markers. Existing owner/glyph styles remain the source.

Focused collapse/detail/owner/click tests PASS0.517s. Full couchtty package including native geometry PASS51.541s. Artifactpath PASS13.452s and repolint PASS14.741s; affected reader inventory and diff whitespace checks pass. Documentation updated in README and atlas. No timeout, production timer, persisted state or multiline change. Full recorded suite, close review and staged operator look remain pending; #789 snapshot suite is preserved independently.

### 2026-10-22 — review / suite overlap, operator stage pending

Pushed candidate3ece1cc55496b71e5071f6de9eb1e43fab49376b has recorded suite session46229, attempt107b78df6d26c5cb3a9ac02ef8d2145e, lease-2835455686 in immutable snapshot /Volumes/TestScratch/local-checks/sdlc-local-2612811728, initially FIFO16. Existing789 run lease-2426150993 remains queued and untouched. TL5f6b7266 requests close review now; pair:0 retains753 stage until its operator re-look. Stage819 only after TL go; operator acceptance and an exact-source recorded pass remain mandatory before landing. Checkbox wording separates completed implementation proof from those external landing gates; none is claimed complete.

### 2026-10-22 — SHIP, waiting on stage window and recorded suite

Close review round1 SHIP, no findings; evidence9d25b6df22496ce4cfbacde1c67a8c0ac461b44b, mirror757ed9cd238af166ef7fe83689039f8bd8745f4a. Reviewer independently passed full couchcore279.273s, couchtty49.645s, reader inventory and pinned diff checks. PR550 is open. Required exact-source suite session46229 / lease-2835455686 remains queued FIFO15;789 session41982 / lease-2426150993 remains FIFO11. Neither is canceled or requeued. No merge runner is active.

Next: on TL stage go after753 releases pair:0, verify main ancestry and stage/build through SDLC, report exact binary revisions and the requested operator look. Do not land until operator acceptance and exact-source recorded pass. On789 suite result, checkpoint819 and return789 for its prescribed fixed review and merge; never switch while a review/stage/merge owns the checkout. Both suite snapshots are independent of checkout switching. Stage has not been touched.
