---
collaboration-mode: discussion
---

# Couch Scheduler Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Run each hub's declared operational tasks on schedule, with one cached Go entrypoint for automatic and manual execution, explicit missed-run behavior, and inspectable local outcomes.

**Architecture:** A pure recurrence/ownership/occurrence reducer decides effects from explicit time and observations. A supervisor-owned coordinator discovers only configured hub repositories, commits occurrences before effects, and uses bounded build/process/notice adapters. The existing Couch socket serves status; the manual helper uses the same task builder, execution owner, local store, and per-task execution lease.

**Tech Stack:** Go, the existing Couch service/broker and Unix process ownership, `durablefile`, `strictjson`, the existing machine test-lock helper, and the parsing/next-time portion of `github.com/robfig/cron/v3` pinned at v3.0.1. No separate cron daemon, election, shell task interpreter, or general sandbox.

**Issue:** pair#737. **Project/design:** `ops/teams/devinfra/projects/couch-scheduling/{project,design}.md` (TL-owned; read only here).

**Approval boundary:** Approved by the operator via ops:0 on 2026-10-20, with the three superseding decisions recorded in Revisions. Run `sdlc change-code` and implement M1 first. The first close's live acceptance still needs the coordinated ops task described below.

## Settled contour and resolved discrepancies

- Each task is a Go package at `scripts/scheduled/<task>/main.go`, not a checked-in executable named `main`. Couch builds a cached executable and invokes its subcommands. The shared human/agent entrypoint is `couch --schedule run <task> -- <args>` (TL clarification, 2026-10-20).
- **There is no general sandbox in v1.** Tasks are reviewed team-repository code, executed as the operator, with the hub repository as cwd, network access, a timeout, bounded output, and process-group cleanup. Heavy tasks take the existing machine test lock. This supersedes the original issue's sandbox/all-task-budget wording; no isolation guarantee is implied.
- Implement the small team-owner admission check now. The earlier project text reserving team scope is superseded for this check only: no real team task, election, distributed lock, or failover is added. A team owner must be listed in that hub's `team.md` Operators section. Missing section/file is tolerated for machine tasks and refuses team tasks.
- The live Operators format is `## Operators` followed by `- <slug> (<display name>): <description>`. The operator slug is the repository-effective `git user.name`, lowercased and normalized to ASCII alphanumeric/hyphen form; empty identity refuses team admission. The comparison is not based on the worker agent or the first hub.
- Existing archive/Drive timers remain in place. Migrating them and shipping the ops tasks belong to later project items. No edits to the ops project, team files, or task code occur in this issue.

## Core concepts

### Pure entities

| Name | Lives in | Status | Responsibility and relationships |
|---|---|---|---|
| `TaskSpec`, `OwnerAdmission` | `cmd/internal/couchschedule/spec.go` | delivered M1 | Validated declaration and deterministic team owner admission. |
| `TaskKey`, `Declaration`, `TaskState`, `Attempt`, `RunSummary`, `Event`, `Effect`, `Advance`, `NewRecoveryEvent` | `cmd/internal/couchschedule/model.go` | delivered M1; recovery producer extended M2 | Per-task transition authority; attempts represent occurrences, with scheduled due instants or manual nonces. Recovery construction requires an observation time. |
| `Calendar` | `cmd/internal/couchschedule/calendar.go` | delivered M1 | Five-field cron and strictly-future `Next` in an injected location. |
| `BuildManifest`, `BuildInputDigest`, `buildContentKey` | `cmd/internal/couchschedule/build_manifest.go` | delivered M1 | Captured input digests and content key, independently of task arguments. |
| `StatusSnapshot`, `TaskStatus`, `BuildStatus` | `cmd/internal/couchschedule/status.go` | delivered M1 | Immutable status projection with per-task refusals. |
| `Record`, `RunKey`, `OutputBuffer` | `cmd/internal/operationalrun/record.go`, `output.go` | delivered M1 | Run execution metadata, durable identity, and capped head/tail output. |

The key is rooted in the **hub checkout**, never its claimed product repositories. One task has many occurrences; a scheduled occurrence is keyed by task, declaration revision, and due UTC instant. Manual attempts have a separate nonce and never consume or advance a scheduled occurrence. One state snapshot owns current task authority; immutable per-run records and bounded output live separately under the Couch data folder. The reducer owns scheduling decisions (ARCH-PURE/ARCH-ORDER); adapters must not implement competing catch-up or exit-code rules (ARCH-DRY).

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| `Catalog` | `cmd/internal/couchschedule/catalog.go` | delivered M1 | Bounded repository discovery, Operators, Git identity and declaration revision. |
| `Ledger`, `StateStore` | `cmd/internal/couchschedule/store.go`, `store_unix.go`, `state_json.go` | delivered M1 | Locked, atomic additive authority snapshots; separate from run history. |
| `StateStore.WithAuthority` | `cmd/internal/couchschedule/store.go` | implemented M2 | Retain existing scheduler authority lock through deadline-bounded external admission, without rewriting the snapshot. |
| `Builder`, `Artifact` | `cmd/internal/couchschedule/build.go`; cache in `cache.go` | delivered M1 | Captured Go builds, cache publication, lifetime leases and retention. |
| `buildScratch`, `BuildCleanupError` | `cmd/internal/couchschedule/capture.go`, `execute_unix.go` | delivered M1 | Two capture slots and held command-writer leases; explicit synchronous join for uncertain cleanup, reused by the execution owner. |
| `TaskExecutor`, `Executor`, `ExecutionRequest`, `ExecutionResult` | `cmd/internal/couchschedule/execute.go`, `execute_unix.go` | delivered M1 | Controller/process ownership, timeout and bounded output. |
| `acquireExecutionPermits`, `executionLease` | `cmd/internal/couchschedule/admission.go` | delivered M1 | Per-task lease and two machine permits; declared heavy uses existing test-lock helper. |
| `FakeTaskRuntime` | `cmd/internal/couchschedule/runtime_fake.go` | delivered M1 | Stateful execution double with held starts, exits and cancellation. |
| `Store.Publish`, `Store.List`, `Store.Read`, `Store.SweepPage` | `cmd/internal/operationalrun/store.go`, `retention.go` | delivered M1 | Immutable execution bundles and shared operational retention. |
| `automaticScheduler`, `scheduleClock`, `nativeScheduleClock` | `cmd/internal/couchcmd/schedule_automatic.go` | implemented M2; live acceptance pending | Due transitions, injected cancelable timer stream, joined bounded effect workers, fair hub dispatch and saved-result recovery. |
| `scheduleCoordinator` | `cmd/internal/couchcmd/schedule_service.go` | delivered M1; extended M2 | Joined service lifetime, declaration refresh, recovery and published snapshots. |
| `manualTaskRunner` | `cmd/internal/couchcmd/schedule_manual.go` | delivered M1; extended M2 | Durable manual reservation; shared execution of persisted manual/scheduled reservations and run publication. |
| `scheduleNotice`, `scheduleNoticeAuthority` | `cmd/internal/couchcmd/schedule_notice.go` | implemented M2; live acceptance pending | Exact hub binding, gate-before-authority admission callback, broker admission and canonical receipt observation. |
| `parseScheduleCLI`, `runScheduleCLI` | `cmd/internal/couchcmd/schedule_cli.go`, `schedule_status.go` | delivered M1 | Closed CLI grammar, status exchange, manual run and retained history. |
| `Store.PublishOccurrence` | `cmd/internal/operationalrun/occurrence.go` | implemented M2; live acceptance pending | Canonical per-occurrence history adoption with existing task locks and durability checks. |
| `Store.ReadDurable` | `cmd/internal/operationalrun/occurrence.go` | implemented M2; live acceptance pending | Existing-record task-locked durability confirmation; never creates absent history. Inspection retains `Store.Read`. |
| `ensurePassiveRun`, `ensureUncertainRuns` | `cmd/internal/couchcmd/schedule_records.go` | implemented M2; live acceptance pending | Preserve already durable summary facts before replacement and publish protected unknown history after classification commits. |
| `quarantineTaskHistory`, `taskHistoryError` | `cmd/internal/couchcmd/schedule_records.go` | implemented M2 | Shared task-local history refusal and retained pending-evidence retry classification. |
| `Store.FinalizeNotice` | `cmd/internal/operationalrun/notice.go` | implemented M2; live acceptance pending | Monotonic pending delivery finalization under the existing task lock. |

Reuse `couchcore.Runner`/`FakeRunner` for process identity and group lifetime where their contract fits. They inherit output today; do not pretend they already capture scheduled stdout. Add a focused scheduled-execution adapter and reuse `procutil` group/identity primitives rather than changing terminal/PTY execution semantics. `zellijexec.Run` is explicitly a read-only Zellij boundary and must not be reused unchanged for arbitrary tasks.

## Declaration and command contracts

### Proposed `schedule` file

Use a small UTF-8 JSON object named `schedule` beside `main.go`; this avoids adding a YAML parser and makes required types/duplicates deterministic. This exact format is part of operator review, then becomes the ops-task contract:

```json
{
  "version": 1,
  "cron": "0 7 * * *",
  "timeout": "10m",
  "scope": "machine",
  "missed": "skip",
  "script_run": true,
  "heavy": false
}
```

- Required: version 1, a five-field cron string, positive timeout up to 24h. Defaults: `scope=machine`, `missed=skip`, `script_run=false`, `heavy=false`. `scope=team` additionally requires `owner`.
- `missed` accepts `skip` or `once`. `script_run` explicitly opts into the implemented subcommand; do not discover capabilities by running the task, or scrape human help output. A declared but missing subcommand is a task failure, never a silent fallback.
- `heavy=true` takes one machine suite slot before build/run. It is a declaration of resource cost, not a sandbox policy. Nested test invocations inherit the verified owner marker only after admission.
- Cron uses minute/hour/day/month/weekday fields through a parser configured without seconds/descriptors. Reject extra fields and `TZ=`/`CRON_TZ=` prefixes. Interpret in the machine location supplied by the IO shell; persist due instants in UTC and display the location. Spring-forward nonexistent times skip; fall-back repeated local times run once, at their first matching instant; the repeated interval is suppressed even if Couch starts during its second copy. These are explicit reviewable semantics, pinned by fixtures.
- A declaration error refuses that task, with its hub/path/reason visible; it does not stop other tasks. Missing `scripts/scheduled` is an empty catalog, not an error. Missing `main.go` for a declared task is a refusal.
- Read declarations from the configured hub root only. Task names are single plain directory components; reject traversal, control bytes, absolute references, symlink escapes, duplicate/case-colliding task names and declarations over 32 KiB. Observe an invalid replacement as disabled/refused; do not continue executing an old last-good declaration silently.
- Read Operators only until the next heading of level 1 or 2. Parse the leading slug in the agreed list format; reject duplicate slugs/ambiguous entries for team validation. A listed owner on another operator's machine is `not-owner`, not an invalid declaration. Machine scope does not require Git identity or Operators.

### CLI

```text
couch --schedule --status [--json]
couch --schedule history <task> [--json]
couch --schedule history <task> --run <run-id>
couch --schedule run <task> -- <args...>
couch --schedule run landing-plan -- --help
couch --schedule run landing-plan -- script-run
```

Status is read-only and uses the existing operator-private socket/transport, like `--resources`. It reports hub, task, declaration state, scope/owner, cadence/location, last attempt/source/result, next due, and active stage; never spawn/build or refresh the filesystem on a status request.

For `run`, resolve cwd through existing Git/workspace resolution and the supervisor's configured hub catalog. A product worker checkout does not become a scheduler root and cannot select another hub's task by passing a path. Task selection is one slug in the current hub; same-name tasks in two hubs are unambiguous. Resolve the hub once, then use the same catalog/builder/executor/store as automatic execution. Argument tails are argv, not shell text. `--help` is passed to the built task; the helper does not reinterpret task flags.

The manual process owns cancellation, streams/captures bounded task output, and returns the task exit code (including 10); it does not auto-wake the hub or advance `next_due`. It records a manual attempt so status remains honest. Automatic execution interprets 0/10/other through the reducer. The helper requires the configured Couch catalog to resolve its hub; no offline scan of unrelated repositories. Every manual or automatic process attempt takes the same per-task lease **and one of two machine-wide execution-permit flocks** before starting build/task work. A manual caller cannot evade the global cap by executing a different task.

## Time and state semantics

### Recurrence

- First discovery with no saved state arms `Next(now)` and does not fabricate a missed run. A declaration revision re-arms from now rather than replaying the prior definition's backlog.
- Normal due checks reserve an occurrence if observed within its scheduled minute. A clock jump/pause that misses that minute follows missed policy. Backward jumps never repeat a consumed UTC occurrence.
- At restart or after a sleep gap, overdue saved slots advance directly to `Next(now)`. Default records one `skipped-missed` summary; no loop over missed ticks. `missed=once` reserves **one** catch-up occurrence, irrespective of missed count, and also advances next due beyond now.
- An occurrence observed and reserved while running may wait for local worker/machine admission; it is not a new catch-up run. The declaration's one end-to-end timeout covers admission, build, execution and cleanup initiation. Timeout is recorded, the next scheduled slot remains armed, and no immediate automatic retry follows.
- At most one active attempt per task; manual and scheduled runs contend on the same held execution lease. A due slot overlapping an active attempt records `skipped-running` and advances; no unbounded backlog. Round-robin dispatch between hubs prevents one hub's ready tasks taking all worker starts.
- Hub connectivity does not gate direct execution: an eligible `script_run` task reserves and runs when due even if its hub is offline. A notify-only task with an offline hub records `hub offline, skipped` and advances immediately. An exit-10/failure notice whose hub is offline is dropped with an explicit retained status/history line; reconnection does not retry it. There is no waiting-hub state. Wrong/missing owner is refused/non-owner with next due advanced, never an offline retry.

### Reducer shape

```go
type Event struct {
    Kind EventKind
    At time.Time
    Declaration *Declaration
    Revision string
    Attempt string
    // Typed payload for catalog, persistence, admission, build, process,
    // receipt or cancellation observation; no ambient reads here.
}
func Advance(state TaskState, event Event) (TaskState, []Effect)
```

`TaskState` holds an immutable current declaration and `NextDue`, plus an optional tagged active attempt. Persisted attempt stages are reserved, running, notifying, delivery, stopping, and uncertain. Admission/build are execution-owner work within the running stage; visible projections may name that work without adding a second phase authority. Process attempts retain source (manual/scheduled), controller PID/start token, work-group identity, owning parent, and absolute deadline. Final records carry quiet/wake/failed/skipped/refused/uncertain outcomes. Typed event/stage/attempt checks ignore stale or duplicate completions; callers cannot mutate authoritative phase fields directly.

