---
id: 000239
status: working
deps: [ariadne#224]
github_issue:
created: 2026-09-12
updated: 2026-09-13
estimate_hours:
started: 2026-09-13T16:20:21-07:00
---

# Pair's own data store has no garbage collection: 13 GB under ~/.local/share/pair and nothing ever prunes it

## Problem

Measured on the operator's machine, 2026-09-12. `~/.local/share/pair` is
**13 GB**. No code path in the tree deletes anything under it except the
exact per-pane sidecars removed on Alt+x; every other family grows until the
disk does.

| family | bytes | notes |
|---|---|---|
| `wrap-events-*.jsonl` | 12.7 GB | 163 files; one is 4.8 GB (`repos/e108517d46ab4575/wrap-events-pair.jsonl`); 345 MB is older than 60 days |
| `scrollback-*.raw` | 767 MB | live scrollback rings; one is 203 MB |
| `parked-scrollback-*.raw` | 543 MB | one per park; parks that were later resumed keep theirs |
| `ledger-*.jsonl` | 73 MB | 143 ledgers; 99% of the bytes are launch boundary snapshots (#238) |
| `draft-`, `log-`, `config-`, `agent-ready-`, `thread-claim-` … | small | per-thread sidecars, hundreds of files |

By contrast the agent's own store, `~/.claude/projects`, is bounded: Claude
Code sweeps transcripts after 30 days and the oldest file on this machine
is 29 days old (650 MB, 2,629 files). Pair records more than the agent does
about the same sessions and keeps it forever.

This is the first instance of the principle filed as ariadne#224
(`ARCH-FUNERAL`): every artifact a program creates needs a designed end — a
time-bound sweep, or an explicit growth bound with its consequences stated.
Pair's data families have neither.

## Spec

Brainstorm first; the questions below are the design, not the answer.

1. **Inventory the families and their consumers.** For each artifact
   family under the data dir (the manifest in `artifactpath/manifest.go`
   already enumerates them): who writes it, who reads it, and what is the
   last moment a reader can need it. That moment is its funeral. Some
   candidates from the reads already known:
   - `wrap-events`: the agent-pane event log; consumers are the scrollback
     oracle and perf capture. Nothing reads events older than the current
     thread's life. Time-bound by age, and size-bound per file (a 4.8 GB
     single file is a rotation failure, not a retention failure).
   - `scrollback-*.raw`: live ring for the running pane; dead once the
     pane's process is gone. Owner-dead → collect.
   - `parked-scrollback-*`: needed until the park is resumed or the thread
     is retired; a resumed thread's old park file is garbage.
   - `ledger-*`: append-only by contract, never truncated; its growth is
     #238's problem (shrink the row), not GC's.
   - Per-thread sidecars (`draft-`, `log-`, `config-`, …): live while the
     thread record exists in couch's store; orphaned when it is retired.
2. **One routine, explicit policy per family.** A `pair gc` verb (dry-run
   by default, `--apply` to delete) that walks the manifest families and
   applies each family's declared policy: `age > N days`, `owner process
   dead`, `thread retired`, `superseded by a newer generation`. The policy
   lives next to the family in the manifest so a new family cannot be added
   without one (the manifest's conformance test is the enforcement point).
3. **Where it runs.** Candidates: at `pair` launch (cheap, bounded walk;
   like Claude Code's own sweep), and on demand. Never in a keystroke path
   (ARCH-CONSTRAINTS). Must be safe under concurrent sessions: a file whose
   owner is alive is never touched, and deletion is by exact path from the
   walk, never by glob.
4. **Rotation for the unbounded writers.** `wrap-events` needs a size cap
   with rotation or truncation at the writer, independent of GC; GC
   collects rotated segments by age.
5. **Report.** `pair gc` prints per family: files, bytes, what it would (or
   did) remove and why. The number the operator sees first is bytes freed.

## Done when

- Every artifact family in the manifest declares a lifecycle policy; the
  manifest conformance test fails for a family without one.
- `pair gc` dry-run on the operator's store reports the 13 GB by family and
  what each policy would remove; `--apply` brings the store under a stated
  bound (target: the live working set — threads in couch's store plus N
  days of history).
- A file whose owner process or thread is alive is never removed, asserted
  with a fake that reports live owners.
- `wrap-events` cannot grow a single file past its cap; asserted at the
  writer.
- `atlas/` records each family's funeral in the artifact table.

## Plan

- [ ] Inventory: every family in `artifactpath/manifest.go` × writer × readers × last-needed moment; write the table into the Spec
- [ ] Brainstorm the policy set and the run trigger with the operator
- [ ] Manifest lifecycle field + conformance test
- [ ] `pair gc` dry-run + `--apply`, per-family policies, liveness guard, report
- [ ] `wrap-events` writer cap + rotation
- [ ] Run on the operator's store; record before/after bytes in the Log

## Log

### 2026-09-12

- Filed from the #237/#238 investigation. Measurements taken with `du` and
  `find -exec stat` over `~/.local/share/pair`; the agent-store comparison
  from `find ~/.claude/projects -mtime`. The principle this instantiates is
  ariadne#224 (`ARCH-FUNERAL`), filed the same day; this issue is its first
  application and should be designed against the principle's `at-plan` text
  once that lands.

### 2026-09-13 — spec discussion opened

Operator requests settling the feature spec together before implementation.
Claimed for design only; no collection or code changes authorized yet.
Fresh metadata-only scan (logical file bytes, GiB): wrap-events 12.859 / 171
files; scrollback 1.003 / 130; parked-scrollback 0.682 / 300; ledger 0.086 /
248; other families about 0.177. The two largest wrapper-event logs total
8.547 GiB. These figures replace the old sizing baseline, not the proposed
policies. Consumer/lifecycle audit is in progress; protection of drafts and
threads versus age-based expiry is the first operator decision pending.
