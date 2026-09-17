# Lifecycle Transition Authority Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make recoverability a fact about the **world**, not about whether couch
survived long enough to write a record — so a timed-out park stops wedging its
thread (#271) and a dead launcher stops hiding a running agent (#272).

**Architecture:** One rule, applied by deletion rather than addition: **the
classifier reads the session; it does not read the bookkeeping.** `Incarnation`
and `record.Park` leave the classification path entirely. That is sound because
the zellij server is PPID 1 at birth, so a couch death kills only the launcher —
couch's own child — which means a clean detach and a couch crash leave *identical*
external state. Treating the missing record as brokenness is the whole bug family.
Then the mutation guards stop reading the raw record and consume the
classification, so what the switcher offers and what the store permits cannot
disagree.

**Tech Stack:** Go; `cmd/internal/couchcore` (classifier, store, recovery),
`cmd/internal/launcher` (zellij session snapshot), `cmd/internal/couchtty`
(switcher menu). Tests: table-driven pure unit tests via the existing
`actionableTestThread` builder, `FakeProcOps` for the seam, sequence tests over
production transition functions.

---

## Context: the measurement this plan rests on

From the operator's live process table, 2026-09-16, with couch at pid 65018
**still running**:

```
65018   61189    bin/couch
66197   65018      pair resume couch-dbc88727c6378a0f --layout3   ← launcher
66242   66197        zellij (client)
66246       1    zellij --server … 📁brain-couch-26               ← PPID 1 AT BIRTH
66247   66246      pair wrap → 66261 claude --session-id d16964c6-…
```

The zellij **server** is PPID 1 while couch is alive — it daemonized, it was not
reparented. `pair wrap`, `pair term`, nvim and the agent are its children, not
couch's.

`Detach` (`detach.go:91-99`) does exactly three things: SIGTERM the launcher's
process group, wait for that pid to exit, clear the incarnation. The launcher is
**couch's own child**, so:

> **A clean detach and a couch crash leave identical external state.** The only
> difference is whether the bookkeeping ran.

Today that difference decides everything — with the record, `detached`
(recoverable, ranked highest at startup); without it, `stale — helper ownership
unresolved` (debris). Same world, opposite verdicts. Every issue in this family
is a consequence.

The liveness referent is therefore the **zellij session**: independent from birth,
it dominates `pair wrap` and the agent in lifetime, and it is the entity
`zellij list-sessions` already reports (ARCH-DRY). The proof is
**agent-independent by construction**, which is this plan's answer to #272's
"enumerate what survives for each supported agent" — the enumeration is
unnecessary because the proof does not depend on the agent.

### Store state after the operator's 2026-09-16 cleanup

17 records → 7, across 4 scopes, 40 archived.

| Record | Scope | Fixture for |
|---|---|---|
| `couch-3b82bfd593cac896` | brain | **Task 6** — dead incarnation, no binding, unarchivable |
| `couch-e1a31510b7033d08` | brain | **Tasks 2 + 6** — wedged park, then the same wall |
| `couch-dbc88727c6378a0f` | brain | live (📁brain-couch-26) |
| `couch-2e662a595ae09564` | tools | live |
| `couch-441f91ad8ccc9bad` | parley | live |
| `couch-ff764f69b258e5d6` | ariadne | live |
| `couch-c945633f5c806f21` | pair | live — **the session executing this plan** |

**#272's primary fixture is gone.** The three muse threads with dead launchers and
live agents were archived, so "eleven records with a dead launcher pid" is no
longer reproducible; M2's verification builds the fixture instead of observing it.

Still live and recordless — now owned by **#276**, not this plan:
`couch-797c45e8e649a9bb` (📁parley-couch) and `couch-2583ed61c0ab6ebe`
(📁parley-couch-2), both running since 2026-08-30. Also `87464` 📁brain-couch-2,
a server whose agent is gone.

**Safety note:** the operator's session runs inside `couch-c945633f5c806f21` /
📁pair-couch-37. Parking or archiving that thread kills the session doing the
work. Never exercise lifecycle verbs against the hosting thread; the fakes exist
so you do not have to.

---

## Core concepts

### What durable state has to earn

Durable state earns its place only when it records something **not derivable from
the world**. Measured against the record today:

| Keep — not observable | Drop from the *classification path* — a shadow of what you can look at |
|---|---|
| native session id (the conversation) | `Incarnation{PID, Identity, State}` |
| address → session-name binding | `ParkTransaction{Phase, Attempts, …}` |
| launch profile, paths, name, description | — |

Neither field is **deleted** here — that is a schema change, and the park half is
**#275**. They simply stop being read by `ClassifyThread`. That distinction keeps
this issue bounded while still removing every way the bookkeeping can wedge a
thread.

### Two questions, two authorities

The gate's `classification-not-authority` family names the trap this plan must
not fall into: moving the classifier off the bookkeeping while the **guards**
still read it. Post-M1 there are two distinct questions, and they cannot share a
predicate:

| Question | Authority | Who asks | Needs evidence? |
|---|---|---|---|
| Is this thread actionable right now? | the **classification** | `DecideResume`, `Couch.ArchiveThread`, the menu | yes |
| Is this record structurally safe to mutate? | the **record alone** | `ThreadStore` inside `withLock` | no — it has none |

`occupiedIncarnation` (`thread.go:377`) is today one rule serving both, shared by
resume and archive and re-read by the store. Its referent is the launcher, which
dies with couch — so post-M1 it answers the first question wrongly and is the
only thing answering the second. **It is deleted**, and the two questions get
separate guards (Tasks 8 and 8a).

This is also why `ArchivableState(state, reason)` cannot simply be dropped into
`threadstore.go:1106`: that call site is inside `s.withLock` on a record decoded
from disk, with no evidence, no session observation and no `Couch`. Handing it a
caller-computed classification would destroy the independence the comment at
`:1093-1095` claims ("second line of defence"). The store keeps a **record-only
integrity** guard instead — a narrower question it can actually answer.

### The resource/ownership map

#256 requires modelling these **without flattening them into one global state
machine** — each has its own lifetime (ARCH-ORDER).

| Resource | Exact identity | Re-observed via | Survives couch death |
|---|---|---|---|
| Thread | `{RepoScope, Tag}` | it *is* the record | yes (durable) |
| Zellij session | session name | `zellij list-sessions` | **yes** |
| Agent conversation | native session id | ledger / binding index | yes |
| Couch-hosted client | pty child | couch's own child table (in memory) | no |
| Launcher process | `{PID, Identity}` | `ProcOps` | **no** |

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `SessionObservation` | `cmd/internal/couchcore/sessionevidence.go` | new |
| `ThreadEvidence` | `cmd/internal/couchcore/actionableinventory.go` | modified |
| `ClassifyThread` | `cmd/internal/couchcore/actionableinventory.go` | modified |
| `ArchivableState` | `cmd/internal/couchcore/thread.go` | new |
| `AllThreadStates` | `cmd/internal/couchcore/actionableinventory.go` | new |
| `archivableRecord` | `cmd/internal/couchcore/thread.go` | deleted |
| `occupiedIncarnation` | `cmd/internal/couchcore/thread.go` | deleted |
| `liveProofMatches` | `cmd/internal/couchcore/actionableinventory.go` | deleted |
| `startInFlight` | `cmd/internal/couchcore/actionableinventory.go` | deleted |

- **SessionObservation** — one thread's zellij session as a **three-state**
  answer:

  | Value | Meaning |
  |---|---|
  | `SessionUnresolved` | the question could not be asked (index unreadable, snapshot failed, binding contested) |
  | `SessionAbsent` | asked, and no live session is bound to this address |
  | `SessionPresent` | asked, and a live session is bound |

  - **Why three and not four:** an earlier draft carried a fourth value,
    `SessionHeldElsewhere`. Under optimistic inventory (below) the refresh can
    never produce it, and the only consumer that needs attached-versus-detached
    is the **action** path, which re-observes independently and is already
    governed by `RequireAttachState` (`launcher/session.go:24`). A value no
    producer can emit is a state that exists only to be handled.
  - **`SessionUnresolved` must never collapse into `SessionAbsent`.**
    `DetachedSessions` (`artifactcollision.go:280-363`) fails closed per scope
    with `continue`, so today "no binding", "scope unreadable" and "session
    exited" all read as absence → `session-gone`, which is archive-eligible. With
    recoverability keyed to sessions, that collapse becomes destructive.

- **ThreadEvidence** — modified: gains `Session`. **`Live` keeps its current
  union** — the console's pty children *plus* `ObserveRecordedProcesses`
  (`actionableinventory.go:413-437`) — and becomes **positive-only**.
  - An earlier draft narrowed it to couch's own children. That strands the CLI:
    `ThreadInventoryContext` passes `nil` observations
    (`threadinventory.go:97-100`) precisely so "the CLI and the console read the
    same proof", so the union is `couch --list`'s *only* liveness evidence.
    Narrowing it would make every running thread read `detached` there — #181's
    "one store, two stories", reintroduced.
  - The union was never the bug. For a thread couch hosts, the launcher **is**
    alive (it is couch's child), so a positive observation is correct. The bug
    was reading its **absence** as proof of death. After Task 2 no branch does:
    absence simply falls through to the session branch.

- **ArchivableState** — pure predicate over `(ActionableThreadState,
  ThreadReason)`, consumed by both the menu and the store so the offer and the
  permission cannot disagree.

- **`startInFlight` is deleted, not narrowed.** A start in flight is
  **couch-local, in-memory** knowledge — couch knows what it is currently
  spawning. It arrives as an observation alongside `Live`, the same way the pty
  children do, rather than being inferred from a durable `creating` incarnation
  that outlives the process it describes. Ephemeral state stays ephemeral.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ProcOps` | `cmd/internal/couchcore/procops.go` | unchanged | process table |
| `ObserveRecordedProcesses` | `cmd/internal/couchcore/actionableinventory.go` | modified | `ProcOps` |
| `observeSessions` | `cmd/internal/couchcore/sessionevidence.go` | new | `zellij list-sessions` |

- **ObserveRecordedProcesses** — modified to preserve `Unknown` rather than
  collapsing it into the `Dead` branch's silent `continue`
  (`actionableinventory.go:582`; `procops.go:22` says *"prune only on Dead.
  Unknown must fail CLOSED"*).
  - **Its consumers, enumerated — and what each receives after the change**
    (the rule the gate named: never change an evidence producer without this
    list). Its output is unioned into `Live` at `:436`, and `Live` is
    classification row 4 — so it **remains** a classification input, contrary to
    an earlier draft of this bullet.

    | Consumer | After the change |
    |---|---|
    | `Live` (→ `ClassifyThread` row 4) | **only confirmed-`Live`** observations enter; `Unknown` never does |
    | `DecideRecovery` (`in.Helper`) | receives `Unknown` distinctly from `Dead` |
    | `Couch.ArchiveThread` | receives `Unknown` distinctly from `Dead`; must fail closed |

  - **Why `Unknown` must not enter `Live`:** it would classify the thread `live`,
    which hides it from resume *and* from recovery — strictly worse than the
    `stale` it replaces. The demotion that matters is not "no longer a
    classification input" (it is one); it is that **absence** of a positive
    observation no longer produces `stale-incarnation`.

- **observeSessions** — **optimistic inventory, strict action** (operator
  decision, 2026-09-16). The refresh asks one host-wide `list-sessions` and never
  `list-clients`.
  - **Why it is safe:** the reattach path already re-observes with attach state
    before committing (`resume.go:406-418`), so the expensive question is asked
    for the one thread the operator pressed Enter on. Cost is proportional to
    what you **do**, not what you **have**.
  - **ARCH-CONSTRAINTS:** `list-clients` costs ~250 ms per live session
    (measured, #228); ~6 live couch-tagged sessions on the operator's host, so
    always-asking would have added ~1.5 s to a startup #218 already exists to
    investigate. **#191 is a backstop, not a dependency.**
  - **The cost is per-refresh, not only per-startup.** `gatherThreadEvidence`
    today bounds whether the zellij snapshot runs at all
    (`actionableinventory.go:441-447` — *"a couch with nothing detachable pays
    nothing"*); Task 1 gathers session evidence for every record, so **every
    refresh** pays one `list-sessions`. That is off the keystroke path — the
    refresh runs in a worker goroutine and is coalesced by generation
    (`couchtty/console_menu.go:83-113`) — so it costs latency nowhere the
    operator waits. Budget both figures and record them in `## Log`: the startup
    evidence round, and one steady-state refresh, on the operator's store
    (7 records, 4 scopes post-cleanup).
  - **Accepted tradeoff:** a session the operator manually attached to reads
    present, and Enter fails — loudly, with a re-observed reason, rather than
    silently wrong. Taxing every startup for a case that essentially never occurs
    is the worse trade.

---

## Milestones

Three review boundaries. M1 is the rule and fixes both filed bugs; M2 proves it
against the world; M3 closes the doors that let the drift happen.

- **M1** — The classifier reads the session, not the bookkeeping (#271, #272).
- **M2** — Make the operator's rows actually reachable, and verify against real
  sessions.
- **M3** — Guards consume the classification; close the arbitrary-mutation door.

---

## Chunk 1: M1 — the classifier reads the world

### Task 1: Session evidence as a three-state answer

**Files:**
- Create: `cmd/internal/couchcore/sessionevidence.go`, `sessionevidence_test.go`
- Modify: `cmd/internal/couchcore/actionableinventory.go` (`gatherThreadEvidence`)

New file rather than growing `actionableinventory.go`, already 642 lines and
holding the state vocabulary, evidence types, label helpers, the pure classifier
**and** the `Couch` IO methods.

`ProjectDetachedSessions` hard-refuses an existence-only snapshot via
`launcher.RequireAttachState` — keep that governing the **action** path and add
the existence pass beside it, rather than relaxing it.

- [ ] **Step 1: Write the failing test** — a table over the three values,
  including the two that today collapse into "no observation":

```go
{"index unreadable",   …, want: SessionUnresolved},
{"binding contested",  …, want: SessionUnresolved},
{"no live session",    …, want: SessionAbsent},
{"live session bound", …, want: SessionPresent},
```

- [ ] **Step 2: Run it and watch it fail** — the type does not exist.
- [ ] **Step 3: Implement.** Preserve `ProjectDetachedSessions`' fail-closed
  uniqueness rules (`claims == 1`, no duplicate rows) — those stand in for #272's
  "exact `ProcessIdentity` match", since a session *name* carries no start token.
  State that equivalence in the doc comment rather than dropping #272's bullet.
  Gather for **every** record, not only resume-shaped ones — the `resumeShaped`
  gate at `:460` is why a record carrying an incarnation never got asked about.
- [ ] **Step 4:** `go test ./cmd/internal/couchcore/` → PASS.
- [ ] **Step 5: Commit** — `#256 M1: session existence is evidence for every record`

### Task 2: Delete the bookkeeping from the classification path

**Files:**
- Modify: `cmd/internal/couchcore/actionableinventory.go` (`ClassifyThread`; delete `liveProofMatches`, `startInFlight`; the `ThreadBusy` doc comment at `:21-23`)
- Modify: `cmd/internal/couchcore/layout.go:78-88` (`holdsSession`) — **comment only**
- Test: `cmd/internal/couchcore/actionableinventory_test.go`

**`ThreadBusy` survives, with a new meaning.** It stops meaning "a park is in
flight" and starts meaning "couch is starting this thread right now". Every
consumer must be revisited, because the state name is unchanged while its
referent is not — the failure mode a rename would have caught:

| Consumer | After Task 2 |
|---|---|
| `layout.go:87` `holdsSession` | **code stays** — a starting thread may hold a session — but its comment's rationale ("a park in flight can still fail") is now false and must be replaced |
| `recovery.go:75` `ProjectRecoveryChoices` | still correct: a starting thread offers no recovery |
| `couchcmd/run.go:770`, `couchtty/menu_render.go:432` | **wording is now wrong** — see Task 4 |

**This one task fixes both #271 and #272**, by removing reads rather than adding
rules. The new branch order:

| # | Condition | State |
|---|---|---|
| 1 | record fails validation | `unusable` / `invalid` |
| 2 | reservation | `unusable` / `never-started` |
| 3 | couch is starting this thread **right now** (in-memory observation) | `busy` |
| 4 | couch hosts this pty | `live` |
| 5 | `SessionPresent` | `detached` |
| 6 | `SessionUnresolved` | `unusable` / `unknown` |
| 7 | `VerifiedPark` payload + resolvable native id | `parked` |
| 8 | ledger holds a resolvable native id | `unusable` / `binding-lost` |
| 9 | otherwise | `unusable` / `session-gone` |

`record.Incarnations` and `record.Park` appear **nowhere**. Rows 7–9 read only
resume authority, which is genuinely durable.

- [ ] **Step 1: Write the two failing tests — the operator's actual rows**

```go
func TestWedgedParkDoesNotWedgeClassification(t *testing.T) {
	// couch-e1a31510b7033d08, rev 293: park phase awaiting_completion since
	// 2026-09-15, pid 64734 gone. `if record.Park != nil { return ThreadBusy }`
	// is total and unconditional, so this row read `parking…` for ~18 hours and
	// the switcher offered it exactly two actions: name and describe.
	record := actionableTestThread("couch-e1a31510b7033d08", time.Now())
	record.Park = wedgedParkFixture(record.Address)
	record.Incarnations = []ThreadIncarnation{{PID: 64734, Identity: "1789535173.46673", State: IncarnationLive}}
	state, _ := ClassifyThread(record, ThreadEvidence{Session: SessionObservation{State: SessionAbsent}})
	if state == ThreadBusy {
		t.Fatal("an open park transaction still decides recoverability")
	}
}

func TestDeadLauncherWithLiveSessionIsDetached(t *testing.T) {
	// pair#272: the launcher dies with couch; the session does not. A clean
	// detach and a crash leave identical external state, so they must classify
	// identically.
	record := actionableTestThread("couch-95293a9b6c0d459a", time.Now())
	record.Incarnations = []ThreadIncarnation{{PID: 81935, Identity: "dead", State: IncarnationLive}}
	record.LatestLaunchProfile = &LaunchProfile{Agent: "muse", Argv: []string{}}
	state, _ := ClassifyThread(record, ThreadEvidence{Session: SessionObservation{State: SessionPresent, Name: "📁pair-couch-32"}})
	if state != ThreadDetached {
		t.Fatalf("state = %q, want detached", state)
	}
}
```

- [ ] **Step 2: Run them and watch them fail** — `busy` and
  `unusable/stale-incarnation` respectively.
- [ ] **Step 3: Implement** the branch table above. Delete `liveProofMatches` and
  `startInFlight`. Delete the disproved comment at `:21-23` ("ThreadBusy … it
  resolves on its own") — its twin in `menu.go` goes in Task 4.
- [ ] **Step 4:** Run the suite. Existing stale-incarnation and busy tests will
  fail; restate each expectation **with the reason in the test name or a
  comment**. Do not weaken an assertion to make it pass — a test that cannot be
  restated is evidence the rule is wrong.
- [ ] **Step 5: Commit** — `#256 M1: recoverability is a fact about the session`

### Task 3: Retire the reasons the rule made unreachable

**Files:**
- Modify: `cmd/internal/couchcore/threadreason.go`, `threadreason_test.go`
- Modify: `cmd/internal/couchcore/classify_test.go`, `cmd/internal/couchtty/menu.go` (`unusableThreadNotice`)

`ReasonStaleIncarnation` and `ReasonUnrecordedChild` are produced by branches
Task 2 deletes. `classify_test.go:271` asserts *"nothing produces reason %q"*, so
the suite will say which are now orphaned — that guard is the worklist.

Three guards hard-fail on any reason-vocabulary change and must be updated
together: `threadreason_test.go`'s `defining` map, `menu_test.go:1259`
`TestEveryReasonExplainsItselfOnEnter` (via `unusableThreadNotice`), and
`classify_test.go:271`.

- [ ] **Step 1:** Run the three guards; let them name the orphaned reasons.
- [ ] **Step 2:** Remove what is genuinely unreachable. **Keep**
  `ReasonUnrecordedChild` — #276 will produce it — with a comment saying which
  issue does, so the guard stays honest rather than being silenced.
- [ ] **Step 3:** Re-run → PASS.
- [ ] **Step 4:** Commit, then `sdlc milestone-close --issue 256 --milestone M1`.

---

## Chunk 2: M2 — make the rows reachable and prove it

### Task 4: The wedged row offers something

**Files:**
- Modify: `cmd/internal/couchtty/menu.go:1238-1242`, `cmd/internal/couchtty/menu_render.go:432`, `cmd/internal/couchcmd/run.go:770`
- Test: `cmd/internal/couchtty/menu_test.go`, `cmd/internal/couchcmd/run_test.go`

`menu.go:1238` gives a `ThreadBusy` row exactly `name` and `describe`, behind a
comment asserting the thing Task 2 disproves: *"It resolves on its own."* After
Task 2 that row is no longer `busy`, so the branch is dead code stating a false
premise — delete both.

- [ ] **Step 1:** Write the failing test — the wedged fixture offers `archive`.
- [ ] **Step 2:** Red.
- [ ] **Step 3:** Delete the comment and the branch; let `menuActionsFor` fall
  through to the non-live action set.
- [ ] **Step 3b: Re-word both renderers.** An earlier draft said the wording
  "follows automatically — verify, do not edit". That was **wrong**: `ThreadBusy`
  survives Task 2 with a new referent, so `menu_render.go:432` (`"parking…"`) and
  `couchcmd/run.go:770` (`"parking in progress"`) now describe the wrong thing.
  Both become a starting-up wording. Two renderers, one meaning — assert they
  agree, the way `ThreadReason.Label()` is single-sourced for exactly this reason.
- [ ] **Step 4:** Green.
- [ ] **Step 5:** Commit.

### Task 5: `DecideRecovery` stops gating on an open park

**Files:**
- Modify: `cmd/internal/couchcore/recovery.go:33-47`
- Test: `cmd/internal/couchcore/recovery_test.go`

`ArchiveThread` refuses **earlier** than `archivableRecord`, at
`DecideRecovery:33` — *"a park transaction is still open; let lifecycle recovery
finish."* Task 2 alone therefore changes the row's label and nothing else: the
gesture still fails. Same for `:38-47`, which gate on incarnation shape.

- [ ] **Step 1:** Write the failing test — the wedged record is archivable.
- [ ] **Step 2:** Red — the park gate refuses.
- [ ] **Step 3:** Remove the park gate and the incarnation-shape gates; keep the
  helper-liveness gate, which guards an irreversible act.
- [ ] **Step 4:** Green.
- [ ] **Step 5:** Commit.

### Task 6: An absent binding must not shield a dead incarnation

**Files:**
- Modify: `cmd/internal/couchcore/recovery_execute.go:63-68` (`observeRecoverySession` — where the absent-binding error is **raised**; `reconcileRecoveryHelper:105` only propagates it), `cmd/internal/couchcore/detach.go:262-272`
- Test: `cmd/internal/couchcore/archive_test.go`

**The task that actually makes the operator's two `brain` rows archivable.**
Traced live 2026-09-16 — both carry one incarnation with `state: live`,
`start: nil`, a pid confirmed gone (ESRCH), and **no session-name binding**
(`repos/2e51fcf9799b1d8f/session-names.jsonl` holds only
`couch-dbc88727c6378a0f`). `ArchiveThread`'s binding-absent escape hatch admits a
record only when `len(Incarnations) == 0 && Park == nil && Continuation == nil`.

The retirement logic that fixes this **already exists and is correct** —
`reconcileRecoveryHelper:120-123` re-probes the exact `{PID, Identity}`, requires
`Dead`, and calls `RetireIncarnation`. It is **unreachable**: `observeRecovery`
errors at `:105` on the absent binding, fifteen lines earlier.

The rule: **an absent session binding is not a reason to skip retiring a
provably-dead incarnation — it is corroborating evidence the thread is gone.**

- [ ] **Step 1: Write the failing test**

```go
func TestSpawnedButNeverBoundThreadIsArchivable(t *testing.T) {
	// couch-3b82bfd593cac896: one incarnation {pid 87309, state live, start
	// nil}, pid gone, no row in session-names.jsonl, last_active_at at the zero
	// time — #273's fresh-spawn shape. Archive refuses today.
	…
	if _, err := couch.ArchiveThread(ctx, address); err != nil {
		t.Fatalf("a thread with a dead helper and no binding is unarchivable: %v", err)
	}
}
```

- [ ] **Step 2:** Red — "recovery session state could not be checked".
- [ ] **Step 3:** When the binding is absent **and** the single incarnation is
  provably `Dead` with `Start == nil`, treat presence as `SessionAbsent` rather
  than erroring, so `:120-123` runs. Do **not** widen to `Unknown` liveness — an
  unanswerable probe must still fail closed.
- [ ] **Step 4:** Green.
- [ ] **Step 5: Commit** — `#256 M2: an absent binding corroborates death, it does not hide it`

### Task 7: Sequence tests against the real failure modes

**Files:**
- Test: `cmd/internal/couchcore/recovery_test.go`

ARCH-ORDER asks which events the caller **cannot** block. Four apply:

| Event | Governed by | Rolls back? |
|---|---|---|
| couch killed, session survives | classify from the session | n/a — nothing to roll back |
| couch killed, session also dies | `session-gone` or `binding-lost` via ledger | n/a |
| park times out, process dies | nothing reads `record.Park` | n/a |
| liveness probe returns Unknown | fail closed — `unusable/unknown` | n/a |

Not applicable, stated rather than marked: there is no durable transaction left in
the classification path to interrupt, which is the point of the re-cut. Rollback
is absent because the classifier holds no state between events — it is a pure
function of (record, evidence).

- [ ] **Steps 1–4:** One test per row.
- [ ] **Step 5:** Commit, then `sdlc milestone-close --issue 256 --milestone M2`.

**Operator verification for M2:**

1. `brain` returns to usable work: the two wedged rows are archivable, and a
   fresh start does not mint another orphan. #273 observed three orphans, one per
   attempt; the count must stop growing.
2. **Build #272's fixture rather than waiting for one:** kill a couch while a
   thread runs — the Context measurement says the session survives — then confirm
   the thread reads `detached` **and that Enter actually reattaches**. A row that
   merely reads `detached` proves nothing.
3. Record the startup evidence-round timing per the ARCH-CONSTRAINTS budget.

---

## Chunk 3: M3 — guards consume the classification

### Task 8: Every action guard consumes the classification

**Files:**
- Modify: `cmd/internal/couchcore/thread.go:356-366` (replace `archivableRecord` with `ArchivableState`), `threadstore.go:1106`, `detach.go:277`
- Modify: `cmd/internal/couchcore/archive_test.go:108`, `continuation_guard_test.go:26` (existing call sites)
- Test: `cmd/internal/couchtty/archive_agreement_test.go` (new — **in `couchtty`**, which imports `couchcore`, not the reverse)

- [ ] **Step 1: Write the failing test** — iterate **states × reasons**, not
  reasons alone. The menu offers archive from `ThreadParked` and the default
  branch (`menu.go:1260-1263`), and after M1 `detached` is where a record can
  still carry an occupied incarnation.

```go
for _, state := range couchcore.AllThreadStates() {
	for _, reason := range couchcore.AllThreadReasons() {
		offered := containsMenuItem(menuActionItems(row(state, reason)), "archive")
		permitted := couchcore.ArchivableState(state, reason)
		if offered != permitted {
			t.Errorf("%s/%s: menu offers=%v store permits=%v", state, reason, offered, permitted)
		}
	}
}
```

- [ ] **Step 2:** Red.
- [ ] **Step 3:** Add `AllThreadStates()` beside `AllThreadReasons()`
  (`threadreason.go:67`) — it does not exist today, and the codebase's own
  rationale applies verbatim: *"Go cannot check a switch for exhaustiveness; this
  enumeration is what does."*
- [ ] **Step 4:** Add `ArchivableState` and have the **menu** and
  `Couch.ArchiveThread` consume it. Delete `occupiedIncarnation`.
- [ ] **Step 5:** `go test ./cmd/internal/couchcore/ ./cmd/internal/couchtty/` → PASS.
- [ ] **Step 6:** Commit.

### Task 8a: The same rule for resume — the other half of the class

**Files:**
- Modify: `cmd/internal/couchcore/resume.go:96-106` (`DecideResume`), `cmd/internal/couchcore/startup.go` (`SelectResumableRoot` callers)
- Test: `cmd/internal/couchcore/resume_test.go`, `cmd/internal/couchtty/archive_agreement_test.go`

**Archive is not the only guard reading the bookkeeping.** `DecideResume` refuses
on `record.Park != nil` (`:98`) and on `occupiedIncarnation` (`:104`) — and its
own comment says *"One occupancy rule, shared with archive."* So post-M1 a #272
record classifies `detached`, `SelectResumableRoot` (`startup.go:37`) ranks
`detached` **highest** and auto-selects it at startup, `menu.go:1263` offers
resume, and the resume refuses on an incarnation nothing reads any more.

That is the identical defect Task 8 fixes for archive. Fixing only archive would
be the instance, not the class (ARCH-PURPOSE).

- [ ] **Step 1: Write the failing test** — a #272-shaped record classified
  `detached` is resumable, and a guard test asserting **offered ⇒ permitted** for
  resume exactly as Task 8 does for archive, over states × reasons.
- [ ] **Step 2:** Red — `resume-creating` / `resume-live` refusal.
- [ ] **Step 3:** `DecideResume` consumes the classification. Drop the park and
  occupancy reads; keep the refusals that rest on genuinely durable facts (path,
  profile, agent support, resume authority).
- [ ] **Step 4:** `go test ./cmd/internal/couchcore/ ./cmd/internal/couchtty/` → PASS.
- [ ] **Step 5: Commit** — `#256 M3: resume and archive read one authority`

### Task 8b: The store keeps a record-only integrity guard

**Files:**
- Modify: `cmd/internal/couchcore/threadstore.go:1072-1106`, `cmd/internal/couchcore/thread.go:356-366`
- Test: `cmd/internal/couchcore/threadstore_test.go`

The store cannot evaluate a classification and must not pretend to. Its guard
narrows to what a decoded record proves on its own — and the **consequence must
be stated rather than discovered**: post-M1 `ArchivableState(detached, "")` is
true, so the store stops independently refusing a record that still carries a
live incarnation. That refusal moves to `Couch.ArchiveThread`, which has the
evidence to make it.

This mirrors the existing precedent at `startup.go:68-77` — *"three predicates,
deliberately distinct, because they ask different things… collapsing them would
force one answer onto three questions."*

- [ ] **Step 1:** Failing test — the store refuses a structurally impossible
  archive and permits one whose actionability only the Couch layer can judge.
- [ ] **Step 2:** Red.
- [ ] **Step 3:** Narrow the store guard; update the `:1093-1095` comment to say
  what the two layers now each own, rather than claiming a defence it no longer
  provides.
- [ ] **Step 4:** Green.
- [ ] **Step 5:** Commit.

### Task 9: Preserve Unknown on the destructive paths

**Files:**
- Modify: `cmd/internal/couchcore/actionableinventory.go:568-596` (`ObserveRecordedProcesses`)
- Test: `cmd/internal/couchcore/actionableinventory_test.go`

`procops.go:22` says *"prune only on Dead. Unknown must fail CLOSED"*, and `:582`
collapses both into one silent `continue`. After M1 this is no longer a
classification input — its consumers are `DecideRecovery` and archive, where an
irreversible act is gated. #256's Done-when ("Unknown observations cannot become
confirmed absence or authorize destructive recovery") lands here.

- [ ] **Step 1:** Failing test — an `Unknown` probe does not authorize archive.
- [ ] **Step 2:** Red.
- [ ] **Step 3:** Three-way switch; carry `Unknown` to the recovery consumers.
- [ ] **Step 4:** Green.
- [ ] **Step 5:** Commit.

### Task 10: Archive confirms before stopping a live agent

**Files:**
- Modify: `cmd/internal/couchtty/menu.go` (`confirmationMenuItems`, ~`:1517`)
- Test: `cmd/internal/couchtty/menu_test.go`

**Operator decision, 2026-09-16:** archive on a row whose session is alive
**proceeds, behind an extra confirmation naming the agent** — the shape the
per-thread park already uses. It does not refuse: `recovery.go:56-58`
deliberately allows both Recover and Archive for a live detached session, and
refusing would break the ordinary gesture and violate Task 8's invariant.

- [ ] **Step 0: Establish what `Quiesce` actually reaps.** `ArchiveThread` runs
  `Quiesce` first (`detach.go:298`) — `zellij delete-session --force`, polled
  until the **session** is gone. But #274 measures that a dead session does not
  reap its panes: `pair term` ignores SIGHUP (inherited `SIG_IGN`, which Go
  preserves), leaving 106 orphaned trees and ~77 GB. Whether `pair wrap` and the
  agent share that fate is **not established** — #274 measured `pair term` and
  `nvim`, and the operator's cleanup removed the population that would have
  answered. Start a thread against the fakes, `Quiesce` it, assert `pair wrap`
  and the agent are gone, not merely the session. If they survive, #272's
  "archive cannot silently abandon a running agent" is **not** satisfiable by
  ordering, #274 becomes a dependency, and this confirmation must say "stops the
  session; the agent may survive". Record the result in `## Log` either way.
- [ ] **Step 1–4:** Failing test that the confirmation names the agent; red;
  extend `confirmationMenuItems` following the existing `leave` precedent; green.
- [ ] **Step 5:** Commit.

### Task 11: Close the arbitrary-mutation door

**Files:**
- Modify: `cmd/internal/couchcore/threadstore.go:293`, `recovery_execute.go:320`, `continuation_recovery.go:127,310`
- Modify: `cmd/internal/couchtty/park_latency_test.go:48`, `cmd/internal/couchcmd/warm_reattach_test.go:179`, `recovery_acceptance_test.go:105`, `run_test.go:331`
- Test: `cmd/internal/couchcore/mutation_door_test.go` (new)

**18** non-test call sites = **12** named wrappers in `threadstore.go` + **3**
named `ThreadStore` wrappers elsewhere (`continuation_store.go:11,35`,
`threadmetadata.go:15`) + **3** passing arbitrary lifecycle mutations:

| Site | Mutation | Becomes |
|---|---|---|
| `continuation_recovery.go:127` | `VerifiedPark = nil` | `ClearVerifiedPark` |
| `continuation_recovery.go:310` | `Incarnations = nil` | `RetireIncarnations` |
| `recovery_execute.go:320` | sets `Continuation`, clears `Incarnations` | `BeginContinuationFromRetiredIncarnation` |

**The guard must be file-scoped, not package-scoped.** All three leaking sites are
in package `couchcore`, same as `threadstore.go` — so unexporting
`UpdateExistingThread` stops nothing on its own, and a guard modelled on
`discoverConsumerPackages` (`terminal/no_destination_contract_test.go:43`), which
enumerates *sibling packages*, cannot see them. Derive the permitted **file** set
rather than listing it (ARCH-PURPOSE: the class, not the three instances).

`ThreadStore` is a concrete `*ThreadStore` (`couch.go:36`), so unexporting is a
pure compile-time change with no interface to update.

- [ ] **Step 1:** Write the AST guard; run it — must fail on all three sites.
- [ ] **Step 2:** Add the three named transitions; unexport `UpdateExistingThread`.
- [ ] **Step 3:** Fix the four external test call sites — three mutate lifecycle
  fields (`park_latency_test.go:48` wants `ApplyThreadMetadata`); budget them
  rather than meeting them at compile time.
- [ ] **Step 4:** `go test ./cmd/...` → PASS.
- [ ] **Step 5:** Commit.

### Task 12: Atlas and lessons

**Files:** `atlas/couch.md`, `workshop/lessons.md`

- [ ] **Step 1:** Record the resource/ownership map and the durable-state rule in
  `atlas/couch.md`.
- [ ] **Step 2:** Two lessons: *a liveness proof keyed to a process that dies
  before the thing it proves will report every crash as a lost thread*; and
  *when a clean shutdown and a crash leave identical external state, the record
  of the shutdown must not decide recoverability*.
- [ ] **Step 3:** Commit, then `sdlc close --issue 256 --verified '<evidence>'`.

---

## Settled scope decisions

**No bulk-archive sweep** (operator, 2026-09-16): *archive is an outlier
operation, typically means something went wrong, so we can afford to do things
individually.* A sweep optimises the path you least want to be fast.

**Park's performance is #275.** Park's one non-cosmetic property is that it is
irreversible and ordered — the native id and scrollback must be captured before
teardown. That needs ordering and idempotence, not phases, attempts, nonces and
tombstones. Replacing the machinery is #275; this issue only stops the
*classifier* reading it.

**A surviving session should repair its own bookkeeping — deferred, not denied**
(operator, 2026-09-17). This plan keeps `profile-missing` and `path-missing`
ahead of the session branch, so a thread whose session is alive but whose profile
was never recorded still reads unusable. The operator's objection is correct and
is the same one this whole issue rests on:

> if external state is clean and can be resumed, and we refuse to resume, merely
> because we didn't make some immaterial house keeping steps, then we should
> really rely on external state and repair our internal state.

It is accepted **for now** because `DecideResume` genuinely needs the profile, so
showing `detached` without one would offer a resume that always fails — the
anti-pattern this issue exists to remove. The repair is concrete and cheap:
`ledger-<tag>.jsonl` keeps every launch generation with its agent and argv, so a
missing `LatestLaunchProfile` is recoverable from the ledger rather than fatal.
Landing that here would widen M1 into a repair path with its own write
transaction, so it goes in the same family as #275 and #276. Task 12 records it
as a lesson so the next person meets the argument, not just the refusal.

**Unrecorded agents are #276.** Reporting a couch-tagged session with no record
needs a `repos/*` enumeration `DetachedSessionResolver` deliberately lacks, a new
seam and fake, and a new projection field — for a report-only row. Additive, not
corrective. #272's corresponding Done-when transfers there.

---

## Revisions

### 2026-09-16 (fourth) — plan-quality gate, round 1

Five findings; three blocking. All verified against current code before fixing.

- **PQ-1 (Critical), `classification-not-authority` — and it is the finding the
  third revision claimed to have dissolved.** It dissolved it for the
  *classifier* only. `DecideResume` (`resume.go:98`, `:104`) still refuses on
  `record.Park` and `occupiedIncarnation`, and no task touched `resume.go` — so
  post-M1 a #272 record classifies `detached`, `SelectResumableRoot` ranks it
  highest and auto-selects it, the menu offers resume, and resume refuses. The
  fix is the **class**, not the site: a new *Two questions, two authorities*
  section, `occupiedIncarnation` deleted, and **Task 8a** giving resume the same
  treatment Task 8 gives archive. Fixing archive alone would have been the
  instance (ARCH-PURPOSE).
- **PQ-2 (Important) — narrowing `Live` strands the CLI.**
  `ThreadInventoryContext` passes `nil` observations
  (`threadinventory.go:97-100`) *by design*, so the
  console-children ∪ `ObserveRecordedProcesses` union is `couch --list`'s only
  liveness proof; narrowing it re-creates #181's "one store, two stories". `Live`
  now keeps the union and becomes **positive-only** — the union was never the
  bug, reading its *absence* as death was.
- **PQ-3 (Important) — `ArchivableState` cannot be evaluated inside the store
  lock.** `threadstore.go:1106` sits in `s.withLock` with a decoded record and no
  evidence. **Task 8b** gives the store a record-only integrity guard and states
  the consequence the gate asked for: post-M1 `ArchivableState(detached, "")` is
  true, so the store stops independently refusing a record with a live
  incarnation, and that refusal moves to the layer holding the evidence.
- **PQ-4 (Minor)** — `AllThreadStates()` does not exist; added as a step beside
  `AllThreadReasons()`. Task 6's range corrected to `observeRecoverySession:63-68`,
  where the absent-binding error is raised rather than propagated.
- **PQ-5 (Minor)** — the budget measured startup, but Task 1 makes **every
  refresh** pay one `list-sessions`. Both figures now budgeted, with the note
  that the refresh runs in a worker goroutine coalesced by generation
  (`console_menu.go:83-113`), so it costs latency nowhere the operator waits.

### 2026-09-16 (third) — over-engineering audit; re-cut around the operator's rule

Reason: the operator observed that detach has no external effect couch's own
death does not already have, so *"if we missed that recording step, we shouldn't
declare threads unrecoverable"*, and asked whether the plan was over-engineered.
Verified in code (`detach.go:91-99`) — it is. Deltas:

- **The rule inverted from addition to deletion.** Instead of teaching the
  classifier to interpret bookkeeping more carefully, it stops reading it.
  `Incarnation` and `record.Park` leave the classification path; one task (Task 2)
  now fixes both #271 and #272.
- **Four milestones → three, 18 tasks → 12.**
- **Dropped the re-adoption task.** You do not re-adopt what you never disowned —
  it existed only because the stale incarnation was still an authority.
- **`SessionObservation` four states → three.** Under optimistic inventory the
  refresh can never emit `SessionHeldElsewhere`, and the action path that needs
  attach state re-observes anyway. A value no producer can emit is a state that
  exists only to be handled.
- **`startInFlight` deleted rather than narrowed.** A start in flight is
  couch-local in-memory knowledge, delivered as an observation like the pty
  children — not inferred from a durable `creating` incarnation that outlives the
  process it describes.
- **Three-valued liveness demoted from foundation to guard.** It was load-bearing
  only while the process was a classification authority; its real consumers are
  the destructive paths. Was M1's whole milestone, now Task 9.
- **Split out #275** (park as an ordered idempotent write) and **#276** (surface
  unrecorded agents), both depending on this issue.
- **Added Task 5.** `DecideRecovery:33` refuses on an open park *before*
  `archivableRecord`, so Task 2 alone would have changed the label and left the
  gesture broken.

### 2026-09-16 (second) — two fresh-context plan reviews

Both reviewers, over disjoint halves, converged on one defect: re-basing liveness
onto the session without retiring the stale incarnation left #272's rows
auto-selected by `SelectResumableRoot` and offered a resume `DecideResume`
refuses — strictly fewer working gestures than today. The third re-cut above
dissolves that finding rather than fixing it: with the incarnation out of the
classification path, there is no stale claim to retire.

Also from those reviews, and **retained** in the re-cut: `RecoverActiveParks` is
already wired (`couchcmd/run.go:348`) so the unused `ReconcileActiveParks` is
deleted not wired (now #275's concern); adding a `ThreadReason` hard-fails three
guards (Task 3); `DecideRecovery` refuses before `archivableRecord` (Task 5); the
mutation-door guard must be file-scoped and the arithmetic is 12+3+3 (Task 11);
test fixtures must use `actionableTestThread`; `SessionUnresolved` must not
collapse into absence.

Superseded by the re-cut: the `observeParkOwners` channel (dropped — `ParkIdentity`
is copied from `soleParkableIncarnation`, the same process the incarnation
carries), and the careful park/session branch ordering (moot — neither is read).

### 2026-09-16 (first) — initial plan

Written against the measurement that the zellij server is PPID 1 at birth.
