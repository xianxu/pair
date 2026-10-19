---
id: 000683
status: codecomplete
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-19
estimate_hours: 3.851
severity: high
collaboration-mode: collaboration
card_mirror: 'cee747adafbd06a2c09cd52bfac2ecdf4a6b3821' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T02:08:51-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 3.20
---

# Persist every lifecycle failure reason

## Problem

Operator UI relaunches of ariadne:0 and pair:0 failed on 2026-10-18 around 17:49–17:51 PDT. The reported Couch build included #651, #676 and #670. Each cleanup helper published a matching success completion and preserved scrollback, but Couch left the park awaiting_completion and did not publish VerifiedPark. Subsequent resume refused resume-parking. The original UI failure message was not retained with an inspectable operation, and the hub reported no new launch-diagnostics/failures.json entry; the initiating cause could not be established from the surviving evidence.

The operator later recovered through the menu; both outcomes were reported as recovered (Fresh). Read-only inspection found new live tags, while the old records had been archived with their parks still unresolved. Whether the operator selected Fresh or Couch selected it is not established; do not infer an automatic Fresh transition from the label alone.

## Spec

Reconcile a matching successful cleanup into a verified park after exact owner-death proof; keep unknown/replaced ownership refusal and never manufacture success. Make interrupted post-cleanup finalization recoverable on the exact retained transaction. Persist every lifecycle failure's message with its operation, including UI-triggered relaunch/park/recovery failures before a new launch, so later diagnosis does not depend on a transient banner or a launch-only diagnostic entry. Preserve bounded diagnostics, identity and effect-phase evidence without recording transcript bodies or secrets.

Distinguish verified park recovery, retained unresolved evidence and explicit Fresh in the UI and durable operation outcome. Investigate why the menu path archived these unresolved records; do not silently equate archival or a new Fresh slot with successful park finalization. Preserve prior conversations and capture authority.

## Done when

- [ ] Stateful coverage reproduces cleanup success followed by interrupted/failed Couch finalization; exact recovery finalizes the retained transaction without repasting, restarting or silently choosing Fresh.
- [ ] Lifecycle errors from both menu and CLI remain durably inspectable with message, exact target/transaction, phase and outcome, including failures with no new launch attempt.
- [ ] Recovery and Fresh outcomes explicitly distinguish finalized park from archived unresolved evidence; unknown/replaced owner guards remain intact.
- [ ] The retained incident is diagnosed as far as evidence permits; missing original error and uncertain menu choice remain explicit, not reconstructed as facts.
- [ ] Required verification and review pass before normal landing.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only; calibration is marked stale by estimate-source, so the rates are provisional. Scope includes design, tests, private conformance, docs and one close review; machine queue idle time is excluded by actuals.

