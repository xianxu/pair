# Title process lifetime and memory implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Use superpowers-executing-plans for the session-warm lifecycle work; bounded investigations may use subagents. Steps use checkbox syntax.

**Goal:** One title process per live wrapper, prompt title exit when its owner dies, measured and bounded memory work, and identity-fenced cleanup of proven dead-thread leftovers.

**Architecture:** Move title launch from the short-lived launcher to the actual wrapper. Give the title only the read end of a wrapper-owned pipe; EOF cancels its complete lifetime, including external commands. Measure Go allocations before choosing a memory change; reuse existing inventory/activity mechanisms where profiling confirms they address the cause.

**Tech Stack:** Go, Unix close-on-exec pipes and process identity, existing titlepoller/sessioninventory/threadactivity boundaries, macOS heap/vmmap, runtime/pprof in a controlled diagnostic process.

## Approval and evidence gates

This is a proposal for TL review, not implementation authorization. No production or test source has been edited. TL requested the plan before code. Do not run change-code or implement until TL answers this plan. A memory fix requires attributed Go heap/alloc evidence; the current native heap snapshot does not satisfy that attribution. Final memory entities and budgets must be appended as a plan revision after attribution, before implementing that part.

Coordination with pair:4 on #583 is pending (message 6a93aca7-49d2-4fff-8a4f-07677803c112). No shared cleanup changes or live cleanup until its identity/confirmation contract is reconciled. Do not stack on #583's unlanded branch.

## Measurements and gaps (2026-10-16, about 12:55 PDT)

- Live title PID 25522: argv identifies tag 1-ops-33 / claude / session 📁1-256; started Oct 15 at 23:27:36, PPID 1; ps reported 289472 KiB RSS. The corresponding wrapper PID 43773 is still running. Thus PPID 1 is not proof of a dead thread.
- heap and vmmap both succeeded on this live title: current physical footprint 24.5 MiB, peak 271.8 MiB. vmmap reported 269.9 MiB resident in Untagged regions, of which 22.1 MiB was dirty. Native malloc allocations total only 66640 bytes in 770 nodes. These are different measures; no claim that 280 MiB is retained Go heap follows.
- Raw evidence is local/private: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-617-evidence.zbitEI/{ps,heap,vmmap}.txt. SHA256: ps 3973586a024ff5b2d223742ae7a6c73fd59c34abeec36fe8759aa9c6234839b7; heap 5fcc6b918901a407cef2a9256c4a8b5330dc409130029bc39e2b4683e26efb09; vmmap 0a2910e79912fe4f92fee228ebe99f2169a86362de0f51f3b9e1b7e40076906b. No raw transcript or memory dump goes into the public repository.
- Existing binary has no discovered Go pprof listener, heap-dump command, or safe profile signal. Never send an unhandled signal to obtain a profile. The native heap utility does not enumerate Go allocation stacks. A profile of a controlled process running the actual production poll path is feasible, but is explicitly different from the original live process. Ask TL to approve that substitute, or a specifically coordinated diagnostic title rollover, before claiming the requested live Go profile is fulfilled.
- Code evidence: launcher/osruntime.go SpawnTitlePoller starts a detached sidecar on create and attach. Its owner is the launcher, not the wrapper. titlepoller/run.go checks session liveness every 60s, debounced five misses; this does not observe wrapper death. Existing single-instance protection is pidfile plus an argv substring, not generation ownership.
- Memory candidate, not proven cause: titlepoller/runtime.go constructs two fresh inventory runtimes across SessionActivity and ContextCount; contextcmd.Run rebuilds another runtime. QuerySession reads/decodes the full owner ledger/catalog; TokenUsageForRoot scans the whole authorized transcript on active ticks, allowing arbitrarily large individual JSON records. No retained transcript tree was found. Frame cache holds per-pane short strings but currently never prunes departed pane IDs.

## Alternatives

