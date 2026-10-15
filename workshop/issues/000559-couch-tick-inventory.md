---
id: 000559
status: working
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours: 2.106
card_mirror: '87180bbb553fae5be0306e1b1cf1e32494bb0686' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T16:33:51-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Stop reparsing inventory on Couch ticks

## Problem

The operator's Couch supervisor pid 16699 (reported build fc145480, staged #528 M1) continuously consumes 40–100% CPU; messaging/socket replies miss deadlines under load. Repeated UI/activity and message-authority observations decode unchanged session indexes and inventory evidence. #157 concerns the full-inventory CLI, whereas this issue is the long-lived Couch hot loop. #508 owns oversized durable ledger rows, not this repeated consumption.

## Spec

- Unchanged warm Couch activity and session-name collision observations perform metadata checks, not index/ledger/native-content JSON decoding proportional to historical bytes.
- Reuse existing strict parsers, owner-launch cache, artifact fingerprints and scanner authority (ARCH-DRY). Do not replace scanner validation with a filename/native-ID guess.
- Keep current launch selection, previous-established resume fallback, cross-scope collision rules, and unknown/fail-closed behavior. Cache parsed durable evidence only; live server ownership and effect revalidation remain fresh.
- Invalidate on append, rewrite, truncation, replacement, disappearance/recreation, catalog/schema changes and any incomplete observation. A failed read never revives earlier success.
- Caches belong to long-lived callers, are bounded, retain no open descriptors, and disappear at caller exit. No durable schema change, new watcher, TTL authority, UI tick slowdown, or deadline increase.
- See durable plan workshop/plans/000559-couch-tick-inventory-plan.md. TL reviews before change-code.

## Done when

