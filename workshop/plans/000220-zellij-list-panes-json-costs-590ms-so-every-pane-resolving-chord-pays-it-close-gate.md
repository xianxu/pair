---
gate: boundary-review
issue: 220
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-09T15:07:48-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Toggle-focused's geometry justification holds for only one of its two entry points
          detail: |-
            layoutcmd.OSRuntime.ListPanesJSON passes --geometry (layoutcmd.go:255) but termcmd's does not
            (run.go:1624), and handleTerminalChord reaches RunToggleFocused with the termcmd runtime
            (run.go:537). tiledScreenSize (layoutcmd.go:229-243) reads X/Columns/Rows, so Alt+Shift+Enter from
            the terminal pane already computes screen size from zeroed geometry. Does not change the row's
            disposition; note it so the close review can decide whether it is a separate issue.
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: divergent-runtime-impls
          round: 1
        - id: BR-2
          severity: Minor
          title: Fallback predicate is stated for one substituted field, not for every field the pane list supplied
          detail: |-
            2nd in this family, so the deliverable is the rule, not the instance: a sidecar
            substitution falls back to the pane list when ANY replaced field is unavailable.
            The termcmd:144 row replaces focused pane, role and draft id but names a fallback
            only for empty CurrentPaneID. CachedDraftPaneID can miss (route.go:53-68); with no
            recorded LastLeftPaneID, Decide falls through to DraftPaneID (shortcut.go:264-267)
            and runDecision silently returns nil on an empty target (run.go:192-195), so Alt+k
            from the terminal becomes inert where the pane list answers today. RouteLua already
            implements the disjunctive predicate at route.go:75-87.
            (carried from plan-quality PQ-8, deferred to the boundary review)
          family: sidecar-fastpath-case-enumeration
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-09T15:07:48-07:00"
      agent: claude
      findings:
        - id: BR-3
          severity: Important
          title: 'Both termcmd fast paths execute in zero tests (coverage profile: count 0)'
          detail: |-
            run.go:165-169 and run.go:604-606 are never entered by any test; the coverage
            profile reports count 0 for both blocks. No fixture sets currentPaneID and a
            registry containing it. Add a fakeRuntime with failList:true asserting
            splitTerminalDown succeeds with listCalls == 0, and the same for
            focusedWorkbenchPanes once the seam bypass is fixed.
          family: fastpath-untested
          round: 2
        - id: BR-4
          severity: Important
          title: focusedWorkbenchPanes calls draftroute.CachedDraftPaneIDFromEnv instead of rt.CachedDraftPaneID
          detail: |-
            run.go:164 reads os.Getenv plus the real filesystem directly, though
            Runtime.CachedDraftPaneID exists (run.go:29) and OSRuntime implements it as
            exactly that call (run.go:1669). Production and test flow no longer share the
            boundary (ARCH-MOCK), the fast path becomes untestable with the fake, and which
            branch a test takes depends on ambient PAIR_DATA_DIR/PAIR_TAG/ZELLIJ_SESSION_NAME.
          family: seam-bypass
          round: 2
        - id: BR-5
          severity: Important
          title: Fast path synthesises role RightTerminal from a bare ZELLIJ_PANE_ID with no registry check
          detail: |-
            run.go:163 trusts any non-empty current pane id, unlike its sibling at run.go:601
            which checks registry membership. handleChord is also reachable via --test-shortcut,
            which tests/term-pane-shortcuts-test.sh drives from the draft's and the review pane's
            perspective inside a real zellij pane that exports ZELLIJ_PANE_ID; the Plan's "the
            --test-shortcut path has no live pane" is false there. Gate on
            currentID in rt.TerminalPaneIDs() (ARCH-DRY with run.go:601).
          family: unchecked-fastpath-premise
          round: 2
        - id: BR-6
          severity: Important
          title: procutil.Alive's two declared behavior changes are pinned by no test
          detail: |-
            procutil_test.go:11 asserts only empty/self/bogus-high, all of which pass against the
            old exec-based implementation. Measured: kill -0 0 exits 0 (old Alive("0") was true,
            new false) and kill -0 1 exits 1 (old Alive("1") was false, new true via EPERM).
            Assert Alive("0"), Alive("-1"), Alive("abc") false and Alive("1") true, else PQ-4's
            positivePID guard is protection with zero assertion sites.
          family: claimed-fix-unpinned
          round: 2
        - id: BR-7
          severity: Important
          title: Generated agreement space never exercises the single-live-id branch
          detail: |-
            layoutcmd_test.go:304 narrows subsets to empty/full, so all 54 answered cases come from
            the record-hit branch and the record axis collapses (with a full registry
            "unregistered-present" == "live-registered"). The len(liveIDs)==1 branch — the one that
            answers with no cross-check — is agreement-checked nowhere. The case "subset" and
            case "disjoint" arms at :319/:323 are dead code that reads as coverage.
          family: agreement-oracle-strength
          round: 2
        - id: BR-8
          severity: Minor
          title: resolveRightTerminal fabricates a zellijpane.Pane with only ID populated
          detail: |-
            layoutcmd.go:90 wraps the id in a Pane whose remaining fields are zero rather than
            unknown. Both callers read only .ID; drop the wrapper and call resolveRightTerminalID
            directly so no synthetic pane escapes for a later caller to read .IsFocused off.
          family: fabricated-value-escapes-type
          round: 2
        - id: BR-9
          severity: Minor
          title: Alive's doc comment misstates the old EPERM behavior
          detail: |-
            procutil.go:32 says the exit-status check "silently got right" the EPERM distinction.
            Measured: kill -0 on another user's pid exits 1, so the old code reported it dead.
            The change is a fix, not preservation.
          family: comment-vs-measured-behavior
          round: 2
        - id: BR-10
          severity: Minor
          title: Registry pane id is never validated as a pane id, only the pid is
          detail: |-
            shortcut.go:590 parses fields[1] as an int but takes fields[0] verbatim, and the fast
            path now hands it to focus-pane-id / write --pane-id without the pane-report
            intersection that previously constrained it to a real pane. PQ-4 named the registry as
            untrusted input; the fix took the pid instance and left this sibling (ARCH-SECURE).
          family: untrusted-sidecar-parse
          round: 2
        - id: BR-11
          severity: Minor
          title: Spec's unscoped agreement claim and the Plan's "same resolver" row no longer match the code
          detail: |-
            The Spec still states the fast path produces the same id as the slow path in every case
            it handles; the delivered guarantee is scoped to registry-consistent states. The Plan's
            caller table says currentRightTerminalPane uses the same resolver; it uses an inline
            registry-membership check, correctly, since its question is different. Record both in a
            "## Revisions" entry rather than leaving them only in the Log.
          family: plan-code-drift
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-09T15:24:19-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: 'Still true; re-measured the consequence (Alt+Shift+Enter inert from the terminal pane). Pre-existing, outside #220''s window — file as its own issue.'
          round: 3
        - id: BR-2
          disposition: addressed
          note: Fast path now gates on all three substituted fields; the "no cached draft" case pins the fallback. The general rule is implemented but not written down.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Coverage profile now reports count 1 on run.go:188 and run.go:625 (was 0), and the saving is asserted via listCalls.
          round: 3
        - id: BR-4
          disposition: addressed
          note: draftroute.CachedDraftPaneIDFromEnv now appears only at run.go:1688 (the OSRuntime impl); the fast path uses rt.CachedDraftPaneID().
          round: 3
        - id: BR-5
          disposition: addressed
          note: Verified by mutation — removing "&& registered(rt, currentID)" from both sites reddens two tests.
          round: 3
        - id: BR-6
          disposition: addressed
          note: Verified by mutation — dropping positivePID reddens Alive("0")/Alive("-1"); dropping the EPERM branch reddens Alive("1").
          round: 3
        - id: BR-7
          disposition: addressed
          note: Verified by mutation — a bogus return from the single-live-id branch reddens 18 generated cases. The branch is agreement-checked; its reachability GUARD is not (new finding).
          round: 3
        - id: BR-8
          disposition: not-addressed
          note: layoutcmd.go:95 unchanged, and this diff added three more synthesised panes (run.go:190, :191, :626) — the class is now 4 sites.
          round: 3
        - id: BR-9
          disposition: not-addressed
          note: 'Re-measured: /bin/kill -0 1 exits 1. procutil.go:37 still claims the exit-status check got it right, and procutil_test.go:129 now repeats the claim.'
          round: 3
        - id: BR-10
          disposition: not-addressed
          note: 'shortcut.go:588 unchanged. Sibling: the registry line carries no session field, unlike CachedPaneRecord, so a cross-session id can now reach focus-pane-id unintersected.'
          round: 3
        - id: BR-11
          disposition: not-addressed
          note: No "## Revisions" section exists in the issue file; the Spec's unscoped claim and the Plan's "same resolver" row are both unchanged.
          round: 3
      findings:
        - id: BR-12
          severity: Important
          title: The single-live-id reachability guard counts an input predicate, so it survives the branch's deletion
          detail: |-
            2nd in this family, so the deliverable is the RULE: a reachability/coverage guard
            in a generated test must be incremented at the site it names, not derived from an
            input predicate that correlates with it. layoutcmd_test.go:349 increments `single`
            on len(registry)==1, which the RECORD branch also satisfies. Measured: deleting
            the `len(liveIDs)==1` branch from resolveFromSidecars leaves the test PASSING and
            still logging "36 via the single-live-id branch". The reported 54 is wrong today
            too — only 18 tuples enter that branch, and since kindB is a dead axis when
            size==1, those 18 are 2 distinct worlds repeated 9 times. Prevalence in this diff:
            5 guards, 1 derived (checked/answered/listCalls x2 are all counted at their real
            sites). Fix the rule: return which branch answered and count on that, so the guard
            cannot drift from the code; and skip the kindB loop when size==1.
          family: agreement-oracle-strength
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-09T15:40:34-07:00"
      agent: claude
      dispose:
        - id: BR-12
          disposition: addressed
          note: 'Verified by mutation in a scratch worktree — deleting the single-live-id branch now fails the guard by name; residue: the kindB axis is still dead when size==1, so the reported 18 is 6 distinct worlds.'
          round: 4
        - id: BR-1
          disposition: not-addressed
          note: 'Re-confirmed: termcmd''s ListPanesJSON (run.go:1684) omits --geometry, layoutcmd''s (layoutcmd.go:305) has it, so Alt+Shift+Enter from the terminal is inert — pre-existing, close it by filing its own issue rather than by more work in 220.'
          round: 4
        - id: BR-8
          disposition: not-addressed
          note: layoutcmd.go:95 unchanged; the class is 4 sites now (run.go:190, :191, :626).
          round: 4
        - id: BR-9
          disposition: not-addressed
          note: 'Re-measured on this host: /bin/kill -0 1 exits 1, so the old code called an EPERM process dead; procutil.go:37 and procutil_test.go:129 both still claim otherwise.'
          round: 4
        - id: BR-10
          disposition: not-addressed
          note: shortcut.go:587-595 unchanged — fields[1] is parsed as an int, fields[0] is taken verbatim and now reaches focus-pane-id unintersected.
          round: 4
        - id: BR-11
          disposition: not-addressed
          note: No "## Revisions" section exists; and a third instance has accumulated — the Plan's termcmd:144 row still carries the "--test-shortcut path has no live pane" justification BR-5 falsified.
          round: 4
      findings:
        - id: BR-13
          severity: Important
          title: Two of the three fast paths pin their saving and their gate but never their answer
          detail: |-
            2nd in this family, so the deliverable is the RULE: a fast path substituting for an
            existing slow path is not tested by asserting listCalls==0 and that it declines when
            gated; it is tested when its ANSWER is differentially pinned against the slow path on
            a shared fixture. Measured prevalence in this diff: 3 fast paths, 1 pinned —
            layoutcmd's resolveFromSidecars has a 360-case agreement oracle, focusedWorkbenchPanes
            (run.go:188) and currentRightTerminalPane (run.go:625) have none. Concrete gap:
            run_test.go:1243 sets cachedDraft "2" while the fall-back fixture's draft is also id 2,
            so the assertion cannot distinguish "reads the cache" from "agrees with the report" —
            set cachedDraft to "7" and the fast path returns 7 with the suite green. DraftPaneID is
            live via shortcut.go:265-267 (right-terminal Alt+K with an empty LastLeftPaneID). Fix
            the rule: run each fixture twice, gate satisfied and gate unsatisfied, and assert the
            two results are equal instead of comparing to a literal.
          family: fastpath-untested
          round: 4
        - id: BR-14
          severity: Minor
          title: Registry membership is open-coded a third time instead of living with the registry
          detail: |-
            run.go:159 `registered` duplicates the same "is this pane id in the live registry" loop
            that resolveFromSidecars carries at layoutcmd.go:66 and RoleForPaneWith at
            shortcut.go:606. workbenchshortcut owns the registry; export
            Registered(ids []string, paneID string) bool there and have all three call it (ARCH-DRY).
          family: shared-predicate-not-extracted
          round: 4
      blocked: true
    - "n": 5
      timestamp: "2026-09-09T15:59:35-07:00"
      agent: claude
      dispose:
        - id: BR-13
          disposition: addressed
          note: 'Verified by mutation, not by the commit message: draftID+"9" and currentID+"9" each redden the new differential test.'
          round: 5
        - id: BR-14
          disposition: addressed
          note: workbenchshortcut.Registered has exactly 3 call sites; grep finds no fourth open-coded membership loop.
          round: 5
        - id: BR-11
          disposition: not-addressed
          note: Still no "## Revisions" section, and a fourth delta has accumulated - the Plan's "not new trust, only earlier trust" claim is falsified by the removed pane-report intersection.
          round: 5
        - id: BR-10
          disposition: not-addressed
          note: fields[0] still verbatim; sharpened - the registry carries no session field and no incarnation identity though procutil.Identity exists, so a recycled pid now yields an answer.
          round: 5
        - id: BR-9
          disposition: not-addressed
          note: Comment unchanged at procutil.go:35; I re-measured, kill -0 1 exits 1 here, so the old code reported another user's process dead.
          round: 5
        - id: BR-8
          disposition: not-addressed
          note: 'layoutcmd.go:90 still returns zellijpane.Pane{ID: id}; both callers read only .ID.'
          round: 5
        - id: BR-1
          disposition: not-addressed
          note: Confirmed unchanged and pre-existing (run.go:1679 omits --geometry, reached via run.go:256/583); recommend a separate issue rather than this close.
          round: 5
      findings:
        - id: BR-15
          severity: Minor
          title: Round 3's review rule reached the issue Log and a test comment but not workshop/lessons.md
          detail: |-
            AGENTS.md section 4 makes lessons.md the durable form of a rule found in review, and
            round 2's rule was recorded there (lessons.md:4225). BR-13's rule - a fast path
            substituting for an existing slow path is pinned by its ANSWER, not by asserting
            listCalls==0 and that it declines when gated - exists only in the issue Log and in a
            comment above run_test.go:1236. grep for "fast path" in lessons.md finds no entry. It
            is the more generalizable of the two rules and the one most likely to recur in a
            different file.
          family: review-rule-not-recorded
          round: 5
      blocked: false
