---
collaboration-mode: discussion
---

# Native-only recovery candidates implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development for bounded independent tasks or superpowers-executing-plans for session-warm implementation. Steps use checkbox syntax for tracking.

**Goal:** Offer explicitly unconfirmed native transcript alternatives when the saved screen cannot support inference, preserving exact ownership and final launch validation.

**Architecture:** Separate screen availability from native inventory completeness. Keep capture-backed context v1; introduce native-only context v2 with a fixed observation cutoff, source revision, exact ordered evidence digest and five-minute selection lifetime. Carry that source through selection, preservation and launcher validation without manufacturing a transcript binding or capture.

**Tech Stack:** Go, existing threadrecord codecs, sessioninventory RecoveryRuntime, Couch recovery transitions and launcher authority; portable filesystem/stateful runtime fixtures.

## Current contracts and scope

This adapts the approved original #714 M2/M3 scope now owned by #721. #714 landed in PR #471; do not redo its failed-launch disposition. #695 also landed: schema 4 records preserve unknown top-level fields, TranscriptBinding preserves its own unknown members, but recovery nested objects still use strict decoding. TestRecoverySchemaRejectsUnknownNestedFields is concrete evidence; do not assume record-level tolerance reaches those objects.

Current OSRecoveryEvidenceReader.Read returns before native discovery on capture errors. ResumeCandidates and recoveryObservation then discard partial observations. DecideSlotRecovery requires a valid capture, and executePreparedRecovery rebuilds keys after preservation. ValidateRecoveryRecord and OSRuntime.ValidateRecoveryAuthority always require a retained archive for inferred selections. These are the owners to change, rather than adding another recovery route (ARCH-DRY).

One atomic implementation and one issue-close review boundary: a native-only preview without final launcher support would advertise an unusable action. The original milestone commands in the issue Spec are historical. Tasks below are commit checkpoints, not separate Mx boundaries.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| RecoverySourceKind / NativeRecoverySource | cmd/internal/threadrecord/recovery.go | new |
| RecoveryContext / RecoveryCandidate / recovery validation and clones | cmd/internal/threadrecord/recovery.go | modified |
| Recovery source codec | cmd/internal/threadrecord/recovery_source_json.go | new |
| RecoveryCollectRequest / native ordering | cmd/internal/sessioninventory/recovery_collect.go and recovery_match.go | modified |
| RecoveryObservationRequest / ScreenAvailability / activity provenance | cmd/internal/couchcore/recovery_preview.go | new |
| RecoveryObservation / RecoveryPreview / RecoveryChoice / DecideSlotRecovery | cmd/internal/couchcore/recovery_preview.go | modified |
| RecoveryCandidateSelector | cmd/internal/couchcore/recovery_key.go | modified |
| ValidateRecoveryRecord | cmd/internal/launcher/recovery_authority_os.go | modified |

NativeRecoverySource is owned 1:1 by a v2 context, with source kind native-only, fixed UTC cutoff, nonzero source record revision and optional attributable launch hint. A context owns at most five exact candidate snapshots. Candidate fingerprints reuse existing root, relative path, stable file ID, generation/mutation tokens, size and mtime; add physical CWD to the persisted native-only evidence so a later cwd remap cannot be hidden by display formatting. Native timestamps come from semantic native messages; file mtime is separately labelled capture-file activity. Never infer causal conversation activity from a copy/publication clock.

ScreenAvailability describes available, missing, invalid, over-budget or expired screen evidence with a bounded reason and any independently readable activity. Native completeness describes only the native scan. Neither is launch authority. Incomplete native observations retain diagnostics and already observed facts but offer no unbound actionable candidate keys. Proven and Fresh remain independently available.

The selector remains an untrusted reference. V1 fields/wire stay unchanged. V2 adds source kind, cutoff, source revision and expiry so the next request can reproduce the same evidence window. A source digest binds the address, source kind, revision, cutoff, optional launch hint and complete ordered candidate fingerprint/activity evidence; candidate keys and the digest itself are excluded to avoid a cycle. CreatedAt equals the native cutoff and ExpiresAt equals cutoff + five minutes; a rescan never renews either. New listings can produce new keys. Bounded caller-supplied selector fields do not confer provenance: the server reconstructs all evidence, compares the digest and checks the current source revision before any effect.

