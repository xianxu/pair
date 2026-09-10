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
