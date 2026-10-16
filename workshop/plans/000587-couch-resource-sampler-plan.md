# Couch resource sampler implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Use superpowers-executing-plans for the shared ownership/lifecycle integration; bounded native-reader and pure-model tasks may use subagents. Checkboxes are tasks, not separate milestone review boundaries.

**Goal:** Detect costly slot processes and proven orphan descendants within minutes, and expose the same observations through `couch --resources [--json]` without doing process IO on the request path.

**Architecture:** One supervisor-owned worker takes bounded native observations every minute. A pure reducer attributes process generations, computes rates, maintains alert episodes and bounded history; the shell best-effort persists aggregate history and submits advisory alerts through Couch's existing broker. Resource evidence never authorizes killing or reassigning a process.

**Tech stack:** Go; Darwin libproc/sysctl via a small SDK-backed cgo adapter; existing procutil identities, Couch socket protocol, broker delivery, and durablefile atomic publication. Other platforms/non-cgo builds expose an explicit unsupported capability rather than fabricated zero measurements.

## Design choices for TL approval

1. Use authenticated Zellij server roots, wrapper bindings, and Couch-owned launch/attach roots. Wrapper descendants alone miss draft nvim and terminal children. Capture existing `PairSessionBinding.Owner.Server` evidence during admission/lifecycle observation, rather than running Zellij or argv scans every minute. Validate the generation against each process table. Include starting/draft-only/parked slot rows even when measurements are unavailable; a missing broker connection is not proof of an orphan.
2. Track only proven descendants as orphans. Remember `(PID, start identity, original thread/slot)` when first observed; keep attribution through reparenting during this supervisor run while that exact process survives. Name-based host-wide orphan guesses would misattribute unrelated work. A child born and orphaned entirely between samples cannot be discovered from ancestry alone; state this limitation in README and JSON coverage. Previously unobserved pre-upgrade orphans likewise remain outside proven attribution.
3. Initial defaults: 60s cadence; slot writes >2 MiB/s continuously for 10m; any process RSS >8 GiB for one valid sample; slot CPU >90% of one core for 15m while the wrapper reports settled and no delivery is pending; proven orphan RSS >256 MiB for two valid samples. CPU may exceed 100% for multithreaded work. These are proposed defaults, not measurements. Parse documented `COUCH_RESOURCE_*` overrides once at startup with bounded durations/values; unknown idle status cannot trigger idle-CPU alerts.
4. Keep history as one namespace-owned bounded snapshot (24h, at most 1,440 aggregate frames, at most 4 MiB encoded), with all ownership/episode state held only in memory. Reuse atomic file publication; prune oldest frames before serialization and expose the actual oldest sample when the byte cap shortens coverage. This favors a simple best-effort history store over a bespoke file ring. Count its write cost explicitly: at most 4 MiB per minute, no history IO on CLI queries. If measured encoding/write CPU misses the budget, stop and revise storage before implementing a more complex ring.
5. Darwin adapter calls the supported `proc_pid_rusage` API using SDK types, avoiding a hand-copied ABI/raw syscall or new FFI dependency. Confirm normal builds have cgo available; document unsupported non-cgo builds explicitly. Do not silently weaken accounting there.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| ResourceProcessKey / ResourceRoot / ResourceSlot / ResourceSlotInventory / ResourceProvenance | `cmd/internal/couchcore/resource.go` | new |
| ResourceUsageObservation / ResourceSample / ResourceView / ResourceCost / ResourceProcessView / ResourceSlotView / ResourceSelection | `cmd/internal/couchcore/resource.go` | new |
| ResourcePolicy | `cmd/internal/couchcore/resource_policy.go` | new |
| ResourceState / ResourcePrevious / ResourceSlotTotal / AdvanceResources | `cmd/internal/couchcore/resource_reduce.go` | new |
| ResourceEpisode / ResourceEffect / ResourceDeliveryOutcome / ResourceEpisodeEvent | `cmd/internal/couchcore/resource_episode.go` | new |
| ResourceHistory / ResourceFrame | `cmd/internal/couchcore/resource_history.go` | new |

