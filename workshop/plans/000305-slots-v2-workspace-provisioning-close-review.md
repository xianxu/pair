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
