package repolint

import (
	"strings"
	"testing"
)

func TestTerminalRegistryOwnerRule(t *testing.T) {
	for _, tc := range []struct {
		name, pkg, fn, body string
		want                int
	}{
		{"native admission", "couchcmd", "read", "return reg.LiveIDs(alive)", 1},
		{"parenthesized", "termcmd", "read", "return (reg).LiveIDs(alive)", 1},
		{"layout admission", "layoutcmd", "read", "return reg.LiveIDs(func(pid int)bool{return true})", 1},
		{"clipboard admission", "clipcmd", "read", "return reg.LiveIDs(alive)", 1},
		{"owner-aware", "couchcmd", "read", "return reg.LiveOwnerIDs(alive, processes)", 0},
		{"storage model test", "couchcmd", "TestRegistry", "return reg.LiveIDs(alive)", 0},
		{"shared adapter", "workbenchshortcut", "liveOwnerIDs", "return reg.LiveIDs(owner)", 0},
		{"unrelated package", "elsewhere", "read", "return reg.LiveIDs(alive)", 0},
		{"literal only", "couchcmd", "read", "return \"reg.LiveIDs(alive)\"", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package " + tc.pkg + "\nfunc " + tc.fn + "(){\n" + tc.body + "\n}")
			repository := newFakeRepository()
			repository.index["cmd/internal/"+tc.pkg+"/registry.go"] = source
			findings, _, count, err := scanTrackedWithRepository(repository, "repo", func(string) ([]byte, error) { return source, nil })
			if err != nil || count != 1 || len(findings) != tc.want {
				t.Fatalf("findings=%v count=%d err=%v want=%d", findings, count, err, tc.want)
			}
			for _, f := range findings {
				if f.Finding.Rule != "terminal-registry-owner" || !strings.Contains(f.Finding.Message, "pair#731") {
					t.Fatalf("unexpected finding %+v", f)
				}
			}
		})
	}
}
