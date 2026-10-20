# Boundary Review — pair#742 (whole-issue close)

| field | value |
|-------|-------|
| issue | 742 — Couch interrupt refuses a working Codex turn |
| repo | pair |
| issue file | workshop/issues/000742-codex-working-interrupt.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9c926136d44af5bad734eaef7986e327a06a1965..a16907dd04af7aa327515f3778a85ed7459ffb1b |
| command | sdlc close --issue 742 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T13:35:44-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation passes focused tests and race checks, and the new regressions fail against the base behavior. No executable correctness defect was established. The blocking issue is stale acceptance criteria: the revised Codex contract intentionally overrides cached stop grace, while “Done when” still requires unconditional refusal.

Done-when: stale — the stop-grace refusal clause does not distinguish Claude’s lifecycle requirement from Codex’s current native work proof.

```findings
findings:
  - id: new
    severity: Important
    family: acceptance-criteria-match-contract
    title: |
      Update the stop-grace acceptance criterion for per-agent work proof
    detail: |
      workshop/issues/000742-codex-working-interrupt.md:50 requires stop grace to refuse without a write, but the revised Spec at lines 30–32 and cmd/internal/wrapcmd/agent_drive.go:73–83 authorize current recognized Codex work despite cached stop grace. The sole instance in this window is this unconditional Done-when clause. Qualify refusal by agent and current evidence, and explicitly exercise Codex working evidence during cached grace alongside the existing Claude grace-refusal test.
  - id: new
    severity: Minor
    family: plan-completion-matches-evidence
    title: |
      The completed capture item overstates the native Waiting evidence
    detail: |
      workshop/issues/000742-codex-working-interrupt.md:55 marks native Working and background-wait capture complete, although its Revisions and testdata/drive/codex/0.163.0/metadata.json:8 explicitly identify Waiting fixtures as synthetic. This is the sole completed-plan evidence overstatement in this window. Describe captured Working/running-shell evidence and reported Waiting wording with synthetic fixtures separately.
```

1. **Strengths**

   - Lifecycle observation and interrupt admission share `RecognizeCodexWorking`.
   - Production-output tests exercise the actual mailbox/writer, one Ctrl+C, fresh idle confirmation and second-interrupt refusal.
   - README and atlas explain the changed work proof and confirmation requirement.

2. **Critical findings:** None.

3. **Important findings:** Revise the stop-grace acceptance criterion and cover both agents’ intended grace behavior, as detailed above.

4. **Minor findings:** Correct the completed capture item’s description of synthetic Waiting evidence.

5. **Test coverage notes**

   Focused interrupt/Codex tests passed in both packages, including under `-race`. Temporary Go overlays restoring base behavior made the production-output regressions and unknown-screen confirmation regression fail; the checkout remained unchanged. Recorded local and hub live checks remain pending in the pinned tracker.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Both work-proof consumers use the shared recognizer.
   - **ARCH-PURE: pass.** Recognition and reduction remain deterministic; IO stays in adapters.
   - **ARCH-PURPOSE: pass for implementation.** The change addresses native status recognition, stale lifecycle evidence and positive completion.
   - **ARCH-EVOLVE: pass.** No persisted runtime format changes.

7. **Plan revision recommendations**

   Add a `## Revisions` entry distinguishing Claude grace refusal from Codex’s current-work override, and update “Done when” accordingly. Correct the capture checklist wording to match the documented native/synthetic evidence.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"45a3559d-2d42-45f0-a44b-bac679c1b3fa","repository":"github.com/xianxu/pair","issue_id":"000742","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/5369351013e3462e","reviewed_head":"a16907dd04af7aa327515f3778a85ed7459ffb1b","slot":"pair:2","requested_at":"2026-10-20T20:33:24.151072Z"},"phase":"finished","pid":56505,"started_at":"2026-10-20T20:33:24.26066Z","finished_at":"2026-10-20T20:35:43.863355Z","session_id":"01a15b04-d05b-7ce2-ba97-aa5e60a99725","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.95535975s
head: "a16907dd04af7aa327515f3778a85ed7459ffb1b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411
