---
collaboration-mode: discussion
---

# Operator acquisition at the writer commit boundary

> **For agentic workers:** Consult AGENTS.md Section 3. Use superpowers-executing-plans for the coupled ownership and transaction work; SDLC owns the fresh review at close. Steps use checkbox syntax.

**Goal:** An acknowledged registered operator acquisition excludes every subsequent session-sync writer commit until that exact operator turn releases protection.

**Architecture:** Run acquisition and release inside the existing registry transaction, which already encloses registered document, shadow, state and round writes. Persist the caller's turn token in the existing operator sidecar and derive retry decisions from those exact bytes. Parley owns freshness after acknowledgment and its existing move journal; no second transaction phase, daemon dependency or new journal.

**Tech Stack:** Go, existing registry flock and durablefile publication, public couch CLI, disposable filesystem-backed tests.

## Core concepts

### Pure entities

| Name | Lives in | Status |
| --- | --- | --- |
| OperatorTurn / operator lock proof | `cmd/internal/sessionsync/operator_lock_model.go` | new |
| OperatorLock result and decision | `cmd/internal/sessionsync/operator_lock_model.go` | new |
| sessionSyncArgs / sessionSyncResponse | `cmd/internal/couchcmd/session_sync.go` | modified |

An OperatorTurn is `(registration ID, caller turn token)`: one registration has successive turns, and one turn can have several identical sidecars during an existing move/alias lifecycle. The token is a caller-generated 128-bit nonce encoded as exactly 32 lowercase hexadecimal characters. Registration IDs retain their existing strict format. Tokens are ownership witnesses, never timeouts or expiring leases.

The pure decision consumes the requested turn and a bounded list of lock observations, and returns refusal, existing ownership, publication or removal effects. All existing operator locks must match before any mutation. A malformed/legacy operator lock is foreign protection, not free space. Legacy `holder: agent` remains non-protective and is never adopted as operator ownership. Reuse ParseHolder for this distinction. Strict proof parsing rejects duplicate known fields, missing/invalid ID or token and oversized text; unknown lines are tolerated and the full raw text is retained unchanged on retries/moves. Unit tests require no IO mocks.

The result is one tagged State, not combinations of ownership flags. The same model drives acquire/release decisions and JSON encoding. Future ownership metadata can be additive without changing turn identity; there is no speculative owner service or lease manager.

### Integration points

| Name | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| Registered operator transaction | `cmd/internal/sessionsync/operator_lock.go` | new | existing withRegistry, canonicalDocument, documentLockPaths |
| Operator lock files | `cmd/internal/sessionsync/operator_lock.go` | new | bounded regular-file reads, durable publication/removal and directory fsync |
| Registry store error classification | `cmd/internal/sessionsync/registry.go` | modified | store IO failures, distinguished from registry validation/busy refusals |
| Public operator CLI | `cmd/internal/couchcmd/session_sync.go` | modified | argv, namespace resolution and JSON stdout |

Operator lock files have a small package-private filesystem seam held by registryStore, defaulting to the real portable directory backend. Tests use that real backend with a counting/failure decorator and a stateful in-memory lock-file fake for publication/removal uncertainty. The fake keeps separate visible and directory-confirmed path-to-bytes maps: effects change visible state before injected uncertain errors, successful directory sync advances durable state, and simulated crash restores durable state. It is not a list of expected function calls. Real-filesystem and executable tests check conformance with it.

The writer keeps its current transaction and mutation order. No new writer lock, watcher, subprocess or retry worker is introduced. A private registry-access error wrapper identifies only actual store access failures for the new commands; existing verbs retain their current outward errors.

## Pinned consumer contract

The hub relayed parley.nvim:1's agreement in Couch d359cf75-8956-45ac-b6ac-ae5957cca584. This plan adds the three requested pins; exact failure taxonomy is included for final approval.

```text
couch --session-sync operator-acquire FILE --id ID --turn TOKEN --json
couch --session-sync operator-release FILE --id ID --turn TOKEN --json
```

Both commands require JSON, ID and TOKEN. FILE must identify the currently registered document; aliases are resolved under the registry transaction and the acknowledgment returns its current canonical File. An ID mismatch, missing registration, invalid/missing path or invalid token refuses before sidecar mutation. There is no legacy/unregistered fallback. The retired bare `release` command keeps its existing no-op compatibility behavior.

