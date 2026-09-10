---
id: 000223
status: open
deps: []
github_issue:
created: 2026-09-09
updated: 2026-09-09
estimate_hours:
---

# scroll-triggered strip repaint lands the shell's cursor in the tab strip

## Problem

Operator report, 2026-09-09, reproduced by them twice: **running a command whose
output scrolls the screen sometimes leaves the shell's cursor inside the tab
strip.** Typed characters then land on the strip row instead of at the prompt.
Screenshot shows `l` (an `ls` alias) with the caret on
`terminal 1 [terminal 2]`.

**This is the hazard `#199` already characterised, not a new one**, and
`atlas/architecture.md:636-648` records the mechanism: the DECSC save slot is
SHARED, one per terminal; zsh draws its right-hand prompt with terminfo
`sc`/`rc`, which are exactly `DECSC`/`DECRC`; so a strip paint that lands
between the child's save and its restore leaves the slot holding the CONSOLE's
position, and the child's restore goes there.

There is no second slot to move to. `probes/cursorsaveslots`, re-run 2026-09-09
against the live terminal, is unambiguous and worse than "they alias":

```
SCOSC restore landed at row 0 (found=false) — remembered row 5
DECSC restore landed at row 11 (found=true) — remembered row 10
RESULT: NO USABLE SECOND SLOT
```

So the mitigation `#199` chose is the only one available: **do not write while a
save is held outside the alt screen**, gated by `ptychild.Screen.SafeToPaint`
(`!MidSequence() && !(cursorSaved && !altScreen)`).

### What is actually unexplained

That gate EXISTS and is consulted on both consoles — `termcmd.unsafeToPaint`
(`run.go:1032`) and `couchtty` at `console.go:1042`, `:1142`, `:1172`. The
symptom says it is not holding for the scroll case. Which is the question this
issue is for, and it is NOT yet answered:

1. Does the scroll-driven strip repaint go through the gated path at all, or is
   there a write that bypasses `unsafeToPaint`?
2. Does the scanner SEE zsh's `DECSC` before the paint decides — i.e. is the
   save observed, or does the paint race the bytes that would have closed the
   gate?
3. Does a scroll itself invalidate the belief? The reserved row interacts with
   `DECSTBM`, and a scroll region change is exactly the kind of state the
   scanner tracks separately.

**Do not fix before answering which.** The three have different fixes, and this
repo has been repeatedly right that the mechanism must be measured rather than
inferred — `#209` spent five plan rounds on exactly that lesson.

### Not caused by `#209`

Recorded because I mis-attributed it in the moment and the operator deserves the
correction in the durable record too. `#209`'s withdrawn `?1049` buffer
assertion DID move the cursor and was rightly withdrawn (`213d64cf`), but the
symptom persisted after the withdrawal, the trigger is **scroll rather than
switch**, and `#209` changed nothing on the scroll path: `termcmd`'s takeover
still feeds `replay` to the scanner as before, and for a non-empty replay
`hostty.Repaint` now emits bytes identical to the previous clear-then-write.

## Spec

## Done when

- The mechanism is identified — one of the three questions above answered with
  evidence, not a guess.
- A command whose output scrolls the screen leaves the cursor at the prompt,
  reproducibly, with the strip still painted.
- Whatever the answer, a regression test at the seam it names.

## Plan

- [x] **Question 1 is ANSWERED: no ungated write to the strip row exists in
      `termcmd`.** See `## Log` 2026-09-10 for the enumeration and the corrected
      premise — the test this row named was retired on purpose and replaced by a
      type that makes an ungated door a compile error.
