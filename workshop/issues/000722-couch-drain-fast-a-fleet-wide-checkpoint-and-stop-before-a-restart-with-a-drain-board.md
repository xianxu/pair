---
id: 000722
status: blocked
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-20
estimate_hours: 9.6185
collaboration-mode: collaboration
card_mirror: 'c08e228457c45dc6801d7d5d11c64e03093ed2e1' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T23:16:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# couch --drain [--fast]: a fleet-wide checkpoint-and-stop before a restart, with a drain board

## Problem

There is no safe, on-demand moment to restart the machine or Couch. Work never ends, so "safe to restart" has to be a
state the fleet can reach on request. Today a restart is a crash (10-19 15:55) or the TL pinging slots one by one,
and nobody can read off whether it's safe. Operator item 15 (10-19), discussion:
ops `teams/devinfra/tl-status-xian-xu.side/2026-10-19-15-fleet-drain.md`.

## Spec

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
