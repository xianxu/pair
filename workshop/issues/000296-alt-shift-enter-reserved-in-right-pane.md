---
id: 000296
status: open
deps: [pair#227]
github_issue:
created: 2026-09-20
updated: 2026-09-20
estimate_hours:
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

## Spec

`Shift+Alt+Return` is intercepted by `pair term` whenever the right pane holds
focus, regardless of what the active tab's child is doing with the screen. It
invokes `ActionToggleFocusedLayout` and is **not** forwarded to the child.

Scope boundaries:

- Right-pane focus only. Draft and agent focus are unchanged — the chord has no
  global binding and this issue does not add one. `layoutcmd.RunToggleFocused`
  is already a no-op unless a right terminal is focused
  (`layoutcmd.go:251-254`), so a global promotion would be a different, larger
  feature.
- Every other role-scoped chord keeps #227 passthrough unchanged. This issue
  adds exactly one reservation.
- `Alt+k` keeps its reservation and its reason.

Design decision to make in the plan — **do not add a second hard-coded
exception to `RightTerminalChordPassesThrough`.** One ad-hoc `chord !=
ChordAltK` was defensible; two is the moment the reservation becomes a property
of a chord rather than a clause in a predicate (ARCH-DRY). Recommended shape: a
`PairReserved` field on the binding data, with the predicate derived from it and
help/key metadata rendering the same field — #257's rule, "describe the
exception from the same binding source used for routing; do not add a second
hand-maintained list". Note that `roleBindings` is currently *descriptive* by
contract (`shortcut.go:201-211`: "The table describes the switch; it does not
drive it"), so making it drive routing changes that contract and its comment
must be updated, or the reservation must live in a source that already drives.
Pick one and say why; keep the decision pure and testable, with the pump
retaining ownership of the effect (ARCH-PURE).

Encoding gap to resolve in the same pass: `ChordAltShiftEnter` registers only
`\x1b[13;4u`. `shortcut.go:423-428` establishes the rule that both modifier
families are registered — bit-2 "alt" gives modifier 3/4, bit-8 "meta" gives
9/10 — precisely so a chord is not silently dead on a meta-style terminal, and
`TestMetaSiblings` (`shortcut_test.go:~620`) enforces it for the chords it
covers. The `;10u` sibling is absent. Confirm whether it is genuinely
unreachable for this key or an oversight, and register it if reachable.

Open question, not a blocker for the above: `\x1b[13;4u` is a Kitty-keyboard
encoding, and Pair pushes `\x1b[>3u` to the host from the presenter
(`terminal/presenter.go:847`). On a host terminal without KKP, `Shift+Alt+Enter`
is likely indistinguishable from `Alt+Enter` at the byte level, which would make
this chord host-dependent in the same way #232/#233 are. Establish which hosts
deliver it before promising the behaviour in the README's *Terminal setup*
table; if it is host-dependent, that is a documentation row, not code.

## Done when

- With the right pane focused on a full-screen TUI (nvim, or any child for which
  `activeChildOwnsScreen()` is true), `Shift+Alt+Return` toggles the focused
  side's width and the child receives no bytes for the chord.
- With the right pane focused on a plain shell, the chord behaves exactly as it
  does today.
- Every other role-scoped right-terminal chord still reaches a full-screen
  child: `Alt+t`, `Alt+w`, `Alt+r`, `Alt+Shift+D`, `Alt+Left`, `Alt+Right` at
  minimum.
- `Alt+k` still escapes to the left stack under a full-screen child.
- `TestRightTerminalChordPassesThrough` no longer lists `ChordAltShiftEnter` as
  passing through, and asserts the reservation with the same wording style as
  the `Alt+k` case (the *reason*, not just the site).
- The reservation is expressed once: a guard fails if a reserved chord is
  reachable through the passthrough predicate, or if help/key metadata describes
  a reservation the router does not implement.
- Behaviour is asserted at the production pump boundary (`termcmd`, the
  `ptyWriter`/`activeChildOwnsScreen` seam), not only in the pure shortcut table.
- `Alt+Shift+Enter`'s meta-family sibling is either registered or its absence is
  documented at the table with the reason.
- Help output (`Alt+h`) describes the chord as Pair-owned in the right pane, and
  the description derives from the routing source.
- Operator smoke-tests it live in `~/workspace/pair`: nvim in the right pane,
  chord resizes, nvim shows no stray input.

## Plan

- [ ] Reproduce at the pump: a `termcmd` test with `activeChildOwnsScreen()`
      true showing the chord reaching the child today (red).
- [ ] Decide where the reservation lives (see Spec) and record the choice in the
      Log; implement the derived predicate.
- [ ] Move `ChordAltShiftEnter` out of the passthrough list in
      `TestRightTerminalChordPassesThrough`, add the reservation assertion and
      the "reserved chords cannot pass through" guard.
- [ ] Resolve the `;10u` meta sibling: register it, or document why not.
- [ ] Update help/key metadata + `atlas/terminal.md` if the routing contract
      wording changes.
- [ ] Run the focused `workbenchshortcut` + `termcmd` tests, then the full
      `make test`; ask the operator to smoke-test live before closing.

## Log

### 2026-09-20

Filed from an operator request: intercept `Shift+Alt+Return` in the right pane
even under a TUI, because pane resizing is conventionally a workbench operation
and Pair's broad passthrough is otherwise deliberate.

Traced before filing rather than assumed. The chord, its decode, its role
binding and its action all exist already; the only missing piece is the #227
passthrough exception. `TestRightTerminalChordPassesThrough` currently asserts
the opposite behaviour, so the change is a contract flip in a green test, not a
gap in coverage. Also noted the absent `;10u` meta sibling and the KKP
dependency of `\x1b[13;4u` as adjacent questions.
