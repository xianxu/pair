---
gate: boundary-review
issue: 199
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-07T16:30:11-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Important
          title: The rename-pane consumer set is asserted from memory; the two real title matchers are never named
          detail: |-
            The plan enumerates consumers as "run.go:229, #118, #123", but run.go:229 is
            the RegisterTerminalPane comment, not a title reader. The actual matchers are
            launcher/layoutflow.go:62 (HasPrefix(pane.Title, "[terminal") — the bracket is
            part of the format M3.6 removes) and workbenchshortcut/shortcut.go:186
            (HasPrefix(title, "terminal ")). M3.6's assertion only checks that a rename
            still fires and is not the packed string, so it passes while the format stops
            matching. Derive the set by grep and assert against RoleForPane /
            ClassifyLiveLayout directly.
            (carried from plan-quality PQ-1, deferred to the boundary review)
          family: consumer-set-not-derived
          round: 1
        - id: BR-2
          severity: Important
          title: M1.5's "no couch test edited" is unsatisfiable, and the concept contract binds the moved symbols
          detail: |-
            couchtty/reserve_test.go:15-58 calls Reserve/ChildRows/PaintRow/Release
            directly, so the package will not compile after M1.4 deletes them.
            core_concepts_contract_test.go:45 pins those symbols and resolves their
            declared path from workshop/history/plans/000146-...-plan.md:88
            (cmd/internal/couchtty/reserve.go), so TestCoreConceptsContract fails until
            that row is marked deleted and #199 is registered in conceptPlans. Restate the
            acceptance as "no behavioral couch test edited" and list both files as M1 work.
            (carried from plan-quality PQ-2, deferred to the boundary review)
          family: consumer-set-not-derived
          round: 1
        - id: BR-3
          severity: Important
          title: The strip repaint trigger names ptychild.Child.TakeRowDirty, which is always false in termcmd
          detail: |-
            Child.outputBatchLocked take-and-clears rowDirty into batch.RowDirty whenever a
            sink is set (ptychild/child.go:176), and termcmd always sets one
            (run.go:660). couch reads ch.batch.RowDirty (couchtty/console.go:1135). ptyChunk
            (run.go:600-603) has no RowDirty field and the sink drops it at run.go:661, so
            M3 needs a struct field, not a method call. Failure is silent: the strip
            vanishes on nvim's startup clear and never returns — a Done-when bullet.
            (carried from plan-quality PQ-3, deferred to the boundary review)
          family: wrong-seam-named
          round: 1
        - id: BR-4
          severity: Important
          title: Two writers to the pane's tty sit outside the single-writer envelope, and the M2 test cannot see them
          detail: |-
            OSRuntime.RunZellijAction gives the zellij subprocess os.Stdout (run.go:1088)
            with Stderr = os.Stderr (run.go:1103) — a writer on the wheel path
            (run.go:453-463) and the rename path, outside the writer loop and outside the
            paint gate; RunZellijActionQuiet already exists. The resize goroutine
            (run.go:262) becomes a writer in M3 but is outside M2's stated routing scope. A
            goroutine-id wrapper around stdout sees neither, so
            TestOnlyOneGoroutineWritesTheHost stays green over both.
            (carried from plan-quality PQ-4, deferred to the boundary review)
          family: envelope-claim-unenforced
          round: 1
        - id: BR-5
          severity: Important
          title: M2 restates two of couch's three gate rules; the takeover reset is the missing one
          detail: |-
            couchtty/console.go:981-988 resets hostScan and clears paintPending on a screen
            takeover, because the old child's partial sequence is no longer on screen to
            corrupt. redrawTab is exactly a takeover (HomeAndClear plus a different child's
            replay), so without the reset the gate frames one child's pending bytes against
            another child's stream — the same class of wrong answer as BR-21.
            (carried from plan-quality PQ-5, deferred to the boundary review)
          family: takeover-resets-framing
          round: 1
        - id: BR-6
          severity: Important
          title: M4 modifies the generated mirror of main-3.kdl and config.kdl rather than their source
          detail: |-
            cmd/internal/runtimebundle/assets/runtime/files/zellij/layouts/main-3.kdl and
            .../zellij/config.kdl are Kind GeneratedMirror
            (artifactpath/manifest.go:472-474), regenerated from zellij/layouts/main-3.kdl
            and zellij/config.kdl by make runtimebundle-generate (Makefile.local:138) and
            checked by artifactpath/coverage_test.go:152-184. The files are byte-identical
            today so the nine line numbers hold, but the edit must land on the source and
            the mirror must be regenerated. termcmd/run_test.go:616 is the precedent for
            reading the root layout from a test.
            (carried from plan-quality PQ-6, deferred to the boundary review)
          family: edit-source-not-mirror
          round: 1
        - id: BR-7
          severity: Important
          title: RenderStrip cannot reuse couchtty's sanitize/truncate — they are unexported in another package
          detail: |-
            The plan's ARCH-SECURE and ARCH-DRY sections both rest on "the same helpers
            couchtty uses", but sanitize and truncate are unexported in package couchtty
            (reserve.go) and the Core-concepts table keeps that file otherwise unchanged.
            As written the implementer duplicates them, which is the second escape-stripping
            policy the plan says it is avoiding. Name the shared home (textwidth already
            owns Width; ansi.Strip is the framing dependency).
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: shared-helper-not-reachable
          round: 1
        - id: BR-8
          severity: Minor
          title: M4's manual acceptance never splits the pane, leaving six of nine borderless sites unverified
          detail: |-
            :138/:139, :165/:166 and :191/:192 are the *-split rungs and only take effect
            after Alt+Shift+d. The Done-when bullet "two right-pane halves each draw their
            own strip" has no step anywhere in the plan, and divider loss in a split is the
            exact failure that caused the 2026-07-27 frameless revert. One keystroke added
            to M4.3.
            (carried from plan-quality PQ-8, deferred to the boundary review)
          family: acceptance-misses-changed-sites
          round: 1
        - id: BR-9
          severity: Minor
          title: The mid-sequence gate test picks one hand-chosen split point over an arbitrary byte stream
          detail: |-
            The plan's own ARCH-ORDER note says a test can split an escape sequence at any
            index; TestPaintDefersMidSequenceAndIsOwed splits one. Parameterize over every
            index of a representative sequence set — the hand-picked case is by construction
            blind to the boundary the author did not think of.
            (carried from plan-quality PQ-9, deferred to the boundary review)
          family: single-interleaving-oracle
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-07T16:30:11-07:00"
      agent: claude
      findings:
        - id: BR-10
          severity: Important
          title: Paint's new 1-row behaviour is an undeclared, unpinned change from the moved PaintRow
          detail: |-
            cmd/internal/hostty/reserve.go:102 guards on !usable() (Rows > 1) where
            PaintRow guarded only hostRows == 0, so a 1-row host now draws nothing
            instead of drawing over the child's only line. This is the right change --
            it is the residual half of pair#146's BR-32 -- but the milestone claims a
            MOVE and nothing pins it. Mutation-verified: restoring the old height
            behaviour keeps ./cmd/internal/hostty and ./cmd/internal/couchtty fully
            green. TestDegenerateHeightsNeverProduceAZeroRowChild covers ChildRows and
            Reserve at rows 0 and 1 but never Paint. Add the Paint assertion to that
            loop and record the deviation in the plan's Revisions.
          family: uncovered-negative-assertion
          round: 2
        - id: BR-11
          severity: Important
          title: Four Core-concepts rows claim `new` at cmd/internal/termcmd/strip.go, which does not exist
          detail: |-
            plan lines 114-117 declare TabChip, StripModel, RenderedStrip and
            RenderStrip as `new` at a file that is not in the tree; they are M3 work.
            core_concepts_contract_test.go documents the convention this breaks --
            unshipped rows carry `planned` so the status column doubles as the build
            tracker -- and this milestone is what registered #199 in conceptPlans. The
            contract passes only because the couchtty package filter drops those rows,
            so the guard is silent rather than satisfied. Flip the four rows to
            `planned — M3`.
          family: plan-table-drift
          round: 2
        - id: BR-12
          severity: Important
          title: M1 code landed with the plan-quality gate still blocked and no estimate recorded
          detail: |-
            The plan-gate ledger's only round is `blocked: true` with all nine PQ
            findings open, and the issue's estimate_hours is empty with no Estimate
            block -- so change-code appears not to have cleared. PQ-2 was the only
            M1-scoped finding and its remediation did ship here, but the ledger still
            reports it open, which makes the open set wrong entering M2. PQ-3 in
            particular contradicts the plan's own Integration-points row before M3 is
            written. Re-run the gate so PQ-2 disposes `addressed`, and land the
            estimate. Not a defect in M1's code.
          family: gate-not-cleared-before-code
          round: 2
        - id: BR-13
          severity: Minor
          title: NewReservation has zero production callers; couch constructs the struct directly
          detail: |-
            hostty/reserve.go:49 is the validating constructor, but couchtty/console.go:921
            builds Reservation{} literally. The exported struct makes the door optional
            and the fail-closed methods are what actually protect. Either construct
            through NewReservation at the couch site or drop it and let usable() carry
            the contract alone.
          family: validating-door-bypassed
          round: 2
        - id: BR-14
          severity: Minor
          title: atlas/couch.md's "The reserved row" section still reads as couch-owned mechanism
          detail: |-
            atlas/couch.md:228 describes the reservation with no pointer to
            hostty.Reservation, while architecture.md carries the new note. That
            section is where a reader lands for this subject; add one cross-reference.
          family: atlas-points-at-old-home
          round: 2
        - id: BR-15
          severity: Minor
          title: Paint's doc does not state the caller's obligation to sanitize and clamp its text
          detail: |-
            hostty/reserve.go:97 writes caller text verbatim between SaveCursor and
            RestoreCursor. Faithful to PaintRow, and couchtty.RenderStatusRow sanitizes
            upstream, but the mechanism/policy split just moved that obligation across
            a package boundary and M3 adds a consumer whose input is operator-supplied
            tab names. One doc line at the new door (ARCH-SECURE).
          family: caller-obligation-undocumented
          round: 2
      boundary: M1
      blocked: true
    - "n": 3
      timestamp: "2026-09-07T17:03:57-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Finding 9 records the grep and all three matcher forms; I re-ran it and it reproduces layoutflow.go:56,59,62 and shortcut.go:184-189. Residual restatements at :304 and :391 folded into the new plan-table-drift finding.
          round: 3
        - id: BR-2
          disposition: addressed
          note: 'M1.5 restated in Revisions to behavioural tests; console_live_test.go and vtscreen_test.go diff empty; conceptPlans, conceptInventory and #146''s row all landed and the contract is green.'
          round: 3
        - id: BR-3
          disposition: addressed
          note: Finding 7 names batch.RowDirty in the Sink callback and checks out against child.go:176 and run.go:661. Residual at the Integration-points table (:271) and M3.4/M3.5 folded into the new plan-table-drift finding.
          round: 3
        - id: BR-4
          disposition: not-addressed
          note: The stdout half is derived and routed via RunZellijActionQuiet, but runZellij hardwires cmd.Stderr = os.Stderr at run.go:1104 for BOTH methods, so the subprocess's second descriptor is still unrouted and "M2's envelope covers all five" is still false.
          round: 3
        - id: BR-5
          disposition: addressed
          note: M2's preamble names the takeover reset as couch's third rule and maps it to redrawTab; M2.3 step 1 resets hostScan plus the deferred slot. Verified against console.go:992-995.
          round: 3
        - id: BR-6
          disposition: addressed
          note: M4's Files block now names zellij/layouts/main-3.kdl and zellij/config.kdl as the SOURCE, explains the GeneratedMirror, and adds the regenerate step.
          round: 3
        - id: BR-7
          disposition: addressed
          note: rowtext.Sanitize/Fit is named as the shared home with the "unexported in another package" rationale. Residual at ARCH-SECURE (:372-373) and M3.3 (:560) folded into the new plan-table-drift finding.
          round: 3
        - id: BR-8
          disposition: not-addressed
          note: M4.3 still has no Alt+Shift+d step; the six *-split rungs stay unverified. Minor, carried.
          round: 3
        - id: BR-9
          disposition: not-addressed
          note: TestPaintDefersMidSequenceAndIsOwed still splits at one hand-picked index inside "\x1b[3". Minor, carried.
          round: 3
        - id: BR-10
          disposition: addressed
          note: 'Mutation-verified: reverting Paint''s guard to r.Rows == 0 turns reserve_test.go:66 and :101 red. The plan-Revisions half of the ask is still missing and is listed as a plan revision recommendation.'
          round: 3
        - id: BR-11
          disposition: addressed
          note: All four strip.go rows flipped to "planned — M3" in 692aa11e; the rowtext row carries the same status.
          round: 3
        - id: BR-12
          disposition: addressed
          note: Plan-gate rounds 5 and 6 are blocked:false, PQ-2 disposed addressed in round 2, and estimate_hours 7.52 plus an itemized Estimate block landed. Only PQ-8/PQ-9 remain open, consistent with BR-8/BR-9.
          round: 3
        - id: BR-13
          disposition: not-addressed
          note: NewReservation still has zero production callers; console.go:922 still builds the struct literal. It also validates only the edge, not rows. Minor, carried.
          round: 3
        - id: BR-14
          disposition: not-addressed
          note: atlas/couch.md:228 "The reserved row" still describes the mechanism with no pointer to hostty.Reservation. Minor, carried.
          round: 3
        - id: BR-15
          disposition: not-addressed
          note: hostty/reserve.go:97-108 documents save/restore and the one-row deviation but still states no caller obligation to sanitize or clamp. Minor, carried.
          round: 3
      findings:
        - id: BR-16
          severity: Important
          title: The probe behind finding 5 -- the design's load-bearing measurement -- does not exist anywhere in the repo
          detail: |-
            The plan closes finding 5 with "Probe kept at scratchpad/199-probe/", but there
            is no scratchpad/ on disk, nothing matching *199*probe*, no such path ever added
            in git log --all --diff-filter=A, and no .gitignore entry hiding one. The plan
            calls this "the load-bearing fact" without which M1/M3 "would have been built on
            sand", and atlas/architecture.md:490-493 now states the result as settled fact.
            ARCH-MOCK at-review: behavior we depend on has a one-time manual reading and no
            retained apparatus or conformance check. Land the probe as a tracked package
            (cmd/probes/couchstartrecovery is the repo's own precedent) or correct the plan
            and atlas to say the method is recorded but the apparatus was not kept.
          family: unreproducible-measurement
          round: 3
        - id: BR-17
          severity: Important
          title: Three gate corrections landed at one site each and left seven restatements of the superseded facts in the same file
          detail: |-
            This is the 2nd finding in family plan-table-drift. Do NOT fix these seven sites
            one by one -- the deliverable is the rule and the sweep. Measured prevalence, all
            in workshop/plans/000199-...-plan.md: (1) finding 7 corrects the repaint seam to
            batch.RowDirty, but :271's Integration-points row still names
            ptychild.Child.TakeRowDirty as the wrapped seam and M3.4/M3.5 (:561,:562) still
            say "repaint on TakeRowDirty" -- :271 is what an M3 implementer reads to learn
            which seam to wrap; (2) finding 9 establishes run.go:229 is RegisterTerminalPane
            and not a title reader (confirmed at termcmd/run.go:226-231), but :304 still says
            the #118/#123 consumers "read it (run.go:229)" and :391's ARCH-PURPOSE still
            prints the superseded list AND calls it an enumeration "written out rather than
            left as sweep", which is exactly the shape the plan's own PQ-10 rule forbids;
            (3) the rowtext entry establishes couchtty's sanitize/truncate are unreachable,
            but ARCH-SECURE (:372-373) and M3.3 (:560) still say "exactly as
            couchtty.RenderStatusRow does" / "the same helpers couchtty uses".
            The rule: when a gate finding corrects a fact, the unit of repair is the fact,
            not the sentence the finding quoted -- grep the superseded token across the
            artifact and replace every occurrence in the same round. Checkable form, runnable
            before the next disposition:
            grep -n 'TakeRowDirty\|run\.go:229\|same helpers' workshop/plans/000199-*-plan.md
            must return only the passage documenting the correction itself.
          family: plan-table-drift
          round: 3
        - id: BR-18
          severity: Minor
          title: M1.6 is ticked claiming its grep returns nothing; run as written it returns 27 lines
          detail: |-
            plan :452. Every hit is a _test.go fixture and no production file outside hostty
            emits SetRegion or a DECSTBM sequence, so the invariant M1.6 defends does hold --
            but the stated command contradicts the tick, and the next reader to re-run it
            gets output. Corrected form: append "| grep -v _test.go".
          family: acceptance-command-does-not-hold
          round: 3
      boundary: M1
      blocked: true
    - "n": 4
      timestamp: "2026-09-07T17:21:53-07:00"
      agent: claude
      dispose:
        - id: BR-4
          disposition: not-addressed
          note: Finding 8 now derives six writers, but M2.3's four pieces and M2.3b still cover stdout only; run.go:1103 still hardwires cmd.Stderr = os.Stderr for both methods.
          round: 4
        - id: BR-8
          disposition: not-addressed
          note: M4.3 (plan :601-605) is verbatim unchanged; still no Alt+Shift+d step, and the issue Done-when still requires two halves each drawing a strip.
          round: 4
        - id: BR-9
          disposition: not-addressed
          note: M2.1's TestPaintDefersMidSequenceAndIsOwed still splits one hand-picked index inside "\x1b[3".
          round: 4
        - id: BR-13
          disposition: not-addressed
          note: console.go:922 still builds Reservation{} literally; the round instead added rows validation to NewReservation, so the door and the production path now carry different contracts for the same type.
          round: 4
        - id: BR-14
          disposition: not-addressed
          note: atlas/couch.md:228 is unchanged; only atlas/architecture.md was touched in this window.
          round: 4
        - id: BR-15
          disposition: not-addressed
          note: hostty/reserve.go:107-124 Paint still documents save/restore and the one-row deviation but names no caller obligation to sanitize or clamp.
          round: 4
        - id: BR-16
          disposition: addressed
          note: Probe landed at cmd/probes/zellijscrollregion, self-locating, with artifactpath and .gitignore updated; see N1/N2 for defects in the landed apparatus.
          round: 4
        - id: BR-17
          disposition: not-addressed
          note: The finding's own acceptance grep still returns M3.3 (:568); seven superseded facts remain, including the scratchpad path at :149 in the commit that landed the probe.
          round: 4
        - id: BR-18
          disposition: addressed
          note: M1.6 now carries "| grep -v _test.go | grep -v hostty/"; I ran it and it returns nothing.
          round: 4
      findings:
        - id: BR-19
          severity: Important
          title: The probe's reader goroutine and its verdict share a strings.Builder with no synchronization
          detail: |-
            cmd/probes/zellijscrollregion/main.go:143-153 writes `seen` from a reader
            goroutine while main reads it at :177, :180-181 and :197.
            strings.Builder is not safe for concurrent use; String() exposes the
            backing slice while Write may be growing it, so `go run -race` flags this
            and a torn read yields a confident wrong verdict -- the exact failure class
            the probe's own doc comment says it hit three times. The goroutine also has
            no cancellation path, so its extent is not bounded by main. Guard with a
            mutex, or have the goroutine own the buffer and hand the final string over
            a channel at EOF.
          family: shared-state-unsynchronized
          round: 4
        - id: BR-20
          severity: Important
          title: An unused diagnostic slices the pty stream without a bounds check and can panic before the verdict prints
          detail: |-
            cmd/probes/zellijscrollregion/main.go:200 evaluates
            frame[len(frame)-3000:] with no length guard, so a pty yielding fewer than
            3000 bytes panics on the line before the landmarks are checked. tailOf
            (:86-91) is the safe helper this same file already defines (ARCH-DRY). The
            value is never used in the verdict, and it matches "scroll line 0 " with a
            trailing space against a stream where probe.sh emits "scroll line 0\n" --
            yet its `false` output is quoted as evidence in the plan (:143) behind an
            atlas claim. Delete it or make it a checked assertion.
          family: external-input-assumed-wellformed
          round: 4
        - id: BR-21
          severity: Important
          title: This is the 2nd finding in family atlas-points-at-old-home -- the atlas names one probe home and the code now has two
          detail: |-
            Do NOT fix these sites one at a time; BR-14 is still open from the last
            round, which is the family reporting that the enumeration was never
            written. The rule: an atlas statement about where a class of thing lives
            and what runs it must be re-derived whenever the code adds or moves a home,
            and the atlas records the derivation rather than the answer. Measured
            prevalence, three sites: atlas/index.md:17-22 says probes live in probes/
            and "make test-smoke runs every directory under probes/, so a new probe is
            covered by existing" -- the new probe is under cmd/probes/, which that
            sentence does not reach, and grep -n "smoke\|probes" Makefile returns
            nothing, so the target it rests on does not exist; atlas/couch.md:228
            (BR-14) still presents the reserved row as couch-owned mechanism with no
            pointer to hostty.Reservation.
          family: atlas-points-at-old-home
          round: 4
        - id: BR-22
          severity: Minor
          title: Every os.Exit path in the probe skips its deferred session and temp-file cleanup
          detail: |-
            cmd/probes/zellijscrollregion/main.go registers cleanup at :120, :142 and
            :156, then exits via os.Exit at :178, :190 and :209 -- so an inconclusive
            or errored run leaks a live zellij session with a `sleep 30` pane and a
            temp KDL. Wrap the body in `run() int` and call os.Exit(run()). Same file,
            :186-192: dump-screen's output is discarded (`_ = out`) yet its failure
            aborts the probe, letting an irrelevant call kill a good measurement.
          family: exit-path-drops-cleanup
          round: 4
      boundary: M1
      blocked: true
    - "n": 5
      timestamp: "2026-09-07T17:36:01-07:00"
      agent: claude
      dispose:
        - id: BR-4
          disposition: not-addressed
          note: Finding 8 derives six writers, but M2.3 at HEAD still routes stdout only; run.go:1104 hardwires cmd.Stderr for both methods and the resize goroutine (run.go:261-266) is in no routing step.
          round: 5
        - id: BR-8
          disposition: not-addressed
          note: M4.3 unchanged; no split step anywhere in the plan.
          round: 5
        - id: BR-9
          disposition: not-addressed
          note: M2.1's TestPaintDefersMidSequenceAndIsOwed still splits one hand-chosen index.
          round: 5
        - id: BR-13
          disposition: not-addressed
          note: console.go:921-922 still builds the literal; NewReservation has zero production callers and now carries a stricter contract than the path production takes.
          round: 5
        - id: BR-14
          disposition: not-addressed
          note: atlas/couch.md is not in the review window at all; :228 unchanged.
          round: 5
        - id: BR-15
          disposition: not-addressed
          note: hostty/reserve.go:105-116 still names no sanitize/clamp obligation at the new door.
          round: 5
        - id: BR-17
          disposition: not-addressed
          note: 3rd round. Its own acceptance grep still returns :568; :149 and :380 also stand, plus the old probe home at probes/zellijscrollregion/main.go:124.
          round: 5
        - id: BR-19
          disposition: addressed
          note: syncBuffer guards Write/String/Len with a mutex and the verdict reads one snapshot at main.go:233; the unbounded goroutine extent survives only as BR-22's leak.
          round: 5
        - id: BR-20
          disposition: not-addressed
          note: 'The panic is fixed via tailOf, but the stated remedy was not applied: line000 (main.go:237) is still unused, still trailing-space-matched, still quoted as evidence at plan :143.'
          round: 5
        - id: BR-21
          disposition: not-addressed
          note: 'The probe moved and make test-smoke picks it up, but the enumeration was never written: atlas/index.md:16-22 still records the answer, cmd/probes/couchstartrecovery is still a second home, and BR-14 is untouched. (make test-smoke does exist, at Makefile.local:52.)'
          round: 5
        - id: BR-22
          disposition: not-addressed
          note: os.Exit at main.go:211/223/246 still skips the defers at :148/:170/:189; `_ = out` at :225 unchanged.
          round: 5
      findings:
        - id: BR-23
          severity: Minor
          title: The atlas and the new package doc state two consumers of Reservation; production has one
          detail: |-
            atlas/architecture.md:474-475 says the primitive "is host-half mechanism
            with two consumers -- couch holds the host's bottom row ... and `pair term`
            holds its pane's bottom row for a tab strip", and hostty/reserve.go:13-16
            says the same. `grep -rn "hostty.Reservation" cmd --include='*.go'` finds
            one production consumer, couchtty/console.go; termcmd acquires its row in
            M3. The atlas is defined in AGENTS.md as the current state of the codebase,
            so a planned consumer stated in the present tense is the same drift class
            this issue keeps paying for. Mark it planned-M3 in both places.
          family: doc-states-planned-as-current
          round: 5
        - id: BR-24
          severity: Minor
          title: M1.6's acceptance command aborts in the repo's shell before it checks anything
          detail: |-
            This is the 2nd finding in family `acceptance-command-does-not-hold`
            (BR-18 was the first, same step). Do NOT just re-quote the command --
            state the rule. The rule: an acceptance command recorded in a plan must be
            recorded in a form that runs UNMODIFIED in the repo's default shell, and
            the round that ticks the step pastes the command's actual output rather
            than its expected outcome. Measured: plan :456 writes
            `grep -rn "SetRegion\|\\x1b\[.*r\"" cmd --include=*.go | ...`; under zsh
            the unquoted `--include=*.go` fails with "no matches found" and the
            pipeline never runs. Quoted, it returns 0 lines and the invariant holds --
            which is exactly BR-18's shape again: the invariant was fine, the recorded
            command was not.
          family: acceptance-command-does-not-hold
          round: 5
      boundary: M1
      blocked: false
    - "n": 6
      timestamp: "2026-09-07T21:54:54-07:00"
      agent: claude
      dispose:
        - id: BR-4
          disposition: addressed
          note: 'Verified by mutation, not by the commit message: restoring cmd.Stderr = os.Stderr reddens all four FD subtests; resize now goes through resizeThroughWriter.'
          round: 6
        - id: BR-8
          disposition: not-addressed
          note: M4.3 is unchanged in this window; still no Alt+Shift+d step, and the Done-when still requires two halves each drawing a strip.
          round: 6
        - id: BR-9
          disposition: not-addressed
          note: TestPaintDefersMidSequenceAndIsOwed still splits one hand-chosen index; the gate now exists, so parameterizing is cheap.
          round: 6
      findings:
        - id: BR-25
          severity: Critical
          title: removeTab runs on the writer goroutine and posts to the writer goroutine's own channel, deadlocking the pane
          detail: 'run.go:758 dispatches removeTab on the writer goroutine; removeTab calls setPaneTitle (a blocking zellij subprocess) and then redrawTab (run.go:1081), which since M2 is enqueue -> m.output <- chunk. With the 64-slot buffer full from a second tab''s output, the writer blocks sending to the channel only it drains, and enqueue''s only escape (m.done) never closes. Reproduced with an overlay test: two tabs, close one while the other floods, writer parked permanently and all pane output stops. Fix: apply the takeover inline from removeTab via a helper shared with the takeover case; add the two-tab close-under-load regression test.'
          family: handler-posts-to-own-queue
          round: 6
        - id: BR-26
          severity: Important
          title: Two assert-absent tests pass vacuously — the owed-paint drop and the subprocess stdout arm are unpinned
          detail: 'This is the 2nd finding in family uncovered-negative-assertion, so fix the rule, not the two sites. Measured by overlay mutation: deleting m.owed = nil (run.go:766) leaves the entire M2 suite green because writer_test.go:197 never feeds the boundary-ending chunk that would flush it; cmd.Stdout = os.Stdout also stays green because with no live zellij session the succeeding verb writes nothing to stdout, and the positive control built at writer_test.go:227-228 is never asserted. The rule: an assert-absent is evidence only when the same test first establishes the write would have happened. Sweep the four sites in writer_test.go plus run_test.go:910.'
          family: uncovered-negative-assertion
          round: 6
        - id: BR-27
          severity: Important
          title: The milestone keeps diagnostics off the pane by destroying them — discarded stderr, and a coalescing slot shared with paints
          detail: run.go:1331 passes io.Discard for the subprocess stderr where the plan specifies capture-and-log, so a failing zellij action reduces to bare exit status, including on the non-pane --test-shortcut path that used to print it. run.go:833 routes reportError through the same single owed slot as paints (run.go:809), where a later paint replaces it, redrawTab drops it (run.go:766), and nothing flushes it if the child stops writing. couch avoids this by owing a paintPending bool re-derived by paintNow rather than bytes (ARCH-DRY). M3's per-repaint paint makes the clobber routine.
          family: diagnostic-silently-discarded
          round: 6
        - id: BR-28
          severity: Important
          title: M2.3 steps 3, 4 and 5 describe an implementation the code does not have, with no Revisions entry
          detail: 'This is the 3rd finding in family plan-table-drift, so state the rule rather than patching three lines. Piece 3 names five call sites to reroute that are unchanged (the Runtime was changed instead); piece 5 specifies capture-and-log stderr that is discarded; piece 4 states no exception for the startup term: diagnostics that still write stderr directly. The rule: ticking a step asserts the code matches its text, and a deviation gets a Revisions delta in the same commit — mechanized by adding these token pairs to tests/plan-superseded-facts-test.sh, which already exists for exactly this failure.'
          family: plan-table-drift
          round: 6
        - id: BR-29
          severity: Minor
          title: fakeRuntime.reportedUnused and its reported field are unreachable after the interface change
          detail: run_test.go:941 renames the removed interface method instead of deleting it; nothing calls it and nothing reads fakeRuntime.reported. The live recorder is fakeMux.reported.
          family: dead-test-scaffolding
          round: 6
        - id: BR-30
          severity: Minor
          title: runDecision still receives the pane's stdout, unused, on the input goroutine
          detail: run.go:166 takes stdin and stdout and uses neither; stdout is threaded from runShell through pumpStdin and handleChord. Removing the parameters makes a future write from the input goroutine a compile error rather than a review catch — the same argument the diff makes for the Runtime.
          family: handler-posts-to-own-queue
          round: 6
        - id: BR-31
          severity: Minor
          title: The gate adds a second full parse of every child byte on the output path, undeclared in the envelope
          detail: 'handleChunk calls hostScan.FeedFraming on every chunk while ptychild.Child already parses the same bytes into its own Screen. ARCH-CONSTRAINTS budgets paints but not this. Also worth stating there: a long unterminated OSC holds MidSequence true up to maxPending, stalling any owed paint.'
          family: hot-path-cost-undeclared
          round: 6
      boundary: M2
      blocked: true
    - "n": 7
      timestamp: "2026-09-07T22:12:24-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: not-addressed
          note: M4 is untouched in this window; still no Alt+Shift+d step and the Done-when still requires two right-pane halves each drawing a strip.
          round: 7
        - id: BR-9
          disposition: not-addressed
          note: TestPaintDefersMidSequenceAndIsOwed (writer_test.go:115) still splits one hand-chosen index.
          round: 7
        - id: BR-25
          disposition: not-addressed
          note: No code change; reproduced again with an overlay test - 64 slots filled, writer parked past a 3s timeout. run.go:767 -> 1120 -> 1281 -> enqueue.
          round: 7
        - id: BR-26
          disposition: not-addressed
          note: 'Two sites got controls; the class has five. Measured GREEN under mutation: m.owed = nil, cmd.Stdout = stdout, runZellijCaptured, resizeThroughWriter; plus plan-superseded-facts-test.sh:65 bans a token that never existed in the plan.'
          round: 7
        - id: BR-27
          disposition: not-addressed
          note: 'Coalescing half closed and mutation-pinned. Discard half: reverting runZellijCaptured to io.Discard leaves the suite green, and run.go:461,467,1118,1210 still drop the error, so a failing wheel tick or rename is still completely silent.'
          round: 7
        - id: BR-28
          disposition: addressed
          note: M2.3 pieces rewritten to match the code, Revisions delta present, and the piece-3 guard is mutation-live (restoring the wording reddens plan:515). The second guard pair is inert - folded into BR-26.
          round: 7
        - id: BR-29
          disposition: not-addressed
          note: run_test.go:941 fakeRuntime.reportedUnused still present and still uncalled.
          round: 7
        - id: BR-30
          disposition: not-addressed
          note: run.go:167 runDecision still takes stdin and stdout and uses neither.
          round: 7
        - id: BR-31
          disposition: not-addressed
          note: ARCH-CONSTRAINTS unchanged; the second parse and the unterminated-OSC stall are still undeclared.
          round: 7
      findings:
        - id: BR-32
          severity: Important
          title: The BR-27 fix routes an external process's bytes onto the pane's tty, unsanitized and unbounded
          detail: 'This is the 2nd finding in family external-input-assumed-wellformed, so the deliverable is the rule, not the site. runZellijCaptured (run.go:1396-1414) folds the subprocess''s stderr - or its stdout at run.go:1408 - into the error; reportError (run.go:872) writes err.Error() straight to the pane''s pty. Reachable via handleChord -> runDecision -> RunZellijAction (reported at run.go:441) and splitTerminalDown (run.go:511). Before this commit the error was a bare exec.ExitError, so no external bytes reached the pane. Only the newline is bounded (run.go:1411); length and escape bytes are not, and setPaneTitle''s argument is an operator-controlled tab name zellij''s error text can echo back. Measured: current zellij on a pipe emits no escapes, so this is latent rather than observed. The rule - bytes this process did not produce become safe row text at the boundary where they enter, not at the writer that prints them - covers four enumerable sites: runZellijCaptured''s detail, reportError''s err.Error(), run.go:66''s term: %v, and M3''s tab names, all of which the plan already routes through rowtext.Sanitize/Fit.'
          family: external-input-assumed-wellformed
          round: 7
        - id: BR-33
          severity: Important
          title: The atlas paragraph added in this window was falsified by the next commit in the same window
          detail: 'This is the 4th finding in family plan-table-drift, so state the rule rather than patching the paragraph. atlas/architecture.md:501-503 says the gate is consulted before "any console-originated write" and that such a write "is deferred into a single coalescing slot", and :506-508 that a takeover drops it - false for diagnostics since 34918a3c, which gave them a queue (run.go:818) that survives a takeover (run.go:779-786). The paragraph''s own first sentence lists diagnostics among the writes on the loop, so a reader takes from the atlas the opposite of the decision this milestone was about; :519 has the matching gap for where a failing action''s stderr goes now. The rule: an artifact asserting a design is re-checked against the code in the commit that changes the design, and the check is a test. The existing mechanism is scoped to one plan file and to hand-registered tokens - extend tests/plan-superseded-facts-test.sh to atlas/ and register this pair, noting that registering a pair is not evidence unless it fires (see BR-26).'
          family: plan-table-drift
          round: 7
        - id: BR-34
          severity: Minor
          title: owedDiag is an unbounded queue on an externally-timed path, one function below a comment rejecting Feed for that reason
          detail: 'This is the 2nd finding in family hot-path-cost-undeclared, so the rule: every buffer introduced on a path whose fill rate is set by something outside the process gets a declared bound in ARCH-CONSTRAINTS and a cap in code. run.go:818 appends without limit - one entry per failed refreshRename keystroke while the child sits mid-sequence - and flushes them all at once at the next boundary, while run.go:796-799 explains that Feed was rejected precisely because it would be an unbounded buffer growing behind a terminal that never reads it. Same envelope gap as the undeclared second parse.'
          family: hot-path-cost-undeclared
          round: 7
      boundary: M2
      blocked: true
    - "n": 8
      timestamp: "2026-09-07T22:34:06-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: not-addressed
          note: M4 untouched in this window; M4.3 still has no Alt+Shift+d step while the Done-when still requires two right-pane halves each drawing a strip.
          round: 8
        - id: BR-9
          disposition: not-addressed
          note: writer_test.go:115 still splits one hand-chosen index of one sequence.
          round: 8
        - id: BR-25
          disposition: not-addressed
          note: No code change. Reproduced again at HEAD with an overlay test - 64 slots filled, writer parked past a 3s timeout. run.go:768 -> removeTab:1127 -> redrawTab:1287 -> enqueue:905.
          round: 8
        - id: BR-26
          disposition: not-addressed
          note: Four sites still GREEN under mutation at HEAD - m.owed = nil (run.go:776), runZellijCaptured's detail (:1430 and :1434), resizeThroughWriter (:1156). The stdout arm flipped RED here only because this machine has a live zellij session, which is the ambient-dependence the finding named. plan-superseded-facts-test.sh:65 still registers a token absent from every historical revision of the plan.
          round: 8
        - id: BR-27
          disposition: not-addressed
          note: Capture half landed but is unpinned - returning the bare error leaves the suite green. Log half still absent - run.go:462,468,1125,1217 discard the error, so a failing wheel tick or rename is still completely silent.
          round: 8
        - id: BR-29
          disposition: not-addressed
          note: run_test.go:941 fakeRuntime.reportedUnused still present and uncalled. Production sibling of the same class - terminalMux.stderr is set by newTerminalMux and read by nothing after the reportError change.
          round: 8
        - id: BR-30
          disposition: not-addressed
          note: run.go:168 runDecision still takes stdin and stdout and uses neither.
          round: 8
        - id: BR-31
          disposition: not-addressed
          note: ARCH-CONSTRAINTS unchanged. The gap is wider than stated - the second parse covers every tab's output, not the active tab's.
          round: 8
        - id: BR-32
          disposition: addressed
          note: Mutation-verified rather than taken from the commit - removing rowtext.SanitizeAndFit at run.go:878 fails TestAZellijErrorCannotPutAnEscapeOrAnUnboundedLineOnThePane. The duplicate strip inside runZellijCaptured is unpinned, but the egress covers the pane path.
          round: 8
        - id: BR-33
          disposition: not-addressed
          note: atlas/architecture.md:501-508 still says a console-originated write is deferred into a single coalescing slot and dropped by a takeover, which is false for diagnostics since 34918a3c. The head commit added a further inaccuracy to the same block - :532 states pair term's strip as a current rowtext consumer, which M3 has not built.
          round: 8
        - id: BR-34
          disposition: not-addressed
          note: run.go:819 still appends to owedDiag without a bound, and ARCH-CONSTRAINTS still declares no budget for it.
          round: 8
      findings:
        - id: BR-35
          severity: Critical
          title: The paint gate is fed bytes the terminal never saw, and not fed bytes it did
          detail: This is the 2nd finding in family wrong-seam-named, so the deliverable is the rule, not the two sites. run.go:804 calls hostScan.FeedFraming on EVERY chunk while run.go:808 writes only the active tab's, so a background tab releases or stalls the gate for the foreground stream; run.go:775 resets the scanner to a zero Screen and then writes the takeover's replay unscanned, and ptychild/replay.go:64-68 documents that a replay can end mid-sequence because Ring bisects it. Both reproduced with overlay tests - "visible\x1b[3PAINT" and "\x1b[1;1H\x1b[Jrestored screen\x1b[3PAINT", a paint inside a live CSI, which is the one failure M2 exists to prevent. Unreachable by the existing suite (one tab) and by M2.5 (yes emits no ESC). The rule - a predicate that guards a stream is computed from exactly the bytes that reach that stream, at the point they are written - covers three enumerable sites - FeedFraming moved inside the isActive branch; the takeover carrying its child half separately and feeding it as it writes it; and M3's batch.RowDirty read, which must likewise be scoped to the active tab because a background child's clear never touched the terminal.
          family: wrong-seam-named
          round: 8
        - id: BR-36
          severity: Important
          title: M2.5's manual acceptance cannot enter the gate path it is recorded as accepting
          detail: This is the 3rd finding in family acceptance-command-does-not-hold, so state the rule rather than rewriting one step. plan:546 specifies `yes "aaaa…"` flooding one tab; yes emits no ESC byte, so MidSequence is false for the whole run and the defer/owe path never executes, yet the issue Log reasons that "a redraw issued while yes is mid-escape is precisely the interleaving M2 exists to make safe". The run is real evidence for the single-writer half and none for the gate. The rule - an acceptance step names the observable that proves it entered the path under test, and the observation records that observable rather than that the command ran - applies to M2.5 (needs a child emitting escapes plus a second tab, with the deferral counted), M3.7 and M4.3.
          family: acceptance-command-does-not-hold
          round: 8
        - id: BR-37
          severity: Important
          title: rowtext.Sanitize passes the C1 controls, and it is now the repo's single row-safety strip
          detail: This is the 3rd finding in family external-input-assumed-wellformed, so the rule rather than the site. rowtext.go:37 drops only r < 0x20 and 0x7f; measured, "a\u009b31mred" and "a\u009d0;titlex" survive unchanged, and U+009B/U+009D are the 8-bit CSI and OSC introducers a UTF-8 terminal with C1 handling will act on. ansi.Strip does not frame them either. Inherited from couchtty, but the package doc and atlas/architecture.md:532 now make this the one implementation, and M3 routes operator-typed tab names through it. The rule - the sanitizer's notion of "control" is derived from what a terminal treats as a control (C0, DEL, C1 U+0080-U+009F, and the introducers in both 7- and 8-bit forms) and its test table enumerates those classes rather than the ones a past incident produced. Same edit - couchtty/menu_render.go:625 open-codes Fit(Sanitize(...)) where SanitizeAndFit exists.
          family: external-input-assumed-wellformed
          round: 8
        - id: BR-38
          severity: Minor
          title: couchtty/reserve.go keeps the doc comments for the sanitize and truncate it no longer has
          detail: This is the 3rd finding in family atlas-points-at-old-home, so the rule - an extraction moves the prose with the symbol and leaves neither comment nor stub at the old home. reserve.go:143-152 ends in two paragraphs describing functions now in rowtext, so a reader greps couchtty for sanitize and finds documentation for a function that is not there. Greppable form of the check - a doc comment whose subject identifier no longer resolves in its own file.
          family: atlas-points-at-old-home
          round: 8
      boundary: M2
      blocked: true
---

# Gate ledger — pair#199 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-07T16:30:11-07:00 (sdlc) — passed

### Raised

- **BR-1** [Important] `consumer-set-not-derived` The rename-pane consumer set is asserted from memory; the two real title matchers are never named
  The plan enumerates consumers as "run.go:229, #118, #123", but run.go:229 is
  the RegisterTerminalPane comment, not a title reader. The actual matchers are
  launcher/layoutflow.go:62 (HasPrefix(pane.Title, "[terminal") — the bracket is
  part of the format M3.6 removes) and workbenchshortcut/shortcut.go:186
  (HasPrefix(title, "terminal ")). M3.6's assertion only checks that a rename
  still fires and is not the packed string, so it passes while the format stops
  matching. Derive the set by grep and assert against RoleForPane /
  ClassifyLiveLayout directly.
  (carried from plan-quality PQ-1, deferred to the boundary review)
- **BR-2** [Important] `consumer-set-not-derived` M1.5's "no couch test edited" is unsatisfiable, and the concept contract binds the moved symbols
  couchtty/reserve_test.go:15-58 calls Reserve/ChildRows/PaintRow/Release
  directly, so the package will not compile after M1.4 deletes them.
  core_concepts_contract_test.go:45 pins those symbols and resolves their
  declared path from workshop/history/plans/000146-...-plan.md:88
  (cmd/internal/couchtty/reserve.go), so TestCoreConceptsContract fails until
  that row is marked deleted and #199 is registered in conceptPlans. Restate the
  acceptance as "no behavioral couch test edited" and list both files as M1 work.
  (carried from plan-quality PQ-2, deferred to the boundary review)
- **BR-3** [Important] `wrong-seam-named` The strip repaint trigger names ptychild.Child.TakeRowDirty, which is always false in termcmd
  Child.outputBatchLocked take-and-clears rowDirty into batch.RowDirty whenever a
  sink is set (ptychild/child.go:176), and termcmd always sets one
  (run.go:660). couch reads ch.batch.RowDirty (couchtty/console.go:1135). ptyChunk
  (run.go:600-603) has no RowDirty field and the sink drops it at run.go:661, so
  M3 needs a struct field, not a method call. Failure is silent: the strip
  vanishes on nvim's startup clear and never returns — a Done-when bullet.
  (carried from plan-quality PQ-3, deferred to the boundary review)
- **BR-4** [Important] `envelope-claim-unenforced` Two writers to the pane's tty sit outside the single-writer envelope, and the M2 test cannot see them
  OSRuntime.RunZellijAction gives the zellij subprocess os.Stdout (run.go:1088)
  with Stderr = os.Stderr (run.go:1103) — a writer on the wheel path
  (run.go:453-463) and the rename path, outside the writer loop and outside the
  paint gate; RunZellijActionQuiet already exists. The resize goroutine
  (run.go:262) becomes a writer in M3 but is outside M2's stated routing scope. A
  goroutine-id wrapper around stdout sees neither, so
  TestOnlyOneGoroutineWritesTheHost stays green over both.
  (carried from plan-quality PQ-4, deferred to the boundary review)
- **BR-5** [Important] `takeover-resets-framing` M2 restates two of couch's three gate rules; the takeover reset is the missing one
  couchtty/console.go:981-988 resets hostScan and clears paintPending on a screen
  takeover, because the old child's partial sequence is no longer on screen to
  corrupt. redrawTab is exactly a takeover (HomeAndClear plus a different child's
  replay), so without the reset the gate frames one child's pending bytes against
  another child's stream — the same class of wrong answer as BR-21.
  (carried from plan-quality PQ-5, deferred to the boundary review)
- **BR-6** [Important] `edit-source-not-mirror` M4 modifies the generated mirror of main-3.kdl and config.kdl rather than their source
  cmd/internal/runtimebundle/assets/runtime/files/zellij/layouts/main-3.kdl and
  .../zellij/config.kdl are Kind GeneratedMirror
  (artifactpath/manifest.go:472-474), regenerated from zellij/layouts/main-3.kdl
  and zellij/config.kdl by make runtimebundle-generate (Makefile.local:138) and
  checked by artifactpath/coverage_test.go:152-184. The files are byte-identical
  today so the nine line numbers hold, but the edit must land on the source and
  the mirror must be regenerated. termcmd/run_test.go:616 is the precedent for
  reading the root layout from a test.
  (carried from plan-quality PQ-6, deferred to the boundary review)
- **BR-7** [Important] `shared-helper-not-reachable` RenderStrip cannot reuse couchtty's sanitize/truncate — they are unexported in another package
  The plan's ARCH-SECURE and ARCH-DRY sections both rest on "the same helpers
  couchtty uses", but sanitize and truncate are unexported in package couchtty
  (reserve.go) and the Core-concepts table keeps that file otherwise unchanged.
  As written the implementer duplicates them, which is the second escape-stripping
  policy the plan says it is avoiding. Name the shared home (textwidth already
  owns Width; ansi.Strip is the framing dependency).
  (carried from plan-quality PQ-7, deferred to the boundary review)
- **BR-8** [Minor] `acceptance-misses-changed-sites` M4's manual acceptance never splits the pane, leaving six of nine borderless sites unverified
  :138/:139, :165/:166 and :191/:192 are the *-split rungs and only take effect
  after Alt+Shift+d. The Done-when bullet "two right-pane halves each draw their
  own strip" has no step anywhere in the plan, and divider loss in a split is the
  exact failure that caused the 2026-07-27 frameless revert. One keystroke added
  to M4.3.
  (carried from plan-quality PQ-8, deferred to the boundary review)
- **BR-9** [Minor] `single-interleaving-oracle` The mid-sequence gate test picks one hand-chosen split point over an arbitrary byte stream
  The plan's own ARCH-ORDER note says a test can split an escape sequence at any
  index; TestPaintDefersMidSequenceAndIsOwed splits one. Parameterize over every
  index of a representative sequence set — the hand-picked case is by construction
  blind to the boundary the author did not think of.
  (carried from plan-quality PQ-9, deferred to the boundary review)

## Round 2 — 2026-09-07T16:30:11-07:00 (claude) — BLOCKED

### Raised

- **BR-10** [Important] `uncovered-negative-assertion` Paint's new 1-row behaviour is an undeclared, unpinned change from the moved PaintRow
  cmd/internal/hostty/reserve.go:102 guards on !usable() (Rows > 1) where
  PaintRow guarded only hostRows == 0, so a 1-row host now draws nothing
  instead of drawing over the child's only line. This is the right change --
  it is the residual half of pair#146's BR-32 -- but the milestone claims a
  MOVE and nothing pins it. Mutation-verified: restoring the old height
  behaviour keeps ./cmd/internal/hostty and ./cmd/internal/couchtty fully
  green. TestDegenerateHeightsNeverProduceAZeroRowChild covers ChildRows and
  Reserve at rows 0 and 1 but never Paint. Add the Paint assertion to that
  loop and record the deviation in the plan's Revisions.
- **BR-11** [Important] `plan-table-drift` Four Core-concepts rows claim `new` at cmd/internal/termcmd/strip.go, which does not exist
  plan lines 114-117 declare TabChip, StripModel, RenderedStrip and
  RenderStrip as `new` at a file that is not in the tree; they are M3 work.
  core_concepts_contract_test.go documents the convention this breaks --
  unshipped rows carry `planned` so the status column doubles as the build
  tracker -- and this milestone is what registered #199 in conceptPlans. The
  contract passes only because the couchtty package filter drops those rows,
  so the guard is silent rather than satisfied. Flip the four rows to
  `planned — M3`.
- **BR-12** [Important] `gate-not-cleared-before-code` M1 code landed with the plan-quality gate still blocked and no estimate recorded
  The plan-gate ledger's only round is `blocked: true` with all nine PQ
  findings open, and the issue's estimate_hours is empty with no Estimate
  block -- so change-code appears not to have cleared. PQ-2 was the only
  M1-scoped finding and its remediation did ship here, but the ledger still
  reports it open, which makes the open set wrong entering M2. PQ-3 in
  particular contradicts the plan's own Integration-points row before M3 is
  written. Re-run the gate so PQ-2 disposes `addressed`, and land the
  estimate. Not a defect in M1's code.
- **BR-13** [Minor] `validating-door-bypassed` NewReservation has zero production callers; couch constructs the struct directly
  hostty/reserve.go:49 is the validating constructor, but couchtty/console.go:921
  builds Reservation{} literally. The exported struct makes the door optional
  and the fail-closed methods are what actually protect. Either construct
  through NewReservation at the couch site or drop it and let usable() carry
  the contract alone.
- **BR-14** [Minor] `atlas-points-at-old-home` atlas/couch.md's "The reserved row" section still reads as couch-owned mechanism
  atlas/couch.md:228 describes the reservation with no pointer to
  hostty.Reservation, while architecture.md carries the new note. That
  section is where a reader lands for this subject; add one cross-reference.
- **BR-15** [Minor] `caller-obligation-undocumented` Paint's doc does not state the caller's obligation to sanitize and clamp its text
  hostty/reserve.go:97 writes caller text verbatim between SaveCursor and
  RestoreCursor. Faithful to PaintRow, and couchtty.RenderStatusRow sanitizes
  upstream, but the mechanism/policy split just moved that obligation across
  a package boundary and M3 adds a consumer whose input is operator-supplied
  tab names. One doc line at the new door (ARCH-SECURE).

## Round 3 — 2026-09-07T17:03:57-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Finding 9 records the grep and all three matcher forms; I re-ran it and it reproduces layoutflow.go:56,59,62 and shortcut.go:184-189. Residual restatements at :304 and :391 folded into the new plan-table-drift finding.
- BR-2 — addressed — M1.5 restated in Revisions to behavioural tests; console_live_test.go and vtscreen_test.go diff empty; conceptPlans, conceptInventory and #146's row all landed and the contract is green.
- BR-3 — addressed — Finding 7 names batch.RowDirty in the Sink callback and checks out against child.go:176 and run.go:661. Residual at the Integration-points table (:271) and M3.4/M3.5 folded into the new plan-table-drift finding.
- BR-4 — not-addressed — The stdout half is derived and routed via RunZellijActionQuiet, but runZellij hardwires cmd.Stderr = os.Stderr at run.go:1104 for BOTH methods, so the subprocess's second descriptor is still unrouted and "M2's envelope covers all five" is still false.
- BR-5 — addressed — M2's preamble names the takeover reset as couch's third rule and maps it to redrawTab; M2.3 step 1 resets hostScan plus the deferred slot. Verified against console.go:992-995.
- BR-6 — addressed — M4's Files block now names zellij/layouts/main-3.kdl and zellij/config.kdl as the SOURCE, explains the GeneratedMirror, and adds the regenerate step.
- BR-7 — addressed — rowtext.Sanitize/Fit is named as the shared home with the "unexported in another package" rationale. Residual at ARCH-SECURE (:372-373) and M3.3 (:560) folded into the new plan-table-drift finding.
- BR-8 — not-addressed — M4.3 still has no Alt+Shift+d step; the six *-split rungs stay unverified. Minor, carried.
- BR-9 — not-addressed — TestPaintDefersMidSequenceAndIsOwed still splits at one hand-picked index inside "\x1b[3". Minor, carried.
- BR-10 — addressed — Mutation-verified: reverting Paint's guard to r.Rows == 0 turns reserve_test.go:66 and :101 red. The plan-Revisions half of the ask is still missing and is listed as a plan revision recommendation.
- BR-11 — addressed — All four strip.go rows flipped to "planned — M3" in 692aa11e; the rowtext row carries the same status.
- BR-12 — addressed — Plan-gate rounds 5 and 6 are blocked:false, PQ-2 disposed addressed in round 2, and estimate_hours 7.52 plus an itemized Estimate block landed. Only PQ-8/PQ-9 remain open, consistent with BR-8/BR-9.
- BR-13 — not-addressed — NewReservation still has zero production callers; console.go:922 still builds the struct literal. It also validates only the edge, not rows. Minor, carried.
- BR-14 — not-addressed — atlas/couch.md:228 "The reserved row" still describes the mechanism with no pointer to hostty.Reservation. Minor, carried.
- BR-15 — not-addressed — hostty/reserve.go:97-108 documents save/restore and the one-row deviation but still states no caller obligation to sanitize or clamp. Minor, carried.

### Raised

- **BR-16** [Important] `unreproducible-measurement` The probe behind finding 5 -- the design's load-bearing measurement -- does not exist anywhere in the repo
  The plan closes finding 5 with "Probe kept at scratchpad/199-probe/", but there
  is no scratchpad/ on disk, nothing matching *199*probe*, no such path ever added
  in git log --all --diff-filter=A, and no .gitignore entry hiding one. The plan
  calls this "the load-bearing fact" without which M1/M3 "would have been built on
  sand", and atlas/architecture.md:490-493 now states the result as settled fact.
  ARCH-MOCK at-review: behavior we depend on has a one-time manual reading and no
  retained apparatus or conformance check. Land the probe as a tracked package
  (cmd/probes/couchstartrecovery is the repo's own precedent) or correct the plan
  and atlas to say the method is recorded but the apparatus was not kept.
- **BR-17** [Important] `plan-table-drift` Three gate corrections landed at one site each and left seven restatements of the superseded facts in the same file
  This is the 2nd finding in family plan-table-drift. Do NOT fix these seven sites
  one by one -- the deliverable is the rule and the sweep. Measured prevalence, all
  in workshop/plans/000199-...-plan.md: (1) finding 7 corrects the repaint seam to
  batch.RowDirty, but :271's Integration-points row still names
  ptychild.Child.TakeRowDirty as the wrapped seam and M3.4/M3.5 (:561,:562) still
  say "repaint on TakeRowDirty" -- :271 is what an M3 implementer reads to learn
  which seam to wrap; (2) finding 9 establishes run.go:229 is RegisterTerminalPane
  and not a title reader (confirmed at termcmd/run.go:226-231), but :304 still says
  the #118/#123 consumers "read it (run.go:229)" and :391's ARCH-PURPOSE still
  prints the superseded list AND calls it an enumeration "written out rather than
  left as sweep", which is exactly the shape the plan's own PQ-10 rule forbids;
  (3) the rowtext entry establishes couchtty's sanitize/truncate are unreachable,
  but ARCH-SECURE (:372-373) and M3.3 (:560) still say "exactly as
  couchtty.RenderStatusRow does" / "the same helpers couchtty uses".
  The rule: when a gate finding corrects a fact, the unit of repair is the fact,
  not the sentence the finding quoted -- grep the superseded token across the
  artifact and replace every occurrence in the same round. Checkable form, runnable
  before the next disposition:
  grep -n 'TakeRowDirty\|run\.go:229\|same helpers' workshop/plans/000199-*-plan.md
  must return only the passage documenting the correction itself.
- **BR-18** [Minor] `acceptance-command-does-not-hold` M1.6 is ticked claiming its grep returns nothing; run as written it returns 27 lines
  plan :452. Every hit is a _test.go fixture and no production file outside hostty
  emits SetRegion or a DECSTBM sequence, so the invariant M1.6 defends does hold --
  but the stated command contradicts the tick, and the next reader to re-run it
  gets output. Corrected form: append "| grep -v _test.go".

## Round 4 — 2026-09-07T17:21:53-07:00 (claude) — BLOCKED

### Disposed

- BR-4 — not-addressed — Finding 8 now derives six writers, but M2.3's four pieces and M2.3b still cover stdout only; run.go:1103 still hardwires cmd.Stderr = os.Stderr for both methods.
- BR-8 — not-addressed — M4.3 (plan :601-605) is verbatim unchanged; still no Alt+Shift+d step, and the issue Done-when still requires two halves each drawing a strip.
- BR-9 — not-addressed — M2.1's TestPaintDefersMidSequenceAndIsOwed still splits one hand-picked index inside "\x1b[3".
- BR-13 — not-addressed — console.go:922 still builds Reservation{} literally; the round instead added rows validation to NewReservation, so the door and the production path now carry different contracts for the same type.
- BR-14 — not-addressed — atlas/couch.md:228 is unchanged; only atlas/architecture.md was touched in this window.
- BR-15 — not-addressed — hostty/reserve.go:107-124 Paint still documents save/restore and the one-row deviation but names no caller obligation to sanitize or clamp.
- BR-16 — addressed — Probe landed at cmd/probes/zellijscrollregion, self-locating, with artifactpath and .gitignore updated; see N1/N2 for defects in the landed apparatus.
- BR-17 — not-addressed — The finding's own acceptance grep still returns M3.3 (:568); seven superseded facts remain, including the scratchpad path at :149 in the commit that landed the probe.
- BR-18 — addressed — M1.6 now carries "| grep -v _test.go | grep -v hostty/"; I ran it and it returns nothing.

### Raised

- **BR-19** [Important] `shared-state-unsynchronized` The probe's reader goroutine and its verdict share a strings.Builder with no synchronization
  cmd/probes/zellijscrollregion/main.go:143-153 writes `seen` from a reader
  goroutine while main reads it at :177, :180-181 and :197.
  strings.Builder is not safe for concurrent use; String() exposes the
  backing slice while Write may be growing it, so `go run -race` flags this
  and a torn read yields a confident wrong verdict -- the exact failure class
  the probe's own doc comment says it hit three times. The goroutine also has
  no cancellation path, so its extent is not bounded by main. Guard with a
  mutex, or have the goroutine own the buffer and hand the final string over
  a channel at EOF.
- **BR-20** [Important] `external-input-assumed-wellformed` An unused diagnostic slices the pty stream without a bounds check and can panic before the verdict prints
  cmd/probes/zellijscrollregion/main.go:200 evaluates
  frame[len(frame)-3000:] with no length guard, so a pty yielding fewer than
  3000 bytes panics on the line before the landmarks are checked. tailOf
  (:86-91) is the safe helper this same file already defines (ARCH-DRY). The
  value is never used in the verdict, and it matches "scroll line 0 " with a
  trailing space against a stream where probe.sh emits "scroll line 0\n" --
  yet its `false` output is quoted as evidence in the plan (:143) behind an
  atlas claim. Delete it or make it a checked assertion.
- **BR-21** [Important] `atlas-points-at-old-home` This is the 2nd finding in family atlas-points-at-old-home -- the atlas names one probe home and the code now has two
  Do NOT fix these sites one at a time; BR-14 is still open from the last
  round, which is the family reporting that the enumeration was never
  written. The rule: an atlas statement about where a class of thing lives
  and what runs it must be re-derived whenever the code adds or moves a home,
  and the atlas records the derivation rather than the answer. Measured
  prevalence, three sites: atlas/index.md:17-22 says probes live in probes/
  and "make test-smoke runs every directory under probes/, so a new probe is
  covered by existing" -- the new probe is under cmd/probes/, which that
  sentence does not reach, and grep -n "smoke\|probes" Makefile returns
  nothing, so the target it rests on does not exist; atlas/couch.md:228
  (BR-14) still presents the reserved row as couch-owned mechanism with no
  pointer to hostty.Reservation.
- **BR-22** [Minor] `exit-path-drops-cleanup` Every os.Exit path in the probe skips its deferred session and temp-file cleanup
  cmd/probes/zellijscrollregion/main.go registers cleanup at :120, :142 and
  :156, then exits via os.Exit at :178, :190 and :209 -- so an inconclusive
  or errored run leaks a live zellij session with a `sleep 30` pane and a
  temp KDL. Wrap the body in `run() int` and call os.Exit(run()). Same file,
  :186-192: dump-screen's output is discarded (`_ = out`) yet its failure
  aborts the probe, letting an irrelevant call kill a good measurement.

## Round 5 — 2026-09-07T17:36:01-07:00 (claude) — passed

### Disposed

- BR-4 — not-addressed — Finding 8 derives six writers, but M2.3 at HEAD still routes stdout only; run.go:1104 hardwires cmd.Stderr for both methods and the resize goroutine (run.go:261-266) is in no routing step.
- BR-8 — not-addressed — M4.3 unchanged; no split step anywhere in the plan.
- BR-9 — not-addressed — M2.1's TestPaintDefersMidSequenceAndIsOwed still splits one hand-chosen index.
- BR-13 — not-addressed — console.go:921-922 still builds the literal; NewReservation has zero production callers and now carries a stricter contract than the path production takes.
- BR-14 — not-addressed — atlas/couch.md is not in the review window at all; :228 unchanged.
- BR-15 — not-addressed — hostty/reserve.go:105-116 still names no sanitize/clamp obligation at the new door.
- BR-17 — not-addressed — 3rd round. Its own acceptance grep still returns :568; :149 and :380 also stand, plus the old probe home at probes/zellijscrollregion/main.go:124.
- BR-19 — addressed — syncBuffer guards Write/String/Len with a mutex and the verdict reads one snapshot at main.go:233; the unbounded goroutine extent survives only as BR-22's leak.
- BR-20 — not-addressed — The panic is fixed via tailOf, but the stated remedy was not applied: line000 (main.go:237) is still unused, still trailing-space-matched, still quoted as evidence at plan :143.
- BR-21 — not-addressed — The probe moved and make test-smoke picks it up, but the enumeration was never written: atlas/index.md:16-22 still records the answer, cmd/probes/couchstartrecovery is still a second home, and BR-14 is untouched. (make test-smoke does exist, at Makefile.local:52.)
- BR-22 — not-addressed — os.Exit at main.go:211/223/246 still skips the defers at :148/:170/:189; `_ = out` at :225 unchanged.

### Raised

- **BR-23** [Minor] `doc-states-planned-as-current` The atlas and the new package doc state two consumers of Reservation; production has one
  atlas/architecture.md:474-475 says the primitive "is host-half mechanism
  with two consumers -- couch holds the host's bottom row ... and `pair term`
  holds its pane's bottom row for a tab strip", and hostty/reserve.go:13-16
  says the same. `grep -rn "hostty.Reservation" cmd --include='*.go'` finds
  one production consumer, couchtty/console.go; termcmd acquires its row in
  M3. The atlas is defined in AGENTS.md as the current state of the codebase,
  so a planned consumer stated in the present tense is the same drift class
  this issue keeps paying for. Mark it planned-M3 in both places.
- **BR-24** [Minor] `acceptance-command-does-not-hold` M1.6's acceptance command aborts in the repo's shell before it checks anything
  This is the 2nd finding in family `acceptance-command-does-not-hold`
  (BR-18 was the first, same step). Do NOT just re-quote the command --
  state the rule. The rule: an acceptance command recorded in a plan must be
  recorded in a form that runs UNMODIFIED in the repo's default shell, and
  the round that ticks the step pastes the command's actual output rather
  than its expected outcome. Measured: plan :456 writes
  `grep -rn "SetRegion\|\\x1b\[.*r\"" cmd --include=*.go | ...`; under zsh
  the unquoted `--include=*.go` fails with "no matches found" and the
  pipeline never runs. Quoted, it returns 0 lines and the invariant holds --
  which is exactly BR-18's shape again: the invariant was fine, the recorded
  command was not.

## Round 6 — 2026-09-07T21:54:54-07:00 (claude) — BLOCKED

### Disposed

- BR-4 — addressed — Verified by mutation, not by the commit message: restoring cmd.Stderr = os.Stderr reddens all four FD subtests; resize now goes through resizeThroughWriter.
- BR-8 — not-addressed — M4.3 is unchanged in this window; still no Alt+Shift+d step, and the Done-when still requires two halves each drawing a strip.
- BR-9 — not-addressed — TestPaintDefersMidSequenceAndIsOwed still splits one hand-chosen index; the gate now exists, so parameterizing is cheap.

### Raised

- **BR-25** [Critical] `handler-posts-to-own-queue` removeTab runs on the writer goroutine and posts to the writer goroutine's own channel, deadlocking the pane
  run.go:758 dispatches removeTab on the writer goroutine; removeTab calls setPaneTitle (a blocking zellij subprocess) and then redrawTab (run.go:1081), which since M2 is enqueue -> m.output <- chunk. With the 64-slot buffer full from a second tab's output, the writer blocks sending to the channel only it drains, and enqueue's only escape (m.done) never closes. Reproduced with an overlay test: two tabs, close one while the other floods, writer parked permanently and all pane output stops. Fix: apply the takeover inline from removeTab via a helper shared with the takeover case; add the two-tab close-under-load regression test.
- **BR-26** [Important] `uncovered-negative-assertion` Two assert-absent tests pass vacuously — the owed-paint drop and the subprocess stdout arm are unpinned
  This is the 2nd finding in family uncovered-negative-assertion, so fix the rule, not the two sites. Measured by overlay mutation: deleting m.owed = nil (run.go:766) leaves the entire M2 suite green because writer_test.go:197 never feeds the boundary-ending chunk that would flush it; cmd.Stdout = os.Stdout also stays green because with no live zellij session the succeeding verb writes nothing to stdout, and the positive control built at writer_test.go:227-228 is never asserted. The rule: an assert-absent is evidence only when the same test first establishes the write would have happened. Sweep the four sites in writer_test.go plus run_test.go:910.
- **BR-27** [Important] `diagnostic-silently-discarded` The milestone keeps diagnostics off the pane by destroying them — discarded stderr, and a coalescing slot shared with paints
  run.go:1331 passes io.Discard for the subprocess stderr where the plan specifies capture-and-log, so a failing zellij action reduces to bare exit status, including on the non-pane --test-shortcut path that used to print it. run.go:833 routes reportError through the same single owed slot as paints (run.go:809), where a later paint replaces it, redrawTab drops it (run.go:766), and nothing flushes it if the child stops writing. couch avoids this by owing a paintPending bool re-derived by paintNow rather than bytes (ARCH-DRY). M3's per-repaint paint makes the clobber routine.
- **BR-28** [Important] `plan-table-drift` M2.3 steps 3, 4 and 5 describe an implementation the code does not have, with no Revisions entry
  This is the 3rd finding in family plan-table-drift, so state the rule rather than patching three lines. Piece 3 names five call sites to reroute that are unchanged (the Runtime was changed instead); piece 5 specifies capture-and-log stderr that is discarded; piece 4 states no exception for the startup term: diagnostics that still write stderr directly. The rule: ticking a step asserts the code matches its text, and a deviation gets a Revisions delta in the same commit — mechanized by adding these token pairs to tests/plan-superseded-facts-test.sh, which already exists for exactly this failure.
- **BR-29** [Minor] `dead-test-scaffolding` fakeRuntime.reportedUnused and its reported field are unreachable after the interface change
  run_test.go:941 renames the removed interface method instead of deleting it; nothing calls it and nothing reads fakeRuntime.reported. The live recorder is fakeMux.reported.
- **BR-30** [Minor] `handler-posts-to-own-queue` runDecision still receives the pane's stdout, unused, on the input goroutine
  run.go:166 takes stdin and stdout and uses neither; stdout is threaded from runShell through pumpStdin and handleChord. Removing the parameters makes a future write from the input goroutine a compile error rather than a review catch — the same argument the diff makes for the Runtime.
- **BR-31** [Minor] `hot-path-cost-undeclared` The gate adds a second full parse of every child byte on the output path, undeclared in the envelope
  handleChunk calls hostScan.FeedFraming on every chunk while ptychild.Child already parses the same bytes into its own Screen. ARCH-CONSTRAINTS budgets paints but not this. Also worth stating there: a long unterminated OSC holds MidSequence true up to maxPending, stalling any owed paint.

## Round 7 — 2026-09-07T22:12:24-07:00 (claude) — BLOCKED

### Disposed

- BR-8 — not-addressed — M4 is untouched in this window; still no Alt+Shift+d step and the Done-when still requires two right-pane halves each drawing a strip.
- BR-9 — not-addressed — TestPaintDefersMidSequenceAndIsOwed (writer_test.go:115) still splits one hand-chosen index.
- BR-25 — not-addressed — No code change; reproduced again with an overlay test - 64 slots filled, writer parked past a 3s timeout. run.go:767 -> 1120 -> 1281 -> enqueue.
- BR-26 — not-addressed — Two sites got controls; the class has five. Measured GREEN under mutation: m.owed = nil, cmd.Stdout = stdout, runZellijCaptured, resizeThroughWriter; plus plan-superseded-facts-test.sh:65 bans a token that never existed in the plan.
- BR-27 — not-addressed — Coalescing half closed and mutation-pinned. Discard half: reverting runZellijCaptured to io.Discard leaves the suite green, and run.go:461,467,1118,1210 still drop the error, so a failing wheel tick or rename is still completely silent.
- BR-28 — addressed — M2.3 pieces rewritten to match the code, Revisions delta present, and the piece-3 guard is mutation-live (restoring the wording reddens plan:515). The second guard pair is inert - folded into BR-26.
- BR-29 — not-addressed — run_test.go:941 fakeRuntime.reportedUnused still present and still uncalled.
- BR-30 — not-addressed — run.go:167 runDecision still takes stdin and stdout and uses neither.
- BR-31 — not-addressed — ARCH-CONSTRAINTS unchanged; the second parse and the unterminated-OSC stall are still undeclared.

### Raised

- **BR-32** [Important] `external-input-assumed-wellformed` The BR-27 fix routes an external process's bytes onto the pane's tty, unsanitized and unbounded
  This is the 2nd finding in family external-input-assumed-wellformed, so the deliverable is the rule, not the site. runZellijCaptured (run.go:1396-1414) folds the subprocess's stderr - or its stdout at run.go:1408 - into the error; reportError (run.go:872) writes err.Error() straight to the pane's pty. Reachable via handleChord -> runDecision -> RunZellijAction (reported at run.go:441) and splitTerminalDown (run.go:511). Before this commit the error was a bare exec.ExitError, so no external bytes reached the pane. Only the newline is bounded (run.go:1411); length and escape bytes are not, and setPaneTitle's argument is an operator-controlled tab name zellij's error text can echo back. Measured: current zellij on a pipe emits no escapes, so this is latent rather than observed. The rule - bytes this process did not produce become safe row text at the boundary where they enter, not at the writer that prints them - covers four enumerable sites: runZellijCaptured's detail, reportError's err.Error(), run.go:66's term: %v, and M3's tab names, all of which the plan already routes through rowtext.Sanitize/Fit.
- **BR-33** [Important] `plan-table-drift` The atlas paragraph added in this window was falsified by the next commit in the same window
  This is the 4th finding in family plan-table-drift, so state the rule rather than patching the paragraph. atlas/architecture.md:501-503 says the gate is consulted before "any console-originated write" and that such a write "is deferred into a single coalescing slot", and :506-508 that a takeover drops it - false for diagnostics since 34918a3c, which gave them a queue (run.go:818) that survives a takeover (run.go:779-786). The paragraph's own first sentence lists diagnostics among the writes on the loop, so a reader takes from the atlas the opposite of the decision this milestone was about; :519 has the matching gap for where a failing action's stderr goes now. The rule: an artifact asserting a design is re-checked against the code in the commit that changes the design, and the check is a test. The existing mechanism is scoped to one plan file and to hand-registered tokens - extend tests/plan-superseded-facts-test.sh to atlas/ and register this pair, noting that registering a pair is not evidence unless it fires (see BR-26).
- **BR-34** [Minor] `hot-path-cost-undeclared` owedDiag is an unbounded queue on an externally-timed path, one function below a comment rejecting Feed for that reason
  This is the 2nd finding in family hot-path-cost-undeclared, so the rule: every buffer introduced on a path whose fill rate is set by something outside the process gets a declared bound in ARCH-CONSTRAINTS and a cap in code. run.go:818 appends without limit - one entry per failed refreshRename keystroke while the child sits mid-sequence - and flushes them all at once at the next boundary, while run.go:796-799 explains that Feed was rejected precisely because it would be an unbounded buffer growing behind a terminal that never reads it. Same envelope gap as the undeclared second parse.

## Round 8 — 2026-09-07T22:34:06-07:00 (claude) — BLOCKED

### Disposed

- BR-8 — not-addressed — M4 untouched in this window; M4.3 still has no Alt+Shift+d step while the Done-when still requires two right-pane halves each drawing a strip.
- BR-9 — not-addressed — writer_test.go:115 still splits one hand-chosen index of one sequence.
- BR-25 — not-addressed — No code change. Reproduced again at HEAD with an overlay test - 64 slots filled, writer parked past a 3s timeout. run.go:768 -> removeTab:1127 -> redrawTab:1287 -> enqueue:905.
- BR-26 — not-addressed — Four sites still GREEN under mutation at HEAD - m.owed = nil (run.go:776), runZellijCaptured's detail (:1430 and :1434), resizeThroughWriter (:1156). The stdout arm flipped RED here only because this machine has a live zellij session, which is the ambient-dependence the finding named. plan-superseded-facts-test.sh:65 still registers a token absent from every historical revision of the plan.
- BR-27 — not-addressed — Capture half landed but is unpinned - returning the bare error leaves the suite green. Log half still absent - run.go:462,468,1125,1217 discard the error, so a failing wheel tick or rename is still completely silent.
- BR-29 — not-addressed — run_test.go:941 fakeRuntime.reportedUnused still present and uncalled. Production sibling of the same class - terminalMux.stderr is set by newTerminalMux and read by nothing after the reportError change.
- BR-30 — not-addressed — run.go:168 runDecision still takes stdin and stdout and uses neither.
- BR-31 — not-addressed — ARCH-CONSTRAINTS unchanged. The gap is wider than stated - the second parse covers every tab's output, not the active tab's.
- BR-32 — addressed — Mutation-verified rather than taken from the commit - removing rowtext.SanitizeAndFit at run.go:878 fails TestAZellijErrorCannotPutAnEscapeOrAnUnboundedLineOnThePane. The duplicate strip inside runZellijCaptured is unpinned, but the egress covers the pane path.
- BR-33 — not-addressed — atlas/architecture.md:501-508 still says a console-originated write is deferred into a single coalescing slot and dropped by a takeover, which is false for diagnostics since 34918a3c. The head commit added a further inaccuracy to the same block - :532 states pair term's strip as a current rowtext consumer, which M3 has not built.
- BR-34 — not-addressed — run.go:819 still appends to owedDiag without a bound, and ARCH-CONSTRAINTS still declares no budget for it.

### Raised

- **BR-35** [Critical] `wrong-seam-named` The paint gate is fed bytes the terminal never saw, and not fed bytes it did
  This is the 2nd finding in family wrong-seam-named, so the deliverable is the rule, not the two sites. run.go:804 calls hostScan.FeedFraming on EVERY chunk while run.go:808 writes only the active tab's, so a background tab releases or stalls the gate for the foreground stream; run.go:775 resets the scanner to a zero Screen and then writes the takeover's replay unscanned, and ptychild/replay.go:64-68 documents that a replay can end mid-sequence because Ring bisects it. Both reproduced with overlay tests - "visible\x1b[3PAINT" and "\x1b[1;1H\x1b[Jrestored screen\x1b[3PAINT", a paint inside a live CSI, which is the one failure M2 exists to prevent. Unreachable by the existing suite (one tab) and by M2.5 (yes emits no ESC). The rule - a predicate that guards a stream is computed from exactly the bytes that reach that stream, at the point they are written - covers three enumerable sites - FeedFraming moved inside the isActive branch; the takeover carrying its child half separately and feeding it as it writes it; and M3's batch.RowDirty read, which must likewise be scoped to the active tab because a background child's clear never touched the terminal.
- **BR-36** [Important] `acceptance-command-does-not-hold` M2.5's manual acceptance cannot enter the gate path it is recorded as accepting
  This is the 3rd finding in family acceptance-command-does-not-hold, so state the rule rather than rewriting one step. plan:546 specifies `yes "aaaa…"` flooding one tab; yes emits no ESC byte, so MidSequence is false for the whole run and the defer/owe path never executes, yet the issue Log reasons that "a redraw issued while yes is mid-escape is precisely the interleaving M2 exists to make safe". The run is real evidence for the single-writer half and none for the gate. The rule - an acceptance step names the observable that proves it entered the path under test, and the observation records that observable rather than that the command ran - applies to M2.5 (needs a child emitting escapes plus a second tab, with the deferral counted), M3.7 and M4.3.
- **BR-37** [Important] `external-input-assumed-wellformed` rowtext.Sanitize passes the C1 controls, and it is now the repo's single row-safety strip
  This is the 3rd finding in family external-input-assumed-wellformed, so the rule rather than the site. rowtext.go:37 drops only r < 0x20 and 0x7f; measured, "a\u009b31mred" and "a\u009d0;titlex" survive unchanged, and U+009B/U+009D are the 8-bit CSI and OSC introducers a UTF-8 terminal with C1 handling will act on. ansi.Strip does not frame them either. Inherited from couchtty, but the package doc and atlas/architecture.md:532 now make this the one implementation, and M3 routes operator-typed tab names through it. The rule - the sanitizer's notion of "control" is derived from what a terminal treats as a control (C0, DEL, C1 U+0080-U+009F, and the introducers in both 7- and 8-bit forms) and its test table enumerates those classes rather than the ones a past incident produced. Same edit - couchtty/menu_render.go:625 open-codes Fit(Sanitize(...)) where SanitizeAndFit exists.
- **BR-38** [Minor] `atlas-points-at-old-home` couchtty/reserve.go keeps the doc comments for the sanitize and truncate it no longer has
  This is the 3rd finding in family atlas-points-at-old-home, so the rule - an extraction moves the prose with the symbol and leaves neither comment nor stub at the old home. reserve.go:143-152 ends in two paragraphs describing functions now in rowtext, so a reader greps couchtty for sanitize and finds documentation for a function that is not there. Greppable form of the check - a doc comment whose subject identifier no longer resolves in its own file.

## Open findings

- **BR-8** [Minor] `acceptance-misses-changed-sites` M4's manual acceptance never splits the pane, leaving six of nine borderless sites unverified
- **BR-9** [Minor] `single-interleaving-oracle` The mid-sequence gate test picks one hand-chosen split point over an arbitrary byte stream
- **BR-13** [Minor] `validating-door-bypassed` NewReservation has zero production callers; couch constructs the struct directly
- **BR-14** [Minor] `atlas-points-at-old-home` atlas/couch.md's "The reserved row" section still reads as couch-owned mechanism
- **BR-15** [Minor] `caller-obligation-undocumented` Paint's doc does not state the caller's obligation to sanitize and clamp its text
- **BR-17** [Important] `plan-table-drift` Three gate corrections landed at one site each and left seven restatements of the superseded facts in the same file
- **BR-20** [Important] `external-input-assumed-wellformed` An unused diagnostic slices the pty stream without a bounds check and can panic before the verdict prints
- **BR-21** [Important] `atlas-points-at-old-home` This is the 2nd finding in family atlas-points-at-old-home -- the atlas names one probe home and the code now has two
- **BR-22** [Minor] `exit-path-drops-cleanup` Every os.Exit path in the probe skips its deferred session and temp-file cleanup
- **BR-23** [Minor] `doc-states-planned-as-current` The atlas and the new package doc state two consumers of Reservation; production has one
- **BR-24** [Minor] `acceptance-command-does-not-hold` M1.6's acceptance command aborts in the repo's shell before it checks anything
- **BR-25** [Critical] `handler-posts-to-own-queue` removeTab runs on the writer goroutine and posts to the writer goroutine's own channel, deadlocking the pane
- **BR-26** [Important] `uncovered-negative-assertion` Two assert-absent tests pass vacuously — the owed-paint drop and the subprocess stdout arm are unpinned
- **BR-27** [Important] `diagnostic-silently-discarded` The milestone keeps diagnostics off the pane by destroying them — discarded stderr, and a coalescing slot shared with paints
- **BR-29** [Minor] `dead-test-scaffolding` fakeRuntime.reportedUnused and its reported field are unreachable after the interface change
- **BR-30** [Minor] `handler-posts-to-own-queue` runDecision still receives the pane's stdout, unused, on the input goroutine
- **BR-31** [Minor] `hot-path-cost-undeclared` The gate adds a second full parse of every child byte on the output path, undeclared in the envelope
- **BR-33** [Important] `plan-table-drift` The atlas paragraph added in this window was falsified by the next commit in the same window
- **BR-34** [Minor] `hot-path-cost-undeclared` owedDiag is an unbounded queue on an externally-timed path, one function below a comment rejecting Feed for that reason
- **BR-35** [Critical] `wrong-seam-named` The paint gate is fed bytes the terminal never saw, and not fed bytes it did
- **BR-36** [Important] `acceptance-command-does-not-hold` M2.5's manual acceptance cannot enter the gate path it is recorded as accepting
- **BR-37** [Important] `external-input-assumed-wellformed` rowtext.Sanitize passes the C1 controls, and it is now the repo's single row-safety strip
- **BR-38** [Minor] `atlas-points-at-old-home` couchtty/reserve.go keeps the doc comments for the sanitize and truncate it no longer has
