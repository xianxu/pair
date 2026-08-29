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

---

## Re-review — 2026-08-28T18:07:18-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 155 — deterministic agent session-tree inventory |
| repo | pair |
| issue file | workshop/issues/000155-agent-session-tree-inventory.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4c454436038e2ae049690bc343def9f0511fca8c..c902ed6beaec87342d77737b411a318a2c8b926b |
| command | sdlc close --issue 155 |
| reviewer | codex |
| timestamp | 2026-08-28T18:07:18-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation substantially delivers the deterministic inventory and fixes three of the four open findings, with full Go and Lua suites passing. It cannot ship yet: BR-19 remains incomplete, slugging still maintains an independent native-transcript parser contrary to the central migration contract, and the new compatibility-ledger classifier accepts malformed rows too broadly.

1. Strengths

- Missing process identity now disables only process corroboration; the unique causal-round path remains reachable and is directly exercised at `sessionwatch/run_test.go:82`.
- Native event and token-usage readers process records beyond 32 MiB with focused regressions at `events_test.go:60` and `query_test.go:83`.
- Partial Pair listings retain usable files while emitting a diagnostic, exercised end-to-end at `query_test.go:96`.
- README and atlas changes cover the public command, binding lifecycle, consumer migration, and storage model.
- The injected runtime, stateful fake, and live conformance design provide a credible ARCH-MOCK boundary.

2. Critical findings

- BR-19 remains open (`bounded-record-streaming`). `ReadRootTranscript` calls `readJSONLines`, which reconstructs the entire transcript in a `bytes.Buffer` at `scan_helpers.go:81-91`; slug then parses that full buffer at `slugcmd.go:82-88,141`. No long-transcript regression reaches this migrated consumer, so the claimed fix lacks the required red-before/green-after test. Additionally, event, usage, and transcript reads pass `acceptFinal=true`, accepting an unterminated final record even though the Spec requires truncated records to contribute no fact. Fix by exposing a bounded record/event projection consumed directly by slug, retaining only its required recent window, and add regressions for a useful record after 32 MiB and an unterminated final record.
- ARCH-DRY / ARCH-PURPOSE: slug remains an independent native-format authority. `slug.go:69-80` dispatches among local parsers, and `slug.go:84-180` begins duplicate Claude/Codex schemas; Agy and Muse duplicates also remain later in the file. Unknown agents even fail open to Claude parsing. This contradicts the issue’s “no consumer outside inventory may parse a native transcript independently” requirement and the plan’s NativeEvent rationale. Delete these adapters and consume a sessioninventory-owned text-event projection; strengthen `TestShadowSweep` to detect native record adapters outside the inventory package.

3. Important findings

- ARCH-PURPOSE: this is the 2nd finding in family `mixed-ledger-formats-are-classified`. The shared predicate at `sessionledger/record.go:118-129` classifies every single JSON object lacking `v` and `kind` but having a nonempty `agent` as compatible. Consequently rows with missing legacy fields, unknown fields, or unsupported agents avoid malformed diagnostics, and launcher parsing accepts them at `launcher/ledger.go:40-52`. Define and strictly decode the complete supported legacy shape, validate its agent, and table-test typed, exact legacy, unsupported, partial, unknown-field, trailing-value, and malformed rows across both launcher and inventory consumers.

4. Minor findings

None.

5. Test coverage notes

- Passed: `go test ./... -count=1`.
- Passed: `make test-lua`.
- Passed: focused inventory, ledger, watcher, launcher, and slug suites.
- Passed: pinned-range `git diff --check`.
- Missing: long-transcript coverage through slug/`ReadRootTranscript`; truncated-final-record rejection; negative compatibility-row classification.
- The full green suite does not detect the explicit shadow-parser violation.

6. Architectural notes for upcoming work

- ARCH-DRY: flag — native parsing remains duplicated in slug.
- ARCH-PURE: pass — forest, matching, ordering, and projections remain largely pure behind the runtime seam.
- ARCH-PURPOSE: flag — final consumer migration and bounded-record handling are incomplete.
- ARCH-MOCK: pass — production and tests share the injected external boundary and stateful fake, with live conformance coverage.

