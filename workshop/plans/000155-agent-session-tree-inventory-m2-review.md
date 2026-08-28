# Boundary Review — pair#155 (milestone M2)

| field | value |
|-------|-------|
| issue | 155 — deterministic agent session-tree inventory |
| repo | pair |
| issue file | workshop/issues/000155-agent-session-tree-inventory.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 4f151b037dc2d0f20d413c5af6a2353e131cec8e..b6d5bdb6614c6095dfb092cfe6e1493f28f3023b |
| command | sdlc milestone-close --issue 155 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-08-28T15:49:52-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M2 establishes a strong pure-core/IO-shell foundation, removes first/newest watcher selection, and adds durable generation-aware ledger writes. It cannot close yet: restart can still recover from a stale config while the current typed launch is provisional, valid authored Markdown can disable correlation, unknown/unreadable evidence is silently discarded, and public ordering violates schema v1.

```findings
findings:
  - id: new
    severity: Critical
    family: established-binding-is-sole-recovery-authority
    title: |
      Plain restart can resume a stale config while the current typed launch is provisional
    detail: |
      ARCH-PURPOSE: cmd/internal/launcher/markers.go:87-90 falls back from the marker's established-ledger session ID to saved.SessionID. The M2 contract requires an intentionally fresh restart while the latest typed launch is provisional; a stale compatibility cache must never restore an older root. Remove the session-ID fallback while retaining saved arguments, and add a regression where a provisional typed launch coexists with stale config. The current marker test asserting that fallback must be reversed.

  - id: new
    severity: Critical
    family: diagnostic-registry-single-source
    title: |
      Unrecognized and unreadable evidence is still silently discarded
    detail: |
      This is the 2nd finding in family diagnostic-registry-single-source. Do not fix only one site: state and enforce the class rule that every unrecognized versioned shape and failed evidence read produces a registry-backed diagnostic. The current sweep finds cmd/internal/sessioninventory/event.go:316-332 silently ignoring an unknown Muse run-event kind, plus cmd/internal/sessioninventory/pair_inventory.go:74-90 silently continuing after unreadable Pair logs/configs. Add tests for each adapter default and each Pair-artifact read boundary.

  - id: new
    severity: Critical
    family: documented-total-order
    title: |
      Diagnostic ordering puts a null agent first despite schema v1 requiring null last
    detail: |
      This is the 2nd finding in family documented-total-order. cmd/internal/sessioninventory/order.go:168-178 compares Agent as its raw empty string, which sorts before named agents; the documented tuple says agent null last. State the general nullable-comparator rule, sweep every nullable component, and add an exhaustive comparator test rather than patching this field alone.

  - id: new
    severity: Critical
    family: authored-log-framing-round-trips
    title: |
      A valid Markdown horizontal rule inside authored text makes the entire round suffix unusable
    detail: |
      cmd/internal/sessioninventory/pairfacts.go:58-90 treats the first blank-line-delimited horizontal rule as the entry terminator. SessionLogStore permits the same bytes inside an authored body, so a prompt containing before, a Markdown horizontal rule, and after is persisted successfully but ParsePairLog rejects the remainder as a missing timestamp header. Live and offline correlation then remain provisional. Make framing round-trip valid authored Markdown and add a store-to-parser regression.

  - id: new
    severity: Important
    family: public-cli-readme
    title: |
      README documentation is missing for the new public session-inventory command
    detail: |
      The range adds pair session-inventory with user-facing flags and exit behavior, but README.md is unchanged. Document the command, its human/JSON and conformance modes, and the provisional versus established meaning.

  - id: new
    severity: Important
    family: cli-result-matrix-is-executable
    title: |
      Task 7 claims full result-matrix privacy goldens but tests cover only a subset
    detail: |
      cmd/internal/sessioninventory/runcli_test.go:13-82 exercises six cases and cmd/internal/sessioninventory/render_test.go:52-72 contains only empty-output goldens. The plan explicitly requires byte goldens for every normal/conformance result row, including partial scans, schema drift, privacy failure with zero stdout, and serialization failure. Add the missing executable cases before keeping Task 7 checked.
```

