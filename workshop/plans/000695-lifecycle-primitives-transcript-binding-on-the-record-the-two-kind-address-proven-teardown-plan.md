---
collaboration-mode: discussion
---

# Lifecycle primitives Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** land the shared lifecycle primitives on main: the transcript binding stored on the record (P2), the
two-kind `LifecycleAddress`, and proven teardown with `TeardownProof` (P1). pair#684's `Converge` (its remaining
milestones) and pair#694's team lifecycle build on them.

**Architecture:** teardown is proven by three witnesses, one per kind of process. A per-address *lifetime lease*
covers pair's own processes: they hold it with a shared `flock`, and it is free when an exclusive non-blocking
`flock` succeeds. The agent's pty is the second witness (no process left on its controlling tty), and the zellij
session the third (absent twice). Binding stores the agent's exact native ID on the record. All of it switches on
only under Couch (`COUCH_*` env).

**Tech Stack:** Go (`cmd/internal/couchcore`, `threadrecord`, `launcher`, `wrapcmd`, `sessioninventory`), zellij,
`flock(2)`, `lsof`, `ps`.

**Origin:** split from pair#684's plan on the hub's decision (2026-10-18, delivery 51ef8069). The sections below
were #684's §2, its §3 address type, and its M1–M2 tasks, moved here unchanged except for renumbering. #684's
plan now points here for them. Prerequisite: pair#683 (P3, failure persistence) lands first. The design is
approved (operator, 2026-10-18, five changes; later compatibility and invariant revisions recorded in #684's
Revisions).

---

## 1. Core concepts

| Name | Lives in | Status |
|------|----------|--------|
| `TranscriptBinding` | `threadrecord/transcript.go` | new |
| `ThreadRecord` (tolerant, preserving decode) | `couchcore/thread.go`, `threadrecord/record.go` | modified |
| `LifecycleAddress`, `TeamIdentity` | `couchcore/lifecycleaddress.go` | new |
| `TeardownProof`, `Teardown` | `couchcore/teardown.go` | new |
| `LeftRunningOrphan` registry | `couchcore/orphans.go` | new |
| lifetime lease (`lease.Inherit`) | `launcher`/`wrapcmd` (M2 detail) | new |

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `LeaseHolders` | `couchcore` (M2) | new | `lsof -F p` |
| `TTYProcesses` | process-table seam | new | `ps -t` |

The contracts and their reasoning are in §2 and §3; the fakes are in §5.

## 2. The primitives

### P1 — `Teardown(slot) (TeardownProof, error)`: proven teardown of everything but the transcript

A slot's processes are of three kinds, and each gets the witness that fits it. No single witness covers all
three without either missing processes or holding the slot hostage to them.

| Kind | Members | Witness: gone when | Killed by teardown |
|---|---|---|---|
| **pair's own** | the pane's `pair wrap`, the attach client Couch runs as the actor, session-watch, `pair title` | the slot's lifetime lease is free | yes, by exact identity |
| **the agent** | the agent process and everything still in its pty session (its in-flight tool commands) | no process has the agent's pty as its controlling tty | yes, by exact identity |
| **the terminal** | the zellij server and what it hosts (nvim) | both session names absent twice (session_quiescence.go:63-95); nvim reaped by its sidecar pid (the existing `reapNvimSidecars`) | yes (existing paths) |
| *agent-spawned, left the session* | a setsid'd runner, a daemon, a language server that detached | not required | **no**: named in the proof only |

