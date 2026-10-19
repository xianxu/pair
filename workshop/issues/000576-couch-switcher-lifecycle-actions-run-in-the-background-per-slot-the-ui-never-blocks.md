---
id: 000576
status: codecomplete
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-18
estimate_hours: 2.34
card_mirror: '5884c04b76356898e59e3c41211dd4b2d6b982aa' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T19:21:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 7.13
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

- [x] Implement the per-target reducer, row progress/results and navigation intent in [the durable plan](../plans/000576-slot-jobs-plan.md), with failing ordering regressions first.
- [x] Integrate the bounded Console execution/adoption path after #651 lands; preserve synchronous CLI callbacks and #574's final-outcome observer seam.
- [x] Verify all action routes, targeted/race tests and recorded full suite; obtain hub's acceptance decision, close and land on SHIP.

## Log

### 2026-10-15

### 2026-10-17 — claimed and planned

- Card was unowned; claimed by pair:7 and start-plan completed. Automatic dependency refresh required one explicit start-plan retry; no machine package installation occurred.
- Main-session inspection and independent read-only subagent agree: lifecycle calls already run on workers. `dispatchMenuOperation`'s global `MenuState.InFlight` suppresses unrelated slot operations. `finishOperation` also performs implicit attach on Console.Run, and installation waits for resize/registration under `c.mu`; the plan covers both seams rather than adding another executor (ARCH-DRY, ARCH-PURPOSE).
- Read PR414 at ecf055bee02b7abcc6c738993d49dcd0062d25ed. Pair:5 confirmed exact address/park-nonce binding and shared park worker reservation must survive UI cancellation. Correct menu path is `menu.go`, not `menu_ops.go`. No production work before #651 lands on main.
- Pair:3 supplied #574's observer contract: final-adoption completion observation outside `c.mu`, before menu reduction and reattach-drained; preserve one publisher. Neither peer branch is an implementation base.
- Durable plan is ready for hub approval; no code or tests changed. Proposed live check remains subject to hub scheduling, no pair:0 staging authorized.

### 2026-10-18 — reducer, adoption and navigation integration
- 2026-10-18: closed — Corrected code3186d65d582435877d4f33080dbdcd708265f9bb passes recorded full suite648263354a17ffbc53c1040762991a8c, exit0 in13m27s, log2526bf41a295; all shell/editor and Go checks pass, including couchcore369.183s and couchtty57.350s. CI conformance and guards green. BR-1: original reservation and Origin retained; shared exact-checkout/primary-scope aliases cover materialization before effect return and held adoption, symmetric refusals, visible progress and busy selection. Red reducer and real Console.Run regressions now pass, focused race10 passes4.205s, source/declaration guards pass. No retry or unrelated failures on corrected head. Hub approved BR-1 revision and live acceptance at next natural restart, with no staging; no live acceptance claimed. No new architectural surface beyond documented atlas changes.; review verdict: SHIP

Implemented the approved existing-worker split and per-target jobs. The hosted-navigation saturation regression first failed waiting behind lifecycle work; it now passes alongside a lost-hosted-proof transfer/overload regression. A late resume focus regression first failed by selecting its old target after navigation; current intent now fences it. Real Console.Run tests hold relaunch and adoption separately, switch to worker B and hub, deliver actual stdin, and prove late completion stays on the hub. Explicit busy-row selection lands after adoption with exactly one lifecycle/adoption call. Worker completion owns final adoption, observer ordering and exactly-once remote/shutdown settlement. Original notification captures are released without acknowledging when a switch never lands.

Legacy test migrations preserve the new contract: admitted forms close to root; failure keeps a row result rather than rebuilding a confirmation/start form; exact preview fixtures carry their accepted Worktree, and a successful start fake must actually return its typed child/address. The owner-queue fixture now pauses only lifecycle workers. Existing advice and exact-nonce/fingerprint/identity assertions remain. Early package runs timed out after waiting on those obsolete fixtures; they are retained at /tmp/pair576-couchtty-first.log and /tmp/pair576-targeted-packages.log. The latter's couchcmd package passed133.760s; no full suite acceptance is claimed by these targeted attempts. The next couchtty run completed51.426s with only12 grouped golden fixtures differing in the intentionally updated copy-help line; regenerated fixtures change that line only.

Row outcomes preserve captured label/target/path, bounded valid UTF-8 detail and one result per listed stable row; missing target results retain the exact diagnostic globally. Ctrl+Y copies the selected failure, otherwise a current error or last notice; fake-tool tests cover full text. Unconfirmed outcomes say unconfirmed. Startup adoption owns its row against a second UI action; independent rows stay available. Existing clock/tick inputs produce elapsed time. Fresh-frame applicability and final landing require captured frame identity.