Native-only ordering is deterministic semantic last activity descending, then agent/native ID/artifact path. It does not invoke tail scoring or label an option best by screen match. Capture-backed ranking remains unchanged. RecoveryChoice exposes source kind and CWD; native-only JSON omits tail-match/count fields rather than fabricating zero-percent evidence. Preserve existing capture-backed JSON fields including meaningful zero scores through source-aware encoding.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| RecoveryEvidenceReader / OSRecoveryEvidenceReader.Read | cmd/internal/couchcore/recovery_preview.go and recovery_evidence_os.go | modified | Capture files, leases, native roots, physical cwd and bounded Git lookup |
| CollectRecoveryEvidence / RevalidateRecoveryCandidate | cmd/internal/sessioninventory/recovery_collect.go | modified | Existing bounded RecoveryRuntime |
| executePreparedRecovery / launchRecoveryClaim | cmd/internal/couchcore/recovery_prepared.go | modified | Confirmation, durable intent CAS, retirement and start claim |
| OSRuntime.ValidateRecoveryAuthority | cmd/internal/launcher/recovery_authority_os.go | modified | Durable record and exact native artifact immediately before exec |
| CLI/menu projection | cmd/internal/couchcmd/resume_candidates.go and cmd/internal/couchtty/menu_recovery.go | modified | Authenticated CLI/JSON and terminal rendering |
| Production evidence wiring | cmd/internal/couchcmd/run.go | modified | Injected observation request and clock |

Reuse the stateful RecoveryRuntime fake in recovery_collect_test.go: metadata inventory, contents, mutation/identity tokens, aliases, injected read/stat failures and operation counters. Reuse recoveryEvidenceFixture and newRecoveryExecutionFixture, with the latter's FakeRunner, process identities, workspace readiness and portable ThreadStore. Native OS tests place all homes, data roots and transcripts under fixture folders; ambient native homes must be poisoned/overridden. Live conformance uses the existing disposable recovery acceptance seam under hub-approved staging, never a reboot of a working slot (ARCH-MOCK).

## State, actions and ordering

Actions: inspect, refresh candidates, explicitly resume candidate, resume established binding, start fresh, confirm exact stop set.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|-------------------------------|---------------|-----------------|---------------|
| Usable capture, complete native scan | Existing scored alternatives | Explicitly resume candidate | Wrong explicit choice can omit desired working context |
| Missing/invalid/expired/oversized screen; complete native scan | Unconfirmed native candidates with cwd/activity/native ID and exact screen reason | Explicitly resume candidate | No previous-screen orientation; candidate is not causally bound |
| Missing attributable launch hint; complete native scan | Same unconfirmed choices; launch association unknown | Explicitly resume candidate | More unrelated same-cwd alternatives |
| Native scan partial, timed out or over budget | Candidate scan incomplete; retained descriptive evidence | Refresh candidates | No unbound resume authority from partial inventory |
| Established transcript binding; screen unavailable | Resume from disk (confirmed) | Resume established binding | Screen orientation unavailable |
| Key expired/revision changed/artifact changed | Refresh required, no effects | Refresh candidates | None added by refused selection |
| Live attributed source with exact stop set | Confirmation required | Confirm exact stop set | Only the named owned processes stop |
| Foreign/unknown remnants or conversation in use | Existing refusal/inspection advice; Fresh remains honest | Inspect | No new forced teardown |
| Evidence persisted; retirement interrupted | Existing retained recovery state and evidence | Inspect | Source may already be stopped; intent remains durable |
| Replacement launch unconfirmed | Launch unconfirmed | Inspect | Outcome unknown; exact attempt stays occupied |
| No useful evidence | Start fresh with context-loss warning | Start fresh | Previous conversational working context |

