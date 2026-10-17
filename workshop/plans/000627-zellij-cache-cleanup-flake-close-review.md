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

---

## Re-review — 2026-10-16T21:14:47-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 627 — Live shortcut fixture races Zellij cache removal during TempDir cleanup |
| repo | pair |
| issue file | workshop/issues/000627-zellij-cache-cleanup-flake.md |
| boundary | whole-issue close |
| milestone | — |
| window | 628ab0cadba9a7027130383d886fff443708c948..f00c302c6c0937bc6101e83b79208e9a3cd87fa4 |
| command | sdlc close --issue 627 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-16T21:14:47-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range satisfies the revised Spec and Plan. BR-1 is addressed by preserving unreadable Linux PIDs and retaining uncertainty at both teardown consumers. Regression tests fail when the old behavior is restored. Fresh race-enabled package tests and ten isolated live shortcut runs pass. No new blocking findings remain.

Done-when: current — the cleanup regression and repeated live-run criteria cover the revised teardown, Linux uncertainty handling, and preservation of shortcut assertions.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      command_procfs.go:50-62 retains PID-only rows for stat-read and parse failures. table_procfs_test.go:24 tests both failures through Table and SnapshotNamed; restoring omission makes both cases fail. live_zellij_close_test.go:210 verifies capture and shutdown polling retain owned files on unknown rows.
```

1. **Strengths**

   - Teardown tracks exact process generations and distinguishes absence, replacement, zombies, and unknown identity (`live_zellij.go:184-219`).
   - Missing-cache exit 2 is narrowly matched; other shutdown errors retain their diagnostic context (`live_zellij.go:157-165`).
   - Layout readiness retries empty observations while returning the first nonempty layout unchanged, preserving the original assertions (`shortcut_conformance_live_test.go:158-200`).

2. **Critical findings**

   None.

3. **Important findings**

   None. Atlas documents the changed Linux observation contract. No new user-facing command, flag, or configuration requires a README update.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Full `procutil` and `pairlifecycletest` packages pass with `-race`.
   - Snapshot regressions and live shortcut conformance pass with `-race -count=10` in private filesystem and socket namespaces.
   - Temporary overlays restoring old Linux omission, teardown, and empty-snapshot behavior each produce the expected regression failure.
   - Linux amd64 test binary cross-compiles. Linux reader behavior was exercised through its portable filesystem adapter on Darwin.
   - Recorded full-suite and native-target log hashes match the tracker; the qualified source and pinned head have no code differences. Checkout remains clean.

6. **Architectural notes**

   - **ARCH-DRY: pass** — shared fixture and native process reader own the correction.
   - **ARCH-PURE: pass** — process IO remains behind the injected reader; filesystem tests exercise the integration adapter.
   - **ARCH-PURPOSE: pass** — both capture and polling preserve the uncertainty previously lost.
   - **ARCH-MOCK: pass** — controllable process observations model delayed writer completion through the production seam; native conformance exercises real Zellij.
   - **ARCH-CONSTRAINTS: pass** — cleanup uses a five-second context and bounded polling.
   - **ARCH-SECURE: pass** — malformed metadata becomes unavailable evidence; live verification uses private state.
   - **ARCH-ORDER: pass** — shutdown precedes terminal-generation observation and file removal; channel-controlled tests reproduce delayed completion.
   - **ARCH-FUNERAL: pass** — owned files are removed after terminal evidence and retained when evidence is unavailable; no new durable artifact family is introduced.

7. **Plan revision recommendations**

   None. R1 and R3 describe the implemented scope accurately. No Core concepts table requires reconciliation.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7e592cde-7edf-4317-b228-14dc9b4cc80f","repository":"github.com/xianxu/pair","issue_id":"000627","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/ea0a9c6e02b06df4","reviewed_head":"f00c302c6c0937bc6101e83b79208e9a3cd87fa4","slot":"pair:1","requested_at":"2026-10-17T04:12:31.568543Z"},"phase":"finished","pid":8543,"started_at":"2026-10-17T04:12:31.720959Z","finished_at":"2026-10-17T04:14:47.404483Z","session_id":"01a1480f-b783-72f3-835e-577c8f5a21e5","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.958479667s
head: "f00c302c6c0937bc6101e83b79208e9a3cd87fa4"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
