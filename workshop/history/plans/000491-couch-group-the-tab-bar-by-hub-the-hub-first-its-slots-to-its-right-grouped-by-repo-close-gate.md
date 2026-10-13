---
gate: boundary-review
issue: 491
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T16:31:33-07:00"
      dispatched: "2026-10-12T16:26:47-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: A collapsed group with no :0 in it draws a bare +N that reads as the previous repository's
          detail: layoutChips hides every unkept SlotNumber>0 chip when collapsed, including the group's first member. In the default one-hub layout (hub ops, with ops:1 and ops:2) the row narrows to "ops  ariadne  :1  blog  +2  pair  +2", which misattributes the count to blog. Keep the group's first member drawn with its full label (ops:1  +1) and add a narrow-width test case for it.
          family: collapsed-count-loses-owner
          round: 1
        - id: BR-2
          severity: Important
          title: No test for the hub reaching the Console (SetHub, startMessageService, ensureMenuLocked keeping Hub)
          detail: The plan promises a wiring test. No test calls Console.SetHub. Mutating ensureMenuLocked to drop Hub left the couchtty package green, so a lost hub would silently bring back the old layout. Add a Console-level test and a couchcmd test.
          family: wiring-untested
          round: 1
        - id: BR-3
          severity: Minor
          title: A +N count cell can be clipped mid-token at the edge (pair  +)
          family: clip-mid-token
          round: 1
        - id: BR-4
          severity: Minor
          title: The gap choice is written twice (RenderStatusRow and chipsWidth); extract a cellGap helper (ARCH-DRY)
          family: shared-helper
          round: 1
        - id: BR-5
          severity: Minor
          title: Plan Revisions still describe a hubOf membership function that the code doesn't have
          family: plan-code-drift
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T16:41:00-07:00"
      dispatched: "2026-10-12T16:36:24-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: layoutChips keeps i == span[0]; reverting fails TestRenderStatusRowCollapsedCountFollowsItsOwnRepository (verified).
          round: 2
        - id: BR-2
          disposition: addressed
          note: Dropping Hub in ensureMenuLocked fails TestConsoleHubLeadsTheTabBarAcrossTheFirstMenuBuild (verified); showHub is tested in couchcmd.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Removing the whole-or-nothing check fails TestRenderStatusRowDropsACountThatDoesNotFit (verified).
          round: 2
        - id: BR-4
          disposition: addressed
          note: cellGap at reserve.go:433 is read by both RenderStatusRow and chipsWidth.
          round: 2
        - id: BR-5
          disposition: addressed
          note: The plan Revisions now say there is no hubOf. The issue checklist still names it; raised separately as a plan-code-drift repeat.
          round: 2
      findings:
        - id: BR-6
          severity: Minor
          title: The whole-group fold count trails another repository's kept chip (ops  ariadne  +2  pair:2  +3)
          detail: '2nd finding in this family. layoutChips appends the combined hidden count after the last drawn chip, often a full-labelled kept chip from another repository, so +3 (blog, pair, pair:1) reads as pair''s count; at widths 30-31 the fold count is dropped and two slots vanish without a count. The comment at reserve.go:465 claims counts always follow their own repository. Rule: every +N either directly follows a chip of the single repository it counts or looks clearly like the row''s overflow, and an overflow count that can''t be drawn means folding one more group, not dropping it. Apply this to both kinds of count in layoutChips/fitChips.'
          family: collapsed-count-loses-owner
          round: 2
        - id: BR-7
          severity: Minor
          title: The issue's Plan checklist still claims a hubOf membership function
          detail: '2nd finding in this family. Issue file line 47 is checked and names hubOf; the code puts membership inside hubSection. Rule: when a revision withdraws a named entity, grep the issue, plan, atlas and README for that name in the same change.'
          family: plan-code-drift
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 0134f9dbd86e2b809da5c33d14953f9e59a61ff0
      blocked: false
---

# Gate ledger — pair#491 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T16:31:33-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `collapsed-count-loses-owner` A collapsed group with no :0 in it draws a bare +N that reads as the previous repository's
  layoutChips hides every unkept SlotNumber>0 chip when collapsed, including the group's first member. In the default one-hub layout (hub ops, with ops:1 and ops:2) the row narrows to "ops  ariadne  :1  blog  +2  pair  +2", which misattributes the count to blog. Keep the group's first member drawn with its full label (ops:1  +1) and add a narrow-width test case for it.
- **BR-2** [Important] `wiring-untested` No test for the hub reaching the Console (SetHub, startMessageService, ensureMenuLocked keeping Hub)
  The plan promises a wiring test. No test calls Console.SetHub. Mutating ensureMenuLocked to drop Hub left the couchtty package green, so a lost hub would silently bring back the old layout. Add a Console-level test and a couchcmd test.
- **BR-3** [Minor] `clip-mid-token` A +N count cell can be clipped mid-token at the edge (pair  +)
- **BR-4** [Minor] `shared-helper` The gap choice is written twice (RenderStatusRow and chipsWidth); extract a cellGap helper (ARCH-DRY)
- **BR-5** [Minor] `plan-code-drift` Plan Revisions still describe a hubOf membership function that the code doesn't have

## Round 2 — 2026-10-12T16:41:00-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — layoutChips keeps i == span[0]; reverting fails TestRenderStatusRowCollapsedCountFollowsItsOwnRepository (verified).
- BR-2 — addressed — Dropping Hub in ensureMenuLocked fails TestConsoleHubLeadsTheTabBarAcrossTheFirstMenuBuild (verified); showHub is tested in couchcmd.
- BR-3 — addressed — Removing the whole-or-nothing check fails TestRenderStatusRowDropsACountThatDoesNotFit (verified).
- BR-4 — addressed — cellGap at reserve.go:433 is read by both RenderStatusRow and chipsWidth.
- BR-5 — addressed — The plan Revisions now say there is no hubOf. The issue checklist still names it; raised separately as a plan-code-drift repeat.

### Raised

- **BR-6** [Minor] `collapsed-count-loses-owner` The whole-group fold count trails another repository's kept chip (ops  ariadne  +2  pair:2  +3)
  2nd finding in this family. layoutChips appends the combined hidden count after the last drawn chip, often a full-labelled kept chip from another repository, so +3 (blog, pair, pair:1) reads as pair's count; at widths 30-31 the fold count is dropped and two slots vanish without a count. The comment at reserve.go:465 claims counts always follow their own repository. Rule: every +N either directly follows a chip of the single repository it counts or looks clearly like the row's overflow, and an overflow count that can't be drawn means folding one more group, not dropping it. Apply this to both kinds of count in layoutChips/fitChips.
- **BR-7** [Minor] `plan-code-drift` The issue's Plan checklist still claims a hubOf membership function
  2nd finding in this family. Issue file line 47 is checked and names hubOf; the code puts membership inside hubSection. Rule: when a revision withdraws a named entity, grep the issue, plan, atlas and README for that name in the same change.

## Open findings

- **BR-6** [Minor] `collapsed-count-loses-owner` The whole-group fold count trails another repository's kept chip (ops  ariadne  +2  pair:2  +3)
- **BR-7** [Minor] `plan-code-drift` The issue's Plan checklist still claims a hubOf membership function
