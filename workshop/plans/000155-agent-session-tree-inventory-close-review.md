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

---

## Re-review — 2026-08-28T21:10:31-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 155 — deterministic agent session-tree inventory |
| repo | pair |
| issue file | workshop/issues/000155-agent-session-tree-inventory.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4c454436038e2ae049690bc343def9f0511fca8c..935001261b6e4279f66675356620236ed3dde45b |
| command | sdlc close --issue 155 |
| reviewer | codex |
| timestamp | 2026-08-28T21:10:31-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation is broadly strong and all repository verification passed, but BR-23 remains incomplete. Ledger decoding now rejects duplicate and unknown keys through the shared strict decoder; however, malformed nested watermarks and nullable legacy fields can still be classified as valid. This violates the stated exhaustive typed/exact-legacy/malformed rule and can widen offline recovery beyond the recorded launch boundary.

```findings
dispose:
  - id: BR-23
    disposition: not-addressed
    note: |
      Duplicate, unknown, unsupported, partial, and trailing-value cases are covered, but missing or null nested event_position and explicit null legacy_import fields are still accepted instead of classified malformed.
```

1. Strengths

- Typed and compatibility rows share one classification authority in [record.go](/Users/xianxu/workspace/pair/cmd/internal/sessionledger/record.go:117), consumed by both launcher and inventory paths.
- Recursive duplicate-key rejection correctly covers nested objects in [decode.go](/Users/xianxu/workspace/pair/cmd/internal/strictjson/decode.go:37).
- The inventory maintains a clear pure-core/runtime boundary, with a portable stateful fake and production `sqlite3`/process/filesystem effects behind the same interface.
- The Core Concepts table is enforced bidirectionally against source declarations.
- README and atlas changes document the public command, binding lifecycle, consumer migration, and architectural surface.

2. Critical findings

None.

3. Important findings

- **BR-23 remains open — exact ledger classification still admits malformed fields.** [record.go](/Users/xianxu/workspace/pair/cmd/internal/sessionledger/record.go:22) decodes `NativeWatermark.EventPosition` into a plain `uint64`. Consequently, both an omitted `"event_position"` and `"event_position":null` become zero and pass validation at line 192; zero itself is valid baseline state, so absence cannot be distinguished from an explicit value. A malformed launch can therefore move its watermark backward and make pre-launch events eligible for offline correlation. Similarly, `compatibilityWireRecord.LegacyImport *bool` at line 92 accepts an explicitly null value as though the optional field were absent.

  **This is the 3rd finding in family `mixed-ledger-formats-are-classified`.** Complete the class rule, not merely these examples: every variant-required key must be present exactly once with the correct non-null type; optional keys may be absent, but when present must have the declared type. Use wire-only pointer/presence types for nested watermark fields and `legacy_import`, then extend the ledger, launcher, and inventory matrices with missing/null nested fields and null optional fields.

4. Minor findings

None.

5. Test coverage notes

- Passed: `go test ./... -count=1`.
- Passed: focused ledger, launcher, and inventory tests.
- Passed: `go vet ./...`, `make test-lua`, four named shell suites, Zellij configuration validation, and `git diff --check`.
- Existing duplicate-key regressions exercise ledger, launcher, and inventory consumers and would fail under the replaced standard decoder’s overwrite behavior.
- No test covers omitted/null `event_position` or explicit-null `legacy_import`; those are the missing BR-23 red cases.

6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass—the classifier and strict JSON traversal are shared.
- `ARCH-PURE`: Pass—forest, correlation, ordering, and classification logic remain separate from the injected IO shell.
- `ARCH-PURPOSE`: Flagged by BR-23—the implementation still delivers only a subset of the promised exhaustive classification rule.
- `ARCH-MOCK`: Pass—the external runtime has a stateful fake behind the production seam, with opt-in live conformance coverage.

7. Plan revision recommendations

