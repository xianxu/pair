# Native build upgrade fixture implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Execute this tightly coupled harness change in the current session; SDLC owns the fresh boundary review.

**Goal:** Prove a candidate can adopt and use genuine warm sessions created by an older main build, and capture the 10-24 failed-stage incompatibility.

**Architecture:** Keep one existing testcouch.Fleet lifetime and one isolated namespace. Build both complete binary sets with BuildOwned, start old actors, detach the old console, replace its private runtime executables, then start the candidate against the retained sessions. Public candidate commands and independent old identity/socket evidence form the oracle.

**Tech Stack:** Go, existing native testcouch owner and fixture personalities, real isolated Zellij/nvim, opt-in cmd/couch acceptance test.

## Alternatives and decision

1. Extend the existing Fleet with a detached-only runtime replacement (chosen). Preserves all roots, cleanup authority and real old processes; one shared fixture for #866 and future stage checks (ARCH-DRY).
2. Create a second Fleet over the first Fleet's roots. Rejected: two owners could race cleanup and allocation assumes fresh roots.
3. Manufacture old socket state or delete candidate readiness receipts. Rejected: does not prove an actual version upgrade or the old-writer compatibility defect (ARCH-PURPOSE).

## Core concepts

| Pure entity | Lives in | Status |
|---|---|---|
| compareUpgradeCohort | cmd/couch/upgrade_acceptance_test.go | new: compare slot, complete actor binding and fixture PID/generation/native conversation identity before/after; no IO |
| OwnedRun / RunEvent | cmd/internal/testcouch/ownership.go | reused: partial replacement poisons new admissions through existing unknown outcome; cleanup remains authorized |

| Integration | Lives in | Status | Wraps |
|---|---|---|---|
| Fleet.ReplaceBuild | cmd/internal/testcouch/upgrade.go | new | existing runtime executable copier, detached console, private bin and explicit environment |
| Two-build native qualification | cmd/couch/upgrade_acceptance_test.go | new | BuildOwned, NewOwned, public candidate startup/list/resume/attach, existing fixture mailbox and native terminal queries |
| upgradeMarkerCount | cmd/couch/upgrade_acceptance_test.go | new | read-only diagnostic count of actual writer receipts; no absence requirement or receipt mutation |
| Public document registration | cmd/internal/testcouchbridge/native.go:NativeOwner.Readiness pattern | reused | ordinary parent and side document registered by the old executable, read by candidate |

The cohort comparison is unit-tested without IO. Existing fixture personalities are the stateful agent double; the native acceptance is the conformance check against actual Zellij. Portable replacement tests use a real temporary private filesystem and executable files, rather than mock call counts. No new process launcher or cleanup authority.

## Scope and operating bounds

One explicit hub and one worker per personality (Claude then Codex), sequential; no model/network calls. Build each coupled pair/couch/helper set once from explicit absolute source checkouts; fixtures remain the baseline version. Record source commits, dirty state, executable hashes and Go build identities before execution; refuse a dirty baseline or source/binary mismatch. Failed stage is93a57a380d290105b497dfd620ffb9e2e5a15528; baseline38fe8ea272d960705134e21ad3fbf61d26a4bb00.

Existing build deadline180s per set; provisioning240s per case; runtime180s per case; independent cleanup30s. Overall targeted Go timeout20m. Use existing scratch preference and socket-path bounds; do not change production budgets. At most one native case is active. File copies retain existing512MiB per executable bound. Evidence is a small bounded summary plus existing failure/startup logs; successful scratch cleanup removes the entire exact owned root. Retain a root only on uncertain cleanup, report it and stop. CPU spinners, load stress, scaling and the live fleet are outside scope (ARCH-CONSTRAINTS, ARCH-FUNERAL).

## Ordering, ownership and failure handling

