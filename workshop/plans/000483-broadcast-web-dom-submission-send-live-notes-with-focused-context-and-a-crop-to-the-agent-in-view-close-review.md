# Boundary Review — pair#483 (whole-issue close)

| field | value |
|-------|-------|
| issue | 483 — broadcast-web: DOM submission — send live notes with focused context and a crop to the agent in view |
| repo | pair |
| issue file | workshop/issues/000483-broadcast-web-dom-submission-send-live-notes-with-focused-context-and-a-crop-to-the-agent-in-view.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7f9f58a6186de2c68e0be57da569358fab3bce9e..1fb4f96ace12473186cdafdee7166dbbd9019403 |
| command | sdlc close --issue 483 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-12T21:15:36-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The implementation covers the intended submission flow, with useful tests for navigation, frozen selection, revision-guarded resolution, and acknowledgment-only recovery. Shipping is blocked by a durability retry defect and a mismatch between the promised pure state model and the Lua coordinator. Some integration verification was blocked by missing generated runtime assets.

1. **Strengths**

   - Capture fences CDP events through the recorder and checks document identity, including same-URL reloads.
   - Resolution checks note revisions at emission time, protecting reopened notes from late acknowledgment.
   - Store validation bounds PNG dimensions, bundle sizes, paths, and retained storage.
   - README and atlas document the new gesture, evidence format, retention, and recovery behavior.

2. **Critical findings**

   - **Durability retries can report success without establishing durability** — `cmd/internal/submission/store.go:467`. If rename succeeds at line 495 but directory sync fails at line 498, another `Begin` reads `delivering` and returns through the unchanged-state shortcut without retrying synchronization. The caller can consequently paste without confirmed durable intent. The existing-bundle shortcut at line 400 has the corresponding publication problem. Retry the required directory synchronization before returning success on these paths; add injectable post-rename sync failures and repeated-retry tests. **ARCH-ORDER**.

   - **The declared PURE Lua entity is an effectful coordinator without a pure transition authority** — `workshop/plans/000483-dom-submission-plan.md:36`, `nvim/dom_submission.lua:5`. The coordinator directly invokes RPC, scheduling, notification, and delivery callbacks while independently mutating `pending`, `ack_id`, and `busy`. Its tests require RPC/send doubles (`nvim/dom_submission_test.lua:4`). This contradicts the Core concepts classification and the required structural state-machine boundary. Classify the adapter as INTEGRATION and extract an explicit state/event transition function that owns authoritative state and declares effects. **ARCH-PURE, ARCH-ORDER**.

3. **Important findings**

   - **Agent-pane discovery duplicates an existing implementation** — `nvim/dom_transport.lua:25` repeats recursive pane collection and the `pair wrap` identity predicate from `nvim/pair_poke.lua:26`. Extract a shared pane parser/identity helper; keep submission-specific uniqueness and incarnation checks in the transport. This avoids two routing consumers evolving different definitions of an agent pane. **ARCH-DRY**.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed: Go packages `submission`, `webcast/submission`, `webcast/board`, `webcast/recorder`, `cdp`, `webcast`, and `couchmessage`.
   - Passed: submission tests under Neovim, draft-send/DOM adapter/transport Lua tests, and all eight headless DOM integration cases.
   - Couch Go packages and the full transaction script could not build: `cmd/internal/runtimebundle/embed.go:8` reports missing `assets/runtime/files`. The Makefile supplies a generation prerequisite; these failures are verification limitations, not established feature regressions.
   - Existing store failure tests inject file-sync failures before rename, but do not cover directory-sync failure after publication.
   - Live Chrome and operator acceptance were not rerun. The issue explicitly schedules operator acceptance after local close.
   - No prior findings required disposition. The checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY — Flag:** duplicated pane discovery.
   - **ARCH-PURE — Flag:** effectful Lua coordinator classified PURE.
   - **ARCH-PURPOSE — Pass:** the implemented path covers the revised single-hotkey scope; deferred operator acceptance is explicit.
   - **ARCH-MOCK — Pass:** capture doubles, portable store fixtures, and stateful draft tests use production seams; a live Chrome conformance test exists.
   - **ARCH-CONSTRAINTS — Pass:** explicit capture deadlines, sequential capture, input limits, and writer admission bounds.
   - **ARCH-SECURE — Pass:** inspected boundaries validate identities, paths, manifests, and PNG data.
   - **ARCH-ORDER — Flag:** durability uncertainty is collapsed into success on retry; Lua lacks the required pure transition authority.
   - **ARCH-FUNERAL — Pass:** bundles have explicit retention, cleanup guidance, and a writer cap that includes crash residue.

7. **Plan revision recommendations**

   Add `## Revisions` entries that:

   - Correct the Lua entity classification and name the extracted transition model, its states/events, and sequence tests.
   - Specify recovery from successful rename followed by failed directory synchronization, including idempotent retries.
   - Record the shared pane-discovery helper and its consumers.

```findings
findings:
  - id: new
    severity: Critical
    family: durable-retry-confirmation
    title: |
      Idempotent retries bypass failed publication synchronization
    detail: |
      cmd/internal/submission/store.go:467 returns success for an unchanged delivery state after an earlier rename succeeded but syncDir failed at line 498; Begin can therefore authorize paste without confirmed durable intent. The existing-bundle return at line 400 has the corresponding publication gap. Retry directory synchronization before success and add post-rename sync-failure regression tests across these paths. ARCH-ORDER.
  - id: new
    severity: Critical
    family: pure-state-authority
    title: |
      The Lua coordinator contradicts its PURE classification and lacks a pure transition authority
    detail: |
      workshop/plans/000483-dom-submission-plan.md:36 classifies the coordinator as PURE, but nvim/dom_submission.lua:5 independently mutates pending, ack_id, and busy while invoking RPC, scheduling, and delivery effects; nvim/dom_submission_test.lua:4 requires doubles. Reclassify the adapter as INTEGRATION, extract the explicit state/event transition authority, and add a plan Revisions entry. ARCH-PURE and ARCH-ORDER.
  - id: new
    severity: Important
    family: shared-recipient-discovery
    title: |
      Submission transport duplicates agent-pane discovery
    detail: |
      nvim/dom_transport.lua:25 repeats recursive pane collection and the agent identity predicate already implemented in nvim/pair_poke.lua:26. Consolidate parsing and identification in a shared helper while retaining submission-specific uniqueness and incarnation validation at the caller. ARCH-DRY.
```
