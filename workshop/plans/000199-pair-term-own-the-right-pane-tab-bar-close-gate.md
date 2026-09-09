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
    - "n": 9
      timestamp: "2026-09-07T23:06:21-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: not-addressed
          note: M4 untouched in this window; M4.3 still has no Alt+Shift+d step while the Done-when still requires two right-pane halves each drawing a strip.
          round: 9
        - id: BR-9
          disposition: not-addressed
          note: writer_test.go:115 still splits one hand-chosen index of one sequence.
          round: 9
        - id: BR-25
          disposition: not-addressed
          note: No code change. Reproduced twice at HEAD with overlay tests - deterministic (full queue, handleChunk parked >3s) and under load (writer never drained again). run.go:772 -> 1088 -> 1303 -> 922.
          round: 9
        - id: BR-26
          disposition: not-addressed
          note: Measured green under mutation at HEAD - m.owed = nil (run.go:780), runZellijCaptured's SanitizeAndFit (:1447), resizeThroughWriter (:264); plan-superseded-facts-test.sh:65 still bans a string git log -S finds in no revision of the plan.
          round: 9
        - id: BR-27
          disposition: not-addressed
          note: Log half still absent - run.go:462,468,1142,1234 discard the error, so a failing wheel tick or rename is still completely silent while reportError sits one line away.
          round: 9
        - id: BR-29
          disposition: not-addressed
          note: run_test.go:941 fakeRuntime.reportedUnused still present and uncalled; terminalMux.stderr is still set at run.go:683 and read by nothing.
          round: 9
        - id: BR-30
          disposition: not-addressed
          note: run.go:168 runDecision still takes stdin and stdout and uses neither.
          round: 9
        - id: BR-31
          disposition: not-addressed
          note: ARCH-CONSTRAINTS unchanged. The BR-35 fix narrowed the second parse to the active tab, which is the fact to declare.
          round: 9
        - id: BR-33
          disposition: not-addressed
          note: atlas:501-508 still says a single coalescing slot dropped by a takeover; :527 says diagnostics are gated like any other write, which C2 falsifies; :532 still lists M3's strip as a current consumer. The superseded-facts test still covers only the plan.
          round: 9
        - id: BR-34
          disposition: not-addressed
          note: run.go:819 still appends to owedDiag unbounded; ARCH-CONSTRAINTS declares no budget. Same gap covers diagnosticWidth = 200, whose stated basis (far short of wrapping) is false for an 80-column pane while m.cols is on the receiver.
          round: 9
        - id: BR-35
          disposition: addressed
          note: Mutation-verified both directions rather than taken from the commit - moving FeedFraming out of the isActive branch and deleting the replay feed each redden their own subtest.
          round: 9
        - id: BR-36
          disposition: not-addressed
          note: Evidence half corrected honestly (Log + M2.5), but the correction defers the gate's manual acceptance to M3.7 and plan:589 has no step that enters the gate path - no second tab, no escape-emitting child, no deferral counted.
          round: 9
        - id: BR-37
          disposition: addressed
          note: Reverting the C1 clause at rowtext.go:44 reddens four subtests; raw 8-bit spellings degrade to U+FFFD (measured). Residual, not re-raised - menu_render.go:625 still open-codes Fit(Sanitize(...)).
          round: 9
        - id: BR-38
          disposition: not-addressed
          note: couchtty/reserve.go:143-152 still ends in doc comments for functions that moved to rowtext, and this window added the same shape at run.go:1410-1418, where two consts now sit between runZellijCaptured's prose and runZellijCaptured.
          round: 9
      findings:
        - id: BR-39
          severity: Critical
          title: The takeover writes owed diagnostics straight to the pane after feeding the replay to the gate, so a diagnostic lands inside the replay's open sequence
          detail: 'This is the 2nd finding in family validating-door-bypassed, so the deliverable is the rule, not the site. run.go:792-793 writes each queued diagnostic with m.stdout.Write, immediately after run.go:791 feeds chunk.replay to hostScan - so when the replay ends mid-sequence (the case run.go:787-790 exists to handle, and which writer_test.go:419 asserts is real), MidSequence is true and the write goes out anyway. Reproduced with an overlay test - a diagnostic queued while the child was mid-sequence, then redrawTab("restored\x1b[3"), yields "x\x1b[3\x1b[1;1H\x1b[Jrestored\x1b[3pair term: queued failure\r\n" on the pane, the diagnostic''s leading bytes swallowed as CSI parameters. Reachable: a failed rename or a failed zellij action while the child is mid-escape, then any tab switch. The takeover''s own write at :786 is NOT the same case and must stay exempt - it begins with ESC, which cancels a pending CSI in xterm-class terminals, which is why couch''s takeover writes ungated; that exemption should be stated rather than re-derived. The rule - every byte reaching m.stdout from the writer loop passes through the gate-consulting door (writeOwn / writeDiag), or carries a written exemption naming why the gate does not apply - covers the enumerable set `grep -n ''m\.stdout'' cmd/internal/termcmd/run.go`: today :786 (exempt, takeover), :793 (the defect), :826 (the child stream itself), :839/:857/:868/:876 (inside the doors), :1321 (teardown, exempt). The test that pins the rule enumerates the console-originated write paths and asserts each defers while the gate is mid-sequence, which is what the current suite does for paints and not for the takeover-carried diagnostics.'
          family: validating-door-bypassed
          round: 9
      boundary: M2
      blocked: true
    - "n": 10
      timestamp: "2026-09-07T23:27:09-07:00"
      agent: claude
      dispose:
        - id: BR-25
          disposition: addressed
          note: 'Verified by revert, not by the commit message: restoring redrawTab in removeTab reddens TestAChildExitingWithAFullBufferDoesNotWedgeThePane at the 3s timeout. The rewritten test is deterministic (buffer saturated, no loop running).'
          round: 10
        - id: BR-39
          disposition: not-addressed
          note: 'Instance fixed and mutation-verified (reverting writeDiag to m.stdout.Write reproduces the exact predicted bytes). Class half absent: applyTakeover''s own ungated write at run.go:826-827 carries no written exemption, and no test enumerates the console-originated doors.'
          round: 10
        - id: BR-26
          disposition: not-addressed
          note: 'Re-measured at HEAD. Green under all three mutations: m.owed = nil (run.go:822), runZellijCaptured''s SanitizeAndFit, resizeThroughWriter. plan-superseded-facts-test.sh:65 bans a string git log -S finds in no revision of the plan, so that registered pair can never fire.'
          round: 10
        - id: BR-27
          disposition: not-addressed
          note: Log half absent. run.go:462,468,1165,1257 still discard the error; there is no log sink, so a failing wheel tick or rename is completely silent where pre-M2 it printed. run.go:1437's own comment states this as the reason capture exists. Note run.go:1165 is on the writer goroutine, so its fix needs inline writeDiag, not reportError.
          round: 10
        - id: BR-33
          disposition: not-addressed
          note: atlas:502-503 still describes every console-originated write as one coalescing slot, false for diagnostics; atlas:532-534 lists pair term's strip as a current rowtext consumer when it is M3. tests/plan-superseded-facts-test.sh still covers only the plan file.
          round: 10
        - id: BR-34
          disposition: not-addressed
          note: run.go:848 still appends to owedDiag with no cap; ARCH-CONSTRAINTS declares no budget. diagnosticWidth = 200 claims to be far short of wrapping while m.cols is on the receiver.
          round: 10
        - id: BR-31
          disposition: not-addressed
          note: ARCH-CONSTRAINTS unchanged. Screen.feedFraming is a per-byte loop run over every active chunk in addition to ptychild's own parse; the BR-35 fix narrowed it to the active tab, which is the fact to declare. Same gap covers the zellij exec now running on the sole writer goroutine in removeTab.
          round: 10
        - id: BR-36
          disposition: not-addressed
          note: M2.5 and the Log are corrected honestly, but M3.7(b) still specifies yes, which emits no ESC, so the deferral path still cannot be entered; no observable is named and no deferral is counted. M4.3 unchanged.
          round: 10
        - id: BR-38
          disposition: not-addressed
          note: couchtty/reserve.go:143-152 still ends in doc comments for sanitize and truncate, which live in rowtext now. This window added the same shape at run.go:1431-1441, where two consts sit between runZellijCaptured's prose and runZellijCaptured.
          round: 10
        - id: BR-29
          disposition: not-addressed
          note: run_test.go:941 fakeRuntime.reportedUnused still present and uncalled. terminalMux.stderr is set at run.go:683 and read nowhere (grep for m.stderr returns nothing).
          round: 10
        - id: BR-30
          disposition: not-addressed
          note: run.go:168 runDecision still takes stdin and stdout and uses neither; panes is likewise unused in the body.
          round: 10
        - id: BR-8
          disposition: not-addressed
          note: M4 untouched in this window; M4.3 still has no Alt+Shift+d step while the Done-when still requires two right-pane halves each drawing a strip.
          round: 10
        - id: BR-9
          disposition: not-addressed
          note: writer_test.go:115-116 still splits one hand-chosen index of one sequence.
          round: 10
      findings:
        - id: BR-40
          severity: Important
          title: The takeover's scan reset -- couch's third gate rule, M2.3 step 1's headline -- can be deleted with the whole suite green
          detail: 'This is the 3rd finding in family uncovered-negative-assertion, so the deliverable is the rule, not the site; it also shows BR-26''s stated sweep list ("the four sites in writer_test.go plus run_test.go:910") was an enumeration written from memory, since this site is not on it. Measured in a scratch copy: deleting m.hostScan = ptychild.Screen{} (run.go:822) leaves ./cmd/internal/termcmd fully green. The reason is the fixtures, not the assertion -- both takeovers replay bytes beginning with f (0x66) and r (0x72), and both are legal CSI final bytes, so FeedFraming(replay) closes the child''s pending sequence on its own and midSequenceForTest() reads false with or without the reset. A discriminating fixture is one line: feed "x\x1b[3", then redrawTab([]byte("0000")) -- parameter bytes with no final. Green with the reset, red without it (verified both directions). The rule: an assertion that a state was CLEARED is evidence only when the fixture cannot reach that state by any path other than the clearing. In practice that means a fixture chosen to be inert with respect to the mechanism under test, and the check that makes it stick is the mutation, not the read -- which is the same rule BR-26 states for assert-absent writes, now shown to govern assert-absent STATE as well.'
          family: uncovered-negative-assertion
          round: 10
        - id: BR-41
          severity: Important
          title: runZellij hands the subprocess the pane's raw-mode stdin, while code and atlas both claim it gives neither descriptor
          detail: 'This is the 2nd finding in family envelope-claim-unenforced, so state the rule rather than patching the line. run.go:1426 sets cmd.Stdin = os.Stdin -- the pane''s own descriptor, put in raw mode by host.MakeRaw() at run.go:242 -- while run.go:1394 asserts "NEITHER method gives a subprocess the pane''s descriptors, and that is the point" and atlas/architecture.md:514 repeats it. The enforcement test is named TestNeitherZellijMethodHandsTheSubprocessThePanesDescriptors and captures two of the three, so the claim''s third case is unasserted by construction. Whether any zellij action verb reads stdin is unmeasured; a verb that reads one byte eats an operator keystroke silently, which is exactly the class the milestone exists to close. The rule: an envelope claim about a RESOURCE enumerates every handle of that resource and the test asserts each one -- "descriptors" means the three the process was started with, not the two the finding that prompted the work happened to name. The enumerable set here is the three fields of exec.Cmd set in runZellij; either close stdin or narrow the claim in all three artifacts (plan finding 8, run.go:1394, atlas:514) in the same commit.'
          family: envelope-claim-unenforced
          round: 10
        - id: BR-42
          severity: Minor
          title: The Integration-points table still marks three M3/M4 rows new/modified after BR-11 flipped only the Pure-entities table
          detail: 'This is the 5th finding in family plan-table-drift, so do not patch the three rows -- state the rule. plan:279-281 declare "strip repaint trigger" new, "degraded rename-pane" modified and "right pane chrome" modified; none exists at HEAD, all three are M3/M4. BR-11 raised exactly this for plan:114-117 and those rows now read "planned -- M3", which is the instance fixed and the class left standing in the table directly below it. The guard cannot catch it: core_concepts_contract_test.go filters on conceptPackage = cmd/internal/couchtty/, so every #199 row declaring termcmd, hostty, rowtext or main-3.kdl is unchecked, and the status column that "doubles as the build tracker" is only a tracker for one package. The rule: the status column is the build tracker for EVERY row of EVERY table in a Core concepts section, and the check that enforces it is scoped to the plan, not to a package -- which is the same gap BR-33 names for atlas, and the reason both should be answered by one mechanism rather than two hand-maintained lists.'
          family: plan-table-drift
          round: 10
      boundary: M2
      blocked: true
    - "n": 11
      timestamp: "2026-09-07T23:47:27-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: not-addressed
          note: M4.3 (plan:601-605) verbatim unchanged; no Alt+Shift+d step anywhere, while the issue Done-when still requires two right-pane halves each drawing a strip.
          round: 11
        - id: BR-9
          disposition: not-addressed
          note: writer_test.go:116-117 still splits one hand-chosen index of one sequence; the gate now exists, so parameterizing is cheap.
          round: 11
        - id: BR-26
          disposition: not-addressed
          note: Re-measured at HEAD in a scratch copy - green under mutation at three sites - m.owed = nil (run.go:827), runZellijCaptured's SanitizeAndFit (run.go:1488), resizeThroughWriter (run.go:264). plan-superseded-facts-test.sh:65 still bans 'route only stdout through', which git log -S finds in no revision of the plan, so that registered pair can never fire.
          round: 11
        - id: BR-27
          disposition: not-addressed
          note: Capture half landed; log half still absent. run.go:462, 468, 1178, 1270 discard the error and there is no log sink, so a failing wheel tick or rename is completely silent where pre-M2 it printed. run.go:1178 is on the writer goroutine, so its fix needs an inline writeDiag rather than reportError.
          round: 11
        - id: BR-29
          disposition: not-addressed
          note: run_test.go:941 fakeRuntime.reportedUnused still present and called from nowhere; terminalMux.stderr (run.go:643) is set by newTerminalMux and read by nothing - grep for m.stderr returns no hits.
          round: 11
        - id: BR-30
          disposition: not-addressed
          note: run.go:168 runDecision still takes stdin and stdout and uses neither; nothing in the handleChord path writes stdout.
          round: 11
        - id: BR-31
          disposition: not-addressed
          note: ARCH-CONSTRAINTS is unchanged in this window - the plan diff touches only Revisions. The second parse, the unterminated-OSC stall, and the synchronous zellij exec on the sole writer goroutine in removeTab are all still undeclared.
          round: 11
        - id: BR-33
          disposition: not-addressed
          note: atlas/architecture.md:502-505 still says a write issued mid-sequence is deferred into a single coalescing slot with a later paint replacing an earlier one - false for diagnostics, which queue in owedDiag and survive a takeover. atlas:537 still lists pair term's strip as a current rowtext consumer when it is M3. tests/plan-superseded-facts-test.sh still covers only the plan file and one probe.
          round: 11
        - id: BR-34
          disposition: not-addressed
          note: run.go:861 still appends to owedDiag with no cap and ARCH-CONSTRAINTS declares no budget; diagnosticWidth = 200 still claims to be far short of wrapping while m.cols is on the receiver.
          round: 11
        - id: BR-36
          disposition: not-addressed
          note: M3.7(b) landed (plan:595) but still specifies flooding with yes, which emits no ESC, so MidSequence is false for the whole run and the defer path still cannot be entered. No observable is named and no deferral is counted. M4.3 unchanged.
          round: 11
        - id: BR-38
          disposition: not-addressed
          note: couchtty/reserve.go:143-152 still ends in doc paragraphs for sanitize and truncate, which now live in rowtext. This window also left run.go:1454-1466, where two consts sit between runZellijCaptured's doc prose and runZellijCaptured, so go doc attaches that prose to diagnosticWidth.
          round: 11
        - id: BR-39
          disposition: addressed
          note: Instance fixed in round 4 and class instrument added here - verified by mutation, an ungated m.stdout.Write in reportError reddens TestEveryConsoleWriteIsGatedOrExplicitlyExempt. Two residuals raised as a new finding - the predicate's blind spots - and the takeover exemption's stated reason still omits the mechanism that actually earns it (a leading ESC cancels a pending CSI), which round 9 asked to be written rather than re-derived.
          round: 11
        - id: BR-40
          disposition: not-addressed
          note: Re-measured in a scratch copy of HEAD - replacing m.hostScan = ptychild.Screen{} (run.go:826) with a no-op leaves ./cmd/internal/termcmd fully green. The one-line discriminating fixture round 10 supplied (feed "x\x1b[3", then redrawTab([]byte("0000"))) was not added; both takeover fixtures still replay bytes beginning with legal CSI final bytes.
          round: 11
        - id: BR-41
          disposition: addressed
          note: Verified by revert - restoring cmd.Stdin = os.Stdin reddens TestTheZellijSubprocessGetsNoStdinEither. atlas:517-520 now says none of the pane's descriptors, including stdin. The stdin arm is still asserted by a source substring rather than behaviorally, which is folded into the new envelope-claim-unenforced finding.
          round: 11
        - id: BR-42
          disposition: not-addressed
          note: plan:277-281 unchanged - strip repaint trigger still new, degraded rename-pane and right pane chrome still modified, none existing at HEAD. The same table marks couchtty.RenderStatusRow unchanged after this window changed its body to call rowtext.
          round: 11
      findings:
        - id: BR-43
          severity: Important
          title: The door-enumeration test enforces a substring, not the claim - an ungated Fprintf, a leaked exemption marker, and every file but run.go all pass
          detail: This is the 3rd finding in family envelope-claim-unenforced, so the deliverable is the rule, not the three sites. writer_test.go:563 requires a door line to contain ".Write" or "WriteString", so fmt.Fprintf(m.stdout, ...) is invisible - measured, adding one to inheritSize leaves the entire termcmd suite green, and Fprintf is the spelling the pre-M2 code used for exactly these diagnostics. writer_test.go:539-552 stops the exemption walk only on a non-empty non-comment line, so a blank line does not stop it - measured, an unrelated "gate-exempt:" comment two lines above an added ungated write exempts it, which is what writer_test.go:544's own comment claims to prevent. And writer_test.go:525 reads run.go alone, while the plan's M3 file list creates cmd/internal/termcmd/strip.go in the same package. The rule - an enforcement test for a claim of the form "every X does Y" derives X from the language, not from the spellings the author happened to write, and the instrument is mutation-checked against each spelling and each file before it is trusted. The milestone already made the stronger choice once, in M2.3 step 3 - make the RUNTIME incapable rather than fix call sites - and the same move applies here - put m.stdout behind a paneWriter whose only methods are own / diag / exempt(reason), so a new door is a compile error rather than a grep.
          family: envelope-claim-unenforced
          round: 11
        - id: BR-44
          severity: Minor
          title: flushOwed has one call site, in the child-data branch, so an owed write is stranded for as long as the child is silent
          detail: run.go:813 is the only call to flushOwed, inside handleChunk's default (child data) branch. Nothing else pays the debt - not a bare event, not a takeover, not resize, and there is no timer. So while the child's stream is mid-sequence and the child produces nothing further (a prefix written before the child blocks, or Screen.skipping latched by a sequence over maxPending that is never terminated), an owed paint and every queued diagnostic sit invisible indefinitely. That is the same stale row that nothing repaints that run.go:866-874 gives as the reason for owing rather than dropping. Latent at M2 because nothing paints; at M3 it means a strip that stops updating with no path back. couch shares the shape (console.go:1137), so the answer belongs in the shared design - a bounded wait after which the write goes out, or paying the debt from every event rather than only from child output.
          family: deferred-work-lacks-own-trigger
          round: 11
      boundary: M2
      blocked: false
    - "n": 12
      timestamp: "2026-09-08T13:28:35-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: not-addressed
          note: M4.3's acceptance still has no Alt+Shift+d step, so :138/:139, :165/:166, :191/:192 remain unverified.
          round: 12
        - id: BR-9
          disposition: not-addressed
          note: TestPaintDefersMidSequenceAndIsOwed is unchanged and still splits one hand-chosen sequence at one index.
          round: 12
      findings:
        - id: BR-45
          severity: Important
          title: A tab exiting during a rename repaints nothing, so the strip keeps listing the closed tab
          detail: |-
            run.go:1296 -- removeTab's `if !preserveRename { m.applyTakeover(...) }` skips the only
            thing that repaints the strip on that path. Reproduced: zero bytes written after tab 1's
            child EOFs while a rename is open. 2nd in this family: do NOT patch removeTab alone.
            The rule is that every site mutating the strip model owes a repaint, and the site set is
            enumerated by a test -- a table over {newTab, switchRelative, closeActive, removeTab (both
            branches), beginRename, refreshRename, finishRename, inheritSize, rowDirty} that drives
            each and asserts a repaint carrying the post-mutation model. It fails on exactly one row today.
          family: exit-path-drops-cleanup
          round: 12
        - id: BR-46
          severity: Important
          title: flushOwed writes the coalesced older row after the fresh inline repaint already landed
          detail: |-
            run.go:866-872 -- the row-dirty debt repaints inline with the current model, then flushOwed
            writes m.owed, which is older. Reproduced: the recorder's last write is "one [two]" after
            "one [built]". This contradicts writeOwn's own stated invariant that the freshest paint is
            the only one worth landing. Reachable via removeTab's preserveRename branch, which mutates
            the model on the writer goroutine without posting a paint or clearing m.owed. Fix: clear
            m.owed when paintStripInline writes directly, or flushOwed before the inline repaint.
          family: superseded-write-not-dropped
          round: 12
        - id: BR-47
          severity: Important
          title: M3.7 is ticked but (b) -- the escape-carrying flood that reaches the defer-and-owe path -- was never run
          detail: |-
            4th in this family, so do NOT just run the step. Every load generator in the record emits no
            escapes: the smoke pass predates the current gate, the Ctrl-C probe used `yes aaa`, and
            couchnestedrows floods with `seq 1 400`. So the defer-and-owe path has no manual evidence in
            the milestone that WIDENED the defer condition to include HoldsCursorSave. The rule: a manual
            acceptance step is ticked only against a Log line quoting the command run and what was seen,
            and milestone-close refuses a ticked manual step with no such line. Until then un-tick M3.7.
          family: acceptance-command-does-not-hold
          round: 12
        - id: BR-48
          severity: Important
          title: The degraded title asserts one of two derived consumers, and the sibling producer was never swept
          detail: |-
            3rd in this family. Two measured instances of one rule. (1) M3.6 required asserting
            ClassifyLiveLayout as well as RoleForPane; only RoleForPane is asserted, and measured,
            Title="terminal 1" with no command now classifies layout3 as layout2 where the packed
            "[terminal 1] work" classified layout3 -- layoutflow_test.go:143 still fixtures the retired
            "[terminal 1]" form. (2) renamePaneTitleLocked (run.go:1533) was not degraded at all: it still
            packs the tab set and drops the load-bearing prefix, measured as
            "[rename: work-bar] other" -> PaneRoleOther. The rule: where a milestone changes a value a
            derived consumer set reads, the test is a table over every producer x every consumer,
            generated from the derivation the plan already records -- not one hand-picked pair.
          family: consumer-set-not-derived
          round: 12
        - id: BR-49
          severity: Important
          title: The defer condition was widened to one that can persist indefinitely, and M3's assigned flush deadline was not delivered
          detail: |-
            2nd in this family. run.go:981 states "M3 owes it a flush deadline"; no deadline exists.
            Meanwhile unsafeToPaint (run.go:939) now also defers on HoldsCursorSave, which unlike
            MidSequence can be held for as long as a child chooses, and owedDiag is an unbounded append
            with no cap. The rule: a write deferred on a condition the deferrer does not control needs a
            trigger the deferrer does control -- a deadline, a bound, or a written argument that unbounded
            deferral is correct for that payload class. State it per class: unbounded is right for the
            paint (stale beats wrong, already tested); it is not right for a diagnostic.
          family: deferred-work-lacks-own-trigger
          round: 12
        - id: BR-50
          severity: Important
          title: A zellij subprocess forks per rename keystroke, against the plan's declared "not on every render" budget
          detail: |-
            4th in this family. refreshRename (run.go:1173), called once per rename event from the stdin
            pump (run.go:390), calls setPaneTitle -> RunZellijAction -> a process fork, on the interaction
            path ARCH-CONSTRAINTS names as the one that matters, and M3 now adds a strip paint alongside
            it. The issue's Problem opens with "a subprocess per title change" as cost #1. The rule: an
            ARCH-CONSTRAINTS budget must be enforced by something a change trips over, not asserted in
            prose -- count fakeRuntime ops across a scripted rename and assert the bound, or rewrite the
            budget to state the real cost. Cheapest true fix: refresh the title on commit only, since the
            strip now renders the field and the pane is focused throughout a rename.
          family: envelope-claim-unenforced
          round: 12
        - id: BR-51
          severity: Important
          title: probes/cursorsaveslots force-deletes a zellij session it did not create, from make test-smoke
          detail: |-
            It diffs `zellij list-sessions` before/after, picks an arbitrary new name by ranging a map,
            and defers `zellij delete-session <name> --force`. Any session appearing in that 8-second
            window -- the operator's own workbench -- is destroyed. It lives in probes/, which test-smoke
            runs wholesale. The fix was found in this same milestone and not applied: couchnestedrows
            (main.go:112) uses --new-session-with-layout with a deterministic couchnestedrows-<pid> name,
            which is exactly the flag the session-diff dance exists to work around.
          family: test-can-touch-real-state
          round: 12
        - id: BR-52
          severity: Important
          title: 196 identical lines duplicated between the two probes/ zellij harnesses
          detail: |-
            probes/cursorsaveslots/main.go (287 lines) and probes/zellijscrollregion/main.go (281 lines)
            share 196 identical lines: sessionSet, syncBuffer, tailOf, writeLayout, the ZELLIJ* scrub, the
            pty reader goroutine, lastCursorRowBefore, and the session discovery/cleanup. couchnestedrows
            repeats 83, termctrlc 49, termrows 45. Nothing blocks extraction -- a plain probes/zellijprobe/
            package is importable by every probes/* main, unlike BR-7's unexported-in-another-package case.
            The cost is already realised: the finding above is a bug present in the copy and fixed only in
            the copy's successor.
          family: copy-instead-of-extract
          round: 12
        - id: BR-53
          severity: Minor
          title: Three superseded doc comments left standing beside their corrections
          detail: |-
            2nd in this family, so state the rule rather than editing three sites: when a comment is
            superseded, DELETE it -- do not prepend the correction, and never leave two doc comments on
            one declaration. Sites: run.go:1508 (paneTitleLocked claims the packed title matched
            shortcut.go's arm NEVER, twenty lines above the paragraph saying it did, and the plan records
            the first as measured-false); run.go:1368-1370 (childSizeLocked has two stacked doc comments,
            the first now the opposite of the behaviour); ptychild/screen.go:141-156 (HoldsCursorSave's
            doc was inserted mid-comment, so godoc attaches TakeRowDirty's whole rationale to it and
            TakeRowDirty has none). A vet-style check for two comment blocks on one declaration catches
            the class.
          family: doc-states-planned-as-current
          round: 12
        - id: BR-54
          severity: Minor
          title: The Core concepts table still says "planned — M3" and omits five entities M3 introduced
          detail: |-
            6th in this family. plan.md:225-228 still marks TabChip/StripModel/RenderedStrip/RenderStrip as
            planned; RenameField, TabSpan, RenameEditor.Field, hostty.ResetSGR/ReserveAndPaint and
            Screen.HoldsCursorSave are absent, as is the stripOwed debt from the integration table. The
            rule, since instance-fixing has not held five times: a hand-maintained table restating the
            tree drifts every milestone. Either derive it -- a test that greps each row's stated path for
            its stated symbol and fails on a status that disagrees with the diff -- or drop the status
            column so the table stops making a claim nothing checks.
          family: plan-table-drift
          round: 12
        - id: BR-55
          severity: Minor
          title: README does not mention that the right pane loses a row to the strip or gains a tab bar
          detail: |-
            README.md:15 describes the pane's tabs but not the strip. probes/termsmoke changed its
            assertion from "40 100" to "39 100": the child is now one row shorter, which is observable to
            anything the operator runs in the pane. M4 is the natural place to land both, but it should be
            listed there rather than left to be noticed.
          family: user-facing-change-undocumented
          round: 12
      boundary: M3
      blocked: true
    - "n": 13
      timestamp: "2026-09-08T14:02:38-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: not-addressed
          note: M4.3 still has no Alt+Shift+d step; the plan records it as deferred to M4.
          round: 13
        - id: BR-9
          disposition: not-addressed
          note: TestPaintDefersMidSequenceAndIsOwed is byte-identical and still splits one sequence at one index.
          round: 13
        - id: BR-45
          disposition: addressed
          note: 'Mutation-verified: removing removeTab''s paintStripInline turns the rename-branch table row red.'
          round: 13
        - id: BR-46
          disposition: addressed
          note: 'Mutation-verified: removing writeOwn''s `m.owed = nil` makes the stale row land after the fresh one.'
          round: 13
        - id: BR-47
          disposition: addressed
          note: Automated in couchnestedrows with an SGR-per-line flood plus tab switches; could not re-run here (needs zellij + a real pty).
          round: 13
        - id: BR-48
          disposition: addressed
          note: Second producer deleted, ClassifyLiveLayout derives from RoleForPane, table mutation-verified; see the new finding for the producer's own restatement of the predicate.
          round: 13
        - id: BR-49
          disposition: addressed
          note: Decided per payload; owedDiag capped at maxOwedDiag with a test, unbounded paint deferral argued and pinned.
          round: 13
        - id: BR-50
          disposition: addressed
          note: TestARenameCostsExactlyOneZellijSubprocess asserts exactly one recorded op across three keystrokes.
          round: 13
        - id: BR-51
          disposition: addressed
          note: zellijprobe.Start names and Close deletes that name; swept the tree, every delete-session under probes/ and cmd/probes/ targets a self-chosen name.
          round: 13
        - id: BR-52
          disposition: addressed
          note: Verified by line count -- cursorsaveslots 172, zellijscrollregion 165, shared harness 219.
          round: 13
        - id: BR-53
          disposition: not-addressed
          note: paneTitleLocked's godoc still carries the measured-false claim with a correction appended below it, and redrawTab still stacks three doc paragraphs; no class guard added.
          round: 13
        - id: BR-54
          disposition: addressed
          note: Table flipped and extended and the planned-Mx rot guarded; residual noted in the new plan-table-drift finding.
          round: 13
        - id: BR-55
          disposition: addressed
          note: README now documents the strip and the row the pane gives up.
          round: 13
      findings:
        - id: BR-56
          severity: Important
          title: paneTitleLocked restates RoleForPane's predicate instead of asking it, so a tab named `terminals` classifies as PaneRoleOther
          detail: |-
            This is the 4th finding in family `consumer-set-not-derived`, so state the rule rather than
            patching the prefix. Measured against the real consumer at run.go:1557 --
            tab `terminalwork` -> title `terminalwork` -> PaneRoleOther; same for `terminal-2` and
            `terminals`. The guard is `HasPrefix(name, "terminal")`, an approximation of
            `title == "terminal" || HasPrefix(title, "terminal ")`, so the pane loses the classification
            that routes global shortcuts whenever TerminalCommand is unavailable -- the exact consequence
            BR-48 described, one level upstream of where BR-48 was fixed.
            THE RULE: a producer must never restate the predicate its consumer applies; it ASKS the
            consumer. Export the predicate from workbenchshortcut and have paneTitleLocked add the prefix
            iff the bare name does not already classify -- then the disagreement is unrepresentable.
            And the producer x consumer table needs its PRODUCER axis derived too: three hand-picked names
            cannot see a boundary case, which is why this survived a table written specifically for BR-48.
          family: consumer-set-not-derived
          round: 13
        - id: BR-57
          severity: Important
          title: A background tab's rowDirty batch repaints the strip and re-asserts DECSTBM over the active child's margins
          detail: |-
            This is the 5th finding in family `envelope-claim-unenforced`, so state the rule rather than
            moving the one `if`. run.go:850 sets `m.stripOwed = true` outside the `m.isActive(chunk.id)`
            guard, so a tab whose bytes never reached the terminal drives a full re-Reserve plus repaint.
            Measured with the repo's own harness: `ptyChunk{id: 1, data: "\x1b[2J", rowDirty: true}` on the
            BACKGROUND tab wrote `\x1b7\x1b[1;23r\x1b[24;1H\x1b[0m\x1b[2Kone [two]\x1b[0m\x1b8`. That
            violates the plan's declared budget ("a repaint happens on tab change, resize, and
            batch.RowDirty; NOT per output chunk") on the keystroke path, and clobbers the ACTIVE child's
            own scroll region for a screen the terminal never saw -- M2's BR-35 lesson (the gate models the
            terminal, so feed it exactly what the terminal is shown) applied to the repaint trigger rather
            than to the gate.
            THE RULE: every budget bullet in the plan's ARCH-CONSTRAINTS block is an enumeration, and each
            entry owes a test that trips when it is exceeded. Two of the four have one now
            (TestARenameCostsExactlyOneZellijSubprocess, TestOnlyOneGoroutineWritesTheHost); the paint-
            frequency budget is the one with no test, and it is the one the tree violates. Count paints
            across a scripted background-tab flood and assert the bound, rather than asserting the bound in
            prose next to a mechanism that does not honour it.
          family: envelope-claim-unenforced
          round: 13
        - id: BR-58
          severity: Important
          title: The Core concepts table declares ResetSGR at hostty/reserve.go; it is defined at hostty/control.go:41
          detail: |-
            This is the 7th finding in family `plan-table-drift`. Do NOT fix the row -- the rule is what is
            missing. plan.md:239 states `ResetSGR / ReserveAndPaint | cmd/internal/hostty/reserve.go`;
            ReserveAndPaint is there, ResetSGR is at control.go:41. Same table, `right pane chrome` carries
            status `modified` while main-3.kdl still has no terminal-pane borderless=true (M4 unshipped).
            Rated Important, not Critical: the symbol exists and is exported from the declared package, so
            no consumer is misled about behaviour -- what is unchecked is the table's own claim.
            THE RULE, and it is the half of BR-54 that was not delivered: the new guard
            (TestNoPlannedRowSurvivesItsTickedMilestone) only regexes `planned - Mx` after Mx is ticked,
            and couchtty's contract filters #199's rows to `cmd/internal/couchtty/` paths -- so NO test
            reads a single one of M3's twelve new rows. Either grep each row's stated path for its stated
            symbol (which is what BR-54 asked for and what would have caught this in the same commit that
            wrote it), or drop the path and status columns so the table stops making claims nothing checks.
          family: plan-table-drift
          round: 13
        - id: BR-59
          severity: Minor
          title: When the renamed tab itself exits, the in-progress rename field vanishes from the row
          detail: |-
            Measured: with a rename open on the active tab, driving removeTab on that tab's id leaves the
            row as `[one]` -- no field at all -- while the stdin pump goes on routing keystrokes into the
            editor. stripModelLocked (run.go:1428) resolves the missing tab to `Tab: -1` and the renderer
            treats that as "mark nothing". The deleted renamePaneTitleLocked had an explicit
            `if !found { append("[rename: "+field+"]") }` branch for exactly this case; the move from the
            title to the strip did not carry it. The commit is a no-op afterwards, so the cost is brief
            blind typing rather than data loss -- but it is a branch lost in a move that was presented as a
            move.
          family: moved-surface-drops-a-case
          round: 13
        - id: BR-60
          severity: Minor
          title: stripRepaintCase.needsPty is set at one site and read at zero, and two imports are kept alive by blank vars
          detail: |-
            This is the 2nd finding in family `dead-test-scaffolding`, so state the rule: a test-harness
            field or declaration that no assertion reads is scaffolding that reads as protection.
            stripmutation_test.go:53 declares `needsPty`, :80 sets it, nothing reads it -- mustNewTab's
            t.Skipf does the work. :325-326 carry `var _ = io.Discard` and `var _ ptychild.Size`, which
            exist only to keep two otherwise-unused imports compiling. Delete the field and the imports;
            a struct field with no reader is the same shape as the "field set at zero call sites" the
            claimed-fix check exists to catch.
          family: dead-test-scaffolding
          round: 13
      boundary: M3
      blocked: true
    - "n": 14
      timestamp: "2026-09-08T14:26:24-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: not-addressed
          note: M4.3 still has no Alt+Shift+d step; the plan's own revision defers it to M4.
          round: 14
        - id: BR-9
          disposition: not-addressed
          note: writer_test.go:117 still splits one hand-chosen sequence at one index.
          round: 14
        - id: BR-53
          disposition: addressed
          note: Three sites rewritten and TestNoDeclarationCarriesTwoStackedGodocs added; its scope is Important 3.
          round: 14
        - id: BR-56
          disposition: addressed
          note: 'Mutation-verified: restoring HasPrefix(name,"terminal") fails on terminals/terminal-2/terminalwork for both consumers.'
          round: 14
        - id: BR-57
          disposition: addressed
          note: Instance fixed and mutation-verified; the rule it stated is only half delivered, raised anew below.
          round: 14
        - id: BR-58
          disposition: addressed
          note: Row corrected to control.go and the guard checks a DEFINITION, not a mention.
          round: 14
        - id: BR-59
          disposition: addressed
          note: 'Mutation-verified: forcing detached=false reddens TestARenameWhoseTabExitedStaysOnTheRow and a stripRepaintCase.'
          round: 14
        - id: BR-60
          disposition: addressed
          note: needsPty and both blank vars are gone; no `var _` remains in the package's tests.
          round: 14
      findings:
        - id: BR-61
          severity: Important
          title: The Core-concepts guard hardcodes the ACTIVE plan path, so make test breaks the moment this plan is archived
          detail: |-
            plantable_test.go:133 reads workshop/plans/000199-...-plan.md and t.Fatal(err)s on failure.
            sdlc close archives plans to workshop/history/plans/ (39 are already there), so the whole
            repo's suite goes red at M4.5, the next gate. This is the 2nd finding in family
            moved-surface-drops-a-case, so state the rule rather than editing the path: the repo
            ALREADY resolves a plan active-or-archived (couchtty/core_concepts_contract_test.go:272,
            findConceptPlans walks workshop/history when workshop/plans misses), and this guard is a
            deliberate re-implementation that dropped that case. THE RULE - a guard that reads a
            workshop artifact resolves it through one shared active-or-archived resolver; a guard that
            hardcodes a path under workshop/plans is asserting the issue will never close.
          family: moved-surface-drops-a-case
          round: 14
        - id: BR-62
          severity: Important
          title: The paint-frequency budget and the resize transition still have no test that trips when exceeded
          detail: |-
            This is the 6th finding in family envelope-claim-unenforced. BR-57 stated the rule -- every
            budget bullet is an enumeration and each entry owes a test that trips when exceeded -- and
            the round delivered the test for exactly the entry the finding named. Measured against the
            full termcmd suite with a compiler overlay - (a) run.go:863, changing `if chunk.rowDirty` to
            `if true`, i.e. a repaint on EVERY active-tab output chunk, which is the literal violation of
            ARCH-CONSTRAINTS bullet 1 ("a repaint happens on tab change, resize, and batch.RowDirty; NOT
            per output chunk"), leaves the suite green; (b) deleting m.paintStripInline() from
            inheritSize (run.go:1387), the ARCH-ORDER row "host resize -> re-Reserve, re-Paint", also
            green -- after which a resized pane keeps a scroll region computed from the OLD row count.
            For the enumeration's sake, replacing restoreTerminal's ResetRegion write (ARCH-ORDER's
            "Release on every exit path", pre-dating this window) is green too. THE RULE - write the
            enumeration itself as a table in the test file, one row per ARCH-CONSTRAINTS bullet and per
            ARCH-ORDER transition, each owning an assertion that fails when its bound is exceeded. The
            paint bound is a COUNT over a scripted chunk stream, not a boolean "did a paint happen":
            TestOnlyTheActiveTabsOutputDirtiesTheRow is the only test in the package that asks the
            negative question, and it asks it about one chunk.
          family: envelope-claim-unenforced
          round: 14
        - id: BR-63
          severity: Important
          title: Both new structural guards check less than they claim - one file of five, and no grouped declaration
          detail: |-
            This is the 5th finding in family acceptance-command-does-not-hold, so state the rule rather
            than widening two globs. stripmutation_test.go:199 does parser.ParseFile(fset, "run.go", nil, 0)
            while its own doc says it "is what makes a method added later announce itself" -- the package
            has five non-test .go files, and a terminalMux method added in strip.go, rename.go or a new
            file assigns m.tabs/m.active/m.rename invisibly. doccomment_test.go:86 returns no name for a
            GenDecl with more than one Spec, so cmd/internal/hostty/control.go:19-80 -- ONE grouped
            const block, and the file where M3 rewrote ResetSGR's and HomeAndClear's doc comments -- is
            not inspected at all by the guard written for exactly that class; its `checked == 0` floor
            still passes because the file's two funcs are counted. The plan's own 2026-09-07 revision
            already records this shape ("it read run.go only, while M3 adds strip.go to the same
            package") as one of three gaps that made M2's scan insufficient. THE RULE - a structural
            guard enumerates the package's non-test files and descends into grouped declarations, or its
            doc names the narrower scope it actually has, so nobody reads it as covering the class.
          family: acceptance-command-does-not-hold
          round: 14
        - id: BR-64
          severity: Minor
          title: RenameEditor.Field's doc justifies itself by two surfaces; the second was deleted in the same window
          detail: |-
            rename.go:64 says "One function, because there are now two surfaces -- the tab strip and the
            degraded pane title -- and a caret drawn two ways is a caret that disagrees with itself".
            renamePaneTitleLocked is gone and Field() has exactly one production caller (run.go:1441).
            3rd in family doc-states-planned-as-current. The rule cannot be mechanised the way the
            stacked-godoc one was -- TestNoDeclarationCarriesTwoStackedGodocs catches a doc RESTARTED,
            not a single opener whose stated fact died -- so record the prevalence instead: when a
            commit deletes a producer, every doc that cited it as a live reason is part of that commit.
          family: doc-states-planned-as-current
          round: 14
        - id: BR-65
          severity: Minor
          title: ReserveAndPaint copies Paint's body rather than composing it, and Reserve() now has no production caller
          detail: |-
            2nd in family copy-instead-of-extract. reserve.go:117 and :147 differ only by the SetRegion
            insertion; TestReserveAndPaintAgreesWithItsParts checks only that the region substring is
            present, so a change to Paint (a hide-cursor, a different erase) silently will not reach
            ReserveAndPaint. THE RULE - extract the shared tail (MoveTo + ResetSGR + ClearLine + text +
            ResetSGR) and let both spell only what differs. Separately, Reservation.Reserve() has zero
            production callers now that couch and termcmd both use ReserveAndPaint.
          family: copy-instead-of-extract
          round: 14
        - id: BR-66
          severity: Minor
          title: A new exported symbol in a shared package has no Core-concepts row, and nothing checks that direction
          detail: |-
            This is the 8th finding in family plan-table-drift. workbenchshortcut.TitleIdentifiesRightTerminal
            (shortcut.go:198) is new exported surface in a package two other components consume, with no
            row in the plan's table and no mention in atlas/architecture.md's paragraph about the
            predicate. The round-12 fix built the table->code direction
            (TestEveryCoreConceptRowNamesASymbolThatExists); nothing checks code->table. THE RULE - the
            table needs both directions, the machinery exists (couchtty's conceptInventory), and it is
            scoped to one package; widening it is pair#188. Until then, say so in the plan rather than
            leaving the table's completeness as an unstated claim.
          family: plan-table-drift
          round: 14
      boundary: M3
      blocked: true
    - "n": 15
      timestamp: "2026-09-08T14:46:06-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: not-addressed
          note: M4.3 still has no Alt+Shift+d step; six of nine borderless rungs stay unverified.
          round: 15
        - id: BR-9
          disposition: not-addressed
          note: writer_test.go:117 still splits one hand-chosen sequence at one index.
          round: 15
        - id: BR-61
          disposition: not-addressed
          note: tests/plan-superseded-facts-test.sh:45 still hardcodes workshop/plans/000199-…-plan.md and is wired into make test; measured 7 failures with the plan archived.
          round: 15
        - id: BR-62
          disposition: addressed
          note: All three mutations re-run in a scratch clone and all three now go red.
          round: 15
        - id: BR-63
          disposition: addressed
          note: 'Verified by mutation: a mutator in strip.go and a stacked godoc in control.go''s grouped const are both caught.'
          round: 15
        - id: BR-64
          disposition: addressed
          note: rename.go:64 now states one surface and explains the second was deleted.
          round: 15
        - id: BR-65
          disposition: addressed
          note: drawRow extracted; TestReserveAndPaintIsPaintPlusTheRegion asserts equality, not substring presence; Reserve() deleted.
          round: 15
        - id: BR-66
          disposition: addressed
          note: Table row and atlas paragraph added; the table now states it is checked table→code only, attributing the other direction to pair#188.
          round: 15
      findings:
        - id: BR-67
          severity: Important
          title: The issue's Done-when still carries a bullet its own Revisions marks struck, and omits the one that entry says replaced it
          detail: |-
            workshop/issues/000199-…md:170 still lists the scroll-position deliverable that :701
            records as "Struck"; the borderless=true bullet the same entry says was "Added to
            Done when" is absent; and the Spec at :113 still calls borderless a follow-on that
            the same entry moved into scope. This is the artifact sdlc close verifies against.
            Fourth in family, so state the rule rather than editing three bullets: this family's
            class guard already exists as tests/plan-superseded-facts-test.sh and is bounded to
            $PLAN, never reading the issue. Widen it to take (file, token, replacement, max_line)
            over both artifacts. The same enumeration catches the code-side instance in this
            window, run.go:1359 ("inheritSize does not write the pane today, but it becomes a
            writer in M3"), which paintbudget_test.go:71 now asserts the opposite of.
          family: doc-states-planned-as-current
          round: 15
        - id: BR-68
          severity: Important
          title: The Core-concepts guard reads only pipe rows, so the bullets under the table are unchecked and one names a method this commit deleted
          detail: |-
            Plan :266-267 says Reservation "answers ChildRows(), Reserve(), Release(),
            Paint(text)"; Reservation.Reserve() was deleted in fc318eb9, and reserve.go's own
            godoc now says "There is no bare Reserve()". plantable_test.go:175 skips every line
            not starting with "|", so the descriptive bullets immediately beneath the table --
            which name the same symbols at the same declared paths -- are invisible to the guard
            written for exactly that claim. Ninth in family: do not edit the bullet. The guard
            reads the whole "## Core concepts" section's backticked identifiers; goIdentifier
            already filters the non-symbols (escapes, zellij action names, issue refs), so the
            widening is close to free.
          family: plan-table-drift
          round: 15
        - id: BR-69
          severity: Minor
          title: Every probe registers teardown as a defer and then leaves through os.Exit, so a failed test-smoke leaks the session it created
          detail: |-
            probes/cursorsaveslots/main.go:72 defers session.Close() and then os.Exit(2)s at :80,
            :124 and :150; probes/zellijscrollregion/main.go:89 has the same shape at :97/:135/:138;
            cmd/probes/couchnestedrows/main.go defers delete-session (:396), os.RemoveAll(dataDir)
            (:387) and os.Remove(layout) (:392) then calls inconclusive(), which os.Exit(1)s -- and
            the path most likely to fire is "the session never came up". os.Exit skips defers, so a
            live zellij session named <prefix>-<pid> plus a temp data dir survive each failure, from
            make test-smoke. BR-51 made "a probe can only destroy a session it made" structural;
            nothing made "a probe always destroys the session it made" structural. THE RULE - a
            probe's main is func main() { os.Exit(run()) } and every diagnostic path returns a code,
            so cleanup is a defer in run. Mechanisable with the go/ast machinery this package
            already uses: fail when a function containing a defer also calls os.Exit. Measured
            prevalence: 3 probes, 11 such exit sites.
          family: exit-path-drops-cleanup
          round: 15
      boundary: M3
      blocked: false
    - "n": 16
      timestamp: "2026-09-08T15:49:37-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: not-addressed
          note: M4.3 (plan:663) still has no Alt+Shift+d step; the Done-when "Splits still work" bullet closes unverified.
          round: 16
        - id: BR-9
          disposition: not-addressed
          note: writer_test.go:127 still splits one hand-chosen sequence at one index; only ptychild's scanner has byte-at-a-time equivalence.
          round: 16
        - id: BR-13
          disposition: not-addressed
          note: termcmd now goes through NewReservation (run.go:1412,1422) but couchtty/console.go:922 still builds the struct literally.
          round: 16
        - id: BR-14
          disposition: not-addressed
          note: atlas/couch.md:228 still presents the reserved row as couch-owned; no mention of hostty.Reservation anywhere in the file.
          round: 16
        - id: BR-15
          disposition: not-addressed
          note: hostty/reserve.go:138-154 still states no caller obligation to sanitize or clamp the text it writes verbatim.
          round: 16
        - id: BR-17
          disposition: addressed
          note: The stated grep now returns only correction passages; the tokens are registered in plan-superseded-facts-test.sh and it fires (mutation-checked).
          round: 16
        - id: BR-20
          disposition: addressed
          note: probes/zellijscrollregion/main.go:134 uses zellijprobe.TailOf and the value is now a checked precondition of the verdict.
          round: 16
        - id: BR-21
          disposition: addressed
          note: atlas/index.md:17-46 states both probe homes with the two reasons; the per-probe targets exist in Makefile.local:96-108.
          round: 16
        - id: BR-22
          disposition: addressed
          note: Every probe is func main(){os.Exit(run())}; the go/ast guard goes red when a defer+os.Exit function is injected (verified).
          round: 16
        - id: BR-23
          disposition: addressed
          note: termcmd is a real second consumer since M3 (run.go:1412), so the atlas and package doc are now true in the present tense.
          round: 16
        - id: BR-24
          disposition: not-addressed
          note: 'plan:495 still fails in zsh ("no matches found: --include=*.go"); same unquoted form at plan:95 and in run.go''s paneTitleLocked doc.'
          round: 16
        - id: BR-26
          disposition: addressed
          note: Positive controls precede the assert-absents (writer_test.go:227, :388) and each was mutation-checked.
          round: 16
        - id: BR-27
          disposition: addressed
          note: runZellijCaptured folds captured stderr into the error; diagnostics have their own capped queue, separate from the paint slot.
          round: 16
        - id: BR-29
          disposition: addressed
          note: reportedUnused and the dead field are gone.
          round: 16
        - id: BR-30
          disposition: not-addressed
          note: run.go:168 still takes stdin and stdout; neither is referenced in the body.
          round: 16
        - id: BR-31
          disposition: not-addressed
          note: ARCH-CONSTRAINTS (plan:357-377) still declares no cost for the per-chunk FeedFraming, nor the unterminated-sequence stall.
          round: 16
        - id: BR-33
          disposition: not-addressed
          note: atlas/architecture.md:517-519 still says every console-originated write defers into one coalescing slot; diagnostics use a capped queue that survives a takeover. The stderr gap at :519 is filled; the guard was never extended to atlas.
          round: 16
        - id: BR-34
          disposition: addressed
          note: maxOwedDiag=64 with oldest-dropped (run.go:956-968); the envelope-declaration half rides with BR-31.
          round: 16
        - id: BR-36
          disposition: addressed
          note: M3.7(b) is automated in couchnestedrows with an escape-emitting flood; M4.3 is recorded at the operator's own granularity with what it does not itemise flagged in the Log.
          round: 16
        - id: BR-38
          disposition: not-addressed
          note: couchtty/reserve.go:143-151 still carries the sanitize and truncate doc paragraphs for functions that now live in rowtext.
          round: 16
        - id: BR-40
          disposition: addressed
          note: TestTheTakeoverResetIsLoadBearing uses a pure-digit replay that cannot terminate the stale CSI.
          round: 16
        - id: BR-42
          disposition: addressed
          note: The status column is now guarded for every row of this plan; the guard is currently RED on the M4 row - see the new Critical.
          round: 16
        - id: BR-43
          disposition: addressed
          note: The substring scan is gone; paneWriter has no Write method and the mux holds no other io.Writer to the pane.
          round: 16
        - id: BR-44
          disposition: addressed
          note: 'Decided per payload and documented at flushOwed (run.go:1000-1024): unbounded for the paint by choice, capped for diagnostics.'
          round: 16
        - id: BR-61
          disposition: addressed
          note: Verified by archiving the plan in a scratch copy - both Go guards and the shell script resolve it from workshop/history.
          round: 16
        - id: BR-67
          disposition: addressed
          note: 'Mutation-verified: reintroducing the struck Done-when bullet fails plan-superseded-facts-test.sh.'
          round: 16
        - id: BR-68
          disposition: addressed
          note: 'Mutation-verified: naming a deleted symbol in the Core-concepts prose fails TestEveryCoreConceptRowNamesASymbolThatExists.'
          round: 16
        - id: BR-69
          disposition: addressed
          note: 'Mutation-verified: a function with a defer that calls os.Exit fails TestNoProbeExitsPastItsOwnCleanup.'
          round: 16
      findings:
        - id: BR-70
          severity: Critical
          title: make test is RED at HEAD - the ticked M4 collides with the plan's `planned — M4` row, and the commit that ticked it ran nothing
          detail: |-
            go test ./... at 316bc86c fails TestNoPlannedRowSurvivesItsTickedMilestone
            (plantable_test.go:153): plan:314 still reads `planned — M4` while issue:191
            ticks M4. `git log -S'- [x] M4 —'` shows the tick landed only in 316bc86c,
            whose message records no test run; 5edb47cf's "make test exit 0 (195 ok)" was
            green precisely because the tick was not yet there. This is the 10th finding
            in family plan-table-drift, so do NOT just edit the row. THE RULE - an edit to
            a tracked artifact is an input to a guard, so the commit that ticks a
            milestone runs the suite exactly like a code commit, and the tick plus its
            table-status flip are ONE edit. Everything else failing in that run is the
            documented sandbox pty class; this one is pure file reads.
          family: plan-table-drift
          round: 16
        - id: BR-71
          severity: Important
          title: The guard that catches the Critical goes silent when sdlc close archives the issue, so closing hides the failure instead of fixing it
          detail: |-
            plantable_test.go:40 resolves the PLAN active-or-archived (BR-61's fix) but
            :116 enumerates issues with a bare Glob over workshop/issues/*.md. Measured in
            a scratch copy - move issue and plan to workshop/history and the currently-RED
            TestNoPlannedRowSurvivesItsTickedMilestone reports ok with `planned — M4`
            still in the table. sdlc close performs exactly that move, so the close would
            record green over the drift and archive a plan that contradicts the code. This
            is the 3rd finding in family moved-surface-drops-a-case. THE RULE - one shared
            active-or-archived resolver for EVERY workshop artifact a guard reads, issues
            included; a guard whose coverage can be removed by archiving is asserting the
            issue will never close.
          family: moved-surface-drops-a-case
          round: 16
        - id: BR-72
          severity: Important
          title: M4 falsified four "the terminal pane is framed" statements and swept none, two of them in the files it edited
          detail: |-
            This is the 11th finding in family plan-table-drift, so state the rule rather
            than patching four sentences. Measured prevalence, all live at HEAD:
            atlas/architecture.md:401 ("Pane frame asymmetry ... the agent pane and
            layout-3 terminal render frames ... the draft pane opts out") - the section a
            reader lands on for this subject, now the opposite of the paragraph M4 added
            at :556; atlas/architecture.md:381 ("while keeping frames");
            zellij/config.kdl:12-14 ("Split terminal panes stay pinned and keep their
            frames (the frame is the only visible divider between the split halves and
            carries the tab title)") - two lines above the bullet M4 rewrote;
            zellij/layouts/main-3.kdl:22 ("keeping frames (divider, tab title, scroll
            indicator)") - in the file M4 edited nine times. BR-33 asked for the mechanism
            and BR-67 extended tests/plan-superseded-facts-test.sh to the issue plus one
            run.go comment. THE RULE - that script's file list IS the enumeration of
            artifacts asserting this design, and it must name every artifact class: plan,
            issue, atlas, layout/config comments, code comments. A pair registered for one
            file while the same superseded fact lives in four others is coverage that
            cannot fire.
          family: plan-table-drift
          round: 16
        - id: BR-73
          severity: Minor
          title: 'Two cosmetic slips in the M2 extraction: redundant parens and an out-of-order manifest entry'
          detail: |-
            cmd/internal/couchtty/menu_render.go:625 reads
            `rowtext.SanitizeAndFit((line), width)`, and the rowtext import at :5 sits
            inside the stdlib group unlike every other file in the package.
            cmd/internal/artifactpath/manifest.go:631 inserts
            cmd/internal/rowtext/rowtext.go between procutil and ptychild, breaking the
            list's sort order.
          family: formatting-drift
          round: 16
      blocked: true
    - "n": 17
      timestamp: "2026-09-08T16:41:27-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: not-addressed
          note: M4.3 still has no Alt+Shift+d step and the operator record does not mention the split; now load-bearing because finding (a) shows the guard is blind on those same rungs and 78f8c046 restored frames there for a divider.
          round: 17
        - id: BR-9
          disposition: not-addressed
          note: writer_test.go:117 still splits one hand-chosen sequence at one index.
          round: 17
        - id: BR-13
          disposition: not-addressed
          note: Premise partly overtaken - run.go:1412/1422 are production callers now - but couchtty/console.go:922 still bypasses the door via bottomReservation.
          round: 17
        - id: BR-14
          disposition: not-addressed
          note: atlas/couch.md:228 unchanged; still no pointer to hostty.Reservation from the section a reader lands on.
          round: 17
        - id: BR-15
          disposition: not-addressed
          note: No caller-obligation line anywhere in cmd/internal/hostty; grep for sanitiz/rowtext in that package returns nothing.
          round: 17
        - id: BR-24
          disposition: not-addressed
          note: 'plan:528 unchanged. Measured this round in the repo''s zsh - "(eval):1: no matches found: --include=*.go", exit 1, pipeline never runs.'
          round: 17
        - id: BR-30
          disposition: not-addressed
          note: run.go:168 still takes stdin and stdout; neither is referenced in the body.
          round: 17
        - id: BR-31
          disposition: not-addressed
          note: The envelope at plan:390-410 still budgets paints only; no entry for the second full parse per chunk or the unterminated-OSC stall.
          round: 17
        - id: BR-33
          disposition: not-addressed
          note: The mechanism half landed and fires, but BR-33's own paragraph is unchanged - atlas/architecture.md:517-519 still says a mid-sequence write goes into a single coalescing slot, false for diagnostics since 34918a3c gave them owedDiag.
          round: 17
        - id: BR-38
          disposition: not-addressed
          note: couchtty/reserve.go:143-151 still ends in two paragraphs documenting sanitize/truncate; doccomment_test.go's roots exclude couchtty and it checks stacked godocs, not orphaned ones.
          round: 17
        - id: BR-70
          disposition: addressed
          note: Verified at HEAD - the row reads "modified - M4", the guard passes, and go test ./... is green apart from the documented sandbox pty class.
          round: 17
        - id: BR-71
          disposition: addressed
          note: Verified by construction - archived issue and plan to workshop/history in a scratch tree, reintroduced the drift, guard still fired.
          round: 17
        - id: BR-72
          disposition: not-addressed
          note: One of four sites fixed. Live at HEAD - atlas/architecture.md:381 "while keeping frames", zellij/config.kdl:11-14 "keep their frames (the frame is the only visible divider between the split halves)", zellij/layouts/main-3.kdl:22 "keeping frames (divider, tab title, scroll indicator)". The registered config/layout token covers a different sentence in the same file.
          round: 17
        - id: BR-73
          disposition: not-addressed
          note: All three present - menu_render.go:625 parens, its rowtext import in the stdlib group, manifest.go:631 out of order.
          round: 17
      findings:
        - id: BR-74
          severity: Important
          title: The borderless guard's two-line window lets a neighbour satisfy the check, so 3 of its 9 rungs cannot fail
          detail: |-
            This is the 4th finding in family `uncovered-negative-assertion`, so state the
            rule rather than patching the window. THE RULE - a per-item guard must bind its
            evidence to the item; matching inside a window that spans the next item lets a
            neighbour satisfy the assertion, and the mutation check must cover every
            position class (sole, first-of-pair, last-of-pair) rather than one
            representative. Measured: terminalborderless_test.go:47-51 builds
            `window := line + "\n" + lines[i+1]`, and the split rungs are adjacent
            `pane name="terminal"` lines; deleting ` borderless=true` from main-3.kdl:138 in
            both source and mirror leaves the test ok. Same shape at :165 and :191, which
            are exactly the rungs BR-8 says were never exercised by hand. All nine sites
            carry the attribute on their own line, so the window is unnecessary - delete it.
            Greppable form of the class: a guard that constructs evidence from
            `lines[i]+lines[i+1]` and then asserts on the concatenation.
          family: uncovered-negative-assertion
          round: 17
        - id: BR-75
          severity: Important
          title: The cursor-save gate tracks DECSC only; the aliasing SCOSC form the repo's own probe names is untracked
          detail: |-
            This is the 4th finding in family `external-input-assumed-wellformed`, so the
            rule, not the site. THE RULE - when this code models a terminal control by its
            byte form, it must enumerate EVERY byte form the terminal accepts for that
            control, and the enumeration is greppable because the repo already writes both
            forms down. BR-37 was the same rule for C0 versus C1 in rowtext.Sanitize; this
            is DECSC (ESC 7 / ESC 8) versus SCOSC (CSI s / CSI u) in
            ptychild/screen.go:439-455, where the CSI switch at :503 handles only 'r' and
            'J'. probes/cursorsaveslots/main.go:3 names both forms in its first sentence, so
            the sibling was enumerated before the fix was written and then not covered. A
            child using CSI s holds the same one slot the code's own comment says terminals
            keep, HoldsCursorSave stays false, and the paint lands inside the pair - the
            exact operator symptom M3 exists to remove. Fix at the class: classify every
            save/restore spelling, or extend the probe to answer the child-side question and
            record the negative where the gate is defined.
          family: external-input-assumed-wellformed
          round: 17
        - id: BR-76
          severity: Minor
          title: BR-70's rule is recorded only in a commit message, so nothing will read it next time
          detail: |-
            This is the 12th finding in family `plan-table-drift`. The round's own commit
            states the rule well - a tracked-artifact edit is an input to a guard, so the
            commit that ticks a milestone runs the suite, and the tick plus its table-status
            flip are one edit - but states it in `git log` only. workshop/lessons.md took 159
            lines in this window and none of them is that rule, which AGENTS.md section 4
            makes an obligation of running a review. A rule that lives only in a commit
            message is the same defect as a sweep that depends on remembering to sweep.
          family: plan-table-drift
          round: 17
      blocked: true
    - "n": 18
      timestamp: "2026-09-08T18:22:51-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: not-addressed
          note: plan:696-702 still has no Alt+Shift+d step; and splitTerminalDown (run.go:537) creates the split pane with no --borderless while tests/term-pane-shortcuts-test.sh:255 forbids one, so "each half draws its own strip" is unverified by hand and unasserted by test.
          round: 18
        - id: BR-9
          disposition: not-addressed
          note: writer_test.go:117 still splits one hand-chosen sequence at one index; no parameterisation over indices.
          round: 18
        - id: BR-13
          disposition: not-addressed
          note: 'couchtty/console.go:922 still returns hostty.Reservation{Rows: rows, Edge: hostty.EdgeBottom} directly.'
          round: 18
        - id: BR-14
          disposition: not-addressed
          note: grep for "Reservation" in atlas/couch.md returns zero hits; the reserved-row section still reads as couch-owned.
          round: 18
        - id: BR-15
          disposition: not-addressed
          note: No caller obligation stated anywhere in cmd/internal/hostty; grep for sanitiz/rowtext in that package returns one unrelated line about row clamping.
          round: 18
        - id: BR-24
          disposition: not-addressed
          note: 'plan:528 unchanged; re-measured in the repo''s zsh this round -- "(eval):1: no matches found: --include=*.go", exit 1, pipeline never runs. Same shape at run.go:1541.'
          round: 18
        - id: BR-30
          disposition: not-addressed
          note: run.go:168 still takes stdin and stdout -- and panes -- none of which the body references.
          round: 18
        - id: BR-31
          disposition: not-addressed
          note: The envelope still budgets paints and subprocesses only; no entry for the per-chunk FeedFraming second parse or the unterminated-OSC stall.
          round: 18
        - id: BR-33
          disposition: addressed
          note: atlas:518-523 now scopes coalescing to a paint and the takeover drop to the paint; the ban token was the literal prior text and is registered at plan-superseded-facts-test.sh:156. The atlas still does not state that diagnostics queue and survive a takeover -- worth one sentence, not a re-raise.
          round: 18
        - id: BR-38
          disposition: not-addressed
          note: couchtty/reserve.go:143-152 still ends in the two doc paragraphs for sanitize and truncate, which now live in rowtext.
          round: 18
        - id: BR-72
          disposition: not-addressed
          note: 'Two more live sites, and the class hole is the point: tests/term-pane-shortcuts-test.sh:253-257 is a make-test guard ENFORCING "split panes keep zellij default frames", a sixth artifact class the enumeration never names; atlas/architecture.md:711 still calls the frame the visible divider between split halves. The sweep also left main-3.kdl:22-26 and atlas:381 mid-sentence.'
          round: 18
        - id: BR-73
          disposition: not-addressed
          note: All three present at HEAD -- menu_render.go:625 redundant parens, its rowtext import inside the stdlib group at :5, manifest.go:631 out of sort order.
          round: 18
        - id: BR-74
          disposition: addressed
          note: 'Mutation-verified independently: deleting borderless=true from each of the nine rungs in source and mirror reds the guard in all nine cases, including sole, first-of-pair, last-of-pair and block-opening positions.'
          round: 18
        - id: BR-75
          disposition: not-addressed
          note: 'The named site is fixed and pinned (revert-verified). The RULE is not: measured at HEAD, "\x1b[?1048h" and "\x1b[?1049h" both leave HoldsCursorSave false -- and screen.go:492 actively clears the flag on the alt-screen transition, false for 1049 which is defined as DECSC-then-switch and is how nearly every full-screen child takes the slot. Parameter-free CSI s is also read as SCOSC with ?69/DECLRMM untracked. Write the enumeration as a table with one assertion per spelling, and let probes/cursorsaveslots answer the zellij-side premise.'
          round: 18
        - id: BR-76
          disposition: not-addressed
          note: lessons.md's two new 2026-09-08 entries are adjacent rules (mutation-check the guard; stating a class is not closing it) but not BR-70's; that rule is still only in git log.
          round: 18
      findings:
        - id: BR-77
          severity: Important
          title: The cursor-save precondition landed in termcmd's private predicate, not at the shared Reservation door, so couch paints unguarded
          detail: |-
            This is the 3rd finding in family `wrong-seam-named`, so the rule, not the
            site. hostty.Reservation.Paint / ReserveAndPaint (reserve.go:119,154) emit
            \x1b7 ... \x1b8 -- a property of the MECHANISM this issue lifted into the
            shared package. The precondition M3 measured for it ("never write while the
            child holds a save") was implemented as terminalMux.unsafeToPaint
            (run.go:949), private to one consumer. The other consumer of the same
            primitive, couchtty/console.go:1062 via writeOwn (:1005-1013), gates on
            MidSequence() alone; grep -rn HoldsCursorSave cmd/internal/couchtty returns
            nothing, and couch's children are alt-screen agent TUIs, so the identical
            clobber is reachable there with no guard. THE RULE - a precondition that is
            a property of a shared mechanism belongs at that mechanism's seam, stated so
            every consumer must satisfy it, not in whichever consumer happened to
            discover it. The enumeration is one grep
            (`ReserveAndPaint\|\.Paint(` over cmd, minus tests) and yields exactly two
            production call sites, so a guard over that set is cheap. Same door as
            BR-15's undocumented sanitize obligation - both are obligations of
            hostty.Paint recorded only in the callers that learned them (ARCH-DRY).
          family: wrong-seam-named
          round: 18
      blocked: true
    - "n": 19
      timestamp: "2026-09-08T18:46:39-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: not-addressed
          note: plan:696-702 still has no Alt+Shift+d step; the split rungs remain hand-unverified.
          round: 19
        - id: BR-9
          disposition: not-addressed
          note: writer_test.go:117 still splits one hand-chosen sequence at one index.
          round: 19
        - id: BR-13
          disposition: not-addressed
          note: 'console.go:922 still returns hostty.Reservation{Rows: rows, Edge: hostty.EdgeBottom} directly.'
          round: 19
        - id: BR-14
          disposition: not-addressed
          note: grep Reservation in atlas/couch.md returns nothing; hostty appears only at :213 and :812.
          round: 19
        - id: BR-15
          disposition: not-addressed
          note: No sanitize/clamp obligation in cmd/internal/hostty; the same door now also omits the cursor-save precondition.
          round: 19
        - id: BR-24
          disposition: not-addressed
          note: plan:528 unchanged; the unquoted --include=*.go still aborts in the repo's zsh.
          round: 19
        - id: BR-30
          disposition: not-addressed
          note: run.go:168 still takes panes, stdin and stdout; none is referenced in the body.
          round: 19
        - id: BR-31
          disposition: not-addressed
          note: plan:390-410 still budgets paints and subprocesses only; no entry for the per-chunk FeedFraming or the stall.
          round: 19
        - id: BR-38
          disposition: not-addressed
          note: couchtty/reserve.go:143-152 still ends in the sanitize and truncate doc paragraphs for functions now in rowtext.
          round: 19
        - id: BR-72
          disposition: addressed
          note: Sixth class registered and the sweep is complete; verified the new GUARD pair fires and the script exits 1.
          round: 19
        - id: BR-73
          disposition: not-addressed
          note: All three present; plus a third instance from this round's sweep, the line-wrap artifact at atlas:517-521.
          round: 19
        - id: BR-75
          disposition: addressed
          note: Enumeration is a table and mutation-verified; its 1049 arm creates the new Critical, raised separately.
          round: 19
        - id: BR-76
          disposition: not-addressed
          note: lessons.md untouched this round; BR-70's rule still lives only in git log.
          round: 19
        - id: BR-77
          disposition: not-addressed
          note: Reverting all four couch call sites to MidSequence leaves the suite green, the obligation is still unstated at hostty.Paint, and the swap as landed spins.
          round: 19
      findings:
        - id: BR-78
          severity: Critical
          title: The predicate swap reached four of five sites; the drain loop kept the old one and spins forever
          detail: |-
            couchtty/console.go:1202 still continues on MidSequence() while :1111 defers on
            !SafeToPaint(), so a save taken during the drain makes the loop re-pop a chunk it
            cannot progress on. Reproduced deterministically (drain never returns in 3s; the same
            test returns instantly at f233b29d^). couch's Run goroutine is its only writer, so this
            is a frozen console at 100% CPU. THE RULE - a defer condition and the loop that drains
            it are ONE pair and must be the same expression; after replacing a gate predicate,
            grep the old name (one production site remains outside ptychild).
          family: moved-surface-drops-a-case
          round: 19
        - id: BR-79
          severity: Critical
          title: The new 1049 arm holds the paint gate closed for the child's whole alt-screen lifetime, in both consumers
          detail: |-
            screen.go:527-531 makes cursorSaved true from ?1049h until ?1049l, and SafeToPaint
            (:181) is false throughout. Measured - in termcmd the strip never paints again after
            \x1b[?1049h (five paint rounds, nothing reaches the pane; reverting only that arm
            paints), and in couchtty repaint writes nothing. So less/vim/man freeze the strip and
            make Alt+r blind typing, the symptom M3 exists to remove. flushOwed's justification for
            unbounded deferral ("paid by the child's own DECRC") no longer covers this case, and
            the premise that ?1049h takes the SAME slot our paint clobbers is asserted from the
            spec while probes/cursorsaveslots, built for exactly this question, was not extended.
            This is the 3rd finding in family deferred-work-lacks-own-trigger. THE RULE - a defer
            condition is only admissible with a release the deferrer can observe will arrive;
            widening it to a condition the child controls indefinitely requires either a narrower
            condition or a bounded release, and the measurement that settles which.
          family: deferred-work-lacks-own-trigger
          round: 19
        - id: BR-80
          severity: Important
          title: SafeToPaint is the one new exported shared-package symbol with no plan row, no atlas sentence and no test
          detail: |-
            This is the 13th finding in family plan-table-drift, so the rule, not the site.
            Measured - the window adds 16 exported declarations under cmd/internal/{hostty,
            ptychild,rowtext,workbenchshortcut}; 15 appear in the plan's Core-concepts material and
            SafeToPaint appears zero times, in the plan, in the atlas, and in any test.
            atlas/architecture.md:604-615 still names HoldsCursorSave as the bit that says so. The
            Core-concepts guard is table-to-code only, so this direction cannot fire. THE RULE -
            the artifact enumeration BR-72 wrote for SUPERSEDED facts needs its mirror for ADDED
            shared surface (table row, atlas sentence, a test naming the symbol), and that
            enumeration is producible from the diff's new exported declarations rather than from
            memory.
          family: plan-table-drift
          round: 19
      blocked: true
    - "n": 20
      timestamp: "2026-09-08T19:15:27-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: not-addressed
          note: plan:696-702 still has no Alt+Shift+d step; and the config.kdl comment added this window asserts split halves are borderless, which depends on the exact_panes=4 swap layout applying the attribute to a pane splitTerminalDown creates with no --borderless. Unverified by hand and unasserted by test.
          round: 20
        - id: BR-9
          disposition: not-addressed
          note: writer_test.go:117 TestPaintDefersMidSequenceAndIsOwed still splits "before\x1b[3" at one hand-chosen index; no parameterisation.
          round: 20
        - id: BR-13
          disposition: not-addressed
          note: 'couchtty/console.go:921 bottomReservation still returns hostty.Reservation{Rows: rows, Edge: hostty.EdgeBottom} directly.'
          round: 20
        - id: BR-14
          disposition: not-addressed
          note: grep Reservation in atlas/couch.md returns nothing; hostty appears only at :213 and :812.
          round: 20
        - id: BR-15
          disposition: not-addressed
          note: grep -rn "sanitiz|rowtext" cmd/internal/hostty returns one unrelated line about row clamping; Paint/ReserveAndPaint still state no caller obligation.
          round: 20
        - id: BR-24
          disposition: not-addressed
          note: plan:528 unchanged; re-measured in the repo's zsh this round, the unquoted --include=*.go aborts with "no matches found" before the pipeline runs.
          round: 20
        - id: BR-30
          disposition: not-addressed
          note: run.go:168 runDecision still takes panes, stdin and stdout; none is referenced in the body.
          round: 20
        - id: BR-31
          disposition: not-addressed
          note: The envelope still budgets paints and subprocesses only; no entry for the per-chunk FeedFraming second parse or the unterminated-OSC stall.
          round: 20
        - id: BR-38
          disposition: not-addressed
          note: couchtty/reserve.go:143-152 still ends in the two doc paragraphs for sanitize and truncate, which live in rowtext.
          round: 20
        - id: BR-73
          disposition: not-addressed
          note: All three present at HEAD - menu_render.go:625 redundant parens, its rowtext import inside the stdlib group at :5, manifest.go:631 out of sort order.
          round: 20
        - id: BR-76
          disposition: not-addressed
          note: lessons.md untouched since 82976ec6; grep for the tick-runs-the-suite rule returns nothing. BR-70's rule still lives only in git log.
          round: 20
        - id: BR-77
          disposition: not-addressed
          note: One of the three sub-claims cleared (the spin). Measured this round - reverting console.go:1011/1111/1141 to MidSequence leaves the couchtty suite GREEN, and cmd/internal/hostty still states no cursor-save precondition at Paint/ReserveAndPaint.
          round: 20
        - id: BR-78
          disposition: addressed
          note: console.go:1207 now asks SafeToPaint; I reproduced the spin deterministically and it is gone. The oracle chosen for it is raised separately.
          round: 20
        - id: BR-79
          disposition: addressed
          note: screen.go:201 carve-out is pinned - reverting it turns TestTheSaveGateDoesNotFreezeThroughAnAltScreenSession red. The new dependency it introduces is raised separately.
          round: 20
        - id: BR-80
          disposition: not-addressed
          note: Tests now name SafeToPaint, so that third is closed. plan:277/:343 and atlas/architecture.md:614 still name HoldsCursorSave as the bit that decides, which is now a rule the code deliberately does not follow.
          round: 20
      findings:
        - id: BR-81
          severity: Critical
          title: altScreen became a safety bit at HEAD; RIS never clears it, so the save gate is off for good after a reset
          detail: |-
            SafeToPaint (screen.go:201) now reads !(cursorSaved && !altScreen), promoting altScreen
            from a bit with zero production consumers to the one that decides whether the paint gate
            closes. Two classify arms were not re-enumerated for the new role, both measured.
            (1) The RIS arm (screen.go:475-479) clears cursorSaved and sets rowDirty but not
            altScreen: "\x1b[?1049h" then "\x1bc" then "\x1b7" leaves altScreen=true and
            SafeToPaint()=true while the child holds a save on the NORMAL screen -- M3's exact
            operator-visible symptom, reachable after any reset/tput reset, and persisting until the
            child next enters and leaves an alt screen. (2) The 1047/47 arm keeps cursorSaved set on
            purpose, with a comment saying clearing it "would hand a console permission to paint
            inside a save the child still holds"; measured, "\x1b7" then "\x1b[?1047h" leaves
            HoldsCursorSave()=true and SafeToPaint()=true, handing exactly that permission for a save
            taken on, and restored to, the normal screen. THE RULE - when a field becomes
            safety-load-bearing its full transition set must be re-enumerated, and the enumeration
            already exists here: TestEverySaveSlotSpellingIsAccountedFor (screen_test.go:519) has one
            axis (HoldsCursorSave, from empty/held). Add a "from alt screen" starting state and a
            SafeToPaint column and both instances fail immediately (ARCH-ORDER).
          family: state-bit-promoted-without-enumeration
          round: 20
        - id: BR-82
          severity: Important
          title: The takeover mints a fresh Screen, so the alt-screen carve-out's premise is false after every switch
          detail: |-
            This is the 2nd finding in family takeover-resets-framing, so the rule, not the site.
            termcmd/run.go:894 and couchtty/console.go:994 both do hostScan = ptychild.Screen{} on a
            screen replacement, and the replay only re-establishes ?1049h while it is still inside the
            bounded ring. Measured - "\x1b[?1049h", then Screen{}, then "\x1b[?1048h" gives
            SafeToPaint()=false: BR-79's indefinite freeze, reached through the takeover rather than
            through the predicate. THE RULE - a takeover resets the scanner to "knows nothing", so
            every bit the gate reads needs a defined meaning under unknown. MidSequence degrades
            safely because false is the conservative answer; a bit whose false value WIDENS the gate
            does not, and must be re-established from the replay or excluded from the gate. The
            enumeration is one grep for `= ptychild.Screen{}` and yields exactly the two sites.
          family: takeover-resets-framing
          round: 20
        - id: BR-83
          severity: Important
          title: BR-78's Critical is pinned by grepping console.go, justified by a claim that is measurably false
          detail: |-
            console_test.go:864 asserts that flushDeferredNotifications' source text does not contain
            "hostScan.MidSequence()", with a comment saying "reproducing the spin requires the exact
            interleaving that made it a Critical rather than a flake". Measured false - a 35-line,
            single-goroutine, deterministic reproduction exists (feed the active actor a batch of
            [{Bytes:"\x1b[31"},{Notification},{Bytes:"\x1b7"},{Notification}], let it defer at the
            mid-sequence, then complete it): it returns instantly at HEAD and hangs the full 3s
            timeout with only line 1207 reverted. The grep also cannot catch the same defect at the
            other defer/drain pair (termcmd's owed paint), goes red on an extract-to-helper refactor
            that changes nothing, and stays green for a differently-spelled wrong condition. THE RULE
            - a source-text guard is admissible when the property IS textual (plan tables, doc
            comments, import order); when the defect is reachable in-process, the pin is a test that
            reaches it, and the fix's own commit message is not where that judgement gets recorded.
          family: source-text-proxy-oracle
          round: 20
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

## Round 9 — 2026-09-07T23:06:21-07:00 (claude) — BLOCKED

### Disposed

- BR-8 — not-addressed — M4 untouched in this window; M4.3 still has no Alt+Shift+d step while the Done-when still requires two right-pane halves each drawing a strip.
- BR-9 — not-addressed — writer_test.go:115 still splits one hand-chosen index of one sequence.
- BR-25 — not-addressed — No code change. Reproduced twice at HEAD with overlay tests - deterministic (full queue, handleChunk parked >3s) and under load (writer never drained again). run.go:772 -> 1088 -> 1303 -> 922.
- BR-26 — not-addressed — Measured green under mutation at HEAD - m.owed = nil (run.go:780), runZellijCaptured's SanitizeAndFit (:1447), resizeThroughWriter (:264); plan-superseded-facts-test.sh:65 still bans a string git log -S finds in no revision of the plan.
- BR-27 — not-addressed — Log half still absent - run.go:462,468,1142,1234 discard the error, so a failing wheel tick or rename is still completely silent while reportError sits one line away.
- BR-29 — not-addressed — run_test.go:941 fakeRuntime.reportedUnused still present and uncalled; terminalMux.stderr is still set at run.go:683 and read by nothing.
- BR-30 — not-addressed — run.go:168 runDecision still takes stdin and stdout and uses neither.
- BR-31 — not-addressed — ARCH-CONSTRAINTS unchanged. The BR-35 fix narrowed the second parse to the active tab, which is the fact to declare.
- BR-33 — not-addressed — atlas:501-508 still says a single coalescing slot dropped by a takeover; :527 says diagnostics are gated like any other write, which C2 falsifies; :532 still lists M3's strip as a current consumer. The superseded-facts test still covers only the plan.
- BR-34 — not-addressed — run.go:819 still appends to owedDiag unbounded; ARCH-CONSTRAINTS declares no budget. Same gap covers diagnosticWidth = 200, whose stated basis (far short of wrapping) is false for an 80-column pane while m.cols is on the receiver.
- BR-35 — addressed — Mutation-verified both directions rather than taken from the commit - moving FeedFraming out of the isActive branch and deleting the replay feed each redden their own subtest.
- BR-36 — not-addressed — Evidence half corrected honestly (Log + M2.5), but the correction defers the gate's manual acceptance to M3.7 and plan:589 has no step that enters the gate path - no second tab, no escape-emitting child, no deferral counted.
- BR-37 — addressed — Reverting the C1 clause at rowtext.go:44 reddens four subtests; raw 8-bit spellings degrade to U+FFFD (measured). Residual, not re-raised - menu_render.go:625 still open-codes Fit(Sanitize(...)).
- BR-38 — not-addressed — couchtty/reserve.go:143-152 still ends in doc comments for functions that moved to rowtext, and this window added the same shape at run.go:1410-1418, where two consts now sit between runZellijCaptured's prose and runZellijCaptured.

### Raised

- **BR-39** [Critical] `validating-door-bypassed` The takeover writes owed diagnostics straight to the pane after feeding the replay to the gate, so a diagnostic lands inside the replay's open sequence
  This is the 2nd finding in family validating-door-bypassed, so the deliverable is the rule, not the site. run.go:792-793 writes each queued diagnostic with m.stdout.Write, immediately after run.go:791 feeds chunk.replay to hostScan - so when the replay ends mid-sequence (the case run.go:787-790 exists to handle, and which writer_test.go:419 asserts is real), MidSequence is true and the write goes out anyway. Reproduced with an overlay test - a diagnostic queued while the child was mid-sequence, then redrawTab("restored\x1b[3"), yields "x\x1b[3\x1b[1;1H\x1b[Jrestored\x1b[3pair term: queued failure\r\n" on the pane, the diagnostic's leading bytes swallowed as CSI parameters. Reachable: a failed rename or a failed zellij action while the child is mid-escape, then any tab switch. The takeover's own write at :786 is NOT the same case and must stay exempt - it begins with ESC, which cancels a pending CSI in xterm-class terminals, which is why couch's takeover writes ungated; that exemption should be stated rather than re-derived. The rule - every byte reaching m.stdout from the writer loop passes through the gate-consulting door (writeOwn / writeDiag), or carries a written exemption naming why the gate does not apply - covers the enumerable set `grep -n 'm\.stdout' cmd/internal/termcmd/run.go`: today :786 (exempt, takeover), :793 (the defect), :826 (the child stream itself), :839/:857/:868/:876 (inside the doors), :1321 (teardown, exempt). The test that pins the rule enumerates the console-originated write paths and asserts each defers while the gate is mid-sequence, which is what the current suite does for paints and not for the takeover-carried diagnostics.

## Round 10 — 2026-09-07T23:27:09-07:00 (claude) — BLOCKED

### Disposed

- BR-25 — addressed — Verified by revert, not by the commit message: restoring redrawTab in removeTab reddens TestAChildExitingWithAFullBufferDoesNotWedgeThePane at the 3s timeout. The rewritten test is deterministic (buffer saturated, no loop running).
- BR-39 — not-addressed — Instance fixed and mutation-verified (reverting writeDiag to m.stdout.Write reproduces the exact predicted bytes). Class half absent: applyTakeover's own ungated write at run.go:826-827 carries no written exemption, and no test enumerates the console-originated doors.
- BR-26 — not-addressed — Re-measured at HEAD. Green under all three mutations: m.owed = nil (run.go:822), runZellijCaptured's SanitizeAndFit, resizeThroughWriter. plan-superseded-facts-test.sh:65 bans a string git log -S finds in no revision of the plan, so that registered pair can never fire.
- BR-27 — not-addressed — Log half absent. run.go:462,468,1165,1257 still discard the error; there is no log sink, so a failing wheel tick or rename is completely silent where pre-M2 it printed. run.go:1437's own comment states this as the reason capture exists. Note run.go:1165 is on the writer goroutine, so its fix needs inline writeDiag, not reportError.
- BR-33 — not-addressed — atlas:502-503 still describes every console-originated write as one coalescing slot, false for diagnostics; atlas:532-534 lists pair term's strip as a current rowtext consumer when it is M3. tests/plan-superseded-facts-test.sh still covers only the plan file.
- BR-34 — not-addressed — run.go:848 still appends to owedDiag with no cap; ARCH-CONSTRAINTS declares no budget. diagnosticWidth = 200 claims to be far short of wrapping while m.cols is on the receiver.
- BR-31 — not-addressed — ARCH-CONSTRAINTS unchanged. Screen.feedFraming is a per-byte loop run over every active chunk in addition to ptychild's own parse; the BR-35 fix narrowed it to the active tab, which is the fact to declare. Same gap covers the zellij exec now running on the sole writer goroutine in removeTab.
- BR-36 — not-addressed — M2.5 and the Log are corrected honestly, but M3.7(b) still specifies yes, which emits no ESC, so the deferral path still cannot be entered; no observable is named and no deferral is counted. M4.3 unchanged.
- BR-38 — not-addressed — couchtty/reserve.go:143-152 still ends in doc comments for sanitize and truncate, which live in rowtext now. This window added the same shape at run.go:1431-1441, where two consts sit between runZellijCaptured's prose and runZellijCaptured.
- BR-29 — not-addressed — run_test.go:941 fakeRuntime.reportedUnused still present and uncalled. terminalMux.stderr is set at run.go:683 and read nowhere (grep for m.stderr returns nothing).
- BR-30 — not-addressed — run.go:168 runDecision still takes stdin and stdout and uses neither; panes is likewise unused in the body.
- BR-8 — not-addressed — M4 untouched in this window; M4.3 still has no Alt+Shift+d step while the Done-when still requires two right-pane halves each drawing a strip.
- BR-9 — not-addressed — writer_test.go:115-116 still splits one hand-chosen index of one sequence.

### Raised

- **BR-40** [Important] `uncovered-negative-assertion` The takeover's scan reset -- couch's third gate rule, M2.3 step 1's headline -- can be deleted with the whole suite green
  This is the 3rd finding in family uncovered-negative-assertion, so the deliverable is the rule, not the site; it also shows BR-26's stated sweep list ("the four sites in writer_test.go plus run_test.go:910") was an enumeration written from memory, since this site is not on it. Measured in a scratch copy: deleting m.hostScan = ptychild.Screen{} (run.go:822) leaves ./cmd/internal/termcmd fully green. The reason is the fixtures, not the assertion -- both takeovers replay bytes beginning with f (0x66) and r (0x72), and both are legal CSI final bytes, so FeedFraming(replay) closes the child's pending sequence on its own and midSequenceForTest() reads false with or without the reset. A discriminating fixture is one line: feed "x\x1b[3", then redrawTab([]byte("0000")) -- parameter bytes with no final. Green with the reset, red without it (verified both directions). The rule: an assertion that a state was CLEARED is evidence only when the fixture cannot reach that state by any path other than the clearing. In practice that means a fixture chosen to be inert with respect to the mechanism under test, and the check that makes it stick is the mutation, not the read -- which is the same rule BR-26 states for assert-absent writes, now shown to govern assert-absent STATE as well.
- **BR-41** [Important] `envelope-claim-unenforced` runZellij hands the subprocess the pane's raw-mode stdin, while code and atlas both claim it gives neither descriptor
  This is the 2nd finding in family envelope-claim-unenforced, so state the rule rather than patching the line. run.go:1426 sets cmd.Stdin = os.Stdin -- the pane's own descriptor, put in raw mode by host.MakeRaw() at run.go:242 -- while run.go:1394 asserts "NEITHER method gives a subprocess the pane's descriptors, and that is the point" and atlas/architecture.md:514 repeats it. The enforcement test is named TestNeitherZellijMethodHandsTheSubprocessThePanesDescriptors and captures two of the three, so the claim's third case is unasserted by construction. Whether any zellij action verb reads stdin is unmeasured; a verb that reads one byte eats an operator keystroke silently, which is exactly the class the milestone exists to close. The rule: an envelope claim about a RESOURCE enumerates every handle of that resource and the test asserts each one -- "descriptors" means the three the process was started with, not the two the finding that prompted the work happened to name. The enumerable set here is the three fields of exec.Cmd set in runZellij; either close stdin or narrow the claim in all three artifacts (plan finding 8, run.go:1394, atlas:514) in the same commit.
- **BR-42** [Minor] `plan-table-drift` The Integration-points table still marks three M3/M4 rows new/modified after BR-11 flipped only the Pure-entities table
  This is the 5th finding in family plan-table-drift, so do not patch the three rows -- state the rule. plan:279-281 declare "strip repaint trigger" new, "degraded rename-pane" modified and "right pane chrome" modified; none exists at HEAD, all three are M3/M4. BR-11 raised exactly this for plan:114-117 and those rows now read "planned -- M3", which is the instance fixed and the class left standing in the table directly below it. The guard cannot catch it: core_concepts_contract_test.go filters on conceptPackage = cmd/internal/couchtty/, so every #199 row declaring termcmd, hostty, rowtext or main-3.kdl is unchecked, and the status column that "doubles as the build tracker" is only a tracker for one package. The rule: the status column is the build tracker for EVERY row of EVERY table in a Core concepts section, and the check that enforces it is scoped to the plan, not to a package -- which is the same gap BR-33 names for atlas, and the reason both should be answered by one mechanism rather than two hand-maintained lists.

## Round 11 — 2026-09-07T23:47:27-07:00 (claude) — passed

### Disposed

- BR-8 — not-addressed — M4.3 (plan:601-605) verbatim unchanged; no Alt+Shift+d step anywhere, while the issue Done-when still requires two right-pane halves each drawing a strip.
- BR-9 — not-addressed — writer_test.go:116-117 still splits one hand-chosen index of one sequence; the gate now exists, so parameterizing is cheap.
- BR-26 — not-addressed — Re-measured at HEAD in a scratch copy - green under mutation at three sites - m.owed = nil (run.go:827), runZellijCaptured's SanitizeAndFit (run.go:1488), resizeThroughWriter (run.go:264). plan-superseded-facts-test.sh:65 still bans 'route only stdout through', which git log -S finds in no revision of the plan, so that registered pair can never fire.
- BR-27 — not-addressed — Capture half landed; log half still absent. run.go:462, 468, 1178, 1270 discard the error and there is no log sink, so a failing wheel tick or rename is completely silent where pre-M2 it printed. run.go:1178 is on the writer goroutine, so its fix needs an inline writeDiag rather than reportError.
- BR-29 — not-addressed — run_test.go:941 fakeRuntime.reportedUnused still present and called from nowhere; terminalMux.stderr (run.go:643) is set by newTerminalMux and read by nothing - grep for m.stderr returns no hits.
- BR-30 — not-addressed — run.go:168 runDecision still takes stdin and stdout and uses neither; nothing in the handleChord path writes stdout.
- BR-31 — not-addressed — ARCH-CONSTRAINTS is unchanged in this window - the plan diff touches only Revisions. The second parse, the unterminated-OSC stall, and the synchronous zellij exec on the sole writer goroutine in removeTab are all still undeclared.
- BR-33 — not-addressed — atlas/architecture.md:502-505 still says a write issued mid-sequence is deferred into a single coalescing slot with a later paint replacing an earlier one - false for diagnostics, which queue in owedDiag and survive a takeover. atlas:537 still lists pair term's strip as a current rowtext consumer when it is M3. tests/plan-superseded-facts-test.sh still covers only the plan file and one probe.
- BR-34 — not-addressed — run.go:861 still appends to owedDiag with no cap and ARCH-CONSTRAINTS declares no budget; diagnosticWidth = 200 still claims to be far short of wrapping while m.cols is on the receiver.
- BR-36 — not-addressed — M3.7(b) landed (plan:595) but still specifies flooding with yes, which emits no ESC, so MidSequence is false for the whole run and the defer path still cannot be entered. No observable is named and no deferral is counted. M4.3 unchanged.
- BR-38 — not-addressed — couchtty/reserve.go:143-152 still ends in doc paragraphs for sanitize and truncate, which now live in rowtext. This window also left run.go:1454-1466, where two consts sit between runZellijCaptured's doc prose and runZellijCaptured, so go doc attaches that prose to diagnosticWidth.
- BR-39 — addressed — Instance fixed in round 4 and class instrument added here - verified by mutation, an ungated m.stdout.Write in reportError reddens TestEveryConsoleWriteIsGatedOrExplicitlyExempt. Two residuals raised as a new finding - the predicate's blind spots - and the takeover exemption's stated reason still omits the mechanism that actually earns it (a leading ESC cancels a pending CSI), which round 9 asked to be written rather than re-derived.
- BR-40 — not-addressed — Re-measured in a scratch copy of HEAD - replacing m.hostScan = ptychild.Screen{} (run.go:826) with a no-op leaves ./cmd/internal/termcmd fully green. The one-line discriminating fixture round 10 supplied (feed "x\x1b[3", then redrawTab([]byte("0000"))) was not added; both takeover fixtures still replay bytes beginning with legal CSI final bytes.
- BR-41 — addressed — Verified by revert - restoring cmd.Stdin = os.Stdin reddens TestTheZellijSubprocessGetsNoStdinEither. atlas:517-520 now says none of the pane's descriptors, including stdin. The stdin arm is still asserted by a source substring rather than behaviorally, which is folded into the new envelope-claim-unenforced finding.
- BR-42 — not-addressed — plan:277-281 unchanged - strip repaint trigger still new, degraded rename-pane and right pane chrome still modified, none existing at HEAD. The same table marks couchtty.RenderStatusRow unchanged after this window changed its body to call rowtext.

### Raised

- **BR-43** [Important] `envelope-claim-unenforced` The door-enumeration test enforces a substring, not the claim - an ungated Fprintf, a leaked exemption marker, and every file but run.go all pass
  This is the 3rd finding in family envelope-claim-unenforced, so the deliverable is the rule, not the three sites. writer_test.go:563 requires a door line to contain ".Write" or "WriteString", so fmt.Fprintf(m.stdout, ...) is invisible - measured, adding one to inheritSize leaves the entire termcmd suite green, and Fprintf is the spelling the pre-M2 code used for exactly these diagnostics. writer_test.go:539-552 stops the exemption walk only on a non-empty non-comment line, so a blank line does not stop it - measured, an unrelated "gate-exempt:" comment two lines above an added ungated write exempts it, which is what writer_test.go:544's own comment claims to prevent. And writer_test.go:525 reads run.go alone, while the plan's M3 file list creates cmd/internal/termcmd/strip.go in the same package. The rule - an enforcement test for a claim of the form "every X does Y" derives X from the language, not from the spellings the author happened to write, and the instrument is mutation-checked against each spelling and each file before it is trusted. The milestone already made the stronger choice once, in M2.3 step 3 - make the RUNTIME incapable rather than fix call sites - and the same move applies here - put m.stdout behind a paneWriter whose only methods are own / diag / exempt(reason), so a new door is a compile error rather than a grep.
- **BR-44** [Minor] `deferred-work-lacks-own-trigger` flushOwed has one call site, in the child-data branch, so an owed write is stranded for as long as the child is silent
  run.go:813 is the only call to flushOwed, inside handleChunk's default (child data) branch. Nothing else pays the debt - not a bare event, not a takeover, not resize, and there is no timer. So while the child's stream is mid-sequence and the child produces nothing further (a prefix written before the child blocks, or Screen.skipping latched by a sequence over maxPending that is never terminated), an owed paint and every queued diagnostic sit invisible indefinitely. That is the same stale row that nothing repaints that run.go:866-874 gives as the reason for owing rather than dropping. Latent at M2 because nothing paints; at M3 it means a strip that stops updating with no path back. couch shares the shape (console.go:1137), so the answer belongs in the shared design - a bounded wait after which the write goes out, or paying the debt from every event rather than only from child output.

## Round 12 — 2026-09-08T13:28:35-07:00 (claude) — BLOCKED

### Disposed

- BR-8 — not-addressed — M4.3's acceptance still has no Alt+Shift+d step, so :138/:139, :165/:166, :191/:192 remain unverified.
- BR-9 — not-addressed — TestPaintDefersMidSequenceAndIsOwed is unchanged and still splits one hand-chosen sequence at one index.

### Raised

- **BR-45** [Important] `exit-path-drops-cleanup` A tab exiting during a rename repaints nothing, so the strip keeps listing the closed tab
  run.go:1296 -- removeTab's `if !preserveRename { m.applyTakeover(...) }` skips the only
  thing that repaints the strip on that path. Reproduced: zero bytes written after tab 1's
  child EOFs while a rename is open. 2nd in this family: do NOT patch removeTab alone.
  The rule is that every site mutating the strip model owes a repaint, and the site set is
  enumerated by a test -- a table over {newTab, switchRelative, closeActive, removeTab (both
  branches), beginRename, refreshRename, finishRename, inheritSize, rowDirty} that drives
  each and asserts a repaint carrying the post-mutation model. It fails on exactly one row today.
- **BR-46** [Important] `superseded-write-not-dropped` flushOwed writes the coalesced older row after the fresh inline repaint already landed
  run.go:866-872 -- the row-dirty debt repaints inline with the current model, then flushOwed
  writes m.owed, which is older. Reproduced: the recorder's last write is "one [two]" after
  "one [built]". This contradicts writeOwn's own stated invariant that the freshest paint is
  the only one worth landing. Reachable via removeTab's preserveRename branch, which mutates
  the model on the writer goroutine without posting a paint or clearing m.owed. Fix: clear
  m.owed when paintStripInline writes directly, or flushOwed before the inline repaint.
- **BR-47** [Important] `acceptance-command-does-not-hold` M3.7 is ticked but (b) -- the escape-carrying flood that reaches the defer-and-owe path -- was never run
  4th in this family, so do NOT just run the step. Every load generator in the record emits no
  escapes: the smoke pass predates the current gate, the Ctrl-C probe used `yes aaa`, and
  couchnestedrows floods with `seq 1 400`. So the defer-and-owe path has no manual evidence in
  the milestone that WIDENED the defer condition to include HoldsCursorSave. The rule: a manual
  acceptance step is ticked only against a Log line quoting the command run and what was seen,
  and milestone-close refuses a ticked manual step with no such line. Until then un-tick M3.7.
- **BR-48** [Important] `consumer-set-not-derived` The degraded title asserts one of two derived consumers, and the sibling producer was never swept
  3rd in this family. Two measured instances of one rule. (1) M3.6 required asserting
  ClassifyLiveLayout as well as RoleForPane; only RoleForPane is asserted, and measured,
  Title="terminal 1" with no command now classifies layout3 as layout2 where the packed
  "[terminal 1] work" classified layout3 -- layoutflow_test.go:143 still fixtures the retired
  "[terminal 1]" form. (2) renamePaneTitleLocked (run.go:1533) was not degraded at all: it still
  packs the tab set and drops the load-bearing prefix, measured as
  "[rename: work-bar] other" -> PaneRoleOther. The rule: where a milestone changes a value a
  derived consumer set reads, the test is a table over every producer x every consumer,
  generated from the derivation the plan already records -- not one hand-picked pair.
- **BR-49** [Important] `deferred-work-lacks-own-trigger` The defer condition was widened to one that can persist indefinitely, and M3's assigned flush deadline was not delivered
  2nd in this family. run.go:981 states "M3 owes it a flush deadline"; no deadline exists.
  Meanwhile unsafeToPaint (run.go:939) now also defers on HoldsCursorSave, which unlike
  MidSequence can be held for as long as a child chooses, and owedDiag is an unbounded append
  with no cap. The rule: a write deferred on a condition the deferrer does not control needs a
  trigger the deferrer does control -- a deadline, a bound, or a written argument that unbounded
  deferral is correct for that payload class. State it per class: unbounded is right for the
  paint (stale beats wrong, already tested); it is not right for a diagnostic.
- **BR-50** [Important] `envelope-claim-unenforced` A zellij subprocess forks per rename keystroke, against the plan's declared "not on every render" budget
  4th in this family. refreshRename (run.go:1173), called once per rename event from the stdin
  pump (run.go:390), calls setPaneTitle -> RunZellijAction -> a process fork, on the interaction
  path ARCH-CONSTRAINTS names as the one that matters, and M3 now adds a strip paint alongside
  it. The issue's Problem opens with "a subprocess per title change" as cost #1. The rule: an
  ARCH-CONSTRAINTS budget must be enforced by something a change trips over, not asserted in
  prose -- count fakeRuntime ops across a scripted rename and assert the bound, or rewrite the
  budget to state the real cost. Cheapest true fix: refresh the title on commit only, since the
  strip now renders the field and the pane is focused throughout a rename.
- **BR-51** [Important] `test-can-touch-real-state` probes/cursorsaveslots force-deletes a zellij session it did not create, from make test-smoke
  It diffs `zellij list-sessions` before/after, picks an arbitrary new name by ranging a map,
  and defers `zellij delete-session <name> --force`. Any session appearing in that 8-second
  window -- the operator's own workbench -- is destroyed. It lives in probes/, which test-smoke
  runs wholesale. The fix was found in this same milestone and not applied: couchnestedrows
  (main.go:112) uses --new-session-with-layout with a deterministic couchnestedrows-<pid> name,
  which is exactly the flag the session-diff dance exists to work around.
- **BR-52** [Important] `copy-instead-of-extract` 196 identical lines duplicated between the two probes/ zellij harnesses
  probes/cursorsaveslots/main.go (287 lines) and probes/zellijscrollregion/main.go (281 lines)
  share 196 identical lines: sessionSet, syncBuffer, tailOf, writeLayout, the ZELLIJ* scrub, the
  pty reader goroutine, lastCursorRowBefore, and the session discovery/cleanup. couchnestedrows
  repeats 83, termctrlc 49, termrows 45. Nothing blocks extraction -- a plain probes/zellijprobe/
  package is importable by every probes/* main, unlike BR-7's unexported-in-another-package case.
  The cost is already realised: the finding above is a bug present in the copy and fixed only in
  the copy's successor.
- **BR-53** [Minor] `doc-states-planned-as-current` Three superseded doc comments left standing beside their corrections
  2nd in this family, so state the rule rather than editing three sites: when a comment is
  superseded, DELETE it -- do not prepend the correction, and never leave two doc comments on
  one declaration. Sites: run.go:1508 (paneTitleLocked claims the packed title matched
  shortcut.go's arm NEVER, twenty lines above the paragraph saying it did, and the plan records
  the first as measured-false); run.go:1368-1370 (childSizeLocked has two stacked doc comments,
  the first now the opposite of the behaviour); ptychild/screen.go:141-156 (HoldsCursorSave's
  doc was inserted mid-comment, so godoc attaches TakeRowDirty's whole rationale to it and
  TakeRowDirty has none). A vet-style check for two comment blocks on one declaration catches
  the class.
- **BR-54** [Minor] `plan-table-drift` The Core concepts table still says "planned — M3" and omits five entities M3 introduced
  6th in this family. plan.md:225-228 still marks TabChip/StripModel/RenderedStrip/RenderStrip as
  planned; RenameField, TabSpan, RenameEditor.Field, hostty.ResetSGR/ReserveAndPaint and
  Screen.HoldsCursorSave are absent, as is the stripOwed debt from the integration table. The
  rule, since instance-fixing has not held five times: a hand-maintained table restating the
  tree drifts every milestone. Either derive it -- a test that greps each row's stated path for
  its stated symbol and fails on a status that disagrees with the diff -- or drop the status
  column so the table stops making a claim nothing checks.
- **BR-55** [Minor] `user-facing-change-undocumented` README does not mention that the right pane loses a row to the strip or gains a tab bar
  README.md:15 describes the pane's tabs but not the strip. probes/termsmoke changed its
  assertion from "40 100" to "39 100": the child is now one row shorter, which is observable to
  anything the operator runs in the pane. M4 is the natural place to land both, but it should be
  listed there rather than left to be noticed.

## Round 13 — 2026-09-08T14:02:38-07:00 (claude) — BLOCKED

### Disposed

- BR-8 — not-addressed — M4.3 still has no Alt+Shift+d step; the plan records it as deferred to M4.
- BR-9 — not-addressed — TestPaintDefersMidSequenceAndIsOwed is byte-identical and still splits one sequence at one index.
- BR-45 — addressed — Mutation-verified: removing removeTab's paintStripInline turns the rename-branch table row red.
- BR-46 — addressed — Mutation-verified: removing writeOwn's `m.owed = nil` makes the stale row land after the fresh one.
- BR-47 — addressed — Automated in couchnestedrows with an SGR-per-line flood plus tab switches; could not re-run here (needs zellij + a real pty).
- BR-48 — addressed — Second producer deleted, ClassifyLiveLayout derives from RoleForPane, table mutation-verified; see the new finding for the producer's own restatement of the predicate.
- BR-49 — addressed — Decided per payload; owedDiag capped at maxOwedDiag with a test, unbounded paint deferral argued and pinned.
- BR-50 — addressed — TestARenameCostsExactlyOneZellijSubprocess asserts exactly one recorded op across three keystrokes.
- BR-51 — addressed — zellijprobe.Start names and Close deletes that name; swept the tree, every delete-session under probes/ and cmd/probes/ targets a self-chosen name.
- BR-52 — addressed — Verified by line count -- cursorsaveslots 172, zellijscrollregion 165, shared harness 219.
- BR-53 — not-addressed — paneTitleLocked's godoc still carries the measured-false claim with a correction appended below it, and redrawTab still stacks three doc paragraphs; no class guard added.
- BR-54 — addressed — Table flipped and extended and the planned-Mx rot guarded; residual noted in the new plan-table-drift finding.
- BR-55 — addressed — README now documents the strip and the row the pane gives up.

### Raised

- **BR-56** [Important] `consumer-set-not-derived` paneTitleLocked restates RoleForPane's predicate instead of asking it, so a tab named `terminals` classifies as PaneRoleOther
  This is the 4th finding in family `consumer-set-not-derived`, so state the rule rather than
  patching the prefix. Measured against the real consumer at run.go:1557 --
  tab `terminalwork` -> title `terminalwork` -> PaneRoleOther; same for `terminal-2` and
  `terminals`. The guard is `HasPrefix(name, "terminal")`, an approximation of
  `title == "terminal" || HasPrefix(title, "terminal ")`, so the pane loses the classification
  that routes global shortcuts whenever TerminalCommand is unavailable -- the exact consequence
  BR-48 described, one level upstream of where BR-48 was fixed.
  THE RULE: a producer must never restate the predicate its consumer applies; it ASKS the
  consumer. Export the predicate from workbenchshortcut and have paneTitleLocked add the prefix
  iff the bare name does not already classify -- then the disagreement is unrepresentable.
  And the producer x consumer table needs its PRODUCER axis derived too: three hand-picked names
  cannot see a boundary case, which is why this survived a table written specifically for BR-48.
- **BR-57** [Important] `envelope-claim-unenforced` A background tab's rowDirty batch repaints the strip and re-asserts DECSTBM over the active child's margins
  This is the 5th finding in family `envelope-claim-unenforced`, so state the rule rather than
  moving the one `if`. run.go:850 sets `m.stripOwed = true` outside the `m.isActive(chunk.id)`
  guard, so a tab whose bytes never reached the terminal drives a full re-Reserve plus repaint.
  Measured with the repo's own harness: `ptyChunk{id: 1, data: "\x1b[2J", rowDirty: true}` on the
  BACKGROUND tab wrote `\x1b7\x1b[1;23r\x1b[24;1H\x1b[0m\x1b[2Kone [two]\x1b[0m\x1b8`. That
  violates the plan's declared budget ("a repaint happens on tab change, resize, and
  batch.RowDirty; NOT per output chunk") on the keystroke path, and clobbers the ACTIVE child's
  own scroll region for a screen the terminal never saw -- M2's BR-35 lesson (the gate models the
  terminal, so feed it exactly what the terminal is shown) applied to the repaint trigger rather
  than to the gate.
  THE RULE: every budget bullet in the plan's ARCH-CONSTRAINTS block is an enumeration, and each
  entry owes a test that trips when it is exceeded. Two of the four have one now
  (TestARenameCostsExactlyOneZellijSubprocess, TestOnlyOneGoroutineWritesTheHost); the paint-
  frequency budget is the one with no test, and it is the one the tree violates. Count paints
  across a scripted background-tab flood and assert the bound, rather than asserting the bound in
  prose next to a mechanism that does not honour it.
- **BR-58** [Important] `plan-table-drift` The Core concepts table declares ResetSGR at hostty/reserve.go; it is defined at hostty/control.go:41
  This is the 7th finding in family `plan-table-drift`. Do NOT fix the row -- the rule is what is
  missing. plan.md:239 states `ResetSGR / ReserveAndPaint | cmd/internal/hostty/reserve.go`;
  ReserveAndPaint is there, ResetSGR is at control.go:41. Same table, `right pane chrome` carries
  status `modified` while main-3.kdl still has no terminal-pane borderless=true (M4 unshipped).
  Rated Important, not Critical: the symbol exists and is exported from the declared package, so
  no consumer is misled about behaviour -- what is unchecked is the table's own claim.
  THE RULE, and it is the half of BR-54 that was not delivered: the new guard
  (TestNoPlannedRowSurvivesItsTickedMilestone) only regexes `planned - Mx` after Mx is ticked,
  and couchtty's contract filters #199's rows to `cmd/internal/couchtty/` paths -- so NO test
  reads a single one of M3's twelve new rows. Either grep each row's stated path for its stated
  symbol (which is what BR-54 asked for and what would have caught this in the same commit that
  wrote it), or drop the path and status columns so the table stops making claims nothing checks.
- **BR-59** [Minor] `moved-surface-drops-a-case` When the renamed tab itself exits, the in-progress rename field vanishes from the row
  Measured: with a rename open on the active tab, driving removeTab on that tab's id leaves the
  row as `[one]` -- no field at all -- while the stdin pump goes on routing keystrokes into the
  editor. stripModelLocked (run.go:1428) resolves the missing tab to `Tab: -1` and the renderer
  treats that as "mark nothing". The deleted renamePaneTitleLocked had an explicit
  `if !found { append("[rename: "+field+"]") }` branch for exactly this case; the move from the
  title to the strip did not carry it. The commit is a no-op afterwards, so the cost is brief
  blind typing rather than data loss -- but it is a branch lost in a move that was presented as a
  move.
- **BR-60** [Minor] `dead-test-scaffolding` stripRepaintCase.needsPty is set at one site and read at zero, and two imports are kept alive by blank vars
  This is the 2nd finding in family `dead-test-scaffolding`, so state the rule: a test-harness
  field or declaration that no assertion reads is scaffolding that reads as protection.
  stripmutation_test.go:53 declares `needsPty`, :80 sets it, nothing reads it -- mustNewTab's
  t.Skipf does the work. :325-326 carry `var _ = io.Discard` and `var _ ptychild.Size`, which
  exist only to keep two otherwise-unused imports compiling. Delete the field and the imports;
  a struct field with no reader is the same shape as the "field set at zero call sites" the
  claimed-fix check exists to catch.

## Round 14 — 2026-09-08T14:26:24-07:00 (claude) — BLOCKED

### Disposed

- BR-8 — not-addressed — M4.3 still has no Alt+Shift+d step; the plan's own revision defers it to M4.
- BR-9 — not-addressed — writer_test.go:117 still splits one hand-chosen sequence at one index.
- BR-53 — addressed — Three sites rewritten and TestNoDeclarationCarriesTwoStackedGodocs added; its scope is Important 3.
- BR-56 — addressed — Mutation-verified: restoring HasPrefix(name,"terminal") fails on terminals/terminal-2/terminalwork for both consumers.
- BR-57 — addressed — Instance fixed and mutation-verified; the rule it stated is only half delivered, raised anew below.
- BR-58 — addressed — Row corrected to control.go and the guard checks a DEFINITION, not a mention.
- BR-59 — addressed — Mutation-verified: forcing detached=false reddens TestARenameWhoseTabExitedStaysOnTheRow and a stripRepaintCase.
- BR-60 — addressed — needsPty and both blank vars are gone; no `var _` remains in the package's tests.

### Raised

- **BR-61** [Important] `moved-surface-drops-a-case` The Core-concepts guard hardcodes the ACTIVE plan path, so make test breaks the moment this plan is archived
  plantable_test.go:133 reads workshop/plans/000199-...-plan.md and t.Fatal(err)s on failure.
  sdlc close archives plans to workshop/history/plans/ (39 are already there), so the whole
  repo's suite goes red at M4.5, the next gate. This is the 2nd finding in family
  moved-surface-drops-a-case, so state the rule rather than editing the path: the repo
  ALREADY resolves a plan active-or-archived (couchtty/core_concepts_contract_test.go:272,
  findConceptPlans walks workshop/history when workshop/plans misses), and this guard is a
  deliberate re-implementation that dropped that case. THE RULE - a guard that reads a
  workshop artifact resolves it through one shared active-or-archived resolver; a guard that
  hardcodes a path under workshop/plans is asserting the issue will never close.
- **BR-62** [Important] `envelope-claim-unenforced` The paint-frequency budget and the resize transition still have no test that trips when exceeded
  This is the 6th finding in family envelope-claim-unenforced. BR-57 stated the rule -- every
  budget bullet is an enumeration and each entry owes a test that trips when exceeded -- and
  the round delivered the test for exactly the entry the finding named. Measured against the
  full termcmd suite with a compiler overlay - (a) run.go:863, changing `if chunk.rowDirty` to
  `if true`, i.e. a repaint on EVERY active-tab output chunk, which is the literal violation of
  ARCH-CONSTRAINTS bullet 1 ("a repaint happens on tab change, resize, and batch.RowDirty; NOT
  per output chunk"), leaves the suite green; (b) deleting m.paintStripInline() from
  inheritSize (run.go:1387), the ARCH-ORDER row "host resize -> re-Reserve, re-Paint", also
  green -- after which a resized pane keeps a scroll region computed from the OLD row count.
  For the enumeration's sake, replacing restoreTerminal's ResetRegion write (ARCH-ORDER's
  "Release on every exit path", pre-dating this window) is green too. THE RULE - write the
  enumeration itself as a table in the test file, one row per ARCH-CONSTRAINTS bullet and per
  ARCH-ORDER transition, each owning an assertion that fails when its bound is exceeded. The
  paint bound is a COUNT over a scripted chunk stream, not a boolean "did a paint happen":
  TestOnlyTheActiveTabsOutputDirtiesTheRow is the only test in the package that asks the
  negative question, and it asks it about one chunk.
- **BR-63** [Important] `acceptance-command-does-not-hold` Both new structural guards check less than they claim - one file of five, and no grouped declaration
  This is the 5th finding in family acceptance-command-does-not-hold, so state the rule rather
  than widening two globs. stripmutation_test.go:199 does parser.ParseFile(fset, "run.go", nil, 0)
  while its own doc says it "is what makes a method added later announce itself" -- the package
  has five non-test .go files, and a terminalMux method added in strip.go, rename.go or a new
  file assigns m.tabs/m.active/m.rename invisibly. doccomment_test.go:86 returns no name for a
  GenDecl with more than one Spec, so cmd/internal/hostty/control.go:19-80 -- ONE grouped
  const block, and the file where M3 rewrote ResetSGR's and HomeAndClear's doc comments -- is
  not inspected at all by the guard written for exactly that class; its `checked == 0` floor
  still passes because the file's two funcs are counted. The plan's own 2026-09-07 revision
  already records this shape ("it read run.go only, while M3 adds strip.go to the same
  package") as one of three gaps that made M2's scan insufficient. THE RULE - a structural
  guard enumerates the package's non-test files and descends into grouped declarations, or its
  doc names the narrower scope it actually has, so nobody reads it as covering the class.
- **BR-64** [Minor] `doc-states-planned-as-current` RenameEditor.Field's doc justifies itself by two surfaces; the second was deleted in the same window
  rename.go:64 says "One function, because there are now two surfaces -- the tab strip and the
  degraded pane title -- and a caret drawn two ways is a caret that disagrees with itself".
  renamePaneTitleLocked is gone and Field() has exactly one production caller (run.go:1441).
  3rd in family doc-states-planned-as-current. The rule cannot be mechanised the way the
  stacked-godoc one was -- TestNoDeclarationCarriesTwoStackedGodocs catches a doc RESTARTED,
  not a single opener whose stated fact died -- so record the prevalence instead: when a
  commit deletes a producer, every doc that cited it as a live reason is part of that commit.
- **BR-65** [Minor] `copy-instead-of-extract` ReserveAndPaint copies Paint's body rather than composing it, and Reserve() now has no production caller
  2nd in family copy-instead-of-extract. reserve.go:117 and :147 differ only by the SetRegion
  insertion; TestReserveAndPaintAgreesWithItsParts checks only that the region substring is
  present, so a change to Paint (a hide-cursor, a different erase) silently will not reach
  ReserveAndPaint. THE RULE - extract the shared tail (MoveTo + ResetSGR + ClearLine + text +
  ResetSGR) and let both spell only what differs. Separately, Reservation.Reserve() has zero
  production callers now that couch and termcmd both use ReserveAndPaint.
- **BR-66** [Minor] `plan-table-drift` A new exported symbol in a shared package has no Core-concepts row, and nothing checks that direction
  This is the 8th finding in family plan-table-drift. workbenchshortcut.TitleIdentifiesRightTerminal
  (shortcut.go:198) is new exported surface in a package two other components consume, with no
  row in the plan's table and no mention in atlas/architecture.md's paragraph about the
  predicate. The round-12 fix built the table->code direction
  (TestEveryCoreConceptRowNamesASymbolThatExists); nothing checks code->table. THE RULE - the
  table needs both directions, the machinery exists (couchtty's conceptInventory), and it is
  scoped to one package; widening it is pair#188. Until then, say so in the plan rather than
  leaving the table's completeness as an unstated claim.

## Round 15 — 2026-09-08T14:46:06-07:00 (claude) — passed

### Disposed

- BR-8 — not-addressed — M4.3 still has no Alt+Shift+d step; six of nine borderless rungs stay unverified.
- BR-9 — not-addressed — writer_test.go:117 still splits one hand-chosen sequence at one index.
- BR-61 — not-addressed — tests/plan-superseded-facts-test.sh:45 still hardcodes workshop/plans/000199-…-plan.md and is wired into make test; measured 7 failures with the plan archived.
- BR-62 — addressed — All three mutations re-run in a scratch clone and all three now go red.
- BR-63 — addressed — Verified by mutation: a mutator in strip.go and a stacked godoc in control.go's grouped const are both caught.
- BR-64 — addressed — rename.go:64 now states one surface and explains the second was deleted.
- BR-65 — addressed — drawRow extracted; TestReserveAndPaintIsPaintPlusTheRegion asserts equality, not substring presence; Reserve() deleted.
- BR-66 — addressed — Table row and atlas paragraph added; the table now states it is checked table→code only, attributing the other direction to pair#188.

### Raised

- **BR-67** [Important] `doc-states-planned-as-current` The issue's Done-when still carries a bullet its own Revisions marks struck, and omits the one that entry says replaced it
  workshop/issues/000199-…md:170 still lists the scroll-position deliverable that :701
  records as "Struck"; the borderless=true bullet the same entry says was "Added to
  Done when" is absent; and the Spec at :113 still calls borderless a follow-on that
  the same entry moved into scope. This is the artifact sdlc close verifies against.
  Fourth in family, so state the rule rather than editing three bullets: this family's
  class guard already exists as tests/plan-superseded-facts-test.sh and is bounded to
  $PLAN, never reading the issue. Widen it to take (file, token, replacement, max_line)
  over both artifacts. The same enumeration catches the code-side instance in this
  window, run.go:1359 ("inheritSize does not write the pane today, but it becomes a
  writer in M3"), which paintbudget_test.go:71 now asserts the opposite of.
- **BR-68** [Important] `plan-table-drift` The Core-concepts guard reads only pipe rows, so the bullets under the table are unchecked and one names a method this commit deleted
  Plan :266-267 says Reservation "answers ChildRows(), Reserve(), Release(),
  Paint(text)"; Reservation.Reserve() was deleted in fc318eb9, and reserve.go's own
  godoc now says "There is no bare Reserve()". plantable_test.go:175 skips every line
  not starting with "|", so the descriptive bullets immediately beneath the table --
  which name the same symbols at the same declared paths -- are invisible to the guard
  written for exactly that claim. Ninth in family: do not edit the bullet. The guard
  reads the whole "## Core concepts" section's backticked identifiers; goIdentifier
  already filters the non-symbols (escapes, zellij action names, issue refs), so the
  widening is close to free.
- **BR-69** [Minor] `exit-path-drops-cleanup` Every probe registers teardown as a defer and then leaves through os.Exit, so a failed test-smoke leaks the session it created
  probes/cursorsaveslots/main.go:72 defers session.Close() and then os.Exit(2)s at :80,
  :124 and :150; probes/zellijscrollregion/main.go:89 has the same shape at :97/:135/:138;
  cmd/probes/couchnestedrows/main.go defers delete-session (:396), os.RemoveAll(dataDir)
  (:387) and os.Remove(layout) (:392) then calls inconclusive(), which os.Exit(1)s -- and
  the path most likely to fire is "the session never came up". os.Exit skips defers, so a
  live zellij session named <prefix>-<pid> plus a temp data dir survive each failure, from
  make test-smoke. BR-51 made "a probe can only destroy a session it made" structural;
  nothing made "a probe always destroys the session it made" structural. THE RULE - a
  probe's main is func main() { os.Exit(run()) } and every diagnostic path returns a code,
  so cleanup is a defer in run. Mechanisable with the go/ast machinery this package
  already uses: fail when a function containing a defer also calls os.Exit. Measured
  prevalence: 3 probes, 11 such exit sites.

## Round 16 — 2026-09-08T15:49:37-07:00 (claude) — BLOCKED

### Disposed

- BR-8 — not-addressed — M4.3 (plan:663) still has no Alt+Shift+d step; the Done-when "Splits still work" bullet closes unverified.
- BR-9 — not-addressed — writer_test.go:127 still splits one hand-chosen sequence at one index; only ptychild's scanner has byte-at-a-time equivalence.
- BR-13 — not-addressed — termcmd now goes through NewReservation (run.go:1412,1422) but couchtty/console.go:922 still builds the struct literally.
- BR-14 — not-addressed — atlas/couch.md:228 still presents the reserved row as couch-owned; no mention of hostty.Reservation anywhere in the file.
- BR-15 — not-addressed — hostty/reserve.go:138-154 still states no caller obligation to sanitize or clamp the text it writes verbatim.
- BR-17 — addressed — The stated grep now returns only correction passages; the tokens are registered in plan-superseded-facts-test.sh and it fires (mutation-checked).
- BR-20 — addressed — probes/zellijscrollregion/main.go:134 uses zellijprobe.TailOf and the value is now a checked precondition of the verdict.
- BR-21 — addressed — atlas/index.md:17-46 states both probe homes with the two reasons; the per-probe targets exist in Makefile.local:96-108.
- BR-22 — addressed — Every probe is func main(){os.Exit(run())}; the go/ast guard goes red when a defer+os.Exit function is injected (verified).
- BR-23 — addressed — termcmd is a real second consumer since M3 (run.go:1412), so the atlas and package doc are now true in the present tense.
- BR-24 — not-addressed — plan:495 still fails in zsh ("no matches found: --include=*.go"); same unquoted form at plan:95 and in run.go's paneTitleLocked doc.
- BR-26 — addressed — Positive controls precede the assert-absents (writer_test.go:227, :388) and each was mutation-checked.
- BR-27 — addressed — runZellijCaptured folds captured stderr into the error; diagnostics have their own capped queue, separate from the paint slot.
- BR-29 — addressed — reportedUnused and the dead field are gone.
- BR-30 — not-addressed — run.go:168 still takes stdin and stdout; neither is referenced in the body.
- BR-31 — not-addressed — ARCH-CONSTRAINTS (plan:357-377) still declares no cost for the per-chunk FeedFraming, nor the unterminated-sequence stall.
- BR-33 — not-addressed — atlas/architecture.md:517-519 still says every console-originated write defers into one coalescing slot; diagnostics use a capped queue that survives a takeover. The stderr gap at :519 is filled; the guard was never extended to atlas.
- BR-34 — addressed — maxOwedDiag=64 with oldest-dropped (run.go:956-968); the envelope-declaration half rides with BR-31.
- BR-36 — addressed — M3.7(b) is automated in couchnestedrows with an escape-emitting flood; M4.3 is recorded at the operator's own granularity with what it does not itemise flagged in the Log.
- BR-38 — not-addressed — couchtty/reserve.go:143-151 still carries the sanitize and truncate doc paragraphs for functions that now live in rowtext.
- BR-40 — addressed — TestTheTakeoverResetIsLoadBearing uses a pure-digit replay that cannot terminate the stale CSI.
- BR-42 — addressed — The status column is now guarded for every row of this plan; the guard is currently RED on the M4 row - see the new Critical.
- BR-43 — addressed — The substring scan is gone; paneWriter has no Write method and the mux holds no other io.Writer to the pane.
- BR-44 — addressed — Decided per payload and documented at flushOwed (run.go:1000-1024): unbounded for the paint by choice, capped for diagnostics.
- BR-61 — addressed — Verified by archiving the plan in a scratch copy - both Go guards and the shell script resolve it from workshop/history.
- BR-67 — addressed — Mutation-verified: reintroducing the struck Done-when bullet fails plan-superseded-facts-test.sh.
- BR-68 — addressed — Mutation-verified: naming a deleted symbol in the Core-concepts prose fails TestEveryCoreConceptRowNamesASymbolThatExists.
- BR-69 — addressed — Mutation-verified: a function with a defer that calls os.Exit fails TestNoProbeExitsPastItsOwnCleanup.

### Raised

- **BR-70** [Critical] `plan-table-drift` make test is RED at HEAD - the ticked M4 collides with the plan's `planned — M4` row, and the commit that ticked it ran nothing
  go test ./... at 316bc86c fails TestNoPlannedRowSurvivesItsTickedMilestone
  (plantable_test.go:153): plan:314 still reads `planned — M4` while issue:191
  ticks M4. `git log -S'- [x] M4 —'` shows the tick landed only in 316bc86c,
  whose message records no test run; 5edb47cf's "make test exit 0 (195 ok)" was
  green precisely because the tick was not yet there. This is the 10th finding
  in family plan-table-drift, so do NOT just edit the row. THE RULE - an edit to
  a tracked artifact is an input to a guard, so the commit that ticks a
  milestone runs the suite exactly like a code commit, and the tick plus its
  table-status flip are ONE edit. Everything else failing in that run is the
  documented sandbox pty class; this one is pure file reads.
- **BR-71** [Important] `moved-surface-drops-a-case` The guard that catches the Critical goes silent when sdlc close archives the issue, so closing hides the failure instead of fixing it
  plantable_test.go:40 resolves the PLAN active-or-archived (BR-61's fix) but
  :116 enumerates issues with a bare Glob over workshop/issues/*.md. Measured in
  a scratch copy - move issue and plan to workshop/history and the currently-RED
  TestNoPlannedRowSurvivesItsTickedMilestone reports ok with `planned — M4`
  still in the table. sdlc close performs exactly that move, so the close would
  record green over the drift and archive a plan that contradicts the code. This
  is the 3rd finding in family moved-surface-drops-a-case. THE RULE - one shared
  active-or-archived resolver for EVERY workshop artifact a guard reads, issues
  included; a guard whose coverage can be removed by archiving is asserting the
  issue will never close.
- **BR-72** [Important] `plan-table-drift` M4 falsified four "the terminal pane is framed" statements and swept none, two of them in the files it edited
  This is the 11th finding in family plan-table-drift, so state the rule rather
  than patching four sentences. Measured prevalence, all live at HEAD:
  atlas/architecture.md:401 ("Pane frame asymmetry ... the agent pane and
  layout-3 terminal render frames ... the draft pane opts out") - the section a
  reader lands on for this subject, now the opposite of the paragraph M4 added
  at :556; atlas/architecture.md:381 ("while keeping frames");
  zellij/config.kdl:12-14 ("Split terminal panes stay pinned and keep their
  frames (the frame is the only visible divider between the split halves and
  carries the tab title)") - two lines above the bullet M4 rewrote;
  zellij/layouts/main-3.kdl:22 ("keeping frames (divider, tab title, scroll
  indicator)") - in the file M4 edited nine times. BR-33 asked for the mechanism
  and BR-67 extended tests/plan-superseded-facts-test.sh to the issue plus one
  run.go comment. THE RULE - that script's file list IS the enumeration of
  artifacts asserting this design, and it must name every artifact class: plan,
  issue, atlas, layout/config comments, code comments. A pair registered for one
  file while the same superseded fact lives in four others is coverage that
  cannot fire.
- **BR-73** [Minor] `formatting-drift` Two cosmetic slips in the M2 extraction: redundant parens and an out-of-order manifest entry
  cmd/internal/couchtty/menu_render.go:625 reads
  `rowtext.SanitizeAndFit((line), width)`, and the rowtext import at :5 sits
  inside the stdlib group unlike every other file in the package.
  cmd/internal/artifactpath/manifest.go:631 inserts
  cmd/internal/rowtext/rowtext.go between procutil and ptychild, breaking the
  list's sort order.

## Round 17 — 2026-09-08T16:41:27-07:00 (claude) — BLOCKED

### Disposed

- BR-8 — not-addressed — M4.3 still has no Alt+Shift+d step and the operator record does not mention the split; now load-bearing because finding (a) shows the guard is blind on those same rungs and 78f8c046 restored frames there for a divider.
- BR-9 — not-addressed — writer_test.go:117 still splits one hand-chosen sequence at one index.
- BR-13 — not-addressed — Premise partly overtaken - run.go:1412/1422 are production callers now - but couchtty/console.go:922 still bypasses the door via bottomReservation.
- BR-14 — not-addressed — atlas/couch.md:228 unchanged; still no pointer to hostty.Reservation from the section a reader lands on.
- BR-15 — not-addressed — No caller-obligation line anywhere in cmd/internal/hostty; grep for sanitiz/rowtext in that package returns nothing.
- BR-24 — not-addressed — plan:528 unchanged. Measured this round in the repo's zsh - "(eval):1: no matches found: --include=*.go", exit 1, pipeline never runs.
- BR-30 — not-addressed — run.go:168 still takes stdin and stdout; neither is referenced in the body.
- BR-31 — not-addressed — The envelope at plan:390-410 still budgets paints only; no entry for the second full parse per chunk or the unterminated-OSC stall.
- BR-33 — not-addressed — The mechanism half landed and fires, but BR-33's own paragraph is unchanged - atlas/architecture.md:517-519 still says a mid-sequence write goes into a single coalescing slot, false for diagnostics since 34918a3c gave them owedDiag.
- BR-38 — not-addressed — couchtty/reserve.go:143-151 still ends in two paragraphs documenting sanitize/truncate; doccomment_test.go's roots exclude couchtty and it checks stacked godocs, not orphaned ones.
- BR-70 — addressed — Verified at HEAD - the row reads "modified - M4", the guard passes, and go test ./... is green apart from the documented sandbox pty class.
- BR-71 — addressed — Verified by construction - archived issue and plan to workshop/history in a scratch tree, reintroduced the drift, guard still fired.
- BR-72 — not-addressed — One of four sites fixed. Live at HEAD - atlas/architecture.md:381 "while keeping frames", zellij/config.kdl:11-14 "keep their frames (the frame is the only visible divider between the split halves)", zellij/layouts/main-3.kdl:22 "keeping frames (divider, tab title, scroll indicator)". The registered config/layout token covers a different sentence in the same file.
- BR-73 — not-addressed — All three present - menu_render.go:625 parens, its rowtext import in the stdlib group, manifest.go:631 out of order.

### Raised

- **BR-74** [Important] `uncovered-negative-assertion` The borderless guard's two-line window lets a neighbour satisfy the check, so 3 of its 9 rungs cannot fail
  This is the 4th finding in family `uncovered-negative-assertion`, so state the
  rule rather than patching the window. THE RULE - a per-item guard must bind its
  evidence to the item; matching inside a window that spans the next item lets a
  neighbour satisfy the assertion, and the mutation check must cover every
  position class (sole, first-of-pair, last-of-pair) rather than one
  representative. Measured: terminalborderless_test.go:47-51 builds
  `window := line + "\n" + lines[i+1]`, and the split rungs are adjacent
  `pane name="terminal"` lines; deleting ` borderless=true` from main-3.kdl:138 in
  both source and mirror leaves the test ok. Same shape at :165 and :191, which
  are exactly the rungs BR-8 says were never exercised by hand. All nine sites
  carry the attribute on their own line, so the window is unnecessary - delete it.
  Greppable form of the class: a guard that constructs evidence from
  `lines[i]+lines[i+1]` and then asserts on the concatenation.
- **BR-75** [Important] `external-input-assumed-wellformed` The cursor-save gate tracks DECSC only; the aliasing SCOSC form the repo's own probe names is untracked
  This is the 4th finding in family `external-input-assumed-wellformed`, so the
  rule, not the site. THE RULE - when this code models a terminal control by its
  byte form, it must enumerate EVERY byte form the terminal accepts for that
  control, and the enumeration is greppable because the repo already writes both
  forms down. BR-37 was the same rule for C0 versus C1 in rowtext.Sanitize; this
  is DECSC (ESC 7 / ESC 8) versus SCOSC (CSI s / CSI u) in
  ptychild/screen.go:439-455, where the CSI switch at :503 handles only 'r' and
  'J'. probes/cursorsaveslots/main.go:3 names both forms in its first sentence, so
  the sibling was enumerated before the fix was written and then not covered. A
  child using CSI s holds the same one slot the code's own comment says terminals
  keep, HoldsCursorSave stays false, and the paint lands inside the pair - the
  exact operator symptom M3 exists to remove. Fix at the class: classify every
  save/restore spelling, or extend the probe to answer the child-side question and
  record the negative where the gate is defined.
- **BR-76** [Minor] `plan-table-drift` BR-70's rule is recorded only in a commit message, so nothing will read it next time
  This is the 12th finding in family `plan-table-drift`. The round's own commit
  states the rule well - a tracked-artifact edit is an input to a guard, so the
  commit that ticks a milestone runs the suite, and the tick plus its table-status
  flip are one edit - but states it in `git log` only. workshop/lessons.md took 159
  lines in this window and none of them is that rule, which AGENTS.md section 4
  makes an obligation of running a review. A rule that lives only in a commit
  message is the same defect as a sweep that depends on remembering to sweep.

## Round 18 — 2026-09-08T18:22:51-07:00 (claude) — BLOCKED

### Disposed

- BR-8 — not-addressed — plan:696-702 still has no Alt+Shift+d step; and splitTerminalDown (run.go:537) creates the split pane with no --borderless while tests/term-pane-shortcuts-test.sh:255 forbids one, so "each half draws its own strip" is unverified by hand and unasserted by test.
- BR-9 — not-addressed — writer_test.go:117 still splits one hand-chosen sequence at one index; no parameterisation over indices.
- BR-13 — not-addressed — couchtty/console.go:922 still returns hostty.Reservation{Rows: rows, Edge: hostty.EdgeBottom} directly.
- BR-14 — not-addressed — grep for "Reservation" in atlas/couch.md returns zero hits; the reserved-row section still reads as couch-owned.
- BR-15 — not-addressed — No caller obligation stated anywhere in cmd/internal/hostty; grep for sanitiz/rowtext in that package returns one unrelated line about row clamping.
- BR-24 — not-addressed — plan:528 unchanged; re-measured in the repo's zsh this round -- "(eval):1: no matches found: --include=*.go", exit 1, pipeline never runs. Same shape at run.go:1541.
- BR-30 — not-addressed — run.go:168 still takes stdin and stdout -- and panes -- none of which the body references.
- BR-31 — not-addressed — The envelope still budgets paints and subprocesses only; no entry for the per-chunk FeedFraming second parse or the unterminated-OSC stall.
- BR-33 — addressed — atlas:518-523 now scopes coalescing to a paint and the takeover drop to the paint; the ban token was the literal prior text and is registered at plan-superseded-facts-test.sh:156. The atlas still does not state that diagnostics queue and survive a takeover -- worth one sentence, not a re-raise.
- BR-38 — not-addressed — couchtty/reserve.go:143-152 still ends in the two doc paragraphs for sanitize and truncate, which now live in rowtext.
- BR-72 — not-addressed — Two more live sites, and the class hole is the point: tests/term-pane-shortcuts-test.sh:253-257 is a make-test guard ENFORCING "split panes keep zellij default frames", a sixth artifact class the enumeration never names; atlas/architecture.md:711 still calls the frame the visible divider between split halves. The sweep also left main-3.kdl:22-26 and atlas:381 mid-sentence.
- BR-73 — not-addressed — All three present at HEAD -- menu_render.go:625 redundant parens, its rowtext import inside the stdlib group at :5, manifest.go:631 out of sort order.
- BR-74 — addressed — Mutation-verified independently: deleting borderless=true from each of the nine rungs in source and mirror reds the guard in all nine cases, including sole, first-of-pair, last-of-pair and block-opening positions.
- BR-75 — not-addressed — The named site is fixed and pinned (revert-verified). The RULE is not: measured at HEAD, "\x1b[?1048h" and "\x1b[?1049h" both leave HoldsCursorSave false -- and screen.go:492 actively clears the flag on the alt-screen transition, false for 1049 which is defined as DECSC-then-switch and is how nearly every full-screen child takes the slot. Parameter-free CSI s is also read as SCOSC with ?69/DECLRMM untracked. Write the enumeration as a table with one assertion per spelling, and let probes/cursorsaveslots answer the zellij-side premise.
- BR-76 — not-addressed — lessons.md's two new 2026-09-08 entries are adjacent rules (mutation-check the guard; stating a class is not closing it) but not BR-70's; that rule is still only in git log.

### Raised

- **BR-77** [Important] `wrong-seam-named` The cursor-save precondition landed in termcmd's private predicate, not at the shared Reservation door, so couch paints unguarded
  This is the 3rd finding in family `wrong-seam-named`, so the rule, not the
  site. hostty.Reservation.Paint / ReserveAndPaint (reserve.go:119,154) emit
  \x1b7 ... \x1b8 -- a property of the MECHANISM this issue lifted into the
  shared package. The precondition M3 measured for it ("never write while the
  child holds a save") was implemented as terminalMux.unsafeToPaint
  (run.go:949), private to one consumer. The other consumer of the same
  primitive, couchtty/console.go:1062 via writeOwn (:1005-1013), gates on
  MidSequence() alone; grep -rn HoldsCursorSave cmd/internal/couchtty returns
  nothing, and couch's children are alt-screen agent TUIs, so the identical
  clobber is reachable there with no guard. THE RULE - a precondition that is
  a property of a shared mechanism belongs at that mechanism's seam, stated so
  every consumer must satisfy it, not in whichever consumer happened to
  discover it. The enumeration is one grep
  (`ReserveAndPaint\|\.Paint(` over cmd, minus tests) and yields exactly two
  production call sites, so a guard over that set is cheap. Same door as
  BR-15's undocumented sanitize obligation - both are obligations of
  hostty.Paint recorded only in the callers that learned them (ARCH-DRY).

## Round 19 — 2026-09-08T18:46:39-07:00 (claude) — BLOCKED

### Disposed

- BR-8 — not-addressed — plan:696-702 still has no Alt+Shift+d step; the split rungs remain hand-unverified.
- BR-9 — not-addressed — writer_test.go:117 still splits one hand-chosen sequence at one index.
- BR-13 — not-addressed — console.go:922 still returns hostty.Reservation{Rows: rows, Edge: hostty.EdgeBottom} directly.
- BR-14 — not-addressed — grep Reservation in atlas/couch.md returns nothing; hostty appears only at :213 and :812.
- BR-15 — not-addressed — No sanitize/clamp obligation in cmd/internal/hostty; the same door now also omits the cursor-save precondition.
- BR-24 — not-addressed — plan:528 unchanged; the unquoted --include=*.go still aborts in the repo's zsh.
- BR-30 — not-addressed — run.go:168 still takes panes, stdin and stdout; none is referenced in the body.
- BR-31 — not-addressed — plan:390-410 still budgets paints and subprocesses only; no entry for the per-chunk FeedFraming or the stall.
- BR-38 — not-addressed — couchtty/reserve.go:143-152 still ends in the sanitize and truncate doc paragraphs for functions now in rowtext.
- BR-72 — addressed — Sixth class registered and the sweep is complete; verified the new GUARD pair fires and the script exits 1.
- BR-73 — not-addressed — All three present; plus a third instance from this round's sweep, the line-wrap artifact at atlas:517-521.
- BR-75 — addressed — Enumeration is a table and mutation-verified; its 1049 arm creates the new Critical, raised separately.
- BR-76 — not-addressed — lessons.md untouched this round; BR-70's rule still lives only in git log.
- BR-77 — not-addressed — Reverting all four couch call sites to MidSequence leaves the suite green, the obligation is still unstated at hostty.Paint, and the swap as landed spins.

### Raised

- **BR-78** [Critical] `moved-surface-drops-a-case` The predicate swap reached four of five sites; the drain loop kept the old one and spins forever
  couchtty/console.go:1202 still continues on MidSequence() while :1111 defers on
  !SafeToPaint(), so a save taken during the drain makes the loop re-pop a chunk it
  cannot progress on. Reproduced deterministically (drain never returns in 3s; the same
  test returns instantly at f233b29d^). couch's Run goroutine is its only writer, so this
  is a frozen console at 100% CPU. THE RULE - a defer condition and the loop that drains
  it are ONE pair and must be the same expression; after replacing a gate predicate,
  grep the old name (one production site remains outside ptychild).
- **BR-79** [Critical] `deferred-work-lacks-own-trigger` The new 1049 arm holds the paint gate closed for the child's whole alt-screen lifetime, in both consumers
  screen.go:527-531 makes cursorSaved true from ?1049h until ?1049l, and SafeToPaint
  (:181) is false throughout. Measured - in termcmd the strip never paints again after
  \x1b[?1049h (five paint rounds, nothing reaches the pane; reverting only that arm
  paints), and in couchtty repaint writes nothing. So less/vim/man freeze the strip and
  make Alt+r blind typing, the symptom M3 exists to remove. flushOwed's justification for
  unbounded deferral ("paid by the child's own DECRC") no longer covers this case, and
  the premise that ?1049h takes the SAME slot our paint clobbers is asserted from the
  spec while probes/cursorsaveslots, built for exactly this question, was not extended.
  This is the 3rd finding in family deferred-work-lacks-own-trigger. THE RULE - a defer
  condition is only admissible with a release the deferrer can observe will arrive;
  widening it to a condition the child controls indefinitely requires either a narrower
  condition or a bounded release, and the measurement that settles which.
- **BR-80** [Important] `plan-table-drift` SafeToPaint is the one new exported shared-package symbol with no plan row, no atlas sentence and no test
  This is the 13th finding in family plan-table-drift, so the rule, not the site.
  Measured - the window adds 16 exported declarations under cmd/internal/{hostty,
  ptychild,rowtext,workbenchshortcut}; 15 appear in the plan's Core-concepts material and
  SafeToPaint appears zero times, in the plan, in the atlas, and in any test.
  atlas/architecture.md:604-615 still names HoldsCursorSave as the bit that says so. The
  Core-concepts guard is table-to-code only, so this direction cannot fire. THE RULE -
  the artifact enumeration BR-72 wrote for SUPERSEDED facts needs its mirror for ADDED
  shared surface (table row, atlas sentence, a test naming the symbol), and that
  enumeration is producible from the diff's new exported declarations rather than from
  memory.

## Round 20 — 2026-09-08T19:15:27-07:00 (claude) — BLOCKED

### Disposed

- BR-8 — not-addressed — plan:696-702 still has no Alt+Shift+d step; and the config.kdl comment added this window asserts split halves are borderless, which depends on the exact_panes=4 swap layout applying the attribute to a pane splitTerminalDown creates with no --borderless. Unverified by hand and unasserted by test.
- BR-9 — not-addressed — writer_test.go:117 TestPaintDefersMidSequenceAndIsOwed still splits "before\x1b[3" at one hand-chosen index; no parameterisation.
- BR-13 — not-addressed — couchtty/console.go:921 bottomReservation still returns hostty.Reservation{Rows: rows, Edge: hostty.EdgeBottom} directly.
- BR-14 — not-addressed — grep Reservation in atlas/couch.md returns nothing; hostty appears only at :213 and :812.
- BR-15 — not-addressed — grep -rn "sanitiz|rowtext" cmd/internal/hostty returns one unrelated line about row clamping; Paint/ReserveAndPaint still state no caller obligation.
- BR-24 — not-addressed — plan:528 unchanged; re-measured in the repo's zsh this round, the unquoted --include=*.go aborts with "no matches found" before the pipeline runs.
- BR-30 — not-addressed — run.go:168 runDecision still takes panes, stdin and stdout; none is referenced in the body.
- BR-31 — not-addressed — The envelope still budgets paints and subprocesses only; no entry for the per-chunk FeedFraming second parse or the unterminated-OSC stall.
- BR-38 — not-addressed — couchtty/reserve.go:143-152 still ends in the two doc paragraphs for sanitize and truncate, which live in rowtext.
- BR-73 — not-addressed — All three present at HEAD - menu_render.go:625 redundant parens, its rowtext import inside the stdlib group at :5, manifest.go:631 out of sort order.
- BR-76 — not-addressed — lessons.md untouched since 82976ec6; grep for the tick-runs-the-suite rule returns nothing. BR-70's rule still lives only in git log.
- BR-77 — not-addressed — One of the three sub-claims cleared (the spin). Measured this round - reverting console.go:1011/1111/1141 to MidSequence leaves the couchtty suite GREEN, and cmd/internal/hostty still states no cursor-save precondition at Paint/ReserveAndPaint.
- BR-78 — addressed — console.go:1207 now asks SafeToPaint; I reproduced the spin deterministically and it is gone. The oracle chosen for it is raised separately.
- BR-79 — addressed — screen.go:201 carve-out is pinned - reverting it turns TestTheSaveGateDoesNotFreezeThroughAnAltScreenSession red. The new dependency it introduces is raised separately.
- BR-80 — not-addressed — Tests now name SafeToPaint, so that third is closed. plan:277/:343 and atlas/architecture.md:614 still name HoldsCursorSave as the bit that decides, which is now a rule the code deliberately does not follow.

### Raised

- **BR-81** [Critical] `state-bit-promoted-without-enumeration` altScreen became a safety bit at HEAD; RIS never clears it, so the save gate is off for good after a reset
  SafeToPaint (screen.go:201) now reads !(cursorSaved && !altScreen), promoting altScreen
  from a bit with zero production consumers to the one that decides whether the paint gate
  closes. Two classify arms were not re-enumerated for the new role, both measured.
  (1) The RIS arm (screen.go:475-479) clears cursorSaved and sets rowDirty but not
  altScreen: "\x1b[?1049h" then "\x1bc" then "\x1b7" leaves altScreen=true and
  SafeToPaint()=true while the child holds a save on the NORMAL screen -- M3's exact
  operator-visible symptom, reachable after any reset/tput reset, and persisting until the
  child next enters and leaves an alt screen. (2) The 1047/47 arm keeps cursorSaved set on
  purpose, with a comment saying clearing it "would hand a console permission to paint
  inside a save the child still holds"; measured, "\x1b7" then "\x1b[?1047h" leaves
  HoldsCursorSave()=true and SafeToPaint()=true, handing exactly that permission for a save
  taken on, and restored to, the normal screen. THE RULE - when a field becomes
  safety-load-bearing its full transition set must be re-enumerated, and the enumeration
  already exists here: TestEverySaveSlotSpellingIsAccountedFor (screen_test.go:519) has one
  axis (HoldsCursorSave, from empty/held). Add a "from alt screen" starting state and a
  SafeToPaint column and both instances fail immediately (ARCH-ORDER).
- **BR-82** [Important] `takeover-resets-framing` The takeover mints a fresh Screen, so the alt-screen carve-out's premise is false after every switch
  This is the 2nd finding in family takeover-resets-framing, so the rule, not the site.
  termcmd/run.go:894 and couchtty/console.go:994 both do hostScan = ptychild.Screen{} on a
  screen replacement, and the replay only re-establishes ?1049h while it is still inside the
  bounded ring. Measured - "\x1b[?1049h", then Screen{}, then "\x1b[?1048h" gives
  SafeToPaint()=false: BR-79's indefinite freeze, reached through the takeover rather than
  through the predicate. THE RULE - a takeover resets the scanner to "knows nothing", so
  every bit the gate reads needs a defined meaning under unknown. MidSequence degrades
  safely because false is the conservative answer; a bit whose false value WIDENS the gate
  does not, and must be re-established from the replay or excluded from the gate. The
  enumeration is one grep for `= ptychild.Screen{}` and yields exactly the two sites.
- **BR-83** [Important] `source-text-proxy-oracle` BR-78's Critical is pinned by grepping console.go, justified by a claim that is measurably false
  console_test.go:864 asserts that flushDeferredNotifications' source text does not contain
  "hostScan.MidSequence()", with a comment saying "reproducing the spin requires the exact
  interleaving that made it a Critical rather than a flake". Measured false - a 35-line,
  single-goroutine, deterministic reproduction exists (feed the active actor a batch of
  [{Bytes:"\x1b[31"},{Notification},{Bytes:"\x1b7"},{Notification}], let it defer at the
  mid-sequence, then complete it): it returns instantly at HEAD and hangs the full 3s
  timeout with only line 1207 reverted. The grep also cannot catch the same defect at the
  other defer/drain pair (termcmd's owed paint), goes red on an extract-to-helper refactor
  that changes nothing, and stays green for a differently-spelled wrong condition. THE RULE
  - a source-text guard is admissible when the property IS textual (plan tables, doc
  comments, import order); when the defect is reachable in-process, the pin is a test that
  reaches it, and the fix's own commit message is not where that judgement gets recorded.

## Open findings

- **BR-8** [Minor] `acceptance-misses-changed-sites` M4's manual acceptance never splits the pane, leaving six of nine borderless sites unverified
- **BR-9** [Minor] `single-interleaving-oracle` The mid-sequence gate test picks one hand-chosen split point over an arbitrary byte stream
- **BR-13** [Minor] `validating-door-bypassed` NewReservation has zero production callers; couch constructs the struct directly
- **BR-14** [Minor] `atlas-points-at-old-home` atlas/couch.md's "The reserved row" section still reads as couch-owned mechanism
- **BR-15** [Minor] `caller-obligation-undocumented` Paint's doc does not state the caller's obligation to sanitize and clamp its text
- **BR-24** [Minor] `acceptance-command-does-not-hold` M1.6's acceptance command aborts in the repo's shell before it checks anything
- **BR-30** [Minor] `handler-posts-to-own-queue` runDecision still receives the pane's stdout, unused, on the input goroutine
- **BR-31** [Minor] `hot-path-cost-undeclared` The gate adds a second full parse of every child byte on the output path, undeclared in the envelope
- **BR-38** [Minor] `atlas-points-at-old-home` couchtty/reserve.go keeps the doc comments for the sanitize and truncate it no longer has
- **BR-73** [Minor] `formatting-drift` Two cosmetic slips in the M2 extraction: redundant parens and an out-of-order manifest entry
- **BR-76** [Minor] `plan-table-drift` BR-70's rule is recorded only in a commit message, so nothing will read it next time
- **BR-77** [Important] `wrong-seam-named` The cursor-save precondition landed in termcmd's private predicate, not at the shared Reservation door, so couch paints unguarded
- **BR-80** [Important] `plan-table-drift` SafeToPaint is the one new exported shared-package symbol with no plan row, no atlas sentence and no test
- **BR-81** [Critical] `state-bit-promoted-without-enumeration` altScreen became a safety bit at HEAD; RIS never clears it, so the save gate is off for good after a reset
- **BR-82** [Important] `takeover-resets-framing` The takeover mints a fresh Screen, so the alt-screen carve-out's premise is false after every switch
- **BR-83** [Important] `source-text-proxy-oracle` BR-78's Critical is pinned by grepping console.go, justified by a claim that is measurably false
