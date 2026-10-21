# Boundary Review — pair#763 (whole-issue close)

| field | value |
|-------|-------|
| issue | 763 — Go reader audit: isolate certificates so independent changes merge cleanly |
| repo | pair |
| issue file | workshop/issues/000763-artifactpath-manifest-split-the-2-000-line-central-inventory-by-area-so-parallel-landings-stop-conflicting.md |
| boundary | whole-issue close |
| milestone | — |
| window | 34af4bb26c7a16dd0d69e49c075f1e996e64d940..bf3ffda3f49e3009e1065c20868be7e782de91e2 |
| command | sdlc close --issue 763 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T04:21:26-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The per-reader split preserves all 462 existing audit entries and passes the targeted tests, including a real Git merge. One blocking defect remains: named regeneration silently changes numeric values in unknown certificate fields, violating the preservation contract.

Done-when: current — it covers the revised certificate layout, merge isolation, audit refusals, unknown-field preservation, and documentation.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: persisted-unknown-value-preservation
    title: |
      ARCH-EVOLVE: recording silently changes unknown JSON numbers
    detail: |
      tests/affected-go-readers-test.py:123 decodes unknown values through binary floats, and :165 serializes those altered values during --record. Using the existing FastInventoryCLI fixture, recording returned success but changed nested 0.12345678901234567890123456789 to 0.12345678901234568 and valid JSON 1e400 to nonstandard Infinity. The complete family in this window is the single certificate decode/rewrite path, affecting numeric unknown members at any nesting depth, including arrays. Preserve numeric values losslessly, reject non-JSON constants, and add regressions for precise fractions and large exponents; the existing large-integer test does not exercise this loss.
```

1. **Strengths**

   - Independently confirmed exact equality of all 462 migrated paths, dispositions, and SHA256 values.
   - Real-Git regression verifies independent reader updates merge cleanly.
   - Named regeneration leaves unrelated changed hashes unaccepted.
   - README and atlas document certificate ownership and repair commands.

2. **Critical findings**

   - Unknown numeric values lose precision during regeneration; see the finding above. Use lossless JSON number handling through both decoding and serialization.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   All 18 inventory tests and 12 affected-runner tests passed. The inventory check and diff whitespace check passed. Separate fixture probes reproduced both numeric corruption cases. Recorded full-suite completion remains explicitly pending in the issue; it was not independently established here.

6. **Architectural notes**

   - **ARCH-DRY: pass** — one loader reconstructs the existing audit model.
   - **ARCH-PURE: pass** — audit logic remains separate from certificate IO.
   - **ARCH-PURPOSE: pass** — the aggregate is removed, consumers use the shared loader, and merge isolation is exercised.
   - **ARCH-EVOLVE: flag** — unknown numbers do not survive rewriting unchanged.

7. **Plan revision recommendations**

   None. The documented contract already requires unknown-field preservation; the implementation and regression coverage need correction.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"77ef0265-2bab-40c5-9b02-33913fdfb7dd","repository":"github.com/xianxu/pair","issue_id":"000763","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/f9f4d7fbdc43757d","reviewed_head":"bf3ffda3f49e3009e1065c20868be7e782de91e2","slot":"pair:2","requested_at":"2026-10-21T11:19:23.899919Z"},"phase":"finished","pid":38425,"started_at":"2026-10-21T11:19:24.065532Z","finished_at":"2026-10-21T11:21:25.459867Z","session_id":"01a15e2f-f828-7fe1-a353-f0bd84cca01c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.948482916s
head: "bf3ffda3f49e3009e1065c20868be7e782de91e2"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
