# Boundary Review — pair#708 (whole-issue close)

| field | value |
|-------|-------|
| issue | 708 — couch: put slots' temp folders and Zellij session metadata on the TestScratch RAM disk |
| repo | pair |
| issue file | workshop/issues/000708-couch-put-slots-temp-folders-and-zellij-session-metadata-on-the-testscratch-ram-disk.md |
| boundary | whole-issue close |
| milestone | — |
| window | e51a0e6eda6910f209e68fde2df8f9ea494ec07f..abb07ec7f47283c2241bb339ca43d02e2a2fcc40 |
| command | sdlc close --issue 708 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T11:39:43-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation follows the approved R1/R3 scope, and focused tests pass. Two Important gaps block approval: interactive launches can wait behind an entire cleanup sweep, and tests do not exercise the RAM-enabled production composition. README and atlas updates are present.

1. **Strengths**

   - Metadata fixtures verify live-server no-op, backup preservation, fallback, and interrupted-rename recovery.
   - Cleanup uses nofollow descriptors, checks descendant ages, retains special files, and measures linear inspection work.
   - Immutable owner declarations preserve unknown fields without rewriting them.
   - Executable-boundary tests verify socket pinning survives changed child TMPDIR values.

2. **Critical findings**

   None.

3. **Important findings**

   - **Blocking optional cleanup on launch — ARCH-CONSTRAINTS.** At `cmd/internal/couchscratch/temp.go:75`, `PrepareSlot` acquires the mutex held throughout `Sweep` (`sweep.go:142`). An hourly traversal of another slot’s large tree therefore blocks interactive launch/attach, without cancellation or a wait bound. Use nonblocking admission with inherited-temp fallback, or separate sweep serialization from preparation. Add a controlled-interleaving regression that pauses a sweep and proves preparation returns promptly.

   - **Production composition lacks regression coverage — ARCH-MOCK, ARCH-PURPOSE.** `cmd/internal/couchcmd/scratch_test.go:115` checks callback presence with scratch disabled. Other tests manually compose the helper and runner. No test supplies `scratchProbeFn`, exercises the constructor’s RAM-enabled identity-to-overlay path, or proves the production startup/maintenance call sites run and join. Moreover, `scratch.go:102` hardcodes host server observation. Inject that observation and scheduling through production composition; test enabled/fallback launches, startup ordering, and service shutdown. Removing the wiring at `run.go:595` or `message_service.go:531` should fail those tests, as the plan promises.

4. **Minor findings**

   None.

5. **Test coverage notes**

   The complete `couchscratch` package and focused scratch/socket/launch tests across all five affected packages passed. `git diff --check` passed; tracked files remain unchanged.

   The broader run failed on Unix socket path limits and production-registration fixture errors; its remaining core test process was interrupted. A representative socket failure passed with `TMPDIR=/tmp`. This review does not establish a passing full suite.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared socket resolver replaces both fallback calculations.
   - **ARCH-PURE — pass:** current pure inventory exists and uses direct, IO-free tests.
   - **ARCH-PURPOSE — flag:** production qualification promised by the plan remains incomplete.
   - **ARCH-MOCK — flag:** constructor metadata observation bypasses an injectable test seam.
   - **ARCH-CONSTRAINTS — flag:** sweep duration enters interactive admission through the shared mutex.
   - **ARCH-SECURE — pass:** bounded owner parsing, strict known fields, and nofollow traversal.
   - **ARCH-ORDER — pass structurally:** serial maintenance is context-owned and joined; production lifecycle coverage remains missing.
   - **ARCH-FUNERAL — pass:** ordinary contents have age cleanup; roots/specials end at reboot; permanent backup retention is explicitly approved.
   - **ARCH-EVOLVE — pass:** immutable declarations retain unknown bytes; existing durable formats remain unchanged.

