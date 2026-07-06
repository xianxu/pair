# Boundary Review — pair#89 (whole-issue close)

| field | value |
|-------|-------|
| issue | 89 — review mode should disable edit while agent update the doc |
| repo | pair |
| issue file | workshop/issues/000089-review-mode-should-disable-edit-while-agent-update-the-doc.md |
| boundary | whole-issue close |
| milestone | — |
| window | c71f2597ba11bc269fd021547d1446744482baab..HEAD |
| command | sdlc close --issue 89 |
| reviewer | claude |
| timestamp | 2026-07-05T20:39:43-07:00 |
| verdict | SHIP |

## Review

I have everything I need. I ran all affected suites (green, exit 0), traced the reconcile engine / gate / durability paths against Spec §1/§3/§8 and the M1–M3 Done-when, verified the cross-repo agent consumer derives the wire format, and confirmed the M1 DRY consolidation retired `highlight_spans` in production. Here's the whole-issue close review.

```verdict
verdict: SHIP
confidence: high
```

**Summary.** pair#89 delivers its full rescoped purpose — the human keeps editing while the agent produces a round, and the round is reconciled onto the live buffer per-record rather than blocked by a lock. The three milestones compose cleanly: M1 multi-line `🤖<…>` markers (highlight + resolve across rows), M2 the pure reconcile engine (`classify`/`conflict_marker`/`plan_conflicts` + one-`apply.apply` glue), M3 the pure apply-gate + defer/winbar/durability. I ran `gate`/`reconcile`/`markers`/`poke_bodies` (pure) and `review-reconcile`/`review-loop`/`review-window` (glue/e2e) — all pass, zero FAIL. Each milestone's boundary review ran and its Critical/Important findings are genuinely closed (verified independently below, not taken on trust). Nothing blocks the close: no crashes on any traced input, no contract drift, no silent-drop path, docs (atlas + protocol + ariadne consumer) updated. Every surviving finding is Minor. The one real caveat is a *process* one, not a code one — the live pane smoke can't run headless, so the `sdlc close --verified` evidence must record its **actual** keyboard-run result, not assert it.

**1. Strengths (confirmed-good ground)**
- **ARCH-PURE is exemplary and I verified it end-to-end.** `gate.decide_apply` (`nvim/review/gate.lua:13`), `reconcile.classify`/`conflict_marker`/`plan_conflicts`, and `markers.spans_multiline` are all string/table-in, no vim API; their tests run under `nvim -l` with no exec/net/fs. `plan_conflicts` takes `hunks` as **data** so the hard hunk-mapping logic is testable without `vim.diff` — the discipline the plan committed to, actually honored.
- **The "conflict = synthetic replacement record → one `apply.apply`" design holds.** It dissolves the clean-vs-conflict ordering/undo hazard by routing everything through one snapshot/undo block. `review-reconcile-test.sh` case (e) proves a single undo reverts the whole reconcile round; case (d) proves only the clean edit gets a `DiffChange` highlight (the synthetic marker self-highlights via `is_marker_proposal`, extmark count == 1).
- **The M2/M3 findings are genuinely fixed, not papered over.** The 3.1 silent-drop is closed by the `nearest_nonempty` fallback (`reconcile.lua:111`, `210`) with an all-blank-`v1` degenerate that emits `old=''` so `apply.apply` counts + WARNs it (`init.lua:70-73`); Task 3.0's fold (`reconcile.lua:161-184` + `reconcile_round:70-78`) folds a clean edit sharing a human-changed line into the conflict marker rather than dropping it — I confirmed live via `review-reconcile-test.sh` case (f) ("nothing dropped").
- **DRY consolidation at the parser level (M1) is complete.** `grep` confirms **no production caller** of the retired `highlight_spans` survives (only comments name it); `render_markers` (`review.lua:250`) derives from the single multi-line `spans_multiline`, and the v0 snapshot joins through the one shared `apply.buf_content` (`init.lua:28`, `review.lua:438`) so gate case-1 and the fast-path agree by construction.
- **Durability is real and both legs are now tested.** `on_defer` saves *before* stashing (`review.lua:465-470`); the window test asserts the file on disk holds the deferred edit **and** the post-defer VimLeave edit (`vimleave-saves`) — the M3 review's one Important (VimLeave untested) is closed.

**2. Critical findings** — none.

**3. Important findings** — none. (No crash on any traced input; the nil-base first-round edge defers-then-fast-path-applies benignly; the `FocusLost`-miss fallback only over-defers, never mis-applies.)

**4. Minor findings**
- **ARCH-DRY: the offset→(row,col) helper family is now duplicated across four files.** `markers.lua` carries *two* verbatim binary-search twins — `offset_to_pos` (`:155`, inside `parse_markers`) and `pos_of` (`:233`, inside `spans_multiline`) — plus `reconstruct.pos_at`/`line_of` (linear) and `reconcile.split_lines`/`v1_starts`. The M1 review recommended folding this into M2 "before a fourth copy appears"; it wasn't done. Consolidate to one shared `reconstruct` helper. Cite **ARCH-DRY**. Non-blocking, correct as-is.
- **README not updated for the review-pane winbar + Alt+Return dual-dispatch.** The `✨ agent results ready · ⌥⏎ to apply` winbar and the pending-state meaning of `Alt+Return` are new visible surface, but the whole review-workbench pane is still pre-live (gated on ariadne#000121) and only its `Alt+c` open is in README today — so this is consistent with the feature's doc state, not a regression, and the atlas *is* updated. Note for when the feature goes live.
- **`nearest_nonempty` comment vs behavior** (`reconcile.lua:107-110`) — says "backward-first … the preceding kept line," but `d=0` returns `start_idx` itself (the line *following* a deletion). Cosmetic; carried from the M2/M3 reviews, still present.
- **Fold-then-synthetic-dropped edge (untested):** a clean record folded into a conflict group is excluded from the apply set; if that group's synthetic marker is itself dropped by apply (overlap/degenerate), the folded intent is lost entirely (neither applied nor surfaced). Extreme edge (human rewrites the same line the agent hit *and* two synthetics collide); acceptable, worth a note.
- **Stale-pending is handled but implicitly:** a second handoff arriving in normal mode applies directly and relies on `after_agent_round` (`review.lua:477`) to clear `pending_records` rather than clearing at the top of the direct-apply path. Defensive-only (protocol-guarded), recoverable, never corruption.

**5. Test coverage notes**
- Strong and layered: pure (gate 5-case incl. `V`/`R`; classify clean/conflict + `occurrence or 1`; conflict_marker escaping/round-trip + multi-line; plan_conflicts coalesce/repeated-hunk/deletion/blank-hunk/blank-line-1/huge-hunk/fold), glue (reconcile_round fast/clean-only/conflict/mixed-one-undo/fold), e2e (loop reconcile + gate defer/apply), and pane wiring (window `pane-state`/`defer-*`/`pending-*`/`vimleave-saves`). All green when I ran them.
- **Genuine gap is non-headless only:** the live pane smoke — real zellij focus + winbar render + agent round-trip, incl. defer→quit→reopen durability across a real reopen — can't run headless and is marked "remaining manual proof" in the atlas. This is the one thing the headless suite structurally can't cover; the `sdlc close --verified` evidence must state its actual keyboard-run result.

**6. Architectural notes for upcoming work**
- ARCH-DRY **pass** (one apply path; classify reuses `nth_offset`; save-on-defer reuses `human_round`; gate single-sourced) — modulo the offset-helper family in Minor #1; pick one home (`reconstruct`) before the next module needs offset math.
- ARCH-PURE **pass** (as above).
- ARCH-PURPOSE **pass** — shadow-sweep of the `conflict_marker` wire-format consumers: pair renders it (`spans_multiline`→extmarks), pair resolves it (`resolve_at_cursor` multi-line), `review-protocol.md` derives it, and the ariadne `xx-fix` agent note is present on ariadne `main` (`SKILL.md:227-237`, verified this run) with the exact `🤖<…>[reconcile — agent wanted: • old → new (why …)]` format. The real live agent recognizing review-mode is legitimately a separate issue (ariadne#000121), not the deferred point of *this* one.

**7. Plan revision recommendations** — none. The plan and Spec were already corrected during the milestone reviews: the false "disjoint by construction" invariant was fixed and Task 3.0's fold is documented as chosen option (a) in Spec §8 ("Clean edit inside a conflict hunk — FOLDED"). Plan Core-concepts rows match the code at their stated paths; all three Plan checkboxes are accurately ticked.
