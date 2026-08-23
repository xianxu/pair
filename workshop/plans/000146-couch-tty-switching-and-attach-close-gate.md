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
    - "n": 2
      timestamp: "2026-08-22T18:54:31-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Threshold moved 256 -> 64 KiB; the everyday OSC 52 case is genuinely cured, but chunk-invariance is not restored above the new bound. See I0 for the measurement.
          round: 2
        - id: BR-2
          disposition: addressed
          note: defer host.Close() now registers after defer mux.closeAll(), so LIFO runs it first; resized is closed by its only sender. No test pins the ordering -- that is I1, not a re-raise.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Doc corrected to match the code. Verified by revert - closing done at construction panics TestFakeChildExitReportsItsCode; the real-vs-fake conformance test exists but could not run in my shell.
          round: 2
        - id: BR-4
          disposition: addressed
          note: Claim retracted in ring.go's comment, Allocated()'s doc, and the issue Log, each carrying the cap=48-vs-64 measurement that falsified it.
          round: 2
        - id: BR-5
          disposition: addressed
          note: 'Core-concepts row now reads "modified (now writes hostty.ResetRegion; the method stays, the constant moved)", recorded via a ## Revisions entry.'
          round: 2
        - id: BR-6
          disposition: not-addressed
          note: waitCode still byte-identical at couchcore/runner.go:90 and ptychild/child.go:130; asExitError still wrapped in two packages.
          round: 2
        - id: BR-7
          disposition: not-addressed
          note: All three stale references remain (run.go:1052, run.go:1059, replay.go:37), and Makefile.local:73-77 is now a fourth -- it still explains test-race as scoped to pair-wrap after the target was repointed at ./cmd/...
          round: 2
        - id: BR-8
          disposition: not-addressed
          note: fake.go:65 still writes c.sink unlocked with no `if c.fake == nil` guard.
          round: 2
        - id: BR-9
          disposition: not-addressed
          note: newTab still calls bufferSnapshotLocked(tab) after ptychild.Start has launched the pump; a chunk landing in the gap is replayed and written live.
          round: 2
        - id: BR-10
          disposition: addressed
          note: test-race is now `go test -count=1 -race ./cmd/...` and runnable. Residual, folded into I1 rather than re-raised - neither test-race nor test-live is in make test's prerequisites.
          round: 2
        - id: BR-11
          disposition: addressed
          note: TestOSHostCoalescesRealSIGWINCH drives the real watcher against real signals. It skips without pty.Open, which it does not need -- that is I1, not a re-raise.
          round: 2
        - id: BR-12
          disposition: addressed
          note: Verified by revert - removing the `final != 'h' && final != 'l'` guard reddens TestScreenRegionLost/DECRSTR_private_r.
          round: 2
        - id: BR-13
          disposition: addressed
          note: OSHost.state deleted; resized closed by the watcher. Verified load-bearing - deleting `defer close(h.resized)` reddens an ungated copy of the pin in 2s.
          round: 2
        - id: BR-14
          disposition: addressed
          note: cleanup() runs before os.Exit(1) at main.go:119-120; splitAny replaced by splitParams, which allocates no separator slice per byte.
          round: 2
      findings:
        - id: BR-15
          severity: Important
          title: the pins for three of this round's fixes skip themselves in the environment the issue documents as its agent shell
          detail: |-
            2nd in this family. Do NOT fix the three instances -- the rule is that a fix is defended
            only by a test the default suite EXECUTES and that goes red without it; a t.Skipf on an
            unavailable dependency and a comment are both zero. Measured at HEAD: `go test
            ./cmd/internal/hostty/` reports ok with three silent skips (3 of 9 tests), including
            BR-11's and BR-13's pins. Two of the three need no pty at all -- rewritten against a
            plain temp file both pass, and with `defer close(h.resized)` deleted the ungated copy
            fails in 2s while the shipped test still skips and the package still reports ok.
            ptychild handles the identical constraint by failing loudly, so one milestone ships two
            handlings of one condition. BR-2's defer ordering has no test anywhere (grep runShell
            over termcmd's tests is empty), so reverting it leaves the suite green. Neither
            test-race nor test-live is in make test's prerequisites (Makefile.local:81).
          family: fix-not-pinned-by-failing-test
          round: 2
        - id: BR-16
          severity: Important
          title: probes/ is new top-level surface with no make target and no atlas entry
          detail: |-
            2nd in this family. Do NOT just add an atlas line -- the rule is that a committed probe
            is a first-class artifact: a make target, an entry in atlas/, and self-cleanup.
            probes/ was created in this window (git log --diff-filter=A -- probes/ -> fd550be).
            grep over Makefile, Makefile.local, tests/ and .github/ for "termsmoke|probes" returns
            nothing; grep over atlas/ and README.md for "probes/" returns nothing; atlas/index.md
            links every other file. The probe's own header states why it is committed -- output
            quoted in an issue Log must be re-runnable against a later commit -- and its 8/8 result
            IS quoted as M1 evidence. AGENTS.md section 8 asks for atlas updates on new surface and
            file-tree locations at each milestone close. Cheap to sweep now, with one member; the
            plan schedules three more smokes at Tasks 2.7, 3.5 and 4.6.
          family: probe-hygiene
          round: 2
        - id: BR-17
          severity: Minor
          title: the plan's Screen description declares MarginsDirty and Bell; the code has regionLost and bell behind Take* readers
          detail: |-
            2nd in this family on this issue, 3rd counting pair#145's BR-41. Do NOT just rename the
            two words -- the rule is that the plan's Core-concepts table is a hand-maintained
            restatement of code shapes, so either stop restating them (as atlas/couch.md now
            correctly does for the operation set) or append the ## Revisions entry in the same
            commit that changes the shape. Plan line 99 vs screen.go:39-42, 52-68.
          family: plan-table-drift
          round: 2
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

