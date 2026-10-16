---
id: 000593
status: codecomplete
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours: 7.2025
card_mirror: 'c6f444b67d926a10d43c210c786bf7beab9d896f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T20:15:22-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 2.20
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

- [x] Implement and qualify the foundation through one whole-issue review boundary.

## Log
### 2026-10-15 — claimed and shaping
- 2026-10-15: closed — BR-1/2/3 addressed with RED/GREEN producer-path and capacity regressions. Final archive/fakes/CLI packages pass (28.878s/0.124s/0.453s); inventory/catalog race pass2.764s; CLI Couch-root regression pass. Real disposable-key full recovery and real ops QA checksum/source-preservation smoke passed. PR358 recorded suite queued on corrected head; no gates waived.; review verdict: SHIP
Claimed pair:1; start-plan refreshed the inherited ariadne tooling and succeeded on retry. Read operator-approved ops design and project, existing dispatcher/storagegc/artifactpath/durablefile/osfs seams. zstd, gpg and tar are installed. The initial sdlc issue show printed headings only; the full original issue was subsequently read and preserved. Ops MEMORY.md is absent; AGENTS.local.md read, no ops edits. Keep the foundation separate from retention authority: ARCH-DRY one format/inventory, ARCH-PURPOSE complete verified round-trip, ARCH-FUNERAL explicit artifact ownership without an extra job journal. Plan approval requested from hub before code.

Read-only bounded seam review confirmed there is no general backup inventory or streaming subprocess pipeline to reuse. Reuse durablefile for small metadata/sync and commitoutcome for uncertain publication; existing session catalogs are replacement/trimmed stores and cannot serve the append-only backup contract. Plan covers parent-symlink confinement, joined pipeline shutdown, no-replace objects and explicit torn-tail refusal.


### 2026-10-15 — implementation and targeted qualification

Implemented the inventory/pure format, descriptor-confined selection/tar streaming, joined zstd/GPG pipeline, local no-replace sink, repaired/rolling catalog, synchronous coordinator and thin CLI. Stateful doubles cover retained late publications, corruption and cancellation; live conformance uses a disposable key through the full Archiver→LocalSink→Get→independent decrypt/decompress path. Three empty/binary/Unicode fixtures recovered with identical hashes/manifest; corrupted ciphertext was rejected (6,144-byte tar, 1,048-byte ciphertext, 1.198s pipeline; 5.42s including key generation/cleanup). Initial artifact classification failed on the new files and passed after registering their ownership. CLI/dispatcher route tests passed. Catalog failed-open cleanup initially leaked its lock through a named-return defer; regression failed first and passed after capturing the owned catalog. Namespace regression likewise proved filesystem-root and staging/sink overlaps were admitted, then passed with pre-write refusal.

Targeted package and race runs passed before the final sink staging cleanup refinement. Final package run after that change passed (historyarchive6.763s, archivetest0.136s, archivecmd0.296s); final race run passed (18.208s/1.139s/1.208s). Final isolated live test passed in2.346s: full three-file pipeline1.078s, 6,144-byte tar→1,049-byte ciphertext, source hashes preserved and corruption rejected. Sink stale-partial cleanup, active writer protection and reserved-name tests each failed first, then focused race passed after serialized cleanup. The estimate-quality informational notes (catalog implementation risk and bundled real-tool discovery allowance) are acknowledged; keep the accepted estimate unchanged and measure actual work rather than revise it retrospectively.

Built CLI live smoke: one selected real ops QA document (1,150 bytes) encrypted with the approved public recipient to a 1,715-byte bundle in /private/tmp/pair593-live-68t2s1xa. Read-back plus independent SHA256 matched 9eb1c9a746cba16d427dc6c1d0549e81c349c97fb8129dd47b93226def8322b3; source hash unchanged; repeat invocation retained exactly one catalog record. /usr/bin/time -l reported 0.86s real, 0.02s user, 0.04s system and 86,851,584-byte maximum RSS (fixture measurement, not a general bound). No personal private-key operation, source output or ops edit. Encrypted fixture, private catalog and request are retained there for inspection until host temporary-file cleanup; this temporary smoke is not durable backup storage.

### 2026-10-15 — boundary review round 1

REWORK on BR-1 retained-storage-must-not-exhaust-maintenance and BR-2 inventory-must-cover-produced-artifacts. Confirmed both findings; appended correction Revision and prevention lesson. The old-head recorded suite was canceled while still queued (exit130, no suite ran) to avoid spending the machine slot on superseded code. BR-2 regression failed on both actual rolled shards, then passed with explicit numeric shard pattern; lock/staging negative cases remain excluded. BR-1 regression failed with10,001 retained final files, then passed with isolated staging; focused LocalSink/SafeFS race run passed6.199s. Review itself independently passed package/race/classification/live conformance and verified the preserved real-source smoke artifact.

### 2026-10-15 — boundary review round 2

Reviewer disposed BR-1/BR-2 with independently failing negative controls, and repeated package/race/live smoke verification. New BR-3 exposed a second instance of inventory-must-cover-produced-artifacts: slot-local Couch stores were omitted. Close remains working despite FIX-THEN-SHIP recommendation because the ledger blocks on BR-3. Sweep supported producer/backend locations and test producer-generated paths before another close. Canceled only this slot's second queued suite (exit130, still no suite executed) because the inventory correction changes the tested head. Built corrected publication path and isolated full codec round trip passed5.704s before this inventory revision.

