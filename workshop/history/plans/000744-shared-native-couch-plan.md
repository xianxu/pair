---
collaboration-mode: discussion
---

# Shared native disposable-Couch harness

> **For agentic workers:** Follow AGENTS.md §3 for bounded delegation; use superpowers-executing-plans after TL approval and change-code. This is one atomic close boundary, without milestone tags.

**Goal:** Give #722, #740 and #741 one genuinely isolated native fleet whose caller identities and cleanup are qualified once.

**Architecture:** A test-only owner creates fresh roots, provisions real numbered workspaces, launches the real Couch/Pair/wrapper/native-Zellij stack, and controls deterministic agent fixtures. Pure confinement, ownership and agent reducers decide actions; thin process/filesystem adapters execute them. No production admission or lifecycle policy changes. The PQ-1 isolation correction below adds one transport-root seam using existing explicit isolation authority.

**Tech stack:** Go, existing testpair/testbinary/procutil/terminal helpers, PTYs, installed Git/SDLC/Weave/Zellij, current candidate Couch and Pair binaries. No model service, credentials, package install or network.

## Authority, alternatives and scope

TL2d5e80a3 approves native topology and dispatches this issue from main. #722 is parked on its own pushed branch, retaining the claim; do not import its unlanded drain code. Its failed prototype is evidence about seams, not reusable authority. Native Zellij 0.45.1 is present on the development host; qualification records the actual executable/version each run.

Choose native Zellij with private configuration, as demonstrated by `cmd/internal/couchtty/terminal_native_test.go:126–146`. Extending the registration fake is rejected: it has already disagreed with kernel executable naming and daemon/client process groups. A VM-only harness is also rejected here: #743/#745 own VM execution, while the same portable helper should work there later. Existing narrow stateful fakes remain in normal unit tests; native conformance validates the process adapter.

Included: root/environment guards, genuine provisioning and worker launches, actual caller execution, deterministic Claude/Codex fixtures, current-frame/input control, restart and exact teardown, plus one native qualification consumer. Excluded: drain policy, park-all correctness, cold-resume batching correctness, all old-test migration, Qoder, installed-model runs, new production test flags, CI/workflow/local-gate declarations and operator-slot staging. The helper supplies control and native conversation artifacts needed by the consumer issues; their scenario-specific acceptance remains there.

## Core concepts

| Pure entity | Lives in | Status |
|---|---|---|
| RootsPlan / ValidateRoots | cmd/internal/testcouch/roots.go | new |
| OwnedRun / ReduceRun | cmd/internal/testcouch/ownership.go | new |
| AgentState / ReduceAgent | cmd/internal/testcouch/agentfixture/model.go | new |
| Claude/Codex Paint | cmd/internal/testcouch/agentfixture/paint.go | new |
| Command / Result values, ValidateCommand / DecodeCommand and ActorDirectory | cmd/internal/testcouch/agentfixture/protocol.go | new |

RootsPlan names every mutable root and the explicit read-only tool/source inputs. OwnedRun owns 1:N children, native sessions and IO pumps; it accepts observations about exact generations, never naked PIDs. AgentState is one deterministic fake agent with two supported personalities, not a new production agent kind. Actor requests identify a fixture actor and request ID, never a fabricated Couch binding. ARCH-PURE/DRY: one owner for each fact, and no shadow terminal or broker protocol.

| Integration | Lives in | Status | Wraps/reuses |
|---|---|---|---|
| Candidate tool placement | cmd/internal/testcouch/binaries.go | new | testpair.Binary/At, testbinary.Cache pattern, go build, explicit native executable discovery |
| Root lease and empty environment | cmd/internal/testcouch/environment.go | new | testpair.Environment/EnvironmentFrom, production isolation contract |
| Git/provisioning | cmd/internal/testcouch/provision.go | new | actual Git, Couch provision-workspace, installed SDLC/Weave |
| Fleet / ActorCLI / frame and input control | cmd/internal/testcouch/fleet.go, actor.go | new | real public CLI, PTY, existing terminal emulator, private fixture control |
| Native session/process owner | cmd/internal/testcouch/process.go | new | procutil generation checks, existing native terminal configuration and shutdown patterns |
| Mailbox publication, ReadResult correlation, retirement and cancellation | cmd/internal/testcouch/agentfixture/protocol.go, run.go | new | private filesystem, admission lock, joined process-group runner |
| Fixture executable | cmd/internal/testcouch/agentfixture/cmd/main.go | new | real agent stdin/stdout, controlled CLI child processes, native transcript fixtures |
| First live consumer | cmd/couch/fleet_acceptance_test.go | new | shared helper; no private fleet implementation |

Keep the helper independent of couchcmd/wrapcmd implementation packages to avoid consumer import cycles. Typed protocol/socket derivation may reuse couchmessage; a couchcore test migrating later must use an external package or executable boundary. Do not duplicate production socket hashing or process-identity algorithms. Build candidate binaries once per qualification owner, pass them through subcases, and clean that owner's build directory after all children end; no immortal new global cache.

## Public helper shape

`Build(t)` returns absolute candidate Couch/Pair and installed tool paths, with source head/version evidence. `New(t, binaries, Options{Workers, AgentKind, RuntimeBudget})` allocates an owned fleet; `Prepare(ctx)` provisions it and `Start(ctx)` starts the sole steward. `StartWorker(ctx, n)` uses the genuine steward's CLI. `ActorCLI(ctx, actor, argv...)` returns stdout/stderr/exit plus the actual public JSON, without reconstructing identity. `Input`, `Resize`, `Screen`, `Detach`, `RestartSupervisor` and idempotent `Close(ctx)` operate only on that fleet. Consumers can inspect fixture-owned conversation IDs and accepted-input logs; those are observations, not authority to call another fleet.

