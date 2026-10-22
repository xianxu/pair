---
id: 000722
status: working
deps: [ariadne#506]
github_issue:
created: 2026-10-19
updated: 2026-10-22
estimate_hours: 18.3755
collaboration-mode: collaboration
card_mirror: '5cc2c0564ab8f73e5cbfe2dfb56f8bddb6123e50' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T23:16:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 17.18
---

# couch --drain [--fast]: a fleet-wide checkpoint-and-stop before a restart, with a drain board

## Problem

There is no safe, on-demand moment to restart the machine or Couch. Work never ends, so "safe to restart" has to be a
state the fleet can reach on request. Today a restart is a crash (10-19 15:55) or the TL pinging slots one by one,
and nobody can read off whether it's safe. Operator item 15 (10-19), discussion:
ops `teams/devinfra/tl-status-xian-xu.side/2026-10-19-15-fleet-drain.md`.

## Spec

Current registered-operation addition (TL5f7e03f6/d8fba425): start/lift register a caller-scoped parent and every child ID before publishing flags, acknowledge immediately, and admit targets independently through four owned workers with separate2s budgets and per-target FIFO. Operation status retains each child result; flagged-slot status adds delivered/pending/failed with explicit uncertainty. No timeout extension. Lift explicitly needs no checkpoint/recovery action when no drain notice arrived. The exact latest plan Revision governs; watched acceptance must be rerun before landing.

`couch --drain [--fast]` marks slots with an in-memory requested-posture flag, sends an authenticated system notice and keeps delivering ordinary messages. Agents own checkpointing and reply to new work with ordinary acknowledgement text; Couch neither requires nor parses those replies. `--drain --status` lists flagged slots with idle/quiet observations from existing in-memory Couch state. One compact global badge, no per-chip drain label.

`couch --undrain [--message TEXT]` clears flags and sends the generic “drain lifted; continue your current work” plus the steward's optional note. No drain persistence, formal ack CLI, completion barrier, dispatch/lifecycle hold, automatic Esc, automatic queue pause/resume or automatic recovery. The steward drives SDLC pause/resume and judges restart readiness. Startup safe mode/restart-kind is #739; global park is #740.

TL831e9502 confirms operator approval of the re-scoped plan: exclude the issuing steward hub from flags and drain notices; other connected hubs remain selected. Edges 2–5 are approved as proposed. The issuing steward keeps driving under the separate tl-lead rule, ariadne#506. #622 has landed, freeing canonical pair:5 for the renewed change-code gate.

## Done when

- In-memory flags plus ordinary authenticated system notices work through actual steward CLI and admitted slot routing; restart creates no flags and never reads/writes drain.json.
- Ordinary peer/system messages, receipt inspection, quit and detach remain usable while flagged. Agent acknowledgement text is never parsed into Couch authority or readiness.
- Status reports current flagged slots and existing idle/quiet evidence without new sampling, queue calls or filesystem IO; unknown/stale/disconnected evidence remains explicit. No safe-to-restart or all-drained verdict.
- Undrain clears flags regardless of notice outcome, then delivers the generic continuation and optional steward note through ordinary receipts.
- One width-bounded global badge leaves actor chips and selection intact.
- Pure/service/parser/renderer regressions and a bounded disposable installed-Couch test qualify actual caller registration, message flow, undrain, quit/detach and restart. Retain exact-head argv/result/log/hash evidence. No real fleet mutation before renewed review and TL/operator-watched acceptance.
- Remove the obsolete coordinator, store, automatic queue/interrupt/revocation/ack protocol and their dependent hooks/tests. Keep unrelated terminal output draining untouched. #734 is retired, not a dependency.

## Plan

- [x] Operator approved the re-scoped Revision through TL831e9502, excluding the issuing steward; reopened through the setter in canonical pair:5 after #622 landed. Renewed change-code follows.
- [x] Remove the obsolete completion/persistence/control machinery; implement only transient flags, ordinary notices/status and undrain with tests.
- [x] Qualify message/lifecycle independence, restart emptiness, truthful idle observations and one compact badge through normal tests and the bounded disposable installed composition.
- [x] Record the passing full suite on the final code. Mandatory close review, authorized staging and TL/operator-watched live acceptance remain subsequent workflow gates before ordinary landing.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Calibration is stale per estimate-source, so this is provisional focused ship-time, not a calendar promise or a prediction of the #481 wait.

Decomposition follows the approved plan: initial spec/coordination uses issue-spec design1.0×0.5 and impl0.3×0.4. The pure drain model and supervisor coordinator are two greenfield Go single concerns (design1.0/1.5×0.2, impl0.8×0.4 each). Queue adapter, broker control authority and wrapper interrupt are three API integrations (design1.5/2.0/1.5×0.2, impl1.2/1.5/1.2×0.4). Portable store extends the existing namespace/atomic-write patterns (design0.3×0.2, impl0.5×0.4). Board UI extends current console rendering (design1.0×0.2, impl0.8×0.4). Docs use design0.15×0.2, impl0.2×0.4; one close review uses design0.2×0.2, impl0.5×0.4; real queue/endpoint conformance uses one discovery budget impl0.6×0.4. Existing Go JSON/context/atomic storage and Couch broker/registry/console seams avoid a novel stack; no library replaces this project-specific generation/ownership protocol. Familiarity1.0, thorough-plan design buffer15%, no parallel execution discount. Design2.33×1.15 + impl3.36 =6.0395h.

### 2026-10-22 — active forecast after registered-operation approval

Plan-quality round9 is CLEAN. Address its nonblocking estimate-quality INFO explicitly rather than claiming the historical14.947h covered a new executor. Same Method A/v3.1 and stale calibration: pure receipt model design1.0×0.2/impl0.8×0.4; bounded admission module design1.5×0.2/impl1.5×0.4; shared registry/auth/flag integration design0.75×0.2/impl1.5×0.4; status/CLI/consumer integration design0.5×0.2/impl1.0×0.4; fresh transport/recorded/watched execution impl1.0×0.4; renewed review design0.2×0.2/impl0.5×0.4. Existing broker, operation reducer and ordinary receipt transport avoid a new stack. Incremental design0.79×1.15+impl2.52=3.4285h. First machine block below adds these to the historical forecast, giving18.3755h cumulative forecast. Historical allowances are provenance, including superseded work, not measured spending or a remaining-work promise; no actual hours are typed. External queue/operator waits are excluded. This separately covers renewed proof/review rather than double-crediting earlier green evidence.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.5 impl=0.12
item: greenfield-go-module design=0.2 impl=0.32
item: api-integration design=0.3 impl=0.48
item: smaller-go-module design=0.06 impl=0.2
item: api-integration design=0.4 impl=0.6
item: api-integration design=0.3 impl=0.48
item: greenfield-go-module design=0.3 impl=0.32
item: tui-screen design=0.2 impl=0.32
item: atlas-docs design=0.03 impl=0.08
item: milestone-review design=0.04 impl=0.2
item: real-api-discovery design=0 impl=0.24
item: api-integration design=0.3 impl=0.6
item: greenfield-go-module design=0.3 impl=0.32
item: smaller-go-module design=0.06 impl=0.2
item: api-integration design=0.2 impl=0.4
item: api-integration design=0.2 impl=0.6
item: real-api-discovery design=0 impl=0.24
item: issue-spec design=0.25 impl=0.12
item: api-integration design=0.2 impl=0.8
item: smaller-go-module design=0.06 impl=0.2
item: api-integration design=0.2 impl=0.6
item: tui-screen design=0.1 impl=0.32
item: api-integration design=0.3 impl=1.2
item: atlas-docs design=0.04 impl=0.12
item: milestone-review design=0.04 impl=0.2
item: real-api-discovery design=0 impl=0.4
item: greenfield-go-module design=0.2 impl=0.32
item: greenfield-go-module design=0.3 impl=0.6
item: api-integration design=0.15 impl=0.6
item: api-integration design=0.1 impl=0.4
item: real-api-discovery design=0 impl=0.4
item: milestone-review design=0.04 impl=0.2
design-buffer: 0.15
total: 18.3755
```

### Active cumulative estimate revision — 2026-10-20

Renewed plan-quality round4 passed before this derivation. The original prose and machine block below are retained as provenance; this first block supersedes their total for the approved epoch mechanism. Same v3.1/stale calibration, familiarity1.0 and thorough-plan buffer15%. Add: typed fence/protocol integration design1.5×0.2, impl1.5×0.4; pure generation ownership model design1.5×0.2, impl0.8×0.4; epoch/schema store extension design0.3×0.2, impl0.5×0.4; coordinator propagation/recovery integration design1.0×0.2, impl1.0×0.4; counted-cycle/transport/race/live-conformance execution design1.0×0.2, impl1.5×0.4; second external endpoint discovery impl0.6×0.4. The existing library/IO seams remain reusable; no library supplies this protocol-specific generation authority. These incremental primitives explicitly cover the re-plan rather than hiding it in the original coordinator allowance.

Design3.39×1.15 + impl5.72 =9.6185h cumulative idle-removed ship wall-clock, including agent execution and active verification/review, excluding external #481/operator waits and registered suite-queue waits. The added execution integration allowance includes implementing/adapting the stateful fake and 1000/2000-cycle checks, renewed races/recorded suite and hands-on isolated/live acceptance; discovery is separate. Prior completed transport checks stay evidence, but changed code requires fresh execution. This is a provisional forecast, not measured actual time or a remaining-hours promise.

### 2026-10-20 14:20 PDT — renewed cumulative forecast after approved redesign

Plan-quality round5 is CLEAN. Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`, Method A only; calibration remains stale/provisional. Prior9.6185h is retained as historical scope, including the discarded persisted coordinator; it is not a remaining-work forecast. Add issue-spec design0.5×0.5/impl0.3×0.4 for the re-scope; cross-cutting deletion/regression integration design1.0×0.2/impl2.0×0.4; pure flag helpers design0.3×0.2/impl0.5×0.4; service/CLI integration design1.0×0.2/impl1.5×0.4; compact console design0.5×0.2/impl0.8×0.4; genuine installed-fleet qualification design1.5×0.2/impl3.0×0.4; docs design0.2×0.2/impl0.3×0.4; renewed review design0.2×0.2/impl0.5×0.4; installed-path discovery impl1.0×0.4. Existing Go maps, ordinary Broker.SendSystem, cached liveness and installed PTY helpers replace novel protocol work; no library search or new stack is needed. Thorough approved design receives15% design buffer, familiarity1.0, no parallelism discount. Incremental design1.19 and impl3.96 yield5.3285h; cumulative design4.58×1.15+impl9.68=14.947h. Includes verification and watched-check execution, excludes external capacity/operator waits; not measured actuals.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.5 impl=0.12
item: greenfield-go-module design=0.2 impl=0.32
item: api-integration design=0.3 impl=0.48
item: smaller-go-module design=0.06 impl=0.2
item: api-integration design=0.4 impl=0.6
item: api-integration design=0.3 impl=0.48
item: greenfield-go-module design=0.3 impl=0.32
item: tui-screen design=0.2 impl=0.32
item: atlas-docs design=0.03 impl=0.08
item: milestone-review design=0.04 impl=0.2
item: real-api-discovery design=0 impl=0.24
item: api-integration design=0.3 impl=0.6
item: greenfield-go-module design=0.3 impl=0.32
item: smaller-go-module design=0.06 impl=0.2
item: api-integration design=0.2 impl=0.4
item: api-integration design=0.2 impl=0.6
item: real-api-discovery design=0 impl=0.24
item: issue-spec design=0.25 impl=0.12
item: api-integration design=0.2 impl=0.8
item: smaller-go-module design=0.06 impl=0.2
item: api-integration design=0.2 impl=0.6
item: tui-screen design=0.1 impl=0.32
item: api-integration design=0.3 impl=1.2
item: atlas-docs design=0.04 impl=0.12
item: milestone-review design=0.04 impl=0.2
item: real-api-discovery design=0 impl=0.4
design-buffer: 0.15
total: 14.947
```

### Previous accepted machine block — historical provenance

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.5 impl=0.12
item: greenfield-go-module design=0.2 impl=0.32
item: api-integration design=0.3 impl=0.48
item: smaller-go-module design=0.06 impl=0.2
item: api-integration design=0.4 impl=0.6
item: api-integration design=0.3 impl=0.48
item: greenfield-go-module design=0.3 impl=0.32
item: tui-screen design=0.2 impl=0.32
item: atlas-docs design=0.03 impl=0.08
item: milestone-review design=0.04 impl=0.2
item: real-api-discovery design=0 impl=0.24
item: api-integration design=0.3 impl=0.6
item: greenfield-go-module design=0.3 impl=0.32
item: smaller-go-module design=0.06 impl=0.2
item: api-integration design=0.2 impl=0.4
item: api-integration design=0.2 impl=0.6
item: real-api-discovery design=0 impl=0.24
design-buffer: 0.15
total: 9.6185
```

### Original accepted estimate — retained provenance

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.5 impl=0.12
item: greenfield-go-module design=0.2 impl=0.32
item: api-integration design=0.3 impl=0.48
item: smaller-go-module design=0.06 impl=0.2
item: api-integration design=0.4 impl=0.6
item: api-integration design=0.3 impl=0.48
item: greenfield-go-module design=0.3 impl=0.32
item: tui-screen design=0.2 impl=0.32
item: atlas-docs design=0.03 impl=0.08
item: milestone-review design=0.04 impl=0.2
item: real-api-discovery design=0 impl=0.24
design-buffer: 0.15
total: 6.0395
```

## Log

### 2026-10-20 14:46 PDT — parked for independently landed #744

Verified submitted TL2d5e80a3 approves the native-Zellij re-plan and dispatches #744 to pair:5. Park #722 with claim retained, clean committed/pushed branch and PR477 unchanged. Start #744 from main on its own branch, without importing #722 commits. Its scope is the native shared disposable-Couch harness for #722/#740/#741: isolated roots, real supervisor and N real slots, deterministic/Claude/Codex fixtures, exact cleanup and live-fleet guard. Plan first. After #744 lands, resume this branch, integrate landed main and replace its failed private acceptance prototype with the shared helper; audit both reader mappings, run exact-head installed acceptance/required suite/renewed close, then request TL/operator live scheduling. No stage, merge or acceptance claim for #722 meanwhile.


### 2026-10-20 14:44 PDT — stop at fixture third delta; revised native/shared-harness proposal

Main integrated once; focused Drain/HubTarget/Warming/Activity tests pass. New8/16 service fanout test counts exactly one reserve and deliver per selected slot, no launch/process/git re-probes, bounded use-time file checks and zero status IO. Temporary admission-error tracing was removed without committing it. Trace proved the symlinked fake had kernel P_comm=terminal-fixture while argv claimed zellij; installing a physical executable named zellij fixed real registration without a product bypass.

Run9 then established all three actors, notices, ordinary work and exact ack through real wrappers, plus normal quit confirmation. It FAILED at public detach: the fake server shares the client process group that real detach intentionally terminates. Final prototype Setsid/PTY and current-VT-frame edits only compile; no acceptance claimed. Stop under the third-delta rule and append a concrete native/shared-harness re-plan. #744 is now on main with a mandatory shared helper/new-test contract; ask TL to dispatch/resolve that prerequisite rather than continue inventing a private terminal lifecycle. Keep the #722 claim and PR, no close/stage/live action.


### 2026-10-20 14:38 PDT — disposable qualification remains failed; integrate landed main

Complete couchcmd/couchmessage/wrapcmd package runs passed (125.764s/26.477s/124.465s). Core/console agent reports complete package passes, plus final focused changed-code tests; no full repository suite yet. The opt-in installed harness now uses real Couch, Pair and wrappers under isolated roots and archives raw evidence. Numbered worker launch refused because its fixture had no remote/main or compiled setup; do not fake provisioning. Three disposable primary repos are actual hubs, exercising issuing-steward exclusion and other hubs acting in worker posture; non-hub worker ownership is not claimed.

Development run7 FAILED at90s runtime: all three real wrappers/agents launch, but CLI caller registration remains unavailable. Wrapper environments and version subprocess markers eliminate missing environment/early-return hypotheses; no PEER-start-fail is recorded. Public list and private read-only refusal diagnostics return no admission reason. Full failed log: /tmp/pair-722-drain-acceptance-development-7.log; retained artifact directory: /Volumes/TestScratch/tmp/pair-slot5/pair-drain-artifacts-3573115657. Recorded owned server generations are gone. The last cleanup-generation checks and atomic fixture inbox changes still require a fresh run. No acceptance, SHIP, stage or live-success claim.

Checkpoint the fixture, then integrate already-landed main once before final qualification/required suite. This branch predates landed Couch fixes and the new reader inventory; importing only inventory hashes would misrepresent its source. Trace actual registration/admission failure after integration, without a synthetic binding or bypass. Watched live acceptance remains blocked on successful disposable qualification and renewed close.


### 2026-10-20 14:25 PDT — transient redesign implementation checkpoint

Verified submitted TL831e9502; reopened through set-status in canonical pair:5 after PR474 merged. Renewed change-code completed (plan-quality CLEAN; estimate-quality INFO, new cumulative14.947h explicitly includes deletion, service integration and installed qualification). Old9.6185h remains historical forecast, not measured incurred work; no unexecuted old live-acceptance allowance is claimed as completed. Deletion/regression allowance includes broad message/lifecycle tests, races, lint and active suite diagnosis; installed fixture qualification remains the principal forecast uncertainty.

Replaced persisted coordinator with mutex-owned transient flags and ordinary system notices, excluding only the issuing steward. Removed drain queue, wrapper controls, broker holds, persisted state and operation guards. Undrain clears before notifications; status uses cached liveness/activity with unknown idle explicit. One compact badge; source inventory/docs/help aligned. Cap regression observed131 flags before fix; now refuses a new cohort over128 before effects. Focused service tests verify other hubs selected, steward excluded, ordinary peer submission while flagged, status with no authority IO, disconnected lift, empty lift and restart ignoring legacy bytes. Wrapper ordinary input/lifecycle checks passed; focused Drain/Activity/retired endpoint races passed across couchcmd,couchmessage,couchtty. Complete repolint/artifactpath packages passed. No full-suite or installed-fleet pass is claimed yet.

TL28cecc2b confirms installed acceptance scope: normal quit confirmation while flagged then cancel, public detach/exit and restart empty; park-all belongs to #740. Additionally exercise a flagged worker receiving ordinary work and replying with the exact ack prose through the real wrapper. Fixture must refuse outside disposable roots. Existing installed fixture substitutes the wrapper, so an explicit test-only real-wrapper mode is being qualified; no live fleet effects or staging performed. Previous ariadne:4 zero NoticeStarted discrepancy remains unreproduced; deleting the old formal ack protocol is not evidence that its cause was found.



- 2026-10-20: closed — All behavioral findings BR-1/2/3/4 addressed in prior reviews with negative controls, races, real conformance and AST enforcement. BR-5 unverifiable dispute supplies full uncached pinned31899be8 package run: go test ./cmd/internal/couchcore ./cmd/internal/couchtty -count=1 -json PASS exit0, native caps still enforced core96768000/134217728 and tty39788544/67108864, repeats1. JSON log sha256e69040f5cc38dd750769b510d24b3de942dbd2ec9b70409531c34c9af82af45f and exact head/env/argv metadata retained in /Volumes/TestScratch/tmp/pair-slot5/pair722-write-budget.ougil02v. Reviewer temp env was unrecorded, so exact discrepancy cause remains unproven; no budget raise or test filter used. Pending dispute requires explicit reviewer disposition. Code unchanged since31899be8, recorded required suite retains queue position there; current09140407b75d866a912c26cb32e435de2451e4ee adds only measured evidence and review artifacts. TL review-first receipts authorize only --no-plan-check for open live/landing items: stage only after SHIP; operator-present live check and final recorded suite still precede merge.; review verdict: SHIP
### 2026-10-19

Claimed by pair:5 under canonical TL dispatch c1f508a1. Read the operator's fleet-drain discussion and ariadne#481's existing Spec. start-plan delivered ARCH principles and created the independent issue branch from main. Its weave refresh regenerated the two CI workflows; reverted only those generated files per the standing #699 instruction, then start-plan succeeded. No implementation code, real pause, interrupt, restart or live drain was performed.

Read-only code exploration confirmed HubOwnership.Main as steward authority, exact liveCaller/Registry session evidence, Broker.SendSystem, console event publication and existing stateful fake patterns. Park/continuation execution does not provide an agent-authored checkpoint and is not reused as the drain's stop mechanism. Drafted the durable plan with all nine ARCH lenses; no estimate is derived before plan-quality.

### 2026-10-22 — resumed after #778; shared-harness qualification remains
- 2026-10-22: closed — Corrected graceful notice leaves queued work outstanding: independent complete-body tests RED on old formatter then GREEN, byte-identical to current #506 fixture537 bytes. Consumer screen regression RED with2 reads/blank redraw then GREEN retaining validated frame; acceptance oracles and reader inventory pass. Required current-code recorded suite PASS11m13s on ec88486f, attempta45eb202b109f691b0d02a890fe0933d; full log SHA25612c8f44901b69799ae07afaf510f5d14decfbb0e875a91f1fdb7e3c7d8ad3b55, no failures/reruns/waivers. Native evidence is deliberately qualified per direct TLef5588c8 Revision: prior full Claude+Codex pass on68c581ac; f39a3ae7 full Claude and corrected Codex notice/work/ack proof before separately regressed/fixed screen race; 6aada862 corrected-frame Claude notice/fast/undrain/message/badge/selection/quit/detach proofs before shared RestartSupervisor registration deadline. Both startup failures attached to #773 as related/unclassified, all cleanup passed. Complete raw logs/manifests retained in workshop/evidence/000722 and000773. NO current-source complete native pass claimed; TL directed stop native retries and review this evidence. If fresh native pass is required, report to TL, never waive/retry independently. #506 must land; fresh TL-authorized stage and operator-watched real-slot acceptance still precede merge. Historical ariadne:4 NoticeStarted discrepancy remains unproved.
- 2026-10-22: closed — Transient notice-only drain: make test-drain-acceptance passed both Claude and Codex through shared native Zellij0.45.1 fleet on68c581ac; real steward plus2workers, public submitted receipts and independent accepted bodies prove notices, flagged work/ack, undrain, one badge/selection, quit confirmation/cancel, detach, restart-empty and post-restart delivery; exact cleanup passed. Native log SHA256021376a6cc8edda5076f874a99f62c5b2e0597a869120b368193c65922e9d3b8. Required recorded suite on557da87c passed11m35s, attempt9b8b7c6dd0c1bf9a0e7f3921359c3fa4, full log SHA256ce90f6852a4fc573f909b010601f25ddec55b446f106306aed49af4989c43fce; no failures/rerun/waiver; later commits evidence only. Focused races, lint, artifact/source-reader inventory passed. Logs/manifests in workshop/evidence/000722. Old ariadne:4 notice/NoticeStarted discrepancy NOT REPRODUCED/NOT ROOT-CAUSED; removed formal ack is not a credited fix. No live acceptance claimed: ariadne#506 must land, then renewed SHIP enables a fresh hub-authorized stage and operator-watched live check before merge.

#778 landed as PR536, carddone/archive verified and merge exited0. Canonical sdlc claim resumed the held #722 branch/claim; PR477 is OPEN at parked7880cc6c2. TL next-item receipt a5d4365f supersedes #772/#740. Sent resumed-state report9b71b2cc to hub: transient product implementation exists, but private daemon/PTY acceptance failed and never qualified. Fresh native acceptance/suite/close and watched live remain outstanding; old ariadne:4 discrepancy remains unproved.

Integrated landed main in fdf65982d, including #744/#761/#778. Resolved four additive conflicts preserving both drain and new mock command/help/source inventory. Broad merge diff whitespace check named incoming raw terminal captures and an archived diagnostic patch; preserve their exact bytes. Check limited to all four conflict resolutions against incoming main passes. No test pass is claimed yet; the obsolete private acceptance caller also predates installedEnv's new signature and will be replaced, not grown. Appended the shared-consumer execution checklist to the durable plan; same approved product/state contract. Renew change-code before resumed implementation.

### 2026-10-22 — resumed implementation and component qualification

Renewed full change-code admitted implementation: plan-quality CLEAN, estimate-quality INFO. Its advisory correctly names native qualification as remaining forecast uncertainty;14.947h remains the prior cumulative forecast, not measured spending, and discarded/unexecuted acceptance allowances are not claimed as incurred. Shared-harness implementation belongs to #744, not this issue. The native consumer uses the existing qualification allowance; renewed main integration/hook/certificate work is the resumed regression task, with uncertainty retained rather than retroactively rewriting historical forecasts.

Replaced the entire private drain acceptance prototype with landed testcouch: genuine steward plus2numbered workers, both Claude/Codex, per-personality90s runtime, independent submitted receipt/accepted-body oracles, current cell geometry/selection, quit/cancel, detach/restart and legacy-byte checks. New pure oracle negatives and wide/combined-cell controls pass with the default native test skipped (0.187s); no native acceptance claimed yet. Read-only hook audit proves REAL_WRAPPER/ACCEPTANCE_DIR are used only by the obsolete test. Restored terminal_registration_fixture_test.go exactly to landed main, preserving all existing narrow fixture modes and their case catalogue. All original TestRegistration tests PASS1.451s.

Focused drain product tests PASS; broadened drain/lifecycle races PASS in core1.474s, cmd1.853s, message2.297s, tty1.518s and wrap1.742s. Full repolint13.926s/artifactpath11.975s PASS. Native consumer reader audit uses non_go_full_fallback because testcouch.Build consumes repository build inputs; recorded the one new per-file certificate via the exact scanner command, then inventory PASS. No protected gate config changed. The native snapshot serializes continuation cells explicitly, so scenario geometry and click columns count serialized grapheme cells rather than double-counting wide glyphs; pure controls cover this before native run.

Coordination: ariadne#506 remains OPEN/unclaimed and already owns the new worker/steward rules. Current tl-worker still prints removed formal ack CLI; reported this to the hub for rollout before watched live acceptance, without editing peer code. Native disposable qualification can proceed independently. Old ariadne:4 discrepancy is still unproved.

### 2026-10-22 — live-rule dependency confirmed

Authenticated TLde357e2c assigns ariadne#506 to ariadne:5 after #541 and explicitly requires it landed before watched live acceptance. Added the details-level cross-repo dependency (deps is not a tracker card field); no card mirror edited. ariadne:5 requested exact canonical notice text via a6427ee4. Supply source-pinned graceful/fast/undrain text and worker/steward response semantics; #739 remains its own startup/automation contract, never an inferred recovery permission from empty drain flags.

### 2026-10-22 — shared-native run1 failed: consumer viewport below fixture contract

Complete make test-drain-acceptance run on clean pushed638032c6183d82aa72145d22825c422fcef7651a exited2 in123.067s (Claude case112.21s including cleanup; Codex not started). Retained full log and original run manifest: workshop/evidence/000722/native-shared-638032c6-failed.log/json, SHA256fd79ab1be1fba5dab62e08e4d711ce6224854dfec8b19625262697495067fdec. Source tool/version manifest, genuine identities, public receipts/accepted bodies and current frames are in the log. Graceful notice, ordinary flagged work and exact worker ack submitted and independently appeared in actor input ledgers; selected worker1 frame has one drain2badge. No full acceptance claimed.

The pre-cleanup pane capture explains the next undrain timeout: all3agent panes contain fixture resize outside supported dimensions. Shared agentfixture/run.go:217 rejects cols<20 (or rows<10), and the consumer resized the outer native Pair to30columns while Pair splits that geometry between panes. The exact transient child dimensions were not retained; the pane error proves the supported envelope was violated, rather than identifying which minimum was crossed. All fixture agents exited, so the next mailbox command could not execute and the90s caller deadline expired with typed cancellation uncertainty. The shared Close then proved all owned generations absent and joined child/IO before deleting the root; no live fleet effect. This is invalid consumer geometry, not a drain delivery-hold finding or a timeout-budget defect.

Use80and120columns for native integration, within the existing fixture envelope; keep pure ConsoleDrainStatus tests across widths1..139, including narrow30. No fixture guard, product contract, public protocol or90s runtime change. Existing end-to-end later fast/lift/worker-delivery actions verify the agents survive resize. The original failed run remains the RED evidence for the invalid geometry; no new test that merely asserts the chosen constant is added. Refresh only the changed native-consumer certificate, run its cheap oracle/compile checks, commit/push, then rerun on the corrected clean source.

### 2026-10-22 — native shared-fleet acceptance PASS, both personalities

make test-drain-acceptance on clean pushed68c581ac8d55bc71ec56f3b60bd2237d72bdc21f passed exit0: Go package126.419s, whole invocation127.854s. Zellij0.45.1, genuine Claude2.1.237 and Codex0.152.0 fixture schemas/captures, real Couch/Pair/wrappers and provisioned steward+2workers each. Runtime after setup was46.677s Claude and52.340s Codex, below the unchanged90s bound; full subtests including setup/cleanup55.75s and61.74s. Full raw log/manifest are native-shared-68c581ac-pass.log/json, SHA256021376a6cc8edda5076f874a99f62c5b2e0597a869120b368193c65922e9d3b8. Manifests now use their transported log basename; source/exit/time/hash values are preserved.

Both cases prove issuing-steward exclusion, worker status with unknown idle remaining unavailable, graceful/fast notice receipts plus exact independently accepted envelopes, ordinary flagged work and exact worker ack text, generic/custom lift, one current80/120-column badge and real chip selection, normal quit confirmation/cancel while flagged, public detach and same-root restart with zero flags, and post-restart ordinary delivery. Opaque legacy drain.json is unchanged and no new drain file exists. Claude rootpc-2160992725/rune80e06aae77682ffb4f394b713916114 and Codex rootpc-3096060203/rund4d83cd29bfb15974452d8f72bd67316 both report all owned generations absent and direct child/IO joined.

The earlier failed30-column run is retained separately. Its generic fixture error proves unsupported nested geometry; the precise transient columns/rows were not retained, so the earlier hub shorthand<20columns is an inference rather than an exact dimension capture. The80/120 run validates the corrected consumer envelope; pure renderer coverage remains1..139. No fixture guard or deadline was raised. Old ariadne:4 NoticeStarted contradiction remains NOT REPRODUCED/NOT ROOT-CAUSED; new notice-only delivery success does not resolve a removed formal-ack counter's cause.

Implementation/native checklist now complete; required fresh recorded suite, renewed close, ariadne#506 landing and TL/operator-watched live acceptance remain. Commit/push this checkpoint before queueing; do not stage or merge without those remaining gates.

### 2026-10-22 — final-code recorded suite queued

Published native-evidence head557da87cca104ea388821c10c1042760078251e4 to PR477; behavior is unchanged from native-qualified68c581ac. Required suite lease4130677290, attempt9b8b7c6dd0c1bf9a0e7f3921359c3fa4, is retained in session83028 on557da87c; checkout /Volumes/TestScratch/local-checks/sdlc-local-703820345. Initial position3 behind active ariadne:5 and queued ariadne work. Keep the runner; no continuation/restart/duplicate while queued or running. The ordinary affected-dispatch Python preflight passed13tests in12.955s before queueing. Full suite and close are still pending; no stage or live drain authorized.

### 2026-10-22 — final-source recorded suite PASS

Retained session83028 completed exit0. Required suite on pushed557da87cca104ea388821c10c1042760078251e4 passed695.370s (11m35s), after925.319s queue wait; attempt9b8b7c6dd0c1bf9a0e7f3921359c3fa4, lease4130677290. Full raw log and original completed attempt receipt retained as workshop/evidence/000722/suite-557da87c-pass.log/json; SHA256ce90f6852a4fc573f909b010601f25ddec55b446f106306aed49af4989c43fce recomputed and matched. No test failure, flake rerun or waiver. All Go packages passed, including cmd/couch6.899s, couchcmd52.279s, couchcore217.515s, couchmessage26.051s, couchtty41.321s, testcouch21.431s, wrapcmd57.769s and repolint6.801s. Ordinary shell/editor/Python groups passed; affected-config upstream-source qualification and real-docflow smoke reported their existing unavailable-peer-source skips, not new acceptance evidence. Native acceptance remains the separately recorded Claude/Codex pass on68c581ac; all subsequent changes are evidence/checklist only.

Implementation verification checkbox now complete. Renew mandatory full close; historical persisted-drain SHIP does not apply. No stage/live action authorized, no live acceptance or landing claimed. ariadne#506 must land before the TL/operator-watched run. The old ariadne:4 notice/NoticeStarted contradiction remains NOT REPRODUCED/NOT ROOT-CAUSED, explicitly carried into close evidence rather than credited as fixed.

### 2026-10-22 — renewed SHIP; waiting for watched-run go

Full close session30510 completed exit0: round6 SHIP, high confidence, no Critical/Important/Minor findings, all five historical findings disposed, codecomplete published. Reviewer freshly passed complete couchcore/couchmessage/couchcmd/couchtty/wrapcmd packages and verified retained native/suite hashes. Close evidence a75ef8fb911db7e41f0fa754b909b0ed4252b539; Git HEAD after mirrored card e506105de9a83599abd396840e735a60fab3c5be. Outcome report2fd3318c submitted to hub. Review produced no new lesson beyond the retained rules; no implementation change followed the tested/native-qualified source.

Authenticated submitted TLbece403b approves the watched script and says STAND BY for a fresh go. Schedule: on pair:0 after #753, #739 and #759 operator checks, before #737 day-long run only if ariadne#506 has landed; otherwise after #737's day. At go: refresh native pre-stage evidence, sdlc stage --build and report printed revisions; TL drives external owner-scoped queue pause and graceful notice, confirms issuing-steward exclusion/status/one badge, ordinary work and prose acknowledgement from flagged Claude and Codex, then undrain and continued delivery. No stage, live drain/queue operation or merge is authorized now. Claim and candidate retained; no background check remains. Old ariadne:4 counter discrepancy remains unresolved and is not credited as fixed.

### 2026-10-22 — graceful queued-work clarification RED/GREEN

Submitted peer500c1056 relayed #506's running-versus-queued clarification; direct submitted TL e220cac5 approved the exact source wording. Final vector sent to ariadne:5 as2d574326, submitted, for byte-for-byte rule alignment. Renewed change-code completed: plan-quality CLEAN round8, estimate-quality INFO (historical forecast allowances may overlap; renewed qualification/review allowance uncertain). Keep historical forecasts distinct from measured actuals; no hours invented or unexecuted allowance claimed as spent.

Before production edits, independent full expected bodies failed in both TestGracefulDrainNoticeLeavesQueuedWorkOutstanding and TestDrainAcceptanceGracefulBody: old output said current background work and omitted leaving queued work outstanding (go test core/cmd/couch exit1). Changed only the production graceful opening; fast and shared checkpoint/ack prose unchanged. Both exact-body regressions and existing notice/acceptance-oracle tests then PASS core0.289s/cmd0.193s. Broader drain tests PASS core0.294s, couchcmd0.646s, couchmessage0.256s, couchtty0.325s, wrapcmd0.443s. README/atlas now distinguish admitted running work and outstanding queued work. Native expected body remains an independent literal, now also exercised in the ordinary suite. Re-audited the unchanged testcouch repository-build inputs and regenerated only the changed reader certificate with non_go_full_fallback; make test-affected-inventory PASS.

Fresh native and recorded suite evidence on committed corrected source, then re-close, remain pending. Earlier SHIP and passes stay historical. No stage, live control or merge performed; watched live remains held for TL go and ariadne#506 landing.

### 2026-10-22 — final-source native failures retained; no acceptance credited

On clean pushedf39a3ae7bbf0933ebb1a2db038b803479e7678d7, make test-drain-acceptance exited2 in156.081s. Claude completed PASS52.01s/runtime42.392s with corrected notice accepted, while Codex steward startup failed before any drain call: context deadline exceeded, terminal admission unverified, agent-ready missing; wrapper/startup/failure identities unavailable and startup_sources0. Both exact cleanup proofs passed. Retained full native-shared-f39a3ae7-startup-failed.log/json, SHA2560341a32830cf0dbc8bc97b46a11b56eddce1ae8deba385fd3260c4ac84b6d062. TL95172ebe classifies this as shared #773-class/unclassified, not feature evidence, authorizes one same-source Codex-only invocation and forbids another blind retry. Submitted owner evidence attachment d9f19a30 to pair:7 with full capture/hash; no #773 root cause is claimed.

That one authorized focused invocation exited1 in32.943s on the exact same clean source. All three Codex actors registered, corrected notices/work/ack were submitted and independently accepted, then the consumer failed at drain_acceptance_test.go261: the preceding80/120 badge polls succeeded, but a fresh Screen read for chip coordinates saw a blank footer. The consumer checked one mutable observation and used another. Retained full native-codex-f39a3ae7-frame-failed.log/json, SHA2565012f1b53bc7288c62d5ed2adaced41c73ee4a91e7bf04cc43754f995e0557c7. Exact cleanup passed. This is a different failure, not a #773 startup recurrence or a native acceptance pass. No suite queued, timeout raised, waiver, stage or live effect.

### 2026-10-22 — snapshot consumer regression RED/GREEN

Extracted the prior poll-then-read pattern into the test consumer helper and exercised a controlled source that returns a valid frame once, then a blank redraw. TestDrainAcceptanceScreenKeepsValidatedSnapshot failed with frame empty, reads2, errnil, reproducing the native failure deterministically without a fleet. Corrected waitDrainAcceptanceScreen to read once per poll, retain the matching immutable string and return that exact observation. Both80/120 badge evidence and chip coordinates now derive from it; selected-worker and normal quit-confirmation success captures use the same helper instead of later reads. Fresh Screen reads remain only inside checks or explicit failure diagnostics, not post-validation success data. Existing polling cadence,90s runtime and4m outer timeout unchanged; no product changes.

All TestDrainAcceptance tests PASS0.188s, including exact graceful body, changing-frame regression and renderer/status negative controls. Re-audited unchanged native repository-build inputs and regenerated the single reader certificate, make test-affected-inventory PASS; diff whitespace clean. Keep both prior native failures retained separately. Next: commit/push, refresh full Claude/Codex native matrix, then required suite and re-close; no stage/live permission or acceptance claimed.

### 2026-10-22 — restart-registration failure retained; TL directs suite/re-close

On clean pushed6aada8623f438c3fc6d400faa4e6863d1678cfe0, refreshed make test-drain-acceptance exited2 in107.203s (package106.067s, Claude subtest97.96s; Codex not started). The corrected-frame Claude case completed all reached drain checks: corrected graceful notices, fast notices, flagged ordinary messages and exact ack, generic/custom undrain, exact retained80/120 badge frames, normal worker-chip selection, quit confirmation/cancel and detach. It then failed inside shared RestartSupervisor -> Start -> WaitActors: steward registration expected at least1 registered actor, caller context deadline exceeded; screen empty. This is PARTIAL behavior evidence, never a complete native pass. Failure capture found surviving worker ready records and native panes, but not the steward's current registration; exact cause remains unclassified. Cleanup proved all owned generations absent and direct child/IO joined.

Full run capture and manifest attached under workshop/evidence/000773/pair722-6aada862-restart-registration-failed.log/json, SHA256e06556d9e51a859c49f34f5902a20c03486800d2a4aedcb0e92fed98c8a0355d verified against the original. Attribution per TL: “steward registration deadline after RestartSupervisor,” related #773 startup class, not a diagnosed #722 defect. Notify #773 owner pair:7 with this durable reference. Earlier #773-class/unclassified Codex pre-agent startup failure and separately fixed consumer frame race remain retained in evidence/000722; they are not relabeled passes.

Authenticated submitted TL ef5588c8 directs STOP native reruns, record every result, then required current-head suite and re-close. Evidence distinctions: full both-personality native PASS on68c581ac predates the graceful wording clarification and snapshot-consumer fix; exact corrected notice-body regressions passed RED/GREEN; f39a3ae7 Claude completed full scenario while Codex failed startup; focused f39a3ae7 Codex registered and delivered notices/work/ack before the consumer race; 6aada862 Claude completed corrected-frame feature checks but failed restart registration. No fresh complete native pass on current source exists. Watched operator-present acceptance with real slots must settle remaining acceptance before merge. If the reviewer requires another complete native pass, bring that refusal to TL; do not retry or waive it independently. No suite, live drain, queue mutation, staging or merge started during this diagnosis.

### 2026-10-22 — corrected-source suite retained in FIFO

Required suite is queued on pushedec88486fe63198753349d1fc12e5ce7b9fed968d, behavior unchanged from6aada862 plus retained evidence/accepted verification sequencing. Session52424, lease2976977985, attempta45eb202b109f691b0d02a890fe0933d; isolated checkout /Volumes/TestScratch/local-checks/sdlc-local-2469983776. Initial position3 behind active ariadne:2 and queued ariadne:5. Preserve this runner; no duplicate, cancellation, continuation or restart while owned check is queued/running. Suite and renewed close are pending; previous SHIP is superseded by corrected notices and consumer tests. #773 restart attachment notification d5ee8343 reached pair:7 as submitted. No native runner remains and no more native matrix rerun is authorized; live/stage/merge remain held.

### 2026-10-22 — corrected-source full suite PASS; renewed close due

Retained session52424 completed exit0. Required suite on pushedec88486fe63198753349d1fc12e5ce7b9fed968d passed672.852s (11m13s), after993.556s queue wait; attempta45eb202b109f691b0d02a890fe0933d/lease2976977985. Full log and original completed attempt retained as workshop/evidence/000722/suite-ec88486f-pass.log/json; recomputed SHA25612c8f44901b69799ae07afaf510f5d14decfbb0e875a91f1fdb7e3c7d8ad3b55 matches. No test failures, flake reruns or waivers. All Go packages passed, including cmd/couch7.316s, couchcmd52.495s, couchcore170.078s, couchmessage25.093s, couchtty39.614s, testcouch20.995s, wrapcmd68.195s and repolint7.007s. Shell/editor/Python groups passed with existing upstream-source affected-config and missing-peer real-docflow smoke skips. Only issue-log metadata changed after the tested head.

Current ariadne:5 graceful fixture vector was compared with the complete production notice byte for byte:537 UTF-8 bytes, SHA256612bc49844512b2c992db2b3e973b117244c659a37411d2b2d06febf36d5fe92; peer comparison report06d83c76 submitted. This verifies current text alignment, not #506 landing. Its landing remains a live-run prerequisite.

Final-code suite checkbox now complete. Re-close using TLef5588c8's explicit evidence sequencing: retain old complete68c581ac native pass, corrected-source partial native proofs, all three later failures and their attribution; no fresh complete native pass is claimed or independently retried. If review demands another native pass, bring it to TL. No staging, live controls or merge performed. Watched operator-present current-build acceptance and historical ariadne:4 uncertainty remain outstanding.

### 2026-10-22 — corrected-source re-close SHIP; standing by

Full close session48819 completed exit0: round7 SHIP, no Critical/Important/Minor findings, five prior findings disposed. Reviewer freshly ran complete core/message/CLI/console/wrapper packages and focused races, verified log hashes, and confirmed the notice regression turns RED with the ambiguous text restored. It accepted the final TL-approved native evidence sequencing and did not require another matrix run; explicitly noted no complete current-source native pass exists and watched acceptance remains required before landing. Reviewed head221af63022ff5fcd7247e04830d6b7c9853ec0f8; Git HEAD after card mirror ba9a90c3e16d340af534b13176045d769eee8b1a, clean/pushed.

Outcome report3309e524 submitted to hub: exact537-byte #506 vector alignment, corrected-source suite PASS11m13s, all native failures attributed and retained, #773 attachment received by pair:7, no root-cause credit and no full-current-native-pass claim. STAND BY under the existing hub schedule for fresh staging/live go; no owned check/review/native runner remains. No stage, real drain/queue control or merge performed. Claim and PR477 retained; ariadne#506 landing and operator-watched live acceptance still precede landing. This post-close checkpoint is issue-log-only; no executable source changed.

### 2026-10-22 — ariadne#506 landed; waiting for staging go

Authenticated submitted ariadne:5 receipt345e697a reports PR390 landed. Independently verified ariadne tracker card done/archived/landing present and Git merge commit5b1a91057db2940c5dadca951d72277b65fa2bdb (PR390). This clears the rule-rollout dependency; retain deps for provenance. Direct submitted TL1ff56125 confirms the fleet tl-worker has the new DRAIN rule and the watched run can happen when the operator is present; TL will give a separate go to stage. This is readiness/scheduling, not staging or live-control authorization. No stage, drain, queue control, merge, new native run or source change performed. Candidate remains SHIP on corrected-source suite evidence, with the disclosed #773-class native failures and watched real-slot acceptance outstanding.

### 2026-10-22 — TL-directed current-main integration before watched run

Authenticated submitted TL b8dde491 directs preparation only: merge current origin/main, targeted checks and push; pair:0 is occupied by #753's recheck, and a separate explicit staging go will follow. Starting clean head8c09b371b88ffd3aca29b226bd6409716ccc130c, fetched mainfea66e36b0d25b3e37f2dc70a084226a8ed1697a and integrated it in place. Six additive conflicts: retain both atlas sections; both artifact-source inventories; drain mutex/flags plus editor-endpoint and proven-orphan service fields; drain light verbs plus editor-endpoint/reap timing help; both protocol payload/result fields and both closed-shape guards; drain prefix before the new wrapped-tab prefix budget snapshot. Preserve the newer two-row planner and row-relative hit spans. No new drain behavior or topology.

Targeted drain checks PASS across core0.298s, couchcmd0.608s, couchmessage2.231s, couchtty0.314s, wrapcmd0.356s, cmd/couch0.186s. Integration checks for wrapped bars/chrome, orphan protocol/reap and editor endpoints PASS couchtty0.977s, couchmessage0.165s, couchcmd0.937s. Added meaningful integration coverage: one drain badge across two rows with all four actors and valid row-relative chip clicks at widths18..24; undrain removes it. Added closed-shape negatives for legacy/proven-orphan targets and signal mixed into drain. The wrapped-bar regression turned RED when the merged drain prefix was temporarily removed, and GREEN0.335s after exact restoration; no temporary edit remains. make test-affected-inventory PASS, artifactpath15.148s/repolint14.883s PASS, resolved-file whitespace clean. No reader hash refresh or protected config edit needed.

Reset final-code full-suite checkbox to pending: earlier suite and SHIP receipts remain historical and do not qualify this newer main integration. This is targeted preparation for watched staging only; future applicable recorded gates/re-close and watched acceptance remain required before landing. No new native matrix run, stage, real drain/queue control or merge landing performed. The shared #773 startup/re-registration evidence remains unclassified. Commit/push the integration, report its Git revision, then wait for the hub's explicit sdlc stage --build go after #753 releases pair:0.

### 2026-10-22 — authorized stage/build verified; TL owns watched drain

Authenticated TL e0be8efe authorizes sdlc stage --build after pair:9 clears #753; TL0edc5a39 approves tools:1 (Claude) and parley.nvim:3 (Codex), both idle/no live item. Parley:3 has TL's queued #409 assignment, which remains ordinary work if delivered during drain. Initial stage dry-run refused the live #753 lease; preserved it. Pair:9's submitted040aff6e confirmed its owned clear completed; only then ran sdlc stage --build. No abandoned-stage override or peer transaction interference.

Stage session16167 completed exit0. Pair:0 in /Users/xianxu/workspace/pair stagedcc44f0658727c1d3a0796d3c431e3a6469a784a0; Git HEAD independently rechecked after build matches. Printed binary proof:

    bin/couch vcs.revision=cc44f0658727c1d3a0796d3c431e3a6469a784a0 vcs.modified=false
    bin/pair vcs.revision=cc44f0658727c1d3a0796d3c431e3a6469a784a0 vcs.modified=false
    bin/pair-launch-helper vcs.revision=cc44f0658727c1d3a0796d3c431e3a6469a784a0 vcs.modified=false
    bin/pair-slug vcs.revision=cc44f0658727c1d3a0796d3c431e3a6469a784a0 vcs.modified=false

Pair:5 keeps its issue branch; fresh post-build peeks confirmed both proposed workers still have finished turns/empty composers and no background-job footer. Staged outcome and exact revision lines reached TL as submittedc81810e1. Keep #722's stage lease for operator activation and the TL-driven watched check; this build result does not prove a running supervisor has loaded it. No drain/undrain, queue, restart or recovery control sent by this slot; live acceptance and renewed final-code recorded/review gates still pending before merge. No current native full-pass or #773 root-cause claim. Next: receive the TL/operator's live result, retain exact evidence, and follow its disposition; do not independently drive or clear the stage before the watched run.

### 2026-10-22 11:30 PDT — watched acceptance failed; partial fanout reproduced

Authenticated TL6d103a22/5fbe7baa records watched failure on staged cc44f065: drain e1e143f6-d8ea-4633-9848-e234e7fa4899 set29 flags but timed out; ariadne:4 received a notice while tools:1 and parley.nvim:3 did ordinary work without one. Lift35205e9e-6f62-4909-8823-a6ea083e376e cleared flags but also timed out. TLf3eb9819 confirms lifted and hand-notified recovery. Both parent IDs read unknown: current code does not register them or retain their child-ID mapping. Current latest-refusal list shows admission timeouts but cannot correlate exact live child receipts, so the precise first exhausting target remains unproven.

Root cause verified in code: default2s client/server transport wraps sequential SendSystem admissions despite handler's nominal5s context. Server drops a response after cancellation. Private real-socket TestDrainOperationSlowFirstTargetDoesNotLoseExchange (29 targets, first Observe held) RED: elapsed2.000721125s, context deadline exceeded, nil operation, zero healthy reserves/deliveries. Existing instant/direct-handler tests bypassed this boundary. No product fix or live repeat yet.

TL5f7e03f6 directs keep fleet-wide cross-hub contour, register operations/child receipts, immediate flag acknowledgment and independent per-target admission/status, then private proof and a fresh watched run. Reopened through setter to working. New appended plan Revision records state/recovery table, shared receipt lifecycle, bounded4-worker executor and ordering/cancellation proof, pending TL approval and renewed change-code. Stage remains retained unchanged on pair:0; do not land. Earlier suite/SHIP is superseded by this acceptance failure. Prior restart-resend report330a37f5 and diagnosis e392864e are both submitted. Previous terminal capture requested at recovery is unavailable (previous session descriptor absent); issue/plan/Git evidence remains the recovery record.

Submitted TLd8fba425 approved the appended registered-operation design and directed change-code; add the failed-drain/successful-lift case and explicit harmless-lift wording. Plan report d23c3630 submitted. Production code remains unchanged pending the plan-quality gate.

### 2026-10-22 — asynchronous notice fix, private proof complete; final checks pending

Change-code round9 CLEAN, estimate-quality INFO (incremental forecast now explicit), accepted design commit592ca93f5e56c88bc43a24be02a7f9e3224cc3dc. Implemented parent/child operation registration on the existing caller-scoped receipt map and reducer; the command returns after flag publication, with four service-owned admission workers and independent2s target budgets. Per-target lanes preserve start/lift order; service Close joins admissions before broker shutdown. Pure child batch reduction is O(children+events), detached and atomic; status reads local snapshots without wrapper probes. Child summaries retain IDs and explicit uncertainty, with JSON-encoded diagnostic bounds. No command timeout change, persisted state or cross-hub scope change.

Private real-socket29-target RED before fix:2.000721125s deadline, nil operation and0 healthy admissions. GREEN after fix:925.666µs acknowledgment; all28 healthy targets delivered while the first remained blocked. Added real socket write-and-disconnect lost-reply recovery, failed-drain/successful-harmless-lift, ID replay after lift without reflagging, conflicting-ID refusal, caller-scoped status, cap refusal before effects, latest-notice fencing across delayed start/lift/new-start and exact notice order. Lift wording expected-body RED against the previous source, then GREEN; invalid unbounded operation ID RED then existing shared validation applied. Executor target tests and race count3 passed11.692s; combined current drain/shared-receipt races passed7.085s(couchcmd)/1.389s(couchmessage). Five-package targeted drain checks passed; artifactpath17.104s and repolint14.486s passed, then fresh checks18.791s/20.636s passed. make test-affected-inventory passed after auditing the changed native consumer's existing private store/pane/agent IO and recording only its full-fallback certificate. One initial preflight command named the nonexistent cmd/internal/repolint path; corrected to internal/repolint, no verification credit for the setup failure.

The native consumer now follows operation IDs then independent child body receipts, but the native matrix was not rerun under TL's standing disposition. Pure/endpoint/real-socket tests qualify this fix; final recorded check, full close review and fresh watched acceptance remain pending. Stage cc44f065 on pair:0 is untouched; no live drain, queue, restart, restage or landing. Next: push the final fix, record affected (which policy may expand to suite), mandatory re-close after inventory, then report to TL for authorized restage and the watched rerun.

### 2026-10-22 12:05 PDT — round8 review and corrected-head verification

Full review round8 returned REWORK, no established Critical correctness finding. BR-6 records final-source verification pending and the reviewer's complete couchcore package hitting its existing10m timeout600.316s; no causal attribution was established. The other four complete internal packages, focused behavior/races, lint, inventory and oracles passed. Supply the exact-source managed full result before re-close; investigate if the core timeout recurs. No timer, write cap or gate weakened, and no native pass demanded.

BR-7 (Minor) is a real embedded-doc defect: the first inline status-command anchor was inside the top command table. Added a contiguous public-index regression, RED on the broken row, then moved the paragraph below the complete table. Audited README and atlas placements as well; those already sit in their intended sections. Added the prevention rule to lessons. Because embedded Markdown changes executable input, cancelled only owned queued suite lease-2848868042 through local-check cancel (no suite had started; session92025 exited on cancellation), rather than pretending its old head qualifies the fix. Will record a fresh affected/full check on the corrected pushed head. Stage unchanged, no restage/live repeat/landing. The initial normal close refused the truthful unchecked verification row; the subsequent review used only no-plan-check for authorized overlap, with the suite still required. Further re-close waits for its passing record; no SHIP claimed from round-cap demotion.

### 2026-10-22 12:10 PDT — noon main integration before corrected-head check

Fetched and integrated origin/main d6cb2ca88ae20435ebff7f48628dcd8f3ca8b1ad (includes #803/PR544 landed11:08, required by the most-recent-noon freshness gate). Git auto-merged artifact inventory and lessons additively; no conflict or manual product resolution. Targeted drain/skill/query/read/receipt checks passed: couchcmd3.721s, launcher5.802s, core9.103s, message0.184s, tty0.338s, wrapper0.413s. The initial selected regex found no tests in the new pure zellijquery package, so its complete package was then run explicitly; only that actual run supplies its proof. Current inventory PASS; repo lint19.929s and artifact classification21.257s PASS. Embedded command-index regression RED on broken row then GREEN2.908s with the complete skill/drain set. Full source verification remains pending on the corrected integrated head; no old source result is promoted. Preserve stage and wait for recorded check/re-close before requesting watched rerun.

### 2026-10-22 12:15 PDT — corrected source pushed; retained full check

Corrected integrated Git HEAD40257b270a50264eb55e7fd71730a00a3480e214 pushed; sdlc pr confirmed PR477 updated to40257b270a50. Updated PR title/body to the final transient flags plus registered delivery design and explicit pending verification/watched gates. The full new zellijquery package passed16.749s after its initial unmatched test filter; no credit for the unmatched filter. New recorded affected selection expanded to suite by Makefile.local, attempt f9a52fa035a97e231ae22409c107b1da, lease-758217443, session61908, private checkout /Volumes/TestScratch/local-checks/sdlc-local-2054765533. Initial queue position15; preserve this exact runner. No duplicate check, queue reorder or grant request.

TL report61f4325e queued with review/corrected-head/lease evidence; inspect its receipt rather than resending blindly. Earlier report ebbfeb97 also remains subject to canonical receipt confirmation, not assumed received. BR6 remains unresolved until this recorded source passes; BR7 is repaired and awaits review disposition. Next after completed suite: retain log/head/environment/duration/hash, check final Plan verification item truthfully, run affected inventory, then full re-close with all evidence. Investigate a repeating core timeout rather than change a timer or claim a blind pass. Keep pair:0's cc44f065 stage unchanged until TL authorizes replacement; no live repeat/landing before fresh gates. Do not restart/checkpoint a continuation while this owned check is queued or running.

### 2026-10-22 12:45 PDT — confirmed new registry publication race; fixed before staging

A cold private status probe demonstrated real RACE reports on newMessageService's late drainOps assignment and freshly constructed slotOperations mutex/map versus an already reachable status handler (source629fab876567f54f17ee12de78d878a2f539fa9b plus probe). RED exit1/Test0.21s/package0.652s; sanitized bounded stack excerpt workshop/evidence/000722/service-publication-race-red.txt SHA256292f229c6cb171f7d62771b3fc71163e7c4e29319e3b57e74b6f7e80c6626bc4. No live server, pane or actor was probed. This new async-registry defect cannot explain the pre-existing #773 incidents and is not claimed to do so.

Moved all new request-visible drain dependencies before the first listener publication and joined the owned executor on both subsequent constructor-error exits. The same race probe passed count3 in1.554s, followed by the broader focused drain/skill/shared-operation race set7.990s; after using the shared short canonical socket root it passed the focused set6.660s. Added Pair's drain-state-before-listen golint catalog extension: unsafe constructor cases RED, then GREEN0.142s. It checks the owned object and imported listener, ignores uncalled closure writes and rejects missing/late dependencies; its syntactic limits are explicit. Complete repo lint15.948s and artifact classification14.201s PASS; affected inventory PASS after the new tests were tracked. No existing guard was weakened and no deadline changed.

Cancelled only queued lease-758217443 through local-check cancel before changing executable input; session61908 exited on cancellation and no suite had started. The new source requires a new recorded check; no old queue place or pass is transplanted. Additional embedded-guide audit found its old vague graceful bullet and missing latest-notice wording: independent guide assertions RED before aligning to the approved queued-work rule/status contract. No formatter or ariadne#506 notice vector changed. Stage is still cc44f065, untouched; final source check/re-close/watched rerun remain mandatory.

### 2026-10-22 12:57 PDT — publication fix pushed; replacement check retained

Git-confirmed source e3c7e6badda23964b0116310fff8cdb8c94291b1 is pushed and PR477 reports that exact head. Final focused drain/skill/shared-operation race set passed7.451s after the initialization fix and guide reconciliation; lint/artifact/inventory results above remain current. Recorded affected selection expanded to suite; retained session37048 owns lease-2074561176 on this source, started12:52:31 and still queued (latest observed position12). No suite has run yet; no duplicate check. Previous PR-update terminal is no longer retained, but a fresh authoritative PR read verifies the intended head. This docs-only checkpoint does not change the tested executable input.

Reported the privately reproduced publication race/fix and current lease to TL as receipt96edacd6-e5aa-4c5d-b5f7-d2b08c3320c8, queued at publication; confirm submission rather than resend blindly. Final-source recorded pass, renewed close and watched acceptance remain pending; keep the runner and cc44f065 stage untouched.

### 2026-10-22 13:34 PDT — old live stage released for ordered smoke lane

Authenticated submitted TL55e2cb84 supersedes the old keep-stage instruction: clear the pre-fix11:11 stage now, then #753 re-look, #739 shutdown check, and #722 only after suite/re-close and renewed staging authorization. Supported sdlc stage --clear completed exit0: pair:0 restored to freshly pinned main7b9440e8c92663f8ee189f54a1260e3043a78f07, all four disk binaries report that revision and modified=false, owned stage lease released. No running slot was restarted and no drain/queue action was executed. Sent exact clear receipt to pair:9 (dd0d7af0-3984-4ac8-a200-507bb4a2949a) and hub (65fc04f6-6955-4103-9986-6f0bd363e322); query their submission rather than blindly resend.

Retained suite session37048/lease-2074561176 still waits for capacity, latest observed position9; do not restart or duplicate it. No suite result yet. The tested source remains e3c7e6badda23964b0116310fff8cdb8c94291b1; subsequent commits contain Log evidence only. ariadne#557 fleet notice was authenticated, but this checkout was clean, so no seeded-workflow repair applied.

### 2026-10-22 15:46 PDT — final-source recorded suite PASS

Retained session37048 completed exit0. Declared affected selection expanded to suite on pushed source e3c7e6badda23964b0116310fff8cdb8c94291b1, attempt d9ef653c801635ccceeb4dc03f27fd69: queue9660.140s, acquired15:33:30 PDT, PASS11m56s (715.607s), finished15:45:26. No failures, flake reruns, timeout changes or waivers. SDLC posted sdlc-local/suite on that exact source. Complete raw log and attempt metadata retained at workshop/evidence/000722/suite-e3c7e6ba-pass.{log,json}; bytes independently hash to a75339213f2f019545eb95405abb4980d455df88993bd173e27c2cd6e72cc44b. Managed pinned checkout on Mac.localdomain, ordinary declared make test; normal suite isolation and runtime environment guard passed. No private per-run cache override.

Complete couchcore passed163.798s, couchcmd53.061s, couchmessage26.020s, couchtty42.446s and wrapcmd88.730s; the prior review's core10m timeout did not recur, but its original cause remains unattributed. New source-publication race regression/catalog guard, slow-target socket/fanout tests and embedded-guide fixes are included. Later branch commits change only issue Log evidence, not executable inputs. Mark the active implementation verification checkbox complete. Next: affected inventory, mandatory full re-close (BR6 source proof and BR7 doc guard disposal), then TL-owned authorized stage/watched acceptance after #753/#739. No new native matrix run, live drain, queue control or landing is authorized by this pass.

## Revisions

### 2026-10-19 — authenticated acknowledgement and shared queue ownership

TL b2e72204 explicitly approves `couch --drain --ack ID` instead of free-text drained reports, as the only worker protocol action. The board derives idle, incarnation and SDLC quiescence; otherwise-quiet/no-ack shows stuck immediately. Workers still checkpoint as directed by the system notice; no separate hub report is required for drain completion.

TL 205542db approves the stable ariadne#481 seam: `sdlc queue pause --reason TEXT --owner TOKEN [--cancel-running]`, `resume --owner TOKEN`, and `status --json` with paused/unknown plus active suite, merge and landing owners by workspace. Resume releases only its owner's pause. The plan specifies exact argv/schema and a stateful fake; TL will copy the approved contract into #481's Spec. #481 is not implemented by this pair issue.

TL 126cc5c8 narrows the original “steward hub or operator” entry to steward CLI only: the operator asks the TL to drain. Read-only status is available to anyone. Bare outside-Couch shell mutation and a console action are out of scope; a console action can be filed separately if requested. Steward remains the first hub on the command line, reusing existing HubOwnership.Main.

The remaining draft proposes asynchronous start (the steward must also checkpoint and become idle), a supervisor-owned deadline with no automatic unpause, owner-intent persistence, fail-closed unknown effects and bounded fast interruption. These details and the user-visible state table are submitted for TL review before any code.

### 2026-10-19 — one advisory review clarifies control ordering

A fresh-context read found that the existing peer FIFO could hold the drain notice itself, that a delayed Esc could hit the checkpoint turn, and that cancellation must revoke admitted notices rather than only coordinator goroutines. Accepted all three; the durable plan's governing Revisions specify one typed generation control lane, original-turn fencing before notice admission, and wrapper-lease revocation/reconciliation before cancellation releases dispatch. Added named adversarial tests for each. This was one advisory read, not an approval loop or a replacement for change-code's plan-quality review. No scope approval, implementation or live mutation is inferred from it.

### 2026-10-19 — TL approval to enter change-code

Canonical submitted receipt 0a9b7377 approves the complete plan, four visible states (draining/stuck/drained/cancelled) and proposed async/deadline/persistence choices. TL copied the notice and SDLC seam verbatim to ariadne#481. Run full-flow change-code and its plan-quality gate before implementation; real #481 conformance still gates enablement/close. No bare operator-shell or console mutation entry is added.

### 2026-10-19 — PQ-1 test strategy correction

Full-flow change-code round1 refused only Important PQ-1 (function-level-test-strategy). Compressed task test inventories into production-function/adversarial-strategy/mechanical-guard descriptions across the plan. The approved design/seam is unchanged; implementation still waits for change-code admission.

Plan-quality round2 CLEAN disposes PQ-1 with no new findings. Derived the estimate only after that result; calibration stale warning retained. Change-code must still finish estimate admission before implementation.

### 2026-10-19 — implementation recovery checkpoint

Full-flow change-code admitted implementation after clean plan-quality and estimate gates (design checkpoint 65e2d0bb). The 6.0395h provisional estimate covers the whole issue, including already incurred design, rather than remaining implementation time. Implemented the pure state/board/notice model, owner-scoped queue adapter and stateful executable fake, portable bounded intent store, coordinator, authenticated CLI/ack wiring, broker control lane, wrapper interruption/revocation capability, console board and operational docs.

Targeted integrated verification returned PASS for couchcmd, couchcore, couchmessage, couchtty and wrapcmd using `go test ... -run 'TestDrain|TestSkillDocumentsRecovery' -count=1`. Independent broker, wrapper and coordinator race runs passed. The initial integration compile failure occurred while the shared DrainRevokeResult.Instance field was being added; the completed run passes. Final delivery and cancellation both compare the frozen wrapper instance, including same-Binding wrapper re-execution. Complete cached fleet inventory retains unregistered/unsupported live wrappers as stuck; status reads never trigger inventory scans or queue subprocesses.

This is an implementation checkpoint, not close or enablement evidence. Remaining: broader regression/race checks, three-member real-transport headless integration, real ariadne#481 isolated queue conformance, recorded suite, mandatory close review, and hub-authorized staging/operator board acceptance. No real queue pause, fleet drain, Esc, restart or slot relaunch was performed. Same-slot replacement remains conservatively stuck with cancel/inspect recovery while old effects may be unknown; cancellation never discards those effects.

Recovery evidence: `couch --peek pair:5 --previous --all` reports the previous screen unavailable; SDLC state and Git identify this issue branch (not main-slot5). The historical empty issue value was not interpreted as a new issue. Earlier #622/PR474 remains parked on ariadne#483 and #723/PR475 on ariadne#486 until TL signals; historical screen text is not new operator authorization.

### 2026-10-20 — isolated transport and integration verification

Three-member graceful and fast drains now exercise actual Server/RemoteEndpoint framing and retained outcomes over isolated owned sockets. Stateful queue suite/merge/landing work blocks completion; exact ack plus later idle completes it; cancellation preserves another owner. Lost interrupt replies cause one Esc only. Combined coordinator/transport race run passed (14.193s).

Root audit found and reproduced a same-Binding re-exec race: a replacement wrapper's final idle observation could seal the old ack after the earlier control check. The final observation now compares the persisted wrapper instance and exact member identity; mismatch invalidates the ack. The independent regression went RED then GREEN. Global stuck boards now use inspect-status recovery rather than wait.

The broader five-package run passed couchcore (133.304s), couchmessage (26.103s) and couchtty (42.591s). It exposed two deterministic branch defects: missing drain help-cost catalog entry (help panic), and wrapper doc-comment subject mismatch. Both were corrected; targeted help/skill and doc guards pass. CLI lost-ack replies now include the actual checkpoint generation (RED→GREEN). CLI fuzz passed 16,002 executions in 2s; wire-request fuzz passed 50,108 in 2s (two workers each). Repository lint catalog and slot-state coverage audit pass. Final staged-head suite remains required; these are not substituted for it.

Read-only authoritative ariadne#481 evidence at 2026-10-20T07:03:13Z still shows open, unattributed, no issue branch and not-landed. Real queue conformance and operator board acceptance remain outstanding. The earlier estimate is cumulative, not a remaining-work promise; no real fleet effects have been exercised.

### 2026-10-20 — decoder authority and persisted extension ownership

Full couchcmd regression passed (57.587s) and full wrapcmd regression passed (87.008s) after the documented fixes. A concurrent earlier run compiled the final-idle regression during its deliberately failing pre-fix window; the stable-head regression passed ten repetitions, and the full package then passed. No intermittent failure is asserted from that mixed-source run.

PR 477 opened and its server conformance, merge-check and local-gates guard passed on 5cd025c8. Before its queued suite acquired a slot, root audit reproduced two codec defects. Required exact queue keys could be overridden by case-fold aliases during typed JSON decoding, including nested owner/activity keys. Decoder now rejects known-field aliases while retaining unrelated additive fields; sixteen adversarial cases went RED→GREEN. Queue race tests passed and bounded fuzz passed 6,123 executions. Activity extensions were also incorrectly associated by array index after reorder/removal; they now follow exact ID/kind/workspace in a linear keyed merge. Reorder, retirement, new ID/kind/workspace and ambiguous identity tests went RED→GREEN; record race tests passed. Cancelled only this checkout's queued suite using sdlc local-check cancel, so the recorded suite can run on the corrected head.

Canonical TL receipt 5030d0e3 confirms #481 will be dispatched when an ariadne slot frees, and live board acceptance follows #481 with the operator present. The approved plan requires that conformance before close/enablement; PR 477 waits rather than introducing an unapproved disabled-feature split. No stage or production drain is authorized by the historical messages.

### 2026-10-20 — exhaustive source inventory integration

The recorded suite on 816a4976 finished every package and failed only artifactpath.TestProductionArtifactReferencesAreExactlyClassified: fifteen new production files were missing from the exhaustive source inventory. It recorded failure (exit2, 8m26s, log sha256 dd77bc3e972f); this is a deterministic branch defect, not a flake or waiver. All affected feature packages and the rest of the suite passed.

Classified the fifteen files in artifactpath.NonArtifactSources after checking their path and vocabulary ownership. They introduce no tag-bearing Pair artifact family; drainStore owns one bounded private drain.json under the existing Couch namespace, matching attention/resource metadata ownership. No constructor authority, family vocabulary or pre-existing-violation quarantine changed. Full artifactpath tests passed (9.114s), and the tracked-source lint catalog passed. The corrected pushed head needs a new recorded full suite.

### 2026-10-20 — full suite pass and bounded-residue re-plan

Recorded full suite PASS on d8fce3b575a117b9cc30d517d56e9b43ae4a3ddc: exit0 in8m46s, log sha25695cb7909907a, posted as sdlc-local/suite. The earlier inventory failure is fixed; no other suite failure occurred. Server CI also passed this code head. All implementation remains unclosed/unmerged pending real #481 conformance and live acceptance.

A subsequent residue audit found a separate lifecycle limit before close: wrapper revocation IDs survive every generation until its fixed128 cap. An independent temporary probe measured cancellation64 with128 retained IDs and known outcomes, then cancellations65/66 and three retries returning unknown; interrupt receipt129 is also refused. The probe was removed and tracked source hashes were unchanged. Do not claim successful repeated-drain lifecycle acceptance from the passing suite: it did not exercise this horizon.

The durable plan now contains a full re-plan proposal replacing lifetime ID retention with one persisted monotonic epoch and wrapper input-lease high-water mark/current generation outcomes. It states alternatives, authority, state/recovery, growth and test strategy. This is a new mechanism requiring TL approval and a renewed change-code gate before production edits; no such edits have begun. Shared #481 notice/argv/JSON and operator-facing scope do not change.

Canonical submitted TL receipt68935e6f approves the re-plan and asks for a counted1000-cycle bound test and the simplest rejected alternative. Both are now explicit in the plan (N=1000 and2N=2000; refuse-until-reclaim fails because safe reclamation itself needs the generation fence). Renew change-code and re-derive any revised estimate only after its clean plan-quality verdict. No production code has changed since the passing suite.

### 2026-10-20 — renewed gate and cumulative estimate revision

Renewed change-code passed plan-quality round4 and admitted the approved re-plan. Estimate-quality INFO correctly noted that the original6.0395h decomposition did not allocate the epoch revision or active final integration clearly. Added an explicitly labelled active cumulative9.6185h block while retaining the original6.0395h block and prose unchanged below it. The increment derives from six v3.1 primitives after the clean plan gate; it covers protocol/FSM/store/coordinator plus stateful validation/active acceptance and a second discovery seam. Units are idle-removed ship wall-clock including agent execution, not remaining work or dependency wait. Card setter must match the new active block before production edits.

### 2026-10-20 — epoch implementation checkpoint

Implemented the approved schema-2 monotonic intent, typed protocol-2 fences on every control/outcome and wrapper input-lease generation authority. The pure FSM retains two fixed slots; wrapper outcome maps retain only the current epoch. New generations require confirmed prior revocation. Invalid/old/foreign records lose all decoded authority and retain the dispatch hold. Status never advances epochs. Namespace, UUID, epoch, incarnation and kind are checked at transport, broker and final input boundaries. Unknown paste-only cancellation preserves quarantine; a previously handed submit may be reconciled by reads without sending more input. Added the new pure source to the exhaustive artifact inventory and updated atlas.

Counted tests cover1000 and2000 cycles with fixed state and exact effects; the old implementation failed at cancellation65. Integrated TestDrain passed couchcmd27.288s, couchmessage0.334s and wrapcmd0.791s. Full couchmessage race passed30.679s; fence fuzz passed61849 executions. Stateful service/transport race passed22.172s; expanded real three-wrapper transport race passed17.718s across opposite-mode epochs and supervisor recovery, exactly two notices and one Esc per wrapper. Wrapper final race and the new recorded full suite remain pending at this checkpoint. The d8fce3b5 suite pass predates this code and does not cover it. Real #481 conformance and operator-present live board acceptance remain required before close.

Final wrapper race selection passed12.664s. Root counted run independently confirmed1000/2000 cycles, peak/current3 outcomes and500/1000 Esc effects. Artifactpath11.571s, couchcore234.352s, couchtty41.051s and tracked-source catalog3.501s passed. Broad affected-package regression passed couchcmd112.998s and couchmessage25.720s but failed wrapcmd196.827s in TestTerminalModelControlObserverBoundsStringStorage (26 large vs6 small allocations) and TestWrapperTitleLeaseNotInherited (3.07s startup timeout). Both passed three focused repetitions on identical44d7bdccf812 in3.814s without edits; their source/tests match origin/main. Filed unclaimed #724 and registered both exact tests with this observed fail-to-pass evidence. Root causes remain unconfirmed; no suite result is inferred from those focused passes.

PR477 updated to44d7bdccf812; recorded suite requested on that exact head and queued normally at position1. Server merge-check and local-gates guard passed; conformance pending at this read. A fresh authoritative read of ariadne#481 at08:10:06Z still shows open, unclaimed, no branch or landing. No live fleet action has been performed.

### 2026-10-20 — epoch suite passed; waiting on real seam

Recorded full suite PASS on44d7bdccf8124f25d15082a7286ba1c0ff0f4a46: exit0 in8m40s after564s normal FIFO wait, log sha256466a921231a5, posted sdlc-local/suite. All packages passed, including wrapcmd54.785s; this run needed no known-flake rerun. Server conformance, merge-check and local-gates guard also passed on that code head. Later commits only record evidence. The approved1000/2000-cycle bound and simplest rejected alternative are now implemented/documented; renewed change-code passed before implementation.

TL receipt f9b3f58b dispatches ariadne#481 to ariadne:3. Replied to its contract question with exact adapter/fake/test paths, inherited environment,5s/1MiB process limits and all-three-kind completeness requirements (submitted receipt2a473297). Peer receipt e3b2faa1 says its durable plan awaits hub approval and proposes WF_SDLC_QUEUE_ROOT plus private test lock/fleet discovery scope; it will supply the implemented executable and private paths. No live pause is permitted for conformance. Wait for that isolated real-seam test, then hub-authorized operator-present board acceptance and mandatory close review. Mark #722 blocked, retain claim and PR477; no close/merge or production drain was performed.

### 2026-10-20 — #481 M2 real executable consumer conformance

Peer receipt5f63938e supplied the M2 executable and exclusive private fixture. Independently checked Go build metadata: ariadne revision b39a97b35386e2567c05862bc0b961f6e60bb8f5, modified=false; binary SHA256 c4e420fa468e7d1299b36eae8f04211dde099a1efcf2578d17431ef5ed693bc5 unchanged before/after. Ran the existing production sdlcDrainQueue adapter and strict decodeQueueSnapshot against that executable, from its private fixture cwd, with all three supplied private queue/lock/fleet-root settings. Unset COUCH_TREE only inside the fixture probe; restoring the real worker identity caused mutation refusal without a queue effect.

TestPair722RealQueueConformance PASS15.128s (test14.80s): exact two-owner pause/status/resume, literal reason preservation, idempotent replay, owner-only release, waiting exclusion, and live held private leases exposing suite/merge/landing with workspace/head. Graceful pause preserved active work; fast pause published suite interrupting while merge/landing remained running. Resume did not restart the interrupted suite. Managed lease retirement returned interrupted and retained the separate interruption record, then primary queue returned running/complete/empty. A separate private sibling queue retained all three unknown kinds after lifetime-lease loss, remained incomplete after owner pause/resume, and a corrupt lease produced an additional unknown rather than absence. This exercises queue intent/retirement, not actual suite descendant cleanup or real landing FIFO.

TestPair722RealQueueLostReplyConformance PASS4.863s (test4.51s): a private executable proxy ran the real binary then deliberately returned exit29 after successful pause/resume. The adapter correctly returned unavailable/uncertain; subsequent real status proved pause ownership, then owner absence while preserving the other owner's hold. Final primary queue was running/complete/empty; no automatic restart or blind pause replay. The no-effect proxy path delegates argv unchanged.

Temporary probes were copied to /Volumes/TestScratch/tmp/pair-slot5/pair722-realqueue.Fgg78v and removed from the Pair checkout. Original consumer source SHA256 aa072a0056f270a14aacb8233707bab5fa67b0faee2591dba367049e7661fb88; lease helper source58a2862924c7f072e8c7a673072d9c4b9b7f234dc78f33a89bb5758dd7e4bc20. Private primary fixture is free; queue-unknown-pair5 intentionally retains synthetic diagnostic residue. Consumer evidence was sent to ariadne:3 (submitted85ed0bd0). No production code, live queue, actual suite, operator terminal or :0 changed. Keep #722 blocked pending landed #481/M3 real admission/FIFO/process coverage and hub-authorized operator-present board acceptance, then close review. The recorded epoch suite still covers unchanged code.

### 2026-10-20 — updated #481 executable and legacy-waiter conformance

Canonical submitted peer receipt cee8c6a5 requested the same private checks after #481's nonblocking secondary-lock and legacy-waiter changes. Independently verified Git HEAD/build revision afb94a8e2be9d2477fbbe7b53d7b8eb7f52152c7, modified=false. Binary SHA256 e011b8266c5a26cdcca05cc8b99acdaf4b12f44af827b11855a839cb7f6e0b3f remained unchanged across checks. Created fresh private roots under the supplied fixture's consumer-repeat.2TGKnT so prior interruption IDs and synthetic unknown residue could not contaminate the repeat. Rebuilt the private lifetime-lease helper against the same source, using its real kernel process incarnation for the new binding checks.

Production adapter/strict decoder repeat PASS24.049s: core conformance19.66s and lost-response4.06s. This repeats two owners/literal reasons/replays/worker refusal, all active kinds, waiting exclusion, graceful/fast intent, interrupted retirement, unknown/corrupt residue and pause/resume reconciliation. New TestPair722RealQueueLegacyWaiterConformance PASS3.814s (test3.48s): separately held unbound token/ticket each produced complete=false, a diagnostic naming its path and active=[]; status preserved the held bytes. Releasing each restored complete=true. Token and ticket bound to the same live kernel-verified waiting lease produced complete=true, active=[], empty detail. Pause refused its admission; after releasing the pending fixture resources, resume admitted exactly one running suite, whose clean retirement restored running/complete/empty.

The first legacy probe passed every product assertion but failed its own cleanup: deferred context cancellation killed the helper before cleanup joined it. Corrected only the temporary probe's cancellation/join order; retained failed source/log and passing rerun. No #481 or Pair product source changed, no flake claimed. The original helper build also needed evidence copies renamed with a .txt suffix to avoid mixing package names in the scratch module; this was fixture packaging, not a binary failure.

All temporary Pair tests were removed. Sources and logs remain in /Volumes/TestScratch/tmp/ariadne-slot3/ariadne-481-conformance.Eb696l/consumer-repeat.2TGKnT; adapter.log SHA2561796565c7a806799b9320125bd6c64be96f4601f8b4772ba46af60fe6b2aa879 and legacy.log10dbff6effe50c3504501af01107eb030f53d87b19580145ede8858d5cd3878a. Primary fresh queue and legacy queue ended running/complete/empty; a separate synthetic unknown sibling retains diagnostic residue. Results sent to ariadne:3 receipt e3701d33. These are consumer protocol/waiter checks, not direct secondary-resource-lock or real suite-descendant/FIFO coverage. #722 remains blocked on final landed #481/M3 coverage and authorized operator-present board acceptance; the recorded epoch suite still covers unchanged production code.

### 2026-10-20 — TL orders close review before scheduled live acceptance

Canonical submitted TL receipt f35727b5 explicitly orders the remaining gates as close review now, then operator-present live board acceptance; the TL will schedule the smallest graceful drain and the next pair:0 rebuild/Couch restart. This supersedes the earlier live-before-close ordering, without removing or declaring the live acceptance complete. Keep the live-check and landing Plan items open; acknowledge just that mechanical unchecked-plan guard with --no-plan-check when dispatching close. The mandatory boundary review, actual measurement, recorded suite and all other gates remain active. A SHIP verdict is implementation review evidence, not permission to merge before the live check.

Peer receipt c7a9c68e reports #481 landed as ariadne PR341 and its final integration suite passed12 shards/1851 SDLC tests plus75 other packages in9m15s, without a flake rerun. Independently verified Git merge7836225efe51d963adb901c31181d52d0b76e415 is an ancestor of central build702b01324fa543038178e23b831d3e3346c1204a; build metadata is modified=false. Existing consumer conformance and #481's final own coverage satisfy the queue dependency under the TL's remaining-gates disposition. A read-only production queue status returned running, complete=false, no pause owners and legacy suite/landing/merge unknowns (active #483 suite and #480 landing). That is truthful incomplete rollout evidence, not a quiet fleet or a failed consumer contract. No mutation, staging or drain was run.

The exact minimal live proposal is appended to the durable plan. Wait for the scheduled Pair build/restart and current wrapper capability, then for complete quiet queue evidence. The steward starts one graceful generation; each slot checkpoints, acks its ID and ends idle, with the steward last. The operator observes the board drained, then requests cancellation; resumed steward activity is expected at that point. Verify only this generation's queue owner is removed and ordinary dispatch is available. Record build identities, ID, board/queue transitions and the operator's attestation before any merge.

### 2026-10-20 — provenance repaired and combined live-build integration

First close attempt stopped before any review: suite attempt2677ae7a2d8d830e7b4f3b464f9afab6 lacked slot provenance, so actual-hours adoption refused. Git identifies its head1704163e8940 as train176's #717 prefix; retained landing events show pair:7 conducted train176 with #724/#717. Canonical TL receipt191ea2e0 says the hub restored slot pair:7 while keeping the original record aside (ariadne#492). No measurement record was edited by this slot. Rerun sdlc actual succeeded with7.07h preview and four recorded waits; close will adopt a fresh measurement rather than a typed value.

TL receipt3efec2c1 approves the written graceful-only live proposal and explicitly authorizes sdlc stage --build after SHIP. Stage stages exact HEAD and does not integrate main; ancestry proved this branch lacked landed #717. Integrated main7d4e0c3aba6b39fb12d5125ee967dcde9e5be4be before the first review so the reviewed/staged build includes the requested landed fixes. Three conflicts were limited to CLI dispatch, request validation and embedded cost documentation; resolution retains --drain plus the landed --interrupt/IsDriveOp support. Targeted drain/agent-drive/key/help tests passed couchcmd28.277s, couchmessage0.395s, wrapcmd4.631s. No regenerated workflow files changed. The combined code now requires a fresh recorded suite; earlier44d7bdcc evidence is historical only for this integration. Dispatch close on the combined head, allowing overlap with its queued suite per worker rules, then stage only after SHIP and report the actual build revisions. Live acceptance and merge remain held for the scheduled operator sitting.

### 2026-10-20 06:25 PDT — close round 1 corrections

The binary-owned close review returned REWORK (BR-1 acknowledgement-observation-ordering, Critical; BR-2 external-dependency-conformance, Important). Actual measurement succeeded after the hub repaired provenance. The queued combined-head suite was cancelled before execution because these corrections change code. No staging or live effect has occurred.

BR-1 preserves the approved checkpoint meaning: sample authoritative registry submission and identity-fenced wrapper input at ack time, carry that activity baseline into the pure acknowledgement event, and require a subsequent observation before sealing idle. Discard observations started before that ack using the existing ack revision fence. Enumerate registry activity, wrapper input, pre-ack delayed observation, post-ack activity, duplicate ack, missing evidence and identity/generation replacement; test each at the reducer/service seam (ARCH-ORDER/ARCH-PURPOSE). This corrects evidence ordering without a new public state or control mechanism.

BR-2 retains an opt-in real executable conformance runner through the production adapter/decoder, with newly created private queue/capacity/fleet roots only. Cover two-owner isolation, complete versus unknown observations, active suite/merge/landing retirement/interruption, worker authority and lost-response reconciliation. Document execution on queue-contract/dependency changes and before closing drain adapter changes (ARCH-MOCK); normal tests continue to use the stateful fake. Live acceptance and landing remain open, and staging remains conditional on SHIP.

### 2026-10-20 06:31 PDT — close findings corrected and verified

BR-1 regression failed before the correction with stored counter10 instead of ack-time11 in both reducer and service (registry, wrapper and combined activity cases). Ack now reads current wrapper input and registry submission under exact namespace/epoch/generation/instance/member checks and supplies the baseline to the pure reducer. It never seals idle at ack. A second deterministic service regression held the registry read before ack and failed because reconciliation captured its revision too late; the revision fence now precedes the entire snapshot and discards that old observation. Post-ack activity still invalidates provisional/sealed acks; duplicate ack does not silently renew the baseline. Missing/disconnected registry evidence, missing/failed wrapper, old protocol/epoch/instance, other generation, registry replacement and intent replacement all refuse acknowledgement. Existing identity, subsequent-idle and extension tests continue to pass.

Targeted all drain tests passed couchcore0.186s/couchcmd27.358s. Focused acknowledgement race checks passed couchcore1.314s/couchcmd2.204s. Retained opt-in real queue conformance passed from this checkout8.152s against Git-verified source/build702b01324fa543038178e23b831d3e3346c1204a. It invokes the production adapter/strict decoder with new private roots only, including two owners, replay, worker refusal, all active kinds, interrupted retirement, unknown/corrupt observations, legacy and verified waiters, and lost successful pause/resume responses. The runner refuses missing build provenance before mutation; an invalid executable negative control failed there. Its documented cadence is before enablement and on dependency/argv/schema/adapter/decoder/fake changes. No live queue state was mutated. Review sidecars and prevention lessons are retained. Full drain races and the recorded suite/next close review follow on the corrected head; live acceptance and landing remain held.

Full drain race verification passed couchcmd30.213s, couchmessage4.761s and wrapcmd5.275s; git diff --check clean.

### 2026-10-20 06:30 PDT — complete the observation-path sweep

Correct the preceding entry's prematurely written06:31 timestamp: those targeted checks completed before06:28 PDT. During the last class sweep, found missing-member observations still called the generic event door instead of the ack-revision/generation-fenced observation door. Extended the held-snapshot regression to absent-member evidence; missing=true failed on3e745f757. Routed that path through observeMember too. Interrupted only this slot's just-starting review and cancelled only its queued suite before mutation; no verdict was admitted and no test result is being reused for the new code. Both present and absent snapshot paths now share the same baseline ordering rule. Reverify and requeue on the final head before staging.

The complete drain race set passed after the missing-member correction: couchcore1.326s, couchcmd31.843s, couchmessage4.081s, wrapcmd8.731s. The only remaining production member-observation event construction is inside observeMember, so both positive and absent-member evidence share the generation/revision fence.

### 2026-10-20 06:37 PDT — approved fast-mode Spec revision (BR-3)

Canonical submitted TL receipt e881ba7d-158e-4a5a-812f-335f0a0fb030 explicitly approves this revision, superseding the original Spec promise that fast drain cancels merges. Fast cancels running suites and records them interrupted for a later explicit rerun. An admitted merge or landing finishes or retires at its own safe boundary; killing integration or a push is riskier than waiting. The landed #481 admission gate already stops a merge at its next suite boundary. Fast never authorizes killing a merge or landing process. The board remains draining with named active work until authoritative complete active=[] and all member checkpoint/idle conditions hold; timeout remains stuck without release.

| Queue operation | Fast pause | Completion evidence |
| --- | --- | --- |
| suite | Request interruption; preserve interrupted outcome; no restart on resume | Suite retires, no active row |
| merge | Preserve running work; finish/retire at its safe boundary, including the next suite admission boundary | Merge retires, no active row |
| landing | Preserve running work; finish/retire at its safe boundary | Landing retires, no active row |

The fake must match the real seam across this complete kind enumeration and owner replay/resume; the retained real conformance already asserts the suite-only interruption policy. Add fake and coordinator regressions that require live merge/landing rows to keep the board draining even after every agent has acked/settled and the suite retired. The public CLI, notice text, JSON schema, owner-scoped hold and approved graceful-only live check are unchanged. BR-1 and BR-2 were disposed addressed by round2, including reviewer-run mutation controls and real conformance; only BR-3 remains. Cancelled this slot's still-queued prior-head suite because the fake corrections change the tested head. No stage or live effect occurred.

### 2026-10-20 06:39 PDT — approved queue-kind policy verified

New fake/coordinator regressions failed before the correction: fake owner replay changed the original reason/mode, and fast merge state incorrectly became interrupting. Fake pause now preserves its first owner record, matching real idempotent pause, and targets only running suite-owning operations; several kind rows for one operation share its phase, while separate merge/landing owners remain running. Resume does not restart or newly interrupt work. The coordinator test acks/settles every agent, retires the suite, and still requires draining with named merge/landing rows; retiring merge alone continues waiting for landing; only complete empty activity yields drained.

New policy tests pass under race1.678s. Retained real adapter conformance reran PASS16.540s using fresh private roots and the same clean matching source/build702b01324fa543038178e23b831d3e3346c1204a. README and atlas explicitly describe suite interruption versus safe merge/landing retirement. This implements canonical TL-approved Spec revision e881ba7d; no queue argv/schema or production coordinator behavior changed in this correction. All drain tests and a new recorded suite/close review follow on the corrected pushed head; staging still requires SHIP.

All couchcmd drain tests passed28.164s after the fake policy correction; git diff --check clean.

### 2026-10-20 06:48 PDT — approved transition-authority correction (BR-4)

Canonical submitted TL receipt e9934f5f-75ed-411a-a949-86a7a8273a05 approves the correction. Every authoritative phase/error update is reducer-owned. Add a generation-fenced broker-release result: only queue-release-finished intent accepts it; failed/unknown release returns to the existing cancelling/error recovery, explicit cancel retries the existing owner-release sequence, and successful release clears the failure. Active intent, stale ID and unrelated outcomes cannot change it. The coordinator applies results only to its exact current unreleased intent and never marks a replacement intent released. Storage, dispatch-hold and member-cap failures use DrainFailed rather than direct Error writes. A local Go AST catalog rule (pair#722) rejects direct coordinator phase/error assignment and common aliases; constructors and whole-state reducer results remain allowed. This enumerates existing production failure paths without changing public states, CLI, persisted schema or the #481 seam. Verify release failure/retry/stale outcomes in reducer/service tests, and exercise the lint's positive/negative fixtures and actual repository scan before another close. The round3 FIX-THEN-SHIP verdict was not finalized because the ledger correctly retained this Important blocker; do not bypass it. Live/stage/landing holds remain unchanged.

### 2026-10-20 06:52 PDT — reducer-owned failures and structural guard verified

BR-4 pure regressions failed before correction: a broker-release failure after queue release was ignored in finished state, and a storage failure there was discarded. DrainBrokerReleaseResult now owns success/failure and rejects stale ID or active-phase outcomes; failure returns to cancelling, records the error and invalidates the old queue snapshot so it cannot immediately re-complete. Retry traverses the existing queue/revoke observation sequence, and known successful broker release clears its error. DrainFailed owns storage/hold/member-cap errors without accidentally triggering unrelated cancellation effects. Storage failure after queue release remains visibly stuck, rather than disappearing behind the finished projection. The coordinator revalidates current unreleased generation/phase before and after the broker call; late or duplicate release paths cannot mutate a replacement intent. Service regression uses the actual broker refusal, clears that exact conflicting hold, retries through the production reducer, and proves final cancellation and stale-outcome rejection.

Focused core/service release races PASS1.308s/1.418s. Complete drain races PASS couchcmd30.425s/couchmessage4.769s/wrapcmd5.261s. The new pair#722 drain-state-reducer AST catalog guard covers direct writes, renamed/new methods, constructor locals, nested closures, multiassign and receiver/record/state/field aliases; whole-state initialization/reducer output and unrelated fixture structs remain allowed. No authoritative coordinator phase/error assignments remain.

The catalog caught the retained conformance helper compiling once per opt-in test. Reused internal/testbinary.Cache, compiling once per test process under a one-minute context and retaining the shared executable until TestMain joins cleanup; each fixture keeps fresh independent private state roots. Real conformance reran PASS5.667s on Git/build-verified clean dc76211e27578aca7aa62d434db9ca2bd43eed5b. Prior private probe diagnostics are untouched. Record the final catalog/all-core checks, then push and run the final-head recorded suite/close review; no stage or live effect occurred.

Final AST matcher and actual tracked-source catalog scan PASS1.839s.

### 2026-10-20 07:12 PDT — BR-5 answering pinned-head measurements

Round4 disposed BR-4 addressed and found no new behavioral defect, but returned REWORK with Important/unverifiable BR-5: ordinary six-package go test reported physical-write cap failures (core139386880/134217728, tty87937024/67108864). The ledger round cap demoted this finding, but no cap/guard is being waived and its REWORK verdict still holds close. Keep the recorded suite queued on31899be8; code has not changed.

Re-ran both entire affected packages, uncached, on exact Git head31899be8ee9f96a6869ded680014c974b38fb64f: go test ./cmd/internal/couchcore ./cmd/internal/couchtty -count=1 -json. PASS exit0; couchcore184.552s with96768000 bytes against unchanged134217728 cap, couchtty55.873s with39788544 against unchanged67108864 cap, repeats1. Metadata records TMPDIR and GOTMPDIR=/Volumes/TestScratch/tmp/pair-slot5. The retained Go JSON output is /Volumes/TestScratch/tmp/pair-slot5/pair722-write-budget.ougil02v/head-packages.jsonl, SHA256e69040f5cc38dd750769b510d24b3de942dbd2ec9b70409531c34c9af82af45f, with adjacent metadata.json containing the Git-derived head, environment and exact argv. These are full package runs, not drain-only filters; native enforcement remained enabled.

Read-only source/transcript audit found no supported attribution to the drain changes: all18 new core tests are pure and the four tty cases use one fake-console fixture without filesystem calls. Self-executable fixtures use symlinks; shared Pair copies happen only in a child across filesystems, while the budget samples current-process rusage. Therefore the earlier test-binary-size-copy conjecture is rejected. Reviewer checkout was itself on TestScratch, and its commands neither sampled nor explicitly set TMPDIR/GOTMPDIR; a default-SSD explanation is unproven. Its Git-backed baseline passed but Go suppressed the successful native counters. Package selection/parallel workload also differed (reviewer six packages; answering measurement the two affected complete packages). These measurements resolve availability of green pinned-head evidence, without claiming an exact cause for the differing physical-write totals. Submit them as a measured unverifiable-finding dispute for explicit reviewer disposition; the final recorded suite and live acceptance remain required.

### 2026-10-20 07:23 PDT — SHIP and staged live-check checkpoint

Round5 returned SHIP with zero remaining findings. Reviewer explicitly withdrew BR-5 after verifying the retained JSON hash/metadata and independently passing the complete pinned-head packages (core96641024/134217728 bytes). Original differing totals remain unexplained, but green validation is available with no budget change. All behavioral/structural findings BR-1..BR-4 are addressed. Close recorded8.64 measured hours and evidence commit63645ddd1014. Final publication initially refused because the tracker card was still blocked; the named recovery also refused that precondition. Used the supported set-status working verb, then issue recovery reconcile --issue722; completion finished, codecomplete/card binding and mirror were published, branch pushed. No mirror/status fields were edited manually.

Canonical submitted TL receipt2e7afaf3 owns disposition of an obsolete ignored pair:0 bin/pair-hoprtt (source command absent since its old build): the TL preserved it under the local obsolete-bin directory. The first stage had moved the target and built current binaries but refused that stale leftover; after TL disposition, repeated sdlc stage --build exit0. Selected target Git HEAD=f3c597f8dab1955b8b81c304e0cf2e99b935071b; bin/couch, bin/pair, bin/pair-launch-helper and bin/pair-slug all printed that exact vcs.revision and vcs.modified=false. Both target and worker had zero tracked/untracked dirt. No regenerated workflow files changed. Target remains staged on pair:0; worker retains this issue branch. Outcome sent to hub receipt02650d38-ccbe-4de9-9276-8a3cddf3e61e. This subsequent checkpoint changes documentation only; the selected staged build remains f3c597f8.

The final recorded suite remains running on31899be8ee9f96a6869ded680014c974b38fb64f (identical code to staged build; later commits are review/tracker evidence), after1508s in the machine queue. Its retained run log is /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/debe572ea70f66e5cb05c775859debea/run.log; detached suite checkout /Volumes/TestScratch/local-checks/sdlc-local-1218534078. Ordinary integration steps observed so far are passing; no suite pass is claimed yet. Do not cancel or duplicate it after a restart; inspect its saved result/log through local-check tooling. Live acceptance and landing checkboxes remain open. Await the TL/operator's scheduled Couch restart and graceful-only board acceptance after complete quiet queue evidence; every slot checkpoints/acks, steward last, then cancel. No real drain, queue pause, slot relaunch or merge has been performed by this worker.

### 2026-10-20 07:27 PDT — final recorded suite PASS; awaiting scheduled live acceptance

The required recorded sdlc-local/suite completed exit0 in8m29s on31899be8ee9f96a6869ded680014c974b38fb64f, first pass without a flake rerun, after1508s queue wait. Saved log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/31899be8ee9f96a6869ded680014c974b38fb64f/suite.log has SHA256b2a403d3103941e6d15ca68e0ff978e94d405e63b1b8faa859e0eb7d4dac0e7a. Full package results include couchcore124.094s, couchmessage27.175s, couchtty44.854s, couchcmd65.114s and wrapcmd53.463s; all preceding shell/Lua/Python integration checks passed. This standard declared suite keeps native physical-write enforcement intact and confirms the same code as selected staged f3c597f8; intervening branch commits contain only review/tracker/checkpoint records.

Filed independent unclaimed ariadne#497 for the blocked-card close preflight/premature-success diagnostic gap, with observed failure and supported recovery; hub notification receipt9618e814 was submitted. The obsolete stage binary was already preserved by the TL, and the repeated stage succeeded; no workaround or gate bypass was used. Pair:0 remains clean at f3c597f8dab1955b8b81c304e0cf2e99b935071b. Preserve that stage for the operator sitting. No further test rerun or main integration is needed absent a code change or failed gate. Remain claimed/codecomplete, with live and landing Plan items open, until the TL schedules/verifies the operator-present graceful drain/cancel check. Merge remains held for that evidence; #622 remains parked pending its separate SDLC branch/close gap.


### 2026-10-20 11:07 PDT — failed live acceptance; recovery-first re-plan proposal

Authenticated TL dispatch31d0d041 requires clearing the stage and re-planning before code. Read both failure sections of the operator's retained ops reboot record. Acks refused, messages were held before checkpoint confirmation, cancellation could not settle across reboot, quit/detach were blocked and repeated chip states/reasons overwhelmed the bar. Preserve the operator's drain.json.wedged-1020 unchanged; no failed live check is relabeled successful. The saved record's zero NoticeStarted flags do not explain the TL-attested ariadne:4 checkpoint request: exact notice origin/generation and failed ack predicate remain unproven until public-path reproduction or receipt evidence.

Cleared pair:0 with sdlc stage --clear: clean main at Git-verified e712d8335f3e77a0694d6bd8869f5e3bf69b033f; couch, pair, pair-launch-helper and pair-slug all report that revision and modified=false. Stage report008bd648 submitted to the hub. Another authorized item, #622, has re-entered landing in the original checkout. This documentation-only re-plan uses the same canonical #722 branch in an isolated temporary worktree; it does not move ownership or stack issue code.

The matching durable-plan Revision proposes queue/member preflight before effects; checkpoint collection with messages open; a generation-scoped local dispatch lease only at the confirmed final barrier; cancellation/local escapes independent of old wrappers; diagnostic-only recovered intent; and one concise bar indicator. Keep exact identities/epoch fencing, pure reducer authority, owner-isolated queue seam and bounded persistence. Cancellation cannot undo a handed notice or Esc; uncertain effects remain warnings/recovery evidence, not grounds to suppress ordinary coordination forever.

The new acceptance oracle is a disposable installed-Couch fleet with real Pair wrappers, admitted callers, public CLI ack, private state/queue roots, controlled checkpoint markers and rendered PTY evidence. It drives normal, fast, stuck queue, cancel and restart, plus quit/detach and cross-fixture isolation, before any further live run. Read-only exploration verified the early HoldDrain, all-wrapper revoke dependency, restored recovery hold, broad lifecycle guard and per-chip renderer causes. No product code, queue contract or real-fleet drain action changed in this re-plan. Await TL approval, then supported reopen/change-code, new tests/fixes, recorded suite and close; only then schedule a new watched live check before merge.


### 2026-10-20 11:09 PDT — re-plan approved; opt-in matrix required

Canonical submitted TL receipt71ddbf62 approves Revision11:07, supported reopen and renewed change-code. Keep pure/endpoint regressions in the ordinary suite; the disposable matrix becomes the named opt-in recorded local check drain-acceptance (make test-drain-acceptance), required in close evidence and immediately before staging. Seven90s installed-fleet cases must not burden every ordinary suite. The ariadne:4 delivered-request/zero-started-flags discrepancy is an explicit acceptance item: reproduce or explicitly report non-reproduction and retained uncertainty in the close, never close around it. Live acceptance remains watched/scheduled through TL.

The documentation-only auxiliary worktree was refused by sdlc issue set-status working --issue722 because its parent pair-slot5-plan is not a valid numbered environment. No card transition occurred. Do not evade workspace identity or bypass the gate. Resume #722's setter/change-code from its canonical pair:5 checkout once the concurrently authorized #622 landing releases that checkout. No product code has changed.

### 2026-10-20 11:23 PDT — recording dependency and interim evidence

Canonical submitted TL receipt5c8b8b29 directs a separate operator-verified local-gates declaration issue and permits manual exact-head matrix evidence in this Log until that declaration lands. Filed unclaimed #734, Declare opt-in drain acceptance local check, published card/details on origin. The frontmatter dependency is specifically for named-check recording, not a prohibition on building and manually qualifying the #722 harness beforehand. #734 owns only the optional protected declaration; #722 owns the make target, actual installed-fleet tests and close/pre-stage evidence requirement. Retain exact argv/head/result/duration/log/hash manually until the recorder can discover the main declaration. No .sdlc file was edited and no operator label was applied. The same TL message confirms reopening from canonical pair:5 after #622 releases it.


### 2026-10-20 11:36 PDT — HOLD pending operator's simpler contour

Canonical submitted TL receipt379c72ba supersedes the earlier permission to reopen/change-code: HOLD #722; do not reopen, enter change-code or implement the approved Revision. The operator is shaping a steward-driven in-memory per-slot flag and ack model with messages remaining available and no persisted drain state. This is a proposal, not an implementation contract; TL will return with what survives and what is dropped. Preserve current code and evidence unchanged meanwhile. The compact badge and ack diagnostics may survive, but are not independently dispatched. No product code changed after the failed live check, and the attempted auxiliary-worktree reopen never changed the card. Pair:0's stage was cleared/rebuilt earlier. #734's recording declaration was filed unclaimed before this hold; its relevance awaits the revised contour. Continue only #622's already-authorized landing in canonical pair:5.


### 2026-10-20 12:18 PDT — operator-reviewed in-memory contract; plan awaits operator review

Authenticated TL4160a300 supersedes the earlier hold with a new scoped design task, not permission to code. Read the ten points and settled answers in ops/teams/devinfra/tl-status-xian-xu.side/2026-10-20-17-drain-redesign.md. Keep #722 and its branch; the fresh dispatch overrides the side doc's earlier replacement-issue suggestion. Couch owns only transient slot flags, notices, status and a compact badge. Workers keep receiving messages and answer in ordinary prose; steward owns queue pause/resume and readiness judgment. No persisted drain, formal acknowledgement, barrier, hold or automatic wrapper interruption survives. #739/#740 own startup/global park separately.

#734 was still open/unattributed. The supported abandon command required a claim; claimed narrowly without starting work, then sdlc abandon --issue734 --as wontfix recorded the supersession reason and archived details on main. No gate configuration or label changed. Remove its recording dependency. Retain a bounded installed-Couch qualification inside #722, recording evidence in the issue; no named protected declaration is needed for the removed queue/ack matrix.

The current Spec/Done when/Plan above now describe this contract; previous active text is retained below as superseded provenance. The cumulative estimate and former SHIP remain historical until renewed plan-quality/estimate/code/close gates. No product code changed. Re-scope plan goes through TL to the operator before code; #622's active train has priority.

#### Previous Spec — superseded

**Governing re-plan proposal:** the 2026-10-20 failed-live-acceptance Revision below and the matching durable-plan Revision supersede the early message hold, all-wrapper cancellation prerequisite and recovery hold. They await TL approval before code. The original contract below is retained as provenance, with prior approved fast-mode corrections still governing suite-only interruption.

`couch --drain [--reason TEXT] [--fast]` and `couch --drain --cancel` (steward hub or operator only):
- **Graceful (default):** every live slot gets one system notice ("checkpoint and stop: finish the current atomic
  step, commit, push, write where you stopped in the issue Log, start nothing new, reply drained"). Couch stops
  delivering dispatches meanwhile.
- **`--fast`:** for work that can be restarted later. Couch interrupts running agent turns (Esc), and the notice says
  "stop now: write where you stopped, don't finish the step". Suites and merges are cancelled through sdlc's pause
  (ariadne counterpart) and recorded as interrupted, and rerun on the same head after the restart. Nothing durable is
  lost: the working tree and the transcript are on disk; only in-flight turn context goes.
- **Drain board** (`couch --drain --status`, plus the tab bar): per slot **draining / drained / stuck (why)**. Drained
  means the agent is idle at its prompt, no running suite, merge or landing is held, and the slot's last report says
  drained. A timeout (`--timeout`) ends in a list of the stuck slots, never "go anyway".
- After the restart, "resume the fleet" (ariadne#456) brings back the slots that had work.

The [durable implementation plan](../plans/000722-fleet-drain-plan.md) defines the draft transition model, state/recovery table, exact v1 notice text and shared SDLC argv/JSON/fake contract. TL-approved refinements below supersede the original operator-entry/free-text-report wording; remaining design proposals await TL approval before code.

#### Previous Done when — superseded

- The installed-binary disposable-Couch harness (one steward, two throwaway workers, private state and queue roots) passes graceful, fast, stuck-queue refusal, cancellation and reboot-during-drain through actual registration, delivery, CLI acknowledgements and wrapper endpoints. Component fakes alone do not satisfy acceptance.
- Starting against unknown/incomplete/stuck or already-paused queue evidence refuses before intent, pause, notice, interrupt or delivery suppression. Ordinary coordination remains available until the exact current checkpoints and final quiet barrier are confirmed.
- Cancel releases local control/delivery authority without old-wrapper observation; only its own SDLC pause is reconciled. Unknown release remains visible and retryable while ordinary messages, quit and detach remain usable, including after supervisor/fleet restart.
- The tab bar has one bounded global drain indicator, selectable ordinary chips and no repeated reason or per-chip drain suffix. Full reasons remain in status.
- A renewed full-flow review, recorded suite and successful TL/operator-watched live acceptance follow the disposable matrix before merge. The failed morning run is retained as failure evidence; no former SHIP or interruption result substitutes for these gates.

- A fake fleet of 3 slots (one idle, one mid-turn, one with a running suite) reaches all-drained under graceful and
  under `--fast`, with the board showing each transition (tests on the stateful fake).
- `--cancel` restores dispatch delivery and lifts sdlc's pause.
- A slot that never reports shows as stuck, with its reason, at the timeout.
- A matching per-drain ack plus fresh exact-incarnation idle and authoritative SDLC quiescence is required for drained; quiet without ack is stuck: no ack. Old incarnations/acks, incomplete queue status and late effects cannot manufacture completion.
- Steward-only control, dispatch holding/release, owner-isolated queue cancellation, fast interruption identity checks and restart/uncertain-effect recovery pass deterministic sequence tests. The real ariadne#481 queue adapter conforms in an isolated portable state root before production enablement/close.

#### Previous Plan — superseded

- [x] Approve/checkpoint the design and pass full-flow change-code before implementation.
- [x] Implement the pure model/board/notices and owner-scoped queue/store seams with stateful fakes and sequence tests.
- [x] Wire steward CLI, authenticated ack, dispatch gate, fast wrapper interrupt and tab/status board.
- [ ] Verify real ariadne#481 conformance, isolated wrapper integration, races, docs and required recorded suite; stage/live board check with hub authorization.
- [ ] Close through the mandatory SDLC review, land through the ordinary gates and report the outcome.

### 2026-10-22 — implementation checklist ends at verification; live remains a pre-landing gate

The prior final Plan row read: “Run required verification and mandatory close review, then TL-scheduled watched live acceptance before ordinary landing.” Its own close/live receipts cannot be checked before the close that prepares staging. Clarify the implementation checkbox to end at the recorded passing full suite, while explicitly keeping mandatory close, authorized staging, ariadne#506 landing and watched live acceptance as subsequent required gates before landing. No acceptance criterion is removed, no milestone split, and no historical SHIP/stage instruction becomes fresh authorization. The operator-approved transient product/state contract is unchanged.

### 2026-10-22 02:26 PDT — distinguish running background work from queued work

Authenticated submitted TL e220cac5 approves this exact graceful opening, coordinated with ariadne#506: “Drain: finish your current atomic step and wait for background work that is already running to finish. Leave queued, not-started work queued and note it in the issue Log as outstanding.” It replaces the ambiguous “current background work” opening only; all remaining graceful guidance and the entire fast notice stay unchanged. Queued work is not admitted work and may remain held by the steward's external queue pause; do not wait hours for it or claim it complete. Record it in the checkpoint Log as outstanding. No queue call, flag/state change, admission policy or new mechanism is added (ARCH-PURPOSE/DRY).

Update the pure notice expected-body regression, the independent shared-native expected body, README and atlas. Demonstrate the corrected expected body fails against the old notice and passes with the correction. Refresh final-source native and required suite evidence, then re-close before any staging. Earlier SHIP/native/suite evidence remains historical and does not qualify this wording change. The watched-run hold, ariadne#506 prerequisite and unresolved historical ariadne:4 discrepancy remain unchanged.

### 2026-10-22 02:37 PDT — retain validated native frame through its use

Authenticated TL134c9b95 approves the consumer-only correction: poll, validate, log and compute chip coordinates from the same retained immutable screen/row, rather than re-reading after the badge check. Sweep other success diagnostics for the same check/use split, including selected-worker and quit-confirmation frames. Add a controlled changing-screen regression that fails if a second read replaces the validated snapshot. No product behavior, timeout, native topology or polling budget changes (ARCH-ORDER). Keep the #773-class startup failure and separate consumer failure as failed evidence, then refresh native and recorded suite evidence and re-close. Live hold and ariadne#506 prerequisite stand.

### 2026-10-22 02:44 PDT — TL acceptance sequencing after shared-native failures

Direct submitted TL ef5588c8 supersedes the demand for another complete fresh native pass before re-close: stop matrix reruns after the shared startup/registration failures. Retain and attribute all failures to the related #773 startup class or the independently regressed/fixed consumer snapshot race; do not claim their roots resolved. Use the retained full Claude/Codex pass on68c581ac, exact corrected-body RED/GREEN, current reached native feature proofs and fresh current-head recorded suite for renewed review. No current-source complete native pass is claimed. The watched operator-present run with real slots remains required before landing and must qualify the actual current notices, message flow, badge/status and undrain. Bring any reviewer demand for a fresh native pass back to TL; no manual waiver or blind retry. No product, timeout, topology or queue contract changes. Existing live hold and ariadne#506 prerequisite remain.

### 2026-10-22 11:30 PDT — registered asynchronous notice operations

Submitted TL5f7e03f6 retains approved fleet-wide selection but supersedes synchronous fanout: drain and lift must register parent IDs and child receipts, return when flags are applied, and expose per-slot delivered/pending/failed state through status plus operation-status. One slow target must not consume the exchange. The appended durable plan Revision specifies the state table, bounded ownership and private slow-target RED/GREEN. Watched acceptance has failed; no landing until corrected-source verification/re-close and a fresh operator-watched rerun.

### 2026-10-22 12:45 PDT — readiness before publishing the service

Initialize the newly registered drain operation dependencies before exposing request handlers, with owned executor cleanup on every later constructor failure. This enforces the approved publication/lifetime ordering; public CLI, notice vectors, cross-hub contour and deadlines remain unchanged. Retain the private race RED/GREEN and normal catalog guard.
