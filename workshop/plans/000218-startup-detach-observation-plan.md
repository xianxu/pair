# Startup and detach observation implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Use bounded delegation for independent adapters/tests, main session for warm-context lifecycle integration. Steps use test-first verification.

**Goal:** Explain attach/detach latency, eliminate repeated host-wide argv inspection, and select parallelism from measured 6/20/40 results.

**Architecture:** Reuse native procutil enrichment and launcher ServerStates. A startup/detach pass owns one immutable bulk server observation; slot probes read that initial evidence and freshly revalidate the exact server around pane queries and before effects. Child Pair launches cannot inherit an in-memory Couch snapshot, so standalone server discovery enriches only native zellij candidates. No global cache or time-to-live.

**Tech Stack:** Go, Darwin sysctl/Linux procfs, existing Couch lifecycle/console reducers, controlled Zellij fixtures and PATH tracing shim.

## Approved scope and alternatives

The operator approved one snapshot per pass (10-14 issue revisions, dispatch receipt 4cb65884-fcfb-4a3b-a75b-a53020d42711). Prefer shared bulk discovery plus fresh targeted fences; candidate filtering also fixes the spawned Pair's independent checks. Candidate filtering alone is the authorized minimum but leaves avoidable table reads in the parent. Raising parallelism alone amplifies native IO and has no justification before reducing that work. Do not serialize an ownership snapshot into a launcher argument or persistent file: another process must recheck its own current authority.

Read-only trace found approximately twelve full snapshots in a healthy warm attach (parent checks/registration plus four child checks), and approximately six in a healthy detach. These are path counts, not timed claims; baseline measurement comes first.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| Native candidate selection (zellij or unknown short name; no terminal rows) | cmd/internal/procutil/command.go | new |
| ServerState and ClassifyServers | cmd/internal/launcher/session_servers.go | reused |
| SessionOwnerObservation / owner classification | cmd/internal/launcher/session_owner.go | reused |
| ReattachPass state and bounded limit | cmd/internal/couchtty/menu_reattach.go | reused |

The short kernel name is only a candidate filter, never positive executable/argv authority. Unknown names remain conservative candidates. Real zellij's short name is zellij on supported OSes; native command matching and per-read generation fences still decide membership. Exact argv-spoofed conformance helpers must have a zellij executable name when exercising the candidate boundary. Diagnostic inventory keeps the complete Snapshot API and its long-executable semantics.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| SnapshotNamed / shared snapshotSelected reader | cmd/internal/procutil/command.go | new | existing native table, command, identity reads |
| OSServerStates and SessionServers | cmd/internal/launcher/session_servers.go, session_quiescence.go | modified | named native snapshot and existing serverProcesses |
| Exact observed-server validation | cmd/internal/launcher/session_owner_os.go | new | targeted identity/argv/identity checks |
| SessionOwnerPass | cmd/internal/launcher/session_owner_pass.go | new | one ServerStates capture plus owner IO/probe |
| Pass-scoped artifact observation context | cmd/internal/couchcore/artifactcollision.go, session_owner_pass.go | new | existing checker Servers and OwnerProbe seams |
| Startup/leave pass wiring | cmd/internal/couchtty/console_reattach.go, console.go; cmd/internal/couchcmd/run.go; cmd/internal/couchcore/park.go | modified | existing lifetime/operation contexts and Leave fanout |
| Opt-in timing/live experiment harness | cmd/internal/couchcore/startup_detach_timing_live_test.go; launcher timing test if needed | new | controlled test-owned sessions, real launch/detach, trace shim |

SessionOwnerPass owns one immutable map and memoized capture error, scoped to the pass context. Slot cancellation cannot poison siblings' initial capture; pass cancellation stops the capture and all already-admitted work through existing contexts. No map is shared by mutation. Snapshot failure remains uncertainty and refuses rather than falling back to per-slot full scans. A cached missing row never authorizes creation, signaling, or retirement. Single actions outside a pass keep fresh standalone behavior.

