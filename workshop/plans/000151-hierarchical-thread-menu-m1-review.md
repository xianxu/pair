# Boundary Review — pair#151 (milestone M1)

| field | value |
|-------|-------|
| issue | 151 — couch: hierarchical work-thread menu |
| repo | pair |
| issue file | workshop/issues/000151-hierarchical-thread-menu.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 9157998f17a19d3794efe5f5f0cb376752674df4..cf3321f5d639c0956974878c2657ad50aee665e8 |
| command | sdlc milestone-close --issue 151 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-08-30T20:57:48-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1 establishes solid pure projection, fingerprint, grant, and prepared-admission foundations, and its focused package/race suites pass. It cannot close yet: making `start.token` mandatory broke the existing Console start path, the actionable projector accepts structurally corrupt “verified park” evidence, and the repository’s exhaustive source-inventory test rejects all three new production files.

## 1. Strengths

- `StartGrantStore` is bounded, mutex-owned, one-attempt authority with defensive cloning and explicit expiry/collision behavior ([startgrant.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/startgrant.go:32)).
- The start fingerprint uses a versioned, length-delimited encoding and covers every current `PolicyResult` field plus preference/default evidence ([startresolution.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/startresolution.go:104)).
- `SpawnPrepared` consumes the grant once and refuses changed evidence before allocation or fork ([couch.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/couch.go:173)).
- Candidate policy is passed directly into prepared admission rather than reread after acceptance ([admission.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/admission.go:80)).
- The atlas explains the new token lifecycle, limits, and diagnostic/actionable inventory distinction.

## 2. Critical findings

1. [ops.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/ops.go:124), [console.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console.go:862), [console.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console.go:1183) — `start` now requires an implicit token, but the existing Console still dispatches `start` with only `path`. Both existing Console regressions time out because validation rejects the call before their executor runs. Update this consumer to perform `prepare-start` followed by token-bound `start`, or stage the schema change atomically with the replacement controller. Preserve tests that exercise the real declared dispatcher. This flags ARCH-DRY and ARCH-PURPOSE.

2. [actionableinventory.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/actionableinventory.go:87) — the authoritative projector treats any non-nil `VerifiedPark` as exact verified resume evidence and never validates the `ThreadRecord`. The tests themselves use a zero-identity, zero-attempt `VerifiedPark`, which `ValidateThreadRecord` rejects. Consequently malformed live records and corrupt parked records can be projected despite the Spec’s fail-closed contract. Validate records before projection and add malformed-record regressions using valid positive fixtures. This flags ARCH-PURPOSE.

## 3. Important findings

1. [manifest.go](/Users/xianxu/workspace/pair/cmd/internal/artifactpath/manifest.go:481) — the exhaustive production-source inventory was not updated for `actionableinventory.go`, `startgrant.go`, or `startresolution.go`. Add the appropriate classifications; `TestProductionArtifactReferencesAreExactlyClassified` currently fails on all three.

2. [atlas/couch.md](/Users/xianxu/workspace/pair/atlas/couch.md:28), [run.go](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/run.go:415) — the atlas says the ordinary switcher consumes `ActionableThreadInventory`, but production still wires `ThreadInventory` into the transitional panel. Document the M1 surface as available but awaiting M3 Console integration, or wire it now. This flags ARCH-PURPOSE.

## 4. Minor findings

None beyond the plan revisions below.

## 5. Test coverage notes

Clean-archive results:

- `go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchcmd -count=1`: PASS.
- Focused race suite for the changed M1 behavior: PASS.
- Existing Console start regressions: FAIL reproducibly.
- Exhaustive artifact source inventory: FAIL reproducibly.
- Repository-wide testing also encountered unrelated archive/sandbox limitations involving pinned Git-object tests and `/bin/ps`; those are not raised as M1 findings.

## 6. Architectural notes for upcoming work

- ARCH-DRY: Flag — the shared `start` declaration changed without sweeping the Console consumer.
- ARCH-PURE: Pass — resolution/projection logic is separated from snapshot, policy, entropy, and launch I/O.
- ARCH-PURPOSE: Flag — the token contract breaks an existing start surface, and malformed lifecycle evidence does not fail closed.
- ARCH-MOCK: Pass — new external behavior remains behind existing injected policy, runner, clock, entropy, and store seams.
- ARCH-CONSTRAINTS: Pass for M1 — grant capacity, TTL, collision attempts, and concurrency are bounded and tested. UI latency evidence properly remains later-milestone work.

## 7. Plan revision recommendations

Append a `## Revisions` entry recording:

- M1 must preserve the transitional Console by migrating its start consumer alongside the required-token schema.
- `ActionableThreadInventory` is introduced in M1 but not consumed by the ordinary switcher until M3; adjust the M1 wording and atlas accordingly.
- Task 11’s context steps are stale: `OperationCall.Context` and start propagation already exist, so its RED expectation should target the remaining lifecycle/runner consumers rather than a contextless operation seam.

