---
id: 000232
status: open
created: 2026-09-11
updated: 2026-09-11
estimate_hours:
github_issue:
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
