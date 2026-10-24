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

- In the open switcher, a press on a different eligible row selects without switching. A press on the selected visible root row activates; rapid100ms double presses and delayed5s second presses are covered through real mouse input, with exactly one activation and no timer.
- Open-switcher chips follow the same rule. A hidden chip first reveals/selects, clearing only the filter/group/view that hides it and preserving visible restrictions. A chip clicked from a nested frame first returns to the root and selects without activation.
- Chips outside the switcher still activate immediately; +N still opens its group. Canonical row identity, current operation vetoes and non-left/release/motion/wheel negatives remain covered.
- Enter and the other switcher keys behave as before (existing tests pass). README and atlas describe selection versus activation and retrying a selected row.

## Plan

- [x] Add RED pure event and real SGR mouse-path regressions for different/selected rows, rapid double press and delayed selected press; canonical recordless-slot identity, reorder/replacement, hidden-chip reveal, group counts, nested frames and current operation vetoes.
- [x] Implement the shared pure select/activate transition and scoped Console routing/classification. Preserve direct actor-view chip switching and keyboard behavior. Pin release/motion/right/modifier negatives, manual-versus-paging origin, stale geometry and8/16 dispatch counts.
- [x] Run focused and race checks plus complete couchtty; update the existing atlas/user interaction text and individually audit any named affected-reader changes. Commit/push; run sdlc local-check run --pre-close and inventory before binary-owned close. Request a hub-authorized :0 stage/operator click check after the tested build is reviewable; then normal SDLC landing and stage cleanup.

## Log


- 2026-10-24: closed — BR-3 fixed by evaluating Filter within retained GroupScope after membership proof; mixed Focus/scoped-fuzzy/outside-exact regression RED before fix, GREEN after; converse legitimate scope/filter clearing pinned. Ten focused repetitions PASS18.612s, focused race3.882s, full couchtty61.516s, fresh inventory PASS. New pushed-head pre-close768030bc is retained (suite fallback); pass required before landing. Prior recorded suite13m54s and operator combined3d0135b0 click PASS remain documented. BR-1/2 addressed. Measurement refused: measurement branch proof: measurement sources exceed snapshot limit 128; attempt window2026-10-23T20:43:25-07:00 through2026-10-24T10:40:39-07:00; hub818613af/20d35b9a authorizes precise --no-actual, no typed hours.; review verdict: SHIP
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

### 2026-10-24 — Retained check failed; audited source classification and shared fixture

Original terminal87226 finished exit1 (recorded affected exit2) on implementation d689f57acf6035397911a4896d81cb4e5efbd5b4: 55s execution after4784s FIFO wait, log SHA256 prefix33609fb3a72a, retained failed log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/1f978bc15086ed3e71a7d91e440b549a.log. Source preflight failed artifactpath.TestProductionArtifactReferencesAreExactlyClassified because new menu_mouse.go was absent from the exhaustive production inventory; every other reported package passed. This is a branch omission, not a flake. Audited the complete pure file: only current MenuState/event/row transformations and no artifact paths or I/O; added it to NonArtifactSources. Exact guard now passes3.282s.

Before another queued run, the targeted python3 -B tests/affected-config-test.py also reproduced838's setup failure: missing github.com/xianxu/ariadne/cmd/sdlc/internal/processgroup. This was not reached by the original source preflight. Hub authorization1a70aa46-b302-4a01-b07d-e84e6e962a34 permits reuse of pair:3's isolated b3da4973ef63b6a1af8ab941721abde8f3038faf only after failure; now the retained runner has failed and the exact fixture is independently RED. Next cherry-pick only that fixture commit, verify it, audit affected inventory, push and start one new recorded pre-close. Preserve the existing851 operator stage;838 remains not stage-ready until its repaired check passes. Hub received original failure report19d873c6-2963-4368-9a34-17c3de52c87f as Submitted.

### 2026-10-24 — Fixture repaired with measured RED/GREEN

Reused only isolated838 commit b3da4973ef63b6a1af8ab941721abde8f3038faf as cherry-pick0dde1e26c43e6345c5d52a50eef35d580a4aa96e, after the retained run ended and the exact fixture independently failed. The unchanged upstream source set now includes its processgroup/procgroup dependencies and same-snapshot module metadata. Targeted fixture GREEN47.787s, including all778 group/file inputs; staged-reader inventory passes with no repairs. PR577 updated to that pushed head. Full artifactpath package verification is running; once read, start one new pre-close run. No UI code or existing stage revision changed.838's separate pin-related repolint and removed-editorial-source failures remain with pair:3; do not copy those unrelated changes.

### 2026-10-24 — One repaired-head pre-close retained

