# Boundary Review — pair#457 (whole-issue close)

| field | value |
|-------|-------|
| issue | 457 — broadcast-web: recording format and component seams (contracts, fixtures, reference reducer) |
| repo | pair |
| issue file | workshop/issues/000457-broadcast-web-recording-format-and-component-seams-contracts-fixtures-reference-reducer.md |
| boundary | whole-issue close |
| milestone | — |
| window | f7b070ab27e3f3c64ac57631effcc1897475d2a3..0c9fc6e3e5817838c51e0fd45d023032f092d308 |
| command | sdlc close --issue 457 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T00:18:01-07:00 |
| verdict | REWORK |

## Review

Still reviewing the code. Next I'm checking whether the catch-up stream gives `notes.Live` the right notes after a resume.

Confirmed one bug. Next: the directory reader/writer, assets, and query.

```verdict
verdict: REWORK
confidence: high
```

The format, the reference reducer, the validator and the five component interfaces with their fakes are well built. The webcast suite passes with `-race` (`recording`, `assets`, `notes`, `query`, `fixturegen`). `artifactpath` fails in this checkout because the generated runtime-bundle assets are missing; that has nothing to do with this diff. One correctness bug blocks the close. `CatchUp` carries the current document's notes across a resume, but it attaches them to the wrong segment. So when a viewer joins after a resume, the notes model's `Live` rule (`n.Seg >= st.DocSeg`) drops the very notes the catch-up was meant to carry. The atlas and the plan's Revisions both promise the opposite. The equivalence test misses it because it compares raw note IDs, not the `Live` rule. The fix is small, but it is a stated-contract bug in a seam that #460 and #461 both consume. The plan's Core-concepts table also points at files that don't exist. Two Important findings are cheap to fix: sheet ids are not recorded in the id namespace, and the Sink seam can't carry asset bodies or frames.

**1. Strengths**
- `State.Apply` is a single, atomic, strict transition function: every reference is checked before any change (`recording/reduce.go:132-152`). Dangling references are errors, and the validator reports each one at its file and line. This is the explicit `(state, event)` model that ARCH-ORDER asks for.
- All the per-kind behavior derives from the one `kinds` table: Decode, `FileOf`, `Known` and the validator's warnings (ARCH-DRY).
- Every seam has a contract suite (`RunLiveContract`, `RunStoreContract`, `RunModelContract`, `RunSurfaceContract`) that takes `recording.TB`. The real implementations in wave 1 will run the same suites as the fakes (ARCH-MOCK).
- The fixtures are well disciplined. They are held by canonical pins, by a catch-up check at every pin (`fixture_test.go:43-58`), by `TestFixtureCoverage` and by a size budget, so a regenerated fixture can't silently lose a case.
- Path-traversal guards are in place: `Recording.Asset` validates the hash, and `PutFrame` and `ValidateDir` check frame names against the regex before touching the filesystem (ARCH-SECURE).

**2. Critical findings**
- **`CatchUp` attaches carried notes to segment 0 (`recording/catchup.go:14-63`).** I confirmed this with a probe on `twoDocs`, cut just before the navigation:
  - Full recording: `DocSeg=1`; notes a1 and old are in seg 1, a2 is in seg 2.
  - Catch-up: `DocSeg=2`; a1 and old are in seg 0, so `live=false`. Only a2 counts as live.
  
  The notes come before any `seg` in the catch-up, so they get `Seg=0`. Then the resume `seg` finds `DocSeg==0` and sets `DocSeg` to the resume's own number. As a result, `notes.FakeModel.Live` (and #461's real model) gives a different answer for a late joiner than for the full recording.
  - **Fix:** when `doc < last`, also emit `recs[doc]` (the `seg` that began the document). The reducer accepts a `seg` with no top doc, and the resume `seg` that follows resets it. The carried notes then reduce with their real `Seg`, and `DocSeg` matches.
  - **Regression test:** in `TestCatchUpEquivalence`, apply the `n.Seg >= DocSeg` filter on the catch-up side too, and assert `cu.DocSeg == all.DocSeg`.