7. **Plan revision recommendations**

   Add a `## Revisions` entry specifying nonblocking scratch admission during cleanup and the production composition regression strategy. Reconcile the promised omission-sensitive wiring checks with actual tests.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: optional-maintenance-blocks-admission
    title: |
      Scratch preparation waits behind the complete fleet sweep
    detail: |
      cmd/internal/couchscratch/temp.go:75 acquires the mutex held throughout Sweep at sweep.go:142, making interactive launch/attach wait for O(S+F) cleanup without cancellation or a wait bound. ARCH-CONSTRAINTS: use nonblocking admission with inherited-temp fallback or separate synchronization, and add a paused-sweep regression proving preparation returns promptly.
  - id: new
    severity: Important
    kind: defect
    family: production-composition-regression-coverage
    title: |
      RAM-enabled production wiring is not exercised through injected dependencies
    detail: |
      cmd/internal/couchcmd/scratch_test.go:115 constructs only scratch-disabled composition; no test supplies scratchProbeFn or proves the production startup and maintenance call sites execute and join. scratch.go:102 hardcodes host server observation. ARCH-MOCK and ARCH-PURPOSE: inject native observation and scheduling, exercise enabled/fallback constructor-to-launch behavior and service shutdown, and prove removing run.go:595 or message_service.go:531 wiring fails the regressions promised by plan line 320.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"d693ece0-4c08-41af-9667-e52e1029a640","repository":"github.com/xianxu/pair","issue_id":"000708","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/6ecb283cc5f529d1","reviewed_head":"abb07ec7f47283c2241bb339ca43d02e2a2fcc40","slot":"pair:3","requested_at":"2026-10-19T18:35:26.000793Z"},"phase":"finished","pid":69525,"started_at":"2026-10-19T18:35:26.207042Z","finished_at":"2026-10-19T18:39:42.670333Z","session_id":"01a15572-741c-73f0-b70d-b63beed7ca4e","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.804907375s
head: "abb07ec7f47283c2241bb339ca43d02e2a2fcc40"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-19T11:58:07-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 708 — couch: put slots' temp folders and Zellij session metadata on the TestScratch RAM disk |
| repo | pair |
| issue file | workshop/issues/000708-couch-put-slots-temp-folders-and-zellij-session-metadata-on-the-testscratch-ram-disk.md |
| boundary | whole-issue close |
| milestone | — |
| window | e51a0e6eda6910f209e68fde2df8f9ea494ec07f..613439241486df66548977b33c25e9d2d6b4f0c9 |
| command | sdlc close --issue 708 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T11:58:07-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The implementation matches the approved R1/R3 contracts. Both prior findings are addressed with reachable regressions that fail when the fixes are removed. No new blocking defects found. Focused verification passed; this review does not establish a passing full suite.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      temp.go:83 uses TryLock before probing or resolving identity. Restoring blocking acquisition made TestPrepareFallsBackWhileSweepIsPaused fail; the restored regression passes.
  - id: BR-2
    disposition: addressed
    note: |
      Injected probe, server observation, clock and ticks exercise enabled/fallback constructor behavior, startup effects and maintenance cancellation/join. Removing run.go startup or message_service.go maintenance independently failed the corresponding regression; restored regressions pass.
