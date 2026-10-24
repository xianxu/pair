---
id: 000866
status: working
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours: 5.5
severity: high
card_mirror: '24b09c75df20a849af40816d6c8c35055c3a90b2' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T02:48:13-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Wait for Zellij readiness before startup probe clients

## Problem

During pair#863's few shared disposable native runs, Zellij 0.45.1 twice panicked at zellij-server/src/lib.rs:1462:26 (Option::unwrap on absent session_data). A startup probe can connect/remove a client before session metadata exists. Pair should investigate a readiness-qualified probe so its own startup observation does not trigger that upstream lifecycle window. Exact standalone causation is not yet proven; this is the bounded Pair-side avoidance issue, not an upstream bug report.

First occurrence: pair#863 source25a6c4847 native Claude fresh primary startup, before a wrapper started; full matrix FAIL456.39s, exact cleanup passed. Retained source-branch evidence:
- workshop/evidence/000863/native-first-fixed.log, SHA2562320f9563c197a28f32160e7150bcf7e0d685be60ed1896f28aa760fbde4e38c
- workshop/evidence/000863/fresh-fixed-claude-zellij.log (full server capture).

Second occurrence: pinned main4053f3da995746efcc1c6296cffbde4a67a7f343, baseline explicitly started its primary through the operator form, then initial worker1 startup failed before any switch-agent step. TestDisposableStartupArguments FAIL42.33s (Claude34.11s); Pair exited1 before registering. The retained server log contains the same lib.rs1462:26 panic at02:44:40 PDT on2026-10-24. Exact owned generations absent and direct child/IO joined; Codex skipped by existing test loop. Evidence workshop/evidence/000863/main-startup-arguments-4053f3da.log, SHA256d17dbd817ceb9225e85e8ec6ab48f5c10b465af8f2e8390eb2d9e17570692497.

The evidence is committed on pair#863's pushed branch (latest evidence7a71f3353), not yet main; this issue has no dependency on #863 and can use git show that branch. Do not confuse these panics with #863 BR-2's different post-startup switch-agent adoption failure.

## Spec

Diagnose Pair's startup probing against an owned disposable Zellij session. Investigate a reliable readiness signal before a probe attaches or detaches a client; preserve caller-owned startup deadlines and conservative no-answer behavior. No timeout increase, hot retry loop, general session ownership weakening, live-fleet manipulation or unrelated Zellij patch.

Installed binary /opt/homebrew/Cellar/zellij/0.45.1/bin/zellij. Verified local tagged source /Users/xianxu/workspace/zellij, v0.45.1 commit efd8fd5a89a20c07a111d248ad7fce53848d2c18. In zellij-server/src/lib.rs1462, regular RemoveClient removes the client then unwraps session_data while it can still be None. Inspect the surrounding FirstClientConnected initialization and Pair's startup session probing paths. Pair arguments/environment were not shown malformed; a timing interaction is the hypothesis.

## Done when

- Retained evidence identifies the Pair probe path and the readiness ordering that can trigger or avoid the early client-removal panic.
- A deterministic stateful-fake/control proves no attach/detach probe before the selected readiness witness; uncertainty remains conservative and bounded by the caller's deadline.
- A bounded shared disposable native qualification shows startup without the panic, with exact owned cleanup; preserve failing captures and do not claim a calibrated flake rate from two occurrences.
- Record whether a standalone upstream reproducer is available; do not file an upstream report without one.

## Plan

- [x] Approve the readiness/coordination contour in [the durable plan](../plans/000866-zellij-startup-readiness-plan.md), then enter change-code; no estimate before its gate.
- [x] Implement one readiness authority and bounded shared admission with deterministic regression controls and normal bypass prevention.
- [ ] Qualify shared disposable native startup, retain exact cleanup/failure evidence, record pre-close verification, then close/land through the gates.

## Log

2026-10-24: Filed unclaimed at TL8029db79's direction, for pair:5; do not wait for #863 to land. Both native failures and source pointer retained above. No production workaround in #863.


### 2026-10-24 — planning investigation and parked #722

Claimed #866 and started its own main-based branch through SDLC; #722 was checkpointed/pushed separately and remains SHIP/codecomplete awaiting its watched morning rerun. start-plan catch-up deferred stale merge-check.yml rather than changing protected config. No production code or gate configuration changed.

