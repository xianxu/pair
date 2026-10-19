---
collaboration-mode: discussion
---

# Database backup freshness alerts implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Keep policy and gate tests pure; use the existing stateful snapshot/archive fakes and a stateful command fixture for integration. The SDLC binary owns the fresh boundary review.

**Goal:** Expose a failed/deferred database whose verified protection is too old, without false alarms for content proven unchanged or repeated warnings every pass.

**Architecture:** The continuous pass keeps deriving last verification from its existing catalog index. One pure predicate projects freshness. The archive command transports its typed report to Couch through bounded JSON stdout; the existing scheduler owns volatile daily admission and uses existing operator attention/desktop output. Source acquisition, publication, scheduling and retry cadence remain unchanged.

**Tech Stack:** Go, existing historyarchive/archivecmd/couchcmd components, native child-process fixture, fake clock and existing archive/snapshot fakes.

## Approved semantics and scope

- Default age threshold: 48h. An optional inherited `PAIR_ARCHIVE_DATABASE_MAX_AGE` duration overrides it. Unset defaults; malformed/nonpositive/overflow values refuse before work. No config file or machine setting changes are needed.
- Age remains catalog `VerifiedAt`; never substitute a pass-completion marker or acquisition time. Do not mutate the catalog just to refresh age.
- A failed/deferred row needs attention if LastVerified is zero or strictly older than the threshold. Equality is not older. Future verification timestamps clamp age at zero.
- Archived content is healthy. Unchanged content is healthy only when this pass actually validated a snapshot and matched verified content (CheckedAt and LastVerified present). Preserve its old LastVerified in the report. No recompression.
- Never-verified failed/deferred rows alert immediately. Failed/deferred rows with recent protection retain their ordinary failure/defer report and existing pass-failure notice but do not get a stale-backup alert.
- Gate once per exact source identity, kind and relative path per UTC calendar day within one Couch lifetime. A restart may warn once again. Recovery followed by another stale result on the same day remains suppressed.
- Reserve before notification IO; an uncertain desktop write does not retry this pass/day. An attention-list item provides a reachable warning independently of desktop presentation. This is attempted notification, not proof that the OS displayed it.
- No new durable store, notification journal, snapshot path, timers, hot retries or background worker. Normal passes stay daily; existing failed-pass backoff stays six hours. Manual CLI runs report warnings but do not independently send desktop alerts.

## Core concepts

### Pure entities

| Name | Lives in | Status |
| --- | --- | --- |
| DatabaseBackupSummary | cmd/internal/historyarchive/continuous_snapshot_model.go | modified: carry exact source identity with existing runtime evidence |
| Backup freshness projection | cmd/internal/historyarchive/continuous_snapshot_model.go | new pure predicate/rendering from summary, clock and threshold |
| Continuous command report | cmd/internal/historyarchive/continuous_report.go | new bounded/versioned subprocess envelope, no persisted authority |
| Daily backup warning admission | cmd/internal/couchcmd/archive_alert_model.go | new pure UTC-day/source gate |

One freshness predicate serves CLI and Couch (ARCH-DRY). The command report has explicit complete/failed/not-configured/already-running outcomes, a threshold and the original typed partial report; empty or malformed output is never a healthy result. A failed child can still return useful partial database evidence.

### Integration points

| Name | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| ContinuousPass | cmd/internal/historyarchive/continuous.go | modified | existing catalog and snapshot/archive pipeline; no additional catalog read |
| Archive command | cmd/internal/archivecmd/continuous.go | modified | env option, human output or --json stdout; stderr remains diagnostics |
| Archive scheduler | cmd/internal/couchcmd/archive_schedule.go | modified | nice/pair subprocess, bounded stdout capture and existing latest-pass log |
| Operator warning delivery | cmd/internal/couchcmd/message_service.go | modified | existing attention fact plus serialized desktop presenter |

The scheduler launch seam returns a typed command report and process error. Decode report stdout independently of exit code so a failed pass can warn about its partial rows; invalid/absent/oversize JSON yields an explicit protocol failure and the ordinary backup failure recovery surface. Keep raw logs bounded as currently owned; never parse human text to reconstruct authority.

## Operator projection and recovery

Actions: ask the hub to repair the named backup; retry a pass after repair; dismiss an attention item.

