# Conversation flock implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach. Follow the approved plan and SDLC gates. This draft is PLAN ONLY; do not enter change-code until the decisions below are resolved and the TL approves the revision.

**Goal:** Replace historical native-writer forensics and durable admission reservations with one kernel-held lock per native conversation.

**Architecture:** The wrapper owns a nonblocking exclusive flock, and a native child retaining the same open-file description extends its lifetime. The kernel decides contention; holder metadata supplies a display label only. Preserve the existing recovery confirmation and transcript-selection paths.

**Tech stack:** Go, Darwin/Linux flock, exec.Cmd.ExtraFiles, existing Pair wrapper/PTY and recovery flows.

**Target:** [Slot recovery](../targets/slot-recovery.md). Continue/Resume from disk/Start fresh remain the visible outcomes; startup must not perform recovery writes.

## Status and decisions required before code

The primitive works, but passing ExtraFiles alone does **not** satisfy every promise in the issue. Two measured blockers must be resolved explicitly, not covered by a new fallback scanner:

1. **Codex's native engine does not inherit the fd through its Node shim.** The installed `/opt/homebrew/bin/codex:241` invokes `spawn(binaryPath, args, {stdio: "inherit", env})`. A scratch reproduction shows Node holding the lock, the fake engine receiving EBADF for the passed fd, and a contender acquiring after Node is killed while that engine remains alive. Merely passing an fd to the immediate launcher is insufficient for the orphan guarantee.
2. **Fresh unknown IDs are discovered after exec.** Claude/Qoder/Grok have caller-minted IDs (`launcher.MintsSessionID`). Codex/Agy/Muse use the detached watcher after startup. An fd opened after that event cannot retroactively be inherited by an already-running agent. In-app conversation changes/forks have the same lifetime question.

The issue's strict promise needs proven native launch/ID integration for these cases before deleting the old guard. The alternative is an explicitly weaker wrapper-lifetime guarantee, which is simpler but does not meet the current orphan-child Done when. **Recommendation:** retain the current guarantee unless the operator explicitly changes it; first approve a bounded feasibility step for the three primary agents, then approve its concrete integration before implementation. Do not silently weaken the invariant or build a permanent forensic fallback.

The proposed two-boundary cutover below also needs approval: an intermediate build carries the new lock plus the old guard only for migration; after the operator's confirmed fleet relaunch, the final build deletes the old mechanism. An atomic alternative requires a maintenance window with all old native writers stopped before the new-only build runs. A final build deployed over unguarded live writers has no safe automatic cutover proof.

## Core concepts (proposed; not yet implemented)

| Entity | Kind | File | Status |
|---|---|---|---|
| Native conversation key and safe filename | PURE | cmd/internal/nativelock/key.go | new |
| Free/held/error outcome and holder-label formatting | PURE | cmd/internal/nativelock/result.go | new |
| Secure stable-inode flock handle; close-only lifetime | INTEGRATION | cmd/internal/nativelock/lock.go | new |
| Native start/descriptor handoff boundary | INTEGRATION | cmd/internal/wrapcmd/native_lock.go | replaces native_admission.go |
| Lock artifact path | PURE | cmd/internal/artifactpath/native_lock.go | new |
| Selector resolution/pinning | INTEGRATION | cmd/internal/launcher/native_resume_selector_resolve.go | retained/modified |
| Fresh binding notification | INTEGRATION | cmd/internal/sessionwatch/lifecycle.go; wrapcmd lifecycle consumer | unresolved; modified only after ID/fd design approval |
| Exact source retirement attribution | INTEGRATION | cmd/internal/launcher/recovery_process_owners.go | moved from forensic file; not a conversation-ownership scan |
| One-operation migration override | INTEGRATION | existing launcher/recovery and Couch relaunch adapters | temporary in proposed M1; removed in M2 |
| Native writer forensic proofs and reservation store | INTEGRATION | deletion inventory below | deleted in final M2 |

The key identifies the agent and exact canonical native ID, not cwd, tty overlap or a guessed latest conversation. Resolve existing indirect selectors through their existing native inventory semantics before admission; do not regress the operator's supported selectors. Inventory limitations remain explicit prerequisites, not an excuse to introduce another reservation.

## Kernel ownership contract

