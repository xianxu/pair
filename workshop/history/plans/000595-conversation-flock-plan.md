# Conversation flock implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). TL approved revision 2833b837 in receipt 659dde9c; use the normal change-code and close gates before implementation. This is one atomic implementation and one issue-close boundary, with no intermediate migration build.

**Goal:** Replace global native-owner forensics and durable admission reservations with wrapper-lifetime conversation locks.

**Architecture:** A Pair wrapper takes one nonblocking exclusive flock per known conversation ID and holds it until that wrapper exits. Its agent runs in the wrapper-owned child process group, which the wrapper terminates on every controlled exit before releasing its locks. A resume whose lock another live wrapper holds refuses at native start naming the holder slot, the same "in use" refusal as Couch's registry check; there is no cutover gate or admission (Revisions 2026-10-16).

**Tech stack:** Go, local Darwin/Linux flock, existing PTY process groups, readiness records, Pair session watcher and recovery confirmation.

**Target:** [Slot recovery](../targets/slot-recovery.md). Continue / Resume from disk / Start fresh remain the visible outcomes. Couch always starts; the transition guard runs only on explicit disk resume, never startup or attach.

## Approved scope revision and limit

TL cb293d05 replaces the original inherited-child guarantee with **wrapper lifetime**. Do not pass lock fds into native agents, adapt the Codex shim, preallocate native IDs through new native APIs, or add another ownership-discovery system. The wrapper starts locking when it knows an ID; Fresh with an unknown ID remains unbound until the existing Pair watcher reports it.

The accepted gap is wrapper SIGKILL while an orphan engine keeps writing: the kernel releases that wrapper's lock, and this guard does not discover the orphan. Document and route that case to #583's reaper and #587's unattributed report. Agents outside Pair are also outside the guarantee. No orphan-child lock-survival test is claimed as acceptance for this revision.

The same lock build removes the scan and store. There is no dual-guard build, temporary override flag, global cutover journal, or permanent forensic fallback.

## Core concepts

| Entity | Kind | File | Status |
|---|---|---|---|
| Exact agent/conversation key; safe filename and diagnostic label | PURE | cmd/internal/nativelock/key.go | new |
| Free/held/IO-error decision and known-ID set | PURE | cmd/internal/nativelock/state.go | new |
| Stable-inode, single-attempt flock handle | INTEGRATION | cmd/internal/nativelock/lock.go | new |
| Lock cache path | PURE | cmd/internal/artifactpath/native_lock.go | new |
| Wrapper lock acquisition / controlled shutdown | INTEGRATION | cmd/internal/wrapcmd/native_lock.go; wrap.go | replaces native_admission.go and modifies native start/exit |
| Current-launch binding notification | INTEGRATION | cmd/internal/sessionwatch/lifecycle.go; wrapcmd watcher launch | small notification from existing discovery, not a new discovery mechanism |
| ~~Native-lock readiness capability~~ (dropped 2026-10-16) | INTEGRATION | cmd/internal/readiness/record.go; wrapcmd readiness publication | unchanged |
| ~~Live legacy-slot transition guard~~ (dropped 2026-10-16) | PURE predicate / INTEGRATION observation | cmd/internal/launcher/native_lock_capability.go; Couch recovery adapters | new; replaces global owner checks |
| Exact confirmed stop-set attribution | INTEGRATION | cmd/internal/launcher/recovery_terminal_confirmed_owner.go | moved, preserving existing checks |
| Native owner forensics and admission reservations | INTEGRATION | deletion inventory below | deleted in this change |

Key/label/state tests run without IO. Kernel, watcher, liveness and process-group behavior use portable stateful fakes at their own seams plus real temporary-process conformance. Do not call IO orchestration PURE.

## Runtime contract

### Conversation lock