Full artifactpath package passed15.417s. Pushed402b6b1d8d111dac5c27abcad67b8a0e56faafe9; new original terminal6353 launched via --pre-close. The selector correctly chooses a full suite fallback because tests/affected-config-test.py is changed: lease-3842004767, last queued5. Do not duplicate/cancel it. Log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/242e82dfd9ebece0eb479ac56dc8ac86/run.log; scratch /Volumes/TestScratch/local-checks/sdlc-local-2384763872. Hub repair reportcd6f62be-73e8-42da-83c7-c019e8902d47 is Submitted. Next inspect terminal6353 result, handle actual failures with evidence, then continue pending838 combined stage/operator acceptance and close.851 UI code and existing staged binary remain unchanged.

### 2026-10-24 — Full suite PASS and refreshed stage

Retained terminal6353 completed exit0: required full-suite fallback passed13m54s on402b6b1d8d111dac5c27abcad67b8a0e56faafe9, log SHA256 prefix30e6ba7b0c61. Saved log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/402b6b1d8d111dac5c27abcad67b8a0e56faafe9/suite.log. No failures or flake reruns. Hub report67167838-a40e-44ee-afc3-427ca62138d2 and peer readiness reportb0649d81-475c-4a06-b15b-a52dc6940557 are Submitted. No own check remains active.

Refreshed own851 stage via sdlc stage --build: pair:0 HEADde20498a199cd6681fa9c69deddff6508be7becb, exact tested code plus prior Log; all five binaries verified that vcs.revision and vcs.modified=false. PR577 all three CI workflows green. Stage remains851's lease.838 is waiting on main-based861 adoption, then its own repaired pre-close; peer has promised readiness before any stage change. Await hub-authorized combined build/operator check, then finish Plan, audit inventory immediately before close, binary-owned close, merge and clear stage. Do not invent acceptance or start an overlapping item. Hub8076087e-ed4a-48ff-a886-30e3f5980928 permits --no-actual only if close actually refuses ambiguous concurrent-claim measurement; include exact refusal/window, never typed hours.

### 2026-10-24 — Measurement exception revoked

Authenticated Submitted hub delivery86d5fc90-bb04-4766-b4fd-6e67748b6f24: ariadne591 landed and tools rebuilt. Earlier concurrent-claims/resting-branch --no-actual authorization is revoked; measure normally at close. Only an actual separate64KiB snapshot refusal (ariadne593) retains --no-actual with exact evidence.851 still awaits838 combined stage and operator acceptance; no further check or close runner is active.

### 2026-10-24 — Supported combined-stage handoff

Hub morning plan b4122ce2-bf6b-4ced-a667-2c9b38c26959 assigns pair:3 one combined stage containing main856 plus851/838/842, with857 only if independently ready. Handed851 exact clean pushed d74a152d8b18978dbcda2c9961067678f5854069, recorded suite evidence and operator gestures; receipts72dce12c-d5bd-4541-9edc-83650252428c and b134b6cd-58c2-41d7-ab03-10744ea5cd7f are Submitted. Pair:3 confirms assembled/tested staging-only candidate3d0135b0021209730956929c8926e30f9799ba79 includes the851 pin and no857; coordinator owns combined validation/provenance reporting.

At authenticated ready request59af07a4-0538-4fd6-ac5b-abe10cda0833, ran sdlc stage --clear from851. Completed exit0: pair:0 restored/fast-forwarded to main7cb0b672ebb6aca081ae63f19c27b0aafc45270f, all five binaries verified that vcs.revision and vcs.modified=false;851 lease released. Sent exact verified completion1bbd6f4f-a797-4575-8753-950f0cf2f7e7 to pair:3, who will acquire its own lease with sdlc stage --build. Do not clear or rebuild pair:0 again from851: coordinator now owns the next stage. This Log-only checkpoint does not alter handed-off851 code. Next wait for combined build and operator acceptance, then inventory and close normally; no own runner remains active.

### 2026-10-24 — Combined build verified, awaiting operator

Pair:3 confirmed combined stage BUILT (Submitted470c6612-bee3-4b6d-bdac-1a75e91a54a5). Independently verified pair:0 Git clean at3d0135b0021209730956929c8926e30f9799ba79, and all five Go1.27.3 binaries report that exact vcs.revision with vcs.modified=false. Read checklist /Users/xianxu/workspace/pair/workshop/evidence/000838/morning-combined-check.md: main856 plus851/838/842;857 excluded;851 gestures match the approved contract.838 owns the supported stage lease and will coordinate its clear before individual landing. Hub notified via6baf5f53-86b0-45a8-a337-f351b6643266. Await actual operator result; build provenance and coordinator targeted passes are not operator acceptance. No own runner or pending code change.

