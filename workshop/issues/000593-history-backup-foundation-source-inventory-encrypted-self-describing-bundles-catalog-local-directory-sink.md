---
id: 000593
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '76220a7268e4debd60440367ef0ae0305541eccf' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T20:15:22-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# history-backup foundation: source inventory, encrypted self-describing bundles, catalog, local-directory sink

## Problem
Valuable history outside git has no common verified backup format or destination contract. Pair retention and continuous agent/operator-history backup need the same foundation before either may rely on archived bytes.

## Spec
Implement piece 1 of the operator-approved [history-backup design](/Users/xianxu/workspace/ops/teams/devinfra/projects/history-backup/design.md): one declared source inventory, versioned self-describing tar/zstd/GPG bundles, a local-directory sink with read-back checksum verification, and a private append-only JSONL catalog. Explicit manual input and a reusable Go API exercise the complete path. No automatic scanning, retention changes, deletion, SQLite snapshotting, cloud upload, restore/find command or scheduling in this issue: those are separate project pieces (#394/#592 and later work).

Files and manifests are encrypted; the approved plaintext label contains only kind, time range, counts/bytes, format/compression, recipient fingerprints and ciphertext checksum. The encryption path accepts public keys only and never asks for a password. Public-key availability is checked before reading payloads. Tests use an isolated temporary keyring, never the operator's secret key. Document the operator-owned key recovery/export checklist without performing it.

A successful catalog record means the exact stored bundle and label were read back and matched. A failed/canceled operation never deletes source files or declares verification. Interrupted publication can leave an uncataloged encrypted object; retry verifies and adopts the same manifest identity rather than requiring a durable job journal. A corrupt existing object or catalog is an error, not permission to overwrite it.

## Done when
- [ ] Checked-in inventory covers project source kinds with owner, root/pattern, mode, local-retention policy and sink rule; unknown kinds and malformed rules refuse.
- [ ] Explicit selected regular files produce bounded month/size-grouped bundles and encrypted MANIFEST.json with original relative paths, source identity, times, sizes and hashes.
- [ ] Local sink put/get/list/verify and append-only catalog work with repeat, partial publication, corruption and cancellation cases; no catalog success before both objects verify.
- [ ] Isolated real zstd/GPG round-trip recovers every fixture byte and manifest field; plaintext labels expose no file paths, and sources remain unchanged on success/failure.
- [ ] CLI, README, atlas and artifact classifications document the usable foundation, operating bounds and key-recovery responsibility; targeted and recorded suite pass; close is SHIP.

## Plan
See [durable plan](../plans/000593-history-backup-plan.md). One whole-issue review boundary; plain task checkboxes.

## Log
### 2026-10-15 — claimed and shaping
Claimed pair:1; start-plan refreshed the inherited ariadne tooling and succeeded on retry. Read operator-approved ops design and project, existing dispatcher/storagegc/artifactpath/durablefile/osfs seams. zstd, gpg and tar are installed. Ops MEMORY.md is absent; AGENTS.local.md read, no ops edits. Keep the foundation separate from retention authority: ARCH-DRY one format/inventory, ARCH-PURPOSE complete verified round-trip, ARCH-FUNERAL explicit artifact ownership without an extra job journal. Plan approval requested from hub before code.

Read-only bounded seam review confirmed there is no general backup inventory or streaming subprocess pipeline to reuse. Reuse durablefile for small metadata/sync and commitoutcome for uncertain publication; existing session catalogs are replacement/trimmed stores and cannot serve the append-only backup contract. Plan covers parent-symlink confinement, joined pipeline shutdown, no-replace objects and explicit torn-tail refusal.
