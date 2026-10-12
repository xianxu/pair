# Boundary Review — pair#473 (whole-issue close)

| field | value |
|-------|-------|
| issue | 473 — tests: two flaky pair tests (couchtty collaboration timers; couchcore launch-helper post-ack retries) |
| repo | pair |
| issue file | workshop/issues/000473-tests-two-flaky-pair-tests-couchtty-collaboration-timers-couchcore-launch-helper-post-ack-retries.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5803b69ffaa13fe705e12fc8a120a2fb55ec6ee4..98a31e97e991bc5a624e7403f42482ba509c4a80 |
| command | sdlc close --issue 473 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T03:52:56-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Barrier and injected clock look sound so far. Next I'm running the two fixed tests under `-race` to confirm they pass.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Both flaky tests now have correct root-cause fixes, and neither fix is a retry or a longer sleep. **couchtty:** `pointerOff` sets the phase to "off", releases the lock, and only then tears down the notes and marks timers in the same Run-loop step (`console_pointer.go:129-137`). The test checked the phase from its own goroutine, so it could catch the moment between the two. The fix sends a no-op command through the loop and waits for it to complete. **couchcore:** the post-ack cleanup now takes its timeout from an injectable `Couch.after` (default `time.After`). The tests give it a clock that follows the fake child, so the child's reap is never raced against a real 2ms timer. I checked both mechanisms against the code and found them sound. The only gap is evidence: the Done-when asks for "200 runs under `-race` **and** repeated full-suite runs", and the Log's Proof entry records only the first.

**1. Strengths**
- **The barrier is genuine.** `terminalCommands` is buffered (16), but `runTerminalCommand` blocks on `command.done`, which only the Run loop's `select` serves (`console.go:851`). So `settle` returns only after the step in progress has finished. The only other place that serves commands, `drainChunks`, is called outside the pointer-off step (`console.go:905`).
- **The clock seam is small and stays where it belongs.** It defaults to `time.After` when nil (`couch.go:972-975`), and the only consumer is `handleCleanup.attempt` (`couch.go:989,999`). Production behaviour is unchanged.
- **Each fake clock is tied to the event it waits for.** `delayedQuiescenceHandle.after` (`couch_test.go:1105`) gives up at once while the child is alive. Once the child is dead it waits up to 10s for the reap, so a broken cleanup still fails rather than hangs. The test no longer sets `postAckQuiesceTimeout = 2ms`.
- **Both siblings were fixed together.** `TestPostAckCleanupEndsItsHelperWithoutOwningTheSession` had the same flake and gets the same fix. `settle` also covers both `broadcast end` subtests.
- **Each cause was measured.** The Log records that widening each window with a sleep made the tests fail 3/3, and that with the fix they passed 5/5 with that sleep still in place.

**2. Critical:** none.

**3. Important**
- **Done-when "repeated full-suite runs" has no recorded evidence** (issue Log, "Proof" bullet). The clause names two conditions, and only the 200× `-race` runs are recorded, yet Plan item 3 is checked. The couchcore flake was only ever seen inside a full run, so that half matters most. To fix: run `make -k test` several times and log the count and the result. This is the only instance of its kind in this window.

**4. Minor**
- **Inconsistent defaulting:** `sleep` is set to `time.Sleep` in the constructor (`couch.go:197`) and also nil-checked at its use, while `after` is only nil-checked in `newHandleCleanup`. Either pattern works; using one would be cleaner.
- **Unused duration:** `delayedQuiescenceHandle.after` ignores its duration argument, so `postAckQuiesceTimeout` is no longer exercised by these tests. That is acceptable but not stated.

**5. Test coverage notes**
- I ran `go test -race -count=50 -run TestPostAck ./cmd/internal/couchcore/` and it passed.
- I couldn't build couchtty in this checkout: `runtimebundle/embed.go:8` has no generated assets (`pattern assets/runtime/files: no matching files found`). That is a checkout setup problem, not this diff. My view of the couchtty fix rests on reading the code: the loop's `select` and the order of operations in `pointerOff`.
- Other code paths that leave `Couch.after` nil still exercise the production default.

**6. Architecture**
- **ARCH-DRY: pass.** `settle` names a barrier the test file already uses in place (`console_notes_test.go:88,104,130…`). Nothing is duplicated.
- **ARCH-PURE: pass.** The clock is now injected at the IO seam, which shrinks the real-time dependency.
- **ARCH-PURPOSE: pass.** Both reported tests and the sibling with the same flake are fixed at the root. The only part missing is the recorded full-suite evidence (see Important).

**7. Plan revision recommendations**
- Either add the full-suite run results to the Log, or uncheck the "repeated full-suite runs" part of Plan item 3 until they're done.

