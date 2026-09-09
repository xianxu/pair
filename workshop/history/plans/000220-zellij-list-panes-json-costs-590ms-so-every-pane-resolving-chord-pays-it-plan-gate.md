---
gate: plan-quality
issue: 220
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-09T13:45:02-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Fast path treats the registry as the set of right terminals; it is a subset, and 0-id / unregistered cases are unspecified
          detail: |-
            pickRightTerminal classifies via RoleForPaneWith (layoutcmd.go:245, shortcut.go:210-226), which admits panes by
            terminal_command or title with an EMPTY registry — layoutcmd_test.go:81 and :126 both resolve with terminalPaneIDs nil.
            A pair term can be live and unregistered (run.go:1656-1659 no-ops without ZELLIJ_PANE_ID; run.go:242-244 warns and
            continues on error). Registry {4}, no record, report has focused pane 3 with terminal_command "sh -c exec pair term"
            plus pane 4: slow path returns 3, the plan's single-live-id rule returns 4 — Alt+k and Alt+Shift+arrow then mean
            different split halves, the #216 BR-10 property. Zero live ids and registry-read-error are named by neither branch.
            Put the (live ids x recorded preference) -> (answer | fall through) table in the Plan, and say what happens when the
            registry names a live pid whose pane is gone, since the pane report is today the cross-check that drops it.
          family: sidecar-fastpath-case-enumeration
          round: 1
        - id: PQ-2
          severity: Important
          title: Agreement test names no function and no adversarial input class
          detail: |-
            "Fast-path resolution in layoutcmd" names nothing unit-testable. Name the pure decision function, taking no Runtime
            (ARCH-PURE) e.g. resolveFromSidecars(liveIDs []string, lastTerminal string) (string, bool), and give one strategy
            line: generated cross-product over pane sets (command-classified, title-classified, registry-only) x registry
            subsets (empty, subset, disjoint) x record (none, live, stale, unregistered-but-present), asserting fast answers
            implies fast equals pickRightTerminal. The adversarial class is the registry disagreeing with the pane report;
            hand-picked cases are blind to it by construction.
          family: agreement-oracle-strength
          round: 1
        - id: PQ-3
          severity: Important
          title: Two of at least five interactive list-panes callers are fixed; Alt+k from the terminal still pays 590ms
          detail: |-
            handleTerminalChord (run.go:514-541) does not handle Alt+k, so it falls through to handleChord (run.go:451-453) ->
            focusedWorkbenchPanes -> ListPanesJSON (run.go:116, :143-144). splitTerminalDown -> currentRightTerminalPane
            (run.go:545, :567-568) and Alt+Shift+Enter -> RunToggleFocused (run.go:537, layoutcmd.go:189) pay it too. After this
            plan lands, Alt+k right-to-left is still ~600ms while left-to-right is fast. Enumerate every interactive
            ListPanesJSON caller and mark each fixed-now or deliberately-not with the reason; toggle-focused needs --geometry
            (tiledScreenSize, layoutcmd.go:229-243) and belongs in the Not-in-scope list, which currently omits it.
          family: latency-class-not-instance
          round: 1
        - id: PQ-4
          severity: Important
          title: syscall.Kill(pid, 0) needs the pid>0 guard; positivePID already exists and is unused by Alive
          detail: |-
            LiveIDs passes strconv.Atoi output straight to alive (shortcut.go:585-591), so a registry line "4 0" or "4 -1"
            reaches syscall.Kill(0,0) / Kill(-1,0) — process-group and broadcast probes that return nil, reporting the id alive
            forever and pinning a dead pane the fast path will hand to focus-pane-id. The registry is an append-only,
            concurrently written, hand-editable file: untrusted input (ARCH-SECURE). Reuse positivePID (procutil.go:14, already
            used by identity_darwin.go:14) rather than a new check (ARCH-DRY), and name how the not-ours row gets its pid
            (pid 1 -> EPERM).
          family: untrusted-sidecar-parse
          round: 1
        - id: PQ-5
          severity: Minor
          title: draftroute already implements cache-then-list-panes-on-miss; name it as the precedent
          detail: |-
            RouteLua consults CachedDraftPaneID and only calls ListPanesJSON on a miss (route.go:60-70, :79-87), with
            ValidateCachedDraftPane gating the sidecar on session plus procutil.Alive. Say whether the new fast path reuses that
            validation shape or deliberately differs (ARCH-DRY).
          family: existing-pattern-reuse
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-09T13:49:56-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Registry-as-subset stated, four-branch case table given, race failure mode named and accepted in writing.
          round: 2
        - id: PQ-2
          disposition: not-addressed
          note: Still no named function; nine prose cases substituted for the generation strategy, and bullet 2 points at RouteLua's IO shape rather than ValidateCachedDraftPane's pure one.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: All five ListPanesJSON call sites enumerated with dispositions; verified against the tree.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: positivePID reuse named with the kill(0)/kill(-1) rationale; procutil.go:14 confirmed unused by Alive.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: draftroute named as the precedent with file:line.
          round: 2
      findings:
        - id: PQ-6
          severity: Important
          title: The termcmd:144 row claims focusedWorkbenchPanes needs IDs only; run.go:129 consumes the whole focused pane for role classification
          detail: |-
            The row cites :131 and :133 and skips :129, where panes.focused goes into RoleForPaneWith
            (shortcut.go:603-614), which classifies on TerminalCommand/Title, not ID. Decide branches on that
            Role (shortcut.go:253) and Alt+k from the terminal lives only in the PaneRoleRightTerminal arm
            (shortcut.go:264-278); everything else hits DispositionPass at :309. Building workbenchPanes from
            rt.CurrentPaneID() alone therefore silently disables Alt+k AND its RecordLastTerminalPaneID write,
            starving the last-terminal sidecar this issue's own fast path reads. Worse where it matters:
            CurrentPaneID is os.Getenv("ZELLIJ_PANE_ID") (run.go:1631-1633), empty in exactly the case where
            RegisterTerminalPane no-ops (run.go:1656-1660), and the is_focused scan at run.go:157-160 is the
            fallback that disappears. State how Role is derived without the list -- RoleForPaneWith on a bare
            Pane{ID} plus the registry overlay -- and what happens when the pane is unregistered.
          family: unverified-existing-behavior-claim
          round: 2
        - id: PQ-7
          severity: Minor
          title: Toggle-focused's geometry justification holds for only one of its two entry points
          detail: |-
            layoutcmd.OSRuntime.ListPanesJSON passes --geometry (layoutcmd.go:255) but termcmd's does not
            (run.go:1624), and handleTerminalChord reaches RunToggleFocused with the termcmd runtime
            (run.go:537). tiledScreenSize (layoutcmd.go:229-243) reads X/Columns/Rows, so Alt+Shift+Enter from
            the terminal pane already computes screen size from zeroed geometry. Does not change the row's
            disposition; note it so the close review can decide whether it is a separate issue.
          family: divergent-runtime-impls
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-09T13:53:16-07:00"
      agent: claude
      dispose:
        - id: PQ-2
          disposition: addressed
          note: Names resolveFromSidecars with the exact signature, no Runtime, plus the generated cross-product and the equals-pickRightTerminal property.
          round: 3
        - id: PQ-6
          disposition: addressed
          note: Answered by role-known-by-construction rather than the overlay I proposed; run.go:152-155 backs the claim and run.go:180-184 keeps the sidecar write.
          round: 3
        - id: PQ-7
          disposition: not-addressed
          note: Re-verified (run.go:1624 vs layoutcmd.go:256, reached via run.go:537); Minor, carried to close review.
          round: 3
      findings:
        - id: PQ-8
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
          family: sidecar-fastpath-case-enumeration
          round: 3
      blocked: false