- A process key is PID plus kernel start identity, never PID alone. A root binds that key to an exact thread/slot incarnation and the evidence source. A process belongs to at most one slot; overlapping roots within one slot deduplicate, conflicting ownership is unknown rather than double-counted. The pure traversal reuses/extracts the existing ancestry logic where appropriate, without importing renice policy.
- Observations carry counter capability and failure reasons. CPU is delta(user+system nanoseconds)/actual monotonic interval; RSS is a current gauge; writes are delta of `ri_diskio_byteswritten`, not logical write calls. Per-process lifetime counters and accumulated **observed since sampler start** slot totals have distinct labels. First observation establishes a baseline. Child-exit counters are not added again (no double counting).
- ResourceState owns prior counters, provenance, episodes and observed totals; ResourceHistory owns retained aggregate frames. No caller mutates its maps. Unknown/partial data, process replacement, counter decreases, clock regressions and sample gaps cannot fabricate a continuous high-rate interval. Exited-between-samples work is an explicit sampling limitation, not exact whole-lifetime billing.
- ResourceHistory stores aggregate frames and relevant offender summaries, not every process's argv/counter history. One core model feeds JSON, text, alert text and storage validation (ARCH-DRY).

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| NativeResourceReader | `cmd/internal/procutil/resource.go` (declaration), `resource_darwin.go` / `resource_unsupported.go` (platform backend) | new | libproc and bounded process table/identity sysctls |
| resourceFakeWorld | `cmd/internal/couchcore/resource_fake_test.go` | new | stateful fake process generations, counters, errors and clock |
| ResourceRootCache / ResourceRootEvidence | `cmd/internal/couchcore/resource_roots.go` | new integration cache | revision-fenced root proof from modified artifactcollision.go ownership checker |
| resourceInventory / Console.ResourceInventory | `cmd/internal/couchcmd/resources_inventory.go`, `cmd/internal/couchtty/console_resources.go` | new adapter methods | cached console rows, hosted roots, workspace proof and inventory completeness |
| resourceSampler / resourceReader | `cmd/internal/couchcmd/resources.go` | new | reader, cached slot/idle snapshots, timer, reducer, store, broker |
| ResourceHistoryStore | `cmd/internal/couchcore/resource_store.go` | new | bounded validated namespace files + durablefile |
| resourceHistoryPersistence | `cmd/internal/couchcmd/resources_persist.go` | new adapter | best-effort bounded history, independent of alert delivery |
| resourceAlertDispatcher | `cmd/internal/couchcmd/resources_alert.go` | new adapter | owning-hub broker send and pure in-memory delivery outcomes |
| projectResourceView | `cmd/internal/couchcmd/resources_projection.go` | new pure projection | existing broker frame byte envelope |
| resources request/response | `cmd/internal/couchmessage/protocol.go`, `cmd/internal/couchcmd/cli.go`, `messages.go`, `message_service.go`, `resources_render.go` | modified/new | existing authenticated local socket |

resourceFakeWorld models process generations, counters, partial tables and restart; selection tests model ancestry/reparenting/reuse, native fakes model permission failures, and resourceBrokerEndpoint exercises queued/delivered alerts through the real broker. Native conformance uses only owned scratch children. Storage tests use portable temporary directories; integration never reaches real Couch state.

## Ownership and lifecycle contract

- Capture the authenticated `launcher.SessionOwnerObservation.Server` already returned by the session ownership probe; retain it only after the relevant admission/lifecycle succeeds. A failed or superseded admission cannot install a root for the current binding. Replacement roots need fresh authenticated evidence.
- Publish a small immutable root/slot inventory from the existing console and lifecycle observations, including non-connected slots, without polling all thread files on each sample. Reuse `launcher.AgentBelongsToServer` to verify wrapper/server relationships. The console's attached process is a distinct client/helper root, not a substitute for the detached server.
- For slots discovered before root proof is available, show `unavailable: ownership unconfirmed`; never omit the row. Boot/start/resume/reattach/switch/park/removal update the same root inventory. Restart has no remembered ownership; fresh authenticated roots establish attribution.
- A remembered process that no longer descends from an active root becomes an orphan only when ownership loss is confirmed. Unknown slot/root state retains `ownership unknown`. Parked-but-owned terminal processes remain owned. Keep original thread identity even after the numeric slot hosts another thread.
- Sampler worker uses `messageService.lifetime` and `workers`; one synchronous pass at a time, ticker skips missed periods, cancellation joins. Read and disk IO never run holding console/broker locks. CLI returns the cached view with sample time, age, coverage and errors.

## Episodes and delivery (ARCH-ORDER)

