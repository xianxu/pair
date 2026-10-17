# Boundary Review — pair#624 (whole-issue close)

| field | value |
|-------|-------|
| issue | 624 — Something rebuilds pair:0's bin/couch during other slots' work, deleting it for seconds (sends fail) |
| repo | pair |
| issue file | workshop/issues/000624-something-rebuilds-pair-0-s-bin-couch-during-other-slots-work-deleting-it-for-seconds-sends-fail.md |
| boundary | whole-issue close |
| milestone | — |
| window | fe4071b2fbf68bf1f6a0a0d8d2c22f0d8376fc12..a7bc095aadaf6c196ca2ada0136c137f35be9a34 |
| command | sdlc close --issue 624 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-16T23:09:44-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The atomic publisher works in the paths exercised here, with meaningful regression coverage and matching documentation. The remaining blocker is verification of the generic scanner’s opt-out behavior: its dependency is absent from this checkout.

Done-when: current — the two clauses still cover worktree isolation and atomic publication under the revised scope.

1. Strengths

   - All three explicit binary targets reuse one publisher, avoiding duplicated publication logic.
   - Same-filesystem staging preserves the old executable during compilation and failed builds.
   - README and atlas document the changed build behavior and distinguish the unknown historical caller from proven writers.

2. Critical findings

   None.

3. Important findings

```findings
findings:
  - id: new
    severity: Important
    kind: unverifiable
    family: dependency-build-path-verification
    title: |
      Generic scanner opt-out remains unverified in this checkout
    detail: |
      tests/build-go-atomic-test.py:37-39 conditionally omits Makefile.workflow; inspection with ls reported that dependency absent. All eight tests pass here, but in a disposable fixture the root-build regression also passes after removing both new sentinels, so it cannot verify that the generic scanner stops publishing directly. The complete consumer family is cmd/pair-go, cmd/couch, and cmd/pair-launch-helper; the latter two receive new sentinels in this window. Provide a woven checkout with the actual scanner, a passing head run, and a regression run showing that removing the new sentinels makes the root-build test fail. This answers the ARCH-PURPOSE verification gap without assuming another repository's scanner conventions.
```

4. Minor findings

   None.

5. Test coverage notes

   - All eight new tests passed.
   - Restoring base build recipes in a disposable fixture produced five failed assertions covering all three direct targets, failed-build preservation, and root publication.
   - A real Go build through the helper produced a runnable executable and left no staging directories.
   - The full repository suite was not rerun; recorded results were not treated as independently verified evidence.

6. Architectural notes

   - **ARCH-DRY: pass.** One shared publication helper serves all explicit targets.
   - **ARCH-PURE: pass.** The change is a thin build IO boundary; no business logic was added.
   - **ARCH-PURPOSE: flag.** Explicit publishers are verified; exclusion of generic publishers needs the evidence above.

7. Plan revision recommendations

   None currently. The implementation matches the revised plan structurally; the outstanding item is verification evidence.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"2243c32f-9de2-4872-be38-39475b8dcaf1","repository":"github.com/xianxu/pair","issue_id":"000624","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/63bd7c505e45b320","reviewed_head":"a7bc095aadaf6c196ca2ada0136c137f35be9a34","slot":"pair:7","requested_at":"2026-10-17T06:08:10.020015Z"},"phase":"finished","pid":56056,"started_at":"2026-10-17T06:08:10.188457Z","finished_at":"2026-10-17T06:09:43.378808Z","session_id":"01a14879-97b7-7202-b980-05d9312a7c94","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.032005167s
head: "a7bc095aadaf6c196ca2ada0136c137f35be9a34"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-16T23:12:26-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 624 — Something rebuilds pair:0's bin/couch during other slots' work, deleting it for seconds (sends fail) |
| repo | pair |
| issue file | workshop/issues/000624-something-rebuilds-pair-0-s-bin-couch-during-other-slots-work-deleting-it-for-seconds-sends-fail.md |
| boundary | whole-issue close |
| milestone | — |
| window | fe4071b2fbf68bf1f6a0a0d8d2c22f0d8376fc12..8e88e3629694568666a4b6a5141c5801c65e2935 |
| command | sdlc close --issue 624 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-16T23:12:26-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation satisfies the revised contract. Atomic publication, failure preservation, worktree isolation, and generic-scanner exclusion are verified. BR-1’s missing measurement is now independently reproduced; no new findings.

Done-when: current — both clauses still cover the revised scope: worker preparation leaves the primary binaries untouched, and builds publish completed binaries atomically.

```findings
dispose:
  - id: BR-1
    disposition: withdrawn
    note: |
      Independently verified retained sources match pinned head and actual worker Makefile.workflow matches SHA256 b816fcb7c31d3426775ea583e802835c756756a97c241f4610be51d7e1b345ec. Reproduction passed with three staged writes; removing Couch’s sentinel, helper’s sentinel, or both failed with respectively four, four, and five writes, including the corresponding direct outputs. This supplies the missing scanner measurement for the complete owned-binary family.
```

1. Strengths

   - All three explicit targets share one publisher (`Makefile.local:524–535`).
   - Same-filesystem staging preserves existing executables during compilation and failure (`scripts/build-go-atomic.sh:14–19`).
   - README and atlas explain publication behavior and accurately retain uncertainty about the historical caller.

2. Critical findings

   None.

3. Important findings

   None.

4. Minor findings

   None.

5. Test coverage notes

   - All eight new tests passed.
   - Restoring base recipes privately caused four regression failures covering all three direct targets and failed-build preservation.
   - Independently reproduced all four scanner variants described above.
   - A real Go build produced a runnable executable and left no staging directory.
   - Full repository suite was not rerun during this review.

6. Architectural notes

   - **ARCH-DRY: pass.** One publication helper serves every owned binary.
   - **ARCH-PURE: pass.** Changes remain within a thin build IO boundary, tested with an injected compiler.
   - **ARCH-PURPOSE: pass.** Enumerated consumers—Pair, Couch, and launch helper—all use the publisher and exclude the generic writer. Preparation matches `.sdlc/review-prepare.json`.

7. Plan revision recommendations

   None. The implementation matches the revised plan; no Core concepts table requires reconciliation.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7364a85d-c457-4eb3-ba4b-5d148f825f41","repository":"github.com/xianxu/pair","issue_id":"000624","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/0565d3ba626c2d86","reviewed_head":"8e88e3629694568666a4b6a5141c5801c65e2935","slot":"pair:7","requested_at":"2026-10-17T06:11:37.119988Z"},"phase":"finished","pid":85495,"started_at":"2026-10-17T06:11:37.217763Z","finished_at":"2026-10-17T06:12:25.55872Z","session_id":"01a1487c-bf64-75e3-81b6-2d6e3f6a82ff","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.494336625s
head: "8e88e3629694568666a4b6a5141c5801c65e2935"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