Actions: run one qualification; cancel; inspect retained evidence after an uncertain outcome. Retry only as a new fresh fleet after positive cleanup, never over an uncertain root.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Old actors ready, old console alive | baseline ready | detach through existing owner | none |
| Console joined, no admitted command, all staged files validated | ready to replace | replace under Fleet mutex | none |
| Copies fail before any rename | replacement refused | cleanup old fleet | temporary staged files |
| One rename succeeded, later rename/cancellation fails | partial replacement, new admissions refused | cleanup through retained Fleet owner | candidate did not start; no rollback claim |
| Candidate startup/list/resume fails | compatibility failure with exact phase | capture evidence, cleanup | qualification fails; old sessions remain until owned cleanup |
| Cleanup cannot prove absence | retained-root uncertainty | inspect and report to hub | no automatic root deletion or retry |
| All checks and cleanup pass | qualified | finish | all disposable data removed |

Use Fleet.begin and its active count; after entering replacement, hold Fleet.mu, require it is the sole active operation, not starting, and console.done positively closed. The mutex excludes competing command admissions during publication, without a new independent lock or booleans. Stage all three candidate copies under the owned bin, validate source regular executable identity and bound, then rename each over pair/couch/pair-launch-helper. Atomicity is per executable; there is no cross-file transaction claim. No command can start through the owner during the three renames. Update only the three selected binary facts and PAIR_HOME after all succeed, preserving utilities, fixtures, RunID and every namespace variable. On partial publication, use existing unknown RunEvent after releasing the mutex; Close remains the only safe operation. Candidate owners outlive fleet cleanup. Never overwrite mapped executable bytes in place (ARCH-ORDER).

All externally persisted input is read through existing bounded typed parsers. Build paths are explicit regular files; replacement destinations are confined to the factory-owned physical bin. Registries/thread-store records come from old real product writes, not handwritten substitutes. No credential value is printed; identity comparisons and redacted summaries suffice. No production persisted format changes (ARCH-SECURE, ARCH-EVOLVE).

## Task 1: one reusable replacement seam

Files: add cmd/internal/testcouch/upgrade.go and upgrade_test.go; narrowly reuse runtime_files.go copier if necessary.

- [x] Write portable RED controls: live console, another admitted operation, cancellation and invalid candidate refuse; incomplete executable publication cannot become a runnable candidate; three installed byte contents and PAIR_HOME actually change on success while namespace/RunID/fixtures are identical.
- [x] Run only those controls, count1, and confirm the intended missing behavior fails.
- [x] Implement detached-only ReplaceBuild with the existing begin/mutex/lifetime and copier. Always remove owned staging files; never remove old state or readiness receipts.
- [x] Add pure cohort mismatch controls (missing/extra/duplicate actor, changed complete binding, changed PID/generation/native ID) and filesystem cancellation/partial-publication controls.
- [x] Run focused testcouch tests count1 and focused race count1, sequential. Commit with issue and Agent trailers.

## Task 2: same two-build acceptance for healthy and failed candidates

Files: add cmd/couch/upgrade_acceptance_test.go; modify atlas/couch-testing.md; individually audit/add affected-reader certificate when inventory identifies it.

- [x] Add opt-in TestDisposableCouchBuildUpgrade, reading explicit baseline/candidate source paths. It builds both coupled sets through BuildOwned and registers cleanup before fleets.
- [x] For each personality prepare/start old explicit hub + worker. Assert old public actor bindings and fixture readiness; snapshot exact native socket file identity and thread/session identities. Observe readiness-marker counts without deleting, creating or requiring their absence; naturally absent markers qualify the historical pre-#866 capture only.
- [x] Register one ordinary parent and one side document through the old executable using the existing bridge's public registration pattern. Retain returned IDs; candidate lookup/read must return those same IDs and document content. Snapshot old thread records through the public list and typed store reader for comparison.
- [x] Detach old console and replace the physical executable triplet plus candidate PAIR_HOME using Task1. Preserve existing actor processes, sockets, roots, registration and native conversation IDs.
- [x] Start candidate and require fresh startup adoption, public --list and --actors showing every baseline slot; use explicit --resume for each warm target, follow the operation receipt to confirmed success, then exercise console selection/attach with the real terminal policy. Independently require positive clients on the exact old session and unchanged socket/actor identities. Do not treat an old ready.json or record alone as proof candidate startup/attachment succeeded.
- [x] On failure record phase, candidate startup/CLI error and whether native helper started; run existing Close and AssertCleaned on independent cleanup context. RED requires the expected resume-unknown/listing/missing readiness receipt refusal, not an unrelated timeout or cleanup failure.
- [x] Run healthy baseline-to-same-baseline control first (count1, sequential personalities). Then run the exact same case baseline-to-failed-stage and save its expected nonzero outcome and bounded evidence under workshop/evidence/000879/. No passing retry can replace that RED.

