---
gate: boundary-review
issue: 182
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-04T10:47:03-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Two of the envelope's three durations name budgets that do not exist as constants
          detail: |-
            Family repeat (2nd). Rule, not instance: every duration in the operating
            envelope must cite the constant producing it by file:line, and say so
            explicitly where the budget is derived rather than declared. Here the
            "5s exact-child-death wait (couch.go:119)" is not separate — child death is
            awaited inside the single 15s CompletionTimeout (park.go:549-555) — and the
            "10s blocked-start acknowledgement" is actually resumeRegistrationTimeout,
            5s (couch.go:107, launch_existing.go:110-111). Real worst case ~20s, not
            ~30s; the plan over-budgets, so no downstream decision changes.
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: operating-envelope
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-04T10:47:03-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Important
          title: A resolver IO failure is reported as "binding is not established", and the branch meant to catch it is dead code
          detail: |-
            relaunch.go:96-108 — ResolveEstablished returns a ZERO resolution on a real
            IO error (artifactpath.Resolve / os.UserHomeDir / QuerySessionContext), so
            bindingResumeDiagnostic's default arm yields ResumeBindingUnbound and
            CheckResumePreconditions returns at line 103; `if bindingErr != nil` at 106
            is unreachable. Proved by deleting lines 106-108 in a scratch worktree and
            running -run 'Relaunch|Resume|Park': still green. ResumeContext
            (resume.go:279-283) returns the resolver's error directly, so the two
            callers of the same seam disagree. Resolve the binding only after the
            non-binding preconditions pass, return bindingErr when it is not a
            *ResumeRefusal, and pin it with an erroring-resolver case. ARCH-SECURE.
          family: swallowed-cause-fabricated-diagnostic
          round: 2
        - id: BR-3
          severity: Important
          title: Relaunch on a thread that is not running refuses with `resume-live` and names no next action
          detail: |-
            relaunch.go:81-83 maps every soleParkableIncarnation failure to
            refuseResume(ResumeLive, err.Error()), but park.go:768-783 fails for NO
            live/unknown incarnation as well as for two. On a parked row the operator
            gets "resume-live: park requires exactly one identified live or unknown
            incarnation" — a code contradicting the state and a message about park's
            internals. Reachable: the Spec's panel form relaunches the HIGHLIGHTED row,
            which is often parked. Split the cases; for a thread with no incarnation
            refuse with its own code naming the working gesture ("Enter resumes it").
          family: refusal-names-no-next-action
          round: 2
        - id: BR-4
          severity: Important
          title: Resume's evidence-gathering preamble is re-derived in Relaunch, and the first copy already diverges
          detail: |-
            relaunch.go:88-101 vs resume.go:262-283 both derive `agent` from the saved
            profile, type-assert c.Artifacts to NativeBindingResolver, and compute
            pathExists from c.Path.Physical. They already differ on nil Path: relaunch
            defaults pathExists to true and skips the call, so the precondition PASSES
            and ResumeContext then nil-derefs one step later. The plan extracted the
            RULES for exactly this reason; the evidence needs the same treatment —
            one resumeEvidence(ctx, thread, address) helper with one nil-Path policy.
            ARCH-DRY.
          family: parallel-derivation-drift
          round: 2
        - id: BR-5
          severity: Important
          title: The declared operation reaches no operator surface, and the plan asserts it does
          detail: |-
            menuActionItems (menu.go:1008) returns hardcoded slices and consumes
            Operations() nowhere; ParseCLI is a closed flag set. So the plan's Chunk 2
            premise ("reachable from the switcher's action list the moment it is
            declared") is false, and the Done-when bullet "an actor action relaunch
            appears alongside detach and park ... reachable from the same
            declared-operation surface" has no owning task in M1 or M2. The class is
            six hand-maintained per-operation sites, not the two Task 10 enumerates:
            menu.go:1008, confirmationMenuItems, menu.go:1306, menu.go:1320,
            console.go:1375, console.go:1425. Fix the plan now and sweep the
            enumeration in one M2 round. ARCH-PURPOSE, ARCH-DRY.
          family: declared-source-hand-maintained-consumers
          round: 2
        - id: BR-6
          severity: Minor
          title: No Relaunch-level test for the agent-unsupported and profile-missing refusals
          detail: |-
            The plan's Task 2 table lists four cases; three shipped. The rules are
            covered purely in precondition_test.go and the diagnostic PRECEDENCE is
            correct (profile checks run before the binding check, so an unsupported
            agent still yields ResumeAgentUnsupported even though ResolveEstablished
            was already called with it), but nothing pins that precedence through
            Relaunch.
          family: done-when-untested
          round: 2
        - id: BR-7
          severity: Minor
          title: Task 1 Step 5 is unticked though committed, and the issue's stale-estimate warning contradicts the re-derived Estimate block
          detail: |-
            plan line 159 ("Step 5: Commit") is unchecked although e7c6c6e8 landed it —
            milestone-close's plan-unchecked guard will refuse on it. Separately, the
            issue's ## Revisions still says "⚠ The estimate is now stale" while ##
            Estimate has since been re-derived for the grown scope (3.58 → 6.20).
          family: plan-record-lags-code
          round: 2
        - id: BR-8
          severity: Minor
          title: RelaunchResult does not compose with finishOperation's existing ParkResult/StartResult arms
          detail: |-
            relaunch.go:43 — console.go:1324 and :1328 already know how to land a
            ParkResult address and force-switch onto a StartResult handle.
            RelaunchResult{Outcome, Record, Handle} matches neither, so M2 adds a third
            arm. RelaunchResult{Outcome, Start StartResult} would reuse the resume arm
            unchanged.
          family: result-shape-forces-new-consumer-arm
          round: 2
        - id: BR-9
          severity: Minor
          title: relaunch.go inserted out of alphabetical order in NonArtifactSources
          detail: |-
            artifactpath/manifest.go:524 places it between pathops.go and procops.go.
            Nothing enforces the order; noted so the next insert does not compound it.
          family: manifest-ordering
          round: 2
      boundary: M1
      blocked: true
    - "n": 3
      timestamp: "2026-09-04T11:04:55-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan lines 32-37 unchanged; couch.go:119 is the 15s CompletionTimeout and couch.go:107 sets resumeRegistrationTimeout to 5s, so ~20s worst case, not ~30s.
          round: 3
        - id: BR-2
          disposition: addressed
          note: 'Verified by revert: replacing the bindingErr guard with `_ = bindingErr` turns "the resolver itself fails" red with resume-binding-unbound.'
          round: 3
        - id: BR-3
          disposition: addressed
          note: 'Verified by revert: deleting the hasOccupiedIncarnation gate turns "the thread is not running at all" red with resume-live.'
          round: 3
        - id: BR-4
          disposition: not-addressed
          note: resumeEvidence has ONE caller (relaunch.go:99); ResumeContext (resume.go:298-321) still derives pathExists, the NativeBindingResolver assert and agent itself, with a different nil-Path policy, and the helper's "Shared because" comment is false at HEAD.
          round: 3
        - id: BR-5
          disposition: addressed
          note: Plan now carries the six-site table plus Step 2b, and the issue's M2 bullet names the sweep; I verified all six sites exist. Delivery is M2's.
          round: 3
        - id: BR-6
          disposition: not-addressed
          note: relaunch_test.go still has five cases; no agent-unsupported or profile-missing case reaches Relaunch.
          round: 3
        - id: BR-7
          disposition: not-addressed
          note: Plan:159 still unticked, and now Task 8's steps (472/476) too though 4821dda3 landed them; the stale-estimate warning still contradicts the re-derived block; the M1 table never gained resumeEvidence / hasOccupiedIncarnation / ResumeNotRunning.
          round: 3
        - id: BR-8
          disposition: not-addressed
          note: relaunch.go:42-46 unchanged. finishOperation does carry value alongside err, so the third arm is still what M2 must add.
          round: 3
        - id: BR-9
          disposition: not-addressed
          note: manifest.go:524 still places relaunch.go between pathops.go and procops.go.
          round: 3
      findings:
        - id: BR-10
          severity: Important
          title: 'Alt+n is intercepted and then silently dropped: HitRelaunch has no arm in processInput''s switch'
          detail: |-
            2nd finding in this family, so the deliverable is the RULE, not this site.
            keys.go:94 declares HitRelaunch and keys.go:255 intercepts both chords, but
            console.go:603-610 handles only Switch/Park/Previous/Detach and has no
            default -- so the chord is consumed off the child's input stream and does
            nothing. Before 4821dda3 it reached Pair and reloaded the workbench;
            README.md:141 still documents that, and menu.go:19 menuControls has no
            Alt+n row. Rule: a value the interceptor can emit must reach a handler by
            construction. Three per-hit sites still enumerate by hand (console.go's
            switch, keys.go:234, keys.go:237) after the commit that claimed to close
            the class. Write the enumeration -- a test walking every seqKind through
            hit() plus the legacy branches, asserting each non-HitNone value routes --
            and fold it into Step 2b, which already owns the operation-side half of the
            same rule. ARCH-DRY, ARCH-PURPOSE.
          family: declared-source-hand-maintained-consumers
          round: 3
        - id: BR-11
          severity: Minor
          title: The gate handed this round a base == head window, so every diff recipe returned empty
          detail: |-
            base and head were both 4a7d96e2, so stat, name-status and full diff were
            all empty and a reviewer following the recipes literally would have had
            nothing to inspect. I reviewed 9acfd8e5..4a7d96e2 instead, derived from the
            prior round's `Review-Window: 88fe1de0..9acfd8e5` trailer -- which is what
            brought the unreviewed commit 4821dda3 into scope. Worth a look at how the
            boundary computes BASE_SHA when the previous round's fix commit is HEAD.
          family: review-window-degenerate
          round: 3
      boundary: M1
      blocked: true
    - "n": 4
      timestamp: "2026-09-04T13:55:31-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan lines 32-37 unchanged; the plan file was not touched in this window at all.
          round: 4
        - id: BR-4
          disposition: addressed
          note: resumeEvidence now has two callers and ResumeContext derives none of the three facts itself; the consolidation's own side effect on the warm path is raised separately as a new Critical.
          round: 4
        - id: BR-6
          disposition: not-addressed
          note: relaunch_test.go still has five refusal cases; no agent-unsupported or profile-missing case reaches Relaunch.
          round: 4
        - id: BR-7
          disposition: not-addressed
          note: plan:159 still unticked; Task 8 and parts of Task 10/11 now shipped and unticked too; the issue's stale-estimate warning still contradicts the re-derived Estimate block.
          round: 4
        - id: BR-8
          disposition: addressed
          note: StartedChild generalises the arm instead of adding a third; verified by revert — restoring the concrete StartResult assertion reds TestRelaunchResultIsAdoptedByTheConsole/a_completed_relaunch_is_adopted.
          round: 4
        - id: BR-9
          disposition: not-addressed
          note: manifest.go:524 still places relaunch.go between pathops.go and procops.go; the new inputtrace.go entry was inserted in order, so the convention is being followed for new rows only.
          round: 4
        - id: BR-10
          disposition: not-addressed
          note: The instance is fixed and genuinely pinned (removing the case reds two byte-driven tests), but the requested deliverable was the enumeration and no test walks seqKind/InterceptorHit to a handler; the same class then shipped a new Critical at menu.go:549. README.md:141 and menuControls are also still unchanged.
          round: 4
        - id: BR-11
          disposition: addressed
          note: This round's window is a real range — 4a7d96e2..4ae9d278, 10 commits, 18 files; every diff recipe returned content.
          round: 4
      findings:
        - id: BR-12
          severity: Critical
          title: relaunch is offered in the switcher's action list and Enter on it does nothing
          detail: |-
            3rd finding in this family, so the deliverable is the RULE, not this site.
            menuActionItems (menu.go:1027) now returns "relaunch", but reduceActionKey's
            Enter switch (menu.go:549) routes only park/archive, detach/resume and
            name/describe and has no default -- so Enter on the row produces no frame,
            no effect and no notice. Confirmed with a scratch test through ReduceMenu:
            after Tab then Down, SelectedItem == "relaunch"; after Enter, frame.Action
            == "" and effects == []. That is the issue's FIRST Done-when bullet shipped
            half-wired, and the smoke test missed it because it drove only Alt+n.
            Rule - an item a surface can OFFER must be routed by construction.
            Measured prevalence this round - the plan names six per-operation sites,
            the sweep touched twelve (menu.go 467, 502, 506, 609, 1027,
            confirmationMenuItems, 1258, 1308, 1337, 1351, 1462; console.go 1467,
            1487), and two remain - menu.go:549 and console.go:1445, whose landing arm
            is still a literal == "resume" though RelaunchResult now satisfies
            StartedChild. Extend Step 2b's enumeration from "appears in menuActionItems"
            to "and Enter on it yields a dispatch effect or a frame". ARCH-DRY,
            ARCH-PURPOSE.
          family: declared-source-hand-maintained-consumers
          round: 4
        - id: BR-13
          severity: Critical
          title: The resumeEvidence consolidation makes a warm reattach resolve a binding it never resolved, and fail when that resolver errors
          detail: |-
            resume.go:347-353 - the VerifiedPark == nil branch now calls resumeEvidence,
            which always calls ResolveEstablished and whose binding it then DISCARDS; it
            wanted only pathExists. Any non-ResumeRefusal error now aborts the warm
            reattach. At 4a7d96e2 that branch called the resolver zero times. Proved
            differentially with one test in a scratch worktree - PASS at base, at HEAD
            "warm reattach refused because the binding resolver failed: resolve home
            directory: permission denied". The comment three lines above still states the
            old contract ("Resolve the binding only where it is the authority ... which
            is why relaxing that alone left the operator's detached thread
            unreachable"), so code and contract now disagree, and nothing pins either
            direction - TestDetachedResumeDoesNotRequireAnEstablishedBinding is a pure
            DecideResume test. Reachable causes are ordinary - ListFiles ErrStorageAbsent
            and ReadFile ErrReadLimit once a ledger sidecar passes 8 MB
            (sessioninventory/runtime_os.go:157), besides os.UserHomeDir. Split the
            helper along the axis the callers differ on so the warm branch pays no
            resolver call, keeping ONE nil-Path policy; pin both directions.
            ARCH-CONSTRAINTS (the warm path now pays a ListFiles, an up-to-8MB read,
            proof validation and a possible catalog write, and throws the answer away).
          family: shared-helper-widens-caller-contract
          round: 4
        - id: BR-14
          severity: Important
          title: COUCH_INPUT_TRACE is a new operator surface that records every keystroke and is documented nowhere
          detail: |-
            inputtrace.go:59. A tree-wide grep finds the name only in inputtrace.go, one
            console.go comment and the issue Log - not README.md, not atlas/couch.md, not
            menuControls. Two things need saying together - that it exists and how to
            enable it, and WHAT it captures. pumpStdin traces every chunk before the
            Interceptor splits it, so the file holds everything typed or pasted into the
            hosted agent, prompts and credentials included. 0600 and opt-in are the right
            defaults; the gap is telling the operator at the moment they enable it. Add
            it to atlas/couch.md beside COUCH_STORE_DIR / COUCH_THREAD_*. ARCH-SECURE.
          family: new-surface-undocumented
          round: 4
        - id: BR-15
          severity: Minor
          title: newInputTracer swallows its open error, so a failed probe reads as "the terminal sent nothing"
          detail: |-
            2nd finding in this family, so the deliverable is the RULE, not this site.
            inputtrace.go:58-67 returns nil when os.OpenFile fails, so an unwritable
            COUCH_INPUT_TRACE path yields an empty trace indistinguishable from "no bytes
            arrived" - the exact ambiguity the probe exists to remove, and the same shape
            as BR-2's zero resolution read as "unbound". Rule - an instrument that cannot
            start must say so; the INABILITY to observe must never be presentable as an
            observation. A one-line setNotice on the open failure satisfies it without
            letting a diagnostic take the console down.
          family: swallowed-cause-fabricated-diagnostic
          round: 4
      boundary: M1
      blocked: true
    - "n": 5
      timestamp: "2026-09-04T14:17:13-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan lines 32-37 unchanged; the plan file was not touched in this window at all.
          round: 5
        - id: BR-6
          disposition: not-addressed
          note: relaunch_test.go untouched this window; its five refusal cases still include no agent-unsupported or profile-missing case reaching Relaunch.
          round: 5
        - id: BR-7
          disposition: not-addressed
          note: plan:159 still unticked, Tasks 8/10/11 still unticked though shipped, and the issue's stale-estimate warning still contradicts the re-derived Estimate block.
          round: 5
        - id: BR-9
          disposition: not-addressed
          note: manifest.go:524 still places relaunch.go between pathops.go and procops.go.
          round: 5
        - id: BR-10
          disposition: not-addressed
          note: No test walks seqKind/InterceptorHit to a processInput arm; README.md:141 and menuControls are still unchanged, so the chord ships undocumented and the class deliverable is still the instance.
          round: 5
        - id: BR-12
          disposition: addressed
          note: Verified by revert - restoring the old park/archive+detach/resume switch reds TestEveryOfferedActionIsReachableFromEnter/live/relaunch. console.go:1451's literal is deliberate for M1 and noted as Minor.
          round: 5
        - id: BR-13
          disposition: addressed
          note: Verified by revert - making ResumeContext resolve unconditionally reds TestWarmReattachNeitherConsultsNorIsFailedByTheBindingResolver on both assertions.
          round: 5
        - id: BR-14
          disposition: addressed
          note: atlas/couch.md:764 documents the variable, the 0600 opt-in default, and that the pumpStdin tap captures everything typed or pasted into the hosted agent.
          round: 5
        - id: BR-15
          disposition: addressed
          note: newInputTracer returns an error and New pushes a control-priority notice; TestATraceThatCannotStartSaysSoInsteadOfTracingNothing asserts both halves.
          round: 5
      findings:
        - id: BR-16
          severity: Critical
          title: A relaunch that parks and then fails has its recovery message erased by the next refresh
          detail: |-
            2nd finding in this family, so the deliverable is the RULE, not this site.
            menu.go:1320 excludes relaunch from the park/leave confirmation-close, and
            the comment justifies that by the transient RefusedBeforePark refusal. But
            for ParkIncomplete and ParkedNotResumed the thread is NOT live, InFlight is
            already cleared at menu.go:1306, and the refresh finishOperation always
            requests (console.go:1456 -> menu.go:317 -> reconcileMenuFrames) hits the
            "!operationInFlight && !thread.Live()" arm at menu.go:1271 - discarding the
            confirmation AND overwriting the notice. Reproduced differentially - after
            the result the notice is "relaunch brain: it is parked and the restart did
            not take; Enter resumes it"; after one refresh it is "thread action is no
            longer applicable" and the frames are back at root. That is the issue's
            Done-when bullet ("leaves a resumable parked thread AND SAYS SO") true in
            couchcore and false at the surface, on the destructive path, reading as the
            data loss the Spec calls the whole design question. Rule - a notice carrying
            a just-completed operation's outcome outranks a frame-reconciliation notice,
            and reconciliation must not invalidate a frame on state the operation it
            belongs to produced. The ownership hook exists already
            (Notice.Owner = MenuProgressOwner{OperationAttempt}, menu.go:1324); use it
            rather than adding relaunch to another hand-written list. Measured
            prevalence - park and leave close their frame, archive is exempted by name,
            detach and resume have no confirmation, so relaunch is the only current
            instance and every future confirmed operation inherits it. ARCH-PURPOSE.
          family: refusal-names-no-next-action
          round: 5
        - id: BR-17
          severity: Important
          title: The child a successful relaunch adopts is pre-marked as an expected exit, so its first real death is silent
          detail: |-
            console.go:1427 walks c.panes and marks every pane whose thread == address,
            but the attach dispatched at console.go:1385 has already installed the NEW
            child's pane synchronously (installObservedThreadActor). Verified with a
            dispatcher that installs a pane on attach - expectedExits["new-child"] is
            true when finishOperation returns. onExit (console.go:768) consumes the
            marker and suppresses ExitNotice, so if the freshly relaunched Pair crashes
            or the operator quits it, couch says nothing. Park and detach never hit this
            because no pane is created for the address in the same completion; relaunch
            is the first operation that both ends a child and starts one.
            consumeExpectedParkExitLocked's own doc already states the right rule -
            "only the EXACT child selected by a Park attempt" - while the bridge selects
            by address. Fix - capture the dying pane ids before the attach dispatch, or
            exclude startedHandleID's pane, so the marker names the child expected to
            die rather than the address about to host two. Pin with a test that the
            adopted pane is absent from expectedExits and that its later exit notices.
          family: suppression-marker-overmatches
          round: 5
        - id: BR-18
          severity: Minor
          title: The in-flight frame exemption matches on address only, not on the operation that owns the frame
          detail: |-
            menu.go:1268 computes operationInFlight from state.InFlight.Address ==
            frame.Thread without comparing state.InFlight.Operation to frame.Action, so
            any confirmation opened on that thread while something is in flight is also
            exempted from the staleness check - reduceActionKey has no in-flight guard,
            only dispatchMenuOperation does (menu.go:1429). The consequence is a delayed
            rather than wrong refusal, but the exemption is broader than the comment and
            the recorded lesson both claim ("scope the exemption to that window"). Adding
            the operation to the comparison is one clause.
          family: exemption-wider-than-its-rationale
          round: 5
        - id: BR-19
          severity: Minor
          title: reduceParkHotkey builds its refusal wording from an inline two-entry map that silently yields an empty word
          detail: |-
            4th finding in this family; the family's routing/confirmation rule is now
            delivered via OperationConfirms and the Enter sweep, so this is residue on a
            different axis - wording, not routing. menu.go:506 allocates
            map[string]string{"park": "parked", "relaunch": "relaunched"} per call and
            indexes it with event.Operation; a third operation joining the guard two
            lines above produces "only a running thread can be ". Do not re-derive the
            whole family for this - a package-level map with a fallback word, or a
            declared past participle, closes it.
          family: declared-source-hand-maintained-consumers
          round: 5
      boundary: M1
      blocked: true
    - "n": 6
      timestamp: "2026-09-04T14:37:05-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: workshop/plans/000182-relaunch-an-actor-plan.md is unmodified in this window; the ~30s figure and both phantom budgets stand.
          round: 6
        - id: BR-6
          disposition: not-addressed
          note: relaunch_test.go:84-122 still has five cases; no agent-unsupported and no profile-missing case at the Relaunch level.
          round: 6
        - id: BR-7
          disposition: not-addressed
          note: Plan line 159 is still "- [ ]" and the issue's Revisions still says "the estimate is now stale"; additionally M2's chord/sweep work landed here while Tasks 8-10 stay unchecked.
          round: 6
        - id: BR-9
          disposition: not-addressed
          note: manifest.go:524 still places relaunch.go between pathops.go and procops.go; the new inputtrace.go entry was ordered correctly.
          round: 6
        - id: BR-10
          disposition: not-addressed
          note: Instance fixed and pinned (revert-verified); the class enumeration over InterceptorHit, README.md:141, menuControls and Tab-relaunch remain.
          round: 6
        - id: BR-16
          disposition: not-addressed
          note: Guard cannot fire on the production path; reproduced the original symptom against HEAD through ReduceMenu.
          round: 6
        - id: BR-17
          disposition: addressed
          note: Revert-verified red; pane ids are handle ids (console.go:1562), so the exclusion names the right child in production too.
          round: 6
        - id: BR-18
          disposition: not-addressed
          note: The Operation clause landed at menu.go:1301 but the whole couchtty suite stays green when it is reverted; no test enters the narrowed case.
          round: 6
        - id: BR-19
          disposition: not-addressed
          note: pastParticiple landed with a fallback but nothing tests it, and only park/relaunch can reach the guard, so the fallback has no caller.
          round: 6
      findings:
        - id: BR-20
          severity: Important
          title: endsItsOwnChild has one call site while the second list it was written to replace still enumerates by hand
          detail: |-
            This is the 5th finding in family `declared-source-hand-maintained-consumers`, so the deliverable is the RULE, not this
            site. console.go:1477 declares endsItsOwnChild and its doc says it exists
            "so the two sites that need the answer cannot disagree ... the expectedExits
            bridge and the switch below". Only the bridge (console.go:1431) calls it;
            consumeExpectedParkExitLocked at console.go:1499 still spells
            `case "park", "detach", "relaunch":` itself. The two agree today, so nothing
            is broken -- but the divergence the helper was written to remove is fully
            intact, and the comment asserts a property the code does not have, which is
            worse than no helper because the next reader trusts it.
            Rule: a per-operation fact has exactly one predicate and EVERY consumer calls
            it; a helper introduced for DRY with a single call site has not been adopted,
            it has been added. Measured prevalence in couchtty after this window: seven
            hand-maintained restatements of facts the Operation declaration already owns
            or could own -- consumeExpectedParkExitLocked (console.go:1499),
            operationNeedsProjectionRefresh (menu.go:1397), reduceOperationResult's case
            list (menu.go:1384), reduceParkHotkey's case list (menu.go:483),
            menuOperationProgressText (menu.go:1495), pastParticiple (menu.go:542), and
            menuActionItems' hardcoded per-state slices (menu.go:1068). OperationConfirms
            proves the shape works; write the enumeration and route these through it (or
            through fields on Operation) as one sweep, and fold it into Task 10 Step 2b
            which already owns the operation-side half. ARCH-DRY, ARCH-PURPOSE.
          family: declared-source-hand-maintained-consumers
          round: 6
        - id: BR-21
          severity: Minor
          title: The input tracer is opened inside New() from ambient env and never closed
          detail: |-
            console.go:163 calls newInputTracer() from the Console constructor, which
            reads os.Getenv("COUCH_INPUT_TRACE") (inputtrace.go:65) and opens a real file
            for append. Nothing closes it -- neither Stop() nor teardown() touches
            c.trace. In production that is one fd for the process lifetime, which is
            fine. In tests it is not: couchtty builds many Consoles per run, so a
            developer whose shell exports the variable (this repo has already been bitten
            by PAIR_SESSION_ID/PAIR_TAG leaking into `make test`) gets one open fd per
            constructed Console and a trace file polluted with fixture bytes. ARCH-SECURE's
            at-review lens names exactly this: tests able to write real user state.
            Take the path as a parameter from the composition root, or gate the Getenv
            behind the same seam the rest of couchtty uses, and close the file in
            teardown.
          family: constructor-io-from-ambient-env
          round: 6
      boundary: M1
      blocked: true
---

# Gate ledger — pair#182 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-04T10:47:03-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `operating-envelope` Two of the envelope's three durations name budgets that do not exist as constants
  Family repeat (2nd). Rule, not instance: every duration in the operating
  envelope must cite the constant producing it by file:line, and say so
  explicitly where the budget is derived rather than declared. Here the
  "5s exact-child-death wait (couch.go:119)" is not separate — child death is
  awaited inside the single 15s CompletionTimeout (park.go:549-555) — and the
  "10s blocked-start acknowledgement" is actually resumeRegistrationTimeout,
  5s (couch.go:107, launch_existing.go:110-111). Real worst case ~20s, not
  ~30s; the plan over-budgets, so no downstream decision changes.
  (carried from plan-quality PQ-7, deferred to the boundary review)

## Round 2 — 2026-09-04T10:47:03-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Important] `swallowed-cause-fabricated-diagnostic` A resolver IO failure is reported as "binding is not established", and the branch meant to catch it is dead code
  relaunch.go:96-108 — ResolveEstablished returns a ZERO resolution on a real
  IO error (artifactpath.Resolve / os.UserHomeDir / QuerySessionContext), so
  bindingResumeDiagnostic's default arm yields ResumeBindingUnbound and
  CheckResumePreconditions returns at line 103; `if bindingErr != nil` at 106
  is unreachable. Proved by deleting lines 106-108 in a scratch worktree and
  running -run 'Relaunch|Resume|Park': still green. ResumeContext
  (resume.go:279-283) returns the resolver's error directly, so the two
  callers of the same seam disagree. Resolve the binding only after the
  non-binding preconditions pass, return bindingErr when it is not a
  *ResumeRefusal, and pin it with an erroring-resolver case. ARCH-SECURE.
- **BR-3** [Important] `refusal-names-no-next-action` Relaunch on a thread that is not running refuses with `resume-live` and names no next action
  relaunch.go:81-83 maps every soleParkableIncarnation failure to
  refuseResume(ResumeLive, err.Error()), but park.go:768-783 fails for NO
  live/unknown incarnation as well as for two. On a parked row the operator
  gets "resume-live: park requires exactly one identified live or unknown
  incarnation" — a code contradicting the state and a message about park's
  internals. Reachable: the Spec's panel form relaunches the HIGHLIGHTED row,
  which is often parked. Split the cases; for a thread with no incarnation
  refuse with its own code naming the working gesture ("Enter resumes it").
- **BR-4** [Important] `parallel-derivation-drift` Resume's evidence-gathering preamble is re-derived in Relaunch, and the first copy already diverges
  relaunch.go:88-101 vs resume.go:262-283 both derive `agent` from the saved
  profile, type-assert c.Artifacts to NativeBindingResolver, and compute
  pathExists from c.Path.Physical. They already differ on nil Path: relaunch
  defaults pathExists to true and skips the call, so the precondition PASSES
  and ResumeContext then nil-derefs one step later. The plan extracted the
  RULES for exactly this reason; the evidence needs the same treatment —
  one resumeEvidence(ctx, thread, address) helper with one nil-Path policy.
  ARCH-DRY.
- **BR-5** [Important] `declared-source-hand-maintained-consumers` The declared operation reaches no operator surface, and the plan asserts it does
  menuActionItems (menu.go:1008) returns hardcoded slices and consumes
  Operations() nowhere; ParseCLI is a closed flag set. So the plan's Chunk 2
  premise ("reachable from the switcher's action list the moment it is
  declared") is false, and the Done-when bullet "an actor action relaunch
  appears alongside detach and park ... reachable from the same
  declared-operation surface" has no owning task in M1 or M2. The class is
  six hand-maintained per-operation sites, not the two Task 10 enumerates:
  menu.go:1008, confirmationMenuItems, menu.go:1306, menu.go:1320,
  console.go:1375, console.go:1425. Fix the plan now and sweep the
  enumeration in one M2 round. ARCH-PURPOSE, ARCH-DRY.
- **BR-6** [Minor] `done-when-untested` No Relaunch-level test for the agent-unsupported and profile-missing refusals
  The plan's Task 2 table lists four cases; three shipped. The rules are
  covered purely in precondition_test.go and the diagnostic PRECEDENCE is
  correct (profile checks run before the binding check, so an unsupported
  agent still yields ResumeAgentUnsupported even though ResolveEstablished
  was already called with it), but nothing pins that precedence through
  Relaunch.
- **BR-7** [Minor] `plan-record-lags-code` Task 1 Step 5 is unticked though committed, and the issue's stale-estimate warning contradicts the re-derived Estimate block
  plan line 159 ("Step 5: Commit") is unchecked although e7c6c6e8 landed it —
  milestone-close's plan-unchecked guard will refuse on it. Separately, the
  issue's ## Revisions still says "⚠ The estimate is now stale" while ##
  Estimate has since been re-derived for the grown scope (3.58 → 6.20).
- **BR-8** [Minor] `result-shape-forces-new-consumer-arm` RelaunchResult does not compose with finishOperation's existing ParkResult/StartResult arms
  relaunch.go:43 — console.go:1324 and :1328 already know how to land a
  ParkResult address and force-switch onto a StartResult handle.
  RelaunchResult{Outcome, Record, Handle} matches neither, so M2 adds a third
  arm. RelaunchResult{Outcome, Start StartResult} would reuse the resume arm
  unchanged.
- **BR-9** [Minor] `manifest-ordering` relaunch.go inserted out of alphabetical order in NonArtifactSources
  artifactpath/manifest.go:524 places it between pathops.go and procops.go.
  Nothing enforces the order; noted so the next insert does not compound it.

## Round 3 — 2026-09-04T11:04:55-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Plan lines 32-37 unchanged; couch.go:119 is the 15s CompletionTimeout and couch.go:107 sets resumeRegistrationTimeout to 5s, so ~20s worst case, not ~30s.
- BR-2 — addressed — Verified by revert: replacing the bindingErr guard with `_ = bindingErr` turns "the resolver itself fails" red with resume-binding-unbound.
- BR-3 — addressed — Verified by revert: deleting the hasOccupiedIncarnation gate turns "the thread is not running at all" red with resume-live.
- BR-4 — not-addressed — resumeEvidence has ONE caller (relaunch.go:99); ResumeContext (resume.go:298-321) still derives pathExists, the NativeBindingResolver assert and agent itself, with a different nil-Path policy, and the helper's "Shared because" comment is false at HEAD.
- BR-5 — addressed — Plan now carries the six-site table plus Step 2b, and the issue's M2 bullet names the sweep; I verified all six sites exist. Delivery is M2's.
- BR-6 — not-addressed — relaunch_test.go still has five cases; no agent-unsupported or profile-missing case reaches Relaunch.
- BR-7 — not-addressed — Plan:159 still unticked, and now Task 8's steps (472/476) too though 4821dda3 landed them; the stale-estimate warning still contradicts the re-derived block; the M1 table never gained resumeEvidence / hasOccupiedIncarnation / ResumeNotRunning.
- BR-8 — not-addressed — relaunch.go:42-46 unchanged. finishOperation does carry value alongside err, so the third arm is still what M2 must add.
- BR-9 — not-addressed — manifest.go:524 still places relaunch.go between pathops.go and procops.go.

### Raised

- **BR-10** [Important] `declared-source-hand-maintained-consumers` Alt+n is intercepted and then silently dropped: HitRelaunch has no arm in processInput's switch
  2nd finding in this family, so the deliverable is the RULE, not this site.
  keys.go:94 declares HitRelaunch and keys.go:255 intercepts both chords, but
  console.go:603-610 handles only Switch/Park/Previous/Detach and has no
  default -- so the chord is consumed off the child's input stream and does
  nothing. Before 4821dda3 it reached Pair and reloaded the workbench;
  README.md:141 still documents that, and menu.go:19 menuControls has no
  Alt+n row. Rule: a value the interceptor can emit must reach a handler by
  construction. Three per-hit sites still enumerate by hand (console.go's
  switch, keys.go:234, keys.go:237) after the commit that claimed to close
  the class. Write the enumeration -- a test walking every seqKind through
  hit() plus the legacy branches, asserting each non-HitNone value routes --
  and fold it into Step 2b, which already owns the operation-side half of the
  same rule. ARCH-DRY, ARCH-PURPOSE.
- **BR-11** [Minor] `review-window-degenerate` The gate handed this round a base == head window, so every diff recipe returned empty
  base and head were both 4a7d96e2, so stat, name-status and full diff were
  all empty and a reviewer following the recipes literally would have had
  nothing to inspect. I reviewed 9acfd8e5..4a7d96e2 instead, derived from the
  prior round's `Review-Window: 88fe1de0..9acfd8e5` trailer -- which is what
  brought the unreviewed commit 4821dda3 into scope. Worth a look at how the
  boundary computes BASE_SHA when the previous round's fix commit is HEAD.

## Round 4 — 2026-09-04T13:55:31-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Plan lines 32-37 unchanged; the plan file was not touched in this window at all.
- BR-4 — addressed — resumeEvidence now has two callers and ResumeContext derives none of the three facts itself; the consolidation's own side effect on the warm path is raised separately as a new Critical.
- BR-6 — not-addressed — relaunch_test.go still has five refusal cases; no agent-unsupported or profile-missing case reaches Relaunch.
- BR-7 — not-addressed — plan:159 still unticked; Task 8 and parts of Task 10/11 now shipped and unticked too; the issue's stale-estimate warning still contradicts the re-derived Estimate block.
- BR-8 — addressed — StartedChild generalises the arm instead of adding a third; verified by revert — restoring the concrete StartResult assertion reds TestRelaunchResultIsAdoptedByTheConsole/a_completed_relaunch_is_adopted.
- BR-9 — not-addressed — manifest.go:524 still places relaunch.go between pathops.go and procops.go; the new inputtrace.go entry was inserted in order, so the convention is being followed for new rows only.
- BR-10 — not-addressed — The instance is fixed and genuinely pinned (removing the case reds two byte-driven tests), but the requested deliverable was the enumeration and no test walks seqKind/InterceptorHit to a handler; the same class then shipped a new Critical at menu.go:549. README.md:141 and menuControls are also still unchanged.
- BR-11 — addressed — This round's window is a real range — 4a7d96e2..4ae9d278, 10 commits, 18 files; every diff recipe returned content.

### Raised

- **BR-12** [Critical] `declared-source-hand-maintained-consumers` relaunch is offered in the switcher's action list and Enter on it does nothing
  3rd finding in this family, so the deliverable is the RULE, not this site.
  menuActionItems (menu.go:1027) now returns "relaunch", but reduceActionKey's
  Enter switch (menu.go:549) routes only park/archive, detach/resume and
  name/describe and has no default -- so Enter on the row produces no frame,
  no effect and no notice. Confirmed with a scratch test through ReduceMenu:
  after Tab then Down, SelectedItem == "relaunch"; after Enter, frame.Action
  == "" and effects == []. That is the issue's FIRST Done-when bullet shipped
  half-wired, and the smoke test missed it because it drove only Alt+n.
  Rule - an item a surface can OFFER must be routed by construction.
  Measured prevalence this round - the plan names six per-operation sites,
  the sweep touched twelve (menu.go 467, 502, 506, 609, 1027,
  confirmationMenuItems, 1258, 1308, 1337, 1351, 1462; console.go 1467,
  1487), and two remain - menu.go:549 and console.go:1445, whose landing arm
  is still a literal == "resume" though RelaunchResult now satisfies
  StartedChild. Extend Step 2b's enumeration from "appears in menuActionItems"
  to "and Enter on it yields a dispatch effect or a frame". ARCH-DRY,
  ARCH-PURPOSE.
- **BR-13** [Critical] `shared-helper-widens-caller-contract` The resumeEvidence consolidation makes a warm reattach resolve a binding it never resolved, and fail when that resolver errors
  resume.go:347-353 - the VerifiedPark == nil branch now calls resumeEvidence,
  which always calls ResolveEstablished and whose binding it then DISCARDS; it
  wanted only pathExists. Any non-ResumeRefusal error now aborts the warm
  reattach. At 4a7d96e2 that branch called the resolver zero times. Proved
  differentially with one test in a scratch worktree - PASS at base, at HEAD
  "warm reattach refused because the binding resolver failed: resolve home
  directory: permission denied". The comment three lines above still states the
  old contract ("Resolve the binding only where it is the authority ... which
  is why relaxing that alone left the operator's detached thread
  unreachable"), so code and contract now disagree, and nothing pins either
  direction - TestDetachedResumeDoesNotRequireAnEstablishedBinding is a pure
  DecideResume test. Reachable causes are ordinary - ListFiles ErrStorageAbsent
  and ReadFile ErrReadLimit once a ledger sidecar passes 8 MB
  (sessioninventory/runtime_os.go:157), besides os.UserHomeDir. Split the
  helper along the axis the callers differ on so the warm branch pays no
  resolver call, keeping ONE nil-Path policy; pin both directions.
  ARCH-CONSTRAINTS (the warm path now pays a ListFiles, an up-to-8MB read,
  proof validation and a possible catalog write, and throws the answer away).
- **BR-14** [Important] `new-surface-undocumented` COUCH_INPUT_TRACE is a new operator surface that records every keystroke and is documented nowhere
  inputtrace.go:59. A tree-wide grep finds the name only in inputtrace.go, one
  console.go comment and the issue Log - not README.md, not atlas/couch.md, not
  menuControls. Two things need saying together - that it exists and how to
  enable it, and WHAT it captures. pumpStdin traces every chunk before the
  Interceptor splits it, so the file holds everything typed or pasted into the
  hosted agent, prompts and credentials included. 0600 and opt-in are the right
  defaults; the gap is telling the operator at the moment they enable it. Add
  it to atlas/couch.md beside COUCH_STORE_DIR / COUCH_THREAD_*. ARCH-SECURE.
- **BR-15** [Minor] `swallowed-cause-fabricated-diagnostic` newInputTracer swallows its open error, so a failed probe reads as "the terminal sent nothing"
  2nd finding in this family, so the deliverable is the RULE, not this site.
  inputtrace.go:58-67 returns nil when os.OpenFile fails, so an unwritable
  COUCH_INPUT_TRACE path yields an empty trace indistinguishable from "no bytes
  arrived" - the exact ambiguity the probe exists to remove, and the same shape
  as BR-2's zero resolution read as "unbound". Rule - an instrument that cannot
  start must say so; the INABILITY to observe must never be presentable as an
  observation. A one-line setNotice on the open failure satisfies it without
  letting a diagnostic take the console down.

## Round 5 — 2026-09-04T14:17:13-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Plan lines 32-37 unchanged; the plan file was not touched in this window at all.
- BR-6 — not-addressed — relaunch_test.go untouched this window; its five refusal cases still include no agent-unsupported or profile-missing case reaching Relaunch.
- BR-7 — not-addressed — plan:159 still unticked, Tasks 8/10/11 still unticked though shipped, and the issue's stale-estimate warning still contradicts the re-derived Estimate block.
- BR-9 — not-addressed — manifest.go:524 still places relaunch.go between pathops.go and procops.go.
- BR-10 — not-addressed — No test walks seqKind/InterceptorHit to a processInput arm; README.md:141 and menuControls are still unchanged, so the chord ships undocumented and the class deliverable is still the instance.
- BR-12 — addressed — Verified by revert - restoring the old park/archive+detach/resume switch reds TestEveryOfferedActionIsReachableFromEnter/live/relaunch. console.go:1451's literal is deliberate for M1 and noted as Minor.
- BR-13 — addressed — Verified by revert - making ResumeContext resolve unconditionally reds TestWarmReattachNeitherConsultsNorIsFailedByTheBindingResolver on both assertions.
- BR-14 — addressed — atlas/couch.md:764 documents the variable, the 0600 opt-in default, and that the pumpStdin tap captures everything typed or pasted into the hosted agent.
- BR-15 — addressed — newInputTracer returns an error and New pushes a control-priority notice; TestATraceThatCannotStartSaysSoInsteadOfTracingNothing asserts both halves.

### Raised

- **BR-16** [Critical] `refusal-names-no-next-action` A relaunch that parks and then fails has its recovery message erased by the next refresh
  2nd finding in this family, so the deliverable is the RULE, not this site.
  menu.go:1320 excludes relaunch from the park/leave confirmation-close, and
  the comment justifies that by the transient RefusedBeforePark refusal. But
  for ParkIncomplete and ParkedNotResumed the thread is NOT live, InFlight is
  already cleared at menu.go:1306, and the refresh finishOperation always
  requests (console.go:1456 -> menu.go:317 -> reconcileMenuFrames) hits the
  "!operationInFlight && !thread.Live()" arm at menu.go:1271 - discarding the
  confirmation AND overwriting the notice. Reproduced differentially - after
  the result the notice is "relaunch brain: it is parked and the restart did
  not take; Enter resumes it"; after one refresh it is "thread action is no
  longer applicable" and the frames are back at root. That is the issue's
  Done-when bullet ("leaves a resumable parked thread AND SAYS SO") true in
  couchcore and false at the surface, on the destructive path, reading as the
  data loss the Spec calls the whole design question. Rule - a notice carrying
  a just-completed operation's outcome outranks a frame-reconciliation notice,
  and reconciliation must not invalidate a frame on state the operation it
  belongs to produced. The ownership hook exists already
  (Notice.Owner = MenuProgressOwner{OperationAttempt}, menu.go:1324); use it
  rather than adding relaunch to another hand-written list. Measured
  prevalence - park and leave close their frame, archive is exempted by name,
  detach and resume have no confirmation, so relaunch is the only current
  instance and every future confirmed operation inherits it. ARCH-PURPOSE.
- **BR-17** [Important] `suppression-marker-overmatches` The child a successful relaunch adopts is pre-marked as an expected exit, so its first real death is silent
  console.go:1427 walks c.panes and marks every pane whose thread == address,
  but the attach dispatched at console.go:1385 has already installed the NEW
  child's pane synchronously (installObservedThreadActor). Verified with a
  dispatcher that installs a pane on attach - expectedExits["new-child"] is
  true when finishOperation returns. onExit (console.go:768) consumes the
  marker and suppresses ExitNotice, so if the freshly relaunched Pair crashes
  or the operator quits it, couch says nothing. Park and detach never hit this
  because no pane is created for the address in the same completion; relaunch
  is the first operation that both ends a child and starts one.
  consumeExpectedParkExitLocked's own doc already states the right rule -
  "only the EXACT child selected by a Park attempt" - while the bridge selects
  by address. Fix - capture the dying pane ids before the attach dispatch, or
  exclude startedHandleID's pane, so the marker names the child expected to
  die rather than the address about to host two. Pin with a test that the
  adopted pane is absent from expectedExits and that its later exit notices.
- **BR-18** [Minor] `exemption-wider-than-its-rationale` The in-flight frame exemption matches on address only, not on the operation that owns the frame
  menu.go:1268 computes operationInFlight from state.InFlight.Address ==
  frame.Thread without comparing state.InFlight.Operation to frame.Action, so
  any confirmation opened on that thread while something is in flight is also
  exempted from the staleness check - reduceActionKey has no in-flight guard,
  only dispatchMenuOperation does (menu.go:1429). The consequence is a delayed
  rather than wrong refusal, but the exemption is broader than the comment and
  the recorded lesson both claim ("scope the exemption to that window"). Adding
  the operation to the comparison is one clause.
- **BR-19** [Minor] `declared-source-hand-maintained-consumers` reduceParkHotkey builds its refusal wording from an inline two-entry map that silently yields an empty word
  4th finding in this family; the family's routing/confirmation rule is now
  delivered via OperationConfirms and the Enter sweep, so this is residue on a
  different axis - wording, not routing. menu.go:506 allocates
  map[string]string{"park": "parked", "relaunch": "relaunched"} per call and
  indexes it with event.Operation; a third operation joining the guard two
  lines above produces "only a running thread can be ". Do not re-derive the
  whole family for this - a package-level map with a fallback word, or a
  declared past participle, closes it.

## Round 6 — 2026-09-04T14:37:05-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — workshop/plans/000182-relaunch-an-actor-plan.md is unmodified in this window; the ~30s figure and both phantom budgets stand.
- BR-6 — not-addressed — relaunch_test.go:84-122 still has five cases; no agent-unsupported and no profile-missing case at the Relaunch level.
- BR-7 — not-addressed — Plan line 159 is still "- [ ]" and the issue's Revisions still says "the estimate is now stale"; additionally M2's chord/sweep work landed here while Tasks 8-10 stay unchecked.
- BR-9 — not-addressed — manifest.go:524 still places relaunch.go between pathops.go and procops.go; the new inputtrace.go entry was ordered correctly.
- BR-10 — not-addressed — Instance fixed and pinned (revert-verified); the class enumeration over InterceptorHit, README.md:141, menuControls and Tab-relaunch remain.
- BR-16 — not-addressed — Guard cannot fire on the production path; reproduced the original symptom against HEAD through ReduceMenu.
- BR-17 — addressed — Revert-verified red; pane ids are handle ids (console.go:1562), so the exclusion names the right child in production too.
- BR-18 — not-addressed — The Operation clause landed at menu.go:1301 but the whole couchtty suite stays green when it is reverted; no test enters the narrowed case.
- BR-19 — not-addressed — pastParticiple landed with a fallback but nothing tests it, and only park/relaunch can reach the guard, so the fallback has no caller.

### Raised

- **BR-20** [Important] `declared-source-hand-maintained-consumers` endsItsOwnChild has one call site while the second list it was written to replace still enumerates by hand
  This is the 5th finding in family `declared-source-hand-maintained-consumers`, so the deliverable is the RULE, not this
  site. console.go:1477 declares endsItsOwnChild and its doc says it exists
  "so the two sites that need the answer cannot disagree ... the expectedExits
  bridge and the switch below". Only the bridge (console.go:1431) calls it;
  consumeExpectedParkExitLocked at console.go:1499 still spells
  `case "park", "detach", "relaunch":` itself. The two agree today, so nothing
  is broken -- but the divergence the helper was written to remove is fully
  intact, and the comment asserts a property the code does not have, which is
  worse than no helper because the next reader trusts it.
  Rule: a per-operation fact has exactly one predicate and EVERY consumer calls
  it; a helper introduced for DRY with a single call site has not been adopted,
  it has been added. Measured prevalence in couchtty after this window: seven
  hand-maintained restatements of facts the Operation declaration already owns
  or could own -- consumeExpectedParkExitLocked (console.go:1499),
  operationNeedsProjectionRefresh (menu.go:1397), reduceOperationResult's case
  list (menu.go:1384), reduceParkHotkey's case list (menu.go:483),
  menuOperationProgressText (menu.go:1495), pastParticiple (menu.go:542), and
  menuActionItems' hardcoded per-state slices (menu.go:1068). OperationConfirms
  proves the shape works; write the enumeration and route these through it (or
  through fields on Operation) as one sweep, and fold it into Task 10 Step 2b
  which already owns the operation-side half. ARCH-DRY, ARCH-PURPOSE.
- **BR-21** [Minor] `constructor-io-from-ambient-env` The input tracer is opened inside New() from ambient env and never closed
  console.go:163 calls newInputTracer() from the Console constructor, which
  reads os.Getenv("COUCH_INPUT_TRACE") (inputtrace.go:65) and opens a real file
  for append. Nothing closes it -- neither Stop() nor teardown() touches
  c.trace. In production that is one fd for the process lifetime, which is
  fine. In tests it is not: couchtty builds many Consoles per run, so a
  developer whose shell exports the variable (this repo has already been bitten
  by PAIR_SESSION_ID/PAIR_TAG leaking into `make test`) gets one open fd per
  constructed Console and a trace file polluted with fixture bytes. ARCH-SECURE's
  at-review lens names exactly this: tests able to write real user state.
  Take the path as a parameter from the composition root, or gate the Getenv
  behind the same seam the rest of couchtty uses, and close the file in
  teardown.

## Open findings

- **BR-1** [Minor] `operating-envelope` Two of the envelope's three durations name budgets that do not exist as constants
- **BR-6** [Minor] `done-when-untested` No Relaunch-level test for the agent-unsupported and profile-missing refusals
- **BR-7** [Minor] `plan-record-lags-code` Task 1 Step 5 is unticked though committed, and the issue's stale-estimate warning contradicts the re-derived Estimate block
- **BR-9** [Minor] `manifest-ordering` relaunch.go inserted out of alphabetical order in NonArtifactSources
- **BR-10** [Important] `declared-source-hand-maintained-consumers` Alt+n is intercepted and then silently dropped: HitRelaunch has no arm in processInput's switch
- **BR-16** [Critical] `refusal-names-no-next-action` A relaunch that parks and then fails has its recovery message erased by the next refresh
- **BR-18** [Minor] `exemption-wider-than-its-rationale` The in-flight frame exemption matches on address only, not on the operation that owns the frame
- **BR-19** [Minor] `declared-source-hand-maintained-consumers` reduceParkHotkey builds its refusal wording from an inline two-entry map that silently yields an empty word
- **BR-20** [Important] `declared-source-hand-maintained-consumers` endsItsOwnChild has one call site while the second list it was written to replace still enumerates by hand
- **BR-21** [Minor] `constructor-io-from-ambient-env` The input tracer is opened inside New() from ambient env and never closed
