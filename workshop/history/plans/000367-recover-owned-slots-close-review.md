# Boundary Review — pair#367 (whole-issue close)

| field | value |
|-------|-------|
| issue | 367 — Recover local slots from durable issue ownership |
| repo | pair |
| issue file | workshop/issues/000367-recover-owned-slots.md |
| boundary | whole-issue close |
| milestone | — |
| window | aad9bfa2f104af768d182b31c3357ee45a3eea6a..da19d16e4f1e35c839d00321eab157f86849208e |
| command | sdlc close --issue 367 |
| reviewer | claude |
| timestamp | 2026-10-05T10:45:56-07:00 |
| verdict | SHIP |

## Review

I've finished checking the prior findings and the window's last commit; writing the verdict now.

```verdict
verdict: SHIP
confidence: medium
```

Three of the four open findings are fixed: BR-15, BR-16 and BR-17. BR-8 is still open as a Minor deferral, and the code says it is deferred. This round I checked the fixes and the commits added since the last round. Those are the M3 drop, which is recorded as a dated Revision and removed from Done-when, and the side-quest `da19d16e`, which splits store reads by caller class. Nothing new blocks the close.

1. **Strengths**
   - **Slot-level reboot guard.** Each dependency member now gets its own tree verdict through `memberTree` (`recoverplan.go:1169`), whatever its claim verdict. `foldDependencies` folds those to the worst tree, and `slotGitUnknown` and `slotDirty` read the folded value (`recoverplan.go:485-491`). So a dependency with an unread state or dirty files can no longer get a reboot step without a hold or a note.
   - **Invariant test.** The property-style check at `recoverplan_test.go:626-635` states the reboot invariants independently of the rule table. It does not just repeat the implementation (ARCH-ORDER).
   - **Note ownership.** `withRuleA` clones the notes it is given, and `restoreWorkspaceDecision` builds them with `slices.Concat` (`recoverplan.go:679-697`). The aliasing bug is gone in structure, not just at the one call site.
   - **Lock ordering by caller class.** `da19d16e` makes it explicit: a read taken while another lock is held uses `withNestedPreviewLock` and waits zero; a foreground read waits up to `storeReadLockWait`. The commit names the regression that fails without it.
   - **M3 drop.** It is recorded as a Revision with its delta, and the Done-when was changed with it.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `ThreadStore.withLock` sends read-only stores to `withPreviewLock`, which waits up to 1 s (`threadstore.go:157`). Any read-only `withLock` caller that holds another lock would wait under it. Today the only known nested sites are converted, but a future nested caller can't be seen from the call site.

5. **Test coverage**
   - The reboot-guard cases at `recoverplan_test.go:410` and `recoverplan_test.go:420` pin BR-15: a claimed dependency with an unread base, and a dirty claimed dependency.
   - The invariant sweep at `recoverplan_test.go:626-635` covers the class, not only those two sites.
   - BR-16 has no test of its own, but the fix makes the bug impossible by construction, so it is acceptable at Minor.

6. **Architecture, per principle**
   - **ARCH-DRY:** pass, apart from BR-8 (the terminal-status list restated from ariadne).
   - **ARCH-PURE:** pass. `DeriveRecoverPlan` and `classifyRecover` are pure.
   - **ARCH-PURPOSE:** pass. M3 was moved to #387 with an explicit revision, and its content is not the issue's stated purpose.
   - **ARCH-MOCK:** pass. There is a recover-plan fake plus sdlc fixtures, and a conformance cadence was addressed under BR-2.
   - **ARCH-CONSTRAINTS:** pass. Store waits are bounded and nested reads take a zero wait.
   - **ARCH-SECURE:** pass. Callers are authenticated by Couch liveness, and a request with no identity is `invalid-request`.
   - **ARCH-ORDER:** pass. Receipts follow a state machine, and lock ordering is now explicit.
   - **ARCH-FUNERAL:** pass. Receipts live only in memory.
   - **For upcoming work:** #387's slot reconciler should take on the dependency fold (`foldDependencies`) rather than restate it.

7. **Plan revisions:** none.

```findings
dispose:
  - id: BR-8
    disposition: not-addressed
    note: |
      Still restated at recoverplan.go:753; deferral is recorded in code (sdlc should emit terminality). Minor, non-blocking.
  - id: BR-15
    disposition: addressed
    note: |
      memberTree computes per-member tree regardless of claim verdict; slotGitUnknown/slotDirty fold DepTree; tests at recoverplan_test.go:410,420 and invariant sweep :626-635.
  - id: BR-16
    disposition: addressed
    note: |
      restoreWorkspaceDecision uses slices.Concat and withRuleA clones notes (recoverplan.go:679-697); aliasing impossible by construction.
  - id: BR-17
    disposition: addressed
    note: |
      atlas/couch.md:403-405 now states a no-identity request is invalid-request and only a failed caller check is unavailable, matching revision (d).
findings:
  - id: new
    severity: Minor
    family: nested-read-waits-under-held-lock
    title: |
      ThreadStore.withLock routes read-only stores to the waiting withPreviewLock, so a nested read-only caller would wait under a held lock
    detail: |
      threadstore.go:157. da19d16e fixed the known nested sites by caller class; the read-only withLock path picks the wait implicitly. Consider making the wait an explicit parameter there.
```
