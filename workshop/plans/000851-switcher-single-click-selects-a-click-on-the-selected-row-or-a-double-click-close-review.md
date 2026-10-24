# Boundary Review — pair#851 (whole-issue close)

| field | value |
|-------|-------|
| issue | 851 — Switcher: single click selects, a click on the selected row or a double click switches |
| repo | pair |
| issue file | workshop/issues/000851-switcher-single-click-selects-a-click-on-the-selected-row-or-a-double-click.md |
| boundary | whole-issue close |
| milestone | — |
| window | 715b1e199cf2368c46c7e43cb9a4d8c67c4624e3..4c251d1ab612e26805c9c387979838b8a8ee7f58 |
| command | sdlc close --issue 851 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T10:44:05-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation follows the revised select-first design, preserves shared activation checks, and passes the reviewed tests. Two documentation gaps need correction: README still promises immediate switching, and Done-when still requires a timing window while omitting the expanded chip acceptance cases. No executable correctness defect was found.

Done-when: stale — it names a double-click timing window despite the timer-free contract and does not explicitly cover chip behavior inside/outside the switcher or hidden/nested target reveal.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: user-facing-behavior-documentation
    title: |
      README update is missing for the changed switcher click behavior.
    detail: |
      README.md:1925-1928 still says any switcher-row click switches immediately and every click is a manual switch; README.md:1226 also promises an unqualified one-click warm-attachment retry. These are all affected README claims in this window. Update them to distinguish selection from activation, chips inside versus outside the switcher, and retrying an already-selected row. atlas/couch.md:3292 already documents the new policy, but README.md has no change in the pinned range.
  - id: new
    severity: Important
    kind: defect
    family: acceptance-contract-synchronization
    title: |
      Done-when has not been reconciled with the revised click contract.
    detail: |
      workshop/issues/000851-switcher-single-click-selects-a-click-on-the-selected-row-or-a-double-click.md:67-71 retains the double-click timing-window requirement, while Spec:38-44 explicitly rejects a timer and adds scoped chip behavior. The stale acceptance family comprises the timing-window clause and missing explicit acceptance for inside/outside chips, hidden-target reveal with selective filter/scope preservation, and nested-frame selection before activation. Revise Done-when to name those observable outcomes and rapid/delayed press tests.
```

1. **Strengths**

   - `menu_mouse.go:7` keeps selection decisions pure and uses canonical row identity.
   - `menu.go:2002` shares the existing manual activation path, preserving operation checks and origin handling.
   - Reducer tests cover recordless slots, inventory replacement, hidden chips, nested frames, operation vetoes, and exact effect counts.
   - Real SGR tests exercise routing and reject release, motion, wheel, modified, and right-button activation.

2. **Critical findings:** None.

3. **Important findings:** The two documentation findings above. Both are inexpensive corrections; neither requires a runtime change.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed:

   - Full `couchtty` suite: 51.647s.
   - Focused mouse/switcher race tests: 3.508s.
   - Production artifact classification guard: 2.290s.
   - Affected-config qualification against the pinned dependency: 12.973s.
   - `git diff --check`.

   PURE tests exercise real reducer logic without IO; integration tests use injected host, child, and dispatcher fixtures.

6. **Architectural notes**

   - **ARCH-DRY: pass** — activation policy is shared.
   - **ARCH-PURE: pass** — selection logic is pure; Console owns IO and effect dispatch.
   - **ARCH-PURPOSE: pass** — both row and chip consumers receive the scoped policy; external chips and group-count actions retain their behavior.
   - **ARCH-EVOLVE: pass** — no persisted format changes.

   The core-concepts table matches the inspected entities and boundaries. Atlas coverage is present.

7. **Plan revision recommendations**

   Add a `## Revisions` entry reconciling Done-when with timer-free activation and the chip acceptance cases. Correct both affected README passages in the same documentation sweep.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"5b3689f0-f2b3-42aa-88f4-31a6c95f0f1e","repository":"github.com/xianxu/pair","issue_id":"000851","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/a384579be7e3defa","reviewed_head":"4c251d1ab612e26805c9c387979838b8a8ee7f58","slot":"pair:2","requested_at":"2026-10-24T17:41:42.430356Z"},"phase":"finished","pid":60062,"started_at":"2026-10-24T17:41:42.563935Z","finished_at":"2026-10-24T17:43:45.773136Z","session_id":"01a16f01-11fd-7d02-b49e-5fb85bf5d0d6","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.386636834s
