---
gate: boundary-review
issue: 457
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T00:18:01-07:00"
      dispatched: "2026-10-12T00:15:00-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Critical
          title: CatchUp carries notes across a resume with Seg=0 and DocSeg=resume seg, so notes.Live drops them for late joiners
          detail: 'Probe on twoDocs cut before navigate: full DocSeg=1 (a1,old seg 1); catch-up DocSeg=2 (a1,old seg 0, not live). Fix: emit recs[doc] (the seg the doc began in) when doc<last; regression: TestCatchUpEquivalence should compare the DocSeg-filtered notes and assert cu.DocSeg==all.DocSeg.'
          family: catchup-state-equivalence
          round: 1
        - id: BR-2
          severity: Critical
          title: Core-concepts table and tasks name recording/live.go, sink.go, canonical_test.go, files_test.go, dir_test.go, which do not exist
          detail: 'Live/FakeHub are in fake.go, RunLiveContract in contract.go, Sink/Tee in dir.go; the tests are in reduce_test.go and validate_test.go. Prose-only fix: add a Revisions entry and correct the table.'
          family: plan-table-drift
          round: 1
        - id: BR-3
          severity: Important
          title: Sheet ids are never recorded in State.types, so a later node may reuse a sheet id (reduce.go claim only checks)
          detail: 'Confirmed: Sheet{ID:9999} then Ins span id 9999 reduces without error. Set s.types[r.ID]="sheet" after claim; add a test.'
          family: id-namespace-enforcement
          round: 1
        - id: BR-4
          severity: Important
          title: Sink seam cannot carry asset bodies or frames through Tee; DirWriter.PutAsset and assets.Store both store bodies; FakeHub forwards frame records to viewers
          detail: Name the single writer of assets/<hash> (Store vs DirWriter) and make Live/FakeHub drop or refuse frame records, covered by RunLiveContract.
          family: seam-ownership-ambiguity
          round: 1
        - id: BR-5
          severity: Minor
          title: Atlas says "canonical goldens" (now pins) and "seeks from the latest seg" (Reduce replays from record 0)
          family: doc-drift
          round: 1
        - id: BR-6
          severity: Minor
          title: 'forget scans all sheets per forgotten node: O(N*S) on large kids/rm'
          family: hot-path-complexity
          round: 1
        - id: BR-7
          severity: Minor
          title: Kids checkNew with replacing=p allows reuse of ids live in p's shadow root or template content, which are not forgotten
          family: id-namespace-enforcement
          round: 1
        - id: BR-8
          severity: Minor
          title: DirWriter.PutAsset skips existing files, so a crash-truncated body persists; write to a temp file and rename
          family: atomic-file-write
          round: 1
        - id: BR-9
          severity: Minor
          title: RunLiveContract concurrency relies on time.Sleep interleavings with no seam to reproduce a specific ordering
          family: ordering-test-seam
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T00:48:03-07:00"
      dispatched: "2026-10-12T00:44:23-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: catchup.go:35-40 emits the document's seg bare; reverting it fails TestCatchUpEquivalence at cut 8 (DocSeg 2 vs 1).
          round: 2
        - id: BR-2
          disposition: addressed
          note: Revisions entry (plan lines 186-196) lists the real files; table row line 59 and Task Files lines are still stale (see new minor).
          round: 2
        - id: BR-3
          disposition: addressed
          note: reduce.go sets types[id]="sheet" and refuses live-node ids; reverting it fails TestSheetAndNodeIDsShareOneNamespace.
          round: 2
        - id: BR-4
          disposition: addressed
          note: Sink is records only, AssetWriter/DirWriter named as the single body writer, FakeHub refuses frames; reverting it fails RunLiveContract via fake_test.go.
          round: 2
        - id: BR-5
          disposition: addressed
          note: atlas now says pins and that Reduce replays from record 0 (webcast-format.md:210-212, seam table).
          round: 2
        - id: BR-6
          disposition: addressed
          note: forget scans sheets only for style/link, the only owners the Sheet check (reduce.go ~330) allows.
          round: 2
        - id: BR-7
          disposition: addressed
          note: checkNew(&r.C[i], p.Kids...); reverting to p fails TestKidsCannotReuseIDsOutsideTheReplacedChildren.
          round: 2
        - id: BR-8
          disposition: not-addressed
          note: Temp file plus rename is correct, but reverting to os.WriteFile leaves the suite green; no test fails without the fix. PutFrame (dir.go:143) is the same class and still non-atomic.
          round: 2
        - id: BR-9
          disposition: addressed
          note: contract.go:69-91 checks CatchUp(first k)+rest at every k with no timing dependence; the timed concurrent run stays only as a stress test.
          round: 2
      findings:
        - id: BR-10
          severity: Minor
          title: Atlas claims DocSeg/notes catch-up equivalence at every fixture pin, but fixture_test.go compares only the canonical DOM
          detail: '2nd finding in this family. Rule: every catch-up equivalence check (synthetic cuts and fixture pins) compares every view consumers read (Canonical, Assets, DocSeg, live notes) through one shared helper. Prevalence: 1 of 2 check sites. A probe over every fixture cut passed, so this is a coverage and doc gap, not a bug.'
          family: catchup-state-equivalence
          round: 2
        - id: BR-11
          severity: Minor
          title: DirWriter.PutFrame still writes frames/<n>.png with a plain os.WriteFile; crash-orphaned assets/.tmp-* files have no remover
          detail: '2nd finding in this family. Rule: every DirWriter file body goes through one atomic writeFileAtomic helper, used by both PutAsset and PutFrame; ValidateDir already reports stray .tmp files, so decide whether NewDirWriter''s callers or a sweep removes them.'
          family: atomic-file-write
          round: 2
        - id: BR-12
          severity: Minor
          title: Plan Integration-points table still names live.go and PutAsset(hash, body); Task 3/4/5 Files lines name files that do not exist
          detail: '2nd finding in this family. Rule: when a revision moves or renames an entity, edit the Core-concepts and Task tables in place, and record only the why under Revisions, so the table stays greppable truth.'
          family: plan-table-drift
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 3cbaceb7e0f6fb7e2661e427281cf4ae0e1d5449
      blocked: false
    - "n": 3
      timestamp: "2026-10-12T00:50:56-07:00"
      dispatched: "2026-10-12T00:50:03-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: addressed
          note: PutAsset writes through writeFileAtomic (temp file in the same directory, then rename), at dir.go:111-123.
          round: 3
        - id: BR-10
          disposition: addressed
          note: Both catch-up checks (catchup_test.go cuts, fixture_test.go pins) call CheckCatchUp, which compares Canonical, Assets by hash, DocSeg and live notes with state.
          round: 3
        - id: BR-11
          disposition: not-addressed
          note: PutFrame is now atomic, but ValidateDir lists only assets/, so a crash-orphaned frames/.tmp-* is never reported, contrary to the dir.go:139 comment.
          round: 3
        - id: BR-12
          disposition: addressed
          note: Core-concepts, Integration and Task 3/4/5/7 Files lines edited in place; every named entity verified at its stated path.
          round: 3
      recipe: fix-review
      isolation: fenced
      reviewed: ffae0334f3ae699132e0b400e529cc31dcaa2491
      blocked: false
