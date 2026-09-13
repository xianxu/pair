package launcher

import (
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"testing"
)

func TestOrientationFreshProfileAndNonce(t *testing.T) {
	request := orientation.Request{SchemaVersion: 1, Tag: "work", Agent: "codex", Attempt: "attempt-1", Body: "read context"}
	p := TrustedLaunchProfile{SchemaVersion: 1, Tag: "work", Agent: "codex", Argv: []string{}, AgentSource: "explicit", ArgvSource: "explicit", FreshRequired: true, Orientation: &request}
	raw, _ := json.Marshal(p)
	args, _, err := ApplyCouchLaunchProfile(LaunchArgs{ForcedTag: "work"}, string(raw))
	if err != nil {
		t.Fatal(err)
	}
	rt := newFakeRuntime()
	code, err := run(t, baseOpts(args), rt)
	if err != nil || code != 0 {
		t.Fatalf("launch %d %v", code, err)
	}
	if rt.env["PAIR_LAUNCH_NONCE"] != request.Attempt || rt.env[orientation.Env] == "" {
		t.Fatalf("missing request/nonce %#v", rt.env)
	}
	p.FreshRequired = false
	if ValidateTrustedLaunchProfile(p) == nil {
		t.Fatal("accepted ordinary orientation")
	}
	p.FreshRequired = true
	p.Orientation.Tag = "other"
	if ValidateTrustedLaunchProfile(p) == nil {
		t.Fatal("accepted different tag")
	}
}
