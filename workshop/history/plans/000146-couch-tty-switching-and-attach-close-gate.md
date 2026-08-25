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
    - "n": 3
      timestamp: "2026-08-22T19:08:56-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Resync is genuinely pinned but insufficient - 3 of 10 measured shapes still diverge, and the 4096-byte production path now DROPS a real BEL where whole-feed rings it.
          round: 3
        - id: BR-6
          disposition: not-addressed
          note: waitCode still byte-identical at couchcore/runner.go:90 and ptychild/child.go:130; asExitError still a one-line wrapper in both packages.
          round: 3
        - id: BR-7
          disposition: not-addressed
          note: All four remain (run.go:1062, run.go:1069, replay.go:37, Makefile.local:73-77); bufferSnapshotLocked's stated m.mu contract is now false as well as stale.
          round: 3
        - id: BR-8
          disposition: not-addressed
          note: fake.go:65 still writes c.sink unlocked with no `if c.fake == nil` guard.
          round: 3
        - id: BR-9
          disposition: not-addressed
          note: newTab still snapshots at run.go:723 after ptychild.Start has launched the pump.
          round: 3
        - id: BR-15
          disposition: addressed
          note: Verified - hostty runs 8 of 9 tests in this sandboxed shell with zero skips (the 9th fails loudly), and swapping teardown's two statements reddens TestTeardownStopsTheWatcherBeforeClosingChildren from the default suite.
          round: 3
        - id: BR-16
          disposition: addressed
          note: make test-smoke exists, atlas/index.md carries an entry stating what earns a place in probes/, and cleanup runs before os.Exit(1).
          round: 3
        - id: BR-17
          disposition: addressed
          note: The Core-concepts row now describes what Screen answers and points at screen.go instead of restating field names; residual copies in Task 1.3 and the Child bullet noted as a plan recommendation, not re-raised.
          round: 3
      findings:
        - id: BR-18
          severity: Important
          title: FakeHost panics on a post-Close SetSize where OSHost is inert, and both conformance tests stop at the terminal transition
          detail: |-
            2nd in this family. Do NOT just guard SetSize -- the rule is that every lifecycle
            transition a fake exposes must match the real implementation's, and the conformance
            test must drive BOTH past the terminal state rather than up to it. BR-3's class fix
            (TestFakeChildConformsToRealChildLifecycle) had the right shape and stopped one step
            short; TestCloseReleasesResizedConsumers stops at Close too. Measured at HEAD:
            FakeHost.Close closes h.resized under h.mu while SetSize sends outside the lock and
            never consults h.closed, so a post-Close SetSize panics with "send on closed channel"
            while OSHost absorbs a 20-signal SIGWINCH burst inertly. Enumeration this implies -
            Host post-Close is {resize, Write, Size, MakeRaw}, 1 of 4 diverges fatally; Child
            post-Close is {Write, Resize, Signal}, 3 of 3 diverge (fake returns n=1/nil, nil, nil;
            a real child returns "file already closed", an ioctl error, "process already
            finished"). 4 of 7 post-terminal-state stimuli diverge, 1 fatally. FakeHost is the
            double M2's console tests are built on, so a panic there crashes a run instead of
            failing it.
          family: fake-diverges-from-production
          round: 3
      boundary: M1
      blocked: true
    - "n": 4
      timestamp: "2026-08-22T19:28:37-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: 'Round 3 genuinely closed the systematic break (measured: 10 shapes x 4 chunkings, 400 randomized iterations, zero divergences). One boundary residual remains -- skipTerminator drops the ESC it is commented to hold, so an ST split across a read swallows the next real bell. 1 of 70,550 cut positions.'
          round: 4
        - id: BR-6
          disposition: addressed
          note: Verified - procutil.WaitCode is the single source, both byte-identical copies and both errors.As wrappers deleted; couchcore/errors.go and ptychild/errors.go no longer exist.
          round: 4
        - id: BR-7
          disposition: not-addressed
          note: Makefile.local fixed; run.go:1062, run.go:1069 and replay.go:37 remain, and bufferSnapshotLocked's stated m.mu contract is now false as well as stale.
          round: 4
        - id: BR-8
          disposition: not-addressed
          note: fake.go:70 still writes c.sink unlocked with no `if c.fake == nil` guard while the pump reads it.
          round: 4
        - id: BR-9
          disposition: not-addressed
          note: newTab still snapshots at run.go:723 after ptychild.Start launched the pump at :707; both interleavings traced, the duplicate-write one is real.
          round: 4
        - id: BR-18
          disposition: addressed
          note: Verified by revert with mutate+compile+traverse confirmed - deleting FakeHost.SetSize's `if h.closed` guard panics TestHostsAgreeAfterClose at fake.go:72. Both conformance tests now drive past the terminal transition.
          round: 4
      findings:
        - id: BR-19
          severity: Minor
          title: frame treats DCS/APC/PM/SOS as two-byte escapes, so their payloads are scanned as plain text and a BEL inside one falsely rings
          detail: |-
            screen.go:275-278's `default: return 2, true` covers ESC-c style two-byte escapes only;
            the string-terminated classes fall through it. Measured: "\x1bP+q616263\x07\x1b\\" ->
            bell=true; "\x1b_Ga=T,f=100;PAYLOAD\x07\x1b\\" -> bell=true; PM and SOS likewise; and
            "\x1bPtmux;\x1b[?1049h\x1b\\" -> alt=true, region=true. 4 of the 5 string-terminated
            classes leak, OSC being the one covered. Contradicts TakeBell's own doc ("BEL is only
            counted outside a sequence") and atlas/architecture.md:462, which states it as a design
            property. Chunk-invariant, so not the chunking family -- the rule is that the framing
            must cover every sequence class the invariant is claimed over. Reachability today is low
            (kitty-graphics APC and XTGETTCAP DCS carry base64/hex; sixel's alphabet excludes BEL),
            so this is a stated invariant narrower than claimed rather than a live false positive.
            Fix is one `case 'P', '_', '^', 'X':` routing to the same ansi.OSCEnd scan.
          family: framing-omits-sequence-class
          round: 4
        - id: BR-20
          severity: Minor
          title: Child.Replay has zero production callers while the one repaint site reimplements it
          detail: |-
            child.go:170-173 documents itself as "what a repaint should write... Prefer Replay for
            repainting a screen"; redrawTab (termcmd/run.go:1062-1065) hand-composes HomeAndClear +
            StripQueries(Snapshot()), which is Replay() plus the clear. grep confirms Replay() is
            called only from child_test.go. 2nd in this family - the rule is that a helper naming a
            decision must be the only place that decision is made; if the sole production caller
            reimplements it, either the caller adopts it or the helper is deleted. Prevalence: 1
            helper, 0 production callers, 1 site reimplementing. Worth closing now because M3 Task
            3.3's contract spells the same expression out again for couch's attach path, which would
            make it two divergent repaint policies rather than one.
          family: needless-indirection
          round: 4
      boundary: M1
      blocked: false
    - "n": 5
      timestamp: "2026-08-23T08:39:37-07:00"
      agent: claude
      findings:
        - id: BR-21
          severity: Critical
          title: MidSequence() describes the newest chunk READ, not the chunk WRITTEN, so the row paint still splices into the child's escape sequences
          detail: |-
            ptychild's pump feeds Screen before calling the sink (child.go:113-120) and the console
            drains a 256-deep channel later, so when console.go:182 asks MidSequence() about the chunk
            it just wrote, the answer describes a later chunk. Reproduced in a scratch copy by holding
            the console inside host.Write while the child completes the sequence -- emitted stream was
            "\x1b[2J\x1b[38;2;76" + "\x1b[1;23r\x1b7\x1b[24;1H\x1b[2K[brain]\x1b8" + ";82;88mCOLOURED",
            byte-for-byte the nvim corruption this commit claims to fix. console_test.go:191-196
            describes this exact ordering and concludes the bug "does not reproduce"; the shipped test
            avoids the window rather than covering it. Second mode, measured: Screen.Pending() is
            len(pending), which is 0 while skipping != skipNone, so MidSequence() returns false in the
            middle of an over-64KiB OSC 52 -- the everyday clipboard case this repo already documents.
            Fix: compute pending-ness over the bytes the console has EMITTED (a per-pane ptychild.Screen
            fed as chunks are written, which is what console_live_test.go's splicedPaint already does),
            fold in the skipping state, and serialise host writes -- onChunk, watchResize and onHotkey
            all write to c.host today with no ordering between them.
          family: guard-reads-wrong-stream-position
          round: 5
        - id: BR-22
          severity: Critical
          title: a lone ESC keystroke is held indefinitely by the Interceptor and then delivered glued to the next key as a meta prefix
          detail: |-
            sequenceAt (keys.go:127-141) classifies a bare 0x1b as seqPartial because it is a real prefix
            of "\x1b[200~", so Feed holds it (keys.go:93) and returns nothing. Measured: Feed("\x1b") ->
            before=""; the following Feed("i") -> before="\x1bi", which a child reads as Alt+i. Driven
            end to end through the console, an ESC keystroke never reaches the child at all, and held
            bytes are never flushed at teardown. ESC is interrupt in Claude Code and mode-switch in nvim
            -- the Spec cites both. Caveat stated honestly: once zellij's Kitty protocol is up the
            terminal likely sends "\x1b[27u", which falls through as seqNone and is forwarded, which is
            plausibly why the smoke did not see it; the legacy encoding still governs before the child
            enables KKP (pair's config picker), after it disables it, and for M3's panel. Fix: hold only
            from two bytes on, or flush a held run when a read ends without extending the match, and add
            the negative test -- FuzzInterceptorFeed bounds output length only, so it cannot find this.
            workbenchshortcut.FindChord, the shape this copies, holds nothing.
          family: prefix-parks-a-complete-key
          round: 5
        - id: BR-23
          severity: Critical
          title: couch start off a tty spawns and registers the child, gives it a 0-row pty, then exits 1 with no output
          detail: |-
            consoleRunner (run.go:157-170) type-asserts stdin/stdout to *os.File and never asks whether
            they are terminals, while dimCodes at run.go:305 in the same file already uses
            term.IsTerminal. With stdin a pipe -- any script, cron, or agent shell, including #148's
            advisor -- host.Size() fails so Console keeps size{0,0}, ChildSize() returns 0x0 and the
            child is started on a zero-row pty; then MakeRaw() fails and console.go:103-106 discards the
            wrapped error and returns 1. Measured through RunWithRuntime: exit code 1, stderr "". The
            actor record persists, the child is hung up at process exit, and nothing mentions
            --no-console. Fix: gate the console on term.IsTerminal for both fds and fall back to the
            announced ExecRunner path; report the MakeRaw/Size errors instead of collapsing them to 1.
          family: swallowed-seam-error
          round: 5
        - id: BR-24
          severity: Important
          title: the milestone's central wiring is unpinned -- disabling the console entirely leaves the whole suite green
          detail: |-
            3rd in this family on this issue. Do NOT fix these two instances -- the rule, extending round
            2's, is that the mutation you must run is removal of the WIRING, not of a helper: every
            behaviour a boundary claims, including its central one, needs a default-suite test that goes
            red when that behaviour is deleted. Measured, both green under mutation: (a) forcing
            consoleRunner to always return (nil, ExecRunner) -- couch start is no longer the console --
            leaves ./cmd/internal/couchcmd and ./cmd/internal/couchtty ok, though run_test.go:34-40
            claims the branch "is observable in the rendered output"; only the --no-console side asserts
            anything. (b) deleting the path == "" -> "." default in ops.go:78-81, which is Decision 1's
            "cd brain and couch start is what makes brain home", leaves the tree green.
          family: fix-not-pinned-by-failing-test
          round: 5
        - id: BR-25
          severity: Important
          title: probes/zellijpark ships with no make target and no atlas entry, one round after that rule was written
          detail: |-
            3rd in this family. Do NOT just add a target for zellijpark. The rule was already stated in
            round 2 -- a committed probe is a first-class artifact: a make target, an atlas/ entry, and
            self-cleanup -- and the very next probe skipped two of three. make test-smoke
            (Makefile.local:50-51) runs only termsmoke; atlas/index.md:17-21 names only termsmoke; and
            zellijpark's output is quoted as the evidence that corrected workshop/projects/couch.md's
            park-vs-kill claim. Class fix: make the target enumerate probes/ so a new probe cannot skip
            it, and state the convention in the atlas entry rather than listing members.
          family: probe-hygiene
          round: 5
        - id: BR-26
          severity: Important
          title: the plan's Decision 11 and three Core-concepts rows now contradict the code, with no Revisions entry
          detail: |-
            3rd in this family on this issue (4th counting pair#145 BR-41). Do NOT correct the rows. The
            round-2 rule was "stop maintaining a second copy of a code shape in prose"; this round shows
            it is owed by DECISIONS and TASK CONTRACTS too -- a plan statement asserting external
            behaviour must be measured before it is written, and a boundary that reverses a Decision
            writes the Revisions entry in the same window. Measured drift, none of it recorded: Decision
            11 and Task 2.6a still say --layout2 is removed and that resume "refuses any third argv
            element (launcher/args.go:104)", which is false and which the code plus couch_test.go:588
            now pin the opposite of; StatusModel/RenderStatusRow are declared at couchtty/statusrow.go,
            a file that does not exist (they are in reserve.go); couchcore.TerminalHandle is declared at
            couchcore/runner.go but lives in ptyrunner.go, and Task 2.1 still specifies Terminal() as an
            interface where the code deliberately returns *ptychild.Child. Task 1.3:199 and Task 2.5:296
            still say MarginsDirty, flagged as a residual at M1 round 4 and since consumed.
          family: plan-table-drift
          round: 5
        - id: BR-27
          severity: Important
          title: three members added this window have zero writers -- FakeRunner.Sink, FakeRunner.Emit, and StatusActor.Bell
          detail: |-
            2nd in this family. Do NOT delete just one. The rule: a fake's surface, and a model field,
            must be exercised by the flow it exists for; a member added for symmetry with production and
            set at zero call sites is decoration that reads as coverage. Measured: FakeRunner.Sink
            (runner_fake.go:48) is never assigned anywhere in the tree -- production uses PtyRunner and
            testRT.NewCouchWith discards the runner; FakeRunner.Emit (runner_fake.go:228) has no callers,
            left over from the snapshot-content conformance design the file's own comment says was
            abandoned; StatusActor.Bell is hardcoded false at console.go:207, so the "*" marker
            reserve_test.go:58 pins can never appear in production.
          family: dead-field-and-leaked-consumer
          round: 5
        - id: BR-28
          severity: Important
          title: Task 2.3's ctrl-space audit of claude and nvim was never recorded in the issue Log
          detail: |-
            The plan makes it a deliverable with an explicit clause -- check claude and nvim for a
            ctrl-space binding, record the result in the issue Log, and if something rides on it say how
            a literal ctrl-space reaches a child. The Log mentions "the ctrl-space audit" only
            retrospectively at :750; its result appears nowhere. Not ceremony: in Vim, <C-Space> maps to
            <C-@>, which in insert mode is i_CTRL-@, and couch now shadows the key in both encodings
            with no documented escape hatch. An unrecorded audit is an unperformed audit.
          family: undelivered-plan-step
          round: 5
        - id: BR-29
          severity: Important
          title: Deliver drops child output on a full buffer, justified by a repaint-from-ring that does not exist at this boundary
          detail: |-
            console.go:76-83's default case drops the chunk and the comment says "the ring still has it,
            so the next repaint is correct". grep for Replay/Snapshot over couchtty's non-test files is
            empty: M2 has no repaint-from-ring, so a dropped chunk is gone from the screen permanently,
            and a chunk dropped mid-sequence leaves the host terminal corrupted with nothing to resync
            it. Either block with a bounded wait, or latch a "this pane lost bytes" flag that forces a
            full repaint once M3 lands the replay -- and correct the comment now, since it is a forward
            reference presented as a present-tense guarantee.
          family: unrecoverable-silent-drop
          round: 5
        - id: BR-30
          severity: Important
          title: the forced resume tag is the tree's basename, so two different trees resume one zellij session
          detail: |-
            couch.go:107 uses launcher.DefaultTag, which is NormalizeDisplayComponent(filepath.Base(path))
            (tag.go:25-31). Any two trees sharing a basename -- a git worktree, a clone under a different
            parent, foo.bar vs foo_bar which both normalise to foo_bar -- derive the same tag, so couch
            start on the second attaches to the first's session while the registry records an actor
            against the second tree. That breaks the correspondence between couch's one-agent-per-TREE
            key and the session the operator lands in. TestSpawnDerivesTheTagFromTheTreeNotTheCwd pins
            derivation; nothing pins uniqueness. Cheap mitigation until #149: suffix a short hash of the
            full tree path.
          family: derived-id-not-unique
          round: 5
        - id: BR-31
          severity: Important
          title: README still describes couch as a spawner, and atlas/architecture.md still describes Screen as reporting region-lost
          detail: |-
            README.md:260-265 says couch "registers agent sessions one-per-worktree and can spawn them";
            couch start now owns the operator's terminal for its lifetime, intercepts ctrl-space,
            reserves a row, and has a new --no-console flag -- user-facing surface a reader types.
            atlas/architecture.md:458-460 still describes Screen as reporting "region-lost edges
            (DECSTBM, RIS, or an alt-screen transition)", omitting the ERASE case that is the entire
            discovery of this milestone, and does not mention MidSequence, new public surface on
            ptychild.Child that the console's correctness depends on. atlas/couch.md itself is updated
            well; these two are the same window's other doc sites.
          family: docs-lag-the-surface
          round: 5
        - id: BR-32
          severity: Minor
          title: ChildRows(0) returns 0 while its doc says "It never returns zero", and no test covers the boundary case
          detail: |-
            2nd in this family. reserve.go:21-26 returns hostRows for hostRows <= 1, so ChildRows(0) is
            0; reserve_test.go:31 tests only ChildRows(1). The rule: an invariant asserted in a doc
            comment needs a test at its boundary case, or the comment is the lie. Reachable in
            production via the non-tty path (see the swallowed-seam-error finding), where it produces a
            zero-row pty.
          family: uncovered-negative-assertion
          round: 5
        - id: BR-33
          severity: Minor
          title: Console.Run never calls Stop() or host.Close(), so the resize watcher and the SIGWINCH registration outlive the console
          detail: |-
            2nd in this family. console.go:102-130 defers restore() and release() but nothing stops the
            console's own goroutines or closes the host. The rule from M1's BR-2 applies unchanged:
            teardown is explicit and ordered, not emergent -- here from process exit rather than from
            defer LIFO. Latent while there is one console per process; ptyrunner.go:88's own comment
            anticipates #147 putting more than one in.
          family: signal-goroutine-outlives-close
          round: 5
        - id: BR-34
          severity: Minor
          title: several comments overstate or misplace what the code does
          detail: |-
            2nd in this family; low value individually, listed together. screen.go's case 'J' says "ED,
            every form" but \x1b[?2J (DECSED) returns early at the private-introducer branch;
            Screen.Pending's doc says "Exported for the tests that pin the bound" and now has a
            production consumer; ops.go:65 says the path default is applied "at the CLI" when it is
            applied in couchcore's Invoke; atlas/couch.md:32 reads "hands the child its own stdio and
            block".
          family: stale-comment-reference
          round: 5
        - id: BR-35
          severity: Minor
          title: the live conformance scenario and Run's exit select are both racy by construction
          detail: |-
            conformance_live_test.go:245-249 reads doneBeforeExit AFTER the write that makes the child
            exit and then Fatalfs if it is true. console.go's select between c.chunks and exited is a
            coin flip on exit, so the child's final output is likely dropped before release() clears the
            row. Also test-side: vtscreen_test.go redefines min, shadowing the builtin, and
            waitFor/waitLong/waitUntilTrue are three near-identical polling helpers across two packages.
          family: test-harness-races
          round: 5
      boundary: M2
      blocked: true
    - "n": 6
      timestamp: "2026-08-23T09:05:24-07:00"
      agent: claude
      dispose:
        - id: BR-21
          disposition: not-addressed
          note: hostScan is right and pinned, but "serialise host writes" was skipped; applyLayout's Reserve write splices deterministically on SIGWINCH, and the hotkey path reproduces with no injected delay.
          round: 6
        - id: BR-22
          disposition: not-addressed
          note: the discriminator covers only a sole-byte read; Feed("abc\x1b")+Feed("i") still yields "\x1bi", Feed("\x1b\x1b") holds one ESC, and held is still never flushed.
          round: 6
        - id: BR-23
          disposition: not-addressed
          note: the no-terminal fallback is in and pinned; Console.Run still returns 1 silently on MakeRaw error, and the gate reads only the input fd.
          round: 6
        - id: BR-24
          disposition: not-addressed
          note: both new pins t.Skipf on pty.Open in the documented environment, so the disable-the-console mutation is still green; the path default is still unpinned.
          round: 6
        - id: BR-25
          disposition: not-addressed
          note: Makefile.local hardcodes a second go run line and atlas/index.md lists the new member; neither is the enumeration the class fix asked for.
          round: 6
        - id: BR-26
          disposition: not-addressed
          note: the Revisions entry landed for Decision 11 but asserts a table sweep that did not happen; five stale sites remain unchanged.
          round: 6
        - id: BR-27
          disposition: addressed
          note: Sink/Emit deleted, Bell wired and pinned (deletion check red); noting only that the !isActive guard is unreachable until M3 attaches a second pane.
          round: 6
        - id: BR-28
          disposition: addressed
          note: the audit and what it missed are both in the issue Log now.
          round: 6
        - id: BR-29
          disposition: addressed
          note: Deliver blocks and yields to stop; reverting to drop reddens TestConsoleDoesNotDropChildOutputUnderBurst.
          round: 6
        - id: BR-30
          disposition: withdrawn
          note: 'verified independently: scope.Key plus liveOwnedByOther means a live collision bumps the suffix; the mechanism I claimed does not hold.'
          round: 6
        - id: BR-31
          disposition: addressed
          note: README and atlas/architecture.md both reconciled, MidSequence and the erase case included.
          round: 6
        - id: BR-32
          disposition: not-addressed
          note: reserve.go:21-26 and reserve_test.go:31 unchanged; ChildRows(0) is still 0 under a doc saying it never returns zero.
          round: 6
        - id: BR-33
          disposition: not-addressed
          note: Console.Run still defers only restore and release; no Stop(), no host.Close().
          round: 6
        - id: BR-34
          disposition: not-addressed
          note: Pending's doc updated; the "ED, every form" comment, ops.go:65's "at the CLI", and atlas/couch.md's "stdio and block" are unchanged, and run_test.go:38-42 still claims the console branch is observable in the rendered output.
          round: 6
        - id: BR-35
          disposition: not-addressed
          note: min renamed to minInt; doneBeforeExit is still read after the write that ends the child, Run's exit select is unchanged, and waitUntilTrue still duplicates waitFor across packages.
          round: 6
      findings:
        - id: BR-36
          severity: Important
          title: Task 2.7's operator smoke is partly unrecorded and two of its items are carried to M3 with no Revisions entry
          detail: |-
            2nd in this family. Do NOT just record the missing observations. The rule, which BR-28's
            instance fix did not reach: a milestone's Plan line enumerates that milestone's deliverables,
            so moving one to a later milestone is a scope event and is written as a plan `## Revisions`
            entry plus an amended issue `## Plan` line in the same window -- a Log paragraph is where a
            deferral goes to be forgotten. Measured: the issue `## Plan` M2 line still reads "Smoke step 1
            (one real pair + claude child, resize, nvim in and out, reattach across a kill -9) lands here",
            and plan Task 2.7 still lists the kill -9 reattach and the park-vs-kill determination through
            the full couch stop path, while the Log (2026-08-23) carries both to M3. Of Task 2.7's seven
            recorded-observation items the Log records three (row survives pair startup, ctrl-space
            intercepted, layout2); resize reflow, the row while claude streams, nvim in-and-out (the
            margin-reset case Decision 4 rests on) and "quitting restores the terminal" appear nowhere.
            atlas/couch.md:70-72 nonetheless claims the section is "confirmed by operator smoke on the
            full Ghostty -> couch -> pair -> zellij -> claude stack".
          family: undelivered-plan-step
          round: 6
        - id: BR-37
          severity: Important
          title: atlas/couch.md still tells the reader the console asks Child.MidSequence(), a method this round deleted, on the stream BR-21 proved wrong
          detail: |-
            2nd in this family, and a regression introduced by the fix commit rather than a leftover. Do
            NOT just edit the line. The rule BR-31's instance fix did not reach: the commit that changes a
            public surface updates every doc that names it, and the cheap enforcement is that an
            identifier named in atlas/ must be greppable in the tree -- run that grep at the boundary
            instead of re-reading the prose. Measured: `grep -rn MidSequence atlas/ cmd/internal/ptychild/`
            returns atlas/couch.md:69 "the console therefore asks `Child.MidSequence()` and defers", while
            `Child.MidSequence` was removed in 5975f10 and the console now frames its own written stream
            via `Console.hostScan`. The doc therefore teaches the exact mistake the review caught -- ask
            the child -- which is worse than being merely stale. atlas/architecture.md:461 has the same
            surface described correctly, so the two atlas files now disagree.
          family: docs-lag-the-surface
          round: 6
      boundary: M2
      blocked: true
    - "n": 7
      timestamp: "2026-08-23T09:22:30-07:00"
      agent: claude
      dispose:
        - id: BR-21
          disposition: addressed
          note: Run is the sole host writer and hostScan is fed child bytes only; re-adding applyLayout's write reddens TestConsoleNeverSplicesFromAnyPath 3/3, and removing the writeOwn gate reddens 3 tests.
          round: 7
        - id: BR-22
          disposition: addressed
          note: discriminator keys on partial length; measured "abc\x1b"+"i" -> "abc\x1b" then "i", "\x1b\x1b" both forwarded, split CSI-u still fires; removal reddens 4 tests.
          round: 7
        - id: BR-23
          disposition: addressed
          note: both fds gated, WantsConsole pure and pinned unconditionally, MakeRaw error reported and pinned; the both-fds composition itself is unpinned and folds into BR-24's seam.
          round: 7
        - id: BR-24
          disposition: not-addressed
          note: 'measured at HEAD: forcing consoleRunner to (nil, ExecRunner) leaves couchcmd and couchtty ok, and deleting the path default leaves the tree green -- TestStartDefaultsItsPathToCwd asserts on ArgSpec.Required, not on the default.'
          round: 7
        - id: BR-25
          disposition: addressed
          note: make test-smoke enumerates probes/*/ and atlas/index.md states the convention; termsmoke defaults bin to ./bin/pair so dropping the argv is safe.
          round: 7
        - id: BR-26
          disposition: not-addressed
          note: 0 of the 5 named sites changed; only the prose bullets moved, while the round-2 Revisions entry asserts the table rows did -- the second consecutive entry claiming a sweep it did not perform.
          round: 7
        - id: BR-32
          disposition: not-addressed
          note: reserve.go:21-26 and reserve_test.go:28 unchanged; ChildRows(0) is still 0 under a doc saying it never returns zero.
          round: 7
        - id: BR-33
          disposition: not-addressed
          note: Run still defers only restore and release; with Deliver now blocking, returning via the exited case leaves the child's pump permanently blocked because stop is never closed.
          round: 7
        - id: BR-34
          disposition: not-addressed
          note: screen.go:277 "ED, every form", ops.go:64 "at the CLI" and atlas/couch.md:32 "stdio and block" unchanged; run_test.go:40 still claims the console branch is observable in the rendered output, measured false again; new -- consoleRunner's doc comment now runs into WantsConsole's.
          round: 7
        - id: BR-35
          disposition: not-addressed
          note: doneBeforeExit is still read after the write that ends the child, Run's exit select is unchanged, and waitUntilTrue still duplicates waitFor -- four pollers across three packages now.
          round: 7
        - id: BR-36
          disposition: not-addressed
          note: the plan Revisions entry and the Log carry landed, but the issue Plan M2 line still says the kill -9 reattach lands at M2, and resize reflow / row-while-claude-streams / nvim in-and-out are still unrecorded by name.
          round: 7
        - id: BR-37
          disposition: addressed
          note: atlas/couch.md no longer names Child.MidSequence; I ran the class check -- every backticked identifier in atlas/couch.md resolves in cmd/.
          round: 7
      findings:
        - id: BR-38
          severity: Important
          title: README documents the pty and --no-console but not ctrl-space or the path default, and the atlas-identifier check has no README counterpart
          detail: |-
            3rd in this family. Do NOT just add two README lines. BR-37's class fix landed for atlas/ --
            I ran the grep and every identifier in atlas/couch.md resolves in cmd/ -- but it has no
            counterpart for README's TYPED surface, so the same class recurred at the site the
            enumeration does not cover. Measured: couch start now claims ctrl-space globally from every
            child in both encodings (legacy NUL and CSI-u), and its <repo> argument is now optional
            defaulting to "." -- Decision 1's entire mechanism. README.md:261-267 documents the pty, the
            reserved row and --no-console and neither of those; it still shows `couch start <repo>` as
            though the argument were required. An operator whose ctrl-space stops reaching their editor
            has no documented explanation. Rule to enforce at the boundary: enumerate the flags,
            keybindings and argument defaults the window added and confirm each appears in README, the
            same way atlas identifiers are now grepped.
          family: docs-lag-the-surface
          round: 7
        - id: BR-39
          severity: Minor
          title: onChunk consumes TakeRowDirty for every pane but acts on it only for the active one, so an inactive pane's dirty-row latch is thrown away
          detail: |-
            console.go:320 reads rowDirty for every chunk of every pane; console.go:333 acts only
            `if rowDirty && isActive`. TakeRowDirty is a latch that clears on read, so for an inactive
            pane the event is consumed and discarded -- by the time the operator switches to it, the
            console has no record that its row needs re-asserting. The bell path immediately above gets
            this right (it stores into p.bell). Unreachable in M2 production, which attaches exactly one
            pane; reachable the moment M3 attaches a second. Fix: store into a per-pane dirty flag and
            consume it at switch time, mirroring how p.bell is handled.
          family: latch-consumed-by-wrong-consumer
          round: 7
      boundary: M2
      blocked: true
    - "n": 8
      timestamp: "2026-08-23T09:42:21-07:00"
      agent: claude
      dispose:
        - id: BR-24
          disposition: not-addressed
          note: consoleRunnerFor and the path default are both pinned now (both mutations red), but gutting consoleRunner itself is still green, as is dropping the outFile half of the tty gate.
          round: 8
        - id: BR-26
          disposition: addressed
          note: all five named sites measured changed and the Revisions entry enumerates them individually; the sweep introduced one new instance, raised separately.
          round: 8
        - id: BR-32
          disposition: not-addressed
          note: unchanged, and the doc's second clause is false too -- on a 1-row host PaintRow still draws the row, so "simply not drawn" is wrong at hostRows==1 as well as ChildRows(0)==0.
          round: 8
        - id: BR-33
          disposition: not-addressed
          note: unchanged, plus a new site -- NewOSHost registers SIGWINCH and starts watch() inside consoleRunnerFor, before Invoke, so a refused couch start leaks both without Run ever being called.
          round: 8
        - id: BR-34
          disposition: not-addressed
          note: three of four sites unchanged; run_test.go:41's claim measured false again; go doc WantsConsole confirmed to print consoleRunner's comment; new -- render's StartResult branch still says "this milestone has no pty".
          round: 8
        - id: BR-35
          disposition: not-addressed
          note: unchanged -- doneBeforeExit still read after the write that ends the child, Run's exit select unchanged, four pollers across three packages.
          round: 8
        - id: BR-36
          disposition: addressed
          note: item-by-item smoke record, issue Plan M2 line amended, plan Revisions entry, carries stated with reasons and one item explicitly not claimed; only atlas/couch.md's smoke sentence still over-reaches.
          round: 8
        - id: BR-38
          disposition: addressed
          note: readme_test derives from Operations() and every FlagOnly arg; I mutated the README three ways and all three axes went red.
          round: 8
        - id: BR-39
          disposition: not-addressed
          note: console.go:321 still takes TakeRowDirty for every pane while :333 acts only when active.
          round: 8
      findings:
        - id: BR-40
          severity: Important
          title: the sweep that closed this family filed Console under "Pure entities" while the row itself calls it a thin IO shell
          detail: |-
            6th in this family on this issue. Do NOT just move the row. The five named sites ARE fixed, so
            the rule this instance adds is about the sweep: a row added to a CLASSIFIED table asserts that
            section's classification, so a fix for table drift must classify what it files or it ships a new
            instance of the family it closed. Measured: plan:86 puts `Console` under `### Pure entities`,
            while the row's own text reads "thin IO shell", console.go holds a mutex, starts two goroutines,
            drives hostty.Host and ptychild.Child, and console_test.go cannot run without hostty.FakeHost
            plus ptychild.NewFakeChild -- it belongs in `### Integration points` beside hostty.Host. Evidence
            the section has stopped classifying rather than slipped once: termcmd.restoreTerminal, a method
            that writes escapes to a terminal, is in the same Pure table. The enumeration the rule implies is
            one pass over BOTH tables checking each row's section against the code's actual IO surface. Also
            in scope: Task 2.1's "Files: Modify cmd/internal/couchcore/runner.go" (plan:250) -- runner.go is
            untouched in this window; TerminalHandle landed in ptyrunner.go.
          family: plan-table-drift
          round: 8
        - id: BR-41
          severity: Important
          title: the README enumeration's one hand-written exemption points at atlas/couch.md, which does not document publish-description either
          detail: |-
            4th in this family. Do NOT just add a line to atlas/couch.md. BR-38's enumeration landed and
            works -- I mutated the README three ways and all three axes went red -- so the rule this instance
            adds is: an exemption from a derived-documentation check must itself be derived; an exemption may
            redirect to another ENFORCED document, never to a sentence. Measured: readme_test.go:33-37 skips
            publish-description on the stated grounds that "it belongs in atlas/couch.md rather than in the
            operator's README", and `grep -rn publish atlas/ docs/ README.md` returns zero hits for it. So
            couch's only agent-facing verb is documented nowhere while a test comment asserts it is, and the
            enumeration reads as complete coverage. Five-line fix: assert the exempted operation appears in
            atlas/couch.md, making the exemption a redirection so every declared operation is documented
            somewhere enforced. Fold in an adjacent weakness while there: `strings.Contains(doc, "couch "+
            op.Name)` is prefix-ambiguous -- a future `couch stop-all` line would satisfy the check for
            `couch stop`.
          family: docs-lag-the-surface
          round: 8
      boundary: M2
      blocked: false
    - "n": 9
      timestamp: "2026-08-23T22:32:43-07:00"
      agent: codex
      findings:
        - id: BR-42
          severity: Critical
          title: Input after a hotkey can be routed to the focus being left
          detail: console.go:573-600 queues the hotkey but processes rest before the Run goroutine acknowledges the focus change. This is the 2nd finding in family chunking-invariance; enumerate both input framers and test every legal read split rather than fixing one byte grouping.
          family: chunking-invariance
          round: 9
        - id: BR-43
          severity: Critical
          title: Production bypasses NewPanelModel and loses parked or updated tree metadata
          detail: panel.go:75-97 implements the planned TreeSummary model, but console.go:652-676 rebuilds rows only from hosted panes and pane.desc is never populated. This is the 3rd finding in family dead-field-and-leaked-consumer; make all production refreshes consume the shared summary source and join routing IDs afterward.
          family: dead-field-and-leaked-consumer
          round: 9
        - id: BR-44
          severity: Critical
          title: Panel action keys make valid typeahead prefixes unreachable
          detail: console.go:765-807 consumes s, x, n, d and digits as commands when the query is empty, so names and descriptions beginning with those characters cannot be searched. Separate command input from filter text and test every reserved prefix.
          family: input-namespace-collision
          round: 9
        - id: BR-45
          severity: Critical
          title: The Core concepts table misclassifies and misstates M3 entities
          detail: The plan's Pure entities table classifies Console as PURE despite its IO dependencies, lists nonexistent Home, and omits DecodePanelKeys. This is the 5th finding in family plan-table-drift; enforce entity existence and kind classification across the complete table, then append a Revisions entry.
          family: plan-table-drift
          round: 9
        - id: BR-46
          severity: Important
          title: M3 completion evidence omits the smoke work explicitly carried into this milestone
          detail: issue lines 1020-1030 carry the composed kill -9 reattach and real nvim in-and-out checks to M3, while lines 1272-1279 record only the two-actor panel smoke. This is the 3rd finding in family undelivered-plan-step; enumerate every M3 and carried checkbox and supply evidence for each.
          family: undelivered-plan-step
          round: 9
        - id: BR-47
          severity: Important
          title: README does not document the M3 focus ladder or panel controls
          detail: README.md:280-290 documents only ctrl-space interception, not child-to-root-to-panel navigation or the keys a user types in the panel. This is the 5th finding in family docs-lag-the-surface; establish an enforced documentation home for every user-entered key surface.
          family: docs-lag-the-surface
          round: 9
      boundary: M3
      blocked: true
    - "n": 10
      timestamp: "2026-08-24T16:02:55-07:00"
      agent: codex
      dispose:
        - id: BR-42
          disposition: not-addressed
          note: Same-read suffix ordering is pinned, but keys.go:110-119 still forwards a Kitty hotkey split immediately after ESC; the required every-split enumeration is absent.
          round: 10
        - id: BR-43
          disposition: addressed
          note: Production injects Couch.Summarize(nil), rebuildPanel starts from NewPanelModel, and wiring, parked-row, refresh, and pure target-join tests pin the path.
          round: 10
        - id: BR-44
          disposition: addressed
          note: Commands now require the colon namespace, and tests pin typeahead beginning with s, x, n, d, and digits plus namespaced direct jumps.
          round: 10
        - id: BR-45
          disposition: not-addressed
          note: The named table cells changed, but no claimed entity/kind audit exists and the plan still says Console holds no policy despite console.go:779-945.
          round: 10
        - id: BR-46
          disposition: addressed
          note: The tracker records the composed kill -9 reattach evidence and a Spec/Plan revision explains why the literal layout3-only nvim step does not apply to the shipped layout2 configuration.
          round: 10
        - id: BR-47
          disposition: addressed
          note: README documents the full focus ladder and namespaced controls, with coverage derived from couchtty.PanelControls.
          round: 10
      findings:
        - id: BR-48
          severity: Minor
          title: The captured M3 review transcript fails git diff --check
          detail: workshop/plans/000146-couch-tty-switching-and-attach-m3-review.md contains extensive added trailing whitespace and space-before-tab lines; clean the generated artifact or its writer.
          family: generated-artifact-hygiene
          round: 10
      boundary: M3
      blocked: true
    - "n": 11
      timestamp: "2026-08-24T16:21:31-07:00"
      agent: codex
      dispose:
        - id: BR-42
          disposition: addressed
          note: The exact ESC-split Kitty hotkey is exercised through Console, while interceptor and panel-key tests enumerate the recognized sequences at every listed byte split.
          round: 11
        - id: BR-43
          disposition: addressed
          note: Prior disposition remains valid; production consumes the summary-derived panel and the wiring is exercised.
          round: 11
        - id: BR-44
          disposition: addressed
          note: Prior disposition remains valid; printable prefixes remain typeahead and panel commands use the colon namespace.
          round: 11
        - id: BR-45
          disposition: not-addressed
          note: The rows were manually corrected, but no executable table-contract audit exists, so reverting a path, symbol, deletion, or classification cannot fail a test; Console's source comment also still claims it holds no policy.
          round: 11
        - id: BR-46
          disposition: addressed
          note: Prior disposition remains valid under the recorded layout2 Spec and Plan revision.
          round: 11
        - id: BR-47
          disposition: addressed
          note: Prior disposition remains valid; README coverage derives from the shared panel-control inventory.
          round: 11
        - id: BR-48
          disposition: addressed
          note: git diff --check passes for both the complete M3 range and the supplied review window.
          round: 11
      findings:
        - id: BR-49
          severity: Minor
          title: The timed stdin framer adds another background worker that Console cannot cancel or join
          detail: 'This is the 3rd finding in family signal-goroutine-outlives-close. console.go:613-631 starts a reader blocked on an arbitrary io.Reader, while Run does not own an ordered stop, host close, and worker join. State the class rule: a console lifecycle has one teardown owner that cancels or closes every input/signal seam and joins every worker before returning.'
          family: signal-goroutine-outlives-close
          round: 11
      boundary: M3
      blocked: true
    - "n": 12
      timestamp: "2026-08-24T16:38:07-07:00"
      agent: codex
      dispose:
        - id: BR-45
          disposition: not-addressed
          note: The contract validates present rows but cannot detect an omitted row; deleting the complete PanelKey and DecodePanelKeys row still passes, so the recurring plan-table-drift class is not pinned.
          round: 12
        - id: BR-49
          disposition: withdrawn
          note: The revised event-loop design removes the additional nested stdin worker this finding identified; full ownership and joining of the remaining lifecycle workers stays explicitly scheduled for M4.
          round: 12
      boundary: M3
      blocked: true
    - "n": 13
      timestamp: "2026-08-24T16:46:21-07:00"
      agent: codex
      dispose:
        - id: BR-45
          disposition: addressed
          note: The corrected table and bidirectional inventory contract cover all three original drift modes, and isolated mutations made each mode fail.
          round: 13
      boundary: M3
      blocked: false
    - "n": 14
      timestamp: "2026-08-25T07:29:38-07:00"
      agent: ""
      no_cap: true
      blocked: true
      protocol_error: 'review did not run: dispatch codex (owner bin "/Users/xianxu/workspace/ariadne/bin" prepended to PATH=/Users/xianxu/.local/bin:/opt/homebrew/bin:/opt/homebrew/sbin:/usr/local/bin:/System/Cryptexes/App/usr/bin:/usr/bin:/bin:/usr/sbin:/sbin:/var/run/com.apple.security.cryptexd/codex.system/bootstrap/usr/local/bin:/var/run/com.apple.security.cryptexd/codex.system/bootstrap/usr/bin:/var/run/com.apple.security.cryptexd/codex.system/bootstrap/usr/appleinternal/bin:/pkg/env/global/bin:/Library/TeX/texbin:/Users/xianxu/workspace/ariadne/bin:/opt/homebrew/lib/node_modules/@openai/codex/node_modules/@openai/codex-darwin-arm64/vendor/aarch64-apple-darwin/codex-path:/Users/xianxu/.codex/tmp/arg0/codex-arg0Y9MH27:/Users/xianxu/workspace/pair/bin:/Library/Java/JavaVirtualMachines/jdk-25.jdk/Contents/Home/bin:/Users/xianxu/.local/share/bob/nvim-bin:/Users/xianxu/.luarocks/bin:/opt/homebrew/opt/lua@5.4/bin:/Users/xianxu/.local/bin:/opt/homebrew/opt/ruby/bin:/Users/xianxu:.mix/escripts:/Users/xianxu/bin:/usr/local/sbin:/Applications/Ghostty.app/Contents/MacOS:/opt/homebrew/opt/fzf/bin): fork/exec /opt/homebrew/bin/codex: argument list too long'
    - "n": 15
      timestamp: "2026-08-25T13:25:54-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Over-long sequences now retain framing state instead of rescanning payload as text; long OSC/ST split regressions and latch-aware fuzz coverage pin the false-bell case.
          round: 15
        - id: BR-7
          disposition: not-addressed
          note: termcmd/run.go still cites deleted queries.go, and ptychild/replay.go still says output returns through deleted readPTY.
          round: 15
        - id: BR-8
          disposition: not-addressed
          note: Child.SetSink still writes c.sink without synchronization or a fake-only guard while the real pump reads it.
          round: 15
        - id: BR-9
          disposition: not-addressed
          note: newTab still snapshots after Start launches the pump, leaving a window where one chunk is replayed and later written from the live queue.
          round: 15
        - id: BR-19
          disposition: addressed
          note: frame routes DCS, APC, PM, and SOS through string-terminator framing, with tests covering false bells and nested tmux controls.
          round: 15
        - id: BR-20
          disposition: not-addressed
          note: Both production repaint sites now call Child.Replay, but replacing the helper call with the former StripQueries(Snapshot()) composition leaves the behavioral test green, so helper adoption itself is not pinned.
          round: 15
        - id: BR-24
          disposition: not-addressed
          note: consoleRunnerFor and the path default are pinned, but the production consoleRunner link is not; terminal-path tests bypass it by calling consoleRunnerFor directly.
          round: 15
        - id: BR-32
          disposition: not-addressed
          note: ChildRows(0) still returns zero despite the documented invariant, and the boundary test still begins at one.
          round: 15
        - id: BR-33
          disposition: not-addressed
          note: Normal Run teardown is fixed, but MakeRaw failure returns before teardown and RunWithRuntime constructs OSHost before domain errors that can return without ever running or closing the console.
          round: 15
        - id: BR-34
          disposition: not-addressed
          note: Screen still says ED every form, ops.go says the default is applied at the CLI, and keys.go still describes Console as policy-free glue.
          round: 15
        - id: BR-35
          disposition: not-addressed
          note: Run can still select the exit event before draining already-queued final chunks, and doneBeforeExit is still sampled after the write that may end the live child.
          round: 15
        - id: BR-39
          disposition: addressed
          note: Inactive row damage is retained in pane.rowDirty and a consumer-ordered regression test observes it before switching.
          round: 15
        - id: BR-40
          disposition: addressed
          note: Console and terminal-writing entities are classified as Integration, and the bidirectional Core concepts contract pins missing, extra, relocated, and misclassified rows.
          round: 15
        - id: BR-41
          disposition: not-addressed
          note: The atlas redirection is now enforced, but the operator-operation check remains prefix-ambiguous because it uses strings.Contains on couch plus the operation name.
          round: 15
      findings:
        - id: BR-50
          severity: Important
          title: Enter treats a live actor hosted elsewhere as a parked worktree and dispatches start
          detail: Couch.Summarize(nil) includes globally registered live actors, but BindTargets adds a Target only for children hosted by this Console. console.go:968 checks Target alone, so a Live row with no local Target takes the parked-start branch and reaches the occupied-tree refusal. Model local-live, remote-live, and parked as distinct states; only parked should start, while remote-live should explain that attachment requires pair#147. Add a composed test because existing fixtures cover only local-live and parked rows (ARCH-PURPOSE).
          family: routing-capability-conflated-with-liveness
          round: 15
      blocked: true
    - "n": 16
      timestamp: "2026-08-25T13:43:46-07:00"
      agent: codex
      dispose:
        - id: BR-7
          disposition: not-addressed
          note: The four stale references are gone, but no test or static contract fails if a deleted identifier is reintroduced into these comments.
          round: 16
        - id: BR-8
          disposition: not-addressed
          note: SetSink and both sink reads now use Child.mu, but no concurrent regression exercises SetSink against the real pump; reverting the synchronization is not test-pinned.
          round: 16
        - id: BR-9
          disposition: addressed
          note: newTab clears with a nil replay before releasing startup output, and TestTerminalMuxNewTabPrintsStartupOutputOnce pins the duplicate-output behavior.
          round: 16
        - id: BR-20
          disposition: not-addressed
          note: Both production repaint paths now call Child.Replay, but existing tests only pin equivalent query-stripping behavior and remain green if the decision is hand-composed again.
          round: 16
        - id: BR-24
          disposition: addressed
          note: TestConsoleRunnerDetectsARealPTY drives the actual consoleRunner terminal-detection link, while TestStartDefaultsItsPathToCwd pins the operation-level dot default through a spawn.
          round: 16
        - id: BR-32
          disposition: addressed
          note: ChildRows(0) now returns 1 and reserve_test.go covers the exact zero-row boundary.
          round: 16
        - id: BR-33
          disposition: not-addressed
          note: Run now owns ordered normal/signal teardown with regressions, but MakeRaw failure and pre-Run spawn refusal still return after OSHost construction without calling host.Close.
          round: 16
        - id: BR-34
          disposition: not-addressed
          note: Most named comments were corrected, but atlas/couch.md still says the console no longer “hands the child its own stdio and block”; the comment sweep is also unpinned.
          round: 16
        - id: BR-35
          disposition: not-addressed
          note: Exit ordering and doneBeforeExit are fixed and tested, but the separate couchcore waitUntilTrue polling implementation remains alongside couchtty's shared waitUpTo helper.
          round: 16
        - id: BR-41
          disposition: addressed
          note: The agent-facing exemption is enforced against atlas/couch.md, and token-aware README matching has a negative longer-operation-prefix regression.
          round: 16
        - id: BR-50
          disposition: addressed
          note: Panel rows retain global Live independently of local Target, and the composed Console regression proves Enter on a remote-live row reports deferred attachment without dispatching start.
          round: 16
      blocked: false
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

## Round 3 — 2026-08-22T19:08:56-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Resync is genuinely pinned but insufficient - 3 of 10 measured shapes still diverge, and the 4096-byte production path now DROPS a real BEL where whole-feed rings it.
- BR-6 — not-addressed — waitCode still byte-identical at couchcore/runner.go:90 and ptychild/child.go:130; asExitError still a one-line wrapper in both packages.
- BR-7 — not-addressed — All four remain (run.go:1062, run.go:1069, replay.go:37, Makefile.local:73-77); bufferSnapshotLocked's stated m.mu contract is now false as well as stale.
- BR-8 — not-addressed — fake.go:65 still writes c.sink unlocked with no `if c.fake == nil` guard.
- BR-9 — not-addressed — newTab still snapshots at run.go:723 after ptychild.Start has launched the pump.
- BR-15 — addressed — Verified - hostty runs 8 of 9 tests in this sandboxed shell with zero skips (the 9th fails loudly), and swapping teardown's two statements reddens TestTeardownStopsTheWatcherBeforeClosingChildren from the default suite.
- BR-16 — addressed — make test-smoke exists, atlas/index.md carries an entry stating what earns a place in probes/, and cleanup runs before os.Exit(1).
- BR-17 — addressed — The Core-concepts row now describes what Screen answers and points at screen.go instead of restating field names; residual copies in Task 1.3 and the Child bullet noted as a plan recommendation, not re-raised.

### Raised

- **BR-18** [Important] `fake-diverges-from-production` FakeHost panics on a post-Close SetSize where OSHost is inert, and both conformance tests stop at the terminal transition
  2nd in this family. Do NOT just guard SetSize -- the rule is that every lifecycle
  transition a fake exposes must match the real implementation's, and the conformance
  test must drive BOTH past the terminal state rather than up to it. BR-3's class fix
  (TestFakeChildConformsToRealChildLifecycle) had the right shape and stopped one step
  short; TestCloseReleasesResizedConsumers stops at Close too. Measured at HEAD:
  FakeHost.Close closes h.resized under h.mu while SetSize sends outside the lock and
  never consults h.closed, so a post-Close SetSize panics with "send on closed channel"
  while OSHost absorbs a 20-signal SIGWINCH burst inertly. Enumeration this implies -
  Host post-Close is {resize, Write, Size, MakeRaw}, 1 of 4 diverges fatally; Child
  post-Close is {Write, Resize, Signal}, 3 of 3 diverge (fake returns n=1/nil, nil, nil;
  a real child returns "file already closed", an ioctl error, "process already
  finished"). 4 of 7 post-terminal-state stimuli diverge, 1 fatally. FakeHost is the
  double M2's console tests are built on, so a panic there crashes a run instead of
  failing it.

## Round 4 — 2026-08-22T19:28:37-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Round 3 genuinely closed the systematic break (measured: 10 shapes x 4 chunkings, 400 randomized iterations, zero divergences). One boundary residual remains -- skipTerminator drops the ESC it is commented to hold, so an ST split across a read swallows the next real bell. 1 of 70,550 cut positions.
- BR-6 — addressed — Verified - procutil.WaitCode is the single source, both byte-identical copies and both errors.As wrappers deleted; couchcore/errors.go and ptychild/errors.go no longer exist.
- BR-7 — not-addressed — Makefile.local fixed; run.go:1062, run.go:1069 and replay.go:37 remain, and bufferSnapshotLocked's stated m.mu contract is now false as well as stale.
- BR-8 — not-addressed — fake.go:70 still writes c.sink unlocked with no `if c.fake == nil` guard while the pump reads it.
- BR-9 — not-addressed — newTab still snapshots at run.go:723 after ptychild.Start launched the pump at :707; both interleavings traced, the duplicate-write one is real.
- BR-18 — addressed — Verified by revert with mutate+compile+traverse confirmed - deleting FakeHost.SetSize's `if h.closed` guard panics TestHostsAgreeAfterClose at fake.go:72. Both conformance tests now drive past the terminal transition.

### Raised

- **BR-19** [Minor] `framing-omits-sequence-class` frame treats DCS/APC/PM/SOS as two-byte escapes, so their payloads are scanned as plain text and a BEL inside one falsely rings
  screen.go:275-278's `default: return 2, true` covers ESC-c style two-byte escapes only;
  the string-terminated classes fall through it. Measured: "\x1bP+q616263\x07\x1b\\" ->
  bell=true; "\x1b_Ga=T,f=100;PAYLOAD\x07\x1b\\" -> bell=true; PM and SOS likewise; and
  "\x1bPtmux;\x1b[?1049h\x1b\\" -> alt=true, region=true. 4 of the 5 string-terminated
  classes leak, OSC being the one covered. Contradicts TakeBell's own doc ("BEL is only
  counted outside a sequence") and atlas/architecture.md:462, which states it as a design
  property. Chunk-invariant, so not the chunking family -- the rule is that the framing
  must cover every sequence class the invariant is claimed over. Reachability today is low
  (kitty-graphics APC and XTGETTCAP DCS carry base64/hex; sixel's alphabet excludes BEL),
  so this is a stated invariant narrower than claimed rather than a live false positive.
  Fix is one `case 'P', '_', '^', 'X':` routing to the same ansi.OSCEnd scan.
- **BR-20** [Minor] `needless-indirection` Child.Replay has zero production callers while the one repaint site reimplements it
  child.go:170-173 documents itself as "what a repaint should write... Prefer Replay for
  repainting a screen"; redrawTab (termcmd/run.go:1062-1065) hand-composes HomeAndClear +
  StripQueries(Snapshot()), which is Replay() plus the clear. grep confirms Replay() is
  called only from child_test.go. 2nd in this family - the rule is that a helper naming a
  decision must be the only place that decision is made; if the sole production caller
  reimplements it, either the caller adopts it or the helper is deleted. Prevalence: 1
  helper, 0 production callers, 1 site reimplementing. Worth closing now because M3 Task
  3.3's contract spells the same expression out again for couch's attach path, which would
  make it two divergent repaint policies rather than one.

## Round 5 — 2026-08-23T08:39:37-07:00 (claude) — BLOCKED

### Raised

- **BR-21** [Critical] `guard-reads-wrong-stream-position` MidSequence() describes the newest chunk READ, not the chunk WRITTEN, so the row paint still splices into the child's escape sequences
  ptychild's pump feeds Screen before calling the sink (child.go:113-120) and the console
  drains a 256-deep channel later, so when console.go:182 asks MidSequence() about the chunk
  it just wrote, the answer describes a later chunk. Reproduced in a scratch copy by holding
  the console inside host.Write while the child completes the sequence -- emitted stream was
  "\x1b[2J\x1b[38;2;76" + "\x1b[1;23r\x1b7\x1b[24;1H\x1b[2K[brain]\x1b8" + ";82;88mCOLOURED",
  byte-for-byte the nvim corruption this commit claims to fix. console_test.go:191-196
  describes this exact ordering and concludes the bug "does not reproduce"; the shipped test
  avoids the window rather than covering it. Second mode, measured: Screen.Pending() is
  len(pending), which is 0 while skipping != skipNone, so MidSequence() returns false in the
  middle of an over-64KiB OSC 52 -- the everyday clipboard case this repo already documents.
  Fix: compute pending-ness over the bytes the console has EMITTED (a per-pane ptychild.Screen
  fed as chunks are written, which is what console_live_test.go's splicedPaint already does),
  fold in the skipping state, and serialise host writes -- onChunk, watchResize and onHotkey
  all write to c.host today with no ordering between them.
- **BR-22** [Critical] `prefix-parks-a-complete-key` a lone ESC keystroke is held indefinitely by the Interceptor and then delivered glued to the next key as a meta prefix
  sequenceAt (keys.go:127-141) classifies a bare 0x1b as seqPartial because it is a real prefix
  of "\x1b[200~", so Feed holds it (keys.go:93) and returns nothing. Measured: Feed("\x1b") ->
  before=""; the following Feed("i") -> before="\x1bi", which a child reads as Alt+i. Driven
  end to end through the console, an ESC keystroke never reaches the child at all, and held
  bytes are never flushed at teardown. ESC is interrupt in Claude Code and mode-switch in nvim
  -- the Spec cites both. Caveat stated honestly: once zellij's Kitty protocol is up the
  terminal likely sends "\x1b[27u", which falls through as seqNone and is forwarded, which is
  plausibly why the smoke did not see it; the legacy encoding still governs before the child
  enables KKP (pair's config picker), after it disables it, and for M3's panel. Fix: hold only
  from two bytes on, or flush a held run when a read ends without extending the match, and add
  the negative test -- FuzzInterceptorFeed bounds output length only, so it cannot find this.
  workbenchshortcut.FindChord, the shape this copies, holds nothing.
- **BR-23** [Critical] `swallowed-seam-error` couch start off a tty spawns and registers the child, gives it a 0-row pty, then exits 1 with no output
  consoleRunner (run.go:157-170) type-asserts stdin/stdout to *os.File and never asks whether
  they are terminals, while dimCodes at run.go:305 in the same file already uses
  term.IsTerminal. With stdin a pipe -- any script, cron, or agent shell, including #148's
  advisor -- host.Size() fails so Console keeps size{0,0}, ChildSize() returns 0x0 and the
  child is started on a zero-row pty; then MakeRaw() fails and console.go:103-106 discards the
  wrapped error and returns 1. Measured through RunWithRuntime: exit code 1, stderr "". The
  actor record persists, the child is hung up at process exit, and nothing mentions
  --no-console. Fix: gate the console on term.IsTerminal for both fds and fall back to the
  announced ExecRunner path; report the MakeRaw/Size errors instead of collapsing them to 1.
- **BR-24** [Important] `fix-not-pinned-by-failing-test` the milestone's central wiring is unpinned -- disabling the console entirely leaves the whole suite green
  3rd in this family on this issue. Do NOT fix these two instances -- the rule, extending round
  2's, is that the mutation you must run is removal of the WIRING, not of a helper: every
  behaviour a boundary claims, including its central one, needs a default-suite test that goes
  red when that behaviour is deleted. Measured, both green under mutation: (a) forcing
  consoleRunner to always return (nil, ExecRunner) -- couch start is no longer the console --
  leaves ./cmd/internal/couchcmd and ./cmd/internal/couchtty ok, though run_test.go:34-40
  claims the branch "is observable in the rendered output"; only the --no-console side asserts
  anything. (b) deleting the path == "" -> "." default in ops.go:78-81, which is Decision 1's
  "cd brain and couch start is what makes brain home", leaves the tree green.
- **BR-25** [Important] `probe-hygiene` probes/zellijpark ships with no make target and no atlas entry, one round after that rule was written
  3rd in this family. Do NOT just add a target for zellijpark. The rule was already stated in
  round 2 -- a committed probe is a first-class artifact: a make target, an atlas/ entry, and
  self-cleanup -- and the very next probe skipped two of three. make test-smoke
  (Makefile.local:50-51) runs only termsmoke; atlas/index.md:17-21 names only termsmoke; and
  zellijpark's output is quoted as the evidence that corrected workshop/projects/couch.md's
  park-vs-kill claim. Class fix: make the target enumerate probes/ so a new probe cannot skip
  it, and state the convention in the atlas entry rather than listing members.
- **BR-26** [Important] `plan-table-drift` the plan's Decision 11 and three Core-concepts rows now contradict the code, with no Revisions entry
  3rd in this family on this issue (4th counting pair#145 BR-41). Do NOT correct the rows. The
  round-2 rule was "stop maintaining a second copy of a code shape in prose"; this round shows
  it is owed by DECISIONS and TASK CONTRACTS too -- a plan statement asserting external
  behaviour must be measured before it is written, and a boundary that reverses a Decision
  writes the Revisions entry in the same window. Measured drift, none of it recorded: Decision
  11 and Task 2.6a still say --layout2 is removed and that resume "refuses any third argv
  element (launcher/args.go:104)", which is false and which the code plus couch_test.go:588
  now pin the opposite of; StatusModel/RenderStatusRow are declared at couchtty/statusrow.go,
  a file that does not exist (they are in reserve.go); couchcore.TerminalHandle is declared at
  couchcore/runner.go but lives in ptyrunner.go, and Task 2.1 still specifies Terminal() as an
  interface where the code deliberately returns *ptychild.Child. Task 1.3:199 and Task 2.5:296
  still say MarginsDirty, flagged as a residual at M1 round 4 and since consumed.
- **BR-27** [Important] `dead-field-and-leaked-consumer` three members added this window have zero writers -- FakeRunner.Sink, FakeRunner.Emit, and StatusActor.Bell
  2nd in this family. Do NOT delete just one. The rule: a fake's surface, and a model field,
  must be exercised by the flow it exists for; a member added for symmetry with production and
  set at zero call sites is decoration that reads as coverage. Measured: FakeRunner.Sink
  (runner_fake.go:48) is never assigned anywhere in the tree -- production uses PtyRunner and
  testRT.NewCouchWith discards the runner; FakeRunner.Emit (runner_fake.go:228) has no callers,
  left over from the snapshot-content conformance design the file's own comment says was
  abandoned; StatusActor.Bell is hardcoded false at console.go:207, so the "*" marker
  reserve_test.go:58 pins can never appear in production.
- **BR-28** [Important] `undelivered-plan-step` Task 2.3's ctrl-space audit of claude and nvim was never recorded in the issue Log
  The plan makes it a deliverable with an explicit clause -- check claude and nvim for a
  ctrl-space binding, record the result in the issue Log, and if something rides on it say how
  a literal ctrl-space reaches a child. The Log mentions "the ctrl-space audit" only
  retrospectively at :750; its result appears nowhere. Not ceremony: in Vim, <C-Space> maps to
  <C-@>, which in insert mode is i_CTRL-@, and couch now shadows the key in both encodings
  with no documented escape hatch. An unrecorded audit is an unperformed audit.
- **BR-29** [Important] `unrecoverable-silent-drop` Deliver drops child output on a full buffer, justified by a repaint-from-ring that does not exist at this boundary
  console.go:76-83's default case drops the chunk and the comment says "the ring still has it,
  so the next repaint is correct". grep for Replay/Snapshot over couchtty's non-test files is
  empty: M2 has no repaint-from-ring, so a dropped chunk is gone from the screen permanently,
  and a chunk dropped mid-sequence leaves the host terminal corrupted with nothing to resync
  it. Either block with a bounded wait, or latch a "this pane lost bytes" flag that forces a
  full repaint once M3 lands the replay -- and correct the comment now, since it is a forward
  reference presented as a present-tense guarantee.
- **BR-30** [Important] `derived-id-not-unique` the forced resume tag is the tree's basename, so two different trees resume one zellij session
  couch.go:107 uses launcher.DefaultTag, which is NormalizeDisplayComponent(filepath.Base(path))
  (tag.go:25-31). Any two trees sharing a basename -- a git worktree, a clone under a different
  parent, foo.bar vs foo_bar which both normalise to foo_bar -- derive the same tag, so couch
  start on the second attaches to the first's session while the registry records an actor
  against the second tree. That breaks the correspondence between couch's one-agent-per-TREE
  key and the session the operator lands in. TestSpawnDerivesTheTagFromTheTreeNotTheCwd pins
  derivation; nothing pins uniqueness. Cheap mitigation until #149: suffix a short hash of the
  full tree path.
- **BR-31** [Important] `docs-lag-the-surface` README still describes couch as a spawner, and atlas/architecture.md still describes Screen as reporting region-lost
  README.md:260-265 says couch "registers agent sessions one-per-worktree and can spawn them";
  couch start now owns the operator's terminal for its lifetime, intercepts ctrl-space,
  reserves a row, and has a new --no-console flag -- user-facing surface a reader types.
  atlas/architecture.md:458-460 still describes Screen as reporting "region-lost edges
  (DECSTBM, RIS, or an alt-screen transition)", omitting the ERASE case that is the entire
  discovery of this milestone, and does not mention MidSequence, new public surface on
  ptychild.Child that the console's correctness depends on. atlas/couch.md itself is updated
  well; these two are the same window's other doc sites.
- **BR-32** [Minor] `uncovered-negative-assertion` ChildRows(0) returns 0 while its doc says "It never returns zero", and no test covers the boundary case
  2nd in this family. reserve.go:21-26 returns hostRows for hostRows <= 1, so ChildRows(0) is
  0; reserve_test.go:31 tests only ChildRows(1). The rule: an invariant asserted in a doc
  comment needs a test at its boundary case, or the comment is the lie. Reachable in
  production via the non-tty path (see the swallowed-seam-error finding), where it produces a
  zero-row pty.
- **BR-33** [Minor] `signal-goroutine-outlives-close` Console.Run never calls Stop() or host.Close(), so the resize watcher and the SIGWINCH registration outlive the console
  2nd in this family. console.go:102-130 defers restore() and release() but nothing stops the
  console's own goroutines or closes the host. The rule from M1's BR-2 applies unchanged:
  teardown is explicit and ordered, not emergent -- here from process exit rather than from
  defer LIFO. Latent while there is one console per process; ptyrunner.go:88's own comment
  anticipates #147 putting more than one in.
- **BR-34** [Minor] `stale-comment-reference` several comments overstate or misplace what the code does
  2nd in this family; low value individually, listed together. screen.go's case 'J' says "ED,
  every form" but \x1b[?2J (DECSED) returns early at the private-introducer branch;
  Screen.Pending's doc says "Exported for the tests that pin the bound" and now has a
  production consumer; ops.go:65 says the path default is applied "at the CLI" when it is
  applied in couchcore's Invoke; atlas/couch.md:32 reads "hands the child its own stdio and
  block".
- **BR-35** [Minor] `test-harness-races` the live conformance scenario and Run's exit select are both racy by construction
  conformance_live_test.go:245-249 reads doneBeforeExit AFTER the write that makes the child
  exit and then Fatalfs if it is true. console.go's select between c.chunks and exited is a
  coin flip on exit, so the child's final output is likely dropped before release() clears the
  row. Also test-side: vtscreen_test.go redefines min, shadowing the builtin, and
  waitFor/waitLong/waitUntilTrue are three near-identical polling helpers across two packages.

## Round 6 — 2026-08-23T09:05:24-07:00 (claude) — BLOCKED

### Disposed

- BR-21 — not-addressed — hostScan is right and pinned, but "serialise host writes" was skipped; applyLayout's Reserve write splices deterministically on SIGWINCH, and the hotkey path reproduces with no injected delay.
- BR-22 — not-addressed — the discriminator covers only a sole-byte read; Feed("abc\x1b")+Feed("i") still yields "\x1bi", Feed("\x1b\x1b") holds one ESC, and held is still never flushed.
- BR-23 — not-addressed — the no-terminal fallback is in and pinned; Console.Run still returns 1 silently on MakeRaw error, and the gate reads only the input fd.
- BR-24 — not-addressed — both new pins t.Skipf on pty.Open in the documented environment, so the disable-the-console mutation is still green; the path default is still unpinned.
- BR-25 — not-addressed — Makefile.local hardcodes a second go run line and atlas/index.md lists the new member; neither is the enumeration the class fix asked for.
- BR-26 — not-addressed — the Revisions entry landed for Decision 11 but asserts a table sweep that did not happen; five stale sites remain unchanged.
- BR-27 — addressed — Sink/Emit deleted, Bell wired and pinned (deletion check red); noting only that the !isActive guard is unreachable until M3 attaches a second pane.
- BR-28 — addressed — the audit and what it missed are both in the issue Log now.
- BR-29 — addressed — Deliver blocks and yields to stop; reverting to drop reddens TestConsoleDoesNotDropChildOutputUnderBurst.
- BR-30 — withdrawn — verified independently: scope.Key plus liveOwnedByOther means a live collision bumps the suffix; the mechanism I claimed does not hold.
- BR-31 — addressed — README and atlas/architecture.md both reconciled, MidSequence and the erase case included.
- BR-32 — not-addressed — reserve.go:21-26 and reserve_test.go:31 unchanged; ChildRows(0) is still 0 under a doc saying it never returns zero.
- BR-33 — not-addressed — Console.Run still defers only restore and release; no Stop(), no host.Close().
- BR-34 — not-addressed — Pending's doc updated; the "ED, every form" comment, ops.go:65's "at the CLI", and atlas/couch.md's "stdio and block" are unchanged, and run_test.go:38-42 still claims the console branch is observable in the rendered output.
- BR-35 — not-addressed — min renamed to minInt; doneBeforeExit is still read after the write that ends the child, Run's exit select is unchanged, and waitUntilTrue still duplicates waitFor across packages.

### Raised

- **BR-36** [Important] `undelivered-plan-step` Task 2.7's operator smoke is partly unrecorded and two of its items are carried to M3 with no Revisions entry
  2nd in this family. Do NOT just record the missing observations. The rule, which BR-28's
  instance fix did not reach: a milestone's Plan line enumerates that milestone's deliverables,
  so moving one to a later milestone is a scope event and is written as a plan `## Revisions`
  entry plus an amended issue `## Plan` line in the same window -- a Log paragraph is where a
  deferral goes to be forgotten. Measured: the issue `## Plan` M2 line still reads "Smoke step 1
  (one real pair + claude child, resize, nvim in and out, reattach across a kill -9) lands here",
  and plan Task 2.7 still lists the kill -9 reattach and the park-vs-kill determination through
  the full couch stop path, while the Log (2026-08-23) carries both to M3. Of Task 2.7's seven
  recorded-observation items the Log records three (row survives pair startup, ctrl-space
  intercepted, layout2); resize reflow, the row while claude streams, nvim in-and-out (the
  margin-reset case Decision 4 rests on) and "quitting restores the terminal" appear nowhere.
  atlas/couch.md:70-72 nonetheless claims the section is "confirmed by operator smoke on the
  full Ghostty -> couch -> pair -> zellij -> claude stack".
- **BR-37** [Important] `docs-lag-the-surface` atlas/couch.md still tells the reader the console asks Child.MidSequence(), a method this round deleted, on the stream BR-21 proved wrong
  2nd in this family, and a regression introduced by the fix commit rather than a leftover. Do
  NOT just edit the line. The rule BR-31's instance fix did not reach: the commit that changes a
  public surface updates every doc that names it, and the cheap enforcement is that an
  identifier named in atlas/ must be greppable in the tree -- run that grep at the boundary
  instead of re-reading the prose. Measured: `grep -rn MidSequence atlas/ cmd/internal/ptychild/`
  returns atlas/couch.md:69 "the console therefore asks `Child.MidSequence()` and defers", while
  `Child.MidSequence` was removed in 5975f10 and the console now frames its own written stream
  via `Console.hostScan`. The doc therefore teaches the exact mistake the review caught -- ask
  the child -- which is worse than being merely stale. atlas/architecture.md:461 has the same
  surface described correctly, so the two atlas files now disagree.

## Round 7 — 2026-08-23T09:22:30-07:00 (claude) — BLOCKED

### Disposed

- BR-21 — addressed — Run is the sole host writer and hostScan is fed child bytes only; re-adding applyLayout's write reddens TestConsoleNeverSplicesFromAnyPath 3/3, and removing the writeOwn gate reddens 3 tests.
- BR-22 — addressed — discriminator keys on partial length; measured "abc\x1b"+"i" -> "abc\x1b" then "i", "\x1b\x1b" both forwarded, split CSI-u still fires; removal reddens 4 tests.
- BR-23 — addressed — both fds gated, WantsConsole pure and pinned unconditionally, MakeRaw error reported and pinned; the both-fds composition itself is unpinned and folds into BR-24's seam.
- BR-24 — not-addressed — measured at HEAD: forcing consoleRunner to (nil, ExecRunner) leaves couchcmd and couchtty ok, and deleting the path default leaves the tree green -- TestStartDefaultsItsPathToCwd asserts on ArgSpec.Required, not on the default.
- BR-25 — addressed — make test-smoke enumerates probes/*/ and atlas/index.md states the convention; termsmoke defaults bin to ./bin/pair so dropping the argv is safe.
- BR-26 — not-addressed — 0 of the 5 named sites changed; only the prose bullets moved, while the round-2 Revisions entry asserts the table rows did -- the second consecutive entry claiming a sweep it did not perform.
- BR-32 — not-addressed — reserve.go:21-26 and reserve_test.go:28 unchanged; ChildRows(0) is still 0 under a doc saying it never returns zero.
- BR-33 — not-addressed — Run still defers only restore and release; with Deliver now blocking, returning via the exited case leaves the child's pump permanently blocked because stop is never closed.
- BR-34 — not-addressed — screen.go:277 "ED, every form", ops.go:64 "at the CLI" and atlas/couch.md:32 "stdio and block" unchanged; run_test.go:40 still claims the console branch is observable in the rendered output, measured false again; new -- consoleRunner's doc comment now runs into WantsConsole's.
- BR-35 — not-addressed — doneBeforeExit is still read after the write that ends the child, Run's exit select is unchanged, and waitUntilTrue still duplicates waitFor -- four pollers across three packages now.
- BR-36 — not-addressed — the plan Revisions entry and the Log carry landed, but the issue Plan M2 line still says the kill -9 reattach lands at M2, and resize reflow / row-while-claude-streams / nvim in-and-out are still unrecorded by name.
- BR-37 — addressed — atlas/couch.md no longer names Child.MidSequence; I ran the class check -- every backticked identifier in atlas/couch.md resolves in cmd/.

### Raised

- **BR-38** [Important] `docs-lag-the-surface` README documents the pty and --no-console but not ctrl-space or the path default, and the atlas-identifier check has no README counterpart
  3rd in this family. Do NOT just add two README lines. BR-37's class fix landed for atlas/ --
  I ran the grep and every identifier in atlas/couch.md resolves in cmd/ -- but it has no
  counterpart for README's TYPED surface, so the same class recurred at the site the
  enumeration does not cover. Measured: couch start now claims ctrl-space globally from every
  child in both encodings (legacy NUL and CSI-u), and its <repo> argument is now optional
  defaulting to "." -- Decision 1's entire mechanism. README.md:261-267 documents the pty, the
  reserved row and --no-console and neither of those; it still shows `couch start <repo>` as
  though the argument were required. An operator whose ctrl-space stops reaching their editor
  has no documented explanation. Rule to enforce at the boundary: enumerate the flags,
  keybindings and argument defaults the window added and confirm each appears in README, the
  same way atlas identifiers are now grepped.
- **BR-39** [Minor] `latch-consumed-by-wrong-consumer` onChunk consumes TakeRowDirty for every pane but acts on it only for the active one, so an inactive pane's dirty-row latch is thrown away
  console.go:320 reads rowDirty for every chunk of every pane; console.go:333 acts only
  `if rowDirty && isActive`. TakeRowDirty is a latch that clears on read, so for an inactive
  pane the event is consumed and discarded -- by the time the operator switches to it, the
  console has no record that its row needs re-asserting. The bell path immediately above gets
  this right (it stores into p.bell). Unreachable in M2 production, which attaches exactly one
  pane; reachable the moment M3 attaches a second. Fix: store into a per-pane dirty flag and
  consume it at switch time, mirroring how p.bell is handled.

## Round 8 — 2026-08-23T09:42:21-07:00 (claude) — passed

### Disposed

- BR-24 — not-addressed — consoleRunnerFor and the path default are both pinned now (both mutations red), but gutting consoleRunner itself is still green, as is dropping the outFile half of the tty gate.
- BR-26 — addressed — all five named sites measured changed and the Revisions entry enumerates them individually; the sweep introduced one new instance, raised separately.
- BR-32 — not-addressed — unchanged, and the doc's second clause is false too -- on a 1-row host PaintRow still draws the row, so "simply not drawn" is wrong at hostRows==1 as well as ChildRows(0)==0.
- BR-33 — not-addressed — unchanged, plus a new site -- NewOSHost registers SIGWINCH and starts watch() inside consoleRunnerFor, before Invoke, so a refused couch start leaks both without Run ever being called.
- BR-34 — not-addressed — three of four sites unchanged; run_test.go:41's claim measured false again; go doc WantsConsole confirmed to print consoleRunner's comment; new -- render's StartResult branch still says "this milestone has no pty".
- BR-35 — not-addressed — unchanged -- doneBeforeExit still read after the write that ends the child, Run's exit select unchanged, four pollers across three packages.
- BR-36 — addressed — item-by-item smoke record, issue Plan M2 line amended, plan Revisions entry, carries stated with reasons and one item explicitly not claimed; only atlas/couch.md's smoke sentence still over-reaches.
- BR-38 — addressed — readme_test derives from Operations() and every FlagOnly arg; I mutated the README three ways and all three axes went red.
- BR-39 — not-addressed — console.go:321 still takes TakeRowDirty for every pane while :333 acts only when active.

### Raised

- **BR-40** [Important] `plan-table-drift` the sweep that closed this family filed Console under "Pure entities" while the row itself calls it a thin IO shell
  6th in this family on this issue. Do NOT just move the row. The five named sites ARE fixed, so
  the rule this instance adds is about the sweep: a row added to a CLASSIFIED table asserts that
  section's classification, so a fix for table drift must classify what it files or it ships a new
  instance of the family it closed. Measured: plan:86 puts `Console` under `### Pure entities`,
  while the row's own text reads "thin IO shell", console.go holds a mutex, starts two goroutines,
  drives hostty.Host and ptychild.Child, and console_test.go cannot run without hostty.FakeHost
  plus ptychild.NewFakeChild -- it belongs in `### Integration points` beside hostty.Host. Evidence
  the section has stopped classifying rather than slipped once: termcmd.restoreTerminal, a method
  that writes escapes to a terminal, is in the same Pure table. The enumeration the rule implies is
  one pass over BOTH tables checking each row's section against the code's actual IO surface. Also
  in scope: Task 2.1's "Files: Modify cmd/internal/couchcore/runner.go" (plan:250) -- runner.go is
  untouched in this window; TerminalHandle landed in ptyrunner.go.
- **BR-41** [Important] `docs-lag-the-surface` the README enumeration's one hand-written exemption points at atlas/couch.md, which does not document publish-description either
  4th in this family. Do NOT just add a line to atlas/couch.md. BR-38's enumeration landed and
  works -- I mutated the README three ways and all three axes went red -- so the rule this instance
  adds is: an exemption from a derived-documentation check must itself be derived; an exemption may
  redirect to another ENFORCED document, never to a sentence. Measured: readme_test.go:33-37 skips
  publish-description on the stated grounds that "it belongs in atlas/couch.md rather than in the
  operator's README", and `grep -rn publish atlas/ docs/ README.md` returns zero hits for it. So
  couch's only agent-facing verb is documented nowhere while a test comment asserts it is, and the
  enumeration reads as complete coverage. Five-line fix: assert the exempted operation appears in
  atlas/couch.md, making the exemption a redirection so every declared operation is documented
  somewhere enforced. Fold in an adjacent weakness while there: `strings.Contains(doc, "couch "+
  op.Name)` is prefix-ambiguous -- a future `couch stop-all` line would satisfy the check for
  `couch stop`.

## Round 9 — 2026-08-23T22:32:43-07:00 (codex) — BLOCKED

### Raised

- **BR-42** [Critical] `chunking-invariance` Input after a hotkey can be routed to the focus being left
  console.go:573-600 queues the hotkey but processes rest before the Run goroutine acknowledges the focus change. This is the 2nd finding in family chunking-invariance; enumerate both input framers and test every legal read split rather than fixing one byte grouping.
- **BR-43** [Critical] `dead-field-and-leaked-consumer` Production bypasses NewPanelModel and loses parked or updated tree metadata
  panel.go:75-97 implements the planned TreeSummary model, but console.go:652-676 rebuilds rows only from hosted panes and pane.desc is never populated. This is the 3rd finding in family dead-field-and-leaked-consumer; make all production refreshes consume the shared summary source and join routing IDs afterward.
- **BR-44** [Critical] `input-namespace-collision` Panel action keys make valid typeahead prefixes unreachable
  console.go:765-807 consumes s, x, n, d and digits as commands when the query is empty, so names and descriptions beginning with those characters cannot be searched. Separate command input from filter text and test every reserved prefix.
- **BR-45** [Critical] `plan-table-drift` The Core concepts table misclassifies and misstates M3 entities
  The plan's Pure entities table classifies Console as PURE despite its IO dependencies, lists nonexistent Home, and omits DecodePanelKeys. This is the 5th finding in family plan-table-drift; enforce entity existence and kind classification across the complete table, then append a Revisions entry.
- **BR-46** [Important] `undelivered-plan-step` M3 completion evidence omits the smoke work explicitly carried into this milestone
  issue lines 1020-1030 carry the composed kill -9 reattach and real nvim in-and-out checks to M3, while lines 1272-1279 record only the two-actor panel smoke. This is the 3rd finding in family undelivered-plan-step; enumerate every M3 and carried checkbox and supply evidence for each.
- **BR-47** [Important] `docs-lag-the-surface` README does not document the M3 focus ladder or panel controls
  README.md:280-290 documents only ctrl-space interception, not child-to-root-to-panel navigation or the keys a user types in the panel. This is the 5th finding in family docs-lag-the-surface; establish an enforced documentation home for every user-entered key surface.

## Round 10 — 2026-08-24T16:02:55-07:00 (codex) — BLOCKED

### Disposed

- BR-42 — not-addressed — Same-read suffix ordering is pinned, but keys.go:110-119 still forwards a Kitty hotkey split immediately after ESC; the required every-split enumeration is absent.
- BR-43 — addressed — Production injects Couch.Summarize(nil), rebuildPanel starts from NewPanelModel, and wiring, parked-row, refresh, and pure target-join tests pin the path.
- BR-44 — addressed — Commands now require the colon namespace, and tests pin typeahead beginning with s, x, n, d, and digits plus namespaced direct jumps.
- BR-45 — not-addressed — The named table cells changed, but no claimed entity/kind audit exists and the plan still says Console holds no policy despite console.go:779-945.
- BR-46 — addressed — The tracker records the composed kill -9 reattach evidence and a Spec/Plan revision explains why the literal layout3-only nvim step does not apply to the shipped layout2 configuration.
- BR-47 — addressed — README documents the full focus ladder and namespaced controls, with coverage derived from couchtty.PanelControls.

### Raised

- **BR-48** [Minor] `generated-artifact-hygiene` The captured M3 review transcript fails git diff --check
  workshop/plans/000146-couch-tty-switching-and-attach-m3-review.md contains extensive added trailing whitespace and space-before-tab lines; clean the generated artifact or its writer.

## Round 11 — 2026-08-24T16:21:31-07:00 (codex) — BLOCKED

### Disposed

- BR-42 — addressed — The exact ESC-split Kitty hotkey is exercised through Console, while interceptor and panel-key tests enumerate the recognized sequences at every listed byte split.
- BR-43 — addressed — Prior disposition remains valid; production consumes the summary-derived panel and the wiring is exercised.
- BR-44 — addressed — Prior disposition remains valid; printable prefixes remain typeahead and panel commands use the colon namespace.
- BR-45 — not-addressed — The rows were manually corrected, but no executable table-contract audit exists, so reverting a path, symbol, deletion, or classification cannot fail a test; Console's source comment also still claims it holds no policy.
- BR-46 — addressed — Prior disposition remains valid under the recorded layout2 Spec and Plan revision.
- BR-47 — addressed — Prior disposition remains valid; README coverage derives from the shared panel-control inventory.
- BR-48 — addressed — git diff --check passes for both the complete M3 range and the supplied review window.

### Raised

- **BR-49** [Minor] `signal-goroutine-outlives-close` The timed stdin framer adds another background worker that Console cannot cancel or join
  This is the 3rd finding in family signal-goroutine-outlives-close. console.go:613-631 starts a reader blocked on an arbitrary io.Reader, while Run does not own an ordered stop, host close, and worker join. State the class rule: a console lifecycle has one teardown owner that cancels or closes every input/signal seam and joins every worker before returning.

## Round 12 — 2026-08-24T16:38:07-07:00 (codex) — BLOCKED

### Disposed

- BR-45 — not-addressed — The contract validates present rows but cannot detect an omitted row; deleting the complete PanelKey and DecodePanelKeys row still passes, so the recurring plan-table-drift class is not pinned.
- BR-49 — withdrawn — The revised event-loop design removes the additional nested stdin worker this finding identified; full ownership and joining of the remaining lifecycle workers stays explicitly scheduled for M4.

## Round 13 — 2026-08-24T16:46:21-07:00 (codex) — passed

### Disposed

- BR-45 — addressed — The corrected table and bidirectional inventory contract cover all three original drift modes, and isolated mutations made each mode fail.

## Round 14 — 2026-08-25T07:29:38-07:00 () — BLOCKED

**Protocol error:** review did not run: dispatch codex (owner bin "/Users/xianxu/workspace/ariadne/bin" prepended to PATH=/Users/xianxu/.local/bin:/opt/homebrew/bin:/opt/homebrew/sbin:/usr/local/bin:/System/Cryptexes/App/usr/bin:/usr/bin:/bin:/usr/sbin:/sbin:/var/run/com.apple.security.cryptexd/codex.system/bootstrap/usr/local/bin:/var/run/com.apple.security.cryptexd/codex.system/bootstrap/usr/bin:/var/run/com.apple.security.cryptexd/codex.system/bootstrap/usr/appleinternal/bin:/pkg/env/global/bin:/Library/TeX/texbin:/Users/xianxu/workspace/ariadne/bin:/opt/homebrew/lib/node_modules/@openai/codex/node_modules/@openai/codex-darwin-arm64/vendor/aarch64-apple-darwin/codex-path:/Users/xianxu/.codex/tmp/arg0/codex-arg0Y9MH27:/Users/xianxu/workspace/pair/bin:/Library/Java/JavaVirtualMachines/jdk-25.jdk/Contents/Home/bin:/Users/xianxu/.local/share/bob/nvim-bin:/Users/xianxu/.luarocks/bin:/opt/homebrew/opt/lua@5.4/bin:/Users/xianxu/.local/bin:/opt/homebrew/opt/ruby/bin:/Users/xianxu:.mix/escripts:/Users/xianxu/bin:/usr/local/sbin:/Applications/Ghostty.app/Contents/MacOS:/opt/homebrew/opt/fzf/bin): fork/exec /opt/homebrew/bin/codex: argument list too long — this round contributed no findings.

## Round 15 — 2026-08-25T13:25:54-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Over-long sequences now retain framing state instead of rescanning payload as text; long OSC/ST split regressions and latch-aware fuzz coverage pin the false-bell case.
- BR-7 — not-addressed — termcmd/run.go still cites deleted queries.go, and ptychild/replay.go still says output returns through deleted readPTY.
- BR-8 — not-addressed — Child.SetSink still writes c.sink without synchronization or a fake-only guard while the real pump reads it.
- BR-9 — not-addressed — newTab still snapshots after Start launches the pump, leaving a window where one chunk is replayed and later written from the live queue.
- BR-19 — addressed — frame routes DCS, APC, PM, and SOS through string-terminator framing, with tests covering false bells and nested tmux controls.
- BR-20 — not-addressed — Both production repaint sites now call Child.Replay, but replacing the helper call with the former StripQueries(Snapshot()) composition leaves the behavioral test green, so helper adoption itself is not pinned.
- BR-24 — not-addressed — consoleRunnerFor and the path default are pinned, but the production consoleRunner link is not; terminal-path tests bypass it by calling consoleRunnerFor directly.
- BR-32 — not-addressed — ChildRows(0) still returns zero despite the documented invariant, and the boundary test still begins at one.
- BR-33 — not-addressed — Normal Run teardown is fixed, but MakeRaw failure returns before teardown and RunWithRuntime constructs OSHost before domain errors that can return without ever running or closing the console.
- BR-34 — not-addressed — Screen still says ED every form, ops.go says the default is applied at the CLI, and keys.go still describes Console as policy-free glue.
- BR-35 — not-addressed — Run can still select the exit event before draining already-queued final chunks, and doneBeforeExit is still sampled after the write that may end the live child.
- BR-39 — addressed — Inactive row damage is retained in pane.rowDirty and a consumer-ordered regression test observes it before switching.
- BR-40 — addressed — Console and terminal-writing entities are classified as Integration, and the bidirectional Core concepts contract pins missing, extra, relocated, and misclassified rows.
- BR-41 — not-addressed — The atlas redirection is now enforced, but the operator-operation check remains prefix-ambiguous because it uses strings.Contains on couch plus the operation name.

### Raised

- **BR-50** [Important] `routing-capability-conflated-with-liveness` Enter treats a live actor hosted elsewhere as a parked worktree and dispatches start
  Couch.Summarize(nil) includes globally registered live actors, but BindTargets adds a Target only for children hosted by this Console. console.go:968 checks Target alone, so a Live row with no local Target takes the parked-start branch and reaches the occupied-tree refusal. Model local-live, remote-live, and parked as distinct states; only parked should start, while remote-live should explain that attachment requires pair#147. Add a composed test because existing fixtures cover only local-live and parked rows (ARCH-PURPOSE).

## Round 16 — 2026-08-25T13:43:46-07:00 (codex) — passed

### Disposed

- BR-7 — not-addressed — The four stale references are gone, but no test or static contract fails if a deleted identifier is reintroduced into these comments.
- BR-8 — not-addressed — SetSink and both sink reads now use Child.mu, but no concurrent regression exercises SetSink against the real pump; reverting the synchronization is not test-pinned.
- BR-9 — addressed — newTab clears with a nil replay before releasing startup output, and TestTerminalMuxNewTabPrintsStartupOutputOnce pins the duplicate-output behavior.
- BR-20 — not-addressed — Both production repaint paths now call Child.Replay, but existing tests only pin equivalent query-stripping behavior and remain green if the decision is hand-composed again.
- BR-24 — addressed — TestConsoleRunnerDetectsARealPTY drives the actual consoleRunner terminal-detection link, while TestStartDefaultsItsPathToCwd pins the operation-level dot default through a spawn.
- BR-32 — addressed — ChildRows(0) now returns 1 and reserve_test.go covers the exact zero-row boundary.
- BR-33 — not-addressed — Run now owns ordered normal/signal teardown with regressions, but MakeRaw failure and pre-Run spawn refusal still return after OSHost construction without calling host.Close.
- BR-34 — not-addressed — Most named comments were corrected, but atlas/couch.md still says the console no longer “hands the child its own stdio and block”; the comment sweep is also unpinned.
- BR-35 — not-addressed — Exit ordering and doneBeforeExit are fixed and tested, but the separate couchcore waitUntilTrue polling implementation remains alongside couchtty's shared waitUpTo helper.
- BR-41 — addressed — The agent-facing exemption is enforced against atlas/couch.md, and token-aware README matching has a negative longer-operation-prefix regression.
- BR-50 — addressed — Panel rows retain global Live independently of local Target, and the composed Console regression proves Enter on a remote-live row reports deferred attachment without dispatching start.

## Open findings

- **BR-7** [Minor] `stale-comment-reference` comments still cite queries.go, appendBuffer, tab.buffer and readPTY, all deleted by this diff
- **BR-8** [Minor] `unsynchronised-shared-state` Child.SetSink writes c.sink unlocked with no fake-only guard while the pump reads it
- **BR-20** [Minor] `needless-indirection` Child.Replay has zero production callers while the one repaint site reimplements it
- **BR-33** [Minor] `signal-goroutine-outlives-close` Console.Run never calls Stop() or host.Close(), so the resize watcher and the SIGWINCH registration outlive the console
- **BR-34** [Minor] `stale-comment-reference` several comments overstate or misplace what the code does
- **BR-35** [Minor] `test-harness-races` the live conformance scenario and Run's exit select are both racy by construction
