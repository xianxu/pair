---
collaboration-mode: discussion
---

# Slot lifecycle: one mechanism, three primitives, three outcomes

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every slot lifecycle path lands deterministically in Continue, Resume from disk or Start fresh, through
one mechanism (`Converge`) built from three primitives: proven teardown, binding by the transcript ID stored on
the record, and a persisted failure record.

**Architecture:** A per-slot *lifetime lease* (a file every process of the slot's tree holds open with a shared
`flock`) turns "everything except the transcript is gone" into a one-step check: an exclusive non-blocking
`flock` succeeds. `Converge(slot, target)` is a pure planner over observed evidence plus a thin executor. Every
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
| orphans hold the lease (title, session-watch, attach client) | Resume / Start fresh | the action's teardown kills them by identity | none |
| two live actors or sessions | Resume / Start fresh, detail names both | the action tears both down | live terminal state |
| park awaiting_completion, success completion present (#683) | Resume from disk | Resume (teardown folds the completion first) | none |
| park awaiting_completion, cleanup never ran | Resume from disk | Resume (teardown re-runs cleanup, needs proof) | scrollback tail if the capture never ran |
| teardown unproven (holder survives SIGKILL, lsof fails, zellij unanswering) | unchanged, annotated with the failure | same action (retry) after the operator removes the named holder | nothing is touched: no archive, no launch |
| launch fails after proven teardown | unchanged, annotated | same action; the partial launch is itself torn down with proof before the failure is recorded | none beyond the failed attempt |
| Couch dies mid-Converge | re-observed on restart: whatever the evidence says | the row's action | none: every step is idempotent and gated on fresh evidence |

**Covered:** every path in the audit. **Not covered, on purpose:** transients from developing Couch while using
it (operator's call); D17 crash-record gaps and D18 helper debug logs (not lifecycle actions; separate items if
wanted); Inferred tail-match recovery stays only as the legacy fallback for records without a binding.

## 2. The three primitives

### P1 — `Teardown(slot) (TeardownProof, error)`: proven teardown of everything but the transcript

**The single resource (tl-worker PLANNING rule).** The invariant "no live process of the old tree when the new
one starts" is a one-step check on a lease held for the tree's lifetime, instead of an observation sweep:

- Couch creates `<slot store dir>/lifetime.lease` (one file per slot) and passes its path to the launch as
  `PAIR_LIFETIME_LEASE`. Couch itself never holds it open past a probe.
- Every `pair` process started for the tree (the pane's `pair wrap`, the attach client Couch runs as its actor,
  `pair title`, the nvim launcher) opens the path and takes `LOCK_SH` (blocking, 2s bound) at start, without
  `O_CLOEXEC`, and passes the fd in `ExtraFiles` to what it spawns: the agent pty child (wrap.go:2918),
  session-watch (wrap.go:2574), the title process and nvim. One helper, `lease.Inherit(cmd)`, at each site.
- The tree is gone iff `flock(LOCK_EX|LOCK_NB)` on a fresh open succeeds (then released at once).
- **Layering:** pair only learns "hold this path for your lifetime"; it has no slot concept and works without
  Couch (no env, no lease). Couch owns the slot meaning (memory: layer dependency direction).

**Why observation alone can't be the proof:** the audit's survivors (D1, D2, #684's title pid 17468 from Oct 14,
session-watch after `Setsid` + `Release`) are processes that left the parent tree, so no PID-tree or record walk
finds them. Anything that inherited the lease is found by `lsof <lease>`, wherever it was reparented.

**The zellij server** is third-party and may close inherited fds, so the lease doesn't cover it; its existing
proof stays: two consecutive absence observations of both session names (launcher/session_quiescence.go:63-95).

**Contract:**

```go
// couchcore/teardown.go
type TeardownProof struct {
    Slot       SlotIdentity
    LeaseFree  time.Time      // the EX|NB acquire that succeeded
    Sessions   []string       // names proven absent twice
    Observed   SlotSessionObservation // ObserveSlotSessions after the kill: Absent == true
}
// Teardown is idempotent: on an already-down slot it costs one flock, one
// ObserveSlotSessions and one list-sessions, and returns a proof.
func (c *Couch) Teardown(ctx context.Context, slot SlotIdentity, op OperationRef) (TeardownProof, error)
```

Steps, bounded at 10s total (ARCH-CONSTRAINTS):

1. **Preserve** (only if a session is live): run the park capture (scrollback) and copy the ledger's current
   native ID onto the record (P2's write point). An open park transaction is folded first: a matching success
   completion with the owner proven dead finalizes it (park_recovery.go:201, #683); otherwise its cleanup runs.
2. **Stop:** quit the zellij session (existing `TriggerBoundQuit`), kill the recorded server and its descendants
   (`PrepareConfirmedTerminal`, recovery_retirement_os.go:22), SIGTERM the slot's actors from the registry.
3. **Sweep:** while the lease is held, `holders := LeaseHolders(path)` (lsof seam); SIGTERM each by exact
   identity, 3s, then SIGKILL, 2s (`awaitExactProcessExit`, detach.go:213). At most 3 rounds.
4. **Prove:** lease free, both session names absent twice, `ObserveSlotSessions(slot).Absent` (slotsessions.go:149
   — kept as the second witness for trees started before the lease existed, and for the registry and zellij).
5. **Clean debris** only after proof: sidecars, the wrapper and notify sockets of the now-dead wrapper pids (D1's
   sockets), stale layouts. Debris errors are recorded in the proof, not ignored (lifecycle.go:387-420).

Any step that can't reach its proof returns `ErrTeardownUnproven{Phase, Holders, Cause}` and nothing else runs.

**Structural enforcement (ARCH-ORDER):** `ArchiveThread`, `archiveThread`, `ReplaceThreadExpected` and
`replaceSlotCurrent` take a `TeardownProof` argument whose `Slot` must match; the Fresh exemption
(threadstore_replace.go:11-16) is deleted (D5). A proof is consumed within the same slot-operation admission
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
  thread.go:86-91 precedent). A binary older than M2 refuses records M2 writes. Accepted, as for #198.
- **Open question for the drill:** whether Claude's or Codex's native ID changes across a resume. Teardown
  re-copies the ledger's root ID every time, so a change is picked up; the drill asserts the ID it observes.

### P3 — `RecordFailure`: every lifecycle failure persists its reason

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
| 12 | Reload-context | `Converge(Fresh, same agent)` | the separate live-restart path; the zellij session is rebuilt (target: rebuild boldly) |
| 13 | Detach | unchanged: the slot stays Continue | — |
| 14 | Stop | await the actor's exact exit before `Forget`; Unknown refuses (D6) | the no-wait `Forget` |
| 15 | Park abandon, reboot archive-only | deleted; covered by Down and Archive | both bookkeeping-only paths |
| 16 | Startup classification | observe only; one of three outcomes, count-aware (D19, D20), last failure attached | the Unknown→Fresh mapping, `ContinuityStarting` as a state |

## 5. Operating envelope (ARCH-CONSTRAINTS)

- **Teardown:** an operator action, budget 10s; one `lsof` per sweep round (≤3), measured ~0.1–0.5s each
  (to confirm in M3). Exceeded → unproven, recorded, stop.
- **Resume bind:** one `stat`. Counted test: with 8 and 16 transcripts in a fake root, `Bind` does 1 stat and 0
  listings (rejects any scan).
- **Classification refresh:** unchanged plus one mtime check of the failure store per refresh. No flock or lsof on
  the refresh path.
- **Lease probe:** acquire+release of a local `flock`, microseconds; never blocking (`LOCK_NB`).

## 6. Fakes and conformance (ARCH-MOCK)

- `LeaseHolders` (wraps `lsof -F p <path>`): a stateful fake keyed by path, with holders that exit on SIGTERM,
  on SIGKILL only, or never. Real `flock` on a temp file in tests, held by real `sleep` children via
  `ExtraFiles` (cheap, exact).
- zellij, process table, agent transcript roots, the failure store: the existing fakes
  (`FakeLaunchFailureStore`, artifact and session fakes) extended, not duplicated.
- Live conformance: the nightly drill (M6) runs the real zellij, Claude and Codex.

## 7. Lifetimes (ARCH-FUNERAL)

- `lifetime.lease`: one empty file per slot, created by the first launch, removed with the slot's store dir.
- `ThreadRecord.Transcript`: one per record; archived with it.
- Failure ring: 64 entries, one file.
- Drill results: one JSON per night under the pair data dir, pruned at 30 days by the drill itself.

## 8. Milestones

Each `Mx` is a review boundary (`sdlc milestone-close`). M1 and M2 are cheap and unblock diagnosis; M3 builds
the proof; M4 fixes the incident paths; M5 finishes the rest; M6 proves it nightly.

- [ ] M1 — Persist every lifecycle failure (P3). Covers D14–D16 and #683's diagnostics half.
- [ ] M2 — Store the transcript binding on the record; Resume binds by one stat (P2). Covers D9, D10, D12.
- [ ] M3 — Lifetime lease and `Teardown` with `TeardownProof`; archive and replace require a proof (P1). Covers
  D1–D8 and #683's finalization half.
- [ ] M4 — `Converge`, and paths 1, 2, 3, 5, 6, 7, 8, 9, 10 onto it. Fixes #684.
- [ ] M5 — Paths 4, 11, 12, 14, 15, 16; classification collapsed to three outcomes with the two-actor anomaly.
- [ ] M6 — Nightly drill on a disposable slot, with aged state; 7 green nights closes the project's done-when.

### M1 tasks

**Files:** modify `couchcore/launchfailure.go`, `launchfailure_store.go`, `launchfailure_capture.go`,
`launchfailure_show.go`, `operationdispatch.go`; `couchtty/menu_operations.go`, `menu_reattach.go`,
`console_presentation.go`; tests beside each.

- [ ] **Test (red):** `TestDispatchPersistsEveryOperationFailure` — table over every declared operation name;
  an executor fake returns an error; assert one `LifecycleFailure` with the operation, slot, phase and message
  lands in `FakeLaunchFailureStore`. Fails today for all but launch-stage errors.
- [ ] **Test (red):** `TestFailureRingKeeps64NewestAndSlotLevelEntries`.
- [ ] Rename `LaunchFailure` → `LifecycleFailure` (type alias for one release is not needed: one package); add
  `Phase`, `Holders`, `Outcome`; allow empty `Tag` with `RepoScope` set; limit 64.
- [ ] Wrap `DispatchOperation`: on `err != nil`, `c.saveLifecycleFailure(failureFromCall(call, err))`; a save
  error joins the returned error (never swallowed). `failureFromCall` is pure, with its own unit test.
- [ ] Phase-carrying errors: a small `PhaseError{Phase, Err}` the park and resume steps wrap their returns in, so
  §3's early returns record which step failed (park.go:762-833, 878, 942).
- [ ] **Test (red→green):** `TestReattachFailurePersists` — the reattach pass's warm-attach failure reaches the
  store (menu_reattach.go:300-313).
- [ ] Row detail and `couch --show` read the latest failure per slot (mtime-cached); success supersedes.
  **Test:** `TestRowShowsLastFailureUntilSuccess`; `TestShowListsFailureAfterReceiptExpiry`.
- [ ] Targeted runs: `go test ./cmd/internal/couchcore/ -run 'Failure|Dispatch'`, `./cmd/internal/couchtty/ -run
  'Reattach|Row'`. Commit `#684 M1: …`.

### M2 tasks

**Files:** create `threadrecord/transcript.go` (+ test); modify `couchcore/thread.go`, `threadrecord/record.go`
(validation), `launcher/createflow.go` (minted IDs reported back), `couchcore/resume.go`,
`couchcore/recovery_prepared.go`, `couchcore/relaunch.go`, `sessioninventory/native_selector*.go` (resume no
longer calls them).

- [ ] **Test (red):** `TestTranscriptBindingValidate` — absolute, clean, under root, basename contains ID;
  relative, `..`, foreign root, mismatched ID all rejected (pure).
- [ ] **Test (red):** `TestResumeBindsByOneStat` — fake root with 8 then 16 Codex and Claude transcripts;
  `Bind` performs exactly 1 stat and 0 listings; a missing file yields `ErrNoTranscript{Reason: missing}`.
- [ ] **Test (red):** `TestLegacyRecordBindsFromLedgerOnceThenStores` — first Resume resolves via the ledger and
  writes `Source: legacy-ledger`; the second performs no ledger query.
- [ ] Add the field, its validation and clone; write at minted launch; Resume, relaunch and recovery Resume read
  only the record (relaunch's pre-park derivation deleted).
- [ ] Teardown's preserve write point lands in M3; until then the park path copies the ledger ID onto the record
  at `FinalizePark` (one call, moved into Teardown in M3).
- [ ] Commit `#684 M2: …`.

### M3 tasks (detailed when M3 starts; a `## Revisions` entry records the detail)

- `launcher/lease` package (pure path validation + thin `Hold`, `Inherit(cmd)`, `ProbeFree`): tests with real
  `flock` and `sleep` children: held by a grandchild after the parent exits → not free; all exit → free.
- Every pair process started for a Couch tree holds the lease; a test enumerates the spawn sites in `wrapcmd`
  and fails when an `exec.Command` there lacks `lease.Inherit` (added to `pkg/golint` if it generalizes).
- `couchcore/teardown.go` with `LeaseHolders` seam and fake; `TestTeardownSweepsReparentedHolders` (a holder
  that survives SIGTERM, one that survives the round limit → `ErrTeardownUnproven` naming it).
- Archive/replace signatures take `TeardownProof`; `TestArchiveRefusesWithoutProof`, `TestFreshReplaceRequiresProof`.
- `TestTeardownFoldsCompletedPark` (#683's retained shape: success completion, owner dead, session absent).

### M4 tasks (detailed when M4 starts)

- `couchcore/converge.go` (pure `PlanConverge`, table-tested over every target × evidence class, including
  two-actor and orphan evidence) and `converge_exec.go`.
- **#684 regression:** `TestFreshAfterFailedResumeTearsDownOrRefuses` — record parked, resume fails, an actor
  and the old session still alive: before, two actors; after, Teardown proves them gone and Fresh starts, or
  (holder unkillable in the fake) Fresh refuses with the holder named and nothing archived.
- Paths 1, 2, 3, 5, 6, 7, 8, 9, 10 rewired; their old gates deleted; the per-path tests already in the tree keep
  passing or are replaced by the Converge table where they restate it.

### M5 tasks (detailed when M5 starts)

- Paths 4, 11, 12, 14, 15 rewired or deleted; `ClassifyThread` returns one of three outcomes plus an anomaly
  count; tab bar shows "2 actors" instead of raw names (D20). `TestTwoLiveActorsNeverContinue`.

### M6 — nightly drill (detailed when M6 starts)

- `scripts/lifecycle-drill` drives a running Couch through its CLI on a disposable slot (default `pair:9`, a
  configurable env var, no hardcoded paths): for Claude and for Codex, start → one trivial turn → Alt+n
  relaunch (assert Continue, same native ID) → park → Resume (same ID) → Start fresh (new ID, old archived) →
  archive. After each step it asserts: one live actor at most, the lease state, and that every failure has a
  store entry.
- **Aged state** seeded before the run: a record without a transcript binding (legacy), a tree started without
  the lease (pre-M3), an orphan lease holder aged by an old start time, and a Codex root with more than 10,000
  session files (via a drill `CODEX_HOME` whose `auth.json` symlinks to the real one — **operator decision**).
- Results as one JSON per night; a summary sent to the hub. Scheduled by a launchd agent (a machine change —
  **hub/operator approval** before installing). Seven consecutive green nights close the project.
- The daily read-only resume check (#680) stays its own issue; the drill reuses its resolver.

## 9. Relationship to other issues

- **#683:** M1 is its diagnostics half and M3's park fold its finalization half. Proposal: fold #683 into this
  vehicle (close it by reference at M3), or run M1 under #683; the hub decides.
- **#681** (admission before inventory) is landing separately and unchanged by this plan.
- **#686** (session-watch busy loop) is separate; M3's lease makes a spinning watcher at least killable by
  teardown.
