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

## 2. The three primitives (in pair#695)

P1 (proven teardown, `Teardown` with `TeardownProof`), P2 (the transcript binding on the record) and §2a (record
compatibility) moved to pair#695's plan on the hub's decision (2026-10-18, delivery 51ef8069), unchanged:
`workshop/plans/000695-lifecycle-primitives-transcript-binding-on-the-record-the-two-kind-address-proven-teardown-plan.md`
(under `workshop/history/` once #695 lands). P3 is #683. This plan uses them as landed on main.

## 3. The mechanism: `Converge`

```go
// couchcore/converge.go (pure)
type Target int // TargetContinue, TargetResume, TargetFresh, TargetDown
type ConvergeInput struct { Target Target; Evidence SlotEvidence; Agent string }
type Step int // StepAttach, StepTeardown, StepBind, StepMint, StepLaunch, StepArchive, StepCommit
func PlanConverge(in ConvergeInput) ([]Step, error)

// couchcore/converge_exec.go (thin IO shell)
func (c *Couch) Converge(ctx context.Context, addr LifecycleAddress, in ConvergeRequest) (ConvergeResult, error)
```

**The address** is pair#695's `LifecycleAddress` (two kinds: slot and team). Converge takes it and never
switches on the kind.

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

**Exact identity through queues (pair:5 handoff, 2026-10-18).** A `ConvergeRequest` carries the exact
`ThreadAddress`, the record `Revision` it was decided on, and an open park's nonce. It is never re-resolved
"by slot" after it leaves the menu, the reattach queue or the CLI. The executor refuses with a recorded reason
(`target moved: revision N → M`) when the record has moved since. Then the row re-projects and the operator
picks again.

**Archive never hides an unresolved park.** Archiving does not establish `VerifiedPark`, and a client's exit
does not establish session death. Teardown's preserve step folds an open park first (finalized on a matching
success completion with the owner proven dead, or its cleanup re-run under proof). If neither proves out,
teardown is unproven, so nothing is archived. `archiveThread` refuses a record whose park is still open.

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

- **Teardown and bind:** their budgets are #695's (§4 there). Converge adds no probe of its own; it re-gates each
  step on fresh evidence from the existing observers.
- **Classification refresh:** unchanged, plus one mtime check of the failure store per refresh. No flock, lsof
  or ps on the refresh path.

## 6. Fakes and conformance (ARCH-MOCK)

- Converge's executor runs against #695's `LeaseHolders` and `TTYProcesses` fakes and the existing zellij,
  process-table, transcript-root and failure-store fakes (`FakeLaunchFailureStore`), extended rather than
  duplicated.
- Live conformance: the drill (M3) runs the real zellij, Claude and Codex.

## 7. Lifetimes (ARCH-FUNERAL)

- The lease, the tty field, the transcript binding and the orphan registry are #695's (§6 there). The failure ring
  is #683's.
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

