---
id: 000355
status: working
deps: []
github_issue:
created: 2026-09-30
updated: 2026-09-30
estimate_hours: 4.193
card_mirror: '0b4898ebd3943ade4e8aeffda3154d810a780b03' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-30T09:37:39-07:00
flow: {kind: full, provenance: inferred}
---

# Allocate Couch session identities and enforce repository families

## Problem

Couch allocates global Zellij session names using per-checkout name indexes.
After pair:2 was parked, its retained `📁pair-couch-3` name was also allocated
to pair:1. The live session belonged to pair:1, but pair:2's fresh-slot check
treated that name as its own surviving session and refused startup. Separately,
pair:2's unconfirmed Codex binding blocked open-slot. The binding was recovered
by unique prompt/progress correlation and the stale name reference was backed
up and removed; the allocation defect remains.

Human-readable internal names and independent name-shortening/deduplication
obscure ownership. Operators need repository labels, threads, and slot numbers;
they do not need memorable Pair tags or Zellij names for Couch-managed sessions.
Starting at two different subdirectories of the same repository must also not
create competing slot families over the same physical repository.

## Spec

### Identity allocation

Use three independently owned monotonic counters, replacing the proposed
store-path hash with an explicitly allocated store number:

- `C`: a host registry assigns a Couch store number to its canonical store path.
  The registry is shared by all Couch stores using the same Zellij namespace.
- `N`: each Couch store allocates Pair conversation numbers.
- `M`: each Couch store separately allocates Zellij session numbers.

Names are `C-repo-N` for Couch-generated Pair tags and `📁C-M` for Zellij
sessions. The repository component is descriptive; uniqueness comes from the
allocated numbers. Couch supplies the exact Zellij name to Pair, without a
second shortening or deduplication scheme. Standalone Pair retains support for
user-chosen memorable tags. Native chat applications keep their own UUIDs;
Couch/Pair retain the binding to those UUIDs rather than trying to replace them.

| Event | Pair N | Zellij M |
|---|---|---|
| New conversation / start fresh | New | New |
| Attach to surviving session | Keep | Keep |
| Park and resume after terminal teardown | Keep | New |
| Restart agent inside surviving Zellij session | Keep | Keep |
| Recreate terminal session for same conversation | Keep | New |

Persist allocation before launch; serialize allocation at the owning store or
registry. Failed starts may leave gaps. Do not recycle numbers. Keep the current
Pair-to-Zellij association in durable state and verify ownership before using a
live session; an occupied name is not proof that its session belongs to us.
The practical operating model is one running Couch switchboard, with independent
stores distinguished by C rather than by a separate naming service.

### Repository families and starting directories

A thread is an agent conversation with a starting directory and launch profile.
A slot is a persistent workspace that can host successive threads. A repository
family groups :0 and its isolated worktree slots :1, :2, etc., and owns one
relative starting directory within the repository.

For example, starting :0 at `kbench/competition/arc-agi-3` records the repository
and relative directory `competition/arc-agi-3`. Additional slots use that same
relative directory inside their worktree checkouts. Starting a new family at
`kbench/competition/arc-agi-2` must be refused while the first family exists,
with a diagnostic identifying the existing family. This applies to another
request at the repository root too, and to aliases resolving to that same
repository. Opening an existing slot and deliberately starting fresh within it
remain valid operations.

Parked threads count: a family remains present and reserves its repository and
starting-directory choice until explicitly removed, not merely until its agents
stop. The operator sees repositories, starting directories, and slot offsets;
the C/N/M identifiers remain internal.

### Design work still required

Choose the registry location, storage/locking integration, and migration from
existing names and records without interrupting or misattaching live sessions.
Specify store relocation/reset/restore behavior so counters cannot silently
reuse surviving identities. Resolve common repository identity across worktrees
and aliases, and the behavior when a configured relative starting directory is
missing. Keep these decisions in the implementation plan; this task captures
the agreed model, not a completed implementation design.

ARCH-DRY: each identifier has one allocating authority. ARCH-ORDER: allocation
and launch have explicit crash/retry behavior. ARCH-FUNERAL: define retention of
retired session associations while preserving monotonic counter high-water marks.

## Done when

- Host/store counters allocate distinct C/N/M identities with durable concurrent
  and interrupted-allocation tests; numbers are not reused after failure/restart.
- Boundary tests cover every lifecycle row above, including multiple Zellij
  sessions over the lifetime of one Pair conversation.
- A parked session name cannot be reassigned to another managed conversation;
  the pair:2/pair:1 collision is reproduced as a failing regression before the
  fix, and foreign live ownership never authorizes attach, park, or teardown.
- Existing live and parked conversations migrate or remain explicitly compatible
  without losing prompts, history, TTY captures, native bindings, or standalone
  Pair's user-chosen tags; names fit the actual Zellij socket-path budget.