content_hash: 9bab8dd4933f7787c22692aa0d157d94a335dd332522a8f1eff98423d7c7cd2c
---

# Gate ledger — pair#220 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-09T13:45:02-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `sidecar-fastpath-case-enumeration` Fast path treats the registry as the set of right terminals; it is a subset, and 0-id / unregistered cases are unspecified
  pickRightTerminal classifies via RoleForPaneWith (layoutcmd.go:245, shortcut.go:210-226), which admits panes by
  terminal_command or title with an EMPTY registry — layoutcmd_test.go:81 and :126 both resolve with terminalPaneIDs nil.
  A pair term can be live and unregistered (run.go:1656-1659 no-ops without ZELLIJ_PANE_ID; run.go:242-244 warns and
  continues on error). Registry {4}, no record, report has focused pane 3 with terminal_command "sh -c exec pair term"
  plus pane 4: slow path returns 3, the plan's single-live-id rule returns 4 — Alt+k and Alt+Shift+arrow then mean
  different split halves, the #216 BR-10 property. Zero live ids and registry-read-error are named by neither branch.
  Put the (live ids x recorded preference) -> (answer | fall through) table in the Plan, and say what happens when the
  registry names a live pid whose pane is gone, since the pane report is today the cross-check that drops it.
- **PQ-2** [Important] `agreement-oracle-strength` Agreement test names no function and no adversarial input class
  "Fast-path resolution in layoutcmd" names nothing unit-testable. Name the pure decision function, taking no Runtime
  (ARCH-PURE) e.g. resolveFromSidecars(liveIDs []string, lastTerminal string) (string, bool), and give one strategy
  line: generated cross-product over pane sets (command-classified, title-classified, registry-only) x registry
  subsets (empty, subset, disjoint) x record (none, live, stale, unregistered-but-present), asserting fast answers
  implies fast equals pickRightTerminal. The adversarial class is the registry disagreeing with the pane report;
  hand-picked cases are blind to it by construction.