---

# Gate ledger — pair#457 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T00:18:01-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `catchup-state-equivalence` CatchUp carries notes across a resume with Seg=0 and DocSeg=resume seg, so notes.Live drops them for late joiners
  Probe on twoDocs cut before navigate: full DocSeg=1 (a1,old seg 1); catch-up DocSeg=2 (a1,old seg 0, not live). Fix: emit recs[doc] (the seg the doc began in) when doc<last; regression: TestCatchUpEquivalence should compare the DocSeg-filtered notes and assert cu.DocSeg==all.DocSeg.
- **BR-2** [Critical] `plan-table-drift` Core-concepts table and tasks name recording/live.go, sink.go, canonical_test.go, files_test.go, dir_test.go, which do not exist
  Live/FakeHub are in fake.go, RunLiveContract in contract.go, Sink/Tee in dir.go; the tests are in reduce_test.go and validate_test.go. Prose-only fix: add a Revisions entry and correct the table.
- **BR-3** [Important] `id-namespace-enforcement` Sheet ids are never recorded in State.types, so a later node may reuse a sheet id (reduce.go claim only checks)
  Confirmed: Sheet{ID:9999} then Ins span id 9999 reduces without error. Set s.types[r.ID]="sheet" after claim; add a test.
- **BR-4** [Important] `seam-ownership-ambiguity` Sink seam cannot carry asset bodies or frames through Tee; DirWriter.PutAsset and assets.Store both store bodies; FakeHub forwards frame records to viewers
  Name the single writer of assets/<hash> (Store vs DirWriter) and make Live/FakeHub drop or refuse frame records, covered by RunLiveContract.
