package repolint

import (
	"strings"
	"testing"
)

func TestScribePTYPrintfRule(t *testing.T) {
	for _, tc := range []struct {
		name, pkg, body string
		want            int
	}{
		{"direct printf", "scribecmd", `runWithPTY(t, []string{"-log", log, "--", "printf", "marker"})`, 1},
		{"absolute printf", "scribecmd", `runWithPTYReply(t, []string{"--", "/usr/bin/printf", "marker"}, "reply")`, 1},
		{"parentheses", "scribecmd", `(runWithPTY)(t, ([]string{"--", ("printf"), "marker"}))`, 1},
		{"raw string", "scribecmd", "runWithPTY(t, []string{`--`, `printf`, `marker`})", 1},
		{"tty producer", "scribecmd", `runWithPTYReply(t, []string{"--", "/bin/sh", "-c", "exec 3<>/dev/tty; read ack <&3; printf marker"}, "ack")`, 0},
		{"printf as argument", "scribecmd", `runWithPTY(t, []string{"--", "echo", "printf"})`, 0},
		{"printf as log", "scribecmd", `runWithPTY(t, []string{"-log", "printf", "--", "sh", "-c", "exit 0"})`, 0},
		{"other executable", "scribecmd", `runWithPTY(t, []string{"--", "myprintf", "marker"})`, 0},
		{"other helper", "scribecmd", `runWithoutPTY(t, []string{"--", "printf", "marker"})`, 0},
		{"other package", "other", `runWithPTY(t, []string{"--", "printf", "marker"})`, 0},
		{"args variable outside syntax guard", "scribecmd", `args := []string{"--", "printf", "marker"}; runWithPTY(t, args)`, 0},
		{"comment and string", "scribecmd", "// runWithPTY(t, []string{\"--\", \"printf\"})\n_ = `runWithPTY(t, []string{\"--\", \"printf\"})`", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package " + tc.pkg + "\nfunc TestExample(){\n" + tc.body + "\n}")
			repository := newFakeRepository()
			repository.index["cmd/internal/"+tc.pkg+"/example_test.go"] = source
			findings, _, count, err := scanTrackedWithRepository(repository, "repo", func(string) ([]byte, error) { return source, nil })
			if err != nil || count != 1 || len(findings) != tc.want {
				t.Fatalf("findings=%v count=%d err=%v want=%d", findings, count, err, tc.want)
			}
			for _, f := range findings {
				if f.Finding.Rule != "scribe-pty-printf" || !strings.Contains(f.Finding.Message, "pair#766") {
					t.Fatalf("unexpected finding %+v", f)
				}
			}
		})
	}
}
