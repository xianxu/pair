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

### Approved design

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

- [x] After TL plan approval, run change-code and add deterministic stateful regressions in title_projection_test.go: admission (including post-validation growth), between-poll tail growth, mid-suffix growth, repeated growth, true mutations, mixed growth/mutation, invalid SQLite authority after growth, cancellation. Run the focused family and retain RED evidence.
- [x] Implement only the shared classification/reset correction in title_projection.go. Reuse existing growth predicates, reset, validator and owner checks; run projection/provider/owner/cancellation families to GREEN.
- [x] In couchcmd/titles_test.go, use the disposable OS fixture/shim pattern with actual SQLite (runtime_os_test.go adapter recipe), Agy ledger/pane/transcript roots and controlled database growth. Prove same source recovers and activity advances after admission/tail growth. Preserve warm unchanged command/IO behavior. No installed Agy/operator data or native Zellij session.
- [x] Add counted 8/16 growth/recovery workload and mutation controls; run affected packages and targeted races. Update atlas/couch.md and add a mechanical lint rule only if investigation identifies a syntactically enforceable mistake class; a provider-specific semantic policy alone is not such a rule.
- [x] Commit/push the tested source; run required recorded suite, retain all failures and fix reproducible branch defects. Prepare close evidence and PR; acceptance gates are close review, SHIP and sdlc merge, then report to hub.

Live verification proposal for TL: include #656 on the next operator restart alongside #649, observing Agy growth/recovery and title/fade behavior. The #649 no-staging waiver does not automatically cover this new issue; no :0 staging, installed-native probe or relaunch without the hub's direction.

## Log

### 2026-10-17

### 2026-10-18 — planning evidence

Verified dispatch deacf91a-32e1-4adc-af82-4e1b357bb7ee; card was open/unowned, claimed by pair:6 and start-plan completed on fresh main479b91376e5055f8106fe95b34bc64c71c6d2bc6 containing #649. Start-plan first refreshed the local SDLC dependency and refused before branching; the instructed repeat created this issue branch. No implementation or new test code has been written.

Existing TestTitleProjectionAgyDatabaseMutationIsTerminal passes on this main (0.159s) because its fixture appends bytes and expects a permanent error; that demonstrates the reported sticky growth path, not a true same-size mutation control. Read-only investigation confirms #653 already covers some baseline-only admission growth; tail errors still stop unconditionally, and checkArtifacts fingerprints all p.other artifacts exactly.

## Revisions

- 2026-10-18 — Proposed fixture clarification for TL approval: retain TestTitleProjectionAgyDatabaseMutationIsTerminal and its terminal assertion, but replace its AppendFile stimulus with true shrink/replacement/same-size mutation cases. The original append expectation contradicts the requested positive growth behavior. Cover append growth separately in positive admission/tail regressions. No expansion to arbitrary SQLite rewrites, count parsing, other providers or updater scheduling.

### 2026-10-18 — approved implementation

Hub approval 2b7c3766-64cb-49fb-bd1d-c1342a815fba verified submitted: narrow growth readmission, true terminal mutation stimuli under the same test name, counted cycles. Live check explicitly shares #649 next operator restart; no separate stage. change-code passed inferred quick flow, accepted design commit fd6b137be261433b04b0c0b5d9520ea82fcc9d60. Stateful projection and disposable SQLite/updater regressions are being written before production changes. Initial growth recovery test fails on current production with ErrArtifactChanged and lost Activity; admission already covered by #653 remains a positive control.

### 2026-10-18 — implementation and targeted verification

Shared projection now classifies same-identity Agy database growth at checkArtifacts, checks all auxiliary artifacts and retained transcript boundary before allowing readmission, preserves the post-tail owner fence, and resets to the last displayed facts. The next poll fully revalidates SQLite authority. No consumer recovery path or API added (ARCH-DRY). Updated the architecture/Couch maps. This is provider-specific semantic evidence classification, not an enforceable syntactic misuse; no lint rule added.

RED retained: TestTitleProjectionAgyDatabaseGrowthRecovers failed with sticky ErrArtifactChanged and lost Activity before the fix. The disposable actual-SQLite Couch test's between-poll and mid-suffix cases also failed before the fix; admission was the existing #653 positive control. Existing terminal test keeps its name/assertion, with true shrink/replacement/same-size/birth/generation/missing stimuli instead of append growth, per approval.