None. The existing “make every ledger object structurally strict” revision already states the correct intended rule; implementation and regression coverage need to be brought into conformance with it.

---

## Re-review — 2026-08-28T21:19:45-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 155 — deterministic agent session-tree inventory |
| repo | pair |
| issue file | workshop/issues/000155-agent-session-tree-inventory.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4c454436038e2ae049690bc343def9f0511fca8c..6cf116558cefdc33329cf8537651141f96a16414 |
| command | sdlc close --issue 155 |
| reviewer | codex |
| timestamp | 2026-08-28T21:19:45-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The inventory architecture, consumer migration, documentation, and BR-23 classification fix are well executed and thoroughly tested. One blocking durability mismatch remains: a ledger append may return failure after its bytes become readable, while recovery treats those bytes as authoritative. This violates the explicit guarantee that a failed binding append leaves the launch provisional.

1. Strengths

- BR-23 is addressed through one shared strict classifier in `sessionledger`, used by both launcher and inventory consumers.
- The classifier rejects missing, null, duplicate, unknown, trailing, unsupported-agent, and invalid kind-specific fields.
- Reverting the final field-presence fix in a scratch checkout made all three relevant consumer tests fail, confirming the fix is reachable and genuinely pinned.
- Core-concepts entities exist at their documented locations with appropriate pure/integration separation.
- README and atlas changes cover the public CLI, identity model, architecture, and migration surface.

2. Critical findings

- `cmd/internal/sessionledger/store.go:126`: **ARCH-PURPOSE, ARCH-MOCK — failed ledger writes can still become recovery authority.** The store writes the complete record before `Sync`, `Close`, directory sync, and deferred unlock. Any failure at `store.go:130-139` returns an error while leaving a parseable row visible. `ParseLedger` accepts that row, and `sessioninventory/pair_inventory.go:74-78` subsequently treats it as authoritative. For a binding, this can establish a root even though `AppendBindingIfCurrent` reported failure, contradicting the Spec at lines 633–640 and the plan’s “failed binding append leaves the latest launch provisional” contract.

  Fix the class, not one site: enumerate short-write-after-N-bytes, file-sync, close, directory-sync, and unlock failures for both launch and binding records. Define a commit-result protocol under which the caller’s result and subsequent recovery agree. If post-write failures are inherently indeterminate, revise the contract and return/reconcile an explicit indeterminate outcome rather than reporting an ordinary failed append. Add stateful recovery tests proving every failure point either remains non-authoritative or is reported and recovered as committed.

3. Important findings

None.

4. Minor findings

None.

5. Test coverage notes

- Passed: `go test ./... -count=1`.
- Passed: `go vet ./...`.
- Passed: complete Lua suite.
- Passed: session-watch, review-toggle, changelog-session-key, terminal-shortcut, and queue-send shell suites.
- Passed: Zellij configuration validation and pinned-range `git diff --check`.
- Existing `TestLedgerStoreFailureRetryUsesNextPhysicalOrdinal` checks only that a failed row consumes an ordinal; it does not assert whether that row remains authoritative. Its fsync fake already exposes the missing recovery assertion.

6. Architectural notes for upcoming work

- **ARCH-DRY: pass.** Native parsing, ledger classification, activity, and consumer authority are consolidated; the shadow sweep enforces the boundary.
- **ARCH-PURE: pass.** Correlation and ordering remain pure, with filesystem/process/SQLite behavior behind injected runtimes.
- **ARCH-PURPOSE: flag.** The implementation does not yet deliver the promised failed-binding durability semantics.
- **ARCH-MOCK: flag.** A stateful filesystem fake exists, but post-write failure tests do not validate recovered authority across the shared production seam.

7. Plan revision recommendations

Add a `## Revisions` entry defining the ledger commit point and exhaustive result/recovery behavior for write, fsync, close, directory-sync, and unlock failures. If the filesystem cannot guarantee “error means provisional” after bytes are written, explicitly replace that promise with an indeterminate-result protocol and update the Spec, plan, callers, and tests together.