1. Resolve `<Pair cache>/native-locks/<agent>-<safe exact ID>.lock` through artifactpath. Validate/encode IDs without traversal; no cwd or tail inference.
2. Securely open one stable regular inode, with no-follow and owner-only permissions. Make exactly one `LOCK_EX|LOCK_NB` attempt. No polling or reservation.
3. Free: record a bounded diagnostic holder slot in place, then keep the handle in the wrapper. Held: refuse with `in use by <slot>` and offer Fresh. Missing/malformed holder metadata reads `in use by another Pair session`; it never changes the kernel decision. Other IO errors are visible, not absence.
4. Do not unlink or atomically replace an online lock inode. A losing contender never truncates the holder's metadata. The fd stays close-on-exec and is not passed to the native agent.
5. Exact-ID launches acquire before the agent starts. Resolve selectors with existing adapters where supported; retain today's indirect-selector UX. For an ID only learned after startup, acquire when the existing current-launch watcher reports it. A repeated report is idempotent; keep each known ID locked until wrapper exit. A late conflict stops the wrapper's owned child group and reports the held conversation, without retrying input or continuing unguarded.
6. Reuse the existing native binding decision, not historical ledger decoding or a new native integration. The minimal internal handoff is a private notification pipe from the already-launched Pair watcher to its wrapper, carrying the current launch ordinal and discovered ID. The watcher reports after its existing binding decision commits. It cannot notify a replacement wrapper through an old pipe. Close/join the bounded reader on wrapper shutdown; do not persist callback state or poll whole ledgers.

The notification is a best-effort transfer of knowledge, not a new prerequisite for Fresh: watcher start failure, expiry, EOF before a valid binding, malformed/truncated data, or death between ledger commit and delivery leaves an unknown-ID wrapper unknown. Show a truthful notice that its conversation lock is unavailable until discovery succeeds (next Relaunch can pin the established ID); never claim a held lock. An already-held exact ID remains protected on channel failure. After one valid delivered binding, normal watcher EOF is completion, not loss of the held lock. A late held-ID or lock IO error is different: the wrapper now knows the ID and must stop its own group. No input retry, silent unsafe success or recovery scan is introduced. Bound the notification to one small record at a time and reject a foreign ordinal before any acquire. Tests exercise the production pipe and commit-to-delivery ordering.

This deliberately promises nothing before an unknown Fresh ID is known. It preserves existing launch prompts/configuration and does not add an app-server bootstrap, native fd shim or conversation-ID selection policy.

### Owned process group and exit

`pty.Start` already establishes a new session and controlling terminal (`creack/pty@v1.1.24/start.go` sets Setsid and Setctty); retain and test that ownership. Consolidate cleanup around the directly created child group, never discovered foreign processes.

- Normal completion, setup failure after spawn, output failure, catchable termination (TERM/INT/HUP/QUIT), panic unwinding and fresh restart all use the same owned-child cleanup boundary.
- Signal only that owned group, use the existing one-second TERM grace then KILL, and reap the direct child. Terminate the group **before reaping the leader**, so a delayed signal cannot hit a recycled group ID. Cleanup is idempotent; cancel/join any timer or worker before returning.
- Hold conversation locks throughout cleanup. Release after the group shutdown/wait path, then allow fresh wrapper re-exec to acquire new ownership. Replace the current unjoined delayed SIGUSR2 group-kill goroutine with that same owned lifecycle.
- SIGKILL cannot execute cleanup; its orphan gap is explicit and belongs to the reaper/report paths. No process-table/argv/lsof admission proof is reintroduced.

### Rollout (no cutover gate)

Superseded 2026-10-16 (TL e93b109d): no readiness capability, no live legacy-slot guard, no Relaunch migration marker and no native admission. A wrapper started before this build holds no lock until its slot is relaunched; the gap closes as slots relaunch. The holder label falls back to the Pair tag (else "standalone") and is relabelled to the slot address once a background lookup resolves it.

## Delete and retain (verified at 484bc381)

Delete in the same code change:

- `launcher/native_admission.go`, `native_admission_os.go`; wrapper `native_admission.go`; `artifactpath/native_admission.go`; reservation-only tests/fakes, grant/digest/environment plumbing and runtime methods.
- Global owner proof in `launcher/recovery_native_owner.go`, plus `recovery_native_legacy.go`, `recovery_native_metadata.go`, `recovery_native_owner_orphans.go`, `recovery_native_parallel.go` and forensic-only tests.
- Couch's forensic diagnostic adapter/fake and calls in `couchcore/recovery_prepared.go` and `launcher/recovery_authority_os.go`; replace admission with the wrapper lock alone (no capability check; Revisions 2026-10-16), preserving prepared recovery/transcript/capture validation.
- The now-unused `procutil.LsofWritableNames` block and its dedicated parser/live tests, `sessioninventory.NativeIDForArtifact` export, and the live Make selection/companion workflow-command expectation if the final caller sweep still finds no other consumers.
- Artifactpath store family, bindings, source inventory entries and obsolete references. No reads/writes of native-admission-v1 remain.

