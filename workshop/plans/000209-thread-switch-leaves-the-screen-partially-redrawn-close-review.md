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
