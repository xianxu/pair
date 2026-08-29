# Boundary Review — pair#155 (whole-issue close)

| field | value |
|-------|-------|
| issue | 155 — deterministic agent session-tree inventory |
| repo | pair |
| issue file | workshop/issues/000155-agent-session-tree-inventory.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4c454436038e2ae049690bc343def9f0511fca8c..cc50108e52130f0df4539f19d8daa937e057ac21 |
| command | sdlc close --issue 155 |
| reviewer | codex |
| timestamp | 2026-08-28T17:49:56-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The deterministic model, injected runtime, stateful fake, consumer sweep, documentation, and broad verification are strong. However, three correctness gaps block close: unavailable process identity suppresses the portable causal matcher, valid transcripts over 32 MiB cannot establish or serve consumers, and partial Pair-artifact listings discard usable evidence. Mixed legacy/typed ledgers also produce false error diagnostics during normal launches.

```findings
findings:
  - id: new
    severity: Critical
    family: usable-process-evidence-only
    title: |
      Missing process identity prevents portable causal-round establishment
    detail: |
      This is the 2nd finding in family `usable-process-evidence-only`. A current PID file plus an unavailable identity token returns before causal matching. State the class rule: process evidence may constrain a match only when usable; its absence must never suppress a unique completed round.
  - id: new
    severity: Critical
    family: bounded-record-streaming
    title: |
      Whole-file limits make valid long transcripts unusable
    detail: |
      Native events, token usage, and transcript consumers read the entire artifact through a 32 MiB cap although the contract bounds individual JSONL records. Four installed valid transcripts already exceed that cap, so the affected roots cannot establish or serve migrated consumers.
  - id: new
    severity: Critical
    family: storage-boundary-regular-partial
    title: |
      Partial Pair-artifact enumeration discards valid evidence
    detail: |
      This is the 2nd finding in family `storage-boundary-regular-partial`. `RecoverPairBindings` treats every non-absence listing error as fatal even when `ListFiles` returned valid ledger/config/log entries. Apply the class rule to every storage root: retain regular partial results and diagnose rejected entries.
  - id: new
    severity: Important
    family: mixed-ledger-formats-are-classified
    title: |
      Valid compatibility ledger rows are reported as malformed
    detail: |
      Every launch still appends a legacy `LedgerEntry` before its typed launch row, but the inventory parser classifies every non-typed row as malformed. Mixed ledgers need one shared classification that distinguishes supported compatibility rows, typed authority, and genuinely corrupt rows.
```

1. Strengths

- The forest and ordering core is deterministic, deeply cloned, and tested with permutation/fuzz coverage.
- The Core Concepts contract checks plan declarations bidirectionally against source markers.
- `LedgerStore` serializes cross-process append, retains physical ordinals, fsyncs, and checks stale launches under the same lock.
- The shadow sweep removes independent native-session selection from governed consumers.
- README and six relevant atlas surfaces were updated in the review range.

2. Critical findings

- [cmd/internal/sessionwatch/run.go:66](/Users/xianxu/workspace/pair/cmd/internal/sessionwatch/run.go:66): a present PID with `ProcessIdentity(pid) == ""` exits immediately. Continue without process corroboration when identity is unavailable; retain fail-closed identity-change handling only after a usable token was captured. Add a regression where a unique completed round binds despite an empty identity token.

- [cmd/internal/sessioninventory/events.go:10](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/events.go:10): `NativeEventsWithRuntime` loads the whole transcript with a 32 MiB cap. [query.go:75](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/query.go:75) applies the same cap to slug/context/token consumers. Stream via `ReadAt`, enforcing the specified 8 MiB per-record bound. Tests must place valid events and usage after byte 32 MiB and fail under the current implementation.

- [cmd/internal/sessioninventory/pair_inventory.go:30](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/pair_inventory.go:30): valid `files` are discarded whenever listing also returns `ListingIssuesError`. Preserve returned regular files and attach coded diagnostics, mirroring native scanner behavior. Test a valid typed ledger beside a rejected symlink/special entry.

3. Important findings

- [cmd/internal/launcher/createflow.go:489](/Users/xianxu/workspace/pair/cmd/internal/launcher/createflow.go:489) writes a compatibility row through [launcher/osruntime.go:570](/Users/xianxu/workspace/pair/cmd/internal/launcher/osruntime.go:570), while [sessionledger/record.go:103](/Users/xianxu/workspace/pair/cmd/internal/sessionledger/record.go:103) classifies it as malformed and [pair_inventory.go:66](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/pair_inventory.go:66) emits an error diagnostic. Consolidate mixed-ledger classification and test legacy + typed + corrupt rows together.

4. Minor findings

None.

5. Test coverage notes

Passed:

- `git diff --check <base> <head>`
- `go test ./... -count=1`
- `make test-lua`
- Watcher, review, changelog, terminal shortcut shell suites
- Zellij configuration check
- Live redacted conformance for all four agents

Missing regressions correspond exactly to the four findings. The live conformance probe does not exercise causal-event reading of long transcripts; it passed even though three installed Codex transcripts and one Claude transcript exceed the event-reader cap.

6. Architectural notes for upcoming work

- `ARCH-DRY`: Flagged for separate launcher and inventory interpretations of the same mixed ledger.
- `ARCH-PURE`: The pure forest, matching, ordering, and binding cores pass. The transcript integration shell should stream bounded records into that core instead of materializing whole files.
- `ARCH-PURPOSE`: Flagged because optional process evidence, complete supported transcripts, and partial-result behavior do not yet fulfill the final round-gated contract.
- `ARCH-MOCK`: Pass. Production and tests share an injected stateful runtime; extend its existing error/state facilities with the missing partial-listing, empty-identity, and long-stream cases.

7. Plan revision recommendations

Append this to the plan’s `## Revisions`:

> `### 2026-08-28 — close review: portable evidence and bounded mixed storage`
>
> **Reason:** close review found that unusable process identity suppressed portable matching, whole-file caps rejected valid long JSONL histories, partial Pair listings discarded valid facts, and compatibility ledger rows were misclassified.
>
> **Delta:** make process corroboration conditional on a usable identity token; stream native records with per-record bounds across every consumer; preserve regular partial results for native and Pair roots; and introduce one mixed-ledger classifier shared by launcher and inventory, with regressions for each class.
