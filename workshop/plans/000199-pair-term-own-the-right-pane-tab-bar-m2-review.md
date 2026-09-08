# Boundary Review — pair#199 (milestone M2)

| field | value |
|-------|-------|
| issue | 199 — pair term: own the right pane tab bar |
| repo | pair |
| issue file | workshop/issues/000199-pair-term-own-the-right-pane-tab-bar.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 96f780da661b35aef301b612b14ee8399bd9a6cf..9b111c646a5659234ff8b840a91230c6caca037c |
| command | sdlc milestone-close --issue 199 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-07T21:54:54-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M2's core mechanism is right and is genuinely pinned: I mutation-tested the diff and confirmed that making `redrawTab` write directly turns `TestOnlyOneGoroutineWritesTheHost` red (2 writers), that removing the `MidSequence()` check turns the defer/owe test red, that feeding our own bytes into `hostScan` turns `TestGateIsNotFedOurOwnWrites` red, and that restoring `cmd.Stderr = os.Stderr` turns the FD test red — so BR-4's stderr half is really closed, and closing it at the `Runtime` rather than at five call sites is the class-level fix ARCH-PURPOSE asks for. What blocks the boundary is one thing the tests structurally cannot see: **`removeTab` runs on the writer goroutine and posts to the writer goroutine's own channel**, so closing a tab while another tab is producing output deadlocks the sole writer permanently. I reproduced it with a scratch test (below) — 64 slots filled, writer parked forever. Two Important findings follow behind it: two negative assertions in the new suite pass vacuously (mutation-verified — deleting `m.owed = nil` leaves the whole suite green), and the milestone silently discards the diagnostics the plan said it would capture and log.

## 1. Strengths

- **The subprocess envelope was closed at the Runtime, not at the call sites** (`run.go:1325-1333`). `RunZellijAction` delegating to `RunZellijActionQuiet` covers `layoutcmd`/`draftroute`'s sites and the next site anyone adds; the comment states exactly why. This is the finding-answered-as-a-class that ARCH-PURPOSE asks for, and I verified `termcmd.OSRuntime` has no consumer outside `termcmd` so the blast radius is real and bounded.
- **The FD assertion is mutation-sensitive where it matters most.** Overlaying `cmd.Stderr = os.Stderr` fails all four subtests with byte counts (28 / 167). BR-4's third raise is genuinely disposed on the code side.
- **The gate's hardest rule is pinned.** Overlaying `m.hostScan.FeedFraming(b)` into `writeOwn` reddens three tests — the "child bytes only" half that couch got wrong first is defended by a test, not by a comment.
- **The M2.5 Log entry is honest evidence** (`issue:190-210`): it separates the idle-pane run ("proves little — an idle child never leaves the stream mid-sequence") from the flood run, and checks the PID/start-time to confirm the new binary was under test. That is the shape a manual acceptance should have.
- **`drainForTest`'s bare-event design** (`run.go:895`) and the `chunk.data == nil` branch: a probe that does not perturb what it observes, with the earlier failure recorded.

## 2. Critical findings

**C1 — `removeTab` posts to the channel the writer goroutine drains; a full buffer deadlocks the pane permanently.** `run.go:758` → `run.go:1081` → `enqueue` (`run.go:859`).

`handleChunk`'s `chunk.err != nil` branch runs `removeTab` **on the writer goroutine**. `removeTab` calls `setPaneTitle` (a real `zellij action rename-pane` subprocess, tens of ms) and then `m.redrawTab(activeSnapshot)`, which since this milestone is `m.enqueue(...)` → `m.output <- chunk`. The buffer is 64 (`run.go:673`) and `ptychild` reads in 4 KiB chunks, so a second tab producing output fills it in milliseconds while the writer is parked in the subprocess. The send then blocks with no consumer, and `enqueue`'s only escape is `<-m.done`, which nothing will close. The pane freezes for good; the children then block on the Sink's unguarded `m.output <-` (`run.go:698`), so output stops everywhere, not just on the strip.

Reproduced (overlay test, no repo edits) — `blockingRuntime.RunZellijAction` stands in for the zellij spawn:

```
filled 64 slots
DEADLOCK: writer goroutine blocked posting to its own channel
```

Failure scenario: two tabs, one running `yes`/a build/`tail -f`, operator closes the other → `pair term` renders nothing ever again. This is the exact workflow the issue is about ("I don't split really… I use tab").

Fix sketch: `removeTab`'s only production caller is the writer goroutine, so split the takeover and apply it inline —

```go
func (m *terminalMux) applyTakeover(body []byte) { // writer goroutine only
    m.hostScan = ptychild.Screen{}; m.owed = nil; _, _ = m.stdout.Write(body)
}
func (m *terminalMux) redrawTab(replay []byte) { m.enqueue(ptyChunk{own: takeoverBody(replay), takeover: true}) }
```
and have `removeTab` call `applyTakeover` (the `takeover` case in `handleChunk` becomes `applyTakeover(chunk.own)`, so there is one implementation). Also move `setPaneTitle`'s subprocess off the writer loop — pre-existing, but it is now a stall of *all* pane output and it is what widens the deadlock window. Regression test: two tabs, one flooding, close the other, assert the loop still drains within a timeout.

## 3. Important findings

**I2 — Two negative assertions in the new suite pass vacuously. This is the 2nd finding in family `uncovered-negative-assertion`, so the fix is the rule, not the instance.**

Measured, both by overlay mutation:
- Deleting `m.owed = nil` (`run.go:766`) leaves **every** M2 test green, including `TestTakeoverResetsTheGateAndDropsTheOwedPaint`. The `STALE` assertion (`writer_test.go:197`) cannot fail, because after the takeover the test feeds no further child chunk and `flushOwed` is only reachable from the `default` branch. The scan-reset half is pinned; the drop half is not — so couch's third gate rule is half-defended, which is the half BR-5 was raised about.
- `cmd.Stdout = os.Stdout` in `runZellij` also leaves the FD test green here, because with no live zellij session `list-clients` fails and writes only stderr. The test's comment asserts "a succeeding action writes stdout", which is ambient-state-dependent; on a machine without `zellij` installed, *both* arms are vacuous. Note `writer_test.go:227-228` builds the positive control (`var stdout, stderr bytes.Buffer` + a direct `runZellij` call) and then never asserts on it — the control was written and dropped. (ARCH-MOCK: this test also drives the real `zellij` binary rather than a seam, so its coverage varies by machine.)

The rule: **a negative assertion ("nothing was written", "the stale bytes did not land") is only evidence when the same test first establishes that the thing *would* have been written.** Every `assert-absent` gets a paired positive control in the same test — assert the recording buffer actually received the subprocess's bytes; drive the event that *would* flush the owed paint (a following boundary-ending child chunk) before asserting it did not land. Enumeration to sweep now: the four assert-absent sites in `writer_test.go` (:118 — has a control, fine; :197 — none; :255, :258 — control built and unasserted) and `run_test.go:910`.

**I3 — Diagnostics are silently destroyed by the mechanism that keeps them off the pane.** Two sites, one rule.
- `run.go:1331`: `runZellij(cmdArgs, io.Discard, io.Discard)`. The plan (finding 8, M2.3 piece 5) specifies "the captured text is **logged**, never written to the pane's fd"; the code discards it. A failing `zellij action` now surfaces as bare `exit status 1` with the reason gone — including on the non-pane `pair term --test-shortcut` path, where the old code did print it.
- `run.go:833`: `reportError` routes through `paintOwn`, so a diagnostic lands in the **coalescing** `owed` slot (`run.go:809`). A later paint replaces it, `redrawTab` drops it (`run.go:766`), and nothing flushes it if the child stops writing — the slot has no timer and no flush on the `own`/`err`/`takeover` branches. couch does not have this problem because its debt is a `paintPending` **bool** re-derived by `paintNow` (`couchtty/console.go:1007,1137`); `termcmd` stores bytes, and the "freshest paint wins" argument in the comment is true for a strip and false for an event. M3 makes it certain: every strip repaint will clobber a pending diagnostic. (ARCH-DRY / ARCH-ORDER.) Fix: keep diagnostics out of the coalescing slot (a small queue, or the same `pending bool` + re-render shape couch uses), and capture `zellij`'s stderr into a buffer that `reportError` surfaces. Minor, same site: `err.Error()` reaches the pane unsanitized — route it through `rowtext.Sanitize` when M3 lands it.

**I4 — M2.3's steps 3, 4 and 5 describe an implementation the code does not have, with no `## Revisions` entry. This is the 3rd finding in family `plan-table-drift`, so state the rule, don't patch the three lines.** Piece 3 says "route every `RunZellijAction` call in `termcmd` through `RunZellijActionQuiet` … the five sites are `:191, :457, :463, :961, :963`" — those call sites are unchanged; the Runtime was changed instead (better, and the atlas explains it, but the plan still directs the next reader to the wrong artifact). Piece 5 says capture and log stderr — discarded. Piece 4 says route `term:` diagnostics through the writer loop with no exception — the startup ones still write stderr directly (correctly, and the code says why; the plan does not). The rule: **ticking a step asserts the code matches the step's text; where the implementation deviated, the step's text is corrected in a `## Revisions` delta in the same commit.** The mechanism already exists — `tests/plan-superseded-facts-test.sh` from the M1 round — so add these three token pairs to it rather than hand-editing prose that will rot the same way.

## 4. Minor findings

- `run_test.go:941` — `fakeRuntime.reportedUnused` is dead (never called), and `fakeRuntime.reported` is now written by nothing live. Delete both; the live recorder is `fakeMux.reported`.
- `run.go:166` — `runDecision`'s `stdin`/`stdout` parameters are unused, yet `stdout` is still threaded `runShell → pumpStdin → handleChord → runDecision`. That is the pane's raw descriptor travelling to the input goroutine — the exact shape M2 removed from the Runtime. Drop the parameters so a future writer there is a compile error, not a review catch.
- ARCH-CONSTRAINTS: the gate adds a **second** full parse of every child byte on the output hot path (`ptychild.Child` already feeds its own `Screen`), and the plan's envelope declares a budget only for paints. Undeclared, unmeasured; couch's precedent suggests it is fine, but say so in the envelope rather than leaving it implicit. Related: a long unterminated OSC keeps `MidSequence()` true until `maxPending`, so an owed paint can be stalled by a hostile/hung child — worth a line in ARCH-ORDER.

## 5. Test coverage notes

- Mutation results (overlay, repo untouched): `redrawTab` direct-write → RED; gate removed → RED; own-bytes fed to gate → RED; `cmd.Stderr = os.Stderr` → RED; `m.owed = nil` deleted → **GREEN**; `cmd.Stdout = os.Stdout` → **GREEN** (session-less environment).
- `go test ./cmd/internal/termcmd/ -count=1 -race` passes outside the sandbox (inside it, the two `TerminalMux…` pty tests fail on `operation not permitted` — environmental, not this diff). The rest of `./cmd/...` fails only on the same pty/exec sandbox restriction.
- No test exercises `removeTab` with the writer loop running, which is why C1 shipped green. The scratch repro above is a ready template: two tabs, a `Runtime` fake that blocks in `RunZellijAction`, fill the buffer, release, assert the loop drains within a timeout.
- No test covers `terminalMux.reportError` itself — every reporting test uses `fakeMux`, so the real path (gate → coalescing slot) is unobserved.
- BR-9 stands: `TestPaintDefersMidSequenceAndIsOwed` still splits one hand-chosen index. Now that the gate exists this is a five-line table over every index of a representative sequence set (CSI, OSC+ST, two-byte ST, a wide-rune run).

