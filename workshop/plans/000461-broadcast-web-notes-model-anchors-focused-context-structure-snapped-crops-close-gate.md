---
gate: boundary-review
issue: 461
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T01:53:15-07:00"
      dispatched: "2026-10-12T01:51:35-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: selectorFor emits [data-testid="..."] steps whose value contains " > " or " >>> ", which matchSelector can never match
          detail: 'selector.go:72-78 rejects only a quote in the value; matchSelector splits on the separators before parsing steps. Probe: data-testid="a > b" gives a selector matching 0 elements, so re-render resolve fails. Skip values containing ">" and add a TestSelectorSteps case.'
          family: selector-generator-matcher-symmetry
          round: 1
        - id: BR-2
          severity: Minor
          title: Template-content nodes get a host >>> path selector, but matchSelector never steps into Content
          detail: selector.go:55 handles template content when generating, but the matcher (and the walkElements round-trip helper) covers only Shadow and Frame. Either refuse to anchor such nodes or match into Content.
          family: selector-generator-matcher-symmetry
          round: 1
        - id: BR-3
          severity: Minor
          title: model.go's Context.Landmark doc says Node itself when none is closer; Structural returns the parent element
          detail: 'The atlas and plan describe the parent element (body at most). Report to the TL so the doc for #457''s seam matches.'
          family: seam-doc-drift
          round: 1
        - id: BR-4
          severity: Minor
          title: Atlas says crops less than a margin apart merge; cluster merges at a gap <= margin
          family: seam-doc-drift
          round: 1
        - id: BR-5
          severity: Minor
          title: Plan file task checkboxes remain unchecked although the issue marks all four tasks done
          family: plan-bookkeeping
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T01:56:29-07:00"
      dispatched: "2026-10-12T01:55:15-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: selector.go:68 skips quote/`>` values and structural.go:27 drops any selector that doesn't round-trip; reverting either turns TestSelectorSteps / TestAnchorDropsASelectorThatCantRoundTrip red.
          round: 2
        - id: BR-2
          disposition: addressed
          note: matchSelector steps into h.Content (selector.go:88) and walkElements covers Content; the `#row >>> b` case fails when the change is reverted.
          round: 2
        - id: BR-3
          disposition: addressed
          note: model.go Context.Landmark doc now says the parent element when there is no landmark, matching landmark.go:36-39 and the atlas.
          round: 2
        - id: BR-4
          disposition: addressed
          note: The atlas now says "at most a margin apart"; the sibling code comment crop.go:86 is still stale and is raised as a new seam-doc-drift finding.
          round: 2
        - id: BR-5
          disposition: addressed
          note: All plan Task 1-4 checkboxes are ticked in 62293ee2.
          round: 2
      findings:
        - id: BR-6
          severity: Minor
          title: Code comments still describe the old behaviour this round changed (crop.go:86 "less than margin", selector.go:85 omits template content)
          detail: '3rd finding in seam-doc-drift. Rule: a behaviour-changing commit must grep and update every prose restatement (doc comments, atlas, plan) in the same commit. Sweep notes/*.go comments against the atlas notes-model section, not just these two lines.'
          family: seam-doc-drift
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 62293ee2868491c4e9e7b401407ab777a0cd5779
      blocked: false
    - "n": 3
      timestamp: "2026-10-12T01:59:11-07:00"
      dispatched: "2026-10-12T01:58:28-07:00"
      agent: claude
      dispose:
        - id: BR-6
          disposition: not-addressed
          note: 'crop.go:86 and selector.go:85/132 fixed, but the sweep is partial: model.go:35-37 still says Anchor yields "a stable selector" though Structural.Anchor blanks Sel when it does not round-trip (the atlas says so), and selector.go:16 omits the quote and > exclusion. Rule: a behaviour change must also update every comment that describes it, in the same commit.'
          round: 3
      findings:
        - id: BR-7
          severity: Minor
          title: Conflict resolution in atlas/webcast-format.md leaves no blank line before the Attribution and control heading
          detail: This is the 4th finding in family seam-doc-drift; it is a formatting slip from the merge, not a new defect class. Line 228 "Region notes ... no region." runs straight into "## Attribution and control (#466)". It renders under CommonMark but differs from every other section break; add a blank line.
          family: seam-doc-drift
          round: 3
      recipe: fix-review
      isolation: fenced
      reviewed: e3ca1a7bfeb789cc14ca1c42c46f036325e9152b
      blocked: false
