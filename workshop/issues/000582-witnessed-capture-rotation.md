---
id: 000582
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: 'bbbd71074ea466c7d1c63d10ae3b5660fe42d64a' # card fields mirrored from issue-cards; edit via sdlc
---

# Rotate witnessed terminal captures with faithful VT checkpoints

## Problem

Split from #579 by the owning hub on 2026-10-15. Raw terminal captures and their event sidecars grow without size-based rotation. They cannot safely use independent log rotation: readers rely on producer witnesses, exact file identities, cumulative offsets and faithful terminal replay. This issue is deliberately unassigned; #579 handles diagnostic write amplification and diagnostic bounds.

## Spec

Design and implement bounded size-based rotation/retention of a witnessed raw+events capture as one generation, through certified segments and full VT semantic checkpoints (or an equivalently faithful representation). Migrate park, recovery, peek and render together. No copy-truncate, arbitrary suffix eviction, screen-only ANSI seed or independent raw/events rename.

Consumer constraints established during #579:

- Producer: cmd/internal/wrapcmd/capture_witness.go openWitnessedCapture invalidates old events before raw truncation. PublishCaptureWitness writes the first sidecar record and syncs it before child output. cmd/internal/artifactpath/capture_witness.go binds scope/tag/agent/launch ordinal/nonce, producer OpenedAt epoch and both device/inode IDs. Preserve that authority when adding segments; rotation must not fabricate a later native launch timestamp.
- cmd/internal/wrapcmd/wrap.go handleChunk writes original PTY bytes; logScrollbackEvent resize/time offsets count cumulative raw bytes. Sidecar offset rebasing must agree with every renderer; preserve minute time/day and prompt-anchor semantics.
- cmd/internal/launcher/osruntime.go ParkScrollbackContext validates witness and current IDs, preserves exactly one pair today, syncs destination before source removal, and publishes exact CaptureMetadata. A new representation must atomically preserve the selected whole generation. Parked archives selected by recovery/previous peek must be outside the live writer eviction ring and remain protected by retention/reader ownership.
- cmd/internal/couchcore/recovery_evidence_os.go snapshots raw/events inode+size+mtime, validates witness/archive metadata, renders then checks unchanged. Preservation hashes the pair, parks it, verifies archived hashes and records exact coordinates. Update this boundary rather than weakening proof or falling back to another capture on uncertainty.
- cmd/internal/couchcore/peek.go PeekThreadSelection prefers live tail only for ordinary peek. Every --previous and --all requires the exact recording. peek_selection.go selects saved token/capture and refuses unavailable previous evidence rather than falling back to current. couchcmd/run.go SlotCapture uses scrollbackcmd RenderOwnedCaptureLines. peek_retained_os.go ReadRetainedTerminal validates exact CaptureMetadata/identities before and after rendering. Preserve that identity contract across rotation; explicitly define completeness as complete retained history when eviction has occurred.
- cmd/internal/scrollbackcmd/bounded.go enforces 64MiB raw, 8MiB events and 100k rendered rows. Current readers replay from the beginning; discarding a prefix silently corrupts VT semantics. New segment/checkpoint readers must be bounded too.
- cmd/internal/opener/run.go RunScrollback reads current capture; RunChangelog and the changelog distiller pass exact PCL_RAW/PCL_EVENTS into replay and retain timestamp/prompt-anchor meaning. Migrate these paths and viewer/renderer storagegc leases, not just Couch peek.
- artifactpath/manifest.go classifies current scrollback as SessionRetention and parked-scrollback as CaptureRetention; storagegc/collector.go groups immutable parked members by exact raw identity and independent age/reader leases. Arbitrary .raw.1 names would be unclassified and invisible. Register all new files and define cleanup/pinning before introducing the family.

No existing faithful checkpoint API was found. terminalModel.Snapshot exports visible cells/cursor/alt-screen/frame completeness only. It omits hidden main/alt screen, bounded history row metadata, SGR pen, terminal modes, scroll margins, saved cursor, charset/tabstops, pending-wrap and split UTF8/CSI/OSC/grapheme/parser state. vt Render/String and terminal RenderWithHistory serialize a display projection, not a complete state from which arbitrary future raw input can be interpreted. Do not treat them as a checkpoint. A full export/restore API must validate persisted input and limits, preserve both screens and parser state (or prove rotation at a safe parser boundary), and have replay-equivalence tests.

## Done when

- Raw/events rotate by size with explicit count/age and byte bounds, retaining useful recent history and reporting evicted-history completeness honestly.
- Replay before/after rotation matches uninterrupted replay for main/alt switches, resize, history/reflow, SGR/charset/modes/margins/saved cursor, wrap, and split parser/UTF8/grapheme sequences.
- Park, crash recovery, current/previous peek (including --all), scrollback and changelog use the shared certified representation and keep exact selected-capture identity.
- Deterministic interruption/replacement/reader-race tests cover checkpoint publication, segment handoff, eviction and archive preservation. Unknown/truncated/older metadata refuses or visibly degrades without fabricating evidence.
- Storage artifact inventory, leases, cleanup, bounded readers and atlas/README are updated; a live recorded rotation smoke proves recovery and previous peek on the retained generation.

## Plan

Plan from main after claim/start-plan. Resolve complete checkpoint schema and all reader migrations before implementation. This is not part of #579's implementation branch.

## Log

### 2026-10-15

Filed unassigned at hub request (verified receipt 8cb950d6-7819-48dd-9813-e10321f20ad8). Findings above came from read-only consumer exploration during #579; no raw rotation code has been written.

## Revisions

### 2026-10-15T23:00:00Z — distinguish per-thread diagnostics from per-wrapper capture

Hub observation during #579: wrap-events-<tag>.jsonl is keyed by the thread tag and survives relaunches (pair-8: born October1, now4.1GB); scrollback .raw restarts per wrapper and old raw/events pairs are parked. There are1,242 parked files totaling5.3GB since September23. The wrap-events diagnostic JSONL is distinct from the witnessed raw capture's offset sidecar; #579 bounds the former and verifies the existing parked-capture expiry path. This issue must preserve that distinction when designing capture size rotation, and account for accumulation across repeated wrapper launches. Existing parked captures expire through the independent CaptureRetentionPeriod constant (currently365days), with exact active readers/handoffs protected; diagnostic count eviction must never target recovery captures.
