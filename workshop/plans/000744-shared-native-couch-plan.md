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
| AgentState / ReduceAgent and Claude/Codex paint adapters | cmd/internal/testcouch/agentfixture/model.go | new |
| AgentCommand / AgentResult validation and request correlation | cmd/internal/testcouch/agentfixture/protocol.go | new |

RootsPlan names every mutable root and the explicit read-only tool/source inputs. OwnedRun owns 1:N children, native sessions and IO pumps; it accepts observations about exact generations, never naked PIDs. AgentState is one deterministic fake agent with two supported personalities, not a new production agent kind. Actor requests identify a fixture actor and request ID, never a fabricated Couch binding. ARCH-PURE/DRY: one owner for each fact, and no shadow terminal or broker protocol.

| Integration | Lives in | Status | Wraps/reuses |
|---|---|---|---|
| Candidate tool placement | cmd/internal/testcouch/binaries.go | new | testpair.Binary/At, testbinary.Cache pattern, go build, explicit native executable discovery |
| Root lease and empty environment | cmd/internal/testcouch/environment.go | new | testpair.Environment/EnvironmentFrom, production isolation contract |
| Git/provisioning | cmd/internal/testcouch/provision.go | new | actual Git, Couch provision-workspace, installed SDLC/Weave |
| Fleet / ActorCLI / frame and input control | cmd/internal/testcouch/fleet.go, actor.go | new | real public CLI, PTY, existing terminal emulator, private fixture control |
| Native session/process owner | cmd/internal/testcouch/process.go | new | procutil generation checks, existing native terminal configuration and shutdown patterns |
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
- [ ] Compose native Fleet/ActorCLI/frames/restart/Close and genuine worker startup. Qualify actual registration, concurrent actor commands and exact teardown through the first opt-in consumer in cmd/couch/fleet_acceptance_test.go; do not add a private fleet beside the shared owner.
- [ ] Add `make test-couch-fleet` in Makefile.local with `PAIR_NATIVE_COUCH_FLEET=1` and an explicit overall timeout; no .sdlc declaration. Record current source head, native versions, isolation manifest, elapsed setup/runtime/cleanup, exact receipt/body evidence and full captured log/hash. Preserve failures; classify any repeated same-head flake through the existing SDLC rule.
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
