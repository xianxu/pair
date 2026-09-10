# Boundary Review — pair#209 (whole-issue close)

| field | value |
|-------|-------|
| issue | 209 — thread switch leaves the screen partially redrawn |
| repo | pair |
| issue file | workshop/issues/000209-thread-switch-leaves-the-screen-partially-redrawn.md |
| boundary | whole-issue close |
| milestone | — |
| window | 11e12276c7602d583d78fd1348a2bdd95fb4fa1f..3fa15d432457de4797080c866497c1372579c778 |
| command | sdlc close --issue 209 |
| reviewer | claude |
| timestamp | 2026-09-09T21:16:33-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The composition half of this fix is the strongest work in the diff: `hostty.Repaint` is genuinely pure, its ordering rationale is written down where the next reader will find it, and its four unit tests pin the three things the plan gate spent four rounds establishing. The reproduction suite is real and deterministic. What blocks a clean SHIP is the *request* half — the part that actually fixes the operator's report. Measured, not inferred: deleting `c.requestRepaint(p.child)` from `couchtty/console.go:504` turns **nothing** red (I ran the couchtty suite in a scratch copy with and without it — identical failure sets, both only the documented pty-sandbox class). Deleting the `ChildModes` plumbing at either consumer likewise turns nothing red. The live conformance probe measures a resize sequence with a 1.5 s gap that production does not issue. `removeTab` lost its unconditional clear and has no nudge to recover with. And the nudge itself is copy-pasted into two packages, which is the one thing the plan's step 4 said not to do.

## 1. Strengths

- **`hostty/repaint.go:1-72` — pure, and the doc comment is the design record.** The three constraints (buffer before clear, observed-only assertion, cursor-save is a category error) are stated with *why each earlier version was wrong*. `ARCH-PURE` passes cleanly: `repaint_test.go` runs with no IO, no fakes, no terminal.
- **`ptychild/screen.go:39-45` — `altScreenObserved` sits beside `mouseObserved` with the distinction written down.** The comment explaining *when* absence is admissible (a `Screen` scanning from `Start` vs. a bounded ring) is the sentence that stops the next reader from re-deriving `#196` a third time. `classify` latches it on RIS, `?1049`, and `?47`/`?1047` — I checked all three arms.
- **`termcmd/run.go:1704-1714` — `redrawTab` / `clearTab` as two doors.** PQ-1's rule is implemented in the type, not in a comment, and `TestTabSwitchIssuesARepaintRequestAndRestoresTheSize` is verified-by-mutation: dropping `requestChildRepaint` gives `resizes = []` and the test fails with the message it promises. That is the one claimed fix in this diff that is properly pinned.
- **`#196`'s reattach test passes unmodified** — `TestAReattachedChildKeepsItsTrackingMode` green, and no test of it was touched. Done-when clause satisfied.
- **`probes/zellijrepaint/main.go` distinguishes `PROBE-INCONCLUSIVE` from a verdict** (pair#208's lesson), and `main` is two lines because `os.Exit` skips defers. Good probe hygiene.

## 2. Critical findings

**C1 — `couchtty`'s repaint request is pinned by no test at all (`termcmd`'s is).** `console.go:504`.
Measured in a scratch copy: `perl`-removing `c.requestRepaint(p.child)` leaves the couchtty suite with an identical failure set (only `TestNotificationPTYConformance`, the sandbox class). Same for replacing `hostty.ChildModes{AltScreen: altScreen, AltScreenObserved: observed}` with `hostty.ChildModes{}` at `console.go:498`, and same for dropping `modes.AltScreen, modes.AltScreenObserved = tab.child.RepaintModes()` at `run.go:1333`. So of the four wiring decisions this diff makes, exactly one — termcmd's nudge — is defended. couch *is* the operator's report.
Fix: a couchtty test mirroring `TestTabSwitchIssuesARepaintRequestAndRestoresTheSize` (the fixture already has `f.child.Resizes()`), plus the differential row the plan promised — assert `couchtty` and `termcmd` emit **byte-identical** repaint output for the same `ChildModes`, not merely that both call `Repaint`.

**C2 — the conformance probe does not exercise the production sequence, so the load-bearing assumption is still unmeasured.** `probes/zellijrepaint/main.go:104-113` vs `couchtty/console.go:1012-1026` and `termcmd/run.go:1345-1357`.
The probe does `resize(23,80)` → `sleep 1500ms` → `resize(24,80)` → `sleep 2500ms`. Production does `Resize(rows-1)` immediately followed by `Resize(rows)` — two `TIOCSWINSZ` ioctls microseconds apart (`ptychild/child.go:206`, a bare `pty.Setsize`). Standard signals do not queue: zellij can receive a single `SIGWINCH`, read the winsize *after* both ioctls have landed, see its cached size unchanged, and re-render nothing. The `#209` Log states "The SIGWINCH assumption is confirmed against the real binary, not asserted" — the probe confirms a *different* interaction than the one that shipped. The in-process fake cannot catch this either: `Child.Resize` on a fake just appends to `fake.resizes`, so `resizes = [23x80, 24x80]` is green whether or not a real child would repaint. This is precisely the shape the review brief names — "a field set at zero call sites… passes every test suite while doing nothing, and reads as protection."
Fix: make the probe issue shrink-then-restore back-to-back (no sleep between the two ioctls), re-run, and record the result. If it comes back `NOT REPAINTED` / `PARTIAL`, the nudge needs a settle delay or a different mechanism. I could not run it here — `pty.Open` returns `operation not permitted` in this sandbox (the documented class), which is also why this is a finding rather than a measurement.

**C3 — `removeTab` lost its unconditional clear and has no repaint request to recover with.** `termcmd/run.go:1429-1433`.
`applyTakeover` used to write `hostty.HomeAndClear` unconditionally (`run.go:982` pre-diff). It now routes through `Repaint(modes, replay, RepaintReplace)`, which emits **zero bytes** when the replay is empty. `removeTab` passes `clear=false`, and unlike `switchRelative` it issues no nudge. Failure scenario: operator hits Alt+T (tab 2 spawns), Alt+Left back to tab 1, Alt+W to close tab 1 — `replaySnapshotLocked(active)` for the just-created tab 2 returns nil before its child's first output is retained, so `Repaint` writes nothing and the **closed tab's screen stays on the pane indefinitely** (only the strip repaints, via `paintStripInline`). The "a stale frame beats a blank one" rationale is sound for a switch backed by a nudge; it is not sound for a frame belonging to a tab that no longer exists. Same window opens when a tab's whole retained tail is stripped queries.
Fix: `removeTab` is a deliberate replacement — pass `clear=true`, or issue `requestChildRepaint(active, size)` on that path too. Prefer the former; the destroyed tab's pixels should not survive the close under any child type.

## 3. Important findings

**I1 — `ARCH-DRY`: the nudge is copy-pasted into two packages, contradicting the plan's own "prefer one answer over two."** `couchtty/console.go:1012-1026` and `termcmd/run.go:1345-1357` are the same five lines (nil guard, `Rows < 2` guard, `Rows--`, `Resize`, `Resize`) with different receivers. Plan step 4 named `hostty` as the shared home *specifically* so the fix would exist once; only the composition landed there, and the mechanism — the actual answer to mode 1 — did not. These will drift: a settle delay added for C2, or a coalescing rule, has to be written twice and one site will be missed.
Fix: `func (c *Child) RequestRepaint(size Size)` on `ptychild.Child` (needs no `hostty` constants, so no cycle), or `hostty.RequestRepaint(child, size)`. Both consumers then call one thing.

**I2 — `ARCH-ORDER`: the composed mode assertion is written to the host but never fed back to `hostScan`, so the scanner's model and the terminal disagree.** `couchtty/console.go:1036` writes `hostty.Repaint(modes, body, ...)` but `console.go:1050` feeds only `body`; `termcmd/run.go:996` writes `Repaint(...)` but `run.go:1000` feeds only `replay`. The `?1049h` prefix exists *precisely because* the body does not contain it (it aged out of the ring), so this is the common case on every couch switch, not an edge. Failure scenario: zellij's retained tail contains `?1048h` (cursor save) but not the `?1049h` from startup. The host is put on the alt buffer by our prefix; `hostScan` records `cursorSaved=true, altScreen=false`; `SafeToPaint()` (`screen.go:215`) is then `!(true && !false)` → `false` **for the rest of the session** — BR-79's frozen strip, arriving by exactly the road BR-82's comment at `console.go:1040-1049` was written to close. PQ-3 warned about this ("composed mode-assertion bytes will reach `SafeToPaint`'s alt-screen carve-out (BR-79/82)") and the composition shipped without answering it.
Fix: feed the composed bytes, not the body — `out := hostty.Repaint(...); c.host.Write(out); c.hostScan.FeedFraming(out)` at both sites. (`HomeAndClear` is framing-complete, so feeding it is harmless; if the `rowDirty` side effect is unwanted, feed the assertion prefix and the body but not the clear.)