| Internal combination | Visible state | One primary recovery action | Possible loss |
| --- | --- | --- | --- |
| Archived, or validated unchanged with verified content | Healthy/current row; original verified time remains visible | None | No missing known content inferred |
| Failed/deferred with recent verified protection | Existing failed/deferred row and existing pass failure notice as applicable | Ask hub to repair the named backup | Changes since last verified version may be unprotected |
| Failed/deferred and old/never verified; eligible day | Bounded warning naming database, age/never, reason; attention item and one desktop attempt | Ask hub to repair the named backup | Same, or all content if never protected |
| Same stale source again on same UTC day | CLI/log still shows stale; no second attention/desktop attempt | Use the existing warning to ask hub for repair | No additional deletion or retry |
| Child/report unavailable or malformed | Backup check unavailable, with existing log location; never healthy | Ask hub to repair the backup check | Freshness is unknown |
| Couch restart | Next stale result may emit one fresh warning | Use the warning to ask hub for repair | Only duplicate attention; backup evidence is unchanged |

Warnings are bounded/sanitized and point to continuous.log for detail. They do not claim a full failure report is itself recovery. Reuse current attention ownership/routing; do not invent an agent-exit or source-removal state.

## Resource and format contracts

- Catalog age uses the index already loaded once per pass. No new catalog scan, native directory walk, source open or snapshot attempt.
- For N database rows: N freshness evaluations and at most N admission lookups/output attempts. Tests at N and 2N count these operations and assert no extra catalog or snapshot calls.
- Subprocess report stdout is bounded by MaxMetadataBytes (4 MiB), decoded once, duplicate keys/known-field types/version validated. stderr is logged separately; no free-form text becomes an authority field.
- Daily gate is single-owner in the scheduler goroutine. Prune older-day reservations on forward UTC-day rollover; a backward clock must not reopen the earlier day. Cap reservations at historyarchive.MaxFiles (10,000 distinct sources per UTC day); one overflow notice per day says alert admission is incomplete and asks the hub to repair the backup check. Overflow never permits duplicate output or labels a source healthy.
- The gate dies with Couch, as approved. Existing continuous.log is overwritten per pass; existing attention items follow their current dismissal/eviction lifetime. Neither controls admission.
- Existing persisted config, catalog, manifest, labels and marker formats are unchanged. Runtime JSON is an additive observation protocol: accept unknown fields, retain the raw bytes when writing the existing diagnostic log, keep known field/duplicate/version checks strict. No read-modify-write of an unknown record occurs (ARCH-EVOLVE).
- Keep one existing scheduler worker and existing serial backup lane. No new queue or retry mechanism.

## Implementation tasks (one atomic close, no Mx tags)

### 1. Pure policy and daily gate — red before green

Files: continuous_snapshot_model.go and tests; archive_alert_model.go and tests; new files registered in artifactpath/manifest.go.

- [x] Implement the ParseDatabaseMaxAge and DatabaseBackupSummary.NeedsAttention/Warning test strategies in the function-level table below.
- [x] Implement the archiveWarningGate.Reserve event-sequence strategy in the function-level table below.
- [x] Run targeted tests and retain behavioral red evidence before implementing the predicate/gate.
- [x] Implement pure rules with one freshness owner and reservation-before-effect semantics; run tests green and commit.

### 2. Typed report through the existing command — red before green

Files: continuous.go, continuous_report.go, archivecmd/continuous.go, run.go and corresponding tests.

- [x] Implement the EncodeContinuousReport/DecodeContinuousReport strategies below against real pass evidence and the command boundary.
- [x] Implement the ParseDatabaseMaxAge and report-decoder adversarial/fuzz strategies below through the CLI boundary.
- [x] Implement bounded versioned encoding/decoding using existing strictjson structural validation; avoid shared decoder rewrites or persisted codecs.
- [x] Keep stderr diagnostic output separate from typed stdout. Preserve CLI nonzero exit for failed passes and existing lock behavior; run targeted tests green and commit.

### 3. Scheduler delivery and qualification

Files: archive_schedule.go/tests, message_service.go, new notification integration tests as needed, atlas/history-backup.md and README.

- [x] Build a stateful fake pair executable behind the actual launch seam. It emits scripted report/exit/diagnostic combinations and records starts, without SQLite, personal data or real desktop output.
- [x] Implement the archiveScheduler.tick and messageService.notifyArchiveWarning/archiveWarningOwner production-seam strategies below, retaining behavioral red evidence.
- [x] Wire the typed launch result to pure admission and existing attention/desktop delivery, preserving generic failed-pass backoff and cancellation. Inject fake output in tests; do not issue real operator notifications during development.
- [x] Count N/2N rows and existing snapshot/catalog calls; run targeted archivecmd/historyarchive/couchcmd tests and the source inventory guard.
- [x] Update operator docs for threshold env, healthy quiet databases, per-lifetime UTC-day suppression, restart duplicate and recovery action. Capture any review lesson.
- [ ] Submit PR and recorded full suite on the pushed code. Close via the binary's review, then enter sdlc merge --yes; the landing turn integrates main and qualifies changed integrated code.
- [ ] Report Git-printed merge hash/PR and gate evidence to the hub; leave project updates to the TL and stop.

