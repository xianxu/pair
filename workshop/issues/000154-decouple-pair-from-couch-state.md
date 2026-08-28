---
id: 000154
status: working
deps: []
github_issue:
created: 2026-08-27
updated: 2026-08-27
estimate_hours: 3.35
started: 2026-08-27T16:23:37-07:00
---

# decouple Pair from Couch state

## Problem

Pair is the lower-level wrapper around coding harnesses. Couch is an integrated
workspace shell which hosts Pair, but Couch's configuration and durable thread
model must not become prerequisites for using Pair directly.

The #149 implementation crossed that boundary: every direct Pair launch reads
the Couch thread-store manifest, and ordinary Pair launches register themselves
into that store. A valid Couch manifest written after M5 added
`legacy_migration_version`; Pair's private strict projection did not add the
field, so `pair` now exits before launching any harness. Under Couch this appears
secondarily as an `await Pair registration` timeout.

## Spec

Pair owns harness wrapping, Pair tags, native agent-session continuity, Zellij
session bindings, and Pair's tag-scoped artifacts. It must launch, list, attach,
resume, rename, and continue without reading, validating, or writing Couch's
namespace, manifest, thread records, names, descriptions, policy, or lifecycle
state.

Couch exclusively owns its durable work-thread inventory and human metadata.
It resolves human names and paths itself, then invokes Pair using the exact
opaque Pair tag plus ordinary launch inputs. Pair does not resolve Couch human
names. `pair resume <tag>` remains supported; `pair resume <Couch human name>`
is deliberately removed.

Pair does not classify tags by origin or readability. A user-assigned tag such
as `compiler-fix` and a Couch-generated tag such as
`couch-3dcfba1308775e82` are both ordinary valid Pair tags and both resume by
exact equality. The separate mutable human name stored on a Couch work thread
is not a Pair tag and therefore resolves only through Couch.

Couch may pass one-shot process inputs needed to host Pair—currently the exact
thread scope/tag and launch profile—and may pass opaque Couch environment such
as `COUCH_TREE` and `COUCH_STORE_DIR` through Pair to the hosted harness and its
tools. Pair may transparently propagate those values but must not interpret,
validate, open, or mutate the Couch namespace or schema. New Couch configuration
fields must not require corresponding Pair configuration changes. If a future
Couch feature cannot be built without widening Pair's lower-level invocation
contract, that boundary change requires operator consultation first.

Hosted readiness uses two independent authorities. Pair owns and publishes its
existing tag-scoped address claim when Pair has accepted the exact Couch-reserved
scope/tag and established that Pair address. This is address-registration
evidence, not proof that every sidecar or the Zellij workbench is already
running. Couch observes that Pair-owned evidence and, only after validating the
expected exact scope/tag and helper process identity, promotes its own creating
incarnation in `ThreadStore`. Missing, malformed, unreadable, mismatched, or
still-reserved Pair evidence fails Couch start closed and leaves Pair entirely
unaware of Couch persistence. Pair never promotes or registers a Couch thread.

Remove the standalone-Pair-to-Couch registration path and the portable Couch
thread-index reader from the Pair launcher. Couch retains name/path resolution
against its own `ThreadStore`; shared generic Pair utilities may remain only
where they carry no Couch persistence model. Keep strict JSON validation on
Couch-owned persisted state inside Couch.

The regression boundary is process behavior. For every direct command family
(`launch`/attach/resume picker flow, `list`, `rename`, `continue`, `restart`, and
`quit`), an IO-spy or denied-store runtime proves no operation opens the Couch
namespace; launch/create also proves it creates no Couch thread record. Seeded
valid-forward, malformed, unreadable, and missing Couch stores must be
observationally irrelevant to Pair. Separately, a Couch integration test proves
start passes its exact opaque tag to Pair, Pair publishes only Pair-owned
readiness, and Couch alone completes registration in `ThreadStore`. This keeps
the core launch decision pure and tests filesystem integration through the
existing injected runtime seams (ARCH-PURE, ARCH-PURPOSE).

## Revisions

### 2026-08-27 — make readiness and opaque environment ownership explicit

**Reason:** fresh spec review found that removing the thread-index coupling
could accidentally remove Couch's fail-closed readiness proof, or reintroduce
the coupling through environment propagation. It also found that one successful
launch would not prove the stated independence of every direct Pair command.

**Delta:** Pair retains its Pair-owned tag-scoped readiness artifact; Couch
observes it and exclusively promotes `ThreadStore`. Pair may pass opaque Couch
environment to hosted children but cannot interpret Couch persistence. The
regression sweep now enumerates every direct command family and denies Couch
namespace IO, including malformed and unreadable stores (ARCH-PURPOSE).

