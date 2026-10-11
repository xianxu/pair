# Boundary Review — pair#442 (whole-issue close)

| field | value |
|-------|-------|
| issue | 442 — Slot creation: size each step's timeout to the step, and keep the failure |
| repo | pair |
| issue file | workshop/issues/000442-slot-creation-size-each-step-s-timeout-to-the-step-and-keep-the-failure.md |
| boundary | whole-issue close |
| milestone | — |
| window | 98cc5f83cf8a7049adb56a9bdf103b7a8541c860..c673348aaa95f3212a3cd6b016bb305238bedfc1 |
| command | sdlc close --issue 442 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T10:23:16-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking whether the failure record's stderr can exceed the 16 KiB store limit, and how slot directories get removed.

Running the new tests and checking test coverage of the `--show` rendering.

The retry half of `TestFailedCreationIsRecordedAndRetryIsClean` failed. Rerunning it alone to see if it's flaky.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** The fix matches the root cause the issue confirmed. `git worktree add` and `git worktree repair` now run on their own 2-minute budget instead of the 5 s probe budget. Every provisioning command failure now names its command, its budget, and whether that budget ran out. The failure advice names the step that failed. A blocking outcome is saved per slot and shown by `couch --show`. The plan's Revisions honestly explain what was left out (identity probe, branch creation) and why.

Two problems stop a clean SHIP:

1. **The new retry test is flaky.** I reproduced it twice: once in a filtered run of the package, and once in 5 repeated runs of the three new tests. The test's second run (the retry) does a real `git worktree add`, but the 200 ms budget used to force the first failure is still in place. That is the same mistake this issue fixes, now inside the test.
2. **The failure record can be silently dropped.** The saved cause includes up to 64 KiB of a command's error output, but the store refuses any record over 16 KiB. When that happens the write fails and the error is thrown away, so failures with long output leave no record — the very gap this issue was meant to close.

Both fixes are small.

### 1. Strengths
- **One git runner.** `gitWithin(timeout)` is the single runner, and `git` / `gitWrite` are thin wrappers over it (`provision.go:67-78`). Budgets are chosen by step class in one place, so there are no parallel runners.
- **Timeout attribution is careful.** `TimedOut` is set only when the command's own budget expires, not when the caller's deadline does (`ctx.Err() == nil` in `provision.go:62`). `TestCommandErrorTimedOutIsTheCommandsOwnBudget` checks this, and `Unwrap` keeps existing `errors.Is` / `errors.As` checks working.
- **Correct record location.** The record lives in the Git common directory, beside the creation lease (`slotlayout.go:104-113`). It survives a failure that never created the slot directory, is limited to one per slot, and is removed when the slot converges. It is a plain function, not a `SlotLayout` method, as `TestSlotLayoutMethodsBelongToOneResource` requires.
- **The step reaches every caller.** It is set in `ClassifyConvergeError` and rendered by `ReconcileAdvice`, so every UI path that shows advice now shows the step.
- **The slow fake behaves like the real thing.** `slowWorktreeAdd` honours `c.Timeout` the way `OSProvisionIO` does, and it is a wrapper around the real `ProvisionFixture`, not a stub. The claim that putting `worktree add` back on the probe budget turns the test red is believable.

### 2. Critical findings
None.

### 3. Important findings
- **Flaky retry test** — `cmd/internal/couchcore/provision_budget_test.go:93-97`
  - `p.WriteTimeout = 200 * time.Millisecond` is still set when `slow.delay = 0` sends the retry to the fixture's real `git worktree add`.
  - Observed failure: `retry: … git worktree add …: timed out after 200ms: signal: terminated … Preparing worktree`.
  - Fix: set `p.WriteTimeout = time.Minute` before the second `Ensure`.
- **Record dropped when the cause is over 16 KiB** — `cmd/internal/couchcore/slotmemo.go:99-101` and `:118`
  - `newSlotFailure` stores the full `e.Err.Error()`, which can carry up to 64 KiB of error output (`provision_io.go:89`).
  - `ProvisionStore.Write` rejects anything over `provisionRecordLimit` (16 KiB, `provision_store.go:15,80`), and `recordSlotFailure` discards that error.
  - A `weave compile` failure, whose output is streamed into the diagnostics buffer, can easily go over the limit, and then no record is kept.
  - Fix: cut `Cause` down to its last ~8 KiB (keep the end, where the error usually is) before writing. Add a test with a large error output.
  - (Spec "a failure is kept"; ARCH-SECURE: failures should show up, not vanish silently.)

