# Boundary Review — pair#526 (whole-issue close)

| field | value |
|-------|-------|
| issue | 526 — broadcast-web: from submission to issue: a privacy flag per bundle (origin policy) and the couch skill's rule for writing notes up as issues |
| repo | pair |
| issue file | workshop/issues/000526-broadcast-web-from-submission-to-issue-a-privacy-flag-per-bundle-origin-policy-and-the-couch-skill-s-rule-for-writing-notes-up-as-issues.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9f4542b83d47ed88f0ce4266425d49e1bb9fcd16..4a1558530da4e19a6811e33144aa9af56e0b7d0f |
| command | sdlc close --issue 526 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T09:23:51-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation delivers the main privacy flow and updates both documentation surfaces. One blocking parser bug remains: ambiguous JSON keys can override an explicit private rule and grant sharing. This was reproduced against the pinned source; the checkout remains unchanged.

1. **Strengths**
   - Policy parsing and matching are separated from filesystem access, with bounded reads and safe error messages.
   - Privacy uses capture-bound identity, and recorder tests exercise identity changes during capture.
   - Store tests verify that privacy survives reopening and delivery, and reject retries that change permission.
   - Editable-subtree tests cover ancestor accessibility names, shadow content, and preservation of static labels. README and atlas explain the new configuration and evidence rules.

2. **Critical findings**
   - **Ambiguous JSON can grant sharing — ARCH-SECURE.** At [privacy.go:237](/Users/xianxu/Library/Caches/sdlc/review-checkouts/b4d053494b14be91/cmd/internal/submission/privacy.go:237), duplicate detection compares keys case-sensitively, while the subsequent Go struct decoder matches them case-insensitively. This policy is accepted and produces `shareable`:
     ```json
     {"origins":[{"origin":"https://example.test","privacy":"private","Privacy":"shareable"}]}
     ```
     A second reproduction using conflicting top-level `origins` and `Origins` also grants sharing. This violates the explicit rejection of ambiguous permissions and invalid-policy private fallback. Enforce exact schema keys before decoding, or reject every alias the decoder treats as the same field. Sweep policy envelopes, origin rules, and persisted rule evidence; add regression tests proving rejection and private fallback with a notice.

3. **Important findings:** None separately.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Full submission and recorder package tests passed.
   - Focused webcast privacy/origin/capture tests passed.
   - Both adversarial parser reproductions returned `accepted: shareable`; existing tests miss this ambiguity.
   - Couch command tests could not compile: `cmd/internal/runtimebundle/embed.go:8` requires missing generated `assets/runtime/files`. Full-suite and live Chrome results were not independently verified.
   - Pinned stat/name-status inspection and `git diff --check` completed successfully.

6. **Architectural notes**
   - **ARCH-DRY — pass:** centralized origin matching and shared reading-guide forwarding.
   - **ARCH-PURE — pass:** policy decisions are pure; filesystem and capture operations remain adapters.
   - **ARCH-PURPOSE — pass:** producer metadata and agent evidence rules are both implemented; the parser defect blocks reliable enforcement.
   - **ARCH-MOCK — pass:** injected capture/policy seams, stateful recorder fixtures, temporary storage, and a live Chrome conformance test exist.
   - **ARCH-CONSTRAINTS — pass:** policy bytes and rule count are bounded; no new background worker.
   - **ARCH-SECURE — flag:** conflicting field aliases bypass strict parsing.
   - **ARCH-ORDER — pass:** existing capture fences and delivery transitions retain ownership; persisted decisions remain frozen.
   - **ARCH-FUNERAL — pass:** no new per-event artifact family; configuration ownership and existing bundle capacity/cleanup are documented.

7. **Plan revision recommendations**
   - Add a `## Revisions` entry specifying exact-key validation across the policy envelope, rules, and persisted evidence, with regressions for conflicting case variants and successful private submission after rejection.

```findings
findings:
  - id: new
    severity: Critical
    family: ambiguous-permission-decoding
    title: |
      Case-variant JSON keys can override private policy and grant sharing
    detail: |
      cmd/internal/submission/privacy.go:237 checks duplicate keys case-sensitively, but the struct decoder at line 257 matches fields case-insensitively. Reproduced conflicting privacy/Privacy and origins/Origins inputs that return shareable instead of invalid-policy private fallback (ARCH-SECURE). Enforce exact schema keys or reject decoder-equivalent aliases across policy and rule evidence, and add regression coverage for rejection, private fallback, and the notice.
```
