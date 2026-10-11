# Boundary Review — pair#444 (whole-issue close)

| field | value |
|-------|-------|
| issue | 444 — session-sync: a concurrent writer sometimes fails with a generic error instead of stale |
| repo | pair |
| issue file | workshop/issues/000444-session-sync-a-concurrent-writer-sometimes-fails-with-a-generic-error-instead-of-stale.md |
| boundary | whole-issue close |
| milestone | — |
| window | 59aca179292392050cbb279f1133d64607e1425a..9ced18fc7ffb0a02498109b26ea1b84f92ea3a80 |
| command | sdlc close --issue 444 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-11T11:10:47-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Fix confirmed: reverting it makes both tests fail. The fallback branch (re-opening the flock file after a failed unlocked check) has no test, so I'm checking whether the rig ever reaches it.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Verdict: fix then ship.** The fix is correct, and the cause is now measured instead of guessed. When the losing writer lost, it was being refused as "tl.md is not a session-sync file". It read the file without the lock while the winner was rewriting it in place, so it saw a half-written file. That refusal exits with code 1, the same code as a generic error, which is why it looked like one. The new `openFlock` (`cmd/internal/sessionsync/sync.go:149`) leaves the decision to the read under the lock. I checked the claimed evidence directly. With the base `sync.go` restored, `TestAWriteJudgesATornFileUnderTheFlock` fails with exactly that refusal, and the race test fails within 200 runs (`exit codes [0 1] … *sessionsync.Refusal: … is not a session-sync file`). With the fix in place, the full package passes, and the race test passes at both 200 and 400 runs.

One thing keeps this from a plain SHIP. The fix has two halves, and only the first is tested. The second half covers the case where no flock file exists yet when the check runs. I deleted that half and every test still passed, including the race test at 400 runs.

**1. Strengths**
- The cause is named precisely in the Log: an unlocked check racing an in-place rewrite, plus code 1 shared between "refused" and "generic error".
- The race helper now prints each writer's error on failure (`race_test.go:27`, `:75`), which satisfies the second part of the first "Done when" clause.
- There are no sleeps in the fix. "The flock file is never removed" holds: nothing in `sessionsync` removes `.Flock`. The only `os.Remove` there is the gap record in `gap.go:72`.
- `TestAWrongFileLeavesNoFlockBehind` still passes, so a wrong path still leaves nothing behind.
- The new test pins the real behaviour. It holds the flock over a truncated file, then checks that the waiting write blocks and is refused as stale (`sync_test.go:173`).

**2. Critical:** none.

**3. Important**
- **The second half of the fix has no test** (`sync.go:155-160`). This is the path where no flock file exists, the check fails, and the code re-opens the flock file without creating it.
  - Every test that writes first calls `r.hash`, and that read goes through `locked`, which already creates the flock file. So no test starts without a flock file and then hits a failed check.
  - Removing lines 157-159 leaves the whole package green, the race test included at 400 runs.
  - In practice this path is a first-ever write to a new file that has no prior read, racing a writer that is mid-rewrite.
  - Fix sketch: make the check a parameter, e.g. `openFlock(path, check func(string) error)`, and add a test whose `check` creates the flock file and then returns a refusal. Assert that `openFlock` returns a handle, not the error.
  - Family, enumerated for this window: there is one untested branch (`sync.go:157`). The other branches are covered: the existing-file early return by the new test, and create-after-valid-check by `newRig`'s first read. The bad-path branch (line 160) is covered by `TestAWrongFileLeavesNoFlockBehind`.

**4. Minor**
- The comment at `sync.go:147` says "a writer rewrites it in place … that writer holds the flock file". That is true for session-sync writers. The operator's editor also writes in place, but it is fenced by `.lock`, not the flock. The claim is correct within this module; the wording just reads as universal.

**5. Test coverage notes**
- The first "Done when" clause has two parts and both are exercised: the 200-run pass, and the error print, which I saw printed when I reverted the fix. The second clause, the cause named in the Log, is met.
- The gap is the one branch described under Important.

**6. Architectural notes**
- **ARCH-DRY: pass.** `openFlock` reuses `load` and `PathsOf`, and nothing in the window duplicates it.
- **ARCH-PURE: pass.** The change is filesystem glue by nature and stays small. Turning the check into a parameter (the fix sketch above) would also make the branch testable without races.
- **ARCH-PURPOSE: pass.** All four entry points (`Read`, `Status`, the third `locked` caller at `sync.go:295`, `Write`) go through `locked` and so through `openFlock`. Every concurrent path now either waits on the flock or is judged under it.
- No atlas or README change is needed: there is no new user-facing surface.

**7. Plan revision recommendations:** none. The plan's single checkbox matches what was delivered.