## Acceptance proposal

Use deterministic fake-clock and stateful fake-subprocess/headless notification seams for acceptance: no pair:0 staging, native source read, clipboard probe or desktop alert is needed to prove the policy. Ask hub to confirm this arrangement with the plan. The existing desktop presenter has native conformance already; the operator's next natural restart can exercise normal backup warnings without creating a false stale incident.

## Revisions

- 2026-10-19: incorporate hub 44e2a280: volatile UTC-day admission with restart duplicate allowed; validated unchanged content is current, preserving original catalog time. No implementation before hub sees this written plan and change-code admits it.

### 2026-10-19 — plan-quality round 1 disposition (before implementation)

PQ-1: The backup is global, so its notification owner is Couch's primary hub (`authority.hub`, the first explicit hub, or the implicit launch worktree). Do not choose a hub from a database path or a worker's repo. At emission use existing `messageService.hubBinding()`/`hubBindingAt` to resolve the primary hub's newest admitted connected incarnation, recheck connectivity, and derive the attention Thread through `bindingThread`; use that hub's label in multi-hub mode. Other hubs receive no duplicate notification. A replacement does not reset the source/day reservation; an item already emitted stays bound to its original exact thread rather than silently retargeting another conversation.

If there is no proven connected primary hub (including Couch started without a hub), emit the bounded existing Couch status notice and serialized desktop attempt and retain the report in continuous.log, explicitly saying the hub is unavailable. Do not create an unroutable attention item, queue delivery, search for a substitute owner or retry on reconnect. Daily admission is still consumed by this attempted global warning; the next eligible day or Couch restart may warn again. The warning/report remains reachable through the existing log, while the attention list is available only with a proven owner. If replacement races after the final check, the existing exact-thread attention selection refusal applies; preserve the text and do not rearm admission. Hub approval of this explicit fallback is requested before the next gate run.

PQ-2: Function-level test strategies (tests before implementation):

| Risky function / transition | Adversarial class | Mechanical guard |
| --- | --- | --- |
| ParseDatabaseMaxAge(raw) | empty/default, zero/negative, malformed and overflowing durations | Table tests pin default and refusal before side effects; bounded fuzz strings never panic or admit nonpositive values |
| DatabaseBackupSummary.NeedsAttention(now, maxAge) | threshold equality, future times, zero verification, contradictory unchanged/check evidence | Exhaustive table across existing statuses; property tests vary clocks around boundary and require no timestamp mutation |
| DatabaseBackupSummary.Warning(now, maxAge) | controls, long Unicode paths/details, missing evidence | Deterministic rendering and rune/byte bounds; exact database/age/recovery wording asserted |
| archiveWarningGate.Reserve(now, key) | duplicate outcomes, same path/different identities, recovery, UTC rollover, backward clock, capacity | Reproducible generated event sequences against a small reference set; at most one admission/key/day, one overflow/day, bounded map and no IO |
| EncodeContinuousReport / DecodeContinuousReport | empty/trailing/duplicate/truncated/oversize JSON, wrong versions/types, unknown fields and invalid outcome/row combinations | Round-trip tables and native Go fuzz seeds; size check before decode, structural duplicate checks and explicit known-field validation; preserve raw logged bytes |
| runArchivePass subprocess seam | partial report plus nonzero exit, empty output, held stdout/stderr, cancellation | Stateful fake pair executable drives actual OS pipes and exits; bound retained stdout, preserve typed partial data, join command and retain process error independently |
| archiveScheduler.tick | stale partial pass followed by retry/day/restart/recovery | Fake clock plus real launch fixture; count child starts, gate admissions and output attempts; existing cadence unchanged |
| messageService.notifyArchiveWarning / archiveWarningOwner | no hub, multiple hubs, primary replacement/disconnect, uncertain desktop write | Existing live service fixture with fake attention/presenter outputs; exact primary/latest binding only, fallback emits no phantom item, reservation never rearmed |

Counted N/2N integration checks assert one projection/admission attempt per row, no extra catalog read or snapshot, and at most one output attempt per eligible source. These strategies replace adding independent test lists without identifying the responsible function.

- 2026-10-19, hub 053b4ee5: primary-hub resolution and unavailable-owner fallback approved. Keep `messageService.archiveWarningOwner` as one small owner-resolution function so pair#694 can redirect machine-wide backup stewardship without changing freshness/admission/delivery logic. All output attempts consume the daily reservation as above; no broadcast, inferred owner or reconnect retry.

- 2026-10-19, plan-quality round 2: PQ-1 addressed. PQ-2 accepted the named strategies but required removal of the duplicated active case inventories. Tasks 1–3 now reference the function-level strategy table instead; this is a presentation correction, not another mechanism/design change.