## Round 2 — 2026-08-22T18:54:31-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Threshold moved 256 -> 64 KiB; the everyday OSC 52 case is genuinely cured, but chunk-invariance is not restored above the new bound. See I0 for the measurement.
- BR-2 — addressed — defer host.Close() now registers after defer mux.closeAll(), so LIFO runs it first; resized is closed by its only sender. No test pins the ordering -- that is I1, not a re-raise.
- BR-3 — addressed — Doc corrected to match the code. Verified by revert - closing done at construction panics TestFakeChildExitReportsItsCode; the real-vs-fake conformance test exists but could not run in my shell.
- BR-4 — addressed — Claim retracted in ring.go's comment, Allocated()'s doc, and the issue Log, each carrying the cap=48-vs-64 measurement that falsified it.
- BR-5 — addressed — Core-concepts row now reads "modified (now writes hostty.ResetRegion; the method stays, the constant moved)", recorded via a ## Revisions entry.
- BR-6 — not-addressed — waitCode still byte-identical at couchcore/runner.go:90 and ptychild/child.go:130; asExitError still wrapped in two packages.
- BR-7 — not-addressed — All three stale references remain (run.go:1052, run.go:1059, replay.go:37), and Makefile.local:73-77 is now a fourth -- it still explains test-race as scoped to pair-wrap after the target was repointed at ./cmd/...
- BR-8 — not-addressed — fake.go:65 still writes c.sink unlocked with no `if c.fake == nil` guard.
- BR-9 — not-addressed — newTab still calls bufferSnapshotLocked(tab) after ptychild.Start has launched the pump; a chunk landing in the gap is replayed and written live.
- BR-10 — addressed — test-race is now `go test -count=1 -race ./cmd/...` and runnable. Residual, folded into I1 rather than re-raised - neither test-race nor test-live is in make test's prerequisites.
- BR-11 — addressed — TestOSHostCoalescesRealSIGWINCH drives the real watcher against real signals. It skips without pty.Open, which it does not need -- that is I1, not a re-raise.
- BR-12 — addressed — Verified by revert - removing the `final != 'h' && final != 'l'` guard reddens TestScreenRegionLost/DECRSTR_private_r.
- BR-13 — addressed — OSHost.state deleted; resized closed by the watcher. Verified load-bearing - deleting `defer close(h.resized)` reddens an ungated copy of the pin in 2s.
- BR-14 — addressed — cleanup() runs before os.Exit(1) at main.go:119-120; splitAny replaced by splitParams, which allocates no separator slice per byte.

