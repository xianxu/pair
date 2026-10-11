# Boundary Review — pair#437 (whole-issue close)

| field | value |
|-------|-------|
| issue | 437 — couch --session-sync: the one mutation path for a session-sync file, with an owed-events ledger |
| repo | pair |
| issue file | workshop/issues/000437-couch-session-sync-the-one-mutation-path-for-a-session-sync-file-with-an-owed-events-ledger.md |
| boundary | whole-issue close |
| milestone | — |
| window | f6e650f0ebfda91bdd5ab119e39638d88932219e..83fd79abe3e13d08dd08d98df11fb75e2b868e9f |
| command | sdlc close --issue 437 |
| reviewer | claude |
| timestamp | 2026-10-11T00:28:49-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I'm partway through reviewing #437. The `sessionsync` package and the CLI look sound so far. Next I'm checking the daemon wiring and running the tests.

Tests pass. Now I'm checking the tests themselves and the docs.

I'm finishing the review: checking how couch already validates slot addresses, then writing the verdict.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The core works and matches the Spec. The pure `sessionsync` package (sections, restamp, owner/holder parsing, owed computation) is cleanly separated, and the guards are tested. The daemon's delivery recording is tested end to end through the real message service. I ran `go test -race ./cmd/internal/sessionsync/` and the couchcmd `SessionSync|Deliver|...` tests, and they pass. The couchcmd tests needed the sandbox off because they `mkdir /tmp`. All four in-repo Done-when rows have tests that pin them. Four Important gaps stand between this and SHIP:
- `release` refuses with an instruction it can't accept.
- Slot addresses are checked by a second, narrower copy of the slot grammar, and one bad append disables `status` for the whole store.
- A Couch restart turns a lost ledger row into "nothing owed".
- README.md doesn't mention the new verb.

None is a crash or data loss, and each fix is cheap.

1. **Strengths**
   - `ReplaceSection` (`cmd/internal/sessionsync/file.go`) matches whole heading lines and refuses unless exactly one matches. The "quoted heading" Done-when holds by construction, not by heuristic.
   - `Owed` is pure: the symlink-resolving `canonical` function is passed in. `status`, `--fold` validation and `release` all call it (ARCH-DRY, ARCH-PURE).
   - The recording rule errs toward "owed": endpoint errors and non-terminal answers are recorded, and only cancel or expiry is skipped. `TestDeliveryRecordingKeepsEverythingThatMayHaveLanded` covers this as a table.
   - Every failure path for "nothing owed" refuses instead: no marker, a failed append, an invalid `owner:`. ARCH-SECURE is satisfied here; parse failures show up as refusals.
   - The operator-lock test compares file, shadow, sidecar and lock bytes before and after, so "changes nothing" is actually checked.

2. **Critical**: none.

3. **Important**
   - **`release` can't follow its own instruction** (`sync.go:147`, `session_sync.go:43`). `Release` calls `guard(false)`. The unread-edits refusal says "pass --accept-edits", but `release` only allows `--fold`, so following the message gives a usage error (exit 2). An agent that has read the operator's edits and has nothing to write can't release without making a write. Fix: let `release` take `--accept-edits` and refresh the shadow, or make the message depend on the verb.
   - **A second, narrower slot grammar** (`ledger.go:54`; ARCH-DRY). `slotAddress` (`^[A-Za-z0-9._-]+:[0-9]+$`) copies couch's slot grammar, but `couchcore.ParseWorkspaceReference` / `validFamily` accept any UTF-8 family without `/`, `\`, `:` or whitespace. A delivery to a valid slot outside the regex fails `Append`, which calls `MarkFailed`, and then **every** `status` in the store refuses `not-recording` until Couch restarts. Fix: validate with `couchcore.ParseWorkspaceReference` and encode the family safely in the filename.
   - **A restart hides a lost row** (`sync.go:330-349`, `message_deliveries.go:22`; ARCH-ORDER). `MarkRecording` at startup clears `Failed`, but the missing row is never recovered. Any file whose `Since` predates the failure then reports a gap as "nothing owed", and the refusal text ("incomplete until Couch restarts") promises more than a restart delivers. Fix: keep failure timestamps across restarts and refuse (or warn) for files whose `Since` is before the last failure, until the agent moves its horizon past it.
   - **README.md has no `--session-sync` line.** README.md:399-414 lists the couch CLI verbs (`--actors`, `--broadcast-list`, `--send-to`), and the new verb is missing there; it appears only in atlas, the skill and the usage text.