---

# Gate ledger — pair#220 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-09T15:07:48-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `divergent-runtime-impls` Toggle-focused's geometry justification holds for only one of its two entry points
  layoutcmd.OSRuntime.ListPanesJSON passes --geometry (layoutcmd.go:255) but termcmd's does not
  (run.go:1624), and handleTerminalChord reaches RunToggleFocused with the termcmd runtime
  (run.go:537). tiledScreenSize (layoutcmd.go:229-243) reads X/Columns/Rows, so Alt+Shift+Enter from
  the terminal pane already computes screen size from zeroed geometry. Does not change the row's
  disposition; note it so the close review can decide whether it is a separate issue.
  (carried from plan-quality PQ-7, deferred to the boundary review)
- **BR-2** [Minor] `sidecar-fastpath-case-enumeration` Fallback predicate is stated for one substituted field, not for every field the pane list supplied
  2nd in this family, so the deliverable is the rule, not the instance: a sidecar
  substitution falls back to the pane list when ANY replaced field is unavailable.
  The termcmd:144 row replaces focused pane, role and draft id but names a fallback
  only for empty CurrentPaneID. CachedDraftPaneID can miss (route.go:53-68); with no
  recorded LastLeftPaneID, Decide falls through to DraftPaneID (shortcut.go:264-267)
  and runDecision silently returns nil on an empty target (run.go:192-195), so Alt+k
  from the terminal becomes inert where the pane list answers today. RouteLua already
  implements the disjunctive predicate at route.go:75-87.
  (carried from plan-quality PQ-8, deferred to the boundary review)

