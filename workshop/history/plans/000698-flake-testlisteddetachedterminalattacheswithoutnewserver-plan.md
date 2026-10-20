---
collaboration-mode: discussion
---

# Terminal fixture readiness publication plan

> **For agentic workers:** Consult AGENTS.md Section 3. Execute this small, atomic change in the current session; SDLC close owns the fresh review.

**Goal:** Remove #698's proven readiness/listing publication race without changing production lifecycle behavior.

**Architecture:** Publish the fixture listing marker first and the agent-ready record last through one small test-only ordered IO helper. A filesystem-backed observer queries the real fixture binary at the boundary after the first atomic write, forcing the problematic interleaving without sleeps or racing a deadline.

**Tech Stack:** Go tests, existing osfs.FS atomic writes, existing private Zellij fixture and reply generator.

## Evidence and choices

The retained failure is a 0.45s cold handoff refusal, not a wait timeout. The wrapper writes agent-ready before listing-ready. Production admission sees agent-ready and makes a single listing query; the fixture may still answer no sessions. An investigation overlay held precisely this gap until that query observed it: existing target red in 0.37s, unchanged control green in 0.77s. No production edits were made.

Recommend fixing publication order. Merely waiting for listing-ready in the parent cannot stop concurrent admission from querying earlier. Retrying the query or lengthening a deadline would change production policy and hide the invalid fixture state. Collapsing all readiness markers would change unrelated fixture contracts without need.

## Core concepts

There is no new pure domain entity: this change orders two IO publications. The invariant is that agent-ready implies fixture listing-ready; it is tested through the existing consumer rather than a second policy implementation.

| Integration | Lives in | Status | Wraps |
|-------------|----------|--------|-------|
| publishRegistrationFixtureReadiness | cmd/internal/couchcore/terminal_registration_fixture_test.go | new | two atomic writes using a minimal WriteAtomic interface, implemented by existing osfs.FS |
| registration readiness observer | cmd/internal/couchcore/terminal_registration_reply_test.go | new | real temporary filesystem plus an interposed stateful writer; invokes the real fixture list-sessions between writes |
| registrationTerminalFixture | cmd/internal/couchcore/terminal_registration_fixture_test.go | modified | delegates its readiness publications to that helper |

ARCH-DRY: reuse osfs.FS and the canonical fixture reply generator; no parallel listing parser or production API. ARCH-PURE: these are IO test seams, not pretend pure entities. ARCH-MOCK: the double stores actual files and exposes the same fixture command as existing tests.

## Ordering and bounds

| Publication state/event | Observation | Next action | Loss |
|-------------------------|-------------|-------------|------|
| Neither marker, first write fails | no readiness | return error | fixture startup only |
| Listing marker published, before final write | real list-sessions names the session; agent-ready absent | publish agent-ready | none |
| Final write fails | listing remains; agent-ready absent | return error | fixture startup only |
| Both published | admission may query and verify the session | existing cold detach then warm attach | none |

The observer runs synchronously after the first write; order comes from the call stack, not scheduler speed. Restore the old write order to prove a wrong observation immediately. Error cases must preserve the original write error and never announce final readiness. Process lifetime, socket identity, leases and cancellation stay unchanged (ARCH-ORDER).

Only owned temporary files and fixture subprocesses are used. Existing test environments and cleanup own their end; no operator state or real Zellij session (ARCH-SECURE, ARCH-FUNERAL). Two writes and a fixed number of observations per specimen; no new hot path, workers, buffers or persistent formats (ARCH-CONSTRAINTS, ARCH-EVOLVE). ARCH-PURPOSE: retain all existing detached reuse assertions; do not replace the end-to-end test with a helper-only check.

## Tasks

- [x] Add the ordered-publication helper initially preserving the old order, and route the existing wrapper through it. Add the filesystem observer regression: after the first write run the real fixture list-sessions with both supported listing forms, require the session live and agent-ready absent; after completion require the exact ready record. Inject first/second publication failures and check retained state and error.
- [x] Run `go test ./cmd/internal/couchcore -run 'TestRegistration.*Readiness' -count=1 -v`. Confirm wrong-listing/premature-readiness red, never a timeout. Swap to listing first, agent record last; rerun green. Reverse only the order in an overlay and require the same deterministic red.
- [x] Run existing detached attach and registration/environment fixture tests, plus artifactpath/testpair/termcmd guards. Log red/green evidence and add the publication rule to workshop/lessons.md. This is domain-specific semantic ordering; ask the hub to approve executable enforcement instead of a Go AST lint.
- [ ] Push, record the required suite, close with SDLC's fresh review, and land on SHIP. Verify clean resting workspace, exact merge on main and #698 absent from active registry; report to hub. No :0 staging or operator restart is needed for this test-only fix; private fixture conformance is the integration proof.
