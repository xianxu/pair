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
