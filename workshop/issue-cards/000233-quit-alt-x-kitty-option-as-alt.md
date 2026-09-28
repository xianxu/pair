---
id: 000233
status: open
created: 2026-09-11
updated: 2026-09-11
estimate_hours:
github_issue:
---

# <M-x> quit does nothing under kitty: its macOS default keeps Option as a composing key (macos_option_as_alt no), so Alt+x arrives as ≈

## Problem

Operator report, 2026-09-11: `Alt+x` (quit, `ChordAltX` →
`ActionConfirmQuit`, `workbenchshortcut/shortcut.go:149`) does nothing under
kitty. Works under Ghostty, iTerm2 and Terminal.app.

Root cause, verified against the installed kitty rather than inferred
(`kitty 0.48.2`, no `~/.config/kitty/kitty.conf` — defaults apply):

    $ kitty +runpy "from kitty.options.types import defaults; print(defaults.macos_option_as_alt)"
    0

kitty's macOS default is `macos_option_as_alt no`. Its own documentation says
what that does: *"kitty will use the macOS native Option+Key to enter Unicode
character behavior. This will break any Alt+Key keyboard shortcuts in your
terminal programs."* So `Alt+x` reaches the pty as `≈`, not as `\x1bx` or the
KKP form `\x1b[120;3u` that pair decodes (`shortcut.go:357`). This is
precisely the symptom the README's *Terminal setup* section already describes
(`README.md:239`: "`Alt+x` prints `≈` in nvim…") — kitty is simply not in the
table, so nobody is told to flip it. kitty has no default mapping on `alt+x`
itself; the chord never becomes Alt at all.

Consistent with the report naming only `Alt+x`: `Alt+Return` has no macOS
composed character, so it still sends (the README notes this), which makes
the failure look chord-specific when it is terminal-wide — every `Alt+letter`
chord (`x/d/n/N/i/l/q/h/b`) is broken the same way.

Sibling of pair#232 (WezTerm), which is the *other* dimension: there Option is
Alt by default and a terminal default keybinding steals one chord. Two
terminals, two different reasons a chord never arrives; the doctor probe
should cover both.
