---
id: 000729
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'd82605fc78123db4f10591d9d358827156d0567a' # card fields mirrored from issue-cards; edit via sdlc
---

# recover-plan: an enrolled non-repository directory shows as ignored with a reason, not an unavailable fleet

## Problem

`couch --recover-plan-from-sdlc` reports an enrolled directory that is not a git repository as an unavailable fleet whose error is raw `sdlc` output ("resolve fleet: resolve workspace: sdlc workspace --json: exit status 1: … fatal: not a git repository"). On this machine the home directory `/Users/xianxu` is enrolled, because a thread was started there. The row reads like a working-directory failure. The hub read it that way during pair#726, but the report works from any directory and the main fleet reports present.

## Spec

An enrolled directory that is not a git repository appears under the report's `ignored` section with that reason (for example `ignored.not_repositories: [{path, reason}]`, an additive JSON field), not as an unavailable fleet.

Design note from the pair#726 attempt: a filesystem check (an existing directory with no `.git`) is the wrong seam. It bypasses the slot catalog (`SlotWorkspaceResolver`) that the recover-plan tests fake, and three existing tests enroll real temporary directories without `.git` modelled through that fake. Classify through the resolver instead: `ResolveWorkspace` returns a typed not-a-repository outcome (the OS catalog derives it from git's own exit or `rev-parse` result, and the fake models it), and `RecoverPlan` routes that outcome to `ignored`.

## Done when

- A recover-plan test with an enrolled non-repository (typed by the fake catalog) shows it under `ignored` with its reason, and no fleet row for it.
- The OS catalog maps a real non-repository directory to the typed outcome (test with a temporary directory).
- Existing recover-plan tests unchanged.

## Plan

- [ ]

## Log

### 2026-10-20
