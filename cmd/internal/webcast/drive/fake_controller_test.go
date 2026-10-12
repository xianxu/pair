package drive

import (
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestFakeControllerContract(t *testing.T) {
	RunControllerContract(t, func(sink recording.Sink, inj Injector, clock func() int64, grace time.Duration) Controller {
		return NewFakeController(sink, inj, clock, grace)
	})
}

func TestFakeControllerLabels(t *testing.T) {
	for in, want := range map[string]string{"  Ana ": "Ana", "": "Viewer 1", "\x07": "Viewer 1", "Bo\x1b[31m": "Bo[31m", "Ana\u202e": "Ana", "A\u2028B": "AB"} {
		if got := label(in, 1); got != want {
			t.Errorf("label(%q) = %q, want %q", in, got, want)
		}
	}
}
