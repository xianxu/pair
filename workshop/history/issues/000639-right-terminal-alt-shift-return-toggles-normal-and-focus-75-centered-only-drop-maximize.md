---
id: 000639
status: done
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '1a6cb427cfe4553f95ede11fabaedb300c2fc810' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T00:09:57-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "ae5d0bc4", done: "7fb2d7a1"}
actual_hours: 1.08
---

# Right terminal: Alt+Shift+Return toggles Normal and Focus (75% centered) only; drop Maximize

## Problem

## Spec

Operator, 2026-10-16: "change right pane maximize, from 3 states to just two: left/right even split, and the right pane in center 3/4. remove the actual maximized view. I find the 3/4 width of screen is enough and still give some peek of background activity in the agent pane."

Today Alt+Shift+Return cycles the selected right terminal Normal → Focus → Maximize → Normal (`cmd/internal/layoutcmd/rightpane.go`; README's key table):
- Focus floats it, centered at 75% width and full height, with the agent pane dimmed.
- Maximize makes it native zellij fullscreen.

New behaviour: a two-state toggle, Normal ↔ Focus. Maximize goes away as a target of the cycle.
- Alt+Shift+Return from Focus returns to the tiling, focusing the invoking pane, as Maximize → Normal does today.
- A pane already fullscreen when the cycle runs, left over from a previous build or set natively, still returns to Normal. Don't strand it.
- Update the README key table, the atlas and the key help.
- Keep the split rule: a split acts only on the selected half.
- TL authorized landing on recorded suite + close review without staging in pair:0. Operator real-setup smoke belongs to the next Couch restart; it remains pending after landing and any defect is fixed forward.

## Done when

- The pure plan tests show Normal → Focus → Normal, and that no plan targets Maximize.
- An existing fullscreen right terminal returns to Normal on one press; a test covers it.
- The live chord conformance test passes on the two-state cycle.
- README, the atlas and the key help describe two states.
- Next-restart operator smoke steps are recorded; execution is explicitly deferred by TL and does not block landing.

## Plan

- [x] Make planner/runtime tests require Focus → Normal and retain native-fullscreen exit coverage; observe red.
- [x] Reuse existing restore/focus/clear operations for Focus exit; update source-derived help, README, atlas and conformance scenarios.
- [x] Run affected packages and native chord conformance, coordinate real-setup smoke with hub, and pass the recorded full suite. Mandatory close review and landing follow through their SDLC gates.

## Log

### 2026-10-16

### 2026-10-17 — quick design
- 2026-10-17: closed — BR-1 corrected: removed only the stale third press from rung conformance. Native red reproduced base-restored failure in 12.29s; corrected all six base/minimized/third/split cases pass 5.14s with every geometry, half-order, record-clear and diagnostic assertion preserved; complete layoutcmd package passes. Enumerated transition consumers by function/action identity. Production unchanged since recorded full suite 0bad7dee0ea7ab202adb00e055dbc6585cd4a109, exit0 13m25s SHA256 02007bddd16aefcb1a54a1d69b8d16fce1f9c5717ac2c9247fc8d0940e336655; native chord/direct conformance and race checks passed. Final-head suite remains enforced at landing. TL owns project; operator smoke explicitly deferred to next Couch restart, steps recorded, pair:0 untouched. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

TL dispatched the operator’s explicit two-state contract and authorized quick flow. ARCH-PURE/ARCH-DRY: retain PlanRightPane as the one transition owner and reuse its existing operations. Focus exit re-embeds, restores the recorded tiling and split order, returns to the invoking pane, clears the return record and nudges dimming. Native fullscreen remains observable only for the one-press compatibility exit; no plan enters it. Existing lock, stop-on-error and degraded-restore behavior remain. No new durable artifacts, external calls or background work. Existing runtime fake and native chord/conformance tests cover single and split panes, invoking-focus restoration, failure ordering and native fullscreen. The fixed 75% centered geometry is unchanged. Plan posted to hub, proceeding unless objected.

### 2026-10-17 — close round 1, BR-1 test-consumer sweep

Close review returned FIX-THEN-SHIP with Important BR-1 (transition-contract-test-consumers), so the ledger blocked finalization. Verified locally: unmodified TestRightPaneRungsZellijConformance fails base-restored after 12.29s because its third press re-enters Focus. Removed only that extra press; all six shared-helper rung cases and existing exact geometry, half-order, cleared-record and diagnostic assertions remain. Enumeration by RunToggleFocused/action identity covers pure planner, stateful runtime/adversarial, direct native conformance, actual chord, rung conformance, terminal/wrapper routing, editor route and source-derived key-help tests. Other cycle sequences already use two presses; native fullscreen exit is a separate one-press compatibility check. Added the missed-family lesson. No production changes or contract revision. Rung conformance now passes all six cases in 5.14s; complete layoutcmd package passes 0.161s. Red/green logs retained at /tmp/pair639-7xyqf5rg/rung-before-red.log and rung-after-green.log. The recorded 13m25s full suite remains evidence for unchanged production; landing must record the final reviewed/integrated head through the normal local-suite gate.

## Revisions

### 2026-10-17 — native return-focus conformance

Native conformance shows re-embed can already focus the invoking pane (focus-pane-id then exits 2), or move away when the terminal itself is the recorded return. TL approved always issuing StepFocus for a known return target, and accepting only exit 2 with exactly the target-specific `Pane Terminal(N) is already focused` diagnostic. Preserve native action output in typed errors; every other failure still stops. Pin the external wording and code in real-Zellij conformance, and model idempotence plus focus movement in the stateful fake. This corrects the Focus→Normal transition without new state or artifacts.

### 2026-10-17 — real-setup acceptance scheduling

TL explicitly prohibited staging/building in pair:0, which supplies the running Couch and its next restart. Land after recorded suite and close review; operator owns the pending real-setup check at the next Couch restart, with fixes forward if needed. This supersedes the Plan's pre-landing smoke coordination: coordination is complete and execution is deferred.

Operator smoke after restart: (1) In a split right terminal, press Alt+Shift+Return in each half separately: only the selected half floats, centered at 75% width, full height, with background agent activity still visible. (2) From Focus press once: both halves return to their original tiled order and the invoking pane regains focus; repeat twice to prove the cycle has only Normal and Focus. Include an invoking draft/agent pane and a full-screen child app in the terminal. (3) Leave a right terminal natively fullscreen, then press once: it returns directly to Normal, without visiting Focus. Check return focus and absence of new failure diagnostics.

### 2026-10-17 — implementation and targeted verification

Initial pure-plan red showed Focus still targeted Maximize. Added same-pane return red, and stateful fake controls reproduced native re-embed focus movement and exact already-focused exit2. Implemented direct Normal↔Focus, preserving the native-fullscreen exit-only observation. Shared the existing recorded-return lookup between both exits. Both native action producers (layout CLI/agent and terminal) now retain identical typed focus evidence. No new exported API or artifact family.

Affected layoutcmd/termcmd/keyhelp/workbenchshortcut packages pass normally and under race (latest 1.674s/3.024s/1.148s/1.245s). Old-adapter overlays make the new terminal record-clear regression and layout evidence regression fail; controls are retained in /tmp/pair639-7xyqf5rg. Direct native conformance passes 2.66s, pinning exit2 and the exact target-specific diagnostic, all four caller/split cases, native one-press exit and native focus-away. Corrected actual-chord live test passes all five paths (draft mapping, generic wrapper, terminal bottom/top, nvim child) in 4.41s, with exact geometry/focus/record-clear assertions after the second press.

Qualification failures retained: first candidate chord lost session before input; original-layoutcmd binary/body control reproduced session loss later (4.22s), filed unassigned #640, root cause unknown. Original-test-only/candidate-binary control is excluded. Direct conformance with inherited long socket root was refused before render; short private socket qualification passes. Candidate's already-focused and same-pane return failures were corrected here. Direct shell shortcut check initially used old local bin/pair; after make build in this worker slot it passes. No primary-slot build or stage occurred.

### 2026-10-17 — recorded full suite and relaunch checkpoint

Recorded suite passed exit0 in 13m25s on 0bad7dee0ea7ab202adb00e055dbc6585cd4a109 (make PAIR_GO_TRIMPATH=-trimpath test). Log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/0bad7dee0ea7ab202adb00e055dbc6585cd4a109/suite.log; SHA256 02007bddd16aefcb1a54a1d69b8d16fce1f9c5717ac2c9247fc8d0940e336655. All GitHub checks are green, including native conformance. The recorded suite completed with couchcore446.903s, layoutcmd0.300s, termcmd0.997s, keyhelp0.160s, workbenchshortcut1.130s and wrapcmd53.334s. No test failures in this suite. PR #391 is open.

No background work remains. Reporting idle for relaunch at the requested checkpoint, before starting another long wait. Mandatory close review and landing are still pending; no acceptance verdict has been claimed. Last relaunch was refused pending #618 M2, after which TL authorized starting #639. Real-setup smoke remains scheduled for the next Couch restart per the approved revision.

### 2026-10-17 — resume close and landing

Relaunch was again refused due to provisional Codex binding. TL explicitly directed continuing close review and landing, and no more relaunch stops until the hub announces #618 M2 landed. Verification checkbox now records completed checks; close/landing remain their mandatory gates rather than an unchecked implementation task. No requirements or implementation changed.
