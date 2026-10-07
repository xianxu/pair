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
| `ServerProcess` (pid, start identity, session, socket path) | `cmd/internal/launcher/session_servers.go` | new |
| `ParseServerProcesses` (ps text → `[]ServerProcess`) | `cmd/internal/launcher/session_servers.go` | new |
| `ClassifyServers` (servers + socket-exists → live/orphaned per session) | `cmd/internal/launcher/session_servers.go` | new |
| `SessionOwnerOrphaned` | `cmd/internal/launcher/session_owner.go` | modified (new state) |
| `SessionOrphaned` + `SessionObservation.Orphan` | `cmd/internal/couchcore/sessionevidence.go` | modified |
| `ProjectSessionPresence` | `cmd/internal/couchcore/sessionevidence.go` | modified (orphan input) |
| `ReasonOrphanedServer` | `cmd/internal/couchcore/threadreason.go` | new |
| `ClassifyThread` | `cmd/internal/couchcore/actionableinventory.go` | modified |
| `ActorActions` | `cmd/internal/couchcore/actor_actions.go` | modified (offers `reap`) |
| `AgentOrphaned`, `OfferReap`, rule-A `reap`→`resume` | `cmd/internal/couchcore/recoverplan.go` | modified |
| `ResumeOrphanedServer` diagnostic code | `cmd/internal/couchcore/resume.go` | new |
| `ReapPlan` (tree snapshot → ordered signal plan) | `cmd/internal/launcher/session_reap.go` | new |

- **ServerProcess** — what `ps` says about one zellij server. Today
  `sessionServerIdentity` has pid/identity/session but throws the socket path
  away; the path is the third argv field and is the only evidence of orphaning.
  - **DRY rationale:** `zellijServerPIDs` and `isExactZellijServerCommand` already
    parse this argv; they become thin wrappers over `ParseServerProcesses`, so there
    is one argv grammar.
  - **Future extensions:** a socket that exists but refuses connections (stale
    socket file) is a third state; it widens `ClassifyServers`, not its callers.
- **SessionOrphaned** — the session-evidence value "a server for this name is
  alive and unreachable". The comment on `SessionState` forbids a value no
  producer emits; this one has a producer (`SessionPresence` via `ClassifyServers`).
  `SessionObservation` regains a field (the comment says M2 would re-add one with
  its consumer): `Orphan *ServerProcess`, consumed by the diagnostic and by reap.
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

### Task 1: ServerProcess parsing and classification (launcher, pure)

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
	want := []ServerProcess{{PID: 101, Session: "📁1-37", Socket: "/T/zellij-501/contract_version_1/📁1-37"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
}

func TestClassifyServers(t *testing.T) {
	servers := []ServerProcess{{PID: 1, Session: "a", Socket: "/s/a"}, {PID: 2, Session: "b", Socket: "/s/b"}}
	exists := func(p string) bool { return p == "/s/a" }
	got := ClassifyServers(servers, exists)
	if got["a"].Orphaned || !got["b"].Orphaned || got["b"].Server.PID != 2 {
		t.Fatalf("got %+v", got)
	}
}
```

Two servers for one session name classify as ambiguous (neither orphaned nor
live) — a test row for that, mirroring `indexSessionsByName`'s fail-closed rule.

- [ ] **Step 2:** `go test ./cmd/internal/launcher -run 'ParseServer|ClassifyServers'` → FAIL (undefined).
- [ ] **Step 3: implement** `ServerProcess{PID int; Identity string; Session string; Socket string}`,
  `ParseServerProcesses(raw string) []ServerProcess` (exact 3-field argv,
  `filepath.Base(socket) == session`), `ServerState{Server ServerProcess; Orphaned, Ambiguous bool}`,
  `ClassifyServers(servers, exists func(string) bool) map[string]ServerState`.
  Rewrite `zellijServerPIDs`/`isExactZellijServerCommand` on top.
- [ ] **Step 4:** run launcher tests → PASS (including existing quiescence tests).
- [ ] **Step 5:** commit `#399 M1: launcher: parse zellij servers with their socket path`.

### Task 2: ServerSnapshot (launcher, IO) and Probe's orphaned state

**Files:**
- Modify: `cmd/internal/launcher/session_servers.go` (`ServerSnapshot`, OS impl)
- Modify: `cmd/internal/launcher/session_owner.go` (`SessionOwnerOrphaned`)
- Modify: `cmd/internal/launcher/session_owner_os.go` (`Probe`, `SessionOwnerIO`)
- Test: `cmd/internal/launcher/session_owner_os_test.go`

- [ ] **Step 1: failing test** using the existing fake `SessionOwnerIO`, extended
  with `SocketExists(path) bool`:

```go
func TestProbeReportsOrphanedServerInsteadOfAnError(t *testing.T) {
	io := &fakeOwnerIO{servers: []SessionServerIdentity{{PID: 7, Identity: "t7", Session: "📁1-37", Socket: "/gone"}},
		panesErr: errors.New("exit status 1")}
	got, err := SessionOwnerProbe{IO: io}.Probe(ctx, "📁1-37", dataDir, scope, tag)
	if err != nil || got.State != SessionOwnerOrphaned || got.Server.PID != 7 {
		t.Fatalf("got %+v, %v", got, err)
	}
	if io.panesCalls != 0 {
		t.Fatal("an orphan has no socket to ask")
	}
}
```

Plus: socket present + `list-panes` failing still returns the error (only a
missing socket is the orphan state — an unreachable live socket stays a fault).

- [ ] **Step 2:** run → FAIL.
- [ ] **Step 3: implement:** `SessionServerIdentity` gains `Socket`; `Probe`
  checks `SocketExists(before[0].Socket)` before `SessionPanes`; on absence returns
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
	servers := map[string]launcher.ServerState{"📁1-37": {Server: launcher.ServerProcess{PID: 9}, Orphaned: true}}
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
  `SessionObservation.Orphan *launcher.ServerProcess`; `ProjectSessionPresence`
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
- [ ] **Step 2:** FAIL. **Step 3:** `PlanReap(root ServerProcess, table Snapshot) []ReapStep`
  (pure: descendants BFS from the snapshot, children before the server);
  `Reaper{Table ProcessTable; TermWait, KillWait, Poll time.Duration}.Reap(ctx, root)`.
  The OS `ProcessTable` reads `ps -axo pid=,ppid=` once and `procutil.Identity`.
- [ ] **Step 4:** PASS. **Step 5:** commit.

### Task 8: `Couch.Reap` and the operation surface

**Files:**
- Modify: `couchcore/ops.go` (declare `reap`: `ExecuteLiveOwner`, `EffectProcess`,
  `ConfirmRequired`, `RowAction: false`), `couchcore/operationdispatch.go`,
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
