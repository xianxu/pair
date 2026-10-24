# Cast feedback workspace mock

Synthetic layout mock for pair#856/#857. Run `couch mock serve workshop/plans/000856-cast-feedback-workspace-plan.side/mocks --port 0`.

Entries use type icons, with no thumbnails. Hover or focus a row to glow its page anchor; absent anchors do not guess. Try the mixed Notes list, Add note / read-only View note, Show on page, minimize and drag (or focus the header and use arrow keys). Toggle Sketch for the one-column tools. Resize to a narrow window. Use the scenario controls to compare reload, tab return, app update, unsaved editor and status. These are local synthetic transitions, not a recorder implementation. Production code waits for operator layout approval.

Open the URL in a **non-cast Chrome window**. Opening a tab inside a live cast injects the real tool UI over this mock; the mock does not disable or reconfigure capture. The mock tool column uses Live/Pause, Pointer, Pen with colors, Note, available Undo/Redo and Done. With Pen active, mark a synthetic card to exercise history. With Note active, select a card to open a draft. New tab starts empty even though it shares the synthetic page URL; Switch tab returns to the board load.
