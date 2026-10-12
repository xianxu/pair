# Boundary Review — pair#454 (milestone M1)

| field | value |
|-------|-------|
| issue | 454 — broadcast-web: Chrome in Couch — start, go live, pause, end; one cast |
| repo | pair |
| issue file | workshop/issues/000454-broadcast-web-chrome-in-couch-start-go-live-pause-end-one-cast.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 1e941c926f0c345c15fdfbf4b28963967a77385b..8a15f620b09a70c227da4c75ee6c5ac7ce66d775 |
| command | sdlc milestone-close --issue 454 --milestone M1 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-11T20:38:54-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the M1 diff: cdp and pure webcast files are done; reading chrome.go, fake, inject.js and the tests next.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M1 delivers the browser-control component the plan describes. That covers the `cdp` pipe client (bounded queue, frame cap, routing by id), the pure `Tracker` and `translate`, the `Browser` seam, a stateful `FakeBrowser` that drives the same `Tracker`, the isolated-world `inject.js` with its node tests, and real `Chrome` with a conformance suite shared between the fake and live Chrome. `go vet` is clean and `go test -race` passes for `cdp` and `webcast`.

I could not run the live conformance check. This review process is still inside a sandbox, so Chrome exits with status 21 at once ("Failed to create a ProcessSingleton… socket directory"). That is the limitation the plan's spike notes record, so the live behavior rests on the implementor's run. That gap is why confidence is medium.

What stands between this and SHIP is four cheap fixes:
- `Chrome.Err()` labels a protocol failure as "the operator closed Chrome".
- `cmd.Wait` has no bound, so `Close` can hang.
- Navigating to a page the script can't run in shows the previous page's title.
- `ErrProfileInUse` is reported for any exit 21 and drops Chrome's output.

The other gap is that the Chrome event loop's state transitions are only tested by the live run.

## 1. Strengths
- **`cdp.Conn` design** (`cmd/internal/cdp/conn.go:204-261`). The reader never blocks on the consumer, the queue is bounded and fails with `ErrBacklog`, and a call made from inside the event loop doesn't deadlock (`TestCallFromEventLoopWithQueuedEvents`). Events are still drained after EOF. This is solid.
- **`Phase` is a tagged enum and one goroutine owns the state** (`state.go`, `chrome.go:234-272`). CDP events, requests and timers all arrive through one `select`, so there is no set of independent flags guarded by locks. This is ARCH-ORDER in spirit.
- **The active-tab rule exists once** (ARCH-DRY). `FakeBrowser` drives the real `Tracker` instead of restating it. `TestTrackerInvariants` checks generated input sequences against independently stated invariants. `FuzzFrameReader` and `FuzzTranslate` cover the parsers that read untrusted input.
- **The binding is invisible to the page.** It lives in an isolated world, the control sits in a closed shadow root marked `data-couchcast-ui`, and the injection guard makes it idempotent. The node tests pin all of this, run the way Chrome runs the script.
- The atlas (`atlas/webcast.md`) is accurate, is linked from the index, and states when the live check must be run.

## 2. Critical findings
None.

## 3. Important findings
- **`chrome.go:405-416`: the cause of a connection failure is reported as the operator closing Chrome (ARCH-SECURE: degrade visibly).**
  - What happens: if the Conn fails with `ErrBacklog`, `ErrFrameTooLarge` or a malformed message, the events channel closes. `finish` then closes the pipe and Chrome exits with status 0. The `default` branch sets `c.err = ErrClosedByOperator`.
  - Why it matters: the plan promises `ErrBacklog` "is never silently dropped" and that `Err()` returns "the failure". Couch would show "Chrome closed" for a protocol failure.
  - Fix: in `default`, check first whether `c.conn.Err()` is set and is neither `cdp.ErrClosed` nor an EOF read error; if so, return it wrapped.
  - Test: run the loop against a scripted Conn that trips the backlog (see the next finding).
