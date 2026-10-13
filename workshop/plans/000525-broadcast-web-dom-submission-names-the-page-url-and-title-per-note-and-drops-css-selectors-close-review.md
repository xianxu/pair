# Boundary Review — pair#525 (whole-issue close)

| field | value |
|-------|-------|
| issue | 525 — broadcast-web: DOM submission names the page (URL and title per note) and drops CSS selectors |
| repo | pair |
| issue file | workshop/issues/000525-broadcast-web-dom-submission-names-the-page-url-and-title-per-note-and-drops-css-selectors.md |
| boundary | whole-issue close |
| milestone | — |
| window | 844d29e698e481712f6a58f37f04e4d717a405f3..ef0693bc8d4c0e04291f65529fe31c699ef56e43 |
| command | sdlc close --issue 525 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T10:58:01-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The implementation matches the documented schema, captured-page binding, selector removal, and compatibility design. No correctness defect was found in the inspected code. Verification remains blocked: Couch tests could not compile without generated runtime assets, and live Chrome crashed before exercising the fixture. REWORK reflects incomplete verification, not a demonstrated implementation bug.

1. **Strengths**
   - Page identity and privacy derive from the same captured URL, with checks around text and screenshot operations.
   - Submission projection clears selectors without mutating recorded anchors; regression coverage preserves `@N`.
   - Collector tests cover ordered text nodes, escaped classes, shadow/slot ownership, clipping, and byte limits.
   - Version-1 compatibility and immutable retries are tested. README, atlas, and shipped guidance describe version 2 consistently.

2. **Critical findings**

   None.

3. **Important findings**

   **Required verification could not complete** — `workshop/plans/000525-captured-page-plan.md:90–91`, `cmd/internal/webcast/live_submission_test.go:50`, `cmd/internal/runtimebundle/embed.go:8`.

   The affected-package command failed to compile `couchcmd`: `assets/runtime/files` was missing. The prescribed live test failed during Chrome startup with “GPU process isn't usable,” before its assertions ran. Restore the normal generated-asset prerequisites and a working headless Chrome environment, then rerun these checks and the boundary review. **ARCH-MOCK:** live conformance remains unverified in this review.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Independently passed all tests in `submission`, `webcast/submission`, `webcast/recorder`, and `webcast`; `git diff --check` also passed. Coverage exercises page mutation at text/pixel fences, invalid metadata, source references, legacy loading, and frozen retries. No prior findings required disposition. Repository files remain unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared page authority and existing privacy matcher.
   - **ARCH-PURE — pass:** schema/projection logic remains separate from CDP and persistence.
   - **ARCH-PURPOSE — pass:** identity, both text arrays, anchors, privacy, compatibility, and documentation are covered.
   - **ARCH-MOCK — flag:** stateful fixtures pass; live conformance is blocked as detailed above.
   - **ARCH-CONSTRAINTS — pass:** bounded metadata, collector output, operation deadlines, and bundle admission.
   - **ARCH-SECURE — pass:** authoritative JSON keys, page validation, and private unavailable captures.
   - **ARCH-ORDER — pass:** existing recorder ownership and capture fences reject changed identities; retries preserve frozen evidence.
   - **ARCH-FUNERAL — pass:** metadata uses the existing capacity-limited bundle store; no new artifact family.

   Core-concept entities exist at their stated locations and match their classifications.

7. **Plan revision recommendations**

   Record this review’s verification blockers and subsequent rerun results. No implementation-design revision is indicated.

```findings
findings:
  - id: new
    severity: Important
    family: required-verification-evidence
    title: |
      Required package and live-conformance verification could not complete
    detail: |
      Plan lines 90–91 require affected-package and real-Chrome verification. couchcmd compilation failed at cmd/internal/runtimebundle/embed.go:8 because generated assets/runtime/files is absent; TestLiveSubmissionCrop failed at live_submission_test.go:50 because Chrome's GPU process crashed before opening the fixture. Restore verification prerequisites and rerun both checks and this review; ARCH-MOCK live conformance remains unverified.
```

