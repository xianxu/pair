---
id: 000078
status: open
deps: []
github_issue:
created: 2026-06-26
updated: 2026-06-26
estimate_hours:
---

# pair Go stateful shell glue

## Problem

After the public entrypoint is Go-owned, remaining stateful shell scripts can keep packaging brittle and hide reliability bugs. The biggest candidates are long-running or session-observing scripts, not short native glue.

## Spec

Port stateful shell glue where the packaging or reliability payoff is clear. Candidates include:

- `pair-title.sh` — long-running poller, pane title/context meter, cmux title ownership.
- `pair-session-watch.sh` — session-id discovery, PID tree/lsof behavior, atomic config write.
- review readiness/target helpers if #73 finds packaging benefit.
- opener scripts only when the Go entrypoint can replace their orchestration cleanly.

This issue may be split further if #73 shows the candidates are too large. Keep native assets native. Do not port Lua or zellij KDL into Go.

## Done when

- [ ] A prioritized subset of stateful shell glue is ported or split into smaller issues.
- [ ] Ported behavior has process-level tests with fake external commands/files.
- [ ] Legacy script callers either route to Go or remain as compatibility shims.
- [ ] Short shell scripts with no packaging/reliability payoff are explicitly left alone.
- [ ] Pair remains usable after each merge.

## Plan

- [ ] Choose the candidate from #73's priority table.
- [ ] Capture existing behavior in tests before porting.
- [ ] Port pure decision logic and thin IO seams.
- [ ] Keep compatibility shims until all callers move.
- [ ] Verify live or fake end-to-end behavior.

## Log

### 2026-06-26

Created from #72. This is intentionally later in the sequence; porting shell before the entrypoint shape is clear risks wasted work.
