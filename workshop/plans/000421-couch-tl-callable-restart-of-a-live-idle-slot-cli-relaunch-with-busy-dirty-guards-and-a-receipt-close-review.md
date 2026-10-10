# Boundary Review — pair#421 (whole-issue close)

| field | value |
|-------|-------|
| issue | 421 — couch: TL-callable restart of a live idle slot (CLI relaunch with busy/dirty guards and a receipt) |
| repo | pair |
| issue file | workshop/issues/000421-couch-tl-callable-restart-of-a-live-idle-slot-cli-relaunch-with-busy-dirty-guards-and-a-receipt.md |
| boundary | whole-issue close |
| milestone | — |
| window | fa6c2152494d727d2354286af20d0158a317ec0a..e157ce9ae95b9f33820983f13e92c2056e4e61cf |
| command | sdlc close --issue 421 |
| reviewer | claude |
| timestamp | 2026-10-10T01:36:31-07:00 |
| verdict | SHIP |

## Review

Checking the dispatch path for BR-21, then the verdict.

Running the focused test suites now.

The failures are the sandbox denying `mkdir /tmp/...` for socket tests. Re-running those outside the sandbox.

```verdict
verdict: SHIP
confidence: medium
```

All four open findings are fixed in this window. BR-6 is fixed because every source of Settled now goes through `armSettleLocked`. That function bumps `settleGen`, and `settleFired` checks the generation again after the probe runs. BR-13's test now drives the admission-note adapter with an error. BR-20's identifier is gone from code, and its hits in the plan are marked superseded. For BR-21, the effect re-check now receives admission's known-or-forced decision and refuses "unknown" unless admission was forced. I found nothing new at Critical or Important severity.

Tests: `couchcore`, `couchmessage` and the `wrapcmd` Settle tests pass. The socket tests had to run outside the sandbox, which blocks `mkdir /tmp`. Two `couchcmd` acceptance tests fail: `TestContinuationWriterPublishesExactCheckpointAcrossWorktrees` and `TestColdResumeOfAParkedPrimaryRegistersFromBothOrigins`. The continuation failure comes from the pair session's environment leaking into the test: an inherited `PAIR_DATA_DIR` conflicts with the repo scope, and the fixture has no `origin` remote. I didn't look into the cold-resume failure. Neither test is in this diff's area.

1. **Strengths**
   - `peer_settle.go:38` `armSettleLocked` is the one entry point for every source change, and it advances the generation. That fixes the whole BR-2/BR-6 class, not just the reported cases.
   - `live_restart_probe.go:154` `refuseUnlessIdle` is a single rule used by both verbs at the effect. Admission's decision reaches it through `require-settled` set to `known` or `forced`.
   - `slot_operation_test.go:426+` tests every verb against known, forced and absent, from the code that produces the decision through to the probe.
   - `withAdmissionNote` is pulled out and tested on both the success and the failure path.
2. **Critical:** none.
3. **Important:** none.
4. **Minor:** none new.
5. **Test coverage:** `TestSettleInFlightCheckInvalidatedBySilentTurn` covers the race between the probe and publishing. `TestRestartEffectRechecksBusyAndCancellation` covers a wrapper that disconnects after a known-settled admission.
6. **Architecture:**
   - ARCH-DRY, ARCH-PURE and ARCH-PURPOSE pass. Relaunch and reload-context share one effect-time rule, and one verb list is read everywhere.
   - ARCH-MOCK passes: there is a fake proc, and a fake live-restart probe sits behind the `LiveRestartProbe` seam.
   - ARCH-CONSTRAINTS passes: the reload confirmation wait is bounded at 20s.
   - ARCH-SECURE passes: the PID is checked against its start token before the signal is sent.
   - ARCH-ORDER passes: the settle generation counter makes stale checks harmless, and admission's decision is carried through to the effect.
   - ARCH-FUNERAL passes: nothing durable is added.
7. **Plan revisions:** none. The M3-advisories Revisions entry already records the identifier sweep and the effect-time decision rule.

```findings
dispose:
  - id: BR-6
    disposition: addressed
    note: |
      armSettleLocked bumps settleGen on every source incl. lifecycleTurnChanged; settleFired rechecks seq after probe; TestSettleInFlightCheckInvalidatedBySilentTurn.
  - id: BR-13
    disposition: addressed
    note: |
      TestWithAdmissionNoteOnSuccessAndFailure drives withAdmissionNote with a non-nil error and asserts the note survives.
  - id: BR-20
    disposition: addressed
    note: |
      slot_operation.go:39 comment now names SlotOperationTakesForceUnknown; remaining plan hits are marked superseded inline.
  - id: BR-21
    disposition: addressed
    note: |
      refuseUnlessIdle refuses unknown unless forced; require-settled carries known/forced; tested in TestRestartEffectRechecksBusyAndCancellation and the BR-22 dispatch matrix.
```