- **The plan's Core-concepts table contradicts the code.**
  - It places `Live`, `FakeHub` and `RunLiveContract` in `recording/live.go`. That file doesn't exist: `Live` and `FakeHub` are in `fake.go`, `RunLiveContract` is in `contract.go`, and `Sink`/`Tee` are in `dir.go` (Task 5 lists a `sink.go`).
  - Tasks 3 and 4 list `canonical_test.go`, `files_test.go` and `dir_test.go`. Those tests actually live in `reduce_test.go` and `validate_test.go`.
  - No code change is needed. Add a Revisions entry (see section 7).

**3. Important findings**
- **Sheet ids are not recorded in the id namespace (`recording/reduce.go:303`).** `claim(r.ID, "sheet")` only checks; it never writes `s.types[r.ID]`. So an element can later take a sheet's id and the reducer accepts it. I confirmed this with a probe: `Sheet{ID:9999}` followed by `Ins{span id 9999}` gives `err=nil`. This breaks the atlas rule "one namespace… ids never reused".
  - **Fix:** set `s.types[r.ID] = "sheet"` after the claim, and add a case to `TestAnIDNeverChangesWhatItNames`.
- **The recorder's output seam can't carry asset bodies or frames (`recording/dir.go:16-40`, `fake.go`).**
  - `Sink` has only `Put` and `Close`. Asset bodies go through `DirWriter.PutAsset` and screenshots through `DirWriter.PutFrame`, but neither can go through a `Tee`.
  - `assets.Store` stores bodies as well, and only it resamples. #458 and #459 therefore have to work out between them who writes `assets/<hash>`, which goes against "start without asking".
  - `FakeHub` forwards `frame` records to viewers, although the atlas says frames are "not in the live stream" (ARCH-PURPOSE / API design).
  - **Fix:** name the single writer of `assets/<hash>`, for example "the Store writes, `DirWriter` only records". Make `FakeHub.Put` drop or refuse `frame` records, and cover that in `RunLiveContract`.

**4. Minor findings**
- The atlas's #460 row still says "the fixtures' canonical goldens"; since the Revisions they are pins (`canonical.sha256`).
- The atlas says the reducer "seeks by replaying from the latest `seg` before t", but `Reduce` replays from record 0.
- `forget` scans every sheet for every forgotten node, which is O(N·S) on large `kids` and `rm` operations (`reduce.go:584`).
- `Kids` calls `checkNew(…, replacing=p)`. That lets an id still live in `p`'s shadow root or template content be reused, while only `p.Kids` is forgotten.
- The tree walks in `checkNew` and `dupIDs` are near duplicates (ARCH-DRY nit).
- `DirWriter.PutAsset` skips writing when the file already exists. A partial body left by a crash therefore persists; `ValidateDir` would catch it. Write to a temp file and rename.
- The concurrent part of `RunLiveContract` gets its interleaving from `time.Sleep`. The oracle accepts any cut, but there is no seam to reproduce a particular ordering (ARCH-ORDER).
- `FakeSurface.Notes` reduces the prefix once per note, which is O(notes × records). That's acceptable for a fake.

**5. Test coverage notes**
- The reducer rules, the validator findings, the merge tie order, the `DirWriter`/`ReadDir` round trip, the codec round trip of every kind, and the `cdpconv` normalizations each have a test.
- The gap that shipped the Critical bug: the catch-up tests check note *membership*, not the `DocSeg`/`Seg` attribution that `notes.Live` depends on. `Canonical()` doesn't include notes, so the fixture pins can't catch it either.
- `RunModelContract` checks `Live` only on a full recording, never on a catch-up.