Inspected tagged Zellij source efd8fd5a via git show (checkout HEAD differs). listener accepts probe clients before metadata initialization; list-sessions ConnStatus does not certify readiness and removal reaches the None unwrap. Existing Pair create Admit already waits for exact agent readiness, but global ListingPoller and concurrent launcher observations are not excluded by that local wait. The exact historic probing process is still unattributed. First executed pane or actual server Render is a valid witness; no universal passive, current-generation readiness cache/CLI was found. Proposed shared namespace startup/probe exclusion and explicit retained uncertainty in the durable plan; needs TL approval before code.

TL8db7cb72 supplied the third capture at workshop/evidence/000863/branch-required-native-7a71f335.log on #863's remote branch; inspected bytes/hash217ecb09534658cc33ca98bdc08f71ff87141e1305d0ca28b8ba42af424f0f11 and the same pre-initialization1462 panic. No flake-rate claim.

A private direct-server diagnostic started under /Volumes/TestScratch/p866-kmgx1mqo, then its Python ps parser raised ValueError on an unrelated command's unmatched quotation, before the test query and cleanup. Reported gap to TL2d29a1aa (submitted); per TL78d1179c stopped direct socket/process probing. TL5ead059e (submitted) confirms exact private server21479 terminated and no other matches. No standalone reproduction result is claimed and no upstream issue filed. Further evidence comes from documented CLI/output and shared native harness only. Root cause research is source-supported, not proof that a specific old process sent the trigger.

### 2026-10-24 — admitted implementation; readiness core checkpoint

Consolidated plan-quality round2 CLEAN disposed PQ-1/PQ-2. After deriving and setting the5.5h card estimate, change-code rechecked the unchanged mechanism CLEAN in round3 and estimate-quality returned INFO (allocations optimistic, not fabricated); admission succeeded. The estimate includes regression/bypass controls inside module/integration/migration allocations and whole-issue research, not only remaining work. No additional mechanism was requested or added.

Implemented WIP core: shared stable-file lease extraction (launcher API delegates); canonical namespace/marker path constructors; positive marker codecs sharing the pure panebirth LaunchIdentity; complete passive marker qualification; CreatorGuard and scoped synchronous preflight permits; Source.readAttempt exclusion before injected runners; bounded-command admission/argv classifier; additive pane-sidecar codec/publisher; early internal pane-ready CLI route; creator handoff that releases namespace exclusion after exact pane identity and before unchanged AgentReady. This is **not yet feature-complete or safe to stage**: layouts and remaining raw native commands are not migrated, retirement/inventory enforcement is incomplete, and no native qualification or recorded pre-close check has run.

Behavioral RED: the Source regression originally reached the independent initializing-native scanner (probe removed before first pane); stale readiness still admitted3 scans. Both now GREEN. Creator controlled test independently withholds/stales the pane sidecar, then observes ordinary guarded read admission while AgentReady is absent; bypassing publication made that test fail and restoring it passed. Pane codec and publisher tested missing APIs first, then GREEN; the early CLI route initially panicked through a nil launcher runtime, now returns its own failure before runtime setup. The previous close-on-exec test moved unchanged to the shared lease package rather than expose descriptors as API.

Verification at checkpoint: go test ./cmd/internal/zellijquery ./cmd/internal/panebirth -count=1 PASS (17.914s/0.126s); launcher targeted TestCreatorPaneReadinessReleasesBeforeAgentReady, TestPanePublisher and TestTerminalLease PASS0.747s; pair-go targeted internal pane-ready/exec-alarm routes PASS0.242s; git diff --check clean. One existing child-socket test initially failed because its private socket lacked a positive marker; its fixture now publishes exact private readiness and the complete query package passes. No full launcher package, native run, suite or close is claimed. All targeted terminals exited; no queued/running recorded checks or merge runner.