## Strengths

- `LedgerStore.AppendBindingIfCurrent` verifies launch currency under the append lock, preventing stale watcher generations from binding.
- Round qualification and binding resolution are pure and directly tested, including thresholds, causal progress, ambiguity intersection, and parent-only propagation.
- Pair-log and ledger writes have explicit lock/write/fsync boundaries with injected failure tests and subprocess concurrency coverage.
- The watcher uses the shared scanner runtime and treats process/open-file state as corroboration rather than selection.
- Atlas documentation covers the new lifecycle, CLI, and migration boundary.

## Critical findings

The four Critical findings above block M2 close. The stale-config fallback is especially central: the plan named `markers.go` and `markers_test.go` for modification, but neither changed in the pinned range, and their existing behavior contradicts the established-only authority rule.

## Important findings

README coverage and the claimed result-matrix test coverage must be completed before the boundary.

## Minor findings

None.

## Test coverage notes

- Required stat and name-status inspections succeeded for the pinned range.
- Focused M2 Go packages passed.
- `make test-lua`, `tests/pair-session-watch-test.sh`, and `git diff --check` passed.
- A clean archive passed the core M2 packages and `cmd/internal/couchcore`; Lua tests also passed.
- Clean-archive `dispatcher`/`pair-go` compilation was blocked by pre-existing untracked generated runtime-bundle assets; the base revision fails identically. The working checkout’s `cmd/pair-go` tests passed.
- The dirty checkout’s full suite also saw an unrelated Couch startup timeout and uncommitted boundary metadata in `workshop/projects/couch.md`; neither belongs to the pinned range.

## Architectural notes for upcoming work

- `ARCH-DRY`: Pass. Shared normalization, scanner facts, ledger storage, and watcher inventory replace substantial duplication.
- `ARCH-PURE`: Pass. Matching, ordering, binding, and propagation are pure; filesystem/process work remains at injected boundaries.
- `ARCH-PURPOSE`: Flag. The stale-config fallback and non-round-tripping Pair log under-deliver the stated established-only recovery purpose.
- `ARCH-MOCK`: Pass. Stateful runtime fakes, filesystem-backed store tests, subprocess concurrency, and live conformance use the production seams.

## Plan revision recommendations

Append these `## Revisions` entries:

- “M2 boundary review — established authority closure”: include `markers.go`/`markers_test.go`, prohibit config session-ID fallback when the typed generation is provisional, and add the stale-cache regression.
- “M2 boundary review — evidence diagnostics and framing”: enumerate every native-adapter default and Pair-artifact read boundary; require diagnostic emission and arbitrary authored-Markdown round trips.
- “M2 boundary review — executable CLI contract”: enumerate every result-matrix row, add byte/privacy goldens, and include README.md in Task 7’s file list.

---

## Re-review — 2026-08-28T16:02:09-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 155 — deterministic agent session-tree inventory |
| repo | pair |
| issue file | workshop/issues/000155-agent-session-tree-inventory.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 528a730bd93a432071b274902fefd23096c7faee..528a730bd93a432071b274902fefd23096c7faee |
| command | sdlc milestone-close --issue 155 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-08-28T16:02:09-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The focused implementation is largely sound, and BR-8 through BR-11 have reachable regressions. The boundary cannot ship: the pinned review window is empty, a repository contract test fails because M2 has premature closed metadata, BR-12 lacks the required regression, and BR-13 still lacks byte goldens for the complete CLI result matrix.

1. Strengths

