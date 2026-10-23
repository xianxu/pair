---
id: 000823
status: codecomplete
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-23
estimate_hours: 4.274
collaboration-mode: collaboration
card_mirror: '77f226dc27424a9e499b34e93f7aef1e185c8914' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T03:16:56-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 4.31
---

# unified-editorial-interface: Couch parses, protects and enforces per-section modes (heading comments, aliases, migration)

## Problem

Couch currently reads only the legacy document-level `presentation` setting. Section directives, canonical aliases,
collaboration routing and processing policy have no shared Go resolver. Its section write hash covers the section
body but not the document settings that govern that body. Adding independent readers would let the read, apply and
workbench paths disagree about authority and presentation.

## Spec

**Settled with the operator (10-22, design chat in brain `session.side/2026-10/2026-10-22-01-editorial-modes.md`):**
- **Uniform names:** `collaboration-mode` (where an answer goes: discussion / collaboration / chat),
  `presentation-mode` (how edits are shown: highlight / direct / propose), `processing-mode` (how an edit is made:
  the review modes in `ariadne/construct/local/universal-review/modes/`: copy-editing, developmental, fact-check,
  free-form, line-editing, proofreading; unique prefixes allowed). The old keys `presentation` and
  `process-instruction` stay readable as aliases. `process-instruction`'s free text splits out into
  `processing-instruction`.
- **Per section:** all three can be set for one section by an HTML comment directly under its heading:
  `<!-- collaboration-mode: chat; processing-mode: copy-editing -->`. Couch protects that line as operator-owned;
  parley conceals it as a badge.
- **Frontmatter:** each key is one value covering the whole document; the per-section map form
  (`collaboration-mode: {chat: ["Operator instructions"]}`) is retired.
- **Precedence:** section comment, then document frontmatter, then the built-in default.
- **Typeahead (required):** on a line next to a heading, `<!-- ` offers the three keys, `key: ` offers that key's
  values, `; ` offers the keys not yet used, and accepting the first key inserts the closing ` -->`. Values come from
  the modules that enforce them (extending parley's `lua/parley/frontmatter_enum.lua`), never a second list.


Project `unified-editorial-interface`; implements the spec in ariadne (dependency). In Couch:
- **Parse** the three keys (and the aliases) from frontmatter as single values, and the per-section heading comments,
  resolving each section's effective modes by the precedence; read legacy per-section maps with a deprecation warning.
- **Protect** the heading-comment line as operator-owned, like the operator-owned frontmatter keys: an agent edit
  that adds, removes or changes it is refused; the operator's edits pass.
- **Enforce** the effective per-section modes in the review round (collaboration routing, presentation of edits,
  processing mode), section by section.
- **Migrate** existing documents' per-section maps (e.g. `ops-echo/session.md`) to heading comments, as an
  operator-run step, since the keys are operator-owned.


**Side-doc layout (decided 10-22):** `<doc>.side/YYYY-MM/YYYY-MM-DD-NN-slug.md`: a year-month subfolder, with the full date kept in the filename so a file still makes sense when linked, grepped or moved. **No migration:** existing flat names stay valid (readers accept both). Reason: `ops/session.side/` already holds about 270 entries, 261 from October alone. For this issue: make this repo's side-doc creator use the new layout and its readers accept both. Applies to the naming rule in the universal-review spec (`collaboration-modes.md` §4) and to every side-doc creator (parley, Couch, and tl-lead's side-doc helpers).

## Done when

- A document with frontmatter defaults and two sections overriding different keys resolves each section's modes
  correctly (test), aliases included.
- An agent edit to a heading comment is refused; an operator edit passes (test).
- A round applies chat in one section and collaboration in another in the same document (test).
- The migration converts a per-section map to heading comments exactly (test).

## Plan

