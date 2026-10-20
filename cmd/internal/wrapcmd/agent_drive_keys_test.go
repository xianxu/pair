package wrapcmd

import (
	"bytes"
	"testing"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
)

// TestAgentDriveSpecialKeysReachTheAgentAsTableBytes drives every named key
// through the wrapper's real writer, including Pair's input translation, and
// expects exactly the agentdrive.Steps bytes, one step at a time (pair#717).
func TestAgentDriveSpecialKeysReachTheAgentAsTableBytes(t *testing.T) {
	var names []string
	for _, step := range agentdrive.Steps {
		switch step.Kind {
		case agentdrive.KeyText, agentdrive.KeyWait, agentdrive.KeyEnter, agentdrive.KeySubmit, agentdrive.KeyEsc:
			// enter and submit are remapped for the composer and pinned by
			// their own tests; esc is the recovery key.
			continue
		}
		names = append(names, string(step.Kind))
	}
	for start := 0; start < len(names); start += agentdrive.MaxKeySteps {
		batch := names[start:min(start+agentdrive.MaxKeySteps, len(names))]
		keys, err := agentdrive.ParseKeySequence(batch)
		if err != nil {
			t.Fatal(err)
		}
		f, d, _ := driveFixture(t)
		if r, err := driveStart(t, d, agentdrive.Request{Keys: keys}); err != nil || r.Terminal() {
			t.Fatalf("admit %+v %v", r, err)
		}
		var out bytes.Buffer
		var want []byte
		for _, k := range keys {
			before := out.Len()
			for i := 0; i < 4 && out.Len() == before; i++ {
				f.proxy.dispatchAgentDrive(&out)
			}
			got := out.Bytes()[before:]
			if !bytes.Equal(got, k.InputBytes()) {
				t.Errorf("%s reached the agent as %q, want %q", k.Kind, got, k.InputBytes())
			}
			want = append(want, k.InputBytes()...)
		}
		if !bytes.Equal(out.Bytes(), want) {
			t.Fatalf("stream %q, want %q", out.Bytes(), want)
		}
	}
}
