---
id: 000851
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
card_mirror: 'cbff748d967f8c6470543136378743f8e62c92d0' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T20:43:25-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "3dc35b44", done: "4d90c246"}
---

# Switcher: single click selects, a click on the selected row or a double click switches

## Problem

## Spec

**Switcher clicks: select first, then switch (operator, 10-23).** In the Couch switcher, a single mouse click today
jumps straight to the clicked slot, and the operator sometimes isn't sure where they clicked. New behaviour:
- **Single click** selects the row (moves the highlight) without switching.
- **A click on the already-selected row** switches to it.
- **Double click** on any row switches to it directly.
Keyboard behaviour is unchanged (Enter still switches to the selected row). Other click targets in the switcher
(chips, +N groups, action buttons) keep their current behaviour unless they also jump on a single click, in which
case they follow the same rule.

### Proposed correction — 2026-10-23

Quick flow, estimated under100 added production lines. The switcher uses its existing durable row selection as the click acknowledgment: an eligible press on a different row selects/repaints it with zero operation effects; a press on the already-selected visible root row activates through the existing manual MouseSwitch dispatch. Compare canonical ThreadRowKey, not index or possibly empty Address, so two recordless slots stay distinct and inventory reordering cannot redirect selection. A double click is two presses: the first selects and the second activates, exactly once. A later click on the selected row must also activate. No double-click timer, expiry, click-history field or new goroutine is necessary (ARCH-ORDER, ARCH-PURPOSE).

Add one semantic MenuEventMouseRowClick handled by a pure reduceSwitcherRowClick transition, and share the existing manual activation arm instead of duplicating its operation/reattach/ownership policy. The root-list hit path emits this event only from confirmed geometry. Actor chips also use select-first while the switcher/panel is open; outside the switcher their existing immediate switch remains. For a chip target hidden by the root's filter or GroupScope, the event explicitly reveals that target by clearing only the hiding filter/scope before selecting it; a first press from a nested frame returns to the visible root and selects, without activation. Preserve an already-visible filter/scope. A list-row event cannot reveal a hidden/stale target. Include the new semantic input in navigation-generation, landing-attempt and copy/notice input classification, so selection intent fences later automatic focus exactly as other navigation does (ARCH-PURE, ARCH-DRY).

Scope census: terminal_input.go routes root row extents and actor chips; +N group counts call openGroupSwitcher and do not switch, so retain that action. Attention/notes/cast controls precede this route and retain their actions. Action/confirmation/recovery controls keep their current explicit actions; this policy changes row/chip activation, not buttons. Pending, blocked, unknown, missing, busy and in-flight targets retain current admission/operation vetoes; selection never creates dispatch authority or bypasses them. Enter, arrows, Tab, filters, Escape, manual mouse pinning and keyboard paging/notification-hop behavior keep their existing semantics. Mouse release, motion, wheel, modified and right presses do not become row activations. No parser, protocol, persisted format, lifecycle or agent-composer change (ARCH-SECURE, ARCH-EVOLVE).

Alternatives: a time-based double-click recognizer duplicates the selected-row rule and would incorrectly expire a slow second click; Console-local click history would duplicate menu state and become stale across refresh/filter/async events. Use the existing pure selection state and current typed target instead. SGR mouse reports carry presses/releases rather than a click count; no host double-click setting is required. Pin rapid100ms press pairs and delayed5s selected-row presses with the existing Console testClock/event At seam, without sleeps or a new timing window.

Actions: click a row/chip, Enter, navigate/retype filter, Escape.

