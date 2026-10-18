---
id: 000656
status: working
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-18
estimate_hours:
severity: low
collaboration-mode: collaboration
card_mirror: 'ddca2a8172e2e1ef3acc4e79d579efae493fa574' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T05:12:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "2bcb021c", done: "7e9ac94d"}
---

# Title projection: agy database growth still stops the title for good

## Problem

The compact title projection pins the Agy SQLite database alongside its transcript. After admission, database growth fails the unchanged-artifact check and permanently stops the projection used by the Couch updater and standalone poller. Database growth must invalidate admission and trigger full authority revalidation, rather than authorize SQLite as append-only.

## Spec

Found in the pair#653 close review. The title projection (cmd/internal/sessioninventory/title_projection.go, checkArtifacts) requires an unchanged fingerprint for every non-transcript artifact. For agy that is its database, which grows during normal work, so that growth still stops the title for good, the same failure class #653 fixed for Claude/Codex transcripts. Classify same-identity growth of p.other as transient, the same way grewFrom does for the transcript, in both admission and tail.

### Proposed design (approval pending)

Keep this inside the quick-flow shell: one shared projection bugfix, expected fewer than 100 added production lines, tests excluded; no new public API, process, timer, artifact family or milestones. The single Couch updater landed in #649 and already consumes this projection.

Recommended: recognize same-identity Agy database growth at the artifact-check boundary, then reuse errTitleReadmit/reset to discard admission and return the last displayed facts with nil error. The next existing poll re-admits through the existing SQLite header/schema/identity and joined-transcript validator. #653 already handles admission failures that prove baseline-only growth; retain it and cover its Agy cases. Never broadly retry every tail ErrArtifactChanged.

Before a database-growth result can become transient, inspect all auxiliary artifacts and preserve transcript identity, shrink, same-size fingerprint and retained-boundary checks. An auxiliary mutation or a transcript rewrite must win over simultaneous database growth. Preserve the exact owner/agent/ordinal checks, including the post-tail owner check, before resetting. Context cancellation still rolls back the transaction through PollContext. No validated database content or new facts are committed from the interrupted pass.

Alternatives: accepting the database's new fingerprint without revalidation would bypass its authority contract; retrying all tail errors would hide mutation. A separate database-only validator would duplicate existing admission ownership and add a new state path. Reuse the existing admission/reset path (ARCH-DRY, ARCH-ORDER, ARCH-SECURE).

### Concepts and boundaries

| Concept / integration | Location | Change |
|---|---|---|
| Exact-invocation TitleProjection | cmd/internal/sessioninventory/title_projection.go | Bugfix its artifact-growth classification and transient tail reset; retain public Poll/PollContext contracts |
| File identity/growth predicate | same file: grewFrom, sameTitleIdentity, onlyGrew | Reuse pure predicates; no parallel growth definition |
| Native artifact/SQLite seam | existing Runtime/FakeRuntime and OSRuntime.QuerySQLiteContext | Reused stateful fake plus disposable actual SQLite fixtures; production-native files untouched |
| Couch updater consumer | cmd/internal/couchcmd/titles.go | Reused without a second recovery mechanism; add consumer regression in titles_test.go |

Actions: continue normal work; existing scheduled poll automatically retries; an invalid native authority requires repair and a fresh invocation as today.

| Internal state / event | Visible result | Recovery action | Possible loss |
|---|---|---|---|
| Admitted + unchanged database | Current display; no database query/history scan | Continue work | None |
| Admission/tail + same-identity database growth only | Keep last valid display; mark unadmitted | Next scheduled poll fully revalidates | Only uncommitted display progress; native data unchanged |
| Any + shrink/replacement/same-size mutation, transcript rewrite or lost owner | Existing terminal stop; no further title effects | Repair authority and fresh invocation | Display freshness until repaired; native data unchanged |
| Retry + invalid SQLite header/schema/identity | Terminal stop on stable validation | Repair authority and fresh invocation | Same display-only limitation |
| Any + canceled poll | No committed projection state change | Existing next pass retries | Uncommitted display progress only |

### Cost and coverage

