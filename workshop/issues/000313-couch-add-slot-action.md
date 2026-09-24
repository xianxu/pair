---
id: 000313
status: working
deps: []
github_issue:
created: 2026-09-23
updated: 2026-09-23
estimate_hours:
started: 2026-09-23T19:48:21-07:00
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


## Plan

- [ ] Add failing reducer tests for primary/subdirectory and numbered rows, exact-path disambiguation, cancellation, accepted StartCreate submission and errors.
- [ ] Add the action and reuse existing form/preview helpers; retain prior action ordering and avoid adding the action when root identity is unavailable.
- [ ] Run couchtty suite/race and relevant core creation-refusal tests; update README/atlas/project, build, close and publish.


## Log

### 2026-09-23

Root lookup and launch form are already implemented; this issue connects them through the thread action menu.