1. **Recommended: wrapper-owned pipe.** EOF is a kernel lifetime event even on SIGKILL. No polling process table and no later reaper is needed. Requires moving launch responsibility and auditing inheritance/re-exec.
2. Wrapper-held flock watched by title. Supports permanent per-owner inodes, but introduces periodic lock probing, replacement fencing and artifact lifecycle. Reuse only if #583 establishes a suitable shared primitive; do not introduce a second lease registry merely for this title.
3. Title observes wrapper PID plus start stamp. Small change but requires bounded periodic checks and preserves probe uncertainty. Keep for explicitly fenced diagnostics or compatibility; do not use session-name/PPID checks as lifetime authority.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| TitleOwner | cmd/internal/titlepoller/owner.go | new |
| TitleLifetime | cmd/internal/titlepoller/lifetime.go | new |
| LegacyTitleDecision | cmd/internal/titlepoller/legacy.go | new |
| SessionEnv | cmd/internal/titlepoller/runcli.go | modified |

TitleOwner is an exact scope/tag/agent/session and wrapper PID/start identity, parsed at the boundary. One wrapper owns at most one title. SessionEnv remains the sole title launch environment declaration; extend it positionally, update every caller, and strip the private lease descriptor variable from unrelated descendants. No new issue/status lifecycle is introduced.

TitleLifetime owns legal state transitions. States: awaiting-owner, active, stopping, exited. Events: lease-established, owner-EOF, cancellation, poll-finished, poll-failed, child-joined. Effects: start-poll, cancel-poll, join, exit. EOF/cancellation are terminal: no subsequent rename or new poll is allowed. A late poll completion after stopping cannot reactivate. Single owner executes effects; pure sequence tests enumerate interruption before startup, while sleeping, and while an external command is running. Immutable owner identity cannot be relabeled onto a later wrapper.

LegacyTitleDecision consumes complete/partial process evidence and returns keep, eligible-for-confirmed-cleanup, or unknown. Eligibility requires exact title executable/argv identity, scope/tag binding, PID/start stamp, and confirmed absence of a matching wrapper. PPID 1 and command name are never sufficient. Unknown probes or ambiguous legacy scope stay untouched. Cleanup revalidates before signaling and never adopts a reused PID. Reuse #583's actual typed cleanup primitive if available after it lands; revise this table rather than duplicating it.

Memory concepts are intentionally not invented before profiling. Existing threadactivity.OSRuntime already owns bounded current-launch/activity caches. Existing contextcmd.RunWithRuntime is an injection seam. Any added occupancy cache or record bound needs its measured necessity and invalidation contract added here first.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| wrapperTitleProcess | cmd/internal/wrapcmd/title_process.go | new | exec.Command, pipe endpoints, child Wait |
| RunCLI / Run | cmd/internal/titlepoller/runcli.go; run.go | modified | validated lease input, context cancellation, clock |
| OSRuntime | cmd/internal/titlepoller/runtime.go | modified | bounded zellij/cmux commands and interruptible waits |
| SpawnTitlePoller | cmd/internal/launcher/runtime.go; osruntime.go; createflow.go; lifecycle.go | deleted | obsolete detached create/attach spawn contract |
| KillTitlePoller / KillTitlePollerContext | cmd/internal/launcher/osruntime.go; lifecycle_os.go | modified | existing shutdown paths, generation-fenced legacy migration |

Reuse procutil native process snapshots/start identity and artifactpath resolved paths. Do not concatenate artifact filenames. Test doubles are the existing stateful title fakeRuntime, extended with lease/cancellation events, plus real disposable pipe/child processes. Existing zellij fake fixtures hold pane/session/title state and can block a command until cancelled; use the same production command seam. Live conformance runs on a disposable session after wrapper/title changes and zellij upgrades, never on another slot's terminal.

## Operating envelope and trust

