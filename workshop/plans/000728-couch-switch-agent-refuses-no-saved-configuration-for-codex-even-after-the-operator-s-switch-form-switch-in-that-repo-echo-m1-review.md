# Boundary Review — pair#728 (milestone M1)

| field | value |
|-------|-------|
| issue | 728 — couch --switch-agent refuses 'no saved configuration for codex' even after the operator's switch-form switch in that repo (echo) |
| repo | pair |
| issue file | workshop/issues/000728-couch-switch-agent-refuses-no-saved-configuration-for-codex-even-after-the-operator-s-switch-form-switch-in-that-repo-echo.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 4a7ab2ca1a4c0b0d4aa34e8cada1d8c2917427bc..5a0fb9f504e8672b8491d6637b1828fe0161ee73 |
| command | sdlc milestone-close --issue 728 --milestone M1 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T13:10:02-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1 establishes Couch-owned repository arguments, preserves Pair’s standalone behavior, and publishes settings after successful attachment. SHIP is blocked by missing provenance in the active borrowing flow, stale README instructions, and two reproducible failures in the broader couchcore suite.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: configuration-acceptance-provenance
    title: |
      Borrowed and conflicting arguments reach the form without their required provenance
    detail: |
      cmd/internal/couchcore/switchagent.go:274 now prefills another repository's arguments and returns Borrowed/Conflict metadata at line 309, but cmd/internal/couchtty/menu_switchagent.go:181 renders neither field. The existing reducer displays and accepts these arguments as ordinary parameters; a migration conflict likewise appears as an unexplained empty field. This violates M1's marked-borrowing and explicit-conflict contracts (ARCH-PURPOSE). Display the donor and “Borrowed — not saved for this repository”, expose conflicts, and add reducer/render/acceptance regressions covering own, borrowed, absent and conflicting configurations.
  - id: new
    severity: Important
    kind: defect
    family: user-documentation-traceability
    title: |
      README still directs users to the retired preference authority
    detail: |
      README.md:1331 describes per-path argument precedence, line 1344 identifies numbered .couch/preferences.json as current settings, and line 1355 instructs users to change Couch defaults by launching Pair directly. The implementation now uses Couch's global repository document and removes Pair-default reads, so that documented repair procedure cannot work. README.md is unchanged in the pinned range. Update these passages and the CLI description at line 676 for the delivered M1 behavior; atlas changes alone do not satisfy the README gate.
  - id: new
    severity: Important
    kind: defect
    family: regression-suite-migration
    title: |
      Two existing regressions were left on the retired path-preference contract
    detail: |
      The full couchcore package and an isolated rerun fail TestSpawnPreparedRefusesDriftByFingerprint at startfingerprint_test.go:62 and TestNestedRepositoryStorageRemainsIndependentOfOuterSlot at nested_repository_storage_test.go:97. The former now returns “read repository arguments: no such file or directory”; the latter reads through the obsolete routed helper retained at repopreferences_fixture_support_test.go:58 and finds no preference. Migrate these tests to the production repository-preference seam, retaining their stale-preview/no-effects and nested-repository isolation assertions, then rerun the package.
```

1. **Strengths**

   - Global publication uses a serialized read-modify-write, preserving unrelated repository and agent records: `repopreferences_store.go:57`.
   - The codec preserves unknown members and original legacy bytes while rejecting malformed known fields; focused codec tests passed.
   - Publication follows successful console attachment, with failures retained as warnings. The actual attachment regression and Claude/Codex standalone isolation controls passed.

2. **Critical findings**

   The provenance finding above is reachable through the existing form today. Deferring broader UI work to M2 does not prevent M1’s new borrowing behavior from being displayed and accepted without its required label.

3. **Important findings**

   Repair README’s current-setting instructions and migrate the two failing regressions. Neither requires implementing the deferred restart or CLI-editing features.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed: targeted repository/migration/adoption tests, switch-form tests, launcher isolation controls, actual console-attachment tests, affected-reader inventory, and diff whitespace checks.

   The broader five-package run passed couchcmd, launcher, threadrecord and artifactpath. Couchcore failed the two tests listed above; both reproduced together in an isolated rerun. Existing form tests do not assert borrowed or conflict presentation.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** Couch consumers use the repository authority; Pair-default reads are removed.
   - **ARCH-PURE — pass:** pure decisions and codec are separated from storage and adoption.
   - **ARCH-PURPOSE — flag:** provenance metadata is produced but never presented by its active consumer.
   - **ARCH-MOCK — pass:** existing injected runner/Git seams and private filesystem fixtures are reused.
   - **ARCH-CONSTRAINTS — pass:** document, inventory and argument bounds are enforced; counted migration/borrowing tests passed.
   - **ARCH-SECURE — pass:** migration verifies repository identity and slot containment; malformed input refuses visibly.
   - **ARCH-ORDER — pass:** publication follows attachment, shares one completion, and excludes automatic/warm/recovery learning.
   - **ARCH-FUNERAL — pass:** the bounded settings document replaces itself and follows namespace retention.
   - **ARCH-EVOLVE — pass:** the new repository format retains unknown members across its nested records and rewrites.

   Governing M1 entities exist at the revised file locations. M2/M3 restart and interruption work remains outside this boundary.

7. **Plan revision recommendations**

   Add a `## Revisions` entry making provenance rendering part of M1 whenever borrowing is active. Also record migration of the stale-preview and nested-repository regressions to the canonical store, preserving their original invariants.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"b3dfcf92-2230-45b7-a6a1-8918c2d5e14f","repository":"github.com/xianxu/pair","issue_id":"000728","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/f9c922cf1d8d9e56","reviewed_head":"5a0fb9f504e8672b8491d6637b1828fe0161ee73","slot":"pair:3","requested_at":"2026-10-22T20:03:48.742598Z"},"phase":"finished","pid":30769,"started_at":"2026-10-22T20:03:48.955302Z","finished_at":"2026-10-22T20:10:00.874922Z","session_id":"01a16536-7203-78e0-8f10-ad72537d7922","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.345169625s
