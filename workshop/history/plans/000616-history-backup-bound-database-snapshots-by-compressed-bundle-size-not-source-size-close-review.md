# Boundary Review — pair#616 (whole-issue close)

| field | value |
|-------|-------|
| issue | 616 — history-backup: bound database snapshots by compressed bundle size, not source size |
| repo | pair |
| issue file | workshop/issues/000616-history-backup-bound-database-snapshots-by-compressed-bundle-size-not-source-size.md |
| boundary | whole-issue close |
| milestone | — |
| window | fb3e2e6debf0b009a94a358fa2b7f9d0bb6c1a77..c9aa472665128fecbdaaf2294a9cb07e47e62a77 |
| command | sdlc close --issue 616 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T11:41:13-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the #616 diff: I've read the full change. I'm now checking the storage-growth docs, then I'll run the package tests.

Tests pass. Next I'll undo the new skip branch and rerun the tests, to check the regression test actually catches its removal.

RED check confirmed and the checkout is restored. One last look at what `Defaults()` costs, since `rawBundleLimit` now calls it on every catalog record read.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Verdict: fix then ship.** The main change is correct. The raw-size limit now depends on the file's kind (8 GiB for database snapshots, 1 GiB for everything else) and one function supplies it to every check that writes or reads a bundle. Without that, a snapshot over 1 GiB could be written and then rejected on every catalog read, and that failure is now ruled out. The package tests pass. I removed the new skip branch in `continuous.go`, ran the tests, and `TestCiphertextOverflowIsTheSentinelAndASnapshotSkip` failed as it should (it errors with "compressed bundle exceeds the ciphertext bound" and `skipped=[]`). I then put the file back; the checkout is clean.

One thing needs fixing before ship. When one snapshot is too big even after compression, the continuous pass skips every snapshot still waiting in that request, not just the big one, and the skip note doesn't name the file. That fails the issue's "refused with a clear note" condition, and the other databases would stop being backed up.

**1. Strengths**
- **One limit function, used everywhere.** `rawBundleLimit` (`model.go:202`) is used by `Describe`, `planBundles`, `ValidateManifest`, `ValidateLabel` and both callers of `SelectedFiles`. I checked the whole tree: no other `MaxBundleBytes` check remains in `historyarchive`.
- **Reliable overflow detection.** The size-limited writer records the overflow itself in `bounded.exceeded` (`archive.go:246`), so the error is caught even if the encoder throws away the underlying write error.
- **Real tests, confirmed to fail without the fix.** `bounds_internal_test.go` runs the actual `Archiver.Create` and `ContinuousPass.Run` code, with a simple copy encoder and a lowered limit, and checks validation for both kinds of file.
- **Cheap check before copying.** The size check runs before the copy (`sqlite_snapshot.go:80`), so the pass never copies gigabytes it can't archive anyway.

**2. Critical:** none.

**3. Important**
- **Siblings dropped on overflow** (`continuous.go:177-183`, ARCH-PURPOSE).
  - `Archiver.create` stops at the first bundle that fails (`archive.go:85-88`).
  - The new branch then marks the whole request as handled (`fresh = fresh[n:]`), including files planned *after* the oversized bundle.
  - Files are sorted by month and then path, so the same oversized snapshot comes first on every pass. Every snapshot sorted after it is therefore never archived.
  - The note says only `codex-database: …; skipped`, with no file name.
  - Fix: for snapshot sources, make one archive request per file, or on this error retry the remaining files without the failing ones. Put the path in the note. Add a test with two snapshots where the oversized one sorts first.
- **Storage growth not documented** (ARCH-FUNERAL).
  - This change starts writing a roughly 115 MB bundle every time `thread_history_1.sqlite` changes, which in practice is daily, into a sink that never deletes anything. That is on the order of 40 GB a year.
  - The atlas update records the compression ratio but not this growth or what limits it.
  - Fix: add the per-pass cost to the history-backup storage section, either the atlas text near `atlas/history-backup.md:148` or `storage-retention-io.md`.

**4. Minor**
- `rawBundleLimit` re-reads and re-checks the built-in inventory JSON on every call, including once for every catalog record read. Parse it once (for example with `sync.OnceValue`). It also quietly falls back to 1 GiB if that parse fails.
- `planBundles` (`plan.go:103`) estimates label overhead with `RawBytes: MaxBundleBytes`. That happens to be right because 8 GiB has the same number of digits, but it should use `rawBundleLimit(r.Kind)`.
- The test helpers ignore errors from `os.MkdirAll` and `os.WriteFile`.
- The empty `### 2026-10-16` heading in the issue log.

**5. Test coverage**
- Kind-based validation for manifests and labels is covered.
- The overflow error and the skip are covered, and confirmed to fail without the fix.
- Not covered: a request holding several snapshots where one overflows (the bug above).
- The real check that a file over 1 GiB gets archived is the live pass recorded in the issue log, which I did not re-run. That's reasonable given the file sizes involved.