- **PQ-3** [Important] `latency-class-not-instance` Two of at least five interactive list-panes callers are fixed; Alt+k from the terminal still pays 590ms
  handleTerminalChord (run.go:514-541) does not handle Alt+k, so it falls through to handleChord (run.go:451-453) ->
  focusedWorkbenchPanes -> ListPanesJSON (run.go:116, :143-144). splitTerminalDown -> currentRightTerminalPane
  (run.go:545, :567-568) and Alt+Shift+Enter -> RunToggleFocused (run.go:537, layoutcmd.go:189) pay it too. After this
  plan lands, Alt+k right-to-left is still ~600ms while left-to-right is fast. Enumerate every interactive
  ListPanesJSON caller and mark each fixed-now or deliberately-not with the reason; toggle-focused needs --geometry
  (tiledScreenSize, layoutcmd.go:229-243) and belongs in the Not-in-scope list, which currently omits it.
- **PQ-4** [Important] `untrusted-sidecar-parse` syscall.Kill(pid, 0) needs the pid>0 guard; positivePID already exists and is unused by Alive
  LiveIDs passes strconv.Atoi output straight to alive (shortcut.go:585-591), so a registry line "4 0" or "4 -1"
  reaches syscall.Kill(0,0) / Kill(-1,0) — process-group and broadcast probes that return nil, reporting the id alive
  forever and pinning a dead pane the fast path will hand to focus-pane-id. The registry is an append-only,
  concurrently written, hand-editable file: untrusted input (ARCH-SECURE). Reuse positivePID (procutil.go:14, already
  used by identity_darwin.go:14) rather than a new check (ARCH-DRY), and name how the not-ours row gets its pid
  (pid 1 -> EPERM).
- **PQ-5** [Minor] `existing-pattern-reuse` draftroute already implements cache-then-list-panes-on-miss; name it as the precedent
  RouteLua consults CachedDraftPaneID and only calls ListPanesJSON on a miss (route.go:60-70, :79-87), with
  ValidateCachedDraftPane gating the sidecar on session plus procutil.Alive. Say whether the new fast path reuses that
  validation shape or deliberately differs (ARCH-DRY).