- Resolve `<Pair cache>/native-locks/<agent>-<safe exact ID>.lock` through artifactpath. No traversal via IDs. Use one stable inode per key; never unlink or atomically replace it online.
- Secure open uses no-follow, regular-file and same-inode checks, with owner-only directory/file permissions. Reuse existing secure-open patterns. Do **not** reuse terminalLease.Close: it explicitly LOCK_UNs the shared description and would drop a surviving child's lock.
- Make exactly one LOCK_EX|LOCK_NB attempt. Held means refuse with `in use by <slot>` and offer Fresh. Missing/malformed holder metadata means `in use by another Pair session`; it never changes the decision. Other IO errors remain visible errors, never proof that the lock is free.
- Write bounded holder metadata in place only after acquiring; a losing contender must not truncate it. Keep it display-only; no PID inference, readiness scan or stale-owner repair.
- Pass the owned descriptor into the **actual native writer's** supported launch chain before exec. The wrapper closes only its copy after the child is done or launch fails; it must not explicitly unlock a shared inherited handle.
- A successful preflight probe is not admission. The wrapper's actual acquisition immediately before native startup is definitive. Source retirement/confirmation remains separate; do not stop a source merely because another conversation is held.
- No fd is attached to arbitrary tools to make a test pass. Installed launcher chains and native runtime descriptor behavior need conformance, not a fake that always preserves fds.

## Verified source map and deletion inventory

Replace these authority paths:

- `launcher/native_createflow_os.go`: ReserveNativeResume and CancelNativeResume.
- `launcher/createflow.go:750–830,1030`: admission reservation, command digest and PAIR_NATIVE_ADMISSION export. Keep native selector pinning, prompt and launch-status behavior.
- `wrapcmd/native_admission.go`: grant claim/child publication at startNativePTY; both wrapper entry forms use this boundary.
- `couchcore/recovery_prepared.go:179,388` and `launcher/recovery_authority_os.go:192`: global native owner checks. Keep immutable prepared recovery, source confirmation, transcript/capture validation and authority consumption.

Delete after cutover:

- `launcher/native_admission.go`, `native_admission_os.go`, the wrapper grant/store interface and their store-specific tests/fakes.
- `launcher/recovery_native_legacy.go`, `recovery_native_metadata.go`, `recovery_native_owner_orphans.go`, `recovery_native_parallel.go`, global writer-proof portions of `recovery_native_owner.go`, and their forensic-only tests.
- `couchcore/recovery_native_owner.go` and its forensic diagnostic adapter/tests when its last caller is gone.
- `artifactpath/native_admission.go`, native-admission runtime methods/tokens/retention entries, source inventory entries and obsolete store references.
- `procutil.LsofWritableNames`, its strict parser/live conformance fixtures and Makefile.local selection if the caller sweep still finds only the deleted forensic consumer; similarly sessioninventory.NativeIDForArtifact.

Do not blindly delete every recovery_native symbol. `launcher/recovery_terminal_confirmed.go:170,174,291` uses recoveryNativeWrappers/recoveryWrapperReady to attribute the **confirmed stop set**. Move that narrow responsibility to the named source-retirement integration and keep its identity/ancestry safety tests. This is not global conversation-ownership proof.

Retain shared strict JSON/ledger decoding improvements, ordinary LsofNames, native inventory/selector resolution, transcript bindings, launch history and session lifecycle observation: they have other consumers. Enumerate callers with rg before each deletion and run the artifact-reference inventory check.

## Proposed sequence (not approved yet)

### Before M1: resolve the measured native integration gap

- [ ] Verify native fd survival for Claude/Codex/Qoder against scratch launches, without working slots or real user data. Exercise the installed Codex shim shape, not only direct exec.
- [ ] Present the smallest concrete native fd/unknown-ID solution and its upkeep cost; or obtain an explicit operator revision of the orphan guarantee. No implementation until approved.
- [ ] Name every entry mode: exact resume; latest/continue; Fresh with caller ID; Fresh with post-start ID; native in-app switch/fork; wrapper fresh re-exec. No mode silently bypasses the chosen policy.

### M1 — lock integration and confirmed cutover (proposed review boundary)