- **The Chrome event loop's transitions are only tested live (ARCH-ORDER / ARCH-PURE).** The live check is skipped in normal `go test`, so none of these are exercised by default:
  - the first cause of ending wins;
  - `setLive` is ignored while ending;
  - a `TargetCreated` cancels the no-pages timer;
  - how the ending cause maps to `Err()`;
  - `hello` gets an answer.
  - Fix: make `handle` / `setLive` / `beginClose` return effects (or take an injected `call` function), or build `Chrome` over an injected `cdp.Conn` plus a process handle. Then sequence tests can run against a scripted pipe peer.
- **`chrome.go:139-164`: `cmd.Wait` has no time bound.**
  - `cmd.Stdout` and `cmd.Stderr` are a non-file writer, so Wait also waits for every inherited copy of the stderr pipe to close. Chrome's helper processes inherit it.
  - If a helper outlives the browser process, `exited` never closes, and `finish`'s `Kill(); <-c.exited` blocks forever. `Close()` would then hang (plausible; I couldn't run Chrome to confirm).
  - Fix: set `cmd.WaitDelay`, for example to 2s.
- **`tracker.go:86-89`: the previous page's title survives a navigation to a page the script can't run in.**
  - What happens: `TargetChanged` updates the URL but keeps `pageTitle`. After navigating to `chrome://settings` or a `chrome-error://` page, `Active()` returns the new URL with the old document's title.
  - Why it matters: in M2 that title reaches viewers, and the Done-when says the URL and title update when the operator navigates.
  - Fix: clear `pageTitle` when the URL in `TargetChanged` differs from the stored URL.
  - Test: `TargetCreated A` → `PageTitle A "one"` → `TargetChanged A chrome://settings "Settings"` should give the title "Settings".
- **`chrome.go:190-196`: exit status 21 is always reported as `ErrProfileInUse`, and Chrome's output is dropped.**
  - Chrome also exits 21 when it can't create its singleton socket directory. I reproduced this here with an empty temporary profile; every live test reported "profile in use".
  - Fix: append `c.output.quote()` to the `ErrProfileInUse` error, and match on the singleton text rather than on the exit code alone.

## 4. Minor findings
- `tracker.go:116-131`: when `activated` values tie (for example two pages opened in the background with `activated` = 0 and every page hidden), the choice depends on map iteration order and can flip between publishes. Break ties by ID.
- `chrome.go:348-351`: `SetLive` during ending returns nil without doing anything. The contract says `ErrEnded`.
- `FakeBrowser.CloseTab` ends at once, while real Chrome waits `noPagesGrace`. The fake also never models the page-title path (`PageTitle`), so the stale-title bug can't show up against the fake.
- `cdp.Conn.Call`: a blocked `Write` ignores the ctx deadline. This is unlikely on a pipe.
- Titles from the page have no length cap other than the 16 MiB frame limit. Bound them before they go to viewers in M2.
- `finish` kills only the main process, not Chrome's helper processes.

## 5. Test coverage notes
- Covered well: the pure layer (tracker, translate, publisher, framing with fuzz tests), `cdp.Conn` error paths, the fake, and inject.js under node. Everything passes with `-race`.
- Not covered by default: the event loop in `chrome.go`. Without the live run, nothing executes it.
- The live run (`WEBCAST_LIVE_CHROME=1`) could not run in this review's environment. The operator should run it unsandboxed before closing M1, as `atlas/webcast.md` asks.

## 6. Architectural notes for upcoming work
Each principle, with pass or flag:

| Principle | Result | Note |
|---|---|---|
| ARCH-DRY | pass | `publisher[T]` and `Tracker` are shared. |
| ARCH-PURE | flag | The loop's logic sits inside the IO shell (see Important). |
| ARCH-PURPOSE | pass | The M1 scope is fully delivered. |
| ARCH-MOCK | pass, with notes | The fake and the conformance suite are in place; the live cadence is documented. The title path isn't modeled in the fake. |
| ARCH-CONSTRAINTS | pass | Queue cap, frame cap, call timeout and the bounded stderr ring are all in place. |
| ARCH-SECURE | pass, with notes | The pipe has no network surface and the binding is invisible to the page. The `Err()` misattribution and the page title flowing to viewers need care in M2. |
| ARCH-ORDER | flag | Owning all state on one loop is good, but its transitions aren't sequence-tested. |
| ARCH-FUNERAL | pass | Chrome is collected through the pipe, killed after the wait, and the profile dir is documented. `WaitDelay` is still needed to bound reaping. |

