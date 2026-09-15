---
id: 000239
status: working
deps: [ariadne#224]
github_issue:
created: 2026-09-12
updated: 2026-09-13
estimate_hours: 6.57
started: 2026-09-13T16:20:21-07:00
---

# Pair's own data store has no garbage collection: 13 GB under ~/.local/share/pair and nothing ever prunes it

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only; calibration currently marked stale.

Mapping: issue/spec discussion; exact artifact ownership extension; five
single-concern modules (policy, coordinator/use, store registry, recoverable
collector, scheduler); cross-cutting Go runtime wiring; Lua use integration;
CLI dispatch; atlas; two milestone reviews plus final close review.
Library check: reuse artifactpath, procutil, x/sys locks and Couch journals;
no external library supplies this repository-specific ownership policy.
Module design uses 1h base times 0.2 thorough-spec discount; smaller module
0.3 times 0.2; Lua 2 times 0.2; wiring 0.6 times 0.2; dispatcher 0.5 times
0.2; docs 0.15 times 0.2; reviews 0.2 times 0.2. Spec/design already performed
uses 0.75h without a second spec discount. Implementation is 40% of v2:
modules 0.8h, smaller/wiring/dispatcher/review 0.5h, Lua 1.5h, spec 0.3h,
docs 0.2h. Familiar stack 1.0; thorough-plan design buffer 15%.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.75 impl=0.12
item: smaller-go-module design=0.06 impl=0.2
item: greenfield-go-module design=0.2 impl=0.32
item: greenfield-go-module design=0.2 impl=0.32
item: greenfield-go-module design=0.2 impl=0.32
item: greenfield-go-module design=0.2 impl=0.32
item: greenfield-go-module design=0.2 impl=0.32
item: cross-cutting-refactor design=0.12 impl=0.2
item: lua-neovim design=0.4 impl=0.6
item: skill-or-dispatcher design=0.1 impl=0.2
item: atlas-docs design=0.03 impl=0.08
item: milestone-review design=0.04 impl=0.2
item: milestone-review design=0.04 impl=0.2
item: milestone-review design=0.04 impl=0.2
design-buffer: 0.15
total: 6.57
```

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

- [ ] M1 — Exact ownership, explicit meaningful-use tracking, live protection and Couch archive grace; no deletion
- [ ] M2 — Recoverable collection, preview/apply, bounded automatic sweep, verification and publication

Implementation details and task checks: [approved plan](../plans/000239-storage-gc-plan.md).

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

## Revisions

### 2026-09-13 — parked captures have independent expiry

Operator requests deleting captures older than 60 days independently of a
reused tag's lifetime. Each timestamped parked capture and its companion
events expire together 60 days after capture, even if the parent tag remains
active or visible in Couch. Active readers temporarily protect their capture;
viewing it does not restart its age clock. This supersedes the earlier
blanket rule that Couch visibility protects every associated Pair artifact.
Diagnostics remain seven days; other session data remains tag-wide sixty days
since meaningful use, with visible Couch thread protection.

### 2026-09-13 — debugging logs use seven days

Operator separates debugging-level information into a **7-day bucket**:
wrapper event traces, adaptation flight recorders and optional Pair/Couch
debug traces. This supersedes their earlier blanket 60-day classification
and the decision to defer writer rotation. Session/recovery data retains
the approved 60-day meaningful-use policy and visible Couch protection.
The durable plan's diagnostic appendix specifies writer rotation and
independent age-based expiry; background diagnostic writes do not refresh
the session-use clock. No real data has been deleted.

Metadata-only sizing: total Pair data about 15.04 GiB; 0.52 GiB (3.5%) had
not been written for 60 days. Wrapper diagnostics alone total about 13 GiB;
11.538 GiB had not been written for 7 days. These are age-based candidates,
not actual eligibility: live/unknown writers and ownership still need checks.
Reusing a tag can retain older contents in recently used session files, but
wrapper raw scrollback and wrapper-event files truncate at wrapper startup;
parked captures are separate preserved copies.

### 2026-09-13 — durable technical plan drafted

The implementation plan is [storage GC](../plans/000239-storage-gc-plan.md).
It proposes two actual review boundaries: M1 adds ownership, explicit use
clocks, live protection and archive grace without deletion; M2 adds
recoverable collection, preview/apply and bounded automatic sweeps. The
original Plan above is historical; its writer-cap and immediate real-store
apply steps are superseded by this design, subject to plan approval.

Fresh review identified cross-filesystem custom Couch archives and detached
children surviving their launcher as ordering gaps. The plan now specifies
store-local journal coordination and actual-process lifetime registrations,
with fault/race tests (ARCH-ORDER). A one-time migration completeness check
for custom Couch stores is proposed before destructive GC can be enabled.
No production files have been deleted and implementation has not begun.

### 2026-09-13 — approved feature policy

Reason: settle feature scope with the operator before implementation. This
revision supersedes the speculative retention rules in the original Spec and
its derived Done-when/Plan wherever they disagree. The operator approved this
policy and said to proceed.

- Couch and standalone Pair have different retention authorities. A thread
  still accessible in Couch's switcher, including a parked thread, protects
  its associated Pair data indefinitely. Background visibility is protection,
  not an access-timestamp update.
- Standalone Pair session data is eligible after 60 days without meaningful
  use. Saved drafts, sent prompt history, queues, scrollback and recovery
  records are within that policy; the earlier tentative keep-forever rule
  was explicitly replaced in discussion.
- Archiving a Couch thread removes switcher protection and explicitly starts
  a fresh 60-day grace. Later meaningful use refreshes it. File-move times
  do not define the policy.
- Meaningful use includes a selected open/resume, viewing session history or
  scrollback through Pair, and a meaningful content write. It is not merely
  last write or filesystem atime. Background inventory, passive redraws,
  polling, diagnostics, and unchanged autosaves do not refresh it.
- Retain data being used by a live session or reader. Unknown liveness,
  uncertain ownership, malformed retention state, or incomplete inventory
  must prevent collection rather than imply expiry.
- Scope is Pair-owned storage and Couch's archived records. Agent-native
  conversation stores and repository working files are outside this GC.

### 2026-09-13 — source corrections and implementation proposals

The consumer audit corrected initial assumptions: live raw scrollback is an
unbounded append stream, not a ring; raw/event files are paired by byte
offsets. Both raw scrollback and wrapper diagnostics truncate at wrapper
startup, but can grow within an incarnation. A resumed park capture can still
be consumed by an incoming agent, so resume alone is not a disposal event.
History discovery reads drafts/logs/ledgers independently of Couch records;
absence from Couch alone does not prove orphanhood. The artifact manifest
enumerates tag-bearing vocabulary, not every physical data-directory entry.

Proposals for the implementation plan, distinguished from the agreed policy:
existing data without explicit use metadata gets a full 60-day onboarding
grace; preview is read-only; applied/on-startup GC initializes missing
metadata. Managed open/view paths refresh use, while arbitrary external file
reads cannot be observed. A bounded daily opportunistic sweep plus explicit
preview/apply commands will implement expiry. Writer size caps from the
original sketch are deferred: deleting history inside its approved 60-day
window requires a separate policy, and this work promises retention, not a
global disk-space ceiling.

### 2026-09-13 — implementation authorized

Operator approved the reviewed technical plan and said “go ahead.” The
active Plan now names its two review boundaries; the original checklist is
preserved below as historical scope, superseded by the approved policy.
The operative acceptance criteria are the durable plan's Product contract
and Acceptance evidence, including no size cap or destructive real-store test.

Original checklist

- Inventory: every family in `artifactpath/manifest.go` × writer × readers × last-needed moment; write the table into the Spec
- Brainstorm the policy set and the run trigger with the operator
- Manifest lifecycle field + conformance test
- `pair gc` dry-run + `--apply`, per-family policies, liveness guard, report
- `wrap-events` writer cap + rotation
- Run on the operator's store; record before/after bytes in the Log

### 2026-09-13 — agreed final parked-capture policy

Operator confirmed 7 days for old parked captures, superseding the preceding
60-day capture revision. Expiry follows each capture’s age, independent of
tag activity and Couch visibility; raw and event sidecars are removed together.
Active readers or incomplete handoffs defer deletion. Session data remains
60 days since meaningful use; Couch-visible threads remain protected until
archival, which starts a fresh 60-day grace.

### 2026-09-13 — implementation checkpoint

Final approved buckets are session60d, debug7d, immutable parked captures7d;
Couch membership protects session data until archive begins fresh grace.
Implemented actual-process leases, durable content-use intents, startup/reader
handoffs, Couch archive clocks/receipts, exact artifact inventory, quarantine
recovery, capture-only rediscovery, shared binding cleanup, and diagnostic
writers. Foreground Lua CLI integration and unchanged-save behavior pass real
Neovim/CLI tests. Full Couch suite passed76.058s; quarantine race suite passed;
remaining full-tree checks and both review boundaries are outstanding.
Public GC command and readiness-triggered scheduler are being composed now.
No user storage has been deleted and no new binary has been installed.

### 2026-09-13 21:08 — integration checkpoint before continuation

Public `pair gc` preview/apply/migration and readiness-triggered scheduling are
implemented, along with all three retention buckets. New parked captures now
publish an explicit UTC creation timestamp with exact raw/events identities;
the collector validates those identities and falls back conservatively only
when metadata is absent. Changed/missing payload or malformed metadata retains.
The new publication-clock regression failed before collector integration and
passes afterward; full storagegc suite passed8.797s. Adapt shell/Lua schema
integration passed (contended optional logs may skip); main changelog/streaming
fixture regressions passed1.761s. Diagnostic race and Linux compile passed;
source inventory passes after new metadata sources were classified.

The first full-tree run exposed a production path-spelling regression in
ParkScrollback (/var versus /private/var), now fixed and under full Couch tests,
and a timerless intentionally-idle subprocess fixture that Go could call deadlocked.
Full-tree checks must be rerun after these fixes. Lua/real CLI checks are running
in /tmp/pair-239-final-shell.log. Neither milestone has crossed its mandatory
review boundary. No completion, installation, real-store migration or deletion
is claimed. Next: finish full checks, preview the real store read-only, reconcile
plan checkboxes and run SDLC M1/M2/close reviews, then publish.


### 2026-09-14 — Resumed integration and fresh verification

Resumed the committed implementation in its existing worktree. Merged current
main, preserving checkpoint acknowledgment and retention startup handoffs;
focused restart/continuation race tests pass. The combined Neovim configuration
exceeded Lua's 200-local limit; scoped its existing help/changelog helper, then
native compilation and `make test-lua test-retention test-adapt-schema` passed
(`/tmp/pair239-resume-shell-fixed.log`). Full post-merge Go suite passed
(`/tmp/pair239-resume-full-go.log`). Storage, diagnostic, runtime, CLI, Couch and
launcher race suites passed (`/tmp/pair239-resume-race.log`).

Real-store public CLI preview completed with no migration/apply flags:
`/tmp/pair239-preview gc --root /Users/xianxu/.local/share/pair --json`, output
`/tmp/pair239-real-preview.json`. Recognized totals: session 232 groups / 1.106
GiB, parked captures 108 groups / 1.584 GiB, diagnostics 215 entries / 15.121
GiB. No eligible deletion: Couch registry unavailable/unacknowledged and legacy
runtime inventory incomplete. 1,942 unknown paths are separately reported,
not authorized for collection. These are logical bytes, not promised reclaimable
bytes. No user data was deleted; no new binary installed.

Acceptance audit found the public apply survivor/idempotence fixture and checked
managed-I/O inventory still missing; those are being completed before review.


Public CLI apply acceptance now passes under race (`/tmp/pair-239-public-apply.log`):
real portable stores and public migration/preview/apply commands collect expired
standalone and archive data plus independent old capture/debug data, while
preserving Couch-visible parked, new archive and actual live-process owners.
Second apply collects zero. Replacing public Apply with Preview fails the
collection assertion (`/tmp/pair-239-public-apply-mutation.log`). Broad process
and open-file inventories are deterministic command fixtures; live owner birth
identity is checked by the OS. No production seam was added for this test.

`atlas/storage-retention-io.md` records 19 managed entrypoint families and their
actual protection/use/test chains, including merged continuation/recovery and
programmatic Lua saves. `TestManagedIOInventoryAnchors` checks source/test anchors
and 16 direct guard-call edges; passed after missing-doc RED. This bounded audit
does not claim whole-program control-flow proof. README documents public GC and
its explicit batch limits. All technical M1 obligations are now ready for the
first mandatory milestone review over the combined checkpoint implementation.


### 2026-09-14 22:34 PDT — M1 rework: BR-1 and BR-5 verified

Eligible metadata-only owners now retire through the existing recoverable
collection journal, including activity recreated during independent capture
cleanup. Visible/live/unknown/unfinished-handoff protections remain effective.
`ReduceTransaction` now enforces the real prepared → detached → finalized
lifecycle; production phase publication uses one adapter and frozen deletion
authority is unchanged. The plan's concept tables and source classification now
match this implementation (ARCH-FUNERAL, ARCH-PURE, ARCH-ORDER).

Original failure: `/tmp/pair239-br1-red.log`. Pure/integration coverage:
`/tmp/pair239-br1-br5-focused.log`; regressive transition mutation caught in
`/tmp/pair239-br5-mutation.log`. Full storagegc race pass: 14.875s in
`/tmp/pair239-br1-br5-race.log`; final focused pass: 5.203s in
`/tmp/pair239-br1-br5-final-focused.log`. Other M1 findings and the review gate
remain separate outstanding work; no new completion or real-store deletion claim.


### 2026-09-14 22:47 PDT — Remaining M1 rework integrated; full verification running

BR-2/BR-3 now cover killed metadata publishers, bounded shared cleanup of legacy
root/scoped/owner residues, dead pre-spawn reservation reclamation, explicit
uncertain-start resolution, and no-op recovery without metadata rewrites. Logs:
`/tmp/pair239-maintenance-recovery-race.log`,
`/tmp/pair239-temp-budget-race.log`, `/tmp/pair239-legacy-pending-race.log`.

BR-4 now gives absent legacy archive clocks a journaled full grace for selected
owners, recovers pending store journals before snapshot, and retains malformed
or replaced identities. Public acceptance exposed flat standalone owners at the
Couch adapter boundary; root/owner validation now precedes flat-owner filtering.
Mixed scoped/flat and public apply tests pass under race (gcruntime1.595s,
gccmd3.119s). Preview remains read-only.

BR-6 automatic work now has a real owner cursor, bounded visited-owner effects,
nonblocking root acquisition, a cooperative two-second context budget, and
cancellation checks across transaction and diagnostic effects. One payload
inventory is reused only while the same root lock remains held. Recovery budgets
yield for retry, and the durable cursor is preserved on deadline or contention.
100,000-filename production coverage proves the inventory cap retains otherwise
eligible data; small counted tests cover pre-migration progress, busy locks,
deadlines and expired contexts. Explicit apply still scans past retained owners;
its regression passes. Oversized stores are not promised automatic completion
within two seconds; partial inventory never authorizes deletion.

Full-suite integration also exposed a missing console-test fixture shutdown
join. Production already joins; the fixture now does too, with race x20 coverage
(`/tmp/pair239-console-join-race.log`) and full couchtty pass. Regenerated ignored
runtime bundle assets after the embedded-source drift check identified stale
Neovim content. `/tmp/pair239-rework-full-go-final.log` is running on this tree;
the prior full run was not a pass. No real-store apply/migration/install occurred.

`sdlc actual --issue 239 --brain-dir /Users/xianxu/workspace/brain` still measures
0.21h with explicit incomplete attribution warnings
(`/tmp/pair239-rework-actual.log`). This is a tool measurement, not an assertion
that total development took0.21h or a valid estimate comparison.


Final rework verification: full `go test ./... -count=1` passed with selected
runtime environment cleared (`/tmp/pair239-rework-full-go-final.log`). Storage
and runtime race suites passed; public apply acceptance passed after flat-owner
adapter correction. Legacy pending integration race passed2.721s and console
shutdown regression race x20 passed3.027s. `git diff --check` passed. Ready for
M1 round2; BR finding disposition belongs to that fresh review, not this log.


### 2026-09-14 23:07 PDT — M1 round2 REWORK; class fixes underway

The fresh review addressed BR-1, BR-2, BR-3 and BR-5 with mutation checks.
BR-4 still missed durable owners whose payload namespace was absent; BR-6 still
ran session probes during diagnostic pages and blocked on nested Couch locks;
BR-7 found unique quarantine directories created before journal publication.
These are real remaining gaps, not a completed milestone.

Absent-namespace legacy/tracked archive and metadata-only cases now pass with
full grace and eventual collection (focused race: storagegc1.300s,
gcruntime2.627s). Diagnostic-page regression reproduced9 session probes and now
passes with zero probes/writes while collecting old diagnostics. All five nested
Couch maintenance APIs pass contention/cancellation regressions under race
(`/tmp/pair239-nested-lock-race.log`). Journal-before-quarantine publication
passes repeated errors, cancellation, real killed publisher and unsafe-path
coverage; full storagegc race passed14.745s (`/tmp/pair239-br7-race.log`). The
related Couch publisher stage lifecycle is being completed before round3.

Draft PR138 runs CI. Bootstrap fixes are committed/pushed at4ef4d69b and3149cdd7;
workflow regressions pass. Hosted tests now run, but one live Zellij detach
fixture timed out after preceding live checks passed. The exact group passes
locally4.437s on the same Zellij0.45.1; one hosted retry was requested. No test
or timeout was weakened. Publication remains gated.


Round3 final verification passed: full `go test ./... -count=1` with selected
runtime environment cleared (`/tmp/pair239-round3-full-go-final.log`), including
couchcore138.339s and gcruntime123.919s. A preceding run found three unused
compatibility wrappers; they were removed without allowlisting, and the final
full run is clean. Storagegc/gcruntime/gccmd race suites passed
(`/tmp/pair239-round3-race.log`), plus targeted Couch publication/retention/archive
race11.295s. The complete branch diff passes `git diff 6b06b449 --check`.
Hosted conformance passed on the single retry; merge-check passed. Ready for M1
round3, with BR-4/BR-6/BR-7 disposition still owned by the fresh reviewer.

Main checkout's older239 plan edits and untracked plan-gate copy are preserved
in stash ac42bde4cc2c8976ee5f0096c663a7757575c5fc before publication; the current
worktree plan contains the evolved design and review revisions. Unrelated
.nvimlog was left alone. No real-store apply/migration/install was performed.

- 2026-09-14 23:35 PDT: M1 round3 returned REWORK on diagnostic nested cancellation (BR-6), interrupted registry publication/pagination (BR-7), and deletion replay after ancestor removal (BR-8). Addressed BR-4 absent namespace onboarding. Extending fixes across diagnostic publishers and nested effects; root traversal cancellation race passes15.782s, mutation regression catches missing checks. Review cap will be raised to10 to preserve Important blocking severity. No real-store mutation performed.

- 2026-09-14: Round4 fixes pass full Go suite `/tmp/pair239-round4-full-go-final.log`, storage race `/tmp/pair239-round4-storage-race.log`, and diagnostic/runtime/CLI race `/tmp/pair239-round4-integration-race.log`. Publisher process-death, registry pagination/cancellation, preserved scheduler cursor and deletion replay effect boundaries are covered. Live CI failure traced to external Zellij0.45.1 startup panic; fixture investigation continues before next review.

- 2026-09-14: Live fixture startup race resolved without changing production: wait for server-rendered screen before probing Zellij. Helper race2.249s and fresh-config live group3 repetitions16.426s pass. Preparing M1 round4 with Important findings kept blocking; actual attribution remains incomplete, so close will record N/A rather than fabricate milestone increments.

- 2026-09-15: M1 round4 cleared BR-1 through BR-8 and raised BR-9 ordinary append recovery. Implemented bounded exact append intents, observed-prefix reconciliation preserving partial-write counts, and staged-inode current creation. Direct collection recovers killed creation publishers without reopen. Avoided per-record fsync latency (25ms) with ordered atomic hot append publication (0.65–0.87ms), preserving durable lifecycle sync. Full Go `/tmp/pair239-round5-full-go.log`, diagnostic/creation/append races and complete isolated live conformance pass. Preparing round5; actual remains N/A due incomplete attribution.

- 2026-09-15: M1 round5 found only the remaining StoreRegistry PURE classification contradiction; corrected to INTEGRATION with revision (ARCH-PURE). Reviewer independently passed diagnostic race/package tests and mutation-tested append recovery. Hosted shortcut fixture failure reproduced by stalling its client200ms: sender sleep combined ESC+[ and the printable barrier into incomplete CSI. Receiver acknowledgment fixes it with unchanged per-delivery deadlines; original fails and patched all13keys pass, including three normal repetitions.