- [x] Approve the durable [implementation plan](../plans/000823-editorial-modes-plan.md), including the mechanical/semantic boundary, constrained operator migration, and #419 consumer dependency.
- [x] Run `sdlc change-code` for the full-flow plan-quality gate and derived estimate after TL approval.
- [x] Pin contract fixtures/catalog; implement and test the pure snapshot compiler, additive fill and map migration.
- [x] Integrate policy-bearing reads, scoped CAS and one guard across all agent-write paths; propagate per-span presentation through live/resumed consumers.
- [ ] Expose the constrained migration command, update agent guidance/atlas, qualify agents, run recorded checks and close review.
- [ ] Coordinate the #419 mixed-rendering check and TL-authorized operator smoke, then land and report.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.
Read v2/v2.1 for primitive ranges and library availability; v3.1 scales implementation hours to 40%. This is focused
ship-time, including qualification/docs/review; queue wait and #419's own implementation are outside this issue's
estimate. Familiarity 1.0 reflects the existing session-sync/write/strictjson surfaces. The thorough plan earns
×0.2 design discount on implementation primitives and +15% design buffer; issue shaping is counted separately.

| Work unit | Primitive | Raw design → recorded | Raw implementation → recorded |
|---|---|---|---|
| Issue shaping and peer contract | issue-spec | 0.5 → 0.5 | 0.2 → 0.08 |
| Pinned catalog | smaller-go-module | 0.2 × 0.2 = 0.04 | 0.3 × 0.4 = 0.12 |
| Compiler/source/fill | greenfield-go-module | 1.5 × 0.5 library × 0.2 = 0.15 | 0.8 × 0.4 = 0.32 |
| Pure migration | smaller-go-module | 0.3 × 0.2 = 0.06 | 0.5 × 0.4 = 0.2 |
| Edit classifier/guard | greenfield-go-module | 1.0 × 0.2 = 0.2 | 0.8 × 0.4 = 0.32 |
| Scoped CAS/read integration | smaller-go-module | 0.3 × 0.2 = 0.06 | 0.5 × 0.4 = 0.2 |
| Preserving round codecs | smaller-go-module | 0.2 × 0.2 = 0.04 | 0.4 × 0.4 = 0.16 |
| Live/resume consumers | lua-neovim | 1.0 × 0.2 = 0.2 | 0.8 × 0.4 = 0.32 |
| Constrained migration CLI | smaller-go-module | 0.3 × 0.2 = 0.06 | 0.5 × 0.4 = 0.2 |
| Agent guidance | skill-or-dispatcher | 0.3 × 0.2 = 0.06 | 0.3 × 0.4 = 0.12 |
| Atlas | atlas-docs | 0.1 × 0.2 = 0.02 | 0.1 × 0.4 = 0.04 |
| Shared consumer qualification | cross-repo-refactor-small | 0.2 × 0.2 = 0.04 | 0.2 × 0.4 = 0.08 |
| Disposable-agent qualification | smaller-go-module | 0.3 × 0.2 = 0.06 | 0.5 × 0.4 = 0.2 |
| Final review boundary | milestone-review | 0 | 0.5 × 0.4 = 0.2 |

