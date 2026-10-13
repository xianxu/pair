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
- [x] (approved) create the volume; `no_log`; remount; `mdutil -i off`; tmutil exclusion; dirs
- [x] (approved) `go env -w`; Claude settings `env` + `allowWrite`
- [x] Verify in a fresh session: `go env GOCACHE`, sandboxed and unsandboxed `TMPDIR`, a test run writing to the volume
- [x] atlas runbook: setup and teardown (`atlas/agent-scratch-volume.md`)
- [x] Measure fseventsd after a multi-slot workload, compared with the baseline
- [x] (approved) remove the old 92 GB cache

## Revisions

- 2026-10-12 13:55: dropped the pair `~/.config/pair/env` launcher channel. Reason: each tool's own per-user config covers every session (including outside pair), and Claude's sandbox overrides an inherited `TMPDIR` anyway, so a pair channel adds a surface that does nothing extra (Simplicity First). Delta: no code; the repo change is the atlas runbook only.

- 2026-10-12 20:25: outcome against Done-when. The volume, the slot configuration and the docs are met. "fseventsd's CPU and memory staying low" was **not demonstrated**: CPU did not drop measurably (11.2% → 12.7%, uncontrolled workload), and memory stayed low in both windows without the 10 GB episode recurring, so the cure is unconfirmed. Closing with that result, per the TL; the remaining load is likely `~/workspace` git/worktree churn, a separate question.

## Log

### 2026-10-12

