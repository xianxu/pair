---
collaboration-mode: discussion
---

# Exact Claude native-root validation

> **For agentic workers:** Consult AGENTS.md Section 3. Use the approved bounded-task delegation while the main worker monitors #651. Checkboxes are implementation steps, not milestone boundaries.

**Goal:** Resume an exact validated Claude root even when unrelated native history exceeds the recursive discovery cap.

**Architecture:** List immediate authorized project directories and probe only `<project>/<canonical UUID>.jsonl`. Feed candidate metadata into the existing scanner, target selection and stable-EOF validation. Do not derive a project solely from current cwd, which would break relocated sessions and duplicate detection.

**Tech stack:** Go, existing sessioninventory Runtime/RecoveryRuntime, descriptor-relative NOFOLLOW filesystem IO.

## Core concepts

No pure model or public result changes: reuse NativeSelectorRequest/Resolution, Artifact, FileEntry, TargetValidation, and the scanner's existing root/identity rules. The bug is the IO query scope, not root eligibility (ARCH-PURE, ARCH-DRY).

| Integration | Location | Status | Wraps |
|---|---|---|---|
| Exact Claude candidate visitor | cmd/internal/sessioninventory/native_selector_claude_os.go | new | Authorized immediate project directories and exact UUID leaves |
| nativeSelectorRuntime.ListFiles dispatch | cmd/internal/sessioninventory/native_selector_io.go | modified | Optional exact-ID visitor with existing runtime fallback |

Add an internal optional capability `VisitNativeSelectorFiles(context.Context, StorageRoot, string, int, func(FileEntry) error) (int, error)`. Its OS implementation accepts only the Claude authorized projects root and canonical UUID. Set an exact-Claude hint only after canonical input checking in native_selector.go. Existing fake runtimes without the capability retain existing behavior. The OS fixture and counting runtime exercise the actual production capability; no new binary dependency or process.

## Bounds, authority and lifetime

- Keep RecoveryScanTimeout, RecoveryMetadataLimit and RecoveryTotalReadLimit unchanged. Never silently truncate or accept partial candidate discovery.
- P immediate root entries cost one bounded directory-entry read each, at most one exact-leaf stat per project, plus bounded parent identity checks. Unrelated descendants H cost zero traversal/reads. Transcript validation remains linear in selected target bytes B, with existing byte/time limits.
- Read one root entry at a time, reject root/project/target symlinks and nonregular targets, check cancellation between syscalls, and preserve authorized root descriptors through NOFOLLOW opens. Root/project replacement or uncertain absence refuses. Only a positively absent target in an unchanged project can be skipped.
- Snapshot root/project directory identity around discovery/probes so a renamed or replaced parent cannot hide a conflicting target. Preserve same-ID duplicate refusal across projects and all scanner role/identity contradictions.
- Latest discovery still traverses its original bounded scope and refuses at the unchanged global cap.
- No persistent state, cache, worker, registry, schema or new residue; local descriptors and maps end before return (ARCH-FUNERAL). No detached IO work or background launch.

Actions: retry exact resume; choose another exact session; explicitly Start fresh. No automatic Fresh.

| Evidence | Visible result | Recovery | Possible loss |
|---|---|---|---|
| One validated exact root, large unrelated history | existing resolved result | exact resume | none from lookup |
| Missing, child, disputed or duplicate target | existing none/refusal | select correct exact ID | none from lookup |
| Unreadable/replaced/changed evidence, limit or cancellation | existing incomplete/error | retry after evidence settles | none from lookup |

This changes no durable state between events: lookup only produces a result consumed by the existing launch gate. In-flight cancellation closes owned descriptors; a synchronous syscall already entered must finish. A missing scrollback does not authorize or veto this native root query.

## Task 1: demonstrate the regression

- [ ] Add native_selector_claude_os_test.go using an isolated NewOSRuntime home and a valid `<project>/<UUID>.jsonl`.
- [ ] Add over 10,000 unrelated descendant entries, including directories; request the explicit ID with a different/absent cwd. Expect exactly that canonical root to resolve.
- [ ] Run `go test ./cmd/internal/sessioninventory -run '^TestNativeSelectorClaudeExactLargeTree$' -count=1 -v`; retain the current incomplete-result failure before implementation.

## Task 2: bounded exact-leaf visitor

- [ ] Implement the optional visitor in native_selector_claude_os.go using existing openRecoveryFile, StatRecoveryFile, sameRecoveryInfo, root authority, errors and budgets. Do not hardcode a new native home policy.
- [ ] Dispatch only exact Claude root inspection through that capability; keep fallback and latest semantics. Keep scanner and final identity validation unchanged.
- [ ] Re-run the regression and existing NativeSelector tests, then record red/green evidence.

## Task 3: safety and growth proof

- [ ] Cover duplicate exact IDs across projects, child/sidechain, contradictory metadata, missing target, malformed/incomplete content, target/project symlinks, replaced metadata and cancellation.
- [ ] Count metadata work with N and 2N unrelated descendants and confirm unchanged shallow counts; with P and 2P projects confirm linear bounded work. Test immediate-root cap exhaustion still refuses.
- [ ] Assert latest lookup still returns incomplete on the same over-cap tree. Reuse the production scanner and real OS fixture for conformance; existing mutable fake covers metadata mutation/negative scanner semantics.
- [ ] Run targeted race/catalog checks and the required recorded full suite on the pushed head; retain every failed attempt.

## Task 4: review, landing and acceptance

- [ ] Update atlas/couch.md with bounded exact Claude lookup and keep limits explicit; add the directory-versus-descendant lesson.
- [ ] Close once with fresh review; land after #651 through normal sdlc merge. Hub grants landing priority; do not apply a grant or bypass gates locally.
- [ ] Coordinate operator acceptance through the hub before Claude hub restart. Read-only native resolution can verify preserved transcripts without starting or replacing an agent; any actual relaunch needs hub coordination.

## Approval and alternatives

Hub delivery c4bfa120-b5ef-4b05-aa04-71fb3bdd09c8 approved bounded exact-target validation, unchanged global cap/identity, >10k regression, landing after #651. Delivery 506f4ca5-b991-4e2f-97bf-2ae28c69befc posted the shallow exact-leaf refinement preserving relocated and duplicate-root cases. Raising the cap only postpones failure; current-cwd-only lookup loses existing identity guarantees. This plan uses no third mechanism or new operator state.