## Round 2 — 2026-09-09T13:49:56-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — addressed — Registry-as-subset stated, four-branch case table given, race failure mode named and accepted in writing.
- PQ-2 — not-addressed — Still no named function; nine prose cases substituted for the generation strategy, and bullet 2 points at RouteLua's IO shape rather than ValidateCachedDraftPane's pure one.
- PQ-3 — addressed — All five ListPanesJSON call sites enumerated with dispositions; verified against the tree.
- PQ-4 — addressed — positivePID reuse named with the kill(0)/kill(-1) rationale; procutil.go:14 confirmed unused by Alive.
- PQ-5 — addressed — draftroute named as the precedent with file:line.

### Raised

- **PQ-6** [Important] `unverified-existing-behavior-claim` The termcmd:144 row claims focusedWorkbenchPanes needs IDs only; run.go:129 consumes the whole focused pane for role classification
  The row cites :131 and :133 and skips :129, where panes.focused goes into RoleForPaneWith
  (shortcut.go:603-614), which classifies on TerminalCommand/Title, not ID. Decide branches on that
  Role (shortcut.go:253) and Alt+k from the terminal lives only in the PaneRoleRightTerminal arm
  (shortcut.go:264-278); everything else hits DispositionPass at :309. Building workbenchPanes from
  rt.CurrentPaneID() alone therefore silently disables Alt+k AND its RecordLastTerminalPaneID write,
  starving the last-terminal sidecar this issue's own fast path reads. Worse where it matters:
  CurrentPaneID is os.Getenv("ZELLIJ_PANE_ID") (run.go:1631-1633), empty in exactly the case where
  RegisterTerminalPane no-ops (run.go:1656-1660), and the is_focused scan at run.go:157-160 is the
  fallback that disappears. State how Role is derived without the list -- RoleForPaneWith on a bare
  Pane{ID} plus the registry overlay -- and what happens when the pane is unregistered.
- **PQ-7** [Minor] `divergent-runtime-impls` Toggle-focused's geometry justification holds for only one of its two entry points
  layoutcmd.OSRuntime.ListPanesJSON passes --geometry (layoutcmd.go:255) but termcmd's does not
  (run.go:1624), and handleTerminalChord reaches RunToggleFocused with the termcmd runtime
  (run.go:537). tiledScreenSize (layoutcmd.go:229-243) reads X/Columns/Rows, so Alt+Shift+Enter from
  the terminal pane already computes screen size from zeroed geometry. Does not change the row's
  disposition; note it so the close review can decide whether it is a separate issue.

## Round 3 — 2026-09-09T13:53:16-07:00 (claude) — passed

### Disposed

- PQ-2 — addressed — Names resolveFromSidecars with the exact signature, no Runtime, plus the generated cross-product and the equals-pickRightTerminal property.
- PQ-6 — addressed — Answered by role-known-by-construction rather than the overlay I proposed; run.go:152-155 backs the claim and run.go:180-184 keeps the sidecar write.
- PQ-7 — not-addressed — Re-verified (run.go:1624 vs layoutcmd.go:256, reached via run.go:537); Minor, carried to close review.

### Raised

- **PQ-8** [Minor] `sidecar-fastpath-case-enumeration` Fallback predicate is stated for one substituted field, not for every field the pane list supplied
  2nd in this family, so the deliverable is the rule, not the instance: a sidecar
  substitution falls back to the pane list when ANY replaced field is unavailable.
  The termcmd:144 row replaces focused pane, role and draft id but names a fallback
  only for empty CurrentPaneID. CachedDraftPaneID can miss (route.go:53-68); with no
  recorded LastLeftPaneID, Decide falls through to DraftPaneID (shortcut.go:264-267)
  and runDecision silently returns nil on an empty target (run.go:192-195), so Alt+k
  from the terminal becomes inert where the pane list answers today. RouteLua already
  implements the disjunctive predicate at route.go:75-87.

## Open findings

- **PQ-7** [Minor] `divergent-runtime-impls` Toggle-focused's geometry justification holds for only one of its two entry points
- **PQ-8** [Minor] `sidecar-fastpath-case-enumeration` Fallback predicate is stated for one substituted field, not for every field the pane list supplied
