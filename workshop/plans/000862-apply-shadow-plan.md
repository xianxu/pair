# Preserve unread changes across session-sync apply

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Steps use checkbox syntax. One atomic issue-close boundary; no Mx milestones.

**Goal:** An apply acknowledges its own changed ranges without acknowledging unread operator edits elsewhere.

**Architecture:** Compute the prospective shadow in the existing locked preparation phase. A pure mapper reuses the bounded line/byte diff to translate actual apply changes into unchanged shadow coordinates. A conservative mapping failure keeps the previous shadow and returns a visible warning; document edits still obey the existing all-or-none apply transaction.

**Tech stack:** Go, existing sessionsync diff/resolver/publication machinery, portable file rigs, existing Couch CLI.

## Decision and alternatives

1. Recommended: map actual changed ranges through the unchanged intervals of the old shadow versus the captured current file. Preserve every other shadow byte. Reuse changedRuns/LineDiff rather than resolving record occurrences independently (ARCH-DRY, ARCH-SECURE).
2. Re-resolve records independently on the shadow: smaller, but unsafe when unread operator insertions/removals change occurrence counts. A matching occurrence is not a provenance mapping.
3. New persisted coverage/acknowledgment state or a general three-way merge: unnecessary lifecycle and compatibility surface. Existing shadow plus conservative warning represents uncertainty safely.

The recommendation requires TL approval of the conservative outcome below. It follows the existing section-write fallback that keeps a shadow when its section cannot be located; the warning makes the resulting own-edit noise explicit.

## Contract

- Apply keeps the existing guarded current-file hash, exact record resolution, editorial checks, round, folds and all-or-none document edit.
- Only apply changes shadow behavior. Explicit whole-file read/write, scoped section/frontmatter writes and standalone ApplyFile retain their contracts.
- Use actual changed byte runs between captured current content and the record-built content, before automatic restamping. A broad Old record and a no-op must not acknowledge unchanged operator text merely because it lies inside Old.
- Compute unread runs from previous shadow to captured current content via the existing bounded changedRuns helper. Their complement gives ordered byte-equal spans with base/shadow coordinates.
- Map each actual apply run only when its complete old range lies in one byte-equal span. Handle an insertion at a boundary conservatively: both neighboring coordinates must agree; a boundary containing unread insertion/deletion is unmappable. Verify the mapped old bytes and preserve order/nonoverlap.
- Replay mapped runs on the old shadow using one ordered builder. Apply the existing generated Updated restamp behavior to the prospective shadow; never replace its surrounding unread content. Pin stamp presence/absence, Unicode and CRLF cases.
- If any changed range is unmappable, or there is no prior baseline for a changing apply, keep the entire old shadow (empty when absent). Do not fall back to the new whole document. The document apply still succeeds with warning: baseline was retained; own edits may remain in diff; read the intended scope to acknowledge it. No raw document text in that warning.
- A no-op apply advances no unread content. Existing automatic stamp handling stays explicit in tests.
- Preparation computes the complete prospective shadow, warning, encoded state and registered limits before any write. Existing publication failure behavior remains: document/shadow/state are separate effects, IO errors require inspection before retry. Do not add a retry or store.
- Compose the baseline warning with round/late-lock warnings so none disappears.

## Core concepts