### 4. Minor findings
- `provision.go:130-131`: `clearSlotFailure` runs before `verifyHost`. If `verifyHost` (or reading the setup marker) then fails, the old record is already gone and nothing new is saved. Moving the clear to just before the successful return would fix it.
- `ReconcileAdvice` adds the step only on the retryable and hand-off paths. The `FailureUnknown` and `FailureHold` paths still print just the resource.
- `slotreport.go:153`: if `ReadSlotFailure` hits an unreadable or corrupt record, the error is dropped and `--show` prints nothing about it. A one-line "last failure: unreadable record" would make that visible (ARCH-SECURE).
- `couchcmd/run.go:1122-1128`: no test covers the new `last failure … at <step>` line. `slotreport_render_test.go` already exists and is the natural place for one.
- ARCH-FUNERAL: `last-failure.json` stays behind for a slot number that is never retried or reused. It is bounded to one file per slot number, the same as the existing `creation.json`. Note only.

### 5. Test coverage notes
- All three Done-when tests exist and check real behaviour:
  - a slow add completes;
  - the record carries step, command, timeout and error output;
  - a retry succeeds without manual cleanup and removes the record.
- The retry test fails under load, as described above.
- Nothing covers a large-output record or the `--show` rendering.
- I did not re-check the live creation of `parley.nvim:3`. It rests on the Log.

### 6. Architectural notes
- **ARCH-DRY: pass.** A single runner, with the budget passed in.
- **ARCH-PURE: pass.** `newSlotFailure` is a pure function, and reading and writing the record goes through the existing store and lease.
- **ARCH-PURPOSE: pass**, apart from the record-size gap above. Write-class steps were listed and swept: add and repair now use the write budget, the fetch already had its own budget, and `update-ref` / `config` are single-ref writes.
- **ARCH-MOCK: pass.** Tests run against the stateful `ProvisionFixture` behind the `ProvisionIO` seam.
- **ARCH-CONSTRAINTS: pass.** Budgets are explicit by class, and 2 min matches `FetchTimeout`. The 16 KiB vs 64 KiB mismatch is listed above.
- **ARCH-SECURE: minor.** A corrupt record is ignored silently.
- **ARCH-ORDER: pass.** Writing and clearing the record happen under the creation lease. The ordering problem with `verifyHost` is minor.
- **ARCH-FUNERAL: pass.** One record per slot, removed on success.

### 7. Plan revision recommendations
None required. Optionally, add to the plan's Design §4 that the saved cause is cut to fit the store's 16 KiB limit, once that fix lands.

```findings
findings:
  - id: new
    severity: Important
    family: test-budget-below-real-work
    title: |
      TestFailedCreationIsRecordedAndRetryIsClean retry runs real git worktree add under the 200ms WriteTimeout and flakes
    detail: |
      provision_budget_test.go:93-97 keeps p.WriteTimeout=200ms when slow.delay=0 delegates to the fixture's real checkout; reproduced twice (once in a filtered package run, once in -count=5 of the three new tests) with "timed out after 200ms ... Preparing worktree". Restore p.WriteTimeout (e.g. time.Minute) before the retry Ensure.
  - id: new
    severity: Important
    family: best-effort-write-silently-dropped
    title: |
      Failure record silently not written when the cause exceeds the 16 KiB provision record limit
    detail: |
      newSlotFailure stores the full e.Err.Error(), which carries up to a 64 KiB diagnostic tail (provision_io.go:89); ProvisionStore.Write rejects records over 16 KiB (provision_store.go:15,80) and recordSlotFailure discards the error. Long-output failures such as weave compile leave no record. Keep only the last ~8 KiB of Cause before writing, and test it.
  - id: new
    severity: Minor
    family: record-cleared-before-success
    title: |
      clearSlotFailure runs before verifyHost; a later verifyHost or marker failure erases the record and saves nothing
  - id: new
    severity: Minor
    family: step-in-advice-coverage
    title: |
      ReconcileAdvice FailureUnknown and FailureHold branches omit the step name
  - id: new
    severity: Minor
    family: untrusted-read-degrades-visibly
    title: |
      slotReport ignores ReadSlotFailure errors, so a corrupt last-failure.json shows nothing in couch --show
  - id: new
    severity: Minor
    family: render-surface-untested
    title: |
      couch --show "last failure" line has no render test in slotreport_render_test.go
```

---

