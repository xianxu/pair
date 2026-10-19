---
id: 000708
status: open
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '99e678f4f0869874b054ce7f27a0855dec1d6e80' # card fields mirrored from issue-cards; edit via sdlc
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

## Done when

- With TestScratch mounted, a new slot's agent and shells report TMPDIR under `/Volumes/TestScratch/tmp/<slot>/`, and a
  targeted `go test` in a Codex slot leaves nothing in `/var/folders/.../T` (live check).
- `session_info` resolves to the RAM disk; `zellij ls` and attach keep working across a Couch restart (live check).
- Without TestScratch, both behave exactly as today (fixture tests for both branches).
- A 30 s write-only fs_usage sample on a quiet fleet shows the `/var/folders` and Zellij lines gone or near zero (the
  operator runs it; command in the ops side doc).

## Plan

- [ ]

## Log

### 2026-10-19
