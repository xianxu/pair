---
gate: boundary-review
issue: 305
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-23T12:50:51-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Important
          title: 'Define the #306 reservation spanning selection and provisioning'
          detail: |-
            The plan delegates reservation to #306 but does not define its token, owner, lifetime, or atomic handoff across number selection and provisioning. Make the contract executable and declare pair#306 as a dependency.
            (carried from plan-quality PQ-2, deferred to the boundary review)
          family: cross-issue-capability-contract
          round: 1
        - id: BR-2
          severity: Important
          title: 'Define the #306 reservation spanning selection and provisioning'
          detail: |-
            This is the 2nd finding in family `cross-issue-capability-contract`. State the reusable rule for every cross-issue capability: define the token/identity, authority, acquisition point after selection, ownership, lifetime through provisioning and launch, atomic handoff, and release/recovery on failure or cancellation; update the issue dependency metadata to declare pair#306. The current plan only says that #306 reserves before provisioning and releases on failure (plan:41-45), while also declaring that #305 has no reservation capability (plan:37-40), so competing automatic callers can still select the same workspace without an executable exclusion contract.
            (carried from plan-quality PQ-8, deferred to the boundary review)
          family: cross-issue-capability-contract
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-23T12:50:51-07:00"
      agent: codex
      findings:
        - id: BR-3
          severity: Critical
          title: Core concepts and Task 2 name a nonexistent fixture file
          detail: 'workshop/plans/000305-slots-v2-workspace-provisioning-plan.md:96-103,294-295 names provision_fake_test.go, but ProvisionFixture is implemented in cmd/internal/couchcore/provision_git_test.go. Update the plan and record the correction in ## Revisions.'
          family: plan-entity-table-truth
          round: 2
      recipe: milestone-review
      blocked: true
---

# Gate ledger — pair#305 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-23T12:50:51-07:00 (sdlc) — passed

### Raised

- **BR-1** [Important] `cross-issue-capability-contract` Define the #306 reservation spanning selection and provisioning
  The plan delegates reservation to #306 but does not define its token, owner, lifetime, or atomic handoff across number selection and provisioning. Make the contract executable and declare pair#306 as a dependency.
  (carried from plan-quality PQ-2, deferred to the boundary review)
- **BR-2** [Important] `cross-issue-capability-contract` Define the #306 reservation spanning selection and provisioning
  This is the 2nd finding in family `cross-issue-capability-contract`. State the reusable rule for every cross-issue capability: define the token/identity, authority, acquisition point after selection, ownership, lifetime through provisioning and launch, atomic handoff, and release/recovery on failure or cancellation; update the issue dependency metadata to declare pair#306. The current plan only says that #306 reserves before provisioning and releases on failure (plan:41-45), while also declaring that #305 has no reservation capability (plan:37-40), so competing automatic callers can still select the same workspace without an executable exclusion contract.
  (carried from plan-quality PQ-8, deferred to the boundary review)

## Round 2 — 2026-09-23T12:50:51-07:00 (codex) — BLOCKED

### Raised

- **BR-3** [Critical] `plan-entity-table-truth` Core concepts and Task 2 name a nonexistent fixture file
  workshop/plans/000305-slots-v2-workspace-provisioning-plan.md:96-103,294-295 names provision_fake_test.go, but ProvisionFixture is implemented in cmd/internal/couchcore/provision_git_test.go. Update the plan and record the correction in ## Revisions.

## Open findings

- **BR-1** [Important] `cross-issue-capability-contract` Define the #306 reservation spanning selection and provisioning
- **BR-2** [Important] `cross-issue-capability-contract` Define the #306 reservation spanning selection and provisioning
- **BR-3** [Critical] `plan-entity-table-truth` Core concepts and Task 2 name a nonexistent fixture file
