---
type: continuation
slug: editorial-snapshot-compiler
agent: codex
created: 2026-10-23T03:51:11
supersedes: cast-sketch-gesture-qualification
branch: 000823-unified-editorial-interface-couch-parses-protects-and-enforces-per-section-modes-heading-comments-aliases-migration
worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
issues: [000823]
---

# Continuation: editorial-snapshot-compiler

## NEXT ACTION

Run `sdlc state` and read pair#823's latest Log plus `workshop/plans/000823-editorial-modes-plan.md`. Full-flow change-code has PASSED; do not repeat the plan or estimate review just because this continuation was added. Start Task 2 with failing `Compile`/`parseOwnedSource` tests in `cmd/internal/editorial/`, seeded with the pinned source fixtures already committed there. Implement byte-preserving document policy resolution, additive fill and requested map migration according to the approved plan. No suite, stage, review runner or merge is active. Do not follow the obsolete #752 NEXT ACTION in the old .couch continuation.

## State of play

Slot pair:9, hub ops:0. CWD `/Users/xianxu/workspace/worktree/pair-slot9/pair`. Current branch `000823-unified-editorial-interface-couch-parses-protects-and-enforces-per-section-modes-heading-comments-aliases-migration`. Clean source checkpoint `eb9bef27285f318fd42884726115e35caac13112` completes Task 1. #752, #767, #771 and #753 are landed; do not revive their runners or stages. #783 was an earlier next item but the TL dispatched #823 instead.

TL approval delivery `d7b0c3ed-ff27-4595-a0c6-310b167fca0b` approves all three boundaries below and directs implementation. Initial change-code inferred quick; root re-ran with `--flow full`. Plan-quality required named-function adversarial test strategies and recursive duplicate rejection; both are addressed, all subsequent reviews CLEAN. Estimate-quality INFO accepts the v3.1 derivation (4.274 hours) but warns custom compiler/guard and verification may be optimistic. Gate admission passed, flow is recorded full. Do not manufacture new milestone boundaries or redo accepted gates unnecessarily.

## Thread arc & user model

The operator/TL wants one unified three-axis editorial contract, with section comments protected and enforced consistently across Couch and Parley. They expect autonomous implementation after the approved plan, without another product interview. The important distinction is truthful enforcement: Couch owns mechanical authority/scope/presentation; agents still interpret prose and processing briefs. Preserve old documents and unknown metadata. Migration is explicitly requested/operator-confirmed, never an incidental review rewrite. Coordinate the consumer seam rather than implementing a competing Parley resolver.

## Artifact map

Read issue `workshop/issues/000823-unified-editorial-interface-couch-parses-protects-and-enforces-per-section-modes-heading-comments-aliases-migration.md` first. Its latest Log carries exact admission/test evidence and next action. The approved durable plan includes the pure/integration inventory, function test strategies, state/failure table, scoped CAS binding and schema proposal. Its binary-owned gate ledger has the long canonical issue stem followed by `-plan-gate.md`; never edit it manually.

Producer checkout `/Users/xianxu/workspace/worktree/pair-slot9/ariadne` contains the authoritative contract at `construct/local/universal-review/`. Pin is landed #562 commit `4f10a45d038acb2324c5a19e74a78f4a593571e6`; `workshop/evidence/000823/contract-provenance.json` records 39 exact source digests (all eight fill cases = 24 files). Task 1 copied six exact mode files into `cmd/internal/editorial/modes/` and remaining 33 files into `testdata/contract/`. Contract tests compare against the separate provenance, and the spec reviewer independently compared all source Git objects.

`cmd/internal/editorial/catalog.go` is now the pure catalog owner. API: `Mode{Name, Order, Scope, Frontier, Brief}`, Scope WholeDocument or MarkersOnly; `DefaultCatalog() (*Catalog,error)`, `NewCatalog(fs.FS)`, `Modes()` copied slice, `Lookup(name)`, `ResolveProcessing(exact-or-unique-prefix)`. Strict `ResolveCollaboration` uses `vocab.CollaborationModes`; strict `ResolvePresentation` accepts highlight/direct/propose. Document fallback belongs the upcoming compiler. Brief bytes preserved exactly. Goldmark v1.7.13 and go.yaml.in/yaml/v3 v3.0.4 added for compiler; no ariadne module upgrade was necessary because the old module already exports collaboration vocabulary.