## 6. Architectural notes for upcoming work

- **ARCH-DRY** — flag (I3): the `writeOwn`/`flushOwed`/takeover trio is couch's shape with one divergence (bytes vs `paintPending` bool) and that divergence is where the loss bug lives. The plan's decision not to lift yet is sound; when M3 gives the second real caller, lift couch's *bool + re-render* policy, not the byte slot.
- **ARCH-PURE** — pass, with a pointer: `handleChunk` is a state machine whose transitions are interleaved with `m.stdout.Write`. A pure `step(state, event) -> (state, []effect)` with the caller performing effects would have made C1 unrepresentable — a handler that returns effects cannot post to its own queue. Worth doing before M3 adds resize-repaint and `RowDirty` to the same switch.
- **ARCH-PURPOSE** — pass. All six writers from finding 8 are accounted for, and the subprocess fix went to the class.
- **ARCH-MOCK** — flag: the zellij FD test drives the real binary; behind `Runtime` there is already a fake, but `runZellij` sits below that seam. Consider asserting the wiring with a stub command (`/bin/sh -c 'echo out; echo err >&2'`) so the test is deterministic and machine-independent.
- **ARCH-CONSTRAINTS** — flag (Minor, above).
- **ARCH-SECURE** — pass: no credentials; the untrusted child stream goes through `ptychild`'s existing `maxPending`-bounded parser, and the diff does not extend it. One residual: `reportError`'s unsanitized `err.Error()`.
- **ARCH-ORDER** — flag (C1). The plan's table lists the events but not "an event handler generating a new event", which is the interleaving that bit. When M3 adds repaint-on-resize, `chunk.onWriter` becomes a second self-send site — fix the shape now, not the site.

## 7. Plan revision recommendations

- `## Revisions` entry: **"M2 landed; three steps deviate from their text."** Record that (a) the subprocess envelope was closed by redefining `OSRuntime.RunZellijAction` rather than editing the five call sites, and why that is the stronger form; (b) `runZellij` **discards** rather than captures-and-logs, or change the code to match the plan; (c) startup `term:` diagnostics stay on stderr because the loop does not exist yet. Add the corresponding token pairs to `tests/plan-superseded-facts-test.sh` so the next deviation is caught rather than remembered.
- `## Revisions` entry for **C1**, naming the rule for M3: nothing running on the writer goroutine may call `enqueue`; the takeover has an inline form and a queued form sharing one implementation. M3's repaint-on-resize path must be written against that rule.
- Amend M2's ARCH-ORDER paragraph: "resize and paint serialize by construction on that loop" is true, but "no event handler generates an event" was the unstated premise that failed.

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      Verified by mutation, not by the commit message: restoring cmd.Stderr = os.Stderr reddens all four FD subtests; resize now goes through resizeThroughWriter.
  - id: BR-8
    disposition: not-addressed
    note: |
      M4.3 is unchanged in this window; still no Alt+Shift+d step, and the Done-when still requires two halves each drawing a strip.
  - id: BR-9
    disposition: not-addressed
    note: |
      TestPaintDefersMidSequenceAndIsOwed still splits one hand-chosen index; the gate now exists, so parameterizing is cheap.
findings:
  - id: new
    severity: Critical
    family: handler-posts-to-own-queue
    title: |
      removeTab runs on the writer goroutine and posts to the writer goroutine's own channel, deadlocking the pane
    detail: |
      run.go:758 dispatches removeTab on the writer goroutine; removeTab calls setPaneTitle (a blocking zellij subprocess) and then redrawTab (run.go:1081), which since M2 is enqueue -> m.output <- chunk. With the 64-slot buffer full from a second tab's output, the writer blocks sending to the channel only it drains, and enqueue's only escape (m.done) never closes. Reproduced with an overlay test: two tabs, close one while the other floods, writer parked permanently and all pane output stops. Fix: apply the takeover inline from removeTab via a helper shared with the takeover case; add the two-tab close-under-load regression test.
  - id: new
    severity: Important
    family: uncovered-negative-assertion
    title: |
      Two assert-absent tests pass vacuously — the owed-paint drop and the subprocess stdout arm are unpinned
    detail: |
      This is the 2nd finding in family uncovered-negative-assertion, so fix the rule, not the two sites. Measured by overlay mutation: deleting m.owed = nil (run.go:766) leaves the entire M2 suite green because writer_test.go:197 never feeds the boundary-ending chunk that would flush it; cmd.Stdout = os.Stdout also stays green because with no live zellij session the succeeding verb writes nothing to stdout, and the positive control built at writer_test.go:227-228 is never asserted. The rule: an assert-absent is evidence only when the same test first establishes the write would have happened. Sweep the four sites in writer_test.go plus run_test.go:910.
  - id: new
    severity: Important
    family: diagnostic-silently-discarded
    title: |
      The milestone keeps diagnostics off the pane by destroying them — discarded stderr, and a coalescing slot shared with paints
    detail: |
      run.go:1331 passes io.Discard for the subprocess stderr where the plan specifies capture-and-log, so a failing zellij action reduces to bare exit status, including on the non-pane --test-shortcut path that used to print it. run.go:833 routes reportError through the same single owed slot as paints (run.go:809), where a later paint replaces it, redrawTab drops it (run.go:766), and nothing flushes it if the child stops writing. couch avoids this by owing a paintPending bool re-derived by paintNow rather than bytes (ARCH-DRY). M3's per-repaint paint makes the clobber routine.
  - id: new
    severity: Important
    family: plan-table-drift
    title: |
      M2.3 steps 3, 4 and 5 describe an implementation the code does not have, with no Revisions entry
    detail: |
      This is the 3rd finding in family plan-table-drift, so state the rule rather than patching three lines. Piece 3 names five call sites to reroute that are unchanged (the Runtime was changed instead); piece 5 specifies capture-and-log stderr that is discarded; piece 4 states no exception for the startup term: diagnostics that still write stderr directly. The rule: ticking a step asserts the code matches its text, and a deviation gets a Revisions delta in the same commit — mechanized by adding these token pairs to tests/plan-superseded-facts-test.sh, which already exists for exactly this failure.
  - id: new
    severity: Minor
    family: dead-test-scaffolding
    title: |
      fakeRuntime.reportedUnused and its reported field are unreachable after the interface change
    detail: |
      run_test.go:941 renames the removed interface method instead of deleting it; nothing calls it and nothing reads fakeRuntime.reported. The live recorder is fakeMux.reported.
  - id: new
    severity: Minor
    family: handler-posts-to-own-queue
    title: |
      runDecision still receives the pane's stdout, unused, on the input goroutine
    detail: |
      run.go:166 takes stdin and stdout and uses neither; stdout is threaded from runShell through pumpStdin and handleChord. Removing the parameters makes a future write from the input goroutine a compile error rather than a review catch — the same argument the diff makes for the Runtime.
  - id: new
    severity: Minor
    family: hot-path-cost-undeclared
    title: |
      The gate adds a second full parse of every child byte on the output path, undeclared in the envelope
    detail: |
      handleChunk calls hostScan.FeedFraming on every chunk while ptychild.Child already parses the same bytes into its own Screen. ARCH-CONSTRAINTS budgets paints but not this. Also worth stating there: a long unterminated OSC holds MidSequence true up to maxPending, stalling any owed paint.
