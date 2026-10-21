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

### 2026-10-21 07:21 PDT — BR-2 approved completion table and implementation checkpoint

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
