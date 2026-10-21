# Boundary Review — pair#731 (whole-issue close)

| field | value |
|-------|-------|
| issue | 731 — A pane that fell back to a native shell still classifies as a Pair terminal |
| repo | pair |
| issue file | workshop/issues/000731-a-pane-that-fell-back-to-a-native-shell-still-classifies-as-a-pair-terminal.md |
| boundary | whole-issue close |
| milestone | — |
| window | 49566452104f129c38c8db5e4e985a0a5ae5087e..a596629fca8c9283755a3d5ac9ed289e3f222b87 |
| command | sdlc close --issue 731 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T21:09:07-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation matches the revised ownership policy, and all seven targeted packages passed. Native conformance also confirmed that shell exec removes routing ownership while preserving the working shell, another registered pane, and registry bytes. One documentation correction remains: the unchanged acceptance criteria contradict the approved compatibility exception and name the wrong registry API.

Done-when: stale — it promises unconditional removal through `LiveIDs`, while production uses `LiveOwnerIDs` and deliberately retains ownership when inspection is unavailable.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: acceptance-criteria-contract-alignment
    title: |
      Done-when still promises the superseded unconditional ownership policy
    detail: |
      workshop/issues/000731-a-pane-that-fell-back-to-a-native-shell-still-classifies-as-a-pair-terminal.md:44 requires LiveIDs and RoleForPaneWith to stop granting ownership after fallback. LiveIDs remains an injected-predicate storage reader, and terminal_owner.go:56 implements LiveOwnerIDs with the approved unavailable-inspection exception documented at issue line 68. Update Done when to name LiveOwnerIDs, require rejection of confirmed shell argv or identity changes, and explicitly cover retained ownership plus bounded diagnostics when inspection is unavailable. This is an ARCH-PURPOSE traceability gap.
```

1. **Strengths**

   - Shared owner filtering covers term, layout, clipboard and Couch, including registry-only fast paths.
   - Identity-bracketed argv inspection distinguishes confirmed mismatch from unavailable evidence.
   - Stateful tests cover exec, identity replacement, inspection failures, diagnostic failure and bounded inspection counts.
   - Native conformance verifies shell usability, control-pane ownership, unchanged registry bytes and exact cleanup.

2. **Critical findings**

   None.

3. **Important findings**

   The acceptance-criteria mismatch above. Correcting the issue text is sufficient; the implemented compatibility policy is explicitly approved in its revisions.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed uncached tests for `workbenchshortcut`, `termcmd`, `layoutcmd`, `clipcmd`, `couchcmd`, `artifactpath` and `repolint`.

   Native conformance passed with `TMPDIR=/tmp`. Its first invocation rejected the inherited temporary path because the private socket address exceeded its length guard. No checkout files were changed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared adapter and predicate caching avoid duplicate ownership logic.
   - **ARCH-PURE — pass:** role/argv decisions remain directly testable; native observation uses an injected source.
   - **ARCH-PURPOSE — flag:** implementation covers the routing consumers; acceptance criteria need reconciliation.
   - **ARCH-MOCK — pass:** stateful process doubles share the production seam; native conformance exercises real observations.
   - **ARCH-CONSTRAINTS — pass:** N/2N tests establish per-PID bounds and prohibit process-table enumeration.
   - **ARCH-SECURE — pass:** diagnostics contain bounded reason categories rather than argv or native errors.
   - **ARCH-ORDER — pass:** observations retain no authoritative state between reads; tests control replacement during inspection.
   - **ARCH-FUNERAL — pass:** no registry rewrite or new durable artifact family; diagnostics reuse managed retention.
   - **ARCH-EVOLVE — pass:** registry format remains unchanged, with unknown bytes preserved.

   Atlas documents the new ownership surface. No new command, flag, keybinding or configuration requires a README addition.

7. **Plan revision recommendations**

   Add a `## Revisions` entry recording the acceptance-criteria reconciliation: owner-aware API naming, confirmed-mismatch rejection, and the inspection-unavailable compatibility exception. Existing behavioral tests support this prose correction.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"9800e770-407a-4376-a76b-a7b6cf143fb3","repository":"github.com/xianxu/pair","issue_id":"000731","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/ba22136acb4c22f6","reviewed_head":"a596629fca8c9283755a3d5ac9ed289e3f222b87","slot":"pair:6","requested_at":"2026-10-21T04:07:04.742808Z"},"phase":"finished","pid":85826,"started_at":"2026-10-21T04:07:04.867544Z","finished_at":"2026-10-21T04:09:06.377737Z","session_id":"01a15ca4-2ab9-7e80-b38e-79dc727bdd53","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.868029292s