```findings
dispose:
  - id: BR-23
    disposition: addressed
    note: |
      The shared strict classifier now enforces exact typed versus exact legacy versus malformed rows across ledger, launcher, and inventory consumers; reverting the field-presence fix makes all three regression tests fail.
findings:
  - id: new
    severity: Critical
    family: ledger-append-result-matches-authority
    title: |
      Failed binding appends can still become authoritative
    detail: |
      cmd/internal/sessionledger/store.go:126-139 returns errors after a complete parseable row may already be visible, while cmd/internal/sessioninventory/pair_inventory.go:74-78 later accepts that row as recovery authority. Enumerate every post-write failure point and make the returned outcome agree with recovered authority; add stateful launch and binding tests covering short writes, file sync, close, directory sync, and unlock failures (ARCH-PURPOSE, ARCH-MOCK).
```

---

## Re-review — 2026-08-28T21:33:43-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 155 — deterministic agent session-tree inventory |
| repo | pair |
| issue file | workshop/issues/000155-agent-session-tree-inventory.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4c454436038e2ae049690bc343def9f0511fca8c..dfdf5b364e95c36f6d29ee5404bf114a7996d630 |
| command | sdlc close --issue 155 |
| reviewer | codex |
| timestamp | 2026-08-28T21:33:43-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The deterministic inventory, consumer migration, documentation, and verification are strong, but two authority-result mismatches block closure. BR-24 remains open because production callers ignore the new store outcomes. The Pair-log writer has the same class of defect after publication: directory-sync or unlock failure reports failure even though the new log entry is already visible and authoritative.

## 1. Strengths

- The Core Concepts inventory is executable and matches declarations through [concept_contract_test.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/concept_contract_test.go:19).
- The pure inventory model is cleanly separated from the injected runtime, with a reusable stateful fake.
- [shadow_test.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/shadow_test.go:1) enforces removal of independent native parsers and first/newest selection logic.
- README and atlas changes document the public CLI, provisional/established semantics, schema, and new architecture.
- Store-level tests thoroughly cover incomplete byte prefixes and post-write failure stages for typed launch and binding records.

## 2. Critical findings

### BR-24 — production callers discard authoritative append outcomes

[store.go](/Users/xianxu/workspace/pair/cmd/internal/sessionledger/store.go:15) correctly distinguishes `not-authoritative`, `indeterminate`, and `committed`, but `AppendOutcomeOf` has no production call sites.

- [lifecycle.go](/Users/xianxu/workspace/pair/cmd/internal/sessionwatch/lifecycle.go:78) discards a launch record and ordinal whenever the returned error is non-nil.
- Explicit-resume binding does the same at [lifecycle.go](/Users/xianxu/workspace/pair/cmd/internal/sessionwatch/lifecycle.go:90).
- Live establishment returns the old provisional projection and skips config refresh at [lifecycle.go](/Users/xianxu/workspace/pair/cmd/internal/sessionwatch/lifecycle.go:128), even for a committed binding with an unlock error.
- Legacy append discards the ordinal/outcome at [osruntime.go](/Users/xianxu/workspace/pair/cmd/internal/launcher/osruntime.go:570).

Recovery later accepts these rows, so production’s reported state can still disagree with recovered authority. Existing tests prove only the store classification; none fails when lifecycle callers ignore it.

Fix the class across launch, binding, and legacy consumers: preserve the returned record, advance state on committed outcomes, define immediate reconciliation for indeterminate outcomes, and add lifecycle-level stateful tests.

### New — Pair-log publication repeats the authority mismatch

[store.go](/Users/xianxu/workspace/pair/cmd/internal/pairlog/store.go:94) publishes the replacement before directory sync. A directory-sync failure at line 97 or unlock failure through line 59 then returns an ordinary error, causing [submission.lua](/Users/xianxu/workspace/pair/nvim/submission.lua:7) to retain the draft and suppress submission even though the new Pair-log turn is already readable.