### Raised

- **BR-15** [Important] `fix-not-pinned-by-failing-test` the pins for three of this round's fixes skip themselves in the environment the issue documents as its agent shell
  2nd in this family. Do NOT fix the three instances -- the rule is that a fix is defended
  only by a test the default suite EXECUTES and that goes red without it; a t.Skipf on an
  unavailable dependency and a comment are both zero. Measured at HEAD: `go test
  ./cmd/internal/hostty/` reports ok with three silent skips (3 of 9 tests), including
  BR-11's and BR-13's pins. Two of the three need no pty at all -- rewritten against a
  plain temp file both pass, and with `defer close(h.resized)` deleted the ungated copy
  fails in 2s while the shipped test still skips and the package still reports ok.
  ptychild handles the identical constraint by failing loudly, so one milestone ships two
  handlings of one condition. BR-2's defer ordering has no test anywhere (grep runShell
  over termcmd's tests is empty), so reverting it leaves the suite green. Neither
  test-race nor test-live is in make test's prerequisites (Makefile.local:81).
- **BR-16** [Important] `probe-hygiene` probes/ is new top-level surface with no make target and no atlas entry
  2nd in this family. Do NOT just add an atlas line -- the rule is that a committed probe
  is a first-class artifact: a make target, an entry in atlas/, and self-cleanup.
  probes/ was created in this window (git log --diff-filter=A -- probes/ -> fd550be).
  grep over Makefile, Makefile.local, tests/ and .github/ for "termsmoke|probes" returns
  nothing; grep over atlas/ and README.md for "probes/" returns nothing; atlas/index.md
  links every other file. The probe's own header states why it is committed -- output
  quoted in an issue Log must be re-runnable against a later commit -- and its 8/8 result
  IS quoted as M1 evidence. AGENTS.md section 8 asks for atlas updates on new surface and
  file-tree locations at each milestone close. Cheap to sweep now, with one member; the
  plan schedules three more smokes at Tasks 2.7, 3.5 and 4.6.
- **BR-17** [Minor] `plan-table-drift` the plan's Screen description declares MarginsDirty and Bell; the code has regionLost and bell behind Take* readers
  2nd in this family on this issue, 3rd counting pair#145's BR-41. Do NOT just rename the
  two words -- the rule is that the plan's Core-concepts table is a hand-maintained
  restatement of code shapes, so either stop restating them (as atlas/couch.md now
  correctly does for the operation set) or append the ## Revisions entry in the same
  commit that changes the shape. Plan line 99 vs screen.go:39-42, 52-68.

## Open findings

- **BR-1** [Important] `chunking-invariance` Screen raises a false bell for any sequence longer than maxPending split across two reads
- **BR-6** [Minor] `needless-indirection` waitCode is duplicated verbatim across couchcore and ptychild, and asExitError now exists in three packages
- **BR-7** [Minor] `stale-comment-reference` comments still cite queries.go, appendBuffer, tab.buffer and readPTY, all deleted by this diff
- **BR-8** [Minor] `unsynchronised-shared-state` Child.SetSink writes c.sink unlocked with no fake-only guard while the pump reads it
- **BR-9** [Minor] `replay-duplicates-live-output` newTab widens the window where a chunk is both replayed and written live
- **BR-15** [Important] `fix-not-pinned-by-failing-test` the pins for three of this round's fixes skip themselves in the environment the issue documents as its agent shell
- **BR-16** [Important] `probe-hygiene` probes/ is new top-level surface with no make target and no atlas entry
- **BR-17** [Minor] `plan-table-drift` the plan's Screen description declares MarginsDirty and Bell; the code has regionLost and bell behind Take* readers
