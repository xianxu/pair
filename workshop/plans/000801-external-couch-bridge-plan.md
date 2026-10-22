---
collaboration-mode: discussion
---

# External Couch conformance bridge implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach. Steps use checkbox syntax for tracking.

**Goal:** Let Parley's existing conformance driver use one genuine private Couch producer without duplicating Pair's launcher or cleanup authority.

**Architecture:** Extract error-returning construction/build seams from the existing testcouch owner; keep its testing.TB APIs as wrappers over those seams. A thin executable drives that same owner, emits readiness after a real editor-endpoint roundtrip, then holds until EOF, TERM, failure or its deadline. One pure transition model owns admission to readiness and the single joined stop.

**Tech stack:** Go, existing testcouch/procutil/session-sync/editor-endpoint code, genuine private Zellij and deterministic agent fixtures; native Neovim is a consumer of the emitted environment.

## Approval and alternatives

Full flow, one atomic close, no milestone tags. No code until the hub approves this design and parley.nvim:1 confirms the executable contract below. No live-fleet or pair:0 use.

Recommended: an error-returning shared testcouch entry point and a standalone binary. A testing.TB shim cannot implement testing.TB's private methods. Driving an actual Go test binary would retain TB but mixes test-runner output, exit and signal ownership into the external protocol. A separate Python/shell Couch launcher would duplicate the very ownership and teardown that #801 must centralize (ARCH-DRY).

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| Options | cmd/internal/testcouchbridge/options.go | new |
| Readiness / Document | cmd/internal/testcouchbridge/readiness.go | new |
| Phase / EventKind / Effect / State / Event / Reduce | cmd/internal/testcouchbridge/model.go | new |

- Options validates argv, explicit absolute candidate/scratch paths and fixed resource limits before effects. One invocation owns one fleet. The supplied Couch overrides only the Couch candidate; companion Pair/helper and fixture binaries come from this checkout's shared candidate builder, never an installed Pair. Candidate compatibility is exercised by startup, not guessed from a version string.
- Readiness contains Version, Namespace, Socket, Environment and a proposed Documents array of two `{ID,File}` entries. The array gives Parley registered private fixtures without discovering live documents. Namespace/socket/document paths are physical absolute paths, document IDs come from the actual registry. Environment replaces the entire client environment, never overlays ambient HOME/COUCH variables. These values are invocation-local data, not a durable authority file.
- Reduce is the only writer of bridge phase. It owns whether a readiness publication may begin and whether a stop is already committed; Fleet's existing lifetime gate remains the resource authority for all process effects. No second ownership registry.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Candidate owner/build API | cmd/internal/testcouch/binaries.go and new owner_build.go | modified | existing candidate builds and owned binary cleanup |
| Fleet construction/reporting | cmd/internal/testcouch/fleet.go and new owner.go | modified | existing roots, runtime installation, process owner and cleanup |
| RunBridge / Owner | cmd/internal/testcouchbridge/run.go | new | shared owner plus input, signal, clock and bounded output adapters |
| BridgeIO | cmd/internal/testcouchbridge/run.go | new | cancellable publication/control interface |
| NativeOwner | cmd/internal/testcouchbridge/native.go | new | shared fleet/candidate owner and public Couch CLI |
| PipeIO | cmd/internal/testcouchbridge/pipes_unix.go | new | exclusive cancellable stdin/stdout pipes |
| Diagnostics / DiagnosticPipe | cmd/internal/testcouchbridge/diagnostics.go and diagnostic_pipe_unix.go | new | bounded diagnostic sink and nonblocking stderr adapter |
| Main | cmd/couch-conformance-bridge/main.go | new | argument/signal wiring and exit status only |

Existing Build(t)/New(t) remain source-compatible convenience wrappers. They use error-returning implementations and register the same cleanup. Replace Fleet's dependency on TB for diagnostics with explicit reporting; the test wrapper supplies Logf/Failed and the command supplies a bounded stderr sink/failure signal. Constructor failures return any partially created owner needed for cleanup; they never hide live resources behind a nil result. Preserve the existing exact-generation census, creation revocation, retained descendant handling, and joined direct-child/IO cleanup. Build artifacts are a subordinate resource of the invocation: remove them only after fleet cleanup succeeds; retain them with the evidence if cleanup is uncertain.

