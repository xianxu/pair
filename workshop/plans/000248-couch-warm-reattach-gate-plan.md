# Warm reattachment and shared recovery contract implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Execute this session's warm-context core work locally; delegate bounded acceptance work after the implementation gate. Steps use checkboxes for tracking.

**Goal:** Make a uniquely owned live detached thread selectable and attachable without a native conversation binding, with no cold-launch fallback.

**Architecture:** Keep thread address, process ownership, and conversation source separate. Inventory, execution and its final recheck share one pure warm-session proof matcher. Reuse the existing tracked start transaction, warm-only operation argument, stateful external fakes, and Console attachment.

**Tech Stack:** Go, Couch core and terminal reducer, existing Pair/Zellij process seams.

## Coordinated design: #248, #249, #250

The operator approved this direction and authorized design and implementation on
2026-09-14, stopping after the first independently smoke-testable issue.

| Operation | Authority | Outcome |
|---|---|---|
| Reattach | Exact address/session ownership, live server, zero clients | Preserve running agent |
| Native resume | Verified native conversation binding and safe launch ownership | Restart existing conversation |
| Continue | Exact available checkpoint and safe launch ownership | Fresh conversation seeded from checkpoint |
| Archive | Reconciled ownership plus explicit action | Remove working-set row, preserve history |

An attempted operation never silently substitutes another source. Inspection
does not kill processes or mutate ownership. Unknown evidence is not death.

#248 delivers reattachment eligibility and preserves selected warm intent.
#249 owns a durable pending continuation request and the normal/retry execution
path. It must separate existing-address registration from fresh/native/checkpoint
conversation initialization, preserve exact checkpoint contents through final
prompt delivery, and retain the request across failed replacement. Reuse the
existing tracked fresh-existing launch lifecycle; decide helper survival and
checkpoint retention in #249's own detailed plan before implementation. A slug,
or a basename reconstructed relative to the launcher's checkout, is insufficient.
`FreshRequired` currently clears continuation fields, and marker consumption
currently precedes successful replacement: neither is an acceptable shortcut.

#250 owns exact stale-owner reconciliation and recovery choices, calling #248's
reattachment and #249's continuation executor. It must not build a second
launcher. Existing `RetireIncarnation` handles proven-dead live records without
open transactions; unknown/creating/open transaction cases need explicit
reconciliation through their owning lifecycle. Manual retirement already removed
the reported stale incarnation; usable conversation recovery still needs fresh
verification and operator confirmation.

This document implements only #248. #249 and #250 retain separate completion
boundaries and receive executable plans after the #248 smoke test.

## Core concepts

| Name | Kind | Lives in | Status |
|---|---|---|---|
| DetachedSessionObservation | PURE | cmd/internal/couchcore/actionableinventory.go | modified |
| DetachedCandidate | PURE | cmd/internal/couchcore/artifactcollision.go | modified |
| SessionNameBinding | PURE | cmd/internal/couchcore/detachedsessions.go | modified |
| detachedResumeProofMatches | PURE | cmd/internal/couchcore/actionableinventory.go | modified |
| ClassifyThread | PURE | cmd/internal/couchcore/actionableinventory.go | modified |
| ProjectDetachedSessions | PURE | cmd/internal/couchcore/detachedsessions.go | modified |
| dispatchMenuOperation | PURE | cmd/internal/couchtty/menu.go | modified |

Remove NativeID from the three warm-only evidence shapes. Retain address,
session name, and launch-agent correlation; a candidate is a query, and only
ProjectDetachedSessions' uniquely owned detached-session observation is proof.
Keep the matcher name and record argument, sharing it across classification,
initial execution and post-claim recheck. It checks supported profile, non-nil
argv, exactly one observation, exact address/agent and nonempty session name.
It does not check record occupancy: callers own that stage-specific condition,
and recheck occurs after claiming a creating incarnation.

| Name | Kind | Lives in | Status | Wraps |
|---|---|---|---|---|
| gatherThreadEvidence | INTEGRATION | cmd/internal/couchcore/actionableinventory.go | modified | ThreadStore, path, binding/session resolvers |
| ResumeContextWith | INTEGRATION | cmd/internal/couchcore/resume.go | modified | Session proof, tracked ownership and helper launch |
| confirmStillDetached | INTEGRATION | cmd/internal/couchcore/resume.go | modified | Final session observation |
| StartInteractive | INTEGRATION | cmd/internal/couchcore/startup.go | modified | Inventory selection to resume |
| ScopedThreadArtifactCollisionChecker.DetachedSessions | INTEGRATION | cmd/internal/couchcore/artifactcollision.go | modified | Session index and Zellij snapshot |
| FakeThreadArtifactCollisionChecker.DetachedSessions | INTEGRATION | cmd/internal/couchcore/artifactcollision_fake.go | modified | Stateful portable session/binding fake |

Existing FakeEnvironment/FakeChildRunner and CouchTTY test console are reused.
Native binding resolution moves inside the parked branch, including interface
availability checks. Warm classification/execution succeeds without that
interface and performs zero ResolveEstablished calls. Parked behavior keeps its
existing resolution/refusal contract. Startup and foreground menu selection
carry WarmOnly when the selected row is detached, using the existing operation
argument; background reattachment already does so.