Retry appends the same authored turn again. That duplicate makes the fingerprint non-unique in [round.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/round.go:129), potentially preventing the causal round from establishing or recovering the native root.

This is the 2nd finding in family `ledger-append-result-matches-authority`. Do not patch only directory sync: define the rule for every authoritative append/replace transaction, enumerate all pre- and post-publication failures, make retries idempotent, and test the production submission-to-recovery flow.

## 3. Important findings

None.

## 4. Minor findings

None.

## 5. Test coverage notes

Passed:

- `go test ./... -count=1`
- Focused ledger, inventory, watcher, launcher, and fake-runtime tests
- Core-concept, shadow-sweep, CLI matrix, rendering, and scanner contracts
- `go vet ./...`
- `make test-lua`
- Named session-watch, review, changelog, and terminal shell suites
- Pinned-range `git diff --check`

Missing blocking regressions:

- Lifecycle tests consuming each `AppendOutcome`.
- Pair-log directory-sync and unlock failures.
- Retry after an already-published Pair-log entry.
- End-to-end proof that such retry cannot poison causal-round matching.

## 6. Architectural notes

- **ARCH-DRY — flag:** authoritative publication outcomes are handled in the ledger store but independently—and incompletely—in Pair-log storage. Establish one shared outcome rule/vocabulary.
- **ARCH-PURE — pass:** matching, ordering, forest construction, binding resolution, and rendering remain pure; IO is behind explicit seams.
- **ARCH-PURPOSE — flag:** recovery authority must agree with the production operation result end-to-end, not merely inside store tests.
- **ARCH-MOCK — flag:** stateful doubles exist, but fault coverage stops below the production lifecycle/submission flows where the mismatch occurs.

## 7. Plan revision recommendations

Append revisions covering:

- “End-to-end append outcome consumption”: enumerate launch, explicit/live binding, and legacy consumers; define behavior for all three outcomes.
- “Idempotent Pair-log publication outcomes”: define the publication point, directory-sync/unlock semantics, retry behavior, and production-flow tests.

```findings
dispose:
  - id: BR-24
    disposition: not-addressed
    note: |
      The store labels post-write authority, but AppendOutcomeOf has zero production call sites: PrepareLaunch, ObserveAndPersist, and AppendLegacy collapse indeterminate and committed results into ordinary failure, discard returned records or ordinals, and lack lifecycle tests proving committed advancement or indeterminate reconciliation.
findings:
  - id: new
    severity: Critical
    family: ledger-append-result-matches-authority
    title: |
      Published Pair-log entries are reported as failed and duplicated on retry
    detail: |
      This is the 2nd finding in this family. SessionLogStore renames the authoritative replacement before directory sync and unlock, but either later failure is returned as an ordinary append failure. Submission is suppressed while the entry remains readable; retry appends it again, making exact turn evidence non-unique. Define the outcome rule for every authoritative append or replacement, make post-publication retries idempotent, and add stateful production-flow tests for directory-sync and unlock failure.
```

---

## Re-review — 2026-08-28T22:03:27-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 155 — deterministic agent session-tree inventory |
| repo | pair |
| issue file | workshop/issues/000155-agent-session-tree-inventory.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4c454436038e2ae049690bc343def9f0511fca8c..222dfb3f94da7fdde6bd54e94d42fdb74969ad07 |
| command | sdlc close --issue 155 |
| reviewer | codex |
| timestamp | 2026-08-28T22:03:27-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The ledger publication fix is well structured and BR-24 is addressed, but BR-25 remains open: changing a preserved draft after an indeterminate append creates a new append ID while retaining the prior readable-but-unsent entry as correlation evidence. The new publication-outcome entity also escapes the Core Concepts inventory. Focused tests pass; full repository verification could not complete because the sandbox denied `/bin/ps`.

1. Strengths