4. **Minor**
   - The fifth Done-when row (the xx-tl skill switches over) is still unchecked and depends on ops. At close, either get the TL's confirmation or add a Revision saying it's tracked outside this issue.
   - A new file with neither sidecar nor shadow starts its horizon at zero time, so it owes the slot's whole ledger (up to about 14 days and 512 rows). There's no fold-all. Starting from the file's mtime would be friendlier.
   - `writeAtomic` is about the fifth copy of temp+rename in the repo (`wrapcmd`, `couchcore`, `runtimebundle`, `changelogcmd`). The duplication predates this change.
   - `trim` keeps up to 512 young rows of up to about 4 KiB each, which can stay above `trimBytes`. If so, every append re-reads and rewrites the file. Unlikely in practice.
   - `Write` writes the file before the shadow. If the shadow write fails, the agent's own write later shows up as "unread edits".
   - A FILE argument starting with `--` is parsed as a flag. Only `--section` and `--fold` values are taken verbatim.

5. **Test coverage**
   - The four Done-when rows each have a pinning test, and the end-to-end delivery-to-owed path runs through `messageService.handle`.
   - Not covered:
     - `release` over unread edits after the agent has read them (the first Important finding).
     - A slot outside the regex (the second).
     - A restart after a failed append that still has an owed gap (the third). The current test asserts the restart clears the failure, which locks in the gap.
   - Sandbox note: the couchcmd delivery tests need a writable `/tmp`.

6. **Architecture**
   - ARCH-DRY: flagged (slot grammar).
   - ARCH-PURE: pass. `Owed`, `ReplaceSection`, `Restamp` and the parsers are pure; `Sync` is the IO layer with injected clock, diff and canonicalization.
   - ARCH-PURPOSE: pass. The ledger, guards and owed list deliver the liveness purpose; the ops skill switch is correctly ops' edit.
   - ARCH-MOCK: pass. `diff` is behind the injected `Diff`, and the ledger is a plain directory, so tests run against temp dirs.
   - ARCH-CONSTRAINTS: pass. One stat per append; the CLI reads at most about 256 KiB per call.
   - ARCH-SECURE: pass. `owner:` is validated before it becomes a path, the store files are 0600/0700, and only the first body line is kept.
   - ARCH-ORDER: flagged (restart collapses an unknown gap into "nothing owed"). The sidecar race is explicitly deferred in the plan, which is acceptable for one agent per file.
   - ARCH-FUNERAL: pass.
     - Ledger: bounded by size and age per slot.
     - `Folded`: pruned to IDs still in the ledger.
     - `recording.json`: a single file.
     - Shadow and sidecar: tiny and tied to the file, but they aren't removed when the file is deleted.

   For upcoming work: parley.nvim#314 consumes the `status --json` envelope. The `Uncertain` field (omitempty) goes beyond the agreed shape; it's harmless but worth telling them about.

7. **Plan revisions**
   - Add a Revision for whichever fix lands for the restart gap (Failed survives a restart, or the horizon is checked against failure times).
   - Add one stating that slot validation uses `couchcore.ParseWorkspaceReference`.
   - Add one for the fifth Done-when row's disposition (closed on TL confirmation, or tracked outside this issue).

```findings
findings:
  - id: new
    severity: Important
    family: refusal-names-unreachable-remedy
    title: |
      release refuses unread edits telling the agent to pass --accept-edits, which release does not accept
    detail: |
      Release calls guard(false) in sync.go, but parseSessionSyncCLI allows only --fold on release, so following the refusal is a usage error and the agent cannot release without writing. Accept --accept-edits on release (refreshing the shadow) or make the refusal text depend on the verb.
  - id: new
    severity: Important
    family: single-source-slot-grammar
    title: |
      ledger slotAddress regex is a second, narrower slot grammar; one out-of-grammar delivery disables status store-wide
    detail: |
      ledger.go:54 allows only [A-Za-z0-9._-] families while couchcore.ParseWorkspaceReference/validFamily accept any UTF-8 family without separators; Append then fails, MarkFailed stamps recording.json, and every status refuses not-recording until restart. Validate with couchcore and encode the family into a safe filename (ARCH-DRY).
  - id: new
    severity: Important
    family: uncertainty-collapsed-to-absence
    title: |
      a Couch restart clears Failed, so a lost ledger row silently reads as nothing owed
    detail: |
      MarkRecording at startup overwrites Failed without recovering the missing row; files whose Since predates the failure report the gap as complete. Keep failure timestamps across restarts and refuse or warn for files whose horizon precedes the last failure (ARCH-ORDER).
  - id: new
    severity: Important
    family: docs-surface-update
    title: |
      README.md missing the new couch --session-sync verb
    detail: |
      README.md:399-414 lists couch CLI verbs (--actors, --broadcast-list, --send-to); the new user-facing verb is absent there.
  - id: new
    severity: Minor
    family: done-when-traceability
    title: |
      Done-when row 5 (xx-tl switches to the verb) is unchecked and depends on ops
    detail: |
      Get the TL's confirmation before close, or add a Revision moving it out of scope.
  - id: new
    severity: Minor
    family: owed-horizon-default
    title: |
      a fresh file with no sidecar or shadow owes the slot's whole ledger history
  - id: new
    severity: Minor
    family: shared-atomic-write-helper
    title: |
      writeAtomic is about the fifth copy of temp+rename in cmd/internal
  - id: new
    severity: Minor
    family: ledger-trim-thrash
    title: |
      trim can stay above trimBytes with 512 long rows, rewriting the file on every append
  - id: new
    severity: Minor
    family: write-order-self-edit
    title: |
      Write writes the file before the shadow; a failed shadow write turns the agent's own write into unread edits
```