For M2:
- `Cast` should treat `Err()` as authoritative, so fix the misattribution first.
- The viewer must render titles with `textContent` only.

## 7. Plan revision recommendations
Add a `## Revisions` entry for M1 covering these divergences from the plan:
- **Task 3:** `subscribe.go`/`latest[T]` became `publish.go`/`publisher[T]` (fan-out to several subscribers, with `finish`).
- **Task 4:** the node test lives at `tests/webcast/*.test.mjs` and runs via `node_test.go`.
- **Core concepts / Tracker:** new `PageTitle` input and `title` binding report, because Chrome reports `<title>` late (measured).
- **Chrome attach:** browser-level `Target.setAutoAttach{waitForDebuggerOnStart}` plus `runIfWaitingForDebugger` replaces attaching to each target.
- **LaunchOptions:** now `{Binary, ProfileDir, StartURL, StartTimeout, Headless}`; `Stderr io.Writer` became the internal 8 KiB tail.
- **Task 9a:** the live harness runs headless, not headful. Activation uses focus emulation, and real tab switching moves to the smoke test. So far this is recorded only in the issue's Log.

```findings
findings:
  - id: new
    severity: Important
    family: failure-cause-misattributed
    title: |
      Chrome.Err() reports a cdp Conn failure (ErrBacklog, ErrFrameTooLarge, malformed) as ErrClosedByOperator
    detail: |
      chrome.go:405-416 default branch ignores c.conn.Err(); after a conn failure Chrome exits 0 on pipe close, so the cause is lost, contrary to the plan's never-silently-dropped promise. Prefer a non-EOF, non-ErrClosed conn error; add a regression test.
  - id: new
    severity: Important
    family: state-machine-untested-sequences
    title: |
      Chrome run-loop transitions (end cause precedence, setLive while ending, noPages cancel, Err mapping) have no default-run tests (ARCH-ORDER/ARCH-PURE)
    detail: |
      Only the env-gated live test reaches chrome.go's loop. Extract the transitions to return effects, or build Chrome over an injected Conn plus process handle, then sequence-test against a scripted pipe peer.
  - id: new
    severity: Important
    family: unbounded-process-reap
    title: |
      cmd.Wait without WaitDelay can block forever on inherited stderr, hanging Close via finish's Kill then wait on exited
    detail: |
      chrome.go:139-164, 397-402. Stdout/Stderr are a non-file writer, so Wait waits on helper processes holding the pipe. Set cmd.WaitDelay.
  - id: new
    severity: Important
    family: stale-derived-state-on-navigation
    title: |
      Tracker keeps the previous document's pageTitle after navigating to an uninjectable page (chrome://, chrome-error://)
    detail: |
      tracker.go:86-89 updates URL but not pageTitle, so Active() pairs the new URL with the old title and viewers would see the wrong title. Clear pageTitle when the URL changes; add a tracker case.
  - id: new
    severity: Important
    family: failure-cause-misattributed
    title: |
      launchError maps every exit 21 to ErrProfileInUse and drops Chrome's output
    detail: |
      Reproduced: with a fresh empty profile, Chrome exits 21 when it cannot create its ProcessSingleton socket dir, and every live test reported profile in use. Quote c.output in the error and match the singleton text, not the code alone.
  - id: new
    severity: Minor
    family: nondeterministic-tie-break
    title: |
      Tracker.Active picks among equal activated values by map iteration order, so the published tab can flip
  - id: new
    severity: Minor
    family: contract-drift
    title: |
      Chrome.SetLive while ending returns nil with no effect; Browser contract says ErrEnded
  - id: new
    severity: Minor
    family: fake-fidelity
    title: |
      FakeBrowser does not model the PageTitle path or noPagesGrace, so title and close-grace bugs cannot show up against the fake
  - id: new
    severity: Minor
    family: plan-drift
    title: |
      Plan needs a Revisions entry for the M1 divergences
    detail: |
      publisher[T] replaces latest[T]; PageTitle input added; setAutoAttach replaces per-target attach; LaunchOptions changed; live harness headless; node test path moved.
```

