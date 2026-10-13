---
gate: boundary-review
issue: 506
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T17:37:46-07:00"
      dispatched: "2026-10-12T17:35:20-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: Right-click notice still says the copied link points and takes notes; at M1 it is view-only
          detail: cmd/internal/couchtty/console_webcast.go:263 copies u.Link (now view-only) with the text "(watch, point, notes)". M2 Task 8 must change it; there is also no Couch path to the pointer link until then.
          family: couch-notice-matches-link-grants
          round: 1
        - id: BR-2
          severity: Minor
          title: atlas/webcast-format.md:431 still says a pin's chip opens the note with Resolve
          detail: Under the view link Resolve is hidden (annotate.js:505); qualify the sentence or point to the new paragraph in atlas/webcast.md.
          family: atlas-tracks-surface-change
          round: 1
        - id: BR-3
          severity: Minor
          title: Plan Task 4's "all three links clear when the mount is lost" assertion is not tested
          detail: TestCastPublishesThreeLinks only checks publication and that a cast without a board has no pointer link; cast.go:277 clearing PointerLink has no regression test.
          family: plan-step-test-delivered
          round: 1
        - id: BR-4
          severity: Minor
          title: The link URL Base()+"/"+token+"/" is built three times in Cast.run
          detail: cast.go:268-273; a small linkURL(base, token) helper would be the single source (ARCH-DRY nit).
          family: shared-helper-for-repeated-construction
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: e6ecba8e381805ad9c0558c6b5f3e2a78ff6b2dc
      blocked: false
---

# Gate ledger — pair#506 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T17:37:46-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `couch-notice-matches-link-grants` Right-click notice still says the copied link points and takes notes; at M1 it is view-only
  cmd/internal/couchtty/console_webcast.go:263 copies u.Link (now view-only) with the text "(watch, point, notes)". M2 Task 8 must change it; there is also no Couch path to the pointer link until then.
- **BR-2** [Minor] `atlas-tracks-surface-change` atlas/webcast-format.md:431 still says a pin's chip opens the note with Resolve
  Under the view link Resolve is hidden (annotate.js:505); qualify the sentence or point to the new paragraph in atlas/webcast.md.
- **BR-3** [Minor] `plan-step-test-delivered` Plan Task 4's "all three links clear when the mount is lost" assertion is not tested
  TestCastPublishesThreeLinks only checks publication and that a cast without a board has no pointer link; cast.go:277 clearing PointerLink has no regression test.
- **BR-4** [Minor] `shared-helper-for-repeated-construction` The link URL Base()+"/"+token+"/" is built three times in Cast.run
  cast.go:268-273; a small linkURL(base, token) helper would be the single source (ARCH-DRY nit).

## Open findings

- **BR-1** [Minor] `couch-notice-matches-link-grants` Right-click notice still says the copied link points and takes notes; at M1 it is view-only
- **BR-2** [Minor] `atlas-tracks-surface-change` atlas/webcast-format.md:431 still says a pin's chip opens the note with Resolve
- **BR-3** [Minor] `plan-step-test-delivered` Plan Task 4's "all three links clear when the mount is lost" assertion is not tested
- **BR-4** [Minor] `shared-helper-for-repeated-construction` The link URL Base()+"/"+token+"/" is built three times in Cast.run
