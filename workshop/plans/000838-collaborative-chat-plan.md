# Single collaborative mode and chat sections — implementation plan

> **For agentic workers:** Consult AGENTS.md §3 for execution strategy. This is one atomic full-flow change, with one issue-close review; plain task checkboxes are not milestone boundaries.

**Goal:** Couch follows the operator's single collaborative editing contract, recognizes chat sections, preserves legacy chat documents, and highlights new agent edits.

**Architecture:** Reuse ariadne's pure `sidedoc.ChatSections` as the authority for tags, legacy aliases and notices. Reduce Pair's editorial compiler to chat-region/source protection and existing marker/item guards. Keep document transactions and historical round decoding in their existing owners.

**Tech stack:** Go, Goldmark, existing session-sync file transactions and portable test rig, shared disposable testcouch harness.

## Status and sources

Draft for ops:0 approval; no implementation authorized yet. #838 is based on fresh main after #811 landed, never on its unlanded branch. No estimate until change-code's plan-quality gate.

Sources read: ariadne#575; ops `session.side/2026-10-23-03-mode-simplification.md`; committed #575 candidate `6ae80e4e69aacb72d8745e479791ada7c4454c17`, especially `construct/local/universal-review/{collaboration-modes.md,round.md,export.json,examples/}` and `pkg/sidedoc/chat.go`. The candidate is planning evidence, not an installed dependency or landing acceptance.

ariadne:2 confirmed the shared API and fixtures final for #575 in receipt c8bda0b0-3e7f-43f4-9301-6a5521d89dc6. The owner will fix two notice defects and quadratic heading work in ariadne#576. Proposed gate: implementation starts only after TL approval and #575 lands; final source pin, performance qualification and close require #576 landed too. No local scanner fork or copied candidate dependency. Confirm the actual landed source at each gate.

The plan writes no legacy mode frontmatter; that overrides the old plan-template mode header in accordance with the operator's pivot.

## Product contract and boundaries

- Direct operator edits are content. A ready marker is classified by the agent: question/unclear command -> linked side doc; clear command -> action/edit; uncertain -> question. Shared round instructions own this semantic choice, not a punctuation classifier in Couch.
- Scope: current/preceding paragraph or exact anchor by default; widen only when asked; a marker before the first heading addresses the document unless narrowed. The agent follows shared instructions and fixtures; Couch enforces source ownership and marker/chat structural guards.
- New edits are clean text with highlight metadata. Old persisted direct/propose rounds remain readable/renderable; nothing rewrites their meaning.
- The shared scanner owns exact tag grammar, heading extents, code exclusions, legacy names/defaults/comments, and notices. Convert its whole-document 1-based lines to original byte offsets once.
- Tagged chat takes numbered shallowest-indent items. Legacy chat additionally takes `-` and `*`. Overlap is one chat scope; preserve the legacy item grammar where legacy authority applies. A marker inside a chat section remains an ordinary marker ask.
- Protect real tags and their attachment against every agent write seam, including insertion, removal, rename/reparenting and code-fence tricks. Misplaced tags are ordinary text and get the shared misplaced-tag notice.
- Ordinary rounds never add, remove or rewrite legacy settings. Malformed/unknown legacy values produce shared nonfatal notices. Other ownership, stale-hash, invalid-edit and lock refusals remain effective.
- Return the shared notice sequence verbatim once in the round's read snapshot; the CLI/skill reports it once per round, without a persistent deduplication ledger.
- Reuse existing side-doc/index handling; no new side-doc allocator, decision engine, transport or round-submission diff. #839 owns incremental submission separately.

## Alternatives

