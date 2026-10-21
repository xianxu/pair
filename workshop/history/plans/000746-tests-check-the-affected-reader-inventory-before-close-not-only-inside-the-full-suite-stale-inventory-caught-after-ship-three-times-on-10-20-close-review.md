# Boundary Review — pair#746 (whole-issue close)

| field | value |
|-------|-------|
| issue | 746 — tests: check the affected-reader inventory before close, not only inside the full suite (stale inventory caught after SHIP three times on 10-20) |
| repo | pair |
| issue file | workshop/issues/000746-tests-check-the-affected-reader-inventory-before-close-not-only-inside-the-full-suite-stale-inventory-caught-after-ship-three-times-on-10-20.md |
| boundary | whole-issue close |
| milestone | — |
| window | 80fe2ecd13f3b64c0bf57ca32afbc8e9ce31ba54..baef9a3ee1b02cd122391d4adee649cf57704617 |
| command | sdlc close --issue 746 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T17:16:42-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The fast target and inventory-first execution work, and README/atlas updates cover the new surface. One Important defect remains: a reader’s changed classification can produce a repair command that the same CLI rejects.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: repair-command-validity
    title: |
      Changed source readers receive an invalid repair disposition
    detail: |
      tests/affected-go-readers-test.py:152 prefers the recorded disposition even when repository_source_scan now requires fixed_scanner. Both affected cases were reproduced: local_or_literal_fixture → repository source reader and non_go_full_fallback → repository source reader. With every test already registered in SOURCE_GUARDS, --check prints the old disposition; executing that command fails with "repository source reader requires fixed_scanner". Explicitly recording fixed_scanner succeeds and clears the check. Prefer the required current classification over the historical disposition, and add regression tests that execute the printed command for both transitions. This leaves the Spec's exact-repair-command requirement incomplete (ARCH-PURPOSE).
```

1. **Strengths**

   - `scripts/test-affected.py:452` runs the inventory check before selected children; failure-ordering coverage passes.
   - The fast target reuses the existing audit without launching Go or seeded guard tests.
   - Explicit record/remove operations preserve other JSON fields and reject duplicate keys; CLI tests pass.
   - README and atlas both document the new target.

2. **Critical findings:** None.

3. **Important findings:** Invalid repair disposition, detailed above.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed the real `make test-affected-inventory`, all 12 runner tests, four CLI tests, and seven existing inventory tests. Independent fixtures reproduced both classification transitions. The seeded Go negative-control test was not rerun. Checkout remains clean.

6. **Architectural notes**

   - **ARCH-DRY — pass:** existing scanner, audit, and guard registry are reused.
   - **ARCH-PURE — pass:** no new business logic mixed into application IO; filesystem behavior is exercised through private fixtures.
   - **ARCH-PURPOSE — flag:** generated repair commands do not cover classification changes correctly.
   - **ARCH-EVOLVE — pass:** no schema change; inventory rewrites retain unrelated fields and reject duplicate JSON keys.

7. **Plan revision recommendations**

   Add a `## Revisions` entry requiring generated repair commands to use the current required disposition, with executable regression coverage for both non-scanner-to-source-reader transitions.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"2447ed29-978d-4268-b034-2d6adc3c4763","repository":"github.com/xianxu/pair","issue_id":"000746","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/1c8fa493d2184272","reviewed_head":"baef9a3ee1b02cd122391d4adee649cf57704617","slot":"pair:7","requested_at":"2026-10-21T00:15:02.97201Z"},"phase":"finished","pid":73470,"started_at":"2026-10-21T00:15:03.09505Z","finished_at":"2026-10-21T00:16:26.175173Z","session_id":"01a15bcf-bd79-72e1-8388-729281578299","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.071979458s
head: "baef9a3ee1b02cd122391d4adee649cf57704617"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T17:19:26-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 746 — tests: check the affected-reader inventory before close, not only inside the full suite (stale inventory caught after SHIP three times on 10-20) |
| repo | pair |
| issue file | workshop/issues/000746-tests-check-the-affected-reader-inventory-before-close-not-only-inside-the-full-suite-stale-inventory-caught-after-ship-three-times-on-10-20.md |
| boundary | whole-issue close |
| milestone | — |
| window | 80fe2ecd13f3b64c0bf57ca32afbc8e9ce31ba54..a4f0b6ae0c9dd679c3829679485b43239f2de7a4 |
| command | sdlc close --issue 746 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T17:19:26-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned changes deliver the fast inventory check, run it before affected work, and document the before-close requirement. BR-1 is addressed with executable regression coverage. No new findings.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      tests/affected-go-readers-test.py:152 prioritizes the required current classification. The regression at line 240 executes emitted commands for both historical dispositions and confirms the check clears. Restoring the old expression in a temporary copy makes both cases fail.
```

Done-when: current — the target, early execution, checklist instruction, and diagnostic tests cover the implementation; the revision repairs the existing exact-command contract.

1. **Strengths**

   - The fast target reuses the existing audit and scanner registry.
   - Inventory failure stops affected execution before selected children (`scripts/test-affected.py:452`).
   - Record operations reject scanner waivers and duplicate keys, while preserving unrelated JSON fields.
   - README and atlas document the new target and repair workflow.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed the real `make test-affected-inventory`, all 12 runner tests, five CLI tests, and seven inventory tests. Confirmed both BR-1 regression cases fail without the fix. The unchanged seeded Go negative-control test and full suite were not rerun. Checkout remains clean; range whitespace checks passed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** existing audit, reader discovery, classification, and registry are reused.
   - **ARCH-PURE — pass:** the runner remains a thin orchestration layer; CLI integration uses private fixtures and injected child executables.
   - **ARCH-PURPOSE — pass:** early stale-inventory detection and executable repair advice fulfill the issue’s purpose.
   - **ARCH-EVOLVE — pass:** no schema change; rewrites retain unrelated fields, and duplicate keys are rejected.

7. **Plan revision recommendations:** None. The existing BR-1 revision matches the verified correction.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"db218483-f63a-4bad-a73f-5216df891b51","repository":"github.com/xianxu/pair","issue_id":"000746","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/8c4c9956375163ad","reviewed_head":"a4f0b6ae0c9dd679c3829679485b43239f2de7a4","slot":"pair:7","requested_at":"2026-10-21T00:18:16.010851Z"},"phase":"finished","pid":2758,"started_at":"2026-10-21T00:18:16.232567Z","finished_at":"2026-10-21T00:19:25.896986Z","session_id":"01a15bd2-b071-7e91-924b-b6951c66b99f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.527078708s
head: "a4f0b6ae0c9dd679c3829679485b43239f2de7a4"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
