# Boundary Review — pair#730 (whole-issue close)

| field | value |
|-------|-------|
| issue | 730 — couch: switcher draws hub rows underlined, not reversed, so the selection is the only reverse-video row |
| repo | pair |
| issue file | workshop/issues/000730-couch-switcher-draws-hub-rows-underlined-not-reversed-so-the-selection-is-the-only-reverse-video-row.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0aa4bee4ad83c49eea27226d718236e01698ad26..4d5e387c09e72ec67ba9c4db75c477a5e5c8d2f0 |
| command | sdlc close --issue 730 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T11:11:40-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The rendering change matches the Spec, and targeted tests pass. Update the stale README description before shipping. The recorded suite remains a separate landing prerequisite; its pass is not established by this review.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: user-facing-doc-consistency
    title: |
      README still describes switcher hubs as reversed
    detail: |
      README.md:1238 says both the hub tab and switcher row use reversed colours, contradicting menu_render.go:695–705 and the corrected atlas/couch.md:1748–1750. Update the paragraph to distinguish the reversed tab chip from the underlined switcher hub, with reverse reserved for selection. Family sweep found this single stale description in README.md and atlas/.
```

1. **Strengths**
   - Reuses the existing reset-restoring decorator, preserving styles through glyph resets.
   - Tests all 16 combinations of hub, selection, blocked, and color states at the terminal-cell boundary.
   - Leaves tab-chip styling and the foreign-hub dimming branch unchanged.

2. **Critical findings:** None.

3. **Important findings:** README inconsistency above. This prose correction needs source inspection, not a wording-presence test.

4. **Minor findings:** None.

5. **Test coverage:** `go test ./cmd/internal/couchtty -run 'TestHub|TestRenderMenu' -count=1` passed. Assertions cover underline, selection reverse, selected-hub bold, blocked red, and style reset after the row. Existing multi-hub tests check foreign-hub dimming. Live inspection is explicitly deferred by the revised acceptance criteria.

6. **Architecture**
   - **ARCH-DRY: pass** — modifies the existing decorator without introducing parallel logic.
   - **ARCH-PURE: pass** — deterministic rendering remains directly testable without IO.
   - **ARCH-PURPOSE: pass** — all hub rows receive underline; reverse follows selection.
   - **ARCH-EVOLVE: pass** — no persisted format changes.

7. **Plan revisions:** None required. No Core concepts table is present.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"4647af9f-143b-422a-a62d-20131a1ee87f","repository":"github.com/xianxu/pair","issue_id":"000730","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/1b2275b46ca2c4e2","reviewed_head":"4d5e387c09e72ec67ba9c4db75c477a5e5c8d2f0","slot":"pair:4","requested_at":"2026-10-20T18:10:50.589596Z"},"phase":"finished","pid":83362,"started_at":"2026-10-20T18:10:50.786434Z","finished_at":"2026-10-20T18:11:39.267803Z","session_id":"01a15a82-4c99-7000-be42-c50942484eda","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.627268083s
head: "4d5e387c09e72ec67ba9c4db75c477a5e5c8d2f0"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T11:13:56-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 730 — couch: switcher draws hub rows underlined, not reversed, so the selection is the only reverse-video row |
| repo | pair |
| issue file | workshop/issues/000730-couch-switcher-draws-hub-rows-underlined-not-reversed-so-the-selection-is-the-only-reverse-video-row.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0aa4bee4ad83c49eea27226d718236e01698ad26..e4fd5c1cd2372111d926c0cfddfb4e7faf0b8c83 |
| command | sdlc close --issue 730 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T11:13:56-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The change matches the revised Spec: hub rows are underlined, selection controls reverse video, and tab styling remains unchanged. BR-1 is addressed; no new findings. The recorded suite remains a separate landing prerequisite and was not verified here.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:1238–1241 now distinguishes reversed hub tabs from underlined switcher rows, reserving reverse for selection. This matches menu_render.go:695–705 and atlas/couch.md:1738–1750; the README/atlas sweep found no remaining stale description.
```

1. **Strengths**
   - Reuses the existing decorator and restores hub styling after glyph resets.
   - Tests all 16 combinations of hub, selection, blocked, and color states.
   - Updates both README and atlas consistently.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   - `go test ./cmd/internal/couchtty -run 'TestHub|TestRenderMenu' -count=1` passed, including existing multi-hub dimming checks.
   - Running the new regression test with the base renderer through a temporary Go overlay failed on the unselected hub’s reverse attribute, confirming the test catches the original bug.
   - Tracked files remain unchanged.

6. **Architectural notes**
   - **ARCH-DRY: pass** — modifies the existing styling path.
   - **ARCH-PURE: pass** — deterministic rendering is tested directly without IO.
   - **ARCH-PURPOSE: pass** — hub styling composes with selection and preserves foreign-hub dimming.
   - **ARCH-EVOLVE: pass** — no persisted formats change.

7. **Plan revision recommendations:** None. The plan matches the implementation; live inspection is explicitly deferred.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"08dfa981-acf6-4571-92e0-350daed393ba","repository":"github.com/xianxu/pair","issue_id":"000730","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/e2b82ccf78a236d5","reviewed_head":"e4fd5c1cd2372111d926c0cfddfb4e7faf0b8c83","slot":"pair:4","requested_at":"2026-10-20T18:13:01.546561Z"},"phase":"finished","pid":75621,"started_at":"2026-10-20T18:13:01.724068Z","finished_at":"2026-10-20T18:13:55.406761Z","session_id":"01a15a84-4c13-76e0-9810-b5ab1f9f3dcd","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.843471541s
head: "e4fd5c1cd2372111d926c0cfddfb4e7faf0b8c83"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
