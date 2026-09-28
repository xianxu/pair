---
id: '000234'
status: done
started: 2026-09-12T15:41:35-07:00
created: 2026-09-12
updated: 2026-09-12
estimate_hours: 0.79
actual_hours: 0.71
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