Growth drivers are current targets K, transcript bytes T, tail bytes B and genuine database-growth invalidations G. Warm unchanged polls retain zero native body reads, zero SQLite queries and no new liveness queries. A growth-detection pass performs bounded metadata/retained-boundary checks; one stable re-admission uses the existing full validator, O(T), so recovery costs O(sum of re-admitted T), not a database suffix scan. No loop retries within a tick. Couch retains its existing 2-second whole-pass budget, fair continuation and lifetime ownership (ARCH-CONSTRAINTS, ARCH-FUNERAL).

Count workload at 8 and16 growth/recovery cycles: metadata/boundary checks and SQLite queries scale at most linearly; each stable cycle admits once, subsequent unchanged polls add zero native-history/SQLite work. Account for the existing stale-proof path's possible four SQLite queries rather than assuming every admission always needs two. Test cancellation without poisoning reuse, repeated growth, and a steady neighbor. No writes to native data outside disposable test fixtures.

## Done when

- A test where an agy database grows during admission and during tail fails on current main and passes with the fix; shrink, replacement and same-size mutation stay terminal (TestTitleProjectionAgyDatabaseMutationIsTerminal keeps passing).

- Both consumers recover through the shared projection: the Couch updater's same source resumes activity after real SQLite growth; the standalone Poll path remains valid. Agy count parsing is outside this issue; assert generic agy title/activity, not invented usage.
- Growth plus transcript/other-artifact mutation remains terminal; invalid SQLite identity/schema after growth fails stable re-admission; canceled recovery remains reusable.
- Warm zero-work and 8/16 counted growth/recovery checks pass, with required recorded suite and close review before landing.

## Plan

- [ ] After TL plan approval, run change-code and add deterministic stateful regressions in title_projection_test.go: admission (including post-validation growth), between-poll tail growth, mid-suffix growth, repeated growth, true mutations, mixed growth/mutation, invalid SQLite authority after growth, cancellation. Run the focused family and retain RED evidence.
- [ ] Implement only the shared classification/reset correction in title_projection.go. Reuse existing growth predicates, reset, validator and owner checks; run projection/provider/owner/cancellation families to GREEN.
- [ ] In couchcmd/titles_test.go, use the disposable OS fixture/shim pattern with actual SQLite (runtime_os_test.go adapter recipe), Agy ledger/pane/transcript roots and controlled database growth. Prove same source recovers and activity advances after admission/tail growth. Preserve warm unchanged command/IO behavior. No installed Agy/operator data or native Zellij session.
- [ ] Add counted 8/16 growth/recovery workload and mutation controls; run affected packages and targeted races. Update atlas/couch.md and add a mechanical lint rule only if investigation identifies a syntactically enforceable mistake class; a provider-specific semantic policy alone is not such a rule.
- [ ] Commit/push the tested source; run required recorded suite, retain all failures and fix reproducible branch defects. Prepare close evidence and PR; acceptance gates are close review, SHIP and sdlc merge, then report to hub.

Live verification proposal for TL: include #656 on the next operator restart alongside #649, observing Agy growth/recovery and title/fade behavior. The #649 no-staging waiver does not automatically cover this new issue; no :0 staging, installed-native probe or relaunch without the hub's direction.

## Log

### 2026-10-17

### 2026-10-18 — planning evidence

Verified dispatch deacf91a-32e1-4adc-af82-4e1b357bb7ee; card was open/unowned, claimed by pair:6 and start-plan completed on fresh main479b91376e5055f8106fe95b34bc64c71c6d2bc6 containing #649. Start-plan first refreshed the local SDLC dependency and refused before branching; the instructed repeat created this issue branch. No implementation or new test code has been written.

Existing TestTitleProjectionAgyDatabaseMutationIsTerminal passes on this main (0.159s) because its fixture appends bytes and expects a permanent error; that demonstrates the reported sticky growth path, not a true same-size mutation control. Read-only investigation confirms #653 already covers some baseline-only admission growth; tail errors still stop unconditionally, and checkArtifacts fingerprints all p.other artifacts exactly.

## Revisions

- 2026-10-18 — Proposed fixture clarification for TL approval: retain TestTitleProjectionAgyDatabaseMutationIsTerminal and its terminal assertion, but replace its AppendFile stimulus with true shrink/replacement/same-size mutation cases. The original append expectation contradicts the requested positive growth behavior. Cover append growth separately in positive admission/tail regressions. No expansion to arbitrary SQLite rewrites, count parsing, other providers or updater scheduling.
