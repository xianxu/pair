---
id: 000872
status: working
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: 'a0ae1aa1aa7fdc9bfae2779b246362d6a1612101' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T16:04:42-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "45e204ae", done: "bdbf53cd"}
---

# Admitted worker reboot loses steward positive-list authentication during receipt polling

## Problem

In the approved #865 disposable Claude scale collection, N10 failed while starting worker5. The CLI first safely retried a catalog-warming refusal, then admitted reboot request c7f11c5f-10b2-472f-822d-ced172133631. Subsequent operation-status polling failed with unavailable: caller steward thread9223742121301770243-fixture-1 was not positively listed. The CLI correctly reported outcome uncertain, and no reboot was resent. This is a captured symptom; actual session disappearance versus discovery/projection error is unresolved. Do not assert a confirmed scale threshold or merge this with #866's earlier startup-probe panic.

## Spec

Classify and correct post-admission loss of positive-list caller authentication without weakening caller ownership, resending an uncertain mutation, increasing timeouts, or mutating the live fleet. Follow the accepted reboot to pollSlotOperation in cmd/internal/couchcmd/messages.go and the shared validated listing through messageThreadResolver in message_service.go and ListedPairSessionContext in cmd/internal/couchcore/artifactcollision.go.

The retained capture is /Volumes/TestScratch/tmp/pair865-scale.HgmR9S/evidence/collection.log; partial manifest claude-N10/manifest.json. Source1eb2135667e8b608cd369529ffdff19353f8d486, compiled vcs.modified=false; exact executable retained with SHA256a44dd8dfa3d6161fe3d7981ac12c649372496bc71f474a96346b322eddd001ff. The N10 subtest failed142.50s, before detach/workloads/profile. All owned generations were proved absent and direct child/IO joined; native root was removed by owner. Four worker pane captures and bounded server log tails exist; the failing caller steward pane, raw listing, receipt and binding census were not retained. Absence of #866's lib.rs1462 panic in truncated logs does not exclude every earlier server event.

## Done when

- Diagnostics are in place; cause unclassified (not reproduced in one authorized sample); the next occurrence carries the evidence.
- A refused positive-list caller proof retains the exact judged listing, observation time, full hash/count, bounded bytes/truncation and selected projection facts through the existing refusal.
- A deterministic accepted-to-status-auth-failure regression exercises the existing stateful seam and preserves conservative uncertain outcome, ownership and no blind resend.
- The authorized shared disposable Claude sample and exact cleanup are recorded; no live-fleet manipulation or additional native retry. Open a causal follow-up only if the failure recurs.

## Plan

- Preserve the existing capture, add bounded admission/status/listing-generation/steward-binding evidence at the refusal seam, and design only after classifying the loss.
- Follow normal issue gates; this filing grants no implementation or native rerun approval to another slot.

Approved diagnostic phase (no correction selected; final disposition recorded in Revisions):

Approved diagnostic implementation detail: [diagnostic plan](../plans/000872-reboot-receipt-caller-positive-list-plan.md). Submitted ops:0 receipt 967ef946-fe21-42f9-9a25-a412cfe8a2da approves this phase and one headless Claude sample through worker5, with nothing alongside it; stop after that result before behavior changes.

