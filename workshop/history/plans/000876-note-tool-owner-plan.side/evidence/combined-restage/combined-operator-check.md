# Combined operator check: #857, #866, #844, #876

Stage revision: **9bfd7ab51cef5f9d563b19dea087c9ef084c19c9**, all five binaries verified at this revision, modified=false. Source: pinned current main **dc3031079bddd3c160cd9481fdef94f1ff29c83b** plus the four tested issue heads in stage-provenance.json. This is a local staging-only merge, not a landing.

Independent main-to-exact-stage warm check: **PASS146.58s** (package146.745s), Claude and Codex hub+worker, all four original actors. Startup adoption, public listing and resume, exact-tag Continue/native list-clients all pass; full bindings, fixture PID/generation/nativeID/transcript and socket inode unchanged. Both disposable fleets cleaned, all owned generations absent/direct child and IO joined. Native log SHA256 **7f87f8c973bcefd6cbf4950d01cadb5645c43d34c06a6811c4f19442c163340c**. The main-only scratch checkout is removed after the test; the staged build is retained.

For each issue, reply **PASS** or the **first problem**.

## Restart and CPU profile

The original ~/couch-profile-871 already exists and is preserved. Use this new output folder, which must not exist before restarting:

```sh
COUCH_PROFILE_DIR="$HOME/couch-profile-871-restage-20261024" couch
```

Keep this combined build after the check. Do not clear it until the hub sequences the next stage.

## #857 — Feedback belongs to this tab and page load

1. On a disposable cast page, save a note and labelled sketch. Pause/resume and switch to another tab and back without reloading: the original page load keeps its feedback.
2. Open a new tab at the same URL: its Notes panel starts empty. Return to the original tab: its feedback returns.
3. Reload the original page: earlier notes/sketches disappear from the page, Notes panel and fresh submission selection. Add a new note: only current-load feedback appears; earlier feedback stays in recording history.

Legacy upgrade caveat: an old cast segment without page-load identity refuses new notes with “The page changed. Try again.” Reload that page before adding feedback.

Result: PASS / first problem:

## #866 — Warm restart and fresh agent startup

1. Confirm existing warm threads appear after restart and can be opened/resumed without startup refusal.
2. Start one fresh Claude thread and one fresh Codex thread. Each reaches its normal agent screen, with its pane title.
3. Confirm ordinary right-terminal pane focus/open and text input work.

A named “terminal first-pane readiness unverified” error means an old/custom layout: update the selected Pair binary, Lua and layout assets together. This does not claim creator-death or simultaneous interactive reconnect coverage.

Result: PASS / first problem:

## #844 — Bell holds requests for the operator

1. Open the bell: ordinary reattach/freshness feedback causes no per-slot storm; the switcher feed still retains that feedback.
2. TL sends the explicit human notice below. Confirm one bell item appears and opens the requesting hub:

```sh
couch --notify --title '#844 live check' '#844 explicit human notification'
```

3. Start/end a cast normally, or close cast Chrome normally: progress/success/expected stop stay feed-only. Unexpected-error, ownership, OFF/no-replay and no-owner cases remain covered by deterministic production-path tests.

Result: PASS / first problem:

## #876 — Owner Note and Pointer with viewer Annotation OFF

1. Leave viewer Annotation OFF. In the operator's cast Chrome, select Note, draw a box, and save its note. Select Pointer and point on the page; both tools work without enabling viewers.
2. Copy the pointer link using right-click (without toggling Annotation ON). Open it on another device or in a non-cast browser window: the viewer cannot annotate, and Notes explains how to request annotation. A view-only link explains that an annotating link is needed.
3. Expand Pen and narrow the window. The labelled Note button remains reachable through the tool column; switching from Pen still completes its required sketch-label handoff.

Result: PASS / first problem:
