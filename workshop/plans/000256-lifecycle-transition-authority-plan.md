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

Re-derived against the code at the M1 boundary (see `## Revisions`, 2026-09-17
round 2) — every row's name and path grepped, rather than left asserting what the
plan intended.

| Name | Lives in | Status | Landed |
|------|----------|--------|--------|
| `SessionState` / `SessionObservation` | `cmd/internal/couchcore/sessionevidence.go` | new | M1 |
| `ProjectSessionPresence` | `cmd/internal/couchcore/sessionevidence.go` | new | M1 |
| `indexSessionsByName` / `uniquelyClaimed` | `cmd/internal/couchcore/sessionevidence.go` | new | M1 |
| `ThreadEvidence` | `cmd/internal/couchcore/actionableinventory.go` | modified | M1 |
| `ClassifyThread` | `cmd/internal/couchcore/actionableinventory.go` | modified | M1 |
| `startClaimed` | `cmd/internal/couchcore/actionableinventory.go` | new | M1 |
| `ThreadReason` | `cmd/internal/couchcore/threadreason.go` | modified | M1 |
| `liveProofMatches` | `cmd/internal/couchcore/actionableinventory.go` | deleted | M1 |
| `startInFlight` | `cmd/internal/couchcore/actionableinventory.go` | deleted | M1 |
| `occupiedResumeCode` | `cmd/internal/couchcore/resume.go` | deleted | M1 |
| `ArchivableState` | `cmd/internal/couchcore/thread.go` | new | M3 |
| `AllThreadStates` | `cmd/internal/couchcore/actionableinventory.go` | new | M3 |
| `archivableRecord` | `cmd/internal/couchcore/thread.go` | deleted | M3 |
| `occupiedIncarnation` | `cmd/internal/couchcore/thread.go` | deleted | M3 |

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

- **`startInFlight` is deleted and replaced by `startClaimed`.** An earlier
  draft said a start in flight is in-memory knowledge and "ephemeral state stays
  ephemeral". The code reads the durable `ThreadStartClaim` instead — adopted,
  not reverted, because a start claim genuinely IS couch's record of its own
  operation, and plumbing a second observation channel for a value couch already
  writes down buys nothing. See the 2026-09-17 round-1 Revisions entry for the
  bound and the residual risk.

### Integration points

| Name | Lives in | Status | Wraps | Landed |
|------|----------|--------|-------|--------|
| `ProcOps` | `cmd/internal/couchcore/procops.go` | unchanged | process table | — |
| `ObserveRecordedProcesses` | `cmd/internal/couchcore/actionableinventory.go` | modified | `ProcOps` | M3 |
| `SessionPresenceResolver` | `cmd/internal/couchcore/sessionevidence.go` | new | the seam | M1 |
| `…CollisionChecker.SessionPresence` | `cmd/internal/couchcore/artifactcollision.go` | new | `zellij list-sessions` | M1 |
| `resolveScopedBindings` | `cmd/internal/couchcore/artifactcollision.go` | new | session-name index | M1 |
| `retireDeadIncarnationBeforeStart` | `cmd/internal/couchcore/resume.go` | new | `ThreadStore` | M1 |

An earlier draft named `observeSessions`, which the code never shipped — the
resolver is an interface plus a method on the existing checker, because that is
where the index read already lived.

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