Stateful fake: a fake owner behind the same runner interface tracks allocated roots, startup admission, endpoint observations, active callers, readiness publication and joined cleanup. It can release each boundary causally, fail publication or return uncertain cleanup. Real-code conformance runs the public executable and native service with the same protocol.

## External executable contract — proposed for consumer confirmation

Build the thin executable once with `make couch-conformance-bridge` (the target uses the existing atomic Go publisher). Invoke the absolute resulting binary:

```
/ABS/pair/bin/couch-conformance-bridge --couch /ABS/candidate/couch --scratch-parent /ABS/private-scratch-parent
```

Both paths must be existing physical absolute paths of the expected kind; reject invalid/overlapping roots and an overlong native socket placement before starting anything. Do not silently relocate an explicitly chosen scratch parent. Omit --couch to build Couch with the other candidates from this checkout. Candidate preparation uses the existing builder and installed read-only native tools; runtime processes receive only the explicit isolated environment and fixture agents. No network or agent account/credential access. The command does not add a production Couch mode or install any binary globally.

One newline-terminated JSON object on stdout, once:

```
{"Version":1,"Namespace":"/physical/private/store","Socket":"/physical/private/m/editor.sock","Environment":{"HOME":"/physical/private/home","PATH":"/physical/private/bin"},"Documents":[{"ID":"<registry-id>","File":"/physical/private/artifacts/first.md"},{"ID":"<registry-id>","File":"/physical/private/artifacts/second.md"}]}
```

The sample Environment is abbreviated only here; the actual map is complete. Candidate couch and selected native nvim are on its private PATH. Socket is derived using the same explicit isolated selection as the producer, never the driver's ambient live COUCH environment. No token, actor mailbox credential or ambient environment dump is included. Unknown additional response metadata may be ignored by the consumer; known keys/types and Version remain strict, duplicate known keys invalid. No persistent schema changes.

Create two small scratch Markdown documents, register them through the public session-sync register path under the private namespace and fixture steward, then perform a genuine `couch --editor-endpoint lookup --namespace NS --document-id ID --json` in the complete environment. Require exit0, Version1, Code=ok, empty Error and confirmed empty binding before publishing readiness. Socket existence or an unavailable owner is not readiness. Client is responsible for its own native nvim/probe lifetime and endpoint registration/unregistration; close clients before EOF. The bridge never owns or guesses the producer parent's descendants.

### Bounds and their basis

| Resource | Bound | Basis and exceeded behavior |
|----------|-------|-----------------------------|
| Startup through complete readiness write | 300s from driver entry | Proposed envelope, allows existing180s candidate preparation plus native setup; one absolute deadline, not renewed by phases. Abort startup and enter joined stop. |
| Total active invocation | 600s from driver entry | Proposed conformance envelope; includes startup/hold, excludes final15s cleanup. Expiry fails the run and stops once. |
| Joined stop | 15s from first stop request | Existing shared test cleanup budget. Never renew on repeated EOF/TERM. Unknown cleanup retains root and exact evidence, nonzero exit. |
| Readiness stdout | one line, at most64KiB including newline | Reject before publication if encoding exceeds bound. Partial/lost write is startup failure, never a second line or automatic retry. |
| Diagnostic output |64KiB aggregate retained/emitted plus one truncation marker | Existing builder/terminal diagnostic scale; continue draining child output after cap without unbounded storage. Terminal failure/root identity has a reserved bounded final diagnostic. |
| Input | EOF is control; nonempty input refused | No commands, queue or interactive shell on stdin. TERM/INT use the same stop. |
| Documents | two fixtures, each at most64KiB at creation | Enough for one active and one replacement/rejection case; all are under owned root. No access to operator files. |
| Candidate files | existing512MiB per runtime executable | Reuse current validation, no new arbitrary executable reader. |
| Concurrency | one fleet, one provisioned but unstarted worker, one fixture steward | Reuse New's existing1-worker minimum; do not widen the harness to zero workers or launch unnecessary workers. No polling while held. |

Parent should allow startup300s plus cleanup15s, total lifetime600s plus cleanup15s, and a small process-observation margin; it owns its process group and external wall-clock deadline. Driver self-bounds are correctness limits, not proof against SIGKILL/OS death. SIGKILL cannot promise teardown: retain exact root/run evidence and require identity-aware Pair recovery, never `pkill`, guessed PIDs, a second launcher or recursive removal before proven absence. If the parent kills its whole process group, shared ownership reconciliation must still prove daemonized generations absent.