- Restart recovery now treats an empty marker as provisional and drops stale config authority while retaining arguments (`cmd/internal/launcher/markers.go:70`).
- Unknown Muse events reach the shared diagnostic path (`cmd/internal/sessioninventory/event.go:316`, `events.go:64`).
- Diagnostic nullable coordinates consistently use null-last projections, with exhaustive equal-prefix tests (`order.go:168`, `order_test.go:57`).
- Versioned byte-counted Pair-log framing round-trips authored horizontal rules through the real store/parser seam (`pairfacts.go:62`, `pairlog/store_test.go:39`).
- Core Concepts declarations match the plan through a bidirectional executable contract (`concept_contract_test.go:31`).

2. Critical findings

- The review range is `528a730..528a730`; both required stat and name-status commands return no changes. Therefore no implementation or claimed-fix delta is reviewable within the authoritative pinned window. Re-run with the previous accepted boundary/finding-round SHA as base and `528a730` as head.
- `go test ./... -count=1` fails at `TestUncheckedProjectMilestoneHasNoClosedMetadata`: `workshop/projects/couch.md:173` leaves pair#155 M2 unchecked while lines 399–400 already contain actual/closed metadata. This is the second finding in family `repository-contracts-stay-green`; establish the general rule that issue, plan, and project milestone closure state changes only in the successful close-gate transaction.

3. Important findings

- BR-12 remains open under the claimed-fix rule. README prose exists at `README.md:376-392`, but no test fails when that public-command documentation is removed.
- BR-13 remains open. Only three golden files exist, while partial scans, schema drift, privacy failure, serialization failure, and most other CLI matrix rows use substring or inline assertions (`runcli_test.go:15`, `runcli_failure_test.go:9`). The promised complete byte-golden matrix is not delivered.

4. Minor findings

None.

5. Test coverage notes

- Passed: focused `sessioninventory`, `pairlog`, and `launcher` packages.
- Passed: `make test-lua` and `tests/pair-session-watch-test.sh`.
- Failed: repository-wide Go tests due to the premature M2 project metadata.
- Additional `cmd/pair-go` failures came from sandbox denial of `/bin/ps`; they are not treated as product findings here.
- BR-8 through BR-11 each have regressions that would fail against the preceding implementation. BR-12 and BR-13 do not satisfy the stated proof standard.

6. Architectural notes for upcoming work

- `ARCH-DRY`: pass—shared framing, nullable comparators, diagnostics, and runtime seams are reused.
- `ARCH-PURE`: pass—matching, ordering, framing, and rendering remain pure behind thin injected IO boundaries.
- `ARCH-PURPOSE`: flag—the empty review range and incomplete CLI golden matrix under-deliver the boundary’s verification purpose.
- `ARCH-MOCK`: pass—the stateful fake and production inventory entry point share the same boundary; live conformance remains represented.

7. Plan revision recommendations

Append a revision recording that Task 7 is not complete until every normal/conformance result row has checked-in exact stdout/stderr bytes, including partial, drift, privacy-zero-stdout, serialization, and writer failures. Also record that project actual/closed metadata must be written only after the milestone-close transaction succeeds.

```findings
dispose:
  - id: BR-8
    disposition: addressed
    note: |
      Empty-marker restart now drops stale config authority, and the regression fails against the former saved-session fallback.
  - id: BR-9
    disposition: addressed
    note: |
      Unknown Muse run events become near-misses and all three Pair artifact read boundaries emit registry-backed storage_unreadable diagnostics with executable coverage.
  - id: BR-10
    disposition: addressed
    note: |
      Agent, native ID, path, and source-ref nullability now use null-last projections with exhaustive equal-prefix comparisons.
  - id: BR-11
    disposition: addressed
    note: |
      Byte-counted framing round-trips authored horizontal rules through SessionLogStore and ParsePairLog.
  - id: BR-12
    disposition: not-addressed
    note: |
      README prose exists, but no regression fails when the new public command documentation is removed.
  - id: BR-13
    disposition: not-addressed
    note: |
      The CLI branches are exercised, but complete byte goldens still do not exist for the promised normal and conformance result matrix.
findings:
  - id: new
    severity: Critical
    family: boundary-review-window-captures-delta
    title: |
      The authoritative review window contains no changes
    detail: |
      Base and head are both 528a730, and the required stat and name-status recipes are empty. Re-run with a base preceding the M2 implementation or finding-closure commits so the anti-collusion review can inspect the actual delta.
  - id: new
    severity: Critical
    family: repository-contracts-stay-green
    title: |
      M2 carries closed project metadata before the milestone is checked
    detail: |
      workshop/projects/couch.md leaves pair#155 M2 unchecked at line 173 but records actual and closed values at lines 399-400, causing TestUncheckedProjectMilestoneHasNoClosedMetadata to fail. This is the 2nd finding in family repository-contracts-stay-green. Do not fix only this instance: enforce that issue, plan, and project closure metadata changes only through the successful close-gate transaction; the current sweep measured one violation.
```

