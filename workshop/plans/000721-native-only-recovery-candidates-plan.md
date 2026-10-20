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
| RecoverySourceKind | cmd/internal/sessioninventory/recovery_collect.go | new |
| NativeRecoverySource | cmd/internal/threadrecord/recovery.go | new |
| RecoveryContext / RecoveryCandidate / recovery validation and clones | cmd/internal/threadrecord/recovery.go | modified |
| Recovery source codec | cmd/internal/threadrecord/recovery_source_json.go | new |
| Frozen record schema recovery guard | cmd/internal/threadrecord/record.go | modified |
| RecoveryCollectRequest / native ordering | cmd/internal/sessioninventory/recovery_collect.go and recovery_match.go | modified |
| RecoveryObservationRequest / ScreenAvailability / activity provenance | cmd/internal/couchcore/recovery_preview.go | new |
| RecoveryObservation / RecoveryPreview / RecoveryChoice / DecideSlotRecovery | cmd/internal/couchcore/recovery_preview.go | modified |
| RecoveryCandidateSelector | cmd/internal/couchcore/recovery_key.go | modified |
| RecoveryChoice source-aware JSON | cmd/internal/couchcore/recovery_choice_json.go | new |
| ValidateRecoveryRecord | cmd/internal/launcher/recovery_authority_os.go | modified |

NativeRecoverySource is owned 1:1 by a v2 context, with source kind native-only, fixed UTC cutoff, nonzero source record revision and optional attributable launch hint. A context owns at most five exact candidate snapshots. Candidate fingerprints reuse existing root, relative path, stable file ID, generation/mutation tokens, size and mtime; add physical CWD to the persisted native-only evidence so a later cwd remap cannot be hidden by display formatting. Native timestamps come from semantic native messages; file mtime is separately labelled capture-file activity. Never infer causal conversation activity from a copy/publication clock.

ScreenAvailability describes available or unavailable screen evidence with an exact missing/invalid/over-budget/expired reason and any independently readable activity. Native completeness describes only the native scan. Neither is launch authority. Incomplete native observations retain diagnostics and already observed facts but offer no unbound actionable candidate keys. Proven and Fresh remain independently available.

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

**Files:** modify cmd/internal/threadrecord/recovery.go, recovery_context_test.go; create recovery_source_json.go and recovery_source_json_test.go in that package; modify cmd/internal/couchcore/recovery_key.go and recovery_key_test.go. Classify the new production file in cmd/internal/artifactpath/manifest.go.

- [x] Write failing pure tests for the codec and selector strategies in the function table below, using literal fixtures and the frozen v1 encoding as the compatibility oracle.
- [x] Run `go test ./cmd/internal/threadrecord ./cmd/internal/couchcore -run 'Recovery.*(Context|Key|Selector|Schema|NativeOnly)' -count=1`; require failure from missing behavior before editing production.
- [x] Implement the source model, type-owned codec, clone behavior and source-bound selector according to the format and lifetime contracts above. Clock-dependent admission remains a pure helper taking now.
- [x] Rerun the tests and `go test ./cmd/internal/artifactpath -count=1`; require PASS, review the v1 golden diff and commit with #721 plus session trailers.

## Task 2: independent acquisition and bounded native collection

**Files:** modify cmd/internal/sessioninventory/recovery_collect.go, recovery_match.go and colocated tests; cmd/internal/couchcore/recovery_preview.go, recovery_evidence_os.go, recovery_evidence_os_test.go, recovery_prepared.go; cmd/internal/couchcmd/run.go and evidence-reader fixture assignments. Create cmd/internal/couchcore/recovery_native_only_test.go.

- [x] Write failing tests for acquisition, collection and production wiring using the existing portable OS and stateful runtime fixtures, following the function strategies below.
- [x] Run `go test ./cmd/internal/sessioninventory ./cmd/internal/couchcore -run 'Recovery.*(Collector|Evidence|NativeOnly|Preview)' -count=1`; require the missing behavior to fail.
- [x] Pass typed observation requests through RecoveryEvidenceReader, production wiring and callers. Listing uses the injected clock; selection and retained intents reuse their original source. Refactor Read into independent screen/native observation under one deadline and accounting owner, retaining partial facts on errors. Share existing collector parsing, eligibility and budget mechanics; branch only source requirements and ordering.
- [x] Add counted scaling and exact-bound tests against the declared operating envelope. Run the targeted tests and `go test ./cmd/internal/artifactpath ./cmd/internal/testpair -count=1`; commit after PASS.