**Prerequisite, landing separately as #695 (hub decision, 2026-10-18):** the primitives (binding, the two-kind
address, proven teardown), formerly this issue's M1–M2. #695 depends on #683. #684 continues from main after #695
lands. No stacked branches (#272).

**#684's milestones.** Each `Mx` is a review boundary (`sdlc milestone-close`). M1 fixes the incident paths. M2
finishes the rest. M3 proves it with the drill.

- [ ] M1 — `Converge`, with paths 1, 2, 3, 5, 6, 7, 8, 9 and 10 moved onto it. Fixes #684.
- [ ] M2 — Paths 4, 11, 14, 15 and 16; classification collapsed to three outcomes, with the two-actor
  anomaly.
- [ ] M3 — The lifecycle drill on a disposable slot, with aged state, triggered from the hub. Seven consecutive
  green nights close the project's done-when.

### M1 tasks (detailed when M1 starts)

- `couchcore/converge.go`: the pure `PlanConverge`, table-tested over every target and evidence class, including
  two-actor and orphan evidence. Its executor goes in `converge_exec.go`.
- **#684 regression:** `TestFreshAfterFailedResumeTearsDownOrRefuses`. The record is parked, the resume fails, and
  an actor and the old session are still alive. Today that ends with two actors. After the fix there are two
  passing outcomes:
  - Teardown proves them gone and Fresh starts;
  - the fake's holder can't be killed, so Fresh refuses with the holder named and nothing archived.
- Paths 1, 2, 3, 5, 6, 7, 8, 9 and 10 rewired, with their old gates deleted. The per-path tests already in the
  tree either keep passing or, where they restate it, are replaced by the Converge table.

### M2 tasks (detailed when M2 starts)

- Paths 4, 11, 14 and 15 rewired or deleted.
- `ClassifyThread` returns one of three outcomes plus an anomaly count.
- The tab bar shows "2 actors" instead of raw names (D20). `TestTwoLiveActorsNeverContinue`.

### M3 — lifecycle drill (detailed when M3 starts)

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
  - a tree started without the lease (pre-#695);
  - a stray pair process holding the lease, aged by an old start time;
  - a drill `CODEX_HOME` (with `auth.json` symlinked to the real one) holding a copy of the last 2 weeks of
    Codex session files. It is bounded and realistic, not a tree of more than 10,000; #695's counted test is what
    rejects any scan.
- **Cost per run:** one minimal turn per agent, two in all. Each turn's input is the agent's own base context
  (system prompt and tools, on the order of tens of thousands of input tokens, largely cached after the first
  night), and the output is a few tokens. The prompt is fixed ("reply ok").
- **Trigger:** the hub runs it nightly until a Couch schedule exists, then the schedule runs it. No launchd. The
  drill writes one JSON per run and sends a summary to the hub. Seven consecutive green nights close the project.
- The daily read-only resume check (#680) stays its own issue. The drill reuses its resolver.

## 9. Relationship to other issues

- **#683:** it lands P3 first, as its own issue. #695 M2's park fold delivers its finalization half; the #695 M2 close
  notes that for #683's record.
- **#681** (admission before inventory) is landing separately and unchanged by this plan.
- **#686** (session-watch busy loop) is separate. #695's lease at least makes a spinning watcher killable by
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
- 2026-10-18 22:10 — operator: compatibility, not refusal (replaces decision b). Added §2a. The decoders for the
  thread record, the agent sidecar and the orphan registry become tolerant of unknown fields **and preserve
  them through a rewrite**, because older CLI builds run the direct-store writers and would otherwise erase the
  binding. #683 does the same for the failure store. No version bump: no new field is a guard, and teardown has
  no persisted phase. The first tolerant build can't help builds from before it.
- 2026-10-18 22:30 — pair:5's evidence handoff (delivery 566735e7) made two implied invariants explicit: exact address, revision and park nonce carried through queues (refuse when moved), and archive refuses an open park. Tests added to M2/M3. No new state or mechanism.
- 2026-10-18 23:00 — operator scope addition via hub (delivery da27249b), from the introduce-team project: the
  lifecycle address gets two kinds from the start, a slot (`repo:N`) and a team hub (a team folder with a
  committed `.hub`, several teams possibly in one ops checkout, no sdlc claims). `Teardown`, `Converge`,
  `TeardownProof` and the orphan origin take `LifecycleAddress` instead of `SlotIdentity`, and the lease, store
  and failure records are keyed by `Address.Key()`. The two tests land in M1, with the type. This is a type
  change only: no milestone grows, and hub addressing, startup and the switcher stay in pair#694. The existing
  `HubSlot`/`HubDefinition` (report routing) is untouched here; pair#694 decides whether team identity replaces
  it.
- 2026-10-18 23:40 — naming aligned with the introduce-team design and pair#694's plan: the address kind is `team`,
  the arm is `TeamIdentity{CheckoutRoot, TeamDir, Name}` (Name derived from the folder), and `Key()` for a team is
  `team/<scope(CheckoutRoot)>/<TeamDir>`. Rename only; no behaviour change.
- 2026-10-18 23:55 — split (hub delivery 51ef8069): the primitives (former M1–M2: binding, the two-kind address,
  proven teardown, orphan registry) are pair#695, filed by pair:1 and landed from main like any issue, with #683 as
  its prerequisite. §2 and the address type moved to #695's plan unchanged, and this plan points there. #684 depends on
  #695 and continues from main after it lands. Milestones renumbered: old M3 (Converge, paths 1–3, 5–10) → M1,
  old M4 → M2, old M5 (drill) → M3. Revision entries above keep the old numbers.
