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

Configuration, not code (operator, 10-12). Each tool already has a per-user setting that isn't the shell profile:
- Go: `go env -w GOCACHE=… GOTMPDIR=…` (Go's own `~/Library/Application Support/go/env`), which covers every Go invocation: slots, hand-started pair, and the operator's own terminal.
- Claude Code: `~/.claude/settings.json`, with `env.CLAUDE_CODE_TMPDIR` (the sandbox replaces `TMPDIR` per command, so this is what moves test temp roots) plus `sandbox.filesystem.allowWrite` for the volume.
- Volume: case-insensitive APFS in container disk3, 200 GB quota, `.fseventsd/no_log`, `mdutil -i off`, Time Machine exclusion. Lifecycle (ARCH-FUNERAL): Go trims its cache itself; leaked temp dirs are bounded by the quota; teardown deletes the volume.
- Other agents (Codex etc.) need their own temp-dir equivalent; the runbook lists it.

Steps:
- [ ] (approved) create the volume; `no_log`; remount; `mdutil -i off`; tmutil exclusion; dirs
- [ ] (approved) `go env -w`; Claude settings `env` + `allowWrite`
- [ ] Verify in a fresh session: `go env GOCACHE`, sandboxed and unsandboxed `TMPDIR`, a test run writing to the volume
- [ ] atlas runbook: setup and teardown
- [ ] Measure fseventsd after a multi-slot workload, compared with the baseline
- [ ] (approved) remove the old 92 GB cache

## Revisions

- 2026-10-12 13:55: dropped the pair `~/.config/pair/env` launcher channel. Reason: each tool's own per-user config covers every session (including outside pair), and Claude's sandbox overrides an inherited `TMPDIR` anyway, so a pair channel adds a surface that does nothing extra (Simplicity First). Delta: no code; the repo change is the atlas runbook only.

## Log

### 2026-10-12

- 13:30 baseline: fseventsd (restarted 11:45 by the operator) 11:36 CPU over 105 min, about 11% average. RSS 13 MB; top MEM 3.7 MB. The operator saw 10+ GB before killing it, so memory grows episodically and needs a time series: sampler every 5 min to the scratchpad `fseventsd-baseline.log`. GOCACHE `~/Library/Caches/go-build` = 92 GB (one cache key per worktree path, many slots). Container disk3 has 427 GB unallocated.
- Claude Code sandbox sets its own `TMPDIR` per command and ignores an inherited one; `CLAUDE_CODE_TMPDIR` relocates its temp base (docs unclear whether it moves the sandbox `TMPDIR`; verify live). `sandbox.filesystem.allowWrite` grants the volume.
- Spotlight Privacy already excludes `~/Library/Caches` and `~/workspace` (operator screenshot), and fseventsd still ran hot. A Spotlight exclusion is a downstream filter: fseventsd still journals each event and delivers it to mds, which drops it. So the cost sits in fseventsd itself, and the volume's `no_log` is the part that matters. Turning off journaling on the whole Data volume was rejected: Time Machine (local destination) would deep-scan 17M files per backup, Spotlight would rescan after a reboot, and live delivery, the likely source of the memory growth, would continue anyway.
- 14:48: the operator created the volume: `diskutil apfs addVolume disk3 APFS AgentScratch -quota 200g`, `no_log`, remount, `mdutil -i off`, `tmutil addexclusion -v` (`-p` fails with EINVAL on a volume mount point; the runbook uses `-v`), and the `go-build`/`tmp` dirs. Verified: case-insensitive APFS in disk3, 200 GB quota with no reserve, Spotlight `Indexing and searching disabled`, TM `[Excluded]`, `.fseventsd` holds only `fseventsd-uuid` + `no_log` (no journal pages), dirs owned by the operator.
- 14:58 (operator approved "do it for me"): `go env -w GOCACHE=/Volumes/AgentScratch/go-build GOTMPDIR=/Volumes/AgentScratch/tmp`. Edited `~/.claude/settings.json` to add `env.CLAUDE_CODE_TMPDIR=/Volumes/AgentScratch/tmp` and append `/Volumes/AgentScratch` to `sandbox.filesystem.allowWrite`; backup in the session scratchpad. A sandboxed `go build` in the live session then wrote 104 MB to the volume cache, so the allowWrite change applied without a restart. `CLAUDE_CODE_TMPDIR` also took effect live: sandboxed commands in this session now get `TMPDIR=/Volumes/AgentScratch/tmp/claude-501`, so it does move the sandbox TMPDIR (the open docs question is answered).