**6. Architectural notes for upcoming work**
- **ARCH-DRY:** passes (one minor walk duplication).
- **ARCH-PURE:** passes. The reducer, validator, catch-up, codec and converter are pure; the IO is `DirWriter`, `ReadDir` and the fixturegen main.
- **ARCH-PURPOSE:** flagged. The Sink seam leaves asset-body and frame ownership open between #458 and #459.
- **ARCH-MOCK:** passes. There are stateful fakes and shared contract suites; no external binary is touched here.
- **ARCH-CONSTRAINTS:** passes with notes. FakeHub retention is bounded per segment and slow subscribers are dropped. The replay-from-zero seek and the O(N·S) `forget` are the hot spots to watch for #462.
- **ARCH-SECURE:** passes for this boundary. Hash and frame names are validated. Tree depth and size are unbounded; that is declared in the plan ("recordings are the operator's own") and should be revisited when recordings are uploaded or shared.
- **ARCH-ORDER:** flagged. The transition function is explicit and good. The CatchUp↔reducer `DocSeg` interaction is exactly the kind of cross-event invariant to state and test across event sequences.
- **ARCH-FUNERAL:** passes. Recordings are kept until the operator deletes them, with the size per minute stated in the atlas. FakeHub subscriptions end on cancel or Close.

**7. Plan revision recommendations**
- Add a `## Revisions` entry: "File layout: `Sink` and `Tee` live in `dir.go`; `Live`, `MemSink`, `FakeHub` and `FakeRecorder` in `fake.go`; `RunLiveContract` in `contract.go` (no `live.go` or `sink.go`). The canonical, merge and dir tests are in `reduce_test.go` and `validate_test.go`." Update the Integration-points row accordingly.
- After the fix, add a Revisions note that `CatchUp` emits the `seg` that began the document, so the carried notes keep their segment.

```findings
findings:
  - id: new
    severity: Critical
    family: catchup-state-equivalence
    title: |
      CatchUp carries notes across a resume with Seg=0 and DocSeg=resume seg, so notes.Live drops them for late joiners
    detail: |
      Probe on twoDocs cut before navigate: full DocSeg=1 (a1,old seg 1); catch-up DocSeg=2 (a1,old seg 0, not live). Fix: emit recs[doc] (the seg the doc began in) when doc<last; regression: TestCatchUpEquivalence should compare the DocSeg-filtered notes and assert cu.DocSeg==all.DocSeg.
  - id: new
    severity: Critical
    family: plan-table-drift
    title: |
      Core-concepts table and tasks name recording/live.go, sink.go, canonical_test.go, files_test.go, dir_test.go, which do not exist
    detail: |
      Live/FakeHub are in fake.go, RunLiveContract in contract.go, Sink/Tee in dir.go; the tests are in reduce_test.go and validate_test.go. Prose-only fix: add a Revisions entry and correct the table.
  - id: new
    severity: Important
    family: id-namespace-enforcement
    title: |
      Sheet ids are never recorded in State.types, so a later node may reuse a sheet id (reduce.go claim only checks)
    detail: |
      Confirmed: Sheet{ID:9999} then Ins span id 9999 reduces without error. Set s.types[r.ID]="sheet" after claim; add a test.
  - id: new
    severity: Important
    family: seam-ownership-ambiguity
    title: |
      Sink seam cannot carry asset bodies or frames through Tee; DirWriter.PutAsset and assets.Store both store bodies; FakeHub forwards frame records to viewers
    detail: |
      Name the single writer of assets/<hash> (Store vs DirWriter) and make Live/FakeHub drop or refuse frame records, covered by RunLiveContract.
  - id: new
    severity: Minor
    family: doc-drift
    title: |
      Atlas says "canonical goldens" (now pins) and "seeks from the latest seg" (Reduce replays from record 0)
  - id: new
    severity: Minor
    family: hot-path-complexity
    title: |
      forget scans all sheets per forgotten node: O(N*S) on large kids/rm
  - id: new
    severity: Minor
    family: id-namespace-enforcement
    title: |
      Kids checkNew with replacing=p allows reuse of ids live in p's shadow root or template content, which are not forgotten
  - id: new
    severity: Minor
    family: atomic-file-write
    title: |
      DirWriter.PutAsset skips existing files, so a crash-truncated body persists; write to a temp file and rename
  - id: new
    severity: Minor
    family: ordering-test-seam
    title: |
      RunLiveContract concurrency relies on time.Sleep interleavings with no seam to reproduce a specific ordering
```

