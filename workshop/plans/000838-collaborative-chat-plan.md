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
- Tagged chat takes numbered shallowest-indent items; legacy chat additionally takes `-` and `*`. The innermost shared section governs each line; its Legacy flag chooses item grammar independently of Source. Same-heading tag plus legacy retains Legacy=true. Whole-document legacy governs the preamble and suppresses emitted inner tag sections. A marker inside a chat section remains an ordinary marker ask.
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

## Shared provenance contract — PQ-1

The correction is owned upstream by ariadne#576. Its explicit chosen API is:

```go
type ChatSection struct {
    Heading, End int
    Source string
    Legacy bool
}
type ChatScan struct {
    Sections []ChatSection
    Tags, Misplaced []int
    Notices []string
}
func ChatSections(doc string) ChatScan
func (s ChatScan) Innermost(line int) (ChatSection, bool)
```

This shape was verified in committed upstream candidate `9dbb69c87a7e5626753e29280bdb1eacb2bb96bd`; receipts fba9ba42-e53f-41f4-8983-b65306a5e152 and 13f616c6-968e-4f6e-841b-8e104a62d140 establish its owner and #576 landing boundary. No candidate source is installed.

| Pair consumer | Shared authority fact |
|---|---|
| region routing | laminar section Heading/End; innermost containing section |
| item grammar | governing section Legacy flag, never infer it from Source=tag |
| protected tags/attachments | every line in Tags, including tags suppressed as sections by whole-document legacy; owning heading is tag+1 |
| migration proof | before/after routing and eligible-item projection from the same shared facts |

Source retains the visible owner (`tag` wins on its heading), while Legacy independently preserves mixed authority. Tags and Misplaced never share a line. The shared randomized 500-document oracle checks laminar/innermost semantics against smallest extents. Pair's generated interval/item oracle must separately reject lost legacy grammar and unauthorized tag detachment; use ordered interval/line joins instead of calling the linear Innermost search for every item.

**Dependency gate:** #576's landed provenance API is required before implementing/qualifying `Compile`, tag ownership and `selectChatItems`; #575 alone cannot supply it. Other code entry still requires the approved plan-quality gate and estimates. Never compensate with a Pair legacy parser or tag recognizer. The owner is also extracting shared ChatItems from the existing TL splitter; consume that seam only after its signature/coordinate contract is explicit and the version is landed, recording the final API before editing the item adapter.

## Function-scoped verification strategy

| Production function | Adversarial input class -> mechanical guard |
|---|---|
| `Compile` | Arbitrary Markdown and legacy values -> shared fixture replay plus fuzz, independently compare byte-region coverage with scanner facts, one scanner call per snapshot and N/2N adapter counters. |
| `ValidateEdits` | Generated source/candidate structural changes -> independent authority/unchanged-surviving-byte oracle; sorted monotone joins; reject the complete batch before effects. |
| `selectChatItems` | Generated mixed-authority nested lists -> independent eligible-item oracle with numbered-only tags and legacy bullets, including overlapping authorities; N/2N join counts. |
| `PlanMigration` | Generated legacy maps and headings -> unchanged eligible-item/heading projection, untouched unrelated bytes, deterministic section-naming refusal and idempotence. |
| `sectionPolicyHash` | Operator changes policy attachment between read and write -> stale token refusal through actual transaction APIs; unrelated prose does not spuriously invalidate it. |
| `Sync.read`, write/Apply/ApplyFile | Adversarial ownership and malformed retired metadata -> stateful filesystem rig proves nonfatal shared notices and no acknowledgement or partial write on refusal. |
| `BuildEdit`, `contentStep`, `decodeRound`, Span/Round JSON codecs | Old/extended saved records and new writes -> golden historical rendering plus unknown-member round trips; new rounds highlight and old meanings stay unchanged. |
| `Sync.MigrateEditorial` | Interleaved source changes, publication/retirement faults and duplicate confirmation -> stateful sequence tests assert exact-source authority and distinguish uncertain publication from absence. |

Fixture examples remain source data rather than another prose case inventory. Commands in the execution tasks run these strategies; the native harness qualifies semantic instruction-following separately.

## Execution tasks — after approval and dependencies

### 1. Pin the landed shared contract

Files: `cmd/internal/editorial/contract_test.go`, `testdata/contract838/`; `workshop/evidence/000838/contract-provenance.json`; affected reader certificates only after auditing actual reads.

