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

The second admitted signature is direct exit1 with exactly empty stdout and stderr There is no active session!, only for the same allowlisted session-specific read and only while its exact socket is observed present. Reuse ResolveZellijSocketRoot and the existing contract_version_1 path spelling; validate the session basename and use ObserveSocket(os.Lstat), never a host scan or another native query. Unknown/non-socket/gone observations refuse.

Only the established ErrZellijNoReply/errZellijCrossedReply loop gains these classified signatures. Total attempts stay3 and share the existing logical deadline. A replacement response must still pass the real consumer parser and independent ownership agreement/generation fences. No mutation, unknown argv, collector error, arbitrary stderr, malformed JSON, absent target or exited-only target is replayed. A genuinely gone session remains fail-closed. Socket presence permits only another read; it is never ownership or absence evidence.

## Evidence flow, bounds and lifecycle

During SessionPanes, scope a chained observer and retain only the current attempt capture. Select it only when projectSessionOwnerReply/ParsePanes rejects the original complete bytes, not when a clipped prefix is invalid JSON. A custom SessionOwnerIO without observer data can retain supplied bytes with exit/version explicitly unavailable. Preserve Probe's Unknown,nil-error contract; attach an optional typed diagnostic cause instead of changing the authority state.

Capture stdout/stderr prefixes at most4KiB each, argv metadata at most1KiB, version at most256bytes, original counts and truncation flags, and nullable exit status. Never confuse missing observations with exit0. Retain executable identity from the actual query boundary so version probing uses the same binary, not a different PATH lookup. Probe version only after failure, through the existing joined runner and remaining original caller/proof deadline; record unavailable/canceled version explicitly. No new timeout allowance, persistent version cache or success-path subprocess.

Both formatted Unknown consumers must preserve the typed cause: NamedPairSessionContext and launcher verifyCouchSessionCreate. Keep their current operator message; raw output stays outside Error(), notices, receipts and tab labels. lifecycleFailureForAttempt extracts the typed evidence through errors.As and populates additive top-level fields on the existing failure row. Clone byte slices/pointers; normalize writer and reader bounds; enrich only the matching operation nonce in Upsert. Existing duplicate-key/known-type checks and unrelated unknown fields round-trip unchanged. Do not replace existing launch output. Keep64-row/1MiB whole-entry retention; no new archive.

Actions remain inspect failure evidence and rerun after resolving the native failure.

| Internal outcome | Visible state | Recovery | Possible loss |
|---|---|---|---|
| Validated false miss; later read proves ownership | operation continues | inspect retained attempt diagnostics if needed | transient failed attempt, never counted as authority |
| Signature exhausts3 attempts, parse remains invalid or generation changes | same concise refused/unknown operation | inspect failure evidence | prefixes beyond caps are explicitly truncated |
| Version query unavailable or canceled | original failure with version-unavailable evidence | inspect failure evidence | version observation only |
| Failure persistence fails | original refusal plus existing save warning | inspect storage failure | optional diagnostic record; ownership remains unknown |

Successful reads create no new retained state. Capture/error values die with operation lifetime. Only existing bounded lifecycle rows remain after failure (ARCH-FUNERAL). The classifier never authorizes a destructive effect (ARCH-SECURE/ORDER).

## Tests and tasks — one boundary

- [x] Preserve a minimal native reproduction sequence, installed executable version/checksum, rate, exact bad stdout/stderr and all failed samples. Prove private generation cleanup and report the exit1-control gap honestly.
- [x] Write failing tests for each named risky function in the strategy table below, then implement the least change that makes its independent invariant pass. Keep reproduction bytes as seeds and mutation controls for retry admission.
- [x] Implement minimal helpers and adapters with existing runner/codec ownership; audit artifact/source-reader inventories. Run go test ./cmd/internal/launcher ./cmd/internal/couchcore ./cmd/internal/artifactpath ./internal/repolint -count=1 plus relevant races and make test-affected-inventory.
- [x] Run the captured mixed native schedule through the fixed production boundary. Count every attempt and final parse; never erase failures with a blind test rerun. Recheck actual production owner Probe/Revalidate and detach/restart with exact cleanup using main's existing private native fixtures. #744's full matrix waits until this issue lands.
- [ ] Update atlas/couch.md and atlas/index.md for failure-evidence location and workaround removal condition. Push final head, required recorded suite, mandatory full close review, PR and ordinary merge. Report landing; only then integrate landed main into #744 and rerun its matrix.

## Function-specific test strategies

These names define the implementation seams; tests qualify the actual function rather than a separate model.