- **`SessionPresence`** — **optimistic inventory, strict action** (operator
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

- [x] **Step 1: Write the failing test** — a table over the three values,
  including the two that today collapse into "no observation":

```go
{"index unreadable",   …, want: SessionUnresolved},
{"binding contested",  …, want: SessionUnresolved},
{"no live session",    …, want: SessionAbsent},
{"live session bound", …, want: SessionPresent},
```

- [x] **Step 2: Run it and watch it fail** — the type does not exist.
- [x] **Step 3: Implement.** Preserve `ProjectDetachedSessions`' fail-closed
  uniqueness rules (`claims == 1`, no duplicate rows) — those stand in for #272's
  "exact `ProcessIdentity` match", since a session *name* carries no start token.
  State that equivalence in the doc comment rather than dropping #272's bullet.
  Gather for **every** record, not only resume-shaped ones — the `resumeShaped`
  gate at `:460` is why a record carrying an incarnation never got asked about.
- [x] **Step 4:** `go test ./cmd/internal/couchcore/` → PASS.
- [x] **Step 5: Commit** — `#256 M1: session existence is evidence for every record`

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
rules. The branch order is **not restated here**. It moved twice during the boundary
rounds — `SessionUnresolved` was above `VerifiedPark` and had to be inverted —
and each time this table became a set of instructions to undo the fix. A
hand-maintained restatement of the model is a deferred consumer (ARCH-PURPOSE).

The model is `ClassifyThread`, and the enumeration that proves it total is
`classify_test.go`'s `everyThreadShape` plus
`TestEveryReasonIsProducedBySomeShape`. Read those.

What this task commits to, which is stable:

`record.Incarnations` liveness fields and `record.Park` appear **nowhere** in
`ClassifyThread`. The one exception is `startClaimed`, which reads
`Incarnation.Start` — couch's record of its own in-flight operation, not a claim
about an external process. Rows 7–9 read only
resume authority, which is genuinely durable.

- [x] **Step 1: Write the two failing tests — the operator's actual rows**

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

- [x] **Step 2: Run them and watch them fail** — `busy` and
  `unusable/stale-incarnation` respectively.
- [x] **Step 3: Implement** the branch table above. Delete `liveProofMatches` and
  `startInFlight`. Delete the disproved comment at `:21-23` ("ThreadBusy … it
  resolves on its own") — its twin in `menu.go` goes in Task 4.
- [x] **Step 4:** Run the suite. Existing stale-incarnation and busy tests will
  fail; restate each expectation **with the reason in the test name or a
  comment**. Do not weaken an assertion to make it pass — a test that cannot be
  restated is evidence the rule is wrong.
- [x] **Step 5: Commit** — `#256 M1: recoverability is a fact about the session`

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

- [x] **Step 1:** Run the three guards; let them name the orphaned reasons.
- [x] **Step 2:** Remove what is genuinely unreachable. **Both** are removed,
  including `ReasonUnrecordedChild` — an earlier draft said keep it for #276, but
  a vocabulary entry with no producer is exactly what
  `TestEveryReasonIsProducedBySomeShape` forbids, and an exemption would silence
  that guard for every future orphan. #276 re-adds it with its producer.
- [x] **Step 3:** Re-run → PASS.
- [x] **Step 4:** Commit, then `sdlc milestone-close --issue 256 --milestone M1`.

---

## Chunk 2: M2 — make the rows reachable and prove it

### Task 4: A start nobody is driving is not a start in flight

**Files:**
- Modify: `cmd/internal/couchcore/actionableinventory.go` (`ThreadEvidence`, `ClassifyThread`, `gatherThreadEvidence`)
- Test: `cmd/internal/couchcore/classify_test.go`, `cmd/internal/couchcore/startclaim_test.go`

**Re-scoped 2026-09-17.** The original premise — "after Task 2 that row is no
longer `busy`, so the branch is dead code" — is **false**. `ThreadBusy` survived
M1 with a new producer, `startClaimed`, so the busy row needs an **escape**, not
a deletion. Step 3b already landed in M1: both renderers say *starting*, not
*parking*.

`startClaimed` is the one piece of bookkeeping M1 deliberately left in the
classification path, justified because a `ThreadStartClaim` "is recoverable on
its own terms". It carries `{OwnerPID, OwnerIdentity}` — the couch that began
the transaction — precisely so those terms can be checked. Nothing checks them.
So a couch that dies mid-start leaves a row that reads `starting…` forever and
offers only `name` and `describe`: no detach, no park, no archive, no resume.
`retireDeadIncarnationBeforeStart` bails on it too (`resume.go:578`, "not
debris"). That is the #271/#272 wedge in its last hiding place — bookkeeping
outliving the process it describes.

**The rule: a start is in flight while its claiming couch is alive or
unprovable. A claim whose owner is provably `Dead` has no driver, and the row
must report what the world shows instead.** `Unknown` fails closed, as
`ReconcileStart` already does ("Unknown evidence always keeps capacity
occupied").

Two domains meet here and the gap must fail closed: `startClaimed` is true when
**any** incarnation carries a `Start`, while `CurrentStartTransaction` resolves
only when **exactly one** does. Multiple tracked starts therefore leave
`StartOwner` at its zero value — `Unknown` — and the row stays `busy`. That is
the right answer (ownership is unresolved), but it is an answer by construction
rather than by intent, so it gets its own test.

- [x] **Step 1: Write the failing tests.** Three, against `ClassifyThread`
  directly, plus one through `gatherThreadEvidence`:

```go
// A start claimed by a couch that is gone, whose session survived, is detached
// -- not "starting" forever.
func TestDriverlessStartClaimClassifiesFromTheWorld(t *testing.T) { … }

// Fail closed: an owner we could not probe keeps the row busy.
func TestUnprovableStartOwnerKeepsTheRowBusy(t *testing.T) { … }

// The normal case must not regress: this couch's own in-flight start is busy,
// and stays busy BEFORE the launcher has a pid or a session.
func TestOwnStartInFlightStaysBusy(t *testing.T) { … }

// The domain gap: two tracked starts cannot resolve an owner, so the row stays
// busy rather than being released by a zero value read as Dead.
func TestMultipleTrackedStartsKeepTheRowBusy(t *testing.T) { … }
```

- [x] **Step 2:** Red — all three release cases currently return `ThreadBusy`.
- [x] **Step 3: Add the evidence.** `ThreadEvidence.StartOwner Liveness`. Its
  zero value is `Unknown` (`procops.go:27`), so a record no gather branch
  reached cannot claim its owner is dead — the same fail-closed-by-construction
  shape as `SessionUnresolved`. `gatherThreadEvidence` fills it for every record
  via `CurrentStartTransaction` + the existing `observeExactProcess` helper; a
  resolution error leaves it `Unknown`.
- [x] **Step 4: Replace the branch.** `startClaimed(record)` becomes
  `startInFlight(record, evidence)` — claimed **and** `evidence.StartOwner !=
  Dead`. Keep it ahead of the live branch: a start this couch is driving still
  has no session yet, and the comment explaining that ordering stays true.
- [x] **Step 5:** Green.
- [x] **Step 6: Mutation-check.** Invert the probe (`== Dead`) and confirm
  `TestOwnStartInFlightStaysBusy` fails; drop the fail-closed arm and confirm
  `TestUnprovableStartOwnerKeepsTheRowBusy` fails. A guard nothing pins is not a
  guard (M1, round 6).
- [x] **Step 7: Commit** — `#256 M2: a start nobody is driving is not in flight`

### Task 4a: The released row's archive must actually work

**Files:**
- Modify: `cmd/internal/couchcore/detach.go:226` (`ArchiveThread`)
- Test: `cmd/internal/couchcore/archive_test.go`

Task 4 alone moves a driverless row out of `busy` and, when its session is also
gone, into `unusable/session-gone` — where `menuActionsFor` offers `archive`.
That archive **fails**: `archivableRecord` refuses on `occupiedIncarnation`,
which counts `creating`. Offering an action that always fails is the exact
anti-pattern the menu's own comments name, so the two halves ship together.

The rollback already exists and is correct — `rollbackTrackedStart` deletes the
claim and releases the capacity — and, as in Task 6, it is simply unreachable
from here. Same shape as `retireDeadIncarnationBeforeStart`: screen before
writing, and authorize the write with a probe of the entity it acts on.

- [ ] **Step 1: Write the failing test** — a record carrying a start claim whose
  owner is dead and whose helper is dead is archivable; one whose owner cannot be
  probed is refused, **with a diagnostic naming what could not be proved**.
- [ ] **Step 2:** Red — "is creating; park or detach it before archiving".
- [ ] **Step 3:** Before archiving, roll back a start claim whose owner is
  provably `Dead` **and** whose helper is not `Live`. Probe each entity
  separately — the owner couch and the helper are different processes, which is
  the round-4 lesson. Refuse with a code otherwise.
- [ ] **Step 4:** Green.
- [ ] **Step 5: Commit** — `#256 M2: archive can clear a claim with no claimant`

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

### Task 6b: The ledger is cold-resume authority; the park receipt is not

**Files:**
- Modify: `cmd/internal/couchcore/actionableinventory.go` (`gatherThreadEvidence`, `ClassifyThread`)
- Test: `cmd/internal/couchcore/classify_test.go`

**Added 2026-09-17**, answering both questions the M1 continuation left open —
they are one fact seen from two sides, and the operator chose the unified rule.

`ResolveEstablished` is the ledger read that answers *"is there a native
conversation to resume into?"* Today it runs only when `record.VerifiedPark !=
nil` (`actionableinventory.go:522`). So:

- A thread whose session is gone but whose `ledger-<tag>.jsonl` still names a
  resolvable conversation reads `session-gone` — **archive-eligible** — and
  nobody asked. M2's other tasks make archive *easier* to reach, so this is the
  safety half and ships with them.
- `VerifiedPark` is a **receipt**. It carries a `ParkIdentity`, not a native id.
  Letting it decide `parked` is bookkeeping deciding recoverability — the exact
  thing #256 exists to stop, surviving one level below where M1 cut.

**The rule: cold-resumability is a fact about the ledger.** Ask
`ResolveEstablished` for every resume-shaped record whose session is not
present, and let the answer — not the receipt — produce the state. `VerifiedPark`
leaves the classification path entirely, the way `record.Park` and the
incarnation liveness fields did in M1.

The three-valued discipline carries over unchanged: a ledger that could not be
read is `ProofUnresolved` → `unusable/unknown`, never `session-gone`.

- [ ] **Step 1: Write the failing tests.**

```go
// The safety half: a resolvable conversation is not "gone", whatever the
// bookkeeping says about parks.
func TestSessionAbsentWithResolvableLedgerIsResumable(t *testing.T) { … }

// The collapse: a park receipt whose ledger no longer resolves is NOT parked.
func TestParkReceiptWithoutALedgerEntryIsNotParked(t *testing.T) { … }

// Fail closed: an unreadable ledger is unknown, never session-gone.
func TestUnreadableLedgerIsUnknownNotGone(t *testing.T) { … }

// Cost guard: the extra read happens only when the session is NOT present.
func TestLiveAndDetachedRowsAskNoLedgerQuestion(t *testing.T) { … }
```

- [ ] **Step 2:** Red.
- [ ] **Step 3: Widen the gather.** Resolve the binding for every resume-shaped
  record whose `Session.State != SessionPresent`, not only the parked ones.
  Reuse the existing `resumable`/`ParkedStatus` plumbing rather than adding a
  second channel — the question is the same one, asked of more rows.
- [ ] **Step 4: Delete the `record.VerifiedPark != nil` gate** from
  `ClassifyThread`; the proof status and the observation decide.
- [ ] **Step 5:** Green, and re-run the refresh call-count guard
  (`countingArtifacts`, `classify_test.go:516`) — the ARCH-CONSTRAINTS budget is
  what bounds this widening.
- [ ] **Step 6: Mutation-check** the fail-closed arm, then update the
  `everyThreadShape` table so the new rows are part of the totality claim rather
  than beside it.
- [ ] **Step 7: Commit** — `#256 M2: the ledger decides resumability, not the receipt`

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

> **LANDED IN M1.** Pulled forward: M1 cannot be green while a guard contradicts
> the classification it feeds, and the acceptance suite proved it. The red state
> below names `resume-creating`, a code deleted in the same milestone when its
> only producer went. Kept for the record; do not execute it again.

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

### 2026-09-17 — M2 opening: Task 4 re-scoped, two tasks added

**Task 4's premise was false, and the M1 close is what disproved it.** The task
said the `ThreadBusy` menu branch becomes dead code after Task 2, so both the
branch and its comment should go. But `ThreadBusy` *survived* M1 with a new
producer — `startClaimed` — so deleting the branch would have left a live state
with no menu treatment. The row needs an **escape**, not a deletion. Step 3b of
the old task (re-word both renderers) already landed in M1; the rest is replaced.

**What the escape is.** `startClaimed` is the one piece of bookkeeping M1 left in
the classification path, on the grounds that a `ThreadStartClaim` is "recoverable
on its own terms". Those terms are `{OwnerPID, OwnerIdentity}`, and nothing
consulted them — so a couch dying mid-start left a row reading `starting…`
forever, offering neither archive nor resume. That is #271/#272's wedge in its
last hiding place. Task 4 now probes the owner and fails closed on `Unknown`,
exactly as `ReconcileStart` already does.

**Task 4a added, because half an escape is the anti-pattern.** Releasing the row
from `busy` lands it on `archive`, and `archivableRecord` refuses a `creating`
incarnation — an action that always fails. The rollback exists
(`rollbackTrackedStart`) and is unreachable from archive, the same shape as Task
6. The two halves ship together or neither does.

**Task 6b added**, resolving both questions the M1 continuation left open. They
turned out to be one fact: `ResolveEstablished` — the ledger read that answers
whether a resumable conversation exists — runs only for records carrying a
`VerifiedPark`. So a session-gone row with a live conversation in its ledger is
archive-eligible and unasked (the safety half), while a park *receipt* carrying
no native id is what decides `parked` (the authority half). One rule covers
both: cold-resumability is a fact about the ledger, and `VerifiedPark` leaves the
classification path the way `record.Park` did in M1. The operator chose this over
shipping only the safety half. Deletion, not addition — the rule that re-cut this
plan the first time.

M2 is now Tasks 4, 4a, 5, 6, 6b, 7.


### 2026-09-17 — M1 boundary review, round 5 (REWORK)

**The same validator exception, at a different incarnation count.** Round 4 found
a foreign-owned park; round 5 found an open park with **zero** incarnations. Both
are `threadrecord/lifecycle.go`'s `replacementUnknown` escape, and both wedged
`couch` in the whole tree with an uncoded refusal.

The cause is not the shapes — it is that round 2's rule, *"every guard refusing
on `record.Incarnations` or `record.Park`"*, was **written into the code as four
sites rather than as that predicate**. A predicate covers shapes nobody thought
of; a list covers the ones someone did.

So the clearing pass is now **total over the shapes `validateLifecycle`
accepts**, which is the right domain because ARCH-SECURE treats a record from
another version as untrusted input. Screening completes before any write (round
4's rule), each write is authorized by a probe of the entity it acts on (round
4's other rule), and `TestReAdoptionExitsAreTotalAndCoded` gained an
incarnation-count dimension — mutation-proven against the old count bail.

Also: **the plan stops restating the branch table.** It moved twice during these
rounds, and each time the restatement became instructions to undo the fix. The
model is `ClassifyThread` and the enumeration that proves it total is
`everyThreadShape`; a hand-maintained copy is a deferred consumer
(ARCH-PURPOSE). Same treatment for the atlas passage that still carried round 4's
disproved premise, and for the outer comment in `resume.go` that restated rules
ten lines above their own correction — **a claim restated away from its test is
how two of them came to be wrong.**

### 2026-09-17 — M1 boundary review, round 4 (REWORK)

**The "unrepresentable" claim in round 2 was false, and it mattered.** I deleted
the park-identity guard on the grounds that `validateLifecycle` makes a
foreign-owned park impossible — read from its main clause, missing the exception
one line above: zero matches ARE permitted when the phase is `unknown` and the
transaction carries a `replacement_incarnation` failure, which `park.go`
produces. So the code probed the incarnation and wrote a permanent tombstone
about a different process, one that could be alive and mid-park; its own
`FinalizePark` would then fail forever and #275's audit trail for it would be
gone.

Two rules from it:

- **An irreversible step's precondition is proved about the exact entity the
  step acts on.** Two entities, two probes.
- **A guard omitted as "unrepresentable" must cite the validator clause that
  makes it so, read including its exceptions, and be pinned by a test that tries
  to build the fixture through the real store.** `TestForeignOwnedParkIsRepresentableAndRefused`
  does exactly that — and the store accepted the record, which is the proof the
  claim was wrong.

Also this round:

- **A diagnostic code is a claim.** `ResumeNotRunning` ("not running at all --
  the OPPOSITE of ResumeLive") was being emitted on an exit reached when couch
  could not tell, so the background reattach pass would render "resume-not-running"
  over a live conversation. Now `ResumeUnknown`, and a test pins which code each
  exit emits — there was none.
- Six comments asserting behaviour the code no longer has, swept as one
  enumeration. The mechanism, since three rounds of hand-enumeration each missed
  sites: **a comment asserting what a path classifies names the test that pins
  it**, and a comment enumerating callers points at the one home rather than
  restating the list.
- **The re-derivation rule widened from the Core-concepts tables to every
  normative statement in the plan** (round 2 scoped it too narrowly). Three task
  bodies were directing work the boundary had already reversed: the branch table
  ordered `SessionUnresolved` above `VerifiedPark`, which is the bug round 1
  fixed by inverting them; Task 3 said keep `ReasonUnrecordedChild`; Task 8a
  described work pulled into M1 and named a since-deleted code as its red state.

### 2026-09-17 — M1 boundary review, round 3 (FIX-THEN-SHIP)

Round 2's Rule 1 was itself wrong, and the review caught it. Forcing every
producer to carry a `ResumeDiagnosticCode` **changed what the value means** —
from "is a structured refusal" to "came out of resume" — so every reader that
used the distinction broke (the background reattach pass rendered
`resume-unknown` instead of the error's first line), and rebuilding the error
from `retErr.Error()` meant `errors.As`, `errors.Is` and `Unwrap` could no
longer see through it.

**The rule belonged at the consumer, not on every producer.** `startupResumeRefusal`
now decorates *any* failure — which is what the operator needed all along — with
a message that is TRUE of the failure in hand: a structured refusal names the
thread couch found, while an internal failure says couch could not tell, because
claiming to have found a resumable thread in a store it could not read would be
a lie. The code goes back to meaning exactly one thing.

The general rule, which the review stated and is worth keeping: **when a value's
meaning changes, enumerate every reader and re-derive each in the same round.**

Also this round:

- `SessionPresenceResolver` and its siblings now have **compile-time bindings**.
  They are reached by type assertion on `c.Artifacts`, which fails *silently*:
  drop a method and presence is simply never gathered, every thread reads
  `unknown`, and the result is indistinguishable from a host that could not be
  asked.
- `ResumeDiagnosticCode` gained the **produced-by guard** `ThreadReason` has had
  all along — its absence is why deleting `occupiedResumeCode` orphaned
  `ResumeCreating` in the same commit, unnoticed. The orphan is deleted and the
  guard derives its identifiers from the declaration, so it cannot be satisfied
  by forgetting to update it. Mutation-proven.
- The atlas said "one class, three sites" while the code and plan said four.

**A flake to watch, not a regression:** `TestParkCoordinatorConstructorDoesNotQueryPairSession`
failed once inside a full `make test` ("New blocked before returning: <nil>") and
passes at the base commit, three times in isolation, and on the full re-run. It
is timing-sensitive around `couch.New`; worth a look if it recurs.

### 2026-09-17 — M1 boundary review, round 2 (REWORK): rules, not sites

The gate's own summary was the finding: *"Not converging: fix rules, not
instances."* Round 1's ten findings were disposed, but two of the fixes were
site-shaped and round 2 reproduced the same wedge through exits they did not
cover. Three rules replace them:

- **Every error leaving `ResumeContextWith` carries a `ResumeDiagnosticCode.`**
  `startupResumeRefusal` decorates only coded errors, so an uncoded one refuses
  `couch` in the whole tree with an internal message. Round 1 coded the store's
  retire error; round 2 reproduced the identical wedge through
  `CommitStartClaim`, with `resolveRepoIdentity`, `Proc.Current`,
  `allocateStartNonce` and two observe errors still bare. Hand-enumerating exits
  is what kept missing one, so the rule now lives at the function boundary as a
  single deferred wrap above every early return. Mutation-proven.
- **An irreversible step never precedes a revocable check.** `AbandonPark`
  appends a permanent tombstone, and it was running before `RetireIncarnation`'s
  own preconditions were screened — so an `IncarnationUnknown` record (which
  `soleParkableIncarnation` explicitly permits parking) lost its park forever and
  then failed the retire on every retry, leaving a thread that could be neither
  resumed nor archived. All of the retirement's preconditions are now screened
  first. Mutation-proven.
- **A fail-closed guard is pinned by a table over its exits, not by a test per
  incident.** One table over {park shape} × {liveness} × {incarnation state}
  asserts a coded refusal and unchanged durable state for every combination, and
  fails whenever a new exit appears. Writing it found that the park-identity
  mismatch branch was **unreachable** — `validateLifecycle` requires an active
  park's identity to match an incarnation, and that path requires exactly one —
  so the guard was deleted rather than tested: validation already owns it.

Also: the Core-concepts tables above are re-derived against the code (they named
an `observeSessions` that never shipped), and the rule is to re-derive them at
each milestone close rather than let the plan assert what the code does not
deliver.

### 2026-09-17 — M1 boundary review, round 1 (REWORK)

Ten findings, eight blocking. The two that change the plan rather than the code:

- **The class has FOUR sites, not three.** `RetireIncarnation` refuses while a
  park is open (`threadstore.go:551`), and the M1 re-adoption fix made a
  park-open record reachable for the first time — so the store's precondition
  went live and `couch` refused to start in that tree with a raw store message.
  The reviewer reproduced it end to end. This is the instance-vs-class failure
  ARCH-PURPOSE names: the Log enumerated three sites and stopped, and the fourth
  became reachable *because of* the fix for the third. `retireDeadIncarnationBeforeStart`
  now abandons an orphaned park when the same probe proves its owner dead — the
  park identity is copied from the incarnation, so one probe answers both — and
  every residual failure is wrapped in a `refuseResume` diagnostic, because
  startup only decorates coded refusals.
- **`ThreadBusy` is produced by a durable `ThreadStartClaim`, not an in-memory
  observation.** The plan said *"ephemeral state stays ephemeral"* and the branch
  table's row 3 says "(in-memory observation)". The code reads
  `Incarnation.Start != nil` off the record. **Adopted, not reverted**: the
  alternative is plumbing a second observation channel through `Couch.Spawn` and
  the console for a value couch already writes down, and a start claim is
  genuinely couch's record of its *own* operation rather than a claim about an
  external process. The bound: `reconcileInterruptedStarts` runs at
  `couch.New` (`couch.go:153`). The residual risk the reviewer names is real —
  `ReconcileStart` keeps a claim occupied on Unknown evidence
  (`starttransaction.go:186-210`), so such a record reads `busy` indefinitely —
  and **M2's Task 4 is re-scoped accordingly**: its stated premise ("after Task 2
  that row is no longer busy, so the branch is dead code") is false, so the busy
  row needs an escape rather than a deletion.

Also folded in, with mutation checks where the review found a guard unpinned:
`SessionUnresolved` no longer masks a parked row's durable cold-resume authority
(one failed `list-sessions` was demoting every parked thread in the store); the
`Dead`-only re-adoption gate is now pinned by a test proven red under mutation;
the production `SessionPresence` seam is tested through the stubbed-`zellij`
harness, including the readable-vs-unreadable-scope branch that decides
archive-eligibility, also mutation-proven; the shared fail-closed *rule*
(`indexSessionsByName` + `uniquelyClaimed`) is extracted, the read having been
extracted already; and the docs half of the boundary is finished — three atlas
passages and two README claims still presented the retired vocabulary and the
park-diagnostic behaviour as current.

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
