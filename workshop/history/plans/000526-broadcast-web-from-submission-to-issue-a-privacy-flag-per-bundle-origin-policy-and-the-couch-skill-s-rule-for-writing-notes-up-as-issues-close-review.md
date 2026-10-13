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

---

## Re-review — 2026-10-13T09:37:18-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 526 — broadcast-web: from submission to issue: a privacy flag per bundle (origin policy) and the couch skill's rule for writing notes up as issues |
| repo | pair |
| issue file | workshop/issues/000526-broadcast-web-from-submission-to-issue-a-privacy-flag-per-bundle-origin-policy-and-the-couch-skill-s-rule-for-writing-notes-up-as-issues.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9f4542b83d47ed88f0ce4266425d49e1bb9fcd16..5aa4b1ec16fd7358aff0bd31e133634cc720ccd6 |
| command | sdlc close --issue 526 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T09:37:18-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The policy matcher, private fallback, capture fencing, and documentation are well integrated. BR-1’s regressions genuinely detect the implemented fix, but the same ambiguity remains in persisted manifests: conflicting case-variant keys can turn a private bundle into shareable evidence. This blocks shipping.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Policy envelopes and nested rule evidence are fixed; removing the exact-key guard makes their regression tests fail. However, cmd/internal/submission/store.go:219 still decodes top-level permission fields case-insensitively. Appending Privacy:"shareable" and Privacy_rule containing a matching shareable rule to an explicitly private manifest makes readMetadata return a validated shareable bundle. Complete the ambiguous-permission-decoding family sweep across all permission-bearing JSON boundaries, including persisted manifest fields, and add a store-reload regression (ARCH-SECURE, ARCH-PURPOSE).
```

1. **Strengths**
   - Policy parsing and matching remain pure; file loading is injected into submission preparation.
   - Bounded file reads reject nonregular files and preserve submission through explicit private fallback.
   - Capture origin participates in the existing identity fence; navigation regression coverage exercises that fence.
   - README, atlas, shipped skill, and reading guide document the new contract.

2. **Critical findings**
   - **BR-1 remains open — `cmd/internal/submission/store.go:219`, `cmd/internal/submission/privacy.go:70`.** The manifest decoder accepts conflicting `privacy`/`Privacy` and `privacy_rule`/`Privacy_rule` fields. A disposable store-level test reproduced successful reload as `shareable`. Reject duplicate and noncanonical permission keys before decoding loses that information. Cover the policy envelope, nested rules, and manifest permission fields as one enumerated class.

3. **Important findings**
   - None additional.

4. **Minor findings**
   - None.

5. **Test coverage notes**
   - Existing targeted privacy, origin, submission, capture, collector, and guide tests passed after generating required runtime assets in the disposable copy.
   - Removing the exact-key guard made policy, evidence, and fallback/notice regressions fail.
   - The added manifest-reload regression fails at pinned head.
   - Live Chrome validation failed during startup because its GPU process crashed; conformance was not independently confirmed.
   - Disposable files were removed; the checkout remains unchanged.

6. **Architectural notes**
   - **ARCH-DRY: pass** — shared origin matcher and existing delivery machinery reused.
   - **ARCH-PURE: pass** — policy logic separated from file and capture IO.
   - **ARCH-PURPOSE: flag** — BR-1’s permission-decoding class remains incompletely swept.
   - **ARCH-MOCK: pass structurally** — stateful capture seams and Chrome conformance fixture exist; live execution was blocked as noted.
   - **ARCH-CONSTRAINTS: pass** — bounded policy size/rule count and existing capture limits.
   - **ARCH-SECURE: flag** — persisted manifest ambiguity can grant sharing.
   - **ARCH-ORDER: pass** — existing identity fences and delivery transitions preserve frozen decisions.
   - **ARCH-FUNERAL: pass** — metadata follows existing bounded bundle storage and documented cleanup ownership.

7. **Plan revision recommendations**
   - Add a `## Revisions` entry enumerating every permission-bearing JSON boundary and requiring exact keys plus duplicate rejection at each.
   - Include conflicting top-level manifest fields in the store-reload verification strategy.

---

## Re-review — 2026-10-13T09:57:33-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 526 — broadcast-web: from submission to issue: a privacy flag per bundle (origin policy) and the couch skill's rule for writing notes up as issues |
| repo | pair |
| issue file | workshop/issues/000526-broadcast-web-from-submission-to-issue-a-privacy-flag-per-bundle-origin-policy-and-the-couch-skill-s-rule-for-writing-notes-up-as-issues.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9f4542b83d47ed88f0ce4266425d49e1bb9fcd16..f5fab033a4398712a5e60c24d634368b0de888f1 |
| command | sdlc close --issue 526 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T09:57:33-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

BR-1 is addressed across policy parsing, nested evidence, and persisted manifests. Regression tests demonstrably fail without the guards and pass with them. The affected package suites pass, and the implementation matches the revised privacy contract. Confidence is limited by live Chrome failing during GPU startup, before exercising the conformance fixture.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      cmd/internal/submission/privacy.go:261 enforces exact policy/evidence keys; :279 rejects ambiguous manifest permissions, including store reload. Removing these guards in a disposable pinned-head tree made TestOriginPrivacyRejectsCaseAliases, TestLoadOriginPolicy, and TestPrivacyManifestRejectsAmbiguousKeysOnReload fail with sharing escalation or missing fallback/notice. Restoring them passed all affected package suites. The permission-boundary enumeration is recorded in the plan's Revisions.
```

1. **Strengths**

   - Pure policy decisions remain separate from the bounded filesystem adapter; invalid files produce private evidence and an actionable notice.
   - Captured origin participates in existing identity fences, with controlled navigation tests.
   - Store tests verify frozen permissions across reopening and delivery, and reject permission-changing retries.
   - README, atlas, shipped skill, and reading guide cover operator ownership, private summaries, crop authorization, and local recording lifetime.

2. **Critical findings:** None remaining.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Required pinned stat/name-status inspection and patch review completed; `git diff --check` passed.
   - Full suites passed for `submission`, `couchcmd`, `webcast`, and `webcast/recorder`.
   - Mutation checks independently established BR-1’s regression sensitivity.
   - The initial missing runtime assets were resolved using the repository’s generator inside a disposable copy.
   - Live `TestLiveSubmissionCrop` failed because Chrome’s GPU process crashed during startup. Actual browser conformance was not independently confirmed.
   - The repository-wide suite was not rerun. Disposable files were removed; tracked checkout files remain unchanged.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** shared origin normalization and JSON-key scanning; existing delivery machinery reused.
   - **ARCH-PURE — pass:** parsing, matching, and validation remain deterministic; IO stays in adapters.
   - **ARCH-PURPOSE — pass:** producer metadata and agent externalization rules are delivered; BR-1’s boundary sweep includes persisted manifests.
   - **ARCH-MOCK — pass structurally:** injected policy/capture seams, stateful recorder doubles, temporary storage, and a live conformance fixture exist. Live execution has the limitation above.
   - **ARCH-CONSTRAINTS — pass:** bounded policy bytes and rule count, regular-file checks, and existing capture limits.
   - **ARCH-SECURE — pass:** ambiguous permission keys are rejected before decoding can overwrite authority; errors preserve private fallback.
   - **ARCH-ORDER — pass:** capture fences and existing delivery transitions retain ownership; retries preserve frozen decisions.
   - **ARCH-FUNERAL — pass:** metadata follows existing bundle capacity and cleanup ownership; no new per-event artifact family.

7. **Plan revision recommendations:** None required. Preserve the documented #525 requirement to use the same capture-bound identity when integrating the full page URL.
