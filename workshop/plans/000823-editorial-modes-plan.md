---
collaboration-mode: discussion
presentation-mode: highlight
processing-mode: free-form
---

# Unified editorial modes implementation plan

> **For agentic workers:** Consult AGENTS.md §3 to choose bounded subagent work or session-local execution. Use the executing-plans and test-driven-development skills after TL approval and `sdlc change-code`. This is one atomic feature with one close review; task numbers are not milestone boundaries.

**Goal:** Resolve one document snapshot into per-section editorial rules, use those rules in every Couch agent-write path, and provide an explicit, semantics-preserving legacy-map migration.

**Architecture:** A pure editorial compiler owns parsing, inheritance, diagnostics and edit classification. The existing session-sync transaction compiles its base snapshot once, validates edits against it and publishes through its existing lock/CAS boundary. Parley consumes additive per-span presentation metadata; Couch does not add an independent review orchestrator or side-doc delivery route.

**Tech stack:** Go, existing sessionsync transactions and marker/edit-record machinery, Goldmark Markdown parsing and YAML nodes from the ariadne dependency where needed, Lua review consumers, pinned universal-review fixtures.

## Approval boundary and alternatives

Status: planning only. No production code or executable fixtures have changed. TL approval precedes `change-code`, its fresh-context plan-quality gate and the estimate.

Recommended: one compiler and one write-policy guard shared by read, write, apply and the standalone review workbench (ARCH-DRY, ARCH-PURE). Alternatives rejected: three independent frontmatter readers would disagree on canonical presence and inheritance; changing only the emitted prompt would leave write/apply paths able to bypass protected settings. A new review orchestration service is unnecessary: routing prose is already the reviewing agent's responsibility.

Three boundaries need explicit TL agreement:

1. **Mechanical enforcement versus semantic review.** Couch can enforce immutable settings, region bounds, chat-item preservation, frontier/scope restrictions and presentation conversion. It exposes resolved collaboration instructions, processing briefs and free text to the reviewing agent. Classifying an ask as a question or imperative, answering it well, and following a copy-editing brief remain agent obligations, tested through contract/native qualification rather than claimed to be decidable from arbitrary replacement prose.
2. **Operator migration.** Existing operator-acquire proves registered document/turn ownership, not human identity; session-sync CLI is local IO. Recommend a dedicated constrained preview/apply command with explicit operator confirmation, whole-source CAS and the normal registry lock. It only applies the compiler's verified map-to-comments transform, never arbitrary supplied content. This is operator-run workflow enforcement, not a same-UID security boundary. If a stronger authenticated human channel is required, ship preview output only until such a channel exists; do not invent an `--operator` bypass.
3. **Consumer and creator ownership.** Parley #419 owns mixed-span presentation rendering and section UI; #418 owns document fallback/fill. Couch has no production side-doc allocator to change. Update Couch's emitted guidance and compatible link handling here; actual Parley writers/month allocation remain #419. Do not add a new Couch filesystem creator merely to satisfy an inventory assumption.

## Contract pin and compatibility

Authoritative producer: ariadne#562, landed commit `4f10a45d038acb2324c5a19e74a78f4a593571e6`. The dependency checkout contains it; the root Go module currently predates it, so any new imported base helpers require an explicit module upgrade during implementation. `workshop/evidence/000823/contract-provenance.json` pins source paths, sizes and SHA256 digests, including all eight editorial-fill cases (24 files), heading-chat, mode files and prose/adapter contract. This is provenance, not fabricated test evidence. Copy source bytes only after implementation admission; test the digests and retain provenance with the fixtures.

Defaults: discussion / highlight / free-form. Resolve each axis independently: nearest applicable heading override, document canonical scalar or readable legacy component, then default. Subsections inherit per axis; text before the first heading uses document settings. Read all settings from the round's base snapshot.

Canonical key **presence** beats its alias. Missing processing mode and instruction components fall back independently to legacy `process-instruction`. Resolve exact/unique mode prefixes through mode metadata, not another name list. Multiple leading legacy modes remain readable with all briefs in order and first-mode flags; canonical fill refuses the whole document unchanged. Ordinary fill retains alias lines and unknown metadata byte for byte and does not migrate maps.

