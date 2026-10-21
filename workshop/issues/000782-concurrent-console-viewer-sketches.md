---
id: 000782
status: open
deps: [pair#752]
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '6e6b3a77f77b0d12e003a70cd4df24303184de35' # card fields mirrored from issue-cards; edit via sdlc
---

# Concurrent console and viewer sketches

## Problem

The approved product direction permits the console and current viewer driver to sketch concurrently. #752 deliberately ships the existing single-holder model for prompt usability, with console admission independent of viewer annotation permission. TL deferred simultaneous lanes and scoped clearing from #752 on 2026-10-21.

## Spec

Provide two bounded drawing lanes: one console and one current viewer driver. Each owns its sketch identity, open stroke, undo/redo stacks, source lease and expiry. Existing ordered writer serializes events. Viewer A→B→A epochs invalidate stale viewer work only; console retains independent cast/context lifetime. Pause/navigation/end affect both; handoff affects viewer only. Turning viewer annotation off clears viewer sketches while preserving console sketches. Review pointer/note cleanup separately; do not silently change unrelated annotation behavior.

Existing v1 logs implicitly interrupt operator sketches on ctl handoff, without explicit sketch-interrupt records. New readers must preserve those legacy semantics; simply allowing all operator histories across handoff reinterprets historical recordings. Use an explicit new recording-version boundary for independent-console semantics, with old-reader refusal and unchanged v1 replay. This compatibility choice requires design approval before implementation.

## Done when

- Console and current driver draw concurrently, with independent undo/redo, labels, sources and expiry; no overlapping identity or ownership.
- Viewer handoff and annotation-off fence/clear viewers without retiring console work; pause/navigation/end remain correct for both.
- Legacy v1 replay remains unchanged; new-version Go/JS reducers, snapshots, replay seeks and catch-up agree, and older readers refuse new semantics.
- Controlled concurrency, bounded-resource and actual two-surface Chrome checks prove the contract before landing.

## Plan

Design first, outside the quick shell. Audit these singleton seams: recording/sketch_state.go and web/sketch-model.js global active and author checks; recording/sketch_snapshot.go and JS restore exactly-one-active constraints; Go BeforeChange/Change and web/sketch-render.js record-display adapter single open stroke; board/sketch_delivery.go contextual deltas; board/sketch.go global source/handle/sourceSketch/lastMutation/leaseVersion and busy[begin]; drive/drawing.go operator permission currently shares viewer epoch/witness; cast.go admission and chrome_operator_sketch_wiring.go/operator-sketch.js permission projection.

Replace active/source/begin reservations with bounded lane ownership, not unbounded concurrency. Keep one timer scheduled to earliest of at most two lease deadlines and compare per-lease renewal versions at final expiry dispatch. Existing per-source limits are16MiB prefix and8MiB state; budget two retained sources explicitly. Existing display store, patch grammar, painter and ordered writer already support multiple sketch identities; reuse them.

Regression strategies: interleaved console/viewer begins/points/undo/redo/labels; delayed begin ACK/refusal/unknown sink outcomes cannot cross-install sources; A→B→A plus late viewer callbacks while console contact continues; pause/navigation/end with two open strokes updates both delivery and replay; expiry/renewal of either lease leaves the other correct; shared Go/JS prefix corpus and two-active snapshot/catch-up/seek; historical operator→handoff→later-operator fixture; real console/viewer pointer input and private-scene exclusion.

## Log

2026-10-21: filed unclaimed at TL direction22603f68-f45b-4ba9-99a6-428ed9bff6e1. Read-only audit during #752 confirmed that this is a material reducer, source lease, delivery, authority and persistence redesign, not a boolean permission change. No concurrent-lane implementation landed in #752. ARCH-DRY: reuse ordered writer/display; ARCH-ORDER: scope transitions to lanes; ARCH-EVOLVE: explicit version boundary protects implicit legacy handoff semantics; ARCH-CONSTRAINTS/FUNERAL: at most two leases with bounded storage and independent retirement.
