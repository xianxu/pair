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

## Task 2: red lifetime regressions before implementation

Files: cmd/internal/wrapcmd/title_process_test.go, cmd/internal/titlepoller/{owner,lifetime,run}_test.go, launcher poller_env_test.go.

- [ ] Read @superpowers-test-driven-development, add the production wrapper SIGKILL/inheritance/replacement acceptance regression, run it and record the intended behavioral failure.
- [ ] Test the risky functions using the strategy table below. Run go test ./cmd/internal/titlepoller ./cmd/internal/wrapcmd ./cmd/internal/launcher -run 'Test.*Title|Test.*Poller' -count=1 and retain RED evidence.

| Production function | Adversarial class and mechanical guard |
|---|---|
| ParseTitleOwner | malformed or borrowed owner/FD metadata: fuzz strict typed parsing and refusal before descriptor/child effects |
| AdvanceTitleLifetime | arbitrary EOF/cancel/completion order: pure sequence tests prove terminal stop and no later poll effect |
| startWrapperTitleProcess | wrapper death and descriptor inheritance across replacement: real production wrap subprocess SIGKILL, surviving sentinel, exact-child exit/reap oracle |
| RunContext | cancellation at any poll phase: held stateful runtime makes scan, birth, sleep and command interruption reproducible and asserts no late rename |

## Task 3: implement owner lease and sweep old spawning/cleanup

Files: title_process.go, owner.go, lifetime.go, titlepoller run.go/runcli.go/runtime.go; launcher createflow.go/lifecycle.go/runtime.go/osruntime.go/lifecycle_os.go; cmd/internal/artifactpath/manifest.go.

- [ ] Implement the smallest pipe-owned child boundary and pure lifetime transitions. Close parent copy of the read end after spawn; release writer and boundedly join child on every controlled exit before wrapper re-exec. The title closes its inherited reader when it exits. Never retain a writer in the title or agent child.
- [ ] Start titles only for valid launched Pair agent wrappers with exact tag/scope/session identity; bare generic wrap commands must not inherit another session's title contract.
- [ ] Remove both launcher detached spawn calls and obsolete fake methods. Preserve cmux workspace/context display, birth safety, scope propagation and storage retention. Ensure attaching to the same running wrapper creates no extra title.
- [ ] Replace stale pidfile singleton decisions with generation-aware ownership and safe handoff. Existing malformed/legacy records are never blanket signaled. Harden both old title kill call sites through the agreed #583 identity seam; conditional record removal cannot erase successor evidence.
- [ ] Run the red tests green, then targeted package race tests. Mutation-check removing lease propagation/cancellation and prove the actual subprocess regression fails.
- [ ] Commit lifecycle changes and evidence with model attribution.

## Task 4: measured memory regression and fix

Files depend on Task 1 attribution; likely titlepoller/runtime.go, contextcmd/contextcmd.go and existing sessioninventory usage/activity seams. Append the measured delta before implementing this part.

- [ ] Encode the attributed work/retention defect through the actual title production path as a failing regression.
- [ ] Apply only the profile-supported change, favoring existing threadactivity.OSRuntime caches and contextcmd.RunWithRuntime over duplicate policy. Define any new cache/record budget and invalidation function before implementing it.
- [ ] Run the strategy below, then the disposable helper soak with fixed-phase Go profiles, IO/decode counters and post-GC heap measurements. Log sanitized attribution/bounds and commit.

| Production function | Adversarial class and mechanical guard |
|---|---|
| OSRuntime.SessionActivity / ContextCount | history growth and file/owner mutation: stateful real-file fixture plus QueryWorkObserver proves warm work bounds and current-root parity |
| TokenUsageForRoot (if changed) | arbitrary JSONL framing/mutation: fuzz against current full-scan oracle and cancellation-aware range reader |
| frameCache.changed (if changed) | pane churn: property test retained keys bounded by currently live panes |

## Task 5: coordinated cleanup and acceptance

Files: titlepoller legacy.go/legacy_test.go or the landed #583 equivalent, atlas/architecture.md, README for any changed operator commands, artifactpath inventory.

