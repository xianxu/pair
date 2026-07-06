# Boundary Review — pair#89 (milestone M1)

| field | value |
|-------|-------|
| issue | 89 — review mode should disable edit while agent update the doc |
| repo | pair |
| issue file | workshop/issues/000089-review-mode-should-disable-edit-while-agent-update-the-doc.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 01c77c3e49ed75cfc60b58fd4d2126582d0020ac^..HEAD |
| command | sdlc milestone-close --issue 89 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-07-05T15:49:29-07:00 |
| verdict | SHIP |

## Review

All suites pass and I've verified the code against the source. Here's my review.

```verdict
verdict: SHIP
confidence: high
```

**Summary.** M1 delivers exactly its stated purpose — multi-line `🤖<…>` markers now highlight across rows and resolve from any spanned line, and the section budget is raised so a conflict-sized quote parses. The implementation is byte-faithful to the retired per-line `highlight_spans` (verified offset math against the old code and the tests), the new `spans_multiline` is genuinely pure (`lines → spans`, no vim API), and coverage is strong at both the unit (`markers_test`) and glue (`review-window-test`) levels. I ran `make test-lua`, `make test-review`, and `tests/review-window-test.sh` — all green, including the three new M1 assertions. Nothing blocks the boundary; the only finding is a small ARCH-DRY duplication worth folding into M2.

**1. Strengths**
- **Real DRY consolidation at the parser level (ARCH-DRY pass).** `spans_multiline` now derives from the single multi-line `parse_markers` (`markers.lua:247`) instead of the old per-line rescanner that couldn't see across rows — one highlight path, and `grep` confirms no production caller of `highlight_spans` survives.
- **Byte-faithful port.** The quoted/strike spans still start at the 🤖 itself and sections at their bracket, with `end_col` exclusive-through-closer — I traced `'before 🤖<q>[u]{a} after'` by hand (`end_col == 14`) and it matches the retired behavior exactly (`markers_test.lua:96`). The port promise held.
- **Both budget bounds pinned.** The 60-line body now parses (would've failed at 50) *and* the 210-line stray still yields no marker (`markers_test.lua:68,111`) — the 50→200 change is bracketed on both sides, not just asserted downward.
- **Glue tested where the bug would ship.** `resolve_at_cursor` from the marker's *second* line (`review-window-test.sh:196`) and a cross-row extmark assertion (`:117`) exercise the actual failure modes, not mocks.
- **`resolve_at_cursor` containment is careful** (`review.lua:289-294`): outer line-range gate + a fallback target + precise `after_start/before_end` refinement handles overlapping same-line markers correctly (I walked the two-marker-on-one-line case).

**2. Critical findings** — none.

**3. Important findings** — none.

**4. Minor findings**
- **ARCH-DRY: `spans_multiline` copy-pastes the offset→pos helper (`markers.lua:230-240`).** Its `line_starts` construction + binary-search `pos_of` are a verbatim duplicate of `parse_markers`' own `offset_to_pos` (`markers.lua:146-162`) — same file, near-identical functions. There is even a third, semantically-equivalent exported version (`reconstruct.pos_at`, linear scan). Consolidate to a shared local `build_pos_index(lines) → pos_of` in `markers.lua` that both `parse_markers` and `spans_multiline` call; if `offset_to_pos` ever changes, `pos_of` will silently drift. Non-blocking and correct as-is; M2 (which adds `reconcile.lua` also doing offset math) is the natural place to fold this in. Cite ARCH-DRY.
- **`resolve_paragraph_to_cursor` still keys paragraph membership off `m.line` only** (`review.lua:323`) — a marker that *starts* above a paragraph but *ends* inside it isn't captured. The plan (Task 1.3) explicitly deferred this as YAGNI and the added characterization test only covers a marker fully inside one paragraph. Acceptable for M1; note it if reconcile markers ever straddle a blank line.

**5. Test coverage notes**
- Plan Task 1.3 mentioned a `jump_marker` multi-line assertion ("`]m` lands on the marker's start line"); no such test was added. Behavior is trivially correct — `jump_marker` keys off `pick.line` (the start line) and I confirmed forward/backward from inside a multi-line marker both behave sensibly — so this is a documentation-vs-delivery mismatch in the plan, not a real gap. The paragraph-resolve half *did* get its characterization test (`review-window-test.sh:204`).
- The budget upper bound is tested at 210 (>200 fails) and 60 (passes); no exact-boundary (200/201) case, but the bracketing is sufficient.

**6. Architectural notes for upcoming work**
- ARCH-PURE holds cleanly here (pure `spans_multiline`, thin `render_markers` glue). When M2 lands `reconcile.classify/plan_conflicts` as pure with `hunks`-as-data, keep the same discipline — the plan's Core-concepts table already commits to it.
- Do the offset→pos consolidation (Minor #1) as part of M2 rather than a separate pass, since `reconcile.plan_conflicts` will need line/offset mapping too and `reconstruct.line_of`/`pos_at` already exist — pick one source before a fourth copy appears.

**7. Plan revision recommendations** — none required. The plan's M1 Core-concepts rows (`markers.spans_multiline` modified, `MULTILINE_LINE_BUDGET` 50→200 modified) match the code at the stated paths, and the M1 Plan checkbox is accurately ticked. If the operator wants strict plan-vs-delivery fidelity, add a one-line note to Task 1.3 that the `jump_marker` assertion was folded into the audit/no-code-change outcome rather than a standalone test — optional, not blocking.