Mutation integration: `cmd/internal/sessionsync/round.go` contentStep/BuildEdit, `sync.go` read/prepareWrite, `applyfile.go` standalone ApplyFile including empty edits. Current Presentation reads only legacy document alias. `file.go` H2 section addressing remains unchanged; new ATX policy tree is separate. Existing section body hash omits policy and must bind document/ancestor authority as an opaque scoped hash. Validate before advancing Shadow or mutating file/round. Existing marker.go owns ready-marker bracket parsing; reuse/extract the one owner instead of duplicating grammar when implementing frontier/chat classification. Avoid an editorial↔sessionsync import cycle.

Consumer/format integration: Go Round/Span and roundRecord, Pair Lua `nvim/review/round_watch.lua` live and `reconstruct.lua` resume (both currently highlight all spans); record.lua already passes unknown fields through table JSON. Existing `cmd/internal/strictjson` SplitObjectPreserveNumbers/EncodeObjectPreserveNumbers reject recursive duplicates and preserve raw unknown fields. Reuse them at changed round/span/record and enclosing artifact boundaries, validate owned presence/types/null and positional tuples explicitly. Do not lose opaque nested members through an incidental json.Marshal re-compaction.

Actual Couch guidance source is `cmd/internal/couchcmd/skills/couch/SKILL.md` embedded by skill.go. Atlas pages are couch-agentic-documents.md and review-workbench.md, linked by atlas/index.md. No production Couch side-doc allocator exists; guidance/readers here, real Parley writers in #419.

## Decisions & dead ends

Approved boundaries: (1) mechanical ownership/scope/chat preservation/presentation in Couch, semantic review and routing classification remain agent obligations; (2) constrained preview/confirmed migration under existing lock/CAS, not human-authentication against same UID; (3) Couch guidance/readers versus actual Parley writers. No generic operator bypass, no arbitrary migration body payload, no new review service, no auto migration and no external ops-echo rewrite. Tracked Pair files have zero real legacy maps; use conformance fixtures and later explicitly selected operator document.

Contract details: defaults discussion/highlight/free-form; per-axis nearest heading comment > canonical document or legacy component > defaults. Canonical presence beats aliases, even empty/unknown presentation: fallback highlight and warn on unknown, never fall through. Heading declarations strict. Missing processing mode/instruction fall back to legacy components independently. Legacy multiple leading modes readable all briefs/first flags, but fill refuses unchanged. Maps remain readable/warn; all duplicate normalized headings match. Unknown/conflicting map assignments fail, unmatched names warn on read and refuse migration. Requested migration preserves effective routing and other axes; its explicit additive fill is the production consumer of FillCanonical (ordinary agent writes never fill owned settings).

Parley coordination: parley.nvim:3 owns #418 whole-document resolver/fill, pin same producer commit and source semantics. TL explicitly assigned #419 mixed-span rendering and section UI/writers. Proposed optional `Span.presentation` = direct/highlight/propose; absent falls back to Round.presentation then highlight. New spans explicit from BASE snapshot; carry into commit review-records/resume. Keep top-level Round.presentation document default, no mixed enum. #419 consumer upgrade is acceptance dependency: cannot claim old Parley renders mixed spans correctly. Exact schema sent in submitted receipt `7f759ac0-0451-4860-82a6-e2589d0103fc`; plan sent TL submitted receipt `46d63df6-7827-4aec-81e6-7c63a872057a`. Coordinate exact fixture with #419's assigned owner, not #418.

## Lessons learned

Task 1 TDD: missing new APIs failed before implementation; implementer targeted GREEN0.136s/fuzz seeds0.067s. Root `go test ./cmd/internal/editorial` PASS0.066s. Independent spec review COMPLIANT and quality review APPROVED. Existing artifact inventory failed on the new source, then catalog classified as no Pair artifact paths and exact guard PASS2.138s. Every later new production Go source needs that manifest classification too. No full suite claimed.

Before every close/reclose run make test-affected-inventory; audit each named reader before per-file hash repair, never bulk refresh. Use SDLC for recorded checks and merge; no direct full make suites. No restarting while an owned recorded suite queues/runs. Mandatory close/plan fresh review belongs binary; don't add redundant boundary reviews. Task-level bounded subagent work is authorized by AGENTS§3, but do not spawn without a concrete independent prompt. Catalog implementer/spec/quality agents are finished; no agent is writing now.

Whole bare couch/sdlc calls, no wrapping/pipes/substitution. Hub for product questions/staging go; no in-pane permission prompts. Current agent trailer verified: `Agent: codex/gpt-6.1-sol/medium`; coauthor GPT-6.1 Sol <noreply@openai.com>. This continuation checkpoints completed Task 1 before the larger compiler unit; continue Task 2, not another plan-writing loop.
