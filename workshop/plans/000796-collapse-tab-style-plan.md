---
collaboration-mode: discussion
---

# Default collapse tab style implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for delegation. Use the executing-plans skill for session-warm work. This is one atomic close boundary; plain checkboxes are tasks, not milestone promises.

**Goal:** Default Couch to one measured tab row that retains the active and attention slots, folds quiet slots and inactive hubs into useful counts, and offers explicit multiline compatibility.

**Architecture:** Select candidates using a pure style policy, then feed the existing measured packer and shared clipped renderer. Counts carry typed group identities and spans from the drawing pass. The Console publishes those spans under its existing confirmed chrome revision and opens the existing switcher with an exact temporary scope.

**Tech stack:** Go, existing Couch CLI parser, couchtty pure renderer/reducer, terminal.Presenter, native PTY test harness.

## Spec and decisions for approval

`couch [path ...] --tab-style collapse|multiline` defaults to collapse. Accept the separate value and equals form; refuse missing/unknown/conflicting values. Like layout, this is launch-only. Opaque message bodies and arguments following `--` remain untouched. No environment preference, runtime toggle or durable setting is added.

Collapse uses exactly one row when the host has room for chrome plus one child row, otherwise the existing no-chrome fallback. Multiline keeps the existing at-most-two-row algorithm. The same selected policy supplies birth geometry, every painted frame, revision metadata and viewer chrome_rows.

Within each presentation group, show active and Bell/Blocked members. Preserve the current hub and non-inactive group's :0 as useful anchors when space allows. Quiet inactive hubs themselves fold with their quiet workers. Dirty/diverged git glyphs alone do not count as notification attention. Pending placeholders count once; their existing non-actor-clickability remains intact. Ordinary ungrouped tabs remain individual candidates and obey global scarcity priorities.

Remaining quiet members become one labelled `+N` cell per group. A fully quiet inactive group still identifies its repository before the count. Group identity is `(OwnerHub, GroupKey)`, not a display label, so equal repo names under different hubs remain separate. Use the existing presentation order and grouping authority.

Clicking a group count opens the switcher scoped exactly to that owner/group. It does not switch a child or expand the tab row. A global scarcity count opens the ordinary switcher. This is the issue's allowed switcher alternative, chosen over in-place expansion because expansion can immediately exceed one row and needs extra state/reset rules.

When required chips alone exceed physical width: retain active first, Bell/Blocked next, ordinary anchors/counts last. Reuse measured overflow and the clipped-but-clickable active fallback. Hidden attention is marked `+N !` whenever the complete marker fits; the bell still opens the full attention list. An impossible one-cell pane cannot promise every chip or marker; no phantom span is published. Hidden counts represent actor cardinality, not number of synthetic cells, and never double-count already folded actors.

This removes content-driven one/two-row chrome changes implicated in #810: attention updates, focus changes, arriving actors and notices cannot change collapse's child height. Physical host-size changes still legitimately resize PTYs. The plan does not relax any admission guard or claim all stalls are fixed.

Alternatives considered: only set maxRows=1 (reject: keeps every hub and provides unclickable global counts); independent collapse renderer (reject: duplicates clipping, broadcast controls and geometry policy); in-row count expansion (reject: extra state and the same width problem). Recommended: typed candidates through the existing renderer plus scoped switcher.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| TabStyle and parsing/default | cmd/internal/couchtty/tab_style.go; CLI uses it | new |
| StatusModel style | cmd/internal/couchtty/reserve.go | modified |
| Group identity and count candidate | cmd/internal/couchtty/status_collapse.go | new |
| Measured candidate packing | cmd/internal/couchtty/status_layout.go | modified |
| Drawn group-count span | cmd/internal/couchtty/reserve.go | new |
| Exact root menu scope | cmd/internal/couchtty/menu.go; menu_reattach.go | modified |

TabStyle has one production default (collapse); legacy low-level render tests explicitly request multiline when that is their contract. Console defaults and CLI defaults derive from the same constant. Group identity owns no inventory; it references the existing presentation's owner/group keys. Candidates own per-frame actor cardinality and attention facts; emitted cells/spans die with that frame. A menu scope is optional root-frame state, not another inventory or a second owner resolver. Future style variants extend the enum/policy; no plugin framework is needed.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Launch style plumbing | cmd/internal/couchcmd/cli.go; run.go | modified | public argv to Console configuration |
| Console geometry projection | cmd/internal/couchtty/console_chrome_layout.go; console_presentation.go | modified | selected policy to child PTY geometry |
| Chrome publication and clicks | cmd/internal/couchtty/terminal.go; terminal_input.go | modified | Presenter revision and mouse input |
| Scoped switcher opener | cmd/internal/couchtty/console.go; console_menu.go | modified | existing root-menu focus/refresh lifecycle |

