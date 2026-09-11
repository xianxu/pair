---
id: 000233
status: open
deps: []
github_issue:
created: 2026-09-11
updated: 2026-09-11
estimate_hours:
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

## Spec

1. **Document it.** Add a kitty row to the *Terminal setup* table:

   | **kitty** | `macos_option_as_alt yes` in `~/.config/kitty/kitty.conf` | `no` | set it; `left` if the right Option should keep composing |

   kitty's doc notes a reload does not apply this option — restart kitty. Say
   so in the row, since "I changed it and it still prints `≈`" is the next
   report.

2. **Detect it.** In `pair-doctor`, when `TERM_PROGRAM` is `kitty`, read the
   effective value with kitty's own runtime — `kitty +runpy` over
   `kitty.options.utils`/the loaded config, or `kitten @ get-config`-style
   query where available — and flag `no` with the exact line to add. Prefer
   asking kitty over parsing `kitty.conf` by hand: the effective value
   includes includes and overrides.

   Fold this into the same probe #232 adds for WezTerm: one "does this
   terminal deliver pair's chords" check, two terminal-specific readers.

Out of scope: pair-side chord changes, or decoding `≈` as if it were `Alt+x`.
Ghostty, iTerm2 and Terminal.app users get Alt via one setting; kitty is the
same setting under a different name, and guessing from composed characters
would misfire on anyone who types them on purpose.

## Done when

- README *Terminal setup* has a kitty row (setting, default, restart note).
- `pair-doctor` under kitty with defaults reports the Option-as-Alt problem
  and prints the config line; after setting it (and restarting kitty) reports
  clean. Fake-output test on the seam plus a live check gated like the other
  terminal probes.
- Manual: kitty with `macos_option_as_alt yes`, `Alt+x` opens the quit
  confirmation; `Alt+d/n/i` also work.

## Plan

- [ ] README: kitty row + restart note in *Terminal setup*
- [ ] `pair-doctor`: kitty reader for the effective `macos_option_as_alt`, sharing the chord-delivery probe with #232
- [ ] Manual verification under kitty before and after

## Log

### 2026-09-11

- Filed from the brain advisor session on the operator's report, same session
  as pair#232. Default read from kitty's runtime (`kitty +runpy … defaults`)
  and doc text from the bundled `kitty.conf` under
  `kitty.app/Contents/Resources/doc/`; `--debug-config` is not a flag on
  0.48.2. No user config present.
