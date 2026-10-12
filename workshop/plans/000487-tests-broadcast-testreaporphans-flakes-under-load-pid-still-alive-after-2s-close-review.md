# Boundary Review — pair#487 (whole-issue close)

| field | value |
|-------|-------|
| issue | 487 — tests: broadcast TestReapOrphans flakes under load (pid still alive after 2s) |
| repo | pair |
| issue file | workshop/issues/000487-tests-broadcast-testreaporphans-flakes-under-load-pid-still-alive-after-2s.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0c26f966973382e8ee7b2af30eedb47be50379ab..ac8c9a6b92cd53266f19bf8d9b908b4df14be690 |
| command | sdlc close --issue 487 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T13:28:52-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Diff and issue reviewed; I'm checking the adjacent wall-clock windows and running the tests before giving the verdict.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The diagnosis holds up against the code. `killIfOurs` (`cmd/internal/broadcast/records.go:220`) checks the live command for `cloudflared`, so a fixture running `exec sleep 300` turns into `sleep 300` and is correctly spared. Under load the reaper therefore depended on beating the exec. Keeping the shell (no `exec`) and waiting for its sleep child fixes the fixture without touching the product. All five `waitDead` callers in the package were moved off fixed windows. The targeted guard and close tests passed 5/5 under `-race` here. **`TestReapOrphans` could not run in this review:** `ps` is blocked even outside the sandbox (`operation not permitted: ps`), so `needPS` skips it and my mutation check (putting `exec` back) could not run. That makes the reasoning sound but unverified, hence medium confidence. What stands between this and SHIP is cheap: the diff writes its own deadline-bounded polling twice when the tree already has a helper for exactly this (`cmd/internal/testwait`, from #479).

### 1. Strengths
- **Root cause in the fixture, not a patch on the timeout.** The Log explains why the product is right, and shows that the "live owner", "recycled pid" and "no owner identity" subtests could pass for the wrong reason under the old fixture. The new fixture closes that too.
- **The settled-state check is real.** `cloudflared_test.go:286-294` waits for the sleep child, then asserts the command still contains `cloudflared`. If someone restores `exec`, the fixture fails loudly instead of flaking.
- **Product code left alone.** `records.go` is untouched and correctly so; the identity check is doing its job.
- **Every `waitDead` caller was swept:** `guard_test.go:134,135,156`, `cloudflared_test.go:220,334`, `live_test.go:68`.

### 2. Critical findings
None.

### 3. Important findings
- **ARCH-DRY: deadline-bounded polling written by hand twice, while `testwait` exists.**
  - `guard_test.go:107-116` (`waitDead`) and `cloudflared_test.go:285-291` (the `startOrphan` loop) each repeat the `t.Deadline()` / `bounded` / margin logic.
  - `cmd/internal/testwait/testwait.go` was built for this rule and documents why: `Until(t, what, cond)` with a 10 s margin, or half the remaining time when less is left.
  - The hand-written copies use a 1 s margin. That leaves less room for the failure message and for later tests to report, and it is a second source of truth for the same rule.
  - Fix:
    - `waitDead`: `testwait.Until(t, "pid "+pid+" to exit", func() bool { return !procutil.Alive(pid) })`
    - `startOrphan`: `testwait.Until(t, "orphan's sleep child (did the fixture exec away?)", func() bool { return len(procutil.ProcessChildren()[pid]) > 0 })`
  - Both copies of this pattern in the window are listed above.

### 4. Minor findings
- **ARCH-PURPOSE: fixed wall-clock windows remain next to the swept helper.** The spec says "same rules as #480", and these are the same kind of window:
  - `guard_test.go:92` (`waitFile`, 5 s)
  - `guard_test.go:131` (`time.After(5s)`, "guard did not stop")
  - `cloudflared_test.go:218` ("Close took" > 3 s)
  - `live_test.go:65` (15 s teardown)

  The `waitFile` one shares a file with `waitDead` and fits the same `testwait.Until` shape. It isn't the issue's named flake, but it is the next one in the family.
