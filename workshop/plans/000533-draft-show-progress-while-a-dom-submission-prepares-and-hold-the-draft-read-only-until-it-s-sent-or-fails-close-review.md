# Boundary Review — pair#533 (whole-issue close)

| field | value |
|-------|-------|
| issue | 533 — draft: show progress while a DOM submission prepares, and hold the draft read-only until it's sent or fails |
| repo | pair |
| issue file | workshop/issues/000533-draft-show-progress-while-a-dom-submission-prepares-and-hold-the-draft-read-only-until-it-s-sent-or-fails.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7efc4f904ece07c6fa2b554e9002e16dafde11df..c4f2b8f13c013c08f1f88d256dd91c6b7d443ef3 |
| command | sdlc close --issue 533 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T14:47:40-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Progress propagation, deadline handling and visible UI feedback are well covered, but the new read-only lock permits navigation to partially mutate draft state. I reproduced this overwriting an unrelated queued item after submission failure. That blocks shipping.

1. **Strengths**

   - Backend progress uses immutable snapshots, mutex protection and job-identity fencing.
   - The reducer preserves active operation ownership and separates late delivery receipts from draft clearing.
   - Embedded Neovim tests verify rendered progress before held external work; subprocess tests exercise actual timeout enforcement.
   - README and atlas document the changed behavior.

2. **Critical findings**

   **Read-only ownership does not guard draft navigation — ARCH-ORDER, ARCH-PURPOSE.**

   The lock in [nvim/init.lua:1666](/Users/xianxu/Library/Caches/sdlc/review-checkouts/887ec09011d723e1/nvim/init.lua:1666) rejects buffer writes, but `go_to` changes `nav.pos` before attempting that write (`nvim/init.lua:2527`). The resulting exception leaves the logical slot pointing at content that was never loaded.

   Reproduced against the pinned head with private files:

   - Draft contains `draft`; queue item contains `queued`.
   - Start DOM submission and hold preparation.
   - Press Alt+Right: navigation raises “Buffer is not 'modifiable'”.
   - Return preparation failure, then press Alt+Left.
   - `autosave_current_slot` writes `draft` over the queue item (`nvim/init.lua:2439`).

   Guard actions affecting the owned draft **before** changing navigation state or files. Sweep navigation, queueing, deletion, reset and send/clear actions for the same partial-mutation pattern. Add production integration tests asserting slot identity and persisted contents through failure and timeout.

3. **Important findings**

   None separately.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed: Lua reducer/adapter/transport tests; sender tests under Neovim; DOM headless and embedded UI suite; Zellij timeout/redaction tests; Go submission, couchmessage, recorder and webcast packages.

   The checkout lacked generated runtime assets. After generating them in a disposable copy, couchcmd passed; the combined couchcmd/couchtty command reached the review’s 180-second limit before couchtty reported a result. No couchtty or full-suite pass is claimed. The disposable copy was removed.

   Existing lock tests reject direct buffer edits but omit navigation during an active operation—the reproduced failure class. No prior review findings required disposition.

6. **Architectural notes**

   - **ARCH-DRY — pass:** existing polling, reducer and sender seams are reused.
   - **ARCH-PURE — pass:** transition logic remains value-only; IO stays in adapters. Core-concept locations and classifications match the code.
   - **ARCH-PURPOSE — flag:** preserving authored state requires guarding existing draft actions, beyond rejecting text edits.
   - **ARCH-MOCK — pass:** controlled cast/CDP and pane fixtures share production seams.
   - **ARCH-CONSTRAINTS — pass:** remaining-time budgets reach DOM subprocesses; blocked-child tests verify enforcement.
   - **ARCH-SECURE — pass:** new progress formatting validates counts and escapes statusline text.
   - **ARCH-ORDER — flag:** submission ownership does not govern navigation’s state/file mutations.
   - **ARCH-FUNERAL — pass:** new progress is volatile; operation timers have retirement paths.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining the policy for navigation and draft-mutating actions during active submission, enumerating affected handlers and requiring persisted-content regression tests. Keep final verification pending until its actual result is recorded.

```findings
findings:
  - id: new
    severity: Critical
    family: operation-ownership-must-guard-mutators
    title: |
      Read-only DOM preparation permits partial navigation that overwrites queued text
    detail: |
      nvim/init.lua:1666 applies nomodifiable without guarding go_to, which changes nav.pos at line 2527 before its buffer write fails. Reproduced: hold preparation with draft="draft" and queue="queued", press Alt+Right, return preparation failure, then press Alt+Left; autosave_current_slot at line 2439 overwrites the queue with "draft". ARCH-ORDER and ARCH-PURPOSE: guard owned-draft actions before state or file mutations, sweep navigation/queue/delete/reset/send-clear siblings, and add production regression tests preserving slot identity and stored contents through failure and timeout.
```
