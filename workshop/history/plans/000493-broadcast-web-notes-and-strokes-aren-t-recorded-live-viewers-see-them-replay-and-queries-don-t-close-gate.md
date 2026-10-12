---
gate: boundary-review
issue: 493
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T14:54:27-07:00"
      dispatched: "2026-10-12T14:52:59-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: ReadDir fails the whole recording on a half-written last line, but replay and query read live recordings mid-write
          detail: 'dir.go Put comment claims "a reader (or a crash) sees whole lines only". POSIX does not make one write atomic to a concurrent reader, and large events.jsonl records can tear. ReadDir rejects the whole recording on any bad line, and replay.go:129/173 and query/cli.go:138 call it on live recordings. Fix: drop an unterminated final segment before DecodeFile, add a torn-tail ReadDir unit test, and correct all three claims: the Put comment, the DirWriter doc comment, and the atlas/webcast-format.md "in one write" line.'
          family: live-reader-torn-tail
          round: 1
        - id: BR-2
          severity: Minor
          title: New cast-board e2e test copies about 35 lines of StartCast/board setup from TestCastBoardRecordsNotes (ARCH-DRY)
          detail: cast_board_test.go:106-140 duplicates :32-70. Extract a startBoardCast(t) helper used by both tests.
          family: test-setup-duplication
          round: 1
        - id: BR-3
          severity: Minor
          title: No recording-package unit test pins that DirWriter makes a record visible on disk before Close
          detail: 'Only the cast-level e2e test guards it. Add a short dir_test case: Put a note, then os.ReadFile notes.jsonl without Close.'
          family: regression-test-at-owning-layer
          round: 1
        - id: BR-4
          severity: Minor
          title: Done-when clause 2 (the operator's next live cast replays notes and the query lists them) is not yet recorded as checked
          family: done-when-unverified
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T15:08:51-07:00"
      dispatched: "2026-10-12T15:07:17-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: ReadDir decodes completeLines(data) at dir.go:208; Files stays raw so ValidateDir still flags the tail. TestReadDirDecodesCompleteLinesOnly fails with completeLines reverted (verified). Put, DirWriter doc and atlas claims corrected.
          round: 2
        - id: BR-2
          disposition: addressed
          note: startRecordedBoardCast and open (cast_board_test.go:39-81) are shared by both cast-board tests.
          round: 2
        - id: BR-3
          disposition: addressed
          note: dir_live_test.go TestDirWriterRecordIsOnDiskBeforeClose reads back before Close; goes red against base dir.go (verified, as a panic on the empty slice).
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: Done-when clause 2 is the operator's live-cast check, still marked pending in the Log; it needs a manual run, not code.
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: 4761073bbc46e2ee35a636f8e9fa134c5df9dd32
      blocked: false
---

# Gate ledger — pair#493 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T14:54:27-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `live-reader-torn-tail` ReadDir fails the whole recording on a half-written last line, but replay and query read live recordings mid-write
  dir.go Put comment claims "a reader (or a crash) sees whole lines only". POSIX does not make one write atomic to a concurrent reader, and large events.jsonl records can tear. ReadDir rejects the whole recording on any bad line, and replay.go:129/173 and query/cli.go:138 call it on live recordings. Fix: drop an unterminated final segment before DecodeFile, add a torn-tail ReadDir unit test, and correct all three claims: the Put comment, the DirWriter doc comment, and the atlas/webcast-format.md "in one write" line.
- **BR-2** [Minor] `test-setup-duplication` New cast-board e2e test copies about 35 lines of StartCast/board setup from TestCastBoardRecordsNotes (ARCH-DRY)
  cast_board_test.go:106-140 duplicates :32-70. Extract a startBoardCast(t) helper used by both tests.
- **BR-3** [Minor] `regression-test-at-owning-layer` No recording-package unit test pins that DirWriter makes a record visible on disk before Close
  Only the cast-level e2e test guards it. Add a short dir_test case: Put a note, then os.ReadFile notes.jsonl without Close.
- **BR-4** [Minor] `done-when-unverified` Done-when clause 2 (the operator's next live cast replays notes and the query lists them) is not yet recorded as checked

## Round 2 — 2026-10-12T15:08:51-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — ReadDir decodes completeLines(data) at dir.go:208; Files stays raw so ValidateDir still flags the tail. TestReadDirDecodesCompleteLinesOnly fails with completeLines reverted (verified). Put, DirWriter doc and atlas claims corrected.
- BR-2 — addressed — startRecordedBoardCast and open (cast_board_test.go:39-81) are shared by both cast-board tests.
- BR-3 — addressed — dir_live_test.go TestDirWriterRecordIsOnDiskBeforeClose reads back before Close; goes red against base dir.go (verified, as a panic on the empty slice).
- BR-4 — not-addressed — Done-when clause 2 is the operator's live-cast check, still marked pending in the Log; it needs a manual run, not code.

## Open findings

- **BR-4** [Minor] `done-when-unverified` Done-when clause 2 (the operator's next live cast replays notes and the query lists them) is not yet recorded as checked