- [ ] Reconcile #583 with pair:4; never stack on its unlanded branch. Implement shared cleanup only after the exact identity/confirmation seam is known.
- [ ] Red then green: DecideLegacyTitle consumes incomplete, replaced or reused process evidence in a stateful process-table fake; invariant is that only exact unchanged title generations with complete matching-wrapper absence proof may reach a confirmed signal. Unknown keeps the process. Effect-time revalidation and conditional metadata removal share the same authority.
- [ ] Present the exact one-time live candidate set and confirmation contract to TL. Execute only that authorized set and remeasure retained owners; TL coordinates any legacy active-wrapper rollout.
- [ ] Update atlas ownership/profile/uncertainty contracts, operator README where relevant and every new source inventory entry.
- [ ] Run targeted affected package race tests, push head and run the declared full suite; normal close/fresh review/landing. No artificial per-task milestones.
- [ ] Acceptance still requires production wrapper-death exit, bounded measured soak and final live inventory with one title per rolled-out live wrapper and none for proven dead threads. Explicitly report any outstanding live rollout rather than claiming Done.

## Revisions

### 2026-10-16 — TL approval and profiling choice

Verified submitted receipt 72aed150-a96e-46b8-8e68-8bb2dcbaa16f from ops:0 to this wrapper incarnation. TL approved the pipe lease, red SIGKILL/inheritance/replacement tests first, and proceeding to change-code. TL chose a controlled replica running the production title path against the real ledger/catalog with a Go heap profile. No diagnostic rollover of live titles. This resolves the approval/profile-method gates above; exact cause still requires profile attribution. Native current-versus-peak footprint is consistent with a prior spike, but does not itself attribute Go allocations. Cleanup coordination with #583 remains required.

### 2026-10-16 — plan-quality findings PQ-1/PQ-2

PQ-2: compressed the previous detailed case lists in Tasks 2/4/5 into function-level adversarial strategies, preserving the required production SIGKILL/inheritance/replacement and mutation acceptance obligations. PQ-1: proposed the shutdown contract below to TL (receipt d4707560-510f-42d6-a747-eb62e6e2dcd0); await approval of the bounded exit fallback before implementation.

**Revised shutdown contract:** Introduce RunContext for the loop. A title-lifetime context reaches QuerySessionContext and a context-aware inventory Runtime adapter that checks cancellation before/after each IO range; usage scanning uses this adapter as well. Metadata/stat and parsing boundaries check cancellation before beginning the next operation. OSRuntime Sleep and zellij/cmux calls consume that same context, with finite command timeouts and WaitDelay. No replacement poll begins after cancellation. EOF is watched independently of synchronous scans. The title CLI permits a short cooperative grace (250ms); if a non-interruptible decode remains, it finishes process teardown after cancelling and joining its owned external commands rather than waiting indefinitely for in-process decoding. The process boundary terminates remaining Go work and the wrapper reaps its exact child. No worker escapes the title process; no agent/server is killed. The ordinary joined path remains the default; the fallback is explicitly a CLI-process exit contract, not safe reuse of RunCLI as an in-process server routine. The acceptance helper executes the actual title command in a disposable child, holds production scan IO to prove the fallback, and observes both title exit and absence of owned command leftovers. Diagnostic replica invokes runtime methods directly and does not take this CLI-only exit path.

Task 1 profiling isolation: copy the real selected owner's ledger and catalog into a disposable Pair scope, with real native storage read-only. QuerySession can advance its catalog, so pointing PAIR_DATA_DIR at live storage is prohibited for the replica. Capture source metadata and digests, pass the unchanged logical scope key, and let only copied catalog state mutate.

### 2026-10-16 — measured cause and first memory delta

Controlled production-path replica profiles at /tmp/pair-617-profile.24gkdy: ten poll pairs allocate6,450,991,016 bytes total versus10,675,720 baseline (~644MB/poll), natural HeapAlloc1,576,424 bytes, postGC296,600 bytes versus201,072 baseline. Sampled parseOwnerLedger65.47%, TokenUsageForRoot34.18%; no retained transcript/ledger after GC. Original ledger/catalog hashes unchanged. This attributes repeated read/decode churn; it does not identify the original process's RSS composition.

