---
status: done
actual_hours: 0.8
updated: 2026-05-31
---

# Codex slug does not update when session config is missing

## Spec

`pair codex` emits timely turn-end notifications, but the orientation slug can
stay stale when `config-<tag>-codex.json` is missing. `pair-slug` currently
requires that config to resolve the Codex rollout transcript, so a failed
session watcher leaves all later turn-end spawns as silent no-ops.

## Plan

- [x] Make Codex session discovery less dependent on `pgrep -P`.
- [x] Add a fallback so `pair-slug` can recover the live Codex rollout from
      `agent-pid-<tag>` when the saved config is absent.
- [x] Verify with unit tests and a manual live-session check.

## Log


- 2026-05-31: closed — env GOCACHE=/private/tmp/pair-go-cache make test; live watcher wrote config-pair-codex.json for session 019e8178-79c2-7862-91db-e8fa1be3b162; fake-model pair-slug reached model gate
- 2026-06-01: Live `PAIR_TAG=pair` / `PAIR_AGENT=codex` has no
  `config-pair-codex.json`; manual `pair-slug` logs
  `no session_id in config-pair-codex.json`. The live rollout is open in the
  native Codex child process.
- 2026-06-01: Fixed watcher descendant discovery to use `ps` instead of
  `pgrep -P`; on this host `pgrep -P 79965` missed the native Codex child while
  `ps` showed `79965 -> 80001`. Added `pair-slug` live Codex fallback through
  `agent-pid-<tag>` + descendant `lsof`, and covered it with a process-level
  fake test.
- 2026-06-01: Verified: `bash -n bin/pair-session-watch.sh`; patched watcher
  writes live `config-pair-codex.json` with session
  `019e8178-79c2-7862-91db-e8fa1be3b162`; fake-model `pair-slug` now reaches
  the model gate instead of `no session_id`; `env GOCACHE=/private/tmp/pair-go-cache make test` passes.