7. Plan revision recommendations

After fixing the implementation, append a `## Revisions` entry correcting the current claim that every native JSONL consumer streams bounded records. Record the shared bounded text-event projection, removal of slug’s four native parsers, truncated-record behavior, and expanded shadow-sweep enforcement.

```findings
dispose:
  - id: BR-18
    disposition: addressed
    note: |
      The watcher regression directly establishes a unique completed round when a current PID has no usable identity token; restoring the former early return would prevent the asserted binding.
  - id: BR-19
    disposition: not-addressed
    note: |
      Event and usage paths have long-record tests, but the migrated slug transcript consumer still reconstructs and parses the whole artifact without a failing long-transcript regression; the shared helper also accepts unterminated final records contrary to the bounded-record contract.
  - id: BR-20
    disposition: addressed
    note: |
      The QuerySession regression returns an established binding from valid files alongside a rejected listing entry and requires the corresponding diagnostic; the former fatal return would fail it.
  - id: BR-21
    disposition: addressed
    note: |
      A valid legacy row followed by typed authority is classified without a malformed diagnostic, and the shared parser is used by both inventory and launcher.
findings:
  - id: new
    severity: Critical
    family: native-record-parsing-is-single-source
    title: |
      Slug remains a second native transcript parser
    detail: |
      ARCH-DRY and ARCH-PURPOSE: slugcmd reads the complete transcript and maintains separate Claude, Codex, Agy, and Muse adapters, including an unknown-agent fallback to Claude. The issue explicitly requires every native parser consumer to derive from sessioninventory; expose a bounded shared text-event projection, migrate slug to it, and enforce the class in the shadow sweep.
  - id: new
    severity: Important
    family: mixed-ledger-formats-are-classified
    title: |
      Compatibility classification admits malformed and unsupported rows
    detail: |
      This is the 2nd finding in family `mixed-ledger-formats-are-classified`. The classifier treats any single JSON object with a nonempty agent and no v or kind as compatible, so partial, unknown-field, and unsupported-agent rows escape malformed diagnostics and can enter launcher history. State the exhaustive typed versus exact-legacy versus malformed rule and test the complete classification matrix in both consumers.
```

---

## Re-review — 2026-08-28T18:34:46-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 155 — deterministic agent session-tree inventory |
| repo | pair |
| issue file | workshop/issues/000155-agent-session-tree-inventory.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4c454436038e2ae049690bc343def9f0511fca8c..c81059d70abb24b09d53c3fd02008a04acea20bd |
| command | sdlc close --issue 155 |
| reviewer | codex |
| timestamp | 2026-08-28T18:34:46-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The inventory architecture and two major close-review fixes are sound, but the boundary cannot close: the repository-wide Go suite fails, and the ledger classifier still admits unsupported typed agents into launcher state. BR-19 and BR-22 are verified addressed; BR-1 has regressed and BR-23 remains incomplete.

1. Strengths

- Native transcripts are streamed with per-record limits rather than whole-file caps ([scan_helpers.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/scan_helpers.go:31)).
- Slug consumes the shared bounded event projection ([slugcmd.go](/Users/xianxu/workspace/pair/cmd/internal/slugcmd/slugcmd.go:80)); its four duplicate native parsers were removed and the shadow sweep guards against their return.
- The strict legacy compatibility classifier is shared by inventory and launcher, with unknown-field, partial-row, type, and trailing-data coverage ([record.go](/Users/xianxu/workspace/pair/cmd/internal/sessionledger/record.go:139)).
- README and atlas document the public command, provisional/established semantics, and internal activity projection.
- Core-concept declarations match the plan; pure entities remain independently testable and integration points use the injected runtime/stateful fake.

2. Critical findings

