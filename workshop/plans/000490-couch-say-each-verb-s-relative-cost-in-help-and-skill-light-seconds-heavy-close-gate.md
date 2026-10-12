---
gate: boundary-review
issue: 490
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T11:20:33-07:00"
      dispatched: "2026-10-12T11:17:45-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: Spec still puts --prepare and --reload-context in "seconds" and defines light as milliseconds; the code says heavy and under a second
          detail: The Log explains the change, but the Spec text was never updated. Add a Revisions entry so the Spec matches cost.go.
          family: spec-not-revised-after-design-change
          round: 1
        - id: BR-2
          severity: Minor
          title: Done-when says the tags are backed by measured timings, but eight verbs were tagged by reasoning, not timed
          detail: --reconcile, --reap, --reload-context, --prepare, --reboot, --recover, --send-to and the bare launch were not timed. The Log says so; the Done-when wording should too.
          family: done-when-overclaims-evidence
          round: 1
        - id: BR-3
          severity: Minor
          title: README rewords costGuidance by hand and no test checks it (ARCH-DRY)
          detail: README.md:447-451 restates the tier meanings and the run-heavy-verbs-one-at-a-time advice. Only --help and the skill are checked against cost.go.
          family: unenforced-restatement-of-source
          round: 1
      recipe: milestone-review
      isolation: fenced
      reviewed: ad8c8828b2fc4d6f97afa522633f86f1684bbf76
      blocked: false
---

# Gate ledger — pair#490 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T11:20:33-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `spec-not-revised-after-design-change` Spec still puts --prepare and --reload-context in "seconds" and defines light as milliseconds; the code says heavy and under a second
  The Log explains the change, but the Spec text was never updated. Add a Revisions entry so the Spec matches cost.go.
- **BR-2** [Minor] `done-when-overclaims-evidence` Done-when says the tags are backed by measured timings, but eight verbs were tagged by reasoning, not timed
  --reconcile, --reap, --reload-context, --prepare, --reboot, --recover, --send-to and the bare launch were not timed. The Log says so; the Done-when wording should too.
- **BR-3** [Minor] `unenforced-restatement-of-source` README rewords costGuidance by hand and no test checks it (ARCH-DRY)
  README.md:447-451 restates the tier meanings and the run-heavy-verbs-one-at-a-time advice. Only --help and the skill are checked against cost.go.

## Open findings

- **BR-1** [Minor] `spec-not-revised-after-design-change` Spec still puts --prepare and --reload-context in "seconds" and defines light as milliseconds; the code says heavy and under a second
- **BR-2** [Minor] `done-when-overclaims-evidence` Done-when says the tags are backed by measured timings, but eight verbs were tagged by reasoning, not timed
- **BR-3** [Minor] `unenforced-restatement-of-source` README rewords costGuidance by hand and no test checks it (ARCH-DRY)