Document presentation absent/empty/unknown resolves to highlight; unknown nonempty warns. An empty/unknown canonical value never falls through to the legacy alias and is never rewritten during fill. Heading declarations are strict: empty, unknown, duplicate or malformed recognized declarations refuse with location and valid choices. Ordinary comments, detached comments and fenced examples remain text.

Legacy maps remain readable with warnings. Normalize heading names exactly as #562 specifies, including one trailing parenthetical annotation. Duplicate normalized heading occurrences **all** match, at ATX levels 1–6. Conflicting normalized assignments fail; unmatched map names warn on read and block migration to avoid losing intent. Requested migration replaces the map with its document scalar and adds/merges comments at every matched occurrence, preserving other axes; conflicts refuse rather than pick a winner.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| `Catalog` | `cmd/internal/editorial/catalog.go` | new |
| `DocumentPolicy` / `Region` | `cmd/internal/editorial/document.go` | new |
| `OwnedSource` / `Diagnostic` | `cmd/internal/editorial/source.go` | new |
| `EditPolicy` | `cmd/internal/editorial/edits.go` | new |
| `Migration` | `cmd/internal/editorial/migrate.go` | new |
| `Round` / `Span` | `cmd/internal/sessionsync/round.go` | modified |
| Scoped hash input | `cmd/internal/sessionsync/file.go` | modified |

- **Catalog:** immutable per binary; embeds pinned mode metadata and briefs once. Collaboration values come from `vocab.CollaborationModes`; the presentation resolver owns its enum/fallback. Catalog resolution is injected into the compiler, with a small in-memory catalog for unit tests. Future mode additions change the producer files, not a completion-only enum.
- **DocumentPolicy:** one snapshot owns an ordered region table, parent heading relationships, effective axis values, processing flags/briefs/instruction and diagnostics. The compiler uses ATX headings outside code, not the existing H2-only section-address helper. Existing `--section` addressing stays unchanged. Unit tests cover inheritance, ordinary comments, fences, duplicate names and aliases without IO.
- **OwnedSource:** exact source ranges and bytes of all canonical settings, legacy aliases/maps, processing instruction, and attached heading directives. Retain unknown fields and unrelated text. It records directive-to-heading association so moving/renaming/reparenting a directive cannot silently change authority. Insert/delete/change of a protected declaration is rejected on agent paths in every presentation mode. Operator editor writes remain normal editor writes.
- **EditPolicy:** consumes resolved base edits plus DocumentPolicy and produces allowed direct/proposed pieces or a located refusal before effects. It never re-resolves permissions from agent-written output. Existing marker parsing, ready/pending rules and side-doc exceptions remain the owner of marker semantics. New region policy feeds them rather than forking them.
- **Migration:** pure `PlanMigration(source) -> {beforeHash, after, diagnostics}`. Compile before and after; verify effective collaboration routing for every original heading/body range is identical. Comments preserve unrelated axes/formatting. Its explicit operator-run frontmatter fill is the production consumer of the additive fill transform: materialize missing effective scalar defaults without replacing existing values or alias lines. Multiple leading legacy processing modes refuse the entire migration rather than discard a brief. Ordinary agent writes never invoke fill. No clocks, filesystem scans, or external document writes in this entity. Its output may later support an authenticated operator UI without changing the transform.
- **Round/Span:** retain current document presentation plus an optional per-span presentation value. Preserve unknown JSON fields through any decode/re-encode; validate known field types and enums. Old artifacts with absent per-span values retain document fallback, then highlight. Future additions do not require a lossy rewrite.
- **Scoped hash:** section reads/writes bind both target bytes and the base editorial authority source, including document defaults and heading context. Whole-document hashes stay byte hashes. The section hash is opaque: document policy changes must stale a section write even when its body is identical. Specify a domain-separated digest and update all returned scope hashes consistently; do not require callers to derive it locally or add an unauthenticated policy bypass flag.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Policy-bearing read | `cmd/internal/sessionsync/sync.go` | modified | snapshot, Shadow acknowledgement, registry transaction |
| Shared agent-write guard | `cmd/internal/sessionsync/round.go`, `sync.go`, `applyfile.go` | modified | BuildEdit/contentStep/prepareWrite, file publish and round publish |
| Operator migration CLI | `cmd/internal/couchcmd/session_sync.go` | new | constrained transform under existing document lock/CAS |
| CLI/help contract | `cmd/internal/couchcmd/run.go`, `cmd/internal/couchcmd/skills/couch/SKILL.md` | modified | agent-facing settings, refusal recovery and operator invocation |
| Review consumers | `nvim/review/round_watch.lua`, `reconstruct.lua`, `record.lua` | modified | live and resume decorations/record serialization |
| Parley round consumer | parley.nvim #419 | external dependency | additive span metadata and mixed-section rendering |