---

## Re-review — 2026-10-11T20:47:01-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 454 — broadcast-web: Chrome in Couch — start, go live, pause, end; one cast |
| repo | pair |
| issue file | workshop/issues/000454-broadcast-web-chrome-in-couch-start-go-live-pause-end-one-cast.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 1e941c926f0c345c15fdfbf4b28963967a77385b..32b648e3992d2d58975aee05cf2a4702c00e6154 |
| command | sdlc milestone-close --issue 454 --milestone M1 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-11T20:47:01-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the M1 range: I've read the fix commit, chrome.go, the run-loop tests and the cdp conn. Next I'm running the test suite and checking the fake, tracker and plan against the prior findings.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Most of the prior findings are fixed with real evidence. I broke the fixes for BR-2, BR-5 and BR-8 one at a time in the scratch checkout and each regression test failed. Adding the `process` seam plus a scripted CDP peer (`chrome_loop_test.go`) is the right answer to BR-3. Pulling the end reason into a pure `endError` is clean and covered row by row.

Three things stand in the way of SHIP:
- **Flaky test.** `TestChromeExitReasons/lost_connection` failed 4 of 10 times on unmodified code (`go test -count=10`).
- **New race in the BR-2 fix.** It can report an ordinary operator quit as "lost Chrome". I reproduced the mechanism.
- **BR-4 has no test.** The `WaitDelay` fix has nothing that fails without it.

`go vet` is clean, and `go test -race -count=3` passed for both packages.

**1. Strengths**
- `endError` (`chrome.go:451`) is a pure precedence table. `TestEndError` tests every row, and dropping the conn-error branch makes the backlog row fail.
- The `process` interface plus `peer` (`chrome_loop_test.go:21-137`) drive the real `run`/`finish` code with no real Chrome. They cover setup-then-release, release after a failed setup, last-window grace and its cancel, kill after `closeWait`, and `SetLive` while ending.
- `profileInUse` reads Chrome's own output, and `TestProfileInUse` (`chrome_loop_test.go:321`) includes a look-alike exit-21 failure that must not match.
- `DocumentChanged` comes only from top-frame navigations (`translate.go:114-129`). Same-document navigations keep the title, and a subframe case was added to the spike fixture.
- Fuzz tests (`FuzzFrameReader`, `FuzzTranslate`) and a generated-sequence Tracker property test exist, as the Revisions promised.

**2. Critical findings**
None.

**3. Important findings**
- **Flaky lost-connection test** (`chrome_loop_test.go:277-284`; root cause `cdp/conn.go:197-199`).
  - The cdp reader stops reading after a malformed or oversized frame. This contradicts its own comment at `conn.go:205-206` ("the reader keeps draining, so Chrome never blocks writing").
  - The peer then blocks writing. It never sees its command pipe close, so `finish` kills it after `closeWait` (5s). `waitDone`'s deadline is also 5s, so the test is a coin flip.
  - Real Chrome gets the same treatment: any connection failure ends in SIGKILL after 5s instead of a clean exit.
  - Fix: make `read` keep discarding frames after `fail` for every failure kind, and add a cdp test that the writer never blocks after a malformed frame. Set the test deadline from `closeWait`, or shorten `closeWait` in the test.
  - This is the 2nd finding in family `contract-drift`. Rule: every property a doc comment promises about `Conn` (drains after failure, the list of possible `Err` values) is pinned by a test. Apply it to every such comment in `conn.go`, not just this one.