| State/event | Decision/effect | Next visible state |
|---|---|---|
| New task / catalog accepted | Persist initial next due | Waiting |
| Waiting / due, eligible owner | Persist reserve plus next due; direct task runs regardless of hub connectivity; offline notify-only task records skip without admission | Reserved or hub offline, skipped |
| Waiting / due, refused or different owner | Consume the slot without effects and advance next due; retain the owner reason | Refused/not owner |
| Reserved / commit durable | Obtain per-task execution lease, then admission/build or due-notice effect | Queued/building/notifying |
| Reserved / write or directory-sync failure | No external effect; retain failure evidence | Refused/uncertain storage |
| Active / another due or manual request | Skip scheduled overlap; manual returns busy with the active attempt | Existing active attempt |
| Building / cache ready with same revision | Execute the pinned binary and argv | Running |
| Running / exit 0 | Record completion, bounded output, release run owner | Quiet |
| Running / exit 10 | Record completed execution and freeze one notice body/ID; never execute again while notifying | Notifying |
| Running / other exit or timeout | Stop/join owned group; record failure; freeze failure notice | Failed + notifying |
| Notifying / exact hub offline | Drop notice, retain output and explicit reason; no reconnect retry | Notice dropped, hub offline |
| Notifying / connected hub, known non-admission | Retry only admission within the one-minute window | Notifying |
| Notifying / broker admission or uncertain commit | Retain receipt; observe that same receipt, never allocate another paste | Queued/submitted/uncertain delivery |
| Active / declaration removal, changed owner, or shutdown | Cancel not-started work; stop/join started group; retain outcome of this attempt | Cancelled/interrupted |
| Reopen / reserved or started attempt without terminal record | Never replay automatically; reconcile exact native owner, preserve unavailable system receipt as terminal delivery uncertainty, apply missed policy to future slots | Recovering/next due |
| Reopen / live manual controller | Observe its shared store and held permits; do not cancel or redispatch work owned by a different live CLI parent | Running manual |
| Any / completion for older attempt/revision | No scheduling or notification effect | Unchanged |

Persist the occurrence reservation **before** starting a task or admitting a notice; this provides at-most-once automatic admission per occurrence, not exactly-once task side effects. A process may have done work before a crash or failed result publication. Never report that as a confirmed failure-to-run or retry it silently. A live/unknown retained run owner blocks a new attempt until its lifetime is reconciled. Intentionally daemonized/escaped descendants are outside the reviewed task contract; v1 tasks must return only after their work completes.

### Actions, failure visibility and recovery

Actions available to the operator/hub: inspect status/log; edit/commit the declaration or task; invoke the manual helper; stop the owned run through helper cancellation; repair the named local storage/process condition; restart Couch after repair. No election, automatic effect retry, or new general job-control UI.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Invalid cron/owner/module or missing Operators | Refused, exact field/path reason | Edit/commit declaration/team file, then inspect status after reload | This slot is not run |
| Valid owner belongs to another operator | Not owner, next due shown | Correct reviewed owner if responsibility changed | No automatic cross-machine failover |
| Notify-only due while hub offline | Hub offline, skipped | Inspect history; invoke manually if still useful | Due notice lost; no reconnect replay |
| Direct run finishes while hub offline | Actual task result, notice dropped/hub offline | Read retained output through history | Wake/failure notice lost, task result retained |
| Broker reports delivery expiry | Delivery expired | Inspect history; invoke manually if still useful | Notice not delivered; no backlog replay |
| Build/toolchain/dependency failure | Failed build, bounded log | Repair source/toolchain, then manual helper | This occurrence is not executed |
| Heavy admission paused/expired/unavailable | Queued or admission failed | Inspect shared queue; retry manually after repair | This occurrence may expire |
| Timeout or helper cancellation with joined group | Timed out/cancelled | Inspect log, then manually rerun if safe | Partial task work may remain |
| Crash/result write failure/native owner uncertain | Interrupted/uncertain; no replay | Inspect exact retained owner/log and repair/stop it before manual rerun | External effects may already have occurred |
| State corrupt/unreadable/too new | Scheduler blocked for that store, no effects | Repair/restore the named local file, then restart | No invented last-run/next-due evidence |
| Broker commit unconfirmed | Delivery uncertain, canonical receipt ID shown | Inspect receipt while available; after broker restart retain uncertainty and never resend that occurrence | Hub input may already be queued/submitted |

## Build, execution and local persistence

### Build cache

Choose a **private captured-source build view**, not before/after hashes around compilation in the live checkout. V1 tasks use the hub's ordinary root Go module, with pure-Go code (`CGO_ENABLED=0`), no vendor/workspace mode. This proposed build envelope is explicit operator-review material; unsupported inputs refuse with a reason instead of silently weakening the cache identity.

1. Resolve the installed local Go toolchain and native target once per build admission. Use explicit `GOWORK=off`, `GOTOOLCHAIN=local`, cleared ambient `GOFLAGS`, and fixed `-trimpath -buildvcs=false` options for graph/build commands. Module graph resolution may fetch normal pinned dependencies; it must not rewrite the hub's go.mod/go.sum.
2. Run bounded `go list -deps -json` and module metadata queries against the selected task. Reject incomplete graphs, unsupported cgo/SWIG/foreign-object inputs and escaping local replacements. Capture all selected non-standard Go/assembly/header/embed inputs and module manifests as bytes; relative local replacements stay inside the captured hub tree. Preserve original module identities and selected versions. A dependency edit or missing/generated embed input cannot disappear from the hash.
3. Materialize those captured inputs into a private staging view, with deterministic module-to-directory mapping. Generate a private root modfile whose replacements point **every captured non-standard module** at its captured copy; include the selected module-manifest graph so the build need not read dependency sources from the live checkout/cache. Reject absolute/escaping local replace paths; do not copy or mutate the live modfiles. Standard-library/compiler inputs belong to the trusted, version/fingerprint-pinned local toolchain.
4. `buildContentKey` hashes canonical module/path labels, captured file bytes, original module metadata, normalized private-view recipe, toolchain fingerprint, target and fixed options. Build only that view, with proxy/module fetching disabled during the actual compile. An import added during graph capture that was not captured makes the build fail; it never gets fetched behind the manifest's back. Bound capture to 64 MiB/10,000 files and reject unsafe/symlinked inputs.
5. Reuse a validated cached executable for that key. Otherwise build a temporary executable, hash it, and atomically publish binary plus manifest while holding the same-key build lease. Recheck the declaration/owner revision before task execution; a completed stale build may be cached but cannot execute a superseded task. Never publish a partial file or claim an unverified artifact matches its manifest.

`Builder` must not call arbitrary `go generate`, execute a task to detect capabilities, or write a binary into the hub repository. Go's ordinary module fetches and toolchain selection are operator-level IO, not sandbox isolation. Discovery/status never compile packages. Automatic notice-only tasks build only when the hub uses the manual helper. Complete snapshots die when the build adapter returns; cached binaries have the separate retention budget below.

### Process and heavy admission ownership

Give each execution a short-lived owner process, not the long-lived Couch PID. For a heavy attempt, that owner waits before executing any task code while `scripts/test-lock.py acquire --watch <owner-pid> --slots 1` obtains admission. Resolve the helper through the workflow source location (as `Makefile.workflow` does), not an assumed sibling path or a copied Python script.

Sanitize inherited lock bypass/reentrancy variables before admission. Accept a positively acquired lease only; a verified same-owner nested run may reuse it. Treat skipped, malformed, missing-helper, paused, timeout and uncertain output as non-admission. Set `WF_TEST_LOCK_HELD_BY` to that owner only **after** acquisition for nested test commands. Do not watch the Couch PID, spawn task work before admission, or release the owner while its task group is still alive.

The watched **controller stays outside the separate build/task work group it terminates**. The controller owns the per-task lease, one of the two machine-wide permits, a parent-lifetime pipe and the work group's direct child handle. Its identity/source is durably recorded while it is blocked; only a later authorization acknowledgement permits admission/build/task effects. Parent EOF/cancellation starts work-group TERM, then KILL after a one-second grace, then reaping/pipe-drain completion. The controller exits last, keeping admission and execution permits until cleanup is confirmed; retain an uncertain owner if it cannot be confirmed. Use `procutil` PID/start-token matching; bare PID disappearance is not proof that a surviving group is gone. Native parent-death tests must observe both descendant cleanup **and actual machine-slot release** in that order.

The manual CLI parent, rather than Couch, owns a manual controller. A Couch restart observes a surviving exact manual controller and its held permits without cancelling it or issuing its work again. A scheduled controller is owned by the scheduler service and cancels on that parent's EOF. Store updates name source/controller identity/attempt; stale results from either caller cannot clear another active owner.

The same end-to-end timeout bounds manual and automatic admission/build/task work; cached builds still obey it. Cleanup has its separate grace and uncertainty outcome. A notice-only occurrence uses its original deadline for admission. A completed direct task needing a wake/failure notice gets a separate, persisted one-minute admission window starting when its terminal result is recorded; a timeout can therefore still report its failure. Neither deadline invents an expiry for input already accepted by the broker. Capture stdout/stderr with a writer that continues draining after the retained 256 KiB head-and-tail cap (128 KiB each, with total byte count and a truncation marker), marks truncation, and cannot bypass its cap through `io.ReaderFrom`. Notice rendering removes terminal controls and truncates UTF-8 safely to the broker's body limit; output is not shell input. Task output is reviewed operational data, retained privately, and may be deliberately delivered on exit 10.

### Store and retention

