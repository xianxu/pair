---
id: '000111'
status: done
started: 2026-07-07T23:38:22-07:00
created: 2026-07-07
updated: 2026-07-08
estimate_hours: 0.40
actual_hours: 0.21
---

# conditional replacement of brain/pair word

## Problem

The cmux workspace title convention currently replaces `brain` with `🧠` and
`pair` with `♋` unconditionally. That is useful for compound Pair session names
such as `pair-brain`, but it is wrong when the whole title is just `brain` or
`pair`: a plain shell cwd/workspace title should stay readable as the literal
repo name.

There are two active implementations of the convention:

- `cmd/internal/launcher.EmojiTitle`, used by launcher cmux renames.
- `cmd/internal/titlepoller.cmuxWorkspaceTitle`, used by the title poller heat
  ramp.