## Round 2 — 2026-09-09T15:07:48-07:00 (claude) — BLOCKED

### Raised

- **BR-3** [Important] `fastpath-untested` Both termcmd fast paths execute in zero tests (coverage profile: count 0)
  run.go:165-169 and run.go:604-606 are never entered by any test; the coverage
  profile reports count 0 for both blocks. No fixture sets currentPaneID and a
  registry containing it. Add a fakeRuntime with failList:true asserting
  splitTerminalDown succeeds with listCalls == 0, and the same for
  focusedWorkbenchPanes once the seam bypass is fixed.
- **BR-4** [Important] `seam-bypass` focusedWorkbenchPanes calls draftroute.CachedDraftPaneIDFromEnv instead of rt.CachedDraftPaneID
  run.go:164 reads os.Getenv plus the real filesystem directly, though
  Runtime.CachedDraftPaneID exists (run.go:29) and OSRuntime implements it as
  exactly that call (run.go:1669). Production and test flow no longer share the
  boundary (ARCH-MOCK), the fast path becomes untestable with the fake, and which
  branch a test takes depends on ambient PAIR_DATA_DIR/PAIR_TAG/ZELLIJ_SESSION_NAME.
- **BR-5** [Important] `unchecked-fastpath-premise` Fast path synthesises role RightTerminal from a bare ZELLIJ_PANE_ID with no registry check
  run.go:163 trusts any non-empty current pane id, unlike its sibling at run.go:601
  which checks registry membership. handleChord is also reachable via --test-shortcut,
  which tests/term-pane-shortcuts-test.sh drives from the draft's and the review pane's
  perspective inside a real zellij pane that exports ZELLIJ_PANE_ID; the Plan's "the
  --test-shortcut path has no live pane" is false there. Gate on
  currentID in rt.TerminalPaneIDs() (ARCH-DRY with run.go:601).
