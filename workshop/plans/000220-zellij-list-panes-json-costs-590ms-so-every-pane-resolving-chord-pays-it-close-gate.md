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

## Open findings

- **BR-1** [Minor] `divergent-runtime-impls` Toggle-focused's geometry justification holds for only one of its two entry points
- **BR-2** [Minor] `sidecar-fastpath-case-enumeration` Fallback predicate is stated for one substituted field, not for every field the pane list supplied
- **BR-3** [Important] `fastpath-untested` Both termcmd fast paths execute in zero tests (coverage profile: count 0)
- **BR-4** [Important] `seam-bypass` focusedWorkbenchPanes calls draftroute.CachedDraftPaneIDFromEnv instead of rt.CachedDraftPaneID
- **BR-5** [Important] `unchecked-fastpath-premise` Fast path synthesises role RightTerminal from a bare ZELLIJ_PANE_ID with no registry check
- **BR-6** [Important] `claimed-fix-unpinned` procutil.Alive's two declared behavior changes are pinned by no test
- **BR-7** [Important] `agreement-oracle-strength` Generated agreement space never exercises the single-live-id branch
- **BR-8** [Minor] `fabricated-value-escapes-type` resolveRightTerminal fabricates a zellijpane.Pane with only ID populated
- **BR-9** [Minor] `comment-vs-measured-behavior` Alive's doc comment misstates the old EPERM behavior
- **BR-10** [Minor] `untrusted-sidecar-parse` Registry pane id is never validated as a pane id, only the pid is
- **BR-11** [Minor] `plan-code-drift` Spec's unscoped agreement claim and the Plan's "same resolver" row no longer match the code
