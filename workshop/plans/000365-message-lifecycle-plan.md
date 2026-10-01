# Message Lifecycle Events Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Couch messaging learns about wrapper and pane lifecycle from events carried by a persistent wrapper→broker session plus Console attach/exit notifications, so an idle multi-slot Couch runs zero ownership probes, process scans or Zellij queries for messaging.

**Architecture:** A pure `Registry` reducer (couchmessage) owns registration state: wrapper sessions keyed by a connection token, Console panes keyed by pane handle, and admission state per session. A new `registry` socket holds one long-lived connection per wrapper; EOF *is* the death/exec event. The Couch service shell turns reducer effects into IO (one full authority check per lifecycle event, broker Register/Disconnect, bounded admission retries). Use-time checks shrink to cheap in-process evidence; the 1 s heartbeat, the 1 s reconcile ticker, the 10 s verification window and messaging's private `git status` are deleted.

**Tech Stack:** Go; unix domain sockets (existing 4-byte + strict JSON framing in `couchmessage/transport.go`); `golang.org/x/sys/unix` for peer credentials.

---

## Context (read first)

- Today: `wrapcmd/peer_runtime.go:147-169` dials the broker with `register` every 1 s for each wrapper's whole life. `couchcmd/message_service.go` answers with a cheap refresh inside a 10 s window (#360 stopgap) and otherwise with `admitRegistration` (two full `messageAuthority.live` checks). A 1 s ticker (`reconcile`, :382) re-checks every binding whose window lapsed, plus `ObserveResting` (`git status`). `Reserve`/`Deliver` (:427-447) always run the full check. One full check = `thread` (in-memory) + `process` (kill+sysctl) + `wrapperPID` (file) + `launch` (two `SessionOwnerProbe.Probe` = 4× `ps -axo` + 2× `zellij action list-panes`) + `thread`.
- Measured (project log, `workshop/projects/cross-slot-work-scheduling.md` §Log 2026-10-01): Couch at 170% CPU; spindump dominated by VT parsing/GC and fork/wait4; `list-panes` causes a ~10.6 KB redraw per query in an isolated Zellij.
- Couch already owns pane lifecycle: `couchtty/console.go:~414-445` installs a pane, `onExit` (:954) removes it. Nothing tells messaging. Couch already probes slot git every 10 s for its glyphs (`console_slotgit.go`).
- Delivery: broker → wrapper endpoint socket (`observe/reserve/commit/status/release`) → PTY via `dispatchPeer`. Unchanged except the duplicate-ID guard (Task 3.2).

## Check inventory (ARCH-PURPOSE)

Every check that survives names the user-visible failure it prevents and when it runs.