Owned stdin/stdout adapters must be cancellable and joined, including an unread/full stdout pipe. Do not implement a timeout by abandoning a goroutine blocked in Write/Read. Stop revokes publication, closes the owned pipe adapter to unblock it, and joins it before cleanup can report success. A readiness-write completion that arrives after Stop cannot move the model back to Held. A partial line is not readiness. Tests force this ordering.

## States, ordering and visible recovery

Actions: run a fresh invocation; close client then EOF/TERM; inspect retained exact evidence. No automatic retry of a producer or uncertain write.

| Internal state / failure | Visible state | One recovery action | Possible loss |
|--------------------------|---------------|---------------------|---------------|
| Preparing/Starting/Checking | no readiness; bounded stderr progress | wait within startup bound | no client work yet |
| Publishing | readiness still unconfirmed until full line | wait within startup bound | partial JSON must be discarded |
| Held | one readiness line; real service available | close client then EOF/TERM | private fixture edits disappear on clean end |
| Startup/producer failure, cancellation or deadline | failure diagnostic; stopping | wait for bounded joined stop | incomplete client/fixture work |
| Stopped and cleanup proven | exit0 for requested EOF/TERM, otherwise nonzero with original failure | run fresh invocation | scratch intentionally removed |
| Cleanup uncertain/deadline/owner crash | nonzero or missing exit; retained exact root/run | inspect retained exact evidence | surviving resources possible; no automatic deletion |

Transition enumeration: Begin→Preparing; candidates→Starting; native registration→Checking; confirmed endpoint+documents→Publishing; complete write→Held. EOF/TERM/INT from any nonterminal phase→Stopping exactly once, revoking startup/publication and cancelling/joining admitted work. Startup failure/producer death/expiry/output error→Stopping with a retained failure cause. Late stage/readiness completions while Stopping are ignored. Successful Close+AssertCleaned→Stopped; any uncertain cleanup→Retained, never Stopped. Repeated stop joins the same result; no budget renewal or overlapping effects. Order tests cover EOF before allocation, after allocation and during readiness; TERM races with EOF; supervisor death while held; late readiness; partial output; cleanup failure.

## Operating cost and residue

Startup is offline qualification work, not a UI hot path. For N selected environment entries or diagnostics bytes, validate/encode once, O(N); count4k/8k input-byte visits and assert linear growth and the fixed output cap. Exactly one endpoint readiness roundtrip after native setup, no timer-driven lookups while held. Count startup service calls and zero calls over many held-state events. Process scans remain the existing owner's bounded startup/cleanup scans; no second polling census. CPU/disk are dominated by existing candidate compilation and private provisioning; no timing performance claim. Native-tool versions and measured startup/cleanup durations are retained with acceptance.

New residue is one invocation root plus candidate directory, created by its shared owner, last needed by client/cleanup, removed after exact absence and joined IO. Uncertain endings keep bounded diagnostics and existing ownership evidence with explicit root identity; no automatic age-based deletion of possibly live state. No new durable fleet registry, server queue, replay store, global cache or background reaper. Runtime copies retain the existing per-file512MiB cap; normal small-fixture disk use is measured in native acceptance, not invented as an enforced total quota (ARCH-FUNERAL/CONSTRAINTS).

## Implementation tasks and verification

### Task1 — share the existing owner without TB coupling

Files: modify testcouch/binaries.go, fleet.go, cleanup.go; add owner.go/owner_build.go and colocated tests.

- [x] Write red tests calling the error-returning API: caller-selected scratch, explicit Couch override, partial constructor/build failure, cancellation and retained cleanup failure. Prove no installed Pair fallback and no live-root overlap effects.
- [x] Run `go test ./cmd/internal/testcouch -run 'Owner|Candidate|Roots|Environment|Runtime|Cleanup' -count=1`; require the new APIs absent/red first.
- [x] Extract shared implementations; keep testing wrappers and every existing ownership/cleanup control. Native lifetime still belongs to Fleet, not a second executor.
- [x] Re-run focused tests plus the existing complete testcouch package; commit the isolated refactor.

