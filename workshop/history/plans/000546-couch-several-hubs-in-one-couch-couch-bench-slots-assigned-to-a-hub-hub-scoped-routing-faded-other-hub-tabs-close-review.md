# Boundary Review — pair#546 (whole-issue close)

| field | value |
|-------|-------|
| issue | 546 — couch: several hubs in one Couch (couch . ../bench): slots assigned to a hub, hub-scoped routing, faded other-hub tabs |
| repo | pair |
| issue file | workshop/issues/000546-couch-several-hubs-in-one-couch-couch-bench-slots-assigned-to-a-hub-hub-scoped-routing-faded-other-hub-tabs.md |
| boundary | whole-issue close |
| milestone | — |
| window | b63f4bd76e3e81d1433cfeda544bb3bed8501522..99a69265df7d5bd9a6cf5a5409ba8521bb7c0217 |
| command | sdlc close --issue 546 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T01:18:55-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation substantially delivers the revised `.hub` model and approved CLI-only scope. Shared ownership drives routing, authority, supervision, and presentation; the existing suites pass. Two reproduced ordering and durability bugs block shipping.

```findings
findings:
  - id: new
    severity: Critical
    family: durable-success-requires-confirmation
    title: |
      Persistent retries can report success without confirming durability
    detail: |
      cmd/internal/couchcmd/hub_move.go:152-159 replans from visible bytes, dropping already-satisfied edits. If publication renamed the file but directory sync failed, retry supplies no edit to hubClaimFiles.Apply, bypassing its reconciliation sync and reporting persisted success. Fault injection reproduced success while SyncDirectory was still failing. ARCH-ORDER: reconcile durability even when no byte changes remain.
  - id: new
    severity: Critical
    family: reply-evidence-preserves-admission-ownership
    title: |
      Reassignment can turn a cross-hub message into reply evidence
    detail: |
      cmd/internal/couchcmd/supervision.go:157-164 checks ownership at delivery completion without retaining admission ownership. A foreign hub sends an explicit message, the worker moves to that hub, and delayed delivery clears the worker's unanswered report, suppressing nudges and escalation. A controlled sequence reproduced this. ARCH-ORDER: retain recipient ownership evidence at admission and prevent cross-hub messages from becoming own-hub replies.
  - id: new
    severity: Minor
    family: architecture-docs-match-execution
    title: |
      Atlas still describes the operation queue as serial
    detail: |
      atlas/couch.md:3592 and 3600 attribute serialization to the queue, although it has multiple workers. Document dispatchHubOperation's effect gate and the explicit startup adoption chain as the ordering mechanisms. ARCH-ORDER.
```

1. **Strengths**

   - Immutable `HubOwnership` centralizes resolution and revision-fenced moves.
   - Mutation tests enumerate all six existing slot operations plus switch-agent, including stale authority.
   - Claim edits preserve comments and broad claims; generated cases verify convergence and unrelated ownership.
   - Presentation tests verify single-hub frame equality, fading, and unchanged hit spans. README, atlas, skill, and source inventory updates are present.

2. **Critical findings**

   - [hub_move.go:152](/Users/xianxu/Library/Caches/sdlc/review-checkouts/12631072e26751de/cmd/internal/couchcmd/hub_move.go:152): confirm directory durability on satisfied retries before publishing runtime ownership or returning success. Cover failures after rename, including partial multi-file completion.
   - [supervision.go:157](/Users/xianxu/Library/Caches/sdlc/review-checkouts/12631072e26751de/cmd/internal/couchcmd/supervision.go:157): capture ownership when the recipient is admitted. Delivery must satisfy both admission provenance and current ownership requirements. Test both directions of reassignment around delayed delivery.

3. **Important findings**

   None separately.

4. **Minor findings**

   - [atlas/couch.md:3592](/Users/xianxu/Library/Caches/sdlc/review-checkouts/12631072e26751de/atlas/couch.md:3592): replace the serial-queue explanation with the actual effect gate and adoption chain.