Invocation contract (source checkouts supplied read-only by caller):

    PAIR_NATIVE_COUCH_UPGRADE=1 PAIR_COUCH_UPGRADE_BASE_SOURCE=/absolute/old-checkout PAIR_COUCH_UPGRADE_CANDIDATE_SOURCE=/absolute/candidate-checkout go test ./cmd/couch -run '^TestDisposableCouchBuildUpgrade$' -count=1 -v -timeout=20m

This test asserts compatibility and intentionally fails on the bad candidate. No hardcoded expected-failure mode in the reusable acceptance case. A combined-stage caller can supply its candidate checkout. Default suite skips this native opt-in but runs portable controls; changes to live-session/persisted-state contracts must run the documented native check before close.

## Task 3: evidence and independent landing

- [x] Record both exact source/build identities, native RED signature, all cleanup results and healthy-control result in issue Log/evidence. Update atlas with invocation and when it is required; no change to local gate configuration.
- [x] Stage new tests, run make test-affected-inventory; audit each named reader's actual sources and repair only its certificate. Run focused tests needed for any changes.

Outstanding workflow gates (owned by their verbs, tracked in issue Log): recorded queued pre-close pass; mandatory close review and finding dispositions; merge/archive; hub outcome report with landed API/invocation/evidence. Hub offered a queue grant; request only with concrete PR ready. Corrected #866's separate manual proof is independent; no unlanded branch dependency.

One atomic review boundary at issue close; no Mx tags. No estimate until change-code admits the full plan. This is harness/testing work; no production zellijquery change, operator restart, live state repair or stage deployment.

## Revisions

### 2026-10-24 — implementation seams and independent #866 check

Pure cohort facts/comparison live beside their acceptance consumer in cmd/couch/upgrade_acceptance_test.go and use scalar device/inode socket identity, so mismatch tests need no IO. Runtime replacement remains in testcouch/upgrade.go. The old and candidate checkouts need their own generated (ignored) runtime bundle; create only in owned disposable sources, preserving clean tracked source. Physical selection uses each retained Binding.Tag rather than fixture:0, because the shared menu deliberately treats repo:N filters as numbered-slot-only. Distinct old drafts are typed through that console gesture, respecting the prohibition on a hub driving its own slot with --keys. These qualify the approved oracle, rather than changing production authority. Stop the personality series on its first failed case or uncertain cleanup. Hub3ef90225 allows #866's manual immediate upgrade check independently; this fixture continues toward its own landing.

### 2026-10-24 — executed oracle details and gate checklist

Public --list is compact; --show provides the candidate process's typed read of the complete old composite record address. Public --resume internally polls and prints a raw final OperationReceipt; an already-live no-op may omit Tag, so full old-cohort proof is checked immediately after every success. The stage RED captures the initialization-unverified listing refusal at adoption, preceding the helper timeout; the clipped path and unreached later oracles are explicit in evidence, and helper_started is not invented when unavailable. Implemented steps are ticked; workflow gates remain outstanding prose/Log so they do not circularly block the close that performs them.

### 2026-10-24 18:10 — boundary review: reusable baseline independence

BR-1 identifies the class of historical prerequisites accidentally embedded in generic upgrade oracles. Enumerated all snapshot marker reads and all reusable guidance: snapshots now only log actual marker counts, accepting both marker-bearing baselines and later candidate publication, while the already captured pre-#866 absence remains historical evidence. No receipt is created or removed by the native driver. Portable generated-file regressions cover both permitted writer states, preserved bytes, empty/absent directories and failed observations. BR-2 corrects the authoritative PURE table to compareUpgradeCohort's actual consumer-side path; all table referents were checked. BR-3 adds README discovery and source-generation prerequisites alongside the existing fleet entry. These repairs preserve the approved replacement/identity/cleanup design (ARCH-PURPOSE, ARCH-PURE).
