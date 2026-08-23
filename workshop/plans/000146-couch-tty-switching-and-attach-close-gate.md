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

## Open findings

- **BR-1** [Important] `chunking-invariance` Screen raises a false bell for any sequence longer than maxPending split across two reads
- **BR-7** [Minor] `stale-comment-reference` comments still cite queries.go, appendBuffer, tab.buffer and readPTY, all deleted by this diff
- **BR-8** [Minor] `unsynchronised-shared-state` Child.SetSink writes c.sink unlocked with no fake-only guard while the pump reads it
- **BR-9** [Minor] `replay-duplicates-live-output` newTab widens the window where a chunk is both replayed and written live
- **BR-19** [Minor] `framing-omits-sequence-class` frame treats DCS/APC/PM/SOS as two-byte escapes, so their payloads are scanned as plain text and a BEL inside one falsely rings
- **BR-20** [Minor] `needless-indirection` Child.Replay has zero production callers while the one repaint site reimplements it
- **BR-21** [Critical] `guard-reads-wrong-stream-position` MidSequence() describes the newest chunk READ, not the chunk WRITTEN, so the row paint still splices into the child's escape sequences
- **BR-22** [Critical] `prefix-parks-a-complete-key` a lone ESC keystroke is held indefinitely by the Interceptor and then delivered glued to the next key as a meta prefix
- **BR-23** [Critical] `swallowed-seam-error` couch start off a tty spawns and registers the child, gives it a 0-row pty, then exits 1 with no output
- **BR-24** [Important] `fix-not-pinned-by-failing-test` the milestone's central wiring is unpinned -- disabling the console entirely leaves the whole suite green
- **BR-25** [Important] `probe-hygiene` probes/zellijpark ships with no make target and no atlas entry, one round after that rule was written
- **BR-26** [Important] `plan-table-drift` the plan's Decision 11 and three Core-concepts rows now contradict the code, with no Revisions entry
- **BR-27** [Important] `dead-field-and-leaked-consumer` three members added this window have zero writers -- FakeRunner.Sink, FakeRunner.Emit, and StatusActor.Bell
- **BR-28** [Important] `undelivered-plan-step` Task 2.3's ctrl-space audit of claude and nvim was never recorded in the issue Log
- **BR-29** [Important] `unrecoverable-silent-drop` Deliver drops child output on a full buffer, justified by a repaint-from-ring that does not exist at this boundary
- **BR-30** [Important] `derived-id-not-unique` the forced resume tag is the tree's basename, so two different trees resume one zellij session
- **BR-31** [Important] `docs-lag-the-surface` README still describes couch as a spawner, and atlas/architecture.md still describes Screen as reporting region-lost
- **BR-32** [Minor] `uncovered-negative-assertion` ChildRows(0) returns 0 while its doc says "It never returns zero", and no test covers the boundary case
- **BR-33** [Minor] `signal-goroutine-outlives-close` Console.Run never calls Stop() or host.Close(), so the resize watcher and the SIGWINCH registration outlive the console
- **BR-34** [Minor] `stale-comment-reference` several comments overstate or misplace what the code does
- **BR-35** [Minor] `test-harness-races` the live conformance scenario and Run's exit select are both racy by construction