---

# Gate ledger — pair#461 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T01:53:15-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `selector-generator-matcher-symmetry` selectorFor emits [data-testid="..."] steps whose value contains " > " or " >>> ", which matchSelector can never match
  selector.go:72-78 rejects only a quote in the value; matchSelector splits on the separators before parsing steps. Probe: data-testid="a > b" gives a selector matching 0 elements, so re-render resolve fails. Skip values containing ">" and add a TestSelectorSteps case.
- **BR-2** [Minor] `selector-generator-matcher-symmetry` Template-content nodes get a host >>> path selector, but matchSelector never steps into Content
  selector.go:55 handles template content when generating, but the matcher (and the walkElements round-trip helper) covers only Shadow and Frame. Either refuse to anchor such nodes or match into Content.
- **BR-3** [Minor] `seam-doc-drift` model.go's Context.Landmark doc says Node itself when none is closer; Structural returns the parent element
  The atlas and plan describe the parent element (body at most). Report to the TL so the doc for #457's seam matches.
- **BR-4** [Minor] `seam-doc-drift` Atlas says crops less than a margin apart merge; cluster merges at a gap <= margin
- **BR-5** [Minor] `plan-bookkeeping` Plan file task checkboxes remain unchecked although the issue marks all four tasks done

## Round 2 — 2026-10-12T01:56:29-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — selector.go:68 skips quote/`>` values and structural.go:27 drops any selector that doesn't round-trip; reverting either turns TestSelectorSteps / TestAnchorDropsASelectorThatCantRoundTrip red.
- BR-2 — addressed — matchSelector steps into h.Content (selector.go:88) and walkElements covers Content; the `#row >>> b` case fails when the change is reverted.
- BR-3 — addressed — model.go Context.Landmark doc now says the parent element when there is no landmark, matching landmark.go:36-39 and the atlas.
- BR-4 — addressed — The atlas now says "at most a margin apart"; the sibling code comment crop.go:86 is still stale and is raised as a new seam-doc-drift finding.
- BR-5 — addressed — All plan Task 1-4 checkboxes are ticked in 62293ee2.

### Raised

- **BR-6** [Minor] `seam-doc-drift` Code comments still describe the old behaviour this round changed (crop.go:86 "less than margin", selector.go:85 omits template content)
  3rd finding in seam-doc-drift. Rule: a behaviour-changing commit must grep and update every prose restatement (doc comments, atlas, plan) in the same commit. Sweep notes/*.go comments against the atlas notes-model section, not just these two lines.

## Round 3 — 2026-10-12T01:59:11-07:00 (claude) — passed

### Disposed

- BR-6 — not-addressed — crop.go:86 and selector.go:85/132 fixed, but the sweep is partial: model.go:35-37 still says Anchor yields "a stable selector" though Structural.Anchor blanks Sel when it does not round-trip (the atlas says so), and selector.go:16 omits the quote and > exclusion. Rule: a behaviour change must also update every comment that describes it, in the same commit.

### Raised

- **BR-7** [Minor] `seam-doc-drift` Conflict resolution in atlas/webcast-format.md leaves no blank line before the Attribution and control heading
  This is the 4th finding in family seam-doc-drift; it is a formatting slip from the merge, not a new defect class. Line 228 "Region notes ... no region." runs straight into "## Attribution and control (#466)". It renders under CommonMark but differs from every other section break; add a blank line.

## Open findings

- **BR-6** [Minor] `seam-doc-drift` Code comments still describe the old behaviour this round changed (crop.go:86 "less than margin", selector.go:85 omits template content)
- **BR-7** [Minor] `seam-doc-drift` Conflict resolution in atlas/webcast-format.md leaves no blank line before the Attribution and control heading
