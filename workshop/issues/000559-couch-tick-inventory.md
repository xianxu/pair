---
id: 000559
status: open
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
card_mirror: '5e6409f45f76e7005461f5bdda48cc61c3916b1f' # card fields mirrored from issue-cards; edit via sdlc
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

## Plan

- [ ] Add regression fixtures and capture uncached read/allocation profile.
- [ ] Cache strict session-name evidence and share existing ledger parsing across long-lived collision queries.
- [ ] Reuse validated activity projection while its exact dependencies are unchanged; retain changed-input validation.
- [ ] Verify invalidation/races/scale, document cache ownership, run full suite, review, and coordinate acceptance/landing.

## Log

### 2026-10-14

Claimed as separate #559 because #157's complete forest CLI contract is materially broader. start-plan refreshed pair and Ariadne and required its normal retry; no branch/code existed before the successful retry. Read-only samples: /tmp/pair-hotloop-live.sample (3 seconds) and /tmp/pair559-live-5s.sample (5 seconds). ps observed pid 16699 at 95.3% CPU, cumulative CPU 47:48.82 at elapsed 2:00:33. The first sample catches Broker messageAuthority.current -> ListedPairSessionContext -> resolveScopedBindings -> ReadSessionNameIndex -> DecodeSessionNameIndex. The second also catches advanceActivity -> wireIdleFading -> QuerySessionContext -> ValidateTargetWork. Stack occurrence counts are not CPU percentages.

Important existing behavior: threadactivity already shares OwnerLaunchCache (#507), so adding another ledger-current cache would duplicate it. QueryResumeTarget reads the full owner ledger uncached, and QuerySession loads/validates content/catalog even after an unchanged launch-cache hit. Read-only subagent exploration confirmed these paths and warned that metadata-only resume identity is weaker than current scanner-authorized activity. No operator state, wrapper, or staged checkout changed.

## Revisions

### 2026-10-14 — name both release regression paths

TL relayed the operator's requirement: deterministic bytes-decoded invariants at 20 slots × 30 MB ledgers, covering messageAuthority/ListedPairSessionContext and idle activity QuerySessionContext/ValidateTargetWork separately. Plan now names TestMessageAuthorityUnchangedFilesDecodeZeroBytes and TestIdleActivityTickUnchangedFilesDecodeZeroBytes, with separate actual-read/decoder counters by source family, positive cold/mutated controls, and reusable counter snapshots for #204. Existing Done-when zero-work criteria are strengthened to that exact scale; no timing assertions.