**Move/keep 114 production lines** from owner.go: recoveryNativeWrapper, recoverySourceAncestor, recoveryNativeWrappers and recoveryWrapperReady. `recovery_terminal_confirmed.go:170,174,291` still needs exact source stop-set attribution. Move them to recovery_terminal_confirmed_owner.go, preserving identity/ancestry checks and replacing the forensic error-type dependency with a source-retirement error.

Keep selector classification/pinning and its UX tests, ordinary LsofNames, shared strict JSON/ledger validation, launch history, transcript bindings, native inventory and notification lifecycle consumers. Check every deletion's callers with rg; do not delete whole test files that still cover selectors, recovery confirmation or stop safety.

## Named implementation and test seams

| Function / boundary | Strategy and mechanical guard |
|---|---|
| nativelock.NewKey / ParseHolder | Fuzz malformed identifiers and metadata; canonical, bounded components and diagnostic-only decoding cannot influence lock ownership. |
| nativelock.Advance | Sequence-test unknown / acquiring / held / stopping / exited with discovered, acquired, failed, stop and cleanup-complete events; private state plus returned effects are the only route to acquire, stop and release. A single wrapper owner serializes events and IO completions; stopping rejects discoveries, retains late-acquired handles through group cleanup, and permits release only after cleanup-complete. |
| nativelock.Acquire / Handle.Close | Real subprocess barrier contention and kill/reap conformance; exactly one nonblocking flock, stable inode, loser cannot truncate, bounded holder text, close-on-exec. |
| sessionwatch.NotifyBinding / wrapcmd.DecodeBindingNotice | Pipe truncation/EOF/foreign-generation sequence tests after committed ObserveAndPersist; bounded records, ordinal fence and visible unknown-versus-held status. |
| ~~launcher.LegacyLockSlots~~ (dropped 2026-10-16) | Permute mixed current/live and stale readiness observations; pure filtering/sorting never derives liveness from file existence. |
| wrapcmd.stopOwnedGroup | Real scratch descendant lifecycle tests plus injected event ordering; signal only the directly created group before leader reap, join cleanup and hold locks until cleanup-complete. |
| existing recovery / confirmed Relaunch | Couch keeps #597's registry check; the wrapper lock refuses a held conversation naming its holder; stop-set confirmation is retained. (The maintenance refusal and migration exemption were dropped 2026-10-16.) |

Real flock, pipe and owned-group tests run in ordinary Go package tests, hence the existing required make test / recorded suite on Darwin and Linux CI; they do not require a new optional manual target. A temp-file measurement reports metadata bytes and filesystem allocated blocks per distinct conversation in the Log: payload bounded to 1 KiB, one inode and ordinarily one filesystem allocation unit (filesystem-dependent). Stable files are permanent during online operation; reclamation requires operator-approved quiescence of all cooperating wrappers, never an online unlink.

## Net deletion estimate

Source-counted old blocks, using Python splitlines counts and rg caller verification:

| Replaceable scope | Production lines | Test lines |
|---|---:|---:|
| Five global scan files | 880 | 1,124 |
| Reservation/store plus wrapper admission | 767 | 1,230 |
| Writable-lsof-only block/tests | 130 | 274 |
| Couch legacy-advice test | 0 | 138 |
| Subtotal | 1,777 | 2,766 |
| Preserve/move stop-set duties | -114 | retained safety tests |
| **Measured replaceable baseline** | **1,663** | **2,766** |

Allow roughly **350–650 added production lines** for the small lock, watcher notification and group cleanup (the capability check was later dropped), and **500–900 new/updated test lines**. Expected net deletion is therefore about **1,000–1,300 production lines**, or **2,900–3,600 production-plus-test lines**, before additional bridge/call-site removal. These are planning ranges, not promised final counts; report actual diff against branch base at close. Docs/review artifacts are counted separately.

## Implementation and acceptance (one atomic close boundary)

