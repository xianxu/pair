# Boundary Review — pair#819 (whole-issue close)

| field | value |
|-------|-------|
| issue | 819 — Collapse tab bar: a one-glyph-per-slot state strip next to +N |
| repo | pair |
| issue file | workshop/issues/000819-collapse-tab-bar-a-one-glyph-per-slot-state-strip-next-to-n.md |
| boundary | whole-issue close |
| milestone | — |
| window | 09c8964de39cdb5e6bd9db20601c232446af6949..98cfab5975758da5746d8cdd9e6af0248ad9297c |
| command | sdlc close --issue 819 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T17:19:56-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the issue’s revised Spec and approved design. Hidden-member strips preserve slot order, fit whole or disappear, and share the existing group click target. No blocking defects or documentation gaps found. The recorded suite and operator-stage acceptance remain separate landing gates.

```findings
{}
```

1. **Strengths**

   - Closed glyph projection rejects malformed evidence and preserves the approved priority (`cmd/internal/couchcore/slotgit.go:51`).
   - Optional strips consume spare width after base packing, preserving mandatory actors, counts, controls, and attention (`cmd/internal/couchtty/status_layout.go:366`).
   - Drawing extends the existing group span through the separator and markers; tests exercise every physical cell through the production decoder (`cmd/internal/couchtty/collapse_mouse_test.go:13`).
   - README and atlas document the new behavior consistently with the implementation.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed:

   - Focused glyph, folded-strip, and collapse regressions.
   - Full `couchcore` package: 279.273s.
   - Full `couchtty` package: 49.645s.
   - `make test-affected-inventory`.
   - Pinned-range `git diff --check`.

   Coverage includes malformed glyphs, marker priority, presentation order, attention exclusion, width boundaries, palette/owner styling, multiline compatibility, counted-work bounds, physical clicks, and stale-frame rejection. No prior boundary findings required regression-fix dispositions.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Reuses glyph evidence, shared measurement, styling, and navigation.
   - **ARCH-PURE — pass:** Projection and layout remain deterministic; declared pure entities introduce no IO.
   - **ARCH-PURPOSE — pass:** Delivers ordered cardinality, whole-strip fallback, and whole-chip navigation.
   - **ARCH-MOCK — pass:** No new external dependency; integration tests reuse existing fake boundaries.
   - **ARCH-CONSTRAINTS — pass:** Bounded frame-local work, tested scaling, and enforced width admission.
   - **ARCH-SECURE — pass:** Compact output uses a closed alphabet; confirmed geometry guards clicks.
   - **ARCH-ORDER — pass:** No new cross-event state; existing publication and menu ownership remain authoritative.
   - **ARCH-FUNERAL — pass:** Strip data dies with its frame; no new durable artifact family.
   - **ARCH-EVOLVE — pass:** No persisted-format changes.

   Core-concept locations and modifications match the plan.

7. **Plan revision recommendations:** None. The explicitly pending landing tasks remain pending; this review does not establish their completion.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"afc7db4d-00d1-444a-973d-cc2287b88ac0","repository":"github.com/xianxu/pair","issue_id":"000819","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/8baac1c1e024b2ba","reviewed_head":"98cfab5975758da5746d8cdd9e6af0248ad9297c","slot":"pair:4","requested_at":"2026-10-23T00:14:24.478403Z"},"phase":"finished","pid":38466,"started_at":"2026-10-23T00:14:24.633543Z","finished_at":"2026-10-23T00:19:55.548469Z","session_id":"01a1661b-def4-7ce3-82b1-4bc26ad29816","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.942771791s
head: "98cfab5975758da5746d8cdd9e6af0248ad9297c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
