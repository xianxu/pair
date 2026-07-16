---
id: 000069
status: done
deps: []
github_issue:
created: 2026-06-24
updated: 2026-06-24
estimate_hours:
started: 2026-06-24T14:09:23-07:00
---

# Fix cmux workspace title update: poller respawn + stale ownership

## Problem

The cmux workspace title (the activity-heat prefix `🔴/🟠/🟡/🔵 pair-<tag>`
maintained by `bin/pair-cmux-title.sh`) stopped updating. Two independent
defects, surfaced while debugging a live session (tag 211) whose title was
frozen at `🔴 ♋-♋`:

1. **Poller death is not reliably healed.** The poller is a long-lived
   background process spawned only at `pair` create/attach/restart. A host
   sleep / reboot / SIGKILL kills it mid-session (all pidfiles were found
   pointing at dead PIDs), and nothing respawns it until the next entry. Worse,
   the single-instance guard used a bare `kill -0 $old_pid`: after a reboot the
   kernel can recycle the dead poller's PID onto an unrelated live process, so
   the guard reads "still alive" and **suppresses the respawn even across a pair
   restart**.

2. **Stale workspace ownership freezes the title.** `cmux_rename_workspace`
   deferred whenever a *different* tag's `pair-<owner>` zellij session existed
   *anywhere*. When a cmux workspace is reused by a new tag while the old owner
   lives on in a *different* workspace, the owner file (`cmux-owner-<wsid>`)
   goes stale and the new occupant defers to a ghost — permanently. This is why
   tag 211 showed `🔴 ♋-♋` (= `pair-pair`): a stale `pair` owner blocked it.

(Note: cmux itself is healthy — `cmux ping` → PONG. The broken-pipe seen from
the agent Bash tool is sandbox-only; the poller runs in the real pane shell.)

## Spec

- A dead/recycled poller must be reliably revived on the next pair
  create/attach/restart (the user's stated trigger).
- A session must own the title of the cmux workspace it is provably running in,
  even if a stale owner file names a different (live-elsewhere) tag.
- No regression to the genuine two-pairs-in-one-workspace case: it degrades to
  last-launcher-wins and self-heals on exit, rather than a permanent freeze.
- Scope intentionally excludes self-healing a poller that dies *mid-session*
  with no subsequent entry (a possible later follow-up: a heartbeat from the
  long-lived pair-wrap). [[ARCH-DRY]] the duplicated poller-spawn snippet.

## Done when

- Poller single-instance guard is identity-checked (not a bare `kill -0`), so a
  recycled PID cannot wedge the respawn.
- `bin/pair` ensures the poller on every entry via one helper (create + attach +
  restart), replacing the duplicated spawn.
- `cmux_rename_workspace` claims ownership unconditionally (presence beats a
  stale flag).
- Automated tests cover both fixes; `make test-cmux-title test-cmux-ownership`
  green; relocation does not regress the existing `bin/pair` helper suite.

## Plan

- [x] Harden poller guard (`poller_alive` identity check) + `ensure_cmux_title_poller` + unconditional ownership claim, with tests

## Log

### 2026-06-24
- 2026-06-24: closed — make test-cmux-title + test-cmux-ownership pass; pair-continue-test (relocation regression) green; bash -n clean; live repro fixed — stale cmux-owner corrected so the running 211 poller reclaimed the title. (--no-actual: active-time-v3 engine absent in this checkout.); review verdict: SHIP

- Diagnosed live: all `cmux-title-pid-*` pollers dead with surviving pidfiles
  (sleep/SIGKILL signature). `cmux ping` PONG → cmux healthy; agent-shell
  broken-pipe is sandbox-only.
- Root causes: (1) bare `kill -0` guard + spawn only at entry; (2) ownership
  defer to a live-elsewhere owner via a stale `cmux-owner-<wsid>` file.
- Fix: `poller_alive` (ps-identity, trailing-space tag match) in
  `bin/pair-cmux-title.sh`; `ensure_cmux_title_poller` in `bin/pair` routing
  create+attach (restart re-execs through create); relocated the cmux ownership
  helpers into the early-helpers block (#55 hoist pattern) so
  `cmux_rename_workspace` is `PAIR_TEST_CALL`-testable; made the claim
  unconditional. session_blocks_reuse still used by tag-collision paths.
- Tests: `tests/cmux-title-poller-test.sh` (match / recycled-PID / dead / tag
  prefix / empty), `tests/cmux-ownership-test.sh` (claim over stale owner / ♋
  substitution / unowned / no-op outside cmux). Hit the make-test session-env
  leak — scrubbed `CMUX_WORKSPACE_ID` with `env -u` in the outside-cmux case.
- Immediate user remedy applied live: corrected stale `cmux-owner-<2861…>` to
  `211` so the running poller reclaims the title.
- Landed via clean PR off `origin/main` (local `main` diverged: ahead 10 incl.
  #68 WIP, behind 6 = origin's #67/PR-37 — left untouched per user). `bin/pair`
  3-way-merged with no conflict; only `Makefile.local` needed trivial resolution.
- Boundary review SHIP (high), 0 Critical / 0 Important. Addressed the one
  doc-accuracy minor (softened the "self-heals on exit" atlas wording).
- Deferred follow-ups flagged by review: (a) extract emoji-substitution /
  owner-file logic into a shared lib *before* any mid-session poller-death
  heartbeat lands (else a 3rd copy); (b) the poller's in-loop owner-defer uses a
  bare `grep -qx` (an EXITED resurrect row would satisfy it) — harden via
  `session_blocks_reuse` if the two-pairs path is revisited.