1. **Recommended:** shared scanner + small Pair byte-range adapter; delete retired policy branches. One grammar and notice owner, minimal compatibility surface.
2. Keep all three axes and force their defaults. Less initial deletion, but preserves misleading branches and a second legacy resolver; fails the simplification goal.
3. Reimplement the tag/legacy scanner in Pair. Avoids the upstream dependency wait but duplicates the contract and its known cost defects; reject under ARCH-DRY.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| `DocumentPolicy`, `Effective`, `Region`, `Heading` | `cmd/internal/editorial/document.go` | modified: one collaborative policy plus chat provenance, no processing brief resolution |
| `OwnedSource`, `Range`, `Diagnostic` | `cmd/internal/editorial/source.go` | modified: source-preserving declarations/tags and compatibility diagnostics |
| `Compile`, `CompileStats` | `cmd/internal/editorial/document.go` | modified: adapt one shared ChatScan per source snapshot |
| `ValidateEdits`, `EditPolicy`, chat-item guards | `cmd/internal/editorial/edits.go` | modified: protect tag authority, correct item grammar, always classify new edits highlight |
| `Migration`, `PlanMigration` | `cmd/internal/editorial/migrate.go` | modified: explicit map-to-tag transform with routing proof |
| `Catalog`, `Mode`, `Scope`, axis resolvers, `FillCanonical` | `cmd/internal/editorial/catalog.go`, `migrate.go` | deleted: no runtime catalog/axis selection or automatic mode-key fill |

DocumentPolicy belongs 1:1 to one immutable source snapshot. Regions refer to its original source coordinates. The shared ChatScan is ephemeral and consumed by that policy; no additional stored mode state. Chat provenance is needed only to preserve numbered-only versus legacy bullet item grammar. New parsing responsibilities stay with `sidedoc`; Pair's adapter owns byte ranges and edit authorization. Existing marker/list/source facts and monotone joins are reused.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| `compileEditorial`, `file.editorialPolicy`, `sectionPolicyHash` | `cmd/internal/sessionsync/editorial.go` | modified | one transaction snapshot and freshness token |
| `Sync.read` and write/apply paths | `cmd/internal/sessionsync/sync.go`, `applyfile.go` | modified | existing document/registration ownership and acknowledgement |
| round construction/resolution | `cmd/internal/sessionsync/round.go`, `editorial_codec.go` | modified | new highlight rounds and old additive round codec |
| `Sync.MigrateEditorial` | `cmd/internal/sessionsync/editorial_migration.go` | modified | existing operator preview/hash-confirmed transaction |
| CLI read/migration + skill | `cmd/internal/couchcmd/session_sync.go`, `skills/couch/SKILL.md` | modified | human/JSON output and agent instructions |

Reuse the stateful sessionsync temp-folder rig: document, shadow, round, registry, receipt/state and operator lock. Existing testcouch provides disposable workers and isolated store for CLI/native conformance. No live fleet writes in automated tests.

## Actions, states and preservation

Actions: submit/read, apply/write, edit/resubmit, preview migration, confirm exact preview, retry after inspection.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| new collaborative or chat snapshot | resolved policy; tag/misplaced/legacy notices as applicable | apply/write | none |
| malformed retired value | nonfatal notice; shared legacy fallback | submit/read | no source rewrite |
| stale content or changed chat attachment | stale/ownership refusal, no acknowledgement/publication | edit/resubmit | unpublished candidate only |
| foreign/operator lock | existing locked refusal | submit after operator unlocks | unpublished candidate only |
| tag tampering or chat-item rewrite | located refusal, whole batch rejected | edit/resubmit | unpublished candidate only |
| safe map conversion | preview and exact source hash | confirm exact preview | old settings replaced only by requested migration |
| map conversion would change eligible asks | actionable migration refusal; source unchanged | keep legacy and edit/resubmit intentionally | none |
| publication uncertain during confirmed migration | existing migration-uncertain result | inspect document, then preview again | old round may already be cleared; no claim that publication failed |
| old persisted direct/propose round | old round still readable | normal resolve/apply using existing contract | none from decoding or rewriting |

One existing per-document transaction lock owns mutations; no observation-based exclusion or new lock. Fresh source/hash and registered-owner checks still precede effects. Sequence tests cover operator changes after read, stale confirmation, lock arrival, IO failure after round retirement and duplicate/no-op migration. No spawned worker or new background lifetime.

### Migration safety decision for approval

Map conversion replaces only the requested map field and inserts tags above the selected real headings, preserving unrelated bytes. Compare before/after heading extents **and eligible chat-item projections**, not just effective section labels. Refuse when tag-only numbered grammar would drop legacy bullet asks, or `default: chat` includes preamble asks with no heading that can carry a tag. Keep legacy reading fully functional; do not silently renumber the operator's text or invent a persisted compatibility bit. Unmatched names are a nonfatal normal-read notice but may prevent a lossless conversion. Confirm this bounded refusal with the TL before code.