- Keep scheduler authority in a private namespace-local `schedule/state.json`, outside hub repositories. Store one result per run under the Couch data folder at `schedule/<hub>/<task>/runs/<run-id>/`, with strict metadata and capped output. Canonical hub identity maps to a safe collision-resistant path component. Each result records task, declaration revision, scheduled/manual source, scheduled instant if any, start/end, exit (nullable while unknown), outcome and notice-delivery state. Do not embed historical results/output in authority state. Manual and automatic writers use short context-aware flock transactions and `Advance` against freshly loaded state, so one cannot erase another outcome/next due. Publish a durable run record before clearing completed authority; recovery can finish that idempotent publication without rerunning task effects.
- Use `strictjson.SplitObject`/`EncodeObject` at **every** persisted object level; preserve unknown members on read/modify/write. Known types, required fields, duplicate members, bounds, timestamps and versions remain strict. Schema corruption never resets the ledger into a runnable empty state.
- Publish with `durablefile`; failed file/directory durability prevents first effects. A result-publication failure after execution is recorded/returned as uncertainty, not a reason to execute again.
- Operating envelope: at most 256 tasks per supervisor; 32 KiB/declaration and 2 MiB authority state; 256 KiB head/tail output per run. Two machine-wide execution permits shared by all manual/automatic processes and supervisor namespaces, one lease per task. Admission/build/run use the declared timeout; cancellation/join is a separate bounded cleanup phase with visible uncertainty if it fails.
- Register run folders as Couch-wide **operational data** through the same retention mechanism used by diagnostic logs (#677/#687), default 30 days and 300 runs per task. Reuse its preview, identity-fenced deletion and failure reporting; no parallel age-pruning loop. Protect live/uncertain authority from retention; retained terminal result/output can age out without making a task runnable again. Add `couch --schedule history <task> [--json]` for run lists and `--run <run-id>` for one run's metadata/output. History is an explicit bounded IO operation; status still reads only the coordinator snapshot.
- Cache budget: 256 MiB per supervisor. Evict unused build keys by LRU after publication; never remove a live execution's binary. Shared operational-data retention removes each expired run record together with only its owned output. Removed-task state is pruned after 30 days, retaining active/uncertain owners. Sweep only validated scheduler paths; symlink or identity ambiguity refuses that cleanup rather than following it.
- These caps are initial design assumptions, not measurements. Exceeding a cap is visible and bounded; no silent partial catalog or unbounded history growth (ARCH-CONSTRAINTS/ARCH-FUNERAL).

### Operational-data adapter (PQ-1)

Existing `diagnosticlog` owns exact log generations, while `storagegc` transactions require session/capture owners; neither is a run-directory API. Add a focused `cmd/internal/operationalrun/` package with `store.go`, `record.go`, `retention.go`, `collection.go` and `store_unix.go`. `Store{Namespace}` owns publication, bounded List/Read and `SweepPage(ctx, now, cursor, apply, limit)`. Scheduler authority remains in couchschedule; result envelopes belong to operationalrun, avoiding a retention dependency on the scheduler service. Its independent operational policy is 30 days/300 completed runs per task, not the diagnostic seven-day setting or obsolete generation-count fields.

Publication, history reads and collection acquire the same permanent per-task flock outside run folders, using the existing no-follow/close-on-exec pattern in `couchidentity/store_unix.go`. Only a finalized bundle containing exactly record.json plus output is eligible; live/incomplete/unknown-execution records stay protected. In M2, pending notice delivery also protects a record; known execution with terminal uncertain delivery follows ordinary retention. Execution fields and output stay immutable, with only the approved pending-to-terminal delivery finalization permitted. Publish output and metadata in a private staging directory, sync and atomically rename it into runs before clearing completed scheduler authority. Retrying that publication checks the same immutable run identity and never repeats a task.

`operationalrun/collection.go` persists a narrow deletion journal with exact source-relative name, directory/member identities and unique quarantine target before moving a bundle. Under the task lock, rename the whole run folder, sync both parents, mark detached, and dispose only its certified contents. Recovery follows the frozen identities; after detachment it never revisits the source name. Unknown members, symlinks or substituted identity block that run. This reuses durablefile publication and the prepared/detached/finalized ordering from `storagegc/transaction_model.go`, without importing session owner semantics or broadening diagnostic generation retention.

Add `cmd/internal/gcruntime/operational.go`: discover existing registered Couch namespaces through `Coordinator.ReadRegistry().Stores`, as `runtime.go:crashRows` does, and call SweepPage. Extend `gcruntime.Report`, Preview and Apply with separate operational rows/bytes, before the session MigrationComplete early return. Extend `gcruntime/schedule.go:Service.Batch` with an operational phase after diagnostics (including its empty-path and last-path exits), persisting namespace/hub/task/run cursor. Use the operational task lock without nesting the Pair root lock. Preserve path-local blocker progress; cancellation/contention retains the cursor. No separate scheduler retention loop, new generic class framework, changed diagnostic policy or session-backup prerequisite.

## Supervisor, notices and status integration

- `startMessageService` owns one scheduler worker and joins it in `messageService.Close`, inside the already-held supervisor lease. Enumerate explicit `HubOwnership.Hubs()` or the existing implicit `hubWorktree`; do not scan `.hub` claims as task roots. Reconcile task removal/replacement through reducer events.
- Use one injected clock/timer stream. Scan declarations at startup and at a bounded 30-second change-observation interval; schedule wakeups at the earlier of the next scan and next due. Scans and builds run off the console reducer. This gives a stated reload bound without a new watcher dependency.
- Resolve the exact current hub via `hubBindingAt`, serialize admission with `holdHubEffect`, and call `Broker.SendSystem` using the current binding and a stable per-occurrence notice ID. Reuse canonical receipt polling, not a second queue or input path. Queue admission is not proof of agent execution.
- Freeze body/output/recipient identity before the first admission attempt. Preserve known non-admission versus uncertain commit. Retry known non-admission only before the persisted deadline. Once accepted, use the existing broker's delivery semantics: context cancellation does not expire queued input, and v1 adds no absolute queue TTL. Keep one pending notice per task; later due slots skip while it is pending. Include the actual due instant in its body so delayed input remains interpretable. Exit-10 retries only known non-admission of the notice, never the script.
- Before `SendSystem`, durably freeze the body, occurrence, target binding, stable ID and an `admission-may-have-happened` marker. The existing broker's system receipts are memory-resident and its retained adoption accepts peer origin only. While the broker lives, observe its canonical receipt; after restart, an unavailable system receipt is terminal delivery uncertainty, **never** a reason to resend or execute the task again. No new system-receipt adoption mechanism is part of v1.
- The inspected Pair source has no generic supervisor-drain hook; do not confuse lifecycle `restart-drained` with one. V1 guarantees cancellation and joining on service shutdown, while heavy admission honors the existing queue pause. It does not add a fleet-drain command or promise that pausing the heavy queue stops light tasks. This limitation is explicit operator-review material; a broader drain integration requires its own reviewed scope if requested.
- Add a typed `ScheduleStatus` response to the existing private socket protocol. Status returns the coordinator's immutable bounded snapshot, refreshed after store/manual-run observations. No subprocess probes, filesystem scans, or task builds on a status request. Keep malformed request/response rejection in the existing validators.

## Complexity and test strategy

Let H be configured hubs, N task entries, B changed declaration bytes, I build-input bytes, and R retained history rows. A catalog pass reads each directory/task declaration once: O(H+N+B); one due pass does at most one recurrence transition per task, not one per missed occurrence. Status copies O(N) current-task summaries; explicit history lists O(R) retained run entries for the selected task. Input hashing/build metadata work is O(I) per admitted build and absent from ordinary clock/status paths. No task count multiplies another full catalog scan.

Count operations, not wall-clock speed: fixtures at N=8 and 16 reject more than two file reads/recurrence evaluations per task per pass; 1,000 missed slots still cause one transition and at most one catch-up effect; same content across two calls causes one build; two hubs cannot exceed the two-worker cap or run the same TaskKey concurrently. Native tests prove timeout/parent EOF/descendant cleanup separately from fake-clock scheduling tests.

| Production function / seam | Adversarial input or event | Mechanical guard/test |
|---|---|---|
| `ParseTaskSpec`, `ParseOperators`, `OwnerAdmission` | Duplicate keys/slugs, missing Operators, invalid owner, foreign current operator, symlink/traversal | Pure fixture table; wrong owner never emits an execution/notice effect. |
| `Calendar.Next`, `Advance` | DST, backward clock, startup late, sleep gap, 1,000 missed slots | Explicit instants/expected next due; effect count bounded independently of elapsed slots. |
| `Advance` | Exit 0/10/other, duplicate/stale completion, config change during build/run, overlapping manual call | Attempt/revision guards; output-10 produces exactly one notice, zero repeat executions. |
| `Catalog` | Untrusted declaration trees and replacement | Bounded no-follow folder fixtures; refusal stays local; N/2N read counts reject repeated scans. |
| `operationalrun.Store.Publish/List/Read` | Concurrent publication/history and partial persistence | Shared task lock, bounded no-follow reads, immutable run identity; faults never expose partial bundles. |
| `operationalrun.SweepPage` | Retention-boundary and crash/replacement uncertainty | Pure age/count classifier and certified bundle detachment/recovery; active/unknown protection and cursor progress through faults. |
| `gcruntime.Service.Batch/Preview/Apply` | Empty diagnostics, blocked run, session migration incomplete | Registered namespaces observe operational phase/report independently of diagnostic/session policy. |
| `StateStore` | Truncation, unknown nested fields, write/rename/sync refusal, concurrent manual completion | Reopen real folder-backed state; unknown bytes survive and no effect precedes durable reservation. |
| `Builder` | Local dependency/embed/module/toolchain changes, dirty input during capture/build, unsupported module inputs, concurrent same key, failed build | Captured manifests and held stateful builder; compile reads only captured sources; unsupported inputs refuse; no stale-key executable or partial publication. |
| `TaskExecutor`, `acquireExecutionPermits` | Paused/unknown/skipped helper outcome, forged inherited bypass, timeout, parent death, lingering descendant, three mixed manual/automatic callers | No pre-admission task code; actual machine lease survives owned cleanup; controller outside killed group; machine-wide permits cap callers across supervisor namespaces at two. |
| `scheduleNotice` | Offline notify-only skip/direct execution/exit-10 drop, timeout-result admission window, busy/replaced hub, exhausted allowance, commit uncertainty, broker restart/expiry | Actual fake-backed broker path; same frozen receipt/body; timeout still reports; no resend on lost system receipt, no second paste/script, no claimed absolute queue expiry. |
| CLI/protocol/status | Cross-hub task reference, argument separator, malformed status, large/controlled output | Exact parser/socket assertions; JSON/text project same snapshot and status performs zero build/probe IO. |

## Implementation tasks

### M1 — One shared manual task path, durable state, and inspectable status

### Task 1: Declaration, ownership, recurrence and transition kernel

**Create:** `cmd/internal/couchschedule/{spec,calendar,model,status}.go` and colocated `_test.go` files. **Modify:** `go.mod`, `go.sum` (pin cron parser), `cmd/internal/artifactpath/manifest.go` for every new production-shaped file.

- [x] Write failing tests for the pure functions using their single strategy rows below; keep event ordering explicit.
- [x] Run `go test ./cmd/internal/couchschedule -run 'Test(Spec|Owner|Calendar|Advance)' -count=1`; demonstrate red before implementation.
- [x] Implement the typed parsers, location-injected calendar wrapper and `Advance`; no IO in the kernel. Use one occurrence reservation plus next-due update, not a missed-slot loop.
- [x] Run the focused tests green; add N/2N counters and stale-event mutation checks. Verify `missed=once` cannot turn a completed script into a retry.
- [x] Register new source files and commit `#737 M1: add scheduled-task transition kernel` with Agent/Co-Authored-By trailers.

### Task 2: Local discovery and preserving, bounded state

**Create:** `cmd/internal/couchschedule/{catalog,store,store_unix,state_json}.go`, `cmd/internal/operationalrun/{record,store,store_unix,retention,collection}.go`, and tests; `cmd/internal/gcruntime/operational.go`. **Modify:** `gcruntime/{runtime,schedule}.go` for the explicit operational report/phase described above. **Reuse:** `cmd/internal/durablefile/write.go`, `cmd/internal/strictjson/object.go`.

- [x] Write failing folder-backed tests for Catalog, StateStore and operationalrun methods using the strategy matrix, through real bounded files and controlled publication faults.
- [x] Run `go test ./cmd/internal/couchschedule -run 'Test(Catalog|Store|StateJSON)' -count=1`; confirm the intended red failures.
- [x] Implement bounded discovery and context-aware store transactions. Keep task refusal local; state-store corruption blocks effects. Publish per-run records and 256 KiB head/tail output; register operational data with shared diagnostic retention (30 days/300 runs per task). Test preview/delete fencing and partial publication recovery.
- [x] Run the focused tests green and the counted catalog-growth check; all fixtures use disposable folders.
- [x] Commit `#737 M1: persist scheduled occurrences and bounded history` with trailers.

### Task 3: Cached Go entrypoint and owned execution/admission

**Create:** `cmd/internal/couchschedule/{build_manifest,build,execute,execute_unix,admission,runtime_fake}.go` and tests; a dedicated native fixture under `cmd/internal/couchschedule/testdata/`. **Modify:** internal command dispatch in `cmd/internal/couchcmd/{cli,run}.go` only as needed for the task-lifetime owner.

- [x] Write failing stateful tests for Builder, TaskExecutor and execution admission using their matrix strategies.
- [x] Run `go test ./cmd/internal/couchschedule -run 'Test(Build|Execute|Admission|Output)' -count=1`; verify red. Pin the specified captured-source build view with fixtures for module replacements, embeds, unsupported inputs and changes after capture; no mtime-only shortcut.
- [x] Implement the captured manifest/build cache and parent-owned subprocess wrapper. Use argv, the verified hub cwd, reviewed operator-level environment, and positive heavy admission. Never watch Couch's PID or accept `WF_TEST_LOCK=off` as acquired.
- [x] Run fake tests green and qualify the runtime seams against native disposable Go/process fixtures using the same matrix guards. Document deliberate process-group escape as outside the supported task contract.
- [x] Commit `#737 M1: own scheduled task builds and process lifetimes` with trailers.

### Task 4: Manual helper and read-only status

**Create:** `cmd/internal/couchcmd/{schedule_cli,schedule_status}.go` and tests. **Modify:** `cmd/internal/couchcmd/{cli,messages,run}.go`, `cmd/internal/couchmessage/protocol.go` and protocol tests; `README.md`, `atlas/couch.md`, `atlas/index.md`, `cmd/internal/artifactpath/manifest.go`.

- [x] Write failing CLI/socket tests for the proposed command forms using the CLI/protocol/status matrix strategy.
- [x] Run `go test ./cmd/internal/couchcmd ./cmd/internal/couchmessage -run 'TestSchedule' -count=1`; confirm red.
- [x] Wire the helper to the shared builder/executor/store and status to the existing private socket. Add team/hub/TL glossary to Pair atlas; leave `tl-lead --help` changes to the Ariadne project item.
- [x] Run targeted tests, source inventory guard, and the required recorded affected/full selection on the pushed head. Verify a native temporary hub task's `--help`, script-run and status without driving the operator's live Couch.
- [ ] Update atlas and issue Log, then run `sdlc milestone-close --issue 737 --milestone M1 --verified '<specific manual/helper/status evidence>'`; resolve gate findings before M2.


### M2 — superseded by the one-writer re-plan (operator, 10-21)

Historical tasks below describe the superseded model. They are not another open implementation/review boundary. The approved replacement is M3 in the final revision; prior M2 findings and REWORK evidence remain intact.

### Task 5: Joined scheduler service and authenticated due notices

**Create:** `cmd/internal/couchcmd/{schedule_service,schedule_notice}.go` and tests. **Modify:** `cmd/internal/couchcmd/message_service.go`, scheduler status composition, source inventory.

- [ ] Write failing fake-clock sequences through actual service composition for Calendar/Advance and scheduleNotice using their matrix strategies.
- [ ] Run `go test ./cmd/internal/couchcmd -run 'TestSchedule' -count=1`; confirm red. Hold executor/receipt observations explicitly; no sleeps to choose event order.
- [ ] Start/join one coordinator on service lifetime; feed pure transitions with catalog/time/process/receipt events. Bound workers and persist reservations before effects. Admit exact system notices through `hubBindingAt`/`holdHubEffect`/`SendSystem` and observe their canonical receipts.
- [ ] Run the complete scheduling packages green; mutation-check the matrix invariants and counted catch-up guard.
- [ ] Commit `#737 M2: schedule each hub and deliver due outcomes` with trailers; update atlas data flow and the issue Log at this boundary.

### Task 6: Integration, operator rollout, and required day-long acceptance

**Modify:** `README.md`, `atlas/couch.md`, `atlas/index.md`, issue Log; add focused native acceptance tests in `cmd/internal/couchcmd/schedule_acceptance_test.go`. Ops task and project records remain owned by their respective workers/TL.

- [ ] Run targeted packages and `git diff --check`; sweep every new production-shaped file against the artifact inventory. Run the repository's declared affected check on the pushed candidate, with the full suite wherever selection or landing requires it.
- [ ] Coordinate an ops code-track issue/worker for the trivial landing-plan task using the approved declaration/entrypoint contract. This is a prerequisite for #737's live Done when, not an excuse to silently waive it. The later ops task can be developed against this reviewed contract without importing an unlanded Pair Go package.
- [ ] Ask ops:0 to arrange `sdlc stage --build`, an authorized Couch restart, and a real trivial ops task scheduled for one day. Do not stage into an occupied :0 or restart the operator's session independently. Record task declaration, staged binary revision, due/start/end observations, bounded logs and status; include a harmless notice/exit-10 and timeout qualification under the same real setup.
- [ ] Keep #737 open if that one-day acceptance has not completed. Any change to acceptance timing/scope is an explicit operator/TL revision, not an agent assumption. Send project outcome evidence to the TL instead of editing the project file.
- [ ] After live evidence and required checks pass, close M2 through `sdlc milestone-close`, then whole-issue `sdlc close` with measured actuals and behavioral evidence. On SHIP run `sdlc merge --yes`; let the landing own current-main integration verification; report the PR outcome and stop.

## Architecture/accountability check

- **ARCH-DRY:** one calendar/transition kernel, owner parser, build cache and executor for both callers; one existing broker/socket and machine admission helper. Existing archive timers are not copied as a second general scheduler.
- **ARCH-PURE:** time, files, Git, Go builds, process ownership and receipts arrive through named integration seams; pure state/status tests need no function-call mocks.
- **ARCH-PURPOSE:** automatic and manual execution, both scopes, skip/once, output/notice semantics, status, recovery and the live day are owned here; ops task code and later timer migrations stay in their project items.
- **ARCH-MOCK:** stateful fake clock/build/process/admission/broker adapters plus portable real store folders; native Go/helper/termination tests qualify the fake model.
- **ARCH-CONSTRAINTS:** minute-resolution schedules, 30s change observation, bounded catalog/history/output/cache, two jobs, declared timeout, operation counts at N/2N. No scans or builds in a console/status hot path.
- **ARCH-SECURE:** reviewed operator-level execution, network allowed, **no sandbox**. Hub-root/path validation is routing and accidental-escape prevention, not privilege isolation. Private outputs/state; no credential/environment dumps; no shell interpolation.
- **ARCH-ORDER:** durable occurrence before first effect; actual lease held through execution; stale events and uncertain outcomes explicit; shutdown is cancel-and-join. No exactly-once promise or automatic retry of possibly completed work.
- **ARCH-FUNERAL:** bounded history removes owned logs, cache has LRU byte budget, removed tasks age out, active/unknown owners are retained rather than pruned while work might survive.
- **ARCH-EVOLVE:** preserving codecs at every state/history nesting; strict known fields and safe refusal on corrupt/unsupported versions; source declarations are never rewritten by Couch.

## References and reviewer decisions

- Source seams: `message_service.go:startMessageService/Close`; `wait_notice.go:hubBindingAt`; `hubs.go:holdHubEffect`; `couchmessage/broker.go:SendSystem`; `durablefile/write.go`; `strictjson/object.go`; `couchcore/runner.go`; Ariadne `scripts/test-lock.py` / `Makefile.workflow`.
- Cron library API/source: [package documentation](https://pkg.go.dev/github.com/robfig/cron/v3), [configured parser](https://github.com/robfig/cron/blob/master/parser.go). Use only the parser/`Next` value logic; its own goroutine scheduler does not own Couch's lifecycle.
- Go build metadata contract was checked through local `go help list`; dependencies and embeds must participate in cache identity. Task 3's fixtures qualify the captured-source build recipe specified above before production code is accepted.
- Operator review should confirm the concrete JSON declaration, explicit script-run opt-in, timeout covering admission/build/run, local-time/DST semantics, manual helper behavior, pure-Go/root-module build envelope, resource caps, existing broker queue semantics and possible delivery uncertainty after restart, shutdown rather than general fleet-drain integration, and the staged day-long live acceptance dependency. Those are implementation decisions proposed here, not additional settled operator requirements.

## Revisions

### 2026-10-20 — initial plan after TL clarifications

Team-owner enforcement is included now despite the earlier project reservation. Literal `main` becomes a cached Go-package entrypoint plus one helper. Original sandbox wording is superseded by explicitly trusted operator-level execution; machine admission applies to declared heavy tasks. No implementation has started.

### 2026-10-20 — one fresh-eyes plan pass

Code-backed review exposed six incomplete contracts. The revised design uses existing broker delivery semantics without inventing an absolute TTL; treats unavailable system receipts after restart as terminal uncertainty; keeps the watched controller outside its work group until cleanup completes; shares lifetime execution permits across manual and automatic callers; specifies a captured-source Go build view; and reserves an eligible owner's offline occurrence before waiting for reconnection. Tests now name each boundary. The absence of a generic fleet-drain hook is an explicit v1 limitation rather than deferred implementation design. No production code or acceptance requirement changed.

### 2026-10-20 — operator approval through ops:0

Approved with three superseding decisions: (1) fall-back repeated local times run once; spring-forward missing times skip. (2) Remove waiting-hub: offline notify-only tasks skip/advance; direct tasks run anyway, dropping offline notices with explicit status/history. (3) Separate per-run records under Couch data schedule/<hub>/<task>/runs, with task/revision/source/start/end/exit/outcome/notice-delivery metadata and 256 KiB head/tail output; operational-data retention reuses diagnostic-log machinery, default 30 days/300 runs per task; add history listing and per-run output. The two execution permits are machine-wide. JSON declaration, one-minute result-notice window and uncertainty/no-replay remain approved. Begin change-code and M1. Earlier revision entries preserve superseded design history.

### 2026-10-20 — plan gate PQ-1/PQ-2

PQ-1: specify the operationalrun store/collector and gcruntime report/bounded-phase integration, independent policy, shared task lock, exact bundle journal and replacement-safe recovery. Existing diagnostic generation counts are not reused. PQ-2: consolidate repeated task test inventories into the function-level strategy matrix; keep verification commands and implementation steps. These refinements implement the approved retention contour without changing acceptance or adding a new fleet mechanism.

### 2026-10-20 — M1 kernel implementation checkpoint

Advance operates on one TaskState per event; StateStore composes these into the ledger. This avoids cloning/scanning the whole task map per due event while preserving the approved single transition owner. A caller may supply its pre-parsed pure Next function; counted tests at 8/16 tasks show one recurrence evaluation per task even after 1,000 missed slots.

### 2026-10-20 — M1 verification checkpoint

Manual execution, immutable operational results, shared retention, captured cached builds, live artifact leases, controller ownership, status/history CLI and the catalogue worker are implemented. Native acceptance reproduced the matching-parent-deadline EOF race and pinned the correction: declared timeout remains timeout, earlier cancellation remains cancellation. Nonblocking FD3 output streams a bounded prefix and retained suffix without delaying group cleanup; uncertainty closes output before the controller keeps its resource leases. Saved manual results recover interrupted authority publication without replay. Automatic due execution and broker notices remain M2. Recorded affected/full selection and the mandatory M1 review are next; M1 is not closed yet.

### 2026-10-20 — M1 review reconciliation and approved M2 notice finalization

M1 review returned REWORK (BR-1 through BR-6); M1 remains open. Reconcile the core-concept inventory against actual declarations and paths, replacing design-only ScheduleState/Occurrence/BuildKey/TaskCatalog/TaskBuilder/TaskAdmission names with implemented owners. Explicitly label automatic notice delivery and its finalization API as planned M2. Fix malformed declaration isolation, interrupted cache mutation recovery, persisted state combinations, executable skill examples and additive manifest metadata before re-review.

TL approved the M2 notice revision after the M1 review: execution facts and output remain immutable. Publish the execution record before notice admission, with pending notice state when delivery remains outstanding. One `FinalizeNotice` operation under the existing task lock atomically replaces only delivery metadata, moving pending to submitted, expired, dropped with a reason, or uncertain. Repeating the same terminal value succeeds idempotently; a different terminal value refuses. Uncertain is terminal and never authorizes a retry. Retention protects pending records; terminal delivery uncertainty does not turn known execution into unknown execution. Preserve additive metadata and output bytes. No new retention loop or notice sidecar is introduced. This supersedes blanket metadata immutability for this one monotonic delivery field; the approved execution and no-replay contracts remain intact.

### 2026-10-20 — M1 cache recovery simplification (TL direction)

Before adding a journal, use the cache's rebuildable content-addressed contract: publish a complete temporary directory by atomic rename; touch the entry mtime without staged metadata replacement; evict by rename to trash followed by deletion. Lookup recovery ignores in-progress temporary/trash names and cleans stale unreferenced leftovers after one minute, respecting existing mutation and lifetime locks and preserving ambiguous replacements. Corrupt entries are rebuildable from captured source. Prove interrupted publication, touch and eviction with abrupt process termination. A journal requires a concrete crash counterexample that this smaller design cannot handle; none is presently established, so do not add one.

### 2026-10-20 — Complete concept-name correction before retrying M1 review

Final read-through found one missed table label: implemented `scheduleCoordinator` was still called scheduleService. Interrupted the owned review attempt before any verdict, corrected that label, and checked every delivered core-table symbol against its listed source files. The interrupted attempt admitted no findings or close result. This is a documentation correction within BR-4, with no code or acceptance change.

### 2026-10-20 — BR-7 complete staging inventory and approved recovery revision

M1 round 2 disposed BR-1 through BR-6, then returned FIX-THEN-SHIP with Important BR-7 still blocking finalization. The named build-capture and ledger-temp gaps were instances of the same durable-artifact-crash-recovery family; a complete writer sweep also found `.collection-stage`. TL approved this whole-inventory revision before code. Rule: every scheduler-owned staging family is covered by an owner-held resource and recovery under its lock, including interruption before its first authoritative publication. No journal, worker or cleanup loop is added. This is the third change to M1 recovery mechanics: if the next review finds another staging gap, stop and bring the complete inventory to the TL rather than patching that instance in place.

| Writer / owned artifact family | Lifetime resource | Recovery trigger, identity fence and bound | Required evidence |
|---|---|---|---|
| `Builder`: captured module metadata, `view/mNNNN/**`, module edits and private compiled `task` | New `capture.go`: `CacheDir + ".captures"` has permanent `allocation.lock`, `slot-0.lock`, `slot-1.lock`; each fixed slot contains one generation-unique `capture-*` tree. Acquire slot lease before creating its tree, hold through build and cleanup. Go commands inherit the lease descriptor through both native command adapters; existing group ownership joins tool work before normal release. | Serialized allocation/recovery tries slot leases nonblocking, skips live slots, and removes only identity-certified no-follow trees. Two slots bound live/abandoned cardinality; a third standalone builder waits with its context. Enforce file/depth/inventory bounds based on 10,000/64 MiB inputs plus rewritten module metadata and private binary. Never reuse a generation pathname. Normal completion and the next allocation recover disposable scratch. | Abrupt death before creation, partial capture/view, module edit, binary/cache completion and partial cleanup; live command retains inherited lease after caller death; held-slot cancellation, replacement/symlink preservation and repeated-crash cardinality. |
| `StateStore.Update → publishSchedule`: `.state-<32 lowercase hex>.tmp` | Existing permanent `.state.lock`, held across creation/write/sync/rename | After locking and validating current authority, bounded exact-name scan; descriptor-relative no-follow opens, regular/private/owned/single-link/size checks, captured identity rechecked immediately before unlink, directory sync. Never promote a temporary snapshot. `View` remains read-only. | Abrupt created/written/synced/pre-rename/post-rename exits; live writer excluded; replacement/unknown/symlink/hardlink preservation; failed cleanup durability prevents mutation. |
| Cache `.publication-*`, `.trash-*`; legacy key `.manifest-*`/`.binary-*`; known partial final entries | Existing maintenance lock, publication/key locks and artifact lifetime leases | Existing rename-only recovery and rebuild protocol; one-minute grace for unreferenced stages, bounded inventory and bytes, identity-fenced unlink; final executable capacity 256 MiB. | Existing fourteen abrupt checkpoints plus zero-byte repair, live artifact/replacement/unsafe shape and young-residue capacity regressions. |
| `operationalrun.Store.Publish`: `.run-stage-*`, including nested `.record-*`/`.output-*` and final staged payload names | Existing permanent task `.runs.lock` | Existing `cleanupStages` before Publish and apply sweeps; exact reserved shapes, no-follow identity checks, bounded task inventory. Read/List never collect. | Abrupt partial publication, recovery, live publisher exclusion and replacement preservation through the production lock path. |
| `operationalrun.saveJournal`: `.collection-stage` | Same task `.runs.lock` | Recover non-authoritative leftover staging even when no `.collection.json` exists; validate regular private bounded identity before unlink and sync. A partial staged journal is never promoted or treated as deletion authority. | Abrupt first-publication/updated-stage exits, live collector exclusion, replacement/symlink preservation and ordinary apply sweep recovery without another collection. |
| Operational `.collection.json` and `.quarantine-*` | Existing task lock plus authoritative prepared/detached/finalized journal | Existing identity-certified recovery; journal precedes detachment, detached recovery never revisits the old source name; finalization removes quarantine then journal. | Existing phase/replacement tests plus native collection interruption qualification. |

Final `state.json`, cache key bundles and run bundles are authoritative/persistent data rather than staging. Their owners remain StateStore, bounded LRU and shared operational retention (30 days/300 completed runs, protected unknown execution), respectively. Coordination inodes are deliberately permanent to prevent lock-identity splitting: one state lock, three capture-pool locks, cache locks within its existing 16,384-name inventory, two machine permits and one zero-byte host task lease per historical TaskKey, plus run-task locks in their structural directories. Task lease growth is per new task identity, never per run; no collector unlinks reused lock names. The inventory/count and regular-file checks distinguish these resources from disposable payloads.

Anonymous control/output pipes and process handles have kernel/controller lifetimes and create no named disk staging. Production schedule CLI/service code delegates all disk publication to the owners above. Go-owned shared module/build caches and ordinary Go work directories remain under Go/operator lifecycle; workflow queue/lease artifacts remain owned by Ariadne's existing test-lock helper. Arbitrary reviewed task writes are task side effects, not scheduler staging. The unlanded global TMPDIR couch-build layout stops being created; old un-attributed folders lack namespace/lease proof and must not be swept by prefix.

Implementation clarification within the approved inventory: uncertain Go-group cleanup transfers the held capture resource to the existing uncertainty owner, rather than closing it at Build return. A standalone caller receives `BuildCleanupError.Join(context.Context)`; no new cleanup worker is introduced. An ignored uncertainty remains fail-closed, retaining its raw lease until explicit join/process exit, including across garbage collection. Cleanup bounds traversal/entry work and validates identities; an oversized known regular binary can be unlinked by metadata without reading its contents. Build-input and executable read limits remain enforced separately.

The same inventory sweep covers normal returned-error cleanup, not only restart recovery. Captured creation/member identities now fence ledger, cache and run-publication defers. Cache/run bundles verify those identities before final rename. The shared `durablefile.publish` callee also verifies the opened file identity before rename and deferred removal, preserving the original write error and any foreign replacement. This changes no publication API, ownership lock or journal state. Tests reproduced deleted replacement paths and substituted published data before the correction. Uncertain capture join observes group completion without signalling a potentially recycled saved group ID; the existing initial owned stop/join remains unchanged.

### 2026-10-20 — Complete BR-7 implementation verification

All staged families in the approved inventory now have implemented recovery and replacement protection, including ordinary error cleanup and pre-rename publication. Captures use generation-unique children, inherited writer leases, a raw retained descriptor across ignored uncertainty/GC, bounded concurrent Join, and context-aware cleanup. Shared publication and immutable run bundles preserve substituted paths instead of deleting or publishing them. The final combined scheduler, operationalrun, durablefile, message, artifact-path, GC runtime and GC CLI packages pass; targeted scheduling/skill CLI tests and actual native test-lock conformance pass. All delivered core table declarations and paths are re-audited before review. M1 still needs the recorded final-head check and its third boundary review; M2 has not begun.


### 2026-10-21 04:17 UTC — BR-8 whole-inventory re-plan, awaiting TL

Round 3 disposed BR-7 but returned REWORK with Critical BR-8, the third durable-artifact-crash-recovery finding. The prior verification claim was too broad: creation identity guards ledger error cleanup, but `publishSchedule` still renames an unchecked temporary pathname. The reviewer replaced it at the existing pre-rename hook with a valid empty ledger; Update reported success and discarded retained execution authority. This supersedes the preceding claim that every publication was protected. No production correction is authorized or made yet: the TL's next-gap instruction requires a whole-inventory decision first.

The revised proof unit is **artifact family × operation**, rather than a family-level replacement checkbox. Every applicable publication, normal-error deletion and restart deletion must independently verify the captured identity under its existing lifetime resource. ARCH-SECURE/ARCH-PURPOSE: the authority used to publish must be the authority used to clean up. The simplest alternative, a lock alone, fails because a pathname can be substituted while the cooperating writer holds that lock, as the pinned-head regression demonstrates. A journal adds durable state without repairing the missing publication ownership check; it is not proposed.

This operation inventory records the present implementation, not a new assertion that the complete rule is certified:

| Family / implementation | Held resource | Authoritative publication | Ordinary cleanup | Crash recovery |
|---|---|---|---|---|
| Ledger `.state-*.tmp`, `couchschedule/store_unix.go` | `.state.lock`, admitted directory FD | **GAP BR-8:** unchecked Renameat after pre-rename hook | `removeLedgerPublicationStage` compares captured dev/inode/mode/uid/link/type before unlink | `cleanupLedgerStages` exact grammar, bounded descriptor-relative scan and identity-checked unlink |
| Cache `.publication-*`, `couchschedule/cache.go` | Maintenance/key locks and artifact leases | `captureCacheDir` compares captured root and exact payload identities before rename | Captured snapshot and guarded unlink | Aged stage/trash recovery under locks, bounded identity checks |
| Cache eviction `.trash-*`, legacy `.manifest-*`/`.binary-*`, partial keys | Same locks/leases | Eviction checks captured source root before rename; legacy stages never promoted | Snapshot-checked deletion | Existing bounded rename-only recovery/rebuild; live leases excluded |
| Build capture pool, `couchschedule/capture.go` | Allocation lock plus inherited slot lease | No authority promotion; cache publication is separate | Captured tree with root/generation guards, bounded descriptor-relative unlink | Same cleanup after acquiring abandoned slot; live inherited leases excluded |
| Run `.run-stage-*`, `operationalrun/store.go` and `staging.go` | `.runs.lock`, retained root/stage FDs | `verifyCapturedRunStage` checks root, stage and exact member identities before rename | Same captured-stage verifier before removal | Reserved-shape bounded scan under task lock with identity checks |
| Inner `.record-*`/`.output-*`, `durablefile/write.go` | Enclosing task publication lock | `verifyOwnedStage` compares opened file identity before rename | `removeOwnedStage` checks same identity | Enclosing unpublished run stage recovery; no independent promotion |
| Collection `.collection-stage`, shared `durablefile` publisher | `.runs.lock` | Shared `verifyOwnedStage` before journal rename | Shared identity-checked cleanup | Orphan-stage recovery even without an authoritative journal |
| Collection `.quarantine-*`, `operationalrun/collection.go` | `.runs.lock` and existing phase journal | `verifyBundle(source, journal identity)` before detachment rename | Journal-identity-certified removal | Prepared/detached/finalized recovery; detached never revisits old source |

Persistent final ledger/cache/run bundles and permanent coordination lock inodes are not temporary families. Anonymous process channels are kernel-owned; Go shared caches and test-lock files retain their external owners; arbitrary task writes are task side effects. The preceding full family inventory retains the cardinality, byte, depth and exclusion bounds; no journal, worker, loop or prefix-based sweep is proposed.

Proposed execution after TL approval: (1) pin BR-8 red with the existing production hook, requiring refusal, byte-identical previous authority and preserved foreign replacement; (2) share the ledger captured-identity predicate across pre-publication and ordinary cleanup, descriptor-relative under the existing lock; (3) audit and qualify each applicable cell above separately, including nil-returning replacement hooks as well as error hooks, retaining abrupt-death and held-live-owner regressions; (4) run targeted/race checks, push and record the final-head check; (5) use the TL-authorized boundary continuation route. The three default review rounds have been used; do not raise the cap or waive findings without explicit direction. M1 stays open and M2 has not begun.


### 2026-10-21 — TL-approved M1 threat model and BR-8 scope

**Threat model:** M1 recovery defends against crashes, abrupt process death, concurrent cooperating Couch processes, and stale or partial leftovers. It does not defend against a same-user process deliberately substituting files inside Couch's private 0700 data folder. That actor can already rewrite the authoritative store directly; an inode check does not establish a security boundary against it. Ownership checks remain useful consistency and recovery checks, not a claim of isolation from the operator's other processes.

TL approved the inexpensive BR-8 consistency correction only: reuse the ledger ownership predicate immediately before Renameat. Pin refusal, unchanged prior authority and preserved replacement with the pre-rename hook; requalify that cell, not the whole matrix. The preceding proposed whole-matrix requalification is superseded. Further findings requiring a hostile same-user file swapper are outside this threat model and must be disputed through `sdlc finding dispute`, citing this revision. Crash-class gaps remain in scope.

TL authorizes exactly one additional M1 boundary round beyond the default three (`WF_BOUNDARY_ROUND_CAP=4` for that invocation only). If that round finds another crash-class gap, stop and escalate rather than patching it in place. No ledger waiver is authorized. M2 remains paused until M1 closes. Cancelled this checkout's pre-correction recorded check through SDLC; other checkout runs were left alone.


### 2026-10-21 — M2 implementation checkpoint (acceptance remains open)

M1 finalized SHIP in authorized round four, with BR-8 disposed and no new findings. Its evidence commit is `3eaef955395b67863f0f8824ba53d587ef8a948c`. M2 now composes the existing reducer/store/executor with an injected clock and authenticated broker adapter. Core inventory above names implemented owners; it does not mark live acceptance complete.

The shared runner executes already-persisted reservations and passes literal `script-run` for scheduled direct execution. Source, ID, revision, due/start/deadline are retained; manual calls still allocate their own reservation. Known execution and prelaunch results publish under the existing state lock before completion authority clears. Cancellation/stale revision produces terminal dropped history; a saved pending record cannot recreate notice authority after cancellation. Unknown execution gets a protected operational record even when the interrupted process never published one, without replay or rewriting unknown facts later.

Notice payload, exact binding, stable ID and admission marker are persisted before effects. Known non-admission waits for a clock-driven one-second retry before its existing deadline; accepted input is observed through canonical broker receipts, including the active-callback fence. Pending delivery is finalized monotonically through the existing task lock and `.run-stage-*` staging family; output is never rewritten. Saved terminal history wins after a failed ledger commit. Completion results remain retained in the coordinator until their durable reconciliation succeeds. Startup store refusals retry at the existing 30-second scan bound, without stopping the scheduler permanently or running effects against an unavailable catalogue.

Counted 8/16-task tests reproduced 30/54 hub binding resolutions before a per-pass hub cache; both now resolve only the two configured hubs. Hub queues are interleaved before rotation; a large hub cannot take both ready execution/admission slots ahead of another. Builds remain outside catalogue/due/status paths. Catalogue scans remain every 30 seconds; completion wakes publish/project without another discovery pass. Frozen binding bytes are cloned at both status projection boundaries.

No live Couch restart, :0 stage, ops task change or project edit has occurred here. The required real one-day task, exit-10/timeout qualification and final recorded checks still precede M2/issue close. Cancelled the obsolete M1 recorded suite through SDLC once M2 changed code; other checkout runs were left alone. A new pushed-head check follows the implementation checkpoint.


M2 worker-budget clarification: two execution jobs and two notice-admission jobs have independent bounded slots, serviced by four joined workers on the same coordinator lifetime. Machine execution permits remain exactly two and manual calls share them. A red held-execution regression showed that sharing two generic effect slots would delay due notices behind long/heavy scripts; independent admission prevents that coupling without adding a queue, journal or recovery worker.


### 2026-10-21 04:56 UTC — M2 transaction inventory re-plan; implementation paused

**Known failing checkpoint, not ready for staging or landing.** A final crash-ordering audit found a new M2 gap before a milestone review: notify-only and skipped occurrences can publish immutable history inside the same ledger transaction that first creates/consumes the occurrence. `BeforePersist` then fails, leaving no persisted occurrence facts but an existing run bundle. A one-second-later retry derives the same occurrence ID and different start/end facts; history refuses it forever as `immutable operational run already exists`. `TestScheduleDueHistorySurvivesFailedOccurrenceCommit` reproduces both online notify-only and offline skip cases. No input was dispatched before the failed commit. This is a sequencing problem under the approved crash/cooperating-process threat model, not a hostile-file substitution.

Stop further production edits and bring this complete **transaction** inventory to the TL before another recovery correction. M1 remains closed; its staging families and ownership checks are unchanged. The earlier M2 checkpoint's final-check wording does not claim readiness: the new red regression blocks it.

| Transaction | Durable facts before external publication/effect | Failure recovery today | Assessment |
|---|---|---|---|
| Manual reservation and script authorization | Existing persisted Active tuple, then owner acknowledgment | Exact tuple guards shared runner; unknown never re-executes | Covered by manual/reserved/native qualification |
| Scheduled script reservation | Current due pass commits Active before workers execute | Result recovery uses exact source/revision/due/start | Covered |
| Script/prelaunch result -> authority completion | Existing Active under state lock; publish result before assigning completion | Saved immutable record is adopted after ledger failure; cancellation finalizes pending as dropped | Covered by result/ledger-fault/stale-worker tests |
| Notify-only occurrence -> completed/pending run record | **First Active reservation is only in the current uncommitted transaction** | Retry recomputes start/end and conflicts with the existing immutable record | **Gap: explicit reservation commit barrier required** |
| Offline/missed/refused due summary -> run record | **Consumed due and Last facts are only in the current uncommitted transaction** | Same conflict; a later declaration revision could also replace the only available revision fact | **Gap: persist passive occurrence facts first** |
| Pending -> terminal notice history -> ledger clear | Existing durable frozen delivery attempt; finalize metadata before clearing | Retain admission completion on failure; adopt saved terminal history after restart; never resend uncertain input | Covered by real broker, retained-result and native finalizer crash tests |
| Catalogue cancellation of pending notice | Existing pending attempt/run under state/task locks | Terminalize history before clearing attempt; same final idempotent | Covered |
| Lost execution owner -> unknown history/authority | Existing persisted Active source/revision/due/start | Publish protected unknown record and retain uncertain execution; no replay | Covered for both manual and scheduled restart |

**Proposed invariant:** every immutable run publication refers to occurrence facts that are already durably present in Active or Last. No transition overwrites the sole retained facts until that occurrence's record is durable. The existing state lock remains the one lifetime resource; there is no journal, staging family, recovery worker or new queue.

Proposed correction for TL review, not yet implemented:

1. Separate due reservation/consumption from outcome publication with an explicit ledger commit barrier. Direct scripts retain the current shared worker path. Notify-only reservations are persisted before computing/publishing completion; if a prior attempt already published a matching record, adopt its immutable facts through the existing recovery transition before any admission.
2. Preserve declaration revision on `RunSummary` as a typed additive fact (legacy missing field tolerated), because passive skipped outcomes have no Active tuple. A skipped/refused passive Last is committed before its run record; the next existing coordinator scan/step repairs a missing record from those exact facts. Before any automatic or manual reservation can overwrite that Last, ensure its record is durable. Catalogue replacement must not erase that revision. No output reconstruction is needed for these no-execution outcomes.
3. Keep the completed native/prelaunch and terminal-notice paths as they are; do not weaken immutable publication or replace a prior run's execution facts to make retries pass. Stable IDs alone cannot repair changed facts, as the red test demonstrates.
4. Qualify before/after the reservation barrier, after run publication/before completion commit, declaration replacement before passive-history repair, manual reservation racing passive-history repair, repeated restart, and existing no-replay/canonical-receipt cases. Run the new red test plus the existing scheduling targets; update the transaction inventory with the measured evidence before requesting staging.

The tradeoff remains the approved conservative one: a crash after reservation but before knowing execution/admission may leave unknown and skip that occurrence; it cannot authorize replay. Passive skips can recover their history from durable Last facts. The required live day and final recorded checks remain open. No M2 boundary round, stage, restart or landing has been attempted.


### 2026-10-21 — TL-approved M2 transaction model

TL approved the preceding whole transaction rule and correction: commit due reservation/consumption before publication; immutable records reference only already durable facts; preserve the sole Active/Last copy until history is durable; add declaration revision to RunSummary. Add one simplification: publication is idempotent per occurrence. An existing record for the same occurrence ID is canonical history and is adopted, without comparing proposed timestamps or rewriting its immutable facts/output. Keep conservative unknown/no replay. If the next review finds another ordering gap, stop and bring it to TL against this rule rather than add a mechanism.

Implementation owners: operationalrun adds `Store.PublishOccurrence` alongside the existing strict primitive, returning the canonical record/output under existing locks and durability checks. The reducer's `ResultRecovered` accepts canonical summary facts for the same ID without timestamp reconstruction. A shared `ensurePassiveRun` guards both automatic and manual replacement of retained passive facts; a due commit barrier precedes outcome publication. Existing schema-1 summaries may omit the new revision field; a missing legacy revision never authorizes invented history facts. No new phase, journal, queue, staging family or recovery worker.


### 2026-10-21 — Approved transaction rule implemented; final verification in progress

The due path now commits reservation/consumption before publishing current history. Both barrier-side regressions were measured red against the pre-correction source through a disposable Go overlay: first-commit failure had already published history, second-commit failure was unreachable, and changed stored timestamps were rejected. The corrected targeted automatic/result/barrier/clock/unknown/proof race selection passes (12.767s). Shared execution reads canonical same-ID history before invoking an executor, adopts the actual publication winner, and preserves passive facts before manual replacement; focused shared runner race checks pass (12.370s, final canonical/passive subset 3.503s). `PublishOccurrence` retains the strict primitive, syncs canonical existing bundles, adopts concurrent winners and completes lost post-rename acknowledgments without rewriting; full operational race checks pass (16.604s).

Unknown classification is committed before its protected history; repeated history outages preserve the already durable unknown observation instead of recomputing its time. Canonical unknown history is adopted with no execution or notice effect. Native Last summaries already prove their preceding result publication and may outlive retained output. Only no-execution scheduled passive summaries need missing-history reconstruction, using their own revision or a matching retained Active revision, never the latest declaration's revision. A bounded process-local canonical terminal proof avoids recreating correctly collected passive history on every wake; proof and retry entries are pruned against current ledger keys. Pending/active authority is never recovered from that cache.

Parent integration tests also preserve an older passive declaration through manual replacement, refuse missing legacy revision rather than invent facts, and retain manual unknown facts across a history outage. The required live day, operator-coordinated staging/restart, final recorded checks and M2/issue boundaries remain open. The next review's ordering gap, if any, returns to TL against the approved transaction rule; no new mechanism is authorized by this checkpoint.

### 2026-10-21 — TL acceptance sequencing

The live day uses ops#1's hourly `scripts/scheduled/testscratch`, after it lands on ops main. pair:0 remains staged for #742 until that lands after #747. Then finish the M2 boundary review and recorded suite, integrate current main, and run `sdlc stage --build`; TL coordinates one operator Couch restart onto that build. No independent restart or early :0 change. The day-long acceptance remains required before issue close. Exit-10/timeout qualification must use a disposable fleet or manual scratch hub, never a synthetic task on ops main. Proposed to TL: after pair#744's harness lands, scratch Go tasks qualify due exit-10 authenticated notice/history and timeout history/process-group cleanup in an isolated native fleet.

TL subsequently approved that exact pair#744 scratch-fleet qualification: one authenticated notice plus terminal history for exit 10; timeout history plus process-group cleanup for the deadline case. Nothing is added to ops main; the live day remains testscratch.

### 2026-10-21 — M2 first review; approved admission fence and complete consumer sweeps

M2 review returned REWORK on BR-9/10/11. The earlier transaction inventory's cancellation Covered claim was incomplete: a worker validated authority before waiting for the hub gate, then admitted after committed cancellation or the notice deadline. This is M2's second ordering correction. TL approved the following rule and conditions; a third ordering correction stops work for a whole concurrency-model re-plan.

**BR-9:** admission must retain authority until the actual broker boundary. The one lock order is **hub-effect gate, then scheduler authority**. Acquire the gate before final tuple/deadline validation, then hold the existing authority lock through bounded SendSystem admission. The persisted notice deadline bounds the complete gate wait and admission; never renew that budget. Catalogue cancellation uses scheduler authority and cannot clear the tuple while admission owns it. Waiting admission holds no authority lock. Once broker admission happened or is uncertain, preserve canonical receipt authority rather than declaring known non-admission. No new lock, queue, journal or recovery mechanism. Sweep scheduleNotice admission, worker authorization, catalogue cancellation and ownership-move gate acquirers for reverse acquisition; add an executable lock-order regression and verify a held gate cannot indefinitely block unrelated authority users.

| Admission ordering | Required outcome |
|---|---|
| Worker queued; cancellation commits before gate acquisition | After gate acquisition, tuple validation refuses; no SendSystem |
| Worker holds/waits gate; persisted deadline elapses | Deadline cancels all wait/admission work; no later fresh admission |
| Admission owns gate and authority before cancellation | Bounded broker boundary finishes first; cancellation reconciles actual receipt/uncertainty |
| Broker accepts; metadata reconciliation fails or process dies | Canonical receipt/history recovery; no replay or conflicting terminal overwrite |
| Gate wait overlaps status/catalogue/other-task authority work | Waiting worker owns no authority; healthy operations proceed |

**BR-10:** a task-history failure quarantines that task visibly and preserves its authority/facts; shared authority-store failure blocks globally. The full consumer inventory is startup recovery, due/passive repair, shared manual replacement/completion, automatic completion reconciliation, catalogue cancellation finalization, notice finalization, and unknown repair. Sweep each consumer, not only startup. Tests pair a damaged task with a healthy second hub and exercise startup and subsequent dispatch/reconciliation; status must name the quarantined task and refusal, history must not be invented, and no uncertain execution is replayed. Context cancellation and shared ledger access/publication errors retain global propagation.

**BR-11:** preserve unknown fields on every same-identity rewrite. The summary schema and operational-record schema are distinct namespaces. Canonical known facts win, but existing Last.Unknown survives recovery of that same run ID; do not copy operational Record.Unknown into summary fields. Sweep reducer active/passive recovery, manual/scheduled/uncertain adoption, completion and notice rewrites. Reopen-and-rewrite tests with additive nested JSON qualify the real codec/consumer path; cloning must avoid map/raw-byte aliases. A different occurrence does not inherit the previous occurrence's unknown members.

Qualify each regression red before correction and green after, then combined scheduling/core/operational packages, diff check and fresh pushed-head recorded check. Gate owns the next review. Real acceptance remains open, and the TL owns stage/restart/day coordination.

### 2026-10-21 — Review corrections implemented and qualified

BR-9's real-broker held-gate regressions failed before correction for both deadline and committed cancellation (the broker admitted both invalidated notices). Final authorization now runs through `StateStore.WithAuthority` only after `scheduleNotice.Admit` owns the hub-effect gate, sharing the authority lifetime with SendSystem. One joined clock timer bounds the complete admission attempt by its persisted deadline, including a held hub gate and a blocked broker reservation. Permanent tests prove gate waiters leave scheduler authority available, cancellation wins before final validation, expired gate waits finish without gate release, and broker deadline cancellation releases authority for other users. Accepted/uncertain results keep existing receipt semantics. Audited the three production hub-gate acquirers (`hubs.go`, lifecycle notices, schedule notices); only schedule notices acquire scheduler authority, after the gate. TL adds the combined #739 total order: **hub-effect gate -> scheduler authority -> broker -> automation**, with automation held only for in-memory commits. No send callback reacquires the hub gate; whichever issue lands second tests the combination.

BR-10 uses one named-task quarantine helper at every history consumer. Seven baseline-overlay consumer probes failed before correction; startup/passive/completion/notice/cancellation/unknown and manual/evidence retry regressions now pass. Task-local history failure preserves the original authority/facts and returns typed local failure only when existing pending worker evidence must remain for its one-second retry; healthy dispatch continues and those completed workers no longer occupy job capacity. Shared ledger failure and context cancellation remain global. A repaired startup-quarantined task reconciles its saved evidence through the existing scan, without replay or another queue.

BR-11 centrally preserves summary unknown fields on all same-ID reducer rewrites and detaches nested attempt/summary/status metadata. Full codec/rewrite matrix covers manual, scheduled, passive, uncertain, completion, notice and cancellation paths; different occurrences inherit no old summary metadata. Real manual/automatic consumer overlay probes failed on metadata loss before correction and now preserve summary fields without importing operational-record unknown fields. Coordinator memory snapshots also clone those fields.

Final combined `Test(Schedule|Skill|HelpAndSkill|MessageSkill)` race selection passes (13.474s); full couchschedule/operationalrun packages pass (18.298s/11.169s). An earlier combined run caught a new fixture mutating message authority after service startup; construction now injects that authority before goroutines start. Recorded final-head check and M2 second review follow commit/push; live exit-10/timeout and one-day acceptance remain open.

### 2026-10-21 — BR-12 bounded recovery-producer correction approved

Second M2 review disposed BR-9/10/11 and raised BR-12: cached passive proof recovery omits Event.At, so the reducer ignores canonical history after a failed ledger commit. TL explicitly classifies this as a missing required producer field within the existing model, not the third concurrency trigger. No lock/interleaving/model change is approved or needed. Direct and cached scheduled adoption must use one shared constructor/transition path. All production recovery producers (scheduled direct, scheduled cached, manual canonical, catalogue manual recovery, unknown repair) use a recovery-event constructor requiring the observation time as a positional argument. Failed-commit retries must assert reopened ledger-to-history equality, not only history immutability: due/start/end, ID/revision/source, exit, outcome and notice; preserved summary metadata remains in its own namespace. Qualify direct/cache equivalence and retained output/no replay, then recorded final-head check and the remaining M2 review round.

Implemented `NewRecoveryEvent` for all production producers and shared `adoptScheduledSummary` for record/cache paths, with all canonical known facts checked after the transition. The three-case failed-commit regression now compares reopened ledger facts, direct/cache equivalence, immutable record/output and unchanged due/no effects. Restoring the missing-At cached producer with a disposable overlay fails all three cases (0.501s); current full scheduling/skill/help/message-skill race selection passes (22.953s), and full couchschedule passes (45.565s). Test fixture comparisons normalize absent versus empty unknown maps while retaining all known-fact checks. Diff check clean; commit/push, fresh recorded check and M2 third review follow. No stage/live acceptance claimed.

### 2026-10-21 — Third M2 ordering gap: whole concurrency/evidence re-plan, code stopped

Third M2 review disposed BR-12 and raised BR-13. A run publication can rename its bundle and then fail a required directory sync. The bundle is readable, but startup recovery, reserved execution and catalogue manual recovery currently clear authority using `Store.Read`; notify-only adoption also bypasses confirmation. The review's fault probes fail in those three native consumers. This is a real missing history-before-authority fence, not another omitted event argument. Per TL's explicit third-ordering-correction instruction, **stop production edits and bring this whole model for decision**. All three default M2 review rounds are used; any further boundary needs an authorized continuation, never a cap change by the worker.

#### One authority model across all concurrent owners

The scheduler owns **occurrence facts** in Active/Last. Native execution owns **whether an external script may have run**, through the retained process identity and lifetime leases. The broker owns **whether a frozen notice was accepted/delivered**, through its canonical receipt. Operational history owns **durable completed execution/output and monotonic notice facts**. Readability is an observation of history, not durable completion evidence. An authority transition may consume history only after the task-locked boundary has read/validated it and completed required directory syncs. Failed confirmation leaves prior occurrence authority/facts, visible task quarantine, pending evidence and no-replay policy intact. A later successful confirmation permits exactly the same canonical adoption, with no regenerated facts.

| Resource | Owner/lifetime | Consumers/order | What it proves |
|---|---|---|---|
| Hub-effect gate | Ownership mutation or one bounded notice admission | Acquire before scheduler authority; native lifecycle/ownership callers never acquire scheduler authority | Exact hub ownership cannot move across notice admission |
| Scheduler authority lock | One ledger transaction or bounded broker admission | Hub gate -> authority -> broker -> automation; history operations under authority take task-history lock, release it before any broker call | Occurrence facts/cancellation and admission serialize |
| Permanent task-history lock | Bundle publication, durable adoption, finalization or collection | Authority -> task-history; GC takes task-history alone and never takes scheduler authority | One canonical bundle; durability confirmation before authority consumption |
| Broker mutex / automation mutex | Native broker admission and brief in-memory automation commit | Hub gate -> authority -> broker -> automation; no earlier resource acquired while held | Canonical accepted/uncertain receipt and current automation policy |
| Task execution lifetime / two machine permits / optional heavy lock | Existing controller and inherited leases | Acquired before native owner acknowledgment; no authority callback waits to acquire them; dispatch occurs after authority commit | No overlap/replay and bounded machine execution capacity |
| Build/cache leases | Existing M1 captured-input builder and controller | Outside due/status/notice paths; M1 recovery model remains closed | Reproducible held executable, independent of completion authority |
| Bounded passive proof / pending worker result | Existing coordinator lifetime, pruned by current ledger identities | Proof populated only after confirmed durable adoption; pending facts retained until history+ledger reconciliation succeeds | A prior completed durability proof, never new execution/notice authority |

No new resource, queue, journal, worker, retry phase or election is proposed. Existing resource lifetimes already suffice; the missing consumer contract must be made explicit and shared.

#### Complete history-to-authority consumer inventory

| Consumer | Current evidence | Required boundary before its authority action |
|---|---|---|
| Automatic startup and later execution reconciliation (`recoverExecutionHistory`) | Direct `Store.Read` | Confirm existing canonical bundle under task lock before adopting/clearing Active; sync failure quarantines only task |
| Reserved manual/scheduled runner pre-execution existing-result check | `runReader.Read` | Durable existing-result adoption before skipping executor and clearing retained authority; absent remains absent, damage never reconstructs output |
| Catalogue manual-owner recovery | Direct `Store.Read` | Same durable existing-result boundary before replacing Last/clearing Active |
| Notify-only reserved result adoption | Direct `Store.Read` | Same durable boundary before completion/admission authority |
| Passive summary repair and proof cache | Read then `PublishOccurrence` | Already confirms durability, but converge on same shared adoption boundary; cache only its successful proof |
| Unknown repair | `ensureRetainedRun` then `PublishOccurrence` | Already confirms existing/new history; preserve unknown and no replay on failure |
| Newly published native/prelaunch/notify completion | `PublishOccurrence` result | Already durable; consume canonical returned facts, never fall back to mere Read after an error |
| Pending-to-terminal finalization and retry | Task-locked `FinalizeNotice`; same terminal repeats sync | Initial bundle must be durably adopted before finalization; terminal metadata must be confirmed before clearing ledger. Inventory run, runs-parent and task sync obligations together |
| Quarantine repair before allowing a task again | Passive helper or direct active `Read` | Durable boundary before lifting quarantine or later consuming that evidence |
| CLI history/status/output inspection | Read / published snapshot | No authority action: retain read-only APIs and do not add sync or recreation here |

#### Recovery outcomes and scope of possible loss

| Reachable state/failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Bundle renamed, directory sync fails, Active retained | Named task refusal; other hubs continue | Inspect/repair storage; existing scan retries durable confirmation | That occurrence may remain unknown; never replay script |
| Bundle confirmation succeeds, ledger commit fails | Prior authority plus immutable canonical history | Existing reconciliation adopts exactly that durable record | No regenerated execution/output facts |
| Broker accepted before cancellation/finalization failure | Pending/uncertain canonical receipt; immutable output | Observe existing receipt and retry monotonic finalization | Delivery may be uncertain; no duplicate send |
| History collected after an earlier confirmed terminal passive proof | Completed retained summary | Existing bounded proof, no resurrection on every wake | Retained output may be gone by policy; known completion remains |
| Shared ledger unavailable | Scheduler-wide blocked status | Repair shared authority store | No new effects until authority is available |

#### Decision proposal for TL

Prefer one explicit **durable existing-record read** alongside `PublishOccurrence`, reusing its existing `readOccurrence` task-lock/read/sync implementation and refusing absence rather than creating a record. Every authority consumer uses that boundary; inspection keeps `Read`. Publication and monotonic finalization continue to return only after their required syncs. Test publishers must model durability or refuse adoption, never inherit a false guarantee from a read-only interface. A single table-driven regression injects failure after rename and during repeated durable confirmation through every consumer above, asserting unchanged prior authority, named quarantine, no replay and healthy second-hub progress; then repair sync and assert reopened ledger/history equality. Include finalizer idempotent retry, canonical concurrent winner and cached proof reuse.

Alternatives: changing all `Store.Read` calls to sync would obscure the authority boundary and give inspection unwanted write obligations; adding a journal/worker would duplicate already held resources without solving consumer misuse. Neither is proposed. This is a **proposal awaiting TL approval**, not implemented. Keep current queued suite as evidence for its exact pre-correction head unless code is authorized; no staging or landing.

### 2026-10-21 — Whole M2 model approved; one fresh full-model review authorized

TL approved the entire preceding resource/evidence/consumer model. Implement the durable existing-record read by reusing PublishOccurrence's task lock, read and sync boundary; keep inspection Read-only, route every adoption/quarantine release through durable evidence, never create absent native history, and qualify all finalization sync obligations. No new lock, journal, worker or queue. TL authorizes exactly one fresh **full-model M2** review after implementation, using this inventory as its checklist (fourth M2 invocation only, not a delta review). Another history/authority ordering REWORK stops work and goes to the operator through TL, with no patch loop. Cancelled exact stale lease `lease-767412298` through `sdlc local-check cancel --run`; no other checkout run was touched. This supersedes the earlier permission to leave that pre-fix run queued.

### 2026-10-21 — Whole-model durable boundary implemented; integration checks running

`Store.ReadDurable` now reuses the same task-lock/read/sync boundary as `PublishOccurrence`, refusing absent or damaged history rather than creating/repairing it. Inspection `Read` remains unchanged. Publication, durable existing-record adoption and fresh/idempotent notice finalization share a fixed six-directory confirmation: **run -> runs -> task -> hub -> schedule -> namespace**, never its parent. This includes freshly created hub/task links, not just the run rename. Errors from every required sync propagate. Payload/stage atomic-write sync and captured-descriptor cleanup sync retain their existing ownership boundaries. Finalization validates both immutable payloads before fresh or repeated success, including output-size validation.

The coordinator owns one native Store value, carried through automatic, catalogue, passive/unknown, quarantine and finalization consumers. Manual native work carries the same store/hook; reserved adoption requires explicit ReadDurable capability, and custom Read-only publishers refuse visibly. The passive path consumes the canonical return from its existing durable helper rather than repeating a second six-fsync read; bounded cached proofs still require an earlier successful confirmation. Unknown diagnostic history may still be published from committed unknown facts under the already-approved separate repair rule; existing-record reads never invent completed native output.

All 13 consumer cases first failed with a real renamed/readable bundle whose repeated directory sync failed: startup, completion, catalogue manual, reserved script/manual/notify, passive, manual replacement, notice preparation/finalization, catalogue cancellation, unknown repair and quarantine release. The corrected matrix preserves exact reopened Active/Last and named local quarantine on repeated failure, keeps the second hub dispatching, then asserts repaired ledger/history equality and unchanged output with no executor replay. The Read-only publisher negative control also fails without its capability guard. Store tests fault each of six sync paths across fresh/idempotent publication and finalization and existing-record adoption, plus absence, damage, locks, cancellation, additive metadata and oversized finalizer output. Final operationalrun package passed (38.926s); final scheduling race selection passed in the bounded consumer task (42.485s). Parent combined scheduling/skill/help and core/GC checks remain in progress; no recorded pass, stage or live acceptance is claimed.

Parent final combined verification passed: scheduling/skill/help/message-skill race selection 42.489s; full operationalrun 48.791s, couchschedule 30.685s, artifactpath 10.525s, gcruntime 19.565s, gccmd 1.755s. Final code and stronger repair assertions are included. Diff whitespace check clean. Commit/push and recorded final-head check precede the TL-authorized fourth **full-model** M2 review. A new history/authority ordering REWORK goes to TL/operator without another implementation loop.

### 2026-10-21 — Authorized full-model review REWORK; operator escalation, work stopped

The one TL-authorized fourth M2 full-model review disposed BR-13 and retained BR-9/10/11/12 dispositions, but raised Critical BR-14. Its production-path probe confirms this sequence for both declaration removal and revision: receipt polling commits terminal `submitted` history; the following authority write fails; catalogue change attempts a different terminal cancellation finalization before adopting the confirmed terminal record. FinalizeNotice correctly refuses the conflict, but Active stays delivery and Last stays pending through three refresh/step cycles; quarantine then prevents the reconciliation branch from consuming the terminal record. The existing resource/durable-read fences therefore do not yet establish convergence for competing cancellation/finalization events.

This qualifies the previous recovery inventory: durable evidence is required and now confirmed, but **canonical terminal evidence must precede a competing cancellation/replacement transition** as well. Readable-versus-durable correction is verified; it is not proof that all already-terminal history is reconciled before catalogue transitions. The unqualified earlier claim that existing reconciliation always adopts durable history is superseded for failed-commit followed by removal/revision/owner change or competing finalization. The reviewer calls for that complete sequence family, not one cancellation-site patch.

Per TL's explicit final-review condition, **no production patch, further boundary invocation, round-cap change, stage or restart is authorized here**. Stop and let TL take the new ordering gap to the operator. Review ledger/sidecar and issue Log are checkpointed on the branch. Current code candidate is `c999468484f4832f68db1d3ac41d639a3fd66e5b`; its full-suite fallback remains queued under `lease-1820886732` (session 30501; test-stats inflight c3ce620724bf54909d7d876d73edebf7), not passed. No day-long live acceptance or disposable exit-10/timeout acceptance has occurred. Further work waits for the operator/TL decision.

### 2026-10-21 — Operator selects A: replace M2's competing writers with one owner

**Decision and approval boundary.** Authenticated TL delivery `8935cb14-0bb2-47d2-a8db-5325ca4023aa` relays operator approval of option A. This revision replaces M2 Tasks 5–6's writer/concurrency model and the transaction-per-caller portions of subsequent revisions. M1's calendar, executor, cache, operational-data format/retention and crash threat model remain the starting point; its state-lock lifetime and manual integration are deliberately requalified below. This is the concrete re-plan **for TL approval before code**, not a statement that it is implemented. The superseded candidate is not acceptance evidence for A. Cancelled its exact owned suite `lease-1820886732` through SDLC; cancellation returned successfully. No new recorded suite, review, stage or restart is being started during planning.

#### Core concepts for the replacement

| Pure entity | Lives in | Status |
|---|---|---|
| `TaskState`, `Event`, `Advance` | `cmd/internal/couchschedule/model.go` | modified: one transition kernel for all callers; completion/canonical evidence precedes edits |
| `Settlement` and `PlanSettlement` | `cmd/internal/couchschedule/settlement.go` | new: tagged classification and deterministic next action for retained authority plus observed evidence |
| `ManualIntent` | `cmd/internal/couchschedule/manual.go` | new: run ID, configured hub/task, bounded argv and canonical intent digest; independent of RPC connection |

One namespace has one owner; one task has at most one retained occurrence; each occurrence has one native run bundle and at most one canonical notice receipt. `Settlement` describes stable, executing, awaiting-receipt, settling, quarantined, or unknown state, with explicit required payloads rather than independent flags. It does not create another durable phase log. `PlanSettlement` consumes typed facts and returns a transition/effect decision; it performs no IO. Canonical history and the receipt are evidence about the retained occurrence, never permission to replay execution. These concepts remove separate manual, scheduled, cached and catalogue recovery decisions (ARCH-DRY, ARCH-PURE, ARCH-ORDER).

| Integration | Lives in | Status | Wraps |
|---|---|---|---|
| `StateOwner` | `cmd/internal/couchschedule/{store,store_unix}.go` | modified/new owned API | existing permanent `.state.lock`, descriptor-relative publisher and ledger recovery |
| `scheduleOwner` | `cmd/internal/couchcmd/schedule_owner.go` | new; replaces distributed orchestration | existing scheduler service loop, durable store, history and bounded effect workers |
| Owner transport | `cmd/internal/couchcmd/{schedule_rpc,schedule_manual,message_service}.go`, `cmd/internal/couchmessage/{protocol,transport}.go` | modified/new | existing private service socket; typed submit/observe/cancel manual operations |
| Fact-only execution adapter | `cmd/internal/couchcmd/schedule_execution.go`, `cmd/internal/couchschedule/{execute,execute_unix}.go` | modified/extracted | existing builder, TaskExecutor, controller identity and process-group lifecycle |
| Gate-ready notice adapter | `cmd/internal/couchcmd/schedule_notice.go` | modified | existing hub-effect gate, broker and current automation predicate |
| Discovery/status | `cmd/internal/couchcmd/schedule_service.go` | modified | bounded catalogue facts and immutable atomic status snapshots |

Use the existing fake clock, `FakeTaskRuntime`, real temporary stores, stateful broker/serviceRig, and disposable native process fixtures behind these seams. No new scheduler database, journal, durable command queue, election, retention worker or sandbox. Growth axes remain task count, run history and bounded active jobs; adding policies later extends the pure events, not a second writer.

#### What collapses; what remains

| Existing producer/path | Replacement |
|---|---|
| `scheduleCoordinator.refresh`, manual-owner recovery and catalogue cancellation | Discover facts, then send one catalogue event to owner; owner settles each affected occurrence before applying its edit |
| Automatic `reserveDue`, `recover`, `step`, `applyAdmission`, `reconcileExecution` | One owner loop advances due state, admits effects and consumes results through the same settlement function |
| `manualTaskRunner.run`, owner-ack callback, completion, prelaunch refusal, unknown/adoption | CLI submits intent; execution adapter reports owner identity, prelaunch facts and final result; only owner writes |
| Passive/unknown/quarantine helpers and `passiveDurable` cache | One canonical-evidence settlement path; remove independent authority decisions and passive proof cache |
| `pending`, `inFlight`, `jobKinds`, retry bookkeeping | One bounded typed operation slot per active task; keep necessary outstanding facts until settled, without parallel authority flags |
| Worker `StateStore.Update` / `WithAuthority` callbacks | No writer access in workers; callbacks submit typed facts and await an owner decision |
| Status and CLI history inspection | Remain read-only snapshots / native `Read`; never acquire owner or history publication authority |

Retain operationalrun's permanent task lock (shared with retention), atomic metadata/bundle publication and six-directory durable confirmation; builder/cache/capture leases; task execution exclusion and two machine permits; optional heavy-test lock; hub-effect gate, broker and automation locks. Their resources have different lifetimes and are not replaced by owner serialization. Scheduler publication/finalization is owner-only; GC remains the existing task-locked collector. No history writer is introduced in transport handlers, workers or diagnostic inspection.

#### Namespace ownership and M1 recovery requalification

Acquire the existing permanent `.state.lock` once at scheduler startup with a bounded/context-cancellable attempt; retain its descriptor and flock for the scheduler's entire lifetime. A second cooperating process reports scheduler-owned/busy without discovering, repairing, publishing or launching. Do not unlink, rename, time out, or steal the lock. Mark its descriptor close-on-exec: a long script cannot inherit namespace ownership. Startup with an unavailable owner leaves read-only status visibly blocked; unrelated message service functions remain available.

Refactor StateStore into a read-only view plus a non-copyable owner handle that alone exposes transition persistence. Its held descriptor performs existing read/validate, ledger-stage cleanup, encode, ownership check, atomic rename and directory-sync operations without recursively reacquiring the flock. Close joins all worker/callback lifetimes before releasing ownership. Calls after Close refuse. Remove unowned production update/admission APIs; migrate test fixtures to explicit owner lifetimes. The owner handle is passed only to the owner loop, never workers or service request handlers. Add a source/AST dependency guard covering every ledger write and scheduling `PublishOccurrence`/`FinalizeNotice` call, not only conventions in comments.

Requalify M1's full ledger-stage family because the held resource lifetime changes: abrupt death at created/written/synced/pre-rename/post-rename boundaries; partial/stale stage cleanup only after exclusive owner acquisition; live owner prevents cleanup; cancelled contender preserves owner authority; ordinary returned errors preserve replacements; BR-8 pre-rename ownership predicate still refuses replacement, keeps previous authority unchanged and preserves replacement. Include owner shutdown/reacquisition and descriptor-not-inherited tests. Cache/capture and operationalrun ownership mechanisms are unchanged; retain their existing regressions without claiming a fresh full-M1 close. Same-user deliberate substitution remains outside the approved threat model; keep the existing cheap predicate anyway.

The owner-lifetime flock is a startup exclusivity fence, not a short transition mutex acquired during admission. Gate acquisition never waits while executing the owner loop, and no owner action reacquires this flock. The owner slot serializes runtime transitions. Document that distinction to avoid an apparent gate -> state-lock inversion hiding an actual wait cycle.

#### Settlement before catalogue edits (the BR-14 class)

An occurrence is unsettled from durable reservation through native completion, canonical history confirmation, receipt terminalization and matching ledger commit, or until it is explicitly retained unknown/quarantined. Catalogue removal, revision, owner-policy changes and replacement are applied only after the prior occurrence's known effects have settled. While executing/awaiting-receipt, record a visible **catalogue change pending: task busy** projection and retain the old occurrence identity. Do not mutate its declaration authority or request conflicting finalization. The next ordinary discovery scan observes current files again; retain at most the latest bounded observation for display, with no persisted edit queue. Stop new occurrences for a task with a pending edit. Other tasks continue.

File edits cannot be prevented. Before script release, revalidate exact declaration/owner facts outside owner and return them with the reservation identity; owner rejects changed/stale evidence rather than authorizing old code. Once running, a script keeps its reserved declaration and may finish under its original deadline. This changes catalogue edits from cancelling a running occurrence to becoming effective after settlement. Explicit manual cancellation and supervisor shutdown still stop their exact controller/process group and settle its result. Hub ownership and automation changes remain immediate external authorization checks; they do not require a catalogue edit.

| State/fact + incoming event | Owner action and ordering |
|---|---|
| Stable + due/manual intent | Preserve any sole prior facts; commit reservation/consumption before dispatch; never start after failed commit |
| Reserved + owner-ready facts | Validate current tuple, declaration and deadline; commit controller/start authority before acknowledging script release |
| Executing + result | Retain bounded facts; derive completion once; publish/adopt canonical history durably; commit matching ledger before releasing occurrence |
| Awaiting receipt + terminal receipt | Finalize history monotonically; confirm canonical terminal record; commit matching ledger; no competing finalizer |
| Any occurrence + already-terminal native history | Durable existing-record read; canonical terminal facts win before cancellation, edit or replacement |
| History succeeded + ledger failure | Keep settling slot and previous authority; retry canonical adoption; block edits/new runs for that task; never repeat script/send |
| Executing/awaiting-receipt/settling + catalogue change | Busy for this task; settle existing occurrence first; rediscover current declaration afterward |
| Damaged/unsyncable history + any edit/run | Preserve exact authority and facts, named quarantine; existing scan may confirm repair; no fabricated native output |
| No history + retained possibly-started attempt after restart | Persist conservative unknown, no replay; optional unknown diagnostic record uses only committed unknown facts |
| Duplicate/late result | Match occurrence/revision/source/phase; adopt canonical same occurrence or ignore stale facts; never grant new authority |
| Shared ledger access/sync failure | Scheduler globally blocked; retain facts; no new external effects until reopened durable authority is established |

The owner attempts a task's settlement before processing its edit. A failed attempt is not rolled back across already-completed IO. For post-rename ledger sync failure, re-open/confirm the durable ledger boundary before treating either old or new state as authoritative; no acknowledgement or external start relies on the in-memory candidate. On restart the same settlement function reconstructs from ledger plus durable history/receipt evidence; missing native completion remains unknown. No special recovery writer, cache bypass or catalogue-only finalizer remains.

#### Notice admission and deadlock prevention

Workers acquire the hub-effect gate **outside** the owner, bounded by the persisted one-minute notice deadline. With the gate held they submit a gate-ready admission command. The owner checks the exact current occurrence tuple/body/binding/deadline, then invokes the bounded broker admission while no other owner transition can change that authority. The broker checks current hub/automation policy. The owner never waits to acquire the hub gate or awaits a worker result while holding an outstanding transition. Gate waiting workers own no scheduler transition slot.

Wait order is **hub-effect gate -> owner admission turn -> broker -> automation**; history lock is used in separate settlement steps and released before broker calls. The namespace flock stays held independently and has no runtime contenders on an owner path. A gate-ready handoff has explicit completion acknowledgement: cancellation cannot release the gate while its owner command may still call SendSystem. Cancelled/expired queued commands are rejected before send; an executing command is joined through its bounded result before gate release. Preserve accepted/uncertain receipt identity regardless of deadline expiry. No closure may call back into the owner or reacquire the gate. Qualify this against real ownership mutation and #739's broker -> automation order; whichever lands second wires and verifies the combined predicate. Explicit manual execution is independent of the automatic toggle; scheduled starts and all wake notices consult the service-owned current predicate.

#### Manual service transport and lifetime

Replace status-query-then-direct-file-writes with typed `schedule-run`, `schedule-run-status` and `schedule-run-cancel` operations on the existing private local socket. Keep `--schedule run <task> -- <args>` and task execution exit semantics. Server validates the configured canonical hub root, task name, argument count/bytes, request ID and per-operation closed shape; client-supplied roots cannot broaden configured hubs. Reuse the existing operator-private namespace/socket permission boundary (same operator who can invoke the current CLI); no new credentials or requirement to run inside an agent slot. This does not grant access outside that namespace. Existing strict protocol validation must reject schedule fields on other operations.

Client allocates a run ID once before submission. Owner persists that ID and a digest of hub/task/argv with the reservation; retain an optional additive intent digest with Active, Last and manual run history so duplicate retained IDs adopt/query the same occurrence and differing intents refuse. No accepted acknowledgement before durable reservation. A missing or timed-out submit reply is **unknown admission**, never permission to submit again. Print run ID and query only that ID; explicit new user invocation creates a new intent. Duplicate detection is bounded by existing authority/history retention, not an indefinite request registry; there is no automatic resubmission after retention. Pre-reservation transport cancellation refuses execution; after acceptance the run belongs to the service and survives a dropped client connection until its deadline or explicit cancellation.

Status polling is a bounded read of the exact run ID; no poll authorizes repair or launch. On normal completion the CLI prints the retained capped output and returns the recorded exit. **Proposed small UX delta for TL approval:** output is printed when the run completes, rather than streamed byte-for-byte from a subprocess owned by the CLI. Fetch output in bounded chunks (for example 16 KiB raw per response) below the existing JSON frame bound, total at most the existing approximately 256 KiB head/tail cap; this avoids a second streaming protocol or raising MaxFrameBytes. `history` still prints retained output after disconnect. Ctrl-C submits cancellation for that exact ID with a bounded cleanup context, reports uncertain cancellation if its acknowledgement is lost, and never locally rewrites authority. Cancellation is idempotent; owner waits for native process-group cleanup facts before recording a definitive stopped result. Service shutdown cancels/joins workers; if completion cannot be confirmed, preserve unknown and no replay.

#### Actions, visibility and possible loss

Actions remain inspect status/history, run explicitly, cancel that manual invocation, edit repository declaration, and repair storage; no retry-execution or new repair CLI is added.

| Internal state/failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Running/receipt unsettled + files edited | Task busy; catalogue change pending | Inspect; edit takes effect after settlement/current rediscovery | Effective edit delayed until deadline/settlement; script may finish old revision |
| Canonical history committed, ledger pending | Settling; edits/new run busy | Inspect; bounded normal scan retries durable adoption | No known facts rewritten; completion display delayed |
| Native history damaged or sync failing | Named task quarantine | Repair storage then inspect normal reconciliation | Output may be unrecoverable; execution never replayed |
| Shared ledger unavailable | Scheduler blocked | Repair shared storage | New scheduled runs skip when their windows pass |
| Submit/cancel reply lost | Unknown request outcome with exact run ID | Inspect that ID; cancel explicitly if still active | Live output/ack may be lost; no duplicate launch |
| Owner dies with possibly-running native work | Unknown retained occurrence | Inspect history/status; controller cleanup as M1 | Unconfirmed output/result can be lost; no replay |
| Other process owns namespace | Scheduler busy/owned elsewhere | Inspect owning service; stop it normally if needed | No second writer or owner stealing |

Expected operating envelope remains at most 256 tasks, two machine script permits and two notice jobs, bounded worker/result slots and approximately 256 KiB output per run. Reuse the current four-worker budget; discovery facts can be produced by the existing bounded discovery call (no state writes), with its context checked before owner admission. No subprocess build/execution or permit wait runs in the owner. Status reads stay atomic and independent of owner IO. Bounded transport admission reports busy when command capacity is full; it never accumulates callers as an unbounded queue. Owner broker admission may occupy its turn only until the persisted deadline; test independent progress after that deadline. Filesystem IO has context checks and surfaced failures, not a claimed hard latency bound on a stalled kernel.

One settlement pass visits each task once plus the bounded current result set; no directory/history scan per worker callback. Reuse the catalogue's bounded enumeration. At 8 and 16 tasks count ledger traversals, canonical reads and hub-binding lookups; reject quadratic growth and repeated fsync of unchanged passive records. Remove passive-proof caching only with a rule that normal settled ticks do not reread old history: canonical read is required at an actual authority/settlement boundary, not every status/timer observation. Operation slots are released after matching durable settlement; quarantined retained facts are bounded by task count and output cap. Manual result lookup uses existing retained records, not a second result store. No new durable artifact family; existing 30-day/count retention remains, and pending notices stay protected (ARCH-CONSTRAINTS, ARCH-FUNERAL).

#### Implementation tasks after TL approves this revision

These are tasks inside the existing M2 boundary, **not additional milestone tags or independently approved code slices**. Each task follows red -> minimal implementation -> green -> checkpoint. No implementation begins from this draft.

1. **Owner capability and M1 state-lock recovery.** Modify `cmd/internal/couchschedule/{store,store_unix}.go`; add `owner_test.go` and update store/abrupt-death fixtures. First fail concurrent-owner, stale-handle and inherited-descriptor probes; run `go test ./cmd/internal/couchschedule -run 'Test(Owner|Ledger|Store)' -race -count=1`. Implement the lifetime handle and unchanged publisher under it; run the entire M1 ledger matrix green. Commit with Agent/Co-Authored-By trailers.
2. **Pure settlement model and BR-14 sequence.** Add `cmd/internal/couchschedule/{settlement,settlement_test,manual,manual_test}.go`; modify `model.go`, `state_json.go` and their preservation tests as required. First fail terminal-history -> ledger-write failure -> removal/revision/owner-change, including repeated failure, already-terminal conflict, duplicate/out-of-order facts, and unknown restart. Run `go test ./cmd/internal/couchschedule -run 'Test(Settlement|Advance|Manual|Ledger)' -count=1`. Implement one transition enumeration; verify canonical equality and unchanged immutable payloads/additive members. Commit.
3. **One production owner and fact-only workers.** Create `cmd/internal/couchcmd/{schedule_owner,schedule_execution}.go` with tests; replace writer paths in `{schedule_service,schedule_automatic,schedule_manual,schedule_records}.go`; compose in `message_service.go`. First fail production BR-14 probes and write-ownership guard, then implement all paths in the inventory above. Run `go test ./cmd/internal/couchcmd -run 'TestSchedule(Owner|Settlement|Durable|History|Unknown|Canonical|Result|Execution)' -race -count=1`; require unhealthy-task/healthy-hub progress and no script replay. Delete bypass helpers/caches only after their tests exercise the owner. Commit.
4. **Manual transport through the same owner.** Add `cmd/internal/couchcmd/{schedule_rpc,schedule_rpc_test}.go`; update `{schedule_cli,schedule_status,schedule_manual,message_service}.go`, `cmd/internal/couchmessage/{protocol,transport}.go` and tests; additive intent fields/codec tests in couchschedule/operationalrun. First fail CLI transport lost-ack, duplicate ID, conflicting intent, malformed/foreign-root/oversized frames, disconnect, Ctrl-C and restart sequences using actual socket plus stateful executor. Implement submit/observe/cancel and bounded output pages. Run `go test ./cmd/internal/couchcmd ./cmd/internal/couchmessage -run 'Test(Schedule|Protocol|Transport)' -race -count=1`; confirm no direct CLI writes. Commit.
5. **Gate-ready admission and joined shutdown.** Modify `cmd/internal/couchcmd/{schedule_notice,schedule_owner,schedule_admission_test,schedule_clock_test}.go`. Red-test gate hold -> owner progress, cancellation/deadline before turn, cancellation during send, current hub/automation change, and gate not released before callback completion. Run `go test ./cmd/internal/couchcmd -run 'TestSchedule(Admission|Owner|Clock)' -race -count=1`. Implement bounded handoff, preserving receipt identity and lock order; verify shutdown joins every worker before releasing owner. Commit.
6. **Whole inventory qualification and documentation.** Update `README.md`, `atlas/couch.md`, `atlas/index.md`, source/artifact inventory and issue Log; TL alone updates project. Run full couchschedule/operationalrun/durablefile/artifactpath/GC packages and broad scheduling/skill/help/message-skill race selection, plus counted 8/16-task tests and `git diff --check`. Fault every history and ledger write/sync boundary, then restart and compare reopened ledger/history; canonical terminal precedence must hold for all edit producers and all finalization consumers. Commit/push final candidate, then a fresh recorded affected/full check through SDLC. Superseded candidate passes cannot qualify changed code.

The qualification checklist for the full replacement is: BR-14 removal/revision/owner changes; repeated persistence failure plus busy edits; duplicate/late worker and CLI facts; owner crash/restart/no replay; one namespace owner across two processes; complete M1 ledger-stage abrupt-death/live-owner/replacement matrix; hub gate/current authority/broker/automation order; unhealthy-task isolation and deadline-bounded healthy progress; additive metadata preservation; bounded operation/output/transport growth; native controller/group cleanup. Tests use production paths with controllable barriers, not timing sleeps. Mutation-check exclusion, canonical precedence and gate-lifetime assertions.

**Review/acceptance remain distinct.** The previous M2 review route exhausted its explicitly authorized fourth invocation; this approval to draft A does not reset gate history or authorize another round-cap override. Ask TL for the supported fresh full-model review route after plan approval and implementation; retain BR-14 open until independently verified at that boundary. Do not modify gate configuration or erase findings. Existing disposable-fleet exit-10/timeout qualification, one real ops#1 testscratch day, TL-coordinated stage/restart, final suite and issue/landing gates still apply. No ops-main fake task or unilateral live restart. No code or further boundary invocation until TL approves this concrete re-plan.

**Architecture checks.** ARCH-DRY/PURE: one settlement kernel and owner; scripts/discovery are fact producers. ARCH-PURPOSE/ORDER: every writer and competing terminal/edit sequence is in the inventory, including manual IPC and restart. ARCH-MOCK: reuse stateful runtime/broker/clock and native disposable fixtures on the same seams. ARCH-CONSTRAINTS/FUNERAL: existing task/job/output/history limits, bounded busy admission, no durable intent queue. ARCH-SECURE: private same-operator socket and configured-root validation; strict persisted/protocol decoding, no secret creation and no sandbox claim. ARCH-EVOLVE: optional intent fields only; preserve unknown fields at every nested rewrite; distinguish intent identity from canonical execution facts and do not reinterpret old records missing the new digest as matching arbitrary new intent.

### 2026-10-21 — Approved replacement becomes M3, not another M2 round

Authenticated TL delivery `29bde47e-ecea-4daf-bf0f-0278b7fa6145` approves the complete option-A revision and printing bounded manual output on completion, paged under the existing frame bound. The operator selected replacement of the concurrency model; TL explicitly directs a genuinely new milestone. **M2 is superseded by the one-writer re-plan (operator, 10-21)**. Its review cap/history/findings remain intact, with no SHIP claim, erasure or override. This supersedes the preceding draft's request for a further M2 review route: M3 has its own fresh boundary, reviewing the complete replacement and its inherited recovery obligations.

### M3 — One writer, settlement before edits, service manual runs and required acceptance

- [ ] Implement tasks 1–6 in the preceding option-A revision as M3, using their full concept/resource/consumer and qualification inventories. Commit subjects identify `#737 M3`.
- [ ] Requalify every affected M1 ledger recovery cell under owner-lifetime `.state.lock`; retain the independent cache/history/process protections.
- [ ] Prove BR-14's full terminal-history/failed-ledger/edit sequence class and every competing producer through the one production owner. Prior M2 fixes are inherited obligations, not automatic M3 proof.
- [ ] Complete targeted/race/native checks, documentation and fresh recorded affected/full check on the pushed M3 candidate; run `sdlc milestone-close --issue 737 --milestone M3` for its fresh full-model boundary, without an M2 round-cap override. Retain unresolved inherited findings for this review to assess; do not edit gate verdicts.
- [ ] Complete disposable-fleet exit-10/timeout qualification and the real ops#1 testscratch day with TL-coordinated staging/restart. Milestone review may precede the day under the already-approved sequencing; whole-issue close/landing still require acceptance.

Proceed through `sdlc change-code --issue 737 --worktree=no` before production edits. No estimate is guessed to fit this re-scope: if the gate requires an updated estimate, derive it after the new plan-quality result. Project changes remain TL-owned.

### 2026-10-21 — M3 implementation checkpoint; remaining sweep is explicit

Change-code accepted M3 (plan-quality CLEAN; estimate-quality INFO). Derived the replacement allowance after that result and set the revised planned total through the card setter. Owner capability and affected M1 recovery, pure settlement, additive intent identity, manual transport, fact-only execution, owner manual methods, bounded joined event loop/gate handoff and production service composition are implemented in focused files. The issue Log records exact red/green evidence. This is still an implementation checkpoint, not M3 SHIP.

The counted due-tick regression confirmed quadratic full-ledger encoding (160 rows at 8 tasks; 576 at 16). Due/startup reservation/recovery now use batched owner transactions, and capacity limits bound per-turn notice preparation: due proof is 56/112 rows with seven publications at either size. This enforces the already-approved operating envelope; it introduces no durable phase or concurrency mechanism. Capacity retains unadmitted reserved tasks instead of claiming unsent delivery. Before claiming the complete growth inventory qualified, also batch/count terminal receipt sweeps and retained quarantine/unknown recovery; preserve canonical history precedence and reservation-before-publication at every batch fault boundary. Pending catalogue edits continue to block new old-revision reservations after their previous occurrence settles until rediscovery applies the edit.

Remaining integration: production source/capability guard and retirement of legacy writers; full callback/fairness/automation/GC and crash qualification; atlas/help/artifact inventory; final-head recorded check and M3 fresh boundary. Required disposable exit-10/timeout and real testscratch day remain open. No M2 review reset, staged restart or landing has occurred.
