---
collaboration-mode: discussion
---

# Native pane-query false misses and failure evidence

> **For agentic workers:** Follow AGENTS.md Section3 and the executing-plans skill. One close boundary; no Mx milestones.

**Goal:** Recover the demonstrated Zellij0.45.1 false session-selection signature within the existing bounded read policy, and retain raw evidence whenever authoritative pane parsing still fails.

**Architecture:** Extend the existing pure read-recovery classifier, not ownership authority. Carry one typed bounded parse-failure capture through the existing error chain into the operation's existing failure row. Diagnostic IO is failure-only and cannot authorize cleanup or turn unknown into absence.

**Tech stack:** Go; existing ZellijSource/zellijexec runner, strict native decoders, operationattempt recorder and additive lifecycle failure codec.

## Evidence and approved scope

TL a9a79b15 approves only the validated false-miss signature plus typed persisted evidence; TL19a8db73 requires stdout/stderr/exit/version retention. Current independent main-based branch is000761-native-detach-owner-refusal; #744 remains parked, with no stacked source.

The three-private-session control produced two pane requests returning186bytes of complete colored session listings at exit0. Stderr said Session TARGET not found, while the stdout listing contained TARGET. Taggedv0.45.1 src/commands.rs400–442 first calls get_active_session, then repeats get_sessions for selection; a false miss invokes list_sessions(false,false,true), which exits0 before the subsequent exit1. sessions.rs assert_socket trusts a single ConnStatus reply and treats any other message as false. This is a native false selection/fallback, not a partial JSON read. The exact upstream connection-ID race is established context, not a newly instrumented upstream proof.

Retain baseline/mixed controls and their failures. Classify the separate exit1 no-active-session samples with exact-generation observations and a separately counted same-deadline diagnostic requery. They do not become a blanket retry signature.

## Core concepts

| Pure entity | Lives in | Status |
|---|---|---|
| Validated false-session-miss classifier | cmd/internal/launcher/zellij_read_recovery.go, session_reply.go | modified |
| Bounded native-query evidence and typed diagnostic error | cmd/internal/launcher/zellij_query_evidence.go | new |
| SessionOwnerObservation optional diagnostic cause | cmd/internal/launcher/session_owner.go | modified |
| LifecycleFailure optional native-query fields; clone/normalize/Upsert | cmd/internal/couchcore/launchfailure.go | modified |

One failed parse owns one immutable evidence value: argv/session, bounded stdout/stderr prefixes, original byte counts/truncation, nullable exit, version/result. Pure bounding/cloning operates on supplied bytes and observations, without subprocesses. Existing lifecycle row/history owns its retention; no new global cache, background collector or file family.

| Integration | Lives in | Status | Reuses |
|---|---|---|---|
| Read attempts and signature admission | launcher/zellij.go | modified | existing three attempts, original deadline and zellijexec.Run |
| Parse-selected capture and failure-only version query | launcher/session_owner_os.go, zellij_query_evidence.go | modified/new | existing optional query observer; actual command executable and joined runner |
| Diagnostic-cause propagation | couchcore/artifactcollision.go; launcher/couch_session.go | modified | wrapped error chain with concise unchanged text |
| Persist exact-operation evidence | couchcore/operationattempt_record.go, lifecyclefailure_codec.go | modified | errors.As, existing failure Save and unknown-field codec |
| Native diagnostic controls | workshop/evidence/000761/ plus owned scratch | new evidence | existing bare-Zellij fixture; no Couch broker/real agent |

The version probe belongs in the thin failure adapter, not the pure evidence value or successful query path. Query observers remain chained; their bounded copies never replace the full parser input.

## Recovery classifier

Retry only an allowlisted idempotent read already admitted by zellijReadKind. Require direct successful collection/exit0, stderr exactly the native Session TARGET not found / active-sessions message for the requested argv target, and a fully validated stdout listing containing that exact target as live. Recognize plain listings through the existing strict decoder and colored listings through the exact native SGR envelope: green name, magenta duration, reset, optional native current/exited suffix. Validate every row, duration, duplicate and opaque name; reject any unrelated control/escape/trailing text. Do not broadly strip ANSI or accept the listing as ownership evidence.

Only the established ErrZellijNoReply/errZellijCrossedReply loop gains this classified signature. Total attempts stay3 and share the existing logical deadline. A replacement response must still pass the real consumer parser and independent ownership agreement/generation fences. No mutation, unknown argv, collector error, arbitrary stderr, malformed JSON, absent target or exited-only target is replayed. Exit1 no-active-session remains fail-closed unless a later explicit approved revision says otherwise.

## Evidence flow, bounds and lifecycle

During SessionPanes, scope a chained observer and retain only the current attempt capture. Select it only when projectSessionOwnerReply/ParsePanes rejects the original complete bytes, not when a clipped prefix is invalid JSON. A custom SessionOwnerIO without observer data can retain supplied bytes with exit/version explicitly unavailable. Preserve Probe's Unknown,nil-error contract; attach an optional typed diagnostic cause instead of changing the authority state.