AgentKind is fleet-wide initially: a Claude subcase and a Codex subcase cover both adapters. Mixed-agent cohorts can be added at the existing preference seam when a consumer needs them; there is no arbitrary generic extension framework. The deterministic engine provides controlled working/idle/picker states and correlated command execution. Native conversation files use the existing sessioninventory fixture schemas under private agent homes, so lifecycle consumers can script checkpoint/resume behavior; do not fake Couch records, ready files, bindings or setup success. The helper does not claim a stand-in agent proves a real installed model's reaction.

## Isolation and genuine provisioning

The factory creates fresh mode0700 temporary roots, preferring TestScratch and falling back to TMPDIR. Callers cannot attach the helper to arbitrary existing roots. Retain ambient live paths only as a deny-set for validation, then construct the child environment with the canonical empty-environment owner. Explicit read-only exceptions are current source/runtime assets and absolute tool binaries. All writable configuration/artifacts must remain private.

Own HOME, Claude/Codex homes, all XDG roots, Couch isolated/store/identity roots, Pair data, native Zellij socket/config/data/runtime, fixture control and SDLC queue roots. Set `WF_SDLC_QUEUE_ROOT`, `WF_TEST_LOCK_FILE` and `WF_SDLC_QUEUE_FLEET_ROOT` together. No inherited COUCH_TREE/binding/session, native-agent credentials, global Git config, Zellij identity or session-sync authority crosses the boundary. Pin PATH to the owned agent/wrapper directory plus explicit required tool directories; never fall back to installed Claude/Codex.

Canonicalize physical paths before effects; reject empty paths, symlink escapes, direct live-root reuse and an attempted ancestor covering live state. Long AF_UNIX paths use a second fresh short socket root with the same run owner, never the live socket namespace. Under explicit COUCH_ISOLATED_ROOT, the canonical transport derivation will place broker and wrapper sockets in an owner-private message directory derived from that root; no listener in that fleet opens the ambient per-UID message parent. Use canonical endpoint hashing unchanged and assert both directory confinement and disjoint addresses. Reject an invalid, escaping or overlong isolated socket root before creating a listener; the harness chooses a fresh short isolation root when needed. Native Zellij always runs through the actual absolute executable; no fake daemon, PTY relay or symlink pretending to be a native terminal.

Create a minimal Git primary and bare upstream/main under the owned root, committing a minimal construct/base.manifest and no Brewfile/Makefile. Reuse the topology of TestProvisionConformance, not its ambient-environment constructor. Invoke actual `couch --internal provision-workspace PRIMARY --slot=N --remote=upstream`; validate real `sdlc workspace --json`, upstream/HEAD and production-generated setup markers. No synthesized identities or markers. A dependency source, if required, uses a fixture HTTPS identity rewritten only to an owned bare remote, as existing real provisioning tests do; no network is allowed.

Start Couch with only PRIMARY as its hub. Once the steward is genuinely admitted, its fixture agent executes `couch --reboot fixture:N --confirm --json` for each prepared recordless worker. The existing slot-operation/reboot contract permits recordless fresh recovery and retains normal ownership/version/workspace checks. Verify real actors show exactly one hub and N non-hub workers. Agent selection comes from an explicit private PAIR_AGENT at the supervisor boundary, rather than borrowing the operator's setting.

## Agent and execution seams

Build one deterministic fixture executable and install physical owned files named claude/codex; version/profile and kernel-name checks must reflect the actual executable. Reuse pinned composer captures/profile expectations from wrapcmd testdata and native transcript schemas from sessioninventory. Version queries exit immediately. Accepted peer text traverses actual stdin, composer recognition, paste, submit and receipt confirmation; do not mark an input received merely because bytes were pasted.

Fixture control is separate from ordinary stdin. Commands/results are bounded and request-correlated, published atomically under the owned root. ActorCLI asks the already-running actor to spawn the real Couch command with its actual inherited environment; the parent never pastes a captured binding into a request. Permit at least three concurrent correlated CLI children from one steward for #741, with a small explicit cap and joined cancellation. A consumer-owned workspace-discovery barrier may delay an actual SDLC invocation for the held-cold case, but must forward its real output unchanged rather than inject warming state or fabricate identity.

Root/command/process adapters have stateful portable doubles for unit tests: owned folders, command invocations, process generations and lifecycle events. Native conformance drives the same adapter boundary with real tools. No function-call-only mock can qualify native daemon/session behavior. Future agent personality growth is isolated to the existing adapter contract, not another fleet implementation.

## Ownership, ordering and recovery

Actions: inspect diagnostics; close the owned run; explicitly rerun a fresh fleet. One fresh root lease is the single resource preventing concurrent ownership of the same writable state; no import-existing-root API exists. A run-local creation gate closes before teardown so no child can appear after the final ownership census. Register ownership before exposing a process/session to a consumer, and feed all completion/cleanup observations back to ReduceRun.

