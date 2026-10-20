# Boundary Review — pair#726 (milestone M1)

| field | value |
|-------|-------|
| issue | 726 — Hub lifecycle ops time out client-side (resume pair:6, relaunch parley.nvim:4: 'context deadline exceeded') with no diagnosis |
| repo | pair |
| issue file | workshop/issues/000726-hub-lifecycle-ops-time-out-client-side-resume-pair-6-relaunch-parley-nvim-4-context-deadline-exceeded-with-no-diagnosis.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | e2ac2a19f253961b715b6edf104ca82d72b5c40a..4dd0a6107ef9867aebde532f2027b10b86f72473 |
| command | sdlc milestone-close --issue 726 --milestone M1 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-20T08:33:44-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Diff read; now checking `slotOps.handle` for synchronous work outside the new budget, then running the tests.

Checking `Close` ordering and the receipt-filter callers, then running tests.

Running the new tests (and package suites) under `-race`.

A data race surfaced in `TestHubSupervisionReassignmentChangesBinding`; checking whether it's caused by this diff.

Spot-checking one claimed mutation: making admission resolve the target fresh should turn the warm test red.

Mutation confirmed red on all three admission tests. Now checking what runs outside the 1.5 s budget (`reference`, `preparePark`).

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M1 does what it set out to do. Hub slot-operation admission now reads warm sources: the caller comes from `connectedWorkspace` and the target from the new `hubTargetMemo`. It runs under its own 1.5 s budget, and every refusal names the phase that caused it. The queued guard still re-proves authority on fresh discovery before any effect. Discover now verifies slot rows concurrently. Its refactor into `verifySlotRow` (`cmd/internal/couchcore/slotcatalog.go:651`) checks the same things in the same order as the old serial loop, and the replacement fence is kept. I made admission resolve the target fresh again, and all three admission tests went red. Nothing blocks the gate.

Two things should be fixed before shipping, both cheap:
- **Lost error on a cold target:** when a cold resolution fails after the budget ends, the error is thrown away, so the caller sees "warming; retry" forever and never the real reason.
- **Hub move has the same bug:** hub-move admission still runs one full Discover per named slot, in the request, with no budget.

I could not run the couchcore Discover tests: the sandbox's `git` wrapper refuses `git init` in temp dirs (it says "outside the reviewer's directories").

1. **Strengths**
   - The safety boundary is in the right place (ARCH-ORDER). Admission is only a cheap gate, and authority is still proven fresh in queue prepare and at the effect gate (`cmd/internal/couchcmd/hub_guards.go:128-150`). The test `TestHubSlotAdmissionStaleMemoRefusesBeforeAnyEffect` checks exactly that: a stale memo gets admitted, then the queued guard refuses it.
   - The memo does not answer `Fresh` calls itself. `message_service.go:618-622` sets `authority.hubTarget = memo.Fresh`, so every fresh resolution, including the queued guard's, refreshes the memo. There are no hand-maintained update points.
   - Refusals are typed: `admissionPhase` (`hub_guards.go:265`) turns any overrun into "admission ran out of its 1.5s budget in the X check; nothing was queued". It never stays silent.
   - Background work has a clear end: each in-flight resolution runs under the service lifetime, Close cancels that lifetime and joins it (`message_service.go:266-273`), and a test proves the join.
   - The admission test setup counts calls at the two places that spawn processes in production, and its mutations really do fail. I re-ran the target-source mutation myself: all three tests went red.

2. **Critical findings:** none.