### Task2 — pure protocol/lifecycle plus bounded IO runner

Files: create testcouchbridge/options.go, readiness.go, model.go, run.go, native.go, pipes_unix.go and colocated tests/stateful fake (actual adapter split reconciled in the close BR-3 Revision).

- [x] Write red table/sequence tests for the states, invalid paths, duplicate environment keys, no ambient overlay, malformed/oversized endpoint responses, output cap, early/repeated stop, failed/late readiness, deadline, producer death, failed cleanup and unread stdout.
- [x] Run `go test ./cmd/internal/testcouchbridge -count=1`; demonstrate missing protocol/transition behavior first.
- [x] Implement one reducer-owned runner using shared Fleet owner. Register the private documents and qualify the actual CLI response. No readiness from a listener stat. Bind socket derivation to the complete environment without process-global environment mutation.
- [x] Test fake N/2N byte work and service calls, joined cancellable pipe behavior, no effects after Stop, and root retention on uncertainty. Run focused race tests; commit.

### Task3 — executable, native consumer and external handoff

Files: create cmd/couch-conformance-bridge/main.go and main_test.go; modify Makefile.local; add native bridge conformance in cmd/couch or the bridge command; update README.md and atlas/couch.md/index.md.

- [x] Add argv/exit tests using fake tools, then the thin main and atomic build target. No global installation or new Couch production switch.
- [x] Run normal package tests and `make test-affected-inventory`; add exact reader/artifact classifications where applicable, never broad exemptions. New fixture-owning reader certificates must name non-Go input fallback honestly.
- [x] Bounded opt-in native acceptance: actual public executable, shared fleet, real service roundtrip, complete env with native nvim, two registered scratch documents; EOF and TERM end through the owner and prove exact cleanup. Exercise both deterministic Claude and Codex fixture personalities internally without requiring a consumer flag. Test selected explicit candidate and default candidate construction. Retain exact source/tool versions, durations, outcome and cleanup evidence.
- [x] Send final contract/evidence to parley.nvim:1; coordinate its actual Lua driver invocation without modifying Parley's tree or duplicating teardown. Real-producer acceptance is not claimed from fakes or socket existence.
Workflow checkpoint (gate-owned, not an implementation checkbox): push, obtain recorded affected/full verification, close through the binary-owned review, then `sdlc merge --yes` on SHIP. Hub owns project and acceptance decisions.

## Architecture disposition

ARCH-DRY: one existing fleet owner/build pipeline, test and executable adapters only. ARCH-PURE/ORDER: pure options/envelope/reducer, bounded IO results return as events. ARCH-PURPOSE: real producer admission and Parley handoff are required, not documentation-only. ARCH-MOCK: stateful fake uses the runner seam; actual endpoint/native nvim qualify it. ARCH-CONSTRAINTS: limits above are explicit proposals for hub/consumer approval. ARCH-SECURE: argv and endpoint replies are typed/bounded, environment starts empty, roots are canonical and denied-live overlap checked before effects; no native agent credentials or network. ARCH-FUNERAL: exact joined owner end, preserve uncertainty. ARCH-EVOLVE: no persisted schema mutation; readiness Version1 allows additional metadata while preserving strict known semantics.


## Revisions

### 2026-10-22 — consumer-confirmed parent/side-doc fixtures and native selection

Hub0cdb7c57 approved the design subject to consumer confirmation. Canonical parley.nvim:1 receiptd0f4eb15 confirms argv, readiness fields, Documents, real roundtrip, startup300s/lifetime600s/stop15s and64KiB bounds, with a630s parent observation window. It requires Documents in this order: first is artifacts/first.md (registered parent); second is artifacts/first.side/second.md, frontmatter type: side-doc and parent: ../first.md. Both have distinct actual registry IDs in the advertised Namespace. This supersedes the earlier illustrative second-file path; no new field, verb or fixture owner slot. Native nvim stays selected once from the invocation's PATH through existing resolveTool so the consumer can qualify0.11.7 and0.12.5 in separate runs, then uses that exact tool through each private PATH. Add deterministic fixture-layout/namespace/distinct-ID controls and qualify both available native versions in consumer acceptance. No default-nvim substitution after selection. Consumer owns and closes its native editors before EOF and retains evidence on nonzero/uncertain teardown; it adds no producer teardown. Hub requested folding any consumer gap before change-code, so this confirmation fulfills that condition.


