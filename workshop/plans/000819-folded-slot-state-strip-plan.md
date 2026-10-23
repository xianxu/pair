# Folded slot state strip implementation plan

**For the implementing agent:** use superpowers-executing-plans and TDD; SDLC owns the one boundary review.

**Goal:** Show one existing checkout-state marker per hidden live slot beside each collapse group count, without sacrificing visible actors, controls, attention or the one-row contract.

**Architecture:** Derive compact markers from the existing SlotGlyph evidence, retain them on each frame-local synthetic count, and add complete strips only after the shared packer selects the base cells. Draw and hit-test the count and strip together using existing ownership shading and confirmed-revision publication.

**Tech stack:** Go; couchcore slot-glyph projection; couchtty measured packer/renderer; existing terminal.Cell, FakeHost, menu and native geometry tests.

## Approved product contract

TL ebda3dbd clarified the operator sketch: `pair⌥  +9 ⌥⌥+-⌥*--⌥`, with ⌥ standing for the existing branch glyph. This is checkout state, never agent working/idle inference. One single-column marker per hidden LIVE member in numeric slot/presentation order. Priority for a two-part glyph is dirty `*`, diverged `±`, ahead `+` or behind `-`, then the existing branch glyph. Neutral `·` covers clean/resting/unknown evidence. The count means hidden live members; its strip has exactly that cardinality when shown.

The three-member threshold, active/attention preservation, launch-hub anchors, parked exclusion, explicit pending placeholders, exact owner/repository identity and explicit multiline policy from #796 remain unchanged. Selecting a hidden slot legitimately changes the hidden membership. Whole +N, separator and strip open the same exact group. No per-marker activation or new panel.

## Design choices

1. Recommended: decorate only the winning base packing with optional complete strips. Existing count/actor selection wins before decoration, so quiet detail cannot hide an actor or a global attention marker. Each group takes available columns left-to-right; if that group's entire strip cannot fit, it keeps plain +N. Later smaller strips may still fit.
2. Rejected: include strips in count labels before packing. Their optional width could force additional actors/counts out and change existing scarcity selection.
3. Rejected: truncate strips or replace the existing packer. A partial strip no longer has one marker per hidden member; a parallel layout would duplicate controls and geometry policy.

Transient notices retain the existing free-space policy: selected actor/count cells, including admitted optional detail, precede a notice using the final row's remaining space. Fixed controls and the attention badge retain their existing reservation. Global mixed-group overflow remains its existing +N / +N ! without a strip. A clipped-active-only fallback receives no decoration.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| CompactSlotGlyph | cmd/internal/couchcore/slotgit.go | new |
| StatusActor | cmd/internal/couchtty/reserve.go | modified: frame-local FoldedStateStrip |
| chipCell | cmd/internal/couchtty/reserve.go | modified: optional admitted strip |
| prepareCollapsedStatusChips | cmd/internal/couchtty/status_collapse.go | modified |
| decorateFoldedCounts | cmd/internal/couchtty/status_layout.go | new |
| renderStatusBar | cmd/internal/couchtty/status_layout.go | modified |
| chipsWidth / measureChipCells | cmd/internal/couchtty/reserve.go | modified wrapper / new counted shared measurement |
| renderStatusCells | cmd/internal/couchtty/reserve.go | modified |

CompactSlotGlyph consumes the existing closed glyph string and emits one allowed single-cell marker. Its implementation uses the existing private branch constant and dirty/diverged constants in their owning package; no duplicate branch-symbol definition and no Git probe. Unknown/empty/malformed strings produce the neutral marker rather than forwarding arbitrary text. CompactSlotGlyph is tested over generated existing glyph grammar and arbitrary malformed strings, enforcing the fixed alphabet, priority and one-terminal-cell invariant.

FoldedStateStrip is data, not an inventory. The existing exact-group pass derives it only from the members actually folded. Its order derives from PresentThreads, not map iteration or a new sort. Kept actors and placeholders contribute no marker. Store a compact string, at most three UTF-8 bytes per hidden member with this fixed alphabet; no retained slice of actors or new durable record.

