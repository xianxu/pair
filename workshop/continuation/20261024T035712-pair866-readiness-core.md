---
type: continuation
slug: pair866-readiness-core
agent: codex
created: 2026-10-24T03:57:12
branch: 000866-zellij-startup-probe-readiness
worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
issues: [866, 722]
---

# Continuation: pair866-readiness-core

## NEXT ACTION

Continue #866 from pushed WIP b256988a4cae48fa5ab73e37b49200bc81a9c15f on 000866-zellij-startup-probe-readiness, own pair:5 checkout. Read the issue Log's readiness-core checkpoint and the durable plan's **consolidated third revision** first. Wire the early `pair internal zellij-command -- <native argv>` route through existing zellijquery.RunNativeCommand; add RED/GREEN route/exit tests. Then change the two layouts to run existing `pair internal pane-ready` before any guarded rename, and migrate the finite bounded Go/shell/Lua inventory. This completes the consumer wiring missing from the tested core; do not native-test or stage the present incomplete head.

## State of play

#866 is working, claimed pair:5, full-flow change-code **admitted** at1d05c58b2. Plan rounds2/3 CLEAN disposed PQ-1/PQ-2; estimate5.5h derived via v3.1, estimate-quality INFO (optimistic allocations, not a blocker). Latest pushed source checkpoint b256988a4 is clean; no pending code edits, active subagent, targeted terminal, recorded check, review or merge runner. No native qualification, full launcher package or pre-close check yet. Use sdlc state for workflow recovery, not a new start-plan/branch.

#722 is independently parked SHIP/codecomplete on 000722-couch-drain-fast-a-fleet-wide-checkpoint-and-stop-before-a-restart-with-a-drain-board (parked latest6804aca08, PR477). It awaits the operator-watched rerun; pair:0 held #739's stage at last TL report. Do not stage/merge722 from old messages; await fresh GO. Its tested async delivery fix remains separate. All old train/622/744/775/778 work is historical, not current dispatch.

## Thread arc & user model

TL wants the smallest credible avoidance of Zellij0.45.1's startup RemoveClient panic, supported by actual source ordering and deterministic controls. The first global lease plus durable unfinished-intent proposal was rejected. Marker-only checks cannot stop the public CLI's hidden global scan or publication race. TL approved a kernel file lock plus positive exact-launch markers, explicitly accepting creator-death and non-cooperating-client limits. A plan review then exposed interactive/exec lifetime gaps and a real rename-before-AgentReady deadlock. The approved third consolidation resolves bounded commands/creator ordering while explicitly leaving interactive attach/native reconnect unguarded. Do not grow a fourth mechanism: report a real design gap to TL for re-scope.

## Artifact map

Read workshop/issues/000866-zellij-startup-probe-readiness.md first; its latest Log gives exact passing targeted commands and unfinished integration. workshop/plans/000866-zellij-startup-readiness-plan.md is append-only history: **only its last consolidated third revision governs**. The initial pending journal/reducer/store design is rejected. The plan-gate sidecar records PQ-1/PQ-2 addressed and no open findings.

Core code is cmd/internal/zellijquery/{lease,readiness,readiness_os,command}.go. ReadyIdentity aliases panebirth.LaunchIdentity to avoid duplicate identity codecs. AcquireCreator owns a kernel namespace lease; Preflight lends a callback-scoped non-serializable permit; Prepare runs after existing authority checks/before spawn; PublishReady accepts only the exact identity, persists positive evidence and releases. Source.readAttempt now acquires ordinary exclusion before all runners, including injected Run. WithProbe wraps synchronous bounded native work with the existing5s/caller bound; RunNativeCommand is the shell/helper argv seam. It already refuses top-level create/attach/watch/subscribe/server commands. Marker/cache reads themselves do not acquire a lease. The launcher AcquireTerminalLease delegates to extracted AcquireFileLease; its old FD_CLOEXEC assertion moved unchanged to zellijquery/lease_test.go.

cmd/internal/panebirth/record.go supplies strict8KiB additive pane and identity codecs with unknown fields at both levels. Legacy pane records decode but cannot qualify exact readiness. cmd/internal/launcher/pane_ready.go PublishPaneReady validates inherited identity, canonical namespace and artifact binding, then atomically writes the existing pane sidecar without a native call/lock. cmd/pair-go/main.go already handles internal pane-ready before resolving launcher runtime. Layouts are **not updated yet**.