### 2026-10-22 — PQ-1 candidate preparation isolation and provenance

This makes the build boundary explicit, superseding the earlier shorthand “existing builder.” Current facts: testcouch.Build calls testpair.AtContext (binaries.go:94); testpair.BinaryContext admits ambient PAIR_TEST_BINARY, and place appends os.Environ before invoking the sole Pair builder script (testpair/binary.go:43,67,83). testcouch.buildCommand uses OSProvisionIO without Env (binaries.go:184; couchcore/provision_io.go:67). Runtime environment isolation alone therefore does not isolate preparation.

Extend Task1 to cmd/internal/testpair/binary.go and new candidate.go plus colocated tests. Introduce error-returning testpair.PlaceCandidate(ctx, Placement, Runner), sharing the existing tests/lib/pair-test-binary.sh placement/compiler path. Placement carries a verified physical Checkout, destination, explicit source selection and complete environment; Runner owns bounded foreground execution/join. Existing AtContext/place remain legacy-policy wrappers, including their explicit shared test image, while the new strict testcouch path always supplies empty PAIR_TEST_BINARY and its own environment. Do not import couchcore into testpair (same-package tests would cycle); testcouch supplies the existing OSProvisionIO adapter. No second direct Pair compiler or copied placement script.

BuildOwned(ctx, BuildConfig) returns a candidate owner and error, including a partial owner when cleanup remains owed. Shared Build(t) wraps this strict path; other legacy testpair consumers retain their existing policy. Standalone source is reporoot.Find from the physical executable's containing directory (the supported bin entry under the Pair checkout), not cwd or ambient PAIR_HOME. Relocated standalone binaries outside a verified Pair checkout refuse. Test wrappers may explicitly select their verified current checkout. Supplied --couch is copied as that exact validated executable, with no installed-Couch fallback; Pair/helper/fixture companions are built only from the verified checkout. No caller PAIR_TEST_BINARY can override this choice. Capture actual source head and selected executable paths in bounded diagnostics without inventing compatibility from a version string.

Resolve native tools once from invocation PATH, including nvim, Go and the small build-tool set; pass absolute executables or a PATH containing only those selected tools/system compiler utilities. Build environment is assembled from scratch: private HOME/XDG/config/temp under the invocation's allocated preparation root; explicit GOENV=off, GOWORK=off, GOTOOLCHAIN=local, GOPROXY=off, GOSUMDB=off, GOFLAGS=-mod=readonly. No inherited GOFLAGS, tool-exec wrapper, agent variables, credential helper, proxy or application configuration. Resolve only typed Go cache locations with a bounded go-env metadata query with GOENV=off; the invoking HOME may be used solely to calculate normal cache paths, not as a child build HOME. Explicit GOCACHE/GOMODCACHE paths are validated and passed deliberately: shared Go build/module caches are the sole allowed non-owned build storage, not Couch/agent roots. Normal Go build cache writes are permitted; no settings changes, private per-run cache or module/toolchain download. Missing cached modules, SDK/toolchain or invalid cache selection fails offline and enters cleanup, never falls back to network or global config. Native provisioning commands likewise use the complete private runtime environment through their existing explicit runner.

Build work uses the same absolute startup deadline, its existing180s sub-bound capped by remaining startup time, bounded/drained output and joined process group cancellation. Every candidate/preparation path is allocated by the invocation owner before the command; partial failure retains its owner for joined cleanup. The source checkout and supplied executable are read-only inputs; no outputs in their trees except the explicitly requested make target before invocation. No credential/auth files are opened. This is a refinement of the approved isolation contract, not a new configuration resolver or build cache.

### 2026-10-22 — PQ-2 canonical function-level test strategies

The following replaces the illustrative prose test inventories in Tasks1–2 as the executable testing contract; those earlier examples remain only as design history. Each risky function has one adversarial strategy and mechanical guard. Test names/cases will be written in code, not duplicated in this plan.