Capture stdout/stderr prefixes at most4KiB each, argv metadata at most1KiB, version at most256bytes, original counts and truncation flags, and nullable exit status. Never confuse missing observations with exit0. Retain executable identity from the actual query boundary so version probing uses the same binary, not a different PATH lookup. Probe version only after failure, through the existing joined runner and remaining original caller/proof deadline; record unavailable/canceled version explicitly. No new timeout allowance, persistent version cache or success-path subprocess.

Both formatted Unknown consumers must preserve the typed cause: NamedPairSessionContext and launcher ensureCouchConversationTerminal. Keep their current operator message; raw output stays outside Error(), notices, receipts and tab labels. lifecycleFailureForAttempt extracts the typed evidence through errors.As and populates additive top-level fields on the existing failure row. Clone byte slices/pointers; normalize writer and reader bounds; enrich only the matching operation nonce in Upsert. Existing duplicate-key/known-type checks and unrelated unknown fields round-trip unchanged. Do not replace existing launch output. Keep64-row/1MiB whole-entry retention; no new archive.

Actions remain inspect failure evidence and rerun after resolving the native failure.

| Internal outcome | Visible state | Recovery | Possible loss |
|---|---|---|---|
| Validated false miss; later read proves ownership | operation continues | inspect retained attempt diagnostics if needed | transient failed attempt, never counted as authority |
| Signature exhausts3 attempts, parse remains invalid or generation changes | same concise refused/unknown operation | inspect failure evidence | prefixes beyond caps are explicitly truncated |
| Version query unavailable or canceled | original failure with version-unavailable evidence | inspect failure evidence | version observation only |
| Failure persistence fails | original refusal plus existing save warning | inspect storage failure | optional diagnostic record; ownership remains unknown |

Successful reads create no new retained state. Capture/error values die with operation lifetime. Only existing bounded lifecycle rows remain after failure (ARCH-FUNERAL). The classifier never authorizes a destructive effect (ARCH-SECURE/ORDER).

## Tests and tasks — one boundary

- [ ] Preserve a minimal native reproduction sequence, installed executable version/checksum, rate, exact bad stdout/stderr and all failed samples. Prove private generation cleanup and report the exit1-control gap honestly.
- [ ] TDD the pure classifier with the captured186byte fallback. Stateful query frames: false miss→valid response, three false misses→refusal, late deadline, target absent/exited, wrong stderr/argv, malformed or arbitrary escape bytes, collector errors and mutation negative controls. Observe RED before admitting only the exact signature.
- [ ] TDD parse-selected typed evidence: malformed native bytes with stderr/exit; valid JSON larger than observer cap must not falsely capture; observer chaining; Unknown,nil semantics; cause preserved through both consumers; successful reads execute zero version commands, failure at most one. Exercise actual fake executable/counters, not only callback mocks.
- [ ] TDD persistence through real OperationAttempt→Save→reopen: exact raw bytes/counts/nullable exit/version, concise UI, cloning, truncation, matching-nonce enrichment, unrelated launch output, known-field refusal and unknown-field round trips. Count retained row/byte bounds.
- [ ] Implement minimal helpers and adapters with existing runner/codec ownership; audit artifact/source-reader inventories. Run go test ./cmd/internal/launcher ./cmd/internal/couchcore ./cmd/internal/artifactpath ./internal/repolint -count=1 plus relevant races and make test-affected-inventory.
- [ ] Run the captured mixed native schedule through the fixed production boundary. Count every attempt and final parse; never erase failures with a blind test rerun. Recheck actual production owner Probe/Revalidate and detach/restart with exact cleanup using main's existing private native fixtures. #744's full matrix waits until this issue lands.
- [ ] Update atlas/couch.md and atlas/index.md for failure-evidence location and workaround removal condition. Push final head, required recorded suite, mandatory full close review, PR and ordinary merge. Report landing; only then integrate landed main into #744 and rerun its matrix.

## Growth and trust

At most3 query attempts per logical allowlisted read; no additional success-path commands. At most one version query and one bounded evidence value per failed pane proof. Add counted tests for zero/one version calls,3-attempt exhaustion and history at N/2N. CPU cost is linear in supplied bounded bytes, with one native-listing pass; no per-row requery. Concurrent operations own independent captures. Native bytes are untrusted; raw evidence is never used as a command or ownership capability. Existing stateful query fake and real private native controls share the actual invocation/decoder seam (ARCH-MOCK/DRY).

## Approval and estimate

TL approved the narrow fix and evidence requirement; send this concrete full plan for confirmation before product implementation. Re-run change-code so plan-quality checks the expanded flow, then derive the estimate from the calibrated method; do not invent actual hours. Any broader retry signature requires a dated Revision and TL approval.