## Re-review — 2026-10-11T10:33:06-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 442 — Slot creation: size each step's timeout to the step, and keep the failure |
| repo | pair |
| issue file | workshop/issues/000442-slot-creation-size-each-step-s-timeout-to-the-step-and-keep-the-failure.md |
| boundary | whole-issue close |
| milestone | — |
| window | 98cc5f83cf8a7049adb56a9bdf103b7a8541c860..7e9636615f8d47c548f73d1a4279877062aa04a5 |
| command | sdlc close --issue 442 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T10:33:06-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Read it in chunks.

Checking the claimed fixes now: running the new tests and probing whether the record size cap holds once the cause is JSON-encoded.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The main change works. `git worktree add` and `git worktree repair` now run on a separate 2-minute write budget instead of the 5-second probe budget. Every command failure is wrapped in `ProvisionCommandError`, which names the command, its budget, and whether that budget actually ran out. The advice text names the failed step, and `couch --show` prints the kept failure. I ran the four new couchcore tests plus the advice and layout tests five times each, and they all passed. `TestFailedCreationIsRecordedAndRetryIsClean` no longer flakes. Two things stop this from being SHIP:

- **BR-2 is only half fixed.** `boundCause` limits the cause's raw byte length. The store's 16 KiB limit applies to the size after `json.Marshal`, which turns `\x1b` into `\u001b` and `<` into `\u003c`. I wrote a throwaway test (deleted afterwards) that records a 64 KiB stderr made of ANSI color codes, and another made of `<`. In both cases `recordSlotFailure` silently wrote nothing. Coloured compiler output is a likely real-world cause, so the failure the issue set out to keep can still be lost.
- **The atlas was not updated** for the new write budget or for `last-failure.json`.

1. **Strengths**
   - `gitWithin` is the one git runner, and `git` and `gitWrite` are thin wrappers that only pick the budget (`provision.go:114-126`). Budgeting by class is a single decision in one place.
   - `TimedOut` is true only when the command's own budget expired, not when the caller's deadline did (`provision.go:110`). `TestCommandErrorTimedOutIsTheCommandsOwnBudget` pins both cases.
   - The failure record is now cleared only after `verifyHost` and the setup-marker read succeed (`provision.go:139`). Nothing after that point can fail. BR-3 is fixed.
   - The retry test drives a real fixture repository and shows that the leftover environment directory gets reused without any manual cleanup. That is the issue's "retry is clean" requirement.
   - The record lives under the Git common directory, is written under the creation lease, and holds one entry per slot. All callers reach `Reconcile` through `Ensure`, so the record is always cleared on success.

2. **Critical:** none.

3. **Important**
   - **BR-2, not addressed** (`slotmemo.go:449-467`): the limit is on raw bytes, but the store checks encoded bytes. Under the ARCH-PURPOSE rule for this family (state the rule, fix the class), the rule is: *a best-effort record must be bounded by the measure its store enforces, and when a write is refused it falls back to something smaller rather than vanishing.* Fix sketch:
     - Marshal the record, and while it is over `provisionRecordLimit`, halve the cause budget and try again.
     - As a last resort, write the record with an empty cause plus `cause_truncated: true`.
     - Add test cases with ANSI-heavy and `<`-heavy stderr to `TestSlotFailureRecordFitsTheStore`.
   - **Atlas update appears missing** (`atlas/workspace-provisioning.md:138-140`): it still says "Local probes have 5-second ceilings, fetch 120 seconds, compile 20 minutes". It needs the 2-minute write budget for `worktree add`/`repair`, the `couch-workspaces/N/last-failure.json` record (who writes it, when it is cleared, its size bound), and the `couch --show` "last failure" line.

4. **Minor**
   - **BR-4:** the code fix is correct and every advice branch now includes the step. Only the retryable path is pinned, indirectly, by `provision_budget_test.go:226`. `TestReconcileAdviceText` asserts the step in no branch.
   - **ARCH-ORDER:** `recordSlotFailure` takes the lease only after `Reconcile` returns. A concurrent `Ensure` that succeeds and clears the record could have its clear overwritten by this stale failure. The window is narrow.
   - **ARCH-FUNERAL:** a slot number that fails and is never created again keeps `last-failure.json` indefinitely. It is bounded at one per number, so this is acceptable, but it should be written down in the atlas entry.

5. **Test coverage**
   - The four new couchcore tests passed at `-count=5`.
   - I could not run the couchcmd package in this checkout: `runtimebundle/embed.go` fails because `assets/runtime/files` is missing, which is an environment problem. I reviewed `TestRenderSlotReportLastFailure` by reading it, and it covers both the readable and the unreadable record.
   - Nothing covers JSON escaping making the record too big. That gap is how BR-2 survived.

