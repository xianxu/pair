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
