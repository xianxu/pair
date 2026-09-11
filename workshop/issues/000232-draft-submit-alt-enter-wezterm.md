---
id: 000232
status: open
deps: []
github_issue:
created: 2026-09-11
updated: 2026-09-11
estimate_hours:
---

# <M-CR> submit from the draft does nothing under WezTerm: its default keymap binds Alt+Enter to ToggleFullScreen

## Problem

Operator report, 2026-09-11: `Alt+Return` in the draft nvim does not send
under WezTerm. The same build sends fine under iTerm2 and Terminal.app (and
Ghostty, the daily driver).

Root cause, verified against the installed WezTerm rather than inferred
(`wezterm 20240203-110809-5046fc22`, no user config present — defaults apply):

    $ wezterm show-keys | grep -i enter
        ALT                  Enter              ->   ToggleFullScreen

WezTerm's **default** key table assigns `Alt+Enter` to `ToggleFullScreen`. The
chord is consumed by the terminal before any bytes reach the pty, so nvim never
sees `<M-CR>` (`nvim/init.lua:3479`) and the KKP form pair-wrap also
recognizes (`\x1b[13;3u`, `wrapcmd/wrap.go:1308`) never arrives either. It is
not an Option-as-Alt problem: WezTerm treats the left Option as Alt by default,
and every other chord pair uses is clear — the only other bare-`ALT` defaults
are CopyMode-only (`Alt+b/f/m/←/→`) and mouse block-selection, none of which
apply in the normal key table. `Alt+Enter` is the single collision.

`<S-M-CR>` (`init.lua:3482`) is not a workaround: it is a different action
(append without send).

The README's *Terminal setup* table (`README.md:229-237`) covers Ghostty,
iTerm2 and Terminal.app, and only the Option-as-Alt dimension. It has no
WezTerm row, and nothing in `pair-doctor` detects a terminal that has claimed
one of pair's chords for itself.

## Spec

**The fix is documentation, per terminal — not code.** pair's chord is right;
what has to change is on the terminal's side, and it differs per terminal. For
WezTerm the default `Alt+Enter → ToggleFullScreen` binding has to go. Verified
by the operator 2026-09-11: with that one line, `Alt+Return` sends.

Two halves; the second is what stops the next terminal from being a report.

1. **Document it.** Add a WezTerm row to the *Terminal setup* table. Option
   handling needs nothing (left Option is Alt by default). The required
   setting is one line in `~/.wezterm.lua`:

   ```lua
   local wezterm = require 'wezterm'
   local config = wezterm.config_builder()
   config.keys = {
     { key = 'Enter', mods = 'ALT', action = wezterm.action.DisableDefaultAssignment },
   }
   return config
   ```

   State the symptom the way the existing paragraph does for the Unicode
   insertions: *the window toggles fullscreen and nothing is sent.*

2. **Detect it.** `pair-doctor` already checks the host; add a WezTerm probe:
   when `TERM_PROGRAM` is `WezTerm`, run `wezterm show-keys` and flag any
   assignment on a chord pair uses (`Alt+Return` today; check the full chord
   list from `pair keys`, not just Enter, so a future default collides
   loudly). Report the exact `config.keys` line to add. `show-keys` is
   WezTerm's own effective-keymap dump, so this reads the truth including the
   user's overrides rather than guessing from the version.

Out of scope: changing pair's send chord. `Alt+Return` is the send chord in
claude too, deliberately (`README.md:111`), and the collision is one terminal's
default, fixable in one line on that terminal.

## Done when

- README *Terminal setup* has a WezTerm row with the `DisableDefaultAssignment`
  line and the fullscreen symptom.
- `pair-doctor` under WezTerm with defaults reports the Alt+Enter collision and
  prints the fix; after the config line it reports clean. Test with a fake
  `show-keys` output on the seam, plus a live check gated like the other
  terminal probes.
- Manual: WezTerm with the one-line config, draft `Alt+Return` sends.

## Plan

- [ ] README: WezTerm row + symptom sentence in *Terminal setup*
- [ ] `pair-doctor`: WezTerm probe over `wezterm show-keys` against the `pair keys` chord list; fake-output test; live check
- [ ] Manual verification under WezTerm before and after the config line

## Log

### 2026-09-11

- Filed from the brain advisor session on the operator's report. Root cause
  taken from `wezterm show-keys` on the operator's machine (no `~/.wezterm.lua`
  or `~/.config/wezterm/wezterm.lua` present), not from documentation. Full
  bare-`ALT` default list checked: `Enter` is the only normal-mode collision.
- Sibling: pair#233 (kitty) is the other dimension — Option not Alt at all.
  The doctor probe in Spec 2 should be one chord-delivery check with a reader
  per terminal, covering both.
- **Operator verified the fix** (2026-09-11): `DisableDefaultAssignment` on
  `ALT+Enter` in `~/.wezterm.lua`, and the draft sends. Confirms the root
  cause and the shape of the fix — a per-terminal note, WezTerm's being
  "remove the default binding". Nothing in pair changes for the chord itself.
