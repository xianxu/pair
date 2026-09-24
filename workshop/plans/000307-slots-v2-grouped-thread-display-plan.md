# Grouped Thread Display Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development or superpowers-executing-plans. Steps use checkbox syntax for tracking.

**Goal:** Present each repository and its numbered slots together in Couch's switcher and tab bar, with stable selection and shared ordering.

**Architecture:** Derive an ephemeral ordered presentation from existing thread summaries and verified slot identities. Both UI surfaces consume it; native addresses continue routing terminal operations and `ThreadRowKey` continues identifying durable slot rows. No filesystem discovery or lifecycle changes belong in rendering.

**Tech Stack:** Go, existing Couch reducers/renderers, ANSI terminal fixtures and console test harness.

**Status:** Operator approved; implementation complete, acceptance verification recorded below; close review pending.

## Design

### Behavior

- Group by the existing primary checkout repo scope, never basename or display label. A verified slot supplies `PrimaryRoot`: use the existing pure `launcher.ResolveRepoScope(PrimaryRoot).Key` to match ordinary `Address.RepoScope`. Each slot maps to that primary scope; ordinary members use their own scope. Addressless slots retain host-path row identity. Distinct checkouts named `pair` remain distinct groups. No new hash or persisted identity is introduced.
- Derive stable group names independently of mutable `Name` and `WorkingPath`. Slot groups use the verified `Repo` and `PrimaryRoot`. For ordinary groups, recover the checkout root by comparing `launcher.ResolveRepoScope` keys for the absolute `StartingPath` and its lexical ancestors to the existing address scope (bounded by path depth, pure, no filesystem calls); use that root basename. Missing/unmatched legacy paths fall back to the scope as sort key and retain existing display-label behavior. Do not guess that the starting directory itself is a checkout. Attached fallback rows use `pane.tree` as their starting-path candidate and the same native scope; fallback → inventory retains the group.
- Sort groups by case-folded stable repository name, then canonical group key; primary/ordinary members precede numbered members; numbered members sort numerically. Native scope/tag and stable row key break remaining ties. Do not let input order, state, recency, renaming or attachment completion change the order.
- Use the verified slot's repo name for its group. Groups without slots retain existing ordinary labels. In a slot group, primary displays `pair`; numbered rows display `pair:1`, `pair:2`. Preserve custom names as supplementary switcher text and as search inputs, without replacing the workspace address. Keep existing native-address suffix disambiguation for all colliding ordinary labels, including multiple legacy primary conversations inside a slot group; same-name repository groups receive a stable path qualifier on their group label.
- Switcher rows show full workspace labels and actual host checkout paths; numbered rows gain two spaces of indentation. They retain state/age, notifications, selection styling, and existing recovery actions. Missing or filtered-out primary creates no synthetic actionable row: full slot labels remain self-explanatory.
- Tab membership stays as today: attached panes and pending reattachment placeholders. Parked/unattached slots remain visible in the switcher; this work does not start them or add inert tabs for them. Sort tab members through the same group derivation, including placeholders in their proper group position. Reattachment scheduling remains recency-based and independent of display order.
- Within visible tabs, the primary uses the repo label; later slot chips use `:N`. If no primary tab is present, the first visible slot uses `pair:N`, followed by `:N`. Placeholder-to-attached replacement keeps its position and all existing spinner, focus and notification behavior.
- Retain left-to-right width clipping and the switcher's 40x10 minimum. There is no new horizontal scrolling policy. Since clipping removes the right suffix, a shorthand slot can only be drawn after its group anchor; if that anchor consumes the width, later chips have no span. Paths and supplementary names yield space before the workspace address. Test clipped labels and wide/control characters through the existing sanitizing helpers.
- Selection/actions never parse label text. Slot refresh preserves `ThreadRowKey` across native conversation replacement. Native address remains the terminal click target; an absent native pane cannot be selected by tab click. Existing pending placeholders remain unclickable.
- Dependency clones are not discovered by this projection: its input is only existing actionable inventory and actual attached panes. No directory scan or basename-based inference may invent additional members.

### Unmatched/legacy path contract