- Shutdown path: title must stop within seconds after wrapper SIGKILL. Proposed acceptance budget is 2 seconds on an unloaded live conformance run, with bounded eventual-exit default tests and logged timing to avoid host-load flakes. Pipe EOF must cancel 60s sleeps and in-flight external commands immediately; commands have a finite timeout even while the owner remains alive.
- Polling stays at the existing 60s cadence. One active title child, one lease reader, and a bounded number of joined workers per wrapper; no per-tick detached goroutine. No network dependency is added.
- Only the wrapper owns the pipe writer. os.Pipe descriptors are close-on-exec; only the title read descriptor is in ExtraFiles. Unrelated descendants cannot inherit the writer or keep the title alive. Audit fork windows per lessons #436/#545. Close and join before controlled re-exec; the replacement wrapper creates a fresh lease/title. On SIGKILL the kernel closes the writer.
- Title creation failure degrades optional titles with a bounded diagnostic; it does not kill the agent. No title starts without a valid owner/lease. Missing or malformed descriptor/identity/scope fails closed without touching arbitrary inherited FDs.
- Preserve storagegc title-process retention and birth gating. PID/lease files are untrusted metadata; parse bounded regular files, fence replacements, and never unlink a successor's record. Single-instance behavior must distinguish old generation from new, not trust a stale matching argv.
- Memory: measure retained post-GC heap, HeapInuse/HeapSys, TotalAlloc, profile stacks, native footprint and RSS separately. Final numerical heap/work budgets come from the reproduced cause. Large transcript/ledger inputs and identity replacement are required workloads. No arbitrary whole-file truncation may silently change token counts or binding authority.
- Local profile files are private, mode 0600, explicitly requested and bounded in number. Do not add a general HTTP debug server, read unrelated transcripts into issue prose, or manufacture credentials.
- ARCH-DRY: reuse SessionEnv, procutil, artifactpath and existing activity caches. ARCH-PURE/ORDER: pure lifetime and cleanup decisions, thin pipe/process shell. ARCH-PURPOSE: lifetime, measured growth, and live cleanup all remain acceptance obligations. ARCH-MOCK: real kernel pipe conformance plus stateful external doubles. ARCH-CONSTRAINTS: cancellation and memory-work budgets above. ARCH-SECURE: exact identity and private diagnostics; unknown is never absence.

## Task 1: obtain Go heap attribution before selecting a memory change

Files: prospective cmd/internal/titlepoller/profile_live_test.go (opt-in controlled helper only), titlepoller/runtime.go, existing workbenchperftest fixture and sessioninventory observer seams.

- [ ] Obtain TL's decision on the original-process Go profile gap. No attach, rollover, live kill, or restart is implied by this plan.
- [ ] Once approved, create a diagnostic helper that runs the actual OSRuntime poll operations on the selected authorized owner. Keep zellij/cmux effects in a recording/no-write shell for a replica. Export before/after inuse_space and alloc_space profiles from the running diagnostic process at fixed phases; record its PID/start identity, exact build and selected data generation. A replica is labeled a replica.
- [ ] Replay a private snapshot of a representative large ledger/catalog/transcript in disposable HOME/data roots. Vary history size, unchanged polls, appends, giant irrelevant records, and owner rebinding separately. Compare stack-attributed allocations and post-GC retention with the live macOS evidence.
- [ ] Append measured cause and exact memory fix to this plan and issue Log. If the candidate is disproved, stop and redesign the memory task; do not add a cache based on RSS alone.

## Task 2: red tests for actual wrapper/title lifetime

Files: new cmd/internal/wrapcmd/title_process_test.go; cmd/internal/titlepoller/lifetime_test.go; cmd/internal/titlepoller/run_test.go; launcher create/attach tests and poller_env_test.go.

- [ ] Read @superpowers-test-driven-development before writing tests.
- [ ] Add a subprocess regression through production wrap Run: title is observed ready; SIGKILL only its owning test wrapper; title exits and unrelated sentinel child stays alive. Exercise both scratch and Pair virtual wrapper transports if their startup ownership differs.
- [ ] Add deterministic held-order cases: EOF during birth wait, tick sleep, blocked zellij command, and a poll result racing EOF. Assert no further rename after stopping and all workers/child handles joined.
- [ ] Add real descriptor tests: launch unrelated child after title start, kill wrapper while that child remains, and verify it does not keep the title alive. Test graceful exit, failed agent exec, title-start failure, controlled agent restart/re-exec, duplicate attach, and replacement generation.
- [ ] Run go test ./cmd/internal/titlepoller ./cmd/internal/wrapcmd ./cmd/internal/launcher -run 'Test.*Title|Test.*Poller' -count=1. Record the intended failing behavior before implementation; compile-only failure is not the final regression proof.

## Task 3: implement owner lease and sweep old spawning/cleanup

Files: title_process.go, owner.go, lifetime.go, titlepoller run.go/runcli.go/runtime.go; launcher createflow.go/lifecycle.go/runtime.go/osruntime.go/lifecycle_os.go; cmd/internal/artifactpath/manifest.go.

