# Boundary Review — pair#331 (whole-issue close)

| field | value |
|-------|-------|
| issue | 331 — Starting a thread on a free primary allocates a new slot when numbered slots exist |
| repo | pair |
| issue file | workshop/issues/000331-start-on-free-primary-allocates-new-slot.md |
| boundary | whole-issue close |
| milestone | — |
| window | e1168bc3195314f488f44ee729da66edf0dfb4fc..35902f2523c9875bbd2417f86eaf4f2504390e95 |
| command | sdlc close --issue 331 |
| reviewer | claude |
| timestamp | 2026-09-25T10:35:41-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

**Verdict: SHIP.** The fix goes after the root cause. Before, :0 counted as occupied whenever the repo had any numbered slot (`len(repository.Slots) > 0`) or any thread record in any of the repo's scopes. Now `slotstart.go:48-58` counts :0 as occupied only when a thread record lives in :0's own scope key. Everything the Spec says should stay the same still does. A live or parked :0 still leads to allocating a new slot, and `checkSlotCreation` still blocks when parked work exists. An unreadable record anywhere in the repo still refuses, because that check still uses the repo-wide `scopes` map.

I checked the regression test by putting the base-revision `slotstart.go` into a scratch worktree at head. With the old code the test fails ("free primary resolved to … Number:2, want :0"); with the fix it passes. The `TestManaged|Slot` subset of `couchcore` passes. Every Done-when item is delivered, and the atlas is updated. The findings are minor only.

1. **What was done well**
   - The fix is small and exact. `ResolveRepoScope(primary).Key` is the same key that thread records carry, and `slotRepositoryScopes` builds its keys the same way, so there's no new way of computing the key (`slotstart.go:48`).
   - The unreadable-record guard deliberately keeps using the repo-wide scope set. The Spec requires that behavior to stay unchanged.
   - The regression test covers the exact case from the report. The primary is archived while the slot1 thread is still live, so it proves that a live numbered-slot thread doesn't make :0 occupied. It also checks that the preview has no side effects: `f.host(2)` must not exist (`slotstart_test.go:396`).
   - The test's first assertion (`slot.Thread != primary.Thread`) also covers the other direction. The old `len(Slots) > 0` fallback is gone, so if the primary's record and the computed key ever stopped matching, a create would reuse a :0 that's still occupied. That assertion catches it.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `atlas/workspace-provisioning.md:78`: the edited paragraph leaves one line far longer than the surrounding wrapped lines.
   - `slotstart.go:54-58`: the loop could `break` as soon as it finds a match. This is cosmetic.
   - The test stops at `PrepareStart` (the preview). It never actually starts on :0 after the primary is archived. That is acceptable because the preview carries the target the submission uses, but a spawn assertion would close the loop from start to finish.

5. **Test coverage:** I confirmed the regression test fails without the fix. Existing tests still cover a live primary leading to allocation (the first assertion) and parked work blocking a create (`TestManagedLaunchThenParkBlocksNextCreateButAllowsExistingOpen`). No test covers :0 being free while a numbered slot has a *parked* thread. `checkSlotCreation` only runs when a new slot is allocated, so that path now starts on :0 with nothing blocking it. That matches the Spec, which only blocks *adding a slot*, but there is no test pinning it.

6. **Architecture principles**
   - **ARCH-DRY: pass.** The scope keys come from one function, `launcher.ResolveRepoScope`, and nothing is duplicated.
   - **ARCH-PURE: pass.** The occupancy check is a pure comparison over the snapshot, and the scope resolution is just a hash of the path.
   - **ARCH-PURPOSE: pass.** The Add slot entry in the thread menu uses the same `resolveManagedStart` path, so it's fixed too, as the Spec says.
   - **ARCH-MOCK: pass.** The test uses the existing provisioning fixture with real git in a temporary directory.
   - **ARCH-CONSTRAINTS: pass.** The check is one linear pass over the snapshot.
   - **ARCH-SECURE: pass.** No new untrusted input is introduced; the record scope keys were already validated.
   - **ARCH-ORDER: pass.** The preview still returns an exact target, and submission still refuses if the selection changed.
   - **ARCH-FUNERAL: pass.** The change creates nothing durable. It prevents stray slots from being created.

7. **Plan revisions:** none needed; the plan matches the code.

```findings
findings:
  - id: new
    severity: Minor
    family: test-stops-at-preview
    title: |
      Regression test asserts only the PrepareStart preview, not an actual start on the freed primary
    detail: |
      slotstart_test.go:389 — a spawn after archive, asserting the new thread lands on :0, would cover the path from preview to submission.
  - id: new
    severity: Minor
    family: doc-line-wrap
    title: |
      atlas/workspace-provisioning.md:78 leaves one line far longer than the wrapped lines around it
```