- [x] Verify #575 landed; refresh the ordinary dependency through weave. Verify #576 before final fixture pin/performance evidence. Record actual Git revision and exported file digests; no candidate copying into the dependency.
- [x] Add manifest-driven tests verifying exact spec/round/examples bytes. Replay all applicable exported fixtures, especially chat-tag, misplaced-tag, legacy scalar/map/comment, ignored/malformed keys, no-asks, command/question and scope examples. Keep #823's historical evidence intact, removing only obsolete production-catalog coupling from its test.
- [x] Write `TestCompileSharedChatContract` and `TestCompileRetiredValuesNeverRefuse`; run `go test ./cmd/internal/editorial -run 'TestCompileSharedChatContract|TestCompileRetiredValuesNeverRefuse'` and record RED on old behavior before implementation.

### 2. Simplify compilation and edit guards

Files: `editorial/{document,source,edits,markdown,catalog}.go` and colocated tests; remove runtime mode embeds when no consumer remains.

- [x] Adapt ChatScan sections/notices to byte intervals and protected source ownership, retaining Goldmark source facts for edit validation. One shared scan per Compile; include tag and owning heading/ancestor in section freshness tokens.
- [x] Delete retired processing/presentation/override resolution, domain/frontier enforcement and fill paths. Keep legacy declarations byte protected and nonfatal. Compile unknown/malformed settings using shared fallback without discarding unrelated ownership metadata.
- [x] Implement the function-scoped strategies below for authority preservation and chat grammar; preserve existing ready-referral and pending-marker guards.
- [ ] Pin adapter scan counts at N/2N and fuzz generated base/candidate edits with an independent interval/item oracle. Run `go test ./cmd/internal/editorial` and `go test -race ./cmd/internal/editorial`; expected PASS including existing item-integrity regressions.
- [x] Commit the coherent pure-core change with issue, Agent and Co-Authored-By trailers.

### 3. Wire rounds, notices and operator migration

Files: `sessionsync/{editorial,editorial_migration,editorial_codec,round,applyfile,sync}.go`, colocated tests; `couchcmd/session_sync.go` and `session_sync_editorial_test.go`.

- [ ] Write failing public-seam tests through the stateful file rig for each write API; assert the common source/authority invariant and unchanged shadow on every rejected transaction.
- [x] Remove the runtime catalog dependency. Carry the shared notice sequence once in Read.Editorial; CLI renders the sequence without duplicated wrapper notices. Preserve old round codec/render semantics and unknown-field round trips.
- [x] Exercise PlanMigration with generated legacy documents and the independent chat-item projection oracle; test the existing confirmed transaction as event sequences. Implement only transformations proved to preserve chat items.
- [ ] Run `go test ./cmd/internal/sessionsync ./cmd/internal/couchcmd -run 'Editorial|Round|SessionSync'`, then focused race tests. Commit the integration with the same trailers.

### 4. Agent contract, acceptance and close

Files: `cmd/internal/couchcmd/skills/couch/SKILL.md`, `cmd/couch/editorial_acceptance_test.go`; `atlas/couch-agentic-documents.md`, `atlas/couch.md`, `atlas/index.md`, README command/help references where affected.

- [x] Update all current guidance to single collaborative mode, shared question/command/scope rules, highlight and tag/legacy notices; remove instructions to fill mode keys or select processing axes. Preserve old fixture/evidence provenance, not stale product instructions.
- [x] Read-only audit real ops and brain chat/session docs: resolve actual paths, retain local digests and routing/notice summaries, never publish their content or modify them. Prove brain's existing chat document remains accepted unchanged.
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

### 2026-10-23 — PQ-2 function-scoped strategies

The first plan-quality review identified prose case inventories. Replaced those
inventories with one adversarial-input/mechanical-guard strategy per production
function, keeping executable fixture replay, fuzz oracles, count checks and
public transaction tests. This changes test description, not approved behavior.