3. **Important findings**
   - **A cold target that fails slowly never reports why** (`cmd/internal/couchcmd/hub_target_memo.go:401-416`). Only successful resolutions are remembered. Suppose a resolution fails after the waiting admission's budget has run out (for example Discover's "slot candidate limit exceeded", "slot host replaced", or an `sdlc` failure on a slow host). Its error is dropped when the in-flight entry is deleted. The next `Warm` starts a new resolution and refuses with "warming; retry" again, so a retry loop never shows the cause. That is the "no diagnosis" failure this issue exists to fix. **Fix:** keep the last failure next to the entry for a short time and return it, wrapped with its phase, to the next `Warm`. Add a test where the resolver fails after the budget and the next admission shows the real error.
   - **Hub-move admission still has the original problem** (ARCH-PURPOSE; `cmd/internal/couchcmd/hub_move.go:84-90`). `handleHubMove` calls `s.authority.hubTarget` (now `memo.Fresh`) once per comma-separated slot, synchronously, with no budget. Each call is a full Discover through `WorkspaceReferencePath`, about 1 s on pair after this change. So moving two slots goes past the 2 s exchange and fails with the same undiagnosed client deadline. The move job does not re-prove its slots on fresh discovery, so simply switching it to `Warm` would be unsafe. **Fix:** either add a budget and warm sources with a fresh re-proof inside the move job, or write a plan Revision that explicitly scopes hub-move out as a non-goal.

4. **Minor findings**
   - ARCH-DRY: the admission closure (`hub_guards.go:222-242`) repeats the structure of `guard` and differs only in where the origin and target come from. One helper that takes those two sources would cover both.
   - Plan drift: the plan says the memo is keyed by "canonical target reference", but the code uses the raw `r.Target`. Aliases and prefixes therefore get separate entries and separate in-flight resolutions. This is harmless, but the plan text should match the code.
   - ARCH-CONSTRAINTS: concurrent Discover runs as wide as the row count, up to 128, with each row spawning `sdlc` plus two `git` calls. Background resolutions are also unbounded across distinct references. A small semaphore (8–16) would protect a co-tenanted host at load 30. The concurrency test's "bounded" check (`p4 > 4`) only re-states the row count.
   - ARCH-FUNERAL: memo entries are overwritten but never removed. That is fine in practice because they are bounded by the set of resolvable reference spellings, but the bound should be written down.
   - ARCH-ORDER: `remember` is last-writer-wins by completion time and stamps `now()` at completion. A slower, older resolution can overwrite a newer one and get a fresher timestamp. This is harmless because prepare re-proves, but it should be noted.

5. **Test coverage notes**
   - The admission tests (3) and memo tests (2) pass under `-race`. The target-source mutation is red on all three admission tests.
   - Not run here: `TestDiscoverVerifiesSlotsConcurrently` and `TestDiscoverStillFailsWholeInventoryOnReplacedHost`, because of the `git init` confinement. The pre-existing `TestDiscoverUnionsNumberSources` fails the same way, so this is the environment, not the diff.
   - A full run of the couchcmd package showed other failures caused by the same confinement. It also reported one race in `TestHubSupervisionReassignmentChangesBinding`, which does not touch the changed code; 45 isolated `-race` reruns of that test passed. I treat it as an existing flake, not something this diff introduced.
   - Gaps: there is no admission test on warm sources for `recovery-confirmation` or `recover-park`, both of which this diff changed. There is no relaunch-op admission test (Done-when says "relaunch of an idle slot"), though the admission path doesn't depend on the op. There is no test for a slow failure on a cold target (see Important #1).

6. **Architectural notes for upcoming work**
   - ARCH-PURE: pass. The memo has an injected clock and resolver.
   - ARCH-MOCK: pass, on the same terms as the existing fixtures. The Discover tests wrap a real git fixture that adds latency, and there is no new external dependency.
   - ARCH-SECURE: pass. Target references are validated by the resolver, the memo holds only data this process produced, and no credentials are involved.
   - ARCH-EVOLVE: N/A. The memo is in memory only and nothing is persisted.
   - ARCH-DRY, ARCH-PURPOSE, ARCH-CONSTRAINTS, ARCH-ORDER and ARCH-FUNERAL: flagged above.
   - For M2: hold the one-wide effect gate only after the native-scan skip is in place, since that scan is the next bottleneck once admission passes.
   - Docs: the atlas is updated (`atlas/couch.md:1106`). README needs nothing, since no flags, keys or subcommands were added.