---

## Re-review — 2026-08-28T16:13:29-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 155 — deterministic agent session-tree inventory |
| repo | pair |
| issue file | workshop/issues/000155-agent-session-tree-inventory.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | cac75f3a5f86eb76600572d18bf1d8f6c09fea8f..dec526e5f5f888fe1245fdf64183437011b66af7 |
| command | sdlc milestone-close --issue 155 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-08-28T16:13:29-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The prior README, CLI-matrix, and project-state findings are resolved and their focused tests pass. However, the authoritative range still excludes the entire M2 implementation: it contains only one project/preflight documentation commit. The boundary therefore cannot verify M2’s code, tests, pure/IO separation, or stateful fake, so BR-14 remains blocking.

1. Strengths

- [README.md](/Users/xianxu/workspace/pair/README.md:376) clearly documents the command, flags, exit behavior, privacy, and provisional/established/ambiguous meanings.
- [runcli_test.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/runcli_test.go:16) and [runcli_failure_test.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/runcli_failure_test.go:12) now exercise normal, partial, conformance, usage, fatal, privacy, serialization, and writer outcomes against checked-in byte matrices.
- [project_state_contract_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/project_state_contract_test.go:18) catches the exact unchecked-plus-closed-metadata inconsistency; reverting the M2 checkbox at [couch.md](/Users/xianxu/workspace/pair/workshop/projects/couch.md:173) would make it fail.
- The plan records the corrected preflight interpretation through an appended revision rather than rewriting history.

2. Critical findings

- **BR-14 remains open — `boundary-review-window-captures-delta`.** The required stat/name-status inspection shows only `workshop/lessons.md`, the plan, and `workshop/projects/couch.md`. All M2 implementation commits (`8d8b074` through `cac75f3`) precede the supplied base. This is the second finding in this family. Do not adjust another isolated base: enforce that an M2 boundary window begins at the previous successful boundary, `4f151b037dc2d0f20d413c5af6a2353e131cec8e`, and ends at the proposed M2 head. Re-run the review over `4f151b037dc2d0f20d413c5af6a2353e131cec8e..dec526e5f5f888fe1245fdf64183437011b66af7`.

3. Important findings

None beyond the blocking inspection failure.

4. Minor findings

None.

5. Test coverage notes

Passed:

- `go test ./cmd/internal/sessioninventory -count=1`
- `go test ./cmd/internal/couchcore -run 'TestUncheckedProjectMilestoneHasNoClosedMetadata|TestProjectMilestoneStateProblemsRejectsPrematureClose' -count=1`
- `go test ./cmd/pair-go -count=1`
- `git diff --check cac75f3..dec526e`

These validate the prior-finding fixes, but cannot substitute for reviewing the omitted M2 implementation range.

6. Architectural notes for upcoming work

- **ARCH-DRY:** Pass for the inspected delta; it introduces no duplicated implementation logic.
- **ARCH-PURE:** Flagged as unassessable under BR-14 because the pure entities and IO seams are outside the range.
- **ARCH-PURPOSE:** Flagged under BR-14; a boundary review that omits the implementation cannot demonstrate the milestone’s stated purpose.
- **ARCH-MOCK:** Flagged as unassessable under BR-14; the shared runtime seam, stateful fake, and conformance flow are outside the range.

