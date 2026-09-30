# Boundary Review — pair#355 (milestone M1)

| field | value |
|-------|-------|
| issue | 355 — Allocate Couch session identities and enforce repository families |
| repo | pair |
| issue file | workshop/issues/000355-couch-session-identities.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | bd55cd41d429b8118774d08eb42fe9c6c92230f4..b9e55fc27fb4bfac3bcc68801cdf8cb32aaf1186 |
| command | sdlc milestone-close --issue 355 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-09-30T10:37:08-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1’s allocator, explicit name handoff, ownership classifier, and documentation are well developed. Two lifecycle gaps block shipping: cold launch still treats conversation ownership as terminal-incarnation authority, and warm attach can use ownership evidence made stale during blocking preparation.

1. **Strengths**

   - Allocation commits host high-water marks before local counters, with real subprocess contention and interrupted-publication tests.
   - Managed launches require the version flag and structured intent; exact-name assignment refuses silent naming fallbacks.
   - Ownership checks use complete artifact paths and bracket pane observations with server-generation checks.
   - README and atlas updates cover allocation, compatibility, restore limitations, and ownership.

2. **Critical findings**

   - **Terminal-incarnation authority is incomplete.** [launch_existing.go:463](/Users/xianxu/workspace/pair/cmd/internal/couchcore/launch_existing.go:463) allocates a new M whenever `warm` is false. A live but attached session fails the detached test and reaches this path. The launcher then filters its live-session view to the newly allocated name at [createflow.go:427](/Users/xianxu/workspace/pair/cmd/internal/launcher/createflow.go:427), allowing another agent for the same conversation. Registration can meanwhile succeed against the old indexed terminal; failure cleanup also resolves that old index and can delete the surviving terminal. Require proven absence before cold creation, and bind registration and cleanup to the proposed terminal incarnation. **ARCH-ORDER, ARCH-PURPOSE.**

   - **Warm ownership proof can become stale before attachment.** [createflow.go:397](/Users/xianxu/workspace/pair/cmd/internal/launcher/createflow.go:397) revalidates before calling `runAttach`, but [lifecycle.go:43](/Users/xianxu/workspace/pair/cmd/internal/launcher/lifecycle.go:43) subsequently acquires retention and performs blocking preparation, including synchronous `CmuxRename`. If the named server is replaced during that interval, line 110 attaches the replacement without revalidation. Carry the original generation proof through preparation and check it immediately before `AttachSession`. **ARCH-ORDER, ARCH-SECURE.**

3. **Important findings:** None separate from the blocking findings.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed allocator, durablefile, launcher, threadrecord, checkpoint, and zellijpane suites; targeted Couch lifecycle/continuation/recovery tests; and `git diff --check`.

   The existing cold-against-live test passes because its runner never executes the launcher’s new-name behavior. Add a composed regression asserting no second terminal, no false registration, and no deletion of the original terminal. Extend generation-change coverage beyond the layout probe to retention or cmux preparation. Live conformance was not rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared allocator, owner classifier, and durable publication.
   - **ARCH-PURE — pass:** allocation and binding transitions have pure cores; M1 entity locations match the plan.
   - **ARCH-PURPOSE — flag:** registration and cleanup still retain the old single-terminal assumption.
   - **ARCH-MOCK — pass:** injected persistent storage and stateful ownership seams exist; composed coverage needs the additions above.
   - **ARCH-CONSTRAINTS — pass:** allocation and owner probes stay off passive refresh paths and have declared bounds.
   - **ARCH-SECURE — flag:** warm attach can consume stale ownership evidence.
   - **ARCH-ORDER — flag:** both findings concern authority across external events.
   - **ARCH-FUNERAL — pass:** bounded host reservations and existing binding retention provide explicit lifecycles.

7. **Plan revision recommendations**

   Append a `## Revisions` entry requiring exact proposed-binding authority across cold admission, registration, and cleanup, plus generation revalidation after attach preparation. Name the two regression sequences above. M2 family work remains outside this boundary.

```findings
findings:
  - id: new
    severity: Critical
    family: terminal-binding-authority
    title: |
      Cold launch, registration, and cleanup do not consistently use the terminal incarnation
    detail: |
      launch_existing.go:463 allocates a new M for an attached-live conversation, while createflow.go:427 filters out its existing terminal, permitting a duplicate agent. Registration and cleanup still resolve the old address index; require proven absence before creation and use the proposed binding for registration and teardown, with composed regressions (ARCH-ORDER, ARCH-PURPOSE).
  - id: new
    severity: Critical
    family: owner-proof-at-effect-boundary
    title: |
      Warm attach revalidates ownership before blocking preparation rather than at handoff
    detail: |
      createflow.go:397 checks the generation before lifecycle.go:43-110 performs retention and cmux preparation. Replacement during that interval reaches AttachSession unchecked; retain the original proof and revalidate immediately before attachment, testing replacement during preparation (ARCH-ORDER, ARCH-SECURE).
```