Dependency inspection also confirms Go imports resolve through the pinned
Ariadne module in go.mod, not through the refreshed substrate checkout. Task 1
therefore includes updating go.mod/go.sum to the verified landed #575 revision
(with the #576 refresh for qualification), using an ordinary versioned module
requirement, never a local replace or copied candidate source.

### 2026-10-23 — PQ-1 shared authority projection

The plan gate found Source alone loses independent legacy grammar when a tag
and legacy chat declaration share a heading. The same authority class also
includes valid tags suppressed as sections by whole-document legacy. Upstream
#576 now explicitly supplies Legacy and all valid Tags, plus laminar/innermost
semantics. The consumer table above sweeps routing, grammar, source ownership
and migration proof. #576 must land before those core adapters can use the new
API; the approved #575-only code-entry threshold cannot manufacture missing
provenance. TL ccfe62c7-130f-4095-944f-e13cfa85b3d8 confirmed upstream
ownership and rerunning the gate once the API is explicit.

### 2026-10-23 14:36 — explicit shared item API and implementation wait

ariadne:2 receipt4258f321-73da-41b0-9212-41fb9ba1dfee supplies the shared
item seam in upstream candidate cbdced604d90234af1cbcc0fae5f104fa40e3b93:

```go
type ChatItem struct {
    Pos, Start int
    Lines []string
    Answered bool
}
func ChatItems(section string, legacy bool) []ChatItem
func IndentOf(line string) int
```

Start is a zero-based source-line index in the passed section; Pos is ordinal,
not the operator's reused item number. Lines include the item content and
exclude trailing blank separators. Answered also recognizes legacy done links.
The Pair selectChatItems adapter consumes these source facts and maps them to
original byte ranges; it no longer selects item grammar from Goldmark list
syntax independently. Goldmark's existing render/code facts still protect
marker identity and syntax in validation.

Build disjoint governing source regions once from laminar sections. Feed
source chunks to ChatItems without rescanning full overlapping parent/child
sections; associate candidates with the original governing section and join
its shallowest-indent decision using IndentOf. Count total bytes/lines fed to
the shared parser and joins at N/2N; never call a linear Innermost search per
item. The independent source oracle and complete-item guards must agree before
retiring the old item-selection path.

Source inspection found the candidate ChatItems fence mask toggles on either
backtick or tilde prefixes without matching kind/length. Sent upstream
be9a4d4d-7f6e-46cb-9e3a-28e2c5cf7e97 for qualification against its existing
Goldmark/CodeLiteralMask seam. This is a suspected grammar defect, not yet a
measured failure here. Pair will not copy or patch the upstream parser.

Plan-quality resolved PQ-1/PQ-2; estimates passed with advisory INFO, and
change-code recorded accepted design4ed7f036fac126928237ad92e9f4e89a17483710.
The new core APIs exist only on #576's branch, so feature implementation must
wait for its landed module version. Recheck the final shared API and run the
gate on this clarified plan before core edits; do not install candidate source.

### 2026-10-23 14:39 — shared fence defect confirmed and corrected

Authenticated ariadne:2 receipt1fbe8bfc-2363-4734-a721-86910b5b597c reports
measured pre-fix item exposure across mixed-kind and short closing fences.
Upstream candidate f8b819032777535397f4eb4be6af4c020def126a replaces the
toggle with Goldmark-backed `sidedoc.CodeLines(text string) []bool`; ChatItems
consumes that shared code mask. Source inspection here verified the call and
TestChatItemsCodeFollowsCommonMark. The owner reports counted N/2N coverage;
this is upstream evidence, not a local candidate-module test or landing proof.

CodeLines includes fence delimiters and indented code. Qualification must
follow actual CommonMark structure: an unindented line can terminate a list
item's fence and become a real new bullet. The Pair independent complete-item
oracle must preserve actual source membership rather than the old toggle's
incorrect expectation. Add CodeLines to the shared API provenance pin when
#576 lands; no Pair fence scanner. The dependency wait and exact next step
remain unchanged.

### 2026-10-23 17:26 — final shared API landed

Verified ariadne#576/PR420 MERGED at
354e5fac5c96e0b2c26350ee712eb835746a6d67, present in the refreshed normal
substrate. The ordinary versioned module requirement now pins that merge,
not a worktree replace. This completes the upstream API gate.

Final ChatScan also exposes Headings []ChatHeading{Line, Level, Name} and
Heading(level, name); use those shared heading facts for byte-coordinate
association rather than a second heading-name normalizer. Shared ChatItems,
IndentOf and CodeLines are now landed with Legacy/Tags and the fence fix.
Pin export-v4 source/fixtures at this same revision. The adapter keeps plain
original byte offsets; Compile and ValidateEdits no longer accept a runtime
processing Catalog. Existing transaction/round codec owners stay in place.

Next: refresh change-code on this final API description; implement the already
RED core tests, then finish all planned public-seam, native and operator
acceptance before close. No further upstream API wait.

### 2026-10-23 17:44 — shared answered-link correction qualification

The implemented #576 adapter exposed an upstream defect: invalid/empty link
URLs are marked answered. Owner-confirmed ariadne#581 corrects that shared
reader/writer grammar; final close now additionally requires its normal landed
module pin and the unchanged regressions GREEN. The architecture and product
contract are unchanged; no local parser fork or retry substitutes for the fix.

Mechanical native conformance uses the established testcouch Claude/Codex
fixture personalities and does not claim model semantic judgment. Both passed
on afb420dc6863376050548e1e7f40697ca09345d2; the operator's agreed stage check
still qualifies the actual editor/agent behavior. Qoder is not a supported
personality in this harness, and no real-model/Qoder credit run is claimed.
All 13 collaborative primary fixtures pass public CLI; localized records retain
the existing coarse diff refusal rather than expanding #838 into a new diff.
