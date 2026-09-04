# Boundary Review — pair#181 (milestone M3)

| field | value |
|-------|-------|
| issue | 181 — One honest inventory: every thread gets a row and a reason |
| repo | pair |
| issue file | workshop/issues/000181-one-honest-inventory-every-thread-gets-a-row-and-a-reason.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 8e6f1af08201c97e10e972f61b9fe99e2dafcdce..f78e845c6e881530d0e3ad50195f57d5b1b6b79c |
| command | sdlc milestone-close --issue 181 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-09-03T22:51:05-07:00 |
| verdict | unknown |

## Review

Failed to authenticate: OAuth session expired and could not be refreshed

---

## Re-review — 2026-09-03T23:13:54-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 181 — One honest inventory: every thread gets a row and a reason |
| repo | pair |
| issue file | workshop/issues/000181-one-honest-inventory-every-thread-gets-a-row-and-a-reason.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 8e6f1af08201c97e10e972f61b9fe99e2dafcdce..8b12dbd6f85c082695488c74714973394a0799b8 |
| command | sdlc milestone-close --issue 181 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-09-03T23:13:54-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M3's model is right and most of it is well built — the archive is journaled as one atomic entry, quiesce-before-move is pinned by an order test, the startup reversal is a pure ranked selector with real cases, and labels-from-directories has a collision guard. What blocks SHIP is that the milestone's headline operator action does not work: the switcher dispatches `archive` with `{repo-scope, tag}` and the direct-store executor resolves only `a["ref"]`, so every Tab → archive → confirm fails with `thread reference not found: empty reference`. I confirmed this empirically through the real CLI runtime (probe below), not by reading. Behind it sit three Important items: the `invalid` reason is unreachable in production while one undecodable record fails the *entire* inventory (contradicting the Spec's "every record gets a row, no exceptions" and its `invalid → archive, inspectable` exit), README/atlas still assert the exact rules this window reversed (including "it does not refuse a start because a tree is already busy", which is now false), and the new refusal message's two "ways forward" are both wrong.

**Test evidence.** `env -u PAIR_SESSION_ID -u PAIR_TAG go test ./cmd/... -count=1`: every non-pty test passes; the only failures are the documented sandbox class (`ptychild: … operation not permitted`, `fork/exec /bin/ps`, `mktemp`), which includes `TestInteractiveLaunchReattachesUniqueDetachedRoot` — M2's warm-reattach assertion cannot be verified in this environment. Probes were run with `go test -overlay`, so the repo tree was never modified (verified clean).

## 1. Strengths

- `threadstore.go:988-1000` — the archive, the record deletion and the manifest rewrite are one `storeJournal` with correct expected/after images; `applyJournalEntry`'s idempotent replay makes crash recovery real rather than asserted.
- `detach.go:170-190` + `archive_test.go:120-140` — quiesce-first is pinned by the *order* property (a Quiesce failure must leave the record in the working set), not just by "both effects happened". That is the test that would fail if someone reordered the calls.
- `startup.go:5-45` / `startupselect_test.go` — the selector reversal is argued in place against the claim it replaces, and the ratchet ("two rows make a third") is exercised by the crowded-path case.
- `resume.go:230-241` — resolving the binding only where it is the authority is the right shape: the warm path's precondition is re-proved by `confirmStillDetached`, mirroring `RequireNativeResumeBinding` rather than skipping the recheck.
- `threadinventory.go:99-111` — `BuildArchivedInventory` refusing to classify archived rows (instead of letting them render "checking…") is the correct fix at the correct layer.

## 2. Critical findings

**C-1 — `archive` is unreachable from the only surface that presents it.** `cmd/internal/couchcore/operationdispatch.go:167`

`DirectStoreExecutor`'s archive case resolves `c.ResolveThreadReference(a["repo-scope"], a["ref"])`, but the switcher's dispatch is `threadEffect` (`couchtty/menu.go:1454`), which sends `{repo-scope, tag}` and no `ref`. `normalizeThreadReference("")` errors, so the confirmed archive fails. `name`/`describe` only work because `reduceTextKey` (`menu.go:643-647`) hand-builds a `ref`; the live-owner ops work because `resolveOperationThread` (`operationdispatch.go:321`) accepts *either* `tag` or `ref`. `archive` is the first direct-store op dispatched through `threadEffect`, and nothing crosses that seam in a test — `menu_test.go:1310` asserts the effect is *emitted*, and `archive_test.go` calls `couch.ArchiveThread` directly.

Proof, through `RunWithRuntime`:
```
archive(tag): code=1 err="couch: thread reference not found: empty reference"
archive(ref): code=0, records left=0     # same call, ref instead of tag
--archived:   code=0 out="repo  /repo\n  archived (restore by moving it back …)"
```
Fix: replace lines 164-175 with `address, err := resolveOperationThread(c, a)` and dispatch on that — it already handles `tag`, `ref`, and the both-supplied refusal. Consider moving the `name`/`describe`/`show` cases onto it too, so one dialect serves both executors. Pin it with `runTypedRT(rt, OperationCall{Name:"archive", Args:{"repo-scope":…, "tag":…}, Implicit:true})` in `couchcmd` — the seam, not either side of it.

## 3. Important findings

**I-1 — an unreadable record has no row, cannot be archived, and takes the whole inventory down with it.** `cmd/internal/couchcore/threadstore.go:517-521`

`Snapshot` returns an error if *any* manifest-listed record fails `decodeThreadRaw`, and `DecodePersisted` already runs `ValidatePersisted` = `Validate` + address match. So `ClassifyThread`'s `ValidateThreadRecord(record) != nil → ReasonInvalid` branch (`actionableinventory.go:211`) cannot fire in production: a record that would be `invalid` instead makes `couch --list` exit 1 and the switcher show nothing. The reason is nonetheless in `AllThreadReasons()`, has a label, has an Enter notice, and the atlas now promises it an archive exit — but `ArchiveThread` decodes before moving (`threadstore.go:967-973`) and returns the decode error, and `ArchivedThreads` silently skips what it cannot decode (`threadstore.go:1027-1032`, whose comment says the record "is still evidence the operator may want" while dropping it). Four sites, one rule: **a record the decoder rejects must still produce a visible row; it must never remove other rows or vanish.** That is this issue's own thesis, applied to the one shape #181 never tested end-to-end — `couchWithOneRecordOfEveryShape` (`classify_test.go:274`) has six shapes and no invalid one, because `CreateThread` won't accept it. Fix sketch: have `Snapshot` carry undecodable addresses (a `Malformed []ThreadAddress`, or a record shape flagged invalid) so the projector emits a `ReasonInvalid` row; let `ArchiveThread` move a record from raw bytes without decoding it; let `ArchivedThreads` emit an address-only archived row. Test by writing a corrupt/`schema_version: 99` file into a real store and asserting `--list` still lists N rows with one `invalid`, and that archiving it works. (`ARCH-PURPOSE`, `ARCH-SECURE` — the failure path should degrade visibly.)

**I-2 — README and atlas still assert the exact rules this window reversed.** *(2nd finding in family `unbacked-existing-behavior-claim`.)* Earlier rounds fixed the instance (the "36 call sites" count). Do not fix these sites one by one — **state and apply the rule: a milestone that reverses a documented rule sweeps every prose restatement of that rule in the same commit, enumerated by grepping the superseded symbol name and the rule's distinctive phrases before the boundary is crossed.** Measured prevalence for this window, 7 sites:

- `README.md:310-312` — "Any number of threads may share one path: Couch is … not a gatekeeper, and it does not refuse a start because a tree is already busy." The opposite is now enforced (`couch.go:350`).
- `README.md:299-302` — "resumes the sole exact resumable thread … With zero or multiple exact candidates, Couch starts a new thread instead; it does not rank, prompt, or guess." It ranks now.
- `README.md:277` — "otherwise the opaque tag is the label." It is the directory base (`threadLabel`).
- `README.md:267-270` — the `couch [<repo>] / --list / --show` block omits `--archived`, which `usage()` in `run.go:659` prints.
- `atlas/couch.md:516-532` — applies `SelectUniqueResumableRoot` (deleted symbol), "Exactly one matching resumable row", "adds no ranking", "Exactness is *preserved*, not relaxed". All four false; `PathHoldsUsableThread` has no atlas entry at all, nor does the label rule.
- `atlas/couch.md:541-558` — the native-binding gate "drops any whose binding is not one exact established root" and "a detached thread … is now hidden from the switcher entirely": M1 stopped dropping, M2 stopped requiring the binding on the warm path.
- `cmd/internal/couchcore/threadreason.go:34` — a comment naming `SelectUniqueResumableRoot`.

**I-3 — occupancy is decided in five places with four different definitions.** *(2nd finding in family `new-state-unhandled-at-consumers`.)* Earlier rounds fixed instances. **The rule: "can this thread be acted on / is it occupied" must be one predicate over the classified state, and one over incarnation states, shared by every consumer — not re-derived per call site.** Current enumeration:

| site | admits | omits |
|---|---|---|
| `menu.go:966 menuThreadActionable` | live/parked/detached | busy, unusable |
| `menu.go:1006 menuActionItems` | offers `archive` to everything non-actionable | — **includes `ThreadBusy`** |
| `threadstore.go:974-981 ArchiveThread` | refuses `Park != nil` + `IncarnationLive` | **admits `creating`/`unknown`** |
| `resume.go:80-95 DecideResume` | refuses live + creating + unknown + park | — |
| `startup.go PathHoldsUsableThread` | live/detached/parked block a start | busy does not block |

Concrete failure (reachable once C-1 is fixed): Tab → archive on a row that is mid-start (`creating` incarnation, process observable → `ThreadBusy`) passes the store guard, so `Quiesce` kills the session being created and the record leaves the manifest while the spawn is still in flight — the "console owns a thread the store no longer lists" state that guard exists to prevent. Minimum fix: `ArchiveThread` refuses any occupied incarnation, matching `DecideResume`'s set; better, one exported predicate both call.

**I-4 — the one-thread-per-path refusal offers two next steps, neither of which works.** `cmd/internal/couchcore/couch.go:351-355`

`"return to it: couch <path>"` — in the interactive case this guard only fires when a **live** row holds the path (a resumable one would have been selected), so the suggestion is the command the operator just ran, and it will refuse again. From the TUI start form the operator is already inside couch, and a second `couch` cannot take the supervisor lease. `"or retire it: couch --show <tag>"` — `--show` is a read-only listing; there is no CLI archive (`archive` is `PresentationTUI`), so this cannot retire anything. `couch_test.go:1347-1352` pins both strings, so the test currently guards the wrong text. Suggested: name the switcher gestures (`Enter` on its row to return, `Tab → archive` to retire) and keep `couch --show <tag>` labelled as *inspect*.

## 4. Minor findings

- `archive_test.go:59` — `TestArchiveThreadRefusesALiveOrParkingThread` never builds a `record.Park != nil` case; the mid-park branch (`threadstore.go:974`) is unexercised despite the name.
- `startup_test.go:132-143` — named `…ResumesTheNewestOfSeveralParkedCandidates` but asserts only "one of the two"; the recency rule is pinned solely in the pure test.
- `menu_render.go:295-302` and `run.go:578-583` build the identical `[]LabelRow` adapter loop; one shared helper taking the two summary types would remove the copy (`ARCH-DRY`).
- `threadstore.go:986` — re-archiving a tag whose archive file exists with *different* bytes fails with "neither expected-before nor exact after-image", a message that will not tell an operator what happened after a manual restore.
- The issue's Plan still shows `- [ ] M2` unticked with no `closed M2` log line, though all of M2's code is inside this window and is being closed as M3.

## 5. Test coverage notes

- The gap that shipped C-1 is structural: no test dispatches a menu-originated operation through `DispatchOperation`. `runTypedRT` already exists and would have caught it in three lines. Add one per TUI-dispatched direct-store op.
- `couch --archived` has no end-to-end test (parse → dispatch → render); I verified it works by probe, but nothing pins it. The `archived` row's state text (`run.go:630`) is likewise unpinned.
- `ThreadBusy` has no behavioural test in `couchtty` at all — `menuThreadActionable` excluding it is unpinned, which is why BR-1's fix cannot be verified by a failing test (see disposition).
- `TestInteractiveLaunchReattachesUniqueDetachedRoot` — M2's headline claim — is pty-gated and unrunnable in this environment. If the M2 evidence was gathered on the live store, the close's `--verified` should say so explicitly rather than citing a suite that skips it here.

## 6. Architectural notes

- **ARCH-DRY** — flagged twice: two thread-resolution helpers with different accepted argument sets behind one menu dispatcher (C-1), and four occupancy definitions (I-3). Both are "the same fact computed per call site", the shape #181 exists to remove.
- **ARCH-PURE** — pass on the new pure entities (`SelectResumableRoot`, `PathHoldsUsableThread`, `threadLabel`, `DisambiguateLabels`, `startupResumeRefusal` are all tested without IO). One note: the archive *policy* lives inside the store's IO method, re-deriving from incarnations what the classifier already computed; a pure `ArchivableState(...)` injected into the thin store call would fix I-3 structurally.
- **ARCH-PURPOSE** — flagged: C-1 (the action the milestone is *for* never reaches the operator) and I-1 (`invalid`'s declared exit is documented, rendered, and unreachable). The shadow-sweep over the docs consumers is I-2.
- **ARCH-MOCK** — pass. `Quiesce` goes through the injected `Artifacts` seam with a stateful fake that records calls and supports failure injection; production and test flow share the boundary. Gap noted: the operation-dispatch boundary has no equivalent coverage.
- **ARCH-CONSTRAINTS** — pass with a note. `Spawn`/`SpawnPrepared` now pay a full evidence pass (possible zellij snapshot) per start; `StartInteractive` correctly threads its existing rows through instead. `DisambiguateLabels` is O(n) per render on the keystroke path — bounded and cheap at switcher scale, but no budget is stated for the added start-path cost.
- **ARCH-SECURE** — pass on inputs (`validateThreadAddress` before path construction, decode enforces address/file agreement, journal paths are traversal-checked, no credentials touched). Flagged under I-1: `ArchivedThreads` degrades *invisibly* on a record it cannot parse, which is the failure mode this principle asks to be visible.

## 7. Plan revision recommendations

- Add to the M3 Revisions entry the landed-entity list, as the M1 entry did: the shipped names are `ThreadStore.ArchiveThread` (the plan's table says `ThreadStore.Archive`), `ThreadStore.ArchivedThreads`, `Couch.ArchiveThread`, `BuildArchivedInventory`, `ThreadArchived`, plus the unplanned `SelectResumableRoot`, `PathHoldsUsableThread`, `threadLabel`, `LabelRow`/`DisambiguateLabels`, `startupResumeRefusal`, `confirmStillDetached`, `ResumeSessionGone`, and `trackedThreadLaunch.Warm`.
- Record that the Spec's `invalid → archive, inspectable` exit is **not delivered** (I-1), with either the fix or an explicit follow-up issue — the Done-when line "rows == records, always, with no exceptions and a test that asserts the equality over a store containing every reason" is not satisfied by a six-shape fixture that structurally cannot hold an invalid record.
- Correct `workshop/issues/000181…md:161` — the estimate table still says "36 call sites in 6 files"; measured at HEAD it is 63 references across 15 files (the plan body at `:463` and the issue at `:192` are already correct).

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      menuThreadActionable now gates Enter and the action list, but nothing pins the busy case and archive is offered on busy rows — see the rule-level finding.
  - id: BR-2
    disposition: not-addressed
    note: |
      plan:463 corrected to 62/15 (measured 63/15 at HEAD); issue:161 still says "36 call sites in 6 files".
findings:
  - id: new
    severity: Critical
    family: dispatch-arg-contract-mismatch
    title: |
      Tab-archive always fails — the switcher sends "tag", the executor reads "ref"
    detail: |
      operationdispatch.go:167 resolves a["ref"] while couchtty/menu.go:1454 threadEffect sends {repo-scope, tag}.
      Confirmed through RunWithRuntime: archive(tag) exits 1 with "thread reference not found: empty reference";
      the same call with ref archives the record. archive is the first direct-store op dispatched via threadEffect,
      and no test crosses the dispatcher seam. Fix: use resolveOperationThread(c, a), which accepts either form.
  - id: new
    severity: Important
    family: decode-failure-drops-the-row
    title: |
      An unreadable record has no row, cannot be archived, and fails the whole inventory
    detail: |
      Snapshot (threadstore.go:517) errors on any undecodable record, and DecodePersisted already runs Validate, so
      ClassifyThread's ReasonInvalid branch cannot fire in production — one bad record empties both views instead of
      producing one honest row. ArchiveThread decodes before moving, so an invalid record can never leave; ArchivedThreads
      silently skips what it cannot decode. Four sites, one rule: a record the decoder rejects must still produce a
      visible row and must never remove other rows.
  - id: new
    severity: Important
    family: unbacked-existing-behavior-claim
    title: |
      README and atlas still assert the startup and label rules this window reversed
    detail: |
      2nd in family — state the rule, do not patch instances: a milestone reversing a documented rule sweeps every prose
      restatement in the same commit, enumerated by grepping the superseded symbol and the rule's distinctive phrases.
      Measured 7 sites: README:310-312 ("does not refuse a start because a tree is already busy"), README:299-302
      ("does not rank"), README:277 (tag is the label), README:267-270 (--archived missing); atlas:516-532 (names the
      deleted SelectUniqueResumableRoot, "exactness preserved"), atlas:541-558 (binding gate / rows hidden);
      threadreason.go:34 (dead symbol name). PathHoldsUsableThread and the label rule have no atlas entry at all.
  - id: new
    severity: Important
    family: new-state-unhandled-at-consumers
    title: |
      Occupancy is decided in five places with four different definitions
    detail: |
      2nd in family — state the rule: "can this be acted on / is it occupied" must be one predicate over the classified
      state and one over incarnation states, shared by all consumers. Sites: menuThreadActionable (menu.go:966),
      menuActionItems (menu.go:1006, offers archive to ThreadBusy), ThreadStore.ArchiveThread (threadstore.go:977,
      refuses only IncarnationLive), DecideResume (resume.go:80-95, refuses live+creating+unknown),
      PathHoldsUsableThread (busy does not block). Failure: archiving a mid-start row quiesces the session being created
      and unlists the record while the spawn is in flight.
  - id: new
    severity: Important
    family: unnavigable-refusal
    title: |
      The one-thread-per-path refusal names two next steps, neither of which works
    detail: |
      couch.go:351-355 suggests "couch <path>" — the command the operator just ran (the guard fires interactively only
      when a live row holds the path) and unavailable from inside the TUI (supervisor lease) — and "couch --show <tag>"
      to "retire it", but --show is read-only and there is no CLI archive. couch_test.go:1347 pins both strings.
  - id: new
    severity: Minor
    family: test-name-overclaims
    title: |
      TestArchiveThreadRefusesALiveOrParkingThread never exercises the parking branch
    detail: |
      archive_test.go:59 builds only a live incarnation; threadstore.go:974's Park != nil refusal is unpinned. The same
      shape appears at startup_test.go:132, named "ResumesTheNewest" while asserting only "one of the two".
```

---

## Re-review — 2026-09-04T07:35:28-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 181 — One honest inventory: every thread gets a row and a reason |
| repo | pair |
| issue file | workshop/issues/000181-one-honest-inventory-every-thread-gets-a-row-and-a-reason.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | abc39d62baaeedb8057caa05d94eb113dc1d3b1b..f9f6cdd6ee27163095b514b69d306897dbd13ced |
| command | sdlc milestone-close --issue 181 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-09-04T07:35:28-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The M3 code is in good shape and the round-3 Critical is genuinely dead: `resolveOperationThread` is now the single dialect, and reverting it turns both new seam tests red (I verified by overlay revert). What blocks SHIP is threefold. First, the window's *only* commit (`f9f6cdd6`, the project-file detail blocks) leaves the tree failing a real test — `TestUncheckedProjectMilestoneHasNoClosedMetadata` — because the M3 block carries `**closed:** 2026-09-04` / `**actual:** 2.4h` while its row is unticked; those two values are gate-owned output (`sdlc milestone-close` "updates the project file's task row + detail block"), hand-written ahead of the gate that measures them, and the same commit records a measured close for M2, which never crossed a milestone-close at all. Second, BR-4 is claimed fixed but is only fixed *below the seam*: through the real dispatcher, archiving an undecodable record still fails (`archive` with `tag` → `couch: EOF`; with `ref` → `thread reference not found`), and the new test calls `couch.ArchiveThread` directly — the exact pattern the commit's own lesson ("test the seam, not both sides of it") was written about. Third, BR-8's new park-branch test is non-discriminating: I deleted the `record.Park != nil` refusal entirely and the test stayed green, because the same fixture also carries a live incarnation.

## 1. Strengths

- **`operationdispatch.go:164-198` + `run_test.go:1443-1497`** — the round-3 Critical was fixed as a *dialect*, not a site, and writing the seam test surfaced that `name`/`describe` never declared `tag` at all. Revert-proof: restoring `ResolveThreadReference(a["repo-scope"], a["ref"])` makes both new tests fail with `empty reference`.
- **`archive_test.go:225-249`** — `TestArchiveRefusesEveryOccupiedIncarnationNotJustLive` is a table over live/creating/unknown that also asserts the record did not move. That is the test that would have caught the mid-start kill, and it is discriminating.
- **`atlas/couch.md:513-577`** — the startup reversal is argued against the claim it replaces, quoting the superseded sentence, and it noticed the atlas had *pre-recorded the fork* ("the fork to revisit if an operator hits this") and closed the loop. That is unusually good archaeology.
- **`workshop/lessons.md:3074-3109`** — five rules, correctly pitched at the class rather than the site; "two green halves are not evidence of a working whole" is the right generalization.
- **`actionableinventory.go:562-570`** — `LabelsFor` consolidated the duplicated `[]LabelRow` adapter loop flagged as a round-3 Minor (`ARCH-DRY`), quietly and correctly.

## 2. Critical findings

**C-1 — the window's only commit ships a red test and pre-empts the gate that owns its numbers.** `workshop/projects/couch.md:198,300-306`

`env -u PAIR_SESSION_ID -u PAIR_TAG go test ./cmd/internal/couchcore/ -run TestUncheckedProjectMilestoneHasNoClosedMetadata` → `FAIL: unchecked milestone pair#181 M3 carries closed metadata`. Green at `abc39d62`, red at `f9f6cdd6`. **This is the 3rd finding in family `unbacked-existing-behavior-claim`** — so do not just untick-or-tick this one block. The rule: **a state or number in a portfolio artifact is written by the gate that produced it; if a value was judged rather than measured, it says so where it is read.** Measured enumeration for `#181`, all three blocks added by this commit:

| block | claim | backing |
|---|---|---|
| M3 | `closed: 2026-09-04`, `actual: 2.4h` | no close has run; row is `- [ ]`; contract test red |
| M2 | `closed: 2026-09-03`, `actual: 0.9h` | no `Review-Verdict` trailer, no `closed M2` log line, issue `## Plan` still `- [ ] M2`; the project row was hand-ticked in `6572ef69` |
| M1 | `actual: 0.85h` | the issue Log says this is "labeled judgment estimate, not measured — `sdlc actual --issue 181` reports no measurable activity"; the block drops that qualification |

M2's *code* was reviewed (the M3 window starts at M1's close, so rounds 2-3 covered it) — what is missing is the recorded close and a measured actual, and `**actual:**` feeds velocity calibration, which is precisely what AGENTS.md §5 forbids hand-typing. Fix: let `milestone-close` write `closed:`/`actual:` for M3; either run M2's close or mark its numbers as derived-from-the-M3-window; restore M1's provenance note. Sweep: every `**actual:**` in `couch.md` (20 today) whose milestone lacks a `Review-Verdict` trailer.

## 3. Important findings

**I-1 — `Snapshot` reports "I could not read it" as the verdict "this record is invalid", and drops it from the record set.** `threadstore.go:517-537`

`os.ReadFile` failure and `decodeThreadRaw` failure both become `Malformed → ReasonInvalid` ("unreadable record", whose documented exit is archive). `DecodePersisted` rejects an unknown `schema_version` outright (`threadrecord/record.go:204`) and `strictjson` rejects unknown fields, so **an older couch reading a store written by a newer one classifies every thread as debris**. Because the record also leaves `snapshot.Records`, `PathHoldsUsableThread` stops seeing it, so `couch <path>` creates a fresh thread at a path that already holds live work — the exact ratchet M3 just closed, now reachable silently where the old code failed loudly. This is the same shape as M1's own `ProofStatus` lesson: a total classifier that cannot say "I could not ask" converts one failed read into an assertion. Fix: separate `Unreadable` from `Invalid` (a `ReasonUnreadable`, or a `ProofStatus`-style qualifier on the malformed address), and make an unreadable-but-manifest-listed record *block* a path rather than freeing it. (`ARCH-SECURE` — input crossing a version boundary; `ARCH-PURPOSE`.)

**I-2 — the malformed row set is an opt-in variadic, so the pre-`#181` behavior is the default.** `actionableinventory.go:174`, `threadinventory.go:49`

`ProjectActionableThreads(records, evidence, malformed ...[]ThreadAddress)` compiles fine when the third argument is omitted, and omitting it silently restores "some records get no row" — the regression this whole issue exists to prevent, with no compile error and no test that would notice. **This is the 3rd finding in family `new-state-unhandled-at-consumers`.** Do not fix the two call sites. The rule: **evidence and the records it describes travel as one value, so a consumer cannot opt out of part of the projection.** Pass `ThreadSnapshot` (or a `ThreadInventoryInput{Records, Evidence, Malformed}`) to both projectors; the ~29 call sites already moved once for the evidence parameter, and this makes the next omission a compile error instead of a hidden filter.

## 4. Minor findings

- `detach.go:179-187` — on an unreadable record `Couch.ArchiveThread` returns a synthesized `ThreadRecord{Address: address}` as the archive *result*; downstream `ResultThread` renders it as a real record with every field zero (`ARCH-SECURE`: a fabricated value read as evidence). An address-only result type would be honest.
- `threadinventory.go:113` — `BuildArchivedInventory` passes `nil` records-only, so archived rows recovered address-only by `ArchivedThreads` are never end-to-end tested (`couch --archived` on a corrupt archive file).
- `workshop/lessons.md` gained five rules but not the one BR-8's family names: a test named for a specific refusal branch must go red when that branch is deleted.

## 5. Test coverage notes

- `go test ./cmd/... -count=1`: the only non-sandbox failure is `TestUncheckedProjectMilestoneHasNoClosedMetadata` (C-1). Everything else is the documented `operation not permitted` pty class, which again includes `TestInteractiveLaunchReattachesUniqueDetachedRoot` — M2's headline claim is still unverifiable here, and M2's `--verified` evidence should say so.
- The new seam test covers three ops in the switcher dialect on **healthy** threads only. The corrupt-record case never crosses the dispatcher, which is why BR-4's remainder survived (probe: `archive{tag}` → `couch: EOF`, `archive{ref}` → `not found`, `--archived` → `no threads`). Extend `TestSwitcherDialectReachesEveryDirectStoreOperation` with a malformed fixture rather than adding a fourth store-level test.
- Verified by revert (overlay, tree untouched): deleting `archivableRecord`'s `record.Park != nil` branch leaves `TestArchiveThreadRefusesALiveOrParkingThread` green.
- `ThreadBusy` still has no behavioural test in `couchtty` (BR-1).

## 6. Architectural notes

- **ARCH-DRY** — flag. One dialect now (good), but `occupiedIncarnation` has exactly one caller while `DecideResume` (`resume.go:76-92`) still inlines the same three-state loop, and `menuThreadActionable` / `PathHoldsUsableThread` are two hand-copies of `{live, detached, parked}`. See BR-6.
- **ARCH-PURE** — pass. Malformed rows are built in the pure projector from evidence the IO shell carries; `SelectResumableRoot`, `PathHoldsUsableThread`, `threadLabel`, `LabelsFor`, `archivableRecord` are all pure and tested without IO.
- **ARCH-PURPOSE** — flag. The Spec's `invalid → archive, inspectable` exit is reachable from no operator surface (BR-4); the plan's new Revisions entry and the commit message both assert it works.
- **ARCH-MOCK** — pass. `Quiesce` and the whole archive path run through injected stateful fakes; `runTypedRT` exercises production dispatch against them, which is how I probed this review without touching the tree.
- **ARCH-CONSTRAINTS** — pass. No new per-keystroke or per-start cost; rows stay sorted by `{scope, tag}`, so malformed rows do not jump the list.
- **ARCH-SECURE** — flag (I-1, and the fabricated result record in §4). The failure path now degrades visibly for a corrupt record's *row*, but invisibly for version skew and for the archive result.

## 7. Plan revision recommendations

- Amend the plan's "M3 review round 2" entry: `archive moves bytes it cannot decode` is true of `ThreadStore.ArchiveThread` and false of every surface that reaches it. State the store/dispatcher split, or mark the `invalid → archive` exit as still undelivered.
- Add to the issue: M2 had no milestone-close of its own; its code was reviewed inside the M3 window (`8e6f1af0..HEAD`), and its project `closed:`/`actual:` are derived from that, not measured at an M2 gate.
- Record the class named in I-1 — "unreadable" and "invalid" are different verdicts — as a scope note, since the Spec's reason table lists `invalid` with an archive exit and never contemplated a transient or version-skew read failure.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Enter now refuses busy via menuThreadActionable and says "it is busy", but nothing pins it and menuActionItems still offers archive to a busy row; unchanged this round.
  - id: BR-2
    disposition: addressed
    note: |
      issue:161 now reads 63 references across 15 files, and the plan records the measurement in its Revisions.
  - id: BR-3
    disposition: addressed
    note: |
      Verified by overlay revert: restoring ResolveThreadReference(a["ref"]) makes both new seam tests fail with "empty reference".
  - id: BR-4
    disposition: not-addressed
    note: |
      The row is visible now, but archive is still unreachable for it: through the real dispatcher archive{tag} fails with "couch: EOF" and archive{ref} with "not found" -- the new test calls couch.ArchiveThread directly, below the very seam whose absence caused BR-3.
  - id: BR-5
    disposition: not-addressed
    note: |
      Six of seven sites swept well; README's synopsis block (255-270) still omits --archived while usage() prints it, and the new Malformed/invalid-row rule has no atlas entry.
  - id: BR-6
    disposition: not-addressed
    note: |
      The reachable failure is fixed and well pinned, but the stated rule is not: occupiedIncarnation has one caller, DecideResume still inlines the same three states, and menuThreadActionable/PathHoldsUsableThread remain two copies of one set.
  - id: BR-7
    disposition: addressed
    note: |
      The refusal now names switcher gestures reachable from where it fires, and couch_test.go:1355 pins them.
  - id: BR-8
    disposition: not-addressed
    note: |
      Verified by revert: deleting archivableRecord's Park != nil branch leaves TestArchiveThreadRefusesALiveOrParkingThread green, because the fixture also carries a live incarnation. The startup recency half is genuinely fixed.
findings:
  - id: new
    severity: Critical
    family: unbacked-existing-behavior-claim
    title: |
      The project's new detail blocks break a contract test and record closes the gates never ran
    detail: |
      3rd in family -- state the rule, do not patch the block: a state or number in a portfolio artifact is
      written by the gate that produced it, and a judged value says so where it is read. go test
      ./cmd/internal/couchcore -run TestUncheckedProjectMilestoneHasNoClosedMetadata FAILS at HEAD and passed at
      the base: M3's block carries closed/actual while its row is unticked. M2's block records closed 2026-09-03
      and actual 0.9h though no milestone-close ever ran (no Review-Verdict trailer, no closed M2 log line, issue
      Plan still unticked, project row hand-ticked in 6572ef69). M1's actual 0.85h drops the "judgment estimate,
      not measured" qualification the issue Log carries. sdlc milestone-close owns the task row AND the detail
      block; hand-writing actuals ahead of it pollutes velocity calibration, which is what the gate exists to stop.
  - id: new
    severity: Important
    family: transient-failure-as-verdict
    title: |
      Snapshot reports "could not read" as the verdict "invalid", and the record leaves the usable set
    detail: |
      threadstore.go:517-537 folds an os.ReadFile error and a decode error into one Malformed list rendered as
      ReasonInvalid, whose documented exit is archive. DecodePersisted rejects an unknown schema_version and
      strictjson rejects unknown fields, so an older couch reading a newer store classifies every thread as
      debris -- and because the record also leaves snapshot.Records, PathHoldsUsableThread stops blocking, so
      couch <path> creates a fresh thread over live work. That is the ratchet M3 just closed, now silent where
      the old code failed loudly. Same shape as M1's own ProofStatus lesson: separate unreadable from invalid,
      and let an unreadable manifest-listed record still hold its path.
  - id: new
    severity: Important
    family: new-state-unhandled-at-consumers
    title: |
      Malformed rows are an opt-in variadic, so the pre-181 behaviour is the compile-clean default
    detail: |
      3rd in family -- do not fix the two call sites. The rule: the records and the evidence describing them
      travel as one value, so a consumer cannot opt out of part of the projection. ProjectActionableThreads
      (actionableinventory.go:174) and BuildThreadInventory (threadinventory.go:49) take malformed as a variadic;
      omitting it silently restores "some records get no row" with no compile error and no failing test. Pass
      ThreadSnapshot or a single input struct -- the call sites already moved once for the evidence parameter.
```

---

## Re-review — 2026-09-04T08:03:03-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 181 — One honest inventory: every thread gets a row and a reason |
| repo | pair |
| issue file | workshop/issues/000181-one-honest-inventory-every-thread-gets-a-row-and-a-reason.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | abc39d62baaeedb8057caa05d94eb113dc1d3b1b..5939d698a8336831a4e477b99540aa4fab76347d |
| command | sdlc milestone-close --issue 181 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-09-04T08:03:03-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

This round genuinely closed the three findings it set out to close, and I verified each by revert rather than by reading the commit message: `resolveThreadForArchive` (BR-4) — reverting it to `resolveOperationThread` turns the new runtime test red with `couch: EOF`; the `Park != nil` branch (BR-8) — deleting it now fails at `archive_test.go:106`; the project detail blocks (BR-9) — `TestUncheckedProjectMilestoneHasNoClosedMetadata` is red at `f9f6cdd6` and green at HEAD, and M2's block now records the missing gate instead of inventing an actual. What blocks SHIP is the new `PathHoldsUnreadableThread` behaviour. It is a real behaviour change — an unreadable record now refuses every start in its repository scope — and it shipped with no test at any seam, a refusal whose two named next steps both fail (`couch --show <tag>` → `thread reference not found`; `ctrl-space` → a TUI the refusal itself prevents opening in that repo), and three prose sites in the same tree still asserting the opposite rule, including the atlas paragraph four lines below the new one that says the reversal exists to avoid "one corrupt record locks its repo out permanently." In the version-skew case the change names as its own motivation — an older couch reading a newer store, where no record decodes — `couch` refuses to start in *every* repository with a message telling the operator to press a key inside a switcher they can never reach.

## 1. Strengths

- **`operationdispatch.go:327-345`** — `resolveThreadForArchive` is the right seam distinction, argued in the comment ("an archive target is addressed, not read") and revert-verified. `run_test.go:1505-1541` drives it through the real dispatcher, which is the level BR-4 was missing.
- **`threadreason.go:33-43` + `threadstore.go:517-541`** — the `unreadable`/`invalid` split is correctly motivated by version skew, not just corruption, and it is drawn at the layer where the read actually fails rather than downstream.
- **`archive_test.go:100-107`** — the added assertion is genuinely discriminating; it isolates the park branch from the live incarnation the fixture needed to build one. This is the pattern the `test-name-overclaims` family was asking for.
- **`workshop/projects/couch.md:265-275`** — M2's block records "no `milestone-close` was run for M2" instead of manufacturing a number for it. Recording a process deviation as a deviation is the honest answer, and it keeps `**actual:**` out of velocity calibration.
- **`actionableinventory.go:165-184`** — `ThreadProjectionInput` + `FromSnapshot` pairs the records with the addresses that could not become records; the production path can no longer split them.

## 2. Critical findings

**C-1 — the unreadable-record start refusal has no working next step, and no test.** `couch.go:350-359`

Measured against the real dispatcher (probe run against `newRT`/`seedThread` with a record overwritten as `{"schema_version":99,"nope":`):

| gesture | result |
|---|---|
| `couch /repo` | exit 1 — `couch cannot read thread couch-… in this repository` |
| `couch --show couch-…` (the refusal's step 1) | exit 1 — `thread reference not found` |
| `ctrl-space, select it, Tab → archive` (step 2) | unreachable: `run.go:288` renders the error and returns 1, so the TUI never opens in this repo |
| `couch --list` | works — the row is visible |

**This is the 2nd finding in family `unnavigable-refusal`.** BR-7 fixed the *other* refusal in this same function, and the comment it left at `couch.go:361-367` — "The next steps have to be ones that WORK from where the operator is… Both were dead ends printed at the moment someone was already stuck" — sits three lines below the new block that reintroduces them. Do not patch this message. State the rule and enforce it: **a refusal that names a command or gesture is pinned by a test that executes that gesture in the fixture that produced the refusal and asserts it succeeds.** Enumeration is cheap — grep `couchcore` for error strings containing two-space-indented next-step lines; there are three today (`couch.go:350`, `couch.go:368`, `startup.go:138`), and only `couch.go:368` has such a test (`couch_test.go:1355`).

Two things make this Critical rather than Important. First, the escape that does exist — start couch from a *different* repository, where the switcher lists all scopes — is nowhere stated. Second, in the version-skew scenario `threadreason.go:36-39` names as the whole reason for the split, *every* record is undecodable, so every scope carries an unreadable row and `couch` refuses to start anywhere; the recovery gesture is then unreachable by construction. `atlas/couch.md:565-566` states this hazard verbatim ("or one corrupt record locks its repo out permanently") as the reason the old rule existed, and the new paragraph at `:546-554` reverses it without answering it.

Third, and the reason the first two survived: `spawnResolved`'s refusal has **no test at any seam**. The only coverage is `TestAnUnreadableRecordBlocksStartsInItsRepository` (`archive_test.go:269-289`), which calls the pure predicate directly. A seam test would have had to name the next steps to assert on them.

## 3. Important findings

**I-1 — the unreadable set reaches the two projectors and no other snapshot consumer, so `--show` lies about existence.** `threadmetadata.go:28-34`

`Couch.ResolveThreadReference` passes `snapshot.Records` and drops `snapshot.Unreadable`, so every ref-resolving surface — `show`, `name`, `describe`, `park`, `resume`, and `archive` by `ref` — answers `thread reference not found` about a thread `couch --list` just printed. **This is the 2nd finding in family `decode-failure-drops-the-row`.** BR-4's rule was "a record the decoder rejects must still produce a visible row and must never remove other rows"; this is that rule one consumer over. Do not special-case `--show`. The tell is that archive was fixed by *adding a second resolver* (`resolveThreadForArchive`) that bypasses decoding rather than by making reference resolution total — a per-consumer patch where a shared rule belongs. State it as: **every consumer of `ThreadSnapshot` that answers "does this thread exist" sees the unreadable set.** Enumeration: 7 `\.Snapshot()` call sites in `couchcore` (`park.go:178`, `park.go:431`, `couch.go:63`, `couch.go:664`, `actionableinventory.go:386`, `threadmetadata.go:22`, plus the archive path); the four park/start-reconciliation loops legitimately filter on `record.Park`, `ResolveThreadReference` does not.

**I-2 — `ReasonInvalid` still renders to the operator as "unreadable record".** `threadreason.go:100-103`

The round split the two states and updated `unusableThreadNotice` (`menu.go:990-993`) but not `Label()`, so `couch --list` and the switcher column now print `unreadable record` for `invalid` and `could not be read — needs a look` for `unreadable`, side by side in one listing. **This is the 2nd finding in family `transient-failure-as-verdict`.** `TestEveryReasonHasADistinctOperatorLabel` (`threadreason_test.go:8`) passes because it compares exact strings; a label that borrows another state's defining word clears that bar. The rule: **when a state is split, every renderer of the old state is re-worded in the same commit, and the vocabulary guard checks meaning-collision, not string equality** — e.g. reject a label containing another reason's slug words. Renderers are enumerable: `Label()`, `unusableThreadNotice()`, `menu_render.go:286`, `atlas/couch.md`, `README.md`.

## 4. Minor findings

- `threadreason.go:41` claims `unreadable` is "never archive-eligible", but there is no archive-eligibility rule in the tree (`DecideRetirement` was not built), `menuActionItems` offers archive to it, `atlas/couch.md:551-553` says it "CAN be archived by the operator on purpose", and `ReasonUnknown` twenty lines below still claims to be "the only one that is never archive-eligible by construction". **4th in family `unbacked-existing-behavior-claim`** — see the finding block; the rule is that a behavioural claim in a comment names the code that enforces it or is deleted.
- `actionableinventory.go:165-172` / `atlas/couch.md:556-560` / `lessons.md` all claim one value makes "the next omission a compile error". `ThreadProjectionInput{Records: r, Evidence: e}` still omits `Unreadable` silently, and `BuildArchivedInventory` (`threadinventory.go:112`) does exactly that. The bundling is right; the claim overstates it. Same family.
- `threadstore.go:1049` synthesizes `ThreadRecord{Address: address}` for an undecodable archived record, without the `Reservation: true` marker the same round added at `detach.go:188` for the identical shape. Same class, one of two sites fixed (`ARCH-PURPOSE`).
- `couch --list` renders an unreadable row as bare tag plus a trailing space (`"couch-0102030405060708 \n"`) — no path, so the operator cannot tell which tree it belongs to, which is the one fact the refusal is about.

## 5. Test coverage notes

- Full suite at HEAD: `env -u PAIR_SESSION_ID -u PAIR_TAG go test ./... -count=1` — every failure is the documented sandbox `operation not permitted` pty class, plus `pairhelp_shim_test.go:51` and `agent_restart_test.go:84` (subprocess spawn, same class). No logic failures. `TestUncheckedProjectMilestoneHasNoClosedMetadata` passes.
- Revert-verified this round (tree restored after each): reverting `resolveThreadForArchive` → `resolveOperationThread` fails `TestAnUnreadableRecordCanBeArchivedThroughTheRuntime` with `couch: EOF`; deleting `archivableRecord`'s `record.Park != nil` branch fails `TestArchiveThreadRefusesALiveOrParkingThread` at line 106. Both fixes are real and pinned.
- The gap that ships C-1: **zero tests cross the seam for the new start refusal.** `TestAnUnreadableRecordBlocksStartsInItsRepository` is a pure-predicate test; nothing drives `StartInteractive`/`dispatchInteractiveStart` with an unreadable record. Add one there and assert on the next steps it prints, not just that it refuses.
- No test covers `couch --show` against an unreadable record (I-1), and none covers two repo scopes where one is blocked and the other must still start.
- `run_test.go:1509-1515` builds a `Couch` and immediately discards it (`_ = c`); either the construction is load-bearing and should say why, or it should go.
- `ThreadBusy` still has no behavioural test in `couchtty` (BR-1, unchanged).

## 6. Architectural notes

- **ARCH-DRY — flag.** `FromSnapshot`/`LabelsFor` are good consolidations, but `PathHoldsUnreadableThread` (`startup.go:91`) is now a *sixth* independent answer to "is this occupied / can this be acted on", alongside `menuThreadActionable`, `menuActionItems`, `ThreadStore.ArchiveThread`, `DecideResume` (`resume.go:80-95`, still inlining the same three-state loop) and `PathHoldsUsableThread`. `occupiedIncarnation` still has exactly one caller. See BR-6.
- **ARCH-PURE — pass.** `ProjectActionableThreads`, `BuildThreadInventory`, `PathHoldsUnreadableThread`, `ClassifyThread`, `archivableRecord` are all pure and tested without IO; the read failure is classified in the IO shell (`Snapshot`) and carried as data.
- **ARCH-PURPOSE — flag.** The shadow-sweep on the unreadable set: two projectors derive from it, `ResolveThreadReference` does not (I-1), and archive derives via a bypass rather than the shared value. The Spec's promise — "every thread gets a row and a reason" — is delivered for the listing surfaces and not for the resolving ones.
- **ARCH-MOCK — pass.** The new archive path runs through `runTypedRT` against the injected stateful fakes; that is how I probed `--show`, `--list` and interactive start without touching the real store.
- **ARCH-CONSTRAINTS — pass.** `PathHoldsUnreadableThread` is O(rows) on an inventory the caller already holds; no new per-keystroke or per-start cost, no extra zellij snapshot.
- **ARCH-SECURE — flag.** Good: `resolveThreadForArchive` builds a raw operator-supplied tag into a filesystem path, and `ArchiveThread`'s `validateThreadAddress` → `^[A-Za-z0-9_-]+$` makes traversal unrepresentable — worth keeping that dependency explicit in the comment, which currently credits the manifest instead. Bad: the version-skew input (a store written by a newer binary) now degrades to a hard refusal with no reachable recovery (C-1), and `--show` substitutes `not found` — a fabricated verdict about existence — for "could not read" (I-1).

## 7. Plan revision recommendations

- Amend the plan's **"2026-09-03 — M3 review round 2"** entry: it says "`Snapshot` now carries `Malformed` addresses, the projectors emit them as `invalid` rows." Both halves were reversed by this window — the field is `ThreadSnapshot.Unreadable` and the reason is `unreadable`. State the split and why (`invalid` is a verdict about a record that was read; `unreadable` is the absence of one).
- Add a `## Revisions` entry recording the startup rule reversal introduced here: **an unreadable record blocks starts scope-wide.** It contradicts the issue's own `2026-09-03` revision ("Debris does NOT block… or a path whose only rows are unusable could never be started in again"), `README.md:319-321`, and `atlas/couch.md:563-567`. The reversal may well be right, but it needs the same in-place recording M3 gave the `SelectUniqueResumableRoot` reversal, plus the escape it leaves the operator.
- Extend the M3 entity list with what actually landed this round and is in no table: `ReasonUnreadable`, `ThreadSnapshot.Unreadable`, `ThreadProjectionInput`, `FromSnapshot`, `PathHoldsUnreadableThread`, `resolveThreadForArchive`.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Unchanged this round -- menu.go's only edit was unusableThreadNotice; menuActionItems still offers archive to a ThreadBusy row and couchtty still has no ThreadBusy behavioural test.
  - id: BR-4
    disposition: addressed
    note: |
      Verified by revert: restoring resolveOperationThread in the archive branch fails TestAnUnreadableRecordCanBeArchivedThroughTheRuntime with "couch: EOF". Row is visible and removable through the real dispatcher.
  - id: BR-5
    disposition: not-addressed
    note: |
      The atlas gained the unreadable-record entry, but the same window reversed "debris does not block" and swept none of its restatements: README:319-321, atlas:563-567 (four lines below the new paragraph, and it names the exact hazard now realized), issue Revisions:244-246, and the plan's round-2 entry still says Snapshot carries "Malformed" emitted as "invalid" rows. README's synopsis block still omits --archived.
  - id: BR-6
    disposition: not-addressed
    note: |
      Unchanged, and now worse: PathHoldsUnreadableThread is a sixth independent occupancy/actionability predicate. occupiedIncarnation still has one caller; DecideResume (resume.go:80-95) still inlines the same three states.
  - id: BR-8
    disposition: addressed
    note: |
      Verified by revert: deleting archivableRecord's Park != nil branch now fails at archive_test.go:106. The startup recency half was fixed in the prior round.
  - id: BR-9
    disposition: addressed
    note: |
      TestUncheckedProjectMilestoneHasNoClosedMetadata is red at f9f6cdd6 and green at HEAD (measured both). M3's block carries no closed/actual and its row is unticked; M2 records the missing gate instead of inventing an actual; M1's "judgment estimate, not measured" qualification is restored. sdlc's upsertField inserts after **est:**, so the gate can still write them.
  - id: BR-10
    disposition: addressed
    note: |
      ReasonUnreadable is split from ReasonInvalid at the layer where the read fails, and the record still blocks its scope. Two residues raised separately: ReasonInvalid.Label() still says "unreadable record", and the block has no seam test and no working next step.
  - id: BR-11
    disposition: addressed
    note: |
      ProjectActionableThreads and BuildThreadInventory both take ThreadProjectionInput, and FromSnapshot keeps records and unreadable together on the production path. The "next omission is a compile error" claim is overstated -- a named-field literal still omits Unreadable -- raised as a Minor.
findings:
  - id: new
    severity: Critical
    family: unnavigable-refusal
    title: |
      The unreadable-record start refusal names two next steps, neither of which works, and has no seam test
    detail: |
      2nd in family -- do not patch the message. Measured against the real dispatcher with one record
      overwritten as `{"schema_version":99,"nope":`: `couch /repo` exits 1 (run.go:288 renders and returns,
      so the TUI never opens in that repo), `couch --show <tag>` exits 1 with "thread reference not found",
      and `ctrl-space, select it, Tab -> archive` is unreachable from the repository the refusal fires in.
      The working escape -- start couch from a different repository, where the switcher is global -- is
      stated nowhere; and in the version-skew case threadreason.go:36-39 names as the split's whole
      motivation, no record decodes, so every scope is blocked and there is no unblocked repo to start from.
      atlas/couch.md:565-566 states this hazard verbatim as the reason the old rule existed and the new
      paragraph at :546-554 reverses it without answering it. The reason all of this survived: couch.go:350's
      refusal has no test at any seam -- the only coverage is the pure predicate at archive_test.go:269.
      The rule: a refusal that names a command or gesture is pinned by a test that executes that gesture in
      the fixture that produced the refusal and asserts it succeeds. Enumerable today: couch.go:350,
      couch.go:368, startup.go:138 -- only couch.go:368 has one (couch_test.go:1355).
  - id: new
    severity: Important
    family: decode-failure-drops-the-row
    title: |
      ResolveThreadReference still reads snapshot.Records only, so --show reports "not found" for a row --list shows
    detail: |
      2nd in family -- do not special-case --show. threadmetadata.go:28-34 drops snapshot.Unreadable, so
      every ref-resolving surface (show, name, describe, park, resume, archive-by-ref) answers "thread
      reference not found" about a thread the inventory just printed. The tell that the rule was not stated:
      archive was fixed by ADDING a second resolver (resolveThreadForArchive) that bypasses decoding, rather
      than by making reference resolution total -- a per-consumer patch where a shared rule belongs. State
      it as: every consumer of ThreadSnapshot that answers "does this thread exist" sees the unreadable set.
      Enumeration: 7 `.Snapshot()` sites in couchcore; the four park/start-reconciliation loops legitimately
      filter on record.Park, ResolveThreadReference does not.
  - id: new
    severity: Important
    family: transient-failure-as-verdict
    title: |
      ReasonInvalid still renders to the operator as "unreadable record", the word the round just gave the other state
    detail: |
      2nd in family. threadreason.go:100-103 was not updated when menu.go:990-993 was, so one `couch --list`
      can print "unreadable record" for `invalid` and "could not be read - needs a look" for `unreadable`
      side by side. TestEveryReasonHasADistinctOperatorLabel passes because it compares exact strings, and a
      label that borrows another state's defining word clears that bar. The rule: when a state is split,
      every renderer of the old state is re-worded in the same commit, and the vocabulary guard checks
      meaning-collision rather than string equality. Renderers are enumerable: Label(), unusableThreadNotice,
      menu_render.go:286, atlas/couch.md, README.md.
  - id: new
    severity: Minor
    family: unbacked-existing-behavior-claim
    title: |
      Two behavioural claims added this window have no enforcing code, and one contradicts a comment 20 lines away
    detail: |
      4th in family -- state the rule, do not edit the two comments. (1) threadreason.go:41 says `unreadable`
      is "never archive-eligible": no archive-eligibility rule exists in the tree (DecideRetirement was not
      built), menuActionItems offers archive to it, atlas:551-553 says it CAN be archived on purpose, and
      ReasonUnknown at :59-60 still claims to be "the only one that is never archive-eligible by
      construction". (2) actionableinventory.go:165-172, atlas:556-560 and lessons.md all claim one value
      makes "the next omission a compile error" -- a named-field ThreadProjectionInput literal still omits
      Unreadable silently, and BuildArchivedInventory (threadinventory.go:112) does exactly that. The rule:
      a behavioural claim in a comment, atlas or lesson names the code that enforces it, and is deleted or
      demoted to intent when no such code exists. Enumerable by grepping the window's added prose for
      "never", "always", "cannot" and checking each against an enforcing site plus a test.
```

---

## Re-review — 2026-09-04T08:31:50-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 181 — One honest inventory: every thread gets a row and a reason |
| repo | pair |
| issue file | workshop/issues/000181-one-honest-inventory-every-thread-gets-a-row-and-a-reason.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | abc39d62baaeedb8057caa05d94eb113dc1d3b1b..501d7be4bd3aa2c2ca385fc846796cc9c805c121 |
| command | sdlc milestone-close --issue 181 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-09-04T08:31:50-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

This round did real, well-pinned work: `ReasonUnreadable` is now split from `ReasonInvalid` at the layer where the read fails, `ResolveThreadReference` sees the unreadable set (verified by revert — restoring `snapshot.Records` makes the new seam test fail with "thread reference not found"), the variadic `malformed` parameter is gone in favour of `ThreadProjectionInput`, `DecideResume` now shares `occupiedIncarnation` with archive, the docs sweep landed, and the project's calibration blocks stopped recording closes that never ran. What blocks SHIP is that the *headline behaviour this round added* — an unreadable record blocking starts in its repository, with a refusal naming working next steps — is entered by **zero tests**: I deleted the whole `PathHoldsUnreadableThread` block from `spawnResolved` and not one test outcome changed. I then wrote the 12-line seam test BR-12's own rule asks for and confirmed it goes red without the guard (`Spawn err=<nil>` — a second thread created over an unreadable record) and green with it. Separately, BR-6's occupancy rule got measurably worse rather than better: the switcher offers `archive` on an `unreadable` row, and `Couch.ArchiveThread` calls `Quiesce` **before** any guard, so taking that offer kills a live agent and files the record — measured with the stateful artifact fake. That is precisely the harm `threadreason.go`, `atlas/couch.md` and this round's new `lessons.md` entry all claim the split prevents.

### 1. Strengths

- `cmd/internal/couchcmd/run_test.go:1543-1583` — `TestARefusalsNamedCommandsActuallyWork` executes `couch --show` against the corrupt fixture through the real typed dispatcher. Verified discriminating: reverting `threadmetadata.go:36-40` fails it with the exact "not found" the finding described. Production and test share the boundary (`ARCH-MOCK` pass) — a real `ThreadStore` on a temp dir with a real truncated record, not a stubbed decoder.
- `cmd/internal/couchcore/threadreason.go:32-42` + `threadstore.go:39-47` — the `unreadable` / `invalid` split is placed at the layer where the read actually fails, and `ThreadSnapshot` carries the reason for the distinction in its own doc comment. `TestNoLabelBorrowsAnotherReasonsDefiningWord` (threadreason_test.go:17) is a genuinely better guard than string inequality.
- `cmd/internal/couchcore/actionableinventory.go:165-184` — `ThreadProjectionInput` + `FromSnapshot` collapse two parallel projector signatures into one value. `ARCH-DRY` pass on the shape.
- `cmd/internal/couchcore/archive_test.go:100-106` — the DISCRIMINATING assertion added to `TestArchiveThreadRefusesALiveOrParkingThread` calls `archivableRecord` with a park and *no* incarnation, so the park branch can no longer be deleted silently. This is the right pattern.
- `workshop/projects/couch.md:262-273` — M2's block records "no `milestone-close` was run" instead of inventing an actual. Refusing to write a number the gate didn't produce is exactly right.

### 2. Critical findings

**BR-12 (carried, still open) — the unreadable-record start guard is entered by no test.** `cmd/internal/couchcore/couch.go:354-370`.
The message is fixed (`--show` now works and is pinned; the switcher-from-another-repo escape and the record path are stated), but "has no seam test" is still literally true. Measured: replacing the whole block with `_ = PathHoldsUnreadableThread` changes zero test outcomes across `couchcore`/`couchcmd`/`couchtty`. Only three fixtures in the tree build a corrupt record (`run_test.go:1515`, `:1558`, `archive_test.go:186`) and none drives a start. Fix sketch — verified red-without / green-with in a scratch copy:

```go
func TestUnreadableRecordRefusesAStartAndSaysHow(t *testing.T) {
	env := newTestEnv(t, "/repo")
	first, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	os.WriteFile(env.Couch.Threads.recordPath(first.Thread), []byte(`{"schema_version":99,"nope":`), 0o600)
	_, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	if err == nil { t.Fatal("a start proceeded in a scope holding a record couch cannot read") }
	for _, want := range []string{string(first.Thread.Tag), "couch --show", "another repository", "Tab → archive", ".json"} {
		if !strings.Contains(err.Error(), want) { t.Fatalf("refusal %q does not mention %q", err, want) }
	}
}
```
Without the guard this reports `Spawn err=<nil>` — a second thread created over an unreadable record. Also unswept from BR-12's own enumeration: `startup.go:132`'s refusal names `couch --show <tag>` and no test executes it.

**BR-6 (carried, still open) — the occupancy guard is bypassed entirely for unreadable records, and `Quiesce` runs before it.** `cmd/internal/couchcore/detach.go:192-195`, `threadstore.go:990-996`, `menu.go:1006-1013`.
Chain, each link verified: `menuActionItems` offers `archive` to every non-actionable row, including `unreadable` and `ThreadBusy` → `threadEffect` dispatches `{repo-scope, tag}` → `resolveThreadForArchive` returns the address without decoding → `Couch.ArchiveThread` calls `Artifacts.Quiesce(address)` → *then* `ThreadStore.ArchiveThread` runs `archivableRecord`, which is skipped when `decodeErr != nil`. Measured with `FakeThreadArtifactCollisionChecker`:
- unreadable record + live Pair session → `err=<nil>`, `quiesced: [{816fc349d3faebf8 couch-…02}]`, record filed. The agent is killed and unlisted with no guard.
- park-in-flight row → `quiesced BEFORE the refusal`, then `err=thread … has a park in flight`. Session dead, record still listed.

This is the version-skew harm `threadreason.go:36-42`, `atlas/couch.md:549-551` and the new `lessons.md` entry all say is prevented.

### 3. Important findings

**BR-15 (carried, still open) — two behavioural claims still have no enforcing code, and were re-stated in two more artifacts this window.**
- "never archive-eligible" (`threadreason.go:41`): no archive-eligibility rule exists; `ReasonUnknown` at `:59-60` still claims to be "the only one that is never archive-eligible by construction" 20 lines below; `atlas/couch.md:549-551` says it "CAN be archived by the operator on purpose"; and `run_test.go:1500` now *asserts* an unreadable record archives successfully. Four artifacts, three positions.
- "the next omission is a compile error" (`actionableinventory.go:171-172`, and newly added to `atlas/couch.md:556-560` and `lessons.md`): still false. Shadow-sweep — 5 construction sites, 1 via `FromSnapshot`; `BuildArchivedInventory` (`threadinventory.go:112`) is production code building `ThreadProjectionInput{Records: records}` with `Unreadable` silently omitted. `ARCH-PURPOSE`: the single source is documentation, not enforcement.

**BR-1 (carried, still open) — busy-row behaviour is unpinned.** `menuThreadActionable` does exclude `ThreadBusy`, so the switch/resume half of the original finding is not what the code does. What remains: no `couchtty` test drives Enter on a `ThreadBusy` row, and `TestEveryReasonExplainsItselfOnEnter` iterates `AllThreadReasons()`, which never reaches the `case ""` → "it is busy" arm.

### 4. Minor findings

- `operationdispatch.go:334-345` vs `:348-360` — `resolveThreadForArchive` duplicates `resolveOperationThread`'s tag branch verbatim, differing only in the `GetThread` call. Now that `ResolveThreadReference` is total, the clean shape is one resolver whose tag branch checks the *manifest* rather than the decoder; `resolveThreadForArchive` then deletes. This is the residue BR-13 predicted ("a per-consumer patch where a shared rule belongs"). `ARCH-DRY`.
- Three address-only synthesis sites, two shapes: `detach.go:191` and `threadmetadata.go:40` set `Reservation: true`, `threadstore.go:1061` does not.
- `plan …-plan.md:1013-1017` (round-2 Revisions) still lists `ThreadSnapshot.Malformed` among "entities that landed"; the issue's Revisions still names `SelectUniqueResumableRoot`. Both are historical entries, so low-stakes — but the round-2 one is a claim about the current tree.
- `threadstore.go:101-110` — `RecordPath` exported solely to build one error string. Acceptable and documented; noting it as surface that now can't shrink.

### 5. Test coverage notes

Environment: `go build ./...` and `go vet` clean. All remaining failures in `couchcore`/`couchcmd`/`couchtty` are `ptychild: operation not permitted` / `fork/exec /bin/ps` — sandbox, not code, and identical in baseline and reverted runs. The docs contract tests (`TestREADMEDocumentsTheOperatorFacingSurface`, `TestAtlasDocumentsEveryTypedOperation`, `TestM3DocsMatchActionableSwitcherInventoryProvider`, and 7 more) all pass, so the `--archived` addition satisfies the README gate.

The gap is not volume, it's placement: `TestAnUnreadableRecordBlocksStartsInItsRepository` (archive_test.go:270) tests the pure predicate directly and never reaches its caller, which is why deleting the caller costs nothing. Note that `TestASecondThreadAtOnePathIsRefused` (couch_test.go:1346) — the analogous seam test for the *other* refusal — runs fine in this environment, so the missing test is cheap.

### 6. Architectural notes

- **ARCH-DRY** — flag: `resolveThreadForArchive` duplication; occupancy/actionability still split across `menuThreadActionable`, `menuActionItems`, `PathHoldsUsableThread`, `PathHoldsUnreadableThread` (BR-6). Pass on `occupiedIncarnation` and `ThreadProjectionInput`.
- **ARCH-PURE** — pass on the new pure entities. Flag: the refusal text is built inline in `spawnResolved`'s IO glue, which is *why* it went untested; a pure `unreadableStartRefusal(held, path, recordPath) error` would be directly assertable, though the seam test above is the more valuable fix.
- **ARCH-PURPOSE** — flag: the "one value, compile error" single-source claim has a hand-maintained consumer left in production code (BR-15); BR-12's own 3-site enumeration was 2/3 swept.
- **ARCH-MOCK** — pass. The new tests corrupt a real record in a real store and go through the real dispatcher; the stateful `FakeThreadArtifactCollisionChecker` is what let me measure the `Quiesce`-before-guard ordering at all.
- **ARCH-CONSTRAINTS** — pass. `PathHoldsUnreadableThread` is O(rows) over the already-resolved inventory, one call per start, no new IO or fan-out.
- **ARCH-SECURE** — pass on the parse boundary: a decode failure becomes a typed `Unreadable` address and degrades visibly. Flag: dropping the existence check in `resolveThreadForArchive` means `Quiesce` now fires on a caller-supplied address before anything validates it; and `Reservation: true` as a "synthesized, not real" marker overloads a flag `ClassifyThread:244` already reads as `never-started`, so any future consumer projecting the resolver's output relabels an unreadable record as a known state.

### 7. Plan revision recommendations

- **`## Revisions` — "M3 review round 3: the entities that landed"**: `ThreadProjectionInput`, `FromSnapshot`, `PathHoldsUnreadableThread`, `ReasonUnreadable`, `resolveThreadForArchive`, `ThreadStore.RecordPath`, `occupiedResumeCode` are all new and named in no entity table. State which are PURE.
- Same entry: correct the round-2 list's `ThreadSnapshot.Malformed` → `Unreadable`.
- **M1 Core concepts (plan:75-84)**: the vocabulary is enumerated as nine reasons without `unreadable`, and states `unknown` "is the only reason that is both transient and never archive-eligible" — the same unbacked claim as BR-15. Either delete the archive-eligibility clause or name the code that enforces it.
- **M3 Core concepts (plan:868-887)**: still tables `RetirementVerdict` / `DecideRetirement` in `retire.go` as "new" while Task 11 is marked NOT BUILT. The Revisions entry explains it; the table should carry a `deleted — see Revisions` status so a grep of the table doesn't claim a file that isn't there.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Enter's switch/resume half is not what the code does (menuThreadActionable excludes ThreadBusy), but no test drives Enter or the "it is busy" arm on a busy row; unchanged this round.
  - id: BR-5
    disposition: addressed
    note: |
      README gained --archived and the blocks-on-unreadable rule; atlas gained the unreadable, label and PathHoldsUsableThread entries; the rule is recorded in lessons.md. Residue is two historical Revisions entries.
  - id: BR-6
    disposition: not-addressed
    note: |
      DecideResume now shares occupiedIncarnation, but measured: archiving an unreadable-but-live row quiesces the agent and files the record with no guard, and a park-in-flight row is quiesced before the refusal.
  - id: BR-12
    disposition: not-addressed
    note: |
      Message and named gestures are fixed and pinned; the guard is not — deleting couch.go:354-370 changes zero test outcomes, and a 12-line seam test goes red without it (verified).
  - id: BR-13
    disposition: addressed
    note: |
      Verified by revert: restoring snapshot.Records fails TestARefusalsNamedCommandsActuallyWork with "thread reference not found". Residue: resolveThreadForArchive is now a near-duplicate that could fold back in.
  - id: BR-14
    disposition: addressed
    note: |
      ReasonInvalid renders "record failed validation", unusableThreadNotice reworded, and the new guard checks meaning-collision — though only over Label(), not unusableThreadNotice.
  - id: BR-15
    disposition: not-addressed
    note: |
      Both claims stand unenforced and were re-stated this window in atlas and lessons.md; "never archive-eligible" is now contradicted by a shipped test that archives an unreadable record.
findings:
  - id: new
    severity: Minor
    family: new-state-unhandled-at-consumers
    title: |
      The switcher offers name and describe on an unreadable row; both fail with the raw decoder error "couch: EOF"
    detail: |
      This is the 4th finding in family new-state-unhandled-at-consumers. Do not
      fix the instance — state the rule. Measured through the real dispatcher
      against a record overwritten as {"schema_version":99,"nope": — list and
      show render "unusable: could not be read — may need a newer couch", while
      name and describe both exit 1 with `couch: EOF`, and menuActionItems
      (menu.go:1010-1013) offers both on exactly that row. The rule: when a
      state is added, every consumer that OFFERS an action on a row in that
      state either supports the action or does not offer it, and any refusal it
      produces is couch's own worded message, not a raw decoder error.
      Enumeration: menuActionItems (offers archive/name/describe to every
      non-actionable row, including unreadable and busy), resolveOperationThread,
      ApplyThreadMetadata. Related trap in the same class: the synthesized
      ThreadRecord{Address, Reservation: true} at threadmetadata.go:40 overloads
      a flag ClassifyThread:244 already reads as never-started, so a future
      consumer projecting the resolver's output relabels an unreadable record as
      a known state — the exact conflation this round split apart.
```

---

## Re-review — 2026-09-04T08:57:26-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 181 — One honest inventory: every thread gets a row and a reason |
| repo | pair |
| issue file | workshop/issues/000181-one-honest-inventory-every-thread-gets-a-row-and-a-reason.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | abc39d62baaeedb8057caa05d94eb113dc1d3b1b..9c709f2d9a01a9db40268e20acccd895b436778e |
| command | sdlc milestone-close --issue 181 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-09-04T08:57:26-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

This round did the hard thing correctly: the two carried blockers are genuinely dead, and I verified each by mutation rather than by reading the commit message. Deleting the `PathHoldsUnreadableThread` block from `spawnResolved` now fails `TestSpawnRefusesWhileAnUnreadableRecordIsInTheRepository`; moving `Quiesce` back ahead of `archivableRecord` fails `TestARefusedArchiveStopsNothing`; quiescing on the unreadable branch fails `TestArchivingAnUnreadableRecordNeverStopsItsSession`; adding `ThreadBusy` to `menuThreadActionable` or disabling the new busy branch fails `TestEnterOnABusyRowExplainsAndOffersNoLifecycleAction`. Nothing Critical survives, and no logic test fails at HEAD (every failure is the documented `ptychild: operation not permitted` environment class). What keeps this off SHIP is that the round's own fix introduced a new consumer gap — `UnreadableArchiveWarning` is a *success* delivered on the failure channel, so the CLI exits 1 on an archive that worked and the switcher (the surface the start refusal names as the recovery path) shows a red error, stays in the archive confirmation frame, and then overwrites the "a session may still be running" warning with a routine notice — and that two carried findings were closed at a secondary site while the site they explicitly named still stands. BR-15 named `threadreason.go:41`; the commit fixed `:59` instead and the commit message claims all four artifacts now agree. They do not.

## 1. Strengths

- **`cmd/internal/couchcore/archive_test.go:349-368`** — `TestSpawnRefusesWhileAnUnreadableRecordIsInTheRepository` is the test BR-12 asked for, at the right seam. Verified: replacing the guard with `_ = PathHoldsUnreadableThread` fails it with "started a second thread while a record in this repository could not be read". The choice of `Spawn` as the funnel is right — one test covers `couch <path>`, the TUI start form and `SpawnPrepared`.
- **`cmd/internal/couchcore/detach.go:186-200`** — guard-before-effect is the correct shape, and `TestARefusedArchiveStopsNothing` asserts the property worth asserting ("it refuses *and nothing happened*"), not just the refusal. The `lessons.md` entry generalises it correctly.
- **`cmd/internal/couchcore/detach.go:201-211`** — declining to `Quiesce` a record couch could not read is the right call and is argued from the actual dependency (the guard proving a thread is dead needs the record that failed to decode). The behaviour is pinned by a discriminating test.
- **Three address-only synthesis sites now agree** — `detach.go:203`, `threadmetadata.go:44`, `threadstore.go:1065` are all bare `ThreadRecord{Address: address}`. Dropping `Reservation` closes the `ClassifyThread:244` conflation, and this is the first round where the sweep is complete rather than 2-of-3.
- **`cmd/internal/couchtty/menu_test.go:1343-1367`** — both halves discriminate (verified separately), and writing it found the busy-row archive offer, which is how a test earns its keep.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — `UnreadableArchiveWarning` is a success on the failure channel, and every consumer reads it as "the archive failed."** `cmd/internal/couchcore/detach.go:211`

**This is the 5th finding in family `new-state-unhandled-at-consumers`.** Do not fix the instance — state the rule. Measured through the real dispatcher and the pure menu reducer:

| consumer | behaviour |
|---|---|
| CLI (`archive{tag}`) | **exit 1**, `couch: archived couch-…, but couch could not read its record…` — while `list` → `no threads` and `archived` → the row. A `&&` chain or script sees failure. |
| `console.go:1349` | `Success: err == nil` → `reduceOperationResult` (`menu.go:1276`) takes the failure branch: red `MenuNoticeError`, stays in the **archive confirmation frame** (`frames=3 kind=confirmation action="archive"`), skips `ProjectionPending`. |
| next refresh | the frame self-heals, but the notice becomes `thread compiler (scope/couch-one) is no longer actionable` — the "a session may still be running" warning, the entire reason the value exists, is the one thing the operator loses. |
| `menu.go:1063` | the confirmation the operator accepts reads `archive <label> — stops its session`, which *this commit* made false for exactly this state. |
| retry (the natural response to a red error) | `couch: thread not found: {RepoScope:816fc349d3faebf8 Tag:couch-0102030405060708}` — a raw struct dump. |

The rule: **an outcome that is not a failure does not travel on the failure channel.** Either carry it on the result (`ArchiveResult{Record, SessionLeftRunning bool}`) so every renderer can show it as a warning, or keep the error type and update every consumer in the same commit. Enumeration is three sites: `operationdispatch.go:180` → `runTypedOperation`'s exit code, `console.go:1349`'s `Success`, and `confirmationMenuItems` (`menu.go:1058-1063`). Aggravating: the two tests that touched this were *weakened* rather than extended — `run_test.go:1536` dropped its `code != 0` assertion and `run_test.go:1584` now passes on an empty error too, so the exit-code change is both undecided and unpinned.

**I-2 — BR-6's rule is still unstated in code, and the two remaining copies now actively disagree.** `cmd/internal/couchcore/startup.go:73`, `cmd/internal/couchtty/menu.go:968`

Disposed `not-addressed`. The measured harms are fixed and pinned (`occupiedIncarnation` is shared by `archivableRecord` and `DecideResume`; the busy archive offer is gone) — but "one predicate over the classified state" is not. `PathHoldsUsableThread` and `menuThreadActionable` remain two hand-copies of `{ThreadLive, ThreadDetached, ThreadParked}` in two packages, and this round made them diverge: `menuActionItems` now treats `ThreadBusy` as occupied while `PathHoldsUsableThread` does not. Measured — `PathHoldsUsableThread(rows{State: ThreadBusy, WorkingPath: "/repo"}, "s", "/repo")` → `false`, so a start in flight at a path does not block a second start at that path. That is the one-thread-per-path ratchet with a hole at the exact state the round just protected in the archive path. Fix: one exported predicate in `couchcore` (`func (s ActionableThreadSummary) Occupied() bool`), called by both, with a test asserting they cannot answer differently.

**I-3 — the plan's M3 Core-concepts tables name a file and a verb that do not exist.** `workshop/plans/…-plan.md:871-872`, `:906-908`

**5th in family `unbacked-existing-behavior-claim`.** Verified against the tree: `cmd/internal/couchcore/retire.go` does not exist, `DecideRetirement`/`RetirementVerdict` are in no file, `couch prune` is in no registry, and `ThreadStore.Archive` shipped as `ThreadStore.ArchiveThread`. Four rows, four claims, zero backing — while Task 11/13 twenty lines below say `NOT BUILT`. Round 4 recommended the table carry a `not built — see Revisions` status in §7 and it was not done, so repeating it as a recommendation has already failed once; it is a finding now. The class rule is BR-15's: a claim in an artifact names the code that backs it, or is demoted. A greppable entity table is the highest-value place to enforce it, because it is the thing a future agent greps.

## 4. Minor findings

- `threadreason.go:41` still says `unreadable` is "never archive-eligible" — the exact line BR-15 named — contradicted by `menuActionItems` (measured: `[archive name describe]`), by two shipped tests that assert an unreadable record archives, and by `atlas/couch.md:549-551`. The commit fixed the *secondary* site (`:59`) and the commit message claims "one position now, in all four artifacts."
- `startup.go:63-65` still justifies itself with "or a corrupted record would lock its repo out permanently" — the sentence the atlas swept and qualified four lines below the new predicate that now does exactly that.
- `workshop/projects/couch.md:210` — `**status:** M1-M3 closed` while the M3 row is `- [ ]` and this gate has not passed. Pre-dates the window (`6572ef69`), but the window rewrote every other `#181` block around it. `TestUncheckedProjectMilestoneHasNoClosedMetadata` checks `**closed:**`/`**actual:**` only, not `**status:**`.
- `operationdispatch.go:334-345` vs `:348-360` — `resolveThreadForArchive` still duplicates `resolveOperationThread`'s tag branch verbatim (round-4 Minor, unchanged). `ARCH-DRY`.
- README documents `Tab → archive` in detail (`:388-391`) but not that archiving an unreadable record leaves its session running. The atlas covers it; the README paragraph that says archive is "offered on every row couch is not hosting" is also now false for `ThreadBusy`.

## 5. Test coverage notes

- Full suite at HEAD, `env -u PAIR_SESSION_ID -u PAIR_TAG go test ./... -count=1`: every failure is `ptychild: operation not permitted` / `fork/exec` / `mkstemp` — the documented environment class, identical across baseline and reverted runs. No logic failures. `go build ./...` and `go vet` clean. All docs contract tests pass.
- Revert-verified this round, tree restored after each (`git status` clean at HEAD): the start guard, the guard/quiesce ordering, the unreadable no-quiesce rule, the busy `menuActionItems` branch and the busy Enter arm all go red when removed. Four for four — this is the first round of the five where every claimed fix is mutation-checked.
- **The gap that ships I-1:** no test at any seam asserts what the CLI exit code or the switcher does with `UnreadableArchiveWarning`, and the two tests that could have caught it were relaxed to accommodate it.
- BR-16 unmeasured by any test and confirmed by probe: `name{tag}`, `name{ref}`, `describe{tag}`, `describe{ref}` on an unreadable row all exit 1 with the raw `couch: EOF`, while `menuActionItems` offers both on exactly that row.
- BR-12's own enumeration is 2 of 3 swept: `startup.go:141`'s refusal names `couch --show <tag>`, and `warmresume_test.go:138` asserts only that the *string* contains it — nothing executes it against the fixture that produced the refusal. The command does work; the pin does not exist.

## 6. Architectural notes

- **ARCH-DRY — flag.** I-2 (two copies of the state set, now disagreeing) and the `resolveThreadForArchive` duplication. Pass on `occupiedIncarnation`, `ThreadProjectionInput`/`FromSnapshot`, and the now-uniform address-only synthesis.
- **ARCH-PURE — pass.** `PathHoldsUnreadableThread`, `archivableRecord`, `occupiedIncarnation`, `menuActionItems`, `SelectResumableRoot` are all pure and tested without IO; the read failure is classified in the IO shell and carried as data. The one glue-resident policy (`readErr != nil` → do not quiesce) is small and inherently tied to the read.
- **ARCH-PURPOSE — flag.** Two fixes landed at a site adjacent to the one the finding named: BR-15 corrected `ReasonUnknown` and left `ReasonUnreadable`, BR-12's enumeration is 2/3. This is the instance/class inversion in its sharpest form — the class was *stated in the finding* and the sweep still stopped one line short.
- **ARCH-MOCK — pass.** `FakeThreadArtifactCollisionChecker` with `Quiesces()` is what made the ordering bug measurable and is what its test asserts on; `newRT`/`runTypedRT` drive real dispatch against a real `ThreadStore` on a temp dir with a genuinely truncated record. Production and test share the boundary.
- **ARCH-CONSTRAINTS — pass.** No new per-keystroke or per-start cost; `PathHoldsUnreadableThread` is O(rows) over an inventory the caller already holds.
- **ARCH-SECURE — pass on the parse boundary.** A decode failure becomes a typed `Unreadable` address and degrades visibly; `validateThreadAddress`'s `^[A-Za-z0-9_-]+$` still makes traversal unrepresentable on the archive path; the `Reservation` overload is gone. Worth recording as a deliberate accepted risk: once an unreadable record is archived, `PathHoldsUnreadableThread` stops blocking, so `couch <path>` will start a new thread over an agent the warning just said might still be running. That is the operator's choice and the warning names it — but it is exactly the "second thread over live work" the atlas paragraph at `:546-554` argues the guard prevents, so the atlas should say the escape hands that risk back.

## 7. Plan revision recommendations

- **M3 Core concepts (`plan:869-908`)** — mark `RetirementVerdict`, `DecideRetirement` and `couch prune` `not built — see Revisions`, and rename `ThreadStore.Archive` → `ThreadStore.ArchiveThread`. I-3; a `deleted`/`not built` status column is what stops a grep of the table claiming a file that is not there.
- **New `## Revisions` entry — "M3 review round 4"** — record the entities this round added that are in no table: `UnreadableArchiveWarning` (and that archiving an unreadable record deliberately does not quiesce), plus round 3's still-unrecorded `ThreadProjectionInput`, `FromSnapshot`, `PathHoldsUnreadableThread`, `ReasonUnreadable`, `resolveThreadForArchive`, `ThreadStore.RecordPath`. State which are PURE. This was recommended last round and not done.
- **M1 Core concepts (`plan:75-84`)** — the vocabulary is still enumerated as nine reasons without `unreadable`, and still says `unknown` "is the only reason that is both transient and never archive-eligible" — the same clause the code just deleted from `threadreason.go:59`. Delete it here too.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Verified by revert on both halves: adding ThreadBusy to menuThreadActionable makes the test fail with a dispatched switch effect, and disabling the new menuActionItems branch fails it with "busy row offers archive".
  - id: BR-6
    disposition: not-addressed
    note: |
      The harms are fixed and pinned, but the rule is not: startup.go:73 and menu.go:968 remain two copies of the state set, and this round made them disagree -- PathHoldsUsableThread returns false for a ThreadBusy row, so a start in flight does not block a second start at that path.
  - id: BR-12
    disposition: addressed
    note: |
      Verified by revert: replacing the guard with `_ = PathHoldsUnreadableThread` fails TestSpawnRefusesWhileAnUnreadableRecordIsInTheRepository. Residue -- startup.go:141's refusal names `couch --show <tag>` and warmresume_test.go:138 asserts only that the string contains it; nothing executes it.
  - id: BR-15
    disposition: not-addressed
    note: |
      The site the finding NAMED (threadreason.go:41, "never archive-eligible") still stands, contradicted by menuActionItems and by two shipped tests; only the secondary site at :59 was corrected, while the commit message claims all four artifacts now agree.
  - id: BR-16
    disposition: not-addressed
    note: |
      Re-measured through the real dispatcher: name{tag}, name{ref}, describe{tag} and describe{ref} on an unreadable row all exit 1 with the raw "couch: EOF", and menuActionItems still offers both on that row. Unchanged this round.
findings:
  - id: new
    severity: Important
    family: new-state-unhandled-at-consumers
    title: |
      UnreadableArchiveWarning is a success delivered on the failure channel, and every consumer reads it as a failed archive
    detail: |
      5th in family -- do not fix the instance, state the rule. Measured end to end. CLI: archive{tag} on an
      unreadable record exits 1 with "couch: archived ..., but couch could not read its record ...", while
      list reports "no threads" and archived lists the row -- the mutation happened. Switcher (the gesture
      the start refusal names as the recovery path): console.go:1349 sets Success: err == nil, so
      reduceOperationResult (menu.go:1276) takes the failure branch -- red error notice, stays in the archive
      confirmation frame, skips ProjectionPending -- and one refresh later the notice is replaced by "thread
      ... is no longer actionable", so the "a session may still be running" warning, the whole reason the
      value exists, is the one thing the operator loses. The confirmation they accepted reads "archive <label>
      -- stops its session" (menu.go:1063), which this same commit made false for this state. Retrying, the
      natural response to a red error, yields the raw "thread not found: {RepoScope:... Tag:...}". The rule:
      an outcome that is not a failure does not travel on the failure channel -- carry it on the result
      (ArchiveResult{Record, SessionLeftRunning}) so every renderer can show a warning, or update every
      consumer in the same commit. Enumeration is three sites: operationdispatch.go:180 -> exit code,
      console.go:1349's Success, confirmationMenuItems (menu.go:1058-1063). Aggravating: run_test.go:1536
      dropped its `code != 0` assertion and :1584 now passes on an empty error, so the exit-code change is
      both undecided and unpinned.
  - id: new
    severity: Important
    family: unbacked-existing-behavior-claim
    title: |
      The plan's M3 entity tables name a file and a verb that do not exist in the tree
    detail: |
      5th in family. Verified against the tree, not the prose: cmd/internal/couchcore/retire.go does not
      exist, DecideRetirement and RetirementVerdict are in no file, "couch prune" is in no registry, and
      ThreadStore.Archive shipped as ThreadStore.ArchiveThread -- four rows in the M3 Pure-entities and
      Integration-points tables (plan:871-872, :906-908), four claims, zero backing, while Task 11 and Task 13
      twenty lines below say NOT BUILT. Round 4 raised this as a section-7 plan-revision recommendation and it
      was not actioned, so recommending it again has already failed once. The class rule is BR-15's own: a
      claim in an artifact names the code that backs it or is demoted to intent -- and a greppable entity
      table is the highest-value place to enforce it, because it is what a future agent greps instead of the
      Revisions section. Fix: a `not built -- see Revisions` status on those rows, and the same sweep over the
      M1 table's nine-reason list, which still omits `unreadable` and still carries the archive-eligibility
      clause the code deleted from threadreason.go:59 this round.
```