- **A clean quit can be reported as "lost Chrome"** (`chrome.go:157-161`, `:432`, `:458`).
  - Launch's Wait goroutine closes `outR` before closing `exited`. If that happens before the reader has seen EOF, the reader's error is `cdp: read: read |0: file already closed` (an `os.ErrClosed`, not `io.EOF`).
  - I reproduced it: `endError` then returns `webcast: lost Chrome: …` for an operator quit.
  - A write to a pipe Chrome already closed (EPIPE, wrapped as `cdp: write:`) gets the same misreading.
  - The scripted peer can't show this, because its exit closes the writer side, never `outR`.
  - This is the 3rd finding in family `failure-cause-misattributed`. Rule: an end cause is only something observed before our own teardown began. Errors that teardown can produce (`conn.Close`, `outR.Close`, writes after exit) are never causes.
  - Fix: have the loop record whether the event stream closed while the process was still running. Pass that fact to `endError` instead of the raw `conn.Err()`. Add a peer mode whose exit closes the reader side, and a test where it must yield `ErrClosedByOperator`.
- **BR-4 is not addressed.** `cmd.WaitDelay` at `chrome.go:155` is correct on inspection, but no test fails without it (I removed it and everything still builds and passes). A cheap regression test: build `osProcess` from `sh -c 'sleep 30 & exit 0'` with stdout set to a `tail`, and assert `Exited()` closes within `WaitDelay` plus a margin. Moving the cmd-to-`osProcess` setup into a small helper would make that test reach production code.

**4. Minor findings**
- `Chrome.exited` (`chrome.go:71`) is now dead: never set and never read.
- BR-7: the tie-break fix is in, but no test sets up two pages with equal `activated` values.
- BR-9: `FakeBrowser` still doesn't model `PageTitle`/`DocumentChanged` or the no-pages grace (`fake.go` is unchanged).

**5. Test coverage notes**
- My mutation checks: dropping the conn-error branch fails `TestEndError`; making `setLive` return nil while ending fails `TestChromeEndCausePrecedenceAndKill`; dropping the title clear fails `TestTrackerURLTitleFollowChanges`. Removing `WaitDelay` breaks nothing.
- Make the loop tests' deadlines depend on the production timers they wait through (`closeWait`, `noPagesGrace`). Fixed 5s constants are what made the lost-connection test flaky.

**6. Architectural notes**
- **ARCH-DRY: pass.** Minor overlap: kill-then-wait appears in both `killAt` and `finish`.
- **ARCH-PURE: pass.** `endError`, `profileInUse`, `translate` and `Tracker` are pure and tested without IO.
- **ARCH-PURPOSE: pass** for the M1 scope.
- **ARCH-MOCK: partial.** The peer models the conn and process together, which is good. It doesn't model the OS closing `outR` (the race above), and `FakeBrowser` still has the BR-9 gaps. The live conformance suite and its schedule exist.
- **ARCH-CONSTRAINTS: pass.** Bounded output tail, call timeout, start timeout, and backlog cap.
- **ARCH-SECURE: pass.** The output tail stays in memory, the profile dir is created 0700, and Chrome is started with an argument list, not a shell string.
- **ARCH-ORDER: flag.** The loop treats "process exited" and "event stream closed" as the same input, so their relative order — the evidence that tells a quit from a lost connection — is thrown away (the race above). The single-goroutine design is otherwise sound.
- **ARCH-FUNERAL: pass.** The process is always reaped within a bounded time once `WaitDelay` is set. The profile dir is durable by design and documented.