- **BR-6** [Important] `claimed-fix-unpinned` procutil.Alive's two declared behavior changes are pinned by no test
  procutil_test.go:11 asserts only empty/self/bogus-high, all of which pass against the
  old exec-based implementation. Measured: kill -0 0 exits 0 (old Alive("0") was true,
  new false) and kill -0 1 exits 1 (old Alive("1") was false, new true via EPERM).
  Assert Alive("0"), Alive("-1"), Alive("abc") false and Alive("1") true, else PQ-4's
  positivePID guard is protection with zero assertion sites.
- **BR-7** [Important] `agreement-oracle-strength` Generated agreement space never exercises the single-live-id branch
  layoutcmd_test.go:304 narrows subsets to empty/full, so all 54 answered cases come from
  the record-hit branch and the record axis collapses (with a full registry
  "unregistered-present" == "live-registered"). The len(liveIDs)==1 branch — the one that
  answers with no cross-check — is agreement-checked nowhere. The case "subset" and
  case "disjoint" arms at :319/:323 are dead code that reads as coverage.
- **BR-8** [Minor] `fabricated-value-escapes-type` resolveRightTerminal fabricates a zellijpane.Pane with only ID populated
  layoutcmd.go:90 wraps the id in a Pane whose remaining fields are zero rather than
  unknown. Both callers read only .ID; drop the wrapper and call resolveRightTerminalID
  directly so no synthetic pane escapes for a later caller to read .IsFocused off.