### 2026-10-24 — Operator acceptance PASS, closing

Authenticated Submitted ops:0 delivery51719893-1354-48dd-b33a-a4c70d7d94b4 reports operator switcher-click PASS ("works") on verified combined3d0135b0021209730956929c8926e30f9799ba79 and directs close/landing. No851 code changed since recorded full-suite PASS13m54s on402b6b1d8d111dac5c27abcad67b8a0e56faafe9 (30e6ba7b0c61); branch descendants are issue Log only. Fresh make test-affected-inventory passes with no repairs. Atlas, targeted/race/real SGR tests, exact-count and keyboard proofs plus native operator acceptance satisfy the plan; ticked final verification row. Normal measurement and binary-owned boundary review next, then SDLC merge. Pair:3 owns combined stage clear; coordination3fb502d1-828f-4573-a1f2-8ee75eed7e48 is Submitted. Do not touch another issue's lease.

### 2026-10-24 — Exact measurement refusal authorized; preserve shared stage

Normal close attempt terminal3575 ended before review with: "can't auto-compute (measurement branch proof: measurement sources exceed snapshot limit 128)". Attempted measurement window2026-10-23T20:43:25-07:00 through2026-10-24T10:40:39-07:00. No actuals were fabricated or review dispatched. Submitted hub818613af-945d-4b2c-ba4b-ced126499fa8 explicitly authorizes precise --no-actual with this source-count refusal and window in --verified; the long multi-session window is intentionally refused by the source cap. Re-run inventory, then close with that exact acknowledgment.

Hub correction a1f718e7-b5b6-4074-aa67-d6d74dd875b8 supersedes stage-clear coordination for851 landing: land independently; preserve838's combined stage while editorial operator check continues. Pair:3 received a16e8e21-871b-430a-9c56-5ecc296c8beb as Submitted.851 does not clear or wait for that stage. No code changes since passing suite/operator candidate.

### 2026-10-24 — BR-1/BR-2 documentation and acceptance synchronization

First close review found no executable defect and independently passed couchtty51.647s, focused race3.508s, artifact guard2.290s and pinned fixture12.973s, but FIX-THEN-SHIP could not finalize with Important ledger findingsBR-1/BR-2 open. README still described immediate switcher-row clicks and unqualified one-click warm retry; original Done-when still named a double-click timing window and omitted scoped chips. Corrected the complete affected README passages and the equivalent atlas warm-retry claim. Reconciled Done-when to the already-approved no-timer design, existing rapid/delayed press proof, inside/outside and hidden/nested chip outcomes, selective restriction preservation, existing vetoes and keyboard behavior. This changes no implementation, authority or scope; it records the acceptance already approved and observed. The previous timing-window wording is superseded; no timing recognizer was added. Re-run full close as its ledger refusal explicitly directs, so both findings receive dispositions.

Stage status: pair:3 subsequently reported authenticated supported clear c9fdefd3-b08a-44bb-b714-7ba41ccf3442 after838 operator PASS; its lease is released and pair:0 rebuilt on main.851 took no stage action. Fresh inventory and diff whitespace checks pass for these prose-only review corrections; executable code remains exactly the passing suite/operator code.

### 2026-10-24 — BR-3 preserve effective reveal context

Review2 disposed BR-1/BR-2, but raised CriticalBR-3 preserve-effective-reveal-context: Filter exact-over-fuzzy precedence depends on the active GroupScope. The old reveal predicate tested filter globally, so an outside exact match could suppress the target's scoped fuzzy match and erase a valid filter when only Focus view hid the target. Existing operator acceptance covered normal clicks; this mixed-restriction branch defect blocks close until corrected.

Retained TestSwitcherRevealPreservesEffectiveFilterContext with three compositions: preserve scoped fuzzy match despite outside exact match when only Focus hides; clear filter when a same-group exact match legitimately suppresses target; remove wrong scope and recompute filter over its new candidates. RED showed filter="" in the preservation case on pre-fix code. Corrected the one context-losing predicate to retain root.GroupScope after any necessary scope removal. Scope membership still checks independently; view still checks the retained complete frame. No new state, input policy, deadline, I/O or effect path; this enforces the approved selective-preservation contract (ARCH-PURPOSE). GREEN: focused reducer tests10 repetitions passed18.612s; focused race3.882s; full couchtty61.516s; inventory and diff whitespace checks pass. Next commit/push and one fresh recorded pre-close for the executable correction, followed by full close to dispose BR-3. Hub notified via Submitted ef871782-26dc-46bb-9581-53068dcd11a3.