- **BR-1 — repository contract suite is red** ([plan_contract_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/plan_contract_test.go:143)). `go test ./... -count=1` fails because `cmd/internal/slugcmd/slug_test.go` appears in the moving milestone diff but is absent from `issue149M5GoSources`.

  **This is the 3rd finding in family `repository-contracts-stay-green`.** Do not merely add this filename again. State and enforce the general rule: the declaration-disposition source set must derive from one authoritative bounded range or marker inventory, rather than manually duplicating a `...HEAD` diff that changes on every later edit. This is also an ARCH-DRY flag.

3. Important findings

- **BR-23 — unsupported typed rows still enter launcher state** ([ledger.go](/Users/xianxu/workspace/pair/cmd/internal/launcher/ledger.go:54)). `sessionledger.ParseLedger` accepts a typed launch with `agent:"future"`; launcher then produces a current `LedgerEntry`, allowing `InferAgent` and history selection to consume it. A scratch regression expecting rejection failed with an entry marked `Typed:true`.

  **This is the 3rd finding in family `mixed-ledger-formats-are-classified`.** Complete the class-wide rule across both consumers: exact supported typed row, exact supported legacy row, otherwise malformed/rejected. Add launcher coverage for unsupported typed launch and binding rows. ARCH-PURPOSE is flagged because only the legacy half of the promised classification matrix was swept.

4. Minor findings

None.

5. Test coverage notes

- Focused inventory, ledger, slug, launcher, and fake-runtime packages pass.
- Mutation checks confirmed:
  - Restoring a 32 MiB whole-file failure makes all three long-transcript regressions fail.
  - Restoring permissive compatibility classification makes ledger, launcher, and inventory tests fail.
  - Reintroducing a slug-native parser makes `TestShadowSweep` fail.
- `go vet ./...`, Lua tests, named shell suites, Zellij validation, and `git diff --check` pass.
- `go test ./... -count=1` fails at the repository contract described under BR-1.

6. Architectural notes for upcoming work

- **ARCH-DRY:** Session parsing and ledger classification are correctly centralized. Flagged for the manually synchronized historical source catalog.
- **ARCH-PURE:** Pass. Forest assembly, ordering, matching, and ledger projection remain pure; IO stays behind runtime/store seams.
- **ARCH-PURPOSE:** Flagged. Unsupported typed ledger rows show the exhaustive classification purpose is not fully delivered.
- **ARCH-MOCK:** Pass. Filesystem, SQLite, process identity, PID reuse, and open-file behavior use the same injected runtime seam and stateful fake; live conformance remains available.

7. Plan revision recommendations

Append a `## Revisions` entry recording:

- The typed/legacy/malformed rule must validate supported agents in both inventory and launcher, with typed unsupported launch/binding regressions.
- The repository source-disposition contract must stop duplicating a moving `HEAD` source set; document the authoritative derived or pinned enumeration and its enforcement.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      The repository-wide Go suite again fails its declaration-disposition source contract after slug_test.go changed without entering the hand-maintained catalog.
  - id: BR-19
    disposition: addressed
    note: |
      Native events, usage, and slug text projection stream arbitrary-length JSONL with per-record bounds; restoring a 32 MiB cutoff makes the long-transcript regressions fail.
  - id: BR-22
    disposition: addressed
    note: |
      Slug now consumes TextEventWindowForRoot, the duplicate four-agent adapters are deleted, and the shadow-sweep regression fails if a native parser is reintroduced.
  - id: BR-23
    disposition: not-addressed
    note: |
      Legacy shapes are strict, but an unsupported typed ledger row still becomes a launcher LedgerEntry and can influence history or agent inference.