- **BR-9** [Minor] `comment-vs-measured-behavior` Alive's doc comment misstates the old EPERM behavior
  procutil.go:32 says the exit-status check "silently got right" the EPERM distinction.
  Measured: kill -0 on another user's pid exits 1, so the old code reported it dead.
  The change is a fix, not preservation.
- **BR-10** [Minor] `untrusted-sidecar-parse` Registry pane id is never validated as a pane id, only the pid is
  shortcut.go:590 parses fields[1] as an int but takes fields[0] verbatim, and the fast
  path now hands it to focus-pane-id / write --pane-id without the pane-report
  intersection that previously constrained it to a real pane. PQ-4 named the registry as
  untrusted input; the fix took the pid instance and left this sibling (ARCH-SECURE).
- **BR-11** [Minor] `plan-code-drift` Spec's unscoped agreement claim and the Plan's "same resolver" row no longer match the code
  The Spec still states the fast path produces the same id as the slow path in every case
  it handles; the delivered guarantee is scoped to registry-consistent states. The Plan's
  caller table says currentRightTerminalPane uses the same resolver; it uses an inline
  registry-membership check, correctly, since its question is different. Record both in a
  "## Revisions" entry rather than leaving them only in the Log.

## Round 3 — 2026-09-09T15:24:19-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Still true; re-measured the consequence (Alt+Shift+Enter inert from the terminal pane). Pre-existing, outside #220's window — file as its own issue.
- BR-2 — addressed — Fast path now gates on all three substituted fields; the "no cached draft" case pins the fallback. The general rule is implemented but not written down.
- BR-3 — addressed — Coverage profile now reports count 1 on run.go:188 and run.go:625 (was 0), and the saving is asserted via listCalls.
- BR-4 — addressed — draftroute.CachedDraftPaneIDFromEnv now appears only at run.go:1688 (the OSRuntime impl); the fast path uses rt.CachedDraftPaneID().
- BR-5 — addressed — Verified by mutation — removing "&& registered(rt, currentID)" from both sites reddens two tests.
- BR-6 — addressed — Verified by mutation — dropping positivePID reddens Alive("0")/Alive("-1"); dropping the EPERM branch reddens Alive("1").
- BR-7 — addressed — Verified by mutation — a bogus return from the single-live-id branch reddens 18 generated cases. The branch is agreement-checked; its reachability GUARD is not (new finding).
- BR-8 — not-addressed — layoutcmd.go:95 unchanged, and this diff added three more synthesised panes (run.go:190, :191, :626) — the class is now 4 sites.
- BR-9 — not-addressed — Re-measured: /bin/kill -0 1 exits 1. procutil.go:37 still claims the exit-status check got it right, and procutil_test.go:129 now repeats the claim.
- BR-10 — not-addressed — shortcut.go:588 unchanged. Sibling: the registry line carries no session field, unlike CachedPaneRecord, so a cross-session id can now reach focus-pane-id unintersected.
- BR-11 — not-addressed — No "## Revisions" section exists in the issue file; the Spec's unscoped claim and the Plan's "same resolver" row are both unchanged.

