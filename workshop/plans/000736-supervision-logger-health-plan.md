---
collaboration-mode: discussion
---

# Supervision logger health implementation plan

> **For agentic workers:** Follow AGENTS.md §3. The SDLC binary owns review boundaries; use the executing-plans skill for the stateful owner work.

**Goal:** Make a frozen supervision stream diagnosable, then repair the measured cause.

**Architecture:** The existing supervisionFactLog worker owns all sink IO and its
outcomes. A pure lifecycle and bounded health value project those outcomes through
the existing resources request; reading health never opens or probes a file.
No retry, new worker, queue, registry or log family is added.

**Tech stack:** Go, existing io.WriteCloser/factSink seam, Couch resources socket.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| factLogLifecycle | cmd/internal/couchcmd/supervision_facts.go | modified |
| SupervisionLogHealth | cmd/internal/couchcore/resource.go | new |
| ResourceView | cmd/internal/couchcore/resource.go | modified |

The lifecycle adds a failed-write state and successful-write event. Opening,
working, unavailable-open, failed-write and closed are distinct; closed absorbs
all late events. Unavailable-open never silently reopens. Failed-write remains
eligible for already-existing later independent offers, never replay of a failed
row. One complete successful write restores working. The health value contains
only state, offered/written/dropped counters, bounded last backend error and its
operation/time, and last successful write time. No row, message, slot binding,
prompt or source prose enters the health value. The last error remains historical
after recovery; current state identifies whether logging currently works.
Pure lifecycle tests enumerate all states/events including close before open or
write completion. No persisted vocabulary or diagnostic format changes.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| supervisionFactLog | cmd/internal/couchcmd/supervision_factlog.go | modified | existing asynchronous sink and owner mutex |
| resources handler | cmd/internal/couchcmd/message_service.go | modified | existing request/response socket |
| resource projection/render | cmd/internal/couchcmd/resources_projection.go, resources_render.go | modified | bounded response and terminal output |

Reuse factSink for controlled block/fail/succeed outcomes. Add short-write mode
and attempt counting to that stateful fake as needed. The production sink is
unbuffered diagnosticlog.Open; it returns metadata/filesystem errors, never fact
content. Error text is sanitized and limited to 512 UTF-8 bytes at capture, before
wire encoding; do not concatenate input rows into errors. Capture startup entropy
failures too. Zero/nil owner projects unavailable diagnostics, never healthy empty
data. Existing resources wire projection retains the bounded health even if detail
rows are truncated; it must run regardless of hosted-ingest availability.

## Visible states and actions

Actions: inspect resources again; continue; report the shown error to the hub;
ask the hub to restart Couch. There is no worker retry control.

| Internal state/failure | Visible state | One action | Possible loss |
|---|---|---|---|
| Sink open unresolved | opening | inspect resources again | bounded queue may drop facts |
| Complete write acknowledged | working with counters/last write | continue | earlier dropped facts stay lost |
| Open/entropy failed | unavailable with bounded error | report the shown error to the hub | all facts this lifetime may be missing |
| Error or short write | failed-write with bounded error | report the shown error to the hub | failed fact not retried; a partial row may exist |
| Owner closed, even with late completion | closed | ask the hub to restart Couch | queued/unflushed facts may be missing |
| Older server has no health field | health unavailable | ask the hub to restart Couch | historical stream health unknown |

Offered count is the existing admitted sequence; written counts complete sink
acknowledgments only, not durable fsync claims. Dropped retains its existing
semantics, and queued/inflight rows must not be called lost. Closing does not
retroactively assert a complete queue drain. No admission outcome depends on any
health field. Reading a snapshot while sink IO blocks must still return.

## Operating envelope and principles

- ARCH-DRY: one health owner; resources samples it directly, not a second file watcher.
- ARCH-PURE: lifecycle transition tests are pure; the existing worker records IO outcomes.
- ARCH-PURPOSE: M1 is instrumentation only; M2 still owes the observed cause and live writes.
- ARCH-MOCK: the existing factSink holds/rejects writes deterministically; real socket/CLI test exercises publication.
- ARCH-CONSTRAINTS: O(1) bounded updates per offer/write and health read; max256 queued rows remains. Error capture capped512 bytes. Count16/32 independent events and repeated reads to prove no extra sink attempts or scans.
- ARCH-SECURE: no payload enters health; terminal controls removed before JSON/text projection. Test distinctive prompt text through normal fact generation and assert absence in health.
- ARCH-STATE: all lifecycle changes use advance; sink worker remains lexically owned and joined by Close. No time grants recovery authority.
- ARCH-FUNERAL: no new durable artifacts; health dies with the existing owner. Tests own/release blocked sinks and temporary sockets.
- ARCH-EVOLVE: additive optional in-memory wire field only; no persisted decoder or rewrite changes. Old servers render unknown, and old clients ignore the additive value.

## M1 — instrument and land

1. Add failing lifecycle/health tests in supervision_factlog_test.go (or a focused
   supervision_health_test.go) for open held/failure, sink held, write failure,
   short write, later independent success and closed absorbing late completion.
   Assert one open attempt, one attempt per offered row, no automatic replay.
2. Run targeted tests and retain the semantic red before implementation.
3. Implement the bounded health value, capture and memory-only snapshot at the
   existing mutex; add success transition. Entropy/open/write errors remain local.
4. Test/render health through resources_cli_test.go and resources_projection_test.go:
   text and JSON, older server nil field, unchanged sampler cache, held sink,
   no hosted-ingest prerequisite, sanitized bounded error, admission unaffected.
5. Run go test ./cmd/internal/couchcmd -run 'Test(Supervision|Resources)' -count=1;
   broaden to the affected packages required by the actual diff. Update README
   and atlas for health inspection, factual JSONL location, retained receipts and
   unavailable old-server behavior. Run make test-affected-inventory and explicitly
   audit/register any new source reader, never bulk-refresh hashes.
6. Commit/push, open PR, run recorded affected check (full fallback remains #738),
   then milestone-close M1 with measured actual, verified evidence and --no-project
   (TL owns project). Review may overlap the queue per tl-worker; final pass before
   merge. Land via sdlc merge --yes and report Git-verified commit to hub.
7. Wait for the hub's coordinated restart and exact resources observation. No
   staging or mutation of running Couch. M1 changes no retry/admission policy.

## M2 — observed cause repair

This boundary deliberately waits for live evidence. It is not permission to
invent a root cause. Record the hub's actual health state/error and fresh file
metadata, then append a cause-specific Revision and get hub approval before code.
Create a private reproducible failing fixture at that storage/lifecycle boundary,
fix only the proven cause, and retain no-retry/optional-admission invariants.
Run targeted and recorded verification, close/review, land on a fresh branch
from main (the M1 PR branch cannot be reused), and have the hub verify new facts
reach live storage. Whole-issue close must retain both milestones' evidence;
M1 alone does not satisfy the issue's live-cause Done when.

## Acceptance ownership

Hub approval: instrumentation first, no stage; planned coordinated restart after
#742 and pair:0 rebuild. The hub reads resources and supplies actual failure.
The worker owns deterministic tests and review/landing. The operator/hub owns
the running supervisor and any restart. No historical-cause claim before M2.
