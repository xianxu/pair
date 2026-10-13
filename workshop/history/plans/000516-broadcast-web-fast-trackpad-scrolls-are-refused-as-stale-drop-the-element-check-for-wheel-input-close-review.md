# Boundary Review — pair#516 (whole-issue close)

| field | value |
|-------|-------|
| issue | 516 — broadcast-web: fast trackpad scrolls are refused as stale; drop the element check for wheel input |
| repo | pair |
| issue file | workshop/issues/000516-broadcast-web-fast-trackpad-scrolls-are-refused-as-stale-drop-the-element-check-for-wheel-input.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0d5bab17ecd8252a684a2cac07fb123ccc16de98..4ed215e54eeb3477c446bbf8b6604be349905016 |
| command | sdlc close --issue 516 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T19:24:51-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Diff and docs read; now running the drive package tests and a mutation check of the wheel rule.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The fix is small and correct. In `TabInjector.inject` (`cmd/internal/webcast/drive/tab.go:167-178`), a wheel now skips the `onTarget` identity check, and a wheel with nothing under its point is still refused (`hit == 0` → ErrStale). Downs and clicks are unchanged. The new contract cases (`contract.go:130-137`) run against every injector. I ran a mutation check: with the old rule put back, `TestFakeInjectorContract` fails with "a wheel over another element than the viewer saw: got drive: the page changed, want it injected". The drive and recorder packages pass at head and `go vet` is clean. The live flick test (`TestDriveRealCastFlick`) is gated on `WEBCAST_LIVE_CHROME`, so I did not run it; its numbers come from the Log. Two things stand between this and a clean SHIP. First, two doc comments still state the old rule for wheels. Second, the Done-when promised a flick test that "fails on main", but the deterministic red test is the contract test instead; the plan explains this, but no `## Revisions` entry records it.

## 1. Strengths
- `tab.go:172-175`: the wheel exception is narrow. It changes only the stale decision, keeps the bounds check, and leaves the rest of the loop (Confine, dispatch, `note`) alike for every act.
- `contract.go:130-137`: both halves of the new rule are covered: a wheel at a different element goes in, and a wheel with nothing under it gets ErrStale. Because they sit in `RunInjectorContract`, the fake and real injectors share them, and the mutation check confirms they catch the old rule.
- The existing click-stale case (`contract.go:100`) still guards the Done-when clause "clicks after a scroll still refuse a stale target".
- `startRealCast(t, path)` pulls the live harness out so `TestDriveRealCast` and the flick test share it instead of copying it (ARCH-DRY).
- The coalescing decision follows the data. The Log records arrival spacing, injection time, how long injection ran past the gesture, and the per-batch dy curve, which is what the operator's Revisions asked for.

## 2. Critical findings
None.

## 3. Important findings
- **Doc comments still say a wheel is refused when the element differs** (`doc-claims-track-rule-change`). The window updated `Injector.Inject`'s doc and the atlas, but two sibling statements of the same rule were missed:
  - `cmd/internal/webcast/drive/act.go:54-56`, the `Act` doc: "ID is the element ... for down, click and wheel: the injector refuses the act if the tab has another element there now (ErrStale)". This is now false for wheel.
  - `cmd/internal/webcast/drive/injector.go:40-41`, the `ErrStale` doc: "the element the viewer aimed at isn't at that point". It should also cover "or a wheel's point has nothing under it".
  - The atlas `webcast-format.md:566` ("`down`, `click` and `wheel` need it") is still true, since the ID is still required. I checked it and it is fine.

  Fix: reword both comments to match `injector.go:28-31`.

## 4. Minor findings
- **`TestDriveRealCast` lost its doc comment** (`live_drive_real_test.go:~90-96`). Inserting `type realCast` moved the test's doc comment, including its `go test -run DriveRealCast` line, onto `realCast`. Move it back above `func TestDriveRealCast`.
- **Dead binding:** `live_drive_real_test.go:212` has `c, h, v, spy := ...` followed by `_ = c`. Drop `c` from the destructure instead.
- **Messy locking in `errSpy.Inject`:** it locks, appends and unlocks `s.mu`, then immediately locks it again with a deferred unlock. Use one critical section. The later loop variable `a` also shadows the `arrival` called `a`.
- **Flick test can miss 503s:** its `busy` counter only sees context errors that reach the injector. A 503 from the board's queue (`server_board.go:210`) never gets that far. The `injected >= 0.95*sent` assertion still catches lost scroll, so the claim that 503s are covered rests on that assertion.
- **`hit == 0` means two things** (`recorder/drive.go:179`). `IDOf` returns 0 for a node the mirror hasn't recorded yet as well as for an empty point, so a wheel over a node the recording doesn't have yet is refused as "nothing under it". The old rule refused it too, so this is not a regression, but the "bounds check" comment overstates what the check does.