---

## Re-review — 2026-10-13T11:00:43-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 525 — broadcast-web: DOM submission names the page (URL and title per note) and drops CSS selectors |
| repo | pair |
| issue file | workshop/issues/000525-broadcast-web-dom-submission-names-the-page-url-and-title-per-note-and-drops-css-selectors.md |
| boundary | whole-issue close |
| milestone | — |
| window | 844d29e698e481712f6a58f37f04e4d717a405f3..4684915e164ce67a4a70c8ff108091f1dc46cead |
| command | sdlc close --issue 525 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T11:00:43-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The implementation matches the captured-page schema, selector removal, privacy binding, and compatibility plan. Package and race checks pass. BR-1 remains partially unresolved: generated assets fixed package verification, but two independent live Chrome attempts crashed before exercising the fixture. This is a verification blocker, not a demonstrated code defect.

1. **Strengths**

   - Capture fences bind URL/title to text and pixels; privacy derives from that same URL.
   - Selector removal preserves recording handles and leaves original recorded anchors unchanged.
   - Collector tests cover ordering, mixed text, shadow/slot ownership, escaped classes, and size limits.
   - Legacy loading and frozen retries have coverage. README, atlas, and shipped guidance describe v2 consistently.

2. **Critical findings**

   None.

3. **Important findings**

   **BR-1 — not-addressed, narrowed to live conformance.** At `cmd/internal/webcast/live_submission_test.go:50`, both prescribed runs failed during Chrome startup with `GPU process isn't usable`, before opening the fixture.

   The dispute correctly identifies `make runtimebundle-generate` as the package prerequisite: generation and subsequent Couch checks passed here. However, the reported successful live runs could not be reproduced. Restore a working Chrome environment and rerun the unchanged conformance command and review. **ARCH-MOCK** remains unverified at the real-browser boundary.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Independently passed:

   - Full tests for `submission`, `webcast/submission`, `webcast/recorder`, and `webcast`.
   - Targeted checks across those packages plus `couchcmd` and `couchtty`.
   - Targeted race checks for schema, projection, capture, and privacy.
   - `git diff --check`.

   Live Chrome failed twice without reaching assertions. No production changes were made; tracked checkout status remains clean.

6. **Architectural notes**

   - **ARCH-DRY — pass:** one captured-page authority feeds references and privacy.
   - **ARCH-PURE — pass:** schema/projection remain separate from CDP and persistence.
   - **ARCH-PURPOSE — pass:** source identity, anchors, both text arrays, compatibility, and documentation are covered.
   - **ARCH-MOCK — flag:** stateful fixtures pass; live conformance remains blocked under BR-1.
   - **ARCH-CONSTRAINTS — pass:** metadata/output bounds and operation deadlines remain enforced.
   - **ARCH-SECURE — pass:** authoritative keys and page metadata are validated; unavailable capture stays private.
   - **ARCH-ORDER — pass:** existing recorder ownership and fences reject identity changes; retries retain frozen evidence.
   - **ARCH-FUNERAL — pass:** added metadata uses the existing capacity-limited submission store.

   Core-concept entities exist at the documented paths and match their classifications.

7. **Plan revision recommendations**

   Add a `## Revisions` entry recording that fresh-checkout package verification now passes after generation, while this review’s two live attempts failed during Chrome startup. Record the successful conformance rerun when available; no implementation-design revision is indicated.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Package verification is resolved: make runtimebundle-generate and targeted Couch checks passed. Live conformance remains unresolved: two prescribed TestLiveSubmissionCrop runs failed at live_submission_test.go:50 with “GPU process isn't usable” before opening the fixture. This supports the environmental diagnosis but does not reproduce the claimed live PASS; restore Chrome verification and rerun. No new finding is raised.
```