## Operating envelope and architectural checks

- **ARCH-DRY / ARCH-PURE:** shared ChatSections/notice owner; pure byte adapter/validator/migration; IO only in existing transaction layer. Do not call `ChatScan.InChat` once per item (it scans all sections); normalize intervals once and sweep them with source items.
- **ARCH-PURPOSE:** sweep all read/write/apply/frontmatter/migration/CLI/embedded-skill consumers, not only Compile. Legacy ops/brain docs and migration remain acceptance requirements.
- **ARCH-MOCK:** existing portable file rig and testcouch cover real public seams; live Claude/Codex conformance on synthetic docs, Qoder when credits permit.
- **ARCH-CONSTRAINTS:** path is per document snapshot/agent round, not per terminal byte. Existing registered-doc bound is 4 MiB; preserve it and existing registry growth refusals. Adapter work targets O(B+H+C+I+E+M) after shared scan, with one call per compiled snapshot, a constant two snapshots for base/candidate validation. B bytes, H headings, C chat intervals, I items, E edits, M markers. New IO/threads/timeouts: zero. Count operations for 8/16 and 128/256 headings/items/edits; upstream #576 owns its own source/heading count bound. #575 alone has O(B*H+H²) scanner cost; do not claim end-to-end linear work or close on that version. Keep existing session-sized timing budgets, remeasure with generated large documents under normal machine load; counts, not a timeout increase, defend growth.
- **ARCH-SECURE:** operator tags/settings are authority; candidate text cannot grant permission, forge a ready marker or change attachment. Unknown retired values are data, never an instruction or refusal trigger. Refusals preserve shadow/receipt/round except the already-documented uncertain migration path.
- **ARCH-ORDER:** existing document transaction and exact source hash bind state/effects; no new durable state machine. Test stale reads, foreign locks, repeated confirmation and uncertain publication through the current production seams.
- **ARCH-FUNERAL:** policy/notice/line-index values die with the transaction. No new durable record family; existing round retirement, session-sync state and side-doc lifecycle stay with their owners. Pinned test fixtures/provenance live and archive with this issue under existing repository policy.
- **ARCH-EVOLVE:** retain strict known-field/duplicate checks and nested unknown JSON preservation in old persisted rounds. New round writers always emit highlight. Never reinterpret an old direct/propose span under the new policy. Read.Editorial is a fresh projection, not a migration of saved round records.

## Execution tasks — after approval and dependencies

### 1. Pin the landed shared contract

Files: `cmd/internal/editorial/contract_test.go`, `testdata/contract838/`; `workshop/evidence/000838/contract-provenance.json`; affected reader certificates only after auditing actual reads.

- [ ] Verify #575 landed; refresh the ordinary dependency through weave. Verify #576 before final fixture pin/performance evidence. Record actual Git revision and exported file digests; no candidate copying into the dependency.
- [ ] Add manifest-driven tests verifying exact spec/round/examples bytes. Replay all applicable exported fixtures, especially chat-tag, misplaced-tag, legacy scalar/map/comment, ignored/malformed keys, no-asks, command/question and scope examples. Keep #823's historical evidence intact, removing only obsolete production-catalog coupling from its test.
- [ ] Write `TestCompileSharedChatContract` and `TestCompileRetiredValuesNeverRefuse`; run `go test ./cmd/internal/editorial -run 'TestCompileSharedChatContract|TestCompileRetiredValuesNeverRefuse'` and record RED on old behavior before implementation.

### 2. Simplify compilation and edit guards

Files: `editorial/{document,source,edits,markdown,catalog}.go` and colocated tests; remove runtime mode embeds when no consumer remains.

- [ ] Adapt ChatScan sections/notices to byte intervals and protected source ownership, retaining Goldmark source facts for edit validation. One shared scan per Compile; include tag and owning heading/ancestor in section freshness tokens.
- [ ] Delete retired processing/presentation/override resolution, domain/frontier enforcement and fill paths. Keep legacy declarations byte protected and nonfatal. Compile unknown/malformed settings using shared fallback without discarding unrelated ownership metadata.
- [ ] Add tag insertion/removal/detachment/reparenting, fake headings/code masks, numbered-versus-legacy-bullet, overlap, tabs, nested sections, `→ done:` and actual-link tests. Preserve ready-marker referral behavior inside chat and pending-marker identity guards.
- [ ] Pin adapter scan counts at N/2N and fuzz generated base/candidate edits with an independent interval/item oracle. Run `go test ./cmd/internal/editorial` and `go test -race ./cmd/internal/editorial`; expected PASS including existing item-integrity regressions.
- [ ] Commit the coherent pure-core change with issue, Agent and Co-Authored-By trailers.