- [ ] So the answer is question 2 or 3. Instrument the live path: log
      `SafeToPaint`'s inputs (`cursorSaved`, `altScreen`, `MidSequence`) at each
      strip paint, and reproduce with a scrolling command. Question 2 (the paint
      races the bytes that would have closed the gate) and question 3 (a scroll
      invalidates the belief) predict different values at the same instant,
      which is what makes one log line decide between them.
      One specific candidate for question 3, cheap to check first: a scroll
      emits NO escape sequence — it is newlines and text — so nothing in
      `Screen.classify` sets `rowDirty` for it (the arms are RIS, `?1049`/
      `?1047`/`?47`, mouse modes, DECSTBM and ED). If the strip is nonetheless
      being repainted on a scroll, the trigger is not the row-dirty debt and the
      enumeration above needs to name what it IS.
- [ ] Fix what the evidence indicts; regression test at that seam.

## Log

### 2026-09-09

### 2026-09-10 — question 1 answered, and the premise it rested on corrected

**The test this issue's Plan named does not exist, and that is deliberate rather
than rot.** `TestEveryConsoleWriteIsGatedOrExplicitlyExempt` was added by
`#199` M2 (`fe89c8a5`) and RETIRED by that milestone's own close (`244a72a5`)
after round 6 measured the test narrower than its claim: `fmt.Fprintf(m.stdout,
…)` left it green, a `gate-exempt:` comment separated by a blank line exempted
an unrelated write, and it read `run.go` only while M3 was adding `strip.go` to
the same package. The replacement is stronger than the test — `paneWriter` is
NOT an `io.Writer`, so a new door does not compile. `run.go:999` still cites the
removed test by name; that citation is stale and is the reason this issue went
looking for it.

**The enumeration, done against the type instead.** Every byte that reaches the
pane goes through `paneWriter.raw`, and there are seven call sites:

| `run.go` | reason | gate |
|---|---|---|
| 944 | child output | exempt — it is what the gate MODELS, not a user of it |
| 1008 | takeover | exempt — the scan is reset immediately before |
| 1069 | diagnostic | `writeDiag`, gate consulted |
| 1095 | paint | `writeOwn`, gate consulted |
| 1130 | owed diagnostic | `flushOwed`, gate consulted at entry |
| 1138 | owed paint | `flushOwed`, gate consulted at entry |
| 1752 | teardown `ResetRegion` | exempt — the loop may already be gone |

**Every strip paint is on a gated row.** `paintStrip` → `paintOwn` → `writeOwn`,
and `paintStripInline` → `writeOwn` directly; there is no third road. So the
symptom is NOT an ungated write, and question 1 is closed by inspection without
spending a live reproduction on it.

That leaves questions 2 and 3, which is where the instrumentation should go.

**Relation to `#209`, checked rather than assumed.** They are the same SEAM and
the same failure FAMILY, and not cause and effect:

- `#209` did briefly create an instance of exactly this class — its `?1049l`
  buffer assertion performed a cursor restore from the shared save slot and
  landed typed characters mid-screen. That is withdrawn (`213d64cf`) and pinned
  by `hostty`'s `TestRepaintEmitsNoCursorMovingBufferAssertion`, so it cannot
  come back without a probe.
- `#209` touches the TAKEOVER path (`applyTakeover`, `takeOverScreen`) and the
  SWITCH path (the repaint nudge). A scroll reaches neither: it is child output
  through `handleChunk`'s default branch, which `#209` did not change.
- The family they share is the one `#196` opened and the atlas records: a belief
  about the child's terminal state, held by a component that may not have
  witnessed the whole stream, used as a SAFETY input. `#196` was mouse mode,
  `#209` the screen buffer, and this would be the cursor-save slot — the third.
  Whatever this issue finds belongs in that account, not in a fourth private
  one.

### 2026-09-10 — the mechanism is zellij's autowrap, and none of the three questions was it

**Measured, and it is not the DECSC slot.** The operator's live report gave the
clue the three questions above missed: `ls -la` output dies on exactly the one
line that WRAPS (`Makefile.workflow → ../ariadne/Makefile.workflow`), and every
later line — and the prompt — is overprinted onto the strip row. A second report
narrowed it further: short single-line output at the bottom margin scrolls
correctly, every time.