- [ ] Implement the smallest pipe-owned child boundary and pure lifetime transitions. Close parent copy of the read end after spawn; release writer and boundedly join child on every controlled exit before wrapper re-exec. The title closes its inherited reader when it exits. Never retain a writer in the title or agent child.
- [ ] Start titles only for valid launched Pair agent wrappers with exact tag/scope/session identity; bare generic wrap commands must not inherit another session's title contract.
- [ ] Remove both launcher detached spawn calls and obsolete fake methods. Preserve cmux workspace/context display, birth safety, scope propagation and storage retention. Ensure attaching to the same running wrapper creates no extra title.
- [ ] Replace stale pidfile singleton decisions with generation-aware ownership and safe handoff. Existing malformed/legacy records are never blanket signaled. Harden both old title kill call sites through the agreed #583 identity seam; conditional record removal cannot erase successor evidence.
- [ ] Run the red tests green, then targeted package race tests. Mutation-check removing lease propagation/cancellation and prove the actual subprocess regression fails.
- [ ] Commit lifecycle changes and evidence with model attribution.

## Task 4: red/green memory regression after attribution

Files depend on Task 1 result; likely titlepoller/runtime.go, contextcmd/contextcmd.go and existing sessioninventory usage/activity seams. Approve the appended concrete delta before implementation if it changes the design.

- [ ] Encode the measured cause as a failing work or retention regression through title's actual production path, not merely an already-warmed unrelated cache test.
- [ ] If attribution confirms repeated inventory decode: reuse threadactivity.OSRuntime and its existing caches, sharing exact current-owner evidence with occupancy through the established runtime injection seam. If occupancy scanning dominates, add only the measured bounded mechanism; specify identity/fingerprint/invalidation and giant-record behavior first.
- [ ] Exercise unchanged large history, append, same-size replacement/restored mtime, truncate, malformed/unterminated tail and rebound/ambiguous owner. No stale usage from a previous root may render.
- [ ] Prune frame-cache entries to current live panes if retention measurement shows the unbounded key count matters; do not present that small map as the 280 MiB cause without evidence.
- [ ] Run a disposable long-running title soak with fixed-phase profiles, actual read/decode counters and post-GC heap measurements. Publish sanitized measured bounds and attribution in Log, not raw session data. Commit.

## Task 5: coordinated cleanup, documentation, and acceptance

Files: agreed legacy.go/legacy_test.go seam; atlas/architecture.md and atlas/index.md if a new page is added; README if operator diagnostics/cleanup commands change; artifactpath inventory for every new production file.

- [ ] Reconcile #583 contract with pair:4 before implementation of shared cleanup. Existing leaves require exact executable/argv, scope/tag and PID/start stamp, fresh complete wrapper-absence evidence, and effect-time identity revalidation. Probe failure/ambiguous owner/PID reuse means keep. Never kill every PPID-1 title.
- [ ] Add red/green stateful cleanup tests: live wrapper survives, unknown snapshot survives, stale pidfile/reused PID survives, changed generation between preview/effect refuses, and confirmed dead owner permits only that exact title. Any TERM escalation revalidates identity again.
- [ ] Present the exact one-time live candidate set and confirmation contract to the TL. Execute only that authorized set, preserve unknowns, then remeasure every retained live owner and cleared candidate. Restart/rollout is coordinated by TL; no fleet restart is implied here.
- [ ] Update atlas to describe wrapper ownership, EOF cancellation, profile interpretation, measured memory bound and cleanup uncertainty; sweep obsolete launcher/session-only lifetime prose and spawn inventory.
- [ ] Run targeted race tests for titlepoller/wrapcmd/launcher plus contextcmd/sessioninventory/threadactivity and artifactpath tests as touched. Push head, run declared sdlc local-check suite, and normal close with fresh review. This is one issue-close boundary, not artificial per-task milestones.
- [ ] Before claiming Done, demonstrate disposable live wrapper death, bounded soak, and final live inventory: one title per rolled-out live wrapper, no title for proven dead threads. Report any legacy active wrappers awaiting TL rollout explicitly. Close/land through normal SDLC gates only after required acceptance is satisfied.