- `commitoutcome.Outcome` provides a shared, fail-safe publication vocabulary.
- Ledger tests cover every incomplete byte boundary and post-write failure for launch and binding records.
- Ledger lifecycle consumers reconcile exact ordinal/bytes rather than appending duplicate generations.
- Pair-log identical-body retries are idempotent and preserve causal-round uniqueness.
- Atlas documentation covers publication outcomes and reconciliation.

2. Critical findings

- BR-25 — not addressed: [nvim/submission.lua:12](/Users/xianxu/workspace/pair/nvim/submission.lua:12) replaces the pending append ID whenever authored text changes. After directory-sync failure, the first entry is already readable but submission was suppressed. Editing the draft then creates a second ID and appends the revised body, leaving the first unsent entry in `ParsePairLog`’s correlation facts. No test exercises failure → edit → retry. Define the complete retry-state rule, including changed or cleared drafts, and add a production-flow regression proving no unsent entry can authorize correlation. This is the third observed instance in family `ledger-append-result-matches-authority`; fix the full retry-state class.

- Core Concepts contract — [outcome.go:9](/Users/xianxu/workspace/pair/cmd/internal/commitoutcome/outcome.go:9) introduces the central `Outcome`/`Error` entity, but it is absent from the plan’s Core Concepts table and has no concept declaration. Meanwhile [concept_contract_test.go:26](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/concept_contract_test.go:26) validates only M1/M2 and cannot detect unmarked final-round entities. This is the third finding in family `core-concepts-match-code`. Establish a rule covering every issue-owned domain type and every `Introduced` value, rather than adding only this missing row.

3. Important findings

None.

4. Minor findings

None.

5. Test coverage notes

Focused Go packages, Lua tests, shell integration tests, and `git diff --check` passed. `go test ./... -count=1` could not complete: `cmd/pair-go` tests were blocked by the sandbox denying `fork/exec /bin/ps: operation not permitted`. Per the review contract, this unavailable required inspection independently prevents SHIP.

6. Architectural notes

- `ARCH-DRY`: Pass for shared outcome handling and ledger reconciliation; the plan-contract omission is flagged separately.
- `ARCH-PURE`: Pass. Outcome classification is pure and filesystem effects remain in thin stores/runtimes.
- `ARCH-PURPOSE`: Flag. Changed-draft retries still permit unsent Pair-log evidence, under-delivering the authoritative retry contract.
- `ARCH-MOCK`: Pass for the reviewed failure seams: stateful tests cover short writes, sync, close, directory sync, unlock, recovery parsing, and identical-body retries.

7. Plan revision recommendations

Append a `## Revisions` entry that:

- Defines behavior for indeterminate append followed by unchanged, edited, or cleared draft.
- Requires Pair-log correlation to exclude every operator turn not proven submitted.
- Adds the shared `CommitOutcome` entity to Core Concepts.
- Extends the concept contract across every milestone/final introduction and prevents unmarked domain entities from escaping it.

```findings
dispose:
  - id: BR-24
    disposition: addressed
    note: |
      Store and lifecycle tests cover launch and binding authority across every incomplete byte boundary plus write, file-sync, close, directory-sync, and unlock failures; production consumers reconcile indeterminate rows by exact ordinal and bytes.
  - id: BR-25
    disposition: not-addressed
    note: |
      Identical-body retry is idempotent, but editing the preserved draft after an indeterminate published append mints a new ID and leaves the prior readable-but-unsent entry as correlation evidence; no regression covers this branch.
findings:
  - id: new
    severity: Critical
    family: core-concepts-match-code
    title: |
      Shared publication outcome escapes the Core Concepts contract
    detail: |
      This is the 3rd finding in family `core-concepts-match-code`. The new central Outcome/Error entity is absent from the plan table and unmarked, while the contract checks only M1/M2. State and enforce the class rule for every issue-owned domain type and every introduction stage, then sweep the full range.
```
