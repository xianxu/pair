# Boundary Review — pair#262 (milestone M1)

| field | value |
|-------|-------|
| issue | 262 — Screen flicker: the compositor re-emits global terminal state every frame (#255) |
| repo | pair |
| issue file | workshop/issues/000262-diagnose-input-screen-flicker.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | a1b0a7d8e53f878b1ce6b543c2f6e4fead4c970b..72c2bbfcb30b02ccb5db66aac7590a4b71f55d6a |
| command | sdlc milestone-close --issue 262 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-17T20:16:42-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The M1 code is correct, small and well pinned. Both pure renderers now wrap every frame in one DECSET 2026 bracket (a synchronized-output begin/end pair). On an alt-screen switch, the begin marker goes out as its own write, so the `?1049h`/`l` packet stays whole and `Presenter.write`'s exact-match alt tracking still works. Release closes a bracket that a failed write left open, and the cursor epilogue is shared by both renderers. I confirmed the key regression evidence myself. With the release fix reverted through a `go test -overlay` scratch copy (the repo was not touched), `TestPresenterReleaseClosesSyncAfterAnyCutWrite` fails in all three shapes at cut 8, as the Log says. With the fix in place, the package and the xterm oracle suites pass, and the new tests pass 30× under `-race`.

One thing needs fixing before the boundary. An M1 Done-when bullet was dropped without revising the issue: the plain-zellij `pair term` smoke, and the record of whether zellij honours 2026. Every live `pair term` pane in the smoke ran the pre-M1 binary, so the bracketed `Emit` has had no live run through zellij.

1. **Strengths**
   - `history_render.go:305-307`: `e.add(syncBegin)` goes before the alt packets, and `packet()` flushes it as a separate write. This keeps both the "alt switch inside the bracket" rule and the "`?1049h` packet stays whole" invariant (`presenter.go:196`) with no new state.
   - `presenter.go:273`: `syncEnd` comes right after CAN/ST, so a half-written `?2026h` is aborted and a complete one is closed. Release is a convergent write, so the parent's bracket state needs no tracking (ARCH-ORDER handled by construction).
   - `presenter_test.go:931-1017`: the cut sweep is deterministic. Scripted `ttyio.Fake` steps cover three write layouts, and every failing offset reproduces exactly. It goes red without the fix, which I verified.
   - `render.go:24-38`: `cursorEpilogue` is a faithful lift. The CUP format matches `historyEmitter.cup`, and DECSCUSR/`?25h` are unchanged, so M2 has a single site to change.
   - The prose sweep went beyond symbols: architecture.md is condensed correctly, and I confirmed `stripmutation_test.go` is gone and `ptychild.Screen` has no production consumer. The couch.md teardown sentence is now accurate against `releaseAlt` + `parentReleaseControls`.

2. **Critical findings:** none.

3. **Important findings**
   - **M1 Done-when bullet 5 was waived without an issue revision** (`workshop/issues/000262-…md`, Done-when). It requires "`pair term` under plain zellij is smoked too, and whether zellij honours 2026 from a pane is recorded whichever way it goes." The Log says outright that this was not done. The plan's Task 5 checkbox is `[x]` anyway, and the only disclosure is a plan-side "execution notes" revision.
     - The Log calls the omission "harmless either way", but nothing supports that. Under plain pair (no couch), which must keep working standalone, `Emit` still erases and repaints the whole pane on every dirty frame. If zellij 0.45.1 ignores 2026 from a pane, zellij can render between the erase and the repaint, so the plain-pair flicker is unknown.
     - The native zellij oracle only shows that the end state is unchanged. It can't show whether zellij honours the bracket.
     - Fix, either way cheap: have the operator open a fresh `pair term` split in plain zellij and record the result, or add an issue `## Revisions` entry that moves this bullet to M2 or drops it, with the reason.
     - The "static agent pane + typing" regime is also not explicitly recorded; the smoke quotes describe typing while the agent streams.

4. **Minor findings**
   - Stale "two consumers" prose survived the Task 4 sweep, in the same class PQ-1 named:
     - `hostty/reserve.go:51-55` still says "both consumers assert the region and draw the row together".
     - The `couchnestedrows/main.go:4-14` header still says, in present tense, "Under couch there are two" reserved rows.
     - The new `hostty/reserve.go:15-16` text says the region reset is owned "not [by] this file", yet this file's `Release` still writes `\x1b[r` for the probe.
   - The plan's 2026-09-17 execution note says "the caret-blink question is still open", but commit `72c2bbfc` answered it in the Log. The plan's Task 5 `[x]` also covers sub-item 3, which was never done.

5. **Test coverage notes**
   - Covered: `Render` (first paint, one cell, cursor only, no-op); `Emit` (reset, history append, steady edit, idle, alt enter/leave, multi-chunk); the presenter stream; release after any cut.
   - I confirmed through an overlay probe that the "history append" case really takes the push path: its wire contains `ESC[1;2r`.
   - `assertStreamBracketed` uses `ESC[?25l` as its stand-in for frame bytes. At stream level it only checks the start of the preamble, and the renderer tests cover the suffix. That's fine for now, but weaker than the "no frame byte outside the bracket" wording.
   - The alt-leave case is checked for prefix and suffix but not with `assertOneBracket`.
   - I could not re-run the native zellij oracle or the pty-child suites (couchtty/termcmd soak and PTY tests). This environment returns EPERM on `/tmp` mkdir and pty spawn, which is not a code failure. For those, the Log's with-sandbox-off results are the only evidence.

6. **Architectural notes**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | Pass: one set of bracket constants, one epilogue. The preamble literals repeated in `parentReleaseControls` predate this diff. |
   | ARCH-PURE | Pass: the bracket lives in the pure renderers and is tested at byte level. |
   | ARCH-PURPOSE | Flag: the couch-path purpose is delivered; the plain-zellij verification and one class sibling are not (Important and first Minor above). |
   | ARCH-MOCK | Pass: `ttyio.Writer` is the seam and `ttyio.Fake` the stateful double. xterm and zellij act as oracles. Ghostty's honouring path is covered only by the operator smoke, which is acceptable. |
   | ARCH-CONSTRAINTS | Pass: +16 B per frame. Sync-open time is bounded by one paint, with each write capped at 2s. The per-keystroke full repaint is deferred with a named trigger (#120 or a measurement). |
   | ARCH-SECURE | N/A: the change only emits constant sequences. |
   | ARCH-ORDER | Pass: no state is carried between frames, release is convergent, and the failure orderings are reproducible through the fake. |
   | ARCH-FUNERAL | Pass: the change creates nothing durable. |

   For M2: `cursorEpilogue`'s `shape == 0 → 1` is where the "terminal default is not representable" finding lands. Carrying Shape 0 through needs the vt-level change noted in the Log.

7. **Plan revision recommendations**
   - Issue `## Revisions`: amend M1 Done-when bullet 5, deferring or dropping the plain-zellij smoke and the zellij-2026 record with the reason. Or run the smoke and log it.
   - Plan `## Revisions`: note that Task 5's sub-item 3 was not performed even though the box is ticked, and that the caret-blink question was answered in `72c2bbfc`.

```findings
findings:
  - id: new
    severity: Important
    family: done-when-waived-without-revision
    title: |
      M1 Done-when "pair term under plain zellij smoked + zellij 2026 honour recorded" unmet, issue not revised
    detail: |
      The Log says plain-zellij pair term was not smoked and zellij's 2026 handling is unrecorded, and all live pair term panes ran the pre-M1 binary. The plan Task 5 box is ticked anyway. "Harmless either way" is unsupported: under plain pair, Emit's full-pane erase+repaint can still be caught mid-way if zellij ignores 2026 from a pane. Run the operator smoke or add an issue Revisions entry that re-scopes the bullet.
  - id: new
    severity: Minor
    family: unverified-existing-code-claim
    title: |
      Stale-prose sweep missed "both consumers" siblings (hostty/reserve.go:51-55, couchnestedrows/main.go:4-14)
    detail: |
      reserve.go:52 still says both consumers assert the region and draw the row. The probe header still describes couch and pair term as reserving rows, in present tense. The new reserve.go:15-16 says the region reset is owned "not [by] this file" while this file's Release still writes it for the probe.
  - id: new
    severity: Minor
    family: plan-record-stale-after-log
    title: |
      Plan execution note says caret blink is still open (answered in 72c2bbfc); Task 5 is ticked over an unperformed sub-item
```

---

## Re-review — 2026-09-17T20:24:17-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 262 — Screen flicker: the compositor re-emits global terminal state every frame (#255) |
| repo | pair |
| issue file | workshop/issues/000262-diagnose-input-screen-flicker.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | a1b0a7d8e53f878b1ce6b543c2f6e4fead4c970b..b28ee73c94246976b928ad71dfcfcea895b6376b |
| command | sdlc milestone-close --issue 262 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-17T20:24:17-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All three round-1 findings are addressed and the M1 code holds up. Both renderers now open every frame with the 2026 begin marker and close it at the end. `parentReleaseControls` closes sync after the CAN/ST abort. The shared `cursorEpilogue` produces the same bytes as the two blocks it replaced. I extracted the head commit into a scratch copy and ran the `terminal` package: it passes, including all six bracket and sweep tests. Then I removed `syncEnd` from `parentReleaseControls`, and `TestPresenterReleaseClosesSyncAfterAnyCutWrite` failed in all three frame layouts. So the sweep does catch the bug it exists for. For BR-1, the Revisions entry now re-scopes Done-when bullets 2 and 5 and gives the reason for each. The plain-zellij question is answered by a kept measurement (`sync_hold.py`), and its verdict logic has unit tests. I could not rerun that measurement live: this environment blocks `/tmp` and pty allocation. Its logic is sound on reading. That leaves one Minor issue in the new script's exit codes; it does not block.

**1. Strengths**
- The bracket lives in the pure renderers: `render.go:61`/`:110`, and `history_render.go:307`/`:407`. The ALT packet stays whole, because `packet()` flushes the pending begin marker as its own write first (`history_render.go:202-207`). That keeps `Presenter.write`'s exact-match alt tracking (`presenter.go:196-201`) correct. The test checks it at `history_render_test.go:92-107`.
- The cut sweep runs the real `Presenter` against `ttyio.Fake` step scripts, so the failure ordering is controlled and repeatable (ARCH-ORDER). It learns write boundaries with `recordingParent` instead of hard-coding them.
- `sync_hold.py` measures the right thing: what zellij sends to its client, not `dump-screen`, which the docstring notes would update either way. The unbracketed control makes a "never arrived" result inconclusive, not a pass.
- The widened prose sweep searched phrases, not just symbols. My own re-sweep at head (`SafeToPaint|TakeRowDirty|ReserveAndPaint|paneWriter|only here|both consumers|row-dirty|…`) found only correctly framed text: the historical atlas note, `reserve.go`'s own API, and `screen.go:228`, which the new type-level status note covers.

**2. Critical findings:** none.

**3. Important findings:** none.

**4. Minor findings**
- `tests/terminal-oracle/discovery/sync_hold.py:137-142`: any uncaught exception exits with code 1, which the docstring and README define as "NOT honoured". I got two different environment crashes here: `/tmp` not writable at `:37`, and "out of pty devices" at `:71`. Both exited 1. A slow zellij start would do the same: the pane must launch within about 1s of `Popen`, or reading `t_write`/`t_end` at `:104-105` raises `FileNotFoundError`. Fix: catch setup/IO failures and exit 2 (inconclusive) with the error printed, and treat a missing `t_write`/`t_end` as inconclusive.
- Nit: the new Revisions entry says "Bullet 4's sweep was widened in round 2 (BR-2)", but the Log entry that records it is titled "round 1". Separately, the issue's M1 Plan row still says "Operator smoke under couch and `pair term`". Both Revisions entries cover the substitution, so I'm not raising this as a finding.

**5. Test coverage notes**
- Done-when bullet 1 (Render first paint / one-cell / cursor-only / no-op): covered. Emit reset, history append, steady change, idle, alt enter and alt leave (each now also checked with `assertOneBracket`), and the multi-chunk frame: covered. The presenter stream is covered by `TestPresenterStreamIsBracketedAcrossPaintsAndRelease`.
- Done-when bullet 2: every offset is cut for the single-write and alt-switch layouts. The 321 KB layout is cut at boundaries ±2, the first and last 32 bytes, and a 4093-byte stride, as the Revisions entry re-scopes it. The mutation confirms the test goes red.
- In the scratch run, `hostty`'s `TestOSHostOwnsFlagsThroughMeasurementAndJoinedRead` failed with "operation not permitted". This is the known sandbox pty block, and `hostty` has only comment changes in this range.

**6. Architecture (per marker)**
- ARCH-DRY, pass: `cursorEpilogue` replaces two copies with identical bytes, and `syncBegin`/`syncEnd` are the one source for both renderers, release and all assertions.
- ARCH-PURE, pass: the bracket is emitted by pure functions tested without IO. The presenter change is a pure bytes function.
- ARCH-PURPOSE, pass. I checked every parent write site (`presenter.go:131,331,346,352,709,825`). All visible drawing goes through Render or Emit and is bracketed. The mode delta and the OSC title/clipboard/notify effects draw nothing, and release closes the bracket.
- ARCH-MOCK, pass: `ttyio.Fake` is the stateful parent double behind the production `Presenter.write` seam, and `sync_hold.py` is the live zellij conformance check. See the Minor finding for its exit contract.
- ARCH-CONSTRAINTS, pass: the bracket opens and closes inside one synchronous `Emit`. Each write is bounded by `WriteTimeout`, and after a failure the terminal's own sync timeout limits the freeze. The cost is 16 bytes per frame.
- ARCH-SECURE, pass: no new untrusted parsing. The probe isolates XDG, socket and data dirs in a tempdir and strips the `ZELLIJ_*` variables, so it can't attach to the operator's session.
- ARCH-ORDER, pass: nothing new is carried between events, and the failure-at-any-prefix sweep drives the production presenter under scripted orderings.
- ARCH-FUNERAL, pass: the probe's temp dirs are removed automatically and it kills its session in `finally`. Nothing durable is created.
- For M2: `cursorEpilogue(c Cursor)` is the single site for the DECSCUSR decision, and widening it to `(prev, next)` is local. The "terminal default cursor style" finding (Shape is never 0) needs a change at the vt level, not just in the epilogue.

**7. Plan revision recommendations:** none. The Core-concepts table matches the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Revisions entry re-scopes bullets 2 and 5 with reasons; the zellij-honours-2026 question is answered by the kept sync_hold.py measurement, whose verdict logic is unit-tested (I could not rerun it live: pty/tmp blocked here).
  - id: BR-2
    disposition: addressed
    note: |
      hostty/reserve.go:13-19 and :52-57, the couchnestedrows header and runOuter doc are fixed; my phrase-vocabulary re-sweep at head finds only correctly framed text.
  - id: BR-3
    disposition: addressed
    note: |
      Plan Revisions entry (BR-3) corrects the caret-blink note (answered in 72c2bbfc) and states the Task 5 tick means question answered, not smoke run.
findings:
  - id: new
    severity: Minor
    family: instrument-failure-encoded-as-verdict
    title: |
      sync_hold.py exits 1 ("NOT honoured") on any crash, so an environment failure reads as a zellij verdict
    detail: |
      Uncaught exceptions exit 1, the code the docstring and README assign to "NOT honoured". Reproduced here twice (/tmp not writable at :37, "out of pty devices" at :71), and a zellij start slower than about 1s gives FileNotFoundError at :104-105. Catch setup/IO failures and exit 2 (inconclusive).
```
