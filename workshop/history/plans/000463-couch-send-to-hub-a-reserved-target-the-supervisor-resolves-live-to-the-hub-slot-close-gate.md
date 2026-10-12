---
gate: boundary-review
issue: 463
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T00:33:40-07:00"
      dispatched: "2026-10-12T00:31:52-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: README and atlas not updated for the reserved send-to hub target
          detail: |-
            README.md:422-424 lists send-to forms without hub; atlas/couch.md:185 still says send-to repo[:N];
            atlas/couch.md:143 says every send-to resolves via ResolveRepositoryName, now false for hub.
            Add the hub row to README and an atlas note that hub is resolved live in the service via hubBinding before name resolution.
          family: docs-gate-new-surface
          round: 1
        - id: BR-2
          severity: Important
          title: Done-when clause 1 (live check) is stale after the Revision moved it to ariadne 336
          detail: |-
            Revisions moved the live delivery check to ariadne 336 but Done-when still lists it as this issue's exit criterion.
            Amend the clause to name where it is delivered.
          family: done-when-tracks-revisions
          round: 1
        - id: BR-3
          severity: Minor
          title: Reserving hub also shadows unique-prefix matches (e.g. hubble); SKILL.md says only repository names
          detail: |-
            SKILL.md reservation paragraph should say hub is reserved ahead of prefix resolution too, matching the Spec.
            No test pins that a repo named hub stays reachable as hub:N while bare hub goes to the hub slot.
          family: reserved-name-shadowing
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T00:36:24-07:00"
      dispatched: "2026-10-12T00:35:06-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: README.md:425 adds the hub row; atlas/couch.md:144 notes the exception, :186 the usage, :193-198 explain hubRecipient/hubBinding ahead of name and prefix resolution.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Done-when clause 1 now names the service test here and that the live check moved to ariadne 336, matching the Revisions entry.
          round: 2
        - id: BR-3
          disposition: addressed
          note: SKILL.md now covers prefixes (hubble); TestMessageHubTargetIsReservedAheadOfRepositoryNames pins bare hub vs hub:3 and fails with the rewrite disabled (verified by revert).
          round: 2
      findings:
        - id: BR-4
          severity: Minor
          title: atlas/couch.md:143-144 has two consecutive parentheticals; SKILL.md says hub:N where hubble is meant
          detail: 'The window is small, so I checked every instance: these two are the only wording issues; README and the --help usage line read correctly. Move the exception after the repositoryname.go path, and write hub:N / hubble in SKILL.md.'
          family: docs-gate-new-surface
          round: 2
        - id: BR-5
          severity: Minor
          title: Plan Tests line omits TestMessageHubTargetIsReservedAheadOfRepositoryNames added in the fix round
          detail: 'The rule: the Plan and Done-when must be updated in the same commit as whatever changes their claims. Here the review-fix commit added a test without updating the Plan list. This is the only instance in the window.'
          family: done-when-tracks-revisions
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: 3c8791ab6240bb4862bec2b12802514de2d3cafd
      blocked: false
---

# Gate ledger — pair#463 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T00:33:40-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `docs-gate-new-surface` README and atlas not updated for the reserved send-to hub target
  README.md:422-424 lists send-to forms without hub; atlas/couch.md:185 still says send-to repo[:N];
  atlas/couch.md:143 says every send-to resolves via ResolveRepositoryName, now false for hub.
  Add the hub row to README and an atlas note that hub is resolved live in the service via hubBinding before name resolution.
- **BR-2** [Important] `done-when-tracks-revisions` Done-when clause 1 (live check) is stale after the Revision moved it to ariadne 336
  Revisions moved the live delivery check to ariadne 336 but Done-when still lists it as this issue's exit criterion.
  Amend the clause to name where it is delivered.
- **BR-3** [Minor] `reserved-name-shadowing` Reserving hub also shadows unique-prefix matches (e.g. hubble); SKILL.md says only repository names
  SKILL.md reservation paragraph should say hub is reserved ahead of prefix resolution too, matching the Spec.
  No test pins that a repo named hub stays reachable as hub:N while bare hub goes to the hub slot.

## Round 2 — 2026-10-12T00:36:24-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — README.md:425 adds the hub row; atlas/couch.md:144 notes the exception, :186 the usage, :193-198 explain hubRecipient/hubBinding ahead of name and prefix resolution.
- BR-2 — addressed — Done-when clause 1 now names the service test here and that the live check moved to ariadne 336, matching the Revisions entry.
- BR-3 — addressed — SKILL.md now covers prefixes (hubble); TestMessageHubTargetIsReservedAheadOfRepositoryNames pins bare hub vs hub:3 and fails with the rewrite disabled (verified by revert).

### Raised

- **BR-4** [Minor] `docs-gate-new-surface` atlas/couch.md:143-144 has two consecutive parentheticals; SKILL.md says hub:N where hubble is meant
  The window is small, so I checked every instance: these two are the only wording issues; README and the --help usage line read correctly. Move the exception after the repositoryname.go path, and write hub:N / hubble in SKILL.md.
- **BR-5** [Minor] `done-when-tracks-revisions` Plan Tests line omits TestMessageHubTargetIsReservedAheadOfRepositoryNames added in the fix round
  The rule: the Plan and Done-when must be updated in the same commit as whatever changes their claims. Here the review-fix commit added a test without updating the Plan list. This is the only instance in the window.

## Open findings

- **BR-4** [Minor] `docs-gate-new-surface` atlas/couch.md:143-144 has two consecutive parentheticals; SKILL.md says hub:N where hubble is meant
- **BR-5** [Minor] `done-when-tracks-revisions` Plan Tests line omits TestMessageHubTargetIsReservedAheadOfRepositoryNames added in the fix round
