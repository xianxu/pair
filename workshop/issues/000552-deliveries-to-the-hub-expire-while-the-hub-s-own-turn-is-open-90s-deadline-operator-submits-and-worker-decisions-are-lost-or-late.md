---
id: 000552
status: codecomplete
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours: 2.112
card_mirror: '75b2413ae683e366b7571d4152050be602fb4bd2' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T11:23:25-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 1.05
---

# Deliveries to the hub expire while the hub's own turn is open (90s deadline): operator submits and worker decisions are lost or late

## Problem

2026-10-14: deliveries into a slot's agent pane can wedge in "waiting for previous automatic input to clear" with an
empty composer and no turn open, until each message's 90s deadline expires. Seen twice:
- **parley.nvim:1** from ~07:40 (after the TL drove `/status` into its pane with raw zellij keys): every Couch
  delivery expired, and `couch --relaunch` refused "busy" (composer holds text / input seen) though the composer was
  empty.
- **The hub, ops:0**: the operator's session-sync submits "did nothing", worker decision questions to the hub expired
  (ariadne:1's #240, twice), and `--actors` showed ops:0 `pending` with allowance 64/64 (not the leaky bucket).
  The TL was **not** mid-turn. The operator typed into the hub's draft pane and three queued messages arrived right
  after: a genuine operator input reset the wrapper's "previous automatic input" state.

So the wrapper's automatic-input gate can stay set after the input it waits on is gone, and only an operator
keystroke clears it.

## Spec

Find what sets the "previous automatic input" gate and why it doesn't clear (input injected outside the wrapper, a
delivery whose echo or settle signal was missed, a picker or overlay drawn and dismissed). The gate must clear from
observed composer state (empty composer, no turn) within a bounded time, not only on operator input. Hub-bound
messages and the operator's own submits must never expire because of it. `--message-status` should name the gate's
cause, and `--relaunch` must not report "composer holds text" when the composer is empty.

## Done when

- A reproduction (raw key injection, then a normal delivery) wedges on main and delivers after the fix (test).
- With the composer empty and no turn, a pending delivery goes through within the bound, with no operator keystroke.

## Plan

Durable design: [automatic-input plan](../plans/000552-automatic-input-plan.md).

- [x] Fix capitalized Codex status labels and confirmed-empty lease completion; both root regressions pass with tripwire recovery disabled.
- [x] Add a bounded, safe tripwire with detailed cause, local screen diagnostic, owning-hub notice and supervisor desktop alert; prove admission and shutdown behavior headlessly.
- [x] Replay the preserved parley recording read-only, update atlas and finish the full suite for acceptance; SDLC close/landing follow those checks.

## Log

### 2026-10-14
- 2026-10-14: closed — Complete make -k test exited 0, no failures/exclusions (/tmp/pair552-full-suite.log; couchcore 356.652s, couchcmd 63.254s, couchmessage 7.324s, wrapcmd 45.293s). Targeted wrapper/supervisor/protocol race tests passed. Both root regressions proved red then green without recovery and also pass with recovery enabled. Preserved parley raw replay 6752804 bytes at 94x39 reads complete Empty and relaunch-idle accepted. Latest origin/main already integrated. TL accepted headless evidence and authorized landing; original wrapper untouched, live relaunch/delivery check is TL-owned after main rebuild.; review verdict: SHIP

- Investigation: original wrapper pid 84975 preserved. Production read-only Observe/Tail agrees with raw replay at 94×39. `/status` capitalized field labels cause `codexStartupStatus` to return pending despite an active empty composer; removing only the title row yields Empty. Sanitized scratch regression fails current code.
- Investigation: real dispatch paste → occupied echo → submit → empty confirmation → Submitted leaves phase AwaitingEmpty with fence=epoch=3. A distinct next delivery, without another render, waits at the exact reported gate. Scratch regression fails without any fallback present.
- Coordination: initial claim waited for another live SDLC merge; claim subsequently succeeded. Missing caller registration and temporary missing couch command prevented earlier reports. Restart report 5d19956a-58f8-4aa6-8063-869bdecb98a1 queued successfully; TL receipt e4f76aee-471c-4759-b27c-79af37dc4ba8 confirms both root causes and requests this plan. Refreshed start-plan succeeded on the issue branch; no production edits.

### 2026-10-14 — implementation checkpoint

TL approved the plan in 323226c7-e429-4a71-97cb-d259e135af11. Full-flow change-code passed after PQ-1/PQ-2 were addressed and the 2.112h v3.1 estimate was derived; estimate-quality INFO notes reporter integration is tight and shipping verification belongs to the test/review rows. Root regressions failed on refreshed main, then passed with only case-insensitive field labels and same-observation Empty completion added. No recovery existed during this red/green proof. Turn-only and Indeterminate cases retain AwaitingEmpty. Targeted wrapper/message tests passed; full wrapper package passed in 45.183s. Baseline /tmp/pair552-root-red.log; green /tmp/pair552-root-green.log and /tmp/pair552-root-wrap.log. Tripwire model tests are next; backend adapter is independently in progress. Original wrapper untouched.

### 2026-10-14 — tripwire and preserved replay

