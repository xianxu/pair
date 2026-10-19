---
id: 000677
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours: 2.52
collaboration-mode: collaboration
card_mirror: 'a0d3b9722c1e5f4ce24d7d660631ab1c4f0995e5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T16:54:38-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# diagnostic logs: untie from backup, enforce the per-file cap, keep ~7 days (48 GB, +1.5 GB/day)

## Problem

Pair's diagnostic logs (wrap-events, .pair-diagnostics segments, adapt, fullscreen) are ~48 GB and grow ~1.5 GB a day (survey 2026-10-18, ops qa 2026-10-18-16-history-backup-review.md). Their deletion is blocked by diagnosticlog.KeepDiagnosticsUntilBackup = true ("diagnostic eviction awaits history backup"), and that backup (pair#394) does not exist. Single files reach 4.9 GB despite the 64 MB per-generation cap. The operator agreed debugging logs can be trimmed without a backup.

## Spec

Untie diagnostic retention from history backup: diagnostics are not history. Enforce the existing per-file/generation cap so files rotate, and keep about 7 days of generations (configurable), deleting older ones without archiving. Find why a single wrap-events file grew to 4.9 GB past the cap and fix that rotation. Review whether the wrap-events detail level is needed at this volume (a measured bytes-per-turn figure). Existing 48 GB: the first sweep reclaims it; report how much.

## Done when

- Every newly admitted generation stays at or below64MiB; deterministic tests cross the boundary and observe bounded rotation. The first sweep evicts proven oversized legacy generations, including young ones; unknown/live holders stay blocked and reported.
- Diagnostic age/size eviction needs no backup or session-migration acknowledgment. Default7days, PAIR_DIAGNOSTIC_DAYS1..365; invalid configuration fails closed. Counted N/2N tests cover independent diagnostic-only/scheduled sweeps and preserve session/history data.
- Send the hub real preview file/byte categories and blocked reasons before live apply. Record the first approved diagnostic-only sweep's actual reclaimed bytes and remaining blockers; record owned-turn bytes and the detail-level decision.

## Plan

See [durable implementation plan](../plans/000677-diagnostic-log-retention-plan.md). Hub approved full flow; change-code is next. The approved live arrangement requires sending the real diagnostic-only preview categories and blocked reasons to the hub before apply, with no :0 staging.

- [ ] Replace the backup/count fence with one configurable age/oversize policy and deterministic regressions; preserve hard append bounds and deletion recovery.
- [ ] Separate diagnostic collection from session migration, including scheduled pages and a diagnostic-only CLI scope; count discovery/sweep work at N/2N.
- [ ] Measure controlled bytes per turn, document the detail-level decision, verify with owned fixtures and the approved real diagnostic-only sweep; report actual reclaimed bytes and blockers.
- [ ] Run targeted and recorded suites, review/CI/merge gates and report the verified landing.

## Log

### 2026-10-18

### 2026-10-18 — claim, root-cause evidence and proposed design

Claimed unowned #677 after #668 diagnostic landing/park. start-plan first refreshed ariadne ce8cbf95 and explicitly refused the old build; its requested rerun created this issue branch. No product code or live log mutation yet. The bounded read-only helper agent failed at model capacity; the main session completed the source audit instead.

Read-only metadata inventory under /Users/xianxu/.local/share/pair visited50,017 entries in0.695s, complete with no errors:760 diagnostic-path files,62,725,806,921 logical bytes and62,727,901,184 allocated bytes;33,212,814,029 bytes have filesystem mtime older than7days. This is a metadata estimate, not deletion authority or promised reclamation. It includes diagnostic sidecars; the actual policy uses certified last-write evidence and writer proof.

The largest current log is repos/e108517d46ab4575/wrap-events-pair.jsonl,4,933,163,904bytes, created2026-07-16, last changed2026-09-16, v1 state with no persisted limits. Shared strict byte-limit commit96b64e58e is dated2026-10-15, so this file predates enforcement. Another4,442,555,470-byte closed generation in repo990458a6956a804b was bornOct1 and last writtenOct15; metadata preserves the original oversized generation. ensureCurrent deliberately adopts preexisting bytes unchanged; policy_test.go/TestDiagnosticOversizedLegacyGenerationPreservedUntilBackup explicitly requires a4GiB sparse legacy segment to survive. Current Writer.Write performs whole-batch admission under the stable shared lock, refusing over-cap growth; hard-limit/oversized-legacy targeted tests pass (0.950s). No measured evidence says the current guarded append path writes a fresh generation past64MiB. The missing behavior is enforcing the new policy on inherited oversized files, not inventing a new append timeout/retry.