```findings
findings:
  - id: new
    severity: Critical
    family: shared-operation-consumer-sweep
    title: |
      Required start tokens break the existing Console start consumer
    detail: |
      start now requires an implicit token, but Console still dispatches start with only path; both existing Console start regressions time out. Sweep every shared-operation consumer and make Console perform prepare-start followed by token-bound start, or stage the schema change atomically.
  - id: new
    severity: Critical
    family: lifecycle-evidence-validation
    title: |
      The actionable projector accepts corrupt verified-park evidence
    detail: |
      ProjectActionableThreads checks only that VerifiedPark is non-nil and never validates the durable record, so a zero identity/attempt and other malformed records can become actionable despite the fail-closed Spec. Validate the record and add malformed live and parked regressions using valid positive fixtures.
  - id: new
    severity: Important
    family: exhaustive-production-source-inventory
    title: |
      New production files are absent from the exhaustive source inventory
    detail: |
      TestProductionArtifactReferencesAreExactlyClassified rejects actionableinventory.go, startgrant.go, and startresolution.go. Classify all three in the artifact authority inventory.
  - id: new
    severity: Important
    family: documentation-current-state-accuracy
    title: |
      Atlas claims the ordinary switcher already consumes the actionable inventory
    detail: |
      The atlas describes ActionableThreadInventory as the current switcher source, while run.go still wires ThreadInventory into the transitional panel. Document the milestone staging accurately or complete the wiring.
```

---

## Re-review — 2026-08-30T21:10:24-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 151 — couch: hierarchical work-thread menu |
| repo | pair |
| issue file | workshop/issues/000151-hierarchical-thread-menu.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 54ebd13707fe942c8580db7b6a90b9b30b7ab36d..714b627d9dcc2c1746eb63e7dce9695bcd86fd77 |
| command | sdlc milestone-close --issue 151 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-08-30T21:10:24-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation and three prior correctness fixes are verified, including mutation checks. BR-4 remains open: although the atlas now describes M1 staging accurately, the project milestone still falsely claims the ordinary switcher consumes the actionable inventory. The pinned review range itself changes only the recorded actual from 4.74h to 5.14h.

## Strengths

- BR-1 is addressed: both production start consumers use `prepare-start` followed by token-bound `start`. Reverting the Console fix makes both panel regressions fail.
- BR-2 is addressed: [actionableinventory.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/actionableinventory.go:63) validates complete records before projection. Reverting it makes malformed-live and malformed-park tests fail.
- BR-3 is addressed: all three new production files are classified in [manifest.go](/Users/xianxu/workspace/pair/cmd/internal/artifactpath/manifest.go:501). Reverting those entries makes the exhaustive inventory test fail.
- The atlas accurately distinguishes the M1 authority from the planned M3 consumer wiring at [atlas/couch.md](/Users/xianxu/workspace/pair/atlas/couch.md:30).

## Critical findings

None.

## Important findings

- **BR-4 — not addressed (`documentation-current-state-accuracy`).** [workshop/projects/couch.md](/Users/xianxu/workspace/pair/workshop/projects/couch.md:422) says Couch exposes proof-bearing rows “to the ordinary switcher,” but [run.go](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/run.go:415) still supplies raw `ThreadInventory`, as the corrected atlas acknowledges.

  **This is the 2nd finding in family `documentation-current-state-accuracy`.** Do not fix only this sentence. Establish and sweep the rule that every current-state summary of a staged consumer migration must distinguish “authority exists” from “consumer is wired.” Enumerate at least atlas, project milestone, issue log, plan revisions, and README, and add a regression covering the declared/current consumer stage. This is also an ARCH-PURPOSE class-sweep failure.

## Minor findings

None.

## Test coverage notes

Fresh verification passed:

- `go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/couchtty ./cmd/internal/artifactpath -count=1`
- Focused M1 race tests passed.
- Mutation checks proved BR-1, BR-2, and BR-3 regressions go red without their production fixes.
- No test guards BR-4’s staging accuracy; existing atlas tests check operation presence, not which inventory the switcher consumes.

## Architectural notes for upcoming work

- **ARCH-DRY — pass:** shared projection and operation declarations remain authoritative.
- **ARCH-PURE — pass:** projection and resolution/fingerprint logic are pure; snapshot and dispatch wrappers are thin.
- **ARCH-PURPOSE — flag:** M1 code fulfills its scoped authority work, but the documentation fix handled one instance rather than the complete current-state-documentation class.
- **ARCH-MOCK — pass:** external policy, runner, and terminal behavior stays behind existing injected seams and stateful fakes.
- **ARCH-CONSTRAINTS — pass for M1:** grants are explicitly bounded by capacity, TTL, and collision attempts; later UI performance commitments are not claimed at this boundary.

## Plan revision recommendations

None. The plan’s existing M1 boundary revision accurately says M1 exposes the authority and M3 adopts it. The project milestone record and its missing cross-document guard need correction.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Both production consumers now perform prepare-start then token-bound start; reverting the Console fix makes both panel start regressions fail.
  - id: BR-2
    disposition: addressed
    note: |
      Projection validates the complete durable record; reverting that validation makes malformed live and parked regressions fail.
  - id: BR-3
    disposition: addressed
    note: |
      All three sources are classified; reverting the entries makes the exhaustive production-source inventory test fail.
  - id: BR-4
    disposition: not-addressed
    note: |
      The atlas was corrected, but the project milestone still claims the ordinary switcher consumes actionable inventory while run.go wires raw ThreadInventory, and no regression guards this staging claim.
```