- [x] Retain the exact immutable cached listing used by a failed caller proof, with observation time, full byte count/hash, bounded raw bytes and the exact selected name/state; include selected binding/claim-count facts from that same projection. Do not perform a second listing and label it as the judged observation. Reuse ListingPoller's existing stored snapshot; no new durable barrier or liveness authority.
- [x] Carry the bounded diagnostic through the existing unavailable/refusal path so the public CLI test can retain it before cleanup. Keep human-facing uncertainty/no-resend semantics, exact caller checks, parser grammar, five-second query budget and poll cadence unchanged. No fallback to remembered connection flags or server presence as authorization. Precise evidence representation and transport limits are pinned in the implementation plan.
- [x] Exercise accepted reboot -> valid listing omits caller / marks it EXITED -> status refused using the real resolver and stateful Zellij fixture. Assert one mutation, no admission resend, request ID retained, unavailable/uncertain preserved, and diagnostic raw bytes correspond to the exact judged generation even if a later refresh restores the caller. Controls cover malformed/stale listings and conflicting claims without collapsing their distinct outcomes. Count listing queries at 8 and 16 rows; status authentication must spawn none.
- [x] Add shared disposable-harness failure capture before cleanup: caller pane, exact operation request/receipt if independently readable, selected launch/PID/binding evidence, and a separately timestamped read-only native listing/socket/process observation. An unavailable receipt remains unknown; do not use another actor or broaden authorization to retrieve it. Bound byte counts explicitly and retain hashes/truncation/errors. Later probes are causal diagnostic evidence, never authority for the refused operation.
- [x] Obtain approval and execute one focused disposable Claude startup through worker5, the stage at which the retained N10 case failed (no scale threshold is proven). Current main permits up to 8 workers; no #865 scale harness copy or stacked branch. Actual existing fixture time/output bounds retained; no profiler/spinners, full native matrix or live fleet.
- [x] Return the successful sample and unresolved cause to the TL before any behavioral fix. Stop rather than repeat until lucky. The explicit diagnostic-only scope revision below authorizes normal close and landing.

## Log

Filed unclaimed by pair:9 per TL ops:0 direction. Largest completed original cohort N5 (its waveform was separately invalid because startup system submissions stopped redraw before capture). #865 retains workload repair and measurement; #871 separately lands the opt-in profiler. No retained native owner remains from this failure.

### 2026-10-24 — claim and initial causal trace

Dispatched by verified ops:0 receipt 2fab61cc-ee8e-4641-abad-09e875dc45b2 after #851 landed, while #874's approved planning-only contract waits for #737. Claimed and start-plan completed on this separate main-based branch. The first start-plan correctly refused while on the parked #874 branch; its committed/pushed clean checkpoint was preserved, switched to main-slot2, then reran successfully. No stacked commits or runtime changes.

Read retained collection.log and traced CLI admission -> pollSlotOperation -> handleSlotOperation/liveCaller -> messageAuthority.current -> messageThreadResolver -> ListedPairSessionContext. The exact captured error is the resolver's absent/non-present projection branch, not a stale-poller error, timeout, duplicate attribution error, wrapper PID mismatch or nonce mismatch. That narrows which evidence failed but does not prove why the session was absent. The capture lacks the raw listing/caller pane/binding census needed to distinguish actual disappearance from an erroneous listing. Preserve conservative uncertain outcome and no resend.

Source investigation is read-only, including a bounded delegated listing/projection analysis. No native rerun, code, new timeout or authentication relaxation. Next: finish tracing listing parsing and namespace effects, inspect retained manifest and exact original source, then send the hub a causal classification and bounded diagnostic/correction plan before change-code.

### 2026-10-24 — evidence classification and proposed next experiment

The read-only listing/projection audit found no row-count truncation or special N10 limit. Successful validated listings wholly replace the previous cached answer, including an omitted selected row, an EXITED row or the native exact no-active-sessions diagnostic. Malformed refreshes retain the old listing until its five-second staleness refusal. Production refresh has one owner; namespace cache eviction does not delete census rows. A missing or EXITED selected row yields this exact error; ambiguous claims and stale/error observations yield different errors. Source alone cannot distinguish real session/socket loss from a valid but incomplete native answer. No corrective hypothesis is proven.

Retained manifest pins candidate SHA256 a44dd8dfa3d6161fe3d7981ac12c649372496bc71f474a96346b322eddd001ff, source 1eb2135667e8b608cd369529ffdff19353f8d486, Go1.27.3 and Zellij0.45.1, cleanup_proved=true. Failure at08:22:50 predates cleanup start08:22:50.003769; the post-cleanup client errors at08:22:52 cannot establish a prior server failure. No current-source changes to the traced production files were found since that source pin. Existing selected-listing attached/detached/absent/exited and ambiguity controls PASS: go test ./cmd/internal/couchcore -run '^TestSelectedListing' -count=1, 2.520s. This is baseline behavior evidence, not reproduction of the cause.