- Production-wired repeated collision/index observations read/decode unchanged files once; legacy/scoped precedence and ambiguity stay identical to uncached behavior.
- Production-wired repeated idle activity queries read/decode no unchanged ledger, catalog or transcript bodies after warmup, with memory allocations independent of historical file size.
- Every changed input triggers existing validation; tests reject stale authority after same-size rewrite, replacement, missing/corrupt inputs and concurrent change/cancellation.
- A synthetic 15–33 MiB fixture gives before/after CPU or allocation profiles plus deterministic byte/read-count regression; native growth and previous-launch tests remain correct.
- Targeted/race checks and full required suite pass; close review SHIP; rollout/live CPU and normal-delivery check coordinated with hub.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; calibration is flagged stale by sdlc, so provisional. Use v2.1 thorough-spec design ×0.2 and +15% design buffer, v3.1 implementation ×0.40, familiarity ×1.0. Standard-library synchronization and existing fingerprint/parser/catalog APIs avoid a novel stack; session-index cache design receives the library-availability half. No new external API discovery.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.10 impl=0.08
item: greenfield-go-module design=0.05 impl=0.32
item: greenfield-go-module design=0.10 impl=0.32
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.01 impl=0.08
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 2.106
```

Rows map to issue/spec, session-name cache, activity cache, ledger reuse, message invariant fixture, idle invariant fixture, atlas, one close review. Greenfield implementation uses upper v2 0.8h ×0.4; smaller modules use upper 0.5h ×0.4 for stateful/race coverage. Design subtotal 0.44h, buffered 0.506h; implementation 1.60h, total 2.106h. Prior high-estimate warnings noted: no extra generic risk multiplier or second implementation scale applied.

## Plan

- [x] Add regression fixtures and capture uncached read/allocation profile.
- [x] Cache strict session-name evidence and share existing ledger parsing across long-lived collision queries.
- [x] Reuse validated activity projection while its exact dependencies are unchanged; retain changed-input validation.
- [ ] Verify invalidation/races/scale, document cache ownership, run full suite, review, and coordinate acceptance/landing.

## Log

### 2026-10-14

Claimed as separate #559 because #157's complete forest CLI contract is materially broader. start-plan refreshed pair and Ariadne and required its normal retry; no branch/code existed before the successful retry. Read-only samples: /tmp/pair-hotloop-live.sample (3 seconds) and /tmp/pair559-live-5s.sample (5 seconds). ps observed pid 16699 at 95.3% CPU, cumulative CPU 47:48.82 at elapsed 2:00:33. The first sample catches Broker messageAuthority.current -> ListedPairSessionContext -> resolveScopedBindings -> ReadSessionNameIndex -> DecodeSessionNameIndex. The second also catches advanceActivity -> wireIdleFading -> QuerySessionContext -> ValidateTargetWork. Stack occurrence counts are not CPU percentages.

Important existing behavior: threadactivity already shares OwnerLaunchCache (#507), so adding another ledger-current cache would duplicate it. QueryResumeTarget reads the full owner ledger uncached, and QuerySession loads/validates content/catalog even after an unchanged launch-cache hit. Read-only subagent exploration confirmed these paths and warned that metadata-only resume identity is weaker than current scanner-authorized activity. No operator state, wrapper, or staged checkout changed.

## Revisions

### 2026-10-14 — name both release regression paths

TL relayed the operator's requirement: deterministic bytes-decoded invariants at 20 slots × 30 MB ledgers, covering messageAuthority/ListedPairSessionContext and idle activity QuerySessionContext/ValidateTargetWork separately. Plan now names TestMessageAuthorityUnchangedFilesDecodeZeroBytes and TestIdleActivityTickUnchangedFilesDecodeZeroBytes, with separate actual-read/decoder counters by source family, positive cold/mutated controls, and reusable counter snapshots for #204. Existing Done-when zero-work criteria are strengthened to that exact scale; no timing assertions.

Plan-quality PQ-1: added concrete cache read/query/transition/projection function names with one adversarial strategy and mechanical guard each. This refines the approved design, not scope.

Implementation checkpoint: change-code full flow accepted after plan-quality CLEAN (PQ-1 concrete function contract addressed) and estimate-quality INFO (verification/rollout included in fixture/review rows, activity dependency/publication design remains custom). Bounded subagents implemented strict index cache/checker wiring and shared owner-ledger parse/counters with red-green and targeted race checks. Activity cache implemented with current-launch scanner semantics, metadata dependency capture and per-key publication; separate read-only safety review found a disposable-catalog metadata failure regression, now pinned red then green by TestActivityCacheCatalogMetadataFailureUsesUncachedValidation. Cache falls back to original uncached validation there.

Counted fixture first green: 20 × 30,000,000-byte real streamed ledgers read/decoded 600,000,000 bytes cold, zero read/decode deltas on repeated warm passes. Message fixture cold positive indexes, zero warm bytes while process/thread guards execute. Counterfactual overlays disabling only the relevant cache hit fail both named regressions (activity repeated catalog read/decode; message repeated index read/decode). Integrated targeted activity race checks passed; full sessioninventory package passed during integration, final invariants/benchmark/source inventory validation in progress. No live supervisor mutation, stage, or restart.

Final named regressions passed in /tmp/pair559-final-invariants.log. Warm allocation comparison: 1 KB and 30 MB histories both 297 allocations/query. Twenty-slot benchmark (100 passes): enabled 609,366 B/op and 5,940 allocations/op versus cache-hit-disabled overlay 2,469,997 B/op and 25,047 allocations/op; enabled reads/decodes zero bytes versus 25,109 body/42,629 decoder bytes per pass disabled. Timing measured 6.09 ms vs19.50 ms on this host but is diagnostic only, never the release assertion. Integrated targeted race packages passed; corrected the couchcore test filter to its actual TestPairSessionNameAndScopedBindingsShareIndexCache name. Full source-inventory, launcher, inventory and threadactivity compatibility passed. PR refused uncommitted documentation and suite refused unpushed head; no suite ran in those refused attempts. Commit final docs before publishing, then record suite on pushed head.
