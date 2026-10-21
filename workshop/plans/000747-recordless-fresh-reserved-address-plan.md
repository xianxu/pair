---
collaboration-mode: discussion
---

# Recordless Fresh registration implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Keep #742's branch, claim and pair:0 stage intact; switch back to land it when the hub reports live acceptance.

**Goal:** An explicitly requested recordless CLI Fresh launch establishes and registers its own new address through existing validated recovery authority.

**Architecture:** Repair the producer that omits recovery authority. Commit an existing tagged Fresh intent on the new record/start claim, re-read the committed value, and reuse the existing recovery launch tail. The launcher and marker validators remain authoritative and unchanged.

**Tech Stack:** Go; Couch's file-backed thread store and start reducer; Pair launcher; existing runner/process/terminal seams.

## Evidence and alternatives

The #744 native failure output is the established-address refusal for a genuinely reserved worker address. The recordless producer emits FreshRequired/FreshNonce but Recovery=nil. createflow.go therefore calls read-only RegisterExistingCouchThread instead of its validated reserved-establishment branch. Its cleanup also lacks the recovery intent that StartFailedConfirmed requires.

1. **Recommended:** reuse tagged RecoveryFresh + launchRecoveryClaim. One authority contract serves saved-record and recordless recovery; preserves the exact helper/nonce gate and cleanup model.
2. Add another launch-authority type for recordless Fresh. Rejected: duplicates the existing intent/start-claim protocol without a distinct permission.
3. Let any FreshRequired/couch-owned launch establish reserved markers. Rejected: removes existing validation, including the ordinary Fresh reserved-refusal contract pinned in checkpoint_transport_test.go.

#728 currently names saved configuration lookup and parked-source admission failures. Neither proves this marker defect; no switch behavior changes belong here.

## Core concepts

### Pure entities

No new entity or persisted format is introduced. Existing RecoveryIntent, RecoveryAuthority and StartClaimed retain their shapes and validation semantics. The new consumer constructs the existing Fresh case, rather than inferring authority from its prompt.

| Name | Lives in | Status |
|---|---|---|
| RecoveryIntent / RecoveryAuthority / StartClaimed | cmd/internal/threadrecord/recovery.go; cmd/internal/launcher/recovery_authority.go; cmd/internal/couchcore/starttransaction.go | existing, reused |

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Recordless Fresh producer | cmd/internal/couchcore/recovery_fresh_missing.go; slotrecovery.go | modified | Existing slot selection, O_EXCL address claim, store CAS |
| Authorized recovery launch | cmd/internal/couchcore/recovery_prepared.go | existing, reused | Workspace readiness, committed profile, acknowledged helper |
| Pair launch validation | cmd/internal/launcher/createflow.go; recovery_authority_os.go; thread_claim.go | unchanged consumer | Real record/helper observation and reserved marker establishment |

The durable O_EXCL marker owns the address for the thread lifetime; the committed creating start owns its one helper/nonce. Existing CAS and helper acknowledgment serialize launch. No scan or retry substitutes for these resources.

## Implementation tasks (one atomic close boundary)

### 1. Pin the producer-to-launcher failure

**Files:** cmd/internal/couchcore/recovery_fresh_missing_test.go; new focused recordless launch-boundary test beside it if needed; cmd/internal/launcher/recovery_fresh_claim_test.go.

- [ ] Extend the stateful recordless fixture to inspect the actual emitted profile and committed start after helper acknowledgment. Require RecoveryFresh intent equality, exact new address, source revision no newer than committed revision, exact nonce/helper and existing orientation. Require the real launcher recovery-record validator to accept it. Current code must fail because Recovery is nil.
- [ ] Add an executable-boundary regression using the actual emitted child argv/environment and file-backed marker/record. Use the existing shared Pair test binary and isolated testpair environment, with controlled external terminal handoff; exercise public launch parsing and real OS authority validation. Assert reserved -> established only for the exact authorized helper. A controlled terminal stop can bound this check after claim validation; #744 remains the full native worker-registration acceptance.
- [ ] Keep the regression's pre-fix failure diagnostic: existing established-address refusal, or earlier missing genuine Recovery authority. Do not manufacture a registered artifact to make this boundary pass.
- [ ] Bound and join the test child/IO; keep all homes, stores and command fixtures under disposable roots. Do not touch the operator's native sessions.

### 2. Carry genuine Fresh intent through the existing transaction

**Files:** cmd/internal/couchcore/recovery_fresh_missing.go; cmd/internal/couchcore/slotrecovery.go. Remove the unused prompt-driven Fresh tail in cmd/internal/couchcore/couch.go only if this routing leaves it without callers.

