package launcher

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

// These complete native replies preserve omission, not an invented empty field.
// Fix the retirement ordering; this unknown observation must never grant a kill.
func TestSessionOwnerCapturedHeldPane(t *testing.T) {
	for _, tc := range []struct{ agent, root, scope, tag string }{
		{"claude", "/Volumes/TestScratch/pc-2023113339/data/pair", "01934240e5ec5fde", "9223484470508650497-fixture-2"},
		{"codex", "/Volumes/TestScratch/pc-1617406610/data/pair", "5c9204b411267177", "9223718782449483777-fixture-2"},
	} {
		t.Run(tc.agent, func(t *testing.T) {
			raw := []byte(zellijtest.Case(t, "recovery-owner-"+tc.agent+"-held-797").Stdout)
			panes, err := zellijreply.ParsePanes(raw, zellijreply.PaneRequirements{})
			if err != nil {
				t.Fatal(err)
			}
			owner, err := artifactpath.NewStorageOwner(tc.root, tc.scope, tc.tag)
			if err != nil {
				t.Fatal(err)
			}
			got := ClassifySessionOwner(owner, panes)
			if got.State != SessionOwnerUnknown || !strings.Contains(got.Diagnostic, "actual command") {
				t.Fatalf("incomplete held-pane evidence authorized: %+v", got)
			}
			var liveDraft bool
			for _, pane := range panes {
				if pane.Title == "draft" {
					liveDraft = ClassifySessionOwner(owner, []zellijpane.Pane{pane}).State == SessionOwnerOwned
				}
			}
			if !liveDraft {
				t.Fatal("capture lost its independent positive live draft witness")
			}
		})
	}
}