The pass supplies initial server evidence only. Probe's after-pane check and Revalidate must not read the same cached value as a false fresh observation. Native targeted validation compares the exact observed PID/start identity and full socket argv, fences the command interval with identity reads, checks context, and refuses replacement/unavailable evidence. Existing stateful owner fake supports ordered disappearance/reuse/foreign ownership tests. Contested and unreadable snapshot verdicts never become a usable single server. Existing client identity/signal guards, conditional record updates, and #528's exact-generation quiescence are retained.

## Operating envelope and ordering

- ARCH-DRY: one ServerStates capture per parent pass, reusing ClassifyServers/serverProcesses; one native enrichment implementation. No hand-maintained alternate ps parser or session table.
- ARCH-PURE: metadata selection and server/owner projections remain pure. Snapshot IO, context scope and effect fences remain explicit integrations.
- ARCH-PURPOSE: enumerate parent startup, confirmStillDetached, sessionBindingForLaunch, registration polling, child launcher ownership/final revalidation, Leave precheck, Detach pre-signal and retirement checks. Optimize all broad native discovery consumers, while diagnostic inventory keeps its full observation contract.
- ARCH-MOCK: extend existing process-reader/owner/fleet worlds, not PATH-fake native process inspection. Real conformance uses a private 21-session fleet, a stand-in pane rather than a model process, and unrelated sentinels. Never operate the operator's live slots.
- ARCH-CONSTRAINTS: startup/shutdown batch path; operator's 21-slot case, host roughly 1,320 processes, current limit six. Baseline includes actual process/server counts. Full enrichment falls from host-process count to zellij/unknown candidates. One parent bulk capture is O(process count + zellij candidates); subsequent exact checks are O(1). Bounds/timing budgets remain existing query/lifecycle deadlines. 6/20/40 concurrency uses a test-only override; retain the production default unless measured latency/load supports a change.
- ARCH-SECURE: kernel/procfs responses and pane JSON remain untrusted typed evidence. #553 EINVAL/ESRCH/ENOENT and zombie semantics stay gone, but permission errors remain unknown. Snapshot identity is point-in-time evidence; no stale snapshot can authorize an effect. No argv or pane command contents in timing logs, only phase names/counts/durations.
- ARCH-ORDER: pass (created -> capture or error -> immutable observations -> complete/cancel). First worker captures once; siblings reuse. Slot cancellation stops that slot; parent cancellation stops all. A server exits/recycles/execs after capture: fresh exact check refuses. Contested state refuses. A server missing from initial evidence cannot silently become authority for a later replacement. Existing reducer queues, thread holds, and conditional revisions remain authoritative; no new lifecycle status.
- ARCH-FUNERAL: no durable cache. Pass map/context die with the pass/console lifetime. Test sessions/helpers have exact cleanup, bounded waits and an unrelated-survivor assertion. Timing outputs are opt-in stdout and temporary files owned by the experiment, summarized in the issue Log.

## Tasks and review boundaries

- [ ] M1 — Baseline and native candidate discovery: capture one real attach and production detach breakdown before changing behavior; implement named enrichment and fresh observed-server checks with red/green safety/count tests; verify process conformance and close M1.
- [ ] M2 — Shared pass and measurements: integrate one bulk parent capture across startup/Leave, test count/order/cancellation invariants, remeasure the same fleet at 6/20/40, select only supported concurrency changes, update atlas/timeout comment, full suite and issue close/land.

### M1 steps

