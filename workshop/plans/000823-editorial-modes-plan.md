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
| CLI/help contract | `cmd/internal/couchcmd/run.go`, emitted Couch skill source located during implementation inventory | modified | agent-facing settings, refusal recovery and operator invocation |
| Review consumers | `nvim/review/round_watch.lua`, `reconstruct.lua`, `record.lua` | modified | live and resume decorations/record serialization |
| Parley round consumer | parley.nvim #419 | external dependency | additive span metadata and mixed-section rendering |

No external service is added. Transaction tests use real temporary files/registry state and controlled barriers; existing disposable Couch harness supplies native agent qualification. No call-count mocks substitute for lock/CAS and publication tests.

## Public data and transaction contract

Add a resolved editorial payload to `ReadResult`: document defaults, ordered byte ranges with effective axes and processing metadata, source hash and warnings. Keep existing body/file/section/hash fields. Section reads report relevant regions in document coordinates explicitly; never imply offsets are relative to the returned section. CLI help explains coordinate origin. Read validates before advancing Shadow; malformed policy does not acknowledge a new baseline accidentally.

Proposed round wire shared with Parley #419:

```json
{"presentation":"highlight","spans":[{"start":[8,0],"end":[8,3],"old":"old","new":"new","presentation":"direct"}]}
```

`Span.presentation` is optional for legacy artifacts, and is `direct|highlight|propose` when present. New rounds stamp each span from base policy. Keep `Round.presentation` as the document default; no invented `mixed` value. Carry the same optional field into commit `review-records` and resume reconstruction. Direct spans receive no change decoration, highlight spans do, and proposal markers retain existing proposal rendering. Explicit known invalid/null values refuse; unrelated extension fields survive rewrites. Mixed-mode UI acceptance requires #419's consumer upgrade; an older Parley ignoring this field cannot be claimed compatible. Coordinate landing/stage order with TL.

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

## Implementation tasks

### Task 1: Pin executable contract fixtures and mode sources

Files: create `cmd/internal/editorial/testdata/contract/` and `cmd/internal/editorial/modes/`; add `catalog.go`, `catalog_test.go`, `contract_test.go`; modify `go.mod`/`go.sum` only for needed producer packages/parser dependencies.

- [ ] Write a provenance test verifying the pinned source bytes/digests, all eight fill cases, heading-chat, and mode metadata/brief resolution; it must fail before copied artifacts/catalog exist.
- [ ] Copy exact producer bytes using the recorded commit and implement the small immutable Catalog. Add exact/prefix/ambiguous/unknown mode tests, ordered legacy brief tests and first-mode flags.
- [ ] Run `go test ./cmd/internal/editorial -run 'TestContract|TestCatalog'`; expect PASS. Commit the source pin and catalog with issue/trailer attribution.

### Task 2: Compile document policy and plan migration purely

Files: create `document.go`, `source.go`, `migrate.go` and matching `_test.go` files in `cmd/internal/editorial/`.

- [ ] Write table tests first: defaults, canonical presence, independent alias components, flow/block maps, all ATX levels, fences, nearest per-axis inheritance, repeated heading names, malformed recognized comments, detached/ordinary comments and duplicate adjacent directives.
- [ ] Use Markdown parsing/code masks rather than a competing fence regex. Parse isolated owned YAML values with YAML nodes while preserving raw source slices; legacy composite `process-instruction` syntax must remain readable even where it is not valid whole-document YAML. Reject ambiguous duplicate owned keys. Never serialize an entire frontmatter map to make a narrow change.
- [ ] Implement the compiler and byte-preserving additive fill as pure transforms. Replay the eight fill fixtures; multi-mode refusal must return no candidate bytes. Normal fill preserves legacy maps.
- [ ] Add migration tests for every duplicate match, child inheritance, existing other-axis comments, identical/conflicting existing collaboration assignments, unmatched names, CRLF/final-newline preservation and unchanged unknown metadata. Prove before/after effective routing equality for mapped source ranges.
- [ ] Run `go test ./cmd/internal/editorial`; expect PASS. Commit the compiler, fill and migration transform.

### Task 3: Enforce base-snapshot policy across every write path

Files: create `cmd/internal/editorial/edits.go`, `edits_test.go`; modify `cmd/internal/sessionsync/{sync.go,file.go,round.go,applyfile.go}` and corresponding tests, plus `round_write_test.go`, `marker_test.go`, `race_test.go`.

- [ ] Start with failing tests for adding/removing/changing a heading comment under direct/highlight/propose; operator editor changes followed by fresh read must pass. Cover canonical/alias settings, byte-equivalent-value rewrites and heading re-association attempts.
- [ ] Add stale-section tests: keep body fixed while changing document defaults, parent heading policy or a legacy map; all old scoped writes must refuse. Confirm unrelated body edits outside scope preserve existing concurrency behavior where policy is unchanged.
- [ ] Add the resolved read payload and scoped digest. Validate before Shadow acknowledgement. Route whole/section/frontmatter write, registered/legacy apply and standalone ApplyFile through the same policy guard, including empty edits. Retain current owner/type/lock/CAS behavior.
- [ ] Add one mixed-round fixture with chat, collaboration, discussion and processing overrides, including chat asks below a frontier, pending markers and a narrow ready side-doc exception. Test one direct region, one highlight region and one propose region in a single atomic write. Assert cross-boundary refusal leaves file, Shadow and round unchanged.
- [ ] Run `go test ./cmd/internal/editorial ./cmd/internal/sessionsync ./cmd/internal/reviewapply`; expect PASS. Use controlled concurrent writers to verify one winner and a stale loser; never sleep to approximate the lock window. Commit the shared enforcement.

### Task 4: Carry presentation through live and resumed review consumers

Files: modify `sessionsync/round.go`, `applyfile.go` and tests; modify `nvim/review/{round_watch.lua,reconstruct.lua,record.lua}` only where required and their existing tests. Coordinate the exact schema and fixtures with Parley #419.

- [ ] Write mixed direct/highlight/propose live/resume cases, legacy absent-span fallback and invalid known-field tests. Round-trip a fixture with unknown round/span/record fields without losing them.
- [ ] Implement base-policy span attribution and round-record propagation. Pair's current Lua record codec already keeps arbitrary table fields; retain that behavior. Pair's live/resume decoration paths currently highlight every change and must honor effective span presentation consistently.
- [ ] Run focused Go round/apply tests and the existing Lua review test targets discovered in the Makefile; record exact commands/results in the issue Log. Obtain #419 consumer fixture agreement before claiming end-to-end mixed rendering. Commit schema and local consumers together.

### Task 5: Expose operator migration and update actual agent interfaces

Files: modify `cmd/internal/couchcmd/session_sync.go`, `run.go`, existing `session_sync*_test.go`; add `session_sync_editorial_test.go`; update the emitted Couch skill at its canonical source, and `atlas/`'s existing session-sync/review pages and index.

- [ ] Write CLI parser/transaction tests for preview, confirmed transform, stale preview, conflicting comments, held operator buffer, symlink/canonical identity, retained permissions, and refusal before publication. Confirm no generic body payload is accepted by the migration verb.
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
