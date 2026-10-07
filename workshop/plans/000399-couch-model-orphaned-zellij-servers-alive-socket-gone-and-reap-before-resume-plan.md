# Orphaned zellij servers: observe, name, reap — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A zellij server that is alive but has lost its socket becomes a named
state (`orphaned`) everywhere couch looks — probe, inventory, recovery report,
startup, switcher — and a confirmed `reap` operation takes down its whole process
tree so the thread can be resumed.

**Architecture:** Detection is one pure rule over one bulk `ps` snapshot: a
`zellij --server <socket>` process whose `<socket>` path no longer exists. The
launcher owns that rule and the reaper (it already owns server discovery and the
identity-checked `KillServer`). Couch adds one session state, one thread reason,
one operation, and one recovery-report agent value; every table that enumerates
those vocabularies gains the new member rather than a special case.

**Tech Stack:** Go; zellij; `ps`; existing `procutil` identity tokens.

---

## Why, in one paragraph

On 2026-10-06 a test run deleted `$TMPDIR`, which held zellij's sockets. All 20
servers kept running with their agents but became unreachable. Couch had no word
for this: the inventory (fed by `list-sessions`, which no longer lists them) called
the threads `parked` and offered `resume`, and `resume`'s `Probe` then failed with
a raw `zellij … list-panes: exit status 1`. Had the probe been bypassed, resume
would have started a second agent on a conversation the orphan was still writing.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `SessionServerIdentity` gains `Socket`; the lowercase `sessionServerIdentity` duplicate is deleted | `cmd/internal/launcher/session_owner.go`, `session_quiescence.go` | modified / deleted |
| `ParseServerProcesses` (ps text → `[]SessionServerIdentity`) | `cmd/internal/launcher/session_servers.go` | new |
| `SocketState` (present / gone / unknown) and `ClassifyServers` (servers + socket states → per-session reachable / orphaned / unresolved) | `cmd/internal/launcher/session_servers.go` | new |
| `SessionOwnerOrphaned` | `cmd/internal/launcher/session_owner.go` | modified (new state) |
| `SessionOrphaned` + `SessionObservation.Orphan` | `cmd/internal/couchcore/sessionevidence.go` | modified |
| `ProjectSessionPresence` | `cmd/internal/couchcore/sessionevidence.go` | modified (orphan input) |
| `ReasonOrphanedServer` | `cmd/internal/couchcore/threadreason.go` | new |
| `ClassifyThread` | `cmd/internal/couchcore/actionableinventory.go` | modified |
| `ActorActions` | `cmd/internal/couchcore/actor_actions.go` | modified (offers `reap`) |
| `AgentOrphaned`, `OfferReap`, rule-A `reap`→`resume` | `cmd/internal/couchcore/recoverplan.go` | modified |
| `ResumeOrphanedServer` diagnostic code | `cmd/internal/couchcore/resume.go` | new |
| `ReapPlan` (tree snapshot → ordered signal plan) | `cmd/internal/launcher/session_reap.go` | new |

- **SessionServerIdentity (+Socket)** — what `ps` says about one zellij server.
  Today two structs carry this fact (`SessionServerIdentity` in session_owner.go
  and `sessionServerIdentity` in session_quiescence.go, copied field-for-field in
  `osSessionOwnerIO.SessionServers`); both drop the socket path, the third argv
  field and the only evidence of orphaning. The exported one gains `Socket` and the
  lowercase copy is deleted (ARCH-DRY, PQ-1).
  - **DRY rationale:** `zellijServerPIDs` and `isExactZellijServerCommand` already
    parse this argv; they become thin wrappers over `ParseServerProcesses`, so there
    is one argv grammar and one struct.