Pure eligible-time transition, same-snapshot completeness, detailed gate causes, bounded runtime reporter and supervisor-owned notifications implemented. Root tests explicitly disable recovery. Runtime cancellation/blocked IO/timeout/overflow and backend high-water/authority tests pass under race detection: wrapcmd 3.790s, couchcmd 2.043s, couchmessage 1.225s. Full artifactpath package passed in 4.140s. One reporter cancellation test initially exposed selection of a queued item after cancellation; recording cancellation and suppressing the RPC at dequeue resolved it. Full-binding hashed occurrence IDs also fix PID-reuse collision before landing. Original parley raw replay passed: 6,752,804 bytes at 94×39 now reads complete Empty and relaunch-idle accepted; no original process input, attach or restart. Evidence /tmp/pair552-final-target-race.log and /tmp/pair552-preserved-replay.log. Atlas updated; full suite next.

### 2026-10-14 — acceptance and rollout decision

TL receipt 1625cf7b-db6e-4d6d-8442-220ff0eacc93 verifies dd00dc23 and accepts read-only replay plus disabled-recovery regressions as strong pre-landing evidence. Finish the full suite and close review, then land normally. No stage: pair:0 is reserved for #528. TL owns the post-landing live check after main is rebuilt: relaunch parley.nvim:1, then send one normal Couch delivery. Preserve its current wrapper until that TL-run rollout. The full suite is currently running, after 382 seconds in the machine queue.

### 2026-10-14 — full verification

Complete `make -k test` exited 0 with no failures: /tmp/pair552-full-suite.log. The machine gate queued it for 382s; all shell/terminal checks and `go test ./... -count=1` completed, including couchcore 356.652s, couchcmd 63.254s, couchmessage 7.324s and wrapcmd 45.293s. No known failure/flake exclusions needed. Targeted race checks and preserved raw replay passed as recorded above. Root tests also pass through a temporary test overlay with recovery enabled, complementing committed disabled-recovery regressions; production source unchanged. Latest origin/main merge reports already up to date. Preparing the one mandatory close review, then authorized normal landing; live rollout remains TL-owned after landing.

## Revisions

- 2026-10-14 — add (TL, from the code): the gate is `automaticInputAwaitingEmpty` (wrapcmd/automatic_input.go). After an automatic submit it clears only when the render epoch passes the recorded fence AND `peerComposerState(agent, snapshot)` reads Empty. Check both causes: a render-epoch increment no longer emitted (pair#522 removed an observer that "held the only render-epoch increment peer delivery needs"), and a composer parse that never reads Empty around pickers or "Model changed" lines. Give AwaitingEmpty a bounded exit: release with a diagnostic once the box parses empty or the agent is idle for N seconds.

- 2026-10-14 — revise (operator): **root cause first; no silent release.** (1) Reproduce (raw-key drive then a normal delivery; or the hub case) and fix the actual cause; the reproduction must pass with any automatic release disabled. (2) Keep a bounded release only as a tripwire: each firing sends a `couch --notify` desktop alert to the operator, a hub notice and a diagnostic with the screen state; after the fix it should never fire.

- 2026-10-14 — root causes confirmed with TL; plan both fixes and the exceptional tripwire. Parse status labels correctly and let the confirmation's own post-submit empty evidence satisfy the lease. Proposed 30-second tripwire interval measures eligible blocked delivery time, not quiet time since completion. Every firing records bounded screen evidence and requests owning-hub and desktop notification; uncertain writes or an unrecognized/occupied composer never gain permission from elapsed time. The original parley.nvim:1 wrapper remains untouched through local verification. See the durable plan for transitions, budgets, projection and acceptance tests. Plan awaiting TL approval before change-code.

- 2026-10-14 — plan-quality refinement: name risky functions and adversarial test strategies instead of prose case inventories (PQ-1). Define monotonic per-wrapper occurrence sequences and atomic supervisor high-water rejection, including old duplicate after new occurrence and same-binding reconnect (PQ-2). Bounded suppression records persist through disconnect and are superseded by a verified replacement; explicit 128-slot saturation refusal preserves duplicate evidence. Existing approved release policy is unchanged.

- 2026-10-14 — TL accepts headless evidence for landing after full suite and close review; live verification moves explicitly after landing and pair:0 rebuild, run by TL. No stage or live-wrapper mutation from this slot. Clarify the last implementation checkbox as pre-close verification, avoiding a circular claim that landing happened before the close gate.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only; calibration source is marked stale, so rates are provisional. Familiarity 1.0. Thorough executable plan applies the v2 design discount ×0.2 and design buffer +15%; implementation values are the v2 rates ×0.4. No novel external stack: existing terminal, transport, supervisor and presenter seams are reused.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.03 impl=0.20
item: smaller-go-module design=0.03 impl=0.20
item: greenfield-go-module design=0.20 impl=0.32
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.02 impl=0.04
item: milestone-review design=0.02 impl=0.20
design-buffer: 0.15
total: 2.112
```

Rows respectively cover status parsing, completion-evidence wiring, the pure lease/tripwire transition, typed report/protocol validation, supervisor notification admission, runtime reporter/shutdown, atlas and one close review. The first two use v2 smaller-module design 0.15 and implementation 0.5; the next uses greenfield design 1.0 and implementation 0.8; three adapters use smaller-module design 0.3 and implementation 0.5. Atlas uses 0.1/0.1; review 0.1/0.5. Design sum 0.48 ×1.15 plus implementation sum 1.56 = 2.112 hours. No arbitrary total or additional discovery allowance.