The existing chipsWidth behavior remains one shared measurement implementation, with an optional counted seam for the new decoration pass. It includes labels, active brackets, glyphs, tails and actual cellGap separators; the caller also accounts for the shared prefix's first gap. The winning base packing's cells include any global overflow marker before decoration. Measure base use once, then visit group-count cells once, admitting a full strip only when `1 + Folded` columns fit the remaining measured budget. Omit the strip otherwise. Never repeatedly rescan a growing prefix.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Rendered count emission | cmd/internal/couchtty/reserve.go | modified | terminal-safe text, glyph colour and complete GroupCountSpan |

Unchanged reuse: PresentThreads and console_presentation.go supply canonical live membership/order/owner; SlotGlyph supplies checkout evidence; slotGlyphBase, FadeStyle and inactiveOwnerStyle supply marker colour and owner shade. Console/Painter publishes the existing GroupCountSpan in the confirmed chrome revision, and terminal_input.go opens the existing scoped menu. No new mouse branch, provider, IO seam, background task or persisted setting.

Draw the existing count label, then the admitted separator and markers. Dirty/diverged markers retain their existing amber base; all tokens retain the group's existing owner dimming and palette fallbacks. Count span ends after the final drawn marker; when no strip is admitted it ends at the base count. The preflight checks complete label plus optional strip against actual remaining width, so no partial marker or phantom span appears.

## State and recovery

Actions: view, resize, change active tab, click count/strip, Escape, ordinary switcher reopen.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Group has fewer than three eligible members | existing individual chips | view / resize | none beyond existing width overflow |
| Folded group and complete strip fits | +N followed by N state markers | click count/strip | secondary two-part checkout marker suppressed by approved priority |
| Strip cannot fit after base packing | existing plain +N | click count / resize | per-slot detail absent until space is available |
| Multiple groups compete for spare columns | earlier fitting strips shown; each non-fitting group stays plain | click count / resize | no visible actor or mandatory count displaced |
| Global scarcity / hidden attention | existing +N or +N ! | click global count / bell | no per-group strip for mixed hidden membership |
| Unknown or clean/resting Git evidence | neutral dot at that slot's position | click group to inspect existing inventory | dot does not claim clean when evidence is unknown |
| Membership/focus changes | same live projection, count and strip recomputed together | view / click after repaint | selecting a hidden member legitimately removes it from both |
| Parked row or pending placeholder | excluded from counts / existing explicit placeholder | existing switcher / wait for attachment | no parked or starting slot falsely counted as live |
| Resize or stalled repaint | existing confirmed frame only is actionable | wait for successful repaint / resize | stale coordinates do not open any group |
| Slot disappears while menu is scoped | existing refreshed/empty exact scope | Escape / ordinary reopen | no widening to another owner or repo |

All new data is immutable frame-local derivation and dies with the frame. There is no new transition owner, retry, cancellation or external-outcome policy; existing Console/Presenter/menu ownership governs those events. Covered: drawing, scarcity, clicks, membership, geometry and scope transitions. Not added: agent-runtime state, configurable glyph language, new Git freshness policy, per-token navigation, partial strips or persisted preferences.

## Constraints and trust

ARCH-DRY/PURE: one checkout-glyph source, one measured packer, one width implementation and existing count click route. ARCH-PURPOSE: deliver cardinality/order, complete fallback, actual whole-chip click and owner styling together. ARCH-MOCK: no new external dependency; reuse FakeHost/fake children and native conformance fixtures. ARCH-SECURE: emit only the fixed marker alphabet; never interpolate arbitrary Git/agent strings as styling or controls. Confirmed geometry remains the admission guard. ARCH-ORDER: no state across frames; existing publication revision and menu reducer own mutable behavior. ARCH-FUNERAL/EVOLVE: no new durable artifact or persisted-format change.

ARCH-CONSTRAINTS: within existing actor/grid limits, A actors, G groups and B existing label/glyph bytes. Marker mapping and measurement add O(A+B) work, at most O(A) transient bytes (three bytes/marker plus per-count separator). No network/disk/process work, no extra concurrency or timer. One row always; a complete strip that exceeds actual spare width is absent. Count actual mapping/measurement/decoration visits in the production statusLayoutWork seam and pin N=128/256, both many small groups and one large group. Preserve existing shared O(B+A log A) planner bounds and allocation budgets; no timeout/budget bumps.