1. Build the existing probes/zellijcalls shim; use unique PAIR_ZELLIJ_TRACE and run its report on controlled real lifecycle operations. Record shim overhead and distinguish aggregate subprocess duration from wall critical path. Do this before adding any stage instrument.
2. If native/other time dominates, add an opt-in timing fixture around SessionOwnerIO (SessionServers/Socket/SessionPanes/SessionPresent) and production Couch launch/detach stages. Reuse existing launchSteps and COUCH_TRACE where possible. Enumerate one attach and one detach from first entry to registration/client retirement, including residual wall time. Keep startup creation and cleanup out of attach/detach wall samples, but report their costs separately.
3. Run baseline at unchanged behavior; record process count, server candidates, call counts, monotonic durations, failures and host load. Persist exact commands, code hash, and attribution in the Log before optimization.
4. Add SnapshotNamed(ctx, name) using the same native snapshotSelected(ctx, reader, selector) core as Snapshot. Test unrelated row command reads are zero; selected/unknown rows inspected; zombie/gone/recycled rows omitted; unreadable selected evidence retained; canceled capture stops reads. Demonstrate red against all-row enrichment. Keep Snapshot and diagnostic callers unchanged.
5. Route OSServerStates and SessionServers through SnapshotNamed(ctx, "zellij"). Preserve #553 semantics and existing parser. Update real spoofed helper fixture to exercise the actual candidate name. Run native candidate conformance and Linux portable fixtures/cross-build.
6. Add native exact observed-server validation, shared by after-pane fencing and Revalidate: current identity equals captured identity, native command's exact session AND socket equal captured values, final identity still equal, cancellation refuses. Preserve fake SessionOwnerIO fallback for existing worlds, and explicitly test native targeted revalidator with mutable generations. Count zero host-wide snapshots at final fences; prove swapped PID/socket/argv cannot reach an effect.
7. Run targeted procutil/launcher/couchcore tests and race where appropriate. Update atlas for the new API and affected artifact ownership inventory. Commit and sdlc milestone-close M1 with measured evidence.

### M2 steps

1. Add SessionOwnerPass with an existing ServerStates source and SessionOwnerProbe. It captures initial bulk evidence once (lazily in worker scope), memoizes error, and supplies per-session initial verdicts to the common probe logic. Fresh after-pane and final effect checks always bypass the cache. Inject fake source and stateful IO to model capture/read/error/cancel/disappearance/reuse; deterministic count test at 21 slots must report one bulk capture.
2. Add checker pass context factory and Couch facade. Native checker uses its existing Servers source; fake/custom owner probes retain their explicit test seam. Scope binding to the exact checker/root so unrelated operations cannot accidentally consume a different pass. NamedPairSessionContext/recoverySession observe through it; RevalidatePairSession stays fresh.
3. Console owns a reattach observation context created at arm and passed to all background OperationCall contexts. Use the existing lifetime and reducer completion/cancellation; capture runs in worker scope, not during first-frame paint. Add injected pass-context factory via couchcmd wiring. Root attach remains a separate action or explicitly joins the same initial startup scope where lifecycle allows. Drop/cancel the pass context after all attempts complete or console closes.
4. Leave creates one observation scope before fanout and threads it through leaveOne/Detach/retirement. Keep selected single Detach fresh when no enclosing pass exists. Cover two simultaneous passes, one slot canceled while siblings continue, capture error without repeated fallback, stale missing/contested rows, and exact identity guards at signals.
5. Reuse the baseline controlled 21-session fleet. After optimization, run the same production reattach+LeaveDetach with temporary limits 6/20/40, at least three samples per setting in interleaved order. Record total wall, per-stage attribution, attempts/max concurrency, native table/argv counts, peak host load/CPU and failures. Sample top/uptime as measurement tools, never introduce production ps executions. Distinguish co-tenant host load from fixture demand. Restore override and terminate only test-owned clients/sessions. If a full safe fleet cannot exercise the child path, name the measurement limitation and get hub guidance rather than present a partial benchmark as complete restart evidence.
6. Select production parallelism only if latency and load data justify it; otherwise retain NumCPU/2 and explain. Update pairRegistrationTimeout's comment with the current measured phases/envelope instead of the historical unexplained multiple. Record unresolved residual honestly.
7. Merge fresh main regularly, preserving #528 and #553. Run focused/race, make -k build, then push and sdlc local-check run --name suite (main's declared full suite; no duplicate direct full run). Close M2, then whole issue close per SDLC, PR/merge, report exact hashes to hub.

## Verification commands

- go test ./cmd/internal/procutil ./cmd/internal/launcher -run '<candidate/owner/pass tests>' -count=1
- go test -race ./cmd/internal/procutil ./cmd/internal/launcher ./cmd/internal/couchcore ./cmd/internal/couchtty -run '<changed ordering/count tests>' -count=1
- PAIR_LIVE_COUCH=1 PAIR_LIVE_STARTUP_PERF=1 go test ./cmd/internal/couchcore -run '^TestStartupDetachTimingLive$' -count=1 -v (exact options finalized with fixture)
- go test -c -o /tmp/pair218-launcher.test ./cmd/internal/launcher, then separate sandbox-exec invocation for native owner/process conformance
- GOOS=linux GOARCH=amd64 go test -c -o /tmp/pair218-procutil-linux.test ./cmd/internal/procutil; portable fixture tests locally. Native Linux remains unavailable; do not start a VM.
- make -k build; sdlc local-check run --name suite (separate commands)

