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

func TestFreshProfileCarriesLaunchNonceWithoutOrientation(t *testing.T) {
	raw := `{"schema_version":1,"tag":"work","agent":"claude","argv":[],"agent_source":"root","argv_source":"repo-default","fresh_required":true,"launch_nonce":"couch-attempt"}`
	args, _, err := ApplyCouchLaunchProfile(LaunchArgs{ForcedTag: "work"}, raw)
	if err != nil {
		t.Fatal(err)
	}
	rt := newFakeRuntime()
	code, err := run(t, baseOpts(args), rt)
	if err != nil || code != 0 {
		t.Fatalf("launch %d %v", code, err)
	}
	if got := rt.env["PAIR_LAUNCH_NONCE"]; got != "couch-attempt" {
		t.Fatalf("nonce %q", got)
	}
}

func TestFreshLaunchNonceValidation(t *testing.T) {
	p := TrustedLaunchProfile{SchemaVersion: 1, Tag: "work", Agent: "claude", Argv: []string{}, AgentSource: "root", ArgvSource: "repo-default", LaunchNonce: "attempt"}
	if ValidateTrustedLaunchProfile(p) == nil {
		t.Fatal("ordinary profile accepted fresh nonce")
	}
	p.FreshRequired = true
	p.Orientation = &orientation.Request{SchemaVersion: 1, Tag: "work", Agent: "claude", Attempt: "other", Body: "context"}
	if ValidateTrustedLaunchProfile(p) == nil {
		t.Fatal("conflicting orientation nonce accepted")
	}
	p.Orientation.Attempt = p.LaunchNonce
	if err := ValidateTrustedLaunchProfile(p); err != nil {
		t.Fatal(err)
	}
}