## Tasks (one atomic close boundary)

### Task 1 — compact state evidence and exact folded membership

Files: core slotgit.go/slotgit_test.go; tty reserve.go, status_collapse.go and status_collapse_test.go.

- [x] CompactSlotGlyph: generate existing glyph combinations and arbitrary malformed strings; assert the closed marker alphabet, priority and one-terminal-cell output against an independent oracle, observing red before implementation.
- [x] prepareCollapsedStatusChips: generate inventories through PresentThreads; independently assert exact hidden live cardinality, numeric order, owner/repository identity and preserved-member exclusion under focus changes.
- [x] Implement the pure mapper and frame-local strip derivation, retaining existing grouping and threshold policy.
- [x] Run `go test ./cmd/internal/couchcore -run 'Test(CompactSlotGlyph|SlotGlyph)' -count=1` and focused collapse tests; expect PASS after observed red. Commit.

### Task 2 — spare-width admission, rendering and whole-chip navigation

Files: tty status_layout.go, reserve.go, status_owner_fade_test.go, status_collapse_test.go, collapse_mouse_test.go; focused folded_state_strip_test.go if needed.

- [x] measureChipCells and decorateFoldedCounts: generate widths and grouped base packings; assert unchanged base selection, reserved-control bounds and complete-or-absent strips with left-to-right admission. Observe red before implementation.
- [x] renderStatusCells: generate admitted detail, palette and ownership combinations; assert terminal-cell bounds, existing glyph/owner styling, complete hit spans and unchanged multiline output.
- [x] Decode real physical cells through existing confirmed-frame input: assert every admitted count/strip cell opens exactly its owner/group, stale frames admit nothing, and Escape restores ordinary navigation.
- [x] Implement counted shared measurement, optional admission and complete-strip drawing through existing helpers and group navigation.
- [x] Count actual production visits at N/2N for adversarial group distributions; enforce existing allocation/cost bounds. Run focused tests and the full couchtty package, including its native geometry and guard regressions. Commit.

### Task 3 — document, stage and land

Files: README.md; atlas/couch.md; this plan and issue.

- [x] Document marker priority, neutral uncertainty, slot ordering, whole-count click, all-or-none per-group width fallback and no agent-runtime meaning.
- [x] Run reader/source inventory checks and diff whitespace checks; classify new files only if their actual source reads require it.
- [ ] Commit/push; submit recorded exact-head suite. Close through SDLC's mandatory review; fix any executable finding with red/green evidence.
- [x] Obtain TL's stage window, stage/build through SDLC, verify all binaries' exact revision/modified=false, and request the operator's real look before landing. Never replace pair:0 without a go.
- [ ] After suite/review/operator gates pass, enter sdlc merge promptly; its admitted turn owns main integration. Clear own stage after landing, verify main tip, report PR to hub.

## Parallel #789 recovery

#789 has no merge runner. Its immutable suite on73288d8654a9b9ca426bdf71b765805d72c4e78c remains session41982/lease-2426150993. TL62d7df8e authorizes this independent819 plan from main. On789 suite result, checkpoint819, return789 for the required fixed review and merge; never switch checkout while a new789 merge runner owns it. No commits from789 are stacked here.

### 2026-10-22 — TL plan approval (c4f6db71)

Approved optional strip allocation after base packing, left-to-right using spare columns: each strip fits whole or the group remains plain +N; detail never displaces active/attention/counts/controls. One hit span, unchanged multiline. Proceed to change-code; the real staged operator look remains required before landing. #789 retained snapshot suite is independent and will be resumed for its fixed review/merge on result.

## Revisions

### 2026-10-22 — PQ-1 function-test-strategy

Compressed all task-level test inventories into named risky functions, adversarial input classes and mechanical invariants. The approved behavior, production seams, cost bounds, verification commands and staged operator requirement are unchanged.

### 2026-10-22 — accepted operator stage

TL21cc4008 records operator “works”. Owned stage cleared and verified at current main; only the recorded-suite pass and landing remain pending.
