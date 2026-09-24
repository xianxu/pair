---
id: 000307
status: done
deps: [pair#306]
github_issue:
created: 2026-09-22
updated: 2026-09-23
estimate_hours: 3.36
started: 2026-09-23T16:46:34-07:00
flow: {kind: full, provenance: operator}
actual_hours: 1.01
---

# Slots v2: group switcher and tab bar

## Problem

Multiple threads are useful only if the operator can immediately recognize their repo grouping and select the intended workspace.

## Spec

Project: `pair/workshop/projects/couch-slots-v2.md`. Fresh task derived from the current v2 contract; historical task bodies are not prerequisites or implementation plans.

Group primary and slots together in both switcher and tab bar using one ordering derivation. Switcher shows full names and actual paths: pair /path/to/main, then indented pair:1 /path/to/worktree/pair-slot1, pair:2 ... . Tab bar shows pair :1 :2 brain ariadne ... . :0 denotes the primary but its normal display remains repo. Sort slot numbers numerically; retain stable workspace identity for selection/actions rather than parsing displayed text.

Preserve existing lifecycle/state, focus, notification, and navigation behavior. Define rendering when the primary has no visible thread, when slots are parked, and when width is constrained: group context must remain understandable. Keep full address available where shorthand would be ambiguous. ARCH-DRY: switcher and bar consume the same grouped order; no unrelated visual redesign.

### Agreed scope — 2026-09-23

This section takes precedence over earlier conflicting layout or policy text.

Update the agreed switcher examples to `pair /workspace/pair`, indented `pair:1 /workspace/worktree/pair-slot1/pair`, and `pair:2 /workspace/worktree/pair-slot2/pair`. Tab labels remain `pair :1 :2 brain ariadne ...`. Show the actual main checkout path, not only its enclosing environment directory. Primary repositories retain their own direct Couch entries; ordinary dependency clones inside numbered environments do not automatically appear as additional repos/slots in the switcher or tab bar. Operators access those dependencies through the parent thread; no new dependency-management UI is required.

### Proposed display details — 2026-09-23

Shared pure UI projection groups by existing primary checkout repo scope (slots map through their verified primary root), sorts repo groups alphabetically with deterministic path tie-breaks, and sorts slot numbers numerically. Grouped rows retain canonical workspace labels; custom names remain supplementary and searchable. Different checkouts with the same repo name remain distinct and get path qualifiers.

Switcher shows all existing actionable members, including parked and addressless recovery slots, with full labels and slot indentation. Tabs retain attached/pending membership; parked slots do not gain tabs. Without a primary tab the first visible slot is `pair:1`, followed by `:2`. Missing primary creates no synthetic action. Existing left-to-right clipping remains; full switcher labels preserve context after filtering or scrolling.

Both consumers use the same projection; selection remains keyed by `ThreadRowKey`, terminal actions by native address. Reattachment schedule, notifications, focus and storage are unchanged. The detailed plan records consumer integration and sequence tests. These display details are proposed for operator review, not yet approved.

## Done when

- Switcher shows grouped full labels and actual checkout paths with slot indentation; tab bar shows repo followed by :N labels.
- :2 sorts before :10; input record order cannot split a group or change navigation order unexpectedly.
- Keyboard/click selection activates the correct workspace after refresh/reordering; no selection relies on label parsing.
- Tests cover absent primary, parked members, multiple repos, narrow widths, and existing notification/focus states.
- Operator help and screenshots or rendered fixtures show the agreed examples.

- Rendered and activation fixtures use nested main-checkout paths and exclude incidental dependency clones from automatic thread/slot listings.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Calibration source is stale (ledger newer), so provisional. One issue/design unit, one pure projection extending the existing module, two existing TUI consumers, two acceptance harness extensions, docs and one close review. The three smaller-go-module rows are, in order: pure projection; deterministic golden fixture generation/comparison; disposable multi-workspace activation harness and full/race/performance/build verification. Tests specific to each UI belong to its tui-screen row; acceptance rows cover cross-consumer evidence. Atlas-docs covers README/atlas/project only. Existing Go sorting, repo-scope, rowtext and console harness provide the needed libraries; no novel external API. TUI design uses 0.2 spec discount on 1.5h per consumer; projection and acceptance harness units use the smaller-go-module range (0.2 × 0.3h design and 0.4 × 0.5h implementation). Implementation uses v3.1's 40% scale; familiarity 1.0. Thorough approved plan uses 15% design buffer.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.5 impl=0.04
item: smaller-go-module design=0.06 impl=0.2
item: tui-screen design=0.3 impl=0.4
item: tui-screen design=0.3 impl=0.4
item: smaller-go-module design=0.06 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: atlas-docs design=0.05 impl=0.08
item: milestone-review design=0.1 impl=0.2
design-buffer: 0.15
total: 3.36
```

## Plan

Detailed proposed plan: [Grouped Thread Display](../plans/000307-slots-v2-grouped-thread-display-plan.md). Operator approved on 2026-09-23; change-code gates in progress. The original outline below remains the issue-level acceptance checklist.

- [x] Specify shared group ordering and absent-primary/narrow-width presentation.
- [x] Wire both UI projections and selection/navigation to the shared order.
- [x] Verify rendered examples plus real activation routing.

## Log

### 2026-09-22 — fresh v2 task

Created from the agreed workspace/UI contract and the request for a clean task breakdown. Implementation has not started; estimates follow design approval.

### 2026-09-23 — #307 claimed and implementation mapped
- 2026-09-23: closed — Full repository suite passed before localized review fix; final affected packages pass (couchtty 6.967s, artifactpath 3.128s), full UI race passes (19.996s), complete-target malformed variants pass native-only Enter/click routing including addressless refusal; three-workspace Run-loop click trial passes, rendered fixtures and allocation bounds pass, make pair bin/couch and vet pass. First review typed-target-validation finding fixed across explicit malformed kinds and native mismatch; first close was stale due concurrent Ariadne project progress, preserved in current HEAD. Optional M2 timing harness fails identically at raw-frame correlation on unchanged baseline; no latency-pass claim. Evidence /tmp/pair307-review-fix-{ui2,race2}.log, /tmp/pair307-full.log, /tmp/pair307-baseline-performance.log.; review verdict: SHIP

Ran claim and start-plan after #306 merged. Switcher currently retains inventory order; status tabs use attachment order followed by placeholders. Mapped both consumer paths and confirmed keyboard selection already uses stable slot row keys. Prepared a single-boundary implementation plan using a pure shared UI projection (ARCH-DRY/ARCH-PURE); no runtime code changed. Plan approval is pending.

## Revisions

### 2026-09-23 — Show nested main paths without inventing dependency slots

Reason: operator agreed nested environments, ordinary remote dependency clones and existing per-repository publication. Delta: added the authoritative scope clarification and acceptance criteria above; original task context remains as provenance. No implementation or lifecycle-status change is claimed by this revision.

### 2026-09-23 — consume durable slot row identity from #306

Reason: #306 separates directory identity from the replaceable native conversation.
Delta: group `ThreadTarget`/`ThreadRowKey` by repository and slot number; retain
host-path row identity across refresh and fresh conversation. Addressless recovery
rows remain selectable through `open-slot`/`fresh-slot`. Native scope/tag remains
the process/terminal lookup key. #306 supplies these functional rows, while this
issue still owns shared ordering, indentation and grouped tab labels.

### 2026-09-23 — proposed shared presentation design

Reason: #306 interfaces are now available and #307 is next. Delta: added explicit proposed group ordering, naming, absent-primary, parked-tab membership and width rules, plus a durable implementation plan; retained the original outline and prior scope decisions. No new lifecycle states or persistence are proposed.

### 2026-09-23 — review corrected primary identity

Reason: ordinary StartingPath can name a subdirectory, not a checkout root. Delta: proposed grouping now uses existing repo scopes, derives stable names independently of mutable labels, and preserves label disambiguation for legacy primary conversations sharing a checkout. The plan adds production sequence tests for these cases.

### 2026-09-23 — operator approval and gate clarification

Operator approved the detailed plan. PQ-1 is addressed by the plan's explicit known-root/legacy display-path and qualifier contract; source rows and operation identities remain unchanged. Renamed the plan to the exact issue stem for gate discovery. Implementation has not begun.

### 2026-09-23 — grouping integrated

Shared pure projection, switcher inventory ordering/rendering and grouped tab assembly implemented. Regression tests first reproduced unsorted switcher rows, hidden canonical slot names and attach-ordered tabs; the targeted tests now pass. Full couchtty suite passes (7.040s), including its unchanged allocation budget. Native tab clicks are exercised through Run and activate the intended pane. Rendered fixtures cover normal, absent primary, parked, filtered and narrow views. Broader verification and close review remain.

### 2026-09-23 — acceptance verification

Full repository suite passed (`/tmp/pair307-full.log`, couchcore 248.017s). The subsequent accepted-inventory chrome repaint fix is covered by the full affected UI suite (7.033s), build/vet and a three-workspace temporary-directory activation trial (five repetitions under race). Normal/absent-primary/parked/filtered/narrow golden fixtures use production renderers. `make pair bin/couch`, affected-package vet and diff whitespace checks passed. The unchanged allocation tests pass; 1,000-row projection benchmark measured 1.559ms/op. Optional target-machine timing integration times out on both the working code and an unmodified HEAD snapshot at raw-frame correlation; recorded as a pre-existing harness limitation, with no latency-pass claim. Final UI race result and boundary review follow.

Final affected-package race verification: `go test -race ./cmd/internal/couchtty -count=1` passed in 20.852s (`/tmp/pair307-race-verified.log`).

### 2026-09-23 — first close feedback fixed

The boundary reviewer identified incomplete enclosing-target validation. Added a red regression across malformed typed-target variants, then centralized full-target normalization at projection/ingestion: native identity owns fallback selection and Enter/click routing; malformed addressless targets cannot dispatch. Targetless legacy snapshots remain compatible. The first close also refused finalization because Ariadne updated the shared project while review ran; no stale verdict was accepted, and the peer progress entry is retained. Affected normal/race tests and rebuilt binaries are checked before the next close.

Post-review verification: affected packages passed (couchtty 6.967s; artifactpath 3.128s), full UI race passed (19.996s), rebuilt pair/couch and vet/diff checks passed.