---

## Re-review — 2026-10-11T00:42:44-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 437 — couch --session-sync: the one mutation path for a session-sync file, with an owed-events ledger |
| repo | pair |
| issue file | workshop/issues/000437-couch-session-sync-the-one-mutation-path-for-a-session-sync-file-with-an-owed-events-ledger.md |
| boundary | whole-issue close |
| milestone | — |
| window | f6e650f0ebfda91bdd5ab119e39638d88932219e..b7c433831a5125b2d39992bc7ac223432a36c078 |
| command | sdlc close --issue 437 |
| reviewer | claude |
| timestamp | 2026-10-11T00:42:44-07:00 |
| verdict | SHIP |

## Review

Round 1 fix commit is e77a5d8e + b7c43383. Let me look at those and the core files.

Reviewing the round-1 fixes now; the sandbox blocks some couchcmd tests (mkdir /tmp), so I'll re-run the targeted ones scoped.

```verdict
verdict: SHIP
confidence: high
```

The round-1 fixes hold up, and every Important finding from that round is resolved. `release` now accepts `--accept-edits` and `--accept-gap`, so each refusal names a remedy the verb it came from takes. Slot validation now goes through couch's own `ParseWorkspaceReference`, and the address is escaped into the filename. The store-wide "failed" marker is gone. In its place each slot gets a `<slot>.gap` record that survives a Couch restart and is accepted per file. README, the atlas and the couch skill all describe the new surface. `go test ./cmd/internal/sessionsync/` passes. The couchcmd session-sync and delivery tests pass when run outside the sandbox; inside it they fail only on `mkdir /tmp/...`, which is the sandbox blocking the test socket directory. Three Minor findings from round 1 are still open (BR-6, BR-7, BR-9), and I found one new Minor issue. None of them blocks the close.

**1. Strengths**
- `sessionsync/sync.go:47` `Accepts` is one struct used by both `WriteRequest` and `ReleaseRequest`. That fixes the whole class of "refusal names a flag this verb doesn't take", not just the one site BR-1 named.
- `ledger.go:58` `ValidSlot` uses couchcore's grammar and adds a round-trip check (`ref.String() == slot`). `url.QueryEscape` keeps the filename safe whatever characters the repo family uses (ARCH-DRY).
- `gap.go` reasons correctly about uncertainty. The gap only matters for files whose horizon reaches it (`!gap.Last.Before(Since)`). A later failure makes it refuse again (`GapAccepted` vs `Last`). It expires after `keepFor`, the same window after which any row it could hide would have been trimmed anyway (ARCH-ORDER, ARCH-FUNERAL).
- `TestALedgerAppendFailureRecordsAGapForThatSlotOnly` drives the real `messageEndpoint.Deliver` path, simulates a restart with `recordDeliveries`, and checks that other slots are unaffected. That is real regression evidence for BR-2 and BR-3.
- `trim` now stops at `trimBytes/2` newest-first, so a trim can't be triggered again by the very next append. `TestLedgerTrimEndsWellUnderTheBound` pins this.

**2. Critical findings**
None.

**3. Important findings**
None.

**4. Minor findings**
- `gap.go:83`: if a gap record fails to parse, `readGap` makes one up with `Last: time.Now()`, and uses the wall clock rather than `l.now`.
  - Because `Last` changes on every read, `GapAccepted` can never catch up and `expireGap` never removes the record.
  - Result: `status` refuses with `delivery-gap` forever, and the only fix is the operator deleting the file by hand.
  - Fix: use the record's mtime for both `First` and `Last`, so the gap is stable and can expire.
  - The odds are low because `writeAtomic` prevents torn writes, but it is the ARCH-SECURE "fabricated value read as evidence" pattern.