GREEN: go test ./cmd/internal/sessioninventory ./cmd/internal/titlepoller ./cmd/internal/couchcmd ./cmd/internal/couchtty -count=1 (12.898s/1.802s/80.942s/43.384s). Targeted race: go test -race ./cmd/internal/sessioninventory ./cmd/internal/couchcmd ./cmd/internal/couchtty -run 'Test.*(TitleProjection|CurrentTitle|CouchTitles|TitleActivity)' -count=1 (4.767s/16.362s/1.497s). Focused Agy projection/updater matrix also passed. Covers growth during admission/between polls/mid-suffix, actual SQLite authority revalidation, mutation/owner/cancellation controls, same source recovery and steady neighbor. Disposable fixtures only; no native session or operator data touched.

Counted workload: 8 growth/recovery cycles used native ReadAt32, SQLite16, Stat104 and ListFiles32; 16 cycles used64/32/208/64 respectively (exact doubling). Growth-detection ticks query SQLite zero times; stable proofless readmission twice. Three warm polls per cycle add zero native body reads, SQLite queries or listings. Real-SQLite updater recovered with source opened once; ten unchanged polls added zero native read/decode/SQLite work and zero rename commands. git diff --check passed. Required recorded suite and close review pending.

### 2026-10-18 — recorded full suite

Required sdlc-local/suite passed on pushed source e073de64ee6839c1cb6a1fb7f173d20c5cf43d81: make PAIR_GO_TRIMPATH=-trimpath test, exit0 in12m27s, log sha256 prefix8fbe9b2341f2. Queue wait1032s; one attempt, no failures/retries. Full Go, shell/editor and diagnostic controls passed. Log: /Users/xianxu/workspace/pair/.git/sdlc/local-checks/e073de64ee6839c1cb6a1fb7f173d20c5cf43d81/suite.log. PR427 opened; server checks green. No code changes after the tested source; closing this branch without manually chasing main. Live check remains explicitly deferred by hub approval to the next operator restart alongside #649.

### 2026-10-18 — BR-1 mutation-precedence revision

Close round1 returned REWORK, Critical BR-1 (mutation-precedence-before-readmission): growth returned before the existing paired fingerprint checks, so a same-size rewrite after append during boundary/suffix reads could be hidden; the added recovery boundary read also lacked post-read metadata fencing for transcript/database replacement. This is the existing terminal-mutation contract, not expanded recovery scope. Add four deterministic interleavings plus no-growth controls. Carry the immediately preceding transcript observation into the artifact check before growth classification, then fence the recovery boundary read with fresh transcript and every auxiliary artifact snapshot. A terminal change always wins before errTitleReadmit. Keep stable growth recovery, owner and cancellation contracts and counted work. The source suite passed before this finding; code correction requires a fresh suite and close review.

### 2026-10-18 — BR-1 correction verification

Deterministic RED captured for TestTitleProjectionAgyGrowthCannotHideRewriteDuringTailRead boundary/database-growth and suffix/database-growth (same-size rewrite outside retained boundary after append); no-growth controls passed. TestTitleProjectionAgyRecoveryBoundaryRejectsArtifactReplacement transcript/database also failed before correction. Corrected by moving paired fingerprint comparisons into checkArtifacts before transient classification and bracketing the added recovery boundary read with fresh metadata snapshots of all dependent artifacts. No new recovery state/API. The same matrix now passes, expanded with shrink/same-size/birth/generation/missing cases for both artifacts and positive extra-growth cases that still re-admit through SQLite. Lesson recorded for the mutation-precedence class (ARCH-PURPOSE).

Fresh GREEN: all four affected packages passed (sessioninventory11.589s, titlepoller1.709s, couchcmd82.186s, couchtty43.777s); targeted race checks passed (inventory3.499s, couchcmd13.879s, couchtty1.520s). Counted 8/16 cycles now read32/64, SQLite16/32, Stat120/240, listings32/64: the post-read fence adds2 stats per growth cycle, still exact doubling and zero warm native-history/SQLite/listing work. git diff --check passed. Production patch66 added lines remains inside the shell. Fresh recorded full suite and repeat close review pending.