| Internal state / failure | Visible result | Recovery | Possible loss |
|---|---|---|---|
| Invalid/ambient/escaping roots | refused before effects | rerun with factory-owned roots | none |
| Prepared, starting or ready | owned fleet; current observed phase | close run | transient fixture interaction |
| Partial start or consumer cancellation | failed with retained ownership ledger | close run | partial test output if not captured |
| Closing; exact children/session generations remain | cleanup pending | close run | none claimed complete |
| PID reused, session ownership unknown or cleanup deadline | cleanup failed with exact diagnostics | inspect diagnostics before rerun | residue deliberately remains; no foreign process is killed |
| All direct children/IO joined and owned daemon generations absent | cleaned | rerun fresh fleet | temporary fixture data intentionally removed |

Close cancels request producers, joins CLI/control workers and IO, stops only private native sessions and generation-verified children, then observes daemon absence before deleting paths. Every native kill uses the private configuration/socket root and exact owned session; no delete-session against ambient state. Reused PIDs are not signalled. A daemon is not an OS child that can be Wait'ed: prove the exact generation absent; direct children and readers are joined. A timeout is failure, not absence. Repeated Close only reconciles the same owned resources and cannot signal a newer generation.

Successful runs remove all owned roots, sockets and binaries. Emit bounded diagnostics to the test log before cleanup; the invoking named target captures its full output as workflow evidence. Unknown cleanup deliberately retains the run's manifest/root and fails the test with exact recovery facts; do not start another fleet in that qualification process until it is resolved. Parent SIGKILL/host crash cannot guarantee teardown and must remain an explicit limitation with an inspectable owned manifest, not a claim of cleanup. Existing temporary-root/host reclamation remains the last resort; no new persistent global sweeper or unlimited artifact archive.

## Operating envelope and tests

Initial native qualification: one active fleet, one steward plus three workers, two serial agent-personality subcases. Helper supports1–8 workers; pure count tests use8/16 synthetic members without spawning16 native slots. Preparation/build budget initially180s; runtime120s per helper qualification subcase and teardown15s, all provisional bounds to measure. #722 may request its own approved90s scenario budget. Explicit opt-in must fail, not skip, if required native tools are missing; default ordinary test execution skips only the native qualification.

Growth: O(N) provision/launch/ownership records, one fleet observation snapshot per explicit check, bounded control concurrency and per-slot capped input/diagnostic buffers. No idle poller. Count creation, snapshot and cleanup effects at N/2N and reject repeated fleet census per member; wall time alone is not the guard. All command output has a declared byte cap, and stale response IDs cannot complete newer commands.

Risky pure functions: ValidateRoots → table/property tests over empty, ancestor, alias and path-traversal inputs, with physical symlink checks in its thin adapter; ReduceRun → generated lifecycle orderings covering partial start, cancellation, late completion and PID reuse, with a stateful process double; ReduceAgent/ValidateAgentCommand → malformed bytes/argv/request IDs and reordered responses, seeded fuzz tests plus pinned independent frame/transcript fixtures. Native adapter tests observe actual process identities and socket roots, rather than mocking a binding.

## Implementation tasks — one close boundary

- [x] Write confinement, ownership and agent-protocol tests first, observe the intended failures, then implement pure entities and portable stateful doubles. Run `go test ./cmd/internal/testcouch/... -count=1` with native opt-in absent.
- [x] Implement actual candidate/environment/provisioning adapters, using testpair and existing native patterns. Write the live-fleet guard before allowing any process start; prove a forged ambient root causes zero command effects.
- [x] Compose native Fleet/ActorCLI/frames/restart/Close and genuine worker startup. Qualify actual registration, concurrent actor commands and exact teardown through the first opt-in consumer in cmd/couch/fleet_acceptance_test.go; do not add a private fleet beside the shared owner.
- [x] Add `make test-couch-fleet` in Makefile.local with `PAIR_NATIVE_COUCH_FLEET=1` and an explicit overall timeout; no .sdlc declaration. Record current source head, native versions, isolation manifest, elapsed setup/runtime/cleanup, exact receipt/body evidence and full captured log/hash. Preserve failures; classify any repeated same-head flake through the existing SDLC rule.
- [ ] Update atlas/couch.md and atlas/index.md (new atlas/couch-testing.md if needed) and audited reader/artifact inventories. Run focused races, guard/adapter tests and actual native qualification; then pushed-head required suite, mandatory close review and ordinary merge. Only after landing, resume #722, integrate landed main and replace its private prototype with this helper. #740/#741 retain their own acceptance and issue ownership.

No implementation begins until TL reviews this plan. Derive an estimate only after plan-quality; current decomposition is not a measured duration or a performance claim.

## Revisions

### 2026-10-20 — PQ-1: confine listener startup effects, pending TL scope approval

The original plan kept the per-UID message parent unchanged and checked only distinct endpoint addresses. That is insufficient: transportListen calls collectDeadEndpointSockets over its whole parent (couchmessage/transport.go and socket_cleanup.go), so an isolated fleet could delete foreign proven-dead wrapper residue. Address the whole startup/cleanup authority class by extending the existing COUCH_ISOLATED_ROOT contract to message-directory derivation. This is a narrow production transport seam, replacing the original no-production-change assumption for this boundary only; normal unisolated addresses, protocol, admission and cleanup policy remain the same.

SocketPath reads the explicit isolation selection at its IO boundary and passes it to a named pure message-directory/address selector. Reuse the existing physical-root confinement helper by extracting it to a dependency-free internal owner if required, rather than duplicate symlink rules. Validate the namespace and resolved message parent remain physically within that isolation root. Independent broker, wrappers and actor CLI derive the same private parent from inherited isolation authority. Do not add a second socket-root override, bypass cleanup, introduce a fake listener or rely on TMPDIR. The factory owns a short fresh root, and rejects address-length overflow before launch.

