# Boundary Review — pair#493 (whole-issue close)

| field | value |
|-------|-------|
| issue | 493 — broadcast-web: notes and strokes aren't recorded (live viewers see them; replay and queries don't) |
| repo | pair |
| issue file | workshop/issues/000493-broadcast-web-notes-and-strokes-aren-t-recorded-live-viewers-see-them-replay-and-queries-don-t.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0043d4415b2b73eefb3f6387b5fc5fce7c6784a0..b5d34bc0fb002b93fdce737eb29cdc5489c7848e |
| command | sdlc close --issue 493 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T14:54:27-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I'm running the new end-to-end test and the recording package tests, then repeating them with the old buffered writer swapped back in to check that they fail without the fix.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The root cause is real and the fix works. The recorder buffered each output file in 4 KiB chunks, so the few notes and strokes in a cast stayed in memory until the cast ended, while `events.jsonl` filled its buffer constantly. `DirWriter.Put` now writes each record to disk as it arrives, and `Close` was simplified to match. I checked the red/green claim myself: with `dir.go` from base 0043d4415 restored, `TestCastBoardNotesAndStrokesReachDiskWhileLive` fails with "note=false stroke=false (0 records)". With the head version it passes. I put the file back afterwards and the working tree is clean.

What keeps this from a plain SHIP is one claim about reading a live recording. The new comment says a reader "sees whole lines only", but nothing makes that true. Replay (`replay.go:129,173`) and the query CLI (`query/cli.go:138`) read the directory while the cast is still writing it, through `recording.ReadDir`. `ReadDir` rejects the whole recording if any line fails to decode (`dir.go` ReadDir: `if len(bad) > 0 { return nil, ... }`). A read that lands in the middle of a large write can see a half-written last line. That line fails to decode, so the replay or query errors out instead of showing slightly older data.

1. **Strengths**
   - `recording/dir.go` `Put` writes each line with a single `f.Write(append(line, '\n'))`, and the two parallel maps (`files`/`osf`) are now one. That removes a way for the two maps to drift apart.
   - The end-to-end test (`cast_board_test.go:106`) runs a real `StartCast` and posts through the real HTTP viewer routes. It then reads the recording with the same reader replay uses and runs the real query, without closing the recording first. That matches the Spec's "fails today" requirement, and I confirmed it fails without the fix.
   - `TestLiveStrokeReachesDiskWhileLive` (`live_record_test.go:597`) wires the real recorder (including its document check) and the board exactly as `recordingOutput` does. That rules out a second cause before the files are written, which the Log needed.
   - The Log separates "late" from "missing" using evidence from the operator's actual recording directory, and splits off paused-stroke acceptance as its own issue, #499.

2. **Critical**: none.

3. **Important**
   - **A half-written last line makes the whole live read fail.** The new comment in `recording/dir.go` `Put` says "a reader (or a crash) sees whole lines only". POSIX doesn't guarantee that a concurrent reader won't see part of a write, and `events.jsonl` records such as DOM snapshots can be large. A read during that window gets an incomplete last line, and `ReadDir` then refuses the whole recording, which breaks the mid-cast replay and query this issue exists to enable (ARCH-PURPOSE).
     - Fix: have `ReadDir` ignore a final segment that has no trailing `\n`, e.g. trim `data` to its last newline before `DecodeFile`. Add a unit test that writes a file with a torn tail and checks `ReadDir` still succeeds. Then correct the comment.
     - Every instance in this window: the `Put` comment ("whole lines only"), the `DirWriter` doc comment ("a crash would have lost every file's tail", which implies none can now), and the atlas line "appends each record to its file in one write".

4. **Minor**
   - **ARCH-DRY:** `TestCastBoardNotesAndStrokesReachDiskWhileLive` repeats about 35 lines of setup from `TestCastBoardRecordsNotes`: `emittingBrowser`, `StartCast` with the Hub `NewLive`, `httptest`, `ro.Out()`, the board writer, and the builder page. A `startBoardCast(t) (c, base, eb, p)` helper would serve both tests.
   - There is no unit test in the `recording` package for "a record is visible on disk before `Close`". The only regression guard is the cast-level e2e test. A three-line `DirWriter` test (Put, then `os.ReadFile`) would pin it where the code lives.
   - The second Done-when clause ("the operator's next live cast replays with notes…") is a manual check. Neither the issue nor the Log says yet that it was done.