## Architecture and operating envelope

- ARCH-DRY/PURE: one warm proof matcher; pure matching and menu intent, thin IO
  gathering. No general action framework or new persisted lifecycle state.
- ARCH-PURPOSE: cross inventory, Enter, operation dispatch and final Console
  attachment. Sweep startup, background and foreground entrypoints, diagnostics,
  README, atlas and current tests for the old binding requirement.
- ARCH-MOCK: reuse fake session ownership maps and real ProjectDetachedSessions;
  real portable ThreadStore and fake acknowledged helpers share production
  orchestration. Use existing Zellij conformance tests for occupied/dead/contested
  sessions; the operator smoke verifies actual installed-session attachment.
- ARCH-CONSTRAINTS: inventory/startup interaction, same candidate-bounded
  snapshots/timeouts as today; zero native resolution for warm candidates is a
  counted requirement. No added polling, concurrency, background tasks or disk
  artifacts. Cancellation retains existing bounded tracked-start cleanup.
- ARCH-SECURE: persisted thread validation and session-index parsing remain at
  existing boundaries. Failed/ambiguous observations grant no attachment. Do not
  infer native authority from warm success. All automated tests use temporary
  stores/fake process identities, never production runtime state.
- ARCH-FUNERAL: no new artifact family; existing creating/live incarnation and
  helper cleanup remain owned by tracked start. Preserve the external session
  on any failed warm attachment because this attempt did not create it.

## Ordering contract (ARCH-ORDER)

| State/event | Result/effects |
|---|---|
| Detached row selected | Dispatch warm-only resume |
| Selected row becomes parked before execution | Refuse before claim/helper; no cold resume |
| Session absent, occupied, ambiguous or query fails before claim | Refuse; no helper |
| Record changes before claim | Revision conflict; no helper |
| Session disappears, becomes occupied/ambiguous, or name changes after claim | Shared matcher plus original session-name comparison refuses; rollback own claim |
| Valid recheck | Existing tracked warm launch, registration and Console attachment |
| Session dies after final check | Pair's warm resume refuses; tracked warm cleanup preserves anything surviving |
| Cancellation/helper failure | Existing tracked-start rollback/retire/unknown policy, never kill preexisting session |

Use deterministic fake hooks at observation/claim/acknowledgement to select these
orderings; no timing sleeps as race oracles. Concurrent resume is serialized by
the existing revision/nonce start claim. No new synchronization owner is added.

## Chunk 1: #248 implementation

### Task 1: Reproduce the delivered failure

Files: `cmd/internal/couchcore/warmresume_test.go`,
`cmd/internal/couchcore/classify_test.go`,
`cmd/internal/couchcmd/warm_reattach_test.go` (new acceptance tests using
`newRT`, `seedDetachedThread`, and production `wireResolver`).

- [ ] Add a stateful unbound detached fixture and assert inventory yields a
  detached row. Pass that real row through menu Enter, the declared operation,
  and Console attachment; assert same session, warm shape, bare resume argv,
  no native binding fabricated, no quiesce/fresh launch.
- [ ] Add zero-resolution and missing binding-interface tests for warm inventory
  and resume, alongside cold refusal controls. Add malformed/duplicate/mismatched
  proof tests and deterministic post-claim loss/rebinding tests.
- [ ] Run focused tests and record expected red failures from the old gate.

### Task 2: Share warm proof and preserve intent

Files: the core concept files above; associated detachedsessions,
actionableinventory, startup_proof, warmresume and artifactcollision tests.

- [ ] Delete NativeID only from warm evidence shapes and all their producers/
  fixtures. Keep native parked/resume binding types unchanged.
- [ ] Move native resolver availability/resolution inside the cold branch in
  gathering and execution. Reuse detachedResumeProofMatches in all three sites.
  Final recheck additionally compares the original observed session name.
- [ ] In dispatchMenuOperation, set `warm-only=true` for a resume whose selected
  row is detached. In StartInteractive, derive WarmOnly from its selected row
  before calling ResumeContextWith. Preserve cold selection behavior.
- [ ] Update old tests which intentionally pinned binding-lost for an otherwise
  valid detached observation. Add foreground/startup stale-selection tests;
  retain session ownership/active-client and existing cleanup suites.
- [ ] Run `go test ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd -count=1`;
  fix regressions and commit issue-scoped implementation with model attribution.

### Task 3: Verify and hand off for smoke

Files: `README.md`, `atlas/couch.md`, issue log and this plan.

- [ ] Sweep current docs/comments for warm native-binding requirements; explain
  warm access does not establish durable native recovery. Invalid warm evidence
  reports uncertainty, not binding loss. Broader stale-helper diagnosis is #250.
- [ ] Run `env -u PAIR_SESSION_ID -u PAIR_TAG make test`, focused race tests,
  `git diff --check`, and build `make build`. Record exact evidence.
- [ ] Close through `sdlc close --issue 248 --verified '<evidence>'`, fix blocking
  fresh-context review findings, and leave the reviewed build available in the
  operator's checkout. Stop before #249 implementation for the requested smoke.

Smoke: use the built Couch to select an exactly owned detached thread with no
native binding, confirm its existing prompt/conversation survives attachment,
detach and reattach once more. Do not restart a production session for this
test. Record operator result before moving to #249.
