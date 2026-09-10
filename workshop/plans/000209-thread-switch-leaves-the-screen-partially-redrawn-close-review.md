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