6. **Architecture**
   - **ARCH-DRY:** pass (one git runner).
   - **ARCH-PURE:** pass. `boundCause` and `newSlotFailure` are pure, and the IO around them is thin.
   - **ARCH-PURPOSE:** flagged through BR-2. The record can still be dropped.
   - **ARCH-MOCK:** pass. The slow fake wraps the real-git fixture behind the `ProvisionIO` seam.
   - **ARCH-CONSTRAINTS:** pass. The write budget is explicit, and the tests prove `worktree add` receives it.
   - **ARCH-SECURE:** pass. The record is read through the strict-JSON store, and an unreadable one is shown as unreadable rather than hidden.
   - **ARCH-ORDER:** minor note above.
   - **ARCH-FUNERAL:** minor note above.

7. **Plan revisions**
   - Plan design item 4 says the record holds the "full cause with stderr tail". Add a Revisions entry: the cause is limited so the *encoded* record fits the store's 16 KiB, and if it still doesn't fit, a record without the cause is written instead.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      provision_budget_test.go:243 restores WriteTimeout=time.Minute before the retry; 5 runs green at -count=5.
  - id: BR-2
    disposition: not-addressed
    note: |
      boundCause limits raw bytes but the 16 KiB limit applies after json.Marshal; 64 KiB stderr of "\x1b[0m" or "<" still records nothing (probe-tested). Rule: limit by encoded size, fall back to a smaller record, never drop.
  - id: BR-3
    disposition: addressed
    note: |
      clearSlotFailure now runs after verifyHost and the marker read (provision.go:139); nothing after it can fail.
  - id: BR-4
    disposition: not-addressed
    note: |
      Code fixed in every branch, but no test asserts the step in Unknown/Hold advice; TestReconcileAdviceText unchanged.
  - id: BR-5
    disposition: addressed
    note: |
      slotreport.go:530-534 sets LastFailureError; run.go renders "last failure: unreadable"; covered by TestRenderSlotReportLastFailure.
  - id: BR-6
    disposition: addressed
    note: |
      TestRenderSlotReportLastFailure (slotreport_render_test.go:127) pins both lines (read; not run here because embed assets are missing).
findings:
  - id: new
    severity: Important
    family: atlas-missing-new-surface
    title: |
      atlas/workspace-provisioning.md not updated for the write budget or last-failure.json
    detail: |
      Line 138 still lists probe 5 s / fetch 120 s / compile 20 min with no 2-min write budget; the couch-workspaces/N/last-failure.json record (writer, clearing, size bound) and the couch --show "last failure" line are undocumented.