- [x] After TL approval, run change-code; derive the full-flow estimate only after plan-quality clears.
- [x] TDD the named lock, transition, notification and group-cleanup seams above, recording RED/GREEN and real scratch conformance evidence.
- [x] Implement lock ownership, move stop-set attribution, then remove the forensic/reservation inventory in the same build. Sweep callers and artifact references; no partial dual-guard deployment.
- [x] Update atlas/couch.md and issue Spec/Done when to this approved scope, preserve superseded promises in Revisions, and document orphan responsibility and maintenance rollout. Update target only if its existing commitments need the accepted limit stated.
- [x] Build, targeted/race/conformance regressions, required recorded suite, actual LOC report and mandatory issue-close review. No code may land on REWORK.
- [ ] TL schedules the operator live acceptance: Couch restarts over live slots, slots are relaunched onto this build at a quiet time, a parked scratch conversation resumes, and a held scratch conversation refuses naming its holder slot.
- [ ] Land on SHIP as the TL directs (after #589) and report the merge hash plus the operator acceptance still pending. Do not silently count an unrun live check as complete.

## Constraints and lifecycle

- **ARCH-DRY / PURE:** key/decision logic is pure; flock, watcher and process cleanup stay in small IO seams. Reuse current native discovery and recovery confirmation. The old authority is removed, not hidden behind the new one.
- **ARCH-ORDER:** explicit wrapper states are unknown-ID / held known-ID set / stopping / exited. Known-ID arrival acquires once; contention or IO failure starts controlled stop; stopping rejects late arrivals; exit closes the handles. No rollback guesses a native child dead, and no restart replays input. The owned group is killed before leader reap and lock release.
- **ARCH-CONSTRAINTS:** local Darwin/Linux filesystem; one nonblocking flock attempt, no process scan, metadata bounded to 1 KiB, current live wrapper set bounded by existing slot limits. Fresh discovery uses its existing cadence; no new native scan budget or background ownership poll.
- **ARCH-SECURE / MOCK:** validate IDs and current-launch notification at the boundary; metadata is display-only. Tests use temp cache/config and scratch descendants; stateful fakes model open/contend/close, binding order and controlled shutdown; real flock/PTY process tests check the model.
- **ARCH-FUNERAL:** fd lifetime is the wrapper's lifetime. Keep stable lock inodes per conversation to prevent split-inode races; no per-turn/launch append. Measure inode/metadata cost and never unlink them online. The obsolete reservation files may be removed only in a separate operator-approved quiescent cleanup after no old binary uses them; do not add a compatibility reader or cleanup daemon.
- **ARCH-PURPOSE:** the guarantee is cooperating wrapper lifetime, not orphan-engine exclusion. No native fd adapter, ID bootstrap, new reservation store, capability-policy sidecar or dual-guard migration build.

## Measurements retained locally

`/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-595-flock.XreBFC/` contains measure.py, extra.py, spawn-proof.py and their JSONL results. The kernel proof and Codex shim loss remain valid evidence for why the original inherited-child promise was rejected. All helpers were scratch-owned and cleaned up; no real coding agent or live slot was launched/stopped.

## Revisions

- **2026-10-15, draft 484bc381:** proposed inherited-child ownership and two-boundary migration, with explicit native fd/unknown-ID gaps; no code was authorized.
- **2026-10-15 19:58 PDT, TL cb293d05:** supersedes that draft: wrapper lifetime only, owned process-group cleanup on controlled exits, accepted SIGKILL/orphan gap handled by #583/#587; maintenance cutover with a live legacy-capability refusal; scan/store deleted in the same build. Removed native fd/ID integration, dual-guard stages and override. Added source-counted deletion estimate. Revised active plan and issue acceptance together so superseded promises do not remain closure requirements. This revision awaits TL approval before code.

- **2026-10-15 20:08 PDT, plan-quality PQ-1/2/3:** TL approval 659dde9c confirmed scope. Clarified best-effort unknown-ID handoff failures versus known-ID lock failure, named the pure lifecycle transition owner and IO effects, and compressed tests to named risk/guard strategies. Added recurring conformance and permanent inode cost measurement. No scope or guarantee expansion; code has not begun.

- **2026-10-15, implementation checkpoint:** Existing cold ResumeContext does not use prepared recovery, so post-park Relaunch provenance is carried in its existing persisted/validated RecoveryIntent.Relaunch field; exact source agent/ID must match. Shared selector parsing now retains NewNativeID separately from fork source. Notification failure preserves config/lifecycle work and reports diagnostics. These clarify approved mechanisms; no new user flag, native integration or scan.
- **2026-10-16, TL e93b109d (re-plan after #597/#600/#605/#607 landed):** supersedes the capability and migration parts above: the readiness capability, the live legacy-slot guard ("relaunch older slots first") and the confirmed-Relaunch migration marker are dropped, and no launch takes native admission. A resume whose lock another live wrapper holds refuses at native start naming the holder slot, the same "in use" refusal as Couch's registry check. Pre-#595 wrappers hold no lock until relaunched (accepted gap). The holder label resolves lazily on first lock; notices after stop or with unusable IDs are dropped; owned-group cleanup ends once no live member remains.
