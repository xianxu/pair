---
id: 000234
status: working
deps: []
github_issue:
created: 2026-09-12
updated: 2026-09-12
estimate_hours: 0.79
started: 2026-09-12T15:41:35-07:00
---

# A bare ESC in the right pane is held until the next keystroke: pair term treats it as an unfinished Alt chord and has no ambiguity timer outside rename

## Problem

Operator report, 2026-09-12: with nvim running inside the right pane
(`pair term`), leaving insert mode takes **two** presses of ESC.

Root cause, from the mux's stdin loop (`cmd/internal/termcmd/run.go`):

- Every legacy Alt chord pair recognizes begins with ESC (`\x1bx`, `\x1bj`,
  …, `workbenchshortcut/shortcut.go:357`). So a read that contains only
  `\x1b` satisfies `IsChordPrefix` (`shortcut.go:493`: shorter than a
  candidate and a prefix of it).
- The main loop then does exactly this (`run.go:540-543`):

  ```go
  if workbenchshortcut.IsChordPrefix(data) || isSGRMousePrefix(data) {
      held = append(held, data...)
      break
  }
  ```

  and waits for the **next read**. The only timer in that `select` is the
  rename one (`case <-timer.C(): applyRename(...)`), armed only inside a
  rename session — where a lone pending ESC gets `timer.Reset(50 *
  time.Millisecond)`. Outside rename, nothing ever releases `held`.

So a single ESC sits in `held` until the operator types something else. The
second ESC arrives, `held+data` is `\x1b\x1b`, which is no chord, and both
bytes go to nvim at once — normal mode, on the second press. That is the
symptom exactly.

**It is worse than a double-press.** The follower decides what the held ESC
becomes. ESC then `j` — the most common thing a vim user types after leaving
insert — is delivered as `\x1bj`, which is `ChordAltJ`: pair's focus-left
chord fires, and nvim sees neither the ESC nor the `j`. Same for ESC then
`k`, `t`, `w`, `r`, `x`, `/`. By construction from the chord table; not
reproduced, and worth reproducing first because it turns "annoying" into
"keystrokes go to the wrong pane".

This is the ambiguity every ESC-prefix decoder has, and pair has already
solved it twice: couch's two input framers share `escapeAmbiguity = 35 *
time.Millisecond` (`couchtty/keys.go:49`, "the one deadline used by both
terminal-input framers to distinguish an ESC key from the first byte of a
split escape sequence"), and termcmd's own rename decoder uses 50 ms. The
main path of `pair term` is the one framer without it. `workshop/lessons.md`
already carries the rule (§"Escape decoders must distinguish prefixes…", and
the read-boundary lesson: "a bare ESC that might be the prefix of a following
CSI: read boundaries carry no semantic meaning, so resolve the ambiguity
explicitly").

Why it does not bite in the draft pane or the agent pane: the draft's nvim
receives keys through zellij with the kitty protocol pushed, where ESC is
`\x1b[27u` and unambiguous; the agent pane goes through pair-wrap, which has
its own decoder. Only `pair term`'s stdin path forwards raw legacy bytes with
a hold and no deadline.

## Spec

Give the main loop the same deadline the other three framers have.

1. **One ambiguity deadline, one owner.** Lift `escapeAmbiguity` (35 ms) out
   of `couchtty` into `workbenchshortcut` — the package that owns the chord
   table is the right owner of "how long a chord prefix may stay open" — and
   use it from couch's two framers, termcmd's rename decoder (replacing the
   local 50 ms), and the main loop. Four sites, one constant (ARCH-DRY).
2. **Main-loop timer.** When `held` becomes non-empty and is a bare ESC (or
   any chord prefix), arm the timer; on expiry, `mux.writeActive(held)` and
   clear it. New input before expiry appends and re-evaluates as today; a
   complete chord or a non-prefix flushes and stops the timer. Mirror the
   rename decoder's shape — it is the same problem three functions up.
3. **Tests generated from the chord table**, per the lessons rule: for every
   chord sequence, the split where only the first byte arrives, then (a) the
   rest arrives before the deadline → chord fires, (b) nothing arrives →
   bytes forwarded as typed after the deadline, (c) a non-chord byte arrives →
   both forwarded, no chord. Plus the specific regressions: `ESC`,`ESC` →
   two ESCs to the child; `ESC` … 35 ms … `j` → ESC then `j` to the child,
   not `ChordAltJ`.

Out of scope: making the right pane's child speak the kitty protocol (would
remove the ambiguity at the source, but it is a much larger change to how
`pair term` hosts its children, and the deadline is what every other framer
in this repo does).

## Done when

- In the right pane, one ESC leaves insert mode in nvim.
- ESC followed by `j`/`k` after the deadline reaches nvim as two keys; the
  Alt chords still fire when typed as chords. (`ESC`,`j` typed inside the
  deadline still decodes as a chord — that residual is #227's.)
- `escapeAmbiguity` has one definition, used by all four framers.
- The generated split tests pass; the two regressions above are explicit.

## Plan

Durable plan: `workshop/plans/000234-right-pane-bare-esc-held-without-timer-plan.md`.

- [ ] Reproduce ESC-then-`j` in the right pane (does focus jump? record it)
- [ ] Move `escapeAmbiguity` to `workbenchshortcut`; re-point couch's framers and termcmd's rename decoder
- [ ] Arm/expire the timer in the main loop; flush `held` on expiry
- [ ] Table-generated split tests + the two explicit regressions
- [ ] Manual: nvim in the right pane — single ESC, ESC+j, Alt+j

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* Design at ×0.2 (the plan pre-resolves the timer ownership, the constant's home, and the test oracle); impl at 40% of the v2 ranges; +15% buffer for a thorough plan doc.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module  design=0.03 impl=0.08
item: smaller-go-module  design=0.06 impl=0.16
item: smaller-go-module  design=0.04 impl=0.20
item: atlas-docs         design=0.02 impl=0.04
item: milestone-review   design=0.00 impl=0.14
design-buffer: 0.15
total: 0.79
```

- lift `EscapeAmbiguity` + re-point three sites (mirror of couch's arm) — 0.03 / 0.08
- main-loop arm + expiry branch + timer-type rename — 0.06 / 0.16
- generated split tests, two regressions, fake hook, deterministic split test — 0.04 / 0.20
- atlas paragraph — 0.02 / 0.04
- close review — 0.00 / 0.14

## Log

### 2026-09-12

- Filed from the brain advisor session on the operator's report. Mechanism
  read from `run.go:540-543` and the `select` at `:461`; the rename path's
  50 ms timer and couch's 35 ms `escapeAmbiguity` are the in-repo precedents.
  The ESC-then-`j` misroute is derived from the chord table, not observed —
  first Plan step is to observe it.
