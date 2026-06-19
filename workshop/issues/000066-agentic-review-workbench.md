---
id: 000066
status: open
deps: []
github_issue:
created: 2026-06-18
updated: 2026-06-18
estimate_hours:
---

# Agentic memory-backed review as a document workbench in pair

## Problem

parley's `review` is one-shot and amnesiac: it assembles a prompt, forces a single
`propose_edits` tool call, and stops. No agentic loop, no transcript, no cross-repo
memory discovery — the review tip floats over a corpus (brain, pensives, the repos)
it can't reach. Meanwhile pair already runs the structure that fixes this: a
persistent `claude --resume` session paired with nvim document surfaces. So the goal
is to make review a memory-backed agentic loop **hosted in pair**, with an embedded
nvim review pane as the document workbench — not to build a new harness.

Design rationale lives in `workshop/pensive/2026-06-18-01-pensive-agentic-review-workbench.md`.

## Spec

Converged design decisions:

- **Two workbenches.** pair's persistent agent = conversational/compute surface; an
  embedded review nvim = document surface. **(B)-first:** delegate the loop to pair's
  session (it already has tools + native reach into brain/pensives/datatype skills);
  owning a custom loop is a later, productization move.
- **Embed by extraction.** Port parley's review *consumer half* into pair as inline lua
  (render, journal, projection, diagnosis, marker editing, modes). Drop the LLM-invoke
  half (`run_via_invoke`, dispatcher, provider) — pair's agent is the producer now.
  Roughly half ports as-is; the invoke entanglement is isolated to `run_via_invoke`.
- **Window.** `:PairReview <file>` / alt+r opens a full-screen review nvim; alt+r leaves
  (never bare esc — nvim's most-pressed key). pair's draft nvim loads no external plugins,
  but pair already spawns role-specific nvims (scrollback, changelog); this is one more.
- **Record → four uses.** The agent proposes `{old, occurrence, new, explain}` records
  (`propose_edits` + an occurrence disambiguator, so multiple-old_string is input
  validation, never a re-anchor problem). The review pane applies each as an **undo-able**
  buffer op, drops a **riding** extmark, renders `explain` as the diagnosis, and makes the
  docflow commit.
- **git is checkpoints, not history.** Three non-collapsible layers: (1) nvim undo —
  continuous across commit boundaries, persistent across sessions (undo back to the first
  review); (2) git commits — docflow round checkpoints; (3) review-history doc — the
  per-hunk explains git can't hold. The undo requirement forces "agent proposes records,
  pane applies undo-ably" over "agent writes file, pane reloads" (a reload resets undo).
- **Styling accumulates.** Agent highlights persist (the human's standing cue of what's
  unreviewed). Typing / alt+a (accept) / alt+r (reject) all ride; a human round adds its
  own styling but never clears the agent's. Agent styling clears on the **next conversation
  turn** (the agent repaints), optionally on an explicit end-of-human-turn.
- **Modes.** Port parley's review modes (aggressiveness / edit+explain style); converge with
  fix's simpler version. `mode.lua` + briefs are in the port bucket.
- **Contract = review SKILL.md + record/commit format.** pair/Claude is one *consumer*, not
  a hard dependency (agnosticism guard). Possibly keep parley's in-process one-shot as a
  no-pair fallback producer of the same contract.

Open (resolve at/before start-plan):

- Cross-session undo — trust nvim persistent `undofile`, or reconstruct from the history
  doc? Decide after reading ariadne **docflow**.
- Where per-hunk explains live — commit message body (line-anchored) / git notes /
  history-doc sidecar. Align with docflow's convention; don't fork.
- Divergence from copying parley's review code — accepted for B-first; shared module only
  if both keep evolving.

## Done when

- `:PairReview <file>` opens a full-screen review nvim in a pair session; alt+r toggles
  between it and the agent pane.
- The pair agent's persistent session (with memory discovery) proposes edits as
  `{old, occurrence, new, explain}` records; the review pane applies them as undo-able
  buffer ops, styles them, and commits the round.
- nvim undo is continuous across commit boundaries and persists across sessions.
- Agent styling accumulates and rides human typing/accept/reject; clears on the next
  conversation turn.
- At least one mode drives the agent; modes are selectable.
- A process-level end-to-end round-trip works: agent proposes → pane applies+styles+commits
  → human edits/accepts → next turn repaints. (Not just unit tests — a faithful demo.)

## Plan

Milestones are review boundaries; sub-steps firm up after M0.

- [ ] M0 — Read ariadne docflow; decide history/undo mechanism + per-hunk-explain home;
  finalize the record + docflow-commit contract. (Design; gates the rest.)
- [ ] M1 — Contract + history foundation: record format, docflow-commit round boundary,
  undo-preserving buffer apply.
- [ ] M2 — Extract parley review consumer-half into pair as inline lua (render / projection /
  diagnosis / markers / modes); drop the invoke path.
- [ ] M3 — Review window + pair integration (`:PairReview` / alt+r pane; poke channel to the
  agent).
- [ ] M4 — Agent protocol (review SKILL.md + modes + memory discovery); end-to-end round-trip.

## Log

### 2026-06-18

- Design brainstormed and captured in
  `workshop/pensive/2026-06-18-01-pensive-agentic-review-workbench.md`.
- Two subagent digests grounded it: (1) parley's review subsystem — journal + projection +
  diag_display + `propose_edits` already form a rich, git-tracked state model with the
  `{old,new,explain}` triple; the only real gap is the one-shot forced `tool_choice`.
  (2) pair — tag-addressed persistent `claude --resume` session; `zellij ... write-chars`
  poke channel; no existing parley↔pair integration (this is new but architecturally
  compatible).
- Key decisions: (B)-delegate-to-pair first; extract the consumer-half; `{old,occurrence,
  new,explain}` applied undo-ably + committed; git = checkpoints not history; styling
  accumulates and clears on the next conversation turn.
- Next: read ariadne docflow (M0) before planning the contract.