head: "5a0fb9f504e8672b8491d6637b1828fe0161ee73"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T13:28:06-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 728 — couch --switch-agent refuses 'no saved configuration for codex' even after the operator's switch-form switch in that repo (echo) |
| repo | pair |
| issue file | workshop/issues/000728-couch-switch-agent-refuses-no-saved-configuration-for-codex-even-after-the-operator-s-switch-form-switch-in-that-repo-echo.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 4a7ab2ca1a4c0b0d4aa34e8cada1d8c2917427bc..32ababba112e3519b0009afb904b3df84acaf724 |
| command | sdlc milestone-close --issue 728 --milestone M1 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T13:28:06-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

All three prior findings are addressed, with negative controls confirming the provenance and fixture corrections. M1’s repository authority, migration, and adoption flow are substantially delivered. One new issue blocks SHIP: optional preference publication can indefinitely block the console’s input and rendering path.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      menu_switchagent.go:184-199 renders own, borrowed/donor, absent and conflict states. The reducer/render/cancel/acceptance regression passes at head; removing the rendering correction through a temporary overlay makes all eight agent/state cases fail.
  - id: BR-2
    disposition: addressed
    note: |
      README.md:676-682 and 1328-1364 now describe Couch repository arguments, legacy migration inputs, independent Pair defaults and post-attachment publication/repair. These passages match repopreferences_store.go:28, run.go:1025-1030 and launcher's trusted-profile guards.
  - id: BR-3
    disposition: addressed
    note: |
      Both tests now use canonical repository preferences and pass in isolation. Restoring their original fixtures through a temporary overlay reproduces both reported failures. The complete couchcore rerun failed an unchanged live PTY teardown test, which passed on isolated rerun.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: optional-console-work-bounds
    title: |
      Optional preference publication can block console input and rendering indefinitely
    detail: |
      couchcmd/run.go:1029 synchronously calls PublishAdoptedConfiguration on the console dispatcher. repopreferences_adoption.go:62 then enters repopreferences_store.go:57's blocking withLock without passing the migration deadline; this ultimately uses blocking flock. A controlled competing-writer check remained blocked after 5.303 seconds and completed only when the lock was released, with no warning. ARCH-CONSTRAINTS: keep optional publication off the console event path, bound coordinator/store lock acquisition, and report timeout through the existing started-but-not-remembered warning without stopping the adopted child. Add a controlled contention regression.
```

1. **Strengths**

   - Repository arguments use the global Couch namespace regardless of slot routing; Pair’s standalone authority remains separate.
   - The codec preserves nested unknown fields and original legacy bytes while enforcing known types, duplicate rejection, and size bounds.
   - Publication follows successful console attachment, carries explicit acceptance intent, and shares one completion across result copies.
   - Provenance coverage exercises both agents across own, borrowed, absent, and conflicting configurations.

2. **Critical findings**

   The blocking publication path above needs bounded lock acquisition and execution outside the console event loop. Preserve warning delivery before operation/receipt completion; avoid introducing an unowned background task.

3. **Important findings**

   None additional.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed: complete couchtty, couchcmd, threadrecord, and launcher packages.
   - Complete couchcore failed `TestTeardownProvesARealPtyTree` on a lease-holder probe deadline. Its isolated rerun passed; the full package run was therefore not green.
   - Passed: both BR-3 tests, affected-reader inventory, production artifact classification, and diff whitespace check.
   - Temporary overlays reproduced BR-1 without its fix, both original BR-3 failures, and publication exceeding its migration deadline. Repository files remained unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** centralized preference policy and store; existing lifecycle reused.
   - **ARCH-PURE — pass:** model/codec tests exercise deterministic logic without IO.
   - **ARCH-PURPOSE — pass:** enumerated Couch consumers use repository arguments; active provenance is rendered.
   - **ARCH-MOCK — pass:** stateful launcher/process/session seams and private filesystem/Git fixtures cover the changed flow.
   - **ARCH-CONSTRAINTS — flag:** optional publication has an unbounded lock wait on the console path.
   - **ARCH-SECURE — pass:** migration verifies identity/containment; argv stays structured; fixtures use private state.
   - **ARCH-ORDER — pass:** claims carry acceptance intent; publication follows adoption and executes once.
   - **ARCH-FUNERAL — pass:** bounded replacement document and existing namespace retention; no new outbox.
   - **ARCH-EVOLVE — pass:** additive fields and nested unknown-member preservation are covered.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining publication’s total latency budget, execution owner, contention/timeout behavior, and controlled coordinator/store-lock tests. State explicitly that optional persistence cannot block console input or rendering.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"642e58ec-c8de-4d74-8fa4-8c319eb6dfe6","repository":"github.com/xianxu/pair","issue_id":"000728","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/ad3a4abaf30575fb","reviewed_head":"32ababba112e3519b0009afb904b3df84acaf724","slot":"pair:3","requested_at":"2026-10-22T20:18:37.80375Z"},"phase":"finished","pid":91108,"started_at":"2026-10-22T20:18:37.915697Z","finished_at":"2026-10-22T20:28:05.507033Z","session_id":"01a16544-02de-72e0-8b78-e0bf30018dc7","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.899645166s
head: "32ababba112e3519b0009afb904b3df84acaf724"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
