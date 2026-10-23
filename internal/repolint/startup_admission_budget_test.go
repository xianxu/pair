package repolint

import (
	"strings"
	"testing"
)

func TestStartupAdmissionBudgetRule(t *testing.T) {
	for _, tc := range []struct {
		name, pkg, imports, body string
		want                     int
	}{
		{"private admission", "launcher", "", `coordinateTerminalHandoff(lifetime,query,rt,birthBound,true)`, 1},
		{"public admission", "launcher", "", `CoordinateTerminalHandoff(lifetime,rt,birthBound)`, 1},
		{"parenthesized admission", "launcher", "", `(coordinateTerminalHandoff)(lifetime,query,rt,((birthBound)),true)`, 1},
		{"context timeout", "launcher", `import "context"`, `context.WithTimeout(ctx,birthBound)`, 1},
		{"aliased context timeout", "launcher", `import ctxpkg "context"`, `ctxpkg.WithTimeout(ctx,(birthBound))`, 1},
		{"birth watcher", "launcher", "", `watchBirth(ctx,rt,clock,evidence,session,birthBound,abort)`, 0},
		{"watched launch", "launcher", "", `launchWatched(ctx,rt,clock,evidence,session,config,layout,birthBound)`, 0},
		{"diagnostic", "launcher", "", `fmt.Sprintf("birth probe after %s",birthBound)`, 0},
		{"caller budget", "launcher", "", `coordinateTerminalHandoff(lifetime,query,rt,startupRemaining,true)`, 0},
		{"unrelated bound", "launcher", `import "context"`, `context.WithTimeout(ctx,queryBound)`, 0},
		{"not budget position", "launcher", "", `coordinateTerminalHandoff(birthBound,query,rt,startupRemaining,true)`, 0},
		{"other timeout provider", "launcher", `import "example/context"`, `context.WithTimeout(ctx,birthBound)`, 0},
		{"other function", "launcher", `import "context"`, `context.WithDeadline(ctx,birthBound)`, 0},
		{"other package", "other", `import "context"`, `context.WithTimeout(ctx,birthBound)`, 0},
		{"local alias outside syntax scope", "launcher", `import "context"`, `bound:=birthBound;context.WithTimeout(ctx,bound)`, 0},
		{"converted expression outside syntax scope", "launcher", `import "context"`, `context.WithTimeout(ctx,time.Duration(birthBound))`, 0},
		{"function alias outside syntax scope", "launcher", "", `admit:=CoordinateTerminalHandoff;admit(ctx,rt,birthBound)`, 0},
		{"comment and literal", "launcher", "", "// CoordinateTerminalHandoff(ctx,rt,birthBound)\n_ = `context.WithTimeout(ctx,birthBound)`", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package " + tc.pkg + "\n" + tc.imports + "\nfunc run(){\n" + tc.body + "\n}")
			repository := newFakeRepository()
			repository.index["cmd/internal/"+tc.pkg+"/admission.go"] = source
			findings, _, count, err := scanTrackedWithRepository(repository, "repo", func(string) ([]byte, error) { return source, nil })
			if err != nil || count != 1 || len(findings) != tc.want {
				t.Fatalf("findings=%+v count=%d err=%v want=%d", findings, count, err, tc.want)
			}
			for _, finding := range findings {
				if finding.Finding.Rule != "startup-admission-budget" || !strings.Contains(finding.Finding.Message, "pair#773") {
					t.Fatalf("unexpected finding %+v", finding)
				}
			}
		})
	}
}
