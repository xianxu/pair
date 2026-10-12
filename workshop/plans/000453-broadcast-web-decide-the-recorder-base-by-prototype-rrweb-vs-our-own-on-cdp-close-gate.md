---
gate: boundary-review
issue: 453
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-11T20:41:38-07:00"
      dispatched: "2026-10-11T20:40:54-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Done-when requires a summary in the ops project Log; nothing in the window shows it
          detail: Done-when says the comparison is summarized in teams/devinfra/projects/couch-broadcast-web/project.md (ops repo). The window only touches the pair issue file. Write or confirm the project-Log summary and put a pointer to it in this issue's Log before closing.
          family: done-when-clause-unevidenced
          round: 1
        - id: BR-2
          severity: Minor
          title: Summary claims are stronger than the tables behind them
          detail: 'All instances: "~1.5x gz bytes" (the ratios are 1.15x/1.48x/1.63x/2.05x, and Wikipedia is 2x); "at least as faithful on every page" (on HN, B lagged on 3 moments vs A''s 1); "lowest page overhead" (measured only on the stress page); B''s estimate "7-8d" while its parts sum to 7-7.5d.'
          family: summary-claim-exceeds-data
          round: 1
        - id: BR-3
          severity: Minor
          title: B's coverage-gap list leaves out form values inside closed shadow roots
          detail: The fidelity table says B's observer covers form values "not inside closed roots", but section 3's B gaps do not list it.
          family: gap-list-incomplete
          round: 1
        - id: BR-4
          severity: Minor
          title: The prototypes lived in a session scratchpad, not the prototypes/ dir or scratch branch the Spec allowed
          detail: The measurements can't be re-run. Push an unmerged scratch branch and name it in the Log.
          family: evidence-not-reproducible
          round: 1
      recipe: small-diff-review
      isolation: checkout
      blocked: true
    - "n": 2
      timestamp: "2026-10-11T20:43:58-07:00"
      dispatched: "2026-10-11T20:43:15-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'ops commit 2dc3159 on main adds the "2026-10-11 — pair#453 recorder base: prototypes compared" entry; this issue''s Log points to it, and its numbers match the issue tables.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: Ratios now exact (1.15/1.48/1.63/2.05, checked); fidelity claim scoped to medians with the HN lag disclosed; overhead scoped to the stress page; B estimate 7–7.5d matches its parts. One residual from the new wording is raised separately.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Section 3's B gaps now list form values inside closed shadow roots, with the fix (DOM.resolveNode or a main-world hook).
          round: 2
        - id: BR-4
          disposition: addressed
          note: Branch 453-recorder-prototypes (afe773a4) is on origin with prototypes/broadcast-web-recorder/ and a README; the Log names it.
          round: 2
      findings:
        - id: BR-5
          severity: Minor
          title: '"about half a day more of build" understates the A/B estimate gap (6–7d vs 7–7.5d is 0.5–1d)'
          detail: 'This is the 2nd finding in family summary-claim-exceeds-data. The rule: each comparison figure in the summary is computed from the cited table ranges, endpoints included, not rounded to the favourable end. Instances in the window: 1, this phrase in the Recommendation; all other summary figures were checked and match. Fix: write "half a day to a day more".'
          family: summary-claim-exceeds-data
          round: 2
      recipe: small-diff-review
      isolation: checkout
      reviewed: e2313925a654d41f67f5dcc265dc8aa121d37e6b
      blocked: false
---

# Gate ledger — pair#453 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-11T20:41:38-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `done-when-clause-unevidenced` Done-when requires a summary in the ops project Log; nothing in the window shows it
  Done-when says the comparison is summarized in teams/devinfra/projects/couch-broadcast-web/project.md (ops repo). The window only touches the pair issue file. Write or confirm the project-Log summary and put a pointer to it in this issue's Log before closing.
- **BR-2** [Minor] `summary-claim-exceeds-data` Summary claims are stronger than the tables behind them
  All instances: "~1.5x gz bytes" (the ratios are 1.15x/1.48x/1.63x/2.05x, and Wikipedia is 2x); "at least as faithful on every page" (on HN, B lagged on 3 moments vs A's 1); "lowest page overhead" (measured only on the stress page); B's estimate "7-8d" while its parts sum to 7-7.5d.
- **BR-3** [Minor] `gap-list-incomplete` B's coverage-gap list leaves out form values inside closed shadow roots
  The fidelity table says B's observer covers form values "not inside closed roots", but section 3's B gaps do not list it.
- **BR-4** [Minor] `evidence-not-reproducible` The prototypes lived in a session scratchpad, not the prototypes/ dir or scratch branch the Spec allowed
  The measurements can't be re-run. Push an unmerged scratch branch and name it in the Log.

## Round 2 — 2026-10-11T20:43:58-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — ops commit 2dc3159 on main adds the "2026-10-11 — pair#453 recorder base: prototypes compared" entry; this issue's Log points to it, and its numbers match the issue tables.
- BR-2 — addressed — Ratios now exact (1.15/1.48/1.63/2.05, checked); fidelity claim scoped to medians with the HN lag disclosed; overhead scoped to the stress page; B estimate 7–7.5d matches its parts. One residual from the new wording is raised separately.
- BR-3 — addressed — Section 3's B gaps now list form values inside closed shadow roots, with the fix (DOM.resolveNode or a main-world hook).
- BR-4 — addressed — Branch 453-recorder-prototypes (afe773a4) is on origin with prototypes/broadcast-web-recorder/ and a README; the Log names it.

### Raised

- **BR-5** [Minor] `summary-claim-exceeds-data` "about half a day more of build" understates the A/B estimate gap (6–7d vs 7–7.5d is 0.5–1d)
  This is the 2nd finding in family summary-claim-exceeds-data. The rule: each comparison figure in the summary is computed from the cited table ranges, endpoints included, not rounded to the favourable end. Instances in the window: 1, this phrase in the Recommendation; all other summary figures were checked and match. Fix: write "half a day to a day more".

## Open findings

- **BR-5** [Minor] `summary-claim-exceeds-data` "about half a day more of build" understates the A/B estimate gap (6–7d vs 7–7.5d is 0.5–1d)