- **Fixture depends on shell behaviour.** The fixture assumes `/bin/sh` doesn't exec the last command of a script. Some shells do this for `-c` strings. `sleep 300 & wait`, the pattern the guard test already uses, keeps the shell as parent on any shell. Today a shell that did exec would fail loudly rather than flake, so this is low risk.
- **Done-when only partly met under load.** It asks for 200 runs "under -race and load". The 200 runs saw only ambient load (load average 8–14), with the 500 ms delayed-reap variant standing in for load. That substitution is stated in the Log, but the delayed-reap variant isn't committed, so it can't be re-run.

### 5. Test coverage notes
- The regression guard is the fixture's own assertion. Restoring `exec` should fail with "orphan never started its sleep child", which is effectively deterministic. I could not confirm this because `ps` is unavailable in the review environment.
- With `-timeout 0`, `waitDead` and `startOrphan` wait forever. This matches `testwait`'s documented behaviour, so it's fine.

### 6. Architectural notes for upcoming work
- New test waits in `broadcast` should use `testwait` rather than hand-written loops. Converting this package's remaining `time.Now().Add(N*time.Second)` polls is the natural follow-up for the #479/#480/#487 family:
  - `host_test.go:244`
  - `server_test.go:354,411`
  - `session_test.go:321`
  - `notes_session_test.go:240`
  - `pointer_test.go:530`

