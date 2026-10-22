package launcher

import (
	"os/exec"
	"testing"

	"github.com/xianxu/pair/cmd/internal/zellijexec"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

func TestSessionNameProbeRecoversEmptyThroughSharedRead(t *testing.T) {
	args := []string{"--session", "pair-a", "action", "list-clients"}
	f := newEmptyReplyFake(t, emptyReplyFrame{Args: args}, emptyReplyFrame{Args: args, Stdout: zellijtest.Case(t, "launcher-frames-clients").Stdout})
	rt := OSRuntime{nameQueryRun: func(cmd *exec.Cmd) error {
		cmd.Path = f.source.Path
		return zellijexec.Run(cmd)
	}}
	if err := rt.ProbeSessionNameContext(t.Context(), "pair-a"); err != nil || f.calls(t) != 2 {
		t.Fatalf("probe=%v calls=%d", err, f.calls(t))
	}
}
