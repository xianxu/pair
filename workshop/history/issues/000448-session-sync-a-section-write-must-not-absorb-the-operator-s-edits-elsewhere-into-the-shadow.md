---
id: 000448
status: done
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: 'ac832f4442fd12cfd1aa85da82b6050529a103bd' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-11T12:26:20-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "e2304f79", done: "a3b93419"}
actual_hours: 0.29
---

# session-sync: a section write must not absorb the operator's edits elsewhere into the shadow

## Problem

Since pair#439, a section write succeeds while the operator has edits in other sections (by design). But the write then refreshes the shadow copy from the whole file, so those other edits become part of the shadow and `couch --session-sync diff` no longer shows them. Seen 10-11 12:25 on the ops status file: the operator added `🤖[done]` under Needs you and submitted; the TL's Threads write landed first, and the following `diff` was empty. The TL found the marker only by grepping. A marker the agent never sees goes unanswered.

## Spec

The shadow means "the file as the agent last saw it". A section write updates only that section in the shadow, leaving every other section as it was, so `diff` keeps showing the operator's unread edits elsewhere. A `write --all` with a matching whole-file hash replaces the whole shadow (the agent read everything). Folding a submit doesn't change the shadow; reading the diff is the agent's job, and an `--accept-edits`-style acknowledgement isn't reintroduced. Decide in the design whether `read --all` should also refresh the shadow (probably yes: it is the agent seeing the file).

## Done when

- Operator edits section A; the agent writes section B; `diff` still shows the edit to A (test).
- `write --all` with the right hash leaves `diff` empty (test).

## Plan

- [x] Section-scoped shadow update; tests for the Done-when rows; atlas couch.md.

## Log

### 2026-10-11
- 2026-10-11: closed — go test ./cmd/internal/sessionsync green incl. new tests: section write keeps edits elsewhere in diff (Done-when 1), write --all / read --all empty diff (Done-when 2), read --all horizon pin and no-shadow write (both mutation-checked), operator-added section keeps the shadow. Round-1 fixes: SKILL.md diff/read --all rows + unseen-edits bullet, comments, single shadow writer. make -k test: only review-window-test failed in sandbox, passes unsandboxed. go test ./...: launcher failures reproduce on main; TestContinuationWriter fails on inherited live-session pair scope (env), unrelated.; review verdict: SHIP

- Design: `shadowAfter` in `sync.go` replaces only the written section in the old shadow and restamps it; `write --all` stores the whole file. `read --all` also refreshes the shadow (decided yes: it is the agent seeing every edit). Hazard found: a sidecar-less file's owed horizon is the shadow's mtime, so the first shadow `read --all` writes would silently drop owed events. `saveShadow` pins the horizon in the sidecar first (test, mutation-checked). A shadow without the written section (the operator added it) stays unchanged, so `diff` shows noise, never loses an edit. Without any shadow the write stores the whole file, as before.