No external service is added. Transaction tests use real temporary files/registry state and controlled barriers; existing disposable Couch harness supplies native agent qualification. No call-count mocks substitute for lock/CAS and publication tests.

## Public data and transaction contract

Add a resolved editorial payload to `ReadResult`: document defaults, ordered byte ranges with effective axes and processing metadata, source hash and warnings. Keep existing body/file/section/hash fields. Section reads report relevant regions in document coordinates explicitly; never imply offsets are relative to the returned section. CLI help explains coordinate origin. Read validates before advancing Shadow; malformed policy does not acknowledge a new baseline accidentally.

Proposed round wire shared with Parley #419:

```json
{"presentation":"highlight","spans":[{"start":[8,0],"end":[8,3],"old":"old","new":"new","presentation":"direct"}]}
```

`Span.presentation` is optional for legacy artifacts, and is `direct|highlight|propose` when present. New rounds stamp each span from base policy. Keep `Round.presentation` as the document default; no invented `mixed` value. Carry the same optional field into commit `review-records` and resume reconstruction. Direct spans receive no change decoration, highlight spans do, and proposal markers retain existing proposal rendering. Known invalid/null values and duplicate JSON keys at any object depth refuse, including duplicates nested in otherwise unknown members; unrelated valid extension fields survive rewrites. Mixed-mode UI acceptance requires #419's consumer upgrade; an older Parley ignoring this field cannot be claimed compatible. Coordinate landing/stage order with TL.

One base snapshot, one policy compilation and one existing document lock own each write transaction. Resolved edits are classified in base coordinates. Use existing diff narrowing to remove unchanged record context; split independently representable changes at region boundaries. If a genuine indivisible edit crosses different policies, refuse atomically and ask for separate edits. No partial acceptance of a batch. At insertion boundaries, determine the containing base region explicitly; a deleted heading cannot lend a later region's permission to preceding content.

Before any disk or Shadow/round mutation: validate source policy, compare protected source/associations, validate scope/frontier/chat restrictions, convert propose-only pieces with existing marker rules, compile the candidate for structural validity, and check registration/owner/type invariants. Preserve original effective policy for surviving base text; a heading edit that re-scopes controlled content is refused. Apply-file's empty/no-edit path also validates policy before retiring/publishing a round. Read-only warnings do not mutate metadata. No new mutable policy cache or registry is added; compiled snapshots die with the transaction.

Chat ordinary shallowest list items are asks; preserve their original text and only append their answer link, or retain an existing link for follow-up. Ready markers within chat retain discussion semantics. Chat item asks remain reachable even when processing scope/frontier would otherwise exclude them. Collaboration answers are in-place; discussion questions link outward. The agent receives these routing instructions alongside processing briefs; lexical guards check item preservation and proposal authorization, not answer quality. Existing pending-marker and metadata-direct-under-propose exceptions remain intact.

## State and failure design

Actions: read, correct settings in editor, retry after fresh read, split a cross-policy edit, preview migration, confirm/apply migration, inspect document/round after an uncertain reply.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Valid scalar/comment policy | body plus resolved settings | read | none |
| Readable alias/map or unknown document presentation | body/settings plus located warning | correct settings in editor | none; original bytes retained |
| Bad heading declaration, conflicting map or processing selection | located policy refusal with valid choices | correct settings in editor | no body, Shadow or round change |
| Stale body or policy snapshot | stale refusal | retry after fresh read | no submitted edit applied |
| Agent attempts owned-source change | protected-setting refusal | correct settings in editor | no submitted edit applied |
| Indivisible cross-policy edit | boundary refusal | split a cross-policy edit | no partial batch applied |
| Operator holds document | existing locked reply | retry after fresh read when editing ends | none |
| Multi-mode legacy canonical fill | explicit choose-one refusal | correct settings in editor | no bytes changed or brief discarded |
| Migration preview valid | before hash and exact diff/candidate | confirm/apply migration | none |
| Migration conflict, dangling name or unsaved operator buffer | migration refusal with affected heading | correct settings in editor | no map/comment rewrite |
| Source moves after preview | stale migration refusal | preview migration | none |
| File publishes but round publication fails | existing applied-with-warning result | inspect document/round after an uncertain reply | decorations may be unavailable; document work retained |