- **SocketState** — `present`, `gone` (Lstat says ENOENT, and only that) or
  `unknown` (any other Lstat error, or a non-socket file). Only `gone` makes an
  orphan; `unknown` projects to `SessionUnresolved`, so an unreadable socket
  directory can never manufacture orphans (PQ-2, ARCH-SECURE: the trusted fact is
  ENOENT, nothing weaker).
  - **Future extensions:** a socket that exists but refuses connections (stale
    socket file) is a third state; it widens `ClassifyServers`, not its callers.
- **SessionOrphaned** — the session-evidence value "a server for this name is
  alive and unreachable". The comment on `SessionState` forbids a value no
  producer emits; this one has a producer (`SessionPresence` via `ClassifyServers`).
  `SessionObservation` regains a field (the comment says M2 would re-add one with
  its consumer): `Orphan *launcher.SessionServerIdentity`, consumed by the diagnostic and by reap.
- **ReasonOrphanedServer** — `unusable/orphaned-server`. Not archive-eligible
  (`ArchivableState` must refuse it: the agent may still be writing), not
  resumable, offers only `reap`.
- **ReapPlan** — given the server, a pid→ppid table and identity tokens, the
  ordered list of `(pid, identity, signal)` steps: TERM the descendants (wrap,
  term, nvim, title), TERM the server, then KILL whatever survives the bound. Pure
  so the escalation order and the identity gate are unit-testable.
  - **Relationships:** 1 server : N descendants, snapshotted once while the parent
    chain is intact (children reparent to launchd the moment the server dies,
    which is why the 2026-10-06 manual recovery found PPID-1 residue).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ServerSnapshot` (one `ps -axo pid=,ppid=,command=` + `Lstat` per server socket) | `cmd/internal/launcher/session_servers.go` | new | `ps`, filesystem |
| `SessionOwnerProbe.Probe` | `cmd/internal/launcher/session_owner_os.go` | modified | zellij, `ps` |
| `ScopedThreadArtifactCollisionChecker.SessionPresence` | `cmd/internal/couchcore/artifactcollision.go` | modified | `list-sessions` + `ServerSnapshot` |
| `Reaper` (signal + identity re-read + bounded wait) | `cmd/internal/launcher/session_reap.go` | new | signals |
| `reap` operation | `couchcore/ops.go`, `operationdispatch.go`, `slot_operation.go`, `couchcmd/cli.go`, `couchcmd/messages.go`, `couchmessage/protocol.go`, `couchcmd/message_service.go`, `couchcmd/run.go` | new | couch live owner |

- **ServerSnapshot** is injected into `Probe` (through `SessionOwnerIO`) and into
  `SessionPresence`; tests use a fake process table with a fake socket set — a
  stateful fake, not call mocks (ARCH-MOCK). One snapshot per inventory refresh,
  never one per row (memory rule: listings answer from bulk reads).
- **Reaper** wraps `kill(2)` behind a `ProcessTable` interface
  (`Snapshot()`, `Identity(pid)`, `Signal(pid, sig)`); the fake models processes
  that ignore SIGTERM, which is the case that bit on 2026-10-06.

## Non-goals

- Reaping automatically: an orphan's agent may still be writing; the operator
  confirms every reap.
- A socket file that exists but refuses connections (stale socket): `present`
  stays `present`; a refused `list-panes` stays an error. A third socket state can
  widen `SocketState` later.
- The lifecycle queue's capacity-one behaviour noted in the issue Log (switches
  waiting behind remote resumes): separate issue.
- Making `recover` replace resume and reboot in the menu: both stay.

## Operating envelope (ARCH-CONSTRAINTS)

- Detection: interactive-refresh path. Adds one `ps` (≈20 ms) per inventory
  refresh plus one `Lstat` per zellij server on the host (tens). No per-row IO.
- Reap: operator-confirmed, rare. Bound: TERM, wait up to 3 s polling 50 ms, KILL,
  wait up to 2 s. Never signals a pid whose start identity changed (ARCH-SECURE:
  the trusted fact is the identity token read at snapshot time, re-read before
  every signal).

## Ordering (ARCH-ORDER)

Reap: observe (re-probe; refuse unless still orphaned with the same server
identity) → snapshot tree → TERM descendants → TERM server → bounded wait → KILL
survivors (identity-gated) → bounded wait → existing title-poller/nvim pidfile
reapers for the tag → `delete-session --force` record cleanup via the existing
quiesce → re-observe: must read absent. Any refusal before the first signal
leaves the world untouched.

## Lifecycle (ARCH-FUNERAL)

Reap creates nothing. It ends an orphaned server and every descendant snapshotted
from it; the thread then reads `parked` (or `session-gone`) through the existing
classifier, and `resume` is its ordinary next step.

---

## Chunk 1 — M1: observe and name the orphan

### Task 1: server parsing (with socket path) and classification (launcher, pure)

**Files:**
- Create: `cmd/internal/launcher/session_servers.go`
- Test: `cmd/internal/launcher/session_servers_test.go`
- Modify: `cmd/internal/launcher/session_quiescence.go` (`zellijServerPIDs`,
  `isExactZellijServerCommand` delegate to the new parser)

- [ ] **Step 1: failing tests**

```go
func TestParseServerProcessesKeepsTheSocketPath(t *testing.T) {
	raw := "  101 1 /opt/homebrew/bin/zellij --server /T/zellij-501/contract_version_1/📁1-37\n" +
		"  102 101 /usr/local/bin/pair wrap\n" +
		"  103 1 zellij --server /T/zellij-501/contract_version_1/other extra\n"
	got := ParseServerProcesses(raw)
	want := []SessionServerIdentity{{PID: 101, Session: "📁1-37", Socket: "/T/zellij-501/contract_version_1/📁1-37"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
}

func TestClassifyServers(t *testing.T) {
	servers := []SessionServerIdentity{{PID: 1, Session: "a", Socket: "/s/a"}, {PID: 2, Session: "b", Socket: "/s/b"}}
	sockets := map[string]SocketState{"/s/a": SocketPresent, "/s/b": SocketGone}
	got := ClassifyServers(servers, sockets)
	if got["a"].Orphaned || !got["b"].Orphaned || got["b"].Server.PID != 2 {
		t.Fatalf("got %+v", got)
	}
}
```

Strategy (one line per risky function): `ParseServerProcesses` — `go test -fuzz`
seed corpus of real `ps` lines plus a fuzz target asserting it never panics and
every result round-trips `isExactZellijServerCommand`; `ClassifyServers` — table
over {present, gone, unknown} × {one server, two servers for a name}.

- [ ] **Step 2:** `go test ./cmd/internal/launcher -run 'ParseServer|ClassifyServers'` → FAIL (undefined).
- [ ] **Step 3: implement** `SessionServerIdentity.Socket`; delete
  `sessionServerIdentity` (quiescence ops use the exported type);
  `ParseServerProcesses(raw string) []SessionServerIdentity` (exact 3-field argv,
  `filepath.Base(socket) == session`); `SocketState` + `ObserveSocket(lstat) SocketState`
  (ENOENT → gone, socket mode → present, anything else → unknown);
  `ServerState{Server SessionServerIdentity; Orphaned, Unresolved bool}`;
  `ClassifyServers(servers, sockets map[string]SocketState) map[string]ServerState`
  (a `gone` socket → orphaned; `unknown` or two servers for one name → unresolved).
  Rewrite `zellijServerPIDs`/`isExactZellijServerCommand` on top.
- [ ] **Step 4:** run launcher tests → PASS (including existing quiescence tests).
- [ ] **Step 5:** commit `#399 M1: launcher: parse zellij servers with their socket path`.

### Task 2: ServerSnapshot (launcher, IO) and Probe's orphaned state

**Files:**
- Modify: `cmd/internal/launcher/session_servers.go` (`ServerSnapshot`, OS impl)
- Modify: `cmd/internal/launcher/session_owner.go` (`SessionOwnerOrphaned`)
- Modify: `cmd/internal/launcher/session_owner_os.go` (`Probe`, `SessionOwnerIO`)
- Test: `cmd/internal/launcher/session_owner_test.go` (extend the `ownerWorld` fake, `session_owner_test.go:49`)

- [ ] **Step 1: failing test** using the existing `ownerWorld` fake, extended
  with a socket table (`map[string]SocketState`) behind `SessionOwnerIO.Socket(path) SocketState`:

```go
func TestProbeReportsOrphanedServerInsteadOfAnError(t *testing.T) {
	io := newOwnerWorld(t) // session_owner_test.go
	io.addServer(SessionServerIdentity{PID: 7, Identity: "t7", Session: "📁1-37", Socket: "/gone"})
	io.sockets["/gone"] = SocketGone
	io.panesErr = errors.New("exit status 1")
	got, err := SessionOwnerProbe{IO: io}.Probe(ctx, "📁1-37", dataDir, scope, tag)
	if err != nil || got.State != SessionOwnerOrphaned || got.Server.PID != 7 {
		t.Fatalf("got %+v, %v", got, err)
	}
	if io.panesCalls != 0 {
		t.Fatal("an orphan has no socket to ask")
	}
}
```

Strategy: table over socket {present, gone, unknown} × list-panes {ok, error}:
only `gone` yields orphaned (and skips list-panes); `unknown` yields
`SessionOwnerUnknown` with a diagnostic; `present` + error stays an error.

- [ ] **Step 2:** run → FAIL.
- [ ] **Step 3: implement:** `SessionServerIdentity` gains `Socket`; `Probe`
  checks `Socket(before[0].Socket)` before `SessionPanes` (gone → orphaned; unknown → `SessionOwnerUnknown` + diagnostic); on absence returns
  `{State: SessionOwnerOrphaned, Server: before[0], Diagnostic: OrphanDiagnostic(name, pid)}`.
  `OrphanDiagnostic` is the one sentence: `"<name>: server PID <n> lost its socket — reap to resume"`.
- [ ] **Step 4:** PASS. **Step 5:** commit `#399 M1: launcher: Probe names an orphaned server`.

### Task 3: SessionOrphaned in session presence (couchcore)

**Files:**
- Modify: `cmd/internal/couchcore/sessionevidence.go`
- Modify: `cmd/internal/couchcore/artifactcollision.go` (`SessionPresence` takes one `ServerSnapshot`)
- Modify: `cmd/internal/couchcore/artifactcollision_fake.go` (`SetOrphanedServer`)
- Test: `cmd/internal/couchcore/sessionevidence_test.go`

- [ ] **Step 1: failing test** on the pure projector:

```go
func TestProjectSessionPresenceSeesAnOrphan(t *testing.T) {
	bindings := []SessionNameBinding{{Address: a, SessionName: "📁1-37"}}
	servers := map[string]launcher.ServerState{"📁1-37": {Server: launcher.SessionServerIdentity{PID: 9}, Orphaned: true}}
	got := ProjectSessionPresence(bindings, nil, servers, map[string]int{"📁1-37": 1})
	if got[a].State != SessionOrphaned || got[a].Orphan.PID != 9 {
		t.Fatalf("got %+v", got[a])
	}
}
```

Also: an orphan for a name two addresses claim stays `SessionUnresolved`
(the existing `uniquelyClaimed` rule); a listed-live session wins over a stale
server row (cannot happen with one server per socket, but the projector must not
call a reachable session orphaned).

- [ ] **Step 2:** FAIL. **Step 3:** add `SessionOrphaned` (String `"orphaned"`),
  `SessionObservation.Orphan *launcher.SessionServerIdentity`; `ProjectSessionPresence`
  gains the `servers` argument and checks it before the absent default. The OS
  `SessionPresence` takes one `ServerSnapshot` alongside `LivenessContext`; a
  snapshot error makes the bound addresses unresolved (fail closed), never absent.
  Update the doc comment on `SessionState` (four values now; name the producer).
- [ ] **Step 4:** PASS (fix every `ProjectSessionPresence` caller). **Step 5:** commit.

### Task 4: ReasonOrphanedServer through classification and actions

**Files:**
- Modify: `couchcore/threadreason.go`, `couchcore/actionableinventory.go`
  (`ClassifyThread`, `ArchivableState`), `couchcore/actor_actions.go`
- Test: `couchcore/classify_test.go`, `couchcore/actor_actions_test.go`

- [ ] **Step 1: failing test:** a record with resolvable parked proof AND
  `Session.State == SessionOrphaned` classifies `ThreadUnusable, ReasonOrphanedServer`
  — never `ThreadParked`. `ArchivableState(ThreadUnusable, ReasonOrphanedServer)`
  is false. Update `actorActionSpec` so the derived-domain test expects
  `ActorActions` → `nil` for it in M1 (M2 changes this to `["reap"]`).
- [ ] **Step 2:** FAIL. **Step 3:** add the reason to `AllThreadReasons` with an
  operator label ("server lost its socket"); in `ClassifyThread` the orphan check
  sits before the unresolved/parked decisions (an orphan is evidence a session is
  ALIVE, so it must outrank every "no session" reading); exclude it in
  `ArchivableState`.
- [ ] **Step 4:** run `go test ./cmd/internal/couchcore -run 'Classify|Reason|ActorActions|Archiv'` and the
  `TestEveryReasonIsProducedBySomeShape` / `TestEveryReasonHasADistinctOperatorLabel` guards → PASS.
- [ ] **Step 5:** commit.

### Task 5: Named diagnostic at resume, startup and the switcher

**Files:**
- Modify: `couchcore/resume.go` (`ResumeOrphanedServer`), `couchcore/artifactcollision.go`
  (`NamedPairSessionContext` maps `SessionOwnerOrphaned` to the coded refusal),
  `couchcore/startup.go` (`startupResumeRefusal` prints the code's sentence)
- Test: `couchcore/resume_test.go`, `couchcore/startup_test.go`

- [ ] **Step 1: failing tests:** resume of an orphaned thread returns a
  `ResumeRefusal` with code `resume-orphaned-server` whose message is
  `OrphanDiagnostic(...)`, and no agent is spawned (fake runner ops empty);
  startup's refusal text contains "lost its socket — reap to resume" and not
  "exit status". `TestEveryResumeDiagnosticCodeIsProducedBySomeSite` must pass.
- [ ] **Step 2–4:** FAIL → implement → PASS. The switcher's reattach-pass row
  already renders `ResumeDiagnosticOf(err)`; assert with the existing
  `menu_reattach` test helpers that the row reads the sentence.
- [ ] **Step 5:** commit.

### Task 6: Recovery report says `orphaned` (no step yet)

**Files:**
- Modify: `couchcore/recoverplan.go` (`AgentOrphaned`, `agentEvidence`,
  `AllEvidenceAgents`, `HoldOrphanedServer`), `couchcore/slotreport.go` (rank map)
- Test: `couchcore/recoverplan_test.go` (+ the `AllEvidenceAgents` loops listed in
  the survey: `slotfailure_test.go`, `slotsave_test.go`, `slotplan_test.go`,
  `slotreconcile_test.go`)

- [ ] **Step 1: failing test:** an orphaned thread row yields agent
  `{State: "orphaned", PID: n}` and class hold `orphaned-server`, no steps.
  Totality (`TestDeriveRecoverPlanIsTotalOverTheEvidenceDomain`) includes the new
  agent value; `consistent()` ties `AgentOrphaned` to `OfferNone` in M1.
- [ ] **Step 2–4:** FAIL → implement → PASS (`go test ./cmd/internal/couchcore`).
- [ ] **Step 5:** commit; then **`sdlc milestone-close --issue 399 --milestone M1`**.

## Chunk 2 — M2: reap

### Task 7: ReapPlan and Reaper (launcher)

**Files:**
- Create: `cmd/internal/launcher/session_reap.go`, `session_reap_test.go`

- [ ] **Step 1: failing tests** against a fake `ProcessTable` (pid → ppid,
  identity, alive, ignoresTERM):
  - server 10 with children 11 (wrap, ignores TERM), 12 (nvim), grandchild 13
    (title): after `Reap`, all four are gone; 11 received TERM then KILL.
  - a descendant whose identity changes between snapshot and signal (pid
    recycled) is never signalled.
  - the server's identity changed before the first signal → `Reap` refuses
    with no signal sent at all.
  - waits are bounded: a process that survives KILL (fake "unkillable") makes
    `Reap` return an error naming the pid after the bound, not hang.
- [ ] **Step 2:** FAIL. **Step 3:** `PlanReap(root SessionServerIdentity, table Snapshot) []ReapStep`
  (pure: descendants BFS from the snapshot, children before the server);
  `Reaper{Table ProcessTable; TermWait, KillWait, Poll time.Duration}.Reap(ctx, root)`.
  The OS `ProcessTable` reads `ps -axo pid=,ppid=` once and `procutil.Identity`.
- [ ] **Step 4:** PASS. **Step 5:** commit.

### Task 8: `Couch.Reap` and the operation surface

**Files:**
- Modify: `couchcore/ops.go` (declare `reap`: `ExecuteLiveOwner`, `EffectProcess`,
  `ConfirmRequired`, `RowAction: false`), `couchcore/operationdispatch.go`,
  `couchcmd/slot_operations.go` (the socket side of resume/reboot),
  `couchcore/slot_operation.go` (`slotOperations`), `couchcore/actor_actions.go`
  (offer `["reap"]` for `ReasonOrphanedServer`), new `couchcore/reap.go`
  (`Couch.Reap`), `couchcmd/cli.go` (`--reap` beside `--resume`/`--reboot`),
  `couchcmd/messages.go`, `couchmessage/protocol.go`, `couchcmd/message_service.go`,
  `couchcmd/run.go` (`operationOwnsLive`, `operationUsesCurrentRepoScope`)
- Test: `couchcore/reap_test.go`, `couchcmd/cli_test.go`, the operation-table audit tests

- [ ] **Step 1: failing tests:**
  - `Couch.Reap` on an orphaned thread: re-probes, reaps via an injected fake
    reaper, runs the tag's title/nvim pidfile reapers and the session-record
    quiesce, and the next inventory reads the thread `parked`.
  - `Couch.Reap` on a thread that is not orphaned (live, parked, unknown) refuses
    with no signal.
  - `couch --reap pair:2` without `--confirm` is a usage error; with it, the CLI
    dispatches through the socket like `--resume`.
  - Operation audits (`OperationNames` identity, `OperationConfirms`) pass.
- [ ] **Step 2–4:** FAIL → implement → PASS. `Couch.Reap` order is the plan's
  ARCH-ORDER list; the session-record cleanup reuses `quiesceZellijSession`
  (it already tolerates a vanished server).
- [ ] **Step 5:** commit.

### Task 9: Recovery report steps `reap` → `resume`

**Files:** `couchcore/recoverplan.go` (`OfferReap`, `offerOf`, `actions`,
`withRuleA`, `consistent`), `couchcore/recoverplan_test.go` (totality step switch
gains `"reap"`), `couchcore/slot_operation.go` (`SlotOperationCommand` renders
`couch --reap <addr> --confirm`)

- [ ] **Step 1: failing tests:** orphaned row → steps `[reap, resume]` with
  commands `couch --reap pair:2 --confirm`, `couch --resume pair:2`; after a
  simulated reap (thread now parked) the report gives `[resume]` only.
- [ ] **Step 2–4:** FAIL → implement → PASS.
- [ ] **Step 5:** commit; update `couch --skill`'s recovery section (one line:
  orphaned rows reap first, confirm with the operator); then
  **`sdlc milestone-close --issue 399 --milestone M2`**.

### Task 9b: `recover` — the switcher's one "do the right thing" action

Operator decision (2026-10-06): the switcher gets a Tab action **recover** that
runs the steps the recovery report computes for that row, so the two can't
disagree. Resume and reboot stay as separate entries.

**Files:** `couchcore/ops.go` (declare `recover`: `ExecuteLiveOwner`,
`EffectProcess`, `RowAction: true`; confirmation is per-plan, see below), new
`couchcore/recover_action.go` (`Couch.Recover`), `couchcore/recoverplan.go`
(extract `RecoverRowFor(ctx, address)` — one slot's row through the same
`DeriveRecoverPlan` inputs, ARCH-DRY), `couchcore/actor_actions.go` (offer
`recover` wherever `ActorActions` offers anything), `couchtty/menu_actions.go` +
`couchtty/console_menu.go` (the action, its progress text, the sweep test), `couchcmd/cli.go` (`--recover repo:N`),
socket registration as in Task 8 (`couchcmd/slot_operations.go`); menu code in
`couchtty/menu_actions.go` and `couchtty/console_menu.go`.

- **Offer (cheap):** recover appears on a row whenever `ActorActions` offers any
  actor operation. The menu never runs the report per repaint.
- **Execute (exact):** `Couch.Recover` derives that one row's report decision
  and runs its steps in order: `[resume]`; `[reap, resume]`; `[reboot]`. A hold
  (unknown, conflict, unsafe git, ambiguous threads) refuses with the hold's
  reason and runs nothing.
- **Confirmation follows the steps — a named change to the seam (PQ-3).**
  Today `Operation.Confirmation` (`ops.go:58-60`) is fixed per operation and
  `OperationConfirms` (`operationdispatch.go:81`) answers a bool from it. This adds
  one declared value, `ConfirmByPlan`: the operation confirms exactly when its
  resolved plan contains a destructive step.
  - **Core:** `PrepareRecover(ctx, address) (RecoverPreview, error)` returns
    `{Steps []string; Confirm bool; Text string; Hold string}`. `Confirm` is true iff
    Steps contains `reap` or `reboot`; `Text` is the one sentence ("reap server
    PID N, then resume" / "archive this conversation and start a fresh agent").
    Execution takes the preview's steps plus a `confirmed` arg and re-derives;
    if the re-derived steps differ from the preview it refuses ("the row changed;
    review again"), the same stale-confirmation rule menu.go already applies.
  - **Menu:** Enter on recover calls `PrepareRecover`; `Confirm` false → dispatch
    at once; true → a confirmation frame showing `Text`, then dispatch with
    `confirmed=true`; `Hold` non-empty → the hold as an error notice, no frame.
  - **CLI:** `couch --recover repo:N` prints the preview; if `Confirm` it refuses
    without `--confirm` ("re-run with --confirm to …"). `--confirm` on a plan that
    doesn't need it is accepted (harmless) — unlike fixed-confirmation ops, the
    CLI cannot know the plan when parsing.
  - **`OperationConfirms` / audits:** `OperationConfirms` gains a third answer for
    `ConfirmByPlan` (`confirms, declared, byPlan`), and the CLI's fixed-confirm
    check skips by-plan operations. The operation audit tests enumerate the new
    value; `SlotOperationCommand` is not asked to render `recover` (the report never
    emits it as a step).

- [ ] **Step 1: failing tests:** parked row → recover resumes with no
  confirmation; orphaned row → confirmation text names the pid, then reap and
  resume run in order; unusable non-resumable row → reboot's confirmation; a held
  row (e.g. `unusable-unknown`) → refusal carrying the hold, no effect; the menu
  sweep test sees `recover` exactly where `ActorActions` is non-empty.
- [ ] **Step 2–4:** FAIL → implement → PASS.
- [ ] **Step 5:** commit.

## Chunk 3 — M3: live acceptance and the map

### Task 10: Live acceptance (manual, recorded in the issue Log)

Scripted where possible in `cmd/probes/orphanreap/` is NOT built (YAGNI); run by
hand, unsandboxed, from a throwaway thread:

1. `couch` → start a scratch thread in a scratch repo; note its session name.
2. `rm "$TMPDIR/zellij-$(id -u)/contract_version_1/<session>"` — the ONE socket,
   never the directory.
3. `couch --recover-plan-from-sdlc` → the row shows agent `orphaned` with the
   server pid and steps `reap`, `resume`.
4. `couch --reap <addr> --confirm` → succeeds; `ps` shows no `pair title`,
   `pair term`, `pair wrap` or nvim for that tag.
5. Report again → `parked`, `resume` only; `couch --resume <addr>` → succeeds.

- [ ] Ask the operator to run steps 1–5 live (they own the live Couch); record
  the output in `## Log`.

### Task 11: Atlas, lessons

- [ ] `atlas/couch.md`: the session-state table gains `orphaned` (producer,
  consumers, the one action); the recovery-report section gains `reap`.
- [ ] `workshop/lessons.md`: tests may only `RemoveAll` paths from their own
  `t.TempDir()`; never derive a removal root by walking up (`filepath.Dir`) from
  a path a child process reported; unsandboxed `go test` must run with `TMPDIR`
  pointed at the scratchpad.
- [ ] `sdlc close --issue 399` (full `make -k test` + `go test ./...` first).

## Revisions

### 2026-10-06 — switcher `recover` action (operator)

Reason: the operator asked for one switcher action that resumes when it can and
otherwise does the right recovery. Delta: Task 9b added to M2. `recover` runs the
recovery report's steps for that row (resume / reap→resume / reboot, or refuse on a
hold), and its confirmation follows the steps. Reap is reached from the switcher
through recover rather than as its own menu entry. Resume and reboot stay.

### 2026-10-06 — plan-quality round 1

Reason: change-code's plan review. Delta: PQ-1 — no `ServerProcess`; the existing
`SessionServerIdentity` gains `Socket` and the lowercase duplicate is deleted.
PQ-2 — `SocketState` is tri-state; only ENOENT is `gone`; `unknown` projects to
unresolved. PQ-3 — `ConfirmByPlan` named, with `PrepareRecover`, menu, CLI and
audit handling. Minors: real fake/file names (`ownerWorld`,
`couchcmd/slot_operations.go`, `menu_actions.go`/`console_menu.go`), test strategy
as one line per risky function (fuzz `ParseServerProcesses`), Non-goals section.

### 2026-10-07 — M1 review carries into M2

Reason: M1 boundary review. Delta for M2:
- **A single snapshot is provisional.** A zellij server that is starting appears
  in `ps` before it binds its socket, so for that moment it reads orphaned.
  Reap (Task 7/8) must require the orphan verdict on two snapshots at least a
  short interval apart (or a minimum process age), and re-read the server's
  identity before every signal; a fixture with a just-started server proves it
  is never reaped.
- **One source for the advice.** Startup's refusal currently prints manual steps
  (list the tree, kill descendants, then the server). Once `PlanReap` exists the
  refusal renders `couch --recover <ref>` and the plan's steps, so the text and
  the behaviour cannot drift.

### 2026-10-07 — reap is its own switcher entry too (M2)

Reason: the M2 review flagged drift from the 9b revision ("reap is reached from
the switcher through recover rather than as its own menu entry"). Delta: the
switcher's row menu reads `ActorActions` unfiltered (its own comment forbids
filtering, so offered and declared cannot silently disagree), and `ActorActions`
offers `reap` on an orphaned row, so the switcher shows `[recover, reap]` there.
This matches the operator's decision to keep the specific actions (resume, reboot)
beside recover. recover remains the default, first entry.