5. **Test coverage**
   - The e2e test pins the bug in both states: before the fix (0 records while live) and after. I ran that check myself.
   - The live-Chrome test only runs when `WEBCAST_LIVE_CHROME` is set, so CI doesn't exercise it. That's acceptable, since the Log records a manual red/green run.
   - Nothing tests reading a recording while a write is in progress. That's the gap behind the Important finding.

6. **Architectural notes**
   - **ARCH-PURE:** pass. The change is confined to the IO writer, and `DecodeFile`/`Merge` stay pure. The torn-tail fix belongs in `ReadDir`'s IO step or as a pure "complete lines" helper; either is fine.
   - **ARCH-DRY:** flagged, see the duplicated test setup above.
   - **ARCH-PURPOSE:** flagged on the reader side. Records now reach disk as they arrive, but the readers this issue cares about still fail on a partial last line.

7. **Plan revisions**: none needed. The plan's checklist matches the code. Once the torn-tail fix lands, add a Plan item for it.

```findings
findings:
  - id: new
    severity: Important
    family: live-reader-torn-tail
    title: |
      ReadDir fails the whole recording on a half-written last line, but replay and query read live recordings mid-write
    detail: |
      dir.go Put comment claims "a reader (or a crash) sees whole lines only". POSIX does not make one write atomic to a concurrent reader, and large events.jsonl records can tear. ReadDir rejects the whole recording on any bad line, and replay.go:129/173 and query/cli.go:138 call it on live recordings. Fix: drop an unterminated final segment before DecodeFile, add a torn-tail ReadDir unit test, and correct all three claims: the Put comment, the DirWriter doc comment, and the atlas/webcast-format.md "in one write" line.
  - id: new
    severity: Minor
    family: test-setup-duplication
    title: |
      New cast-board e2e test copies about 35 lines of StartCast/board setup from TestCastBoardRecordsNotes (ARCH-DRY)
    detail: |
      cast_board_test.go:106-140 duplicates :32-70. Extract a startBoardCast(t) helper used by both tests.
  - id: new
    severity: Minor
    family: regression-test-at-owning-layer
    title: |
      No recording-package unit test pins that DirWriter makes a record visible on disk before Close
    detail: |
      Only the cast-level e2e test guards it. Add a short dir_test case: Put a note, then os.ReadFile notes.jsonl without Close.
  - id: new
    severity: Minor
    family: done-when-unverified
    title: |
      Done-when clause 2 (the operator's next live cast replays notes and the query lists them) is not yet recorded as checked
```

---

## Re-review — 2026-10-12T15:08:51-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 493 — broadcast-web: notes and strokes aren't recorded (live viewers see them; replay and queries don't) |
| repo | pair |
| issue file | workshop/issues/000493-broadcast-web-notes-and-strokes-aren-t-recorded-live-viewers-see-them-replay-and-queries-don-t.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0043d4415b2b73eefb3f6387b5fc5fce7c6784a0..4761073bbc46e2ee35a636f8e9fa134c5df9dd32 |
| command | sdlc close --issue 493 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T15:08:51-07:00 |
| verdict | SHIP |

## Review

The torn-tail fix goes red when reverted, which confirms it. Next I'm checking the BR-3 owning-layer test against the old buffered writer, then running the full webcast tests.

```verdict
verdict: SHIP
confidence: high
```

The fix holds up. `DirWriter.Put` (`cmd/internal/webcast/recording/dir.go:95-110`) now writes each record straight to its file with no buffer, so notes and strokes are on disk while a cast is still running. That was the cause the issue describes. The round-1 torn-tail hazard (BR-1) is closed at the one place every live reader goes through: `ReadDir` decodes complete lines only (`dir.go:208`, `completeLines` at `:222`). I checked every live caller (`replay.go:129` and `:173`, `query/cli.go:138`, `fidelity_check.go:53`). None decodes record files without going through `ReadDir`.

