# Boundary Review — pair#841 (whole-issue close)

| field | value |
|-------|-------|
| issue | 841 — Bell notifications: highlight slot (repo:N) and issue (repo#N) references in an accent color |
| repo | pair |
| issue file | workshop/issues/000841-bell-notifications-highlight-slot-repo-n-and-issue-repo-n-references-in-an.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1ee96e9872ef195450d5c53f096f530ca7e95ee3..1f9fed7b419af84537105f82a99e9b764bf13776 |
| command | sdlc close --issue 841 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T17:07:12-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The change preserves plain-text copying and keeps reference detection separate from terminal rendering. Two matching defects block shipping: generated timestamps receive reference colors, and valid references beside paths or URLs lose their colors.

Done-when: current — it still covers the display-only behavior, punctuation/path boundaries, wrapping, and plain-text copy; the revision changes release sequencing only.

1. **Strengths**

   - Source detection happens before wrapping, preventing wrapped path fragments from becoming references.
   - Bell styling is opt-in; existing notes-menu rendering remains covered.
   - Clipboard coverage exercises Ctrl+Y and checks the actual plain-text payload.
   - `atlas/couch.md` documents the rendering change. No new command, keybinding, or configuration requires a README update.

2. **Critical findings**

```findings
findings:
  - id: new
    severity: Critical
    family: reference-highlighting-respects-source-role
    title: |
      Generated notification timestamps are highlighted as slot references
    detail: |
      cmd/internal/couchtty/console_attention.go:129 opts the entire composed row into matching, including the timestamp added at line 100. attention_references.go:12-20 accepts numeric repository names, so every HH:MM timestamp with minutes 10–59 receives the accent. This affects notify, prompt and question rows, with and without hover. Confirmed by running env TZ=Asia/Kolkata go test ./cmd/internal/couchtty -run '^TestAttentionReferenceCells/prose$' -count=1: all six assertions failed with an unexpected colored "05:36" prefix. Exclude generated timestamp metadata from reference spans, preserve legitimate numeric repository names, and use fixed timestamps covering minutes 09, 10 and 59 in regression tests. This violates the "and only those" contract (ARCH-PURPOSE).
  - id: new
    severity: Critical
    family: reference-exclusion-respects-token-boundaries
    title: |
      Path and URL exclusion suppresses separate punctuation-delimited references
    detail: |
      cmd/internal/couchtty/attention_references.go:32-36 rejects an entire whitespace chunk before separating punctuation-delimited tokens. Consequently [pair:3](https://example.test), [ariadne#575](https://example.test), and [#841](https://example.test) receive no reference accent even though their labels are outside the URL. The same guard suppresses separate references beside slash paths, backslash paths, or @-containing text; these are all instances of this single exclusion site. Apply exclusion to the actual path/URL token while retaining adjacent references, and add render regressions for all three reference forms alongside positive path/URL exclusions. Update the plan's whitespace-chunk rejection design accordingly (ARCH-PURPOSE).
```

3. **Important findings**

   None beyond the regression coverage required by the findings above.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Focused reference/notes tests passed, as did the attention/notes/toggle race subset and exhaustive artifact-source classification check. The timezone run exposed an existing wall-clock-dependent failure. Current punctuation fixtures do not combine references with neighboring paths or URLs.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Reuses wrapping, palette, cells, and canonical workspace validation.
   - **ARCH-PURE: pass.** Detection and projection remain pure; clipboard IO uses the existing integration fixture.
   - **ARCH-PURPOSE: flag.** Both findings violate the promised matching boundaries.
   - **ARCH-EVOLVE: pass.** No persisted format changes.

7. **Plan revision recommendations**

   Add a `## Revisions` entry replacing whole-whitespace-chunk rejection with token-scoped exclusion and identifying generated timestamps as unaccented metadata. Include deterministic timestamp boundary tests and mixed reference/path/URL fixtures.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"9bdd736f-4f6c-4aa5-8b9e-03626fafa986","repository":"github.com/xianxu/pair","issue_id":"000841","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/66b08b03d8458ede","reviewed_head":"1f9fed7b419af84537105f82a99e9b764bf13776","slot":"pair:9","requested_at":"2026-10-24T00:04:35.505926Z"},"phase":"finished","pid":53162,"started_at":"2026-10-24T00:04:35.684297Z","finished_at":"2026-10-24T00:07:10.905189Z","session_id":"01a16b39-3f05-7ce1-ac89-41331f3c43f1","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.398375417s
