---
collaboration-mode: discussion
---

# Slot lifecycle: one mechanism, three primitives, three outcomes

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every slot lifecycle path lands deterministically in Continue, Resume from disk or Start fresh, through
one mechanism (`Converge`) built from three primitives: proven teardown, binding by the transcript ID stored on
the record, and a persisted failure record.

**Architecture:** Teardown is proven by three witnesses, one per kind of process: a per-slot *lifetime lease*
(a file pair's own processes hold with a shared `flock`; free when an exclusive non-blocking `flock` succeeds),
the agent's pty (no process left on its controlling tty), and the zellij session (absent twice). `Converge(slot, target)` is a pure planner over observed evidence plus a thin executor. Every
path in the audit becomes one `Converge` call, and the operation dispatcher persists every failure.

**Tech Stack:** Go (`cmd/internal/couchcore`, `launcher`, `wrapcmd`, `sessioninventory`), zellij, `flock(2)`,
`lsof`, launchd for the drill.

**Inputs:** the project file (`ops:teams/devinfra/projects/slot-lifecycle/project.md`), its `audit.md` (paths
1–16, deviations D1–D20; this plan cites them by number), targets `slot-continuity` and `slot-recovery`, pair#683.

---

## 1. The model

### Visible states (the target's three, no fourth)

| Visible state | Evidence | The row's one action |
|---|---|---|
| **Continue** | exactly one live actor and one live session for the slot, and the record names its transcript | attach |
| **Resume from disk** | not Continue, and the record's transcript binding resolves by one `stat` | Resume (tear down, then launch resuming that ID) |
| **Start fresh** | not Continue, and no transcript binding resolves | Start fresh (tear down, then launch a new conversation) |

Everything the audit found as an extra state collapses into these (D19): `ReasonUnknown`, needs-relaunch (an
orphaned server), a stale start claim and two live actors (D20) are all "not Continue"; the row shows Resume or
Start fresh by the transcript evidence, and the action's own teardown removes whatever is left. The uncertainty
is shown as the row's detail, never as a state.

A **failure is an annotation, not a state.** When an action fails, the row keeps its outcome and its action, and
shows the persisted reason ("last Resume failed 18:21: teardown unproven: pid 17468 `pair title` holds the slot
lease"). Retrying is the same action. The reason names the exact holder, so the operator can act on it.

### State design table (tl-worker contract)

Actions: attach, Resume, Start fresh, Park (Alt+x), Archive, Relaunch (Alt+n: park then Resume).

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| one live tree, binding stored | Continue | attach | none |
| live tree, record has no binding yet (Codex before its first turn) | Continue | attach | none |
| tree gone, binding stored, transcript file present | Resume from disk | Resume | the live terminal state only |
| tree gone, binding stored, transcript file missing | Start fresh (detail: "transcript `<id>` missing at `<path>`") | Start fresh | the conversation (target: accepted) |
| tree gone, no binding on record (legacy) | Resume if the ledger resolves it once (then stored), else Start fresh | Resume / Start fresh | as above |
| pair orphans hold the lease (title, session-watch, attach client) | Resume / Start fresh | the action's teardown kills them by identity | none |
| processes left on the agent's tty (its in-flight tool commands) | Resume / Start fresh | the action's teardown kills them with the agent (the operator confirmed stopping the agent) | that command's in-flight work |
| agent-spawned process that left the agent's session (a setsid'd runner, a daemon) | unaffected: named in the proof and the orphan registry, never killed by teardown, never required gone | the operator may reap it from `couch --resources` / `--reap-orphans --confirm` | none |
| two live actors or sessions | Resume / Start fresh, detail names both | the action tears both down | live terminal state |
| park awaiting_completion, success completion present (#683) | Resume from disk | Resume (teardown folds the completion first) | none |
| park awaiting_completion, cleanup never ran | Resume from disk | Resume (teardown re-runs cleanup, needs proof) | scrollback tail if the capture never ran |
| teardown unproven (a lease holder or tty process survives SIGKILL, lsof or ps fails, zellij unanswering) | unchanged, annotated with the failure | same action (retry) after the operator removes the named holder | nothing is touched: no archive, no launch |
| launch fails after proven teardown | unchanged, annotated | same action; the partial launch is itself torn down with proof before the failure is recorded | none beyond the failed attempt |
| Couch dies mid-Converge | re-observed on restart: whatever the evidence says | the row's action | none: every step is idempotent and gated on fresh evidence |

**Covered:** every path in the audit. **Not covered, on purpose:** transients from developing Couch while using
it (operator's call); D17 crash-record gaps and D18 helper debug logs (not lifecycle actions; separate items if
wanted); Inferred tail-match recovery stays only as the legacy fallback for records without a binding.

## 2. The three primitives

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
    Origin    SlotIdentity      // the slot (repo:N) whose teardown left it
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
    Slot        SlotIdentity
    LeaseFree   time.Time              // the EX|NB acquire that succeeded
    AgentTTY    string                 // proven empty; "" when no agent was recorded
    Sessions    []string               // names proven absent twice
    Observed    SlotSessionObservation // ObserveSlotSessions after the kill: Absent == true
    LeftRunning []LeftRunningOrphan    // agent-spawned, outside its session: named and registered, never killed by teardown
}
// Teardown is idempotent: on an already-down slot it costs one flock, one ps,
// one ObserveSlotSessions and one list-sessions, and returns a proof.
func (c *Couch) Teardown(ctx context.Context, slot SlotIdentity, op OperationRef) (TeardownProof, error)
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
- **Compatibility:** adding the field is forward-compatible only (strictjson rejects unknown fields;
  thread.go:86-91 precedent). A binary older than M1 refuses records M1 writes. Accepted, as for #198.
- **After pair's own context reset (settled by the hub, 2026-10-18).** Shift+Alt+N changes the agent's
  conversation without Couch. So `Bind` makes **one exact read of the tag's own ledger file** (its known path,
  no listing). If that file names a newer root than the record's `BoundAt`, Bind adopts it and stores it
  (`Source: ledger`), so Resume opens the conversation the operator was in. If the file is missing or
  unreadable, Bind falls back to the stored binding and puts a note on the row ("ledger unreadable; resumed the
  stored binding"). It never scans. The counted test stays: 1 ledger read, 1 transcript stat, 0 listings.
- **Open question for the drill:** whether Claude's or Codex's native ID changes across a resume. Teardown
  re-copies the ledger's root ID every time, so a change is picked up; the drill asserts the ID it observes.

### P3 — `RecordFailure`: every lifecycle failure persists its reason (implemented by #683, before #684's code)

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

## 3. The mechanism: `Converge`

```go
// couchcore/converge.go (pure)
type Target int // TargetContinue, TargetResume, TargetFresh, TargetDown
type ConvergeInput struct { Target Target; Evidence SlotEvidence; Agent string }
type Step int // StepAttach, StepTeardown, StepBind, StepMint, StepLaunch, StepArchive, StepCommit
func PlanConverge(in ConvergeInput) ([]Step, error)

// couchcore/converge_exec.go (thin IO shell)
func (c *Couch) Converge(ctx context.Context, slot SlotIdentity, in ConvergeRequest) (ConvergeResult, error)
```

| Target | Steps |
|---|---|
| Continue | exactly one live actor+session and no lease anomaly → Attach; otherwise refuse (`not Continue: <evidence>`) and re-project. Continue never tears down. |
| Resume | Teardown → Bind (stored ID; fail → `ErrNoTranscript`, no fallback to Fresh) → Launch(resume ID) → Commit (same record, new incarnation) |
| Fresh | Teardown → Mint (new ID where the agent accepts one) → Launch(fresh) → Archive(old, proof) → Commit(new current) |
| Down | Teardown → Commit (record parked; outcome Resume from disk) |

A launch failure after a proven teardown runs Teardown again on the partial tree before the failure is recorded,
so a failed attempt never leaves a second actor. **There is no automatic Resume→Fresh fallback**: a failed Resume
records its reason, and Start fresh is the row's action only when the evidence says the transcript is gone. The
operator can always pick Start fresh from the menu (the target's "always reachable").

**Events the caller can't block (ARCH-ORDER):** Couch dying mid-Converge (every step re-gates on fresh evidence;
the lease makes a half-launched tree visible as "held"); a second operation on the same slot (queued by the
existing per-slot admission); a park completion arriving late (Teardown's fold reads it); a holder respawning
during the sweep (the round limit bounds it, then unproven). The one most likely to be mishandled: **a timeout or
a failed probe read as absence** — every probe returns live/absent/unknown, and unknown is `ErrTeardownUnproven`.

**Concurrent extent:** nothing Converge spawns outlives it except the launched tree, which is owned by the lease.

## 4. The sixteen paths

| # | Path | Becomes | Deletes / changes |
|---|---|---|---|
| 1 | Park (Alt+x, CLI) | `Converge(Down)` | the title no-op stage (D2); launcher-only death proof (D3) |
| 2 | Relaunch (Alt+n, switcher, CLI) | `Converge(Resume)` | the double native-ID derivation (relaunch.go:193, resume.go:483); `ParkedNotResumed` limbo |
| 3 | Cold resume | `Converge(Resume)` | ledger lookups and the provisional-ID walk (query.go:552-607) on this path |
| 4 | Warm attach, Couch restart reattach | `Converge(Continue)` | attaching beside a second client (D13): attach refuses when another client is attached |
| 5 | Recover-park | `Converge(Down)` | retire without cleanup (park_recovery.go:229-236, D8) |
| 6 | Recovery: Resume from disk | `Converge(Resume)` | the Codex capped walk (D10) |
| 7 | Recovery: Start fresh (#684) | `Converge(Fresh)` | the waivers (recovery_prepared.go:160-192, 509-516, D4) |
| 8 | Start fresh from the start form | `Converge(Fresh)` | its private absence gate (now Teardown's proof on an empty slot) |
| 9 | Archive | `Converge(Down)` + archive(proof) | `prepareRetirement`'s partial proof (D5) and its stale comment |
| 10 | Store archive / replace | require `TeardownProof` | the Fresh exemption |
| 11 | Switch agent | `Converge(Fresh, agent)` | "agent may survive" (menu.go:1493, D3) |
| 12 | Reload-context (Shift+Alt+N) | **unchanged** (operator, 2026-10-18): pair's own fast context reset, not a Couch lifecycle path | — (under Couch the wrapper republishes the agent's pid and tty; the record's binding is refreshed at the next teardown) |
| 13 | Detach | unchanged: the slot stays Continue | — |
| 14 | Stop | await the actor's exact exit before `Forget`; Unknown refuses (D6) | the no-wait `Forget` |
| 15 | Park abandon, reboot archive-only | deleted; covered by Down and Archive | both bookkeeping-only paths |
| 16 | Startup classification | observe only; one of three outcomes, count-aware (D19, D20), last failure attached | the Unknown→Fresh mapping, `ContinuityStarting` as a state |

## 5. Operating envelope (ARCH-CONSTRAINTS)

- **Teardown:** an operator action with a 10s budget. Each sweep round (at most 3) runs one `lsof` on the lease
  and one `ps -t` on the agent's tty. lsof is estimated at 0.1–0.5s, to be measured in M2. Exceeding the budget
  means unproven: recorded, then stop.
- **Resume bind:** one `stat`. Counted test: with 8 and 16 transcripts in a fake root, `Bind` does 1 stat and 0
  listings (rejects any scan).
- **Classification refresh:** unchanged, plus one mtime check of the failure store per refresh. No flock, lsof
  or ps on the refresh path.
- **Lease probe:** acquire and release of a local `flock`, microseconds, never blocking (`LOCK_NB`).

## 6. Fakes and conformance (ARCH-MOCK)

- `LeaseHolders` (wraps `lsof -F p <path>`) and `TTYProcesses` (wraps `ps -t <tty>`, on the existing
  process-table seam): stateful fakes keyed by path and tty. Their holders exit on SIGTERM, on SIGKILL only, or
  never. Tests use real `flock` on a temp file, held by real `sleep` children via `ExtraFiles` (cheap, exact).
- zellij, the process table, agent transcript roots and the failure store reuse the existing fakes
  (`FakeLaunchFailureStore`, artifact and session fakes), extended rather than duplicated.
- Live conformance: the drill (M5) runs the real zellij, Claude and Codex, plus one real-pty test in M2 (a shell
  on a pty that starts a setsid'd child: the tty empties while the child survives and is named).

## 7. Lifetimes (ARCH-FUNERAL)

- `lifetime.lease`: one empty file per slot, created by the first launch, removed with the slot's store dir.
- Agent tty in the agent sidecar: one field, removed with the sidecar at debris cleanup.
- `ThreadRecord.Transcript`: one per record, archived with it.
- Failure ring: 64 entries, one file (#683).
- Orphan registry: `orphans.json`, at most 256 entries. An entry is dropped when its process is observed gone, or
  evicted oldest-first at the bound.
- Drill results: one JSON per run under the pair data dir, pruned at 30 days by the drill itself.

## 8. Milestones

**Prerequisite, landing separately as #683 (hub decision, 2026-10-18):** P3, failure persistence. #684 starts
its code from main after #683 lands. No stacked branches (#272). #683's task outline (offered to that issue,
not a #684 milestone):

- `TestDispatchPersistsEveryOperationFailure`: a table over every declared operation. An executor fake returns
  an error, and the test asserts one `LifecycleFailure` with the operation, slot, phase and message.
  `TestFailureRingKeeps64NewestAndSlotLevelEntries`.
- Widen `LaunchFailure` to `LifecycleFailure` (`Phase`, `Holders`, `Outcome`; an empty `Tag` is allowed when
  `RepoScope` is set; the limit becomes 64).
- Wrap `DispatchOperation`. A save error joins the returned error. `failureFromCall` is pure and unit-tested.
- `PhaseError{Phase, Err}` on the park's early returns (park.go:762-833, 878, 942).
- `TestReattachFailurePersists` (menu_reattach.go:300-313).
- The row and `couch --show` read the latest failure per slot; a success supersedes it.
  `TestRowShowsLastFailureUntilSuccess`, `TestShowListsFailureAfterReceiptExpiry`.

**#684's milestones.** Each `Mx` is a review boundary (`sdlc milestone-close`). M1 is cheap and unblocks
diagnosis. M2 builds the proof. M3 fixes the incident paths. M4 finishes the rest. M5 proves it with the drill.

- [ ] M1 — Store the transcript binding on the record; Resume binds by one stat (P2). Covers D9, D10, D12.
- [ ] M2 — The lifetime lease, the agent tty witness, and `Teardown` with `TeardownProof`; archive and replace
  require a proof (P1). Covers D1–D8 and #683's finalization half.
- [ ] M3 — `Converge`, with paths 1, 2, 3, 5, 6, 7, 8, 9 and 10 moved onto it. Fixes #684.
- [ ] M4 — Paths 4, 11, 14, 15 and 16; classification collapsed to three outcomes, with the two-actor
  anomaly.
- [ ] M5 — The lifecycle drill on a disposable slot, with aged state, triggered from the hub. Seven consecutive
  green nights close the project's done-when.

### M1 tasks

**Files:**
- Create `threadrecord/transcript.go` and its test.
- Modify `couchcore/thread.go`, `threadrecord/record.go` (validation), `launcher/createflow.go` (minted IDs
  reported back), `couchcore/resume.go`, `couchcore/recovery_prepared.go` and `couchcore/relaunch.go`.
- `sessioninventory/native_selector*.go`: Resume stops calling them.

Steps:

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
- [ ] Commit `#684 M1: …`.

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
- Under Couch, the wrapper records the agent's tty in the agent sidecar. A real-pty test checks two things: a shell's
  foreground child on the pty is in the tty set, and a setsid'd grandchild is outside it and named in
  `LeftRunning`.
- `couchcore/teardown.go`, with the `LeaseHolders` and `TTYProcesses` seams and their fakes.
  `TestTeardownSweepsReparentedHolders` uses two holders: one survives SIGTERM, and one survives the round limit,
  which gives `ErrTeardownUnproven` naming it. `TestTeardownNeverKillsDetachedAgentChildren`.
- Archive and replace signatures take `TeardownProof`. `TestArchiveRefusesWithoutProof`,
  `TestFreshReplaceRequiresProof`.
- `TestTeardownFoldsCompletedPark` uses #683's retained shape: a success completion, the owner dead, the session
  absent.

### M3 tasks (detailed when M3 starts)

- `couchcore/converge.go`: the pure `PlanConverge`, table-tested over every target and evidence class, including
  two-actor and orphan evidence. Its executor goes in `converge_exec.go`.
- **#684 regression:** `TestFreshAfterFailedResumeTearsDownOrRefuses`. The record is parked, the resume fails, and
  an actor and the old session are still alive. Today that ends with two actors. After the fix there are two
  passing outcomes:
  - Teardown proves them gone and Fresh starts;
  - the fake's holder can't be killed, so Fresh refuses with the holder named and nothing archived.
- Paths 1, 2, 3, 5, 6, 7, 8, 9 and 10 rewired, with their old gates deleted. The per-path tests already in the
  tree either keep passing or, where they restate it, are replaced by the Converge table.

### M4 tasks (detailed when M4 starts)

- Paths 4, 11, 14 and 15 rewired or deleted.
- `ClassifyThread` returns one of three outcomes plus an anomaly count.
- The tab bar shows "2 actors" instead of raw names (D20). `TestTwoLiveActorsNeverContinue`.

### M5 — lifecycle drill (detailed when M5 starts)

- `scripts/lifecycle-drill` drives a running Couch through its CLI on a disposable slot. The default is
  `pair:9`, set by an env var, with no hardcoded paths.
- For Claude and for Codex, the sequence is:
  1. start;
  2. one trivial turn, to create the transcript;
  3. Alt+n relaunch, asserting the agent was relaunched with the same native ID;
  4. park;
  5. Resume, asserting the same ID;
  6. Start fresh, asserting a new ID and the old thread archived;
  7. archive.

  Resume is verified from the launch argv and the agent staying up, with no second turn.
- After each step the drill asserts four things:
  - at most one live actor;
  - the lease and the tty are free after each teardown;
  - every failure has a store entry;
  - `LeftRunning` named a seeded setsid'd child that survived.
- **Aged state** seeded before the run:
  - a record without a transcript binding (legacy);
  - a tree started without the lease (pre-M2);
  - a stray pair process holding the lease, aged by an old start time;
  - a drill `CODEX_HOME` (with `auth.json` symlinked to the real one) holding a copy of the last 2 weeks of
    Codex session files. It is bounded and realistic, not a tree of more than 10,000; M1's counted test is what
    rejects any scan.
- **Cost per run:** one minimal turn per agent, two in all. Each turn's input is the agent's own base context
  (system prompt and tools, on the order of tens of thousands of input tokens, largely cached after the first
  night), and the output is a few tokens. The prompt is fixed ("reply ok").
- **Trigger:** the hub runs it nightly until a Couch schedule exists, then the schedule runs it. No launchd. The
  drill writes one JSON per run and sends a summary to the hub. Seven consecutive green nights close the project.
- The daily read-only resume check (#680) stays its own issue. The drill reuses its resolver.

## 9. Relationship to other issues

- **#683:** it lands P3 first, as its own issue. M2's park fold delivers its finalization half; the M2 close
  notes that for #683's record.
- **#681** (admission before inventory) is landing separately and unchanged by this plan.
- **#686** (session-watch busy loop) is separate. M2's lease at least makes a spinning watcher killable by
  teardown.

## Revisions

- 2026-10-18 21:40 — operator approval with five changes (hub delivery 90271deb):
  1. Shift+Alt+N stays pair's fast context reset, untangled from Couch. Path 12 is dropped from the rewrite.
  2. Hard constraint: plain pair (no Couch, several sessions per directory) behaves exactly as today. The lease,
     the tty publication and `lease.Inherit` switch on only under `COUCH_LIFETIME_LEASE` + `COUCH_THREAD_TAG`.
     The binding lives in Couch's record. Added `TestPlainPairUnchangedWithoutCouch`.
  3. The drill's aged state is a copy of the last 2 weeks of session files, not a tree of more than 10,000.
  4. `LeftRunning` is persisted into an orphan registry recording its origin (slot, tag, command, start time).
     `couch --resources` names entries by originating slot, and `--reap-orphans --confirm` reaps them.
  5. Old-build refusal of M1 records: accepted as written.

  Code starts after #683 lands.
- 2026-10-18 21:55 — hub settled the reset corner: Bind reads the tag's own ledger file once (exact path) and adopts a newer root; a missing or unreadable file falls back to the stored binding with a row note, never a scan. Added `TestBindAdoptsNewerLedgerRootAfterReset`; the counted test is now 1 read, 1 stat, 0 listings.
