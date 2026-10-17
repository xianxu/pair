---
collaboration-mode: discussion
---

# Frontmatter session-sync implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Execute this small, context-warm change in the current session using superpowers-executing-plans. The SDLC binary owns the fresh-context reviews.

**Goal:** Read and write a document's frontmatter without acknowledging or changing its body.

**Architecture:** Add an explicit frontmatter scope beside section and whole-file operations. Pure byte projection/replacement feeds the existing document lock, hash guard and existing per-file publication path. Scoped shadow projection preserves the old body baseline even when there was no shadow.

**Tech Stack:** Go, existing sessionsync file/registry stores and Couch CLI.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| FrontmatterBody / ReplaceFrontmatter | cmd/internal/sessionsync/file.go | new |
| WriteRequest | cmd/internal/sessionsync/sync.go | modified |
| sessionSyncArgs | cmd/internal/couchcmd/session_sync.go | modified |

- FrontmatterBody identifies exactly the closed leading delimiter block. Opening and closing delimiter lines use the existing trim-space `---` grammar, without YAML reserialization. Hash bytes include both delimiters but exclude the closing line ending. That line ending and every subsequent byte belong to the preserved body tail. Missing or unterminated blocks refuse this explicit scope.
- ReplaceFrontmatter accepts a complete delimited block, with optional trailing whitespace but no non-whitespace body payload. It preserves the destination tail exactly, including LF/CRLF boundaries and the Updated stamp. A block without a final newline remains safe because the original tail supplies the separator.
- WriteRequest gains an explicit Frontmatter boolean; it cannot coexist with Section. Existing empty Section still means all. Read keeps its existing signature and gains a ReadFrontmatter entry point sharing a private read implementation.
- Shadow projection uses the same replacement helper when its baseline has frontmatter. Without a shadow it records only the new block; with a prior ordinary-document baseline lacking frontmatter, prefix the block plus a separator and preserve the entire previous baseline. An unterminated leading block in a shadow refuses rather than silently acknowledging body bytes.
- ARCH-PURE: these byte transformations have colocated tests without IO mocks. ARCH-DRY: one splitter defines hash and replacement boundaries. No generic YAML model or scope framework is needed.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Sync.Read / ReadFrontmatter / Write | cmd/internal/sessionsync/sync.go | modified | existing locked document and shadow/state stores |
| parseSessionSyncCLI / runSessionSyncCLI | cmd/internal/couchcmd/session_sync.go | modified | CLI arguments and existing Sync operations |

Reuse real temporary file stores, the existing test rig and registry fixtures; no new external service or function-call mock is introduced. The existing state pin preserves the owed-event horizon before changing shadow mtime. Frontmatter writes retain type/owner validation, stale-hash and operator-lock refusal, folds, limits and atomic preparation. Only this scope skips Restamp because the body is outside its ownership. Reads retain the existing ability to operate under an operator lock.

## Tasks

### Byte projection and guarded operations

- [ ] Add function-level tests in cmd/internal/sessionsync/file_test.go and sync_test.go: FrontmatterBody and ReplaceFrontmatter get a byte-scanner fuzz oracle with independent suffix-preservation and round-trip invariants; ReadFrontmatter and Write get real-store state-transition tests asserting that acknowledged frontmatter never advances the body baseline. Exercise both legacy and registered routing and initial baseline creation.
- [ ] Run `go test ./cmd/internal/sessionsync -run Frontmatter -count=1`; observe the missing behavior fail before implementing it.
- [ ] Implement the pure helpers and explicit scope in the existing guarded path. Both content and projected shadow must be fully prepared before any write. Guard Write with refusal-state invariants over document/shadow/state and the owed-event horizon. Inject a publication failure by loading the real transaction fixture and directing its shadow output at an existing directory; assert that the document may contain the new frontmatter, its body and original body baseline remain unchanged, no fold is lost, and a fresh read/recomputed write completes safely.
- [ ] Run `go test ./cmd/internal/sessionsync -count=1`; commit the implementation and verified evidence.

### CLI and documentation

- [ ] Add parseSessionSyncCLI tests in cmd/internal/couchcmd/session_sync_test.go: generate combinations of scope/verb/flag tokens and assert exactly one permitted scope, preserving literal value handling. Test runSessionSyncCLI through real disposable read/hash/write/diff state transitions. Keep all existing forms compatible.
- [ ] Run `go test ./cmd/internal/couchcmd -run 'SessionSync.*Frontmatter|Frontmatter.*SessionSync' -count=1` and observe the missing flag fail, then extend parsing and dispatch.
- [ ] Update root `couch --help`, sessionSyncUsage, README.md, atlas/couch.md and the shipped cmd/internal/couchcmd/skills/couch/SKILL.md. Document the complete block payload, scoped acknowledgement, body timestamp exception, and light cost. Keep atlas/index.md's existing Couch link.
- [ ] Run `go test ./cmd/internal/sessionsync ./cmd/internal/couchcmd -count=1` and targeted race tests. Build the real CLI using the existing build target; run it from this slot in an empty allowlisted environment with disposable HOME/data paths, record read/write/diff, stale and operator-lock results. TL approved this live route; no pair:0 staging.

### Single close boundary and landing

- [ ] Push the candidate and run `sdlc local-check run --name suite`; retain its complete result. Run `sdlc close --issue 643 --verified '<concrete evidence>'`, resolving mandatory fresh-context findings and logging the verdict.
- [ ] Open the PR, integrate main and run `sdlc merge --yes`; let it own integrated checks and the landing line. Verify the merge on origin/main and report the PR and Git-derived merge hash to the hub. TL routes ariadne#424 adoption after landing.

## Verification details

The key regression begins with an agent baseline, changes an operator section in the actual file, reads then writes only the frontmatter, and asserts the operator addition remains in real unified diff while the frontmatter edit does not. Repeat for registered and legacy routing and absent shadows. Compare body suffix bytes rather than reconstructed Markdown. Stale and lock tests compare document, shadow and state before/after the refused operation. The real CLI smoke uses disposable inputs and outputs only; it must not read or write operator session documents.

## Publication and operating envelope

This is synchronous local CLI work. Reuse registered document/shadow/state limits from cmd/internal/sessionsync/registry_model.go and registry.go; reject oversized registered results before publication. Legacy files retain their existing whole-file in-memory IO envelope; this change does not impose a new legacy limit or streaming path.

Publication is not a multi-file transaction. sync.go publishes the document, then pinned state (if absent), shadow, and final fold/last-write state as separate file operations. A deterministic validation refusal occurs before effects; an IO error or process death may leave a changed document with old shadow/state, or an updated shadow with old final state. Scoped projection must keep the old body baseline in every such outcome. A caller treats an error as potentially applied: read the frontmatter again, inspect diff/status, recompute the desired block and hash, and explicitly retry any still-owed folds. Never replay an old hash blindly. This preserves the existing conservative acknowledgement contract rather than adding rollback or new transaction machinery.

## Revisions

- 2026-10-17: plan gate PQ-1/PQ-2: replace case inventories with named function-level property/state-transition strategies; correct the atomicity claim and specify partial publication, reconciliation, and controlled IO-failure verification. PQ-3: state the inherited registered limits and legacy work envelope.
