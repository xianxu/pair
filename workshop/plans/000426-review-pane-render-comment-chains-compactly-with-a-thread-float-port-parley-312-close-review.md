# Boundary Review — pair#426 (whole-issue close)

| field | value |
|-------|-------|
| issue | 426 — Review pane: render 🤖 comment chains compactly with a thread float (port parley#312) |
| repo | pair |
| issue file | workshop/issues/000426-review-pane-render-comment-chains-compactly-with-a-thread-float-port-parley-312.md |
| boundary | whole-issue close |
| milestone | — |
| window | 78d8609079842a9cd3d1f745379aba30d3504460..dca4d6678252e9b248da36b920af7876061e3cb6 |
| command | sdlc close --issue 426 |
| reviewer | codex |
| timestamp | 2026-10-10T12:33:48-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The compact rendering, codec, and review handoff work in the targeted tests, and README/atlas updates cover the new controls. Two reproducible correctness bugs block shipping: insert-mode entry can modify concealed turns, and replacing the float’s buffer leaves stale resource ownership.

1. **Strengths**

   - Rendering, cursor targeting, and float opening share one buffer-aware projection.
   - Codec tests combine independent slash-parity fixtures with generated round trips.
   - Real PTY tests verify painted text, speaker colors, save/discard, and undo/redo.
   - Source comparison and range tracking protect float saves against concurrent document edits.

2. **Critical findings**

   - **Insert protection runs too late.** [comment.lua:108](/Users/xianxu/workspace/pair/nvim/review/comment.lua:108) checks cursor movement and text changes, but not entry into insert mode. With `🤖[old]{answer}[reply]`, position the normal cursor on the first `]` (byte column 8), press `i`, then type `X`. Even with separate event-loop turns between keys, the result is `🤖[oldX]{answer}[reply]`: concealed historical text changed. Enforce insertion eligibility before text is inserted, including entry without cursor movement and queued input. Add a real-input regression. **ARCH-PURPOSE, ARCH-ORDER.**

   - **Scratch-buffer replacement leaves stale float ownership.** [comment_float.lua:131](/Users/xianxu/workspace/pair/nvim/review/comment_float.lua:131) handles window closure and source-buffer removal, but not replacement/removal of its scratch buffer. Run `:enew` in a clean thread: the scratch buffer is wiped while its window remains valid. Subsequent thread opening takes the window-only check at [line 88](/Users/xianxu/workspace/pair/nvim/review/comment_float.lua:88), focuses the ordinary replacement buffer, and incorrectly reports success. Later cleanup forcibly closes that replacement window. Handle scratch-buffer departure and validate the window’s buffer identity before focusing or closing it. Add a switch-buffer → reopen → cleanup regression. **ARCH-ORDER, ARCH-FUNERAL.**

3. **Important findings**

   None separate from the regression coverage required above.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed: all five new comment tests, marker and resolution tests, PTY comment rendering, review controls/handoff, review apply, keyhelp package tests, and pinned-range whitespace checks.

   The existing tests miss both reproduced event sequences. The broader `make test-lua` run stopped at `scrollback_test.lua` with an editor-storage permission error; it did not complete. No tracked files were changed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared parser/projection and codec consumers avoid parallel interpretations.
   - **ARCH-PURE — pass:** geometry, serialization, and save decisions remain separate from Neovim effects.
   - **ARCH-PURPOSE — flag:** concealed-turn edit protection does not meet the stated contract.
   - **ARCH-MOCK — pass:** no new external dependency; handoff exercises the existing stateful host seam.
   - **ARCH-CONSTRAINTS — pass:** compact projection has explicit size limits, cached cursor lookup, and clamped float geometry.
   - **ARCH-SECURE — pass:** serialization validation and exact source comparison guard replacement.
   - **ARCH-ORDER — flag:** insert entry and scratch-buffer replacement are missing event cases.
   - **ARCH-FUNERAL — flag:** scratch replacement strands ownership and tracking resources until later teardown.

7. **Plan revision recommendations**

   Append a `## Revisions` entry covering insertion admission before mutation and scratch-buffer replacement/ownership transitions. Extend Tasks 2–3 with the two production-event regressions before marking their verification complete.

```findings
findings:
  - id: new
    severity: Critical
    family: insertion-admission-before-mutation
    title: |
      Entering insert mode permits edits to concealed historical turns
    detail: |
      nvim/review/comment.lua:108-109 omits insertion-entry admission. On 🤖[old]{answer}[reply], place the cursor at byte column 8, press i, then X; the result is 🤖[oldX]{answer}[reply], even with event-loop turns between keys. Enforce the allowed insertion points before mutation and test real mode-entry and queued-input sequences. ARCH-PURPOSE, ARCH-ORDER.
  - id: new
    severity: Critical
    family: owned-resource-identity-on-retarget
    title: |
      Replacing the thread scratch buffer leaves stale window ownership
    detail: |
      nvim/review/comment_float.lua:131-134 does not handle scratch-buffer replacement. After :enew in a clean thread, line 88 treats the surviving window as a live thread, so reopening focuses an ordinary buffer and later cleanup closes that replacement window. Release ownership on scratch departure and verify buffer identity before focus/close; test replacement, reopening, and cleanup. ARCH-ORDER, ARCH-FUNERAL.
```

---

## Re-review — 2026-10-10T12:41:10-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 426 — Review pane: render 🤖 comment chains compactly with a thread float (port parley#312) |
| repo | pair |
| issue file | workshop/issues/000426-review-pane-render-comment-chains-compactly-with-a-thread-float-port-parley-312.md |
| boundary | whole-issue close |
| milestone | — |
| window | 78d8609079842a9cd3d1f745379aba30d3504460..f3ebcb63dfd0efff68f2265a9e1f8a96c2a7874c |
| command | sdlc close --issue 426 |
| reviewer | codex |
| timestamp | 2026-10-10T12:41:10-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The compact rendering, codec, and thread lifecycle have substantial coverage. BR-2 is fixed and independently regression-verified. BR-1 remains blocking: insertion admission still permits a queued newline inside concealed history.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      nvim/review/comment.lua:116-123 protects mode entry and InsertCharPre, but Enter bypasses the latter. In the real PTY fixture, use 🤖[old]{answer}[reply], position at byte column 8, then queue i<Left><Left><Left><CR>X<Esc>. The concealed agent turn splits, leaving the first line 🤖[old]{answe. The existing typed-character regressions pass, but this added probe fails. Enumerate insertion mechanisms and enforce admission before every supported mutation, including newline insertion; add real queued-input regressions. ARCH-PURPOSE, ARCH-ORDER.
  - id: BR-2
    disposition: addressed
    note: |
      comment_float.lua checks window/buffer identity before focus and close, and releases ownership on scratch departure. Current replacement/reopen/cleanup tests pass. Running the same regression against the pre-fix controller in a temporary copy fails at “reopen must create a real thread,” confirming meaningful regression coverage.
```

1. **Strengths**
   - Shared projection keeps rendering, cursor policy, and thread targeting consistent.
   - Canonical codec fixtures supplement delimiter-generated round-trip tests.
   - Source range identity and byte comparison prevent stale thread saves from overwriting changed text.
   - README, atlas, and derived keyboard help document the new controls.

2. **Critical findings**
   - **BR-1 remains open**, [comment.lua:116](/Users/xianxu/workspace/pair/nvim/review/comment.lua:116). Cover the insertion-admission rule across mutation mechanisms, rather than adding another character-only guard. The reproduced newline bypass belongs to the existing `insertion-admission-before-mutation` family; it is not a new finding.

3. **Important findings**
   - None newly raised.

4. **Minor findings**
   - None.

5. **Test coverage**
   - Passed: shipped PTY test; all comment codec/thread/view/float/attachment tests; marker and resolution tests; key-help package tests; diff whitespace check.
   - BR-2 pre-fix mutation correctly fails.
   - Additional real-PTY queued-newline probe fails at HEAD.
   - Broader `make test-lua` stopped in unchanged `scrollback_test.lua` with a sandbox storage permission error; full-suite success was not independently established.

6. **Architecture**
   - **ARCH-DRY — pass:** shared scanner, projection, and codec.
   - **ARCH-PURE — pass:** deterministic core separated from Neovim effects.
   - **ARCH-PURPOSE — flag:** concealed-history edit protection remains incomplete.
   - **ARCH-MOCK — pass:** no new external dependency; UI tests exercise real isolated Neovim.
   - **ARCH-CONSTRAINTS — pass:** explicit projection limits and bounded float geometry.
   - **ARCH-SECURE — pass:** serialized text and source identity validated before replacement.
   - **ARCH-ORDER — flag:** newline mutation can precede cursor admission.
   - **ARCH-FUNERAL — pass:** scratch departure releases ownership; cleanup preserves replacement windows.

7. **Plan revision recommendation**
   - Append a `## Revisions` entry recording the newline bypass, enumerating insertion event classes, and requiring queued-input tests for mechanisms that do not trigger `InsertCharPre`.

---

## Re-review — 2026-10-10T12:46:25-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 426 — Review pane: render 🤖 comment chains compactly with a thread float (port parley#312) |
| repo | pair |
| issue file | workshop/issues/000426-review-pane-render-comment-chains-compactly-with-a-thread-float-port-parley-312.md |
| boundary | whole-issue close |
| milestone | — |
| window | 78d8609079842a9cd3d1f745379aba30d3504460..320c2a53e080eb288c34dabbf131d97f7fc9f895 |
| command | sdlc close --issue 426 |
| reviewer | codex |
| timestamp | 2026-10-10T12:46:25-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the revised Spec and Plan. BR-1 is addressed with pre-mutation admission and meaningful regression coverage; BR-2 remains addressed. Targeted verification passed, and I found no new blocking issues. Repository files were unchanged.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      InsertEnter and the activation-owned on_key callback admit insertion points before processing input. Current PTY regressions pass; reverting comment.lua to f3ebcb63 in a temporary copy makes the queued-newline regression fail by splitting concealed {answer} text.
  - id: BR-2
    disposition: addressed
    note: |
      Scratch departure ends ownership, and focus/cleanup check the exact window-buffer pair. Clean and dirty replacement, rescue, reopening, and replacement-window preservation tests pass.
```

1. **Strengths**

   - Rendering, cursor protection, and thread targeting share the same projection in `nvim/review/comment_view.lua`.
   - Canonical escape fixtures independently verify wire compatibility; generated round trips additionally cover delimiters and Unicode.
   - Float saves validate both serialization and the anchored source bytes before replacement.
   - PTY tests assert painted cells and actual input behavior. The handoff test verifies the agent host reads the saved encoded reply.
   - README, atlas, runtime classification, and keyboard-help discovery cover the new surface.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed seven targeted Lua suites: codec, thread model, projection, attachment, float lifecycle, markers, and resolution. Also passed `review-comments-test.sh`, `review-apply-test.sh`, `review-controls-test.sh`, keyhelp package tests, and pinned-range whitespace checks.

   The temporary guard reversion failed on the intended hidden-text corruption assertion. This review ran targeted verification, not the entire repository suite.

6. **Architectural notes**

   - **ARCH-DRY — Pass:** Shared projection, codec, and help-source inventory avoid competing definitions.
   - **ARCH-PURE — Pass:** Parsing, layout, serialization, and save decisions remain separate from Neovim effects; core-concept classifications match implementation.
   - **ARCH-PURPOSE — Pass:** Compact rendering, editable threads, protection, and review handoff are delivered.
   - **ARCH-MOCK — Pass:** No new external dependency; handoff uses the stateful host fixture and editor integration uses isolated Neovim.
   - **ARCH-CONSTRAINTS — Pass:** Projection limits, cached cursor handling, raw fallback, and clamped float geometry implement the declared envelope.
   - **ARCH-SECURE — Pass:** Document content remains data; serialization and exact source identity are validated before writes.
   - **ARCH-ORDER — Pass:** Encapsulated save transitions and real queued-input regressions cover the relevant event ordering.
   - **ARCH-FUNERAL — Pass:** Activation and float cleanup release listeners, mappings, marks, windows, and buffers while preserving unsaved text.

7. **Plan revision recommendations:** None.
