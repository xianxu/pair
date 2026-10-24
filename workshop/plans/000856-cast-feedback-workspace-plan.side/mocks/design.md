# Cast feedback workspace mock

Synthetic layout mock for pair#856/#857. Run `couch mock serve workshop/plans/000856-cast-feedback-workspace-plan.side/mocks --port 0`.

Try the mixed Notes list, Add note / read-only View note, Show on page, minimize and drag (or focus the header and use arrow keys). Toggle Sketch for the one-column tools. Resize to a narrow window. Use the scenario controls to compare reload, tab return, app update, unsaved editor and status. These are local synthetic transitions, not a recorder implementation. Production code waits for operator layout approval.
