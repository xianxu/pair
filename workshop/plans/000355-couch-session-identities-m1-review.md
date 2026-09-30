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

---

## Re-review — 2026-09-30T10:59:00-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 355 — Allocate Couch session identities and enforce repository families |
| repo | pair |
| issue file | workshop/issues/000355-couch-session-identities.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | bd55cd41d429b8118774d08eb42fe9c6c92230f4..bce1bcd3884d4b58fc8252ba1546abd35fe48e80 |
| command | sdlc milestone-close --issue 355 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-09-30T10:59:00-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Both prior findings are addressed with meaningful regression tests. M1’s allocation, terminal-binding, and ownership changes are otherwise coherent, but the new repository-token formatter rejects previously supported repository names, blocking new conversations.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Cold admission requires absence; registration and cleanup select the pending/current terminal binding. Passing terminal_incarnation_test.go regressions cover attached-terminal refusal, stale registration, wrong-terminal cleanup, and interrupted recovery; launcher tests cover late terminal appearance.
  - id: BR-2
    disposition: addressed
    note: |
      lifecycle.go:134-145 revalidates the original ownership proof immediately before AttachSession, after blocking preparation. TestCouchSessionWarmGenerationRevalidatedAtAttachEffect injects replacement during retention/cmux through both entrypoints and asserts refusal and poller cleanup.
findings:
  - id: new
    severity: Critical
    family: descriptive-label-must-not-gate-identity
    title: |
      Unicode-only repository names now prevent conversation creation
    detail: |
      cmd/internal/couchidentity/identity.go:41-60 discards every non-ASCII character and rejects the resulting empty token. Both couch.go:488 and slotrecovery.go:320 pass the repository basename, so supported names such as 项目 now fail allocation. Use a deterministic safe fallback when normalization yields nothing; C/N already provide uniqueness. Add pure formatter tests and composed new/fresh launch regressions for Unicode-only and punctuation-only names. ARCH-PURPOSE.
```

1. **Strengths**

   - Host-first counter publication and recovery floors prevent identity reuse after interrupted writes or local rollback.
   - Registration and teardown now follow terminal-incarnation authority rather than a stale compatibility index.
   - Warm attachment preserves and revalidates its original server proof.
   - README and atlas document allocation, migration, and restore limitations.

2. **Critical findings**

   - [identity.go:57](/Users/xianxu/workspace/pair/cmd/internal/couchidentity/identity.go:57): descriptive repository text must not prevent identity allocation. Existing workspace tests explicitly accept `项目:1`; the previous generated tags did not depend on repository spelling. Apply the fallback at the shared formatter so both creation paths inherit it.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**

   Passed allocator, durablefile, threadrecord, checkpoint, zellijpane, focused artifact-owner and launcher tests, focused terminal-authority regressions, and broader core identity/ownership/continuation/recovery tests. Diff whitespace checks passed.

   Current tests miss repository names that normalize to empty. Live conformance and old-binary skew checks were not rerun. The full artifactpath suite reports inventory failures involving unchanged review-editor sources.

6. **Architecture**

   - **ARCH-DRY — pass:** shared allocation and binding-selection authorities.
   - **ARCH-PURE — pass:** allocation and transition logic remain independently testable.
   - **ARCH-PURPOSE — flag:** descriptive labels impose an unintended repository restriction.
   - **ARCH-MOCK — pass:** stateful ownership/generation seams exercise replacement ordering.
   - **ARCH-CONSTRAINTS — pass:** bounded authority storage and ownership queries.
   - **ARCH-SECURE — pass:** strict parsing, exact address evidence, generation revalidation.
   - **ARCH-ORDER — pass:** pending/current bindings preserve uncertain outcomes.
   - **ARCH-FUNERAL — pass:** bounded permanent registrations and existing binding retention.

7. **Plan revision recommendation**

   Append a `## Revisions` entry specifying a nonempty safe fallback for descriptive repository tokens and regression coverage through both conversation-allocation entrypoints. M2 remains outside this boundary.

---

## Re-review — 2026-09-30T11:06:56-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 355 — Allocate Couch session identities and enforce repository families |
| repo | pair |
| issue file | workshop/issues/000355-couch-session-identities.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | bd55cd41d429b8118774d08eb42fe9c6c92230f4..9cc1374798433fe3fe00b66988904fed63cde90a |
| command | sdlc milestone-close --issue 355 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-09-30T11:06:56-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned M1 implementation satisfies the identity-allocation and terminal-binding scope. BR-3 is addressed with verified regression evidence; no new blocking findings emerged. Repository-family work remains explicitly scheduled for M2.

```findings
dispose:
  - id: BR-3
    disposition: addressed
    note: |
      The shared formatter supplies a safe fallback and bounds descriptive tokens. Pure formatter and composed new/fresh launch tests pass at HEAD and fail with the previous formatter substituted through a temporary Go overlay. Unicode-only, punctuation-only, and long repository names are exercised.
```

1. **Strengths**

   - Both conversation-allocation paths use the shared formatter and allocator.
   - Host-first durable counter publication preserves monotonicity across interrupted writes and local rollback.
   - Pending/current terminal bindings reach registration, recovery, cleanup, and slot admission.
   - Warm attachment revalidates the original server generation immediately before handoff.
   - README and atlas changes document naming, compatibility, ownership, and restore restrictions.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed allocator, durablefile, launcher, threadrecord, checkpoint, zellijpane, Couch command, and focused Couch core suites. BR-3 mutation checks failed at the intended formatter and launch boundaries. Diff whitespace checks passed.

   The artifact inventory test reports the same 32 failures at HEAD and in a scratch copy of the pinned base with matching generated assets; these are pre-existing. The complete Couch core suite and opt-in live Zellij conformance were not rerun during this review. Repository files remain unchanged.

6. **Architectural notes**

   | Principle | Assessment |
   |---|---|
   | ARCH-DRY | Pass — shared allocation, ownership classification, and durable publication. |
   | ARCH-PURE | Pass — formatting and transitions separate from filesystem/process effects. |
   | ARCH-PURPOSE | Pass — M1 covers independent conversation/terminal lifetimes and both allocation entry points. |
   | ARCH-MOCK | Pass — injected stateful ownership fixtures and portable storage; live conformance check exists. |
   | ARCH-CONSTRAINTS | Pass — bounded storage, registry capacity, and ownership-query deadlines. |
   | ARCH-SECURE | Pass — strict persisted-input validation and exact scoped artifact evidence. |
   | ARCH-ORDER | Pass — explicit binding transitions preserve unresolved starts and revalidate ownership at effects. |
   | ARCH-FUNERAL | Pass — bounded allocation metadata, replacement index entries, and existing binding retention. |

7. **Plan revision recommendations:** None. The appended BR-3 revision matches the implementation; M2 remains unclaimed.
