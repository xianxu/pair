package termcmd

import "testing"

func TestRenameEditorTransitions(t *testing.T) {
	tests := []struct {
		name       string
		start      string
		events     []RenameEvent
		wantText   string
		wantCursor int
		wantKind   RenameOutcomeKind
		wantName   string
	}{
		{
			name:       "inserts rune at cursor",
			start:      "ac",
			events:     []RenameEvent{{Kind: RenameMoveLeft}, {Kind: RenameInsert, Rune: '界'}},
			wantText:   "a界c",
			wantCursor: 2,
		},
		{
			name:       "moves by unicode rune",
			start:      "a🙂界",
			events:     []RenameEvent{{Kind: RenameMoveLeft}, {Kind: RenameMoveLeft}},
			wantText:   "a🙂界",
			wantCursor: 1,
		},
		{
			name:       "home end and boundary moves",
			start:      "abc",
			events:     []RenameEvent{{Kind: RenameHome}, {Kind: RenameMoveLeft}, {Kind: RenameEnd}, {Kind: RenameMoveRight}},
			wantText:   "abc",
			wantCursor: 3,
		},
		{
			name:       "backspace and delete remove corresponding runes",
			start:      "a🙂界z",
			events:     []RenameEvent{{Kind: RenameMoveLeft}, {Kind: RenameBackspace}, {Kind: RenameDelete}},
			wantText:   "a🙂",
			wantCursor: 2,
		},
		{
			name:       "boundary deletion is consumed no-op",
			start:      "x",
			events:     []RenameEvent{{Kind: RenameHome}, {Kind: RenameBackspace}, {Kind: RenameEnd}, {Kind: RenameDelete}, {Kind: RenameConsume}},
			wantText:   "x",
			wantCursor: 1,
		},
		{
			name:       "commit trims nonempty text",
			start:      "old",
			events:     []RenameEvent{{Kind: RenameHome}, {Kind: RenameInsert, Rune: ' '}, {Kind: RenameEnd}, {Kind: RenameInsert, Rune: ' '}, {Kind: RenameCommit}},
			wantText:   " old ",
			wantCursor: 5,
			wantKind:   RenameOutcomeCommit,
			wantName:   "old",
		},
		{
			name:       "empty commit retains original name",
			start:      "old",
			events:     []RenameEvent{{Kind: RenameHome}, {Kind: RenameDelete}, {Kind: RenameDelete}, {Kind: RenameDelete}, {Kind: RenameCommit}},
			wantText:   "",
			wantCursor: 0,
			wantKind:   RenameOutcomeCommit,
			wantName:   "old",
		},
		{
			name:       "cancel preserves original name",
			start:      "old",
			events:     []RenameEvent{{Kind: RenameInsert, Rune: '!'}, {Kind: RenameCancel}},
			wantText:   "old!",
			wantCursor: 4,
			wantKind:   RenameOutcomeCancel,
			wantName:   "old",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			editor := NewRenameEditor(tt.start)
			var outcome RenameOutcome
			for _, event := range tt.events {
				editor, outcome = editor.Apply(event)
			}
			if got := editor.Text(); got != tt.wantText {
				t.Fatalf("Text() = %q, want %q", got, tt.wantText)
			}
			if got := editor.Cursor(); got != tt.wantCursor {
				t.Fatalf("Cursor() = %d, want %d", got, tt.wantCursor)
			}
			if outcome.Kind != tt.wantKind || outcome.Name != tt.wantName {
				t.Fatalf("outcome = %#v, want kind=%v name=%q", outcome, tt.wantKind, tt.wantName)
			}
			if got := editor.Original(); got != tt.start {
				t.Fatalf("Original() = %q, want immutable %q", got, tt.start)
			}
		})
	}
}
