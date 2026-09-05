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
    - "n": 7
      timestamp: "2026-09-04T14:46:26-07:00"
      agent: claude
      boundary: M1
      blocked: true
      protocol_error: no valid findings block
    - "n": 8
      timestamp: "2026-09-04T15:02:20-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan lines 32-37 untouched this window; 15s CompletionTimeout absorbs the child-death wait and the ack timeout is 5s, so ~20s not ~30s.
          round: 8
        - id: BR-6
          disposition: not-addressed
          note: relaunch_test.go:76 still has five cases; agent-unsupported and profile-missing are pinned only at CheckResumePreconditions.
          round: 8
        - id: BR-7
          disposition: not-addressed
          note: plan:159 still unticked, Tasks 8/10/11 shipped unticked, the stale-estimate warning stands, and the M1 table gained none of the seven new entities.
          round: 8
        - id: BR-9
          disposition: not-addressed
          note: manifest.go:524 still places relaunch.go between pathops.go and procops.go; the new inputtrace.go row was ordered correctly.
          round: 8
        - id: BR-10
          disposition: not-addressed
          note: Instance fixed and revert-verified; no enumeration from InterceptorHit to a handler, and README.md:141/376-391 plus menuControls still carry no Alt+n or relaunch row.
          round: 8
        - id: BR-16
          disposition: addressed
          note: 'Revert-verified both halves: clearsPreviousNotice and the setBookkeepingNotice owner guard each red TestAFailedRelaunchKeepsItsRecoveryMessageAcrossARefresh through ReduceMenu.'
          round: 8
        - id: BR-18
          disposition: addressed
          note: 'Revert-verified: dropping the InFlight.Operation clause reds TestTheInFlightExemptionDoesNotCoverAnotherActionsConfirmation.'
          round: 8
        - id: BR-19
          disposition: addressed
          note: pastParticiple has a fallback and a direct test covering an operation it was not written for; "archiveed" is a cosmetic wart, not a silent empty word.
          round: 8
        - id: BR-20
          disposition: addressed
          note: consumeExpectedParkExitLocked now calls endsItsOwnChild, so the helper's doc comment is true; the wider four-site sweep stays owned by Task 10 Step 2b, and the newly covered relaunch arm there has no test (see coverage notes).
          round: 8
        - id: BR-21
          disposition: addressed
          note: New() no longer reads env, the path comes from the composition root, failure is surfaced, teardown closes; pinned by TestAConsoleOpensNoTraceUnlessAskedTo.
          round: 8
      findings:
        - id: BR-22
          severity: Important
          title: bindingRefusalDiagnostic has one consumer while the real resolver and its stateful fake still hand-write the sentence it replaced
          detail: |-
            6th in this family, so the deliverable is the RULE, not the site: a helper
            introduced to replace a literal is not adopted until no production file
            still contains that literal. resume.go:305 and artifactcollision_fake.go:124
            both still return refuseResume(code, "native session binding is not one
            exact established root"). ResumeContext:349-353 propagates that error before
            DecideResume/CheckResumePreconditions can re-derive the usable sentence, so
            Enter on a parked row with a provisional binding -- the recovery relaunch's
            own ParkedNotResumed message names -- and relaunch's own post-park resume
            failure both land on the developer sentence that 54052fab, lessons.md and
            atlas/couch.md:139-146 all claim was eliminated. Fix the class: one
            refuseBinding(code) called from all three sites, plus a conformance
            assertion that the real resolver and the fake produce the same code AND the
            same message per BindingStatus -- the fake currently conforms to the wrong
            wording, which is what will keep it wrong. ARCH-DRY, ARCH-PURPOSE, ARCH-MOCK.
          family: declared-source-hand-maintained-consumers
          round: 8
        - id: BR-23
          severity: Minor
          title: pumpStdin reads c.trace without c.mu while SetInputTrace and teardown write it under the lock
          detail: |-
            console.go:1127 loads c.trace unsynchronized; console.go:200-203 and
            723-725 store it under c.mu. teardown nils the field before c.Stop() and
            before the workers are joined, so pumpStdin can still be in flight. record
            is nil- and closed-safe, so the defect is the field access itself: latent
            under `make test-race` (go test -race ./cmd/...), not reproducible today.
            Read it under c.mu in pumpStdin, or hold the tracer in an atomic.Pointer.
          family: field-read-outside-its-mutex
          round: 8
      boundary: M1
      blocked: false
    - "n": 9
      timestamp: "2026-09-04T16:37:19-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan envelope untouched; verified 15s CompletionTimeout (couch.go:119) + 5s resumeRegistrationTimeout (couch.go:107) = ~20s, not ~30s.
          round: 9
        - id: BR-6
          disposition: not-addressed
          note: relaunch_test.go still has five cases; agent-unsupported and profile-missing remain pinned only at CheckResumePreconditions.
          round: 9
        - id: BR-7
          disposition: not-addressed
          note: plan Task 1 Step 5 still unticked; the stale-estimate warning still contradicts the re-derived Estimate. Subsumed by the rule in the new plan-record-lags-code finding.
          round: 9
        - id: BR-9
          disposition: not-addressed
          note: manifest.go:524 unchanged, and threadreason.go:503 is a second offender; one sort assertion over the manifest lists closes the family.
          round: 9
        - id: BR-10
          disposition: not-addressed
          note: Instance fixed and revert-verified (deleting the arm reds the chord test), but no enumeration from InterceptorHit to a handler exists, and README:141 plus menuControls still carry no Alt+n row.
          round: 9
        - id: BR-22
          disposition: addressed
          note: refuseBinding is now the only binding-refusal constructor and is called from resume.go:211, resume.go:316 and artifactcollision_fake.go:126; the fake's wording is pinned by test.
          round: 9
        - id: BR-23
          disposition: addressed
          note: pumpStdin now loads c.trace under c.mu (console.go:1195-1200); no test pins it, which is inherent for a latent race.
          round: 9
      findings:
        - id: BR-24
          severity: Important
          title: The pair#186 split was written into one artifact and propagated to none of its three consumers
          detail: |-
            2nd in this family, so the deliverable is the RULE: a scope change is not
            recorded until every artifact restating the scope derives from or cites the
            one entry that made it. Enumeration, all live: the plan doc has no
            "## Revisions" at all and its M2 Core-concepts table still declares
            paneState and RenderHoldingPane in couchtty/holding.go, none of which exist
            anywhere in the tree, with Tasks 8/10/11/12 carrying unticked steps though
            Task 8 and Step 2b shipped; workshop/projects/couch.md:201 still describes
            M2 as "the gesture and a surface that outlives its child" and lists no
            pair#186 row; the issue's "## Done when" still claims "the operator ends on
            the same actor, not in the switcher" while console.go:1383-1387 sets
            FocusPanel unconditionally and its own comment says why; and BR-7's stale
            estimate warning still stands. Also: "## Plan" marks M2 [x] with no
            Review-Verdict trailer and no "closed M2" log line in the range.
          family: plan-record-lags-code
          round: 9
        - id: BR-25
          severity: Important
          title: Both guards written to catch this class read a hand-maintained list, so the new chord and the new operation walked past them
          detail: |-
            7th in this family. Do NOT fix these two sites -- the rule is: a guard whose
            source is a hand-maintained list is a guard the next addition can skip; it
            must derive from the table the production path consumes. (1)
            TestREADMEDocumentsEveryPanelControl (couchcmd/readme_test.go:61) exists so
            "adding a key to the UI makes this test fail until its documentation has a
            home in README" -- it iterates menuControls (menu.go:19). Alt+n and
            Ctrl+Alt+n went into knownSequences (keys.go:170-171) and never into
            menuControls, so the guard could not fire and README.md:141 still tells the
            operator Alt+n reloads pair in place in "any pane", which is false inside
            couch. (2) TestEveryOfferedActionIsReachableFromEnter tests the CONVERSE of
            plan Task 10 Step 2b: it walks menuActionItems asserting each offered action
            is reachable, where the plan asked it to walk Operations() asserting each
            switcher-reachable PresentationTUI operation is OFFERED. Offered-implies-
            reachable cannot catch declared-but-unreachable, the exact failure it was
            written for, and the issue's Plan line claims otherwise. Cheap fix: a
            row-action field on Operation that menuActionItems derives membership from,
            which also collapses several of the seven restatements still standing.
          family: declared-source-hand-maintained-consumers
          round: 9
        - id: BR-26
          severity: Minor
          title: onRelaunchHotkey's panel branch is production code with no test, and the Done-when bullet it serves is unpinned
          detail: |-
            2nd in this family, so the rule rather than the site: the enumeration IS the
            Done-when list -- every bullet the issue still claims must cite the test
            that pins it or the Revisions line that moved it. console.go:1378-1380
            (panel focus takes the highlighted row) is never exercised; only the
            actor-focus branch is driven by console_relaunch_chord_test.go. Done-when
            "Alt+n on the switcher relaunches the highlighted row and leaves the
            operator in the switcher" and plan Task 10 Step 3's "two tests, because the
            endings differ" are both unmet.
          family: done-when-untested
          round: 9
        - id: BR-27
          severity: Minor
          title: The gate handed this round the whole 111-commit branch, including three other issues' already-closed work
          detail: |-
            2nd in this family (BR-11 was the empty variant of the same rule). Base
            88fe1de is the branch point, so the window is 164 files and +24177/-4669
            spanning pair#170, pair#181 and pair#185, each of which already passed its
            own close gate with its own review doc in this very diff. The window should
            be the un-reviewed span: M1's close at b7ec5e64, or at most the first #182
            commit. I reviewed b18f958e..HEAD for the code, which is #182 plus the
            #183/#184/#185/#186 issue-sync commits.
          family: review-window-degenerate
          round: 9
      blocked: true
    - "n": 10
      timestamp: "2026-09-04T16:56:08-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: plan lines 32-35 unchanged; measured CompletionTimeout=15s (couch.go:119) and resumeRegistrationTimeout=5s (couch.go:107), child death awaited inside the 15s (park.go:552).
          round: 10
        - id: BR-6
          disposition: not-addressed
          note: relaunch_test.go:76 still has five cases; no agent-unsupported or profile-missing case at the Relaunch level.
          round: 10
        - id: BR-7
          disposition: not-addressed
          note: the stale-estimate half is resolved in the issue's Revisions; plan lines 159 and 352 are still unticked for work that landed (e7c6c6e8, b7ec5e64) and the new Revisions note does not name them.
          round: 10
        - id: BR-9
          disposition: not-addressed
          note: artifactpath/manifest.go:524 still places relaunch.go between pathops.go and procops.go.
          round: 10
        - id: BR-10
          disposition: addressed
          note: hitHandlers table plus a knownSequences-driven test; the load-bearing direction is derived from the production table. Residual raised as new (intercepts() is still hand-maintained).
          round: 10
        - id: BR-24
          disposition: addressed
          note: plan gained a Revisions section naming the moved tasks and the entities declared-but-absent; project row and the FocusPanel Done-when corrected; M2's Mx tag dropped. Remaining unswept consumers raised as new.
          round: 10
        - id: BR-25
          disposition: addressed
          note: menuControls and README both fixed; RowAction gives membership one source and the declared-to-offered direction is mutation-verified red. The converse direction's unfalsifiability raised as new.
          round: 10
        - id: BR-26
          disposition: not-addressed
          note: console.go:1372-1373 (panel branch takes the highlighted row) still has no test; console_relaunch_chord_test.go drives only the actor branch.
          round: 10
        - id: BR-27
          disposition: withdrawn
          note: sdlc documents branch-point..HEAD as the whole-issue integration window by design (ariadne close.go:975-987), so the base is not a gate defect; the branch carrying four issues is a hygiene fact, not a window bug.
          round: 10
      findings:
        - id: BR-28
          severity: Important
          title: intercepts() and hit() are still two hand-maintained switches over one enum, and the new guard skips whatever intercepts() forgets
          detail: |-
            8th in this family, so the deliverable is the enumeration, not the site. keys.go:50-76 keeps two switches over seqKind where hit() alone is the
            mapping, and console_relaunch_chord_test.go:334 gates its sweep on intercepts() -- the list that can be wrong. Verified in a scratch worktree:
            removing seqRelaunch from intercepts() leaves TestEveryInterceptedChordHasAHandler green. keys.go:270-276 records that this exact omission has
            already shipped once. Fix: intercepts() returns k.hit() != HitNone, so one list remains and the test's skip clause derives from it. ARCH-DRY, ARCH-PURPOSE.
          family: declared-source-hand-maintained-consumers
          round: 10
        - id: BR-29
          severity: Important
          title: menuActionItems filters through declaredRowActions, so the sweep's offered-implies-declared direction can never fail and its failure is now a silent UI drop
          detail: |-
            menu.go:1085-1097 filters the row's offer through couchcore.RowActions() before menu_action_sweep_test.go:96-100 reads it, so offered is a subset of
            declared by construction. Verified: adding "bogus" to the live row in menuActionItemsForState produces zero test failures and the item simply
            disappears from the switcher. The rule: a guard must be able to fail, and production must not coerce its input into agreement. Fix: build the test's
            offered set from menuActionItemsForState, pre-filter.
          family: guard-unfalsifiable-by-construction
          round: 10
        - id: BR-30
          severity: Important
          title: the pair#186 split is still unpropagated into four artifacts, including the two the fix edited
          detail: |-
            3rd in this family; the rule is already in lessons.md, so the deliverable is the sweep. Live: the issue's Done-when bullet 2 still claims the
            rebuilt-binary verification that Revisions moved to pair#186 (the smoke test proved conversation survival, not a rebuilt binary), and the
            ctrl+backspace bullet is likewise unmarked with no test; the plan's M2 Integration-points table still lists onExit/finishOperation as holding-pane
            modifications and describes onRelaunchHotkey as returning to the actor, contradicting the code and the corrected Done-when; projects/couch.md:201
            still labels the row [pair#182 M2] with no detail block though the Mx tag was dropped. ARCH-PURPOSE on bullet 2 -- the rebuilt binary is the point of the issue.
          family: plan-record-lags-code
          round: 10
        - id: BR-31
          severity: Important
          title: the Alt+n interception reached neither atlas/couch.md's chord section nor README's couch section, and the README guard could not notice
          detail: |-
            2nd in this family. atlas/couch.md:436 has "Alt+d is Couch's own detach (pair#170), intercepted like Alt+x" with no Alt+n counterpart, and line 355
            still frames the set as "two lifecycle chords"; README.md:382-389 does the same, documenting Alt+n only inside Pair's own keybinding table, and
            neither records the switcher-row form or the deliberate FocusPanel ending. TestREADMEDocumentsEveryPanelControl substring-matches control.Keys
            against the whole README, so the pre-existing Pair-table "Alt+n" satisfied it. Fix: an atlas paragraph beside Alt+d, a couch-section README sentence,
            and scope the README guard to the couch section.
          family: new-surface-undocumented
          round: 10
        - id: BR-32
          severity: Minor
          title: menuActionItems rebuilds the whole Operations() table on every action-menu render
          detail: |-
            menu.go:1096 calls couchcore.RowActions(), which constructs ~15 Operation structs with ArgSpec slices, once per render at menu_render.go:200 -- a
            keystroke-path render. Negligible in absolute terms; a package-level memoized set costs nothing.
          family: operating-envelope
          round: 10
        - id: BR-33
          severity: Minor
          title: processInput still consumes the chord bytes when a hit has no handler
          detail: |-
            console.go:636-638 drops the hit silently on the nil branch, so a gap is invisible at runtime even though the test now catches it at build time. A
            status notice there would make the missing case observable to an operator.
          family: declared-source-hand-maintained-consumers
          round: 10
      blocked: true
    - "n": 11
      timestamp: "2026-09-04T17:39:21-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: plan lines 32-37 untouched; measured 15s CompletionTimeout (couch.go:119) + 5s resumeRegistrationTimeout (couch.go:107), child death inside the 15s (park.go:552).
          round: 11
        - id: BR-6
          disposition: not-addressed
          note: relaunch_test.go:76 still has five refusal cases; no agent-unsupported or profile-missing case reaches Relaunch.
          round: 11
        - id: BR-7
          disposition: not-addressed
          note: plan lines 159 and 352 still unticked for landed work (e7c6c6e8, b7ec5e64); the new Revisions note names only Task 8's boxes.
          round: 11
        - id: BR-9
          disposition: not-addressed
          note: manifest.go:524 still places relaunch.go between pathops.go and procops.go; threadreason.go:503 is still the second offender.
          round: 11
        - id: BR-26
          disposition: not-addressed
          note: console.go:1379 still has no test, and the untested branch turns out to be wrong — raised as a new Important finding.
          round: 11
        - id: BR-28
          disposition: addressed
          note: intercepts() is now k.hit() != HitNone (keys.go:72); revert-verified — deleting the seqRelaunch arm of hit() reds five chord tests.
          round: 11
        - id: BR-29
          disposition: addressed
          note: declaredRowActions and RowActions() deleted; the sweep reads Operation.RowAction directly. Mutation-verified — adding "bogus" to the live row reds both directions.
          round: 11
        - id: BR-30
          disposition: addressed
          note: Done-when bullets 2/ctrl+backspace/child-exited, the plan's integration-points table and projects/couch.md:201 all now carry the pair#186 split.
          round: 11
        - id: BR-31
          disposition: addressed
          note: atlas/couch.md:438-455 and README.md:387-397 both document the chord, the scope deviation and the FocusPanel ending; readme_test.go:58-77 scopes the guard and fatals on a missing marker.
          round: 11
        - id: BR-32
          disposition: addressed
          note: the per-render Operations() rebuild is gone with declaredRowActions; menu.go no longer calls couchcore.RowActions() at all.
          round: 11
        - id: BR-33
          disposition: addressed
          note: console.go:639-644 sets a status notice on the nil-handler branch, so a gap is observable at runtime and not only at build time.
          round: 11
      findings:
        - id: BR-34
          severity: Important
          title: onRelaunchHotkey reads the selection off CurrentFrame(), so Alt+n from the switcher refuses whenever the operator has drilled into a frame
          detail: |-
            console.go:1379 takes c.menu.CurrentFrame().SelectedAddress, but SelectedAddress is
            populated only on the ROOT frame — the actions frame is built with Thread set and
            SelectedAddress zero (menu.go:457-459), as are confirmation and text frames. So from
            any non-root frame the target is the zero address and the operator gets
            "relaunch: no thread selected" on the status row while the row is highlighted and its
            action list is open. Verified in a scratch worktree: from the root frame the panel
            branch opens the relaunch confirmation; after one Tab the same call changes nothing.
            Reachable on an ordinary path — Alt+n pressed a second time on the confirmation it
            just opened hits it too. 3rd in this family, so the deliverable is the rule, not the
            line: every Done-when bullet still claimed must cite the test that pins it or the
            Revisions line that moved it. Eight bullets are still claimed here; seven cite a test,
            and the one that does not is the one that is broken. Fix: read the root selection
            (c.menu.Frames[0].SelectedAddress, length-guarded), which is what reduceParkHotkey
            already collapses to. ARCH-PURPOSE.
          family: done-when-untested
          round: 11
        - id: BR-35
          severity: Important
          title: this plan's Core concepts table is enforced by no contract test, and two of its rows name symbols that exist nowhere
          detail: |-
            TestCoreConceptsContract exists to turn a drifting architecture table into an
            executable contract, but it reads conceptPlans (core_concepts_contract_test.go:206-214),
            a literal list of two filenames; couchcore/plan_contract_test.go pins 000149 and 000151
            the same way. 000182-relaunch-an-actor-plan.md is in neither, so its table is prose.
            Live: the M2 table declares paneState (console.go, "new") and RenderHoldingPane
            (cmd/internal/couchtty/holding.go, "new"); grep -rn 'paneState|RenderHoldingPane' cmd/
            returns nothing and holding.go does not exist. The Revisions entry disclaims them in
            prose, which is why this is Important and not Critical. 10th in this family, so the
            rule: a guard whose input is a hand-maintained list is a guard the next addition can
            skip. The enumeration is "every plan under workshop/plans/ carrying a Core concepts
            table", discovered by scanning the directory, with the existing planned-status skip
            carrying rows whose work has moved to pair#186. ARCH-DRY.
          family: declared-source-hand-maintained-consumers
          round: 11
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

## Round 7 — 2026-09-04T14:46:26-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 8 — 2026-09-04T15:02:20-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Plan lines 32-37 untouched this window; 15s CompletionTimeout absorbs the child-death wait and the ack timeout is 5s, so ~20s not ~30s.
- BR-6 — not-addressed — relaunch_test.go:76 still has five cases; agent-unsupported and profile-missing are pinned only at CheckResumePreconditions.
- BR-7 — not-addressed — plan:159 still unticked, Tasks 8/10/11 shipped unticked, the stale-estimate warning stands, and the M1 table gained none of the seven new entities.
- BR-9 — not-addressed — manifest.go:524 still places relaunch.go between pathops.go and procops.go; the new inputtrace.go row was ordered correctly.
- BR-10 — not-addressed — Instance fixed and revert-verified; no enumeration from InterceptorHit to a handler, and README.md:141/376-391 plus menuControls still carry no Alt+n or relaunch row.
- BR-16 — addressed — Revert-verified both halves: clearsPreviousNotice and the setBookkeepingNotice owner guard each red TestAFailedRelaunchKeepsItsRecoveryMessageAcrossARefresh through ReduceMenu.
- BR-18 — addressed — Revert-verified: dropping the InFlight.Operation clause reds TestTheInFlightExemptionDoesNotCoverAnotherActionsConfirmation.
- BR-19 — addressed — pastParticiple has a fallback and a direct test covering an operation it was not written for; "archiveed" is a cosmetic wart, not a silent empty word.
- BR-20 — addressed — consumeExpectedParkExitLocked now calls endsItsOwnChild, so the helper's doc comment is true; the wider four-site sweep stays owned by Task 10 Step 2b, and the newly covered relaunch arm there has no test (see coverage notes).
- BR-21 — addressed — New() no longer reads env, the path comes from the composition root, failure is surfaced, teardown closes; pinned by TestAConsoleOpensNoTraceUnlessAskedTo.

### Raised

- **BR-22** [Important] `declared-source-hand-maintained-consumers` bindingRefusalDiagnostic has one consumer while the real resolver and its stateful fake still hand-write the sentence it replaced
  6th in this family, so the deliverable is the RULE, not the site: a helper
  introduced to replace a literal is not adopted until no production file
  still contains that literal. resume.go:305 and artifactcollision_fake.go:124
  both still return refuseResume(code, "native session binding is not one
  exact established root"). ResumeContext:349-353 propagates that error before
  DecideResume/CheckResumePreconditions can re-derive the usable sentence, so
  Enter on a parked row with a provisional binding -- the recovery relaunch's
  own ParkedNotResumed message names -- and relaunch's own post-park resume
  failure both land on the developer sentence that 54052fab, lessons.md and
  atlas/couch.md:139-146 all claim was eliminated. Fix the class: one
  refuseBinding(code) called from all three sites, plus a conformance
  assertion that the real resolver and the fake produce the same code AND the
  same message per BindingStatus -- the fake currently conforms to the wrong
  wording, which is what will keep it wrong. ARCH-DRY, ARCH-PURPOSE, ARCH-MOCK.
- **BR-23** [Minor] `field-read-outside-its-mutex` pumpStdin reads c.trace without c.mu while SetInputTrace and teardown write it under the lock
  console.go:1127 loads c.trace unsynchronized; console.go:200-203 and
  723-725 store it under c.mu. teardown nils the field before c.Stop() and
  before the workers are joined, so pumpStdin can still be in flight. record
  is nil- and closed-safe, so the defect is the field access itself: latent
  under `make test-race` (go test -race ./cmd/...), not reproducible today.
  Read it under c.mu in pumpStdin, or hold the tracer in an atomic.Pointer.

## Round 9 — 2026-09-04T16:37:19-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Plan envelope untouched; verified 15s CompletionTimeout (couch.go:119) + 5s resumeRegistrationTimeout (couch.go:107) = ~20s, not ~30s.
- BR-6 — not-addressed — relaunch_test.go still has five cases; agent-unsupported and profile-missing remain pinned only at CheckResumePreconditions.
- BR-7 — not-addressed — plan Task 1 Step 5 still unticked; the stale-estimate warning still contradicts the re-derived Estimate. Subsumed by the rule in the new plan-record-lags-code finding.
- BR-9 — not-addressed — manifest.go:524 unchanged, and threadreason.go:503 is a second offender; one sort assertion over the manifest lists closes the family.
- BR-10 — not-addressed — Instance fixed and revert-verified (deleting the arm reds the chord test), but no enumeration from InterceptorHit to a handler exists, and README:141 plus menuControls still carry no Alt+n row.
- BR-22 — addressed — refuseBinding is now the only binding-refusal constructor and is called from resume.go:211, resume.go:316 and artifactcollision_fake.go:126; the fake's wording is pinned by test.
- BR-23 — addressed — pumpStdin now loads c.trace under c.mu (console.go:1195-1200); no test pins it, which is inherent for a latent race.

### Raised

- **BR-24** [Important] `plan-record-lags-code` The pair#186 split was written into one artifact and propagated to none of its three consumers
  2nd in this family, so the deliverable is the RULE: a scope change is not
  recorded until every artifact restating the scope derives from or cites the
  one entry that made it. Enumeration, all live: the plan doc has no
  "## Revisions" at all and its M2 Core-concepts table still declares
  paneState and RenderHoldingPane in couchtty/holding.go, none of which exist
  anywhere in the tree, with Tasks 8/10/11/12 carrying unticked steps though
  Task 8 and Step 2b shipped; workshop/projects/couch.md:201 still describes
  M2 as "the gesture and a surface that outlives its child" and lists no
  pair#186 row; the issue's "## Done when" still claims "the operator ends on
  the same actor, not in the switcher" while console.go:1383-1387 sets
  FocusPanel unconditionally and its own comment says why; and BR-7's stale
  estimate warning still stands. Also: "## Plan" marks M2 [x] with no
  Review-Verdict trailer and no "closed M2" log line in the range.
- **BR-25** [Important] `declared-source-hand-maintained-consumers` Both guards written to catch this class read a hand-maintained list, so the new chord and the new operation walked past them
  7th in this family. Do NOT fix these two sites -- the rule is: a guard whose
  source is a hand-maintained list is a guard the next addition can skip; it
  must derive from the table the production path consumes. (1)
  TestREADMEDocumentsEveryPanelControl (couchcmd/readme_test.go:61) exists so
  "adding a key to the UI makes this test fail until its documentation has a
  home in README" -- it iterates menuControls (menu.go:19). Alt+n and
  Ctrl+Alt+n went into knownSequences (keys.go:170-171) and never into
  menuControls, so the guard could not fire and README.md:141 still tells the
  operator Alt+n reloads pair in place in "any pane", which is false inside
  couch. (2) TestEveryOfferedActionIsReachableFromEnter tests the CONVERSE of
  plan Task 10 Step 2b: it walks menuActionItems asserting each offered action
  is reachable, where the plan asked it to walk Operations() asserting each
  switcher-reachable PresentationTUI operation is OFFERED. Offered-implies-
  reachable cannot catch declared-but-unreachable, the exact failure it was
  written for, and the issue's Plan line claims otherwise. Cheap fix: a
  row-action field on Operation that menuActionItems derives membership from,
  which also collapses several of the seven restatements still standing.
- **BR-26** [Minor] `done-when-untested` onRelaunchHotkey's panel branch is production code with no test, and the Done-when bullet it serves is unpinned
  2nd in this family, so the rule rather than the site: the enumeration IS the
  Done-when list -- every bullet the issue still claims must cite the test
  that pins it or the Revisions line that moved it. console.go:1378-1380
  (panel focus takes the highlighted row) is never exercised; only the
  actor-focus branch is driven by console_relaunch_chord_test.go. Done-when
  "Alt+n on the switcher relaunches the highlighted row and leaves the
  operator in the switcher" and plan Task 10 Step 3's "two tests, because the
  endings differ" are both unmet.
- **BR-27** [Minor] `review-window-degenerate` The gate handed this round the whole 111-commit branch, including three other issues' already-closed work
  2nd in this family (BR-11 was the empty variant of the same rule). Base
  88fe1de is the branch point, so the window is 164 files and +24177/-4669
  spanning pair#170, pair#181 and pair#185, each of which already passed its
  own close gate with its own review doc in this very diff. The window should
  be the un-reviewed span: M1's close at b7ec5e64, or at most the first #182
  commit. I reviewed b18f958e..HEAD for the code, which is #182 plus the
  #183/#184/#185/#186 issue-sync commits.

## Round 10 — 2026-09-04T16:56:08-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — plan lines 32-35 unchanged; measured CompletionTimeout=15s (couch.go:119) and resumeRegistrationTimeout=5s (couch.go:107), child death awaited inside the 15s (park.go:552).
- BR-6 — not-addressed — relaunch_test.go:76 still has five cases; no agent-unsupported or profile-missing case at the Relaunch level.
- BR-7 — not-addressed — the stale-estimate half is resolved in the issue's Revisions; plan lines 159 and 352 are still unticked for work that landed (e7c6c6e8, b7ec5e64) and the new Revisions note does not name them.
- BR-9 — not-addressed — artifactpath/manifest.go:524 still places relaunch.go between pathops.go and procops.go.
- BR-10 — addressed — hitHandlers table plus a knownSequences-driven test; the load-bearing direction is derived from the production table. Residual raised as new (intercepts() is still hand-maintained).
- BR-24 — addressed — plan gained a Revisions section naming the moved tasks and the entities declared-but-absent; project row and the FocusPanel Done-when corrected; M2's Mx tag dropped. Remaining unswept consumers raised as new.
- BR-25 — addressed — menuControls and README both fixed; RowAction gives membership one source and the declared-to-offered direction is mutation-verified red. The converse direction's unfalsifiability raised as new.
- BR-26 — not-addressed — console.go:1372-1373 (panel branch takes the highlighted row) still has no test; console_relaunch_chord_test.go drives only the actor branch.
- BR-27 — withdrawn — sdlc documents branch-point..HEAD as the whole-issue integration window by design (ariadne close.go:975-987), so the base is not a gate defect; the branch carrying four issues is a hygiene fact, not a window bug.

### Raised

- **BR-28** [Important] `declared-source-hand-maintained-consumers` intercepts() and hit() are still two hand-maintained switches over one enum, and the new guard skips whatever intercepts() forgets
  8th in this family, so the deliverable is the enumeration, not the site. keys.go:50-76 keeps two switches over seqKind where hit() alone is the
  mapping, and console_relaunch_chord_test.go:334 gates its sweep on intercepts() -- the list that can be wrong. Verified in a scratch worktree:
  removing seqRelaunch from intercepts() leaves TestEveryInterceptedChordHasAHandler green. keys.go:270-276 records that this exact omission has
  already shipped once. Fix: intercepts() returns k.hit() != HitNone, so one list remains and the test's skip clause derives from it. ARCH-DRY, ARCH-PURPOSE.
- **BR-29** [Important] `guard-unfalsifiable-by-construction` menuActionItems filters through declaredRowActions, so the sweep's offered-implies-declared direction can never fail and its failure is now a silent UI drop
  menu.go:1085-1097 filters the row's offer through couchcore.RowActions() before menu_action_sweep_test.go:96-100 reads it, so offered is a subset of
  declared by construction. Verified: adding "bogus" to the live row in menuActionItemsForState produces zero test failures and the item simply
  disappears from the switcher. The rule: a guard must be able to fail, and production must not coerce its input into agreement. Fix: build the test's
  offered set from menuActionItemsForState, pre-filter.
- **BR-30** [Important] `plan-record-lags-code` the pair#186 split is still unpropagated into four artifacts, including the two the fix edited
  3rd in this family; the rule is already in lessons.md, so the deliverable is the sweep. Live: the issue's Done-when bullet 2 still claims the
  rebuilt-binary verification that Revisions moved to pair#186 (the smoke test proved conversation survival, not a rebuilt binary), and the
  ctrl+backspace bullet is likewise unmarked with no test; the plan's M2 Integration-points table still lists onExit/finishOperation as holding-pane
  modifications and describes onRelaunchHotkey as returning to the actor, contradicting the code and the corrected Done-when; projects/couch.md:201
  still labels the row [pair#182 M2] with no detail block though the Mx tag was dropped. ARCH-PURPOSE on bullet 2 -- the rebuilt binary is the point of the issue.
- **BR-31** [Important] `new-surface-undocumented` the Alt+n interception reached neither atlas/couch.md's chord section nor README's couch section, and the README guard could not notice
  2nd in this family. atlas/couch.md:436 has "Alt+d is Couch's own detach (pair#170), intercepted like Alt+x" with no Alt+n counterpart, and line 355
  still frames the set as "two lifecycle chords"; README.md:382-389 does the same, documenting Alt+n only inside Pair's own keybinding table, and
  neither records the switcher-row form or the deliberate FocusPanel ending. TestREADMEDocumentsEveryPanelControl substring-matches control.Keys
  against the whole README, so the pre-existing Pair-table "Alt+n" satisfied it. Fix: an atlas paragraph beside Alt+d, a couch-section README sentence,
  and scope the README guard to the couch section.
- **BR-32** [Minor] `operating-envelope` menuActionItems rebuilds the whole Operations() table on every action-menu render
  menu.go:1096 calls couchcore.RowActions(), which constructs ~15 Operation structs with ArgSpec slices, once per render at menu_render.go:200 -- a
  keystroke-path render. Negligible in absolute terms; a package-level memoized set costs nothing.
- **BR-33** [Minor] `declared-source-hand-maintained-consumers` processInput still consumes the chord bytes when a hit has no handler
  console.go:636-638 drops the hit silently on the nil branch, so a gap is invisible at runtime even though the test now catches it at build time. A
  status notice there would make the missing case observable to an operator.

## Round 11 — 2026-09-04T17:39:21-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — plan lines 32-37 untouched; measured 15s CompletionTimeout (couch.go:119) + 5s resumeRegistrationTimeout (couch.go:107), child death inside the 15s (park.go:552).
- BR-6 — not-addressed — relaunch_test.go:76 still has five refusal cases; no agent-unsupported or profile-missing case reaches Relaunch.
- BR-7 — not-addressed — plan lines 159 and 352 still unticked for landed work (e7c6c6e8, b7ec5e64); the new Revisions note names only Task 8's boxes.
- BR-9 — not-addressed — manifest.go:524 still places relaunch.go between pathops.go and procops.go; threadreason.go:503 is still the second offender.
- BR-26 — not-addressed — console.go:1379 still has no test, and the untested branch turns out to be wrong — raised as a new Important finding.
- BR-28 — addressed — intercepts() is now k.hit() != HitNone (keys.go:72); revert-verified — deleting the seqRelaunch arm of hit() reds five chord tests.
- BR-29 — addressed — declaredRowActions and RowActions() deleted; the sweep reads Operation.RowAction directly. Mutation-verified — adding "bogus" to the live row reds both directions.
- BR-30 — addressed — Done-when bullets 2/ctrl+backspace/child-exited, the plan's integration-points table and projects/couch.md:201 all now carry the pair#186 split.
- BR-31 — addressed — atlas/couch.md:438-455 and README.md:387-397 both document the chord, the scope deviation and the FocusPanel ending; readme_test.go:58-77 scopes the guard and fatals on a missing marker.
- BR-32 — addressed — the per-render Operations() rebuild is gone with declaredRowActions; menu.go no longer calls couchcore.RowActions() at all.
- BR-33 — addressed — console.go:639-644 sets a status notice on the nil-handler branch, so a gap is observable at runtime and not only at build time.

### Raised

- **BR-34** [Important] `done-when-untested` onRelaunchHotkey reads the selection off CurrentFrame(), so Alt+n from the switcher refuses whenever the operator has drilled into a frame
  console.go:1379 takes c.menu.CurrentFrame().SelectedAddress, but SelectedAddress is
  populated only on the ROOT frame — the actions frame is built with Thread set and
  SelectedAddress zero (menu.go:457-459), as are confirmation and text frames. So from
  any non-root frame the target is the zero address and the operator gets
  "relaunch: no thread selected" on the status row while the row is highlighted and its
  action list is open. Verified in a scratch worktree: from the root frame the panel
  branch opens the relaunch confirmation; after one Tab the same call changes nothing.
  Reachable on an ordinary path — Alt+n pressed a second time on the confirmation it
  just opened hits it too. 3rd in this family, so the deliverable is the rule, not the
  line: every Done-when bullet still claimed must cite the test that pins it or the
  Revisions line that moved it. Eight bullets are still claimed here; seven cite a test,
  and the one that does not is the one that is broken. Fix: read the root selection
  (c.menu.Frames[0].SelectedAddress, length-guarded), which is what reduceParkHotkey
  already collapses to. ARCH-PURPOSE.
- **BR-35** [Important] `declared-source-hand-maintained-consumers` this plan's Core concepts table is enforced by no contract test, and two of its rows name symbols that exist nowhere
  TestCoreConceptsContract exists to turn a drifting architecture table into an
  executable contract, but it reads conceptPlans (core_concepts_contract_test.go:206-214),
  a literal list of two filenames; couchcore/plan_contract_test.go pins 000149 and 000151
  the same way. 000182-relaunch-an-actor-plan.md is in neither, so its table is prose.
  Live: the M2 table declares paneState (console.go, "new") and RenderHoldingPane
  (cmd/internal/couchtty/holding.go, "new"); grep -rn 'paneState|RenderHoldingPane' cmd/
  returns nothing and holding.go does not exist. The Revisions entry disclaims them in
  prose, which is why this is Important and not Critical. 10th in this family, so the
  rule: a guard whose input is a hand-maintained list is a guard the next addition can
  skip. The enumeration is "every plan under workshop/plans/ carrying a Core concepts
  table", discovered by scanning the directory, with the existing planned-status skip
  carrying rows whose work has moved to pair#186. ARCH-DRY.

## Open findings

- **BR-1** [Minor] `operating-envelope` Two of the envelope's three durations name budgets that do not exist as constants
- **BR-6** [Minor] `done-when-untested` No Relaunch-level test for the agent-unsupported and profile-missing refusals
- **BR-7** [Minor] `plan-record-lags-code` Task 1 Step 5 is unticked though committed, and the issue's stale-estimate warning contradicts the re-derived Estimate block
- **BR-9** [Minor] `manifest-ordering` relaunch.go inserted out of alphabetical order in NonArtifactSources
- **BR-26** [Minor] `done-when-untested` onRelaunchHotkey's panel branch is production code with no test, and the Done-when bullet it serves is unpinned
- **BR-34** [Important] `done-when-untested` onRelaunchHotkey reads the selection off CurrentFrame(), so Alt+n from the switcher refuses whenever the operator has drilled into a frame
- **BR-35** [Important] `declared-source-hand-maintained-consumers` this plan's Core concepts table is enforced by no contract test, and two of its rows name symbols that exist nowhere