Acquire success exits 0 and emits one JSON object:

```json
{"Code":"ok","Error":"","OperatorLock":{"ID":"0123456789abcdef0123456789abcdef","File":"/canonical/document.md","Holder":"operator","Turn":"fedcba9876543210fedcba9876543210","LockPath":"/canonical/document.md.lock","LockText":"holder: operator\ntime: 2026-10-21T22:00:00Z\ndocument: 0123456789abcdef0123456789abcdef\nturn: fedcba9876543210fedcba9876543210\n","State":"acquired"}}
```

`State` is `acquired` for newly published protection, or `already-held` when all observed protective locks belong to the same ID/token. An identical retry preserves ownership, exact LockText and its time; it does not refresh or create a different turn. New-versus-existing is deliberately allowed to change after a lost reply. A valid success proves all preceding registered writers have returned from their transaction and all subsequent writers observe operator protection. It does not claim the editor buffer is current: Parley must perform its approved unconditional post-ack freshness/read-completion check before admitting input.

Release success uses the same OperatorLock envelope, echoes ID, canonical File and Turn, and has State `released` or `already-released`. Holder is the empty string after release. LockPath is the canonical sidecar pathname and LockText is empty, so the response does not advertise protection that no longer exists. No operator sidecar means `already-released`, without claiming the caller previously owned it. Non-protective legacy agent sidecars are preserved. Any foreign protective sidecar refuses release before any deletion. Several matching aliases are removed under the same transaction; partial removal is uncertain and an identical retry finishes only still-owned protection. Both release states require directory confirmation for every admitted sidecar directory, including paths whose lock is already absent, so a retry reconciles an earlier unlink followed by failed fsync.

| Failure | Code | Exit | Meaning / safe action |
| --- | --- | --- | --- |
| Foreign, legacy or malformed operator protection | `operator-lock` | 3 | No ownership acquired or removed; do not edit as confirmed or delete protection |
| ID differs from current registration | `operator-identity` | 1 | Refresh registration explicitly; never substitute a new ID and replay blindly |
| Unregistered/missing/invalid/nonregular document or unsafe lock path | `operator-path` | 1 | Inspect path; no legacy fallback or editable-unavailable classification |
| Invalid ID/token/argv | `operator-invalid` (CLI usage may precede JSON) | 1 | Correct request; no effects |
| Registry contention | existing `registry-busy` | 1 | Explicit retry; existing 2 s producer wait remains bounded |
| Namespace cannot be resolved, or registry store cannot be accessed | `session-sync-unavailable` | 7 | Consumer may use its approved visibly unconfirmed editing policy |
| Invalid registry/metadata authority | explicit refusal, never unavailable | 1 | Repair/inspect; corruption is not transport absence |
| Sidecar publication/removal/durability may have happened | `operator-lock-uncertain` | 1 | Preserve token and bytes; repeat identical request to reconcile |

Unknown JSON fields are tolerated by the consumer. No success envelope is emitted on errors. The producer does not classify document/sidecar permission errors or malformed authority as store unavailability. Existing stale writer CAS remains `stale`/6; these new commands do not bypass or reinterpret it.

These are in-process commands: they do not contact or construct the Couch supervisor. A stopped supervisor alone does not make acquisition unavailable. Parley's approved synchronous limit is 500 ms; it owns terminating/joining the CLI and classifying timeout separately. Producer registry contention remains at its existing bound rather than tuning another deadline. If the CLI dies after publication but before stdout, protection remains and the same token recovers it; no background operation survives process termination.

## Transaction and sidecar lifecycle

1. Validate ID/token, enter withRegistry, canonicalize FILE and verify registration ID/current regular document.
2. Enumerate documentLockPaths once, including the canonical spelling. Inspect each admitted alias still naming that exact document with a 4096-byte bounded regular-file read. Unsafe/unreadable observations refuse; reused old paths do not grant authority.
3. Apply the pure decision. Foreign protection anywhere refuses the complete operation. Matching ownership is returned verbatim after confirming its containing directory's durability. Otherwise acquire publishes one canonical sidecar with holder/time/document/turn fields.
4. Publication reuses durablefile.WriteAtomicStaged with one deterministic per-document stage, owned exclusively while the registry flock is held. Publish complete bytes, fsync file, rename, fsync the sidecar directory, then encode success. An incomplete stage never counts as protection. The next operator transaction removes/reuses only that exact regular stage under the same exclusion; unsafe stage types refuse. Release removes only matched owned sidecars and fsyncs every admitted sidecar directory before either success state, including already-absent locks. Matching acquire retries also confirm existing ownership's directory durability. Any confirmation failure remains uncertain, regardless of visible file presence or absence.
5. Return and release the existing registry flock. The sidecar remains for the operator turn; it never expires. Release/submit/unchanged-turn cleanup is called by Parley's existing lifetime owner. A crash leaves protection for explicit same-turn recovery or existing operator recovery, not automatic expiry.