**7. Plan revision recommendations**
Add to the "M1 as built" entry: the reader drains after every failure; the end reason uses which happened first (stream closed or process exited), not the raw conn error; and a note on how `WaitDelay` is tested.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The Revisions Test strategy entry adds the per-function strategy lines (splitFrames fuzz, Tracker property test), and FuzzFrameReader and the tracker rand test exist.
  - id: BR-2
    disposition: addressed
    note: |
      endError reads the conn error; dropping that branch fails TestEndError (backlog row). A related race is raised separately.
  - id: BR-3
    disposition: addressed
    note: |
      process seam plus scripted peer in chrome_loop_test.go; the SetLive-while-ending mutation goes red. One of its tests is flaky (raised separately).
  - id: BR-4
    disposition: not-addressed
    note: |
      WaitDelay set at chrome.go:155 but removing it fails no test; add an sh 'sleep 30 &' osProcess test.
  - id: BR-5
    disposition: addressed
    note: |
      DocumentChanged clears pageTitle; removing the clear fails TestTrackerURLTitleFollowChanges.
  - id: BR-6
    disposition: addressed
    note: |
      profileInUse matches Chrome's output, TestProfileInUse covers a look-alike exit-21 failure, and the output is quoted via endError.
  - id: BR-7
    disposition: not-addressed
    note: |
      Tie-break by id implemented at tracker.go:121-125, but no test covers equal activated values.
  - id: BR-8
    disposition: addressed
    note: |
      setLive returns ErrEnded; the mutation fails TestChromeEndCausePrecedenceAndKill.
  - id: BR-9
    disposition: not-addressed
    note: |
      fake.go unchanged: no PageTitle/DocumentChanged and no no-pages grace.
  - id: BR-10
    disposition: addressed
    note: |
      The "M1 as built" Revisions entry lists publisher, PageTitle, setAutoAttach, LaunchOptions, headless harness, and node test path.
findings:
  - id: new
    severity: Important
    family: contract-drift
    title: |
      cdp reader stops draining after a malformed or oversized frame, despite its comment; lost-connection loop test flakes 4 of 10 runs
    detail: |
      conn.go:197-199 returns from read, so the peer (and real Chrome) blocks writing, never sees its command pipe close, and is SIGKILLed after closeWait (5s), which races waitDone's 5s. Rule: every Conn doc-comment promise is pinned by a test. Keep draining after every failure kind, test that the writer never blocks, and set test deadlines from closeWait.
  - id: new
    severity: Important
    family: failure-cause-misattributed
    title: |
      A clean operator quit can be reported as "lost Chrome": the Wait goroutine's outR.Close yields os.ErrClosed, which endError treats as a conn failure
    detail: |
      3rd in family. Rule: an end cause is only what was observed before our own teardown began; errors teardown can produce (outR.Close, conn.Close, EPIPE after exit) are never causes. Have the loop record whether the event stream closed while the process still ran, and pass that to endError. Reproduced: endError(notEnding, "read |0: file already closed", 0) returns "lost Chrome". Add a peer exit mode that closes the reader side.
  - id: new
    severity: Minor
    family: dead-code
    title: |
      Chrome.exited field (chrome.go:71) is never set or read after the process seam refactor