7. Plan revision recommendations

None. Correct the review-window construction and rerun; no additional plan revision is required merely for that rerun.

```findings
dispose:
  - id: BR-12
    disposition: addressed
    note: |
      README lines 376-392 document the public command, flags, exit behavior, privacy, and binding states; TestREADMEDocumentsSessionInventoryContract pins the required surface.
  - id: BR-13
    disposition: addressed
    note: |
      The result and failure matrix tests now compare exact exit, stdout, and stderr bytes for normal, partial, conformance, usage, fatal, privacy, serialization, and writer branches against checked-in goldens.
  - id: BR-14
    disposition: not-addressed
    note: |
      The range is nonempty but still excludes every M2 implementation and test commit; rerun from the previous M1 boundary 4f151b037dc2d0f20d413c5af6a2353e131cec8e.
  - id: BR-15
    disposition: addressed
    note: |
      The Couch checkbox and closure detail now move together, and TestUncheckedProjectMilestoneHasNoClosedMetadata passes while directly rejecting the prior unchecked-plus-closed state.
```

---

## Re-review — 2026-08-28T16:24:46-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 155 — deterministic agent session-tree inventory |
| repo | pair |
| issue file | workshop/issues/000155-agent-session-tree-inventory.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 4f151b037dc2d0f20d413c5af6a2353e131cec8e..dec526e5f5f888fe1245fdf64183437011b66af7 |
| command | sdlc milestone-close --issue 155 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-08-28T16:24:46-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M2 has a strong pure-core/injected-IO design, and the pinned review range is now valid. It cannot close yet: three prior findings remain incompletely addressed, optional process corroboration can suppress valid establishment, and the public JSON differs from the documented schema-v1 contract.

```findings
dispose:
  - id: BR-8
    disposition: not-addressed
    note: |
      The launcher marker instance is fixed and tested, but sessioninventory still selects config evidence when LaunchPresent is true and no round or binding exists, re-establishing stale compatibility state during a provisional typed launch.
  - id: BR-9
    disposition: not-addressed
    note: |
      The adapter and CLI cases are tested, but watcher Pair-log read failures, owner-mismatched typed ledger rows, and scanner-unknown ledger/config native IDs are still silently discarded.
  - id: BR-10
    disposition: addressed
    note: |
      Nullable diagnostic coordinates use the shared null-last projection, with exhaustive equal-prefix comparator coverage.
  - id: BR-11
    disposition: not-addressed
    note: |
      The Go store/parser regression proves byte-counted framing, but Neovim history parsing and rewriting still use delimiter framing and cannot consume new entries or authored separator text.
  - id: BR-12
    disposition: addressed
    note: |
      README documents the command, modes, lifecycle statuses, and exits, with an executable documentation regression.
  - id: BR-13
    disposition: addressed
    note: |
      Checked-in result and failure matrix goldens now pin exact exit, stdout, and stderr bytes across the claimed CLI branches.
  - id: BR-14
    disposition: addressed
    note: |
      The authoritative range is non-empty and contains 94 changed files with 6,422 additions and 1,668 deletions.
  - id: BR-15
    disposition: addressed
    note: |
      Couch now carries the M2 checkbox, actual, date, and detail as one consistent gate-preflight state.
findings:
  - id: new
    severity: Critical
    family: usable-process-evidence-only
    title: |
      Unrelated open files suppress portable causal-round establishment
    detail: |
      ARCH-PURPOSE: cmd/internal/sessionwatch/run.go reports process evidence available whenever the Pair process tree has any open file, even when none maps to a scanner-authorized native artifact. Run then filters every otherwise unique causal round. Return available=true only for relevant native-artifact evidence, and add a stateful-fake regression with unrelated open files plus one globally unique completed round.
  - id: new
    severity: Critical
    family: schema-v1-output-is-exact
    title: |
      Public evidence JSON does not match the documented schema-v1 contract
    detail: |
      ARCH-PURPOSE: the internal Evidence type adds public root_node_id, and render.go copies it directly into the public DTO. Ledger/config evidence also renders required position and fingerprint arrays as null. Introduce an exact evidenceV1 projection with non-nil arrays and test it against an independent contract golden.
```