**The lease (pair's own processes).** The invariant "no pair process of the old tree survives" is a one-step
check on a resource held for each process's lifetime (tl-worker PLANNING rule), not an observation sweep:

- Couch creates `<slot store dir>/lifetime.lease` (one file per slot) and passes its path to the launch as
  `COUCH_LIFETIME_LEASE`, beside the `COUCH_THREAD_SCOPE`/`COUCH_THREAD_TAG` it already sets. Couch never holds
  it open past a probe.
- Each pair process started for the tree opens the path with `O_CLOEXEC` and takes `LOCK_SH` (blocking, 2s
  bound). A pair process passes the fd only to children that are **pair processes** (session-watch, wrap.go:2574;
  the title process), through one helper `lease.Inherit(cmd)` that refuses a non-pair `argv[0]`. The agent, nvim
  and everything they start never see the fd. So nothing an agent runs can hold the slot (hub review,
  2026-10-18: a lease inherited through the agent's pty would reach tests, a background `sdlc merge`, language
  servers and daemons. The sweep would kill them, and one that setsid'd and never exits would wedge teardown).
- The lease is free iff `flock(LOCK_EX|LOCK_NB)` on a fresh open succeeds (released at once). `lsof <lease>`
  names the holders when it isn't: exactly the survivors the audit found outside the parent tree (D1, D2, the
  Oct 14 title pid 17468, session-watch after `Setsid` + `Release`).
- **Plain pair is unchanged (operator constraint, 2026-10-18).** pair must keep working without Couch as a
  light wrapper, including several pair sessions in one directory. The lease, the agent tty publication and
  `lease.Inherit` switch on only when Couch launched pair: `COUCH_LIFETIME_LEASE` and `COUCH_THREAD_TAG` are
  both set. Without them pair opens no lease, publishes no tty and passes no extra fd. The behavior is exactly
  today's. The stored transcript binding (P2) lives only in Couch's record.

**The pty (the agent).** The wrapper starts the agent with `pty.Start` (wrap.go:2925), which makes it a session
leader with the pty slave as its controlling tty. Under Couch only, the wrapper records the agent's pid, start
identity and tty name (`ttysNNN`) in the tag's agent sidecar, beside the agent pid it already publishes for
session-watch. It republishes them on every agent start, including pair's own context reset (Shift+Alt+N). The
agent is gone iff no process has that controlling tty (`ps -t <tty>` through the process-table seam). That set
is the agent plus its in-flight tool commands. Stopping those with the agent is what tearing down an agent
means, and the operator confirmed it by choosing the action. A command that setsid'd leaves both the session
and the tty. It is outside the set, and so it is neither killed nor waited on.

**Named, not killed, and tracked.** Before the kill, teardown snapshots the agent's descendants (ppid chain
from the agent pid). Those not on the agent's tty go into `TeardownProof.LeftRunning` and the success notice
("left running: pid 812 `sdlc merge`"). They never block the proof. Teardown then persists them into the
**orphan registry**, so they stay attributable after the slot's thread is gone:

```go
// couchcore/orphan_registry.go — <couch store dir>/orphans.json
type LeftRunningOrphan struct {
    Origin    LifecycleAddress  // the slot (repo:N) or team hub whose teardown left it
    Thread    ThreadAddress     // the torn-down thread's scope and tag
    Process   ProcessIdentity   // pid + kernel start identity
    Command   string            // bounded argv[0..] line
    StartedAt time.Time
    LeftAt    time.Time
    Operation string            // the teardown's operation nonce
}
```

- `couch --resources` lists them by originating slot ("pair:3 · 1-pair-41 · pid 812 `sdlc merge` · since
  18:21"). This is a new kind beside #583's `OrphanNvim` and #606's detached-root attribution, on the same
  surface.
- `couch --reap-orphans --confirm` reaps them on the operator's confirmation (#583's existing gate), by exact
  identity. A reused pid never matches, so it is never signalled.
- **Removal:** an entry is dropped when a resources sample or a reap observes its identity gone. The registry is
  bounded at 256 entries, evicting the oldest. An evicted live process falls back to #583's unattributed-orphan
  rule: still visible, just no longer named by slot.

**Contract:**

```go
// couchcore/teardown.go
type TeardownProof struct {
    Address     LifecycleAddress
    LeaseFree   time.Time              // the EX|NB acquire that succeeded
    AgentTTY    string                 // proven empty; "" when no agent was recorded
    Sessions    []string               // names proven absent twice
    Observed    SlotSessionObservation // ObserveSlotSessions after the kill: Absent == true
    LeftRunning []LeftRunningOrphan    // agent-spawned, outside its session: named and registered, never killed by teardown
}
// Teardown is idempotent: on an already-down slot it costs one flock, one ps,
// one ObserveSlotSessions and one list-sessions, and returns a proof.
func (c *Couch) Teardown(ctx context.Context, addr LifecycleAddress, op OperationRef) (TeardownProof, error)
```

Steps, bounded at 10s total (ARCH-CONSTRAINTS):

