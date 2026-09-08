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