`ThreadPresentation.Path` is presentation-only: for slots it is the validated `WorktreeRoot`; for primary members matched by scope to a slot identity it is that identity's `PrimaryRoot`; for ordinary rows with a scope-matched ancestor it is that ancestor. Otherwise it is `WorkingPath`, falling back to `StartingPath`, then the literal `(path unavailable)`. This fallback is displayed as recorded context, never asserted to be a verified checkout. Group key is always the existing nonempty ordinary repo scope; a row with no scope uses its complete native address as a private singleton key. Unknown-root groups sort by that key. Same-name groups get ` [root]` when root is known and ` [scope]` otherwise, using the full stable value, never an inferred basename. Label collisions within a group use existing native tag suffix disambiguation. Group metadata chooses known roots before unknown fallback and resolves inconsistent legacy candidates by lexical root order, independent of input order. Path rendering never changes the source row or action arguments; no label/path fallback authorizes a different operation. This is bounded best-effort compatibility for legacy records, not filesystem discovery.

### Approaches considered

1. **Shared pure UI projection (chosen):** presentation grouping belongs beside the two renderers; raw core inventory and reattachment policy retain their existing contracts.
2. Sort the core inventory only: insufficient for attached panes not yet reflected in inventory, pending placeholders, and visibility-dependent tab shorthand.
3. Sort each renderer separately: small initial patch, but duplicates repository identity and ordering rules and cannot enforce consistent navigation.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `ThreadPresentation` | `cmd/internal/couchtty/thread_presentation.go` | new |
| `presentationRow` | `cmd/internal/couchtty/thread_presentation.go` | new |
| `PresentThreads` | `cmd/internal/couchtty/thread_presentation.go` | new |
| `orderedMenuInventory` | `cmd/internal/couchtty/menu_reattach.go` | new |
| `NewMenuState` | `cmd/internal/couchtty/menu.go` | modified |
| `replaceMenuInventory` | `cmd/internal/couchtty/menu_reattach.go` | modified |
| `renderRootMenuFrame` | `cmd/internal/couchtty/menu_render.go` | modified |
| `StatusActor` | `cmd/internal/couchtty/reserve.go` | modified |
| `RenderStatusRow` | `cmd/internal/couchtty/reserve.go` | modified |

`ThreadPresentation` holds a source-row copy (valid targets preserved; malformed typed targets normalized to native routing) plus group key, full workspace label, display path, and indentation. `PresentThreads(rows)` returns one deterministically sorted copy. Group metadata and labels derive from typed identities and the existing pure repo-scope derivation; it does not mutate rows or perform IO. Map presentation entries by `menuRowKey`, including addressless recovery slots; never key them solely by zero native address. One group owns many entries. This removes competing per-surface sorting (ARCH-DRY). It is ephemeral and has no persisted cache.

`orderedMenuInventory` orders snapshots through this projection at `NewMenuState`/`replaceMenuInventory`; `menuRows` retains its existing pass overlay without sorting on each lookup. All switcher navigation, filtering, reconciliation and rendering keep using viewed lookups; no alternate direct reads of `MenuState.Inventory` are introduced.

