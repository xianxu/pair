# Boundary Review — pair#89 (milestone M3)

| field | value |
|-------|-------|
| issue | 89 — review mode should disable edit while agent update the doc |
| repo | pair |
| issue file | workshop/issues/000089-review-mode-should-disable-edit-while-agent-update-the-doc.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | f8d2c119a96a4744634a4281b41842371a6d64fd..HEAD |
| command | sdlc milestone-close --issue 89 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-07-05T17:09:40-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I have enough to finalize. All pure tests (`gate`, `reconcile`, `poke_bodies`) and glue tests (`review-reconcile-test.sh`, `review-loop-test.sh`, `review-window-test.sh`) pass clean; I traced the gate wiring, the fast/reconcile branch, the fold, and the durability paths against Spec §1/§2/§8 and the M3 Done-when. Here's the review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The M3 apply-gate is well-built and does what the Spec promises: `gate.decide_apply` is a clean pure five-case decision, the defer path secures the human's edits to disk before stashing, the winbar/spinner states are mutually exclusive, and Alt+Return's dual dispatch (apply-pending vs submit) is wired and headless-proven. Every M3 Plan task maps to a boundary-tagged commit (including Task 3.0's fold, `8c869f3`), the fold correctly folds a clean edit sharing a human-changed line into the conflict marker (killing the M2-review 3.1 drop), and no silent-drop path survives. Nothing here blocks the boundary — no crashes, no correctness bug, no behavior drift from the stated contract. What keeps it from a straight SHIP is one genuine test-coverage gap on a stated durability Done-when (save-on-`VimLeave` for edits typed *after* a defer is code-only, untested) plus a few cheap cleanups; all non-blocking.

## 1. Strengths (confirmed-good ground)

- **`gate.decide_apply` is textbook ARCH-PURE** (`nvim/review/gate.lua:13`) — string/bool in, string out, zero vim API; `gate_test.lua` runs under `nvim -l` with no IO and pins all five cases including the `V`/`R` mid-edit variants. The gate is single-sourced (`init.lua:27` `M.gate = gate`, exposed for both the test and the UI) so the decision can't drift between callers.
- **The gate/apply-branch v1 computation is consistent.** `on_agent_round` computes `v1 = apply.buf_content(buf)` (`init.lua:122`) and `apply_round` recomputes `base = apply.buf_content(buf)` (`init.lua:53`) with the same join; `review.set_base` stores the base via review.lua's `buf_content` (`review.lua:438`) joined identically — so case-1 (`v1==v0`) in the gate and the fast-path (`base==v0`) in apply_round agree by construction. The "compares equal" spec §8 contract holds.
- **Durability-on-defer is real and tested.** `on_defer` saves *before* stashing (`review.lua:467-472`), reusing `human_round`'s one save path (ARCH-DRY); `review-window-test.sh` asserts the file on disk holds `deferred human edit line` after the defer.
- **The v0-snapshot placement is exactly the M2-review caution** — set in `finish_human_turn` *after* the save, not in `mark_awaiting` (which `request_ship` also calls without saving) — with the reason inlined at `review.lua:504-507`. This is the trap the M2 re-review §6 flagged for M3, and it's honored.
- **Statusline↔winbar coherence** (`review.lua:470` `clear_awaiting()` on defer) is asserted mutually-exclusive by the `defer-no-spinner` window-test case, not just claimed.

## 2. Critical findings

None. No crash on any traced input; the nil-base first-round edge is benign (defers if mid-edit, then Alt+Return → `apply_round` → nil base → fast path applies); no silent drop; the gate never mis-applies (the `FocusLost` benign-fallback only over-defers).

## 3. Important findings

**3.1 — save-on-`VimLeave` durability is code-only, not headless-tested, though it's an explicit M3 Done-when** — `nvim/review.lua:615-622`.

M3 Done-when: *"a modified buffer is also saved on VimLeave; Headless-tested."* The `review-window-test.sh` durability assertion covers **save-on-defer** (`defer-saves-to-disk`) but not the VimLeave path. That path is the one that catches edits typed *after* the defer while the winbar is up (spec §8 names this precisely as the second of the two saves that make the invariant hold). Since save-on-defer already fires at defer time, only post-defer edits are at risk — but that's a real window, and the whole point of §8 is "never lose the human's work." A refactor of `human_round`'s signature or the autocmd guard would silently regress it with no test to catch it. Fix: add a window-test case — after a defer, set new buffer lines, fire `VimLeave` (or call the teardown callback), assert the file on disk holds the post-defer edits. Cheap; non-blocking at the gate.

