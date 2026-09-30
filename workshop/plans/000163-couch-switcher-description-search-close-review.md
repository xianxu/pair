# Boundary Review — pair#163 (whole-issue close)

| field | value |
|-------|-------|
| issue | 163 — Match and show actor descriptions in Couch switcher |
| repo | pair |
| issue file | workshop/issues/000163-couch-switcher-description-search.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6e58383e4d5839f08a22cd282727465227d5d0e0..734623b2e78a8387119d48d1e655faf8c66c0da9 |
| command | sdlc close --issue 163 |
| reviewer | codex |
| timestamp | 2026-09-30T12:40:22-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Description search and rendering are implemented cleanly, with README and atlas updates. One contract violation blocks shipping: exact-reference precedence is enforced separately by row kind, allowing newly added description matches to compete with explicit selections.

1. **Strengths**
   - Search and display reuse the sanitized `menuFocusSummary` projection.
   - Tests exercise both views, both row kinds, case-insensitivity, source precedence, and rendered hit targets.
   - CLI resolution leaves the optional description field empty, preserving its matching surface.

2. **Critical findings**
   - [menu.go:356](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:356) and [menu.go:379](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:379): exact-reference precedence does not cover the complete inventory. All affected cases are enumerated below. Resolve explicit references and exact tags across the inventory before admitting fuzzy description matches; add regression tests in both views.

3. **Important findings:** None separate from the blocking finding.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Full `couchtty` suite passed.
   - Focused core reference tests passed.
   - `git diff --check` passed.
   - Broader `couchcore` suite had not completed at reporting time.
   - Existing exact-reference tests use homogeneous row pairs and miss cross-kind competition.

6. **Architectural notes**
   - **ARCH-DRY: pass.** Existing summary projection and ordinary-thread matcher are reused.
   - **ARCH-PURE: pass.** Changes remain deterministic and introduce no IO.
   - **ARCH-PURPOSE: flag.** The promised exact-reference precedence is incomplete across row kinds.

7. **Plan revision recommendation**
   - Append a `## Revisions` entry extending precedence verification to mixed inventories and slot-owned opaque tags in both views.

```findings
findings:
  - id: new
    severity: Critical
    family: exact-reference-precedence-across-inventory
    title: |
      Description matches bypass exact-reference precedence across row kinds.
    detail: |
      cmd/internal/couchtty/menu.go:356-379 applies exact-over-fuzzy matching only to ordinary rows and independently appends slots. Enumerated affected cases: an ordinary exact tag plus a slot description containing that tag; a slot exact tag plus an ordinary description containing it; a slot exact tag plus another slot description containing it; and an explicit repo:N or :N selection plus an ordinary description containing that reference. Each admits an unrelated description match alongside the exact target, in both default and focus views when rows are live and described. Apply reference precedence across the complete inventory before fuzzy matching, and add both-view regressions for every case. The tests at menu_description_test.go:64 cover only ordinary/ordinary tag competition and slot/slot numeric references. ARCH-PURPOSE: the Spec and Done when explicitly promise preserved exact-reference precedence.
```

---

## Re-review — 2026-09-30T12:45:28-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 163 — Match and show actor descriptions in Couch switcher |
| repo | pair |
| issue file | workshop/issues/000163-couch-switcher-description-search.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6e58383e4d5839f08a22cd282727465227d5d0e0..5e585cac82a1fdb23fca060dca3829f0a272a7f3 |
| command | sdlc close --issue 163 |
| reviewer | codex |
| timestamp | 2026-09-30T12:45:28-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned changes satisfy the issue’s description-search contract. BR-1 is addressed across the enumerated row combinations in both views, with regression tests that fail when the correction is removed. No new blocking findings.

1. **Strengths**
   - Search and rendering share the sanitized `menuFocusSummary` projection.
   - Exact-match precedence now covers the complete inventory.
   - CLI callers leave the description field empty, preserving CLI matching.
   - README and atlas document the changed behavior.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   - Full `couchtty` suite passed.
   - Focused core reference tests passed.
   - A temporary Go overlay restoring the pre-fix filter made BR-1’s regression tests fail.
   - Existing tests cover focus exclusions and description-free behavior.
   - `git diff --check` passed. The broader `couchcore` suite had not completed at reporting time.

6. **Architecture**
   - **ARCH-DRY: pass.** Reuses the shared classifier and displayed-summary projection.
   - **ARCH-PURE: pass.** Matching and rendering remain deterministic, without new IO.
   - **ARCH-PURPOSE: pass.** Both views and row kinds receive description search with inventory-wide precedence.

7. **Plan revisions:** None needed; the appended BR-1 revision matches the correction.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      menu.go:354-391 applies exact-reference precedence across the complete inventory. menu_description_test.go:87-119 covers all five enumerated combinations in both views. These tests pass at pinned HEAD and fail with the pre-fix menu.go supplied through a temporary Go overlay.
```