Add a regression using actual StartServer in a disposable parent containing a foreign dead-wrapper socket: isolated startup must leave that foreign inode untouched and create/clean only the owned directory. Unit-test the pure selector over explicit/unset/invalid/escaping/overlong selections; qualify broker and wrapper derivation together, and preserve normal-mode compatibility. Sweep all message endpoint construction and listener startup paths. The first native consumer must assert its transport directories are under its owned root before any actor command. TL approval of this narrow seam expansion is required before implementing it; the plan-quality ledger remains blocked until reviewed.

### 2026-10-20 — TL3d5e6782 approves the PQ-1 scope correction

Verified submitted TL approval: the new transport parent is active only with COUCH_ISOLATED_ROOT set. With it unset, pin byte-for-byte existing live socket paths and unchanged sweep behavior in compatibility tests. Invalid or overlong isolated paths must refuse before any effect. Require the actual listener-startup foreign-residue regression in a disposable shared parent. The previous revision's pending-approval condition is now satisfied; rerun plan-quality before implementation.

### 2026-10-20 — native qualification exposed #747

The first genuine worker launch exposed reserved-vs-established recordless Fresh authority; the harness does not weaken that product check. TLd31f383f chose #747 independent landing on its actual emitted-profile regression, followed by #744 integrating landed main and running the real three-worker matrix for both issues. No composite or stacked source build. Public singleton adoption preview/apply is now part of real Prepare; native fixture shutdown uses cancellable TTY IO after a real-PTY regression reproduced blocking stdin Close. A failed personality stops qualification before a fresh fleet; old failed generations were recovered by exact ownership, leaving diagnostics only. These refine setup/teardown execution, with the same scenario scope and acceptance.


### 2026-10-20 — pre-rerun adapter audit after four native discoveries

TL69cdbd1e requires a class-wide sweep before another native run. The issue Log records all key/query/deadline/sleep/poll findings and regressions. Execution refinements: physical console keys derive from observed negotiated mode and switcher state; actor CLI carries its caller budget; versions share the declared build budget; foreground command cancellation reuses the existing OSProvisionIO group owner; typed quota refusal terminates observation rather than masking exhaustion. Fixed sleeps do not establish acceptance. Retain the same native topology, explicit outer budgets and scenario contract; no timeout increases or product admission bypass. Shutdown grace/pipe-drain policies remain distinct from readiness allowances.


### 2026-10-20 — reclaim consumed fixture query results within the existing bound

The sweep rerun passed physical detach, then correctly reported the unchanged128-request limit during restart readiness. The original result lifecycle retained every completed observation and its seen/processed bookkeeping until fleet teardown; even a short healthy reconnect could exhaust that lifetime budget. Preserve128 as the maximum unread results and pending mailbox size, not a larger cap. ActorCLI explicitly retires only its matched consumed immutable ordinary result. The fixture retains only active Exec IDs in memory; unread result files reject completed-ID reuse. Admission reserves completed plus active slots so later completions cannot exceed128. Retirement frees the slot; no lifetime tombstone list or counter grows with observations. IDs remain unique per driver and generation-scoped actor control, with no automatic mutation retry. The single reserved shutdown request/result remains independent. Count more than128 consume/retire cycles, pin unread-cap refusal and in-flight duplicate handling before rerunning native. Public Couch identity/protocol and scenario budgets remain unchanged.


### 2026-10-20 — publication and retirement handoff fence

Keep each original command file until its result is consumed, so a completed-ID handoff cannot unlink a reused request. Retirement releases the immutable command before its matching immutable result; stale directory snapshots skip confirmed missing retired entries. One private .admission.lock inode per actor serializes only bounded validation/publication across writers and dies with the root. Count actual published requests/results, excluding atomic-publication scratch names; never count a transient second hard link as another request. Eight rounds of32 writers at127 pending plus a held scratch publication must admit exactly one request to128. This closes concurrent quota admission as well as serial result reclamation, without a larger cap or timeout.


### 2026-10-21 — BR-3: classify the implemented core and filesystem seams

The previous pure table combined AgentCommand/AgentResult validation and request correlation and placed paint in model.go. The implemented names are Command/Result; validation/decoding and path selection are deterministic, while ReadResult and RetireResult read/remove actual mailbox files and belong in INTEGRATION. Paint is pure in paint.go. Physical root discovery remains in the integration adapter, separate from ValidateRoots over supplied physical facts. This corrects the current table rather than inventing a second pure correlation API (ARCH-PURE/DRY).

### 2026-10-21 — BR-2: caller cancellation owns the published request

Verified submitted TL approval8aa5e124 accepts this ordering refinement. Exec carries its absolute caller deadline, fixed at admission rather than renewed when a queued request starts. A correlated cancel marker is bounded by its existing command slot. Queued cancellation suppresses spawning; running cancellation cancels and joins the existing process-group runner. Only a terminal Result after join acknowledges completion. The caller retires command/result/marker and returns the original cancellation. Cancellation cleanup shares the existing15s cleanup budget; exhaustion or publication failure is explicit uncertainty, never success or absence. Admission and retirement serialize marker/ID reuse so a stale cancellation cannot affect another request.

| Request state / event | Transition and effect | Caller result / recovery |
|---|---|---|
| Queued / cancellation or expired absolute deadline | suppress spawn, publish terminal cancellation result | retire all correlated files; return cancellation |
| Running / cancellation or expired deadline | cancel child context, join process-group runner, then terminal result | retire only after acknowledgment; return cancellation |
| Terminal / cancellation racing result | correlate the same terminal result, retire once | original caller cancellation wins |
| Cancellation / acknowledgment unavailable or cleanup budget exhausted | retain uncertain request evidence for owned fleet cleanup | explicit cleanup error; Close owns reconciliation |

