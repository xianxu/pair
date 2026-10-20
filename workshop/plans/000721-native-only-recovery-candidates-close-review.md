# Boundary Review — pair#721 (whole-issue close)

| field | value |
|-------|-------|
| issue | 721 — Offer unconfirmed native recovery candidates without a usable screen |
| repo | pair |
| issue file | workshop/issues/000721-native-only-recovery-candidates.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3a1418f40e77f77d5a6e9e619efa46f8cbe36c8b..0d8f276deb35fa5f2fa201cad88ab884644195c6 |
| command | sdlc close --issue 721 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T00:29:19-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The native-only source reaches preview, persistence, and final launcher validation, and all review-run tests passed. Three contract gaps block shipping: native evidence is reused through retirement without the promised recheck, oversized screens consume budget without being read, and early screen failures discard readable activity. README coverage also needs updating.

1. **Strengths**

   - V1 encoding is pinned by frozen-byte tests; v2 preserves unknown context, source, and candidate fields.
   - Native-only choices remain explicitly unconfirmed and omit fabricated tail scores.
   - Production wiring, portable native homes, interrupted retirement, and exact launcher authority have meaningful coverage.

2. **Critical findings**

   Three findings below cover validation ordering, screen-budget accounting, and partial evidence retention.

3. **Important findings**

   README instructions still describe every candidate listing as ranked by TTY-tail evidence and omit native-only selection’s five-minute lifetime.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - Six-package focused `Recovery|ResumeCandidates` tests.
   - Race tests for `couchcore`, `launcher`, and `sessioninventory`, matching `Recovery|Resume`.
   - `internal/repolint`, `artifactpath`, and range `git diff --check`.

   Existing tests miss the reported cases: mutation between acquisition and retirement; raw captures above the replay limit but below the aggregate limit; and activity retention when events or retained identities fail. No prior findings required disposition. Repository files remain unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** existing collector, retirement, and launcher seams are reused.
   - **ARCH-PURE — pass:** source validation, codecs, ranking, and projection remain directly testable pure logic. Core-concept owners match their listed paths.
   - **ARCH-PURPOSE — flag:** oversized-screen fallback and retained activity do not fully satisfy the documented recovery purpose.
   - **ARCH-MOCK — pass:** stateful inventory/runtime fixtures and portable OS fixtures exercise production seams; production composition is tested.
   - **ARCH-CONSTRAINTS — flag:** reservations charge raw bytes that replay rejects before reading.
   - **ARCH-SECURE — pass:** selectors remain untrusted references; exact artifact and helper authority are independently checked before exec.
   - **ARCH-ORDER — flag:** native evidence is checked again only after retirement and the start claim.
   - **ARCH-FUNERAL — pass:** bounded snapshots use existing record retention; no new durable artifact family appears.
   - **ARCH-EVOLVE — pass:** v2 preserves nested additions, retains strict known-field validation, and keeps v1 compatibility explicit.

7. **Plan revision recommendations**

   Add `## Revisions` entries recording the corrected pre-retirement validation sequence, shared replay-limit preflight, and descriptive activity retained independently of capture authority. Extend the risky-function test strategies with those three regressions; retain the approved contracts.

```findings
findings:
  - id: new
    severity: Critical
    family: recovery-revalidation-before-effects
    title: |
      Native-only evidence is not revalidated before retirement and claim
    detail: |
      cmd/internal/couchcore/recovery_prepared.go:292 skips capture preservation for native-only selections, then line 308 rebuilds authority from the earlier observation without reading the artifact again. Persistence and retirement follow at lines 335–356, and CommitStartClaim at line 412 precedes the next evidence acquisition at line 434. A transcript changed after initial acquisition can therefore cause source retirement and lifecycle writes before selection refuses, contrary to the plan's pre-claim recheck and the Spec's refusal-before-effects contract (ARCH-ORDER). Revalidate the original selection before persistence/retirement and before claim admission, retaining the later checks. Add a controllable mutation regression asserting refusal before retirement or claim.
  - id: new
    severity: Critical
    family: recovery-budget-charges-attempted-io
    title: |
      Oversized screens consume native scan budget despite never being read
    detail: |
      cmd/internal/couchcore/recovery_evidence_os.go:255–261 checks only the 256 MiB aggregate limit before reserving the entire raw capture. Replay independently rejects raw files above 64 MiB before opening them (cmd/internal/scrollbackcmd/bounded.go:14,55). Thus a 200 MiB screen charges roughly 200 MiB without reading its raw bytes, leaving insufficient budget for an otherwise valid 65 MiB native inventory; retry carries that charge forward and candidates remain unavailable. This violates the plan's oversized-screen preflight and remaining-budget contract (ARCH-CONSTRAINTS, ARCH-PURPOSE). Share the replay bounds at preflight and charge only reads that can occur; test captures between 64 and 256 MiB with native evidence that fits the actual remaining budget.
  - id: new
    severity: Critical
    family: recovery-retains-independent-observations
    title: |
      Early screen failures discard independently readable activity
    detail: |
      cmd/internal/couchcore/recovery_evidence_os.go:228–241 returns on archive validation, missing events, or retained identity mismatch before assigning out.Capture and out.Screen at lines 252–253. Consequently a readable raw file with a missing events file, or a retained post-reboot capture rejected for changed device identity, produces no activity despite available descriptive metadata. This contradicts the Spec's readable-activity and partial-observation requirements (ARCH-PURPOSE). Observe and retain explicitly sourced descriptive activity independently of capture authorization, while preserving strict identity refusal. Add OS-reader regressions asserting unavailable-screen reason plus retained activity for these failures.
  - id: new
    severity: Important
    family: user-surface-documentation
    title: |
      README update appears missing for native-only candidate selection
    detail: |
      README.md:531–534 still describes resume-candidates as listing the best choice and alternatives with TTY-tail evidence. The range changes that user-facing command to offer unconfirmed native-only choices without tail scoring and introduces a five-minute selection lifetime, but updates only atlas pages. Update the README command description and recovery usage passage with the source distinction, explicit selection, and expiry/refresh behavior required by the docs gate.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"72b02fcc-f55a-42be-97ed-104b4efefab5","repository":"github.com/xianxu/pair","issue_id":"000721","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/34724b7e71d0d65b","reviewed_head":"0d8f276deb35fa5f2fa201cad88ab884644195c6","slot":"pair:3","requested_at":"2026-10-20T07:25:57.510744Z"},"phase":"finished","pid":97106,"started_at":"2026-10-20T07:25:57.735707Z","finished_at":"2026-10-20T07:29:18.452112Z","session_id":"01a15833-e457-7780-96bd-9d7334c0d38d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.742632125s
head: "0d8f276deb35fa5f2fa201cad88ab884644195c6"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