| Function | Strategy and guard |
|----------|--------------------|
| ValidateBridgeOptions | Bounded fuzzing over malformed argv/path facts; reject before allocation or runner invocation. |
| BuildEnvironment | Property tests over hostile ambient variables and duplicate keys; output equals the complete allowlisted environment and offline settings, with N/2N visit counts. |
| ResolveBuildInputs | Stateful fake metadata/tool resolver; changing cwd, executable location and cache/config facts cannot select another checkout, installed Pair or native nvim after selection. |
| testpair.PlaceCandidate | Real temp checkout plus recording runner; prove exact verified script/source/destination/env are passed, ambient PAIR_TEST_BINARY cannot affect strict caller, cancellation joins. |
| BuildOwned | Stateful runner with causal failure boundaries; partial owner remains recoverable, all tools use explicit config, missing dependencies fail offline without fallback. |
| NewOwned / Fleet.Close / Fleet.AssertCleaned | Existing ownership model plus generated cancellation/failure sequences; no foreign effect, no removal before exact absence and joined IO. |
| ValidateReadiness / EncodeReadiness | Bounded fuzzing of typed external responses and environment/path/document shapes; enforce canonical namespace, distinct parent/side-doc IDs, one line and byte cap without partial prevalidation output. |
| ReduceBridge | Generated event sequences with independent invariants: at most one readiness, Stop dominates late completions, one stop budget, never clean on uncertain evidence. |
| RegisterBridgeDocuments / CheckEndpoint | Stateful CLI fake plus real service conformance; success requires actual registered parent/side-doc observations and an ok service reply, not socket existence or fabricated IDs. |
| RunBridge / owned pipe Read and Write | Causally gated fake owner/pipes and virtual deadlines; interrupted/blocking IO must be revoked and joined, never abandoned or retried into authority. |
| Main | Subprocess argv/env/signal tests using the shared runner seam; exit0 requires requested stop plus Close and AssertCleaned, and stdout contains only the supported readiness protocol. |

Tasks retain their red→minimal implementation→green→commit ordering and same file boundaries, plus the explicit testpair extraction above. Native acceptance remains the real #405 driver and both selected nvim versions, independently invoked. No behavioral code has been written.


### 2026-10-22 — constructor admission and remaining-root evidence

The ad-hoc lifetime audit exposed two implementation gaps in the approved stop contract. Keep the existing separate admitted lifetime, but carry startup admission independently through OwnerConfig.AdmissionContext (nil defaults to the lifetime context). Check admission after candidate preparation and between constructor/installation effects; cancellation returns any partial owner for the same joined cleanup. This adds no executor, persistent state or visible recovery state. After fleet teardown is confirmed, recovery evidence selects a still-owned candidate directory rather than the removed fleet root. Add gated-constructor, post-build cancellation and cleanup-budget-exhaustion controls before another native run. Start failures also print the existing cached console frame through the bounded diagnostic sink, with no extra native query. Hub97f22b2d authorizes one subsequent bounded native run; no timeout extension or repeated startup probes.


### 2026-10-22 — scratch parent clarification from real consumer setup

Consumer7c555269 created an additional private cb-* directory under the selected temporary parent, which exceeded the conservative wrapper-socket limit and correctly refused with joined cleanup. The supported ScratchParent is an existing physical parent, not an extra consumer-created producer root: pass /private/tmp directly (native qualification used /Volumes/TestScratch directly). The producer atomically creates its own fresh0700 pc-* and couch-candidate-* children, then removes only those exact owned children through its owners. No caller rmdir of the shared parent and no silent relocation. Consumer/editor scratch remains separately owned. This clarifies the existing allocation contract; no implementation, bound, refusal or protocol change.


### 2026-10-22 — consumer qualification and gate-owned completion

Consumer738380b8 qualifies the actual existing Parley driver on native nvim0.12.5 and0.11.7, both joined EOF cleanup, using the same supported bridge. Evidence is retained in the issue. Implementation checkboxes are complete. The original final checkbox combined future recorded verification, close and landing with implementation; retain it as a gate-owned workflow instruction instead of falsely ticking a landing before it occurs. Required checks/review/SHIP/merge are unchanged. Current recorded suite lease-869401233 is queued on48162a9b; binary-owned review may overlap it per tl-worker, while landing still requires the exact-code recorded pass. No runtime change after eee3547f.


### 2026-10-22 — close BR-3: reconcile implemented concept map

