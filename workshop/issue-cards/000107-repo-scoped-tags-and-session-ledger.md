---
id: '000107'
status: done
started: 2026-07-07T11:23:27-07:00
created: 2026-07-07
updated: 2026-07-07
estimate_hours: 8.2
actual_hours: 7.55
---

# repo-scoped tags and session ledger

## Problem

pair started as a thin wrapper around claude, grew agent-neutral over time, but
never fully committed to it — so its identity model still **conflates two
independent axes**: *which piece of work* (the tag) and *which agent runs it*.
The picker exposes only the first and silently resolves the second, which
produces concrete bugs and a fragile repo binding.

**Originating bug.** `pair-dev codex -- --sandbox danger-full-access` launched
**claude** with codex's `--sandbox` flag and failed to start. Root cause:
`DecideLaunch` ignores the agent entirely (`decision.go:24`); with history
present it returns `ActionPick`. Picking an existing tag is resume-by-name, so
`runOnce` resets the agent to `""` and re-infers it from disk
(`createflow.go:158-171`) — landing on the tag's last agent (claude) — but the
CLI-forwarded `AgentArgs` (codex-only `--sandbox danger-full-access`) ride along
onto claude. The zellij layout then runs `pair wrap … claude --sandbox …` →
claude chokes. Confirmed by a failing repro test
(`TestRunLaunchPickInferredAgentMustNotInheritCliArgs`): `PAIR_AGENT="claude"`
while `PAIR_AGENT_ARGS` still carried `--sandbox danger-full-access`.

**Structural gaps behind it:**

- **Agent is not a picker axis.** The session picker shows one row per *tag*
  (`pair-<tag>`) with no agent shown; the agent is only inferred *after*
  selection via `InferAgent(tag)`. So the picker can't tell you `pair-work` is a
  *claude* tag — you can't avoid the mismatch.
- **Repo is not a real scope dimension.** History is filtered only by *tag-name
  prefix* where `base = DefaultTag(cwd)` = the cwd **basename**
  (`history.go:74`) — a fragile proxy for "same repo" (`~/work/pair` and
  `~/other/pair` collide). Live detached sessions aren't filtered at all: any
  detached `pair-*` session, from any repo, both triggers the picker and shows
  as a row (`decision.go:34`, `pick.go:48`). The cwd *is* recorded
  (`pane-<tag>-<agent>.json`, claude's transcript path) but nothing uses it to
  scope.
- **Only the last session per (tag, agent) is retained** — a single
  `config-<tag>-<agent>.json` overwritten each launch. There's no record of
  prior sessions or of cross-agent activity within a tag.
