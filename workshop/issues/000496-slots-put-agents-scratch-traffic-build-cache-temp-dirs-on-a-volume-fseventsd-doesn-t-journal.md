---
id: 000496
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'ab41b8d568c4af3b5e59d91ff5574b8fe7fd8879' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T13:29:47-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# slots: put agents' scratch traffic (build cache, temp dirs) on a volume fseventsd doesn't journal

## Problem

10-12: fseventsd (macOS's file-change journal, feeding Spotlight, Time Machine and file watchers) grew to gigabytes of memory and over 100% CPU under agent churn: Go's build cache (~86 GB; ~18,000 files changed per hour) and agent and test temp space (~11,000 per hour; test suites create and delete thousands of temp git repos). It slowed the machine and Couch; the operator restarted it by hand. File watching is tuned for human-scale change, not a fleet of agents.

## Spec

Keep the churn where nothing watches it:
- a dedicated APFS volume (e.g. `/Volumes/AgentScratch`, case-sensitive optional) with `.fseventsd/no_log` at its root and excluded from Spotlight (`mdutil -i off`);
- slots point `GOCACHE`, `TMPDIR` (and the test temp roots sdlc and pair use) at it, through the slot environment Couch/pair sets for a wrapper (or weave's compiled environment), never by editing the operator's shell profile;
- measure before and after (fseventsd CPU and memory, files changed per hour on the main volume) during a normal multi-slot workload;
- document setup and teardown in the atlas.
**Operator permission (the operator, 10-12):** every operation that changes this machine (creating or mounting a volume, moving or deleting caches or logs, changing launch settings, renicing processes) is proposed first, with the exact command, and runs only after the operator approves it, through the TL. Code changes and tests in the repo don't need that; anything outside the repo does.

## Done when

- The volume exists (created with the operator's approval), slots use it, and a multi-slot workload shows fseventsd's CPU and memory staying low, with the measurements in the Log.
- Setup and teardown documented; nothing outside the repo changed without the operator's approval.

## Plan

Design (draft, pending operator review):
- **Mechanism: an operator-owned session environment file, read by the pair launcher.** `$XDG_CONFIG_HOME/pair/env` (default `~/.config/pair/env`), `KEY=VALUE` lines (comments and blanks allowed). The launcher applies it next to its existing `PAIR_*` exports (`launcher/createflow.go:745`), so every pair session inherits it: Couch slots (Couch to `pair resume` to launcher) and hand-started `pair` alike. Pair stays free of knowledge about volumes, Go, or Claude (agent-agnostic, no hardcoded paths); the operator's shell profile is untouched. `PAIR_*`/`COUCH_*` keys are refused, because the launcher owns those. A missing file is a no-op. A malformed line fails the launch loudly with file:line (ARCH-SECURE: parse to typed pairs at the boundary).
- Pure parser `ParseSessionEnv([]byte) ([]EnvPair, error)` with unit tests; a thin IO read in the launcher (ARCH-PURE). Couch needs no change, since `mergeChildEnvironment` passes inherited keys through.
- **Operator's file content** (outside the repo, approval required):
  `GOCACHE=/Volumes/AgentScratch/go-build`, `TMPDIR=/Volumes/AgentScratch/tmp`, `GOTMPDIR=/Volumes/AgentScratch/tmp`, `CLAUDE_CODE_TMPDIR=/Volumes/AgentScratch/tmp`. The last one should move Claude Code's sandbox temp area (today `/private/tmp/claude-501/...`); this needs live verification.
- **Claude sandbox:** sandboxed commands get Claude's own `TMPDIR`, and writes outside the allowlist are denied. That needs `sandbox.filesystem.allowWrite: ["/Volumes/AgentScratch"]` in `~/.claude/settings.json`, which the operator edits.
- **Lifecycle (ARCH-FUNERAL):** Go's cache trims itself (entries unused for 5 days). Leaked temp dirs have no sweep; the volume quota bounds them, and the documented teardown is deleting the volume. An optional periodic sweep stays a separate decision.
- Constraints (ARCH-CONSTRAINTS): the file is read once per launch (a few lines; startup path). ARCH-ORDER: holds no state between events, because the file is read once at launch. ARCH-MOCK: N/A, no external dependency.

Steps:
- [ ] Parser + tests; launcher wiring + test that the env reaches the child exports
- [ ] atlas: setup / teardown runbook (volume, `no_log`, `mdutil`, env file, sandbox allowWrite)
- [ ] (approved) create the volume, `no_log`, `mdutil -i off`, env file, sandbox allowWrite
- [ ] Relaunch slots; verify `go env GOCACHE` and `TMPDIR` inside a slot, sandboxed and not
- [ ] Measure fseventsd after a multi-slot workload; compare with the baseline in Log
- [ ] (approved) remove the old 92 GB cache

## Log

### 2026-10-12

- 13:30 baseline: fseventsd (restarted 11:45 by the operator) 11:36 CPU over 105 min, about 11% average. RSS 13 MB; top MEM 3.7 MB. The operator saw 10+ GB before killing it, so memory grows episodically and needs a time series: sampler every 5 min to the scratchpad `fseventsd-baseline.log`. GOCACHE `~/Library/Caches/go-build` = 92 GB (one cache key per worktree path, many slots). Container disk3 has 427 GB unallocated.
- Claude Code sandbox sets its own `TMPDIR` per command and ignores an inherited one; `CLAUDE_CODE_TMPDIR` relocates its temp base (docs unclear whether it moves the sandbox `TMPDIR`; verify live). `sandbox.filesystem.allowWrite` grants the volume.