```findings
findings:
  - id: new
    severity: Important
    family: fix-branch-without-failing-test
    title: |
      openFlock's fallback (no flock file, check fails, re-open without create) has no test; deleting it leaves all tests green
    detail: |
      sync.go:155-160. Every test reads first via r.hash, and that read creates the flock file through locked, so no test starts without a flock file and then fails the check. Removing lines 157-159 keeps the package green, including the race test at 400 runs. Fix: make the check a parameter of openFlock and test a check that creates the flock file and then refuses; assert a handle is returned. Family members in this window: only sync.go:157. The other openFlock branches are covered by the new torn-file test, newRig's first read, and TestAWrongFileLeavesNoFlockBehind.
  - id: new
    severity: Minor
    family: comment-claim-scope
    title: |
      The openFlock comment says the in-place rewriter holds the flock, but that is true only of session-sync writers
    detail: |
      sync.go:147. The operator's editor also rewrites in place but is fenced by .lock, not the flock. Narrow the wording to session-sync writers.
```

---

## Re-review — 2026-10-11T11:12:37-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 444 — session-sync: a concurrent writer sometimes fails with a generic error instead of stale |
| repo | pair |
| issue file | workshop/issues/000444-session-sync-a-concurrent-writer-sometimes-fails-with-a-generic-error-instead-of-stale.md |
| boundary | whole-issue close |
| milestone | — |
| window | 59aca179292392050cbb279f1133d64607e1425a..fcf20d5bca09db2c82a83e790518b2629f1cf7a2 |
| command | sdlc close --issue 444 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-11T11:12:37-07:00 |
| verdict | SHIP |

## Review

With the fallback removed, the regression test fails as it should. Now I'm running the 200-run race on the restored code.

```verdict
verdict: SHIP
confidence: high
```

Both prior findings are fixed, and I found nothing new. The cause the Log names holds up against the code: `locked` used to check the file without holding the lock, so it could catch a winning writer's in-place rewrite half-done and refuse it as "not a session-sync file". That refusal exits 1, which is why it looked like a generic error. `openFlock` (`sync.go:144-163`) now lets the read under the lock judge the file whenever a flock file exists. When it doesn't, it checks first, so a wrong path still leaves nothing behind. Checks I ran in the disposable checkout:
- The full package passes and `go vet` is clean.
- `TestTwoProcessWritersExactlyOneWins -count=200` passes 200 of 200.
- With the fallback at `sync.go:155-157` removed, `TestAFailedCheckDefersToAFlockCreatedMeanwhile` fails ("…must defer to the lock: torn"). Restoring the fallback makes it pass again.

My first revert attempt also overwrote `sync.go` with a stale `/tmp` copy. I restored it with `git checkout`, and `git status` is clean.

Both clauses of `## Done when` are met: the race test passes 200 runs and prints each writer's error when it fails, and the Log names the cause.

1. **Strengths**
   - `openFlock` takes the check as a parameter (`sync.go:149`). That makes the window between probing for the flock file and checking the file testable directly, without timing tricks (`sync_test.go:200-212`).
   - `TestAWriteJudgesATornFileUnderTheFlock` (`sync_test.go:173-197`) reproduces the actual cause: it holds the flock over a truncated file and asserts `stale` comes back after the lock is released.
   - The race helper now prints each writer's error type and value (`race_test.go:27-29`), and a failure includes both writers' output (`race_test.go:75`). This meets the Done-when clause about printing the loser's error.
   - There is no sleep-based fix. The 100ms in the test only shows that the write waits; it is not used for synchronisation.
   - The argument for correctness is sound: the flock file is never removed, and every couch writer holds it while rewriting.

2. **Critical:** none.

3. **Important:** none.

4. **Minor:** none new.
   - The Log's sentence "a torn read implies a writer holding the flock file" is unqualified. Parley edits are fenced by `.lock`, not the flock. This is in the issue file, which is outside the review window, so I'm not raising it.

5. **Test coverage**
   - Every branch of `openFlock` is covered:
     - flock file exists: `newRig`'s first read, then later writes
     - check passes and the flock file is created: first write in a fresh rig
     - check fails and a flock file appeared meanwhile: the new test
     - check fails with no flock file: the existing wrong-file test
   - I confirmed `newRig` (`sync_test.go:24-36`) creates no flock file, so the new test does reach the fallback.

6. **Architecture**
   - **ARCH-DRY:** pass. The new code doesn't repeat anything already in the tree.
   - **ARCH-PURE:** pass. The opening logic is a small IO helper, and the check is injected, which is what makes the race window testable.
   - **ARCH-PURPOSE:** pass. The fix is in `locked`, which every flock-taking command goes through, so all of them are covered.

7. **Plan revisions:** none needed.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      TestAFailedCheckDefersToAFlockCreatedMeanwhile (sync_test.go:200) goes red with the fallback (sync.go:155-157) removed and green with it restored; check is now an openFlock parameter.
  - id: BR-2
    disposition: addressed
    note: |
      sync.go:146-148 now scopes the claim to "a couch writer", matching locked's "couch's exclusive flock" (sync.go:124-126), which already says parley is fenced by .lock instead.
```