| Existing state / input | Visible result | Recovery action | Possible loss |
|---|---|---|---|
| Different eligible row pressed | New highlight; switcher stays open; no operation | Enter or click highlighted row | None |
| Already-highlighted visible root row pressed | Existing manual activation/preview and its progress | Existing result/refusal recovery | None beyond existing operation contract |
| Rapid double press on one row | First highlight, one activation on second press | Existing result/refusal recovery | None |
| Chip hidden by filter/scope or clicked from nested frame | Root opens with that target visible and selected; no activation | Click/Enter, or retype filter | Only the filter/scope hiding that target is cleared |
| Row removed, stale geometry, blocked/pending or operation veto | Existing no-op/refusal/progress; no new activation | Refresh/navigate or Escape | No target work or drafts lost |
| Inventory reorder/slot conversation replacement | Selection follows stable row identity; current target is revalidated | Click/Enter current row | No cached click authority |

| Concept | Kind | File | Change |
|---|---|---|---|
| MenuEventMouseRowClick / reduceSwitcherRowClick | PURE | cmd/internal/couchtty/menu.go (or focused menu_mouse.go) | new event/transition using existing selection |
| Existing manual mouse activation and target/operation checks | PURE | menu.go, menu_slot.go | reuse/shared arm; no weakened admission |
| Confirmed row/chip routing | INTEGRATION | terminal_input.go | emit select-first event only while switcher owns screen |
| Navigation and feedback input classification | INTEGRATION/PURE | console_menu.go, menu_copy.go | include new event |
| Real mouse-loop fixture and clock | INTEGRATION | console_mouse_test.go, console_activity_test.go | reuse for SGR press/release proof |

Envelope: one decision per accepted press, no IO before the existing declared operation effect; at most one operation on an activating press, zero on selection. Existing hit testing, target resolution, state cloning and painting remain linear in displayed/inventory rows; add only constant row-identity/disposition work and no extra polling. Count dispatched effects over8/16 two-press cycles, requiring exactlyN activations and zero first-press effects, to reject multiplied IO. Selection uses existing state, dies with the menu, and produces no durable record/cache/worker (ARCH-CONSTRAINTS, ARCH-FUNERAL). Existing fake Console/host/dispatcher owns all tests; a stage and real operator click check is requested through the hub only after implementation is concrete (ARCH-MOCK). This is a requested UI policy, not a mechanical API/flake mistake class requiring a new lint rule.

## Done when

- Single click selects without switching; a second click on the selected row switches; a double click switches
  (tests driving the real mouse event path, including the double-click timing window).
- Enter and the other switcher keys behave as before (existing tests pass).

## Plan

- [x] Add RED pure event and real SGR mouse-path regressions for different/selected rows, rapid double press and delayed selected press; canonical recordless-slot identity, reorder/replacement, hidden-chip reveal, group counts, nested frames and current operation vetoes.
- [x] Implement the shared pure select/activate transition and scoped Console routing/classification. Preserve direct actor-view chip switching and keyboard behavior. Pin release/motion/right/modifier negatives, manual-versus-paging origin, stale geometry and8/16 dispatch counts.
- [ ] Run focused and race checks plus complete couchtty; update the existing atlas/user interaction text and individually audit any named affected-reader changes. Commit/push; run sdlc local-check run --pre-close and inventory before binary-owned close. Request a hub-authorized :0 stage/operator click check after the tested build is reviewable; then normal SDLC landing and stage cleanup.

## Log

### 2026-10-23


## Revisions

### 2026-10-23 — Selection, not elapsed time, defines activation

The proposed design makes both operator gestures derive from existing selected-row state, without adding a double-click timer. It explicitly scopes actor-chip changes to the open switcher, reveals a hidden clicked chip in the root, and retains all keyboard/operation admission behavior. This is a proposal to ops:0; wait for approval before change-code.

### 2026-10-23 — Claimed and traced current routing

679's retained merger exited0: PR572 and train PR574 verified MERGED, train334 integrated suite passed12m57s with no split suites. Claimed851, then start-plan created its own branch from main715b1e199cf2368c46c7e43cb9a4d8c67c4624e3. Catch-up rebuilt local tools against ariadne8d3b37c4518bc49f2f8c4bf5adc728dc9de3f93a; seeded workflow changes were deferred automatically, with no local-gates edit.