head: "4c251d1ab612e26805c9c387979838b8a8ee7f58"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-24T10:55:23-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 851 — Switcher: single click selects, a click on the selected row or a double click switches |
| repo | pair |
| issue file | workshop/issues/000851-switcher-single-click-selects-a-click-on-the-selected-row-or-a-double-click.md |
| boundary | whole-issue close |
| milestone | — |
| window | 715b1e199cf2368c46c7e43cb9a4d8c67c4624e3..7a8a64ffd9de577d689963a02abab36e2aaabcfd |
| command | sdlc close --issue 851 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T10:55:23-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Both prior documentation findings are addressed. The implementation preserves shared activation checks and passes the existing tests, but a new regression probe confirms that revealing a chip can unnecessarily erase a valid filter, violating the selective-preservation contract.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:1226-1227 now qualifies warm retry by selection; README.md:1926-1937 distinguishes selection, activation, scoped chips, nested menus and manual activation. These prose corrections match terminal_input.go:278-289, menu_mouse.go:7-35 and the shared activation path; atlas/couch.md:194 also qualifies retry.
  - id: BR-2
    disposition: addressed
    note: |
      The issue's Done-when:69-72 replaces the timing window with rapid/delayed timer-free presses and explicitly covers scoped chips, hidden/nested reveal, restriction preservation, external chips and keyboard behavior. These match Spec:38-44; existing mouse and reducer tests substantiate the documented gestures.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: preserve-effective-reveal-context
    title: |
      Chip reveal evaluates the filter outside its retained group and can erase valid context.
    detail: |
      cmd/internal/couchtty/menu_mouse.go:24-25 tests Filter using a frame without GroupScope. Filtering has exact-over-fuzzy precedence (menu.go:430-433), so results depend on the candidate group. Reproduced with a pair-group slot tagged foo-more, an outside brain-group slot tagged foo, Filter="foo", and Focus view hiding the summaryless target: the target matches the scoped filter, but clicking its chip clears both Focus view and Filter. Only Focus view should clear, per Done-when:70. A temporary overlay regression, TestReviewRevealPreservesScopedFuzzyFilter, failed with filter="" on pinned HEAD. ARCH-PURPOSE: preserve restrictions in their effective context. Evaluate the filter with the retained GroupScope after any necessary scope removal, then evaluate View. Family enumeration: the sole context-losing predicate is lines 24-25; scope membership at lines 21-22 is independent of filtering, and the final view check at lines 27-28 already uses the retained frame. Add regression coverage for this mixed-restriction case.
```

1. **Strengths**

   - Canonical row keys preserve recordless-slot identity and resolve current inventory.
   - Activation reuses the existing manual dispatch and operation vetoes.
   - Real SGR tests cover rapid/delayed presses and non-activation reports.

2. **Critical findings**

   - The selective-reveal defect above requires correction and a retained regression test.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed full `couchtty` tests (61.528s), focused race tests (3.780s), artifact classification (4.814s), pinned affected-config qualification (51.082s), and diff whitespace checks. The temporary overlay probe exposed the missing combination; repository files remain unchanged.

6. **Architectural notes**

   - **ARCH-DRY: pass** — shared activation authority.
   - **ARCH-PURE: pass** — deterministic reducer logic; thin Console routing with injected integration fixtures.
   - **ARCH-PURPOSE: flag** — selective context preservation fails in the reproduced case.
   - **ARCH-EVOLVE: pass** — no persisted-format change.

   Core-concept locations and classifications match the code. README and atlas updates are present.

7. **Plan revision recommendations**

   Add a `## Revisions` entry requiring reveal predicates to retain applicable group context, with a regression combining scoped fuzzy matching, an outside exact match, and Focus-view hiding.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"0e1a5019-333b-444f-a40c-67d8a501a9cc","repository":"github.com/xianxu/pair","issue_id":"000851","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/1cf5cd8101fa1ecc","reviewed_head":"7a8a64ffd9de577d689963a02abab36e2aaabcfd","slot":"pair:2","requested_at":"2026-10-24T17:51:57.736792Z"},"phase":"finished","pid":66720,"started_at":"2026-10-24T17:51:57.905696Z","finished_at":"2026-10-24T17:54:36.627601Z","session_id":"01a16f0a-73be-7e70-9383-bd0720daa837","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 11.1656505s
