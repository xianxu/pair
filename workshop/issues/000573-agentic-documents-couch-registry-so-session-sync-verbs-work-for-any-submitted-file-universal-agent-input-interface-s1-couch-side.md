---
id: 000573
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: 'be7948584baebe10fe2a8039655fc41b89c6acdb' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T10:56:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
---

# Agentic documents: Couch registry so session-sync verbs work for any submitted file (universal-agent-input-interface S1, Couch side)

## Problem

Project universal-agent-input-interface (ops `teams/devinfra/projects/universal-agent-input-interface/design.md`,
slice S1): Couch's session-sync verbs (`read`, `diff`, `write`, `status`, the owed-event ledger, the CAS write and the
operator-lock refusal) work only for files with `type: session-sync` frontmatter. The operator wants every markdown
file submitted from Couch's right-pane nvim to become an **agentic document** with the same guarantees, without
writing anything into the file.

## Spec

- A Couch-side **agentic registry** keyed by canonical path: `couch --session-sync register FILE --owner SLOT` (called
  by parley on the first submit), `couch --session-sync forget FILE`, and listing. Agentic status lasts until forgotten.
- All `--session-sync` verbs accept a registered file without `type: session-sync`; the owner comes from the registry
  (or an explicit `couch-owner:`, which wins).
- The owed-event ledger works for registered files; the submit delivery names the file.
- **Forgotten lock:** if the agent needs to write and the operator has held the lock for more than 30 minutes, the
  write refusal says so (lock age), so the agent can ping the operator (or raise a Needs you item for the TL's file).
- A path move or rename updates the registry entry.

## Done when

- Tests: register, then read/diff/write/status on a file with no frontmatter; owed events; lock-age in the refusal
  after 30 minutes (fake clock); forget; rename. Existing session-sync files are unaffected.
- `couch --skill` documents the registry and the 30-minute rule.

## Plan

Detailed implementation plan: [agentic document registry](../plans/000573-agentic-document-registry-plan.md).

- [ ] Build registry model/store and stable document metadata, with idempotent register/forget/rename and read-only membership discovery.
- [ ] Route every sync verb through registration-aware ownership, CAS, owed matching and lock-age refusals, preserving legacy behavior.
- [ ] Expose the agreed Parley CLI/JSON contract and update help, skill and atlas.
- [ ] Verify targeted/race/full suites and operator live check, close review, then land on SHIP.

## Log

### 2026-10-15

Claimed on TL dispatch; read ops project's project.md and design.md before design. Scope is S1 Couch only; TL owns the project file. Sent proposed contract to parley.nvim:2, confirmed submitted, and received its integration requirements (read-only membership, canonical missing-old rename paths, explicit partial-effect outcomes). Existing ledger and CAS are reused (ARCH-DRY); stable document identity keeps rename from losing shadow/folds or old-path submits (ARCH-ORDER). Read-only design exploration independently confirmed these integration points. Concrete plan pending hub approval; implementation has not started. start-plan's attempted weave refresh refused due to the preserved unrelated untracked pensive; it explicitly admitted planning on the existing substrate, with ariadne #385/#386 in flight.

TL approved the concrete plan, with 4096 registrations / 16 MiB registry and explicit forget advice on overflow. No :0 staging for #573: land independently, TL rebuilds, and the combined plain-note live check follows parley.nvim#332 staging and operator restarts. Await Parley contract agreement before implementing the public verbs. Full flow is selected for the planned code volume; no estimate derived before plan-quality.