The issue/spec item accounts for design already performed. Implementation decomposes into codec, history/store, ordered attempt coordinator, dispatcher, queue integration, receipt mapping, cached projection and row/show rendering. Existing JSON/flock/storage/console seams cover the library check; the ordered attempt semantics are local policy, not a missing external library. Remaining design uses the thorough-spec 0.2 discount; familiar Go/Couch work uses familiarity 1.0. Every impl value below is already 40% of its v2 table value, applied once. One review boundary and one atlas update are included.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.70 impl=0.08
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.16
item: greenfield-go-module design=0.20 impl=0.32
item: skill-or-dispatcher design=0.12 impl=0.20
item: cross-cutting-refactor design=0.12 impl=0.20
item: smaller-go-module design=0.02 impl=0.12
item: smaller-go-module design=0.06 impl=0.20
item: tui-screen design=0.20 impl=0.32
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.02 impl=0.20
design-buffer: 0.15
total: 3.851
```

## Plan

Inspect retained lifecycle/archived records and the operation-diagnostic seam first. Design failure retention and recovery ordering with explicit partial outcomes before implementation. #681 separately owns the confirmed two-second CLI admission timeout; do not duplicate that fix here.

Approved implementation plan: [Lifecycle failure persistence](../plans/000683-lifecycle-failure-persistence-plan.md). The scope disposition and replacement close criteria below were approved in c35c61ae; implementation is gated by change-code.

- [x] Implement and verify the preserving lifecycle-failure codec and bounded pure history (durable plan Task 1).
- [x] Integrate one terminal recorder with queued and synchronous operations and retain receipt nonces (Task 2).
- [ ] Share the cached row/show projection; complete private-service proof, recorded suite, close review and landing (Task 3). Disposable-slot live display acceptance follows the next operator restart.

## Log


- 2026-10-19: closed — BR-1–BR-5 disposed by prior review. BR-6 fixed with terminal publication/readiness/callbacks on existing joined workers after adoption, console lifetime cancellation, no storage for pre-enqueue refusals. Actual menu/reattach/navigation/remote input-and-Stop tests pass; core owner-context regression proved red then green. Composed real private-service stalled-store proof passes and rejects removed constructor lifetime wiring (DeadlineExceeded vs Canceled), source restored then verified. Broad focused core/cmd/tty/message pass30.454s/34.309s/0.516s/1.503s; focused race pass4.903s/2.111s/1.597s; full couchtty pass41.546s; full source/isolation/declaration guards pass6.721s/7.500s/2.497s. Required suite queued on final d8e7494ec code; final-head suite pass must precede landing. --no-plan-check only acknowledges combined suite/close/landing checklist still ongoing, implementation complete; --no-project because hub owns project. Live admitted-failure row/show check owed after operator restart, no :0 staging.; review verdict: SHIP
2026-10-18 — Read-only incident evidence gathered before any worker repair or relaunch. Hub requested this unassigned filing in verified delivery 89941932-b309-48df-b397-e5f5cbadfb58. No production metadata was repaired by the worker.

Before menu recovery, global threadstore records were pair revision 685 and ariadne revision 546, both awaiting_completion, attempt 1 open, no VerifiedPark. The recorded helper PIDs were absent; Zellij sessions 1-282 and 1-284 were absent. Matching success completions and all three parked capture bundle files existed. No persistent lifecycle/store lock holder was found during inspection.

After menu recovery, read at approximately 18:17–18:18 PDT:

- pair:0 old scope e108517d46ab4575, tag 1-pair-2, archived revision **686**, park nonce park-678704160c5bde5f, attempt 1 still awaiting_completion, no VerifiedPark. Original owner PID 56488, kernel identity 1792370871.658511. Success completion at 17:51:15.714059 PDT; capture token 20261018T175115; raw 377597 bytes, events 868 bytes, capture metadata 785 bytes. Prior Codex native ID **01a12720-0b3f-7ee0-a5da-1474622794f5**, retained transcript 3187553 bytes. New active tag 1-pair-69, session 1-310, revision 4, one live incarnation and no open park/start/continuation.
- ariadne:0 old scope b325dbb782174007, tag couch-539ef32aa4744e03, archived revision **547**, park nonce park-98bb4395906efc9d, attempt 1 still awaiting_completion, no VerifiedPark. Original owner PID 56753, kernel identity 1792370872.494384. Success completion at 17:49:13.701393 PDT; capture token 20261018T174913; raw 154319 bytes, events 4625 bytes, capture metadata 828 bytes. Prior Claude native ID **bee2be25-757b-4027-8573-2bdcaf857203**, retained transcript 1337921 bytes. New active tag 1-ariadne-67, session 1-308, revision 4, one live incarnation and no open park/start/continuation.

The old records reside under the global threadstore archive, not records; their request/completion JSON and parked bundles remain in their repository scope directories. No pending threadstore journal was found. New live slots are settled; archived old park bookkeeping is explicitly unresolved. Native transcript existence was checked by path and size only; contents were not printed or attached.

Code audit: runActiveAttempt returns immediately if the terminal-delete trigger reports an error after an effect; awaitCompletionAndChildDeath also returns immediately on completion-observation errors; finalization/store errors can leave the same awaiting record. Without the original UI message those paths cannot be distinguished. Target client exit does not itself cancel the UI operation context, which derives from console lifetime. This audit does not establish an inherent deadlock or a particular initiating failure.

Confirmed separate #681: recover-park prepared full fleet inventory before receipt allocation inside a 2-second CLI admission exchange. Hub verified both timed-out requests had unknown receipts. Adjacent normal resume preparations took 3.055s and 5.233s. This explains the failed CLI recovery attempts, not the original UI relaunch failure.

2026-10-18 — Additional evidence from hub delivery a4fd4c1b-e50a-437b-96d9-28addcead9a2, verified read-only: launch-diagnostics/failures.json contains a later pair:0 launch failure at 18:15:07.913087 PDT, scope e108517d46ab4575, intermediate tag 1-pair-68, nonce start-21db4bb41655e148, stage registration, helper started, before/after exit 1. launch_error is "context canceled / recovery attempt registration is not confirmed". Decoded retained output says the exact Couch session name 1-309 was not accepted because the Zellij name probe was killed and its context deadline was exceeded. This is a later attempted launch, plausibly the operator's Resume from disk before Fresh; that association is the hub's inference, not a proved reconstruction. It does not supply the original relaunch/park failure text. The earlier report of no new launch diagnostic referred to the original approximately17:49–17:51 relaunch failures, not this later launch.

The hub also reports parley.nvim:0 subsequently returned "recovered (Fresh)" with no new failure record, giving approximate time18:40. The time and chosen menu action are not independently verified here; this report is retained as supplied rather than used to infer an automatic Fresh choice or precise ordering. Production state was read only; no additional recovery, launch or diagnostic write was performed by this worker.

2026-10-19 — #681 landed as PR443; hub assigned #683 from fresh main as the failure-persistence prerequisite for #695. Evidence handoff to pair:1 was submitted (566735e7-a204-43f3-9be4-a45c608875c4) and acknowledged. Read the current #695 plan from its origin branch, including §§2a/P3. Verified pair:1 contract messages a5372958 and fc8d4000: one dispatcher recorder, same file with 64 entries, optional slot-only failure, typed phase/holders/outcome capability, preserved unknown nested members, optional superseded_at and generic empty Phase.

Read-only audits found two design hazards: Read currently normalizes through clone/Upsert, so tolerant decode alone would lose unknown members; and queued preparation/guards fail before DispatchOperation while adoption/readiness can fail afterward. The draft plan carries opaque bytes through all rewrites and defers queue-owned recording until one terminal completion re-enters the dispatcher without executing effects. No production code, tests, slot lifecycle actions or operator state were changed during planning.

## Revisions

### 2026-10-19 — Approved scope disposition for the lifecycle project

Reason: the hub explicitly assigned the error-persistence part first, while #695 now owns shared teardown and #684 owns convergence. Preserve the original Spec/Done when above as incident history; the following close contract supersedes them on hub approval c35c61ae.

- Original cleanup-success/finalization acceptance transfers to #695's TestTeardownFoldsCompletedPark and its teardown work; it remains open there and is not claimed fixed by this issue.
- Original Recovery/Fresh ordering acceptance transfers to #695/#684. Archival is not treated as verified park recovery.
- The diagnosis acceptance is satisfied only to the retained evidence's limit: helper success and unresolved bookkeeping are confirmed; the initiating error and operator menu choice remain unknown. Failure persistence enables a later recurrence to supply the missing text.

Approved replacement Done when for #683's implementation:

- [ ] Admitted menu, reattach and CLI lifecycle failures from preparation through final completion retain their original bounded message, frozen target/transaction and operation nonce in the same diagnostics file; typed failure details are preserved without inventing a generic phase.
- [ ] Known types/required fields/duplicate keys stay strict, while unknown entry and nested members survive real read/modify/write and enrichment byte-equal. Legacy launch records remain compatible; the ring is bounded at 64 entries and 1 MiB.
- [ ] Switcher and show share a metadata-cached latest unresolved failure projection, including empty slots, and later terminal success supersedes older failures for that slot without deleting evidence or writing on ordinary success with no failure to supersede.
- [ ] CLI terminal receipts retain FailureNonce; cancellation, partial outcomes, save errors and late/duplicate completions cannot masquerade as successful lifecycle effects or clear newer failures.
- [ ] Pure/stateful/real-private-service proof, the required recorded suite and mandatory close review pass before normal landing; any operator live acceptance is explicitly scheduled with the hub and recorded as owed if deferred.

2026-10-19 — Scope disposition and replacement close contract approved by hub in verified c35c61ae; original clauses above are retained incident history, with park finalization/teardown owned by #695 and Fresh ordering by #684. Pair:1 approved P3/2a fit after correcting byte-budget retention. Hub b00ff428 approved oldest-whole-entry eviction to retain the newest within 64 entries/1 MiB, plus OperationAttempt.Finish as sole recorder instead of dispatcher completion re-entry. No implementation before change-code; private-service proof before landing and disposable-slot display acceptance owed after next operator restart, no :0 staging.

2026-10-19 — Plan-quality round1 blocked on PQ-1..PQ-4: late failure after newer success, existing capture writers, holder-secret boundary and unnamed test strategies. Revised the full class: ordered active-attempt coordinator with ephemeral success watermark; evidence-only captures and one Finish publication; catalog-only executable kind, never argv; named-function strategies. Hub bfb77f25 approved this final design refinement and requires stopping if another coordinator hole appears. No implementation or tests yet.

### 2026-10-19 — Implementation checkpoint before recorded suite

Implemented the approved failure-persistence scope, with no additional coordinator state. The store remains the namespace's launch-diagnostics/failures.json, now 64 entries and at most1MiB with oldest-whole-entry eviction. Unknown entry/address/holder/process members survive byte-equal rewrites, enrichment and supersession. Byte selection encodes each row once: the counted regression first failed at35/135 calls for8/16 rows, then passed at exactly8/16. A near-capacity success annotation regression failed on both real/fake stores, then passed with the same whole-entry budget enforcement.

The sole Finish recorder covers synchronous dispatch and queued preparation, guards, effect execution, adoption and readiness. Existing launch helpers defer evidence publication under an owner. Slot identity is frozen from the selected row/cache, including empty rows selected by RowKey. Receipt mapping retains one FailureNonce through all eight existing typed classifications. Unconfirmed errors remain unproven, refusals remain refused, and generic Phase stays empty. Relative/unresolved paths do not change admission merely to obtain optional diagnostics. Success/cleanup warnings preserve the original typed lifecycle result and do not request repetition of a completed effect.

Verification completed on this implementation worktree:

- Targeted four-package coverage (LifecycleFailure, LaunchFailure, OperationAttempt, Receipt, Readiness, RemoteOperation, SlotOperation, Reattach, OperationAdoption, AgentSwitchService, HubMove, HubStartup) passed; most recent completed run before final pure-order/budget assertions was core22.431s, cmd22.973s, tty0.541s, message1.513s. Follow-up final assertions are run again before the pushed suite.
- Focused race coverage passed on core/cmd/tty (7.448s/2.151s/1.484s); one earlier in-flight invocation compiled the store task's intentional RED counted-test version and failed35/135 calls. This is retained TDD evidence, not a flake or a passing retry over an unexplained defect.
- Artifact/source, test-isolation and declaration guards passed (artifactpath7.353s, testpair1.904s, termcmd0.336s). No local-gates configuration was changed.
- TestLifecycleFailurePrivateServiceSurvivesReceiptLossAndSupersedes passed using the real private Unix service, queue, production hub authority and real temporary diagnostics file. A pre-dispatch failure retained its reason/nonce/park identity; fresh service receipts became unknown after restart; show still printed the reason/nonce; successful recovery hid the failure while a reopened store retained superseded evidence. No operator socket or slot was touched.
- A controlled wiring-removal mutation (skip the queue attempt factory) made that same private-service regression fail exactly at its pre-dispatch retained-failure/nonce assertion. The source was restored in a finally block and normal coverage passes.

The required recorded full suite and mandatory close review are next. No landing or live acceptance is claimed. Hub c35c61ae explicitly accepts private-service proof before landing and owes the disposable-slot row/show check after the operator's next Couch restart, with no :0 staging. #681's pair:8 live acceptance also remains owed to that restart; the hub tracks it separately.

2026-10-19 — Final verification exposed a coordinator identity hole: exact tag-only and cached-path calls for the same thread used different keys. Stopped and reported per hub bfb77f25 (request930e5b57); hub9e79d926 approved binding once through exact ThreadStore.GetThread before Begin after admission, unchanged cached queue paths, unknown binding refused before effects. The production-factory mixed-target and dispatcher unknown-binding regressions both failed as expected before the fix. Central lifecycleFailureTargetKey remains the single key seam for #695 LifecycleAddress.Key(); no new coordinator state or alias map. Sequential legacy exact-thread supersession also retains evidence until a successful same-thread completion.

Final identity-fix verification: core LifecycleFailure/LaunchFailure/OperationAttempt coverage passed8.608s; command/console/message focused coverage passed22.091s/0.596s/1.445s. Focused race coverage passed core10.772s, command2.132s, console1.509s. Artifact/source/isolation/declaration guards passed9.353s/3.563s/0.454s. Both previously red identity regressions now pass; the cached-path attempt performs no second GetThread. Recorded full suite and close review remain next.

### 2026-10-19 — Close REWORK and hub-directed replacement plan

Close round1 returned REWORK with BR-1 unreadable-record Reboot blocked by diagnostic binding, BR-2 missing terminal owners for cache misses/hub moves, BR-3 duplicate standalone switch publication and BR-4 source-symbol guard failures. Existing unreadable-record runtime tests reproduced BR-1. The old-head suite on3b937034 was cancelled while queued through local-check cancel; other checkout runs were left untouched. Server CI had passed, but no required full-suite pass is claimed.

Hub90f47077 requires a simpler replacement: append failure evidence once at completion; no coordinator, admission order, mandatory target binding or write-time supersession; display filters against existing successful thread-record timestamps. BR-2 verified-target work is dropped. The short replacement Revision/state table is appended to the durable plan and awaits pair:1/hub review before recorder code changes. BR-3/BR-4 isolated real fixes are retained: exact-count switch capture tests first showed two writes then pass with one; full artifactpath package first failed three orphan symbols then passed after removing unused exports/test observation from production (the earlier filtered regex had omitted the Symbol test). Root cause of the original production park incident remains unproved and is not changed by this diagnostic work.

Replacement close contract pending approval: the completion-log/read-time-visibility Revision supersedes earlier write-time supersession, coordinator/admission-order and mandatory-binding clauses. Retention, preserving strict codec, failure reason/nonce visibility, original-result preservation, private-service proof, required final-head suite and mandatory close review remain required. No claim of code completion is made for the replacement.

2026-10-19 — Replacement approved by hubadc93fc8 and pair:1 review18e9b3f1, with huba5ce89b5 confirming LastActiveAt is excluded because retirement may represent death. LastLifecycleSuccessAt(record) is the single pure seam: VerifiedPark.ParkedAt and committed live StartedAt only; #695 later adds proven teardown. Original typed details/holder shape and empty generic phase retained. Isolated BR-3 verification passed6.842s; full artifactpath guard passed13.840s. Replacement recorder code begins only after the refreshed change-code gate.

### 2026-10-19 — Replacement implementation and regression proof

Refreshed change-code returned0: plan-quality round4 CLEAN and estimate-quality INFO. The original3.851h whole-issue estimate is retained, not treated as remaining work: its ordered-coordinator primitive is now the simpler completion recorder and deletion/regression update; shared integration modules include private-service restart/receipt-loss, wiring-mutation and focused/race/suite qualification. The estimate review notes that earlier design/rework exceeds a straightforward one-review allowance; this is retained variance rather than retrofitting the estimate.

Removed the pure ordering model, active-slot groups/watermarks, mandatory tag binding, success storage operations and coordinator fields. Retired coordinator JSON members now survive as opaque unknown values, including formerly invalid types, without being read as authority. Every completion owner can retain exact dispatched reference evidence even with no cached row; preparation can contribute the actual call's address. This is evidence enrichment only, not BR-2's rejected verified-target work. Hub moves and preparation/adoption/readiness failures all reach the same Finish. Ordinary success does not allocate diagnostics entropy or read/write storage. BR-3's standalone/nested single-publication proof and BR-4's full symbol guard remain intact.

LastLifecycleSuccessAt(record) is the only success-time definition. It ignores LastActiveAt/death/retirement, and uses verified park or a committed live incarnation's StartedAt. Row/show share the read-time filter; missing/equal timestamps retain historical evidence. The real private service test now proves retirement alone leaves failure visible, a subsequent actual committed start hides it, and the diagnostics bytes stay identical throughout those successes. No operator namespace, focus or slot action was used.

Replacement focused core/cmd tests passed8.163s/1.436s, including both previously failing unreadable-record public recovery tests. Focused race coverage passed core4.841s/cmd1.807s/tty1.409s. Full artifactpath/testpair/termcmd packages passed7.465s/9.107s/2.579s. Two controlled mutations were rejected: allocating/refusing on diagnostics entropy before the action fails the zero-IO-success regression; restoring known-cache-only attempt creation fails all three missing-cache completion cases and the hub-move target case. Both sources were restored in finally blocks; final broad focused coverage runs after restoration before publishing. These are intentional RED proofs, not suite flakes. No required full-suite pass or close SHIP claimed yet.

Final replacement broad focused coverage after restoring both mutations passed core12.489s/cmd15.156s/tty0.409s/message1.325s, including private real-service proof and both unreadable-record recovery regressions. This is the head to publish for the required recorded suite and next full close review.

2026-10-19 — Close round2 confirmed all four earlier findings addressed with reproduced regressions/mutations. New BR-5: reference-only evidence is visible on a row and literal slot-name show, but resolved path/tag show can omit it. The approved model is unchanged; centralize pure display matching and add cross-reference parity tests. Close is NOT finalized despite FIX-THEN-SHIP because its ledger still blocks on BR-5; rerun close after the fix. The6fcefcb8 recorded suite was cancelled while still queued, leaving the two other checkout runs untouched. It produced no test result. Server CI on6fcefcb8 passed; final replacement code requires a new recorded suite.

BR-5 parity proof: the target-only table failed before the fix for canonical primary-root:N, path and exact-tag inspection while the row still displayed evidence. Shared target selection/visibility now covers both stored target forms, four inspection references, missing/equal/later success, empty slots and cross-slot exclusion. Focused core/cmd coverage passed27.715s/6.259s, the full artifactpath package passed18.614s, and focused projection/show race passed5.796s. No lifecycle state, target authority or persistence mechanism changed.

2026-10-19 — Close round3 disposed BR-1–BR-5 but left new Important BR-6 blocking: optional Save runs in the console event reducer and can stall input/rendering. Hub e1f5cd25 approved moving publication after adoption to the existing bounded/joined workers, with remote readiness/callback on that worker and Console lifetime cancellation. No new queue/service/state machine. Before-enqueue refusals remain immediate visible refusals and are not logged. The short worker/context/shutdown Revision and stalled-store menu/reattach/remote proof are recorded in the plan; refreshed change-code precedes production edits. The ba4820ca suite was cancelled through its owned local-check handle because code must change; no suite pass or close completion claimed.

### 2026-10-19 — BR-6 worker publication proof

Refreshed change-code passed (plan-quality CLEAN; estimate INFO retains original baseline variance). Publication now runs after adoption on existing operation workers, including admitted navigation completions; Run only reduces results. A separate action-local once flag protects terminal publication/callback from UI reduction/abandon replay; no registry, queue, service or coordinator was added. Remote readiness/callback stays synchronously on that joined worker and uses Console.OperationLifetime. FinishContext carries that cancellation under the existing10s store budget. Before-enqueue refusals perform no logging, per explicit hube1f5cd25.

RED Run tests showed menu/reattach and remote publication blocked actual input; the enqueue-refusal test showed unintended IO. The core owner-context stub failed immediately because it ignored canceled lifetime. The corrected worker tests cover menu, reattach, hosted navigation and remote input, Stop cancellation and joined cleanup, exactly-once callback/abandon behavior and immediate refusal. Full couchtty package passed41.546s. Final broad focused core/cmd/tty/message coverage passed30.454s/34.309s/0.516s/1.503s. Focused race coverage passed4.903s/2.111s/1.597s. Full artifactpath/testpair/termcmd passed6.721s/7.500s/2.497s.

A real private-service stalled-store proof also passed: actual Unix transport, production authority/queue, console input pump, lifecycle publication and Stop all compose with the same lifetime. Removing only the constructor's lifetime wire makes that regression fail after10.34s with publication context deadline exceeded, rather than context canceled; source restored in finally and broad coverage rerun afterward. This is intentional mutation evidence, not a flake. All owned test runners finished; the required recorded suite and fresh close review are next on the published head. Live display acceptance remains owed after the operator restart and must use an admitted failure, since pre-enqueue refusals are intentionally immediate/unlogged.
