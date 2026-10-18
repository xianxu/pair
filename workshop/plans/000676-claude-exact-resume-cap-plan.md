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

- [x] Add native_selector_claude_os_test.go using an isolated NewOSRuntime home and a valid `<project>/<UUID>.jsonl`.
- [x] Add over 10,000 unrelated descendant entries, including directories; request the explicit ID with a different/absent cwd. Expect exactly that canonical root to resolve.
- [x] Run `go test ./cmd/internal/sessioninventory -run '^TestNativeSelectorClaudeExactLargeTree$' -count=1 -v`; retain the current incomplete-result failure before implementation.

## Task 2: bounded exact-leaf visitor

- [x] Implement the optional visitor in native_selector_claude_os.go using existing openRecoveryFile, StatRecoveryFile, sameRecoveryInfo, root authority, errors and budgets. Do not hardcode a new native home policy.
- [x] Dispatch only exact Claude root inspection through that capability; keep fallback and latest semantics. Keep scanner and final identity validation unchanged.
- [x] Re-run the regression and existing NativeSelector tests, then record red/green evidence.

## Task 3: safety and growth proof

- [x] Cover duplicate exact IDs across projects, child/sidechain, contradictory metadata, missing target, malformed/incomplete content, target/project symlinks, replaced metadata and cancellation.
- [x] Count metadata work with N and 2N unrelated descendants and confirm unchanged shallow counts; with P and 2P projects confirm linear bounded work. Test immediate-root cap exhaustion still refuses.
- [x] Assert latest lookup still returns incomplete on the same over-cap tree. Reuse the production scanner and real OS fixture for conformance; existing mutable fake covers metadata mutation/negative scanner semantics.
- [ ] Run targeted race/catalog checks and the required recorded full suite on the pushed head; retain every failed attempt.

## Task 4: review, landing and acceptance

- [x] Update atlas/couch.md with bounded exact Claude lookup and keep limits explicit; add the directory-versus-descendant lesson.
- [ ] Close once with fresh review; land after #651 through normal sdlc merge. Hub grants landing priority; do not apply a grant or bypass gates locally.
- [ ] Coordinate operator acceptance through the hub before Claude hub restart. Read-only native resolution can verify preserved transcripts without starting or replacing an agent; any actual relaunch needs hub coordination.

## Approval and alternatives

Hub delivery c4bfa120-b5ef-4b05-aa04-71fb3bdd09c8 approved bounded exact-target validation, unchanged global cap/identity, >10k regression, landing after #651. Delivery 506f4ca5-b991-4e2f-97bf-2ae28c69befc posted the shallow exact-leaf refinement preserving relocated and duplicate-root cases. Raising the cap only postpones failure; current-cwd-only lookup loses existing identity guarantees. This plan uses no third mechanism or new operator state.

## Revisions

### 2026-10-18 — PQ-1/PQ-2: directory evidence and executable test seams

This supersedes the earlier reference to sameRecoveryInfo for directories: recovery_os.go:130 requires regular files, so reuse it only for the existing transcript proof. openRecoveryFile (recovery_os.go:20-60) opens a safe descriptor but does not retain ancestor descriptors. The new visitor explicitly owns one root directory descriptor for the whole enumeration and one project descriptor around each exact-leaf probe, at most two additional descriptors, closed synchronously on every return.

Add a pure directoryFingerprint conversion from unix.Stat_t, containing device, inode, directory mode, size, mtime and ctime (seconds/nanoseconds). Read snapshots with unix.Fstat on the held descriptor. Before accepting the probe (including ENOENT), compare the original fingerprint with both current Fstat and a no-follow Fstatat of the directory's current pathname; require all values equal and directory mode. Held descriptors prevent inode reuse; ctime/mtime detect entry mutation even when the same directory remains. Compare the project around every leaf stat and the root around the entire one-entry-at-a-time enumeration. A vanished/replaced project is incomplete, never a positively absent leaf. These are bounded observations at each probe; no global filesystem transaction is claimed. Existing scanner checks revalidate selected content afterward.

For controlled tests, make the private visitor implementation accept a typed runtime seam with openRecoveryFile and StatRecoveryFile (OSRuntime supplies both); OSRuntime.VisitNativeSelectorFiles calls that same helper. Internal-package tests embed the real temporary-folder OSRuntime and override StatRecoveryFile to mutate the actual filesystem at the named probe boundary before delegating, rather than adding production test hooks or fabricating success. This stateful adapter exercises the same visitor helper and real descriptors.

| Pure entity | Location | Status | Unit strategy |
|---|---|---|---|
| directoryFingerprint | native_selector_claude_os.go | new | Distinct device/inode/mode/size/mtime/ctime inputs must compare unequal; non-directory mode refuses |

Function-specific integration proof (replaces the generic Task 3 case inventory):

- VisitNativeSelectorFiles/private helper: real over-10k descendant fixture must resolve through production ResolveNativeSelector; counted helper result at N and 2N descendants remains P immediate entries, and P/2P project fixtures scale linearly. Cap exhaustion, context cancellation, root/project/target symlinks and nonregular target must return incomplete/error. Stateful exact-leaf probe adapter renames/replaces a project, removes its parent, or inserts a target at the absence boundary; fingerprint mismatch is the mechanical refusal guard. A probe-boundary root replacement must fail the final root descriptor/name comparison. File descriptor ownership is checked on success and failure through repeated fixture operations, without scheduling sleeps.
- nativeSelectorRuntime.ListFiles: an explicit canonical Claude request dispatches to the optional visitor, an incapable fake follows the existing path, and other agents preserve existing traversal. A counting adapter traps recursive VisitRecoveryFiles during exact Claude validation; the exact-ID hint must never replace a latest snapshot's full discovery requirement.
- ResolveNativeSelector: duplicate same-ID roots in separate projects must refuse; relocated root with different/absent cwd still resolves. Real-file fixtures cover sidechain, contradictory sessionId, missing target, malformed/incomplete target, and target symlink. Existing mutable fake plus actual ReadRecoveryAt mutation adapters cover changed target metadata/EOF; final canonical identity and root role remain the guard. Latest over-cap fixture remains incomplete.

PQ-1 is answered by this production-function/controlled-ordering inventory; PQ-2 by descriptor ownership and the directory-specific ctime-bearing comparison. No interface or lifetime change outside the exact Claude query is introduced.

### 2026-10-18 — implemented proof names

The executable over-cap regression is TestClaudeExactOverDescendantCap (the earlier command name was a proposed name). TestClaudeExactParentChanges, TestClaudeExactVisitorGrowthAndLimits, TestClaudeExactRejectsUnsafePathsAndContent, TestClaudeExactCanonicalHintAndLatestOrdering, TestDirectoryFingerprintFields, TestClaudeExactChangedContentRefuses and TestClaudeExactVisitorClosesDescriptors pin the approved cases. Main-session review added exact returned-UUID and explicit real descriptor closure assertions before commit.