```

---

## Re-review — 2026-09-07T22:12:24-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 199 — pair term: own the right pane tab bar |
| repo | pair |
| issue file | workshop/issues/000199-pair-term-own-the-right-pane-tab-bar.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 96f780da661b35aef301b612b14ee8399bd9a6cf..34918a3c4524f4ed84a5d5c238c688c28b3524ad |
| command | sdlc milestone-close --issue 199 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-07T22:12:24-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The round disposed BR-26/27/28 and landed a real improvement — diagnostics now queue separately from paints and survive a takeover, and I confirmed both halves by mutation (routing `diag` through `writeOwn` and dropping `pendingDiag` each redden `TestDiagnosticsAreQueuedNotCoalescedAndSurviveATakeover`). But **BR-25, the Critical, was not touched**: `removeTab` still runs on the writer goroutine (`run.go:767`) and still posts to that goroutine's own channel (`run.go:1120` → `redrawTab` → `enqueue`), and I reproduced the permanent deadlock with an overlay test (two tabs, 64 slots filled, writer parked past a 3s timeout). Alt+W while another tab floods freezes the pane for good. Behind it, BR-26's *rule* — an assert-absent is evidence only when the same test establishes the write would have happened — was answered at two sites rather than as a class: I measured **five** deliverables in this window that no test fails without (`m.owed = nil`, `cmd.Stdout = stdout`, `runZellijCaptured`, `resizeThroughWriter`, plus a brand-new guard line in `plan-superseded-facts-test.sh` that can never fire). Two new findings: the BR-27 fix routes untrusted subprocess bytes onto the pane's tty unsanitized, and the atlas paragraph added at `9b111c64` was falsified by `34918a3c` in the same window.

## 1. Strengths

- **The diagnostic/paint policy split is genuinely pinned, not asserted.** Mutation-verified two ways: `case chunk.diag: m.writeOwn(...)` → RED with `"first failure" was coalesced away`; blanking `pendingDiag` in the takeover branch → RED with `a takeover swallowed a diagnostic`. This is the half of BR-27 that is really closed.
- **`-race` is clean** — `go test ./cmd/internal/termcmd/ -count=1 -race` → `ok 1.479s` outside the sandbox (inside it the two `ptychild.Start` tests fail on `operation not permitted`, environmental). M2.4 verified rather than claimed.
- **The stderr FD assertion is strongly mutation-sensitive.** `cmd.Stderr = os.Stderr` reddens all four subtests with byte counts (`28` / `167`, quoting zellij's actual output). BR-4's third raise stays disposed on the code side.
- **Removing `ReportShortcutError` from `Runtime` is the class-level fix.** I checked the whole tree: no live consumer outside `termcmd` (only archived `workshop/history/` plans mention it), so the interface narrowing is contained and the input goroutine now has no route to the pane's fd except `mux.reportError`.
- **The superseded-facts guard did real work.** Restoring the old piece-3 wording into the plan body reddens it at `plan:515` — mutation-confirmed, and it is the first time this repo's `plan-table-drift` class was caught by a test.
- **`enqueue`'s non-waiting contract is load-bearing and correctly documented** (`run.go:880-897`): `runShell` redraws via `newTab()` before `copyActiveOutput()` exists, so a synchronous post really would hang startup. The `output == nil` inline branch is a sound rather than lazy fallback.
- **The M2.5 manual acceptance is honest evidence** (`issue:194-207`): idle run explicitly labelled as proving little, flood run as the one that exercises the gate, PID/start-time checked against the build.

## 2. Critical findings

**BR-25 (re-raised, `not-addressed`) — `removeTab` still self-sends; reproduced this round.**
`run.go:767` dispatches `removeTab` on the writer goroutine → `run.go:1118` `setPaneTitle` (blocking `zellij` subprocess) → `run.go:1120` `redrawTab` → `run.go:1281` `enqueue` → `m.output <- chunk`. Nothing changed in this window.

Reproduced with an overlay test (repo untouched), a `Runtime` whose `RunZellijAction` blocks:
```
filled 64 slots
DEADLOCK: writer goroutine blocked posting to its own channel
--- FAIL: TestReproRemoveTabDeadlockBR25 (3.00s)
```
Production trigger: `closeActive` (Alt+W, `run.go:970`) kills the child → the pump posts `err: io.EOF` → the writer parks in `rename-pane` while the other tab's Sink (`run.go:707`, an unguarded `m.output <-`) fills the 64-slot buffer → the send blocks with no consumer, and `enqueue`'s only escape (`<-m.done`) never closes because `empty == false`. All pane output stops, not just the strip.

Fix sketch unchanged: split the takeover into an inline `applyTakeover(body)` used by both `removeTab` and `handleChunk`'s `takeover` case, so there is one implementation; move `setPaneTitle`'s subprocess off the writer loop. Regression test: two tabs, one flooding, close the other, assert the loop drains within a timeout.

## 3. Important findings

**BR-26 (re-raised, `not-addressed`) — the rule was applied to two sites; the class has five members, and one is new this commit.**
Measured by overlay mutation against the full package (baseline failures are the two sandbox pty tests only):

| mutation | expected | measured |
|---|---|---|
| delete `m.owed = nil` (`run.go:775`) | RED | **GREEN** |
| `cmd.Stdout = os.Stdout` (`run.go:1381`) | RED | **GREEN** (zellij *is* installed here) |
| `runZellijCaptured` → `runZellij(cmdArgs, io.Discard, io.Discard)` | RED | **GREEN** |
| `resizeThroughWriter` → `inheritSize` (`run.go:263`) | RED | **GREEN** |
| restore `route only stdout through` token | RED | **cannot fire** |

The positive control added to `TestTakeoverResetsTheGateAndDropsTheOwedPaint` (`writer_test.go:202`) proves the *writer* works; it does not establish that `STALE` would have landed, which requires feeding a boundary-ending child chunk **after** the takeover. Same shape in the FD test: the control at `writer_test.go:310-311` is a direct `fmt.Fprint`, not the subprocess, and the arms named `Action/succeeds` do not succeed — with `cmd.Stderr` mutated they print `There is no active session!`, i.e. the stdout arm has no live case anywhere. And `tests/plan-superseded-facts-test.sh:65` bans `'route only stdout through'`, a string that has never appeared in the plan file at any commit (`git log -S` finds it only in the M1 review document) — a guard added as evidence of a fix that can never fire.

The class statement this needs: **a deliverable is landed only when a test fails without it, and an assert-absent is evidence only when the same test drives the event that would have produced the presence.** The enumeration to sweep is the five rows above plus `writer_test.go:118` (already has a control) and `run_test.go:910`. For the FD test specifically, ARCH-MOCK gives the deterministic form: drive the wiring through a stub (`/bin/sh -c 'echo out; echo err >&2'`) so both descriptors have a live case independent of whether a zellij session exists on the machine.

**BR-27 (re-raised, `not-addressed`) — the coalescing half is closed; the discard half changed plumbing without changing what the operator sees.**
`runZellijCaptured` (`run.go:1396-1414`) is correct and better than `io.Discard`. But (a) no test fails if it is reverted to the exact `io.Discard` form the finding named — measured GREEN above; and (b) the two call classes the fix's own rationale cites still throw the error away: `run.go:461` `_ = rt.RunZellijAction("scroll-up")`, `run.go:467` `scroll-down`, and `run.go:1118` / `run.go:1210` `_ = m.setPaneTitle(title)`. So a failing wheel tick and a failing `rename-pane` remain *completely silent* — the condition the commit message calls "strictly worse than the noise it replaced". The mechanism was built at the Runtime; the sites that motivated it were not wired to it. That is instance-vs-class (ARCH-PURPOSE) at the level above where the last round applied it.

**N1 (new) — the BR-27 fix opens a path from an external process's bytes to the pane's terminal, unsanitized and unbounded.**
`runZellijCaptured` folds the subprocess's stderr (or, `run.go:1408`, its stdout) into the error; `reportError` (`run.go:872`) wraps `err.Error()` in `"pair term: " + … + "\r\n"` and writes it to `m.stdout`, which is a live pty. Reachable today: `handleChord` → `runDecision` → `RunZellijAction("focus-pane-id", …)` → `mux.reportError` (`run.go:441`), and `splitTerminalDown` → `RunZellijActionQuiet` (`run.go:534`) → `mux.reportError` (`run.go:511`). Before this commit the error was a bare `*exec.ExitError`, so no external bytes reached the pane; now they do. Only the newline is bounded (`run.go:1411`) — length is not, and escape/control bytes are not stripped. The plan's own ARCH-SECURE already commits to the rule for the strip ("a name containing `\x1b` must not become an escape in our row"), and `setPaneTitle`'s argument is an operator-controlled tab name that zellij's error text can echo back. I checked current zellij output on a pipe and saw no escape bytes (clap disables colour for a non-tty), so this is latent, not observed — but it is a new untrusted-input edge introduced without a parse step, on the surface M4 removes the frame from.
The rule (family `external-input-assumed-wellformed`, 2nd): **bytes this process did not produce become safe row text at the boundary where they enter, not at the writer that prints them.** Enumerate now: `runZellijCaptured`'s `detail`, `reportError`'s `err.Error()`, `run.go:66`'s `fmt.Fprintf(stderr, "term: %v\n", err)`, and M3's tab names — all four want the one `rowtext.Sanitize`/`Fit` the plan already specifies. Either land `rowtext` at this boundary or sanitize+cap `detail` inside `runZellijCaptured`.

**N2 (new) — the atlas paragraph landed in this window was falsified by the next commit in the same window; no update.**
`atlas/architecture.md:501-503`: "consulted before **any console-originated write**; a write issued mid-sequence is deferred into a **single coalescing slot**", and :506-508 "a wholesale takeover … **drops** the owed paint". The paragraph's own first sentence lists diagnostics among the writes going through the loop. After `34918a3c` that is false for exactly that class: diagnostics go to `owedDiag` (a queue, `run.go:818`) and *survive* a takeover (`run.go:779-786`). A reader — including M3 — takes from the atlas the opposite of the decision this milestone was about. Line :519 has the matching gap: "a FAILING action did the same on stderr, once per wheel tick" describes the old behaviour and never says where those failures go now.
**This is the 4th finding in family `plan-table-drift`.** Earlier rounds fixed instances and then mechanized the plan file. The rule that covers all four: **an artifact that asserts a design — a plan step, an atlas paragraph — is re-checked against the code in the commit that changes the design, and the check is a test, not a memory.** The mechanism exists but is scoped to one plan file and to hand-registered tokens; the class fix is to extend `tests/plan-superseded-facts-test.sh` to `atlas/` and register `single **coalescing** slot` → `paints coalesce, diagnostics queue` in the same commit that corrects the prose. Note also that the second token pair added this round is inert (see BR-26), so registering a pair is not by itself evidence.

## 4. Minor findings

- **BR-29 `not-addressed`** — `run_test.go:941` `fakeRuntime.reportedUnused` still present, still uncalled; `fakeRuntime.reported` still written by nothing live.
- **BR-30 `not-addressed`** — `run.go:167` `runDecision(…, stdin, stdout)` still takes both and uses neither; `stdout` still threaded `runShell → pumpStdin → handleChord`.
- **BR-31 `not-addressed`** — ARCH-CONSTRAINTS unchanged; the second full parse of every child byte (`hostScan.FeedFraming`, `run.go:800`) is still undeclared, as is the unterminated-OSC stall of an owed paint.
- **BR-8 / BR-9 `not-addressed`** — M4.3 untouched in this window (still no Alt+Shift+d step against a Done-when requiring two halves); `TestPaintDefersMidSequenceAndIsOwed` (`writer_test.go:115`) still splits one hand-chosen index.
- **N3 (new)** — `owedDiag` is an unbounded queue (`run.go:818`) filled by external timing (one entry per failed `refreshRename` keystroke while the child sits mid-sequence) and flushed all at once, sitting one function below a comment (`run.go:796-799`) that rejects `Feed` precisely because it would be "an unbounded buffer growing behind a terminal that never reads it".
- ARCH-DRY nit: the diagnostic drain loop is written twice (`run.go:783-785` and `run.go:848-851`) with different flush order relative to the paint; one `drainDiag()` would state the policy once.

## 5. Test coverage notes

- All mutations above were run as `go test -overlay` against a scratch copy; the repo tree was never modified. Baseline package failures are exactly `TestTerminalMuxNewTabClearsPreviousTabViewport` and `TestTerminalMuxNewTabPrintsStartupOutputOnce` (`ptychild: start /bin/sh: operation not permitted`, sandbox-only — clean outside).
- Still no test exercises `removeTab` with the writer loop running, which is why BR-25 shipped green twice. The overlay repro is a ready template.
- Still no test covers `terminalMux.reportError` end-to-end through a real `Runtime` failure, `resizeThroughWriter`, or `runZellijCaptured`'s error folding.
- The `Action/succeeds` and `Quiet/succeeds` subtests are misnamed on this machine: the verb fails with `There is no active session!`. Any assertion resting on "a succeeding action writes stdout" is ambient-state-dependent.

## 6. Architectural notes for upcoming work

- **ARCH-ORDER — flag (BR-25).** The plan's event table still omits "an event handler generating a new event", which is the interleaving that bites. `resizeThroughWriter` adds a second `enqueue`-from-elsewhere site and M3 adds repaint-on-`RowDirty`; write the rule down before M3: *nothing running on the writer goroutine may call `enqueue`; every takeover has an inline form and a queued form sharing one implementation.*
- **ARCH-PURE — flag.** `handleChunk` is a state machine whose transitions are interleaved with `m.stdout.Write`, and it now carries six cases and two deferral slots. A pure `step(state, event) -> (state, []effect)` with the caller performing effects makes BR-25 unrepresentable — a handler that returns effects cannot post to its own queue. This is the last cheap moment to do it.
- **ARCH-PURPOSE — flag (BR-27).** The Runtime-level fix is right; it is undelivered until a caller reads the error it now carries.
- **ARCH-SECURE — flag (N1).** New untrusted-input edge, no parse at the boundary.
- **ARCH-MOCK — flag.** `TestNeitherZellijMethodHandsTheSubprocessThePanesDescriptors` drives the real `zellij` binary below the `Runtime` seam, so its coverage varies by machine — demonstrated: the stdout arm is vacuous here.
- **ARCH-CONSTRAINTS — flag (BR-31, N3).** Two undeclared costs and one unbounded buffer.
- **ARCH-DRY — pass with the nit above.** The decision not to lift couch's defer/owe policy until M3 gives a second real caller remains correct.

## 7. Plan revision recommendations

- `## Revisions` entry for **BR-25**, stating the rule for M3 as above, and amending M2's ARCH-ORDER paragraph: "resize and paint serialize by construction on that loop" holds, but its unstated premise — *no event handler generates an event* — is false at `run.go:767`.
- `## Revisions` entry for **BR-26 as a class**: record the five measured unpinned deliverables and the inert guard line, and state that a `check` pair is registered only with a demonstration that it fires (the M1 entry already set that precedent — "Mutation-verified: restoring … turns it red"). Delete or correct `tests/plan-superseded-facts-test.sh:65`.
- **ARCH-SECURE** paragraph: extend beyond tab names to the diagnostic path — subprocess stdout/stderr folded into an error is untrusted input reaching the same row, and needs the same `rowtext.Sanitize`/`Fit` plus a length cap.
- **ARCH-CONSTRAINTS** paragraph: declare the second parse of every child byte, the unterminated-OSC stall bound, and a cap on `owedDiag`.