### 2026-08-27 — preserve the existing address-registration commit point

**Reason:** second fresh review found that “workbench established” overstated
the current evidence. Pair establishes the reserved address before later
sidecars and Zellij startup.

**Delta:** the spec now defines the artifact narrowly as proof that Pair
accepted and established the exact reserved scope/tag. Couch's existing
fail-closed promotion remains keyed to that evidence plus the expected helper
identity; this issue does not move the handshake commit point.

### 2026-08-27 — distinguish Pair tags from Couch human names

**Reason:** operator review clarified that human-assigned Pair tags remain
semi-readable and resumable even though Couch also generates opaque hexadecimal
tags.

**Delta:** Pair resumes every valid tag by exact equality regardless of its
origin. Only Couch's separate mutable thread-name attribute is Couch-only.

## Done when

- Direct `pair` commands do not read, validate, create, or mutate Couch thread-store state.
- A Couch manifest containing `legacy_migration_version` cannot prevent direct Pair from launching.
- Couch resolves names/paths itself and starts Pair with an exact opaque tag.
- Pair publishes only Pair-owned readiness; Couch alone promotes or rejects its thread incarnation.
- `pair resume <tag>` continues to work; Couch human-name resolution is available only through Couch.
- Both human-assigned and Couch-generated Pair tags resume by exact equality.
- Automated tests cover both direct Pair independence and Couch-hosted registration.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. The calibration source is currently marked
stale by `sdlc estimate-source`, so the derivation is provisional.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=1.20 impl=0.08
item: cross-cutting-refactor design=0.16 impl=0.20
item: cross-cutting-refactor design=0.16 impl=0.16
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.20 impl=0.08
item: milestone-review design=0.10 impl=0.20
design-buffer: 0.15
total: 3.35
```

## Plan

- [x] Write and approve the durable implementation plan.
- [x] Remove Couch persistence dependencies from Pair's launcher under failing regression tests.
- [ ] Keep Couch-owned lookup and hosted registration working under integration tests.
- [ ] Update the atlas and project record to state the corrected layer boundary.

## Log

### 2026-08-27

The initial nested-Zellij hypothesis came from a controlled reproduction inside
the current hosted Pair session and did not describe the operator's shell.
Running `couch start --no-console` exposed the real child error: Pair rejected
Couch's valid `legacy_migration_version` manifest field. Root-cause tracing found
two private manifest schemas and, more fundamentally, that direct Pair always
consulted Couch state. The operator clarified that Pair is the stable lower
layer and approved losing direct `pair resume <Couch human name>` resolution.
The earlier shared-schema proposal was withdrawn; this revision removes the
dependency instead (ARCH-DRY, ARCH-PURPOSE).

### 2026-08-27 — implementation plan approved

The durable plan passed three fresh-context review rounds. The final design
retains permanent public-entry and every-command Couch-independence regressions,
uses production Pair marker IO in the composed Couch handshake, and enumerates
all current documentation consumers (ARCH-PURPOSE, ARCH-PURE, ARCH-MOCK).

### 2026-08-27 — Couch-local resolution complete

`ResolveThreadReference` now operates directly on Couch `ThreadRecord` values;
the launcher projection is no longer part of Couch name/path resolution. Focused,
full-package, race, and source-catalog checks passed after spec and quality review
(ARCH-DRY, ARCH-PURE).

### 2026-08-27 — public Pair boundary regression red

The built public entry now has a permanent all-command/store matrix. A FIFO
tripwire proves namespace reads, recursive state snapshots prove durable Couch
changes, and deterministic sidecar ownership keeps the harness isolated. The
tests fail only at the current ThreadIndex read and standalone registration
couplings; both review stages passed after hardening process cleanup and ambient
environment control (ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-27 — Pair inventory reads removed

The launcher no longer contains Couch `ThreadIndex` types, filesystem reads,
name/path resolution, picker decoration, or historical Couch-name state. Exact
identity now applies consistently to resume, rename, and restart rename re-entry,
while `📁` inversion remains Pair-owned. Launcher/Couch suites, race checks,
catalog contracts, and both per-task reviews passed (ARCH-DRY, ARCH-PURPOSE).

### 2026-08-27 — standalone Couch writes removed

Direct Pair no longer composes or exposes a Couch registrar, store directory,
or standalone thread upsert. The built public matrix is now green in all 32
command/store combinations while opaque Couch environment still reaches the
hosted child. Pair's address-claim timing and Couch's own promotion path remain
unchanged; full and race suites plus both reviews passed (ARCH-PURE, ARCH-MOCK).
