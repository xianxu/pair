package repolint

import (
	"strings"
	"testing"
)

func TestWrapperChildReleaseRule(t *testing.T) {
	for _, tc := range []struct {
		name, pkg, body string
		want            int
	}{
		{"direct child", "wrapcmd", "_ = cmd.Process.Release()", 1},
		{"field receiver", "wrapcmd", "_ = owner.cmd.Process.Release()", 1},
		{"parenthesized receiver", "wrapcmd", "_ = (owner.cmd.Process).Release()", 1},
		{"joined child", "wrapcmd", "_ = cmd.Wait()", 0},
		{"owned group", "wrapcmd", "group := newOwnedGroup(cmd, timeout); defer group.Wait()", 0},
		{"other package", "launcher", "_ = cmd.Process.Release()", 0},
		{"other release", "wrapcmd", "_ = lease.Release()", 0},
		{"comment and literal", "wrapcmd", "// cmd.Process.Release()\n_ = \"owner.cmd.Process.Release()\"", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package " + tc.pkg + "\nfunc run() {\n" + tc.body + "\n}\n")
			repository := newFakeRepository()
			repository.index["cmd/internal/"+tc.pkg+"/child.go"] = source
			findings, _, count, err := scanTrackedWithRepository(repository, "repo", func(string) ([]byte, error) { return source, nil })
			if err != nil || count != 1 {
				t.Fatalf("scan: count=%d err=%v", count, err)
			}
			if len(findings) != tc.want {
				t.Fatalf("findings=%v, want %d", findings, tc.want)
			}
			for _, found := range findings {
				if found.Finding.Rule != "wrapper-child-release" || found.Finding.Line != 3 || !strings.Contains(found.Finding.Message, "pair#715") {
					t.Fatalf("unexpected finding: %+v", found)
				}
			}
		})
	}
}
