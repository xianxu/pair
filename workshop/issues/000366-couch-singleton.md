---
id: 000366
status: working
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: '9cf6ed18768e9f6d5b3217514d3e97cc8e1642c4' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-01T22:14:48-07:00
---

# Make Couch a local singleton

## Problem

Durable assignment should identify machine + slot without a Couch instance ID. Today namespace-scoped supervisors require an explicit singleton/migration contract before that assumption is valid.

## Spec

Project: pair/workshop/projects/cross-slot-work-scheduling.md. Captured for operator review; no implementation is authorized by this issue creation.

Make Couch a local singleton with one authoritative slot inventory. Proposed boundary is one supervisor per OS user on a machine; validate and settle that boundary in design. Multiple Ariadne operators/machines may independently run their own local Couch and share issue trackers. Multi-machine Couch routing is out of scope.

Define existing-namespace migration/adoption, second-launch behavior, crash restart and slot identity preservation. Reuse supervision/lease mechanisms and ensure tests/isolated diagnostic stores have an explicit supported isolation path. Do not delete existing conversations, reset worktrees or silently adopt conflicting state.

## Done when

- The singleton boundary is explicit; concurrent starts cannot establish two production supervisors inside it.
- A second invocation attaches/routes to the existing supervisor or gives a concrete diagnostic.
- Existing stores and stopped/live slots migrate or are reconciled without losing conversations, preferences or dirty work.
- Restart and isolated-test behavior are verified; durable issue ownership requires no Couch ID.

## Plan

Implementation plan to be designed after issue claim and start-plan; these are requirements, not an approved implementation plan.

## Log

### 2026-10-01

Captured from the performance → messaging guarantees → SDLC ownership/observability → recovery discussion. No implementation started.

## Proposed design — awaiting operator review

### Scope and alternatives

Recommend one production Couch supervisor per OS user per machine, backed by
one persistently selected store. A second launch refuses immediately and names
the selected store and verified owner when available. Connecting another terminal
to the running Console is outside this proposal.

A host-wide lock alone is smaller but allows sequential launches to switch slot
inventories. A shared daemon with multiple attached terminals solves a broader
presentation problem and is unnecessary for the project's ownership contract.
The selected-store design keeps one inventory without introducing that daemon.

### Ownership and storage

Production authority must be independent of COUCH_STORE_DIR, XDG_DATA_HOME and
an overridden HOME. Resolve the actual OS account's home from its UID through an
injected account lookup; failure refuses production startup. Keep the host lease
and selected-store record under that account's pair-host directory. Existing
store identity allocation remains compatible; mismatched legacy host authority
is a migration refusal, never a fresh counter allocation.

Acquire a non-inherited, nonblocking host lease before selecting/adopting a store,
then the existing store supervisor lease before constructing Couch, attaching a
terminal, starting a broker or launching a child. Keep per-store transaction locks.
Release the store lease before the host lease. Kernel lock ownership authorizes;
PID/start-token metadata only explains contention. No polling or PID-based takeover.

Persist the canonical physical selected-store path atomically. A selected path
that is unavailable, corrupt, or no longer canonical refuses with recovery context;
it never selects a fresh default. Ordinary production launch and inventory commands
resolve that same selection. A conflicting explicit store override refuses and
names both paths. Read-only observations do not take the lifetime supervisor lease.
Child routing continues to use the selected physical namespace. Existing C/N/M,
tags, native UUIDs, worktrees, preferences and conversation history remain intact.
C is a historical store identity, not a required field in issue assignment.

### Adoption and legacy reconciliation

Adopt the chosen existing store in place; do not copy it into the default directory.
On first use, inspect the default/requested store and registered legacy stores.
Discovery is bounded by existing registry limits, runs at adoption rather than idle,
and reports unavailable or unreadable entries as unknown. Arbitrary unregistered
custom stores cannot be discovered exhaustively; operators supply those paths.

Provide an explicit adoption command whose preview identifies the selected path,
other known stores, active legacy owners, unresolved observations and conflicting
inventories. Application revalidates the preview under the host lease and refuses
changed evidence. All older Couch supervisors must be stopped and upgraded before
cutover: older executables cannot be made to respect a new host lock. Retain the
selected store's old lease for compatibility with its existing consumers.

Fresh installation and a sole usable legacy store can select/adopt without a
separate confirmation. Multiple populated stores or unresolved ownership require
explicit reconciliation; no selection based on recency and no silent merge.
The proposed first version reports these conflicts and preserves every store.
Whether #366 must additionally merge nonconflicting legacy inventories is the
remaining scope decision for operator review; do not claim migration complete for
an unresolved multi-store installation.

Surviving wrappers belonging to the selected namespace reconnect through #365's
existing lifecycle protocol. Wrappers belonging to other stores are reported as
migration conflicts, never retargeted by changing their identity or socket path.
Crash restart reacquires the kernel lease and reads the durable store selection;
stale owner metadata does not authorize takeover or block a free kernel lease.

### Explicit isolation

Provide an explicit isolated runtime mode for tests and diagnostics with a
caller-supplied absolute root. It scopes Couch ownership, selected-store metadata,
allocation authority, inventory and sockets away from production, and propagates
that configuration to children. Production directory overrides alone do not imply
isolation. Extend tests/with-isolated-pair.sh so actual process tests isolate Pair
artifacts as well, clear inherited session path overrides and protect production
sentinels. Tests use the same runtime resolution and lease path as production.
Isolated runtimes are intentionally separate from production slot scheduling.

### Verification and operating envelope

- Concurrent production starts with distinct store/HOME/XDG overrides produce
  exactly one owner and no losing child, broker or terminal attachment.
