package couchtty

import (
	"context"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// Separate ingestion, attachment and selection, instead of relying on which
// goroutine happens to win a real shell's single startup write.
func TestShellStartupPresentationOrdering(t *testing.T) {
	for _, when := range []string{"before-attach", "before-select", "after-select"} {
		t.Run(when, func(t *testing.T) {
			f := newFixture(t, 24, 100)
			f.child.Feed([]byte("\x1b[?1049h\x1b[2J\x1b[1;1Hnative-ready"))
			waitFor(t, "native alternate screen", func() bool { return strings.Contains(f.screen.childArea(), "native-ready") })
			child := newLaidOutFakeChild(t, f.con)
			child.SetSink(func(ctx context.Context, batch ptychild.OutputBatch) error { return f.con.Deliver(ctx, "shell", batch) })
			output := func() {
				child.Feed([]byte("shell-counterpart-ready\r\n"))
				if err := child.FlushOutput(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			if when == "before-attach" {
				output()
			}
			f.con.Attach("shell", "native-shell", child)
			if when == "before-select" {
				output()
			}
			f.con.Switch("shell")
			waitFor(t, "shell selection", func() bool { return f.con.presenter.View().Admitted == child.Endpoint().ID() })
			if when == "after-select" {
				output()
			}
			waitFor(t, "shell startup text", func() bool { return strings.Contains(f.screen.childArea(), "shell-counterpart-ready") })
		})
	}
}
