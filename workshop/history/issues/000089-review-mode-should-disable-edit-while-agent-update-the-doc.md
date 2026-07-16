---
id: 000089
status: done
deps: []
created: 2026-06-30
updated: 2026-07-05
started: 2026-07-05T12:07:32-07:00
estimate_hours: 11.66
actual_hours: 6.96
---

# review mode should disable edit while agent update the doc

> **Rescoped 2026-07-05** (see `## Revisions`). The title's "disable edit" was a
> hard lock; brainstorming replaced it with **concurrent-edit reconciliation** —
> the human keeps editing and the agent's round is merged onto their live edits,
> surfacing only genuine overlaps as reconcilable markers. The two "minor
> improvements" split out: smart-case search → **#101**; ask-to-squash-on-ship →
> **ariadne#164** (mostly ariadne `docflow`/`xx-fix`).

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
  found → the record is silently dropped (`'not found'`, WARN only). **This is
  what this issue addresses**: surface it as a reconcilable marker instead of a
  silent drop.
- **Occurrence-shift** — the human added/removed an earlier copy of the record's
  `old` text → "the Nth occurrence" now resolves to the *wrong* instance → a
  silent, *incorrect* edit. A distinct latent bug; **out of scope here** (this
  design keeps today's occurrence-against-live-buffer resolution). Noted so it
  isn't mistaken for fixed.

The lock exists only to preserve the invariant *what-the-agent-saw ==
what's-in-the-buffer*. We make that invariant explicit and reconcile against it
per-record instead of enforcing it by disabling the human.

## Estimate

Derived against `estimate-logic-v3.1` (ship wall-clock, AI-paired). Itemized by the
plan's milestones: M1 = one focused Lua/Neovim unit (multi-line markers); M2 splits
into the pure reconcile module + the glue/`init.lua` wiring; M3 = the apply-gate +
pane wiring (the heaviest — focus/mode/winbar/save + live smoke). Plus one
`milestone-review` per boundary and the atlas/protocol/skill docs.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
design-buffer: 0.30
item: lua-neovim         design=0.4 impl=1.2
item: lua-neovim         design=0.6 impl=1.8
item: lua-neovim         design=0.5 impl=2.0
item: lua-neovim         design=0.7 impl=2.6
item: milestone-review   design=0.0 impl=0.2
item: milestone-review   design=0.0 impl=0.2
item: milestone-review   design=0.0 impl=0.2
item: atlas-docs         design=0.0 impl=0.6
total: 11.66
```

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
- Case 4 → **save the human's buffer first** (secure their in-progress edits to
  disk — §8 durability), then stash the records as a **pending round** and show a
  prominent **`winbar`** at the top of the pane: `✨ agent results ready · ⌥⏎ to
  apply` (visible even while heads-down in insert; the awaiting-spinner statusline
  flips to this). Nothing applies until the human acts.

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

While a round is pending, the send-menu variant `Alt+Shift+Return`
(`<M-S-CR>`) also **applies the pending round** (there is nothing to "send" — a
mode/instruction selector is meaningless against results already produced); the
menu is offered only in the no-pending state.

### 3. The reconcile engine — what `on_agent_round` does when `v1 ≠ v0`

**Reconcile per-record, not by whole-document merge.** A line-granular
whole-doc merge (e.g. `git merge-file`) regresses prose: a markdown paragraph is
often a single long line, so two edits to *different words of the same paragraph*
would falsely conflict. Today's record model is span-granular (each `old` is
anchored by exact text, independent of lines), and we preserve that.

Lives in a new pure-ish module `nvim/review/reconcile.lua` with clear seams. The
key move: **a conflict becomes a synthetic replacement record**, so the *whole*
reconcile is a single `apply.apply` call — resolving clean edits and conflict
placements against one buffer snapshot, in one undo block, with no ordering
hazard.

1. **Classify** (pure) — `classify(records, v1) → {clean, conflicts}`: a record is
   `clean` iff `reconstruct.nth_offset(v1, r.old, r.occurrence or 1)` resolves
   (its `old` still exists in the live buffer — the *exact* anchor test + `or 1`
   fallback `apply.apply` uses, so classify faithfully predicts what apply will
   land). Records that don't resolve are `conflicts` (the human edited that exact
   span). Span-granular: non-overlapping same-line edits stay clean, **no
   regression**.
2. **Build each conflict as a synthetic record** — resolve the conflict record's
   `old`@occurrence against **v0** → its base line; `vim.diff(v0, v1,
   {result_type='indices'})` (nvim builtin — no external process, no temp files) →
   the changed hunk covering that base line → the hunk's **v1** line-range is the
   human's current text for that region. Coalesce conflicts by hunk; for each
   conflicted hunk emit a synthetic record
   `{ old = «exact v1 hunk text», occurrence = «its nth in v1», new =
   reconcile.conflict_marker(hunk_text, intents), explain = "reconcile" }` where
   the marker (pure, unit-tested builder) is:

   ```
   🤖<«human's current hunk text»>[reconcile — agent wanted:
     • «old» → «new» (why: «explain»)
     • …]
   ```

   **Both** the `<…>` human text **and** the `[…]` intents are escaped through the
   marker codec (`esc_quote`) so unbalanced brackets in quoted code (`arr[0`, a
   stray `]`) can never break the marker's parse — closing the "a marker never
   fails to parse" invariant. `\[` etc. remain readable in the raw buffer;
   `render_markers`/resolve `unescape` them for display and resolution.
3. **Apply clean + conflicts in one `apply.apply(buf, clean ++ synthetic)`** —
   **unchanged**. `apply.apply` resolves every record's span against a *single*
   live-buffer (`v1`) snapshot up front (`apply.lua:261`), sorts, applies
   bottom-to-top in **one** undo block, and decorates. This is what dissolves the
   ordering/coordinate hazard: conflicts are just replacements resolved against the
   same `v1` as the clean edits — no "place first vs apply first" tension. Clean
   records get `DiffChange` highlights + `explain` diagnoses; a synthetic record's
   `new` is a marker proposal, so `apply`'s `is_marker_proposal` path sets
   `no_highlight` and it self-highlights via `render_markers` (M1) instead, with
   the short `"reconcile"` gutter diagnosis. **Clean edit sharing a human-changed
   line with a conflict → FOLDED (M3, M2-review 3.1):** conflict placement is
   line-granular (the synthetic `old` is the whole changed hunk line) while clean
   edits are span-granular, so a clean edit sharing a *human-changed line* with a
   conflict would fall inside the synthetic `old`. Rather than let `apply`'s overlap
   guard drop it, `plan_conflicts` **folds** such a clean record into that hunk's
   conflict marker (its `old→new` is surfaced in the marker's intent list) and
   `reconcile_round` excludes it from the apply set. So the contested line becomes
   one marker citing *all* the agent's intents for it — nothing dropped. Clean edits
   on other lines apply span-granularly as before. Projection snapshot dance runs as
   in the fast path (§8), so undo/redo stays coherent.
4. **Save + poke** — save; write the landed-artifact (embedding only the *clean*
   enriched records as the agent's actual edits; accounting in §8) and poke
   `agent_applied` so the agent commits the round (Option A).

When `v1 == v0` (fast path), `on_agent_round` is today's `apply.apply(buf,
records)` unchanged — full per-record highlights + diagnoses.

### 4. Conflicts = prefilled instructions (no special mode)

A conflict is just a `🤖<…>[reconcile — …]` marker in the doc. There is no
conflict state machine: the human waits, keeps editing, resolves it by hand
(Alt+r rejects the marker back to their own text; Alt+a is available too — §8),
or resubmits immediately — and on the next Alt+Return it rides to the agent as an
ordinary `🤖[…]` request (which `xx-fix` already fulfills). The reconcile loop is
the existing round-trip.

### 5. Commit attribution — Option A

The reconciled doc (which includes the human's concurrent v0→v1 edits) is
committed as the **agent** round. Simple; everything is preserved in git; the
human/agent alternation in docflow history is slightly muddied (the human's
post-send edits ride into the agent round commit). Accepted — the cleaner Option
B (a separate "commit, don't review" human round) is out of scope.

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
- **audit the other marker-position consumers** for multi-line: `jump_marker`
  and `resolve_paragraph_to_cursor` also key off `m.line` (the marker *start*
  line, `review.lua`) — confirm/repair their behavior for a marker spanning a
  paragraph boundary;
- raise `MULTILINE_LINE_BUDGET` to **200** and have the reconciler cap any single
  conflict hunk it wraps at that budget (larger hunks split or fall back to a
  short "region changed" marker) so a marker never fails to parse.

### 7. Protocol touches

- **pair** `workshop/targets/review-protocol.md` — document the reconcile state:
  the `v0` snapshot, per-record reconcile-on-concurrent-edit, and conflict-marker
  semantics.
- **ariadne** `xx-fix` SKILL — a note that `🤖<…>[reconcile — …]` markers carry a
  human region plus the agent's blocked intents, and how the agent reconciles
  them (small, cross-repo).

### 8. Edge cases & lifecycle

- **`v0` threading + `on_agent_round` signature.** `v0` is snapshotted at each
  send (in `mark_awaiting`) and stored on the review session
  (`sessions[buf].base` in `init.lua`) via a new `review.set_base(buf, content)`;
  `on_agent_round` reads it (its signature gains `v0` from the session, not a new
  positional arg — the handoff watcher passes only records). Both `decide_apply`
  (case 1) and the fast/reconcile branch read this `v0`. **First round / nil base:**
  the normal flow always sends first (so `base` is set), but if `base` is nil
  (never sent, or a stray handoff) `on_agent_round` treats it as the fast path —
  `apply.apply(buf, records)` on the live buffer — and never reaches `vim.diff`.
- **Gate ↔ spinner ↔ winbar wiring.** On DEFER, `on_agent_round` does not run, so
  its `after_agent_round`→`clear_awaiting` never fires. Defer therefore
  **explicitly** calls `clear_awaiting()` (the spinner meant "waiting for the
  agent"; the agent has now replied) and raises the winbar. The winbar is the
  single "pending" signal. Consuming the pending round (Alt+Return) clears the
  winbar and runs `on_agent_round`; `VimLeave` teardown also clears it.
- **At most one pending round; stray second handoff.** By protocol the agent
  awaits the `agent_applied` commit poke before its next round, and that poke only
  fires *after* apply — so while a round is pending (unapplied) no second handoff
  is expected. Defensively, a second handoff **replaces** the pending records
  (same `v0` — no new send happened, so the base is unchanged). Single pending
  slot.
- **Human-edit durability (invariant: never lose the human's work).** The pending
  window is the one place the human's post-submit edits are unsaved (last write was
  at submit = `v0`). Two saves close it: (1) **save-on-defer** — case 4 writes the
  buffer *before* stashing the round (§1); (2) **save-on-`VimLeave`** if modified —
  the existing teardown autocmd (`review.lua`) also writes, covering edits typed
  after the defer while the winbar is up. The review pane is a live workbench under
  agent/git governance (nvim writes no git — this only touches the working file),
  so auto-persisting on exit is safe, not a surprise.
- **Abandon the pending *round* on quit/crash — acceptable, idempotent recovery.**
  `handoff.watch` unlinked the handoff on arrival, so the stashed records live only
  in memory; a true pane close (not the common Alt+c *hide*, which keeps the nvim
  alive) **drops** them. That's fine by design: the human's edits are already on
  disk (above), so on reopen the doc is their saved `v1`; a resubmit re-triggers
  the agent to review the *current* state and re-propose. The dropped round is
  recomputed, not lost work — "resubmit old change + new edits" converges. **No
  pending-round persistence** (confirmed with operator).
- **`FocusLost` reliability.** Case 2 ("not focused → apply") depends on
  `FocusLost` firing on a zellij pane switch, which terminal focus events may not
  guarantee. Failure mode is **benign**: `focused` stays true, so we fall through
  to the mode check and at worst DEFER (show the winbar) when we could have
  applied — never an incorrect apply. Assumption stated; no extra machinery.
- **`vim.diff` failure.** `vim.diff` is a pure builtin; if it errors (shouldn't),
  the reconciler falls back to today's `apply.apply(buf, records)` on the live
  buffer (best-effort; non-anchoring records drop with the existing WARN) rather
  than blocking the round.
- **Landed-artifact accounting (reconcile path).** `applied` = count of `clean`
  records that landed; `conflicts` = count of conflict *markers* placed;
  `dropped` = records `apply.apply` still rejects (empty-old / agent-internal
  record overlap — unchanged, rare). `body` = `record.embed_in_body` of the
  `clean` enriched records (the agent's actual applied edits); the conflict
  markers live in the committed doc, and the poke summary reads
  `"«N» edit(s)«, M conflict(s)»"`. The agent commits the working tree verbatim.
- **Undo/redo decoration coherence.** Because reconcile is a *single*
  `apply.apply` call (clean + synthetic-conflict records), the fast path's
  projection dance (`projection.record_empty_for`/`record`/`ensure_watch`) applies
  verbatim — one snapshot, one undo block — so undo/redo restores decorations with
  no reconcile-specific handling.
- **Agent-internal record overlap.** Two *agent* records targeting overlapping
  spans still drop as today (`apply.lua` overlap guard); this is the agent's own
  records colliding, distinct from human-vs-agent overlap (which is a conflict).
- **Clean edit inside a conflict hunk (M2-review 3.1) — FOLDED (M3, done).** When a
  clean record shares a *human-changed line* with a conflict, the conflict's
  line-granular synthetic `old` (the whole line) would contain the clean record's
  span. `plan_conflicts(conflicts, v0, v1, hunks, clean)` **folds** such a clean
  record into that hunk's conflict marker (its `old→new` joins the marker's intent
  list) and returns it in a second `folded` value; `reconcile_round` excludes the
  folded records from the apply set. So the contested line becomes one marker citing
  all the agent's intents — nothing dropped, more correct than either the drop or a
  half-applied line. Chosen option (a) over accepting the counted drop. Tested:
  `reconcile_test` fold cases + `review-reconcile-test` case (f).
- **Alt+a/Alt+r on a reconcile marker.** A `🤖<human>[reconcile — …]` is a valid
  `<quoted>[user]` chain, so `resolve.resolve` acts on it: **reject** drops the
  marker leaving the human's own text; accept is available but reconciliation is
  normally the agent's job on resubmit. Confirmed intended.

## Done when

- **M1**: a multi-line `🤖<…>` highlights across all its lines; `resolve_at_cursor`
  accepts/rejects it with the cursor on any of its lines; a conflict-sized block
  parses. Asserted headless in `make test-lua` / `make test-review`.
- **M2**: with a concurrent edit seeded (`v1 ≠ v0`), driving an agent round
  reconciles per-record — records whose `old` still anchors apply cleanly
  (decorated, span-granular: a non-overlapping edit to the *same line* stays
  clean); records whose span the human changed become `🤖<…>[reconcile — …]`
  markers placed on the human's changed hunk (`vim.diff`); the fast path
  (`v1==v0`) is unchanged. Landed-artifact accounts `applied`/`conflicts`.
  Protocol docs updated. Headless-tested.
- **M3**: `decide_apply` returns apply/defer correctly for cases 1–4; a case-4
  handoff defers (no *agent* buffer change) **but saves the human's edits to disk**
  and shows the winbar; a modified buffer is also saved on `VimLeave`; Alt+Return
  with a pending round applies it (and without one, submits). Headless-tested
  (incl. asserting the file on disk holds the human's edits after a defer).

## Plan

- [x] M1 — multi-line `🤖<…>` support (highlight across lines, within-range
  `resolve_at_cursor`, section-budget for conflict-sized blocks) + tests
- [x] M2 — reconcile engine `nvim/review/reconcile.lua` (`v0` snapshot at send,
  fast/reconcile branch in `on_agent_round`, per-record classify, `vim.diff`
  conflict placement → `🤖<…>[reconcile — …]`, landed-artifact accounting,
  reconcile-path decorate/save/poke) + protocol docs + tests
- [x] M3 — apply-gate UX (pure `decide_apply`, defer-on-case-4 + winbar, focus/
  mode tracking, Alt+Return dual dispatch, save-on-defer + save-on-`VimLeave`
  durability, + Task 3.0 fold clean-edit-inside-conflict) + tests

(Detailed implementation plan → `workshop/plans/000089-*-plan.md` via the
writing-plans skill.)

## Revisions

### 2026-07-05 — rescope from lock to reconciliation
- **Why:** a hard modifiable=false lock is a workflow degradation; the human
  would rather keep editing. The record model already reconciles non-overlapping
  concurrent edits, so the lock only guarded overlaps + occurrence-shift.
- **Delta:** part 1 "disable edit" → per-record concurrent-edit reconciliation
  with conflict-as-marker (this Spec). Part 1's two "minor improvements" removed
  from scope: smart-case search → **#101**; ask-to-squash-on-ship → **ariadne#164**.

### 2026-07-05 — spec-review pass (mechanism change)
- **Why:** the first draft used a whole-doc `git merge-file`, which is
  line-granular and regresses prose (a paragraph is one long line → same-paragraph
  edits falsely conflict); flagged by the spec-document-reviewer.
- **Delta:** §3 reconciles **per-record** — anchor each record against the live
  buffer (span-granular, today's behavior; no regression), and only records whose
  span the human changed become conflicts, placed via `vim.diff` (nvim builtin,
  no temp files/external process). Dropped the `git merge-file`/scratch-`v0.1`
  path. Retracted the "fixes occurrence-shift" claim (kept as a noted out-of-scope
  edge). Added §8 (edge cases & lifecycle) covering gate↔spinner↔winbar wiring,
  single pending slot, `VimLeave`-abandon, `FocusLost` fallback, `v0` threading,
  landed-artifact accounting, projection coherence, and reconcile-marker resolve.

## Log

### 2026-06-30

### 2026-07-05
- 2026-07-05: closed — full make test green (exit 0): test-lua reconcile/gate + test-review reconcile/loop/window (M3 gate/defer/pending/winbar/VimLeave durability). 3 milestone boundary reviews (M1 SHIP; M2 & M3 FIX-THEN-SHIP, all findings fixed). LIVE PANE SMOKE DEFERRED to operator per "land this, I test later" — zellij focus/winbar + real agent round-trip + defer→quit→reopen durability not yet keyboard-verified.; review verdict: SHIP
- 2026-07-05: closed M3 — full make test green (exit 0); M3: gate.decide_apply (5-case pure gate), init on_agent_round defers on mid-edit, review.lua pane_state/on_defer/winbar/pending-consume/save-on-defer+VimLeave, Task 3.0 fold clean-edit-inside-conflict; gate_test + reconcile fold tests + review-window M3 asserts; review verdict: FIX-THEN-SHIP
- 2026-07-05: closed M2 — M2 boundary review FIX-THEN-SHIP fix applied: plan_conflicts never silently drops a blank-anchor conflict (nearest-non-empty fallback + all-blank-doc counted); blank-hunk/blank-line/huge-hunk/no-hunk tests added; full make test green; 3.2 note already on ariadne main; review verdict: FIX-THEN-SHIP
- 2026-07-05: closed M2 — full make test green (exit 0); reconcile.lua pure classify/conflict_marker/plan_conflicts + reconcile_round glue; review-reconcile-test (clean-only/conflict/mixed-one-undo, real vim.diff) + loop-test concurrent-edit case; init apply_round fast/reconcile branch + landed conflicts accounting; protocol docs (pair + ariadne); review verdict: FIX-THEN-SHIP
- 2026-07-05: closed M1 — make test-lua + make test-review green (exit 0); new asserts: spans_multiline cross-row span, resolve_at_cursor within-range on a multi-line marker, multi-line paragraph resolve, budget-200 parse (markers_test + review-window-test); review verdict: SHIP
- Rescoped after brainstorm (superpowers-brainstorming). Confirmed with operator:
  reconciliation over lock; commit attribution Option A; milestones M1→M2→M3;
  smart-case split to #101.
- Grounding reads: `apply.lua` (records anchor against the live buffer, l.261),
  `markers.lua` (parse crosses lines w/ 50-line budget; highlighter is per-line),
  `marker_codec.lua` (`esc_quote` covers conflict delimiters), ariadne
  `docflow.sh` (`ship` is `--no-ff`, no squash).
- Spec-review (spec-document-reviewer, fresh context) → Issues Found. Key finding:
  whole-doc `git merge-file` is line-granular → prose regression. Reworked §3 to
  per-record reconcile + `vim.diff` placement; added §8 for the enumerated
  edge/lifecycle gaps (spinner↔winbar, pending-slot, abandon, focus, v0 threading,
  landed accounting, projection, marker resolve).
- Spec-review pass 2 → both prior blockers confirmed resolved + all code-grounding
  verified; 2 new §3 wrinkles. Fixed: (a) escape **both** marker sections (`<…>`
  and `[…]`) so unbalanced brackets in quoted code can't break the parse; (b)
  dissolved the clean-vs-conflict ordering/undo tension by modeling each conflict
  as a **synthetic replacement record** — the whole reconcile is one `apply.apply`
  call (one snapshot, one undo block, `apply.apply` unchanged). Code-verified the
  synthetic-record claim directly (`reconstruct.is_marker_proposal` → `🤖[<{~]` so
  the marker gets `no_highlight`; `apply.exact_replacement_marker` needs an agent
  `{}` section our `[user]` marker lacks → inserts verbatim) in lieu of a third
  full pass; residual detail carried into the plan (with its own review gate).
- Operator raised: dropping a pending round could risk the human's *unsaved*
  post-submit edits. Correct — the defer window is the one place edits are unsaved.
  Decision: **save-on-defer + save-on-`VimLeave`** make human edits durable (a
  hard invariant); the *agent* round stays droppable on quit/crash because a
  resubmit re-derives it (idempotent recovery — operator OK with this). No
  pending-round persistence. §1 / §8 / M3 updated.
- change-code gates passed: plan-quality CLEAN (high), estimate-quality INFO
  (non-blocking). Branch `000089-…` in place; `estimate_hours: 11.66`.
- **M1 done** (multi-line markers). 1.1 `spans_multiline` supersedes the per-line
  `highlight_spans` (multi-line extmarks; ARCH-DRY, one highlight path). 1.2
  `resolve_at_cursor` matches a marker from any line it spans. 1.3 audited
  `jump_marker`/`resolve_paragraph_to_cursor` — already multi-line-correct
  (characterization test, no code change). 1.4 `MULTILINE_LINE_BUDGET` 50→200.
  `make test-lua` + `make test-review` green. Review verdict: SHIP.
- **M2 done** (reconcile engine). `reconcile.lua` pure `classify`/`conflict_marker`/
  `plan_conflicts` + `reconcile_round` glue (one `apply.apply`); `init.lua`
  `apply_round` fast/reconcile branch off the `v0` base; landed-artifact partitions
  clean (body) vs conflict (count); protocol docs (pair target + ariadne `xx-fix`
  on `main`). Full `make test` green.
- **M2 review: FIX-THEN-SHIP** (first dispatch hit a 401 auth error → verdict
  "unknown"; the close correctly did NOT finalize; re-ran after `/login`). Fixed 3.1
  (the real one): `plan_conflicts` silently dropped a conflict when the anchor line
  was blank (human blanked the exact line, or the fallback hit a blank line) —
  defeating the issue's core purpose. Now every conflict yields a marker via a
  nearest-non-empty anchor (backward-first); only an all-blank `v1` degenerates to an
  empty-`old` record `apply.apply` counts as dropped+WARNs (never silent). Added
  blank-hunk / blank-line-1 / huge-hunk / no-hunk tests (closed coverage gaps 5b/5c);
  atlas guarantee corrected. 3.2 (ariadne `xx-fix` note) was already delivered on
  ariadne `main`; the reviewer read it "missing" only because the shared ariadne
  checkout was transiently on a peer branch (`000145`). Minor findings
  (occurrence-counter family DRY, `#synth` vs authoritative count) noted,
  non-blocking.
- **M2 re-review: FIX-THEN-SHIP again** — confirmed the 3.1 silent-drop fix + the
  ariadne note (both good), but found a *new* real invariant violation: a **clean**
  record sharing a human-changed line with a **conflict** is dropped by `apply`'s
  overlap guard (the conflict's line-granular synthetic `old` swallows the clean
  span) — counted+WARNed, never silent, but a clean edit vanishes for a round. The
  spec's "disjoint by construction" claim was false within a line (common prose
  case). Dormant until M3. Corrected the invariant in Spec §3/§8 + plan Revisions;
  added **Task 3.0** to decide the handling (recommended: fold the clean edit into
  the conflict marker's intents) before the M3 live smoke. No M2 code change.
- **M3 done** (apply-gate + durability). Task 3.0 fold (clean-inside-conflict →
  marker's intents; the M2-review 3.1 finding, genuinely fixed). `gate.decide_apply`
  (pure 5-case) → `init.on_agent_round` defers on mid-edit → `review.lua` pane
  wiring (`pane_state`, `on_defer` save+winbar, single pending slot, Alt+Return dual
  dispatch, `v0` snapshot after save, save-on-`VimLeave`). Full `make test` green.
- **M3 review: FIX-THEN-SHIP** — gate/defer/fold/durability all confirmed correct;
  one Important (save-on-`VimLeave` was code-only) + 4 minor. All fixed: added the
  VimLeave-save headless test; clear `pending_records` on a direct apply (stale-Alt+
  Return edge); DRY'd `buf_content` (shared `review.buf_content = apply.buf_content`);
  corrected two comments. `make test` green. **Remaining: the live pane smoke**
  (real zellij focus + winbar + agent round-trip, incl. defer→quit→reopen durability)
  — can't run headless; must be recorded from an actual keyboard run at issue close.
- Verdict-trailer anchor: the M2/M3 `Review-Verdict:` trailers first landed on
  `--allow-empty` commits, which `sdlc close` can't see (it greps the trailer on
  commits that touch the issue file). Re-anchored onto issue-file-touching commits
  (M2 + M3).
- M3 verdict trailer re-anchored on this commit (see 68a4e497 for the M3 close note).

