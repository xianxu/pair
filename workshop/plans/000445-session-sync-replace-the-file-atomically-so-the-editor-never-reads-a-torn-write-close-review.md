# Boundary Review — pair#445 (whole-issue close)

| field | value |
|-------|-------|
| issue | 445 — session-sync: replace the file atomically, so the editor never reads a torn write |
| repo | pair |
| issue file | workshop/issues/000445-session-sync-replace-the-file-atomically-so-the-editor-never-reads-a-torn-write.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8f440c876a91d17224e1df2d015197d721ff7cc1..f7dd5eb0d1c1678747e603407bf84a93b54af87a |
| command | sdlc close --issue 445 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T11:34:11-07:00 |
| verdict | SHIP |

## Review

Tests pass. Next I'm checking for other writers of the session-sync file and reverting the fix to confirm the regression test fails without it.

```verdict
verdict: SHIP
confidence: high
```

This is ready to ship. The change replaces the in-place `os.WriteFile` of the operator's session-sync file with a temp file, fsync and rename (`replaceShared` → `durablefile.WriteAtomicMode`), which is what the Spec asks for. The flock lives in a separate `.<base>.flock` file, so swapping the file's inode doesn't break serialization. I checked the regression test by putting `os.WriteFile` back in `sync.go:352`: `TestAReaderOutsideTheFlockNeverSeesATornWrite` failed ("a reader saw a partial file (0 bytes)"). With the fix restored, both packages pass. That was the only write site for this file: `couchcmd/session_sync.go` doesn't write it, and the out-of-date "parley watches this file's inode" comment is gone from the code. Atlas and lessons were updated. Nothing blocks shipping; the notes below are minor.

1. **Strengths**
   - `cmd/internal/sessionsync/ledger.go:212`: `replaceShared` resolves symlinks before the rename and keeps the file's mode. Those are the two usual ways an atomic replace goes wrong on a file the operator edits, and both are tested (`replace_test.go:59`).
   - `durablefile.WriteAtomic` became a thin wrapper over `WriteAtomicMode` (`write.go:10-16`), so the publish routine stays in one place and isn't copied (ARCH-DRY: pass).
   - The torn-read test has a guard against passing without proving anything: it fails if the reader polled fewer than 100 times (`replace_test.go:46`). I confirmed it fails when the fix is removed.
   - The atlas entry (`atlas/couch.md:435`) now matches the code: all three files a write touches go through temp+rename.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - A rename also breaks hard links and drops the file's owner (when couch runs as another user), extended attributes and ACLs. The comment and the lesson only mention symlinks and mode. Neither is realistic for the ops status file; adding one clause to the comment would be enough.
   - The second `Done when` clause ("checked live in the ops status file") was checked against a copy of the file using the operator's real nvim config. The check on the real file is deferred until after merge, as the Plan records. That's reasonable, but the close should cite that post-merge report.
   - The comment "Parley and nvim notice the change by the path's mtime, which a rename moves too" is loosely worded: the rename doesn't change any mtime; the new inode simply has a newer one. Wording only.

5. **Test coverage**
   - Removing the fix makes the torn-read test fail, as above.
   - The mode/symlink test guards against a naive rename, not against `os.WriteFile`, which is correct for what it pins.
   - Parley's reload and the `.clobbered` path have no automated test. They were checked live only, which is acceptable because they belong to the editor.

6. **Architecture**
   - **ARCH-DRY: pass.** It reuses `durablefile.publish` and adds a mode parameter instead of a parallel writer.
   - **ARCH-PURE: pass.** The change is pure I/O glue at the existing boundary and adds no business logic.
   - **ARCH-PURPOSE: pass.** I looked for any other place that rewrites the shared file in place and found only `sync.go:352`, which is now atomic. The shadow, the state sidecar and the gap and recording files were already atomic.

7. **Plan revisions:** none. The plan matches the code.

```findings
findings:
  - id: new
    severity: Minor
    family: atomic-replace-metadata-loss
    title: |
      replaceShared's rename also breaks hard links and drops owner/xattrs/ACLs; comment and lesson name only symlink and mode
    detail: |
      ledger.go:207-211 and workshop/lessons.md list symlink + mode as what a rename changes; hard links, ownership (when couch runs as another user) and xattrs are the rest of that list. Note them in the comment, or accept them explicitly.
  - id: new
    severity: Minor
    family: done-when-deferred-verification
    title: |
      Done-when live check on the real ops status file is deferred to post-merge
    detail: |
      It was verified against a copy with the real nvim config. The close should cite the post-merge ops:0 report that covers the real-file clause.
```
