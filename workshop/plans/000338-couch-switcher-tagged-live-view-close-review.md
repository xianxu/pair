# Boundary Review — pair#338 (whole-issue close)

| field | value |
|-------|-------|
| issue | 338 — Couch switcher space toggles a focus view of tagged live threads |
| repo | pair |
| issue file | workshop/issues/000338-couch-switcher-tagged-live-view.md |
| boundary | whole-issue close |
| milestone | — |
| window | 096a8565271eb60b93ff244c10b01f6d1b29eddc..1fb9ffa233aa6f523165793d135ebedceffd2939 |
| command | sdlc close --issue 338 |
| reviewer | codex |
| timestamp | 2026-09-28T15:14:53-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation matches the focus-view design, and the full couchtty suite passes. One small coverage gap remains: the contract requires literal Space handling with a nonempty filter in both views, but the new test exercises only focus view. No correctness defect was found in the implementation.

1. **Strengths**
   - `menu_reattach.go:352` centralizes focus membership after the existing overlay and search, preserving order and shared selection/render behavior.
   - `menu_render.go:551` reuses full-inventory labels, sanitization, and terminal-cell clipping.
   - `console_focus_view_test.go:12` exercises actual console input, description refresh/removal/restoration, and reopening in both modes.
   - README and atlas document the new interaction and its relationship to notification jumps.

2. **Critical findings:** None.

3. **Important findings**
   - `menu_focus_test.go:99`: Nonempty-filter Space is asserted only after returning to the retained **focus** view. Add the equivalent assertion in **normal** view, ideally parameterizing the test over both modes and asserting unchanged mode, literal `"r "`, and no effects. This is the sole missing instance in the two-mode literal-Space family; focus is already covered.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Passed targeted focus tests and `go test ./cmd/internal/couchtty`.
   - Coverage includes membership, published-summary precedence, sanitization, Unicode width, selection reconciliation, submenu return, mouse extents, and persistence.
   - The issue records operator smoke acceptance; I did not independently repeat that interactive smoke.

6. **Architectural notes**
   - **ARCH-DRY — pass:** Existing ordering, summary precedence, selection, and presentation helpers are reused.
   - **ARCH-PURE — pass:** New decisions remain deterministic reducer/render logic; no new IO is introduced.
   - **ARCH-PURPOSE — pass:** The implementation delivers the specified view and lifetime behavior without deferring functionality.

7. **Plan revision recommendations:** None; add the missing test within the existing acceptance scope.

```findings
findings:
  - id: new
    severity: Important
    family: interaction-mode-coverage
    title: |
      Cover literal Space with a nonempty filter in normal view
    detail: |
      cmd/internal/couchtty/menu_focus_test.go:99 exercises nonempty-filter Space only in focus view. Enumerating the two required instances: focus is covered; normal is missing. Parameterize this assertion over both views and verify that Space appends literally without changing mode or emitting effects.
```

---

## Re-review — 2026-09-28T15:19:25-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 338 — Couch switcher space toggles a focus view of tagged live threads |
| repo | pair |
| issue file | workshop/issues/000338-couch-switcher-tagged-live-view.md |
| boundary | whole-issue close |
| milestone | — |
| window | 096a8565271eb60b93ff244c10b01f6d1b29eddc..27970e7eee2bb6ca06fe2ddc0dbf9bc7ba08df6b |
| command | sdlc close --issue 338 |
| reviewer | codex |
| timestamp | 2026-09-28T15:19:25-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range satisfies the documented focus-view contract. BR-1 is addressed: literal Space now has explicit coverage in both modes. Package tests and repeated race-enabled focus tests pass. No new findings.

1. **Strengths**
   - `menu_reattach.go:352` centralizes membership after existing overlay and search, preserving order.
   - `menu_render.go:551` reuses stable labels, sanitized summaries, and terminal-cell clipping.
   - `console_focus_view_test.go:12` exercises real input, description refresh, switching, and reopening in both views.
   - README and atlas document the interaction and notification-jump distinction.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Passed `go test ./cmd/internal/couchtty`.
   - Passed `go test -race ./cmd/internal/couchtty -run 'TestMenuFocus|TestConsoleFocusView' -count=3`.
   - BR-1’s test checks `"r "`, unchanged mode, and zero effects in normal and focus views. The correction changes coverage, not production behavior.
   - Operator smoke acceptance is recorded in the issue; I did not repeat it.

6. **Architectural notes**
   - **ARCH-DRY — pass:** Shared ordering, selection, summary precedence, and presentation are reused.
   - **ARCH-PURE — pass:** New logic remains deterministic reducer/render code without new IO.
   - **ARCH-PURPOSE — pass:** Filtering, routing, refresh, and console-lifetime persistence deliver the stated purpose.

7. **Plan revision recommendations:** None.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      cmd/internal/couchtty/menu_focus_test.go:99 enumerates normal and focus views and asserts literal "r ", unchanged mode, and no effects. Both cases pass, including repeated race-enabled runs; the correction introduces no production behavior change.
```