head: "7a8a64ffd9de577d689963a02abab36e2aaabcfd"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-24T11:04:25-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 851 — Switcher: single click selects, a click on the selected row or a double click switches |
| repo | pair |
| issue file | workshop/issues/000851-switcher-single-click-selects-a-click-on-the-selected-row-or-a-double-click.md |
| boundary | whole-issue close |
| milestone | — |
| window | 715b1e199cf2368c46c7e43cb9a4d8c67c4624e3..768030bc64f275c875e4cee671b0dd1873b96426 |
| command | sdlc close --issue 851 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T11:04:25-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the revised select-first contract. BR-3 is corrected with a meaningful regression that fails when the fix is removed. Documentation and acceptance criteria are synchronized, and no additional findings remain.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:1226-1227 qualifies warm retry by selection; README.md:1926-1937 documents row/chip selection, activation, hidden/nested targets and manual switching, matching terminal_input.go:278-289 and menu_mouse.go:7-35.
  - id: BR-2
    disposition: addressed
    note: |
      The issue's Done-when:69-72 now matches the timer-free Spec and explicitly covers scoped chips, selective reveal, nested frames, external chips and keyboard behavior; reducer and real SGR tests substantiate these outcomes.
  - id: BR-3
    disposition: addressed
    note: |
      menu_mouse.go:24 retains GroupScope when evaluating Filter. TestSwitcherRevealPreservesEffectiveFilterContext covers scoped preservation and necessary scope/filter removal. Reverting only this fix through a temporary overlay makes the preservation case fail with filter=""; pinned HEAD passes.
```

1. **Strengths**

   - Canonical row keys preserve selection across recordless slots and inventory replacement (`menu_mouse_test.go:51-85`).
   - Activation shares existing manual dispatch and operation vetoes (`menu.go:2002`).
   - Real SGR tests cover rapid/delayed presses, chip routing and non-activation reports (`console_mouse_test.go:297-537`).

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed:

   - Full couchtty suite: 78.647s.
   - Focused race tests: 5.539s.
   - Production artifact classification: 8.333s.
   - Affected-config qualification against the pinned dependency: 120.855s.
   - Diff whitespace checks.

   The BR-3 regression failed without the fix. Pure tests exercise reducer logic without IO; integration fixtures inject host, child and dispatcher dependencies. Tracked files remain unchanged.

6. **Architectural notes**

   - **ARCH-DRY: pass** — shared activation authority.
   - **ARCH-PURE: pass** — deterministic selection logic with thin Console routing.
   - **ARCH-PURPOSE: pass** — row/chip consumers implement the scoped contract, including effective-context preservation.
   - **ARCH-EVOLVE: pass** — no persisted-format changes.

   Core-concept classifications match the code. README and atlas updates cover the changed surface.

7. **Plan revision recommendations:** None; the existing revisions cover the corrections.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"0123dc84-3d19-4834-bb81-ddef1e8edec6","repository":"github.com/xianxu/pair","issue_id":"000851","recipe":"small-diff-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/b7b86c82c440adfb","reviewed_head":"768030bc64f275c875e4cee671b0dd1873b96426","slot":"pair:2","requested_at":"2026-10-24T18:01:37.764022Z"},"phase":"finished","pid":57521,"started_at":"2026-10-24T18:01:37.938383Z","finished_at":"2026-10-24T18:04:17.61944Z","session_id":"01a16f13-4d4b-77d1-bc71-3a2449a00f2e","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 47.781204291s
head: "768030bc64f275c875e4cee671b0dd1873b96426"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000413
