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

---

## Re-review — 2026-08-26T16:43:44-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 149 — couch: opaque tags and a human naming layer |
| repo | pair |
| issue file | workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | cd7168cb4ac6023f6988b7198099c322a00ec74c..0d3bd4cb52c2fed65b2792c36e5dd01826fde7c7 |
| command | sdlc milestone-close --issue 149 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-08-26T16:43:44-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The six prior findings are addressed with reachable production paths and regressions, and the focused and full Go suites pass. Two Critical M3 contract gaps remain: named `couch show` output omits the durable tag explicitly required for diagnostics, and panel inventory/reference callbacks silently swallow authoritative ThreadStore failures.

```findings
dispose:
  - id: BR-18
    disposition: addressed
    note: |
      Production launch now distinguishes typed store absence from corruption, with corrupt-index and absent-store flow regressions.
  - id: BR-19
    disposition: addressed
    note: |
      Required arguments now validate map presence, and CLI tests pin empty name, description, and published-summary clearing.
  - id: BR-20
    disposition: addressed
    note: |
      CLI show, name, and describe derive Git-root repository scope; a repeated-tag regression covers reads and writes across scopes.
  - id: BR-21
    disposition: addressed
    note: |
      Initial attachment dispatches the typed attach operation, and exact pane registration is private to the console executor.
  - id: BR-22
    disposition: addressed
    note: |
      The audited Core concepts table names greppable PURE entities separately from INTEGRATION store and executor surfaces.
  - id: BR-23
    disposition: addressed
    note: |
      README now inventories M3 commands, scoped lookup, rendering, clearing, standalone resolution, and picker behavior.
findings:
  - id: new
    severity: Critical
    family: detail-view-preserves-durable-identity
    title: |
      Named couch show output drops the durable tag promised for diagnostics
    detail: |
      The Spec requires list to stop leading with the system id while retaining it for show and diagnostics, but run.go:432-433 sends list and show through the same renderer and run.go:454-480 emits only ThreadSummary.Label(), which replaces a named thread's tag completely. A named `couch show compiler` therefore cannot reveal the immutable address needed for exact follow-up operations. Preserve name-first list output while making show include the opaque tag, and add a named-show CLI regression that fails when the tag is absent (ARCH-PURPOSE).
  - id: new
    severity: Critical
    family: durable-index-read-failure-authority
    title: |
      Panel callbacks silently turn authoritative ThreadStore failures into empty results
    detail: |
      This is the 2nd finding in family `durable-index-read-failure-authority`. run.go:331-341 discards errors from both ResolveThreadReference and ThreadInventory, while console.go:190-218 and console.go:868-906 expose callbacks that cannot return an error. A corrupt or incomplete store can therefore replace the authoritative panel with an empty list or no matches without any notice. Do not patch only one closure: state the rule that every durable-record read either returns valid state or surfaces its failure, change the callback boundary accordingly, and add a production-wiring regression using a failing store read (ARCH-PURPOSE).
```

### Strengths

- Composite addresses remain intact through inventory, panel filtering, selection, and terminal target binding; same-path thread regressions exercise the distinction.
- Metadata transitions are pure and field-independent, while store updates use revision CAS.
- Operation declarations are closure-free and dispatch selects exactly one injected executor; missing owner capability cannot fall back.
- Standalone Pair reads the real file-backed ThreadStore projection, preserves opaque artifact tags, and distinguishes absent from corrupt durable state.
- Atlas and README coverage now describe the principal M3 surfaces.

### Critical findings

The two findings above must be fixed before rerunning the M3 boundary.

### Important findings

None.

### Minor findings

None.

### Test coverage notes

Fresh verification:

- Focused packages passed: `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/couchtty ./cmd/internal/launcher -count=1`
- Full suite passed on rerun: `go test ./... -count=1`
- `git diff --check cd7168c..0d3bd4c` passed.
- An initial full-suite run timed out in `TestOSRuntimeProbeLiveLayoutUsesSessionScopedPaneReport`; the test then passed three consecutive isolated runs, the launcher package passed, and the full rerun passed.
- Missing regressions correspond directly to the new findings: named `show` retaining its tag and panel behavior under a failing authoritative store read.

