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
