# Close Review: pair#112 review pane inline definitions

- Window: `0d8101a..HEAD`
- Final verdict: `SHIP`
- Final actual: `1.72h`
- Final verification: `bash tests/review-definition-test.sh`, `make test-lua`,
  `make test-review`, `git diff --check`; earlier full verification included
  `go test ./...` and `make test`.

## Boundary Review Trail

- Initial close: `REWORK`
  - Visual selection end column used Neovim's 1-based inclusive mark as a
    0-based inclusive byte column.
  - Continued-review context could expose the managed definition footer.
  - `pair review definition` ignored atomic write errors.
  - README did not document the user-facing command/keybinding.
- Second close: `REWORK`
  - Managed-footer recognition treated ordinary trailing `---` plus Markdown
    footnotes as pair-managed definition state.
- Third close: `REWORK`
  - Durable rehydration expanded only over the last word, not multi-word
    selected phrases.
- Fourth close: `REWORK`
  - Definition application bypassed projection, so undo could leave stale
    extmarks/diagnostics.
- Fifth close: `FIX-THEN-SHIP`
  - Pending definition results used stale raw coordinates after intervening
    buffer edits.
- Re-close after follow-up: `SHIP`
  - No critical, important, or minor findings.

## Resolution

The final implementation:

- Tracks pending selections with an extmark and validates the live term before
  applying a result.
- Writes a stripped `review-context-<tag>.md` artifact for continued review.
- Requires the managed footer shape Pair writes (`---`, blank line, footnotes)
  before stripping or rehydrating definition footnotes.
- Rehydrates phrase spans by matching the preceding text suffix through the same
  `footnote_id` slugger used on write.
- Runs definition application through projection with a fresh undo block, so
  undo/redo updates definition text and decorations together.
- Documents `Shift+Alt+d` and `pair review definition` in README and atlas.