- 13:30 baseline: fseventsd (restarted 11:45 by the operator) 11:36 CPU over 105 min, about 11% average. RSS 13 MB; top MEM 3.7 MB. The operator saw 10+ GB before killing it, so memory grows episodically and needs a time series: sampler every 5 min to the scratchpad `fseventsd-baseline.log`. GOCACHE `~/Library/Caches/go-build` = 92 GB (one cache key per worktree path, many slots). Container disk3 has 427 GB unallocated.
- Claude Code sandbox sets its own `TMPDIR` per command and ignores an inherited one; `CLAUDE_CODE_TMPDIR` relocates its temp base (docs unclear whether it moves the sandbox `TMPDIR`; verify live). `sandbox.filesystem.allowWrite` grants the volume.
- Spotlight Privacy already excludes `~/Library/Caches` and `~/workspace` (operator screenshot), and fseventsd still ran hot. A Spotlight exclusion is a downstream filter: fseventsd still journals each event and delivers it to mds, which drops it. So the cost sits in fseventsd itself, and the volume's `no_log` is the part that matters. Turning off journaling on the whole Data volume was rejected: Time Machine (local destination) would deep-scan 17M files per backup, Spotlight would rescan after a reboot, and live delivery, the likely source of the memory growth, would continue anyway.
- 14:48: the operator created the volume: `diskutil apfs addVolume disk3 APFS AgentScratch -quota 200g`, `no_log`, remount, `mdutil -i off`, `tmutil addexclusion -v` (`-p` fails with EINVAL on a volume mount point; the runbook uses `-v`), and the `go-build`/`tmp` dirs. Verified: case-insensitive APFS in disk3, 200 GB quota with no reserve, Spotlight `Indexing and searching disabled`, TM `[Excluded]`, `.fseventsd` holds only `fseventsd-uuid` + `no_log` (no journal pages), dirs owned by the operator.
- 14:58 (operator approved "do it for me"): `go env -w GOCACHE=/Volumes/AgentScratch/go-build GOTMPDIR=/Volumes/AgentScratch/tmp`. Edited `~/.claude/settings.json` to add `env.CLAUDE_CODE_TMPDIR=/Volumes/AgentScratch/tmp` and append `/Volumes/AgentScratch` to `sandbox.filesystem.allowWrite`; backup in the session scratchpad. A sandboxed `go build` in the live session then wrote 104 MB to the volume cache, so the allowWrite change applied without a restart. `CLAUDE_CODE_TMPDIR` also took effect live: sandboxed commands in this session now get `TMPDIR=/Volumes/AgentScratch/tmp/claude-501`, so it does move the sandbox TMPDIR (the open docs question is answered).
- 15:00: the operator ran `sudo ls .fseventsd`: still only `fseventsd-uuid` + `no_log` after the cache reached 436 MB, so `no_log` holds. Baseline series 13:32–14:52 (80 min): fseventsd CPU 11:52 → 20:50 (8:58), about 11% average; RSS 13.2 → 14.5 MB, top MEM about 3.8 MB flat. No memory growth episode during the baseline window.
- 16:21: the TL reported that the sampler had stopped. It had not: the previous session's loop (pid 61575) was still sampling as an orphan, with the last sample at 16:18:26, so the series has **no gap**. I replaced it with a bounded sampler (48 samples × 5 min, ending about 20:20) that appends to the same series, now copied to `/Volumes/AgentScratch/fseventsd-496.log` because the old session scratchpad is gone.
- Interim after the switch, 14:58 to 16:18 (80 min): fseventsd CPU 21:30 → 27:50 (6:20), **7.9%** against the 11.2% baseline (13:32 to 14:52). RSS 14.6 → 14.9 MB, top MEM about 3.8 MB flat. The volume cache grew to 4.5 GB, so slots are building there. The workload is not controlled, so this is indicative only, not yet the Done-when evidence.
- 17:25 TL lead (an ariadne slot: `mktemp` failed in the sandbox, an empty dir var sent `cd` to HOME, and it wrote `~/Makefile`). Measured in a sandboxed shell: writes to `/Volumes/AgentScratch/tmp` and `$TMPDIR` succeed. macOS `mktemp` **without a template** (`mktemp`, `mktemp -d`, `-t x`) ignores `$TMPDIR` for every value tried (the volume, `/private/tmp/claude-501`, the repo) and uses the Darwin per-user dir `/var/folders/…/T`, which the sandbox denies. Only `mktemp -d "${TMPDIR:-/tmp}/x.XXXXXX"` works. **This predates #496**: the same `mkdtemp failed on /var/folders` appears in transcripts from 2026-09-11 onward (ariadne, tools, pair, brain, parley, slots). It is not caused by the volume or `CLAUDE_CODE_TMPDIR`. Allowing `/var/folders/…/T` in the sandbox would move churn back onto the journaled Data volume, so the fix belongs to the scripts (template + `|| exit 1` on mktemp) and stays out of #496 scope.
- 19:55 after-measurement (the series runs until 20:20; this covers 13:32 to 19:54). fseventsd CPU per ~30 min window: before the switch 11.3 / 10.7 / 10.5%; after it 9.2 / 7.4 / 10.5 / 14.0 / 15.5 / 15.5 / 17.0 / 10.3 / 7.0 / 2.1%. Overall: 11.2% before (80 min) vs 11.1% after (14:58 to 19:54, 296 min). **No measurable CPU reduction**: the rate tracks the fleet workload (2 to 17%) and the workload was not controlled. Memory stayed flat at 15.0 to 16.0 MB RSS and about 4 MB top MEM in both windows; the 10+ GB episode did not recur, so the cure is unconfirmed rather than shown. The volume cache took 15 GB of build churn off the Data volume, and its `.fseventsd` stayed journal-free (checked at 436 MB). Old cache: 97 GB, last directory change 14:55 (before the switch); no running process carries `GOCACHE=` in its environment; Go's env file points at the volume. The likely remaining fseventsd load is other Data-volume churn (worktrees and git under `~/workspace`, which is journaled even though Spotlight-excluded); confirming that needs root `fs_usage`.
- 20:19 series complete (78 samples, 13:32 to 20:19). The final after-window, 14:58 to 20:19 (321 min), averaged **12.7%** CPU against the 11.2% baseline; the last 15 min ran hot (19.9%, 49.3%, 70.8% per 5 min) under fleet load. RSS peaked at 16.5 MB. Conclusion unchanged: no CPU reduction is measurable under an uncontrolled workload, and memory never grew in either window.
- ~20:10: the operator approved, and the TL ran the old-cache removal (`go clean -cache` plus `-fuzzcache` on `~/Library/Caches/go-build`): 97 GB freed, about 40K left (README/trim files).
