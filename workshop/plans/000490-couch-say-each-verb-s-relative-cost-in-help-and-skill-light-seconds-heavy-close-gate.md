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
    - "n": 2
      timestamp: "2026-10-12T11:22:22-07:00"
      dispatched: "2026-10-12T11:21:51-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Revisions entry records light as "under a second", and --prepare/--reload-context as heavy; matches cost.go verbCosts.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Revisions entry amends Done-when, naming the eight untimed verbs as tagged from a shared timed path.
          round: 2
        - id: BR-3
          disposition: addressed
          note: README now carries costGuidance verbatim and cost_test.go:100-106 asserts it. Base README fails the flattened containment (checked by hand; go test cannot build here without generated assets).
          round: 2
      recipe: fix-review
      isolation: fenced
      reviewed: 42a0a322c32fc8685371680be048dfe165051f4e
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

## Round 2 — 2026-10-12T11:22:22-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Revisions entry records light as "under a second", and --prepare/--reload-context as heavy; matches cost.go verbCosts.
- BR-2 — addressed — Revisions entry amends Done-when, naming the eight untimed verbs as tagged from a shared timed path.
- BR-3 — addressed — README now carries costGuidance verbatim and cost_test.go:100-106 asserts it. Base README fails the flattened containment (checked by hand; go test cannot build here without generated assets).

## Open findings

(none — every finding has been disposed)
