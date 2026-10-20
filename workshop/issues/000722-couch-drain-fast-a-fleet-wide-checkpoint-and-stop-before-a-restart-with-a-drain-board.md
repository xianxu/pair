---
id: 000722
status: codecomplete
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-20
estimate_hours: 9.6185
collaboration-mode: collaboration
card_mirror: '49bc382cbe1824d6362ed2770bbee192da1e189e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T23:16:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 8.64
---

# couch --drain [--fast]: a fleet-wide checkpoint-and-stop before a restart, with a drain board

## Problem

There is no safe, on-demand moment to restart the machine or Couch. Work never ends, so "safe to restart" has to be a
state the fleet can reach on request. Today a restart is a crash (10-19 15:55) or the TL pinging slots one by one,
and nobody can read off whether it's safe. Operator item 15 (10-19), discussion:
ops `teams/devinfra/tl-status-xian-xu.side/2026-10-19-15-fleet-drain.md`.

## Spec

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

## Done when

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

## Plan

- [x] Approve/checkpoint the design and pass full-flow change-code before implementation.
- [x] Implement the pure model/board/notices and owner-scoped queue/store seams with stateful fakes and sequence tests.
- [x] Wire steward CLI, authenticated ack, dispatch gate, fast wrapper interrupt and tab/status board.
- [ ] Verify real ariadne#481 conformance, isolated wrapper integration, races, docs and required recorded suite; stage/live board check with hub authorization.
- [ ] Close through the mandatory SDLC review, land through the ordinary gates and report the outcome.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Calibration is stale per estimate-source, so this is provisional focused ship-time, not a calendar promise or a prediction of the #481 wait.

Decomposition follows the approved plan: initial spec/coordination uses issue-spec design1.0×0.5 and impl0.3×0.4. The pure drain model and supervisor coordinator are two greenfield Go single concerns (design1.0/1.5×0.2, impl0.8×0.4 each). Queue adapter, broker control authority and wrapper interrupt are three API integrations (design1.5/2.0/1.5×0.2, impl1.2/1.5/1.2×0.4). Portable store extends the existing namespace/atomic-write patterns (design0.3×0.2, impl0.5×0.4). Board UI extends current console rendering (design1.0×0.2, impl0.8×0.4). Docs use design0.15×0.2, impl0.2×0.4; one close review uses design0.2×0.2, impl0.5×0.4; real queue/endpoint conformance uses one discovery budget impl0.6×0.4. Existing Go JSON/context/atomic storage and Couch broker/registry/console seams avoid a novel stack; no library replaces this project-specific generation/ownership protocol. Familiarity1.0, thorough-plan design buffer15%, no parallel execution discount. Design2.33×1.15 + impl3.36 =6.0395h.

### Active cumulative estimate revision — 2026-10-20

Renewed plan-quality round4 passed before this derivation. The original prose and machine block below are retained as provenance; this first block supersedes their total for the approved epoch mechanism. Same v3.1/stale calibration, familiarity1.0 and thorough-plan buffer15%. Add: typed fence/protocol integration design1.5×0.2, impl1.5×0.4; pure generation ownership model design1.5×0.2, impl0.8×0.4; epoch/schema store extension design0.3×0.2, impl0.5×0.4; coordinator propagation/recovery integration design1.0×0.2, impl1.0×0.4; counted-cycle/transport/race/live-conformance execution design1.0×0.2, impl1.5×0.4; second external endpoint discovery impl0.6×0.4. The existing library/IO seams remain reusable; no library supplies this protocol-specific generation authority. These incremental primitives explicitly cover the re-plan rather than hiding it in the original coordinator allowance.

Design3.39×1.15 + impl5.72 =9.6185h cumulative idle-removed ship wall-clock, including agent execution and active verification/review, excluding external #481/operator waits and registered suite-queue waits. The added execution integration allowance includes implementing/adapting the stateful fake and 1000/2000-cycle checks, renewed races/recorded suite and hands-on isolated/live acceptance; discovery is separate. Prior completed transport checks stay evidence, but changed code requires fresh execution. This is a provisional forecast, not measured actual time or a remaining-hours promise.

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


- 2026-10-20: closed — All behavioral findings BR-1/2/3/4 addressed in prior reviews with negative controls, races, real conformance and AST enforcement. BR-5 unverifiable dispute supplies full uncached pinned31899be8 package run: go test ./cmd/internal/couchcore ./cmd/internal/couchtty -count=1 -json PASS exit0, native caps still enforced core96768000/134217728 and tty39788544/67108864, repeats1. JSON log sha256e69040f5cc38dd750769b510d24b3de942dbd2ec9b70409531c34c9af82af45f and exact head/env/argv metadata retained in /Volumes/TestScratch/tmp/pair-slot5/pair722-write-budget.ougil02v. Reviewer temp env was unrecorded, so exact discrepancy cause remains unproven; no budget raise or test filter used. Pending dispute requires explicit reviewer disposition. Code unchanged since31899be8, recorded required suite retains queue position there; current09140407b75d866a912c26cb32e435de2451e4ee adds only measured evidence and review artifacts. TL review-first receipts authorize only --no-plan-check for open live/landing items: stage only after SHIP; operator-present live check and final recorded suite still precede merge.; review verdict: SHIP
### 2026-10-19

Claimed by pair:5 under canonical TL dispatch c1f508a1. Read the operator's fleet-drain discussion and ariadne#481's existing Spec. start-plan delivered ARCH principles and created the independent issue branch from main. Its weave refresh regenerated the two CI workflows; reverted only those generated files per the standing #699 instruction, then start-plan succeeded. No implementation code, real pause, interrupt, restart or live drain was performed.

Read-only code exploration confirmed HubOwnership.Main as steward authority, exact liveCaller/Registry session evidence, Broker.SendSystem, console event publication and existing stateful fake patterns. Park/continuation execution does not provide an agent-authored checkpoint and is not reused as the drain's stop mechanism. Drafted the durable plan with all nine ARCH lenses; no estimate is derived before plan-quality.

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