Production ordering remains AdvanceRecoveryExecution: prepared → evidence preserved → retired → claimed → launched. The new source is a discriminated value inside that sequence, not another lifecycle. Before preparing a workspace or preserving/retiring anything, decode the selector, reject future/expired times or source-revision mismatch, reconstruct the complete ordered snapshot at its original cutoff and compare the key. Under the existing per-thread hold, keep exact stop-set confirmation and conversation-in-use checks. Persist the bounded native snapshot with PreserveRecoveryBeforeRetirement before retirement. Recheck selected source before claim and after workspace preparation; independently revalidate it in the launcher.

The thread hold and start-claim nonce/revision CAS already own recovery serialization; do not invent a second lock. A native agent in another slot cannot be locked through our ThreadStore, so the existing live-conversation protection remains necessary. A file observation is point-in-time evidence, not a lifetime lease on an external transcript. Exact before/after fingerprints and final validation detect observed changes; no stronger claim of atomic native-file ownership is made.

Cancellation before retirement causes no stop/launch. Cancellation after persistence retains the intent. Duplicate or delayed confirmations retain existing single-use/exact-witness semantics. Expiration while validation is reading refuses at the final current-clock check. StartFailedConfirmed and late registration behavior from #714 stay owned by existing transitions; test composition rather than adding a native-only rollback path.

## Operating envelope and trust

This is an explicit recovery scan, not a keystroke path. Keep 30s total scan timeout, 10,000 metadata visits including directories, 64 eligible roots, 256MiB aggregate actual IO, 2MiB per framed record and existing bounded message windows. No scan-budget increase, device/inode relaxation, background worker or new durable preview cache. Resolve unrelated cwd only after the existing parser provides it; do not add a per-candidate global rescan. One metadata pass, one bounded parse per visited artifact; selected validation reads only the exact artifact. Sorting is O(N log N), parsing O(total bytes), bounded memory O(metadata + eligible roots × existing message window). Count metadata visits, stat calls, byte reads and cwd resolutions at 8 and 16 transcripts and unrelated entries; reject quadratic work. Network and credentials are not used.

Screen preflight rejects oversized raw input before replay and preserves remaining native budget. Any bytes actually read before a screen failure count against the same request budget; native-only fallback does not reset it. If that leaves insufficient bytes, report true native incompleteness. Retry attempts share the request deadline and consumed-IO accounting; a second attempt must not silently double the advertised byte envelope.

Native JSONL, selectors, persisted records, root names and paths cross trust boundaries. Validate known types, duplicates, sizes and required fields; runtime resolution authorizes roots and canonical relative paths, refuses symlinks/special files and matches physical cwd. No prompt/text from native artifacts becomes argv. Native resume gets only the selected native ID through the existing argv builder; orientation is descriptive and is delivered as a launch prompt only for Fresh.

## Format evolution and artifact lifetime

Keep thread record schema 4 and outer RecoveryAuthority version 1. Context v1 keeps its capture-backed meaning, known-field bytes and retention checks. V2 is required because ignoring its native source would erase the safety preconditions on an action: old context validation refuses Version=2, including a v2 record with the new source member stripped. Reject native-only with any capture, capture generation, tail score/counts, missing native selection or invalid mixed source fields. Keep expired v2 evidence structurally readable after restart; only action admission enforces now.

Use type-owned source-aware MarshalJSON/UnmarshalJSON in recovery_source_json.go, following TranscriptBinding's DecodeKnown/AppendMembers pattern. Preserve unknown v2 context/source/candidate members through record read-modify-write, intent clone, persistedRecoveryIntent, start claim and private launch profile encoding. Keep v1's existing strict nested boundary and frozen schema migration guards explicit; do not reinterpret top-level unknown fields as native authority. V2 decoding must also preserve nested additions while rejecting duplicate/ill-typed known fields. Source-aware encoding emits no capture member for v2 and does not alter v1 known-field output. Add narrow v2 wire aliases if necessary to keep v1 strictness independent of v2 tolerance; do not broaden all record codecs.

The preview/key is in-memory or caller-held and expires after five minutes; listing writes nothing. The selected snapshot lives inside existing recovery intent/record retention, at most five bounded candidate metadata objects, no transcript copies or new artifact family. Existing record replacement/retention owns its end. Capture-backed archives keep their existing retention; native-only creates no archive. Add new production files to artifactpath's exhaustive source inventory and generated mirror classification.