## 5. Test coverage notes
- The contract test is the deterministic regression test and it fails without the fix (verified by mutation above).
- **Done-when clause 1 is only partly met as worded.** It asks for a flick test that "fails on main". The Log says the live flick reproduced the refusal in only 1 of 2 runs before the fix, and 0 of 3 with 60 px rows. So the flick test does not reliably fail on main; the contract test fills that role. The plan states this, but the Done-when was never revised.
- Clause 3 is met: the measurement and the coalescing decision are in the Log.

## 6. Architectural notes
- **ARCH-DRY: pass.** The live harness is shared through `startRealCast`, and the wheel rule is stated once in `TabInjector`, which the fake reuses through the `Tab` seam.
- **ARCH-PURE: pass.** The rule lives in pure `inject`/`onTarget` behind the `Tab` seam, and the contract tests run against the fake tab with no IO.
- **ARCH-PURPOSE: pass, with one sweep to finish.** The behavioural purpose is delivered at the single rule site, and coalescing was measured rather than skipped. The comment sweep for the rule change is incomplete (the Important finding above).
- If timestamped 16 ms buckets are built later, the `batcher` merge note in the Log is the starting point.

## 7. Plan revision recommendations
- Add a `## Revisions` entry: "Done-when 1's 'the test fails on main' is met by the `RunInjectorContract` wheel cases (deterministic, mutation-checked). The live flick reproduces the refusal only intermittently before the fix (1 of 2 runs), so it measures and guards the change but is not the red test."

```findings
findings:
  - id: new
    severity: Important
    family: doc-claims-track-rule-change
    title: |
      Act and ErrStale doc comments still state the old wheel stale rule
    detail: |
      The window updated Injector.Inject's doc and the atlas but not the sibling statements of the same rule. act.go:54-56 says "for down, click and wheel: the injector refuses the act if the tab has another element there now (ErrStale)", which is now false for wheel. injector.go:40-41 defines ErrStale only as "the element the viewer aimed at isn't at that point" and omits the wheel bounds case. Reword both to match injector.go:28-31.
  - id: new
    severity: Minor
    family: done-when-matches-delivered-evidence
    title: |
      Done-when says the flick test fails on main; the Log shows it reproduces only intermittently
    detail: |
      The deterministic red test is the RunInjectorContract wheel case (mutation-verified). Add a Revisions entry so Done-when 1 names the evidence that was actually delivered.
  - id: new
    severity: Minor
    family: doc-comment-follows-its-decl
    title: |
      Hoisting type realCast moved TestDriveRealCast's doc comment off the test
    detail: |
      live_drive_real_test.go around lines 90-96: the test's doc and its go test -run line now sit above type realCast. Also drop the unused c binding and the _ = c at line 212.
  - id: new
    severity: Minor
    family: test-spy-hygiene
    title: |
      errSpy.Inject locks the mutex twice in a row and shadows its arrival variable a
    detail: |
      Merge the two critical sections into one. The busy counter cannot see 503s returned by the board queue before the injector; the px-injected assertion is what actually covers lost scroll.
```

---

## Re-review — 2026-10-12T19:35:01-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 516 — broadcast-web: fast trackpad scrolls are refused as stale; drop the element check for wheel input |
| repo | pair |
| issue file | workshop/issues/000516-broadcast-web-fast-trackpad-scrolls-are-refused-as-stale-drop-the-element-check-for-wheel-input.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0d5bab17ecd8252a684a2cac07fb123ccc16de98..85a41d94315c0759eeec1f71a2f398adce87ef06 |
| command | sdlc close --issue 516 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T19:35:01-07:00 |
| verdict | SHIP |

## Review

