# Boundary Review — pair#627 (whole-issue close)

| field | value |
|-------|-------|
| issue | 627 — Live shortcut fixture races Zellij cache removal during TempDir cleanup |
| repo | pair |
| issue file | workshop/issues/000627-zellij-cache-cleanup-flake.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9e5c1bd33ac31606f68fdc5bc3c1f0485d1d98e0..73e3445407d4f2effedb1583a4eeab86cbd99d46 |
| command | sdlc close --issue 627 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-16T20:03:24-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The teardown fix and snapshot refinement have meaningful regression coverage: restoring either original behavior makes its regression fail. Ten isolated live runs also pass under the race detector. One blocking gap remains: the new teardown relies on process-table completeness that the Linux implementation does not guarantee.

Done-when: current — the regression and repeated reliable-cleanup criteria still cover the revised fixture behavior and preservation of shortcut assertions.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: unknown-process-read-is-not-exit
    title: |
      Linux process-read failures can be mistaken for confirmed daemon exit
    detail: |
      cmd/internal/pairlifecycletest/live_zellij.go:194-206 treats an omitted PID as terminal evidence, but cmd/internal/procutil/table_linux.go:23-29 silently omits every stat-read or parse failure, including failures that do not establish disappearance. The same family has two instances in this window: the initial SnapshotNamed capture at live_zellij.go:145 can omit the daemon and leave servers empty, and the shutdown poll at :194 can falsely declare a captured daemon absent. Both paths can authorize cleanup while the writer remains alive, violating the Spec's explicit unknown-read contract (ARCH-PURPOSE). Preserve unreadable PID rows as unknown or propagate non-disappearance failures, and add regression coverage for stat-read and parse failures at both consumers.
```

1. **Strengths**
   - Shutdown tracks kernel generations and handles zombies and PID reuse.
   - Missing-cache exit 2 is narrowly matched; other shutdown errors propagate.
   - Snapshot retries preserve the first nonempty result and all 13 shortcut assertions.

2. **Critical findings**
   - The Linux completeness gap above. Correct the native observation contract before authorizing cleanup from absence.

3. **Important findings**
   - None.

4. **Minor findings**
   - None.

5. **Test coverage notes**
   - Full `pairlifecycletest` package passed.
   - Focused regression tests passed with `-race -count=3`.
   - Live shortcut conformance passed with `-race -count=10`.
   - Temporary overlays restoring original teardown and snapshot behavior both produced the expected regression failures.
   - Existing tests cover returned table errors, but not errors silently discarded inside Linux enumeration.

6. **Architectural notes**
   - **ARCH-DRY: pass** — reuses `procutil` and existing readiness polling.
   - **ARCH-PURE: pass** — process IO is injected; fixture orchestration remains small.
   - **ARCH-PURPOSE: flag** — Linux uncertainty can still authorize premature cleanup.
   - No new user-facing or architectural surface requires README or atlas changes.

7. **Plan revision recommendations**
   - Add a `## Revisions` entry requiring Linux enumeration to preserve uncertainty, with regression controls for both initial capture and shutdown polling.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"59343053-62c8-43e9-8565-0f9b699e17fb","repository":"github.com/xianxu/pair","issue_id":"000627","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/77a3b7eaa394c528","reviewed_head":"73e3445407d4f2effedb1583a4eeab86cbd99d46","slot":"pair:1","requested_at":"2026-10-17T03:00:32.852243Z"},"phase":"finished","pid":97754,"started_at":"2026-10-17T03:00:32.959925Z","finished_at":"2026-10-17T03:03:17.868765Z","session_id":"01a147cd-d12f-76e3-98c6-7b91191aa522","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.39510075s
head: "73e3445407d4f2effedb1583a4eeab86cbd99d46"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411