```findings
findings:
  - id: new
    severity: Important
    family: done-when-clause-unevidenced
    title: |
      Done-when "repeated full-suite runs" has no evidence in the Log
    detail: |
      The Done-when requires 200 -race runs AND repeated full-suite runs. The Log Proof entry records only the 200 -race runs, yet Plan item 3 is checked. The couchcore flake was only ever seen in a full run. Run make -k test several times and log the count and result. This is the only instance in the window.
  - id: new
    severity: Minor
    family: injected-seam-default-idiom
    title: |
      Couch.after defaults at its use while Couch.sleep defaults in the constructor
    detail: |
      sleep is set to time.Sleep in the constructor (couch.go:197) and also nil-checked at use (couch.go:958). after is only nil-checked in newHandleCleanup (couch.go:973). Pick one idiom for injected clocks.
```

---

## Re-review — 2026-10-12T03:54:32-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 473 — tests: two flaky pair tests (couchtty collaboration timers; couchcore launch-helper post-ack retries) |
| repo | pair |
| issue file | workshop/issues/000473-tests-two-flaky-pair-tests-couchtty-collaboration-timers-couchcore-launch-helper-post-ack-retries.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5803b69ffaa13fe705e12fc8a120a2fb55ec6ee4..6ac70d87b1b177ea3849f16b118b8117de9f6e1f |
| command | sdlc close --issue 473 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T03:54:32-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

Both flaky tests now wait on the event that makes their assertion true instead of a real-time window, which is the right fix in both places. In couchtty, `settle` is a valid barrier: `runTerminalCommand` (`cmd/internal/couchtty/terminal.go:29-44`) queues the no-op on the loop that is busy with the current step and returns only after the loop runs it. In couchcore, the injected `Couch.after` makes the cleanup's timeouts follow the fake child's state, so it no longer depends on 2ms of scheduler time. Both prior findings are resolved. BR-1: the Log now has full-suite evidence. BR-2: `after` is set in the constructor and also nil-checked at its use, which matches how `sleep` is handled. I ran the two couchcore tests 20 times under `-race` and they passed. I could not run the couchtty test in this checkout: its setup fails at `runtimebundle/embed.go:8` (`assets/runtime/files: no matching files found`), a generated-asset gap in this checkout that has nothing to do with this diff. One minor wording inconsistency remains in the Log.

1. **Strengths**
   - `cmd/internal/couchtty/console_notes_test.go:176-183`: `settle` uses the product's own command path as the sync point. It needs no new hook and no wait on the clock.
   - `cmd/internal/couchcore/couch_test.go:1101-1112`: the fake's `after` gives up at once while the child lives, and once the child is dead it waits up to 10s for the reap. That pins the intended count of 3 KILLs exactly, and a cleanup that never takes the reap still fails rather than hangs.
   - `cmd/internal/couchcore/couch.go:198`, `:973-976`: `after` follows the same default idiom as `sleep` (`:197`, `:959-961`): set in the constructor, nil-checked at use.
   - The Log records each cause as measured: the window was widened with an injected sleep and the tests failed 3/3, then passed 5/5 with the fix in place.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **Full-suite Log line contradicts itself.** It says "Each Go run also had one unrelated failure", then names failures only for runs 1 and 3 and says "Run 2 was fully green". Fix: change it to "Runs 1 and 3 each had one unrelated failure". This is the only instance of its family in the window.

5. **Test coverage**
   - The fixes change only tests and a test seam. The regression evidence is the widened-window experiments recorded in the Log, which is the appropriate kind of proof for a flake fix.
   - `Couch.after` has no consumers in production: production uses the `time.After` default, so behavior is unchanged.

6. **Architecture**
   - **ARCH-DRY: pass.** `after` reuses the existing injected-clock idiom instead of adding a parallel mechanism.
   - **ARCH-PURE: pass.** The clock is injected at the IO seam (`handleCleanup`), and its business logic is untouched.
   - **ARCH-PURPOSE: pass.** Both flakes are fixed, including the sibling couchcore test that shared the same flake, and the lesson in `workshop/lessons.md` generalizes the rule.

7. **Plan revisions:** none. The plan matches the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Log now records three `go test ./...` runs plus `make -k test` (exit 0), with the four target tests green in every run.
  - id: BR-2
    disposition: addressed
    note: |
      after is set to time.After in New (couch.go:198) and nil-checked at use (couch.go:974), matching sleep (couch.go:197, 959).
findings:
  - id: new
    severity: Minor
    family: doc-claim-unverified-count
    title: |
      Full-suite Log entry says each Go run had an unrelated failure, then says run 2 was fully green
    detail: |
      Issue Log, Full-suite runs bullet: change to "Runs 1 and 3 each had one unrelated failure". Only instance in the window.
```