`StatusActor` carries group context needed by the renderer to derive shorthand among the actually displayed members. Its existing `Thread` field remains the click target. The renderer determines the first visible group member and derives `repo:N` versus `:N`, retaining span creation in the same pass that clips text.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Console.finishMenuRefresh` | `cmd/internal/couchtty/console_menu.go` | modified | accepted inventory publication to tab chrome |
| `Console.statusModelLocked` | `cmd/internal/couchtty/console_presentation.go` | modified | attached panes, current menu snapshot, pending reattachments and attention ledger |

Extract substantive presentation assembly into `cmd/internal/couchtty/console_presentation.go` rather than expanding the large console file. Join current pane addresses to viewed inventory rows. Include an attached-pane fallback row from its canonical `pane.tree`, label and native address when inventory lags; inventory replaces that fallback once observed. Do not drop panes solely because a refresh has not arrived. Project the combined rows once, then retain the attached/pending membership. Do not mutate `c.order`: it also owns process bookkeeping and active-child exit fallback. No new external dependency or fake is needed; use the existing console/PTY harness and deterministic inventory event seam.

## Operating and architecture constraints

- **ARCH-PURE / ARCH-DRY:** one O(n log n) pure projection, O(n) temporary storage, no filesystem/process calls during render or while holding console lock. No new persisted state or background worker.
- **ARCH-PURPOSE:** enumerate and test switcher rows, up/down selection, filter ordering, refresh reconciliation, status tabs, pending placeholders, click spans and native activation. Previous-thread and newest-notification navigation are identity/event-based and retain those rules; they are not reinterpreted as positional navigation.
- **ARCH-MOCK:** existing console harness provides controllable input, pane attachment and inventory; test dispatched operation/address/path plus selected pane, not a function-call count alone.
- **ARCH-CONSTRAINTS:** local terminal interaction path; preserve existing 100-row menu performance budgets (`TestMenuTargetPerformance`: open 50 ms, filter/navigation/render/refresh 16 ms, feedback 100 ms) and add a pure 1,000-row benchmark for growth visibility. These are regression workloads, not hard inventory limits. Width/height bound rendering, not discovery. Disk/network IO and new concurrency are absent.
- **ARCH-SECURE:** labels, names and paths remain untrusted display text, sanitized by existing `rowtext`/menu clipping. Typed slot identities arrive through #306 validation; invalid or missing metadata falls back to an ordinary row without guessing from `-slotN` text. No credential access.
- **ARCH-ORDER:** new projection holds no state across events. Existing menu reducer owns selection. Sequences to pin: select slot → permuted refresh; select slot → new native address; remove selected row → existing fallback; pending placeholder → attached pane before inventory catches up; stale inventory result → existing generation rejection. Presentation must not reset focus, attention acknowledgment, pass state or selection.
- **ARCH-FUNERAL:** creates no durable runtime artifact because presentation values die with the render/model invocation. Test fixtures live in source control; existing thread storage retention is unchanged.

## Chunk 1: Shared grouping through both UI consumers

### Task 1: Pure grouping and label contract

**Files:** create `cmd/internal/couchtty/thread_presentation.go` and `thread_presentation_test.go`; classify the production file in `cmd/internal/artifactpath/manifest.go`.

- [x] Add failing table/permutation tests with `pair:0`, `pair:1`, `pair:2`, `pair:10`, `brain`, two distinct primary roots named `pair`, custom names, no primary, addressless slots and pathless legacy rows. Assert exact ordered row keys and labels, source input unchanged, and no manufactured dependency row.
- [x] Run `go test ./cmd/internal/couchtty -run '^TestPresentThreads' -count=1`; expect failure before implementation.
- [x] Implement `ThreadPresentation` and `PresentThreads` with the group/label rules above. Reuse `menuRowKey`, `SlotIdentity`, `WorkspaceReference`, `Worktree` and existing label sanitization/disambiguation rather than defining new identity types. Use total tie breakers and stable path qualifiers.
- [x] Repeat the targeted tests; expect PASS. Add the 1,000-row benchmark and commit `#307: derive grouped thread presentation` with the model coauthor trailer.

### Task 2: Switcher rendering and stable navigation

**Files:** modify `cmd/internal/couchtty/menu_reattach.go`, `menu_render.go`, `menu.go` only as needed; add `cmd/internal/couchtty/menu_group_test.go`; reuse the existing menu/slot/mouse regression suite.

- [x] Add failing reducer/render tests asserting full labels, two-space slot indentation, actual nested host paths, numeric grouping, custom-name searching, parked status, addressless recovery rows, same-name repos, multiple legacy primary rows with identical custom names, and full labels when filtering hides the primary.
- [x] Drive selection → shuffled refresh → changed native conversation → Enter and click. Assert unchanged slot key and exact `open-slot` path or live native switch address. Include pending pass rows and removal fallback; do not rely solely on direct helper calls.
- [x] Run `go test ./cmd/internal/couchtty -run 'Test.*(Slot|Group|Menu|PresentThreads)' -count=1`; record the relevant red failures.
- [x] Order inventory at ingestion through `PresentThreads`; derive rendered labels by stable key. Keep existing reducer selection and pass overlay machinery, adding only the grouping integration and name search preservation required by the design.
- [x] Repeat tests, confirm PASS, and commit `#307: group switcher rows by workspace`.

