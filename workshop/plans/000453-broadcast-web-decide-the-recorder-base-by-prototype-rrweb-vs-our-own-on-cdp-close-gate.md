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

## Open findings

- **BR-1** [Important] `done-when-clause-unevidenced` Done-when requires a summary in the ops project Log; nothing in the window shows it
- **BR-2** [Minor] `summary-claim-exceeds-data` Summary claims are stronger than the tables behind them
- **BR-3** [Minor] `gap-list-incomplete` B's coverage-gap list leaves out form values inside closed shadow roots
- **BR-4** [Minor] `evidence-not-reproducible` The prototypes lived in a session scratchpad, not the prototypes/ dir or scratch branch the Spec allowed
