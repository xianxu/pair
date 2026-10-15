---
id: 000579
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours: 2.622
card_mirror: '73739e1a687bd7cb0ba861036e92c0672decd7d8' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T15:07:36-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# pair wrap writes ~200× more than it appends (~4 MB/s per slot): per-event sync on scrollback/wrap-events logs; also no rotation

## Problem

2026-10-15 (operator's system monitor): each `pair wrap` process (`--scrollback-log …/scrollback-1-<tag>-codex.raw`)
**writes 3.4–4.1 MB/s to disk**, about 29 MB/s total for the fleet (~2.5 TB/day of write wear). Yet the files it
writes grow only **~17 KB/s** (`wrap-events-1-<tag>.jsonl`) and **~2.3 KB/s** (`scrollback-…raw`) per active slot,
measured over 10 s. That's a **~200× write amplification**, most likely a sync (fsync/F_FULLFSYNC) or a re-open per
small event write, so every few-hundred-byte append costs whole blocks plus filesystem journal writes. It's also likely
behind the FSEvents daemon's ~200% CPU seen on 10-14.

Separately, these files are **never rotated**: wrap-events logs of **3.9 GB** (pair-8), **2.6 GB** (pair-43), 1.9 GB,
1.7 GB, 1.0 GB; the pair data dir was 57 GB on 10-14.

## Spec

1. **Measure first:** confirm the cause (count write/fsync syscalls per second for one wrapper, e.g. `fs_usage` or
   dtrace on a scratch slot), and record bytes-written vs. bytes-appended.
2. **Fix the amplification:** buffer event and scrollback writes and flush on an interval or size threshold; fsync only
   at real durability boundaries (e.g. before acknowledging a delivery, or at turn end), not per event. State what a
   crash may lose (at most the last interval's events) and why that's acceptable for each file.
3. **Retention:** rotate wrap-events and scrollback logs by size (e.g. 64 MB segments) and keep a bounded number or age
   (consistent with crash-only: what recovery and orientation actually read must remain).
4. A counted invariant (#204 style): bytes written per appended byte stays under a small bound for a synthetic event
   stream (no timing thresholds).

## Done when

- Wrapper diagnostics buffer/coalesce records at64KiB or dirty5s; compact reservations contain no payload copy and a healthy batch has one checkpoint rename. The counted real-writer stream stays below3x logical amplification, counting all metadata and reservation bytes.
- Unchanged validated registry declarations skip content writes. Shared persisted diagnostic policy enforces64MiB current+3 generations (256MiB normal payload cap) across all supported writers; unknown writer proof refuses growth/destruction. Protocol-aware deleted roots do not globally block cleanup, while generation fences and legacy unknown-root refusal remain.
- A complete interrupted batch recovers by its exact digest; an uncertifiable incomplete batch is preserved as a marked-unverified quarantined generation and a fresh current log starts after safe writer proof. Nothing unverified is adopted or truncated; quarantines participate in the diagnostic generation bound.
- Raw/events writes flush on250ms/size/orderly shutdown without changing witness identity or raw bytes; raw flush precedes sidecar offset publication. Faithful raw+events size rotation is unassigned #582, outside this issue.
- Deterministic crash/progress/ownership/rotation/buffer tests, recorded suite, build and mandatory fresh SHIP reviews pass. Receipt, park and recovery synchronization remain intact; process-crash and power-loss bounds are documented.
- After landing, live process bytes-written versus raw+event bytes-appended and diagnostic state inode-change rates are recorded in ## Log with before/after ratios, exact binary/process/activity context, and reported to hub. Baseline85.8x/10.006s and2187 observed replacements/60.003s remains the comparison.

## Plan

Proposed durable plan: [wrapper log IO](../plans/000579-wrapper-log-io-plan.md).
The approved scope and protocol are recorded in the Revisions below and the durable plan; faithful raw capture size rotation is unassigned #582.

- [x] M1 — Coalesced diagnostic IO and authority-preserving raw/event batching.
- [ ] M2 — Safe bounded diagnostic generations, compatibility and qualification.

## Log

### 2026-10-15
- 2026-10-15: closed M1 — BR-1 EOF drain/error regression verified RED/GREEN; BR-2 real rename then repeated injected directory-sync failure followed by success preserves bytes/inode/mtime, three syncs/one stage; removing reconfirmation is RED0.07s, real selected registry/amplification/lifecycle tests GREEN-race11.768s; make -k build passes; logical1.0181x and isolated physical61.01x to1.32x measured; M2 and postlanding acceptance pending.; review verdict: SHIP

- Claimed and started on pair:3 from main a3beedd0074c536a49396ca68e5e038eebfb35d7. start-plan refreshed sibling Ariadne, then reran successfully.
- Measurement before design: proc_pid_rusage(pid 73688, RUSAGE_INFO_V2) over 10.006 seconds counted 7,385,088 physical disk-write bytes versus 86,077 appended raw+wrap-event bytes (85.80x). Files grew 63,591 wrap-events +22,486 raw bytes. Python ctypes called libproc directly; no root or mutation of the wrapper. Activity level affects the rate; this is one observed window, not a universal 200x.
- A 3-second sample of PID73688 found 109 sampled rename frames and 43 write frames; sampling is attribution evidence, not a syscall count. sudo -n fs_usage refused because a password is required. Hub confirmed agents cannot sudo and authorized inode-change/file-growth measurements instead (receipt ee56adcd-39bb-4351-a29b-012021701a6b).
- Read path: diagnosticlog.appendChunk publishes an intent (including base64 payload), reopens/appends the log, recoverAppend reopens/reads the tail, then republishes state. Both ordinary state publications are nondurable (no hot-path fsync), but each uses create/write/close/rename. Thus per-event atomic metadata churn, not per-event payload fsync, is the concrete hot path. Root registrations are durably republished on each Open, including unchanged entries. Existing diagnostic rotation is 64MiB/24h but refuses on incomplete writer proof and has no count cap for young generations.
- ARCH-ORDER/ARCH-SECURE: raw/event capture is an identity-bound pair, not two independent diagnostic files. Recovery, park and previous peek validate producer witness plus raw/event identities; naive copy-truncate/rename would invalidate it. Map consumers before choosing the bounded representation. No production change yet.

- Hub-approved alternative syscall evidence: 60.003 seconds, 20ms stat sampling observed **2,187 state inode changes/minute** (a lower bound because multiple replacements between samples collapse), with wrap-events growth 1,185,998 bytes and raw growth122,922 bytes. Two registry files changed once each. Main amplifier in this window is the append state publisher; registry churn is a separate redundant-write path, not observed at every event.
- Consumer exploration (read-only subagent): no full VT semantic checkpoint/restore exists. Visible-screen ANSI seeds omit hidden main/alt screen, modes, pen, margins, saved cursor, charset/tabstops and split parser/grapheme state. Bounded raw suffix eviction would silently change rendering. Exact capture park/recovery/previous-reader migration requires a certified segment/checkpoint representation. Sent scope options to hub (83dfc5b9-dcb6-4550-96f1-91f0054338af); awaiting its choice while developing the diagnostic design. Original requirements unchanged pending that decision.

- Temporary untracked Go overlay probe (no source mutation) used the real diagnostic writer with 1,000 distinct JSONL events: payload44,890 bytes, metadata1,087,924 bytes, exactly2,000 state publications; logical bytes-written/appended25.235x before APFS block/journal overhead. Probe uses existing fault checkpoints after actual metadata publications, marshals the observed disk states, and counts the payload returned by real Write. This is a code-path measurement, not a kernel syscall trace. Scratch overlay under pair579-probe-f2a4f2vd.
- Rotation blocker demonstrated through the production VerifyWriters/OSInspection seam with a read-only tracing processInspection: PID72769 (Zellij) advertises PAIR_RETENTION_PROTOCOL=1, but its PAIR_DATA_DIR points under removed birthrace-scratch-2453556978. Runtimes attempts EvalSymlinks of that irrelevant root before returning the protocol-aware row; failure becomes diagnostic writer inventory incomplete and blocks all rotation. The protocol row is later exempt from legacy-writer root comparison anyway. Proposed minimal correction: retain generation/environment fences, classify protocol participation before unnecessary root canonicalization; unknown legacy roots must still refuse. This is not authority to stop the leftover process.

- 2026-10-15T22:18:28Z: verified acceptance receipt e774d4c0-5be6-4c77-8e0a-8e25ceeb3cd4; explicit post-landing measurement added below. This is not approval of the pending scope/retention design. Production code remains unchanged.


## Revisions

### 2026-10-15T22:18:28Z — operator adds post-landing live acceptance

Reason: hub receipt e774d4c0-5be6-4c77-8e0a-8e25ceeb3cd4 conveys the operator's explicit acceptance request.
Delta: after the fix lands, measure a live slot on the landed binary and record before/after process bytes-written versus actual raw+wrap-event bytes appended, amplification ratio, and diagnostic state inode-change rate in this issue's Log. Record exact binary/commit, PID/generation, paths, sample durations and output/activity context. Inode sampling is a lower bound when replacements occur between observations. Keep the existing before numbers (85.8x over10.006s and2187 observed state replacements over60.003s); do not substitute synthetic counts for the live check. The hub will separately sample whole-disk write rate; do not attribute that aggregate to this slot without its measurement. A staged/pre-merge smoke does not satisfy this additional post-landing check.

- [ ] After landing: record live before/after bytes-written/appended and diagnostic state inode-change rate in ## Log; send results to hub for its whole-disk comparison.


### 2026-10-15T22:23:51Z — #579/#582 scope split and append-protocol revision

Reason: hub's submitted scope decision8cb950d6-7819-48dd-9813-e10321f20ad8. #582 is filed unassigned on main with the witnessed raw/events checkpoint and consumer findings. Delta: #579 now owns only diagnostic write amplification and diagnostic size/count bounds, counted regression and live before/after acceptance. Raw capture rotation/retention moves to #582; raw/event producer and readers remain unchanged here. The original raw-rotation Done-when is discharged by this authorized split, not claimed implemented.

Current Done-when for #579:
- Wrapper diagnostics coalesce writes; each flush uses one small non-payload reservation write and one checkpoint publication instead of per-event intent+commit renames and base64 payload duplication. Healthy synthetic real-writer amplification<3x includes all reservation/metadata bytes; idle and unchanged registry opens write no metadata content.
- Diagnostic generations enforce64MiB size and current+3 count policy where evidence is certified; unknown proof stops growth, preserves uncertain legacy data and reports the block. Demonstrated protocol-aware removed-root rotation refusal is fixed without weakening legacy/unknown process protection.
- Legacy pending intent migration, compact reservation crash/partial/error behavior, replacement refusal, buffer/worker lifetime, rotation/retention and cleanup have deterministic red/green regressions; mandatory suite/build/reviews pass.
- After landing, record live bytes-written/appended and state inode-change rates against the preserved85.8x and2187/minute baseline, exact build/process/activity context, and report to hub. Hub owns aggregate whole-disk sampling.

The governing protocol/tasks and material killed-partial-write tradeoff are appended to the durable plan's latest Revision. This is the proposed executable design for hub approval, not implementation. Initial raw-batching steps are superseded by this narrower scope.


### 2026-10-15T22:25:16Z — proceed on the narrowed diagnostic scope

Verified hub receipt8f573644-79af-4c74-99b6-58677b6b5137 explicitly says Go on diagnostic write amplification/bounds and the narrow protocol-aware root resolution fix; raw/events rotation is unassigned #582. Proceed through the full-flow plan gate with the latest governing protocol revision. No raw/event production or reader changes. Post-landing live measurements remain outstanding acceptance work.


### 2026-10-15T22:28:30Z — plan-quality refinements PQ-1/PQ-2

The first plan-quality review correctly found shared adaptation writers could bypass a wrapper-only option. Governing plan now persists schema-v2 DiagnosticLimits and makes every managed writer enforce them; older strict decoders refuse rather than bypass. Limits can tighten but not widen on another handle. Named-function strategy table replaces the earlier prose test inventories as the current contract. No production code changed before the gate.


## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md, Method A only; source marked stale by sdlc estimate-source, so calibration is provisional. Derived after the two plan-quality findings cleared. Standard Go crypto/sha256, encoding/json, io, timers and the existing coordinated filesystem layer cover the library portion: each greenfield concern uses upper v2 design2h halved for existing libraries and multiplied0.2 for the approved detailed plan (=0.2h), with upper0.8h implementation scaled0.4 (=0.32h). These are two distinct concerns: compact reservation/recovery and bounded asynchronous batching. Smaller existing-policy/registry/proof and integration modules use upper/mid scaled ranges; the record below includes two actual review boundaries and live measurement, not only coding. Familiarity1.0 for this existing Go codebase, thorough-plan design buffer15%.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.02 impl=0.12
item: cross-cutting-refactor design=0.10 impl=0.20
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.02 impl=0.20
item: milestone-review design=0.02 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
design-buffer: 0.15
total: 2.622
```

Design subtotal0.68h; implementation1.84h;0.68*1.15+1.84=2.622h. The final smaller-module row is the repeatable live counter harness and post-landing comparison, not an external API integration. No novel external service/library discovery assumed.


### 2026-10-15T22:43:57Z — explicit approval restores raw batching and chooses partial quarantine

Hub receipt56cfc2ee-a52f-40c2-871c-a0a691b897f1 is submitted and approves the initial M1/M2 shape including250ms raw/events batching; this supersedes the intermediate raw-batching exclusion, while raw size rotation stays #582. Canonical receipt59106c8f-3364-483c-a0cf-e00246426e28 conveys approval of the compact reservation protocol and requests quarantining uncertain partial batches instead of disabling future diagnostics (receipt outcome indeterminate after submit; local reversible design incorporates that request, with no main publication under the hold). Current Done when above is now updated to the authorized split; previous contract is preserved in git and the scope Revisions.

Delta: preserve an incomplete attempted generation as an immutable marked-unverified diagnostic segment, retain its expected-tail offset/length/digest metadata, and create a fresh current generation only after exact identity and writer proof. Full-length foreign-digest mismatch still refuses. Quarantine uses existing durable rotation/replay, participates in the3-generation retention policy and never supplies verified append authority. Register the recovering current process under the stable log lock before destructive recovery, so its open handle does not falsely fail proof. No truncation/adoption of uncertified bytes. Add named pure ClassifyPartialQuarantine tests and folder-backed fault/restart/continuation regression. Unknown writer proof still prohibits rename/destruction/growth; that stronger safety condition remains explicit.

Implementation evidence so far: compact-append regression failed old code with payload-bearing state intent; now passes one-checkpoint/no-duplicate assertion. Existing legacy intent safety fixtures retained as migration tests. Rotation/deletion interruption tests caught early append recovery claiming their missing paths; transaction-specific recovery now keeps ownership, targeted tests green. Registry no-op and protocol-root fixes have targeted red/green and race evidence from bounded workers. No main publication since hub hold74141e94.

### 2026-10-15T22:54:00Z — M1 implementation checkpoint (not yet closed)

Implemented compact reservation/checkpoint appends, bounded buffered diagnostic ownership, raw-before-event capture buffering, unchanged registry publication no-op, protocol-root proof correction, and uncertain partial generation quarantine. Exact stream and witness assertions remain. The 1,000-record production-backed counter measured 44,890 payload bytes +814 metadata/reservation bytes =45,704 total (1.0181× logical), three writes, one checkpoint rename, zero hot syncs. A baseline append.go overlay fails the same test with60,917 metadata bytes and two renames. This measures logical IO, not APFS physical bytes.

Targeted wrapcmd/adapt race tests pass, including actual handleChunk VT-fragment capture and sidecar offsets; compact recovery, old payload-intent compatibility, partial-quarantine recovery and proof-fence tests pass. The artifact inventory found one vocabulary count changed by consolidation; corrected its exact count, recheck pending. M2 persisted shared limits, build, live physical measurement, full recorded suite and mandatory reviews remain. ARCH-DRY: both ordinary rotation and quarantine replay use the existing durable generation transaction; the wrapper owns and joins buffered log lifecycles instead of spawning ad-hoc writers.

### 2026-10-15T23:00:00Z — measured M1 scratch physical IO; parked-capture qualification

The isolated harmless PTY workload emitted192-byte fragments every10ms, with an outer PTY continuously drained and valid capture witnesses. Main6d65c4c4ef29bcdea3ea5b45f3dfdeed41ce1154 over10.0025s wrote21,159,936 process physical bytes for346,846 appended bytes (61.0067x), with726 observed state replacements (4354.92/min). M1affa844e84aa0f7bfd80a02027b74cd00c2aad7f over10.0036s wrote974,848 for737,192 appended bytes (1.32238x), with15 observed state replacements (89.97/min); unchanged adaptation state and registry files had zero replacements. Process physical bytes/sec fell95.39%, observed checkpoint replacements/sec97.93%. The workloads have the same emission cadence but read coalescing differs (333 versus1037 master-chunk records); this is a workload-qualified comparison, not an identical-stream assertion or the required post-landing live-slot measurement. Both exited0, stderr empty, JSONL valid. Harness /tmp/pair579-physical-measure.py; baseline evidence /private/tmp/p579-vv62773o/result.json; branch /private/tmp/p579-h1iifr5a/result.json record full build metadata, SHA256, PID, exact counters and witness IDs.

Hub requests end-to-end qualification of parked raw/events/capture-metadata expiry and notes1,242 files/5.3GB sinceSeptember23. Existing CaptureRetentionPeriod is one365-day constant; recent files intentionally remain young. The same process-inspection classification feeds diagnostic writer proof and parked-capture legacy-root evidence, so the deleted-root fix applies to both. Add portable end-to-end retention evidence without applying diagnostic size eviction to recovery captures. Keep365days unless a later operator decision changes it. Per-thread event-log growth is recorded in #582's revision.

### M1 quarantine expiry regression

Self-review caught a metadata comparison class: adding a pointer-valued unverified marker made separately decoded identical generation metadata compare unequal, blocking eventual expiry. TestQuarantinedGenerationExpiresWithSemanticMetadata reproduced diagnostic deletion metadata changed (RED0.96s). Compare complete generation evidence including marker contents by value; future count-pruning uses the same equality helper. This prevents safe preservation from becoming permanent retention.

### M1 review BR-1 — drain before the capture flush owner stops

Round1 returned REWORK for a reproduced capture lifecycle gap: masterPump stopped its flush timer on PTY EOF, while capture.Close waited beyond cmd.Wait. A child can close its terminal and remain alive. TestMasterPumpDrainsCaptureBeforeReturningOnEOF fails on the reviewed head with accepted offset12 but an empty raw sink, and independently fails missing flush-error diagnostics. A common deferred flush now covers every masterPump exit before child wait, using the same flush-and-report helper as interval ticks. This addresses family drain-before-owner-stop (ARCH-ORDER/ARCH-CONSTRAINTS); descriptors remain owned by Run. Targeted race verification follows before the next review round.

BR-1 verification: EOF-drain and error-reporting regression was RED on the reviewed head (both cases); GREEN selected capture/debug/trace/clock tests under -race in1.939s. Main-integrated full focused packages passed before the correction: diagnosticlog84.821s, wrapcmd81.923s, adapt1.585s, artifactpath14.488s. New common defer covers all pump returns, not only normal EOF.

### M1 review BR-2 — registry durability reconfirmation coverage

Round2 verified BR-1 addressed and requested the promised failure-after-rename regression (Important, family durability-reconfirmation; milestone remained unfinalized). TestRegistryRetryReconfirmsFailedPublicationDurability injects a directory-sync failure after the real declaration rename, repeats the identical failing declaration, then succeeds: three sync attempts, one content stage, exact bytes/inode/mtime unchanged. A narrow per-call directory-sync IO seam avoids global test hooks; ordinary production sync behavior is unchanged. Mutation overlay removing unchanged-entry reconfirmation is RED (retry forgot failed sync, returned nil;0.07s); real test GREEN-race1.280s, selected registry/amplification/lifecycle tests GREEN-race11.768s. No waiver.

### M2 implementation checkpoint

Shared schema-v2 policy is implemented in policy.go:64MiB current +3 immutable generations, tightening-only authority on all writers, no HardMaxBytes opt-in. Count admission reads only cap+1 bounded names; debt refuses even an already-open writer and maintenance owns full proof/certified deletion. Pure selection, migration, strict-old-reader refusal, repeated rotations, all deletion interruption stages, quarantine count/visibility, incomplete/malformed evidence and a sparse4GiB pre-existing segment are covered. Oversized legacy evidence is reported and retained until ordinary certified eviction, not claimed retroactively256MiB-bounded. M1 review closed SHIP at d804bc19a38738a13502730d228ded9ccc3545fa; both findings disposed, no waiver. Restored M2 patch, reusing M1's semantic equality helper.

Targeted M2 policy, registry/amplification, CLI subprocess and parked-runtime regressions pass under -race: diagnosticlog18.676s, diagnosticcmd2.489s, gcruntime4.617s. Baseline persisted-cap/count tests failed before implementation; removing count-debt admission also fails (n=1 instead of refusal). The subprocess command fills a persisted5B limit then refuses overflow without changing optional-command success semantics. Runtime maintenance verifies exact raw/events/capture-metadata expiry plus independent reader and handoff protection; no live data was deleted and365day policy remains unchanged. Full focused packages/build/recorded suite and M2 review follow.

M2 focused package verification GREEN: diagnosticlog69.718s, diagnosticcmd0.867s, gcruntime22.942s, artifactpath13.398s. Supervision-fact consumer checks after removing the obsolete hard-cap flag GREEN1.120s. Build and recorded full suite are next; post-landing live observer /tmp/pair579-live-observe.py records process generation and rejects invalid comparisons on payload replacement/shrink.

### M2 committed-build measurement

Clean build96b64e58ebb1d59df2f540c2e0262473eb9f664e (make -k build EXIT0; binary SHA256ce137fc4f2986aea1a9b5dffda9d5ac71d4bd301ec9aef94e9e9b69a3cfec514) passed the same isolated harmless PTY workload. PID35780 over10.0041s:835,584 physical write bytes /703,412 appended bytes =1.18790x; raw+160,190, events+0, wrap-events+543,222. Observed wrap state replacements11 (65.97/min), adaptation state0 and both registry entries0; witness valid, exit0. Evidence /private/tmp/p579-qhi__1qq/result.json. This includes shared-policy admission overhead and remains below the agreed3x scratch threshold; it is still not the required post-landing live-slot sample. Recorded full suite is running on pushed96b64e58e.

### M2 compatibility and retained expiry decision

Canonical hub decision d5c3df48-0499-420c-82d0-79aae4e926a4 keeps parked-capture expiry365days while the operator wants cloud backup first; no shortening or live capture deletion is authorized. CaptureRetentionPeriod remains the one independent capture-expiry constant. New TestDiagnosticRotationPreservesWitnessedPreviousCapture composes real producer-witness preservation/ParkScrollbackContext and ReadRetainedTerminal with repeated size/count diagnostic rotations in the same portable owner/root. Current and archived raw/events plus capture metadata retain exact IDs and bytes; previous --all selects the exact saved descriptor with identical rendered lines and no current fallback; an independent retained-peek lease survives nested reader use then releases. Selected actual compatibility/lease tests GREEN-race8.275s. No production behavior changes for this final test.

The first recorded suite on96b64e58 passed all editor/shell fixtures including review-fresh-restore and is running Go; because the planned composition test was added after launch, final recorded verification will run on its committed/pushed head.

### Recorded-suite head correction

The96b64e58 run was deliberately interrupted (exit130, not counted as a pass) once the final planned compatibility test was committed. It had passed all editor/shell fixtures and reached Go. SDLC's interrupt left its own make/test process group36357 running; sent SIGINT only to that exact owned group and verified make36357/Go432/test2134,14700,17106 had all exited. The named test processes and their owned group were confirmed gone; no other slot's run was stopped. The required final suite is queued on pushed ed75f02fc05d5f3fbe7065243a68a38a6cb72d98 under the ordinary two-slot test budget, log .git/sdlc/test-stats/inflight/99a310e5ca1372c6bb9ac65f7c59b80a/run.log. This is a source-head correction, not a retry of a failed assertion; no timeout increase, skip or waiver.

### M2 owner-stop class completion

Following BR-1 beyond its initial EOF drain, asynchronous SIGWINCH can finish an already-successful resize after masterPump stops its clock. Flush alone leaves admission open. The strengthened EOF test was RED (late resize accepted); a second production setWinsize test holds its IO result across pump EOF, then releases it. A flush-only mutation is RED with50 pending sidecar bytes. Final pump exit now closes capture-buffer admission and drains atomically before child wait, while Run retains descriptor ownership. Interval ticks still Flush. The selected capture tests pass under -race1.645s (plus prior1.792s); make -k build passes. This completes the drain-before-owner-stop class instead of only its first site (ARCH-ORDER/PURPOSE). Tests join their controlled goroutines on cleanup.

Queued ed75f02f verification was canceled before acquiring a suite slot or starting recipes (exit130); exact owned CLI/group17135 and reservation group19030 plus helper19031 were verified gone. Requeue the required suite on the fixed, committed head; no test assertion failure was retried, skipped or waived. Peer pair:7 reports its manual full suite has completed, while its required recorded run is still current; no foreign process was stopped.

### Landing hold: existing diagnostic backup decision

2026-10-15: hub request900e7813-790a-4dcb-89ec-f4239c3eaeb3 explicitly holds landing pending a read-only first-run eviction audit and an operator decision. The new policy is that potential-value data outside git must have a verified cloud backup before deletion (history-backup). Current M2 count/age pruning can evict existing diagnostic generations; do not deploy it or apply retention to live data. Audit exact files/bytes by family and distinguish current oversized preservation, first rotation, count/age expiry and proof refusals. Also document unique wrap-events information versus raw/parked captures. Required suite remains queued on136d9941b8d97fb58e5b99cee9fbd078a61eabf1; no passing full-suite claim. No live writer was restarted and no live data deleted.

### Diagnostic content and reconstructibility audit

Read-only source audit at39177e3a60b87cc0b6ea44b0ce6f6724db520396: wrap-events JSONL is not a duplicate transcript. wrapcmd.traceWrap records timestamp/sequence/agent/label and structured fields: wrapper startup argv/PID, child PID/exit/error, per-read PTY chunk length/hash/offset and ESC/BEL flags, stdout queue/filter/flush/chunk/write/error observations, stdin raw and translated length/hash, resize successes/failures, idle and automatic-input stall/report evidence. Sources: cmd/internal/wrapcmd/wrap.go traceWrap, handleChunk, forward input, flushStdout, setWinsize and child lifecycle; notification_lifecycle.go; automatic_input_report.go. Hashes do not preserve the underlying stdin text, but the historical input/translation and IO observations themselves are unique. Raw bytes can reproduce output rendering and compute aggregate hashes; they do not preserve read/write chunk boundaries, exact per-chunk timing, queue/flush outcomes, input observations or process/decision/error history. Replaying current code would be a new simulation, not reconstruction of that historical execution.

Raw scrollback is the child-output byte stream. Its separate capture .events.jsonl contains witness, resize-at-byte-offset and minute-debounced time anchors (wrap.go logScrollbackEvent/maybeLogTime and capture_witness.go); it is distinct from wrap-events-<thread-tag>.jsonl. Parked captures preserve previous wrapper raw+sidecar evidence. The thread-keyed wrap-events can span multiple launches, so one current/parked capture is not a substitute for that history. The raw stream is richer for exact terminal output; wrap-events is richer for execution diagnostics. Neither reconstructs the other fully.

Adapt logs independently record timestamp/component/agent/aspect/signal/outcome (fired, bypass, near-miss, fail) and up to200bytes of detail/snippets (cmd/internal/adapt/adapt.go). Fullscreen diagnostics record timestamp and bounded failure.Error strings from fullscreen orchestration (workbenchshortcut/fullscreen_store.go LogFailure). Their historical decisions/failures are also potentially unique and cannot be assumed reconstructible from raw captures. Classifying them as DebugRetention does not establish backup or permission to delete under the newly clarified policy.

### Read-only first-run eviction inventory

Snapshot2026-10-15T23:49:36.340694Z recursively enumerated40,524entries under ~/.local/share/pair (root and all repository scopes), stat plus regular JSON bounded1MiB; no live payload bodies, locks, writer Open, Maintain, Collect, GC/apply or process proof. Artifact /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair579-readonly-deletion-audit-dcosiu10/first-run-report.json, exact paths/stats/metadata in inventory.json and method in README.txt. First maintenance count pruning:0payload files/0bytes and0metadata files/0bytes. First due rotation pretrim:0payload/0metadata files,0bytes. First GC age/count:0diagnostic files/0bytes; parked capture age:0files/0bytes. Every family (wrap-events, adapt, fullscreen-diagnostics, couch-supervision) selects zero. There are15closed generations (maximum2/log): wrap-events13payloads2,490,973,617bytes plus13JSON3,256bytes; adapt2payloads13,999bytes plus2JSON493bytes; fullscreen/supervision0archives. All are younger365days. No durable deleting/rotation/creation intent;3append intents pending. Append recovery is not truncation permission. Some current metadata is stale (86device-only mismatches,3identity/size/clock changes), so this is policy-candidate analysis, not proof those writers can mutate. Empty candidate sets stay empty if proof succeeds.

Managed current wrap-events128files42,487,223,340bytes; unmanaged current162files12,281,903,473bytes. Oversized current logs are preserved whole on rotation. Adapt has26managed current9,885bytes plus34unmanaged32,147bytes; fullscreen1current124bytes; supervision1current8,940,013bytes. Parked captures475descriptors/1,264existing members5,751,832,868bytes allretain under365day policy. Live current sizes can advance after snapshot; exact per-file evidence is timestamped, not a future deletion authorization.

This ZERO first-run result does not resolve the backup conflict: after a log reaches3closed archives, its next rotation deletes its oldest archived payload and metadata. No verified cloud backup exists in this audit. Landing remains on hold for the hub/operator decision; current M2 has no backup fence. No live deletion or relaunch occurred.