- **BR-5** [Minor] `doc-drift` Atlas says "canonical goldens" (now pins) and "seeks from the latest seg" (Reduce replays from record 0)
- **BR-6** [Minor] `hot-path-complexity` forget scans all sheets per forgotten node: O(N*S) on large kids/rm
- **BR-7** [Minor] `id-namespace-enforcement` Kids checkNew with replacing=p allows reuse of ids live in p's shadow root or template content, which are not forgotten
- **BR-8** [Minor] `atomic-file-write` DirWriter.PutAsset skips existing files, so a crash-truncated body persists; write to a temp file and rename
- **BR-9** [Minor] `ordering-test-seam` RunLiveContract concurrency relies on time.Sleep interleavings with no seam to reproduce a specific ordering

## Round 2 — 2026-10-12T00:48:03-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — catchup.go:35-40 emits the document's seg bare; reverting it fails TestCatchUpEquivalence at cut 8 (DocSeg 2 vs 1).
- BR-2 — addressed — Revisions entry (plan lines 186-196) lists the real files; table row line 59 and Task Files lines are still stale (see new minor).
- BR-3 — addressed — reduce.go sets types[id]="sheet" and refuses live-node ids; reverting it fails TestSheetAndNodeIDsShareOneNamespace.
- BR-4 — addressed — Sink is records only, AssetWriter/DirWriter named as the single body writer, FakeHub refuses frames; reverting it fails RunLiveContract via fake_test.go.
- BR-5 — addressed — atlas now says pins and that Reduce replays from record 0 (webcast-format.md:210-212, seam table).
- BR-6 — addressed — forget scans sheets only for style/link, the only owners the Sheet check (reduce.go ~330) allows.
- BR-7 — addressed — checkNew(&r.C[i], p.Kids...); reverting to p fails TestKidsCannotReuseIDsOutsideTheReplacedChildren.
- BR-8 — not-addressed — Temp file plus rename is correct, but reverting to os.WriteFile leaves the suite green; no test fails without the fix. PutFrame (dir.go:143) is the same class and still non-atomic.
- BR-9 — addressed — contract.go:69-91 checks CatchUp(first k)+rest at every k with no timing dependence; the timed concurrent run stays only as a stress test.

### Raised

- **BR-10** [Minor] `catchup-state-equivalence` Atlas claims DocSeg/notes catch-up equivalence at every fixture pin, but fixture_test.go compares only the canonical DOM
  2nd finding in this family. Rule: every catch-up equivalence check (synthetic cuts and fixture pins) compares every view consumers read (Canonical, Assets, DocSeg, live notes) through one shared helper. Prevalence: 1 of 2 check sites. A probe over every fixture cut passed, so this is a coverage and doc gap, not a bug.
- **BR-11** [Minor] `atomic-file-write` DirWriter.PutFrame still writes frames/<n>.png with a plain os.WriteFile; crash-orphaned assets/.tmp-* files have no remover
  2nd finding in this family. Rule: every DirWriter file body goes through one atomic writeFileAtomic helper, used by both PutAsset and PutFrame; ValidateDir already reports stray .tmp files, so decide whether NewDirWriter's callers or a sweep removes them.
- **BR-12** [Minor] `plan-table-drift` Plan Integration-points table still names live.go and PutAsset(hash, body); Task 3/4/5 Files lines name files that do not exist
  2nd finding in this family. Rule: when a revision moves or renames an entity, edit the Core-concepts and Task tables in place, and record only the why under Revisions, so the table stays greppable truth.

## Round 3 — 2026-10-12T00:50:56-07:00 (claude) — passed

### Disposed

- BR-8 — addressed — PutAsset writes through writeFileAtomic (temp file in the same directory, then rename), at dir.go:111-123.
- BR-10 — addressed — Both catch-up checks (catchup_test.go cuts, fixture_test.go pins) call CheckCatchUp, which compares Canonical, Assets by hash, DocSeg and live notes with state.
- BR-11 — not-addressed — PutFrame is now atomic, but ValidateDir lists only assets/, so a crash-orphaned frames/.tmp-* is never reported, contrary to the dir.go:139 comment.
- BR-12 — addressed — Core-concepts, Integration and Task 3/4/5/7 Files lines edited in place; every named entity verified at its stated path.

## Open findings

- **BR-11** [Minor] `atomic-file-write` DirWriter.PutFrame still writes frames/<n>.png with a plain os.WriteFile; crash-orphaned assets/.tmp-* files have no remover