Use existing FakeHost, stateful FakeRunner, PresentThreads and disposable native test harnesses. No new external service, subprocess owner or persistence backend. Native geometry probes use exact event barriers and existing fixture cleanup, never timing as ordering.

## State and recovery

Actions: launch with a style; click count; use existing switcher filter/navigation; Escape; open ordinary switcher; resize; stop.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Valid collapse/multiline launch | selected rendering | relaunch with other option | no state lost; style is launch-local |
| Invalid flag/value or non-launch use | explicit CLI error before launch | correct command | no launch occurs |
| Confirmed group count | scoped switcher with group context | select a row or Escape | no notification dismissed by opening count |
| Scope refresh changes/disappears | refreshed or empty scoped list | Escape clears scope to global root | no actors switched or removed |
| Normal switcher open | unscoped root | existing navigation | scope is intentionally discarded |
| Host geometry/revision pending | last confirmed frame; geometry actions fenced | wait for successful repaint | stale click ignored, keyboard/paste unchanged |
| More attention than available columns | active/attention-first plus +N ! if it fits | bell or count opens list | individual labels hidden, attention retained |
| Clipped/omitted count | actual clipped drawing only | ordinary switcher shortcut | omitted count has no phantom click action |

Scope survives root inventory refresh and nested action return. Escape first clears a text filter, then scope at root; a subsequent Escape follows existing global-root behavior. Ordinary switcher opening clears both scope and filter. Selection reconciliation runs within visible scoped rows. Click handling validates the same confirmed revision as actor and control spans; no stale group identity bypass.

## Work and architectural constraints

N actors, G contiguous presentation groups and B label/glyph bytes drive each render. Sanitize/measure bytes once; build grouping/count/priority facts in fixed linear passes. Reuse the existing bounded candidate-fit search: O(B + N log(N+1)) actor visits, O(N) transient storage, no per-actor full inventory scans. Count actor weights when aggregating overflow. N/2N tests instrument production preparation/packing loops, bound Bytes by a constant times B and Visits by a constant times N(1+ceil(log2(N+1))); no wall-clock benchmark is the oracle.

ARCH-DRY: one draw/clipping pass, one presentation grouping authority, one style default, one geometry projection. ARCH-PURE: candidate policy and scope filtering are pure; handlers only change existing menu state. ARCH-PURPOSE: default wiring, counts, clicks, exact scoping and geometry all ship together. ARCH-MOCK: existing stateful terminal/process doubles and native conformance harness cover IO. ARCH-SECURE: sanitize labels; preserve LIVE fail-closed, privacy, annotation anchors and stale-revision fences. ARCH-CONSTRAINTS: bounded per-frame work above; no new goroutines/timers or independent inventories. ARCH-STATE: scope transitions and pending geometry are explicit above. ARCH-FUNERAL: creates nothing durable; scope dies on ordinary reopen/clear/Console shutdown, frame data is replaced atomically. ARCH-EVOLVE: no persisted-format changes; existing additive chrome_rows remains authoritative.

## Implementation and verification tasks

### 1. Style parsing and configuration

Files: new tab_style.go/tab_style_test.go; couchcmd cli.go, run.go, layout_cli_test.go or dedicated tab_style_cli_test.go; Console configuration in console.go.

- [x] Unit-test `ParseTabStyle` against empty/default, valid and invalid values; guard: typed result or explicit error. Unit-test `extractTabStyleFlag` with repeated/conflicting flags, missing values and `--`; guard: exact preserved argv and one selected style. Unit-test `ParseCLI` with launch/non-launch and opaque-message forms; guard: only launch carries style, message bytes unchanged. Add runtime wiring test around `runTypedOperation`/`consoleRunnerFor`; guard: first Console status model uses the requested style.
- [x] Run targeted parser tests, observe failure, implement the typed default and thin launch plumbing before Console adoption/first geometry. Do not put tab style in Couch's domain workbench Layout.
- [x] Verify actual runtime launch reaches selected Console policy; explicit multiline is not merely a parsed unused value. Commit with Agent/Coauthor trailers.

### 2. Pure collapse packing and measured spans

Files: status_collapse.go/status_collapse_test.go; status_layout.go/status_layout_test.go; reserve.go/reserve_test.go.