The complete writer, including document/shadow/state/round effects, precedes acquisition acknowledgment if it acquired the registry first. If acquisition wins, Write and Apply read its sidecar in their existing guard and refuse before document/metadata mutation. Read/acknowledgment-only commands retain existing semantics.

Migration keeps the existing Parley move journal as owner: acquire OLD before moving, copy the exact lock bytes to NEW before committing registry rename, preserve the registration ID, then release or re-acquire NEW with the same token. `turn:` is explicitly recoverable from the persisted LockText without the old reply. Ownership is path-independent; canonical routing is checked afresh. An old-path release cannot remove a new occupant's lock. Existing rename/forget behavior is not weakened, and forget never deletes operator sidecars. Missing/mismatched registration after forget refuses rather than inventing ownership. Parley owns old-sidecar/journal cleanup.

Direct sidecar writers from older clients do not join this exclusion protocol. Their existing conservative protection remains respected, but the new linearization guarantee is for registered acquisition through these commands. Preserve the writer's post-write warning for legacy/uncoordinated clients and correct its comment to describe that remaining boundary.

## States, failures and recovery

Actions: acquire, retry the identical acquisition, edit after confirmed freshness, release the exact turn, retry identical release, inspect/recover through the existing operator lifecycle.

| Internal state / failure | Visible state | Recovery action | Possible loss |
| --- | --- | --- | --- |
| Earlier writer owns registry | Acquisition waiting | Wait within consumer bound, then retry identical acquisition explicitly | No document bytes lost; earlier commit may be visible after ack |
| No protection, publication succeeds | Confirmed new turn | Edit after freshness | None |
| Same ID/token protected, including lost prior reply | Confirmed existing turn | Edit after freshness | None; original time/text retained |
| Foreign/legacy/malformed protection or unsafe path | Refused, unconfirmed | Inspect/recover existing lifecycle | No bytes changed |
| Process dies before publication | Unconfirmed | Retry identical acquisition | At most one private incomplete stage; no protection claimed |
| Process dies or fsync fails after publication | Unconfirmed, possibly protected | Retry identical acquisition | No document loss; lock can remain until recovery |
| Store/namespace unavailable or consumer kills at 500 ms | Visibly unconfirmed under consumer policy | Retry identical acquisition | Editing may conflict; producer never claims protection from timeout |
| Matching release succeeds | Released | Acquire a new turn before editing | None |
| Release reply lost / partial alias removal | Unconfirmed release | Retry identical release | Protection may remain; no foreign lock removed |
| Release retried after absence | Already released | Acquire a new turn before editing | No past ownership inferred |
| Another turn acquired before stale release/retry | Refused | Inspect/recover existing lifecycle | New turn preserved |
| Move copies lock then commits rename | Same turn at NEW | Release exact turn at NEW | Existing journal owns interrupted move recovery |
| Forget/re-register or old path reused | Identity/path refused | Inspect registration explicitly | No other registration's protection removed |

Cancellation is process termination, not rollback: completed sidecar effects can persist. Clock only stamps a newly created sidecar; it never proves ownership or expiry. Concurrent actors and delayed replies are serialized by the flock and guarded by ID/token. There is no queued asynchronous completion to apply after the command exits. Producer covers acknowledgment and ownership; Parley covers input gating, native freshness, degraded edits and clobber copies in parley.nvim#408.

## Operating envelope and architecture

This is once per new operator turn, not per character. Registry size D remains capped at 16 MiB/4096 documents; admitted spellings P retain the existing 256 bound. Each command performs one registry load/lookup O(D), one alias enumeration and at most one lock read per unique spelling O(P), and at most P removals/directory confirmations on release. Acquire publishes one bounded sidecar; it does not read every document, invoke a process, scan other buffers or load the delivery ledger. Memory is O(D + P*4096). Registry contention uses the existing 2 s bound; caller UI budget is 500 ms. Network/CPU-heavy work is absent. Filesystem permissions and fsync failure produce explicit refusal/uncertainty rather than more retries.