### Task 3: Grouped tab projection and activation

**Files:** create `cmd/internal/couchtty/console_presentation.go` and `console_presentation_test.go`; modify `console.go`, `reserve.go`; add `cmd/internal/couchtty/console_presentation_test.go` and `grouped_fixture_test.go`; reuse the existing status/reattach/mouse regression suite; classify the new production file in `cmd/internal/artifactpath/manifest.go`.

- [x] Add failing console tests attaching panes in reverse order, delivering permuted inventory, mixing groups and pending placeholders. Assert tab order equals the switcher order restricted to tab members. Cover absent/parked primary, missing inventory fallback, fallback → inventory for a thread launched in `/repo/subdir` whose `pane.tree` is `/repo`, renamed ordinary rows, and placeholder completion before refresh. Assert stable group keys/order and exact native activation for legacy duplicate primary rows.
- [x] Add renderer cases for `pair :1 :2`, `pair:1 :2` without a primary, duplicate repo names, active/bell/spinner styles, very narrow widths and wide/control text. Assert every chip span stays within width and selects precisely the intended native address; placeholders/gaps have no target.
- [x] Run `go test ./cmd/internal/couchtty -run 'Test.*(Status|Presentation|Reattach|Mouse|Group)' -count=1`; expect the new grouping assertions to fail.
- [x] Assemble status membership through `PresentThreads` using current inventory plus attached fallback rows. Preserve native lookup and attention/focus semantics, keep reattach scheduling separate, and derive shorthand from the displayed group sequence in `RenderStatusRow`.
- [x] Update old attach-order/placeholder-tail expectations to the approved grouped contract; keep scheduling and unclickable-placeholder assertions intact. Repeat targeted tests, expect PASS, and commit `#307: group status tabs with stable click routing`.

### Task 4: Acceptance fixtures, documentation and close

**Files:** update `README.md`, `atlas/couch.md`, this issue and `workshop/projects/couch-slots-v2.md`; add rendered golden fixtures under `cmd/internal/couchtty/testdata/` with names prefixed `slots_grouped_`.

- [x] Add deterministic rendered fixtures generated through production switcher/tab renderers: normal, absent-primary, parked, filtered, and narrow-width views. Fix fixture timestamps. Document `:0`, full switcher names, shorthand tabs, group ordering and parked-tab membership in operator help/README.
- [ ] Run `go test ./cmd/internal/couchtty ./cmd/internal/artifactpath -count=1`, then `go test -race ./cmd/internal/couchtty -count=1`, `go test ./... -count=1`, and `git diff --check`; expect all to pass. Run the existing menu performance tests and the new presentation benchmark, investigating any regression against the established budget.
- [x] Build via `make pair bin/couch`; verify the fixtures and a disposable console harness exercise click/Enter routing to all three nested host paths without listing their dependency clones. Do not launch or park the operator's live threads to smoke-test.
- [ ] Update atlas and project progress, record verification in the issue, commit the finished work, then use `sdlc close --issue 307 --verified '<actual evidence>'`. The binary owns the fresh boundary review; fix blocking findings and rerun affected checks. One atomic close boundary, no Mx tags.
- [ ] Follow `sdlc pr` and `sdlc merge` gates for integration, and update the project with the resulting actual/closed/PR evidence.

## Revisions

### 2026-09-23 — verify test ownership and explicit budgets

Reason: checked planned file paths and the current performance contract. Delta: corrected switcher mouse test ownership to `mouse_test.go`, included status span tests in `reserve_mouse_test.go`, and named the existing per-interaction budgets.

### 2026-09-23 — fresh review: canonical scope and legacy labels