head: "a596629fca8c9283755a3d5ac9ed289e3f222b87"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T21:14:37-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 731 — A pane that fell back to a native shell still classifies as a Pair terminal |
| repo | pair |
| issue file | workshop/issues/000731-a-pane-that-fell-back-to-a-native-shell-still-classifies-as-a-pair-terminal.md |
| boundary | whole-issue close |
| milestone | — |
| window | 49566452104f129c38c8db5e4e985a0a5ae5087e..f09a8ae39e6891f9c3882967d48c2de3e83da2ae |
| command | sdlc close --issue 731 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T21:14:37-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation satisfies the approved ownership policy, and BR-1’s acceptance-criteria mismatch is corrected. Confirmed shell handoffs lose ownership; unavailable inspection retains the live registry answer with bounded diagnostics. No blocking findings remain.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Issue line 44 now names LiveOwnerIDs and explicitly covers confirmed mismatch/replacement, unavailable-inspection retention, bounded diagnostics, and diagnostic-write failure. This matches terminal_owner.go:32–80 and the owner/diagnostic regression tests. The correction is prose-only; the pinned before/after diff changes no runtime policy.
```

1. **Strengths**

   - [terminal_owner.go:32](/Volumes/TestScratch/review-checkouts/07953dfb79ac5bb6/cmd/internal/workbenchshortcut/terminal_owner.go:32) distinguishes confirmed mismatch from unavailable inspection and brackets argv reads with identity observations.
   - [shortcut.go:757](/Volumes/TestScratch/review-checkouts/07953dfb79ac5bb6/cmd/internal/workbenchshortcut/shortcut.go:757) requires registration for terminal routing while preserving other Pair roles. The reader sweep confirmed all four production consumers use the shared owner filter.
   - [terminal_owner_live_test.go:81](/Volumes/TestScratch/review-checkouts/07953dfb79ac5bb6/cmd/internal/workbenchshortcut/terminal_owner_live_test.go:81) exercises real exec metadata, usable shell input, preservation of another owner and unknown registry bytes, and exact process cleanup.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   All seven reviewed packages passed: workbenchshortcut, termcmd, layoutcmd, clipcmd, couchcmd, artifactpath, and repolint.

   Native conformance passed with `TMPDIR=/tmp`; the inherited temporary path initially exceeded the Unix socket limit. The passing run confirmed unchanged PID/birth identity and stale startup metadata, changed shell argv, removed ownership, preserved control ownership, and verified cleanup.

   Tests cover replacement during inspection, unavailable reads, diagnostic failure, duplicate-PID inspection bounds, stale metadata, and cached-routing refusal. BR-1 requires no runtime mutation test because its correction only reconciles documentation.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared owner adapter covers every production registry reader.
   - **ARCH-PURE — pass:** argv and role decisions remain pure; process and storage IO use explicit seams.
   - **ARCH-PURPOSE — pass:** ownership filtering covers cached routing as well as metadata-based paths.
   - **ARCH-MOCK — pass:** stateful process doubles exercise exec, replacement, and unreadability; live conformance checks native behavior.
   - **ARCH-CONSTRAINTS — pass:** counters enforce per-unique-PID bounds without process-table enumeration.
   - **ARCH-SECURE — pass:** typed argv inspection avoids command-string heuristics; diagnostics omit raw argv and errors.
   - **ARCH-ORDER — pass:** observations retain no authoritative state between reads; controlled callbacks exercise identity changes during inspection.
   - **ARCH-FUNERAL — pass:** no new durable artifact family; diagnostics reuse managed retention and native fixtures verify cleanup.
   - **ARCH-EVOLVE — pass:** registry format remains unchanged, with unknown bytes preserved.

   Atlas documents the ownership contract and lint rule. No new command, flag, configuration key, or usage step requires a README addition.

7. **Plan revision recommendations:** None. Existing revisions explain the shared runtime wiring and approved compatibility policy.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"b61faeaa-46fd-4d3f-ba04-6273a4afa1b6","repository":"github.com/xianxu/pair","issue_id":"000731","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/07953dfb79ac5bb6","reviewed_head":"f09a8ae39e6891f9c3882967d48c2de3e83da2ae","slot":"pair:6","requested_at":"2026-10-21T04:11:05.869467Z"},"phase":"finished","pid":37355,"started_at":"2026-10-21T04:11:05.967209Z","finished_at":"2026-10-21T04:14:36.679396Z","session_id":"01a15ca7-d8c7-7101-8fb2-fef5affa86f9","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.534207667s
head: "f09a8ae39e6891f9c3882967d48c2de3e83da2ae"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
