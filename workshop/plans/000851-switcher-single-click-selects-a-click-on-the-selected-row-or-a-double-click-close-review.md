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
