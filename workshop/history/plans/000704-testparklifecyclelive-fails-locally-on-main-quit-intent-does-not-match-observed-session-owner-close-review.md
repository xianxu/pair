# Boundary Review — pair#704 (whole-issue close)

| field | value |
|-------|-------|
| issue | 704 — TestParkLifecycleLive fails locally on main: quit intent does not match observed session owner |
| repo | pair |
| issue file | workshop/issues/000704-testparklifecyclelive-fails-locally-on-main-quit-intent-does-not-match-observed-session-owner.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5b2d51a4a05d1bb6ba5388f7dd7636af462b2927..647a52a5211b08d8e33e2bb2b25cb00b46321b20 |
| command | sdlc close --issue 704 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-19T09:11:49-07:00 |
| verdict | SHIP |

## Review

Diff read; now running the tests and reverting each fix to confirm its regression test goes red.

```verdict
verdict: SHIP
confidence: high
```

This change fixes a live-test failure where a second trigger delivery found the session already gone, and the error called that an owner mismatch. It does so without changing production behaviour. Both test drivers now follow the rule `park.go` uses in production (around `park.go:816`): trigger only when the session is observed present. The fake driver now also removes the session on trigger, as the real trigger does. `TriggerBoundQuit` now reports an absent session as absent rather than as a mismatch. I ran the tests and reverted each fix in the disposable checkout. Both regression tests fail without their fix and pass with it. Nothing blocks shipping. The only notes are Minor: the conformance trace and some test coverage now say less than they used to.

Done-when: current — the Revisions entry moves where the fake's session removal lives (into the test driver, not the shared Fake). It does not change what the work must deliver. Clause 1 is met by the Plan's diagnosis: the owner observation is `State: Absent` with an empty owner, so this is a vanished session, not a wrong field. Clause 2 is met by the fake conformance regression, which fails deterministically without the fix. The passing local live runs (5/5) are claimed in the Log. CI passing can't be checked from here; this is noted, not raised as a finding.

1. **Strengths**
   - The fix matches the diagnosis. Production `park.go:816` already triggers only when `binding.Present`, so the test drivers were the ones out of step. The production change is limited to a clearer error message.
   - The deterministic red is real. With the presence gate removed from `fakeConformanceDriver.DeliverTrigger`, `TestQuitLifecycleConformanceScenarioFake` fails with `exact session "pair-work" is absent`, the same symptom as the live failure, with no timing involved.
   - `TestTriggerBoundQuitNamesAnAbsentSession` (`session_owner_test.go:198`) fails when the absent-session check at `artifactcollision.go:628` is removed. It also checks that nothing is written to the data directory.
   - The session removal was rightly kept out of the shared `Fake`. The Log and Revisions explain why: `TestParkInterruptionBoundaryMatrix` pairs the Fake with a separate `Artifacts` view, and the two would disagree.
   - The live driver (`park_lifecycle_live_test.go:151`) now observes the session through the production `NamedPairSessionContext` and triggers with `TriggerBoundQuit` on that binding. This is closer to production than the old unconditional `TriggerQuit`.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - **Trace records a delivery that didn't happen.** `RunConformanceScenario` (`fake.go:383-386`) appends `trigger:delivered` after every `DeliverTrigger` call. That now includes a redelivery that did nothing, so the trace no longer shows whether the second delivery reached the session. This applies to both drivers (`fake.go:322`, `park_lifecycle_live_test.go:151`).
   - **Redelivery to a still-present session is no longer covered by the scenario.** In the fake driver, the second delivery is now always a no-op, so the scenario never reaches the Fake's handling of a repeat trigger. That case is still covered only by `fake_test.go:32-35`, and in live runs only when timing leaves the session up (which is what happens on CI).
   - **The presence rule is written out three times (ARCH-DRY).** It lives in `park.go:816`, `fake.go:323` and `park_lifecycle_live_test.go:152`. Each driver talks to a different backend, so sharing code is hard. The comments do point back to `park.go`; this is noted only.
   - **The new absent-session check depends on `Owner` being set.** It fires only when `binding.Owner != nil`. A binding with `Present == false` and `Owner == nil` would still be reported as "does not match". Production never triggers an absent binding, so this is harmless.

5. **Test coverage notes**
   - Commands run: `go test ./cmd/internal/pairlifecycletest/` and `go test ./cmd/internal/couchcore -run 'TriggerBoundQuit|Conformance|Park|SessionOwner'`. Both pass.
   - With the fake gate reverted, the fake scenario fails. With the production absent check reverted, the new unit test fails.
   - The live driver's gate only runs under `PAIR_LIVE_COUCH`. The fake driver's identical gate is its deterministic stand-in.

6. **Architectural notes**
   - ARCH-DRY: pass, with the Minor note about the presence rule written out three times.
   - ARCH-PURE: pass. The test drivers are adapters, and the production change is a pure guard.
   - ARCH-PURPOSE: pass. The diagnosis and the fix cover the actual cause, not just the symptom. A possible follow-up is to make the scenario record no-op deliveries separately in its trace.
   - ARCH-EVOLVE: pass. No saved file or record format changes.
   - Docs gate: no new user-facing surface, so no atlas or README update is needed.

7. **Plan revision recommendations:** none. The existing Revisions entry already matches the code.

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: trace-claims-unperformed-effect
    title: |
      The scenario records trigger:delivered even when the gated driver delivered nothing
    detail: |
      RunConformanceScenario (cmd/internal/pairlifecycletest/fake.go:383-386) adds trigger:delivered after every DeliverTrigger call, so a no-op redelivery looks the same as a real one. This applies to both drivers (fake.go:322, park_lifecycle_live_test.go:151). The scenario also no longer exercises redelivery to a still-present session, which is now covered only by fake_test.go:32-35. Consider recording a separate label such as trigger:absent for the no-op case.
  - id: new
    severity: Minor
    kind: defect
    family: absent-binding-diagnostic-coverage
    title: |
      The absent-session check in TriggerBoundQuit only fires when binding.Owner is set
    detail: |
      At artifactcollision.go:628, a binding with Present false and a nil Owner still gets the "does not match observed session owner" message. Production never triggers an absent binding (park.go:816), so this is harmless. Keying the check on !binding.Present would cover both cases.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"6cfb9925-a57d-4278-934c-ecaf0b8c4ccd","repository":"github.com/xianxu/pair","issue_id":"000704","recipe":"small-diff-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/f38c2ea56e8fd69d","reviewed_head":"647a52a5211b08d8e33e2bb2b25cb00b46321b20","slot":"pair:2","requested_at":"2026-10-19T16:08:39.595689Z"},"phase":"finished","pid":48776,"started_at":"2026-10-19T16:08:40.060592Z","finished_at":"2026-10-19T16:11:48.396334Z","session_id":"33df79c2-8f81-4eb3-a412-31a3d5f2c6ee","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 6.982878042s
head: "647a52a5211b08d8e33e2bb2b25cb00b46321b20"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
