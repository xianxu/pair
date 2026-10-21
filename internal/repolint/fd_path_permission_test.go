package repolint

import (
	"strconv"
	"strings"
	"testing"
)

func TestFDPathPermissionRule(t *testing.T) {
	for _, tc := range []struct {
		shell string
		want  int
	}{
		{"test -r " + "/dev/fd/3 || exit 42", 1},
		{`[ -w ` + `"/dev/fd/12" ]`, 1},
		{"[[ -x " + "'/dev/fd/5' ]]", 1},
		{"IFS= read -r token <&3", 0},
		{"test -r /tmp/regular-file", 0},
		{"echo /dev/fd/3", 0},
	} {
		t.Run(tc.shell, func(t *testing.T) {
			source := []byte("package example\nvar script=" + strconv.Quote(tc.shell) + "\n")
			repository := newFakeRepository()
			repository.index["cmd/internal/example/lease_test.go"] = source
			findings, _, count, err := scanTrackedWithRepository(repository, "repo", func(string) ([]byte, error) { return source, nil })
			if err != nil || count != 1 || len(findings) != tc.want {
				t.Fatalf("findings=%v count=%d err=%v want=%d", findings, count, err, tc.want)
			}
			for _, f := range findings {
				if f.Finding.Rule != "fd-path-permission" || !strings.Contains(f.Finding.Message, "pair#767") {
					t.Fatalf("unexpected finding %+v", f)
				}
			}
		})
	}
}