## Task 3: projection and source-bound selection through retirement

**Files:** modify cmd/internal/couchcore/recovery_preview.go, recovery_preview_test.go, recovery_prepared.go, recovery_resume_integration_test.go, recovery_native_only_test.go; cmd/internal/couchcmd/resume_candidates.go and resume_candidates_test.go; cmd/internal/couchtty/menu_recovery.go and menu_resume_candidates_test.go.

- [x] Write failing projection and stateful execution tests using the function strategies below and newRecoveryExecutionFixture.
- [x] Run `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/couchtty -run 'Recovery|ResumeCandidates|ResumeCandidate' -count=1`; require missing behavior to fail.
- [x] Branch DecideSlotRecovery and the CLI/menu projections by source, following the visible-state contracts above. Preserve v1 key/digest behavior; native-only choices never advertise screen matching. Native-only Fresh uses the ordinary honest action.
- [x] Carry the original v2 source through executePreparedRecovery, persistence, retirement and launchRecoveryClaim. PreserveRecoveryBeforeRetirement stores native evidence without invoking capture preservation; existing lifecycle/confirmation owners remain authoritative. Native-only success notices describe native activity rather than tail match.
- [x] Rerun the command and commit after PASS. Update atlas/couch.md and atlas/session-identity.md with the source/lifetime/compatibility contract; atlas/index.md changes only if a new page is needed.

## Task 4: independent final authority and conformance

**Files:** modify cmd/internal/launcher/recovery_authority_os.go, recovery_authority.go only if source validation requires it, recovery_authority_test.go, recovery_authority_os_test.go, recovery_authority_home_test.go; extend cmd/internal/couchcore/recovery_native_only_test.go and cmd/internal/couchcmd/recovery_acceptance_test.go.

- [x] Write failing final-authority tests using the function strategies below, literal pure records and portable native homes for OS integration.
- [x] Run `go test ./cmd/internal/launcher -run 'Recovery' -count=1`; require failure from missing native-only support.
- [x] Branch ValidateRecoveryRecord's lifetime validation and OSRuntime.ValidateRecoveryAuthority's capture-specific checks by source. Reuse RevalidateRecoveryCandidate and existing argv construction; extract selected-entry conversion if needed rather than duplicating it. Preserve final record-byte reread and current-clock validation.
- [x] Run `go test -race ./cmd/internal/couchcore ./cmd/internal/launcher ./cmd/internal/sessioninventory -run 'Recovery|Resume' -count=1` and `go test ./internal/repolint ./cmd/internal/artifactpath ./cmd/internal/testpair -count=1`. Run existing detached/warm/cold resume and lifecycle/binding regressions, recording actual test selectors in the Log. Verify the function-table guards with targeted mutation controls, restoring code before commit.
- [ ] Ask the hub for a disposable qualification decision on the tested commit before staging. Use the existing acceptance seam; hub owns real-slot actions and decides whether next-natural-recovery qualification suffices for #721. Record the concrete decision and evidence.


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

- 2026-10-19: PQ-1 (function-level-test-strategy): compressed Tasks 1–4 to TDD/verification steps and removed duplicated case inventories. The function table is the test-strategy owner; design, state, compatibility and operating-envelope contracts above remain governing. No scope or user-facing tradeoff changed.

- 2026-10-20: Implementation checkpoint: source enum lives in sessioninventory to avoid a dependency cycle; type-owned choice JSON lives in recovery_choice_json.go; legacy record guards widened in record.go to freeze schema≤3. Core concepts rows now name actual owners. OS reader retains Read as a direct convenience around injected ReadRequest. Screen availability uses available/unavailable plus exact reasons. Fresh listing creates a new native source observation after an interrupted attempt; selected keys/pending launch retain their original source and expiry. These clarify approved owners/lifetimes without adding a user-visible state.
