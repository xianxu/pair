---
id: 000432
status: open
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: 'b780b7ad34ba502e4eb1deabf67ff8d8cc2ede57' # card fields mirrored from issue-cards; edit via sdlc
---

# couchcore live conformance: quit-trigger idempotency and cold-launch ownership poll

## Problem

`couch-zellij-conformance` step 2 (`make -f Makefile.local test-couch-zellij-live`)
had not run on CI for weeks: step 1 failed first on every run (pair#324). Once
pair#324 fixed step 1, step 2 showed that the couchcore live fixtures had gone
stale under #355 M1 (df48593b, 2026-09-30). That change made
`launcher.ClassifySessionOwner` require a live `pair wrap`/`nvim` pane whose
argv is unambiguous. pair#324 modernized the fixtures:
- `ControlledZellij.AddOwnerWrapperPane` gives each session a stub
  `pair wrap --scrollback-log <raw>` pane.
- `standInPaneScript` keeps JSON and `$` out of stand-in pane argv.

With that, ownership resolves and Recovery/warm passes. Three cases still fail
past ownership, so pair#324 excludes them by name in the workflow:

1. **`TestParkLifecycleLive`: duplicate quit trigger.**
   `pairlifecycletest.RunConformanceScenario` delivers the trigger twice, as an
   idempotency check. The first `TriggerQuit` deletes the session. The second
   probes it as Absent, and `TriggerBoundQuit`
   (`cmd/internal/couchcore/artifactcollision.go`) rejects an owner-less
   binding with "quit intent does not match observed session owner".
   (`TestParkLifecycleLiveIntentOnlyMutation` passes; it uses a different
   trigger.)
2. **`TestContinuationZellijSeedTransportLive`: cold-launch ownership poll.**
   `awaitResumeRegistration` polls for 14s and ends with
   `last ownership check: not owned`, with no owner observation attached.
   Undiagnosed. One suspect: `env.Couch.Artifacts` is the test env's checker,
   not the `dataDir` checker the fixture indexed and gave a wrapper pane.
3. **`TestRecoveryRealHelperAndSessionConformanceLive/checkpoint`: same
   poll.** It ends with `not owned (absent)`, even though `AfterAcknowledge`
   restarts the fixture session and adds the wrapper pane. Undiagnosed.

Also seen while verifying pair#324, both pre-existing and outside the
conformance job (it runs only live `-run` selections):
- The full `go test ./cmd/internal/couchcore` takes longer than 15 minutes
  (timed out, with the last running test only 2s in).
- `TestSpawnComposesProductionPairRegistrationBoundary` fails identically on
  origin/main.

## Spec

- **Product decision (duplicate quit):** is a second quit trigger for a session
  already Absent a success, meaning already quiesced? The conformance scenario
  says delivery must be idempotent. `TriggerBoundQuit` currently refuses. Decide
  it, record the decision here, then fix whichever side is wrong (the product
  or the scenario).
- Diagnose the cold-launch ownership poll in (2) and (3). Fix the fixture's
  wiring if that is the cause; if not, say what product behavior changed.
- Remove pair#432's `-skip` exclusion from `Makefile.local`'s
  `test-couch-zellij-live`.
- Triage the couchcore package's run time and the pre-existing
  `TestSpawnComposesProductionPairRegistrationBoundary` failure. File them
  separately if they turn out unrelated.

## Done when

- `TestParkLifecycleLive`, `TestContinuationZellijSeedTransportLive` and
  `TestRecoveryRealHelperAndSessionConformanceLive/checkpoint` pass locally and
  on `couch-zellij-conformance`.
- The pair#432 exclusion is removed, and the job is green on a PR.
- The duplicate-quit product decision is recorded in `## Spec`.

## Plan

- [ ]

## Log

### 2026-10-10

- Filed from pair#324 at ops:0's direction, with evidence from CI run
  38099182884 and local runs of the live tests (`PAIR_LIVE_COUCH=1`).