Covered: data preservation, per-snapshot rules, refusal atomicity, legacy readability and targeted conversion. Not covered: authenticating human identity against arbitrary same-UID code; semantically proving an answer is correct; moving old side docs; rewriting an unrequested external ops-echo file. These boundaries must remain visible in help/tests and the TL handoff.

## Named functions and verification strategies

The following entry points own the tests; implementation tasks reference these strategies rather than reproduce case inventories. Test names may group the same function's invariant, but production names in this table must track delivered symbols.

| Function / location | Adversarial input strategy and mechanical guard |
|---|---|
| `Catalog.ResolveProcessing` / `editorial/catalog.go` | Generate exact names and all prefixes from pinned metadata plus malformed selectors; assert resolution is unique or a located refusal, and legacy brief ordering/first flags retain producer intent. |
| `Compile` / `editorial/document.go` | Fuzz bounded Markdown and isolated owned YAML with producer fixtures as seeds; compare nested inheritance to a small independent heading-stack oracle, count source walks, and require deterministic diagnostics without mutation or panics. |
| `parseOwnedSource` / `editorial/source.go` | Generate delimiter, duplicate-key and comment-association mutations around code/heading boundaries; assert only recognized authority participates and all retained raw source slices reconstruct byte exactly. |
| `FillCanonical` / `editorial/source.go` | Replay pinned before/after fixtures then fuzz additions of opaque metadata; assert idempotence, unchanged existing values/aliases, preservation of effective policy and zero candidate bytes on nonrepresentable legacy selectors. |
| `PlanMigration` / `editorial/migrate.go` | Generate heading trees, duplicate normalized names and maps; independently compare old/new routing over every original text region, assert untouched bytes/other axes survive, and any conflict produces no candidate. |
| `ValidateEdits` / `editorial/edits.go` | Generate sorted base-coordinate edit batches across region/owned-source boundaries; independently project changed ranges and assert no authority change, chat-item rewrite or scope/frontier escape, with atomic refusal. |
| `sectionPolicyHash` / `sessionsync/file.go` | Metamorphically mutate body, ancestor/document authority and irrelevant text; assert every relevant change invalidates the opaque scope token while unrelated changes preserve its intended concurrency scope. |
| `read`, `prepareWrite`, `contentStep`, `ApplyFile` / `sessionsync/` | Drive every registered/legacy/empty-edit seam with the same policy fixture; snapshot file/Shadow/round around refusals and use existing controlled lock barriers for publication/CAS order, requiring zero pre-admission effects. |
| `Round.UnmarshalJSON`, `Span.UnmarshalJSON`, `roundRecord.UnmarshalJSON`, `decodeRound`, `encodeRound` / `sessionsync/round.go`, `applyfile.go` | Mutate known member types/presence, inject duplicates at arbitrary nesting depth and opaque extensions; strict decode refuses ambiguity, and valid decode/re-encode preserves raw nested extensions and legacy omissions. Reuse `strictjson.SplitObjectPreserveNumbers` and its matching encoder rather than another duplicate scanner. |
| `round_watch.decorations`, `reconstruct.decorate`, `record.encode/decode` / `nvim/review/` | Generate mixed presentation records and their legacy projection; compare live/resume visibility to one enum oracle and retain opaque fields through serialization. |
| `parseSessionSyncCLI`, `runSessionSyncCLI` and new `runEditorialMigration` / `couchcmd/` | Parse malformed invocation streams and drive a stateful temporary registry/filesystem through preview/confirmed apply/stale/held phases; assert the only publishable bytes are the verified transform for the captured source hash. |

Strict persisted boundary: `Round.UnmarshalJSON`, `Span.UnmarshalJSON` and `roundRecord.UnmarshalJSON` split owned and unknown fields using the existing strictjson additive-object helpers. The splitter rejects duplicates recursively before typed decoding, including unknown nested objects. Typed owned decoding checks required presence separately from valid zero values and intentional legacy omissions; it never accepts null as a scalar zero. Matching encoders preserve opaque raw member values, including large number tokens. The existing round byte cap bounds input; no permissive json.Unmarshal fallback may bypass these checks. RoundArtifact's enclosing object must use the same strict split when rewritten so nested payload validation is not weakened.