**6. Architecture**
- **ARCH-DRY: pass.** The limit comes from one function.
- **ARCH-PURE: pass.** `rawBundleLimit` has no side effects. The new state is a single field on the writer.
- **ARCH-PURPOSE: flag.** Sibling snapshots are lost on overflow (Important finding 1).
- **ARCH-MOCK: pass.** The copy encoder and copy snapshotter plug in behind the existing interfaces, and the live pass serves as the real-world check.
- **ARCH-CONSTRAINTS: pass.** Data is still streamed in fixed-size pieces. The 2 GiB compressed limit, 8 GiB raw limit and the size check before copying are all enforced. The live run took 633 s.
- **ARCH-SECURE: pass.** Manifests read back from the catalog are still re-validated, now against the kind-based limit.
- **ARCH-ORDER: pass.** The pass keeps no state between runs beyond the catalog index. The overflow flag lives inside one function call.
- **ARCH-FUNERAL: flag.** The archive store grows by more per pass than before (Important finding 2). Temporary encrypted output is still cleaned up through the existing `defer`.

**7. Plan revisions:** none. The plan items are all delivered. When finding 1 is fixed, the Log should say the overflow skip now applies per file.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: partial-batch-failure-drops-siblings
    title: |
      ErrCiphertextLimit skip in the continuous pass drops every other snapshot still pending in the same request
    detail: |
      Archiver.create returns at the first failing manifest (archive.go:85-88), and the new branch (continuous.go:177-183) then skips all n requested files with fresh = fresh[n:], so snapshots planned after the oversized bundle (sorted by month and path, so the same order every pass) are never archived, and the note does not name the file. Fix: archive snapshot roots one file per request, or retry the remaining files minus the failing manifest; put the path in the note; add a test with two snapshots where the oversized one sorts first.
  - id: new
    severity: Important
    kind: defect
    family: growth-cost-unstated
    title: |
      Daily archiving of thread_history_1.sqlite adds about 115 MB per pass to a never-deleting sink, and the atlas does not state the cost
    detail: |
      Before this change the 1.6 GB database was skipped. Now each content change writes a new ~115 MB bundle, roughly 40 GB a year, and bundles are kept until a future retention decision (atlas/history-backup.md:148). Per ARCH-FUNERAL, add the per-pass cost and what bounds it to the history-backup storage section or storage-retention-io.md.
  - id: new
    severity: Minor
    kind: defect
    family: repeated-static-parse
    title: |
      rawBundleLimit re-parses the built-in inventory on every call, including once per catalog record read
    detail: |
      Parse it once (sync.OnceValue). Also, planBundles at plan.go:103 should size the label-overhead estimate with rawBundleLimit(r.Kind) instead of MaxBundleBytes; it is only correct today because 8 GiB and 1 GiB have the same number of digits.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"4948d216-7bdc-4d38-adc4-a44786962dd1","repository":"github.com/xianxu/pair","issue_id":"000616","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/b1fc78674ba95240","reviewed_head":"c9aa472665128fecbdaaf2294a9cb07e47e62a77","slot":"pair:2","requested_at":"2026-10-16T18:38:55.777276Z"},"phase":"finished","pid":52851,"started_at":"2026-10-16T18:38:56.124703Z","finished_at":"2026-10-16T18:41:11.399301Z","session_id":"e39ea5da-edf2-4518-a85c-e5f3ebf75f82","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 4.015757125s
head: "c9aa472665128fecbdaaf2294a9cb07e47e62a77"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-16T11:45:06-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 616 — history-backup: bound database snapshots by compressed bundle size, not source size |
| repo | pair |
| issue file | workshop/issues/000616-history-backup-bound-database-snapshots-by-compressed-bundle-size-not-source-size.md |
| boundary | whole-issue close |
| milestone | — |
| window | fb3e2e6debf0b009a94a358fa2b7f9d0bb6c1a77..bedbf0ba13f30d25b364f1feafcc5173b1970af5 |
| command | sdlc close --issue 616 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T11:45:06-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking the remaining raw-bound sites in the restore and verify paths.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

This boundary does what the issue asks. One kind-based `rawBundleLimit` now sets the raw size bound everywhere it is checked: Describe, planBundles, ValidateManifest, ValidateLabel and the snapshot pre-check. Because the bound depends only on the kind, catalog reads apply the same limit that writes did. The ciphertext overflow now returns a named error, `ErrCiphertextLimit`, which the bounded writer records itself, so it is detected even if the codec doesn't pass the writer's error through. The continuous pass now sends each snapshot as its own request. I checked the BR-1 fix the hard way: with the `n = 1` line removed, `TestCiphertextOverflowIsTheSentinelAndASnapshotSkip` fails ("archived map[codex-database:0]"), and with it restored the full package passes (`go test ./` ok, 27 s). The atlas now states the daily storage cost. Nothing blocks. Two Minor items are left: BR-3 was not fixed, and the skip note still doesn't name the file.