**Next:** wire/test the internal zellij-command adapter and first-pane-before-rename layout commands; migrate the approved bounded Go/shell/Lua caller inventory and ExecKillSession helper; update three direct retained_ready_test.go create fixtures to acquire a private guard (their fake Acquire currently bypasses the new handoff authority). Then finish store/format/capacity/owner-death/alias/cancellation/linear-work tests, exact retirement and artifact lifetime wiring, catalog/negative controls and affected-reader inventories before shared Claude/Codex native qualification and recorded pre-close. Do not add a fourth mechanism revision: bring a real design gap to TL per fbb6fc81. #722 remains independently parked for the watched rerun, with no stage authority from historical messages.

### 2026-10-24 04:02 PDT — resumed integration and approved recovery narrowing

Resumed from the readiness-core continuation using sdlc state, own pair:5 branch. Added early internal zellij-command routing, literal argv/native exit tests and signal cancellation that joins its native child before releasing exclusion. Both layouts now execute the non-native pane publisher before guarded rename. Bounded Go, shell and Lua callers migrate through the shared command seam; title background mutations stay qualified. Restored retained-readiness fixture authority by acquiring the production private creator guard rather than weakening Check.

Integration exposed a design gap: mandatory marker qualification of recovery's SessionPresent refused before exact-authorized KillServer. Submitted TL delivery e39af197-9300-46c9-8503-a328080ebb48 approves narrowing marker refusal to automatic background probes, preserving explicit recovery on unmarked endpoints under the same lease and accepting its outside-window upstream risk. Appended governing plan revision. ARCH-DRY: no new resource or bypass environment. Shared Source now has trusted in-process ExplicitRecovery admission; ordinary observations remain qualified.

RED/GREEN: early helper fell into nil runtime; native termination control without signal handling returned with its child alive, then passed after cancellation/join; private unmarked recovery originally refused at marker qualification, then passed after explicit caller wiring. Guard controls reproduced and corrected dangling namespace symlink aliases, including relative symlink/parent traversal. Complete query/panebirth race packages passed after owner-death, capacity, strict additive marker and counted 8/16 production-seam controls. Seven caller admission regressions and clip shorter-deadline control passed; clipboard shell fixture passed after private namespace isolation. Targeted launcher/CLI routes passed. make build completed in this slot; runtimebundle package passed. Full launcher and pair-go package checks are still in progress; no native qualification, staging, recorded pre-close or close is claimed. Marker lifetime integration, catalog enforcement and final aggregate verification remain outstanding.

### 2026-10-24 — bounded consumer integration checkpoint

Early adapter, layout publication ordering, complete bounded caller migration, explicit recovery admission and exact-generation marker retirement are wired. The shared lease, strict marker/pane codecs, owner-death controls, canonical dangling-symlink resolution and counted 8/16 qualification pass complete query/panebirth race packages. Marker retirement freezes the environment across the authorized callback and synchronizes the containing directory after exact removal. Lifecycle and recovery wrappers invoke it only after their existing proof; no observation gains cleanup authority.

The finite bypass rule lives in Pair's existing **internal/repolint** catalog extension (shared pkg/golint is upstream), with Go/generated-shell/Lua/KDL and production-mutation controls. That package and exhaustive artifactpath coverage passed. Both readiness families have exact resolver/source/retention ownership; generic GC cannot unlink them. Lua adapter uses launcher PATH for copied/runtime-bundle installs rather than assuming an executable inside PAIR_HOME. Script/helper/clipboard and restart/quit fixture checks passed; complete pair-go/runtimebundle and wrapcmd checks passed. Launcher first complete run found inherited-live-namespace fixture contamination; TestMain now isolates native namespace/cache, and the remaining deliberate private socket fixtures publish exact positive evidence without weakening retry/traversal assertions. Final launcher and couchcmd complete checks are running; native qualification is authored/compiles but unrun at this checkpoint. No stage, pre-close or close is claimed.

**Next:** run the bounded shared native qualification on this integrated source for both agents, preserve full server/probe evidence and exact cleanup, finish any deterministic/fixture failures, reconcile atlas and affected-reader inputs, then recorded pre-close and binary-owned close review. #722 stays independently parked.

### 2026-10-24 — first native attempt: pre-start harness mismatch, exact cleanup

