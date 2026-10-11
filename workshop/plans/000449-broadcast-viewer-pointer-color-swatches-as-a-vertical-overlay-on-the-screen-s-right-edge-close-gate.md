---
gate: boundary-review
issue: 449
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-11T12:58:52-07:00"
      dispatched: "2026-10-11T12:58:03-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: Point-through-the-rail Done-when clause is pinned only by a CSS/HTML regex test
          detail: rail.test.mjs:25-34 asserts the selectors, not hit-testing. This is acceptable because node cannot hit-test, but the evidence is structural, not behavioral. It is the only instance in this window.
          family: done-when-clause-behavioral-evidence
          round: 1
        - id: BR-2
          severity: Minor
          title: Rail is outside the stage, so the stale dimming does not apply to the swatches
          detail: 'The #stage.stale opacity and grayscale (viewer.css) skip #rail. If wanted, add a body-level stale class or a matching #rail rule. #rail is the only sibling of the stage that overlays the screen.'
          family: stale-state-styling-scope
          round: 1
      recipe: small-diff-review
      isolation: fenced
      reviewed: 93616acc7c4a32dc82e7a676332d389c66e24a2a
      blocked: false
---

# Gate ledger — pair#449 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-11T12:58:52-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `done-when-clause-behavioral-evidence` Point-through-the-rail Done-when clause is pinned only by a CSS/HTML regex test
  rail.test.mjs:25-34 asserts the selectors, not hit-testing. This is acceptable because node cannot hit-test, but the evidence is structural, not behavioral. It is the only instance in this window.
- **BR-2** [Minor] `stale-state-styling-scope` Rail is outside the stage, so the stale dimming does not apply to the swatches
  The #stage.stale opacity and grayscale (viewer.css) skip #rail. If wanted, add a body-level stale class or a matching #rail rule. #rail is the only sibling of the stage that overlays the screen.

## Open findings

- **BR-1** [Minor] `done-when-clause-behavioral-evidence` Point-through-the-rail Done-when clause is pinned only by a CSS/HTML regex test
- **BR-2** [Minor] `stale-state-styling-scope` Rail is outside the stage, so the stale dimming does not apply to the swatches