## 4. Minor findings

- **Stale `pending_records` on an out-of-protocol second handoff** — `nvim/review/init.lua:120-129` + `review.lua:446`. If a round is pending (winbar up) and a second handoff lands while the human is *not* mid-edit, `on_agent_round` → gate `'apply'` → `apply_round` applies it directly but never clears `pending_records`; the next Alt+Return then applies the stale round instead of submitting. Spec §8 says a second handoff "replaces" the pending slot — that replace only happens on the defer branch, not the apply branch. Protocol-guarded (the agent awaits the commit poke, which only fires after apply, so no second handoff is expected while pending) → defensive-only, recoverable, never corruption. If you want full §8 fidelity, clear `pending_records` at the top of the direct-apply path.
- **`buf_content` triplicated (ARCH-DRY)** — `review.lua:438` re-implements `apply.buf_content` (`apply.lua:41`); the comment even pins them to stay byte-identical. `init.lua` uses `apply.buf_content` directly. A shared reference (expose `review.buf_content`) would remove the "must match apply exactly" coupling. Trivial.
- **Focus-autocmd comment/code mismatch** — `review.lua:451-452` comment says "BufEnter/BufLeave cover the in-nvim case," but only `BufEnter` is registered (no `BufLeave`). Benign for a single-window pane (over-defer only), but the comment overstates the wiring.
- **`nearest_nonempty` deletion comment** — `reconcile.lua:107-109` claims "backward-first … the preceding kept line," but `d=0` returns `start_idx` itself (the line *following* a deletion). Cosmetic; anchor is still a real line. (Carried from the M2 re-review §4.)

## 5. Test coverage notes

- Strong on the M3 logic: gate five cases (pure), the init-level gate dispatch (loop-test `gate: mid-edit round is deferred` / `normal-mode round applies immediately`), and the full pane wiring (window-test `pane-state`/`defer-*`/`pending-*` — 8 assertions). Fold is covered both pure (`reconcile_test` fold cases) and live (`review-reconcile-test.sh` case f).
- **Gap:** the VimLeave save (§3.1). Lower-priority: no headless case for the stale-pending §4 edge (protocol-guarded, so arguably not worth one).

## 6. Architectural notes for upcoming work

- ARCH-DRY: **pass** (one apply.apply path; classify reuses `nth_offset`; save-on-defer reuses `human_round`; gate single-sourced) — modulo the `buf_content` triplication above.
- ARCH-PURE: **pass** (exemplary) — `gate`/`classify`/`conflict_marker`/`plan_conflicts` pure and IO-free-testable; `plan_conflicts` takes `hunks` as data; all focus/mode/winbar/save IO isolated to `review.lua` and injected via `pane_state`/`on_defer`.
- ARCH-PURPOSE: **pass.** Shadow-sweep of consumers — pair renders + reconciles the marker (this diff), the pair `review-protocol.md` derives the wire format from `reconcile.conflict_marker`, and the ariadne `xx-fix` agent note was verified delivered on ariadne `main` (M2 re-review §7). The real live agent recognizing review-mode is legitimately a separate issue (ariadne #000121), not the deferred point of *this* one. **One production-readiness caveat for the operator:** the M3 live smoke (mid-insert defer → winbar → Alt+Return applies; edits survive defer→quit→reopen) genuinely can't run headless and is marked "remaining" in the atlas — the `sdlc close --verified` evidence must record its *actual* result, not assert it done, since the durability-across-reopen leg is exactly what §3.1's missing test can't cover either.

## 7. Plan revision recommendations

None required — the plan and Spec now match the code (the M2-review 3.1 "disjoint by construction" invariant was corrected in `b2bf503`, and Task 3.0's fold is implemented as the plan's recommended option (a), documented in Spec §8's "Clean edit inside a conflict hunk — FOLDED (M3, done)"). If §3.1's test is deferred rather than added, add a one-line `## Revisions` note that the VimLeave-save leg is proven by the live smoke rather than headless, so the plan doesn't over-claim "Headless-tested" for that clause.
