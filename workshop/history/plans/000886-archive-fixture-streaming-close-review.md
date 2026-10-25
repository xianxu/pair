# Boundary Review — pair#886 (whole-issue close)

| field | value |
|-------|-------|
| issue | 886 — Stream oversized archive protocol fixtures and restore the couchcmd write allowance |
| repo | pair |
| issue file | workshop/issues/000886-archive-fixture-streaming.md |
| boundary | whole-issue close |
| milestone | — |
| window | fd31b8116bba42b8405419a850e3cb19b2deff57..6757898a3c009adecb08fea264bafae231a690de |
| command | sdlc close --issue 886 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-25T09:55:44-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The fixture implementation matches the Spec, and focused tests pass. No code defect was found. SHIP is blocked by incomplete qualification evidence: the full couchcmd run exceeded the review’s 4-minute timeout without producing a final write-budget verdict, and the tracker records pre-close as pending.

Done-when: current — the unchanged criteria still cover exact streaming output, retained assertions, write-budget qualification, and recorded checks.

```findings
findings:
  - id: new
    severity: Important
    kind: unverifiable
    family: required-verification-evidence
    title: |
      Complete-package qualification at 96 MiB remains unverified
    detail: |
      workshop/issues/000886-archive-fixture-streaming.md:57-61 requires complete couchcmd qualification and recorded pre-close success; lines 140-144 record that check as pending. The attempted command `go test ./cmd/internal/couchcmd -count=1 -timeout=4m` exited 1 after its 4-minute timeout during TestForcedSwitchSupervisorRejectsChangedAuthority, without a final physical-write verdict. This finding covers both outstanding verification instances: complete couchcmd within the unchanged 100663296-byte cap, and successful recorded pre-close. Supply completed final-code host results for both; the timeout does not establish a defect in the fixture change.
```

1. **Strengths**

   - One shared writer streams both overflow suffixes in checked 32 KiB blocks (`archive_schedule_protocol_test.go:101`).
   - Child-output tests verify exact max+1 length, prefix, and suffix contents for both payloads (`:174`).
   - Refusal tests now pin bounded source-file setup while retaining diagnostic and report-authority assertions (`:286`).
   - Production parsing, capture limits, timeout, and budget enforcement are unchanged.

2. **Critical findings:** None.

3. **Important findings:** Qualification evidence blocker above.

4. **Minor findings:** None.

5. **Test coverage notes**

   Focused padding, write-failure, child-output, protocol, refusal, cancellation, and inherited-descriptor tests passed. Their combined run measured **8,511,488 bytes**, below the **100,663,296-byte** cap. Native actual-excess refusal and budget-verdict tests passed. `make test-affected-inventory` and diff whitespace checks passed. Complete-package qualification remains unresolved.

6. **Architectural notes**

   - **ARCH-DRY: pass** — both payloads share the suffix writer.
   - **ARCH-PURE: pass** — generation uses an injected writer; process and filesystem operations remain in fixture glue.
   - **ARCH-PURPOSE: pass for implementation** — both original overflow cases remain covered; qualification is pending.
   - **ARCH-EVOLVE: pass** — no persisted-format semantics or rewrite behavior changed.

   The atlas documents the fixture surface. No new user-facing command, flag, or configuration requires a README update. The short plan contains no Core concepts table to cross-check.

7. **Plan revision recommendations:** No contract revision needed. Record completed qualification evidence in the issue Log.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"9700a8ed-2cfd-4e50-8890-3b3b861590c9","repository":"github.com/xianxu/pair","issue_id":"000886","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/c6d9ecd20c80f754","reviewed_head":"6757898a3c009adecb08fea264bafae231a690de","slot":"pair:11","requested_at":"2026-10-25T16:50:00.725305Z"},"phase":"finished","pid":30491,"started_at":"2026-10-25T16:50:00.933605Z","finished_at":"2026-10-25T16:55:40.948856Z","session_id":"01a173f8-1821-7ef2-922d-6c07aa3e713a","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 4.410788666s
head: "6757898a3c009adecb08fea264bafae231a690de"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-25T10:34:36-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 886 — Stream oversized archive protocol fixtures and restore the couchcmd write allowance |
| repo | pair |
| issue file | workshop/issues/000886-archive-fixture-streaming.md |
| boundary | whole-issue close |
| milestone | — |
| window | fd31b8116bba42b8405419a850e3cb19b2deff57..3f092cf06e1aee89f33e0868cbfd12127b3b8f8b |
| command | sdlc close --issue 886 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-25T10:34:36-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation satisfies the issue’s streaming contract without changing production parsing, limits, or cleanup behavior. BR-1 is resolved: the retained host log matches the committed digest, its recorded attempt exited successfully, and the tested implementation has no subsequent executable changes. No new findings.

Done-when: current — the criteria still cover exact output, bounded fixture setup, retained protocol assertions, 96 MiB qualification, and recorded pre-close success.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Verified the retained suite.log against SHA-256 a0912105109bdf0534ee4c279ec3d8aeb1b72642cb831e37bedddb632aca18cc and the SDLC attempt record: exit 0, no timeout, implementation head dd3570f1aa18dcffc1eed4beb2a387088362103b. Complete couchcmd passed in 77.066s and testdisk in 18.396s. CouchCmd remains 100663296 bytes with native TestMain enforcement; executable inputs are unchanged through the reviewed head. This answers both outstanding verification instances.
```

1. **Strengths**

   - Both overflow cases share a bounded writer that propagates errors and short writes (`cmd/internal/couchcmd/archive_schedule_protocol_test.go:101`).
   - Real child-output tests verify exact max+1 length and contents for invalid output and valid JSON followed by whitespace (`:174`).
   - Refusal tests enforce small source files while retaining diagnostic, capture-bound, and report-authority assertions (`:286`).
   - The performance atlas documents the fixture behavior and measured savings.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed padding, write-failure, child-output, protocol, refusal, cancellation, and inherited-descriptor tests. Their combined run measured **8,511,488 bytes** against the **100,663,296-byte** cap. Budget-verdict and native actual-excess refusal tests passed. Inventory validation and diff whitespace checks passed. Complete-suite qualification was verified through the retained host evidence.

6. **Architectural notes**

   - **ARCH-DRY: pass** — both overflow payloads use one suffix writer.
   - **ARCH-PURE: pass** — byte generation uses an injected writer; process and filesystem operations remain in fixture glue.
   - **ARCH-PURPOSE: pass** — both original overflow cases and required qualification are delivered.
   - **ARCH-EVOLVE: pass** — no persisted-format semantics or rewrite paths changed.

   Atlas coverage is present. No new user-facing command, flag, or configuration requires a README change. The short plan has no Core concepts table.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e67ed49f-f54e-492c-abb5-2f3486bbdd7e","repository":"github.com/xianxu/pair","issue_id":"000886","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/950cb89604a42a19","reviewed_head":"3f092cf06e1aee89f33e0868cbfd12127b3b8f8b","slot":"pair:11","requested_at":"2026-10-25T17:33:34.358387Z"},"phase":"finished","pid":58789,"started_at":"2026-10-25T17:33:34.455811Z","finished_at":"2026-10-25T17:34:35.071748Z","session_id":"01a1741f-f8a2-7880-b2de-1a248666a40c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.099988417s
head: "3f092cf06e1aee89f33e0868cbfd12127b3b8f8b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