BR-3 producer sweep: artifactpath Paths covers WrapEvents, Parked/parked scrollback raw-events-metadata, Scrollback raw-events-ANSI-viewport, SessionBindings, Ledger, AdaptLog and FullscreenDiagnostics in both legacy and scoped roots. Public ThreadStore Create/Archive and EnrollSlotRepository produce actual global and slot-local records, archive, archive-grace and manifest paths; declarations also cover preferences/journal. DirWriter produces webcast asset/stream files under configured Couch storage. Deliberate exclusions: openlocks/fullscreen control files, store locks/publication staging, saved-work checkout trees, current registry/descriptions, Chrome profiles and origin-privacy configuration. Patterns denote files or directory subtrees; no scanner. Tests first failed on missing legacy/global/local coverage, wrong global root and overbroad control inclusion, then passed. Inventory/catalog race pass2.764s. CLI COUCH_STORE_DIR propagation also failed first (used default home instead of configured store) then passed after adding the input to the CLI adapter.

### 2026-10-15 — SHIP and recorded suite

Round3 returned SHIP with zero open findings; close evidence ab5aed090e8fcef108a311ee45588e0751161f21. Required recorded suite passed on ec92c1fe36f427d3b6accd665a4b7147f79bf3ba: exit0 in13m11s, logSHA256 prefix e37eb0d26278, saved at the git common directory sdlc/local-checks/ec92c1fe36f427d3b6accd665a4b7147f79bf3ba/suite.log. Includes complete shell/editor checks and go test ./... (couchcore483.774s, historyarchive19.646s), no waivers or known failures. Subsequent changes are close evidence/docs and main’s unrelated issue-only commit. PR358 ready for the mandatory landing gate; TL-owned ops project will be reported to hub for ticking.

## Revisions

### 2026-10-15 — proposed foundation implementation scope

The durable plan expands the approved format into a reusable Go API and explicit `pair archive create --request FILE` route. Original Spec and Done-when remain binding, including the live small ops-QA bundle. Daily catalog scheduling belongs to #592; this foundation declares the catalog as a continuous source and does not schedule itself. Root defaults resolve from caller config/XDG/agent defaults without hardcoded personal home paths. The live smoke reads a small selected ops QA file set and writes only an encrypted bundle/label/catalog to a private temporary local sink; never prints source content or modifies ops. Proposed bounds and incomplete-publication behavior are in the plan for TL approval.

### 2026-10-15 — TL approval and catalog simplification

TL receipt4042bd3a approves dc721f5f with two superseding changes: under the catalog lock, repair an incomplete final JSONL line by truncating to the last complete record and syncing before append. Complete records are retained and validated; a complete record conflicting with the stored bundle refuses. Any bundle whose catalog line was torn is a verified uncataloged object, handled by the existing retry adoption. Roll a full64MiB shard automatically to the next numbered shard; no operator shard selection. Shards are one logical catalog and duplicate lookup spans them by streaming reads; no unbounded in-memory index. New-shard creation is synced and serialized by the same catalog lock. Tests cover tail truncation before retry, complete-line conflict, boundary rollover, crash/retry across rollover and preserved earlier shards. These rules supersede the earlier tail-refusal/full-shard-refusal text and tests; all other approved bounds/scope remain.

### 2026-10-15 — plan-quality refinements

Round1 requested durability reconfirmation on retry (PQ-1) and a named function/test-strategy mapping (PQ-2). Added unconditional file/directory sync barriers to read-back/reuse/adoption and catalog reopen/repair/rollover, with repeated-sync-failure regressions. This adds no persisted machinery. The plan's function table consolidates existing test cases into production seams and independent oracles.

### 2026-10-15 — boundary review growth corrections (BR-1/BR-2)

BR-1 identified that counting retained final objects against a maintenance scan budget would stop publication after 5,000 bundles per kind/month. Keep the stable parent lock, but move all temporary sink bytes into its reserved .archive-staging subdirectory and enumerate only that transient namespace. Retained objects never consume cleanup capacity; bounded List remains an explicit query bound. Cover more than10,000 retained entries plus stale-stage recovery and concurrent-writer protection. BR-2 identified missing catalog rollover files in the source declaration: declare the base JSONL and six-digit numbered shards, excluding locks/staging. Test against actual automatically rolled shard names (ARCH-PURPOSE/ARCH-FUNERAL), not just hand-written expected strings.

### 2026-10-15 — producer/backend inventory coverage (BR-3)

Inventory coverage is defined over every supported location for each declared producer, not just one typical path. Sweep artifactpath root-level legacy and repo-scoped Pair layouts; Couch global and slot-local ThreadStore layouts; COUCH_STORE_DIR/default Couch roots and webcast recordings; and catalog base/numbered shard outputs. Add explicit couch-slot-threads using a caller-supplied slot-store root, honor COUCH_STORE_DIR for global threads/webcast, include legacy Pair paths, and narrow globs that accidentally included transient control/lock files. Patterns select matching files or directory subtrees; this is a declaration contract, not a new collector/scanner. Coverage tests use production-generated artifact and Couch paths; lock/staging/control exclusions remain explicit. Agent transcripts, ops and SDLC directory selectors retain their declared root-relative subtree semantics. This corrects the approved inventory surface without adding scheduling, discovery or deletion.

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