KeepDiagnosticsUntilBackup=true disables all deletion and count debt; setting it false alone would revive a3-generation policy that can discard recent high-volume data well before7days and can require complete64-generation inventories. The proposed age-only policy disables that dormant count authority. Separately, the live .retention/stores.json says migration_complete=false; gcruntime.Apply, diagnosticPages and Batch all gate diagnostics on session/Couch migration. A diagnostic-only path is needed to reclaim debug files without authorizing or initializing session-data retention.

Bounded8MiB tail sample of repo990458a6956a804b's current wrap-events:8,388,489bytes,39,163valid records,1,577,454 raw output bytes. master-chunk/stdout-queue/scrollback-write total30,762 records; stdout-batch-flush adds8,386. Those4labels contribute8,385,438bytes (>99.9% of the sample), about5.32 diagnostic bytes per raw-output byte. It spans2026-10-16T20:58:49 through2026-10-17T10:54:58 and contains no turn IDs. Six stdin reads are not six proven turns; no bytes-per-turn claim is inferred from them. The plan measures explicitly delimited owned turns through the production proxy before deciding whether detail must change.

Recommended tradeoff for hub review: expire proven diagnostic generations at7days by default; evict entire inherited oversized diagnostic generations through the existing deletion intent, even when young. This intentionally loses those legacy debug generations and avoids copying/re-writing tens of GiB or adding a compaction journal. Unknown/live legacy holders, identity changes, malformed metadata and cancellation retain data with explicit reasons. Newly admitted generations stay64MiB maximum; remaining unproven oversized legacy files are reported rather than silently claimed fixed. Alternatives and visible states are in the plan. This follows the operator's permission to trim debugging logs without backup; history/session data remains under its existing guards.

### 2026-10-18 — plan gate PQ-1

Full-flow change-code round1 accepted the diagnosis, scope and all architectural lenses but blocked on function-level-test-contract: compress procedural case inventories and name the selection/service entry points. Replaced those lists with strategy/guard tables covering ParseRetentionDays, DecideGeneration, limits/admission, proof/preview/replay, service constructors, PreviewDiagnostics/ApplyDiagnostics, diagnosticPages, Batch and runWithService. Counted N/2N acceptance remains. No product code edited; no live apply or estimate derived before plan-quality acceptance.


## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only. Derived after plan-quality CLEAN; the calibration source is marked stale by sdlc estimate-source, so the result is provisional.

Library availability: this extends the owned diagnosticlog journal, portable storage backend, Inspection seam, existing collector and Go flag parser; no greenfield stack or new external library is needed. Familiarity1.0 reflects that source audit and the existing patterns. Thorough-plan design discount0.2 applies to implementation components, not the issue authoring already performed; design buffer15%. Each impl below is exactly40% of the chosen v2/v2.1 primitive implementation hours, applied once.

Rows in order: issue/spec authoring (base impl0.2); policy (0.5); independent collector (0.5); CLI/config (0.4); count/fence consumer refactor (0.5); controlled turn measurement (0.4); owned/live verification (0.5); docs (0.2); one close review (0.5). Component design picks before the0.2 discount are0.3,0.3,0.2,0.5,0.2,0.2,0.2,0.1; issue authoring design0.5 is undiscounted. No API-discovery primitive is added for an existing file/process seam. Machine queue time is excluded from focused ship-time.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.50 impl=0.08
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.04 impl=0.16
item: cross-cutting-refactor design=0.10 impl=0.20
item: smaller-go-module design=0.04 impl=0.16
item: smaller-go-module design=0.04 impl=0.20
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.02 impl=0.20
design-buffer: 0.15
total: 2.52
```

Design subtotal0.90h ×1.15 + scaled implementation1.48h =2.515h, rounded2.52h; a provisional working range is about1.8–3.3h. This prices the accepted scope, not the earlier queue wait or a predetermined total.

## Revisions

### 2026-10-18 — approved diagnostic-only policy and live gate

Verified hub receipt5ebfef41-eb7d-4f3e-92f4-fff05f5b0a37 approves the committed plan and full flow, including whole eviction of young oversized legacy diagnostics and retaining/reporting unknown or live holders. Clarified Done-when to distinguish bounded newly admitted data from blocked inherited files; the absolute legacy-cap claim cannot override live-writer safety. Added the explicit pre-apply gate: run real diagnostic-only preview and send file/byte categories plus blocked reasons to the hub first. This is a reporting gate in an already-approved live arrangement, not a request to touch :0 or acknowledge session migration.