| State/event | Next state and effects |
|---|---|
| clear + valid high | accumulating, remember interval start (immediate thresholds become pending) |
| accumulating + continuous high at duration | pending, emit one deterministic episode-ID delivery effect |
| accumulating + normal/gap/unknown | clear; no duration carried across missing evidence |
| pending + accepted broker receipt | notified; suppress repeats in memory |
| pending + definite refusal | remain pending; bounded retry on subsequent sample while still high |
| pending + uncertain send outcome | inspect/retry the same broker message ID; never mint a second alert |
| pending/notified + valid recovery | clear/re-arm for the next episode |
| notified + high or unknown | retain suppression; unknown is not recovery |
| any + replaced process/retired owner | retire that identity's episode; preserve historical frame |

Resolve the current owning hub immediately before send, as waitWatcher does; do not hardcode the main hub. Alert names slot, process PID/name, metric, value, duration, sample coverage, and `couch --resources` as the inspection action. No automatic kill/restart. Hub unavailable retains a bounded pending episode. Keep episode identity in memory and retry the same ID within the run; restart re-arms from fresh observations. Admission success, rather than text generation, establishes notification progress. Do not mutate or bypass broker busy-delivery policy.

## Cost and retention envelope

- Requirement: well below 1% of one core. Qualification target: mean sampler CPU <0.2% of one core at default cadence, with no sample CPU >0.5% of a 60s budget (300ms) in the bounded representative run. Report measured CPU time, elapsed time, process count, slot count and storage bytes; do not infer CPU cost from wall time alone. Measurements include traversal, reducer, encoding and persistence.
- Count at native calls, not only interface calls: table enumeration, rusage, identity validation and self-cost measurements. Existing `SysctlKinfoProcSlice` retries ENOMEM without a bound, so do not call it and claim “one syscall.” Add a bounded native table path with shared Process conversion and at most three size/read attempts. Use fixed MIBs to avoid hidden per-call name lookup. Count failed calls as well as successful ones.
- Expected measurement-call bound: at most `6 + 2P + 2` native calls per pass (three two-call table attempts, one rusage and one identity fence per selected process, two sampler CPU measurements), subject to verifying the native adapter implementation. File IO operations and bytes are counted separately; identify any added call in the model rather than hiding it in a helper. PID reuse between table/rusage/fence invalidates that observation.
- Default operating envelope: up to 256 displayed slots and 8,192 tracked live/provenance processes; process-table buffer capped at 65,536 rows. Hard limits produce a visible partial-coverage diagnostic, no silent all-clear or unbounded retries. Select active roots first, then known orphans, with deterministic ordering; report omitted counts. No subprocess launches on steady-state samples. No native process calls, disk writes or history decode on resources requests.
- History snapshot: <=1,440 frames, <=24h by sample timestamp, <=4 MiB encoded. Only history is written. Atomic publication may transiently retain old+new files; enforce <=8 MiB plus fixed metadata during replacement. Reject oversized/corrupt/foreign-schema history, report lost continuity and begin a fresh bounded ring.
- Register all source consumers in `artifactpath/manifest.go` as exact namespace-owned non-Pair-artifact sources, matching existing Couch namespace diagnostics. Explicit rolling retention is owned by ResourceHistoryStore (not diagnosticlog's 365-day policy); document it in `atlas/storage-retention.md`. Age/count/byte pruning occurs on load and writes; stale files left while Couch is stopped remain byte-bounded and are pruned on next open. No global unrelated-file sweep.
- Native unsupported/permission errors disable affected measurements with reasons; the supervisor continues. Failed history persistence marks the view degraded but never blocks advisory alerts. No credentials or full command lines stored. All labels are escaped for terminal output; snapshots are 0600 under the namespace (ARCH-SECURE).

## Implementation tasks (one atomic delivery, full flow)

### 1. Native counters and counted IO

Files: new `procutil/resource*.go` and tests; shared conversion in `table_darwin.go` if needed.

- [x] Write the `NativeResourceReader` function-contract tests below; run to establish red.
- [x] Implement SDK-backed Darwin calls with `proc_pid_rusage` and identity fence; keep a compiling explicit unsupported adapter for other builds.
- [x] Add owned-child conformance: CPU/memory and a bounded synced-file write produce monotonic real counters; reuse the identical reader in production and fake tests. No root required.
- [x] Run targeted tests and record the actual call formula. Commit the passing unit.

### 2. Pure attribution, rates and episodes

Files: new `couchcore/resource*.go` model/reducer tests and fake.

- [x] Establish red for `SelectResourceProcesses` using its function contract below.
- [x] Establish red for `AdvanceResources` and `AdvanceResourceEpisode` using their function contracts below.
- [x] Implement immutable-input reducer and explicit episode states; test deterministic sequences rather than sleeping. Assert each process contributes at most once and unknown evidence cannot become recovery or zero.
- [x] Run targeted tests and commit.

### 3. Root publication and supervisor worker

Files: `message_service.go`, `couchcore/artifactcollision.go`, new `couchtty/console_resources.go`, new `couchcmd/resources.go` and colocated tests.

- [x] Prove current authenticated server evidence is captured once across admission/start/resume and discarded when an older completion loses its generation race.
- [x] Wire draft-only, starting, parked and connected slot rows through one inventory; test the full server tree includes nvim and agent children without process-name inference.
- [x] Own one sampler goroutine and cached view; fake barrier tests prove canceled shutdown joins, blocked IO does not hold interactive locks, and slow passes never overlap or accumulate ticks.
- [x] Count table reads independent of slot count and native calls against the declared bound. Commit.

### 4. Best-effort bounded history and advisory alert delivery

Files: new `couchcore/resource_store.go`, `couchcmd/resources_alert.go`, tests, artifact manifest registration.

- [x] Establish red for `ResourceHistory.Append` and `ResourceHistoryStore.Load/Save` using their function contracts below.
- [x] Implement safe bounded history store and exact storage/source classification. Do not import diagnosticlog's 365-day retention into the rolling history.
- [x] Route effects through `hubBindingAt` / `supervisionCoordinator.sendGenerated`; tests use the real broker with controlled endpoints and receipts. Cover unavailable/moved hub and busy queue.
- [x] Run artifact manifest and retention qualification immediately; commit.

### 5. Cached resources verb, configuration and docs

Files: `couchmessage/protocol.go`, `couchcmd/cli.go`, `messages.go`, `message_service.go`, new `resources_render.go`, README, Atlas entries.

- [x] Establish red for `ParseCLI`, `ValidateRequest`, `renderResources` and `ParseResourcePolicy` using their function contracts below.
- [x] Implement one text/JSON view and strict startup policy parsing. Unknown numeric fields render unavailable; first-sample rates render warming-up.
- [x] Prove repeated resource queries perform zero native observations and no writes. Document defaults, counter semantics, history cap and sampled-orphan limitations. Commit.

### 6. Qualification, review and landing

- [x] Targeted command: `make test-make-env`; use a temporary Makefile including Makefile with a `test-resource-targeted` recipe running `go test ./cmd/internal/procutil ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/couchmessage ./cmd/internal/artifactpath -run 'Resource|ProductionArtifact' -count=1` to retain hosted-test env scrubbing. Expected PASS. Run race tests for the worker/broker ownership seam.
- [x] Add a bounded opt-in native overhead benchmark: representative fleet-size fake graph for exact operation counts, then real local roots for CPU coefficients; record mean/max CPU and counters. Target <0.2% core average at 60s cadence. No expensive sampling loop against all host processes.
- [ ] Live acceptance requires TL-coordinated smoke: staged operator Couch shows every slot; a designated disposable scratch slot writes a bounded file with reduced test threshold/cadence, yielding one hub receipt, sustained silence thereafter, recovery, then one new-episode receipt. No visible window, live restart, or pair:0 staging without explicit authorization. This remains required before claiming Done when.
- [ ] Run the exact recorded suite through `sdlc local-check run --name suite`; if a prerequisite fails, run the rest with `make -k` so no Go failures are hidden. Record all evidence.
- [ ] Close via SDLC fresh review, fix blockers, merge main and `sdlc merge --yes` on SHIP. Report real hashes and measurements to hub. Update the ops project only through its authorized coordination workflow.

## Architecture checks

- **ARCH-DRY:** existing server admission proof, process model, broker delivery and atomic writer; one resource model serves all consumers.
- **ARCH-PURE:** attribution/rates/episodes/history policies are pure; reader/store/worker are IO adapters.
- **ARCH-PURPOSE:** full slot trees, proven orphans, alerts, bounded persistence, CLI and measured overhead all ship together.
- **ARCH-MOCK:** stateful process world and real broker endpoints behind production seams; live owned-child conformance.
- **ARCH-CONSTRAINTS:** bounded cadence, native calls, processes, bytes, history and one worker; explicit partial state on overflow.
- **ARCH-SECURE:** process generation fences, validated bounded metadata, authenticated local socket, no argv/secrets, no cleanup authority derived from metrics.
- **ARCH-ORDER:** reducer owns temporal state; admission supersession, process reuse, send uncertainty, restart and cancellation have explicit transitions/tests.
- **ARCH-FUNERAL:** sampler joins at supervisor shutdown; dead generations retire; byte/time/count-limited snapshots replace old state and expire on reopening. No unbounded telemetry log.

## Sources and pre-implementation findings

- Apple API: https://github.com/apple-oss-distributions/xnu/blob/main/libsyscall/wrappers/libproc/libproc.h and wrapper implementation `libproc.c`; counters: https://developer.apple.com/documentation/kernel/rusage_info_v4 . Use SDK fields instead of reproducing layouts.
- `procutil.Table` currently uses x/sys's retrying `SysctlKinfoProcSlice`; a call to Table is not a literal one-syscall measurement.
- `PairSessionBinding.Owner.Server` exists and is discarded in message admission after validation; the new handoff preserves already-paid-for evidence. Console pane PID alone is a client/helper, not necessarily the session server.
- `diagnosticlog.RetentionPeriod` is 365 days; it does not supply a 24h rolling ring. Namespace source classification and explicit bounded retention avoid conflating these lifetimes.


## Revisions

### 2026-10-15 — TL approval and pre-existing orphan visibility

TL approved bf764e14 via receipt 9f23fae9-2182-488f-8025-196b3940fa66, with one addition: a separate **unattributed (unproven)** view for large launchd-parented processes of known Couch-launched kinds (nvim --embed, Zellij, agent CLIs) that no live slot owns. This is report-only: no alerts, no fabricated slot provenance, and no cleanup authority. Use the configured orphan RSS floor (default256MiB). Classify names through the existing harness catalog; only bounded candidate argv reads distinguish nvim --embed and ambiguous executables. Revalidate identity around those reads. Candidate sampling and command-probe native attempts are counted separately, extending the bound to 8+2P+A, where P includes unproven candidates and A is the measured bounded argv-read attempts, reported explicitly rather than hidden. Cap candidate inspection within the same8192-process budget and disclose overflow. Apply the `ClassifyUnattributed` function contract below. The native conformance/overhead measurement must include this path. Include the section in both text and JSON and label it explicitly as unproven.


### 2026-10-15 — plan gate round1: delivery authority and test contracts

**PQ-1 (supervisor-alert-authority):** Resource alerts are supervisor-generated notices sent from the current owning hub binding to that same exact hub slot, through `supervisionCoordinator.sendGenerated`; the body starts `Couch resource monitor:` and names original process/slot provenance. They never impersonate a disconnected slot or its replacement. Exact-slot self-delivery is supported by the current Broker (family routing alone excludes sender, `broker.go:594`); verify this via a real-broker test. Retain the original verified WorkspaceIdentity with process provenance; resolve its current owning hub through `authority.owner(identity)` and `hubBindingAt`, even after the originating binding disappears. If no hub is live or current ownership is unknown, keep the episode waiting with an explicit unavailable reason, without attributing it to an unrelated hub. The authenticated supervisor alone invokes this adapter; no new socket operation grants a caller arbitrary system-sender authority.

**PQ-2 (stable-delivery-envelope):** Freeze ID, exact from/to bindings, exact route and rendered body at first dispatch, not at threshold start. Persist this envelope and an `attempting` state before calling the broker; checkpoint failure means no send. Record accepted receipt and notified state after admission. Same-ID retries use byte-identical envelope only, including frozen metric text. Before first attempt, re-resolve hub ownership; after an attempt, ownership/hub-binding change never rewrites that envelope. `ReceiptSnapshot` reconciles in-process attempts; after restart use the existing recipient retained-receipt path (`StatusContext`, `broker.go:829`) with bounded lifetime. Confirmed accepted/submitted/indeterminate receipt suppresses a second notice. A crashed or uncertain attempt with missing/expired/unavailable receipt remains `delivery unknown` and visible for inspection; absence alone does not authorize retransmission or a new ID. Definite refusal before admission can retry the identical envelope while its sender/recipient are still current; replacement leaves it waiting/unknown rather than impersonating a newer binding. Valid metric recovery clears the episode, so a later high interval can generate a new ID. This is at-most-once admission per episode with explicit uncertain delivery, not a claim of exactly-once human receipt.

**PQ-3 (function-level-test-strategy):** The following function contracts govern tasks1–5; task steps reference them rather than duplicating scenario inventories.

| Function/method under test | Adversarial strategy and mechanical guard |
|---|---|
| `NativeResourceReader.Table`, `.Usage`, `.Command`, `.SelfCPU` | Stateful native-call seam injects growth/reuse/errors; assert returned generations/capability and exact attempted-call bound including failures; owned-child live parity. |
| `SelectResourceProcesses` | Generated parent graphs with overlaps/cycles/reparenting/reuse; each generation contributes once and only proven roots confer ownership. |
| `AdvanceResources` | Fake-time sequences with counter reset, gaps and partial observations; rates never cross identity/gap boundaries and incomplete evidence cannot satisfy sustained thresholds. |
| `AdvanceResourceEpisode` | Exhaustive state/event sequences including late receipt, retry and restart; at most one immutable admitted envelope per episode, unknown is not recovery. |
| `ClassifyUnattributed` | Known executable/argv candidates, unknown names, PPID changes and PID reuse; only large unowned PPID1 candidates appear, with no alert effects. |
| `ParseResourcePolicy` | Boundary duration/numeric/overflow/malformed inputs; reject invalid startup policy and preserve approved defaults. |
| `ResourceHistory.Append` | Irregular times and oversized aggregate frames; age/count/encoded-byte caps hold and shortened coverage is visible. |
| `ResourceSnapshotStore.LoadCheckpoint`, `.LoadHistory`, `.SaveCheckpoint`, `.SaveHistory` | Portable temp store with interrupted replacement and hostile/truncated/schema/size/symlink inputs; prior valid state survives failed publication and no unvalidated provenance is admitted. |
| `resourceSampler.pass`, `.run`, `.view` | Stateful reader plus explicit barriers for cancellation/slow IO/concurrent queries; no overlap, shutdown joins, view does no IO and never holds interactive locks during probes. |
| `resourceAlertDispatcher.dispatch` | Real Broker with original slot disconnected, moved/restarted hub and controlled retained receipts; hub self-notice succeeds, envelope stays immutable, unavailable evidence never duplicates. |
| `ParseCLI`, `ValidateRequest`, `renderResources` | Text/JSON/query-shape parity and malicious labels; all sections/unknown states explicit, no command IO on cached request. |

No new external dependency or broadened live-slot authorization is introduced by these refinements.

### 2026-10-15 — plan gate round2 presentation refinement

PQ-1 and PQ-2 accepted. PQ-3 required removing duplicate prose test inventories, not merely naming governing contracts. Tasks now reference the function table; integration/live requirements remain. This is a presentation consolidation with no test-coverage or implementation-scope reduction.

### 2026-10-15 — measured cost and implementation naming

TL receipt9f1fdac1-f404-4fd0-9264-df67d2f377f9 clarified the binding budget is well under1% of one core, and authorized a measured <0.2% target after a short optimization look. Native table conversion initially cost74.350ms CPU; bulk SDK conversion removes per-row cgo pointer checks and reduced it to1.643ms mean (same8native attempts in the small native qualification). End-to-end near-cap history qualification (953frames,16slots,4,189,412bytes; ~1007hostrows,27usage observations,58native attempts,2snapshot writes) measured71.331ms mean/74.264ms max CPU across3passes, or0.118885% of one core at60s. Retain simple bounded atomic JSON snapshots; qualify mean<0.2%,max300ms, not the earlier aspirational0.1%. No cadence change, skipped host scan or hidden call. Thread CPU excludes concurrent GC workers; native and full-path numbers remain distinguished.

Existing domain names ResourceObservation and ResourceStore already refer to slot lifecycle resources. Counter observations are ResourceUsageObservation and persistence is ResourceSnapshotStore, avoiding changing those existing nouns. A root observer alone cannot cover draft-only sessions: one startup inventory seed plus lifecycle-triggered proof refresh supplies them, with no probes on sample ticks. Store validates strict JSON and semantics through required callbacks, preserving broker envelope typing in couchcmd without a core/message import cycle. Current-pass final CPU is cached after persistence; history does not claim to measure its own completed write.

### 2026-10-15 — TL sequences live acceptance after close review

Receipt5e56f410-ff46-4aad-b623-660fc4da8ca7: pair:0 remains staged for#544, with#569 awaiting the same restart. TL directs finish recorded suite and close review now, then **hold**. TL will combine#544/#569/#587 for one Couch restart after#584, probably tomorrow morning, with cadence2s/write-duration6s and a designated scratch slot. Only then run all-slot visibility, one write alert, sustained silence, recovery and re-arm. No staging/restart or landing is authorized before that acceptance. The live Done-when remains outstanding; if close requires completed plan checkboxes, use its precise plan-check acknowledgment for this explicitly deferred acceptance, not a fabricated pass.

### 2026-10-15 — close review corrections

BR-1: bound the cached wire projection below the existing broker frame envelope using actual JSON byte sizes. Preserve every slot aggregate; shorten long presentation labels explicitly and allocate independent detail budgets to processes, proven orphans, unproven candidates, episodes and diagnostics, with omission counts. Projection runs inside measured sampling, never on a query. Prove maximum supported cardinalities through a real Unix socket (ARCH-CONSTRAINTS).

BR-2: load checkpoint and history independently. Malformed regular snapshots can rebuild with explicit continuity/deduplication warnings; valid sibling evidence survives. Unsafe paths and transient I/O remain protected and retry rather than permanently latching failure. Recovered checkpoints reset rate baselines before further alert decisions.

BR-3: slot inventory publishes completeness and omitted counts. Startup, cancellation and truncation never prove absence; prioritize live rows under the cap and disclose coverage loss. Positive generation-fenced root death remains evidence. A barrier regression samples while discovery is withheld.

BR-4: correct the integration table's unsupported platform filename to resource_unsupported.go, matching the implemented !darwin || !cgo adapter.

BR-5: production delivery results and restart receipt reconciliation enter the pure episode reducer with immutable effect-ID fencing. Persist and validate admitted/refused/unknown outcomes; adapters no longer assign authoritative phases (ARCH-PURE).

### 2026-10-15 — complete entity traceability sweep

BR-7: enumerated every Core concepts row against its declaration and corrected ResourceUsageObservation, ResourceSnapshotStore and resourceFakeWorld throughout both tables and function contracts. ResourceRootCache/ResourceRootEvidence are the actual proof cache; resourceInventory and Console.ResourceInventory own the inventory adapter. Added the delivered state/value types, resourcePersistence, checkpoint/alert records and projectResourceView with actual files/classifications. NativeResourceReader is declared in resource.go with platform backends; the fake is test-only. ResourceState owns totals rather than history; ResourceHistory owns aggregate frames. This supersedes the earlier planned names, preserving the naming-collision rationale above.

### 2026-10-15 — checkpoint capacity correction

BR-6: preserve the 2 MiB disk cap and encode larger checkpoint payloads in a versioned standard-library gzip envelope, with bounded 64 MiB expansion and full semantic validation. Small v1 JSON stays compatible. No ownership, suppression or exact delivery envelope is discarded to make room. Qualify 8192 process records plus 16896 episodes/attempts, escaped workspace labels, restart equality and real-broker admission/deduplication. Codec CPU remains inside sampler cost; malformed/oversized compressed input fails closed through independent artifact recovery. This corrects the representation rather than raising the approved disk allowance.

### 2026-10-15 — strengthen the checkpoint capacity fixture

The initial sequential-ID fixture fit compression, but a production-shaped fixture (SHA256 effect IDs/generation tokens and 256 distinct escaped workspace identities) exposed that compression alone does not fit the 2 MiB envelope: 32,895,161 expanded bytes. Normalize repeated workspace, process-key and binding authority through typed dictionaries before compression; use checked indices and reconstruct exact values before semantic validation. No authority eviction. Qualify the stronger fixture, including steady persistence and new-alert burst CPU separately; the prior normal-host cost measurement does not establish maximum-cardinality cost.

### 2026-10-15 — authoritative simplification: advisory alerts, no checkpoint

TL receipt e97f725a-c6ac-4ae2-b55f-63604207e0b4 supersedes every earlier checkpoint, durable-intent, restart-deduplication and codec requirement in this plan. Alerts are advisory. The binding CPU budget applies to steady sampling, not a rare maximum-cardinality admission burst. This is the third design delta; simplify the shape rather than adding another persistence mechanism (ARCH-PURPOSE, ARCH-DRY).

**Replacement scope:**

- Keep ResourceState, provenance, rate baselines, episodes and delivery outcomes only in the sampler worker's memory. They die with the supervisor. No checkpoint file, persistent alert attempt, intent/outcome journal, compression codec, normalized dictionaries or publication-proof cache remains.
- Send each pending episode through the existing owning-hub broker. Confirmed admission enters the pure reducer and suppresses repeats while that episode remains high. Unavailable/refused/uncertain sends remain pending for a subsequent sample; reuse the same in-memory effect ID so the broker owns its existing within-run deduplication. No retained-receipt reconciliation or restart identity promise. Recovery clears the episode and a later high interval creates a new one.
- Restart starts fresh rates and episodes from the next observations. A high instantaneous threshold can alert again; sustained thresholds accumulate their configured duration again. A crash can duplicate or miss an advisory alert; no exactly-once claim. Historical ownership is not reconstructed: pre-existing launchd-parented candidates remain explicitly unproven until current evidence establishes ownership.
- Keep the existing bounded ResourceHistory aggregate ring: 24 hours, 1440 frames and 4 MiB. Retain best-effort history.json publication using the existing safe bounded file adapter, with one fixed replacement stage (8 MiB including old+stage). History loss/corruption is diagnostic only; fresh samples and alerts continue. A failed write does not roll back the in-memory ring. No history or disk success gates sending an alert.
- Preserve native sampling, counted calls, cached byte-bounded CLI, cached slot metadata, inventory completeness, unknown coverage, pure rate/episode transitions, unproven candidate reporting and existing policy defaults. No new operator controls or functionality.

**Implementation and verification (after TL approval):**

1. Delete checkpoint/codec/packed/proof-cache files and checkpoint-only tests; shrink the alert adapter to resolve/send/pure outcome. Replace resourcePersistence with a history-only adapter and ResourceSnapshotStore with ResourceHistoryStore. Update actual entity tables and artifact inventory in the same change.
2. Keep real-broker tests for owning-hub notice, one admission per sustained episode, temporary refusal/retry, recovery/re-arm, and restart re-arming from a new in-memory state. Replace durable-intent/reconciliation tests with explicit restart duplicate tolerance and proof that history errors cannot block alerts. Keep corruption/unsafe-path/retention coverage only for history. Preserve transport maximum-cardinality and discovery-barrier regressions.
3. Run targeted tests and race checks, remeasure steady sampler CPU including history, then the recorded full suite and close review. Stage/restart/merge remain held for TL's combined smoke.
4. Report exact net line count against pre-simplification commit 5484a5363, separately identifying discarded uncommitted codec work. Current checkpoint-related group is roughly 2000 lines including tests; expected net deletion is substantial, but measured git diff is the completion evidence, not this estimate.

No production code has been changed for this simplification at the time this Revision is sent. Previously queued full suites were canceled before execution; no full-suite pass or live acceptance is claimed.

### 2026-10-15 — approved simplification implementation map

TL receipt c6eb8f79 approved Revision6af7fb2e. change-code re-ran: plan-quality CLEAN round5 and estimate-quality INFO. The original4.883h estimate remains the historical baseline; this third delta deletes machinery and replaces its tests/requalifies the same item, rather than treating already-incurred design as future work. Actuals include executing tests/reviews/subagents; only genuinely idle coordination/queue waits are excluded.

Current body tables and retention prose now reflect the approved deletion. Earlier Revisions retain superseded evidence; no checkpoint-related table or strategy there describes current implementation. History may contain displayed historical orphan metadata, but it never restores ownership, counters or episodes.

| Current changed function | Strategy replacing checkpoint-era contracts |
|---|---|
| ResourceHistoryStore.Load / Save | Real temporary filesystem: history size/schema, regular/nofollow paths and fixed-stage replacement; no checkpoint API. |
| resourceHistoryPersistence.persist | Corrupt/unreadable initial history starts a diagnostic fresh ring; failed writes retain current memory frames and later safe writes recover. |
| resourceAlertDispatcher.dispatch | Real broker: owning-hub notice, busy queue, refused/uncertain same-ID retries, suppression, recovery and fresh-state repeat. No retained receipt or durable intent. |
| resourceSampler.pass | Stateful reader plus independent alert/history callbacks: history error cannot gate or repeat an admitted alert; restart begins fresh episodes; cache/CPU/native bounds remain. |