Implement titleProjection in cmd/internal/titlepoller/projection.go as an IO integration owner, with no new cache policy: one existing OwnerLaunchCache shared by inventory queries for both activity and context, and the existing ActivityCache.Query. contextcmd.RunWithRuntime preserves occupancy semantics. Add projection_test.go against the existing workbenchperftest real-file fixture and measured read/decode counters; prove unchanged repeated owner history is not decoded. Usage remains the existing full scanner for this first measured delta; rerun the profile before choosing a usage-cache or streaming change. Further mechanism changes still require TL review. Constructor consumes one fixed exact Env per title; no cross-scope/global cache. This is the already-proposed reuse branch of Task4, now backed by profile attribution.

### 2026-10-16 — changing-input spike: stop memory changes and re-plan

The actual wired production-path replica at /tmp/pair-617-current.3de0yz changes the diagnosis beyond the unchanged-input run. Ten polls while the native transcript was actively appending allocated9,041,691,176 bytes; individual polls reached~2.78GB of allocation, natural/postGC live heap remained~2.1MB/~319KB, HeapSys259,555,328 and HeapReleased254,713,856. Sampled8,552.73MiB allocation attributes73.54% to ValidateBindingProof's full revalidation fallback at sessioninventory/query.go:216. ValidateTargetWork's ObserveStableArtifact accounts for3.14GiB, ValidateClaudeDelta1.41GiB and NativeEventsFromRecords1.58GiB cumulative. Changed fingerprints without a generation token are deliberately rejected by AdvanceTargetValidation, then the authorized same-artifact fallback reads all records and constructs events. Activity's bounded drift retry can repeat that work, and count validates again. An activity=false result does not prove loss of binding.

The proposed unchanged usage scalar cache (hub receipt570c5004-fdee-4c9d-b3fb-cfbdd4a37b33) is not sufficient for the measured changing-input spike. Do not implement it as the memory fix. Pause memory implementation here and seek TL review of a revised plan. Simplest alternatives: (a) a title-only memory budget or subprocess merely shifts/reclaims the spike and leaves repeated authorization work; (b) an unchanged-result cache misses active mutations, as measured; (c) recommended investigation/design: a query-only streaming full-revalidation projection that preserves current authorization rules and folds required fact/state without retaining every transcript record or constructing unused event history. That is a shared validation seam change, requiring a concrete reviewed plan before code, including provider parity, adversarial mutation tests, cancellation and a memory bound relative to maximum record/current authority rather than transcript history. No generation-token assumption or reader-only truncation is permitted. Scope remains all original Done when obligations, not a deferred memory fix.

### 2026-10-16 — TL-directed structural test isolation revision

Verified dispatch e6510de0-274d-4237-bf31-8f086d531634 authorizes a shared empty-environment helper and a regression guard covering every test spawning Pair/wrapper, replacing the per-fixture correction. ARCH-DRY/ARCH-SECURE: tests/lib/pair-test-env.sh is the single environment owner across languages; testpair.Environment delegates to its NUL output. Start with only PATH and disposable HOME; every additional fixture value is explicit. Migrate direct binary calls, generated shell stanzas, and wrapper self-exec helpers; guard the original offending forms plus aliases/indirection before accepting the sweep. Behavioral RED reproduces ambient inheritance; GREEN must prove a foreign events sentinel survives while the disposable target is written. Read configured backup catalog only, record existence/absence, never restore. This directive does not approve pending memory or CLI fallback changes.

### 2026-10-16 — isolation review refinement

Fresh-eyes review rejected guard shortcuts that classified environments after a spawn, accepted conditional aliases or unsafe dictionary updates, trusted a public reentry marker, or exempted Python files merely mentioning the helper. Keep the same approved empty-environment boundary; make guard provenance local to each spawn and conservative across branches/replacement/alias mutation. Python uses a pure AST analysis rather than growing text exceptions. The shell bootstrap always invokes the canonical helper and executes the trusted local body after the fixed two-line prelude; original argv stays literal and entrypoint paths use $0. Read-only original-source audit covers direct Go, generated shell, wrapper selfexec, native candidate hooks and nested scripts. Timed watcher fixtures consume the existing shared binary instead of go run under a cold disposable HOME. No change to pending memory/fallback decisions or acceptance.