7. **Plan revision recommendations**
   - Add a Revision saying the memo key is the raw target reference, not a canonical form (or make the code canonicalize it).
   - Add a Revision deciding hub-move admission: either in scope with fresh re-proof in the move job, or an explicit non-goal.
   - Add a Revision saying that a cold resolution's failure is remembered and shown to the next admission, if that fix is taken.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: refusal-must-carry-reason
    title: |
      Cold target whose resolution fails after the budget drops the error; retries see only "warming" forever
    detail: |
      hub_target_memo.go Warm caches only successes. A failed flight is deleted, so each retry starts a new resolution and, past the 1.5 s budget, refuses with "warming; retry" and never the real error (e.g. slot candidate limit, host replaced, sdlc failure). Remember the last failure briefly and return it to the next Warm with its phase; add a test where the resolver fails after the budget.
  - id: new
    severity: Important
    kind: defect
    family: admission-sync-discover-over-exchange
    title: |
      handleHubMove still runs one fresh Discover per named slot synchronously, with no admission budget
    detail: |
      hub_move.go:84-90 calls authority.hubTarget (memo.Fresh, a full Discover of about 1 s) per comma-separated slot. A two-slot move goes past the 2 s exchange and gets the same undiagnosed client deadline this issue fixes for lifecycle ops (ARCH-PURPOSE sibling). Either add a budget and warm sources with a fresh re-proof in the move job, or record it as a non-goal in the plan Revisions.
  - id: new
    severity: Minor
    kind: unverifiable
    family: review-env-git-confinement
    title: |
      couchcore Discover concurrency and replaced-host tests could not be run in the review sandbox
    detail: |
      Ran go test -race -run TestDiscover ./cmd/internal/couchcore/. The .sdlc-review git wrapper refused git init under GOTMPDIR=/Volumes/TestScratch/tmp/pair-slot2 ("outside the reviewer's directories"). The pre-existing TestDiscoverUnionsNumberSources fails the same way. Host evidence: a passing -race run of TestDiscoverVerifiesSlotsConcurrently and TestDiscoverStillFailsWholeInventoryOnReplacedHost, plus a red run with the SameFile fence removed.
  - id: new
    severity: Minor
    kind: defect
    family: bounded-fanout
    title: |
      Discover fan-out and per-reference background resolutions have no small concurrency cap
    detail: |
      Rows run as wide as the row count (up to 128, each spawning sdlc and two git calls), and every distinct target spelling starts its own lifetime-scoped Discover. A semaphore of 8-16 would bound the burst on a co-tenanted host. The concurrency test's p4 > 4 check only re-states the row count.
  - id: new
    severity: Minor
    kind: defect
    family: plan-code-drift
    title: |
      Memo is keyed by raw r.Target, not the "canonical target reference" the plan states
    detail: |
      Aliases and prefixes get separate entries and separate in-flight resolutions. Harmless, but either canonicalize the key or revise the plan text.
  - id: new
    severity: Minor
    kind: defect
    family: duplicated-guard-structure
    title: |
      Admission closure repeats the guard closure and differs only in where origin and target come from
    detail: |
      hub_guards.go:222-242 vs 128-150 (ARCH-DRY). One helper that takes the origin and target sources would keep the two checks from drifting apart.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"fb70bc95-dc2b-40bb-bcb8-5bd1af5e4792","repository":"github.com/xianxu/pair","issue_id":"000726","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/879fd0b1a3c18124","reviewed_head":"4dd0a6107ef9867aebde532f2027b10b86f72473","slot":"pair:2","requested_at":"2026-10-20T15:25:14.797205Z"},"phase":"finished","pid":78181,"started_at":"2026-10-20T15:25:14.931735Z","finished_at":"2026-10-20T15:33:43.749211Z","session_id":"da2e0f22-e2d2-4353-8e81-15232ab15738","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.980584s
head: "4dd0a6107ef9867aebde532f2027b10b86f72473"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409

---

## Re-review — 2026-10-20T08:54:16-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 726 — Hub lifecycle ops time out client-side (resume pair:6, relaunch parley.nvim:4: 'context deadline exceeded') with no diagnosis |
| repo | pair |
| issue file | workshop/issues/000726-hub-lifecycle-ops-time-out-client-side-resume-pair-6-relaunch-parley-nvim-4-context-deadline-exceeded-with-no-diagnosis.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | e2ac2a19f253961b715b6edf104ca82d72b5c40a..906f1c91c30bfdb59be27fff15b74608eb986bb4 |
| command | sdlc milestone-close --issue 726 --milestone M1 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-20T08:54:16-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Verifying the prior findings' fixes in the memo and move code; now checking the guards diff and running tests.