The already successful native run remains recorded. BR-1/BR-2 change the harness, so the final code head must run both three-worker personalities again before landing, alongside focused regressions and the requeued recorded suite. No timeout increase or live-root access is authorized.


### 2026-10-21 — BR-1/BR-4: complete isolation consumers and recovery evidence

BR-1 enumerates every COUCH_ISOLATED_ROOT consumer. Actual service tests in stale_store_test.go, installed CLI setup in cmd/couch/main_test.go, the opt-in stale-session native test and tests/with-isolated-pair.sh need owned short roots after the transport contract change. Pure/fake singleton, capture and deliberate invalid/overflow tests retain their intended inputs. testpair.ShortRoot supplies a canonical0700 short root with test-owned cleanup; the shell wrapper follows the same short-parent selection. Existing private-parent confinement and overlong-path refusal stay unchanged (ARCH-PURPOSE).

BR-4 changes ownership.json publication to the existing durablefile atomic writer. ReduceRun still records authority in memory first. The fleet retains the first publication error, refuses further new operations, and reports it on startup/Close; cleanup still joins and reconciles exact owned generations and removes roots once proved clean. A historic publication failure alone does not leave otherwise-clean roots behind. Tests hold a previous snapshot reader through replacement, force real publication failure while checking prior evidence and retained memory, and prove an actual owned process is still cleaned with the error surfaced (ARCH-FUNERAL/DRY).

### 2026-10-21 — BR-2 leader-first completion: proposed retained uncertainty

Close round2 verifies BR-1, BR-3 and BR-4 addressed and independently passes both native personalities, but keeps BR-2 Critical. Root regression TestCLILeaderExitCannotAcknowledgeLiveDescendant is RED: inherited-pipes returns terminal ErrWaitDelay in1.01s with descendant alive; closed-pipes returns success in0.01s with descendant alive. The regression terminates only its observed matching descendant generation afterward. The queued suite lease75766965 was canceled through SDLC before execution; no suite result is claimed. The final native PASS107.35s remains retained, but must be repeated after any lifecycle correction.

Proposed refinement, awaiting TL approval: preserve the current generation-qualified live-leader cancellation; never copy a raw numeric-PGID kill after its identity has disappeared. Observe exact group quiescence after every completion, including success, child error, cancellation and pipe-drain cutoff. A nonempty or unreadable group is a typed CleanupUncertain result, not terminal acknowledgment. RetireResult must refuse it before removing command/result/cancel files. ActorCLI surfaces explicit cleanup uncertainty, records the existing OwnedRun unknown/stuck outcome and refuses new fleet work; existing Close performs private-run census and generation-qualified descendant reconciliation. If cleanup cannot prove absence, preserve the owned root/manifest as it already does. This uses the existing bounded request slot and fleet owner, without another persistent store or timeout increase.

| Internal outcome | Visible result | Recovery | Possible loss |
|---|---|---|---|
| Leader complete and group proved empty | ordinary terminal result; retire correlated slot | continue | none |
| Leader exited, descendants live or group inspection unknown (pipes held or closed) | cleanup uncertain; correlated evidence retained; no new work | Close owned fleet | transient fixture work is stopped during exact cleanup |
| Leader live, deadline/cancel | existing generation-fenced stop, then group observation | retire only if quiescent; otherwise Close | interrupted CLI work |
| Owned cleanup cannot prove generations absent | cleanup failure with retained root/manifest | inspect evidence before another fleet | residue deliberately retained |

Simplest rejected alternative: copy OSProvisionIO's post-WaitDelay numeric group kill; it neither preserves incarnation authority after leader exit nor confirms absence. A new per-request anchor could supply that authority, but adds a process/control protocol when the existing fail-closed uncertainty/Close path already expresses this failure. Regression sweep: leader-first with inherited/closed pipes; ordinary success/error; deadline and explicit cancellation; unreadable group; no retirement/ID reuse of uncertain slot; refusal of new work; owned exact cleanup of a retained descendant. Keep process/IO joins bounded and errors visible, retain original native runs/#747 checksum, then rerun focused races, native matrix, recorded suite and full close. No implementation of this refinement begins until TL approves.

### 2026-10-21 — BR-2 approved completion table and implementation checkpoint

TL approval cf5a7dec-c77f-45e5-b240-e059f1460666 authorizes the retained-uncertainty refinement above. Every successfully started Exec now observes group quiescence after Wait. No signal other than signal-zero observation is sent after the leader incarnation is gone. Result carries typed cleanup uncertainty; retirement refuses before unlinking, ActorCLI preserves command/result/cancel evidence and records the existing sticky OwnedRun failure, and all fleet mutators (including Input and Resize) share the admission gate. Existing Close owns generation-qualified cleanup; uncertainty is cleared only after proved absence and root removal (ARCH-FUNERAL).

