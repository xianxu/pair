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

---

## Re-review — 2026-10-21T04:28:03-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 763 — Go reader audit: isolate certificates so independent changes merge cleanly |
| repo | pair |
| issue file | workshop/issues/000763-artifactpath-manifest-split-the-2-000-line-central-inventory-by-area-so-parallel-landings-stop-conflicting.md |
| boundary | whole-issue close |
| milestone | — |
| window | 34af4bb26c7a16dd0d69e49c075f1e996e64d940..45081cdbf92674fcd1dd42998ee3ee14c7f281bd |
| command | sdlc close --issue 763 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T04:28:03-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation satisfies the revised certificate-splitting contract. All 462 migrated entries match the base exactly, independent changes merge cleanly, and BR-1 is addressed with passing regressions that fail without the fix. No new findings. Required recorded full-suite completion remains a separate gate; it was not independently established here.

Done-when: current — it covers the revised layout, merge isolation, refusal controls, unknown-field preservation, and documentation.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      tests/affected-go-readers-test.py:108–146 preserves numeric tokens recursively and rejects non-JSON constants. Both regression tests at :261 and :273 pass; restoring default numeric decoding in a temporary copy makes both fail, confirming the fix is exercised.
```

1. **Strengths**

   - Independently verified exact migration of all 462 paths, dispositions, and SHA256 values.
   - Real-Git regression confirms independent reader updates merge with exact certificate bytes.
   - Named regeneration leaves unrelated changed readers unaccepted and retains SOURCE_GUARDS enforcement.
   - README and atlas explain certificate ownership, regeneration, and removal.

2. **Critical findings**

   None.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   All 20 inventory tests and 12 affected-runner tests passed. Inventory checking and diff whitespace checking passed. Regression mutation checks confirmed BR-1 tests fail without the corrected decoding. The checkout remained clean.

6. **Architectural notes**

   - **ARCH-DRY: pass** — one loader reconstructs the existing audit model.
   - **ARCH-PURE: pass** — deterministic encoding and audit decisions remain separate from filesystem and CLI operations.
   - **ARCH-PURPOSE: pass** — both inventory consumers use the shared loader; the aggregate is removed and independent updates are verified.
   - **ARCH-EVOLVE: pass** — nested unknown values survive rewriting, including precise numeric tokens; known fields and duplicate keys remain strict.

7. **Plan revision recommendations**

   None. The pending verification/close/landing checklist accurately identifies remaining external gate work.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"78249a07-0a5e-4e74-96fc-904b24a09362","repository":"github.com/xianxu/pair","issue_id":"000763","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/89810bd97928a4b5","reviewed_head":"45081cdbf92674fcd1dd42998ee3ee14c7f281bd","slot":"pair:2","requested_at":"2026-10-21T11:26:26.924189Z"},"phase":"finished","pid":80589,"started_at":"2026-10-21T11:26:27.066254Z","finished_at":"2026-10-21T11:28:02.912234Z","session_id":"01a15e36-6c93-71d3-a04e-adbc3835ccef","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.298979917s
head: "45081cdbf92674fcd1dd42998ee3ee14c7f281bd"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T05:14:19-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 763 — Go reader audit: isolate certificates so independent changes merge cleanly |
| repo | pair |
| issue file | workshop/issues/000763-artifactpath-manifest-split-the-2-000-line-central-inventory-by-area-so-parallel-landings-stop-conflicting.md |
| boundary | whole-issue close |
| milestone | — |
| window | 54fcc8b82d72355d7314c7adb848a63a57e13f63..7b5e43b79ea17f9c2d8d79b5cb57bbdc3a1cd99b |
| command | sdlc close --issue 763 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T05:14:19-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The conflict resolution preserves both sides’ intent: all 467 aggregate entries retain their exact paths, dispositions, and hashes in per-reader certificates, and every hash matches current source bytes. The aggregate is removed. Tracker revisions record integration work without changing scope. No new defects found in this interdiff.

Done-when: current — it still covers exact migration, independent certificate updates, audit refusal controls, preservation, and documentation.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The corrected numeric-token decoder and recursive encoder remain unchanged in this interdiff. The precise-number roundtrip and non-JSON-constant refusal regressions both pass; the prior correction remains intact.
```

1. **Strengths**

   - Exact comparison verified all 467 migrated entries, including the five additional readers.
   - All 21 changed certificates match the aggregate’s audited values.
   - Existing README and atlas documentation accurately describe the resulting layout and repair commands.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   All 12 `FastInventoryCLI` tests and 12 affected-runner tests passed. Inventory `--check`, source-digest verification, and diff whitespace checks passed. The checkout remains clean. The integrated full suite was not rerun during this narrow review.

6. **Architectural notes**

   - **ARCH-DRY: pass** — the resolution removes the duplicate aggregate and retains the canonical certificate loader.
   - **ARCH-PURE: pass** — no logic or IO boundaries change.
   - **ARCH-PURPOSE: pass** — the complete aggregate is represented in certificates; no audit entry is stranded.
   - **ARCH-EVOLVE: pass** — migrated values remain exact; the lossless unknown-field handling and its regressions remain intact.

7. **Plan revision recommendations:** None. The added revision records conflict resolution and retains the integrated verification requirement.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e4259632-d65b-4ea8-a7ee-6613bea3a6ed","repository":"github.com/xianxu/pair","issue_id":"000763","recipe":"fix-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/f5cf58603c20265b","reviewed_head":"7b5e43b79ea17f9c2d8d79b5cb57bbdc3a1cd99b","slot":"pair:2","requested_at":"2026-10-21T12:13:12.202278Z"},"phase":"finished","pid":46524,"started_at":"2026-10-21T12:13:12.302675Z","finished_at":"2026-10-21T12:14:19.397603Z","session_id":"01a15e61-39f3-7601-be65-0926df1c5bca","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.714838875s
head: "7b5e43b79ea17f9c2d8d79b5cb57bbdc3a1cd99b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