---

## Re-review — 2026-10-12T00:48:03-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 457 — broadcast-web: recording format and component seams (contracts, fixtures, reference reducer) |
| repo | pair |
| issue file | workshop/issues/000457-broadcast-web-recording-format-and-component-seams-contracts-fixtures-reference-reducer.md |
| boundary | whole-issue close |
| milestone | — |
| window | f7b070ab27e3f3c64ac57631effcc1897475d2a3..3cbaceb7e0f6fb7e2661e427281cf4ae0e1d5449 |
| command | sdlc close --issue 457 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T00:48:03-07:00 |
| verdict | SHIP |

## Review

The fix commit is 8120bbb6. Let me look at it.

Prior fixes for BR-1, BR-3 and BR-7 hold: each test fails when its fix is reverted. Next I'm checking the BR-4 seam (Tee and frames) and the PutAsset revert.

BR-2, BR-5, BR-6 and BR-9 look addressed. I'm checking the atlas claim that catch-up is tested at every pinned fixture time, then I'll write up the verdict.

```verdict
verdict: SHIP
confidence: high
```

All nine findings from round 1 are dealt with. Seven are addressed, BR-8's fix is correct but has no test that fails without it, and one table row from BR-2 is still stale. I checked each behaviour fix in this checkout by reverting it and running the tests. Reverting the BR-1 catch-up fix fails `TestCatchUpEquivalence` at cut 8 (segment 2 vs 1). Reverting the sheet-id claim (BR-3) fails `TestSheetAndNodeIDsShareOneNamespace`. Changing kids back to `replacing=p` (BR-7) fails `TestKidsCannotReuseIDsOutsideTheReplacedChildren`. Removing the frame refusal in `FakeHub` (BR-4) fails `RunLiveContract`. After the reverts the tree is clean, and `go test ./cmd/internal/webcast/...` passes. I also ran a separate check of catch-up equivalence (`DocSeg`, live notes, asset count) at every cut of all four fixtures, and it passed. What's left is minor and does not block: the atlas claims more fixture coverage than the tests have, `PutFrame` still writes non-atomically, and the plan's Integration-points table still has stale rows.

1. **Strengths**
   - `catchup.go:35-40`: the bare `seg` record is a small, correct fix. It works because `apply(*Seg)` already keeps `DocSeg` across a resume (`reduce.go:173`). The test now checks `DocSeg` and the notes as `notes.Live` sees them, not just the raw state.
   - `contract.go:69-91`: the sequential join loop checks every join point exactly, which replaces relying on a timed concurrent run. The frame-refusal check is now part of the contract that every `Live` must pass.
   - `reduce.go:337-343` together with `checkNew(..., p.Kids...)`: nodes and sheets now share one id space, enforced at both entry points, with a test for each direction.
   - `sameRecords` replaces the comparison loop that had been written twice (ARCH-DRY).
   - The lessons entries state general rules, not the one bug that was fixed.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `atlas/webcast-format.md:189-191` says the catch-up gives the same `DocSeg` and current-document notes "at every pinned time of every fixture". But `fixture_test.go:43-56` only compares `Canonical()`, which covers the DOM and not notes or `DocSeg`. The behaviour does hold: my check across every fixture cut passed. The test just doesn't pin it.
   - `dir.go:143`: `PutFrame` still writes with a plain `os.WriteFile`. A crash can also leave `assets/.tmp-*` files behind; `ValidateDir` reports them but nothing removes them (ARCH-FUNERAL).
   - Plan line 59 still lists `live.go`, and line 62 gives the signature as `PutAsset(hash, body)`; the real one is `PutAsset(body)`.

5. **Test coverage**
   - Each behaviour fix has a test that fails when the fix is reverted, except BR-8: reverting the atomic write leaves the suite green.
   - Changing the fixture loop to compare a full reduce against the catch-up's `DocSeg`, live notes and assets would cover the real resume data in the `segments` fixture.

