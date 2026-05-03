---
id: 000011
status: working
deps: []
created: 2026-05-03
updated: 2026-05-03
---

# Outer-PTY passthrough for OSC notifications

## Problem

When pair runs inside an outer wrapper (e.g. cmux) that watches the agent's PTY stream for OSC 9 / OSC 777 escapes to surface "agent needs attention" indicators, zellij silently eats those escapes. Empirically verified: OSC 9 and OSC 777 from inside a zellij pane do NOT reach cmux's stream watcher; the same escapes from a plain shell inside cmux do.

Zellij 0.44 forwards BEL (verified via Apple Terminal — `printf '\a'` rings whether zellij is in the chain or not), but cmux watches OSC, not BEL. So the BEL passthrough doesn't help for cmux integration.

## Spec

Give pane-internal hook scripts a way to emit notification escapes that reach the *outer* terminal/wrapper, bypassing zellij's escape filtering.

Mechanism: pair captures its controlling TTY *before* invoking zellij — that path IS the outer PTY — and writes it to a file at a known location keyed by `PAIR_TAG`. A small helper `pair-notify` reads the file and writes an OSC sequence directly to that path.

Why a file (not env var): pane shells inherit env at session-creation time only. On detach + reattach the outer PTY changes; env stays stale. A file gets refreshed on every attach.

## Plan

- [ ] `bin/pair`: capture `tty` after `PAIR_TAG` is finalized on both attach and create branches; write to `$DATA_DIR/outer-tty-${PAIR_TAG}`. Remove the file alongside `cleanup_quit_marker` on full quit.
- [ ] `bin/pair-notify`: new helper. `pair-notify "msg"` → writes OSC 9 to the recorded outer TTY. `--osc 777` switches to urxvt variant. No-op + non-fatal if file or path is stale (so a misconfigured hook never blocks the agent).
- [ ] README: short subsection "Notifications via outer-PTY passthrough" with sample Claude Code `Notification` / `Stop` hook config that calls `pair-notify`.
- [ ] `atlas/architecture.md`: paragraph on the capture-and-file mechanism near existing PTY/zellij notes.
- [ ] Verify: from inside a pane, `printf '\033]9;hi\007' > "$(cat $PAIR_DATA_DIR/outer-tty-$PAIR_TAG)"` triggers cmux badge. Detach + reattach + repeat → still works.

## Log

### 2026-05-03

Filed after empirical investigation showed zellij filters OSC 9/777 (drops them as unrecognized escapes during virtual-screen reconstruction). BEL forwarding (added in zellij 0.44 per #776/PR #981) works but isn't useful for cmux which only watches OSC.