1. Strengths

- `LedgerStore.AppendBindingIfCurrent` verifies launch currency while holding the append lock.
- Matching, ordering, binding intersection, and parent propagation remain pure and directly tested.
- The Go framing regression covers authored horizontal rules through the real store/parser seam.
- README and atlas changes cover the new CLI, establishment lifecycle, ambiguity, and recovery surface.
- All M2 Core Concepts entries exist at their stated paths. PURE entities remain free of IO; INTEGRATION entities use injected seams.

2. Critical findings

- BR-8: [binding.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/binding.go:257) selects rank-4 config even for a current provisional launch. [pair_inventory.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/pair_inventory.go:131) supplies that stale config. A typed launch must suppress config authority until a binding is durably joined.
- BR-9: [run.go](/Users/xianxu/workspace/pair/cmd/internal/sessionwatch/run.go:90) discards Pair-log read errors. [pair_inventory.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/pair_inventory.go:67) silently drops owner-mismatched records, while unknown native IDs are erased before a diagnostic can be emitted.
- BR-11: [init.lua](/Users/xianxu/workspace/pair/nvim/init.lua:520) still splits history on `\n\n---\n\n` and recognizes only legacy headers. New entries expose the byte marker as text; edits can no-op, and authored separators create false entries.
- [run.go](/Users/xianxu/workspace/pair/cmd/internal/sessionwatch/run.go:170) treats unrelated open files as usable corroboration and filters valid exact rounds.
- [binding.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/binding.go:38) and [render.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/render.go:105) expose an undocumented schema field and null arrays.

3. Important findings

None.

4. Minor findings

None.

5. Test coverage notes

- Required stat, name-status, and patch inspections succeeded.
- Focused inventory, ledger, watcher, launcher, pair-log, dispatcher, `pair-go`, and contract tests passed.
- `make test-lua`, `tests/pair-session-watch-test.sh`, and `git diff --check` passed.
- An initial repository-wide run hit a Couch composition timeout. Clean HEAD and base archives both passed that focused test, as did a workspace rerun, so it is not reported as a boundary regression.
- Missing regressions cover provisional launch plus stale config, watcher log-read failure, unrelated open files, Lua v1 framing, and an independent schema oracle.

6. Architectural notes

- `ARCH-DRY`: Flag — Go owns byte-counted framing while Neovim retains a parallel legacy grammar.
- `ARCH-PURE`: Pass — core matching, ordering, resolution, and projection are deterministic and IO-free.
- `ARCH-PURPOSE`: Flag — stale config recovery, unusable process evidence, legacy Lua framing, and schema drift under-deliver the stated contracts.
- `ARCH-MOCK`: Pass structurally, but the missing stateful-fake cases above must be added.

7. Plan revision recommendations

Append revisions recording:

- One recovery authority: a current typed launch suppresses compatibility config until a binding joins it.
- Total Pair-evidence handling: every read, identity rejection, and unknown native ID produces a diagnostic.
- One framing grammar across Go and Neovim, including byte-count updates and authored separators.
- Usable process corroboration: unrelated open files do not disable portable exact-round establishment.
- An independent, exact schema-v1 DTO and non-empty contract golden.

---