Retrying the sandbox-blocked Discover tests with the test temp dir pointed inside the checkout.

Discover tests pass with that setting. Now reverting each claimed fix to confirm its test goes red.

All four fixes go red when reverted. Checking the issue's plan and revisions against the code.

Checking the `HubSlot` shape to judge whether a stale memo entry can wrongly refuse.

Running the full couchcmd and couchcore suites to check for regressions.

`TestDiscoverVerifiesSlotsConcurrently` failed 1 of 3 isolated runs at load ~57. Reading its timing assertion.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

This round fixes both blocking findings from round 1, and the fixes cover the whole class of bug, not just the one site each finding named. A failed cold resolution now returns its own error for 30 s instead of "warming" forever (BR-1). Move-slot admission names its slots from the warm memo inside the admission budget, and the queued move re-checks each slot fresh before any effect (BR-2). I reverted each fix in the checkout and its regression test failed every time:
- Not remembering failures fails `TestHubTargetMemoRemembersALateFailure`.
- A fresh resolution at move admission fails with "ran 1 fresh resolutions".
- Skipping the prepare re-check fails with "moved slot: <nil>".
- Removing the Discover gate fails with "peak concurrency 10 over the cap 8".
- Removing the flight cap fails `TestHubTargetMemoKeysCanonicallyAndCapsFlights`.

The Discover tests that the sandbox blocked last round run and pass once `GOTMPDIR` points inside the reviewer's directories. Before shipping, one thing needs fixing: the new concurrency test checks elapsed time, and it fails on this loaded host. It failed in the full-suite run and in 1 of 3 isolated runs at load average ~57. The rest is Minor.

1. **Strengths**
   - `checkHubSlotOperation` (`cmd/internal/couchcmd/hub_guards.go:181`) puts one check behind two sources: fresh for the queued guard, warm for admission. The refactor keeps behavior; the RecoverPark admission override still applies.
   - `hubTargetMemo.Fresh` only remembers a failure when the caller's context is still live (`hub_target_memo.go:79`). A cancelled resolution proves nothing, so it isn't cached.
   - The move job compares the warm and fresh `HubSlot` in prepare (`hub_move.go:111-118`). It now re-checks closer to the effect than before this change, when the only resolution was at admission.
   - The Discover gate is a counting semaphore inside each row's goroutine, and the deterministic lowest-numbered fatal error is kept (`slotcatalog.go:204-213`).

2. **Critical findings:** none.

3. **Important findings**
   - **Elapsed-time check makes the Discover test flaky** — `cmd/internal/couchcore/slotcatalog_concurrent_test.go:74-77`. The test fails if 4-slot discovery takes 3/4 of the serial time. I saw "4-slot discovery took 454ms, close to the serial 600ms" in 1 of 3 isolated runs and in the full `-race` suite. The peak-concurrency counts (`p4 >= 4`, `p10 <= cap`) already prove concurrency without timing. Fix: delete the elapsed-time check, or replace it with a barrier that releases only once N rows are active.

