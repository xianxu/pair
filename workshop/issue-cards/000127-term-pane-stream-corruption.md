---
id: '000127'
status: done
started: 2026-07-28T16:20:34-07:00
created: 2026-07-28
updated: 2026-07-28
estimate_hours: 1.40
actual_hours: 1.4
---

# right terminal pane corrupts the input stream

## Problem

Two defects reported live in the layout-3 right terminal, both in `pair term`'s
stream handling (`cmd/internal/termcmd/run.go`). They present as one symptom
cluster — "the right pane stops responding and spews escape sequences" — but
have independent root causes.

**A. A mouse release kills the pane's keyboard.** An SGR (1006) mouse event is
`\x1b[<button;col;row` plus a terminator: `M` = press, `m` = RELEASE. Both
`parseSGRMousePressPrefix` and `isSGRMousePrefix` searched only for `'M'`, so a
release matched "sequence not finished yet" and was parked in `pumpStdin`'s
`held` buffer. `held` is prepended to the next read, which re-matched the same
way — so the release *and every keystroke typed after it* accumulated and never
reached the child. Reported as "pressing `a` doesn't do anything".

The child app is simultaneously left holding an unmatched button-press, so nvim
stays in an open mouse drag — visual mode. That is the reported "click to
reposition the cursor becomes a visual selection", and it also explains why the
few bytes that did land looked inert (in visual mode `a` is a pending
text-object, `aw`/`ap`/…).

**B. Tab switching replays capability queries; the replies land on the wrong
tab.** `redrawTab` replays the tab's stored raw output verbatim
(`m.stdout.Write(tab.buffer)`, up to 128 KiB). That buffer still contains the
app's *terminal queries* (DA1, DECRQM, Kitty-keyboard). Replaying re-asks the
host terminal; the host's replies arrive on `pair term`'s stdin and are handed
to `mux.writeActive(...)` — the **now-active** tab's shell — which tries to run
them as a command. Observed live as a shell line reading
`execute: 1e1e/1e1e/1e1e\[?62;4;52c[?2026;2$y[?2031;1$y[?0u[?62;4;52c`
(DA1 reply twice — two replays — plus the synchronized-output,
color-scheme-updates, and Kitty-flags reports).
