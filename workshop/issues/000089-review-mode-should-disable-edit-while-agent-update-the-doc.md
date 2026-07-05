---
id: 000089
status: working
deps: []
created: 2026-06-30
updated: 2026-07-05
started: 2026-07-05T12:07:32-07:00
---

# review mode should disable edit while agent update the doc

> **Rescoped 2026-07-05** (see `## Revisions`). The title's "disable edit" was a
> hard lock; brainstorming replaced it with **concurrent-edit reconciliation** —
> the human keeps editing and the agent's round is merged onto their live edits,
> surfacing only genuine overlaps as reconcilable markers. The two "minor
> improvements" split out: smart-case search → **#101**; ask-to-squash-on-ship →
> its own issue (mostly ariadne `docflow`/`xx-fix`).

## Problem

While the agent is producing a review round, the human may keep editing the doc.
A hard lock (make the buffer non-modifiable during the agent's turn) prevents the
agent's edits from landing on a doc that no longer matches what it reviewed — but
it's a workflow degradation: the human would rather keep working.

The record model already reconciles *non-overlapping* concurrent edits for free:
`apply.lua` resolves each record's `old`@occurrence against the **live** buffer
(`apply.lua:261`, `local base = buf_content(buf)`), so an agent edit to a region
the human didn't touch still anchors and applies. Two failure modes remain:

- **Overlap** — the human edited the exact span the agent targeted → `old` isn't
  found → the record is silently dropped (`'not found'`, WARN only).
- **Occurrence-shift** — the human added/removed an earlier copy of the record's
  `old` text → "the Nth occurrence" now resolves to the *wrong* instance → a
  silent, *incorrect* edit. A latent correctness bug that concurrent editing
  makes reachable.

The lock exists only to preserve the invariant *what-the-agent-saw ==
what's-in-the-buffer*. We make that invariant explicit and reconcile against it
instead of enforcing it by disabling the human.

## Spec

### 1. The apply-gate — *when* to run the agent round

Reframe the problem as **deferral, not locking**: postpone `on_agent_round` until
the human is at a safe point. Nothing about editing is ever disabled. When a
handoff lands, a **pure decision** picks the case from `(v0, v1, focused, mode)`,
where `v0` = the content the agent reviewed (snapshotted at send) and `v1` = the
current buffer:

```
decide_apply(v0, v1, focused, mode):
  v1 == v0            → apply now   (case 1: nothing changed)
  not focused         → apply now   (case 2: human is in another pane)
  mode == normal      → apply now   (case 3: on the pane, not editing)
  else                → DEFER       (case 4: mid-edit — insert/visual/replace)
```

- Cases 1–3 → `on_agent_round` runs immediately (fast path when `v1==v0`, else
  the reconcile engine below).
- Case 4 → stash the records as a **pending round** and show a prominent
  **`winbar`** at the top of the pane: `✨ agent results ready · ⌥⏎ to apply`
  (visible even while heads-down in insert; the awaiting-spinner statusline flips
  to this). Nothing applies until the human acts.

Focus is tracked via `FocusGained`/`FocusLost` (the pane already wires
`FocusGained`); mode via `vim.fn.mode()` (`n` = apply; `i`/`R`/`v`/`V`/`^V`/`s` =
defer). `decide_apply` is a pure function → unit-tested in isolation.

### 2. Alt+Return — disambiguated by pending-state

Same keybind as submit; unambiguous because it branches on whether a round is
pending (the human's insight):

- **Pending round exists** → apply it: run the deferred `on_agent_round(buf,
  pending)` (saves the human's edits, reconciles, shows the result, pokes the
  agent to *commit* the round). Consumes the pending round; **no** new
  "please review" poke.
- **No pending round** → today's `finish_human_turn` (save + poke the agent to
  review).

### 3. The reconcile engine — what `on_agent_round` does when `v1 ≠ v0`

1. `v0.1 = apply(records, v0)` in a scratch buffer — faithful, since the agent's
   occurrences were computed against `v0` (this also fixes the occurrence-shift
   misapply).
2. `merged = git merge-file -p <v1> <v0> <v0.1>` (ours = human, base = v0,
   theirs = agent). git is already required — the doc must be in a repo. Three
   temp files under `PAIR_DATA_DIR`.
3. Rewrite each `<<<<<<< … ======= … >>>>>>>` conflict block into
   `🤖<…both sides…>[please reconcile]` (`esc_quote`'d; `esc_quote` already
   escapes `<`/`>`/`[`/`]`/`{`/`}`/`\`, so git-conflict content is marker-safe).
4. Load `merged` into the buffer as one undo block; save. Decorate the
   cleanly-merged records via the existing resume path,
   `reconstruct.decorate(clean_records, merged, 'new')` (locates records by their
   `new` text). Conflicts self-highlight as `ParleyReview*` markers.
5. Write the landed-artifact + poke `agent_applied` so the agent commits the
   round (Option A, below).

When `v1 == v0` (fast path), `on_agent_round` is today's `apply.apply` unchanged
— full per-record highlights + diagnoses.

### 4. Conflicts = prefilled instructions (no special mode)

A conflict is just a `🤖<…>[please reconcile]` marker in the doc. There is no
conflict state machine: the human waits, keeps editing, resolves it by hand, or
resubmits immediately — and on the next Alt+Return it rides to the agent as an
ordinary `🤖[…]` request (which `xx-fix` already fulfills). The reconcile loop is
the existing round-trip.

### 5. Commit attribution — Option A

The merged doc (which includes the human's concurrent v0→v1 edits) is committed
as the **agent** round. Simple; everything is preserved in git; the human/agent
alternation in docflow history is slightly muddied (the human's post-send edits
ride into the agent round commit). Accepted — the cleaner Option B (a separate
"commit, don't review" human round) is out of scope.

### 6. Multi-line `🤖<…>` support (prerequisite)

Conflicts are inherently multi-line. Today: the semantic parser
(`markers.parse_markers`) crosses lines but is bounded to 50 lines/section
(`MULTILINE_LINE_BUDGET`); the highlighter (`markers.highlight_spans`) is
**per-line only**; `resolve_at_cursor` matches on the marker's **first line
only**; `marker_end_pos`/apply are already multi-line. Needed:

- a multi-line-aware highlight path (drive extmarks from `parse_markers`'
  doc-offset spans, converting offsets → row/col, instead of the per-line
  `highlight_spans`);
- `resolve_at_cursor` matching anywhere in `[start_line, end_line]`;
- a budget bump / large-conflict cap so a big conflict block still parses.

### 7. Protocol touches

- **pair** `workshop/targets/review-protocol.md` — document the reconcile state:
  the `v0` snapshot, merge-on-concurrent-edit, and conflict-marker semantics.
- **ariadne** `xx-fix` SKILL — a note that `🤖<…>[please reconcile]` can wrap
  git-style conflict text and how the agent reconciles it (small, cross-repo).

## Done when

- **M1**: a multi-line `🤖<…>` highlights across all its lines; `resolve_at_cursor`
  accepts/rejects it with the cursor on any of its lines; a conflict-sized block
  parses. Asserted headless in `make test-lua` / `make test-review`.
- **M2**: with a concurrent edit seeded (`v1 ≠ v0`), driving an agent round
  produces a git-merged buffer; non-overlapping edits merge cleanly (decorated),
  overlaps become `🤖<…>[please reconcile]` markers; the fast path (`v1==v0`) is
  unchanged. Protocol docs updated. Headless-tested.
- **M3**: `decide_apply` returns apply/defer correctly for cases 1–4; a case-4
  handoff defers (no buffer change) and shows the winbar; Alt+Return with a
  pending round applies it (and without one, submits). Headless-tested.

## Plan

- [ ] M1 — multi-line `🤖<…>` support (highlight across lines, within-range
  `resolve_at_cursor`, section-budget for conflict-sized blocks) + tests
- [ ] M2 — reconcile engine (`v0` snapshot at send, fast/reconcile branch in
  `on_agent_round`, `git merge-file`, conflict→`🤖<…>[please reconcile]`,
  reconcile-path decorate/save/poke) + protocol docs + tests
- [ ] M3 — apply-gate UX (pure `decide_apply`, defer-on-case-4 + winbar, focus/
  mode tracking, Alt+Return dual dispatch) + tests

(Detailed implementation plan → `workshop/plans/000089-*-plan.md` via the
writing-plans skill.)

## Revisions

### 2026-07-05 — rescope from lock to reconciliation
- **Why:** a hard modifiable=false lock is a workflow degradation; the human
  would rather keep editing. The record model already reconciles non-overlapping
  concurrent edits, so the lock only guarded overlaps + occurrence-shift.
- **Delta:** part 1 "disable edit" → base-aware 3-way reconciliation with
  conflict-as-marker (this Spec). Part 1's two "minor improvements" removed from
  scope: smart-case search → **#101**; ask-to-squash-on-ship → its own issue.

## Log

### 2026-06-30

### 2026-07-05
- Rescoped after brainstorm (superpowers-brainstorming). Confirmed with operator:
  reconciliation over lock; commit attribution Option A; milestones M1→M2→M3;
  smart-case split to #101.
- Grounding reads: `apply.lua` (records anchor against the live buffer, l.261),
  `markers.lua` (parse crosses lines w/ 50-line budget; highlighter is per-line),
  `marker_codec.lua` (`esc_quote` covers conflict delimiters), ariadne
  `docflow.sh` (`ship` is `--no-ff`, no squash).