**I3 — `ARCH-ORDER`: in `termcmd` the nudge's shrink/restore pair races the host-resize path, leaving the child permanently mis-sized.** `run.go:1355-1356`. `requestChildRepaint` runs on the **input** goroutine (`handleTerminalChord` → `mux.nextTab()`, `run.go:575`; the `ptyWriter` doc at `run.go:346-350` states the input goroutine is where these arise), while `resizeAll` runs on the **writer** goroutine via `resizeThroughWriter` → `inheritSize`. Interleaving: input reads `size = S1`; input calls `Resize(S1.Rows-1)`; writer handles a host resize and calls `Resize(S2)`; input calls `Resize(S1)` — the child is left at the stale size with no further event to correct it. This is the hazard PQ-4 named ("otherwise the pane is left permanently mis-sized… what happens when a host resize races a switch"), disposed `addressed` on the strength of "the caller already owns the size" — but owning the value is not serializing the write. couch is safe here (both `onResize` and `switchTo` run on the Run goroutine); termcmd is not.
Fix: route the nudge through `m.enqueue(ptyChunk{onWriter: ...})` like `resizeThroughWriter` does, so the two resize writers serialize on one loop. That also gives the seam an ordering test needs — today no test can inject this interleaving, which is the at-review flag `ARCH-ORDER` calls highest-leverage.

**I4 — `couchtty`'s menu takeover is documented as a deliberate clear and wired as a replace.** `console_menu.go:194-196` comments "couch's OWN surface, not a child's: a deliberate clear", but `takeOverScreen` hardcodes `hostty.RepaintReplace` (`console.go:1036`) — couch has one door where termcmd got two. `ARCH-PURPOSE`: the family `absent-data-is-not-intent` was enumerated for `redrawTab(nil)` and swept at exactly one of the four takeover call sites. The enumeration is: `switchTo` (replace, nudged — correct), `showMenu` (deliberate, wired as replace), `termcmd/newTab` (deliberate, fixed), `termcmd/removeTab` (deliberate, wired as replace — C3). Fix the class: give `takeOverScreen` an intent parameter and pass `RepaintClear` from `showMenu`.

**I5 — the plan's "counted invariant into `#204`" is ticked `[x]` and was not delivered.** Done-when: "A counted invariant lands in `#204`: a switch issues a repaint request, not only a replay write." `workshop/issues/000204-*.md` is untouched in this window (`git diff --stat` over `workshop/issues/` shows only `000209` and `000221`), its invariant table at line 43-47 still lists only `#201`/`#202`/`#203`, and the phrase appears nowhere in the repo outside `#209`'s own file. `run_test.go:1332` *calls itself* "#209's counted invariant, and #204's", but naming it is not landing it.

**I6 — two Plan test rows are ticked but only partly delivered.** (a) "the four `replay_insufficiency_test.go` reproductions become regression rows against the new path — each asserts the mode it demonstrates is now handled" — they did not; all four remain pure reproductions ending in `t.Log`/`t.Logf` with no assertion against `hostty.Repaint` or the nudge. (b) "*Both consumers:* a differential row … assert the ANSWER, that `couchtty` and `termcmd` produce byte-identical repaint output for the same child state, not merely that both call the function" — no such test exists (grep for `hostty.Repaint` across `*_test.go` returns nothing). This is the same measurement as C1 from the plan's side.

## 4. Minor findings

- `couchtty/console.go:987-1010`: `requestRepaint` was inserted **between** `takeOverScreen`'s doc comment and its func, so godoc now attributes "takeOverScreen replaces what is on the screen wholesale…" to `requestRepaint`, and `takeOverScreen` (`console.go:1028`) has no doc at all. Move the comment back.
- `termcmd/run_test.go:1365`: `t.Errorf("restored to %v, want the child'+chr(39)+'s own size %v", ...)` — a botched shell/heredoc escape left literal `'+chr(39)+'` in the failure message.
- `ptychild/child.go:283-289`: `Child.AltScreenObserved()` has **zero callers**, production or test — `RepaintModes()` is what both consumers use. Newly-added dead exported surface; `#192` is the standing issue for exactly this class.
- `couchtty/console.go:498` dereferences `p.child` unguarded while `requestRepaint` six lines later guards `child == nil`. Pre-existing exposure (`ReplayThrough` did the same), but the two now disagree about whether `p.child` can be nil.
- `ARCH-CONSTRAINTS`: PQ-7 was disposed with "two SIGWINCHes cost one extra repaint." The probe measured **19,317 bytes** for a *single-pane* session; couch runs 10+ panes and a rows-only resize reflows the whole zellij layout. On a keystroke path. Worth a number in `#204` alongside I5's invariant rather than a prose assurance.

## 5. Test coverage notes

`hostty/repaint_test.go` is the model of what the rest should look like: four tests, no IO, each pinning one stated rule, and `TestRepaintCarriesIntentRatherThanInferringItFromAnEmptyReplay` asserts *both* halves of the intent distinction. `TestTabSwitchIssuesARepaintRequestAndRestoresTheSize` is verified-by-mutation and checks the right three things (count 2, rows shrink-then-restore, cols untouched).

The gap is entirely at the **wiring**, and it is measurable rather than theoretical: three of the four consumer-side decisions this diff makes survive deletion with a green suite (C1). `ARCH-MOCK` is half-satisfied — the fake models "a resize was requested", which is not the behaviour the fix depends on ("a resize produces a full repaint"), and the live half measures a sequence production does not issue (C2). No test can currently observe the input-goroutine/writer-goroutine interleaving in I3.

Sandbox note: `ptychild`, `hostty`, `termcmd` and `couchtty` all have pre-existing `operation not permitted` failures on pty-spawning tests in this environment; I treated those as the documented class and confirmed identical failure sets across every mutation, so the mutation results are not confounded by them.

## 6. Architectural notes for upcoming work

- **ARCH-DRY** — flag, I1. The composition is shared; the mechanism is not.
- **ARCH-PURE** — pass. `Repaint` is a pure function returning bytes, tested with no terminal; both IO shells are thin. Exactly what the plan promised.
- **ARCH-PURPOSE** — flag, I4 (class enumerated, swept at one of four sites) and I5/I6 (three ticked Plan rows delivered short of what they claim). The purpose of `#209` is couch's switch; couch's half is the unpinned one (C1).
- **ARCH-MOCK** — flag, C2. Fake and live conformance check both model something adjacent to what production does.
- **ARCH-CONSTRAINTS** — pass with a note (§4). Two ioctls on a keystroke path is cheap; the downstream reflow across a 10-pane session is the number nobody has.
- **ARCH-SECURE** — pass, `N/A` on secrets. The one new host-bound byte sequence is derived from a parsed mode bit, not from raw child bytes, and the replay path's existing query-stripping is untouched.
- **ARCH-ORDER** — flag, I2 (scanner state diverges from what was written) and I3 (two resize writers on two goroutines, no seam to inject the interleaving). Both are the "state carried between events" lens, not the single-parse one.