## Re-review — 2026-08-28T16:44:34-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 155 — deterministic agent session-tree inventory |
| repo | pair |
| issue file | workshop/issues/000155-agent-session-tree-inventory.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 4f151b037dc2d0f20d413c5af6a2353e131cec8e..a749c0dde1a57890fa78267441e8d9936dfe2686 |
| command | sdlc milestone-close --issue 155 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-08-28T16:44:34-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The five repairs are reachable and regression-pinned, but BR-9 remains open: valid v1 ledger evidence with an unsupported agent is silently discarded. Because this is the third occurrence of `diagnostic-registry-single-source`, the boundary cannot close until the class rule is enforced across all Pair-evidence rejection paths.

1. Strengths

- BR-8, BR-11, BR-16, and BR-17 have effective regressions; reverting each fix in a scratch copy made its test fail.
- Byte-counted Pair-log framing round-trips authored Markdown through Go persistence/parsing and Lua history editing.
- The public `evidenceV1` DTO excludes internal `root_node_id` and emits non-null arrays.
- README and `atlas/session-identity.md` document the public CLI and round-gated lifecycle.
- Focused Go packages, Lua tests, watcher/queue shell tests, and `git diff --check` pass.

2. Critical findings

- **BR-9 not addressed — `cmd/internal/sessioninventory/pair_inventory.go:67-70`.** A syntactically valid v1 ledger row whose `agent` is unsupported reaches `ParseLedger`, then `RecoverPairBindings` silently continues because it is absent from the requested-agent map. A scratch regression expecting any registry-backed diagnostic fails with zero diagnostics.

  **This is the 3rd finding in family `diagnostic-registry-single-source`.** Do not patch only this branch. State and enforce the class rule: supported-but-unrequested evidence may be filtered; unsupported versioned values and recognized Pair evidence rejected for invalid ownership/path/shape must emit a registry-backed diagnostic. Sweep all silent rejection branches, including recognized sidecars rejected by `selectedPairArtifact`, and add end-to-end diagnostics tests.

3. Important findings

None.

4. Minor findings

None.

5. Test coverage notes

- Fresh focused Go verification passed all M2-related packages.
- `make test-lua`, `tests/pair-session-watch-test.sh`, and `tests/queue-send-test.sh` passed.
- Mutation checks went red for BR-8, BR-9’s named Muse/read cases, BR-11, BR-16, and BR-17.
- One full-suite attempt encountered sandbox denial executing `/bin/ps`; the affected `cmd/pair-go` package subsequently passed fresh with `-count=1`.

6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass—shared normalization, ledger store, Pair-log framing, and inventory projections replace parallel implementations.
- `ARCH-PURE`: Pass—matching, ordering, parsing, and binding remain pure behind narrow runtime/store seams.
- `ARCH-PURPOSE`: Flag—BR-9 fixed named instances but not the enumerable diagnostic class.
- `ARCH-MOCK`: Pass—stateful portable runtimes and injected persistence/process seams exercise production boundaries.

7. Plan revision recommendations

Append a `## Revisions` entry recording that the diagnostic closure claim was incomplete, defining the supported/unrequested/unsupported evidence rule, enumerating every Pair-evidence rejection branch, and requiring regressions for the full sweep.

```findings
dispose:
  - id: BR-8
    disposition: addressed
    note: |
      Provisional plain restart now drops stale config identity, and reverting the behavior makes TestPlanRestart fail.
  - id: BR-9
    disposition: not-addressed
    note: |
      Named Muse and read-boundary cases are fixed, but valid v1 ledger rows with unsupported agents are still silently discarded at pair_inventory.go:69-70.
  - id: BR-11
    disposition: addressed
    note: |
      Versioned byte-count framing round-trips arbitrary authored Markdown; reverting the writer to legacy framing makes the store-to-parser regression fail.
  - id: BR-16
    disposition: addressed
    note: |
      Only scanner-authorized open artifacts activate process corroboration; reverting availability to any open file makes the regression hang instead of establishing the unique round.
  - id: BR-17
    disposition: addressed
    note: |
      The exact public evidence projection removes internal root_node_id and forces arrays; reintroducing the field makes the independent golden fail.
```
