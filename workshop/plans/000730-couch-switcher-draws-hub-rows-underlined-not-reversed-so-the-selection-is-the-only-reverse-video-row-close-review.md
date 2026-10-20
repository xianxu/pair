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
