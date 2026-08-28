---
id: 000154
status: working
deps: []
github_issue:
created: 2026-08-27
updated: 2026-08-27
estimate_hours:
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

Couch may pass one-shot process inputs needed to host Pair—currently the exact
thread scope/tag and launch profile—and may pass opaque Couch environment such
as `COUCH_TREE` and `COUCH_STORE_DIR` through Pair to the hosted harness and its
tools. Pair may transparently propagate those values but must not interpret,
validate, open, or mutate the Couch namespace or schema. New Couch configuration
fields must not require corresponding Pair configuration changes. If a future
Couch feature cannot be built without widening Pair's lower-level invocation
contract, that boundary change requires operator consultation first.

Hosted readiness uses two independent authorities. Pair owns and publishes its
existing tag-scoped readiness/address artifact after the Pair workbench is
established. Couch observes that Pair-owned evidence and, only after validating
the expected exact scope/tag and process identity, promotes its own creating
incarnation in `ThreadStore`. Missing, malformed, unreadable, mismatched, or
unestablished Pair readiness fails Couch start closed and leaves Pair entirely
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

## Done when

- Direct `pair` commands do not read, validate, create, or mutate Couch thread-store state.
- A Couch manifest containing `legacy_migration_version` cannot prevent direct Pair from launching.
- Couch resolves names/paths itself and starts Pair with an exact opaque tag.
- Pair publishes only Pair-owned readiness; Couch alone promotes or rejects its thread incarnation.
- `pair resume <tag>` continues to work; Couch human-name resolution is available only through Couch.
- Automated tests cover both direct Pair independence and Couch-hosted registration.

## Plan

- [ ] Write and approve the durable implementation plan.
- [ ] Remove Couch persistence dependencies from Pair's launcher under failing regression tests.
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