## Implementation tasks

### Task 1: Pin executable contract fixtures and mode sources

Files: create `cmd/internal/editorial/testdata/contract/` and `cmd/internal/editorial/modes/`; add `catalog.go`, `catalog_test.go`, `contract_test.go`; modify `go.mod`/`go.sum` only for needed producer packages/parser dependencies.

- [x] Write the provenance test and failing Catalog.ResolveProcessing tests from the strategy table; missing pinned artifacts/catalog must fail explicitly.
- [x] Copy exact producer bytes using the recorded commit and implement the small immutable Catalog; retain producer metadata as its value owner.
- [x] Run `go test ./cmd/internal/editorial -run 'TestContract|TestCatalog'`; expect PASS. Commit the source pin and catalog with issue/trailer attribution.

### Task 2: Compile document policy and plan migration purely

Files: create `document.go`, `source.go`, `migrate.go` and matching `_test.go` files in `cmd/internal/editorial/`.

- [ ] Write failing Compile and parseOwnedSource tests using the generated/fuzz strategies above, seeded with the pinned producer fixtures.
- [ ] Use Markdown parsing/code masks rather than a competing fence regex. Parse isolated owned YAML values with YAML nodes while preserving raw source slices; legacy composite `process-instruction` syntax must remain readable even where it is not valid whole-document YAML. Reject ambiguous duplicate owned keys. Never serialize an entire frontmatter map to make a narrow change.
- [ ] Implement the compiler and byte-preserving additive fill as pure transforms. Replay the eight fill fixtures; multi-mode refusal must return no candidate bytes. Normal fill preserves legacy maps.
- [ ] Write failing PlanMigration property tests from the strategy table, then implement the constrained transform with independent before/after routing equivalence.
- [ ] Run `go test ./cmd/internal/editorial`; expect PASS. Commit the compiler, fill and migration transform.

### Task 3: Enforce base-snapshot policy across every write path

Files: create `cmd/internal/editorial/edits.go`, `edits_test.go`; modify `cmd/internal/sessionsync/{sync.go,file.go,round.go,applyfile.go}` and corresponding tests, plus `round_write_test.go`, `marker_test.go`, `race_test.go`.

- [ ] Write failing ValidateEdits preservation tests from the strategy table across all presentation modes, with operator editor changes observed through a fresh snapshot as the positive control.
- [ ] Write the sectionPolicyHash metamorphic tests before changing its implementation; include the unchanged-body/changing-document-policy stale-write regression.
- [ ] Add the resolved read payload and scoped digest. Validate before Shadow acknowledgement. Route whole/section/frontmatter write, registered/legacy apply and standalone ApplyFile through the same policy guard, including empty edits. Retain current owner/type/lock/CAS behavior.
- [ ] Replay a pinned mixed-region acceptance fixture through every write seam and the controlled transaction strategy above; assert presentation, routing instructions and refusal atomicity against the compiled base policy.
- [ ] Run `go test ./cmd/internal/editorial ./cmd/internal/sessionsync ./cmd/internal/reviewapply`; expect PASS. Use controlled concurrent writers to verify one winner and a stale loser; never sleep to approximate the lock window. Commit the shared enforcement.

### Task 4: Carry presentation through live and resumed review consumers

Files: modify `sessionsync/round.go`, `applyfile.go` and tests; modify `nvim/review/{round_watch.lua,reconstruct.lua,record.lua}` only where required and their existing tests. Coordinate the exact schema and fixtures with Parley #419.

- [ ] Write failing codec and live/resume consumer tests from the strategy table, including recursive duplicate injection and the legacy omission contract.
- [ ] Implement base-policy span attribution and round-record propagation. Pair's current Lua record codec already keeps arbitrary table fields; retain that behavior. Pair's live/resume decoration paths currently highlight every change and must honor effective span presentation consistently.
- [ ] Run focused Go round/apply tests and `bash tests/review-applied-round-test.sh` plus the existing Lua round-watch/record/reconstruct test entry points; record exact commands/results in the issue Log. Obtain #419 consumer fixture agreement before claiming end-to-end mixed rendering. Commit schema and local consumers together.

### Task 5: Expose operator migration and update actual agent interfaces

