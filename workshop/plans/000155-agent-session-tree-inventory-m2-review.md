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