```findings
dispose:
  - id: BR-8
    disposition: not-addressed
    note: |
      M4 is untouched in this window; still no Alt+Shift+d step and the Done-when still requires two right-pane halves each drawing a strip.
  - id: BR-9
    disposition: not-addressed
    note: |
      TestPaintDefersMidSequenceAndIsOwed (writer_test.go:115) still splits one hand-chosen index.
  - id: BR-25
    disposition: not-addressed
    note: |
      No code change; reproduced again with an overlay test - 64 slots filled, writer parked past a 3s timeout. run.go:767 -> 1120 -> 1281 -> enqueue.
  - id: BR-26
    disposition: not-addressed
    note: |
      Two sites got controls; the class has five. Measured GREEN under mutation: m.owed = nil, cmd.Stdout = stdout, runZellijCaptured, resizeThroughWriter; plus plan-superseded-facts-test.sh:65 bans a token that never existed in the plan.
  - id: BR-27
    disposition: not-addressed
    note: |
      Coalescing half closed and mutation-pinned. Discard half: reverting runZellijCaptured to io.Discard leaves the suite green, and run.go:461,467,1118,1210 still drop the error, so a failing wheel tick or rename is still completely silent.
  - id: BR-28
    disposition: addressed
    note: |
      M2.3 pieces rewritten to match the code, Revisions delta present, and the piece-3 guard is mutation-live (restoring the wording reddens plan:515). The second guard pair is inert - folded into BR-26.
  - id: BR-29
    disposition: not-addressed
    note: |
      run_test.go:941 fakeRuntime.reportedUnused still present and still uncalled.
  - id: BR-30
    disposition: not-addressed
    note: |
      run.go:167 runDecision still takes stdin and stdout and uses neither.
  - id: BR-31
    disposition: not-addressed
    note: |
      ARCH-CONSTRAINTS unchanged; the second parse and the unterminated-OSC stall are still undeclared.
findings:
  - id: new
    severity: Important
    family: external-input-assumed-wellformed
    title: |
      The BR-27 fix routes an external process's bytes onto the pane's tty, unsanitized and unbounded
    detail: |
      This is the 2nd finding in family external-input-assumed-wellformed, so the deliverable is the rule, not the site. runZellijCaptured (run.go:1396-1414) folds the subprocess's stderr - or its stdout at run.go:1408 - into the error; reportError (run.go:872) writes err.Error() straight to the pane's pty. Reachable via handleChord -> runDecision -> RunZellijAction (reported at run.go:441) and splitTerminalDown (run.go:511). Before this commit the error was a bare exec.ExitError, so no external bytes reached the pane. Only the newline is bounded (run.go:1411); length and escape bytes are not, and setPaneTitle's argument is an operator-controlled tab name zellij's error text can echo back. Measured: current zellij on a pipe emits no escapes, so this is latent rather than observed. The rule - bytes this process did not produce become safe row text at the boundary where they enter, not at the writer that prints them - covers four enumerable sites: runZellijCaptured's detail, reportError's err.Error(), run.go:66's term: %v, and M3's tab names, all of which the plan already routes through rowtext.Sanitize/Fit.
  - id: new
    severity: Important
    family: plan-table-drift
    title: |
      The atlas paragraph added in this window was falsified by the next commit in the same window
    detail: |
      This is the 4th finding in family plan-table-drift, so state the rule rather than patching the paragraph. atlas/architecture.md:501-503 says the gate is consulted before "any console-originated write" and that such a write "is deferred into a single coalescing slot", and :506-508 that a takeover drops it - false for diagnostics since 34918a3c, which gave them a queue (run.go:818) that survives a takeover (run.go:779-786). The paragraph's own first sentence lists diagnostics among the writes on the loop, so a reader takes from the atlas the opposite of the decision this milestone was about; :519 has the matching gap for where a failing action's stderr goes now. The rule: an artifact asserting a design is re-checked against the code in the commit that changes the design, and the check is a test. The existing mechanism is scoped to one plan file and to hand-registered tokens - extend tests/plan-superseded-facts-test.sh to atlas/ and register this pair, noting that registering a pair is not evidence unless it fires (see BR-26).
  - id: new
    severity: Minor
    family: hot-path-cost-undeclared
    title: |
      owedDiag is an unbounded queue on an externally-timed path, one function below a comment rejecting Feed for that reason
    detail: |
      This is the 2nd finding in family hot-path-cost-undeclared, so the rule: every buffer introduced on a path whose fill rate is set by something outside the process gets a declared bound in ARCH-CONSTRAINTS and a cap in code. run.go:818 appends without limit - one entry per failed refreshRename keystroke while the child sits mid-sequence - and flushes them all at once at the next boundary, while run.go:796-799 explains that Feed was rejected precisely because it would be an unbounded buffer growing behind a terminal that never reads it. Same envelope gap as the undeclared second parse.
```

---

## Re-review — 2026-09-07T22:34:06-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 199 — pair term: own the right pane tab bar |
| repo | pair |
| issue file | workshop/issues/000199-pair-term-own-the-right-pane-tab-bar.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 96f780da661b35aef301b612b14ee8399bd9a6cf..e9a988cec2f2a5afbe5c1aa4d8e5e6c92883275f |
| command | sdlc milestone-close --issue 199 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-07T22:34:06-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The window's shape is right and much of it is genuinely pinned — I re-verified by mutation that `redrawTab` writing directly reddens the one-writer test, that removing the `MidSequence()` check reddens the defer/owe test, that feeding our own bytes into `hostScan` reddens `TestGateIsNotFedOurOwnWrites`, that `cmd.Stderr = os.Stderr` reddens the FD test, and that BR-32's new egress sanitizer is real (deleting `rowtext.SanitizeAndFit` at `run.go:878` fails `TestAZellijErrorCannotPutAnEscapeOrAnUnboundedLineOnThePane`). What blocks the boundary is that the gate itself models the wrong stream: `handleChunk` feeds `hostScan` from **every** tab including ones whose bytes never reach the terminal, and resets it to zero on a takeover whose replay it then writes **without** scanning. I reproduced both directions — `"visible\x1b[3PAINT"` and `"\x1b[1;1H\x1b[Jrestored screen\x1b[3PAINT"` — a paint landing inside a live CSI, which is the single failure M2 exists to prevent, reachable with the two-tab workflow the issue is premised on. BR-25 (Critical, third round) is still live and I reproduced the deadlock again; BR-26/27/29/30/31/33/34 are untouched since round 7, which only landed the BR-32 fix.

## 1. Strengths

- **`rowtext` is a real extraction, not a copy** (`cmd/internal/rowtext/rowtext.go`). `couchtty`'s unexported `sanitize`/`truncate` are *deleted*, both consumers repointed, and the whole `couchtty` suite passes with **no couchtty test edited** — the same "prove it was a move" discipline M1 used. The package now has tests of its own where neither original did.
- **The egress choice is the right one and it is mutation-pinned.** Sanitizing at `reportError` (`run.go:878`) rather than at each producer is the correct answer to BR-32, and the Revisions entry argues it from `#208`'s `ps` precedent rather than asserting it. Removing it fails a test.
- **The subprocess envelope closed at the `Runtime`, not the call sites** (`run.go:1360-1372`), which covers `layoutcmd`/`draftroute` and the next site anyone adds.
- **`drainForTest`'s bare-event design** (`run.go:941`) and the `chunk.data == nil` branch: a probe that does not perturb what it observes, with the earlier self-inflicted failure recorded in the comment.
- **`TestTakeoverResetsTheGateAndDropsTheOwedPaint` gained a real positive control** (`writer_test.go:202`) — the takeover's own replay must arrive before absence means anything.

## 2. Critical findings

**C1 — the gate is fed bytes the terminal never saw, and not fed bytes it did.** `cmd/internal/termcmd/run.go:804` and `:775`.

`handleChunk`'s default branch scans **every** chunk, but writes only the active tab's:

```go
m.hostScan.FeedFraming(chunk.data)   // :804 — every tab
if m.isActive(chunk.id) {            // :808 — only the active one
    _, _ = m.stdout.Write(chunk.data)
}
```

Every tab's `Sink` posts to `m.output` unconditionally (`run.go:707`), so a background `yes`/build/`tail -f` drives the gate for a stream it is not part of. Reproduced with an overlay test, two tabs, tab 1 active:

```
a background tab's byte released the gate; the paint landed inside the
FOREGROUND child's escape sequence: "visible\x1b[3PAINT"
```

and the mirror — a background tab's unterminated OSC defers a paint the foreground stream was ready for.

The same rule is broken in the other direction on the takeover path. `run.go:775` resets `hostScan` to a zero `Screen{}` and then writes `chunk.own` — which contains the child's replay — without scanning it. `ptychild/replay.go:64-68` states in its own doc comment that a replay "can legitimately begin or end mid-sequence" because `Ring` bisects whatever spans its boundary. So after a takeover the gate reports *boundary* while the terminal is mid-CSI:

```
a paint landed inside the replay's bisected sequence:
"\x1b[1;1H\x1b[Jrestored screen\x1b[3PAINT"
```

Both are `PAINT` inside a live CSI (`\x1b[3P` is DCH — the terminal eats characters and prints `AINT`). Neither M2's tests nor M2.5's manual run could see it: every test uses one tab and `yes` emits no escapes.

**This is the 2nd finding in family `wrong-seam-named`** (BR-3 was the repaint trigger named at a seam that is always false in `termcmd`). Do not fix the two sites — state and apply the rule: **a predicate that guards a stream is computed from exactly the bytes that reach that stream, at the point they are written.** The enumeration that follows from it, all three of which the code gets wrong or will:

1. `run.go:804` — move `FeedFraming` inside the `isActive` branch, so scan and write are one statement pair.
2. `run.go:775-782` — the takeover must carry its child half separately from our `HomeAndClear` (e.g. `ptyChunk{own: HomeAndClear, data: replay, takeover: true}`) and feed the replay to the fresh scanner as it writes it. The gate is still "child bytes only": a replay *is* child bytes.
3. M3's `batch.RowDirty` read (plan finding 7) has the identical shape — a background `nvim`'s startup clear never touched the terminal and must not trigger a re-`Reserve`. Write the rule into the plan's M3.4 now, not after the next review finds it.

Regression tests: the two above, plus the takeover-replay one. All three are five-liners against the existing `writerRecorder`.

## 3. Important findings

**I1 — M2.5's acceptance command cannot exercise the thing it is recorded as accepting.** `plan:546`, issue `## Log` 2026-09-07.

The step is `yes "aaaa…"` flooding one tab while switching tabs, and the Log reasons that "a redraw issued while `yes` is mid-escape is precisely the interleaving M2 exists to make safe". `yes` emits no ESC byte at all, so `MidSequence()` is false for the entire run and the defer/owe path — the half of M2 that is new — never executes once. What the run did exercise is the single-writer envelope, which is real evidence for that half and should be recorded as such.

**This is the 3rd finding in family `acceptance-command-does-not-hold`** (BR-24, and the M1.6 grep before it). The rule: **an acceptance step names the observable that proves it entered the path under test, and the observation records that observable — not that the command ran.** For a gate that means a child which actually emits escape sequences (`vim`, `htop`, or a loop of `printf '\033[31m%s' …` split across writes) **and a second tab**, with the deferral visibly counted or logged. Sweep the remaining acceptance steps under the same rule: M2.5 (rewrite), M3.7 (`nvim`'s startup clear — names its observable, holds), M4.3 (BR-8's missing split — same defect, already open).

**I2 — `rowtext.Sanitize` passes the C1 controls, and it is now the repo's single security-relevant strip.** `cmd/internal/rowtext/rowtext.go:37`.

