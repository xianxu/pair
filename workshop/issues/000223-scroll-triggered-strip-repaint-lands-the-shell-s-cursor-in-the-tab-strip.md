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

- [ ] Answer question 1 first, because it is the cheapest and the most likely:
      enumerate every write to the strip row and confirm each is gated. The repo
      already has `TestEveryConsoleWriteIsGatedOrExplicitlyExempt` — check
      whether the scroll path is in its enumeration or exempt from it.
- [ ] If the gate is consulted and still lets it through, instrument the live
      path: log `SafeToPaint`'s inputs at each strip paint and reproduce with a
      scrolling command.
- [ ] Fix what the evidence indicts; regression test at that seam.

## Log

### 2026-09-09