Current terminal_input.go sends root row hits straight to MenuEventMouseSwitch, while actor chips call switchToThread regardless of panel focus. ReduceMenu's MouseSwitch arm resolves a current typed target, applies existing busy/selectability/operation checks, selects it, then dispatches with Manual origin. +N groups only open a scoped root. newMouseFixture already drives real SGR bytes through Console.Run and the confirmed hit map, with FakeHost, fake children, owned teardown and a recording dispatcher; Console.now and testClock already exist. No production edit, native UI action, stage, recorded check or merge runner has started for851. Next: send this committed plan to ops:0 and await approval; use the new --pre-close selector for the eventual recorded check.

### 2026-10-23 — Approved implementation and local proof

ops:0 approved the selected-row acknowledgment and all scoped chip behavior (delivery acf73f3c-f5d0-462c-ab33-302794103a3b, authenticated Submitted). change-code completed quick flow. The real SGR regression was RED: a different-row first press dispatched switch; a hidden chip failed to reveal its target. Added MenuEventMouseRowClick and a pure selection transition; the former MouseSwitch arm is shared unchanged for activation. Root list hits and panel chip hits use selection; external chips remain immediate. Typed keys, visible root membership and current operation vetoes remain mandatory. Navigation generation, landing attempt and copy feedback recognize the new input.

GREEN: focused real-path tests, reducer tests (8/16 effect counts, recordless identities, reorder/replacement, hidden and nested chips, pending and operation vetoes), non-left/release/motion/wheel negatives; focused race check passed4.336s. Full couchtty passed46.402s. Go1.27.3 make build passed. Atlas updated. After staging the new tests, make test-affected-inventory passed without reader repairs. The focused selection, raw mouse, +N and stale-geometry cases also passed10 repetitions in14.033s. Next commit/push and run the recorded --pre-close selector on the pushed head.

Stage dry-run refuses because pair:0 is held by #849, stage/000849-000850 from pair:6. Preserve it. The approved operator check waits for those cast-notes items to free the stage, then sdlc stage --build and report exact binary revisions to ops:0. No stage effect, PR, recorded check or merge runner started yet.

### 2026-10-23 — PR577, retained check and operator stage

Implementation head d689f57acf6035397911a4896d81cb4e5efbd5b4 is pushed as PR577. sdlc local-check run --pre-close selected affected as intended (trains enabled); original terminal87226/run lease-4179351376 remains queued, last position10 behind the two-slot Ariadne run. Run log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/1f978bc15086ed3e71a7d91e440b549a/run.log. Preserve this original runner/FIFO; inspect its result before any retry. No close or merge runner yet.

pair:6 cleared the cast-notes stage. sdlc stage --build completed exit0: pair:0 HEAD is exactly d689f57acf6035397911a4896d81cb4e5efbd5b4, and bin/couch, couch-conformance-bridge, pair, pair-launch-helper and pair-slug each verified that vcs.revision with vcs.modified=false. Hub stage report0fbfaafc-e553-44ce-b1a8-da5f6e6c6fe0 is Submitted. Current live processes still use their existing binaries until operator restart.

ops:0 delivery ce4948a3-f51c-40ae-9d68-2182b7bb3f38 authorizes a combined stage with pair:3's838 when its retained suite passes, for one operator restart. Coordination reportf9f2477a-c711-447c-bf8b-e7e68c940f5e is Submitted; pair:3 will send the exact passing-check head/readiness. Preserve851's stage lease meanwhile. Next: read retained check, construct the authorized combined validation stage after838 readiness without stacking issue development, verify build provenance, and have ops:0 obtain operator click acceptance. Then inventory audit, tick final Plan row and binary-owned close, followed by merge and stage cleanup. The current work is waiting on machine capacity and that external acceptance; no native acceptance or completion is claimed.