The rune pass drops `r < 0x20 || r == 0x7f`. Measured:

```
"a\u009b31mred"    -> "a\u009b31mred"
"a\u009d0;titlex"  -> "a\u009d0;titlex"
```

U+009B is CSI and U+009D is OSC in 8-bit form; a terminal decoding UTF-8 with C1 handling on takes those as sequence introducers, so a "sanitized" diagnostic can still open a control sequence. `ansi.Strip` does not frame them either. The behavior is inherited from `couchtty`, but the package doc and `atlas/architecture.md:532` now promote it to *the* implementation of "make untrusted text safe for a row", and M3 routes operator-typed tab names through it.

**This is the 3rd finding in family `external-input-assumed-wellformed`.** The rule, since the previous two were fixed as sites: **the sanitizer's notion of "control" is derived from what a terminal treats as a control — C0, DEL, C1 (U+0080–U+009F), and the sequence introducers in both 7- and 8-bit forms — and its test table enumerates those classes, not the ones a past incident happened to produce.** One predicate change plus a table row per class. Co-located nit for the same edit: `couchtty/menu_render.go:625` open-codes `rowtext.Fit(rowtext.Sanitize(line), width)` where `SanitizeAndFit` exists and is documented as "the pair, in the order that matters".

## 4. Minor findings

- `cmd/internal/couchtty/reserve.go:143-152` — the doc comments for `sanitize` and `truncate` survived the functions; the file now ends in two paragraphs describing symbols that are not in it. **This is the 3rd finding in family `atlas-points-at-old-home`** — rule: an extraction moves the prose with the symbol and leaves neither comment nor stub at the old home; the greppable form is "a doc comment whose subject identifier no longer resolves in its file".
- `terminalMux.stderr` is now written by `newTerminalMux` and read by nothing (production sibling of BR-29 — see the disposition note).

## 5. Test coverage notes

Mutation results at HEAD, overlay only, repo untouched (`-overlay`, no edits):

| mutation | result |
|---|---|
| `redrawTab` writes directly | RED |
| gate check removed from `writeOwn` | RED |
| own bytes fed to `hostScan` | RED |
| `cmd.Stderr = os.Stderr` | RED |
| `rowtext.SanitizeAndFit` removed from `reportError` (`:878`) | **RED** — BR-32 is really pinned |
| `m.owed = nil` deleted from the takeover (`:776`) | **GREEN** |
| `rowtext.SanitizeAndFit` removed from `runZellijCaptured` (`:1430`) | **GREEN** |
| `return err` instead of folding the detail (`:1434`) | **GREEN** |
| `inheritSize` off the writer loop (`:1156`) | **GREEN** |
| `cmd.Stdout = os.Stdout` (`:1388`) | RED **here**, GREEN in round 7 — ambient |

That last row is the ARCH-MOCK point made concrete: the test drives the real `zellij` below the `Runtime` seam, so its stdout arm is red on a machine inside a live session and vacuous everywhere else. Baseline `go test ./cmd/... -count=1` and `./cmd/internal/termcmd -race` are green outside the sandbox; inside it the two `TerminalMux…` pty tests fail on `operation not permitted` (environmental).

No test exercises `removeTab` with the writer loop running (BR-25), `terminalMux.reportError`'s real path outside the two new tests, `runZellijCaptured`'s capture, or `resizeThroughWriter`. The `TestPaintDefersMidSequenceAndIsOwed` split point is still one hand-chosen index (BR-9).

## 6. Architectural notes for upcoming work

- **ARCH-DRY** — pass, one nit. The `rowtext` extraction is the right shape (one implementation, tests, both consumers repointed). Nit: `clipMenuLine` recomposes the pair; dangling comments at the old home.
- **ARCH-PURE** — flag, and C1 is the bill. `handleChunk` is a state machine whose transitions interleave with `m.stdout.Write`, so "scan what you write" is a convention rather than a structure. A pure `step(state, event) -> (state, []effect)` with the caller performing effects makes both C1 halves and BR-25 unrepresentable: a handler that returns effects cannot desynchronise its scanner from its writes, and cannot post to its own queue. Do this **before** M3 adds resize-repaint and `RowDirty` to the same switch — that is three more transitions on a shape that has now produced two Criticals.
- **ARCH-PURPOSE** — flag. M2's purpose is "no paint lands inside a child's escape sequence"; with C1 open it is not delivered for the multi-tab case that is the issue's own premise. Separately, BR-27's capture landed but nothing reports it at the four dropping call sites (`:462, :468, :1125, :1217`), so "capture **and log**" is still the easy half.
- **ARCH-MOCK** — flag. The FD test drives the real binary; a stub command (`/bin/sh -c 'echo out; echo err >&2'`) makes it deterministic and machine-independent, and would have made the round-6/round-8 disagreement impossible.
- **ARCH-CONSTRAINTS** — flag (BR-31, BR-34 open). Worth noting the envelope gap is *larger* than BR-31 stated: the second parse is of **every tab's** output, not the active one's — which C1's fix incidentally repairs.
- **ARCH-SECURE** — flag (I2). The child stream itself is still behind `ptychild`'s `maxPending`-bounded parser and the diff does not extend it; no credentials anywhere.
- **ARCH-ORDER** — flag. BR-25 remains the canonical instance (a handler generating an event on its own queue). The takeover's `hostScan = Screen{}` is a second: a transition that *asserts* a terminal state which the very next write in the same branch contradicts. Both belong in the plan's ARCH-ORDER table, which today lists the events but not "an event handler emitting an event" or "our own write changes the stream the gate models".

## 7. Plan revision recommendations

- **`## Revisions` entry for C1**, naming the rule for M3: *the gate is fed exactly the bytes written to the terminal, at the point they are written.* Amend M2.3 piece 2 ("a `ptychild.Screen` fed child chunks before they are written") — it is the sentence the code implements literally and wrongly; it must say **active-tab** chunks, and must say the takeover's replay is fed as it is written. Amend M3.4 in the same edit: `batch.RowDirty` is read only for the active tab.
- **`## Revisions` entry for BR-25**, unchanged from round 6's recommendation and still not written: nothing running on the writer goroutine may call `enqueue`; the takeover needs an inline form and a queued form sharing one implementation. M3's repaint-on-resize is written against that rule or it reproduces it.
- **Rewrite M2.5** per I1, and un-tick it. Ticking it asserts the gate was accepted; the command named cannot reach the gate.
- **M2.6 is ticked** (`plan:547`) for a `milestone-close` that has not passed — three rounds now. Either untick it or add "ticked at commit, close pending" so the plan stops asserting a boundary that has not been crossed.
- **ARCH-CONSTRAINTS**: add the second-parse cost (per tab, not per pane), the `owedDiag` bound (BR-34), and the unterminated-OSC stall (BR-31), with a cap in code for the queue.
- **`tests/plan-superseded-facts-test.sh:65`** registers `'route only stdout through'`, a phrase that appears in no revision of the plan — I checked every historical blob. An inert guard reads as protection and is not; drop it or replace it with the token the plan actually carried.

```findings
dispose:
  - id: BR-8
    disposition: not-addressed
    note: |
      M4 untouched in this window; M4.3 still has no Alt+Shift+d step while the Done-when still requires two right-pane halves each drawing a strip.
  - id: BR-9
    disposition: not-addressed
    note: |
      writer_test.go:115 still splits one hand-chosen index of one sequence.
  - id: BR-25
    disposition: not-addressed
    note: |
      No code change. Reproduced again at HEAD with an overlay test - 64 slots filled, writer parked past a 3s timeout. run.go:768 -> removeTab:1127 -> redrawTab:1287 -> enqueue:905.
  - id: BR-26
    disposition: not-addressed
    note: |
      Four sites still GREEN under mutation at HEAD - m.owed = nil (run.go:776), runZellijCaptured's detail (:1430 and :1434), resizeThroughWriter (:1156). The stdout arm flipped RED here only because this machine has a live zellij session, which is the ambient-dependence the finding named. plan-superseded-facts-test.sh:65 still registers a token absent from every historical revision of the plan.
  - id: BR-27
    disposition: not-addressed
    note: |
      Capture half landed but is unpinned - returning the bare error leaves the suite green. Log half still absent - run.go:462,468,1125,1217 discard the error, so a failing wheel tick or rename is still completely silent.
  - id: BR-29
    disposition: not-addressed
    note: |
      run_test.go:941 fakeRuntime.reportedUnused still present and uncalled. Production sibling of the same class - terminalMux.stderr is set by newTerminalMux and read by nothing after the reportError change.
  - id: BR-30
    disposition: not-addressed
    note: |
      run.go:168 runDecision still takes stdin and stdout and uses neither.
  - id: BR-31
    disposition: not-addressed
    note: |
      ARCH-CONSTRAINTS unchanged. The gap is wider than stated - the second parse covers every tab's output, not the active tab's.
  - id: BR-32
    disposition: addressed
    note: |
      Mutation-verified rather than taken from the commit - removing rowtext.SanitizeAndFit at run.go:878 fails TestAZellijErrorCannotPutAnEscapeOrAnUnboundedLineOnThePane. The duplicate strip inside runZellijCaptured is unpinned, but the egress covers the pane path.
  - id: BR-33
    disposition: not-addressed
    note: |
      atlas/architecture.md:501-508 still says a console-originated write is deferred into a single coalescing slot and dropped by a takeover, which is false for diagnostics since 34918a3c. The head commit added a further inaccuracy to the same block - :532 states pair term's strip as a current rowtext consumer, which M3 has not built.
  - id: BR-34
    disposition: not-addressed
    note: |
      run.go:819 still appends to owedDiag without a bound, and ARCH-CONSTRAINTS still declares no budget for it.
findings:
  - id: new
    severity: Critical
    family: wrong-seam-named
    title: |
      The paint gate is fed bytes the terminal never saw, and not fed bytes it did
    detail: |
      This is the 2nd finding in family wrong-seam-named, so the deliverable is the rule, not the two sites. run.go:804 calls hostScan.FeedFraming on EVERY chunk while run.go:808 writes only the active tab's, so a background tab releases or stalls the gate for the foreground stream; run.go:775 resets the scanner to a zero Screen and then writes the takeover's replay unscanned, and ptychild/replay.go:64-68 documents that a replay can end mid-sequence because Ring bisects it. Both reproduced with overlay tests - "visible\x1b[3PAINT" and "\x1b[1;1H\x1b[Jrestored screen\x1b[3PAINT", a paint inside a live CSI, which is the one failure M2 exists to prevent. Unreachable by the existing suite (one tab) and by M2.5 (yes emits no ESC). The rule - a predicate that guards a stream is computed from exactly the bytes that reach that stream, at the point they are written - covers three enumerable sites - FeedFraming moved inside the isActive branch; the takeover carrying its child half separately and feeding it as it writes it; and M3's batch.RowDirty read, which must likewise be scoped to the active tab because a background child's clear never touched the terminal.
  - id: new
    severity: Important
    family: acceptance-command-does-not-hold
    title: |
      M2.5's manual acceptance cannot enter the gate path it is recorded as accepting
    detail: |
      This is the 3rd finding in family acceptance-command-does-not-hold, so state the rule rather than rewriting one step. plan:546 specifies `yes "aaaa…"` flooding one tab; yes emits no ESC byte, so MidSequence is false for the whole run and the defer/owe path never executes, yet the issue Log reasons that "a redraw issued while yes is mid-escape is precisely the interleaving M2 exists to make safe". The run is real evidence for the single-writer half and none for the gate. The rule - an acceptance step names the observable that proves it entered the path under test, and the observation records that observable rather than that the command ran - applies to M2.5 (needs a child emitting escapes plus a second tab, with the deferral counted), M3.7 and M4.3.
  - id: new
    severity: Important
    family: external-input-assumed-wellformed
    title: |
      rowtext.Sanitize passes the C1 controls, and it is now the repo's single row-safety strip
    detail: |
      This is the 3rd finding in family external-input-assumed-wellformed, so the rule rather than the site. rowtext.go:37 drops only r < 0x20 and 0x7f; measured, "a\u009b31mred" and "a\u009d0;titlex" survive unchanged, and U+009B/U+009D are the 8-bit CSI and OSC introducers a UTF-8 terminal with C1 handling will act on. ansi.Strip does not frame them either. Inherited from couchtty, but the package doc and atlas/architecture.md:532 now make this the one implementation, and M3 routes operator-typed tab names through it. The rule - the sanitizer's notion of "control" is derived from what a terminal treats as a control (C0, DEL, C1 U+0080-U+009F, and the introducers in both 7- and 8-bit forms) and its test table enumerates those classes rather than the ones a past incident produced. Same edit - couchtty/menu_render.go:625 open-codes Fit(Sanitize(...)) where SanitizeAndFit exists.
  - id: new
    severity: Minor
    family: atlas-points-at-old-home
    title: |
      couchtty/reserve.go keeps the doc comments for the sanitize and truncate it no longer has
    detail: |
      This is the 3rd finding in family atlas-points-at-old-home, so the rule - an extraction moves the prose with the symbol and leaves neither comment nor stub at the old home. reserve.go:143-152 ends in two paragraphs describing functions now in rowtext, so a reader greps couchtty for sanitize and finds documentation for a function that is not there. Greppable form of the check - a doc comment whose subject identifier no longer resolves in its own file.
```