- Same-store aliases contend; independently rooted isolated runtimes coexist;
  isolated subprocesses and descendants cannot alter production sentinels.
- Kill/restart releases ownership without deleting lock files or resetting counters;
  child processes do not inherit the lifetime lock descriptor.
- In-place adoption preserves tags, UUIDs, preferences, history, dirty worktrees and
  selected-store identity. Refused adoption changes no source inventory.
- Legacy live owners, ambiguous stores, missing/corrupt selected state and a stale
  preview each refuse visibly, preserving unknown observations.
- Production launch, inventory reads and child messaging agree on the selected
  namespace; existing delivery/reconnect and idle-no-discovery regressions stay green.

Ownership acquisition uses one nonblocking kernel-lock attempt; no waiting loop.
Steady-state operation introduces zero recurring discovery or subprocess probes.
Adoption uses bounded registry enumeration and existing observation timeouts;
exhaustion remains an explicit unresolved result. Supported platforms remain the
existing Unix targets. No remote routing, automatic takeover, daemon, or durable
message queue. ARCH-PURPOSE separates durable selection from transient ownership;
ARCH-DRY reuses leases and #365 lifecycle delivery; ARCH-MOCK requires isolated
production-path subprocess tests; ARCH-ORDER requires explicit adoption outcomes.

## Revisions

### 2026-10-01 — initial design exploration

Claimed #366 and entered planning on its own branch from fresh main. Expanded the
project requirements into the proposed design above without replacing the captured
Spec. Second-launch refusal is a recommendation pending the operator's answer;
legacy multi-store reconciliation remains an explicit scope decision. No code changed.

### 2026-10-01 — second-launch decision and baseline

Operator selected immediate refusal with a clear running-owner diagnostic for a
second launch. This settles that part of the proposal; attaching another terminal
is excluded. Multi-store migration scope is still awaiting an answer.

Existing focused supervisor/namespace/CLI lease tests passed:
`go test ./cmd/internal/couchcore ./cmd/internal/couchidentity ./cmd/internal/couchcmd -run 'Test(Supervisor|AcquireSupervisor|ResolveCouchNamespace|StartAcquires|ResumeAcquires|HeldSupervisor|OSRuntimeRefuses)' -count=1`.
The identity package had no tests matching that filter; this is baseline evidence
for existing lease behavior only, not verification of #366. A fresh-context spec
review is in progress before implementation planning.

### 2026-10-01 — spec review revisions (supersede matching proposal paragraphs)

Fresh-context review identified supporting-root drift, incomplete legacy recovery
semantics, and a mismatch between refusal-only migration and the captured acceptance.

- Persist the adopted runtime's canonical Couch store, Pair data root and existing
  allocation-authority root together as one bounded, versioned configuration. The
  production singleton lease still belongs to the real OS account, independent of
  those paths. Preserve a valid legacy authority in place even if it is under an
  older HOME; never create substitute counters to make adoption pass. All Couch
  readers, launch configuration, continuation/slug/history/session readers, traces,
  inherited child paths and messaging consume this resolved configuration. Explicit
  conflicting ambient paths refuse with the configured path and corrective action.
  A fresh installation uses the real account's default local data roots. Adoption
  previews the associated paths and accepts explicit legacy paths when discovery
  cannot establish them; missing/regressed authority requires restoring the valid
  existing authority, not re-enrollment. Standalone Pair's configuration is outside
  this change; hosted Pair receives the selected configuration explicitly.
- Authority and adopted state must reside on machine-local storage. Shared/network
  home/state directories are outside this first version's supported envelope; the
  per-machine guarantee assumes local storage and does not introduce a distributed
  filesystem lock protocol.
- The adoption preview is ephemeral. Apply reacquires host and selected-store leases,
  re-reads configuration and relevant source revisions under existing transaction
  locks, then publishes one atomic configuration file. Failure before publication
  leaves selection absent and sources untouched; lost acknowledgment is recovered
  by reading the selection, with an identical retry converging on that configuration.
  A different existing selection refuses. No source-copy or migration journal.
- Known obsolete/missing registrations stay visible in the report. An operator can
  explicitly acknowledge one as retired/outside the adopted production inventory;
  that acknowledgment neither deletes its data/identity reservation nor proves a
  process absent. Potentially live or unreadable stores remain unresolved unless
  the operator completes the stop/upgrade and disposition steps. No automatic
  inference from a failed probe. Unregistered legacy paths must be supplied.
- For ambiguous populated stores, guidance is: stop/upgrade all legacy supervisors,
  retain backups of each store and its identity authority, inspect the report and
  explicitly choose which stores remain outside the production inventory. If the
  operator needs their inventories combined, adoption stays refused until a reviewed
  consolidation is performed; #366's recommended scope includes no manual JSON
  editing recipe or automatic merge. The report labels these installations
  UNMIGRATED and identifies the paths and unresolved decisions. No work is discarded.
- Sweep every actual-process test that currently relies on HOME/XDG isolation,
  including continuation/stale-store/host-identity tests. Inject account lookup when
  testing production resolution so the tests cannot take the real account's lease.
  Clear inherited session-artifact variables and verify descendant sentinel safety.

Proposed acceptance revision, contingent on approving refusal-only migration:
replace the captured migration bullet with “A sole legacy store adopts in place
with its supporting roots and identities preserved; ambiguous multiple-store
installations receive an actionable preservation report, remain explicitly
unmigrated, and require operator reconciliation before adoption.” All remaining
Done when bullets remain applicable. This is a scope revision awaiting approval,
not a claim that refusing a conflicting installation completes its migration.