```

---

## Re-review — 2026-08-28T21:03:26-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 155 — deterministic agent session-tree inventory |
| repo | pair |
| issue file | workshop/issues/000155-agent-session-tree-inventory.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4c454436038e2ae049690bc343def9f0511fca8c..7d9f6ccb32b9854d62636374e332c8f08abc9ce4 |
| command | sdlc close --issue 155 |
| reviewer | codex |
| timestamp | 2026-08-28T21:03:26-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The inventory architecture, migration, documentation, and most verification are strong, and BR-1 is demonstrably fixed. BR-23 remains open: both typed and compatibility ledger decoders accept duplicate JSON keys, contradicting the claimed exhaustive exact-legacy/typed/malformed classification. Because this is the third occurrence in `mixed-ledger-formats-are-classified`, fix the decoding rule across the class before closing.

## 1. Strengths

- The pure inventory core is isolated behind the injected [`Runtime`](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/runtime.go:44), with a reusable stateful [`FakeRuntime`](/Users/xianxu/workspace/pair/cmd/internal/sessioninventorytest/fake_runtime.go:39).
- BR-1’s artifact-path and immutable issue-149 source-set contracts pass. Running the former moving-`HEAD` implementation in a scratch copy reproduced the expected failure.
- Unsupported typed agents are now rejected centrally at [`validateRecord`](/Users/xianxu/workspace/pair/cmd/internal/sessionledger/record.go:200). Removing that validation made both ledger and launcher regressions fail.
- README and atlas cover the public command, schema, lifecycle, migration, and consumer authority. The shadow sweep passes.

## 2. Critical findings

None.

## 3. Important findings

- **BR-23 — duplicate keys escape the exhaustive ledger classifier**
  [`record.go:139`](/Users/xianxu/workspace/pair/cmd/internal/sessionledger/record.go:139) and [`record.go:163`](/Users/xianxu/workspace/pair/cmd/internal/sessionledger/record.go:163) use `json.Decoder` directly. Go accepts duplicate object keys, so rows such as duplicate `agent` fields are admitted as valid compatibility or typed records instead of malformed. A clean scratch regression confirmed both cases.

  **This is the 3rd finding in family `mixed-ledger-formats-are-classified`.** Do not patch only those examples. State and enforce the complete rule: every typed or compatibility row has exactly one occurrence of every permitted key, no unknown keys or trailing values, complete required fields, correct types, and supported enums—including nested objects. Reuse [`strictjson.Decode`](/Users/xianxu/workspace/pair/cmd/internal/strictjson/decode.go:15), then add duplicate-key cases to the ledger, launcher, and inventory matrices. This flags `ARCH-DRY` and `ARCH-PURPOSE`.

## 4. Minor findings

None.

## 5. Test coverage notes

- Passed focused artifact-path, issue-149 contract, ledger, launcher, and inventory tests.
- Passed race tests for inventory, fake runtime, ledger, Pair log, watcher, and launcher.
- Passed Lua, watcher, review, changelog, terminal-shortcut, Zellij configuration, and `git diff --check` verification.
- Scratch red checks proved BR-1 and the unsupported-agent portion of BR-23.
- `go test ./... -count=1` reached one environment failure: `cmd/pair-go` cleanup invokes `/bin/ps`, which the review sandbox rejects with `operation not permitted`. Other reported packages passed.

## 6. Architectural notes for upcoming work

- `ARCH-DRY`: **Flag** — ledger decoding duplicates the existing strict JSON authority.
- `ARCH-PURE`: **Pass** — matching/model logic is pure; filesystem, SQLite, and process access remain behind `Runtime`.
- `ARCH-PURPOSE`: **Flag** — the promised exhaustive malformed-row class remains incomplete.
- `ARCH-MOCK`: **Pass** — production inventory flow and stateful test flow share the runtime boundary, with live conformance isolated separately.

## 7. Plan revision recommendations

Append a revision recording that duplicate keys—at the top level and inside typed nested structures—are malformed for both formats; both decoders now derive strictness from `strictjson.Decode`; and ledger, launcher, and inventory tests cover the same complete matrix.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Artifact-path and immutable issue-149 source-set contracts pass; the prior moving-HEAD implementation fails in a scratch reproduction.
  - id: BR-23
    disposition: not-addressed
    note: |
      Partial, unknown-field, and unsupported-agent cases are fixed, but duplicate-key typed and compatibility rows are still accepted rather than classified malformed.
```