head: "1f9fed7b419af84537105f82a99e9b764bf13776"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-23T17:15:06-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 841 — Bell notifications: highlight slot (repo:N) and issue (repo#N) references in an accent color |
| repo | pair |
| issue file | workshop/issues/000841-bell-notifications-highlight-slot-repo-n-and-issue-repo-n-references-in-an.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1ee96e9872ef195450d5c53f096f530ca7e95ee3..4719c824927fb0285bb4432fd380d632c7ea7621 |
| command | sdlc close --issue 841 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T17:15:06-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-1 is fixed, with regression tests that fail when timestamp exclusion is removed. BR-2’s original examples now pass, but its exclusion rule still suppresses references following closed Markdown URLs. A separate projection defect drops accents from clipped rows containing zero-width characters. Both block shipping.

Done-when: current — the revisions refine matching boundaries; the existing clauses still cover reference-only coloring, punctuation, wrapping, and plain-text copying.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      console_attention.go:129 excludes generated timestamp bytes. Fixed-minute tests cover 09, 10 and 59, numeric repositories, all three notification kinds and both hover states. The timezone suite passes; shortening the exclusion in a disposable copy makes prose and numeric-repository regressions fail.
  - id: BR-2
    disposition: not-addressed
    note: |
      attention_references.go:12 still consumes closing Markdown delimiters and subsequent references as part of a URL. Render regressions fail for [x](https://x)(pair:3), [x](https://x)[ariadne#575], [x](https://x)[#841], and [pair:3](https://x)[#841](https://y): the references following the URL remain plain. All four cases fail across notify, prompt and question rows, with and without hover. This remains family reference-exclusion-respects-token-boundaries (ARCH-PURPOSE). Define and enforce where an enclosing Markdown URL ends while preserving brackets internal to paths/URLs; sweep all three reference forms and adjacent links rather than patching individual examples.
findings:
  - id: new
    severity: Critical
    family: reference-projection-preserves-visible-spans
    title: |
      Clipping can remove every reference accent from the final visible row
    detail: |
      attention_references.go:59-62 requires each rendered line to be a contiguous source substring, but notes_menu.go:188 clips through rowtext.Fit, which can remove zero-width characters. With source "10:09  pair:0  first\nsecond\npair:3 \u200b plain\nend", the visible third row becomes "pair:3  plain…" and projection returns nil, leaving pair:3 plain. Supplemental render tests fail independently for pair:3, ariadne#575 and #841. The family has one projector and one layout caller in this window; it affects all bell kinds and hover states whenever clipping removes an interior zero-width character. Preserve source-to-display positions through fitting instead of relying on substring identity, and assert exact accents on the clipped row (ARCH-PURPOSE).
```

1. **Strengths**

   - Detection precedes wrapping, preserving path exclusions across hard wraps.
   - Timestamp coverage is deterministic and retains legitimate numeric repository names.
   - Ctrl+Y coverage exercises the clipboard integration and verifies the exact plain-text payload.
   - Atlas documentation covers the rendering change. No new runnable surface requires a README update.

2. **Critical findings**

   BR-2 remains open, and the clipping defect above is newly identified. Both have failing render reproductions against the pinned implementation.

3. **Important findings**

   None separately; missing regression cases belong with the correctness fixes.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Reference/notes tests passed under `TZ=Asia/Kolkata`; the focused race suite and artifact classification checks also passed. Removing each prior fix independently made its existing regressions fail.

   Supplemental tests exposed both remaining defects. The current clipping assertion at `attention_references_test.go:70` checks only that *some* accent exists; the earlier `pair:0` label satisfies it even when the clipped row loses all accents.

   Disposable test files were removed; the reviewed checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Reuses canonical slot validation, wrapping, palette, and clipboard behavior.
   - **ARCH-PURE: pass.** Matching and projection remain directly testable without IO.
   - **ARCH-PURPOSE: flag.** URL exclusion and clipping still omit valid visible references.
   - **ARCH-EVOLVE: pass.** No persisted formats change.

7. **Plan revision recommendations**

   Add `## Revisions` entries defining delimiter-aware exclusion around closed Markdown destinations and correcting the assumption that fitted lines remain contiguous source substrings. Require exact clipped-row accent assertions and the complete three-form regression matrix.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"5a1ca516-4168-4699-b5c0-7b9931065abc","repository":"github.com/xianxu/pair","issue_id":"000841","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/23a39a5bba2b3856","reviewed_head":"4719c824927fb0285bb4432fd380d632c7ea7621","slot":"pair:9","requested_at":"2026-10-24T00:12:28.411525Z"},"phase":"finished","pid":30966,"started_at":"2026-10-24T00:12:28.719874Z","finished_at":"2026-10-24T00:15:02.613226Z","session_id":"01a16b40-76e6-7d83-8d04-04b39930be71","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.93990575s
head: "4719c824927fb0285bb4432fd380d632c7ea7621"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-23T17:23:09-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 841 — Bell notifications: highlight slot (repo:N) and issue (repo#N) references in an accent color |
| repo | pair |
| issue file | workshop/issues/000841-bell-notifications-highlight-slot-repo-n-and-issue-repo-n-references-in-an.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1ee96e9872ef195450d5c53f096f530ca7e95ee3..d30f7c517f25996e15b8bf190d2ad6327e7bd4a5 |
| command | sdlc close --issue 841 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T17:23:09-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation matches the revised conservative visual-aid contract. Standalone references receive accents, ambiguous syntax stays plain, and copying remains unchanged. The prior exclusion finding is superseded by the documented scope revision; the clipping defect is fixed with regression evidence. No new findings.

```findings
dispose:
  - id: BR-2
    disposition: withdrawn
    note: |
      The issue's “TL narrows acceptance to a conservative visual aid” revision explicitly permits uncolored Markdown labels and references adjacent to paths/URLs. attention_references.go:25-44 implements that conservative whole-word rule; atlas/couch.md documents it. Bracket/link render controls pass at head and fail with the previous renderer restored in a disposable copy.
  - id: BR-3
    disposition: addressed
    note: |
      notes_menu.go:155 matches final fitted rows, removing the source-substring projection dependency. TestAttentionReferenceClippedRow asserts exact clipped-row accents for pair:3, ariadne#575 and #841. All pass at head; restoring the previous renderer makes all three fail with empty accents.
```

1. **Strengths**

   - Matching remains pure and reuses canonical workspace-reference validation.
   - Bell-only opt-in preserves ordinary notes-menu rendering.
   - Tests exercise all notification kinds, hover states, deterministic timestamps, and actual Ctrl+Y clipboard output.
   - Atlas accurately documents the conservative matching behavior.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Focused rendering/copy tests, the timezone race suite, selected artifact-source guards, and diff whitespace checks passed. Regression checks against the previous renderer failed as expected. The disposable copy was removed; the pinned checkout remains clean.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Reuses validation, wrapping, palette, and clipboard infrastructure.
   - **ARCH-PURE: pass.** Detection is directly testable without IO; cell styling stays thin.
   - **ARCH-PURPOSE: pass.** Delivers the explicitly revised conservative contract.
   - **ARCH-EVOLVE: pass.** No persisted formats change.

   No new command, flag, keybinding, or configuration requires a README update.

7. **Plan revision recommendations:** None; existing revisions describe the implemented design.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"edf27a24-f45a-4b99-8740-6cedd93c2763","repository":"github.com/xianxu/pair","issue_id":"000841","recipe":"small-diff-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/91da7eb11ff49149","reviewed_head":"d30f7c517f25996e15b8bf190d2ad6327e7bd4a5","slot":"pair:9","requested_at":"2026-10-24T00:21:32.580264Z"},"phase":"finished","pid":90932,"started_at":"2026-10-24T00:21:32.808064Z","finished_at":"2026-10-24T00:23:08.724842Z","session_id":"01a16b48-c45c-7a63-9b1c-5e90d88e6d1d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.174086125s
head: "d30f7c517f25996e15b8bf190d2ad6327e7bd4a5"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