4. **Minor findings**
   - **BR-5 still open:** `hubTargetKey` changes nothing. `ParseWorkspaceReference` only accepts canonical numbers and `String()` gives back the same text, so `:6` and `pair:6` still get separate entries. The atlas line "keys are canonical `repo:N`" (`atlas/couch.md:1113`) is therefore wrong. Either drop the helper and fix the atlas, or really resolve `:N` to `repo:N`.
   - **`hubReceiptFilter` has no budget:** it calls `warmHubTarget(ctx, …)` on the request context (`hub_guards.go:249`), so a cold `operation-status` poll waits as long as the background resolution takes. This is the 2nd finding in family `admission-sync-discover-over-exchange` — see the rule in the findings block.
   - **No-op guard closure:** the guard's `phase: func(_ string, err error) error { return err }` exists only to fit the struct. A nil `phase` meaning "no labeling" would be lighter. Nit.
   - **Move admission resolves slots one after another:** each slot's resolution only starts after the previous one finishes. A cold two-slot move uses up the budget on slot 1 and refuses "warming" on slot 2, so it needs one retry. Starting every resolution before waiting on any would avoid that.
   - **Older result can overwrite a newer one (ARCH-ORDER):** `remember` stamps the time a resolution finished, not when its observation began. A background resolution that started earlier can finish after a newer `Fresh` and overwrite it. The damage is limited to one wrong admission answer for up to 30 s or 10 min, and the queued guard re-proves before any effect.
   - **Memo entries are never removed (ARCH-FUNERAL):** failures are now remembered under any target string a caller sends, and entries are never deleted, only ignored once old.
   - **Plan text drifted:** the Plan still says Discover runs "at most `MaxSlotCandidates`" at a time; the code caps it at 8. The state table has no row for "remembered failure (30 s)" and the move design has no fresh re-check in prepare. These changes are recorded only in the Log.

5. **Test coverage notes**
   - Each claimed fix has a test that fails when the fix is reverted (verified above).
   - `go vet` is clean. The targeted `-race` runs pass.
   - The full `couchcmd` run also failed `TestNativeExitAdmitsOnlyTheCurrentChild` once. It passed 3/3 in isolation, and that code is outside this diff, so it is a pre-existing flaky test, not a finding here.
   - The 10-slot check has an upper bound but no lower bound (`p10 >= 8` is missing). A gate of 1 would still pass it, though `p4 >= 4` covers the concurrency.

6. **Architecture (ARCH-\*)**

   | Principle | Result | Note |
   |---|---|---|
   | ARCH-DRY | pass | BR-6 consolidated |
   | ARCH-PURE | pass | memo is thin glue with an injected resolver and clock |
   | ARCH-PURPOSE | pass with a Minor | move sibling swept; receipt filter unbudgeted |
   | ARCH-MOCK | pass | latency `ProvisionIO` fake; injected resolver |
   | ARCH-CONSTRAINTS | pass | both fan-outs capped at 8; one test flaky by elapsed time (Important) |
   | ARCH-SECURE | pass | remembered errors go only to authenticated callers on the same supervisor |
   | ARCH-ORDER | flag (Minor) | older result can overwrite a newer one |
   | ARCH-FUNERAL | flag (Minor) | memo map has no removal path |
   | ARCH-EVOLVE | N/A | no persisted format changed; the memo is in-memory only |

7. **Plan revision recommendations**
   - Add a `## Revisions` entry recording:
     - the Discover cap of 8 (replacing `MaxSlotCandidates`) and the background-resolution cap of 8;
     - remembered failures with a 30 s window, plus a state-table row for "resolution failed → refused with its own error; retry after 30 s";
     - move-slot admission on the warm memo, with a fresh re-check per slot in prepare;
     - the actual memo key, either raw reference or truly canonical.
   - Correct the ARCH-FUNERAL note "never accumulated": entries are replaced per reference but never removed.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Failures remembered 30 s (hub_target_memo.go:79-81,97-100); TestHubTargetMemoRemembersALateFailure goes red when Fresh stops remembering errors.
  - id: BR-2
    disposition: addressed
    note: |
      handleHubMove admits on warmHubTarget under slotAdmissionBudget and prepare re-resolves fresh; test red with fresh admission (1 resolution) and with the prepare check disabled.
  - id: BR-3
    disposition: withdrawn
    note: |
      With GOTMPDIR inside .sdlc-review/tmp, TestDiscoverVerifiesSlotsConcurrently, TestDiscoverStillFailsWholeInventoryOnReplacedHost and TestDiscoverUnionsNumberSources pass under -race.
  - id: BR-4
    disposition: addressed
    note: |
      slotVerifyConcurrency=8 gate plus hubTargetMaxFlights=8; the 10-slot peak check and the flight-cap test each go red when the cap is removed.
  - id: BR-5
    disposition: not-addressed
    note: |
      hubTargetKey is identity on every accepted input, so :N and repo:N still key separately and the atlas claim "keys are canonical repo:N" is wrong; drop the helper and fix the text, or really canonicalize.
  - id: BR-6
    disposition: addressed
    note: |
      checkHubSlotOperation with hubSlotSources serves both guard and admission (hub_guards.go:171-201); existing admission and guard tests pass.