### Architectural notes for upcoming work

- **ARCH-DRY:** Pass. Matcher, inventory construction, and operation dispatch are shared rather than restated by clients.
- **ARCH-PURE:** Pass. Metadata transition, inventory construction, reference matching, and operation declarations have direct IO-free tests.
- **ARCH-PURPOSE:** Flag. Diagnostic identity and fail-closed panel reads are committed behavior that the current diff under-delivers.
- **ARCH-MOCK:** Pass. Production and tests share the store/runtime/executor seams, including a portable file-backed conformance fixture.

### Plan revision recommendations

Append a `## Revisions` entry recording:

- the rendering rule: list is name-first and may hide a named tag, while `show` must retain the immutable tag for diagnostics;
- the durable-read rule: panel inventory and resolver boundaries propagate ThreadStore failures visibly instead of converting them to empty state;
- the corresponding named-show and failing-store production-wiring regressions.

---

## Re-review — 2026-08-26T16:57:06-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 149 — couch: opaque tags and a human naming layer |
| repo | pair |
| issue file | workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | cd7168cb4ac6023f6988b7198099c322a00ec74c..e5e1d6f99d890a29549450d00b7627a72fe6abbc |
| command | sdlc milestone-close --issue 149 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-08-26T16:57:06-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-24 and BR-25 are addressed with reachable production paths and regressions, and all focused/full Go tests pass. One Critical fail-closed gap remains: standalone Pair’s duplicate ThreadStore decoder accepts records that Couch rejects as structurally invalid, allowing corrupt authoritative state to participate in name resolution and launch.

```findings
dispose:
  - id: BR-24
    disposition: addressed
    note: |
      Named show output now includes the immutable composite address, and the CLI regression fails if the opaque tag is removed.
  - id: BR-25
    disposition: addressed
    note: |
      Both production panel callbacks propagate ThreadStore failures, with real corrupt-store wiring tests and visible-error console regressions.
findings:
  - id: new
    severity: Critical
    family: durable-index-read-failure-authority
    title: |
      Standalone Pair accepts ThreadStore records that Couch rejects as invalid
    detail: |
      This is the 3rd finding in family `durable-index-read-failure-authority`. The shadow record schema in thread_index.go:54-63 omits required `starting_path` and `claim_generation`, and LoadThreadIndex at lines 99-110 consequently accepts the fixture at thread_index_test.go:49-58 even though Couch rejects that same record at thread.go:73-80 and threadstore.go:578-590. This permits malformed or incomplete authoritative records to resolve human names and launch opaque tags while Couch refuses the store. Do NOT patch only these two fields: state the rule that every durable-record reader shares the authoritative structural acceptance contract, enumerate all persisted invariants, and derive the portable projection from a common lower-layer decoder or enforce acceptance parity with conformance tests (ARCH-DRY, ARCH-PURPOSE).
```

### Strengths

- BR-24 cleanly separates compact list rendering from diagnostic show rendering at `cmd/internal/couchcmd/run.go:434-490`.
- BR-25 carries errors through both production callbacks and preserves the prior panel model at `cmd/internal/couchtty/console.go:896-934`.
- Composite addresses remain intact through inventory, filtering, selection, and terminal routing.
- Metadata transitions remain pure and independently field-preserving; storage updates use revision CAS.
- README and atlas cover the new M3 commands, identity semantics, standalone lookup, and panel failure behavior.

### Critical findings

See the machine-readable finding above. The portable reader must not accept a record the authoritative ThreadStore rejects.

### Important findings

None.

### Minor findings

None.

### Test coverage notes

Passed:

- `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/couchtty ./cmd/internal/launcher -count=1`
- `go test ./... -count=1`
- `git diff --check cd7168cb4ac6023f6988b7198099c322a00ec74c..e5e1d6f99d890a29549450d00b7627a72fe6abbc`

