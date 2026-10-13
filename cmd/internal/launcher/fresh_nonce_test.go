package launcher

import (
	"encoding/json"
	"testing"
)

func TestFreshLaunchNonceWithoutOrientationReachesReadiness(t *testing.T) {
	raw := `{"schema_version":1,"tag":"work","agent":"codex","argv":[],"agent_source":"explicit","argv_source":"explicit","fresh_required":true,"fresh_nonce":"couch-exact-attempt"}`
	args, _, err := ApplyCouchLaunchProfile(LaunchArgs{ForcedTag: "work"}, raw)
	if err != nil {
		t.Fatal(err)
	}
	rt := newFakeRuntime()
	opts := baseOpts(args)
	opts.Env.DataDir = t.TempDir()
	if code, err := run(t, opts, rt); err != nil || code != 0 {
		t.Fatalf("launch %d %v", code, err)
	}
	if rt.env["PAIR_LAUNCH_NONCE"] != "couch-exact-attempt" {
		t.Fatalf("nonce changed: %q", rt.env["PAIR_LAUNCH_NONCE"])
	}
	if args.Orientation != nil {
		t.Fatal("fresh created an orientation")
	}
}

func TestTrustedFreshNonceRefusesMismatchAndNonFresh(t *testing.T) {
	for _, raw := range []string{
		`{"schema_version":1,"tag":"work","agent":"codex","argv":[],"agent_source":"explicit","argv_source":"explicit","fresh_nonce":"n"}`,
		`{"schema_version":1,"tag":"work","agent":"codex","argv":[],"agent_source":"explicit","argv_source":"explicit","fresh_required":true,"fresh_nonce":"n","orientation":{"schema_version":1,"tag":"work","agent":"codex","attempt":"other","body":"brief"}}`,
		`{"schema_version":1,"tag":"work","agent":"codex","argv":[],"agent_source":"explicit","argv_source":"explicit","fresh_required":true,"fresh_nonce":"bad\nnonce"}`,
	} {
		var profile TrustedLaunchProfile
		if err := json.Unmarshal([]byte(raw), &profile); err != nil {
			t.Fatal(err)
		}
		if err := ValidateTrustedLaunchProfile(profile); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}