### 2026-10-16 — third-delta re-plan: one admission per wrapper invocation (awaiting TL)

Verified TL receipt36649870-8348-4568-b299-c02445643d9d explicitly stops the memory design for re-plan: measure polling/proof before implementation; propose one unchanged full proof followed by title-only append tails. Immediate os.Exit on lease EOF with owned command groups is the teardown decision, superseding every earlier cooperative-join/grace/fallback proposal. The shared environment allowlist+guard decision is already implemented at7a39140d8. No new production code is authorized by sending this proposal.

Measurement used a disposable Go test overlay outside the checkout. The production default loop executed three60s sleeps and three activity/usage query pairs with a controlled clock: configured1 poll/min, actual period60s plus work; not a live fleet-frequency observation. One uncached existing ValidateBindingProof call on the real ops authorized transcript used copied Pair storage, read-only native reads and an empty catalog to exercise full proof. Wall1.907860916s; cumulative allocation1,075,449,240B=1,025.628MiB (~1.002GiB);1,581,004 allocations;87,035,043B=83.003MiB read in1,329calls;47,387 records and15,225events;nil error. Cold owner lookup separately185.029041ms/212,920,080B (~203.056MiB). Immediate HeapSys217,677,824B, HeapAlloc119,538,072B; after releasing measurement results and GC, live heap323,704B (~316KiB). This GC result is a measurement-process observation, not yet a title steady-state bound. Go1.27.1, GOMAXPROCS3, GOFLAGS -p=3; busy shared host load32.74/42.07/51.09 after run, not idle timing. Private evidence/overlay: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-617-proof-cost.wY3HyU; stats SHA25664f5a051b73ea03a96063cb578399e9c647083886d9c6dfeb1e12ee9dba5f5f9. No live title was driven, restarted or killed; no native/catalog writes.

ARCH-ORDER/ARCH-SECURE audit: title starts before watcher confirmation, so AwaitingBinding must tolerate an idle first prompt for the lease lifetime; bind to exact launch ordinal, not just PID (fresh invocation syscall.Exec keeps PID but closes old title lease). ConfirmIfCurrent fences conflicting roots, while legacy AppendBinding[Proof]IfCurrent still allows a second root and newer launches can be published concurrently. Keep a cheap owner-ledger metadata/change fence and exit on conflict/supersession instead of assuming universal ledger immutability. Inode+offset cannot detect an overwrite or truncate-and-regrow wholly between polls: the append-only native-writer operating assumption is explicit in the approval request.

Proposed pure title state: AwaitingBinding → Admitted(compact scanner identity/schema, pinned artifact identity, raw/parser offsets, partial record, last accepted usage/activity) → Exited. Admission runs the existing full validator once, folds initial facts from its already-read records, discards record/event arrays; no full proof calls in steady state. Thin IO polls pinned artifact identity/mtime and64KiB appended chunks. Reuse provider tail/usage rules; no invented occupancy for unsupported providers. Proposed partial-record ceiling32MiB, excess exits; inode change, observed shrink, malformed complete records or identity contradiction exits, with no silent full-query fallback or root discovery. Owner ledger changes are checked with existing compact/incremental owner cache. EOF ends immediately and commands have owned process groups; no decoder-join fallback timer. Red tests cover delayed initial confirmation, zero warm full proofs, unchanged/no tail reads, append/split records, provider parity, conflict/replacement/shrink, growing-history retained-state bound and EOF during proof/command work. This supersedes scalar usage caching and streaming full-proof reconstruction.

The one-paragraph measured plan was sent to hub as ffff4ada-6afe-41fb-b745-4eb23b6bf8eb, queued position7 at admission. Await TL approval of this revised contract and budgets before implementation or a new change-code gate.

### 2026-10-16 — consolidated TL decisions and cleanup fallback limits

