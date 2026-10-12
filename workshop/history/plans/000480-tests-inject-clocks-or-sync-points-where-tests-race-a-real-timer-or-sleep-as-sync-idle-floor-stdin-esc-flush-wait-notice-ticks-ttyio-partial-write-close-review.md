# Boundary Review — pair#480 (whole-issue close)

| field | value |
|-------|-------|
| issue | 480 — tests: inject clocks or sync points where tests race a real timer or sleep as sync (idle floor, stdin ESC flush, wait-notice ticks, ttyio partial write, ...) |
| repo | pair |
| issue file | workshop/issues/000480-tests-inject-clocks-or-sync-points-where-tests-race-a-real-timer-or-sleep-as-sync-idle-floor-stdin-esc-flush-wait-notice-ticks-ttyio-partial-write.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5bd8db0d5db11f07e55a654168e5fedb8839ccbc..1fc4a3fbff3cf5e41eeee0c0e94a90748b3a4452 |
| command | sdlc close --issue 480 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T06:51:00-07:00 |
| verdict | SHIP |

## Review

Production seams look sound so far; now checking the remaining test diffs (ttyio, couchcmd, broadcast, hostty, launcher).

Admission path is synchronous, so the `tickAt` claim holds. Running the touched tests once to confirm they compile and pass.

All touched tests pass under `-race`. Running a quick mutant check on the idle-floor bump to confirm the new test catches it.

```verdict
verdict: SHIP
confidence: high
```

All 13 flaky timing sites named in pair#480 are fixed, plus the orientation-deadline test added during the work. Each test now waits on the event it actually means, an injected clock, or a counter, instead of wall time. I checked the three product seams line by line (`loopTimer` for the idle floor and the held-ESC flush, `Couch.registrationDeadline`, `LaunchOptions.BirthClock`). Each defaults to the wall clock when nil, so production behaviour does not change. The touched tests pass under `-race`: ttyio, hostty, couchmessage, couchcore, launcher, broadcast, and wrapcmd/couchcmd after generating the runtime bundle. To check the new idle-floor test can fail, I removed `p.bumpIdleDeadline()` from `wrap.go` and it went red (FAIL); the file is restored. I did not repeat the 200-run soak; that clause rests on the Log. Nothing blocks shipping. The two Minor findings are a small leftover wall-time margin in two ttyio assertions and one inaccurate comment.

**1. Strengths**
- `wrap.go:1555-1572` now calls `drainStop`, the shared helper, instead of two copies of stop-and-drain code, and `stoppedWallTimer` replaces two hand-written "start stopped" blocks (ARCH-DRY gain).
- `wait_notice_test.go` `tickAt`: I checked the claim that a notice is counted before the wait starts. It holds: `broker.Send` → `tryAdmission` calls `endpoint.Reserve` synchronously (`broker.go:554`), and `tick` calls `Send` directly (`wait_notice.go:349`). So "no notice" assertions are still exact.
- `hostty_test.go`: `Close` joins the watcher and closes `resized` (`os.go:64-75,138-154`), so `for range h.Resized()` gives the final count. A watcher stuck on a blocking send now makes the test hang rather than flake.
- `spendableBudget` is reused by both the registration-phase and orientation-deadline tests rather than written twice.
- `never_turned_test.go` asserts directly that the registration budget was never taken, which is what "refuses at once" means, not a 200 ms wall-clock proxy for it.

**2. Critical findings:** none.

**3. Important findings:** none.

**4. Minor findings**
- **`ttyio/writer_test.go:76`**: the busy-poll limit `2*(time.Since(progressed)/pollSlice)+10` only counts time after `progressed`. But the reader goroutine has been polling since it started, before the readability wait. Under heavy load that earlier window could use up the +10 slack. Fix: take the timestamp before starting the reader goroutine.
- **`ttyio/writer_test.go:285`** (same family): `before` is read before `cancel()`. If more than one poll slice (50 ms) passes between that read and the cancel, the `after > before+1` check fails even though the code is fine.
- **`idle_floor_test.go:205-207`**: the comment says a chunk reaching stdout means the bump is done. It isn't quite: `handleChunk` hands the chunk to the asynchronous stdout pump, then `bumpIdleDeadline` runs (`wrap.go:3263-3266`), so the pump can write first. The test is still correct, because the previous chunk's bump is always complete and a late bump only pushes the deadline later. Only the comment should be fixed.
- **`idle_floor_test.go:144`** (ARCH-DRY, note only): the new unbounded `await` sits next to the existing bounded `waitFor`. Keeping both is deliberate since they behave differently; a one-line comment explaining why would help.

**5. Test coverage notes**
- Every listed test was replaced as claimed. The remaining `settle(50ms)` sleeps only prove that nothing happened: a slow machine can make them pass falsely but never fail falsely, which their comments say.
- The idle-floor mutant (no bump → FAIL) confirms the output test catches the bug it is named for.

**6. Architectural notes**
- **ARCH-DRY: pass.** The repeated stop-and-drain code was consolidated, and the budget helper is shared. Each package keeps its own one-line polling loop, which is fine.
- **ARCH-PURE: pass.** Every seam is an injected function or interface on the owner struct, nil means the wall clock, and nothing in the business logic reaches for the clock directly.
- **ARCH-PURPOSE: pass.** Every site the issue lists, plus the orientation test, now waits on an event or an injected clock. Nothing was deferred, and #482 was correctly split out as a separate flake.
- **Docs gate:** no user-facing surface was added. The seams are internal test hooks, so atlas and README don't need changes; `lessons.md` records the pattern.