6. **Architecture notes**
   - ARCH-DRY, PURE, MOCK, CONSTRAINTS, SECURE and ORDER pass. The sequential join enumeration gives ARCH-ORDER a reproducible oracle.
   - ARCH-PURPOSE passes for a contracts issue. `AssetWriter` is named but nothing uses it yet: `assets.Store` and `MemStore` don't take one. #459 should accept an `AssetWriter` in its constructor so "one writer of `assets/<hash>`" is enforced by the code, not just the docs.
   - ARCH-FUNERAL: there is a small residue gap from orphaned temp files (see Minor).

7. **Plan revisions**
   - Correct the Integration-points table in place: the `Live` row should point to `fake.go`/`contract.go`, the Sink text should say `PutAsset(body)` and mention `AssetWriter`, and the Task 3/4/5 "Files:" lines should drop the files that don't exist. The Revisions entry should keep only the why.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      catchup.go:35-40 emits the document's seg bare; reverting it fails TestCatchUpEquivalence at cut 8 (DocSeg 2 vs 1).
  - id: BR-2
    disposition: addressed
    note: |
      Revisions entry (plan lines 186-196) lists the real files; table row line 59 and Task Files lines are still stale (see new minor).
  - id: BR-3
    disposition: addressed
    note: |
      reduce.go sets types[id]="sheet" and refuses live-node ids; reverting it fails TestSheetAndNodeIDsShareOneNamespace.
  - id: BR-4
    disposition: addressed
    note: |
      Sink is records only, AssetWriter/DirWriter named as the single body writer, FakeHub refuses frames; reverting it fails RunLiveContract via fake_test.go.
  - id: BR-5
    disposition: addressed
    note: |
      atlas now says pins and that Reduce replays from record 0 (webcast-format.md:210-212, seam table).
  - id: BR-6
    disposition: addressed
    note: |
      forget scans sheets only for style/link, the only owners the Sheet check (reduce.go ~330) allows.
  - id: BR-7
    disposition: addressed
    note: |
      checkNew(&r.C[i], p.Kids...); reverting to p fails TestKidsCannotReuseIDsOutsideTheReplacedChildren.
  - id: BR-8
    disposition: not-addressed
    note: |
      Temp file plus rename is correct, but reverting to os.WriteFile leaves the suite green; no test fails without the fix. PutFrame (dir.go:143) is the same class and still non-atomic.
  - id: BR-9
    disposition: addressed
    note: |
      contract.go:69-91 checks CatchUp(first k)+rest at every k with no timing dependence; the timed concurrent run stays only as a stress test.
findings:
  - id: new
    severity: Minor
    family: catchup-state-equivalence
    title: |
      Atlas claims DocSeg/notes catch-up equivalence at every fixture pin, but fixture_test.go compares only the canonical DOM
    detail: |
      2nd finding in this family. Rule: every catch-up equivalence check (synthetic cuts and fixture pins) compares every view consumers read (Canonical, Assets, DocSeg, live notes) through one shared helper. Prevalence: 1 of 2 check sites. A probe over every fixture cut passed, so this is a coverage and doc gap, not a bug.
  - id: new
    severity: Minor
    family: atomic-file-write
    title: |
      DirWriter.PutFrame still writes frames/<n>.png with a plain os.WriteFile; crash-orphaned assets/.tmp-* files have no remover
    detail: |
      2nd finding in this family. Rule: every DirWriter file body goes through one atomic writeFileAtomic helper, used by both PutAsset and PutFrame; ValidateDir already reports stray .tmp files, so decide whether NewDirWriter's callers or a sweep removes them.
  - id: new
    severity: Minor
    family: plan-table-drift
    title: |
      Plan Integration-points table still names live.go and PutAsset(hash, body); Task 3/4/5 Files lines name files that do not exist
    detail: |
      2nd finding in this family. Rule: when a revision moves or renames an entity, edit the Core-concepts and Task tables in place, and record only the why under Revisions, so the table stays greppable truth.
```