**1. Strengths**
- `model.go:202`: making the bound depend on the kind rather than a request flag is the right call. ValidateManifest runs again on every catalog read, so a flag would have let a >1 GiB snapshot be written and then fail every later read. The test `bounds_internal_test.go:23` pins this for both manifests and labels.
- `archive.go:245-248`: the writer keeps its own `exceeded` flag, so the error is detected whatever the codec does with the write error.
- `continuous.go:171-175`: one request per snapshot fixes the whole class of "one failure drops the others in its batch", not just the case in the earlier finding. The test puts the oversized `big.sqlite` before `small.sqlite` in sort order, and it fails without the fix.
- `atlas/history-backup.md:219`: the docs now give the bound, the skip behaviour and the yearly storage cost (~40 GB a year), and say retention is still undecided.

**2. Critical findings:** none.

**3. Important findings:** none new.

**4. Minor findings**
- BR-3 (still open): `rawBundleLimit` calls `Defaults()`, which re-decodes the built-in inventory JSON on every call (`inventory.go:49`). `plan.go:103` still uses `MaxBundleBytes` in the label-overhead prototype.
- Leftover from BR-1: the skip note at `continuous.go:183` gives only `root.Kind`, not the file. Since `n == 1`, adding `fresh[0]` is a one-token fix.
- `maxCiphertext` is a package-level variable that tests overwrite. That is safe only while those tests never run in parallel.

**5. Test coverage notes**
- Both done-when cases are covered: a snapshot whose compressed bundle fits is archived, and an oversized one is skipped with a note. The kind-based bound is covered on manifests and labels, and the pre-check test was updated for the 8 GiB bound.
- The live pass (115 MB bundle, sha256 verified) is recorded only in the issue log. I could not re-run it from this checkout.

**6. Architecture**
- **ARCH-DRY:** pass. There is one limit function, and the `Limit == 0` default lives in one place.
- **ARCH-PURE:** pass. `rawBundleLimit` and the validators are pure, and the sentinel is tested through an in-package copy codec with a temp-dir sink.
- **ARCH-PURPOSE:** pass. Every place the raw bound is checked now uses the shared function, and the live pass archives `thread_history_1.sqlite`.
- **ARCH-MOCK:** pass. `copyCodec` and `copySnapshotter` plug in at the same interfaces production uses.
- **ARCH-CONSTRAINTS:** pass. Streaming is still bounded by the pipe and the bounded writer. The 8 GiB raw bound puts a ceiling on how much is copied for a snapshot.
- **ARCH-SECURE:** pass. Catalog records are still re-validated, now against the kind's own bound. Kind comes from the record, and unknown kinds fall back to the 1 GiB bound.
- **ARCH-ORDER:** pass. The change adds no state that lives between events. Each request's outcome only decides whether the loop moves on.
- **ARCH-FUNERAL:** pass. The growth from daily snapshot versions (~115 MB per pass) is now stated, and pruning is explicitly left as an open retention decision.

**7. Plan revision recommendations:** none. The plan's three checked items match the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      continuous.go:171-175 sends each snapshot as its own request (n = 1 when collected). Removing that line makes TestCiphertextOverflowIsTheSentinelAndASnapshotSkip fail, where big.sqlite sorts before small.sqlite. Leftover nit: the skip note names only the kind, not fresh[0].
  - id: BR-2
    disposition: addressed
    note: |
      atlas/history-backup.md:219 now states that a database changing daily is a new version every pass, about 115 MB per bundle or roughly 40 GB a year, that the sink never deletes, and that pruning is a pending retention decision.
  - id: BR-3
    disposition: not-addressed
    note: |
      rawBundleLimit still calls Defaults(), which re-decodes sourcesJSON on every call (inventory.go:49), and plan.go:103 still uses MaxBundleBytes for the label RawBytes prototype.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"bd40ca0d-d7bd-4b2a-ba77-5edd2e326588","repository":"github.com/xianxu/pair","issue_id":"000616","recipe":"milestone-review","round":2,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/9a4db2b4da2ca939","reviewed_head":"bedbf0ba13f30d25b364f1feafcc5173b1970af5","slot":"pair:2","requested_at":"2026-10-16T18:43:24.392582Z"},"phase":"finished","pid":8954,"started_at":"2026-10-16T18:43:24.602169Z","finished_at":"2026-10-16T18:45:03.751645Z","session_id":"c5c4ff31-e891-460c-9d7b-be732664f221","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.858262042s
head: "bedbf0ba13f30d25b364f1feafcc5173b1970af5"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