Counted tests at 8 and 16 admitted aliases assert one registry transaction, at most one observation per unique lock pathname, one acquire publication, and at most P releases. Duplicate aliases do not cause quadratic rescans. The file seam's real-backend decorator counts operations; wall-clock timing is secondary evidence only.

- **ARCH-DRY:** existing registry flock, canonical identity, alias authority and ParseHolder remain owners; bare retired release is not repurposed.
- **ARCH-PURE:** turn/proof parsing and effect decisions are pure; a small transaction shell performs IO.
- **ARCH-PURPOSE:** test Write, Apply and move-copy/release against the same acknowledgment boundary; consumer delivery is explicitly coordinated with #408.
- **ARCH-MOCK:** portable folder backend and stateful uncertain-effect fake share the file seam; executable CLI conformance uses disposable namespace files.
- **ARCH-CONSTRAINTS:** existing bounds plus 500 ms consumer budget and 8/16 counted alias tests above.
- **ARCH-SECURE:** the caller is a local same-user editor, not an authenticated remote principal; the nonce proves a particular turn rather than the operator's person. Registration ID plus freshly canonicalized path bounds authority. Registry and sidecar inputs remain untrusted: reject unsafe file types/known-field corruption, preserve foreign protection and never follow sidecar symlinks for destructive effects. A client that deliberately bypasses the transaction is outside the cooperative exclusion contract.
- **ARCH-ORDER:** the tagged decision and state/recovery table preserve lost replies, process death, partial release and stale actors; no new fan-out, queue or background lifetime survives transaction return/CLI exit.
- **ARCH-FUNERAL:** one sidecar per protected spelling, removed by exact-turn release/existing move owner; one deterministic stage per document, reclaimed on next operator transaction. No accumulating receipt ledger or process.
- **ARCH-EVOLVE:** additive document/turn sidecar lines; retain raw unknown lines on existing ownership, never rewrite foreign locks. Registry JSON and move journal schemas do not change.

## Tasks — one close boundary

The production test surfaces are named here; tasks below apply these strategies rather than maintaining a second prose inventory of cases.

| Risky production function | Adversarial strategy and mechanical guard |
| --- | --- |
| `validateOperatorTurn` | Arbitrary ID/token strings; strict fixed-width lowercase-hex validation before IO |
| `parseOperatorProof` | Fuzz arbitrary bounded sidecar bytes, seeded with ambiguous known fields; reject ambiguous ownership and preserve raw unknown lines |
| `operatorLockText` | Pure encode/parse properties; exact nonce/identity recovery without consulting clock or a previous reply |
| `decideOperatorLock` | Sequence-test competing turns and partial observations; foreign protection yields zero effects and stale ownership never deletes a new turn |
| `(*Sync).AcquireOperator` | Controlled writer/contender ordering and publication failure/death; no ack crosses an earlier writer transaction and no future writer commits under acknowledged protection |
| `(*Sync).ReleaseOperator` | Visible-versus-durable fake crash sequences; neither success state precedes directory-confirmed removal, and identical retry completes only owned effects |
| `operatorLockFiles` methods | Real portable-folder conformance and stateful uncertain-effect fake; bound observations/effects, reject unsafe file types, and count linear work at 8/16 aliases |
| `withRegistry` store-error wrapper | Filesystem-access versus authority-corruption injection; only access unavailability maps to Code/7, with all existing verb outcomes preserved |
| `parseSessionSyncCLI` / `runSessionSyncCLI` | Closed argv grammar and executable JSON conformance; require identity/token and distinguish unavailable from explicit refusal without stderr inference |
| `(*Sync).Rename` / `(*Sync).Forget` integration | Move/re-registration event sequences using persisted lock bytes; identity survives valid move and never transfers to a reused path or replacement registration |

### 1. Pin contract and write pure red tests

Files: `cmd/internal/sessionsync/operator_lock_model.go`, `operator_lock_model_test.go`; issue and this plan.

