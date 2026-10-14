---
id: 000547
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-14
estimate_hours: 3.59
card_mirror: '2a468a35224412ddcce31958cdfe6960ac8ed1d2' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T00:00:06-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# launcher: five call sites exec setuid /bin/ps; read the process table through procutil instead (works under sandboxes)

## Problem

pair#528 M1's boundary review looped ten rounds because its sandbox couldn't run `/bin/ps`. ariadne:3 measured why (ariadne#364, 2026-10-14): on macOS `/bin/ps` is **setuid root**, and seatbelt refuses to exec setuid binaries even under `(allow default)`; non-setuid `pgrep`, which reads the process table in-process, works under the same profile. pair still shells out to `ps` in five places:
- `cmd/internal/launcher/osruntime.go:903` (`ps -ww -A -o args=`), `:1113` (`ps -o comm= -p`), `:1121` (`ps -o ppid= -p`)
- `cmd/internal/launcher/session_quiescence.go:129` and `session_servers.go:147` (`ps -axo pid=,command=`)

So these paths fail under any sandboxed reviewer or agent, and they duplicate `procutil.Table`, which already reads every process in one `kern.proc.all` sysctl (ARCH-DRY).

## Spec

Route all five through `procutil` (extend it for argv/command where it lacks it: `kern.procargs2` on darwin, `/proc/<pid>/cmdline` on linux), with one snapshot per decision where several facts must describe the same moment. Keep behaviour identical; no `ps` exec remains in non-test code (add a test or lint that fails on `exec.Command("ps"`).

## Done when

- No non-test code execs `ps`; the five call sites use procutil, with tests on both platforms' table readers.
- The process-dependent tests pass inside a seatbelt profile that denies setuid exec (e.g. `sandbox-exec` with `(allow default)`), demonstrating reviewer-sandbox compatibility.

## Plan

- [ ] Implement and verify the native process inspection plan: [plan](../plans/000547-native-process-inspection-plan.md).

## Log

### 2026-10-13

### 2026-10-14 — design

Claimed and started from fresh main. Hub approved sweeping all production ps calls and full flow. Plan above records native argv/executable distinction, per-row uncertainty and snapshot identity fencing (ARCH-DRY, ARCH-ORDER). Prior continuation commit remains on its own branch; untracked pensive preserved. #528 overlap coordinated with pair:2.

## Revisions

- 2026-10-14: enumerated the full no-ps contract beyond the five launcher sites: procutil helpers/unsupported fallback and diagnostic runtime inventory are in scope, approved by the hub.

- Plan-quality round 1 requested named risky functions and mechanical test strategies (PQ-1), plus explicit non-goals (PQ-2). Appended concrete interfaces, strategy table, composition seams and scope boundaries to the plan. Baseline seatbelt probe: TestCommand skips because ps is unavailable, while TestTableMatchesIdentityAndParent passes; replacement must remove that skip on supported OSes.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only; calibration is marked stale and provisional. Native adapters reuse x/sys/unix and existing procfs/sysctl patterns; no greenfield library is needed. Thorough reviewed plan applies 0.2 design discount to implementation primitives, implementation values use v3.1's 0.4 scale, familiarity 1.0. Two API items represent Darwin and Linux adapters plus conformance; two refactors represent launcher and diagnostic/composition migrations. Spec item accounts for the actual cross-consumer shaping rather than discounting it away.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.70 impl=0.08
item: api-integration design=0.20 impl=0.60
item: api-integration design=0.20 impl=0.60
item: cross-cutting-refactor design=0.12 impl=0.20
item: cross-cutting-refactor design=0.12 impl=0.20
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.20
design-buffer: 0.15
total: 3.59
```