Mutation check confirms the contract goes red. One more mutation: dropping the wheel bounds check.

```verdict
verdict: SHIP
confidence: high
```

All four findings from the last round are fixed. The wheel rule in `tab.go:167-178` matches the Spec: a wheel skips the identity check and keeps the bounds check (`hit == 0` → ErrStale), while downs and clicks are unchanged. I checked both halves by breaking them. Removing the wheel exemption makes the contract fail with "a wheel over another element … want it injected". Removing the bounds check makes it fail with "a wheel with nothing under its point … want ErrStale". The rule is now written out fully in one place, `Injector.Inject` (`injector.go:24-33`). `Act`, `ErrStale` and `onTarget` point back to it instead of repeating it, and the atlas states the same rule. All `./cmd/internal/webcast/...` tests pass. Nothing blocks shipping. Two small cleanups are left, listed below.

Done-when: current. The new Revisions entry says what proves clause 1 (the contract's wheel case, which fails on main every time). Clause 2 is covered by the click-stale case at `contract.go:99-105`, which passes. Clause 3, the measurement and the coalescing decision, is in the Log. The operator's two coalescing Revisions only applied if the measurement showed a need, and it showed none, so Done-when doesn't have to name them.

1. **Strengths**
   - The wheel rule lives in the shared `RunInjectorContract` (`contract.go:130-137`), so every injector is held to it, including the fake. I confirmed both mutations above turn it red.
   - The other doc sites now name `Injector.Inject` as the rule's owner rather than restating it (`act.go:55-56`, `injector.go:40-41`, `tab.go:188`). That fixes the whole class, not just the one site, and `workshop/lessons.md` records the habit.
   - The flick test fires its events inside the viewer page, so the timing is realistic. It asserts that every pixel sent is injected, not just that nothing was refused.
   - `startRealCast` is shared by both live tests instead of being a copy-paste of the setup.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `live_drive_real_test.go:32-40`: the `realCast` fields `c`, `ctrl` and `mem` are set but nothing reads them. They are what's left of the earlier unused `c` binding. Drop them or use them.
   - `tab.go:188`: the reworded `onTarget` comment line runs past the column the lines around it wrap at.

5. **Test coverage**
   - The contract covers the wheel in both states (another element under the point → injected; nothing under it → ErrStale) and the click-stale case.
   - The live flick test needs Chrome, so the normal test run skips it, and its pass is backed only by the Log. That is fine, because the contract is the test that always runs.
   - `busy` in the flick test only counts context errors at the injector. A 503 from the board queue never reaches the spy. The `injected >= 0.95*sent` assertion is what actually catches lost scroll.

6. **Architecture**
   - **ARCH-DRY: pass.** The rule is written out once and the other sites point to it. The live setup is shared.
   - **ARCH-PURE: pass.** The decision is a branch on values already computed (`hit`, `a.A`) inside the existing `TabInjector`, and it is tested through the fake with no IO.
   - **ARCH-PURPOSE: pass.** Wheels are no longer refused as stale. The measurement and coalescing decision are recorded. Coalescing was dropped because the measurement showed no need, as the Spec allows, not put off as follow-up work.

7. **Plan revisions:** none. The plan matches the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      act.go:55-56 and injector.go:40-41 now defer to Injector.Inject; onTarget (tab.go:188) scoped to down/click; atlas matches.
  - id: BR-2
    disposition: addressed
    note: |
      Revisions entry (close review) names the contract wheel case as the red-on-main evidence; mutation-verified red this round.
  - id: BR-3
    disposition: addressed
    note: |
      TestDriveRealCast doc and go test -run line sit directly above the func again; the c binding and `_ = c` are gone.
  - id: BR-4
    disposition: addressed
    note: |
      errSpy.Inject takes the lock once and the loop variable is renamed x; the px-injected assertion is the lost-scroll check.
findings:
  - id: new
    severity: Minor
    family: test-spy-hygiene
    title: |
      realCast fields c, ctrl, mem are set but never read
    detail: |
      This is the 2nd finding in family test-spy-hygiene. The rule: every field and binding a test helper sets must have a reader. In this window the only instances are realCast.c, .ctrl and .mem (live_drive_real_test.go:32-40); spy, h and v are read. Drop the three fields or use them.
```