Proposed diagnostic steps above preserve the exact proof observation, then perform separately labelled bounded probes before native cleanup. Await phase approval before code or a native experiment. #874's complete planning-only contract was approved in submitted ops:0 receipt 02a7b9c6-ae86-422c-a2a5-e1ef20819a82 and committed/pushed on its own branch; it remains waiting for #737.

### 2026-10-24 — diagnostic admission and first RED/GREEN proof

Verified submitted ops:0 receipt 967ef946-fe21-42f9-9a25-a412cfe8a2da approves the bounded diagnostic phase and ONE headless Claude sample through worker5, alone, stopping afterward before behavior changes. change-code completed exit0 (quick inferred; close will upgrade if implementation leaves the shell). Canonicalized the durable plan filename after its naming notice. No native run yet.

RED: launcher tests lacked evidence APIs; core test lacked selected evidence; accepted-then-lost CLI tests failed because the existing error omitted judged evidence. GREEN: launcher focused evidence/poller/count tests PASS0.388s; selected-listing controls and diagnostics PASS2.163s; real resolver/stateful native listing fixture -> unavailable -> public CLI uncertainty/no-resend controls PASS2.057s. Both missing and EXITED snapshots are retained even after a subsequent good refresh. One reboot admission and one same-ID status read; three explicit refresh queries and zero query/mutation effects from the proof reads. No deadline, parser or caller authorization behavior changed.

Plan correction discovered before native execution: cmd/internal/couchcmd has an existing96MiB TestMain self-write counter; cmd/couch and native children do not inherit that counter. The initial plan's phrase per-process write96MiB was inaccurate. Keep all actual existing byte/file/deadline bounds; do not invent a new child write-budget mechanism for this diagnostic. Report the precise limit before the approved sample. The bounded delegated harness task owns the one targeted Go invocation slot now; primary runs no tests until it returns capacity.

### 2026-10-24 — diagnostic candidate ready; one native sample next

Explicit caller capture now prioritizes the real public Actors binding within the existing four-source selection and private descriptor/socket bounds. It records selected PID generation, bounded allowlisted fixture/ready/pane evidence and a separately labelled later native listing, with existing shared diagnostic deadline and exact cleanup. No native run yet. The native test verifies the emitted Couch binary's own vcs.revision equals its recorded source head and vcs.modified=false before starting the private fleet.

RED/GREEN: helper initially lacked the caller capture API; focused tests then passed. Parent review added a failing control for a fifth source omitted by caller priority but truncation falsely absent; fixed the diagnostic flag, retaining the four-source bound. Added an unbounded-valid-caller refusal control and a >8192-byte listing proof that diagnostic truncation never truncates parsed authority. Focused race checks passed sequentially across launcher1.325s, core2.922s, couchcmd1.930s and testcouch9.360s; after the final helper/large-list controls, launcher race PASS1.304s and testcouch race PASS7.763s. cmd/couch native test compiled with an explicit normal SKIP0.256s; that is no native acceptance claim.

Next, after this checkpoint is pushed and the tree is clean: execute exactly once, alone and headless: PAIR_NATIVE_CALLER_LISTING_DIAGNOSTIC=1 go test ./cmd/couch -run '^TestDisposableCallerListingDiagnostic$' -count=1 -timeout=10m -v. Its shared fixture setup180s/runtime120s and existing read/output bounds remain; no profiler/workload and no new process write counter. Preserve the complete log outside the disposable root, joined cleanup outcome and actual candidate pin. Stop after this sample, commit/push its result and report to the hub before any behavior change, native rerun, close or merge.

