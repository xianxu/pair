package repolint

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestFixtureCLIGroupCompletionRule(t *testing.T) {
	for _, tc := range []struct {
		name, pkg, body string
		want            int
	}{
		{"unobserved wait", "agentfixture", `func runCLI(){ cmd.Wait() }`, 1},
		{"ordinary error still requires observation", "agentfixture", `func runCLI(){ if err := cmd.Wait(); err != nil { return } }`, 1},
		{"observed completion", "agentfixture", `func runCLI(){ cmd.Wait(); cliGroupError(signalErr, members, inspectionErr) }`, 0},
		{"comment is not observation", "agentfixture", "func runCLI(){ cmd.Wait(); /* cliGroupError() */ }", 1},
		{"other function", "agentfixture", `func unrelated(){ cmd.Wait() }`, 0},
		{"other package", "other", `func runCLI(){ cmd.Wait() }`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, err := parser.ParseFile(token.NewFileSet(), "fixture.go", "package "+tc.pkg+"\n"+tc.body, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(fixtureCLIUnobservedGroups(f)); got != tc.want {
				t.Fatalf("findings=%d want=%d", got, tc.want)
			}
		})
	}
}