## Task 1: source model, codecs and selectors

**Files:** modify threadrecord/recovery.go, threadrecord/recovery_context_test.go; create threadrecord/recovery_source_json.go and recovery_source_json_test.go; modify couchcore/recovery_key.go and recovery_key_test.go; classify new production source in cmd/internal/artifactpath/manifest.go.

- [ ] Write TestRecoveryNativeOnlyContextRoundTrip, TestRecoveryNativeOnlyContextRejectsMixedEvidence and TestRecoveryNativeOnlyKeyLifetime. Use literal pure fixtures: cutoff T, expiry T+5m, revision 7, one exact candidate with native ID/CWD and zero capture. Assert v2 decode/rewrite/clone preserves unknown context/source/candidate members and rejects wrong known types, duplicate keys, unsafe paths, missing revision, non-five-minute expiry, future activity/hint and mixed capture/tail fields. Expired snapshots remain readable. Freeze v1 bytes and migration fixtures, and pin old-reader refusal using the pre-change v1 validator behavior, including stripped v2 source.
- [ ] Run `go test ./cmd/internal/threadrecord ./cmd/internal/couchcore -run 'Recovery.*(Context|Key|Selector|Schema|NativeOnly)' -count=1`; require the new missing behavior to fail before production edits.
- [ ] Implement the discriminated source and codec, clones and v2 selector fields. Structural validation is clock-independent; a pure admission helper takes now and validates future/expiry/revision. Preserve the 2048-byte key bound. Digest only canonical evidence, never display labels, chosen rank or clock at revalidation.
- [ ] Rerun the targeted tests and `go test ./cmd/internal/artifactpath -count=1`; require PASS, review the v1 golden diff, then commit with #721 and the session Agent/Co-Authored-By trailers.

## Task 2: independent acquisition and bounded native collection

**Files:** modify sessioninventory/recovery_collect.go, recovery_match.go and colocated tests; couchcore/recovery_preview.go, recovery_evidence_os.go, recovery_evidence_os_test.go, recovery_prepared.go; couchcmd/run.go and evidence-reader fixture assignments; create couchcore/recovery_native_only_test.go.

- [ ] Add TestRecoveryNativeOnlyCollectorWithoutLaunchHint and TestRecoveryNativeOnlyReadRetainsScreenFailure. Use the existing runtime fake and portable OS fixture for old device IDs with identical inodes, invalid/absent witness, raw over-budget, missing Pair log/binding, multiple same-cwd roots, native scan failure, semantic activity vs file mtime, and a proven binding with failed screen. Listing must not provision, bind, preserve or write thread lifecycle state.
- [ ] Run `go test ./cmd/internal/sessioninventory ./cmd/internal/couchcore -run 'Recovery.*(Collector|Evidence|NativeOnly|Preview)' -count=1`; record the failing assertions.
- [ ] Introduce typed observation requests containing observation time and optional fixed native source from a decoded v2 selector. Pass the injected Couch clock at listing, the selector source at selection, and the persisted source after claim; adapt existing RecoveryEvidenceReader callsites and production wiring together. Retained v2 intent supplies the original source rather than the latest launch clock. Do not use hidden context values for source policy.
- [ ] Refactor Read into screen observation plus native observation under one deadline/accounting owner. Keep partial facts on screen errors; choose capture-backed source only when existing capture provenance can support it, otherwise native-only with its precise reason. Optional hints require attributable original launch evidence; omit them on witness mismatch or absent provenance. Native-only collector uses cutoff and optional hint but no tail/capture epoch requirement. Share parser/filter/budget mechanics with capture mode; do not call the collector once per candidate.
- [ ] Add TestRecoveryNativeOnlyLinearIO and TestRecoveryNativeOnlyExactBudget: compare counted visits/reads/stats at N=8 and 2N=16 with matching and unrelated roots, exact remaining bytes and one-byte-over, early screen rejection, partial-read fallback, retries, cancellation and mutation during reads. Incomplete inventory retains facts but issues no keys. Assert native-only ranking never invokes scoring and missing hint alone does not set Incomplete.
- [ ] Rerun tests, plus `go test ./cmd/internal/artifactpath ./cmd/internal/testpair -count=1` for source/classification guards; commit the verified acquisition change.

