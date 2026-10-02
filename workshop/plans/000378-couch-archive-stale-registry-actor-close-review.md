# Boundary Review — pair#378 (whole-issue close)

| field | value |
|-------|-------|
| issue | 378 — couch archive refuses thread whose registered agent is dead |
| repo | pair |
| issue file | workshop/issues/000378-couch-archive-stale-registry-actor.md |
| boundary | whole-issue close |
| milestone | — |
| window | 562c8f53f231d9f0b2e7b633609961bab313b1d3..805f68c5bc53a62c748e7fc506b45804f9e23061 |
| command | sdlc close --issue 378 |
| reviewer | claude |
| timestamp | 2026-10-01T22:21:05-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The fix targets the right root cause. `classifyForAction` was the one registry reader that skipped the liveness probe, and only fresh spawns pruned dead records. Both are fixed: the action path now probes its actors, and the prune moved into the one function every launch goes through (`launchTrackedThread`). The tests failed before the fix, and the old "live" fixtures were only live because of this bug; they now mark the pid alive. Two things stop a clean SHIP. First, the new `Unknown` early return in `classifyForAction` is a second "unknown" rule that bypasses `ClassifyThread`'s documented precedence. Second, one Done-when clause ("classifies as parked") is asserted more weakly than written. Both are cheap to fix.

**1. Strengths**
- `cmd/internal/couchcore/switchagent.go:87-99`: filtering by `actor.Thread != address` also stops the code probing every other thread's actors on the action path.
- `couch.go` `withoutDead` is now the only prune rule. It is pure over a `Registry` value, keeps the "only KNOWN-dead" comment, and is used by spawn and resume because both go through `launchTrackedThread` (ARCH-DRY pass on the prune).
- `registeredActorFixture` merges two copies of the fixture code, and the two old hosted-actor tests now say explicitly that the actor is live.
- `TestResumeReapsTheThreadsDeadActorRecord` reproduces the "two dead warm-reattach records" bug from brain through the real `Resume` call.

**2. Critical:** none.

**3. Important**
- **`switchagent.go:97-98`: the early `Unknown` return duplicates and contradicts `ClassifyThread`'s rule for unproven processes (ARCH-DRY).**
  - `gatherThreadEvidence` already has a channel for unprovable processes (`evidence.Unproven`, `actionableinventory.go:784-793`). `ClassifyThread` (`actionableinventory.go:507-523`) ranks it on purpose: busy and live beat unproven ("a confirmed live proof is not undone by a second incarnation nobody could reach").
  - The new early return ignores that ranking. Any registry actor of the thread with `Unknown` liveness forces `unusable/unknown`, even when another actor is `Live` or a start is in flight. That turns a hosted thread's switch-agent into an "unknown" refusal.
  - It also returns before the `ErrThreadNotFound` check, so a missing thread with leftover unknown records reports `unknown` instead of "not found".
  - Fix: send registry actors with `Unknown` liveness into the same `Unproven` evidence. For example, carry liveness on `LiveTTYObservation`, or add an unproven parameter to `gatherThreadEvidence`. Then the precedence stays in one place.
  - Add a test that pairs a live actor with an unknown one.
  - Every place in this window where unknown liveness is decided: `switchagent.go:97`. Same class, older code: `Couch.Liveness` (`couch.go:1033`) and `observeExactProcess` (`couch.go:999`) implement the same probe twice. They are not in this window, but `withoutDead` uses the first and the classifier uses the second.
- **`action_admission_test.go:TestArchiveIgnoresADeadRegistryActor`: the Done-when clause "a thread … whose record is parked classifies as parked" is not asserted.**
  - The test only checks `state != ThreadLive`.
  - The fixture (`validThreadRecord`) has no park, so it never exercises the "parked" state from the brain report.
  - Fix: either build a parked fixture (e.g. `createParkedThreadInCouch` plus a dead actor) and assert `ThreadParked`, or add a Revisions entry restating the clause as "not live, and archive admits it".

**4. Minor**
- `launch_existing.go:247-251`: if `Save` fails, the rollback removes only the new record. The dead records were already dropped from memory but are still on disk. This is harmless, because they are dead either way, but it is worth a one-line comment.
- `atlas/couch.md:311-314`: "every reader probes `Couch.Liveness`" overclaims. `ownsContinuationHelper` (`continuation.go:381`), `ResolveRef`/`knownTrees` (`couch.go:1073,1123`) and the `owned` loop (`switchagent.go:180`) read records without probing. Some use `observeExactProcess` instead. Reword as "every reader that treats a record as hosting proof probes it".
- `couch.go` `withoutDead`'s comment says it bounds the registry "to live and unprovable actors". In fact the registry is only bounded after the next launch.

**5. Test coverage notes**
- The dead, unknown and live cases are each tested once, all on archive. Switch-agent only has the live case; a dead-actor switch-agent case would pin the second consumer of `classifyForAction`.
- No test mixes live and unknown actors (see the first Important finding).
- `TestPruneKeepsRecordsWhoseLivenessIsUnknown` now covers `withoutDead`'s Unknown branch directly, and the resume test covers the Dead branch.

**6. Architectural notes**
- ARCH-DRY: flagged, because of the parallel unknown rule (the first Important finding). The two existing liveness probes are worth merging later.
- ARCH-PURE: pass. `withoutDead` is a pure transform over `Registry` that takes the probe from `c`, and the IO (`Store.Save`) stays in `launchTrackedThread`.
- ARCH-PURPOSE: pass. Both causes are fixed: the action path reads residue as proof, and the registry grows on every resume. The CLI and switcher now get the same answer from the same evidence. No follow-up is being deferred.