Counted projection/pruning controls at32 and64 rows record exactly32/64 row visits and96/192 job visits each; production counters are nil. A Go-overlay quadratic negative control failed with9312/37056 job visits. Exact production-source inventory guards pass. Focused ordering/queue/job race cohort passes10 repetitions (5.630s), including held adoption/navigation and transfer/overload; earlier independent navigation/adoption race cohorts also passed. Core recover-park/lifecycle/confirmation seams pass12.324s. Complete current couchtty package passes53.808s (/tmp/pair576-couchtty-final-targeted.log). Full suite and close review remain next.

### 2026-10-18 — narrow display correction before full verification

PR441 server checks passed on b6534d715adae5e1aadcd186416fdddd79f1920c. Recorded suite attempt62f9bf9b8f33f1e9fca5f3412c83809b acquired after1989s and passed its initial shell/editor checks, but was explicitly cancelled through local-check cancel as superseded, not counted as a pass or a flake. A fresh-context rendering investigation found job suffixes bypassed the existing menuLabelFloor=16: an oversized failure at40 columns hid both compiler's label and selection marker. This violates the approved clipping/row-identity requirement; fix it through the existing composed-suffix rule and add real-render assertions rather than only width bounds. Preserve full retained detail for Ctrl+Y. All18 actual-render cases first failed and now pass (normal/focus, widths40/60/120, running/failed/unconfirmed); focused display/counting/copy/hosted-navigation race cohort passes10 repetitions (4.155s). The fix reserves the existing label floor after composed job/hub text without truncating stored detail. Full suite is owed on the corrected head.

### 2026-10-18 — first completed recorded attempt and lint correction

Recorded attempt337e96abb8056df574bdefeb29d257a1 on fc757733261bc1182c9d292ec8c089cf685762c7 completed the whole configured suite in12m26s. The only failure was TestNoDeclarationCarriesTwoStackedGodocs (termcmd), at console.go's arrival comment and menu_operations.go's menuOperationRows comment: newly inserted declarations stole their preceding documentation. All other packages passed, including couchcore294.939s, couchtty56.949s and couchcmd58.041s. Retained failure: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/337e96abb8056df574bdefeb29d257a1.log, log prefix03291d65e69f. This is an owned lint defect, not an unrelated flake. Moved both existing comments back to their actual declarations; behavior is unchanged. Complete termcmd and artifactpath guard packages now pass2.109s/6.156s (/tmp/pair576-static-guards.log). A new recorded suite is required on the corrected head; no pass is claimed for the failed attempt.

### 2026-10-18 — full recorded verification passed; close ready

Attempt9edee006603c6a4f05d38fb526062d7d on db6b13712f659e7212484d204b21a7520188b94d passed the entire declared suite (exit0,15m34s; queue485s; log prefix927de451f7a9). Log: /Users/xianxu/workspace/pair/.git/sdlc/local-checks/db6b13712f659e7212484d204b21a7520188b94d/suite.log. All shell/editor checks and Go packages passed, including couchcore528.733s, couchtty54.803s, termcmd10.517s and wrapcmd80.805s. Server conformance/local-gates/merge checks also pass on that head. The earlier failed attempt remains recorded as an owned comment-placement defect, not a flaky pass. The hub's quiet hold allowed this already-running suite to finish and was explicitly lifted before further work; no additional heavy run started during the hold. Plan checkmarks indicate implementation/verification readiness; mandatory close review and gated landing are still pending, not claimed. No live acceptance claimed: the hub approved deterministic headless proof and the real check at its next restart (no staging).

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

### 2026-10-18 — implementation checkpoint (not acceptance)

Full-flow change-code passed after clean plan review; estimate-quality retained an INFO about optimistic integration effort. Hub confirmed lifecycle/nav ownership and the existing worker count with shared capacity16 (6499d26c-bc34-452d-b31e-435b228803bb). Queue lane regressions passed20 repetitions and race20. Installer regressions first reproduced mutex retention and post-stop adoption, then passed repeated race checks. Worker-side adoption regressions now pass, including remote final outcome, duplicate completion and shutdown settlement (race10). Legacy menu migration ran264 tests; four remaining production defects concern leave/pass overlap, captured error labels/paths and replacement-frame selection. Broader console verification is not passing: final navigation integration remains in progress. This commit checkpoints partial implementation, not a completion claim.


### 2026-10-18 — BR-1: ownership across identity materialization (approval pending)

Close round1 returned REWORK with Critical BR-1, target-ownership-survives-identity-materialization. Fresh starts reserve StartPath while inventory later lists an address/row key; exact-key admission and projection do not recognize the same owned work. A new red regression, TestMenuStartReservationSurvivesSlotMaterialization, reproduces all four failures for both a numbered-slot row and an ordinary primary row: progress disappears, detach attempt2 is admitted, busy selection misses attempt1, and automatic address lookup misses the owner. No production fix yet. The prior suite pass remains valid evidence for the old implementation, not acceptance of BR-1.