## Task 3: projection and source-bound selection through retirement

**Files:** modify couchcore/recovery_preview.go, recovery_preview_test.go, recovery_prepared.go, recovery_resume_integration_test.go, recovery_native_only_test.go; couchcmd/resume_candidates.go and resume_candidates_test.go; couchtty/menu_recovery.go and menu_resume_candidates_test.go.

- [ ] Add TestRecoveryNativeOnlyPresentation and TestRecoveryNativeOnlySelectionBeforeEffects. Listing/JSON/menu show unconfirmed, physical cwd, native ID, explicitly sourced semantic activity, screen reason and expiry; never tail percentage, causal claim, automatic selection or best-by-match. Cap native/proven choices at five total as today, with Fresh independently available. Plain resume without established binding still refuses.
- [ ] Run `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/couchtty -run 'Recovery|ResumeCandidates|ResumeCandidate' -count=1`; demonstrate the new cases fail.
- [ ] Branch DecideSlotRecovery by source, preserving v1 key/digest and Fresh-from-screen semantics. V2 gets unconfirmed native choices and an ordinary Fresh action with unavailable-screen orientation. Keep the original cutoff/expiry when rebuilding the source for selection; decode malformed/stale/future/copied keys before workspace preparation. Compare complete ordered evidence, not just selected ID. A changed selected file, another ordered candidate, root/CWD mapping, duplicate ID, source revision or optional hint invalidates the key.
- [ ] In executePreparedRecovery persist the v2 native snapshot before retirement with no RecoveryPreserve capture call. Do not rebuild a new-cutoff key after persistence. Reuse PreserveRecoveryBeforeRetirement, source revision fence, exact stop confirmations and conversation-in-use checks. Keep native-only screen absence out of capture-preservation failures. In launchRecoveryClaim reobserve against the persisted source and original revision; retain the same intent through workspace preparation and claim.
- [ ] With newRecoveryExecutionFixture assert zero prepare/preserve/stop/launch effects on pre-admission refusals; durable snapshot visible before a paused retirement; interrupted retirement/retry and record reopen; changed artifact or expiry after workspace preparation refuses before native exec; no transcript binding manufactured across StartClaimed/registration. Assert native resume argv has the exact ID and no prompt; Fresh still receives honest orientation. Remove tail-match wording from native-only success notices.
- [ ] Rerun the command and commit only after PASS. Update atlas/couch.md and atlas/session-identity.md with source labels, separate completeness, five-minute keys and compatibility; atlas/index.md changes only if a new page is needed.

## Task 4: independent final authority and conformance

**Files:** modify launcher/recovery_authority_os.go, recovery_authority.go only if required by typed source validation, recovery_authority_test.go, recovery_authority_os_test.go, recovery_authority_home_test.go; extend couchcore/recovery_native_only_test.go and couchcmd/recovery_acceptance_test.go.

- [ ] Add TestRecoveryNativeOnlyAuthorityAtExec using portable durable record and native home, plus pure ValidateRecoveryRecord cases. At a paused final validation mutate content (including same-size mutation), stable identity, root, CWD alias, provenance, duplicate selection, nonce/revision/durable intent and clock. Require refusal; a copied key/profile cannot bypass the durable creating claim. Poison ambient native roots. Assert zero archive reads/creation for native-only and unchanged exact archive/witness checks for v1.
- [ ] Run `go test ./cmd/internal/launcher -run 'Recovery' -count=1`; capture failures, then branch ValidateRecoveryRecord expiry by source. V2 requires fixed five-minute lifetime, selected membership, exact source revision consistency, agent/helper/nonce and persisted intent. Preserve final record-byte reread and final now check.
- [ ] Branch OSRuntime.ValidateRecoveryAuthority only at capture-specific checks. Reuse RevalidateRecoveryCandidate against the exact persisted entry, selected ID, physical cwd and source cutoff for both sources. Extract the existing selected-entry conversion into a shared helper if needed; do not duplicate the reader or enumerate unrelated native roots at final validation. Keep known root/runtime authorization, explicit resume argv and no launch prompt.
- [ ] Run `go test -race ./cmd/internal/couchcore ./cmd/internal/launcher ./cmd/internal/sessioninventory -run 'Recovery|Resume' -count=1` and `go test ./internal/repolint ./cmd/internal/artifactpath ./cmd/internal/testpair -count=1`. Run existing detached/warm/cold resume tests, naming the actual discovered test selectors in the Log. Include #714 failure-finalization/late-registration regressions and #695 transcript-binding preservation tests. Record mutation controls: remove fixed cutoff propagation, final artifact revalidation or pre-retirement persistence temporarily and require their regressions to fail; restore before commit.
- [ ] Ask the hub for a concrete disposable qualification decision on the tested commit before `sdlc stage --build`. Qualify candidate listing/explicit selection, emitted revisions, failed-launch diagnostics and immediate retry with portable roots and no active-slot reboot. Hub decides real-slot actions; clear stage afterward unless landing clears it. Record a waiver only if the hub explicitly grants it for #721.

