---
id: 000078
status: working
deps: [000077]
github_issue:
created: 2026-06-26
updated: 2026-06-30
estimate_hours: 3.12
started: 2026-06-30T15:58:17-07:00
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

Selected slice: port `pair-session-watch.sh` first. It owns session-id discovery, PID tree/lsof behavior, atomic restart-config writes, and adapt-log drift signals; those are high-value correctness and packaging surfaces with clear process-level fake coverage. `pair-title.sh` remains shell-owned for this issue and should become a follow-up because it owns a separate UI/title-poller surface: zellij frame titles, cmux workspace ownership, activity buckets, singleton poller identity, and session liveness. `ARCH-PURPOSE`: #78 is satisfied by porting a prioritized stateful subset and explicitly splitting the other stateful candidate instead of blending two long-running scripts into one review boundary.

## Done when

- [ ] A prioritized subset of stateful shell glue is ported or split into smaller issues.
- [ ] Ported behavior has process-level tests with fake external commands/files.
- [ ] Legacy script callers either route to Go or remain as compatibility shims.
- [ ] Short shell scripts with no packaging/reliability payoff are explicitly left alone.
- [ ] Pair remains usable after each merge.

## Plan

- [x] Choose the candidate from #73's priority table.
- [ ] Capture existing behavior in tests before porting.
- [ ] Port pure decision logic and thin IO seams.
- [ ] Keep compatibility shims until all callers move.
- [ ] Verify live or fake end-to-end behavior.

Detailed implementation plan: `workshop/plans/000078-go-stateful-shell-glue-plan.md`.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.20 impl=0.08
item: greenfield-go-module design=0.45 impl=0.55
item: skill-or-dispatcher design=0.35 impl=0.45
item: smaller-go-module design=0.15 impl=0.12
item: atlas-docs design=0.20 impl=0.10
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.20
total: 3.12
```

## Log

### 2026-06-26

Created from #72. This is intentionally later in the sequence; porting shell before the entrypoint shape is clear risks wasted work.

### 2026-06-30

Claimed after #77 landed. Selected `pair-session-watch.sh` as the #78 slice from the #73 migration inventory because it owns restart-config correctness and brittle PID/lsof/session-file discovery. Split `pair-title.sh` out of this issue: it remains stateful shell glue, but its UI title-poller ownership is a separate risk surface. `ARCH-DRY`: the plan centralizes agent watch patterns, id extraction, resume-arg stripping, and config JSON in Go helpers instead of scattering them across shell conditionals. `ARCH-PURE`: pure parsing and config helpers are tested without process IO; process discovery stays behind a fakeable runtime.