Reason: the reviewer found that starting directories can be below the checkout root, mutable names could accidentally influence sort keys, and legacy duplicate primary rows need disambiguation. Delta: group using existing native repo scopes with slots mapped through the pure primary-root scope derivation; recover ordinary root names only by scope-matched lexical ancestors of immutable starting paths, otherwise fall back to scope. Retain native suffix disambiguation within every group and add subdirectory/fallback/rename/legacy-duplicate sequence tests.

### 2026-09-23 — advisory review approved

Fresh-context spec/plan review approved the revised design with no remaining blockers. Operator approval remains pending; no runtime changes or test-pass claims have been made.

### 2026-09-23 — gate feedback and approved execution

Reason: operator approved, and plan gate PQ-1 requested an executable legacy fallback contract. Delta: specify display path, unknown-root qualification, collision and deterministic metadata rules; no action identity changes. Rename this plan to the issue's full stem so the binary includes it directly. PQ-2 test prose compression is advisory; executable tests will carry concrete cases, with the strategy summary below owning their oracles.

Test strategy: `PresentThreads` uses permutation/property tests to assert complete stable identity ordering, source immutability, and exact known/unknown path contracts. `renderRootMenuFrame` uses rendered fixtures and reducer event sequences to prove label/context preservation and exact stable-key activation under filtering and refresh. `statusModelLocked` uses the existing stateful console harness to vary attachment/observation order and compare visible-member order with the switcher. `RenderStatusRow` sweeps widths and untrusted Unicode/control text and asserts drawn cell spans map to the exact native target.

### 2026-09-23 — implementation uses inventory-boundary ordering

Reason: the first integrated render exceeded the existing 1,400-allocation budget because each viewed lookup repeated the projection. Delta: ordering now happens at NewMenuState/replaceMenuInventory through orderedMenuInventory; menuRows only overlays pass state as before. Renderers still derive full presentation through PresentThreads. String concatenation replaces per-row variadic formatting. TestMenu100Bounds now passes without relaxing thresholds. Ordinary tabs preserve their attached labels; verified slot groups use canonical workspace labels. Active concept tables reflect the final owner/file choices.

### 2026-09-23 — acceptance trial exposed tab publication gap

Reason: a production-console test reached updated grouped inventory while the painted chips remained stale in an idle child. Delta: finishMenuRefresh now repaints chrome after an accepted successful inventory when the panel is not focused; generation rejection is unchanged. The three-workspace temporary-folder trial drives actual mouse bytes through Run, awaits operation completion and the resulting painted spans, and asserts each selected pane's exact checkout. This adds the missing integration consumer to the active table.

### 2026-09-23 — verification and commit consolidation

One coherent implementation commit contains Tasks 1–3 and acceptance fixtures because the shared projection and both consumers were verified together. The listed per-task commit subjects describe the intended checkpoints, not separate historical commits. Full `go test ./... -count=1` passed (couchcore 248.017s; full log `/tmp/pair307-full.log`); after the inventory-publication fix, the affected UI suite passed again (7.033s), build and vet passed, and the three-workspace click trial passed five repetitions under race. The final full UI race rerun is recorded in the issue.

The optional `PAIR_MENU_PERF_TARGET=m2-max` integration timing test times out while matching correlated raw output on both current code and an unmodified HEAD snapshot (`/tmp/pair307-baseline-performance.log`). This is an existing timing-harness limitation, not evidence of a regression or a passed latency gate. The unchanged allocation bounds pass; `BenchmarkPresentThreads1000` measured 1.559ms/op with concurrent verification load. No performance threshold was relaxed.

### 2026-09-23 — complete typed-target validation after boundary review

Reason: the first close review found that validating SlotIdentity alone permits an enclosing target that contradictorily carries a native address. Delta: presentationRow validates the full ThreadTarget before a snapshot enters display/selection. Explicit malformed targets, including unknown kinds and ordinary targets with slot payloads, normalize to the row's native address and native row key; invalid addressless fallbacks become unreadable/non-actionable. Valid targets and targetless legacy rows retain their contracts. Regression tests prove native Enter/click routing and no slot-path dispatch. The first close was not finalized because Ariadne concurrently updated this shared project's #246 progress; that update is preserved.