- [ ] Pass structured branch/issue/unavailable-screen orientation from recordless recovery into its existing slot transaction instead of using a nonempty prompt as the permission signal. Ordinary callers continue with no recovery orientation.
- [ ] In the recordless transaction's existing claimFreshRecord Commit callback, construct the existing tagged RecoveryFresh intent for the allocated record address, its initial revision, selected agent and creation time; nonempty selected key bound to that address/revision, no native ID. Normalize through persistedRecoveryIntent. Attach identical cloned intent to record.RecoveryIntent and its creating Start.RecoveryIntent before replaceSlotCurrent.
- [ ] Preserve existing source/current authority CAS, address collision retries, retention claims and rollback. No second record or follow-up publication window.
- [ ] Re-read the committed record from the slot backend, as launchRecoveryFreshAddress does: claimFreshRecord's returned pre-callback value is insufficient.
- [ ] Route recordless launch through launchRecoveryClaim with recoveryClaimed phase, RecoveryFresh selection, committed intent and existing recoveryExistingWorkspace readiness. Preserve slot reference, one-time orientation, no workspace convergence, no stopping unproved owners, and existing registration confirmation.
- [ ] Leave ordinary Fresh callers on launchClaimedThread. Keep RegisterExistingCouchThread, EnsureThreadAddressForPair and the launcher's Recovery gate unchanged.
- [ ] Run the red regressions and focused recovery tests to green; commit the atomic fix and tests with model trailers.

### 3. Defend negative authority and failure outcomes

**Files:** same focused test files; cmd/internal/couchcore/recovery_launch_failure_test.go as needed.

- [ ] Test changed helper generation, attempt nonce, address/scope, agent/cwd, missing or established marker, missing intent and newer/future record. Refusals leave marker bytes and native launch effects unchanged.
- [ ] Retain ordinary Fresh reserved refusal, cold-resume established registration, direct Pair, and foreign-owner/no-setup-refresh cases.
- [ ] Inject post-ack helper failure and verify existing confirmed cleanup can retain the Fresh selection; uncertainty remains unconfirmed and never launches a replacement automatically.
- [ ] Mutate away intent propagation and require the boundary oracle to fail. Mutate the nonce/helper authority check only in a temporary overlay and require its negative oracle to fail; restore and rerun focused checks.

### 4. Qualification and landing

**Files:** atlas/couch.md; README.md if current description changes; tests/affected-go-readers.json for changed audited readers; issue Log and lessons.

- [ ] Record producer/consumer authority in atlas; no new public CLI grammar. Audit and refresh reader digests only after input-scope inspection.
- [ ] Run focused recovery/claim tests in couchcore and launcher, affected couchcmd/command tests, and focused race/vet. Run recorded sdlc local-check affected on the pushed code.
- [ ] Coordinate with pair:5/hub for #744's independent main-based native consumer to start all three genuine numbered workers with no bypass or synthesized registration. Its branch remains independent; no stacking or taking its test harness changes onto747.
- [ ] Run sdlc close with recorded evidence, address findings, check PR CI, and land through sdlc merge when independent native qualification clears. No stage replacement while #742 owns pair:0; coordinate any later live staging through the hub.

## Architecture and bounds

- **ARCH-DRY:** reuse RecoveryFresh, persistedRecoveryIntent, launchRecoveryClaim and the existing OS authority validator; no second protocol.
- **ARCH-PURE:** intent data and start transitions stay pure; thread store, helper and marker IO remain existing adapters.
- **ARCH-PURPOSE:** cover actual producer/consumer handshake plus the independent three-worker native consumer, including the secondary missing-intent cleanup symptom.
- **ARCH-MOCK:** keep current stateful runner/artifact/store fakes; executable boundary and #744 native consumer qualify their modeled handshake. No new external dependency.
- **ARCH-CONSTRAINTS:** per requested launch, one existing bounded tag-allocation loop, one store commit and one committed-record reread before the existing bounded launch/registration flow. No per-byte hot path, new polling loop, cache or goroutine; existing limits remain.
- **ARCH-STATE:** retain existing creating/registered/confirmed-failure/unconfirmed transitions and existing resource ownership. The issue's visible-state table covers refusal, partial effects and recovery actions.
- **ARCH-FUNERAL:** creates no new artifact family; existing thread/claim/recovery retention and failed-helper cleanup own the same artifacts. Test children and IO die with bounded test cleanup.
- **ARCH-EVOLVE:** reuse existing persisted fields/codec; no format or version change. Unknown-field preservation and validation remain with existing codec.
- **Simplicity / root cause:** missing genuine authority is supplied at its producer; no deadline tuning, marker bypass, or saved-configuration fallback.

## Approval checkpoint

Plan only: no production/test edits before hub approval and sdlc change-code. On #742 live acceptance, commit any current747 atomic work, switch to its preserved branch, land742, then return here as directed.