```

---

## Re-review — 2026-10-11T10:43:08-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 442 — Slot creation: size each step's timeout to the step, and keep the failure |
| repo | pair |
| issue file | workshop/issues/000442-slot-creation-size-each-step-s-timeout-to-the-step-and-keep-the-failure.md |
| boundary | whole-issue close |
| milestone | — |
| window | 98cc5f83cf8a7049adb56a9bdf103b7a8541c860..06f754f4e34ff179fbc369e19ea9cafb42327738 |
| command | sdlc close --issue 442 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T10:43:08-07:00 |
| verdict | SHIP |

## Review

I've checked the diff against the plan. Next I'm running the tests and reverting the BR-2 fix to confirm its regression test fails without it.

```verdict
verdict: SHIP
confidence: high
```

All three open findings are fixed, and I checked each against the code, not the commit messages. BR-2 is fixed: `fitSlotFailure` shrinks the cause until the record fits under 16 KiB. When I removed that fix in the review checkout, `TestSlotFailureRecordFitsTheStore/escaped` failed with `provision record too large`; restoring it made it pass again. BR-4 is fixed: `ReconcileAdvice` now names the step for every failure class and cause, and a test loops over `AllFailureClasses()` to pin it. BR-7 is fixed: `atlas/workspace-provisioning.md:138-150` now documents the 2-minute write budget, the `last-failure.json` writer, how it's cleared, its size bound, and the `--show` line. The design matches the plan. `go test ./cmd/internal/couchcore/` passes (209 s). `couchcmd` would not build in this checkout because its embedded runtime assets are missing (`runtimebundle/embed.go:8` matches no files). That's a problem with this checkout, not with the diff, but it means I could not run the render test `TestRenderSlotReportLastFailure` here. Nothing blocks shipping; the two remaining findings are Minor.

1. **Strengths**
   - `provision.go:57-64`: every command failure is wrapped in one `ProvisionCommandError` with `Unwrap`, so `errors.Is`/`errors.As` and the retryable classification still work. `TimedOut` is only set when the command's own budget expired, not the caller's deadline, and a test checks both cases.
   - `gitWithin` is the only git runner, with `git` for probes and `gitWrite` for writes. This keeps budgets in one place (ARCH-DRY).
   - The record is cleared only after `verifyHost` and the marker read succeed (`provision.go:141`), and a record that can't be written is named in the caller's error (`slotfailure.go:183`).
   - `fitSlotFailure` checks the size of the encoded JSON rather than the raw string length, so escaping can't push the record past the limit. The test covers both plain and escape-heavy stderr.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `slotmemo.go:131-142`: `clearSlotFailure` ignores errors when it can't get the creation lease or remove the record. If that happens, `couch --show` keeps showing an old "last failure" for a slot that has since converged. This is the 2nd finding in family `best-effort-write-silently-dropped`. The rule that covers both: any write or remove of the slot-failure record that fails is reported to the caller. For a remove after a successful run, that means a warning added to `ProvisionResult.Warning`.
   - `slotmemo.go:101-108`: `boundCause` panics on a slice index out of range when `limit < head + len(marker)` (the loop reaches limits of 4 and below). That would only happen if the rest of the record (for example, `Command`) were over about 16 KiB, so it isn't realistic today. Guard with something like `if limit <= len(marker) { return "" }`.

5. **Test coverage notes:** I confirmed the BR-2 regression test fails without the fix. The step-in-advice test pins every class and cause combination. The budget tests use an injected fake that respects the command's own timeout the way `OSProvisionIO` does. The retry part of the creation test now restores the write budget before running real git, which disposes the flake BR-1 raised.

6. **Architecture**
   - **ARCH-DRY:** pass. There is one git runner, and the record helpers live in a single file.
   - **ARCH-PURE:** pass. `newSlotFailure`, `fitSlotFailure` and `boundCause` are pure; writing the record stays a thin layer behind the lease.
   - **ARCH-PURPOSE:** pass. Both write steps (`worktree add` and `worktree repair`) get the write budget, and the record reaches `--show`.
   - **ARCH-MOCK:** pass. The tests use the existing `ProvisionFixture` behind the `ProvisionIO` interface.
   - **ARCH-CONSTRAINTS:** pass. Each step class has an explicit budget, and the atlas documents them.
   - **ARCH-SECURE:** pass. The record is read through `ProvisionStore`, which enforces the size limit and path checks, and a corrupt record is shown as unreadable instead of being hidden.
   - **ARCH-ORDER:** pass. Each slot has one record, written and removed under the creation lock, and it is only cleared after verification.
   - **ARCH-FUNERAL:** pass, apart from the minor item above. There is at most one record per slot, the next failure replaces it, and the next verified success removes it.

7. **Plan revisions:** add a Revisions entry noting that writing the record is no longer best-effort and silent: a failed write is surfaced through `RecordErr`. Also list the two tests the plan doesn't mention, `TestSlotFailureRecordFitsTheStore` and `TestUnrecordedFailureIsSaid`.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      fitSlotFailure bounds the encoded record to 16 KiB; removing it makes TestSlotFailureRecordFitsTheStore/escaped fail with "provision record too large".
  - id: BR-4
    disposition: addressed
    note: |
      ReconcileAdvice names the step for every class and cause (slotfailure.go:106-128), pinned by the AllFailureClasses loop in TestReconcileAdviceText.
  - id: BR-7
    disposition: addressed
    note: |
      atlas/workspace-provisioning.md:138-150 documents the 2-minute write budget and the last-failure.json writer, clearing, size bound and --show line.
findings:
  - id: new
    severity: Minor
    family: best-effort-write-silently-dropped
    title: |
      clearSlotFailure ignores lease and remove errors, so --show can keep showing an old failure after the slot converged
    detail: |
      This is the 2nd finding in family best-effort-write-silently-dropped. Rule: any write or remove of the slot-failure record that fails is reported to the caller (RecordErr when the run blocks, a ProvisionResult.Warning after a successful run). Prevalence: two sites, recordSlotFailure (fixed) and clearSlotFailure (slotmemo.go:131-142).
  - id: new
    severity: Minor
    family: bound-helper-total-over-input
    title: |
      boundCause panics with a slice index out of range when limit is smaller than head plus the elision marker
    detail: |
      fitSlotFailure halves limit down to 1; once limit is about 6 or less, len(cause)-(limit-head-len(marker)) is past the end of the string. Only reachable if the rest of the record is over about 16 KiB. Guard with: if limit <= len(marker), return "".
```
