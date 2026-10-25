package repolint

import "testing"

func TestZellijQueryBoundaryRule(t *testing.T) {
	const collector = `import zx "github.com/xianxu/pair/cmd/internal/zellijexec"
`
	const command = `import proc "os/exec"
`
	for _, tc := range []struct {
		name, path, pkg, source string
		want                    int
	}{
		{"collector call", "cmd/internal/newconsumer/query.go", "newconsumer", collector + `func read() { zx.Run(cmd) }`, 1},
		{"collector reference", "cmd/internal/newconsumer/query.go", "newconsumer", collector + `var run = (zx.Run)`, 1},
		{"collector shadow", "cmd/internal/newconsumer/query.go", "newconsumer", collector + `func read() { zx := other; zx.Run(cmd) }`, 0},
		{"collector dot import", "cmd/internal/newconsumer/query.go", "newconsumer", `import . "github.com/xianxu/pair/cmd/internal/zellijexec"
func read() { Run(cmd) }`, 1},
		{"collector blank import", "cmd/internal/newconsumer/query.go", "newconsumer", `import _ "github.com/xianxu/pair/cmd/internal/zellijexec"
func read() { Run(cmd) }`, 0},
		{"shared attempt", "cmd/internal/zellijquery/query.go", "zellijquery", collector + `func (s Source) readAttempt() { run := zx.Run; run(cmd) }`, 0},
		{"other shared method", "cmd/internal/zellijquery/query.go", "zellijquery", collector + `func (s Source) Output() { zx.Run(cmd) }`, 1},
		{"version diagnostic", "cmd/internal/launcher/zellij_query_evidence.go", "launcher", collector + `func queryZellijVersion() { zx.Run(cmd) }`, 0},
		{"diagnostic same name wrong file", "cmd/internal/launcher/query.go", "launcher", collector + `func queryZellijVersion() { zx.Run(cmd) }`, 1},
		{"diagnostic wrong receiver", "cmd/internal/launcher/zellij_query_evidence.go", "launcher", collector + `func (s Other) queryZellijVersion() { zx.Run(cmd) }`, 1},
		{"fixture raw diagnostic", "cmd/internal/testcouch/startup_evidence.go", "testcouch", collector + `func (f *Fleet) runStartupQuery() { zx.Run(cmd) }`, 0},
		{"fixture former owner", "cmd/internal/testcouch/startup_evidence.go", "testcouch", collector + `func (f *Fleet) dumpStartupPane() { zx.Run(cmd) }`, 1},
		{"fixture wrong receiver", "cmd/internal/testcouch/startup_evidence.go", "testcouch", collector + `func (f *Other) runStartupQuery() { zx.Run(cmd) }`, 1},
		{"fixture wrong file", "cmd/internal/testcouch/consumer.go", "testcouch", collector + `func (f *Fleet) runStartupQuery() { zx.Run(cmd) }`, 1},
		{"fixture ordinary consumer", "cmd/internal/testcouch/consumer.go", "testcouch", collector + `func read() { zx.Run(cmd) }`, 1},
		{"test collector", "cmd/internal/newconsumer/query_test.go", "newconsumer", collector + `func read() { zx.Run(cmd) }`, 0},
		{"native probe", "cmd/probes/example/main.go", "main", collector + `func read() { zx.Run(cmd) }`, 0},
		{"root native probe", "probes/example/main.go", "main", collector + `func read() { zx.Run(cmd) }`, 0},
		{"native repro", "cmd/repro/example/main.go", "main", collector + `func read() { zx.Run(cmd) }`, 0},
		{"direct sessions", "cmd/internal/newconsumer/query.go", "newconsumer", command + `func read() { proc.Command("zellij", "list-sessions", "--no-formatting") }`, 1},
		{"direct panes context", "cmd/internal/newconsumer/query.go", "newconsumer", command + `func read() { (proc.CommandContext)(ctx, "zellij", "--session", session, "action", "list-panes", "--json") }`, 1},
		{"direct clients", "cmd/internal/newconsumer/query.go", "newconsumer", command + `func read() { proc.Command("zellij", "action", "list-clients") }`, 1},
		{"direct tab", "cmd/internal/newconsumer/query.go", "newconsumer", command + `func read() { proc.Command("zellij", "action", "current-tab-info", "--json") }`, 1},
		{"direct tab names", "cmd/internal/newconsumer/query.go", "newconsumer", command + `func read() { proc.Command("zellij", "--session", session, "action", "query-tab-names") }`, 1},
		{"mutation containing read argument", "cmd/internal/newconsumer/query.go", "newconsumer", command + `func read() { proc.Command("zellij", "action", "write-chars", "list-panes") }`, 0},
		{"version", "cmd/internal/newconsumer/query.go", "newconsumer", command + `func read() { proc.Command("zellij", "--version") }`, 0},
		{"command shadow", "cmd/internal/newconsumer/query.go", "newconsumer", command + `func read() { proc := other; proc.Command("zellij", "list-sessions") }`, 0},
		{"other executable", "cmd/internal/newconsumer/query.go", "newconsumer", command + `func read() { proc.Command("echo", "list-sessions") }`, 0},
		{"dynamic argv limitation", "cmd/internal/newconsumer/query.go", "newconsumer", command + `func read() { proc.Command("zellij", args...) }`, 0},
		{"dot command", "cmd/internal/newconsumer/query.go", "newconsumer", `import . "os/exec"
func read() { Command("zellij", "list-sessions") }`, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package " + tc.pkg + "\n" + tc.source + "\n")
			repository := newFakeRepository()
			repository.index[tc.path] = source
			findings, _, count, err := scanTrackedWithRepository(repository, "repo", func(string) ([]byte, error) { return source, nil })
			var boundary []locatedFinding
			for _, finding := range findings {
				if finding.Finding.Rule == "zellij-query-boundary" {
					boundary = append(boundary, finding)
				}
			}
			findings = boundary // other catalog rules have their own positive/negative controls
			if err != nil || count != 1 || len(findings) != tc.want {
				t.Fatalf("count=%d err=%v findings=%+v want=%d", count, err, findings, tc.want)
			}
			for _, f := range findings {
				if f.Finding.Rule != "zellij-query-boundary" {
					t.Fatalf("unexpected finding: %+v", f)
				}
			}
		})
	}
}