**7. Plan revision recommendations:** none; the plan and Log match the code.

```findings
findings:
  - id: new
    severity: Minor
    family: residual-wall-time-window
    title: |
      ttyio poll-count assertions still have a small wall-clock margin
    detail: |
      writer_test.go:76 sizes the busy-poll limit from `progressed`, but the reader goroutine's polls started earlier. writer_test.go:285 reads `before` before cancel(), so a gap longer than one pollSlice gives after > before+1. Both instances are in this window. Fix: take the start time before launching the reader goroutine, and read `before` and call cancel back to back (or allow before+2).
  - id: new
    severity: Minor
    family: comment-claim-accuracy
    title: |
      idle-floor comment says a chunk on stdout means the bump already ran
    detail: |
      idle_floor_test.go:205-207. handleChunk queues to the async stdout pump, and bumpIdleDeadline runs after it (wrap.go:3263-3266), so the pump can write first. The test is still sound because the previous chunk's bump has finished; only the comment needs correcting.
```

---

## Re-review — 2026-10-12T06:53:59-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 480 — tests: inject clocks or sync points where tests race a real timer or sleep as sync (idle floor, stdin ESC flush, wait-notice ticks, ttyio partial write, ...) |
| repo | pair |
| issue file | workshop/issues/000480-tests-inject-clocks-or-sync-points-where-tests-race-a-real-timer-or-sleep-as-sync-idle-floor-stdin-esc-flush-wait-notice-ticks-ttyio-partial-write.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0a79be7647906c7271f6a6cee81a13e48f037ef4..8e94173bce9deb53ae2ff5e0548ce5640c968a50 |
| command | sdlc close --issue 480 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T06:53:59-07:00 |
| verdict | SHIP |

## Review

Both prior findings look fixed. I'm running one last stress check on the tighter ttyio limit, then I'll give the verdict.

```verdict
verdict: SHIP
confidence: high
```

Both prior findings are fixed, and the fixes add no Critical or Important defect. The window changes two test files and adds two generated plan files, the gate ledger and the review record. It does not touch `## Spec`, `## Revisions` or `## Done when`, and there were no conflict resolutions to judge.

**BR-1, the wall-clock margin:**
- **Busy-poll test** (`writer_test.go:44`): the start time `started` is now taken before either goroutine can poll, so the limit covers every wait counted.
- **Cancel test** (`writer_test.go:282-285`): it now calls `cancel()` before reading `before`. That makes the `after > before+1` check exact:
  - `wait()` checks `stopped(ctx)` before `polls.Add(1)` (`writer.go:116-129`).
  - So after a cancel, at most one wait can still be counted: the one whose check passed just before the cancel.

**Tighter slack:** the fix also lowered the busy-poll slack from +10 to +4. Each goroutine's polls add up to at most `floor(T/pollSlice)+1`, because the last poll is partial or ends early. So the expected worst case is `2·floor(T/50ms)+2`, which leaves 2 spare. A run of 20 repetitions of both tests under `-race` at `-cpu=1,4` passed.

**BR-2, the comment:** the new comment at `idle_floor_test.go:205-208` matches `wrap.go:3262-3265`. `handleChunk` runs, then `bumpIdleDeadline` runs in the same loop step, so the bump lands before the loop reads the next chunk.

1. **Strengths**
   - `writer_test.go:282-285`: the cancel-then-count ordering turns a timing margin into an exact bound tied to how `wait()` counts.
   - `writer_test.go:44`: the start time now comes before any poll, so the limit no longer leaves out the reader's earlier waits.
   - The idle-floor comment now states the real order: the read and the bump happen in the same loop step.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings** (notes only, nothing raised)
   - The clause "the clock advances at most one 15ms step past any bump" in `idle_floor_test.go:208` can be read two ways.
     - It is true that at the moment chunk i's deadline is bumped, the clock is at most one 15 ms step past where it was when that chunk was seen.
     - Between two consecutive bumps, though, the clock can move up to 30 ms.
     - The test is sound under either reading, since 30 ms is less than the 60 ms floor. I'm not raising it as a separate `comment-claim-accuracy` finding.
   - The slack change from +10 to +4 went beyond what BR-1 asked for. It is still a test-only tightening and leaves 2 spare above the worst case worked out above.

5. **Test coverage notes**
   - Both ttyio tests passed under `-race` at counts 5 and 20, with `-cpu=1,4`.
   - I couldn't build `wrapcmd` in this checkout: `go vet` fails because the generated runtime bundle (`assets/runtime/files`) isn't there. Only a comment changed in that package, so its behaviour is unaffected.

6. **Architectural notes**
   - **ARCH-DRY: pass.** The change is two test edits with no duplication.
   - **ARCH-PURE: pass.** No production code changed.
   - **ARCH-PURPOSE: pass.** Both instances of `residual-wall-time-window` named in BR-1 were fixed in the same round. The finding listed no other instances in this window.

7. **Plan revision recommendations:** none.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      started now precedes both goroutines (writer_test.go:44); cancel() precedes reading before (writer_test.go:284-285), exact given stopped() runs before polls.Add; 20x -race at -cpu=1,4 passes
  - id: BR-2
    disposition: addressed
    note: |
      idle_floor_test.go:205-208 now says the bump follows the read in the same loop step, before the next chunk is read, matching wrap.go:3262-3265
```
