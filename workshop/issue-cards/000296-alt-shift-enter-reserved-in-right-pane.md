---
id: '000296'
status: wontfix
created: 2026-09-20
updated: 2026-09-20
---

# Alt+Shift+Enter is Pair-owned in the right pane, even under a full-screen child

## Problem

Operator report, 2026-09-20: `Shift+Alt+Return` with the right pane focused
does nothing while a TUI is running there. It works when the right pane shows a
plain shell.

The chord already exists and is already wired. `ChordAltShiftEnter`
(`workbenchshortcut/shortcut.go:52`) decodes from `\x1b[13;4u`
(`shortcut.go:437`), is a `PaneRoleRightTerminal` role binding — "toggle the
focused side's width" (`shortcut.go:224`) — and `handleTerminalChord` runs
`layoutcmd.RunToggleFocused` for it (`termcmd/run.go:631`).

What eats it is #227's full-screen passthrough. The pump forwards a role-scoped
chord to the child whenever the active tab owns the screen:

    // termcmd/run.go:516
    if workbenchshortcut.RightTerminalChordPassesThrough(chord) && mux.activeChildOwnsScreen() {
        mux.writeEvents([]terminal.InputEvent{event})
        continue
    }

and the predicate has exactly one exception:

    // workbenchshortcut/shortcut.go:398
    func RightTerminalChordPassesThrough(chord Chord) bool {
        return chord != ChordUnknown && !IsGlobalChord(chord) && chord != ChordAltK
    }

`ChordAltShiftEnter` is neither `ChordUnknown`, nor a global (it is not in
`globalBindings`), nor `ChordAltK` — so it passes through, and Pair never
resizes. This is asserted as intended behaviour today:
`TestRightTerminalChordPassesThrough` (`shortcut_test.go:645-647`) lists
`ChordAltShiftEnter` among the chords that *should* reach a full-screen child.
The fix flips an existing green assertion; the test moves, it is not merely
extended.

This is the same shape as #258 (global tab chords lost under full-screen
passthrough), for a chord that is not a tab command.

Why this chord and not a general retreat from passthrough: Pair passes most
keystrokes through on purpose, so a TUI in the right pane behaves like it would
in a bare terminal. `Shift+Alt+Return` is the exception worth carving out
because pane resizing is a *workbench* operation, not an application one — the
chord is conventionally owned by the multiplexer/window manager, and there is
no from-anywhere equivalent (unlike the tab chords, which have `M-S-left/right/t`).
That makes it the same argument `Alt+k` already won: an operator running a TUI
on the right currently cannot resize without first leaving the pane.