Add an acceptance-parity regression built from a valid real Couch record, then corrupt each required persisted invariant and assert both Couch and standalone Pair reject it.

### Architectural notes for upcoming work

- **ARCH-DRY:** Flag — `threadIndexRecord` is a drifting restatement of the authoritative persisted schema.
- **ARCH-PURE:** Pass — metadata, inventory, matching, and operation declarations have direct IO-free tests.
- **ARCH-PURPOSE:** Flag — the promised fail-closed portable projection accepts authoritative state Couch considers invalid.
- **ARCH-MOCK:** Pass — production and tests share file-backed/runtime seams; extend the conformance fixture to cover invalid-record parity.

The Core concepts table otherwise matches the implemented entities and purity boundaries. Atlas and README gates pass.

### Plan revision recommendations

Append a `## Revisions` entry recording:

- the invariant that all ThreadStore readers share one structural acceptance contract;
- the complete persisted-record invariant enumeration;
- the chosen common lower-layer decoder/projection boundary;
- the Couch-versus-launcher invalid-record conformance regression.

---

## Re-review — 2026-08-26T17:16:34-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 149 — couch: opaque tags and a human naming layer |
| repo | pair |
| issue file | workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | cd7168cb4ac6023f6988b7198099c322a00ec74c..31ecd40745f0ddb8e5f62a6cf7341be20a1abd46 |
| command | sdlc milestone-close --issue 149 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-08-26T17:16:34-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

BR-26 is addressed: Couch and standalone Pair now share one complete persisted-record decoder, and the regression demonstrably fails when Launcher’s former partial decoder is restored. No new findings block M3.

```findings
dispose:
  - id: BR-26
    disposition: addressed
    note: |
      Both readers use the complete shared persisted-record decoder, and the real-store mutation test fails when Launcher is reverted to its former partial schema.
```

### 1. Strengths

- The wire schema and validation are centralized in `threadrecord.Record` and `DecodePersisted` at `cmd/internal/threadrecord/record.go:57`.
- Couch reads through the shared decoder at `cmd/internal/couchcore/threadstore.go:579`; Launcher uses the same decoder at `cmd/internal/launcher/thread_index.go:97`.
- The parity test creates real Couch records and mutates persisted invariants before exercising both readers at `cmd/internal/launcher/thread_index_conformance_test.go:74`.
- Strict duplicate-key, unknown-field, and trailing-value rejection has one shared implementation at `cmd/internal/strictjson/decode.go:13`.
- README and atlas cover the M3 command, identity, picker, diagnostic, and failure semantics.

### 2. Critical findings

None.

### 3. Important findings

None.

### 4. Minor findings

None.

### 5. Test coverage notes

Passed:

- Focused package tests.
- `go test ./... -count=1`
- `go test -race ./cmd/internal/couchcore ./cmd/internal/launcher -count=1`
- `git diff --check cd7168cb4ac6023f6988b7198099c322a00ec74c..31ecd40745f0ddb8e5f62a6cf7341be20a1abd46`

BR-26 red/green verification was also performed in a scratch copy. Restoring Launcher’s old partial decoder made the conformance test fail across 19 mutation cases, including missing `starting_path`, missing/zero `claim_generation`, invalid nested start state, and unknown nested fields. The current implementation passes.

### 6. Architectural notes for upcoming work

- **ARCH-DRY — pass:** persisted schema, validation, and strict JSON decoding each have one authority.
- **ARCH-PURE — pass:** validation, metadata transitions, inventory construction, matching, and operation declarations have direct IO-free tests.
- **ARCH-PURPOSE — pass:** the shadow sweep confirms every ThreadStore writer and both readers derive from the common record contract.
- **ARCH-MOCK — pass:** integration coverage runs production readers against the portable file-backed store through the same seams used in production.

### 7. Plan revision recommendations

None. The Core concepts table, M3 checklist, and existing revision entries match the implementation.
