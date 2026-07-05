# Boundary Review — pair#89 (milestone M2)

| field | value |
|-------|-------|
| issue | 89 — review mode should disable edit while agent update the doc |
| repo | pair |
| issue file | workshop/issues/000089-review-mode-should-disable-edit-while-agent-update-the-doc.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | f8d2c119a96a4744634a4281b41842371a6d64fd..HEAD |
| command | sdlc milestone-close --issue 89 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-07-05T16:16:14-07:00 |
| verdict | unknown |

## Review

Failed to authenticate. API Error: 401 Invalid authentication credentials

---

## Re-review — 2026-07-05T16:29:24-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 89 — review mode should disable edit while agent update the doc |
| repo | pair |
| issue file | workshop/issues/000089-review-mode-should-disable-edit-while-agent-update-the-doc.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | f8d2c119a96a4744634a4281b41842371a6d64fd..HEAD |
| command | sdlc milestone-close --issue 89 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-07-05T16:29:24-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I have everything I need. I ran the full `test-lua` and `test-review` suites (both green, exit 0, zero FAIL), traced the engine against the Spec/Plan/Core-concepts table, confirmed the reconcile engine is dormant in production until M3 wires `set_base`, and reproduced one real gap.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The M2 reconcile engine is well-architected and the tests genuinely pin its behavior — I ran `make test-lua` and `make test-review` and both pass clean, and the pure-core/thin-glue split is exemplary. What holds back a straight SHIP is a **verified silent-drop edge** that defeats the issue's stated core purpose (a conflict whose reconciled hunk text is blank produces *no* marker, no `dropped` count, no `conflicts` count — the intent vanishes), and a **missing consumer-side deliverable** (the ariadne `xx-fix` reconcile note that the plan's Task 2.6 lists for M2 and that the pair atlas/protocol forward-reference). Neither is a crash and neither can bite a real user at M2 (the engine only fires via test-injected `set_base` until M3), so I'm calling it non-blocking — but both are cheap and should be fixed before the close verdict is recorded, and the first *must* be fixed before M3 wires the live path.

## 1. Strengths (confirmed-good ground)

- **The "conflict = synthetic replacement record → one `apply.apply`" design is the right call and actually works.** It dissolves the clean-vs-conflict ordering/undo hazard by routing everything through one snapshot/undo block. Verified live: `review-reconcile-test.sh` case (e) proves a single undo reverts the whole reconcile round, and case (d) proves only the clean edit gets a `DiffChange` highlight (the synthetic marker self-highlights, HL count == 1).
- **ARCH-PURE is textbook here.** `classify`/`conflict_marker`/`plan_conflicts` are pure (string/table in, no vim API), `plan_conflicts` takes `hunks` as data so it's testable without `vim.diff`, and `apply.lua` is lazy-loaded (`reconcile.lua:48-52`) so the pure module + `reconcile_test.lua` never pull the buffer module at load. The pure tests run under `nvim -l` with no exec/net/fs.
- **ARCH-DRY on classification is exactly right:** `classify` reuses `reconstruct.nth_offset` with the same `occurrence or 1` fallback `apply.apply` uses (`reconcile.lua:21` vs `apply.lua:269`), so classification faithfully predicts what apply lands. One apply path, no second engine.
- **Landed accounting filters on the `reconcile` tag** (`init.lua:86-90`), not a fragile `🤖<`-prefix or a count — correctly robust to `apply.apply` reordering/dropping records and to clean marker-replacements whose `new` is itself a `🤖<old>{new}`. This is the subtle trap the plan called out, and it's handled.
- **Repeated-hunk anchoring** (`occurrence_at`, `reconcile.lua:88`) is real, not hand-waved — the `reconcile_test.lua` "repeated hunk text → occurrence 2" case pins it.
- **Escaping both marker sections + a parse round-trip assertion** closes the "a marker never fails to parse" invariant even with unbalanced brackets in quoted code; verified by the `a[0]`/`human [text]` round-trip test.

## 2. Critical findings

None that block the gate (no crashes; no correctness bug on the common path; engine dormant in prod at M2). The empty-anchor gap below is borderline-Critical — see #3.1.

## 3. Important findings

**3.1 — Silent drop of a conflict when the reconciled hunk text (or fallback anchor line) is blank** — `nvim/review/reconcile.lua:161` (guard `if anchor_old ~= ''`), reachable from the `h[4]>=1` branch (`:150`), the huge-hunk branch (`:146`), and the deletion/no-hunk branch (`:157`).

The issue exists to *kill* silent drops ("surface it as a reconcilable marker instead of a silent drop", Problem §). But when `anchor_old` comes out empty the code creates no synthetic record, and — because `dropped` only comes from `apply.apply(combined)` — the intent is lost with **zero accounting** (not in `applied`, not in `conflicts`, not in `dropped`, no poke reflecting it). Verified reproducible:

- **Case A** — human blanks the exact line the agent targeted (`v0="alpha\nbeta content\ngamma"` → `v1="alpha\n\ngamma"`, hunk `{2,1,2,1}`, agent record `beta content→BETA`): `plan_conflicts` returns `#synth == 0`.
- **Case B** — doc starts with a blank line and the no-hunk fallback anchors on line 1 (blank): `#synth == 0`.
- Control (non-blank): `#synth == 1`, as expected.