### 7. Plan revision recommendations
- Add a `## Revisions` entry noting that `waitDead` and `startOrphan` use `testwait.Until` (the #479 helper) for the deadline bound, if the DRY finding is applied.

```findings
findings:
  - id: new
    severity: Important
    family: reuse-testwait-deadline-helper
    title: |
      waitDead and startOrphan hand-roll t.Deadline() polling that cmd/internal/testwait already provides (ARCH-DRY)
    detail: |
      Instances in window - guard_test.go:107-116 (waitDead) and cloudflared_test.go:285-291 (startOrphan sleep-child loop). Both re-implement testwait.Until with a divergent 1s margin vs testwait's 10s/half-remaining. Replace with testwait.Until(t, what, cond).
  - id: new
    severity: Minor
    family: wallclock-window-under-load
    title: |
      Sibling fixed wall-clock windows remain beside the swept waitDead (ARCH-PURPOSE)
    detail: |
      guard_test.go:92 waitFile 5s; guard_test.go:131 guard-did-not-stop 5s; cloudflared_test.go:218 Close took >3s; live_test.go:65 15s teardown. Same rule as #480; waitFile fits testwait.Until directly.
  - id: new
    severity: Minor
    family: fixture-shell-portability
    title: |
      startOrphan relies on /bin/sh not exec-optimizing the final command
    detail: |
      Using `sleep 300 & wait` (as the guard test does) keeps the shell as parent on any shell; current form fails loudly rather than flakes if a shell execs, so low risk.
  - id: new
    severity: Minor
    family: done-when-evidence-substitution
    title: |
      Done-when 200 runs under load were run at ambient load only, with an uncommitted delay variant standing in
    detail: |
      Disclosed in the Log; the 500ms delayed-reap reproduction is not checked in, so it cannot be re-run.
```

---

## Re-review — 2026-10-12T13:40:37-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 487 — tests: broadcast TestReapOrphans flakes under load (pid still alive after 2s) |
| repo | pair |
| issue file | workshop/issues/000487-tests-broadcast-testreaporphans-flakes-under-load-pid-still-alive-after-2s.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0c26f966973382e8ee7b2af30eedb47be50379ab..391d0219603083b5f25cc13781b1b28f46422960 |
| command | sdlc close --issue 487 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T13:40:37-07:00 |
| verdict | SHIP |

## Review

The tests are skipping because `ps` is unavailable in this sandbox. I'm checking the skip condition before deciding how much the run proves.

```verdict
verdict: SHIP
confidence: medium
```

The fix is sound and the round-1 Important finding is resolved. The flake came from the test fixture, not from the reaping code. The fake script ran `exec sleep 300`, which renamed the process to `sleep`. Under load that rename could land before `ReapOrphans` looked at the process, and reaping then correctly left alone a process that was no longer `cloudflared`. The fix drops the `exec`, has `startOrphan` wait until its `sleep` child exists, and then asserts the command is still `cloudflared`. `waitDead`, `waitFile` and the orphan setup loop now all use `testwait.Until`.

One limit on my evidence: `ps` is blocked in this review environment, even outside the sandbox. `needPS` therefore skips `TestReapOrphans` here, so I could not run the `exec` mutation myself. The package passes under `-race` (170s), and the guard and close tests passed. That the mutation goes red rests on reading the code: once the script execs, `sleep` has no children, so `testwait.Until` can only time out, with an error that names what it waited for. That matches the Log. The open items are Minor and do not block.

1. **Strengths**
   - The cause is correctly placed in the fixture, and the product check stays untouched. Reaping a process whose command has changed would be wrong, so leaving `killIfOurs` alone is right (Log, `cloudflared_test.go:269-274`).
   - `startOrphan` waits for a settled state (the child exists), then checks the command (`cloudflared_test.go:285-291`). A regression fails with a clear, repeatable message instead of a flake.
   - ARCH-DRY: `waitDead` is one line on `testwait.Until` (`guard_test.go:104-107`), and its old `within` parameter is gone from every caller (`cloudflared_test.go:221`, `cloudflared_test.go:331`, `guard_test.go:125-126`, `guard_test.go:147`, `live_test.go:68`).
   - `waitFile` captures the trimmed contents inside the poll closure, so it returns exactly the value the condition accepted (`guard_test.go:91-100`).

2. **Critical:** none.

3. **Important:** none.

4. **Minor:** BR-2 and BR-3 stay open (see the dispositions below). A Log nit: the first 2026-10-12 bullets still say "orphan never started its sleep child" and "waits until `t.Deadline()` minus 1 s". A later bullet replaces both, so this is history and not worth a finding.

5. **Test coverage notes**
   - The `startOrphan` settle check is the permanent regression guard. Reading the code confirms it can only time out if `exec` comes back: before the exec the shell's `echo` is a builtin and starts no child, and after it `sleep` has none.
   - Because of the `ps` sandbox limit, the runtime red/green check is the implementer's and was not repeated here.

6. **Architectural notes**
   - ARCH-DRY passes: this window reuses `testwait` and adds no new hand-rolled polling.
   - ARCH-PURE passes: the diff is test code only, with no product logic mixed into IO.
   - ARCH-PURPOSE passes for the issue's stated scope (the 2s window and the reaping race). The leftover fixed windows in the same guard/cloudflared fixture family are BR-2, which is Minor.

7. **Plan revisions:** none needed. The four checked Plan items match the code and the Log.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      waitDead (guard_test.go:104-107), waitFile (guard_test.go:91-100) and startOrphan's settle loop (cloudflared_test.go:285-288) all now call testwait.Until; no hand-rolled t.Deadline polling remains in the window.
  - id: BR-2
    disposition: not-addressed
    note: |
      waitFile is swept, but positive event waits in the same guard/cloudflared family still use fixed windows: guard_test.go:69 (10s), 122/141/162 (5s), cloudflared_test.go:239 (5s), live_test.go:65 (15s). Fix: use select on testwait.After(t). The Log's out-of-scope list (host/notes/pointer/server/session) does not cover these files. Minor, non-blocking.
  - id: BR-3
    disposition: not-addressed
    note: |
      The fixture is still a bare `sleep 300` as the script's last command. bash and dash do not exec-optimize the last command of a script file, and the settle check would turn any such shell into a deterministic, named failure, so this stays low-risk Minor. `sleep 300 & wait` would close it.
  - id: BR-4
    disposition: addressed
    note: |
      The Log now names the permanent guard: startOrphan's settled-state check fails deterministically on the causal regression (exec), so the substitute delay variant no longer carries the evidence. The 200/200 -race runs at ambient load up to 25 are disclosed.
```