The core-concept tables now name actual exported symbols and source files, replacing the illustrative BridgeOptions/BridgeState/BridgeEvent/ReduceBridge/BridgeRunner prefixes with Options/State/Event/Reduce/RunBridge. BridgeIO lives in run.go, not the unimplemented io.go; NativeOwner, PipeIO and Diagnostics/DiagnosticPipe explicitly map the native service, owned pipes and bounded diagnostic adapters. This is a factual traceability correction, with no behavior, ownership or scope change.

The earlier PQ-2 strategy labels map to implemented symbols as follows: BuildEnvironment → testcouch.buildEnvironment; ResolveBuildInputs → tool/cache selection in testcouch/build_environment.go and owner_build.go, plus nativeSource in testcouchbridge/native.go; ReduceBridge → Reduce; RegisterBridgeDocuments/CheckEndpoint → NativeOwner.Readiness; owned pipe Read/Write → PipeIO.WaitStop/Publish. These labels described test responsibilities, not additional exported APIs.


### 2026-10-22 — close BR-4: preserve testing-wrapper selection and ownership

Compatibility rule: existing testing.TB wrappers preserve the legacy default selection and lifetime policies; standalone owners receive explicit isolated inputs and never consult ambient PAIR_TEST_BINARY. Build must resolve Pair through the existing testpair.BinaryContext policy (suite-supplied executable, otherwise its process-owned once-built fallback), then place that exact candidate in the caller-owned preparation directory through the shared placement seam. Closing that directory must not delete the supplied or process-owned source. New preserves its existing default personality/budget, physical scratch selection and socket fallback, independent runtime lifetime and test-owned cleanup. NewOwned retains explicit admission/lifetime and no ambient candidate resolution. No second candidate cache or owner is added.

Affected direct Build consumers, enumerated from source: TestDisposableCouchFleet in cmd/couch/fleet_acceptance_test.go; TestDisposableCouchStartupBudget in cmd/couch/startup_budget_acceptance_test.go; TestLiveProvenOrphanReap in cmd/internal/couchcmd/reap_proven_live_test.go. The correction must prove supplied-candidate reuse, unset-candidate fallback, copied-versus-source cleanup lifetime and standalone ambient isolation with causal regression controls, retaining the already passing default-temp and strict-explicit-path controls. This restores the original API contract without broadening the bridge protocol or selection policy.


### 2026-10-22 — compatibility re-plan: third delta, awaiting hub

Read-only sweep found the extracted PlaceCandidate gate also tightened testpair.At/AtContext: the original script accepted relative destination names after selecting the source checkout, whereas PlaceCandidate requires absolute clean destinations. Together with default-temp aliases and Pair candidate selection, this is the third delta on the extraction compatibility boundary. Stop incremental shims under tl-worker’s third-delta rule and obtain hub approval for the complete contract before more implementation.

Consolidated design: strict BuildOwned/NewOwned/PlaceCandidate remain explicit and isolated; existing testing wrappers adapt their legacy inputs exactly once at their boundaries. Build retains one180s caller deadline, testpair.BinaryContext’s suite-selected or process-owned cached fallback, and caller-owned placement cleanup. New retains its previous defaults, independent runtime and test cleanup. testpair.place translates the legacy relative destination to an absolute clean path in its already-selected source checkout before entering PlaceCandidate; supplied candidate and source lifetime remain unchanged. No ambient resolver moves into standalone APIs. Existing canonical default selection remains shared.

Simplest alternative considered: restore the complete original Build implementation unchanged alongside BuildOwned. That prevents some selection drift but duplicates candidate build/cleanup logic and leaves the relative-placement regression unresolved; prefer thin legacy input adapters around one shared implementation (ARCH-DRY). Required matrix covers supplied/unset candidate selection, source-versus-placement lifetime, relative/absolute destination equivalence, invalid explicit standalone paths, ambient standalone isolation, default temp aliases and New’s existing lifetime/default controls. Enumerate consumers before changing behavior; no global test hooks or additional candidate caches. This is pending hub review, not authorization to code another shim.

The selected-candidate branch of the existing placement script invokes Python3; the legacy wrapper’s private build-tools PATH must retain that explicit read-only tool selection. Both affected consumer packages already use testpair.Run, so process-owned fallback cleanup remains there. Cross-volume symlink placement versus runtime Lstat refusal predates #801 and is excluded from this re-plan.
