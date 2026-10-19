---
id: 000700
status: done
deps: [pair#688]
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours: 2.21
collaboration-mode: collaboration
card_mirror: '91bfa2542b606c8e041ebc9ef2c07e6d1de3da10' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T06:12:56-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 1.19
---

# Warn when a database backup is older than its freshness threshold

## Problem

#688 found thread_history_1.sqlite had no verified bundle for roughly 55 hours while continuous passes could finish with database skips. A pass-completion marker did not expose individual database freshness.

## Spec

Warn the operator when a configured database's last verified backup bundle exceeds an agreed freshness threshold, initially proposed as two days. Derive timestamps from verified catalog records, distinguish never-backed-up and failed/deferred acquisition, and avoid repeated alert spam. Decide how a currently checked-unchanged database should be treated so an old but current bundle is not mistaken for missing data. Coordinate thresholds and recovery wording with the operator before code. Reuse #688's per-database projection; do not add a second snapshot path or hot retries.

## Done when

- A stale or never-verified database produces a clear bounded warning with the database name and one recovery action.
- Tests cover threshold boundaries, unchanged verified content, current failure, no catalog entry and deduplication across repeated checks.
- The warning uses existing operator attention surfaces and has a documented lifetime/reset policy.

## Plan

- [x] Implement and qualify the single-pass design in [the durable plan](../plans/000700-database-backup-freshness-alert-plan.md): pure freshness and daily admission, typed bounded command report, scheduler/operator delivery, deterministic and fake-backed tests.
- [x] Document the threshold, quiet-database exception, once/day lifetime and recovery; submit the recorded full suite. Close and landing follow the gates below.

## Log


- 2026-10-19: closed — BR-1 reproduced by an actual mixed failed/archived non-UTC pass through the codec, then fixed by UTC normalization at ContinuousPass.now. Corrected targeted historyarchive/archivecmd/couchcmd checks passed, including west/east clock offsets and actual archived/unchanged/deferred producer-to-codec evidence. Prior headless checks cover threshold, quiet content, partial failed reports, exact daily admission, unavailable primary hub, uncertainty, bounded subprocess output and inherited descriptors; source/dead-symbol/docs guards passed. Corrected recorded suite is running on 47fd390f227e in parallel with review under tl-worker; final code pass required before landing. Natural live acceptance remains with hub; no staging. --no-project because TL owns project records.; review verdict: SHIP
- Follow-up requested by hub in approval f477a6cb-0f84-4d98-9557-481147bd3338; filed unclaimed, independent of #688's bounded acquisition and report delivery.

## Revisions

- 2026-10-19, hub dispatch 212fe964: use #688's per-database report, default configurable 48h threshold and once-per-source/day Couch notification; catalog owns backup age, with no new store.
- 2026-10-19, hub 44e2a280: suppression is once per source per UTC day within one Couch lifetime; a fresh warning after restart is explicitly allowed. A current validated unchanged snapshot is healthy even with an old catalog timestamp. Warn only failed/deferred checks past the threshold, or immediately when never verified. No catalog timestamp refresh or recompression to make quiet databases look newer.

- 2026-10-19, BR-1 qualification revision: actual pass evidence must cross the JSON codec with non-UTC clocks, including mixed healthy/stale rows and current unchanged evidence. This extends verification without changing the approved behavior.

## Design log

- Claimed and started from fresh main in pair:7. Existing scheduler runs a subprocess daily, retries failed passes after six hours and only exposes a generic status notice. Its failure suppression is volatile. Existing dismissible/evictable attention records and lossy diagnostic logs cannot prove daily delivery, so they are not used as delivery authority (ARCH-PURPOSE).
- Existing LatestVerified already derives catalog evidence by exact source identity/path. Reuse it and carry that source identity into runtime summaries; do not scan or snapshot again (ARCH-DRY).
- Proposed configuration is the inherited PAIR_ARCHIVE_DATABASE_MAX_AGE duration environment variable, unset = 48h; invalid or nonpositive values fail before work. This avoids changing continuous.json or any existing persisted format for one optional knob (ARCH-EVOLVE). No machine environment is changed by this task.
- Read-only seam exploration confirmed the existing scheduler/report and notification boundaries. No production edits or tests/builds for #700 have started. Full plan goes to hub before change-code.

- 2026-10-19, hub e2825ba7: written plan approved, including environment threshold, bounded versioned JSON, scheduler-owned per-source daily reservations, no uncertain-write retry and headless acceptance. The first naturally stale real database is the live check, owned by the hub; no synthetic desktop alert or pair:0 staging. Proceed to change-code on the single-pass full flow.

- 2026-10-19, plan-quality round 1: PQ-1 requires an explicit global-primary-hub notification owner and unavailable/replaced-owner behavior; PQ-2 requires named function-level adversarial strategies. Durable plan now records both dispositions. No code changed. Primary-hub/fallback attention semantics are sent to hub for confirmation before rerunning the gate.

- 2026-10-19, hub 053b4ee5 confirmed PQ-1 owner/fallback contract and requested a single small archiveWarningOwner seam for pair#694's future steward-team routing. Rerun change-code; no implementation until admitted.

- 2026-10-19, plan-quality round 2: primary-hub/fallback contract accepted; PQ-2 remains open only because old task inventories duplicated the new function-level strategies. Replaced active inventories with named-strategy references and recorded the presentation revision. No implementation or estimate derived yet.

- 2026-10-19, implementation checkpoint after change-code CLEAN: pure catalog freshness, exact-source UTC-day reservations, the versioned bounded command report, CLI JSON mode, scheduler subprocess integration and primary-hub notification seam are implemented. Targeted historyarchive/archivecmd/couchcmd/artifactpath checks pass, including failed partial reports, owner-unavailable delivery, daily suppression and counted N/2N work. No real notifications or native database reads were used.
- Stateful subprocess qualification caught a real capture-limit bypass: embedding bytes.Buffer promoted ReadFrom, allowing io.Copy to skip the bounded Write. The oversize regression failed, then passed after using a named buffer field. Cancellation/inherited-descriptor tests pass with explicit fixture handshakes. Remaining work: full package qualification, documentation, recorded suite and close review; natural live acceptance remains owed to the hub.

- 2026-10-19, follow-up root review: added a valid-JSON-prefix plus oversized-padding negative control; it exposed partial-prefix authority after capture overflow. Overflow now refuses the entire report. Connected and disconnected notification bodies are bounded by the existing sanitizer; an extra connected Unicode-bound test passes. Targeted freshness/continuous/CLI/scheduler/protocol/notification tests passed across historyarchive, archivecmd, couchcmd and artifactpath. Documentation now states the threshold, quiet-data exception, exact owner, UTC/lifetime reservation and unavailable-hub behavior. Full suite and close review remain pending.

- 2026-10-19, qualification submitted: PR 457, recorded suite d66ba75ebadba36045b2818f05246ea2 on dd1ee434 is running normally. The full targeted freshness/continuous/CLI/scheduler/protocol/notification set passed; exact artifact reference, source inventory, dead-symbol and documentation guards also passed. The binary-owned close review may overlap this suite under tl-worker; a recorded pass on the final code head remains mandatory before landing. Follow-up ariadne#464 was filed unclaimed at the hub's request for the promoted IO fast-path lint rule. TL owns project records; live acceptance is the first naturally stale source, with no staging or synthetic notification.

- 2026-10-19, close round 1 REWORK (BR-1): a real non-UTC clock produced CheckedAt values rejected by the strict UTC report codec. A mixed failed/archived pass through encoding reproduced the defect. ContinuousPass.now now normalizes both injected/default clocks, matching Archiver.now; mixed rows and counted archived/unchanged/deferred passes now qualify through the codec with west/east offset clocks. The obsolete recorded suite was cancelled via local-check cancel, not treated as a flake; the corrected code gets a new recorded suite. No policy or format change. Targeted corrected historyarchive/archivecmd/couchcmd checks passed (27.273s / 0.223s / 3.184s).


- 2026-10-19, final branch qualification: recorded suite 521fa1f0cd2df2d0cb5b0fcfb2e354dd passed on 47fd390f227e85d08c6caa0008ef50f263e8d484, exit 0 in 18m2s, log SHA256 prefix 7f8a84f380c7. No flake rerun or failure. Close round 2 is SHIP, BR-1 addressed with no new findings; all four GitHub checks passed. The earlier obsolete suite was explicitly cancelled after BR-1, not counted as verification. Entering the normal merge line; it owns main integration and any integrated-code suite. Natural live acceptance remains with the hub.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only; calibration source reports stale, so hours are provisional. Derived after plan-quality CLEAN, not from elapsed task time.

Decomposition: issue/spec; three small Go concerns (freshness projection, daily admission, typed report codec); existing CLI/scheduler/notification integration and its stateful fixture; docs; one atomic close review. Existing time/json/exec libraries and current notification/catalog seams cover the stack, so there is no greenfield API discovery or novel-library allowance. Base v2 design choices are 1.0, 0.2/0.2/0.2, 2.0, 0.1 and 0.1 hours; each gets the thorough-spec 0.2 multiplier. Base implementation choices are 0.2, 0.5/0.5/0.5, 1.0, 0.15 and 0.5 hours; v3.1 scales each by 0.40. Familiar archive/scheduler/Go territory uses 1.0 familiarity, with 15% design buffer. No retry/queue mechanism or new native integration is costed.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.20 impl=0.08
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: api-integration design=0.40 impl=0.40
item: atlas-docs design=0.02 impl=0.06
item: milestone-review design=0.02 impl=0.20
design-buffer: 0.15
total: 2.21
```

Design subtotal 0.76 × 1.15 = 0.874; implementation subtotal 1.34 × 1.0 = 1.34; total 2.214, rounded 2.21 hours. Single repo, existing API seams, no cross-repo surcharge.