```

1. **Strengths**

   - Admission falls back during maintenance without resolving optional identity (`cmd/internal/couchscratch/temp.go:82`).
   - Cleanup checks descendant ages, excludes special files and revalidates before deletion (`cmd/internal/couchscratch/sweep.go:50`).
   - Metadata setup preserves paths on live or unavailable server evidence and retains the original backup (`cmd/internal/couchscratch/metadata.go:101`).
   - README and atlas document age-loss risk, relaunch requirements, fallback and socket pinning.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Complete `couchscratch` tests passed.
   - Relevant race checks passed across scratch, command composition, core launch/identity, launcher and executable socket pinning.
   - All three omission mutations produced regression failures; disposable checkouts were removed.
   - The broader run encountered socket-path-length and production-registration failures and was interrupted. A broad race selection exceeded 90 seconds in unrelated metadata tests; the narrowed core scratch race run passed. No full-suite success is claimed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Native socket fallback consumers share one resolver.
   - **ARCH-PURE — pass:** Policy functions are directly tested; filesystem effects remain in integration code. Current concept classifications match implementation.
   - **ARCH-PURPOSE — pass:** Both launch boundaries, startup cleanup, hourly maintenance and R1 routing are wired.
   - **ARCH-MOCK — pass:** Production composition exposes observation/scheduling seams; tests use portable private filesystem state and the existing fake runner.
   - **ARCH-CONSTRAINTS — pass:** Nonblocking admission, bounded identity lookup, serial sweeps and counted linear work are covered.
   - **ARCH-SECURE — pass:** Owner declarations receive strict validation; traversal refuses symlinks and tests use isolated roots.
   - **ARCH-ORDER — pass:** Metadata decisions use an explicit action model; maintenance cancellation and joining are exercised.
   - **ARCH-FUNERAL — pass:** Ordinary contents have age cleanup; roots and special trees have reboot lifetime; permanent backup retention is explicitly approved.
   - **ARCH-EVOLVE — pass:** Accepted owner declarations remain byte-preserved, including unknown nested fields; malformed known fields are rejected.

7. **Plan revision recommendations:** None. The final revisions describe the implemented inventory, lifecycle and concurrent-deletion limits.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"4dbb2bef-35c5-453b-81be-37e35e918bd3","repository":"github.com/xianxu/pair","issue_id":"000708","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/4a5ce43430719611","reviewed_head":"613439241486df66548977b33c25e9d2d6b4f0c9","slot":"pair:3","requested_at":"2026-10-19T18:53:28.200519Z"},"phase":"finished","pid":61024,"started_at":"2026-10-19T18:53:28.360605Z","finished_at":"2026-10-19T18:58:06.419679Z","session_id":"01a15582-f704-7230-b440-8fae0fb4211c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.012091667s
head: "613439241486df66548977b33c25e9d2d6b4f0c9"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-19T12:29:38-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 708 — couch: put slots' temp folders and Zellij session metadata on the TestScratch RAM disk |
| repo | pair |
| issue file | workshop/issues/000708-couch-put-slots-temp-folders-and-zellij-session-metadata-on-the-testscratch-ram-disk.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8a0cf2d5c605c8f5fad5abd169f04a835749a42c..19fb9c730fe0a2a760350df2f2c84ab371860a5c |
| command | sdlc close --issue 708 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T12:29:38-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The naming consolidation is correct, and both previously disposed fixes remain intact. This interdiff introduces one test defect: honoring a long `TMPDIR` makes the socket fixture exceed the native address limit and fail before testing retention.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      PrepareResolvedSlot retains TryLock before probe/identity resolution; the paused-sweep and unavailable-probe regressions pass.
  - id: BR-2
    disposition: addressed
    note: |
      Injected constructor, production startup, and maintenance cancellation/join regressions remain present and pass.
findings:
  - id: new
    severity: Important
    kind: defect
    family: native-fixture-address-budget
    title: |
      Socket fixture now fails with long inherited TMPDIR
    detail: |
      cmd/internal/couchscratch/os_test.go:26 changes the socket fixture to inherit TMPDIR without budgeting its derived address. Under this review's 84-byte TMPDIR, the socket path is approximately 117 bytes; TestSpecialTopLevelNodesRetained fails at line 47 with "invalid argument", including under -race. The same test passes with TMPDIR=/tmp. Use an owned short socket fixture with a justified lint exception, following cmd/internal/couchmessage/transport_test.go:20, and cover long TMPDIR so the test reaches its retention assertions.
```

1. **Strengths**

   - `policy.go:35` delegates slot naming to `couchcore.EnvName`, preserving the exact basename contract.
   - Both the slot-state coverage audit and repository lint catalog pass.
   - Spec, Done-when, and the implementation plan remain unchanged.

2. **Critical findings:** None.

3. **Important findings:** The socket fixture address-budget defect above.

4. **Minor findings:** None.

5. **Test coverage notes**

   Targeted prior-fix regressions passed. The complete scratch race run failed at the socket bind; the isolated socket test passed with short `TMPDIR`. No full-suite success is claimed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Reuses the existing basename authority.
   - **ARCH-PURE — pass:** Naming stays pure; fixture IO remains at the test boundary.
   - **ARCH-PURPOSE — flag:** The fixture correction must preserve execution of the special-file retention test under inherited temporary roots.
   - **ARCH-EVOLVE — pass:** No persisted-format or basename changes.

   No new user-facing or architectural surface requires README or atlas updates.

