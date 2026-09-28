---
id: '000132'
status: done
started: 2026-07-29T22:16:55-07:00
created: 2026-07-29
updated: 2026-07-29
estimate_hours: 3.18
actual_hours: 0.37
---

# Alt+h keybind help is a circular dead end

## Problem

`Alt+h` is documented and advertised as "pop up the full keybind help". It is
not. The chain is:

```
Alt+h  →  zellij bind (config.kdl:163)  →  bin/pair-help  →  `pair -h`
```

and `pair -h` is a 21-line CLI usage block whose **last line reads**:

> In-session keybindings are on Alt+h.

So pressing the help key produces a message telling you to press the help key.

This is a **regression from #99 M5c**, which retired `bin/pair-shell` and
replaced its `usage()` with the native `launcher.UsageText()`. The shell version
carried a ~50-line `KEYBINDINGS` section — `Alt+Return`, `Alt+c`, `Alt+q`,
`Alt+/`, `Alt+x`, and an `Alt+h  pop up this help in a floating pane` line.
Recoverable verbatim: `git show 308d314^:bin/pair-shell`. The Go port kept the
binding and the pager and dropped the content.

It matters more than a typical doc gap because `Alt+h` is the **last** entry the
draft statusline preserves when the terminal narrows (`nvim/init.lua`,
`PAIR_CHEATS` — "At a minimum we try to keep Alt+h so the user always has a
discoverable path to the full keybind help"). It is the designed discovery path
for every other binding, and it is empty.

The Homebrew formula repeats the false claim in its `caveats`: "Run
`pair --help` for keybindings" — see **#131**.
