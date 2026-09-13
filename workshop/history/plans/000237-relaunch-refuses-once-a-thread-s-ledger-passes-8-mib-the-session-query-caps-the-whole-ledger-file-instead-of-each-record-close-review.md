# Boundary Review — pair#237 (whole-issue close)

| field | value |
|-------|-------|
| issue | 237 — relaunch refuses once a thread's ledger passes 8 MiB: the session query caps the whole ledger file instead of each record |
| repo | pair |
| issue file | workshop/issues/000237-relaunch-refuses-once-a-thread-s-ledger-passes-8-mib-the-session-query-caps-the-whole-ledger-file-instead-of-each-record.md |
| boundary | whole-issue close |
| milestone | — |
| window | 64b0119c7a8b8588b8c6f5ae9dffcdf8e7b16f99..c9c8e1702d869a5d2d6d177f11ff574f96ab703f |
| command | sdlc close --issue 237 |
| reviewer | claude |
| timestamp | 2026-09-12T17:57:05-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

```findings
findings:
  - id: new
    severity: Minor
    family: done-when-operator-verified
    title: |
      Done-when "the operator's thread relaunches" is delegated to the operator, not verified in-range
    detail: |
      The Log verifies the owner query resolves on the real 8,482,765-byte ledger (fixed vs control build), which is the mechanism, but the couch relaunch itself was not exercised. Say so in the close's --verified evidence rather than letting the Plan row read as fully delivered.
  - id: new
    severity: Minor
    family: operating-envelope-unstated
    title: |
      OSRuntime.ReadAt opens the file once per 64 KiB chunk, so the ledger read now costs ~130 open/close cycles per query on the operator's ledger
    detail: |
      runtime_os.go:163 re-opens per range. Pre-existing for transcripts, and #238 shrinks the file, but titlepoller multiplies it on a cadence. Note it in #238 or open the file once per frameJSONLArtifact call if it ever shows up in a profile. ARCH-CONSTRAINTS.
  - id: new
    severity: Minor
    family: test-oracle-weakened
    title: |
      TestQuerySessionEstablishesProofAfterCatalogLossWithoutTranscriptRead went from "ledger read exactly once" to "ledger read at least once"
    detail: |
      query_test.go:63. The old ReadFile==1 pinned single-read; the new ReadAt!=0 would not catch a double read of the ledger. Asserting an exact chunk count (ceil(size/readChunkSize)+1) keeps the original claim.
```

The fix is correct and complete for its stated class. The whole-file cap on the owner ledger is replaced by the per-record bound at both sites the Spec names, the chunk loop is consolidated under one framer with the two consumer-specific decisions (CR strip, unterminated tail) pushed to the consumers, and the four new tests pin real behavior. I independently reverted the two call sites in a scratch copy of HEAD: both >cap regression tests go red on the old read, and the per-record and mid-append tests still pass on both, so they are pins of unchanged behavior rather than tautologies. Package tests, vet, and a 20-second fuzz run pass. The full `go test ./...` failures are all sandbox process-spawn denials in pty/process packages that do not import the changed package, so they are environmental.

1. Strengths
- `frameJSONLArtifact` at `scan_helpers.go:59` is a clean consolidation: the loop lives once, `visitJSONLinesAt` reduces to a CR-strip adapter plus the truncated-tail rule, and `readJSONLArtifact` reduces to a byte-for-byte reassembler. PQ-1 is genuinely addressed, not papered over.
- The mid-append test at `ledger_read_test.go:53` defends the concurrency case that matters: a ledger being appended must stay readable, and `ParseLedger` at `record.go:257` already owns the partial-tail rule, so the reader correctly hands it through instead of inventing a second rule.
- The fuzz test pins the invariants that must agree between the two consumers, including `\r\n` handling and the empty input, and its seeds cover the chunk-boundary case implicitly through the fake's `ReadAt`.
- The test-oracle correction is honest. `OperationCountForRoot` in `fake_runtime.go:170` asserts "no native body reads" which is what those tests were actually claiming, and `test.artifacts` in the class test are all native roots, so the assertion is not vacuous.
- Atlas paragraph in `session-identity.md:88` records the invariant, not the fix.

2. Critical findings
None.

3. Important findings
None.

4. Minor findings
- Done-when's relaunch bullet is operator-verified, not verified in-range. Put that in `--verified`.
- Per-chunk file open in `OSRuntime.ReadAt`, ~130 opens per query on the operator's ledger, multiplied by titlepoller's cadence. ARCH-CONSTRAINTS. Belongs with #238.
- `query_test.go:63` weakened from exactly-once to at-least-once on the ledger read.
- `readJSONLArtifact` builds `body` by repeated append, so peak memory is roughly two to three times the ledger size during a query. Within the envelope the Spec declares. Not actionable here.

5. Test coverage notes
- Both >cap regressions fail without the fix (verified by revert in a scratch copy). Per-record cap and mid-append tolerance are pinned. Fuzz pins reader/framer agreement.
- `OSRuntime.ReadAt` has its own eof-at-boundary tests in `runtime_os_test.go:47`, so the production side of the seam the framer depends on is covered.
- Not covered: a ledger whose length is an exact multiple of `readChunkSize`. The code path handles it (zero-length final chunk with eof returns pending), and the fuzz input cap of 64 KiB equals one chunk, so the fuzzer can reach exactly one such boundary. A deterministic seed at exactly `readChunkSize` bytes would make this explicit.

6. Architectural notes
- ARCH-DRY: pass. One loop, two thin consumers.
- ARCH-PURE: pass. Framing logic is IO-adjacent through the injected `Runtime` seam, which is this package's established pattern; a pure byte-stream framer would be marginally cleaner but is not warranted for this change.
- ARCH-PURPOSE: pass. The class is "whole-file cap on a record-stream read through `sessioninventory.Runtime`". Enumeration: `query.go:102` and `pair_inventory.go:69`, both fixed. `pair_inventory.go:96` and `:109` are `log-*.md` and `config-*.json`, correctly kept. The launcher's own ledger reads use uncapped `os.ReadFile` and are outside the class. PQ-4's caller roster is corrected in the working-tree issue file.
- ARCH-MOCK: pass. `FakeRuntime` is a stateful fake behind the same seam production uses; `OperationCountForRoot` adds observability only.
- ARCH-CONSTRAINTS: pass with the Minor note above. The Spec names the envelope and the bound (#238); the per-chunk open cost is worth recording there.
- ARCH-SECURE: pass. The ledger is untrusted persisted input; a record over the cap fails visibly with `ErrReadLimit`, a partial tail degrades to a malformed ordinal diagnostic rather than a fabricated record.
- ARCH-ORDER: pass. The reader holds no state between events; the one interleaving that matters (append during read) is tested and its outcome is defined by `ParseLedger`.

7. Plan revision recommendations
None. The working-tree issue file already carries the Spec additions for PQ-1, PQ-2, and PQ-4, the Plan ticks, and the Log. Those edits are uncommitted at this head; commit them with the close so the archived issue matches the code.
