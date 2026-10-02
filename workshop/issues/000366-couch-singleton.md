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
