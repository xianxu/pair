# Boundary Review — pair#873 (whole-issue close)

| field | value |
|-------|-------|
| issue | 873 — flake: TestFreshStartupRunSerializesLaunchThroughAdoption |
| repo | pair |
| issue file | workshop/issues/000873-flake-testfreshstartuprunserializeslaunchthroughadoption.md |
| boundary | whole-issue close |
| milestone | — |
| window | 2e1e1b4f18d9069ede72d16d1cb6d8b85334dbd5..55c0fe32201270273c9aa5ff360c0b8ba62498e4 |
| command | sdlc close --issue 873 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T09:47:08-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned change fixes both startup lock acquisitions by reusing the existing bounded foreground policy. Admission guards and lock order remain intact. The targeted tests pass, and independently restoring the old policy makes all six contention regressions fail. No blocking findings.

Done-when: current — it covers both lock sites, refusal safety, maintenance behavior, counted effects, and the original Run regression.

```findings
{}
```

1. **Strengths**
   - `startup_creation.go:61–62` reuses the existing wait policy without adding retries or timeout values.
   - `startup_lock_test.go:15` deterministically exercises primary-global, slot-global, and slot-local contention at N=2/4.
   - Cancellation and late journal/current/archive tests assert zero publication; retained bytes remain unchanged.
   - `atlas/couch.md:24` accurately documents the changed admission policy. No new user-facing commands or configuration require README changes.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   - Targeted startup, foreground-writer, and maintenance tests passed.
   - All `TestFreshStartupRun*` tests passed.
   - Temporary compiler overlay restoring both zero-wait arguments failed all six contention cases with the original lock-refusal error.
   - Pinned-range `git diff --check` passed; checkout remains clean. Full repository suite was not rerun.

6. **Architecture**
   - **ARCH-DRY: pass** — existing lock policy and test helpers reused.
   - **ARCH-PURE: pass** — change stays within the existing IO admission boundary.
   - **ARCH-PURPOSE: pass** — both affected acquisitions are corrected and regression-tested.
   - **ARCH-EVOLVE: pass** — no persisted formats change.

7. **Plan revisions:** None required.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"42e2efa7-d810-4708-92a8-7898540c3201","repository":"github.com/xianxu/pair","issue_id":"000873","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/2157d4e78586b6d4","reviewed_head":"55c0fe32201270273c9aa5ff360c0b8ba62498e4","slot":"pair:7","requested_at":"2026-10-24T16:45:44.544908Z"},"phase":"finished","pid":26698,"started_at":"2026-10-24T16:45:44.785239Z","finished_at":"2026-10-24T16:47:06.617199Z","session_id":"01a16ecd-d52d-7f83-b8c3-ee51bed47421","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 11.051453167s
head: "55c0fe32201270273c9aa5ff360c0b8ba62498e4"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000413