Verified TL556ee0bd-57cc-4a3c-a0cc-62ef9bad3d61. Refine the pending revision: remove #617-introduced context/cancellation plumbing rather than extending it; EOF requires immediate os.Exit, owned external command groups killed with title, and every durable publication uses temp+rename. The current cooperative-join implementation does not satisfy this consolidated contract and remains incomplete. Reuse osfs.WriteAtomic for title PID/workspace-owner publications; keep unrelated pre-existing context APIs intact. Full-proof-once/tail proposal is already submitted (ffff4ada-6afe-41fb-b745-4eb23b6bf8eb); no duplicate and no implementation before approval.

Cleanup instruction: coordinate directly with pair:4; otherwise use #587 ClassifyUnattributed unchanged and note its limits. Concrete request5ebc255a-774d-4f4c-9775-bc628b5c0610 is delivering, waiting for previous automatic input to clear; pair:4 has not answered. Inspected current ClassifyUnattributed plus AgentInventory: candidate alert requires PPID1, non-zombie, known start identity, unclaimed identity, known RSS above floor, and name zellij/supported native agent or nvim with exact --embed. The function explicitly grants no ownership; it neither includes pair/title nor proves wrapper absence. Actual title25522 has comm /Users/xianxu/workspace/pair/bin/pair and PPID1, so it is excluded by unchanged fallback. Do not widen this classifier or turn unproven alert into kill authority. Legacy cleanup remains pending a compatible owner/wrapper-absence predicate; no title has been killed. Exact title PID+start and wrapper absence still require effect-time fencing.

### 2026-10-16 — approved one-proof plan, tail fence and title-specific cleanup

Verified submitted TL receipt 253d1a36-2b1d-4ebd-b1ca-232f683bd8ca approves ffff4ada, including append-only native transcripts. This revision is the active implementation contract and supersedes pending-approval statements and the earlier ClassifyUnattributed/pair:4 fallback. Re-run the full change-code gate before implementation. No live cleanup effects are authorized before the requested report-only comparison.

ARCH-SECURE/ARCH-ORDER: store a SHA-256 hash of the last min(4096, raw offset) bytes ending at the raw observed offset (including an incomplete record). Before every tail read, re-read that window from the pinned artifact, check inode and size, and exit on mismatch, shrink, replacement or failed complete-record parse. Empty tails retain an empty-window hash. Keep parser offset separate from raw offset. The hash detects observed boundary mutation; the approved append-only assumption remains necessary for older interior bytes. Never silently fall back to a full scan. Admission uses the unchanged full validator once; initialize the compact projection from its existing records and discard history arrays. Poll activity and supported usage only, in 64KiB chunks, with a 32MiB maximum partial record and unchanged provider identity/usage semantics. Owner metadata changes trigger the existing compact owner fence; conflict or superseded launch exits.

| Entity / owner | State and effects | Budget / adversarial verification |
|---|---|---|
| Title-only projection (sessioninventory plus titlepoller adapter) | Await binding for the exact invocation, then admit once; maintain scanner state, artifact identity, raw/parser offsets, partial bytes, usage/activity and trailing hash; terminal on authority or tail contradiction | History-independent retained state, 64KiB reads, 32MiB partial record; RED delayed binding, zero warm full proofs, split append, provider parity, hash mutation, replacement/shrink/conflict and growing-history bounds |
| CLI lease watcher / OS command boundary | Lease EOF exits immediately with os.Exit; owned external commands die with title; remove only #617 context plumbing; preserve unrelated existing context APIs | No cancellation join, grace period or bounded CLI fallback; RED EOF during proof and blocked external command, inheritance and replacement; durable PID/workspace-owner writes use existing temp+rename |
| Legacy-title classifier (pure process sample) | Exact pair title TAG argv, PPID 1, start before lease-build activation, and no live wrapper carrying the exact TAG in the same complete sample; unknown evidence keeps process | Pure tests for prefix collisions, incomplete sample, live wrapper, cutoff, PID reuse and malformed identity; never use ClassifyUnattributed |
| Legacy cleanup effects (thin IO) | Report candidates first with PID, start stamp, TAG, age and RSS; after TL comparison, SIGTERM then short wait and SIGKILL only with identity rechecked before each | One process sample proves wrapper absence; PID plus start stamp fences each effect; test replacement between stages and surviving live wrappers. Capture and disclose the legacy cutoff before rollout rather than guessing a process's generation |

