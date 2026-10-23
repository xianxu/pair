package repolint

import "testing"

func TestSlashCommandSubmitRule(t *testing.T) {
	const drive = `import drive "github.com/xianxu/pair/cmd/internal/agentdrive"
`
	for _, tc := range []struct {
		name, path, source string
		want               int
	}{
		{"local sequence", "cmd/internal/agentdrive/example.go", `var steps = KeySequence{{Kind: KeyText, Text: "/model"}, {Kind: KeySubmit}}`, 1},
		{"local step slice", "cmd/internal/agentdrive/example.go", `var steps = []KeyStep{{Kind: KeyText, Text: "/exit"}, {Kind: KeyWait}, {Kind: KeySubmit}}`, 1},
		{"qualified sequence", "cmd/internal/wrapcmd/example.go", drive + `var steps = drive.KeySequence{{Kind: drive.KeyText, Text: "/model gpt"}, {Kind: drive.KeyWait}, {Kind: drive.KeyWait}, {Kind: drive.KeySubmit}}`, 1},
		{"qualified step slice", "cmd/internal/wrapcmd/example.go", drive + "var steps = []drive.KeyStep{drive.KeyStep{Kind: drive.KeyText, Text: `/exit`}, drive.KeyStep{Kind: drive.KeySubmit}}", 1},
		{"later slash sequence", "cmd/internal/agentdrive/example.go", `var steps = KeySequence{{Kind: KeyText, Text: "hello"}, {Kind: KeySubmit}, {Kind: KeyText, Text: "/exit"}, {Kind: KeySubmit}}`, 1},
		{"compatibility test", "cmd/internal/agentdrive/example_test.go", `var steps = KeySequence{{Kind: KeyText, Text: "/model"}, {Kind: KeySubmit}}`, 0},
		{"qualified compatibility test", "cmd/internal/wrapcmd/example_test.go", drive + `var steps = drive.KeySequence{{Kind: drive.KeyText, Text: "/model"}, {Kind: drive.KeySubmit}}`, 0},
		{"command enter", "cmd/internal/agentdrive/example.go", `var steps = KeySequence{{Kind: KeyText, Text: "/model"}, {Kind: KeyEnter}}`, 0},
		{"generic submit", "cmd/internal/agentdrive/example.go", `var steps = KeySequence{{Kind: KeySubmit}}`, 0},
		{"ordinary prose", "cmd/internal/agentdrive/example.go", `var steps = KeySequence{{Kind: KeyText, Text: "hello /model"}, {Kind: KeySubmit}}`, 0},
		{"unsent slash draft", "cmd/internal/agentdrive/example.go", `var steps = KeySequence{{Kind: KeyText, Text: "/model"}, {Kind: KeyWait}}`, 0},
		{"different action intervenes", "cmd/internal/agentdrive/example.go", `var steps = KeySequence{{Kind: KeyText, Text: "/model"}, {Kind: KeyClearLine}, {Kind: KeySubmit}}`, 0},
		{"dynamic text limitation", "cmd/internal/agentdrive/example.go", `var steps = KeySequence{{Kind: KeyText, Text: command}, {Kind: KeySubmit}}`, 0},
		{"dynamic step limitation", "cmd/internal/agentdrive/example.go", `var steps = KeySequence{{Kind: KeyText, Text: "/model"}, next, {Kind: KeySubmit}}`, 0},
		{"unrelated slice", "cmd/internal/agentdrive/example.go", `var steps = []OtherStep{{Kind: KeyText, Text: "/model"}, {Kind: KeySubmit}}`, 0},
		{"unrelated import", "cmd/internal/wrapcmd/example.go", `import drive "example.com/other"
var steps = drive.KeySequence{{Kind: drive.KeyText, Text: "/model"}, {Kind: drive.KeySubmit}}`, 0},
		{"shadowed import", "cmd/internal/wrapcmd/example.go", drive + `func send() { drive := other; _ = drive.KeySequence{{Kind: drive.KeyText, Text: "/model"}, {Kind: drive.KeySubmit}} }`, 0},
		{"separate literals limitation", "cmd/internal/agentdrive/example.go", `var draft = KeySequence{{Kind: KeyText, Text: "/model"}}
var send = KeySequence{{Kind: KeySubmit}}`, 0},
		{"direct peer alt return", "cmd/internal/agentdrive/example.go", `func send() { peer.Send("\x1b\r") }`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package agentdrive\n" + tc.source + "\n")
			repository := newFakeRepository()
			repository.index[tc.path] = source
			findings, _, count, err := scanTrackedWithRepository(repository, "repo", func(string) ([]byte, error) { return source, nil })
			if err != nil || count != 1 || len(findings) != tc.want {
				t.Fatalf("count=%d err=%v findings=%+v want=%d", count, err, findings, tc.want)
			}
			for _, finding := range findings {
				if finding.Finding.Rule != "slash-command-submit" {
					t.Fatalf("unexpected finding: %+v", finding)
				}
			}
		})
	}
}
