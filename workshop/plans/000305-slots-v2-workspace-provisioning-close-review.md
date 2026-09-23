# Boundary Review — pair#305 (whole-issue close)

| field | value |
|-------|-------|
| issue | 305 — Slots v2: provision durable numbered workspaces |
| repo | pair |
| issue file | workshop/issues/000305-slots-v2-workspace-provisioning.md |
| boundary | whole-issue close |
| milestone | — |
| window | 570f8566a62c73c52edd5ca1b43befbaef028eec..ea2b507ff9d18c248309f7595c7e530ace93c63e |
| command | sdlc close --issue 305 |
| reviewer | codex |
| timestamp | 2026-09-23T12:50:51-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation passes targeted, race, and live SDLC/Weave conformance checks. The boundary is blocked by a plan-to-code contradiction: the Core concepts table and Task 2 claim `provision_fake_test.go`, but the delivered fixture is in `provision_git_test.go`.

1. Strengths

- Real Git fixtures verify remote-main capture, upstream setup, collisions, retries, and preservation of local work.
- Stateful recovery tests cover cancellation, concurrent Weave setup, marker publication, and interrupted configuration.
- Production dispatch correctly avoids supervisor/thread/agent creation.
- README and atlas documentation were updated with the new internal operation and lifecycle boundaries.

2. Critical findings

- `workshop/plans/000305-slots-v2-workspace-provisioning-plan.md:96-103,294-295` — `plan-entity-table-truth`: The Core concepts table and Task 2 name `provision_fake_test.go`, which does not exist; `ProvisionFixture` is implemented in `cmd/internal/couchcore/provision_git_test.go`. Correct the plan and add a `## Revisions` entry explaining the final fixture location.

3. Important findings

None.

4. Minor findings

None.

5. Test coverage notes

- Passed targeted provisioning tests.
- Passed `go test -race` for couchcore and couchcmd provisioning tests.
- Passed live SDLC/Weave conformance.
- Full repository suite was not independently completed during this review.

6. Architectural notes

- ARCH-DRY: pass.
- ARCH-PURE: pass; pure parsing/selection is separated from injected process/filesystem IO.
- ARCH-PURPOSE: pass; readiness is delivered while thread admission remains explicitly delegated to #306.
- ARCH-MOCK: pass; stateful fixtures share the production command seam.
- ARCH-CONSTRAINTS: pass; subprocess, output, and diagnostic bounds are enforced.
- ARCH-SECURE: pass; strict records, canonical paths, and argv-based subprocesses are used.
- ARCH-ORDER: pass; interruption and concurrent publication cases are explicitly modeled and tested.
- ARCH-FUNERAL: pass; intent cleanup and retained workspace ownership are documented.

7. Plan revision recommendations

Add:

```markdown
## Revisions

### 2026-09-23 — reconcile fixture location

The Core concepts table and Task 2 initially named `provision_fake_test.go`, but the
stateful `ProvisionFixture` implementation was delivered in
`cmd/internal/couchcore/provision_git_test.go`. Update the entity table and task
file list to match the delivered artifact.
```

```findings
findings:
  - id: new
    severity: Critical
    family: plan-entity-table-truth
    title: |
      Core concepts and Task 2 name a nonexistent fixture file
    detail: |
      workshop/plans/000305-slots-v2-workspace-provisioning-plan.md:96-103,294-295 names provision_fake_test.go, but ProvisionFixture is implemented in cmd/internal/couchcore/provision_git_test.go. Update the plan and record the correction in ## Revisions.
```

---

## Re-review — 2026-09-23T13:00:33-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 305 — Slots v2: provision durable numbered workspaces |
| repo | pair |
| issue file | workshop/issues/000305-slots-v2-workspace-provisioning.md |
| boundary | whole-issue close |
| milestone | — |
| window | 570f8566a62c73c52edd5ca1b43befbaef028eec..b245ceeefd99f1a8c43e97917fbde29382001517 |
| command | sdlc close --issue 305 |
| reviewer | codex |
| timestamp | 2026-09-23T13:00:33-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation is behaviorally strong and all tested suites pass, including full Go tests and live SDLC/Weave conformance. Boundary crossing remains blocked by the unresolved #306 reservation contract and missing dependency metadata.

1. Strengths

- Real Git fixtures validate remote baseline capture, collisions, recovery, and preservation.
- Stateful process/storage tests cover cancellation, retries, marker publication, and lock inheritance.
- Production dispatch avoids supervisor, thread, and agent creation.
- README and atlas documentation are updated.
- Plan entity mappings now match delivered files.

2. Critical findings

None.

3. Important findings

Prior findings remain open:

- BR-1 / BR-2 — `workshop/issues/000305-slots-v2-workspace-provisioning.md:3-4`, plan `:37-46`: define the #306 reservation token, owner, lifetime through provisioning/launch, atomic handoff, and failure/cancellation release; add `pair#306` to dependencies.

4. Minor findings

None.

5. Test coverage notes

- `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd -count=1` passed.
- `go test ./... -count=1` passed.
- Live `TestProvisionConformance` passed.
- Existing targeted/race/vet evidence is consistent with the implementation.

6. Architectural notes

- ARCH-DRY: pass.
- ARCH-PURE: pass; parsing/selection are separated from injected IO.
- ARCH-PURPOSE: pass for provisioning; #306 integration remains explicitly deferred.
- ARCH-MOCK: pass; stateful production-seam fixtures exist.
- ARCH-CONSTRAINTS: pass; subprocess, output, and diagnostic bounds are enforced.
- ARCH-SECURE: pass; strict records, canonical paths, and argv execution are used.
- ARCH-ORDER: pass; interruption and publication races are modeled and tested.
- ARCH-FUNERAL: pass; retained workspaces and bounded intent/marker artifacts have stated lifecycles.

7. Plan revision recommendations

Add the executable cross-issue reservation contract and declare `pair#306` as a dependency before closing. The prior fixture-location correction is addressed.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      The issue still lacks pair#306 dependency metadata and the plan still leaves reservation token, ownership, lifetime, handoff, and recovery undefined.
  - id: BR-2
    disposition: not-addressed
    note: |
      The plan still says #306 designs the reservation representation without an executable selection-to-provisioning exclusion contract.
  - id: BR-3
    disposition: addressed
    note: |
      The active plan now names provision_git_test.go for ProvisionFixture and no longer references the nonexistent provision_fake_test.go.
```
