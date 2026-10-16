---
id: 000593
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours: 7.2025
card_mirror: '555249fd1a3404ab20e2b7df3c7053b0a1e50d63' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T20:15:22-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# history-backup foundation: source inventory, encrypted self-describing bundles, catalog, local-directory sink

## Problem

Ops project history-backup (approved design: `ops/teams/devinfra/projects/history-backup/design.md`) needs its
foundation: a declared source inventory, a self-describing encrypted bundle format, a catalog, and a first sink.

## Spec

Design sections 2.1, 2.2, 2.3 and 2.5, plus the local-directory sink from 2.4.

- **Source inventory:** one checked-in config: kind, path pattern, mode (on-expiry or continuous), local retention,
  sink rule. Agent-agnostic, no hardcoded home paths (resolve from config and XDG/agent defaults).
- **Bundle writer:** tar (original relative paths + `MANIFEST.json`: format version, kind, host, root, time range,
  per-file path/size/mtime/sha256) → `zstd -19` → GPG public-key encryption to configured recipient fingerprints (never
  prompts; no private key needed to write). Plus a plaintext label JSON beside it (format version, kind, time range,
  file count, bytes, compression, recipient fingerprints, bundle sha256). Path
  `<kind>/<YYYY>/<MM>/<kind>-<from>-<to>-<id>.tar.zst.gpg`; bundles ~256 MB–1 GB, cut by time range.
- **Sink interface:** put, get, list, verify (compare the remote checksum with what was written). First sink: a local
  directory (any mounted path).
- **Catalog:** append-only JSONL of bundles (id, kind, time range, sink, bytes, sha256, verify time, file list),
  itself archived daily.
- **Operator checklist** in the docs (not automated): export the GPG secret key and passphrase to 1Password as the
  recovery copy.

## Done when

- Round trip in tests with a throwaway GPG key: files → bundle → local sink → verify → get → decrypt → identical bytes
  and manifest; a corrupted bundle fails verify; labels parse without the key.
- One live bundle of a small real source (ops QA files) written to a local directory and verified.

## Plan

See [durable plan](../plans/000593-history-backup-plan.md).

- [ ] Implement and qualify the foundation through one whole-issue review boundary.

## Log
### 2026-10-15 — claimed and shaping
Claimed pair:1; start-plan refreshed the inherited ariadne tooling and succeeded on retry. Read operator-approved ops design and project, existing dispatcher/storagegc/artifactpath/durablefile/osfs seams. zstd, gpg and tar are installed. The initial sdlc issue show printed headings only; the full original issue was subsequently read and preserved. Ops MEMORY.md is absent; AGENTS.local.md read, no ops edits. Keep the foundation separate from retention authority: ARCH-DRY one format/inventory, ARCH-PURPOSE complete verified round-trip, ARCH-FUNERAL explicit artifact ownership without an extra job journal. Plan approval requested from hub before code.

Read-only bounded seam review confirmed there is no general backup inventory or streaming subprocess pipeline to reuse. Reuse durablefile for small metadata/sync and commitoutcome for uncertain publication; existing session catalogs are replacement/trimmed stores and cannot serve the append-only backup contract. Plan covers parent-symlink confinement, joined pipeline shutdown, no-replace objects and explicit torn-tail refusal.

## Revisions

### 2026-10-15 — proposed foundation implementation scope

The durable plan expands the approved format into a reusable Go API and explicit `pair archive create --request FILE` route. Original Spec and Done-when remain binding, including the live small ops-QA bundle. Daily catalog scheduling belongs to #592; this foundation declares the catalog as a continuous source and does not schedule itself. Root defaults resolve from caller config/XDG/agent defaults without hardcoded personal home paths. The live smoke reads a small selected ops QA file set and writes only an encrypted bundle/label/catalog to a private temporary local sink; never prints source content or modifies ops. Proposed bounds and incomplete-publication behavior are in the plan for TL approval.

### 2026-10-15 — TL approval and catalog simplification

TL receipt4042bd3a approves dc721f5f with two superseding changes: under the catalog lock, repair an incomplete final JSONL line by truncating to the last complete record and syncing before append. Complete records are retained and validated; a complete record conflicting with the stored bundle refuses. Any bundle whose catalog line was torn is a verified uncataloged object, handled by the existing retry adoption. Roll a full64MiB shard automatically to the next numbered shard; no operator shard selection. Shards are one logical catalog and duplicate lookup spans them by streaming reads; no unbounded in-memory index. New-shard creation is synced and serialized by the same catalog lock. Tests cover tail truncation before retry, complete-line conflict, boundary rollover, crash/retry across rollover and preserved earlier shards. These rules supersede the earlier tail-refusal/full-shard-refusal text and tests; all other approved bounds/scope remain.

### 2026-10-15 — plan-quality refinements

Round1 requested durability reconfirmation on retry (PQ-1) and a named function/test-strategy mapping (PQ-2). Added unconditional file/directory sync barriers to read-back/reuse/adoption and catalog reopen/repair/rollover, with repeated-sync-failure regressions. This adds no persisted machinery. The plan's function table consolidates existing test cases into production seams and independent oracles.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; calibration is marked stale by sdlc, so provisional.

Rows in order: issue/spec; inventory/format; selected-file reader; local sink; catalog; codec integration; coordinator; CLI/dispatcher; docs; one whole-issue review; real GPG/zstd discovery plus live fixture. Greenfield design selects1h base ×0.2 thorough-spec discount; file reader also halves design for standard archive/tar and existing safe-open patterns. Codec API design2h ×0.2. Implementation values are v2 high-range ×0.4; novel-but-bounded encryption/pipeline work uses familiarity1.5. Issue/spec0.8h counts this issue’s design work; no second Method B charge. Thorough plan buffer15%; queues/idle waits excluded, actual executing suites/reviews included in qualification. No new library needed: standard tar/SHA/JSON, installed codecs, durablefile/commitoutcome.

```estimate
model: estimate-logic-v3.1
familiarity: 1.5
item: issue-spec design=0.8 impl=0.08
item: greenfield-go-module design=0.2 impl=0.32
item: greenfield-go-module design=0.1 impl=0.32
item: greenfield-go-module design=0.2 impl=0.32
item: greenfield-go-module design=0.2 impl=0.32
item: api-integration design=0.4 impl=0.6
item: greenfield-go-module design=0.2 impl=0.32
item: skill-or-dispatcher design=0.1 impl=0.2
item: atlas-docs design=0.05 impl=0.08
item: milestone-review design=0.1 impl=0.2
item: real-api-discovery design=0 impl=0.24
design-buffer: 0.15
total: 7.2025
```