| Check | Prevents | Before | After |
|---|---|---|---|
| `thread` (Console pane for scope/tag, in-memory) | message to a slot not shown in Couch (detached) | every full check | at admission + at Reserve/Deliver (cheap, no IO); plus pane events |
| `process` (kill + sysctl start time) | message to a dead/reused PID | every full check | at admission; death itself arrives as session EOF |
| peer credential (`LOCAL_PEERPID`/`SO_PEERCRED` == `Binding.PID`) | a process registering someone else's binding | — (new) | once at session hello |
| `wrapperPID` file | an old wrapper still alive after replacement receiving | every full check | at admission + at Reserve/Deliver (one file read) |
| `launch` (ready file + 2 ownership probes: ps×4, zellij×2) | wrong conversation/launch nonce for the thread | ~3/s per wrapper (pre-#360), ~0.2/s after #360 | **once per admission** (lifecycle event only) |
| resting branch (`git status`) | family send to a slot off its resting branch | per reconcile + per family send | per family send only (request path); listings read Console's slot-git cache |
| endpoint `observe` RPC | stale activity/allowance in listings | 1/s per wrapper | pushed by the wrapper on change (≤1 frame/s while active, 0 idle) |

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Registry` (state + `Advance(event) []Effect`) | `cmd/internal/couchmessage/registry.go` | new |
| `RegistryEvent` (tagged: SessionOpened, SessionClosed, PaneAttached, PaneExited, AdmissionDone, RetryDue, Activity, Submit) | `cmd/internal/couchmessage/registry.go` | new |
| `RegistryEffect` (tagged: Admit, Connect, Disconnect, Observe, ScheduleRetry) | `cmd/internal/couchmessage/registry.go` | new |
| `SessionFrame` (hello / ack / activity / submit) | `cmd/internal/couchmessage/session_protocol.go` | new |
| `ReconnectBackoff` | `cmd/internal/couchmessage/backoff.go` | new |
| `RecentDeliveries` (bounded ID→receipt ring) | `cmd/internal/couchmessage/recent.go` | new |
| `Broker` actors table | `cmd/internal/couchmessage/broker.go` | modified (tombstone removal) |
| `messageVerificationWindow`, `recentlyVerified`, `refresh`, `reconcile` | `cmd/internal/couchcmd/message_service.go` | deleted |

- **Registry** — the single authority on "which binding may receive right now". State per session token: `binding`, `admission` ∈ {`Admitting`, `Admitted`, `Rejected{reason, attempt}`, `Dormant{reason}`}; per thread (scope, tag): current pane handle or none. A binding is *connected* iff its session is `Admitted` and its thread has a pane. Tests in `registry_test.go` drive event sequences with no IO.
  - **Relationships:** 1 Registry : N sessions (≤ `MaxActors`); session 1:1 Binding at a time but one Binding may appear in two tokens transiently (exec re-registration) — the newest token wins; a close of an older token is ignored (ARCH-ORDER).
  - **DRY rationale:** replaces three overlapping freshness mechanisms (heartbeat refresh, verification window, reconcile) with one state model; lifecycle producers (connection, Console) own the facts.
  - **Future extensions:** remote sessions or broker-side pokes become new event kinds, not new loops.
- **Effects** are data; the shell executes them. `Admit{token}` → one full `messageAuthority.live` + workspace resolve + endpoint observe; result returns as `AdmissionDone`. `ScheduleRetry{token, delay}` uses the backoff 0.5, 1, 2, 4, 8 s (5 attempts), then `Dormant` until the next event touching that session or thread (pane attach, submit, reconnect). No effect is ever global.
- **SessionFrame** — wrapper→broker: `hello{binding}`, `activity{observation}`, `submit{submissions}`; broker→wrapper: `ack{code}` once after hello. Strict JSON, existing frame size limit.
- **ReconnectBackoff** — 250 ms doubling, capped at 5 s, reset after a session survives 30 s. Pure `Next(attempt) time.Duration`.
- **RecentDeliveries** — wrapper-side ring of the last 64 committed message IDs with their receipts; `reserve`/`enqueue` of a known ID returns the retained receipt instead of pasting again. Dies with the wrapper (ARCH-FUNERAL: in-memory, bounded at 64).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `SessionServer` (registry socket, long-lived conns) | `cmd/internal/couchmessage/session_transport.go` | new | unix listener |
| `SessionClient` (wrapper side, reconnect loop) | `cmd/internal/couchmessage/session_transport.go` | new | unix dial |
| `PeerPID` | `cmd/internal/couchmessage/peercred_{darwin,linux}.go` | new | getsockopt |
| `messageService` (registry shell) | `cmd/internal/couchcmd/message_service.go` | modified | authority probes, broker, timers |
| `Console` message lifecycle hooks | `cmd/internal/couchtty/console_messages.go` | modified | pane install/exit |
| `peerRuntime` | `cmd/internal/wrapcmd/peer_runtime.go` | modified | session client instead of heartbeat |
| `messageidle` probe | `probes/messageidle/` | new | PATH shims counting `zellij`/`ps` from Couch |

- **SessionServer** — separate socket `SocketPath(namespace, "registry")`, prepared under the supervisor lease like the broker socket. Accepts ≤ `MaxActors` concurrent sessions (extra connections are closed with `ack{refused}`); reads frames until EOF; no idle timeout (unix sockets do not drop silently; death closes the fd). Each connection gets a monotonically increasing token. Emits `SessionOpened/Activity/Submit/SessionClosed` to the service's single event loop. *Injected into:* `messageService` (tests use the real socket in a temp dir, as current transport tests do).
- **SessionClient** — runs for the wrapper lifetime: dial, hello, wait ack, then send activity/submit frames; on any error close and sleep `ReconnectBackoff`. The fd is CLOEXEC, so a SIGUSR2 `syscall.Exec` closes it and the re-exec'd wrapper opens a new session — no special case.
- **Console hooks** — `Console.SetMessageLifecycle(func(PaneEvent))`; called after pane install commits (PaneAttached{thread, handle}) and in `onExit` after deletion (PaneExited{thread, handle}). Called outside `c.mu` via a non-blocking buffered channel owned by the service; on overflow the service marks every thread "needs check" once (bounded, logged), never blocks the Console.
- **Service event loop** — one goroutine owns the Registry; sessions, Console hooks, admission results and retry timers all feed one channel, so ordering is the arrival order and every interleaving is reproducible by feeding events in tests. Admissions run on workers with concurrency ≤ 4 (broker-restart burst of N wrappers costs N checks, 4 at a time).
- **Legacy wrappers** — a running wrapper from an older binary still sends `register` on the broker socket. The new broker answers `unsupported` with no probes; such slots become reachable after relaunch. (Operator decision point — see Open questions.)

### Crash and lifecycle classification (Done-when 2–3)

| Event | Observed mechanism | Registry outcome |
|---|---|---|
| Wrapper start | new session hello | Admit → Connect if pane present |
| Couch pane attach (incl. reattach) | Console hook | sessions for that thread with `Rejected/Dormant/Admitted` → Admit (one check) |
| Couch detach / pane exit | Console hook | Disconnect bindings on that thread; sessions stay open (`Admitted`, not connected) |
| Replacement (new wrapper, same slot) | new hello; old session EOF | new admitted → broker `Register` disconnects old same-slot binding (existing); old EOF ignored if token stale |
| Wrapper crash / exit | kernel closes fd → EOF | Disconnect (pending→Cancelled, delivering→Indeterminate; existing model) |
| Agent restart (SIGUSR2 exec) | CLOEXEC EOF + new hello, possibly identical binding | newer token wins; late EOF of old token does not disconnect it |
| Agent crash, wrapper alive | wrapper reducer sees child exit | delivery Indeterminate/Cancelled (existing); session stays; wrapper exits → EOF |
| Broker (Couch) crash/restart | wrappers see EOF, reconnect with backoff ≤5 s | fresh registry; each hello admitted once; in-flight receipts lost at broker, retained at wrapper |
| Zellij crash | wrapper dies (EOF) and Couch pane child exits (hook) | both paths Disconnect; whichever arrives second is a no-op |
| In-flight delivery during replacement | delivery already committed to the old wrapper | completes or ends Indeterminate there; never re-sent to the new one |
| Sender retries an ID after broker restart | broker has no receipt | wrapper `RecentDeliveries` returns the retained receipt; no second paste |

### Operating envelope (ARCH-CONSTRAINTS)

- Workload: background lifecycle + online request (send ≤2 s client deadline). Scale: ≤ `MaxActors`=128 sessions (requirement, existing cap); typical 5–30 (operator setup, assumption).
- Idle CPU attributable to messaging: zero probes, zero process spawns, zero timers per session (requirement, Done-when 1). Measured by the counting acceptance test and the live shim.
- Per lifecycle event: ≤1 full check (~2 ownership probes, ~100–300 ms measured in M1). Burst after broker restart: N checks, concurrency 4.
- Active agent: ≤1 activity frame/s per wrapper (coalesced), ~200 bytes.
- Broker absent: ≤1 `connect()` per 5 s per wrapper; no scans.
- Send path latency drops: Reserve/Deliver no longer spawn processes.

### Trust (ARCH-SECURE)

Registry socket lives in the existing private 0700 per-uid dir. Hello bindings are untrusted input: strict-decoded, `Binding.Validate`, peer PID must equal `Binding.PID`, then the full authority check before admission. Activity/submit frames apply only to the token's own binding — a connection cannot speak for another binding (today `operator-submit` accepted any binding and relied on a full check). Malformed frame → close the session (Dormant via EOF), never a fabricated observation.

### Lifetimes (ARCH-FUNERAL)

- Sessions/tokens: in memory; removed on EOF.
- Broker actor tombstones (today never deleted, cap 128): a disconnected actor is deleted when a different binding registers for the same slot (state carried first) and lazily in `sweep()` once disconnected for longer than the receipt TTL (1 h).
- Wrapper `RecentDeliveries`: 64 entries, in-memory.
- Measurement archive: a few KB of text in `workshop/plans/000365-message-lifecycle-measurements.md`, permanent, archived with the issue.

## Chunk 1: M1 — Baseline and counting harness

### Task 1.1: Probe counters + idle acceptance test (red)

**Files:** Test: `cmd/internal/couchcmd/message_idle_test.go`; Modify: `cmd/internal/couchcmd/message_service.go` (inject clock/ticker seam only).

- [ ] Add a `ticker func(time.Duration) (<-chan time.Time, func())` field to `messageService` (default `time.NewTicker`) so tests drive reconcile deterministically.
- [ ] Write `TestMessageIdleMultiSlotRunsNoProbes`: real broker socket in `t.TempDir()`, `messageAuthorityFake` with counting `launch`, `process`, `branch`; three wrappers registered through the public protocol; then advance the fake clock 60 s in 1 s ticks with no traffic. Assert `launch == 0`, `process == 0`, `branch == 0` after the registration phase.
- [ ] Run `go test ./cmd/internal/couchcmd -run TestMessageIdle -v` — Expected: FAIL (current reconcile re-checks every 10 s); note the counts. Commit it behind `t.Skip("enabled in M2 (#365); baseline: N launch checks/60s")` so M1 lands green; Task 2.4 deletes the skip.
- [ ] Commit `#365 M1: idle messaging probe-count acceptance test`.

### Task 1.2: Live idle measurement probe

**Files:** Create `probes/messageidle/run.sh`, `probes/messageidle/SKILL.md`; reuse `probes/zellijcalls/` (extend its shim to also wrap `ps`, keyed by parent PID).

- [ ] Script: given a Couch PID and a duration (default 120 s), arm shims, record per-command invocation counts whose ancestor is Couch, sample Couch CPU (`ps -o time= -p PID` start/end → CPU-seconds), record Zellij output bytes via the existing isolated query experiment config, disarm. Output a TSV + summary.
- [ ] Run against the live Couch on current `main` build with the operator's usual slot count, idle 120 s. Record commands, slot count, counts and CPU-seconds in `workshop/plans/000365-message-lifecycle-measurements.md` under `## Before`.
- [ ] Commit `#365 M1: live idle messaging baseline`.
- [ ] `sdlc milestone-close --issue 365 --milestone M1`.

## Chunk 2: M2 — Lifecycle protocol replaces polling

### Task 2.1: Registry reducer (TDD, pure)

**Files:** Create `cmd/internal/couchmessage/registry.go`, `registry_test.go`.

- [ ] Write tests first, each feeding events and asserting effects/state:
  - `TestRegistryHelloWithPaneAdmitsThenConnects`
  - `TestRegistryHelloBeforePaneConnectsOnAttach` (attach race)
  - `TestRegistryPaneExitDisconnectsButKeepsSession`; `TestRegistryReattachReadmitsOnce`
  - `TestRegistryStaleCloseDoesNotEraseNewerToken` (exec with identical binding, both orders of close vs hello)
  - `TestRegistryStalePaneExitIgnored` (exit for handle h1 after attach of h2)
  - `TestRegistryAdmissionFailureBacksOffThenDormant` (5 retries 0.5…8 s, then no effect until an attach/submit)
  - `TestRegistryAdmissionResultForClosedTokenIgnored` (late completion)
  - `TestRegistryActivityAndSubmitScopedToToken`
  - `TestRegistryCapacityRefusesBeyondMaxActors`
- [ ] Run — FAIL (undefined). Implement `Registry`, events and effects as tagged structs with an `Advance`. Run — PASS. Commit.

### Task 2.2: Session frames, backoff, peer PID

**Files:** Create `couchmessage/session_protocol.go`, `backoff.go`, `peercred_darwin.go`, `peercred_linux.go` + tests.

- [ ] Tests: frame round-trip and strict rejection of unknown fields/oversize; `ReconnectBackoff` sequence and cap; `PeerPID` of a socketpair equals `os.Getpid()`.
- [ ] Implement; run `go test ./cmd/internal/couchmessage -run 'Frame|Backoff|PeerPID' -v` — PASS. Commit.

### Task 2.3: SessionServer / SessionClient over real sockets

**Files:** Create `couchmessage/session_transport.go`, `session_transport_test.go`.

- [ ] Tests (real sockets in `t.TempDir()`): hello→ack→events delivered in order; client process kill (helper subprocess, as `socket_cleanup_test.go` does) yields `SessionClosed`; server close → client reconnects after restart within the backoff; 129th session refused; malformed frame closes only that session.
- [ ] Implement. Commit.

### Task 2.4: Service rewired to the Registry; polling deleted

**Files:** Modify `couchcmd/message_service.go`, `couchcmd/run.go`, `couchcmd/message_service_test.go`.

- [ ] Single event loop goroutine owns `Registry`; executor maps effects to: admission worker (full `live` + workspace + observe; ≤4 concurrent), `broker.Register`/`Disconnect`, retry timers (via the injected ticker/clock seam), `ReconcileObservation` for activity/submit.
- [ ] `messageEndpoint.Observe/Reserve/Deliver` use `cheapLive` = `thread` + `wrapperPID` + registry says connected; no `launch`.
- [ ] Delete `messageVerificationWindow`, `verified`, `recentlyVerified`, `refresh`, `reconcile`, the reconcile ticker, background `ObserveResting`; legacy `register`/`operator-submit` ops answer `unsupported` without probes.
- [ ] Caller checks for CLI requests (`send` etc.) use the registry's admitted binding for (scope, tag, session, nonce) + `cheapLive`.
- [ ] Update/replace tests that asserted the window (`TestMessageHeartbeatSkipsFullCheckWithinWindow`, `TestMessageEndpointObserveReusesRecentCheckButReserveNever`, `TestMessageReconcileForgetsDeadBindings`) with registry-driven equivalents. Remove the skip from `TestMessageIdleMultiSlotRunsNoProbes`; run — PASS.
- [ ] Commit.

### Task 2.5: Console lifecycle hooks and listing sources

**Files:** Modify `couchtty/console.go` (install + `onExit`), `couchtty/console_messages.go`, `couchtty/console_slotgit.go` (read accessor), `couchmessage/broker.go` (listing resting source).

- [ ] Tests: `TestConsoleEmitsPaneAttachedAfterInstall`, `TestConsoleEmitsPaneExitedAfterRemoval`, hook never blocks with a full channel; broker `--actors` resting branch comes from the injected slot-git reader (no git call — extend `broker_actors_test.go`'s counting probe).
- [ ] Implement `SetMessageLifecycle`, `SlotGitStatus(root)`. Commit.

### Task 2.6: Wrapper session client

**Files:** Modify `wrapcmd/peer_runtime.go`, `wrapcmd/peer_delivery.go`; tests `peer_runtime_test.go`.

- [ ] Replace the ticker loop with `SessionClient`; `d.submit` wakeups send `submit{submissions}`; activity changes coalesced to ≤1 frame/s (`activity{observation}`), none while idle.
- [ ] Test with a real broker service: register once, idle 5 s → zero frames after hello; one submit → one frame; broker restart → re-hello.
- [ ] `go test ./cmd/internal/...` green; commit.
- [ ] `sdlc milestone-close --issue 365 --milestone M2`.

## Chunk 3: M3 — Failure semantics, measurement, docs

### Task 3.1: Crash/interleaving suite

**Files:** `couchcmd/message_lifecycle_test.go` (service + real sockets + fake authority).

- [ ] `TestLifecycleAbsentEndpoint` (exact target never connected → `unavailable`; family → `not-dispatched`).
- [ ] `TestLifecycleWrapperDiesMidDelivery` → receipt `Indeterminate`, no resend.
- [ ] `TestLifecyclePartialInputIsIndeterminate` (existing reducer path through the service).
- [ ] `TestLifecycleLostReceiptAfterBrokerRestartNoDuplicate` (needs 3.2).
- [ ] `TestLifecycleDelayedFramesFromOldIncarnationIgnored`.
- [ ] `TestLifecycleReplacementCompletesAtOldIncarnation` (in-flight commit finishes at old wrapper; new wrapper never receives it).
- [ ] `TestLifecycleStartDetachReattachReplaceRestart` (full sequence, asserting registry state after each step and that no step erases a newer registration).

### Task 3.2: Duplicate guard across broker restart

**Files:** `couchmessage/recent.go` (+test), `wrapcmd/peer_delivery.go`, `couchmessage/broker.go` (map retained receipt to the sender's response).

- [ ] Wrapper retains last 64 receipts; `reserve` of a retained ID answers `already-committed` with the receipt; broker records and returns it. Test both terminal and in-flight retained receipts.

### Task 3.3: Broker tombstone funeral

- [ ] `TestBrokerDropsTombstoneOnSlotReplacement`, `TestBrokerSweepsLongDisconnectedActors`; implement in `broker.go`.

### Task 3.4: After-measurement and docs

- [ ] `make build`; operator relaunches Couch + slots (old wrappers are legacy); rerun `probes/messageidle/run.sh` with the same slot count and duration; record `## After` and the delta (zellij/ps invocations, CPU-seconds) in the measurements file. Repeat the isolated Zellij query experiment to restate redraw bytes per query avoided.
- [ ] Update `atlas/couch.md` "Live peer messages": lifecycle protocol, crash table, remaining uncertainty (receipts lost at broker restart unless the wrapper retains them; no exactly-once task execution; legacy wrappers). Replace the "Liveness is recomputed, never stored" line with the new rule.
- [ ] Lesson in `workshop/lessons.md` if review finds one.
- [ ] Full test sweep per memory: `make -k test`, scratchpad-TMPDIR `test-changelog`, `go test ./...`.
- [ ] Operator smoke test (live): two slots, send, detach/reattach, relaunch one slot, restart Couch, send again.
- [ ] `sdlc close --issue 365 --verified '…'`.

## Open questions for the operator

1. **Legacy wrappers** — refuse old `register` heartbeats (slots reachable only after relaunch), or keep a legacy path that still polls until every wrapper is new? Plan assumes refuse.
2. **Detached slots** — keep #353's rule that a slot with no live Couch pane cannot receive (plan keeps it, now via pane events)?
3. **Activity push** — acceptable to have wrappers push ≤1 small frame/s while their agent is producing output, to keep `--actors` quiet-time current without polling?
