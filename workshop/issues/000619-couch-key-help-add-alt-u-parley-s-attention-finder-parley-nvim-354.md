---
id: 000619
status: codecomplete
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: '18e4392137d3e4d6bf0b359d5ab9cc2fe7571ec3' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T18:18:08-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "210d1afc", done: "c00501e2"}
actual_hours: 0.31
---

# Couch key help: add Alt+Space, parley's attention finder (parley.nvim#354)

## Problem

parley.nvim#354 (the attention finder, part of ops project universal-agent-input-interface) binds **Alt+Space** (M-Space) in
parley to open the list of documents waiting on or answered by an agent. Couch's key help
(`cmd/internal/keyhelp/catalog.go`) doesn't list it, so the operator can't discover it from Couch.

## Spec

- Add one entry for Alt+Space, "parley: attention finder (documents waiting on / answered by an agent)", in the section
  where parley's editor keys are listed, matching the existing entries' style.
- Confirm Alt+Space is not bound by Couch or pair's zellij config (verify for the revised key during implementation) and keep it that way: if the
  catalog has a uniqueness test, it covers the new entry.

## Done when

- The key help shows Alt+Space with that description; the catalog's existing tests pass (plus the uniqueness check).

## Plan

- [x] Add a red rendered-help regression for Alt+Space and Ctrl+Y then Space, uniquely under Parley editor, with no stale Alt+u.
- [x] Add explicit companion-editor help metadata/context and curated rows; retain Pair source drift checks, without key handlers or peer-repo changes (ARCH-DRY/ARCH-PURE). Verify Couch/Zellij do not intercept either key.
- [x] Run package tests and recorded full suite; submit the verified result to close review and normal landing.

## Log

### 2026-10-16
- 2026-10-16: closed — Red-green rendered help regression; keyhelp/keyscmd and Couch legacy/Kitty pass-through tests passed. Recorded full suite on f500a1cc passed exit 0 in 12m46s, log SHA256 6746e0ae12fd, no failures or retries. Reviewer independently passed four affected packages. BR-1 fixed across README key table and the old universal source-derivation claim; atlas and source metadata agree on Parley-local scope. Later changes are prose only. No key handlers or actual operator environment changed.; review verdict: SHIP

### 2026-10-16 — red/green help metadata

The new standalone/hosted rendered-help regression failed before implementation because both keys appeared zero times. Two curated SourceParley/ContextParley rows now render exactly once under Parley editor, with the requested description, no Pair chord and no stale Alt+u. Complete keyhelp and keyscmd packages pass (0.146s / 0.341s). Pair-owned source drift checks remain intact; companion metadata is explicitly documentation rather than a fabricated Pair handler. A Couch interceptor test covers legacy and Kitty encodings of both keys. Read-only inspection finds no Alt+Space or Ctrl+Y binding in Pair's Zellij config; Couch's Ctrl+Y copy action is switcher-local, and the existing focused-child routing test still applies.

Start-plan succeeded despite its non-blocking catch-up warning: weave refresh rejected the unsupported --defer-settings flag; SDLC explicitly allowed planning on the current dependencies. No machine configuration was changed.

### 2026-10-16 — recorded full suite passed

The first recorded suite on f500a1ccf3b165f070d40a2b5478605b6a02b48b completed exit 0 in 12m46s, including all shell/editor and Go checks (keyhelp 0.150s, keyscmd 0.434s, couchtty 51.072s, couchcore 422.550s). No failures or retries. Log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/f500a1ccf3b165f070d40a2b5478605b6a02b48b/suite.log, SHA256 prefix 6746e0ae12fd; recorded as sdlc-local/suite. Only this issue-log/checklist update follows the tested code. The close review and normal landing are next; no staging or live-input change is needed for documentation-only help rows.

### 2026-10-16 — close review README correction

Round 1 found no code defect and independently passed the four affected packages, but its Important BR-1 blocks finalization: README omitted both companion-editor bindings and claimed every help row derives from an implementing keymap. Added both Parley-local shortcuts to the key table and qualified the derivation claim for curated external metadata. Added the general source-boundary lesson. This is a prose-only correction after the passing recorded suite; no wording-presence test or repeat suite is warranted. Running ordinary close again because round 1 did not finalize.

### 2026-10-16 — review SHIP and independent timeout

Round 2 disposed BR-1 and returned SHIP with no new findings. The reviewer's couchtty package run observed unchanged TestWorkbenchPerformanceIdleUITick timing out; its unchanged isolated repeat passed alongside the relevant routing/pass-through tests. Existing open #626 records that same activity-worker timeout. The review sidecar retains both outcomes; this is independent from help metadata, and no timeout or fixture change is folded into #619. The first recorded full suite above passed without retries.

## Revisions

### 2026-10-16 — operator selects Alt+Space

TL delivery c16d5cbf-dd0c-488d-b9dd-82f71d275f9c changes the attention finder from Alt+u (M-u) to Alt+Space (M-Space), per the operator. This supersedes the original key in the Problem, Spec and Done when; the old Alt+u conflict check does not establish that Alt+Space is free. Verify the revised key and implement/test its catalog entry only after #604 lands, preserving the assigned order.

### 2026-10-16 — both attention-finder bindings

TL delivery 309fc964-4631-45a3-8423-9825a462ac4c extends the documented keys to both Alt+Space and <C-y><Space> (Ctrl+Y followed by Space). This supersedes the single-key Spec and Done when: show both as Parley editor bindings, not Pair-global bindings. The catalog has no Parley source today; a small explicit companion-editor metadata source preserves its named-source rule without importing a peer repo or inventing executable bindings.