- [ ] Obtain final hub approval including the unavailable/7 taxonomy and copy the complete contract to parley.nvim:1 through the hub.
- [ ] Add pure tests using the named parsing, identity and sequence strategies above.
- [ ] Run `go test ./cmd/internal/sessionsync -run 'TestOperatorLockModel' -count=1`; record red before implementation.
- [ ] Implement validateOperatorTurn, parseOperatorProof, operatorLockText and decideOperatorLock; rerun green and commit with Agent trailer.

### 2. Implement serialized ownership with adversarial ordering

Files: `cmd/internal/sessionsync/operator_lock.go`, `operator_lock_test.go`, `registry.go`, `sync.go`; existing registry tests.

- [ ] Add folder-backed transaction tests, stateful uncertain-effect fake and counted file decorator before production integration.
- [ ] Force a real registered writer to pause between guard and mutation using the existing injected Now callback at prepareWrite (the second write clock boundary), while its registry flock remains held. Start acquisition, observe contention through the existing TryLock seam rather than sleeping, prove no acknowledgment until writer document/shadow/state/round completion, then prove the next writer refuses unchanged.
- [ ] Apply the AcquireOperator/ReleaseOperator adversarial strategies with reproducible subprocess barriers for death and an independent snapshot of document plus all metadata as the no-mutation oracle.
- [ ] Implement new operations under withRegistry with staged durable sidecar publication and explicit error classification. Retain legacy writer warnings and route/CAS behavior.
- [ ] Apply the Rename/Forget identity sequence strategy using the existing move order and persisted lock bytes; keep existing metadata/sidecar recovery tests green.
- [ ] Count real-backend operations at 8/16 aliases; run `go test ./cmd/internal/sessionsync -count=1` and the targeted race-enabled ownership cases. Commit passing implementation/evidence.

### 3. Expose CLI and executable conformance

Files: `cmd/internal/couchcmd/session_sync.go`, existing session-sync CLI tests, new focused executable tests under `cmd/couch/` if that is the existing public-command owner.

- [ ] Add strict parser/JSON tests using the CLI grammar and result-classification strategy above; preserve retired release tests.
- [ ] Implement the two verbs through the same in-process session-sync command owner. Namespace/store unavailability is machine-readable Code/7; authority/path failures are never normalized to it.
- [ ] Run the built real couch executable against a disposable namespace and registered document to conform ownership, uncertainty, migration and availability decisions to the model. Use shared testcouch isolation primitives for any supervisor case; never the live fleet.
- [ ] Run `go test ./cmd/internal/couchcmd ./cmd/couch -run 'Test.*SessionSync.*Operator|Test.*OperatorLock' -count=1`; inspect actual selected test names and make the command match them before recording proof.
- [ ] Ask hub whether the real public-command proof suffices for producer live acceptance or whether coordinated editor staging is needed; do not touch :0 without permission. #408 performs actual native insert conformance after producer landing.

### 4. Document, verify, close and land

Files: session-sync atlas page found through `atlas/index.md`, public couch skill session-sync contract, this plan/issue Log, per-reader affected certificates when test dependencies change.

- [ ] Document the single acknowledgment boundary, exact CLI/JSON/errors, same-token retry, released semantics, stopped-supervisor behavior, legacy limit and rename recovery. Link from atlas index as needed.
- [ ] Update affected-reader certificates using the repository regenerate target; do not change local-gate configuration or apply verification labels.
- [ ] Push via `sdlc pr`, run required recorded affected/suite check on the pushed head, complete SDLC close's one fresh review, resolve findings, and enter `sdlc merge --yes` promptly on SHIP.
- [ ] Report PR/merge and producer contract availability to hub for parley.nvim#408; retain failure/uncertainty evidence in the Log. No self-reported passing checks before commands return.

## Approval checkpoint

Hub approval 061be603-8e01-4db6-99ab-0572a62d93f6 accepts this full-flow plan, its error taxonomy, and public-command conformance on disposable files as producer live acceptance. No :0 stage. The operator's actual editing after parley.nvim#408 lands is the end-to-end proof. No production edits yet; enter change-code's plan-quality gate, then derive the estimate.

## Revisions

### 2026-10-21 — plan gate durability and test-surface clarification

PQ-1: directory confirmation is mandatory on both release success states, including already-absent sidecars, and on matching acquire retries. The uncertainty fake now distinguishes visible from durable state and simulates crash recovery. PQ-2: name the production test functions and give one adversarial strategy/mechanical guard per function; remove duplicate task-level case inventories. These refinements preserve the approved argv, states, error taxonomy and one-transaction mechanism.
