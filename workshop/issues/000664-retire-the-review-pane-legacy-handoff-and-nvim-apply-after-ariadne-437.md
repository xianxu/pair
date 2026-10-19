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

- [x] nvim: delete `handoff.lua`, `reconcile.lua`, `gate.lua`; `init.lua` loses `set_base`, `apply_round`, `on_agent_round`, the handoff watch and admit wiring; `review.lua` loses `pending_records`, the winbar, `pane_state`, `on_defer`, `has_pending`, `set_focused`, `context`/`admit`/`did_land`; `restore_controller` loses `admit`, the pending-handoff and pending-landed checks, `did_land`; `apply.lua` loses `M.apply`, `place` and their helpers; `poke_bodies` loses `agent_applied` and the unused `review_opened`, and `with_context` names `pair review apply`.
- [x] Fix found by the survey: the awaiting spinner is cleared only by the legacy `after_agent_round`; `on_applied_round` now calls an `after_applied_round` hook (clear awaiting, redraw the active diagnostic), with a test.
- [x] Go: drop `ReviewHandoff`/`ReviewLanded` and their env bindings, gc rows and runtime manifest rows; add the retired names to `deletedvocabulary_test.go`; regenerate the runtime bundle.
- [x] Tests: delete the legacy tests (review-reconcile, review-handoff, review-loop, review-producer-context with `fake-review-agent.sh`, review-branch-restore-diagnostics, Lua reconcile/gate/handoff_acceptance/handoff_context) and their Makefile.local rows; trim review-apply-test to render/snapshot; move review-window, review-projection, the restore tests, review-observation, review-controls, round_authority, restore_controller and poke_bodies tests off the handoff and onto the round artifact.
- [x] Rule test `nvim/review/agent_round_rule_test.lua`: an applied round produces no buffer line change and no write of the reviewed file (on_lines, BufWritePre, writefile/io.open/os.rename spies), with positive controls proving each detector live after the reload (an operator edit, the operator send, a Lua write of the doc).
- [x] Docs: atlas/review-workbench.md, atlas/go-migration-inventory.md, README, workshop/targets/review-protocol.md seams 2/2b.
- [ ] `make test-review`, `make test-lua`, the recorded suite; a live round with an ariadne#437 agent (operator smoke).

## Log

### 2026-10-18
- 2026-10-18: closed — Retired the legacy agent-round path (handoff and landed files plus env vars, buffer apply, reconcile, apply gate, deferred round and winbar, agent_applied poke, the restore controller's handoff and landed checks) and its tests; the pane only reloads and decorates rounds from pair review apply. Rule test agent_round_rule_test.lua: an applied round causes no on_lines change, no BufWritePre and no Lua write of the doc. Every detector has a live positive control after the reload (the listener re-attaches after :edit! detaches it). Mutations: the applied round writing lines fails 4 assertions; the declined round writing lines fails 2. Fix: on_applied_round now calls after_applied_round, which clears the awaiting spinner (window test red without it, green with it). The restore test drives a real pair review apply round, CRLF/BOM preserved. The deleted-vocabulary guard carries the retired names. make test-review green; make test-lua green on rerun (flake #685); Go packages green; recorded suite green on e9b99eb5. The only unchecked Plan item is the operator's live round after the next Couch restart (no-plan-check; BR-2 disputed with evidence). (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP
- Survey (read-only agent): `handoff.lua` is unused by `round_watch.lua`, so it deletes whole; `artifact.lua` stays (definition_seam). The env vars are set only by artifactpath `EnvironmentBindings`. Outside the issue's list, `round_authority_test`, `restore_controller_test`, `review-apply-test.sh`, `review-projection-test.sh` and `review-branch-restore-diagnostics-test.py` also use the legacy path. The runtime bundle mirrors nvim files (regenerate). Gap: `on_applied_round` never clears the awaiting spinner (only legacy `after_agent_round` did), so after an applied round the restore check still says agent request pending; fixed here.
- Built. Also removed as dead once the legacy path went: `init.lua`'s `M.poke` seam, `restore_controller`'s `legacy` field and `init`'s `strict` parameter (both read only by the retired `admit`). `review_opened` had no live caller and is deleted, not reworded. Decided to delete `review-producer-context-test.sh` with `fake-review-agent.sh` (producer context checks live in ariadne's xx-fix now) and `review-branch-restore-diagnostics-test.py` (both cases were built on handoff completion and the landed rename); the restore test instead drives a real `pair review apply` round and checks CRLF/BOM survive.
- RED/GREEN: the window test's applied-round spinner check fails without `after_applied_round` (statusline keeps the spinner) and passes with it. The rule test fails four ways when `on_applied_round` writes the buffer (on_lines=1, BufWritePre=1, the control count, the declined case) and passes with the reload.
- The deleted-vocabulary guard now carries the retired names (`PAIR_REVIEW_HANDOFF_PATH`, `PAIR_REVIEW_LANDED_PATH`, `ReviewHandoff`, `agent_applied`, `decide_apply`, `reconcile_round`, `pending_records`), so a doc describing them as live fails `go test ./cmd/internal/artifactpath`.
- `make test-review` green. `make test-lua`: `nvim-route-poke-admission-test.sh` timed out once under load (unrelated, passes alone); filed flake #685, one unchanged rerun.
- Close review round 1 (FIX-THEN-SHIP). BR-1 (real): `:edit!` detaches a buffer listener (measured: on_detach fires, no on_lines), so the rule test's line detector went deaf after the reload and the declined-round check proved nothing. Fixed the class: the detector counts a detach as the reload and re-attaches, and every detector now has a positive control after the reload. Mutations: the applied round writing lines fails 4 assertions; the declined round writing lines fails 2. Minors folded: spinner.lua comment, `restore.validate_context`'s dead `legacy` parameter, the window test uses `round_watch.hash`. BR-2 (live round) disputed: the live check needs this build running in the operator's Couch, which only happens after landing and a restart; the TL holds live checks for the next restart, as on #654.
- Known gap, not fixed here: a review the agent answers with no edits writes no round artifact, so the awaiting spinner (and Alt+c's "agent request pending") stays until the next send. The legacy path cleared it on an empty handoff. ariadne#437's protocol defines no empty-round reply; raised with the hub.

## Revisions
- 2026-10-18, close review round 1: the rule-test Plan item's controls changed from "an operator send and a marker resolve" to the three controls built (operator edit, send, Lua write of the doc). A marker resolve writes through the same buffer-write path as the send, so it adds no detector coverage.