- [ ] Write red real-process tests for contention, display-only metadata, inherited fd lifetime, wrapper SIGKILL, orphaned child, last-holder SIGKILL and two-start barriers. Test symlink/nonregular/replaced inode and failed launch cleanup.
- [ ] Implement the small lock IO seam and native start integration, with portable temporary storage and stateful fake kernel handles for ordering tests. Keep actual flock subprocess tests as conformance.
- [ ] Integrate the approved Fresh/selector/native-launch solution; inspect source and test ordinary wrapper launch plus from-launch-env and fresh re-exec paths.
- [ ] Add the temporary relaunch-only cutover override. It requires person-or-hub confirmation naming what stops, is bound to the existing prepared operation, cannot authorize arbitrary argv, is consumed once, and is logged. A replay must not retire or launch twice.
- [ ] Run build, focused/race tests, full recorded suite and milestone-close. Stage only when scheduled by the TL.
- [ ] Operator acceptance: scratch lock conflict + orphan cases; then the explicit one-time relaunch of every old slot, with transcripts preserved. Check the already-reported hello-v2 wrapper build identity and no old in-flight launches. Do not declare migration complete from a cache marker alone.

### M2 — remove old authority and finish (proposed review boundary)

- [ ] After the operator confirms cutover, delete the forensic/reservation inventory and temporary override, with obsolete tests. Keep only the lock as native writer authority.
- [ ] Remove the old reservation data/lock files only through an explicit machine cleanup approved after no old binary can still use them; exact paths are `~/.cache/pair/native-admission-v1.json` and `.lock`. Do not delete anything during planning.
- [ ] Update artifactpath inventory, atlas/couch.md and target references; document native agents outside Pair and manually starting pre-cutover Pair binaries as outside this guarantee.
- [ ] Full suite/build and M2 close review; state production and test net LOC against branch base, and prove no forensic/reservation callers remain.
- [ ] TL/operator live acceptance: resume one parked scratch conversation, refuse a conversation held by another live scratch slot with its label, preserve Fresh action. Then whole-issue close and land on SHIP.

## Ordering, constraints and residue

- **ARCH-ORDER:** kernel acquisition is the only writer admission. Competing starts race that same inode; exactly one proceeds. Start failure closes the parent's copy, not another holder's. No state resets infer external child death. Recovery confirmation and source teardown keep their existing exact source guard.
- **ARCH-CONSTRAINTS:** local Darwin/Linux filesystem; one nonblocking acquisition, no polling wait or OS-wide scan, one live handle per held conversation. Measure latency under concurrent starts rather than promising a new arbitrary timeout. No network work on admission.
- **ARCH-SECURE:** IDs and holder data are untrusted typed inputs; metadata is bounded and diagnostic, symlinks/replaced inodes refuse, same-user diagnostic labels are not authority. No new credential.
- **ARCH-FUNERAL:** handles die with their last holder. Stable lock inodes remain per conversation to avoid split-inode races; there is no append per turn or launch. Document and measure inode/metadata cost per conversation. Do not garbage-collect lock files online; any directory reset requires a quiescent machine operation. This residue policy needs operator acceptance.
- **ARCH-DRY / ARCH-PURE:** keep key/label logic pure, kernel/native IO in small seams, use existing artifact cache resolution and recovery flows. Delete the old mechanism, not just bypass it.
- **ARCH-MOCK / ARCH-PURPOSE:** fake handle ownership spans fork/exec/close and competing opens, then conforms against real subprocesses. A direct-exec fake does not establish Codex engine inheritance. Do not claim the strict orphan guarantee until each supported launch mode has a proven path.

## Measurements retained locally

Scratch directory: `/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-595-flock.XreBFC/`.

Run `python3 <directory>/measure.py`, `extra.py`, and `spawn-proof.py`; results are `results.jsonl`, `extra-results.jsonl`, and `spawn-results.jsonl`.

Observed on this Darwin host: a separate reopen, even within the same process, contends with errno 35; dup survives original close; an inherited child retains the lock after wrapper SIGKILL; last-child SIGKILL and reap frees it. The exact Node spawn shape drops fd3 in the fake native child; after killing Node, the fake engine stays live while a second opener succeeds. All helpers were scratch-owned and cleaned up. No coding agent or live slot was launched/stopped.

## Revisions

- 2026-10-15: Draft recorded after #594 landed (47a485ce). Source and kernel measurements expose unresolved inherited-fd and post-start-ID requirements. No code or cleanup is authorized by this draft; get the operator/TL decision before refining it into an executable approved plan.
