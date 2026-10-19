---
id: 000664
status: working
deps: [ariadne#437]
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'aff618176bc191dba82cb39adc524b1ae18aca53' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T18:18:27-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "28a471c7", done: "051474e2"}
---

# Retire the review pane legacy handoff and nvim apply after ariadne#437

## Problem

## Spec

This is the second step of pair#661 M2, ordering (a), approved by the TL on 2026-10-18. pair#661 M2 moves the review pane onto `pair review apply`, but keeps the legacy handoff path working, because agents running xx-fix from before ariadne#437 still apply through it. Once ariadne#437 lands (the xx-fix workbench protocol on `pair review apply`), retire the legacy path:
- the handoff and landed halves of `nvim/review/handoff.lua`, `PAIR_REVIEW_HANDOFF_PATH` and `PAIR_REVIEW_LANDED_PATH` (artifactpath), and `did_land`;
- restore_controller's pending-handoff and pending-landed checks;
- `apply.apply` (the buffer apply), `reconcile.lua`, `gate.lua` and on_defer, pending_records, has_pending, the winbar pending state, and `set_base`/v0;
- the `agent_applied` poke, the dropped and conflict notifications, and the tests for all of these (review-reconcile, review-handoff; Lua reconcile, gate, handoff_acceptance, handoff_context).

Add the rule test: nvim never writes the reviewed file or buffer for an agent round, only for the operator's own actions (save on send, marker accept/reject, comments, definitions).

The design is in pair#661's plan, `workshop/plans/000661-s4-one-edit-path-apply-and-derived-rounds-plan.md`, under "M2 as built".

## Done when

- None of the retired code or env vars remains, and `make test-review` and `make test-lua` pass.
- The rule test fails if an agent-round path writes the reviewed buffer.
- A live review round with an ariadne#437 agent applies through `pair review apply`, and the pane only reloads and decorates.

## Plan

Quick flow: mostly deletion; the design is pair#661's plan, "M2 as built" (Retired by pair#664). Survey of every reference: Log 2026-10-18.

- [ ] nvim: delete `handoff.lua`, `reconcile.lua`, `gate.lua`; `init.lua` loses `set_base`, `apply_round`, `on_agent_round`, the handoff watch and admit wiring; `review.lua` loses `pending_records`, the winbar, `pane_state`, `on_defer`, `has_pending`, `set_focused`, `context`/`admit`/`did_land`; `restore_controller` loses `admit`, the pending-handoff and pending-landed checks, `did_land`; `apply.lua` loses `M.apply`, `place` and their helpers; `poke_bodies` loses `agent_applied` and the unused `review_opened`, and `with_context` names `pair review apply`.
- [ ] Fix found by the survey: the awaiting spinner is cleared only by the legacy `after_agent_round`; `on_applied_round` now calls an `after_applied_round` hook (clear awaiting, redraw the active diagnostic), with a test.
- [ ] Go: drop `ReviewHandoff`/`ReviewLanded` and their env bindings, gc rows and runtime manifest rows; add the retired names to `deletedvocabulary_test.go`; regenerate the runtime bundle.
- [ ] Tests: delete the legacy tests (review-reconcile, review-handoff, review-loop, review-producer-context with `fake-review-agent.sh`, review-branch-restore-diagnostics, Lua reconcile/gate/handoff_acceptance/handoff_context) and their Makefile.local rows; trim review-apply-test to render/snapshot; move review-window, review-projection, the restore tests, review-observation, review-controls, round_authority, restore_controller and poke_bodies tests off the handoff and onto the round artifact.
- [ ] Rule test `nvim/review/agent_round_rule_test.lua`: an applied round produces no buffer line change and no write of the reviewed file (on_lines, BufWritePre, writefile/io.open/os.rename spies), with a positive control that an operator send and a marker resolve trip the detectors.
- [ ] Docs: atlas/review-workbench.md, atlas/go-migration-inventory.md, README, workshop/targets/review-protocol.md seams 2/2b.
- [ ] `make test-review`, `make test-lua`, the recorded suite; a live round with an ariadne#437 agent (operator smoke).

## Log

### 2026-10-18
- Survey (read-only agent): `handoff.lua` is unused by `round_watch.lua`, so it deletes whole; `artifact.lua` stays (definition_seam). The env vars are set only by artifactpath `EnvironmentBindings`. Outside the issue's list, `round_authority_test`, `restore_controller_test`, `review-apply-test.sh`, `review-projection-test.sh` and `review-branch-restore-diagnostics-test.py` also use the legacy path. The runtime bundle mirrors nvim files (regenerate). Gap: `on_applied_round` never clears the awaiting spinner (only legacy `after_agent_round` did), so after an applied round the restore check still says agent request pending; fixed here.
