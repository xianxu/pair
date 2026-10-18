---
id: 000576
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-17
estimate_hours: 2.34
card_mirror: 'e42837b93fee435cd9e54edd5696ad9b133d2ac5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T19:21:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Couch switcher: lifecycle actions run in the background per slot; the UI never blocks

## Problem

While the operator runs a lifecycle action in Couch's switcher (bring a slot up, relaunch, resume), the switcher holds
the screen until the action finishes or fails: the operator can't switch to another slot or the hub meanwhile
(operator, 2026-10-15). A slot's lifecycle is already modelled as states, and startup brings many slots up in the
background, so a single slot's action needn't block the UI.

## Spec

- **Lifecycle actions run as background jobs per slot.** Picking an action starts the job and returns to the switcher
  at once; the slot's row shows its progress (Starting… / Attaching… / Relaunching… / failed: reason).
- **The operator can switch to any other slot or the hub while jobs run;** switching to a busy slot shows its progress,
  then attaches when it's ready.
- **One job per slot at a time,** guarded by the slot's existing lease: a second action on a busy slot is refused with
  "busy: <job>", never queued silently.
- **Failures land on the row** (and, once pair#574 lands, as a hub lifecycle notice), never as a blocking modal.
- The CLI verbs (`couch --relaunch/--resume/--reboot`) keep their synchronous semantics; only the UI becomes async.

## Done when

- A UI test: start a relaunch on one slot, immediately switch to another and to the hub; both respond while the job
  runs, and the first row shows progress then its result.
- A second action on the busy slot is refused with the busy reason.
- A failing job shows its reason on the row; the switcher stays usable.
- Schedule after pair#556 lands (it reworks the same recovery paths).

## Plan

- [ ] Implement the per-target reducer, row progress/results and navigation intent in [the durable plan](../plans/000576-slot-jobs-plan.md), with failing ordering regressions first.
- [ ] Integrate the bounded Console execution/adoption path after #651 lands; preserve synchronous CLI callbacks and #574's final-outcome observer seam.
- [ ] Verify all action routes, targeted/race tests and recorded full suite; obtain hub's acceptance decision, close and land on SHIP.

## Log

### 2026-10-15

### 2026-10-17 — claimed and planned

- Card was unowned; claimed by pair:7 and start-plan completed. Automatic dependency refresh required one explicit start-plan retry; no machine package installation occurred.
- Main-session inspection and independent read-only subagent agree: lifecycle calls already run on workers. `dispatchMenuOperation`'s global `MenuState.InFlight` suppresses unrelated slot operations. `finishOperation` also performs implicit attach on Console.Run, and installation waits for resize/registration under `c.mu`; the plan covers both seams rather than adding another executor (ARCH-DRY, ARCH-PURPOSE).
- Read PR414 at ecf055bee02b7abcc6c738993d49dcd0062d25ed. Pair:5 confirmed exact address/park-nonce binding and shared park worker reservation must survive UI cancellation. Correct menu path is `menu.go`, not `menu_ops.go`. No production work before #651 lands on main.
- Pair:3 supplied #574's observer contract: final-adoption completion observation outside `c.mu`, before menu reduction and reattach-drained; preserve one publisher. Neither peer branch is an implementation base.
- Durable plan is ready for hub approval; no code or tests changed. Proposed live check remains subject to hub scheduling, no pair:0 staging authorized.

## Revisions

### 2026-10-17 — measured structure and dependency coordination

The implementation is a per-target UI ownership and completion change over the existing bounded worker pool, not a new background executor. Keep a target reserved through adoption, retain per-row outcomes, and fence late focus/frame changes by current navigation intent. A busy-row selection waits on the existing attempt; a second mutation refuses explicitly. Cover #651 recover-park once on main and preserve #574's final-outcome notification seam on integration. CLI lifecycle verbs remain synchronous. See the durable plan for state transitions, operator projection, shutdown/uncertainty rules and deterministic acceptance cases.

### 2026-10-17 — approved display and acceptance

Hub approved the design (24b5e642-2afe-4954-8922-b53c54a01ea1): busy rows show what the action is doing and elapsed time; finished/failed actions retain one result line per row. Derive age from existing clock/tick inputs, no new polling. Live check rides the hub's next restart. Wait for the hub's #651 landing announcement before code; no staging.

### 2026-10-18 — dependency landed, implementation resumed

Hub unparked this item after #651/PR414 landed (23fccae3448f10a16aa769abf1e857d67c74733f). Fresh origin/main merged cleanly on this issue branch before production work. Re-read landed OpRecoverPark binding, captured ParkNonce confirmation and guarded preparePark execution; preserve the shared core worker reservation/cancellation and no-child recovery result. #574's lifecycle observer is now also on main, including final-adoption-before-observation ordering. Independent read-only adoption audit reconfirmed Console.Run calls finishOperation synchronously and the installer holds c.mu during Resize and presenter.Register. Using the full flow for the planned cross-cutting reducer/adoption refactor rather than relying on the initial empty-code quick-flow inference.

### 2026-10-18 — first failing regressions and plan gate

Three new reducer regressions failed for the expected current behavior: A's relaunch blocks B's switch, same-target mutation silently retains a progress notice instead of naming its busy owner, and admitted lifecycle confirmation remains open instead of returning to root. No production changes. Full-flow plan review identified two Important refinements: make the reserved navigation worker/lane contract explicit and replace prose test inventories with named-function adversarial strategies. Appended those authoritative contracts to the plan, retaining original design history. The extra worker stays navigation-only; a lost hosted endpoint transfers its owned request to the bounded lifecycle lane before resume/adoption. Aggregate queue capacity and lifecycle worker count stay unchanged.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Calibration is marked stale by estimate-source, so these unit rates remain provisional. Existing Go reducers, the bounded worker queue and Presenter API cover the stack; no new library or external API is needed. Use familiarity1.0 and the thorough-plan design multiplier0.2, with15% design buffer. v3.1 scales the raw implementation rates to40%.

- TUI state/row projection: raw design2.0 ×0.2 =0.40; raw impl1.0 ×0.4 =0.40.
- Cross-cutting menu/origin consumer migration: raw design1.0 ×0.2 =0.20; raw impl0.5 ×0.4 =0.20.
- Cross-cutting adoption/completion ownership: raw design1.0 ×0.2 =0.20; raw impl0.5 ×0.4 =0.20.
- Existing queue extension: smaller-Go-module raw design0.3 ×0.2 =0.06; raw impl0.5 ×0.4 =0.20.
- Atlas/operator guidance: raw design0.1 ×0.2 =0.02; raw impl0.2 ×0.4 =0.08.
- One close review/fix boundary: raw design0.2 ×0.2 =0.04; raw impl0.5 ×0.4 =0.20.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: tui-screen design=0.40 impl=0.40
item: cross-cutting-refactor design=0.20 impl=0.20
item: cross-cutting-refactor design=0.20 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.04 impl=0.20
design-buffer: 0.15
total: 2.34
```