### 2026-10-24 — the ONE authorized Claude sample passed; stopped

Executed the exact invocation above once, alone and headless. PASS124.74s (package124.881s), exit0, through worker5. Candidate's emitted build metadata verified Go1.27.3, vcs.revision=f24dc287ba445bb177a119519049f4005b676770 and vcs.modified=false. Startup tools included Zellij0.45.1; no profiler/workloads or additional native personality ran. Preflight machine load was20.81/22.48/24.35; no test ran alongside this sample in pair:2.

Complete log retained outside the cleaned fleet at /Volumes/TestScratch/pair872-native.EfAG8r/native.log; SHA25649a80409da7725cd3585dc3563bc8576465136ce95f68568c2eb316813ddda6f. Private run49e4cd29137b889b0cbf189ed7b823d5, root /Volumes/TestScratch/pc-3260688545. At23:25:02.699521Z the original genuine public steward binding still matched wrapperPID25082/start1792884218.893949. Its allowlisted fixture identity matched the owned run. The explicitly prioritized steward pane was readable (411bytes, retained SHA256fb1df4090c8d580c434f61b337533c6c63a84fc64fed69dfd5ec07264e3cd14c), with a fresh empty Claude composer. The separately labelled later listing completed23:25:02.81993Z,274bytes, SHA25676a5fcd5d3d367d709d33b3e0df3aa1b129f084d33cf5a2284de02be49565e32, untruncated, all six sessions including the exact steward. These are later diagnostic observations, not an earlier authorization frame.

Cleanup reported all owned generations absent, direct child and IO joined; native owner removed its private root. No original positive-list failure occurred, so there is no native refused-frame capture and the original cause remains unclassified. The deterministic omitted/EXITED-frame transport/no-resend regressions and focused race results remain the proof of diagnostic behavior; this successful native sample is not proof of a root-cause fix.

Stopped after this sample per ops:0's approval. No behavior change, second sample, recorded full check, close or merge started. Next step is the hub's decision on the retained diagnostic coverage and unresolved-cause disposition; do not repeat native collection without authorization. #874 remains approved planning-only, waiting for #737.

### 2026-10-24 — diagnostic landing authorized; reader audit

Verified ops:0's submitted disposition f0957f0c-2f37-423e-9e2f-f4c628a37090: revised acceptance is diagnostic retention with cause explicitly unclassified, no follow-up until recurrence, and normal pre-close/close/merge. Updated active Done-when and retained the original obligation as superseded in Revisions; no root-cause fix is claimed. Added the diagnostic and explicit-caller qualification map to existing indexed Couch atlas pages.

Required inventory initially named two unmapped readers. Audited both bodies and helper inputs before recording individual hashes: cmd/couch/caller_listing_diagnostic_test.go builds the real checkout/runtime assets and uses testcouch, so non_go_full_fallback; cmd/internal/testcouch/caller_evidence_test.go reads only generated private files and its embedded shell fixture, so local_or_literal_fixture. No repository source walker or fixed scanner was added. Recorded only those two per-file entries; make test-affected-inventory now passes. Next: push this exact checkpoint and run sdlc local-check run --pre-close; retain its runner, then close and merge through the normal gates.

## Revisions

2026-10-24 — Verified submitted ops:0 receipt f0957f0c-2f37-423e-9e2f-f4c628a37090 authorizes landing the diagnostic coverage with normal pre-close, close and merge. Revised Done-when: **diagnostics in place, cause unclassified (not reproduced in one authorized sample); the next occurrence carries the evidence.** The original requirements to identify whether the exact steward session/listing/projection changed and correct the measured cause are superseded for this issue, not claimed satisfied. The stateful accepted-then-missing/EXITED uncertainty/no-resend regressions and exact refused-frame retention remain required and implemented. File a causal follow-up only when a new occurrence supplies evidence. No behavioral correction, new native collection or live-fleet manipulation is authorized by this disposition.
