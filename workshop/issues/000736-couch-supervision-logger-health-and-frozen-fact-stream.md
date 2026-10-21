---
id: 000736
status: codecomplete
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-21
estimate_hours: 1.7225
collaboration-mode: collaboration
card_mirror: 'bc2e5e25afcd8e96b2738eeb2b1df1511e82c73a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T20:02:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 2.53
---

# Couch supervision logger health and frozen fact stream

## Problem

During pair#735 investigation, Couch's supervision facts at ~/.local/share/pair/couch/supervision.jsonl were confirmed frozen since 2026-10-19 15:56:04 (26,061,670 bytes). Archived diagnostic segments do exist; there is no replacement per-message readiness-decision log. Existing facts record slot snapshots and completed hub deliveries.

The logger discards its Open error and becomes permanently unavailable for that supervisor lifetime. Write errors increment a dropped counter without exposing the error. Current metadata matches the frozen file. Scratch-copy Open/Write probes pass, both with writer registrations removed and retained, so the historical failure cannot be attributed from disk. Live writer PID/start matches metadata. No live files were mutated during investigation.

## Spec

Expose logger health and a bounded last error through an existing Couch status surface first. Observe the actual live refusal and hand the measured repair to #774, as approved after the one-branch gate refused continuing this landed issue. Do not add speculative retries. Keep diagnostics optional, bounded, and off the interactive admission path. Per-message holds remain visible in retained receipts; document the logging and receipt locations accurately.

## Done when

- Existing status reports distinguish a working, unavailable, and failed-write supervision logger, with a useful bounded error and no message/prompt secrets.
- Tests cover startup Open failure, subsequent write failure/recovery, and closed state without changing admission outcomes.
- The live failure is observed and recorded through the landed health surface; its measured cause and approved recovery direction are preserved in the independent #774. Repair and fresh-live-write verification belong to that issue.

## Plan

- [x] M1 — Expose memory-only logger health in resources; test, review and land instrumentation without staging.

The former M2 repair boundary is superseded by #774 per the 2026-10-21 Revision; no uncompleted implementation milestone remains in this health-only issue.

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


- 2026-10-21: closed — Health-only closure per hub scope revision: M1 landed in PR502; 2026-10-21 live resources exposed unavailable open, offered77/written0/dropped77, exact reservation error and timestamp. Measured device-only mismatch is preserved in #774, which now owns repair and fresh-write acceptance; no historical-fix claim. Docs-only branch. Retained own recorded M1 suite 5090197a passed730.851s and landed-M1 suite d123e71b passed661.660s; original targeted race/protocol/fuzz evidence and SHIP remain. Hub owns project.; review verdict: SHIP
### 2026-10-20
- 2026-10-20: closed M1 — M1 instrumentation only: semantic reds for invisible failure, missing resource field, clone alias and canceled-worker health; targeted supervision/resources race tests pass3.358s, resource/protocol tests pass, bounded2-worker error fuzz2062 executions passes, fast reader inventory and diff checks pass. README/atlas map memory-only health, legacy strict-reader refusal, rebuilt CLI requirement and retained receipt reasons. Recorded affected attempt5090197a is queued on pushed61cc29d9 (known #738 full fallback), review overlaps per tl-worker; pass still required before landing. No stage; hub owns project and coordinated restart; live refusal/cause repair remains M2, no historical-fix claim.; review verdict: SHIP

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

### 2026-10-21 — health accepted live and cause handed off

The operator restarted Couch at08:44:33 on main53a9d42e with M1. Hub's actual resources readout: supervision log unavailable, offered77/written0/dropped77, last write never, last error open08:44:34.8 diagnostic reservation identity or size changed. This is the M1 acceptance: the previously silent startup failure is now visible with counters, cause and operation/time. No logging-repair claim.

Read-only stable bracketed metadata showed payload device16777229 versus checkpoint16777232, with identical inode728006625, size26061670 and mtime_ns1792450564739652454. The reservation was an older committed append (before26055958 plus5712=current26061670), not missing payload. #774 carries this measurement, the source trace, approved device-only recovery direction, adversarial tests and live-write acceptance. No live logger files/processes were changed.

Retained recorded evidence: own M1 suite attempt5090197a974c2c52291cabd5828204b8 on61cc29d9 passed730.851s, log hash317f0f10249a0a3c739c657804df4323c796b476441b17945f8a862d0cf5cca4. Landed M1 merge4a078392 has recorded suite attemptd123e71bcc85f045d6cd44a44eee8334 passed661.660s, log hash17c1eb7e57cabf6a5015b0ddc9ae1341e6d720144ed905b3ea92bc33ab3263ae. Only issue/plan scope handoff prose changes here; source remains main's tested code.

### 2026-10-21 — closure scope revision approved by hub

Newer start-plan refuses continuing an issue whose M1 branch already landed in PR502, explicitly directing a fresh closure branch and a separate cause issue. Hub approved following that gate: close #736 around landed and live-verified M1 health instrumentation, with no claim the logger is repaired; #774 (no dependencies, code prerequisite already main) owns the former M2 fix and live fresh-fact proof. This replaces the earlier two-landings-on-one-issue plan, rather than bypassing the one-branch guard. The original M2 acceptance is preserved above as superseded history. Hub owns project updates and the live rebuild/restart.

2026-10-21 09:05 PDT — Hub explicitly deferred this docs-only closure landing behind urgent #774. PR526 remains open and SHIP. Cancelled queued local check lease-3865002467 through sdlc (never acquired capacity; no test result), then interrupted this slot's waiting sdlc merge with Ctrl-C/SIGINT as the merge contract specifies; both runners exited. Preserve branch; after #774 lands, re-enter this closure's recorded check and merge. No waiver or passing-suite claim. #773 remains held.