`probes/zellijwrapmargin` asks zellij for its own cursor position (DSR) after
each step, with the region set exactly as `pair term` sets it:

```
pane rows=22 cols=78 region=1..21
at-bottom-margin      row=21 col=1
after-short-line      row=21 col=1    newline scrolls the region: correct
after-wrapping-line   row=22 col=11   autowrap lands ON the reserved row
after-its-newline     row=22 col=1    and stays: LF below the region cannot scroll
VERDICT: WRAP ESCAPES THE REGION
```

So on zellij 0.44.3, **autowrap at the bottom margin of a DECSTBM region moves
the cursor below the region instead of scrolling it**, while a newline at the
same spot is handled correctly — zellij disagrees with itself, which is the
signature of a bug rather than a spec reading. Once the cursor is on row N,
every later line overprints it; when zsh's next prompt dirties the row, `pair
term` repaints the strip, and its DECSC/DECRC bracket faithfully restores the
cursor to where the shell left it — on the strip row. The gate, the save slot
and the scanner are all behaving correctly. The "sometimes" is whether a line
long enough to wrap arrives after the screen has filled.

The same class of bug shipped in Windows Terminal (microsoft/terminal#19016). No
zellij report was found.

**Consequences for this issue's framing.** Its Problem section attributes the
symptom to `#199`'s save-slot hazard; that is wrong and is left above as the
record of what was believed. `#209` neither caused nor could fix it. And `couch`
is unaffected: it reserves its row on the HOST terminal, not in zellij's
emulator.

**The fix is a design decision, not a patch**, because nothing inside `pair
term`'s current design can see the wrap happen: it forwards the child's bytes
without modelling the cursor. The candidates are recorded in `## Spec` once the
operator chooses.

**A/B against the pre-`#209` binary: identical.** The operator reported the
symptom as a `#209` regression, so that was tested rather than argued. Two
builds — `11e12276` (the branch point; its only change is `#209`'s issue file,
and it already has `#199`'s strip) and the current head — each run as `pair
term` inside a throwaway zellij session whose tab child fills the region, prints
one line that wraps, then four marker lines. Both end with the same final row:

```
WWWWWWWWWW…WWWWWWWWEND_MARKER1]     ← markers overprinted the strip;
                                      "1]" is what is left of "[terminal 1]"
```

So the escape predates `#209`. It is as old as the reserved row itself —
`#199` M3, 2026-09-08 — and a `pair term` started before that has no scroll
region and cannot hit it; several such panes are still running. It showed up
during `#209` because `#209`'s first version DID move the cursor (the `?1049`
assertion, withdrawn in `213d64cf`), which put cursor symptoms under scrutiny at
the same moment. The harness was a throwaway; `probes/zellijwrapmargin` is the
durable measurement.

**The top edge, measured rather than argued** (`PAIR_PROBE_EDGE=top:<mode>`,
region 2..N). zellij's wrap ignores the region at BOTH edges; only the failure
differs:

| top edge | result |
|---|---|
| ordinary scrolling | strip survives |
| one wrap at the last row | strip scrolled away — the wrap scrolls the whole screen |
| `ESC[H`, no origin mode | draws on the strip (why `reserve.go` refused `EdgeTop`) |
| origin mode, trusted to DECRC | draws on the strip — zellij's DECRC does not restore DECOM |
| origin mode re-asserted after each paint | lands on row 2; strip survives |

So a top strip would not be immune. But its failure is MILDER in kind: the
cursor stays where the shell put it, no output is overprinted and nothing typed
lands in the strip — the strip is pushed off-screen and returns at the next
strip repaint, at the cost of the oldest visible line. The bottom edge loses the
output and the cursor. The price is origin mode: every paint must re-assert
`?6h` itself, and a child that sets or clears DECOM breaks the arrangement.
