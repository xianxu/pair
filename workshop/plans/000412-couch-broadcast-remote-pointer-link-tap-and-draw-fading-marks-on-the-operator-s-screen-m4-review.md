# Boundary Review — pair#412 (milestone M4)

| field | value |
|-------|-------|
| issue | 412 — Couch broadcast: remote pointer link (tap and draw fading marks on the operator's screen) |
| repo | pair |
| issue file | workshop/issues/000412-couch-broadcast-remote-pointer-link-tap-and-draw-fading-marks-on-the-operator-s-screen.md |
| boundary | milestone M4 |
| milestone | M4 |
| window | fab9e3e7547b9f5e744c0494e7c84c12cdfc2ba8..0364076f0fa92dc0d30434f9fdaf7cef9f5e1d82 |
| command | sdlc milestone-close --issue 412 --milestone M4 |
| reviewer | claude |
| timestamp | 2026-10-08T12:18:29-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I've finished checking the diff, the Spec/Done-when and the plan; writing up the verdict.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Verdict: fix then ship.** The M4 code is correct and well bounded. It covers the viewer pointer mode, the narrowed `StatusGuardCols` guard, the hold-then-fast fade, the truecolor blend, and right-click on `LIVE ⏸` to re-copy the view link. I re-checked the arithmetic: `tint` stays in range on the ladder (k ≤ 9 → index ≤ 4), `NextChange` matches the hold and fade steps, and the guard is applied both when a mark is added and when it's painted. The fail-safes (`IndicatorShown`, `PointerShown`) only read cells inside the guard, so opening the rest of the tab bar to marks is safe. The targeted Go tests pass at HEAD.

Two things block a clean SHIP:
- **The issue contract wasn't revised.** The smoke changed it, but the issue's Spec and Done-when still say status-row points are dropped and that marks fade in 3 steps over about 3s. The plan's Revisions entry covers only the status-row change, not the fade.
- **The plan promised tests that weren't written.** Task 4.1 promised page-wiring tests (caps toggling capture and the hint), and they don't exist.

**1. Strengths**
- `marks.go:114`: the `guarded()` predicate is one source used in both `Add` and `Overlay` (ARCH-DRY).
- `reserve_broadcast_test.go:695`: `TestStatusGuardMatchesDrawnControls` ties the guard width to what is actually drawn, so the two can't silently drift apart.
- `viewer.js:440-470`: `cellAt` and `chunkStroke` are pure and tested in node. Each chunk repeats the previous chunk's last point, which honours the Spec's "batches overlap by one point, never joined across requests" rule.
- `server_test.go:379`: the page lint was tightened to exactly one `fetch`, to the relative `'point'`, with `credentials: 'omit'`. A security property is pinned rather than loosened.
- `marks_test.go`: the fade test runs both the ladder path and the truecolor path, and checks the hold, step count and end state.

**2. Critical findings**
None.

**3. Important findings**
- **Spec and docs contradict the code (`docs-match-code`, 2nd finding in this family).**
  - The issue's Spec (lines ~94, 104, 122) still says marks fade in 3 steps over about 3s and never touch the status row. Done-when (lines ~154, 171) says the same.
  - The plan's M4 Revisions entry covers the guard change but not the fade change.
  - `atlas/broadcast.md` contradicts itself in one sentence: "Marks fade in 3 steps after a 1.5s hold … 10 steps".
  - **Rule (fix the rule, not the instance):** when a smoke round changes a contract, the same commit revises every statement of it: issue Spec, Done-when, a plan Revisions entry, and the atlas. Sweep for "3 steps", "3s" and "status row" across all four.
- **Plan table names a function that doesn't exist (`plan-table-matches-code`, 2nd).** The Core concepts table lists `batchPoints`, but the shipped function is `chunkStroke`. It also lists `Marks (… Expired)`, but the code has `Live`/`NextChange`. **Rule:** at each boundary, check every table row against `grep` for the shipped symbol, and record renames in Revisions.
- **Page wiring is untested (`untested-page-wiring`, 2nd).** Task 4.1 promised a test that "caps on/off toggles capture and the notice". `pointer.test.mjs` only checks that a caps event reaches a callback. `pointerMode` is untested: `setOn`, the hint text, the `.pointer` class, the final flush on pointerup, and `end()` when pointing turns off. **Rule:** any page logic beyond a DOM call goes behind a seam that node can drive. For example, `pointerMode(term, stage, hint, post)` with a fake element and an injected `post`.

**4. Minor findings**
- The truecolor blend wiring in `console_pointer.go:169` is only exercised by calling `SetBlend` directly. No couchtty test would fail if the expression were mutated to `false`.
- `viewer.js:565`: a right-button `pointerdown` also starts a stroke.
- `marks.go` `Add` comment: one line runs past the file's wrap width.

**5. Test coverage notes**
Marks tests and couchtty tests (tab-bar point, LIVE stays protected, `LIVE` right-click) pin real behaviour. The gaps are the `pointerMode` wiring and the blend path through the Console.

**6. Architectural notes**
- **ARCH-DRY:** pass. `guarded` and `StatusGuardCols` are derived from the labels.
- **ARCH-PURE:** pass for Go. Marginal for `pointerMode`, which mixes DOM, timer and fetch (see the third Important finding).
- **ARCH-PURPOSE:** pass. The smoke drove both scope changes.
- **ARCH-MOCK:** N/A. No new external dependency.
- **ARCH-CONSTRAINTS:** pass. The fade repaints at 20Hz for 0.5s, and page flushes run at 50ms, under the 30/s rate limit.
- **ARCH-SECURE:** pass. The single `fetch` is relative and sends no credentials, and the server parser is unchanged.
- **ARCH-ORDER:** pass. The page's `on`/`stroke`/`timer` state is small, and `setOn(false)` calls `end()`.
- **ARCH-FUNERAL:** pass. Nothing durable is created; marks are in memory, and the interval is cleared in `end()`.

**7. Plan revision recommendations**
- Add a Revisions entry for the fade: 3 steps over 3s → `MarkHold` 1.5s plus `MarkFade` 0.5s in 10 steps, blending to the background with truecolor and walking a 256-colour ladder without it.
- Add a Revisions entry renaming `batchPoints` → `chunkStroke` and `Expired` → `Live`/`NextChange`.
- Revise the issue's Spec and Done-when to match: points on the guarded controls are dropped, and marks hold 1.5s then are gone by about 2s.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Task 4.1 still enumerates test cases in prose; Minor, never blocks.
findings:
  - id: new
    severity: Important
    family: docs-match-code
    title: |
      Issue Spec/Done-when still say "3 steps over 3s" and "status row dropped"; atlas fade sentence self-contradicts
    detail: |
      2nd in family. Rule: a smoke-driven contract change revises Spec, Done-when, plan Revisions and atlas in the same commit; sweep "3 steps", "3s", "status row" across all four. Plan Revisions omits the fade change; atlas/broadcast.md says "fade in 3 steps ... 10 steps".
  - id: new
    severity: Important
    family: plan-table-matches-code
    title: |
      Core concepts table lists batchPoints and Marks.Expired; code ships chunkStroke and Live/NextChange
    detail: |
      2nd in family. Rule: at each boundary grep every table row's symbol and record renames in a Revisions entry.
  - id: new
    severity: Important
    family: untested-page-wiring
    title: |
      pointerMode (setOn/hint/.pointer class/flush on pointerup) untested despite Task 4.1's promised caps-toggle test
    detail: |
      2nd in family. Rule: page logic beyond a DOM call sits behind a seam node can drive (inject post plus fake stage/hint), and every promised page test exercises that seam. The node test only checks that the caps callback fires.
  - id: new
    severity: Minor
    family: untested-blend-wiring
    title: |
      Console's SetBlend(palette...) wiring has no test; a mutation to false stays green
```
