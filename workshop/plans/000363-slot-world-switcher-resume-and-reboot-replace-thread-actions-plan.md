# Slot-World Switcher: Resume and Reboot Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Couch's switcher offers each row only what the slot model needs. Live rows get the lifecycle
actions. Parked, detached and unusable rows get two actor operations: **resume** (bring the old conversation
back by any path that works) and **reboot** (archive the old record, start a fresh agent in place). A
repository gets at most one primary (`:0`) thread.

**Architecture:** Two new couchcore operations sit on what already exists. `resume` routes through a pure
`ChooseResumeRoute` to `OpenSlot`, `RetryContinuation`, `RecoverThread` or `ResumeContextWith`. A pure
`ResumeRebootAdvice` decides when a failure should name reboot. `reboot` reuses `Couch.ArchiveThread`'s
admission and quiesce half (extracted as `prepareRetirement`), then follows a pure `DecideReboot`: archive
only, or archive plus a fresh claimed record committed in **one store journal**. Slots already have that
journal (`replaceSlotCurrent`). The main store gains `ReplaceThreadExpected`, built from the same
journal-entry builders that `archiveThread` and `CreateThread` use. On the switcher side, `menuActionItems`'
branches collapse into one pure table, `menuRowActions(menuRowFactsOf(row))`. The one-primary rule widens the
startup and occupancy predicates from an exact working path to the repository scope.

**Tech Stack:** Go 1.26, the existing couchcore fakes (`newTestEnv`: `FakeRunner`, `FakeGit`, `FakeProcOps`,
the artifact fake with `SetSessionPresence`, store `hooks.AfterJournal`), and the couchtty reducer tests.

