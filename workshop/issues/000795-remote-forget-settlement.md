---
id: 000795
status: open
deps: [pair#759]
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '71ec28253b1539763d4bfa620b4bd5aa68aed388' # card fields mirrored from issue-cards; edit via sdlc
---

# Settle remote submit evidence before exact-ID forget

## Problem

During pair#759 M3 integration, exact-ID forget settlement could not be proved from retained receipts: broker receipts expire, ledger trimming removes old uncertain rows, and audit retention removes old unmatched intents. An empty bounded scan is not proof an old potentially committed submit settled. TL split this mechanism after its third refinement (delivery d789d04d); M3 conservatively blocks forget for documents with remote or marked-local submits.

## Spec

Design an explicit, bounded settlement model using Couch's existing single-writer origin record. Proposed direction for full review: persist per-document pending-operation evidence before every remote or marked-local submit handoff, clear it only after confirmed completion, and pin unknown outcomes across broker/ledger/audit retention. No independent CLI origin writer, no new tombstone family, and no absence-based retirement inference. Forget must fence the exact registration ID, wait for owned operations and receipts, and never delete a replacement at a reused path. Offline forget refuses with start Couch and retry. Same-UID trust limit remains explicit.

Questions for the full plan: crash before/after intent, enqueue, delivery, receipt publication and evidence clearing; loss of HTTP/CLI reply; completion vs cancellation vs refusal; partial batch outcomes; startup join and receiver incarnation; safe operator reconciliation of permanently uncertain evidence; capacity bounds and overflow; unknown-field preservation; downgrade/mixed-version behavior; and migration of the conservative submit_seen block installed by pair#759. Historical gaps cannot be retroactively declared settled.

## Done when

- A full reviewed plan states one writer, publication ordering, all crash/recovery states and visible recovery actions.
- Uncertainty survives every underlying receipt/audit retention window and never silently clears or permits a second submit.
- Exact-ID forget/re-register and path reuse preserve provenance, including lost responses and restart.
- Capacity/excess and uncertainty reconciliation have explicit safe behavior, with stateful fault tests and counted bounded work.
- M3's conservative forget block is replaced only where new evidence proves settlement; legacy blocked records have an honest migration policy.

## Plan

Full flow; unclaimed follow-up. Start from landed pair#759. Evaluate the proposed origin-record model before code; do not resume incremental patches inside M3.

## Log

2026-10-21: Filed from pair:6 on TL instruction d789d04d. The simpler alternative (refuse all previously remote-submitted forget) is the safe shipping boundary for #759, while this issue owns the convenience/recovery design. The previously approved bounded scan (4096 directory entries/32 MiB, explicit forget only) cannot solve aged-out uncertainty and is not implemented as proof.