Files: modify `cmd/internal/couchcmd/session_sync.go`, `run.go`, existing `session_sync*_test.go`; add `session_sync_editorial_test.go`; update `cmd/internal/couchcmd/skills/couch/SKILL.md`, `atlas/couch-agentic-documents.md`, `atlas/review-workbench.md` and `atlas/index.md`.

- [ ] Write failing parser/runEditorialMigration tests using the stateful registry/preview strategy above; no generic body payload is accepted by the migration verb.
- [ ] Add `migrate-editorial` preview and a narrowly confirmed apply form agreed with TL. Reuse registry/flock/guard/replaceShared rather than an unlock-write-relock sequence. No migration runs during ordinary read/review/fill, and no external live file is changed by this implementation task.
- [ ] Update agent guidance to read resolved policy on every round, use briefs/instruction and section routing, preserve owned comments, and create new side docs with three defaults under the month layout. Keep old flat/qa links valid; test both through existing side-doc authorization rules. No parent legacy brief is copied to a new side doc.
- [ ] Qualify agent-facing guidance on the disposable Couch harness for Claude and Codex (Qoder when credits permit), with the same mixed-region contract fixture; never use the live fleet as a test fixture. Record deterministic contract checks separately from semantic live observations.
- [ ] Run focused Couch CLI/session-sync/native tests and document results. Commit CLI, guidance and atlas updates.

### Task 6: Growth, close and operator acceptance

- [ ] Add counted compiler/classifier tests at N and 2N headings, legacy names and sorted edits. Growth drivers are input bytes B, headings H, map entries M and edits E: one Markdown/source walk, normalized-name index O(H+M), inheritance stack O(H), interval classification O(H+E) after edit sorting. Reuse a single catalog. Do not rescan the whole document per heading or edit. Count owned scans/comparisons, not wall time; distinguish underlying parser cost from our traversal counts. Respect existing 4 MiB document / 16 MiB round limits.
- [ ] Run targeted Go/Lua tests, `make test-affected-inventory` and audit every changed reader before any per-file hash repair. Do not bulk refresh inventory hashes.
- [ ] Push the implementation head through SDLC and run the declared recorded affected check when eligible, otherwise the recorded suite. Full final integrated suite remains required by landing. Log exact head, result and any environmental failure without converting it into passing evidence.
- [ ] Run `sdlc close --issue 823 --verified '<measured evidence>'` after tests/atlas; let it own fresh-context review and actuals. Fix or counter-measure findings through the supported gate path.
- [ ] With TL staging approval and #419 coordination, stage/build for a short real operator check: edit section modes, submit a mixed chat/collaboration round, see direct/highlight/propose behavior, observe a protected-agent-edit refusal, then preview/apply a disposable legacy-map migration. Operator edits must remain allowed. Do not migrate the operator's real document without its explicit selection/approval.
- [ ] After acceptance and required checks, run `sdlc merge --yes`, report landed PR, clear any owned stage, and stop. TL owns the project file.

## Revisions

### 2026-10-23 — Initial author plan from landed #562

Pinned the producer fixtures rather than interpreting alias semantics independently. Made #419's per-span consumer upgrade an explicit dependency. Code inventory found no Couch side-doc creator, no semantic review orchestrator and no authenticated human identity in session-sync; the proposed design keeps those boundaries explicit. Existing section CAS omits document policy, so scoped authority is included in the transaction design before implementation.

### 2026-10-23 — Plan-quality round 1, PQ-1 and PQ-2

Replaced prose test-case inventories with named production-function/adversarial-strategy contracts, retaining the producer fixtures as executable acceptance anchors (PQ-1). Required recursive duplicate-key rejection at every changed persisted round/span/record boundary and reused the existing strictjson additive codec rather than creating a second scanner (PQ-2, ARCH-DRY/ARCH-EVOLVE). Known-field presence/null/type validation remains distinct from opaque extension preservation. These refine the approved design without changing the product boundaries.

### 2026-10-23 — Task 1 execution checkpoint

Full-flow admission passed with estimate 4.274; estimate-quality was INFO, flagging optimistic integration/verification allowances. Task 1 catalog and all pinned artifacts are implemented and pass focused tests; independent spec review verified all 39 Git objects and a subsequent quality review found no Important issues. Located the actual emitted Couch skill and atlas pages above. Parser dependencies are the exact Goldmark/YAML versions used by the producer; the old ariadne module already exports the required collaboration vocabulary, so no unrelated base-module upgrade was needed.
