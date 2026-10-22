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
| BridgeOptions | cmd/internal/testcouchbridge/options.go | new |
| Readiness | cmd/internal/testcouchbridge/readiness.go | new |
| BridgeState / BridgeEvent / ReduceBridge | cmd/internal/testcouchbridge/model.go | new |

- BridgeOptions validates argv, explicit absolute candidate/scratch paths and fixed resource limits before effects. One invocation owns one fleet. The supplied Couch overrides only the Couch candidate; companion Pair/helper and fixture binaries come from this checkout's shared candidate builder, never an installed Pair. Candidate compatibility is exercised by startup, not guessed from a version string.
- Readiness contains Version, Namespace, Socket, Environment and a proposed Documents array of two `{ID,File}` entries. The array gives Parley registered private fixtures without discovering live documents. Namespace/socket/document paths are physical absolute paths, document IDs come from the actual registry. Environment replaces the entire client environment, never overlays ambient HOME/COUCH variables. These values are invocation-local data, not a durable authority file.
- ReduceBridge is the only writer of bridge phase. It owns whether a readiness publication may begin and whether a stop is already committed; Fleet's existing lifetime gate remains the resource authority for all process effects. No second ownership registry.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Candidate owner/build API | cmd/internal/testcouch/binaries.go and new owner_build.go | modified | existing candidate builds and owned binary cleanup |
| Fleet construction/reporting | cmd/internal/testcouch/fleet.go and new owner.go | modified | existing roots, runtime installation, process owner and cleanup |
| BridgeRunner | cmd/internal/testcouchbridge/run.go | new | shared owner plus input, signal, clock and bounded output adapters |
| BridgeIO | cmd/internal/testcouchbridge/io.go | new | complete isolated argv/environment, public Couch CLI and owned pipes |
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

- [ ] Write red tests calling the error-returning API: caller-selected scratch, explicit Couch override, partial constructor/build failure, cancellation and retained cleanup failure. Prove no installed Pair fallback and no live-root overlap effects.
- [ ] Run `go test ./cmd/internal/testcouch -run 'Owner|Candidate|Roots|Environment|Runtime|Cleanup' -count=1`; require the new APIs absent/red first.
- [ ] Extract shared implementations; keep testing wrappers and every existing ownership/cleanup control. Native lifetime still belongs to Fleet, not a second executor.
- [ ] Re-run focused tests plus the existing complete testcouch package; commit the isolated refactor.

### Task2 — pure protocol/lifecycle plus bounded IO runner

Files: create testcouchbridge/options.go, readiness.go, model.go, run.go, io.go and colocated tests/stateful fake.

- [ ] Write red table/sequence tests for the states, invalid paths, duplicate environment keys, no ambient overlay, malformed/oversized endpoint responses, output cap, early/repeated stop, failed/late readiness, deadline, producer death, failed cleanup and unread stdout.
- [ ] Run `go test ./cmd/internal/testcouchbridge -count=1`; demonstrate missing protocol/transition behavior first.
- [ ] Implement one reducer-owned runner using shared Fleet owner. Register the private documents and qualify the actual CLI response. No readiness from a listener stat. Bind socket derivation to the complete environment without process-global environment mutation.
- [ ] Test fake N/2N byte work and service calls, joined cancellable pipe behavior, no effects after Stop, and root retention on uncertainty. Run focused race tests; commit.

### Task3 — executable, native consumer and external handoff

Files: create cmd/couch-conformance-bridge/main.go and main_test.go; modify Makefile.local; add native bridge conformance in cmd/couch or the bridge command; update README.md and atlas/couch.md/index.md.

- [ ] Add argv/exit tests using fake tools, then the thin main and atomic build target. No global installation or new Couch production switch.
- [ ] Run normal package tests and `make test-affected-inventory`; add exact reader/artifact classifications where applicable, never broad exemptions. New fixture-owning reader certificates must name non-Go input fallback honestly.
- [ ] Bounded opt-in native acceptance: actual public executable, shared fleet, real service roundtrip, complete env with native nvim, two registered scratch documents; EOF and TERM end through the owner and prove exact cleanup. Exercise both deterministic Claude and Codex fixture personalities internally without requiring a consumer flag. Test selected explicit candidate and default candidate construction. Retain exact source/tool versions, durations, outcome and cleanup evidence.
- [ ] Send final contract/evidence to parley.nvim:1; coordinate its actual Lua driver invocation without modifying Parley's tree or duplicating teardown. Real-producer acceptance is not claimed from fakes or socket existence.
- [ ] Push, run recorded affected/full verification, close through the binary-owned review, then `sdlc merge --yes` on SHIP. Hub owns project and acceptance decisions.

## Architecture disposition

ARCH-DRY: one existing fleet owner/build pipeline, test and executable adapters only. ARCH-PURE/ORDER: pure options/envelope/reducer, bounded IO results return as events. ARCH-PURPOSE: real producer admission and Parley handoff are required, not documentation-only. ARCH-MOCK: stateful fake uses the runner seam; actual endpoint/native nvim qualify it. ARCH-CONSTRAINTS: limits above are explicit proposals for hub/consumer approval. ARCH-SECURE: argv and endpoint replies are typed/bounded, environment starts empty, roots are canonical and denied-live overlap checked before effects; no native agent credentials or network. ARCH-FUNERAL: exact joined owner end, preserve uncertainty. ARCH-EVOLVE: no persisted schema mutation; readiness Version1 allows additional metadata while preserving strict known semantics.


## Revisions

### 2026-10-22 — consumer-confirmed parent/side-doc fixtures and native selection

Hub0cdb7c57 approved the design subject to consumer confirmation. Canonical parley.nvim:1 receiptd0f4eb15 confirms argv, readiness fields, Documents, real roundtrip, startup300s/lifetime600s/stop15s and64KiB bounds, with a630s parent observation window. It requires Documents in this order: first is artifacts/first.md (registered parent); second is artifacts/first.side/second.md, frontmatter type: side-doc and parent: ../first.md. Both have distinct actual registry IDs in the advertised Namespace. This supersedes the earlier illustrative second-file path; no new field, verb or fixture owner slot. Native nvim stays selected once from the invocation's PATH through existing resolveTool so the consumer can qualify0.11.7 and0.12.5 in separate runs, then uses that exact tool through each private PATH. Add deterministic fixture-layout/namespace/distinct-ID controls and qualify both available native versions in consumer acceptance. No default-nvim substitution after selection. Consumer owns and closes its native editors before EOF and retains evidence on nonzero/uncertain teardown; it adds no producer teardown. Hub requested folding any consumer gap before change-code, so this confirmation fulfills that condition.