findings:
  - id: new
    severity: Important
    kind: defect
    family: wall-clock-test-oracle
    title: |
      TestDiscoverVerifiesSlotsConcurrently asserts elapsed time and fails under host load
    detail: |
      slotcatalog_concurrent_test.go:74-77 fails when 4-slot wall time is at least 3/4 of serial; it failed in 1 of 3 isolated -race runs ("took 454ms, close to the serial 600ms") and in the full suite at load ~57. Peak-concurrency counts already prove concurrency; delete the elapsed-time check or use a barrier.
  - id: new
    severity: Minor
    kind: defect
    family: admission-sync-discover-over-exchange
    title: |
      hubReceiptFilter waits on a cold memo resolution with no budget
    detail: |
      2nd finding in this family. Rule: every request-path read of a target slot must be bounded by the admission budget, so Warm should cap its own wait at slotAdmissionBudget rather than relying on each caller. hub_guards.go:249 calls warmHubTarget on the raw request ctx for operation-status polling.
  - id: new
    severity: Minor
    kind: defect
    family: plan-code-drift
    title: |
      Plan design and state table not revised for the round-2 behavior changes
    detail: |
      2nd finding in this family. Rule: any behavior change made in a review round updates the Plan design and state table (or a Revisions entry) in the same commit, not only the Log. The Plan still says Discover is bounded by MaxSlotCandidates; it lacks the 30 s remembered-failure state and the move prepare re-check.
  - id: new
    severity: Minor
    kind: defect
    family: observation-ordering-stamp
    title: |
      Memo stamps completion time, so an older in-flight resolution can overwrite a newer Fresh result
    detail: |
      remember() records now() when the resolution finishes; a background flight started before a prepare-time Fresh can land after it and overwrite it. The queued guard re-proves before any effect, so the damage is a wrong admission answer for up to 30 s or 10 min. Stamp the observation start and keep the newer entry.
  - id: new
    severity: Minor
    kind: defect
    family: durable-cache-names-removal
    title: |
      hubTargetMemo entries are never deleted; failures for arbitrary target strings now accumulate
    detail: |
      Entries expire logically but stay in the map for the supervisor's lifetime, and this round adds error entries for any reference a caller sends. Delete aged entries on lookup or sweep them, and correct the plan's ARCH-FUNERAL note.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"fcc001b1-7b4d-4a27-90af-4da7c40ff033","repository":"github.com/xianxu/pair","issue_id":"000726","recipe":"milestone-review","round":2,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/6fed915de3c23228","reviewed_head":"906f1c91c30bfdb59be27fff15b74608eb986bb4","slot":"pair:2","requested_at":"2026-10-20T15:42:08.65861Z"},"phase":"finished","pid":98973,"started_at":"2026-10-20T15:42:08.751383Z","finished_at":"2026-10-20T15:54:14.23728Z","session_id":"aef88984-fb13-4d53-abe2-97fcf6ef66bd","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.406623417s
head: "906f1c91c30bfdb59be27fff15b74608eb986bb4"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T09:04:14-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 726 — Hub lifecycle ops time out client-side (resume pair:6, relaunch parley.nvim:4: 'context deadline exceeded') with no diagnosis |
| repo | pair |
| issue file | workshop/issues/000726-hub-lifecycle-ops-time-out-client-side-resume-pair-6-relaunch-parley-nvim-4-context-deadline-exceeded-with-no-diagnosis.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | e2ac2a19f253961b715b6edf104ca82d72b5c40a..5ab066922b8fcd0ffbc58fb4d83e6d420575a72c |
| command | sdlc milestone-close --issue 726 --milestone M1 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-20T09:04:14-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking the prior findings against the code; next I'll run the tests.

Couchcmd tests pass; couchcore tests are blocked by this sandbox's git confinement, so I'm checking the rendezvous test by reading the code.

