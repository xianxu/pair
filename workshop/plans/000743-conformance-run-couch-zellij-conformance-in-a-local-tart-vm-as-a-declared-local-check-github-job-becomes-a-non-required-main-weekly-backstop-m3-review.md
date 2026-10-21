# Boundary Review — pair#743 (milestone M3)

| field | value |
|-------|-------|
| issue | 743 — conformance: run couch-zellij-conformance in a local Tart VM as a declared local check; GitHub job becomes a non-required main/weekly backstop |
| repo | pair |
| issue file | workshop/issues/000743-conformance-run-couch-zellij-conformance-in-a-local-tart-vm-as-a-declared-local-check-github-job-becomes-a-non-required-main-weekly-backstop.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 49ba55e48927d0bda057725b581940f555b4d90a..f9f74384dd7029f78f92c6177daee7f54988c255 |
| command | sdlc milestone-close --issue 743 --milestone M3 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T03:03:06-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The declaration, nightly handoff, docs, and portable tests largely match the revised M3 plan. Retained native evidence independently validates all 28 required cases and rejects the known-bad skip. Two reproduced retention defects block SHIP: abrupt SDLC death releases reserved capacity while descendants can still write, and unreadable directories silently disappear from byte accounting.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: persisted-writer-reader-bounds
    title: |
      SDLC death releases artifact capacity before its writers have stopped
    detail: |
      scripts/vm/nightly.sh:142-151 normalizes a signal-killed SDLC child into exit_code 137; reserved_night_bytes at line 53 then treats that integer as confirmed completion and charges only current bytes. A stateful executable reproduction killed the SDLC leader while its descendant survived: the wrapper returned 137, charged 218 bytes instead of 268505088, and the descendant subsequently wrote evidence. Near saturation, subsequent admission can therefore exceed the promised 4 GiB bound. ARCH-ORDER/ARCH-CONSTRAINTS/ARCH-FUNERAL. This is the 2nd finding in family persisted-writer-reader-bounds. State and enforce the class-wide rule: release remaining-write capacity only after confirmed writer completion; preserve conservative reservations for signal death and every unconfirmed provider outcome. Add a descendant-survival regression that fails without the correction.
  - id: new
    severity: Important
    kind: defect
    family: persisted-writer-reader-bounds
    title: |
      Retention accounting silently skips unreadable evidence directories
    detail: |
      scripts/vm/nightly.sh:29 uses os.walk without an onerror handler, so traversal failures silently undercount retained evidence. On this host, a private fixture containing 1024 bytes beneath a chmod-000 directory returned retained_bytes=0 instead of refusing admission. ARCH-SECURE/ARCH-CONSTRAINTS. This is the 3rd finding in family persisted-writer-reader-bounds. Apply the same complete-accounting rule across traversal and metadata failures: unknown retained size must refuse admission. Sweep enumeration failure paths and add an unreadable-directory regression.
```

1. **Strengths**

   - The declaration preserves `suite` and explicitly makes `conformance-vm` a non-required landing check with runner-owned capacity.
   - Nightly tests verify exact remote-main selection, literal arguments, unchanged source contents, duplicate-date refusal, cancellation, and failed-night retention.
   - The slot-audit exceptions are restricted by file, function, and literal, with negative-coordinate and stale-entry checks.
   - README and atlas changes cover the new command and evidence lifecycle; hosted PR conformance remains required.

2. **Critical findings**

   - [nightly.sh:142](/Volumes/TestScratch/review-checkouts/3fb3d9c1491ab290/scripts/vm/nightly.sh:142): distinguish SDLC leader exit from confirmed writer completion. Preserve full capacity reservations when completion remains unknown.

3. **Important findings**

   - [nightly.sh:29](/Volumes/TestScratch/review-checkouts/3fb3d9c1491ab290/scripts/vm/nightly.sh:29): propagate traversal errors and refuse admission when retained bytes cannot be measured.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed 11 nightly tests, 8 command-contract tests, 19 reducer tests, 4 path tests, and the focused Go slot-audit tests. Existing tests omit both reproduced failure cases.

   Independently replayed retained native stdout: clean evidence passes with 413 events and 28 required cases; known-bad evidence rejects the required-case skip. Native VM execution was not repeated. Production named-check bootstrap remains explicitly post-landing.

6. **Architectural notes for upcoming work**

   | Principle | Assessment |
   |---|---|
   | ARCH-DRY | Pass: existing VM and SDLC resource owners are reused. |
   | ARCH-PURE | Pass: new nightly IO stays in the command boundary. |
   | ARCH-PURPOSE | Pass: revised burn-in scope is delivered without premature promotion. |
   | ARCH-MOCK | Pass: executable stateful fixtures exercise the production command. |
   | ARCH-CONSTRAINTS | Flag: both findings undermine the declared byte ceiling. |
   | ARCH-SECURE | Flag: unreadable persisted input silently becomes zero-byte evidence. |
   | ARCH-ORDER | Flag: leader death is mistaken for writer completion. |
   | ARCH-FUNERAL | Flag: reservations end before all artifact producers necessarily stop. |
   | ARCH-EVOLVE | Pass: no lossy cross-version rewrite identified in this range. |

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining confirmed versus unconfirmed nightly completion and the evidence needed to release reserved capacity. Enumerate wrapper death, SDLC death, provider uncertainty, and filesystem-accounting failure; require conservative reservation or visible refusal for each.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"0970f99d-0ad0-42a1-adb3-5354e03a62c7","repository":"github.com/xianxu/pair","issue_id":"000743","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/3fb3d9c1491ab290","reviewed_head":"f9f74384dd7029f78f92c6177daee7f54988c255","slot":"pair:3","requested_at":"2026-10-21T09:59:41.688976Z"},"phase":"finished","pid":5791,"started_at":"2026-10-21T09:59:41.792311Z","finished_at":"2026-10-21T10:03:06.025243Z","session_id":"01a15de6-feeb-77b0-90dc-ec57e1508290","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.806525583s
head: "f9f74384dd7029f78f92c6177daee7f54988c255"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
