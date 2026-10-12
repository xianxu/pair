package query

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// The three queries on every fixture, as an agent sees them: the page at
// the first note, what happened in the 3 s before it, and every note.
// `go test ./cmd/internal/webcast/query -run Golden -update` rewrites them;
// review the diff, it is what agents read.
func TestGolden(t *testing.T) {
	for _, name := range recording.Fixtures {
		dir := recording.Fixture(name)
		r, err := recording.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		var first int64 = -1
		for _, x := range r.Records {
			if n, ok := x.(*recording.Note); ok {
				first = n.T
				break
			}
		}
		at := strconv.FormatInt(first, 10)
		from := strconv.FormatInt(max(first-3000, 0), 10)
		for query, argv := range map[string][]string{
			"dom":    {dir, "dom", "--at", at},
			"events": {dir, "events", "--from", from, "--to", at},
			"notes":  {dir, "notes"},
		} {
			code, out, errOut := runMain(argv...)
			if code != 0 {
				t.Errorf("%s %s: exit %d: %s", name, query, code, errOut)
				continue
			}
			got := fmt.Sprintf("$ couch %s %s\n%s", Flag, joinArgs(name, argv[1:]), out)
			golden := filepath.Join("testdata", "golden", name+"."+query+".txt")
			if *update {
				if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				continue
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("%v (run with -update to create it)", err)
			}
			if string(want) != got {
				t.Errorf("%s differs from its golden file; run with -update and review the diff", golden)
			}
		}
	}
}

func joinArgs(fixture string, rest []string) string {
	s := "<fixtures>/" + fixture
	for _, a := range rest {
		s += " " + a
	}
	return s
}