Downstream, a round that is *entirely* such conflicts yields `#enriched == 0` → `init.lua:81` returns before the poke → the agent never gets its `agent_applied` commit signal (protocol stall). Also note **the atlas overstates this as fixed**: `atlas/review-workbench.md` claims "deletions/huge-hunks never dropped" — currently false.

Fix sketch: instead of bailing when `anchor_old == ''`, search outward for the nearest non-empty kept `v1` line to anchor on (or, worst case, anchor at the top of the doc on the first non-empty line) so a conflict *always* yields a marker; add a `reconcile_test.lua` case for the blank-hunk and blank-line-1 anchors. This is edge-only and dormant until M3, so it needn't block the M2 gate — but it should land before the close verdict, and it **must** land before M3 wires `set_base` into the live path.

**3.2 — ARCH-PURPOSE: the agent-side consumer (ariadne `xx-fix` reconcile note) is not delivered, yet the pair docs forward-reference it** — Plan Task 2.6 step 2; `workshop/targets/review-protocol.md` ("The agent must recognize reconcile markers — see the `xx-fix` skill's workbench section").

I checked `../ariadne/.claude/skills/xx-fix/SKILL.md`: **zero** occurrences of "reconcile"; its "Pair review workbench" section (line 190) has no `🤖<…>[reconcile — …]` semantics. The whole point of the marker is that the *agent* reads its own blocked intent and folds it back on the next round (ARCH-PURPOSE shadow-sweep: the motivating consumer is left as a not-yet-written doc). Right now the pair atlas + protocol point at a section that doesn't exist — a dangling cross-reference. The plan places this in M2, and M2's Done-when says "Protocol docs updated" (Spec §7 names *both* the pair target and the ariadne skill). Fix: add the note to ariadne's `xx-fix` skill (single-file commit, per Task 2.6's cross-repo discipline). Non-blocking for the M2 engine, but this is a listed M2 deliverable and needed before the marker reaches a live agent (M3).

## 4. Minor findings

- **Occurrence-counter family duplication (mild ARCH-DRY):** `reconcile.occurrence_at` (`reconcile.lua:88`), `apply.new_occurrence_of` (`apply.lua:29`), and `reconstruct.nth_offset` (`reconstruct.lua:15`) are three near-siblings of "non-overlapping match counting." They have deliberately different signatures/robustness, so this isn't lazy copy-paste — but a future consolidation into `reconstruct` (the pure home) would remove the smell. Note, don't block.
- **`reconcile_round`'s third return (`#synth`, `reconcile.lua:72`) can diverge from the authoritative landed count** when a synthetic record is dropped by apply's overlap guard. `init.lua` correctly ignores it and recomputes post-apply, but `review-reconcile-test.sh` asserts on the pre-apply `#synth` — fine today (no drops in those cases), just be aware the two counts aren't identical by contract.
- Ordering of synthetic records is agent-record order, not document order (plan Task 2.3 said "document order"); harmless because `apply.apply` re-sorts by offset. Not worth a code change.

## 5. Test coverage notes

- Solid where it counts: classify (both clean/conflict + `occurrence or 1`), conflict_marker escaping/round-trip + multi-line, plan_conflicts coalesce + repeated-hunk occurrence + deletion, and the glue's fast/clean-only/conflict/mixed/single-undo cases. All green when I ran them.
- **Gaps:** (a) the **blank-anchor edge** in #3.1 — untested and currently wrong; (b) the **huge-hunk `cb > MAX_HUNK_LINES` branch** (`reconcile.lua:143-148`) — no test exercises the "(region changed — N lines)" path; (c) the **no-hunk fallback group** (`reconcile.lua:123`, `h == nil`) — untested. (b) and (c) are lower priority than (a).

## 6. Architectural notes for upcoming work (M3)

- The engine is **dormant in production at M2**: `set_base` is only called by tests, `gate.lua` doesn't exist yet, and `review.lua` has no reconcile/pending/winbar wiring — all correctly deferred to M3. This is why the two Important findings are non-blocking *now*, but #3.1 and #3.2 both become live-path prerequisites the moment M3 wires `finish_human_turn → set_base`.
- M3 should honor the plan's own caution (Task 3.3): snapshot `v0` in `finish_human_turn` *after* the save, not in `mark_awaiting` (which `request_ship` also calls without saving) — otherwise `v0` captures an unsaved buffer and every round looks like a conflict.

## 7. Plan revision recommendations

- If #3.1 is fixed in code (recommended), no plan revision is needed. If instead the operator chooses to accept blank-anchor drops, add a `## Revisions` entry to `workshop/plans/000089-concurrent-edit-reconciliation-plan.md` documenting the exception, and correct `atlas/review-workbench.md` ("deletions/huge-hunks never dropped" → the true guarantee), since the atlas currently asserts an invariant the code doesn't hold.
- For #3.2, no plan change — the plan already lists Task 2.6 step 2; it's simply not done. Either complete it now or move it explicitly to M3 with a `## Revisions` note so the pair atlas/protocol forward-reference isn't left dangling at the recorded M2 close.
