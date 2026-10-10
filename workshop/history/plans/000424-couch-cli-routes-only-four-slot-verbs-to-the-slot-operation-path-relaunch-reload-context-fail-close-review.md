# Boundary Review — pair#424 (whole-issue close)

| field | value |
|-------|-------|
| issue | 424 — couch CLI routes only four slot verbs to the slot-operation path; --relaunch/--reload-context fail |
| repo | pair |
| issue file | workshop/issues/000424-couch-cli-routes-only-four-slot-verbs-to-the-slot-operation-path-relaunch-reload-context-fail.md |
| boundary | whole-issue close |
| milestone | — |
| window | 39f17871f56fc3f0404c3c71e0165ac31abf63e4..54dbb545bd9501e9eda792a4e441749f8957e0a3 |
| command | sdlc close --issue 424 |
| reviewer | claude |
| timestamp | 2026-10-10T09:58:09-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The fix is small and does what the issue asks. `runMessageCLIWithCall` (`cmd/internal/couchcmd/messages.go:25`) now asks `couchcore.IsSlotOperation` which path a verb takes, instead of checking its own four-verb `==` chain. `couchcore/slot_operation.go:34` is the single list. It includes `OpRelaunch` and `OpReloadContext`, so both verbs now reach `runSlotOperationCLI`. The top-level dispatcher in `run.go:279-287` sends every `cliMessage` invocation through this router. Both "Done when" clauses hold. The new router-level test runs real argv through `ParseCLI` and then the router, once for every declared slot verb, and it passes. Before the fix, relaunch and reload-context took the message path, which sets no request ID, so the test's `requests[0].ID == ""` check would have caught them.

1. **Strengths**
   - `messages.go:25`: the router now reads the single source, the `slotOperations` list in `couchcore/slot_operation.go:34`, so the next slot verb added there is routed automatically (ARCH-DRY pass).
   - `messages_test.go:443-474`: the test builds its cases from `couchcore.Operations()` filtered by `IsSlotOperation`, so it covers new verbs without edits. It goes through `ParseCLI` and sets `--confirm` from `OperationConfirms`, which exercises the real parse-then-route seam rather than calling the handler directly. The `n < 6` floor stops the loop from passing vacuously on an empty list.
   - The sweep claim holds up. `run.go:549` (current-repo scope) and `run.go:559` (live ownership) are separate policies. Relaunch and reload-context are broker-side slot operations, so leaving them out of those lists is consistent (ARCH-PURPOSE pass).
   - The lesson entry names the actual miss: the earlier sweep grepped only for `case` lists and skipped `||` chains.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - The test only runs the default flag set. It doesn't check that `--force-unknown` and `--same-binary` reach the broker request for relaunch and reload-context. That forwarding is outside this window and likely covered by #421's handler tests.
   - `run.go:549` and `run.go:559` are still hand-kept verb lists. They are different concepts, which is fine, but they partly overlap with `slotOperations`. A short comment on each saying why the relaunch verbs are deliberately left out would stop the next sweep from re-checking them.

5. **Test coverage:** This is the right level for this kind of bug: router plus parser, with a fake broker call that records requests. The test needs no IO, and the router remains a thin shell (ARCH-PURE pass).

6. **Architectural notes:** none beyond the comment suggestion above.

7. **Plan revisions:** none. The plan matches the code.

```findings
findings:
  - id: new
    severity: Minor
    family: router-test-flag-coverage
    title: |
      Router test exercises only default flags; --force-unknown/--same-binary propagation through the router is unpinned
    detail: |
      TestEverySlotOperationIsRoutedToTheSlotPath builds argv with only --confirm; relaunch/reload-context optional flags are not asserted on the broker request.
```
