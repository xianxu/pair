# Boundary Review — pair#806 (whole-issue close)

| field | value |
|-------|-------|
| issue | 806 — Replacement Zellij disconnects before first wrapper startup |
| repo | pair |
| issue file | workshop/issues/000806-replacement-zellij-startup-disconnect.md |
| boundary | whole-issue close |
| milestone | — |
| window | e814b17f15f5f194119f41b209bc35f76285e1e7..7ad66fa1615447105992dd184773efa8ec505a58 |
| command | sdlc close --issue 806 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T15:51:03-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation matches the revised diagnostic-only scope and preserves production startup behavior. Bounded capture and reader tests pass, and both retained native logs match their recorded hashes. One Important coverage gap remains: tests do not protect the lifecycle wiring that makes failure evidence survive teardown.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: diagnostic-lifecycle-regression-coverage
    title: |
      Exercise server-log retention through Fleet.Close
    detail: |
      cmd/internal/testcouch/fleet.go:395-408 and :454-461 implement capture selection and ordering, but startup_server_log_test.go:28-138 calls capture helpers directly. The repeated-cleanup test never calls Close. Removing Close's pre-teardown capture leaves these tests green; the native test also does not assert diagnostic output. Cover the complete family: failed fleets with the option off/on, successful fleets with it off/on, pre-cancellation capture, post-join capture before root removal, and repeated Close without additional captures. Use controlled lifecycle seams and assert retained markers and counts so removing or moving the captures fails a regression test.
```

1. **Strengths**

   - Reuses the existing root-confined reader, retaining final-symlink, nonregular-file and escape refusal.
   - Separate `sync.Once` budgets include unsuccessful reads; retained bytes are escaped and bounded.
   - The scope revision explicitly preserves uncertainty and defers causal correction.
   - Atlas documentation explains attribution limits and the successful-control option.

2. **Critical findings**

   None.

3. **Important findings**

   The lifecycle regression coverage finding above. Add assertions around actual `Close` execution, including failure selection and capture ordering.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - `go test ./cmd/internal/testcouch -run '^TestStartupEvidence' -count=1` passed.
   - `go test ./cmd/internal/testcouch -count=1` passed.
   - `git diff --check` passed.
   - Independently verified both recorded native-log hashes and their reboot, delivery, two-tail capture and cleanup receipts. No additional native run performed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared reader remains the file-access boundary.
   - **ARCH-PURE — pass:** filesystem capture stays in the integration harness; no new business logic is buried in IO.
   - **ARCH-PURPOSE — pass:** delivers the explicitly revised retention purpose without claiming causation.
   - **ARCH-EVOLVE — pass:** no persisted format changes.
   - Core-concept integration entries match the changed files. Atlas is updated; no new user-facing command, flag or configuration requires a README change.

7. **Plan revision recommendations**

   No scope revision needed. Extend the verification checklist to require the lifecycle regression cases enumerated above.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"138324d2-92d7-4829-b299-916f92a47529","repository":"github.com/xianxu/pair","issue_id":"000806","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/bc81076198d0c789","reviewed_head":"7ad66fa1615447105992dd184773efa8ec505a58","slot":"pair:2","requested_at":"2026-10-22T22:49:39.363546Z"},"phase":"finished","pid":66462,"started_at":"2026-10-22T22:49:39.549883Z","finished_at":"2026-10-22T22:51:01.984855Z","session_id":"01a165ce-477a-7372-8733-6f5ab8d8040f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.366438708s
head: "7ad66fa1615447105992dd184773efa8ec505a58"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T15:57:44-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 806 — Replacement Zellij disconnects before first wrapper startup |
| repo | pair |
| issue file | workshop/issues/000806-replacement-zellij-startup-disconnect.md |
| boundary | whole-issue close |
| milestone | — |
| window | e814b17f15f5f194119f41b209bc35f76285e1e7..867b9bf5303d7d0175d6b4831b73c3b6803783f2 |
| command | sdlc close --issue 806 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T15:57:44-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation matches the revised diagnostic-only scope, and the relevant tests pass. BR-1 is partially corrected: tests now protect selection, cancellation ordering, counts, and repeated Close, but still permit the final snapshot before native producers finish.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      startup_server_log_test.go:191-225 models only the admitted-call join; the console and owned native resources remain absent. Independently moving fleet.go:454-462 immediately after the idle join, ahead of console shutdown, stopAgents and reconcileProcesses, leaves both the lifecycle test and the entire testcouch package green. The remaining family diagnostic-lifecycle-regression-coverage comprises final captures for failed fleets with the option off/on and successful fleets with it on. Enforce the rule that final capture follows every native producer and IO join: hold a later producer, emit its final marker at completion, and require all three capture paths to retain it before root removal. The successful option-off case must continue retaining no snapshots.
```