| Completion path | Visible state and recovery | Regression |
|---|---|---|
| Normal exit, group absent | terminal result; consume and continue | TestCLICompletionPaths/normal |
| Nonzero exit or self-signal, group absent | terminal failure; consume and inspect command error | TestCLICompletionPaths/error and signal |
| Leader exits first, live descendant, inherited or closed pipes | typed cleanup uncertainty; retain slot; Close fleet | TestCLILeaderExitCannotAcknowledgeLiveDescendant |
| Live leader cancelled or deadline reached | incarnation-fenced cancellation, then group observation; consume only if absent, otherwise Close | TestCLILiveLeaderCancellationJoinsBeforeCompletion and existing cancellation tests |
| Group inspection unknown | typed cleanup uncertainty; retain slot; Close fleet | TestCLIGroupObservationClassification |
| Uncertain slot reaches ActorCLI | retained command/result/cancel; refuse new work including console mutation | TestActorCLIUncertainDescendantRetainsSlotAndStopsNewWork; TestUncertainFleetRefusesConsoleMutations |
| Retained private descendant is visible to owned census | Close stops only its matching generation, joins and removes root | TestUncertainLeaderDescendantIsCollectedByExistingClose |

Focused RED/GREEN: Input and Resize returned no-active-console instead of typed uncertainty before their gate, then both passed. The cleanup-proof fixture initially mixed StrictIdentity with the fleet's native Identity token; after correction it also demonstrated the documented Darwin limitation that system sleep exposes no environment. The proof now re-execs the user test binary with the private run marker (the same supported environment-observation boundary as procutil's native regression), reparents it through a shell leader, and passes real census plus Close in0.46s. Shell descendants without readable environment require the existing same-census ancestry/private-TTY witness; this test does not claim arbitrary orphan authority. No production census authority was broadened. Prior direct CLI/signal/result-retention focused tests passed; complete races/native/recorded suite/full close remain pending on this code head. Original native evidence and #747 cross-reference/checksum remain retained.

### 2026-10-21 — final approved BR-2 qualification and concept mapping

The final native matrix passes on clean unchanged head b25d7b0bff019b43f2a0c0e67a8c058cfd242b44:150.09s both personalities, with full receipt/body/restart/cleanup proof, recorded in final-br2 evidence. Original runs and #747 traceability remain in the issue Log. The completion table above maps all outcome paths to regressions. Additional pure concepts are cliGroupError (agentfixture/run.go: observation-to-error classification) and Result.CleanupFailure (protocol.go: typed result projection). Their filesystem publication, cancellation-marker retention, retirement and Fleet admission remain integrations under the existing owner. The catalog guard is syntax-only; completion regressions supply behavioral proof. Full recorded suite and fresh full boundary review still precede landing.

### 2026-10-21 07:34 PDT — third-delta re-plan: one request lifetime, all uncertain exits (pending TL approval)

Round3 REWORK proves the earlier correction covered uncertain terminal Results but missed early returns when CancelCommand cannot publish or the acknowledgment cannot be read/arrive. Reviewer made cancellations read-only after the child started: ActorCLI returned an error while the child remained alive, then admitted another command. Root cause: uncertainty authority is currently invoked by result content, not by the lifetime of the admitted request. This is the third correction on this mechanism; implementation stops for a consolidated re-plan rather than another local branch patch (ARCH-ORDER, ARCH-PURPOSE).

One resource: the already-published correlated mailbox command is owned until joined terminal proof and successful retirement. Keep the existing admission lock, bounded command/result/cancel slot, and OwnedRun sticky uncertainty. No new process, registry, timeout, retry or persistence mechanism. Once WriteCommand succeeds, arm one deferred outcome guard in ActorCLI. Only successful non-uncertain ReadResult plus RetireResult disarms it. Every other return routes through the same existing unknown transition and wraps ErrCleanupUncertain with the original caller/IO error. This removes result-only authority and avoids a new boolean per failure branch. The guard is one request-lifetime proof, not an alternate fleet state machine. An ordinary command error after proved retirement stays an ordinary terminal error. Before publication, validation/readiness/admission errors remain effect-free; atomicJSON returns the publication link/rename result directly, so a failed WriteCommand has not admitted a command.

| Internal request outcome | Visible state | Recovery | Possible loss / regression |
|---|---|---|---|
| Rejected before command publication | ordinary refusal; no request effects | correct input/capacity and retry | none; existing admission tests |
| Joined terminal success/error/signal and slot retired | ordinary result/error; fleet remains usable | continue or inspect CLI error | command failure only; completion table tests |
| Caller canceled, cancellation published, joined acknowledgment and retirement succeed | original cancellation; fleet remains usable | continue | interrupted CLI; existing joined cancellation test |
| Cancellation publication fails | typed cleanup uncertainty, command and available evidence retained, new work refused | Close fleet | command may still run; real read-only cancellations regression with live child |
| Result/ack read fails or correlation is invalid | same typed uncertainty and gate | Close fleet | terminal outcome unknown; private stateful mailbox corruption regression |
| Cancellation acknowledgment budget expires | same typed uncertainty and gate, unchanged15s cleanup bound | Close fleet | pending/active request outcome unknown; private held-mailbox deadline regression |
| Result says group live/unknown | same typed uncertainty; preserve cancellation marker where writable | Close fleet | transient command work; existing leader-first open/closed pipes tests |
| Terminal retirement partially fails | same typed uncertainty, preserve remaining files; no rollback that invents evidence | Close fleet | some correlated files may already be removed; private filesystem refusal regression |
| Close cannot prove owned generations absent | retained root/manifest and cleanup error | inspect evidence before fresh fleet | deliberately retained residue; existing Close failure controls |

