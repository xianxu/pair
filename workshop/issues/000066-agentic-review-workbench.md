---
id: 000066
status: open
deps: [ariadne#000121]
target: review-protocol
github_issue:
created: 2026-06-18
updated: 2026-06-21
estimate_hours: 30
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

- [x] M0 — Read ariadne docflow; decide history/undo mechanism + per-hunk-explain home;
  finalize the record + docflow-commit contract. (Design; gates the rest.) → durable plan at
  `workshop/plans/000066-agentic-review-workbench-plan.md`.
- [x] M1 — Contract + history foundation: record format, docflow-commit round boundary,
  undo-preserving buffer apply. (Fake-agent-driven vertical; all tests green.)
- [x] M2 — Extract parley review consumer-half into pair as inline lua (render / projection /
  diagnosis / markers / modes); drop the invoke path.
- [x] M3 — Review window + pair integration (`:PairReview` / alt+r pane; poke channel to the
  agent).
M4 (agent protocol) — **re-sliced structure-first** (2026-06-21): the whole loop standing thin before any tuning. Agent half = ariadne **#000121**.

- [x] M4a — agent-owns-git inversion + the basic round (records → apply → landed-artifact → fake-agent commits). *Implemented/headless-verified; not separately milestone-closed because M4a'/M4b commits already landed before reconciliation. Review evidence folds into the current M4 skeleton boundary.*
- [x] M4a' — review-start & resume flow: `:PairReview` **proposes** → agent preps (pure readiness probe) → Alt+r opens when ready; reconstruct-on-open (resume repaint). Spec in `workshop/targets/review-protocol.md`; pair side built and headless-tested.
- [ ] M4b — **skeleton** (structure): 🤖[] fulfill/punt + **accept/reject** (parley §5) conversation + default editing posture + **ship** — completes the thin full cycle (open → propose → prep → review → converse → iterate → ship → resume). *Accept/reject + marker nav started; ship/default posture remain.*
- [ ] M4c — **thicken** (tuning; sub-slices when reached): modes menu + 🪄/spinner/lean-history bar; voice (`voice:` frontmatter); fact-check pass (`doc-review` fold); pending-🤖{} quickfix; diagnostic-display polish; `xx-fix`→`writing-assistant` rename. *Spinner pure helper exists as pre-work, unwired.*

## Log

- 2026-06-21: **Reconciled M4 boundary state after continuation `pair-pair`.** The branch had
  advanced past the older `review-m3` checkpoint: M4a pair-side agent-owns-git is built,
  M4a' review-target/readiness/resume is built, M4b accept/reject started, and an unwired
  M4c spinner helper exists. Because the commits crossed the planned M4a/M4a' review lines
  before a milestone-close, we are **not** retroactively closing M4a as a separate boundary;
  the evidence is folded into the current M4 skeleton boundary. Next: bring the atlas/plan
  fully current, run `make test-lua` + `make test-review`, then continue/close the skeleton
  from a truthful checkpoint. Verification after reconciliation: `make test-lua` green
  (slug/scrollback/annotate/changelog/adapt/doctor + review record/reconstruct/markers/mode/
  poke_bodies/wrap/readiness/spinner/resolve) and `make test-review` green (docflow/apply/
  handoff/loop/projection/poke/window/toggle/indicator/resume).

- 2026-06-20: **M4a pair-side done (Tasks 1–4), forked + verified.** The git-ownership
  inversion: the nvim writes git NOWHERE — `on_agent_round` applies+saves → writes the
  landed-artifact (seam #2b, `$XDG_DATA_HOME/pair/review-landed-<tag>.json`, body via the one
  `record.embed_in_body` encoder) → pokes `agent_applied`; `human_round` save-only;
  `review.start` dropped `docflow.start` (agent owns the branch). `fake-agent-v2` owns all
  git (branch + both rounds, reading the artifact verbatim); `pair_poke` injectable for tests.
  Plan-quality gate caught the body-authorship data-flow gap (FAILURE→INFO) → resolved via
  the landed-artifact + a dropped-record test (invariant #3). 103 headless checks green
  (`make test-lua` + `make test-review`). Invariant #1 flipped to BUILT (pair side). Commits
  2a4d95d, fabfaf8, 92f107f. **Tasks 5 (live smoke) + 6 (close) gated on ariadne #000121 M4a**
  (the real SKILL: recognize review-mode, create the branch, read the landed-artifact +
  commit rounds — `fake-agent-v2` is the protocol reference).
- 2026-06-19: M3 milestone-review **FIX-THEN-SHIP** (no Critical) — addressed all 3
  Important before crossing: I1 atlas described a non-existent line-1 `_pair_review_indicator`
  (corrected to the shipped statusline segment); I2 the missing-docflow degradation — the
  exact VimEnter crash class that shipped — had no test (added: `DOCFLOW_BIN=/nope` →
  unavailable=true, start doesn't crash, one INFO not ERROR); I3 the `review-<tag>.open`
  seam path was computed 3× with divergent empty-tag fallback (centralized in
  `nvim/review/seam.lua`, ARCH-DRY). Plus target-honesty (invariant #1 caveated as M4-target,
  not yet held) + plan revision + dead .gitignore line. 112 headless checks green.
- 2026-06-19: closed M3 — M3 review window: :PairReview + Alt+r draft-lua toggle (full-screen, instant) + id-based agent poke + marker render + diagnosis gutter + review-mode statusline bar; live smoke passed (pane opens/toggles full-screen, Alt+Return pokes /xx-fix, docflow render-only quiet); 95 headless checks green (make test-lua + test-review). M3 increment ~5.75h over M2 cumulative 2.57h.; review verdict: FIX-THEN-SHIP
- 2026-06-19: **M3 Alt+r rework + smoke round 2.** Reworked the Alt+r toggle off the
  transient 20%×1 floating "toggle pane" onto draft-nvim lua (`PairReviewToggle()`,
  Alt+d-style routing) + full-screen review pane (`zellij run --width 100%`, dropped
  `tput`). User re-smoked: pane opens full-screen + fast + toggles cleanly (#1/#3/#4/#5
  fixed). Remaining: docflow ENOENT noise (caller fired ERROR notify in VimEnter →
  "Error detected…"; fixed: `result.unavailable` → one calm INFO, render-only) + the
  poke used a basename so the agent couldn't find the doc (fixed: absolute path).
- 2026-06-19: **Direction confirmed — the review nvim writes NO git; the agent owns
  all git** (branch / round commits / ship), driven by prose pokes. Documented the
  full agent↔review-nvim state machine as a target → `workshop/targets/review-protocol.md`
  (`target: review-protocol`), BUILT (M1–M3, test-backed) vs M4-DESIGN clearly split.
  This makes M1's nvim-shells-docflow scaffolding to unwind in M4 (its nvim-commit e2e
  relocates to an agent-side test). M3 stays render-only (the unwind lands in M4).


- 2026-06-19: closed M2 — M2 increment 0.7h (engine measures cumulative 2.57h over the whole-issue window; M1 was 1.87h). Green: make test-lua (markers/mode pure) + make test-review (projection undo/redo coherence + riding + round-2 idempotence; apply snapshot multi-line round-trip; markers ready/pending/strike invariants); review verdict: FIX-THEN-SHIP
- 2026-06-19: M2 milestone-review CONVERGED → **SHIP** (3 judge rounds: FIX-THEN-SHIP → markers edge-case
  tests [inline-code/multi-line/budget] + minors [vim.uv, buf_content dedup→apply.buf_content, pcall-guard
  on_agent_round] → mode.list name≠basename negative test + marker line/col → SHIP; no Critical/Important;
  ARCH-DRY/PURE PASS). Ported the consumer-half: markers/mode/projection + apply.snapshot/apply_snapshot.
  Deferred (documented): marker rendering + interactive styling accumulation → M3; mode.directives into the
  agent SKILL.md → M4; cross-repo grammar-fix divergence → revisit shared-module if both repos keep evolving.
- 2026-06-18: closed M1 — make test-lua + make test-review all green; e2e (review-loop-test) proves handoff→undo-able apply→docflow agent-round (records in body)→undo crosses commit→decorations reconstruct from commit; real-docflow smoke passes; review verdict: unknown
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
- Claimed (`sdlc claim --issue 66 --no-start`, cheap lock on main; estimate deferred). pair's
  sdlc predates `start-plan`, so flow is claim → design → `change-code` → implement.
- M0 in progress: dispatched a docflow digest (commit structure, fine-grained-history-vs-checkpoints,
  per-hunk-explain home) to lock the contract before planning M1.
- M0 docflow findings (`ariadne/scripts/docflow.sh`, ~300-line shell; atlas `atlas/workflow/docflow.md`;
  used by xx-fix; suspend/resume = ariadne #90, open):
  - **Round = two commits** (human then agent) on a `review/<slug>` branch; subject
    `review(<slug>): <side> r<N> — <summary>`; agent rationale in the **commit body** +
    `Co-Authored-By`; `ship` does `--no-ff` merge + branch delete (`--first-parent` = clean
    per-batch view, full log = forensic). Reusable as-is — shell out, don't reimplement.
  - **Deliberately leaves us 3 things:** (a) no undo assumptions / no `undofile`; (b) agent flow is
    xx-fix's file-write-in-place; (c) rationale is commit-level only, no per-hunk explain. These are
    exactly #66's open questions — docflow defers them to us.
  - Reuse `review-convention.md`'s 🤖 marker grammar for human in-doc review requests; don't fork.
- M0 decisions (proposed, pending operator confirm):
  1. **Buffer entry = in-buffer undo-able apply** (the one extension over docflow): apply the agent's
     `{old,occurrence,new,explain}` records as in-buffer ops, then commit via `docflow round`. Preserves undo.
  2. **Cross-session undo = nvim persistent `undofile`**, not history-doc replay → the third
     "review-history doc" layer collapses. Edits-as-in-buffer-ops make the undo tree real + persistable.
  3. **Per-hunk explain = line-anchored in the agent round's commit body** (`- [L12-15] reworded`),
     extending docflow's body-rationale convention. Frozen commit ⇒ no drift. No sidecar / no git-notes.
  - Net: durable record = git (commits + per-hunk explains in body); fine-grained undo = nvim undofile.
    Two mechanisms, no sidecar — three-layer model drops to two.
  - Operator confirmed all three decisions; accepted the one drawback (doc must live in a git repo).
- M0 closed: durable plan written (`workshop/plans/000066-agentic-review-workbench-plan.md`) via writing-plans
  skill. Pure core = `review.record` (one serialization shared by handoff file + agent commit body) +
  `review.reconstruct` (records→decorations, occurrence-anchored, live + resume). M1 = fake-agent-driven
  vertical proving the contract headlessly; M2–M4 outlined as their own plans.
- Plan review (fresh-context, general-purpose): **Approved** — reviewer empirically validated the 3 riskiest
  mechanisms (undojoin single-undo-block on nvim 0.11.7; `vim.json` under `nvim -l`; cross-session
  reconstruction from an externally-authored commit body). 3 advisories folded in: E790-safe undojoin
  (first edit fresh, join 2..N), handoff via timer-poll not fs_event (macOS FSEvents precedent in init.lua),
  e2e human round must mutate+stage the doc (docflow no-ops an empty round).
- `sdlc change-code --issue 66`: structural ✓, plan-quality judge **INFO** (executable as-written), branch
  `000066-agentic-review-workbench` created in-place, estimate 30h, status → working. The judge caught a real
  correctness bug the first reviewer missed — **occurrence mapping** (finding #1): `occurrence` (Nth `old` in
  base) ≠ position of `new` post-apply. Folded the fix into the plan before any code: Records carry both
  `occurrence` and `new_occurrence`; `apply` decorates from its own edited ranges + enriches `new_occurrence`;
  `reconstruct` locates `new` by `new_occurrence`; bottom-to-top apply (finding #2); hermetic `fake-docflow.sh`
  with real commits + gated real-docflow smoke (finding #3).
- Implementing M1 (TDD), starting Task 1 (`review.record`).
- M1 complete (6 commits, all TDD-green). Pure core: `record` (one serialization, handoff==commit
  body), `reconstruct` (records→decorations, occurrence-anchored). Seams: `docflow` (+ hermetic
  `fake-docflow.sh`, real-docflow smoke passing), `apply` (single undo-block, bottom-to-top, decorate
  +new_occurrence), `handoff` (timer-poll, atomic). Orchestrator `init.lua` + `fake-review-agent.sh`.
  E2E (`tests/review-loop-test.sh`) proves the contract headlessly: handoff → undo-able apply →
  docflow agent-round (records in body) → undo crosses the commit → decorations reconstruct from the
  commit; human round survives the agent-round undo. Wired `test-review` into `make test`.
- Verified: `make test-lua` (record/reconstruct) + `make test-review` (docflow/apply/handoff/loop) all green.
- **M1 milestone-review: converged → SHIP-equivalent.** The auto-dispatched fresh-context judge
  ran 4 rounds (binary recorded the first as `unknown` — its leading verdict token wasn't parsed).
  Round 1 (blocking) I1 decorate-from-actual-ranges + consistent occurrence counting, I2 buf-current
  undojoin, I3 surface docflow exits — fixed. Round 2 (blocking) surface dropped/unanchorable records
  + overlap guard (`apply` returns `(enriched, dropped)`) — fixed. Round 3: one non-blocking item
  (malformed-handoff notify) + minors — fixed. Round 4: **"no shipped-code defect, nothing blocks
  M2"**; closed two test-fidelity gaps (in-scope-only staging assertion; `apply.render` coverage).
  Every finding addressed with a regression test. Review-Window 20443c8..HEAD (11 commits).
  Carried to M2/M4 (documented in plan ## Revisions, non-blocking): additive styling vs. clear (M2),
  newline-offset perf index (M2), VimLeave timer cleanup (M3), stronger resume anchor + file-vs-buffer
  newline contract (M4).
- **ACTUAL correction:** M1 milestone-close was passed a hand-typed `--actual 4` (a guess).
  The MEASURED active-time-v3 value (in-binary engine, window 20443c8..HEAD) is **1.87h** —
  the typed 4 was ~2× over. Use 1.87 (≈2.0) for M1; the final `sdlc close --issue 66` will
  recompute over the full window. Root cause: **pair's `./bin/sdlc` is a stale build** (older
  ariadne, before active-time was folded into Go) — `sdlc actual` was "unknown command" and
  the close explainer pointed at a non-existent `active-time-v3.py`. pair's sdlc should be
  rebuilt from current ariadne (base-layer source of truth).