Forward: when C2 is answered, whatever settle rule the real binary requires belongs in the single shared `RequestRepaint` from I1 — writing it twice is how the two consumers diverge for the third time (BR-77, BR-82 are the prior two, both cited in this diff's own comments).

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/issues/000209-...md` covering:

1. **Tests row, "*Shared repaint:*"** — restate what shipped: the four `replay_insufficiency_test.go` rows are **reproductions only**, and the regression surface is `hostty/repaint_test.go` plus `TestTabSwitchIssuesARepaintRequestAndRestoresTheSize`. Either convert them as promised or say plainly that they were not converted and why.
2. **Tests row, "*Both consumers:*"** — the differential byte-identity row was not written. Either write it (it also closes C1) or record it as deferred with its reason.
3. **Tests row, mode-assertion cross-product** — still lists `{alt-screen, mouse, SGR-mouse, cursor-save} × {set, unset}`; the design asserts alt-screen only. This is open finding **PQ-12** from the plan gate, never disposed. Drop the three fields that emit nothing.
4. **Step 4 item 3** — claims `altScreenObserved` **and** `sgrMouseObserved` were added; only `altScreenObserved` exists (`mouseObserved` already covers both mouse fields, as PQ-9 itself noted). Correct the text.
5. **Step 4 composition line** — reads "assert buffer → clear → tail → **buffer-independent modes** → repaint request". No buffer-independent modes are emitted; `Repaint`'s doc explains why (mouse would be a third writer). Bring the plan's composition in line with `hostty/repaint.go:56-72`.
6. **Done-when, `#204` clause** — either land the row in `#204`'s invariant table or revise the clause. Ticked-and-absent is the state that makes a ledger stop meaning anything (I5).

```findings
findings:
  - id: new
    severity: Critical
    family: claimed-fix-unpinned-by-test
    title: |
      couch's repaint request and both consumers' ChildModes wiring are pinned by no test — deleting them leaves the suite green
    detail: |
      Measured in a scratch copy: removing c.requestRepaint(p.child) (console.go:504),
      or replacing the ChildModes at console.go:498 with the zero value, or dropping
      modes.AltScreen/AltScreenObserved at run.go:1333, each leaves the suite with an
      identical failure set (only the documented pty-sandbox class). Only termcmd's
      nudge is defended. couch is the operator's reported bug. Add a couchtty nudge
      test plus the differential byte-identity row the plan promised.
  - id: new
    severity: Critical
    family: external-behavior-unverified
    title: |
      The zellij conformance probe measures a resize sequence with a 1.5s gap that production never issues
    detail: |
      probes/zellijrepaint/main.go:104-113 sleeps 1500ms between the shrink and the
      restore; production issues both TIOCSWINSZ ioctls back-to-back (console.go:1023-1025,
      run.go:1353-1356, via a bare pty.Setsize at child.go:206). Standard signals do not
      queue, so zellij may take one SIGWINCH, read an unchanged winsize, and re-render
      nothing — the fix would be a no-op while the fake (which only records resize calls)
      and the recorded probe are both green. Make the probe drive the production sequence
      and re-record the verdict.
  - id: new
    severity: Critical
    family: absent-data-is-not-intent
    title: |
      removeTab lost its unconditional clear and has no nudge, so a closed tab's screen can persist
    detail: |
      applyTakeover used to write HomeAndClear unconditionally; it now routes through
      Repaint(..., RepaintReplace), which emits zero bytes on an empty replay.
      run.go:1433 passes clear=false and, unlike switchRelative, issues no repaint
      request. Alt+T, Alt+Left, Alt+W closes tab 1 while tab 2's ring is still empty:
      nothing is written and the destroyed tab's content stays on the pane. Pass
      clear=true — a takeover replacing a tab that no longer exists is deliberate.
  - id: new
    severity: Important
    family: shared-mechanism-duplicated
    title: |
      The resize nudge is copy-pasted into couchtty and termcmd, contradicting the plan's "one answer, not two"
    detail: |
      console.go:1012-1026 and run.go:1345-1357 are the same five lines with different
      receivers. Plan step 4 named a shared home so the fix would exist once; only the
      composition landed in hostty. Any settle delay or coalescing rule added later has
      to be written twice. Lift it to Child.RequestRepaint(size) on ptychild (no hostty
      constants needed, so no import cycle).
  - id: new
    severity: Important
    family: scanner-must-see-what-was-written
    title: |
      The composed alt-screen assertion is written to the host but never fed back to hostScan
    detail: |
      console.go:1036 writes Repaint(...) but console.go:1050 feeds only body; run.go:996
      writes Repaint(...) but run.go:1000 feeds only replay. The ?1049h prefix exists
      because the body lacks it, so this is the common case on every couch switch. With a
      ?1048h in the tail, hostScan then holds cursorSaved=true, altScreen=false and
      SafeToPaint() is false for the session — BR-79's frozen strip by the road BR-82's
      comment was written to close, and the hazard PQ-3 named. Feed the composed bytes.
  - id: new
    severity: Important
    family: nudge-ordering-and-extent
    title: |
      In termcmd the nudge's shrink/restore races resizeAll across two goroutines
    detail: |
      requestChildRepaint runs on the input goroutine (handleTerminalChord -> nextTab,
      run.go:575) while resizeAll runs on the writer goroutine (resizeThroughWriter ->
      inheritSize). A host resize landing between the two Resize calls is overwritten by
      the restore leg, leaving the child at the stale size with no event to correct it —
      the "permanently mis-sized" hazard PQ-4 named. couch is safe (both on the Run
      goroutine); termcmd is not. Route the nudge through m.enqueue({onWriter: ...}),
      which also gives an ordering test a seam.
  - id: new
    severity: Important
    family: absent-data-is-not-intent
    title: |
      couch's menu takeover is documented as a deliberate clear but wired as a replace
    detail: |
      console_menu.go:194-196 says "a deliberate clear"; takeOverScreen hardcodes
      RepaintReplace at console.go:1036. termcmd got two doors (redrawTab/clearTab);
      couch still has one. The class enumeration is four takeover call sites — switchTo,
      showMenu, newTab, removeTab — and only newTab was swept. Give takeOverScreen an
      intent parameter and pass RepaintClear from showMenu.
  - id: new
    severity: Important
    family: plan-row-ticked-not-delivered
    title: |
      Three ticked Plan rows are not delivered: the #204 counted invariant, the four regression conversions, and the differential row
    detail: |
      workshop/issues/000204-*.md is untouched in this window and its invariant table
      still lists only #201/#202/#203; the phrase appears nowhere outside #209's own
      file. The four replay_insufficiency_test.go rows remain pure reproductions ending
      in t.Log with no assertion against the new path. No test compares couchtty's and
      termcmd's repaint output for byte identity. All three rows are marked [x].
  - id: new
    severity: Minor
    family: godoc-attached-to-wrong-symbol
    title: |
      requestRepaint was inserted between takeOverScreen's doc comment and its func
    detail: |
      console.go:987-1010 now documents requestRepaint with takeOverScreen's paragraph,
      and takeOverScreen at console.go:1028 has no doc at all.
  - id: new
    severity: Minor
    family: escaping-artifact-in-source
    title: |
      A botched shell escape left literal '+chr(39)+' in a test failure message
    detail: |
      termcmd/run_test.go:1365 — "want the child'+chr(39)+'s own size %v".
  - id: new
    severity: Minor
    family: dead-exported-surface
    title: |
      Child.AltScreenObserved has zero callers, production or test
    detail: |
      ptychild/child.go:283-289. Both consumers use RepaintModes(). Newly-added dead
      exported surface — the #192 class.
  - id: new
    severity: Minor
    family: operating-envelope-unstated
    title: |
      The nudge's cost is stated as "one extra repaint" but the probe measured 19,317 bytes on a single-pane session
    detail: |
      A rows-only resize reflows the whole zellij layout, and couch runs 10+ panes, on a
      keystroke path. PQ-7 was disposed on prose; the number belongs in #204's invariant
      table alongside the repaint-request count.
```

---

## Re-review — 2026-09-09T22:22:28-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 209 — thread switch leaves the screen partially redrawn |
| repo | pair |
| issue file | workshop/issues/000209-thread-switch-leaves-the-screen-partially-redrawn.md |
| boundary | whole-issue close |
| milestone | — |
| window | 11e12276c7602d583d78fd1348a2bdd95fb4fa1f..c4ee363f24718b58f0c4f24f70f82093f3cca18f |
| command | sdlc close --issue 209 |
| reviewer | claude |
| timestamp | 2026-09-09T22:22:28-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

This round did the hard part well: the settle is a *measured* number rather than a guessed one, the probe's own table is what caught a fix that worked half the time, and the godoc guard was generalized to the class rather than the site (I mutation-verified it — inserting the BR-10 shape into `ptychild/child.go` now fails `TestNoDeclarationCarriesTwoStackedGodocs`). BR-2's couch half is genuinely pinned now (deleting `console.go:502` times out `TestSwitchAsksTheIncomingChildToRepaint`), and `HomeAndClear` is emitted from exactly one place in the tree. What blocks SHIP is two measured Criticals plus four prior findings that are still open. First: `probes/zellijrepaint` now defaults to `settle = 0` (`main.go:113`) while production settles 20 ms — so `make test-smoke`, which runs every `probes/*/` with `|| exit 1` (`Makefile.local:59-67`), now runs the sequence this round *itself* measured at **6 of 12**, and its comment at `main.go:107` still asserts "Production issues the pair with nothing between them", which stopped being true in the same commit. Second: couch's nudge (`console.go:502`) runs on the **operationQueue goroutine** for the operator's primary switch gesture, not the Run goroutine that owns `applyLayout` — the exact hazard BR-7 closed in termcmd, reopened in the consumer that is the operator's actual bug report. And three prior fixes (BR-4, BR-7, BR-8) revert silently: I reverted each in a scratch copy and the suite came back with the identical documented pty-sandbox failure set.

## 1. Strengths

- **`hostty/repaint.go` is the right shape and the right home.** Pure function, unit-tested with no IO, `RepaintFor` as the one thin adapter that touches a `*ptychild.Child`. The shadow-sweep is clean: `grep HomeAndClear` over non-test `cmd/` returns exactly one emitter (`repaint.go:77`) and four comments pointing at it.
- **The withdrawal is pinned as a withdrawal.** `TestRepaintEmitsNoCursorMovingBufferAssertion` (`repaint_test.go:16`) makes the `?1049` retreat un-reversible-by-accident, and the reason (the 1049 save slot aliasing DECSC, which the strip paints with) is recorded where the next author will hit it.
- **The godoc guard was generalized, not patched.** `stolenFrom` (`doccomment_test.go:101`) catches the *other* half of the family, and scope is derived by globbing `../*` rather than a hand-list. Mutation-verified: renaming `RequestRepaint`'s doc opener to `Resize` fails with a precise message naming both symbols.
- **`Child.RequestRepaint` has one home and one settle** (`ptychild/child.go:196-233`), with the measurement table in the doc comment where the next reader of the constant will find it. BR-5 is properly closed.
- **`#204`'s invariant row landed with an envelope, not a prose assurance** — 6.6 KB / 19.3 KB / 6-of-12 / 1-per-switch, and the honest note that a single-pane figure is a floor for a 10-pane couch.

## 2. Critical findings

**C1 — `probes/zellijrepaint` defaults to a sequence production no longer issues, and `make test-smoke` runs that default.** `probes/zellijrepaint/main.go:113` sets `settle := time.Duration(0)`; `main.go:107` still says "Production issues the pair with nothing between them". Production now sleeps `repaintSettle` (20 ms) between the two ioctls (`ptychild/child.go:227-229`). `Makefile.local:59-67` loops `go run ./probes/$p || exit 1` with no environment, so the standing conformance check runs at the settle this round measured at 6-of-12 and aborts the whole smoke suite on the ~50% of runs that come back NOT REPAINTED / PARTIAL. Fix sketch: single-source the settle instead of restating it. `atlas/index.md:34-44` already names the branch — a probe that must import `cmd/internal/…` belongs under `cmd/probes/` with its own target — so move it there and read `ptychild.repaintSettle` directly, keeping `PAIR_PROBE_SETTLE` as the override for re-measuring the table.

**C2 — couch issues the nudge off the goroutine that serializes child geometry.** `console.go:502` calls `p.child.RequestRepaint(c.ChildSize())` inside `switchTo`. `switchTo` has two callers' goroutines: the Run loop (`console.go:716`, `console.go:1385`) *and* the operationQueue goroutine — `ExecuteConsoleOperation` case `"switch"` (`console.go:1853`) is reached from `operationQueue.Run` → `request.run()` (`operation_queue.go:62`), wired at `couchcmd/run.go:456-461`, and `"switch"` is declared `Effect: EffectConsole` (`couchcore/ops.go:220`). That is the switcher's Enter and the status-chip click — the operator's primary gesture. Meanwhile couch's only other `child.Resize` is `applyLayout` (`console.go:953`), reachable solely from the Run goroutine (`console.go:536`, `console.go:1283`). So a SIGWINCH landing inside the 20 ms shrink window resizes the child to the new size and the restore leg then writes back the size sampled before it — the child is permanently mis-sized with no event to correct it. This is what `RequestRepaint`'s own doc ("callers must stay on the goroutine that serializes their other resizes", `child.go:214-216`) and `atlas/architecture.md:485-488` both promise does not happen. This is the 2nd finding in family `nudge-ordering-and-extent`; per the escalation, the deliverable is the rule, not this site: **the nudge must run on, and read its restore size from, the goroutine that owns `Child.Resize` for that child — and that ownership must be enforced by the type, not asserted in a doc comment.** The enumeration is every `RequestRepaint` caller (2) crossed with its consumer's resize owner, and both instances fail it: couch calls it from two goroutines, and termcmd (below) samples the size on a third.

## 3. Important findings

**I1 — the class named by `claimed-fix-unpinned-by-test`, with measured prevalence.** This is the 2nd finding in that family. Do not fix the three instances individually; the rule is already written in this gate's own prompt and was violated again by the commit that closed the findings: **a disposition of `addressed` requires a test that goes red when the fix is reverted, and producing that test is part of the fix, not a follow-up.** Measured prevalence this round: of the seven prior findings whose fix changed production code, three revert silently — BR-4 (`run.go:1448` `true`→`false`), BR-7 (`run.go:1371` enqueue→direct call), BR-8 (`console_menu.go:196` `RepaintClear`→`RepaintReplace`) — each leaving the identical documented pty-sandbox failure set. Two are pinned by construction (BR-5, BR-12: the symbol moved, so a revert does not compile), one by a new guard (BR-10), one is byte-inert today and honestly unpinnable (BR-6). The enumeration that should be written into the plan is exactly the round's own addressed-list; running the revert over it is mechanical and would have caught all three.

**I2 — `removeTab` hands the screen to a different child and asks for no repaint.** `run.go:1448` takes the screen over for the surviving tab but issues no `RequestRepaint`; only `switchRelative` (`run.go:1371`) got the nudge. Close tab 1 while tab 2's ring holds no full frame and the operator gets the issue's own symptom — cleared screen, partial content, no way to recover but to press Enter. BR-8's enumeration ("four takeover call sites") was correctly swept for *intent*; the same enumeration was not swept for the *repaint request*. Fix sketch: enqueue the same `nudge` chunk for `childOf(active)` after the takeover, or better, make the nudge a property of the takeover chunk so a site cannot do one without the other.

## 4. Minor findings

- **`hostty.EnterAltScreen` (`control.go:63`) has zero production callers** and its doc claims a rationale ("a repaint must put the paint in the buffer the child is actually using") withdrawn in the same commit; only `repaint_test.go:23`'s forbidden-list references it. 2nd in `dead-exported-surface` — the rule is that a constant added *for* a mechanism must land with it, not ahead of it; if it is scaffolding for the `?1047` attempt, say so in the doc or hold it until the probe runs.
- **`ptyChunk.nudge` / `nudgeSize` (`run.go:713-714`) duplicate the existing `onWriter` seam** (`run.go:727-730`), which BR-7 named explicitly and which already runs before the switch without consuming the chunk. `clear bool` (`run.go:704`) likewise re-spells the `RepaintIntent` enum this same commit introduced, then converts back at `run.go:1013-1016`. 2nd in `shared-mechanism-duplicated` — rule: when a queue already has a "run this on the writer" affordance, route through it rather than growing a parallel field per caller.
- **The no-coalescing decision predates the settle.** The plan's "two SIGWINCHes cost one extra repaint, not a wrong screen" was costed when the nudge was free; it now blocks termcmd's writer goroutine and couch's single Run loop for 20 ms per switch. Held key-repeat tab cycling stalls the loop proportionally (~300 ms/s at a 15/s repeat) while the child's own reflow bytes queue behind it. 2nd in `operating-envelope-unstated` — rule: an envelope stated for one event must be restated when the per-event cost changes.
- `termcmd/run.go:1013-1016` builds `intent` from a bool inside `applyTakeover`; passing `hostty.RepaintIntent` end-to-end would remove the conversion and the field.

## 5. Test coverage notes

- Suite state: `hostty`, `ptychild`, `couchtty`, `termcmd` are green apart from the documented pty-sandbox class (`operation not permitted` on `pty.Open` / `start sh`), which matches the known environment limit and not this diff.
- `TestTabSwitchIssuesARepaintRequestAndRestoresTheSize` (`run_test.go:1336`) builds a mux with `output == nil`, so `enqueue` dispatches inline on the test goroutine (`run.go:1188-1191`). It therefore cannot observe *which* goroutine the nudge runs on — which is why reverting the queue hop is invisible. termcmd already has the seam for this (`captureIDForTest`, `run.go:1489`, added precisely because "testing the seam is not testing the path"); the nudge needs the same treatment. ARCH-ORDER's highest-leverage flag: both nudge tests are samples of size one.
- `TestRequestRepaintLeavesTheShrinkStandingLongEnoughToBeSeen` is a good pin on the settle, and asserting on `elapsed >= repaintSettle` rather than a literal keeps it honest if the constant moves.
- The transitive differential (hostty golden + `TestSwitchWritesExactlyTheComposedRepaint` + `TestTakeoverWritesExactlyTheComposedRepaint`) is a legitimate way to close the row given nothing can drive both consoles, and both legs assert real bytes with a vacuity guard. Accepted.
- BR-6's composed-bytes feed is byte-inert today (`HomeAndClear` is framing-complete) so no test can pin it; that is stated at both sites and is fine. It becomes pinnable the day `?1047` lands, and should be pinned then.

## 6. Architectural notes

- **ARCH-DRY** — flag (Minor): `ptyChunk.nudge`/`nudgeSize` vs `onWriter`; `clear bool` vs `RepaintIntent`. Otherwise a strong pass — one `HomeAndClear` emitter, one `RequestRepaint`.
- **ARCH-PURE** — pass. `Repaint` is deterministic and tested with no IO; `RepaintFor` is the thin read; the `*Child` only appears in the adapter.
- **ARCH-PURPOSE** — flag (I2). The single-source sweep for *intent* is complete; the sweep for the *repaint request* stopped at one of the two termcmd sites that change which child owns the screen.
- **ARCH-MOCK** — flag (C1). The in-process fake and the live probe both exist, which is the right pair; the live half's default no longer models production, and it runs unattended in `make test-smoke`.
- **ARCH-CONSTRAINTS** — flag (Minor). The envelope is declared and measured, which is a real improvement, but the settle blocks the one event loop and the coalescing policy was not revisited.
- **ARCH-SECURE** — pass. `PAIR_PROBE_SETTLE` is parsed with `time.ParseDuration` and fails loudly (`main.go:115-119`); no credentials, no new untrusted-input surface, no test reaching real user state.
- **ARCH-ORDER** — flag (C2, and the coverage note above). The state carried between events here is *child geometry*, and its legal transitions are currently spread across `applyLayout`, `resizeAll`, and `RequestRepaint` with the serialization rule living in a doc comment. Collapsing that into a single owner ("only this goroutine resizes children; the nudge asks it to") is the change that would make both instances unrepresentable rather than caught.

## 7. Plan revision recommendations

1. **Tests row, mode-assertion cross-product (BR-1, still open).** The 2026-09-10 revision claims delta #1 was applied; the Plan row at issue `:262-263` still reads `{alt-screen, mouse, SGR-mouse, cursor-save} x {set, unset}` verbatim. Deltas #2 and #3 *were* applied. Either apply #1 or withdraw it from the Revisions list — a revision entry that describes an edit nobody made is worse than the stale row, because it makes the row look already-fixed.
2. **Same row: "2 and 4 fixed".** The row claims mode 4 is fixed; `hostty/repaint.go:62-78` asserts no mode at all and the Log says "Mode 4 is therefore unfixed for now." Restate as "2 fixed; 4 tracked to the child's `Screen` but not yet asserted on the wire — see the `?1047` candidate."
3. **New Revisions entry for the probe default (C1).** The plan row on the conformance check says the probe "drives the production sequence". It drives a 0 ms settle by default. Record the delta and the single-sourcing decision (move to `cmd/probes/`, read `repaintSettle`).
4. **New Revisions entry for the nudge's goroutine contract (C2).** The plan asserts couch is safe because "both are on the Run goroutine". It is not — the menu path runs on the operationQueue goroutine. Record the correction and the rule that replaces the assumption.
5. **Nudge-coalescing row.** Re-cost "nothing is coalesced" now that a nudge costs 20 ms of the event loop, and state the bound for held key-repeat switching.

---

## Re-review — 2026-09-09T22:57:58-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 209 — thread switch leaves the screen partially redrawn |
| repo | pair |
| issue file | workshop/issues/000209-thread-switch-leaves-the-screen-partially-redrawn.md |
| boundary | whole-issue close |
| milestone | — |
| window | 11e12276c7602d583d78fd1348a2bdd95fb4fa1f..fe4548ba326a2c59e919c83f3295df69d62b94c2 |
| command | sdlc close --issue 209 |
| reviewer | claude |
| timestamp | 2026-09-09T22:57:58-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The two Criticals that returned REWORK last round are genuinely fixed and I verified both by reverting them in a scratch worktree: the probe now drives the production ioctl sequence and *reads* `ptychild.RepaintSettle` rather than restating it (C1), and `RequestRepaint` takes no size — unlocking `geom` across the settle turns `TestAResizeDuringANudgeWinsWhicheverGoroutineItComesFrom` red (C2). Every prior BR finding disposes `addressed`, and three of them (BR-2, BR-4, BR-10) are now defended by tests that go red on the revert — BR-10 by a *mechanical* guard that I confirmed fires on a synthetic recurrence. What blocks SHIP is one thing the mode-2 fix introduced: `RepaintReplace`'s "emit nothing when the replay is empty" is correct only when the frame already on screen belongs to the same child, and **not one of the five takeover call sites is that case** — so an empty replay now leaves the *outgoing* surface (the panel's own body, or the previous thread's screen) standing under the new thread's label, on the primary "start a thread from the panel" path. That is BR-4's shape at the sites BR-4 didn't name. Full suite is green modulo the documented pty-sandbox class; `-race` on all four touched packages is clean.

### 1. Strengths

- **`hostty/repaint.go` is a genuinely pure core.** `Repaint` returns bytes and writes nothing, so the ordering — the whole difficulty — is unit-testable with no terminal, and `RepaintFor` is a two-line IO-reading shell over it. Textbook ARCH-PURE.
- **BR-10's fix is a mechanism, not a habit.** `stolenFrom` (`cmd/internal/termcmd/doccomment_test.go:120`) globs `../*` so it covers `ptychild` too. I inserted a synthetic `func (c *Child) Interloper() {}` between `Resize`'s doc and its func and it failed with the exact diagnosis. That is the class fixed, not the instance.
- **The settle is measured and the measurement is reproducible.** `RepaintSettle` is exported *so the probe cannot restate it*, `PAIR_PROBE_SETTLE` re-measures the table, and the probe sits under `cmd/probes/` (its own Makefile target, not `test-smoke`) precisely so the condemned zero-settle sequence can't run unattended.
- **C2's answer deletes the parameter instead of documenting harder.** `Child` owning `geom` + `size` means no caller has an ordering obligation left to get wrong, and the honest note that the *first* version of the race test was green against the defect it names is the kind of thing that makes a ledger trustworthy.
- **`#204`'s envelope table** (`workshop/issues/000204-*.md:56-70`) carries the byte cost, the coin-flip rate, and the per-nudge event-loop cost — BR-13 answered with numbers.

### 2. Critical findings

**C-1 — an empty replay leaves the *outgoing* surface on screen; no takeover site wants that** (`cmd/internal/hostty/repaint.go:26-29`, `cmd/internal/couchtty/console.go:503`, `cmd/internal/termcmd/run.go:1714`)

> **This is the 3rd finding in family `absent-data-is-not-intent`.** Earlier rounds fixed instances (PQ-1 gave `newTab` its own door; BR-4 gave `removeTab` `RepaintClear`). Do **not** fix this instance — state the rule and fix that.

The rule the three findings share: **what an empty body means is a property of *whose frame is currently on the screen*, not of the call site's name.** `RepaintReplace`'s justification — "a stale frame beats a blank one" — holds only when the stale frame belongs to the child being repainted. Enumerate the five takeover sites and none is:

| site | screen currently shows | intent |
|---|---|---|
| `couchtty switchTo` (`console.go:503`) | previous thread, or the panel body | `RepaintReplace` |
| `couchtty showMenu` (`console_menu.go:196`) | a child | `RepaintClear` ✓ |
| `termcmd switchRelative` (`run.go:1363`) | the previous tab | `RepaintReplace` |
| `termcmd removeTab` (`run.go:1440`) | the destroyed tab | `RepaintClear` ✓ (BR-4) |
| `termcmd newTab` (`run.go:856`) | the previous tab | `RepaintClear` ✓ |

Failure scenario, on the primary flow: start a thread from the panel. `ExecuteConsoleOperation` attaches (`console.go:366` seeds `replayCutoff = child.ReplaySafeEnd()`) and dispatches `switch` on the same operationQueue goroutine, before `Run` has drained a batch to advance the cutoff. `ReplayThrough(cutoff)` returns nothing, `Repaint` returns zero bytes, `c.host.Write` writes nothing — and **the panel's menu body stays on the terminal** until zellij's first frame arrives, now labelled with the new thread. Before this diff it was `HomeAndClear` + nothing: blank, which is honest. A foreign frame under the new thread's label is the misleading version of the same wrong.

Fix the rule, not the row: make the enum name the question the site must answer (`RepaintKeepThisChildsFrame` vs `RepaintBlank`), and note in `repaint.go` that the keep-stale branch presently has **zero** correct callers — either give `switchTo`/`switchRelative` `RepaintClear` or delete the branch and re-derive it when a same-child repaint site exists.

### 3. Important findings

**I-1 — a mandatory `Resize` waits on the optional nudge; `#204` says it costs zero** (`cmd/internal/ptychild/child.go:277-290`)

> **This is the 2nd finding in family `operating-envelope-unstated`.** State the rule.

`nudge()` holds `geom` across `time.Sleep(RepaintSettle)`, and `Resize` takes the same lock. Measured in a scratch copy: **a concurrent `Resize` blocked 20.8 ms**. `termcmd`'s `resizeAll` (`run.go:1600`) runs on the **writer goroutine** — the sole writer of the pane — via `resizeThroughWriter`, so a SIGWINCH landing inside a switch stalls all pane output for a settle. `#204`'s table says `event-loop time per nudge | 0 — the settle runs off the caller's goroutine`, which is true only for the caller that *requests* the nudge.

The rule: **an envelope must be stated for every path the mechanism can block, not only the path that invokes it.** `#204` already writes the sibling rule one paragraph down ("an envelope stated for one event has to be restated when the per-event cost changes") — this is that rule applied to contention rather than to cost. Enumerate the `geom` contenders (`Console.applyLayout`, `terminalMux.resizeAll`, `Child.Resize` from teardown) and give each a row; or let `Resize` preempt an in-flight nudge (bump a generation, have the restore leg skip when superseded) so the optional work never gates the mandatory one.

**I-2 — `takeOverScreen` still claims "Run-goroutine-only, like every other writer", and this round's own test disproves it** (`cmd/internal/couchtty/console.go:996`)

> **This is the 2nd finding in family `nudge-ordering-and-extent`.** State the rule.

C2's whole discovery was that `switchTo` is reached from the operationQueue goroutine (`ExecuteConsoleOperation`, `console.go:1862`) as well as `Run` — and `TestASwitchThroughTheOperationQueueNudgesLikeAnyOther` drives exactly that. `switchTo` calls `takeOverScreen`, which writes `c.host` directly; `hostty.Host` is a bare `io.Writer` with no serialization, and couch has five unsynchronized write sites (`console.go:915,983,1009,1058`; `console_menu.go:193,199,202`). The nudge's ordering was fixed by mechanism; the *writer's* ordering is still a sentence.

The rule is the one C2 wrote: **a goroutine-ownership claim in a comment is not a mechanism.** `termcmd` already has the answer next door — `paneWriter` is deliberately *not* an `io.Writer` so "a door that skips this reasoning does not compile" (`run.go:993`). Either give couch the same typed single-writer door, or — cheaply, this round — delete the false sentence and file the divergence, because leaving it is what let C2's rule read as satisfied. (Same lens, smaller: `go c.nudge()` has no tie to `c.done`, so the goroutine outlives `Close` by a settle. Harmless today — both branches of `resizeLocked` error on a dead child — but it is unbounded extent with no cancellation path.)

**I-3 — the fake and the real `Child` disagree on initial geometry, which is the exact state the fix reads** (`cmd/internal/ptychild/fake.go:37-46` vs `child.go:129-135`)

`Start` records `size: opts.Size`, so a real child always has geometry and `RequestRepaint` always nudges. `NewFakeChild` sets no size, so a fake declines silently (`size.Rows < 2`) until someone resizes it. Every new test works around this, and `run_test.go:1341` rationalises it as "that is what production does" — but production *cannot* reach the zero-geometry state the fake starts in. `TestFakeAndRealChildAgreeAfterTheChildHasEnded` drives `Write`/`Resize`/`Signal` and does not cover `Size`/`RequestRepaint`.

ARCH-MOCK: the double must model the seam's state, or a test can be green because the fake declined where production would have nudged. Give `NewFakeChild` a starting size (or an options form that mirrors `Start`) and add `Size` + `RequestRepaint` to the conformance stimulus set.

### 4. Minor findings

- **`dead-exported-surface`, 2nd in family.** `hostty.EnterAltScreen` (`control.go:68`) has one consumer: a forbidden-list in `repaint_test.go` *in the same package* — it never needed exporting. `Child.Size()` (`child.go:346`) has zero production callers, two test ones. `hostty.Repaint` is reachable only through `RepaintFor` and its own package's tests. Don't unexport these three individually — the rule is "exported surface needs a consumer outside its own package's tests"; this repo already writes AST guards (`doccomment_test.go`), so the class fix is one more.
- `console_test.go:1056` gives the mutation recipe as "dropping `p.child.RequestRepaint(c.ChildSize())` from `switchTo`" — that signature and that call site both stopped existing in `fe4548ba`. A reader following it finds nothing to delete.
- `TestAResizeDuringANudgeWinsWhicheverGoroutineItComesFrom` (`replay_insufficiency_test.go:196`) sleeps `RepaintSettle/4` to let the spawned nudge take `geom`. If it hasn't (loaded machine, `-race`), the racing `Resize` lands first and `got[1].Rows != before.Rows-1` fails spuriously. Wait on `waitForResizes(t, child, 2)` instead — the test's own comment already teaches this lesson about the *other* party.
- `Repaint`'s capacity hint still budgets `len(LeaveAltScreen)` (`repaint.go:75`) for bytes the withdrawal guarantees it never emits.
- `nudge`'s floor comment (`child.go:283`) says "one row for the child plus the reserved row" — but `c.size` is already the child's size with the reservation subtracted, so the reserved row isn't in that number.

### 5. Test coverage notes

Mutation-verified this round (each revert done in a detached worktree at `fe4548ba`):

| revert | result |
|---|---|
| drop `child.RequestRepaint()` from `couchtty.takeOverScreen` | 2 tests red ("timed out waiting for the incoming child to be asked to repaint") |
| drop `child.RequestRepaint()` from `termcmd.applyTakeover` | 2 tests red, incl. `TestClosingATabAsksTheSurvivingChildToRepaint` |
| `removeTab` `RepaintClear` → `RepaintReplace` | `TestClosingATabBlanksTheDeadTabsScreenEvenWithNothingToDraw` red |
| drop `time.Sleep(RepaintSettle)` | red at 51.9 µs |
| unlock `geom` across the settle (pre-C2 shape) | `TestAResizeDuringANudge...` red: "the restore leg overwrote a size it did not read" |
| insert a decl between `Resize`'s doc and its func | the new `stolenFrom` guard fires |

Gaps: C-1 is unpinned in both consoles (no test asserts what a switch does when the replay is empty — the only empty-replay assertion is `hostty`'s, which asserts the behaviour C-1 says is wrong at every site). BR-6's composed-feed and BR-8's intent remain byte-inert and correctly recorded as such rather than pinned by a test that would assert nothing. `-race` across `couchtty`/`ptychild`/`termcmd` reports no races. Full `go test ./...`: every failure traces to `operation not permitted` (the documented pty-sandbox class).

### 6. Architecture

- **ARCH-DRY — pass.** `Child.RequestRepaint` is one home (BR-5), `hostty.RepaintFor` is the one mode read (BR-2/BR-12), `childOf` is spelled once. No duplicated block found in the diff.
- **ARCH-PURE — pass.** `Repaint` is pure and tested without IO; the child read is isolated in `RepaintFor`; `hostty` importing `ptychild` (not the reverse) keeps `#146`'s split intact.
- **ARCH-PURPOSE — flag (C-1).** The class sweep is real and impressive — both consumers, all four takeover sites, the request bound *inside* the takeover so a site cannot compose without asking. But the empty-body policy was swept for `RepaintClear` sites and not for `RepaintReplace` ones: the instance, again, not the class.
- **ARCH-MOCK — flag (I-3).** The live half exists and is now honest (`make test-zellij-repaint` reading the production constant). The in-process half diverges from the real `Child` on the new geometry state, and there is no scheduled cadence for the live check — it is a manual target, consistent with this repo's other probes but worth naming.
- **ARCH-CONSTRAINTS — flag (I-1).** The envelope is stated and measured for the requesting caller and silent for the contending one.
- **ARCH-SECURE — pass.** `PAIR_PROBE_SETTLE` is parsed at the boundary and fails visibly rather than defaulting; the probe scrubs `ZELLIJ*`; no credential or untrusted-persistence surface in the diff.
- **ARCH-ORDER — flag (I-2).** The nudge's `(state, event)` set is now readable off the code — `nudging` is a single flag under `geom`, and "a request during one in flight is dropped" is enumerated and tested. The takeover *writer*'s interleaving is not, and there is no seam to inject it.

### 7. Plan revision recommendations

The plan now matches the code — the 2026-09-10 Revisions entry corrected all three rows BR-9 named, and the Tests row no longer asks for the withdrawn cross-product (BR-1). Two additions if C-1 and I-1 are taken:

- **`## Revisions` — "Mode 2's answer is scoped to a same-child repaint."** The plan's Mode-2 row states "a repaint that has nothing to draw must NOT blank (a stale frame beats a blank one)" as an unqualified rule. Record that the enumeration of takeover sites contains no same-child repaint, so the rule as written has no correct caller, and state the qualified version.
- **`## Revisions` — "the nudge's envelope covers the contending path too."** Mode 1's row says a failed nudge "degrades to today's behaviour"; it does not say a concurrent `Resize` waits a settle (measured 20.8 ms) on `termcmd`'s sole pane writer. Add it here and as a row in `#204`'s table beside `event-loop time per nudge`.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Tests row now names the two replacement tests; no cross-product remains.
  - id: BR-2
    disposition: addressed
    note: |
      Verified by revert: dropping RequestRepaint from takeOverScreen reds 2 couchtty tests.
  - id: BR-3
    disposition: addressed
    note: |
      Probe drives the production sequence and reads ptychild.RepaintSettle; own target, not test-smoke.
  - id: BR-4
    disposition: addressed
    note: |
      Verified by revert: RepaintClear -> RepaintReplace reds TestClosingATabBlanksTheDeadTabsScreen...
  - id: BR-5
    disposition: addressed
    note: |
      Child.RequestRepaint is the single home; both consumers call it, no copy remains.
  - id: BR-6
    disposition: addressed
    note: |
      Both consoles feed the composed bytes; byte-inert today and honestly recorded as such.
  - id: BR-7
    disposition: addressed
    note: |
      Verified by revert: unlocking geom across the settle reds the goroutine-independence test.
  - id: BR-8
    disposition: addressed
    note: |
      Intent parameter landed and showMenu passes RepaintClear; byte-inert exception recorded, not papered over.
  - id: BR-9
    disposition: addressed
    note: |
      204's invariant row and envelope table landed; the four reproductions carry answer tests; differential closes transitively.
  - id: BR-10
    disposition: addressed
    note: |
      Doc restored AND a mechanical guard added; I confirmed the guard fires on a synthetic recurrence.
  - id: BR-11
    disposition: addressed
    note: |
      No chr(39) artifact anywhere in cmd/.
  - id: BR-12
    disposition: addressed
    note: |
      Child.AltScreenObserved is gone; RepaintModes is the only door. New instances raised separately.
  - id: BR-13
    disposition: addressed
    note: |
      6.6 KB back-to-back / 19.3 KB with a gap now sit in 204's table beside the counted invariant.
findings:
  - id: new
    severity: Critical
    family: absent-data-is-not-intent
    title: |
      An empty replay leaves the OUTGOING surface on screen, and no takeover site wants that
    detail: |
      3rd in this family, so the deliverable is the rule, not the row. The rule:
      what an empty body means is a property of whose frame is already on the
      screen, not of the call site's name. "A stale frame beats a blank one"
      (repaint.go:26-29) holds only for a same-child repaint, and the enumeration
      of takeover sites contains none - switchTo, switchRelative, showMenu,
      newTab and removeTab all replace a DIFFERENT surface. Reachable on the
      primary flow: starting a thread from the panel attaches (console.go:366
      seeds replayCutoff = ReplaySafeEnd) and dispatches switch on the same
      operationQueue goroutine before Run drains a batch, so ReplayThrough
      returns nothing, Repaint emits zero bytes, and the PANEL'S OWN MENU BODY
      stays on the terminal under the new thread's label until zellij's first
      frame. Before this diff it was HomeAndClear plus nothing - blank, which is
      honest. This is BR-4's shape at the sites BR-4 did not name. Fix: make the
      enum name the question the site must answer, and record that the
      keep-stale branch currently has zero correct callers.
  - id: new
    severity: Important
    family: operating-envelope-unstated
    title: |
      A mandatory Resize waits a full settle on the nudge's lock, while 204 states the cost as zero
    detail: |
      2nd in this family, so state the rule. nudge() holds geom across
      time.Sleep(RepaintSettle) (child.go:277-290) and Resize takes the same
      lock; measured in a scratch copy, a concurrent Resize blocked 20.8 ms.
      termcmd's resizeAll (run.go:1600) runs on the WRITER goroutine via
      resizeThroughWriter, so a SIGWINCH landing inside a switch stalls all pane
      output for a settle. 204's table says "event-loop time per nudge | 0",
      which is true only for the caller that requests the nudge. The rule: an
      envelope must be stated for every path the mechanism can block, not only
      the path that invokes it - the contention sibling of the rule 204 already
      writes for cost. Enumerate the geom contenders (Console.applyLayout,
      terminalMux.resizeAll, teardown) and give each a row, or let Resize
      preempt an in-flight nudge via a generation the restore leg checks.
  - id: new
    severity: Important
    family: nudge-ordering-and-extent
    title: |
      takeOverScreen still claims Run-goroutine-only, and this round's own test drives it from the operationQueue
    detail: |
      2nd in this family, so state the rule: a goroutine-ownership claim in a
      comment is not a mechanism - which is exactly what C2 concluded for the
      nudge and did not apply to the writer. console.go:996 asserts "still
      Run-goroutine-only, like every other writer", but switchTo is reached from
      ExecuteConsoleOperation (console.go:1862) and
      TestASwitchThroughTheOperationQueueNudgesLikeAnyOther exercises that path.
      takeOverScreen writes c.host directly; hostty.Host is a bare io.Writer and
      couch has five unsynchronized write sites (console.go:915,983,1009,1058;
      console_menu.go:193,199,202). termcmd already solved this next door -
      paneWriter is deliberately not an io.Writer so a door that skips the
      reasoning does not compile (run.go:993). Either give couch the same typed
      single-writer door or delete the false sentence and file the divergence;
      leaving it is what lets C2's rule read as satisfied. Same lens, smaller:
      go c.nudge() has no tie to c.done, so the goroutine outlives Close by a
      settle with no cancellation path.
  - id: new
    severity: Important
    family: fake-diverges-from-real
    title: |
      NewFakeChild starts with no geometry while Start records opts.Size, and geometry is what the nudge reads
    detail: |
      Start sets size: opts.Size (child.go:129-135) so a real child always
      nudges; NewFakeChild (fake.go:37-46) sets nothing, so a fake silently
      declines via size.Rows < 2 until a test resizes it. run_test.go:1341
      rationalises the workaround as "what production does", but production
      cannot reach the zero-geometry state the fake starts in - a consumer that
      forgot to size a child would be invisible in tests and would nudge in
      production. TestFakeAndRealChildAgreeAfterTheChildHasEnded drives
      Write/Resize/Signal and does not cover Size or RequestRepaint. ARCH-MOCK:
      give NewFakeChild a starting size mirroring Start, and add both to the
      conformance stimulus set.
  - id: new
    severity: Minor
    family: dead-exported-surface
    title: |
      Three newly-exported identifiers have no consumer outside their own package's tests
    detail: |
      2nd in this family, so fix the rule rather than the three. hostty.EnterAltScreen
      (control.go:68) is referenced only by repaint_test.go's forbidden list IN
      THE SAME PACKAGE, so it never needed exporting; Child.Size() (child.go:346)
      has zero production callers and two test ones; hostty.Repaint is reachable
      only through RepaintFor and its own package's tests. Measured prevalence:
      three instances this round plus BR-12 last round. The rule - exported
      surface needs a consumer outside its own package's tests - is AST-checkable,
      and this repo already writes that kind of guard (doccomment_test.go).
  - id: new
    severity: Minor
    family: comment-cites-code-that-moved
    title: |
      The mutation recipe in console_test.go names a signature and a call site that do not exist at HEAD
    detail: |
      console_test.go:1056 says the test reds on "dropping
      p.child.RequestRepaint(c.ChildSize()) from switchTo". RequestRepaint takes
      no argument since C2 and the call moved into takeOverScreen. A reader
      following the recipe finds nothing to delete and may conclude the test is
      unpinned - the precise failure mode this issue's ledger exists to prevent.
  - id: new
    severity: Minor
    family: test-sleeps-instead-of-synchronizing
    title: |
      The goroutine-independence race test sleeps to let the nudge take the lock, so it can fail spuriously
    detail: |
      replay_insufficiency_test.go:196 sleeps RepaintSettle/4 after
      RequestRepaint, assuming the spawned nudge has taken geom. Under load or
      -race it may not have; the racing Resize then lands first and the
      assertion got[1].Rows != before.Rows-1 fails on a correct tree. Wait on
      waitForResizes(t, child, 2) instead - the same lesson the test's own
      comment teaches about the other party.
```

---

## Re-review — 2026-09-09T23:25:17-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 209 — thread switch leaves the screen partially redrawn |
| repo | pair |
| issue file | workshop/issues/000209-thread-switch-leaves-the-screen-partially-redrawn.md |
| boundary | whole-issue close |
| milestone | — |
| window | 11e12276c7602d583d78fd1348a2bdd95fb4fa1f..b947b4d747958982c0e121d623345e16b3d42fdf |
| command | sdlc close --issue 209 |
| reviewer | claude |
| timestamp | 2026-09-09T23:25:17-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The mechanism this issue set out to build is now correct, measured against the real binary, and defended by tests I verified by mutation: the takeover always blanks (C-1's withdrawal is real, not just documented), `RequestRepaint` takes no size and releases `geom` across the settle (I checked — holding it turns `TestAResizeDuringANudgeWinsWhicheverGoroutineItComesFrom` red), the request rides *inside* `takeOverScreen`/`applyTakeover` so no site can compose a screen without asking, and the probe reads `ptychild.RepaintSettle` instead of restating it. What blocks a clean SHIP is that the C-1 withdrawal was swept through the *code* and through the three Plan rows the last review named, but not through the enumerable rest of the places the withdrawn rule is written: seven prose sites (including `atlas/architecture.md`, which now teaches the deleted rule as current, and two mutation recipes that name symbols a reader cannot find) and one unrevised `## Done when` bullet the shipped code deliberately does not meet. Separately, and measured rather than inferred: BR-17's fix (`FakeChildSize`) reverts with the whole suite green — the second consecutive round in which an `addressed` disposition ships without a red-on-revert test. Full suite on the four touched packages is green modulo the documented pty-sandbox class (identical failure set at base); `-count=3 -race` on the nudge tests is clean; `go vet` clean.

**1. Strengths**

- `cmd/internal/hostty/repaint.go:26-49` — the C-1 withdrawal is written with its *hidden premise* named and the five-site enumeration that refutes it, and it is pinned (`TestATakeoverAlwaysBlanksEvenWithNothingToDraw` covers nil / empty / non-empty). This is the one place in the diff where a reader learns why the obvious design is wrong.
- `cmd/internal/ptychild/child.go:272-330` — deleting the size parameter instead of documenting harder is the right shape of fix, and the generation counter makes "optional work never gates or beats the mandatory kind" a mechanism rather than a rule. Verified by revert in a scratch worktree: holding `geom` across the settle produces `resizes = [{24 80} {23 80} {24 80} {40 120}]` and reds the test.
- `cmd/probes/zellijrepaint/main.go:118` reads the production constant, and the probe lives under `cmd/probes/` with its own `make test-zellij-repaint` target — confirmed `test-smoke` globs only `probes/*/`, so the condemned no-settle sequence can no longer run unattended.
- `console.go:1056` / `run.go:1043` — binding the request to the takeover rather than to the call sites is the class fix I2 claimed, and it holds: `showMenu` and `clearTab` pass `nil`, `RequestRepaint`'s nil-receiver guard makes that total, and both are pinned (`TestOpeningThePanelAsksNoChildToRepaint`, `TestClosingATabAsksTheSurvivingChildToRepaint`).
- `cmd/internal/termcmd/doccomment_test.go:121` — `stolenFrom` converts BR-10's recurrence into a mechanical guard, deliberately narrowed to "opens with another declaration's name in this file" so it stays a bug detector rather than a style rule.

**2. Critical findings** — none.

**3. Important findings**

**(a) The C-1 withdrawal did not reach the prose that states the withdrawn rule** — 2nd in family `comment-cites-code-that-moved`, so the deliverable is the rule, not the seven edits. The rule: **deleting an exported identifier obliges a sweep of every prose reference to it, not only the code that stopped compiling** — and unlike most sweeps this one is fully mechanical (`grep -rn RepaintIntent\|RepaintReplace\|RepaintClear\|hostty.Repaint` over non-`history` files returns the whole set). The repo already writes AST guards for exactly this class (`doccomment_test.go`), so a check that a `//`-comment token matching a `Repaint[A-Z]\w*` shape resolves in the module is the same kind of guard one level over. Measured prevalence at HEAD, 7 sites, all created by the round that deleted the enum:

| site | what it says that is no longer true |
|---|---|
| `atlas/architecture.md:531` | `RepaintFor(child, replay, intent)` — the signature has no `intent` |
| `atlas/architecture.md:537-540` | "Intent is CARRIED, not inferred from an empty slice… a stale frame beats a blank one. `redrawTab` and `clearTab` are separate doors for that reason" — the atlas is the durable map and it teaches the rule C-1 deleted |
| `cmd/internal/termcmd/run.go:1693-1696` | `redrawTab` doc: composition "emitting nothing rather than blanking… the two differ in a case an empty slice cannot express" — contradicted by `clearTab`'s own doc 20 lines below ("not a second behaviour") |
| `cmd/internal/termcmd/run_test.go:1431-1440` | mutation recipe "flipping removeTab's intent back to `RepaintReplace`" — no such symbol; rationale restates "a stale frame beats a blank one… the right call for a switch" |
| `cmd/internal/couchtty/console_test.go:1149-1153` | `RepaintReplace`/`RepaintClear`/`TestRepaintCarriesIntentRatherThanInferringItFromAnEmptyReplay` — all three deleted |
| `cmd/internal/ptychild/replay_insufficiency_test.go:131,137` | "Mode 2's answer lives in `hostty.Repaint` (an empty replay emits nothing rather than blanking)", repeated in the failure message |
| `cmd/internal/ptychild/child.go:484` | `hostty.Repaint` — unexported this round |

Two of these are *mutation recipes*. A reviewer following `run_test.go:1431` to verify BR-4's pin finds nothing to flip and may conclude the test is unpinned — the precise failure mode this issue's ledger exists to prevent, and the one BR-19 named one file over.

**(b) `## Done when` bullet 2 states a criterion the shipped code deliberately does not meet, with no `## Revisions` entry** — 2nd in family `plan-row-ticked-not-delivered`. The rule: **a reversed commitment must be revised everywhere it was written — Plan rows, `## Done when`, and the atlas — not only in the rows a review happened to name.** "The `cutoff < ringStart` path cannot present a cleared screen with nothing drawn" was met by the middle version and is now explicitly declined: `repaint` always blanks, so an empty replay *is* a cleared screen with nothing drawn until the child's frame lands (and permanently for a bare-shell `pair term` tab, per the 2026-09-09 revision's own "best-effort" carve-out). The `## Log` third-pass entry argues this position well — "mode 2's answer was never the empty-replay carve-out" — but the third-pass `## Revisions` Delta lists only three *Plan* rows and never touches Done-when. This is the criterion the merge-time `specs` judge reads.

**(c) BR-17 re-raised as `not-addressed`** — see dispositions. Measured: deleting `size: FakeChildSize` from `NewFakeChild` (`fake.go:47`) leaves all four packages with a failure set byte-identical to the baseline pty-sandbox class. No test observes that a fresh fake has geometry, and the conformance stimulus set (`child_test.go:255-259`) still drives only Write/Resize/Signal — `Size` and `RequestRepaint`, the two surfaces this issue added and the ones the fix exists to serve, were not added as BR-17 asked.

**4. Minor findings**

- BR-18 re-raised as `not-addressed`: two instances unexported individually, no guard written, and the class grew — `hostty.ChildModes` (`repaint.go:21`, consumers only inside hostty), `ptychild.Screen.AltScreenObserved` (`screen.go:127`, consumers only inside ptychild — while `child.go:490` claims "there is deliberately no exported AltScreenObserved to reach for"), `ptychild.FakeChildSize` (`fake.go:59`, an exported *var* with zero consumers anywhere but its own definition), `Child.Size` (`child.go:405`, test-only).
- `Child.Size()` reports the *shrunk* value for the 20 ms of a settle. Harmless today (no production callers) but it makes the accessor lie during exactly the window a debugger would ask.
- `replayChild` (`replay_insufficiency_test.go:22`) builds a `&Child{}` directly and so bypasses `FakeChildSize` — the zero-geometry shape BR-17 says production cannot reach still exists in this package's fixtures.

**5. Test coverage notes**

`hostty/repaint_test.go` is the strongest file in the diff: four tests, no IO, one stated rule each, and the golden that makes the transitive differential (`TestSwitchWritesExactlyTheComposedRepaint` / `TestTakeoverWritesExactlyTheComposedRepaint`) mean something. The nudge suite is genuinely ordering-aware — the race test waits on the other party rather than sleeping toward it, and it asserts *exactly three* resizes, which is what caught the reverted lock-release for me. Two gaps: the `<-c.done` cancellation path has no test (observable against the fake only as timing, so cheap but low-value), and the fake/real conformance gap in 3(c). The `_ = modes` parameter in `repaint` is unpinnable by construction while the buffer assertion is withdrawn — the code says so honestly, which is the right disposition, but it does mean `RepaintFor`'s read reaches nothing today.

**6. Architectural notes**

- **ARCH-DRY** pass — one composer, one `RequestRepaint`, one `HomeAndClear` emitter, `childOf` spelled once. BR-5's copy-paste is fully retired.
- **ARCH-PURE** pass — `repaint` returns bytes and writes nothing; `RepaintFor` is a thin locked read; all four hostty tests run without IO.
- **ARCH-PURPOSE** flag — findings 3(a)/3(b): the withdrawal was swept for the *instances* the last review named and not for the enumerable class, which is the same axis as this round's own I1 lesson.
- **ARCH-MOCK** flag (minor) — the live conformance check is real and now reads the production constant, which is the strong half. The in-process half is where the gap is: 3(c).
- **ARCH-CONSTRAINTS** pass, with one unmeasured note for future work — `#204`'s table bounds nudges *per switch* (1) and in-flight nudges *per child* (1), but not per burst across children: holding alt+→ through N tabs nudges N children, each worth ≥6.6 KB of re-render on a single-pane measurement. Non-active children's bytes are gated at the console, so the cost is child CPU plus ~5% ring churn per nudge. Unmeasured, so a note rather than a finding — but it is the row `#204` will want when the fleet is 10+ panes.
- **ARCH-SECURE** largely N/A — the only new external input is `PAIR_PROBE_SETTLE`, parsed with an explicit error path in a probe. No credentials, no persisted artifacts, no new trust boundary.
- **ARCH-ORDER** pass — `(nudging, geomGen, size)` under one mutex with every exit path resetting `nudging`; cancellation tied to `c.done`; extent bounded by one settle; `Close` takes `geom` so an in-flight ioctl completes first. The interleaving is injectable and the race test drives the adversarial goroutine rather than sampling one ordering.

**7. Plan revision recommendations**

- `## Revisions`, third pass — extend the Delta with a fourth item: *Done-when clause 2* ("the `cutoff < ringStart` path cannot present a cleared screen with nothing drawn") is **withdrawn** for the same reason as the mode-2 row; the answer is the repaint request, and a blank frame for one settle is the intended behaviour, best-effort for a child with no screen model per the 2026-09-09 revision.
- `## Revisions`, new entry — record that the withdrawal's sweep was incomplete, with the seven-site table from 3(a), and name the rule so the `?1047` follow-up does not re-introduce the deleted rationale from the atlas.
- `## Plan`, ARCH-MOCK row — it claims the fake "models the behaviour we depend on"; add that `Size` and `RequestRepaint` join the conformance stimulus set, or say plainly that they do not and why.

```findings
dispose:
  - id: BR-14
    disposition: addressed
    note: |
      Branch and RepaintIntent both deleted; repaint always blanks, pinned by TestATakeoverAlwaysBlanksEvenWithNothingToDraw over nil/empty/non-empty.
  - id: BR-15
    disposition: addressed
    note: |
      Verified by revert — holding geom across the settle reds TestAResizeDuringANudgeWinsWhicheverGoroutineItComesFrom with a 4th resize.
  - id: BR-16
    disposition: addressed
    note: |
      False sentence deleted, divergence filed as pair#224, settle cancellable on c.done, Close takes geom.
  - id: BR-17
    disposition: not-addressed
    note: |
      Measured — deleting `size: FakeChildSize` leaves all four packages at the baseline failure set; Size/RequestRepaint still absent from the conformance stimuli.
  - id: BR-18
    disposition: not-addressed
    note: |
      Instances unexported individually, no AST guard written, and the class grew — ChildModes, Screen.AltScreenObserved, FakeChildSize.
  - id: BR-19
    disposition: addressed
    note: |
      The named recipe now cites child.RequestRepaint() in takeOverScreen, which exists; the class recurred elsewhere — see the new finding.
  - id: BR-20
    disposition: addressed
    note: |
      waitForResizes(t, child, 2) replaces the sleep, exactly as recommended.
findings:
  - id: new
    severity: Important
    family: comment-cites-code-that-moved
    title: |
      The C-1 withdrawal reached the code and the named Plan rows, but not the seven other places the withdrawn rule is written
    detail: |
      2nd in this family, so the deliverable is the rule: deleting an exported
      identifier obliges a sweep of every PROSE reference to it, not only the
      code that stopped compiling. Measured prevalence 7 at HEAD, all created by
      the commit that deleted RepaintIntent — atlas/architecture.md:531 (wrong
      RepaintFor signature) and :537-540 (teaches "a stale frame beats a blank
      one" as current, in the always-current map); termcmd/run.go:1693-1696
      (contradicts clearTab's own doc 20 lines below); termcmd/run_test.go:1431
      and couchtty/console_test.go:1149 (MUTATION RECIPES naming RepaintReplace /
      RepaintClear / TestRepaintCarriesIntent..., so a reviewer verifying a pin
      finds nothing to flip — BR-19's exact failure mode one file over);
      ptychild/replay_insufficiency_test.go:131,137; ptychild/child.go:484. The
      set is fully mechanical (grep for the four deleted names over non-history
      files), and doccomment_test.go is the precedent for turning it into a guard.
  - id: new
    severity: Important
    family: plan-row-ticked-not-delivered
    title: |
      Done-when clause 2 states a criterion the shipped code deliberately does not meet, with no Revisions entry
    detail: |
      2nd in this family, so the rule: a reversed commitment must be revised
      everywhere it was written — Plan rows AND Done-when AND the atlas — not
      only in the rows a review named. "The cutoff < ringStart path cannot
      present a cleared screen with nothing drawn" was met by the middle version
      and is now explicitly declined: repaint always blanks, so an empty replay
      IS a cleared screen with nothing drawn until the child's frame lands, and
      permanently for a bare-shell pair term tab. The third-pass Revisions Delta
      lists three Plan rows and never touches Done-when, which is the section the
      merge-time specs judge reads.
```