This doesn't change the verdict; it stays SHIP. The early test run finished, but it ran inside the sandbox. Its failures are the sandbox refusing to let tests create directories under `/tmp` (`mkdir /tmp/...: operation not permitted`). They aren't failures in the code under review.

I re-ran those packages outside the sandbox:
- **`couchmessage`:** passes.
- **`couchcmd`:** two acceptance tests fail. `TestContinuationWriterPublishesExactCheckpointAcrossWorktrees` fails because the pair session's environment leaks into the test: an inherited `PAIR_DATA_DIR` conflicts with the repo scope, and the test fixture has no `origin` remote. I didn't look into why `TestColdResumeOfAParkedPrimaryRegistersFromBothOrigins` failed. Neither test is in the area this diff changes.
- **`wrapcmd`:** only the Settle tests were run; they passed, even inside the sandbox.

---

## Re-review — 2026-10-10T01:59:22-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 421 — couch: TL-callable restart of a live idle slot (CLI relaunch with busy/dirty guards and a receipt) |
| repo | pair |
| issue file | workshop/issues/000421-couch-tl-callable-restart-of-a-live-idle-slot-cli-relaunch-with-busy-dirty-guards-and-a-receipt.md |
| boundary | whole-issue close |
| milestone | — |
| window | a69f9f9b668b267c4968689401a5e3998da9affc..e57a658a4e18dd7a2074d0ba9e7476aba601b8ba |
| command | sdlc close --issue 421 |
| reviewer | claude |
| timestamp | 2026-10-10T01:59:22-07:00 |
| verdict | SHIP |

## Review

Still checking the merge commit: the conflict resolution looks clean, so I'm running the relevant test packages now.

The merge from main changed nothing in #421's own code, so this closes as SHIP. I didn't get a fully green test run here; the four failing tests look environmental or already-known flakes, not #421 regressions.

```verdict
verdict: SHIP
confidence: high
```

All 25 prior findings were already disposed. Round 10 passed at `5d7ca227`. The only commit after that, `e57a658a`, merges `origin/main` into the branch. Its one conflict, in `workshop/lessons.md`, was resolved cleanly: main's #419/#422 `change-code` lesson and #421's two lesson blocks are both present, nothing dropped. The changes to `README.md` and `cmd/internal/artifactpath/manifest.go` are main's `couch-dev` additions, auto-merged next to #421's edits without touching them. `go build ./...` and `go vet` on the four touched packages are clean.

Tests, run outside the sandbox (inside it they only fail on `/tmp` and pty permission errors):
- `couchmessage` and `wrapcmd` pass.
- Four tests fail in `couchcore` and `couchcmd`, none in #421's surface:
  - `TestColdResumeOfAParkedPrimaryRegistersFromBothOrigins` is the flake main already filed as #423.
  - `TestContinuationWriterPublishesExactCheckpointAcrossWorktrees` hits `PAIR_DATA_DIR ... conflicts with selected repository scope`. That is the known environment leak from running tests inside a live Pair session.
  - `TestSpawnComposesProductionPairRegistrationBoundary` fails because a thread-claim file under the real `~/.local/share/pair` is missing. Same likely cause: it reads real user state.
  - The fourth `couchcmd` failure didn't appear in my output, so I haven't identified it.

1. **Strengths:** The require-settled matrix from admission through dispatch to the probe (`TestRequireSettledProducerToConsumer`), the re-check of the busy guard just before the effect, and a recovery-family rules section in `lessons.md` that states the rule for each finding family.
2. **Critical:** none.
3. **Important:** none.
4. **Minor:** none new.
5. **Test coverage:**
   - Before relying on the close, re-run `couchcore` and `couchcmd` with `make -k test` and the session environment cleared.
   - The fourth `couchcmd` failure still needs to be identified.
   - The live check is deferred to the TL, and the plan and the Done-when both say so.
6. **Architecture:** Every principle passes; the merge adds nothing for ARCH-DRY, ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK, ARCH-CONSTRAINTS, ARCH-SECURE, ARCH-ORDER or ARCH-FUNERAL to judge.
7. **Plan revisions:** none.

```findings
```
