---
id: 000070
status: done
deps: []
github_issue:
created: 2026-06-25
updated: 2026-06-25
estimate_hours:
started: 2026-06-25T09:12:47-07:00
---

# Fix Codex session id capture with stale pidfile

## Problem
In a live Codex pair session, Alt+x shows `session id: <not captured>` even
though Codex exposes a native thread/session id and has a rollout transcript on
disk.

## Spec
Pair should capture the Codex session id for a new launch even when
`agent-pid-<tag>` already exists from a prior launch. The watcher must bind to
the pidfile written for the current launch, not stale state.

## Done when

- Alt+x can read `config-<tag>-codex.json` for the current Codex session.
- A regression test covers stale pidfile replacement.

## Plan

- [x] Reproduce and trace the live failure path.
- [x] Add a failing regression for stale `agent-pid-<tag>`.
- [x] Make `pair-session-watch.sh` wait for a fresh pidfile before binding.
- [x] Wire the regression into `make test`.

## Log

### 2026-06-25
- 2026-06-25: closed — Fixed stale pidfile race in Codex session watcher. Verified bash -n bin/pair bin/pair-session-watch.sh tests/pair-session-watch-test.sh; make test-session-watch; git diff --check. Full env -u PAIR_SESSION_ID -u PAIR_TAG make test reached review-apply-test and failed with empty result file after headless nvim exited 0, outside the session-watch path. --no-actual because sdlc actual reported no measurable activity for this short same-turn issue.; review verdict: FIX-THEN-SHIP
- Live evidence: `CODEX_THREAD_ID=019eff64-6ceb-7e72-9d41-a735a97029ac`, but
  `PAIR_SESSION_ID` is empty and `config-211-codex.json` is absent.
- The matching Codex rollout exists at
  `~/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-019eff64-6ceb-7e72-9d41-a735a97029ac.jsonl`.
- Root cause: `pair-session-watch.sh` treated any non-empty `agent-pid-<tag>`
  as ready. A stale pidfile could be read before `pair-wrap` overwrote it,
  causing the watcher to inspect a dead/old PID and exit before the current
  Codex process opened its rollout file.
- Red/green evidence: `bash tests/pair-session-watch-test.sh` failed before the
  fix because `config-test-codex.json` was never written; it passes after the
  watcher waits for a fresh pidfile.
- Verification: `bash -n bin/pair bin/pair-session-watch.sh
  tests/pair-session-watch-test.sh` PASS; `make test-session-watch` PASS; `git
  diff --check` PASS.
- Full suite note: `env -u PAIR_SESSION_ID -u PAIR_TAG make test` reached the
  review integration tests, then failed in `tests/review-apply-test.sh` with an
  empty result file after headless nvim exited 0. The failure is outside the
  session-watch path and was not introduced by the touched files.
- Repaired the live tag `211` config manually with session id
  `019eff64-6ceb-7e72-9d41-a735a97029ac`, so Alt+x has the current value now.
- Boundary review follow-up: updated `atlas/architecture.md` to describe the
  pidfile mtime freshness gate and `PAIR_SESSION_WATCH_PID_WAIT_SECONDS`.
