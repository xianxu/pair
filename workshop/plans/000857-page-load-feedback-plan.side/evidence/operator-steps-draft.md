Combined #857/#866 operator check — draft, build authorization pending

Restart Couch only after the hub confirms the combined stage revision and its binaries.

Upgrade caveat: a cast recording from before the upgrade may have an active legacy segment without load identity. New notes then refuse with “The page changed. Try again.” Reload that page before testing new feedback; this is the approved unknown-load policy, not a startup defect.

1. In a disposable cast page, save a note and a labelled sketch. Pause/resume, then switch to another tab and back without reloading: the original load retains its feedback. A new tab at the same URL starts empty. Reload the original page: its old notes and sketches disappear from the panel, page and fresh submission; newly added feedback belongs to the new load. Earlier feedback remains recording history.
2. Start one fresh Claude thread and one fresh Codex thread. Each must reach its normal agent screen without a startup failure. Confirm the pane title appears and ordinary terminal pane focus/open and text input work after startup.
3. Report any startup refusal/crash or feedback from a different tab/load. #866's check does not claim concurrent interactive attach/reconnect or creator-death behavior. Existing unsaved-note protection remains.

#866 source/native evidence supplied by pair:5: d1d1388cddeff9818feb1c3f7dff569318d8a149. Verify the final green branch head before assembling; this is not yet the combined revision.