Native qualification at source a14881593ed58eeabcf58e12863c15208d39bc66 failed before any Pair startup: TestDisposableZellijStartupReadiness Claude245.83s/full278.52s, Fleet.Start waited for Agent0 while the attachment-only startup UI offered `hub · Ctrl+Space: start`. There were241 empty native listing samples, no wrapper records and a zero-byte server log; no readiness/panic avoidance coverage is claimed. Shared testcouch confirmed all exact owned generations absent and direct child/IO joined. Full capture [native-startup-readiness-a14881593.log](../evidence/000866/native-startup-readiness-a14881593.log), SHA256e213c94e337e0a8cb59fc3f2f499fc2aa9fd54f7863d9869b2861ee68b0ff864.

Authenticated submitted TL a4eabbf3 authorizes driving the existing public start form in this qualification rather than waiting for #863's unlanded shared harness repair. Submitted pair:6 measurement8c904b9c corrects the exact sequence: wait for switcher root, Ctrl+Space to form, input primary path, Down requests preview, choose agent, wait `args  repo-default`, then **one Enter** starts. There is no resolve Enter and never a repeated Enter. The test now drives that observed form while joining Fleet.Start's existing wait; no production harness, private native owner or readiness mechanism added. Rerun after this checkpoint. All certificate changes were audited per reader; scanner registration adds only the new test names. Complete launcher passed178.379s, wrapcmd135.718s; three Couch fixture contamination failures passed targeted after private isolation. Explicit controlled-native fixture observations now retain exclusion-only test admission after their existing screen-render witness; ordinary probes still refuse their unmarked endpoint.

### 2026-10-24 — shared native startup qualification PASS

At source6f05e60447f4244d6ed438a6fa9925e27330604b the public-form qualification passed59.68s: Claude23.98s and Codex27.29s (startup windows18.260s/21.537s), each with real Zellij0.45.1, native fixture agents, primary+worker exact pane/namespace marker/AgentReady identity checks, and concurrent bounded automatic listing traffic. Complete bounded native logs contained no panic; warning/retry-queue lines and teardown's mouse-mode IO errors remain retained rather than filtered. Both owned fleets confirmed all exact generations absent and direct child/IO joined. This is a bounded sample, not a flake-rate estimate or interactive/crash-case coverage.

Full passing capture [native-startup-readiness-form-6f05e6044.log](../evidence/000866/native-startup-readiness-form-6f05e6044.log), SHA2563ed9ae274e99f983b46d58fb28f6e2c095048e929fe5e784d7ab308a6dd91882. The earlier pre-start failure stays separate and byte-for-byte intact (terminal padding makes raw evidence fail whitespace-only diff checks; source checks exclude these captures). No standalone upstream reproducer succeeded; no report filed. Fresh combined targeted packages passed zellijquery15.545s, panebirth0.135s, pairlifecycletest1.810s, pair-go15.003s and repolint14.101s. Interactive negative control passed its race run3 times; it demonstrates the accepted uncovered overlap rather than certifying it safe. Final affected-reader audit passed after individual input reviews and minimal scanner-name registration. Atlas now maps the implemented boundaries and limits. Next: recorded pre-close on the pushed head, operator smoke coordination, one binary-owned close review and landing.

## Revisions

### 2026-10-24 — proposed shared readiness exclusion

The existing local wait alone cannot protect a newly created session from global observers. The proposed plan adds a namespace-level creator/probe lease and bounded retained uncertainty, using exact existing pane readiness and explicit supported retirement. This is a scope/coordination proposal, not implementation authorization; alternatives and state/recovery table are in the durable plan. Await TL decision, then change-code before production edits.


### 2026-10-24 03:20 PDT — TL comparison request; marker-first revision pending

Submitted TL e242a4dc does not approve the initial shared-lease/unfinished-intent proposal. The durable plan now appends a governing marker-first comparison and withdraws the separate unfinished-intent store. Source-only enumeration confirms that global listing AND named helper actions call get_active_session/get_sessions and assert_socket across the whole namespace before honoring -s or ZELLIJ_SESSION_NAME. Public named attach/watch/kill/delete do likewise; only internal Rust primitives bypass enumeration, with no public local CLI switch found. Thus a marker-qualified target does not stop its CLI from probing an unready neighbor. A precheck also races endpoint publication without exclusion.

