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
