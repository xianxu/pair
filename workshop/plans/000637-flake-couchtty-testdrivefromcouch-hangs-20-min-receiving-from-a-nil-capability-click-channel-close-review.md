# Boundary Review — pair#637 (whole-issue close)

| field | value |
|-------|-------|
| issue | 637 — flake: couchtty TestDriveFromCouch hangs 20 min receiving from a nil capability-click channel |
| repo | pair |
| issue file | workshop/issues/000637-flake-couchtty-testdrivefromcouch-hangs-20-min-receiving-from-a-nil-capability-click-channel.md |
| boundary | whole-issue close |
| milestone | — |
| window | ce783cd4a474d939c174bb079bc687efc2cc457a..1eef1103cac8baeb4fd8730f8df482ee7d611349 |
| command | sdlc close --issue 637 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T04:38:49-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The change removes the unbounded nil-channel receive and detects both documented premature-readiness windows. However, its replacement readiness signal is transient: a delayed control-reset notice can overwrite the completion notice before the test samples it. A controlled scheduling probe reproduced a five-second failure after disable completed.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: durable-completion-evidence
    title: |
      Disable readiness can disappear after successful completion
    detail: |
      cmd/internal/couchtty/console_drive_test.go:221 requires the current notice to equal "drive capability off.". Control delivery runs independently through console_webcast.go:166-178; if its reset arrives after capability completion, applyDrive publishes "Control of the cast tab is back with you." (console_drive.go:87-88,137-138), replacing that status notice. The vulnerable consumers in this window are the waits at console_drive_test.go:282 and :419; the authority-only subtest has no driver reset and does not exercise this ordering. A temporary Go overlay delayed reset delivery until completion presentation and delayed readiness sampling until reset presentation finished: TestDriveFromCouch failed after five seconds with control cleared and the replacement notice displayed. Use durable, operation-specific completion evidence and add a gated regression for completion-before-reset-presentation (ARCH-PURPOSE).
```

1. **Strengths**
   - `awaitDrive` immediately rejects nil and retains its five-second deadline.
   - Both gated regressions fail when the old readiness predicate is restored.
   - Gates release on failure paths; production behavior remains unchanged.

2. **Critical findings**
   - The transient completion predicate above leaves a scheduling-dependent failure. Capture completion independently of the current notice row.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   - Original integration test and both orderings passed 20 repetitions under `-race`, overlapping another webcast test suite.
   - Related webcast race tests passed.
   - Old-predicate mutation failed both regression subtests.
   - The additional scheduling probe reproduced the remaining defect. Repository files remained unchanged.

6. **Architecture**
   - **ARCH-DRY: Pass.** Shared readiness and bounded-await helpers.
   - **ARCH-PURE: Pass.** Integration scheduling stays in injected test fixtures.
   - **ARCH-PURPOSE: Flag.** The load-sensitive synchronization fix still depends on observing a transient state.
   - No new public or architectural surface requires README or atlas changes.

7. **Plan revision recommendation**
   - Add a `## Revisions` entry requiring durable completion evidence and a third gated ordering: completion notice published, reset notice delivered, then readiness sampled.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"183e5cf9-bed2-4d05-a92e-b5f2bc17af6f","repository":"github.com/xianxu/pair","issue_id":"000637","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/27015f11cc815a1d","reviewed_head":"1eef1103cac8baeb4fd8730f8df482ee7d611349","slot":"pair:1","requested_at":"2026-10-17T11:35:38.717009Z"},"phase":"finished","pid":52321,"started_at":"2026-10-17T11:35:38.890283Z","finished_at":"2026-10-17T11:38:48.290414Z","session_id":"01a149a5-6818-79b2-9778-5432765a27fe","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.767227125s
head: "1eef1103cac8baeb4fd8730f8df482ee7d611349"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
