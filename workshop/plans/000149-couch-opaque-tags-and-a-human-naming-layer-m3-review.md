# Boundary Review — pair#149 (milestone M3)

| field | value |
|-------|-------|
| issue | 149 — couch: opaque tags and a human naming layer |
| repo | pair |
| issue file | workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | cd7168cb4ac6023f6988b7198099c322a00ec74c..2a8e0b05917d901138a28c9b95d7ef0524776a14 |
| command | sdlc milestone-close --issue 149 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-08-26T16:19:43-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M3 establishes strong composite-thread primitives, shared inventory, and a closure-free dispatcher, and all focused/full Go tests pass. However, five Critical contract gaps remain: corrupt ThreadIndex reads silently become legacy launches, empty metadata cannot be cleared through operations, CLI metadata lookup loses repository scope, initial console attachment bypasses the declared operation, and the plan’s Core concepts table contradicts the implementation.

```findings
findings:
  - id: new
    severity: Critical
    family: durable-index-read-failure-authority
    title: |
      Corrupt ThreadIndex errors silently fall back to launching the input as a legacy tag
    detail: |
      LoadThreadIndex explicitly fails closed on corrupt or incomplete state, but createflow.go:164-166 ignores that error and resolveResumeTag at createflow.go:852-862 returns the original argument for every read failure. Thus `pair resume compiler` can create or resume a direct `compiler` tag when the authoritative index is corrupt. Distinguish an absent store from malformed/incomplete state, surface the latter, and add a production-flow regression that fails without the error propagation (ARCH-PURPOSE).
  - id: new
    severity: Critical
    family: metadata-empty-value-contract
    title: |
      Required-argument validation makes documented metadata clearing unreachable
    detail: |
      validateOperationCall at operationdispatch.go:90-93 treats an empty value as omission. Consequently `couch name <ref> ""` and `couch publish-description ""` are rejected even though ThreadMetadataPatch promises explicit empty values clear those fields. Validate required argument presence rather than non-empty content where empty is meaningful, and pin both operation paths with red-without-fix tests.
  - id: new
    severity: Critical
    family: composite-address-collision-domain
    title: |
      CLI metadata operations cannot supply the repository scope needed to address repeated tags
    detail: |
      This is the 3rd finding in family `composite-address-collision-domain`. name/describe declare repo-scope as implicit at ops.go:140-155, but bindArgs excludes implicit fields and RunWithRuntime only populates scope/tag for publish-description at run.go:127-137. DirectStoreExecutor therefore resolves CLI name/describe with an empty scope at operationdispatch.go:124-138, making a repeated legacy tag globally ambiguous even when invoked from its repository. Do not patch only name: state and enforce the rule that every composite-address consumer either carries an exact address or derives the current repository scope, then sweep show/name/describe and future metadata clients. Add a CLI-level repeated-tag regression (ARCH-PURPOSE).
  - id: new
    severity: Critical
    family: operation-dispatch-single-authority
    title: |
      Initial console attachment bypasses the declared attach operation
    detail: |
      The plan requires every effectful human action to flow through DispatchOperation, and attach is declared specifically for a newly started terminal. Panel starts comply at console.go:1061-1069, but the initial `couch start` path calls AttachThreadActor directly at run.go:248-266. Route this path through the same typed attach executor and add a wiring test that fails if the declaration is bypassed (ARCH-DRY, ARCH-PURPOSE).
  - id: new
    severity: Critical
    family: core-concept-kind-contract
    title: |
      The Core concepts table no longer describes the M3 entities and purity boundary
    detail: |
      This is the 3rd finding in family `core-concept-kind-contract`. The plan names a nonexistent `ThreadMetadata` entity and classifies its mixed pure/store file as PURE at plan.md:47, while `Operation` remains classified INTEGRATION at line 48 after its declarations became pure and its executors moved to the integration table at line 81. Do not patch these two cells in isolation: state the rule that each row names a greppable entity and one architectural kind, audit the full table, and append a plan revision recording the corrected PURE metadata transition/declarations and INTEGRATION store/executor surfaces.
  - id: new
    severity: Important
    family: user-facing-policy-docs
    title: |
      README still documents the pre-M3 tree and actor interface
    detail: |
      This is the 2nd finding in family `user-facing-policy-docs`. README.md:267-272 still says list shows actors, show targets one tree, and name/describe mutate tree metadata; README.md:326 documents only resume-by-tag and omits human-name resolution and picker behavior. Do not fix only these lines: enumerate every M3 user-facing command, lookup, rendering, and clearing behavior and sweep the README against that inventory.
```

### Strengths

- Composite addresses are preserved through inventory, panel filtering, selection, and target binding; same-path threads have direct regressions.
- Metadata fields are independently modeled and revision-CAS updates reject stale writes.
- `DispatchOperation` correctly refuses missing live-owner capability without falling back to direct-store execution.
- The launcher projection reads the real ThreadStore format in an integration test, while picker tests preserve opaque selection tags behind human labels.
- Atlas coverage was updated for the new metadata, inventory, and standalone lookup surfaces.

### Critical findings

See the machine-readable block above. All five must be fixed before rerunning the M3 boundary.

### Important findings

The README requires a complete M3 public-surface sweep before close.

### Minor findings

None.

### Test coverage notes

Executed successfully:

- `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/couchtty ./cmd/internal/launcher -count=1`
- `go test ./... -count=1`
- `git diff --check <base>..<head>`

Missing regressions correspond directly to the Critical findings: corrupt-index production flow, empty name/summary clearing through dispatch, repeated-tag CLI scope, and initial attach dispatch wiring.

### Architectural notes for upcoming work

- **ARCH-DRY:** Flagged—the initial attach has a parallel execution path outside the declared operation.
- **ARCH-PURE:** Pass in implementation: the metadata transition, inventory construction, and reference matcher are directly unit-testable without IO. The plan table needs correction to reflect that boundary.
- **ARCH-PURPOSE:** Flagged—fail-closed lookup, composite scoping, clearing semantics, and universal operation dispatch are stated M3 purposes but are not fully reachable.
- **ARCH-MOCK:** Pass—the owned file-backed store has a portable integration fixture, and production/test flows share injected runtime and executor seams. No new external service call bypasses its seam.

### Plan revision recommendations

Append a `## Revisions` entry recording:

- the corrected PURE/INTEGRATION entity split for metadata and operation declarations/executors;
- the invariant that every composite-address client derives scope or carries the exact address;
- fail-closed handling for malformed/incomplete ThreadIndex state;
- attach as a universally dispatched effect;
- the actual Task 2 file placement (`threadinventory.go` rather than the planned `couch.go` changes).