## Risky functions: strategy and mechanical guard

| Function / boundary | Adversarial strategy | Guard / regression |
|---------------------|----------------------|--------------------|
| RecoveryContext source codec / CloneRecoveryContext / persistedRecoveryIntent | Newer nested fields disappear or v2 decodes as capture authority | Decode-rewrite-clone-profile roundtrip with unknowns; frozen v1 bytes; old-reader v2 refusal |
| decodeRecoveryCandidateKey / ValidateRecoveryCandidateKey | Caller moves cutoff/expiry/revision or copies key to another tag | Bounded typed selector, digest reconstruction, exact equality, current-clock and source revision tests |
| OSRecoveryEvidenceReader.Read / recoveryObservation / ResumeCandidates | Screen error hides readable native facts or renews cutoff | Independent typed outcomes, partial observation retained, injected clock and missing-screen fixture |
| CollectRecoveryEvidence / readRecoveryBytes / RankRecoveryCandidates | Missing hint marked incomplete; duplicate roots, nonlinear scans or budget reset | Stateful counted N/2N tests, exact byte boundaries and native-only no-score assertion |
| DecideSlotRecovery / CLI / renderRecoveryMenu | Weak candidate appears confirmed or as a measured zero match | Golden JSON/text/menu with source kind/CWD/activity; no Automatic/Proven manufacture |
| executePreparedRecovery / launchRecoveryClaim | Source changes after key check; retirement precedes persistence | Paused stateful execution and durable reopen at each boundary; zero-effects refusal counters |
| ValidateRecoveryRecord / OSRuntime.ValidateRecoveryAuthority | Forged durable intent, stale file, elapsed expiry while reading | Exact claim/helper/nonce checks, real portable native mutation, final bytes/clock checks |
| Production run.go wiring | Tests use a fake source route production never calls | Production reader construction test with controlled native home and non-host clock; disable wiring as mutation control |

## Close and landing

- [ ] Reconcile Core concepts rows with the actual name-status diff; append Revisions for changed locations or scope instead of silently replacing the plan. Tick completed tasks, record measured counts/bounds and remaining historical evidence gaps in the issue Log. Keep #699 workflow stashes untouched.
- [ ] Commit/push the tested branch through `sdlc pr`, then run `sdlc local-check run --name suite` on the pushed head. Record the exact head and suite evidence; do not type an actual-hours guess.
- [ ] Run `sdlc close --issue 721 --verified '<exact commands, suite head, and qualification evidence>'`. Its fresh-context review owns the review boundary. Address findings and record the verdict; no duplicate ad-hoc review. On SHIP enter `sdlc merge --yes`; follow any integration/CI error as its next-action instruction.
- [ ] Verify PR landing/tracker/archive evidence and report the outcome once to hub, checking the delivery receipt. Do not claim shipped while waiting for approval, qualification or landing.

## Revisions

- 2026-10-19: Initial #721 plan adapts approved native-only scope to current #695 transcript/codec ownership. Original #714 M2/M3 become one atomic issue boundary; no behavior is deferred. Added explicit selector cutoff propagation, shared retry IO accounting and source-aware compatibility tests after inspecting current owners.
