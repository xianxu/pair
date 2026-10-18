# Boundary Review — pair#662 (whole-issue close)

| field | value |
|-------|-------|
| issue | 662 — Review observation diagnostic positive control times out in real BufWritePre resolver |
| repo | pair |
| issue file | workshop/issues/000662-review-observation-resolver-timeout.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7795390df65cfbb6bb5b3ead495dd1c87dfc3896..ae9e8f6dda8c8771d87a26259bb9985d8e2057f5 |
| command | sdlc close --issue 662 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-18T04:47:43-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation fulfills the revised scope: ordering checks use a stateful fixture, native deadline conformance remains separate, and empty-stderr failures produce useful refusal diagnostics. Production deadlines and identity/index checks are unchanged. No blocking defects found; one minor duplication remains.

```findings
findings:
  - id: new
    severity: Minor
    family: shared-text-validation
    title: |
      ARCH-DRY: fixture repeats the existing UTF-8 validation algorithm
    detail: |
      tests/lib/review_resolver_fake.lua:4 introduces utf8_text, duplicating the UTF-8 and NUL validation already implemented in nvim/review/document_bytes.lua:4–19. This is the only new instance in the window. Reuse document_bytes.decode for validation, or extract a shared pure validator, so the fixture cannot drift from the document contract.
```

1. **Strengths**
   - The diagnostic fix preserves nonempty stderr and supplies timeout/exit reasons when absent ([identity.lua:15](nvim/review/identity.lua:15)).
   - The fixture observes actual HEAD/ref/document files and captures async bytes before delivery.
   - Refusal controls cover unknown identity, actual branch movement, and timeout while retaining activation and document contents.
   - Atlas documentation accurately describes the separation; no new user-facing command or configuration requires a README update.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** The UTF-8 validation duplication above.

5. **Test coverage**
   - All 13 observation diagnostic tests passed, including native sync/async timeout conformance.
   - Identity and restore-controller Lua tests passed.
   - Focused real Git identity tests passed, including brief stdout-holder success and branch/HEAD/index movement checks.
   - The new identity regression failed against the pinned base implementation with `diagnostic = ""`.
   - Independently inspected retained historical evidence: call 23 timed out after 2578 ms with code 124, empty stderr, and zero stdout bytes. It does not establish an inner Git-phase cause.

6. **Architecture**
   - **ARCH-DRY:** Minor flag above; existing callback-gate reuse is sound.
   - **ARCH-PURE:** Pass. Diagnostic decoding remains pure; filesystem/process simulation stays at the test boundary.
   - **ARCH-PURPOSE:** Pass. The ordering oracle no longer depends on real resolver speed, while genuine refusals and native/backend conformance remain exercised.

7. **Plan revisions:** None required.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"accdf0e4-b2fe-4353-905a-030a4e59fa9d","repository":"github.com/xianxu/pair","issue_id":"000662","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/5e7a0572b436e92d","reviewed_head":"ae9e8f6dda8c8771d87a26259bb9985d8e2057f5","slot":"pair:3","requested_at":"2026-10-18T11:45:58.908163Z"},"phase":"finished","pid":2325,"started_at":"2026-10-18T11:45:59.134667Z","finished_at":"2026-10-18T11:47:42.236378Z","session_id":"01a14ed5-3b39-7dd3-b132-d7c5529df088","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.297846375s
head: "ae9e8f6dda8c8771d87a26259bb9985d8e2057f5"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