5. **Test coverage notes**

   Full couchcmd, couchtty, couchmessage, and artifactpath suites passed after generating runtime assets in a disposable checkout. Focused core tests and `TestHub` race tests across all four Couch packages passed.

   Two temporary regression probes failed as described above. The durability probe injected directory-sync failure after the real rename; the supervision probe controlled send → move → delivery ordering. Existing lost-ack coverage retries the adapter’s original edits, missing production replanning. The disposable checkout was removed; the pinned checkout remains clean. Full `make -k test` was not independently rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared ownership and mutation guards.
   - **ARCH-PURE — pass:** pure parsing, ownership, edits, and presentation with separate IO adapters.
   - **ARCH-PURPOSE — flag:** persistence confirmation and supervision isolation remain incomplete.
   - **ARCH-MOCK — pass:** existing stateful external seams and private storage fixtures are reused.
   - **ARCH-CONSTRAINTS — pass:** explicit claim, hub, and override bounds; no render-time file reads.
   - **ARCH-SECURE — pass:** authenticated callers, bounded parsing, and no-follow claim reads.
   - **ARCH-ORDER — flag:** both reproduced failures lose evidence across events.
   - **ARCH-FUNERAL — pass:** bounded configuration replaced in place; runtime overrides end at restart.

7. **Plan revision recommendations**

   Add `## Revisions` entries specifying:

   - Durability reconciliation for already-visible claims, including retries with no remaining edits.
   - Admission ownership provenance for reply evidence and delayed-delivery sequence tests.

   The final reconciliation otherwise accounts for the implemented paths and approved switcher-move deferral.

---

## Re-review — 2026-10-14T01:31:29-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 546 — couch: several hubs in one Couch (couch . ../bench): slots assigned to a hub, hub-scoped routing, faded other-hub tabs |
| repo | pair |
| issue file | workshop/issues/000546-couch-several-hubs-in-one-couch-couch-bench-slots-assigned-to-a-hub-hub-scoped-routing-faded-other-hub-tabs.md |
| boundary | whole-issue close |
| milestone | — |
| window | b63f4bd76e3e81d1433cfeda544bb3bed8501522..ea90c472427c275666b8da843995d61170f63e5f |
| command | sdlc close --issue 546 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T01:31:29-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The approved CLI-first scope is implemented, and all three prior findings are verified addressed. One Important gap remains: supervision delivery logging still recognizes only the main hub, silently omitting confirmed deliveries to secondary hubs. Fix that consumer and add regression coverage before closing.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      hub_move.go:159-176 confirms unchanged claim snapshots. Removing this correction makes both TestHubMoveRetryConfirmsAlreadyVisibleClaims cases fail; pinned-head tests pass.
  - id: BR-2
    disposition: addressed
    note: |
      supervision.go:187 checks retained admission ownership alongside current ownership. Removing the guard makes TestHubSupervisionDelayedReplyPreservesAdmissionOwnership/foreign-to-own fail; pinned-head tests pass.
  - id: BR-3
    disposition: addressed
    note: |
      atlas/couch.md:3593-3595 now identifies concurrent queue workers, dispatchHubOperation's effect gate, and the adoption chain; hubs.go implements both ordering mechanisms.
findings:
  - id: new
    severity: Important
    family: ownership-consumers-share-projection
    title: |
      Secondary-hub deliveries disappear from supervision facts
    detail: |
      cmd/internal/couchcmd/message_service.go:1093-1102 still uses hubBinding(), which resolves only authority.hub (main), to admit hub_delivery facts. A disposable production-adapter test confirmed that main receives one fact while a confirmed secondary-hub delivery receives zero. ARCH-PURPOSE: classify every active hub through the shared ownership projection, retaining recipient evidence across delivery, and test main, secondary, and non-hub recipients.
```

1. **Strengths**

   - Persistence retries now require directory durability confirmation, including fully visible batches.
   - Delayed reply tests exercise both reassignment directions and unchanged ownership.
   - Pure ownership and edit planning remain separate from bounded filesystem IO.
   - README, atlas, CLI help, and source inventory cover the new surface.

2. **Critical findings**

   None remaining.

3. **Important findings**

   - **Missing secondary-hub delivery facts** — `message_service.go:1093`. Replace the main-only predicate with shared active-hub classification captured before delivery. Preserve explicit cross-hub logging semantics and test all recipient classes.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Full couchcmd, couchmessage, couchtty, and artifactpath tests passed.
   - Focused hub/core tests and hub/supervision race checks passed.
   - Both prior behavioral corrections independently produced failing tests when removed.
   - The exploratory logging test reproduced the new finding.
   - Full `make -k test` was not independently rerun. Disposable checkout removed; review tree unchanged.

6. **Architectural notes**

   - **ARCH-DRY:** Pass — shared ownership and mutation guards.
   - **ARCH-PURE:** Pass — pure claims, resolution, edits, and presentation.
   - **ARCH-PURPOSE:** Flag — delivery logging retains the single-hub assumption.
   - **ARCH-MOCK:** Pass — stateful socket/process fixtures and injected failures.
   - **ARCH-CONSTRAINTS:** Pass — bounded claims, overrides, and cancellable effects.
   - **ARCH-SECURE:** Pass — authenticated capabilities and bounded no-follow reads.
   - **ARCH-ORDER:** Pass — effect gate, adoption chain, revision checks, and controlled sequences.
   - **ARCH-FUNERAL:** Pass — bounded configuration replacement and process-lifetime overrides.

7. **Plan revision recommendation**

   Add a `## Revisions` entry stating that supervision fact logging is an ownership consumer: all active hubs must receive delivery facts, with recipient classification retained across delivery. Include regression coverage for main, secondary, and non-hub recipients.

