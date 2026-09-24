---
id: 000313
status: working
deps: []
github_issue:
created: 2026-09-23
updated: 2026-09-23
estimate_hours:
started: 2026-09-23T19:48:21-07:00
flow: {kind: quick, provenance: inferred, spec: "bf9e0541", done: "e86e2b55"}
---

# Add a slot from the repository thread action menu

## Problem

Adding another slot currently requires retyping a repository path even when its thread is already visible.

## Spec

Add an `Add slot` item to a thread's action menu when its repository root can be
resolved from existing inventory identity. Primary/ordinary rows use the existing
scope-matched root derivation, including starting subdirectories; numbered rows
use their validated primary root. Unknown roots do not offer a guessed target.
The action opens the existing start form, prefilled with that absolute root and
focused on agent selection. It requests the existing StartCreate preview; Enter
submits the accepted fingerprint, Escape cancels without creating anything.
Use current Couch agent/repository defaults, not the selected thread's agent.
Keep parked/admission errors in the existing creation path; no auto-resume or
alternate allocation policy. The ordinary start-from-path entry remains available.
ARCH-DRY/PURE: reuse root derivation and the pure menu reducer/start preview.
No new operation schema, storage, filesystem probes or concurrent workers.

## Done when

- Add slot is available from a known repository's primary or numbered row and targets its primary repository without path typing.
- Existing start form supports agent choice, cancellation and fingerprint-bound creation; repository parked refusal remains enforced.
- Unknown/malformed identity cannot cause a guessed slot target, and same-named repositories use exact paths.
- Combined delivery also includes the separately owned #315 repairs: fresh slots use ordinary new-conversation registration and cleanup, preserving established-only resume/checkpoint behavior; #315 acceptance and review evidence remain in its own issue.


## Plan

- [x] Add failing reducer tests for primary/subdirectory and numbered rows, exact-path disambiguation, cancellation, accepted StartCreate submission and errors.
- [x] Add the action and reuse existing form/preview helpers; retain prior action ordering and avoid adding the action when root identity is unavailable.
- [x] Run couchtty suite/race and relevant core creation-refusal tests; update README/atlas/project, build, close and publish.


## Log

### 2026-09-23

Root lookup and launch form are already implemented; this issue connects them through the thread action menu.

### 2026-09-23 — implemented and verified

Fresh-context spec review approved the shared-form design. New tests failed for
missing Add slot before implementation. Full couchtty passed (6.804s), full UI
race suite passed (19.787s), relevant core create/admission tests passed (8.668s),
and `make pair bin/couch` passed. Exact-root targeting, default agent resolution,
accepted fingerprints, cancellation/late previews, errors and unknown identity
are covered. No new schema or admission policy was added. Close/publication follow.

### 2026-09-23 — smoke-test steering

Filed #314 at the operator's request for optional Ariadne integration; no code
for that task was implemented. Investigated failed pair:2: host/dependency clone
existed but no setup-success marker or conversation. Explicit idempotent CLI
provisioning succeeded (disposition prepared) and wrote the marker, preserving
baseline `7e229bc95b52476929c55f7af2fa3e0c3bc83233`. No agent was launched. Original
failure cause remains unknown pending operator error evidence; captured retry
logs are `/tmp/pair-slot2-provision-diagnostics.log` and result JSON alongside it.

### 2026-09-23 — review scope clarification

The prior review flagged bootstrap/CI gateway changes. Provenance check shows
`aff72f82` (build: adopt ariadne#239 seeded gateway files) already on origin/main;
#313 has no bootstrap/CI diff against origin/main. Preserve this unrelated
upstream work. Fresh-slot runtime smoke failures were repaired separately in
#315 (real claim registration and exact nonce handoff), whose complete-window
review returned SHIP without findings. Add slot itself retains the originally
verified action/form behavior; rerun its boundary review for publication.

## Revisions

### 2026-09-23 — formal combined-delivery scope (BR-1)

The operator reported pair:2 startup failures while #313 was awaiting publication.
The resulting #315 is deliberately a separate bugfix issue, implemented on the
same delivery branch. Revise this boundary's delivery scope to include #315's
already reviewed startup fixes alongside the #313 menu action. The original
"no new operation schema, storage" constraint applies to the Add slot action;
it does not prohibit the separately specified trusted-profile `launch_nonce`
and registration repair in #315. This supersedes the original whole-boundary
interpretation without changing the menu design.

The expanded delivery has two independently specified acceptance contracts:
#313 exact-root menu/preview behavior and #315 reserved-claim plus nonce
registration behavior. #315 was separately reviewed SHIP at
`c6a91fe2..0879c270` (see its close-review artifact), and its final complete
launcher/core/CLI suites and targeted race suites passed. Review both contracts
for this combined publication; neither issue is left as an undeclared side
change. The acceptance criteria above now explicitly include this integration.
Upstream gateway commit `aff72f82` remains outside this delivery's changes
against origin/main; it is not being modified or reverted here.

### 2026-09-23 — combined delivery simplified

Operator requested ordinary new-conversation launch for fresh slots. #315 removes
its superseded registration/nonce additions and fixes only StartFreshSlot routing
plus early argument validation. The active combined acceptance line is updated;
original #313 no-new-operation constraint now also holds across final production
delivery. Historical handshake repair notes above are superseded by #315's latest
specification. Main review baseline now incorporates published upstream changes
while retaining local planning, excluding the unrelated gateway change.