Review done. Prior findings BR-5, BR-7, BR-9, BR-10 and BR-11 are fixed. BR-8 stays open (Minor), and I found one new Minor issue. Nothing blocks the boundary.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M1 does what it set out to do. Slot-operation admission now uses warm sources inside a 1.5 s budget that sits under the 2 s exchange: the caller's connected workspace and `hubTargetMemo`. Queue prepare and the effect gate still re-prove authority on fresh discovery before any effect. Round 2's fixes check out in the code:
- The concurrency test now waits until all slot probes are in flight together, with no elapsed-time assertion.
- Memo entries carry their start time, so a late older result can't replace a newer one.
- Entries are swept and capped at 256.
- The receipt filter is now inside a budget.
- The plan gained a `## Revisions` entry.

I ran the targeted couchcmd tests (`-race -count=5`): memo, admission and move tests all pass. I also undid the start-time check (`old.at.After(started)`) and `TestHubTargetMemoOrdersByStartAndSweeps` went red, then put it back; the checkout is clean.

I could not run any couchcore test here. Every fixture's `git init` is refused by this sandbox's limits ("outside the reviewer's directories (sdlc #339)"). So I checked the BR-7 test by reading it, which is why confidence is medium.

Two Minor items remain:
- **BR-8:** each caller adds its own budget, but `Warm` doesn't enforce one itself, and no test pins the receipt-filter budget.
- **New:** a failed resolution that takes longer than 30 s is remembered as already expired. That brings back BR-1's "warming forever" symptom.

1. **Strengths**
   - `checkHubSlotOperation` (`hub_guards.go:178`) is one check with fresh or warm sources passed in. Admission and the queued guard can no longer drift apart (ARCH-DRY).
   - The concurrency test `rendezvousIO` (`slotcatalog_concurrent_test.go:18-65`) proves concurrent probes without timing them. Holding the count of probes in flight at once, rather than a running total, and leaving out the primary's own probe, are the right choices. A serial pass can never meet the rendezvous.
   - In `handleHubMove` (`hub_move.go:87-124`), admission names slots from the memo under one shared budget. The queued job then re-resolves each slot fresh and refuses on any difference. That is ARCH-ORDER done properly: fresh evidence before the effect.
   - The memo's resolver and clock are injected. Tests drive ordering with channels, not sleeps, e.g. `TestHubTargetMemoOrdersByStartAndSweeps`.
   - `Fresh` refuses to remember a cancelled resolution (`hub_target_memo.go:84`), so a timeout is never stored as a finding about the target.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - **Expired-on-arrival failure** (`hub_target_memo.go:158` and `:102`): failure age is counted from when the resolution started.
     - A failed resolution that ran for 30 s or more is stored already expired. Each retry then starts a new flight and gets "warming", and the operator never sees the cause.
     - It's realistic: `ProbeTimeout` is 5 s per probe, with several probes per row, run in waves of 8 rows. A hung `sdlc` easily pushes a Discover past 30 s.
     - Fix: keep the start time for ordering, and add a finish time that the failure-age check and the sweep use.
   - BR-8 remains (see dispositions).
   - Paragraph in `atlas/couch.md` (around line 1106): doesn't mention the 256-entry cap or the start-time ordering.
   - The `Warm` waiter can get a result that `remember` rejected as older (`hub_target_memo.go:128`). This only affects admission answers, not effects.

5. **Test coverage notes**
   - These couchcmd tests pass: memo, admission, move and refused-admission.
   - couchcore Discover tests can't run in this sandbox. The rendezvous logic was checked against `verifyHost` (`provision.go:188`): there is exactly one `rev-parse --absolute-git-dir` per slot host, which matches what the fake waits on.
   - Missing tests:
     - Nothing covers the receipt filter with a slow resolution.
     - Nothing covers a failing resolution that takes longer than `hubTargetFailureAge`. The fake clock makes that test cheap.