## Coordination

#528 pair:2 is landing changes to launcher session_quiescence/session_servers and couchcore ownership. Start from main (9a2aff687 at start-plan), merge main often, and preserve exact-generation authorization. Do not stack on its unlanded branch. Initial substrate refreshed to ariadne 7a722362; generated TL skill ignore diff parked separately, no lifecycle configuration changes in #218.

## Revisions

### 2026-10-14 — PQ-1/PQ-2/PQ-3: preserve fresh exclusivity and lifecycle progress

This revision supersedes M1 step 6 and the O(1) fresh-fence proposal: preserving the current authority contract requires proving that the original server remains the ONLY claimant, not merely that its PID/argv is unchanged. Reuse SessionOwnerProbe's existing after-pane and Revalidate SessionServers reads. Those reads use narrowed native discovery (zellij/unknown metadata candidates), retain exactly-one-server comparison including exact socket and generation, and keep #553's gone semantics. This intentionally performs a cheap fresh table/candidate enumeration at authority fences; it never repeats host-wide argv enrichment. No new direct-PID-only validator is introduced. The candidate set can change between observations: fresh discovery detects a competing server even when the original generation remains intact. Existing effect guards remain authoritative; no authorization contract changes.

SessionOwnerPass supplies only the initial before-probe value from one shared ServerStates map. The after-probe read and Revalidate always use live IO, never the map. A snapshot error or contested/unresolved row refuses. Missing initial evidence is not positive ownership. The pass is wired only to warm-only background reattach and LeaveDetach, not cold/park/create operations. awaitResumeRegistration explicitly bypasses any pass and uses fresh narrowed discovery: post-launch registration must discover new server generations rather than inherit prelaunch absence. Standalone root startup and child Pair keep their own narrowed fresh reads. This bounds the accepted performance claim: one shared initial bulk observation per parent pass, plus required fresh candidate-only authority checks.

The earlier case inventories in M1 steps 4/6 and M2 steps 1/4 are superseded by this compact executable test contract (retained as design history per the artifact revision rule):

| Risky function | Strategy and mechanical guard |
|---|---|
| processNameCandidate(Process, name) | Metadata-selection property tests ensure no known unrelated process is selected, while unknown names preserve uncertainty. |
| snapshotSelected(ctx, reader, selector) / SnapshotNamed | Stateful reader drives generation changes and read failures; independently assert exact selected read count, gone omission, error retention, and cancellation boundary. |
| SessionOwnerProbe.Probe / Revalidate | Stateful owner world inserts competing claimants and changes generation during pane IO; exactly-one-and-equal original server is the authority invariant. Revalidate always reads live IO. |
| SessionOwnerPass.Probe | Controlled concurrent callers share one source capture; live after-reads introduce new claimants, proving cached equality cannot masquerade as exclusivity. Capture failure/cancellation never grants ownership. |
| ScopedThreadArtifactCollisionChecker.NamedPairSessionContext | Pass-bound reader count plus independent owner identity invariant; contexts from different roots/passes cannot share evidence. |
| awaitResumeRegistration | Controlled birth after an initially absent snapshot must become observable through fresh discovery; old absence never causes warm/create authority. |
| Console.runBackgroundOperation / Couch.Leave | Real worker-pool/fleet worlds use barriers for cancellation and completion ordering; assert one initial capture per pass and unchanged exact-effect guards. |
| Couch.Detach / retireDetachedIncarnation | Stateful generations change between pre-signal and record retirement; signal and conditional retirement require current exact client/server authority, even with shared initial evidence. |

Non-goals: no ownership/contested-name relaxation; no process snapshot serialization to child Pair; no permanent cache; no unconditional parallelism increase; no new supported OS; no optimization of diagnostic writer inventory; no operator-session restart experiment; no redesign of #528 quiescence.
