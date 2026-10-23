---
id: 000823
status: working
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '09bb6bc4e55322b99d40bc8b49d3ccc99135e50b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T03:16:56-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
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

- [ ] Approve the durable [implementation plan](../plans/000823-editorial-modes-plan.md), including the mechanical/semantic boundary, constrained operator migration, and #419 consumer dependency.
- [ ] Run `sdlc change-code` for the full-flow plan-quality gate and derived estimate after TL approval.
- [ ] Pin contract fixtures/catalog; implement and test the pure snapshot compiler, additive fill and map migration.
- [ ] Integrate policy-bearing reads, scoped CAS and one guard across all agent-write paths; propagate per-span presentation through live/resumed consumers.
- [ ] Expose the constrained migration command, update agent guidance/atlas, qualify agents, run recorded checks and close review.
- [ ] Coordinate the #419 mixed-rendering check and TL-authorized operator smoke, then land and report.

## Log

### 2026-10-22

### 2026-10-23 — Claimed and planned

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

## Revisions

### 2026-10-23 — Implementation boundaries from source inventory

Added the durable plan and approval points without changing the operator's settled three-axis contract. The plan
accounts for the missing section-policy CAS binding and for #419's explicit consumer ownership. No automatic
legacy-map migration, external operator-document rewrite or new Couch side-doc service is proposed.
