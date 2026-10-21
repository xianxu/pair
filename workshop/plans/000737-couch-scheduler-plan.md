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
| `TaskKey`, `Declaration`, `TaskState`, `Attempt`, `RunSummary`, `Event`, `Effect`, `Advance` | `cmd/internal/couchschedule/model.go` | delivered M1 | Per-task transition authority; attempts represent occurrences, with scheduled due instants or manual nonces. |
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
| `Builder`, `Artifact` | `cmd/internal/couchschedule/build.go`; cache in `cache.go` | delivered M1 | Captured Go builds, cache publication, lifetime leases and retention. |
| `buildScratch`, `BuildCleanupError` | `cmd/internal/couchschedule/capture.go`, `execute_unix.go` | delivered M1 | Two capture slots and held command-writer leases; explicit synchronous join for uncertain cleanup, reused by the execution owner. |
| `TaskExecutor`, `Executor`, `ExecutionRequest`, `ExecutionResult` | `cmd/internal/couchschedule/execute.go`, `execute_unix.go` | delivered M1 | Controller/process ownership, timeout and bounded output. |
| `acquireExecutionPermits`, `executionLease` | `cmd/internal/couchschedule/admission.go` | delivered M1 | Per-task lease and two machine permits; declared heavy uses existing test-lock helper. |
| `FakeTaskRuntime` | `cmd/internal/couchschedule/runtime_fake.go` | delivered M1 | Stateful execution double with held starts, exits and cancellation. |
| `Store.Publish`, `Store.List`, `Store.Read`, `Store.SweepPage` | `cmd/internal/operationalrun/store.go`, `retention.go` | delivered M1 | Immutable execution bundles and shared operational retention. |
| `scheduleCoordinator` | `cmd/internal/couchcmd/schedule_service.go` | delivered M1 catalogue/status; automatic effects planned M2 | Joined service lifetime, declaration refresh, recovery and published snapshots. |
| `manualTaskRunner` | `cmd/internal/couchcmd/schedule_manual.go` | delivered M1 | Durable manual reservation, shared execution and run publication. |
| `scheduleNotice` | `cmd/internal/couchcmd/schedule_notice.go` | planned M2, not implemented | Exact hub binding, broker admission and canonical receipt observation. |
| `parseScheduleCLI`, `runScheduleCLI` | `cmd/internal/couchcmd/schedule_cli.go`, `schedule_status.go` | delivered M1 | Closed CLI grammar, status exchange, manual run and retained history. |
| `Store.FinalizeNotice` | `cmd/internal/operationalrun/store.go` | planned M2, not implemented | Monotonic pending delivery finalization under the existing task lock. |

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


### M2 — Automatic per-hub scheduling, delivery, recovery, and live acceptance

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