### Raised

- **BR-12** [Important] `agreement-oracle-strength` The single-live-id reachability guard counts an input predicate, so it survives the branch's deletion
  2nd in this family, so the deliverable is the RULE: a reachability/coverage guard
  in a generated test must be incremented at the site it names, not derived from an
  input predicate that correlates with it. layoutcmd_test.go:349 increments `single`
  on len(registry)==1, which the RECORD branch also satisfies. Measured: deleting
  the `len(liveIDs)==1` branch from resolveFromSidecars leaves the test PASSING and
  still logging "36 via the single-live-id branch". The reported 54 is wrong today
  too — only 18 tuples enter that branch, and since kindB is a dead axis when
  size==1, those 18 are 2 distinct worlds repeated 9 times. Prevalence in this diff:
  5 guards, 1 derived (checked/answered/listCalls x2 are all counted at their real
  sites). Fix the rule: return which branch answered and count on that, so the guard
  cannot drift from the code; and skip the kindB loop when size==1.

## Round 4 — 2026-09-09T15:40:34-07:00 (claude) — BLOCKED

### Disposed

- BR-12 — addressed — Verified by mutation in a scratch worktree — deleting the single-live-id branch now fails the guard by name; residue: the kindB axis is still dead when size==1, so the reported 18 is 6 distinct worlds.
- BR-1 — not-addressed — Re-confirmed: termcmd's ListPanesJSON (run.go:1684) omits --geometry, layoutcmd's (layoutcmd.go:305) has it, so Alt+Shift+Enter from the terminal is inert — pre-existing, close it by filing its own issue rather than by more work in 220.
- BR-8 — not-addressed — layoutcmd.go:95 unchanged; the class is 4 sites now (run.go:190, :191, :626).
- BR-9 — not-addressed — Re-measured on this host: /bin/kill -0 1 exits 1, so the old code called an EPERM process dead; procutil.go:37 and procutil_test.go:129 both still claim otherwise.
- BR-10 — not-addressed — shortcut.go:587-595 unchanged — fields[1] is parsed as an int, fields[0] is taken verbatim and now reaches focus-pane-id unintersected.
- BR-11 — not-addressed — No "## Revisions" section exists; and a third instance has accumulated — the Plan's termcmd:144 row still carries the "--test-shortcut path has no live pane" justification BR-5 falsified.

