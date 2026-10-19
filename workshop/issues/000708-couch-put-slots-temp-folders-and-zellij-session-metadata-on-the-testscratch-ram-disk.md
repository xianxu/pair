---
id: 000708
status: working
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '2174f741eed76a860461c787ecbb4dac6e401aa7' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T10:47:24-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# couch: put slots' temp folders and Zellij session metadata on the TestScratch RAM disk

## Problem

The operator's write-only fs_usage sample with the machine quiet (2026-10-19, ops side doc 2026-10-19-06) showed two steady
sources of main-disk file churn outside full suites:
- **Slot temp folders.** Codex and Qoder slots run with the system default TMPDIR (`/var/folders/.../T`, the main SSD), so
  every targeted `go test` they run creates and deletes its temp folders there (~3,700 operations in 30 s). Claude slots use
  `/Volumes/AgentScratch/tmp` (also the SSD).
- **Zellij session metadata.** Every live Zellij session (29 slot panes) rewrites
  `~/Library/Caches/org.Zellij-Contributors.Zellij/contract_version_1/session_info/<session>/session-metadata.kdl` every
  second (~1,840 operations in 30 s), regardless of `session_serialization false`; each write is also journaled by
  fseventsd. The folder also holds 73 session directories for 29 live sessions (stale ones never pruned).

The operator approved a 16 GB RAM disk at `/Volumes/TestScratch` (ops `teams/devinfra/tools/testscratch`, login item;
contents vanish at reboot). ariadne#468 moves sdlc suites, suite checkouts and review checkouts onto it. This issue moves
the slots' own churn. Operator: top priority, for speed first; SSD writes are one bottleneck.

## Spec

When `/Volumes/TestScratch` is mounted (a separate device from `/Volumes`) with room (same floor as ariadne#468, 2 GiB):
- **Slot temp:** Couch starts every slot's agent (Claude, Codex, Qoder) and its shells with `TMPDIR` (and `GOTMPDIR`) set to
  `/Volumes/TestScratch/tmp/<repo>-slot<N>/`, created 0700 at start. Otherwise today's TMPDIR, unchanged. Each slot's
  folder is removed when the slot is torn down; a stale-folder sweep at Couch start removes folders of slots that no longer
  exist. Share the "is the RAM disk usable" rule with ariadne#468 in spirit (same path, same floor); pair owns its copy.
- **Zellij session metadata:** Couch keeps `.../Zellij/contract_version_1/session_info` as a symlink to
  `/Volumes/TestScratch/zellij-session-info/` when the RAM disk is usable, created before the first Zellij session starts;
  when the RAM disk is absent, it restores a real directory. Never delete a live session's metadata (Zellij needs it for
  `zellij ls`/attach); migrate live sessions' folders when switching. Prune session folders with no live Zellij session.
- Neither change touches durable state: transcripts, pair's data dir and the Go build cache stay where they are.
- Respect ARCH-EVOLVE: no format change; the fallback is today's behavior.

**Approved Revision R1 (2026-10-19):** the hub replaced live metadata migration with reboot-only setup. With no native Zellij server running, rename the real session_info to session_info.disk (retain forever) and create the RAM symlink. If any server is live, change nothing and log once `Zellij metadata stays on disk until the next reboot`. If RAM is absent with no server and session_info is our link, replace it with an empty real directory; do not restore .disk. No atomic exchange, layout linking or metadata cache pruning. The original migration/pruning clauses above are superseded. The slot-temp half remains approved, including CLAUDE_CODE_TMPDIR and no-holder cleanup.

## Done when

- With TestScratch mounted, a new slot's agent and shells report TMPDIR under `/Volumes/TestScratch/tmp/<slot>/`, and a
  targeted `go test` in a Codex slot leaves nothing in `/var/folders/.../T` (live check).
- `session_info` resolves to the RAM disk; `zellij ls` and attach keep working across a Couch restart (live check).
- Without TestScratch, both behave exactly as today (fixture tests for both branches).
- A 30 s write-only fs_usage sample on a quiet fleet shows the `/var/folders` and Zellij lines gone or near zero (the
  operator runs it; command in the ops side doc).

- R1 metadata acceptance supersedes the Couch-restart migration check: isolated-HOME fixtures prove no-server RAM setup, live-server no-op and absent-RAM restoration from our link, including retry/failure preservation. The real metadata switch and write-only sample happen at the operator's next reboot. Existing panes need relaunch for their inherited temp settings; no live cache migration/pruning is required or allowed in this issue.

## Plan

- Durable design: [Couch RAM scratch plan](../plans/000708-couch-ram-scratch-plan.md).
- [ ] Implement the shared scratch policy, environment wiring and liveness-gated collection.
- [ ] Implement R1 startup-only metadata routing and the three isolated-HOME fixture branches.
- [ ] Complete counted budgets, documentation, recorded suite, close review and priority landing.

## Log

### 2026-10-19

Verified the hub's canonical assignment receipt dcc7fdf7-95c8-43b1-aad6-fcf150ad0053 (submitted to this pair:3 incarnation), ran tl-worker, claimed and entered start-plan. Start-plan refreshed the slot's ariadne substrate and created the branch from main at e51a0e6eda6910f209e68fde2df8f9ea494ec07f. Read the complete Spec and ops second-machine measurement document. No live session/cache/volume changes made.

Read-only exploration identified both production StartBlocked sites (launchTrackedThread and attachListedTerminalOnce), Claude's CLAUDE_CODE_TMPDIR override, and the need to retain detached servers/background jobs across attachment exit. Exact Zellij v0.45.1 source confirms unlocked pathname-based metadata writes and unconditional parsing of an existing resurrection layout before live attach. A rename-then-symlink gap or incoherent layout copy is unsafe. Proposed atomic exchange retains originals; existing layouts/external files keep their original inode via links while metadata moves to RAM. Native fixtures must use private HOME because XDG_CACHE_HOME alone does not isolate macOS ProjectDirs.

Read-only native cache sizing counted 73 session directories, 72 metadata files, 10 layouts, zero other files/symlinks and 308,437 bytes. This is not liveness or deletion evidence. ariadne#468's in-flight scratchdisk helper uses TestScratch, a separate-device check and 2 GiB floor; pair owns an analogous injected policy. Plan names the remaining approval boundaries: foreign/unproved cache rows remain protected, and the real fleet rollout belongs to the operator's restart. No code or tests run before plan approval.

Hub receipt a7a22cfc-b15d-478f-bac2-c9a1089f683f was verified submitted with exact body/sender/current recipient. It approved temp setup and directed R1: defer metadata switch until no Zellij server is running, keep a single permanent .disk backup, no live migration/exchange/layout links/cache pruning, and fixture-only metadata acceptance before the operator's next reboot. Appended the revised executable design and retained the original proposal as history. No production code has been written.

## Revisions

### 2026-10-19 10:56 PDT — R1, hub-approved simplification

Replace the initial live metadata migration/pruning contract with no-server startup setup and permanent session_info.disk retention. Scope now excludes atomic exchange, layout linking and cache pruning. Done when uses three isolated-HOME fixture branches and the operator's next reboot for real metadata routing. Temp environment, ownership and cleanup remain approved. Detailed final execution contract is Revision R1 of the durable plan.
