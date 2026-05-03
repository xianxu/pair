---
id: 000005
status: working
deps: [000001]
created: 2026-05-02
updated: 2026-05-02
---

# Alt-X to quit

Now that we have Alt+d to detach, add Alt+x to quit. "Quit" means kill all subprocesses *and* delete the session entry on the way out — vs. zellij's default Ctrl+q which kills the session but keeps it in the resurrect list.

## Spec

**Two quit semantics, both supported:**
- **Ctrl+q** (zellij default) — kill session, leave it as a resurrect candidate. Picker still surfaces it under "+ new" since it counts as detached.
- **Alt+x** (this issue) — kill session AND remove its entry from the resurrect list. Truly gone after exit.

**Mechanism:**
1. New script `bin/pair-quit.sh` writes a marker file `~/.cache/pair/quit-<SESSION>` then runs `zellij kill-session $ZELLIJ_SESSION_NAME`.
2. Zellij keybind `Alt+x` → `Run "pair-quit.sh"`. Single action, no parallel `Quit`, so no race between marker write and session kill.
3. `bin/pair` no longer `exec`s zellij — it runs zellij as a child, then on return calls `cleanup_quit_marker`. The cleanup checks for the marker; if present, removes it and runs `zellij delete-session --force $SESSION`.

Plain Ctrl+q leaves no marker, so the cleanup is a no-op and the session stays in resurrect list (current behavior preserved).

**Why `kill-session` rather than `Quit` action?** Zellij's keybind action `Quit` would require a parallel `Run` to write the marker, with race-y ordering. `kill-session` from inside a script is sequential after the marker write, deterministic.

## Plan

- [x] `bin/pair-quit.sh` — touch marker, exec `zellij kill-session $ZELLIJ_SESSION_NAME`.
- [x] `zellij/config.kdl` — `Alt x` → `Run "pair-quit.sh"`. (Resolved by PATH since bin/pair prepends `$PAIR_HOME/bin`.)
- [x] `bin/pair` — drop `exec` from both zellij invocations (attach branch and create branch); add `cleanup_quit_marker()` and call after each.
- [x] `bash -n` and `setup --check` clean.
- [ ] Manual smoke test: launch pair, Alt+x, confirm session is gone from `zellij list-sessions` (vs. lingering in EXITED state after Ctrl+q).

## Log

### 2026-05-02

Filed by user noting Alt+x bind was added without bookkeeping. Implementation done in same patch. Marking working until user verifies the cleanup actually removes the session from the resurrect list (vs. just killing it).
