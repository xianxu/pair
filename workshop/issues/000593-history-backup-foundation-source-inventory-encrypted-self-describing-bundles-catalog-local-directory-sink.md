---
id: 000593
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: 'f215de8fdb01d0fe0eec871074b6d4f912e0788d' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-15