---

## Re-review — 2026-09-07T23:06:21-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 199 — pair term: own the right pane tab bar |
| repo | pair |
| issue file | workshop/issues/000199-pair-term-own-the-right-pane-tab-bar.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 96f780da661b35aef301b612b14ee8399bd9a6cf..4f5bc0ea724ad2b50c8f45a6eb22bf7bba2ff4a3 |
| command | sdlc milestone-close --issue 199 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-07T23:06:21-07:00 |
| verdict | REWORK |

## Review

I've completed the inspection. Verdict and findings follow.

```verdict
verdict: REWORK
confidence: high
```

The two head commits do what they claim: BR-35's gate fix is real and mutation-pinned in both directions (moving `FeedFraming` back outside the `isActive` branch reddens the background-tab subtest; deleting the replay feed reddens the replay subtest), and BR-37's C1 strip is pinned by a class-shaped table plus an over-stripping guard. Full `make test` is green (exit 0) and `go test -race ./cmd/internal/termcmd/` is clean. What blocks the boundary is that the milestone's oldest Critical is still live — **BR-25 reproduced again at HEAD**, deterministically and under load: `handleChunk`'s EOF branch runs `removeTab` on the writer goroutine, which posts to the channel that goroutine is the only drainer of, parking the pane permanently — and the head commit's own fix introduced a second one: the takeover now feeds the replay to the gate (correct) and then writes the owed diagnostics **without consulting it** (run.go:791-793), so a queued `pair term:` line lands inside the replay's open CSI. I reproduced that too: `"x\x1b[3\x1b[1;1H\x1b[Jrestored\x1b[3pair term: queued failure\r\n"` — the exact failure M2 exists to prevent, in the one write path the milestone added last.

## 1. Strengths