### Raised

- **BR-13** [Important] `fastpath-untested` Two of the three fast paths pin their saving and their gate but never their answer
  2nd in this family, so the deliverable is the RULE: a fast path substituting for an
  existing slow path is not tested by asserting listCalls==0 and that it declines when
  gated; it is tested when its ANSWER is differentially pinned against the slow path on
  a shared fixture. Measured prevalence in this diff: 3 fast paths, 1 pinned —
  layoutcmd's resolveFromSidecars has a 360-case agreement oracle, focusedWorkbenchPanes
  (run.go:188) and currentRightTerminalPane (run.go:625) have none. Concrete gap:
  run_test.go:1243 sets cachedDraft "2" while the fall-back fixture's draft is also id 2,
  so the assertion cannot distinguish "reads the cache" from "agrees with the report" —
  set cachedDraft to "7" and the fast path returns 7 with the suite green. DraftPaneID is
  live via shortcut.go:265-267 (right-terminal Alt+K with an empty LastLeftPaneID). Fix
  the rule: run each fixture twice, gate satisfied and gate unsatisfied, and assert the
  two results are equal instead of comparing to a literal.
- **BR-14** [Minor] `shared-predicate-not-extracted` Registry membership is open-coded a third time instead of living with the registry
  run.go:159 `registered` duplicates the same "is this pane id in the live registry" loop
  that resolveFromSidecars carries at layoutcmd.go:66 and RoleForPaneWith at
  shortcut.go:606. workbenchshortcut owns the registry; export
  Registered(ids []string, paneID string) bool there and have all three call it (ARCH-DRY).

## Round 5 — 2026-09-09T15:59:35-07:00 (claude) — passed

### Disposed

- BR-13 — addressed — Verified by mutation, not by the commit message: draftID+"9" and currentID+"9" each redden the new differential test.
- BR-14 — addressed — workbenchshortcut.Registered has exactly 3 call sites; grep finds no fourth open-coded membership loop.
- BR-11 — not-addressed — Still no "## Revisions" section, and a fourth delta has accumulated - the Plan's "not new trust, only earlier trust" claim is falsified by the removed pane-report intersection.
- BR-10 — not-addressed — fields[0] still verbatim; sharpened - the registry carries no session field and no incarnation identity though procutil.Identity exists, so a recycled pid now yields an answer.
- BR-9 — not-addressed — Comment unchanged at procutil.go:35; I re-measured, kill -0 1 exits 1 here, so the old code reported another user's process dead.
- BR-8 — not-addressed — layoutcmd.go:90 still returns zellijpane.Pane{ID: id}; both callers read only .ID.
- BR-1 — not-addressed — Confirmed unchanged and pre-existing (run.go:1679 omits --geometry, reached via run.go:256/583); recommend a separate issue rather than this close.

### Raised

- **BR-15** [Minor] `review-rule-not-recorded` Round 3's review rule reached the issue Log and a test comment but not workshop/lessons.md
  AGENTS.md section 4 makes lessons.md the durable form of a rule found in review, and
  round 2's rule was recorded there (lessons.md:4225). BR-13's rule - a fast path
  substituting for an existing slow path is pinned by its ANSWER, not by asserting
  listCalls==0 and that it declines when gated - exists only in the issue Log and in a
  comment above run_test.go:1236. grep for "fast path" in lessons.md finds no entry. It
  is the more generalizable of the two rules and the one most likely to recur in a
  different file.

## Open findings

- **BR-1** [Minor] `divergent-runtime-impls` Toggle-focused's geometry justification holds for only one of its two entry points
- **BR-8** [Minor] `fabricated-value-escapes-type` resolveRightTerminal fabricates a zellijpane.Pane with only ID populated
- **BR-9** [Minor] `comment-vs-measured-behavior` Alive's doc comment misstates the old EPERM behavior
- **BR-10** [Minor] `untrusted-sidecar-parse` Registry pane id is never validated as a pane id, only the pid is
- **BR-11** [Minor] `plan-code-drift` Spec's unscoped agreement claim and the Plan's "same resolver" row no longer match the code
- **BR-15** [Minor] `review-rule-not-recorded` Round 3's review rule reached the issue Log and a test comment but not workshop/lessons.md