---

## Re-review — 2026-10-14T01:43:00-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 546 — couch: several hubs in one Couch (couch . ../bench): slots assigned to a hub, hub-scoped routing, faded other-hub tabs |
| repo | pair |
| issue file | workshop/issues/000546-couch-several-hubs-in-one-couch-couch-bench-slots-assigned-to-a-hub-hub-scoped-routing-faded-other-hub-tabs.md |
| boundary | whole-issue close |
| milestone | — |
| window | b63f4bd76e3e81d1433cfeda544bb3bed8501522..9effccd231e0121e6c54cded1017eb3979f3db26 |
| command | sdlc close --issue 546 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T01:43:00-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range implements the revised multi-hub contract and approved CLI-only scope. BR-4 is addressed with regression evidence: the test passes at HEAD and fails for both secondary-hub cases when the old classification is restored. No new findings. The review checkout remains unchanged; the disposable checkout was removed.

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      message_service.go:1095 uses shared isHubBinding classification before delivery. TestSupervisionFactsAllHubRecipientsPreserveDeliveryClassification covers main, secondary and worker recipients, including disconnects. Restoring the old hubBinding comparison makes both secondary cases fail with zero facts instead of one; pinned HEAD passes.
```

1. **Strengths**

   - Immutable `HubOwnership` centralizes claim precedence, authorization and revision checks; generated edit tests verify convergence and preservation of unrelated slots.
   - Mutation guards survive preparation and run again inside the effect gate, covering ownership and caller changes.
   - [Delivery classification](/Users/xianxu/Library/Caches/sdlc/review-checkouts/bf7ca6ca95360476/cmd/internal/couchcmd/message_service.go:1093) retains recipient evidence across external IO and disconnect.
   - README and atlas document the new CLI, claims, ownership projection and execution ordering.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Full suites passed for `couchcore`, `couchcmd`, `couchmessage`, `couchtty` and `artifactpath`.
   - Focused race tests passed across the four Couch packages.
   - BR-4’s regression was independently verified red without the fix and green at HEAD.
   - Coverage includes partial persistence, stale authority, delayed delivery, startup failures, single-hub frame compatibility and rendering bounds.
   - Generated runtime assets were required and created only in the disposable checkout. Full `make -k test` and opt-in live conformance were not rerun during this review.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** ownership consumers share the projection; remaining main-hub lookups serve legacy fallbacks or global queries.
   - **ARCH-PURE — pass:** claims, ownership transitions and presentation are pure; filesystem and process work remain in adapters.
   - **ARCH-PURPOSE — pass:** revised routing, mutation isolation, persistence and presentation are delivered; switcher moves are explicitly deferred to #548.
   - **ARCH-MOCK — pass:** existing stateful process, socket and terminal seams are reused, with portable temporary storage and scheduled conformance infrastructure.
   - **ARCH-CONSTRAINTS — pass:** hub, move and claim limits are enforced; rendering introduces no filesystem work.
   - **ARCH-SECURE — pass:** bounded typed parsing, no-follow claim reads and authenticated capabilities preserve trust boundaries.
   - **ARCH-ORDER — pass:** revision checks, effect gating and chained startup adoption handle the tested interleavings.
   - **ARCH-FUNERAL — pass:** claims replace bounded configuration in place; runtime overrides end at restart; existing receipts retain bounded lifetimes.

7. **Plan revision recommendations:** None. The recorded revisions reconcile the original persistence model, CLI-only scope and concurrent queue behavior with the implementation.