- Starting a conflicting family for the same repository is refused for live and
  parked families, including distinct subdirectories and path aliases; the error
  identifies the existing family without creating partial resources.
- Starting from a repository subdirectory carries the same relative starting
  directory into added slots; ordinary existing-slot open/fresh actions work.
- Tests and documentation describe identity ownership, migration, family lifetime,
  recovery, and the repository/subdirectory behavior.

## Plan

- [x] Review and approve the durable [implementation plan](../plans/000355-couch-session-identities-plan.md).
- [ ] M1 — Allocate C/N/M identities, carry terminal bindings through launch,
  and verify ownership with compatible live/parked migration.
- [ ] M2 — Persist repository-family admission and carry its relative starting
  directory through every slot launch, storage, inventory, and menu path.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

Four bounded Go concerns: durable allocator, binding transitions, managed launch
protocol, and repository-family authority. Ownership probing is one external API
integration; slot consumer propagation is one cross-cutting refactor. Existing
flock/journal/process/parser seams are reused. The approved detailed plan applies
the ×0.2 design discount; implementation values are 40% of v2 table upper bounds.
Familiarity is 1.0. Includes documentation and two milestone reviews. The shared
calibration is provisional/stale, so this is a planning estimate, not a deadline.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.2 impl=0.32
item: greenfield-go-module design=0.2 impl=0.32
item: greenfield-go-module design=0.2 impl=0.32
item: greenfield-go-module design=0.2 impl=0.32
item: api-integration design=0.4 impl=0.6
item: cross-cutting-refactor design=0.1 impl=0.2
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.2
item: milestone-review design=0.04 impl=0.2
design-buffer: 0.15
total: 4.193
```

## Log

### 2026-09-30

- Captured the operator's agreed C/N/M allocation model and repository-family
  restriction. Separate N and M reflect distinct Pair and Zellij lifetimes;
  a host counter replaces hashing the Couch store path. Parked families retain
  their repository reservation. No implementation changes made in this task.
- Claimed at operator request to implement. Entered planning on
  `000355-couch-session-identities`; inspected both tag allocation sites, the
  shared tracked-launch boundary, name assignment and ownership readers, and
  slot-family storage/routing consumers. Durable implementation plan drafted;
  fresh-context review in progress. No production code changed yet.
- Fresh-context plan review completed: initial findings about old-launcher
  protocol rejection and counter-restore safety were addressed, and re-review
  approved both chunks with no blocking findings. Issue schema validation and
  diff whitespace checks pass. Awaiting durable-plan approval required by
  AGENTS.md §2 before `sdlc change-code`; implementation has not started.

- Operator approved the durable plan. Initial implementation gate requested explicit
  pending-binding recovery and function-level test strategies; refined both and
  plan quality accepted all findings in round two. Focused launcher/threadrecord/couchcore baseline passed.

- Implementation gate passed. M1 allocator and launcher protocol are implemented;
  counter subprocess/race/fuzz tests, full launcher tests, and the managed cold →
  warm → park/reopen identity test pass. Read-only live Zellij ownership probe
  and socket-budget conformance pass. Broader integration testing is in progress.
- Integration exposed continuation target-session comparisons that assumed the
  old terminal name survived recreation. Updating those consumers to use the
  promoted terminal binding while preserving historical source checks (ARCH-PURPOSE).

- M1 implementation checkpoint: managed launcher, allocator, checkpoint,
  threadrecord, Zellij parser, Couch command and TTY suites pass; targeted Couch
  binding/recovery/ownership race tests pass. Name-handoff and foreign-owner
  guard mutations fail the intended boundary regressions and were restored.
  Clean baseline archive with generated runtime assets reproduces all 32
  artifact inventory findings (existing #348); this branch adds none.
  Final complete Couch core partitions are running before milestone review.

- M1 final verification: all Couch core tests passed in three exhaustive name
  partitions (94.948s, 124.102s, 166.635s); Couch command suite passed (44.613s),
  TTY, launcher, allocator, durablefile, checkpoint, threadrecord and zellijpane
  suites passed. Core boundary race tests and allocator/launcher race tests pass.
  Owner-command fuzz: 25,932 executions; pane-evidence fuzz: 270,652 executions;
  allocation fuzz: 176,771 executions. `make build` rebuilt Pair/Couch/helper;
  its secondary workflow target skipped a duplicate pair-go build via sentinel.
  `git diff --check` passes. Submitting M1 for its mandatory review.

## Revisions

### 2026-09-30 — Implementation requested

Replaced the filing-only placeholder with a durable plan and two actual review
boundaries: identity/terminal lifecycle, then family/subdirectory behavior.
The agreed product model and Done-when contract are unchanged. The plan records
storage, compatibility, crash/retry, and migration choices for review.
