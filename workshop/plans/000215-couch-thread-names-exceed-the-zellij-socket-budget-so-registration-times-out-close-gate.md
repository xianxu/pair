---
gate: boundary-review
issue: 215
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-08T21:57:43-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: defaultSessionNameBudget becomes an acceptance test in the fallback path, against its own stated contract
          detail: |-
            createflow.go:964 stores whatever discoverSessionNameBudget returns, including its
            documented fallback guess of 24 (":913-916" says "a MESSAGE default only -- never an
            acceptance test"). On a machine whose socket path is tight enough that zellij refuses a
            13-byte name, discovery falls back to 24, every later candidate <= 24 bytes is accepted
            arithmetically though zellij refuses all of them, and AssignSessionName returns the LONGEST
            rung. createflow.go:379 then fails with "Pick a shorter tag" -- advice that cannot work,
            since the ladder is what shortens. Pre-#215 the probe judged each rung independently and
            could descend to minSessionRepoBytes and succeed. Fix: return (int, bool) from
            discoverSessionNameBudget and keep probing per candidate when the budget is unknown; update
            the :913-916 comment either way. No test covers this branch.
          family: fallback-guess-used-as-oracle
          round: 1
        - id: BR-2
          severity: Important
          title: the O(1) probe claim and the arithmetic-vs-zellij agreement are pinned only against fakeRuntime
          detail: |-
            ARCH-MOCK. The fix rests on "a candidate's length is a local fact", but nothing runs it
            against real zellij. sessionNameRejected (zellijparse.go:48) matches one substring of
            zellij's stderr, and #215 makes that seam far more load-bearing: a missed rejection used to
            mis-accept one candidate, now one wrong budget decides every remaining candidate without
            asking. make test-live covers quiescence, detach and quit -- nothing covers the name budget.
            Add a PAIR_LIVE_COUCH-gated test asserting the boundary against OSRuntime: budget bytes
            accepted, budget+1 rejected.
          family: fake-is-the-only-oracle
          round: 1
        - id: BR-3
          severity: Important
          title: the zellij tracer injects ~50ms per call into the startup it exists to measure
          detail: |-
            probes/zellijcalls/shim.sh:41-46 spawns python3 twice per traced call for a timestamp
            (~25ms each, measured here) plus a bash fork, and drops the exec when armed. SKILL.md tells
            the operator to compare in-zellij time against wall span -- but the python cost lands
            outside each measured duration and inside the span, inflating exactly the denominator the
            conclusion turns on. At the call volume #215 already measured this is seconds injected into
            an 8.85s budget, in the instrument #218 will use to find where those seconds go. Not yet
            run, so cheap to fix: use bash EPOCHREALTIME with a fallback, and state the residual
            per-call overhead in SKILL.md.
          family: probe-overhead-distorts-measurement
          round: 1
        - id: BR-4
          severity: Important
          title: atlas update appears missing for the shell/PATH-shim probe class introduced by probes/zellijcalls
          detail: |-
            atlas/index.md:17-49 states the probe-home rule and closes with "anything else goes under
            cmd/probes/ with its own target in the same commit, since nothing will run it otherwise".
            probes/zellijcalls/ fits neither branch: it takes arm|report|disarm, needs an interactive
            shell, has no Go code and no target. test-smoke does not break on it (confirmed under
            /bin/sh, exit 0) but prints "shared harness, not a probe" -- a category it does not belong
            to. That atlas passage records itself as written because #199's first probe landed in the
            wrong home; this is the next instance.
          family: new-surface-without-atlas
          round: 1
        - id: BR-5
          severity: Minor
          title: promptForTag hand-rolls the probe-then-discover pair that sessionNameAcceptor now encapsulates
          detail: |-
            createflow.go:713-716. ARCH-DRY. Cost is fine (refusal path only), but there are now two
            encodings of one idea; an acceptor that can also surrender its measured budget collapses
            them, and would also carry the fallback-vs-measured distinction I1 needs.
          family: duplicated-probe-then-discover
          round: 1
        - id: BR-6
          severity: Minor
          title: nothing asserts diagnoseRegistrationFailure's output actually reaches the operator
          detail: |-
            The function is unit-tested in isolation; launch_existing.go:123-124, the %s that wires it
            into the user-visible error, has no test. Reachable, not dead -- but this issue's own Log
            records a path regressing precisely because nothing asserted it. The PairSessionIO
            assertion's false branch is also uncovered.
          family: fix-not-pinned-at-its-call-site
          round: 1
        - id: BR-7
          severity: Minor
          title: an unreadable session index is reported as "NO Pair session -- Pair never started"
          detail: |-
            launch_existing.go:226-229. When observeErr is a read or corruption failure rather than an
            absent binding, the message still asserts the conclusion before quoting the error.
            ARCH-SECURE: degrade visibly rather than substitute a fabricated answer downstream reads as
            evidence.
          family: inconclusive-observation-stated-as-conclusion
          round: 1
        - id: BR-8
          severity: Minor
          title: resumeRegistrationTimeout now equals the cold constant, so the resume/cold branch is a test seam only
          detail: |-
            couch.go:114 initialises it to pairRegistrationTimeout, so launch_existing.go:111-114 gives
            both paths 15s in production while reading as if they differ. Worth a comment. The cold path
            has no equivalent seam, so no test can exercise a cold registration timeout without waiting
            15s.
          family: vestigial-branch-reads-as-policy
          round: 1
        - id: BR-9
          severity: Minor
          title: trace.sh writes a fixed ${TMPDIR:-/tmp}/pair-zellij-trace.tsv and arms PATH with manual disarm
          detail: |-
            trace.sh:15 truncates a predictable name with ": >"; on a Linux box with no TMPDIR that is a
            shared directory. arm prepends a repo directory to the interactive shell's PATH with only a
            manual disarm. Both opt-in developer surface, neither stated. Also trace.sh:32 interpolates
            $DIR into a grep pattern, and shim.sh:45 logs "$*", losing argv quoting that kind() then
            re-splits on whitespace.
          family: predictable-path-in-shared-directory
          round: 1
        - id: BR-10
          severity: Minor
          title: the issue records three mid-stream scope changes in Log subsections instead of a Revisions section, and Done-when now overstates
          detail: |-
            AGENTS.md asks for an appended "## Revisions" section (timestamp + reason + delta) rather
            than overwriting; 4663fb70 rewrote the body in place and clobbered the frontmatter. Two
            "Done when" bullets are now inaccurate: the loaded-machine criterion is asserted by test
            rather than a live run, and "over-long candidates are rejected without a subprocess" is not
            literally met -- the first one costs 1 probe plus ~7 for discovery. That trade-off is
            deliberate and well-argued; the criterion should say what shipped.
          family: issue-overwritten-not-revised
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-08T23:45:44-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Mutation-verified: forcing measured=true turns TestAnUnmeasurableBudget red. See I1 for the unswept sibling exit.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: Live test exists, is in make test-couch-zellij-live, and passes here (budget 57, classifier matches zellij 0.44.3).
          round: 2
        - id: BR-3
          disposition: addressed
          note: python3 clock reads are gone and the shim is Go; the residual overhead CLAIM is a separate defect, raised as I3.
          round: 2
        - id: BR-4
          disposition: addressed
          note: atlas/index.md now names the operator-driven instrument class and why it is safe under the smoke loop.
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: promptForTag:713-717 still hand-rolls probe-then-discover; the acceptor still cannot surrender its budget.
          round: 2
        - id: BR-6
          disposition: not-addressed
          note: The no-observer branch is now covered, but launch_existing.go:123-124 (the wiring) still has no test. See I4.
          round: 2
        - id: BR-7
          disposition: addressed
          note: Unreadable and absent are now distinct states with a test asserting the diagnosis stops short of a verdict.
          round: 2
        - id: BR-8
          disposition: not-addressed
          note: couch.go:114 now reads pairRegistrationTimeout but no comment says the resume branch is a test seam only.
          round: 2
        - id: BR-9
          disposition: addressed
          note: Per-invocation $$ path, exact-match disarm instead of a grep pattern, %q per arg, and PATH persistence stated in SKILL.md.
          round: 2
        - id: BR-10
          disposition: not-addressed
          note: No "## Revisions" section; both overstating Done-when bullets still stand as written.
          round: 2
      findings:
        - id: BR-11
          severity: Important
          title: discoverSessionNameBudget reports its own search ceiling of 64 as MEASURED, so a fitting name is refused arithmetically
          detail: |-
            2nd in family. BR-1 added the measured bool for the LOW exit only; the HIGH
            exit still returns (64, true) when the search saturates. Verified:
            discoverSessionNameBudget(len<=100) yields budget=64 measured=true, and the
            acceptor then refuses a 70-byte name zellij would take -- a regression from
            the pre-#215 per-candidate probe, and the silent shortening the ladder's own
            design forbids. Reachable on Linux, where ~/.cache/zellij leaves well over 64
            bytes and which atlas/session-identity.md:184 already names as the differing
            case. Do NOT patch just this exit. The rule: a number that did not come from
            an observation must never be usable as an acceptance oracle, and discovery is
            the only place that knows which it is -- so EVERY way discovery can fail to
            observe reports through the same channel. The enumeration is closed at three
            exits (refused at lo; converged inside; saturated at hi) and
            TestDiscoverSessionNameBudget covers two of them.
          family: fallback-guess-used-as-oracle
          round: 2
        - id: BR-12
          severity: Important
          title: atlas/session-identity.md still asserts the pre-#215 contract -- that the budget is for the message only
          detail: |-
            2nd in family. Lines 185-187 read "the numeric budget is calibrated lazily,
            only after a rejection, purely so the refusal message can quote real
            numbers". After #215 the budget IS the acceptance oracle for every candidate
            past the first rejection, so the passage asserts the negation of what
            shipped, in the file a reader consults for this contract. BR-4 was new
            surface with no entry; this is changed surface whose entry now lies. The
            rule: at a boundary the atlas must DERIVE from the code for surface the diff
            touched, and the sweep is a grep, not a recollection. I ran the enumeration
            -- grep -rn "ProbeSessionName|acceptance oracle|calibrated lazily|24 bytes"
            atlas/ returns exactly this one passage, and atlas/couch.md carries no stale
            deadline number -- so the sweep is one edit, which should also record the
            measured-vs-fallback fork.
          family: new-surface-without-atlas
          round: 2
        - id: BR-13
          severity: Important
          title: the tracer's self-overhead report compares two identical operations, and SKILL.md publishes a number from it
          detail: |-
            2nd in family. reportOverhead (main.go:114-140) execs the real zellij in
            BOTH arms in-process; the only difference is record(), which trace.sh:38
            disables via PAIR_ZELLIJ_TRACE="". The shim's actual cost -- a second Go
            process exec plus realZellij's EvalSymlinks walk of PATH -- is on neither
            side. Measured: trace.sh overhead prints "overhead -518.557us/call" (a
            negative result is the report saying it is noise), while end-to-end through
            a shim named zellij on PATH gives 12.49/12.30ms against 9.26/8.77ms direct,
            about +3.4ms/call, or 8-17% of a 20-45ms zellij call, landing outside each
            recorded duration and inside the wall span -- the same structural defect as
            BR-3, one order of magnitude smaller. SKILL.md states "under 1ms per call,
            below the noise floor" on this basis, in the instrument #218 will use. The
            rule: an instrument's overhead must be measured through the same entry point
            production traffic takes -- spawn the built artifact, never re-run its inner
            steps in the measuring process -- and the injected cost must fall inside the
            measured interval.
          family: probe-overhead-distorts-measurement
          round: 2
        - id: BR-14
          severity: Important
          title: the argv[0] guard that prevents a leaked zellij server per make test-smoke has no test
          detail: |-
            2nd in family. atlas/index.md now claims probes/zellijcalls is safe under the
            wholesale loop "by construction, not by exemption", and the commit calls the
            hazard structurally closed -- but deleting the guard at main.go:44 leaves
            every test in the tree green while make test-smoke starts running a bare
            zellij. Its position is also load-bearing and unstated: --overhead is handled
            at main.go:34, before the guard. Same rule as BR-6, which is still open: a
            guard whose value is that it prevents something needs a test that goes red
            when the guard is removed; reachable is not asserted. The enumeration for
            this issue is two sites -- this one and BR-6's launch_existing.go:123-124 --
            and both are one short test each.
          family: fix-not-pinned-at-its-call-site
          round: 2
        - id: BR-15
          severity: Important
          title: 'no workshop/lessons.md entry from #215, after a review round that produced ten findings'
          detail: |-
            AGENTS.md section 4 asks for lessons.md rules from what a review found. The
            file gained 251 lines in this window, all from #199 (git log -S confirms).
            #215 added none, though its Log already contains at least two in the house
            format: "bounded and cheap are different claims, and a fix for either can
            quietly pay for it out of the other" (the measured 1->7 resume regression),
            and "verify the artifact the code produces, not the hypothesis you authored"
            (the wrong root cause, filed from a hand-composed string when
            AssignSessionName was exported). I1's own cause is a third: marking one
            non-observation as unmeasured does not close the class. The issue Log
            archives to workshop/history, which agents are told not to read.
          family: lesson-recorded-only-in-the-issue
          round: 2
        - id: BR-16
          severity: Minor
          title: trace.sh's header runbook omits the eval on disarm, so following it leaves PATH armed
          detail: |-
            trace.sh:11 says `probes/zellijcalls/trace.sh disarm`, which prints the
            unset/export lines instead of applying them. SKILL.md's block uses eval. Two
            runbooks for one procedure that disagree on the step that undoes a PATH
            mutation.
          family: runbook-disagrees-with-itself
          round: 2
        - id: BR-17
          severity: Minor
          title: the live conformance test re-derives pad() byte-for-byte from the production padding scheme
          detail: |-
            session_name_budget_live_test.go:41-43 duplicates createflow.go:1005-1007. A
            change to the probe marker or padding alphabet takes the conformance test
            with it silently, which is the one test whose job is to disagree with
            production when production is wrong.
          family: duplicated-probe-then-discover
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-09T00:05:53-07:00"
      agent: claude
      dispose:
        - id: BR-5
          disposition: not-addressed
          note: 290c5e8c added a comment explaining why the discard is safe, but the duplication stands; there are now three copies of the probe closure (createflow.go:713, 956, live test:32).
          round: 3
        - id: BR-6
          disposition: not-addressed
          note: Round 2 fixed BR-14, the sibling this finding's own enumeration named, and left this site untouched; launch_existing.go:123-124 still has no test, and the Present==false/err==nil diagnosis branch is uncovered too.
          round: 3
        - id: BR-8
          disposition: not-addressed
          note: couch.go:114 still initialises resumeRegistrationTimeout to pairRegistrationTimeout with no comment; launch_existing.go:111-114 still reads as if the paths differ.
          round: 3
        - id: BR-10
          disposition: not-addressed
          note: The issue file still has no "## Revisions" section, and the two inaccurate Done-when bullets stand; a third is now needed for the conditional O(1) claim.
          round: 3
        - id: BR-11
          disposition: addressed
          note: 'Mutation-verified: restoring `return hi, true` turns the longest_probe_accepted subtest red. Stated as the rule over both bounds, not patched at one end.'
          round: 3
        - id: BR-12
          disposition: addressed
          note: atlas/session-identity.md:182-199 now says the budget IS the acceptance oracle and records the measured-vs-fallback fork.
          round: 3
        - id: BR-13
          disposition: addressed
          note: Real A/B through a binary named zellij; re-ran `trace.sh overhead` here and got +2.753ms/call, matching SKILL.md's 2.9, and it now refuses a non-positive result.
          round: 3
        - id: BR-14
          disposition: addressed
          note: 'Mutation-verified: deleting the argv[0] guard turns TestTheShimOnlyActsWhenInvokedAsZellij red, and the positive control rules out a shim that forwards nothing.'
          round: 3
        - id: BR-15
          disposition: addressed
          note: workshop/lessons.md gained the provenance entry with the four-value table and the bounded-vs-cheap rule.
          round: 3
        - id: BR-16
          disposition: not-addressed
          note: trace.sh:12 still reads `probes/zellijcalls/trace.sh disarm` with no eval, so following the header leaves PATH armed; SKILL.md's block still disagrees.
          round: 3
        - id: BR-17
          disposition: not-addressed
          note: session_name_budget_live_test.go:41-43 still re-derives pad() from createflow.go:1001-1003.
          round: 3
      findings:
        - id: BR-18
          severity: Important
          title: the O(1) probe count holds only where the socket budget is smaller than the longest candidate, and code, atlas and Done-when all state it unconditionally
          detail: |-
            sessionNameAcceptor reaches discoverSessionNameBudget only after a probe REFUSES a
            candidate, and a refusal requires budget < 34 bytes (the longest candidate,
            "pair-couch-<16hex>-NN" with the folder glyph). Where the budget is roomier the
            acceptor never goes arithmetic and the ladder pays one subprocess per suffix again.
            Measured with the production acceptor against fakeRuntime: budget 24 gives 9 probes at
            both 25 and 60 owned suffixes; budget 80 gives 26 and 61. Not hypothetical --
            TestSessionNameBudgetMatchesRealZellijLive measures 57 bytes in this checkout.
            atlas/session-identity.md:185 names Linux as the roomy case, and couch.go:283-288 uses
            "#215 made it O(1)" as the ARCH-CONSTRAINTS justification for a fixed 15s deadline.
            Fix: trigger discovery on the acceptor's SECOND question rather than on a refusal --
            the ledger short-circuit asks exactly once and the ladder asks more than once, so a
            call counter keeps resume at 1 probe and makes the ladder bounded on every machine.
            Both rounds missed it because every probe-count test pins maxSessionNameBytes: 24; the
            fixture needs a budget dimension.
          family: optimisation-gated-on-unstated-environment
          round: 3
        - id: BR-19
          severity: Minor
          title: the acceptor half of TestTheBudgetSearchReportsEitherBoundAsUNMEASURED never enters the branch it claims to cover
          detail: |-
            This is the 3rd finding in family `fix-not-pinned-at-its-call-site` (BR-6, BR-14).
            Do NOT fix this instance alone. The rule: a test that pins a guard must be shown to go
            RED when the guard is removed -- reachable is not asserted. Verified by panic-mutation:
            replacing the `if measured` body in sessionNameAcceptor with panic() leaves
            session_name_scheme_test.go:317-323 green, because the acceptor's first call is a direct
            probe that succeeds at maxSessionNameBytes: 100, so discovery never runs. The
            enumeration for this window is three sites -- this one, BR-14's argv[0] guard (done,
            with a positive control, the model to copy) and BR-6's launch_existing.go:123-124, still
            open. Run the mutation on each. For this site the fixture must take a rejection first,
            e.g. accepts(strings.Repeat("z", 101)), so the arithmetic path is actually entered.
          family: fix-not-pinned-at-its-call-site
          round: 3
        - id: BR-20
          severity: Minor
          title: promptForTag can print a blank refusal when the fallback budget contradicts zellij's own rejection
          detail: |-
            This is the 3rd finding in family `fallback-guess-used-as-oracle` (BR-1, BR-11). Do NOT
            fix this instance alone. The rule covering all three: an unmeasured number may EXPLAIN a
            refusal but must never contradict one -- when the arithmetic disagrees with the
            observation, report the observation. At createflow.go:714-719, if zellij refuses a
            candidate while discoverSessionNameBudget falls back to 24 and the candidate is <= 24
            bytes, sessionNameFits returns ok=true with an empty message and the operator sees
            "pair: " then "pick a shorter name." with no numbers -- on exactly the long-socket-dir
            machine BR-1 exists for. Pre-#215 in origin (#130), but the line was edited in this
            window and the acceptance half of the same rule was already fixed here.
          family: fallback-guess-used-as-oracle
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-09T00:26:35-07:00"
      agent: claude
      dispose:
        - id: BR-5
          disposition: addressed
          note: measureAcceptedLimit now narrows through the caller's own acceptor; one encoding of "does this fit".
          round: 4
        - id: BR-6
          disposition: not-addressed
          note: Untouched by rounds 2 and 3; still no assertion that the diagnosis reaches launch_existing.go:123-124.
          round: 4
        - id: BR-8
          disposition: not-addressed
          note: couch.go:114 still assigns pairRegistrationTimeout with no note that the resume branch is a test seam.
          round: 4
        - id: BR-10
          disposition: not-addressed
          note: Still no "## Revisions" section, and both inaccurate Done-when bullets stand unchanged.
          round: 4
        - id: BR-16
          disposition: not-addressed
          note: trace.sh:12 still prints `trace.sh disarm` without eval; SKILL.md:15 still has it. Two runbooks, still disagreeing.
          round: 4
        - id: BR-17
          disposition: addressed
          note: The live test now carries its own marker literal and drives the production acceptor, so it no longer follows production silently.
          round: 4
        - id: BR-18
          disposition: addressed
          note: 'Mutation-verified: gating the acceptance cache on a prior refusal reds the generous-budget subtest at 26 vs 61 probes. Measured 3-4 probes flat across budgets 20-200.'
          round: 4
        - id: BR-19
          disposition: not-addressed
          note: The named test was removed with its subject, but the three-site enumeration the finding demanded was never run; BR-6 is still open and a fourth site surfaced under mutation.
          round: 4
        - id: BR-20
          disposition: addressed
          note: promptForTag now defaults to the observation and only upgrades to a measured or refused number.
          round: 4
      findings:
        - id: BR-21
          severity: Important
          title: the live conformance test was renamed and Makefile.local's -run regex was not, so test-couch-zellij-live passes without running it
          detail: |-
            This is the 2nd finding in family `fake-is-the-only-oracle`. 869ec844 renamed
            TestSessionNameBudgetMatchesRealZellijLive to TestTheAcceptorAgreesWithRealZellijLive
            (session_name_budget_live_test.go:25) and left Makefile.local:84 selecting the old name.
            Verified: `go test ./cmd/internal/launcher -run '^(TestSessionNameBudgetMatchesRealZellijLive)$'`
            prints "ok ... [no tests to run]" and exits 0. So round 3's "test-couch-zellij-live exits 0"
            claim is vacuous for that test, and sessionNameRejected -- the single-substring seam #215 made
            load-bearing for every candidate rather than one -- has no live check running anywhere.
            ARCH-MOCK. The rule: a suite selected by a name regex silently passes when the name stops
            matching, so the selection needs a floor that fails when it matches nothing. The other five
            names in that target still resolve; I checked.
          family: fake-is-the-only-oracle
          round: 4
        - id: BR-22
          severity: Important
          title: diagnoseRegistrationFailure's "NO Pair session is live / Pair never started" branch has no test; deleting it leaves the package green
          detail: |-
            This is the 4th finding in family `fix-not-pinned-at-its-call-site` (BR-6, BR-14, BR-19). Do
            NOT fix this instance alone. The rule: a branch or guard whose value is what it says must be
            shown to go RED when it is removed -- reachable is not asserted. Verified by mutation:
            replacing launch_existing.go:234 with `return ""` leaves ./cmd/internal/couchcore passing.
            registration_diagnosis_test.go uses one `present` flag to decide whether SetPairSession is
            called at all, so present:false lands on the error path (BR-7's case) and never on
            observeErr == nil with Present false -- which production reaches whenever the name is in the
            index but no live session matches (artifactcollision.go:165-174). That branch is the first
            discriminator the issue's Done-when names. The enumeration for this window is now four sites:
            BR-14's argv[0] guard (done, with a positive control), BR-19's site (removed with its test
            rather than swept), BR-6's launch_existing.go:123-124, and this branch. Run the mutation on
            each. The fixture change here is one line: SetPairSession(address, "\U0001F4C1pair-couch-26", false).
          family: fix-not-pinned-at-its-call-site
          round: 4
        - id: BR-23
          severity: Minor
          title: createflow.go:310 still describes a budget measurement and a per-tag discovery that round 3 deleted
          detail: |-
            This is the 2nd finding in family `vestigial-branch-reads-as-policy`. The rule: when you delete
            a mechanism, grep for the prose that describes it in the same commit -- `git log -S` finds the
            code but not the sentence. The shared thing is now a length bracket, not a discovery, and "N
            tags cost one discovery, not N" describes an implementation that no longer exists.
          family: vestigial-branch-reads-as-policy
          round: 4
        - id: BR-24
          severity: Minor
          title: deleting TestDiscoverSessionNameBudget took with it the only assertion that calibration probes are synthetic pads
          detail: |-
            The deleted test asserted every probe carried sessionNameProbeMarker, because
            `list-clients` SUCCEEDS against a foreign live session and would read as "fits" for the wrong
            reason -- making the measured limit depend on whatever else is running. measureAcceptedLimit
            still depends on that property and nothing in cmd/ now references sessionNameProbeMarker
            outside createflow.go. The rule: when a replacement test supersedes an old one, diff their
            assertions, not just their names.
          family: property-lost-when-its-test-was-replaced
          round: 4
        - id: BR-25
          severity: Minor
          title: runCreate still calls rt.ProbeSessionName raw, a third encoding of "will zellij take this name" outside the acceptor
          detail: |-
            This is the 3rd finding in family `duplicated-probe-then-discover`. Do NOT fix this instance
            alone. The rule, now that the acceptor exists: the acceptor is the ONLY encoding of "does this
            fit", and any site asking that question takes one rather than calling the probe directly.
            createflow.go:379 costs an unconditional subprocess on every create, after assignment has
            already established acceptability. ARCH-DRY. Worth writing the rule into sessionNameAcceptor's
            godoc so the next site inherits it.
          family: duplicated-probe-then-discover
          round: 4
        - id: BR-26
          severity: Minor
          title: ProbeSessionName discards the exec error, so a missing or timed-out zellij reads as ACCEPT -- and the bracket now generalizes that
          detail: |-
            This is the 2nd finding in family `inconclusive-observation-stated-as-conclusion`. The rule is
            the one BR-7 was fixed under: a failed observation is not a positive one; degrade visibly.
            osruntime.go:108 does `out, _ := exec.CommandContext(...).CombinedOutput()`, so an absent
            zellij, a killed process, or a call past zjTimeout (5s, reachable under exactly the load this
            issue is about) yields empty output, sessionNameRejected returns false, and the name is
            accepted. Pre-#215 that mis-accepted one candidate; the bracket now writes it to longestOK and
            answers every shorter name from it. ARCH-SECURE.
          family: inconclusive-observation-stated-as-conclusion
          round: 4
        - id: BR-27
          severity: Minor
          title: round 3's lesson -- every probe-count fixture pinned one budget, so the fake could not express the failing environment -- is not in workshop/lessons.md
          detail: |-
            This is the 2nd finding in family `lesson-recorded-only-in-the-issue`. The rule: a review
            round's transferable lesson belongs in lessons.md, not only in the commit message and the gate
            ledger, because that is the file AGENTS.md asks you to read at session start. The existing
            #215 entry covers rounds 1-2 (provenance) and stops there; BR-18's lesson -- an optimisation
            whose precondition is unstated, hidden because every fixture pinned maxSessionNameBytes: 24 --
            is the strongest of the issue and the one most likely to recur in a different file.
          family: lesson-recorded-only-in-the-issue
          round: 4
      blocked: false
---

# Gate ledger — pair#215 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-08T21:57:43-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `fallback-guess-used-as-oracle` defaultSessionNameBudget becomes an acceptance test in the fallback path, against its own stated contract
  createflow.go:964 stores whatever discoverSessionNameBudget returns, including its
  documented fallback guess of 24 (":913-916" says "a MESSAGE default only -- never an
  acceptance test"). On a machine whose socket path is tight enough that zellij refuses a
  13-byte name, discovery falls back to 24, every later candidate <= 24 bytes is accepted
  arithmetically though zellij refuses all of them, and AssignSessionName returns the LONGEST
  rung. createflow.go:379 then fails with "Pick a shorter tag" -- advice that cannot work,
  since the ladder is what shortens. Pre-#215 the probe judged each rung independently and
  could descend to minSessionRepoBytes and succeed. Fix: return (int, bool) from
  discoverSessionNameBudget and keep probing per candidate when the budget is unknown; update
  the :913-916 comment either way. No test covers this branch.
- **BR-2** [Important] `fake-is-the-only-oracle` the O(1) probe claim and the arithmetic-vs-zellij agreement are pinned only against fakeRuntime
  ARCH-MOCK. The fix rests on "a candidate's length is a local fact", but nothing runs it
  against real zellij. sessionNameRejected (zellijparse.go:48) matches one substring of
  zellij's stderr, and #215 makes that seam far more load-bearing: a missed rejection used to
  mis-accept one candidate, now one wrong budget decides every remaining candidate without
  asking. make test-live covers quiescence, detach and quit -- nothing covers the name budget.
  Add a PAIR_LIVE_COUCH-gated test asserting the boundary against OSRuntime: budget bytes
  accepted, budget+1 rejected.
- **BR-3** [Important] `probe-overhead-distorts-measurement` the zellij tracer injects ~50ms per call into the startup it exists to measure
  probes/zellijcalls/shim.sh:41-46 spawns python3 twice per traced call for a timestamp
  (~25ms each, measured here) plus a bash fork, and drops the exec when armed. SKILL.md tells
  the operator to compare in-zellij time against wall span -- but the python cost lands
  outside each measured duration and inside the span, inflating exactly the denominator the
  conclusion turns on. At the call volume #215 already measured this is seconds injected into
  an 8.85s budget, in the instrument #218 will use to find where those seconds go. Not yet
  run, so cheap to fix: use bash EPOCHREALTIME with a fallback, and state the residual
  per-call overhead in SKILL.md.
- **BR-4** [Important] `new-surface-without-atlas` atlas update appears missing for the shell/PATH-shim probe class introduced by probes/zellijcalls
  atlas/index.md:17-49 states the probe-home rule and closes with "anything else goes under
  cmd/probes/ with its own target in the same commit, since nothing will run it otherwise".
  probes/zellijcalls/ fits neither branch: it takes arm|report|disarm, needs an interactive
  shell, has no Go code and no target. test-smoke does not break on it (confirmed under
  /bin/sh, exit 0) but prints "shared harness, not a probe" -- a category it does not belong
  to. That atlas passage records itself as written because #199's first probe landed in the
  wrong home; this is the next instance.
- **BR-5** [Minor] `duplicated-probe-then-discover` promptForTag hand-rolls the probe-then-discover pair that sessionNameAcceptor now encapsulates
  createflow.go:713-716. ARCH-DRY. Cost is fine (refusal path only), but there are now two
  encodings of one idea; an acceptor that can also surrender its measured budget collapses
  them, and would also carry the fallback-vs-measured distinction I1 needs.
- **BR-6** [Minor] `fix-not-pinned-at-its-call-site` nothing asserts diagnoseRegistrationFailure's output actually reaches the operator
  The function is unit-tested in isolation; launch_existing.go:123-124, the %s that wires it
  into the user-visible error, has no test. Reachable, not dead -- but this issue's own Log
  records a path regressing precisely because nothing asserted it. The PairSessionIO
  assertion's false branch is also uncovered.
- **BR-7** [Minor] `inconclusive-observation-stated-as-conclusion` an unreadable session index is reported as "NO Pair session -- Pair never started"
  launch_existing.go:226-229. When observeErr is a read or corruption failure rather than an
  absent binding, the message still asserts the conclusion before quoting the error.
  ARCH-SECURE: degrade visibly rather than substitute a fabricated answer downstream reads as
  evidence.
- **BR-8** [Minor] `vestigial-branch-reads-as-policy` resumeRegistrationTimeout now equals the cold constant, so the resume/cold branch is a test seam only
  couch.go:114 initialises it to pairRegistrationTimeout, so launch_existing.go:111-114 gives
  both paths 15s in production while reading as if they differ. Worth a comment. The cold path
  has no equivalent seam, so no test can exercise a cold registration timeout without waiting
  15s.
- **BR-9** [Minor] `predictable-path-in-shared-directory` trace.sh writes a fixed ${TMPDIR:-/tmp}/pair-zellij-trace.tsv and arms PATH with manual disarm
  trace.sh:15 truncates a predictable name with ": >"; on a Linux box with no TMPDIR that is a
  shared directory. arm prepends a repo directory to the interactive shell's PATH with only a
  manual disarm. Both opt-in developer surface, neither stated. Also trace.sh:32 interpolates
  $DIR into a grep pattern, and shim.sh:45 logs "$*", losing argv quoting that kind() then
  re-splits on whitespace.
- **BR-10** [Minor] `issue-overwritten-not-revised` the issue records three mid-stream scope changes in Log subsections instead of a Revisions section, and Done-when now overstates
  AGENTS.md asks for an appended "## Revisions" section (timestamp + reason + delta) rather
  than overwriting; 4663fb70 rewrote the body in place and clobbered the frontmatter. Two
  "Done when" bullets are now inaccurate: the loaded-machine criterion is asserted by test
  rather than a live run, and "over-long candidates are rejected without a subprocess" is not
  literally met -- the first one costs 1 probe plus ~7 for discovery. That trade-off is
  deliberate and well-argued; the criterion should say what shipped.

## Round 2 — 2026-09-08T23:45:44-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Mutation-verified: forcing measured=true turns TestAnUnmeasurableBudget red. See I1 for the unswept sibling exit.
- BR-2 — addressed — Live test exists, is in make test-couch-zellij-live, and passes here (budget 57, classifier matches zellij 0.44.3).
- BR-3 — addressed — python3 clock reads are gone and the shim is Go; the residual overhead CLAIM is a separate defect, raised as I3.
- BR-4 — addressed — atlas/index.md now names the operator-driven instrument class and why it is safe under the smoke loop.
- BR-5 — not-addressed — promptForTag:713-717 still hand-rolls probe-then-discover; the acceptor still cannot surrender its budget.
- BR-6 — not-addressed — The no-observer branch is now covered, but launch_existing.go:123-124 (the wiring) still has no test. See I4.
- BR-7 — addressed — Unreadable and absent are now distinct states with a test asserting the diagnosis stops short of a verdict.
- BR-8 — not-addressed — couch.go:114 now reads pairRegistrationTimeout but no comment says the resume branch is a test seam only.
- BR-9 — addressed — Per-invocation $$ path, exact-match disarm instead of a grep pattern, %q per arg, and PATH persistence stated in SKILL.md.
- BR-10 — not-addressed — No "## Revisions" section; both overstating Done-when bullets still stand as written.

### Raised

- **BR-11** [Important] `fallback-guess-used-as-oracle` discoverSessionNameBudget reports its own search ceiling of 64 as MEASURED, so a fitting name is refused arithmetically
  2nd in family. BR-1 added the measured bool for the LOW exit only; the HIGH
  exit still returns (64, true) when the search saturates. Verified:
  discoverSessionNameBudget(len<=100) yields budget=64 measured=true, and the
  acceptor then refuses a 70-byte name zellij would take -- a regression from
  the pre-#215 per-candidate probe, and the silent shortening the ladder's own
  design forbids. Reachable on Linux, where ~/.cache/zellij leaves well over 64
  bytes and which atlas/session-identity.md:184 already names as the differing
  case. Do NOT patch just this exit. The rule: a number that did not come from
  an observation must never be usable as an acceptance oracle, and discovery is
  the only place that knows which it is -- so EVERY way discovery can fail to
  observe reports through the same channel. The enumeration is closed at three
  exits (refused at lo; converged inside; saturated at hi) and
  TestDiscoverSessionNameBudget covers two of them.
- **BR-12** [Important] `new-surface-without-atlas` atlas/session-identity.md still asserts the pre-#215 contract -- that the budget is for the message only
  2nd in family. Lines 185-187 read "the numeric budget is calibrated lazily,
  only after a rejection, purely so the refusal message can quote real
  numbers". After #215 the budget IS the acceptance oracle for every candidate
  past the first rejection, so the passage asserts the negation of what
  shipped, in the file a reader consults for this contract. BR-4 was new
  surface with no entry; this is changed surface whose entry now lies. The
  rule: at a boundary the atlas must DERIVE from the code for surface the diff
  touched, and the sweep is a grep, not a recollection. I ran the enumeration
  -- grep -rn "ProbeSessionName|acceptance oracle|calibrated lazily|24 bytes"
  atlas/ returns exactly this one passage, and atlas/couch.md carries no stale
  deadline number -- so the sweep is one edit, which should also record the
  measured-vs-fallback fork.
- **BR-13** [Important] `probe-overhead-distorts-measurement` the tracer's self-overhead report compares two identical operations, and SKILL.md publishes a number from it
  2nd in family. reportOverhead (main.go:114-140) execs the real zellij in
  BOTH arms in-process; the only difference is record(), which trace.sh:38
  disables via PAIR_ZELLIJ_TRACE="". The shim's actual cost -- a second Go
  process exec plus realZellij's EvalSymlinks walk of PATH -- is on neither
  side. Measured: trace.sh overhead prints "overhead -518.557us/call" (a
  negative result is the report saying it is noise), while end-to-end through
  a shim named zellij on PATH gives 12.49/12.30ms against 9.26/8.77ms direct,
  about +3.4ms/call, or 8-17% of a 20-45ms zellij call, landing outside each
  recorded duration and inside the wall span -- the same structural defect as
  BR-3, one order of magnitude smaller. SKILL.md states "under 1ms per call,
  below the noise floor" on this basis, in the instrument #218 will use. The
  rule: an instrument's overhead must be measured through the same entry point
  production traffic takes -- spawn the built artifact, never re-run its inner
  steps in the measuring process -- and the injected cost must fall inside the
  measured interval.
- **BR-14** [Important] `fix-not-pinned-at-its-call-site` the argv[0] guard that prevents a leaked zellij server per make test-smoke has no test
  2nd in family. atlas/index.md now claims probes/zellijcalls is safe under the
  wholesale loop "by construction, not by exemption", and the commit calls the
  hazard structurally closed -- but deleting the guard at main.go:44 leaves
  every test in the tree green while make test-smoke starts running a bare
  zellij. Its position is also load-bearing and unstated: --overhead is handled
  at main.go:34, before the guard. Same rule as BR-6, which is still open: a
  guard whose value is that it prevents something needs a test that goes red
  when the guard is removed; reachable is not asserted. The enumeration for
  this issue is two sites -- this one and BR-6's launch_existing.go:123-124 --
  and both are one short test each.
- **BR-15** [Important] `lesson-recorded-only-in-the-issue` no workshop/lessons.md entry from #215, after a review round that produced ten findings
  AGENTS.md section 4 asks for lessons.md rules from what a review found. The
  file gained 251 lines in this window, all from #199 (git log -S confirms).
  #215 added none, though its Log already contains at least two in the house
  format: "bounded and cheap are different claims, and a fix for either can
  quietly pay for it out of the other" (the measured 1->7 resume regression),
  and "verify the artifact the code produces, not the hypothesis you authored"
  (the wrong root cause, filed from a hand-composed string when
  AssignSessionName was exported). I1's own cause is a third: marking one
  non-observation as unmeasured does not close the class. The issue Log
  archives to workshop/history, which agents are told not to read.
- **BR-16** [Minor] `runbook-disagrees-with-itself` trace.sh's header runbook omits the eval on disarm, so following it leaves PATH armed
  trace.sh:11 says `probes/zellijcalls/trace.sh disarm`, which prints the
  unset/export lines instead of applying them. SKILL.md's block uses eval. Two
  runbooks for one procedure that disagree on the step that undoes a PATH
  mutation.
- **BR-17** [Minor] `duplicated-probe-then-discover` the live conformance test re-derives pad() byte-for-byte from the production padding scheme
  session_name_budget_live_test.go:41-43 duplicates createflow.go:1005-1007. A
  change to the probe marker or padding alphabet takes the conformance test
  with it silently, which is the one test whose job is to disagree with
  production when production is wrong.

## Round 3 — 2026-09-09T00:05:53-07:00 (claude) — BLOCKED

### Disposed

- BR-5 — not-addressed — 290c5e8c added a comment explaining why the discard is safe, but the duplication stands; there are now three copies of the probe closure (createflow.go:713, 956, live test:32).
- BR-6 — not-addressed — Round 2 fixed BR-14, the sibling this finding's own enumeration named, and left this site untouched; launch_existing.go:123-124 still has no test, and the Present==false/err==nil diagnosis branch is uncovered too.
- BR-8 — not-addressed — couch.go:114 still initialises resumeRegistrationTimeout to pairRegistrationTimeout with no comment; launch_existing.go:111-114 still reads as if the paths differ.
- BR-10 — not-addressed — The issue file still has no "## Revisions" section, and the two inaccurate Done-when bullets stand; a third is now needed for the conditional O(1) claim.
- BR-11 — addressed — Mutation-verified: restoring `return hi, true` turns the longest_probe_accepted subtest red. Stated as the rule over both bounds, not patched at one end.
- BR-12 — addressed — atlas/session-identity.md:182-199 now says the budget IS the acceptance oracle and records the measured-vs-fallback fork.
- BR-13 — addressed — Real A/B through a binary named zellij; re-ran `trace.sh overhead` here and got +2.753ms/call, matching SKILL.md's 2.9, and it now refuses a non-positive result.
- BR-14 — addressed — Mutation-verified: deleting the argv[0] guard turns TestTheShimOnlyActsWhenInvokedAsZellij red, and the positive control rules out a shim that forwards nothing.
- BR-15 — addressed — workshop/lessons.md gained the provenance entry with the four-value table and the bounded-vs-cheap rule.
- BR-16 — not-addressed — trace.sh:12 still reads `probes/zellijcalls/trace.sh disarm` with no eval, so following the header leaves PATH armed; SKILL.md's block still disagrees.
- BR-17 — not-addressed — session_name_budget_live_test.go:41-43 still re-derives pad() from createflow.go:1001-1003.

### Raised

- **BR-18** [Important] `optimisation-gated-on-unstated-environment` the O(1) probe count holds only where the socket budget is smaller than the longest candidate, and code, atlas and Done-when all state it unconditionally
  sessionNameAcceptor reaches discoverSessionNameBudget only after a probe REFUSES a
  candidate, and a refusal requires budget < 34 bytes (the longest candidate,
  "pair-couch-<16hex>-NN" with the folder glyph). Where the budget is roomier the
  acceptor never goes arithmetic and the ladder pays one subprocess per suffix again.
  Measured with the production acceptor against fakeRuntime: budget 24 gives 9 probes at
  both 25 and 60 owned suffixes; budget 80 gives 26 and 61. Not hypothetical --
  TestSessionNameBudgetMatchesRealZellijLive measures 57 bytes in this checkout.
  atlas/session-identity.md:185 names Linux as the roomy case, and couch.go:283-288 uses
  "#215 made it O(1)" as the ARCH-CONSTRAINTS justification for a fixed 15s deadline.
  Fix: trigger discovery on the acceptor's SECOND question rather than on a refusal --
  the ledger short-circuit asks exactly once and the ladder asks more than once, so a
  call counter keeps resume at 1 probe and makes the ladder bounded on every machine.
  Both rounds missed it because every probe-count test pins maxSessionNameBytes: 24; the
  fixture needs a budget dimension.
- **BR-19** [Minor] `fix-not-pinned-at-its-call-site` the acceptor half of TestTheBudgetSearchReportsEitherBoundAsUNMEASURED never enters the branch it claims to cover
  This is the 3rd finding in family `fix-not-pinned-at-its-call-site` (BR-6, BR-14).
  Do NOT fix this instance alone. The rule: a test that pins a guard must be shown to go
  RED when the guard is removed -- reachable is not asserted. Verified by panic-mutation:
  replacing the `if measured` body in sessionNameAcceptor with panic() leaves
  session_name_scheme_test.go:317-323 green, because the acceptor's first call is a direct
  probe that succeeds at maxSessionNameBytes: 100, so discovery never runs. The
  enumeration for this window is three sites -- this one, BR-14's argv[0] guard (done,
  with a positive control, the model to copy) and BR-6's launch_existing.go:123-124, still
  open. Run the mutation on each. For this site the fixture must take a rejection first,
  e.g. accepts(strings.Repeat("z", 101)), so the arithmetic path is actually entered.
- **BR-20** [Minor] `fallback-guess-used-as-oracle` promptForTag can print a blank refusal when the fallback budget contradicts zellij's own rejection
  This is the 3rd finding in family `fallback-guess-used-as-oracle` (BR-1, BR-11). Do NOT
  fix this instance alone. The rule covering all three: an unmeasured number may EXPLAIN a
  refusal but must never contradict one -- when the arithmetic disagrees with the
  observation, report the observation. At createflow.go:714-719, if zellij refuses a
  candidate while discoverSessionNameBudget falls back to 24 and the candidate is <= 24
  bytes, sessionNameFits returns ok=true with an empty message and the operator sees
  "pair: " then "pick a shorter name." with no numbers -- on exactly the long-socket-dir
  machine BR-1 exists for. Pre-#215 in origin (#130), but the line was edited in this
  window and the acceptance half of the same rule was already fixed here.

## Round 4 — 2026-09-09T00:26:35-07:00 (claude) — passed

### Disposed

- BR-5 — addressed — measureAcceptedLimit now narrows through the caller's own acceptor; one encoding of "does this fit".
- BR-6 — not-addressed — Untouched by rounds 2 and 3; still no assertion that the diagnosis reaches launch_existing.go:123-124.
- BR-8 — not-addressed — couch.go:114 still assigns pairRegistrationTimeout with no note that the resume branch is a test seam.
- BR-10 — not-addressed — Still no "## Revisions" section, and both inaccurate Done-when bullets stand unchanged.
- BR-16 — not-addressed — trace.sh:12 still prints `trace.sh disarm` without eval; SKILL.md:15 still has it. Two runbooks, still disagreeing.
- BR-17 — addressed — The live test now carries its own marker literal and drives the production acceptor, so it no longer follows production silently.
- BR-18 — addressed — Mutation-verified: gating the acceptance cache on a prior refusal reds the generous-budget subtest at 26 vs 61 probes. Measured 3-4 probes flat across budgets 20-200.
- BR-19 — not-addressed — The named test was removed with its subject, but the three-site enumeration the finding demanded was never run; BR-6 is still open and a fourth site surfaced under mutation.
- BR-20 — addressed — promptForTag now defaults to the observation and only upgrades to a measured or refused number.

### Raised

- **BR-21** [Important] `fake-is-the-only-oracle` the live conformance test was renamed and Makefile.local's -run regex was not, so test-couch-zellij-live passes without running it
  This is the 2nd finding in family `fake-is-the-only-oracle`. 869ec844 renamed
  TestSessionNameBudgetMatchesRealZellijLive to TestTheAcceptorAgreesWithRealZellijLive
  (session_name_budget_live_test.go:25) and left Makefile.local:84 selecting the old name.
  Verified: `go test ./cmd/internal/launcher -run '^(TestSessionNameBudgetMatchesRealZellijLive)$'`
  prints "ok ... [no tests to run]" and exits 0. So round 3's "test-couch-zellij-live exits 0"
  claim is vacuous for that test, and sessionNameRejected -- the single-substring seam #215 made
  load-bearing for every candidate rather than one -- has no live check running anywhere.
  ARCH-MOCK. The rule: a suite selected by a name regex silently passes when the name stops
  matching, so the selection needs a floor that fails when it matches nothing. The other five
  names in that target still resolve; I checked.
- **BR-22** [Important] `fix-not-pinned-at-its-call-site` diagnoseRegistrationFailure's "NO Pair session is live / Pair never started" branch has no test; deleting it leaves the package green
  This is the 4th finding in family `fix-not-pinned-at-its-call-site` (BR-6, BR-14, BR-19). Do
  NOT fix this instance alone. The rule: a branch or guard whose value is what it says must be
  shown to go RED when it is removed -- reachable is not asserted. Verified by mutation:
  replacing launch_existing.go:234 with `return ""` leaves ./cmd/internal/couchcore passing.
  registration_diagnosis_test.go uses one `present` flag to decide whether SetPairSession is
  called at all, so present:false lands on the error path (BR-7's case) and never on
  observeErr == nil with Present false -- which production reaches whenever the name is in the
  index but no live session matches (artifactcollision.go:165-174). That branch is the first
  discriminator the issue's Done-when names. The enumeration for this window is now four sites:
  BR-14's argv[0] guard (done, with a positive control), BR-19's site (removed with its test
  rather than swept), BR-6's launch_existing.go:123-124, and this branch. Run the mutation on
  each. The fixture change here is one line: SetPairSession(address, "\U0001F4C1pair-couch-26", false).
- **BR-23** [Minor] `vestigial-branch-reads-as-policy` createflow.go:310 still describes a budget measurement and a per-tag discovery that round 3 deleted
  This is the 2nd finding in family `vestigial-branch-reads-as-policy`. The rule: when you delete
  a mechanism, grep for the prose that describes it in the same commit -- `git log -S` finds the
  code but not the sentence. The shared thing is now a length bracket, not a discovery, and "N
  tags cost one discovery, not N" describes an implementation that no longer exists.
- **BR-24** [Minor] `property-lost-when-its-test-was-replaced` deleting TestDiscoverSessionNameBudget took with it the only assertion that calibration probes are synthetic pads
  The deleted test asserted every probe carried sessionNameProbeMarker, because
  `list-clients` SUCCEEDS against a foreign live session and would read as "fits" for the wrong
  reason -- making the measured limit depend on whatever else is running. measureAcceptedLimit
  still depends on that property and nothing in cmd/ now references sessionNameProbeMarker
  outside createflow.go. The rule: when a replacement test supersedes an old one, diff their
  assertions, not just their names.
- **BR-25** [Minor] `duplicated-probe-then-discover` runCreate still calls rt.ProbeSessionName raw, a third encoding of "will zellij take this name" outside the acceptor
  This is the 3rd finding in family `duplicated-probe-then-discover`. Do NOT fix this instance
  alone. The rule, now that the acceptor exists: the acceptor is the ONLY encoding of "does this
  fit", and any site asking that question takes one rather than calling the probe directly.
  createflow.go:379 costs an unconditional subprocess on every create, after assignment has
  already established acceptability. ARCH-DRY. Worth writing the rule into sessionNameAcceptor's
  godoc so the next site inherits it.
- **BR-26** [Minor] `inconclusive-observation-stated-as-conclusion` ProbeSessionName discards the exec error, so a missing or timed-out zellij reads as ACCEPT -- and the bracket now generalizes that
  This is the 2nd finding in family `inconclusive-observation-stated-as-conclusion`. The rule is
  the one BR-7 was fixed under: a failed observation is not a positive one; degrade visibly.
  osruntime.go:108 does `out, _ := exec.CommandContext(...).CombinedOutput()`, so an absent
  zellij, a killed process, or a call past zjTimeout (5s, reachable under exactly the load this
  issue is about) yields empty output, sessionNameRejected returns false, and the name is
  accepted. Pre-#215 that mis-accepted one candidate; the bracket now writes it to longestOK and
  answers every shorter name from it. ARCH-SECURE.
- **BR-27** [Minor] `lesson-recorded-only-in-the-issue` round 3's lesson -- every probe-count fixture pinned one budget, so the fake could not express the failing environment -- is not in workshop/lessons.md
  This is the 2nd finding in family `lesson-recorded-only-in-the-issue`. The rule: a review
  round's transferable lesson belongs in lessons.md, not only in the commit message and the gate
  ledger, because that is the file AGENTS.md asks you to read at session start. The existing
  #215 entry covers rounds 1-2 (provenance) and stops there; BR-18's lesson -- an optimisation
  whose precondition is unstated, hidden because every fixture pinned maxSessionNameBytes: 24 --
  is the strongest of the issue and the one most likely to recur in a different file.

## Open findings

- **BR-6** [Minor] `fix-not-pinned-at-its-call-site` nothing asserts diagnoseRegistrationFailure's output actually reaches the operator
- **BR-8** [Minor] `vestigial-branch-reads-as-policy` resumeRegistrationTimeout now equals the cold constant, so the resume/cold branch is a test seam only
- **BR-10** [Minor] `issue-overwritten-not-revised` the issue records three mid-stream scope changes in Log subsections instead of a Revisions section, and Done-when now overstates
- **BR-16** [Minor] `runbook-disagrees-with-itself` trace.sh's header runbook omits the eval on disarm, so following it leaves PATH armed
- **BR-19** [Minor] `fix-not-pinned-at-its-call-site` the acceptor half of TestTheBudgetSearchReportsEitherBoundAsUNMEASURED never enters the branch it claims to cover
- **BR-21** [Important] `fake-is-the-only-oracle` the live conformance test was renamed and Makefile.local's -run regex was not, so test-couch-zellij-live passes without running it
- **BR-22** [Important] `fix-not-pinned-at-its-call-site` diagnoseRegistrationFailure's "NO Pair session is live / Pair never started" branch has no test; deleting it leaves the package green
- **BR-23** [Minor] `vestigial-branch-reads-as-policy` createflow.go:310 still describes a budget measurement and a per-tag discovery that round 3 deleted
- **BR-24** [Minor] `property-lost-when-its-test-was-replaced` deleting TestDiscoverSessionNameBudget took with it the only assertion that calibration probes are synthetic pads
- **BR-25** [Minor] `duplicated-probe-then-discover` runCreate still calls rt.ProbeSessionName raw, a third encoding of "will zellij take this name" outside the acceptor
- **BR-26** [Minor] `inconclusive-observation-stated-as-conclusion` ProbeSessionName discards the exec error, so a missing or timed-out zellij reads as ACCEPT -- and the bracket now generalizes that
- **BR-27** [Minor] `lesson-recorded-only-in-the-issue` round 3's lesson -- every probe-count fixture pinned one budget, so the fake could not express the failing environment -- is not in workshop/lessons.md