cmd/internal/launcher/session_handoff_os.go now takes per-session lease then creator namespace lease, wraps Check in Preflight, calls Prepare before Start, waits for exact pane sidecar identity and releases namespace before the unchanged AgentReady/admission wait. creator_readiness_test.go has an independent controlled proof. Three old retained_ready_test.go direct fixtures still bypass Acquire and need proper private guard setup: TestColdLaunchChecksCrossAgentSurvivorUnderLease, TestColdLaunchUnknownReadinessRefusesUnderLease, TestColdLaunchAcceptsProvenDeadRetainedAgent. Preserve their authority assertions; do not loosen strict Check for fixtures.

The relevant Zellij source is /Users/xianxu/workspace/zellij at **tag efd8fd5a89a20c07a111d248ad7fce53848d2c18** via git show; current checkout differs and must not be edited/switched. Startup metadata Some precedes first pane execution; cache/listing/initial terminal bytes are not initialization proof. Three historical panic captures are pinned by path/hash in issue/plan on pair:7's unlanded863 remote branch. Exact historical triggering Pair process is still unattributed; no standalone upstream reproducer succeeded and no upstream issue is authorized.

## Decisions & dead ends

Authenticated TL fbb6fc81-bd9f-4e79-84e9-823bd9adbf7d approved the consolidation and change-code. TL c8f6e5d4-f950-483e-ab13-4be0d8f7562d specifically permits bounded-command/creator-only coverage rather than adding an interactive output witness mechanism. TL734d1a09 accepts kernel owner death releasing the lock and exposing the original crash-only upstream risk; no journal/orphan latch is permitted. A mixed Pair PTY stream is not an attributable native scan-complete witness; native alternate-screen output follows lookup but precedes connection. No new PTY, decoder or private attach logging namespace.

Do not hold exclusion until AgentReady: current layouts rename-pane before starting wrap, so that would self-deadlock. First actual pane execution publishes the exact identity before rename, and creator release precedes the separate agent-ready wait. ExecKillSession's direct syscall.Exec closes a CLOEXEC lock: migrate native kill through an exec-replaced Pair helper that holds its own ordinary lease until the native child joins. Preserve non-native PAIR_KILL_CMD overrides and non-returning exit semantics.

The earlier private direct-server diagnostic failed before its query and briefly left a private server; TL cleaned it. Per TL78d1179c, do no more direct socket/process-owner research experiments. Product code may use its existing authority seams; tests use portable private files/fakes and the shared testcouch native fleet only. Preserve uncertainty and exact cleanup; do not claim an observed cyber-filter denial (TL described one, actual tool trace was a parser exception).

## Live deliberations

This is a tested scaffold, not a finished safety guarantee. Remaining work besides command migration: exact marker retirement and resolver/retention manifest wiring; safe file/inode/alias cases (canonicalNamespace's nonexistent symlink/parent spelling deserves scrutiny); owner-death subprocess tests; nested marker round trips; capacity and counted8/16 guard work; complete catalog/AST bypass controls for Go, shell/layout and Lua; affected-reader certificates after auditing inputs; native Claude/Codex qualification; recorded pre-close and binary-owned close review. Current guard has128-current-session per-namespace capacity and256 stable lock stripes per approved consolidated plan. Do not silently re-scope that policy.

The entire zellijquery/panebirth package passed at checkpoint (17.914s/0.126s). Launcher targeted creator/publisher/terminal-lease tests passed0.747s, pair-go early internal routes passed0.242s, diff check clean. The original private child-socket test failed because its fixture lacked a new positive marker; it now publishes exact private readiness and the full query package passes. No known full-suite result exists for this head.

## Lessons learned

Enumerate actual native callers: guarding zellijexec.Run alone misses Source.Run hooks, raw Go exec, layout shell and Lua. The plan lists the finite inventory. Keep one guard authority and no mutation retries. Test ready-file identity separately from native state: the production Source regression was behaviorally RED by admitting a probe while the independent fake was initializing, then GREEN after wiring guard admission. The creator test similarly rejects stale evidence and releases before AgentReady; bypassing publication was observed RED.

Use whole bare couch/sdlc calls, authenticate incoming receipts and wait for submitted reports. Agent trailer this session is codex/gpt-6.1-sol/medium. No protected .sdlc/workflow changes, no operator label, no extra clone/worktree. Before each close run make test-affected-inventory and audit each named reader before repairs. Recorded pre-close is sdlc local-check run --pre-close; no manual full-suite bypass. Any substantive fourth mechanism request goes to TL rather than another patch loop.