- **BR-35's fix is genuinely defended, not asserted.** Mutation-checked both directions: `run.go:826` (feed inside `isActive`) → `TestTheGateSeesExactlyWhatTheTerminalSees/a_background_tab_cannot_pin_the_gate` red; `run.go:791` (replay feed) → `/a_takeover's_replay_is_fed` red. The rule it states — the gate models the terminal, so it is fed exactly what the terminal is shown — is the right generalization and is now visible in the code's own comments.
- **BR-37 is fixed at the class, and the over-stripping direction is covered too.** `rowtext.go:44` strips C0/DEL/C1; reverting the C1 clause reddens four subtests. `TestSanitizeKeepsPrintableTextAroundTheC1Block` guards the opposite error. I also measured the raw 8-bit spelling: `"a\x9b31mred"` → `"a\uFFFD31mred"`, so the one-byte introducer degrades to a printable replacement rather than reaching the terminal as CSI.
- **The extraction left no second implementation.** `couchtty`'s `sanitize`/`truncate` are gone, both call sites repointed, and `artifactpath.NonArtifactSources` was updated in the same commit — the hand-maintained-list class that cost M1 three lists was remembered this time.
- **The test apparatus does not perturb what it observes.** `drainForTest`'s bare event (`run.go:958`) and `midSequenceForTest`'s `onWriter` probe (`run.go:961`) read gate state on the goroutine that owns it, with the earlier empty-`own` mistake recorded in the comment.

## 2. Critical findings

**C1 — `removeTab` still posts to the writer goroutine's own channel (BR-25, third raise).** `run.go:772` → `removeTab` (`:1088`) → `redrawTab` (`:1303`) → `enqueue` (`:922`).

Reproduced twice at HEAD with an overlay test (since deleted; tree verified clean):

```
TestOverlayBR25RemoveTabPostsToItsOwnQueue  — writer goroutine parked > 3s
TestOverlayBR25CloseUnderLoad               — writer never drained again after a tab closed under load
```

`enqueue`'s only escape is `<-m.done`, which nothing closes on this path. Two tabs, one flooding, operator closes the other → the pane renders nothing ever again, and the children then block on the Sink's unguarded send (`run.go:712`). Fix sketch unchanged from round 6: extract the takeover body into a writer-goroutine-only helper shared by `handleChunk`'s `takeover` case and `removeTab`, so `removeTab` applies it inline; regression test = two tabs, one flooding, close the other, assert the loop drains within a timeout. While you are there, `setPaneTitle` (`run.go:1142`) spawns a `zellij action` subprocess *on the writer goroutine* — tens of ms during which no byte reaches the pane; that is what widens the deadlock window and it is worth moving off the loop in the same edit.

**C2 (new) — the takeover flushes owed diagnostics without consulting the gate it just fed.** `run.go:791-793`.

Reproduced: a diagnostic queued while the child was mid-sequence, then a tab switch whose replay ends mid-sequence, yields `restored\x1b[3pair term: queued failure` on the pane — the diagnostic's leading bytes are swallowed as parameters of the open CSI. Note the asymmetry that makes this specifically the *diagnostic* write: `chunk.own` at `:786` begins with `\x1b[1;1H`, and an ESC inside a CSI parameter string cancels it in xterm-class terminals, so couch's "a takeover ends the stream's relevance" exemption survives; the diagnostic is plain text and does not. Full detail and the rule are in the findings block below.

## 3. Important findings

- **BR-26 (not-addressed, fourth raise).** Four mutations still leave the whole suite green, measured this round: deleting `m.owed = nil` (`run.go:780`), deleting `detail = rowtext.SanitizeAndFit(...)` (`run.go:1447`), and swapping `resizeThroughWriter` back to `inheritSize` (`run.go:264`). `tests/plan-superseded-facts-test.sh:65` still bans `route only stdout through`, a string `git log -S` finds in **no** revision of the plan — a guard that cannot fire. The rule the family asks for still is not written anywhere.
- **BR-27 (not-addressed).** The capture half landed; the *log* half did not. `run.go:462,468` (wheel ticks), `:1142` and `:1234` (`setPaneTitle`) still discard the error, so the failure mode the finding named — a failing action that is completely silent — is unchanged for the highest-frequency callers. `reportError` exists and is one line away from each of them.
- **BR-33 (not-addressed).** `atlas/architecture.md:501-508` still says a console-originated write is deferred into a *single coalescing slot* and dropped by a takeover, which has been false for diagnostics since 34918a3c; `:527` says a diagnostic "is gated like any other write", which C2 shows is false in the takeover path; `:532` still lists `pair term`'s strip as a current `rowtext` consumer, which M3 has not built. The rule the finding asked for (extend the superseded-facts test to `atlas/`) is not implemented.
- **BR-36 (not-addressed).** The evidence half is genuinely corrected — the issue Log and M2.5 now say the flood run accepts the single-writer envelope and not the gate, which is honest and is the right shape. But the correction promises the gate's manual acceptance lands in M3.7, and M3.7 (`plan:589`) has no step that enters the gate path: no second tab, no escape-emitting child under load, no deferral counted. The gate therefore still has no manual acceptance anywhere in the plan.

## 4. Minor findings

- BR-8, BR-9, BR-29, BR-30, BR-31, BR-34, BR-38: all unchanged at HEAD — see the dispositions below.
- The `runZellijCaptured` doc comment (`run.go:1410-1418`) now documents `diagnosticWidth`: two consts were inserted between the prose and its function, so `runZellijCaptured` has no doc comment and `diagnosticWidth` has a wrong one. Same shape as BR-38, in this window.
- `diagnosticWidth = 200` is justified as "far short of wrapping onto the child's area"; the pane is ~80 columns and `m.cols` is a field of the receiver, so the claim is false and the derived bound is available.
- `couchtty/menu_render.go:625` still open-codes `rowtext.Fit(rowtext.Sanitize(line), width)` where `SanitizeAndFit` exists — BR-37's "same edit" clause, half-applied.

## 5. Test coverage notes

The suite's *shape* is right — positive controls before assert-absents, a bare drain event, gate state read on the owning goroutine. The gaps are all "the kind of bug this diff shipped":

1. No test drives `removeTab` through the loop with a full queue (C1), and none can: the three existing `removeTab` tests call it from the test goroutine.
2. No test asserts the owed-diag flush is gated (C2). `TestDiagnosticsAreQueuedNotCoalescedAndSurviveATakeover:242-255` uses a replay (`"replaced"`) that ends on a boundary, so it passes either way.
3. `TestNeitherZellijMethodHandsTheSubprocessThePanesDescriptors` executes the real `zellij`; its mutation-sensitivity for the stdout arm depends on the machine having a live session (measured in round 8). On a machine without `zellij` all four arms are vacuous — ARCH-MOCK's "tests that cannot run the stack against the fake".
4. BR-9's parameterization is still one hand-chosen split index.

## 6. Architectural notes for upcoming work

Working through the markers: **ARCH-PURE** passes (rowtext is pure and IO-free; the writer suite needs no pty). **ARCH-SECURE** passes (single sanitizing egress, pinned by mutation; both C1 spellings closed). **ARCH-DRY** — nit only (the `SanitizeAndFit` open-coding; the takeover re-implementing `writeDiag`'s write, which *is* C2). **ARCH-PURPOSE** — flag: BR-27's class is half-swept, the four dropping call sites are enumerable and one line each. **ARCH-MOCK** — flag: the zellij FD test is a live-binary test wearing a unit test's clothes. **ARCH-CONSTRAINTS** — flag: BR-31/BR-34 still undeclared (the second parse is now narrowed to the active tab by the BR-35 fix, which is worth recording when you declare it). **ARCH-ORDER** — flag, and it is where both Criticals live: a handler posting to the queue it drains, and an owed effect flushed at a point where the ordering predicate is not consulted. Before M3 multiplies the paint sites, write the `(state, event) → (state, effects)` table for the writer loop — `{clean, mid-sequence} × {child chunk, paint, diag, takeover, resize, EOF}` — because M3 adds a repaint on every `batch.RowDirty` and every one of those cells will be hot.

## 7. Plan revision recommendations

- `## Revisions` entry for C2: the takeover's third gate rule has a fourth clause — *a write the takeover carries forward is re-gated against the replay it just fed*, and the exemption for the takeover's own `HomeAndClear` is written down (it begins with ESC, which cancels a pending CSI) rather than left as something a reader must re-derive.
- `## Revisions` entry for M3.7: name the observable that proves the gate path was entered (a child emitting escapes, a second tab, the deferral counted), per BR-36's rule, since M2.5's correction points there.
- ARCH-CONSTRAINTS: add the two declared bounds (BR-31's second parse, now active-tab-only; BR-34's `owedDiag` cap) and correct `diagnosticWidth`'s basis.

```findings
dispose:
  - id: BR-8
    disposition: not-addressed
    note: |
      M4 untouched in this window; M4.3 still has no Alt+Shift+d step while the Done-when still requires two right-pane halves each drawing a strip.
  - id: BR-9
    disposition: not-addressed
    note: |
      writer_test.go:115 still splits one hand-chosen index of one sequence.
  - id: BR-25
    disposition: not-addressed
    note: |
      No code change. Reproduced twice at HEAD with overlay tests - deterministic (full queue, handleChunk parked >3s) and under load (writer never drained again). run.go:772 -> 1088 -> 1303 -> 922.
  - id: BR-26
    disposition: not-addressed
    note: |
      Measured green under mutation at HEAD - m.owed = nil (run.go:780), runZellijCaptured's SanitizeAndFit (:1447), resizeThroughWriter (:264); plan-superseded-facts-test.sh:65 still bans a string git log -S finds in no revision of the plan.
  - id: BR-27
    disposition: not-addressed
    note: |
      Log half still absent - run.go:462,468,1142,1234 discard the error, so a failing wheel tick or rename is still completely silent while reportError sits one line away.
  - id: BR-29
    disposition: not-addressed
    note: |
      run_test.go:941 fakeRuntime.reportedUnused still present and uncalled; terminalMux.stderr is still set at run.go:683 and read by nothing.
  - id: BR-30
    disposition: not-addressed
    note: |
      run.go:168 runDecision still takes stdin and stdout and uses neither.
  - id: BR-31
    disposition: not-addressed
    note: |
      ARCH-CONSTRAINTS unchanged. The BR-35 fix narrowed the second parse to the active tab, which is the fact to declare.
  - id: BR-33
    disposition: not-addressed
    note: |
      atlas:501-508 still says a single coalescing slot dropped by a takeover; :527 says diagnostics are gated like any other write, which C2 falsifies; :532 still lists M3's strip as a current consumer. The superseded-facts test still covers only the plan.
  - id: BR-34
    disposition: not-addressed
    note: |
      run.go:819 still appends to owedDiag unbounded; ARCH-CONSTRAINTS declares no budget. Same gap covers diagnosticWidth = 200, whose stated basis (far short of wrapping) is false for an 80-column pane while m.cols is on the receiver.
  - id: BR-35
    disposition: addressed
    note: |
      Mutation-verified both directions rather than taken from the commit - moving FeedFraming out of the isActive branch and deleting the replay feed each redden their own subtest.
  - id: BR-36
    disposition: not-addressed
    note: |
      Evidence half corrected honestly (Log + M2.5), but the correction defers the gate's manual acceptance to M3.7 and plan:589 has no step that enters the gate path - no second tab, no escape-emitting child, no deferral counted.
  - id: BR-37
    disposition: addressed
    note: |
      Reverting the C1 clause at rowtext.go:44 reddens four subtests; raw 8-bit spellings degrade to U+FFFD (measured). Residual, not re-raised - menu_render.go:625 still open-codes Fit(Sanitize(...)).
  - id: BR-38
    disposition: not-addressed
    note: |
      couchtty/reserve.go:143-152 still ends in doc comments for functions that moved to rowtext, and this window added the same shape at run.go:1410-1418, where two consts now sit between runZellijCaptured's prose and runZellijCaptured.
findings:
  - id: new
    severity: Critical
    family: validating-door-bypassed
    title: |
      The takeover writes owed diagnostics straight to the pane after feeding the replay to the gate, so a diagnostic lands inside the replay's open sequence
    detail: |
      This is the 2nd finding in family validating-door-bypassed, so the deliverable is the rule, not the site. run.go:792-793 writes each queued diagnostic with m.stdout.Write, immediately after run.go:791 feeds chunk.replay to hostScan - so when the replay ends mid-sequence (the case run.go:787-790 exists to handle, and which writer_test.go:419 asserts is real), MidSequence is true and the write goes out anyway. Reproduced with an overlay test - a diagnostic queued while the child was mid-sequence, then redrawTab("restored\x1b[3"), yields "x\x1b[3\x1b[1;1H\x1b[Jrestored\x1b[3pair term: queued failure\r\n" on the pane, the diagnostic's leading bytes swallowed as CSI parameters. Reachable: a failed rename or a failed zellij action while the child is mid-escape, then any tab switch. The takeover's own write at :786 is NOT the same case and must stay exempt - it begins with ESC, which cancels a pending CSI in xterm-class terminals, which is why couch's takeover writes ungated; that exemption should be stated rather than re-derived. The rule - every byte reaching m.stdout from the writer loop passes through the gate-consulting door (writeOwn / writeDiag), or carries a written exemption naming why the gate does not apply - covers the enumerable set `grep -n 'm\.stdout' cmd/internal/termcmd/run.go`: today :786 (exempt, takeover), :793 (the defect), :826 (the child stream itself), :839/:857/:868/:876 (inside the doors), :1321 (teardown, exempt). The test that pins the rule enumerates the console-originated write paths and asserts each defers while the gate is mid-sequence, which is what the current suite does for paints and not for the takeover-carried diagnostics.
```

---

## Re-review — 2026-09-07T23:27:09-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 199 — pair term: own the right pane tab bar |
| repo | pair |
| issue file | workshop/issues/000199-pair-term-own-the-right-pane-tab-bar.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 96f780da661b35aef301b612b14ee8399bd9a6cf..54f2273a63246ab97da220ab259b2c7f55b12ac9 |
| command | sdlc milestone-close --issue 199 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-07T23:27:09-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Both round-9 Criticals are genuinely fixed and I verified each by reverting it in a scratch copy: dropping `applyTakeover` back to `redrawTab` in `removeTab` reddens `TestAChildExitingWithAFullBufferDoesNotWedgeThePane` (BR-25), and putting `m.stdout.Write` back in the takeover's diagnostic loop reddens `TestADiagnosticNeverLandsInsideTheReplaysOpenSequence` with the exact corrupt byte string the finding predicted (BR-39). The full `go test ./cmd/...` is green (unsandboxed; the sandbox's pty failures are the known `ptychild: start: operation not permitted` class, not defects). What blocks SHIP is not a new crash: it is that the milestone's *stated* contract is still violated in two places that the diff's own prose describes as fixed. `runZellijCaptured`'s comment says the capture exists because "the wheel-tick callers also drop the error … a failing action became completely silent", and four callers (`run.go:462,468,1165,1257`) still drop it — pre-M2 those bytes reached the operator on `os.Stderr`, so the window is a net loss of a diagnostic the plan explicitly forbade losing, and there is no log sink anywhere (BR-27, open since round 5). And couch's third gate rule — the reset this milestone was written to import — turns out to be completely unpinned: deleting `m.hostScan = ptychild.Screen{}` (run.go:822) leaves the entire termcmd suite green, because both takeover fixtures (`"fresh"`, `"restored…"`) begin with a byte in the CSI final range that terminates the pending sequence by accident. That is measured, not argued, and it is the headline deliverable of M2.3 step 1.

## 1. Strengths

- **The writer envelope is genuinely swept, not asserted.** All six writers from plan finding 8 are accounted for at HEAD: `grep -rn "os.Stdout\|os.Stderr" cmd/internal/termcmd/*.go` (non-test) returns exactly one hit, and that is a comment. `ReportShortcutError` is gone from the whole tree with no stale implementer. The startup `fmt.Fprintf(stderr, …)` sites are all provably before `copyActiveOutput` starts (run.go:233,243,255 vs :271).
- **`reportError`'s routing is pinned by a failing test.** Mutating it to write `m.stdout` directly reddens two tests with real corrupt output — this is not a comment defending a fix.
- **`owedDiag`'s queue-not-coalesce policy is pinned.** Collapsing it to a single slot reddens `TestDiagnosticsAreQueuedNotCoalescedAndSurviveATakeover` at the right assertion.
- **`ptyChunk` as one type over one channel** (run.go:604-613) with the rationale written down — "two channels would let a select choose between them in an order nothing pins" — is the correct ARCH-ORDER call, and `drainForTest`/`midSequenceForTest` give tests a real seam to inject ordering rather than sleeping. That is the opposite of the single-interleaving oracle the principle warns about.
- **`rowtext` is a clean ARCH-DRY win**: two unexported copies became one package with its own tests, and the C1 clause is mutation-verified (reverting it reddens four subtests).

## 2. Critical findings

None new. The two Criticals from round 9 are fixed in code and mutation-verified; BR-39 remains disposed `not-addressed` only for its *class* half (below), not for the corruption itself.

## 3. Important findings

**N1 — `cmd/internal/termcmd/run.go:822` / `writer_test.go:194`: couch's third gate rule is unobservable.** `m.hostScan = ptychild.Screen{}` can be deleted with the whole `./cmd/internal/termcmd` suite still green. Both takeover fixtures start with `f` (0x66) and `r` (0x72), both legal CSI final bytes, so `FeedFraming(replay)` closes the pending sequence by itself and `midSequenceForTest()` reads false either way. Discriminating fixture, verified in a scratch copy: feed `"x\x1b[3"`, then `redrawTab([]byte("0000"))` — parameter bytes only, no final. Green with the reset, red without it.

**N2 — `cmd/internal/termcmd/run.go:1426`: the descriptor claim is false for stdin.** `cmd.Stdin = os.Stdin` hands the subprocess the pane's stdin — which `host.MakeRaw()` has put in raw mode — while run.go:1394 states "NEITHER method gives a subprocess the pane's descriptors, and that is the point" and `atlas/architecture.md:514` repeats it. `TestNeitherZellijMethodHandsTheSubprocessThePanesDescriptors` is named for all the pane's descriptors and asserts two of three. Whether any `zellij action` verb reads stdin is unmeasured, which is the reason to close it off rather than reason about it.

**BR-27 (still open) is the one that decides the verdict** — see the disposition note. The fix has an interaction worth flagging before it is attempted: `run.go:1165` sits inside `removeTab`, which BR-25 just established runs *on* the writer goroutine, so routing it through `reportError` → `enqueue` re-creates the deadlock. It needs the inline `writeDiag` door, same as `applyTakeover` uses.

## 4. Minor findings

- Plan `:279-281`: the Integration-points table still marks three M3/M4 rows `new`/`modified` while BR-11 flipped only the Pure-entities table to `planned — M3`. `plan-table-drift`, 5th.
- `couchtty/menu_render.go:625` open-codes `rowtext.Fit(rowtext.Sanitize(line), width)` where `SanitizeAndFit` exists. Round 9 recorded this as a residual and declined to raise it; noting it so it is not lost.
- `TestAChildExitingWithAFullBufferDoesNotWedgeThePane` never closes `m.done`, unlike every other test in the file.

## 5. Test coverage notes

Mutation results at HEAD, measured rather than read:

| mutation | suite |
|---|---|
| `applyTakeover` → `redrawTab` in `removeTab` | **red** (BR-25 pinned) |
| takeover diagnostics → `m.stdout.Write` | **red** (BR-39 pinned) |
| `owedDiag` append → single slot | **red** |
| `reportError` writes stdout directly | **red** |
| `m.owed = nil` deleted from `applyTakeover` | *green* (BR-26) |
| `m.hostScan = ptychild.Screen{}` deleted | *green* (N1) |
| `runZellijCaptured`'s `SanitizeAndFit` deleted | *green* (BR-26) |
| `resizeThroughWriter` → direct `inheritSize` | *green* (BR-26) |

Also: `tests/plan-superseded-facts-test.sh:65` bans `'route only stdout through'`, a string `git log -S` finds in **no** revision of the plan — the registered pair has never fired and cannot. That is BR-33's own rule ("registering a pair is not evidence unless it fires") failing inside the mechanism BR-33 asked to extend.

## 6. Architectural notes

- **ARCH-DRY — pass.** `rowtext` removes the duplicate strip; `applyTakeover` is one implementation reachable both inline and by event, which is the right answer to BR-25 rather than a second copy.
- **ARCH-PURE — pass with a note.** The gate is a value fed bytes by the test, no pty needed. The deferral *policy* (`writeOwn`/`writeDiag`/`flushOwed`) is still interleaved with `m.stdout.Write`; a pure `(state, event) -> (state, []effect)` would make N1 unfalsifiable-by-fixture. Not worth doing for M2; worth considering before M3 adds paints.
- **ARCH-PURPOSE — flag.** The shadow-sweep on writers is complete and that is real. The sweep on *diagnostics* is not: capture was built and four consumers still discard (BR-27). A capture no consumer reads is the deferred purpose, not a follow-up.
- **ARCH-MOCK — flag (folded into BR-26).** `TestNeitherZellijMethod…` execs the real `zellij`; there is no stateful fake at the `runZellij` seam, so the `succeeds` arms assert nothing on a box with no live session. `fakeRuntime` sits one layer above and cannot see descriptors.
- **ARCH-CONSTRAINTS — flag (BR-31, BR-34).** Undeclared: a second full byte-scan of every active chunk (`Screen.feedFraming` is a per-byte loop); an unbounded `owedDiag`; `diagnosticWidth = 200` on a pane that is usually 80 columns while `m.cols` is on the receiver. Add: `removeTab` now spawns a `zellij action` subprocess **on the sole writer goroutine** (run.go:1165), so a child exit stalls all pane output for one exec. Not a regression — it was already there — but M2 made that goroutine the only path for every byte, which changes what the stall costs.
- **ARCH-SECURE — pass.** Sanitizing at the single egress rather than per producer is the right seam, and the hostile-error test drives escape, SO, newline, CR and length together.
- **ARCH-ORDER — flag.** The `(event) -> effect` dispatch in `handleChunk` is explicit and readable, and the channel gives tests a real ordering seam — good. Two gaps: `owed`/`owedDiag`/`hostScan` carry an unwritten invariant (`owed != nil` implies `MidSequence()`), held by construction and enforced nowhere; and nothing flushes either slot if the child stops emitting while mid-sequence, which BR-27 named and which `preserveRename` in `removeTab` can also reach.

## 7. Plan revision recommendations

- **`## Revisions` — "the third gate rule was never observable".** Record N1 with the measurement (which byte in each fixture closed the sequence) and the discriminating fixture, so the next reader does not re-derive why `"fresh"` was a bad choice.
- **`## Revisions` — the descriptor claim's scope.** Either narrow the claim in plan finding 8, run.go:1394 and `atlas/architecture.md:514` to *output* descriptors, or close stdin; do not leave three artifacts asserting all three.
- **`## Core concepts` → Integration points.** Flip `strip repaint trigger`, `degraded rename-pane` and `right pane chrome` to `planned — M3` / `planned — M4`, and state the rule that the status column is the build tracker in **both** tables — BR-11 fixed one table and this is the other.
- **`## ARCH-CONSTRAINTS`.** Declare the second parse, bound `owedDiag`, derive `diagnosticWidth` from `m.cols`, and name the subprocess-on-the-writer-goroutine stall.
- **M3.7(b) and M4.3.** `yes` emits no ESC byte, so M3.7(b) as written still cannot enter the defer path — name the observable (an escape-emitting child, a second tab, a counted deferral) rather than the command.

```findings
dispose:
  - id: BR-25
    disposition: addressed
    note: |
      Verified by revert, not by the commit message: restoring redrawTab in removeTab reddens TestAChildExitingWithAFullBufferDoesNotWedgeThePane at the 3s timeout. The rewritten test is deterministic (buffer saturated, no loop running).
  - id: BR-39
    disposition: not-addressed
    note: |
      Instance fixed and mutation-verified (reverting writeDiag to m.stdout.Write reproduces the exact predicted bytes). Class half absent: applyTakeover's own ungated write at run.go:826-827 carries no written exemption, and no test enumerates the console-originated doors.
  - id: BR-26
    disposition: not-addressed
    note: |
      Re-measured at HEAD. Green under all three mutations: m.owed = nil (run.go:822), runZellijCaptured's SanitizeAndFit, resizeThroughWriter. plan-superseded-facts-test.sh:65 bans a string git log -S finds in no revision of the plan, so that registered pair can never fire.
  - id: BR-27
    disposition: not-addressed
    note: |
      Log half absent. run.go:462,468,1165,1257 still discard the error; there is no log sink, so a failing wheel tick or rename is completely silent where pre-M2 it printed. run.go:1437's own comment states this as the reason capture exists. Note run.go:1165 is on the writer goroutine, so its fix needs inline writeDiag, not reportError.
  - id: BR-33
    disposition: not-addressed
    note: |
      atlas:502-503 still describes every console-originated write as one coalescing slot, false for diagnostics; atlas:532-534 lists pair term's strip as a current rowtext consumer when it is M3. tests/plan-superseded-facts-test.sh still covers only the plan file.
  - id: BR-34
    disposition: not-addressed
    note: |
      run.go:848 still appends to owedDiag with no cap; ARCH-CONSTRAINTS declares no budget. diagnosticWidth = 200 claims to be far short of wrapping while m.cols is on the receiver.
  - id: BR-31
    disposition: not-addressed
    note: |
      ARCH-CONSTRAINTS unchanged. Screen.feedFraming is a per-byte loop run over every active chunk in addition to ptychild's own parse; the BR-35 fix narrowed it to the active tab, which is the fact to declare. Same gap covers the zellij exec now running on the sole writer goroutine in removeTab.
  - id: BR-36
    disposition: not-addressed
    note: |
      M2.5 and the Log are corrected honestly, but M3.7(b) still specifies yes, which emits no ESC, so the deferral path still cannot be entered; no observable is named and no deferral is counted. M4.3 unchanged.
  - id: BR-38
    disposition: not-addressed
    note: |
      couchtty/reserve.go:143-152 still ends in doc comments for sanitize and truncate, which live in rowtext now. This window added the same shape at run.go:1431-1441, where two consts sit between runZellijCaptured's prose and runZellijCaptured.
  - id: BR-29
    disposition: not-addressed
    note: |
      run_test.go:941 fakeRuntime.reportedUnused still present and uncalled. terminalMux.stderr is set at run.go:683 and read nowhere (grep for m.stderr returns nothing).
  - id: BR-30
    disposition: not-addressed
    note: |
      run.go:168 runDecision still takes stdin and stdout and uses neither; panes is likewise unused in the body.
  - id: BR-8
    disposition: not-addressed
    note: |
      M4 untouched in this window; M4.3 still has no Alt+Shift+d step while the Done-when still requires two right-pane halves each drawing a strip.
  - id: BR-9
    disposition: not-addressed
    note: |
      writer_test.go:115-116 still splits one hand-chosen index of one sequence.
findings:
  - id: new
    severity: Important
    family: uncovered-negative-assertion
    title: |
      The takeover's scan reset -- couch's third gate rule, M2.3 step 1's headline -- can be deleted with the whole suite green
    detail: |
      This is the 3rd finding in family uncovered-negative-assertion, so the deliverable is the rule, not the site; it also shows BR-26's stated sweep list ("the four sites in writer_test.go plus run_test.go:910") was an enumeration written from memory, since this site is not on it. Measured in a scratch copy: deleting m.hostScan = ptychild.Screen{} (run.go:822) leaves ./cmd/internal/termcmd fully green. The reason is the fixtures, not the assertion -- both takeovers replay bytes beginning with f (0x66) and r (0x72), and both are legal CSI final bytes, so FeedFraming(replay) closes the child's pending sequence on its own and midSequenceForTest() reads false with or without the reset. A discriminating fixture is one line: feed "x\x1b[3", then redrawTab([]byte("0000")) -- parameter bytes with no final. Green with the reset, red without it (verified both directions). The rule: an assertion that a state was CLEARED is evidence only when the fixture cannot reach that state by any path other than the clearing. In practice that means a fixture chosen to be inert with respect to the mechanism under test, and the check that makes it stick is the mutation, not the read -- which is the same rule BR-26 states for assert-absent writes, now shown to govern assert-absent STATE as well.
  - id: new
    severity: Important
    family: envelope-claim-unenforced
    title: |
      runZellij hands the subprocess the pane's raw-mode stdin, while code and atlas both claim it gives neither descriptor
    detail: |
      This is the 2nd finding in family envelope-claim-unenforced, so state the rule rather than patching the line. run.go:1426 sets cmd.Stdin = os.Stdin -- the pane's own descriptor, put in raw mode by host.MakeRaw() at run.go:242 -- while run.go:1394 asserts "NEITHER method gives a subprocess the pane's descriptors, and that is the point" and atlas/architecture.md:514 repeats it. The enforcement test is named TestNeitherZellijMethodHandsTheSubprocessThePanesDescriptors and captures two of the three, so the claim's third case is unasserted by construction. Whether any zellij action verb reads stdin is unmeasured; a verb that reads one byte eats an operator keystroke silently, which is exactly the class the milestone exists to close. The rule: an envelope claim about a RESOURCE enumerates every handle of that resource and the test asserts each one -- "descriptors" means the three the process was started with, not the two the finding that prompted the work happened to name. The enumerable set here is the three fields of exec.Cmd set in runZellij; either close stdin or narrow the claim in all three artifacts (plan finding 8, run.go:1394, atlas:514) in the same commit.
  - id: new
    severity: Minor
    family: plan-table-drift
    title: |
      The Integration-points table still marks three M3/M4 rows new/modified after BR-11 flipped only the Pure-entities table
    detail: |
      This is the 5th finding in family plan-table-drift, so do not patch the three rows -- state the rule. plan:279-281 declare "strip repaint trigger" new, "degraded rename-pane" modified and "right pane chrome" modified; none exists at HEAD, all three are M3/M4. BR-11 raised exactly this for plan:114-117 and those rows now read "planned -- M3", which is the instance fixed and the class left standing in the table directly below it. The guard cannot catch it: core_concepts_contract_test.go filters on conceptPackage = cmd/internal/couchtty/, so every #199 row declaring termcmd, hostty, rowtext or main-3.kdl is unchecked, and the status column that "doubles as the build tracker" is only a tracker for one package. The rule: the status column is the build tracker for EVERY row of EVERY table in a Core concepts section, and the check that enforces it is scoped to the plan, not to a package -- which is the same gap BR-33 names for atlas, and the reason both should be answered by one mechanism rather than two hand-maintained lists.
```
