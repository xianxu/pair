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