1. **Strengths**

   - Reuses the confined reader, including symlink, escape, and nonregular-file refusal.
   - Independent one-shot budgets bound both phases, including failed reads.
   - Lifecycle tests cover all four failure/option combinations and repeated Close.
   - Both retained native logs match their recorded hashes and contain two bounded snapshots.

2. **Critical findings**

   None.

3. **Important findings**

   BR-1 remains open as detailed above. Protect the final native-producer boundary with a controlled completion marker and an ordering mutation.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Startup-evidence tests, full `testcouch` package, targeted launcher admission/handoff tests, and diff whitespace checks passed.
   - Removing either capture or moving the first capture after cancellation fails the lifecycle test.
   - Moving final capture before native teardown incorrectly passes the full package.
   - Mutation overlays were temporary; tracked files remain unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared file-access boundary reused.
   - **ARCH-PURE — pass:** diagnostic IO remains in the integration harness.
   - **ARCH-PURPOSE — pass:** delivers the expressly revised retention scope without claiming causation.
   - **ARCH-EVOLVE — pass:** no persisted-format changes.
   - Core-concept locations match the implementation. Atlas covers the new harness option; no new user-facing surface requires a README update.

7. **Plan revision recommendations**

   Extend `## Revisions` → “BR-1 verification refinement” to distinguish admitted-call completion from native-producer/IO completion and require a regression that rejects final capture before the latter.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"28c3c837-99d1-4f58-9dd5-4c1e2f13e317","repository":"github.com/xianxu/pair","issue_id":"000806","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/0956f5876b00d904","reviewed_head":"867b9bf5303d7d0175d6b4831b73c3b6803783f2","slot":"pair:2","requested_at":"2026-10-22T22:55:47.704524Z"},"phase":"finished","pid":95032,"started_at":"2026-10-22T22:55:47.797881Z","finished_at":"2026-10-22T22:57:43.963972Z","session_id":"01a165d3-e55f-7700-ae2f-aae63d5067e1","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.84084375s
head: "867b9bf5303d7d0175d6b4831b73c3b6803783f2"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-22T16:03:04-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 806 — Replacement Zellij disconnects before first wrapper startup |
| repo | pair |
| issue file | workshop/issues/000806-replacement-zellij-startup-disconnect.md |
| boundary | whole-issue close |
| milestone | — |
| window | e814b17f15f5f194119f41b209bc35f76285e1e7..23c008e9aecc51ee5ef6dcebc1fa97607d3d2f9f |
| command | sdlc close --issue 806 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T16:03:04-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation satisfies the revised diagnostic-only scope. BR-1 is addressed: lifecycle tests now protect capture selection, cancellation ordering, native producer/IO completion, root removal, and repeated cleanup. No new findings.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      startup_server_log_test.go:180-275 exercises real Fleet.Close across all four failure/option combinations. Independent mutations removing either capture, delaying the first past cancellation, or moving the final capture before native teardown all fail in the three capture-enabled cases. Successful option-off retains no snapshots; repeated Close adds none.
```

1. **Strengths**

   - Reuses the confined reader and preserves unsafe-source refusal.
   - Independent one-shot budgets include failed reads; retained tails are bounded and escaped.
   - The controlled producer verifies final bytes and IO completion before root removal.
   - Atlas and plan revisions accurately describe diagnostic scope and attribution limits.

2. **Critical findings:** None.

3. **Important findings:** None remaining.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Startup-evidence tests passed.
   - Full `testcouch` package passed.
   - Targeted launcher admission/handoff tests passed.
   - All four lifecycle mutations failed as expected.
   - Both retained native logs match their recorded hashes and show reboot, delivery, two captures, and cleanup within recorded bounds. No native reruns performed.
   - Diff whitespace checks passed; tracked checkout unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared file-access boundary reused.
   - **ARCH-PURE — pass:** diagnostic IO remains in the integration harness.
   - **ARCH-PURPOSE — pass:** fulfills the explicitly revised retention purpose without claiming causal resolution.
   - **ARCH-EVOLVE — pass:** no persisted-format changes.
   - Core-concept locations match the implementation. Atlas covers the new harness surface; no new user-facing command or configuration requires README changes.

7. **Plan revision recommendations:** None. Existing revisions cover the corrected lifecycle verification.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"a386291c-14c1-4cd1-8c3d-5a144a784b92","repository":"github.com/xianxu/pair","issue_id":"000806","recipe":"small-diff-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/f0c66e7124eec7a2","reviewed_head":"23c008e9aecc51ee5ef6dcebc1fa97607d3d2f9f","slot":"pair:2","requested_at":"2026-10-22T23:01:34.609509Z"},"phase":"finished","pid":73837,"started_at":"2026-10-22T23:01:34.71983Z","finished_at":"2026-10-22T23:03:03.792488Z","session_id":"01a165d9-30d6-74b0-92a1-d546a139145f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.412091209s
head: "23c008e9aecc51ee5ef6dcebc1fa97607d3d2f9f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