**Source of truth for scope:**
`workshop/issues/000363-slot-world-switcher-resume-and-reboot-replace-thread-actions.md`. Where they differ,
the 2026-10-02 Log entry ("design decisions before the durable plan") and the Revision ("boundary with
pair#367 settled: actor only") override the original `## Spec`.

---

## Milestones and why they are cut this way

The suggested cut was M1 = action table plus removals, M2 = resume/reboot. This plan cuts **core first**
instead. If the switcher removed archive and the repair entries before reboot and resume existed, unusable
rows would offer nothing between M1 and M2, and the branch would regress at a review boundary. Each milestone
below leaves `main` releasable.

| Milestone | Boundary | Leaves behind |
|---|---|---|
| **M1** | couchcore actor operations: `resume` (unified), `reboot` (`:0` journaled replace, `:1+` fresh, missing-directory archive-only, no name/description carry) | Old switcher untouched; new ops declared `RowAction: false` and reachable only from tests and dispatch |
| **M2** | Switcher speaks the slot model: pure action table, resume/reboot wired with `RowAction: true`, rename/describe/archive/open-slot/fresh-slot/recover-* deleted, labels and matching stop reading stored name/description, README and atlas | Each row offers exactly the Spec table |
| **M3** | One primary per repository: startup resumes the existing `:0`; a console start refuses | No second primary thread from a subdirectory start |

## Resolved spec ambiguities (operator: confirm or correct)

1. **Non-Git directories cannot start today.** `Resolve` (`worktree.go:30`) runs `git rev-parse
   --show-toplevel`, and `resolveRepoIdentity` (`couch.go:393`) runs `--git-common-dir`. Both fail outside a
   repository, so no non-Git row can exist. *Default:* Task 3.4 pins the refusal with a test; the action table
   has no non-Git kind; "stays supported" stays as true as it is today, which is not at all. If the operator
   wants non-Git starts, that is a new feature for a separate issue.
2. **Reboot on a detached row stops its running agent.** The Spec table offers reboot on detached rows.
   `startFreshSlot` refuses unless sessions are absent. *Default:* reboot runs `ArchiveThread`'s existing
   admission and quiesce (`zellij delete-session`, polled) first, and the confirmation says "stops its running
   agent". This is the same contract archive has today.
3. **Live rows with a FAILED continuation (#280)** keep `retry-continuation` and `dismiss-continuation`,
   filtered through `ContinuationRefuses`. Without them, relaunch and switch-agent would stay blocked with no
   switcher exit. Rows that are not live get resume and reboot instead.
4. **Add slot narrows to live `:0` rows**, per the Spec table. Today any repository row offers it
   (`menuActionsFor`). `copy-orientation` stays a transient clipboard affordance after switch-agent; it is not
   a row action in the Spec's sense.
5. **Unknown rows offer reboot.** Reboot re-classifies when it runs and refuses while the state is still
   unknown ("state could not be checked; retry"). `TestActionOfferedImpliesPermitted` pairs reboot with
   `RebootableState`, which admits `unusable/unknown`, and a comment records why.
6. **Recover from an explicit checkpoint path is removed entirely.** The recover-checkpoint text form was its
   only entry, and there is no CLI. A retained checkpoint still comes back through resume, via `RecoverThread`
   or `RetryContinuation`.
7. **Directory missing on `:1+`.** "Directory missing" means `Reason == ReasonPathMissing`, on either kind. A
   `:1+` slot whose environment directory (with `.couch/`) is gone has no record left to archive. Its row
   offers nothing and shows "directory missing — add slot recreates it"; pair#387 repairs the registration.
8. **An unreadable `:0` record reboots archive-only.** Its path cannot be read, so there is no "same path" to
   start in. The result says to start couch in that directory. Slot records stay archive-and-start: the slot
   path is known, and the corrupt bytes already go to `recovery/<sha>.json`.
9. **One primary per repository applies to usable rows only.** An unusable `:0` still does not block a start,
   which keeps the existing anti-lockout rule from `PathHoldsUsableThread`'s comment. Linked worktrees that
   are not slots keep their own scope, so they keep their own row.
10. **Resume failure names reboot only when the transcript cannot come back.** The `ResumeRebootAdvice` map
    covers every `ResumeDiagnosticCode`. Transient refusals (unknown, starting, parking, provisional binding)
    say to retry instead.

## ARCH-* notes

- **ARCH-DRY:** reboot reuses `ArchiveThread`'s admission and quiesce (`prepareRetirement`),
  `startFreshSlot`'s claim loop (`claimFreshRecord`), `spawnResolved`'s launch tail (`launchClaimedThread`),
  and one set of journal builders for archive, create and replace. Resume composes the existing executors and
  does not re-implement warm, cold or adopt.
- **ARCH-PURE:** `menuRowActions`, `menuRowFactsOf`, `ChooseResumeRoute`, `ResumeRebootAdvice`, `DecideReboot`
  and `RebootableState` are pure and get table tests with no fakes. The IO shells are `Couch.ResumeTarget` and
  `Couch.Reboot`.
- **ARCH-PURPOSE:** every consumer of a removed operation name is found by a derived grep (Tasks 2.6 and 2.8),
  not by a hand list. Labels change at the `Label()` sources, so every surface derives from them.
- **ARCH-MOCK:** there are no new external dependencies. zellij goes through the artifact fake
  (`SetSessionPresence`, `SetPairSession`), git through `FakeGit`/`newProvisionFixture`, processes through
  `FakeProcOps`. Crash safety is tested with the store's `hooks.AfterJournal` plus `RecoverStoreJournal`.
- **ARCH-CONSTRAINTS:** reboot and resume run on an operator keypress, off the UI path, through the operation
  queue. Reboot adds one classification round (one host-wide `list-sessions`) beyond what fresh-slot costs
  today. M3 widens `startupAsks` from one path to one scope. That adds cold proofs only for other records in
  the same scope, which is normally zero or one.
- **ARCH-SECURE:** an unreadable record is never grounds to stop a session (`ArchiveThread`'s rule is reused).
  Replace re-reads the store under lock and compares the expected revision or bytes before writing.
- **ARCH-ORDER:** see the reboot sequence table in Task 1.6.
- **ARCH-FUNERAL:** reboot writes one archive record plus one grace file per reboot. The existing 60-day
  archive GC (`archive_gc.go`) removes them. `:1+` corrupt bytes stay under the existing 16-file / 64 MB
  `recovery/` cap. Nothing new is created: `ReplaceThreadExpected` writes no file family that archive and
  create did not already write.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `menuRowFacts` / `menuRowFactsOf` / `menuRowActions` | `cmd/internal/couchtty/menu_actions.go` | new |
| `menuActionItems` (now `menuRowActions(menuRowFactsOf(row))`) | `cmd/internal/couchtty/menu_actions.go` (moved from `menu.go:1322`) | modified |
| `menuActionsFor` (add-slot moves into the table) | `cmd/internal/couchtty/menu_switchagent.go:201` | modified |
| `enterOperationFor` | `cmd/internal/couchtty/menu.go:751` | modified |
| `menuArchiveOffered`, `slotFreshOffered`, `menuLiveActions` | `couchtty/menu.go`, `couchtty/menu_slot.go` | deleted |
| `ResumeRoute` / `ChooseResumeRoute` | `cmd/internal/couchcore/resume_route.go` | new |
| `ResumeRebootAdvice` / `ResumeNoSurvivor` code | `cmd/internal/couchcore/resume_route.go`, `resume.go:14` | new |
| `RebootFacts` / `RebootPlan` / `DecideReboot` | `cmd/internal/couchcore/reboot_decision.go` | new |
| `RebootableState` | `cmd/internal/couchcore/actionableinventory.go` | new |
| `ActionableThreadSummary.Label` / `ThreadSummary.Label` / `threadLabel` / both `DisplaySummary` | `couchcore/actionableinventory.go:220,252`, `couchcore/threadinventory.go:29,36` | modified |
| `ThreadReferenceFields` (`Name`→`Label`, `Description`→`Summary`) / `classifyNormalizedThreadReferenceFields` | `cmd/internal/couchcore/threadmetadata.go:~165` | modified |
| `checkpoint.Exits` wording | `cmd/internal/checkpoint/request.go:37` | modified |
| Operation declarations: `reboot` new; `resume` gains `path`; `open-slot`, `fresh-slot`, `name`, `describe`, `archive`, `recover-thread`, `recover-checkpoint` | `cmd/internal/couchcore/ops.go` | new / modified / deleted |
| `SelectResumableRoot` / `ScopeHoldsUsableThread` (was `PathHoldsUsableThread`) / `startupAsks` | `cmd/internal/couchcore/startup.go:32,92,156` | modified |

- **menuRowFacts / menuRowActions:** the per-row action authority, written as one table over (kind × phase),
  with no branches spread across states.
  - **Relationships:** 1:1 with a rendered row. Consumers are `menuActionsFor`, `enterOperationFor`, the
    confirmation re-check in `reduceConfirmationKey` and the in-flight re-check at `menu.go:1747`. All of them
    ask `containsMenuItem(menuActionItems(row), action)` in place of the `archive`/`fresh-slot` special cases.
  - **DRY rationale:** replaces six return sites in `menuActionItems`, plus `menuArchiveOffered`,
    `slotFreshOffered`, and three op-name-specific guards.
  - **Future extensions:** #364's "remove last slot" becomes one row in the `menuPhaseLive`/`menuRowPrimary`
    arm.
- **ChooseResumeRoute:** maps (slot?, record present?, continuation phase) to one existing executor. It is
  pure: the shell reads the record and passes its facts in.
  - **Future extensions:** pair#367's bulk resume calls `Couch.ResumeTarget` per slot.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ThreadStore.ReplaceThreadExpected` | `cmd/internal/couchcore/threadstore_replace.go` | new | store journal (filesystem) |
| `archiveJournalEntries` / `createJournalEntries` (extracted) | `cmd/internal/couchcore/threadstore.go:214,1290` | new (extracted) | store journal |
| `ThreadStore.archiveThread` / `CreateThread` | `cmd/internal/couchcore/threadstore.go` | modified (use the builders) | store journal |
| `Couch.prepareRetirement` (extracted from `Couch.ArchiveThread`) | `cmd/internal/couchcore/detach.go:227` | new (extracted) | sessions (zellij via Artifacts), store |
| `Couch.ArchiveThread` | `cmd/internal/couchcore/detach.go` | modified (prepare + archive) | — |
| `Couch.claimFreshRecord` | `cmd/internal/couchcore/fresh_claim.go` | new (extracted from `startFreshSlot`) | artifact claim, entropy |
| `Couch.launchClaimedThread` | `cmd/internal/couchcore/couch.go` | new (extracted from `spawnResolved` tail) | launcher, runner |
| `Couch.startFreshSlot` | `cmd/internal/couchcore/slotrecovery.go:268` | modified (claim helper; no name/description carry) | slot store |
| `Couch.Reboot` / `RebootTarget` / `RebootResult` | `cmd/internal/couchcore/reboot.go` | new | all of the above |
| `Couch.ResumeTarget` / `ResumeTarget` | `cmd/internal/couchcore/resume_route.go` | new | OpenSlot, RetryContinuation, RecoverThread, ResumeContextWith |
| `OpenSlot` survivor refusal → typed `ResumeNoSurvivor` | `cmd/internal/couchcore/slotrecovery.go:505` | modified | — |
| `CouchLiveOwnerExecutor` / `DirectStoreExecutor` | `cmd/internal/couchcore/operationdispatch.go:141,259` | modified | — |
| `dispatchMenuRow` | `cmd/internal/couchtty/menu_slot.go:26` | modified | operation queue |
| `operationUsesCurrentRepoScope` / `operationOwnsLive` / `WantsConsole` | `cmd/internal/couchcmd/run.go:494-521` | modified | process entry |

- **ReplaceThreadExpected(old ThreadAddress, expectedRevision uint64, next ThreadRecord) error:** one
  `commitJournalLocked` that writes the archive bytes and grace for `old`, removes `old`'s record (and its
  materialized continuation file, the same way `archiveThread` does), writes `next`'s record, and replaces the
  manifest once (minus `old`, plus `next`). It refuses a local layout; slots use `replaceSlotCurrent`.
  - **Injected into:** `Couch.Reboot` (`:0` arm). Tests use the real store in `t.TempDir()` with
    `hooks.AfterJournal` as the crash seam.
- **prepareRetirement(ctx, address) (retirement, error):** everything `ArchiveThread` does before its final
  `ArchiveThreadExpected`: classify, check the session, clear debris, reconcile the helper, re-observe,
  quiesce, re-check absence. It returns `{Record, Revision, Unreadable, SessionNotStopped, RolledBack}`.
  `ArchiveThread` becomes prepare plus `ArchiveThreadExpected`. Its existing tests in `archive_test.go` stay
  green and unchanged. That is the refactor's safety net.

---

## Chunk 1: M1 — resume and reboot as couchcore operations

Every `go`/`make` command in this plan runs with the env scrub as a literal prefix. zsh does not word-split a
variable, so paste the prefix itself: `env -u PAIR_DATA_DIR -u PAIR_TAG -u PAIR_RETENTION_PROTOCOL -u
PAIR_RETENTION_BACKGROUND -u PAIR_RETENTION_START_ID`. Below it is abbreviated `SCRUB`.

### Task 1.1: `DecideReboot` and `RebootableState` (pure)

**Files:**
- Create: `cmd/internal/couchcore/reboot_decision.go`, `cmd/internal/couchcore/reboot_decision_test.go`
- Modify: `cmd/internal/couchcore/actionableinventory.go` (add `RebootableState` beside `ArchivableState:431`)

- [ ] **Step 1: Write the failing tests.**

```go
func TestDecideRebootCoversEveryFactCombination(t *testing.T) {
	for _, slot := range []bool{false, true} {
		for _, record := range []RebootRecord{RebootRecordNone, RebootRecordReadable, RebootRecordUnreadable} {
			for _, dir := range []bool{false, true} {
				plan, reason := DecideReboot(RebootFacts{Slot: slot, Record: record, DirectoryPresent: dir})
				want := expectedRebootPlan(slot, record, dir) // literal switch written from the Spec, below
				if plan != want || (plan != RebootArchiveAndStart && plan != RebootStartOnly && reason == "") { t.Errorf(...) }
			}
		}
	}
}
```

The expected values, stated from the Spec and the resolved ambiguities:

| slot | record | dir present | plan |
|---|---|---|---|
| any | readable | yes | `RebootArchiveAndStart` |
| any | readable | no | `RebootArchiveOnly`, reason "directory missing — add slot recreates it" |
| no | unreadable | — | `RebootArchiveOnly`, reason "record unreadable — start couch in its directory for a fresh agent" |
| yes | unreadable | yes | `RebootArchiveAndStart` (corrupt bytes go to `recovery/`) |
| yes | unreadable | no | `RebootRefuse` "directory missing — add slot recreates it" |
| yes | none | yes | `RebootStartOnly` (today's fresh-slot on an empty slot) |
| yes | none | no | `RebootRefuse` |
| no | none | — | `RebootRefuse` "nothing to reboot" |

Add `TestRebootableStateOverEveryClassification`. Iterate `AllThreadStates() × (AllThreadReasons() ∪ "")`,
skipping the combinations the projection cannot produce, and assert true exactly for parked, detached and
every unusable reason.

- [ ] **Step 2:** Run `SCRUB go test ./cmd/internal/couchcore -run 'TestDecideReboot|TestRebootableState'
      -count=1`. Expected: FAIL (undefined).
- [ ] **Step 3: Implement.**

```go
type RebootRecord uint8
const (
	RebootRecordNone RebootRecord = iota + 1
	RebootRecordReadable
	RebootRecordUnreadable
)
type RebootFacts struct {
	Slot             bool
	Record           RebootRecord
	DirectoryPresent bool
}
type RebootPlan uint8
const (
	RebootRefuse RebootPlan = iota + 1
	RebootArchiveOnly
	RebootArchiveAndStart
	RebootStartOnly
)
const RebootDirectoryMissing = "directory missing — add slot recreates it"
func DecideReboot(f RebootFacts) (RebootPlan, string) { /* the table above, as a switch */ }
```

- [ ] **Step 4:** Re-run. Expected: PASS.
- [ ] **Step 5: Commit** `#363 M1: reboot: pure decision and admission`.

### Task 1.2: `ResumeRebootAdvice` and `ChooseResumeRoute` (pure)

**Files:**
- Create: `cmd/internal/couchcore/resume_route.go`, `cmd/internal/couchcore/resume_route_test.go`
- Modify: `cmd/internal/couchcore/resume.go:14` (add `ResumeNoSurvivor ResumeDiagnosticCode =
  "resume-no-survivor"` and `ResumeSurvivorsAmbiguous = "resume-survivors-ambiguous"`)

- [ ] **Step 1: Write the failing tests.**
  - `TestResumeRebootAdviceClassifiesEveryDeclaredCode`. This is the derived enumeration (lessons: "a claim
    about every site needs a derived enumeration"). Parse `resume.go` with `go/parser`, collect every constant
    of type `ResumeDiagnosticCode`, and assert each is a key of `ResumeRebootAdvice`. Also assert the reverse:
    no key is undeclared.
  - `TestResumeRebootAdviceValues`, written as an independent literal from the Spec. **true:** path-missing,
    profile-missing, profile-invalid, agent-unsupported, binding-unbound, binding-root-missing,
    binding-ambiguous, tombstoned, session-gone, no-survivor. **false:** live, unknown, parking, starting,
    binding-provisional, not-detached, not-running, survivors-ambiguous.
  - `TestChooseResumeRoute`, a table:

| Slot | HasRecord | Continuation phase | route |
|---|---|---|---|
| yes | no | — | `ResumeRouteSlot` (OpenSlot adopts a lost pointer) |
| any | yes | failed / running | `ResumeRouteContinuation` |
| any | yes | pending | `ResumeRouteRecover` |
| yes | yes | none / complete | `ResumeRouteSlot` |
| no | yes | none / complete | `ResumeRouteThread` |

  Iterate `checkpoint.AllPhases()` plus `""`, so a new phase fails the test.

- [ ] **Step 2:** Run `SCRUB go test ./cmd/internal/couchcore -run
      'TestResumeRebootAdvice|TestChooseResumeRoute' -count=1`. Expected: FAIL.
- [ ] **Step 3: Implement.**

```go
type ResumeRoute uint8
const (
	ResumeRouteSlot ResumeRoute = iota + 1 // OpenSlot: warm, cold, or adopt a still-running agent
	ResumeRouteContinuation                // RetryContinuation(request.ID)
	ResumeRouteRecover                     // RecoverThread(address): survivor reattach or retained checkpoint
	ResumeRouteThread                      // ResumeContextWith: warm, then cold
)
type ResumeRouteInput struct {
	Slot, HasRecord bool
	Continuation    checkpoint.Phase // "" when the record has none
}
func ChooseResumeRoute(in ResumeRouteInput) ResumeRoute

var ResumeRebootAdvice = map[ResumeDiagnosticCode]bool{ /* every code */ }

// withRebootAdvice keeps errors.As working (%w) and appends the exit.
func withRebootAdvice(err error) error {
	if err == nil || !ResumeRebootAdvice[ResumeDiagnosticOf(err)] { return err }
	return fmt.Errorf("%w; this conversation cannot come back -- Tab → reboot archives it and starts a fresh agent here", err)
}
```

- [ ] **Step 4:** Re-run. Expected: PASS.
- [ ] **Step 5: Commit** `#363 M1: resume: pure route and reboot advice`.

### Task 1.3: Shared journal builders and `ReplaceThreadExpected` (`:0` crash safety)

**Files:**
- Create: `cmd/internal/couchcore/threadstore_replace.go`,
  `cmd/internal/couchcore/threadstore_replace_test.go`
- Modify: `cmd/internal/couchcore/threadstore.go:214-274` (`CreateThread`), `:1290-1363` (`archiveThread`)

- [ ] **Step 1: Write the failing tests** (real store in `t.TempDir()`, as `threadstore_test.go` does):
  - `TestReplaceThreadExpectedArchivesAndCreatesInOneJournal`. Afterwards `GetThread(next)` succeeds;
    `GetThread(old)` is `ErrThreadNotFound`; `archivePath(old)` holds old's exact bytes;
    `archiveGracePath(old)` exists; the manifest lists `next` and not `old`; `ArchivedThreads()` contains old
    with its `Name`/`Description` intact.
  - `TestReplaceThreadExpectedSurvivesACrashAfterTheJournal`. Set `s.hooks.AfterJournal = func() error {
    return errors.New("interrupted") }`, expect an error, reset the hooks, run `s.RecoverStoreJournal()`, then
    make the same four assertions. This mirrors `TestSlotFreshBackupCapAndJournalRecovery/journal`.
  - `TestReplaceThreadExpectedRefusesAStaleRevision`: bump old's revision via `ApplyThreadMetadata`, then
    replace with the old revision. Expect `*ThreadRevisionError`, no archive file, and `next` absent.
  - `TestReplaceThreadExpectedRefusesAnOpenParkOrStartClaim`: an `archivableRecord` refusal, with no journal
    written.
  - `TestReplaceThreadExpectedRefusesLocalLayout`.
- [ ] **Step 2:** Run `SCRUB go test ./cmd/internal/couchcore -run TestReplaceThreadExpected -count=1`.
      Expected: FAIL (undefined).
- [ ] **Step 3: Extract and implement.** Lift the entry construction out of `archiveThread` into
      `archiveJournalEntries(address, raw) ([]storeJournalEntry, error)` (archive, grace, record removal,
      continuation snapshot removal) and out of `CreateThread` into `createJournalEntries(record)
      ([]storeJournalEntry, error)`. Manifest handling stays in each caller, because replace edits the
      manifest once. Then:

```go
func (s *ThreadStore) ReplaceThreadExpected(old ThreadAddress, expectedRevision uint64, next ThreadRecord) error {
	// refuse s.layout.Local; validate next (ValidateThreadRecord, validateLocalOrigin);
	// route both addresses through storeForAddress and refuse if they land on different backends;
	// withLock: loadManifestLocked; read old raw (must exist) and decode; check the revision and archivableRecord;
	// next must not exist and must not be in the manifest (ThreadExistsError);
	// entries := archiveJournalEntries(old, raw) + createJournalEntries(next)
	//   + one manifest entry {Expected: manifestRaw, After: manifest - old + next, Generation+1};
	// commitJournalLocked(storeJournal{SchemaVersion: 1, Entries: entries})
}
```

- [ ] **Step 4:** Re-run the new tests, then the store suites: `SCRUB go test ./cmd/internal/couchcore -run
      'TestReplaceThreadExpected|Archive|CreateThread|Journal' -count=1`. Expected: PASS. The existing archive
      and create tests are the safety net for the extraction.
- [ ] **Step 5: Mutation check** (lessons: "a test must fail when the code under test is reverted").
      Temporarily drop the manifest entry from replace. The crash test must fail. Restore.
- [ ] **Step 6: Commit** `#363 M1: threadstore: one-journal replace for the main store`.

### Task 1.4: Extract `prepareRetirement`, `claimFreshRecord`, `launchClaimedThread`; stop carrying name and description

**Files:**
- Modify: `cmd/internal/couchcore/detach.go:227-394`, `cmd/internal/couchcore/slotrecovery.go:268-379`,
  `cmd/internal/couchcore/couch.go:430-570`
- Create: `cmd/internal/couchcore/fresh_claim.go`
- Test: `cmd/internal/couchcore/slotrecovery_test.go:136`

- [ ] **Step 1: Flip the existing assertion first.** In
      `TestStartFreshSlotReplacesStoppedCurrentAndKeepsPreferences`, require `next.Name == "" &&
      next.Description == ""`. Also read the archived record (`local.ArchivedThreads()`) and require that it
      kept `"durable name"` and `"durable description"`.
- [ ] **Step 2:** Run `SCRUB go test ./cmd/internal/couchcore -run TestStartFreshSlotReplacesStoppedCurrent
      -count=1`. Expected: FAIL (the fresh record still copies them).
- [ ] **Step 3: Refactor.**
  - `prepareRetirement`: move `ArchiveThread`'s body, everything before each final `ArchiveThreadExpected`,
    into it. Return `retirement{Record ThreadRecord; Revision uint64; Unreadable, SessionNotStopped,
    RolledBack bool}`. `ArchiveThread` keeps its doc comment and becomes prepare, then
    `ArchiveThreadExpected(address, r.Revision)` unless `RolledBack`.
  - `claimFreshRecord(ctx, freshClaimInput{ScopeKey, Cwd, RepoIdentity, TagPrefix string; Profile
    LaunchProfileResolution; Used map[ThreadAddress]bool}) (ThreadRecord, string /*nonce*/,
    ThreadArtifactClaim, error)`: the tag loop, `AdvanceStartTransaction` and `Artifacts.Claim` from
    `slotrecovery.go:316-355`. It never copies `Name`/`Description`; delete lines 339-342.
  - `launchClaimedThread(ctx, record, nonce, args, profile)`: `prepareTrackedWorkspace` →
    `BuildCouchLaunchProfile` → `launchTrackedThread`, with `rollbackTrackedStart` on failure. This is the
    tail `spawnResolved` (`couch.go:544-570`) and `startFreshSlot` share. `startFreshSlot` keeps its own
    `verifyOtherSlotOwnersAbsent` / `revalidateCreatedSlot` hook through an optional `afterPrepare func()
    error`.
- [ ] **Step 4:** Run `SCRUB go test ./cmd/internal/couchcore -count=1`. Expected: PASS, apart from the 3
      tests known to fail on main (see Chunk 4). The archive, slot-fresh and spawn suites prove the refactor
      kept behavior.
- [ ] **Step 5: Commit** `#363 M1: couchcore: share retirement, claim and launch; fresh records start
      unnamed`.

### Task 1.5: `Couch.ResumeTarget` and the `resume` operation

**Files:**
- Modify: `cmd/internal/couchcore/resume_route.go`, `cmd/internal/couchcore/slotrecovery.go:505,535`,
  `cmd/internal/couchcore/operationdispatch.go:411-433`, `cmd/internal/couchcore/ops.go` (`resume` gains
  `{Name: "path", Summary: "slot host checkout", Implicit: true}`)
- Test: `cmd/internal/couchcore/resume_route_test.go`, `cmd/internal/couchcore/slotrecovery_test.go`

- [ ] **Step 1: Write the failing tests.** They go through `CouchLiveOwnerExecutor(env.Couch)` with the
      declared `resume` operation, because that is the production boundary.
  - `TestResumeOperationOnASlotPathAdoptsALostPointer`: reuse
    `TestSlotOpenReconstructsSingleDetachedSurvivor`'s fixture, but dispatch `resume {path}`. Expect a live
    actor; `thread.json` now names the survivor.
  - `TestResumeOperationRetriesAFailedContinuation`: a parked record with a `checkpoint.Failed` request.
    Expect the `RetryContinuation` path (assert on its result type and the request phase advancing), and no
    `continuationGuard` refusal.
  - `TestResumeThatCannotSucceedNamesReboot`: a parked `:0` with no native binding (the cold path refuses
    `ResumeBindingUnbound`). Expect an error whose text contains `reboot` and where `ResumeDiagnosticOf(err)
    == ResumeBindingUnbound`.
  - `TestResumeTransientRefusalDoesNotNameReboot`: a `ResumeStarting` refusal. The text must not contain
    `reboot`.
  - `TestOpenSlotZeroSurvivorsIsATypedRefusal`: `ResumeNoSurvivor`. More than one survivor gives
    `ResumeSurvivorsAmbiguous`.
  - `TestWarmOnlyResumeIsUnrouted`: `resume {warm-only: true}` still calls `ResumeContextWith(WarmOnly)`
    directly and refuses `ResumeNotDetached` for a parked thread. The background reattach pass depends on
    this.
- [ ] **Step 2:** Run `SCRUB go test ./cmd/internal/couchcore -run
      'TestResumeOperation|TestResumeThat|TestResumeTransient|TestOpenSlotZero|TestWarmOnlyResume' -count=1`.
      Expected: FAIL.
- [ ] **Step 3: Implement.**

```go
type ResumeTarget struct {
	Path    string        // slot host checkout (:1+)
	Address ThreadAddress // ordinary (:0)
}
func (c *Couch) ResumeTarget(ctx context.Context, t ResumeTarget) (any, error) {
	// read: slot → selectedSlot + observeSlotCurrent; ordinary → Threads.GetThread
	// route := ChooseResumeRoute(...)
	// Slot → OpenSlot(ctx, t.Path, ""); Continuation → RetryContinuation(ctx, addr, request.ID);
	// Recover → RecoverThread(ctx, addr, ""); Thread → ResumeContextWith(ctx, addr, ResumeOptions{})
	// return result, withRebootAdvice(err)
}
```

  In `OpenSlot`, replace the two "choose Start fresh" messages (`slotrecovery.go:505,535`) with
  `refuseResume(ResumeNoSurvivor | ResumeSurvivorsAmbiguous, …)` and wrap the final error with `%w` only.
  `withRebootAdvice` adds the exit text once, at the top. In the `resume` dispatch arm: `warm-only` takes the
  existing direct path; `path` (or a `repo:N` ref, `N > 0`) becomes `ResumeTarget{Path}`; everything else
  becomes `ResumeTarget{Address}`.
- [ ] **Step 4:** Re-run, then `SCRUB go test ./cmd/internal/couchcore -run 'Resume|OpenSlot|Slot' -count=1`.
      Expected: PASS.
- [ ] **Step 5: Commit** `#363 M1: resume: one operation over warm, cold, adopt and continuation`.

### Task 1.6: `Couch.Reboot` and the `reboot` operation

**Files:**
- Create: `cmd/internal/couchcore/reboot.go`, `cmd/internal/couchcore/reboot_test.go`
- Modify: `cmd/internal/couchcore/ops.go` (declare `reboot`), `cmd/internal/couchcore/operationdispatch.go`
  (live-owner arm)

Declaration, with `RowAction: false` until M2 wires the switcher:

```go
{Name: "reboot", Summary: "Archive this conversation and start a fresh agent in the same slot or path",
 Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmRequired, Result: ResultStart,
 Presentation: PresentationTUI, RowAction: false,
 Args: []ArgSpec{
	{Name: "path", Summary: "slot host checkout", Implicit: true},
	{Name: "repo-scope", Summary: "repository scope derived from caller context", Implicit: true},
	{Name: "tag", Summary: "exact thread tag from trusted owner context", Implicit: true},
	{Name: "agent", Summary: "agent for the fresh conversation", FlagOnly: true, ValueRequired: true}}},
```

```go
type RebootTarget struct { Path string; Address ThreadAddress; Agent string }
type RebootResult struct {
	Start             StartResult
	Archived          ThreadAddress
	ArchiveOnly       bool
	Reason            string // DecideReboot's reason when ArchiveOnly
	SessionNotStopped bool
}
func (r RebootResult) Started() (StartResult, bool) { return r.Start, r.Start.Handle != nil }
```

**Sequence and ordering (ARCH-ORDER).** `Couch.Reboot` holds no state between calls. Its durable effects
happen in this order, and each crash point leaves a state the existing machinery already handles:

| Step | Effect | If the process dies here | If a second actor intervenes |
|---|---|---|---|
| 1 classify (`RebootableState`, refuse unknown) | none | nothing changed | — |
| 2 `prepareRetirement` (may quiesce a detached session) | session stopped, record intact | row reads parked/session-gone; reboot retries cleanly | prepare re-observes and refuses on change (existing #256 M2 rule) |
| 3 `DecideReboot` | none | — | — |
| 4a archive-only: `ArchiveThreadExpected(rev)` | one journal | journal replay completes it | revision CAS refuses |
| 4b `:0`: `claimFreshRecord` → `ReplaceThreadExpected(old, rev, claimed)` | artifact claim, then one journal | before the journal: claim released by `releaseClaimIfThreadAbsent` on next start; after: new record with this couch's start claim, recovered by the existing dead-owner claim rule | CAS refuses; claim released |
| 4c `:1+`: `startFreshSlot` (re-observes, `replaceSlotCurrent`) | one journal | existing slot journal replay | byte-compare refuses |
| 5 `launchClaimedThread` | child process | start claim recovery (existing) | — |
| 5′ launch fails | `rollbackTrackedStart` removes the new claim | old stays archived; the path is free, so the next start or reboot is fresh | — |

The event most likely to be mishandled is a **retry after a reboot whose launch failed**. The old record is
already archived, so the row is gone or reads as a free path. A second reboot therefore has nothing to archive
(`:0` → `RebootRefuse` "nothing to reboot"; `:1+` → `RebootStartOnly`). The test below pins that.

- [ ] **Step 1: Write the failing tests.** Dispatch through `CouchLiveOwnerExecutor` with `reboot` args.
  - `TestRebootPrimaryParkedArchivesAndStartsFreshInOneJournal`. Set up a parked `:0` with
    `Name`/`Description` set. Afterwards: result live with a new tag ≠ old; `StartingPath` equal; new record
    `Name == "" && Description == ""`; `ArchivedThreads()` holds the old record with its name; the manifest
    lists only the new address in that scope; a child was spawned (`env.Runner.Ops`).
  - `TestRebootPrimaryCrashAfterJournalRecovers`. Set `env.Couch.Threads.hooks.AfterJournal` to fail, then
    call `RecoverStoreJournal`. The old record is archived and the new claimed record exists; no child was
    spawned. The next inventory classifies the new record from its dead start claim, not as live.
  - `TestRebootDetachedStopsTheSessionFirst`. Detached `:0` (`SetSessionPresence` present, detached).
    Afterwards the quiesce ran (session absent in the fake) **before** the journal. Order is asserted by a
    hook that records `presence` at `AfterJournal` time.
  - `TestRebootRefusesLiveBusyAndStillUnknown`. Each case is refused, with no journal written (archive dir
    empty) and no child spawned.
  - `TestRebootDirectoryMissingArchivesOnly`. A `:0` with `Reason == ReasonPathMissing`. The result has
    `ArchiveOnly` with the "directory missing — add slot recreates it" reason; the record is archived; no
    child.
  - `TestRebootUnreadablePrimaryArchivesOnly`. Bytes are moved as-is and `SessionNotStopped` is true.
  - `TestRebootSlotParkedIsTodaysFreshSlot` (via `slotRecoveryOperationFixture`). New tag, old in `archive/`,
    name not carried.
  - `TestRebootSlotDetachedQuiescesThenStarts`. If `prepareRetirement` cannot classify a slot address (it
    routes through `storeForAddress`), **stop and re-plan** rather than adding a slot-only quiesce. Log the
    finding.
  - `TestRebootAfterFailedLaunchDoesNotArchiveTwice`. Make the launch fail via `FakeRunner`; then reboot
    again. `:0` gives `RebootRefuse` "nothing to reboot" and `:1+` gives a clean fresh start. Exactly one
    archive file.
- [ ] **Step 2:** Run `SCRUB go test ./cmd/internal/couchcore -run TestReboot -count=1`. Expected: FAIL
      (undefined).
- [ ] **Step 3: Implement `Couch.Reboot`** with the sequence above. For the `:0` fresh start, resolve the
      profile with `resolveStartResolution(ctx, StartArgs{Cwd: old.StartingPath, Stack: t.Agent, Action:
      StartOpen})`. Call `enrollPrimaryResolution` the way spawn does (idempotent for an enrolled family;
      verify in the test). Then use `claimFreshRecord` and `ReplaceThreadExpected(old.Address,
      retirement.Revision, claimed)`, and finally `launchClaimedThread`. "Directory present" means the slot's
      `WorktreeRoot`, or the record's `StartingPath`, physicalizes through `c.Path.Physical`. Add the `reboot`
      dispatch arm: `path` → slot; otherwise `resolveThreadForArchive` (it accepts unreadable records by exact
      tag).
- [ ] **Step 4:** Re-run. Expected: PASS. Then run `SCRUB go test ./cmd/internal/couchcore
      ./cmd/internal/couchcmd -count=1`.
- [ ] **Step 5: Mutation checks.** (a) Swap the order of quiesce and journal in the detached arm:
      `TestRebootDetachedStopsTheSessionFirst` must fail. (b) Re-add the name copy in `claimFreshRecord`: the
      `:0` and `:1+` reboot tests must fail. Restore both.
- [ ] **Step 6: Commit** `#363 M1: reboot: archive-then-fresh for :0 and :1+, archive-only when the directory
      is gone`.

### Task 1.7: Close M1

- [ ] Update `atlas/couch.md` operations section (`:521-524`) for the two new operations and the shared
      journal. Update `workshop/issues/000363-…md` `## Log`.
- [ ] Run Chunk 4's full verification recipe.
- [ ] `sdlc milestone-close --issue 363 --milestone M1`.

---

## Chunk 2: M2 — the switcher speaks the slot model

### Task 2.1: The pure action table

**Files:**
- Create: `cmd/internal/couchtty/menu_actions.go`, `cmd/internal/couchtty/menu_actions_test.go`
- Modify: `cmd/internal/couchtty/menu.go:1314-1458` (delete `menuLiveActions`, `menuActionItems`,
  `menuArchiveOffered`), `cmd/internal/couchtty/menu_slot.go:35` (delete `slotFreshOffered`),
  `cmd/internal/couchtty/menu_switchagent.go:201` (add-slot leaves `menuActionsFor`)

- [ ] **Step 1: Write the failing test.** `TestRowActionTableMatchesTheSpec`. The **domain is derived**: kinds
      {primary `:0`, slot `:1`} × `AllThreadStates()` (minus archived) × `AllThreadReasons() ∪ ""` (only the
      combinations the projection can produce) × (`checkpoint.AllPhases() ∪ none`). The **expected value** is
      a literal function written from the Spec table, which is a separate statement and not a call into
      production:

| Row | expected `menuActionItems` |
|---|---|
| live `:0` | `detach relaunch park switch-agent alias add-slot` (alias only when `menuAliasOffered`) |
| live `:1+` | `detach relaunch park switch-agent` |
| live, continuation failed | the above minus `ContinuationRefuses`, with `retry-continuation dismiss-continuation` after `detach` |
| parked or detached, any kind | `resume reboot` |
| not live, continuation pending/running/failed | `resume reboot` |
| unusable `:0` with `Recovery.Recover` | `resume reboot` |
| unusable `:1+` (not path-missing) | `resume reboot` (OpenSlot may adopt) |
| unusable `:0` without a recovery offer, incl. `unknown` | `reboot` |
| unusable, `path-missing` | `reboot` |
| busy | nothing |

  Add `TestBusyRowSaysStartingElsewhere`: render the root list and assert the status text. Change `case "":`
  at `menu.go:1306` to return "starting elsewhere", and Enter's notice to match.
- [ ] **Step 2:** Run `SCRUB go test ./cmd/internal/couchtty -run 'TestRowActionTable|TestBusyRow' -count=1`.
      Expected: FAIL.
- [ ] **Step 3: Implement.**

```go
type menuRowKind uint8
const (
	menuRowPrimary menuRowKind = iota + 1 // ordinary target: the repository's :0
	menuRowSlot                            // slot target: :1+
)
type menuRowPhase uint8
const (
	menuPhaseLive menuRowPhase = iota + 1
	menuPhaseLiveContinuationFailed // #280: a failed request composes with live actions
	menuPhaseResumable              // parked or detached
	menuPhaseUnusable               // unusable, or not live with an unfinished request
	menuPhaseBusy
)
type menuRowFacts struct {
	Kind             menuRowKind
	Phase            menuRowPhase
	ResumeOffered    bool // unusable rows only: slot rows, or a primary with Recovery.Recover or an unfinished request
	DirectoryMissing bool // Reason == ReasonPathMissing
	AliasOffered     bool // menuAliasOffered
	AddSlotOffered   bool // menuAddSlotPath != "" (live :0 only reaches the table's add-slot)
}
func menuRowFactsOf(row couchcore.ActionableThreadSummary) menuRowFacts
func menuRowActions(f menuRowFacts) []string {
	switch f.Phase {
	case menuPhaseLive, menuPhaseLiveContinuationFailed:
		items := []string{}
		for _, op := range []string{"detach", "relaunch", "park", "switch-agent"} {
			if f.Phase == menuPhaseLiveContinuationFailed && couchcore.ContinuationRefuses(op) { continue }
			items = append(items, op)
			if op == "detach" && f.Phase == menuPhaseLiveContinuationFailed {
				items = append(items, "retry-continuation", "dismiss-continuation")
			}
		}
		if f.Kind == menuRowPrimary && f.AliasOffered { items = append(items, "alias") }
		if f.Kind == menuRowPrimary && f.AddSlotOffered { items = append(items, "add-slot") }
		return items
	case menuPhaseResumable:
		return []string{"resume", "reboot"}
	case menuPhaseUnusable:
		if f.ResumeOffered && !f.DirectoryMissing { return []string{"resume", "reboot"} }
		return []string{"reboot"}
	}
	return nil // busy
}
func menuActionItems(row couchcore.ActionableThreadSummary) []string { return menuRowActions(menuRowFactsOf(row)) }
```

  Keep the "NOT filtered through the declaration" doc comment on `menuActionItems`. `menuActionsFor` keeps
  only `copy-orientation`.
- [ ] **Step 4:** Re-run. Expected: PASS.
- [ ] **Step 5: Commit** `#363 M2: switcher: one pure action table per row`.

### Task 2.2: Enter, dispatch, confirmation and in-flight re-checks

**Files:**
- Modify: `cmd/internal/couchtty/menu.go:751-768` (`enterOperationFor`), `:808` (text frames), `:870-890`
  (`reduceConfirmationKey`), `:1480-1520` (`confirmationMenuItems`), `:1745-1800` (in-flight re-check),
  `:1835-1880`, `:1935`, `:2000-2025` (completion text); `cmd/internal/couchtty/menu_slot.go:26`
  (`dispatchMenuRow`); `cmd/internal/couchtty/menu_render.go:352,358,422`
- Test: `cmd/internal/couchtty/menu_test.go`, `menu_slot_test.go`, `menu_recovery_test.go`,
  `menu_continuation_test.go`

- [ ] **Step 1: Write and rewrite the failing tests.**
  - `TestEnterResumesEveryRowThatOffersResume`: over the Task 2.1 domain, Enter on a row whose items contain
    `resume` dispatches `resume`, and on a live row dispatches `switch`. Other rows dispatch nothing and show
    the row's notice plus "Tab → reboot".
  - `TestSlotRowResumeAndRebootSendThePath`: a slot row's `resume`/`reboot` effects carry `{path:
    WorktreeRoot}`, and `InFlight.RowKey` is set. A `:0` row carries `{repo-scope, tag}`.
  - `TestRebootConfirmationNamesItsCost`: the item reads `reboot <label> — archives this conversation, starts
    a fresh agent`. When detached it adds `, stops its running agent`. When path-missing it reads `reboot
    <label> — directory missing: archives the record only; add slot recreates it`.
  - `TestConfirmationRechecksTheOfferForEveryAction`: open a reboot confirmation, flip the row to live, press
    Enter. The frames are discarded and nothing dispatches. This replaces the `archive`/`fresh-slot`
    special-case guards at `menu.go:880-882` and `:1747-1749` with one rule:
    `!containsMenuItem(menuActionItems(thread), frame.Action)`.
  - Rewrite `TestUnusableRowOffersOnlyMetadataActions` (`menu_test.go:1276`) as `TestUnusableRowOffersReboot`.
    Delete the archive-specific tests at `:1291` and their recovery and continuation siblings, and replace
    them with reboot or resume equivalents. Keep the busy test (`:1343`), now asserting no items and "starting
    elsewhere".
- [ ] **Step 2:** Run `SCRUB go test ./cmd/internal/couchtty -count=1`. Expected: FAIL on the new and
      rewritten tests.
- [ ] **Step 3: Implement.** `enterOperationFor`: live → `switch`; `resume` offered → `resume`; else `""`.
      `dispatchMenuRow`: for `resume`/`reboot` on a slot target, send `{path}`; otherwise use
      `dispatchThreadOperation`. In-flight tracking keys by `RowKey` whenever the effect carries a `path`
      (`menu.go:1795,1935`), not by the `open-slot`/`fresh-slot` names. Completion text: `"rebooting " +
      label`. Remove the `archive`/`recover-checkpoint` arms in `menu_render.go:352,358`. Change
      `menu_render.go:422`'s reuse-notice verbs to `resume`/`reboot`.
- [ ] **Step 4:** Re-run. Expected: PASS.
- [ ] **Step 5: Commit** `#363 M2: switcher: Enter, dispatch and confirmations follow the table`.

### Task 2.3: Flip `RowAction`, delete superseded operations

**Files:**
- Modify: `cmd/internal/couchcore/ops.go` (`reboot` gets `RowAction: true`; delete `open-slot`, `fresh-slot`,
  `name`, `describe`, `archive`, `recover-thread`, `recover-checkpoint`),
  `cmd/internal/couchcore/operationdispatch.go:192-224,290-300,323-326`,
  `cmd/internal/couchcmd/run.go:494-521`
- Test: `cmd/internal/couchtty/menu_action_sweep_test.go`, `cmd/internal/couchtty/action_agreement_test.go`,
  `cmd/internal/couchcmd/*_test.go`

- [ ] **Step 1: Update the sweeps first, so they fail.**
  - `TestRowActionDeclarationsAndTheMenuAgreeInBothDirections`: build `offered` from the **same derived
    domain** as Task 2.1 (a shared test helper `everyMenuRowShape(t)` in `menu_actions_test.go`), not the
    current hand-picked rows.
  - `TestEveryOfferedActionIsReachableFromEnter`: iterate `everyMenuRowShape`.
  - `TestActionOfferedImpliesPermitted`: actions become `{"switch-agent", SwitchableState}`, `{"resume",
    ResumableState-or-unusable-with-offer}` and `{"reboot", RebootableState}`. Resume on unusable rows is
    admitted by `ResumeTarget`'s route, so pair it with a `resumeAdmitted(state, reason)` predicate stated in
    the test: resumable, or unusable. Delete `TestRecoveryRowArchiveOfferedImpliesPermitted`.
- [ ] **Step 2:** Run `SCRUB go test ./cmd/internal/couchtty -run
      'TestRowActionDeclarations|TestEveryOfferedAction|TestActionOffered' -count=1`. Expected: FAIL. `reboot`
      is offered but not declared a RowAction, and the deleted operations are declared but not offered.
- [ ] **Step 3: Implement.** Flip `RowAction` and delete the declarations and their executor arms.
      `Couch.ArchiveThread`, `OpenSlot`, `StartFreshSlot`, `RecoverThread` and `ApplyThreadMetadata` stay as
      internals; `publish-description` still uses the last. If `RecoverThread`'s `path` parameter has no
      caller left, delete it and its `selected` branch, and move its tests in `recovery_execute_test.go` to
      the retained-checkpoint form. `run.go`: `operationUsesCurrentRepoScope` drops `name describe
      recover-thread recover-checkpoint archive` and adds `reboot`. `operationOwnsLive` drops `open-slot
      fresh-slot recover-thread recover-checkpoint archive` and adds `reboot`. `WantsConsole` drops the
      `archive` exception.
- [ ] **Step 4:** Run `SCRUB go test ./cmd/internal/couchtty ./cmd/internal/couchcore ./cmd/internal/couchcmd
      -count=1`. Expected: PASS (minus the known 3).
- [ ] **Step 5: Commit** `#363 M2: ops: reboot is a row action; rename, describe, archive and repair entries
      leave`.

### Task 2.4: Labels stop reading the stored name; summaries stop reading the stored description

**Files:**
- Modify: `cmd/internal/couchcore/actionableinventory.go:220-256`,
  `cmd/internal/couchcore/threadinventory.go:29-41`, `cmd/internal/couchtty/menu_render.go:549-551`,
  `cmd/internal/couchtty/console_presentation.go:30,48,57`
- Test: `cmd/internal/couchcore/actionableinventory_test.go`, `cmd/internal/couchtty/menu_test.go`,
  `cmd/internal/couchcmd` list/show tests

- [ ] **Step 1: Write the failing tests.**
  - `TestSlotRowLabelIgnoresStoredName`: a slot row with `Name: "renamed"` gives `repo:1`, and with
    `RepositoryAlias: "pr"` gives `pr:1`. Do the same for `ThreadSummary.Label`.
  - `TestPrimaryRowLabelIgnoresStoredName`: an ordinary row with `Name: "renamed"` labels with the
    working-path basename (or the alias when set).
  - `TestDisplaySummaryIsThePublishedSummaryOnly`: `Description` set and `PublishedSummary` empty give `""`
    from both summary types.
  - `TestSwitcherRowShowsNoStoredNameDetail`: the rendered root line does not contain `(renamed)`.
- [ ] **Step 2:** Run the four tests. Expected: FAIL.
- [ ] **Step 3: Implement.** `threadLabel(workingPath, tag)` loses its `name` parameter. Both `Label()`
      methods stop reading `Name`. Both `DisplaySummary()` methods return `PublishedSummary`. Delete the
      `(name)` detail at `menu_render.go:549`. **Trace `console.go:448`'s `label`** to its source. If it
      derives from the stored name, stop overlaying `rows[index].Name` in `console_presentation.go:48`, so the
      tab bar and the switcher read the same `Label()`. Record the finding in `## Log`.
- [ ] **Step 4:** Re-run, then `SCRUB go test ./cmd/internal/couchcore ./cmd/internal/couchtty
      ./cmd/internal/couchcmd -count=1`. Expected: PASS.
- [ ] **Step 5: Commit** `#363 M2: labels: slot rows always label repo:N or alias:N`.

### Task 2.5: Matching stops reading stored name and description

**Files:**
- Modify: `cmd/internal/couchcore/threadmetadata.go:~150-200` (`ThreadReferenceFields{Address, Label,
  WorkingPath, Summary}`), `cmd/internal/couchtty/menu.go:366-371`, `cmd/internal/couchcore/ops.go:216` and
  every `"thread tag, path, or name"` / `"operator-assigned name"` arg summary
- Test: `cmd/internal/couchcore/threadmetadata_test.go`, `cmd/internal/couchtty/menu_test.go`

- [ ] **Step 1: Write the failing tests.** `TestThreadReferenceDoesNotMatchStoredNameOrDescription`:
      `ResolveThreadReference` on `"renamed"`, where only `Name` holds it, gives `ErrThreadReferenceNotFound`.
      The same applies to `Description`. A tag or path still matches.
      `TestSwitcherFilterMatchesLabelAndPublishedSummary`: a filter `repo:1` or a published-summary word
      selects the row; a stored-name word does not.
- [ ] **Step 2:** Run. Expected: FAIL.
- [ ] **Step 3: Implement.** `ResolveThreadReference` leaves `Label` and `Summary` empty, so only tag and path
      match there. The switcher passes `Label: row.Label()` and `Summary: menuFocusSummary(row)`. Arg
      summaries become `"thread tag or path"`.
- [ ] **Step 4:** Re-run. Expected: PASS.
- [ ] **Step 5: Commit** `#363 M2: matching: stored names and descriptions are no longer searched`.

### Task 2.6: Derived sweep of removed operation names and stale advice strings

- [ ] **Step 1: Enumerate.** Run:

```bash
grep -rnE '"(open-slot|fresh-slot|name|describe|archive|recover-thread|recover-checkpoint)"' cmd/internal | grep -v _test.go
grep -rnE 'Tab → archive|Start fresh|Recover from checkpoint|, or archive|Dismiss continuation|Retry continuation|rename' cmd/internal | grep -v _test.go
```

  Every hit is a site to delete or reword. The known ones are: `couch.go:457,471,486` ("retire it: … Tab →
  reboot"); `recovery.go:62` and `recovery_execute.go:276` ("… or reboot"); `menu.go:38` (help line becomes
  `Tab → reboot` with "archive this conversation and start a fresh agent"); `checkpoint/request.go:37-57`
  (`Exits`: "Resume retries it; Reboot archives it", keeping the `couch --internal
  retry-continuation|dismiss-continuation` CLI forms); `console.go:1891` and `console_continuation.go:230`
  (focus and continuation handling for `recover-thread`/`recover-checkpoint`, which become `reboot` or are
  deleted).
- [ ] **Step 2:** Fix each hit. For every changed message, update or add the test that executes it.
      `spawnResolved`'s refusal comment requires that every gesture it names is executed by a test.
- [ ] **Step 3:** Re-run both greps. Expected: no hits except the `"archive"` directory-name strings
      (`threadstore.go:1251,1373`, `retention.go:176`, `slotsessions.go:251`,
      `repository_family_store.go:219`, `slotmigration.go:187`) and `RecoveryDecision.Archive`'s JSON tag.
      Paste the final grep output into `## Log`.
- [ ] **Step 4:** Run `SCRUB go test ./cmd/internal/... -count=1`. Expected: PASS (minus the known 3).
- [ ] **Step 5: Commit** `#363 M2: side-quest: every refusal names resume or reboot, never a removed action`.

### Task 2.7: Directory-missing row explanation

**Files:** `cmd/internal/couchtty/menu.go:1282-1310` (`unusableThreadNotice`, row reason text)
- [ ] **Step 1:** Write the failing test `TestPathMissingRowExplainsAddSlot`: rendered status and Enter notice
      contain "directory missing — add slot recreates it".
- [ ] **Step 2:** Run. Expected: FAIL.
- [ ] **Step 3:** Implement using `couchcore.RebootDirectoryMissing`, so the row and the reboot result say the
      same thing.
- [ ] **Step 4:** Run. Expected: PASS.
- [ ] **Step 5:** Commit.

### Task 2.8: README and atlas

**Files:** `README.md:383,500-502,543-544,714,768-771,783-798`,
`atlas/couch.md:427,521-524,537,552-553,578-582,668-690,946`

- [ ] **Step 1: Enumerate.** `grep -nE 'rename|describe|Tab → archive|archive|recover|fresh slot|open
      slot|Dismiss|Retry continuation' README.md atlas/couch.md`. Classify each hit: a switcher action (reword
      to resume/reboot or delete), a store mechanism (archive directory, grace: keep), or pair's own `pair
      rename` (keep).
- [ ] **Step 2: Rewrite.** Cover the per-row table, what resume tries in order, reboot's contract (archive
      with evidence, fresh tag, no branch or worktree change, directory-missing archive-only, name and
      description not carried), stored name and description no longer displayed or matched, the busy row's
      "starting elsewhere", and the live-failed continuation exits. The atlas gets the new pure entities and
      `ReplaceThreadExpected`. Keep `atlas/index.md` links valid.
- [ ] **Step 3:** Run README contract checks (`SCRUB make -k test` covers them; lessons: "documentation edits
      can break executable contract checks").
- [ ] **Step 4:** Commit `#363 M2: docs: README and atlas speak resume and reboot`.

### Task 2.9: Close M2

- [ ] Full verification (Chunk 4).
- [ ] **Ask the operator to smoke-test live** (memory: dogfood live). Run `make build` in `pair:0` after `sdlc
      move`. Then: Tab on a live `:1` (4 items); a parked row (resume, reboot); reboot a parked `:0` (new tag
      in the tab bar, old in `couch --archived`); a renamed row now labels `repo:N`.
- [ ] `sdlc milestone-close --issue 363 --milestone M2`.

---

## Chunk 3: M3 — one primary per repository

### Task 3.1: Scope-wide selection and occupancy (pure)

**Files:**
- Modify: `cmd/internal/couchcore/startup.go:32-103,156-163`
- Test: `cmd/internal/couchcore/startupselect_test.go:77-118`,
  `cmd/internal/couchcore/startup_proof_test.go:214`

- [ ] **Step 1: Write the failing tests.**
  - `TestSelectResumableRootMatchesTheRepositoryNotThePath`: a parked row at `/w/repo` in scope `S`, selected
    from `/w/repo/sub`, returns that address. A row in another scope does not.
  - `TestScopeHoldsUsableThreadFromASubdirectory`: live, detached and parked rows in `S` hold; unusable rows
    do not (resolved ambiguity 9).
  - Extend `TestNarrowedStartupAnswersAsAFullProofWould` with a fixture whose `:0` record sits at `/repo`
    while startup runs at `/repo/sub`. Narrowed and full inventories must agree.
- [ ] **Step 2:** Run `SCRUB go test ./cmd/internal/couchcore -run
      'TestSelectResumableRoot|TestScopeHolds|TestNarrowedStartup' -count=1`. Expected: FAIL.
- [ ] **Step 3: Implement.** The signatures become `SelectResumableRoot(rows, repoScope)` and
      `ScopeHoldsUsableThread(rows, repoScope)`. Restrict both to ordinary targets (`row.Target.Kind !=
      ThreadTargetSlot`). `startupAsks(requested, repoScope)` asks for every record in the scope. Update the
      reader list in `startupAsks`' doc comment, which is its one home.
- [ ] **Step 4:** Re-run, plus `SCRUB go test ./cmd/internal/couchcore -run 'Startup|Select|Occupancy'
      -count=1`. Expected: PASS.
- [ ] **Step 5: Commit** `#363 M3: startup: one primary per repository scope`.

### Task 3.2: Startup in a subdirectory resumes the primary

**Files:** `cmd/internal/couchcore/startup.go:182-230`; test `cmd/internal/couchcore/startup_test.go` (or the
file holding `StartInteractive` tests)
- [ ] **Step 1:** Write the failing test `TestStartInteractiveInSubdirectoryResumesTheExistingPrimary`.
      Through `StartInteractive`, with a parked `:0` at the repository root and args `Cwd: <root>/sub`, it
      resumes that address. Assert that no record was created (the manifest count is unchanged) and that
      `ResumeContextWith` ran (a child was spawned with the old tag).
- [ ] **Step 2:** Run. Expected: FAIL (today a second primary is created).
- [ ] **Step 3:** Pass the scope to the new predicates.
- [ ] **Step 4:** Run. Expected: PASS.
- [ ] **Step 5:** Commit.

### Task 3.3: Console start refuses a second primary

**Files:** `cmd/internal/couchcore/couch.go:475-490`; test `cmd/internal/couchcore/couch_test.go`
- [ ] **Step 1:** Write the failing test `TestSpawnInSubdirectoryOfALivePrimaryRefuses`. Through
      `SpawnPrepared`, the error contains "one primary slot per repository" and the existing label. No record
      is created and no child is spawned.
- [ ] **Step 2:** Run. Expected: FAIL.
- [ ] **Step 3:** Reword the refusal: `"%s already has its primary thread %s; couch keeps one primary slot per
      repository\n  return to it: ctrl-space, select it, Enter\n  start fresh: ctrl-space, select it, Tab →
      reboot\n  inspect it: couch --show %s"`.
- [ ] **Step 4:** Run. Expected: PASS.
- [ ] **Step 5:** Commit.

### Task 3.4: Pin the non-Git behavior (resolved ambiguity 1)

**Files:** test `cmd/internal/couchcore/couch_test.go`
- [ ] **Step 1:** Write `TestStartInANonGitDirectoryRefuses`. `FakeGit` returns an error for `rev-parse
      --show-toplevel`. `PrepareStart` refuses, with no record and no child. If it does **not** refuse, stop
      and add the non-Git row kind to Task 2.1's table (no add-slot, no alias; a second start refuses through
      `ScopeHoldsUsableThread`) before closing.
- [ ] **Step 2:** Run. Expected: PASS against today's code. Then revert `Resolve`'s error return temporarily
      and confirm the test fails.
- [ ] **Step 3:** Record the outcome in `## Log` and README ("couch starts only inside a Git repository"), if
      README does not already say so.
- [ ] **Step 4:** Commit.

### Task 3.5: Close M3

- [ ] README and atlas one-primary paragraph (`atlas/couch.md` startup section; `README.md` startup section).
- [ ] Full verification (Chunk 4). Ask the operator to run `couch` from a repository subdirectory live.
- [ ] `sdlc milestone-close --issue 363 --milestone M3`, then `sdlc close --issue 363 --verified
      '<evidence>'`.

---

## Chunk 4: Verification recipe (every milestone close)

Run in this order and redirect output to scratchpad files. Never pipe to `head`, which reports a phantom FAIL
through SIGPIPE.

1. `env -u PAIR_DATA_DIR -u PAIR_TAG -u PAIR_RETENTION_PROTOCOL -u PAIR_RETENTION_BACKGROUND -u
   PAIR_RETENTION_START_ID make -k test > $SCRATCH/make-test.log 2>&1`. Expected: only `test-changelog` fails.
2. `env -u … TMPDIR=$SCRATCH/tmp make test-changelog > $SCRATCH/changelog.log 2>&1`. Expected: PASS.
3. `env -u … go test ./... -count=1 > $SCRATCH/go-test.log 2>&1`. Expected: only
   `TestBareCouchInstalledCommand`, `TestProductionArtifactReferencesAreExactlyClassified` and
   `TestCouchReferencesLocalArchiveLocatorRoundTrip` fail (they fail on main too). Confirm on main with the
   same command if the set differs.
4. PTY-dependent tests (`ptychild`, console attach) report "operation not permitted" inside the sandbox.
   Re-run those packages with the sandbox off before calling them failures.
5. Paste the pass/fail summary into `--verified` and `## Log`.