The revised proposal uses positive exact-launch ready markers, retains conservative starting/unknown projections for unmarked sessions, and keeps a lease only around the native calls that cannot avoid global scanning. It names the still-unresolved crash-before-delayed-endpoint-publication and stale/external-generation cases: inspect existing launch lifecycle authority before code, report a concrete gap rather than invent a replacement journal. No marker, lease, scanner or production change is implemented. Await TL review of the revised comparison; original initial-plan implementation steps are superseded.


### 2026-10-24 — narrower design approved with creator-crash limit

Submitted TL734d1a09 approves positive exact-launch readiness markers plus a kernel-held file lock only around globally scanning native CLI calls and the creator's spawn-to-exact-pane-ready/publication interval. Marker/cache reads stay lock-free; no journal. Creator death automatically releases the lock, and a scanner may then probe that creator's incomplete server: the original upstream crash risk remains bounded to that creator-crash case. This accepted limit supersedes the earlier demand to close every owner-death gap with persistent state. It is explicit in the durable plan's final concepts, state table and execution steps. No production code yet; proceed to change-code and its gates.

### 2026-10-24 — blocked plan gate; consolidated third revision for TL

change-code round1 refused implementation on PQ-1: interactive attach has no scan-complete handoff witness, and ExecKillSession's syscall.Exec would close the ordinary CLOEXEC lease before native scanning. PQ-2 requests explicit nested unknown-field preservation. The bounded-call sweep also found bootstrap rename-pane before wrap/AgentReady, which would deadlock a guard released only at AgentReady. No production code was written and no estimate was set.

Submitted TL c8f6e5d4 permits bounded-command/creator coverage with a declared interactive-attach residual if no attributable existing output witness holds. Source inspection found native startup bytes before connection, a mixed Pair PTY stream and an unattributable shared log; no new output decoder/logging namespace is proposed. The durable plan appends the requested consolidated third revision: publish exact first-pane evidence before rename, retain separate AgentReady semantics, explicitly own/join bounded native commands, exec a guarded Pair helper for kill, and leave interactive attach/native switching unguarded as a known limit. It includes the full Go/shell/Lua bypass inventory, immutable marker publication plus nested unknown-field tests, state/recovery rows and accepted creator-death limit. Await TL decision on this one consolidation, then rerun change-code; a further substantive correction requires stop/re-scope. #722 remains parked, unchanged and ready for its separate watched rerun.

Submitted TL fbb6fc81 approves this consolidated third revision and directs change-code. If plan-quality asks for another mechanism, bring it to TL instead of adding it. Interactive attach/native reconnect remain the approved residual race; exact first-pane publication precedes rename, creator exclusion ends before the unchanged AgentReady wait, and exec replacement uses the joined helper. No code authority is assumed before change-code admits.

## Estimate

*Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only.* The calibration source is marked stale by sdlc estimate-source, so these hours are provisional. Derived only after plan-quality round2 returned CLEAN and disposed PQ-1/PQ-2; estimate is not a design approval substitute.

Decomposition: issue/spec research and three approved contours; one guard/admission module; shared kernel-lease extraction; additive first-pane publisher; native-command/helper integration and stateful fake; complete Go/shell caller migration; Lua adapter migration; one shared native qualification/discovery budget; atlas/inventory evidence; one close boundary. Existing x/sys/unix flock, osfs metadata, artifactpath, panebirth and testcouch cover the library check; no new native protocol or PTY library is assumed. Guard design uses the v2.1 library halving, then the thorough-plan x0.2 discount; other implementation designs use x0.2. Authoring research is counted once without discount. Familiarity1.0 reflects the existing launcher/harness seams; implementation values already include v3.1's40% scale, with no second scale. The design buffer is15% for the approved detailed plan. Native startup uncertainty remains a qualification risk, not permission for a new mechanism or timeout change.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=1.0 impl=0.12
item: greenfield-go-module design=0.2 impl=0.32
item: smaller-go-module design=0.04 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: api-integration design=0.4 impl=0.6
item: cross-cutting-refactor design=0.2 impl=0.2
item: lua-neovim design=0.4 impl=0.6
item: real-api-discovery design=0 impl=0.24
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.2
design-buffer: 0.15
total: 5.5
```

Design2.38h x1.15 + implementation2.76h =5.497h, rounded5.5h of focused ship time. This includes the one mandatory close review, not a redundant milestone boundary.
