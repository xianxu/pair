---
id: 000736
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours: 1.7225
collaboration-mode: collaboration
card_mirror: '783dc86f9b48ee4c63c8736929dfe737fe6161c3' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T20:02:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# Couch supervision logger health and frozen fact stream

## Problem

During pair#735 investigation, Couch's supervision facts at ~/.local/share/pair/couch/supervision.jsonl were confirmed frozen since 2026-10-19 15:56:04 (26,061,670 bytes). Archived diagnostic segments do exist; there is no replacement per-message readiness-decision log. Existing facts record slot snapshots and completed hub deliveries.

The logger discards its Open error and becomes permanently unavailable for that supervisor lifetime. Write errors increment a dropped counter without exposing the error. Current metadata matches the frozen file. Scratch-copy Open/Write probes pass, both with writer registrations removed and retained, so the historical failure cannot be attributed from disk. Live writer PID/start matches metadata. No live files were mutated during investigation.

## Spec

Expose logger health and a bounded last error through an existing Couch status surface first. Observe the actual live refusal, then fix that cause. Do not add speculative retries. Keep diagnostics optional, bounded, and off the interactive admission path. Per-message holds remain visible in retained receipts; document the logging and receipt locations accurately.

## Done when

- Existing status reports distinguish a working, unavailable, and failed-write supervision logger, with a useful bounded error and no message/prompt secrets.
- Tests cover startup Open failure, subsequent write failure/recovery, and closed state without changing admission outcomes.
- The live failure is observed and recorded, its root cause is corrected, and new supervision facts are verified to reach storage.

## Plan

- [ ] M1 — Expose memory-only logger health in resources; test, review and land instrumentation without staging.
- [ ] M2 — Observe the actual refusal after the hub's coordinated restart; approve a cause-specific revision, reproduce it, fix it and verify fresh live facts.

Durable plan: [implementation plan](../plans/000736-supervision-logger-health-plan.md).

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against
baseline-v3.1.md, Method A. Calibration is flagged stale by estimate-source.
Existing log worker, sink and resources projection eliminate greenfield library
work. M1 uses two smaller-go-module extensions at v2 impl0.5 each ×0.4, design0.2
each ×0.2 for the approved detailed plan. Issue authoring uses design0.5 ×0.2,
impl0.1 ×0.4; docs design0.1 ×0.2 and impl0.1 ×0.4. Two actual milestone reviews
use impl0.5 ×0.4 each. M2 provisionally budgets one real-API discovery at0.6 ×0.4,
and one smaller extension at design0.3 ×0.5, impl0.5 ×0.4. This is explicitly an
uncertain M2 allowance, revised after the real error is known, not a claim that
its cause is understood. Familiar stack1.0 and thorough-plan design buffer15%.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.10 impl=0.04
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: atlas-docs design=0.02 impl=0.04
item: milestone-review design=0 impl=0.20
item: milestone-review design=0 impl=0.20
item: real-api-discovery design=0 impl=0.24
item: smaller-go-module design=0.15 impl=0.20
design-buffer: 0.15
total: 1.7225
```

## Log

### 2026-10-20

- Split from pair#735 by ops:0 approval. Start after #735 lands. Investigation scratch probe used a Go overlay in /Volumes/TestScratch/pair735-log-probe-75gnrui4; ephemeral evidence only, not required for future reproduction.

### 2026-10-20 — health-first investigation

Read-only stat confirms supervision.jsonl still has 26,061,670 bytes and the
2026-10-19 15:56:04.739652 timestamp. Metadata names the running Couch PID10525;
resources identifies its start as 2026-10-20 10:32:08.391981. No live storage was
changed. Source confirms one unbuffered diagnosticlog.Open on the existing fact
worker; Open errors disappear, and factLogWriteFailed is a lifecycle no-op.
A bounded read-only subagent review confirmed resources as the existing fresh,
request-time status seam. Historical cause remains unknown.

### 2026-10-20 — M1 implementation and targeted evidence

Entry gate cleared plan-quality (PQ-1/PQ-2 addressed) and estimate-quality INFO.
The two smaller-module estimate items include owner/lifecycle qualification and
resource socket/projection qualification respectively; their implementation
allowance also includes active check/landing handling. Review items cover the
two review/PR boundaries, docs include inventory audit and usage, and M2 discovery
is explicitly an analogy for live storage/lifecycle diagnosis. Hub-owned restart
time is not worker implementation. M2's small-repair allowance must be rederived
once the cause is measured.

Semantic reds: a failed sink write remained active; actual resources JSON omitted
supervision_log; resource cloning aliased the new pointer; and a worker stopped
by parent cancellation still reported working. All are green with one health
owner, no retry, no failed-row replay and memory-only publication. Open failures,
short writes, later independent recovery, held IO, absorbing close and16/32-event
counts run against the stateful sink. The real socket/CLI, actual strict decoder
and frame-budget/clone tests pin the approved coordinated upgrade contract.

go test -race ./cmd/internal/couchcmd -run 'Test(Supervision|Resources)|FuzzSupervisionHealthErrorBound'
-count=1 passed3.358s. Targeted couchcore resources and couchmessage protocol
checks passed3.898s/0.171s. Bounded2-worker error fuzz passed2062 executions
(4.509s including shutdown); error capture remains sanitized, nonempty, valid
UTF-8 and at most512 bytes. make test-affected-inventory and git diff --check
passed. README and atlas explain health versus retained delivery receipts and
the historical cause still owed. Recorded affected verification and M1 boundary
review follow on the pushed head; nothing has been staged or accepted live.

## Revisions

### 2026-10-20 — separately land health before measured repair

The hub approved two landing boundaries: M1 health instrumentation lands through
normal review and merge, no pair:0 stage. At its coordinated restart after #742,
the hub reads couch --resources and supplies the observed failure. M2's precise
cause-specific implementation is revised only after that evidence, never guessed.
M1 does not close the issue or claim the historical logging failure is repaired.
The hub owns rebuilding pair:0, the restart and the project file.

### 2026-10-20 — plan-quality compatibility correction

PQ-1 found the existing strict response decoder rejects additive fields for old
clients. The plan now explicitly requires the rebuilt CLI for resources on the
new supervisor, fulfilled by the coordinated rebuild/restart; no negotiation or
relaxed decoder is added. New-client/old-server absence remains truthful unknown.
PQ-2's named-function strategies replace prose case inventories. No code exists
yet; both findings are returned to the plan gate for disposition.
