# Boundary Review — pair#221 (whole-issue close)

| field | value |
|-------|-------|
| issue | 221 — ctrl+return jumps straight to the newest paging thread |
| repo | pair |
| issue file | workshop/issues/000221-ctrl-return-jumps-straight-to-the-newest-paging-thread.md |
| boundary | whole-issue close |
| milestone | — |
| window | 765c17d8de60e1515d94565f41f70d3a3bcdf2ca..30376a0b332f93279295ee76049fba64264381d0 |
| command | sdlc close --issue 221 |
| reviewer | claude |
| timestamp | 2026-09-10T13:36:38-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The code does what the Spec and Plan say. `ctrl+return` is one `knownSequences` row plus the `seqKind`, `hit()`, `AllInterceptorHits` and `hitHandlers()` entries the Plan calls for. It picks its target with the same `attention.NewestActor()` call `ctrl-space` uses. It lands through `switchTo` with `arrivalNotification`, and all five outcomes in the Design table are implemented as written. I checked the tests rather than trusting the Log. `go build ./...`, `go vet` and `gofmt` are clean. The `couchtty` tests pass under `-race`, except `TestNotificationPTYConformance`, which fails here because this shell can't start a pty; that test is outside the diff. The `couchcmd` README test passes.

In a scratch copy I applied 8 mutations of the handler. Seven fail a named test. The survivor, restating the panel arm as `KeyEnter`, is the one the Log already lists as "honestly unpinned". A ninth mutation, painting the switcher before the jump, is also caught by the differential test's "threads" check. Nothing blocks shipping. What's left: two code comments that still describe the old behaviour, one reasoning error in the issue's ARCH-ORDER note, and a small duplicated check.

1. **Strengths**
   - `console_newest_page_test.go:109` drives both paths through the real input path and compares the whole landing, including the `SwitchTracker`. It uses three actors so the newest pager, the first pager and the first row are all different, and then follows the second page and `ctrl+backspace` home. Mutating to `arrivalOrdinary` or `arrivalPrevious` fails it.
   - `keys.go:32-45`: the legacy-encoding note sits beside `previousByte`, and one named constant feeds both the `knownSequences` row and the panel arm, so the panel's meaning is derived rather than restated.
   - `keys_test.go:470` covers every codepoint-13 encoding Pair consumes, plus the chord's key release, ctrl+shift and a bracketed paste. The fuzz corpus is seeded.
   - `console.go:1438-1471`: the `switch` matches the Design table arm for arm. Using `force=false` gives the "stay" behaviour through `switchTo` without taking over the screen, and the stay test catches `force=true`.
   - The README sweep is complete: I grepped for chord listings and found no stale ones. `menuControls` gets the new key, and the atlas entry records why the design chose a direct `switchTo` over the queued `switch` operation.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - `console.go:447-448` and `switchrule.go:30-31` still say a notification hop is *only* `ctrl-space` + Return. `ctrl+return` now produces one too. The README chord listings were swept, but these definitions weren't. Fix: say "`ctrl-space` + Return or `ctrl+return`", or "an arrival on an actor that was paging when chosen" (ARCH-PURPOSE).
   - The issue's ARCH-ORDER note says "`attention.Mark` runs only in `onChunk`, on this same goroutine". That's false. `switchTo` also runs on the operation goroutine, via `ExecuteConsoleOperation`'s `switch` (`console.go:1940`). It calls `flushDeferredNotifications` (`:504`), which calls `onChunk` and then `Mark` (`:1271`). The effect here is harmless: the jump lands on a thread that is still paging, and the new page stays lit. Only the stated reason needs correcting.
   - `stay` (`console.go:1435`) repeats `switchTo`'s own `already` check under a separate lock. A status-chip click running `switchTo` on the operation goroutine between the two could make the "already on the paging thread" notice wrong or missing. It's very unlikely and only cosmetic. The fix is to have `switchTo` report whether it landed or stayed, so the check has one owner (ARCH-DRY/ARCH-ORDER).
   - The operator saw the screen jump up one line on a thread switch. The Log attributes it to #209's repaint nudge and says it "is left for its own issue", but no issue exists; #224 covers the single-writer door, not this. File one with `sdlc issue new` so it isn't lost.

5. **Test coverage notes:** Every arm of the handler has a test that goes red when that arm is broken. Splits and the handler table are covered by the existing walkers. The interceptor tests are exact-byte and table-driven. The only unpinned choice is `DecodePanelKeys` versus a literal `KeyEnter`, which no behaviour test can tell apart today, and the Log says so. The not-attached test calls the handler directly on a console that isn't running, which is the right way to hold the exit unreduced.

6. **Architectural notes**
   - **ARCH-DRY: pass**, apart from the `stay` note above.
   - **ARCH-PURE: pass.** The handler is glue over the existing pure `AttentionLedger` and `SwitchTracker`.
   - **ARCH-PURPOSE: pass**, apart from the two stale definitions above. Both consumers of "who paged most recently" call `NewestActor()`.
   - **ARCH-MOCK: pass.** There is no new external dependency, and the tests use `FakeHost` and `FakeChild`.
   - **ARCH-CONSTRAINTS: pass.** This is a keystroke path doing microseconds of work.
   - **ARCH-SECURE: N/A.** The input is operator keystrokes matched exactly, with nothing persisted.
   - **ARCH-ORDER: pass**, apart from the reasoning error above. The race between a chip click and a hotkey existed before (`ctrl+backspace` has it too) and has no ordering seam in tests. That belongs with the couch-wide writer work in #224, not here.

7. **Plan revision recommendations:** Add a `## Revisions` entry that corrects the ARCH-ORDER bullet's "Mark runs only on the Run goroutine": `switchTo` → `flushDeferredNotifications` → `onChunk` can also run on the operation goroutine. The consequence for `ctrl+return` is harmless (it lands on a thread that is still paging).

```findings
findings:
  - id: new
    severity: Minor
    family: doc-sweep-enumeration
    title: |
      arrivalNotification and SwitchTracker.Switch docs still define a notification hop as only ctrl-space + Return
    detail: |
      console.go:447-448 and switchrule.go:30-31 state an exclusive definition that ctrl+return (console.go:1464) now breaks. The README chord listings were swept but these arrival-kind definitions were not. Say "ctrl-space + Return or ctrl+return", or "an arrival on an actor that was paging when chosen".
  - id: new
    severity: Minor
    family: false-rationale
    title: |
      The issue's ARCH-ORDER note says attention.Mark runs only on the Run goroutine, which is false
    detail: |
      switchTo also runs on the operation goroutine via ExecuteConsoleOperation's switch (console.go:1940). It calls flushDeferredNotifications (console.go:504), which calls onChunk and then Mark (console.go:1271). The effect on ctrl+return is harmless (it lands on a thread that is still paging); correct the stated reason in a Revisions entry.
  - id: new
    severity: Minor
    family: single-authority-predicate
    title: |
      The stay notice repeats switchTo's own "already" check under a separate lock
    detail: |
      stay (console.go:1435) is computed before the lock is released, and switchTo recomputes already (console.go:474). A status-chip switch on the operation goroutine between the two can make the notice wrong or missing. Have switchTo report whether it landed or stayed (ARCH-DRY/ARCH-ORDER).
  - id: new
    severity: Minor
    family: discovered-defect-untracked
    title: |
      The one-line screen jump on a thread switch is "left for its own issue", but no issue exists
    detail: |
      The Log attributes it to #209's repaint nudge (RepaintSettle); #224 covers the single-writer door, not this. File it with sdlc issue new so it is not lost.
```
