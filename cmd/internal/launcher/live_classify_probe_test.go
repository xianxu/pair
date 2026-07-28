package launcher

import (
	"os"
	"testing"

	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

// Live conformance probe (#123 smoke item 8): when PAIR_LIVE_PANES_JSON names
// a dump of `zellij action list-panes --json ...` from a live layout-3
// session, the classifier must call it layout3. Skipped otherwise.
func TestClassifyLiveLayoutAgainstLiveDump(t *testing.T) {
	path := os.Getenv("PAIR_LIVE_PANES_JSON")
	if path == "" {
		t.Skip("no live dump provided")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mode, ok := ClassifyLiveLayout(zellijpane.Parse(data))
	if !ok || mode != Layout3 {
		t.Fatalf("ClassifyLiveLayout(live) = (%q, %v), want (layout3, true)", mode, ok)
	}
}