Library check: Goldmark/YAML nodes eliminate writing Markdown/YAML grammars, halving compiler design. They do not
supply editorial inheritance/migration/write authority; no library discount is applied there. Existing strictjson
and transaction machinery justify smaller-module classifications rather than greenfield services. No new API is
introduced. Design subtotal 1.49; implementation subtotal 2.56; total 1.49 × 1.15 + 2.56 = **4.274 hours**.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.5 impl=0.08
item: smaller-go-module design=0.04 impl=0.12
item: greenfield-go-module design=0.15 impl=0.32
item: smaller-go-module design=0.06 impl=0.2
item: greenfield-go-module design=0.2 impl=0.32
item: smaller-go-module design=0.06 impl=0.2
item: smaller-go-module design=0.04 impl=0.16
item: lua-neovim design=0.2 impl=0.32
item: smaller-go-module design=0.06 impl=0.2
item: skill-or-dispatcher design=0.06 impl=0.12
item: atlas-docs design=0.02 impl=0.04
item: cross-repo-refactor-small design=0.04 impl=0.08
item: smaller-go-module design=0.06 impl=0.2
item: milestone-review design=0 impl=0.2
design-buffer: 0.15
total: 4.274
```

## Log

### 2026-10-22

### 2026-10-23 — Claimed and planned
- 2026-10-23: closed — BR-1 full source and Markdown structure preserved at d72901e10: full editorial/sessionsync/reviewapply PASS 0.254/12.465/0.595s, real-link/block rendering regressions and ready-referral controls pass, bounded N/2N tail/item joins pass, guard fuzz 11642exec/2s. README BR-2 accepted round2. Reader inventory PASS after per-file audits. Native exact-source e375 PASS22.386s after tracked Zellij startup flake835; d729 rerun underway. Recorded affected check queued on d729; landing requires final recorded pass. no-plan-check acknowledges only current close/recorded checks and post-close #419/TL smoke+landing rows remain pending; implementation complete, no other gate waived.; review verdict: SHIP

Claim/start-plan completed in pair:9 on the issue branch. Read the landed #562 contract and pinned its producer
commit and 39 source files in `workshop/evidence/000823/contract-provenance.json`, including the eight editorial-fill
cases' 24 files. No production code or executable fixtures changed; no tests are claimed to have run.

Read-only code inventory found: all agent mutation paths converge on sessionsync contentStep/BuildEdit except an
empty ApplyFile path that also needs validation; section hashes omit document policy; the existing operator lock
is not human authentication; no production Couch side-doc creator exists. Tracked Pair documents contain no real
legacy per-section maps, so migration needs synthetic conformance fixtures and an explicitly selected operator
document rather than a blind repository rewrite. ARCH-DRY shaped the single compiler/guard; ARCH-PURE keeps the
compiler and migration independent of IO.

Coordinated with parley.nvim:3: #418 owns whole-document canonical/legacy fallback and fill; TL assigned per-span
presentation consumption to #419. Proposed optional span `presentation` with direct/highlight/propose, explicit on
new spans and falling back to the existing document field on old artifacts. Need exact consumer agreement with
#419's assigned owner before end-to-end acceptance. Preserve unknown JSON fields while validating known ones.

The durable plan exposes three approval points: semantic prose/routing remains an agent obligation while Couch
enforces mechanical scope/ownership/presentation; map migration is a constrained operator-confirmed transform,
not an authenticated same-UID security boundary; side-doc guidance/readers are Couch's actual surface while real
Parley writers remain #419. No `change-code`, implementation, suite or stage has started.

**NEXT ACTION:** send the committed plan to the hub for approval. Once approved, run `sdlc change-code --issue 823`,
resolve its plan-quality findings and supply the derived estimate before changing code. Coordinate the optional
span field/fallback with #419's owner when assigned. Do not resume obsolete #752 runners from the old continuation.

### 2026-10-23 — TL approved implementation admission

TL delivery `d7b0c3ed-ff27-4595-a0c6-310b167fca0b` approved the committed plan and all three boundaries: mechanical
enforcement versus semantic agent review; constrained operator-confirmed migration under normal lock/CAS without
human-authentication claims; Couch guidance/readers here, actual Parley writers in #419. Directed proceeding to
`sdlc change-code`. The plan checkpoint is `16f9bf7859a0665e01824d21bc96324c38ecf00f`.

**NEXT ACTION:** run `sdlc change-code --issue 823`; resolve its fresh-context plan-quality findings before code,
then derive the estimate through the gate. No suite, stage or implementation has started.


### 2026-10-23 — Plan-quality round 1 refinement

Initial change-code inferred quick from design line count; re-ran with explicit `--flow full` to honor the approved
full-flow plan before implementation. The fresh Codex review returned two Important findings: PQ-1 named-function
adversarial test strategies and PQ-2 recursive duplicate handling in persisted JSON. Replaced case inventories with
one function/strategy table and specified strictjson's existing preserving split/encode helpers for the changed
round/span/record boundaries. Recorded the class lessons; no implementation or estimate before plan acceptance.

**NEXT ACTION:** re-run `sdlc change-code --issue 823 --flow full`; after plan-quality accepts, derive and set the
estimate using the named v3.1 calibration and re-enter the gate. Ledger is binary-owned; do not edit it by hand.


### 2026-10-23 — Plan accepted; estimate derived

Plan-quality round 2 accepted both PQ-1 and PQ-2, with no new findings. Admission stopped only for the missing
estimate, as expected. Derived the estimate below after that acceptance from the named v3.1 source; the source is
flagged stale by SDLC, so this is provisional calibration rather than a guarantee. No code has changed yet.

**NEXT ACTION:** set the authoritative card estimate to 4.274 hours and rerun full-flow change-code. After the
estimate-quality/admission gates pass, start Task 1 with failing Catalog/contract tests before implementation.


### 2026-10-23 — Full admission and Task 1 complete

Full-flow change-code passed. Plan-quality had no remaining blockers; estimate-quality returned INFO, accepting the
derivation while flagging compiler/write-guard and integration/verification time as optimistic. Active checks,
qualification and stage/build time count toward actuals; queue wait and #419 implementation do not. No gates waived.

Implemented Task 1 in `cmd/internal/editorial/`: immutable Catalog with six exact embedded mode files and all 33
remaining pinned contract files. Modes/Lookup return detached values; ResolveProcessing uses exact/unique prefix;
collaboration values use vocab.CollaborationModes and presentation validates highlight/direct/propose. Metadata
parsing retains exact brief bytes. The shared snapshot compiler will consume this Catalog in Task 2.

TDD: the implementer observed missing Catalog APIs fail before implementation, then targeted tests passed (0.136s)
and fuzz seeds passed (0.067s). Root `go test ./cmd/internal/editorial` passed (0.066s). Independent spec review
compared all 39 files against the pinned Git objects and found compliance; subsequent quality review approved with
no Important issues. The existing artifact inventory initially failed only on the new catalog source (2.561s);
registered it as naming no Pair artifact paths; the exact guard then passed (2.138s). Added the producer's pinned Goldmark/YAML
versions for Task 2; existing ariadne module already supplies the shared vocabulary, avoiding a broad module update.

**NEXT ACTION:** start Task 2 with failing Compile/parseOwnedSource tests and producer fixtures, then implement
byte-preserving document policy/fill/migration. Do not start a suite or stage yet. #419 owns the coordinated mixed
presentation consumer and actual Parley side-doc writers. All prior #752/#753 work is landed; old continuation is
obsolete. Preserve this branch and the approved full-flow design.


### 2026-10-23 — Resume, codec and consumer contract

Recovered via sdlc state; retained approved full-flow admission and resumed Task 2 with failing Compile/source/fill
tests. In parallel with that pure compiler, implemented the independent Task 4 codec/Pair consumer slice. RED tests
showed dropped unknown metadata, accepted null/oversized position tuples, missing required span members and unchecked
span presentation. Preserving codecs now use strictjson recursively at round/span/record/artifact boundaries;
production encoding retains opaque nested values and large number tokens. Resume records materialize the effective
presentation so a legacy span inheriting direct does not resume as highlight. Pair Lua honors the same fallback.

TL receipt 51c11a3a-141f-4652-b618-15554632953b says #419 is unowned and must not block Pair implementation.
Parley :3 confirmed the schema in receipt 5c69c73b-269e-4050-889c-1157e052da7a and requested the committed mixed/legacy
fixture. Added testdata/editorial-rounds/presentation.json with source/result hashes, valid positions, expected
values, opaque metadata and mixed review-record round-trip. No runtime dependency on the unlanded branch.

Verification: full go test ./cmd/internal/sessionsync passes (2.249s); reconstruct, round_watch and record Lua tests
pass; bash tests/review-applied-round-test.sh passes. Independent ad-hoc codec review approved with no Important
findings; its observed existing negative end-column hole now has a failing regression and the corrected guard
passes focused Editorial tests (0.147s). Artifact source inventory passes (2.119s). Task 4 is still pending base-policy
span attribution from Task 3; these results do not claim full mixed-policy enforcement or #419 acceptance.


### 2026-10-23 — Compiler complete, enforcement checkpoint

Task 2 compiler/fill/migration passed independent spec and quality review after fixing concrete adversarial
failures: mixed-newline closer corruption, malformed-versus-ordinary comments, quoted/indented authority,
componentwise obsolete alias selection and explicit block-scalar indentation/chomping. Tests now include
independent generated nested inheritance and migration-routing oracles and pinned fixture seeds.

Implemented the Task 3 pure guard and shared marker owner, base-policy transaction adapter, resolved read payload,
policy-bound section hashes, mixed proposal conversion and span attribution. Every registered/legacy agent path
uses that base policy; standalone empty Apply validates before old-round removal. A full sessionsync run exposed
restamp growth shifting reconstruction offsets at five-digit years; reconstruction now precedes automatic
restamping and the existing prospective-size regression passes. Ad-hoc guard/integration review is pending.

Task 5 constrained migration CLI is implemented and independently approved: exact candidate preview and full
SHA256 confirmation, normal registry/flock/lock guard, no arbitrary body and no Shadow/state acknowledgement.
Native actor qualification is being prepared using the existing shared disposable harness. TL explicitly
confirmed Claude/Codex fixture personalities qualify transport/mechanics; semantic observations wait for combined
operator acceptance with parley.nvim#419. No alternate harness, live fleet fixture, stage or merge started.

Fresh verification: go test ./cmd/internal/editorial ./cmd/internal/sessionsync ./cmd/internal/reviewapply -count=1
PASS (0.195s / 6.739s / 0.255s). Artifact production-source inventory PASS 2.293s. Migration CLI parser/run/registry
regressions PASS 0.379s. Earlier Pair Lua/pane integration PASS remains applicable pending final rerun. No recorded
full suite or close gate is claimed. Emitted skill baseline identified missing resolution/ownership/chat/path
instructions; an independent updated reference probe now answers all those cases from the shipped skill alone.


### 2026-10-23 — Fresh guard approval and final-check preparation

Fresh guard/integration review APPROVED after concrete regressions fixed pending-marker adjacency/forged turns,
code-example approval, chat shallowest indentation and pending escaped answer links. Marker parsing now uses one
linear bracket index; N/2N malformed-input counts pass. Final span modes come from validated base pieces, including
left-biased insertion at heading boundaries. Tests exercise all nine setting-protection seams and seven mixed-output
write seams, operator settings on fresh read, real pending chat links and migration growth at the 4 MiB limit.

Latest complete editorial/sessionsync/reviewapply package run PASS 0.172s / 4.218s / 0.177s. Full couchcmd PASS70.957s;
Pair Lua and real pane apply script PASS. Guard fuzz PASS21,115 executions/3s and codec fuzz PASS72,702/3s.
Reader inventory PASS after per-file audits; artifactpath and repolint packages PASS. Registering the provenance
scanner in the existing fixed preflight preserves affected-runner ownership. Public help now documents resolved
policy/coordinates, opaque section tokens and migration. Existing CLI tests consume returned tokens.

Native fixture mechanical qualification passed for Claude and Codex (25.096s), with joined/cleaned disposable
roots. Earlier Zellij startup panic is filed as unclaimed #835 with the exact failure and qualification-source
limitation. No installed LLM semantic acceptance is claimed. The next step is committed-source native rerun,
SDLC publication/recorded affected check and binary-owned close review. Post-close #419/TL operator acceptance
and landing remain visibly unchecked; no stage or :0 mutation has occurred.

### 2026-10-23 — Pinned native pass, PR and recorded verification

PR https://github.com/xianxu/pair/pull/557 publishes code head dcef993baf17ff93db5b6b2410661cc45f87cb51.
Exact committed-source TestNativeEditorial PASS25.910s, Claude9.57s and Codex8.56s, both owned roots fully joined
and cleaned. The recorded affected check selected the full suite (native test/fixture paths) and queued as
lease-1667798340 on that code head; no suite pass is claimed yet. All fresh ad-hoc compiler/guard/codec/migration
reviews approved. The durable inventory now names delivered helper/type owners rather than predicted files.

The local acceptance review can overlap that queue per tl-worker. The unchecked issue Plan contains the current
close itself and post-close #419/TL smoke/landing obligations; they remain visibly pending, rather than being
falsely ticked for local codecomplete. The precise no-plan-check acknowledgment, if needed at close, applies only
to those workflow obligations. No implementation, test, atlas, actual or review gate is waived. Landing still waits
for the recorded suite and TL/operator acceptance disposition.


### 2026-10-23 — Close review repair

The first full close returned REWORK: BR-1 protected-source-extent found that Goldmark child Lines omit list
block delimiters, so a chat item closing fence could change; BR-2 requested README settings/migration guidance.
The production-seam regression TestEditorialChatProtectsCompleteFencedItem reproduced the fence mutation
(RED: expected editorial-chat, received success). The repair tracks complete list-container source membership
inside the same Goldmark parse, retaining lazy child extents; empty blocks and delimiters remain operator-owned.
README now names canonical defaults, adjacent heading syntax, protected ownership, migration preview/confirmation,
resolved reads and opaque section tokens. ARCH-DRY: observe the existing parser rather than add a fence parser.

Cancelled obsolete queued lease-1667798340 before code changed; its interrupted suite is not a verification pass.
Repair verification: full editorial/sessionsync/reviewapply PASS (0.232s / 11.517s / 0.216s), repolint PASS13.221s,
artifactpath PASS17.991s. Complete source membership, default-renderer equivalence and counted N/2N observer tests
pass, including the production-seam regression. Audited changed document_test.go and new markdown_test.go readers:
only local non-Go contract fixtures, retained full-suite fallback; individually recorded certificates. Reader inventory
PASS. The final repaired head needs a new recorded check, native rerun and full close review. No finalized close
generation exists.

### 2026-10-23 — Round2 structural answer repair

Full close on e3754d59 returned REWORK on BR-1 only; README BR-2 addressed. Complete extents passed mutation checks,
but a final closing-fence answer append retained bytes while changing Markdown meaning and hiding the link.
TestEditorialChatAnswerMustPreserveMarkdownStructure reproduced all five syntax-tail cases RED, with independent
rendering showing hidden links or a changed thematic break. Require a renderable base tail plus real candidate
answer-link AST at the exact inserted label, using existing compiler walks. Final ordinary continuation remains
supported. Cancelled own queued suite lease-829031859 before this code repair; cancellation is not a pass.

Native e3754d59 first run FAIL212.758s: Claude PASS, Codex steward startup timed out; Zellij0.45.1 log at04:51:29
recorded lib.rs:1462:26 Option::unwrap None, before wrapper/agent startup. Root pc-3066221672 was fully cleaned.
Unchanged-source rerun PASS22.386s (Claude8.74s/Codex7.96s), exact guidance/mixed output/protected refusal/migration
and cleanup both pass. Added Codex subtest to unclaimed #835 using the exact fail/pass tool-transcript evidence;
it is not a saved suite attempt or suite-pass substitution. No budget/product/helper/test change between runs.

The same class audit reproduced arbitrary continuation inserted at the old item end: source-intersection alone
missed added ask text. Candidate-selected item identity/extent validation closes this boundary too; syntax outside
an ask may not hide or demote it. Ready-marker referral authorization remains covered by positive controls.

Repair verification: complete editorial/sessionsync/reviewapply PASS (0.254s /12.465s /0.595s); guard fuzz
PASS11,642 executions/2s. Renderable-tail tests retain inline code/link/image/whitespace/backslash/plain-heading
controls. Exact candidate answer links exclude image alt text. 64/128-item real-link batches prove bounded linear
tail/item joins. Ready-referral-only reshape and inline approved edits remain supported; unrelated outside edits
cannot piggyback on a referral reshape. Re-audited edits_test.go's unchanged local non-Go fixture inputs and
individually refreshed its existing full-fallback certificate.

### 2026-10-23 — SHIP, final native pass and read-only real-document audit

Full close round3 SHIP/high confidence, no findings; evidence5c69989257669939a9b55a43ddbe0ff693f902b4,
code d72901e10c6f0d5de5aef2daf05c1aba13130ae3. Reviewer verified both source-extent and safe-answer mutation
regressions fail without fixes; full package/Lua/pane checks and compiler fuzz58,974 executions pass. Final code
native TestNativeEditorial PASS38.692s (Claude12.30s/Codex14.29s), both disposable generations and IO fully joined.
Recorded suite lease-515637891 is queued on that code head; no recorded suite pass claimed yet.

TL disposition631aff87: land on suite pass, no staging; semantic acceptance end to end with parley.nvim#419.
A temporary binary importing the candidate editorial package called DefaultCatalog/Compile over ops/session.md,
all recursively nested ops/session.side Markdown and brain equivalents: 285 files/1,025,315 bytes. Every read's
before/after source hash matched; helper invoked no session-sync IO and wrote no source/sidecars. 280 compiled,
all default discussion/highlight/free-form, with brain Chat correctly inheriting chat from the legacy map. No
presentation differences against the pre-823 reader. Warnings:158 deprecated process-instruction aliases and1
legacy brain map, all as designed. Five TL-authored Oct19 ops side docs initially refused answer/archive values.

TL disposition7216d20a: TL corrected the five own noncanonical values to discussion; recheck only those five,
then merge on suite pass. Read-only recheck PASS5/5, no warnings/refusals: 06-second-machine,07-fleet-health-history,
08-history-backup-status,09-introduce-team-status,10-pair4-qoder (all2026-10-19). Each resolves discussion/highlight/
free-form; all five before/after hashes match. Both temporary source helpers removed. Audit JSON retained in
local scratch pair823-audit.ivNfzH/report.json and five-report.json. No external document was edited by this worker.
TL owns the project tick. No stage or :0 mutation is authorized or needed; suite pass is the remaining merge gate.

## Revisions

### 2026-10-23 — Implementation boundaries from source inventory

Added the durable plan and approval points without changing the operator's settled three-axis contract. The plan
accounts for the missing section-policy CAS binding and for #419's explicit consumer ownership. No automatic
legacy-map migration, external operator-document rewrite or new Couch side-doc service is proposed.

### 2026-10-23 — Complete source extents and user documentation

BR-1 tightens the planned chat-item preservation implementation to include all parsed source syntax, including
empty/fenced blocks, through a transparent list-parser observer. BR-2 restores README user guidance. These are
repairs to the existing preservation/documentation contract; no authority or acceptance boundary changes.

### 2026-10-23 — Safe chat-answer source boundary

BR-1 preservation includes Markdown meaning: answer appends require a renderable final block tail and a real
candidate link, excluding syntax-only/code endings. This tightens the existing answer-link contract and retains
ordinary final continuation support; semantic acceptance and #419 boundaries remain unchanged.