- BR-6, BR-7 and BR-9 are still open; see the dispositions below.

**5. Test coverage notes**
- Write, release and status are each tested against the gap: it refuses, `--accept-gap` is accepted, a later gap refuses again, another owner is unaffected, a gap before the file's horizon is ignored, and an expired gap is removed.
- Delivery recording is tested for every receipt status.
- Not tested: a gap record that fails to parse (the Minor above), and a gap whose `First` is before `Since` while `Last` is after it. The code handles the second case correctly; there's just no explicit fixture for it.

**6. Architectural notes**
- **ARCH-DRY:** flag (Minor, BR-7). The shared helper `durablefile.WriteAtomic` already exists, but `sessionsync` still keeps its own `writeAtomic`. The slot grammar now has a single source.
- **ARCH-PURE:** pass. `Owed`, `ReplaceSection` and `Fold` are pure; `Canonical` is injected.
- **ARCH-PURPOSE:** pass. The purpose was a guarded mutation path plus an owed-events list, and both are delivered. Done-when row 5 moved to ops through a Revision.
- **ARCH-MOCK:** pass. The ledger is a portable folder of files. `diff` is injected through `Sync.Diff`.
- **ARCH-CONSTRAINTS:** pass. Each append costs one stat, and the size bound is enforced.
- **ARCH-SECURE:** flag (Minor, the fabricated gap time above). Slot names are now escaped before reaching the filesystem.
- **ARCH-ORDER:** pass. Gap, accept and later-gap states are explicit and tested in sequence. The window between the guard and the write is reported as a warning, which is acceptable.
- **ARCH-FUNERAL:** pass with a note. A `.gap` file expires only when that slot receives another append. A slot that never receives one leaves its record (and its `.jsonl`) behind; this is bounded by the number of slot addresses.

**7. Plan revision recommendations**
None needed. The round-1 Revision correctly supersedes the earlier line saying "`status` refuses `not-recording` … or an append failed (`recording.json`)".

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      parseSessionSyncCLI allows --accept-edits/--accept-gap on release; TestReleaseAcceptsReadEditsWithoutAWrite pins it.
  - id: BR-2
    disposition: addressed
    note: |
      ValidSlot delegates to couchcore.ParseWorkspaceReference; QueryEscape filename; TestLedgerUsesCouchsSlotGrammar plus per-slot gap test.
  - id: BR-3
    disposition: addressed
    note: |
      Per-slot .gap survives recordDeliveries restart (TestALedgerAppendFailureRecordsAGapForThatSlotOnly); refuses delivery-gap until accepted per file.
  - id: BR-4
    disposition: addressed
    note: |
      README.md:401 lists couch --session-sync.
  - id: BR-5
    disposition: addressed
    note: |
      Round-1 Revision moves Done-when 5 to ops, citing hand-off delivery 3c7ff078.
  - id: BR-6
    disposition: not-addressed
    note: |
      LoadState still returns zero Since with no sidecar or shadow, so a new file owes up to 14 days of rows; Minor, non-blocking.
  - id: BR-7
    disposition: not-addressed
    note: |
      sessionsync/ledger.go:201 writeAtomic is still a local copy; durablefile.WriteAtomic is the shared helper to reuse.
  - id: BR-8
    disposition: addressed
    note: |
      trim stops at trimBytes/2 newest-first; TestLedgerTrimEndsWellUnderTheBound.
  - id: BR-9
    disposition: not-addressed
    note: |
      Write still writes the file before the shadow; Minor, non-blocking.
findings:
  - id: new
    severity: Minor
    family: fabricated-evidence-on-parse-failure
    title: |
      readGap fabricates Last=time.Now() for an unparseable gap record, so it never expires and cannot be durably accepted
    detail: |
      Each read gets a fresh Last, so GapAccepted never covers it and expireGap never removes it; status refuses delivery-gap forever until the file is deleted by hand. Use the record's mtime as First/Last (and l.now, not time.Now).