Implementation order: red projection/tail tests, red immediate-exit/command tests, red cleanup classifier/effect tests; minimal green implementation; affected race tests and controlled production-path growth profile; report-only live inventory to TL; atlas and artifact inventory updates; push, integrated suite and normal close/review. Existing completed isolation tests remain required and every new subprocess fixture uses the empty-environment helper. Legacy rollout and cleanup remain explicit acceptance work, with no unrelated live title rollover.

### 2026-10-16 — implementation checkpoint and remaining gate

One-proof/tail memory work, hash fencing, compact state, final owner publication fence, atomic writes, bounded frame cache and title-specific report/approved-set cleanup helpers are implemented and verified as logged in the issue. Fresh review findings (post-tail authority and positive profile controls) were fixed with adversarial tests. The accelerated real-file growth soak is complete; no live rollout is claimed. Cleanup dry run is submitted93d4945d with5eligible generations and24live-wrapper keeps; no signals.

Remaining active sequence: TL resolves immediate-exit effect question361802ad (own-group kill versus literal os.Exit); implement EOF without decoder join/context plumbing and green the two committed RED subprocess regressions; run integrated checks and normal close/review; obtain TL comparison before approved-set live cleanup and coordinated rollout. Current CLI still cooperatively joins and does not satisfy the final teardown contract. Checkpoint is incomplete, not codecomplete or landing-ready. Startup-proof memory is paid once; final warm polls30–32KB, growing10.6MB/100poll soak retains~0.4MB after GC. Evidence paths, bounds, caveats and exact receipts remain in issue Log.

### 2026-10-16 — immediate owned-group EOF teardown approved

Verified submitted TL receipt2ed371a2-2749-401b-a235-03c3f7121ed4, exact ops:0 to current pair:6. TL accepts SIGKILL of the title's own Setsid process group on lease EOF as the immediate teardown effect, superseding literal os.Exit. Commands inherit that group; one kernel effect kills title and commands even when Start races EOF. No Start/register lock, context plumbing, decoder join, grace period or bounded CLI fallback. Validate title is its own process-group leader before arming this effect; never kill a borrowed group. All title-reachable durable publication remains temp+rename. Existing blocked-proof and stubborn-command descendant RED tests precede implementation. This resolves question361802ad; legacy cleanup comparison remains separate and no live signal is authorized by this teardown decision.

### 2026-10-16 — legacy cleanup acceptance complete

TL approval855a869b compared the dry-run set and corrected the historical PPID1assumption. Fresh complete sampling and per-effect PID/start fences terminated only the5approved no-wrapper generations by SIGTERM; no escalation needed. All24live-wrapper controls survived and final report has0eligible/0unknown. Evidence and receipt3ed3cc1b are in the issue. Do not signal any additional generations under this approval. Remaining acceptance is the staged new lease build/live wrapper lifecycle, integrated suite and ordinary SDLC close/landing.

### 2026-10-16 — combined post-landing restart; no staging

Received TL977e9db5-f9fe-4e8d-9e1e-2992464e4c01 in this actual turn; canonical stored sender, current recipient incarnation and exact body match. Receipt status remains indeterminate (submit observed but turn/paste confirmation timed out), so do not misstate it as submitted; transport discrepancy reported as0375f79e-6188-4aea-9c73-69d7ab77298e. The received direction is NO staging on pair:0: land #617 only after its recorded suite and normal close-review SHIP, TL rebuilds main with #617 and #618, operator restarts Couch once, then this worker runs live wrapper/title acceptance on fresh slots. This supersedes the pre-landing staging proposal and explicitly moves that live acceptance after landing; no live acceptance result is claimed now. The5-title cleanup had already completed under the earlier verified approval855a869b and was not repeated. Additional89183/grok remains outside that approved set and under separate request775cdf34.

The machine test queue is legitimate: the current holder is an Ariadne worktree, whose heavy suite uses both slots; the next named waiter is Pair. Our recorded exact-head suite remains in its original queue entry. Do not bypass the machine gate or fabricate a suite result.