**7. Plan revision recommendations**
- If the Done-when clause is not tightened in the test, add a `## Revisions` entry that restates it as "a dead-actor thread is not classified live and archive admits it".

```findings
findings:
  - id: new
    severity: Important
    family: classification-precedence-single-source
    title: |
      classifyForAction's early Unknown return bypasses ClassifyThread's Unproven precedence (live/busy beat unknown)
    detail: |
      switchagent.go:97-98 returns unusable/unknown for any Unknown registry actor before gatherThreadEvidence/ClassifyThread run, contradicting actionableinventory.go:507-523 (live and busy outrank Unproven) and skipping the ErrThreadNotFound path. Route Unknown registry actors into evidence.Unproven so precedence lives in one place, and add a live+unknown mixed-actor test. Instances in window: switchagent.go:97 only; related older duplication Couch.Liveness (couch.go:1033) vs observeExactProcess (couch.go:999).
  - id: new
    severity: Important
    family: done-when-clause-asserted
    title: |
      Done-when "dead actor + parked record classifies as parked" is asserted only as not-live on a non-parked fixture
    detail: |
      TestArchiveIgnoresADeadRegistryActor uses validThreadRecord (no park) and checks state != ThreadLive. Either build a parked fixture with a dead actor and assert ThreadParked, or revise the Done-when via a Revisions entry.
  - id: new
    severity: Minor
    family: doc-claim-overreach
    title: |
      atlas says every registry reader probes Couch.Liveness; several readers do not
    detail: |
      ownsContinuationHelper (continuation.go:381), ResolveRef/knownTrees (couch.go:1073,1123) and the owned loop (switchagent.go:180) read records without probing, and slotrecovery uses observeExactProcess. Narrow the claim to readers that treat a record as hosting proof. Also, withoutDead's comment that it bounds the registry holds only after the next launch.
```

---

## Re-review — 2026-10-01T22:29:26-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 378 — couch archive refuses thread whose registered agent is dead |
| repo | pair |
| issue file | workshop/issues/000378-couch-archive-stale-registry-actor.md |
| boundary | whole-issue close |
| milestone | — |
| window | 562c8f53f231d9f0b2e7b633609961bab313b1d3..88818fd579e0d24ec39e9f14b401af658e2f5e28 |
| command | sdlc close --issue 378 |
| reviewer | claude |
| timestamp | 2026-10-01T22:29:26-07:00 |
| verdict | SHIP |

## Review

I'm finishing the review. The targeted tests pass, and I checked that the other registry readers pair with incarnation probes.

```verdict
verdict: SHIP
confidence: high
```

All three findings from round 1 are fixed. Unknown registry actors now go into `evidence.Unproven` (`switchagent.go:89-113`), so `ClassifyThread`'s order of precedence is the only one in the code (`actionableinventory.go:495-521`). A live actor still beats an unknown one, and an unknown actor on its own still fails closed as `unusable/unknown`. The dead-actor test now uses a really parked fixture and asserts `ThreadParked`. The atlas claim is narrowed to readers that treat a record as proof the thread is hosted. Every clause in Done-when has a test: dead → parked + archive admits; unknown → unusable/unknown + archive refuses; resume reaps the dead record; the existing hosted fixtures set their pid live. The targeted run `go test ./cmd/internal/couchcore/ -run 'Registry|Archive|SwitchAgent|Resume|Prune'` passes.

1. **Strengths**
   - `withoutDead` (`couch.go:1163`) is now the only prune rule. It sits on `launchTrackedThread`, the single post-claim launch path that spawn, resume, continuation, slot recovery and switch-agent all go through. So deleting the separate prune in `spawnResolved` loses nothing.
   - `registeredActorFixture` removes the duplicate fixture code from the archive and switch-agent hosted tests (ARCH-DRY). The fix also exposed that both old tests only "passed" because of this bug.
   - `TestALiveRegistryActorOutranksAnUnprovableOne` tests the precedence rule directly, not the shortcut it replaced.
   - Known-dead-only pruning is kept, and its test is retargeted to the pure function (`couch_test.go:1617`).

2. **Critical:** none.
3. **Important:** none.
4. **Minor:** none new.
5. **Test coverage:** the three new tests failed before the fix (per the Log) and target the production `classifyForAction`/`Resume` paths. ARCH-PURE passes: `withoutDead` is a function from registry to registry, and IO only happens at the existing `Store.Save` call.
6. **Architecture:** ARCH-DRY passes, with one prune and one classification precedence. ARCH-PURE passes. ARCH-PURPOSE passes: both root-cause effects are fixed, the false "live" in archive/switch and the per-resume registry growth. The overlap between `Couch.Liveness` and `observeExactProcess` existed before this issue and is reasonably out of scope.
7. **Plan revisions:** Spec item 1 says Unknown "fails closed as unusable/unknown". It now reaches that through `ClassifyThread`'s precedence, where a live or busy actor outranks it. The Log records this, so a Revisions note is optional.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Unknown actors join evidence.Unproven (switchagent.go:97-112); TestALiveRegistryActorOutranksAnUnprovableOne pins live-beats-unknown.
  - id: BR-2
    disposition: addressed
    note: |
      TestArchiveIgnoresADeadRegistryActor now uses createParkedThreadInCouch + established binding and asserts ThreadParked.
  - id: BR-3
    disposition: addressed
    note: |
      atlas/couch.md now scopes the probe claim to readers treating a record as hosting proof (classifyForAction); withoutDead comment states the per-launch bound.
```
