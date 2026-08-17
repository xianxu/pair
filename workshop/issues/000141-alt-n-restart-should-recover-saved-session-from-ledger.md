---
id: 000141
status: codecomplete
deps: []
github_issue:
created: 2026-08-16
updated: 2026-08-16
estimate_hours: 0.35
started: 2026-08-16T22:37:52-07:00
actual_hours: 0.19
---

# Alt+n restart should recover live Codex session

## Problem

`Alt+n` restarted the Pair workbench but launched Codex without a native
`resume <session_id>` token, so the restarted agent opened a fresh conversation
instead of continuing the prior one. The live evidence is
`wrap-events-pair.jsonl` showing `argv=["codex","--no-alt-screen"]` on the
restart.

Root cause: plain restart writes only tag/agent/new-session intent into the
restart marker before killing the pane. Re-entry later composes resume args from
saved config only. For a long-running Codex pane whose native session id was not
captured in `config-<tag>-codex.json`, Pair had one last chance to inspect the
live Codex process and recover its open rollout transcript, but the restart path
did not do that before killing the session.

## Spec

On plain `Alt+n`, when the live agent is Codex and the restart is not a fresh
conversation, Pair should recover the current Codex session id from the live
process tree before killing the pane and carry that id through the restart
marker. Restart re-entry should prefer the marker session id over saved config
when composing `codex resume <session_id>`.

Keep the shelling-out at the launcher runtime seam and keep the restart planner
pure (`ARCH-PURE`). Reuse the existing Codex rollout/session-id parsing instead
of adding another ad hoc parser (`ARCH-DRY`). Tests should use fake runtime
state, not real processes (`ARCH-MOCK`).

## Done when

- `pair restart` for a live Codex pane writes a restart marker carrying the live
  Codex session id when one can be recovered.
- `Alt+n` restart re-entry composes `codex resume <marker-session-id>` when the
  saved config has an empty or stale session id.
- Existing restart behavior with a valid config session id still passes.
- Targeted launcher tests pass.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only; `sdlc estimate-source` reports the calibration
source as stale, so the number is provisional but uses the required method.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.10 impl=0.25
total: 0.35
```

## Plan

- [x] Add a failing launcher regression where `pair restart` sees a live Codex
  session id and assert the written restart marker carries it.
- [x] Add a failing restart re-entry regression where a marker session id wins
  over empty saved config and the second launch uses
  `codex resume <id> --no-alt-screen`.
- [x] Change restart capture/re-entry to carry marker session ids and recover
  live Codex rollout ids at the runtime seam.
- [x] Run targeted launcher tests and `git diff --check`.

## Log

### 2026-08-16
- 2026-08-16: closed — Fixed Alt+n Codex restart preservation by carrying live recovered Codex session ids through the restart marker; verified with go test ./cmd/internal/launcher ./cmd/internal/transcript ./cmd/internal/slugcmd -count=1, full go test ./... -count=1, and git diff --check.; review verdict: FIX-THEN-SHIP

- Investigated `Alt+n` route: Zellij forwards `Alt n` to Neovim
  `PairConfirmRestart`, which runs `pair restart`; restart re-entry then calls
  `planRestart` with saved config. Live scoped sidecars showed
  `wrap-events-pair.jsonl` launched the restarted Codex as `codex --no-alt-screen`
  (no resume token), while `adapt-pair.jsonl` later captured
  `session_id=01a00e37-16c4-7100-89fc-42ce26158f71`. Current scoped config now
  has that id, but the failed restart path demonstrates stale/empty saved config
  can suppress resume. `ARCH-DRY`/`ARCH-PURE`: fix should reuse the existing
  ledger/config parsing helpers instead of adding a second resume source.
- Revised the hypothesis after the operator clarified the failed `Alt+n` was in
  a long-running session, so late persistence alone should not explain it.
  Searched Codex transcript files around the earlier run and found no Pair-cwd
  transcript before the new 22:34 session; the actionable root cause is the
  restart marker/re-entry path not attempting live Codex rollout recovery before
  killing the pane. Existing `pair slug` code already performs live Codex
  transcript recovery from `agent-pid-<tag>` plus descendant `lsof`; restart
  should use the same shape at the launcher runtime seam.
- Added failing regressions for marker `session_id` round-trip, restart marker
  capture from a fake live Codex session, and Codex restart re-entry composing
  `resume SID-LIVE --no-alt-screen`. The initial red run failed at compile time
  because `RestartMarker` had no `SessionID` field.
- Implemented marker `session_id` parsing/serialization, made `planRestart`
  prefer marker session ids over saved config, added `Runtime.LiveAgentSessionID`
  for Codex live rollout recovery before kill, and moved Codex rollout session-id
  parsing into `cmd/internal/transcript` so `launcher` and `slugcmd` share it
  (`ARCH-DRY`, `ARCH-PURE`).
- Verified targeted packages with
  `go test ./cmd/internal/launcher ./cmd/internal/transcript ./cmd/internal/slugcmd -count=1`.
- Verified the full suite with `go test ./... -count=1` and checked whitespace
  with `git diff --check`.
- Updated `atlas/architecture.md` to map the new pre-kill Codex live recovery
  pass and the optional `session_id` restart-marker field.
- Addressed the boundary review's `FIX-THEN-SHIP` findings: moved shared
  process tree / `lsof` helpers into `cmd/internal/procutil` so launcher and
  `slugcmd` derive from one implementation (`ARCH-DRY`), added direct
  `OSRuntime.LiveAgentSessionID` coverage with fake `ps`/`lsof`, and refreshed
  the stale `runRestart` comment.
- Re-verified after review fixes with
  `go test ./cmd/internal/launcher ./cmd/internal/transcript ./cmd/internal/slugcmd ./cmd/internal/procutil -count=1`,
  `go test ./... -count=1`, and `git diff --check`.