7. **Plan revision recommendations:** None; repair the fixture within the existing scope.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"df3a0902-e468-461a-85f8-c6bb567d8d79","repository":"github.com/xianxu/pair","issue_id":"000708","recipe":"fix-review","round":3,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/4ac1d0b93b0a368e","reviewed_head":"19fb9c730fe0a2a760350df2f2c84ab371860a5c","slot":"pair:3","requested_at":"2026-10-19T19:27:39.05945Z"},"phase":"finished","pid":81633,"started_at":"2026-10-19T19:27:39.368445Z","finished_at":"2026-10-19T19:29:16.51609Z","session_id":"01a155a2-434c-7c30-94e7-1b636b6d3498","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.382543291s
head: "19fb9c730fe0a2a760350df2f2c84ab371860a5c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-19T12:32:39-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 708 — couch: put slots' temp folders and Zellij session metadata on the TestScratch RAM disk |
| repo | pair |
| issue file | workshop/issues/000708-couch-put-slots-temp-folders-and-zellij-session-metadata-on-the-testscratch-ram-disk.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8a0cf2d5c605c8f5fad5abd169f04a835749a42c..d7593dc70df0e65f516ef7baa1b45e4b1c632514 |
| command | sdlc close --issue 708 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T12:32:39-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

BR-3 is addressed: the real socket/FIFO retention test now passes with a deliberately long `TMPDIR`. Restoring inherited-temp allocation in a disposable copy makes that regression fail at socket binding with `invalid argument`. The interdiff introduces no blocking defects, scope changes, or new documentation surface.

```findings
dispose:
  - id: BR-3
    disposition: addressed
    note: |
      os_test.go:26-32 sets a long TMPDIR and allocates an owned short root with a justified lint exception. The complete scratch race suite passes. Replacing the short root with inherited-temp allocation makes TestSpecialTopLevelNodesRetained fail at socket binding; the disposable copy was removed.
  - id: BR-1
    disposition: addressed
    note: |
      PrepareResolvedSlot retains TryLock before probing or resolving identity. The paused-sweep and unavailable-probe regressions pass in the complete scratch race suite.
  - id: BR-2
    disposition: addressed
    note: |
      Production startup and maintenance callsites remain intact. Constructor RAM/fallback, startup-before-failure, and maintenance cancellation/join regressions pass.
```

1. **Strengths**

   - The fixture exercises real native special files and reaches its retention assertions under long `TMPDIR` (`cmd/internal/couchscratch/os_test.go:26-61`).
   - Slot naming delegates to `couchcore.EnvName`, preserving the exact existing basename contract (`cmd/internal/couchscratch/policy.go:35`).
   - Both the slot-state coverage audit and repository lint catalog pass.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Complete `couchscratch` race suite passed.
   - Targeted production composition, startup, maintenance, slot-state audit, and lint tests passed.
   - The inherited-root mutation failed as expected; its disposable copy was removed.
   - `git diff --check` passed. No full-repository suite success is claimed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Reuses the existing slot basename authority and removes the redundant formatter.
   - **ARCH-PURE — pass:** Naming remains deterministic; native filesystem operations remain in integration tests.
   - **ARCH-PURPOSE — pass:** The correction restores the special-file retention test under the environmental condition that previously prevented it from running.
   - **ARCH-EVOLVE — pass:** No persisted format or basename meaning changes.

   No new user-facing or architectural surface requires README or atlas changes.

7. **Plan revision recommendations:** None. Spec, Done-when, and the implementation plan remain unchanged.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"10cdbc8a-ec15-4385-9b0c-2357c0d9a543","repository":"github.com/xianxu/pair","issue_id":"000708","recipe":"fix-review","round":4,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/bc708be701c3ee3b","reviewed_head":"d7593dc70df0e65f516ef7baa1b45e4b1c632514","slot":"pair:3","requested_at":"2026-10-19T19:31:30.632406Z"},"phase":"finished","pid":19531,"started_at":"2026-10-19T19:31:30.929207Z","finished_at":"2026-10-19T19:32:38.515889Z","session_id":"01a155a5-cbcd-7e90-b86a-fb55fd5875fc","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.322937625s
head: "d7593dc70df0e65f516ef7baa1b45e4b1c632514"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
