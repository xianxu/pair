# Boundary Review — pair#877 (whole-issue close)

| field | value |
|-------|-------|
| issue | 877 — Repository preference publication escapes typed store contention |
| repo | pair |
| issue file | workshop/issues/000877-repo-preference-store-contention.md |
| boundary | whole-issue close |
| milestone | — |
| window | 503ba31b9fadffa4e965105f805391eaccdc0af8..4e87d68f235a09541852d55d81cc89911369ddc6 |
| command | sdlc close --issue 877 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-24T13:34:39-07:00 |
| verdict | SHIP |

## Review

Diff and production caller read; now checking the inner lock-wait path and running the tests.

Failures are the sandbox's git-init fence, unrelated to the diff. Now reverting the predicate to confirm the regression tests go red.

```verdict
verdict: SHIP
confidence: high
```

The fix does what pair#877's Spec asks and stays small. The publication loop in `rememberRepoConfigurationContext` used to retry only the raw `EWOULDBLOCK`/`EAGAIN`. It now also retries the typed `ErrThreadStoreBusy` that the #827 timed wait returns. That check lives in one shared predicate, `isStoreContention`, which `retentionLockError` uses too. The loop is bounded by the caller's deadline and refuses a caller that has no deadline. When the deadline passes after contention, the error carries both the context error and the last busy error. `storeReadLockWait` is unchanged, no timeouts were raised, and no assertions were removed.

I checked that the regression tests actually depend on the fix. In the disposable checkout I put the loop back to the old behaviour: typed busy stops the loop instead of retrying. With that change, `TestRepoPreferencePublicationRetriesTypedContention` and `TestRepoPreferencePublicationRefusesBusyAtTheDeadline` failed in 3 out of 3 runs. With the fix restored, all four new tests passed 5 out of 5, and the concurrent-instance test passed `-count=10`. Some other tests in the package fail here, but all of them stop at `git init refused … outside the reviewer's directories (sdlc #339)`. That is this sandbox blocking `git init`, not the diff. Nothing blocks SHIP. The only open items are wording in the plan that no longer matches what was built.

1. **Strengths**
   - `retention.go:504-508`: one `isStoreContention` predicate shared by `retentionLockError` and the publication loop (ARCH-DRY). I looked for other loops of the same kind: the only other caller of `waitForStoreLockContext` is `orphan_registry.go:271`. Its loop, and the one in `launchfailure_store.go:101`, call raw `flock` directly through `acquireLaunchFailureLock`, so they never see the typed error and checking only raw errno is correct there. `repositoryalias_store.go:56` already handles `ErrThreadStoreBusy`. No call site was missed.
   - `repopreferences_store.go:89-91`: refusing a caller with no deadline means the retry always has a bound, which was the TL's condition. The one production caller, `repopreferences_adoption.go:57`, always passes a 5 s `repoIdentityTimeout`, so this refusal never fires in production.
   - `repopreferences_store.go:101-103`: the error is joined only when contention happened and the context has ended, and the deadline test checks for both errors. That covers the case the Log mentions, where the deadline fell inside the inner wait and the joined busy error used to be lost.
   - The tests use the real `store.lock` held through `holdStoreLock` and a shortened wait, with no mocks of the loop. Each refusal test also checks that `repo-preferences.json` is byte-for-byte unchanged.
   - The `editedrestart_publication_test.go` call that used `Background` was moved to the bounded wrapper, so no test depends on the old unbounded path.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - The plan's RED bullet no longer matches the code in two places. It says the retryable case runs with a "Background ctx", but under the TL condition `Background` is refused, and the test uses a 5 s deadline. It also says the tests are in `repopreferences_store_test.go`, but they are in the new `repopreferences_contention_test.go`. Those are the only two drifted claims in the plan. The atlas note at `atlas/couch.md:3616-3619` matches the code.

5. **Test coverage notes**
   - The Done-when clauses are each covered:
     - retryable contention: `RetriesTypedContention`;
     - caller deadline with a bounded busy refusal: `RefusesBusyAtTheDeadline`;
     - cancellation: `StopsOnCancellation`;
     - caller with no deadline: `RefusesAnUnboundedCaller`;
     - concurrent preservation: the existing 16-writer test.
   - The cancellation test passed even before the fix, as the Log says. That is expected: it checks behaviour that already worked and is guarded against breaking.

6. **Architectural notes**
   - **ARCH-DRY: pass.** One shared predicate, and every sibling loop checked as described above.
   - **ARCH-PURE: pass.** The change sits inside the existing thin lock-and-retry wrapper. No business logic moved into IO.
   - **ARCH-PURPOSE: pass.** The cause found in history (#827 hid typed busy from the #728 loop) is fixed in the loop itself, without raising timeouts. The #854 interaction is recorded in the plan for whichever change lands second.
   - **ARCH-EVOLVE: pass, nothing to check.** No persisted format changed.
   - **Future work:** coordinator busy (`storagegc.ErrCoordinatorBusy` from `coordinator.WithLock`) is still not retried by this loop. That was true before this change as well and is outside #877's scope.

7. **Plan revision recommendations**
   - Add a `## Revisions` entry: "The RED retryable-contention case runs under a bounded (5 s) context, not Background, because the TL condition refuses deadline-less callers. The tests live in `repopreferences_contention_test.go`."

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: plan-claims-match-delivered-code
    title: |
      The plan's RED bullet names a Background ctx and repopreferences_store_test.go; the code uses a deadline ctx and a new test file
    detail: |
      There are only two drifted claims. (1) The retryable-contention case is described as "Background ctx", but under the TL condition Background is refused, and the test uses a 5 s WithTimeout. (2) The location is given as repopreferences_store_test.go, but the tests are in repopreferences_contention_test.go. Add a Revisions entry. No code change needed.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"b5eb85e6-a26f-4d30-9d73-5c31d53abb69","repository":"github.com/xianxu/pair","issue_id":"000877","recipe":"small-diff-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/fe108f4693755582","reviewed_head":"4e87d68f235a09541852d55d81cc89911369ddc6","slot":"pair:6","requested_at":"2026-10-24T20:32:01.77051Z"},"phase":"finished","pid":71106,"started_at":"2026-10-24T20:32:01.924104Z","finished_at":"2026-10-24T20:34:37.230672Z","session_id":"d7709bd6-7ea6-4633-9a14-c25c1c5dc345","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 7.817301541s
head: "4e87d68f235a09541852d55d81cc89911369ddc6"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