### 3. Wire rounds, notices and operator migration

Files: `sessionsync/{editorial,editorial_migration,editorial_codec,round,applyfile,sync}.go`, colocated tests; `couchcmd/session_sync.go` and `session_sync_editorial_test.go`.

- [ ] Write failing public-seam tests for whole/section/frontmatter Write, Apply and ApplyFile: tag protection, stale attachment tokens, highlight spans, nonfatal diagnostics and unchanged shadow on refusal. Include malformed frontmatter/retired values beside valid registered identity.
- [ ] Remove the runtime catalog dependency. Carry the shared notice sequence once in Read.Editorial; CLI renders the sequence without duplicated wrapper notices. Preserve old round codec/render semantics and unknown-field round trips.
- [ ] Write failing map-to-tag migration tests including duplicate heading names, ancestor/descendant overlap, empty/default chat, legacy bullets, CRLF and idempotence. Implement only transformations proved to preserve chat items; test preview/no-op writes zero files, stale confirmation, lock refusal, size refusal and uncertain publication recovery.
- [ ] Run `go test ./cmd/internal/sessionsync ./cmd/internal/couchcmd -run 'Editorial|Round|SessionSync'`, then focused race tests. Commit the integration with the same trailers.

### 4. Agent contract, acceptance and close

Files: `cmd/internal/couchcmd/skills/couch/SKILL.md`, `cmd/couch/editorial_acceptance_test.go`; `atlas/couch-agentic-documents.md`, `atlas/couch.md`, `atlas/index.md`, README command/help references where affected.

- [ ] Update all current guidance to single collaborative mode, shared question/command/scope rules, highlight and tag/legacy notices; remove instructions to fill mode keys or select processing axes. Preserve old fixture/evidence provenance, not stale product instructions.
- [ ] Read-only audit real ops and brain chat/session docs: resolve actual paths, retain local digests and routing/notice summaries, never publish their content or modify them. Prove brain's existing chat document remains accepted unchanged.
- [ ] Replay pinned fixtures through the public CLI in disposable testcouch. Run bounded native Claude and Codex flows on synthetic documents covering command, question, chat, ordinary marker in chat and new highlights; Qoder if available. No live fleet editing. Failures are classified before any rerun; use registered-flake recovery only when evidenced.
- [ ] Propose a short operator stage checklist through the TL for the real editor experience: tagged section, command edit highlighted, question side doc, legacy doc still readable. Touch :0 only after stage authorization and clear only our lease afterward.
- [ ] Run `make test-affected-inventory`; audit each named reader input before its exact repair command, never bulk-refresh. Run targeted packages, push, record `sdlc local-check run --name affected`, retain runner to completion.
- [ ] Run the single full-flow `sdlc close --issue 838 --verified 'actual evidence'` after inventory and required acceptance, using its mandatory review. Address findings by class, record lessons, then PR/merge through normal gates. No separate duplicate review at the boundary. Report landed PR and verified Git hashes to the hub.

## Approval / next action

Send this committed draft to ops:0. Approval requested for shared-scanner architecture, the lossless-only migration refusal, and #576 before final qualification. Await TL approval and dependency landing before code/change-code. There is no pending #811 runner or stage.

## Revisions

### 2026-10-23 14:08 — TL approval

Authenticated ops:0 receipt a7ad7cf6-257b-419f-b538-e40f2b0a457b approves this
design and both sequencing decisions. Lossless-only migration refusals must
name the affected section when numbered tags would drop bullet or preamble
asks. Legacy reads continue unchanged; no renumbering. Natural-language
question/command/scope choices remain in shared agent instructions. The end
operator stage check is the project's live acceptance, approved in principle;
coordinate its timing and lease with the hub.

The Draft/Approval text above records the proposal before approval. Current
state: APPROVED, awaiting ariadne#575 landing before change-code. #576 gates
final fixture pin, performance qualification and close.