```

---

## Re-review — 2026-10-11T01:15:02-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 437 — couch --session-sync: the one mutation path for a session-sync file, with an owed-events ledger |
| repo | pair |
| issue file | workshop/issues/000437-couch-session-sync-the-one-mutation-path-for-a-session-sync-file-with-an-owed-events-ledger.md |
| boundary | whole-issue close |
| milestone | — |
| window | cd2839c70420fb14119058522000ffb6024e77cf..7335c4b96df1bb05898dd5c3c48fcee69255bb97 |
| command | sdlc close --issue 437 |
| reviewer | claude |
| timestamp | 2026-10-11T01:15:02-07:00 |
| verdict | SHIP |

## Review

I've confirmed the build and focused tests pass. Next I'm checking for leftover temp+rename copies in sessionsync (BR-7) and how the BR-6/BR-9 notes were dispositioned.

```verdict
verdict: SHIP
confidence: high
```

**Verdict: SHIP.** This fix window does two things. It fixes the last two open findings, BR-7 and BR-10, each with real coverage. It also resolves the two merge conflicts with main correctly. Both conflicted files at the base commit `cd2839c` had committed conflict markers. At the head commit `7335c4b` the markers are gone, both sides' intent is kept, and the code builds. BR-6 and BR-9 have no change in this window; they stay open as Minor and do not block. The window does not touch `## Spec`, `## Done when` or `## Revisions`; it only records the fixes.

**What I checked**
- `go build ./...` and `go vet` on `sessionsync` and `couchcmd` both pass.
- `go test ./cmd/internal/sessionsync` passes.
- The focused delivery, ledger, gap and message tests in `go test ./cmd/internal/couchcmd -run 'Deliver|Ledger|Gap|SessionSync|Message'` pass when run outside the sandbox.
- The full `couchcmd` run failed inside the sandbox. The errors were "operation not permitted" from `mkdir /tmp/…` and a launcher helper, which is the sandbox blocking those calls, not a code fault.

1. **Strengths**
   - **Conflict in `message_service.go:848`:** the resolution keeps main's `progress` callback and this branch's delivery recording (`r.record(m, receipt, err)`). The test fakes in `message_deliveries_test.go` were updated to the new `Deliver` signature, so they compile and pass.
   - **Conflict in `workshop/lessons.md`:** both the #435 lesson and the #437 lessons are kept, and no markers remain.
   - **BR-10 (`gap.go:86-94`):** an unreadable gap record is now dated by the file's modification time instead of the current time. Its expiry and acceptance therefore go through the same `l.now()` check as a normal gap. The test `TestAnUnreadableGapIsDatedByItsFileSoItCanBeAccepted` pins three things: the refusal, that accepting the gap makes it durable, and that it expires. With the old `time.Now()` stamp this test would fail, because the stamp would be later than both the injected clock and the acceptance.
   - **BR-7 (`ledger.go:202-205`):** `writeAtomic` now hands off to `durablefile.WriteAtomic`. This covers the whole package: all six write sites in `sessionsync` (in `gap.go`, `ledger.go`, `state.go` and `sync.go`) go through the shared helper, and no `CreateTemp`/`Rename` copy is left in the package.

2. **Critical findings:** none.
3. **Important findings:** none.
4. **Minor findings:** none new.
5. **Test coverage:** the BR-10 test covers the refusal, the acceptance and the expiry. The conflict resolution is exercised by `TestDeliveryRecordingKeepsEverythingThatMayHaveLanded` and `TestALedgerAppendFailureRecordsAGapForThatSlotOnly`, which both pass.
6. **Architecture (interdiff only)**
   - **ARCH-DRY: pass.** The local copy of the atomic-write code is gone in favour of the shared helper.
   - **ARCH-PURE: pass.** The one new piece of file IO, an `os.Stat` call, sits in `readGap`, which already reads the file.
   - **ARCH-PURPOSE: pass.** BR-10 is fixed at the cause (the unreadable record now has a real date), not by special-casing the expiry check.
7. **Plan revisions:** none.

```findings
dispose:
  - id: BR-6
    disposition: not-addressed
    note: |
      No change in this window; LoadState still yields zero Since with no sidecar or shadow. Minor, non-blocking.
  - id: BR-7
    disposition: addressed
    note: |
      ledger.go:203 writeAtomic now delegates to durablefile.WriteAtomic; every sessionsync write site (gap, ledger, state, sync shadow and recording) uses it, so no temp+rename copy is left in the package.
  - id: BR-9
    disposition: not-addressed
    note: |
      No change to the order in sync.go Write; the file is still written before the shadow. Minor, non-blocking.
  - id: BR-10
    disposition: addressed
    note: |
      gap.go:86-94 dates an unparseable record by its mtime (First and Last), so Gap and expireGap judge it with l.now(); TestAnUnreadableGapIsDatedByItsFileSoItCanBeAccepted pins the refusal, the acceptance and the expiry, and would fail under the old time.Now() stamp.
```
