---
id: 000127
status: working
deps: []
github_issue:
created: 2026-07-28
updated: 2026-07-28
estimate_hours:
started: 2026-07-28T16:20:34-07:00
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

## Spec

- **M1 — mouse release is a complete event.** Recognize both SGR terminators
  (`M` press, `m` release) in the prefix parser and the partial-sequence test,
  carry the press/release distinction on the event, and forward releases to the
  child. A release is never a wheel tick (the wheel reports press-only), so it
  must not fall into the scroll branch.
- **M2 — a tab switch must not re-ask the terminal, and a reply must never be
  typed into a shell.** Two layers:
  - Root cause: stop replaying capability queries. Filter query sequences out of
    `tab.buffer` as it is appended, so a `redrawTab` replay cannot re-issue them.
  - Defense in depth: recognize known reply forms in `pumpStdin` and absorb them
    instead of forwarding them to the active tab as if typed.
- Both are pure stream-decoding decisions and unit-testable against the existing
  `fakeMux` / `splitReader` harness — no PTY needed (`ARCH-PURE`).

## Done when

- A click-drag-release in the right terminal leaves the pane's keyboard live,
  and nvim ends the drag instead of staying in visual mode.
- Switching tabs emits no capability queries and leaves no escape text on the
  shell prompt of either tab.
- Regression tests cover: a lone release, a keystroke after a release, a
  press+release+payload in one read, and a tab switch over a buffer containing
  queries.
- `go test ./cmd/internal/termcmd/` green; live check in a real session.

## Plan

- [x] M1 — recognize the `m` release terminator; forward releases to the child
- [ ] M2 — stop replaying queries on redraw; absorb replies in the pump

## Log

### 2026-07-28

- Filed from a live report while doing v1.24 release prep. Both defects found by
  reading `pumpStdin`; A was confirmed with a failing test before the fix
  (`keystroke after mouse release is not swallowed` → mux ops came out as one
  merged `write:\x1b[<0;8;2ma` at EOF, proving the bytes were held, not sent).
- B's root cause was confirmed by decoding the reported shell line: the bytes are
  DA1 / DECRPM(2026) / DECRPM(2031) / Kitty-flags *replies*, i.e. answers to
  queries, not keystrokes — which points at a replay, not at user input.
- Note `rename_input_test.go` already exercises both `M` and `m` forms, so the
  release terminator was known elsewhere in this file and simply missed in the
  mouse path.