Use one original reservation through final adoption, with derived aliases rather than a second registry or identity-changing event (ARCH-DRY, ARCH-ORDER). Retain its map key, immutable Origin and attempt. For an untyped start, the accepted absolute Worktree supplies both an exact cleaned slot path and the primary RepoScope derived by the existing pure launcher.ResolveRepoScope; spawnResolved uses exactly that derivation. Primary :0 rows are ordinary, not numbered slots. Never infer ownership from cwd, display name, path prefix or arbitrary inventory order. A running ordinary address-replacing action (the existing reboot/recover/recovery-confirmation vocabulary) similarly holds its original exact primary scope through replacement; other ordinary actions remain address-specific. A primary-scope alias matches only ordinary rows; numbered-slot rows match exact SlotPath only.

Apply one alias policy to symmetric admission overlap, row lookup/busy selection, automatic address lookup and rendering. Running ownership takes precedence over an older final result. Build indexes for running exact targets, slot paths and primary scopes once, then project rows once: O(jobs + rows), with the existing N/2N counted controls extended to materialization. If more than one ordinary row is visible in the owned primary scope, each visibly stays busy under that one reservation; never select an arbitrary one as the result. Pruning keeps the original running reservation; final outcomes still bind only to the exact returned address/stable row, otherwise the existing retained global diagnostic. No scope-wide guessed final outcome and no additional early StartedChild binding are needed: aliases cover inventory publication even before the start function returns.

| Internal state / event | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Start queued/running; row not yet listed | Existing start progress | Work elsewhere | None; same reservation remains |
| Primary/slot row appears, or ordinary replacement changes tag, during work/adoption | Same start/action and elapsed time; second action says busy | Select the busy row to wait on its original attempt | None; no second job |
| Matching completion | Existing exact result/failure projection | Existing declared row action | Existing lifecycle uncertainty only |

Acceptance adds a real Console.Run fixture with a start-result row arriving during held adoption, visible progress and busy:start, exactly one lifecycle/adoption call, and explicit busy-row selection landing only when released. Cover inventory arrival before effect return in reducers, symmetric row/start admission, original Origin unchanged, replacement-address cases, and negatives for matching cwd in a different scope, path-prefix collisions, another numbered checkout, scope versus numbered-slot identity, and independent ordinary operations. Preserve final-result pruning, counted linear scans and stale-completion/focus fences. Add the identity-transition lesson. Then rerun relevant package/race guards, a recorded full suite on the fixed pushed head, and the complete close review. No production code until hub approval of this revision; no stage or live acceptance claimed.


### 2026-10-18 — BR-1 approved and implemented; verification in progress

Hub approved the revision (d2210cda-203e-4e02-87d4-8578ff850e36), with an explicit stop if reservation identity needs another design change. The repeated full-flow change-code gate passed plan-quality CLEAN (round5); estimate-quality remains INFO for optimistic integration/verification coverage, without changing the original estimate after observing actuals. Verification is part of the existing implementation/review allowances; its observed cost remains in actuals rather than being back-fitted into the estimate.

The original reservation key and Origin now survive materialization. Pure derived aliases supply exact numbered checkout or primary scope; symmetric overlap covers creation/replacement versus existing exact-address jobs. Rendering indexes jobs once and rows once; all ordinary rows in an owned primary scope show the same running owner, which outranks older final results. Existing pruning retains the original Running keys and uses exact final identity, with no guessed scope-wide outcome. No new event, stored identity, callback or early adoption binding was introduced.

Reducer regressions first failed for both row kinds; the real Console.Run held-adoption regression first failed for the missing primary owner. The latter's first green-path attempt exposed a fixture mismatch: multi-character input is split by Console, while the reused helper compares the final individual write. A single character follows the existing helper contract and preserves actual child-input verification. No product deadline or behavior changed. Menu regressions pass0.386s; focused alias/cost/real-Run cohort passes race10 (4.205s). Exact source-inventory and declaration guards pass artifactpath5.709s/termcmd2.138s. N/2N materialization controls record32/64 row and32/64 job visits, preserve original reservation maps, and include shared-scope multiple rows plus negative identity controls. Full package, new recorded suite and complete close review still owed.


### 2026-10-18 — BR-1 corrected head full verification passed

Recorded suite648263354a17ffbc53c1040762991a8c passed on3186d65d582435877d4f33080dbdcd708265f9bb, exit0 in13m27s, log hash prefix2526bf41a295. Retained log: /Users/xianxu/workspace/pair/.git/sdlc/local-checks/3186d65d582435877d4f33080dbdcd708265f9bb/suite.log. All shell/editor checks and all Go packages passed, including couchcore369.183s, couchcmd99.201s, couchtty57.350s and wrapcmd66.942s. Server CI is green: conformance2m22s, merge-check1m10s and local-gates11s. No unrelated failure, cancellation or retry on this corrected head. The initial close overlap refused the still-unchecked verification row before review; it was not bypassed. Verification readiness is now checked; the full close review must dispose BR-1 and gated landing remains required. No live acceptance claimed; the approved next-restart arrangement is unchanged.
