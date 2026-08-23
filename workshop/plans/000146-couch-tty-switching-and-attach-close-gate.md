---
gate: boundary-review
issue: 146
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-22T18:31:03-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Screen raises a false bell for any sequence longer than maxPending split across two reads
          detail: |-
            screen.go:104 drops the ESC and rescans the payload as text when an incomplete
            sequence exceeds 256 bytes, so the sequence's terminating BEL is later counted by
            the plain-run scan at screen.go:90. Reproduced: a 300-byte OSC split across two
            Feed calls yields bell=true where the same bytes fed whole yield bell=false. The
            everyday trigger is OSC 52 clipboard writes, which are kilobytes and always cross
            a 4096-byte pty read boundary. Contradicts TakeBell's own doc and
            atlas/architecture.md:462. FuzzScreenFeed asserts chunk-invariance for AltScreen,
            Mouse and Pending but not for the two latches, which is why it cannot find this.
          family: chunking-invariance
          round: 1
        - id: BR-2
          severity: Important
          title: runShell now stops SIGWINCH delivery after closing every child pty, re-opening the scribecmd race
          detail: |-
            termcmd/run.go:220 registers the host.Close() defer first, so LIFO runs it LAST --
            after mux.closeAll(). The watcher goroutine at run.go:246 is therefore still live
            during closeAll, and a SIGWINCH there runs resizeAll -> child.Resize ->
            pty.Setsize -> ptmx.Fd() concurrently with ptmx.Close(). The pre-migration code
            registered defer signal.Stop(winch) last so it ran first. workshop/lessons.md
            prescribes exactly this ordering from the scribecmd bug. Fix: register the
            host.Close() defer immediately after defer mux.closeAll().
          family: signal-goroutine-outlives-close
          round: 1
        - id: BR-3
          severity: Important
          title: NewFakeChild's documented Wait/Done contract is false and no test pins either
          detail: |-
            fake.go:20 documents "Wait returns ExitCode (default 0) immediately; Done is true
            once Exit is called, or immediately if the child was never running". Measured:
            Done() is false at construction and Wait() blocks indefinitely, because done is
            created open at fake.go:26 and only Exit closes it. The fake and the real Child
            have opposite Wait semantics, so an M2/M3 test written from the doc hangs rather
            than fails. No conformance check exists between the fake and a real Child, which is
            the ARCH-MOCK gap that would have caught it -- hostty_test.go:118 is the template.
          family: fake-diverges-from-production
          round: 1
        - id: BR-4
          severity: Important
          title: the Ring copy-vs-reslice fix is claimed as a bug fix, but reverting it leaves the named test green
          detail: |-
            ring.go:48-52 asserts re-slicing is "unbounded growth hiding behind a bounded
            Snapshot", Allocated() is exported solely to pin it, and the issue Log lists "ring
            trim" as a deletion check confirmed red. Reverting to the re-slice leaves
            TestRingDoesNotGrowWithoutBound green. Measured over 2000 appends into a 32-byte
            ring: reslice peaks at cap=48, copy sits at cap=64 -- re-slicing monotonically
            shrinks remaining capacity so append is guaranteed to reallocate. The copy is a
            reasonable cleanup; the defect it claims to fix does not exist.
          family: fix-not-pinned-by-failing-test
          round: 1
        - id: BR-5
          severity: Important
          title: the plan's Core-concepts table says termcmd.restoreTerminal is deleted; it still exists
          detail: |-
            Plan row :89 and Task 1.4a :213 both state restoreTerminal is deleted and folded
            into hostty. The method survives at termcmd/run.go:1061 and writes
            hostty.ResetRegion. The behaviour the row is about -- one escape constant, one site
            -- is delivered and verified, so this is a table-accuracy defect rather than missing
            work, but it needs a "## Revisions" entry rather than a silent edit.
          family: plan-table-drift
          round: 1
        - id: BR-6
          severity: Minor
          title: waitCode is duplicated verbatim across couchcore and ptychild, and asExitError now exists in three packages
          detail: |-
            couchcore/runner.go:90 and ptychild/child.go:130 are byte-identical; ptychild's
            errors.go says it mirrors couchcore's. ARCH-DRY, in the diff whose stated thesis is
            that the second copy is the thing to avoid.
          family: needless-indirection
          round: 1
        - id: BR-7
          severity: Minor
          title: comments still cite queries.go, appendBuffer, tab.buffer and readPTY, all deleted by this diff
          detail: |-
            termcmd/run.go:1046 ("see queries.go"), run.go:1053 ("appendBuffer re-slices
            tab.buffer from the output goroutine"), ptychild/replay.go:37 ("returns through
            readPTY").
          family: stale-comment-reference
          round: 1
        - id: BR-8
          severity: Minor
          title: Child.SetSink writes c.sink unlocked with no fake-only guard while the pump reads it
          detail: |-
            fake.go:57. Documented as being for fakes, but it is a method on *Child with no
            `if c.fake == nil` guard, so calling it on a real child races the read pump. No
            caller today.
          family: unsynchronised-shared-state
          round: 1
        - id: BR-9
          severity: Minor
          title: newTab widens the window where a chunk is both replayed and written live
          detail: |-
            The ring is now filled inside the pump, before bufferSnapshotLocked(tab) at
            run.go:707, so a chunk landing in that gap is replayed by redrawTab and then
            written again by copyActiveOutput. A brand-new tab has nothing to replay; passing
            nil is the honest fix. The same shape recurs in M2 when the console attaches to a
            freshly started child.
          family: replay-duplicates-live-output
          round: 1
        - id: BR-10
          severity: Minor
          title: make test-race still targets ./cmd/pair-wrap/, so Task 1.6's -race requirement has no runnable target
          detail: |-
            make test-live landed this window; the test-race half of the same rule did not, and
            the target fails outright with "directory not found". Plan Task 1.6 requires
            go test ./cmd/... -race over the whole tree.
          family: stale-build-target
          round: 1
        - id: BR-11
          severity: Minor
          title: OSHost's SIGWINCH coalescing has no test; only FakeHost's does
          detail: |-
            TestFakeHostResizesCoalesce drives the fake. OSHost.watch()'s non-blocking send --
            the actual coalescing production depends on -- has no coverage, in the package whose
            stated purpose was making the SIGWINCH path testable.
          family: seam-untested-on-the-real-side
          round: 1
        - id: BR-12
          severity: Minor
          title: the "an r final behind the private introducer is not DECSTBM" negative is asserted twice in comments and covered by no test
          detail: |-
            screen.go:121 and screen_test.go:105 both state the rule; TestScreenRegionLost has
            no \x1b[?<n>r case, and removing the ? branch's early return leaves the suite green.
            The issue Log lists this among deletion checks confirmed red.
          family: uncovered-negative-assertion
          round: 1
        - id: BR-13
          severity: Minor
          title: OSHost.state is written and never read, and h.resized is never closed so a range consumer leaks
          detail: |-
            hostty/os.go:20 stores the term.State under the mutex; only the closure's captured
            copy is used by Restore. Close() stops and closes h.sigs but leaves h.resized open,
            so termcmd's `for range host.Resized()` goroutine never returns.
          family: dead-field-and-leaked-consumer
          round: 1
        - id: BR-14
          severity: Minor
          title: probes/termsmoke exits via os.Exit(1) on failure, skipping the deferred kill of the pair child
          detail: |-
            probes/termsmoke/main.go. A failing probe run leaves the spawned pair process alive.
            Also splitAny (screen.go:166) allocates []byte(seps) once per scanned byte.
          family: probe-hygiene
          round: 1
      boundary: M1
      blocked: true
---

# Gate ledger — pair#146 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-22T18:31:03-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `chunking-invariance` Screen raises a false bell for any sequence longer than maxPending split across two reads
  screen.go:104 drops the ESC and rescans the payload as text when an incomplete
  sequence exceeds 256 bytes, so the sequence's terminating BEL is later counted by
  the plain-run scan at screen.go:90. Reproduced: a 300-byte OSC split across two
  Feed calls yields bell=true where the same bytes fed whole yield bell=false. The
  everyday trigger is OSC 52 clipboard writes, which are kilobytes and always cross
  a 4096-byte pty read boundary. Contradicts TakeBell's own doc and
  atlas/architecture.md:462. FuzzScreenFeed asserts chunk-invariance for AltScreen,
  Mouse and Pending but not for the two latches, which is why it cannot find this.
- **BR-2** [Important] `signal-goroutine-outlives-close` runShell now stops SIGWINCH delivery after closing every child pty, re-opening the scribecmd race
  termcmd/run.go:220 registers the host.Close() defer first, so LIFO runs it LAST --
  after mux.closeAll(). The watcher goroutine at run.go:246 is therefore still live
  during closeAll, and a SIGWINCH there runs resizeAll -> child.Resize ->
  pty.Setsize -> ptmx.Fd() concurrently with ptmx.Close(). The pre-migration code
  registered defer signal.Stop(winch) last so it ran first. workshop/lessons.md
  prescribes exactly this ordering from the scribecmd bug. Fix: register the
  host.Close() defer immediately after defer mux.closeAll().
- **BR-3** [Important] `fake-diverges-from-production` NewFakeChild's documented Wait/Done contract is false and no test pins either
  fake.go:20 documents "Wait returns ExitCode (default 0) immediately; Done is true
  once Exit is called, or immediately if the child was never running". Measured:
  Done() is false at construction and Wait() blocks indefinitely, because done is
  created open at fake.go:26 and only Exit closes it. The fake and the real Child
  have opposite Wait semantics, so an M2/M3 test written from the doc hangs rather
  than fails. No conformance check exists between the fake and a real Child, which is
  the ARCH-MOCK gap that would have caught it -- hostty_test.go:118 is the template.
- **BR-4** [Important] `fix-not-pinned-by-failing-test` the Ring copy-vs-reslice fix is claimed as a bug fix, but reverting it leaves the named test green
  ring.go:48-52 asserts re-slicing is "unbounded growth hiding behind a bounded
  Snapshot", Allocated() is exported solely to pin it, and the issue Log lists "ring
  trim" as a deletion check confirmed red. Reverting to the re-slice leaves
  TestRingDoesNotGrowWithoutBound green. Measured over 2000 appends into a 32-byte
  ring: reslice peaks at cap=48, copy sits at cap=64 -- re-slicing monotonically
  shrinks remaining capacity so append is guaranteed to reallocate. The copy is a
  reasonable cleanup; the defect it claims to fix does not exist.
- **BR-5** [Important] `plan-table-drift` the plan's Core-concepts table says termcmd.restoreTerminal is deleted; it still exists
  Plan row :89 and Task 1.4a :213 both state restoreTerminal is deleted and folded
  into hostty. The method survives at termcmd/run.go:1061 and writes
  hostty.ResetRegion. The behaviour the row is about -- one escape constant, one site
  -- is delivered and verified, so this is a table-accuracy defect rather than missing
  work, but it needs a "## Revisions" entry rather than a silent edit.
- **BR-6** [Minor] `needless-indirection` waitCode is duplicated verbatim across couchcore and ptychild, and asExitError now exists in three packages
  couchcore/runner.go:90 and ptychild/child.go:130 are byte-identical; ptychild's
  errors.go says it mirrors couchcore's. ARCH-DRY, in the diff whose stated thesis is
  that the second copy is the thing to avoid.
- **BR-7** [Minor] `stale-comment-reference` comments still cite queries.go, appendBuffer, tab.buffer and readPTY, all deleted by this diff
  termcmd/run.go:1046 ("see queries.go"), run.go:1053 ("appendBuffer re-slices
  tab.buffer from the output goroutine"), ptychild/replay.go:37 ("returns through
  readPTY").
- **BR-8** [Minor] `unsynchronised-shared-state` Child.SetSink writes c.sink unlocked with no fake-only guard while the pump reads it
  fake.go:57. Documented as being for fakes, but it is a method on *Child with no
  `if c.fake == nil` guard, so calling it on a real child races the read pump. No
  caller today.
- **BR-9** [Minor] `replay-duplicates-live-output` newTab widens the window where a chunk is both replayed and written live
  The ring is now filled inside the pump, before bufferSnapshotLocked(tab) at
  run.go:707, so a chunk landing in that gap is replayed by redrawTab and then
  written again by copyActiveOutput. A brand-new tab has nothing to replay; passing
  nil is the honest fix. The same shape recurs in M2 when the console attaches to a
  freshly started child.
- **BR-10** [Minor] `stale-build-target` make test-race still targets ./cmd/pair-wrap/, so Task 1.6's -race requirement has no runnable target
  make test-live landed this window; the test-race half of the same rule did not, and
  the target fails outright with "directory not found". Plan Task 1.6 requires
  go test ./cmd/... -race over the whole tree.
- **BR-11** [Minor] `seam-untested-on-the-real-side` OSHost's SIGWINCH coalescing has no test; only FakeHost's does
  TestFakeHostResizesCoalesce drives the fake. OSHost.watch()'s non-blocking send --
  the actual coalescing production depends on -- has no coverage, in the package whose
  stated purpose was making the SIGWINCH path testable.
- **BR-12** [Minor] `uncovered-negative-assertion` the "an r final behind the private introducer is not DECSTBM" negative is asserted twice in comments and covered by no test
  screen.go:121 and screen_test.go:105 both state the rule; TestScreenRegionLost has
  no \x1b[?<n>r case, and removing the ? branch's early return leaves the suite green.
  The issue Log lists this among deletion checks confirmed red.
- **BR-13** [Minor] `dead-field-and-leaked-consumer` OSHost.state is written and never read, and h.resized is never closed so a range consumer leaks
  hostty/os.go:20 stores the term.State under the mutex; only the closure's captured
  copy is used by Restore. Close() stops and closes h.sigs but leaves h.resized open,
  so termcmd's `for range host.Resized()` goroutine never returns.
- **BR-14** [Minor] `probe-hygiene` probes/termsmoke exits via os.Exit(1) on failure, skipping the deferred kill of the pair child
  probes/termsmoke/main.go. A failing probe run leaves the spawned pair process alive.
  Also splitAny (screen.go:166) allocates []byte(seps) once per scanned byte.

## Open findings

- **BR-1** [Important] `chunking-invariance` Screen raises a false bell for any sequence longer than maxPending split across two reads
- **BR-2** [Important] `signal-goroutine-outlives-close` runShell now stops SIGWINCH delivery after closing every child pty, re-opening the scribecmd race
- **BR-3** [Important] `fake-diverges-from-production` NewFakeChild's documented Wait/Done contract is false and no test pins either
- **BR-4** [Important] `fix-not-pinned-by-failing-test` the Ring copy-vs-reslice fix is claimed as a bug fix, but reverting it leaves the named test green
- **BR-5** [Important] `plan-table-drift` the plan's Core-concepts table says termcmd.restoreTerminal is deleted; it still exists
- **BR-6** [Minor] `needless-indirection` waitCode is duplicated verbatim across couchcore and ptychild, and asExitError now exists in three packages
- **BR-7** [Minor] `stale-comment-reference` comments still cite queries.go, appendBuffer, tab.buffer and readPTY, all deleted by this diff
- **BR-8** [Minor] `unsynchronised-shared-state` Child.SetSink writes c.sink unlocked with no fake-only guard while the pump reads it
- **BR-9** [Minor] `replay-duplicates-live-output` newTab widens the window where a chunk is both replayed and written live
- **BR-10** [Minor] `stale-build-target` make test-race still targets ./cmd/pair-wrap/, so Task 1.6's -race requirement has no runnable target
- **BR-11** [Minor] `seam-untested-on-the-real-side` OSHost's SIGWINCH coalescing has no test; only FakeHost's does
- **BR-12** [Minor] `uncovered-negative-assertion` the "an r final behind the private introducer is not DECSTBM" negative is asserted twice in comments and covered by no test
- **BR-13** [Minor] `dead-field-and-leaked-consumer` OSHost.state is written and never read, and h.resized is never closed so a range consumer leaks
- **BR-14** [Minor] `probe-hygiene` probes/termsmoke exits via os.Exit(1) on failure, skipping the deferred kill of the pair child
