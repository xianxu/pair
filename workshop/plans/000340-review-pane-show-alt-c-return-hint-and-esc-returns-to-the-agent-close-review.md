# Boundary Review — pair#340 (whole-issue close)

| field | value |
|-------|-------|
| issue | 340 — Review pane: show Alt+c return hint, and Esc returns to the agent |
| repo | pair |
| issue file | workshop/issues/000340-review-pane-show-alt-c-return-hint-and-esc-returns-to-the-agent.md |
| boundary | whole-issue close |
| milestone | — |
| window | b2beefeacd8ca2b73e51a46b44ed87d86117ad54..f5a30cef8905417a93d3aa9d0e608926668ccd26 |
| command | sdlc close --issue 340 |
| reviewer | codex |
| timestamp | 2026-09-28T18:54:22-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation matches the revised draft-return requirements, and the affected tests pass. One Important regression-coverage gap remains: the agent-selection correction lacks an assertion that detects the original wrong-pane behavior.

1. **Strengths**
   - Shared draft-return logic serves review, scrollback, and changelog.
   - Real Neovim mappings exercise mode preservation, popup dismissal, and marker wrapping.
   - Couch tests cover chord encodings and preserve explicit relaunch behavior.
   - Review help derives descriptions from mappings; README and atlas cover the new controls.

2. **Critical findings:** None.

3. **Important findings**
   - [tests/review-controls-test.sh:26](/Users/xianxu/workspace/pair/tests/review-controls-test.sh:26): the reordered-pane fixture accepts `write-chars` and `send-keys` without checking their destination. Meanwhile, [tests/review-poke-test.sh:18](/Users/xianxu/workspace/pair/tests/review-poke-test.sh:18) puts the agent first, so its destination assertions also pass with the old selector. Add the unrelated terminal before the agent in the poke test and assert both operations target pane 7. Confirm restoring the old selector fails that test.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Passed affected Go packages: `keyhelp`, `keyscmd`, `workbenchshortcut`, `couchcmd`, `couchtty`.
   - Passed shell suites: review controls, overlay return, review poke, review toggle, and statusline position.
   - No independent live Pair/Couch smoke or mutation test performed.

6. **Architectural notes**
   - **ARCH-DRY — pass:** shared return helper and source-derived help.
   - **ARCH-PURE — pass:** parsing/classification remain separate from host operations.
   - **ARCH-PURPOSE — pass:** revised controls and documentation are delivered.
   - **ARCH-MOCK — flag:** pane-host double omits the destination invariant described above.
   - **ARCH-CONSTRAINTS — pass:** existing bounded Couch focus probe reused.
   - **ARCH-SECURE — pass:** command arguments remain structured; malformed pane JSON is rejected.
   - **ARCH-ORDER — pass:** viewer tests assert annotation emission precedes hide/focus.
   - **ARCH-FUNERAL — pass:** no new durable runtime artifact family or background process.

7. **Plan revision recommendation**
   - Append a `## Revisions` entry recording the strengthened poke-destination regression and its failure with the old selector.

```findings
findings:
  - id: new
    severity: Important
    family: regression-oracle-covers-corrected-behavior
    title: |
      Agent-selection regression does not assert the selected destination
    detail: |
      tests/review-controls-test.sh:26 accepts writes to any pane despite placing an unrelated terminal before the agent; tests/review-poke-test.sh:18 checks destinations but places the agent first. ARCH-MOCK: add the reordered terminal to the destination-checking fixture, assert both body and submit target pane 7, and verify the old title-based selector makes the test fail.
```
