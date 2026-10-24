package workbenchshortcut

import "testing"

func TestSendFeedbackChordUsesExistingDraftAction(t *testing.T) {
	chord, ok := DecodeChord([]byte("\x1b[13;8u"))
	if !ok || chord == ChordUnknown {
		t.Fatal("send chord is not declared")
	}
	var send GlobalBinding
	for _, b := range GlobalBindings() {
		if b.Chord == chord {
			send = b
		}
	}
	if send.LuaFunction != "PairSendDOM" || send.NvimKey != "<C-S-M-CR>" || !send.AgentReserved || send.FocusDraft {
		t.Fatalf("send binding = %+v", send)
	}
	for _, role := range []PaneRole{PaneRoleLeftAgent, PaneRoleLeftDraft, PaneRoleReview, PaneRoleRightTerminal} {
		d := Decide(ShortcutInput{Role: role, Chord: chord, DraftPaneID: "draft"})
		if role == PaneRoleReview {
			d, _ = DecideGlobal(chord)
		}
		if d.Disposition != DispositionHandle || d.DraftLuaFunction != "PairSendDOM" || d.FocusDraft {
			t.Fatalf("role %v: %+v", role, d)
		}
	}
	if RightTerminalTUIChordPassesThrough(chord) {
		t.Fatal("fullscreen app consumes send")
	}
	if !RightTerminalTUIChordPassesThrough(ChordAltC) {
		t.Fatal("changed unrelated fullscreen shortcut")
	}
}