`ValidateDir` still sees the raw bytes, so it still reports a torn tail on a finished recording. I reverted each fix in this checkout and both regression tests failed. `go vet` and `go test ./cmd/internal/webcast/...` pass. The only thing left open is the operator's live check (BR-4). That is a manual step, not a code defect, so it doesn't block.

**1. Strengths**
- `completeLines` (`dir.go:222`) handles an empty file or one with no newline without extra code: `LastIndexByte` returns -1, so the result is `data[:0]`. It is also applied at the single shared reader rather than separately in replay and query.
- Keeping `Recording.Files` raw means `Validate` and `ValidateDir` still flag a torn tail. `dir_live_test.go:71-73` checks this, so both sides are covered: the live reader tolerates it and the validator reports it.
- Root cause was proven, not guessed. The issue log reverts to the buffered writer and shows both end-to-end tests going red. The opt-in live-Chrome test (`live_record_test.go:596`) also rules out a second drop point in the recorder's `OnDoc` check.
- The comments and atlas now say the same thing about what a concurrent reader can see (`dir.go:61-68`, `atlas/webcast-format.md:358-362`). The old "whole lines only" claim is gone.

**2. Critical findings:** none.

**3. Important findings:** none.

**4. Minor findings** (not filed. They are below the bar, and each would repeat a family already in play):
- `dir_live_test.go:35`: if the old buffered writer comes back, `rec.Records[len(rec.Records)-1]` hits an empty slice and the test panics instead of failing with a clear message. It still goes red (I confirmed this), so it remains a valid regression guard.
- `cast_board_test.go:132`: the cleanup that calls `eb.w.Close()` is registered before `open` sets `eb.w`. If `open` calls `t.Fatal` first, the cleanup dereferences nil. It only affects tests that are already failing.

**5. Test coverage notes**
- Reverting `completeLines` makes `TestReadDirDecodesCompleteLinesOnly` fail with `events.jsonl:4: unexpected end of JSON input`.
- Restoring the base `dir.go` makes `TestDirWriterRecordIsOnDiskBeforeClose` fail (it panics).
- At the cast level, `TestCastBoardNotesAndStrokesReachDiskWhileLive` exercises `ReadDir` and `query.Run … notes` without closing the recording. That matches both outputs named in the Spec.
- Done-when clause 1 asks for "replay shows them". It is covered through replay's reader (`ReadDir`), not through the HTTP replay route. That is acceptable, since both `replay.go` paths call `ReadDir` directly.

**6. Architectural notes**
- **ARCH-DRY: pass.** BR-2 is resolved by `startRecordedBoardCast` and `open` (`cast_board_test.go:39-81`). What the two tests still share is the three-line `c.Notes()` handshake, which isn't worth extracting.
- **ARCH-PURE: pass.** `completeLines` is a pure function, and `ReadDir` stays the thin layer that does the IO. The torn-tail test writes real files, which is appropriate for an integration test of `ReadDir`.
- **ARCH-PURPOSE: pass.** The purpose (notes and strokes visible to replay and query during a live cast) is delivered through the single source, `DirWriter`, and every reader goes through the one fixed `ReadDir`. I found no reader that kept its own copy of the decoding.
- For the future: `ValidateDir` on a recording that is still being written can now report a torn tail as an error. That is correct for its "ended recording" contract, but any later tool that validates live recordings will need the same complete-lines rule.

**7. Plan revision recommendations:** none. The Plan's four checked items match the code. The round-1 dispositions in the Log match what I verified.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      ReadDir decodes completeLines(data) at dir.go:208; Files stays raw so ValidateDir still flags the tail. TestReadDirDecodesCompleteLinesOnly fails with completeLines reverted (verified). Put, DirWriter doc and atlas claims corrected.
  - id: BR-2
    disposition: addressed
    note: |
      startRecordedBoardCast and open (cast_board_test.go:39-81) are shared by both cast-board tests.
  - id: BR-3
    disposition: addressed
    note: |
      dir_live_test.go TestDirWriterRecordIsOnDiskBeforeClose reads back before Close; goes red against base dir.go (verified, as a panic on the empty slice).
  - id: BR-4
    disposition: not-addressed
    note: |
      Done-when clause 2 is the operator's live-cast check, still marked pending in the Log; it needs a manual run, not code.
```
