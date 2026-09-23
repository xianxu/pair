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

---

## Re-review — 2026-09-23T13:05:43-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 305 — Slots v2: provision durable numbered workspaces |
| repo | pair |
| issue file | workshop/issues/000305-slots-v2-workspace-provisioning.md |
| boundary | whole-issue close |
| milestone | — |
| window | 570f8566a62c73c52edd5ca1b43befbaef028eec..835e6a03661d1f649a9cdc1d2f796f6073181d2f |
| command | sdlc close --issue 305 |
| reviewer | claude |
| timestamp | 2026-09-23T13:05:43-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

I recommend SHIP. The #305 range delivers what the Spec and Plan promise: one repeatable readiness operation (`WorkspaceProvisioner.Ensure`) that is reached only through `DirectStoreExecutor`. It captures the new host's starting commit from the `git fetch --porcelain` output itself, not from a later read of the tracking ref. It creates the environment directory exclusively and records the directory's identity. It creates the `main-slotN` branch only if absent and marks it with a reflog token. Recovery re-derives state from Git. A setup-success marker lives in the worktree's Git admin directory and is written under the per-repo creation lock. The focused provisioning tests pass locally: couchcore, `TestProvisionCLI*` and artifactpath. The other couchcmd failures are sandbox PTY/permission denials (`operation not permitted`), not regressions. On the open findings: #306's issue already declares `deps: [pair#305]`, and the plan now places the reservation token, its lifetime and its release inside #306's thread lifecycle. #305 only grants directory readiness and never creates a thread (`TestProvisionCLI` checks for no runner or supervisor effects). A reverse #305→#306 dependency would create a cycle, so I withdraw BR-1/BR-2. Nothing blocks SHIP; the remaining notes are Minor.

1. **Strengths**
   - `provision.go` `ensureHost`: taking ownership needs positive evidence at every step. The directory must match its recorded dev/inode, and the branch must match both its OID and the token-bearing reflog entry. An existing directory or branch with no evidence is refused, never taken over.
   - `provision_lock_unix.go`: the creation lock is released only by closing it (no `LOCK_UN`). Git children inherit it through `ExtraFiles`, so if the caller dies, the lock stays held until Git finishes.
   - `provision_io.go`: each subprocess is bounded. It runs in its own process group, gets SIGTERM then SIGKILL after 2s, has a 1s `WaitDelay`, a 1 MiB stdout cap and a 64 KiB diagnostic tail.
   - `provision_store.go`: stored records are read without following symlinks, capped at 16 KiB and decoded strictly. The creation intent is validated field by field, so a hand-edited or old record is refused instead of trusted (ARCH-SECURE).
   - The recovery tests run real Git repositories through the production command seam, with a stateful fake for SDLC and Weave. They cover a lost success write, cancellation after setup, interrupted upstream config, concurrent Weave, and a marker published by a competing caller.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `provision.go:144-200` (ARCH-ORDER): production never produces `HostConflict` or `SetupConflict`, because `readSuccess` and `verifyHost` errors return before `NextHostAction` runs. The decision table's refuse rows are only exercised by unit tests. Either feed conflicting observations into the table, or document that errors short-circuit before it.
   - `provision.go:92-95` (ARCH-CONSTRAINTS): after a Weave run of up to 20 minutes, the creation lock is re-taken without waiting. A brief lock held by another slot's creation then throws away a completed setup as "unconfirmed". This matches the plan, but a short bounded wait would avoid an unnecessary recompile.
   - `provision_select.go:36`: one partial or unknown candidate anywhere refuses all selection. That is stricter than the Spec's "not free space"; #306 should confirm it wants this.
   - `provision_conformance_test.go:18` (ARCH-MOCK): the live SDLC/Weave conformance check only runs when `PAIR_LIVE_WORKSPACE=1` is set, and no schedule is named.

5. **Test coverage notes:** the Done-when items are covered:
   - remote-main baseline with local HEAD ahead
   - hyphenated repo names and paths with spaces
   - non-origin remotes
   - foreign path and branch refused
   - lost branch acknowledgment
   - interrupted config
   - repeated calls
   - races

   Fixture concurrency is claimed to pass under `-race`; I did not re-run it with `-race`.

6. **Architectural notes**
   - The reservation obligation ("reserve before provisioning, hold through launch, release on failure; a competing launch refuses while it is held") is written only in #305's plan. #306's issue text doesn't state it; its line 23 says only "revalidating concurrent requests". Copy the obligation into #306's Spec when #306 starts planning.
   - `SelectWorkspaceNumber` is an allowlisted dead symbol until #306 wires it in; the allowlist entry names that removal condition.
   - Per-principle results:
     - ARCH-DRY: pass. It reuses `strictjson` and the atomic writer (`writeAtomicBytesWithPattern`).
     - ARCH-PURE: pass. Request, identity, host decisions and selection are pure; the IO sits behind the `ProvisionIO` seam.
     - ARCH-PURPOSE: pass.
     - ARCH-MOCK: minor note above.
     - ARCH-CONSTRAINTS: bounded; minor note above.
     - ARCH-SECURE: pass.
     - ARCH-ORDER: minor note above.
     - ARCH-FUNERAL: pass. The intent file is removed on success, the marker goes away with `git worktree remove`, there is one permanent lock per repo, and stale temp files are cleaned under the lock.

7. **Plan revisions:** none needed; the plan's entity tables match the delivered files.

```findings
dispose:
  - id: BR-1
    disposition: withdrawn
    note: |
      Overtaken by the design: #305 has no thread effects (TestProvisionCLI checks for no runner or supervisor effects). Plan lines 37-47 place the reservation token, lifetime and release in #306, and #306 already declares deps pair#305, so a reverse edge would be a cycle.
  - id: BR-2
    disposition: withdrawn
    note: |
      Same as BR-1. The class rule is stated in the plan: a readiness operation grants directory readiness, not thread capacity; the capability owner (#306) defines its own reservation. No #305 invariant is violated.
findings:
  - id: new
    severity: Minor
    family: decision-table-bypassed-by-early-error
    title: |
      NextHostAction refuse rows (HostConflict/SetupConflict) are never produced by production Ensure
    detail: |
      readSuccess and verifyHost errors return before the table is consulted, so its conflict rows are only exercised by unit tests. Route conflicting observations through the table or document the short-circuit.
  - id: new
    severity: Minor
    family: nonblocking-lease-after-long-work
    title: |
      A busy lock after a long Weave run discards a completed setup as unconfirmed
    detail: |
      Taking the creation lock without waiting after up to 20 minutes of setup turns a brief lock held by another slot's creation into a forced recompile. A short bounded wait would keep the design.
  - id: new
    severity: Minor
    family: live-conformance-cadence
    title: |
      The live SDLC/Weave conformance check only runs when opted in, with no schedule
    detail: |
      TestProvisionConformance runs only with PAIR_LIVE_WORKSPACE=1; ARCH-MOCK asks for a named schedule for live drift checks.
```