1. **Preserve** (only if a session is live): run the park capture (scrollback) and copy the ledger's current
   native ID onto the record (P2's write point). Snapshot the agent's descendants for `LeftRunning`. An open park
   transaction is folded first: a matching success completion with the owner proven dead finalizes it
   (park_recovery.go:201, #683). Otherwise its cleanup runs.
2. **Stop:** quit the zellij session (existing `TriggerBoundQuit`), kill the recorded server and its descendants
   (`PrepareConfirmedTerminal`, recovery_retirement_os.go:22), SIGTERM the slot's actors from the registry.
3. **Sweep**, at most 3 rounds, each SIGTERM by exact identity, then 3s, then SIGKILL, then 2s
   (`awaitExactProcessExit`, detach.go:213), applied to two sets:
   - the lease holders (`LeaseHolders`, the lsof seam);
   - the processes on the agent's tty.
4. **Prove:** all four must hold:
   - the lease is free;
   - the agent's tty is empty;
   - both session names are absent twice;
   - `ObserveSlotSessions(slot).Absent` (slotsessions.go:149). It stays as a second witness for the registry, and
     for trees started before the lease and tty records existed (they're proven as today, by pid and session).
5. **Clean debris** only after proof: sidecars, the wrapper and notify sockets of the now-dead wrapper pids (D1's
   sockets), stale layouts. Debris errors are recorded in the proof, not ignored (lifecycle.go:387-420).

Any step that can't reach its proof returns `ErrTeardownUnproven{Phase, Holders, Cause}` and nothing else runs.
`Holders` lists only pair processes and processes on the agent's tty. An agent's daemon can never appear there.

**Structural enforcement (ARCH-ORDER):** `ArchiveThread`, `archiveThread`, `ReplaceThreadExpected` and
`replaceSlotCurrent` take a `TeardownProof` argument whose `Slot` must match. The Fresh exemption
(threadstore_replace.go:11-16) is deleted (D5). A proof is used within the same slot-operation admission
(the existing per-slot serialization in `prepareSlotOperation`), so it can't go stale between proof and use.

### P2 — `Bind(record) (TranscriptBinding, error)`: bind by the exact ID stored on the record

```go
// couchcore/thread.go — new field on ThreadRecord
Transcript *TranscriptBinding `json:"transcript,omitempty"`

// threadrecord/transcript.go
type TranscriptBinding struct {
    Agent    string    `json:"agent"`
    NativeID string    `json:"native_id"`
    Path     string    `json:"path"`      // absolute, validated under the agent's root
    BoundAt  time.Time `json:"bound_at"`
    Source   string    `json:"source"`    // "minted" | "ledger" | "legacy-ledger"
}
```

- **Written** at three points only: at launch when the ID is minted (Claude, qoder, grok: createflow.go:792-806);
  in Teardown's preserve step (copies the ledger's root ID, covering Codex's session-watch binding, D11); and once
  for a legacy record with no field, by the first Resume that resolves it from the ledger (`Source:
  legacy-ledger`, shown in the row detail).
- **Read** by every Resume: `stat(Path)`, check the basename carries `NativeID`, launch with that ID. No ledger
  query, no directory listing, no 10,000-entry cap (D9, D10, D12). A missing file is "Start fresh, transcript
  missing", never a scan.
- **Untrusted input (ARCH-SECURE):** the record is decoded with strictjson; `Path` must be absolute, clean, under
  the agent's transcript root, and its basename must contain `NativeID`; otherwise the binding is invalid and the
  row says so (Start fresh, reason "invalid stored binding").
- **Compatibility:** additive, per §2a (operator, 2026-10-18). An older binary ignores the field and keeps it.
- **After pair's own context reset (settled by the hub, 2026-10-18).** Shift+Alt+N changes the agent's
  conversation without Couch. So `Bind` makes **one exact read of the tag's own ledger file** (its known path,
  no listing). If that file names a newer root than the record's `BoundAt`, Bind adopts it and stores it
  (`Source: ledger`), so Resume opens the conversation the operator was in. If the file is missing or
  unreadable, Bind falls back to the stored binding and puts a note on the row ("ledger unreadable; resumed the
  stored binding"). It never scans. The counted test stays: 1 ledger read, 1 transcript stat, 0 listings.
- **Open question for the drill:** whether Claude's or Codex's native ID changes across a resume. Teardown
  re-copies the ledger's root ID every time, so a change is picked up; the drill asserts the ID it observes.

### 2a. Record compatibility: additive, not refusal (operator, 2026-10-18)

Persisted records evolve additively. A binary that doesn't know a field ignores it and keeps working. This
replaces the earlier "an older binary refuses M1's records" (decision b).

- **Tolerant and preserving.** Tolerating unknown fields isn't enough. The direct-store operations
  (`publish-description`, `alias`, `reconcile`, `request-continuation`, `provision-workspace`, ops.go:160-300)
  run in whichever `couch` CLI build a slot invokes, and they read, modify and rewrite the thread record. An
  older writer that drops what it doesn't know would silently erase the binding. So each decoder below accepts
  unknown members and **carries them through a rewrite** (an `Unknown map[string]json.RawMessage` captured on
  decode and re-emitted on encode). Types, required fields and duplicate keys stay strict. Only unknown members
  are tolerated.
- **Records in scope:**

  | Record | Today | Change | Lands in |
  |---|---|---|---|
  | `ThreadRecord` (`threadrecord/record.go`, `couchcore/thread.go`) | strictjson | tolerant + preserving | M1, before the new field |
  | agent sidecar (`launcher/markers.go`) | strictjson | tolerant + preserving (a plain pair build reads a Couch-written sidecar with the tty field) | M2, before the tty field |
  | orphan registry (`orphans.json`) | new | tolerant + preserving from birth | M2 |
  | failure store (`launchfailure_store.go`) | strictjson | tolerant + preserving, in #683 (which widens it first) | #683 |
  | lease path | an env var, not a record | none | — |

- **Rollout limit, stated plainly:** tolerance helps only builds that have it. A build from before M1 still
  refuses a record carrying the new field, and fails closed, as today. The first tolerant build ships with the
  usual `:0` rebuild, so on this one machine the window is the rebuild itself.
- **No version bump is needed: no new field is a guard an old reader must honour.** Each field, ignored by an
  old reader, degrades to today's behavior:
  - an ignored binding means the old reader derives the ID from the ledger, as it does now;
  - an ignored tty means the old reader's teardown doesn't sweep the tty, as now;
  - an ignored failure or orphan entry is display only.

  The operator's unsafe example (an old reader archiving a slot that the new binary is mid-teardown on) can't
  arise from a field. Teardown has no persisted in-progress state; it runs inside the singleton Couch's
  per-slot admission. Archive and every lifecycle mutation are live-owner operations that run in that same
  process (ops.go `ExecuteLiveOwner`), never in a CLI build. If a later milestone ever persists a teardown phase
  that other readers must respect, that record gets a version bump then, with the reason written here.
- **Tests (pure):** `TestThreadRecordPreservesUnknownFieldsThroughRewrite` decodes a record with an unknown member,
  changes `PublishedSummary`, re-encodes, and expects the member byte-equal. It also checks that a type mismatch
  and a duplicate key are still refused. The same shape applies to the sidecar and the orphan registry.

### P3 — `RecordFailure`: every lifecycle failure persists its reason (implemented by #683, before this issue's code)

- **One choke point:** `DispatchOperation` (operationdispatch.go:44) is the single entry for CLI, switcher and
  reattach operations (run.go:616, 632, 982; recover_action.go:176). A wrapper there persists any returned error.
  No per-site calls, so a new path can't forget (ARCH-PURPOSE: the class, not the instances).
- **Store:** widen the existing `LaunchFailure` store (couchcore/launchfailure*.go) into `LifecycleFailure`:
  add `Phase` (observe, preserve, stop, sweep, prove, bind, launch, commit), `Holders []ProcessIdentityLine` and
  `Outcome` (failed, unproven, refused); allow a slot-level entry with no tag. Raise the ring from 16 to 64
  entries, oldest evicted (ARCH-FUNERAL: bounded, one file, ~64 × 8 KiB worst case). The existing launch-stage
  captures keep writing into the same store.
- **Visible:** the switcher row and `couch --show` read the latest failure for the slot (store cached by mtime,
  one small read per refresh). A later success for the same slot supersedes it. CLI receipts (D16) carry the
  failure's nonce so `couch --show` finds it after the receipt expires.
- **Park early returns** (§3 of the audit, #683) are covered twice: their errors reach the dispatcher, and the
  next Teardown folds or re-runs the park.

## 3. The lifecycle address

**The address has two kinds (introduce-team, 2026-10-18).** Every primitive and Converge take one
`LifecycleAddress`, never a bare `SlotIdentity`:

```go
// couchcore/lifecycleaddress.go (pure)
type AddressKind string // "slot" | "team"
type TeamIdentity struct {
    CheckoutRoot string // physical worktree root of the operations checkout
    TeamDir string // clean relative path to the team folder, e.g. teams/devinfra (holds the committed .hub)
    Name    string // derived: filepath.Base(TeamDir); the hub's name
}
type LifecycleAddress struct {
    Kind AddressKind
    Slot SlotIdentity // Kind == slot: a worker, repo:N
    Team TeamIdentity // Kind == team: a team hub; several may share one CheckoutRoot
}
func (a LifecycleAddress) Validate() error // exactly one arm set, matching Kind
func (a LifecycleAddress) Key() string     // stable key: store dir, lease path, orphan origin, failure records
```

Both kinds go through the same three outcomes and the same three primitives. The lease and the transcript
binding apply to a hub's agent exactly as to a worker's. Only the guards differ: a slot validates its worktree
layout and may carry sdlc claims, and a team validates `TeamDir` (a folder under `CheckoutRoot` with a committed
`.hub`) and never carries an sdlc claim. Exhaustive switches on `Kind` live in `Validate` and `Key`
only; `PlanConverge` (#684) never sees the kind. This issue builds and tests the type with both arms
(`TestLifecycleAddressValidate`, `TestLifecycleAddressKeyDistinctPerTeam`: two teams in one ops checkout get
distinct keys). It wires only the slot arm, plus today's hub as it is currently addressed. Addressing hubs by
team folder, startup with team folders, and the switcher are pair#694, built on this type.

## 4. Operating envelope (ARCH-CONSTRAINTS)

- **Teardown:** an operator action with a 10s budget. Each sweep round (at most 3) runs one `lsof` on the lease
  and one `ps -t` on the agent's tty. lsof is estimated at 0.1–0.5s, to be measured in M2. Exceeding the budget
  means unproven: recorded, then stop.
- **Resume bind:** one read of the tag's ledger file and one `stat`. Counted test: with 8 and 16 transcripts in a
  fake root, `Bind` does 1 read, 1 stat and 0 listings (it rejects any scan).
- **Lease probe:** acquiring and releasing a local `flock` takes microseconds and never blocks (`LOCK_NB`).
- **Refresh path:** unchanged. No flock, lsof or ps runs on it.

## 5. Fakes and conformance (ARCH-MOCK)

- `LeaseHolders` (wraps `lsof -F p <path>`) and `TTYProcesses` (wraps `ps -t <tty>`, on the existing
  process-table seam) are stateful fakes keyed by path and tty. Their holders exit on SIGTERM, on SIGKILL only,
  or never. Tests use a real `flock` on a temp file, held by real `sleep` children via `ExtraFiles` (cheap,
  exact).
- zellij, the process table and agent transcript roots reuse the existing fakes, extended rather than duplicated.
- Live conformance: one real-pty test in M2 (a shell on a pty starts a setsid'd child; the tty empties while the
  child survives and is named). #684's drill covers the rest later.

## 6. Lifetimes (ARCH-FUNERAL)

- `lifetime.lease`: one empty file per slot, created by the first launch, removed with the slot's store dir.
- Agent tty in the agent sidecar: one field, removed with the sidecar at debris cleanup.
- `ThreadRecord.Transcript`: one per record, archived with it.
- Failure ring: 64 entries, one file (#683).
- Orphan registry: `orphans.json`, at most 256 entries. An entry is dropped when its process is observed gone, or
  evicted oldest-first at the bound.

## 7. Milestones

Each `Mx` is a review boundary (`sdlc milestone-close`). The code starts from main after #683 lands.

- [ ] M1 — Store the transcript binding on the record; Resume binds by one stat (P2). Add the two-kind
  `LifecycleAddress`. Covers #684 audit D9, D10, D12.
- [ ] M2 — The lifetime lease, the agent tty witness, and `Teardown` with `TeardownProof`; archive and replace
  require a proof (P1); the orphan registry. Covers audit D1–D8 and #683's finalization half.

### M1 tasks

**Files:**
- Create `threadrecord/transcript.go` and its test.
- Modify `couchcore/thread.go`, `threadrecord/record.go` (validation), `launcher/createflow.go` (minted IDs
  reported back), `couchcore/resume.go`, `couchcore/recovery_prepared.go` and `couchcore/relaunch.go`.
- `sessioninventory/native_selector*.go`: Resume stops calling them.

Steps:

- [ ] **Test (red):** `TestLifecycleAddressValidate`, `TestLifecycleAddressKeyDistinctPerTeam` (§3). Then
  create `couchcore/lifecycleaddress.go` with both arms; nothing calls the team arm yet.
- [ ] **Test (red):** `TestThreadRecordPreservesUnknownFieldsThroughRewrite` (§2a). Then make the
  `ThreadRecord` decoder tolerant and preserving, before the new field lands.
- [ ] **Test (red):** `TestTranscriptBindingValidate`. A binding is accepted only when its path is absolute,
  clean and under the agent's root, and its basename contains the ID. A relative path, a `..` path, a foreign
  root and a mismatched ID are all rejected. Pure.
- [ ] **Test (red):** `TestResumeBindsByOneStat`. Use a fake root with 8, then 16, Codex and Claude
  transcripts. `Bind` performs exactly 1 ledger-file read, 1 stat and 0 listings. A missing file yields
  `ErrNoTranscript{Reason: missing}`.
- [ ] **Test (red):** `TestBindAdoptsNewerLedgerRootAfterReset`. The record names conversation A and the tag's
  ledger file names a newer root B: Bind returns B, stores it, and performs 1 ledger read, 1 stat and 0
  listings. A second case uses a missing ledger file, and a third an unreadable one. Both return A with the
  row note and still perform 0 listings.
- [ ] **Test (red):** `TestLegacyRecordBindsFromLedgerOnceThenStores`. The first Resume resolves through the
  ledger and writes `Source: legacy-ledger`. The second performs no ledger query.
- [ ] Add the field, its validation and its clone. Write it at a minted launch. Resume, relaunch and recovery
  Resume read only the record; relaunch's pre-park derivation is deleted.
- [ ] Until M2's Teardown exists, the park path copies the ledger ID onto the record at `FinalizePark`. That is
  one call, which moves into Teardown's preserve step in M2.
- [ ] Commit `#695 M1: …`.

### M2 tasks (detailed when M2 starts; a `## Revisions` entry records the detail)

- `launcher/lease` package: pure path validation, plus thin `Hold`, `Inherit(cmd)` and `ProbeFree`. Tests use
  real `flock` and `sleep` children:
  - a pair-child that holds the fd after its parent exits leaves the lease not free;
  - once all exit, the lease is free;
  - `Inherit` refuses a non-pair `argv[0]`.
- Every pair process started for a Couch tree holds the lease, and none passes it to the agent or nvim. A test
  enumerates the spawn sites in `wrapcmd` and fails when one passes the fd to a non-pair binary. The rule goes
  into `pkg/golint` if it generalizes.
- `TestPlainPairUnchangedWithoutCouch`. With no `COUCH_*` env, two pair sessions start in one directory. Each
  must open no lease, write a sidecar without a tty field, pass no extra fd to any spawn, and behave as before.
  A second case sets `COUCH_LIFETIME_LEASE` without `COUCH_THREAD_TAG` and expects the same: both are required.
- `couchcore/orphan_registry.go` (pure merge, prune and bound, plus a thin store).
  `TestTeardownRegistersLeftRunning`, `TestResourcesNamesOrphansByOriginSlot`,
  `TestReapOrphanRefusesReusedPid`, `TestOrphanRegistryBound256`.
- The agent sidecar decoder becomes tolerant and preserving (§2a) before the tty field;
  `TestAgentSidecarPreservesUnknownFields`.
- Under Couch, the wrapper records the agent's tty in the agent sidecar. A real-pty test checks two things: a shell's
  foreground child on the pty is in the tty set, and a setsid'd grandchild is outside it and named in
  `LeftRunning`.
- `couchcore/teardown.go`, with the `LeaseHolders` and `TTYProcesses` seams and their fakes.
  `TestTeardownSweepsReparentedHolders` uses two holders: one survives SIGTERM, and one survives the round limit,
  which gives `ErrTeardownUnproven` naming it. `TestTeardownNeverKillsDetachedAgentChildren`.
- Archive and replace signatures take `TeardownProof`. `TestArchiveRefusesWithoutProof`,
  `TestFreshReplaceRequiresProof`.
- `TestArchiveRefusesOpenPark` and `TestConvergeRefusesMovedRevision` (stale queued request).
- `TestTeardownFoldsCompletedPark` uses #683's retained shape: a success completion, the owner dead, the session
  absent.

## Revisions

- 2026-10-19 — Zellij listing omission is unknown, not absent (pair:2, #692, delivery 0e4c213f). zellij 0.45.1
  drops a session from its listing when the ConnStatus reply is misdelivered (upstream issue 5270). So "both session
  names absent" (§2, the terminal row) is not proof by itself. When the listing omits a session whose exact server
  is independently live (the OS process scan via `SessionServers`, with no native query), the terminal witness is
  **unknown**. #692 classifies this as `launcher.ErrListingOmittedLiveServer`, with `OSRuntime.liveSessionServers`
  as the shared evidence (also used by `SessionBlocksReuse`). Teardown reuses that classification, with no second
  one, and maps it to `ErrTeardownUnproven` with the server named. It refuses and never retries, and the
  listing's absence counts only when the process scan agrees. Test (M2): `TestTeardownUnprovenWhenListingOmitsLiveServer`.
  This depends on #692 landing first. If #695's M2 starts before then, that test waits for #692.
- 2026-10-19 — #683 landed (PR #452, merge d964bbc11); merged into this branch. Its final shapes are folded in here.
  Where §2's P3 text differs, these govern:
  - **Failure log is completion-only.** There is no write-time supersession and no `superseded_at` production, and
    a success writes nothing. A failure is hidden when `LastLifecycleSuccessAt(record)`
    (couchcore/lifecyclefailure_projection.go:130) is strictly later than it. That function is the single seam,
    and today it counts `VerifiedPark.ParkedAt` and a committed live incarnation's `StartedAt`. **M2 adds the
    proven-teardown event:** `TeardownProof.ProvenAt` is persisted with the archived record (or the parked record for
    Down), and `LastLifecycleSuccessAt` takes it into account. This is so that an unproven teardown followed by a
    proven retry no longer shows on the emptied slot. Test (M2): `TestProvenTeardownHidesEarlierUnprovenFailure`.
  - **Teardown errors implement `LifecycleFailureDetails`** (couchcore/lifecyclefailure.go:16: `FailurePhase`,
    `FailureHolders`, `FailureOutcome`). The holders are #683's `ProcessIdentityLine{Process, Kind}`, and the
    `Kind` comes from #683's catalogue (pair, pair-wrap, pair-launch-helper, zellij, nvim). M2 extends that catalogue
    with `agent` and `session-watch` rather than defining a second holder type. `ErrTeardownUnproven` is the
    `unproven` outcome with phase `prove`, and `LeftRunning` is evidence only. There are no capture sites in
    Teardown: the dispatcher's completion recorder reads the interface. Test (M2):
    `TestTeardownUnprovenRecordsPhaseAndHolders`, which goes through the real recorder with the store fake.
  - **The slot arm of `Key()` equals #683's slot key.** `LifecycleFailureSlotKey` is `slot:` plus the canonical
    physical starting path (lifecyclefailure.go:30). `LifecycleAddress{Kind: slot}.Key()` returns exactly that,
    which covers `:0` too: #683's key needs no `Number > 0`, so the slot arm keys by the slot's physical root. File
    names (lease, store dir) are a digest of `Key()`, never the raw path. `lifecycleFailureTargetKey` delegates to
    `Key()` for a slot target in M1, and `TestLifecycleAddressSlotKeyMatchesFailureSlotKey` pins the equality. The
    team arm (`team/<scope>/<TeamDir>`) has no failure-row matcher yet; pair#694 M3 adds it to
    `lifecycleFailureRowTargets`.
- 2026-10-19 — Plan-quality round 1 (PQ-1 and five minors). Where these differ from §1–§7, they govern:
  - **PQ-1, one shared preserving codec (ARCH-DRY).** #683 wrote the strict-known, preserve-unknown mechanism by
    hand: `failureObject` and `failureEncodeObject` (couchcore/lifecyclefailure_codec.go:37,62), plus
    `cloneFailureUnknown` and `mergeFailureUnknown` (lifecyclefailure.go:80,90). It is not repeated per decoder.
    **M1, step 1** extracts it into the leaf package `strictjson`, so that `launcher` (the agent sidecar) can import it
    too, which it cannot do from couchcore:
    - `strictjson.SplitObject(raw, known, nullable) (owned, unknown map[string]json.RawMessage, err)`: duplicate keys
      are rejected recursively first, then null is checked for known members.
    - `strictjson.EncodeObject(owned, unknown, known) ([]byte, error)`: unknown values are appended verbatim, and the
      known names win.
    - `strictjson.CloneRaw` and `strictjson.MergeRaw`.

    #683's failure codec moves onto these with no byte change; its existing round-trip tests are the regression
    guard. `ThreadRecord` (M1), the agent sidecar and `orphans.json` (M2) then use only these functions. The helper is
    tested once, in `strictjson/object_test.go`: `TestSplitEncodeObjectPreservesUnknownBytes` (whitespace, escapes,
    large integers, nested objects) and `FuzzSplitObject` (duplicate keys at every depth, type mismatches, null
    known members). The per-record tests shrink to one wiring check each.
  - **The `TeardownProof` shape.** It is `TeardownProof{Address LifecycleAddress; ProvenAt time.Time; Witnesses
    ...}`. Archive and replace match the proof on `Address.Key()`, never on a slot field. `ProvenAt` is persisted
    with the archived (or parked) record, and it is what `LastLifecycleSuccessAt` reads.
  - **The archive funnel.** The proof parameter goes on the private `archiveThread` (threadstore.go:1408), so
    `ArchiveThread`, `ArchiveThreadExpected` and `ArchiveThreadExpectedBytes` (:1389-1403) all require one
    structurally. The exact-bytes variant, which archives an undecodable record, needs a proof too: teardown proves
    the slot's processes gone from the address, not from the record. Test: `TestEveryArchiveVariantRequiresProof`,
    a table over the three public entry points.
  - **Out of scope.** `TestConvergeRefusesMovedRevision` moves to #684, since Converge is #684's. M2 keeps the archive
    and replace analogue it actually changes: `TestArchiveRefusesProofForMovedRevision`, where a proof taken at
    revision N refuses a record now at revision M.
  - **The key shapes.** `Key()` is `slot:<canonical physical root>` for a slot, which equals #683's
    `LifecycleFailureSlotKey`. For a team it is `team:<scope>/<TeamDir>`, where `scope` is
    `launcher.ResolveRepoScope(CheckoutRoot)` (scope.go:23), the same scope that identifies a checkout everywhere
    else. So two checkouts that hold the same `TeamDir` get distinct keys (`TestLifecycleAddressKeyDistinctPerCheckout`).
    `Validate()` resolves the scope once; `Key()` is pure over the resolved value, so `TeamIdentity` carries a
    `Scope` field filled by `Validate`. Both arms use a `:` separator, so the earlier `team/` spelling is dropped.
  - **Bind's write.** When Bind adopts a newer ledger root, the record write goes through the store's existing
    revision-checked atomic write (`writeStoreAtomicLockedChecked`, the same path every record write uses), inside
    the slot's per-slot admission. A concurrent direct-store rewrite either lands first, in which case the
    revision moves, Bind re-reads and the adoption is retried once, or it lands after and preserves the field (§2a).
    Test (M1): `TestBindAdoptionSurvivesConcurrentRewrite`.
- 2026-10-19 — PQ-7 (contract shape). A value-receiver `Validate()` can't fill a field, so a resolved scope comes
  from constructors, and the §3 signatures become:

  ```go
  type TeamIdentity struct { CheckoutRoot, TeamDir, Name, Scope string } // Scope: launcher.ResolveRepoScope(CheckoutRoot).Key
  func NewSlotAddress(slot SlotIdentity, physicalRoot string) (LifecycleAddress, error) // canonical root; the Key's slot arm
  func NewTeamAddress(checkoutRoot, teamDir string) (LifecycleAddress, error)           // resolves Scope and Name, checks .hub
  func (a LifecycleAddress) Validate() error // pure shape check: exactly one arm, matching Kind, resolved fields present
  func (a LifecycleAddress) Key() string     // pure: "slot:<root>" | "team:<scope>/<TeamDir>"
  ```

  The constructors are the IO seam (scope resolution, `.hub` stat), and `Validate` and `Key` stay pure. The
  slot arm carries the physical root it was built from, so `Key()` needs no lookup. `SlotIdentity` keeps its own
  fields, and the address adds `Root string` beside it. The slot root is for keying only: a `:0` primary is
  addressed by its root, as #683's key already does.
