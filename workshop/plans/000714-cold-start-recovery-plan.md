---
collaboration-mode: discussion
---

# Cold-start recovery implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development for bounded work or superpowers-executing-plans for session-warm work. Steps use checkbox syntax.

**Goal:** Make failed launches retryable when failure is proved, retain their diagnostics, and offer honest recovery choices when screen evidence or native binding is unavailable.

**Architecture:** Keep the existing per-thread operation hold and nonce/revision CAS as the launch owner. Add a terminal failure disposition to the pure start transition, independent of optional diagnostic publication. Separate screen availability from native candidate availability; an explicitly selected native-only snapshot remains inferred evidence and never becomes an established binding.

**Tech Stack:** Go, Couch thread records, existing Runner/Proc/Artifacts stateful fakes, native transcript runtime, bounded file IO, existing recovery authority profile.

## Status and scope

Proposed for TL approval before code. M1 addresses the incident recurring in pair:5 first. M2/M3 complete the existing issue's candidate acceptance criteria. The native-only evidence variant and its five-minute selection lifetime are explicit policy proposals for TL/operator approval.

Covered: acknowledged recovery launch exit, launch diagnostics, retry, actual capture-generation mismatch, scan completeness, unbound interactive Codex roots, CLI/menu parity and final launcher validation. Not covered: fleet-wide orchestration (ariadne#456), repairing historical Pair logs, automatic inference/binding, changing capture retention or native scan limits, CI migration (#699), and a speculative supervision-log fix without a demonstrated defect. Investigate the supervision note read-only and record whether it is related.

## Evidence and alternatives

- `launch_existing.go::unconfirmed` publishes failure without a disposition even when its owned helper has exited. Keeping every outcome occupied is safe but strands a definite failure behind the five-minute stale-start policy. Calling ordinary post-ack cleanup unconditionally would instead destroy a possibly successful recovery session. Use an evidence-based terminal disposition.
- `operationattempt_record.go::lifecycleFailureForAttempt` replaces a captured complete address when the outer call supplies only repo-scope. `ObserveCall` also accepts partial addresses. The strict failure codec correctly rejects the resulting empty tag. Fix evidence composition, not codec strictness.
- Old pair:3 capture witness: device 16777232, raw inode 731744298, events inode 731744297. Current stat: device 16777230, same two inode numbers. This establishes device mismatch, not identity continuity. Do not ignore device/inode checks or rewrite historical witnesses to current stat.
- The same raw capture is 381260612 bytes, larger than the existing 256 MiB scan budget. Raising budgets or relaxing witness checks would not address the coupling between native discovery and screen availability.
- Established binding is absent because the old ledger has a launch but no binding and the Pair log is missing. Existing inference already supports unbound roots with a valid screen. The needed extension is independent native evidence when the screen is unusable; cwd/time must remain visibly weaker than causal turn correlation.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| Recovery launch disposition | `cmd/internal/couchcore/recovery_launch_disposition.go` | new |
| StartEvent / AdvanceStartTransaction | `cmd/internal/couchcore/starttransaction.go` | modified |
| Lifecycle failure target composition | `cmd/internal/couchcore/operationattempt_record.go` | modified |
| RecoveryContext native-only source | `cmd/internal/threadrecord/recovery.go` | modified |
| RecoveryObservation / RecoveryPreview projection | `cmd/internal/couchcore/recovery_preview.go` | modified |

The disposition consumes typed helper/session/registration observations and returns promote, release-failed, or retain-unconfirmed. It performs no probing or storage. StartEvent remains the only owner of creating-incarnation transitions. A release event retains the exact start's selection/context, park and thread, removes only that creating incarnation, and invalidates all later events for that nonce.

Failure target composition uses complete addresses atomically. A partial outer scope never replaces a complete captured address or combines with a tag from another address. A different complete outer target remains the operation target; the captured launch nonce still correlates its child. Scope-only failures use the unresolved reference/path evidence and an empty Address, rather than an invalid half-address. Diagnostic target evidence never grants lifecycle authority.

RecoveryContext v1 remains the current capture-backed form. A new v2 native-only source carries observation cutoff, optional original launch hint, and the existing ordered native artifact fingerprints. No fabricated RecoveryCapture is supplied. The source hint improves presentation/filtering only when attributable to this launch; missing hint is displayed and does not become an invented epoch. Candidates remain explicit, bounded and revision/digest bound. No new RecoveryMode or incarnation state string is required.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Recovery failure finalizer | `cmd/internal/couchcore/launch_existing.go` | modified | helper handle, exact terminal observation, bounded ThreadStore CAS |
| LifecycleFailureRecorder | `cmd/internal/couchcore/operationattempt_record.go` | modified | existing failure store and one terminal publication |
| OSRecoveryEvidenceReader | `cmd/internal/couchcore/recovery_evidence_os.go` | modified | screen metadata, native runtime, selected-root leases |
| Native candidate collector | `cmd/internal/sessioninventory/recovery_collect.go` | modified | bounded RecoveryRuntime enumeration/read/stat |
| Prepared recovery executor | `cmd/internal/couchcore/recovery_prepared.go` | modified | selection persistence, retirement, workspace and launch |
| Recovery authority validator | `cmd/internal/launcher/recovery_authority_os.go` | modified | independent final native fingerprint/cwd/provenance validation |

Reuse Runner/Proc/Artifacts fakes for process identity, exit, session presence and revision races; use the existing file-backed record/failure stores for restart tests. Reuse the stateful RecoveryRuntime fake for transcripts, counts, mutations and path resolution. Live conformance uses a disposable slot only when staged and approved by the hub; no tests touch real native home or current slots.

## Actions and visible state

Actions: inspect exact attempt; retry resume; refresh candidates and explicitly select; attach surviving terminal; choose Start fresh.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Active owned start; helper live; no registration yet | Starting | Inspect exact attempt | None; no duplicate launch |
| Final exact registration established despite helper error/timeout | Running or detached | Attach surviving terminal | No conversation loss |
| Helper exit confirmed, exact proposed terminal absent, no registration | Launch failed; retry available | Retry resume | Failed helper output only if diagnostic storage itself fails |
| Helper dead but terminal present and registration not established | Launch unconfirmed | Inspect exact attempt | No automatic cleanup or new launch |
| Helper/session probe unknown, cancelled observation, or deadline | Launch unconfirmed | Inspect exact attempt | No automatic cleanup or new launch |
| Evidence proves failure but CAS loses revision/nonce or persistence times out | Disposition unconfirmed; claim retained | Inspect exact attempt | No authority discarded |
| Optional diagnostics cannot be saved | Same lifecycle state plus save-failed notice | Inspect exact attempt | Diagnostic bytes may be unavailable; failure/result unchanged |
| Screen mismatch/missing/oversize, native scan complete | Explicit native candidates; screen unavailable with exact reason | Refresh candidates and explicitly select | Previous screen unavailable; no implied binding |
| Several same-cwd candidates | Explicit alternatives, none automatic | Refresh candidates and explicitly select | Wrong user selection can resume a different conversation; label native IDs/activity |
| Native scan incomplete or source changed | Candidates unavailable; specific reason | Refresh candidates and explicitly select | None until user chooses Fresh |
| Proven native binding but screen unavailable | Confirmed resume available; screen warning | Retry resume | Previous screen unavailable only |
| No eligible native root | No conversation found | Choose Start fresh | Unpersisted context may be lost; say so |

## Ordering and invariants

ARCH-ORDER: the existing per-thread operation hold spans final observation and disposition; the exact start nonce/revision fences metadata changes. This is the resource preventing a concurrent launch, rather than absence inferred from a scan. A helper must have completed its owned exit/Wait before confirmed release. Its exact proposed terminal is then observed under a fresh bounded context. Timeout or failed observation never means absent. No automatic signal, terminal deletion or second launch is introduced.

State/event table for an acknowledged recovery claim:

| Event/evidence | Transition | Effects |
|---|---|---|
| Exact registration established | Existing StartRegistered | Adopt existing attempt once |
| Owned helper completed + exact terminal absent + no established registration | New terminal failure event | Retain selection, clear exact creating incarnation via CAS, capture failure |
| Any live/unknown helper or present/unknown terminal | Retain same claim | Capture unconfirmed outcome; no cleanup |
| Cancellation before ack | Existing pre-ack cleanup | Reap blocked helper and rollback exact claim |
| Cancellation/timeout after ack | Final bounded observation, then table above | Preserve partial progress |
| Concurrent revision/nonce change or late registration | Refuse stale release | Preserve newer state; no retry effects |
| Duplicate terminal disposition | Reject old nonce or return already completed result | No second launch/publication |
| Supervisor crash during finalization | Existing occupied claim survives | Existing stale-start recovery remains conservative |

Native-only selection uses the existing inferred route: list read-only -> explicit key -> re-observe complete snapshot -> persist selection before retirement -> confirm exact source absent -> claim -> revalidate selected native artifact -> launcher independently revalidates -> native resume. Screen failure neither authorizes retirement nor hides a proven binding. Invalid/mutated transcript evidence blocks effects. Native-only selection never writes a binding ledger row and never receives a fresh-start prompt.

## Operating envelope and persisted formats

- ARCH-CONSTRAINTS: failure finalization reuses the existing five-second recovery observation budget and a separate bounded persistence budget; neither diagnostics nor lock acquisition may extend it indefinitely. No cost on successful registration beyond existing finalization. Count observer/store calls for failure sequences; contend real store locks to prove bounded return.
- Native scans retain 30 seconds, 10000 metadata entries, 64 eligible roots and 256 MiB total actual reads. Screen failure uses a native-only lane in the same invocation/budget; do not run two full scans or silently repeat deterministic failures. New native-only snapshot lifetime: five minutes from observation (proposal), revalidated before every effect and at Pair exec.
- Growth drivers: N native metadata entries, B eligible recent transcript bytes, C eligible roots. One metadata pass O(N), one bounded streaming parse O(B), ranking O(C log C), five projected choices. Test N=8/16 and B/2B with the existing counted runtime; visits/reads may at most double plus fixed overhead. Selection revalidation stats/reads the chosen artifact, not the whole native tree. No per-candidate rescan.
- ARCH-SECURE: persisted witnesses, records, native JSONL and paths are untrusted. Keep strict typed decoding, regular-file/root/cwd checks, full interactive provenance and exact fingerprints. Failure text uses existing sanitization and output bounds. No credential or transcript content enters the issue Log.
- ARCH-EVOLVE: v1 context bytes/meaning remain unchanged; v2 is necessary because old readers must refuse a native-only intent rather than treat an empty capture as permission. Reuse existing unknown-field preservation for record/context/candidate rewrites, extend for new nested source fields, test duplicates/types/truncation and v1 round trips. The new start event is transient; stored disposition uses existing no-incarnation/retained-choice representation.
- ARCH-FUNERAL: no new file family, process or append log. Failure rows use the existing bounded diagnostic store; native-only snapshots occupy the existing single retained context/intent and are superseded through existing start/retirement lifecycle. At expiry they remain readable evidence but confer no launch authority.
- ARCH-DRY / ARCH-PURE: reuse existing transition/store, recorder, collector and authority owners; source-kind switches are typed and exhaustive. New files isolate disposition/projection policy without creating a parallel recovery pipeline.
- ARCH-MOCK: real file-backed stores plus existing stateful native/process fakes cover ordering; staged disposable live check qualifies the same production seams.
- ARCH-PURPOSE: all three Done-when rows remain in scope; fix retry first, then make candidate selection independent of missing screen/binding evidence. A fleet view remains out of scope.

## M1 — Definite launch failure and durable diagnostics

Files: `couchcore/{recovery_launch_disposition.go,starttransaction.go,launch_existing.go,operationattempt_record.go,lifecyclefailure_record.go}` and colocated tests; `couchcmd/lifecyclefailure_service_test.go`; `atlas/couch.md`.

- [ ] Add failing production-dispatch regression with repo-scope but no tag: recovery helper exits before registering, exact terminal absent, complete captured launch identity and output reach failure store exactly once; a second resume starts immediately without advancing the clock.
- [ ] Add a pure disposition table and pure start-event sequences: exact selection/park retained, input immutable, stale revision/nonce rejected, late registration cannot be cleared. Preserve current helper-exited uncertainty tests when terminal observation fails.
- [ ] Run `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd -run 'RecoveryLaunch|LifecycleFailure|OperationAttempt|StartTransaction' -count=1`; confirm the added regressions fail for the intended wedge/identity reason.
- [ ] Implement the terminal disposition. Observe using exact proposed session binding under the current operation hold, use bounded context-aware metadata reads/writes, then apply one nonce/revision event. A completed helper plus absent terminal releases only this claim; all uncertain cases retain it. Keep ordinary warm/spawn cleanup unchanged.
- [ ] Centralize complete-address diagnostic composition at both ObserveCall and terminal enrichment. Sweep every partial-address assignment in that family. Preserve exact launch address when outer evidence is partial; unresolved refusals remain persistable by path/reference. Do not weaken normalizeLaunchFailure.
- [ ] Test live owner with failed helper, unknown/live terminal, failed probe, canceled wait, late registration, CAS conflict, held store lock, diagnostic store failure, duplicate Finish and successful operation (zero diagnostic IO). Include integration assertions through CLI dispatcher and restarted file-backed store.
- [ ] Add a Pair-local lint rule for the mechanical partial-address override family using the existing local catalog; test the safe centralized composition and bad whole-address nonzero guards. Restrict rule to actual diagnostic composition owners to avoid generic ThreadAddress false positives.
- [ ] Run focused tests plus race checks, update atlas/lessons and Log, commit with Agent/Co-Authored-By trailers, then `sdlc milestone-close --issue 714 --milestone M1`.

## M2 — Independent screen/native evidence and explicit unbound choices

Files: `threadrecord/recovery.go` and context codec tests; `sessioninventory/recovery_collect.go` and tests; `couchcore/{recovery_evidence_os.go,recovery_preview.go,recovery_prepared.go}` plus new focused `recovery_native_only_test.go`; existing CLI/menu presentation and tests where source labels appear; atlas session identity/couch pages.

- [ ] Add failing fixtures: old device IDs with same inodes; absent/invalid witness; oversized screen; no Pair log/binding with an eligible Codex rollout; multiple same-cwd roots; proven binding plus failed screen; incomplete native scan. Assert last activity is explicitly sourced, native incompleteness is separate, and no listing provisions/binds/writes lifecycle state.
- [ ] Model typed screen availability separately from native scan completeness. Retain partial descriptive evidence when a read fails; report exact capture issue rather than generic conversation unavailable. A file mtime is labelled capture-file activity, never causal conversation activity.
- [ ] Add v2 native-only context source and validation/round-trip tests. Carry a fixed observation cutoff, source revision/digest, existing candidate fingerprints, optional attributable launch hint, five-minute expiry, and no capture. Reject mixed native-only/capture fields and invalid future times. V1 remains byte/meaning compatible.
- [ ] Extend collector request with explicit evidence-source kind. Native-only collection uses observation cutoff and optional launch hint, same root/cwd/interactive checks, no tail score and no assertion of a producing capture. Missing launch hint is honest weaker evidence, not an incomplete native scan. Reuse one metadata/read pass and existing stateful fake.
- [ ] Project no more than five explicit choices labelled unconfirmed native candidates with native ID/activity; `Automatic=false`, `Proven=nil` without established binding, no fabricated tail percentage. Plain resume still requires proven binding; candidate-key resume is the explicit choice. Candidate keys bind the source kind, cutoff, revision and exact ordered evidence; expiry is enforced at selection.
- [ ] Exercise changed native content/identity/cwd, duplicate ID, child/noninteractive session, malicious paths, unknown fields and stale keys. Compare listing/selection/menu JSON and text. Add N/2N visits/read counts and exact per-request byte-budget tests.
- [ ] Run `go test ./cmd/internal/threadrecord ./cmd/internal/sessioninventory ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/couchtty -run 'Recovery|ResumeCandidates' -count=1`; log baseline mutation controls, update atlas and commit, then `sdlc milestone-close --issue 714 --milestone M2`.

## M3 — Final authority, restart conformance and landing

Files: `launcher/{recovery_authority_os.go,recovery_authority.go}` and tests; `couchcore/recovery_prepared.go` and end-to-end fixtures; artifactpath source inventory if new production files; atlas/index.md only for new atlas pages.

- [ ] Add failing native-only selection-to-exec fixture, interrupted launch/retry fixture, and independent launcher validation cases. Read/write/reopen records across each boundary using portable fake state. A copied selection or changed root must fail before native exec.
- [ ] Branch inferred preservation and final validation by typed source: capture-backed keeps existing archive/identity requirements; native-only persists its bounded native snapshot before retirement, revalidates chosen artifact/cwd/interactive provenance and expiry before claim/exec, and supplies unavailable-screen orientation only. Preserve existing exact stop-set confirmation and conversation-in-use checks for both.
- [ ] Confirm no ledger binding is manufactured, native argv has the selected resume ID, no prompt is injected, confirmed-binding resume remains available with no screen, Fresh remains available after genuine evidence failure, and late helper registration survives the M1 disposition race.
- [ ] Run `go test -race ./cmd/internal/couchcore ./cmd/internal/launcher ./cmd/internal/sessioninventory -run 'Recovery|Resume' -count=1` and `go test ./internal/repolint ./cmd/internal/artifactpath -count=1`. Run existing detached/warm/cold resume suites and all source/codec audits for new files.
- [ ] Ask hub for staged disposable live qualification using `sdlc stage --build`; verify emitted revisions, early-launch failure diagnostics and immediate retry, and candidate listing/explicit selection on the new binary. Hub owns real-slot actions; clear stage afterward unless landing clears it. Never reproduce by rebooting an active working slot.
- [ ] Update Log with measured failure-finalization bounds, scan counts, evidence classifications and remaining historical gaps. Commit/push final branch, run `sdlc local-check run --name suite`, and close M3/issue through normal gates. On SHIP run `sdlc merge --yes`, report verified PR/landing and stop.
