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
