# Boundary Review — pair#521 (whole-issue close)

| field | value |
|-------|-------|
| issue | 521 — couch: draw the hub's tab in inverted colours, like a bookmark |
| repo | pair |
| issue file | workshop/issues/000521-couch-draw-the-hub-s-tab-in-inverted-colours-like-a-bookmark.md |
| boundary | whole-issue close |
| milestone | — |
| window | 08e14f2ae729a0181259101c65e20da79f4d1bd2..e77acff84b296392c3cd73ccbe6e19164546da0c |
| command | sdlc close --issue 521 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-12T22:29:28-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The rendering changes appear consistent with the expanded Spec: hub inversion preserves existing tab colours, selection remains visible, and styling resets prevent leakage. README and atlas were updated. The blocking issue is stale acceptance criteria. Runtime validation remains incomplete because the package test could not compile without generated runtime assets.

Done-when: stale — it covers tab-bar rendering but omits the now-required switcher behaviour and still requires live approval despite the recorded waiver.

1. **Strengths**
   - `cmd/internal/couchtty/reserve.go:366`: inversion composes with existing colour precedence and applies separately to glyphs.
   - `cmd/internal/couchtty/menu_render.go:618`: hub selection retains its arrow, adds bold, and preserves blocked red; resets restore inversion within the row.
   - `hub_inversion_test.go` checks terminal cells across 768 tab combinations, click-span preservation, folded counts, and switcher selection/blocked/colour combinations.
   - README and atlas describe the changed visual treatment.

2. **Critical findings:** None.

3. **Important findings**
   - `workshop/issues/000521-couch-draw-the-hub-s-tab-in-inverted-colours-like-a-bookmark.md:53`: synchronize acceptance criteria with the revised contract. The complete family in this window is: missing switcher main-row inversion; missing selected-hub arrow/bold and blocked-red preservation; missing attention-detail containment; and the obsolete mandatory live-approval clause. Add these criteria and reference the recorded waiver.

4. **Minor findings:** None.

5. **Test coverage notes**
   - The new tests exercise actual renderers and terminal-cell parsing without mocks.
   - `go test ./cmd/internal/couchtty` failed during setup: `cmd/internal/runtimebundle/embed.go:8:41: pattern assets/runtime/files: no matching files found`.
   - `Makefile.local:247` defines the required asset-generation step. No assets were generated under this read-only review. The reported full-suite success therefore remains independently unverified.
   - No prior findings required regression disposition.

6. **Architectural notes**
   - **ARCH-DRY: pass.** Existing hub projection, colour helpers, and rendering paths are reused; no substantial duplicate behaviour was introduced.
   - **ARCH-PURE: pass.** Changes remain within deterministic presentation functions, with no new IO.
   - **ARCH-PURPOSE: pass for implementation.** Both tab and switcher consumers receive the requested treatment. Acceptance criteria need to catch up.
   - No Core concepts table or new architectural surface requires additional cross-checking.

7. **Plan revision recommendations**
   - Add a `## Revisions` entry recording that `## Done when` now covers switcher inversion, selection/red composition, attention-detail containment, and the existing smoke waiver.

```findings
findings:
  - id: new
    severity: Important
    family: acceptance-criteria-track-contract
    title: |
      Done when does not cover the expanded rendering contract
    detail: |
      workshop/issues/000521-couch-draw-the-hub-s-tab-in-inverted-colours-like-a-bookmark.md:53 still names only tab-bar acceptance. Enumerated gaps are switcher main-row inversion, selected-hub arrow/bold and blocked-red preservation, unchanged attention-detail styling, and mandatory live approval superseded by the recorded waiver. Update Done when to cover all these contract changes and record the synchronization in Revisions.
```

---

## Re-review — 2026-10-12T22:32:43-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 521 — couch: draw the hub's tab in inverted colours, like a bookmark |
| repo | pair |
| issue file | workshop/issues/000521-couch-draw-the-hub-s-tab-in-inverted-colours-like-a-bookmark.md |
| boundary | whole-issue close |
| milestone | — |
| window | 08e14f2ae729a0181259101c65e20da79f4d1bd2..23fb1c94d4f03d07b2844bd359be561d472a288f |
| command | sdlc close --issue 521 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-12T22:32:43-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

BR-1 is addressed: acceptance criteria now cover the expanded rendering contract and recorded smoke waiver. Source inspection found no blocking implementation defect. Runtime confidence is limited because package tests could not compile without generated runtime assets.

1. **Strengths**
   - `reserve.go:366` composes inversion with existing colour precedence and preserves click spans.
   - `menu_render.go:618` confines inversion to the hub main row, retaining selected bold, arrow, and blocked red.
   - Cell-level tests exercise 768 tab combinations plus folding and switcher states.
   - README and atlas both document the changed presentation.

2. **Critical findings:** None.

3. **Important findings:** None remaining.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Tests exercise real, deterministic renderers without mocks.
   - `git diff --check` passed.
   - `go test ./cmd/internal/couchtty` failed during setup: `runtimebundle/embed.go:8` requires missing `assets/runtime/files`. Full-suite success remains independently unverified.
   - BR-1 is a prose-only correction; no behavioral regression test is required for its disposition.

6. **Architectural notes**
   - **ARCH-DRY: pass.** Reuses existing hub projection and colour helpers.
   - **ARCH-PURE: pass.** Changes stay within deterministic presentation functions.
   - **ARCH-PURPOSE: pass.** Both tab and switcher consumers implement the specified treatment.
   - No Core concepts table or state-format migration applies.

7. **Plan revision recommendations:** None; the synchronization is recorded under `## Revisions`.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The issue's Done when at lines 53–55 now covers both rendering surfaces, selected arrow/bold and blocked red, unchanged attention details, containment, and the recorded smoke waiver. Revisions at lines 101–104 records this synchronization. These criteria match the approved composition at lines 32–49 and the implementation in reserve.go:366 and menu_render.go:618; this is a prose-only correction.
```