Implementation steps after approval:
- Add RED filesystem-backed tests for cancellation publication failure (real running CLI child and attempted subsequent ActorCLI/Input/Resize), acknowledgment read/correlation failure, actual15s acknowledgment exhaustion, and retirement failure. Use the same private mailbox seam, not function-call mocks. Confirm original caller errors and typed uncertainty both survive; assert command/available evidence retention, no new effect marker and sticky admission refusal. Restore only owned fixture permissions and perform generation-qualified cleanup in test cleanup.
- Consolidate ActorCLI finalization in cmd/internal/testcouch/actor.go: named returns and one post-publication deferred guard; mark retired only after both joined-result validation and successful retirement. Preserve current normal/cancellation behavior, the result group classifier and no post-leader numeric signal. RetainUncertainCLI becomes a best-effort evidence helper, with the lifetime guard as the sole error-to-sticky-outcome owner. No new PURE entity; the named-return outcome adapter and filesystem retirement remain INTEGRATION; ReduceRun and Result.CleanupFailure remain existing PURE projections.
- Sweep every ActorCLI return and update the catalog guard to catch bypass of the completion owner where a syntax rule can defend the class; document its limits. Re-run helper races, catalog/artifact/composer/transport guards and individually audit changed reader certificates.
- Classify BR-5 from retained reviewer transcript and failure nonce operation-730874d8fa66046c985836b933151494 before any acceptance rerun. Reviewer observed native failure103.75s and unchanged pass107.43s, cleanup successful both times. Preserve both; if independently intermittent, register its exact test on an owning flake issue; if a reproducible product defect blocks qualification, file it in Pair and report to TL for sequencing. No blind native retry or claimed acceptance around an unexplained failure.
- After classification and code checks, rerun final real Claude/Codex three-worker matrix once, retain all prior evidence/#747 checksum, record suite on pushed final head, full close, then merge on SHIP. Prior lease214519138 was canceled through SDLC while queued; no suite result claimed.

Simplest rejected alternative: add the unknown call at the two reviewer-named returns. That leaves future early exits and partial retirement outside the lifetime authority, repeating this defect family. A new per-request process supervisor is also unnecessary: the existing fleet owner and correlated slot already carry cleanup authority. Public helper CLI, isolation, bounds and scenario scope are unchanged. No code changes until TL approves this re-plan.

### 2026-10-21 — third-delta re-plan approval and BR-5 disposition

TL approval ff2a11b1-7eaa-47eb-ab7c-3fa515f7801c approves the single post-publication lifetime guard above: disarm only after joined non-uncertain result AND retirement; all other returns latch typed uncertainty, preserve available evidence and delegate Close. The four new filesystem-backed tests are RED before implementation: read-only cancellation publication leaves a real CLI child alive; malformed acknowledgment returns decode error; held acknowledgment expires at the unchanged15s cleanup bound; command unlink permission failure leaves partial retirement. Every case previously returned without typed uncertainty. The implementation uses one deferred admitted-request guard, with no new fleet state/resource/protocol or timeout.

BR-5 is an independently intermittent production attach/PTY-lifecycle failure, registered on unclaimed pair#772 for exact github.com/xianxu/pair/cmd/couch.TestDisposableCouchFleet/codex. Preserved reviewer failure103.75s and unchanged pass107.43s under workshop/evidence/000744/review-round3-native-{fail.log,pass.log,evidence.json}. The observed endpoint ended within7.1845s during worker3 attach, not a120s scenario timeout; existing projection cannot identify check site, endpoint termination cause, direct child exit or last output. No root cause is claimed. TL directs744 to land without waiting on772, with one approved scope addition: retain the next failure's exact attach site, actual endpoint ended observation/exit status when observed, and bounded recent output before rollback. Reuse LifecycleFailure's existing Stage/BeforeExit/Stream/Output/LaunchError fields; nil exits remain explicitly unobserved. Capture an immutable typed diagnostic error at attach rejection, adopt it into the same operation attempt/nonce, and include the existing fields in the harness's pre-cleanup projection. This is diagnostic evidence, not a fix or retry for772; endpoint termination cause unavailable from current API stays explicit. Regression tests exercise preflight/precommit snapshots, bounded copied output, observed versus unknown exit and unchanged attempt identity. After744 landing, this slot takes772 separately from main per TL dispatch.

### 2026-10-21 — diagnostic addition core concepts

| PURE entity | Lives in | Status | Test |
|---|---|---|---|
| attachFailure immutable diagnostic value and TerminalFailureEvidence owned projection | cmd/internal/couchtty/attach_failure.go | new | value-only copy/error projection regression in attach_failure_test.go |

| INTEGRATION | Lives in | Status | Wraps/reuses |
|---|---|---|---|
| captureAttachFailure | cmd/internal/couchtty/attach_failure.go and console.go rejection sites | new | live child endpoint, bounded snapshot, nonblocking observed-exit inspection; stateful fake-child preflight/precommit tests |
| lifecycleFailureForAttempt terminal evidence adoption | cmd/internal/couchcore/operationattempt_record.go | modified | existing LifecycleFailure fields and operation authority; owned copies, target/nonce preservation |
| nativeFailureEvidence exit projection | cmd/internal/testcouch/cleanup.go | modified | existing failure store; observed/unobserved before/after exits, original pre-cleanup bytes/site |

No new persisted noun, lifecycle status, operator action, timeout or retry is introduced. The diagnostic value owns its output and exit pointer; callers cannot mutate recorded evidence. After-exit observations remain distinct from the original rejection snapshot.

### 2026-10-21 — TL sequencing: product budget issue773 and expanded private evidence