```

---

## Re-review — 2026-10-11T20:51:54-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 454 — broadcast-web: Chrome in Couch — start, go live, pause, end; one cast |
| repo | pair |
| issue file | workshop/issues/000454-broadcast-web-chrome-in-couch-start-go-live-pause-end-one-cast.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 1e941c926f0c345c15fdfbf4b28963967a77385b..6c86576ebc04593f77579108ec0e4aa33dc9dfd7 |
| command | sdlc milestone-close --issue 454 --milestone M1 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-11T20:51:54-07:00 |
| verdict | SHIP |

## Review

Tests pass 10/10. Now checking that each claimed fix goes red when reverted.

```verdict
verdict: SHIP
confidence: high
```

All three Important findings from round 2 (BR-4, BR-11, BR-12) are fixed, and the fixes follow the rule each finding named rather than patching one instance. I reverted each fix in this checkout, and each time a test went red:
- **WaitDelay removed:** `TestStartProcessReapsDespiteInheritedOutput` times out at 5s.
- **Drain removed:** `TestReaderDrainsAfterProtocolFailure` fails with "peer blocked writing".
- **Old deny-list `endError` restored:** the `TestEndError` rows fail every time. The loop-level `quit, reader closed first` subtest also fails, but only on some runs (it depends on goroutine timing).

`go vet` is clean. The two packages passed with `-count=10`, then with `-race -count=3`, and `TestChromeExitReasons` passed 40 runs under `-race`. So the lost-connection test that used to fail 4 of 10 runs is now stable.

The only thing left is BR-7, a Minor: the tie-break code is there, but no test fails when it is removed. Nothing blocks SHIP.

1. **Strengths**
   - `cdp.ErrProtocol` (`cmd/internal/cdp/frame.go:25`) marks the failure class where it starts. `endError` then accepts only that class (`cmd/internal/webcast/chrome.go:470`), so a list of excluded transport errors can't be incomplete again. The lesson is recorded in `workshop/lessons.md`.
   - The reader keeps draining after every kind of failure: oversized and malformed frames drain at `conn.go:195,204`, and a backlog drains through `dispatch`'s `c.err != nil` early return.
   - `startProcess` (`chrome.go:204`) is a small process seam that can be tested on its own, with a real `sh` child holding the output pipe.
   - The scripted peer has a new `closeReader` mode, so the reader-closed-first ordering can be produced on purpose instead of hoped for.
2. **Critical:** none.
3. **Important:** none.
4. **Minor:** only BR-7, carried over (below). One nit: the `ErrProtocol` doc says a backlog is "what the peer sent", but a backlog actually means our consumer stopped reading. The classification is still right for `endError`; only the wording is loose.
5. **Test coverage:** pinned by tests that fail when the fix is removed: BR-4, BR-11 and BR-12. Not pinned: the BR-7 tie-break. Reverting it left the Tracker, Fake and Conformance tests green over 20 runs.
6. **Architecture:**
   - **ARCH-DRY:** pass.
   - **ARCH-PURE:** pass. `endError` is a pure function with a table test.
   - **ARCH-PURPOSE:** pass. BR-12 was fixed by its class, not just the named site.
   - **ARCH-MOCK:** pass. The process seam and the scripted peer share the production boundary. BR-9's gap is covered by the scripted peer.
   - **ARCH-CONSTRAINTS:** pass. Reaping is bounded by `processWaitDelay`.
   - **ARCH-SECURE:** N/A. Nothing new crosses a trust boundary.
   - **ARCH-ORDER:** pass. `finish` reads `conn.Err()` before its own `Close`, and the scripted peer can now produce the reader-closed-first ordering on demand.
   - **ARCH-FUNERAL:** pass. The drain goroutine ends when the reaper closes `outR`.
7. **Plan revisions:** none. The "M1 review round 2" Revisions entry already matches the code.

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      startProcess sets WaitDelay (chrome.go:205); with it reverted, TestStartProcessReapsDespiteInheritedOutput times out at 5s.
  - id: BR-7
    disposition: not-addressed
    note: |
      Tie-break by id is in place (tracker.go:121-125), but reverting it leaves Tracker/Fake/Conformance green over 20 runs. Add a tracker test with 3+ background-only hidden pages and call Active repeatedly.
  - id: BR-9
    disposition: withdrawn
    note: |
      Gap accepted and documented (fake.go CloseTab doc, issue line 91); the scripted-peer loop tests cover the real PageTitle/grace paths through production code.
  - id: BR-11
    disposition: addressed
    note: |
      Drain after oversized/malformed (conn.go:195,204); with it reverted, TestReaderDrainsAfterProtocolFailure fails. Exit-reason tests pass 40/40 under -race.
  - id: BR-12
    disposition: addressed
    note: |
      endError accepts only cdp.ErrProtocol; with the deny-list restored, TestEndError rows fail every time and the closeReader loop subtest fails on some runs.
  - id: BR-13
    disposition: addressed
    note: |
      Chrome.exited field removed (chrome.go:69-73); package builds and vets clean.
```