6. **Architectural notes**
   - **Pass:**
     - ARCH-DRY: one check, two sources.
     - ARCH-PURE: logic plus an injected resolver and clock.
     - ARCH-PURPOSE: admission always answers with a typed outcome inside the budget.
     - ARCH-MOCK: the resolver and catalog fakes share the production seam.
     - ARCH-CONSTRAINTS: 1.5 s budget, 8 flights, 8 Discover rows.
     - ARCH-SECURE: target references come only from callers that passed `liveCaller` first, and entries and flights are capped.
     - ARCH-FUNERAL: sweep plus a 256 cap.
     - ARCH-EVOLVE: no persisted format changes; `manifest.go` only adds a source to a list.
   - **Flag (Minor):**
     - ARCH-ORDER: the failure-age stamp, above.
     - ARCH-CONSTRAINTS: the budget is set at each call site, not inside `Warm` (BR-8). Three call sites each repeat `context.WithTimeout(ctx, slotAdmissionBudget)`.
   - For M2: the same "warm read on the request path, fresh read before the effect" split should govern the recovery-evidence skip.

7. **Plan revision recommendations**
   - Add to `## Revisions`: a failure's 30 s age counts from when its resolution finishes.
   - Fix the Design text that still says Discover runs "at most `MaxSlotCandidates` at a time", and the ARCH-FUNERAL note that says the memo is "never accumulated". Revisions already corrects both, but the Design text still contradicts it.

```findings
dispose:
  - id: BR-5
    disposition: addressed
    note: |
      hubTargetKey (hub_target_memo.go:39) makes repo:N keys canonical via ParseWorkspaceReference; pinned by TestHubTargetMemoKeysCanonicallyAndCapsFlights (passes).
  - id: BR-7
    disposition: addressed
    note: |
      Elapsed-time check removed; rendezvousIO waits until min(slots,8) slot probes are in flight together, which a serial pass can never reach. Checked against verifyHost (one probe per slot host); couchcore can't run here because of sandbox git limits.
  - id: BR-8
    disposition: not-addressed
    note: |
      All three warmHubTarget callers now add their own slotAdmissionBudget, but Warm doesn't cap its own wait as the rule asked, and no test fails without the receipt-filter budget (hub_guards.go:250).
  - id: BR-9
    disposition: addressed
    note: |
      Plan Revisions now lists the 30 s failure age, the move prepare re-check, start-time ordering, the 256 cap, 8-wide fan-out and two new state-table rows.
  - id: BR-10
    disposition: addressed
    note: |
      remember() stamps the start time and keeps the newer entry; undoing the old.at.After(started) check turns TestHubTargetMemoOrdersByStartAndSweeps red (checked here).
  - id: BR-11
    disposition: addressed
    note: |
      remember() deletes expired entries and evicts the oldest past hubTargetMaxEntries=256; the test checks both the cap and the expiry sweep.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: refusal-must-carry-reason
    title: |
      A failed resolution that runs 30 s or more is stored already expired, so admission says "warming" forever again
    detail: |
      2nd finding in this family. Rule: an error from a finished resolution must reach the next admission for the full hubTargetFailureAge, counted from when it finished. Since BR-10, remember() stamps start time (hub_target_memo.go:158) and Warm and the sweep age failures from it (:102, :145). With 5 s ProbeTimeout probes across 8-wide waves of rows, a hung sdlc pushes Discover past 30 s, every retry starts a new flight and refuses "warming", and the cause never shows. Fix: keep the start time for ordering and add a finish time for failure expiry; test with a fake-clock resolver slower than hubTargetFailureAge.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ce2f22b6-150a-483b-b9e4-187226fefd93","repository":"github.com/xianxu/pair","issue_id":"000726","recipe":"milestone-review","round":3,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/ae32fa2026898ccb","reviewed_head":"5ab066922b8fcd0ffbc58fb4d83e6d420575a72c","slot":"pair:2","requested_at":"2026-10-20T16:00:31.437011Z"},"phase":"finished","pid":98682,"started_at":"2026-10-20T16:00:31.52907Z","finished_at":"2026-10-20T16:04:13.321601Z","session_id":"38388f12-42cc-4551-86be-c57037497a57","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.573312333s
head: "5ab066922b8fcd0ffbc58fb4d83e6d420575a72c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