| Risky function | Adversarial input class and mechanical guard |
|---|---|
| retryableSessionLookupMiss(args, stdout, stderr, exit, socketPresent) | Generated combinations of reply kind, exact/mutated native signature and socket fact; independent allowlist oracle, fuzz seeds from both real captures, and actual invocation counters reject any mutation/unknown/nonmatching retry. |
| parseColoredSessionListing(raw) | Arbitrary bytes, malformed/trailing ANSI and metadata/name ambiguity; whole-input parser plus fuzzing accepts only the exact supported native envelope and preserves opaque identities. |
| sessionSocketPresent(name) | Unsafe basenames and missing/regular/symlink/unknown files in a private stateful socket folder; require SocketPresent from exact derived native namespace, with zero host scans or process calls. |
| NativeQueryEvidence.Bounded / boundNativeQueryEvidence(value) | Generated oversize/malformed byte streams and nullable exit metadata; prefix/count/truncation invariants and independently mutated input/output slices prove hard limits and no aliases. |
| SessionOwnerProbe.probe | Controlled full replies and observer sequences, including replies larger than the observer cap; assert existing authority states and Unknown,nil semantics, select evidence only on the full parser result and preserve chained observations. |
| queryZellijVersion(ctx, executable) | Stateful executable fixture records calls and held child lifetime; independent counters require zero success-path probes and at most one failure-path probe, with the original deadline and joined cancellation. |
| lifecycleFailureForAttempt | Wrapped diagnostic error chains and distinct operation identities through actual recorder/Save/reopen; compare original byte/exit/version evidence and independently assert concise operator text and unchanged unrelated launch output. |
| cloneLaunchFailure / normalizeLaunchFailure / LaunchFailureHistory.Upsert | Generated evidence sizes, nullable fields and repeated/mismatched nonces; independent no-alias/bound/retention invariants at N/2N and exact typed-field enrichment. |
| encodeLifecycleFailureRow / decodeLifecycleFailureHistory | Round trips with nested existing unknown members, duplicate/ill-typed known fields and new optional top-level evidence; preserve unrelated bytes while strict malformed-known refusals and total encoded limits remain enforced. |

## Growth and trust

At most3 query attempts per logical allowlisted read; no additional success-path commands. At most one version query and one bounded evidence value per failed pane proof. Add counted tests for zero/one version calls,3-attempt exhaustion and history at N/2N. CPU cost is linear in supplied bounded bytes, with one native-listing pass; no per-row requery. Concurrent operations own independent captures. Native bytes are untrusted; raw evidence is never used as a command or ownership capability. Existing stateful query fake and real private native controls share the actual invocation/decoder seam (ARCH-MOCK/DRY).

## Revisions

### 2026-10-21 — TL3aacb9e7 approves socket-present exit1 recovery

The original plan kept exit1 no-active-session failures fail-closed pending classification. The focused1000-read control established24/1000 live false negatives; all24 retained the same exact server generation/socket before and after a successful diagnostic requery within the original deadline. Verified TL3aacb9e7 approves the exact exit1/empty-stdout/native-stderr signature only when the target socket exists, inside the unchanged3-attempt/deadline bound. Add positive/negative regressions and a1000-read counter run reporting every attempt and residual final failure. The current classifier section reflects this revision; no blanket nonzero retry is permitted.

## Approval and estimate

TL approved the narrow fix and evidence requirement; send this concrete full plan for confirmation before product implementation. Re-run change-code so plan-quality checks the expanded flow, then derive the estimate from the calibrated method; do not invent actual hours. Any broader retry signature requires a dated Revision and TL approval.


### 2026-10-21 — PQ-1/PQ-2: executable function strategies and actual consumer name

The initial plan listed test cases without assigning each risky helper a function-level adversarial strategy. The new table names every classifier/capture/IO/persistence seam with its input class and mechanical oracle; the task now executes those strategies. The launcher consumer is verifyCouchSessionCreate at couch_session.go195/238, not the prior nonexistent ensureCouchConversationTerminal name. Scope, approved retry signatures and authority semantics are unchanged.


### 2026-10-21 — implementation naming and cancellation propagation

BoundNativeQueryEvidence is implemented as the private boundNativeQueryEvidence behind NativeQueryEvidence.Bounded, which clips and clones in one bounded pass; a separate cloneNativeQueryEvidence would duplicate that traversal. Typed-cause propagation also covers SessionOwnerPass and the existing post-observation context fences, preserving cancellation identity without another version query. This is the same error-chain class and does not alter authority or recovery policy.

### 2026-10-21 — BR-1/BR-2: preserve native spelling and budget test sockets

The exact-socket condition includes symlink/parent traversal semantics: append the validated native version/name components without filepath.Join cleaning. Add positive and negative filesystem regressions through the real read retry boundary. The test fixture must choose a short owned root independently of inherited TMPDIR, prefer TestScratch when it fits, and retain strict cleanup; run with an intentionally long TMPDIR. This implements the existing exact-target condition and portable test contract, with unchanged public behavior and retry budget. Requeue the suite on the fixed code, rerun native counters there and re-close against both ledger findings.

### 2026-10-21 — BR-1 source-complete root spelling

Explicit, runtime and temporary roots must all preserve native symlink/parent-segment and separator spelling before the socket condition. Reuse one pure component append in all applicable constructors and the final version/name path; keep root/name validation unchanged. For each source, run positive and negative filesystem arrangements through actual retry admission and require exact command counts. Pure platform-parametric tables exercise Linux and Darwin selection and reject prior normalization. Earlier explicit-only coverage is insufficient. BR-2 remains addressed; no timeout, attempt, public state or failure-evidence contract changes.