| Name | Kind | Location | Status |
|---|---|---|---|
| applyShadow projection | PURE | cmd/internal/sessionsync/apply_shadow.go | new |
| changedRuns / LineDiff | PURE | cmd/internal/sessionsync/round.go, linediff.go | reused |
| preparedWrite / prepareWrite | INTEGRATION | cmd/internal/sessionsync/sync.go | modified |
| Sync.Apply / write publication | INTEGRATION | cmd/internal/sessionsync/sync.go | modified |
| portable session-sync rig | INTEGRATION | cmd/internal/sessionsync/*_test.go | reused |

The projection owns no mutable state or IO. Input is old shadow, captured source, record-built candidate and stamp time; output is prospective shadow plus mapping status. Preparation owns the decision to retain a baseline and produce the existing Result.Warning. Use existing preparedWrite and document transaction; no new publication owner.

## States, actions and recovery

Actions: apply, inspect diff/status, read intended scope, recompute/retry.

| Internal outcome | Visible result | Recovery | Possible loss |
|---|---|---|---|
| All changed runs map | apply succeeds; own changes absent from diff, other unread text remains | inspect diff | none |
| Missing baseline or unmappable run | apply succeeds; baseline-retained warning, own changes may also remain in diff | inspect diff, read intended scope | no unread content silently acknowledged |
| Stale/lock/owner/invalid records/limits | existing refusal before effects | inspect/read/recompute | none |
| Publication fails after document effect | existing IO error and retained artifacts | inspect document/diff/status before recompute/retry | partial metadata publication already possible today |

Existing flock/registry transaction owns the captured file and baseline through preparation/publication. Do not prove exclusion by an additional observation. This issue adds no concurrent worker or process and no new artifact lifecycle (ARCH-ORDER, ARCH-FUNERAL). Existing state formats and unknown fields are unchanged (ARCH-EVOLVE).

## Cost and limits

This is per-apply batch work, not per-keystroke. B = source/shadow/candidate bytes, L = lines, R = unread diff runs, E = actual apply runs. Reuse the existing Myers limit D <= 1024: alignment remains O((Lshadow + Lfile)D), with coarse changed-middle fallback past the cap. The new interval sweep is O(R + E), and builder copying is O(B + replacement bytes); never scan the document once per record. Memory remains bounded by existing document/metadata caps and diff trace, plus linear run tables/output. A coarse unread hunk is uncertainty, not an authorization to acknowledge it.

Count mapper span/run visits and copied bytes at N=8 and 2N=16 with proportional edits/bytes and fixed unread divergence; assert linear increments. Separately pin the existing diff-cap fallback, unchanged prefixes/suffixes, and unmappable middle. Timing under current machine load is not the oracle (ARCH-CONSTRAINTS).

## Execution and verification

- [ ] Add pure fixture tests first in apply_shadow_test.go and testdata/apply-shadow: shifted offsets; frontmatter plus body batch; unread edits before/after and in the same line/section; repeated Old strings with changed occurrence counts; broad Old with narrow change; no-op; insert/delete/multiline/Unicode/CRLF; overlapping unread range; absent baseline; coarse diff fallback. Assert exact expected shadow, preserved unrelated bytes and mapping status. Observe red before implementation.
- [ ] Implement the pure interval projection using changedRuns and ordered replay. Keep ambiguity conservative. Add the counted N/2N assertions and a generated invariant: every shadow modification is attributable to a mapped apply change or the existing stamp, and no unread outside range becomes acknowledged.
- [ ] Integrate only req.edits != nil in prepareWrite, before effects/metadata limits. Carry any warning through preparedWrite and combine with existing warnings. No mutation of the captured baseline during preparation.
- [ ] Public tests in apply_shadow_integration_test.go: registered and legacy files, frontmatter+body changes, unread Needs you edits elsewhere and inside the touched section, alias paths, current hash obtained without an acknowledging read. Assert exact document/shadow/state/round and diff. Pin mapped success, retained-baseline warning, no-op, stale/lock/fold guards and publication failures.
- [ ] CLI regression in cmd/internal/couchcmd/session_sync_test.go uses the portable store/ledger and apply --json plus diff; verify warning JSON and stderr paths without changing wire shape. Standalone ApplyFile and explicit read/write baseline controls remain green.
- [ ] Update README, atlas/couch.md, embedded Couch skill and CLI help to state range-scoped apply acknowledgment and conservative recovery. Audit reader inventory individually if fixtures add inputs.
- [ ] Coordinate the Ariadne tl-lead fake through the hub: approved contract and portable cases go to its owning slot, independently based on Ariadne main, no Pair runtime import or stacking. Replace TestFakeApplyAdvancesBaseline with unread-preservation and mapping-fallback cases. File its update unclaimed with dependency pair#862; the hub assigns it after Pair lands. Pair closes on its own implementation and contract handoff. No conversion of instructions link is in scope.
- [ ] Run focused tests one at a time per slot, count <= 10, under the overload notice. Then inventory, pushed-head sdlc local-check run --pre-close, whole close, authorized stage/live reproduction if the hub requires it, and authoritative queued merge verification. No unqueued full suite/native matrix.

## Approval and boundaries

Plan goes to ops:0 before change-code or implementation. Ask the hub to confirm the successful-apply/retained-baseline warning outcome and assign/coordinate the Ariadne fake update. No guessed estimate before plan-quality. If the mapper cannot safely represent a necessary case, stop and bring a concrete counterexample rather than adding coverage state or widening acknowledgment.

## Revisions

### 2026-10-24 — TL approval and separate fake adoption

Hub5ef6e854 approved the complete design and uncertainty behavior. The original
plan made the updated upstream fake a Pair close requirement; the hub instead
requires an unclaimed dependent Ariadne issue, assigned after Pair lands.
Pair's Done when is narrowed accordingly. Existing fake changes are deferred
to that issue, with no cross-repo implementation on this branch.