TL ac4d118b-2fb9-4022-ba58-bf35b9d23f7f directs filing fixed10s readiness as its own product issue (done: unclaimed pair#773), then extending744 pre-cleanup evidence in scope. Capture bounded private pane/wrapper startup stderr, any AGENT-READY write failure, effective scope/tag/session/nonce, and ready-file sibling census using canonical emitted-profile/artifact owners. Retain only allowlisted identity keys, never complete environment/credentials; cap counts and bytes, show missing/read failures, and reject symlink escapes. Add filesystem-backed regression with real path/schema shapes and escaping/truncation controls. No startup timeout change, AMFI warm-up, product readiness fix or retry policy is part of744. Once capture is verified, run one final native matrix; if it passes, continue suite/close/landing with the original failure attached to773; if it fails, classify from the new capture. #772 remains registered for the exact Codex attach subtest and is this slot's next issue after744. Existing native failures remain failures, retained alongside successful runs.

### 2026-10-21 — approved diagnostic source contour (6e0eb216)

Source inspection established that pre-exec pane-shell stderr and an effective exported-environment profile are not persisted; scrollback raw contains inner-agent PTY bytes only. TL6e0eb216-ce18-4777-9e56-8a739becb6cc confirms the minimal source contour: enable owned bounded PAIR_WRAP_LOG inside the fixture root; record allowlisted actual scope/tag/session/nonce and startup/readiness errors through existing wrapper diagnostics; query the genuine private pane with Zellij dump-screen before teardown; explicitly report unavailable if product rollback already removed it; capture readiness sibling census. Pre-exec shell stderr is explicitly out of scope. No new pane wrapper, probe, process, fake PTY, topology, startup deadline or retry is introduced. Diagnostic capture starts after admitted ActorCLI joins but before supervisor/agent teardown, then the existing final failure projection follows joined cleanup. All file sources are factory-private, physically confined and bounded; reject symlinks/escapes, cap file counts and retained bytes, and retain missing/read/query errors rather than reconstructing values. Extend stateful filesystem/native-query seam tests and value-only allowlist/bounds tests using actual emitted formats, not fictitious profiles. After focused checks and committed clean implementation, one native matrix per TL sequencing; #772 remains exact Codex flake, fixed-budget issue773 remains independent and unclaimed.

### 2026-10-21 — diagnostic implementation concepts and growth bounds

| PURE entity | Lives in | Status |
|---|---|---|
| startupIdentity value and selectStartupRecords | cmd/internal/testcouch/startup_evidence.go | new; allowlisted byte-to-value projection |
| selectReadyNames and selectStartupSources | cmd/internal/testcouch/startup_evidence.go | new; bounded observed-fact selection, one visit per input |
| startupDiagnosticDeadline | cmd/internal/testcouch/startup_evidence.go | new; quarter of caller remaining time, no renewed budget |
| IsAgentReadyCandidate | cmd/internal/artifactpath/paths.go | new; diagnostic filename recognition only, no authority |

| INTEGRATION | Lives in | Status | Wraps/reuses |
|---|---|---|---|
| wrapper startup identity/error publication and close | cmd/internal/wrapcmd/startup_diagnostics.go, wrap.go | new | actual allowlisted environment, existing bounded proxy diagnostic writer |
| evidenceRoot, captureStartupLog and captureStartupEvidence | cmd/internal/testcouch/startup_evidence.go | new | rooted nonblocking regular-file reads, canonical artifact/ready decoders and lifecycle store |
| dumpStartupPane and bounded startupOutput stream adapter | cmd/internal/testcouch/startup_evidence.go | new | existing native query runner, explicit private sockets/environment, one derived diagnostic context |

Per startup, four allowlisted environment values are selected once; no per-byte work is added. Capture visits each bounded log line/identity once, holds at most five ranking candidates, retains four sources, scans at most129 directory entries and retains32 names. Log reads64KiB, per-source/native output16KiB; truncation/unavailable are explicit. N/2N tests count visits8/16 for log rows, names and identity selection. Stateful executable tests qualify argv, actual private socket env, native refusal/output cap and no ambient secret inheritance. Core state remains the existing OwnedRun and correlated mailbox; diagnostic values grant no lifecycle or cleanup authority.

### 2026-10-21 — final approved source contour verified

Final clean code 5ecbef821f2d64af2564de889bd6e781277ee7a6 passes the authorized real Claude/Codex three-worker matrix93.20s, with all cohort/delivery/restart/cleanup checks and no native retry on that head. The Log/evidence retain both independent product failures (772,773), all prior successful qualification and747 cross-reference. Required final recorded suite and full close still precede landing.

### 2026-10-21 — FIX-THEN-SHIP canonical SlotLayout correction

Round4 review confirms BR-2 and BR-5 addressed and independently passes both native personalities154.44s with exact cleanup. It returns FIX-THEN-SHIP only for provisioning's three duplicated layout spellings (BR-6). Existing TestEverySlotStateSiteIsAResource reproduced RED at the worker container, main-slot resting branch and setup-marker filename. All provisioning observations now derive through NewSlotLayout(primary, common,n).Host(), RestingBranch() and SetupMarkerPath(admin); no audit exemption or alternative path rule. Targeted audit, real-Git provisioning, failure/cancellation/adoption and slot-layout race tests PASS couchcore17.804s/testcouch4.771s. No new state or scope, and fixture topology expectations remain the same. The queued pre-fix suite lease4288014647 was canceled via SDLC before execution; no result claimed. Checkpoint fix, rerun native on final code, then finalize this existing close receipt with issue recovery reconcile and run the required final-head recorded suite. No redundant close review is requested under FIX-THEN-SHIP protocol.