- [x] Unit-test new `prepareCollapsedStatusChips` against mixed active/Bell/Blocked/quiet/placeholder members and owner+group alias collisions; guard: exact per-group candidate identity, actor cardinality conservation and priority. Unit-test `packStatusChips`/`renderStatusBar` against required chips exceeding width, empty groups, wide/combining/control labels; guard: one row, active span retained, hidden-attention marker, no duplicate actor accounting, existing measured-work bounds. Unit-test `renderStatusCells` against clipped/omitted group counts; guard: only actually drawn complete count labels produce exact group spans and sanitization preserves width.
- [x] Build typed group/count candidates from original presentation members; keep count actor weights and hidden-attention facts. Feed shared measured packing and renderStatusCells. Distinguish per-group count cells from global scarcity markers so final-row notice placement does not treat every group count as the final overflow.
- [x] Emit count spans in the exact drawing/clipping pass, with row-relative coordinates and no action for absent/truncated marker. Preserve actor/control spans, full or clipped active, bell and LIVE policy.
- [x] Verify conservation: visible actors + per-group folded actor weights + global hidden actor weights equals original input actor count exactly once. Preserve explicit multiline byte/span fixtures. Run N/2N measured work pins and commit.

### 3. Exact scoped switcher and publication

Files: menu.go/menu_reattach.go and tests; console.go/console_menu.go; terminal.go/terminal_input.go; console_tabbar_mouse_test.go.

- [x] Unit-test new `scopedRootThreads` (using `PresentThreads`) against equal labels across hubs and removed membership; guard: exact owner+group membership before text filtering, empty result never broadens. Unit-test `visibleMenuRows`/`reduceKey` against filter+scope combinations and nested returns; guard: selection remains visible, Escape clears filter then scope, ordinary reopen is global. Integration-test new `openGroupSwitcher` through `routeMouseEvent`/`commitChrome` with stale and stalled-writer revisions; guard: only confirmed drawn count opens scope, active child unchanged, zero action on absent span.
- [x] Apply scope through existing PresentThreads membership before root text filtering; reuse root opener/reconciliation/refresh effects instead of replicating menu lifecycle. Publish count spans alongside statusChips under the same revision and guard all geometry input.
- [x] Verify stale revision, stalled writer, clipped count and off-row click controls; retain keyboard/paste behavior. Commit.

### 4. Native geometry, safety and full flow

Files: console_chrome_layout.go; console_tabbar_test.go/console_tabbar_mouse_test.go or dedicated collapse_geometry_test.go; existing broadcast/privacy/annotation tests; README.md; atlas/couch.md.

- [x] Thread style through birth geometry and every paint path. Add native child resize observations: content changes and width boundaries never alter collapse row count or child rows; explicit multiline still changes between one and two rows as before. Cover agent kinds Claude and Codex; Qoder where existing fixture is available.
- [x] Keep exact deterministic barrier for #810 mechanism: change actor/attention content across former wrap threshold, verify no content-driven PTY resize; then physical resize updates width with one-row geometry and confirms publication before geometry clicks.
- [x] Run targeted couchtty and couchcmd suites, broadcast LIVE indicator/fail-closed, privacy rows, annotation anchors, viewer chrome_rows and interrupted-paint tests. Fix all enumerable caller screen assumptions, preserving full-diagnostic assertions (#789 lesson).
- [x] Update help/README/atlas default/option and count interaction. Register any new non-test source in artifactpath manifest; audit repository-reading tests/certificates and run inventory plus diff check.
- [ ] Commit/push; submit recorded affected or suite check according to repo declaration. Stage only when TL sequences an operator window; report concrete large-fleet default-collapse, count navigation, multiline comparison steps. Await operator acceptance before landing.
- [ ] Close through SDLC (mandatory fresh review), resolve findings by class, and merge. Return to #789 immediately when its existing suite result arrives; preserve both branches and never stack them.

## Approval and outstanding dependencies

No code before TL approves the policy decisions and this plan. #789's full-copy/display cap is on a separate unlanded branch when this plan is authored; do not copy it here or build on it. After it lands, normal main integration brings it in and validates both styles together. Keep #812's removal of trailing status text out of this scope; notices remain where the shared renderer currently puts them until that issue changes it.

## Revisions

### 2026-10-22 — approved by TL3fa9a5f6

All policy choices above approved. Run change-code and implement; stage for operator acceptance before landing.

### 2026-10-22 — PQ-1 function-level test strategies

Named the parser/extractor/CLI, collapse preparation/packer/drawing, exact scope filter/reducer and confirmed mouse opener functions. Each has an adversarial input class and mechanical oracle; conservation, counted work and deterministic publication barriers remain. This is a test-contract refinement, with no approved behavior delta.
### 2026-10-22 — BR-1 numbered-hub coverage

Legacy multiline display runs can split one identity after a numbered hub. Collapse aggregates by exact owner+group independently of those display runs and preserves the current hub by role, regardless of slot number. Tests enumerate primary0/numbered1/numbered4 x current/inactive; multiline grouping stays unchanged. The existing O(N) preparation and weighted-count invariants apply.
