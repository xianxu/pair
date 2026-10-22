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
